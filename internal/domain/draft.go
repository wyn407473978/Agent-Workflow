package domain

import (
	"encoding/json"
	"time"
)

type SlotName string

const (
	SlotActivityName SlotName = "activity_name"
	SlotAudienceRule SlotName = "audience_rule"
	SlotScheduledAt  SlotName = "scheduled_at"
	SlotTimeZone     SlotName = "timezone"
	SlotTemplateID   SlotName = "template_id"
	SlotExtra        SlotName = "extra_slots"
)

// AudienceRule 是人群服务使用的结构化筛选条件。
type AudienceRule struct {
	LookbackDays *int         `json:"lookback_days,omitempty"`  // 向前统计消费的天数。
	MinSpendYuan *json.Number `json:"min_spend_yuan,omitempty"` // 消费门槛，单位为人民币元，保留输入精度。
}

// DraftSlots 保存用户可填写的参数。空指针表示该字段尚未填写。
type DraftSlots struct {
	ActivityName *string                    `json:"activity_name,omitempty"`
	AudienceRule *AudienceRule              `json:"audience_rule,omitempty"`
	ScheduledAt  *time.Time                 `json:"scheduled_at,omitempty"`
	TimeZone     *string                    `json:"timezone,omitempty"` // IANA 时区名称，例如 Asia/Shanghai。
	TemplateID   *string                    `json:"template_id,omitempty"`
	ExtraSlots   map[string]json.RawMessage `json:"extra_slots,omitempty"`
}

// DraftRef 标识一个确定的草稿版本。
type DraftRef struct {
	WorkflowID   WorkflowID `json:"workflow_id"`
	DraftVersion int64      `json:"draft_version"`
}

// PreviewRef 是确认或审批所引用的版本与预览。
type PreviewRef struct {
	DraftRef
	PreviewID PreviewID `json:"preview_id"`
}

type Preview struct {
	ID          PreviewID `json:"preview_id"`
	PublishedAt time.Time `json:"previewed_at"`
	ExpiresAt   time.Time `json:"preview_expires_at"`
}

type Confirmation struct {
	DraftVersion int64     `json:"confirmation_version"`
	EventID      EventID   `json:"confirmation_event_id"`
	ConfirmedBy  string    `json:"confirmed_by"`
	ConfirmedAt  time.Time `json:"confirmed_at"`
	ExpiresAt    time.Time `json:"confirmation_expires_at"`
}

type ValidationIssue struct {
	Slot    SlotName `json:"slot,omitempty"`
	Code    string   `json:"code"`
	Message string   `json:"message"`
}

type DraftValidation struct {
	SchemaValid         bool              `json:"schema_valid"`
	BusinessRulesPassed bool              `json:"business_rules_passed"`
	RuleVersion         string            `json:"rule_version,omitempty"`
	CheckedAt           time.Time         `json:"checked_at"`
	Issues              []ValidationIssue `json:"issues,omitempty"`
}

// ActivityDraft 保存一个版本的参数、证据、预览与确认。
type ActivityDraft struct {
	DraftRef
	Slots        DraftSlots       `json:"slots"`
	MissingSlots []SlotName       `json:"missing_slots,omitempty"`
	Evidence     DraftEvidence    `json:"evidence"`
	Validation   *DraftValidation `json:"validation,omitempty"`
	Preview      *Preview         `json:"preview,omitempty"`
	Confirmation *Confirmation    `json:"confirmation,omitempty"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}
