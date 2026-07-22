package excel

import (
	"fmt"
	"path/filepath"
	"strings"

	"bomix-app/backend/logger"
)

// Workbook defines a common interface for reading Excel files (.xlsx, .xls)
type Workbook interface {
	GetSheetList() []string
	GetCellValue(sheet, axis string) (string, error)
	GetRows(sheet string) ([][]string, error)
	Close() error
}

// OpenWorkbook opens an Excel file and returns a Workbook interface,
// handling format detection and multi-library fallback for .xls files.
func OpenWorkbook(filePath string, l *logger.Logger) (Workbook, error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	if ext == ".xlsx" || ext == ".xlsm" || ext == ".xltx" {
		if l != nil {
			l.Debug("使用 excelize 開啟 .xlsx 檔案", "file", filePath)
		}
		wb, err := newExcelizeWorkbook(filePath)
		return wb, err
	} else if ext == ".xls" {
		if l != nil {
			l.Debug("嘗試使用 extrame/xls 開啟 .xls 檔案", "file", filePath)
		}
		wb, err := newExtrameXlsWorkbook(filePath)
		if err == nil {
			return wb, nil
		}

		if l != nil {
			l.Warn("extrame/xls 解析失敗，進行 Fallback，嘗試使用 shakinm/xlsReader 開啟", "file", filePath, "error", err)
		}

		wb2, err2 := newShakinmXlsWorkbook(filePath)
		if err2 == nil {
			return wb2, nil
		}
		return nil, fmt.Errorf("所有 .xls 讀取套件皆失敗: %v", err2)
	}

	return nil, fmt.Errorf("不支援的檔案格式: %s", ext)
}
