package types

import "errors"

// Sentinel errors for BOMIX application.
// See product-spec section 10.3

// Import-related errors
var (
	ErrInvalidFileFormat     = errors.New("invalid file format")
	ErrUnsupportedFormat     = errors.New("unsupported file format")
	ErrFileNotFound          = errors.New("file not found")
	ErrInvalidBOMStructure   = errors.New("invalid BOM structure")
	ErrMissingRequiredField  = errors.New("missing required field")
	ErrInvalidSheetName      = errors.New("invalid sheet name")
	ErrInvalidHeaderFormat   = errors.New("invalid header format")
	ErrInvalidCellData       = errors.New("invalid cell data")
)

// Database-related errors
var (
	ErrDatabaseClosed   = errors.New("database connection is closed")
	ErrDatabaseOpen     = errors.New("failed to open database")
	ErrSeriesNotFound   = errors.New("series not found")
	ErrProjectNotFound  = errors.New("project not found")
	ErrRevisionNotFound = errors.New("revision not found")
	ErrPartNotFound     = errors.New("part not found")
	ErrMatrixModelNotFound = errors.New("matrix model not found")
)

// Task-related errors
var (
	ErrTaskNotFound    = errors.New("task not found")
	ErrTaskCancelled   = errors.New("task was cancelled")
	ErrTaskNotRunning  = errors.New("task is not running")
	ErrInvalidTaskType = errors.New("invalid task type")
)

// Export-related errors
var (
	ErrInvalidExportOptions = errors.New("invalid export options")
	ErrTemplateNotFound     = errors.New("template not found")
	ErrExportFailed         = errors.New("export failed")
)

// Matrix-related errors
var (
	ErrInvalidMatrixSelection = errors.New("invalid matrix selection")
	ErrInvalidModelName       = errors.New("invalid model name")
	ErrInvalidQty             = errors.New("invalid quantity - must be positive integer")
)

// Configuration errors
var (
	ErrConfigNotFound = errors.New("configuration file not found")
	ErrConfigInvalid  = errors.New("invalid configuration")
)

// General errors
var (
	ErrUnauthorized    = errors.New("unauthorized operation")
	ErrInternal        = errors.New("internal server error")
	ErrNotImplemented  = errors.New("not implemented")
	ErrInvalidArgument = errors.New("invalid argument")
)
