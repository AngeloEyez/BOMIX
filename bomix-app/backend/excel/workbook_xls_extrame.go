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
	
	// Determine the maximum column across all rows to pad slices correctly
	maxColOverall := 0
	for i := 0; i <= maxRow; i++ {
		row := targetSheet.Row(i)
		if row != nil {
			lastCol := row.LastCol()
			if lastCol > maxColOverall {
				maxColOverall = lastCol
			}
		}
	}

	for i := 0; i <= maxRow; i++ {
		row := targetSheet.Row(i)
		if row == nil {
			rows = append(rows, []string{})
			continue
		}

		var rowData []string
		for j := 0; j < maxColOverall; j++ {
			rowData = append(rowData, row.Col(j))
		}
		
		// Trim trailing empty strings to match excelize behavior somewhat
		lastNonEmpty := -1
		for j := len(rowData) - 1; j >= 0; j-- {
			if rowData[j] != "" {
				lastNonEmpty = j
				break
			}
		}
		if lastNonEmpty >= 0 {
			rowData = rowData[:lastNonEmpty+1]
		} else {
			rowData = []string{}
		}

		rows = append(rows, rowData)
	}

	return rows, nil
}

func (w *ExtrameXlsWorkbook) Close() error {
	// extrame/xls does not have a Close method for workbook since it maps into memory or reads at once
	return nil
}
