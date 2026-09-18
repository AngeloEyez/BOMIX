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

// TestParseToolCallsFromContent 測試從各種模型文字輸出格式中解析 ToolCall
func TestParseToolCallsFromContent(t *testing.T) {
	// 案例 1：使用者真實遇到的 Qwen 無參數格式
	qwenRaw := `<tool_call>
<function=get_series_overview

</tool_call>`
	tcs1, clean1 := ParseToolCallsFromContent(qwenRaw)
	if len(tcs1) != 1 {
		t.Fatalf("案例 1 預期解析出 1 個 ToolCall，實際為 %d", len(tcs1))
	}
	if tcs1[0].Function.Name != "get_series_overview" {
		t.Errorf("案例 1 函數名稱預期為 get_series_overview，實際為: %s", tcs1[0].Function.Name)
	}
	if tcs1[0].Function.Arguments != "{}" {
		t.Errorf("案例 1 參數預期為 {}，實際為: %s", tcs1[0].Function.Arguments)
	}
	if clean1 != "" {
		t.Errorf("案例 1 清理後文字預期為空，實際為: %q", clean1)
	}

	// 案例 2：Qwen 帶有 parameter 標籤
	qwenParams := `<tool_call>
<function=execute_readonly_sql>
<parameter=sql>
SELECT * FROM parts;
</parameter>
<parameter=explanation>
查詢物料列表
</parameter>
</function>
</tool_call>`
	tcs2, _ := ParseToolCallsFromContent(qwenParams)
	if len(tcs2) != 1 {
		t.Fatalf("案例 2 預期解析出 1 個 ToolCall，實際為 %d", len(tcs2))
	}
	if tcs2[0].Function.Name != "execute_readonly_sql" {
		t.Errorf("案例 2 函數名稱預期為 execute_readonly_sql，實際為: %s", tcs2[0].Function.Name)
	}
	var args2 map[string]interface{}
	if err := json.Unmarshal([]byte(tcs2[0].Function.Arguments), &args2); err != nil {
		t.Fatalf("案例 2 參數 JSON 解析失敗: %v", err)
	}
	if !strings.Contains(args2["sql"].(string), "SELECT * FROM parts") {
		t.Errorf("案例 2 sql 參數不符合預期: %v", args2["sql"])
	}

	// 案例 3：Hermes JSON 格式
	hermesJSON := `<tool_call>
{"name": "compare_revisions_diff", "arguments": {"base_revision_id": 1, "target_revision_id": 2}}
</tool_call>`
	tcs3, _ := ParseToolCallsFromContent(hermesJSON)
	if len(tcs3) != 1 {
		t.Fatalf("案例 3 預期解析出 1 個 ToolCall，實際為 %d", len(tcs3))
	}
	if tcs3[0].Function.Name != "compare_revisions_diff" {
		t.Errorf("案例 3 函數名稱預期為 compare_revisions_diff，實際為: %s", tcs3[0].Function.Name)
	}

	// 案例 4：帶有前導文字與 Qwen 格式
	mixed := `我先為您查詢系列總覽。
<tool_call>
<function=get_series_overview>
</function>
</tool_call>`
	tcs4, clean4 := ParseToolCallsFromContent(mixed)
	if len(tcs4) != 1 {
		t.Fatalf("案例 4 預期解析出 1 個 ToolCall，實際為 %d", len(tcs4))
	}
	if clean4 != "我先為您查詢系列總覽。" {
		t.Errorf("案例 4 預期保留前導文字，實際為: %q", clean4)
	}
}


// TestAgentRun_QwenToolCallInContent 測試當本地模型在 Content 輸出 <tool_call> 標籤時，Agent 能正確解析並執行工具後完成回答
func TestAgentRun_QwenToolCallInContent(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		if callCount == 1 {
			// 第一輪：模型未回傳 tool_calls，而是在 content 輸出 Qwen 的 <tool_call> 標籤
			resp := ChatCompletionResponse{
				Choices: []Choice{
					{
						Message: ChatMessage{
							Role:    "assistant",
							Content: "<tool_call>\n<function=get_series_overview\n\n</tool_call>",
						},
					},
				},
				Usage: &Usage{
					PromptTokens:     100,
					CompletionTokens: 20,
					TotalTokens:      120,
				},
			}
			json.NewEncoder(w).Encode(resp)
		} else {
			// 第二輪：模型收到工具執行結果，輸出最終回答
			resp := ChatCompletionResponse{
				Choices: []Choice{
					{
						Message: ChatMessage{
							Role:    "assistant",
							Content: "系列中包含專案 A 與專案 B，各包含 Rev 1 與 Rev 2 版本。",
						},
					},
				},
				Usage: &Usage{
					PromptTokens:     150,
					CompletionTokens: 40,
					TotalTokens:      190,
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

	client := NewClient(server.URL, "mock-key", "qwen", 5)
	agent := NewAgent(client, nil, "You are a test assistant", 5, 0.0, 1000)
	emitter := &mockEmitter{}

	history := []ChatMessage{
		{Role: "user", Content: "請列出目前系列中包含哪些專案？"},
	}

	err := agent.Run(context.Background(), history, emitter, log, nil)
	if err != nil {
		t.Fatalf("agent.Run 執行失敗: %v", err)
	}

	// 驗證輪次
	if callCount != 2 {
		t.Errorf("預期呼叫模型 2 輪（工具調用 + 最終總結），實際呼叫了 %d 次", callCount)
	}

	// 驗證是否捕捉到了解析出的 ToolCall 日誌
	foundExtractedLog := false
	for _, msg := range capturedLogs {
		if strings.Contains(msg, "從 Content 中解析出 1 個 ToolCall 標籤") {
			foundExtractedLog = true
			break
		}
	}
	if !foundExtractedLog {
		t.Error("預期有記錄「從 Content 中解析出 1 個 ToolCall 標籤」日誌，但未找到")
	}

	// 驗證最終是否有收到 ai:done 事件與正確內容
	doneEvents := emitter.events["ai:done"]
	if len(doneEvents) == 0 {
		t.Fatal("未收到 ai:done 事件")
	}
	donePayload := doneEvents[0].(map[string]interface{})
	fullResp := donePayload["full_response"].(string)
	if !strings.Contains(fullResp, "系列中包含專案 A 與專案 B") {
		t.Errorf("最終回應內容不符預期: %s", fullResp)
	}
}
