package excel

import (
	"errors"
	"strings"

	"github.com/xuri/excelize/v2"
	"bomix-app/backend/types"
)

// Detector detects the format of a BOM Excel file
// See product-spec section 6.3.1
type Detector struct{}

// NewDetector creates a new detector instance
func NewDetector() *Detector {
	return &Detector{}
}

// Detect detects the BOM format from an Excel file
// Returns FormatEBOM, FormatBigMatrix, FormatMatrix, or FormatUnknown
// See product-spec section 6.3.1 格式辨識與偵測規則對照表
func (d *Detector) Detect(f *excelize.File) (types.BOMFormat, error) {
	sheets := f.GetSheetList()

	if len(sheets) == 0 {
		return types.FormatUnknown, errors.New("no sheets found")
	}

	// Check for BigMatrix format: exists sheet named "BigMatrix"
	if d.hasSheet(sheets, "BigMatrix") {
		return types.FormatBigMatrix, nil
	}

	// Check for EBOM format: has SMD, PTH, BOTTOM, NI, PROTO, MP sheets
	// AND SMD/PTH/BOTTOM header cells match specific patterns
	if d.isEBOMFormat(f, sheets) {
		return types.FormatEBOM, nil
	}

	// Check for Matrix format: has SMD sheet with specific header pattern
	if d.isMatrixFormat(f, sheets) {
		return types.FormatMatrix, nil
	}

	return types.FormatUnknown, nil
}

// hasSheet checks if a sheet with the given name exists (case-insensitive)
func (d *Detector) hasSheet(sheets []string, name string) bool {
	for _, sheet := range sheets {
		if strings.EqualFold(sheet, name) {
			return true
		}
	}
	return false
}

// isEBOMFormat checks if the file matches EBOM format
// Conditions:
// 1. Contains SMD, PTH, BOTTOM, NI, PROTO, MP sheets
// 2. SMD/PTH/BOTTOM header H5="Qty" and J7="CCL"
func (d *Detector) isEBOMFormat(f *excelize.File, sheets []string) bool {
	// Check for required sheets
	requiredSheets := []string{"SMD", "PTH", "BOTTOM"}
	foundSheets := 0

	for _, required := range requiredSheets {
		for _, sheet := range sheets {
			if strings.EqualFold(sheet, required) {
				foundSheets++
				break
			}
		}
	}

	// Need at least SMD, PTH, BOTTOM
	if foundSheets < 3 {
		return false
	}

	// Check header patterns in SMD sheet
	// H5 should be "Qty" and J7 should be "CCL"
	smdSheet := "SMD"
	for _, sheet := range sheets {
		if strings.EqualFold(sheet, "SMD") {
			smdSheet = sheet
			break
		}
	}

	// Check H5 (Qty)
	valH5, _ := f.GetCellValue(smdSheet, "H5")
	if !strings.EqualFold(strings.TrimSpace(valH5), "Qty") {
		return false
	}

	// Check J7 (CCL)
	valJ7, _ := f.GetCellValue(smdSheet, "J7")
	if !strings.EqualFold(strings.TrimSpace(valJ7), "CCL") {
		return false
	}

	return true
}

// isMatrixFormat checks if the file matches Matrix format
// Conditions:
// 1. Has SMD sheet
// 2. SMD header H5="Location" and J7="Total Set"
func (d *Detector) isMatrixFormat(f *excelize.File, sheets []string) bool {
	// Check for SMD sheet
	hasSMD := false
	smdSheet := ""
	for _, sheet := range sheets {
		if strings.EqualFold(sheet, "SMD") {
			hasSMD = true
			smdSheet = sheet
			break
		}
	}

	if !hasSMD {
		return false
	}

	// Check H5 (Location) - different from EBOM which has "Qty"
	valH5, _ := f.GetCellValue(smdSheet, "H5")
	if !strings.EqualFold(strings.TrimSpace(valH5), "Location") {
		return false
	}

	// Check J7 (Total Set) - different from EBOM which has "CCL"
	valJ7, _ := f.GetCellValue(smdSheet, "J7")
	if !strings.EqualFold(strings.TrimSpace(valJ7), "Total Set") {
		return false
	}

	return true
}
