package workflow

import (
	"time"

	"Agent-Workflow/internal/domain"
)

type ConfirmDraftInput struct {
	Context RoundContext      `json:"context"`
	Target  domain.PreviewRef `json:"target"`
}

// PreSubmitValidationInput 供用户确认后及审批通过后的重新校验使用。
type PreSubmitValidationInput struct {
	Session  domain.Session          `json:"session"`
	Workflow domain.WorkflowState    `json:"workflow"`
	Target   domain.PreviewRef       `json:"target"`
	Approval *domain.ApprovalRequest `json:"approval,omitempty"`
	At       time.Time               `json:"at"` // 实际提交时刻，用于检查证据和时间窗口。
}

type RequestApprovalInput struct {
	Target              domain.PreviewRef    `json:"target"`
	ConfirmationEventID domain.EventID       `json:"confirmation_event_id"`
	RequestEventID      domain.EventID       `json:"request_event_id"`
	Basis               domain.ApprovalBasis `json:"basis"`
}

type CreateActivityInput struct {
	IdempotencyKey string                 `json:"idempotency_key"`
	Command        domain.ActivityCommand `json:"command"`
}

type ReconcileCreationInput struct {
	IdempotencyKey string `json:"idempotency_key"`
}
