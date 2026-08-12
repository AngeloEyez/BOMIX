package excel

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bomix-app/backend/logger"
	"bomix-app/backend/types"
	"github.com/xuri/excelize/v2"
)

func TestApplyTagReplacement(t *testing.T) {
	// Create a temporary test file
	f := excelize.NewFile()
	wb := &ExcelizeWorkbook{f: f}
	_ = wb
	defer f.Close()

	// Set up test data
	f.SetCellValue("Sheet1", "A1", "{{.Title}}")
	f.SetCellValue("Sheet1", "A2", "{{.Author}}")
	f.SetCellValue("Sheet1", "A3", "Static Text")
	f.SetCellValue("Sheet1", "B1", "{{.Title}}")

	tags := map[string]string{
		"{{.Title}}":  "Test Document",
		"{{.Author}}": "John Doe",
	}

	err := applyTagReplacement(f, tags)
	if err != nil {
		t.Fatalf("applyTagReplacement failed: %v", err)
	}

	// Verify replacements
	title, err := f.GetCellValue("Sheet1", "A1")
	if err != nil {
		t.Fatalf("GetCellValue failed: %v", err)
	}
	if title != "Test Document" {
		t.Errorf("Expected 'Test Document', got '%s'", title)
	}

	author, err := f.GetCellValue("Sheet1", "A2")
	if err != nil {
		t.Fatalf("GetCellValue failed: %v", err)
	}
	if author != "John Doe" {
		t.Errorf("Expected 'John Doe', got '%s'", author)
	}

	// Verify static text unchanged
	static, err := f.GetCellValue("Sheet1", "A3")
	if err != nil {
		t.Fatalf("GetCellValue failed: %v", err)
	}
	if static != "Static Text" {
		t.Errorf("Expected 'Static Text', got '%s'", static)
	}

	// Verify tag replaced in multiple cells
	titleB1, err := f.GetCellValue("Sheet1", "B1")
	if err != nil {
		t.Fatalf("GetCellValue failed: %v", err)
	}
	if titleB1 != "Test Document" {
		t.Errorf("Expected 'Test Document' in B1, got '%s'", titleB1)
	}
}

func TestGetColName(t *testing.T) {
	tests := []struct {
		index  int
		expect string
	}{
		{0, "A"},
		{1, "B"},
		{2, "C"},
		{7, "H"},
		{9, "J"},
		{10, "K"},
		{25, "Z"},
		{26, "AA"},
		{27, "AB"},
	}

	for _, tt := range tests {
		result := getColName(tt.index)
		if result != tt.expect {
			t.Errorf("getColName(%d) = %s, want %s", tt.index, result, tt.expect)
		}
	}
}

func TestColNameToIndex(t *testing.T) {
	tests := []struct {
		colName string
		expect  int
	}{
		{"A", 0},
		{"B", 1},
		{"C", 2},
		{"H", 7},
		{"J", 9},
		{"K", 10},
		{"Z", 25},
		{"AA", 26},
		{"AB", 27},
	}

	for _, tt := range tests {
		result := colNameToIndex(tt.colName)
		if result != tt.expect {
			t.Errorf("colNameToIndex(%s) = %d, want %d", tt.colName, result, tt.expect)
		}
	}
}

func TestGenerateBigMatrixFileName(t *testing.T) {
	tests := []struct {
		name       string
		seriesName string
		revisions  []RevisionData
		date       string
		expect     string
	}{
		{
			name:       "All same phase and version",
			seriesName: "FY27",
			revisions: []RevisionData{
				{Phase: "PV", Version: "0.3"},
				{Phase: "PV", Version: "0.3"},
			},
			date:   "20260717",
			expect: "FY27_BigMatrix_PV_0.3_20260717.xlsx",
		},
		{
			name:       "Same phase, different versions",
			seriesName: "FY27",
			revisions: []RevisionData{
				{Phase: "PV", Version: "0.3"},
				{Phase: "PV", Version: "0.4"},
			},
			date:   "20260717",
			expect: "FY27_BigMatrix_PV_20260717.xlsx",
		},
		{
			name:       "Different phases",
			seriesName: "FY27",
			revisions: []RevisionData{
				{Phase: "PV", Version: "0.3"},
				{Phase: "EV", Version: "0.1"},
			},
			date:   "20260717",
			expect: "FY27_BigMatrix_20260717.xlsx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateBigMatrixFileName(tt.seriesName, tt.revisions, tt.date)
			if result != tt.expect {
				t.Errorf("generateBigMatrixFileName() = %s, want %s", result, tt.expect)
			}
		})
	}
}

func TestFilterPartsByCriteria(t *testing.T) {
	parts := []PartData{
		{
			Supplier:    "Samsung",
			SupplierPn:  "CL05B104KO5NNNC",
			CCL:         true,
			BOMStatus:   "I",
			Description: "Test Part 1",
		},
		{
			Supplier:    "Murata",
			SupplierPn:  "GRM155B81C105KE19D",
			CCL:         true,
			BOMStatus:   "P",
			Description: "Test Part 2 (NPI)",
		},
		{
			Supplier:    "Taiyo Yuden",
			SupplierPn:  "UMK105B7105KV-F",
			CCL:         true,
			BOMStatus:   "M",
			Description: "Test Part 3 (MP)",
		},
		{
			Supplier:    "Yageo",
			SupplierPn:  "CC0402KRX7R9BB104",
			CCL:         true,
			BOMStatus:   "X",
			Description: "Test Part 4 (Not Installed)",
		},
		{
			Supplier:    "Kemet",
			SupplierPn:  "C0603C104K5RACTU",
			CCL:         false,
			BOMStatus:   "I",
			Description: "Test Part 5 (Not CCL)",
		},
	}

	// Test FilterPartsByCriteria
	filtered := FilterPartsByCriteria(parts)
	// CCL=true, status!=X: Part 1(I), Part 2(P), Part 3(M) => 3 parts
	if len(filtered) != 3 {
		t.Errorf("FilterPartsByCriteria: expected 3 parts, got %d", len(filtered))
	}
	for _, p := range filtered {
		if p.BOMStatus == "X" {
			t.Errorf("FilterPartsByCriteria: part %s has invalid status X", p.SupplierPn)
		}
		if !p.CCL {
			t.Errorf("FilterPartsByCriteria: part %s is not CCL", p.SupplierPn)
		}
	}
}

func TestDeduplicateParts(t *testing.T) {
	parts := []PartData{
		{
			Supplier:    "Samsung",
			SupplierPn:  "CL05B104KO5NNNC",
			Description: "Capacitor 10uF",
			SecondSources: []SecondSourceData{
				{Supplier: "Murata", SupplierPn: "GRM155B81C105KE19D"},
			},
		},
		{
			Supplier:    "Samsung",
			SupplierPn:  "CL05B104KO5NNNC",
			Description: "Capacitor 10uF (Duplicate)",
			SecondSources: []SecondSourceData{
				{Supplier: "Taiyo Yuden", SupplierPn: "UMK105B7105KV-F"},
			},
		},
		{
			Supplier:    "Murata",
			SupplierPn:  "GRM155B71H104KA12D",
			Description: "Capacitor 100nF",
		},
	}

	result := deduplicateParts(parts)
	if len(result) != 2 {
		t.Errorf("deduplicateParts: expected 2 parts, got %d", len(result))
	}
}

func TestMergeSecondSources(t *testing.T) {
	parts := []PartData{
		{
			Supplier:    "Samsung",
			SupplierPn:  "CL05B104KO5NNNC",
			Description: "Capacitor 10uF",
			SecondSources: []SecondSourceData{
				{Supplier: "Murata", SupplierPn: "GRM155B81C105KE19D"},
				{Supplier: "Taiyo Yuden", SupplierPn: "UMK105B7105KV-F"},
			},
		},
		{
			Supplier:    "Samsung",
			SupplierPn:  "CL05B104KO5NNNC",
			Description: "Capacitor 10uF (Same Group)",
			SecondSources: []SecondSourceData{
				{Supplier: "Yageo", SupplierPn: "CC0402KRX7R9BB104"},
				{Supplier: "Murata", SupplierPn: "GRM155B81C105KE19D"}, // Duplicate
			},
		},
	}

	result := mergeSecondSources(parts)
	if len(result) != 1 {
		t.Errorf("mergeSecondSources: expected 1 part, got %d", len(result))
	}

	if len(result[0].SecondSources) != 3 {
		t.Errorf("mergeSecondSources: expected 3 second sources, got %d", len(result[0].SecondSources))
	}
}

func TestNormalizeLocationCount(t *testing.T) {
	tests := []struct {
		loc      string
		expected int
	}{
		{"", 0},
		{"R1", 1},
		{"R1,C1,C2", 3},
		{"C1,C2,C3,C4,C5,C6,C7,C8", 8},
	}

	for _, tt := range tests {
		result := NormalizeLocationCount(tt.loc)
		if result != tt.expected {
			t.Errorf("NormalizeLocationCount(%q) = %d, want %d", tt.loc, result, tt.expected)
		}
	}
}

func TestExportBigMatrix_Integration(t *testing.T) {
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
		Format:    types.FormatBigMatrix,
		OutputPath: filepath.Join(tmpDir, "test_bigmatrix.xlsx"),
		Description: "Test BigMatrix Export",
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
				SecondSources: []SecondSourceData{
					{
						HHPN:      "34065Y600-GRT-H",
						Supplier:  "Murata",
						SupplierPn: "GRM155B81C105KE19D",
						Description: "CAP,10uF,+/-20%,X5R,6.3V,SMD0603",
					},
				},
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

	// Verify sheet exists
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		t.Fatal("Exported file has no sheets")
	}

	// Verify data was written
	val, err := f.GetCellValue("BigMatrix", "A6")
	if err != nil {
		t.Fatalf("Failed to read cell A6: %v", err)
	}
	// A6 should be "Item" header or first item data
	if !strings.Contains(val, "Item") && val != "1" {
		t.Errorf("Unexpected value in A6: %s", val)
	}

	t.Logf("Successfully exported BigMatrix to: %s", outputPath)
}

func TestResolveOutputPath(t *testing.T) {
	// Create temporary folder for testing directory check
	tmpDir, err := os.MkdirTemp("", "bomix-resolve-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	defaultName := "Default_BigMatrix.xlsx"

	tests := []struct {
		name        string
		outputPath  string
		outputDir   string
		expectMatch func(res string) bool
	}{
		{
			name:       "Output path is an existing directory",
			outputPath: tmpDir,
			outputDir:  "",
			expectMatch: func(res string) bool {
				return res == filepath.Join(tmpDir, defaultName)
			},
		},
		{
			name:       "Output path has no extension and is not dir",
			outputPath: filepath.Join(tmpDir, "SubFolderWithoutExt"),
			outputDir:  "",
			expectMatch: func(res string) bool {
				return res == filepath.Join(tmpDir, "SubFolderWithoutExt", defaultName)
			},
		},
		{
			name:       "Output path is a valid xlsx file path",
			outputPath: filepath.Join(tmpDir, "custom.xlsx"),
			outputDir:  "",
			expectMatch: func(res string) bool {
				return res == filepath.Join(tmpDir, "custom.xlsx")
			},
		},
		{
			name:       "Output path empty, outputDir provided",
			outputPath: "",
			outputDir:  tmpDir,
			expectMatch: func(res string) bool {
				return res == filepath.Join(tmpDir, defaultName)
			},
		},
		{
			name:       "Both empty",
			outputPath: "",
			outputDir:  "",
			expectMatch: func(res string) bool {
				return res == defaultName
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := resolveOutputPath(tt.outputPath, tt.outputDir, defaultName)
			if !tt.expectMatch(res) {
				t.Errorf("resolveOutputPath() = %s, failed expectation", res)
			}
		})
	}
}

func TestExportBigMatrix_DirectoryOutputPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bomix-export-dir-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writer, err := NewWriter(nil)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	// Supply OutputPath as a directory instead of a full file path
	options := ExportOptions{
		Format:     types.FormatBigMatrix,
		OutputPath: tmpDir,
		PartData:   []PartData{},
	}

	outputPaths, err := writer.ExportExcel(options)
	if err != nil {
		t.Fatalf("ExportExcel with directory OutputPath should succeed, but got error: %v", err)
	}

	if len(outputPaths) != 1 {
		t.Fatalf("Expected 1 output path, got %d", len(outputPaths))
	}

	if filepath.Dir(outputPaths[0]) != tmpDir {
		t.Errorf("Expected output file in %s, got %s", tmpDir, outputPaths[0])
	}
}

func TestExportBigMatrix_InvalidOutputPath(t *testing.T) {
	writer, err := NewWriter(nil)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	// Supply an invalid path that cannot be created or written to
	invalidPath := `X:\NonExistentDriveDirectory1234567\output.xlsx`
	options := ExportOptions{
		Format:     types.FormatBigMatrix,
		OutputPath: invalidPath,
		PartData:   []PartData{},
	}

	_, err = writer.ExportExcel(options)
	if err == nil {
		t.Fatalf("Expected ExportExcel to fail for invalid path, but it succeeded")
	}

	if !strings.Contains(err.Error(), "invalid export output path") && !errors.Is(err, ErrInvalidOutputPath) {
		t.Errorf("Expected error to contain ErrInvalidOutputPath, got: %v", err)
	}
}

func TestExportBigMatrix_TagReplacementAndPartData(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bigmatrix_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writer, err := NewWriter(nil)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	options := ExportOptions{
		Format:      types.FormatBigMatrix,
		OutputPath:  filepath.Join(tmpDir, "test_output.xlsx"),
		Description: "Test BigMatrix Project",
		Revisions: []RevisionData{
			{
				ID:          "1",
				ProjectCode: "DEMO_PROJ",
				Phase:       "PV",
				Version:     "0.3",
				ModelQty:    map[string]int{"A": 2, "B": 1},
			},
		},
		PartData: []PartData{
			{
				Item:        "1",
				HHPN:        "123456789",
				Description: "Resistor 10K",
				Supplier:    "YAGEO",
				SupplierPn:  "RC0603FR-0710KL",
				Qty:         5,
				Location:    "R1,R2,R3,R4,R5",
				Selections: map[string]string{
					"A": "RC0603FR-0710KL",
				},
			},
		},
	}

	paths, err := writer.ExportExcel(options)
	if err != nil {
		t.Fatalf("ExportExcel failed: %v", err)
	}

	if len(paths) != 1 {
		t.Fatalf("Expected 1 path, got %d", len(paths))
	}

	// Verify generated file
	f, err := excelize.OpenFile(paths[0])
	if err != nil {
		t.Fatalf("Failed to open exported file: %v", err)
	}
	defer f.Close()

	// Verify Description tag replacement in B4
	desc, _ := f.GetCellValue("BigMatrix", "B4")
	if !strings.Contains(desc, "Test BigMatrix Project") {
		t.Errorf("Expected B4 to contain 'Test BigMatrix Project', got '%s'", desc)
	}

	// Verify H2 (Project Code)
	proj, _ := f.GetCellValue("BigMatrix", "H2")
	if proj != "DEMO_PROJ" {
		t.Errorf("Expected H2 to be 'DEMO_PROJ', got '%s'", proj)
	}

	// Verify Part Data at Row 6
	item, _ := f.GetCellValue("BigMatrix", "A6")
	if item != "1" {
		t.Errorf("Expected A6 (Item) to be '1', got '%s'", item)
	}

	hhpn, _ := f.GetCellValue("BigMatrix", "B6")
	if hhpn != "123456789" {
		t.Errorf("Expected B6 (HHPN) to be '123456789', got '%s'", hhpn)
	}

	// Verify Selection V at H6 (Model A)
	selA, _ := f.GetCellValue("BigMatrix", "H6")
	if selA != "V" {
		t.Errorf("Expected H6 (Selection A) to be 'V', got '%s'", selA)
	}
}

// TestExportBigMatrix_MultipleRevisionsHeader tests multi-project header export and phase-version formatting
func TestExportBigMatrix_MultipleRevisionsHeader(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bigmatrix_multi_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writer, err := NewWriter(nil)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	options := ExportOptions{
		Format:      types.FormatBigMatrix,
		OutputPath:  filepath.Join(tmpDir, "multi_output.xlsx"),
		Description: "Multi Revision Test",
		Revisions: []RevisionData{
			{
				ID:          "1",
				ProjectCode: "PROJ_ONE",
				Phase:       "PV",
				Version:     "0.3",
				ModelQty:    map[string]int{"A": 1, "B": 1},
			},
			{
				ID:          "2",
				ProjectCode: "PROJ_TWO",
				Phase:       "", // Empty Phase
				Version:     "0.5",
				ModelQty:    map[string]int{"A": 1, "B": 1},
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

	// 1. Verify first project header at H (column 7)
	proj1, _ := f.GetCellValue("BigMatrix", "H2")
	if proj1 != "PROJ_ONE" {
		t.Errorf("Expected H2 to be 'PROJ_ONE', got '%s'", proj1)
	}

	h3, _ := f.GetCellValue("BigMatrix", "H3")
	if h3 != "PV-0.3" {
		t.Errorf("Expected H3 to be 'PV-0.3', got '%s'", h3)
	}

	// 2. Verify second project header at J (column 7 + 2 = 9, i.e., J)
	proj2, _ := f.GetCellValue("BigMatrix", "J2")
	if proj2 != "PROJ_TWO" {
		t.Errorf("Expected J2 to be 'PROJ_TWO', got '%s'", proj2)
	}

	j3, _ := f.GetCellValue("BigMatrix", "J3")
	if j3 != "0.5" {
		t.Errorf("Expected J3 to be '0.5' (no leading dash), got '%s'", j3)
	}
}

// TestExportBigMatrix_StyleInheritance verifies H, I, J archetype style inheritance across multiple BOMs
func TestExportBigMatrix_StyleInheritance(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bigmatrix_style_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writer, err := NewWriter(nil)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	options := ExportOptions{
		Format:      types.FormatBigMatrix,
		OutputPath:  filepath.Join(tmpDir, "style_test.xlsx"),
		Description: "Style Test",
		Revisions: []RevisionData{
			{
				ID:          "1",
				ProjectCode: "BOM_1",
				Phase:       "PV",
				Version:     "0.1",
				ModelQty:    map[string]int{"A": 1, "B": 1, "C": 1}, // H, I, J
			},
			{
				ID:          "2",
				ProjectCode: "BOM_2",
				Phase:       "PV",
				Version:     "0.2",
				ModelQty:    map[string]int{"A": 1, "B": 1, "C": 1}, // K, L, M
			},
		},
		PartData: []PartData{
			{
				Item:        "1",
				HHPN:        "PN1",
				Description: "Resistor",
				Supplier:    "YAGEO",
				SupplierPn:  "RC1",
				Qty:         1,
				Location:    "R1",
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

	// Verify header cell style inheritance on second BOM (K, L, M should inherit from H, I, J)
	styleH4, _ := f.GetCellStyle("BigMatrix", "H4")
	styleK4, _ := f.GetCellStyle("BigMatrix", "K4")
	if styleK4 != styleH4 {
		t.Errorf("Expected K4 (Start) style to match H4 style (%d), got %d", styleH4, styleK4)
	}

	styleI4, _ := f.GetCellStyle("BigMatrix", "I4")
	styleL4, _ := f.GetCellStyle("BigMatrix", "L4")
	if styleL4 != styleI4 {
		t.Errorf("Expected L4 (Inner) style to match I4 style (%d), got %d", styleI4, styleL4)
	}

	styleJ4, _ := f.GetCellStyle("BigMatrix", "J4")
	styleM4, _ := f.GetCellStyle("BigMatrix", "M4")
	if styleM4 != styleJ4 {
		t.Errorf("Expected M4 (End) style to match J4 style (%d), got %d", styleJ4, styleM4)
	}

	// Verify Row 6 data cell style inheritance on second BOM
	styleH6, _ := f.GetCellStyle("BigMatrix", "H6")
	styleK6, _ := f.GetCellStyle("BigMatrix", "K6")
	if styleK6 != styleH6 {
		t.Errorf("Expected K6 (Data Start) style to match H6 style (%d), got %d", styleH6, styleK6)
	}

	styleJ6, _ := f.GetCellStyle("BigMatrix", "J6")
	styleM6, _ := f.GetCellStyle("BigMatrix", "M6")
	if styleM6 != styleJ6 {
		t.Errorf("Expected M6 (Data End) style to match J6 style (%d), got %d", styleJ6, styleM6)
	}
}

// TestExportBigMatrix_AutoColWidth verifies automatic column width calculation based on header text
func TestExportBigMatrix_AutoColWidth(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bigmatrix_width_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writer, err := NewWriter(nil)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	longProjCode := "VERY_VERY_LONG_PROJECT_CODE_NAME" // 32 chars
	options := ExportOptions{
		Format:      types.FormatBigMatrix,
		OutputPath:  filepath.Join(tmpDir, "width_test.xlsx"),
		Description: "Width Test",
		Revisions: []RevisionData{
			{
				ID:          "1",
				ProjectCode: longProjCode,
				Phase:       "PV",
				Version:     "0.1",
				ModelQty:    map[string]int{"A": 1}, // Single column, so width req is (32 + 2) / 1 = 34
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

	w, err := f.GetColWidth("BigMatrix", "H")
	if err != nil {
		t.Fatalf("GetColWidth failed: %v", err)
	}

	if w < 30.0 {
		t.Errorf("Expected H column width to be >= 30.0 for long project code, got %f", w)
	}
}

// TestExportBigMatrix_EmptyQtyAndSelection verifies that empty ModelQty and empty Selections remain empty
func TestExportBigMatrix_EmptyQtyAndSelection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bigmatrix_empty_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writer, err := NewWriter(nil)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	options := ExportOptions{
		Format:      types.FormatBigMatrix,
		OutputPath:  filepath.Join(tmpDir, "empty_test.xlsx"),
		Description: "Empty Test",
		Revisions: []RevisionData{
			{
				ID:          "1",
				ProjectCode: "NO_QTY_PROJ",
				Phase:       "PV",
				Version:     "0.1",
				ModelQty:    map[string]int{}, // Empty ModelQty
			},
		},
		PartData: []PartData{
			{
				Item:        "1",
				HHPN:        "PN1",
				Description: "Capacitor",
				Supplier:    "MURATA",
				SupplierPn:  "CAP1",
				Qty:         2,
				Location:    "C1,C2",
				Selections:  map[string]string{}, // Empty Selections
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

	// 1. Verify Row 5 (Model Qty) is empty, not default 1
	qtyA, _ := f.GetCellValue("BigMatrix", "H5")
	if qtyA != "" {
		t.Errorf("Expected H5 (Model Qty) to be empty when no qty provided, got '%s'", qtyA)
	}

	// 2. Verify Row 6 Selection is empty, not automatically 'V'
	selA, _ := f.GetCellValue("BigMatrix", "H6")
	if selA != "" {
		t.Errorf("Expected H6 (Selection) to be empty when Selections map is empty, got '%s'", selA)
	}
}

// TestExportBigMatrix_GroupZebraStriping 驗證匯出時斑馬紋是依據「物料 Group」切換，而非依據行號 (Row) 切換
func TestExportBigMatrix_GroupZebraStriping(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bomix-group-zebra-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writer, err := NewWriter(nil)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	options := ExportOptions{
		Format:      types.FormatBigMatrix,
		OutputPath:  filepath.Join(tmpDir, "group_zebra_test.xlsx"),
		Description: "Group Zebra Test",
		Revisions: []RevisionData{
			{
				ID:          "10",
				ProjectCode: "ZEBRA_PROJ",
				Phase:       "EVT",
				Version:     "1.0",
				ModelQty:    map[string]int{"A": 1},
			},
		},
		PartData: []PartData{
			// Group 0: Main Part 1 + 1 Second Source -> Rows 6 and 7
			{
				Item:        "1",
				HHPN:        "MAIN_1",
				Description: "Resistor 10K",
				Supplier:    "YAGEO",
				SupplierPn:  "R10K",
				Qty:         1,
				Location:    "R1",
				SecondSources: []SecondSourceData{
					{
						HHPN:        "ALT_1",
						Supplier:    "UNI-ROYAL",
						SupplierPn:  "R10K_ALT",
						Description: "Resistor 10K Alt",
					},
				},
			},
			// Group 1: Main Part 2 -> Row 8
			{
				Item:        "2",
				HHPN:        "MAIN_2",
				Description: "Capacitor 10uF",
				Supplier:    "MURATA",
				SupplierPn:  "C10U",
				Qty:         1,
				Location:    "C1",
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

	// Verify H column (dynamic Model column)
	styleH6, errH6 := f.GetCellStyle("BigMatrix", "H6")
	styleH7, errH7 := f.GetCellStyle("BigMatrix", "H7")
	styleH8, errH8 := f.GetCellStyle("BigMatrix", "H8")

	if errH6 != nil || errH7 != nil || errH8 != nil {
		t.Fatalf("Failed to get H cell styles: %v, %v, %v", errH6, errH7, errH8)
	}

	if styleH6 != styleH7 {
		t.Errorf("Group 0 main part (H6, style %d) and second source (H7, style %d) should have the SAME style", styleH6, styleH7)
	}

	if styleH6 == styleH8 {
		t.Errorf("Group 0 (H6, style %d) and Group 1 (H8, style %d) should have DIFFERENT styles", styleH6, styleH8)
	}

	// Verify A column (basic part data columns A-G)
	styleA6, errA6 := f.GetCellStyle("BigMatrix", "A6")
	styleA7, errA7 := f.GetCellStyle("BigMatrix", "A7")
	styleA8, errA8 := f.GetCellStyle("BigMatrix", "A8")

	if errA6 != nil || errA7 != nil || errA8 != nil {
		t.Fatalf("Failed to get A cell styles: %v, %v, %v", errA6, errA7, errA8)
	}

	if styleA6 != styleA7 {
		t.Errorf("Group 0 main part (A6, style %d) and second source (A7, style %d) should have the SAME style", styleA6, styleA7)
	}

	if styleA6 == styleA8 {
		t.Errorf("Group 0 (A6, style %d) and Group 1 (A8, style %d) should have DIFFERENT styles", styleA6, styleA8)
	}
}

// TestExportBigMatrix_DynamicModelCountAndSelections 驗證 BigMatrix 匯出依據實際存在 Model 數量動態寫入 Model Qty 與 V 勾選
func TestExportBigMatrix_DynamicModelCountAndSelections(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bigmatrix_dynamic_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writer, err := NewWriter(nil)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	options := ExportOptions{
		Format:      types.FormatBigMatrix,
		OutputPath:  filepath.Join(tmpDir, "dynamic_bigmatrix.xlsx"),
		Description: "Dynamic BigMatrix Test",
		Revisions: []RevisionData{
			{
				ID:          "101",
				ProjectCode: "PROJ_A",
				Phase:       "PV",
				Version:     "0.1",
				ModelNames:  []string{"Model 1", "Model 2"},
				ModelQtyByOrder: map[int]int{
					0: 5,
					1: 10,
				},
			},
			{
				ID:          "102",
				ProjectCode: "PROJ_A",
				Phase:       "PV",
				Version:     "0.2",
				ModelNames:  []string{"Model X"},
				ModelQtyByOrder: map[int]int{
					0: 3,
				},
			},
		},
		PartData: []PartData{
			{
				Item:        "1",
				HHPN:        "PN_001",
				Description: "Chip Resistor",
				Supplier:    "YAGEO",
				SupplierPn:  "R10K",
				Qty:         2,
				SelectionsByRevAndOrder: map[string]map[int]string{
					"101": {
						0: "R10K",
						1: "R10K",
					},
					"102": {
						0: "R10K",
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

	// Rev 1 (ID 101): 有 2 個 Model -> 佔用 H, I (Col 7, 8)
	// H4 = A, I4 = B
	// H5 = 5, I5 = 10
	// H6 = V, I6 = V
	nameH4, _ := f.GetCellValue("BigMatrix", "H4")
	nameI4, _ := f.GetCellValue("BigMatrix", "I4")
	if nameH4 != "A" || nameI4 != "B" {
		t.Errorf("Expected H4/I4 to be A / B, got %s / %s", nameH4, nameI4)
	}

	qtyH5, _ := f.GetCellValue("BigMatrix", "H5")
	qtyI5, _ := f.GetCellValue("BigMatrix", "I5")
	if qtyH5 != "5" || qtyI5 != "10" {
		t.Errorf("Expected H5/I5 to be 5 / 10, got %s / %s", qtyH5, qtyI5)
	}

	selH6, _ := f.GetCellValue("BigMatrix", "H6")
	selI6, _ := f.GetCellValue("BigMatrix", "I6")
	if selH6 != "V" || selI6 != "V" {
		t.Errorf("Expected H6/I6 to be V / V, got %s / %s", selH6, selI6)
	}

	// Rev 2 (ID 102): 有 1 個 Model -> 佔用 J (Col 9)
	// J4 = A, J5 = 3, J6 = V
	nameJ4, _ := f.GetCellValue("BigMatrix", "J4")
	qtyJ5, _ := f.GetCellValue("BigMatrix", "J5")
	selJ6, _ := f.GetCellValue("BigMatrix", "J6")
	if nameJ4 != "A" || qtyJ5 != "3" || selJ6 != "V" {
		t.Errorf("Expected J4/J5/J6 to be A / 3 / V, got %s / %s / %s", nameJ4, qtyJ5, selJ6)
	}
}

// TestExportBigMatrix_IsolationBetweenRevisions 驗證 3 個 Revision 中只有 1 個有 Selection 時，其餘 2 個 Revision 的 Model 勾選為空白，不會跨 Revision 誤複製
func TestExportBigMatrix_IsolationBetweenRevisions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bigmatrix_iso_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writer, err := NewWriter(nil)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	options := ExportOptions{
		Format:      types.FormatBigMatrix,
		OutputPath:  filepath.Join(tmpDir, "iso_bigmatrix.xlsx"),
		Description: "Isolation Test",
		Revisions: []RevisionData{
			{
				ID:              "10",
				ProjectCode:     "PROJ_ISO",
				Phase:           "EVT",
				Version:         "0.1",
				ModelNames:      []string{"Model A"},
				ModelQtyByOrder: map[int]int{0: 1},
			},
			{
				ID:              "20",
				ProjectCode:     "PROJ_ISO",
				Phase:           "EVT",
				Version:         "0.2",
				ModelNames:      []string{"Model A"},
				ModelQtyByOrder: map[int]int{0: 1},
			},
			{
				ID:              "30",
				ProjectCode:     "PROJ_ISO",
				Phase:           "EVT",
				Version:         "0.3",
				ModelNames:      []string{"Model A"},
				ModelQtyByOrder: map[int]int{0: 1},
			},
		},
		PartData: []PartData{
			{
				Item:        "1",
				HHPN:        "ISO_PN_001",
				Description: "Resistor",
				Supplier:    "YAGEO",
				SupplierPn:  "R10K_ISO",
				Qty:         1,
				// 全域相容 Selections (模擬 app.go 第一個 Revision 寫入的全域相容 Selection)
				Selections: map[string]string{
					"Model A": "R10K_ISO",
				},
				SelectionsByOrder: map[int]string{
					0: "R10K_ISO",
				},
				// 只有 Rev 10 有實際的 SelectionsByRevAndOrder
				SelectionsByRevAndOrder: map[string]map[int]string{
					"10": {
						0: "R10K_ISO",
					},
					// Rev 20 與 Rev 30 為空 (無勾選)
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

	// Rev 10 (Col H / Col 7): H6 應為 "V"
	selH6, _ := f.GetCellValue("BigMatrix", "H6")
	if selH6 != "V" {
		t.Errorf("Expected H6 (Rev 10 Selection) to be 'V', got '%s'", selH6)
	}

	// Rev 20 (Col I / Col 8): I6 應為 "" (空白，不得被複製為 V)
	selI6, _ := f.GetCellValue("BigMatrix", "I6")
	if selI6 != "" {
		t.Errorf("Expected I6 (Rev 20 Selection) to be empty '', got '%s'", selI6)
	}

	// Rev 30 (Col J / Col 9): J6 應為 "" (空白，不得被複製為 V)
	selJ6, _ := f.GetCellValue("BigMatrix", "J6")
	if selJ6 != "" {
		t.Errorf("Expected J6 (Rev 30 Selection) to be empty '', got '%s'", selJ6)
	}
}

// TestExportBigMatrix_SecondSourceGrayStyleIsolation 驗證 2nd Source 替代料在不屬於的 Revision 欄位會精確填入灰色底色
func TestExportBigMatrix_SecondSourceGrayStyleIsolation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bigmatrix_ss_gray_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writer, err := NewWriter(nil)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	options := ExportOptions{
		Format:      types.FormatBigMatrix,
		OutputPath:  filepath.Join(tmpDir, "ss_gray_bigmatrix.xlsx"),
		Description: "Second Source Gray Style Test",
		Revisions: []RevisionData{
			{
				ID:              "1",
				ProjectCode:     "PROJ_SS",
				Phase:           "EVT",
				Version:         "0.1",
				ModelNames:      []string{"Model A"},
				ModelQtyByOrder: map[int]int{0: 1},
			},
			{
				ID:              "2",
				ProjectCode:     "PROJ_SS",
				Phase:           "EVT",
				Version:         "0.2",
				ModelNames:      []string{"Model A"},
				ModelQtyByOrder: map[int]int{0: 1},
			},
		},
		PartData: []PartData{
			{
				Item:              "1",
				HHPN:              "MAIN_PN",
				Description:       "Main Resistor",
				Supplier:          "YAGEO",
				SupplierPn:        "MAIN_R10K",
				Qty:               1,
				SourceRevisionIDs: []int64{1, 2}, // 主料在 Rev 1 與 Rev 2 均存在
				SecondSources: []SecondSourceData{
					{
						HHPN:              "SS_PN_REV1_ONLY",
						Supplier:          "MURATA",
						SupplierPn:        "SS_MURATA_R10K",
						Description:       "Second Source Rev 1 Only",
						SourceRevisionIDs: []int64{1}, // 此 2nd Source 僅存在於 Rev 1！
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

	// Row 6 是主料，Row 7 是 2nd Source (SS_PN_REV1_ONLY)
	// Col H (Col 7) 對應 Rev 1，Col I (Col 8) 對應 Rev 2
	// 2nd Source 在 Rev 1 (H7) 應該是一般樣式，在 Rev 2 (I7) 必須填入灰色底色
	styleH7, errH7 := f.GetCellStyle("BigMatrix", "H7")
	styleI7, errI7 := f.GetCellStyle("BigMatrix", "I7")

	if errH7 != nil || errI7 != nil {
		t.Fatalf("Failed to get cell styles H7/I7: %v, %v", errH7, errI7)
	}

	// H7 (Rev 1) 與 I7 (Rev 2, 灰色底色) 的 Style ID 必須不同
	if styleH7 == styleI7 {
		t.Errorf("Expected 2nd source in Rev 2 (I7, style %d) to have GRAY background different from Rev 1 (H7, style %d)", styleI7, styleH7)
	}
}

// TestExportBigMatrix_MainPartGrayStyleIsolation 驗證主料在不屬於的 Revision 欄位會精確填入灰色底色且內容為空
func TestExportBigMatrix_MainPartGrayStyleIsolation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bigmatrix_main_gray_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writer, err := NewWriter(nil)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	options := ExportOptions{
		Format:      types.FormatBigMatrix,
		OutputPath:  filepath.Join(tmpDir, "main_gray_bigmatrix.xlsx"),
		Description: "Main Part Gray Style Test",
		Revisions: []RevisionData{
			{
				ID:              "1",
				ProjectCode:     "PROJ_MAIN",
				Phase:           "EVT",
				Version:         "0.1",
				ModelNames:      []string{"Model A"},
				ModelQtyByOrder: map[int]int{0: 1},
			},
			{
				ID:              "2",
				ProjectCode:     "PROJ_MAIN",
				Phase:           "EVT",
				Version:         "0.2",
				ModelNames:      []string{"Model A"},
				ModelQtyByOrder: map[int]int{0: 1},
			},
		},
		PartData: []PartData{
			{
				Item:              "1",
				HHPN:              "MAIN_REV1_ONLY",
				Description:       "Main Resistor Rev 1 Only",
				Supplier:          "YAGEO",
				SupplierPn:        "MAIN_R10K",
				Qty:               1,
				SourceRevisionIDs: []int64{1}, // 此主料僅存在於 Rev 1！在 Rev 2 不存在
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

	// Row 6 是主料 (MAIN_REV1_ONLY)
	// Col H (Col 7) 對應 Rev 1，Col I (Col 8) 對應 Rev 2
	// 主料在 Rev 1 (H6) 是一般樣式，在 Rev 2 (I6) 必須填入灰色底色且值為空
	styleH6, errH6 := f.GetCellStyle("BigMatrix", "H6")
	styleI6, errI6 := f.GetCellStyle("BigMatrix", "I6")
	valI6, _ := f.GetCellValue("BigMatrix", "I6")

	if errH6 != nil || errI6 != nil {
		t.Fatalf("Failed to get cell styles H6/I6: %v, %v", errH6, errI6)
	}

	// H6 (Rev 1) 與 I6 (Rev 2, 灰色底色) 的 Style ID 必須不同，且 I6 為空
	if styleH6 == styleI6 {
		t.Errorf("Expected main part in Rev 2 (I6, style %d) to have GRAY background different from Rev 1 (H6, style %d)", styleI6, styleH6)
	}
	if valI6 != "" {
		t.Errorf("Expected I6 (main part selection in Rev 2) to be empty '', got '%s'", valI6)
	}
}

// TestExportBigMatrix_ProtoGroupRowTextColor 測試當物料群組 BOMStatus == "P" 時，
// 主料與 2nd source 整個 row 的文字顏色是否被正確設定為 #8080C0
func TestExportBigMatrix_ProtoGroupRowTextColor(t *testing.T) {
	tmpDir := t.TempDir()

	writer, err := NewWriter(nil)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	options := ExportOptions{
		Format:      types.FormatBigMatrix,
		OutputPath:  filepath.Join(tmpDir, "proto_color_test.xlsx"),
		Description: "Proto Color Test",
		Revisions: []RevisionData{
			{
				ID:              "1",
				ProjectCode:     "PROJ_PROTO",
				Phase:           "EVT",
				Version:         "0.1",
				ModelNames:      []string{"Model A"},
				ModelQtyByOrder: map[int]int{0: 1},
			},
		},
		PartData: []PartData{
			{
				Item:        "1",
				HHPN:        "PROTO_HHPN_01",
				Description: "Proto Resistor 10K",
				Supplier:    "YAGEO",
				SupplierPn:  "PROTO_R10K",
				Qty:         2,
				Location:    "C1,C2",
				BOMStatus:   "P", // PROTO 物料群組！
				SecondSources: []SecondSourceData{
					{
						HHPN:        "PROTO_HHPN_01_SS",
						Supplier:    "WALSIN",
						SupplierPn:  "PROTO_R10K_SS",
						Description: "Proto Resistor 10K 2nd Source",
					},
				},
				SelectionsByRevAndOrder: map[string]map[int]string{
					"1": {0: "PROTO_R10K"},
				},
			},
			{
				Item:        "2",
				HHPN:        "NORMAL_HHPN_02",
				Description: "Normal Resistor 10K",
				Supplier:    "YAGEO",
				SupplierPn:  "NORMAL_R10K",
				Qty:         1,
				Location:    "C3",
				BOMStatus:   "I", // 一般物料群組 (非 PROTO)
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

	// Row 6: PROTO 主料
	// Row 7: PROTO 2nd source
	// Row 8: 一般物料 (BOMStatus = "I")
	rowsToCheck := []struct {
		row           int
		col           string
		expectProto   bool
		description   string
	}{
		{6, "A", true, "PROTO 主料 A 欄"},
		{6, "B", true, "PROTO 主料 B 欄 (HHPN)"},
		{6, "H", true, "PROTO 主料 H 欄 (Model Selection)"},
		{7, "B", true, "PROTO 2nd Source B 欄 (HHPN)"},
		{7, "H", true, "PROTO 2nd Source H 欄 (Model Selection)"},
		{8, "B", false, "一般物料 B 欄 (HHPN)"},
	}

	for _, tc := range rowsToCheck {
		cellAddr := fmt.Sprintf("%s%d", tc.col, tc.row)
		styleID, err := f.GetCellStyle("BigMatrix", cellAddr)
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

// TestExportBigMatrix_OverwriteExistingFileWithWarning 驗證當 BigMatrix 匯出目標檔案已存在時，
// validateAndPrepareOutputPath 能正常刪除舊檔、輸出 Warning log 並成功覆蓋寫入新檔
func TestExportBigMatrix_OverwriteExistingFileWithWarning(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bomix-bigmatrix-overwrite-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	targetPath := filepath.Join(tmpDir, "TestSeries_BigMatrix_EVT_0.1_20260810.xlsx")
	// 預先寫入一舊舊檔內容
	if err := os.WriteFile(targetPath, []byte("old bigmatrix dummy content"), 0644); err != nil {
		t.Fatalf("Failed to write dummy existing file: %v", err)
	}

	lg := logger.NewLogger(100)
	writer, err := NewWriter(lg)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	options := ExportOptions{
		Format:     types.FormatBigMatrix,
		OutputPath: targetPath,
		PartData: []PartData{
			{
				Item:        "1",
				HHPN:        "OVERWRITE_BM_PN",
				Description: "Overwrite Test BM Part",
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
				ProjectCode:     "TestSeries",
				Phase:           "EVT",
				Version:         "0.1",
				ModelNames:      []string{"Model A"},
				ModelQtyByOrder: map[int]int{0: 1},
			},
		},
	}

	paths, err := writer.ExportExcel(options)
	if err != nil {
		t.Fatalf("ExportExcel failed when target BigMatrix file exists: %v", err)
	}

	if len(paths) == 0 || paths[0] != targetPath {
		t.Fatalf("Expected output path %s, got %v", targetPath, paths)
	}

	f, err := excelize.OpenFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to open overwritten BigMatrix Excel file: %v", err)
	}
	_ = f.Close()
}

// TestExportBigMatrix_Row1Formula 驗證 BigMatrix 匯出時，
// 各 Model 欄位的 Row 1 寫入正確公式 cnt_M - cnt_N - COUNTIF(<Col>6:<Col><EndRow>, "V")
func TestExportBigMatrix_Row1Formula(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bomix-bigmatrix-row1formula-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	targetPath := filepath.Join(tmpDir, "Row1Formula_Test.xlsx")
	lg := logger.NewLogger(100)
	writer, err := NewWriter(lg)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	// 模擬 3 個主料群組 (cnt_M = 3)
	// Part 1: SourceRevisionIDs = [101, 102] (存在於 Rev 101 與 Rev 102)
	// Part 2: SourceRevisionIDs = [101, 102] (存在於 Rev 101 與 Rev 102)
	// Part 3: SourceRevisionIDs = [102]      (僅存在於 Rev 102，Rev 101 不存在，cnt_N for Rev 101 = 1)
	parts := []PartData{
		{
			Item:              "1",
			HHPN:              "PN001",
			Description:       "Capacitor 10uF",
			Supplier:          "YAGEO",
			SupplierPn:        "CC0402_10U",
			Qty:               1,
			Location:          "C1",
			Type:              "SMD",
			BOMStatus:         "I",
			CCL:               true,
			SourceRevisionIDs: []int64{101, 102},
			SelectionsByRevAndOrder: map[string]map[int]string{
				"101": {0: "CC0402_10U"},
				"102": {0: "CC0402_10U"},
			},
		},
		{
			Item:              "2",
			HHPN:              "PN002",
			Description:       "Resistor 10K",
			Supplier:          "YAGEO",
			SupplierPn:        "R0402_10K",
			Qty:               1,
			Location:          "R1",
			Type:              "SMD",
			BOMStatus:         "I",
			CCL:               true,
			SourceRevisionIDs: []int64{101, 102},
			SelectionsByRevAndOrder: map[string]map[int]string{
				"101": {0: "R0402_10K"},
				"102": {0: "R0402_10K"},
			},
		},
		{
			Item:              "3",
			HHPN:              "PN003",
			Description:       "Inductor 1uH",
			Supplier:          "MURATA",
			SupplierPn:        "L0402_1U",
			Qty:               1,
			Location:          "L1",
			Type:              "SMD",
			BOMStatus:         "I",
			CCL:               true,
			SourceRevisionIDs: []int64{102}, // 只在 Rev 102
			SelectionsByRevAndOrder: map[string]map[int]string{
				"102": {0: "L0402_1U"},
			},
		},
	}

	revisions := []RevisionData{
		{
			ID:              "101",
			ProjectCode:     "TEST_SERIES",
			Phase:           "EVT",
			Version:         "0.1",
			ModelNames:      []string{"Model A", "Model B"},
			ModelQtyByOrder: map[int]int{0: 1, 1: 1},
		},
		{
			ID:              "102",
			ProjectCode:     "TEST_SERIES",
			Phase:           "DVT",
			Version:         "0.2",
			ModelNames:      []string{"Model A"},
			ModelQtyByOrder: map[int]int{0: 1},
		},
	}

	options := ExportOptions{
		Format:     types.FormatBigMatrix,
		OutputPath: targetPath,
		PartData:   parts,
		Revisions:  revisions,
	}

	paths, err := writer.ExportExcel(options)
	if err != nil {
		t.Fatalf("ExportExcel failed: %v", err)
	}
	if len(paths) == 0 {
		t.Fatalf("Expected output path, got empty slice")
	}

	f, err := excelize.OpenFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to open exported Excel file: %v", err)
	}
	defer f.Close()


	// Rev 101 有 2 個 Model -> Column H (Model A), Column I (Model B)
	// Rev 102 有 1 個 Model -> Column J (Model A)
	// 資料列為 Row 6..8 (endRow = 8)
	//
	// Rev 101 的 cnt_M = 3, cnt_N = 1 (Part 3 不存在) -> 公式應為 3-1-COUNTIF(H6:H8,"V")
	// Rev 102 的 cnt_M = 3, cnt_N = 0 (全部位於 102) -> 公式應為 3-0-COUNTIF(J6:J8,"V")

	cellFormulaH1, err := f.GetCellFormula("BigMatrix", "H1")
	if err != nil {
		t.Fatalf("GetCellFormula(H1) failed: %v", err)
	}
	expectedH1 := "3-1-COUNTIF(H6:H8,\"V\")"
	if cellFormulaH1 != expectedH1 {
		t.Errorf("H1 formula = %q, want %q", cellFormulaH1, expectedH1)
	}

	cellFormulaI1, err := f.GetCellFormula("BigMatrix", "I1")
	if err != nil {
		t.Fatalf("GetCellFormula(I1) failed: %v", err)
	}
	expectedI1 := "3-1-COUNTIF(I6:I8,\"V\")"
	if cellFormulaI1 != expectedI1 {
		t.Errorf("I1 formula = %q, want %q", cellFormulaI1, expectedI1)
	}

	cellFormulaJ1, err := f.GetCellFormula("BigMatrix", "J1")
	if err != nil {
		t.Fatalf("GetCellFormula(J1) failed: %v", err)
	}
	expectedJ1 := "3-0-COUNTIF(J6:J8,\"V\")"
	if cellFormulaJ1 != expectedJ1 {
		t.Errorf("J1 formula = %q, want %q", cellFormulaJ1, expectedJ1)
	}

	// 驗證 Row 1 的條件格式化範圍是否已成功覆蓋全體 Model 欄位 (H1:J1)
	cfMap, err := f.GetConditionalFormats("BigMatrix")
	if err != nil {
		t.Fatalf("GetConditionalFormats failed: %v", err)
	}
	hasCFOnH1J1 := false
	for rng := range cfMap {
		if strings.Contains(rng, "H1") && strings.Contains(rng, "J1") {
			hasCFOnH1J1 = true
			break
		}
	}
	if !hasCFOnH1J1 {
		t.Errorf("Expected conditional format range covering H1:J1, got cfMap: %+v", cfMap)
	}
}

func TestAddModelSelectionConditionalFormatting(t *testing.T) {
	// 建立測試資料：包含 2 個物料 Group 與 2 個 BOM Revision
	// Group 1 (Row 6..7): 1 主料 + 1 替代料
	// Group 2 (Row 8): 1 主料 (無替代料)
	// Rev 101 (Col H, I): 2 個 Model
	// Rev 102 (Col J): 1 個 Model

	tempDir, err := os.MkdirTemp("", "bigmatrix_cf_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	targetPath := filepath.Join(tempDir, "CF_BigMatrix_Test.xlsx")
	lg := logger.NewLogger(100)
	writer, err := NewWriter(lg)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	parts := []PartData{
		{
			Item:        "1",
			HHPN:        "PN001",
			Description: "Main Part 1",
			Supplier:    "Supp A",
			SupplierPn:  "SPN001",
			Qty:         1,
			Location:    "C1",
			CCL:         true,
			BOMStatus:   "M",
			SecondSources: []SecondSourceData{
				{
					HHPN:        "PN001-SS1",
					Description: "2nd Source 1",
					Supplier:    "Supp B",
					SupplierPn:  "SPN001-SS1",
				},
			},
		},
		{
			Item:        "2",
			HHPN:        "PN002",
			Description: "Main Part 2",
			Supplier:    "Supp C",
			SupplierPn:  "SPN002",
			Qty:         1,
			Location:    "C2",
			CCL:         true,
			BOMStatus:   "M",
		},
	}

	revisions := []RevisionData{
		{
			ID:          "101",
			ProjectCode: "PROJ1",
			Phase:       "EVT",
			Version:     "v1",
			ModelNames:  []string{"Model A", "Model B"},
		},
		{
			ID:          "102",
			ProjectCode: "PROJ1",
			Phase:       "DVT",
			Version:     "v1",
			ModelNames:  []string{"Model C"},
		},
	}

	options := ExportOptions{
		Format:     types.FormatBigMatrix,
		OutputPath: targetPath,
		PartData:   parts,
		Revisions:  revisions,
	}

	paths, err := writer.ExportExcel(options)
	if err != nil {
		t.Fatalf("ExportExcel failed: %v", err)
	}
	if len(paths) == 0 {
		t.Fatalf("Expected output path, got empty slice")
	}

	f, err := excelize.OpenFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to open exported Excel file: %v", err)
	}
	defer f.Close()

	cfMap, err := f.GetConditionalFormats("BigMatrix")
	if err != nil {
		t.Fatalf("GetConditionalFormats failed: %v", err)
	}

	// Group 1 在 Rev 101 (Col H..I) 的非灰底範圍為 H6:I7
	// 相對欄位公式應為: OR(COUNTIF(H$6:H$7,"V")<>1,COUNTIF(H$6:H$7,"")+COUNTIF(H$6:H$7,"V")<>2)
	expectedFormulaGroup1 := "OR(COUNTIF(H$6:H$7,\"V\")<>1,COUNTIF(H$6:H$7,\"\")+COUNTIF(H$6:H$7,\"V\")<>2)"

	foundFormula1 := false
	if cfList, ok := cfMap["H6:I7"]; ok {
		for _, cf := range cfList {
			if cf.Value == expectedFormulaGroup1 || cf.Criteria == expectedFormulaGroup1 {
				foundFormula1 = true
				break
			}
		}
	}

	if !foundFormula1 {
		t.Errorf("Expected conditional format formula %q on range H6:I7 not found in cfMap: %+v", expectedFormulaGroup1, cfMap)
	}

	// Group 2 在 Rev 102 (Col J) 的非灰底範圍為 J8:J8 (單列)
	// 相對欄位公式應為: OR(COUNTIF(J$8,"V")<>1,COUNTIF(J$8,"")+COUNTIF(J$8,"V")<>1)
	expectedFormulaGroup2 := "OR(COUNTIF(J$8,\"V\")<>1,COUNTIF(J$8,\"\")+COUNTIF(J$8,\"V\")<>1)"

	foundFormula2 := false
	if cfList, ok := cfMap["J8:J8"]; ok {
		for _, cf := range cfList {
			if cf.Value == expectedFormulaGroup2 || cf.Criteria == expectedFormulaGroup2 {
				foundFormula2 = true
				break
			}
		}
	}

	if !foundFormula2 {
		t.Errorf("Expected conditional format formula %q on range J8:J8 not found in cfMap: %+v", expectedFormulaGroup2, cfMap)
	}
}

func TestBigMatrixSelectionSupplierMatching(t *testing.T) {
	part := PartData{
		Supplier:   "LRC",
		SupplierPn: "LMBT3906DW1T1G",
		SelectionsByRevAndMaterial: map[string]map[int]string{
			"101": {
				0: "LRC|LMBT3906DW1T1G",
				1: "PANJIT|MMDT3906",
				2: "BLUEROCKET|MMDT3906",
			},
		},
	}

	ss1 := SecondSourceData{
		Supplier:   "PANJIT",
		SupplierPn: "MMDT3906",
	}

	ss2 := SecondSourceData{
		Supplier:   "BLUEROCKET",
		SupplierPn: "MMDT3906",
	}

	// Model 0 (sortOrder=0): Main source LRC|LMBT3906DW1T1G selected
	if !isBigMatrixMainSourceSelected(part, "101", 0, "A") {
		t.Errorf("Expected main source selected for model 0")
	}
	if isBigMatrixSecondSourceSelected(ss1, part, "101", 0, "A") {
		t.Errorf("Expected ss1 not selected for model 0")
	}
	if isBigMatrixSecondSourceSelected(ss2, part, "101", 0, "A") {
		t.Errorf("Expected ss2 not selected for model 0")
	}

	// Model 1 (sortOrder=1): Second source 1 PANJIT|MMDT3906 selected
	if isBigMatrixMainSourceSelected(part, "101", 1, "B") {
		t.Errorf("Expected main source not selected for model 1")
	}
	if !isBigMatrixSecondSourceSelected(ss1, part, "101", 1, "B") {
		t.Errorf("Expected ss1 selected for model 1")
	}
	if isBigMatrixSecondSourceSelected(ss2, part, "101", 1, "B") {
		t.Errorf("Expected ss2 NOT selected for model 1 (even though PN matches)")
	}

	// Model 2 (sortOrder=2): Second source 2 BLUEROCKET|MMDT3906 selected
	if isBigMatrixMainSourceSelected(part, "101", 2, "C") {
		t.Errorf("Expected main source not selected for model 2")
	}
	if isBigMatrixSecondSourceSelected(ss1, part, "101", 2, "C") {
		t.Errorf("Expected ss1 NOT selected for model 2 (even though PN matches)")
	}
	if !isBigMatrixSecondSourceSelected(ss2, part, "101", 2, "C") {
		t.Errorf("Expected ss2 selected for model 2")
	}
}

// TestExportBigMatrix_SheetProtection 驗證 BigMatrix 匯出時工作表保護設定包含欄格式、列格式與儲存格格式權限
func TestExportBigMatrix_SheetProtection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bomix-protection-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writer, err := NewWriter(nil)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	options := ExportOptions{
		Format:      types.FormatBigMatrix,
		OutputPath:  filepath.Join(tmpDir, "test_protection.xlsx"),
		Description: "Test Sheet Protection",
		PartData:    []PartData{},
	}

	paths, err := writer.ExportExcel(options)
	if err != nil {
		t.Fatalf("ExportExcel failed: %v", err)
	}

	f, err := excelize.OpenFile(paths[0])
	if err != nil {
		t.Fatalf("Failed to open exported Excel file: %v", err)
	}
	defer f.Close()

	opts, err := f.GetSheetProtection("BigMatrix")
	if err != nil {
		t.Fatalf("GetSheetProtection failed: %v", err)
	}

	if !opts.FormatCells {
		t.Errorf("Expected FormatCells to be true, got false")
	}
	if !opts.FormatColumns {
		t.Errorf("Expected FormatColumns to be true, got false")
	}
	if !opts.FormatRows {
		t.Errorf("Expected FormatRows to be true, got false")
	}
}

// TestExportBigMatrix_ClearTemplateResidualData 驗證 BigMatrix 匯出時會清空 Row 6 與 Row 7 範本殘留資料
func TestExportBigMatrix_ClearTemplateResidualData(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bomix-clear-template-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writer, err := NewWriter(nil)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	// 測試空物料列表時，Row 6 & 7 殘留範本文字應被完全清空
	options := ExportOptions{
		Format:      types.FormatBigMatrix,
		OutputPath:  filepath.Join(tmpDir, "test_clear_template.xlsx"),
		Description: "Test Clear Template",
		PartData:    []PartData{},
	}

	paths, err := writer.ExportExcel(options)
	if err != nil {
		t.Fatalf("ExportExcel failed: %v", err)
	}

	f, err := excelize.OpenFile(paths[0])
	if err != nil {
		t.Fatalf("Failed to open exported Excel file: %v", err)
	}
	defer f.Close()

	for row := 6; row <= 7; row++ {
		for col := 'A'; col <= 'G'; col++ {
			val, _ := f.GetCellValue("BigMatrix", fmt.Sprintf("%c%d", col, row))
			if val != "" {
				t.Errorf("Expected cell %c%d to be empty, got: '%s'", col, row, val)
			}
		}
	}
}








