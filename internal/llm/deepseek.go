package llm

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"Agent-Workflow/internal/config"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/components/model"
)

// NewDeepSeek 创建可供 Go-Eino 工作流使用的聊天模型。
func NewDeepSeek(ctx context.Context, cfg config.DeepSeek) (model.ToolCallingChatModel, error) {
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		return nil, errors.New("DS_API_KEY is required")
	}
	modelName := strings.TrimSpace(cfg.Model)
	if modelName == "" {
		return nil, errors.New("deepseek.model (DS_MODEL) is required")
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	endpoint, err := url.Parse(base)
	if err != nil || (endpoint.Scheme != "https" && endpoint.Scheme != "http") || endpoint.Hostname() == "" ||
		endpoint.User != nil || endpoint.ForceQuery || endpoint.RawQuery != "" || strings.Contains(base, "#") {
		return nil, errors.New("DS_BASE_URL must be an absolute HTTP(S) API URL without credentials, query or fragment")
	}
	// 组件会自动追加 /chat/completions，同样接受配置中写入的完整端点。
	base = strings.TrimSuffix(base, "/chat/completions")
	chat, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		APIKey:         apiKey,
		BaseURL:        base,
		Model:          modelName,
		Timeout:        30 * time.Second,
		ThinkingConfig: &deepseek.ThinkingConfig{Type: "disabled"},
	})
	if err != nil {
		return nil, fmt.Errorf("initialize DeepSeek model: %w", err)
	}
	return chat, nil
}
