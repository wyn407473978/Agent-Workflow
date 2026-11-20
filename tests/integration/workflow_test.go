package integration_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"Agent-Workflow/internal/domain"
	"Agent-Workflow/internal/store/postgres"
	"Agent-Workflow/internal/workflow"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/jackc/pgx/v5/pgtype"
)

type testChatModel func([]*schema.Message) (string, error)

func (m testChatModel) Generate(_ context.Context, messages []*schema.Message, _ ...model.Option) (*schema.Message, error) {
	text, err := m(messages)
	return schema.AssistantMessage(text, nil), err
}

func (m testChatModel) Stream(context.Context, []*schema.Message, ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, errors.New("unexpected streaming call")
}

func TestSingleNodeWorkflowRestoresMemoryAndReplays(t *testing.T) {
	store, db, ctx := postgresStore(t)
	firstModel := testChatModel(func(messages []*schema.Message) (string, error) {
		if len(messages) != 1 || messages[0].Content != "我的代号是青松" {
			return "", errors.New("unexpected first turn")
		}
		return "已记录代号", nil
	})
	firstFlow, err := workflow.NewTestWorkflow(ctx, store, firstModel, "stub")
	if err != nil {
		t.Fatal(err)
	}
	first, err := firstFlow.Run(ctx, workflow.TestTurnInput{RequestID: "first", Message: "我的代号是青松"})
	if err != nil {
		t.Fatal(err)
	}
	if first.Reply == nil || first.Reply.Text != "已记录代号" || first.Checkpoint == nil || first.Checkpoint.StateRevision != 2 {
		t.Fatalf("unexpected first result: %+v", first)
	}

	calls := 0
	secondModel := testChatModel(func(messages []*schema.Message) (string, error) {
		calls++
		if len(messages) != 3 || messages[0].Role != schema.User || messages[0].Content != "我的代号是青松" ||
			messages[1].Role != schema.Assistant || messages[1].Content != "已记录代号" || messages[2].Content != "我的代号是什么" {
			return "", fmt.Errorf("database history not restored: %v", messages)
		}
		return "青松", nil
	})
	restarted, err := workflow.NewTestWorkflow(ctx, postgres.NewStore(db), secondModel, "stub")
	if err != nil {
		t.Fatal(err)
	}
	input := workflow.TestTurnInput{SessionID: first.SessionID, RequestID: "second", Message: "我的代号是什么"}
	second, err := restarted.Run(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if second.WorkflowID != first.WorkflowID || second.Reply == nil || second.Reply.Text != "青松" ||
		second.Checkpoint == nil || second.Checkpoint.StateRevision != 4 {
		t.Fatalf("unexpected restored result: %+v", second)
	}
	replayed, err := restarted.Run(ctx, input)
	if err != nil || replayed.EventID != second.EventID || replayed.Reply == nil || replayed.Reply.Text != "青松" || calls != 1 {
		t.Fatalf("request was not replayed from the database: %+v, calls=%d, err=%v", replayed, calls, err)
	}
	input.Message = "different payload"
	if _, err := restarted.Run(ctx, input); !errors.Is(err, postgres.ErrConflict) {
		t.Fatalf("reused request key did not conflict: %v", err)
	}

	var workflowID, sessionID pgtype.UUID
	if err := workflowID.Scan(string(first.WorkflowID)); err != nil {
		t.Fatal(err)
	}
	if err := sessionID.Scan(string(first.SessionID)); err != nil {
		t.Fatal(err)
	}
	storedSession, err := store.GetSession(ctx, sessionID)
	if err != nil || storedSession.ActiveWorkflowID != workflowID {
		t.Fatalf("session was not bound to its instance: %+v, %v", storedSession, err)
	}
	history, err := store.ListRecentConversationTurns(ctx, sessionID, workflowID, 1)
	if err != nil || len(history) != 1 || history[0].EventID.String() != string(second.EventID) {
		t.Fatalf("latest turn was not returned: %+v, %v", history, err)
	}
	transitions, err := store.ListWorkflowTransitions(ctx, workflowID, 0, 10)
	if err != nil || len(transitions) != 4 || transitions[3].ToStatus == nil || *transitions[3].ToStatus != string(domain.WorkflowWaitingInput) {
		t.Fatalf("workflow transitions were not recorded: %+v, %v", transitions, err)
	}
	nodes, err := store.ListWorkflowNodeExecutions(ctx, workflowID, 10)
	if err != nil || len(nodes) != 2 || nodes[0].NodeName == nil || *nodes[0].NodeName != "test_chat" || nodes[0].FinishedAt == nil {
		t.Fatalf("test node executions were not recorded: %+v, %v", nodes, err)
	}
}

func TestSingleNodeWorkflowSerializesSessionAndRetries(t *testing.T) {
	store, _, ctx := postgresStore(t)
	started, release := make(chan struct{}), make(chan struct{})
	firstFlow, err := workflow.NewTestWorkflow(ctx, store, testChatModel(func([]*schema.Message) (string, error) {
		close(started)
		select {
		case <-release:
		case <-ctx.Done():
		}
		return "", errors.New("simulated upstream failure")
	}), "stub")
	if err != nil {
		t.Fatal(err)
	}
	input := workflow.TestTurnInput{SessionID: domain.SessionID(testUUID(t).String()), RequestID: "retry", Message: "hello"}
	type runResult struct {
		turn workflow.TestTurnResult
		err  error
	}
	firstDone := make(chan runResult, 1)
	go func() {
		turn, err := firstFlow.Run(ctx, input)
		firstDone <- runResult{turn: turn, err: err}
	}()
	select {
	case <-started:
	case result := <-firstDone:
		t.Fatalf("first model did not start: %v", result.err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	secondFlow, err := workflow.NewTestWorkflow(ctx, store, testChatModel(func(messages []*schema.Message) (string, error) {
		if len(messages) != 1 || messages[0].Content != "hello" {
			return "", errors.New("failed attempts must stay out of conversation memory")
		}
		return "ok", nil
	}), "stub")
	if err != nil {
		t.Fatal(err)
	}
	_, busyErr := secondFlow.Run(ctx, workflow.TestTurnInput{SessionID: input.SessionID, RequestID: "parallel", Message: "parallel"})
	close(release)
	failed := <-firstDone
	if !errors.Is(busyErr, workflow.ErrTestSessionBusy) {
		t.Fatalf("overlapping turn was not rejected: %v", busyErr)
	}
	if !errors.Is(failed.err, workflow.ErrTestModelFailed) || failed.turn.EventStatus != domain.EventRetryWait {
		t.Fatalf("failure did not persist as retryable: %+v, %v", failed.turn, failed.err)
	}
	retried, err := secondFlow.Run(ctx, input)
	if err != nil || retried.EventID != failed.turn.EventID || retried.EventStatus != domain.EventApplied || retried.Reply == nil || retried.Reply.Text != "ok" {
		t.Fatalf("retry did not complete the same event: %+v, %v", retried, err)
	}
}
