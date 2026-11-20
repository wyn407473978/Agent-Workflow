package domain

import "time"

type ApprovalStatus string

const (
	ApprovalPending     ApprovalStatus = "PENDING"
	ApprovalApproved    ApprovalStatus = "APPROVED"
	ApprovalRejected    ApprovalStatus = "REJECTED"
	ApprovalWithdrawn   ApprovalStatus = "WITHDRAWN"
	ApprovalInvalidated ApprovalStatus = "INVALIDATED"
)

// ApprovalBasis 保存提交审批时的规则版本、门槛与触发依据。
type ApprovalBasis struct {
	RuleVersion       string   `json:"rule_version"`
	AudienceThreshold *int64   `json:"audience_threshold,omitempty"`
	BudgetThreshold   *Money   `json:"budget_threshold,omitempty"`
	Reasons           []string `json:"reasons"`
}

// ApprovalRequest 将审批单绑定到具体预览及完成该预览确认的事件。
type ApprovalRequest struct {
	ID                  ApprovalID     `json:"approval_id"`
	Target              PreviewRef     `json:"target"`
	ConfirmationEventID EventID        `json:"confirmation_event_id"`
	Status              ApprovalStatus `json:"status"`
	RequestEventID      EventID        `json:"request_event_id"`
	Basis               ApprovalBasis  `json:"basis"`
	DecisionEventID     EventID        `json:"decision_event_id,omitempty"`
	DecidedBy           string         `json:"decided_by,omitempty"`
	DecidedAt           *time.Time     `json:"decided_at,omitempty"`
	Reason              string         `json:"reason,omitempty"`
	InvalidatedAt       *time.Time     `json:"invalidated_at,omitempty"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
}
