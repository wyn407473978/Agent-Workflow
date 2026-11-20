package postgres

import (
	"context"
	"fmt"

	"Agent-Workflow/internal/store/postgres/model"
	"github.com/jackc/pgx/v5/pgtype"
)

func (q *queries) GetApproval(ctx context.Context, id pgtype.UUID) (model.ApprovalRequest, error) {
	return findOne[model.ApprovalRequest](q.db.WithContext(ctx).Where("approval_id = ?", id))
}

func (q *queries) ListDraftApprovals(ctx context.Context, workflowID pgtype.UUID, version int64, limit int) ([]model.ApprovalRequest, error) {
	return findList[model.ApprovalRequest](q.db.WithContext(ctx).
		Where("workflow_id = ? AND draft_version = ?", workflowID, version).
		Order("created_at DESC NULLS LAST, approval_id"), limit)
}

func (q *queries) ListPendingApprovals(ctx context.Context, limit int) ([]model.ApprovalRequest, error) {
	return findList[model.ApprovalRequest](q.db.WithContext(ctx).Where("status = ?", "PENDING").
		Order("created_at, approval_id"), limit)
}

func (tx *Tx) CreateApproval(ctx context.Context, approval model.ApprovalRequest) error {
	if !approval.ApprovalID.Valid || approval.DraftVersion == nil || *approval.DraftVersion < 1 ||
		!approval.PreviewID.Valid || !approval.ConfirmationEventID.Valid {
		return fmt.Errorf("%w: approval requires a draft version, preview and confirmation", ErrInvalidArgument)
	}
	if err := tx.LockWorkflows(ctx, approval.WorkflowID); err != nil {
		return err
	}
	db := tx.db.WithContext(ctx)
	return insertUnique(db, db.Where("approval_id = ?", approval.ApprovalID), &approval)
}

// UpdateApproval 按预期状态、草稿、预览和确认事件写入审批决定或失效信息。
func (tx *Tx) UpdateApproval(ctx context.Context, next model.ApprovalRequest, expectedStatus string) error {
	if !next.ApprovalID.Valid || next.DraftVersion == nil || *next.DraftVersion < 1 ||
		!next.PreviewID.Valid || !next.ConfirmationEventID.Valid || next.Status == nil || *next.Status == "" ||
		next.UpdatedAt == nil || expectedStatus == "" {
		return fmt.Errorf("%w: incomplete approval update", ErrInvalidArgument)
	}
	if err := tx.LockWorkflows(ctx, next.WorkflowID); err != nil {
		return err
	}
	if _, err := tx.GetApproval(ctx, next.ApprovalID); err != nil {
		return err
	}
	return updateOne(tx.db.WithContext(ctx).Model(&model.ApprovalRequest{}).
		Where("approval_id = ? AND workflow_id = ? AND draft_version = ? AND status = ?",
			next.ApprovalID, next.WorkflowID, *next.DraftVersion, expectedStatus).
		Where("preview_id = ? AND confirmation_event_id = ?", next.PreviewID, next.ConfirmationEventID).
		Updates(map[string]any{
			"status":            next.Status,
			"decision_event_id": next.DecisionEventID,
			"decided_by":        next.DecidedBy,
			"decided_at":        next.DecidedAt,
			"reason":            next.Reason,
			"invalidated_at":    next.InvalidatedAt,
			"updated_at":        next.UpdatedAt,
		}))
}
