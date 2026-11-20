package workflow

import (
	"context"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// NewChatNode 创建可注册到 Go-Eino 工作流的模型节点。
// 输入为按时间排列的消息和回复 token 上限。Invoke 返回完整消息，Stream 返回原始消息分片。
// 节点只调用模型。流式调用方负责保持上下文有效并关闭读取器。
func NewChatNode(chat model.BaseChatModel, maxTokens int) (*compose.Lambda, error) {
	invoke := func(ctx context.Context, messages []*schema.Message, _ ...struct{}) (*schema.Message, error) {
		return chat.Generate(ctx, messages, model.WithMaxTokens(maxTokens))
	}
	stream := func(ctx context.Context, messages []*schema.Message, _ ...struct{}) (*schema.StreamReader[*schema.Message], error) {
		return chat.Stream(ctx, messages, model.WithMaxTokens(maxTokens))
	}
	return compose.AnyLambda(invoke, stream, nil, nil)
}
