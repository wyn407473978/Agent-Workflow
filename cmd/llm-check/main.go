package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"Agent-Workflow/internal/config"
	"Agent-Workflow/internal/llm"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	chat, err := llm.NewDeepSeek(ctx, cfg.DeepSeek)
	if err != nil {
		return err
	}
	reply, err := chat.Generate(ctx, []*schema.Message{schema.UserMessage("Reply with only OK.")}, model.WithMaxTokens(16))
	if err != nil {
		message := strings.ReplaceAll(err.Error(), strings.TrimSpace(cfg.DeepSeek.APIKey), "[redacted]")
		return fmt.Errorf("DeepSeek request failed: %s", message)
	}
	if reply == nil || strings.TrimSpace(reply.Content) == "" {
		return errors.New("DeepSeek returned no text")
	}
	fmt.Println(strings.TrimSpace(reply.Content))
	return nil
}
