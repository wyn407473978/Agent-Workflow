package postgres

import (
	"context"
	"fmt"

	"Agent-Workflow/internal/store/postgres/model"
	"github.com/jackc/pgx/v5/pgtype"
)

func (q *queries) ListWorkflowTransitions(ctx context.Context, workflowID pgtype.UUID, afterRevision int64, limit int) ([]model.WorkflowTransition, error) {
	return findList[model.WorkflowTransition](q.db.WithContext(ctx).
		Where("workflow_id = ? AND state_revision > ?", workflowID, afterRevision).
		Order("state_revision"), limit)
}

// AppendTransition 添加状态迁移审计。同一工作流的同一修订号只能记录一次。
func (tx *Tx) AppendTransition(ctx context.Context, transition model.WorkflowTransition) error {
	if transition.StateRevision == nil || *transition.StateRevision < 1 {
		return fmt.Errorf("%w: transition revision must be positive", ErrInvalidArgument)
	}
	if err := tx.LockWorkflows(ctx, transition.WorkflowID); err != nil {
		return err
	}
	db := tx.db.WithContext(ctx)
	return insertUnique(db, db.Where("workflow_id = ? AND state_revision = ?", transition.WorkflowID, *transition.StateRevision), &transition)
}
