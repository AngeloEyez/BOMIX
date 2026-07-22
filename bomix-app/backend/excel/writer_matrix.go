package excel

import (
	"fmt"
	"path/filepath"
	"strings"

	"bomix-app/backend/types"
)

// exportMatrix exports data to Matrix format
// See product-spec section 8.2
func (w *WriterImpl) exportMatrix(options ExportOptions) ([]string, error) {
	w.logInfo(fmt.Sprintf("[exportMatrix] 開始產生 Matrix 匯出檔 (Revisions 數量: %d)", len(options.RevisionIDs)))
	w.logDebug(fmt.Sprintf("[exportMatrix] 載入範本檔: %s", types.FormatMatrix))

	// Load template
	f, err := w.templateManager.LoadTemplate(types.FormatMatrix)
	if err != nil {
		w.logError(fmt.Sprintf("[exportMatrix] 載入 Matrix 範本失敗: %v", err))
		return nil, fmt.Errorf("failed to load Matrix template: %w", err)
	}
	defer f.Close()

	// In a real implementation, we would fetch revision data from DB
	// For now, use placeholder data
	rev := RevisionData{
		ProjectCode:     "DEMO",
		Description:     "Demo Project",
		SchematicVersion: "1.0",
		PCBVersion:      "1.0",
		PCAPN:           "DEMO-001",
		Phase:           "DB",
		Version:         "0.1",
		Date:            generateTimestamp(),
		ModelQty:        map[string]int{"A": 1, "B": 1, "C": 1, "D": 1, "E": 1, "F": 1},
	}

	date := generateTimestamp()

	// 8.2.2 - Tag replacement for header
	tags := map[string]string{
		"{{.ProjectCode}}":       rev.ProjectCode,
		"{{.Description}}":       rev.Description,
		"{{.SchematicVersion}}":  rev.SchematicVersion,
		"{{.PCBVersion}}":        rev.PCBVersion,
		"{{.PCAPN}}":             rev.PCAPN,
		"{{.Date}}":              date,
		"{{.ModelQtyA}}":         fmt.Sprintf("%d", rev.ModelQty["A"]),
		"{{.ModelQtyB}}":         fmt.Sprintf("%d", rev.ModelQty["B"]),
		"{{.ModelQtyC}}":         fmt.Sprintf("%d", rev.ModelQty["C"]),
		"{{.ModelQtyD}}":         fmt.Sprintf("%d", rev.ModelQty["D"]),
		"{{.ModelQtyE}}":         fmt.Sprintf("%d", rev.ModelQty["E"]),
		"{{.ModelQtyF}}":         fmt.Sprintf("%d", rev.ModelQty["F"]),
	}
	if err := applyTagReplacement(f, tags); err != nil {
		return nil, err
	}

	// 8.2.4 - Dynamic Model column generation (minimum 6 models)
	modelStartCol := 10 // K
	minModelCount := 6
	actualModelCount := len(rev.ModelQty)
	if actualModelCount < minModelCount {
		actualModelCount = minModelCount
	}

	// Write Model names and quantities
	modelNames := make([]string, actualModelCount)
	for i := 0; i < actualModelCount; i++ {
		modelNames[i] = fmt.Sprintf("Model %c", 'A'+i)
		col := getColName(modelStartCol + i)

		// Model name (row 4)
		f.SetCellValue("SMD", fmt.Sprintf("%s4", col), modelNames[i])
		f.SetCellValue("PTH", fmt.Sprintf("%s4", col), modelNames[i])
		f.SetCellValue("BOTTOM", fmt.Sprintf("%s4", col), modelNames[i])

		// Model quantity (row 5)
		qty := rev.ModelQty[string(rune('A'+i))]
		if qty == 0 {
			qty = 1
		}
		f.SetCellValue("SMD", fmt.Sprintf("%s5", col), qty)
		f.SetCellValue("PTH", fmt.Sprintf("%s5", col), qty)
		f.SetCellValue("BOTTOM", fmt.Sprintf("%s5", col), qty)
	}

	// Calculate Remark column position (after all Model columns)
	remarkCol := getColName(modelStartCol + actualModelCount)

	// Get style IDs for alternating rows
	styleRow6, _ := f.GetCellStyle("SMD", "A6")
	styleRow7, _ := f.GetCellStyle("SMD", "A7")

	// 8.2.5 - Write part data to each sheet
	sheets := []string{"SMD", "PTH", "BOTTOM"}
	for _, sheet := range sheets {
		rowIndex := 6

		// Filter parts by sheet type
		var sheetParts []PartData
		for _, part := range options.PartData {
			if strings.ToUpper(part.Type) == strings.ToUpper(sheet) {
				sheetParts = append(sheetParts, part)
			}
		}

		for _, part := range sheetParts {
			// Alternate row styles
			var rowStyle int
			if rowIndex%2 == 0 {
				rowStyle = styleRow6
			} else {
				rowStyle = styleRow7
			}

			// Apply style to columns A-J
			for col := 'A'; col <= 'J'; col++ {
				cell := fmt.Sprintf("%c%d", col, rowIndex)
				f.SetCellStyle(sheet, cell, cell, rowStyle)
			}

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

				modelName := fmt.Sprintf("%c", 'A'+i)
				if selectedPN, ok := part.Selections[modelName]; ok && selectedPN != "" {
					if part.SupplierPn == selectedPN {
						f.SetCellValue(sheet, cell, "V")
					}
				}
			}

			// Write Remark (column after all Model columns)
			f.SetCellValue(sheet, fmt.Sprintf("%s%d", remarkCol, rowIndex), part.Remark)

			rowIndex++

			// Write second sources
			for _, ss := range part.SecondSources {
				// Apply alternating style
				var rowStyle int
				if rowIndex%2 == 0 {
					rowStyle = styleRow6
				} else {
					rowStyle = styleRow7
				}

				for col := 'A'; col <= 'J'; col++ {
					cell := fmt.Sprintf("%c%d", col, rowIndex)
					f.SetCellStyle(sheet, cell, cell, rowStyle)
				}

				// Second sources don't have Item or Location
				f.SetCellValue(sheet, fmt.Sprintf("B%d", rowIndex), ss.HHPN)
				f.SetCellValue(sheet, fmt.Sprintf("D%d", rowIndex), ss.Description)
				f.SetCellValue(sheet, fmt.Sprintf("E%d", rowIndex), ss.Supplier)
				f.SetCellValue(sheet, fmt.Sprintf("F%d", rowIndex), ss.SupplierPn)

				// Formulas for second sources
				formulaI := fmt.Sprintf("=G%d*J%d", rowIndex, rowIndex)
				f.SetCellFormula(sheet, fmt.Sprintf("I%d", rowIndex), formulaI)

				formulaJ := generateMatrixSelectionFormula(modelStartCol, actualModelCount, rowIndex, rev.ModelQty)
				f.SetCellFormula(sheet, fmt.Sprintf("J%d", rowIndex), formulaJ)

				// Model selections for second sources
				for i := 0; i < actualModelCount; i++ {
					col := getColName(modelStartCol + i)
					cell := fmt.Sprintf("%s%d", col, rowIndex)

					modelName := fmt.Sprintf("%c", 'A'+i)
					if selectedPN, ok := part.Selections[modelName]; ok && selectedPN != "" {
						if ss.SupplierPn == selectedPN {
							f.SetCellValue(sheet, cell, "V")
						}
					}
				}

				// Remark for second source
				f.SetCellValue(sheet, fmt.Sprintf("%s%d", remarkCol, rowIndex), ss.Description)

				rowIndex++
			}
		}
	}

	// Save to output path
	fileName := generateMatrixFileName(rev, date)
	var outputPath string
	if options.OutputDir != "" {
		outputPath = filepath.Join(options.OutputDir, fileName)
	} else {
		outputPath = fileName
	}

	if err := f.SaveAs(outputPath); err != nil {
		return nil, fmt.Errorf("failed to save Matrix: %w", err)
	}

	return []string{outputPath}, nil
}

// generateMatrixSelectionFormula generates the formula for J column
// See product-spec section 8.2.5.4
func generateMatrixSelectionFormula(startCol int, modelCount int, row int, modelQty map[string]int) string {
	var parts []string

	for i := 0; i < modelCount; i++ {
		col := getColName(startCol + i)
		modelName := string(rune('A' + i))

		// Get the qty for this model (absolute reference to row 5)
		qty := modelQty[modelName]
		if qty == 0 {
			qty = 1
		}

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
		"{{.ProjectCode}}":       rev.ProjectCode,
		"{{.Description}}":       rev.Description,
		"{{.SchematicVersion}}":  rev.SchematicVersion,
		"{{.PCBVersion}}":        rev.PCBVersion,
		"{{.PCAPN}}":             rev.PCAPN,
		"{{.Date}}":              date,
	}

	// Add model quantities
	modelNames := []string{"A", "B", "C", "D", "E", "F"}
	for _, modelName := range modelNames {
		qty := rev.ModelQty[modelName]
		if qty == 0 {
			qty = 1
		}
		tags[fmt.Sprintf("{{.ModelQty%s}}", modelName)] = fmt.Sprintf("%d", qty)
	}

	if err := applyTagReplacement(f, tags); err != nil {
		return nil, err
	}

	// 8.2.4 - Dynamic Model column generation (minimum 6 models)
	modelStartCol := 10 // K
	minModelCount := 6
	actualModelCount := len(rev.ModelQty)
	if actualModelCount < minModelCount {
		actualModelCount = minModelCount
	}

	// Write Model names and quantities for all sheets
	sheets := []string{"SMD", "PTH", "BOTTOM"}
	for _, sheet := range sheets {
		for i := 0; i < actualModelCount; i++ {
			col := getColName(modelStartCol + i)
			modelName := fmt.Sprintf("Model %c", 'A'+i)

			// Model name (row 4) - always write, even if no data
			f.SetCellValue(sheet, fmt.Sprintf("%s4", col), modelName)

			// Model quantity (row 5)
			qty := rev.ModelQty[string(rune('A'+i))]
			if qty == 0 {
				// Use default if not defined
				qty = 1
			}
			f.SetCellValue(sheet, fmt.Sprintf("%s5", col), qty)
		}
	}

	// Get style IDs for alternating rows
	styleRow6, _ := f.GetCellStyle("SMD", "A6")
	styleRow7, _ := f.GetCellStyle("SMD", "A7")

	// 8.2.6 - Filter parts by criteria
	filteredParts := filterMatrixParts(parts, rev.Mode)

	// 8.2.5 - Write part data to each sheet
	for _, sheet := range sheets {
		rowIndex := 6

		// Filter parts by sheet type
		var sheetParts []PartData
		for _, part := range filteredParts {
			if strings.ToUpper(part.Type) == strings.ToUpper(sheet) {
				sheetParts = append(sheetParts, part)
			}
		}

		for _, part := range sheetParts {
			// Alternate row styles
			var rowStyle int
			if rowIndex%2 == 0 {
				rowStyle = styleRow6
			} else {
				rowStyle = styleRow7
			}

			// Apply style to columns A-J
			for col := 'A'; col <= 'J'; col++ {
				cell := fmt.Sprintf("%c%d", col, rowIndex)
				f.SetCellStyle(sheet, cell, cell, rowStyle)
			}

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

				modelName := fmt.Sprintf("%c", 'A'+i)
				if selectedPN, ok := part.Selections[modelName]; ok && selectedPN != "" {
					if part.SupplierPn == selectedPN {
						f.SetCellValue(sheet, cell, "V")
					}
				}
			}

			// 8.2.5.5 - Write Remark (column after all Model columns)
			remarkCol := getColName(modelStartCol + actualModelCount)
			f.SetCellValue(sheet, fmt.Sprintf("%s%d", remarkCol, rowIndex), part.Remark)

			rowIndex++

			// Write second sources
			for _, ss := range part.SecondSources {
				// Apply alternating style
				var rowStyle int
				if rowIndex%2 == 0 {
					rowStyle = styleRow6
				} else {
					rowStyle = styleRow7
				}

				for col := 'A'; col <= 'J'; col++ {
					cell := fmt.Sprintf("%c%d", col, rowIndex)
					f.SetCellStyle(sheet, cell, cell, rowStyle)
				}

				// Second sources don't have Item or Location
				f.SetCellValue(sheet, fmt.Sprintf("B%d", rowIndex), ss.HHPN)
				f.SetCellValue(sheet, fmt.Sprintf("D%d", rowIndex), ss.Description)
				f.SetCellValue(sheet, fmt.Sprintf("E%d", rowIndex), ss.Supplier)
				f.SetCellValue(sheet, fmt.Sprintf("F%d", rowIndex), ss.SupplierPn)

				// Formulas for second sources
				formulaI := fmt.Sprintf("=G%d*J%d", rowIndex, rowIndex)
				f.SetCellFormula(sheet, fmt.Sprintf("I%d", rowIndex), formulaI)

				formulaJ := generateMatrixSelectionFormula(modelStartCol, actualModelCount, rowIndex, rev.ModelQty)
				f.SetCellFormula(sheet, fmt.Sprintf("J%d", rowIndex), formulaJ)

				// Model selections for second sources
				for i := 0; i < actualModelCount; i++ {
					col := getColName(modelStartCol + i)
					cell := fmt.Sprintf("%s%d", col, rowIndex)

					modelName := fmt.Sprintf("%c", 'A'+i)
					if selectedPN, ok := part.Selections[modelName]; ok && selectedPN != "" {
						if ss.SupplierPn == selectedPN {
							f.SetCellValue(sheet, cell, "V")
						}
					}
				}

				// Remark for second source
				f.SetCellValue(sheet, fmt.Sprintf("%s%d", remarkCol, rowIndex), ss.Description)

				rowIndex++
			}
		}
	}

	// Save to output path
	fileName := generateMatrixFileName(rev, date)
	var outputPath string
	if options.OutputDir != "" {
		outputPath = filepath.Join(options.OutputDir, fileName)
	} else {
		outputPath = fileName
	}

	if err := f.SaveAs(outputPath); err != nil {
		return nil, fmt.Errorf("failed to save Matrix: %w", err)
	}

	return []string{outputPath}, nil
}

// filterMatrixParts filters parts based on Matrix export criteria
// See product-spec section 8.2.6
func filterMatrixParts(parts []PartData, mode string) []PartData {
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
