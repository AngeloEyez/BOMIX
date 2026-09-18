package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ChatMessage 對話訊息結構
type ChatMessage struct {
	Role       string     `json:"role"`                  // "system" | "user" | "assistant" | "tool"
	Content    string     `json:"content"`               // 訊息內文
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`  // Assistant 發出的工具呼叫請求清單
	ToolCallID string     `json:"tool_call_id,omitempty"`// Tool 角色對應的呼叫 ID
	Name       string     `json:"name,omitempty"`        // 函數名稱（可選）
}

// ToolCall LLM 回傳的工具調用結構
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // 固定為 "function"
	Function FunctionCall `json:"function"`
}

// FunctionCall 工具調用的函數名稱與參數
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON 字串
}

// ToolDef OpenAI Function Tool 宣告結構
type ToolDef struct {
	Type     string      `json:"type"` // "function"
	Function FunctionDef `json:"function"`
}

// FunctionDef 工具宣告細節
type FunctionDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

// ChatCompletionRequest 聊天請求負載
type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Tools       []ToolDef     `json:"tools,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream"`
}

// ChatCompletionResponse 聊天回應結構
type ChatCompletionResponse struct {
	ID      string         `json:"id"`
	Choices []Choice       `json:"choices"`
	Usage   *Usage         `json:"usage,omitempty"`
	Error   *APIErrorDetail `json:"error,omitempty"`
}

// Choice 單一選項回應
type Choice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"` // "stop" | "tool_calls" | "length"
}

// Usage Token 使用量統計
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// APIErrorDetail API 錯誤明細
type APIErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// Client 封裝與 OpenAI 相容 API 的 HTTP 通訊
type Client struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewClient 建立新的 OpenAI API 客戶端實例
//
// 參數:
//   - baseURL: API 基礎網址（支援 OpenAI、DeepSeek、Ollama 等）
//   - apiKey: API 金鑰
//   - model: 預設模型名稱
//   - timeoutSec: HTTP 請求逾時秒數（小於等於 0 時預設為 60 秒）
//
// 回傳:
//   - *Client 客戶端實例
func NewClient(baseURL, apiKey, model string, timeoutSec int) *Client {
	if timeoutSec <= 0 {
		timeoutSec = 60
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSec) * time.Second,
		},
	}
}

// ModelListResponse OpenAI 相容端點的 /models 回應結構
type ModelListResponse struct {
	Object string      `json:"object"`
	Data   []ModelItem `json:"data"`
}

// ModelItem 單一模型資訊
type ModelItem struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created,omitempty"`
	OwnedBy string `json:"owned_by,omitempty"`
}

// buildURL 組合完整的 chat/completions 端點 URL
func (c *Client) buildURL() string {
	if strings.HasSuffix(c.baseURL, "/chat/completions") {
		return c.baseURL
	}
	if strings.HasSuffix(c.baseURL, "/v1") {
		return c.baseURL + "/chat/completions"
	}
	return c.baseURL + "/v1/chat/completions"
}

// buildModelsURL 組合完整的 models 端點 URL (OpenAI 相容規範為 GET /v1/models 或 /models)
func (c *Client) buildModelsURL() string {
	base := strings.TrimRight(c.baseURL, "/")
	if strings.HasSuffix(base, "/chat/completions") {
		base = strings.TrimSuffix(base, "/chat/completions")
	}
	if strings.HasSuffix(base, "/v1") {
		return base + "/models"
	}
	return base + "/v1/models"
}

// ListModels 呼叫 GET /models 取得可用的模型 ID 清單
//
// 參數:
//   - ctx: 呼叫上下文
//
// 回傳:
//   - []string: 伺服器回傳之模型 ID 清單
//   - error: 請求失敗或非 200 狀態碼時回傳錯誤
func (c *Client) ListModels(ctx context.Context) ([]string, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.buildModelsURL(), nil)
	if err != nil {
		return nil, fmt.Errorf("建立模型清單 HTTP 請求失敗: %w", err)
	}

	httpReq.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("發送模型清單請求失敗: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("讀取模型清單回應資料失敗: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("取得模型清單失敗 (HTTP %d): %s", resp.StatusCode, string(respBytes))
	}

	var listResp ModelListResponse
	if err := json.Unmarshal(respBytes, &listResp); err != nil {
		return nil, fmt.Errorf("解析模型清單 JSON 失敗: %w", err)
	}

	models := make([]string, 0, len(listResp.Data))
	for _, item := range listResp.Data {
		trimmedID := strings.TrimSpace(item.ID)
		if trimmedID != "" {
			models = append(models, trimmedID)
		}
	}

	return models, nil
}

// SendChat 發送同步對話請求（非串流模式，供 Agentic Loop 內部呼叫）
//
// 參數:
//   - ctx: 呼叫上下文
//   - req: 對話請求參數
//
// 回傳:
//   - *ChatCompletionResponse: 模型完整回應
//   - error: 請求失敗或非 200 狀態碼時回傳錯誤
func (c *Client) SendChat(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error) {
	if req.Model == "" {
		req.Model = c.model
	}
	req.Stream = false

	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化請求資料失敗: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.buildURL(), bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("建立 HTTP 請求失敗: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("發送 HTTP 請求失敗: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("讀取 API 回應本體失敗: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ChatCompletionResponse
		if json.Unmarshal(respBytes, &errResp) == nil && errResp.Error != nil {
			return nil, fmt.Errorf("API 錯誤 (HTTP %d): %s", resp.StatusCode, errResp.Error.Message)
		}
		return nil, fmt.Errorf("API 請求失敗 (HTTP %d): %s", resp.StatusCode, string(respBytes))
	}

	var chatResp ChatCompletionResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return nil, fmt.Errorf("反序列化 API 回應失敗: %w", err)
	}

	return &chatResp, nil
}

// StreamChunk SSE 串流資料片段結構
type StreamChunk struct {
	ID      string `json:"id"`
	Choices []struct {
		Delta struct {
			Role      string     `json:"role"`
			Content   string     `json:"content"`
			ToolCalls []ToolCall `json:"tool_calls,omitempty"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// SendChatStream 發送 SSE 串流對話請求（逐字推送給前端）
//
// 參數:
//   - ctx: 呼叫上下文
//   - req: 對話請求參數
//   - onChunk: 每次收到增量文字片段時的回呼函式
//
// 回傳:
//   - *ChatCompletionResponse: 累積完成的完整回應
//   - error: 串流通訊失敗時回傳錯誤
func (c *Client) SendChatStream(ctx context.Context, req ChatCompletionRequest, onChunk func(chunk string)) (*ChatCompletionResponse, error) {
	if req.Model == "" {
		req.Model = c.model
	}
	req.Stream = true

	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化串流請求資料失敗: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.buildURL(), bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("建立串流 HTTP 請求失敗: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("發送串流 HTTP 請求失敗: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("串流請求失敗 (HTTP %d): %s", resp.StatusCode, string(respBytes))
	}

	reader := bufio.NewReader(resp.Body)
	var fullContent strings.Builder
	var lastFinishReason string

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("讀取串流資料失敗: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}

		var chunk StreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta
			if delta.Content != "" {
				fullContent.WriteString(delta.Content)
				if onChunk != nil {
					onChunk(delta.Content)
				}
			}
			if chunk.Choices[0].FinishReason != "" {
				lastFinishReason = chunk.Choices[0].FinishReason
			}
		}
	}

	return &ChatCompletionResponse{
		Choices: []Choice{
			{
				Message: ChatMessage{
					Role:    "assistant",
					Content: fullContent.String(),
				},
				FinishReason: lastFinishReason,
			},
		},
	}, nil
}
