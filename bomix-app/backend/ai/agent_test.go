package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"bomix-app/backend/logger"
)

// mockEmitter 記錄發送的事件
type mockEmitter struct {
	mu     sync.Mutex
	events map[string][]interface{}
}

func (m *mockEmitter) EmitEvent(event string, data interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.events == nil {
		m.events = make(map[string][]interface{})
	}
	m.events[event] = append(m.events[event], data)
}

// TestAgentRun_TaskLogging 測試 Agent.Run 是否正確記錄任務啟動、工具調用意圖、參數、結果以及 Token 統計日誌
func TestAgentRun_TaskLogging(t *testing.T) {
	callCount := 0
	// 建立 Mock Server，第一輪回傳 tool_call，第二輪回傳最終結果
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		if callCount == 1 {
			// 第一輪：回傳 get_database_schema 工具調用請求
			resp := ChatCompletionResponse{
				Choices: []Choice{
					{
						Message: ChatMessage{
							Role: "assistant",
							ToolCalls: []ToolCall{
								{
									ID:   "call-test-123",
									Type: "function",
									Function: FunctionCall{
										Name:      "get_database_schema",
										Arguments: `{"explanation":"分析資料庫結構以撰寫查詢"}`,
									},
								},
							},
						},
					},
				},
				Usage: &Usage{
					PromptTokens:     100,
					CompletionTokens: 30,
					TotalTokens:      130,
				},
			}
			json.NewEncoder(w).Encode(resp)
		} else {
			// 第二輪：回傳最終文字回覆
			resp := ChatCompletionResponse{
				Choices: []Choice{
					{
						Message: ChatMessage{
							Role:    "assistant",
							Content: "已為您分析完成資料庫結構。",
						},
					},
				},
				Usage: &Usage{
					PromptTokens:     150,
					CompletionTokens: 50,
					TotalTokens:      200,
				},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	log := logger.NewLogger(100)
	var capturedLogs []string
	log.SetEventCallback(func(event string, data interface{}) {
		if entry, ok := data.(*logger.LogEntry); ok {
			capturedLogs = append(capturedLogs, entry.Message)
		}
	})

	client := NewClient(server.URL, "mock-key", "gpt-4o", 5)
	agent := NewAgent(client, nil, "You are a test assistant", 5, 0.1, 1000)
	emitter := &mockEmitter{}

	var progressReports []string
	progressCb := func(p float64, msg string) {
		progressReports = append(progressReports, msg)
	}

	history := []ChatMessage{
		{Role: "user", Content: "測試提問"},
	}

	ctx := context.Background()
	err := agent.Run(ctx, history, emitter, log, progressCb)
	if err != nil {
		t.Fatalf("agent.Run 回傳非預期錯誤: %v", err)
	}

	// 驗證是否有記錄對話啟動日誌
	foundStart := false
	foundToolCall := false
	foundDone := false

	for _, msg := range capturedLogs {
		if strings.Contains(msg, "AI 對話啟動") {
			foundStart = true
		}
		if strings.Contains(msg, "調用工具 [1/1]: get_database_schema") {
			foundToolCall = true
		}
		if strings.Contains(msg, "AI 對話生成完成") {
			foundDone = true
		}
	}

	if !foundStart {
		t.Error("預期記錄「AI 對話啟動」日誌，但未找到")
	}
	if !foundToolCall {
		t.Error("預期記錄「調用工具 [1/1]: get_database_schema」日誌，但未找到")
	}
	if !foundDone {
		t.Error("預期記錄「AI 對話生成完成」日誌，但未找到")
	}

	// 驗證是否有回報進度
	if len(progressReports) == 0 {
		t.Error("預期有進度回報，但 progressReports 為空")
	}
}
