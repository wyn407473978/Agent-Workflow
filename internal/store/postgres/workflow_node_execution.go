package postgres

import (
	"context"
	"fmt"

	"Agent-Workflow/internal/store/postgres/model"
	"github.com/jackc/pgx/v5/pgtype"
)

func (q *queries) GetNodeExecution(ctx context.Context, id pgtype.UUID) (model.WorkflowNodeExecution, error) {
	return findOne[model.WorkflowNodeExecution](q.db.WithContext(ctx).Where("execution_id = ?", id))
}

func (q *queries) ListWorkflowNodeExecutions(ctx context.Context, workflowID pgtype.UUID, limit int) ([]model.WorkflowNodeExecution, error) {
	return findList[model.WorkflowNodeExecution](q.db.WithContext(ctx).Where("workflow_id = ?", workflowID).
		Order("started_at DESC NULLS LAST, execution_id"), limit)
}

func (tx *Tx) CreateNodeExecution(ctx context.Context, execution model.WorkflowNodeExecution) error {
	if !execution.ExecutionID.Valid || execution.StartedAt == nil {
		return fmt.Errorf("%w: execution ID and start time are required", ErrInvalidArgument)
	}
	if err := tx.LockWorkflows(ctx, execution.WorkflowID); err != nil {
		return err
	}
	db := tx.db.WithContext(ctx)
	return insertUnique(db, db.Where("execution_id = ?", execution.ExecutionID), &execution)
}

// FinishNodeExecution 保存节点执行结果，已结束的记录保持原有结果。
func (tx *Tx) FinishNodeExecution(ctx context.Context, execution model.WorkflowNodeExecution) error {
	if !execution.ExecutionID.Valid || execution.FinishedAt == nil || execution.Status == nil || *execution.Status == "" {
		return fmt.Errorf("%w: execution ID, status and finish time are required", ErrInvalidArgument)
	}
	if err := tx.LockWorkflows(ctx, execution.WorkflowID); err != nil {
		return err
	}
	if _, err := tx.GetNodeExecution(ctx, execution.ExecutionID); err != nil {
		return err
	}
	return updateOne(tx.db.WithContext(ctx).Model(&model.WorkflowNodeExecution{}).
		Where("execution_id = ? AND workflow_id = ?", execution.ExecutionID, execution.WorkflowID).
		Where("finished_at IS NULL AND started_at <= ?", execution.FinishedAt).
		Updates(map[string]any{
			"status":      execution.Status,
			"trace_id":    execution.TraceID,
			"output_json": execution.OutputJSON,
			"error_code":  execution.ErrorCode,
			"finished_at": execution.FinishedAt,
		}))
}
