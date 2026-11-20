package integration_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"Agent-Workflow/internal/domain"
	"Agent-Workflow/internal/store/postgres"
	"Agent-Workflow/internal/workflow"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type streamChatModel func(context.Context, []*schema.Message) (*schema.StreamReader[*schema.Message], error)

func (m streamChatModel) Stream(ctx context.Context, messages []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return m(ctx, messages)
}

func (m streamChatModel) Generate(context.Context, []*schema.Message, ...model.Option) (*schema.Message, error) {
	return nil, errors.New("unexpected non-streaming model call")
}

func TestSingleNodeWorkflowStreamsBeforeSavingAndReplays(t *testing.T) {
	store, _, ctx := postgresStore(t)
	firstDelta := make(chan struct{})
	calls := 0
	chat := streamChatModel(func(ctx context.Context, messages []*schema.Message) (*schema.StreamReader[*schema.Message], error) {
		calls++
		if len(messages) != 1 || messages[0].Content != "hello" {
			return nil, errors.New("unexpected messages")
		}
		reader, writer := schema.Pipe[*schema.Message](0)
		go func() {
			defer writer.Close()
			if writer.Send(&schema.Message{Content: "hello"}, nil) {
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-firstDelta:
			}
			writer.Send(&schema.Message{Content: " world", ResponseMeta: &schema.ResponseMeta{FinishReason: "stop"}}, nil)
		}()
		return reader, nil
	})
	flow, err := workflow.NewTestWorkflow(ctx, store, chat, "stub")
	if err != nil {
		t.Fatal(err)
	}
	id := testUUID(t)
	input := workflow.TestTurnInput{SessionID: domain.SessionID(id.String()), RequestID: "stream", Message: "hello"}
	key := postgres.EventKey{Source: "workflow-test:" + id.String(), SourceEventID: input.RequestID}
	var deltas []string
	result, err := flow.Stream(ctx, input, func(delta string) error {
		deltas = append(deltas, delta)
		if len(deltas) > 1 {
			return nil
		}
		defer close(firstDelta)
		event, err := store.GetEventBySource(ctx, key)
		if err != nil || event.Status == nil || *event.Status != string(domain.EventProcessing) || len(event.ResultJSON) != 0 {
			return fmt.Errorf("reply committed before the final chunk: %v", err)
		}
		return nil
	})
	if err != nil || result.EventStatus != domain.EventApplied || result.Reply == nil || result.Reply.Text != "hello world" ||
		!reflect.DeepEqual(deltas, []string{"hello", " world"}) {
		t.Fatalf("unexpected stream: result=%+v deltas=%q err=%v", result, deltas, err)
	}

	deltas = nil
	replayed, err := flow.Stream(ctx, input, func(delta string) error {
		deltas = append(deltas, delta)
		return nil
	})
	if err != nil || replayed.EventID != result.EventID || !reflect.DeepEqual(deltas, []string{"hello world"}) || calls != 1 {
		t.Fatalf("stream replay failed: result=%+v deltas=%q calls=%d err=%v", replayed, deltas, calls, err)
	}
}

func TestSingleNodeWorkflowRetriesTruncatedStream(t *testing.T) {
	store, _, ctx := postgresStore(t)
	calls := 0
	chat := streamChatModel(func(context.Context, []*schema.Message) (*schema.StreamReader[*schema.Message], error) {
		calls++
		if calls == 1 {
			return schema.StreamReaderFromArray([]*schema.Message{{Content: "partial"}}), nil
		}
		return schema.StreamReaderFromArray([]*schema.Message{{Content: "complete", ResponseMeta: &schema.ResponseMeta{FinishReason: "stop"}}}), nil
	})
	flow, err := workflow.NewTestWorkflow(ctx, store, chat, "stub")
	if err != nil {
		t.Fatal(err)
	}
	input := workflow.TestTurnInput{RequestID: "retry-stream", Message: "hello"}
	failed, err := flow.Stream(ctx, input, nil)
	if !errors.Is(err, workflow.ErrTestModelFailed) || failed.Reply != nil || failed.EventStatus != domain.EventRetryWait {
		t.Fatalf("truncated reply was not retryable: %+v, %v", failed, err)
	}
	input.SessionID = failed.SessionID
	retried, err := flow.Stream(ctx, input, nil)
	if err != nil || retried.EventID != failed.EventID || retried.Reply == nil || retried.Reply.Text != "complete" {
		t.Fatalf("retry did not complete the same event: %+v, %v", retried, err)
	}
}
