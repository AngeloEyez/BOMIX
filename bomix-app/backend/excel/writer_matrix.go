package excel

import (
	"fmt"
	"sort"
	"strings"

	"bomix-app/backend/types"

	"github.com/xuri/excelize/v2"
)

// exportMatrix exports data to Matrix format
// See product-spec section 8.2
func (w *WriterImpl) exportMatrix(options ExportOptions) ([]string, error) {
	if w.logger != nil {
		w.logger.Info(fmt.Sprintf("[exportMatrix] 開始產生 Matrix 匯出檔 (Revisions 數量: %d)", len(options.RevisionIDs)))
		w.logger.Debug(fmt.Sprintf("[exportMatrix] 載入範本檔: %s", types.FormatMatrix))
	}

	// Load template
	f, err := w.templateManager.LoadTemplate(types.FormatMatrix)
	if err != nil {
		if w.logger != nil {
			w.logger.Error(fmt.Sprintf("[exportMatrix] 載入 Matrix 範本失敗: %v", err))
		}
		return nil, fmt.Errorf("failed to load Matrix template: %w", err)
	}
	defer f.Close()

	// 取得 Revision metadata（優先取用傳入的 options.Revisions[0]）
	rev := RevisionData{
		ProjectCode:      "BOMIX",
		Description:      options.Description,
		SchematicVersion: "1.0",
		PCBVersion:       "1.0",
		PCAPN:            "",
		Phase:            "DB",
		Version:          "0.1",
		Date:             generateTimestamp(),
		ModelQty:         make(map[string]int),
	}

	if len(options.Revisions) > 0 {
		r0 := options.Revisions[0]
		if r0.ProjectCode != "" {
			rev.ProjectCode = r0.ProjectCode
		}
		if r0.Description != "" {
			rev.Description = r0.Description
		}
		if r0.SchematicVersion != "" {
			rev.SchematicVersion = r0.SchematicVersion
		}
		if r0.PCBVersion != "" {
			rev.PCBVersion = r0.PCBVersion
		}
		if r0.PCAPN != "" {
			rev.PCAPN = r0.PCAPN
		}
		if r0.Phase != "" {
			rev.Phase = r0.Phase
		}
		if r0.Version != "" {
			rev.Version = r0.Version
		}
		if len(r0.ModelQty) > 0 {
			for k, v := range r0.ModelQty {
				rev.ModelQty[k] = v
			}
		}
		if len(r0.ModelQtyByOrder) > 0 {
			rev.ModelQtyByOrder = make(map[int]int)
			for k, v := range r0.ModelQtyByOrder {
				rev.ModelQtyByOrder[k] = v
			}
		}
	} else if options.Description != "" {
		rev.ProjectCode = options.Description
	}

	date := generateTimestamp()

	// 8.2.2 - Tag replacement for header
	tags := map[string]string{
		"{{.ProjectCode}}":      rev.ProjectCode,
		"{{.Description}}":      rev.Description,
		"{{.SchematicVersion}}": rev.SchematicVersion,
		"{{.PCBVersion}}":       rev.PCBVersion,
		"{{.PCAPN}}":            rev.PCAPN,
		"{{.Phase}}":            rev.Phase,
		"{{.Version}}":          rev.Version,
		"{{.Date}}":             date,
	}

	orderedModelNames := getOrderedModelNames(rev.ModelQty)

	for i := 0; i < 7; i++ {
		modelAlias := fmt.Sprintf("%c", 'A'+i)
		qtyStr := ""
		qty := resolveModelQty(rev, i, orderedModelNames)
		if qty > 0 {
			qtyStr = fmt.Sprintf("%d", qty)
		}
		tags[fmt.Sprintf("{{.ModelQty%s}}", modelAlias)] = qtyStr
	}
	if err := applyTagReplacement(f, tags); err != nil {
		return nil, err
	}

	// 8.2.4 - Dynamic Model column generation (minimum 7 models)
	modelStartCol := 10 // K
	minModelCount := 7
	actualModelCount := len(rev.ModelQty)
	if len(rev.ModelQtyByOrder) > actualModelCount {
		actualModelCount = len(rev.ModelQtyByOrder)
	}
	for _, p := range options.PartData {
		for sortOrder, selectedPN := range p.SelectionsByOrder {
			if selectedPN != "" && sortOrder+1 > actualModelCount {
				actualModelCount = sortOrder + 1
			}
		}
	}
	if actualModelCount < minModelCount {
		actualModelCount = minModelCount
	}

	// Ensure SMD, PTH, BOTTOM sheets exist, falling back to copying SMD if PTH/BOTTOM missing
	sheets, err := w.ensureMatrixSheets(f)
	if err != nil {
		return nil, err
	}

	// Write Model quantities
	for i := 0; i < actualModelCount; i++ {
		col := getColName(modelStartCol + i)

		// Model quantity (row 5)
		qty := resolveModelQty(rev, i, orderedModelNames)
		if qty > 0 {
			f.SetCellValue("SMD", fmt.Sprintf("%s5", col), qty)
			f.SetCellValue("PTH", fmt.Sprintf("%s5", col), qty)
			f.SetCellValue("BOTTOM", fmt.Sprintf("%s5", col), qty)
		} else {
			f.SetCellValue("SMD", fmt.Sprintf("%s5", col), "")
			f.SetCellValue("PTH", fmt.Sprintf("%s5", col), "")
			f.SetCellValue("BOTTOM", fmt.Sprintf("%s5", col), "")
		}
	}

	// Calculate Remark column position (after all Model columns)
	remarkCol := getColName(modelStartCol + actualModelCount)

	// 讀取 A~J 欄位、Model 欄位 (K6, K7) 與 Remark 欄位在 Row 6 (偶數群組) 與 Row 7 (奇數群組) 的 Archetype Style ID
	styleCol6 := make(map[string]int)
	styleCol7 := make(map[string]int)
	for c := 'A'; c <= 'J'; c++ {
		colStr := string(c)
		styleCol6[colStr], _ = f.GetCellStyle("SMD", colStr+"6")
		styleCol7[colStr], _ = f.GetCellStyle("SMD", colStr+"7")
	}

	styleModel6, _ := f.GetCellStyle("SMD", "K6")
	styleModel7, _ := f.GetCellStyle("SMD", "K7")

	styleRemark6, _ := f.GetCellStyle("SMD", "Q6")
	styleRemark7, _ := f.GetCellStyle("SMD", "Q7")
	if styleRemark6 == 0 {
		styleRemark6 = styleModel6
	}
	if styleRemark7 == 0 {
		styleRemark7 = styleModel7
	}

	// 清空範本檔在 Row 6 與 Row 7 殘留的預設範例文字 (避免替代料列殘留範本舊文字)
	for _, sheet := range sheets {
		for r := 6; r <= 7; r++ {
			for c := 'A'; c <= 'J'; c++ {
				_ = f.SetCellValue(sheet, fmt.Sprintf("%c%d", c, r), nil)
			}
		}
	}

	// protoStyleCache 樣式快取：以原始 styleID 為鍵，快取 Font.Color: "#8080C0" 版本的 styleID。
	protoStyleCache := make(map[int]int)

	// makeProtoStyle 建立並快取指定樣式的 PROTO 物料文字顏色（Font.Color: "#8080C0"）版本。
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

	// 建立套用全列樣式 (A-R 欄) 的輔助函數
	applyFullMatrixRowStyle := func(f *excelize.File, sheet string, row int, isEven bool, isProto bool) {
		// 1. 套用 A ~ J 欄位範本原生樣式 (保留各欄對齊方式、數字格式與邊框格線)
		for c := 'A'; c <= 'J'; c++ {
			colStr := string(c)
			cell := fmt.Sprintf("%s%d", colStr, row)
			var st int
			if isEven {
				st = styleCol6[colStr]
			} else {
				st = styleCol7[colStr]
			}
			if isProto {
				st = makeProtoStyle(st)
			}
			_ = f.SetCellStyle(sheet, cell, cell, st)
		}

		// 2. 套用 Model 欄位 (K 到 Q 欄) 的 Model Archetype 樣式 (含 "V" 置中、斑馬紋底色與格線邊框)
		for i := 0; i < actualModelCount; i++ {
			colStr := getColName(modelStartCol + i)
			cell := fmt.Sprintf("%s%d", colStr, row)
			var st int
			if isEven {
				st = styleModel6
			} else {
				st = styleModel7
			}
			if isProto {
				st = makeProtoStyle(st)
			}
			_ = f.SetCellStyle(sheet, cell, cell, st)
		}

		// 3. 套用 Remark 欄位 (R 欄) 的 Remark Archetype 樣式 (含對齊、斑馬紋底色與格線邊框)
		remarkCell := fmt.Sprintf("%s%d", remarkCol, row)
		var remarkSt int
		if isEven {
			remarkSt = styleRemark6
		} else {
			remarkSt = styleRemark7
		}
		if isProto {
			remarkSt = makeProtoStyle(remarkSt)
		}
		_ = f.SetCellStyle(sheet, remarkCell, remarkCell, remarkSt)
	}

	// 8.2.5 - Write part data to each sheet
	for _, sheet := range sheets {
		rowIndex := 6

		// Filter parts by sheet type
		var sheetParts []PartData
		for _, part := range options.PartData {
			pType := strings.ToUpper(strings.TrimSpace(part.Type))
			if pType == "" {
				pType = "SMD"
			}
			if pType == strings.ToUpper(sheet) {
				sheetParts = append(sheetParts, part)
			}
		}

		for groupIdx, part := range sheetParts {
			// Alternate row styles by material group (Row 6 for even groups, Row 7 for odd groups)
			isEven := (groupIdx%2 == 0)
			isProtoGroup := strings.EqualFold(part.BOMStatus, "P")
			applyFullMatrixRowStyle(f, sheet, rowIndex, isEven, isProtoGroup)

			// Write basic part data (columns A, B, D, E, F, G, H)
			// Column C is empty per spec 8.2.5.1
			f.SetCellValue(sheet, fmt.Sprintf("A%d", rowIndex), part.Item)
			f.SetCellValue(sheet, fmt.Sprintf("B%d", rowIndex), part.HHPN)
			f.SetCellValue(sheet, fmt.Sprintf("D%d", rowIndex), part.Description)
			f.SetCellValue(sheet, fmt.Sprintf("E%d", rowIndex), part.Supplier)
			f.SetCellValue(sheet, fmt.Sprintf("F%d", rowIndex), part.SupplierPn)
			f.SetCellValue(sheet, fmt.Sprintf("G%d", rowIndex), part.Qty)
			f.SetCellValue(sheet, fmt.Sprintf("H%d", rowIndex), part.Location)

			// 8.2.5.3 - I column formula: =G{row}*J{row}
			formulaI := fmt.Sprintf("=G%d*J%d", rowIndex, rowIndex)
			f.SetCellFormula(sheet, fmt.Sprintf("I%d", rowIndex), formulaI)

			// 8.2.5.4 - J column formula: Sum of selected Model quantities
			formulaJ := generateMatrixSelectionFormula(modelStartCol, actualModelCount, rowIndex, rev.ModelQty)
			f.SetCellFormula(sheet, fmt.Sprintf("J%d", rowIndex), formulaJ)

			// Write Model selections (columns K onwards)
			for i := 0; i < actualModelCount; i++ {
				col := getColName(modelStartCol + i)
				cell := fmt.Sprintf("%s%d", col, rowIndex)
				if isMainSourceSelected(part, "", i, orderedModelNames) {
					f.SetCellValue(sheet, cell, "V")
				}
			}

			// Write Remark (column after all Model columns)
			f.SetCellValue(sheet, fmt.Sprintf("%s%d", remarkCol, rowIndex), part.Remark)

			rowIndex++

			// Write second sources
			for _, ss := range part.SecondSources {
				// Apply same group rowStyle for second sources
				applyFullMatrixRowStyle(f, sheet, rowIndex, isEven, isProtoGroup)

				// Second sources don't have Item, Qty or Location. Clear template residue by setting cell value to nil.
				f.SetCellValue(sheet, fmt.Sprintf("A%d", rowIndex), nil)
				f.SetCellValue(sheet, fmt.Sprintf("B%d", rowIndex), ss.HHPN)
				f.SetCellValue(sheet, fmt.Sprintf("C%d", rowIndex), nil)
				f.SetCellValue(sheet, fmt.Sprintf("D%d", rowIndex), ss.Description)
				f.SetCellValue(sheet, fmt.Sprintf("E%d", rowIndex), ss.Supplier)
				f.SetCellValue(sheet, fmt.Sprintf("F%d", rowIndex), ss.SupplierPn)
				f.SetCellValue(sheet, fmt.Sprintf("G%d", rowIndex), nil)
				f.SetCellValue(sheet, fmt.Sprintf("H%d", rowIndex), nil)

				// Formulas for second sources
				formulaI := fmt.Sprintf("=G%d*J%d", rowIndex, rowIndex)
				f.SetCellFormula(sheet, fmt.Sprintf("I%d", rowIndex), formulaI)

				formulaJ := generateMatrixSelectionFormula(modelStartCol, actualModelCount, rowIndex, rev.ModelQty)
				f.SetCellFormula(sheet, fmt.Sprintf("J%d", rowIndex), formulaJ)

				// Model selections for second sources
				for i := 0; i < actualModelCount; i++ {
					col := getColName(modelStartCol + i)
					cell := fmt.Sprintf("%s%d", col, rowIndex)
					if isSecondSourceSelected(ss, part, "", i, orderedModelNames) {
						f.SetCellValue(sheet, cell, "V")
					}
				}

				// Remark for second source
				f.SetCellValue(sheet, fmt.Sprintf("%s%d", remarkCol, rowIndex), ss.Remark)

				rowIndex++
			}
		}
	}

	// Save to output path using validateAndPrepareOutputPath
	fileName := generateMatrixFileName(rev, date)
	outputPath, err := validateAndPrepareOutputPath(w.logger, options.OutputPath, options.OutputDir, fileName)
	if err != nil {
		return nil, err
	}

	if err := f.SaveAs(outputPath); err != nil {
		return nil, fmt.Errorf("failed to save Matrix: %w", err)
	}

	if w.logger != nil {
		w.logger.Info(fmt.Sprintf("[exportMatrix] 成功匯出 Matrix 檔案: %s", fileName), "path", outputPath)
	}

	return []string{outputPath}, nil
}

// generateMatrixSelectionFormula generates the formula for J column
// See product-spec section 8.2.5.4
func generateMatrixSelectionFormula(startCol int, modelCount int, row int, modelQty map[string]int) string {
	var parts []string

	for i := 0; i < modelCount; i++ {
		col := getColName(startCol + i)
		// Formula: IF(EXACT(col$row,"V"),col$5,0)
		formula := fmt.Sprintf("IF(EXACT(%s%d,\"V\"),%s$5,0)", col, row, col)
		parts = append(parts, formula)
	}

	return strings.Join(parts, "+")
}

// exportMatrixDetailed is an extended version with full data population
// See product-spec sections 8.2.2 - 8.2.10
func (w *WriterImpl) exportMatrixDetailed(options ExportOptions, rev RevisionData, parts []PartData) ([]string, error) {
	// Load template
	f, err := w.templateManager.LoadTemplate(types.FormatMatrix)
	if err != nil {
		return nil, fmt.Errorf("failed to load Matrix template: %w", err)
	}
	defer f.Close()

	date := generateTimestamp()

	// 8.2.3 - Tag replacement for header
	tags := map[string]string{
		"{{.ProjectCode}}":      rev.ProjectCode,
		"{{.Description}}":      rev.Description,
		"{{.SchematicVersion}}": rev.SchematicVersion,
		"{{.PCBVersion}}":       rev.PCBVersion,
		"{{.PCAPN}}":            rev.PCAPN,
		"{{.Phase}}":            rev.Phase,
		"{{.Version}}":          rev.Version,
		"{{.Date}}":             date,
	}

	orderedModelNames := getOrderedModelNames(rev.ModelQty)

	// Add model quantities
	for i := 0; i < 7; i++ {
		modelAlias := fmt.Sprintf("%c", 'A'+i)
		qtyStr := ""
		qty := resolveModelQty(rev, i, orderedModelNames)
		if qty > 0 {
			qtyStr = fmt.Sprintf("%d", qty)
		}
		tags[fmt.Sprintf("{{.ModelQty%s}}", modelAlias)] = qtyStr
	}

	if err := applyTagReplacement(f, tags); err != nil {
		return nil, err
	}

	// 8.2.4 - Dynamic Model column generation (minimum 7 models)
	modelStartCol := 10 // K
	minModelCount := 7
	actualModelCount := len(rev.ModelQty)
	if len(rev.ModelQtyByOrder) > actualModelCount {
		actualModelCount = len(rev.ModelQtyByOrder)
	}
	for _, p := range parts {
		for sortOrder, selectedPN := range p.SelectionsByOrder {
			if selectedPN != "" && sortOrder+1 > actualModelCount {
				actualModelCount = sortOrder + 1
			}
		}
	}
	if actualModelCount < minModelCount {
		actualModelCount = minModelCount
	}

	// Ensure SMD, PTH, BOTTOM sheets exist, falling back to copying SMD if PTH/BOTTOM missing
	sheets, err := w.ensureMatrixSheets(f)
	if err != nil {
		return nil, err
	}

	// Write Model quantities for all sheets
	for _, sheet := range sheets {
		for i := 0; i < actualModelCount; i++ {
			col := getColName(modelStartCol + i)

			// Model quantity (row 5)
			qty := resolveModelQty(rev, i, orderedModelNames)
			if qty > 0 {
				f.SetCellValue(sheet, fmt.Sprintf("%s5", col), qty)
			} else {
				f.SetCellValue(sheet, fmt.Sprintf("%s5", col), "")
			}
		}
	}

	// 讀取 A~J 欄位、Model 欄位 (K6, K7) 與 Remark 欄位在 Row 6 (偶數群組) 與 Row 7 (奇數群組) 的 Archetype Style ID
	styleCol6 := make(map[string]int)
	styleCol7 := make(map[string]int)
	for c := 'A'; c <= 'J'; c++ {
		colStr := string(c)
		styleCol6[colStr], _ = f.GetCellStyle("SMD", colStr+"6")
		styleCol7[colStr], _ = f.GetCellStyle("SMD", colStr+"7")
	}

	styleModel6, _ := f.GetCellStyle("SMD", "K6")
	styleModel7, _ := f.GetCellStyle("SMD", "K7")

	styleRemark6, _ := f.GetCellStyle("SMD", "Q6")
	styleRemark7, _ := f.GetCellStyle("SMD", "Q7")
	if styleRemark6 == 0 {
		styleRemark6 = styleModel6
	}
	if styleRemark7 == 0 {
		styleRemark7 = styleModel7
	}

	// 清空範本檔在 Row 6 與 Row 7 殘留的預設範例文字 (避免替代料列殘留範本舊文字)
	for _, sheet := range sheets {
		for r := 6; r <= 7; r++ {
			for c := 'A'; c <= 'J'; c++ {
				_ = f.SetCellValue(sheet, fmt.Sprintf("%c%d", c, r), nil)
			}
		}
	}

	// Calculate Remark column position
	remarkCol := getColName(modelStartCol + actualModelCount)

	// protoStyleCacheDetailed 樣式快取
	protoStyleCacheDetailed := make(map[int]int)

	// makeProtoStyleDetailed 建立並快取指定樣式的 PROTO 物料文字顏色（Font.Color: "#8080C0"）版本。
	makeProtoStyleDetailed := func(styleID int) int {
		if styleID <= 0 {
			return styleID
		}
		if cached, ok := protoStyleCacheDetailed[styleID]; ok {
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
			protoStyleCacheDetailed[styleID] = styleID
			return styleID
		}
		protoStyleCacheDetailed[styleID] = newID
		return newID
	}

	// 建立套用全列樣式 (A-R 欄) 的輔助函數
	applyFullMatrixRowStyle := func(f *excelize.File, sheet string, row int, isEven bool, isProto bool) {
		// 1. 套用 A ~ J 欄位範本原生樣式 (保留各欄對齊方式、數字格式與邊框格線)
		for c := 'A'; c <= 'J'; c++ {
			colStr := string(c)
			cell := fmt.Sprintf("%s%d", colStr, row)
			var st int
			if isEven {
				st = styleCol6[colStr]
			} else {
				st = styleCol7[colStr]
			}
			if isProto {
				st = makeProtoStyleDetailed(st)
			}
			_ = f.SetCellStyle(sheet, cell, cell, st)
		}

		// 2. 套用 Model 欄位 (K 到 Q 欄) 的 Model Archetype 樣式 (含 "V" 置中、斑馬紋底色與格線邊框)
		for i := 0; i < actualModelCount; i++ {
			colStr := getColName(modelStartCol + i)
			cell := fmt.Sprintf("%s%d", colStr, row)
			var st int
			if isEven {
				st = styleModel6
			} else {
				st = styleModel7
			}
			if isProto {
				st = makeProtoStyleDetailed(st)
			}
			_ = f.SetCellStyle(sheet, cell, cell, st)
		}

		// 3. 套用 Remark 欄位 (R 欄) 的 Remark Archetype 樣式 (含對齊、斑馬紋底色與格線邊框)
		remarkCell := fmt.Sprintf("%s%d", remarkCol, row)
		var remarkSt int
		if isEven {
			remarkSt = styleRemark6
		} else {
			remarkSt = styleRemark7
		}
		if isProto {
			remarkSt = makeProtoStyleDetailed(remarkSt)
		}
		_ = f.SetCellStyle(sheet, remarkCell, remarkCell, remarkSt)
	}

	// 8.2.6 - Filter parts by criteria
	filteredParts := filterMatrixParts(parts)

	// 8.2.5 - Write part data to each sheet
	for _, sheet := range sheets {
		rowIndex := 6

		// Filter parts by sheet type
		var sheetParts []PartData
		for _, part := range filteredParts {
			pType := strings.ToUpper(strings.TrimSpace(part.Type))
			if pType == "" {
				pType = "SMD"
			}
			if pType == strings.ToUpper(sheet) {
				sheetParts = append(sheetParts, part)
			}
		}

		for groupIdx, part := range sheetParts {
			// Alternate row styles by material group (Row 6 for even groups, Row 7 for odd groups)
			isEven := (groupIdx%2 == 0)
			isProtoGroup := strings.EqualFold(part.BOMStatus, "P")
			applyFullMatrixRowStyle(f, sheet, rowIndex, isEven, isProtoGroup)

			// Write basic part data (columns A, B, D, E, F, G, H)
			// Column C is empty per spec 8.2.5.1
			f.SetCellValue(sheet, fmt.Sprintf("A%d", rowIndex), part.Item)
			f.SetCellValue(sheet, fmt.Sprintf("B%d", rowIndex), part.HHPN)
			f.SetCellValue(sheet, fmt.Sprintf("D%d", rowIndex), part.Description)
			f.SetCellValue(sheet, fmt.Sprintf("E%d", rowIndex), part.Supplier)
			f.SetCellValue(sheet, fmt.Sprintf("F%d", rowIndex), part.SupplierPn)
			f.SetCellValue(sheet, fmt.Sprintf("G%d", rowIndex), part.Qty)
			f.SetCellValue(sheet, fmt.Sprintf("H%d", rowIndex), part.Location)

			// 8.2.5.3 - I column formula: =G{row}*J{row}
			formulaI := fmt.Sprintf("=G%d*J%d", rowIndex, rowIndex)
			f.SetCellFormula(sheet, fmt.Sprintf("I%d", rowIndex), formulaI)

			// 8.2.5.4 - J column formula
			formulaJ := generateMatrixSelectionFormula(modelStartCol, actualModelCount, rowIndex, rev.ModelQty)
			f.SetCellFormula(sheet, fmt.Sprintf("J%d", rowIndex), formulaJ)

			// Write Model selections (columns K onwards)
			for i := 0; i < actualModelCount; i++ {
				col := getColName(modelStartCol + i)
				cell := fmt.Sprintf("%s%d", col, rowIndex)
				if isMainSourceSelected(part, rev.ID, i, orderedModelNames) {
					f.SetCellValue(sheet, cell, "V")
				}
			}

			// 8.2.5.5 - Write Remark (column after all Model columns)
			f.SetCellValue(sheet, fmt.Sprintf("%s%d", remarkCol, rowIndex), part.Remark)

			rowIndex++

			// Write second sources
			for _, ss := range part.SecondSources {
				// Apply same group rowStyle for second sources
				applyFullMatrixRowStyle(f, sheet, rowIndex, isEven, isProtoGroup)

				// Second sources don't have Item, Qty or Location. Clear template residue by setting cell value to nil.
				f.SetCellValue(sheet, fmt.Sprintf("A%d", rowIndex), nil)
				f.SetCellValue(sheet, fmt.Sprintf("B%d", rowIndex), ss.HHPN)
				f.SetCellValue(sheet, fmt.Sprintf("C%d", rowIndex), nil)
				f.SetCellValue(sheet, fmt.Sprintf("D%d", rowIndex), ss.Description)
				f.SetCellValue(sheet, fmt.Sprintf("E%d", rowIndex), ss.Supplier)
				f.SetCellValue(sheet, fmt.Sprintf("F%d", rowIndex), ss.SupplierPn)
				f.SetCellValue(sheet, fmt.Sprintf("G%d", rowIndex), nil)
				f.SetCellValue(sheet, fmt.Sprintf("H%d", rowIndex), nil)

				// Formulas for second sources
				formulaI := fmt.Sprintf("=G%d*J%d", rowIndex, rowIndex)
				f.SetCellFormula(sheet, fmt.Sprintf("I%d", rowIndex), formulaI)

				formulaJ := generateMatrixSelectionFormula(modelStartCol, actualModelCount, rowIndex, rev.ModelQty)
				f.SetCellFormula(sheet, fmt.Sprintf("J%d", rowIndex), formulaJ)

				// Model selections for second sources
				for i := 0; i < actualModelCount; i++ {
					col := getColName(modelStartCol + i)
					cell := fmt.Sprintf("%s%d", col, rowIndex)
					if isSecondSourceSelected(ss, part, rev.ID, i, orderedModelNames) {
						f.SetCellValue(sheet, cell, "V")
					}
				}

				// Remark for second source
				f.SetCellValue(sheet, fmt.Sprintf("%s%d", remarkCol, rowIndex), ss.Remark)

				rowIndex++
			}
		}
	}

	// Save to output path using validateAndPrepareOutputPath
	fileName := generateMatrixFileName(rev, date)
	outputPath, err := validateAndPrepareOutputPath(w.logger, options.OutputPath, options.OutputDir, fileName)
	if err != nil {
		return nil, err
	}

	if err := f.SaveAs(outputPath); err != nil {
		return nil, fmt.Errorf("failed to save Matrix: %w", err)
	}

	if w.logger != nil {
		w.logger.Info(fmt.Sprintf("[exportMatrixDetailed] 成功匯出 Matrix 檔案: %s", fileName), "path", outputPath)
	}

	return []string{outputPath}, nil
}

// filterMatrixParts filters parts based on Matrix export criteria (CCL = Y, BOMStatus != X)
// See product-spec section 8.2.6
func filterMatrixParts(parts []PartData) []PartData {
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

// getOrderedModelNames 將 modelQty/Selections 中的模型名稱整理成排序後的名稱清單
func getOrderedModelNames(modelQtyMap map[string]int) []string {
	names := make([]string, 0, len(modelQtyMap))
	for k := range modelQtyMap {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// resolveModelQty 彈性獲取第 index 個 Model 的 Qty (優先依據 SortOrder 索引)
func resolveModelQty(rev RevisionData, index int, orderedNames []string) int {
	if q, ok := rev.ModelQtyByOrder[index]; ok && q > 0 {
		return q
	}
	if len(rev.ModelQty) == 0 {
		return 0
	}
	alias := fmt.Sprintf("%c", 'A'+index)
	if q, ok := rev.ModelQty[alias]; ok && q > 0 {
		return q
	}
	nameWithModel := fmt.Sprintf("Model %s", alias)
	if q, ok := rev.ModelQty[nameWithModel]; ok && q > 0 {
		return q
	}
	nameWithNum := fmt.Sprintf("Model %d", index+1)
	if q, ok := rev.ModelQty[nameWithNum]; ok && q > 0 {
		return q
	}
	if index < len(orderedNames) {
		if q, ok := rev.ModelQty[orderedNames[index]]; ok && q > 0 {
			return q
		}
	}
	return 0
}

// resolveSelectedMaterial 獲取第 index 個 Model 在 Selections 中的選取 Material ("Supplier|SupplierPN")
func resolveSelectedMaterial(part PartData, revID string, index int, orderedNames []string) string {
	if revSelections, ok := part.SelectionsByRevAndMaterial[revID]; ok {
		if mat, exists := revSelections[index]; exists && mat != "" {
			return mat
		}
	}

	if mat, ok := part.SelectionsByMaterialByOrder[index]; ok && mat != "" {
		return mat
	}

	return ""
}

// isMainSourceSelected 判斷主料在指定的 Model 欄位 index 是否被勾選
func isMainSourceSelected(part PartData, revID string, index int, orderedNames []string) bool {
	if part.MainSelectionsByOrder != nil {
		if sel, ok := part.MainSelectionsByOrder[index]; ok {
			return sel
		}
	}

	selMat := resolveSelectedMaterial(part, revID, index, orderedNames)
	if selMat != "" {
		mainMatKey := fmt.Sprintf("%s|%s", strings.TrimSpace(part.Supplier), strings.TrimSpace(part.SupplierPn))
		return strings.EqualFold(selMat, mainMatKey)
	}

	selPN := resolveSelectedPN(part, revID, index, orderedNames)
	if selPN != "" {
		return strings.EqualFold(part.SupplierPn, selPN)
	}

	return false
}

// isSecondSourceSelected 判斷替代料在指定的 Model 欄位 index 是否被勾選
func isSecondSourceSelected(ss SecondSourceData, part PartData, revID string, index int, orderedNames []string) bool {
	if ss.SelectionsByOrder != nil {
		if sel, ok := ss.SelectionsByOrder[index]; ok {
			return sel
		}
	}

	selMat := resolveSelectedMaterial(part, revID, index, orderedNames)
	if selMat != "" {
		ssMatKey := fmt.Sprintf("%s|%s", strings.TrimSpace(ss.Supplier), strings.TrimSpace(ss.SupplierPn))
		return strings.EqualFold(selMat, ssMatKey)
	}

	selPN := resolveSelectedPN(part, revID, index, orderedNames)
	if selPN != "" {
		return strings.EqualFold(ss.SupplierPn, selPN)
	}

	return false
}

// resolveSelectedPN 彈性獲取第 index 個 Model 在 Selections 中的選取 PartPN (優先依據 SortOrder 索引)
func resolveSelectedPN(part PartData, revID string, index int, orderedNames []string) string {
	if revSelections, ok := part.SelectionsByRevAndOrder[revID]; ok {
		if pn, exists := revSelections[index]; exists && pn != "" {
			return pn
		}
	}

	if pn, ok := part.SelectionsByOrder[index]; ok && pn != "" {
		return pn
	}
	if len(part.Selections) == 0 {
		return ""
	}
	alias := fmt.Sprintf("%c", 'A'+index)
	if pn, ok := part.Selections[alias]; ok && pn != "" {
		return pn
	}
	nameWithModel := fmt.Sprintf("Model %s", alias)
	if pn, ok := part.Selections[nameWithModel]; ok && pn != "" {
		return pn
	}
	nameWithNum := fmt.Sprintf("Model %d", index+1)
	if pn, ok := part.Selections[nameWithNum]; ok && pn != "" {
		return pn
	}
	if index < len(orderedNames) {
		if pn, ok := part.Selections[orderedNames[index]]; ok && pn != "" {
			return pn
		}
	}
	return ""
}

// ensureMatrixSheets 確保範本檔中包含 "SMD", "PTH", "BOTTOM" 三個工作頁面。
//  1. 若找不到 SMD 頁面 (不區分大小寫)，輸出 Error Log 並返回錯誤。
//  2. 若找不到 PTH 或 BOTTOM 等其餘頁面，自動退回使用 SMD 頁面當作輸出樣板，
//     並複製創立正確的頁面名稱 (PTH 或 BOTTOM)。
func (w *WriterImpl) ensureMatrixSheets(f *excelize.File) ([]string, error) {
	sheets := []string{"SMD", "PTH", "BOTTOM"}

	// 1. 尋找 SMD 頁面 (不區分大小寫)
	smdRealName := ""
	for _, name := range f.GetSheetList() {
		if strings.EqualFold(name, "SMD") {
			smdRealName = name
			break
		}
	}

	if smdRealName == "" {
		if w.logger != nil {
			w.logger.Error("[Matrix] 範本檔缺少必要的 'SMD' 工作頁面，無法進行 Matrix 匯出", "availableSheets", f.GetSheetList())
		}
		return nil, fmt.Errorf("template missing required 'SMD' sheet")
	}

	// 確保 SMD 頁面名稱更名為標準大寫 "SMD"
	if smdRealName != "SMD" {
		_ = f.SetSheetName(smdRealName, "SMD")
	}

	smdIdx, err := f.GetSheetIndex("SMD")
	if err != nil || smdIdx < 0 {
		if w.logger != nil {
			w.logger.Error("[Matrix] 取得 'SMD' 工作頁面索引失敗", "error", err)
		}
		return nil, fmt.Errorf("failed to get 'SMD' sheet index: %w", err)
	}

	// 2. 處理 PTH 與 BOTTOM 頁面
	for _, target := range sheets[1:] {
		targetRealName := ""
		for _, name := range f.GetSheetList() {
			if strings.EqualFold(name, target) {
				targetRealName = name
				break
			}
		}

		if targetRealName != "" {
			// 若存在舊頁面，確保名稱更名為標準大寫名稱
			if targetRealName != target {
				_ = f.SetSheetName(targetRealName, target)
			}
		} else {
			// 若不存在目標頁面，自動退回使用 SMD 頁面作為樣板
			if w.logger != nil {
				w.logger.Debug(fmt.Sprintf("[Matrix] 範本檔中未找到 '%s' 頁面，自動複製 'SMD' 頁面作為樣板並建立 '%s' 頁面", target, target))
			}
			newIdx, err := f.NewSheet(target)
			if err != nil {
				return nil, fmt.Errorf("failed to create sheet '%s': %w", target, err)
			}
			if err := f.CopySheet(smdIdx, newIdx); err != nil {
				return nil, fmt.Errorf("failed to copy 'SMD' sheet to '%s': %w", target, err)
			}
		}
	}

	return sheets, nil
}
