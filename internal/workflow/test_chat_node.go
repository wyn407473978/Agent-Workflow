package workflow

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

const testChatNodeName = "test_chat"

// testChatNode 接收已按顺序整理的对话消息，返回本轮模型回复。
type testChatNode struct {
	chat model.BaseChatModel
}

func (n testChatNode) invoke(ctx context.Context, messages []*schema.Message) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	reply, err := n.chat.Generate(ctx, messages, model.WithMaxTokens(512))
	if err != nil {
		return "", err
	}
	if reply == nil || strings.TrimSpace(reply.Content) == "" {
		return "", errors.New("test chat model returned no text")
	}
	return strings.TrimSpace(reply.Content), nil
}
