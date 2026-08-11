package excel

import (
	"fmt"
	"sync"
)

// ProgressTracker 提供 Excel 解析與匯入過程中的動態進度計算與分段推播工具。
// 依據「已處理行數 / 總行數」計算進度，並以 10% 為單位（20%, 30%, 40% ... 90%）發送進度通知。
type ProgressTracker struct {
	totalRows       int
	processedRows   int
	lastReportedPct int
	progressCb      func(progress float64, message string)
	taskMsg         string
	mu              sync.Mutex
}

// NewProgressTracker 建立新的進度追蹤器。
//
// 參數:
//   - totalRows: 待處理的總列數 (若 <= 0 則預設為 1 以防除以零)
//   - startPct: 初始百分比（例：20 代表 20%）
//   - taskMsg: 進度提示訊息前綴
//   - progressCb: 進度回報回調函數
func NewProgressTracker(totalRows int, startPct int, taskMsg string, progressCb func(float64, string)) *ProgressTracker {
	if totalRows <= 0 {
		totalRows = 1
	}
	return &ProgressTracker{
		totalRows:       totalRows,
		processedRows:   0,
		lastReportedPct: startPct,
		progressCb:      progressCb,
		taskMsg:         taskMsg,
	}
}

// AddRows 增加已處理行數，並檢查是否跨越 10% 門檻。若跨越門檻且 progressCb 不為 nil 則觸發推播。
//
// 參數:
//   - count: 剛處理完畢的列數
func (pt *ProgressTracker) AddRows(count int) {
	if pt == nil || pt.progressCb == nil {
		return
	}

	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.processedRows += count
	if pt.processedRows > pt.totalRows {
		pt.processedRows = pt.totalRows
	}

	// 解析與寫入階段對應 Task 進度 20% ~ 90% (共 70% 區間)
	ratio := float64(pt.processedRows) / float64(pt.totalRows)
	pct := 20 + int(ratio*70.0)
	if pct > 90 {
		pct = 90
	}

	// 以 10% 為階梯單位向下取整（例如 34% -> 30%）
	pctStep := (pct / 10) * 10

	// 當累積進度跨越下一個 10% 門檻時觸發推播
	if pctStep >= pt.lastReportedPct+10 && pctStep <= 90 {
		pt.lastReportedPct = pctStep
		msg := fmt.Sprintf("%s (%d/%d 列)", pt.taskMsg, pt.processedRows, pt.totalRows)
		pt.progressCb(float64(pctStep)/100.0, msg)
	}
}

// ForceReport 強制以指定的百分比與訊息觸發進度推播。
//
// 參數:
//   - pct: 百分比（例：90 代表 90%）
//   - msg: 進度說明訊息
func (pt *ProgressTracker) ForceReport(pct int, msg string) {
	if pt == nil || pt.progressCb == nil {
		return
	}

	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.lastReportedPct = pct
	pt.progressCb(float64(pct)/100.0, msg)
}
