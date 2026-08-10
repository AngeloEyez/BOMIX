package excel

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
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
	db         *gorm.DB
	result     *types.ImportResult
	filePath   string // EBOM 匯入檔案路徑
	revisionID int64  // 匯入後設定
	logger     *logger.Logger
}

// Import 匯入 EBOM 格式 Excel 檔案。
//
// 採用兩階段匯入流程：
//   Phase 1（主料建置）：處理 SMD / PTH / BOTTOM / NI sheet，
//     依 (supplier, supplier_pn) 去重建立 Part，並為每個 location 建立原子化 PartLocation。
//   Phase 2（狀態覆寫）：處理 PROTO / MP / CCL sheet，
//     僅更新 Phase 1 已建立的 PartLocation 的 BomStatus / CCL 屬性；
//     同時判斷 BomRevision.Mode（NPI 或 MP）。
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

	var phase, version, description, schematicVersion, pcbVersion, pcaPn, date, projectCode string
	var err error
	if smdSheet != "" {
		phase, version, description, schematicVersion, pcbVersion, pcaPn, date, projectCode, err = r.parseHeader(f, smdSheet)
		if err != nil {
			return fmt.Errorf("failed to parse header: %w", err)
		}
	}

	// ─── 建立或更新 BomRevision ─────────────────────────────────────────────
	revisionID, err := r.createOrUpdateRevision(projectCode, phase, version, description, schematicVersion, pcbVersion, pcaPn, date)
	if err != nil {
		return fmt.Errorf("failed to create/update revision: %w", err)
	}
	r.revisionID = revisionID

	// ─── Phase 1：主料去重與 Location 建置 ───────────────────────────────────
	// partMap：(supplier|supplier_pn) → *db.Part，用於去重
	partMap := make(map[string]*db.Part)
	var partList []*db.Part
	var parsedSSList []parsedSecondSource

	// phase1LocationSet 收集 Phase 1 所有已建立的 location，供 Phase 2 Mode 判斷使用
	phase1LocationSet := make(map[string]bool)

	// 收集所有 Phase 1 的 location（parsedPartLocation 格式，含 Part 指標）
	var allParsedLocations []parsedPartLocation

	// 處理主製程 sheet（SMD / PTH / BOTTOM）
	mainSheets := []struct{ name, sheetType string }{
		{smdSheet, "SMD"},
		{pthSheet, "PTH"},
		{bottomSheet, "BOTTOM"},
	}
	for _, ms := range mainSheets {
		if ms.name == "" {
			continue
		}
		locs, ssList := r.parseMainSheetV2(f, ms.name, ms.sheetType, partMap, &partList)
		allParsedLocations = append(allParsedLocations, locs...)
		parsedSSList = append(parsedSSList, ssList...)

		for _, loc := range locs {
			phase1LocationSet[loc.location] = true
		}

		if r.logger != nil {
			r.logger.Debug(fmt.Sprintf("[EBOM Phase1] 工作表 [%s] 解析完成", ms.name),
				"sheet", ms.name, "type", ms.sheetType,
				"partMapSize", len(partMap), "newLocations", len(locs),
			)
		}
	}

	// 處理 NI sheet（bom_status = X）
	if niSheet != "" {
		locs := r.parseNISheet(f, niSheet, partMap, &partList)
		allParsedLocations = append(allParsedLocations, locs...)
		for _, loc := range locs {
			phase1LocationSet[loc.location] = true
		}
		if r.logger != nil {
			r.logger.Debug("[EBOM Phase1] NI 工作表解析完成",
				"sheet", niSheet, "locations", len(locs),
			)
		}
	}

	if r.logger != nil {
		r.logger.Debug("[EBOM Phase1] 所有 sheet 解析完成",
			"uniqueParts", len(partList),
			"totalLocations", len(allParsedLocations),
			"secondSources", len(parsedSSList),
		)
	}

	// ─── 儲存 Phase 1 資料至資料庫 ───────────────────────────────────────────
	savedLocations, savedSecondSources, err := r.saveParts(partList, allParsedLocations, parsedSSList)
	if err != nil {
		return fmt.Errorf("failed to save parts: %w", err)
	}

	if r.result != nil {
		r.result.PartsCount = len(partList)
		r.result.SecondSources = len(savedSecondSources)
	}

	// ─── Phase 2：Location 狀態覆寫 + Mode 判斷 ───────────────────────────────
	// 建立 location → *db.PartLocation 快速查詢映射（來自已儲存後的 savedLocations）
	locIndexMap := make(map[string]*db.PartLocation, len(savedLocations))
	for i := range savedLocations {
		locIndexMap[savedLocations[i].Location] = &savedLocations[i]
	}

	// 收集 PROTO / MP 頁面的 locations，用於 Mode 判斷
	var protoLocations []string
	var mpLocations []string

	// 處理 PROTO sheet（更新 bom_status = P）
	if protoSheet != "" {
		protoLocations = r.parsePhase2Sheet(f, protoSheet)
		var locationIDsToUpdate []int64
		for _, loc := range protoLocations {
			if target, exists := locIndexMap[loc]; exists {
				locationIDsToUpdate = append(locationIDsToUpdate, target.ID)
			} else {
				if r.logger != nil {
					r.logger.Warn(fmt.Sprintf("[EBOM Phase2] PROTO location '%s' 在 Phase 1 中未建立，略過", loc),
						"sheet", protoSheet, "location", loc,
					)
				}
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
				"totalLocations", len(protoLocations),
				"updatedLocations", len(locationIDsToUpdate),
			)
		}
	}

	// 處理 MP sheet（更新 bom_status = M）
	if mpSheet != "" {
		mpLocations = r.parsePhase2Sheet(f, mpSheet)
		var locationIDsToUpdate []int64
		for _, loc := range mpLocations {
			if target, exists := locIndexMap[loc]; exists {
				locationIDsToUpdate = append(locationIDsToUpdate, target.ID)
			} else {
				if r.logger != nil {
					r.logger.Warn(fmt.Sprintf("[EBOM Phase2] MP location '%s' 在 Phase 1 中未建立，略過", loc),
						"sheet", mpSheet, "location", loc,
					)
				}
			}
		}
		if len(locationIDsToUpdate) > 0 {
			if err := r.db.Model(&db.PartLocation{}).
				Where("id IN ?", locationIDsToUpdate).
				Update("bom_status", "M").Error; err != nil {
				return fmt.Errorf("failed to update MP locations: %w", err)
			}
		}
	}

	// 處理 CCL sheet（更新 ccl = true）
	if cclSheet != "" {
		cclLocations := r.parsePhase2Sheet(f, cclSheet)
		var locationIDsToUpdate []int64
		for _, loc := range cclLocations {
			if target, exists := locIndexMap[loc]; exists {
				locationIDsToUpdate = append(locationIDsToUpdate, target.ID)
			}
		}
		if len(locationIDsToUpdate) > 0 {
			if err := r.db.Model(&db.PartLocation{}).
				Where("id IN ?", locationIDsToUpdate).
				Update("ccl", true).Error; err != nil {
				return fmt.Errorf("failed to update CCL locations: %w", err)
			}
		}
	}

	// CCL 欄位覆寫（EBOM 中 CCL 欄位直接標記於零件行）
	// 已在 Phase 1 的 parseMainSheet 中處理

	// ─── 判斷 Mode（NPI / MP）並回寫 BomRevision ───────────────────────────
	mode := determineMode(phase1LocationSet, protoLocations, mpLocations)
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

	// ─── 套用 Merge 演算法（替代料 diff）──────────────────────────────────────
	if err := r.applyMergeAlgorithm(r.revisionID, savedSecondSources); err != nil {
		return fmt.Errorf("failed to apply merge algorithm: %w", err)
	}

	// ─── 自動匯入上一版 Matrix Selection ─────────────────────────────────────
	var revision db.BomRevision
	if err := r.db.Where("id = ?", r.revisionID).First(&revision).Error; err == nil {
		if err := r.autoImportPreviousMatrix(revision); err != nil {
			if r.logger != nil {
				r.logger.Warn(fmt.Sprintf("自動匯入 Matrix 失敗（非致命）: %v", err))
			}
		}
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
func determineMode(phase1LocationSet map[string]bool, protoLocations, mpLocations []string) string {
	for _, loc := range protoLocations {
		if phase1LocationSet[loc] {
			return "NPI"
		}
	}
	for _, loc := range mpLocations {
		if phase1LocationSet[loc] {
			return "MP"
		}
	}
	return "NPI" // 預設
}

// ─── Phase 1 解析函數 ────────────────────────────────────────────────────────

// parsedSecondSource 暫存解析到的替代料，並記錄關聯主料在 partList 中的索引
type parsedSecondSource struct {
	partPtr      *db.Part // 指向對應主料的指標（用於取得儲存後的 ID）
	secondSource db.SecondSource
}

// parseMainSheet 解析主製程 sheet（SMD / PTH / BOTTOM）。
//
// 採用 partMap 去重：同一 (supplier, supplier_pn) 只建一筆 Part。
// 每個 location 原子化，建立 PartLocation（BomStatus='I'）。
// 若 Excel 中 CCL 欄位有值（Y/y），則對應 PartLocation 設 CCL=true。
//
// 參數：
//   - f：Workbook 介面
//   - sheetName：工作表名稱
//   - sheetType：製程類型（SMD / PTH / BOTTOM）
//   - partMap：(supplier|supplier_pn) → *db.Part 去重映射（跨 sheet 共用）
//   - partList：Part 指標列表（跨 sheet 累加）
//
// 回傳：新增的 Part 指標列表、PartLocation 指標列表、SecondSource 列表
func (r *EBOMReader) parseMainSheet(
	f Workbook,
	sheetName, sheetType string,
	partMap map[string]*db.Part,
	partList *[]*db.Part,
) ([]*db.Part, []*db.PartLocation, []parsedSecondSource) {
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, nil, nil
	}

	var newParts []*db.Part
	var locationList []*db.PartLocation
	var secondSources []parsedSecondSource

	var currentMainPart *db.Part // 追蹤當前主料（用於判斷替代料）

	// 從 Row 6（Index 5）開始讀取資料列
	for i := 5; i < len(rows); i++ {
		row := rows[i]
		if len(row) == 0 {
			continue
		}

		// 欄位對應（EBOM 格式）：
		// A(0)=Item, B(1)=HHPN, E(4)=Description, F(5)=Supplier, G(6)=SupplierPN
		// H(7)=Qty, I(8)=Location, J(9)=CCL, L(11)=Remark
		item := safeGetCol(row, 0)
		hhpn := safeGetCol(row, 1)
		description := safeGetCol(row, 4)
		supplier := safeGetCol(row, 5)
		supplierPN := safeGetCol(row, 6)
		locationStr := safeGetCol(row, 8)
		cclVal := safeGetCol(row, 9)
		remark := safeGetCol(row, 11)

		// Supplier 與 SupplierPN 同時存在才視為有效零件資料
		if supplier == "" || supplierPN == "" {
			continue
		}

		if item != "" {
			// 主料（Main Source）：item 欄位有值
			key := supplier + "|" + supplierPN
			part, exists := partMap[key]
			if !exists {
				// 此 (supplier, supplier_pn) 尚未出現：建立新 Part
				part = &db.Part{
					RevisionID:  r.revisionID,
					Type:        sheetType,
					Item:        item,
					HHPN:        hhpn,
					Supplier:    supplier,
					SupplierPN:  supplierPN,
					Description: description,
					Cost:        parseCostStr(safeGetCol(row, 10)),
					Remark:      remark,
				}
				partMap[key] = part
				*partList = append(*partList, part)
				newParts = append(newParts, part)
			}
			currentMainPart = part

			// CCL 欄位判斷：Y/y 表示此物料為 CCL
			isCCL := strings.EqualFold(cclVal, "Y")

			// 原子化 location 並建立 PartLocation
			for _, loc := range atomizeLocations(locationStr) {
				locationList = append(locationList, &db.PartLocation{
					// PartID 在儲存後才能回填，此處透過 part 指標追蹤
					Location:  loc,
					BomStatus: "I",
					CCL:       isCCL,
				})
				// 暫存對應的 Part 指標，儲存後再填入 PartID
				_ = part // 後續在 saveParts 中透過 locationList 與 partList 的對應關係填入
			}

			// 將 location 與 part 的對應關係儲存在 locationList 中
			// 為了能在 saveParts 填入 PartID，我們在結構體擴展欄位中暫存 part 指標
			// 但 db.PartLocation 沒有 Part 指標欄位 → 改用 parsedPartLocation 內部結構
			// 重新整理：使用 parsedPartLocation 取代 *db.PartLocation
			// （見下方 parsedPartLocation 結構定義及重構）

		} else if currentMainPart != nil {
			// 替代料（2nd Source）：item 為空且跟隨主料
			source := db.SecondSource{
				RevisionID:  r.revisionID,
				HHPN:        hhpn,
				Supplier:    supplier,
				SupplierPN:  supplierPN,
				Description: description,
				Remark:      remark,
			}
			secondSources = append(secondSources, parsedSecondSource{
				partPtr:      currentMainPart,
				secondSource: source,
			})
		}
	}

	return newParts, locationList, secondSources
}

// parsedPartLocation 內部結構：暫存原子化 location 及其關聯的 Part 指標
// 用於在批次儲存 Part 後，正確填入 PartID
type parsedPartLocation struct {
	partPtr  *db.Part
	location string
	bomStatus string
	ccl       bool
}

// parseMainSheetV2 解析主製程 sheet，回傳 parsedPartLocation 列表（修正版）
// 此函數取代 parseMainSheet 中關於 locationList 的部分，以正確追蹤 Part 指標
func (r *EBOMReader) parseMainSheetV2(
	f Workbook,
	sheetName, sheetType string,
	partMap map[string]*db.Part,
	partList *[]*db.Part,
) ([]parsedPartLocation, []parsedSecondSource) {
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, nil
	}

	var locationList []parsedPartLocation
	var secondSources []parsedSecondSource
	var currentMainPart *db.Part

	for i := 5; i < len(rows); i++ {
		row := rows[i]
		if len(row) == 0 {
			continue
		}

		item := safeGetCol(row, 0)
		hhpn := safeGetCol(row, 1)
		description := safeGetCol(row, 4)
		supplier := safeGetCol(row, 5)
		supplierPN := safeGetCol(row, 6)
		locationStr := safeGetCol(row, 8)
		cclVal := safeGetCol(row, 9)
		remark := safeGetCol(row, 11)

		if supplier == "" || supplierPN == "" {
			continue
		}

		if item != "" {
			// 主料
			key := supplier + "|" + supplierPN
			part, exists := partMap[key]
			if !exists {
				part = &db.Part{
					RevisionID:  r.revisionID,
					Type:        sheetType,
					Item:        item,
					HHPN:        hhpn,
					Supplier:    supplier,
					SupplierPN:  supplierPN,
					Description: description,
					Cost:        parseCostStr(safeGetCol(row, 10)),
					Remark:      remark,
				}
				partMap[key] = part
				*partList = append(*partList, part)
			}
			currentMainPart = part

			isCCL := strings.EqualFold(cclVal, "Y")
			for _, loc := range atomizeLocations(locationStr) {
				locationList = append(locationList, parsedPartLocation{
					partPtr:   part,
					location:  loc,
					bomStatus: "I",
					ccl:       isCCL,
				})
			}

			if r.logger != nil {
				r.logger.Debug(fmt.Sprintf("[EBOM Phase1] 主料: %s / %s, type=%s, locations=%s", supplier, supplierPN, sheetType, locationStr))
			}
		} else if currentMainPart != nil {
			// 替代料
			source := db.SecondSource{
				RevisionID:  r.revisionID,
				HHPN:        hhpn,
				Supplier:    supplier,
				SupplierPN:  supplierPN,
				Description: description,
				Remark:      remark,
			}
			secondSources = append(secondSources, parsedSecondSource{
				partPtr:      currentMainPart,
				secondSource: source,
			})

			if r.logger != nil {
				r.logger.Debug(fmt.Sprintf("[EBOM Phase1] 替代料: %s / %s", supplier, supplierPN))
			}
		}
	}

	return locationList, secondSources
}

// parseNISheet 解析 NI sheet（不上件），建立 PartLocation（BomStatus='X'）。
// 若 (supplier, supplier_pn) 在 partMap 中已存在，重用既有 Part；否則建立新 Part（Type=''）。
func (r *EBOMReader) parseNISheet(
	f Workbook,
	sheetName string,
	partMap map[string]*db.Part,
	partList *[]*db.Part,
) []parsedPartLocation {
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil
	}

	var locationList []parsedPartLocation

	for i := 5; i < len(rows); i++ {
		row := rows[i]
		if len(row) == 0 {
			continue
		}

		hhpn := safeGetCol(row, 1)
		description := safeGetCol(row, 4)
		supplier := safeGetCol(row, 5)
		supplierPN := safeGetCol(row, 6)
		locationStr := safeGetCol(row, 8)

		if supplier == "" || supplierPN == "" {
			continue
		}

		key := supplier + "|" + supplierPN
		part, exists := partMap[key]
		if !exists {
			// NI 物料在主製程中未出現，建立新 Part（Type 為空，表示無製程類型）
			part = &db.Part{
				RevisionID:  r.revisionID,
				Type:        "",
				HHPN:        hhpn,
				Supplier:    supplier,
				SupplierPN:  supplierPN,
				Description: description,
			}
			partMap[key] = part
			*partList = append(*partList, part)
		}

		for _, loc := range atomizeLocations(locationStr) {
			locationList = append(locationList, parsedPartLocation{
				partPtr:   part,
				location:  loc,
				bomStatus: "X",
				ccl:       false,
			})
		}

		if r.logger != nil {
			r.logger.Debug(fmt.Sprintf("[EBOM Phase1] NI 物料: %s / %s, locations=%s", supplier, supplierPN, locationStr))
		}
	}

	return locationList
}

// parsePhase2Sheet 解析 Phase 2 狀態 sheet（PROTO / MP / CCL），僅收集 location 字串清單。
// Phase 2 不建立新物料，只回傳要更新的 location 列表。
func (r *EBOMReader) parsePhase2Sheet(f Workbook, sheetName string) []string {
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil
	}

	var locations []string
	for i := 5; i < len(rows); i++ {
		row := rows[i]
		if len(row) == 0 {
			continue
		}
		supplier := safeGetCol(row, 5)
		supplierPN := safeGetCol(row, 6)
		if supplier == "" || supplierPN == "" {
			continue
		}
		locationStr := safeGetCol(row, 8)
		locations = append(locations, atomizeLocations(locationStr)...)
	}
	return locations
}

// parseCostStr 解析 Cost 字串為 float64，解析失敗回傳 0
func parseCostStr(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

// ─── 儲存函數 ────────────────────────────────────────────────────────────────

// saveParts 將 Phase 1 解析到的 Part / PartLocation / SecondSource 儲存至資料庫。
//
// 流程：
//  1. 刪除此 Revision 的舊 Part（CASCADE 自動刪除 PartLocations）
//  2. 批次插入去重後的 Parts（GORM 回填 ID）
//  3. 依 Part 指標填入 PartID，批次插入 PartLocations
//  4. 依 Part 指標填入 PartID，批次插入 SecondSources
func (r *EBOMReader) saveParts(
	partList []*db.Part,
	locationList []parsedPartLocation,
	parsedSSList []parsedSecondSource,
) ([]db.PartLocation, []db.SecondSource, error) {
	if r.revisionID == 0 {
		return nil, nil, errors.New("revision ID not set")
	}

	// Step 1: 刪除舊資料（CASCADE 自動刪除 PartLocations）
	if err := r.db.Where("revision_id = ?", r.revisionID).Delete(&db.Part{}).Error; err != nil {
		return nil, nil, fmt.Errorf("刪除舊 Part 失敗: %w", err)
	}

	// Step 2: 批次插入 Parts（GORM 回填 ID）
	if len(partList) > 0 {
		// 轉為非指標切片以便 GORM CreateInBatches
		parts := make([]db.Part, len(partList))
		for i, p := range partList {
			p.RevisionID = r.revisionID
			parts[i] = *p
		}
		if err := r.db.CreateInBatches(&parts, 500).Error; err != nil {
			return nil, nil, fmt.Errorf("批次插入 Part 失敗: %w", err)
		}
		// 回填 ID 至 partList 指標（供後續 PartLocation / SecondSource 使用）
		for i, p := range partList {
			p.ID = parts[i].ID
		}
	}

	// Step 3: 填入 PartID 並批次插入 PartLocations
	locations := make([]db.PartLocation, 0, len(locationList))
	for _, pl := range locationList {
		if pl.partPtr == nil || pl.partPtr.ID == 0 {
			continue
		}
		locations = append(locations, db.PartLocation{
			PartID:    pl.partPtr.ID,
			Location:  pl.location,
			BomStatus: pl.bomStatus,
			CCL:       pl.ccl,
		})
	}
	if err := db.CreatePartLocationsInBatch(r.db, locations); err != nil {
		return nil, nil, fmt.Errorf("批次插入 PartLocation 失敗: %w", err)
	}

	// Step 4: 填入 PartID 並批次插入 SecondSources
	secondSources := make([]db.SecondSource, 0, len(parsedSSList))
	for _, pss := range parsedSSList {
		if pss.partPtr == nil || pss.partPtr.ID == 0 {
			continue
		}
		ss := pss.secondSource
		ss.RevisionID = r.revisionID
		ss.PartID = pss.partPtr.ID
		secondSources = append(secondSources, ss)
	}
	if len(secondSources) > 0 {
		if err := r.db.CreateInBatches(&secondSources, 500).Error; err != nil {
			return nil, nil, fmt.Errorf("批次插入 SecondSource 失敗: %w", err)
		}
	}

	if r.logger != nil {
		r.logger.Info("[EBOM] 資料儲存完成",
			"parts", len(partList),
			"locations", len(locations),
			"secondSources", len(secondSources),
		)
	}

	return locations, secondSources, nil
}

// ─── Revision 管理 ───────────────────────────────────────────────────────────

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

// ─── Merge 演算法（替代料 diff）───────────────────────────────────────────────

// applyMergeAlgorithm 執行替代料 diff（重新匯入時保持使用者對替代料的手動修改）
func (r *EBOMReader) applyMergeAlgorithm(revisionID int64, newSecondSources []db.SecondSource) error {
	// 載入舊的替代料
	var oldSecondSources []db.SecondSource
	if err := r.db.Where("revision_id = ?", revisionID).Find(&oldSecondSources).Error; err != nil {
		return err
	}

	// 建立舊替代料的 group 映射（主料 groupKey → []SecondSource）
	oldGroups := make(map[string][]db.SecondSource)
	for _, ss := range oldSecondSources {
		var mainPart db.Part
		if err := r.db.Where("id = ?", ss.PartID).First(&mainPart).Error; err != nil {
			if r.logger != nil {
				r.logger.Warn("Diff 替代料時未找到對應主料", "partID", ss.PartID, "error", err)
			}
			continue
		}
		key := mainPart.Supplier + "|" + mainPart.SupplierPN
		oldGroups[key] = append(oldGroups[key], ss)
	}

	// 建立新替代料的 group 映射
	newGroups := make(map[string][]db.SecondSource)
	for _, ss := range newSecondSources {
		var mainPart db.Part
		if err := r.db.Where("id = ?", ss.PartID).First(&mainPart).Error; err != nil {
			if r.logger != nil {
				r.logger.Warn("Diff 新替代料時未找到對應主料", "partID", ss.PartID, "error", err)
			}
			continue
		}
		key := mainPart.Supplier + "|" + mainPart.SupplierPN
		newGroups[key] = append(newGroups[key], ss)
	}

	var groupsToDelete []string
	var secondSourcesToDelete []db.SecondSource
	var secondSourcesToUpdate []db.SecondSource
	var secondSourcesToCreate []db.SecondSource

	for key, oldSrcs := range oldGroups {
		if _, exists := newGroups[key]; !exists {
			groupsToDelete = append(groupsToDelete, key)
			secondSourcesToDelete = append(secondSourcesToDelete, oldSrcs...)
		} else {
			newSrcs := newGroups[key]
			oldMap := make(map[string]db.SecondSource)
			for _, ss := range oldSrcs {
				oldMap[ss.Supplier+"|"+ss.SupplierPN] = ss
			}
			newMap := make(map[string]db.SecondSource)
			for _, ss := range newSrcs {
				newMap[ss.Supplier+"|"+ss.SupplierPN] = ss
			}
			for srcKey, oldSS := range oldMap {
				if _, exists := newMap[srcKey]; !exists {
					secondSourcesToDelete = append(secondSourcesToDelete, oldSS)
				}
			}
			for srcKey, newSS := range newMap {
				if _, exists := oldMap[srcKey]; !exists {
					secondSourcesToCreate = append(secondSourcesToCreate, newSS)
				} else {
					oldSS := oldMap[srcKey]
					oldSS.Description = newSS.Description
					oldSS.HHPN = newSS.HHPN
					secondSourcesToUpdate = append(secondSourcesToUpdate, oldSS)
				}
			}
		}
	}

	if len(secondSourcesToDelete) > 0 {
		ids := make([]int64, len(secondSourcesToDelete))
		for i, ss := range secondSourcesToDelete {
			ids[i] = ss.ID
		}
		if err := r.db.Where("id IN ?", ids).Delete(&db.SecondSource{}).Error; err != nil {
			return err
		}
	}
	for _, ss := range secondSourcesToUpdate {
		if err := r.db.Save(&ss).Error; err != nil {
			return err
		}
	}
	if len(secondSourcesToCreate) > 0 {
		if err := r.db.CreateInBatches(&secondSourcesToCreate, 500).Error; err != nil {
			return err
		}
	}

	// 清理無效的 MatrixSelection
	var removedMaterials []string
	for _, ss := range secondSourcesToDelete {
		removedMaterials = append(removedMaterials, ss.Supplier+"|"+ss.SupplierPN)
	}
	return db.DeleteInvalidSelections(r.db, revisionID, groupsToDelete, removedMaterials)
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
				"(ProjectID=%d, Phase=%s, Version=%s)",
				revision.ProjectID, revision.Phase, revision.Version,
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
				"[autoImportPreviousMatrix] 同 Phase 無前一版可繼承 (ProjectID=%d, Phase=%s, Version=%s)",
				revision.ProjectID, revision.Phase, revision.Version,
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
