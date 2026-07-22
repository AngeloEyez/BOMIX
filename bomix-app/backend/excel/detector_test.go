package excel

import (
	"os"
	"testing"

	"github.com/xuri/excelize/v2"
	"bomix-app/backend/types"
)

// TestDetect_EBOM tests detection of EBOM format
func TestDetect_EBOM(t *testing.T) {
	// Create a temporary EBOM format file
	f := excelize.NewFile()
	wb := &ExcelizeWorkbook{f: f}
	_ = wb
	defer f.Close()

	// Create required sheets
	sheets := []string{"SMD", "PTH", "BOTTOM", "NI", "PROTO", "MP"}
	for _, sheet := range sheets {
		f.NewSheet(sheet)
	}

	// Set EBOM-specific header values in SMD sheet
	f.SetCellValue("SMD", "H5", "Qty")
	f.SetCellValue("SMD", "J7", "CCL")

	// Detect format
	detector := NewDetector()
	format, err := detector.Detect(wb)

	if err != nil {
		t.Fatalf("Detect failed with error: %v", err)
	}

	if format != types.FormatEBOM {
		t.Errorf("Expected format EBOM, got %v", format)
	}
}

// TestDetect_BigMatrix tests detection of BigMatrix format
func TestDetect_BigMatrix(t *testing.T) {
	// Create a temporary BigMatrix format file
	f := excelize.NewFile()
	wb := &ExcelizeWorkbook{f: f}
	_ = wb
	defer f.Close()

	// Create BigMatrix sheet
	f.NewSheet("BigMatrix")

	// Detect format
	detector := NewDetector()
	format, err := detector.Detect(wb)

	if err != nil {
		t.Fatalf("Detect failed with error: %v", err)
	}

	if format != types.FormatBigMatrix {
		t.Errorf("Expected format BigMatrix, got %v", format)
	}
}

// TestDetect_Matrix tests detection of Matrix format
func TestDetect_Matrix(t *testing.T) {
	// Create a temporary Matrix format file
	f := excelize.NewFile()
	wb := &ExcelizeWorkbook{f: f}
	_ = wb
	defer f.Close()

	// Create SMD sheet
	f.NewSheet("SMD")

	// Set Matrix-specific header values
	f.SetCellValue("SMD", "H5", "Location")
	f.SetCellValue("SMD", "J7", "Total Set")

	// Detect format
	detector := NewDetector()
	format, err := detector.Detect(wb)

	if err != nil {
		t.Fatalf("Detect failed with error: %v", err)
	}

	if format != types.FormatMatrix {
		t.Errorf("Expected format Matrix, got %v", format)
	}
}

// TestDetect_Unknown tests detection of unknown format
func TestDetect_Unknown(t *testing.T) {
	// Create a temporary file with unknown format
	f := excelize.NewFile()
	wb := &ExcelizeWorkbook{f: f}
	_ = wb
	defer f.Close()

	// Create a generic sheet without specific markers
	f.NewSheet("Sheet1")

	// Detect format
	detector := NewDetector()
	format, err := detector.Detect(wb)

	if err != nil {
		t.Fatalf("Detect failed with error: %v", err)
	}

	if format != types.FormatUnknown {
		t.Errorf("Expected format Unknown, got %v", format)
	}
}

// TestDetect_EmptySheet tests detection with empty sheets
func TestDetect_EmptySheet(t *testing.T) {
	// Create a temporary file with empty sheet
	f := excelize.NewFile()
	wb := &ExcelizeWorkbook{f: f}
	_ = wb
	defer f.Close()

	// Detect format - should return Unknown for a generic empty sheet
	detector := NewDetector()
	format, err := detector.Detect(wb)

	// Empty generic sheet should be detected as Unknown, not error
	if err != nil {
		t.Fatalf("Detect failed with error: %v", err)
	}

	if format != types.FormatUnknown {
		t.Errorf("Expected format Unknown, got %v", format)
	}
}

// TestDetect_EBOMWithPartialSheets tests EBOM detection with partial sheets
func TestDetect_EBOMWithPartialSheets(t *testing.T) {
	// Create a file with only SMD, PTH, BOTTOM (no NI, PROTO, MP)
	f := excelize.NewFile()
	wb := &ExcelizeWorkbook{f: f}
	_ = wb
	defer f.Close()

	sheets := []string{"SMD", "PTH", "BOTTOM"}
	for _, sheet := range sheets {
		f.NewSheet(sheet)
	}

	// Set EBOM-specific header values
	f.SetCellValue("SMD", "H5", "Qty")
	f.SetCellValue("SMD", "J7", "CCL")

	// Detect format
	detector := NewDetector()
	format, err := detector.Detect(wb)

	if err != nil {
		t.Fatalf("Detect failed with error: %v", err)
	}

	if format != types.FormatEBOM {
		t.Errorf("Expected format EBOM, got %v", format)
	}
}

// TestDetect_CaseInsensitiveSheetNames tests case-insensitive sheet name matching
func TestDetect_CaseInsensitiveSheetNames(t *testing.T) {
	f := excelize.NewFile()
	wb := &ExcelizeWorkbook{f: f}
	_ = wb
	defer f.Close()

	// Create sheets with different case
	sheets := []string{"smd", "pth", "bottom"}
	for _, sheet := range sheets {
		f.NewSheet(sheet)
	}

	// Set header values (case shouldn't matter for sheet detection)
	f.SetCellValue("smd", "H5", "Qty")
	f.SetCellValue("smd", "J7", "CCL")

	detector := NewDetector()
	format, err := detector.Detect(wb)

	if err != nil {
		t.Fatalf("Detect failed with error: %v", err)
	}

	if format != types.FormatEBOM {
		t.Errorf("Expected format EBOM, got %v", format)
	}
}

// TestDetect_FileRoundTrip tests saving and reloading a detected file
func TestDetect_FileRoundTrip(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_*.xlsx")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Create EBOM format file
	f := excelize.NewFile()
	wb := &ExcelizeWorkbook{f: f}
	_ = wb
	f.NewSheet("SMD")
	f.NewSheet("PTH")
	f.NewSheet("BOTTOM")
	f.SetCellValue("SMD", "H5", "Qty")
	f.SetCellValue("SMD", "J7", "CCL")

	if err := f.SaveAs(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to save file: %v", err)
	}

	// Reopen and detect
	f2, err := excelize.OpenFile(tmpFile.Name())
	wb2 := &ExcelizeWorkbook{f: f2}
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	defer f2.Close()

	detector := NewDetector()
	format, err := detector.Detect(wb2)

	if err != nil {
		t.Fatalf("Detect failed with error: %v", err)
	}

	if format != types.FormatEBOM {
		t.Errorf("Expected format EBOM, got %v", format)
	}
}
