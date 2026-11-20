package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"Agent-Workflow/internal/config"
	"Agent-Workflow/internal/llm"
	"Agent-Workflow/internal/store/postgres"
	httptransport "Agent-Workflow/internal/transport/http"
	"Agent-Workflow/internal/workflow"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	db, err := postgres.Open(ctx, cfg.Postgres)
	if err != nil {
		return err
	}
	pool, err := db.DB()
	if err != nil {
		return err
	}
	defer pool.Close()

	router := httptransport.NewRouter(db)
	var chatStream httptransport.WorkflowChatStream
	if strings.TrimSpace(cfg.DeepSeek.APIKey) != "" {
		chatStream, err = newChatStream(context.Background(), postgres.NewStore(db), cfg.DeepSeek)
		if err != nil {
			log.Printf("workflow chat unavailable: %v", err)
		}
	}
	httptransport.RegisterWorkflowChat(router, chatStream, cfg.DeepSeek.Model)
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("listening on %s", cfg.HTTPAddr)
	return server.ListenAndServe()
}

func newChatStream(ctx context.Context, store *postgres.Store, cfg config.DeepSeek) (httptransport.WorkflowChatStream, error) {
	chat, err := llm.NewDeepSeek(ctx, cfg)
	if err != nil {
		return nil, err
	}
	flow, err := workflow.NewTestWorkflow(ctx, store, chat, cfg.Model)
	if err != nil {
		return nil, err
	}
	return flow.Stream, nil
}
