package model

import (
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// WorkflowTransition 记录一次工作流业务状态变更，供审计和追溯。
type WorkflowTransition struct {
	// WorkflowID 是发生状态变更的工作流标识。
	WorkflowID pgtype.UUID `gorm:"column:workflow_id;type:uuid"`
	// StateRevision 是此次状态变更提交后的持久化修订号。
	StateRevision *int64 `gorm:"column:state_revision;type:bigint"`
	// EventID 是触发状态变更的事件标识。
	EventID pgtype.UUID `gorm:"column:event_id;type:uuid"`
	// DraftVersion 是状态变更时对应的草稿版本。
	DraftVersion *int64 `gorm:"column:draft_version;type:bigint"`
	// FromStatus 是变更前的状态，工作流初始化时为空。
	FromStatus *string `gorm:"column:from_status;type:text"`
	// ToStatus 是变更后的状态。
	ToStatus *string `gorm:"column:to_status;type:text"`
	// ReasonCode 是此次状态变更的原因代码。
	ReasonCode *string `gorm:"column:reason_code;type:text"`
	// ReasonDetail 是此次状态变更的脱敏说明。
	ReasonDetail json.RawMessage `gorm:"column:reason_detail;type:jsonb"`
	// ActorType 是触发状态变更的操作者类型。
	ActorType *string `gorm:"column:actor_type;type:text"`
	// ActorID 是触发状态变更的操作者标识。
	ActorID *string `gorm:"column:actor_id;type:text"`
	// CreatedAt 是状态变更发生的时间。
	CreatedAt *time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime:false"`
}

func (WorkflowTransition) TableName() string { return "agent_workflow.workflow_transition" }
