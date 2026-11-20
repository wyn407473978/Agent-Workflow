package domain

import (
	"encoding/json"
	"time"
)

type (
	SessionID  string // 多轮对话的内部标识。
	WorkflowID string // 一次工作流实例的内部标识。
	EventID    string // 一条已接收事件的内部标识。
	PreviewID  string // 一份已发布预览的内部标识。
	ApprovalID string // 一张审批单的内部标识。
	ActivityID string // 活动服务分配的业务标识。
	NodeName   string // 编排节点的稳定名称。
)

type WorkflowStatus string

const (
	WorkflowRunning               WorkflowStatus = "RUNNING"
	WorkflowWaitingInput          WorkflowStatus = "WAITING_INPUT"
	WorkflowWaitingConfirmation   WorkflowStatus = "WAITING_CONFIRMATION"
	WorkflowWaitingApproval       WorkflowStatus = "WAITING_APPROVAL"
	WorkflowCreating              WorkflowStatus = "CREATING"
	WorkflowPendingReconciliation WorkflowStatus = "PENDING_RECONCILIATION"
	WorkflowCompleted             WorkflowStatus = "COMPLETED"
	WorkflowRejected              WorkflowStatus = "REJECTED"
	WorkflowFailed                WorkflowStatus = "FAILED"
	WorkflowCancelled             WorkflowStatus = "CANCELLED"
)

// IsTerminal 判断实例是否已停止推进。
func (s WorkflowStatus) IsTerminal() bool {
	switch s {
	case WorkflowCompleted, WorkflowRejected, WorkflowFailed, WorkflowCancelled:
		return true
	default:
		return false
	}
}

type Intent string

const (
	IntentCreateActivity      Intent = "create_activity"
	IntentUpdateActivityDraft Intent = "update_activity_draft"
	IntentProvideSlots        Intent = "provide_slots"
	IntentConfirmActivity     Intent = "confirm_activity"
	IntentCancelWorkflow      Intent = "cancel_workflow"
	IntentQueryActivity       Intent = "query_activity"
	IntentOther               Intent = "other"
)

// Session 保存经过身份验证的会话归属和默认工作流引用。
type Session struct {
	ID               SessionID  `json:"session_id"`
	UserID           string     `json:"user_id"`
	AccountID        string     `json:"account_id"`
	ActiveWorkflowID WorkflowID `json:"active_workflow_id,omitempty"`
}

// WorkflowState 是加载或恢复实例时使用的业务快照。
type WorkflowState struct {
	ID             WorkflowID     `json:"workflow_id"`
	SessionID      SessionID      `json:"session_id"`
	Status         WorkflowStatus `json:"status"`
	CurrentNode    NodeName       `json:"current_node,omitempty"`
	Intent         Intent         `json:"intent,omitempty"`
	Version        string         `json:"workflow_version"` // 工作流定义版本。
	StateRevision  int64          `json:"state_revision"`   // 持久化修订号，用于并发更新校验。
	Draft          ActivityDraft  `json:"draft"`            // 实例当前指向的草稿版本。
	ApprovalID     ApprovalID     `json:"approval_id,omitempty"`
	IdempotencyKey string         `json:"idempotency_key,omitempty"`
	ActivityID     ActivityID     `json:"activity_id,omitempty"`
	LastError      *WorkflowError `json:"last_error,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// Transition 记录一次业务状态变更及其事件来源。
type Transition struct {
	WorkflowID    WorkflowID      `json:"workflow_id"`
	StateRevision int64           `json:"state_revision"`
	DraftVersion  int64           `json:"draft_version"`
	EventID       EventID         `json:"event_id"`
	From          WorkflowStatus  `json:"from_status,omitempty"` // 初始化实例时为空。
	To            WorkflowStatus  `json:"to_status"`
	Actor         Actor           `json:"actor"`
	ReasonCode    string          `json:"reason_code"`
	ReasonDetail  json.RawMessage `json:"reason_detail,omitempty"` // 脱敏的审计说明。
	CreatedAt     time.Time       `json:"created_at"`
}

type ErrorCode string

const (
	ErrorValidation             ErrorCode = "VALIDATION_ERROR"
	ErrorToolTimeoutRead        ErrorCode = "TOOL_TIMEOUT_READ"
	ErrorToolBusinessRejected   ErrorCode = "TOOL_BUSINESS_REJECTED"
	ErrorApprovalRejected       ErrorCode = "APPROVAL_REJECTED"
	ErrorCreateTimeoutUnknown   ErrorCode = "CREATE_TIMEOUT_UNKNOWN"
	ErrorCreatePermanent        ErrorCode = "CREATE_PERMANENT_ERROR"
	ErrorConcurrentModification ErrorCode = "CONCURRENT_MODIFICATION"
	ErrorModelInvalidJSON       ErrorCode = "MODEL_INVALID_JSON"
)

// WorkflowError 保存可持久化的故障信息。
type WorkflowError struct {
	Code      ErrorCode       `json:"code"`
	Detail    json.RawMessage `json:"detail,omitempty"` // 脱敏的结构化诊断信息。
	Retryable bool            `json:"retryable"`
}
