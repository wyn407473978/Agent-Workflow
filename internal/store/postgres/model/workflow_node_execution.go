package model

import (
	"encoding/json"
	"github.com/jackc/pgx/v5/pgtype"
	"time"
)

// WorkflowNodeExecution 记录工作流节点的单次执行尝试。
type WorkflowNodeExecution struct {
	// ExecutionID 是本次节点执行尝试的内部标识。
	ExecutionID pgtype.UUID `gorm:"column:execution_id;type:uuid"`
	// WorkflowID 是节点所属的工作流标识。
	WorkflowID pgtype.UUID `gorm:"column:workflow_id;type:uuid"`
	// EventID 是本次节点执行对应的触发事件标识。
	EventID pgtype.UUID `gorm:"column:event_id;type:uuid"`
	// DraftVersion 是节点执行时对应的草稿版本。
	DraftVersion *int64 `gorm:"column:draft_version;type:bigint"`
	// NodeName 是本次执行的工作流节点名称。
	NodeName *string `gorm:"column:node_name;type:text"`
	// AttemptNo 是本轮节点执行尝试的序号。
	AttemptNo *int32 `gorm:"column:attempt_no;type:integer"`
	// ToolName 是节点本次调用的工具名称。
	ToolName *string `gorm:"column:tool_name;type:text"`
	// ModelName 是节点本次调用的模型名称。
	ModelName *string `gorm:"column:model_name;type:text"`
	// PromptName 是模型调用使用的提示词名称。
	PromptName *string `gorm:"column:prompt_name;type:text"`
	// PromptVersion 是本次使用的提示词版本。
	PromptVersion *string `gorm:"column:prompt_version;type:text"`
	// TraceID 是用于关联观测平台调用链的标识。
	TraceID *string `gorm:"column:trace_id;type:text"`
	// Status 是本次节点执行尝试的状态。
	Status *string `gorm:"column:status;type:text"`
	// InputJSON 是本次执行记录的脱敏输入。
	InputJSON json.RawMessage `gorm:"column:input_json;type:jsonb"`
	// OutputJSON 是本次执行记录的脱敏输出。
	OutputJSON json.RawMessage `gorm:"column:output_json;type:jsonb"`
	// ErrorCode 是本次尝试失败时记录的错误代码。
	ErrorCode *string `gorm:"column:error_code;type:text"`
	// StartedAt 是本次节点执行的开始时间。
	StartedAt *time.Time `gorm:"column:started_at;type:timestamptz"`
	// FinishedAt 是本次节点执行的结束时间。
	FinishedAt *time.Time `gorm:"column:finished_at;type:timestamptz"`
}

func (WorkflowNodeExecution) TableName() string { return "agent_workflow.workflow_node_execution" }
