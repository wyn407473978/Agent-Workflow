package workflow

import (
	"time"

	"Agent-Workflow/internal/domain"
)

const (
	NodeLoadContext         domain.NodeName = "load_context"
	NodeRouteIntent         domain.NodeName = "route_intent"
	NodeExtractSlots        domain.NodeName = "extract_slots"
	NodeMergeDraft          domain.NodeName = "merge_draft"
	NodeValidateSchema      domain.NodeName = "validate_schema"
	NodeQueryAudience       domain.NodeName = "query_audience"
	NodeQueryTemplate       domain.NodeName = "query_template"
	NodeQueryQuota          domain.NodeName = "query_quota"
	NodeRetrievePolicy      domain.NodeName = "retrieve_policy"
	NodeValidateBusiness    domain.NodeName = "validate_business"
	NodePreview             domain.NodeName = "preview"
	NodeRequestConfirmation domain.NodeName = "request_confirmation"
	NodePreSubmitValidation domain.NodeName = "pre_submit_validation"
	NodeRequestApproval     domain.NodeName = "request_approval"
	NodeCreateActivity      domain.NodeName = "create_activity"
	NodeReconcileCreation   domain.NodeName = "reconcile_creation"
	NodeReplyUser           domain.NodeName = "reply_user"
)

type RouteIntentInput struct {
	Message  domain.UserMessagePayload `json:"message"`
	Workflow *domain.WorkflowState     `json:"workflow,omitempty"`
}

type RouteIntentOutput struct {
	Intent      domain.Intent `json:"intent"`
	Confidence  *float64      `json:"confidence,omitempty"`
	Explanation string        `json:"explanation,omitempty"`
}

type ExtractSlotsInput struct {
	Message string            `json:"message"`
	Slots   domain.DraftSlots `json:"slots"`
}

type ExtractSlotsOutput struct {
	Patch domain.SlotPatch `json:"slot_patch"`
}

type MergeDraftInput struct {
	Draft domain.ActivityDraft `json:"draft"`
	Patch domain.SlotPatch     `json:"slot_patch"`
}

type MergeDraftOutput struct {
	Draft        domain.ActivityDraft `json:"draft"`
	ChangedSlots []domain.SlotName    `json:"changed_slots"`
	ScheduleText string               `json:"schedule_text,omitempty"` // 待标准化的时间变更。
}

type ValidateSchemaInput struct {
	Draft           domain.ActivityDraft `json:"draft"`
	ReferenceAt     time.Time            `json:"reference_at"`
	DefaultTimeZone string               `json:"default_timezone"`
	ScheduleText    string               `json:"schedule_text,omitempty"`
}

type ValidateSchemaOutput struct {
	Valid        bool                     `json:"valid"`
	Slots        domain.DraftSlots        `json:"slots"` // 含已标准化的发送时间。
	MissingSlots []domain.SlotName        `json:"missing_slots,omitempty"`
	Issues       []domain.ValidationIssue `json:"issues,omitempty"`
}

type QueryAudienceInput struct {
	AccountID string              `json:"account_id"`
	Rule      domain.AudienceRule `json:"audience_rule"`
}

type QueryTemplateInput struct {
	AccountID  string `json:"account_id"`
	TemplateID string `json:"template_id"`
}

type QueryQuotaInput struct {
	AccountID     string        `json:"account_id"`
	EstimatedCost *domain.Money `json:"estimated_cost,omitempty"` // 允许先独立查询账户额度。
}

type RetrievePolicyInput struct {
	AccountID string            `json:"account_id"`
	Slots     domain.DraftSlots `json:"slots"`
}

// BusinessQueryResult 保留并行查询中成功的各项证据和各分支的错误。
type BusinessQueryResult struct {
	Evidence domain.DraftEvidence                     `json:"evidence"`
	Errors   map[domain.NodeName]domain.WorkflowError `json:"errors,omitempty"`
}

type ValidateBusinessInput struct {
	Session domain.Session       `json:"session"`
	Draft   domain.ActivityDraft `json:"draft"`
	Queries BusinessQueryResult  `json:"queries"`
	At      time.Time            `json:"at"`
}

type ValidateBusinessOutput struct {
	Validation    domain.DraftValidation `json:"validation"`
	EstimatedCost *domain.Money          `json:"estimated_cost,omitempty"`
	ApprovalBasis *domain.ApprovalBasis  `json:"approval_basis,omitempty"`
}

type BuildPreviewInput struct {
	Draft domain.ActivityDraft `json:"draft"`
	At    time.Time            `json:"at"`
}

// PreviewOutput 是校验通过后呈现给用户的活动预览。
type PreviewOutput struct {
	Target        domain.PreviewRef `json:"target"`
	ActivityName  string            `json:"activity_name"`
	AudienceCount int64             `json:"audience_count"`
	TemplateID    string            `json:"template_id"`
	ScheduledAt   time.Time         `json:"scheduled_at"`
	TimeZone      string            `json:"timezone"`
	EstimatedCost domain.Money      `json:"estimated_cost"`
	ExpiresAt     time.Time         `json:"preview_expires_at"`
}

type ReplyInput struct {
	Context RoundContext             `json:"context"`
	Preview *PreviewOutput           `json:"preview,omitempty"`
	Issues  []domain.ValidationIssue `json:"issues,omitempty"`
	Error   *domain.WorkflowError    `json:"error,omitempty"`
}
