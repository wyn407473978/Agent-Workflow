package model

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// WorkflowSession 保存多轮会话的归属和当前工作流上下文。
type WorkflowSession struct {
	// SessionID 是多轮会话的内部标识。
	SessionID pgtype.UUID `gorm:"column:session_id;type:uuid"`
	// UserID 是通过身份验证的会话用户标识。
	UserID *string `gorm:"column:user_id;type:text"`
	// AccountID 是通过身份验证的营销账户标识。
	AccountID *string `gorm:"column:account_id;type:text"`
	// ActiveWorkflowID 是最近绑定的工作流标识，未绑定时为空。
	ActiveWorkflowID pgtype.UUID `gorm:"column:active_workflow_id;type:uuid"`
	// CreatedAt 是会话创建时间。
	CreatedAt *time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime:false"`
	// UpdatedAt 是会话最近更新时间。
	UpdatedAt *time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime:false"`
}

func (WorkflowSession) TableName() string { return "agent_workflow.workflow_session" }
