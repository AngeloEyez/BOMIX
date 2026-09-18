package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// EventEmitter 定義事件發送介面，用於與前端即時通訊
type EventEmitter interface {
	EmitEvent(event string, data interface{})
}

// Agent 封裝 Agentic Loop 控制流與 Tool Calling 迴圈
type Agent struct {
	client        *Client
	toolExecutor  *ToolExecutor
	systemPrompt  string
	maxIterations int
	temperature   float64
	maxTokens     int
}

// NewAgent 建立新的 Agent 實例
//
// 參數:
//   - client: OpenAI API 客戶端
//   - toolExecutor: 工具執行器
//   - systemPrompt: 系統提示詞（含語言約束）
//   - maxIterations: Agentic Loop 最大迭代次數（預設 8）
//   - temperature: 生成溫度
//   - maxTokens: 最大 Token 數
//
// 回傳:
//   - *Agent: Agent 實例
func NewAgent(client *Client, toolExecutor *ToolExecutor, systemPrompt string, maxIterations int, temperature float64, maxTokens int) *Agent {
	if maxIterations <= 0 {
		maxIterations = 8
	}
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	return &Agent{
		client:        client,
		toolExecutor:  toolExecutor,
		systemPrompt:  systemPrompt,
		maxIterations: maxIterations,
		temperature:   temperature,
		maxTokens:     maxTokens,
	}
}

// ToolCallEvent 前端展示用的工具呼叫事件資料
type ToolCallEvent struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Arguments   string `json:"arguments"`
	Explanation string `json:"explanation,omitempty"`
}

// ToolResultEvent 前端展示用的工具執行結果事件資料
type ToolResultEvent struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Result  string `json:"result"`
	Success bool   `json:"success"`
}

// Run 執行完整的 Agentic Loop 邏輯
//
// 執行流程:
// 1. 組裝 System Prompt 與前端傳入的對話歷史
// 2. 進入迴圈 (上限 maxIterations 次):
//    a. 檢查 context 是否被取消 (支援使用者隨時中斷)
//    b. 呼叫 LLM 進行推論
//    c. 若 LLM 請求調用工具 (tool_calls):
//       - 發送 ai:tool_call 事件給前端
//       - 依序透過 ToolExecutor 執行工具
//       - 發送 ai:tool_result 事件給前端
//       - 將 tool 回應塞入對話歷史，回到步驟 a
//    d. 若 LLM 回傳文字回覆 (finish_reason != tool_calls):
//       - 發送 ai:chunk 逐段推送文字
//       - 發送 ai:done 事件通知完成
//       - 結束迴圈
//
// 參數:
//   - ctx: 呼叫上下文（可傳入 context.WithCancel 支援中斷）
//   - history: 使用者與助手的對話歷史紀錄清單
//   - emitter: Wails 事件發送器
//
// 回傳:
//   - error: 執行失敗時回傳錯誤
func (a *Agent) Run(ctx context.Context, history []ChatMessage, emitter EventEmitter) error {
	if a.client == nil {
		return errors.New("AI 客戶端未初始化")
	}

	// 1. 組裝系統訊息與對話歷史
	conversation := make([]ChatMessage, 0, len(history)+2)
	conversation = append(conversation, ChatMessage{
		Role:    "system",
		Content: a.systemPrompt,
	})
	conversation = append(conversation, history...)

	tools := GetBuiltinTools()

	if emitter != nil {
		emitter.EmitEvent("ai:status", map[string]interface{}{
			"status":  "thinking",
			"message": "AI 正在分析您的提問...",
		})
	}

	var roundPromptTokens, roundCompletionTokens, roundTotalTokens int

	for iter := 0; iter < a.maxIterations; iter++ {
		select {
		case <-ctx.Done():
			if emitter != nil {
				emitter.EmitEvent("ai:error", map[string]interface{}{
					"error": "使用者已中斷生成",
				})
			}
			return ctx.Err()
		default:
		}

		req := ChatCompletionRequest{
			Messages:    conversation,
			Tools:       tools,
			Temperature: a.temperature,
			MaxTokens:   a.maxTokens,
		}

		resp, err := a.client.SendChat(ctx, req)
		if err != nil {
			if emitter != nil {
				emitter.EmitEvent("ai:error", map[string]interface{}{
					"error": fmt.Sprintf("AI 呼叫失敗: %v", err),
				})
			}
			return fmt.Errorf("AI 呼叫失敗: %w", err)
		}

		if resp.Usage != nil {
			roundPromptTokens += resp.Usage.PromptTokens
			roundCompletionTokens += resp.Usage.CompletionTokens
			roundTotalTokens += resp.Usage.TotalTokens
		}

		if len(resp.Choices) == 0 {
			err = errors.New("模型未回傳任何候選回應")
			if emitter != nil {
				emitter.EmitEvent("ai:error", map[string]interface{}{"error": err.Error()})
			}
			return err
		}

		choice := resp.Choices[0]
		msg := choice.Message

		// 情況 A: 模型請求調用工具
		if len(msg.ToolCalls) > 0 {
			// 將 Assistant 的工具調用訊息加入歷史
			conversation = append(conversation, msg)

			for _, tc := range msg.ToolCalls {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				// 嘗試從 arguments 解析 explanation
				var explanation string
				var argMap map[string]interface{}
				if json.Unmarshal([]byte(tc.Function.Arguments), &argMap) == nil {
					if exp, ok := argMap["explanation"].(string); ok {
						explanation = exp
					}
				}

				if emitter != nil {
					emitter.EmitEvent("ai:tool_call", ToolCallEvent{
						ID:          tc.ID,
						Name:        tc.Function.Name,
						Arguments:   tc.Function.Arguments,
						Explanation: explanation,
					})
					emitter.EmitEvent("ai:status", map[string]interface{}{
						"status":  "calling_tool",
						"message": fmt.Sprintf("正在執行工具: %s", tc.Function.Name),
					})
				}

				// 執行工具
				var resultStr string
				var execErr error
				if a.toolExecutor != nil {
					resultStr, execErr = a.toolExecutor.Execute(ctx, tc.Function.Name, tc.Function.Arguments)
				} else {
					execErr = errors.New("工具執行器未就緒")
				}

				success := true
				if execErr != nil {
					success = false
					resultStr = fmt.Sprintf("Tool execution error: %v", execErr)
				}

				if emitter != nil {
					emitter.EmitEvent("ai:tool_result", ToolResultEvent{
						ID:      tc.ID,
						Name:    tc.Function.Name,
						Result:  resultStr,
						Success: success,
					})
				}

				// 將工具執行結果加入對話歷史
				conversation = append(conversation, ChatMessage{
					Role:       "tool",
					Content:    resultStr,
					ToolCallID: tc.ID,
					Name:       tc.Function.Name,
				})
			}

			// 繼續下一輪迴圈
			continue
		}

		// 情況 B: 模型給出最終文字回覆 (無更多工具調用)
		content := msg.Content

		if emitter != nil {
			emitter.EmitEvent("ai:status", map[string]interface{}{
				"status":  "streaming",
				"message": "AI 正在回覆...",
			})

			// 平滑分段推送文字
			runes := []rune(content)
			chunkSize := 8
			for i := 0; i < len(runes); i += chunkSize {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}
				end := i + chunkSize
				if end > len(runes) {
					end = len(runes)
				}
				emitter.EmitEvent("ai:chunk", map[string]interface{}{
					"chunk": string(runes[i:end]),
				})
				time.Sleep(12 * time.Millisecond)
			}

			emitter.EmitEvent("ai:done", map[string]interface{}{
				"full_response": content,
				"usage": map[string]interface{}{
					"prompt_tokens":     roundPromptTokens,
					"completion_tokens": roundCompletionTokens,
					"total_tokens":      roundTotalTokens,
				},
			})
			emitter.EmitEvent("ai:status", map[string]interface{}{
				"status":  "idle",
				"message": "完成",
			})
		}

		return nil
	}

	return fmt.Errorf("達到 Agentic Loop 最大迭代次數 (%d)，已中止執行", a.maxIterations)
}
