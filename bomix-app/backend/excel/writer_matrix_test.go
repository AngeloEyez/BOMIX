package excel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bomix-app/backend/types"
	"github.com/xuri/excelize/v2"
)

func TestGenerateMatrixSelectionFormula(t *testing.T) {
	tests := []struct {
		name       string
		startCol   int
		modelCount int
		row        int
		modelQty   map[string]int
		expect     string
	}{
		{
			name:       "6 models",
			startCol:   10, // K
			modelCount: 6,
			row:        6,
			modelQty:   map[string]int{"A": 1, "B": 1, "C": 1, "D": 1, "E": 1, "F": 1},
			expect:     "IF(EXACT(K6,\"V\"),K$5,0)+IF(EXACT(L6,\"V\"),L$5,0)+IF(EXACT(M6,\"V\"),M$5,0)+IF(EXACT(N6,\"V\"),N$5,0)+IF(EXACT(O6,\"V\"),O$5,0)+IF(EXACT(P6,\"V\"),P$5,0)",
		},
		{
			name:       "8 models",
			startCol:   10, // K
			modelCount: 8,
			row:        10,
			modelQty:   map[string]int{"A": 2, "B": 1, "C": 3, "D": 1, "E": 2, "F": 1, "G": 1, "H": 2},
			expect:     "IF(EXACT(K10,\"V\"),K$5,0)+IF(EXACT(L10,\"V\"),L$5,0)+IF(EXACT(M10,\"V\"),M$5,0)+IF(EXACT(N10,\"V\"),N$5,0)+IF(EXACT(O10,\"V\"),O$5,0)+IF(EXACT(P10,\"V\"),P$5,0)+IF(EXACT(Q10,\"V\"),Q$5,0)+IF(EXACT(R10,\"V\"),R$5,0)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateMatrixSelectionFormula(tt.startCol, tt.modelCount, tt.row, tt.modelQty)
			if result != tt.expect {
				t.Errorf("generateMatrixSelectionFormula() = %s\nwant %s", result, tt.expect)
			}
		})
	}
}

func TestGenerateMatrixFileName(t *testing.T) {
	tests := []struct {
		name   string
		rev    RevisionData
		date   string
		expect string
	}{
		{
			name: "Standard Matrix export",
			rev: RevisionData{
				ProjectCode: "TANGLED",
				Phase:       "PV",
				Version:     "0.3",
			},
			date:   "20260717",
			expect: "TANGLED_EZBOM_PV_0.3_MatrixBOM_20260717.xlsx",
		},
		{
			name: "Another project",
			rev: RevisionData{
				ProjectCode: "FY27",
				Phase:       "DB",
				Version:     "0.1",
			},
			date:   "20260801",
			expect: "FY27_EZBOM_DB_0.1_MatrixBOM_20260801.xlsx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateMatrixFileName(tt.rev, tt.date)
			if result != tt.expect {
				t.Errorf("generateMatrixFileName() = %s, want %s", result, tt.expect)
			}
		})
	}
}

func TestFilterMatrixParts(t *testing.T) {
	parts := []PartData{
		{
			Supplier:    "Samsung",
			SupplierPn:  "CL05B104KO5NNNC",
			CCL:         "Y",
			BOMStatus:   "I",
			Type:        "SMD",
			Description: "Test Part 1",
		},
		{
			Supplier:    "Murata",
			SupplierPn:  "GRM155B81C105KE19D",
			CCL:         "Y",
			BOMStatus:   "P",
			Type:        "SMD",
			Description: "Test Part 2 (NPI)",
		},
		{
			Supplier:    "Taiyo Yuden",
			SupplierPn:  "UMK105B7105KV-F",
			CCL:         "Y",
			BOMStatus:   "M",
			Type:        "PTH",
			Description: "Test Part 3 (MP)",
		},
		{
			Supplier:    "Yageo",
			SupplierPn:  "CC0402KRX7R9BB104",
			CCL:         "Y",
			BOMStatus:   "X",
			Type:        "SMD",
			Description: "Test Part 4 (Not Installed)",
		},
		{
			Supplier:    "Kemet",
			SupplierPn:  "C0603C104K5RACTU",
			CCL:         "N",
			BOMStatus:   "I",
			Type:        "SMD",
			Description: "Test Part 5 (Not CCL)",
		},
	}

	// Test NPI mode
	npiFiltered := filterMatrixParts(parts, "NPI")
	if len(npiFiltered) != 2 {
		t.Errorf("NPI filter: expected 2 parts, got %d", len(npiFiltered))
	}
	for _, p := range npiFiltered {
		if p.BOMStatus != "I" && p.BOMStatus != "P" {
			t.Errorf("NPI filter: part %s has invalid status %s", p.SupplierPn, p.BOMStatus)
		}
		if p.CCL != "Y" {
			t.Errorf("NPI filter: part %s is not CCL", p.SupplierPn)
		}
	}

	// Test MP mode
	mpFiltered := filterMatrixParts(parts, "MP")
	if len(mpFiltered) != 2 {
		t.Errorf("MP filter: expected 2 parts, got %d", len(mpFiltered))
	}
	for _, p := range mpFiltered {
		if p.BOMStatus != "I" && p.BOMStatus != "M" {
			t.Errorf("MP filter: part %s has invalid status %s", p.SupplierPn, p.BOMStatus)
		}
		if p.CCL != "Y" {
			t.Errorf("MP filter: part %s is not CCL", p.SupplierPn)
		}
	}
}

func TestExportMatrix_Integration(t *testing.T) {
	// Create a temporary directory for output
	tmpDir, err := os.MkdirTemp("", "bomix-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create writer
	writer, err := NewWriter()
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	// Prepare test data
	options := ExportOptions{
		Format:      types.FormatMatrix,
		OutputPath:  filepath.Join(tmpDir, "test_matrix.xlsx"),
		Description: "Test Matrix Export",
		PartData: []PartData{
			{
				Item:        "1",
				HHPN:        "34065Y600-GRT-H",
				Description: "CAP,22uF,+/-20%,X5R,6.3V,SMD0603",
				Supplier:    "Samsung",
				SupplierPn:  "CL05B104KO5NNNC",
				Qty:         4,
				Location:    "C1,C2,C3,C4",
				Type:        "SMD",
				BOMStatus:   "I",
				CCL:         "Y",
				Remark:      "",
				Selections: map[string]string{
					"A": "CL05B104KO5NNNC",
					"B": "CL05B104KO5NNNC",
				},
			},
			{
				Item:        "2",
				HHPN:        "34065Y601-GRT-H",
				Description: "RES,10K,+/-1%,0603",
				Supplier:    "Yageo",
				SupplierPn:  "RC0603FR-0710KL",
				Qty:         12,
				Location:    "R1,R2,R3,R4,R5,R6,R7,R8,R9,R10,R11,R12",
				Type:        "SMD",
				BOMStatus:   "I",
				CCL:         "Y",
				Remark:      "",
				Selections: map[string]string{
					"A": "RC0603FR-0710KL",
					"B": "RC0603FR-0710KL",
				},
			},
			{
				Item:        "3",
				HHPN:        "34065Y602-GRT-H",
				Description: "CAP,100nF,50V,SMD0402",
				Supplier:    "Murata",
				SupplierPn:  "GRM155B71H104KA12D",
				Qty:         8,
				Location:    "C5,C6,C7,C8,C9,C10,C11,C12",
				Type:        "SMD",
				BOMStatus:   "I",
				CCL:         "Y",
				Remark:      "",
				Selections: map[string]string{
					"A": "GRM155B71H104KA12D",
				},
			},
		},
	}

	// Export
	outputPaths, err := writer.ExportExcel(options)
	if err != nil {
		t.Fatalf("ExportExcel failed: %v", err)
	}

	if len(outputPaths) != 1 {
		t.Errorf("Expected 1 output path, got %d", len(outputPaths))
	}

	// Verify the output file exists and is valid
	outputPath := outputPaths[0]
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatalf("Output file does not exist: %s", outputPath)
	}

	// Try to open the file with excelize
	f, err := excelize.OpenFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to open exported file: %v", err)
	}
	defer f.Close()

	// Verify all three sheets exist
	expectedSheets := []string{"SMD", "PTH", "BOTTOM"}
	sheets := f.GetSheetList()

	if len(sheets) != 3 {
		t.Errorf("Expected 3 sheets, got %d", len(sheets))
	}

	for _, expectedSheet := range expectedSheets {
		found := false
		for _, sheet := range sheets {
			if sheet == expectedSheet {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected sheet '%s' not found", expectedSheet)
		}
	}

	// Verify data was written to SMD sheet
	smdVal, err := f.GetCellValue("SMD", "A6")
	if err != nil {
		t.Fatalf("Failed to read cell A6 from SMD: %v", err)
	}
	// A6 should be "Item" header or first item data
	if !strings.Contains(smdVal, "Item") && smdVal != "1" {
		t.Errorf("Unexpected value in SMD A6: %s", smdVal)
	}

	// Verify formula in I column
	formulaI, err := f.GetCellFormula("SMD", "I6")
	if err != nil {
		t.Fatalf("Failed to get formula from I6: %v", err)
	}
	if formulaI != "=G6*J6" {
		t.Errorf("Unexpected formula in I6: %s, want =G6*J6", formulaI)
	}

	// Verify formula in J column
	formulaJ, err := f.GetCellFormula("SMD", "J6")
	if err != nil {
		t.Fatalf("Failed to get formula from J6: %v", err)
	}
	if !strings.Contains(formulaJ, "IF(EXACT") {
		t.Errorf("Unexpected formula in J6: %s", formulaJ)
	}

	// Verify Model selections were written
	modelAVal, err := f.GetCellValue("SMD", "K6")
	if err != nil {
		t.Fatalf("Failed to read cell K6 from SMD: %v", err)
	}
	if modelAVal != "V" {
		t.Errorf("Expected 'V' in K6 (Model A selection), got '%s'", modelAVal)
	}

	// Verify data exists in PTH and BOTTOM sheets (even if empty, sheets should be there)
	pthVal, err := f.GetCellValue("PTH", "A6")
	if err != nil {
		t.Fatalf("Failed to read cell A6 from PTH: %v", err)
	}
	// PTH should be empty in our test data
	if pthVal != "" {
		t.Logf("PTH A6 has value: %s (expected empty)", pthVal)
	}

	bottomVal, err := f.GetCellValue("BOTTOM", "A6")
	if err != nil {
		t.Fatalf("Failed to read cell A6 from BOTTOM: %v", err)
	}
	// BOTTOM should be empty in our test data
	if bottomVal != "" {
		t.Logf("BOTTOM A6 has value: %s (expected empty)", bottomVal)
	}

	t.Logf("Successfully exported Matrix to: %s", outputPath)
}

func TestApplyRowStyle(t *testing.T) {
	f := excelize.NewFile()
	wb := &ExcelizeWorkbook{f: f}
	_ = wb
	defer f.Close()

	// Create a style
	style, err := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E6F2FF"}, Pattern: 1},
	})
	if err != nil {
		t.Fatalf("Failed to create style: %v", err)
	}

	// Apply row style
	applyRowStyle(f, "Sheet1", 6, style)

	// Verify style was applied
	cellStyle, err := f.GetCellStyle("Sheet1", "A6")
	if err != nil {
		t.Fatalf("Failed to get cell style: %v", err)
	}
	if cellStyle != style {
		t.Error("Style was not applied correctly")
	}

	// Verify multiple columns in the row have the style
	for col := 'A'; col <= 'J'; col++ {
		cell := string(rune(col)) + "6"
		cellStyle, err := f.GetCellStyle("Sheet1", cell)
		if err != nil {
			t.Fatalf("Failed to get cell style for %s: %v", cell, err)
		}
		if cellStyle != style {
			t.Errorf("Style not applied to %s", cell)
		}
	}
}

func TestWriteModelSelections(t *testing.T) {
	f := excelize.NewFile()
	wb := &ExcelizeWorkbook{f: f}
	_ = wb
	defer f.Close()

	// Test data
	row := 6
	modelStartCol := 10 // K
	selections := map[string]string{
		"A": "CL05B104KO5NNNC",
		"B": "CL05B104KO5NNNC",
		"C": "", // Not selected
	}
	modelNames := []string{"A", "B", "C"}

	// Write selections
	writeModelSelections(f, "Sheet1", row, modelStartCol, selections, modelNames)

	// Verify selections
	valA, err := f.GetCellValue("Sheet1", "K6")
	if err != nil {
		t.Fatalf("Failed to read K6: %v", err)
	}
	if valA != "V" {
		t.Errorf("Expected 'V' in K6, got '%s'", valA)
	}

	valB, err := f.GetCellValue("Sheet1", "L6")
	if err != nil {
		t.Fatalf("Failed to read L6: %v", err)
	}
	if valB != "V" {
		t.Errorf("Expected 'V' in L6, got '%s'", valB)
	}

	valC, err := f.GetCellValue("Sheet1", "M6")
	if err != nil {
		t.Fatalf("Failed to read M6: %v", err)
	}
	if valC != "" {
		t.Errorf("Expected empty M6, got '%s'", valC)
	}
}
