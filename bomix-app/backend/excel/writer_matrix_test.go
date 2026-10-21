package excel

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bomix-app/backend/logger"
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
		{
			name: "From EBOM SourceFile replacing _BOM_ and (compared)",
			rev: RevisionData{
				ProjectCode: "PROJ_M",
				Phase:       "EVT",
				Version:     "0.1",
				SourceFile:  "PROJ_M_EZBOM_EVT_0.1_BOM_20260807(compared).xlsx",
			},
			date:   "20260807",
			expect: "PROJ_M_EZBOM_EVT_0.1_MatrixBOM_20260807.xlsx",
		},
		{
			name: "From EBOM SourceFile with full path and spaces",
			rev: RevisionData{
				ProjectCode: "PROJ_ORDER",
				Phase:       "EVT",
				Version:     "0.1",
				SourceFile:  `C:\Path\To\PROJ_ORDER_EZBOM_EVT_0.1_BOM_20260807 (compared).xlsx`,
			},
			date:   "20260807",
			expect: "PROJ_ORDER_EZBOM_EVT_0.1_MatrixBOM_20260807.xlsx",
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
			Description: "Test Part 5 (Not CCL, no selections)",
		},
		{
			Supplier:    "AVX",
			SupplierPn:  "06035C104KAT2A",
			CCL:         false,
			BOMStatus:   "I",
			Type:        "SMD",
			Description: "Test Part 6 (Not CCL, has Main Selection)",
			SelectionsByOrder: map[int]string{0: "06035C104KAT2A"},
		},
		{
			Supplier:    "Nichicon",
			SupplierPn:  "UCL1V100MCL1GS",
			CCL:         false,
			BOMStatus:   "I",
			Type:        "SMD",
			Description: "Test Part 7 (Not CCL, has 2nd Source Selection)",
			SecondSources: []SecondSourceData{
				{
					Supplier:          "Panasonic",
					SupplierPn:        "EEE-FK1V100R",
					SelectionsByOrder: map[int]bool{1: true},
				},
			},
		},
		{
			Supplier:    "TDK",
			SupplierPn:  "C1005X7R1H102K",
			CCL:         false,
			BOMStatus:   "X",
			Type:        "SMD",
			Description: "Test Part 8 (Not CCL, has Selection but status X)",
			SelectionsByOrder: map[int]string{0: "C1005X7R1H102K"},
		},
		{
			Supplier:    "Bourns",
			SupplierPn:  "CR0603-FX-1001ELF",
			CCL:         false,
			BOMStatus:   "I",
			Type:        "PTH",
			Description: "Test Part 9 (Not CCL, revision-specific selection)",
			SelectionsByRevAndOrder: map[string]map[int]string{
				"101": {0: "CR0603-FX-1001ELF"},
			},
		},
	}

	// 1. 未指定 Revision 呼叫 filterMatrixParts(parts)
	// 預期通過項目：
	// - Part 1 (CCL=true, I)
	// - Part 2 (CCL=true, P)
	// - Part 3 (CCL=true, M)
	// - Part 6 (CCL=false, I, 有通用勾選)
	// - Part 7 (CCL=false, I, 替代料有勾選)
	// - Part 9 (CCL=false, I, 容錯模式未限定 revisionID 檢測通過)
	// 預期排除項目：
	// - Part 4 (BOMStatus=X)
	// - Part 5 (CCL=false, 無任何勾選)
	// - Part 8 (BOMStatus=X)
	filtered := filterMatrixParts(parts)
	if len(filtered) != 6 {
		t.Errorf("filterMatrixParts(parts): expected 6 parts, got %d", len(filtered))
	}
	for _, p := range filtered {
		if p.BOMStatus == "X" {
			t.Errorf("filterMatrixParts: part %s has invalid status X", p.SupplierPn)
		}
		if !p.CCL && !hasAnyPartSelection(p) {
			t.Errorf("filterMatrixParts: non-CCL part %s has no selection", p.SupplierPn)
		}
	}

	// 2. 指定 Revision "101" 呼叫 filterMatrixParts(parts, "101")
	// Part 9 在 "101" 有勾選，應被納入 (共 6 筆)
	filtered101 := filterMatrixParts(parts, "101")
	if len(filtered101) != 6 {
		t.Errorf("filterMatrixParts(parts, '101'): expected 6 parts, got %d", len(filtered101))
	}

	// 3. 指定非目標 Revision "999" 呼叫 filterMatrixParts(parts, "999")
	// Part 9 專屬於 "101"，在 "999" 無勾選，應被排除 (共 5 筆: Part 1, 2, 3, 6, 7)
	filtered999 := filterMatrixParts(parts, "999")
	if len(filtered999) != 5 {
		t.Errorf("filterMatrixParts(parts, '999'): expected 5 parts, got %d", len(filtered999))
	}
	for _, p := range filtered999 {
		if p.SupplierPn == "CR0603-FX-1001ELF" {
			t.Errorf("filterMatrixParts(parts, '999'): Part 9 should not be included in revision 999")
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

func TestEnsureMatrixSheets_MissingSMD(t *testing.T) {
	f := excelize.NewFile()
	// NewFile 預設包含 "Sheet1"，沒有 "SMD"
	w, err := NewWriter(nil)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	sheets, err := w.ensureMatrixSheets(f)
	if err == nil {
		t.Fatalf("Expected error when SMD sheet is missing, got nil")
	}
	if !strings.Contains(err.Error(), "missing required 'SMD' sheet") {
		t.Errorf("Expected 'missing required 'SMD' sheet' error, got: %v", err)
	}
	_ = sheets
}

func TestEnsureMatrixSheets_FallbackCopySMD(t *testing.T) {
	f := excelize.NewFile()
	f.NewSheet("SMD")
	f.DeleteSheet("Sheet1") // 只保留 SMD

	_ = f.SetCellValue("SMD", "A5", "Item")
	_ = f.SetCellValue("SMD", "A6", "TestItem")

	w, err := NewWriter(nil)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	sheets, err := w.ensureMatrixSheets(f)
	if err != nil {
		t.Fatalf("ensureMatrixSheets failed: %v", err)
	}

	expectedSheets := []string{"SMD", "PTH", "BOTTOM"}
	if len(sheets) != 3 {
		t.Fatalf("Expected 3 sheets, got %d", len(sheets))
	}
	for i, s := range expectedSheets {
		if sheets[i] != s {
			t.Errorf("Expected sheet %d to be '%s', got '%s'", i, s, sheets[i])
		}
	}

	// 驗證 PTH 與 BOTTOM 是否成功複製了 SMD 內容
	for _, sheetName := range []string{"PTH", "BOTTOM"} {
		idx, err := f.GetSheetIndex(sheetName)
		if err != nil || idx < 0 {
			t.Errorf("Expected sheet '%s' to exist", sheetName)
		}
		val, _ := f.GetCellValue(sheetName, "A6")
		if val != "TestItem" {
			t.Errorf("Expected sheet '%s' A6 to inherit 'TestItem' from SMD, got '%s'", sheetName, val)
		}
	}
}

// TestExportMatrix_ProtoGroupRowTextColor 測試當物料群組 BOMStatus == "P" 時，
// Matrix 匯出中主料與 2nd source 整個 row 的文字顏色是否被正確設定為 #8080C0
func TestExportMatrix_ProtoGroupRowTextColor(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bomix-matrix-proto-test-*")
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
		OutputPath: filepath.Join(tmpDir, "proto_matrix_test.xlsx"),
		PartData: []PartData{
			{
				Item:        "1",
				HHPN:        "PROTO_SMD_PN",
				Description: "Proto SMD Resistor",
				Supplier:    "YAGEO",
				SupplierPn:  "PROTO_R10K",
				Qty:         2,
				Location:    "C1,C2",
				Type:        "SMD",
				BOMStatus:   "P", // PROTO 物料群組
				CCL:         true,
				Remark:      "Proto Remark",
				SecondSources: []SecondSourceData{
					{
						HHPN:        "PROTO_SMD_PN_SS",
						Supplier:    "WALSIN",
						SupplierPn:  "PROTO_R10K_SS",
						Description: "Proto SMD Resistor 2nd Source",
						Remark:      "Proto SS Remark",
					},
				},
				SelectionsByOrder: map[int]string{0: "PROTO_R10K"},
			},
			{
				Item:        "2",
				HHPN:        "NORMAL_SMD_PN",
				Description: "Normal SMD Resistor",
				Supplier:    "YAGEO",
				SupplierPn:  "NORMAL_R10K",
				Qty:         1,
				Location:    "C3",
				Type:        "SMD",
				BOMStatus:   "I", // 一般物料群組
				CCL:         true,
			},
		},
		Revisions: []RevisionData{
			{
				ProjectCode:     "PROJ_MATRIX_PROTO",
				Phase:           "EVT",
				Version:         "0.1",
				ModelNames:      []string{"Model A"},
				ModelQtyByOrder: map[int]int{0: 1},
			},
		},
	}

	paths, err := writer.ExportExcel(options)
	if err != nil {
		t.Fatalf("ExportExcel failed: %v", err)
	}

	f, err := excelize.OpenFile(paths[0])
	if err != nil {
		t.Fatalf("Failed to open exported file: %v", err)
	}
	defer f.Close()

	// Sheet "SMD"
	// Row 6: PROTO 主料
	// Row 7: PROTO 2nd source
	// Row 8: 一般物料 (BOMStatus = "I")
	rowsToCheck := []struct {
		row         int
		col         string
		expectProto bool
		description string
	}{
		{6, "A", true, "PROTO 主料 A 欄"},
		{6, "B", true, "PROTO 主料 B 欄 (HHPN)"},
		{6, "K", true, "PROTO 主料 K 欄 (Model Selection)"},
		{6, "R", true, "PROTO 主料 R 欄 (Remark)"},
		{7, "B", true, "PROTO 2nd Source B 欄 (HHPN)"},
		{7, "K", true, "PROTO 2nd Source K 欄 (Model Selection)"},
		{7, "R", true, "PROTO 2nd Source R 欄 (Remark)"},
		{8, "B", false, "一般物料 B 欄 (HHPN)"},
	}

	for _, tc := range rowsToCheck {
		cellAddr := fmt.Sprintf("%s%d", tc.col, tc.row)
		styleID, err := f.GetCellStyle("SMD", cellAddr)
		if err != nil {
			t.Fatalf("Failed to get style for %s: %v", cellAddr, err)
		}
		styleDef, err := f.GetStyle(styleID)
		if err != nil || styleDef == nil {
			t.Fatalf("Failed to inspect style for %s: %v", cellAddr, err)
		}

		if tc.expectProto {
			if styleDef.Font == nil {
				t.Errorf("[%s] Expect font color #8080C0, but Font is nil", tc.description)
			} else if !strings.EqualFold(styleDef.Font.Color, "#8080C0") && !strings.EqualFold(styleDef.Font.Color, "8080C0") {
				t.Errorf("[%s] Expect font color #8080C0, got %q", tc.description, styleDef.Font.Color)
			}
		} else {
			if styleDef.Font != nil && (strings.EqualFold(styleDef.Font.Color, "#8080C0") || strings.EqualFold(styleDef.Font.Color, "8080C0")) {
				t.Errorf("[%s] Unexpected PROTO font color #8080C0 for non-proto row, got %q", tc.description, styleDef.Font.Color)
			}
		}
	}
}

// TestExportMatrix_OverwriteExistingFileWithWarning 驗證當 Matrix 匯出目標檔案已存在時，
// validateAndPrepareOutputPath 能正常刪除舊檔並成功覆蓋寫入新檔
func TestExportMatrix_OverwriteExistingFileWithWarning(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bomix-matrix-overwrite-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	targetPath := filepath.Join(tmpDir, "PROJ_OVERWRITE_EZBOM_EVT_0.1_MatrixBOM_20260810.xlsx")
	// 預先寫入一舊舊檔內容
	if err := os.WriteFile(targetPath, []byte("old dummy file content"), 0644); err != nil {
		t.Fatalf("Failed to write dummy existing file: %v", err)
	}

	lg := logger.NewLogger(100)
	writer, err := NewWriter(lg)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	options := ExportOptions{
		Format:     types.FormatMatrix,
		OutputPath: targetPath,
		PartData: []PartData{
			{
				Item:        "1",
				HHPN:        "OVERWRITE_PN",
				Description: "Overwrite Test Part",
				Supplier:    "YAGEO",
				SupplierPn:  "R10K",
				Qty:         1,
				Location:    "C1",
				Type:        "SMD",
				BOMStatus:   "I",
				CCL:         true,
			},
		},
		Revisions: []RevisionData{
			{
				ProjectCode:     "PROJ_OVERWRITE",
				Phase:           "EVT",
				Version:         "0.1",
				ModelNames:      []string{"Model A"},
				ModelQtyByOrder: map[int]int{0: 1},
			},
		},
	}

	paths, err := writer.ExportExcel(options)
	if err != nil {
		t.Fatalf("ExportExcel failed when target file exists: %v", err)
	}

	if len(paths) == 0 || paths[0] != targetPath {
		t.Fatalf("Expected output path %s, got %v", targetPath, paths)
	}

	f, err := excelize.OpenFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to open overwritten Excel file: %v", err)
	}
	_ = f.Close()
}

// TestExportMatrix_GroupMasterQtyFormula 驗證 Matrix 匯出時，物料群組內主料與所有替代料的 I 欄公式
// 均固定引用該群組主料列的 G 欄 Qty（例如：Row 6 主料 I6="=G6*J6", Row 7 替代料 I7="=G6*J7", Row 8 替代料 I8="=G6*J8"）。
func TestExportMatrix_GroupMasterQtyFormula(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bomix-groupqty-test-*")
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
		OutputPath: filepath.Join(tmpDir, "group_qty_test_matrix.xlsx"),
		PartData: []PartData{
			{
				Item:        "1",
				HHPN:        "MAIN_PN_01",
				Description: "Resistor 100K",
				Supplier:    "YAGEO",
				SupplierPn:  "R100K",
				Qty:         4,
				Type:        "SMD",
				CCL:         true,
				SecondSources: []SecondSourceData{
					{
						HHPN:        "ALT_PN_01",
						Supplier:    "UNI-ROYAL",
						SupplierPn:  "R100K_ALT1",
						Description: "Resistor 100K Alt 1",
					},
					{
						HHPN:        "ALT_PN_02",
						Supplier:    "WALSIN",
						SupplierPn:  "R100K_ALT2",
						Description: "Resistor 100K Alt 2",
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
		t.Fatalf("Failed to open exported file: %v", err)
	}
	defer f.Close()

	// 主料在 Row 6，替代料分別在 Row 7 與 Row 8
	formulaI6, err6 := f.GetCellFormula("SMD", "I6")
	formulaI7, err7 := f.GetCellFormula("SMD", "I7")
	formulaI8, err8 := f.GetCellFormula("SMD", "I8")

	if err6 != nil || err7 != nil || err8 != nil {
		t.Fatalf("Failed to read formulas from I6/I7/I8: %v, %v, %v", err6, err7, err8)
	}

	expectedI6 := "=G6*J6"
	expectedI7 := "=G6*J7"
	expectedI8 := "=G6*J8"

	if formulaI6 != expectedI6 {
		t.Errorf("I6 formula mismatch: expected %s, got %s", expectedI6, formulaI6)
	}
	if formulaI7 != expectedI7 {
		t.Errorf("I7 formula mismatch: expected %s, got %s", expectedI7, formulaI7)
	}
	if formulaI8 != expectedI8 {
		t.Errorf("I8 formula mismatch: expected %s, got %s", expectedI8, formulaI8)
	}
}

// TestExportMatrix_NonCCLWithSelections 測試 Matrix 匯出時，非 CCL 但有勾選之物料能夠正確匯出至對應 Sheet，
// 未勾選之非 CCL 物料被排除，且 Model 勾選記號 "V" 與公式皆正確生成。
func TestExportMatrix_NonCCLWithSelections(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bomix-matrix-nonccl-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writer, err := NewWriter(nil)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	targetPath := filepath.Join(tmpDir, "matrix_non_ccl.xlsx")
	options := ExportOptions{
		Format:      types.FormatMatrix,
		OutputPath:  targetPath,
		Description: "Test Matrix Non-CCL Export",
		Revisions: []RevisionData{
			{
				ID:              "101",
				ProjectCode:     "TESTPROJ",
				Phase:           "EVT",
				Version:         "0.1",
				ModelNames:      []string{"Model A", "Model B"},
				ModelQtyByOrder: map[int]int{0: 5, 1: 10},
			},
		},
		PartData: []PartData{
			// Part 1: CCL 物料 (SMD) -> 應匯出至 SMD
			{
				Item:        "1",
				HHPN:        "111-CCL",
				Description: "CCL Part SMD",
				Supplier:    "Samsung",
				SupplierPn:  "CCL-SMD-001",
				Qty:         2,
				Location:    "C1,C2",
				Type:        "SMD",
				BOMStatus:   "I",
				CCL:         true,
			},
			// Part 2: 非 CCL 物料 (SMD)，在 Model 0 有勾選 -> 應匯出至 SMD 且 Model 0 有 "V"
			{
				Item:        "2",
				HHPN:        "222-NONCCL",
				Description: "Non-CCL Selected Part SMD",
				Supplier:    "Murata",
				SupplierPn:  "NONCCL-SMD-002",
				Qty:         4,
				Location:    "R1,R2,R3,R4",
				Type:        "SMD",
				BOMStatus:   "I",
				CCL:         false,
				SelectionsByOrder: map[int]string{
					0: "NONCCL-SMD-002",
				},
			},
			// Part 3: 非 CCL 物料 (SMD)，無任何勾選 -> 應被排除
			{
				Item:        "3",
				HHPN:        "333-NONCCL-NOSEL",
				Description: "Non-CCL Unselected Part",
				Supplier:    "Yageo",
				SupplierPn:  "NONCCL-SMD-UNSEL",
				Qty:         1,
				Location:    "R5",
				Type:        "SMD",
				BOMStatus:   "I",
				CCL:         false,
			},
			// Part 4: 非 CCL 物料 (PTH)，替代料在 Model 1 有勾選 -> 應匯出至 PTH，群組主料與替代料均存在，且替代料在 Model 1 有 "V"
			{
				Item:        "4",
				HHPN:        "444-NONCCL-PTH",
				Description: "Non-CCL PTH Group",
				Supplier:    "Molex",
				SupplierPn:  "NONCCL-PTH-MAIN",
				Qty:         1,
				Location:    "J1",
				Type:        "PTH",
				BOMStatus:   "I",
				CCL:         false,
				SecondSources: []SecondSourceData{
					{
						HHPN:              "444-NONCCL-PTH-2ND",
						Description:       "Non-CCL PTH Second Source",
						Supplier:          "Amphenol",
						SupplierPn:        "NONCCL-PTH-2ND",
						SelectionsByOrder: map[int]bool{1: true},
					},
				},
			},
		},
	}

	paths, err := writer.ExportExcel(options)
	if err != nil {
		t.Fatalf("ExportExcel failed: %v", err)
	}

	if len(paths) == 0 {
		t.Fatalf("No output path returned")
	}

	f, err := excelize.OpenFile(paths[0])
	if err != nil {
		t.Fatalf("Failed to open exported file: %v", err)
	}
	defer f.Close()

	// 1. 驗證 SMD Sheet
	// Row 6 應為 CCL-SMD-001
	pnRow6, _ := f.GetCellValue("SMD", "F6")
	if pnRow6 != "CCL-SMD-001" {
		t.Errorf("SMD Row 6 SupplierPn mismatch: expected CCL-SMD-001, got %q", pnRow6)
	}

	// Row 7 應為 NONCCL-SMD-002 (非 CCL 但有勾選，成功匯出)
	pnRow7, _ := f.GetCellValue("SMD", "F7")
	if pnRow7 != "NONCCL-SMD-002" {
		t.Errorf("SMD Row 7 SupplierPn mismatch: expected NONCCL-SMD-002, got %q", pnRow7)
	}
	// Row 7 的 Model 0 (K7) 應有 "V"
	vK7, _ := f.GetCellValue("SMD", "K7")
	if vK7 != "V" {
		t.Errorf("SMD K7 selection mismatch: expected 'V', got %q", vK7)
	}

	// Row 8 應為空白 (未勾選之 Part 3 不得匯出)
	pnRow8, _ := f.GetCellValue("SMD", "F8")
	if pnRow8 != "" {
		t.Errorf("SMD Row 8 should be empty, but got %q", pnRow8)
	}

	// 2. 驗證 PTH Sheet
	// Row 6 應為主料 NONCCL-PTH-MAIN
	pthRow6, _ := f.GetCellValue("PTH", "F6")
	if pthRow6 != "NONCCL-PTH-MAIN" {
		t.Errorf("PTH Row 6 SupplierPn mismatch: expected NONCCL-PTH-MAIN, got %q", pthRow6)
	}
	// Row 7 應為替代料 NONCCL-PTH-2ND
	pthRow7, _ := f.GetCellValue("PTH", "F7")
	if pthRow7 != "NONCCL-PTH-2ND" {
		t.Errorf("PTH Row 7 SupplierPn mismatch: expected NONCCL-PTH-2ND, got %q", pthRow7)
	}
	// Model 1 (L 欄) 勾選應在替代料列 (L7="V")，主料列不應勾選 (L6="")
	vL6, _ := f.GetCellValue("PTH", "L6")
	vL7, _ := f.GetCellValue("PTH", "L7")
	if vL6 != "" {
		t.Errorf("PTH L6 should be empty, got %q", vL6)
	}
	if vL7 != "V" {
		t.Errorf("PTH L7 selection mismatch: expected 'V', got %q", vL7)
	}
}

