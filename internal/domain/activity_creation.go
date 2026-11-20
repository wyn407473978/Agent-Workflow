package domain

import (
	"encoding/json"
	"time"
)

type CreationStatus string

const (
	CreationReady                 CreationStatus = "READY"
	CreationInFlight              CreationStatus = "IN_FLIGHT"
	CreationRetryWait             CreationStatus = "RETRY_WAIT"
	CreationPendingReconciliation CreationStatus = "PENDING_RECONCILIATION"
	CreationReconciling           CreationStatus = "RECONCILING"
	CreationSucceeded             CreationStatus = "SUCCEEDED"
	CreationFailed                CreationStatus = "FAILED"
)

// ActivityCommand 保存授权时冻结的创建参数，重试沿用同一份内容。
type ActivityCommand struct {
	DraftRef
	AccountID          string                     `json:"account_id"`
	CreatedBy          string                     `json:"created_by"`
	ActivityName       string                     `json:"activity_name"`
	AudienceRule       AudienceRule               `json:"audience_rule"`
	AudienceSnapshotID string                     `json:"audience_snapshot_id"`
	AudienceCount      int64                      `json:"audience_count"`
	TemplateID         string                     `json:"template_id"`
	ScheduledAt        time.Time                  `json:"scheduled_at"`
	TimeZone           string                     `json:"timezone"`
	EstimatedCost      Money                      `json:"estimated_cost"`
	ExtraSlots         map[string]json.RawMessage `json:"extra_slots,omitempty"`
}

// ActivityCreation 保存创建命令、授权来源和任务恢复信息。
type ActivityCreation struct {
	IdempotencyKey       string          `json:"idempotency_key"`
	Command              ActivityCommand `json:"command"`
	ConfirmationEventID  EventID         `json:"confirmation_event_id"`
	AuthorizationEventID EventID         `json:"authorization_event_id"`
	RequestHash          string          `json:"request_hash"`
	Status               CreationStatus  `json:"status"`
	ActivityID           ActivityID      `json:"activity_id,omitempty"`
	AttemptCount         int32           `json:"attempt_count"`
	ReconcileCount       int32           `json:"reconcile_count"`
	NextAttemptAt        *time.Time      `json:"next_attempt_at,omitempty"`
	LastError            *WorkflowError  `json:"last_error,omitempty"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
	FinishedAt           *time.Time      `json:"finished_at,omitempty"`
}

type CreationOutcome string

const (
	OutcomeCreated    CreationOutcome = "CREATED"
	OutcomeNotCreated CreationOutcome = "NOT_CREATED"
	OutcomeUnknown    CreationOutcome = "UNKNOWN"
)

// CreationResult 是创建工具或按幂等键核查所得的结果。
type CreationResult struct {
	IdempotencyKey string          `json:"idempotency_key"`
	Outcome        CreationOutcome `json:"outcome"`
	ActivityID     ActivityID      `json:"activity_id,omitempty"`
	Error          *WorkflowError  `json:"error,omitempty"`
}
