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
			CCL:         true,
			BOMStatus:   "I",
			Type:        "SMD",
			Description: "Test Part 1",
		},
		{
			Supplier:    "Murata",
			SupplierPn:  "GRM155B81C105KE19D",
			CCL:         true,
			BOMStatus:   "P",
			Type:        "SMD",
			Description: "Test Part 2 (NPI)",
		},
		{
			Supplier:    "Taiyo Yuden",
			SupplierPn:  "UMK105B7105KV-F",
			CCL:         true,
			BOMStatus:   "M",
			Type:        "PTH",
			Description: "Test Part 3 (MP)",
		},
		{
			Supplier:    "Yageo",
			SupplierPn:  "CC0402KRX7R9BB104",
			CCL:         true,
			BOMStatus:   "X",
			Type:        "SMD",
			Description: "Test Part 4 (Not Installed)",
		},
		{
			Supplier:    "Kemet",
			SupplierPn:  "C0603C104K5RACTU",
			CCL:         false,
			BOMStatus:   "I",
			Type:        "SMD",
			Description: "Test Part 5 (Not CCL)",
		},
	}

	// Test filterMatrixParts
	filtered := filterMatrixParts(parts)
	// CCL=true, status!=X: Part 1(I), Part 2(P), Part 3(M) => 3 parts
	if len(filtered) != 3 {
		t.Errorf("filterMatrixParts: expected 3 parts, got %d", len(filtered))
	}
	for _, p := range filtered {
		if p.BOMStatus == "X" {
			t.Errorf("filterMatrixParts: part %s has invalid status X", p.SupplierPn)
		}
		if !p.CCL {
			t.Errorf("filterMatrixParts: part %s is not CCL", p.SupplierPn)
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
	writer, err := NewWriter(nil)
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
				CCL:         true,
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
				CCL:         true,
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
				CCL:         true,
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

// TestExportMatrix_GroupZebraStriping 驗證 Matrix 匯出時斑馬紋是依據「物料 Group」切換
func TestExportMatrix_GroupZebraStriping(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bomix-matrix-zebra-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writer, err := NewWriter(nil)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	options := ExportOptions{
		Format:     types.FormatMatrix,
		OutputPath: filepath.Join(tmpDir, "matrix_zebra_test.xlsx"),
		PartData: []PartData{
			// Group 0: Main Part 1 + 1 Second Source -> Rows 6 and 7 in SMD sheet
			{
				Item:        "1",
				HHPN:        "MAIN_1",
				Description: "Resistor 10K",
				Supplier:    "YAGEO",
				SupplierPn:  "R10K",
				Qty:         1,
				Type:        "SMD",
				CCL:         true,
				SecondSources: []SecondSourceData{
					{
						HHPN:        "ALT_1",
						Supplier:    "UNI-ROYAL",
						SupplierPn:  "R10K_ALT",
						Description: "Resistor 10K Alt",
					},
				},
			},
			// Group 1: Main Part 2 -> Row 8 in SMD sheet
			{
				Item:        "2",
				HHPN:        "MAIN_2",
				Description: "Capacitor 10uF",
				Supplier:    "MURATA",
				SupplierPn:  "C10U",
				Qty:         1,
				Type:        "SMD",
				CCL:         true,
			},
		},
	}

	paths, err := writer.ExportExcel(options)
	if err != nil {
		t.Fatalf("ExportExcel failed: %v", err)
	}

	f, err := excelize.OpenFile(paths[0])
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	defer f.Close()

	// In Matrix template, A6 (styleRow6) and A7 (styleRow7) are alternating styles
	styleRow6, err6 := f.GetCellStyle("SMD", "A6")
	styleRow7, err7 := f.GetCellStyle("SMD", "A7")
	styleRow8, err8 := f.GetCellStyle("SMD", "A8")

	if err6 != nil || err7 != nil || err8 != nil {
		t.Fatalf("Failed to get cell styles: %v, %v, %v", err6, err7, err8)
	}

	// 1. Group 0 的主料 (Row 6) 與二源料 (Row 7) 樣式應完全一致
	if styleRow6 != styleRow7 {
		t.Errorf("Group 0 main part (A6, style %d) and second source (A7, style %d) should have the SAME style", styleRow6, styleRow7)
	}

	// 2. 不同 Group (Group 0 vs Group 1) 的樣式應不相同
	if styleRow6 == styleRow8 {
		t.Errorf("Group 0 (A6, style %d) and Group 1 (A8, style %d) should have DIFFERENT styles", styleRow6, styleRow8)
	}
}

// TestExportMatrix_PhaseAndVersionTagReplacement 驗證 Matrix 匯出時 {{.Phase}} 與 {{.Version}} 標籤正確被替換
func TestExportMatrix_PhaseAndVersionTagReplacement(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()

	// 模擬範本，在 B2 填入 Phase: {{.Phase}}, 在 B3 填入 Version: {{.Version}}
	_ = f.SetCellValue("Sheet1", "B2", "Phase: {{.Phase}}")
	_ = f.SetCellValue("Sheet1", "B3", "Ver: {{.Version}}")

	tags := map[string]string{
		"{{.Phase}}":   "EVT",
		"{{.Version}}": "1.2",
	}

	err := applyTagReplacement(f, tags)
	if err != nil {
		t.Fatalf("applyTagReplacement failed: %v", err)
	}

	valB2, _ := f.GetCellValue("Sheet1", "B2")
	valB3, _ := f.GetCellValue("Sheet1", "B3")

	if valB2 != "Phase: EVT" {
		t.Errorf("Expected B2 to be 'Phase: EVT', got '%s'", valB2)
	}
	if valB3 != "Ver: 1.2" {
		t.Errorf("Expected B3 to be 'Ver: 1.2', got '%s'", valB3)
	}
}

// TestExportMatrix_MainAndSecondSourceRemark 驗證主料與 2nd Source 的 Remark 是否精確寫入 Matrix 的 Remark 欄 (R 欄)
func TestExportMatrix_MainAndSecondSourceRemark(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bomix-remark-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writer, err := NewWriter(nil)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	options := ExportOptions{
		Format:     types.FormatMatrix,
		OutputPath: filepath.Join(tmpDir, "remark_test_matrix.xlsx"),
		PartData: []PartData{
			{
				Item:        "1",
				HHPN:        "MAIN_PN_01",
				Description: "Resistor 100K",
				Supplier:    "YAGEO",
				SupplierPn:  "R100K",
				Qty:         2,
				Type:        "SMD",
				CCL:         true,
				Remark:      "Main Part Remark 01",
				SecondSources: []SecondSourceData{
					{
						HHPN:        "ALT_PN_01",
						Supplier:    "UNI-ROYAL",
						SupplierPn:  "R100K_ALT",
						Description: "Resistor 100K Alt",
						Remark:      "Second Source Remark 01",
					},
				},
			},
		},
	}

	paths, err := writer.ExportExcel(options)
	if err != nil {
		t.Fatalf("ExportExcel failed: %v", err)
	}

	f, err := excelize.OpenFile(paths[0])
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	defer f.Close()

	// 7 models -> Remark is column R (col 17)
	// Row 6: Main Part -> R6
	// Row 7: Second Source -> R7
	mainRemark, err1 := f.GetCellValue("SMD", "R6")
	ssRemark, err2 := f.GetCellValue("SMD", "R7")

	if err1 != nil || err2 != nil {
		t.Fatalf("Failed to read R6/R7: %v, %v", err1, err2)
	}

	if mainRemark != "Main Part Remark 01" {
		t.Errorf("Expected R6 (Main Part Remark) to be 'Main Part Remark 01', got '%s'", mainRemark)
	}

	if ssRemark != "Second Source Remark 01" {
		t.Errorf("Expected R7 (Second Source Remark) to be 'Second Source Remark 01', got '%s'", ssRemark)
	}
}

// TestExportMatrix_ModelNameMapping 驗證 ModelName 為 "Model A" 或 "Model 1" 時，Qty 與 "V" 勾選均能正常匯出
func TestExportMatrix_ModelNameMapping(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bomix-modelname-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writer, err := NewWriter(nil)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	options := ExportOptions{
		Format:     types.FormatMatrix,
		OutputPath: filepath.Join(tmpDir, "modelname_test_matrix.xlsx"),
		Revisions: []RevisionData{
			{
				ProjectCode: "PROJ_M",
				Phase:       "EVT",
				Version:     "0.1",
				ModelQty: map[string]int{
					"Model A": 2,
					"Model B": 5,
				},
			},
		},
		PartData: []PartData{
			{
				Item:        "1",
				HHPN:        "MAIN_PN_01",
				Description: "Resistor 100K",
				Supplier:    "YAGEO",
				SupplierPn:  "R100K",
				Qty:         2,
				Type:        "SMD",
				CCL:         true,
				Selections: map[string]string{
					"Model A": "R100K",
					"Model B": "R100K",
				},
			},
		},
	}

	paths, err := writer.ExportExcel(options)
	if err != nil {
		t.Fatalf("ExportExcel failed: %v", err)
	}

	f, err := excelize.OpenFile(paths[0])
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	defer f.Close()

	// K5 應有 Model Qty (2)，L5 應有 Model Qty (5)
	qtyK5, _ := f.GetCellValue("SMD", "K5")
	qtyL5, _ := f.GetCellValue("SMD", "L5")
	if qtyK5 != "2" {
		t.Errorf("Expected K5 (Model A Qty) to be '2', got '%s'", qtyK5)
	}
	if qtyL5 != "5" {
		t.Errorf("Expected L5 (Model B Qty) to be '5', got '%s'", qtyL5)
	}

	// K6 應有 "V" 勾選，L6 應有 "V" 勾選
	valK6, _ := f.GetCellValue("SMD", "K6")
	valL6, _ := f.GetCellValue("SMD", "L6")
	if valK6 != "V" {
		t.Errorf("Expected K6 (Model A selection) to be 'V', got '%s'", valK6)
	}
	if valL6 != "V" {
		t.Errorf("Expected L6 (Model B selection) to be 'V', got '%s'", valL6)
	}
}

func TestValidateAndPrepareOutputPath_RemoveExisting(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "export_overwrite_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	targetPath := filepath.Join(tmpDir, "test_existing.xlsx")
	err = os.WriteFile(targetPath, []byte("old content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create dummy existing file: %v", err)
	}

	finalPath, err := validateAndPrepareOutputPath(nil, targetPath, "", "default.xlsx")
	if err != nil {
		t.Fatalf("validateAndPrepareOutputPath failed: %v", err)
	}
	if finalPath != targetPath {
		t.Errorf("Expected finalPath to be %s, got %s", targetPath, finalPath)
	}

	// 驗證原本的舊檔案是否已被刪除
	if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
		t.Errorf("Expected target file to be removed before writing, but it still exists")
	}
}

// TestExportMatrix_ModelQtyByOrder 驗證僅依據 ModelQtyByOrder 與 SelectionsByOrder 順序性也能正確輸出有多個 Model 的 Qty 與 V 勾選
func TestExportMatrix_ModelQtyByOrder(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bomix-modelbyorder-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writer, err := NewWriter(nil)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	options := ExportOptions{
		Format:     types.FormatMatrix,
		OutputPath: filepath.Join(tmpDir, "modelbyorder_test_matrix.xlsx"),
		Revisions: []RevisionData{
			{
				ProjectCode: "PROJ_ORDER",
				Phase:       "EVT",
				Version:     "0.1",
				ModelQtyByOrder: map[int]int{
					0: 3,
					1: 5,
					2: 7,
				},
			},
		},
		PartData: []PartData{
			{
				Item:        "1",
				HHPN:        "MAIN_PN_01",
				Description: "Resistor 100K",
				Supplier:    "YAGEO",
				SupplierPn:  "R100K",
				Qty:         2,
				Type:        "SMD",
				CCL:         true,
				SelectionsByOrder: map[int]string{
					0: "R100K",
					1: "R100K",
					2: "R100K",
				},
			},
		},
	}

	paths, err := writer.ExportExcel(options)
	if err != nil {
		t.Fatalf("ExportExcel failed: %v", err)
	}

	f, err := excelize.OpenFile(paths[0])
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	defer f.Close()

	// K5 應有 Qty 3, L5 應有 Qty 5, M5 應有 Qty 7
	qtyK5, _ := f.GetCellValue("SMD", "K5")
	qtyL5, _ := f.GetCellValue("SMD", "L5")
	qtyM5, _ := f.GetCellValue("SMD", "M5")
	if qtyK5 != "3" {
		t.Errorf("Expected K5 (Model 0 Qty) to be '3', got '%s'", qtyK5)
	}
	if qtyL5 != "5" {
		t.Errorf("Expected L5 (Model 1 Qty) to be '5', got '%s'", qtyL5)
	}
	if qtyM5 != "7" {
		t.Errorf("Expected M5 (Model 2 Qty) to be '7', got '%s'", qtyM5)
	}

	// K6, L6, M6 均應有 "V" 勾選
	valK6, _ := f.GetCellValue("SMD", "K6")
	valL6, _ := f.GetCellValue("SMD", "L6")
	valM6, _ := f.GetCellValue("SMD", "M6")
	if valK6 != "V" {
		t.Errorf("Expected K6 (Model 0 selection) to be 'V', got '%s'", valK6)
	}
	if valL6 != "V" {
		t.Errorf("Expected L6 (Model 1 selection) to be 'V', got '%s'", valL6)
	}
	if valM6 != "V" {
		t.Errorf("Expected M6 (Model 2 selection) to be 'V', got '%s'", valM6)
	}
}





