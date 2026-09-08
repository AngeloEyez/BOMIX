package excel

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"bomix-app/backend/db"
	"bomix-app/backend/logger"
	"bomix-app/backend/types"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// validModelInfo 暫存解析到的有效 Model 資訊
type validModelInfo struct {
	ColIndex  int    // Excel 欄位索引 (K=10 ... Q=16)
	SortOrder int    // 0-based 排序索引 (0, 1, 2...)
	ModelName string // Model 名稱
	Qty       int    // 打件數量
}

// MatrixReader handles Matrix format import
// See product-spec section 7.3
type MatrixReader struct {
	db         *gorm.DB
	result     *types.ImportResult
	logger     *logger.Logger
	progressCb func(progress float64, message string)
}

// NewMatrixReader creates a new MatrixReader instance
func NewMatrixReader(db *gorm.DB, result *types.ImportResult, logger *logger.Logger) *MatrixReader {
	return &MatrixReader{
		db:     db,
		result: result,
		logger: logger,
	}
}

// Import imports a Matrix format file
// 僅更新資料庫中的 model qty 與 matrix selections，不動更零件基礎物料檔。
func (r *MatrixReader) Import(f Workbook) error {
	sheets := f.GetSheetList()

	// ─── 步驟 1：找尋 SMD 頁面並解析表頭 ──────────────────────────────────────────
	smdSheet := r.findSheetCaseInsensitive(sheets, "SMD")
	if smdSheet == "" {
		return errors.New("SMD sheet not found")
	}

	phase, version, description, schematicVersion, pcbVersion, pcaPn, date, projectCode, err := r.parseHeader(f, smdSheet)
	if err != nil {
		return fmt.Errorf("failed to parse matrix header: %w", err)
	}

	if r.logger != nil {
		r.logger.Info("[Matrix] 表頭解析完成",
			"projectCode", projectCode,
			"phase", phase,
			"version", version,
			"description", description,
			"schematicVersion", schematicVersion,
			"pcbVersion", pcbVersion,
			"pcaPn", pcaPn,
			"date", date,
		)
	}

	// ─── 步驟 2：解析 Model 數量與 Model Qty ──────────────────────────────────────
	validModels, err := r.parseValidModels(f, smdSheet)
	if err != nil {
		return fmt.Errorf("failed to parse matrix models: %w", err)
	}

	if r.logger != nil {
		r.logger.Info("[Matrix] Model 解析完成", "validModelCount", len(validModels))
		for _, m := range validModels {
			r.logger.Debug("[Matrix] Model 資訊", "name", m.ModelName, "qty", m.Qty, "colIndex", m.ColIndex)
		}
	}

	// ─── 步驟 3：檢查資料庫中是否存在對應的 BomRevision ──────────────────────────
	var project db.Project
	err = r.db.Where("code = ?", projectCode).First(&project).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			errMsg := fmt.Sprintf("warning: BOM revision for project '%s', phase '%s', version '%s' does not exist", projectCode, phase, version)
			if r.logger != nil {
				r.logger.Warn("[Matrix] 找不到對應的 BOM Revision，終止匯入",
					"projectCode", projectCode,
					"phase", phase,
					"version", version,
				)
			}
			if r.result != nil {
				r.result.Errors = append(r.result.Errors, errMsg)
			}
			return nil
		}
		return fmt.Errorf("查詢 Project 失敗: %w", err)
	}

	var revision db.BomRevision
	err = r.db.Where("project_id = ? AND phase = ? AND version = ?", project.ID, phase, version).First(&revision).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			errMsg := fmt.Sprintf("warning: BOM revision for project '%s', phase '%s', version '%s' does not exist", projectCode, phase, version)
			if r.logger != nil {
				r.logger.Warn("[Matrix] 找不到對應的 BOM Revision，終止匯入",
					"projectCode", projectCode,
					"phase", phase,
					"version", version,
				)
			}
			if r.result != nil {
				r.result.Errors = append(r.result.Errors, errMsg)
			}
			return nil
		}
		return fmt.Errorf("查詢 BomRevision 失敗: %w", err)
	}

	if r.logger != nil {
		r.logger.Info("[Matrix] 找到對應 BOM Revision，準備進行更新", "revisionID", revision.ID)
	}

	// ─── 步驟 4：在 Transaction 中更新/重建 MatrixModel 並清空舊 Selection ──────────
	var modelMap map[int]int64 // ColIndex -> MatrixModel.ID
	err = r.db.Transaction(func(tx *gorm.DB) error {
		// 清空舊的 MatrixSelection
		if err := tx.Where("revision_id = ?", revision.ID).Delete(&db.MatrixSelection{}).Error; err != nil {
			return fmt.Errorf("清空舊 MatrixSelection 失敗: %w", err)
		}

		// 清空舊的 MatrixModel
		if err := tx.Where("revision_id = ?", revision.ID).Delete(&db.MatrixModel{}).Error; err != nil {
			return fmt.Errorf("清空舊 MatrixModel 失敗: %w", err)
		}

		// 建立新的 MatrixModel
		modelMap = make(map[int]int64, len(validModels))
		for _, vm := range validModels {
			modelObj := db.MatrixModel{
				RevisionID: revision.ID,
				SortOrder:  vm.SortOrder,
				ModelName:  vm.ModelName,
				Qty:        vm.Qty,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}
			if err := tx.Create(&modelObj).Error; err != nil {
				return fmt.Errorf("建立 MatrixModel 失敗: %w", err)
			}
			modelMap[vm.ColIndex] = modelObj.ID
		}
		return nil
	})
	if err != nil {
		return err
	}

	if r.logger != nil {
		r.logger.Info("[Matrix] 已成功清空舊 Selection 並重新建立 MatrixModel",
			"revisionID", revision.ID,
			"modelCount", len(modelMap),
		)
	}

	// ─── 步驟 5：讀取該 Revision 的 RevisionComponent 與關聯 Material ────────────
	var components []db.RevisionComponent
	if err := r.db.Where("revision_id = ?", revision.ID).Find(&components).Error; err != nil {
		return fmt.Errorf("載入 RevisionComponent 失敗: %w", err)
	}

	matIDs := make([]int64, 0, len(components))
	for _, c := range components {
		matIDs = append(matIDs, c.MaterialID)
	}

	matMap, err := db.GetMaterialsMapByIDs(r.db, matIDs)
	if err != nil {
		return fmt.Errorf("載入 Material 失敗: %w", err)
	}

	type compMatchInfo struct {
		ComponentID int64
		MaterialID  int64
		Role        string
	}

	compBySupplierPN := make(map[string]compMatchInfo, len(components))
	for _, c := range components {
		if m, ok := matMap[c.MaterialID]; ok {
			key := fmt.Sprintf("%s|%s", strings.TrimSpace(m.Supplier), strings.TrimSpace(m.SupplierPN))
			compBySupplierPN[key] = compMatchInfo{
				ComponentID: c.ID,
				MaterialID:  c.MaterialID,
				Role:        c.Role,
			}
		}
	}

	// ─── 步驟 6：依序讀取 SMD, PTH, BOTTOM 頁面，讀取物料與 Model 勾選 ─────────────
	targetSheets := []string{"SMD", "PTH", "BOTTOM"}
	var selectionsToCreate []db.MatrixSelection
	seenSelections := make(map[string]bool) // Key: modelID|mainMatID|selectedMatID 用於去重

	// 計算全表待處理資料列數並初始化 ProgressTracker
	totalRows := 0
	for _, sReq := range targetSheets {
		if sName := r.findSheetCaseInsensitive(sheets, sReq); sName != "" {
			if rws, err := f.GetRows(sName); err == nil && len(rws) > 5 {
				totalRows += len(rws) - 5
			}
		}
	}
	tracker := NewProgressTracker(totalRows, 20, "正在解析與匯入 Matrix BOM...", r.progressCb)

	if r.logger != nil {
		r.logger.Info("[Matrix] 開始掃描工作表物料勾選",
			"targetSheets", targetSheets,
			"compMapSize", len(compBySupplierPN),
		)
	}

	for _, sheetNameReq := range targetSheets {
		sheetName := r.findSheetCaseInsensitive(sheets, sheetNameReq)
		if sheetName == "" {
			if r.logger != nil {
				r.logger.Debug("[Matrix] 跳過不存在的工作表", "sheet", sheetNameReq)
			}
			continue
		}

		rows, err := f.GetRows(sheetName)
		if err != nil {
			if r.logger != nil {
				r.logger.Warn("[Matrix] 讀取工作表失敗", "sheet", sheetName, "error", err)
			}
			continue
		}

		if r.logger != nil {
			r.logger.Debug("[Matrix] 開始解析工作表", "sheet", sheetName, "totalRows", len(rows))
		}

		var currentMainComp *compMatchInfo

		// 從 Row 6 (Index 5) 開始讀取
		for i := 5; i < len(rows); i++ {
			if tracker != nil {
				tracker.AddRows(1)
			}
			row := rows[i]
			if len(row) == 0 {
				continue
			}

			// A(0)=Item, E(4)=Supplier, F(5)=SupplierPN
			item := strings.TrimSpace(safeGetCol(row, 0))
			supplier := strings.TrimSpace(safeGetCol(row, 4))
			supplierPN := strings.TrimSpace(safeGetCol(row, 5))

			if supplier == "" && supplierPN == "" {
				continue
			}

			key := fmt.Sprintf("%s|%s", supplier, supplierPN)
			var rowMatch compMatchInfo

			if item != "" {
				// 主料列 (Main Source)
				c, exists := compBySupplierPN[key]
				if !exists || c.Role != "M" {
					if r.logger != nil {
						r.logger.Debug("[Matrix] 主料在 DB 中未找到，跳過此物料列",
							"sheet", sheetName,
							"row", i+1,
							"supplier", supplier,
							"supplierPN", supplierPN,
						)
					}
					currentMainComp = nil
					continue
				}
				currentMainComp = &c
				rowMatch = c
			} else {
				// 替代料列 (Second Source)
				if currentMainComp == nil {
					continue
				}
				c, exists := compBySupplierPN[key]
				if !exists {
					continue
				}
				rowMatch = c
			}

			// 檢查各大有效 Model 的勾選欄位
			for _, vm := range validModels {
				mark := safeGetCol(row, vm.ColIndex)
				if strings.EqualFold(mark, "V") {
					modelID, ok := modelMap[vm.ColIndex]
					if !ok {
						continue
					}

					// 唯一性去重鍵 (ModelID, MainMaterialID, SelectedMaterialID)
					dedupKey := fmt.Sprintf("%d|%d|%d", modelID, currentMainComp.MaterialID, rowMatch.MaterialID)
					if seenSelections[dedupKey] {
						continue
					}
					seenSelections[dedupKey] = true

					sel := db.MatrixSelection{
						RevisionID:         revision.ID,
						ModelID:            modelID,
						ComponentID:        currentMainComp.ComponentID,
						MainMaterialID:     currentMainComp.MaterialID,
						SelectedMaterialID: rowMatch.MaterialID,
						IsAutoSelected:     false,
						CreatedAt:          time.Now(),
						UpdatedAt:          time.Now(),
					}
					selectionsToCreate = append(selectionsToCreate, sel)
				}
			}
		}
	}

	// ─── 步驟 7：批次寫入 MatrixSelection 至資料庫 ─────────────────────────────
	if len(selectionsToCreate) > 0 {
		if err := r.db.CreateInBatches(&selectionsToCreate, 500).Error; err != nil {
			return fmt.Errorf("批次寫入 MatrixSelection 失敗: %w", err)
		}
	}

	if r.logger != nil {
		r.logger.Info("[Matrix] Matrix BOM 匯入完成",
			"revisionID", revision.ID,
			"validModels", len(validModels),
			"selectionsCreated", len(selectionsToCreate),
		)
	}

	return nil
}

// findSheetCaseInsensitive 依名稱（不區分大小寫）尋找工作表
func (r *MatrixReader) findSheetCaseInsensitive(sheets []string, name string) string {
	for _, sheet := range sheets {
		if strings.EqualFold(strings.TrimSpace(sheet), name) {
			return sheet
		}
	}
	return ""
}

// parseHeader 解析 Matrix 表頭（從 SMD sheet）
// 讀取位置：
// B3: product code
// B4: description
// D3: schematic version
// F3: PCB version
// F4: PCA PN
// H3: BOM version
// H4: date
// J3: phase
func (r *MatrixReader) parseHeader(f Workbook, sheetName string) (phase, version, description, schematicVersion, pcbVersion, pcaPn, date, projectCode string, err error) {
	valB3, _ := f.GetCellValue(sheetName, "B3")
	projectCode = parseHeaderField(valB3, "Product Code", "Project Code")

	valB4, _ := f.GetCellValue(sheetName, "B4")
	description = parseHeaderField(valB4, "Description")

	valD3, _ := f.GetCellValue(sheetName, "D3")
	schematicVersion = parseHeaderField(valD3, "Schematic Version")

	valF3, _ := f.GetCellValue(sheetName, "F3")
	pcbVersion = parseHeaderField(valF3, "PCB Version")

	valF4, _ := f.GetCellValue(sheetName, "F4")
	pcaPn = parseHeaderField(valF4, "PCA PN")

	valH3, _ := f.GetCellValue(sheetName, "H3")
	version = parseHeaderField(valH3, "BOM Version", "Version")

	valH4, _ := f.GetCellValue(sheetName, "H4")
	date = parseHeaderField(valH4, "Date")

	valJ3, _ := f.GetCellValue(sheetName, "J3")
	phase = parseHeaderField(valJ3, "Phase")

	return phase, version, description, schematicVersion, pcbVersion, pcaPn, date, projectCode, nil
}

// parseValidModels 讀取 Model 總數量與各 Model Qty
// 讀取位置：K4~Q4 為 Model 名稱，K5~Q5 為 Model Qty
// 欄位索引：K=10, L=11, M=12, N=13, O=14, P=15, Q=16
func (r *MatrixReader) parseValidModels(f Workbook, sheetName string) ([]validModelInfo, error) {
	var validModels []validModelInfo

	// 欄位對應（Col Index 10 ~ 16）
	startCol := 10 // K
	endCol := 16   // Q

	for colIdx := startCol; colIdx <= endCol; colIdx++ {
		colName, err := excelize.ColumnNumberToName(colIdx + 1)
		if err != nil {
			continue
		}

		// 讀取 K4~Q4 作為 Model 名稱
		cellName, _ := f.GetCellValue(sheetName, fmt.Sprintf("%s4", colName))
		modelName := strings.TrimSpace(cellName)

		// 讀取 K5~Q5 作為 Model Qty
		cellQty, _ := f.GetCellValue(sheetName, fmt.Sprintf("%s5", colName))
		trimmedQty := strings.TrimSpace(cellQty)

		if trimmedQty == "" {
			continue
		}

		qty, err := strconv.Atoi(trimmedQty)
		if err != nil || qty <= 0 {
			continue
		}

		if modelName == "" {
			// 若 Model 名稱未指定，預設以 "Model X" 命名
			modelName = fmt.Sprintf("Model %d", colIdx-startCol+1)
		}

		validModels = append(validModels, validModelInfo{
			ColIndex:  colIdx,
			SortOrder: colIdx - startCol,
			ModelName: modelName,
			Qty:       qty,
		})
	}

	return validModels, nil
}
