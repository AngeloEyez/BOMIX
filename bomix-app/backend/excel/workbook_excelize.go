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
	rawRows, err := w.f.GetRows(sheet)
	if err != nil {
		return nil, err
	}

	if len(rawRows) == 0 {
		return rawRows, nil
	}

	// 偵測 Row 5 (0-indexed 4) 的欄位數量，預設最少為 12 欄 (涵蓋 A~L 欄)
	targetCols := 12
	if len(rawRows) >= 5 {
		row5Len := len(rawRows[4])
		if row5Len > targetCols {
			targetCols = row5Len
		}
	}

	// 走訪確認是否有任何資料行超過 targetCols
	for _, row := range rawRows {
		if len(row) > targetCols {
			targetCols = len(row)
		}
	}

	// 單次分配並精確補齊每列 Slice 長度
	rows := make([][]string, len(rawRows))
	for i, row := range rawRows {
		padded := make([]string, targetCols)
		copy(padded, row)
		rows[i] = padded
	}

	return rows, nil
}

func (w *ExcelizeWorkbook) Close() error {
	if w.f != nil {
		return w.f.Close()
	}
	return nil
}
