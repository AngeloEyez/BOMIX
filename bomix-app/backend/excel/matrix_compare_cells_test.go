package excel

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestPrintRow5Only(t *testing.T) {
	origPath := filepath.Join("testdata", "TARIS_EZBOM_SI1_0.3_MatrixBOM_20260817_1000.Original.xlsx")
	wpPath := filepath.Join("testdata", "TARIS_EZBOM_SI1_0.3_MatrixBOM_20260817_1000.WP.xlsx")

	origF, _ := excelize.OpenFile(origPath)
	defer origF.Close()
	wpF, _ := excelize.OpenFile(wpPath)
	defer wpF.Close()

	sheet := "SMD"
	r := 5
	for c := 1; c <= 18; c++ {
		colName, _ := excelize.ColumnNumberToName(c)
		cell := fmt.Sprintf("%s%d", colName, r)

		oval, _ := origF.GetCellValue(sheet, cell)
		oraw, _ := origF.GetCellValue(sheet, cell, excelize.Options{RawCellValue: true})
		otype, _ := origF.GetCellType(sheet, cell)
		oform, _ := origF.GetCellFormula(sheet, cell)

		wval, _ := wpF.GetCellValue(sheet, cell)
		wraw, _ := wpF.GetCellValue(sheet, cell, excelize.Options{RawCellValue: true})
		wtype, _ := wpF.GetCellType(sheet, cell)
		wform, _ := wpF.GetCellFormula(sheet, cell)

		t.Logf("[%s5] Orig: val=%q raw=%q type=%v form=%q | WP: val=%q raw=%q type=%v form=%q",
			colName, oval, oraw, otype, oform, wval, wraw, wtype, wform)
	}
}
