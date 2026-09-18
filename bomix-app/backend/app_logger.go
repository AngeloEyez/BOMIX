package backend

import (
	"time"
)

// ==================== 日誌系統 (Logger & Console) ====================

// LogFrontend 接收前端 UI 轉發之日誌訊息，並統一寫入後端結構化日誌系統
//
// 參數：
//   - level: 日誌層級（"DEBUG", "WARN", "ERROR", 預設為 "INFO"）
//   - message: 日誌訊息內容
func (a *App) LogFrontend(level string, message string) {
	switch level {
	case "DEBUG":
		a.logger.Debug(message)
	case "WARN":
		a.logger.Warn(message)
	case "ERROR":
		a.logger.Error(message)
	default:
		a.logger.Info(message)
	}
}

// GetLogs 依層級與筆數上限查詢日誌環形緩衝區中的紀錄
//
// 參數：
//   - level: 欲過濾的最低層級（如 "DEBUG", "INFO", "WARN", "ERROR"）
//   - limit: 最多回傳筆數
//
// 回傳：
//   - []*LogEntry: 日誌紀錄清單 DTO
//   - error: 固定回傳 nil
func (a *App) GetLogs(level string, limit int) ([]*LogEntry, error) {
	entries := a.logger.GetLogs(level, limit)

	result := make([]*LogEntry, len(entries))
	for i, e := range entries {
		result[i] = &LogEntry{
			Level:     e.Level,
			Message:   e.Message,
			Timestamp: e.Timestamp.Format(time.RFC3339),
			Attrs:     e.Attrs,
		}
	}

	return result, nil
}

// ClearLogs 清除記憶體中的所有日誌環形緩衝區紀錄
//
// 回傳：
//   - error: 固定回傳 nil
func (a *App) ClearLogs() error {
	a.logger.ClearLogs()
	return nil
}
