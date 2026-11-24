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

// GetMatrixModels returns all matrix models for a revision (ordered by sort_order ASC)
func GetMatrixModels(db *gorm.DB, revisionID int64) ([]MatrixModel, error) {
	var models []MatrixModel
	if err := db.Where("revision_id = ?", revisionID).Order("sort_order ASC").Preload("Selections").Find(&models).Error; err != nil {
		return nil, err
	}
	return models, nil
}

// CountActiveMatrixModelsByRevisions 批次計算多個 BOM Revision 實際有效（Active）的 MatrixModel 數量。
//
// 有效（Active）Model 判定規則：
//   - IsHidden == false
//   - 且 (Qty > 0 或存在至少一筆 MatrixSelection 勾選紀錄)
//
// 參數：
//   - db: GORM 資料庫實例
//   - revisionIDs: 要查詢的 BOM Revision ID 清單
//
// 回傳：
//   - map[int64]int: RevisionID -> 有效 Model 數量
//   - error: 若查詢失敗則回傳錯誤
func CountActiveMatrixModelsByRevisions(db *gorm.DB, revisionIDs []int64) (map[int64]int, error) {
	counts := make(map[int64]int, len(revisionIDs))
	if len(revisionIDs) == 0 {
		return counts, nil
	}

	// 1. 查詢所有未隱藏的 MatrixModel
	var models []MatrixModel
	if err := db.Where("revision_id IN ? AND is_hidden = ?", revisionIDs, false).Find(&models).Error; err != nil {
		return nil, fmt.Errorf("查詢未隱藏 MatrixModel 失敗: %w", err)
	}
	if len(models) == 0 {
		return counts, nil
	}

	// 2. 查詢這些 Revision 中所有存在勾選紀錄的 model_id 集合
	var modelIDsWithSel []int64
	if err := db.Model(&MatrixSelection{}).
		Where("revision_id IN ?", revisionIDs).
		Distinct("model_id").
		Pluck("model_id", &modelIDsWithSel).Error; err != nil {
		return nil, fmt.Errorf("查詢 MatrixSelection 關聯 ModelID 失敗: %w", err)
	}

	hasSelMap := make(map[int64]bool, len(modelIDsWithSel))
	for _, mid := range modelIDsWithSel {
		hasSelMap[mid] = true
	}

	// 3. 統計各 Revision 中符合 Active 條件（!IsHidden && (Qty > 0 || hasSelection)）的 Model 數
	for _, m := range models {
		if !m.IsHidden && (m.Qty > 0 || hasSelMap[m.ID]) {
			counts[m.RevisionID]++
		}
	}

	return counts, nil
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
		// 若 modelID == 0，優先尋找未隱藏的第一個 MatrixModel；若不存在則建立預設 Model A
		if modelID == 0 {
			var m MatrixModel
			if err := tx.Where("revision_id = ? AND is_hidden = ?", revisionID, false).Order("sort_order ASC").First(&m).Error; err == nil {
				modelID = m.ID
			} else if err2 := tx.Where("revision_id = ?", revisionID).Order("sort_order ASC").First(&m).Error; err2 == nil {
				modelID = m.ID
			} else {
				m = MatrixModel{
					RevisionID: revisionID,
					SortOrder:  0,
					ModelName:  "Default",
					Qty:        1,
					IsHidden:   false,
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
// 若該 MatrixModel 原本處於隱藏狀態且使用者新增勾選，則自動解除隱藏。
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
			// 找到既有之 MatrixModel；若原本隱藏且本次執行勾選，則自動啟用
			if m.IsHidden && selectedMaterialID > 0 {
				if err := tx.Model(&MatrixModel{}).Where("id = ?", m.ID).Update("is_hidden", false).Error; err != nil {
					return fmt.Errorf("解除隱藏 MatrixModel (ID=%d) 失敗: %w", m.ID, err)
				}
			}
		} else {
			// 不存在則依據 sortOrder 自動建立
			alias := GetModelOrderAlias(sortOrder)
			m = MatrixModel{
				RevisionID: revisionID,
				SortOrder:  sortOrder,
				ModelName:  fmt.Sprintf("Model %s", alias),
				Qty:        1,
				IsHidden:   false,
			}
			if err := tx.Create(&m).Error; err != nil {
				return fmt.Errorf("建立 MatrixModel (SortOrder=%d) 失敗: %w", sortOrder, err)
			}
		}

		return upsertMatrixSelectionWithTx(tx, revisionID, m.ID, mainMaterialID, selectedMaterialID)
	})
}

// GetModelOrderAlias 將 0-based 索引轉為字母代號 (0->A, 1->B, 25->Z, 26->AA)。
//
// 參數：
//   - index: 0-based Model 排序索引
//
// 回傳：
//   - string: 對應的大寫字母代號
func GetModelOrderAlias(index int) string {
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

// matrixMaterialPair 用於快取驗證目標版本中合法的 (主料 MaterialID, 選中物料 MaterialID) 組合
type matrixMaterialPair struct {
	mainID int64
	selID  int64
}

// buildTargetAllowedSelections 讀取目標版本的 RevisionComponent 清單，
// 建立主料對應表與合法可選物料組合表（僅允許主料自身與其直屬 Role="S" 替代料）。
//
// 參數：
//   - tx: GORM 交易或資料庫實例
//   - targetRevisionID: 目標 BOM Revision ID
//
// 回傳：
//   - map[int64]int64: MainMaterialID -> 主料 RevisionComponent.ID
//   - map[matrixMaterialPair]bool: 合法的 (MainMaterialID, SelectedMaterialID) 對照表
//   - error: 若查詢 RevisionComponent 失敗則回傳錯誤
func buildTargetAllowedSelections(tx *gorm.DB, targetRevisionID int64) (map[int64]int64, map[matrixMaterialPair]bool, error) {
	var targetComponents []RevisionComponent
	if err := tx.Where("revision_id = ?", targetRevisionID).Find(&targetComponents).Error; err != nil {
		return nil, nil, fmt.Errorf("讀取 target RevisionComponent 清單失敗: %w", err)
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
	targetAllowedSelections := make(map[matrixMaterialPair]bool, len(targetComponents)*2)
	for _, tc := range targetComponents {
		if tc.Role == "M" {
			targetAllowedSelections[matrixMaterialPair{mainID: tc.MaterialID, selID: tc.MaterialID}] = true
		} else if tc.Role == "S" && tc.ParentComponentID > 0 {
			if parentComp, ok := targetCompByID[tc.ParentComponentID]; ok {
				targetAllowedSelections[matrixMaterialPair{mainID: parentComp.MaterialID, selID: tc.MaterialID}] = true
			}
		}
	}

	return targetMainCompMap, targetAllowedSelections, nil
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

	// 建立哪些 model 有 selection 的快速查找 set，計算有效（未隱藏）model 數
	modelHasSelection := make(map[int64]bool, len(sourceSelectionsAll))
	for _, sel := range sourceSelectionsAll {
		modelHasSelection[sel.ModelID] = true
	}
	for _, m := range sourceModels {
		if !m.IsHidden && (m.Qty > 0 || modelHasSelection[m.ID]) {
			stats.SourceModelCount++
		}
	}
	// 若無未隱藏且有資料的 model，但 sourceModels 存在，則以總 model 數作為統計
	if stats.SourceModelCount == 0 {
		stats.SourceModelCount = len(sourceModels)
	}

	if lg != nil {
		lg.Debug(fmt.Sprintf("[ImportMatrix] Source Revision ID=%d: 共 %d 個有效 Model，%d 筆 Selection",
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

		// ─── Step 4：複製 MatrixModel 結構至 Target（維持 SortOrder 與 IsHidden）───
		sourceModelIDToTargetID := make(map[int64]int64, len(sourceModels))
		for _, sm := range sourceModels {
			newModel := MatrixModel{
				RevisionID: targetRevisionID,
				SortOrder:  sm.SortOrder,
				ModelName:  sm.ModelName,
				Qty:        sm.Qty,
				IsHidden:   sm.IsHidden,
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
		targetMainCompMap, targetAllowedSelections, err := buildTargetAllowedSelections(tx, targetRevisionID)
		if err != nil {
			return err
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
			if !targetAllowedSelections[matrixMaterialPair{mainID: sourceSel.MainMaterialID, selID: sourceSel.SelectedMaterialID}] {
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
	IsHidden  bool   `json:"is_hidden"`
}

// SaveRevisionMatrixModels 批次更新指定 Revision 的 MatrixModel 列表（支援稀疏 SortOrder、新增、修改、啟用與隱藏）。
//
// 行為：
//  1. 查詢該 Revision 當前所有 MatrixModel。
//  2. 根據傳入的 models 清單（依 input.SortOrder 精準對應）：
//     - 若 DB 已存在相同 SortOrder 的 Model：更新 ModelName、Qty 與 IsHidden（保留原有 ID 與其關聯之 Selections，隱藏時亦不刪除舊勾選）。
//     - 若 DB 不存在該 SortOrder：僅當 !input.IsHidden（明確啟用）時建立新的 MatrixModel；若本身隱藏且 DB 無紀錄則不建立空列。
//  3. 若 DB 中存在未包含於傳入清單的既有 Model：
//     - 若該 Model 底下有勾選紀錄（MatrixSelection），將其設為 IsHidden = true 以保留歷史勾選，不予刪除。
//     - 若該 Model 底下無任何勾選紀錄，則清理刪除該空 Model 紀錄。
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
				alias := GetModelOrderAlias(sortOrder)
				modelName = fmt.Sprintf("Model %s", alias)
			}
			qty := input.Qty
			if qty < 0 {
				qty = 0
			}

			if existM, ok := existingByOrder[sortOrder]; ok {
				// 更新現有 model (保留既有 ID 與 Selections，隱藏時亦不刪除勾選)
				if err := tx.Model(&MatrixModel{}).Where("id = ?", existM.ID).Updates(map[string]any{
					"model_name": modelName,
					"qty":        qty,
					"is_hidden":  input.IsHidden,
				}).Error; err != nil {
					return fmt.Errorf("更新 MatrixModel ID=%d 失敗: %w", existM.ID, err)
				}
			} else if !input.IsHidden {
				// 僅對明確啟用（非隱藏）的 SortOrder 建立新紀錄，避免為空白未用機種建立冗餘資料
				newM := MatrixModel{
					RevisionID: revisionID,
					SortOrder:  sortOrder,
					ModelName:  modelName,
					Qty:        qty,
					IsHidden:   false,
				}
				if err := tx.Create(&newM).Error; err != nil {
					return fmt.Errorf("建立 MatrixModel (SortOrder=%d) 失敗: %w", sortOrder, err)
				}
			}
		}

		// 3. 處理未包含在傳入清單中的既有 models：
		//    - 若有歷史 MatrixSelection，改為標記 IsHidden=true 以保留勾選紀錄
		//    - 若無任何 MatrixSelection，則直接刪除空 Model
		var omittedIDs []int64
		for _, m := range existing {
			if !keptOrders[m.SortOrder] {
				omittedIDs = append(omittedIDs, m.ID)
			}
		}

		if len(omittedIDs) > 0 {
			var modelIDsWithSel []int64
			if err := tx.Model(&MatrixSelection{}).
				Where("model_id IN ?", omittedIDs).
				Distinct("model_id").
				Pluck("model_id", &modelIDsWithSel).Error; err != nil {
				return fmt.Errorf("檢查待處理 MatrixModel 勾選紀錄失敗: %w", err)
			}

			hasSel := make(map[int64]bool, len(modelIDsWithSel))
			for _, id := range modelIDsWithSel {
				hasSel[id] = true
			}

			var toHideIDs []int64
			var toDeleteIDs []int64
			for _, id := range omittedIDs {
				if hasSel[id] {
					toHideIDs = append(toHideIDs, id)
				} else {
					toDeleteIDs = append(toDeleteIDs, id)
				}
			}

			if len(toHideIDs) > 0 {
				if err := tx.Model(&MatrixModel{}).Where("id IN ?", toHideIDs).Update("is_hidden", true).Error; err != nil {
					return fmt.Errorf("隱藏保留既有 MatrixModel 失敗: %w", err)
				}
			}
			if len(toDeleteIDs) > 0 {
				if err := tx.Where("id IN ?", toDeleteIDs).Delete(&MatrixModel{}).Error; err != nil {
					return fmt.Errorf("刪除未使用 MatrixModel 失敗: %w", err)
				}
			}
		}

		return nil
	})
}

// MatrixModelSummary 描述單一 BOM Revision 內單個 MatrixModel 的摘要與勾選統計資訊。
type MatrixModelSummary struct {
	ID             int64
	RevisionID     int64
	SortOrder      int
	ModelAlias     string
	ModelName      string
	Qty            int
	IsHidden       bool
	SelectionCount int
}

// GetRevisionMatrixModelSummaries 查詢指定 BOM Revision 下所有 MatrixModel 的摘要資訊
// （包含依據 SortOrder 推算的 ModelAlias 與實際有效勾選筆數 SelectionCount），依 sort_order 升冪排序。
//
// 參數：
//   - database: GORM 資料庫實例
//   - revisionID: BOM Revision ID
//
// 回傳：
//   - []MatrixModelSummary: Model 摘要清單（無機種時回傳空切片）
//   - error: 若資料庫查詢失敗或參數無效則回傳錯誤
func GetRevisionMatrixModelSummaries(database *gorm.DB, revisionID int64) ([]MatrixModelSummary, error) {
	if database == nil {
		return nil, fmt.Errorf("資料庫連線為 nil")
	}
	if revisionID <= 0 {
		return nil, fmt.Errorf("無效的 Revision ID: %d", revisionID)
	}

	var models []MatrixModel
	if err := database.Where("revision_id = ?", revisionID).Order("sort_order ASC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("查詢 Revision ID=%d 的 MatrixModel 失敗: %w", revisionID, err)
	}
	if len(models) == 0 {
		return []MatrixModelSummary{}, nil
	}

	// 以單次聚合查詢統計該 Revision 下各 Model 的有效勾選筆數 (selected_material_id > 0)
	type modelSelectionCountRow struct {
		ModelID int64 `gorm:"column:model_id"`
		Count   int   `gorm:"column:count"`
	}
	var countRows []modelSelectionCountRow
	if err := database.Model(&MatrixSelection{}).
		Select("model_id, COUNT(*) as count").
		Where("revision_id = ? AND selected_material_id > 0", revisionID).
		Group("model_id").
		Scan(&countRows).Error; err != nil {
		return nil, fmt.Errorf("統計 Revision ID=%d 的 MatrixSelection 筆數失敗: %w", revisionID, err)
	}

	countMap := make(map[int64]int, len(countRows))
	for _, row := range countRows {
		countMap[row.ModelID] = row.Count
	}

	summaries := make([]MatrixModelSummary, 0, len(models))
	for _, m := range models {
		summaries = append(summaries, MatrixModelSummary{
			ID:             m.ID,
			RevisionID:     m.RevisionID,
			SortOrder:      m.SortOrder,
			ModelAlias:     GetModelOrderAlias(m.SortOrder),
			ModelName:      m.ModelName,
			Qty:            m.Qty,
			IsHidden:       m.IsHidden,
			SelectionCount: countMap[m.ID],
		})
	}

	return summaries, nil
}

// MatrixModelCopyMapping 定義單一組「來源 Model → 目標 Model」的複製對應關係。
type MatrixModelCopyMapping struct {
	SourceModelID int64
	TargetModelID int64
}

// CopyMatrixModelsParams 定義批次 Model-to-Model Matrix 複製的執行參數。
type CopyMatrixModelsParams struct {
	SourceRevisionID int64
	TargetRevisionID int64
	Mappings         []MatrixModelCopyMapping
	CopyQty          bool
}

// CopyMatrixModelPairStat 記錄單組 Model-to-Model 複製的明細執行統計。
type CopyMatrixModelPairStat struct {
	SourceModelID         int64
	SourceModelAlias      string
	TargetModelID         int64
	TargetModelAlias      string
	CopiedSelectionsCount int
	IgnoredMainParts      int
	IgnoredSecondParts    int
	QtyCopied             bool
	CopiedQty             int
}

// CopyMatrixModelsStats 記錄整批 Model-to-Model Matrix 複製的總體與分組統計數據。
type CopyMatrixModelsStats struct {
	TotalMappings         int
	CopiedSelectionsCount int
	IgnoredMainParts      int
	IgnoredSecondParts    int
	PairStats             []CopyMatrixModelPairStat
}

// validateCopyMatrixModelsParams 驗證批次 Model-to-Model 複製參數，並回傳去重後的來源與目標 Model ID 清單。
func validateCopyMatrixModelsParams(params CopyMatrixModelsParams) ([]int64, []int64, error) {
	if params.SourceRevisionID <= 0 || params.TargetRevisionID <= 0 {
		return nil, nil, fmt.Errorf("無效的版本 ID (source=%d, target=%d)", params.SourceRevisionID, params.TargetRevisionID)
	}
	if len(params.Mappings) == 0 {
		return nil, nil, fmt.Errorf("請至少設定一組來源與目標 Model 對應")
	}

	sourceModelIDs := make([]int64, 0, len(params.Mappings))
	targetModelIDs := make([]int64, 0, len(params.Mappings))
	sourceSeen := make(map[int64]bool, len(params.Mappings))
	targetSeen := make(map[int64]bool, len(params.Mappings))
	isSameRevision := params.SourceRevisionID == params.TargetRevisionID

	for i, m := range params.Mappings {
		if m.SourceModelID <= 0 || m.TargetModelID <= 0 {
			return nil, nil, fmt.Errorf("第 %d 組對應包含無效的 Model ID (source=%d, target=%d)", i+1, m.SourceModelID, m.TargetModelID)
		}
		if isSameRevision && m.SourceModelID == m.TargetModelID {
			return nil, nil, fmt.Errorf("第 %d 組對應在同版本內不可選擇相同的來源與目標 Model (ModelID=%d)", i+1, m.SourceModelID)
		}
		if targetSeen[m.TargetModelID] {
			return nil, nil, fmt.Errorf("目標 Model (ID=%d) 在對應清單中重複指定", m.TargetModelID)
		}
		targetSeen[m.TargetModelID] = true
		targetModelIDs = append(targetModelIDs, m.TargetModelID)

		if !sourceSeen[m.SourceModelID] {
			sourceSeen[m.SourceModelID] = true
			sourceModelIDs = append(sourceModelIDs, m.SourceModelID)
		}
	}

	return sourceModelIDs, targetModelIDs, nil
}

// loadAndValidateModelsMap 在交易中查詢指定 Revision 的指定 Model 清單，
// 確認每筆皆存在於該版本且未處於隱藏狀態 (IsHidden == false)。
func loadAndValidateModelsMap(tx *gorm.DB, revisionID int64, modelIDs []int64, roleLabel string) (map[int64]MatrixModel, error) {
	var models []MatrixModel
	if err := tx.Where("revision_id = ? AND id IN ?", revisionID, modelIDs).Find(&models).Error; err != nil {
		return nil, fmt.Errorf("讀取%s MatrixModel 失敗: %w", roleLabel, err)
	}

	modelMap := make(map[int64]MatrixModel, len(models))
	for _, m := range models {
		modelMap[m.ID] = m
	}

	for _, id := range modelIDs {
		m, exists := modelMap[id]
		if !exists {
			return nil, fmt.Errorf("找不到%s Model (ID=%d) 或其不屬於版本 (RevisionID=%d)", roleLabel, id, revisionID)
		}
		if m.IsHidden {
			return nil, fmt.Errorf("%s Model %s (ID=%d) 目前為隱藏狀態，無法複製", roleLabel, GetModelOrderAlias(m.SortOrder), id)
		}
	}

	return modelMap, nil
}

// CopyMatrixModelSelections 以單一交易執行「Model-to-Model」精準覆蓋複製。
//
// 設計原則：
//   - 交易保證：所有驗證、清除、Qty 覆寫與批次 Selection 寫入均在同一 GORM Transaction 內完成。
//   - 記憶體快照：在任何目標刪除或更新之前，先將所有來源 Model（含 Qty 與 MatrixSelection）讀取至記憶體快照，
//     因此支援同版本內的連鎖複製（如 A → B 與 B → C）及互換複製（如 A → B 與 B → A）而不受寫入順序影響。
//   - 僅覆蓋指定目標：僅清空 params.Mappings 所指定目標 Model 的現有 MatrixSelection；目標版本其餘未指定 Model 完整保留。
//   - 僅限既有且未隱藏 Model：不自動建立新 Model，且禁止存取 IsHidden == true 的機種。
//   - 可選 Qty 覆寫：僅當 params.CopyQty == true 時以明確欄位更新目標 Qty（支援覆寫為 0），絕不更動目標 ModelName。
//   - 階層式比對：驗證目標版本存在對應主料 (Role="M") 且選中物料為該主料自身或其直屬 2nd 替代料，寫入 IsAutoSelected=true。
//
// 參數：
//   - database: GORM 資料庫實例
//   - params: Model-to-Model 複製參數（包含來源／目標 Revision ID、Mappings 與 CopyQty）
//   - lg: MatrixLogger 介面實例（可為 nil）
//
// 回傳：
//   - *CopyMatrixModelsStats: 複製總計與每組對應明細統計
//   - error: 若驗證失敗或資料庫操作失敗則回傳錯誤（已自動 Rollback）
func CopyMatrixModelSelections(database *gorm.DB, params CopyMatrixModelsParams, lg MatrixLogger) (*CopyMatrixModelsStats, error) {
	if database == nil {
		return nil, fmt.Errorf("資料庫連線為 nil")
	}

	sourceModelIDs, targetModelIDs, err := validateCopyMatrixModelsParams(params)
	if err != nil {
		return nil, err
	}

	stats := &CopyMatrixModelsStats{
		TotalMappings: len(params.Mappings),
		PairStats:     make([]CopyMatrixModelPairStat, 0, len(params.Mappings)),
	}

	err = database.Transaction(func(tx *gorm.DB) error {
		// 1. 驗證並快照所有來源與目標 MatrixModel（確保存在於對應 Revision 且未隱藏）
		srcModelMap, err := loadAndValidateModelsMap(tx, params.SourceRevisionID, sourceModelIDs, "來源")
		if err != nil {
			return err
		}
		tgtModelMap, err := loadAndValidateModelsMap(tx, params.TargetRevisionID, targetModelIDs, "目標")
		if err != nil {
			return err
		}

		// 2. 快照讀取：於任何刪除／更新前一次讀出所有來源 Model 的有效 MatrixSelection
		var sourceSelectionsAll []MatrixSelection
		if err := tx.Where("revision_id = ? AND model_id IN ? AND selected_material_id > 0", params.SourceRevisionID, sourceModelIDs).
			Find(&sourceSelectionsAll).Error; err != nil {
			return fmt.Errorf("讀取來源 Model MatrixSelection 失敗: %w", err)
		}

		sourceSelectionsByModelID := make(map[int64][]MatrixSelection, len(sourceModelIDs))
		for _, sel := range sourceSelectionsAll {
			sourceSelectionsByModelID[sel.ModelID] = append(sourceSelectionsByModelID[sel.ModelID], sel)
		}

		// 3. 建立目標版本的主料與合法替代料對照表
		targetMainCompMap, targetAllowedSelections, err := buildTargetAllowedSelections(tx, params.TargetRevisionID)
		if err != nil {
			return err
		}

		// 4. 精準清空指定目標 Model 的既有 MatrixSelection（保留目標版本其餘未指定 Model）
		if err := tx.Where("revision_id = ? AND model_id IN ?", params.TargetRevisionID, targetModelIDs).
			Delete(&MatrixSelection{}).Error; err != nil {
			return fmt.Errorf("清除目標 Model MatrixSelection 失敗: %w", err)
		}

		// 5. 預估新寫入切片容量，避免迴圈擴容開銷
		estimatedCap := 0
		for _, mapping := range params.Mappings {
			estimatedCap += len(sourceSelectionsByModelID[mapping.SourceModelID])
		}
		selectionsToCreate := make([]MatrixSelection, 0, estimatedCap)

		// 6. 逐組依據記憶體快照處理 Qty 與 Selection 映射
		for _, mapping := range params.Mappings {
			srcModel := srcModelMap[mapping.SourceModelID]
			tgtModel := tgtModelMap[mapping.TargetModelID]
			srcAlias := GetModelOrderAlias(srcModel.SortOrder)
			tgtAlias := GetModelOrderAlias(tgtModel.SortOrder)

			pairStat := CopyMatrixModelPairStat{
				SourceModelID:    srcModel.ID,
				SourceModelAlias: srcAlias,
				TargetModelID:    tgtModel.ID,
				TargetModelAlias: tgtAlias,
			}

			// 若啟用「同時複製打件數量」，使用 map 更新以支援將 Qty 覆寫為 0，且不動 ModelName
			if params.CopyQty {
				if err := tx.Model(&MatrixModel{}).
					Where("id = ? AND revision_id = ?", tgtModel.ID, params.TargetRevisionID).
					Updates(map[string]any{"qty": srcModel.Qty}).Error; err != nil {
					return fmt.Errorf("更新目標 Model %s (ID=%d) Qty 失敗: %w", tgtAlias, tgtModel.ID, err)
				}
				pairStat.QtyCopied = true
				pairStat.CopiedQty = srcModel.Qty
			}

			srcSelections := sourceSelectionsByModelID[srcModel.ID]
			for _, sourceSel := range srcSelections {
				targetMainCompID, mainExists := targetMainCompMap[sourceSel.MainMaterialID]
				if !mainExists {
					pairStat.IgnoredMainParts++
					if lg != nil {
						lg.Debug(fmt.Sprintf("[CopyMatrixModels] Model %s→%s: 目標不存在主料 MaterialID=%d，略過",
							srcAlias, tgtAlias, sourceSel.MainMaterialID))
					}
					continue
				}

				if !targetAllowedSelections[matrixMaterialPair{mainID: sourceSel.MainMaterialID, selID: sourceSel.SelectedMaterialID}] {
					pairStat.IgnoredSecondParts++
					if lg != nil {
						lg.Debug(fmt.Sprintf("[CopyMatrixModels] Model %s→%s: 目標主料 MaterialID=%d 下不存在合法替代料 MaterialID=%d，略過",
							srcAlias, tgtAlias, sourceSel.MainMaterialID, sourceSel.SelectedMaterialID))
					}
					continue
				}

				selectionsToCreate = append(selectionsToCreate, MatrixSelection{
					RevisionID:         params.TargetRevisionID,
					ModelID:            tgtModel.ID,
					ComponentID:        targetMainCompID,
					MainMaterialID:     sourceSel.MainMaterialID,
					SelectedMaterialID: sourceSel.SelectedMaterialID,
					IsAutoSelected:     true,
				})
				pairStat.CopiedSelectionsCount++
			}

			stats.CopiedSelectionsCount += pairStat.CopiedSelectionsCount
			stats.IgnoredMainParts += pairStat.IgnoredMainParts
			stats.IgnoredSecondParts += pairStat.IgnoredSecondParts
			stats.PairStats = append(stats.PairStats, pairStat)
		}

		// 7. 批次寫入新的 Selection 紀錄
		if len(selectionsToCreate) > 0 {
			if err := tx.CreateInBatches(&selectionsToCreate, 100).Error; err != nil {
				return fmt.Errorf("批次建立目標 Model MatrixSelection 失敗: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return stats, nil
}
