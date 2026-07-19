package db

import "errors"

// Database errors
var (
	ErrSeriesNotFound      = errors.New("series not found")
	ErrProjectNotFound     = errors.New("project not found")
	ErrRevisionNotFound    = errors.New("revision not found")
	ErrPartNotFound        = errors.New("part not found")
	ErrSecondSourceNotFound = errors.New("second source not found")
	ErrMatrixModelNotFound  = errors.New("matrix model not found")
	ErrMatrixSelectionNotFound = errors.New("matrix selection not found")
)

// Import errors
var (
	ErrInvalidFormat = errors.New("invalid format")
)
