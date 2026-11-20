package domain

import (
	"encoding/json"
	"time"
)

type EventType string

const (
	EventUserMessage       EventType = "USER_MESSAGE"
	EventUserConfirm       EventType = "USER_CONFIRM"
	EventUserCancel        EventType = "USER_CANCEL"
	EventApprovalApproved  EventType = "APPROVAL_APPROVED"
	EventApprovalRejected  EventType = "APPROVAL_REJECTED"
	EventApprovalWithdrawn EventType = "APPROVAL_WITHDRAWN"
	EventCreateResult      EventType = "CREATE_RESULT"
	EventReconcileResult   EventType = "RECONCILE_RESULT"
	EventScheduledRetry    EventType = "SCHEDULED_RETRY"
)

type EventStatus string

const (
	EventReceived   EventStatus = "RECEIVED"
	EventProcessing EventStatus = "PROCESSING"
	EventRetryWait  EventStatus = "RETRY_WAIT"
	EventApplied    EventStatus = "APPLIED"
	EventIgnored    EventStatus = "IGNORED"
	EventFailed     EventStatus = "FAILED"
)

type ActorType string

const (
	ActorUser     ActorType = "USER"
	ActorApprover ActorType = "APPROVER"
	ActorSystem   ActorType = "SYSTEM"
)

// Actor 保存事件或状态迁移的已验证操作者。
type Actor struct {
	Type ActorType `json:"type"`
	ID   string    `json:"id"`
}

// WorkflowEvent 是一次输入或恢复事件。普通消息可在路由前缺少工作流标识。
type WorkflowEvent struct {
	ID            EventID         `json:"event_id"`
	SessionID     SessionID       `json:"session_id"`
	WorkflowID    WorkflowID      `json:"workflow_id,omitempty"`
	Type          EventType       `json:"event_type"`
	Source        string          `json:"source"`
	SourceEventID string          `json:"source_event_id"`
	RequestHash   string          `json:"request_hash"`
	Actor         Actor           `json:"actor"`
	ReferenceAt   time.Time       `json:"reference_at"` // 相对时间解析与回放的业务基准时刻。
	CreatedAt     time.Time       `json:"created_at"`
	Payload       json.RawMessage `json:"payload,omitempty"` // 根据事件类型解析的脱敏载荷。
}

type UserMessagePayload struct {
	Text           string  `json:"text"`
	ReplyToEventID EventID `json:"reply_to_event_id,omitempty"`
}

type UserConfirmationPayload struct {
	Target PreviewRef `json:"target"`
}

type UserCancellationPayload struct {
	Reason string `json:"reason,omitempty"`
}

type ApprovalDecisionPayload struct {
	ApprovalID          ApprovalID `json:"approval_id"`
	Target              PreviewRef `json:"target"`
	ConfirmationEventID EventID    `json:"confirmation_event_id"`
	Reason              string     `json:"reason,omitempty"`
}

type ScheduledRetryPayload struct {
	Node           NodeName `json:"node_name,omitempty"`
	IdempotencyKey string   `json:"idempotency_key,omitempty"`
}
