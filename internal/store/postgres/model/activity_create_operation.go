package model

import (
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// ActivityCreateOperation 保存活动创建请求、执行进度和结果核查信息。
type ActivityCreateOperation struct {
	// IdempotencyKey 是创建、重试及结果核查共用的稳定幂等键。
	IdempotencyKey *string `gorm:"column:idempotency_key;type:text"`
	// WorkflowID 是发起创建操作的工作流标识。
	WorkflowID pgtype.UUID `gorm:"column:workflow_id;type:uuid"`
	// DraftVersion 是本次创建操作绑定的草稿版本。
	DraftVersion *int64 `gorm:"column:draft_version;type:bigint"`
	// ConfirmationEventID 是用户确认该草稿版本的事件标识。
	ConfirmationEventID pgtype.UUID `gorm:"column:confirmation_event_id;type:uuid"`
	// AuthorizationEventID 是允许创建活动的事件标识，需审批时为审批通过事件。
	AuthorizationEventID pgtype.UUID `gorm:"column:authorization_event_id;type:uuid"`
	// RequestHash 是首次授权时冻结的规范化创建请求摘要。
	RequestHash *string `gorm:"column:request_hash;type:text"`
	// RequestJSON 是首次授权时冻结的活动创建命令。
	RequestJSON json.RawMessage `gorm:"column:request_json;type:jsonb"`
	// Status 是创建操作当前的执行或结果核查状态。
	Status *string `gorm:"column:status;type:text"`
	// ActivityID 是已经核实的活动业务标识，结果未核实时为空。
	ActivityID *string `gorm:"column:activity_id;type:text"`
	// AttemptCount 是发送创建请求的尝试次数。
	AttemptCount *int32 `gorm:"column:attempt_count;type:integer"`
	// ReconcileCount 是核查创建结果的尝试次数。
	ReconcileCount *int32 `gorm:"column:reconcile_count;type:integer"`
	// NextAttemptAt 是下次可调度创建或结果核查的时间。
	NextAttemptAt *time.Time `gorm:"column:next_attempt_at;type:timestamptz"`
	// LeaseToken 是当前创建或核查任务的认领令牌。
	LeaseToken pgtype.UUID `gorm:"column:lease_token;type:uuid"`
	// LeaseExpiresAt 是当前认领租约的到期时间。
	LeaseExpiresAt *time.Time `gorm:"column:lease_expires_at;type:timestamptz"`
	// LastErrorCode 是最近一次创建或结果核查失败的错误代码。
	LastErrorCode *string `gorm:"column:last_error_code;type:text"`
	// LastErrorDetail 是最近一次错误的脱敏诊断信息。
	LastErrorDetail json.RawMessage `gorm:"column:last_error_detail;type:jsonb"`
	// CreatedAt 是创建操作准备完成并落库的时间。
	CreatedAt *time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime:false"`
	// UpdatedAt 是创建操作最近更新时间。
	UpdatedAt *time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime:false"`
	// FinishedAt 是创建结果被确认为成功或失败的时间。
	FinishedAt *time.Time `gorm:"column:finished_at;type:timestamptz"`
}

func (ActivityCreateOperation) TableName() string { return "agent_workflow.activity_create_operation" }
