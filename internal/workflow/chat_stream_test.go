package workflow

import (
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestConsumeChatStreamPreservesDeltasAndRequiresFinish(t *testing.T) {
	reader := schema.StreamReaderFromArray([]*schema.Message{
		{Role: schema.Assistant},
		{Content: "hello"},
		{Content: " world"},
		{ResponseMeta: &schema.ResponseMeta{FinishReason: "stop"}},
	})
	var deltas []string
	text, err := ConsumeChatStream(t.Context(), reader, func(delta string) error {
		deltas = append(deltas, delta)
		return nil
	})
	if err != nil || text != "hello world" || !reflect.DeepEqual(deltas, []string{"hello", " world"}) {
		t.Fatalf("reply=%q deltas=%q error=%v", text, deltas, err)
	}

	for name, chunks := range map[string][]*schema.Message{
		"missing finish marker": {{Content: "partial"}},
		"aborted":               {{Content: "partial"}, {ResponseMeta: &schema.ResponseMeta{FinishReason: "aborted"}}},
	} {
		t.Run(name, func(t *testing.T) {
			text, err := ConsumeChatStream(t.Context(), schema.StreamReaderFromArray(chunks), nil)
			if text != "" || err == nil {
				t.Fatalf("partial reply=%q error=%v", text, err)
			}
		})
	}
}

func TestConsumeChatStreamClosesReaderOnSinkError(t *testing.T) {
	reader, writer := schema.Pipe[*schema.Message](1)
	writer.Send(&schema.Message{Content: "first"}, nil)
	defer writer.Close()
	text, err := ConsumeChatStream(t.Context(), reader, func(string) error { return io.ErrClosedPipe })
	if text != "" || !errors.Is(err, io.ErrClosedPipe) || !errors.Is(err, ErrChatDeltaDelivery) {
		t.Fatalf("reply=%q error=%v", text, err)
	}
	if closed := writer.Send(&schema.Message{Content: "second"}, nil); !closed {
		t.Fatal("the stream reader was not closed")
	}
}
