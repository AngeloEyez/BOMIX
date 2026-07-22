package excel

import (
	"fmt"
	"path/filepath"
	"strings"
)

// WorkbookLog represents a log entry generated during workbook operations
type WorkbookLog struct {
	Level   string // "DEBUG", "INFO", "WARN", "ERROR"
	Message string
	Args    []any
}

// Workbook defines a common interface for reading Excel files (.xlsx, .xls)
type Workbook interface {
	GetSheetList() []string
	GetCellValue(sheet, axis string) (string, error)
	GetRows(sheet string) ([][]string, error)
	Close() error
}

// OpenWorkbook opens an Excel file and returns a Workbook interface,
// handling format detection and multi-library fallback for .xls files.
// It also returns a slice of WorkbookLog containing diagnostic information.
func OpenWorkbook(filePath string) (Workbook, []WorkbookLog, error) {
	var logs []WorkbookLog
	ext := strings.ToLower(filepath.Ext(filePath))

	if ext == ".xlsx" || ext == ".xlsm" || ext == ".xltx" {
		logs = append(logs, WorkbookLog{Level: "DEBUG", Message: "使用 excelize 開啟 .xlsx 檔案", Args: []any{"file", filePath}})
		wb, err := newExcelizeWorkbook(filePath)
		return wb, logs, err
	} else if ext == ".xls" {
		logs = append(logs, WorkbookLog{Level: "DEBUG", Message: "嘗試使用 extrame/xls 開啟 .xls 檔案", Args: []any{"file", filePath}})
		wb, err := newExtrameXlsWorkbook(filePath)
		if err == nil {
			return wb, logs, nil
		}

		logs = append(logs, WorkbookLog{Level: "WARN", Message: "extrame/xls 解析失敗，進行 Fallback，嘗試使用 shakinm/xlsReader 開啟", Args: []any{"file", filePath, "error", err}})

		wb2, err2 := newShakinmXlsWorkbook(filePath)
		if err2 == nil {
			return wb2, logs, nil
		}
		return nil, logs, fmt.Errorf("所有 .xls 讀取套件皆失敗: %v", err2)
	}

	return nil, logs, fmt.Errorf("不支援的檔案格式: %s", ext)
}
