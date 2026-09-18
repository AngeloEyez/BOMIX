package backend

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"bomix-app/backend/excel"
	"bomix-app/backend/logger"
	"bomix-app/backend/task"
	"bomix-app/backend/types"

	"github.com/google/uuid"
)

// ==================== Excel 匯入工作流 (Excel Import Workflow) ====================

// isMatrixFile 判斷給定的檔案路徑之檔案名稱是否包含 "matrix"（不區分大小寫）
//
// 參數：
//   - filePath: 檔案完整或相對路徑
//
// 回傳：
//   - 若檔案名稱（不含路徑目錄）包含 "matrix"（忽略大小寫）則回傳 true，否則回傳 false
func isMatrixFile(filePath string) bool {
	fileName := filepath.Base(filePath)
	return strings.Contains(strings.ToLower(fileName), "matrix")
}

// ImportExcel 依兩階段分組匯入 Excel 檔案至資料庫
//
// 執行邏輯：
// 1. 檔案分組：依檔名是否包含 "matrix" (不區分大小寫) 分為非 Matrix 組與 Matrix 組。
// 2. 兩階段執行：
//   - 第一階段：先背景執行非 Matrix 組 (EBOM/一般 BOM) 的匯入任務。
//   - 第二階段：等待第一階段所有群組任務全數結束後，Matrix 組自動接續執行開檔與匯入。
//
// 參數：
//   - filePaths: 欲匯入的 Excel 檔案路徑清單
//   - confirmOverwrite: 匯入覆蓋現有 BOM 前是否提示確認
//
// 回傳：
//   - []*ImportResult: 包含所有提交任務之 Task ID 與狀態資訊
//   - error: 若當前無開啟中的 Series 則回傳錯誤
func (a *App) ImportExcel(filePaths []string, confirmOverwrite bool) ([]*ImportResult, error) {
	a.logger.Debug(fmt.Sprintf("準備匯入 %d 個 Excel 檔案 (confirmOverwrite=%v)", len(filePaths), confirmOverwrite))

	a.mu.RLock()
	dbConn := a.db
	a.mu.RUnlock()

	if dbConn == nil {
		return nil, fmt.Errorf("no series is currently open")
	}

	// 1. 檔案分組：將檔案依檔名是否含 "matrix" 分為兩組
	var nonMatrixPaths []string
	var matrixPaths []string
	for _, fp := range filePaths {
		if isMatrixFile(fp) {
			matrixPaths = append(matrixPaths, fp)
		} else {
			nonMatrixPaths = append(nonMatrixPaths, fp)
		}
	}

	results := make([]*ImportResult, 0, len(filePaths))

	// 第一階段同步機制：追蹤非 Matrix 任務群組之完成狀態
	var nonMatrixWg sync.WaitGroup
	var phase1Done chan struct{}

	if len(nonMatrixPaths) > 0 {
		nonMatrixWg.Add(len(nonMatrixPaths))
		phase1Done = make(chan struct{})

		// 在背景 goroutine 等待第一階段所有任務完成（包含等待確認完成）後關閉 channel 通知
		go func() {
			nonMatrixWg.Wait()
			close(phase1Done)
		}()
	}

	// 2. 第一階段：提交非 Matrix 檔案匯入任務
	for _, filePath := range nonMatrixPaths {
		curTaskID := uuid.New().String()
		curTaskName := fmt.Sprintf("Import: %s", filepath.Base(filePath))
		curFilePath := filePath

		func(taskID, taskName, fp string) {
			a.taskMgr.SubmitWithID(
				taskID,
				taskName,
				"Import",
				func(ctx context.Context, progress func(float64, string), taskLogger *logger.Logger) error {
					defer nonMatrixWg.Done()

					taskLogger.Info("開始執行基礎 BOM 匯入作業...", "name", taskName, "taskID", taskID)
					progress(0.2, "正在開啟與辨識 Excel 檔案...")

					// 建立專屬單次開檔與解析的 Reader
					taskExcelReader := excel.NewReader(dbConn, taskLogger)
					taskExcelReader.SetProgressCallback(progress)
					taskExcelReader.SetConfirmOverwrite(confirmOverwrite)

					// 注入覆蓋確認回調函式（綁定特定 taskID）
					taskExcelReader.SetConfirmOverwriteCallback(func(projectCode, phase, version string) (bool, error) {
						ch := make(chan bool, 1)
						a.confirmMu.Lock()
						a.confirmChans[taskID] = ch
						a.confirmMu.Unlock()

						msg := fmt.Sprintf("發現既有版本 (%s %s %s)，等待確認是否覆蓋...", projectCode, phase, version)
						a.taskMgr.SetTaskStatus(taskID, types.TaskWaitingConfirm, msg)
						a.EmitEvent("task:waiting_confirm", map[string]interface{}{
							"taskID":      taskID,
							"projectCode": projectCode,
							"phase":       phase,
							"version":     version,
							"message":     msg,
							"status":      "waiting_confirm",
						})

						taskLogger.Warn(msg, "taskID", taskID, "project", projectCode, "phase", phase, "version", version)

						select {
						case approve, ok := <-ch:
							if !ok {
								return false, errors.New("confirmation channel closed")
							}
							a.taskMgr.SetTaskStatus(taskID, types.TaskRunning, "使用者已完成確認，繼續處理中...")
							return approve, nil
						case <-ctx.Done():
							a.confirmMu.Lock()
							delete(a.confirmChans, taskID)
							a.confirmMu.Unlock()
							return false, ctx.Err()
						}
					})

					// 執行單次開檔、前置驗證與資料匯入
					importResults, err := taskExcelReader.ImportExcel([]string{fp})
					if err != nil {
						taskLogger.Error("開啟或讀取 Excel 檔案重大失敗", "error", err.Error(), "taskID", taskID)
						return err
					}

					if len(importResults) > 0 {
						result := importResults[0]

						// 檢查是否有格式解析錯誤、Unknown 格式，或是非 Matrix 格式但 PartsCount 為 0 筆
						isWarning := len(result.Errors) > 0 || result.Format == types.FormatUnknown || (result.Format != types.FormatMatrix && result.PartsCount == 0)
						if isWarning {
							errMsg := fmt.Sprintf("匯入結果需要確認: 格式=%s, 成功筆數=%d", result.Format, result.PartsCount)
							if len(result.Errors) > 0 {
								errMsg += fmt.Sprintf(", 錯誤=%v", result.Errors)
							}
							taskLogger.Warn(errMsg, "taskID", taskID)
							progress(0.9, errMsg)
							return task.NewWarningError(errors.New(errMsg))
						}

						msg := fmt.Sprintf("成功匯入 %d 筆料件", result.PartsCount)
						progress(0.9, msg)
						taskLogger.Info(msg, "taskID", taskID)
					}

					progress(1.0, "匯入作業完成")
					return nil
				},
			)

			results = append(results, &ImportResult{
				FileName: filepath.Base(fp),
				Status:   "queued",
				Message:  "Task created",
				TaskID:   taskID,
			})
		}(curTaskID, curTaskName, curFilePath)
	}

	// 3. 第二階段：提交 Matrix 檔案匯入任務（若有第一階段，等待第一階段結束後再執行）
	for _, filePath := range matrixPaths {
		curTaskID := uuid.New().String()
		curTaskName := fmt.Sprintf("Import: %s", filepath.Base(filePath))
		curFilePath := filePath

		func(taskID, taskName, fp string) {
			a.taskMgr.SubmitWithID(
				taskID,
				taskName,
				"Import",
				func(ctx context.Context, progress func(float64, string), taskLogger *logger.Logger) error {
					// 若有第一階段非 Matrix 任務，先行等待其群組結束
					if phase1Done != nil {
						taskLogger.Info("等待第一階段非 Matrix BOM 檔案匯入完成...", "name", taskName, "taskID", taskID)
						progress(0.05, "等待非 Matrix BOM 檔案匯入完成...")

						select {
						case <-ctx.Done():
							return ctx.Err()
						case <-phase1Done:
							taskLogger.Info("第一階段匯入已完成，開始執行 Matrix BOM 匯入作業...", "name", taskName, "taskID", taskID)
						}
					} else {
						taskLogger.Info("開始執行 Matrix BOM 匯入作業...", "name", taskName, "taskID", taskID)
					}

					progress(0.2, "正在開啟與辨識 Excel 檔案...")

					// 建立專屬單次開檔與解析的 Reader
					taskExcelReader := excel.NewReader(dbConn, taskLogger)
					taskExcelReader.SetProgressCallback(progress)

					// 執行單次開檔、前置驗證與資料匯入
					importResults, err := taskExcelReader.ImportExcel([]string{fp})
					if err != nil {
						taskLogger.Error("開啟或讀取 Excel 檔案重大失敗", "error", err.Error(), "taskID", taskID)
						return err
					}

					if len(importResults) > 0 {
						result := importResults[0]

						// 檢查是否有格式解析錯誤、Unknown 格式
						isWarning := len(result.Errors) > 0 || result.Format == types.FormatUnknown
						if isWarning {
							errMsg := fmt.Sprintf("匯入結果需要確認: 格式=%s, 成功筆數=%d", result.Format, result.PartsCount)
							if len(result.Errors) > 0 {
								errMsg += fmt.Sprintf(", 錯誤=%v", result.Errors)
							}
							taskLogger.Warn(errMsg, "taskID", taskID)
							progress(0.9, errMsg)
							return task.NewWarningError(errors.New(errMsg))
						}

						msg := "成功匯入 Matrix BOM 勾選資料與 Model 規格"
						progress(0.9, msg)
						taskLogger.Info(msg, "taskID", taskID)
					}

					progress(1.0, "匯入作業完成")
					return nil
				},
			)

			results = append(results, &ImportResult{
				FileName: filepath.Base(fp),
				Status:   "queued",
				Message:  "Task created",
				TaskID:   taskID,
			})
		}(curTaskID, curTaskName, curFilePath)
	}

	return results, nil
}

// ConfirmTaskOverwrite 回應指定任務的覆蓋確認請求。
//
// 參數：
//   - taskID: 任務 ID
//   - overwrite: true 表示確認覆蓋，false 表示略過覆蓋
//
// 回傳：
//   - error: 若任務不存在或未處於等待確認狀態則回傳錯誤
func (a *App) ConfirmTaskOverwrite(taskID string, overwrite bool) error {
	a.confirmMu.Lock()
	ch, exists := a.confirmChans[taskID]
	if exists {
		delete(a.confirmChans, taskID)
	}
	a.confirmMu.Unlock()

	if !exists {
		return fmt.Errorf("任務 %s 目前未處於等待確認狀態", taskID)
	}

	// 解鎖 goroutine 繼續執行
	ch <- overwrite
	return nil
}
