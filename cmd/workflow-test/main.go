package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"log"
	"os"
	"os/signal"
	"time"

	"Agent-Workflow/internal/config"
	"Agent-Workflow/internal/domain"
	"Agent-Workflow/internal/llm"
	"Agent-Workflow/internal/store/postgres"
	"Agent-Workflow/internal/workflow"
)

func main() {
	session := flag.String("session", "", "session UUID (optional when starting a new conversation)")
	request := flag.String("request", "", "request ID (reuse it for a retry)")
	message := flag.String("message", "", "message for the test node")
	stream := flag.Bool("stream", false, "emit newline-delimited JSON events as text arrives")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, workflow.TestTurnInput{SessionID: domain.SessionID(*session), RequestID: *request, Message: *message}, *stream); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, input workflow.TestTurnInput, stream bool) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	chat, err := llm.NewDeepSeek(ctx, cfg.DeepSeek)
	if err != nil {
		return err
	}
	openCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	db, err := postgres.Open(openCtx, cfg.Postgres)
	cancel()
	if err != nil {
		return err
	}
	pool, err := db.DB()
	if err != nil {
		return err
	}
	defer pool.Close()
	flow, err := workflow.NewTestWorkflow(ctx, postgres.NewStore(db), chat, cfg.DeepSeek.Model)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(os.Stdout)
	if stream {
		return runStream(ctx, flow, input, encoder)
	}
	result, err := flow.Run(ctx, input)
	if result.SessionID != "" {
		if encodeErr := encoder.Encode(result); encodeErr != nil {
			return encodeErr
		}
	}
	return err
}

type streamEvent struct {
	Type   string                   `json:"type"`
	Text   string                   `json:"text,omitempty"`
	Result *workflow.TestTurnResult `json:"result,omitempty"`
}

func runStream(ctx context.Context, flow *workflow.TestWorkflow, input workflow.TestTurnInput, encoder *json.Encoder) error {
	result, err := flow.Stream(ctx, input, func(text string) error {
		return encoder.Encode(streamEvent{Type: "delta", Text: text})
	})
	if err != nil {
		return errors.Join(err, encoder.Encode(streamEvent{Type: "error", Result: &result}))
	}
	return encoder.Encode(streamEvent{Type: "done", Result: &result})
}
