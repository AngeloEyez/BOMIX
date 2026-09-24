package db

import (
	"errors"
	"fmt"
	"strings"

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

// UpsertMatrixSelection 更新或刪除指定 Revision 與 Model 下主料群組的 MatrixSelection。
//
// 行為：
//   - 若 selectedMaterialID == 0，表示取消勾選，刪除該 (revisionID, modelID, mainMaterialID) 的現有 selection。
//   - 若 selectedMaterialID > 0，先刪除該 (revisionID, modelID, mainMaterialID) 的現有 selection，
//     再新增一筆選中紀錄（確保同物料群組在同 Model 下僅單選互斥）。
//   - 若 modelID == 0，將自動搜尋該 Revision 第一個有效的 MatrixModel；若不存在則自動建立預設 Model。
//   - 所有寫入操作均包裹於 GORM Transaction 中以維護資料一致性。
//
// 參數：
//   - db: GORM 資料庫實例
//   - revisionID: BOM Revision ID
//   - modelID: Matrix Model ID（若為 0 則自動查找或建立預設 Model）
//   - mainMaterialID: 主料 Material ID
//   - selectedMaterialID: 被選中的物料 Material ID（0 表示取消勾選）
//
// 回傳：
//   - error: 若資料庫操作失敗則回傳錯誤
func UpsertMatrixSelection(db *gorm.DB, revisionID, modelID, mainMaterialID, selectedMaterialID int64) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// 若 modelID == 0，尋找或建立預設 MatrixModel
		if modelID == 0 {
			var m MatrixModel
			if err := tx.Where("revision_id = ?", revisionID).Order("sort_order ASC").First(&m).Error; err == nil {
				modelID = m.ID
			} else {
				m = MatrixModel{
					RevisionID: revisionID,
					SortOrder:  0,
					ModelName:  "Default",
					Qty:        1,
				}
				if err := tx.Create(&m).Error; err != nil {
					return fmt.Errorf("建立預設 MatrixModel 失敗: %w", err)
				}
				modelID = m.ID
			}
		}

		return upsertMatrixSelectionWithTx(tx, revisionID, modelID, mainMaterialID, selectedMaterialID)
	})
}

// UpsertMatrixModelSelection 依據 Revision ID 與 Model SortOrder 更新 MatrixSelection。
//
// 若指定 (revisionID, sortOrder) 之 MatrixModel 尚不存在，將自動建立對應之 MatrixModel，
// 徹底避免跨 Model 操作時錯位回退至第 0 個 Model 的問題。
//
// 參數：
//   - db: GORM 資料庫實例
//   - revisionID: BOM Revision ID
//   - sortOrder: 0-based Model 排序索引 (0, 1, 2...)
//   - mainMaterialID: 主料 Material ID
//   - selectedMaterialID: 被選中的物料 Material ID（0 表示取消勾選）
//
// 回傳：
//   - error: 若資料庫操作失敗則回傳錯誤
func UpsertMatrixModelSelection(db *gorm.DB, revisionID int64, sortOrder int, mainMaterialID, selectedMaterialID int64) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var m MatrixModel
		if err := tx.Where("revision_id = ? AND sort_order = ?", revisionID, sortOrder).First(&m).Error; err == nil {
			// 找到既有之 MatrixModel
		} else {
			// 不存在則依據 sortOrder 自動建立
			alias := getModelOrderAlias(sortOrder)
			m = MatrixModel{
				RevisionID: revisionID,
				SortOrder:  sortOrder,
				ModelName:  fmt.Sprintf("Model %s", alias),
				Qty:        1,
			}
			if err := tx.Create(&m).Error; err != nil {
				return fmt.Errorf("建立 MatrixModel (SortOrder=%d) 失敗: %w", sortOrder, err)
			}
		}

		return upsertMatrixSelectionWithTx(tx, revisionID, m.ID, mainMaterialID, selectedMaterialID)
	})
}

// getModelOrderAlias 將 0-based 索引轉為字母代號 (0->A, 1->B, 25->Z, 26->AA)
func getModelOrderAlias(index int) string {
	if index < 0 {
		return "A"
	}
	name := ""
	for index >= 0 {
		name = string(rune('A'+(index%26))) + name
		index = index/26 - 1
	}
	return name
}

// upsertMatrixSelectionWithTx 在既有 Transaction 內執行互斥更新 Selection 邏輯
func upsertMatrixSelectionWithTx(tx *gorm.DB, revisionID, modelID, mainMaterialID, selectedMaterialID int64) error {
	// 1. 刪除該 (revisionID, modelID, mainMaterialID) 的舊 selection，實現互斥
	if err := tx.Where("revision_id = ? AND model_id = ? AND main_material_id = ?", revisionID, modelID, mainMaterialID).
		Delete(&MatrixSelection{}).Error; err != nil {
		return fmt.Errorf("清除舊 MatrixSelection 失敗: %w", err)
	}

	// 2. 若 selectedMaterialID == 0，表示取消選取，完成刪除即可結束
	if selectedMaterialID == 0 {
		return nil
	}

	// 3. 查詢主料對應的 RevisionComponent ID
	var comp RevisionComponent
	if err := tx.Where("revision_id = ? AND material_id = ? AND role = ?", revisionID, mainMaterialID, "M").First(&comp).Error; err != nil {
		if errFallback := tx.Where("revision_id = ? AND material_id = ?", revisionID, mainMaterialID).First(&comp).Error; errFallback != nil {
			return fmt.Errorf("找不到主料 RevisionComponent (revisionID=%d, materialID=%d): %w", revisionID, mainMaterialID, errFallback)
		}
	}

	// 4. 新增選取紀錄
	newSel := MatrixSelection{
		RevisionID:         revisionID,
		ModelID:            modelID,
		ComponentID:        comp.ID,
		MainMaterialID:     mainMaterialID,
		SelectedMaterialID: selectedMaterialID,
		IsAutoSelected:     false,
	}
	if err := tx.Create(&newSel).Error; err != nil {
		return fmt.Errorf("建立 MatrixSelection 失敗: %w", err)
	}

	return nil
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
//   - 安全性保證：整體操作包裝於 GORM Transaction 中執行，任一步驟失敗皆自動 Rollback。
//   - 防誤刪保護：若 Source Revision 無任何 MatrixModel，立即返回而不修改 Target Revision。
//   - Model 對應：以 SortOrder（順序索引 0, 1, 2...）嚴格對應建立 Target Model。
//   - 階層式比對：驗證 Target 是否存在該主料 (Role="M")，且 SelectedMaterialID 必須為該主料本身或其合法直屬替代料。
//   - 覆蓋模式：在交易中清空 Target Revision 原有的 MatrixModel 與 MatrixSelection，再寫入新資料。
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

	// ─── Step 2：若 Source 無任何 MatrixModel，不觸碰 Target 資料並直接結束 ──
	if len(sourceModels) == 0 {
		if lg != nil {
			lg.Debug(fmt.Sprintf("[ImportMatrix] Source Revision ID=%d 無 Matrix Model，跳過匯入", sourceRevisionID))
		}
		return stats, nil
	}

	// 建立哪些 model 有 selection 的快速查找 set，計算有效 model 數
	modelHasSelection := make(map[int64]bool, len(sourceSelectionsAll))
	for _, sel := range sourceSelectionsAll {
		modelHasSelection[sel.ModelID] = true
	}
	for _, m := range sourceModels {
		if m.Qty > 0 || modelHasSelection[m.ID] {
			stats.SourceModelCount++
		}
	}
	// 若無 qty>0 且無 selection 的 model，但 sourceModels 存在，則以總 model 數作為統計
	if stats.SourceModelCount == 0 {
		stats.SourceModelCount = len(sourceModels)
	}

	if lg != nil {
		lg.Debug(fmt.Sprintf("[ImportMatrix] Source Revision ID=%d: 共 %d 個 Model，%d 筆 Selection",
			sourceRevisionID, stats.SourceModelCount, len(sourceSelectionsAll)))
	}

	// ─── Step 3~7：包裝於 GORM Transaction 中執行 ──────────────────────────────
	err := db.Transaction(func(tx *gorm.DB) error {
		// ─── Step 3：清空 Target 的舊 MatrixSelection 與 MatrixModel ────────
		if err := tx.Where("revision_id = ?", targetRevisionID).Delete(&MatrixSelection{}).Error; err != nil {
			return fmt.Errorf("清除 target MatrixSelection 失敗: %w", err)
		}
		if err := tx.Where("revision_id = ?", targetRevisionID).Delete(&MatrixModel{}).Error; err != nil {
			return fmt.Errorf("清除 target MatrixModel 失敗: %w", err)
		}

		// ─── Step 4：複製 MatrixModel 結構至 Target（維持 SortOrder）───────
		sourceModelIDToTargetID := make(map[int64]int64, len(sourceModels))
		for _, sm := range sourceModels {
			newModel := MatrixModel{
				RevisionID: targetRevisionID,
				SortOrder:  sm.SortOrder,
				ModelName:  sm.ModelName,
				Qty:        sm.Qty,
			}
			if err := tx.Create(&newModel).Error; err != nil {
				return fmt.Errorf("建立 target MatrixModel (SortOrder=%d) 失敗: %w", sm.SortOrder, err)
			}
			sourceModelIDToTargetID[sm.ID] = newModel.ID
		}

		if lg != nil {
			lg.Debug(fmt.Sprintf("[ImportMatrix] 已複製 %d 個 MatrixModel 至 Target Revision ID=%d",
				len(sourceModels), targetRevisionID))
		}

		// ─── Step 5：以 Target 的 RevisionComponent 建立主料與合法替代料結構 ─
		var targetComponents []RevisionComponent
		if err := tx.Where("revision_id = ?", targetRevisionID).Find(&targetComponents).Error; err != nil {
			return fmt.Errorf("讀取 target RevisionComponent 清單失敗: %w", err)
		}

		// targetMainCompMap: MainMaterialID -> RevisionComponent.ID（主料 ComponentID）
		targetMainCompMap := make(map[int64]int64, len(targetComponents))
		// targetCompByID: Component.ID -> RevisionComponent
		targetCompByID := make(map[int64]RevisionComponent, len(targetComponents))

		for _, tc := range targetComponents {
			targetCompByID[tc.ID] = tc
			if tc.Role == "M" {
				targetMainCompMap[tc.MaterialID] = tc.ID
			}
		}

		// targetAllowedSelections: [MainMaterialID, SelectedMaterialID] -> true
		// 僅允許：1) 主料自身 (M == Selected)；2) 屬於該主料的替代料 (Role="S" 且 ParentComponentID == targetMainCompID)
		type matPair struct {
			mainID int64
			selID  int64
		}
		targetAllowedSelections := make(map[matPair]bool, len(targetComponents)*2)

		for _, tc := range targetComponents {
			if tc.Role == "M" {
				// 主料自身可被選中
				targetAllowedSelections[matPair{mainID: tc.MaterialID, selID: tc.MaterialID}] = true
			} else if tc.Role == "S" && tc.ParentComponentID > 0 {
				// 替代料：查找其父主料的 MaterialID
				if parentComp, ok := targetCompByID[tc.ParentComponentID]; ok {
					targetAllowedSelections[matPair{mainID: parentComp.MaterialID, selID: tc.MaterialID}] = true
				}
			}
		}

		// ─── Step 6：逐筆比對 Source Selection 並複製至 Target ──────────────
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

			// 6c：確認被選中物料 (SelectedMaterialID) 是否為該主料組內的合法選項（主料本身或其直屬替代料）
			if !targetAllowedSelections[matPair{mainID: sourceSel.MainMaterialID, selID: sourceSel.SelectedMaterialID}] {
				stats.IgnoredSecondParts++
				if lg != nil {
					lg.Debug(fmt.Sprintf("[ImportMatrix] Target 主料 MaterialID=%d 下不存在合法替代料 MaterialID=%d，略過",
						sourceSel.MainMaterialID, sourceSel.SelectedMaterialID))
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

		// ─── Step 7：批次寫入新 Selection ────────────────────────────────────
		if len(selectionsToCreate) > 0 {
			if err := tx.CreateInBatches(&selectionsToCreate, 100).Error; err != nil {
				return fmt.Errorf("批次建立 target MatrixSelection 失敗: %w", err)
			}
		}
		stats.CopiedSelectionsCount = len(selectionsToCreate)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return stats, nil
}

// MatrixModelInput 定義更新 Revision Models 時的單一 Model 輸入資料
type MatrixModelInput struct {
	ID        int64  `json:"id"`
	SortOrder int    `json:"sort_order"`
	ModelName string `json:"model_name"`
	Qty       int    `json:"qty"`
}

// SaveRevisionMatrixModels 批次更新指定 Revision 的 MatrixModel 列表（支援新增、修改、刪除多餘 Model）。
//
// 行為：
//  1. 查詢該 Revision 當前所有 MatrixModel。
//  2. 根據傳入的 models 清單（按 SortOrder）：
//     - 若 DB 已存在相同 SortOrder 的 Model：更新 ModelName 與 Qty（保留原有 ID 與其關聯之 Selections）。
//     - 若 DB 不存在該 SortOrder：建立新的 MatrixModel（初始無 Selections）。
//  3. 若 DB 中存在 SortOrder 超過傳入清單的 Model（即使用者減少了 Model 數量）：
//     - 刪除這些多餘的 MatrixModel，其關聯之 MatrixSelection 會一併由外鍵級聯刪除或明確刪除。
//  4. 所有操作於 GORM Transaction 內執行以維護資料一致性。
//
// 參數：
//   - db: GORM 資料庫實例
//   - revisionID: BOM Revision ID
//   - models: 欲設定的 Model 清單
//
// 回傳：
//   - error: 失敗時回傳錯誤
func SaveRevisionMatrixModels(db *gorm.DB, revisionID int64, models []MatrixModelInput) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// 1. 查詢既有 models
		var existing []MatrixModel
		if err := tx.Where("revision_id = ?", revisionID).Find(&existing).Error; err != nil {
			return fmt.Errorf("查詢既有 MatrixModel 失敗 (revisionID=%d): %w", revisionID, err)
		}

		existingByOrder := make(map[int]MatrixModel, len(existing))
		for _, m := range existing {
			existingByOrder[m.SortOrder] = m
		}

		keptOrders := make(map[int]bool, len(models))

		// 2. 處理傳入的 models (更新或新增)
		for idx, input := range models {
			sortOrder := input.SortOrder
			if sortOrder < 0 {
				sortOrder = idx
			}
			keptOrders[sortOrder] = true

			modelName := strings.TrimSpace(input.ModelName)
			if modelName == "" {
				alias := getModelOrderAlias(sortOrder)
				modelName = fmt.Sprintf("Model %s", alias)
			}
			qty := input.Qty
			if qty < 0 {
				qty = 0
			}

			if existM, ok := existingByOrder[sortOrder]; ok {
				// 更新現有 model (保留既有 ID 與 Selections)
				if err := tx.Model(&MatrixModel{}).Where("id = ?", existM.ID).Updates(map[string]any{
					"model_name": modelName,
					"qty":        qty,
				}).Error; err != nil {
					return fmt.Errorf("更新 MatrixModel ID=%d 失敗: %w", existM.ID, err)
				}
			} else {
				// 新增 model
				newM := MatrixModel{
					RevisionID: revisionID,
					SortOrder:  sortOrder,
					ModelName:  modelName,
					Qty:        qty,
				}
				if err := tx.Create(&newM).Error; err != nil {
					return fmt.Errorf("建立 MatrixModel (SortOrder=%d) 失敗: %w", sortOrder, err)
				}
			}
		}

		// 3. 刪除未保留的多餘 models (減少 model 數量時觸發)
		var toDeleteIDs []int64
		for _, m := range existing {
			if !keptOrders[m.SortOrder] {
				toDeleteIDs = append(toDeleteIDs, m.ID)
			}
		}

		if len(toDeleteIDs) > 0 {
			// 先刪除關聯的 MatrixSelection (清空該 model 下的勾選狀態)
			if err := tx.Where("model_id IN ?", toDeleteIDs).Delete(&MatrixSelection{}).Error; err != nil {
				return fmt.Errorf("刪除過期 MatrixSelection 失敗: %w", err)
			}
			// 再刪除 MatrixModel
			if err := tx.Where("id IN ?", toDeleteIDs).Delete(&MatrixModel{}).Error; err != nil {
				return fmt.Errorf("刪除過期 MatrixModel 失敗: %w", err)
			}
		}

		return nil
	})
}
