package backend

import (
	"context"
	"fmt"
	"strings"

	"bomix-app/backend/db"
	"bomix-app/backend/logger"
	"bomix-app/backend/view"
)

// ==================== BOM View 查詢與矩陣操作 (View & Matrix Operations) ====================

// GetBOMView 查詢 BOM 視圖資料，是 View 系統的 Wails 綁定入口。
//
// View 系統為無狀態設計，前端顯示與後端匯出可同時以不同條件查詢。
//
// 參數：
//   - revisionIDs: 要查詢的 BOM Revision ID 列表（1個=單一視圖，多個=整合視圖）
//   - viewType: 視圖類型（ALL/SMD/PTH/BOTTOM/NI/PROTO/MP/CCL），空字串預設為 ALL
//
// 回傳：
//   - *view.ViewResult: 查詢結果，包含聚合物料群組與 revision 元資料
//   - error: 若資料庫連線未開啟或查詢失敗則回傳錯誤
func (a *App) GetBOMView(revisionIDs []int64, viewType string) (*view.ViewResult, error) {
	a.mu.RLock()
	dbConn := a.db
	a.mu.RUnlock()

	if dbConn == nil {
		return nil, fmt.Errorf("no series is currently open")
	}

	vTypeUpper := strings.ToUpper(strings.TrimSpace(viewType))
	if vTypeUpper == "" {
		vTypeUpper = view.ViewAll
	}

	query := view.ViewQuery{
		RevisionIDs: revisionIDs,
		ViewType:    vTypeUpper,
	}

	svc := view.NewService(dbConn, a.logger)
	result, err := svc.Query(query)
	if err != nil {
		a.logger.Error(fmt.Sprintf("[GetBOMView] 視圖查詢失敗: %v", err))
		return nil, fmt.Errorf("view query failed: %w", err)
	}

	// 根據視圖類型判定是否進行二階跨製程物料合併：
	// ALL、NI、PROTO、MP、CCL 視圖將相同物料跨 Type（SMD/PTH/BOTTOM）合併為單一 Group（Qty 累加、Locations 合併去重、替代料僅顯示一組）；
	// SMD、PTH、BOTTOM 製程視圖則維持獨立面別群組，不進行跨 Type 合併。
	if vTypeUpper == view.ViewAll || vTypeUpper == view.ViewNI ||
		vTypeUpper == view.ViewProto || vTypeUpper == view.ViewMP || vTypeUpper == view.ViewCCL {
		result.PartGroups = view.MergePartGroupsByMaterial(result.PartGroups)
	}

	a.logger.Debug(fmt.Sprintf("[GetBOMView] 查詢完成: revisions=%d, parts=%d, viewType=%s (條件: %s)",
		len(result.Revisions), len(result.PartGroups), vTypeUpper, view.DescribeCondition(vTypeUpper, "")))

	return result, nil
}

// SetMatrixSelection 更新單一物料群組在指定 Revision 與 Model 的勾選狀態。
//
// 參數：
//   - revisionID: BOM Revision ID
//   - modelID: Matrix Model ID（若為 0 則自動查找或建立預設 Model）
//   - mainMaterialID: 主料 Material ID
//   - selectedMaterialID: 被選中的物料 Material ID（傳入 0 表示取消勾選）
//
// 回傳：
//   - error: 若資料庫未開啟或更新失敗則回傳錯誤
func (a *App) SetMatrixSelection(revisionID, modelID, mainMaterialID, selectedMaterialID int64) error {
	a.mu.RLock()
	dbConn := a.db
	a.mu.RUnlock()

	if dbConn == nil {
		return fmt.Errorf("no series is currently open")
	}

	if err := db.UpsertMatrixSelection(dbConn, revisionID, modelID, mainMaterialID, selectedMaterialID); err != nil {
		a.logger.Error(fmt.Sprintf("[SetMatrixSelection] 更新失敗: %v", err))
		return fmt.Errorf("failed to set matrix selection: %w", err)
	}

	a.logger.Debug(fmt.Sprintf("[SetMatrixSelection] 更新成功: revID=%d, modelID=%d, mainMatID=%d, selectedMatID=%d",
		revisionID, modelID, mainMaterialID, selectedMaterialID))
	return nil
}

// SetMatrixModelSelection 更新單一物料群組在指定 Revision 與 Model 排序索引 (SortOrder) 的勾選狀態。
//
// 參數：
//   - revisionID: BOM Revision ID
//   - sortOrder: 0-based Model 排序索引 (0, 1, 2...)
//   - mainMaterialID: 主料 Material ID
//   - selectedMaterialID: 被選中的物料 Material ID（傳入 0 表示取消勾選）
//
// 回傳：
//   - error: 若資料庫未開啟或更新失敗則回傳錯誤
func (a *App) SetMatrixModelSelection(revisionID int64, sortOrder int, mainMaterialID, selectedMaterialID int64) error {
	a.mu.RLock()
	dbConn := a.db
	a.mu.RUnlock()

	if dbConn == nil {
		return fmt.Errorf("no series is currently open")
	}

	if err := db.UpsertMatrixModelSelection(dbConn, revisionID, sortOrder, mainMaterialID, selectedMaterialID); err != nil {
		a.logger.Error(fmt.Sprintf("[SetMatrixModelSelection] 更新失敗: %v", err))
		return fmt.Errorf("failed to set matrix model selection: %w", err)
	}

	a.logger.Debug(fmt.Sprintf("[SetMatrixModelSelection] 更新成功: revID=%d, sortOrder=%d, mainMatID=%d, selectedMatID=%d",
		revisionID, sortOrder, mainMaterialID, selectedMaterialID))
	return nil
}

// UpdateRevisionMatrixModels 批次更新指定 Revision 的所有 MatrixModel (包含 Model 數量與 Model Qty)。
//
// 參數：
//   - revisionID: BOM Revision ID
//   - models: 欲更新的 Model 列表 (含 SortOrder, ModelName, Qty)
//
// 回傳：
//   - error: 若資料庫未開啟或更新失敗則回傳錯誤
func (a *App) UpdateRevisionMatrixModels(revisionID int64, models []db.MatrixModelInput) error {
	a.mu.RLock()
	dbConn := a.db
	a.mu.RUnlock()

	if dbConn == nil {
		return fmt.Errorf("no series is currently open")
	}

	if revisionID <= 0 {
		return fmt.Errorf("invalid revision ID: %d", revisionID)
	}

	if err := db.SaveRevisionMatrixModels(dbConn, revisionID, models); err != nil {
		a.logger.Error(fmt.Sprintf("[UpdateRevisionMatrixModels] 更新失敗 (revisionID=%d): %v", revisionID, err))
		return fmt.Errorf("failed to update revision matrix models: %w", err)
	}

	a.logger.Info(fmt.Sprintf("[UpdateRevisionMatrixModels] 成功更新 revision ID=%d 的 models 共 %d 個", revisionID, len(models)))
	return nil
}

// UpdateMaterialNote 更新指定物料的 Notes 欄位內容，並持久化至資料庫。
//
// 參數：
//   - materialID: 物料 ID (Material.ID)
//   - notes: 新的 Notes 內容
//
// 回傳：
//   - error: 若資料庫未開啟或更新失敗則回傳錯誤
func (a *App) UpdateMaterialNote(materialID int64, notes string) error {
	a.mu.RLock()
	dbConn := a.db
	a.mu.RUnlock()

	if dbConn == nil {
		return fmt.Errorf("no series is currently open")
	}

	if materialID <= 0 {
		return fmt.Errorf("invalid material ID: %d", materialID)
	}

	if err := db.UpdateMaterialNote(dbConn, materialID, notes); err != nil {
		a.logger.Error(fmt.Sprintf("[UpdateMaterialNote] 更新物料 ID=%d 的 Notes 失敗: %v", materialID, err))
		return fmt.Errorf("update material note failed: %w", err)
	}

	a.logger.Info(fmt.Sprintf("[UpdateMaterialNote] 成功更新物料 ID=%d 的 Notes: %q", materialID, notes))
	return nil
}

// CopyMatrixSelections 以異步任務形式，手動將指定 source revision 的 Matrix Model 與 Selection 複製到 target revision。
//
// 此函數為手動版本複製的 Wails 綁定入口，會以 Task 形式提交至背景執行，
// 讓 UI 可透過 Task ID 追蹤執行進度與結果。
//
// 參數：
//   - sourceRevisionID: 來源版本 ID（Matrix 資料的來源）
//   - targetRevisionID: 目標版本 ID（Matrix 資料的目的地）
//
// 回傳：
//   - string: 任務 ID（taskID），可用於前端 Task 追蹤
//   - error: 若資料庫未開啟或 revision 不存在則回傳錯誤
func (a *App) CopyMatrixSelections(sourceRevisionID, targetRevisionID int64) (string, error) {
	a.mu.RLock()
	dbConn := a.db
	a.mu.RUnlock()

	if dbConn == nil {
		return "", fmt.Errorf("no series is currently open")
	}

	// 預先讀取 source 與 target revision 資訊，供 Task Log 使用
	var sourceRev, targetRev db.BomRevision
	if err := dbConn.First(&sourceRev, sourceRevisionID).Error; err != nil {
		return "", fmt.Errorf("找不到來源 Revision ID=%d: %w", sourceRevisionID, err)
	}
	if err := dbConn.First(&targetRev, targetRevisionID).Error; err != nil {
		return "", fmt.Errorf("找不到目標 Revision ID=%d: %w", targetRevisionID, err)
	}

	taskName := fmt.Sprintf("Copy Matrix: %s → %s", sourceRev.Version, targetRev.Version)

	taskID := a.taskMgr.Submit(
		taskName,
		"CopyMatrix",
		func(ctx context.Context, progress func(float64, string), taskLogger *logger.Logger) error {
			progress(0.1, fmt.Sprintf("開始複製 Matrix Selection（%s → %s）", sourceRev.Version, targetRev.Version))
			taskLogger.Info(fmt.Sprintf("[CopyMatrix] 開始執行 | 來源 RevisionID=%d (Version=%s) → 目標 RevisionID=%d (Version=%s)",
				sourceRevisionID, sourceRev.Version, targetRevisionID, targetRev.Version))

			progress(0.3, "正在複製 Matrix Model 與 Selection...")

			// 執行覆蓋式 Matrix 複製
			stats, err := db.ImportMatrixSelections(dbConn, sourceRevisionID, targetRevisionID, taskLogger)
			if err != nil {
				taskLogger.Error(fmt.Sprintf("[CopyMatrix] 複製失敗: %v", err))
				return fmt.Errorf("複製 Matrix Selection 失敗: %w", err)
			}

			// 輸出完整統計結果
			resultMsg := fmt.Sprintf(
				"Matrix 複製完成 | 有效 Model 數=%d, 複製 Selection 數=%d, 忽略主料數=%d, 忽略 2nd 替代料數=%d",
				stats.SourceModelCount, stats.CopiedSelectionsCount,
				stats.IgnoredMainParts, stats.IgnoredSecondParts,
			)
			taskLogger.Info(fmt.Sprintf("[CopyMatrix] %s", resultMsg))
			progress(1.0, resultMsg)

			return nil
		},
	)

	a.logger.Info(fmt.Sprintf("[CopyMatrixSelections] 已提交任務 %s (taskID=%s)", taskName, taskID))
	return taskID, nil
}
