package model

import (
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// MarketingActivity 保存模拟活动服务创建的营销活动结果。
type MarketingActivity struct {
	// ActivityID 是营销活动的业务标识。
	ActivityID *string `gorm:"column:activity_id;type:text"`
	// IdempotencyKey 是创建活动及回查结果使用的幂等键。
	IdempotencyKey *string `gorm:"column:idempotency_key;type:text"`
	// RequestHash 是创建该活动时规范化请求的摘要。
	RequestHash *string `gorm:"column:request_hash;type:text"`
	// WorkflowID 是创建该活动的工作流标识。
	WorkflowID pgtype.UUID `gorm:"column:workflow_id;type:uuid"`
	// DraftVersion 是创建活动时采用的草稿版本。
	DraftVersion *int64 `gorm:"column:draft_version;type:bigint"`
	// AccountID 是活动所属的营销账户标识。
	AccountID *string `gorm:"column:account_id;type:text"`
	// CreatedBy 是发起创建的用户标识。
	CreatedBy *string `gorm:"column:created_by;type:text"`
	// ActivityName 是已创建的营销活动名称。
	ActivityName *string `gorm:"column:activity_name;type:text"`
	// AudienceRule 是创建时采用的结构化人群筛选条件。
	AudienceRule json.RawMessage `gorm:"column:audience_rule;type:jsonb"`
	// AudienceSnapshotID 是创建时采用的人群查询快照标识。
	AudienceSnapshotID *string `gorm:"column:audience_snapshot_id;type:text"`
	// AudienceCount 是创建时记录的目标人群数量。
	AudienceCount *int64 `gorm:"column:audience_count;type:bigint"`
	// TemplateID 是活动发送使用的消息模板标识。
	TemplateID *string `gorm:"column:template_id;type:text"`
	// ScheduledAt 是活动计划发送时间。
	ScheduledAt *time.Time `gorm:"column:scheduled_at;type:timestamptz"`
	// TimeZone 是发送时间使用的 IANA 业务时区，例如 Asia/Shanghai。
	TimeZone *string `gorm:"column:timezone;type:text"`
	// EstimatedCostMinor 是创建时以货币最小单位计量的预估费用。
	EstimatedCostMinor *int64 `gorm:"column:estimated_cost_minor;type:bigint"`
	// Currency 是费用使用的币种代码，例如 CNY。
	Currency *string `gorm:"column:currency;type:text"`
	// Status 是营销活动当前状态，模拟创建时为 SCHEDULED。
	Status *string `gorm:"column:status;type:text"`
	// CreatedAt 是活动创建时间。
	CreatedAt *time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime:false"`
	// UpdatedAt 是活动最近更新时间。
	UpdatedAt *time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime:false"`
}

func (MarketingActivity) TableName() string { return "agent_workflow.marketing_activity" }
