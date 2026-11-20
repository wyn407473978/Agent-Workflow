package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"Agent-Workflow/internal/domain"
	"Agent-Workflow/internal/workflow"

	"github.com/gin-gonic/gin"
)

func TestWorkflowChatFlushesDeltasBeforeCompletion(t *testing.T) {
	router := gin.New()
	response := httptest.NewRecorder()
	RegisterWorkflowChat(router, func(_ context.Context, input workflow.TestTurnInput, emit workflow.TextDeltaSink) (workflow.TestTurnResult, error) {
		if input.RequestID != "req-1" || input.Message != "你好" {
			t.Fatalf("incorrect input: %+v", input)
		}
		if err := emit("你"); err != nil {
			return workflow.TestTurnResult{}, err
		}
		if !response.Flushed || !strings.Contains(response.Body.String(), `"text":"你"`) {
			t.Fatal("first delta was not flushed")
		}
		if err := emit("好"); err != nil {
			return workflow.TestTurnResult{}, err
		}
		return workflow.TestTurnResult{RoundResult: workflow.RoundResult{
			EventStatus: domain.EventApplied, Reply: &workflow.UserReply{Text: "你好"},
		}}, nil
	}, "test-model")
	request := httptest.NewRequest(http.MethodPost, "http://localhost:8080/api/workflows/test/chat", strings.NewReader(`{"request_id":"req-1","message":"你好"}`))
	request.RemoteAddr = "127.0.0.1:3456"
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://localhost:8080")
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.HasPrefix(response.Header().Get("Content-Type"), "application/x-ndjson") {
		t.Fatalf("unexpected response: %d %s", response.Code, response.Body.String())
	}
	decoder := json.NewDecoder(response.Body)
	for _, expected := range []string{"delta", "delta", "done"} {
		var event workflowChatEvent
		if err := decoder.Decode(&event); err != nil || event.Type != expected {
			t.Fatalf("expected %q, got %+v, %v", expected, event, err)
		}
	}
}

func TestWorkflowChatRequiresLocalOrigin(t *testing.T) {
	router := gin.New()
	RegisterWorkflowChat(router, nil, "test-model")
	for _, input := range []struct {
		url, address, origin string
		status               int
	}{
		{"http://localhost:8080/chat", "127.0.0.1:3456", "", http.StatusOK},
		{"http://localhost:8080/chat", "192.0.2.10:3456", "", http.StatusForbidden},
		{"http://foreign.example/chat", "127.0.0.1:3456", "", http.StatusForbidden},
		{"http://localhost:8080/chat", "127.0.0.1:3456", "http://foreign.example", http.StatusForbidden},
	} {
		request := httptest.NewRequest(http.MethodGet, input.url, nil)
		request.RemoteAddr = input.address
		request.Header.Set("Origin", input.origin)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != input.status {
			t.Fatalf("%+v: got %d", input, response.Code)
		}
		if response.Code == http.StatusOK && !strings.Contains(response.Body.String(), `data-configured="false"`) {
			t.Fatal("the embedded page did not report missing model configuration")
		}
	}
}
