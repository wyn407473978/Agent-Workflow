package postgres

import (
	"context"
	"slices"

	"Agent-Workflow/internal/domain"
	"Agent-Workflow/internal/store/postgres/model"
	"github.com/jackc/pgx/v5/pgtype"
)

// ListRecentConversationTurns 返回最近成功处理的用户消息及其结果，按完成顺序回放。
func (q *queries) ListRecentConversationTurns(ctx context.Context, sessionID, workflowID pgtype.UUID, limit int) ([]model.WorkflowEvent, error) {
	query := q.db.WithContext(ctx).
		Where("session_id = ? AND workflow_id = ? AND event_type = ? AND status = ?", sessionID, workflowID, domain.EventUserMessage, domain.EventApplied).
		Where("processed_at IS NOT NULL AND result_json IS NOT NULL").
		Order("processed_at DESC, created_at DESC, event_id DESC")
	rows, err := findList[model.WorkflowEvent](query, limit)
	slices.Reverse(rows)
	return rows, err
}

// HasOtherActiveSessionTurn 用于持有会话锁时检查是否已有其他轮次正在处理。
func (tx *Tx) HasOtherActiveSessionTurn(ctx context.Context, sessionID, workflowID, eventID pgtype.UUID) (bool, error) {
	var rows []struct{ EventID pgtype.UUID }
	err := tx.db.WithContext(ctx).Model(&model.WorkflowEvent{}).Select("event_id").
		Where("session_id = ? AND workflow_id = ? AND event_id <> ?", sessionID, workflowID, eventID).
		Where("status = ? AND lease_expires_at > clock_timestamp()", domain.EventProcessing).
		Limit(1).Find(&rows).Error
	return len(rows) > 0, err
}
