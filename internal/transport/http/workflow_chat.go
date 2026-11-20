package http

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"html/template"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strings"

	"Agent-Workflow/internal/domain"
	"Agent-Workflow/internal/store/postgres"
	"Agent-Workflow/internal/workflow"

	"github.com/gin-gonic/gin"
)

//go:embed web/chat.html
var workflowChatHTML string

var workflowChatPage = template.Must(template.New("workflow-chat").Parse(workflowChatHTML))

type WorkflowChatStream func(context.Context, workflow.TestTurnInput, workflow.TextDeltaSink) (workflow.TestTurnResult, error)

// RegisterWorkflowChat mounts the local test page and its NDJSON streaming endpoint.
func RegisterWorkflowChat(router *gin.Engine, flow WorkflowChatStream, modelName string) {
	local := router.Group("", localWorkflowChat)
	local.GET("/chat", func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Header("Content-Security-Policy", "default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; connect-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
		if err := workflowChatPage.Execute(c.Writer, struct {
			Configured bool
			Model      string
		}{flow != nil, modelName}); err != nil {
			c.Error(err)
		}
	})
	local.POST("/api/workflows/test/chat", streamWorkflowChat(flow))
}

func localWorkflowChat(c *gin.Context) {
	remote, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil || !net.ParseIP(remote).IsLoopback() || !localHost(c.Request.Host) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	if origin := c.GetHeader("Origin"); origin != "" {
		u, err := url.Parse(origin)
		if err != nil || u.User != nil || !strings.EqualFold(c.Request.Host, u.Host) || (u.Scheme != "http" && u.Scheme != "https") {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
	}
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Next()
}

func localHost(host string) bool {
	if name, _, err := net.SplitHostPort(host); err == nil {
		host = name
	}
	return strings.EqualFold(host, "localhost") || net.ParseIP(strings.Trim(host, "[]")).IsLoopback()
}

type workflowChatRequest struct {
	SessionID domain.SessionID `json:"session_id" binding:"omitempty,uuid"`
	RequestID string           `json:"request_id" binding:"max=128"`
	Message   string           `json:"message" binding:"required,max=4000"`
}

type workflowChatEvent struct {
	Type   string                   `json:"type"`
	Text   string                   `json:"text,omitempty"`
	Result *workflow.TestTurnResult `json:"result,omitempty"`
	Error  *workflowChatError       `json:"error,omitempty"`
}

type workflowChatError struct {
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

func streamWorkflowChat(flow WorkflowChatStream) gin.HandlerFunc {
	return func(c *gin.Context) {
		if flow == nil {
			writeChatError(c, http.StatusServiceUnavailable, "模型未配置。请检查 DS_API_KEY 和 DS_BASE_URL 后重启服务。", true)
			return
		}
		mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
		if err != nil || mediaType != "application/json" {
			writeChatError(c, http.StatusUnsupportedMediaType, "请使用 JSON 发送消息。", false)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 32<<10)
		var input workflowChatRequest
		if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.Message) == "" {
			writeChatError(c, http.StatusBadRequest, "请输入 1 到 4000 个字符，并使用有效的会话 ID。", false)
			return
		}

		ctx := c.Request.Context()
		encoder := json.NewEncoder(c.Writer)
		reply := http.NewResponseController(c.Writer)
		write := func(event workflowChatEvent) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			c.Header("Content-Type", "application/x-ndjson; charset=utf-8")
			c.Header("X-Accel-Buffering", "no")
			if err := encoder.Encode(event); err != nil {
				return err
			}
			return reply.Flush()
		}
		result, err := flow(ctx, workflow.TestTurnInput(input), func(text string) error {
			return write(workflowChatEvent{Type: "delta", Text: text})
		})
		if ctx.Err() != nil || errors.Is(err, workflow.ErrChatDeltaDelivery) {
			return
		}
		if err != nil {
			status, chatErr := chatStreamError(err)
			if status == http.StatusInternalServerError {
				_ = c.Error(err)
			}
			event := workflowChatEvent{Type: "error", Result: &result, Error: &chatErr}
			if !c.Writer.Written() {
				c.JSON(status, event)
				return
			}
			_ = write(event)
			return
		}
		_ = write(workflowChatEvent{Type: "done", Result: &result})
	}
}

func writeChatError(c *gin.Context, status int, message string, retryable bool) {
	c.JSON(status, workflowChatEvent{Type: "error", Error: &workflowChatError{Message: message, Retryable: retryable}})
}

func chatStreamError(err error) (int, workflowChatError) {
	switch {
	case errors.Is(err, workflow.ErrTestSessionBusy):
		return http.StatusConflict, workflowChatError{Message: "会话还有一条消息在处理。稍后重试。", Retryable: true}
	case errors.Is(err, postgres.ErrConflict):
		return http.StatusConflict, workflowChatError{Message: "会话或请求 ID 与已保存的记录冲突。"}
	case errors.Is(err, postgres.ErrInvalidArgument):
		return http.StatusBadRequest, workflowChatError{Message: "会话 ID 或消息格式无效。"}
	case errors.Is(err, workflow.ErrTestModelFailed):
		return http.StatusBadGateway, workflowChatError{Message: "模型回复中断，可重试本轮。", Retryable: true}
	default:
		return http.StatusInternalServerError, workflowChatError{Message: "工作流暂时不可用。请检查数据库初始化和服务日志。", Retryable: true}
	}
}
