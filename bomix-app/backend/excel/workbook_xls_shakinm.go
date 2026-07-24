package excel

import (
	"fmt"

	"github.com/shakinm/xlsReader/xls"
	"github.com/xuri/excelize/v2"
)

type ShakinmXlsWorkbook struct {
	f xls.Workbook
}

func newShakinmXlsWorkbook(filePath string) (wb Workbook, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("shakinm/xlsReader panic: %v", r)
		}
	}()

	f, err := xls.OpenFile(filePath)
	if err != nil {
		return nil, err
	}
	return &ShakinmXlsWorkbook{f: f}, nil
}

func (w *ShakinmXlsWorkbook) GetSheetList() (sheets []string) {
	defer func() {
		if r := recover(); r != nil {
			sheets = nil
		}
	}()

	for i := 0; i < w.f.GetNumberSheets(); i++ {
		sheet, err := w.f.GetSheet(i)
		if err == nil && sheet != nil {
			sheets = append(sheets, sheet.GetName())
		}
	}
	return sheets
}

func (w *ShakinmXlsWorkbook) GetCellValue(sheetName, axis string) (val string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("shakinm/xlsReader GetCellValue panic: %v", r)
		}
	}()

	col, row, err := excelize.CellNameToCoordinates(axis)
	if err != nil {
		return "", err
	}

	targetRow := row - 1
	targetCol := col - 1

	var targetSheet *xls.Sheet
	for i := 0; i < w.f.GetNumberSheets(); i++ {
		s, err := w.f.GetSheet(i)
		if err == nil && s != nil && s.GetName() == sheetName {
			targetSheet = s
			break
		}
	}

	if targetSheet == nil {
		return "", fmt.Errorf("sheet %s not found", sheetName)
	}

	if targetRow >= targetSheet.GetNumberRows() {
		return "", nil // empty cell
	}

	r, err := targetSheet.GetRow(targetRow)
	if err != nil || r == nil {
		return "", nil
	}

	cell, err := r.GetCol(targetCol)
	if err != nil || cell == nil {
		return "", nil
	}

	return cell.GetString(), nil
}

func (w *ShakinmXlsWorkbook) GetRows(sheetName string) (rows [][]string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("shakinm/xlsReader GetRows panic: %v", r)
		}
	}()

	var targetSheet *xls.Sheet
	for i := 0; i < w.f.GetNumberSheets(); i++ {
		s, err := w.f.GetSheet(i)
		if err == nil && s != nil && s.GetName() == sheetName {
			targetSheet = s
			break
		}
	}

	if targetSheet == nil {
		return nil, fmt.Errorf("sheet %s not found", sheetName)
	}

	maxRow := targetSheet.GetNumberRows()

	// 先觀察 Row 5 (index 4) 決定基準欄位數，預設最少 12
	targetCols := 12
	if maxRow >= 5 {
		row5, err := targetSheet.GetRow(4)
		if err == nil && row5 != nil {
			r5Cols := len(row5.GetCols())
			if r5Cols > targetCols {
				targetCols = r5Cols
			}
		}
	}

	rows = make([][]string, 0, maxRow)
	for i := 0; i < maxRow; i++ {
		r, err := targetSheet.GetRow(i)
		if err != nil || r == nil {
			rows = append(rows, make([]string, targetCols))
			continue
		}

		colCount := len(r.GetCols())
		if colCount > targetCols {
			targetCols = colCount
		}

		rowData := make([]string, targetCols)
		for j := 0; j < targetCols; j++ {
			cell, err := r.GetCol(j)
			if err == nil && cell != nil {
				rowData[j] = cell.GetString()
			}
		}

		rows = append(rows, rowData)
	}

	for i := range rows {
		if len(rows[i]) < targetCols {
			padded := make([]string, targetCols)
			copy(padded, rows[i])
			rows[i] = padded
		}
	}

	return rows, nil
}

func (w *ShakinmXlsWorkbook) Close() error {
	return nil
}
