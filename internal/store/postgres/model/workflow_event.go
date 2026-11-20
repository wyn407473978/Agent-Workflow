package model

import (
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// WorkflowEvent 保存入站事件、处理进度和恢复所需的信息。
type WorkflowEvent struct {
	// EventID 是入站事件的内部标识。
	EventID pgtype.UUID `gorm:"column:event_id;type:uuid"`
	// SessionID 是事件所属的会话标识。
	SessionID pgtype.UUID `gorm:"column:session_id;type:uuid"`
	// WorkflowID 是事件目标工作流的标识，尚未路由时可为空。
	WorkflowID pgtype.UUID `gorm:"column:workflow_id;type:uuid"`
	// Source 是经过认证的事件来源命名空间。
	Source *string `gorm:"column:source;type:text"`
	// SourceEventID 是上游稳定的事件标识，与事件来源共同用于去重。
	SourceEventID *string `gorm:"column:source_event_id;type:text"`
	// EventType 是事件的业务类型，例如用户消息或审批回调。
	EventType *string `gorm:"column:event_type;type:text"`
	// ActorType 是触发事件的操作者类型。
	ActorType *string `gorm:"column:actor_type;type:text"`
	// ActorID 是触发事件的操作者标识。
	ActorID *string `gorm:"column:actor_id;type:text"`
	// RequestHash 是规范化入站请求的摘要，用于识别重复投递冲突。
	RequestHash *string `gorm:"column:request_hash;type:text"`
	// Payload 是脱敏后的事件载荷。
	Payload json.RawMessage `gorm:"column:payload;type:jsonb"`
	// ResultJSON 是处理检查点或最终处理结果。
	ResultJSON json.RawMessage `gorm:"column:result_json;type:jsonb"`
	// Status 是事件当前的处理状态。
	Status *string `gorm:"column:status;type:text"`
	// AttemptCount 是该事件已尝试处理的次数。
	AttemptCount *int32 `gorm:"column:attempt_count;type:integer"`
	// NextAttemptAt 是事件下次可被处理的时间。
	NextAttemptAt *time.Time `gorm:"column:next_attempt_at;type:timestamptz"`
	// LeaseToken 是当前处理任务的认领令牌。
	LeaseToken pgtype.UUID `gorm:"column:lease_token;type:uuid"`
	// LeaseExpiresAt 是当前处理租约的到期时间。
	LeaseExpiresAt *time.Time `gorm:"column:lease_expires_at;type:timestamptz"`
	// LastErrorCode 是最近一次处理失败的错误代码。
	LastErrorCode *string `gorm:"column:last_error_code;type:text"`
	// ReferenceAt 是解析相对时间及回放事件时使用的业务基准时刻。
	ReferenceAt *time.Time `gorm:"column:reference_at;type:timestamptz"`
	// CreatedAt 是事件收到并记录的时间。
	CreatedAt *time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime:false"`
	// UpdatedAt 是事件记录最近更新时间。
	UpdatedAt *time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime:false"`
	// ProcessedAt 是事件处理结束的时间。
	ProcessedAt *time.Time `gorm:"column:processed_at;type:timestamptz"`
}

func (WorkflowEvent) TableName() string { return "agent_workflow.workflow_event" }
