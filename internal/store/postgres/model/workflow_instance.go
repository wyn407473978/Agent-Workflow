package model

import (
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// WorkflowInstance 保存工作流实例的状态、恢复位置和关联结果。
type WorkflowInstance struct {
	// WorkflowID 是工作流实例的内部标识。
	WorkflowID pgtype.UUID `gorm:"column:workflow_id;type:uuid"`
	// SessionID 是实例所属的会话标识。
	SessionID pgtype.UUID `gorm:"column:session_id;type:uuid"`
	// Status 是工作流当前的业务状态。
	Status *string `gorm:"column:status;type:text"`
	// CurrentNode 是工作流恢复时继续执行的节点名称。
	CurrentNode *string `gorm:"column:current_node;type:text"`
	// IntentName 是最近一次识别到的业务意图。
	IntentName *string `gorm:"column:intent_name;type:text"`
	// WorkflowVersion 是实例采用的工作流定义版本。
	WorkflowVersion *string `gorm:"column:workflow_version;type:text"`
	// StateRevision 是持久化修订号，从 1 开始，在业务状态或上下文更新时递增。
	StateRevision *int64 `gorm:"column:state_revision;type:bigint"`
	// DraftVersion 是实例当前关联的草稿版本，从 1 开始。
	DraftVersion *int64 `gorm:"column:draft_version;type:bigint"`
	// ApprovalID 是当前有效的审批单标识，尚未申请或审批失效时为空。
	ApprovalID pgtype.UUID `gorm:"column:approval_id;type:uuid"`
	// ActivityID 是已经核实的活动业务标识，结果未核实时为空。
	ActivityID *string `gorm:"column:activity_id;type:text"`
	// LastErrorCode 是最近一次影响工作流的错误代码。
	LastErrorCode *string `gorm:"column:last_error_code;type:text"`
	// LastErrorDetail 是该错误的脱敏详情。
	LastErrorDetail json.RawMessage `gorm:"column:last_error_detail;type:jsonb"`
	// CreatedAt 是实例创建时间。
	CreatedAt *time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime:false"`
	// UpdatedAt 是实例最近更新时间。
	UpdatedAt *time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime:false"`
}

func (WorkflowInstance) TableName() string { return "agent_workflow.workflow_instance" }
