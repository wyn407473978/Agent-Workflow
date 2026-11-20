package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"Agent-Workflow/internal/domain"
	"Agent-Workflow/internal/store/postgres"
	storemodel "Agent-Workflow/internal/store/postgres/model"
	"github.com/cloudwego/eino/schema"
	"github.com/jackc/pgx/v5/pgtype"
	"gorm.io/gorm"
)

type testTurn struct {
	event       storemodel.WorkflowEvent
	workflow    storemodel.WorkflowInstance
	executionID pgtype.UUID
	history     []*schema.Message
	replay      *RoundResult
}

func (w *TestWorkflow) beginTurn(ctx context.Context, sessionID pgtype.UUID, key postgres.EventKey, payload json.RawMessage, requestHash string) (testTurn, error) {
	var turn testTurn
	err := w.store.Transaction(ctx, func(tx *postgres.Tx) error {
		if err := tx.LockSessions(ctx, sessionID); err != nil {
			return err
		}
		session, err := loadTestSession(ctx, tx, sessionID)
		if err != nil {
			return err
		}
		at := time.Now().UTC()
		event, _, err := tx.ReceiveEvent(ctx, storemodel.WorkflowEvent{
			EventID: newTestUUID(), SessionID: sessionID, WorkflowID: session.ActiveWorkflowID,
			Source: &key.Source, SourceEventID: &key.SourceEventID, EventType: new(string(domain.EventUserMessage)),
			ActorType: new(string(domain.ActorUser)), ActorID: new(testOwner), RequestHash: &requestHash,
			Payload: payload, ReferenceAt: &at, CreatedAt: &at,
		})
		if err != nil {
			return err
		}
		if event.SessionID != sessionID || event.EventType == nil || *event.EventType != string(domain.EventUserMessage) {
			return fmt.Errorf("stored test event has a different session or type: %w", postgres.ErrDataIntegrity)
		}
		turn.event = event
		if event.Status != nil && *event.Status == string(domain.EventApplied) {
			turn.replay, err = savedRound(event.ResultJSON)
			if err != nil {
				return err
			}
			if !event.WorkflowID.Valid || turn.replay.WorkflowID != domain.WorkflowID(event.WorkflowID.String()) {
				return fmt.Errorf("saved reply does not match its workflow: %w", postgres.ErrDataIntegrity)
			}
			return nil
		}
		turn.workflow, err = loadTestInstance(ctx, tx, session, event.EventID)
		if err != nil {
			return err
		}
		workflowID := turn.workflow.WorkflowID
		if event.WorkflowID.Valid && event.WorkflowID != workflowID {
			return fmt.Errorf("test event belongs to another conversation: %w", postgres.ErrConflict)
		}
		busy, err := tx.HasOtherActiveSessionTurn(ctx, sessionID, workflowID, event.EventID)
		if err != nil {
			return err
		}
		if busy {
			return ErrTestSessionBusy
		}
		at = time.Now().UTC()
		lease := postgres.Lease{Token: newTestUUID(), ExpiresAt: at.Add(testLeaseDuration)}
		turn.event, err = tx.ClaimEvent(ctx, key, at, lease)
		if errors.Is(err, postgres.ErrConflict) {
			return ErrTestSessionBusy
		}
		if err != nil {
			return err
		}
		if err := tx.SaveEventResult(ctx, key, lease.Token, at, postgres.EventResult{
			WorkflowID: workflowID, Status: string(domain.EventProcessing),
		}); err != nil {
			return err
		}
		if *turn.workflow.Status == string(domain.WorkflowWaitingInput) || *turn.event.AttemptCount > 1 {
			if err := updateTestInstance(ctx, tx, &turn.workflow, domain.WorkflowRunning, event.EventID, "TEST_TURN_STARTED", nil, at); err != nil {
				return err
			}
		}
		recent, err := tx.ListRecentConversationTurns(ctx, sessionID, workflowID, testMemoryTurns)
		if err != nil {
			return err
		}
		turn.history, err = conversationMessages(recent)
		if err != nil {
			return err
		}
		turn.executionID = newTestUUID()
		inputJSON, err := json.Marshal(map[string]int{"history_turns": len(turn.history) / 2})
		if err != nil {
			return err
		}
		return tx.CreateNodeExecution(ctx, storemodel.WorkflowNodeExecution{
			ExecutionID: turn.executionID, WorkflowID: workflowID, EventID: event.EventID,
			NodeName: new(testChatNodeName), AttemptNo: turn.event.AttemptCount, ModelName: &w.modelName,
			Status: new("RUNNING"), InputJSON: inputJSON, StartedAt: new(time.Now().UTC()),
		})
	})
	return turn, err
}

func (w *TestWorkflow) finishTurn(ctx context.Context, sessionID pgtype.UUID, key postgres.EventKey, turn testTurn, reply string) (RoundResult, error) {
	at := time.Now().UTC()
	status := domain.WorkflowWaitingInput
	result := RoundResult{
		WorkflowID: domain.WorkflowID(turn.workflow.WorkflowID.String()), EventStatus: domain.EventApplied,
		Checkpoint: &Checkpoint{StateRevision: *turn.workflow.StateRevision + 1},
	}
	nodeStatus, reason := "SUCCEEDED", "TEST_TURN_REPLIED"
	var errorCode *string
	if reply != "" {
		result.Reply = &UserReply{Text: reply, Status: status}
		result.Checkpoint.CompletedNodes = []domain.NodeName{testChatNodeName}
	} else {
		errorCode = new("TEST_MODEL_ERROR")
		nodeStatus, reason = "FAILED", "TEST_TURN_FAILED"
		result.EventStatus = domain.EventRetryWait
		result.Error = &domain.WorkflowError{Code: domain.ErrorCode(*errorCode), Retryable: true}
		result.NextAttemptAt = &at
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return RoundResult{}, err
	}
	err = w.store.Transaction(ctx, func(tx *postgres.Tx) error {
		if err := tx.LockSessions(ctx, sessionID); err != nil {
			return err
		}
		session, err := tx.GetSession(ctx, sessionID)
		if err != nil {
			return err
		}
		workflowID := turn.workflow.WorkflowID
		if session.ActiveWorkflowID != workflowID {
			return fmt.Errorf("session changed active workflow: %w", postgres.ErrConflict)
		}
		if err := tx.LockEvents(ctx, key); err != nil {
			return err
		}
		if err := tx.SaveEventResult(ctx, key, turn.event.LeaseToken, at, postgres.EventResult{
			WorkflowID: workflowID, Status: string(result.EventStatus), ResultJSON: resultJSON,
			NextAttemptAt: result.NextAttemptAt, LastErrorCode: errorCode,
		}); err != nil {
			return err
		}
		if err := updateTestInstance(ctx, tx, &turn.workflow, status, turn.event.EventID, reason, errorCode, at); err != nil {
			return err
		}
		return tx.FinishNodeExecution(ctx, storemodel.WorkflowNodeExecution{
			ExecutionID: turn.executionID, WorkflowID: workflowID, Status: &nodeStatus, ErrorCode: errorCode, FinishedAt: &at,
		})
	})
	if err != nil {
		return RoundResult{}, err
	}
	return result, nil
}

func loadTestSession(ctx context.Context, tx *postgres.Tx, id pgtype.UUID) (storemodel.WorkflowSession, error) {
	session, err := tx.GetSession(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		at := time.Now().UTC()
		session = storemodel.WorkflowSession{SessionID: id, UserID: new(testOwner), AccountID: new(testOwner), CreatedAt: &at, UpdatedAt: &at}
		return session, tx.CreateSession(ctx, session)
	}
	if err != nil {
		return storemodel.WorkflowSession{}, err
	}
	if session.AccountID == nil || *session.AccountID != testOwner || session.UserID == nil || *session.UserID != testOwner {
		return storemodel.WorkflowSession{}, fmt.Errorf("session does not belong to the local test workflow: %w", postgres.ErrConflict)
	}
	return session, nil
}

func loadTestInstance(ctx context.Context, tx *postgres.Tx, session storemodel.WorkflowSession, eventID pgtype.UUID) (storemodel.WorkflowInstance, error) {
	if session.ActiveWorkflowID.Valid {
		if err := tx.LockWorkflows(ctx, session.ActiveWorkflowID); err != nil {
			return storemodel.WorkflowInstance{}, err
		}
		instance, err := tx.GetWorkflow(ctx, session.ActiveWorkflowID)
		if err != nil {
			return storemodel.WorkflowInstance{}, err
		}
		if instance.SessionID != session.SessionID || instance.WorkflowVersion == nil || *instance.WorkflowVersion != testVersion ||
			instance.StateRevision == nil || instance.Status == nil ||
			(*instance.Status != string(domain.WorkflowRunning) && *instance.Status != string(domain.WorkflowWaitingInput)) {
			return storemodel.WorkflowInstance{}, fmt.Errorf("active workflow cannot run a local test turn: %w", postgres.ErrConflict)
		}
		return instance, nil
	}
	at := time.Now().UTC()
	instance := storemodel.WorkflowInstance{
		WorkflowID: newTestUUID(), SessionID: session.SessionID, Status: new(string(domain.WorkflowRunning)),
		CurrentNode: new(testChatNodeName), WorkflowVersion: new(testVersion), StateRevision: new(int64(1)), CreatedAt: &at, UpdatedAt: &at,
	}
	if err := tx.CreateWorkflow(ctx, instance); err != nil {
		return storemodel.WorkflowInstance{}, err
	}
	if err := tx.SetSessionWorkflow(ctx, session.SessionID, instance.WorkflowID, at); err != nil {
		return storemodel.WorkflowInstance{}, err
	}
	return instance, recordTestTransition(ctx, tx, instance, eventID, nil, "TEST_WORKFLOW_CREATED", at)
}

func updateTestInstance(ctx context.Context, tx *postgres.Tx, current *storemodel.WorkflowInstance, status domain.WorkflowStatus, eventID pgtype.UUID, reason string, errorCode *string, at time.Time) error {
	next := *current
	next.Status = new(string(status))
	next.LastErrorCode = errorCode
	next.UpdatedAt = &at
	if err := tx.UpdateWorkflow(ctx, next, *current.StateRevision, *current.Status); err != nil {
		return err
	}
	next.StateRevision = new(*current.StateRevision + 1)
	if *current.Status != *next.Status {
		if err := recordTestTransition(ctx, tx, next, eventID, current.Status, reason, at); err != nil {
			return err
		}
	}
	*current = next
	return nil
}

func recordTestTransition(ctx context.Context, tx *postgres.Tx, instance storemodel.WorkflowInstance, eventID pgtype.UUID, from *string, reason string, at time.Time) error {
	return tx.AppendTransition(ctx, storemodel.WorkflowTransition{
		WorkflowID: instance.WorkflowID, StateRevision: instance.StateRevision, EventID: eventID,
		FromStatus: from, ToStatus: instance.Status, ReasonCode: &reason,
		ActorType: new(string(domain.ActorUser)), ActorID: new(testOwner), CreatedAt: &at,
	})
}

func conversationMessages(events []storemodel.WorkflowEvent) ([]*schema.Message, error) {
	messages := make([]*schema.Message, 0, len(events)*2+1)
	for _, event := range events {
		var message domain.UserMessagePayload
		if err := json.Unmarshal(event.Payload, &message); err != nil {
			return nil, fmt.Errorf("decode stored test message: %w", err)
		}
		round, err := savedRound(event.ResultJSON)
		if err != nil {
			return nil, err
		}
		messages = append(messages, schema.UserMessage(message.Text), schema.AssistantMessage(round.Reply.Text, nil))
	}
	return messages, nil
}

func savedRound(data json.RawMessage) (*RoundResult, error) {
	var round RoundResult
	if err := json.Unmarshal(data, &round); err != nil {
		return nil, fmt.Errorf("decode saved test turn: %w", err)
	}
	if round.Reply == nil || round.Reply.Text == "" || round.EventStatus != domain.EventApplied {
		return nil, fmt.Errorf("successful test event has no saved reply: %w", postgres.ErrDataIntegrity)
	}
	return &round, nil
}
