package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"Agent-Workflow/internal/store/postgres/model"
	"github.com/jackc/pgx/v5/pgtype"
	"gorm.io/gorm"
)

func (q *queries) GetMarketingActivity(ctx context.Context, id string) (model.MarketingActivity, error) {
	return findOne[model.MarketingActivity](q.db.WithContext(ctx).Where("activity_id = ?", id))
}

func (q *queries) GetMarketingActivityByKey(ctx context.Context, key string) (model.MarketingActivity, error) {
	return findOne[model.MarketingActivity](q.db.WithContext(ctx).Where("idempotency_key = ?", key))
}

// RecordMarketingActivity 写入模拟活动服务的结果。重复请求复用已存在的活动。
func (tx *Tx) RecordMarketingActivity(ctx context.Context, activity model.MarketingActivity, token pgtype.UUID) (model.MarketingActivity, bool, error) {
	if activity.IdempotencyKey == nil || activity.RequestHash == nil || *activity.RequestHash == "" ||
		activity.ActivityID == nil || *activity.ActivityID == "" || !activity.WorkflowID.Valid ||
		activity.DraftVersion == nil || *activity.DraftVersion < 1 || !token.Valid || activity.CreatedAt == nil {
		return model.MarketingActivity{}, false, fmt.Errorf("%w: incomplete marketing activity result", ErrInvalidArgument)
	}
	key := *activity.IdempotencyKey
	if err := tx.LockActivityCreations(ctx, key); err != nil {
		return model.MarketingActivity{}, false, err
	}
	if err := tx.LockActivities(ctx, key); err != nil {
		return model.MarketingActivity{}, false, err
	}
	existing, err := tx.GetMarketingActivityByKey(ctx, key)
	if err == nil {
		if existing.RequestHash == nil || *existing.RequestHash != *activity.RequestHash {
			return model.MarketingActivity{}, false, fmt.Errorf("%w: marketing activity request hash differs", ErrConflict)
		}
		return existing, false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return model.MarketingActivity{}, false, err
	}
	if _, err := tx.GetActivityCreation(ctx, key); err != nil {
		return model.MarketingActivity{}, false, err
	}
	query := tx.db.WithContext(ctx).
		Where("idempotency_key = ? AND status = ? AND request_hash = ?", key, creationInFlight, *activity.RequestHash).
		Where("workflow_id = ? AND draft_version = ?", activity.WorkflowID, *activity.DraftVersion).
		Where("lease_token = ? AND lease_expires_at > clock_timestamp()", token)
	if _, err := findOne[model.ActivityCreateOperation](query); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = fmt.Errorf("%w: active creation lease and request are required", ErrConflict)
		}
		return model.MarketingActivity{}, false, err
	}
	activity.Status = new("SCHEDULED")
	activity.UpdatedAt = activity.CreatedAt
	db := tx.db.WithContext(ctx)
	if err := insertUnique(db, db.Where("activity_id = ?", *activity.ActivityID), &activity); err != nil {
		return model.MarketingActivity{}, false, err
	}
	return activity, true, nil
}

func (tx *Tx) UpdateMarketingActivityStatus(ctx context.Context, key, expectedStatus, status string, at time.Time) error {
	if expectedStatus == "" || status == "" || at.IsZero() {
		return fmt.Errorf("%w: expected status, status and time are required", ErrInvalidArgument)
	}
	if err := tx.LockActivities(ctx, key); err != nil {
		return err
	}
	if _, err := tx.GetMarketingActivityByKey(ctx, key); err != nil {
		return err
	}
	return updateOne(tx.db.WithContext(ctx).Model(&model.MarketingActivity{}).
		Where("idempotency_key = ? AND status = ?", key, expectedStatus).
		Updates(map[string]any{"status": status, "updated_at": at}))
}
