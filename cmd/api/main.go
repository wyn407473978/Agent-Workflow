package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"Agent-Workflow/internal/config"
	"Agent-Workflow/internal/store/postgres"
	httptransport "Agent-Workflow/internal/transport/http"
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

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httptransport.NewRouter(db),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("listening on %s", cfg.HTTPAddr)
	return server.ListenAndServe()
}
