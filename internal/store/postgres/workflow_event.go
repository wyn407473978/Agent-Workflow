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
	eventReceived   = "RECEIVED"
	eventProcessing = "PROCESSING"
	eventRetryWait  = "RETRY_WAIT"
)

// EventCursor 是上页最后一条事件的时间和标识。
type EventCursor struct {
	CreatedAt time.Time
	EventID   pgtype.UUID
}

func (q *queries) GetEvent(ctx context.Context, id pgtype.UUID) (model.WorkflowEvent, error) {
	return findOne[model.WorkflowEvent](q.db.WithContext(ctx).Where("event_id = ?", id))
}

func (q *queries) GetEventBySource(ctx context.Context, key EventKey) (model.WorkflowEvent, error) {
	return findOne[model.WorkflowEvent](eventSource(q.db.WithContext(ctx), key))
}

func (q *queries) ListSessionEvents(ctx context.Context, sessionID pgtype.UUID, after *EventCursor, limit int) ([]model.WorkflowEvent, error) {
	return listEvents(q.db.WithContext(ctx).Where("session_id = ?", sessionID), after, limit)
}

func (q *queries) ListWorkflowEvents(ctx context.Context, workflowID pgtype.UUID, after *EventCursor, limit int) ([]model.WorkflowEvent, error) {
	return listEvents(q.db.WithContext(ctx).Where("workflow_id = ?", workflowID), after, limit)
}

func listEvents(query *gorm.DB, after *EventCursor, limit int) ([]model.WorkflowEvent, error) {
	if after != nil {
		if after.CreatedAt.IsZero() || !after.EventID.Valid {
			return nil, fmt.Errorf("%w: incomplete event cursor", ErrInvalidArgument)
		}
		query = query.Where("(created_at, event_id) > (?, ?)", after.CreatedAt, after.EventID)
	}
	return findList[model.WorkflowEvent](query.Order("created_at, event_id"), limit)
}

// ReceiveEvent 写入首次投递。重复投递返回现有记录，布尔值表示本次是否新增。
func (tx *Tx) ReceiveEvent(ctx context.Context, event model.WorkflowEvent) (model.WorkflowEvent, bool, error) {
	if !event.EventID.Valid || event.Source == nil || event.SourceEventID == nil ||
		event.RequestHash == nil || *event.RequestHash == "" || event.CreatedAt == nil {
		return model.WorkflowEvent{}, false, fmt.Errorf("%w: incomplete incoming event", ErrInvalidArgument)
	}
	key := EventKey{Source: *event.Source, SourceEventID: *event.SourceEventID}
	if err := tx.LockEvents(ctx, key); err != nil {
		return model.WorkflowEvent{}, false, err
	}
	existing, err := tx.GetEventBySource(ctx, key)
	switch {
	case err == nil:
		if existing.RequestHash == nil || *existing.RequestHash != *event.RequestHash {
			return model.WorkflowEvent{}, false, fmt.Errorf("%w: event request hash differs", ErrConflict)
		}
		return existing, false, nil
	case !errors.Is(err, gorm.ErrRecordNotFound):
		return model.WorkflowEvent{}, false, err
	}
	if _, err := tx.GetEvent(ctx, event.EventID); !errors.Is(err, gorm.ErrRecordNotFound) {
		if err == nil {
			err = ErrAlreadyExists
		}
		return model.WorkflowEvent{}, false, err
	}
	event.Status = new(eventReceived)
	event.AttemptCount = new(int32(0))
	event.NextAttemptAt = event.CreatedAt
	event.UpdatedAt = event.CreatedAt
	event.LeaseToken = pgtype.UUID{}
	event.LeaseExpiresAt = nil
	event.ProcessedAt = nil
	event.ResultJSON = nil
	event.LastErrorCode = nil
	if err := tx.db.WithContext(ctx).Create(&event).Error; err != nil {
		return model.WorkflowEvent{}, false, err
	}
	return event, true, nil
}

func (q *queries) ListDueEvents(ctx context.Context, at time.Time, limit int) ([]model.WorkflowEvent, error) {
	return findList[model.WorkflowEvent](dueEvents(q.db.WithContext(ctx), at).
		Order("COALESCE(lease_expires_at, next_attempt_at), event_id"), limit)
}

// ClaimEvent 获取到期事件，或接管处理租约已过期的事件。
func (tx *Tx) ClaimEvent(ctx context.Context, key EventKey, at time.Time, lease Lease) (model.WorkflowEvent, error) {
	if err := lease.validate(at); err != nil {
		return model.WorkflowEvent{}, err
	}
	if err := tx.LockEvents(ctx, key); err != nil {
		return model.WorkflowEvent{}, err
	}
	if _, err := tx.GetEventBySource(ctx, key); err != nil {
		return model.WorkflowEvent{}, err
	}
	query := eventSource(tx.db.WithContext(ctx).Model(&model.WorkflowEvent{}), key)
	if err := updateOne(dueEvents(query, at).
		Where("(CASE WHEN status = ? THEN lease_expires_at ELSE next_attempt_at END) <= clock_timestamp()", eventProcessing).
		Where("? > clock_timestamp()", lease.ExpiresAt).
		Updates(map[string]any{
			"status":           eventProcessing,
			"lease_token":      lease.Token,
			"lease_expires_at": lease.ExpiresAt,
			"next_attempt_at":  nil,
			"attempt_count":    gorm.Expr("COALESCE(attempt_count, 0) + 1"),
			"updated_at":       gorm.Expr("clock_timestamp()"),
		})); err != nil {
		return model.WorkflowEvent{}, err
	}
	return tx.GetEventBySource(ctx, key)
}

type EventResult struct {
	WorkflowID    pgtype.UUID
	Status        string
	ResultJSON    json.RawMessage
	NextAttemptAt *time.Time
	LastErrorCode *string
}

// SaveEventResult 在租约有效时保存完整路由和处理结果。PROCESSING 表示检查点并保留租约。
func (tx *Tx) SaveEventResult(ctx context.Context, key EventKey, token pgtype.UUID, at time.Time, result EventResult) error {
	if !token.Valid || at.IsZero() {
		return fmt.Errorf("%w: event result requires a lease token and time", ErrInvalidArgument)
	}
	updates := map[string]any{
		"workflow_id":     result.WorkflowID,
		"status":          result.Status,
		"result_json":     result.ResultJSON,
		"next_attempt_at": nil,
		"last_error_code": result.LastErrorCode,
		"processed_at":    nil,
		"updated_at":      gorm.Expr("clock_timestamp()"),
	}
	switch result.Status {
	case eventProcessing:
	case eventRetryWait:
		if result.NextAttemptAt == nil {
			return fmt.Errorf("%w: retry requires a next attempt time", ErrInvalidArgument)
		}
		updates["next_attempt_at"] = result.NextAttemptAt
	case "APPLIED", "IGNORED", "FAILED":
		updates["processed_at"] = gorm.Expr("clock_timestamp()")
	default:
		return fmt.Errorf("%w: unsupported event result status %q", ErrInvalidArgument, result.Status)
	}
	if result.Status != eventProcessing {
		updates["lease_token"] = nil
		updates["lease_expires_at"] = nil
	}
	if err := tx.LockEvents(ctx, key); err != nil {
		return err
	}
	if _, err := tx.GetEventBySource(ctx, key); err != nil {
		return err
	}
	query := eventSource(tx.db.WithContext(ctx).Model(&model.WorkflowEvent{}), key).Where("status = ?", eventProcessing)
	return updateOne(withActiveLease(query, token, at).Updates(updates))
}

func eventSource(query *gorm.DB, key EventKey) *gorm.DB {
	return query.Where("source = ? AND source_event_id = ?", key.Source, key.SourceEventID)
}

func dueEvents(query *gorm.DB, at time.Time) *gorm.DB {
	return query.Where("((status IN ? AND next_attempt_at <= ?) OR (status = ? AND lease_expires_at <= ?))",
		[]string{eventReceived, eventRetryWait}, at, eventProcessing, at)
}
