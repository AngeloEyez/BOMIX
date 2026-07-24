package excel

import (
	"fmt"

	"github.com/extrame/xls"
	"github.com/xuri/excelize/v2"
)

type ExtrameXlsWorkbook struct {
	f *xls.WorkBook
}

func newExtrameXlsWorkbook(filePath string) (wb Workbook, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("extrame/xls panic: %v", r)
		}
	}()

	f, err := xls.Open(filePath, "utf-8")
	if err != nil {
		return nil, err
	}
	return &ExtrameXlsWorkbook{f: f}, nil
}

func (w *ExtrameXlsWorkbook) GetSheetList() (sheets []string) {
	defer func() {
		if r := recover(); r != nil {
			sheets = nil
		}
	}()

	for i := 0; i < w.f.NumSheets(); i++ {
		sheet := w.f.GetSheet(i)
		if sheet != nil {
			sheets = append(sheets, sheet.Name)
		}
	}
	return sheets
}

func (w *ExtrameXlsWorkbook) GetCellValue(sheetName, axis string) (val string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("extrame/xls GetCellValue panic: %v", r)
		}
	}()

	col, row, err := excelize.CellNameToCoordinates(axis)
	if err != nil {
		return "", err
	}

	// excelize returns 1-based index (A1 -> col=1, row=1)
	// extrame/xls uses 0-based index
	targetRow := row - 1
	targetCol := col - 1

	var targetSheet *xls.WorkSheet
	for i := 0; i < w.f.NumSheets(); i++ {
		s := w.f.GetSheet(i)
		if s != nil && s.Name == sheetName {
			targetSheet = s
			break
		}
	}

	if targetSheet == nil {
		return "", fmt.Errorf("sheet %s not found", sheetName)
	}

	if targetRow > int(targetSheet.MaxRow) {
		return "", nil // empty cell
	}

	r := targetSheet.Row(targetRow)
	if r == nil {
		return "", nil
	}
	return r.Col(targetCol), nil
}

func (w *ExtrameXlsWorkbook) GetRows(sheetName string) (rows [][]string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("extrame/xls GetRows panic: %v", r)
		}
	}()

	var targetSheet *xls.WorkSheet
	for i := 0; i < w.f.NumSheets(); i++ {
		s := w.f.GetSheet(i)
		if s != nil && s.Name == sheetName {
			targetSheet = s
			break
		}
	}

	if targetSheet == nil {
		return nil, fmt.Errorf("sheet %s not found", sheetName)
	}

	maxRow := int(targetSheet.MaxRow)

	// 先觀察 Row 5 (index 4) 的 LastCol() 決定基準欄位數，預設最少 12
	targetCols := 12
	if maxRow >= 4 {
		row5 := targetSheet.Row(4)
		if row5 != nil {
			r5Last := row5.LastCol()
			if r5Last > targetCols {
				targetCols = r5Last
			}
		}
	}

	// 走訪全頁一次性填入，若某 Row 超過 targetCols 則即時動態調大
	rows = make([][]string, 0, maxRow+1)
	for i := 0; i <= maxRow; i++ {
		row := targetSheet.Row(i)
		if row == nil {
			rows = append(rows, make([]string, targetCols))
			continue
		}

		lastCol := row.LastCol()
		if lastCol > targetCols {
			targetCols = lastCol
		}

		rowData := make([]string, targetCols)
		for j := 0; j < targetCols; j++ {
			rowData[j] = row.Col(j)
		}
		rows = append(rows, rowData)
	}

	// 若在過程中 targetCols 被擴展，將先前的 rowData 補齊
	for i := range rows {
		if len(rows[i]) < targetCols {
			padded := make([]string, targetCols)
			copy(padded, rows[i])
			rows[i] = padded
		}
	}

	return rows, nil
}

func (w *ExtrameXlsWorkbook) Close() error {
	// extrame/xls does not have a Close method for workbook since it maps into memory or reads at once
	return nil
}
