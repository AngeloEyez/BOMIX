package excel

import (
	"fmt"
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

	// 8.1.3 - Dynamic column generation for multiple BOMs and Models
	maxModelCount := 0
	for _, rev := range revisions {
		if len(rev.ModelQty) > maxModelCount {
			maxModelCount = len(rev.ModelQty)
		}
	}
	if maxModelCount == 0 {
		maxModelCount = 3 // Default
	}

	// Model names (A, B, C, ...)
	modelNames := make([]string, maxModelCount)
	for i := 0; i < maxModelCount; i++ {
		modelNames[i] = string(rune('A' + i))
	}

	// Read archetype style IDs from template for H, I, J columns (Header rows 2-5)
	styleH2, _ := f.GetCellStyle("BigMatrix", "H2")
	styleH3, _ := f.GetCellStyle("BigMatrix", "H3")
	styleH4, _ := f.GetCellStyle("BigMatrix", "H4")
	styleH5, _ := f.GetCellStyle("BigMatrix", "H5")

	styleI2, _ := f.GetCellStyle("BigMatrix", "I2")
	styleI3, _ := f.GetCellStyle("BigMatrix", "I3")
	styleI4, _ := f.GetCellStyle("BigMatrix", "I4")
	styleI5, _ := f.GetCellStyle("BigMatrix", "I5")

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
		// Header rows 2..5
		switch row {
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
		revModelCount := len(rev.ModelQty)
		if override, ok := options.ModelCountOverrides[rev.ID]; ok && override > 0 {
			revModelCount = override
		} else if revModelCount == 0 {
			revModelCount = maxModelCount
		}

		startColName := getColName(currentCol)
		endColName := getColName(currentCol + revModelCount - 1)

		// Apply archetype styles for each column in this revision (rows 2..5)
		for i := 0; i < revModelCount; i++ {
			col := getColName(currentCol + i)
			for r := 2; r <= 5; r++ {
				st := getModelStyle(r, i, revModelCount)
				_ = f.SetCellStyle("BigMatrix", fmt.Sprintf("%s%d", col, r), fmt.Sprintf("%s%d", col, r), st)
			}

			// Model name (row 4)
			f.SetCellValue("BigMatrix", fmt.Sprintf("%s4", col), modelNames[i])

			// Model quantity (row 5)
			qty := rev.ModelQty[modelNames[i]]
			if qty == 0 {
				qty = 1 // Default
			}
			f.SetCellValue("BigMatrix", fmt.Sprintf("%s5", col), qty)
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

	// Helper function to apply styles to a row (columns A-G and dynamic Model columns)
	applyFullRowStyle := func(f *excelize.File, sheet string, row int, isEven bool) {
		refRow := 6
		if !isEven {
			refRow = 7
		}

		// Apply A-G styles from template refRow
		for c := 'A'; c <= 'G'; c++ {
			colStr := string(c)
			cell := fmt.Sprintf("%s%d", colStr, row)
			refCell := fmt.Sprintf("%s%d", colStr, refRow)
			st, _ := f.GetCellStyle(sheet, refCell)
			_ = f.SetCellStyle(sheet, cell, cell, st)
		}

		// Apply dynamic Model column styles (H onwards)
		cIdx := bomStartCol
		for _, rev := range revisions {
			revModelCount := len(rev.ModelQty)
			if revModelCount == 0 {
				revModelCount = maxModelCount
			}
			for i := 0; i < revModelCount; i++ {
				colStr := getColName(cIdx + i)
				cell := fmt.Sprintf("%s%d", colStr, row)
				st := getModelStyle(refRow, i, revModelCount)
				_ = f.SetCellStyle(sheet, cell, cell, st)
			}
			cIdx += revModelCount
		}
	}

	// Clear default template sample rows (Row 6 and Row 7)
	for r := 6; r <= 7; r++ {
		for col := 'A'; col <= 'J'; col++ {
			f.SetCellValue("BigMatrix", fmt.Sprintf("%c%d", col, r), "")
		}
	}

	// 8.1.4 - Write part data
	rowIndex := 6
	for _, part := range parts {
		isEven := (rowIndex%2 == 0)
		applyFullRowStyle(f, "BigMatrix", rowIndex, isEven)

		// Write basic part data (columns A-G)
		f.SetCellValue("BigMatrix", fmt.Sprintf("A%d", rowIndex), part.Item)
		f.SetCellValue("BigMatrix", fmt.Sprintf("B%d", rowIndex), part.HHPN)
		f.SetCellValue("BigMatrix", fmt.Sprintf("C%d", rowIndex), part.Description)
		f.SetCellValue("BigMatrix", fmt.Sprintf("D%d", rowIndex), part.Supplier)
		f.SetCellValue("BigMatrix", fmt.Sprintf("E%d", rowIndex), part.SupplierPn)
		f.SetCellValue("BigMatrix", fmt.Sprintf("F%d", rowIndex), part.Qty)
		f.SetCellValue("BigMatrix", fmt.Sprintf("G%d", rowIndex), part.Location)

		// 8.1.5.3 - Write Model selections
		currentCol = bomStartCol
		for _, rev := range revisions {
			revModelCount := len(rev.ModelQty)
			if revModelCount == 0 {
				revModelCount = maxModelCount
			}

			for i := 0; i < revModelCount; i++ {
				col := getColName(currentCol + i)
				cell := fmt.Sprintf("%s%d", col, rowIndex)

				modelName := modelNames[i]
				if selectedPN, ok := part.Selections[modelName]; ok && selectedPN != "" {
					if part.SupplierPn == selectedPN {
						f.SetCellValue("BigMatrix", cell, "V")
					}
				} else if len(part.Selections) == 0 {
					f.SetCellValue("BigMatrix", cell, "V")
				}
			}

			currentCol += revModelCount
		}

		rowIndex++

		// Write second sources
		for _, ss := range part.SecondSources {
			isEvenSS := (rowIndex%2 == 0)
			applyFullRowStyle(f, "BigMatrix", rowIndex, isEvenSS)

			f.SetCellValue("BigMatrix", fmt.Sprintf("B%d", rowIndex), ss.HHPN)
			f.SetCellValue("BigMatrix", fmt.Sprintf("C%d", rowIndex), ss.Description)
			f.SetCellValue("BigMatrix", fmt.Sprintf("D%d", rowIndex), ss.Supplier)
			f.SetCellValue("BigMatrix", fmt.Sprintf("E%d", rowIndex), ss.SupplierPn)

			rowIndex++
		}
	}

	// Save to output path using validateAndPrepareOutputPath
	seriesName := "BOMIX"
	if len(revisions) > 0 && revisions[0].ProjectCode != "" {
		seriesName = revisions[0].ProjectCode
	}
	defaultFileName := generateBigMatrixFileName(seriesName, revisions, date)
	outputPath, err := validateAndPrepareOutputPath(options.OutputPath, options.OutputDir, defaultFileName)
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

// FilterPartsByCriteria filters parts based on export criteria
// See product-spec section 8.1.6
func FilterPartsByCriteria(parts []PartData, mode string) []PartData {
	var filtered []PartData

	for _, part := range parts {
		// Filter by CCL = Y
		if part.CCL != "Y" {
			continue
		}

		// Filter by BOM status
		if part.BOMStatus == "X" {
			continue
		}

		// Filter by mode
		if mode == "NPI" {
			if part.BOMStatus != "I" && part.BOMStatus != "P" {
				continue
			}
		} else if mode == "MP" {
			if part.BOMStatus != "I" && part.BOMStatus != "M" {
				continue
			}
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
			Item:           group[0].Item,
			HHPN:           group[0].HHPN,
			Description:    group[0].Description,
			Supplier:       group[0].Supplier,
			SupplierPn:     group[0].SupplierPn,
			Qty:            group[0].Qty,
			Location:       group[0].Location,
			Type:           group[0].Type,
			BOMStatus:      group[0].BOMStatus,
			CCL:            group[0].CCL,
			Remark:         group[0].Remark,
			Selections:     make(map[string]string),
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
