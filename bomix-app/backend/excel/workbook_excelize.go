package excel

import (
	"github.com/xuri/excelize/v2"
)

type ExcelizeWorkbook struct {
	f *excelize.File
}

func newExcelizeWorkbook(filePath string) (Workbook, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, err
	}
	return &ExcelizeWorkbook{f: f}, nil
}

func (w *ExcelizeWorkbook) GetSheetList() []string {
	return w.f.GetSheetList()
}

func (w *ExcelizeWorkbook) GetCellValue(sheet, axis string) (string, error) {
	return w.f.GetCellValue(sheet, axis)
}

func (w *ExcelizeWorkbook) GetRows(sheet string) ([][]string, error) {
	return w.f.GetRows(sheet)
}

func (w *ExcelizeWorkbook) Close() error {
	if w.f != nil {
		return w.f.Close()
	}
	return nil
}
