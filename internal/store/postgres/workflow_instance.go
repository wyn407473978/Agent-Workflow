package postgres

import (
	"context"
	"fmt"

	"Agent-Workflow/internal/store/postgres/model"

	"github.com/jackc/pgx/v5/pgtype"
	"gorm.io/gorm"
)

func (q *queries) GetWorkflow(ctx context.Context, id pgtype.UUID) (model.WorkflowInstance, error) {
	return findOne[model.WorkflowInstance](q.db.WithContext(ctx).Where("workflow_id = ?", id))
}

func (q *queries) ListSessionWorkflows(ctx context.Context, sessionID pgtype.UUID, limit int) ([]model.WorkflowInstance, error) {
	return findList[model.WorkflowInstance](q.db.WithContext(ctx).Where("session_id = ?", sessionID).
		Order("updated_at DESC NULLS LAST, workflow_id"), limit)
}

func (tx *Tx) CreateWorkflow(ctx context.Context, workflow model.WorkflowInstance) error {
	if err := tx.LockWorkflows(ctx, workflow.WorkflowID); err != nil {
		return err
	}
	db := tx.db.WithContext(ctx)
	return insertUnique(db, db.Where("workflow_id = ?", workflow.WorkflowID), &workflow)
}

// UpdateWorkflow 按预期修订号和状态写入完整的可变字段快照，并递增修订号。
func (tx *Tx) UpdateWorkflow(ctx context.Context, next model.WorkflowInstance, expectedRevision int64, expectedStatus string) error {
	if next.Status == nil || *next.Status == "" || next.UpdatedAt == nil || expectedRevision < 1 || expectedStatus == "" {
		return fmt.Errorf("%w: workflow status, revision and update time are required", ErrInvalidArgument)
	}
	if err := tx.LockWorkflows(ctx, next.WorkflowID); err != nil {
		return err
	}
	if _, err := tx.GetWorkflow(ctx, next.WorkflowID); err != nil {
		return err
	}
	return updateOne(tx.db.WithContext(ctx).Model(&model.WorkflowInstance{}).
		Where("workflow_id = ? AND state_revision = ? AND status = ?", next.WorkflowID, expectedRevision, expectedStatus).
		Updates(map[string]any{
			"status":            next.Status,
			"current_node":      next.CurrentNode,
			"intent_name":       next.IntentName,
			"draft_version":     next.DraftVersion,
			"approval_id":       next.ApprovalID,
			"activity_id":       next.ActivityID,
			"last_error_code":   next.LastErrorCode,
			"last_error_detail": next.LastErrorDetail,
			"updated_at":        next.UpdatedAt,
			"state_revision":    gorm.Expr("state_revision + 1"),
		}))
}
