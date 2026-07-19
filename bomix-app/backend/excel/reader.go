package excel

import (
	"errors"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
	"bomix-app/backend/types"
)

// Reader defines the interface for Excel import operations
type Reader interface {
	// ImportExcel imports one or more Excel files and returns the results
	ImportExcel(filePaths []string) ([]types.ImportResult, error)
}

// ReaderImpl is the main Excel reader implementation
type ReaderImpl struct {
	db     *gorm.DB
	detector *Detector
}

// NewReader creates a new Excel reader
func NewReader(db *gorm.DB) *ReaderImpl {
	return &ReaderImpl{
		db:       db,
		detector: NewDetector(),
	}
}

// ImportExcel imports multiple Excel files
// See product-spec section 6.3.2 匯入流程
func (r *ReaderImpl) ImportExcel(filePaths []string) ([]types.ImportResult, error) {
	results := make([]types.ImportResult, 0, len(filePaths))

	for _, path := range filePaths {
		result, err := r.importFile(path)
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
		}
		results = append(results, result)
	}

	return results, nil
}

// importFile imports a single Excel file
func (r *ReaderImpl) importFile(path string) (types.ImportResult, error) {
	result := types.ImportResult{
		FileName: path,
	}

	// Open the file
	f, err := excelize.OpenFile(path)
	if err != nil {
		return result, err
	}
	defer f.Close()

	// Detect format
	format, err := r.detector.Detect(f)
	if err != nil {
		return result, err
	}
	result.Format = format

	// Process based on format
	switch format {
	case types.FormatEBOM:
		return r.importEBOM(f, path)
	case types.FormatBigMatrix:
		return r.importBigMatrix(f, path)
	case types.FormatMatrix:
		// Matrix format is not supported yet
		return result, ErrInvalidFormat
	default:
		return result, ErrInvalidFormat
	}
}

// importEBOM imports an EBOM format file
func (r *ReaderImpl) importEBOM(f *excelize.File, path string) (types.ImportResult, error) {
	result := types.ImportResult{
		FileName: path,
		Format:   types.FormatEBOM,
	}

	// Delegate to the EBOM reader
	ebomReader := &EBOMReader{
		db:     r.db,
		result: &result,
	}

	err := ebomReader.Import(f)
	return result, err
}

// importBigMatrix imports a BigMatrix format file
func (r *ReaderImpl) importBigMatrix(f *excelize.File, path string) (types.ImportResult, error) {
	result := types.ImportResult{
		FileName: path,
		Format:   types.FormatBigMatrix,
	}

	// Delegate to the BigMatrix reader
	bigMatrixReader := &BigMatrixReader{
		db:     r.db,
		result: &result,
	}

	err := bigMatrixReader.Import(f)
	return result, err
}

// importMatrix imports a Matrix format file (placeholder)
func (r *ReaderImpl) importMatrix(f *excelize.File, path string) (types.ImportResult, error) {
	return types.ImportResult{
		FileName: path,
		Format:   types.FormatMatrix,
	}, ErrInvalidFormat
}

// ErrInvalidFormat is returned when the file format is invalid or not supported
var ErrInvalidFormat = errors.New("invalid or unsupported file format")
