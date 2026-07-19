//go:build ignore

package main

import (
	"fmt"
	"log"

	"github.com/xuri/excelize/v2"
)

func main() {
	if err := generateBigMatrixTemplate(); err != nil {
		log.Fatalf("Failed to generate BigMatrix template: %v", err)
	}
	fmt.Println("Generated bigmatrix.xlsx successfully")

	if err := generateMatrixTemplate(); err != nil {
		log.Fatalf("Failed to generate Matrix template: %v", err)
	}
	fmt.Println("Generated matrix.xlsx successfully")
}

// generateBigMatrixTemplate generates the BigMatrix template
func generateBigMatrixTemplate() error {
	f := excelize.NewFile()

	// Rename default sheet to "BigMatrix"
	if err := f.SetSheetName("Sheet1", "BigMatrix"); err != nil {
		return err
	}

	// Set column widths
	cols := map[string]float64{
		"A": 8, "B": 25, "C": 40, "D": 20, "E": 18, "F": 8, "G": 15,
		"H": 8, "I": 8, "J": 8, "K": 8, "L": 8, "M": 8, "N": 8, "O": 8,
		"P": 8, "Q": 8, "R": 8, "S": 8,
	}
	for col, width := range cols {
		f.SetColWidth("BigMatrix", col, col, width)
	}

	// Row 2: Project codes (will be merged later)
	f.SetCellValue("BigMatrix", "B2", "{{.ProjectCode}}")
	f.SetCellValue("BigMatrix", "H2", "{{.ProjectCode}}")

	// Row 3: Revision IDs (will be merged later)
	f.SetCellValue("BigMatrix", "B3", "{{.RevisionID}}")
	f.SetCellValue("BigMatrix", "H3", "{{.RevisionID}}")

	// Row 4: Model names (A, B, C, ...)
	f.SetCellValue("BigMatrix", "H4", "{{.ModelA}}")
	f.SetCellValue("BigMatrix", "I4", "{{.ModelB}}")
	f.SetCellValue("BigMatrix", "J4", "{{.ModelC}}")

	// Row 5: Model quantities
	f.SetCellValue("BigMatrix", "H5", "{{.QtyA}}")
	f.SetCellValue("BigMatrix", "I5", "{{.QtyB}}")
	f.SetCellValue("BigMatrix", "J5", "{{.QtyC}}")

	// Row 6: Header row with tags
	headers := map[string]string{
		"A6": "Item",
		"B6": "HHPN",
		"C6": "Description",
		"D6": "Supplier",
		"E6": "Supplier PN",
		"F6": "Qty",
		"G6": "Location",
	}
	for cell, value := range headers {
		f.SetCellValue("BigMatrix", cell, value)
	}

	// Row 6 - Prototype row for style (background color 1)
	applyStyle(f, "BigMatrix", 6, "#E6F2FF") // Light blue

	// Row 7 - Prototype row for style (background color 2)
	applyStyle(f, "BigMatrix", 7, "#F2E6FF") // Light purple

	// Set header row style (bold, centered)
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	for col := 'A'; col <= 'G'; col++ {
		cell := string(rune(col)) + "6"
		f.SetCellStyle("BigMatrix", cell, cell, headerStyle)
	}

	// Save to file
	if err := f.SaveAs("../../backend/excel/template/bigmatrix.xlsx"); err != nil {
		return err
	}
	return nil
}

// generateMatrixTemplate generates the Matrix template with SMD, PTH, BOTTOM sheets
func generateMatrixTemplate() error {
	sheets := []string{"SMD", "PTH", "BOTTOM"}

	var f *excelize.File

	for idx, sheetName := range sheets {
		if idx == 0 {
			// Rename default sheet
			f = excelize.NewFile()
			f.SetSheetName("Sheet1", sheetName)
		} else {
			// Create new sheet
			_, err := f.NewSheet(sheetName)
			if err != nil {
				return err
			}
		}

		// Set column widths
		cols := map[string]float64{
			"A": 6, "B": 25, "D": 40, "E": 20, "F": 18, "G": 8, "H": 15,
			"I": 10, "J": 10, "K": 8, "L": 8, "M": 8, "N": 8, "O": 8, "P": 8,
		}
		for col, width := range cols {
			f.SetColWidth(sheetName, col, col, width)
		}

		// Row 2: Project Code
		f.SetCellValue(sheetName, "B2", "{{.ProjectCode}}")

		// Row 3: Description, Schematic Version, PCB Version, PCA PN, Date
		f.SetCellValue(sheetName, "B3", "{{.Description}}")
		f.SetCellValue(sheetName, "D3", "{{.SchematicVersion}}")
		f.SetCellValue(sheetName, "F3", "{{.PCBVersion}}")
		f.SetCellValue(sheetName, "H3", "{{.PCAPN}}")
		f.SetCellValue(sheetName, "J3", "{{.Date}}")

		// Row 4: Model headers starting from K
		modelCols := []string{"K", "L", "M", "N", "O", "P"}
		modelNames := []string{"Model A", "Model B", "Model C", "Model D", "Model E", "Model F"}
		for i, col := range modelCols {
			f.SetCellValue(sheetName, col+"4", modelNames[i])
			// Row 5 for qty
			f.SetCellValue(sheetName, col+"5", fmt.Sprintf("{{.ModelQty%c}}", 'A'+i))
		}

		// Row 6: Header row
		headers := map[string]string{
			"A6": "Item",
			"B6": "HHPN",
			"D6": "Description",
			"E6": "Supplier",
			"F6": "Supplier PN",
			"G6": "Qty",
			"H6": "Location",
			"I6": "Total Qty",
			"J6": "Selected Qty",
		}
		for cell, value := range headers {
			f.SetCellValue(sheetName, cell, value)
		}

		// Apply styles
		applyStyle(f, sheetName, 6, "#E6F2FF") // Light blue
		applyStyle(f, sheetName, 7, "#F2E6FF") // Light purple

		// Set header row style
		headerStyle, _ := f.NewStyle(&excelize.Style{
			Font:      &excelize.Font{Bold: true},
			Alignment: &excelize.Alignment{Horizontal: "center"},
		})
		for _, col := range []string{"A", "B", "D", "E", "F", "G", "H", "I", "J"} {
			cell := col + "6"
			f.SetCellStyle(sheetName, cell, cell, headerStyle)
		}
	}

	// Set active sheet to SMD (first sheet)
	f.SetActiveSheet(0)

	if err := f.SaveAs("../../backend/excel/template/matrix.xlsx"); err != nil {
		return err
	}
	return nil
}

func applyStyle(f *excelize.File, sheet string, row int, color string) {
	style, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{color}, Pattern: 1},
	})
	// Apply to columns A through J
	for col := 'A'; col <= 'J'; col++ {
		cell := string(rune(col)) + fmt.Sprintf("%d", row)
		f.SetCellStyle(sheet, cell, cell, style)
	}
}
