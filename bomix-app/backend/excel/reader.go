package excel

import (
	"errors"
	"fmt"

	"bomix-app/backend/logger"
	"bomix-app/backend/types"

	"gorm.io/gorm"
)

// Reader defines the interface for Excel import operations
type Reader interface {
	// ImportExcel imports one or more Excel files and returns the results
	ImportExcel(filePaths []string) ([]types.ImportResult, error)
}

// ReaderImpl is the main Excel reader implementation
type ReaderImpl struct {
	db       *gorm.DB
	detector *Detector
	logger   *logger.Logger
}

// NewReader creates a new Excel reader
func NewReader(db *gorm.DB, logger *logger.Logger) *ReaderImpl {
	return &ReaderImpl{
		db:       db,
		detector: NewDetector(),
		logger:   logger,
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

	// Open the file using our unified workbook reader
	f, err := OpenWorkbook(path, r.logger)
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

	if r.logger != nil {
		r.logger.Info(fmt.Sprintf("判斷是 %s BOM", format), "file", path)
	}

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
func (r *ReaderImpl) importEBOM(f Workbook, path string) (types.ImportResult, error) {
	result := types.ImportResult{
		FileName: path,
		Format:   types.FormatEBOM,
	}

	// Delegate to the EBOM reader
	ebomReader := &EBOMReader{
		db:     r.db,
		result: &result,
		logger: r.logger,
	}

	err := ebomReader.Import(f)
	return result, err
}

// importBigMatrix imports a BigMatrix format file
func (r *ReaderImpl) importBigMatrix(f Workbook, path string) (types.ImportResult, error) {
	result := types.ImportResult{
		FileName: path,
		Format:   types.FormatBigMatrix,
	}

	// Delegate to the BigMatrix reader
	bigMatrixReader := &BigMatrixReader{
		db:     r.db,
		result: &result,
		logger: r.logger,
	}

	err := bigMatrixReader.Import(f)
	return result, err
}

// importMatrix imports a Matrix format file (placeholder)
func (r *ReaderImpl) importMatrix(f Workbook, path string) (types.ImportResult, error) {
	return types.ImportResult{
		FileName: path,
		Format:   types.FormatMatrix,
	}, ErrInvalidFormat
}

// ErrInvalidFormat is returned when the file format is invalid or not supported
var ErrInvalidFormat = errors.New("invalid or unsupported file format")
