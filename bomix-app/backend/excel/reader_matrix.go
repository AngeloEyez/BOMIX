package excel

// MatrixReader handles Matrix format import (placeholder)
// See product-spec section 7.3 - Not supported in Wave 1
type MatrixReader struct{}

// NewMatrixReader creates a new MatrixReader instance
func NewMatrixReader() *MatrixReader {
	return &MatrixReader{}
}

// Import imports a Matrix format file
// This is a placeholder that returns ErrInvalidFormat
func (r *MatrixReader) Import(f Workbook) error {
	return ErrInvalidFormat
}
