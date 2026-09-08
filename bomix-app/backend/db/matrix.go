package db

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// CreateMatrixModel creates a new matrix model
func CreateMatrixModel(db *gorm.DB, model *MatrixModel) error {
	return db.Create(model).Error
}

// GetMatrixModels returns all matrix models for a revision
func GetMatrixModels(db *gorm.DB, revisionID int64) ([]MatrixModel, error) {
	var models []MatrixModel
	if err := db.Where("revision_id = ?", revisionID).Preload("Selections").Find(&models).Error; err != nil {
		return nil, err
	}
	return models, nil
}

// GetMatrixModel returns a matrix model by ID
func GetMatrixModel(db *gorm.DB, id int64) (*MatrixModel, error) {
	var model MatrixModel
	if err := db.First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMatrixModelNotFound
		}
		return nil, err
	}
	return &model, nil
}

// UpdateMatrixModel updates a matrix model
func UpdateMatrixModel(db *gorm.DB, model *MatrixModel) error {
	return db.Save(model).Error
}

// CreateMatrixSelections creates matrix selections in batch
func CreateMatrixSelections(db *gorm.DB, selections []MatrixSelection) error {
	return db.Create(&selections).Error
}

// DeleteMatrixSelectionsByRevision deletes all matrix selections for a revision
func DeleteMatrixSelectionsByRevision(db *gorm.DB, revisionID int64) error {
	return db.Where("revision_id = ?", revisionID).Delete(&MatrixSelection{}).Error
}

// DeleteMatrixSelection deletes a matrix selection by ID
func DeleteMatrixSelection(db *gorm.DB, id int64) error {
	return db.Delete(&MatrixSelection{}, id).Error
}

// GetMatrixSelections returns matrix selections for a revision and model
func GetMatrixSelections(db *gorm.DB, revisionID int64, modelID int64) ([]MatrixSelection, error) {
	var selections []MatrixSelection
	query := db.Where("revision_id = ? AND model_id = ?", revisionID, modelID)
	if err := query.Find(&selections).Error; err != nil {
		return nil, err
	}
	return selections, nil
}

// GetMatrixSelectionsByRevision returns all matrix selections for a revision
func GetMatrixSelectionsByRevision(db *gorm.DB, revisionID int64) ([]MatrixSelection, error) {
	var selections []MatrixSelection
	if err := db.Where("revision_id = ?", revisionID).Find(&selections).Error; err != nil {
		return nil, err
	}
	return selections, nil
}

// GetMatrixSelectionsByGroupMaterial 依據主料 MaterialID 查詢指定 Revision 的 MatrixSelections
func GetMatrixSelectionsByGroupMaterial(db *gorm.DB, revisionID int64, mainMaterialID int64) ([]MatrixSelection, error) {
	var selections []MatrixSelection
	if err := db.Where("revision_id = ? AND main_material_id = ?", revisionID, mainMaterialID).Find(&selections).Error; err != nil {
		return nil, err
	}
	return selections, nil
}

// DeleteInvalidSelections 根據移除的主料 MaterialIDs 或選中 MaterialIDs 刪除無效的 MatrixSelection
// 於 EBOM merge 或零件異動時清理
func DeleteInvalidSelections(db *gorm.DB, revisionID int64, removedMainMaterialIDs []int64, removedSelectedMaterialIDs []int64) error {
	if len(removedMainMaterialIDs) == 0 && len(removedSelectedMaterialIDs) == 0 {
		return nil
	}

	query := db.Where("revision_id = ?", revisionID)
	if len(removedMainMaterialIDs) > 0 && len(removedSelectedMaterialIDs) > 0 {
		query = query.Where("main_material_id IN ? OR selected_material_id IN ?", removedMainMaterialIDs, removedSelectedMaterialIDs)
	} else if len(removedMainMaterialIDs) > 0 {
		query = query.Where("main_material_id IN ?", removedMainMaterialIDs)
	} else {
		query = query.Where("selected_material_id IN ?", removedSelectedMaterialIDs)
	}

	return query.Delete(&MatrixSelection{}).Error
}

// MatrixLogger 為 ImportMatrixSelections 提供 log 輸出的介面。
// 定義為本地介面以避免 db 套件直接依賴 logger 套件造成循環引用風險。
type MatrixLogger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
}

// ImportMatrixStats 記錄 ImportMatrixSelections 執行結果的統計數據。
type ImportMatrixStats struct {
	// SourceModelCount Source revision 中有效的 model 數量（qty > 0 或有 selection 記錄）
	SourceModelCount int
	// CopiedSelectionsCount 成功複製的 MatrixSelection 筆數
	CopiedSelectionsCount int
	// IgnoredMainParts Target revision 中不存在對應主料，故忽略的 selection 數
	IgnoredMainParts int
	// IgnoredSecondParts Target revision 主料存在，但被選中的物料不存在，故忽略的 selection 數
	IgnoredSecondParts int
}

// ImportMatrixSelections 將 Source Revision 的 Matrix Model 結構與 Selection 複製到 Target Revision。
//
// 設計原則：
//   - Model 對應：以 SortOrder（順序索引 0, 1, 2...）區分 Model。
//   - 物料對應：以 MainMaterialID 與 SelectedMaterialID（全域 Material ID）精準比對。
//   - 覆蓋模式：先清空 Target Revision 原有的 MatrixModel 與 MatrixSelection，再寫入新資料。
//
// 參數：
//   - db: GORM 資料庫連線
//   - sourceRevisionID: 來源版本 ID
//   - targetRevisionID: 目標版本 ID
//   - lg: MatrixLogger 介面實例（可為 nil）
//
// 回傳：
//   - *ImportMatrixStats: 執行結果統計
//   - error: 若資料庫操作失敗則回傳錯誤
func ImportMatrixSelections(db *gorm.DB, sourceRevisionID, targetRevisionID int64, lg MatrixLogger) (*ImportMatrixStats, error) {
	stats := &ImportMatrixStats{}

	// ─── Step 1：讀取 Source 的 MatrixModel 清單（依 SortOrder 排序）─────────
	var sourceModels []MatrixModel
	if err := db.Where("revision_id = ?", sourceRevisionID).Order("sort_order ASC").Find(&sourceModels).Error; err != nil {
		return nil, fmt.Errorf("讀取 source MatrixModel 失敗: %w", err)
	}

	// 讀取 Source 所有 MatrixSelection
	var sourceSelectionsAll []MatrixSelection
	if err := db.Where("revision_id = ?", sourceRevisionID).Find(&sourceSelectionsAll).Error; err != nil {
		return nil, fmt.Errorf("讀取 source MatrixSelection 失敗: %w", err)
	}

	// 建立哪些 model 有 selection 的快速查找 set
	modelHasSelection := make(map[int64]bool, len(sourceSelectionsAll))
	for _, sel := range sourceSelectionsAll {
		modelHasSelection[sel.ModelID] = true
	}

	// 計算 Source 有效 model 數量（qty > 0 或有 selection 記錄）
	for _, m := range sourceModels {
		if m.Qty > 0 || modelHasSelection[m.ID] {
			stats.SourceModelCount++
		}
	}

	if lg != nil {
		lg.Debug(fmt.Sprintf("[ImportMatrix] Source Revision ID=%d: 共 %d 個有效 Model，%d 筆 Selection",
			sourceRevisionID, stats.SourceModelCount, len(sourceSelectionsAll)))
	}

	// ─── Step 2：覆蓋清除 Target 的舊 MatrixSelection 與 MatrixModel ─────────
	if err := db.Where("revision_id = ?", targetRevisionID).Delete(&MatrixSelection{}).Error; err != nil {
		return nil, fmt.Errorf("清除 target MatrixSelection 失敗: %w", err)
	}
	if err := db.Where("revision_id = ?", targetRevisionID).Delete(&MatrixModel{}).Error; err != nil {
		return nil, fmt.Errorf("清除 target MatrixModel 失敗: %w", err)
	}

	// ─── Step 3：若 Source 無有效 model，直接結束 ────────────────────────────
	if stats.SourceModelCount == 0 {
		if lg != nil {
			lg.Debug(fmt.Sprintf("[ImportMatrix] Source Revision ID=%d 無有效 Model，跳過匯入", sourceRevisionID))
		}
		return stats, nil
	}

	// ─── Step 4：複製 MatrixModel 結構至 Target（維持 SortOrder）────────────
	sourceModelIDToTargetID := make(map[int64]int64, len(sourceModels))

	for _, sm := range sourceModels {
		newModel := MatrixModel{
			RevisionID: targetRevisionID,
			SortOrder:  sm.SortOrder,
			ModelName:  sm.ModelName,
			Qty:        sm.Qty,
		}
		if err := db.Create(&newModel).Error; err != nil {
			return nil, fmt.Errorf("建立 target MatrixModel (SortOrder=%d) 失敗: %w", sm.SortOrder, err)
		}
		sourceModelIDToTargetID[sm.ID] = newModel.ID
	}

	if lg != nil {
		lg.Debug(fmt.Sprintf("[ImportMatrix] 已複製 %d 個 MatrixModel 至 Target Revision ID=%d",
			len(sourceModels), targetRevisionID))
	}

	// ─── Step 5：以 Target 的 RevisionComponent 建立快速查找 map ─────────────
	var targetComponents []RevisionComponent
	if err := db.Where("revision_id = ?", targetRevisionID).Find(&targetComponents).Error; err != nil {
		return nil, fmt.Errorf("讀取 target RevisionComponent 清單失敗: %w", err)
	}

	// targetMainCompMap: MainMaterialID -> RevisionComponent.ID（主料 ComponentID）
	targetMainCompMap := make(map[int64]int64, len(targetComponents))
	// targetComponentMaterials: 該 Revision 擁有的所有 MaterialID 集合（主料與替代料）
	targetComponentMaterials := make(map[int64]bool, len(targetComponents))

	for _, tc := range targetComponents {
		targetComponentMaterials[tc.MaterialID] = true
		if tc.Role == "M" {
			targetMainCompMap[tc.MaterialID] = tc.ID
		}
	}

	// ─── Step 6：逐筆比對 Source Selection 並複製至 Target ───────────────────
	selectionsToCreate := make([]MatrixSelection, 0, len(sourceSelectionsAll))

	for _, sourceSel := range sourceSelectionsAll {
		// 6a：以 Source ModelID 查出對應的 Target Model ID（依 SortOrder 對應）
		targetModelID, exists := sourceModelIDToTargetID[sourceSel.ModelID]
		if !exists {
			continue
		}

		// 6b：確認 Target 有對應的主料（Role="M" 且 MaterialID 相符）
		targetMainCompID, mainExists := targetMainCompMap[sourceSel.MainMaterialID]
		if !mainExists {
			stats.IgnoredMainParts++
			if lg != nil {
				lg.Debug(fmt.Sprintf("[ImportMatrix] Target 不存在主料 MaterialID=%d，略過（Source ModelID=%d）",
					sourceSel.MainMaterialID, sourceSel.ModelID))
			}
			continue
		}

		// 6c：確認被選中物料 (SelectedMaterialID) 在 Target 中存在
		if !targetComponentMaterials[sourceSel.SelectedMaterialID] {
			stats.IgnoredSecondParts++
			if lg != nil {
				lg.Debug(fmt.Sprintf("[ImportMatrix] Target 不存在選中物料 MaterialID=%d（主料 MaterialID=%d 存在），略過",
					sourceSel.SelectedMaterialID, sourceSel.MainMaterialID))
			}
			continue
		}

		// 6d：建立新的 MatrixSelection
		selectionsToCreate = append(selectionsToCreate, MatrixSelection{
			RevisionID:         targetRevisionID,
			ModelID:            targetModelID,
			ComponentID:        targetMainCompID,
			MainMaterialID:     sourceSel.MainMaterialID,
			SelectedMaterialID: sourceSel.SelectedMaterialID,
			IsAutoSelected:     true,
		})
	}

	// ─── Step 7：批次寫入新 Selection ────────────────────────────────────────
	if len(selectionsToCreate) > 0 {
		if err := db.CreateInBatches(&selectionsToCreate, 100).Error; err != nil {
			return nil, fmt.Errorf("批次建立 target MatrixSelection 失敗: %w", err)
		}
	}
	stats.CopiedSelectionsCount = len(selectionsToCreate)

	return stats, nil
}
