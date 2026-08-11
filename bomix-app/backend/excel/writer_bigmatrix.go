package excel

import (
	"fmt"
	"strconv"
	"strings"

	"bomix-app/backend/types"

	"github.com/xuri/excelize/v2"
)

// exportBigMatrix exports data to BigMatrix format
// See product-spec section 8.1
func (w *WriterImpl) exportBigMatrix(options ExportOptions) ([]string, error) {
	if w.logger != nil {
		w.logger.Info(fmt.Sprintf("[exportBigMatrix] 開始產生 BigMatrix 匯出檔 (Revisions 數量: %d, PartData 數量: %d)", len(options.Revisions), len(options.PartData)))
	}
	return w.exportBigMatrixDetailed(options, options.Revisions, options.PartData)
}

// applyTagReplacement applies tag replacement to the template
// Scans all cells and replaces tag placeholders with actual values
func applyTagReplacement(f *excelize.File, tags map[string]string) error {
	sheets := f.GetSheetList()

	for _, sheet := range sheets {
		rows, err := f.GetRows(sheet)
		if err != nil {
			return err
		}
		for i, row := range rows {
			for j, cell := range row {
				cellValue := cell
				replaced := false
				for tag, value := range tags {
					if strings.Contains(cellValue, tag) {
						cellValue = strings.ReplaceAll(cellValue, tag, value)
						replaced = true
					}
				}
				if replaced {
					colName := getColName(j)
					rowNum := i + 1
					f.SetCellValue(sheet, fmt.Sprintf("%s%d", colName, rowNum), cellValue)
				}
			}
		}
	}

	return nil
}

// getColName converts column index to Excel column name (0 -> A, 1 -> B, etc.)
func getColName(col int) string {
	if col < 26 {
		return string(rune('A' + col))
	}
	return fmt.Sprintf("%c%c", 'A'+col/26-1, 'A'+col%26)
}

// colNameToIndex converts Excel column name to index (A -> 0, B -> 1, etc.)
func colNameToIndex(colName string) int {
	result := 0
	for _, c := range colName {
		result = result*26 + int(c-'A') + 1
	}
	return result - 1
}

// applyRowStyle applies a style to all cells in a row (columns A-J)
func applyRowStyle(f *excelize.File, sheet string, row int, styleID int) {
	for col := 'A'; col <= 'J'; col++ {
		cell := fmt.Sprintf("%c%d", col, row)
		f.SetCellStyle(sheet, cell, cell, styleID)
	}
}

// writeModelSelections writes model selection markers ("V") to the appropriate cells
func writeModelSelections(f *excelize.File, sheet string, row int, modelStartCol int, selections map[string]string, modelNames []string) {
	for i, modelName := range modelNames {
		col := getColName(modelStartCol + i)
		cell := fmt.Sprintf("%s%d", col, row)

		if selectedPN, ok := selections[modelName]; ok && selectedPN != "" {
			f.SetCellValue(sheet, cell, "V")
		}
	}
}

// exportBigMatrixDetailed is an extended version with full data population
// See product-spec sections 8.1.2 - 8.1.7
func (w *WriterImpl) exportBigMatrixDetailed(options ExportOptions, revisions []RevisionData, parts []PartData) ([]string, error) {
	if w.logger != nil {
		w.logger.Info(fmt.Sprintf("[exportBigMatrixDetailed] 開始詳細產生 BigMatrix 匯出檔 (Revisions: %d, Parts: %d)", len(revisions), len(parts)))
	}

	// Load template
	f, err := w.templateManager.LoadTemplate(types.FormatBigMatrix)
	if err != nil {
		if w.logger != nil {
			w.logger.Error(fmt.Sprintf("[exportBigMatrixDetailed] 載入 BigMatrix 範本失敗: %v", err))
		}
		return nil, fmt.Errorf("failed to load BigMatrix template: %w", err)
	}
	defer f.Close()

	date := generateTimestamp()

	// 8.1.2 - Tag replacement for header
	tags := map[string]string{
		"{{.BOMCount}}":    fmt.Sprintf("%d", len(options.RevisionIDs)),
		"{{.Description}}": options.Description,
		"{{.Date}}":        date,
	}
	if len(revisions) > 0 {
		tags["{{.BOMCount}}"] = fmt.Sprintf("%d", len(revisions))
	}

	if w.logger != nil {
		w.logger.Debug(fmt.Sprintf("[exportBigMatrixDetailed] 替換標籤內容: %+v", tags))
		for idx, rev := range revisions {
			w.logger.Debug(fmt.Sprintf("[exportBigMatrixDetailed] 匯出 BOM Revision [%d]: ID=%s, ProjectCode=%s, Phase=%s, Version=%s, ModelQty=%+v", idx, rev.ID, rev.ProjectCode, rev.Phase, rev.Version, rev.ModelQty))
		}
	}

	if err := applyTagReplacement(f, tags); err != nil {
		if w.logger != nil {
			w.logger.Error(fmt.Sprintf("[exportBigMatrixDetailed] 替換標籤失敗: %v", err))
		}
		return nil, err
	}

	// Read archetype style IDs from template for H, I, J columns (Header rows 1-5)
	styleH1, _ := f.GetCellStyle("BigMatrix", "H1")
	styleH2, _ := f.GetCellStyle("BigMatrix", "H2")
	styleH3, _ := f.GetCellStyle("BigMatrix", "H3")
	styleH4, _ := f.GetCellStyle("BigMatrix", "H4")
	styleH5, _ := f.GetCellStyle("BigMatrix", "H5")

	styleI1, _ := f.GetCellStyle("BigMatrix", "I1")
	styleI2, _ := f.GetCellStyle("BigMatrix", "I2")
	styleI3, _ := f.GetCellStyle("BigMatrix", "I3")
	styleI4, _ := f.GetCellStyle("BigMatrix", "I4")
	styleI5, _ := f.GetCellStyle("BigMatrix", "I5")

	styleJ1, _ := f.GetCellStyle("BigMatrix", "J1")
	styleJ2, _ := f.GetCellStyle("BigMatrix", "J2")
	styleJ3, _ := f.GetCellStyle("BigMatrix", "J3")
	styleJ4, _ := f.GetCellStyle("BigMatrix", "J4")
	styleJ5, _ := f.GetCellStyle("BigMatrix", "J5")

	// Read archetype style IDs for data rows (Row 6 = even, Row 7 = odd)
	styleH6, _ := f.GetCellStyle("BigMatrix", "H6")
	styleH7, _ := f.GetCellStyle("BigMatrix", "H7")
	styleI6, _ := f.GetCellStyle("BigMatrix", "I6")
	styleI7, _ := f.GetCellStyle("BigMatrix", "I7")
	styleJ6, _ := f.GetCellStyle("BigMatrix", "J6")
	styleJ7, _ := f.GetCellStyle("BigMatrix", "J7")

	// Read archetype style IDs for A-G columns in data rows
	styleAG6 := make(map[string]int)
	styleAG7 := make(map[string]int)
	for c := 'A'; c <= 'G'; c++ {
		colStr := string(c)
		s6, _ := f.GetCellStyle("BigMatrix", colStr+"6")
		s7, _ := f.GetCellStyle("BigMatrix", colStr+"7")
		styleAG6[colStr] = s6
		styleAG7[colStr] = s7
	}

	// unlockCache 樣式解鎖快取：以原始 styleID 為鍵，快取 Locked: false 版本的 styleID。
	// 避免對同一基礎樣式重複呼叫 NewStyle，減少樣式表膨脹。
	unlockCache := make(map[int]int)

	// protoStyleCache 樣式快取：以原始 styleID 為鍵，快取 Font.Color: "#8080C0" 版本的 styleID。
	protoStyleCache := make(map[int]int)

	// makeProtoStyle 建立並快取指定樣式的 PROTO 物料文字顏色（Font.Color: "#8080C0"）版本。
	// 當物料群組 (main + 2nd sources) 的 location 均屬於 PROTO 物料 (bom_status = P) 時套用。
	//
	// 參數：
	//   - styleID：原始 Excelize 樣式 ID
	//
	// 回傳：
	//   - int：PROTO 物料文字顏色版本的 Excelize 樣式 ID（若失敗則回傳原始 styleID）
	makeProtoStyle := func(styleID int) int {
		if styleID <= 0 {
			return styleID
		}
		if cached, ok := protoStyleCache[styleID]; ok {
			return cached
		}
		styleDef, err := f.GetStyle(styleID)
		if err != nil || styleDef == nil {
			return styleID
		}
		cp := *styleDef
		if cp.Font != nil {
			fontCopy := *cp.Font
			fontCopy.Color = "#8080C0"
			cp.Font = &fontCopy
		} else {
			cp.Font = &excelize.Font{
				Color: "#8080C0",
			}
		}
		newID, err := f.NewStyle(&cp)
		if err != nil || newID <= 0 {
			protoStyleCache[styleID] = styleID
			return styleID
		}
		protoStyleCache[styleID] = newID
		return newID
	}

	// makeUnlocked 建立並快取指定樣式的解鎖（Protection.Locked: false）版本。
	// 確保工作表保護啟用時，非灰底儲存格允許使用者自由編輯。
	//
	// 參數：
	//   - styleID：原始 Excelize 樣式 ID
	//
	// 回傳：
	//   - int：解鎖版本的 Excelize 樣式 ID（若失敗則回傳原始 styleID）
	makeUnlocked := func(styleID int) int {
		if styleID <= 0 {
			return styleID
		}
		if cached, ok := unlockCache[styleID]; ok {
			return cached
		}
		styleDef, err := f.GetStyle(styleID)
		if err != nil || styleDef == nil {
			return styleID
		}
		cp := *styleDef
		cp.Protection = &excelize.Protection{Locked: false}
		newID, err := f.NewStyle(&cp)
		if err != nil || newID <= 0 {
			unlockCache[styleID] = styleID
			return styleID
		}
		unlockCache[styleID] = newID
		return newID
	}

	// 解鎖 BigMatrix 範本靜態區域（行 1-7，欄 A-G）的所有儲存格。
	// Excel 預設儲存格 Locked: true，若不明確解鎖，啟用工作表保護後這些欄位也會被鎖定。
	for row := 1; row <= 7; row++ {
		for c := 'A'; c <= 'G'; c++ {
			cellAddr := fmt.Sprintf("%c%d", c, row)
			sid, _ := f.GetCellStyle("BigMatrix", cellAddr)
			newSid := makeUnlocked(sid)
			if newSid != sid {
				_ = f.SetCellStyle("BigMatrix", cellAddr, cellAddr, newSid)
			}
		}
	}

	// Helper to get archetype style for a model column based on index and total count
	getModelStyle := func(row int, colIdx int, totalCount int) int {
		if row == 6 || row == 7 {
			if row == 6 {
				if totalCount == 1 || colIdx == totalCount-1 {
					if colIdx == 0 && totalCount > 1 {
						return styleH6
					}
					return styleJ6
				}
				if colIdx == 0 {
					return styleH6
				}
				return styleI6
			} else {
				if totalCount == 1 || colIdx == totalCount-1 {
					if colIdx == 0 && totalCount > 1 {
						return styleH7
					}
					return styleJ7
				}
				if colIdx == 0 {
					return styleH7
				}
				return styleI7
			}
		}
		// Header rows 1..5
		switch row {
		case 1:
			if colIdx == 0 {
				return styleH1
			} else if colIdx == totalCount-1 {
				return styleJ1
			}
			return styleI1
		case 2:
			if colIdx == 0 {
				return styleH2
			} else if colIdx == totalCount-1 {
				return styleJ2
			}
			return styleI2
		case 3:
			if colIdx == 0 {
				return styleH3
			} else if colIdx == totalCount-1 {
				return styleJ3
			}
			return styleI3
		case 4:
			if colIdx == 0 {
				return styleH4
			} else if colIdx == totalCount-1 {
				return styleJ4
			}
			return styleI4
		case 5:
			if colIdx == 0 {
				return styleH5
			} else if colIdx == totalCount-1 {
				return styleJ5
			}
			return styleI5
		}
		return styleI5
	}

	// Start column for BOM data (H = column 7)
	bomStartCol := 7 // H

	// Unmerge template default merged header cells (H2:J2, H3:J3) to allow dynamic revision layout
	_ = f.UnmergeCell("BigMatrix", "H2", "J2")
	_ = f.UnmergeCell("BigMatrix", "H3", "J3")

	// Write dynamic Model columns and header info for each revision
	currentCol := bomStartCol
	for _, rev := range revisions {
		revModelCount := getRevisionModelCount(rev, parts, options.ModelCountOverrides[rev.ID])
		if revModelCount <= 0 {
			continue
		}

		startColName := getColName(currentCol)
		endColName := getColName(currentCol + revModelCount - 1)

		// Apply archetype styles for each column in this revision (rows 2..5)
		// 使用解鎖版樣式，確保 Header 欄在工作表保護時不被誤鎖定
		for i := 0; i < revModelCount; i++ {
			col := getColName(currentCol + i)
			for r := 2; r <= 5; r++ {
				st := makeUnlocked(getModelStyle(r, i, revModelCount))
				_ = f.SetCellStyle("BigMatrix", fmt.Sprintf("%s%d", col, r), fmt.Sprintf("%s%d", col, r), st)
			}

			// Model name (row 4)
			mName := resolveModelName(rev, i)
			f.SetCellValue("BigMatrix", fmt.Sprintf("%s4", col), mName)

			// Model quantity (row 5)
			orderedNames := getOrderedModelNames(rev.ModelQty)
			qty := resolveModelQty(rev, i, orderedNames)
			if qty > 0 {
				f.SetCellValue("BigMatrix", fmt.Sprintf("%s5", col), qty)
			} else {
				f.SetCellValue("BigMatrix", fmt.Sprintf("%s5", col), "")
			}
		}

		// Write Project Code (row 2)
		if rev.ProjectCode != "" {
			f.SetCellValue("BigMatrix", fmt.Sprintf("%s2", startColName), rev.ProjectCode)
		}

		// Write Phase-Version (row 3)
		var phaseVer string
		if rev.Phase != "" && rev.Version != "" {
			phaseVer = fmt.Sprintf("%s-%s", rev.Phase, rev.Version)
		} else if rev.Phase != "" {
			phaseVer = rev.Phase
		} else {
			phaseVer = rev.Version
		}
		if phaseVer != "" {
			f.SetCellValue("BigMatrix", fmt.Sprintf("%s3", startColName), phaseVer)
		}

		// Merge header cells for this revision if it spans multiple model columns
		if revModelCount > 1 {
			_ = f.MergeCell("BigMatrix", fmt.Sprintf("%s2", startColName), fmt.Sprintf("%s2", endColName))
			_ = f.MergeCell("BigMatrix", fmt.Sprintf("%s3", startColName), fmt.Sprintf("%s3", endColName))
		}

		currentCol += revModelCount
	}

	// Calculate and apply uniform column width for all model columns (H onwards)
	maxColWidthReq := 3.0
	padding := 0.5 // 欄寬邊距 (margin) 設定為 1.0

	for _, rev := range revisions {
		revModelCount := getRevisionModelCount(rev, parts, options.ModelCountOverrides[rev.ID])
		if revModelCount <= 0 {
			continue
		}

		// 1. Project Code (Row 2) width requirement per column (considering merged columns)
		if rev.ProjectCode != "" {
			req := (float64(len(rev.ProjectCode)) + padding) / float64(revModelCount)
			if req > maxColWidthReq {
				maxColWidthReq = req
			}
		}

		// 2. Phase-Version (Row 3) width requirement per column (considering merged columns)
		var phaseVer string
		if rev.Phase != "" && rev.Version != "" {
			phaseVer = fmt.Sprintf("%s-%s", rev.Phase, rev.Version)
		} else if rev.Phase != "" {
			phaseVer = rev.Phase
		} else {
			phaseVer = rev.Version
		}
		if phaseVer != "" {
			req := (float64(len(phaseVer)) + padding) / float64(revModelCount)
			if req > maxColWidthReq {
				maxColWidthReq = req
			}
		}

		// 3. Model Qty (Row 5) width requirement per column (single cell)
		orderedNames := getOrderedModelNames(rev.ModelQty)
		for i := 0; i < revModelCount; i++ {
			qty := resolveModelQty(rev, i, orderedNames)
			if qty > 0 {
				req := float64(len(fmt.Sprintf("%d", qty))) + padding
				if req > maxColWidthReq {
					maxColWidthReq = req
				}
			}
		}
	}

	totalModelCols := currentCol - bomStartCol
	if totalModelCols > 0 {
		startColStr := getColName(bomStartCol)
		endColStr := getColName(bomStartCol + totalModelCols - 1)
		_ = f.SetColWidth("BigMatrix", startColStr, endColStr, maxColWidthReq)
	}

	// Helper function to apply styles to a row (columns A-G and dynamic Model columns)
	applyFullRowStyle := func(f *excelize.File, sheet string, row int, isEven bool, isProto bool) {
		refRow := 6
		if !isEven {
			refRow = 7
		}

		// A-G 欄使用解鎖版樣式：確保資料欄在工作表保護模式下仍可自由編輯
		for c := 'A'; c <= 'G'; c++ {
			colStr := string(c)
			cell := fmt.Sprintf("%s%d", colStr, row)
			var baseSt int
			if isEven {
				baseSt = styleAG6[colStr]
			} else {
				baseSt = styleAG7[colStr]
			}
			st := makeUnlocked(baseSt)
			if isProto {
				st = makeProtoStyle(st)
			}
			// makeUnlocked 確保 A-G 欄不被工作表保護鎖定
			_ = f.SetCellStyle(sheet, cell, cell, st)
		}

		// Model 欄先套用基礎樣式（後續由 getGrayStyle / getUnlockedModelStyle 覆蓋）
		cIdx := bomStartCol
		for _, rev := range revisions {
			revModelCount := getRevisionModelCount(rev, parts, options.ModelCountOverrides[rev.ID])
			for i := 0; i < revModelCount; i++ {
				colStr := getColName(cIdx + i)
				cell := fmt.Sprintf("%s%d", colStr, row)
				st := getModelStyle(refRow, i, revModelCount)
				if isProto {
					st = makeProtoStyle(st)
				}
				_ = f.SetCellStyle(sheet, cell, cell, st)
			}
			cIdx += revModelCount
		}
	}

	// 灰色底色樣式快取：以 [4]bool{左粗黑, 右粗黑, 頂細黑, 底細黑} 為鍵，動態建立並快取。
	//
	// 邊框規則說明：
	//   - 左/右（Revision 邊界）：粗黑（Style: 2, Color: 000000）區隔不同 Revision
	//   - 頂/底（群組邊界）：  細黑（Style: 1, Color: 000000）區隔不同物料群組
	//   - 群組內部邊框：       淡灰細線（Style: 1, Color: BFBFBF）
	grayStyleCache := make(map[[4]bool]int)

	// createGrayStyle 依據 [左粗, 右粗, 頂黑, 底黑] 四個邊框旗標，動態建立並快取灰色底色樣式。
	//
	// 參數：
	//   - leftThick：true = 左側粗黑邊框（Revision 最左欄），false = 淡灰細線
	//   - rightThick：true = 右側粗黑邊框（Revision 最右欄），false = 淡灰細線
	//   - topBlack：true = 頂部細黑邊框（群組第一列），false = 淡灰細線
	//   - bottomBlack：true = 底部細黑邊框（群組最後列），false = 淡灰細線
	//
	// 回傳：
	//   - int：Excelize 樣式 ID（-1 表示建立失敗）
	createGrayStyle := func(leftThick, rightThick, topBlack, bottomBlack bool) int {
		key := [4]bool{leftThick, rightThick, topBlack, bottomBlack}
		if id, ok := grayStyleCache[key]; ok {
			return id
		}
		// 左側邊框：Revision 最左欄用粗黑（Style: 2），其餘用淡灰細線
		leftColor, leftStyle := "BFBFBF", 1
		if leftThick {
			leftColor, leftStyle = "000000", 2
		}
		// 右側邊框：Revision 最右欄用粗黑（Style: 2），其餘用淡灰細線
		rightColor, rightStyle := "BFBFBF", 1
		if rightThick {
			rightColor, rightStyle = "000000", 2
		}
		// 頂側邊框：群組第一列用細黑（Color: 000000），其餘用淡灰細線
		topColor := "BFBFBF"
		if topBlack {
			topColor = "000000"
		}
		// 底側邊框：群組最後列用細黑（Color: 000000），其餘用淡灰細線
		bottomColor := "BFBFBF"
		if bottomBlack {
			bottomColor = "000000"
		}
		id, _ := f.NewStyle(&excelize.Style{
			Fill: excelize.Fill{Type: "pattern", Color: []string{"#D9D9D9"}, Pattern: 1},
			Border: []excelize.Border{
				{Type: "left", Color: leftColor, Style: leftStyle},
				{Type: "right", Color: rightColor, Style: rightStyle},
				{Type: "top", Color: topColor, Style: 1},
				{Type: "bottom", Color: bottomColor, Style: 1},
			},
			Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
			Protection: &excelize.Protection{
				Locked: true, // 鎖定：灰底儲存格禁止使用者修改
			},
		})
		if id < 0 {
			return -1
		}
		grayStyleCache[key] = id
		return id
	}

	// getGrayStyle 依據欄位位置與列位置，回傳具備正確邊框組合的灰色底色樣式。
	//
	// 參數：
	//   - colIdx：此欄在當前 Revision 的 0-based 欄位索引
	//   - totalCount：當前 Revision 的 Model 欄位總數
	//   - isTopEdge：此列是否為物料群組的第一列（主料列）
	//   - isBottomEdge：此列是否為物料群組的最後列（最後一筆 2nd source 列，或無 2nd source 的主料列）
	//
	// 回傳：
	//   - int：Excelize 樣式 ID（-1 表示建立失敗）
	getGrayStyle := func(colIdx, totalCount int, isTopEdge, isBottomEdge bool) int {
		isLeftEdge := (colIdx == 0)
		isRightEdge := (totalCount == 1 || colIdx == totalCount-1)
		return createGrayStyle(isLeftEdge, isRightEdge, isTopEdge, isBottomEdge)
	}

	// getUnlockedModelStyle 取得解鎖（Locked: false）版本的 Model 欄位樣式。
	// 由 makeUnlocked 統一處理快取（unlockCache），避免重複 NewStyle 呼叫。
	//
	// 參數：
	//   - row：資料列的斑馬紋基準行（6 = 偶數群組，7 = 奇數群組）
	//   - colIdx：此欄在當前 Revision 的 0-based 欄位索引
	//   - totalCount：當前 Revision 的 Model 欄位總數
	//
	// 回傳：
	//   - int：解鎖版本的 Excelize 樣式 ID
	getUnlockedModelStyle := func(row int, colIdx int, totalCount int) int {
		return makeUnlocked(getModelStyle(row, colIdx, totalCount))
	}

	// 建立 revision ID 字串 -> int64 的解析輔助（用於比對 SourceRevisionIDs）
	revIDToInt64 := make(map[string]int64, len(revisions))
	for _, rev := range revisions {
		var revIDInt int64
		if idVal, parseErr := strconv.ParseInt(strings.TrimSpace(rev.ID), 10, 64); parseErr == nil {
			revIDToInt64[rev.ID] = idVal
		} else if _, scanErr := fmt.Sscanf(rev.ID, "%d", &revIDInt); scanErr == nil {
			revIDToInt64[rev.ID] = revIDInt
		}
	}

	// revisionIDInList 判斷特定 revision 是否存在於 SourceRevisionIDs 列表中
	revisionIDInList := func(revIDStr string, sourceRevisionIDs []int64) bool {
		if len(sourceRevisionIDs) == 0 {
			return false
		}
		revIDInt, ok := revIDToInt64[revIDStr]
		if !ok {
			return false
		}
		for _, srcID := range sourceRevisionIDs {
			if srcID == revIDInt {
				return true
			}
		}
		return false
	}

	// partExistsInRevision 判定主料是否存在於指定的 revision 主料表中
	// (純粹比對該 Revision 的主料表 SourceRevisionIDs，與 Model 是否存在 selection 無關)
	partExistsInRevision := func(part PartData, revID string) bool {
		// 若全域未包含 SourceRevisionIDs (舊 DTO 或單一 Revision 測試)，視為存在
		if len(part.SourceRevisionIDs) == 0 {
			return true
		}
		return revisionIDInList(revID, part.SourceRevisionIDs)
	}

	// ssExistsInRevision 判定 2nd 物料是否存在於指定 revision 同群組的 2nd 物料表中
	ssExistsInRevision := func(part PartData, ss SecondSourceData, revID string) bool {
		// 規則 1：如果主料不存在於該 Revision 的主料表中，則整群組 (主料 + 2nd) 直接算不存在 (全灰底)
		if !partExistsInRevision(part, revID) {
			return false
		}
		// 規則 2：若全域 2nd 未包含 SourceRevisionIDs (舊 DTO)，視為存在
		if len(ss.SourceRevisionIDs) == 0 {
			return true
		}
		// 規則 3：檢查此 2nd 物料是否存在於該 Revision 同群組的 2nd 物料表中
		return revisionIDInList(revID, ss.SourceRevisionIDs)
	}

	// 8.1.4 - Write part data
	rowIndex := 6
	for groupIdx, part := range parts {
		// 依物料群組 (groupIdx) 切換斑馬紋樣式 (Row 6 / Row 7)
		isEven := (groupIdx%2 == 0)
		isProtoGroup := strings.EqualFold(part.BOMStatus, "P")
		applyFullRowStyle(f, "BigMatrix", rowIndex, isEven, isProtoGroup)

		// Write basic part data (columns A-G)
		if itemNum, err := strconv.Atoi(strings.TrimSpace(part.Item)); err == nil {
			f.SetCellValue("BigMatrix", fmt.Sprintf("A%d", rowIndex), itemNum)
		} else {
			f.SetCellValue("BigMatrix", fmt.Sprintf("A%d", rowIndex), part.Item)
		}
		f.SetCellValue("BigMatrix", fmt.Sprintf("B%d", rowIndex), part.HHPN)
		f.SetCellValue("BigMatrix", fmt.Sprintf("C%d", rowIndex), part.Description)
		f.SetCellValue("BigMatrix", fmt.Sprintf("D%d", rowIndex), part.Supplier)
		f.SetCellValue("BigMatrix", fmt.Sprintf("E%d", rowIndex), part.SupplierPn)
		f.SetCellValue("BigMatrix", fmt.Sprintf("F%d", rowIndex), part.Qty)
		f.SetCellValue("BigMatrix", fmt.Sprintf("G%d", rowIndex), part.Location)

		// 8.1.5.3 - Write Model selections
		// 若物料在某 BOM Revision 中不存在，將對應的所有 Model 欄位填入灰色底色
		currentCol = bomStartCol
		for _, rev := range revisions {
			revModelCount := getRevisionModelCount(rev, parts, options.ModelCountOverrides[rev.ID])

			// 精確判斷主料在當前 revision 中是否存在
			partExists := partExistsInRevision(part, rev.ID)

			for i := 0; i < revModelCount; i++ {
				col := getColName(currentCol + i)
				cell := fmt.Sprintf("%s%d", col, rowIndex)

				if !partExists {
					// 主料在此 BOM Revision 不存在：設定灰色底色，且禁止寫入任何資料（清空儲存格內容）
					// isTopEdge=true：主料列永遠是群組的第一列
					// isBottomEdge：若無 2nd Source，主料列同時也是群組最後列
					// See product-spec section 8.1.5.3
					isBottomEdge := len(part.SecondSources) == 0
					st := getGrayStyle(i, revModelCount, true, isBottomEdge)
					if isProtoGroup {
						st = makeProtoStyle(st)
					}
					if st >= 0 {
						_ = f.SetCellStyle("BigMatrix", cell, cell, st)
					}
					// 灰底儲存格禁止寫入任何資料，強制設為空字串
					f.SetCellValue("BigMatrix", cell, "")
				} else {
					// 物料存在：解鎖此儲存格 (Locked: false) 允許編輯，並判斷 Model 勾選狀態
					refRow := 6
					if !isEven {
						refRow = 7
					}
					st := getUnlockedModelStyle(refRow, i, revModelCount)
					if isProtoGroup {
						st = makeProtoStyle(st)
					}
					if st >= 0 {
						_ = f.SetCellStyle("BigMatrix", cell, cell, st)
					}

					mName := resolveModelName(rev, i)
					selectedPN := resolveBigMatrixSelectedPN(part, rev.ID, i, mName)
					if selectedPN != "" && strings.EqualFold(part.SupplierPn, selectedPN) {
						f.SetCellValue("BigMatrix", cell, "V")
					} else {
						f.SetCellValue("BigMatrix", cell, "")
					}
				}
			}

			currentCol += revModelCount
		}

		rowIndex++

		// Write second sources
		for ssIdx, ss := range part.SecondSources {
			isEvenSS := isEven
			applyFullRowStyle(f, "BigMatrix", rowIndex, isEvenSS, isProtoGroup)

			f.SetCellValue("BigMatrix", fmt.Sprintf("B%d", rowIndex), ss.HHPN)
			f.SetCellValue("BigMatrix", fmt.Sprintf("C%d", rowIndex), ss.Description)
			f.SetCellValue("BigMatrix", fmt.Sprintf("D%d", rowIndex), ss.Supplier)
			f.SetCellValue("BigMatrix", fmt.Sprintf("E%d", rowIndex), ss.SupplierPn)

			// 替代料也需要處理灰色底色
			// 規則：當主料在當前 rev 本身不存在，或該 2nd Source 不在對應 rev 中，填入灰色底色
			// isTopEdge=false：替代料列永遠不是群組第一列（主料列才是）
			// isBottomEdge：若為最後一筆替代料，則此列為群組最後列
			ssIsBottomEdge := ssIdx == len(part.SecondSources)-1
			cIdx := bomStartCol
			for _, rev := range revisions {
				revModelCount := getRevisionModelCount(rev, parts, options.ModelCountOverrides[rev.ID])

				// 精確判斷此 2nd Source 在當前 rev 中是否存在
				ssExists := ssExistsInRevision(part, ss, rev.ID)

				for i := 0; i < revModelCount; i++ {
					col := getColName(cIdx + i)
					cell := fmt.Sprintf("%s%d", col, rowIndex)

					if !ssExists {
						// 替代料在此 Revision 的群組中不存在：設定灰色底色，且禁止寫入任何資料（清空儲存格內容）
						st := getGrayStyle(i, revModelCount, false, ssIsBottomEdge)
						if isProtoGroup {
							st = makeProtoStyle(st)
						}
						if st >= 0 {
							_ = f.SetCellStyle("BigMatrix", cell, cell, st)
						}
						// 灰底儲存格禁止寫入任何資料，強制設為空字串
						f.SetCellValue("BigMatrix", cell, "")
					} else {
						// 替代料存在：解鎖此儲存格 (Locked: false) 允許編輯，並判斷 Model 勾選狀態
						refRow := 6
						if !isEvenSS {
							refRow = 7
						}
						st := getUnlockedModelStyle(refRow, i, revModelCount)
						if isProtoGroup {
							st = makeProtoStyle(st)
						}
						if st >= 0 {
							_ = f.SetCellStyle("BigMatrix", cell, cell, st)
						}

						mName := resolveModelName(rev, i)
						selectedPN := resolveBigMatrixSelectedPN(part, rev.ID, i, mName)
						if selectedPN != "" && strings.EqualFold(ss.SupplierPn, selectedPN) {
							f.SetCellValue("BigMatrix", cell, "V")
						} else {
							f.SetCellValue("BigMatrix", cell, "")
						}
					}
				}
				cIdx += revModelCount
			}

			rowIndex++
		}
	}

	// 寫入每個 Model 欄位 (start from H) 的 Row 1 計算公式：
	// 公式規則：cnt_M - cnt_N - COUNTIF(<Col>6:<Col><EndRow>, "V")
	//   - cnt_M：聚合後的 View 總共 main source 的數量 (len(parts))
	//   - cnt_N：該 BOM Revision 不存在的主料數量 (!partExistsInRevision)
	//   - COUNTIF：統計該欄位在 row 6 到最後有資料列 (endRow) 範圍內，存在 "V" 的總和
	cntM := len(parts)
	endRow := rowIndex - 1
	currentCol = bomStartCol
	for _, rev := range revisions {
		revModelCount := getRevisionModelCount(rev, parts, options.ModelCountOverrides[rev.ID])
		if revModelCount <= 0 {
			continue
		}

		// 計算當前 Revision 不存在的主料數量 cnt_N
		cntN := 0
		for _, part := range parts {
			if !partExistsInRevision(part, rev.ID) {
				cntN++
			}
		}

		for i := 0; i < revModelCount; i++ {
			colStr := getColName(currentCol + i)
			cell1 := fmt.Sprintf("%s1", colStr)

			// 設定 Row 1 解鎖樣式
			st := makeUnlocked(getModelStyle(1, i, revModelCount))
			if st > 0 {
				_ = f.SetCellStyle("BigMatrix", cell1, cell1, st)
			}

			// 寫入 Row 1 公式
			var formulaStr string
			if endRow >= 6 {
				formulaStr = fmt.Sprintf("%d-%d-COUNTIF(%s6:%s%d,\"V\")", cntM, cntN, colStr, colStr, endRow)
			} else {
				formulaStr = fmt.Sprintf("%d-%d", cntM, cntN)
			}
			_ = f.SetCellFormula("BigMatrix", cell1, formulaStr)
		}

		currentCol += revModelCount
	}

	// 動態擴充 Row 1 條件格式化範圍至所有 Model 欄位 (H1:endCol1)
	// 解決 Excelize 在動態新增欄位時不會自動擴展範本 sqref 條件格式化範圍的問題
	totalModelCols = currentCol - bomStartCol
	if totalModelCols > 0 {
		endColStr := getColName(bomStartCol + totalModelCols - 1)
		targetRange := fmt.Sprintf("H1:%s1", endColStr)
		if cfMap, err := f.GetConditionalFormats("BigMatrix"); err == nil && len(cfMap) > 0 {
			for origRange, cfOptsList := range cfMap {
				if strings.Contains(origRange, "H1") {
					_ = f.SetConditionalFormat("BigMatrix", targetRange, cfOptsList)
				}
			}
		}
	}

	// 對 Model 勾選區塊（以每個物料 group 內的單一 model column 為單位）設定條件格式化：
	// 檢查條件：
	//   a. 該 model column 區域內必須且只能有一個勾選 ("V" 或 "v")
	//   b. 儲存格內容必須為空白或是 "V" or "v"
	// 若不符合條件，將該區域底色設定為淡紅色 (#FFC7CE)，框線保持不變
	addModelSelectionConditionalFormatting(f, parts, revisions, options, bomStartCol)

	// 啟用 Excel 工作表保護（不設定密碼 / 空密碼，供使用者需要時自由取消保護）
	// 效果：灰底儲存格 (Locked: true) 禁止修改，可編輯儲存格 (Locked: false) 允許修改
	_ = f.ProtectSheet("BigMatrix", &excelize.SheetProtectionOptions{
		AlgorithmName:       "SHA-512",
		Password:            "",
		SelectLockedCells:   true, // 允許點選鎖定儲存格 (唯讀)
		SelectUnlockedCells: true, // 允許點選並編輯解鎖儲存格
		EditObjects:         true,
		EditScenarios:       true,
	})

	// Save to output path using validateAndPrepareOutputPath
	seriesName := "BOMIX"
	if len(revisions) > 0 && revisions[0].ProjectCode != "" {
		seriesName = revisions[0].ProjectCode
	}
	defaultFileName := generateBigMatrixFileName(seriesName, revisions, date)
	outputPath, err := validateAndPrepareOutputPath(w.logger, options.OutputPath, options.OutputDir, defaultFileName)
	if err != nil {
		return nil, err
	}

	if err := f.SaveAs(outputPath); err != nil {
		return nil, fmt.Errorf("failed to save BigMatrix: %w", err)
	}

	if w.logger != nil {
		w.logger.Info(fmt.Sprintf("[exportBigMatrixDetailed] 成功匯出 BigMatrix 檔案至: %s", outputPath))
	}

	return []string{outputPath}, nil
}

// FilterPartsByCriteria filters parts based on export criteria (CCL = Y, BOMStatus != X)
// See product-spec section 8.1.6
func FilterPartsByCriteria(parts []PartData) []PartData {
	var filtered []PartData

	for _, part := range parts {
		// Filter by CCL = true
		if !part.CCL {
			continue
		}

		// Filter by BOM status
		if part.BOMStatus == "X" {
			continue
		}

		filtered = append(filtered, part)
	}

	return filtered
}

// deduplicateParts removes duplicate parts based on supplier + supplier_pn
func deduplicateParts(parts []PartData) []PartData {
	seen := make(map[string]bool)
	var result []PartData

	for _, part := range parts {
		key := fmt.Sprintf("%s|%s", part.Supplier, part.SupplierPn)
		if !seen[key] {
			seen[key] = true
			result = append(result, part)
		}
	}

	return result
}

// mergeSecondSources merges second sources from multiple parts with same key
func mergeSecondSources(parts []PartData) []PartData {
	// Group by main supplier + supplier_pn
	groups := make(map[string][]PartData)

	for _, part := range parts {
		key := fmt.Sprintf("%s|%s", part.Supplier, part.SupplierPn)
		groups[key] = append(groups[key], part)
	}

	// Merge second sources within each group
	var result []PartData
	for _, group := range groups {
		if len(group) == 0 {
			continue
		}

		// Use first part as base (copy to avoid reference issues)
		merged := PartData{
			Item:        group[0].Item,
			HHPN:        group[0].HHPN,
			Description: group[0].Description,
			Supplier:    group[0].Supplier,
			SupplierPn:  group[0].SupplierPn,
			Qty:         group[0].Qty,
			Location:    group[0].Location,
			Type:        group[0].Type,
			BOMStatus:   group[0].BOMStatus,
			CCL:         group[0].CCL,
			Remark:      group[0].Remark,
			Selections:  make(map[string]string),
		}
		for k, v := range group[0].Selections {
			merged.Selections[k] = v
		}

		// Collect unique second sources from all parts in the group
		ssSeen := make(map[string]bool)
		for _, part := range group {
			for _, ss := range part.SecondSources {
				ssKey := fmt.Sprintf("%s|%s", ss.Supplier, ss.SupplierPn)
				if !ssSeen[ssKey] {
					ssSeen[ssKey] = true
					merged.SecondSources = append(merged.SecondSources, ss)
				}
			}
		}

		result = append(result, merged)
	}

	return result
}

// NormalizeLocationCount calculates the quantity from location string
func NormalizeLocationCount(loc string) int {
	if loc == "" {
		return 0
	}
	parts := strings.Split(loc, ",")
	return len(parts)
}

// resolveModelName 取得第 index 個 Model 的字母代號（A, B, C...）。
//
// BigMatrix 的設計原則是以匯入順序（SortOrder）決定 Model 排列順序，
// 資料庫的 ModelName 欄位為選填且不作為識別用途。
// 因此匯出時統一以 A, B, C... 的字母序號表示 Model 欄位，
// 不再讀取資料庫中儲存的 ModelName。
//
// 參數：
//   - rev：RevisionData（保留參數以維持介面一致性，本函式中不使用 ModelNames）
//   - index：Model 的 0-based 排序索引
//
// 回傳：
//   - string：字母代號，例如 index=0 → "A"，index=1 → "B"，index=25 → "Z"，index=26 → "AA"
func resolveModelName(_ RevisionData, index int) string {
	// 直接依 0-based 索引轉換為 Excel 欄名風格的字母序號
	return getColName(index)
}

// getRevisionModelCount 計算 BOM Revision 實際存在的 Model 數量
// 判斷原則：有 selection 或有 qty 就算存在
func getRevisionModelCount(rev RevisionData, parts []PartData, override int) int {
	if override > 0 {
		return override
	}

	count := len(rev.ModelNames)
	if len(rev.ModelQtyByOrder) > count {
		count = len(rev.ModelQtyByOrder)
	}
	if len(rev.ModelQty) > count {
		count = len(rev.ModelQty)
	}

	// 檢查所有物料在該 Revision 中的 Selection 最大 SortOrder
	for _, p := range parts {
		if revOrderMap, ok := p.SelectionsByRevAndOrder[rev.ID]; ok {
			for sortOrder, selectedPN := range revOrderMap {
				if selectedPN != "" {
					if sortOrder+1 > count {
						count = sortOrder + 1
					}
				}
			}
		}
	}

	return count
}

// resolveBigMatrixSelectedPN 獲取特定 Revision 下、特定 Model 的選取 PN
func resolveBigMatrixSelectedPN(part PartData, revID string, sortOrder int, modelName string) string {
	if revID != "" {
		if revOrderMap, ok := part.SelectionsByRevAndOrder[revID]; ok {
			if pn, ok2 := revOrderMap[sortOrder]; ok2 && pn != "" {
				return pn
			}
		}
		if revNameMap, ok := part.SelectionsByRevAndName[revID]; ok {
			if pn, ok2 := revNameMap[modelName]; ok2 && pn != "" {
				return pn
			}
		}
		// 若已提供多 Revision 映射 (SelectionsByRevAndOrder 或 SelectionsByRevAndName)，說明具有精確的跨 Revision 勾選集
		// 此時若該 revID 內無勾選，代表此 Revision 確實未勾選，絕不可降級 (防止多 Revision 輸出相同勾選)
		if len(part.SelectionsByRevAndOrder) > 0 || len(part.SelectionsByRevAndName) > 0 {
			return ""
		}
	}

	if pn, ok := part.SelectionsByOrder[sortOrder]; ok && pn != "" {
		return pn
	}
	if pn, ok := part.Selections[modelName]; ok && pn != "" {
		return pn
	}
	return ""
}

/**
 * addModelSelectionConditionalFormatting 對 BigMatrix Model 勾選區塊設定條件格式化。
 *
 * 以每一個物料 group 內的 Model 欄位區域為單位 (Range: H[startRow]:EndCol[endRow])。
 * 使用相對欄位位址 (H$startRow:H$endRow) 搭配全群組 Model 欄位範圍 (H:EndCol)，
 * 使 Excel 能以單一條件格式化規則，針對直向每個 Model Column 獨立進行判斷：
 *   a. 該 model column 區域內必須且只能有一個勾選 ("V" 或 "v")
 *   b. 儲存格內容必須為空白或是 "V" or "v"
 * 若不符合條件，該 Model Column 區域底色會自動呈現淡紅色 (#FFC7CE)，框線保持不變。
 *
 * @param f *excelize.File - Excelize 檔案物件
 * @param parts []PartData - 聚合後的物料清單
 * @param revisions []RevisionData - BOM Revision 清單
 * @param options ExportOptions - 匯出選項
 * @param bomStartCol int - Model 欄位起始欄索引 (0-based, 一般為 7 即 H 欄)
 */
func addModelSelectionConditionalFormatting(
	f *excelize.File,
	parts []PartData,
	revisions []RevisionData,
	options ExportOptions,
	bomStartCol int,
) {
	if len(parts) == 0 || len(revisions) == 0 {
		return
	}

	// 計算全體 BOM Revision 的 Model 欄位總數
	totalModelCols := 0
	for _, rev := range revisions {
		revModelCount := getRevisionModelCount(rev, parts, options.ModelCountOverrides[rev.ID])
		if revModelCount > 0 {
			totalModelCols += revModelCount
		}
	}
	if totalModelCols <= 0 {
		return
	}

	// 建立淡紅色底色樣式 (#FFC7CE)，當區域不符合條件時套用
	lightRedStyle, err := f.NewConditionalStyle(&excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#FFC7CE"},
			Pattern: 1,
		},
	})
	if err != nil {
		return
	}

	startColStr := getColName(bomStartCol)
	endColStr := getColName(bomStartCol + totalModelCols - 1)

	// 計算每個物料 group 的列範圍 (groupStartRow .. groupEndRow)
	currentRow := 6
	for _, part := range parts {
		startR := currentRow
		endR := currentRow + len(part.SecondSources)
		currentRow = endR + 1

		// 套用至該群組的所有 Model 欄位範圍 (例如 H7:O9)
		targetRange := fmt.Sprintf("%s%d:%s%d", startColStr, startR, endColStr, endR)

		// 相對欄位公式 (例如 H$7:H$9)：
		// 條件 a: COUNTIF(range, "V") <> 1 (必須且只能有一個 V/v)
		// 條件 b: COUNTIF(range, "") + COUNTIF(range, "V") <> groupRowCount (不可包含空字串/空白與 V/v 以外的非法字元)
		// 說明：Excel 中 SetCellValue("", cell, "") 寫入的空字串會被 COUNTIF(range, "<>") 視為非空白而誤判，
		//       使用 COUNTIF(range, "") 能同時統計完全空白儲存格與空字串 ""，完美排除空字串的誤判問題。
		groupRowCount := endR - startR + 1
		colRangeStr := fmt.Sprintf("%s$%d:%s$%d", startColStr, startR, startColStr, endR)
		invalidFormula := fmt.Sprintf(
			"OR(COUNTIF(%s,\"V\")<>1,COUNTIF(%s,\"\")+COUNTIF(%s,\"V\")<>%d)",
			colRangeStr, colRangeStr, colRangeStr, groupRowCount,
		)

		_ = f.SetConditionalFormat("BigMatrix", targetRange, []excelize.ConditionalFormatOptions{
			{Type: "formula", Criteria: invalidFormula, Format: &lightRedStyle},
		})
	}
}
