package excel

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
			CCL:         "Y",
			BOMStatus:   "I",
			Description: "Test Part 1",
		},
		{
			Supplier:    "Murata",
			SupplierPn:  "GRM155B81C105KE19D",
			CCL:         "Y",
			BOMStatus:   "P",
			Description: "Test Part 2 (NPI)",
		},
		{
			Supplier:    "Taiyo Yuden",
			SupplierPn:  "UMK105B7105KV-F",
			CCL:         "Y",
			BOMStatus:   "M",
			Description: "Test Part 3 (MP)",
		},
		{
			Supplier:    "Yageo",
			SupplierPn:  "CC0402KRX7R9BB104",
			CCL:         "Y",
			BOMStatus:   "X",
			Description: "Test Part 4 (Not Installed)",
		},
		{
			Supplier:    "Kemet",
			SupplierPn:  "C0603C104K5RACTU",
			CCL:         "N",
			BOMStatus:   "I",
			Description: "Test Part 5 (Not CCL)",
		},
	}

	// Test NPI mode
	npiFiltered := FilterPartsByCriteria(parts, "NPI")
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
	mpFiltered := FilterPartsByCriteria(parts, "MP")
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
				CCL:         "Y",
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
				CCL:         "Y",
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



