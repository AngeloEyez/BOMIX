package db

import "errors"

// Database errors
var (
	ErrSeriesNotFound          = errors.New("series not found")
	ErrProjectNotFound         = errors.New("project not found")
	ErrRevisionNotFound        = errors.New("revision not found")
	ErrMaterialNotFound        = errors.New("material not found")
	ErrComponentNotFound       = errors.New("component not found")
	ErrPartLocationNotFound    = errors.New("part location not found")
	ErrMatrixModelNotFound     = errors.New("matrix model not found")
	ErrMatrixSelectionNotFound = errors.New("matrix selection not found")
)

// Import errors
var (
	ErrInvalidFormat = errors.New("invalid format")
)
