package workflow

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"Agent-Workflow/internal/domain"
	"Agent-Workflow/internal/store/postgres"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	testVersion       = "test-v1"
	testOwner         = "local-workflow-test"
	testMemoryTurns   = 10
	testModelTimeout  = 30 * time.Second
	testLeaseDuration = 60 * time.Second
)

var (
	ErrTestSessionBusy = errors.New("session already has a running turn, retry with the same request ID")
	ErrTestModelFailed = errors.New("test model request failed, retry with the same session and request ID")
)

type TestWorkflow struct {
	store     *postgres.Store
	chain     compose.Runnable[[]*schema.Message, *schema.Message]
	modelName string
}

type TestTurnInput struct {
	SessionID domain.SessionID
	RequestID string
	Message   string
}

type TestTurnResult struct {
	SessionID domain.SessionID `json:"session_id"`
	RequestID string           `json:"request_id"`
	EventID   domain.EventID   `json:"event_id,omitempty"`
	RoundResult
}

// NewTestWorkflow 构建只有一个模型节点的工作流。会话和数据库操作由调用流程处理。
func NewTestWorkflow(ctx context.Context, store *postgres.Store, chat model.BaseChatModel, modelName string) (*TestWorkflow, error) {
	node, err := newTestChatNode(chat)
	if err != nil {
		return nil, err
	}
	chain, err := compose.NewChain[[]*schema.Message, *schema.Message]().
		AppendLambda(node, compose.WithNodeKey(testChatNodeName)).Compile(ctx, compose.WithGraphName(testVersion))
	if err != nil {
		return nil, fmt.Errorf("compile test workflow: %w", err)
	}
	return &TestWorkflow{store: store, chain: chain, modelName: modelName}, nil
}

// Run 处理一轮测试对话。相同会话和请求标识复用已保存的成功结果。
func (w *TestWorkflow) Run(ctx context.Context, input TestTurnInput) (TestTurnResult, error) {
	return w.run(ctx, input, w.invokeChat, nil)
}

// Stream 逐块发送本轮回复，并在完整回复落库后返回回合结果。
// 已完成的重发只发送一次已保存的完整文本。传入 nil 捕获最终结果但不转发分片。
func (w *TestWorkflow) Stream(ctx context.Context, input TestTurnInput, emit TextDeltaSink) (TestTurnResult, error) {
	return w.run(ctx, input, func(ctx context.Context, messages []*schema.Message) (string, error) {
		reader, err := w.chain.Stream(ctx, messages)
		if err != nil {
			return "", err
		}
		return ConsumeChatStream(ctx, reader, emit)
	}, emit)
}

func (w *TestWorkflow) invokeChat(ctx context.Context, messages []*schema.Message) (string, error) {
	reply, err := w.chain.Invoke(ctx, messages)
	if err != nil {
		return "", err
	}
	if reply == nil || strings.TrimSpace(reply.Content) == "" {
		return "", errors.New("test chat model returned no text")
	}
	return strings.TrimSpace(reply.Content), nil
}

func (w *TestWorkflow) run(ctx context.Context, input TestTurnInput, respond func(context.Context, []*schema.Message) (string, error), emitReplay TextDeltaSink) (TestTurnResult, error) {
	var sessionID pgtype.UUID
	if input.SessionID == "" {
		sessionID = newTestUUID()
	} else if err := sessionID.Scan(string(input.SessionID)); err != nil {
		return TestTurnResult{}, fmt.Errorf("session ID must be a UUID: %w", postgres.ErrInvalidArgument)
	}
	requestID := strings.TrimSpace(input.RequestID)
	if requestID == "" {
		requestID = newTestUUID().String()
	}
	result := TestTurnResult{SessionID: domain.SessionID(sessionID.String()), RequestID: requestID}
	message := strings.TrimSpace(input.Message)
	if message == "" {
		return result, fmt.Errorf("message is empty: %w", postgres.ErrInvalidArgument)
	}
	payload, err := json.Marshal(domain.UserMessagePayload{Text: message})
	if err != nil {
		return result, err
	}
	hash := sha256.Sum256(payload)
	key := postgres.EventKey{Source: "workflow-test:" + sessionID.String(), SourceEventID: requestID}
	turn, err := w.beginTurn(ctx, sessionID, key, payload, hex.EncodeToString(hash[:]))
	if err != nil {
		return result, err
	}
	result.EventID = domain.EventID(turn.event.EventID.String())
	if turn.replay != nil {
		result.RoundResult = *turn.replay
		return result, writeTextDelta(emitReplay, turn.replay.Reply.Text)
	}

	messages := append(turn.history, schema.UserMessage(message))
	modelCtx, cancel := context.WithTimeout(ctx, testModelTimeout)
	text, responseErr := respond(modelCtx, messages)
	cancel()
	var failureCode string
	if responseErr != nil {
		text = ""
		failureCode = "TEST_MODEL_ERROR"
		if errors.Is(responseErr, ErrChatDeltaDelivery) {
			failureCode = "TEST_STREAM_DELIVERY_ERROR"
		}
	}

	// 客户端断开后仍结束数据库记录，重发时可以读取已提交的结果或重试。
	finishCtx, stop := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer stop()
	result.RoundResult, err = w.finishTurn(finishCtx, sessionID, key, turn, text, failureCode)
	if err != nil {
		return result, err
	}
	if errors.Is(responseErr, ErrChatDeltaDelivery) {
		return result, responseErr
	}
	if responseErr != nil {
		return result, ErrTestModelFailed
	}
	return result, nil
}

func newTestUUID() pgtype.UUID {
	id := pgtype.UUID{Valid: true}
	_, _ = rand.Read(id.Bytes[:])
	id.Bytes[6] = id.Bytes[6]&0x0f | 0x40
	id.Bytes[8] = id.Bytes[8]&0x3f | 0x80
	return id
}
