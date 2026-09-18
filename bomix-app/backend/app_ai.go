package backend

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"bomix-app/backend/ai"
	"bomix-app/backend/logger"
)

// ==================== AI 智慧助手 (AI Assistant) ====================

// AIChatSend 接收前端對話歷史，以 Task 任務形式啟動 AI Agentic Loop 進行推論與查詢
//
// 流程：
// 1. 檢查 AI 設定（URL 與 Key）
// 2. 檢查目前是否已開啟系列資料庫
// 3. 中斷任何先前正在執行的對話任務
// 4. 擷取使用者最後提問組成任務名稱，透過 taskMgr.Submit 派發 AIChat 任務
// 5. 執行過程透過 taskLogger 輸出結構化日誌（自動帶 taskID 歸屬 Log Group）
// 6. 立即回傳 nil，結果即時透過 ai:chunk / ai:tool_call / ai:tool_result / ai:done / ai:error 等事件推送
//
// 參數：
//   - messages: 完整的對話歷史紀錄
//
// 回傳：
//   - error: 若未啟用 AI、未開啟資料庫或未設定 URL 則回傳錯誤
func (a *App) AIChatSend(messages []ai.ChatMessage) error {
	a.mu.RLock()
	database := a.db
	aiCfg := a.cfg.AI
	a.mu.RUnlock()

	if !aiCfg.Enabled {
		return errors.New("AI Assistant 功能尚未啟用，請先至設定頁面開啟「Enable Assistant」")
	}

	if database == nil {
		return errors.New("請先開啟系列資料庫，AI 才能讀取物料與專案數據")
	}

	if aiCfg.BaseURL == "" {
		return errors.New("未設定 AI API Base URL，請至設定頁面設定")
	}

	// 確保先前進行中的 AI 任務已中斷
	_ = a.AIChatStop()

	// 取得使用者最後一則提問文字作為任務名稱摘要
	lastPrompt := "AI 對話"
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" && strings.TrimSpace(messages[i].Content) != "" {
			runes := []rune(strings.TrimSpace(messages[i].Content))
			if len(runes) > 25 {
				lastPrompt = string(runes[:25]) + "..."
			} else {
				lastPrompt = string(runes)
			}
			break
		}
	}
	taskName := fmt.Sprintf("AI 對話: %s", lastPrompt)

	client := ai.NewClient(aiCfg.BaseURL, aiCfg.APIKey, aiCfg.Model, aiCfg.Timeout)
	executor := ai.NewToolExecutor(database, a.logger)
	systemPrompt := ai.GetSystemPrompt(aiCfg.Language)
	agent := ai.NewAgent(client, executor, systemPrompt, aiCfg.MaxIterations, aiCfg.Temperature, aiCfg.MaxTokens)

	if a.taskMgr != nil {
		var taskID string
		taskID = a.taskMgr.Submit(
			taskName,
			"AIChat",
			func(ctx context.Context, progress func(float64, string), taskLogger *logger.Logger) error {
				a.aiMu.Lock()
				a.currentAITaskID = taskID
				a.aiMu.Unlock()

				defer func() {
					a.aiMu.Lock()
					if a.currentAITaskID == taskID {
						a.currentAITaskID = ""
					}
					a.aiMu.Unlock()
				}()

				// AI 對話不需要回報覆蓋在 log panel 的進度條，progress 傳入 nil
				err := agent.Run(ctx, messages, a, taskLogger, nil)
				if err != nil {
					if errors.Is(err, context.Canceled) || ctx.Err() == context.Canceled {
						return ctx.Err()
					}
					a.logger.Error("AI Agent 執行錯誤", "taskID", taskID, "error", err)
					return err
				}
				return nil
			},
		)

		a.aiMu.Lock()
		a.currentAITaskID = taskID
		a.aiMu.Unlock()
	} else {
		// 備援：若未初始化 taskMgr (例如獨立單元測試環境)，使用一般 goroutine
		a.aiMu.Lock()
		ctx, cancel := context.WithCancel(context.Background())
		a.aiCancelFunc = cancel
		a.aiMu.Unlock()

		go func() {
			defer func() {
				a.aiMu.Lock()
				a.aiCancelFunc = nil
				a.aiMu.Unlock()
			}()

			_ = agent.Run(ctx, messages, a, a.logger, nil)
		}()
	}

	return nil
}

// AIChatStop 中斷當前正在執行的 AI 生成或工具調用
//
// 回傳：
//   - error: 固定回傳 nil
func (a *App) AIChatStop() error {
	a.aiMu.Lock()
	defer a.aiMu.Unlock()

	if a.currentAITaskID != "" && a.taskMgr != nil {
		_ = a.taskMgr.Cancel(a.currentAITaskID)
		a.currentAITaskID = ""
	}
	if a.aiCancelFunc != nil {
		a.aiCancelFunc()
		a.aiCancelFunc = nil
	}
	return nil
}

// AIChatTestConnection 測試 AI 端點與 API Key 是否有效
//
// 回傳：
//   - error: 若 BaseURL 為空或遠端測試請求失敗則回傳錯誤
func (a *App) AIChatTestConnection() error {
	if a.cfg.AI.BaseURL == "" {
		return errors.New("API Base URL 不能為空")
	}

	timeoutSec := a.cfg.AI.Timeout
	if timeoutSec <= 0 {
		timeoutSec = 15
	}
	client := ai.NewClient(a.cfg.AI.BaseURL, a.cfg.AI.APIKey, a.cfg.AI.Model, timeoutSec)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	_, err := client.SendChat(ctx, ai.ChatCompletionRequest{
		Messages: []ai.ChatMessage{
			{Role: "user", Content: "Hi, this is a connection test. Reply with 'OK'."},
		},
		MaxTokens: 10,
	})
	if err != nil {
		return fmt.Errorf("連線測試失敗: %w", err)
	}

	return nil
}

// AIChatGetAvailableModels 透過當前設定的 BaseURL 與 APIKey 取得伺服器提供的可用模型清單
//
// 回傳：
//   - []string: 可用模型識別字串陣列
//   - error: 若 BaseURL 為空或請求失敗則回傳錯誤
func (a *App) AIChatGetAvailableModels() ([]string, error) {
	baseURL := a.cfg.AI.BaseURL
	if baseURL == "" {
		return nil, errors.New("API Base URL 不能為空")
	}

	timeoutSec := a.cfg.AI.Timeout
	if timeoutSec <= 0 || timeoutSec > 15 {
		timeoutSec = 15
	}
	client := ai.NewClient(baseURL, a.cfg.AI.APIKey, a.cfg.AI.Model, timeoutSec)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	return client.ListModels(ctx)
}

// AIChatFetchModelsWithConfig 支援前端使用指定的 BaseURL 與 APIKey 即時取得伺服器提供的可用模型清單
//
// 參數：
//   - baseURL: 自訂 API Base URL
//   - apiKey: 自訂 API Key
//
// 回傳：
//   - []string: 可用模型清單
//   - error: 若 BaseURL 為空或查詢失敗則回傳錯誤
func (a *App) AIChatFetchModelsWithConfig(baseURL, apiKey string) ([]string, error) {
	if strings.TrimSpace(baseURL) == "" {
		return nil, errors.New("API Base URL 不能為空")
	}

	client := ai.NewClient(strings.TrimSpace(baseURL), strings.TrimSpace(apiKey), "", 15)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	return client.ListModels(ctx)
}
