package workflow

import (
	"time"

	"Agent-Workflow/internal/domain"
)

// RoundInput 是处理一轮消息、回调或恢复任务的入口。
type RoundInput struct {
	Event domain.WorkflowEvent `json:"event"`
}

// RoundContext 保存经归属校验后加载的会话和工作流。
type RoundContext struct {
	Event    domain.WorkflowEvent  `json:"event"`
	Session  domain.Session        `json:"session"`
	Workflow *domain.WorkflowState `json:"workflow,omitempty"` // 通用问答或新任务尚无实例。
}

// Checkpoint 保存本轮已完成的步骤和对应的持久化版本。
type Checkpoint struct {
	StateRevision  int64             `json:"state_revision"`
	DraftVersion   int64             `json:"draft_version"`
	CompletedNodes []domain.NodeName `json:"completed_nodes"`
}

// RoundResult 是用于持久化处理结果或安排后续重试的模型。
type RoundResult struct {
	WorkflowID    domain.WorkflowID     `json:"workflow_id,omitempty"`
	EventStatus   domain.EventStatus    `json:"event_status"`
	Checkpoint    *Checkpoint           `json:"checkpoint,omitempty"`
	Reply         *UserReply            `json:"reply,omitempty"`
	Error         *domain.WorkflowError `json:"error,omitempty"`
	NextAttemptAt *time.Time            `json:"next_attempt_at,omitempty"`
}

// UserReply 是用户本轮看到的文本和可用于后续确认的业务引用。
type UserReply struct {
	Text       string                `json:"text"`
	Status     domain.WorkflowStatus `json:"status,omitempty"`
	Preview    *domain.PreviewRef    `json:"preview,omitempty"`
	ActivityID domain.ActivityID     `json:"activity_id,omitempty"`
}
