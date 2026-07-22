package excel

import (
	"errors"
	"strings"

	"bomix-app/backend/logger"
	"bomix-app/backend/types"
)

// Detector detects the format of a BOM Excel file
// See product-spec section 6.3.1
type Detector struct {
	logger *logger.Logger
}

// NewDetector creates a new detector instance
func NewDetector(loggers ...*logger.Logger) *Detector {
	var lg *logger.Logger
	if len(loggers) > 0 {
		lg = loggers[0]
	}
	return &Detector{logger: lg}
}

// Detect detects the BOM format from an Excel file
// Returns FormatEBOM, FormatBigMatrix, FormatMatrix, or FormatUnknown
// See product-spec section 6.3.1 格式辨識與偵測規則對照表
func (d *Detector) Detect(f Workbook) (types.BOMFormat, error) {
	sheets := f.GetSheetList()

	if d.logger != nil {
		d.logger.Debug("開始偵測 Excel BOM 格式", "sheetCount", len(sheets), "sheets", sheets)
	}

	if len(sheets) == 0 {
		return types.FormatUnknown, errors.New("no sheets found")
	}

	// Check for BigMatrix format: exists sheet named "BigMatrix"
	if d.hasSheet(sheets, "BigMatrix") {
		if d.logger != nil {
			d.logger.Debug("偵測結果: BigMatrix")
		}
		return types.FormatBigMatrix, nil
	}

	// Check for EBOM format: has SMD/SMT, PTH, BOTTOM sheets
	// AND SMD/PTH/BOTTOM header cells match specific patterns
	if d.isEBOMFormat(f, sheets) {
		if d.logger != nil {
			d.logger.Debug("偵測結果: EBOM")
		}
		return types.FormatEBOM, nil
	}

	// Check for Matrix format: has SMD sheet with specific header pattern
	if d.isMatrixFormat(f, sheets) {
		if d.logger != nil {
			d.logger.Debug("偵測結果: Matrix")
		}
		return types.FormatMatrix, nil
	}

	if d.logger != nil {
		d.logger.Debug("偵測結果: FormatUnknown (不符合已知 EBOM/BigMatrix/Matrix 規則)")
	}

	return types.FormatUnknown, nil
}

// hasSheet checks if a sheet with the given name exists (case-insensitive & trimmed)
func (d *Detector) hasSheet(sheets []string, name string) bool {
	for _, sheet := range sheets {
		if strings.EqualFold(strings.TrimSpace(sheet), name) {
			return true
		}
	}
	return false
}

// isEBOMFormat checks if the file matches EBOM format
// Conditions:
// 1. Contains SMD, PTH, BOTTOM sheets
// 2. SMD header H5="Qty" and J7="CCL"
func (d *Detector) isEBOMFormat(f Workbook, sheets []string) bool {
	smdSheet := ""
	pthSheet := ""
	bottomSheet := ""

	for _, sheet := range sheets {
		trimmed := strings.TrimSpace(sheet)
		if smdSheet == "" && strings.EqualFold(trimmed, "SMD") {
			smdSheet = sheet
		}
		if pthSheet == "" && strings.EqualFold(trimmed, "PTH") {
			pthSheet = sheet
		}
		if bottomSheet == "" && strings.EqualFold(trimmed, "BOTTOM") {
			bottomSheet = sheet
		}
	}

	if d.logger != nil {
		d.logger.Debug("EBOM 工作表比對結果",
			"SMD", smdSheet,
			"PTH", pthSheet,
			"BOTTOM", bottomSheet,
		)
	}

	// Need at least SMD sheet
	if smdSheet == "" {
		if d.logger != nil {
			d.logger.Debug("EBOM 判定不符: 未找到 SMD 工作表")
		}
		return false
	}

	// Check header patterns in SMD sheet
	valH5, _ := f.GetCellValue(smdSheet, "H5")
	valJ7, _ := f.GetCellValue(smdSheet, "J7")

	trimmedH5 := strings.TrimSpace(valH5)
	trimmedJ7 := strings.TrimSpace(valJ7)

	if d.logger != nil {
		d.logger.Debug("EBOM 表頭儲存格比對",
			"sheet", smdSheet,
			"H5_raw", valH5,
			"H5_trimmed", trimmedH5,
			"J7_raw", valJ7,
			"J7_trimmed", trimmedJ7,
		)
	}

	// Standard check: H5 = "Qty", J7 = "CCL"
	if strings.EqualFold(trimmedH5, "Qty") && strings.EqualFold(trimmedJ7, "CCL") {
		return true
	}

	// Compatibility fallback: If SMD and at least one of (PTH, BOTTOM) exist, and H5 is not "Location" (Matrix header)
	if (pthSheet != "" || bottomSheet != "") && !strings.EqualFold(trimmedH5, "Location") {
		if d.logger != nil {
			d.logger.Debug("EBOM 觸發相容模式 (包含 SMD 且有 PTH/BOTTOM 工作表，表頭非 Matrix)")
		}
		return true
	}

	return false
}

// isMatrixFormat checks if the file matches Matrix format
// Conditions:
// 1. Has SMD sheet
// 2. SMD header H5="Location" and J7="Total Set"
func (d *Detector) isMatrixFormat(f Workbook, sheets []string) bool {
	smdSheet := ""
	for _, sheet := range sheets {
		trimmed := strings.TrimSpace(sheet)
		if strings.EqualFold(trimmed, "SMD") {
			smdSheet = sheet
			break
		}
	}

	if smdSheet == "" {
		return false
	}

	// Check H5 (Location)
	valH5, _ := f.GetCellValue(smdSheet, "H5")
	trimmedH5 := strings.TrimSpace(valH5)
	if !strings.EqualFold(trimmedH5, "Location") {
		return false
	}

	// Check J7 (Total Set)
	valJ7, _ := f.GetCellValue(smdSheet, "J7")
	trimmedJ7 := strings.TrimSpace(valJ7)
	if !strings.EqualFold(trimmedJ7, "Total Set") {
		return false
	}

	return true
}
