package workflow

import (
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
)

const testChatNodeName = "test_chat"

// newTestChatNode 配置测试对话节点。输入是会话历史加本轮消息，输出是完整回复或逐块的模型消息。
// 测试节点使用通用模型节点，并限制每轮最多生成 512 个 token。
func newTestChatNode(chat model.BaseChatModel) (*compose.Lambda, error) {
	return NewChatNode(chat, 512)
}
