package ai

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ParseToolCallsFromContent 嘗試從模型回傳的文字內容中提取 tool_call 標籤或結構
// 支援以下常見本地開源模型（如 Qwen, Hermes, DeepSeek 等）輸出格式：
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
//   - string: 剔除 tool_call 標籤後的剩餘純文字內容
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
//
// 參數:
//   - block: 原始代碼區塊字串
//
// 回傳:
//   - string: 剝離 markdown 標記後的內文
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

// parseSingleToolBlock 解析單一 tool_call 區塊字串，支援 Qwen 與 Hermes/OpenAI 格式
//
// 參數:
//   - block: 工具調用區塊文字
//   - callIdx: 呼叫序號，用於生成唯一的 Call ID
//
// 回傳:
//   - *ToolCall: 解析成功回傳 ToolCall 指標，失敗或格式不符回傳 nil
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
