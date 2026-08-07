package excel

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"bomix-app/backend/db"
	"bomix-app/backend/logger"
	"bomix-app/backend/task"
)

// BigMatrixReader handles BigMatrix format import
type BigMatrixReader struct {
	db       *gorm.DB
	logger   *logger.Logger
	warnings []string // 累積所有非致命性警告訊息，供最終回傳 WarningError 使用
}

// Import imports a BigMatrix format file.
// 若任何 BOM Revision 在資料庫中不存在，將記錄 warning 並回傳 task.WarningError，
// 使 Task Manager 將任務最終狀態設為 TaskWarning。
// See product-spec section 7.2
func (r *BigMatrixReader) Import(f Workbook) error {
	sheets := f.GetSheetList()

	// 尋找 BigMatrix 工作表（不分大小寫）
	bigMatrixSheet := ""
	for _, sheet := range sheets {
		if strings.EqualFold(sheet, "BigMatrix") {
			bigMatrixSheet = sheet
			break
		}
	}

	if bigMatrixSheet == "" {
		return errors.New("BigMatrix sheet not found")
	}

	// 解析水平多 BOM 組態（依序處理每個 BOM 欄區段）
	// See product-spec section 7.2.2.1
	bomConfigs, err := r.parseBOMConfigs(f, bigMatrixSheet)
	if err != nil {
		return fmt.Errorf("failed to parse BOM configs: %w", err)
	}

	// 解析零件資料與矩陣勾選狀態
	if err := r.parsePartsAndSelections(f, bigMatrixSheet, bomConfigs); err != nil {
		return fmt.Errorf("failed to parse parts and selections: %w", err)
	}

	// 若有任何 warning（例如找不到 Revision），以 WarningError 形式回傳，
	// Task Manager 會將任務狀態設為 TaskWarning
	if len(r.warnings) > 0 {
		combined := strings.Join(r.warnings, "; ")
		return task.NewWarningError(fmt.Errorf("%s", combined))
	}

	return nil
}

// Returns description, number of BOMs, and date
// See product-spec section 7.2.1
func (r *BigMatrixReader) parseHeader(f Workbook, sheetName string) (description string, bomCount int, date string) {
	// B3: "BOMs: {number}"
	valB3, _ := f.GetCellValue(sheetName, "B3")
	bomStr := parseHeaderField(valB3, "BOMs")
	fmt.Sscanf(bomStr, "%d", &bomCount)

	// B4: "Description: {value}"
	valB4, _ := f.GetCellValue(sheetName, "B4")
	description = parseHeaderField(valB4, "Description")

	// E4: "Date: {value}"
	valE4, _ := f.GetCellValue(sheetName, "E4")
	date = parseHeaderField(valE4, "Date")

	return description, bomCount, date
}

// BOMConfig represents a single BOM configuration within BigMatrix
type BOMConfig struct {
	ProjectCode string
	RevisionID  int64 // Database ID
	Phase       string
	Version     string
	ModelStart  int // Starting column index (0-based, H=7)
	ModelCount  int // Number of models in this BOM
	Models      []ModelConfig // Model configurations
}

// ModelConfig represents a single model configuration
type ModelConfig struct {
	ModelName string
	Qty       int
	Column    int // Column index within the BOM
}

// parseBOMConfigs 解析 BigMatrix 工作表中水平排列的多個 BOM 組態。
// 從 H 欄（0-indexed: 7）開始，向右掃描每個 BOM 區段。
// Row 4 中的模型名稱（預設 A, B, C...）與 Row 5 的 Qty 作為 Model 欄位的解析來源。
// 當 Row 4 再次遇到 "A" 或遇到新的 Project Code/Revision 時，代表進入下一個 BOM 區段。
// 若對應的 BOM Revision 不存在於資料庫，則記錄 warning 並跳過該 BOM。
// See product-spec section 7.2.2.1
func (r *BigMatrixReader) parseBOMConfigs(f Workbook, sheetName string) ([]BOMConfig, error) {
	var configs []BOMConfig

	// 從 H 欄開始（0-indexed: 7）
	startCol := 7

	for {
		// 讀取 Row 2 取得 Project Code；若為空則代表已無更多 BOM
		valProj, _ := f.GetCellValue(sheetName, colToCell(startCol, 2))
		if strings.TrimSpace(valProj) == "" {
			break
		}
		projectCode := parseHeaderField(valProj, "Product Code", "Project Code")

		// 讀取 Row 3 取得 Revision（如 "PV-0.3"），解析 Phase 與 Version
		revisionStr, _ := f.GetCellValue(sheetName, colToCell(startCol, 3))
		var phase, version string
		if idx := strings.Index(revisionStr, "-"); idx != -1 {
			phase = strings.TrimSpace(revisionStr[:idx])
			version = strings.TrimSpace(revisionStr[idx+1:])
		} else {
			phase = strings.TrimSpace(revisionStr)
			version = "0.1"
		}

		if r.logger != nil {
			r.logger.Debug("[BigMatrix 讀取] 解析 BOM 組態",
				"projectCode", projectCode,
				"phase", phase,
				"version", version,
			)
		}

		// 掃描以判斷此 BOM 區段包含幾個 Model 欄位。
		var models []ModelConfig
		colIdx := startCol
		for {
			// 若非該 BOM 的第一欄，檢查是否遇到下一個 BOM 區段的開頭。
			if colIdx > startCol {
				nextModelName, _ := f.GetCellValue(sheetName, colToCell(colIdx, 4))
				nextModelName = strings.TrimSpace(nextModelName)

				nextProjVal, _ := f.GetCellValue(sheetName, colToCell(colIdx, 2))
				nextProjCode := parseHeaderField(nextProjVal, "Product Code", "Project Code")

				nextRevStr, _ := f.GetCellValue(sheetName, colToCell(colIdx, 3))
				var nextPhase, nextVersion string
				if idx := strings.Index(nextRevStr, "-"); idx != -1 {
					nextPhase = strings.TrimSpace(nextRevStr[:idx])
					nextVersion = strings.TrimSpace(nextRevStr[idx+1:])
				} else {
					nextPhase = strings.TrimSpace(nextRevStr)
					nextVersion = "0.1"
				}

				// 判斷是否跨入下一個 BOM 區段：
				// 1. Row 4 的名稱為 "A" (全新品號 BOM 開始)
				// 2. 或是出現了與當前 Project Code / Phase / Version 不符的新表頭
				isNextBOM := false
				if strings.EqualFold(nextModelName, "A") {
					isNextBOM = true
				} else if nextProjCode != "" && nextProjCode != projectCode {
					isNextBOM = true
				} else if nextRevStr != "" && (nextPhase != phase || nextVersion != version) {
					isNextBOM = true
				}

				if isNextBOM {
					break
				}
			}

			qtyStr, _ := f.GetCellValue(sheetName, colToCell(colIdx, 5))
			qtyStr = strings.TrimSpace(qtyStr)

			// Row 5 為空白代表此欄不屬於此 BOM（或無數量），跳出
			if qtyStr == "" {
				break
			}

			var qty int
			fmt.Sscanf(qtyStr, "%d", &qty)

			// 優先使用 Row 4 的 Model Name；若空白則按順序產生模型代號（A, B, C...）
			row4Name, _ := f.GetCellValue(sheetName, colToCell(colIdx, 4))
			modelName := strings.TrimSpace(row4Name)
			if modelName == "" {
				modelName = fmt.Sprintf("%c", 'A'+len(models))
			}

			models = append(models, ModelConfig{
				ModelName: modelName,
				Qty:       qty,
				Column:    colIdx - startCol,
			})

			colIdx++

			// 安全上限：單一 BOM 最多 50 個 Model
			if colIdx-startCol > 50 {
				break
			}
		}

		// 若無任何 Model 欄，代表此 BOM 區段不合法，終止掃描
		if len(models) == 0 {
			break
		}

		config := BOMConfig{
			ProjectCode: projectCode,
			Phase:       phase,
			Version:     version,
			ModelStart:  startCol,
			ModelCount:  len(models),
			Models:      models,
		}

		// 查詢資料庫中是否存在對應的 BOM Revision。
		// BigMatrix 匯入不建立新的 Revision，若找不到則記錄 warning 並跳過該 BOM。
		if r.db != nil {
			revisionID, found, err := r.findExistingBOMRevision(config)
			if err != nil {
				return nil, err
			}
			if !found {
				warnMsg := fmt.Sprintf(
					"找不到 BOM Revision，跳過此區段（ProjectCode: %s, Phase: %s, Version: %s）",
					projectCode, phase, version,
				)
				if r.logger != nil {
					r.logger.Warn("[BigMatrix 匯入] " + warnMsg)
				}
				// 累積 warning，不中止匯入流程
				r.warnings = append(r.warnings, warnMsg)
				// 移動到下一個 BOM 區段繼續處理
				startCol = colIdx
				continue
			}
			config.RevisionID = revisionID
		}

		configs = append(configs, config)

		// 移動到下一個 BOM 區段
		startCol = colIdx
	}

	return configs, nil
}

// colToCell converts column index and row number to Excel cell notation
func colToCell(col int, row int) string {
	// Convert 0-indexed column to Excel column letter
	var colStr string
	col++ // Convert to 1-indexed
	for col > 0 {
		col--
		colStr = string(rune('A'+col%26)) + colStr
		col /= 26
	}
	return fmt.Sprintf("%s%d", colStr, row)
}

// findExistingBOMRevision 在資料庫中查詢符合 ProjectCode、Phase、Version 的 BOM Revision。
// BigMatrix 匯入不建立新的 Project 或 Revision，若找不到則回傳 found=false。
// 回傳值：(revisionID int64, found bool, err error)
func (r *BigMatrixReader) findExistingBOMRevision(config BOMConfig) (int64, bool, error) {
	// 先查詢 Project
	var project db.Project
	err := r.db.Where("code = ?", config.ProjectCode).First(&project).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Project 不存在，直接回傳 not found
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("查詢 Project 失敗（code=%s）: %w", config.ProjectCode, err)
	}

	// 再查詢對應的 BOM Revision
	var revision db.BomRevision
	err = r.db.Where("project_id = ? AND phase = ? AND version = ?",
		project.ID, config.Phase, config.Version).First(&revision).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("查詢 BOM Revision 失敗（phase=%s, version=%s）: %w",
			config.Phase, config.Version, err)
	}

	return revision.ID, true, nil
}

// parsePartsAndSelections parses parts and their selections across multiple BOMs
// See product-spec sections 7.2.2.2 and 7.2.2.3
func (r *BigMatrixReader) parsePartsAndSelections(f Workbook, sheetName string, configs []BOMConfig) error {
	// Get all rows from the sheet
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return err
	}

	// revisionModelMap 紀錄每個 revisionID 下 (sortOrder -> MatrixModel.ID) 的映射
	revisionModelMap := make(map[int64]map[int]int64, len(configs))

	// 在 Transaction 中清空舊 Selection/Model，並依據 Excel 欄位資訊重新建立 MatrixModel
	if r.db != nil {
		err = r.db.Transaction(func(tx *gorm.DB) error {
			for _, config := range configs {
				if config.RevisionID == 0 {
					continue
				}
				// 1. 清空舊有 Selection
				if err := tx.Where("revision_id = ?", config.RevisionID).Delete(&db.MatrixSelection{}).Error; err != nil {
					return fmt.Errorf("清空 MatrixSelection 失敗 (revisionID=%d): %w", config.RevisionID, err)
				}
				// 2. 清空舊有 MatrixModel
				if err := tx.Where("revision_id = ?", config.RevisionID).Delete(&db.MatrixModel{}).Error; err != nil {
					return fmt.Errorf("清空 MatrixModel 失敗 (revisionID=%d): %w", config.RevisionID, err)
				}

				// 3. 重建 MatrixModel
				modelIDMap := make(map[int]int64, len(config.Models))
				for idx, model := range config.Models {
					modelObj := db.MatrixModel{
						RevisionID: config.RevisionID,
						SortOrder:  idx,
						ModelName:  model.ModelName,
						Qty:        model.Qty,
					}
					if err := tx.Create(&modelObj).Error; err != nil {
						return fmt.Errorf("建立 MatrixModel 失敗 (revisionID=%d, sortOrder=%d): %w", config.RevisionID, idx, err)
					}
					modelIDMap[idx] = modelObj.ID
				}
				revisionModelMap[config.RevisionID] = modelIDMap
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	var selectionsToCreate []db.MatrixSelection
	seenSelections := make(map[string]bool)

	// 從 Row 6 開始逐行讀取零件與 Model 勾選 (0-indexed 為 5)
	var currentGroupKey string

	for i := 5; i < len(rows); i++ {
		row := rows[i]
		if len(row) == 0 {
			continue
		}

		partData := r.parsePartDataRow(row)
		if partData == nil {
			continue
		}

		// 判斷 Main Source vs 2nd Source 群組
		if partData.item != "" {
			currentGroupKey = partData.supplier + "|" + partData.supplierPN
		} else if currentGroupKey == "" {
			// 若為 2nd Source 但尚未出現過 Main Source，預設使用自身的 supplier|supplierPN
			currentGroupKey = partData.supplier + "|" + partData.supplierPN
		}

		// 對於每個 BOM config 檢查勾選狀態
		for _, config := range configs {
			if config.RevisionID == 0 {
				continue
			}
			modelIDMap, ok := revisionModelMap[config.RevisionID]
			if !ok {
				continue
			}

			// 查詢此 Revision 的既有 Part ID
			partID, err := r.findExistingPart(config.RevisionID, partData.supplier, partData.supplierPN)
			if err != nil {
				// 此 BOM Revision 中未包含此零件，跳過
				continue
			}

			for modelIdx := range config.Models {
				colIdx := config.ModelStart + modelIdx
				cellValue, _ := f.GetCellValue(sheetName, colToCell(colIdx, i+1))

				if strings.EqualFold(strings.TrimSpace(cellValue), "V") {
					matrixModelID, ok := modelIDMap[modelIdx]
					if !ok {
						continue
					}

					materialKey := partData.supplier + "|" + partData.supplierPN
					dedupKey := fmt.Sprintf("%d|%s|%s", matrixModelID, currentGroupKey, materialKey)

					if seenSelections[dedupKey] {
						continue
					}
					seenSelections[dedupKey] = true

					selection := db.MatrixSelection{
						RevisionID:         config.RevisionID,
						ModelID:            matrixModelID,
						PartID:             partID,
						Group:              currentGroupKey,
						Material:           materialKey,
						SelectedSupplier:   partData.supplier,
						SelectedSupplierPn: partData.supplierPN,
						IsAutoSelected:     false,
					}
					selectionsToCreate = append(selectionsToCreate, selection)
				}
			}
		}
	}

	// 批次寫入 MatrixSelections
	if len(selectionsToCreate) > 0 && r.db != nil {
		if err := r.db.Create(&selectionsToCreate).Error; err != nil {
			return fmt.Errorf("批次建立 MatrixSelections 失敗: %w", err)
		}
	}

	return nil
}

// partData represents parsed part information
type partData struct {
	item         string
	hhpn         string
	description  string
	supplier     string
	supplierPN   string
	qty          int
	location     string
}

// parsePartDataRow parses a part data row from BigMatrix
// See product-spec section 7.2.3.1
func (r *BigMatrixReader) parsePartDataRow(row []string) *partData {
	if len(row) == 0 {
		return nil
	}

	data := &partData{}

	// A: Item
	data.item = strings.TrimSpace(row[0])

	// B: HHPN
	if len(row) > 1 {
		data.hhpn = strings.TrimSpace(row[1])
	}

	// C: Description
	if len(row) > 2 {
		data.description = strings.TrimSpace(row[2])
	}

	// D: Supplier
	if len(row) > 3 {
		data.supplier = strings.TrimSpace(row[3])
	}

	// E: Supplier PN
	if len(row) > 4 {
		data.supplierPN = strings.TrimSpace(row[4])
	}

	// F: Qty
	if len(row) > 5 {
		fmt.Sscanf(strings.TrimSpace(row[5]), "%d", &data.qty)
	}

	// G: Location
	if len(row) > 6 {
		data.location = strings.TrimSpace(row[6])
	}

	// 判定是否為零件資料：必須 Supplier 與 Supplier PN 同時存在且不為空白
	if strings.TrimSpace(data.supplier) == "" || strings.TrimSpace(data.supplierPN) == "" {
		return nil
	}

	return data
}

// updateModelQty updates the qty for all models in a BOM revision
func (r *BigMatrixReader) updateModelQty(config BOMConfig) error {
	for idx, model := range config.Models {
		var matrixModel db.MatrixModel
		err := r.db.Where("revision_id = ? AND sort_order = ?", config.RevisionID, idx).
			First(&matrixModel).Error

		if err == nil {
			// Update existing
			matrixModel.ModelName = model.ModelName
			matrixModel.Qty = model.Qty
			if err := r.db.Save(&matrixModel).Error; err != nil {
				return err
			}
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new
			matrixModel = db.MatrixModel{
				RevisionID: config.RevisionID,
				SortOrder:  idx,
				ModelName:  model.ModelName,
				Qty:        model.Qty,
			}
			if err := r.db.Create(&matrixModel).Error; err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return nil
}

// findExistingPart 依 revisionID, supplier, supplierPN 尋找既有 Part。
// BigMatrix 匯入不更新/新增任何 Part 資料，若找不到 Part 則回傳錯誤。
func (r *BigMatrixReader) findExistingPart(revisionID int64, supplier, supplierPN string) (int64, error) {
	var part db.Part
	err := r.db.Where("revision_id = ? AND supplier = ? AND supplier_pn = ?",
		revisionID, supplier, supplierPN).
		First(&part).Error

	if err != nil {
		return 0, err
	}
	return part.ID, nil
}

// findOrCreateMatrixModel finds or creates a MatrixModel in the database
func (r *BigMatrixReader) findOrCreateMatrixModel(revisionID int64, modelName string, qty int) (int64, error) {
	var matrixModel db.MatrixModel
	err := r.db.Where("revision_id = ? AND model_name = ?", revisionID, modelName).
		First(&matrixModel).Error

	if err == nil {
		return matrixModel.ID, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}

	matrixModel = db.MatrixModel{
		RevisionID: revisionID,
		ModelName:  modelName,
		Qty:        qty,
	}

	if err := r.db.Create(&matrixModel).Error; err != nil {
		return 0, err
	}

	return matrixModel.ID, nil
}
