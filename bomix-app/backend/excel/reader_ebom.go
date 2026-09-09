package excel

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"bomix-app/backend/db"
	"bomix-app/backend/logger"
	"bomix-app/backend/task"
	"bomix-app/backend/types"

	"gorm.io/gorm"
)

// EBOMReader handles EBOM format import
type EBOMReader struct {
	db                 *gorm.DB
	result             *types.ImportResult
	filePath           string // EBOM 匯入檔案路徑
	revisionID         int64  // 匯入後設定
	logger             *logger.Logger
	progressCb         func(progress float64, message string)
	confirmOverwrite   bool
	confirmOverwriteCb func(projectCode, phase, version string) (bool, error)
	warnings           []string // 累積所有非致命性警告訊息，供最終回傳 WarningError 使用
}

// Import 匯入 EBOM 格式 Excel 檔案。
//
// 採用兩階段匯入流程：
//
//	Phase 1（主料建置）：處理 SMD / PTH / BOTTOM / NI / MP sheet，
//	  依 (supplier, supplier_pn) 去重建立 Part，並為每個 location 建立原子化 PartLocation。
//	  （部分專用零件僅存在於 MP sheet，在此階段一併載入避免遺漏）。
//	Phase 2（狀態覆寫）：處理 PROTO / MP / CCL sheet，
//	  僅更新 Phase 1 已建立的 PartLocation 的 BomStatus / CCL 屬性；
//	  同時判斷 BomRevision.Mode（NPI 或 MP）。
func (r *EBOMReader) Import(f Workbook) error {
	sheets := f.GetSheetList()

	// ─── 解析表頭（從 SMD sheet）───────────────────────────────────────────
	smdSheet := r.findSheetCaseInsensitive(sheets, "SMD")
	pthSheet := r.findSheetCaseInsensitive(sheets, "PTH")
	bottomSheet := r.findSheetCaseInsensitive(sheets, "BOTTOM")
	niSheet := r.findSheetCaseInsensitive(sheets, "NI")
	protoSheet := r.findSheetCaseInsensitive(sheets, "PROTO")
	mpSheet := r.findSheetCaseInsensitive(sheets, "MP")
	cclSheet := r.findSheetCaseInsensitive(sheets, "CCL")

	// ─── 計算全表待處理資料列數並初始化 10% 單位 ProgressTracker ───────────────
	allSheetsToCount := []string{smdSheet, pthSheet, bottomSheet, niSheet, mpSheet, protoSheet, mpSheet, cclSheet}
	totalRows := 0
	for _, sName := range allSheetsToCount {
		if sName != "" {
			if rws, err := f.GetRows(sName); err == nil && len(rws) > 5 {
				totalRows += len(rws) - 5
			}
		}
	}
	tracker := NewProgressTracker(totalRows, 20, "正在解析與匯入 EBOM BOM...", r.progressCb)

	var phase, version, description, schematicVersion, pcbVersion, pcaPn, date, projectCode string
	var err error
	if smdSheet != "" {
		phase, version, description, schematicVersion, pcbVersion, pcaPn, date, projectCode, err = r.parseHeader(f, smdSheet)
		if err != nil {
			return fmt.Errorf("failed to parse header: %w", err)
		}
	}

	// ─── 檢查既有 Revision 與覆蓋確認 ─────────────────────────────────────────
	isExisting, err := r.checkRevisionExists(projectCode, phase, version)
	if err != nil {
		return fmt.Errorf("failed to check revision: %w", err)
	}

	if isExisting && r.confirmOverwrite && r.confirmOverwriteCb != nil {
		if r.logger != nil {
			r.logger.Info(fmt.Sprintf("發現既有版本 [%s %s %s]，等待使用者確認是否覆蓋...", projectCode, phase, version),
				"project", projectCode, "phase", phase, "version", version)
		}

		approve, err := r.confirmOverwriteCb(projectCode, phase, version)
		if err != nil {
			return fmt.Errorf("確認覆蓋程序發生錯誤: %w", err)
		}
		if !approve {
			// 使用者選擇「略過」
			if r.logger != nil {
				r.logger.Info(fmt.Sprintf("使用者略過覆蓋既有版本 [%s %s %s]，跳過匯入作業", projectCode, phase, version))
			}
			if r.progressCb != nil {
				r.progressCb(1.0, fmt.Sprintf("已略過匯入 (版本 %s %s 已存在)", phase, version))
			}
			return nil
		}
		if r.logger != nil {
			r.logger.Info(fmt.Sprintf("使用者已確認覆蓋既有版本 [%s %s %s]，繼續執行匯入...", projectCode, phase, version))
		}
	}

	// ─── 建立或更新 BomRevision ─────────────────────────────────────────────
	revisionID, err := r.createOrUpdateRevision(projectCode, phase, version, description, schematicVersion, pcbVersion, pcaPn, date)
	if err != nil {
		return fmt.Errorf("failed to create/update revision: %w", err)
	}
	r.revisionID = revisionID

	// ─── Phase 1：物料收集與 Component 建置 ───────────────────────────────────
	materialsMap := make(map[string]db.Material)
	mainCompMap := make(map[string]*parsedComponentMain)
	var mainCompList []*parsedComponentMain
	var secondCompList []*parsedComponentSecond

	// phase1LocationSheetMap 收集 Phase 1 所有已建立的 location 及其所屬 sheet，用於去重檢測與 Phase 2 Mode 判斷
	phase1LocationSheetMap := make(map[string]string)

	// Phase 1 依序處理的工作表設定清單
	// 製程工作表（SMD/SMT, PTH, BOTTOM）預設 bom_status = "I", Type = sheetType, strictDedupe = true (嚴格去重)
	// NI 工作表預設 bom_status = "X", Type = "", strictDedupe = false (遇到已建立 location 自動忽略跳過)
	// MP 工作表預設 bom_status = "M", Type = "", strictDedupe = false (遇到已建立 location 自動忽略跳過)
	phase1Sheets := []struct {
		sheetName     string
		sheetType     string
		defaultStatus string
		strictDedupe  bool
	}{
		{smdSheet, "SMD", "I", true},
		{pthSheet, "PTH", "I", true},
		{bottomSheet, "BOTTOM", "I", true},
		{niSheet, "", "X", false},
		{mpSheet, "", "M", false},
	}

	for _, ps := range phase1Sheets {
		if ps.sheetName == "" {
			continue
		}
		newLocCount, err := r.parsePhase1Sheet(
			f, ps.sheetName, ps.sheetType, ps.defaultStatus, ps.strictDedupe,
			materialsMap, mainCompMap, &mainCompList, &secondCompList,
			phase1LocationSheetMap, tracker,
		)
		if err != nil {
			return err
		}

		if r.logger != nil {
			r.logger.Debug(fmt.Sprintf("[EBOM Phase1] 工作表 [%s] 解析完成", ps.sheetName),
				"sheet", ps.sheetName, "type", ps.sheetType,
				"mainComps", len(mainCompList), "secondComps", len(secondCompList), "newLocations", newLocCount,
			)
		}
	}

	if r.logger != nil {
		r.logger.Debug("[EBOM Phase1] 所有 sheet 解析完成",
			"uniqueMaterials", len(materialsMap),
			"mainComponents", len(mainCompList),
			"secondComponents", len(secondCompList),
			"totalLocations", len(phase1LocationSheetMap),
		)
	}

	// ─── 儲存 Phase 1 資料至資料庫（Material Upsert + Component + Location）──────
	savedLocations, err := r.saveComponents(materialsMap, mainCompList, secondCompList)
	if err != nil {
		return fmt.Errorf("failed to save components: %w", err)
	}

	if r.result != nil {
		r.result.PartsCount = len(mainCompList)
		r.result.SecondSources = len(secondCompList)
		r.result.ComponentsCount = len(mainCompList) + len(secondCompList)
	}

	// ─── Phase 2：Location 狀態覆寫 + Mode 判斷 ───────────────────────────────
	// 1. 建立 ComponentID -> (Supplier, SupplierPN) 映射
	compMaterialMap := make(map[int64]struct{ Supplier, SupplierPN string }, len(mainCompList))
	for _, mc := range mainCompList {
		compMaterialMap[mc.ID] = struct{ Supplier, SupplierPN string }{
			Supplier:   mc.Supplier,
			SupplierPN: mc.SupplierPN,
		}
	}

	// 2. 建立 (Supplier|SupplierPN|Location) -> []*db.PartLocation 複合查詢映射（指向 savedLocations 實體以同步記憶體狀態）
	locIndexMap := make(map[string][]*db.PartLocation, len(savedLocations))
	for i := range savedLocations {
		matInfo := compMaterialMap[savedLocations[i].ComponentID]
		key := makePartLocMatchKey(matInfo.Supplier, matInfo.SupplierPN, savedLocations[i].Location)
		locIndexMap[key] = append(locIndexMap[key], &savedLocations[i])
	}

	// 收集 PROTO / MP 頁面的 locations，用於 Mode 判斷
	var protoLocations []string
	var mpLocations []string

	// 處理 PROTO sheet（更新 bom_status = P，嚴格前置條件：僅覆蓋 BomStatus == "I" 的紀錄）
	if protoSheet != "" {
		protoItems := r.parsePhase2Items(f, protoSheet, tracker)
		var locationIDsToUpdate []int64
		for _, item := range protoItems {
			protoLocations = append(protoLocations, item.Location)
			key := makePartLocMatchKey(item.Supplier, item.SupplierPN, item.Location)
			if targets, exists := locIndexMap[key]; exists {
				for _, target := range targets {
					if target.BomStatus == "I" {
						locationIDsToUpdate = append(locationIDsToUpdate, target.ID)
						target.BomStatus = "P" // 同步更新記憶體狀態
					} else {
						// 原狀態不符合者僅記錄 debug log，不產生 warning
						if r.logger != nil {
							r.logger.Debug(fmt.Sprintf("[EBOM Phase2] PROTO location '%s' (Supplier: '%s', PN: '%s') 原狀態為 '%s' (非 'I')，略過覆寫",
								item.Location, item.Supplier, item.SupplierPN, target.BomStatus))
						}
					}
				}
			} else {
				warnMsg := fmt.Sprintf("[EBOM Phase2] PROTO location '%s' 在 Phase 1 中未建立，略過 (Supplier: '%s', PN: '%s')", item.Location, item.Supplier, item.SupplierPN)
				if r.logger != nil {
					r.logger.Warn(warnMsg, "sheet", protoSheet, "location", item.Location, "supplier", item.Supplier, "supplierPN", item.SupplierPN)
				}
				r.warnings = append(r.warnings, warnMsg)
			}
		}
		if len(locationIDsToUpdate) > 0 {
			if err := r.db.Model(&db.PartLocation{}).
				Where("id IN ?", locationIDsToUpdate).
				Update("bom_status", "P").Error; err != nil {
				return fmt.Errorf("failed to update PROTO locations: %w", err)
			}
		}
		if r.logger != nil {
			r.logger.Debug("[EBOM Phase2] PROTO 覆寫完成",
				"totalItems", len(protoItems),
				"updatedLocations", len(locationIDsToUpdate),
			)
		}
	}

	// 處理 MP sheet（更新 bom_status = M，嚴格前置條件：僅覆蓋 BomStatus == "X" 的紀錄）
	if mpSheet != "" {
		mpItems := r.parsePhase2Items(f, mpSheet, tracker)
		var locationIDsToUpdate []int64
		for _, item := range mpItems {
			mpLocations = append(mpLocations, item.Location)
			key := makePartLocMatchKey(item.Supplier, item.SupplierPN, item.Location)
			if targets, exists := locIndexMap[key]; exists {
				for _, target := range targets {
					if target.BomStatus == "X" {
						locationIDsToUpdate = append(locationIDsToUpdate, target.ID)
						target.BomStatus = "M" // 同步更新記憶體狀態
					} else {
						// 原狀態不符合者僅記錄 debug log，不產生 warning
						if r.logger != nil {
							r.logger.Debug(fmt.Sprintf("[EBOM Phase2] MP location '%s' (Supplier: '%s', PN: '%s') 原狀態為 '%s' (非 'X')，略過覆寫",
								item.Location, item.Supplier, item.SupplierPN, target.BomStatus))
						}
					}
				}
			} else {
				warnMsg := fmt.Sprintf("[EBOM Phase2] MP location '%s' 在 Phase 1 中未建立，略過 (Supplier: '%s', PN: '%s')", item.Location, item.Supplier, item.SupplierPN)
				if r.logger != nil {
					r.logger.Warn(warnMsg, "sheet", mpSheet, "location", item.Location, "supplier", item.Supplier, "supplierPN", item.SupplierPN)
				}
				r.warnings = append(r.warnings, warnMsg)
			}
		}
		if len(locationIDsToUpdate) > 0 {
			if err := r.db.Model(&db.PartLocation{}).
				Where("id IN ?", locationIDsToUpdate).
				Update("bom_status", "M").Error; err != nil {
				return fmt.Errorf("failed to update MP locations: %w", err)
			}
		}
		if r.logger != nil {
			r.logger.Debug("[EBOM Phase2] MP 覆寫完成",
				"totalItems", len(mpItems),
				"updatedLocations", len(locationIDsToUpdate),
			)
		}
	}

	// 處理 CCL sheet（更新 ccl = true，全面覆蓋，不限制 BomStatus）
	if cclSheet != "" {
		cclItems := r.parsePhase2Items(f, cclSheet, tracker)
		var locationIDsToUpdate []int64
		for _, item := range cclItems {
			key := makePartLocMatchKey(item.Supplier, item.SupplierPN, item.Location)
			if targets, exists := locIndexMap[key]; exists {
				for _, target := range targets {
					locationIDsToUpdate = append(locationIDsToUpdate, target.ID)
					target.CCL = true
				}
			} else {
				warnMsg := fmt.Sprintf("[EBOM Phase2] CCL location '%s' 在 Phase 1 中未建立，略過 (Supplier: '%s', PN: '%s')", item.Location, item.Supplier, item.SupplierPN)
				if r.logger != nil {
					r.logger.Warn(warnMsg, "sheet", cclSheet, "location", item.Location, "supplier", item.Supplier, "supplierPN", item.SupplierPN)
				}
				r.warnings = append(r.warnings, warnMsg)
			}
		}
		if len(locationIDsToUpdate) > 0 {
			if err := r.db.Model(&db.PartLocation{}).
				Where("id IN ?", locationIDsToUpdate).
				Update("ccl", true).Error; err != nil {
				return fmt.Errorf("failed to update CCL locations: %w", err)
			}
		}
		if r.logger != nil {
			r.logger.Debug("[EBOM Phase2] CCL 覆寫完成",
				"totalItems", len(cclItems),
				"updatedLocations", len(locationIDsToUpdate),
			)
		}
	}

	// ─── 判斷 Mode（NPI / MP）並回寫 BomRevision ───────────────────────────
	mode := determineMode(phase1LocationSheetMap, protoLocations, mpLocations)
	if err := r.db.Model(&db.BomRevision{}).
		Where("id = ?", r.revisionID).
		Update("mode", mode).Error; err != nil {
		return fmt.Errorf("failed to update revision mode: %w", err)
	}
	if r.logger != nil {
		r.logger.Info(fmt.Sprintf("[EBOM Phase2] BomRevision.Mode 判斷結果: %s", mode),
			"revisionID", r.revisionID,
		)
	}

	// ─── 清理因物料移除而無效的 MatrixSelection ─────────────────────────────────
	if err := r.cleanInvalidMatrixSelections(); err != nil {
		return fmt.Errorf("failed to clean invalid matrix selections: %w", err)
	}

	// ─── 自動匯入上一版 Matrix Selection ─────────────────────────────────────
	var revision db.BomRevision
	if err := r.db.Where("id = ?", r.revisionID).First(&revision).Error; err == nil {
		if err := r.autoImportPreviousMatrix(revision); err != nil {
			if r.logger != nil {
				r.logger.Warn(fmt.Sprintf("自動匯入 Matrix 失敗（非致命）: %v", err))
			}
			if task.IsWarningError(err) {
				r.warnings = append(r.warnings, err.Error())
			}
		}
	}

	// ─── 若有警告，以 WarningError 形式回傳，使 Task Manager 將任務狀態設為 TaskWarning ──
	if len(r.warnings) > 0 {
		combined := strings.Join(r.warnings, "; ")
		return task.NewWarningError(fmt.Errorf("%s", combined))
	}

	return nil
}

// ─── 工具函數 ────────────────────────────────────────────────────────────────

// findSheetCaseInsensitive 依名稱（不區分大小寫）尋找工作表
func (r *EBOMReader) findSheetCaseInsensitive(sheets []string, name string) string {
	for _, sheet := range sheets {
		if strings.EqualFold(strings.TrimSpace(sheet), name) {
			return sheet
		}
	}
	return ""
}

// parseHeaderField 從儲存格字串中萃取前綴後的值
func parseHeaderField(val string, prefixes ...string) string {
	val = strings.TrimSpace(val)
	if val == "" {
		return ""
	}
	lowerVal := strings.ToLower(val)
	for _, p := range prefixes {
		lowerP := strings.ToLower(p)
		if idx := strings.Index(lowerVal, lowerP); idx != -1 {
			res := val[idx+len(lowerP):]
			res = strings.TrimPrefix(res, ":")
			return strings.TrimSpace(res)
		}
	}
	return val
}

// parseHeader 解析 EBOM 表頭（從 SMD sheet）
// 回傳：phase, version, description, schematicVersion, pcbVersion, pcaPn, date, projectCode, error
func (r *EBOMReader) parseHeader(f Workbook, sheetName string) (phase, version, description, schematicVersion, pcbVersion, pcaPn, date, projectCode string, err error) {
	valB3, _ := f.GetCellValue(sheetName, "B3")
	projectCode = parseHeaderField(valB3, "Product Code", "Project Code")

	valB4, _ := f.GetCellValue(sheetName, "B4")
	description = parseHeaderField(valB4, "Description")

	valD3, _ := f.GetCellValue(sheetName, "D3")
	schematicVersion = parseHeaderField(valD3, "Schematic Version")

	valJ3, _ := f.GetCellValue(sheetName, "J3")
	phase = parseHeaderField(valJ3, "Phase")

	valF3, _ := f.GetCellValue(sheetName, "F3")
	pcbVersion = parseHeaderField(valF3, "PCB Version")

	valF4, _ := f.GetCellValue(sheetName, "F4")
	pcaPn = parseHeaderField(valF4, "PCA PN")

	valH3, _ := f.GetCellValue(sheetName, "H3")
	version = parseHeaderField(valH3, "BOM Version", "Version")

	valH4, _ := f.GetCellValue(sheetName, "H4")
	date = parseHeaderField(valH4, "Date")

	if r.logger != nil {
		r.logger.Debug(fmt.Sprintf("[EBOM] 表頭解析: ProjectCode=%s, Phase=%s, Version=%s", projectCode, phase, version))
	}

	return phase, version, description, schematicVersion, pcbVersion, pcaPn, date, projectCode, nil
}

// safeGetCol 安全取得 row 中指定欄位的值（超出範圍回傳空字串）
func safeGetCol(row []string, colIndex int) string {
	if colIndex >= 0 && colIndex < len(row) {
		return strings.TrimSpace(row[colIndex])
	}
	return ""
}

// atomizeLocations 將逗號分隔的 location 字串拆分為獨立的 location 陣列
// 例如 "C1, C2, C3" → ["C1", "C2", "C3"]
func atomizeLocations(locationStr string) []string {
	raw := strings.Split(locationStr, ",")
	result := make([]string, 0, len(raw))
	for _, p := range raw {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// determineMode 根據 Phase 2 的 location 與 Phase 1 已建立的 location 判斷 BOM Mode。
//
// 判斷規則：
//   - 若 PROTO 任一 location 存在於 Phase 1 → NPI
//   - 若 MP 任一 location 存在於 Phase 1 → MP
//   - 否則預設 NPI
func determineMode(phase1LocationSheetMap map[string]string, protoLocations, mpLocations []string) string {
	for _, loc := range protoLocations {
		if _, exists := phase1LocationSheetMap[loc]; exists {
			return "NPI"
		}
	}
	for _, loc := range mpLocations {
		if _, exists := phase1LocationSheetMap[loc]; exists {
			return "MP"
		}
	}
	return "NPI" // 預設
}

// ─── Phase 1 解析結構與函數 ──────────────────────────────────────────────────

// parsedComponentLocation 暫存原子化 location
type parsedComponentLocation struct {
	Location  string
	Type      string
	BomStatus string
	CCL       bool
}

// parsedComponentMain 暫存解析到的主料
type parsedComponentMain struct {
	ID         int64 // 儲存後回填 ComponentID
	Item       string
	Supplier   string
	SupplierPN string
	Locations  []parsedComponentLocation
}

// parsedComponentSecond 暫存解析到的替代料
type parsedComponentSecond struct {
	Supplier   string
	SupplierPN string
	MainComp   *parsedComponentMain // 所屬主料指標
}

// parsePhase1Sheet 解析 Phase 1 之工作表（SMD/SMT、PTH、BOTTOM、NI、MP）。
// 統一遵循 parseMainSheetV2 原則：
//  1. 非料件列過濾：(supplier == "" && supplierPN == "") 或 "Total:" 列予以忽略。
//  2. 物料收集：收集 Material（HHPN, Description, Supplier, SupplierPN, Remark）。
//  3. 主料/替代料判定：Item 有值為主料；Item 為空且緊隨主料為替代料。
//  4. CCL 標記：檢查 Col 9 (CCL) 是否為 "Y"。
//  5. Location 去重：
//     - SMD/PTH/BOTTOM (strictDedupe=true)：嚴格去重，若同表或跨表重複，記錄 r.logger.Error 並回傳 ErrDuplicateLocation 致命錯誤。
//     - NI/MP (strictDedupe=false)：遇到已建立 location 自動忽略跳過 (不中斷、不設定為任務失敗)。
func (r *EBOMReader) parsePhase1Sheet(
	f Workbook,
	sheetName, sheetType, defaultBomStatus string,
	strictDedupe bool,
	materialsMap map[string]db.Material,
	mainCompMap map[string]*parsedComponentMain,
	mainCompList *[]*parsedComponentMain,
	secondCompList *[]*parsedComponentSecond,
	phase1LocationSheetMap map[string]string,
	tracker *ProgressTracker,
) (int, error) {
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return 0, fmt.Errorf("讀取工作表 [%s] 失敗: %w", sheetName, err)
	}

	newLocCount := 0
	var currentMainComp *parsedComponentMain
	niSeenLocations := make(map[string]bool)

	for i := 5; i < len(rows); i++ {
		if tracker != nil {
			tracker.AddRows(1)
		}
		row := rows[i]
		if len(row) == 0 {
			continue
		}

		item := safeGetCol(row, 0)
		hhpn := safeGetCol(row, 1)
		description := safeGetCol(row, 4)
		supplier := strings.TrimSpace(safeGetCol(row, 5))
		supplierPN := strings.TrimSpace(safeGetCol(row, 6))
		locationStr := safeGetCol(row, 8)
		cclVal := safeGetCol(row, 9)
		remark := safeGetCol(row, 11)

		// 1. 非料件列過濾（採用 parseMainSheetV2 原則，同時排除 Total: 統計列）
		if (supplier == "" && supplierPN == "") || strings.EqualFold(supplierPN, "total:") || strings.EqualFold(supplier, "total:") {
			continue
		}

		key := supplier + "|" + supplierPN

		// 收集 Material 物料資訊（供全域 Material Upsert，包含 Remark）
		materialsMap[key] = db.Material{
			Supplier:    supplier,
			SupplierPN:  supplierPN,
			HHPN:        hhpn,
			Description: description,
			Remark:      remark,
		}

		// 1. 主料、替代料判定（採用 parseMainSheetV2 原則）
		if item != "" {
			// 主料 (Main Source)
			mainComp, exists := mainCompMap[key]
			if !exists {
				mainComp = &parsedComponentMain{
					Item:       item,
					Supplier:   supplier,
					SupplierPN: supplierPN,
				}
				mainCompMap[key] = mainComp
				*mainCompList = append(*mainCompList, mainComp)
			}
			currentMainComp = mainComp

			// CCL 標記（Col 9 是否為 "Y"）
			isCCL := strings.EqualFold(cclVal, "Y")
			for _, loc := range atomizeLocations(locationStr) {
				// 2. Location 去重檢測：
				// SMD/PTH/BOTTOM 嚴格去重（若同表或跨表重複，log 錯誤並失敗）；
				// NI/MP 遇到已建立的 location 處理：
				//   - NI 工作表：允許與主製程 (SMD/PTH/BOTTOM) 重複並成立紀錄 (BomStatus="X")，表內自身去重
				//   - MP 工作表：自動忽略跳過 (不中斷、不設為任務失敗)
				if prevSheet, exists := phase1LocationSheetMap[loc]; exists {
					if strictDedupe {
						var errMsg string
						if prevSheet == sheetName {
							errMsg = fmt.Sprintf("工作表 [%s] 發現重複的 Location: '%s' (在工作表內重複定義)", sheetName, loc)
						} else {
							errMsg = fmt.Sprintf("發現重複的 Location: '%s' (工作表 [%s] 與工作表 [%s] 重複)", loc, prevSheet, sheetName)
						}
						if r.logger != nil {
							r.logger.Error(errMsg, "sheet", sheetName, "location", loc, "previousSheet", prevSheet)
						}
						return 0, fmt.Errorf("%s: %w", errMsg, types.ErrDuplicateLocation)
					}

					// ─── 針對 NI 工作表的特殊處理 ───
					if strings.EqualFold(sheetName, "NI") {
						if niSeenLocations[loc] {
							continue // 若在同一個 NI sheet 內重複定義，則跳過，避免在 NI 中重複建檔
						}
						niSeenLocations[loc] = true

						mainComp.Locations = append(mainComp.Locations, parsedComponentLocation{
							Location:  loc,
							Type:      sheetType,
							BomStatus: defaultBomStatus,
							CCL:       isCCL,
						})
						newLocCount++
						continue
					}

					// 其他工作表（如 MP）遇到已建立的 location 自動忽略跳過
					continue
				}

				// 首次出現的 Location 正常記錄
				phase1LocationSheetMap[loc] = sheetName
				if strings.EqualFold(sheetName, "NI") {
					niSeenLocations[loc] = true
				}

				mainComp.Locations = append(mainComp.Locations, parsedComponentLocation{
					Location:  loc,
					Type:      sheetType,
					BomStatus: defaultBomStatus,
					CCL:       isCCL,
				})
				newLocCount++
			}

			if r.logger != nil {
				r.logger.Debug(fmt.Sprintf("[EBOM Phase1] [%s] 主料: %s / %s, type=%s, locations=%s", sheetName, supplier, supplierPN, sheetType, locationStr))
			}
		} else if currentMainComp != nil {
			// 替代料 (Second Source)
			*secondCompList = append(*secondCompList, &parsedComponentSecond{
				Supplier:   supplier,
				SupplierPN: supplierPN,
				MainComp:   currentMainComp,
			})

			if r.logger != nil {
				r.logger.Debug(fmt.Sprintf("[EBOM Phase1] [%s] 替代料: %s / %s", sheetName, supplier, supplierPN))
			}
		}
	}

	return newLocCount, nil
}

// parseMainSheetV2 是對 parsePhase1Sheet 的相容轉發函式（預設 bom_status = "I", strictDedupe = true）。
func (r *EBOMReader) parseMainSheetV2(
	f Workbook,
	sheetName, sheetType string,
	materialsMap map[string]db.Material,
	mainCompMap map[string]*parsedComponentMain,
	mainCompList *[]*parsedComponentMain,
	secondCompList *[]*parsedComponentSecond,
	phase1LocationSheetMap map[string]string,
	tracker *ProgressTracker,
) int {
	newLocCount, _ := r.parsePhase1Sheet(
		f, sheetName, sheetType, "I", true,
		materialsMap, mainCompMap, mainCompList, secondCompList,
		phase1LocationSheetMap, tracker,
	)
	return newLocCount
}

// Phase2Item 暫存 Phase 2 狀態工作表解析項目
type Phase2Item struct {
	Supplier   string
	SupplierPN string
	Location   string
}

// makePartLocMatchKey 產生標準化比對鍵值 (Supplier|SupplierPN|Location)
func makePartLocMatchKey(supplier, supplierPN, location string) string {
	return strings.ToLower(strings.TrimSpace(supplier)) + "|" +
		strings.ToLower(strings.TrimSpace(supplierPN)) + "|" +
		strings.ToUpper(strings.TrimSpace(location))
}

// parsePhase2Items 解析 Phase 2 狀態 sheet（PROTO / MP / CCL），收集包含 Supplier, SupplierPN 與 Location 的結構化清單。
func (r *EBOMReader) parsePhase2Items(f Workbook, sheetName string, tracker *ProgressTracker) []Phase2Item {
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil
	}

	var items []Phase2Item
	for i := 5; i < len(rows); i++ {
		if tracker != nil {
			tracker.AddRows(1)
		}
		row := rows[i]
		if len(row) == 0 {
			continue
		}
		supplier := strings.TrimSpace(safeGetCol(row, 5))
		supplierPN := strings.TrimSpace(safeGetCol(row, 6))
		if supplier == "" || supplierPN == "" {
			continue
		}
		locationStr := safeGetCol(row, 8)
		for _, loc := range atomizeLocations(locationStr) {
			items = append(items, Phase2Item{
				Supplier:   supplier,
				SupplierPN: supplierPN,
				Location:   loc,
			})
		}
	}
	return items
}

// parsePhase2Sheet 解析 Phase 2 狀態 sheet（PROTO / MP / CCL），僅收集 location 字串清單（保留向後相容）。
func (r *EBOMReader) parsePhase2Sheet(f Workbook, sheetName string, tracker *ProgressTracker) []string {
	items := r.parsePhase2Items(f, sheetName, tracker)
	locations := make([]string, len(items))
	for i, it := range items {
		locations[i] = it.Location
	}
	return locations
}

// ─── 儲存函數 ────────────────────────────────────────────────────────────────

// saveComponents 將 Phase 1 解析到的 Material、RevisionComponent 與 PartLocation 寫入資料庫。
//
// 流程：
//  1. 批次 Upsert 全域物料庫（空白值防護：hhpn/desc 保留舊值，remark 覆寫清空）
//  2. 查詢 Material ID 映射
//  3. 事務內：清除舊 Components 與 Locations、建立主料 Component、建立替代料 Component、建立 Locations
func (r *EBOMReader) saveComponents(
	materialsMap map[string]db.Material,
	mainCompList []*parsedComponentMain,
	secondCompList []*parsedComponentSecond,
) ([]db.PartLocation, error) {
	if r.revisionID == 0 {
		return nil, errors.New("revision ID not set")
	}

	// Step 1: 批次 Upsert 全域物料
	toUpsert := make([]db.Material, 0, len(materialsMap))
	for _, m := range materialsMap {
		toUpsert = append(toUpsert, m)
	}
	var lg db.MatrixLogger
	if r.logger != nil {
		lg = r.logger
	}
	ins, upd, err := db.UpsertMaterials(r.db, toUpsert, lg)
	if err != nil {
		return nil, fmt.Errorf("UpsertMaterials 失敗: %w", err)
	}
	if r.result != nil {
		r.result.MaterialsInserted = ins
		r.result.MaterialsUpdated = upd
	}

	// Step 2: 查詢所有物料的 DB Material.ID
	keys := make([]string, 0, len(materialsMap))
	for k := range materialsMap {
		keys = append(keys, k)
	}
	matDBMap, err := db.GetMaterialMapBySupplierPNs(r.db, keys)
	if err != nil {
		return nil, fmt.Errorf("查詢 Material 失敗: %w", err)
	}

	// Step 3: Transaction 內寫入本 Revision 的 RevisionComponent 與 PartLocation
	var savedLocations []db.PartLocation
	err = r.db.Transaction(func(tx *gorm.DB) error {
		// 刪除此 Revision 的舊 Components 與 PartLocations
		if err := db.DeleteComponentsByRevision(tx, r.revisionID); err != nil {
			return fmt.Errorf("刪除舊 RevisionComponent 失敗: %w", err)
		}

		// 批次寫入主料 RevisionComponent (Role="M")
		mainRCs := make([]db.RevisionComponent, len(mainCompList))
		for i, mc := range mainCompList {
			key := mc.Supplier + "|" + mc.SupplierPN
			mat, ok := matDBMap[key]
			if !ok {
				return fmt.Errorf("找不到主料 Material: %s", key)
			}
			mainRCs[i] = db.RevisionComponent{
				RevisionID:        r.revisionID,
				MaterialID:        mat.ID,
				Role:              "M",
				ParentComponentID: 0,
				Item:              mc.Item,
			}
		}
		if len(mainRCs) > 0 {
			if err := tx.CreateInBatches(&mainRCs, 500).Error; err != nil {
				return fmt.Errorf("批次建立主料 RevisionComponent 失敗: %w", err)
			}
			for i, mc := range mainCompList {
				mc.ID = mainRCs[i].ID
			}
		}

		// 批次寫入替代料 RevisionComponent (Role="S")
		secondRCs := make([]db.RevisionComponent, 0, len(secondCompList))
		seenSecond := make(map[string]bool)
		for _, sc := range secondCompList {
			if sc.MainComp == nil || sc.MainComp.ID == 0 {
				continue
			}
			key := sc.Supplier + "|" + sc.SupplierPN
			mat, ok := matDBMap[key]
			if !ok {
				continue
			}
			secKey := fmt.Sprintf("%d_%d", mat.ID, sc.MainComp.ID)
			if seenSecond[secKey] {
				continue
			}
			seenSecond[secKey] = true
			secondRCs = append(secondRCs, db.RevisionComponent{
				RevisionID:        r.revisionID,
				MaterialID:        mat.ID,
				Role:              "S",
				ParentComponentID: sc.MainComp.ID,
			})
		}
		if len(secondRCs) > 0 {
			if err := tx.CreateInBatches(&secondRCs, 500).Error; err != nil {
				return fmt.Errorf("批次建立替代料 RevisionComponent 失敗: %w", err)
			}
		}

		// 批次建立 PartLocation (ComponentID 指向主料 Component.ID)
		var locationsToInsert []db.PartLocation
		for _, mc := range mainCompList {
			if mc.ID == 0 {
				continue
			}
			for _, loc := range mc.Locations {
				locationsToInsert = append(locationsToInsert, db.PartLocation{
					ComponentID: mc.ID,
					Location:    loc.Location,
					Type:        loc.Type,
					BomStatus:   loc.BomStatus,
					CCL:         loc.CCL,
				})
			}
		}
		if len(locationsToInsert) > 0 {
			if err := db.CreatePartLocationsInBatch(tx, locationsToInsert); err != nil {
				return fmt.Errorf("批次建立 PartLocation 失敗: %w", err)
			}

			// 載入具備真實 ID 的 PartLocation 清單以供 Phase 2 狀態覆寫
			mainCompIDs := make([]int64, len(mainCompList))
			for i, mc := range mainCompList {
				mainCompIDs[i] = mc.ID
			}
			var locs []db.PartLocation
			if err := tx.Where("component_id IN ?", mainCompIDs).Find(&locs).Error; err != nil {
				return fmt.Errorf("查詢已儲存之 PartLocation 失敗: %w", err)
			}
			savedLocations = locs
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if r.logger != nil {
		r.logger.Info("[EBOM] 資料儲存完成",
			"mainComponents", len(mainCompList),
			"secondComponents", len(secondCompList),
			"locations", len(savedLocations),
		)
	}

	return savedLocations, nil
}

// ─── Revision 管理 ───────────────────────────────────────────────────────────

// checkRevisionExists 檢查指定 Project + Phase + Version 的 BomRevision 是否已存在於資料庫中
func (r *EBOMReader) checkRevisionExists(projectCode, phase, version string) (bool, error) {
	if r.db == nil {
		return false, errors.New("db is nil")
	}
	series, err := db.GetSeriesInfo(r.db)
	if err != nil {
		return false, fmt.Errorf("取得 Series 失敗: %w", err)
	}

	var project db.Project
	err = r.db.Where("series_id = ? AND code = ?", series.ID, projectCode).First(&project).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	var count int64
	err = r.db.Model(&db.BomRevision{}).
		Where("project_id = ? AND phase = ? AND version = ?", project.ID, phase, version).
		Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// createOrUpdateRevision 建立或更新 BomRevision 記錄，回傳 Revision ID
func (r *EBOMReader) createOrUpdateRevision(projectCode, phase, version, description, schematicVersion, pcbVersion, pcaPn, date string) (int64, error) {
	if r.db == nil {
		return 0, errors.New("db is nil")
	}

	series, err := db.GetSeriesInfo(r.db)
	if err != nil {
		return 0, fmt.Errorf("取得 Series 失敗: %w", err)
	}

	projectPtr, err := db.GetOrCreateProject(r.db, series.ID, projectCode, description)
	if err != nil {
		return 0, fmt.Errorf("取得/建立 Project 失敗: %w", err)
	}
	project := *projectPtr

	var existing db.BomRevision
	err = r.db.Where("project_id = ? AND phase = ? AND version = ?",
		project.ID, phase, version).
		First(&existing).Error

	if err == nil {
		// 更新既有 Revision
		if r.logger != nil {
			r.logger.Info("BOM 覆蓋：更新現有版本", "project", projectCode, "phase", phase, "version", version)
		}
		existing.Description = description
		existing.SchematicVersion = schematicVersion
		existing.PCBVersion = pcbVersion
		existing.PCAPN = pcaPn
		existing.Date = date
		if r.filePath != "" {
			existing.SourceFile = filepath.Base(r.filePath)
		}
		existing.UpdatedAt = time.Now()
		if err := r.db.Save(&existing).Error; err != nil {
			return 0, err
		}
		return existing.ID, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}

	sourceFile := ""
	if r.filePath != "" {
		sourceFile = filepath.Base(r.filePath)
	}

	// 建立新 Revision
	revision := db.BomRevision{
		ProjectID:        project.ID,
		Phase:            phase,
		Version:          version,
		Description:      description,
		SchematicVersion: schematicVersion,
		PCBVersion:       pcbVersion,
		PCAPN:            pcaPn,
		Date:             date,
		SourceFile:       sourceFile,
		Mode:             "NPI", // 預設，Phase 2 後會更新
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	if err := r.db.Create(&revision).Error; err != nil {
		return 0, err
	}

	if r.logger != nil {
		r.logger.Info("BOM 新增：建立全新版本", "project", projectCode, "phase", phase, "version", version)
	}

	return revision.ID, nil
}

// ─── MatrixSelection 清理 ───────────────────────────────────────────────────

// cleanInvalidMatrixSelections 刪除當前 Revision 中，主料或選中物料已不再屬於本 Revision 的 MatrixSelection
func (r *EBOMReader) cleanInvalidMatrixSelections() error {
	return r.db.Exec(`
		DELETE FROM matrix_selections 
		WHERE revision_id = ? 
		  AND (
		    main_material_id NOT IN (SELECT material_id FROM revision_components WHERE revision_id = ? AND role = 'M')
		    OR selected_material_id NOT IN (SELECT material_id FROM revision_components WHERE revision_id = ?)
		  )
	`, r.revisionID, r.revisionID, r.revisionID).Error
}

// ─── 自動匯入上一版 Matrix Selection ─────────────────────────────────────────

// autoImportPreviousMatrix 自動從上一版 BOM Revision 匯入 Matrix Selection。
//
// 執行流程：
//  1. 呼叫 FindPreviousRevisionSmart 以數值方式尋找前一版 BOM Revision。
//  2. 若版本號包含文字字元（無法數值排序），輸出警告 log 並回傳 WarningError 提醒使用者手動複製。
//  3. 若找到前一版，呼叫 ImportMatrixSelections 執行覆蓋式 Matrix 匯入。
//  4. 匯入完成後輸出詳細統計 log（model 數、複製數、忽略主料數、忽略 2nd 數）。
//
// 參數：
//   - revision: 當前已匯入完成的 BomRevision（作為 target）
//
// 回傳：
//   - error: 若版本含文字則為 WarningError；若資料庫操作失敗則為一般 error
func (r *EBOMReader) autoImportPreviousMatrix(revision db.BomRevision) error {
	projectDisplay := fmt.Sprintf("%d", revision.ProjectID)
	var proj db.Project
	if r.db != nil {
		if err := r.db.First(&proj, revision.ProjectID).Error; err == nil && proj.Code != "" {
			projectDisplay = fmt.Sprintf("%s(%d)", proj.Code, revision.ProjectID)
		}
	}

	// 步驟 1：使用智慧版本排序尋找前一版
	previous, hasNonNumeric, err := db.FindPreviousRevisionSmart(r.db, revision.ProjectID, revision.Phase, revision.Version)
	if err != nil {
		return fmt.Errorf("查詢前一版 Revision 失敗: %w", err)
	}

	// 步驟 2：若版本號含文字字元，無法自動排序，輸出警告並告知使用者手動處理
	if hasNonNumeric {
		warnMsg := fmt.Sprintf(
			"[autoImportPreviousMatrix] 版本號包含非數字字元，無法自動進行版本排序比對。"+
				"跳過自動 Matrix 匯入，請手動使用「複製 Matrix」功能選擇版本進行複製 "+
				"(ProjectID=%s, Phase=%s, Version=%s)",
			projectDisplay, revision.Phase, revision.Version,
		)
		if r.logger != nil {
			r.logger.Warn(warnMsg)
		}
		return task.NewWarningError(errors.New(warnMsg))
	}

	// 步驟 3：若無前一版，直接返回（此版本為同 phase 的第一版）
	if previous == nil {
		if r.logger != nil {
			r.logger.Debug(fmt.Sprintf(
				"[autoImportPreviousMatrix] 同 Phase 無前一版可繼承 (ProjectID=%s, Phase=%s, Version=%s)",
				projectDisplay, revision.Phase, revision.Version,
			))
		}
		return nil
	}

	if r.logger != nil {
		r.logger.Info(fmt.Sprintf(
			"[autoImportPreviousMatrix] 找到前一版 Revision ID=%d (Version=%s)，開始自動匯入 Matrix Selection",
			previous.ID, previous.Version,
		))
	}

	// 步驟 4：執行 Matrix Selection 複製（覆蓋模式）
	stats, err := db.ImportMatrixSelections(r.db, previous.ID, revision.ID, r.logger)
	if err != nil {
		return fmt.Errorf("自動匯入 Matrix Selection 失敗: %w", err)
	}

	// 步驟 5：輸出完整統計 log
	if r.logger != nil {
		r.logger.Info(fmt.Sprintf(
			"[autoImportPreviousMatrix] 自動繼承完成 | 來源 Version=%s → 目標 Version=%s | "+
				"有效 Model 數=%d, 複製 Selection 數=%d, 忽略主料數=%d, 忽略 2nd 替代料數=%d",
			previous.Version, revision.Version,
			stats.SourceModelCount, stats.CopiedSelectionsCount,
			stats.IgnoredMainParts, stats.IgnoredSecondParts,
		))
	}

	return nil
}
