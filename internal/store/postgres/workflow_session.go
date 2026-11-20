package postgres

import (
	"context"
	"time"

	"Agent-Workflow/internal/store/postgres/model"

	"github.com/jackc/pgx/v5/pgtype"
)

func (q *queries) GetSession(ctx context.Context, id pgtype.UUID) (model.WorkflowSession, error) {
	return findOne[model.WorkflowSession](q.db.WithContext(ctx).Where("session_id = ?", id))
}

func (tx *Tx) CreateSession(ctx context.Context, session model.WorkflowSession) error {
	if err := tx.LockSessions(ctx, session.SessionID); err != nil {
		return err
	}
	db := tx.db.WithContext(ctx)
	return insertUnique(db, db.Where("session_id = ?", session.SessionID), &session)
}

// SetSessionWorkflow 将会话关联到已通过归属校验的工作流。传入空 UUID 时清除关联。
func (tx *Tx) SetSessionWorkflow(ctx context.Context, sessionID, workflowID pgtype.UUID, at time.Time) error {
	if err := tx.LockSessions(ctx, sessionID); err != nil {
		return err
	}
	if _, err := tx.GetSession(ctx, sessionID); err != nil {
		return err
	}
	return updateOne(tx.db.WithContext(ctx).Model(&model.WorkflowSession{}).
		Where("session_id = ?", sessionID).
		Updates(map[string]any{"active_workflow_id": workflowID, "updated_at": at}))
}
