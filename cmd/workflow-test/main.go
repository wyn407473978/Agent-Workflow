package main

import (
	"context"
	"encoding/json"
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
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, workflow.TestTurnInput{SessionID: domain.SessionID(*session), RequestID: *request, Message: *message}); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, input workflow.TestTurnInput) error {
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
	result, err := flow.Run(ctx, input)
	if result.SessionID != "" {
		if encodeErr := json.NewEncoder(os.Stdout).Encode(result); encodeErr != nil {
			return encodeErr
		}
	}
	return err
}
