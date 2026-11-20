package model

import (
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// ApprovalRequest 记录审批申请、申请依据和审批决定。
type ApprovalRequest struct {
	// ApprovalID 是审批单的内部标识。
	ApprovalID pgtype.UUID `gorm:"column:approval_id;type:uuid"`
	// WorkflowID 是被审批活动所属的工作流标识。
	WorkflowID pgtype.UUID `gorm:"column:workflow_id;type:uuid"`
	// DraftVersion 是审批申请绑定的草稿版本。
	DraftVersion *int64 `gorm:"column:draft_version;type:bigint"`
	// PreviewID 是发起审批所依据的预览标识。
	PreviewID pgtype.UUID `gorm:"column:preview_id;type:uuid"`
	// ConfirmationEventID 是发起审批所依据的用户确认事件标识。
	ConfirmationEventID pgtype.UUID `gorm:"column:confirmation_event_id;type:uuid"`
	// Status 是审批单当前状态，包括待处理、通过、拒绝、已撤回和已失效。
	Status *string `gorm:"column:status;type:text"`
	// RequestEventID 是触发审批申请的事件标识。
	RequestEventID pgtype.UUID `gorm:"column:request_event_id;type:uuid"`
	// RequestJSON 是提交审批时的快照，包含审批阈值和触发依据。
	RequestJSON json.RawMessage `gorm:"column:request_json;type:jsonb"`
	// DecisionEventID 是已生效的审批决定或撤回事件标识。
	DecisionEventID pgtype.UUID `gorm:"column:decision_event_id;type:uuid"`
	// DecidedBy 是处理审批或撤回的操作者标识。
	DecidedBy *string `gorm:"column:decided_by;type:text"`
	// DecidedAt 是审批决定或撤回生效的时间。
	DecidedAt *time.Time `gorm:"column:decided_at;type:timestamptz"`
	// Reason 是审批决定或审批失效的原因。
	Reason *string `gorm:"column:reason;type:text"`
	// InvalidatedAt 是审批申请失效的时间。
	InvalidatedAt *time.Time `gorm:"column:invalidated_at;type:timestamptz"`
	// CreatedAt 是审批申请发起时间。
	CreatedAt *time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime:false"`
	// UpdatedAt 是审批单最近更新时间。
	UpdatedAt *time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime:false"`
}

func (ApprovalRequest) TableName() string { return "agent_workflow.approval_request" }
