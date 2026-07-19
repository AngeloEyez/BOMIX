package types

// ExcelReader defines the interface for reading Excel files.
// This abstraction allows for different Excel library implementations.
type ExcelReader interface {
	// GetSheetNames returns all sheet names in the file.
	GetSheetNames() ([]string, error)

	// ReadSheet reads all data from a specific sheet.
	// Returns a 2D slice where each inner slice represents a row.
	ReadSheet(sheetName string) ([][]string, error)

	// GetCellValue returns the value of a specific cell.
	GetCellValue(sheetName, cell string) (string, error)

	// GetRow returns all cell values in a specific row.
	GetRow(sheetName string, rowIndex int) ([]string, error)

	// Close closes the Excel file.
	Close() error
}

// ExcelWriter defines the interface for writing Excel files.
// This abstraction allows for different Excel library implementations.
type ExcelWriter interface {
	// GetSheetNames returns all sheet names in the file.
	GetSheetNames() ([]string, error)

	// CreateSheet creates a new sheet with the given name.
	CreateSheet(name string) error

	// DeleteSheet deletes an existing sheet.
	DeleteSheet(name string) error

	// SetCellValue sets a single cell value.
	SetCellValue(sheetName, cell string, value interface{}) error

	// SetSheetRow sets a row of values starting from a specific cell.
	SetSheetRow(sheetName, cell string, row interface{}) error

	// SetCellFormula sets a formula in a specific cell.
	SetCellFormula(sheetName, cell, formula string) error

	// SetCellStyle applies a style to a cell.
	SetCellStyle(sheetName, cell string, styleID int) error

	// MergeCells merges a range of cells.
	MergeCells(sheetName string, startCell, endCell string) error

	// SetColumnWidth sets the width of a column.
	SetColumnWidth(sheetName string, column string, width float64) error

	// SaveAs saves the Excel file to the specified path.
	SaveAs(path string) error

	// Close closes the Excel file.
	Close() error
}
