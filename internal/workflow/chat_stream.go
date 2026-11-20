package workflow

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/schema"
)

// TextDeltaSink 接收原样的可见文本分片。返回错误会停止读取并关闭模型流。
type TextDeltaSink func(text string) error

var ErrChatDeltaDelivery = errors.New("could not deliver chat delta")

// ConsumeChatStream 读取 Go-Eino 模型分片，同步转发文本并汇总完整回复。
// 该函数接管并关闭 reader。只有读到正常终止元数据和 EOF 才返回可保存的完整文本。
func ConsumeChatStream(ctx context.Context, reader *schema.StreamReader[*schema.Message], emit TextDeltaSink) (string, error) {
	if reader == nil {
		return "", errors.New("model returned no stream")
	}
	defer reader.Close()

	var text strings.Builder
	var finished bool
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		chunk, err := reader.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}
		if chunk == nil {
			continue
		}
		if meta := chunk.ResponseMeta; meta != nil && meta.FinishReason != "" {
			switch meta.FinishReason {
			case "stop", "length":
				finished = true
			default:
				return "", fmt.Errorf("model stream ended with %q", meta.FinishReason)
			}
		}
		if chunk.Content != "" {
			if err := writeTextDelta(emit, chunk.Content); err != nil {
				return "", err
			}
			text.WriteString(chunk.Content)
		}
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if !finished {
		return "", io.ErrUnexpectedEOF
	}
	if strings.TrimSpace(text.String()) == "" {
		return "", errors.New("model stream returned no text")
	}
	return text.String(), nil
}

func writeTextDelta(emit TextDeltaSink, text string) error {
	if emit == nil {
		return nil
	}
	if err := emit(text); err != nil {
		return errors.Join(ErrChatDeltaDelivery, err)
	}
	return nil
}
