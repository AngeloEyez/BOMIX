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
	db                 *gorm.DB
	detector           *Detector
	logger             *logger.Logger
	progressCb         func(progress float64, message string)
	confirmOverwrite   bool
	confirmOverwriteCb func(projectCode, phase, version string) (bool, error)
}

// NewReader creates a new Excel reader
func NewReader(db *gorm.DB, logger *logger.Logger) *ReaderImpl {
	return &ReaderImpl{
		db:                 db,
		detector:           NewDetector(logger),
		logger:             logger,
		confirmOverwrite:   true, // 預設開啟確認
	}
}

// SetProgressCallback 設定進度回報回調函數
func (r *ReaderImpl) SetProgressCallback(progressCb func(float64, string)) {
	r.progressCb = progressCb
}

// SetConfirmOverwrite 設定是否需在覆蓋前提示確認
func (r *ReaderImpl) SetConfirmOverwrite(confirm bool) {
	r.confirmOverwrite = confirm
}

// SetConfirmOverwriteCallback 設定覆蓋確認回調函數
func (r *ReaderImpl) SetConfirmOverwriteCallback(cb func(projectCode, phase, version string) (bool, error)) {
	r.confirmOverwriteCb = cb
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
		return r.importMatrix(f, path)
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
		db:                 r.db,
		result:             &result,
		filePath:           path,
		logger:             r.logger,
		progressCb:         r.progressCb,
		confirmOverwrite:   r.confirmOverwrite,
		confirmOverwriteCb: r.confirmOverwriteCb,
	}

	err := ebomReader.Import(f)
	return result, err
}

// importBigMatrix 將 BigMatrix 格式的 Excel 檔案委派給 BigMatrixReader 處理。
// BigMatrix 匯入不建立新的 Project/Revision，若找不到已存在的 Revision，
// 會以 WarningError 回傳，讓 Task Manager 將任務狀態設為 TaskWarning。
func (r *ReaderImpl) importBigMatrix(f Workbook, path string) (types.ImportResult, error) {
	result := types.ImportResult{
		FileName: path,
		Format:   types.FormatBigMatrix,
	}

	// 委派給 BigMatrixReader 處理
	bigMatrixReader := &BigMatrixReader{
		db:         r.db,
		result:     &result,
		logger:     r.logger,
		progressCb: r.progressCb,
	}

	err := bigMatrixReader.Import(f)
	return result, err
}

// importMatrix imports a Matrix format file
func (r *ReaderImpl) importMatrix(f Workbook, path string) (types.ImportResult, error) {
	result := types.ImportResult{
		FileName: path,
		Format:   types.FormatMatrix,
	}

	// Delegate to the Matrix reader
	matrixReader := NewMatrixReader(r.db, &result, r.logger)
	matrixReader.progressCb = r.progressCb
	err := matrixReader.Import(f)
	return result, err
}

// ErrInvalidFormat is returned when the file format is invalid or not supported
var ErrInvalidFormat = errors.New("invalid or unsupported file format")
