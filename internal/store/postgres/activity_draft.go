package postgres

import (
	"context"
	"fmt"
	"time"

	"Agent-Workflow/internal/store/postgres/model"

	"github.com/jackc/pgx/v5/pgtype"
)

func (q *queries) GetDraft(ctx context.Context, workflowID pgtype.UUID, version int64) (model.ActivityDraft, error) {
	return findOne[model.ActivityDraft](q.db.WithContext(ctx).
		Where("workflow_id = ? AND draft_version = ?", workflowID, version))
}

func (tx *Tx) CreateDraft(ctx context.Context, draft model.ActivityDraft) error {
	if draft.DraftVersion == nil || *draft.DraftVersion < 1 {
		return fmt.Errorf("%w: draft version must be positive", ErrInvalidArgument)
	}
	if err := tx.LockWorkflows(ctx, draft.WorkflowID); err != nil {
		return err
	}
	db := tx.db.WithContext(ctx)
	return insertUnique(db, db.Where("workflow_id = ? AND draft_version = ?", draft.WorkflowID, *draft.DraftVersion), &draft)
}

// UpdateDraft 写入未发布版本的参数与查询证据。设置预览标识后，该版本的参数被冻结。
func (tx *Tx) UpdateDraft(ctx context.Context, draft model.ActivityDraft) error {
	if draft.DraftVersion == nil || *draft.DraftVersion < 1 || draft.UpdatedAt == nil {
		return fmt.Errorf("%w: draft version and update time are required", ErrInvalidArgument)
	}
	if draft.PreviewID.Valid && (draft.PreviewedAt == nil || draft.PreviewExpiresAt == nil || !draft.PreviewExpiresAt.After(*draft.PreviewedAt)) {
		return fmt.Errorf("%w: preview expiry must follow release time", ErrInvalidArgument)
	}
	if err := tx.LockWorkflows(ctx, draft.WorkflowID); err != nil {
		return err
	}
	if _, err := tx.GetDraft(ctx, draft.WorkflowID, *draft.DraftVersion); err != nil {
		return err
	}
	return updateOne(tx.db.WithContext(ctx).Model(&model.ActivityDraft{}).
		Where("workflow_id = ? AND draft_version = ? AND preview_id IS NULL", draft.WorkflowID, *draft.DraftVersion).
		Updates(map[string]any{
			"activity_name":        draft.ActivityName,
			"audience_rule":        draft.AudienceRule,
			"scheduled_at":         draft.ScheduledAt,
			"timezone":             draft.TimeZone,
			"template_id":          draft.TemplateID,
			"extra_slots":          draft.ExtraSlots,
			"missing_slots":        draft.MissingSlots,
			"audience_snapshot_id": draft.AudienceSnapshotID,
			"audience_count":       draft.AudienceCount,
			"template_status":      draft.TemplateStatus,
			"estimated_cost_minor": draft.EstimatedCostMinor,
			"currency":             draft.Currency,
			"snapshot_json":        draft.SnapshotJSON,
			"rule_version":         draft.RuleVersion,
			"validation_json":      draft.ValidationJSON,
			"preview_id":           draft.PreviewID,
			"previewed_at":         draft.PreviewedAt,
			"preview_expires_at":   draft.PreviewExpiresAt,
			"updated_at":           draft.UpdatedAt,
		}))
}

type DraftConfirmation struct {
	PreviewID   pgtype.UUID
	EventID     pgtype.UUID
	ConfirmedBy string
	ConfirmedAt time.Time
	ExpiresAt   time.Time
}

// ConfirmDraft 记录当前有效预览的首次确认。调用者在同一事务校验工作流状态和用户归属。
func (tx *Tx) ConfirmDraft(ctx context.Context, workflowID pgtype.UUID, version int64, confirmation DraftConfirmation) error {
	if version < 1 || !confirmation.PreviewID.Valid || !confirmation.EventID.Valid || confirmation.ConfirmedBy == "" ||
		confirmation.ConfirmedAt.IsZero() || !confirmation.ExpiresAt.After(confirmation.ConfirmedAt) {
		return fmt.Errorf("%w: incomplete draft confirmation", ErrInvalidArgument)
	}
	if err := tx.LockWorkflows(ctx, workflowID); err != nil {
		return err
	}
	if _, err := tx.GetDraft(ctx, workflowID, version); err != nil {
		return err
	}
	return updateOne(tx.db.WithContext(ctx).Model(&model.ActivityDraft{}).
		Where("workflow_id = ? AND draft_version = ? AND preview_id = ?", workflowID, version, confirmation.PreviewID).
		Where("preview_expires_at > GREATEST(?, clock_timestamp()) AND confirmation_event_id IS NULL", confirmation.ConfirmedAt).
		Updates(map[string]any{
			"confirmation_version":    version,
			"confirmation_event_id":   confirmation.EventID,
			"confirmed_by":            confirmation.ConfirmedBy,
			"confirmed_at":            confirmation.ConfirmedAt,
			"confirmation_expires_at": confirmation.ExpiresAt,
			"updated_at":              confirmation.ConfirmedAt,
		}))
}
