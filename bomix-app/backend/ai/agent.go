package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"bomix-app/backend/logger"
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
//  1. 組裝 System Prompt 與前端傳入的對話歷史
//  2. 進入迴圈 (上限 maxIterations 次):
//     a. 檢查 context 是否被取消 (支援使用者隨時中斷)
//     b. 呼叫 LLM 進行推論
//     c. 若 LLM 請求調用工具 (tool_calls):
//     - 發送 ai:tool_call 事件給前端
//     - 依序透過 ToolExecutor 執行工具並記錄工具狀態日誌
//     - 發送 ai:tool_result 事件給前端
//     - 將 tool 回應塞入對話歷史，回到步驟 a
//     d. 若 LLM 回傳文字回覆 (finish_reason != tool_calls):
//     - 發送 ai:chunk 逐段推送文字
//     - 發送 ai:done 事件通知完成
//     - 結束迴圈
//
// 參數:
//   - ctx: 呼叫上下文（可傳入 context.WithCancel 支援中斷）
//   - history: 使用者與助手的對話歷史紀錄清單
//   - emitter: Wails 事件發送器
//   - taskLogger: 任務結構化日誌記錄器（可為 nil）
//   - progress: 任務進度回報回呼（可為 nil）
//
// 回傳:
//   - error: 執行失敗時回傳錯誤
func (a *Agent) Run(
	ctx context.Context,
	history []ChatMessage,
	emitter EventEmitter,
	taskLogger *logger.Logger,
	progress func(float64, string),
) error {
	if a.client == nil {
		return errors.New("AI 客戶端未初始化")
	}

	if taskLogger != nil {
		taskLogger.Info(fmt.Sprintf("AI 對話啟動 | 模型: %s, 歷史訊息數: %d, 最大迭代上限: %d 輪",
			a.client.model, len(history), a.maxIterations))
		taskLogger.Debug(fmt.Sprintf("AI 配置參數 | BaseURL: %s, Temperature: %.2f, MaxTokens: %d",
			a.client.baseURL, a.temperature, a.maxTokens))
	}
	if progress != nil {
		progress(0.05, "AI 正在分析您的提問...")
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
	if taskLogger != nil {
		taskLogger.Info("AI 正在分析您的提問...")
	}

	var roundPromptTokens, roundCompletionTokens, roundTotalTokens int

	for iter := 0; iter < a.maxIterations; iter++ {
		select {
		case <-ctx.Done():
			if taskLogger != nil {
				taskLogger.Info("使用者已中斷 AI 對話生成")
			}
			if emitter != nil {
				emitter.EmitEvent("ai:error", map[string]interface{}{
					"error": "使用者已中斷生成",
				})
			}
			return ctx.Err()
		default:
		}

		// 若為第 2 輪以上之迭代，發送多輪分析進度狀態
		if iter > 0 {
			roundMsg := fmt.Sprintf("AI 正在分析工具回傳資料 (第 %d/%d 輪)...", iter+1, a.maxIterations)
			if emitter != nil {
				emitter.EmitEvent("ai:status", map[string]interface{}{
					"status":  "thinking",
					"message": roundMsg,
				})
			}
			if taskLogger != nil {
				taskLogger.Info(roundMsg)
			}
		}

		currentProgress := 0.1 + float64(iter)*0.1
		if currentProgress > 0.85 {
			currentProgress = 0.85
		}
		if progress != nil {
			progress(currentProgress, fmt.Sprintf("AI 正在思考與推論 (輪次 %d/%d)...", iter+1, a.maxIterations))
		}
		if taskLogger != nil {
			taskLogger.Debug(fmt.Sprintf("[輪次 %d/%d] 正在向 AI 模型發送推論請求...", iter+1, a.maxIterations))
		}

		var toolChoice string
		if len(tools) > 0 {
			toolChoice = "auto"
		}
		req := ChatCompletionRequest{
			Messages:    conversation,
			Tools:       tools,
			ToolChoice:  toolChoice,
			Temperature: a.temperature,
			MaxTokens:   a.maxTokens,
		}

		resp, err := a.client.SendChat(ctx, req)
		if err != nil {
			errMsg := fmt.Sprintf("AI 模型 API 呼叫失敗: %v", err)
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
				errMsg = fmt.Sprintf("AI 模型 API 呼叫逾時 (已超過限制時間): %v", err)
			}
			if taskLogger != nil {
				taskLogger.Error(errMsg, "round", iter+1)
			}
			if emitter != nil {
				emitter.EmitEvent("ai:error", map[string]interface{}{
					"error": errMsg,
				})
			}
			return fmt.Errorf("AI 呼叫失敗: %w", err)
		}

		if resp.Usage != nil {
			roundPromptTokens += resp.Usage.PromptTokens
			roundCompletionTokens += resp.Usage.CompletionTokens
			roundTotalTokens += resp.Usage.TotalTokens
			if taskLogger != nil {
				taskLogger.Debug(fmt.Sprintf("[輪次 %d/%d] 本次耗用 Tokens: Prompt=%d, Completion=%d, Total=%d",
					iter+1, a.maxIterations, resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens))
			}
		}

		if len(resp.Choices) == 0 {
			err = errors.New("模型未回傳任何候選回應 (Choices 為空)")
			if taskLogger != nil {
				taskLogger.Warn(err.Error(), "round", iter+1)
			}
			if emitter != nil {
				emitter.EmitEvent("ai:error", map[string]interface{}{"error": err.Error()})
			}
			return err
		}

		choice := resp.Choices[0]
		msg := choice.Message

		// 若模型未回傳結構化 ToolCalls，但 Content 中包含 tool_call 標籤（常見於 Qwen、DeepSeek 等本地開源模型），進行自動解析提取
		if len(msg.ToolCalls) == 0 {
			if parsedTools, cleaned := ParseToolCallsFromContent(msg.Content); len(parsedTools) > 0 {
				if taskLogger != nil {
					taskLogger.Info(fmt.Sprintf("[輪次 %d/%d] 從 Content 中解析出 %d 個 ToolCall 標籤", iter+1, a.maxIterations, len(parsedTools)))
				}
				msg.ToolCalls = parsedTools
				msg.Content = cleaned
			}
		}

		// 情況 A: 模型請求調用工具
		if len(msg.ToolCalls) > 0 {
			conversation = append(conversation, msg)
			if taskLogger != nil {
				taskLogger.Info(fmt.Sprintf("[輪次 %d/%d] AI 請求調用 %d 個工具", iter+1, a.maxIterations, len(msg.ToolCalls)))
			}

			for tcIdx, tc := range msg.ToolCalls {
				select {
				case <-ctx.Done():
					if taskLogger != nil {
						taskLogger.Info("使用者已中斷 AI 對話生成 (工具調用中)")
					}
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

				if taskLogger != nil {
					argPreview := tc.Function.Arguments
					if len(argPreview) > 300 {
						argPreview = argPreview[:300] + "... (已截斷)"
					}
					taskLogger.Info(fmt.Sprintf("調用工具 [%d/%d]: %s", tcIdx+1, len(msg.ToolCalls), tc.Function.Name),
						"explanation", explanation)
					taskLogger.Debug(fmt.Sprintf("工具引數 [%s]: %s", tc.Function.Name, argPreview))
				}

				if progress != nil {
					progress(currentProgress+0.05, fmt.Sprintf("正在執行工具: %s...", tc.Function.Name))
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
				if taskLogger != nil {
					taskLogger.Info(fmt.Sprintf("正在執行工具: %s", tc.Function.Name))
				}

				// 執行工具並量測耗時
				var resultStr string
				var execErr error
				toolStartTime := time.Now()
				if a.toolExecutor != nil {
					resultStr, execErr = a.toolExecutor.Execute(ctx, tc.Function.Name, tc.Function.Arguments)
				} else {
					execErr = errors.New("工具執行器未就緒")
				}
				toolDuration := time.Since(toolStartTime)

				success := true
				if execErr != nil {
					success = false
					resultStr = fmt.Sprintf("Tool execution error: %v", execErr)
					if taskLogger != nil {
						taskLogger.Error(fmt.Sprintf("工具 [%s] 執行失敗 (耗時 %v): %v", tc.Function.Name, toolDuration, execErr))
					}
				} else {
					if taskLogger != nil {
						resPreview := resultStr
						if len(resPreview) > 250 {
							resPreview = resPreview[:250] + "... (已截斷)"
						}
						taskLogger.Info(fmt.Sprintf("工具 [%s] 執行成功 (耗時 %v, 回傳長度 %d 字元)", tc.Function.Name, toolDuration, len(resultStr)))
						taskLogger.Debug(fmt.Sprintf("工具 [%s] 回傳內容: %s", tc.Function.Name, resPreview))
					}
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

		if taskLogger != nil {
			taskLogger.Info(fmt.Sprintf("AI 推論完成，開始輸出文字回覆 (共 %d 字元)", len(content)))
		}
		if progress != nil {
			progress(0.9, "AI 正在回覆...")
		}

		if emitter != nil {
			emitter.EmitEvent("ai:status", map[string]interface{}{
				"status":  "streaming",
				"message": "AI 正在回覆...",
			})
		}
		if taskLogger != nil {
			taskLogger.Info("AI 正在回覆...")
		}

		if emitter != nil {
			// 平滑分段推送文字
			runes := []rune(content)
			chunkSize := 8
			for i := 0; i < len(runes); i += chunkSize {
				select {
				case <-ctx.Done():
					if taskLogger != nil {
						taskLogger.Info("使用者已中斷回覆串流")
					}
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

		if taskLogger != nil {
			taskLogger.Info(fmt.Sprintf("AI 對話生成完成 | 消耗 Tokens: Prompt=%d, Completion=%d, Total=%d",
				roundPromptTokens, roundCompletionTokens, roundTotalTokens))
		}
		if progress != nil {
			progress(1.0, "AI 對話完成")
		}

		return nil
	}

	limitErr := fmt.Errorf("達到 Agentic Loop 最大迭代次數 (%d 輪)，AI 未能在限制次數內完成推論，已中止執行", a.maxIterations)
	if taskLogger != nil {
		taskLogger.Warn(limitErr.Error(), "maxIterations", a.maxIterations)
	}
	if emitter != nil {
		emitter.EmitEvent("ai:error", map[string]interface{}{
			"error": limitErr.Error(),
		})
	}
	return limitErr
}

// ParseToolCallsFromContent 嘗試從模型回傳的文字內容中提取 tool_call 標籤或結構
// 支援以下常見本地模型輸出格式：
// 1. Qwen 原生 XML 格式：<tool_call><function=func_name>...</tool_call> 或 <function=func_name...
// 2. Qwen 參數格式：<parameter=param_name>value</parameter>
// 3. Hermes / OpenAI JSON 格式：<tool_call>{"name": "...", "arguments": {...}}</tool_call>
// 4. Markdown 代碼塊包夾之 JSON 格式
//
// 參數:
//   - content: 模型回傳之原始字串
//
// 回傳:
//   - []ToolCall: 解析出之工具呼叫列表
//   - string: 剔除 tool_call 標籤後的剩餘文字內容
func ParseToolCallsFromContent(content string) ([]ToolCall, string) {
	if !strings.Contains(content, "<tool_call>") && !strings.Contains(content, "<function=") {
		return nil, content
	}

	var toolCalls []ToolCall
	var cleanContent strings.Builder
	cursor := 0
	callIdx := 1

	if strings.Contains(content, "<tool_call>") {
		for {
			startIdx := strings.Index(content[cursor:], "<tool_call>")
			if startIdx == -1 {
				cleanContent.WriteString(content[cursor:])
				break
			}
			actualStart := cursor + startIdx
			cleanContent.WriteString(content[cursor:actualStart])

			afterStart := actualStart + len("<tool_call>")
			endIdx := strings.Index(content[afterStart:], "</tool_call>")
			var rawBlock string
			if endIdx == -1 {
				// 若未閉合 </tool_call>，截取至字串結尾
				rawBlock = content[afterStart:]
				cursor = len(content)
			} else {
				rawBlock = content[afterStart : afterStart+endIdx]
				cursor = afterStart + endIdx + len("</tool_call>")
			}

			block := cleanCodeBlock(rawBlock)
			tc := parseSingleToolBlock(block, callIdx)
			if tc != nil {
				toolCalls = append(toolCalls, *tc)
				callIdx++
			}
		}
	} else if strings.Contains(content, "<function=") {
		// 容錯：若模型漏掉 <tool_call> 外層標籤，直接以 <function= 輸出
		startIdx := strings.Index(content, "<function=")
		cleanContent.WriteString(content[:startIdx])
		rawBlock := content[startIdx:]
		block := cleanCodeBlock(rawBlock)
		tc := parseSingleToolBlock(block, callIdx)
		if tc != nil {
			toolCalls = append(toolCalls, *tc)
		}
	}

	cleaned := strings.TrimSpace(cleanContent.String())
	return toolCalls, cleaned
}

// cleanCodeBlock 剝離 markdown 代碼塊標記 (```json ... ```)
func cleanCodeBlock(block string) string {
	block = strings.TrimSpace(block)
	if strings.HasPrefix(block, "```") {
		lines := strings.Split(block, "\n")
		if len(lines) >= 2 {
			if strings.HasPrefix(lines[len(lines)-1], "```") {
				lines = lines[1 : len(lines)-1]
			} else {
				lines = lines[1:]
			}
			block = strings.TrimSpace(strings.Join(lines, "\n"))
		}
	}
	return block
}

// parseSingleToolBlock 解析單一 tool_call 區塊字串
func parseSingleToolBlock(block string, callIdx int) *ToolCall {
	block = strings.TrimSpace(block)
	if block == "" {
		return nil
	}

	// 1. Qwen 格式: <function=name ...
	if strings.Contains(block, "<function=") {
		funcIdx := strings.Index(block, "<function=")
		afterFunc := block[funcIdx+len("<function="):]

		var funcName string
		var rest string
		delimIdx := strings.IndexAny(afterFunc, ">\n\r \t")
		if delimIdx == -1 {
			funcName = strings.TrimSpace(afterFunc)
			rest = ""
		} else {
			funcName = strings.TrimSpace(afterFunc[:delimIdx])
			rest = afterFunc[delimIdx:]
		}

		if funcName == "" {
			return nil
		}

		// 移除可能存在的 '>' 與 '</function>'
		rest = strings.TrimPrefix(strings.TrimSpace(rest), ">")
		if endFuncIdx := strings.Index(rest, "</function>"); endFuncIdx != -1 {
			rest = rest[:endFuncIdx]
		}
		rest = strings.TrimSpace(rest)

		// 檢查是否有 <parameter=name>value</parameter>
		if strings.Contains(rest, "<parameter=") {
			argsMap := make(map[string]interface{})
			paramRest := rest
			for {
				pIdx := strings.Index(paramRest, "<parameter=")
				if pIdx == -1 {
					break
				}
				afterParam := paramRest[pIdx+len("<parameter="):]
				pNameDelim := strings.IndexAny(afterParam, ">\n\r \t")
				if pNameDelim == -1 {
					break
				}
				pName := strings.TrimSpace(afterParam[:pNameDelim])
				pBody := strings.TrimPrefix(strings.TrimSpace(afterParam[pNameDelim:]), ">")

				var pVal string
				if endPIdx := strings.Index(pBody, "</parameter>"); endPIdx != -1 {
					pVal = strings.TrimSpace(pBody[:endPIdx])
					paramRest = pBody[endPIdx+len("</parameter>"):]
				} else {
					nextP := strings.Index(pBody, "<parameter=")
					if nextP != -1 {
						pVal = strings.TrimSpace(pBody[:nextP])
						paramRest = pBody[nextP:]
					} else {
						pVal = strings.TrimSpace(pBody)
						paramRest = ""
					}
				}

				if pName != "" {
					var jsonVal interface{}
					if err := json.Unmarshal([]byte(pVal), &jsonVal); err == nil {
						argsMap[pName] = jsonVal
					} else {
						argsMap[pName] = pVal
					}
				}
			}

			argsBytes, err := json.Marshal(argsMap)
			argsStr := "{}"
			if err == nil {
				argsStr = string(argsBytes)
			}

			return &ToolCall{
				ID:   fmt.Sprintf("call_qwen_%d_%d", callIdx, time.Now().UnixNano()%100000),
				Type: "function",
				Function: FunctionCall{
					Name:      funcName,
					Arguments: argsStr,
				},
			}
		}

		// 檢查 rest 是否包含 JSON 物件 {...}
		firstBrace := strings.Index(rest, "{")
		lastBrace := strings.LastIndex(rest, "}")
		if firstBrace != -1 && lastBrace != -1 && lastBrace > firstBrace {
			jsonCand := rest[firstBrace : lastBrace+1]
			var testMap map[string]interface{}
			if err := json.Unmarshal([]byte(jsonCand), &testMap); err == nil {
				return &ToolCall{
					ID:   fmt.Sprintf("call_qwen_%d_%d", callIdx, time.Now().UnixNano()%100000),
					Type: "function",
					Function: FunctionCall{
						Name:      funcName,
						Arguments: jsonCand,
					},
				}
			}
		}

		// 無參數，預設為 "{}"
		return &ToolCall{
			ID:   fmt.Sprintf("call_qwen_%d_%d", callIdx, time.Now().UnixNano()%100000),
			Type: "function",
			Function: FunctionCall{
				Name:      funcName,
				Arguments: "{}",
			},
		}
	}

	// 2. Hermes / OpenAI JSON 格式: {"name": "...", "arguments": ...}
	firstBrace := strings.Index(block, "{")
	lastBrace := strings.LastIndex(block, "}")
	if firstBrace != -1 && lastBrace != -1 && lastBrace > firstBrace {
		jsonStr := block[firstBrace : lastBrace+1]
		var rawMap map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &rawMap); err == nil {
			var name string
			if n, ok := rawMap["name"].(string); ok {
				name = n
			} else if n, ok := rawMap["function"].(string); ok {
				name = n
			}

			if name != "" {
				var argsStr string
				if rawArgs, exists := rawMap["arguments"]; exists {
					switch v := rawArgs.(type) {
					case string:
						argsStr = v
					default:
						b, _ := json.Marshal(v)
						argsStr = string(b)
					}
				} else if rawParams, exists := rawMap["parameters"]; exists {
					b, _ := json.Marshal(rawParams)
					argsStr = string(b)
				} else {
					delete(rawMap, "name")
					delete(rawMap, "function")
					b, _ := json.Marshal(rawMap)
					argsStr = string(b)
				}

				if argsStr == "" {
					argsStr = "{}"
				}

				return &ToolCall{
					ID:   fmt.Sprintf("call_json_%d_%d", callIdx, time.Now().UnixNano()%100000),
					Type: "function",
					Function: FunctionCall{
						Name:      name,
						Arguments: argsStr,
					},
				}
			}
		}
	}

	return nil
}

