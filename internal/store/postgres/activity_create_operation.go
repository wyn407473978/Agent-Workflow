package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"Agent-Workflow/internal/store/postgres/model"
	"github.com/jackc/pgx/v5/pgtype"
	"gorm.io/gorm"
)

const (
	creationReady                 = "READY"
	creationInFlight              = "IN_FLIGHT"
	creationRetryWait             = "RETRY_WAIT"
	creationPendingReconciliation = "PENDING_RECONCILIATION"
	creationReconciling           = "RECONCILING"
	creationSucceeded             = "SUCCEEDED"
	creationFailed                = "FAILED"
)

func (q *queries) GetActivityCreation(ctx context.Context, key string) (model.ActivityCreateOperation, error) {
	return findOne[model.ActivityCreateOperation](q.db.WithContext(ctx).Where("idempotency_key = ?", key))
}

func (q *queries) GetActivityCreationForDraft(ctx context.Context, workflowID pgtype.UUID, version int64) (model.ActivityCreateOperation, error) {
	return findOne[model.ActivityCreateOperation](q.db.WithContext(ctx).
		Where("workflow_id = ? AND draft_version = ?", workflowID, version))
}

// EnqueueActivityCreation 为一次授权生成稳定幂等键。同一草稿再次提交相同请求时复用原操作。
func (tx *Tx) EnqueueActivityCreation(ctx context.Context, operation model.ActivityCreateOperation) (model.ActivityCreateOperation, bool, error) {
	if !operation.WorkflowID.Valid || operation.DraftVersion == nil || *operation.DraftVersion < 1 ||
		!operation.ConfirmationEventID.Valid || !operation.AuthorizationEventID.Valid || operation.CreatedAt == nil ||
		operation.RequestHash == nil || *operation.RequestHash == "" || !json.Valid(operation.RequestJSON) {
		return model.ActivityCreateOperation{}, false, fmt.Errorf("%w: incomplete activity creation request", ErrInvalidArgument)
	}
	key := fmt.Sprintf("activity-create:v1:%s:%d", operation.WorkflowID, *operation.DraftVersion)
	if operation.IdempotencyKey != nil && *operation.IdempotencyKey != key {
		return model.ActivityCreateOperation{}, false, fmt.Errorf("%w: activity creation key does not match the draft", ErrInvalidArgument)
	}
	if err := tx.LockWorkflows(ctx, operation.WorkflowID); err != nil {
		return model.ActivityCreateOperation{}, false, err
	}
	if err := tx.LockActivityCreations(ctx, key); err != nil {
		return model.ActivityCreateOperation{}, false, err
	}
	existing, err := tx.GetActivityCreationForDraft(ctx, operation.WorkflowID, *operation.DraftVersion)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return model.ActivityCreateOperation{}, false, err
	}
	byKey, keyErr := tx.GetActivityCreation(ctx, key)
	if keyErr != nil && !errors.Is(keyErr, gorm.ErrRecordNotFound) {
		return model.ActivityCreateOperation{}, false, keyErr
	}
	if err == nil || keyErr == nil {
		if err != nil || keyErr != nil || existing.IdempotencyKey == nil || *existing.IdempotencyKey != key ||
			byKey.WorkflowID != operation.WorkflowID || byKey.DraftVersion == nil || *byKey.DraftVersion != *operation.DraftVersion {
			return model.ActivityCreateOperation{}, false, fmt.Errorf("%w: activity creation key and draft disagree", ErrDataIntegrity)
		}
		if existing.RequestHash == nil || *existing.RequestHash != *operation.RequestHash {
			return model.ActivityCreateOperation{}, false, fmt.Errorf("%w: activity creation request hash differs", ErrConflict)
		}
		return existing, false, nil
	}
	operation.IdempotencyKey = &key
	operation.Status = new(creationReady)
	operation.AttemptCount = new(int32(0))
	operation.ReconcileCount = new(int32(0))
	operation.NextAttemptAt = operation.CreatedAt
	operation.UpdatedAt = operation.CreatedAt
	operation.ActivityID = nil
	operation.LeaseToken = pgtype.UUID{}
	operation.LeaseExpiresAt = nil
	operation.LastErrorCode = nil
	operation.LastErrorDetail = nil
	operation.FinishedAt = nil
	if err := tx.db.WithContext(ctx).Create(&operation).Error; err != nil {
		return model.ActivityCreateOperation{}, false, err
	}
	return operation, true, nil
}

// ListDueActivityCreations 扫描可认领的操作和需要恢复的过期租约。
func (q *queries) ListDueActivityCreations(ctx context.Context, at time.Time, limit int) ([]model.ActivityCreateOperation, error) {
	query := q.db.WithContext(ctx).
		Where("((status IN ? AND next_attempt_at <= ?) OR (status IN ? AND lease_expires_at <= ?))",
			[]string{creationReady, creationRetryWait, creationPendingReconciliation}, at,
			[]string{creationInFlight, creationReconciling}, at).
		Order("COALESCE(lease_expires_at, next_attempt_at), idempotency_key")
	return findList[model.ActivityCreateOperation](query, limit)
}

// RescheduleExpiredActivityCreation 将过期的创建或核查租约交回待核查队列。
func (tx *Tx) RescheduleExpiredActivityCreation(ctx context.Context, key string, at time.Time) error {
	if at.IsZero() {
		return fmt.Errorf("%w: recovery time is required", ErrInvalidArgument)
	}
	if err := tx.LockActivityCreations(ctx, key); err != nil {
		return err
	}
	if _, err := tx.GetActivityCreation(ctx, key); err != nil {
		return err
	}
	return updateOne(tx.db.WithContext(ctx).Model(&model.ActivityCreateOperation{}).
		Where("idempotency_key = ? AND status IN ?", key, []string{creationInFlight, creationReconciling}).
		Where("lease_expires_at <= LEAST(?, clock_timestamp())", at).
		Updates(map[string]any{
			"status":           creationPendingReconciliation,
			"next_attempt_at":  gorm.Expr("clock_timestamp()"),
			"lease_token":      nil,
			"lease_expires_at": nil,
			"updated_at":       gorm.Expr("clock_timestamp()"),
		}))
}

// ClaimActivityCreation 认领待创建或待核查的任务，并分别累计尝试次数。
func (tx *Tx) ClaimActivityCreation(ctx context.Context, key string, at time.Time, lease Lease) (model.ActivityCreateOperation, error) {
	if err := lease.validate(at); err != nil {
		return model.ActivityCreateOperation{}, err
	}
	if err := tx.LockActivityCreations(ctx, key); err != nil {
		return model.ActivityCreateOperation{}, err
	}
	operation, err := tx.GetActivityCreation(ctx, key)
	if err != nil {
		return model.ActivityCreateOperation{}, err
	}
	if operation.Status == nil {
		return model.ActivityCreateOperation{}, ErrConflict
	}
	updates := map[string]any{
		"lease_token":      lease.Token,
		"lease_expires_at": lease.ExpiresAt,
		"next_attempt_at":  nil,
		"updated_at":       gorm.Expr("clock_timestamp()"),
	}
	switch *operation.Status {
	case creationReady, creationRetryWait:
		updates["status"] = creationInFlight
		updates["attempt_count"] = gorm.Expr("COALESCE(attempt_count, 0) + 1")
	case creationPendingReconciliation:
		updates["status"] = creationReconciling
		updates["reconcile_count"] = gorm.Expr("COALESCE(reconcile_count, 0) + 1")
	default:
		return model.ActivityCreateOperation{}, ErrConflict
	}
	if err := updateOne(tx.db.WithContext(ctx).Model(&model.ActivityCreateOperation{}).
		Where("idempotency_key = ? AND status = ?", key, *operation.Status).
		Where("next_attempt_at <= LEAST(?, clock_timestamp()) AND ? > clock_timestamp()", at, lease.ExpiresAt).
		Updates(updates)); err != nil {
		return model.ActivityCreateOperation{}, err
	}
	return tx.GetActivityCreation(ctx, key)
}

type ActivityCreationResult struct {
	Status          string
	ActivityID      *string
	NextAttemptAt   *time.Time
	LastErrorCode   *string
	LastErrorDetail json.RawMessage
}

// SaveActivityCreationResult 提交创建或核查的结果并释放租约。
func (tx *Tx) SaveActivityCreationResult(ctx context.Context, key string, token pgtype.UUID, at time.Time, result ActivityCreationResult) error {
	if !token.Valid || at.IsZero() {
		return fmt.Errorf("%w: activity creation result requires a lease token and time", ErrInvalidArgument)
	}
	updates := map[string]any{
		"status":            result.Status,
		"activity_id":       nil,
		"next_attempt_at":   nil,
		"last_error_code":   result.LastErrorCode,
		"last_error_detail": result.LastErrorDetail,
		"lease_token":       nil,
		"lease_expires_at":  nil,
		"updated_at":        gorm.Expr("clock_timestamp()"),
	}
	switch result.Status {
	case creationRetryWait, creationPendingReconciliation:
		if result.NextAttemptAt == nil {
			return fmt.Errorf("%w: rescheduling requires a next attempt time", ErrInvalidArgument)
		}
		updates["next_attempt_at"] = result.NextAttemptAt
	case creationSucceeded:
		if result.ActivityID == nil || *result.ActivityID == "" {
			return fmt.Errorf("%w: successful creation requires an activity ID", ErrInvalidArgument)
		}
		updates["activity_id"] = result.ActivityID
		updates["finished_at"] = gorm.Expr("clock_timestamp()")
	case creationFailed:
		updates["finished_at"] = gorm.Expr("clock_timestamp()")
	default:
		return fmt.Errorf("%w: unsupported activity creation result status %q", ErrInvalidArgument, result.Status)
	}
	if err := tx.LockActivityCreations(ctx, key); err != nil {
		return err
	}
	if _, err := tx.GetActivityCreation(ctx, key); err != nil {
		return err
	}
	query := tx.db.WithContext(ctx).Model(&model.ActivityCreateOperation{}).
		Where("idempotency_key = ? AND status IN ?", key, []string{creationInFlight, creationReconciling})
	return updateOne(withActiveLease(query, token, at).Updates(updates))
}
