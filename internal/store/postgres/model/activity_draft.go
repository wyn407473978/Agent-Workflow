package model

import (
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// ActivityDraft 保存某个版本的活动草稿、查询证据、预览和用户确认。
type ActivityDraft struct {
	// WorkflowID 是草稿所属的工作流标识。
	WorkflowID pgtype.UUID `gorm:"column:workflow_id;type:uuid"`
	// DraftVersion 是该活动草稿的版本。
	DraftVersion *int64 `gorm:"column:draft_version;type:bigint"`
	// ActivityName 是用户拟定的活动名称。
	ActivityName *string `gorm:"column:activity_name;type:text"`
	// AudienceRule 是结构化的人群筛选条件。
	AudienceRule json.RawMessage `gorm:"column:audience_rule;type:jsonb"`
	// ScheduledAt 是计划发送时间。
	ScheduledAt *time.Time `gorm:"column:scheduled_at;type:timestamptz"`
	// TimeZone 是发送时间使用的 IANA 业务时区，例如 Asia/Shanghai。
	TimeZone *string `gorm:"column:timezone;type:text"`
	// TemplateID 是计划使用的消息模板标识。
	TemplateID *string `gorm:"column:template_id;type:text"`
	// ExtraSlots 是结构化的扩展参数对象。
	ExtraSlots json.RawMessage `gorm:"column:extra_slots;type:jsonb"`
	// MissingSlots 是仍需补充的字段名称数组。
	MissingSlots json.RawMessage `gorm:"column:missing_slots;type:jsonb"`
	// AudienceSnapshotID 是人群查询工具返回的快照标识。
	AudienceSnapshotID *string `gorm:"column:audience_snapshot_id;type:text"`
	// AudienceCount 是快照对应的人数，尚未查询时为空。
	AudienceCount *int64 `gorm:"column:audience_count;type:bigint"`
	// TemplateStatus 是工具查询时取得的模板审核状态。
	TemplateStatus *string `gorm:"column:template_status;type:text"`
	// EstimatedCostMinor 是以货币最小单位计量的预估费用，未知时为空。
	EstimatedCostMinor *int64 `gorm:"column:estimated_cost_minor;type:bigint"`
	// Currency 是预估费用使用的币种代码，例如 CNY。
	Currency *string `gorm:"column:currency;type:text"`
	// SnapshotJSON 是人群、模板、额度和政策的查询证据，含适用的查询时间及有效期。
	SnapshotJSON json.RawMessage `gorm:"column:snapshot_json;type:jsonb"`
	// RuleVersion 是校验、预览或审批时采用的业务规则版本。
	RuleVersion *string `gorm:"column:rule_version;type:text"`
	// ValidationJSON 是数据结构和业务规则的校验结果。
	ValidationJSON json.RawMessage `gorm:"column:validation_json;type:jsonb"`
	// PreviewID 是该版本发布的预览标识。
	PreviewID pgtype.UUID `gorm:"column:preview_id;type:uuid"`
	// PreviewedAt 是预览发布时间。
	PreviewedAt *time.Time `gorm:"column:previewed_at;type:timestamptz"`
	// PreviewExpiresAt 是预览失效时间。
	PreviewExpiresAt *time.Time `gorm:"column:preview_expires_at;type:timestamptz"`
	// ConfirmationVersion 是用户确认时绑定的草稿版本。
	ConfirmationVersion *int64 `gorm:"column:confirmation_version;type:bigint"`
	// ConfirmationEventID 是完成用户确认的事件标识。
	ConfirmationEventID pgtype.UUID `gorm:"column:confirmation_event_id;type:uuid"`
	// ConfirmedBy 是确认该草稿版本的用户标识。
	ConfirmedBy *string `gorm:"column:confirmed_by;type:text"`
	// ConfirmedAt 是用户确认时间。
	ConfirmedAt *time.Time `gorm:"column:confirmed_at;type:timestamptz"`
	// ConfirmationExpiresAt 是用户确认失效时间。
	ConfirmationExpiresAt *time.Time `gorm:"column:confirmation_expires_at;type:timestamptz"`
	// CreatedAt 是该草稿版本的创建时间。
	CreatedAt *time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime:false"`
	// UpdatedAt 是该草稿版本的最近更新时间。
	UpdatedAt *time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime:false"`
}

func (ActivityDraft) TableName() string { return "agent_workflow.activity_draft" }
