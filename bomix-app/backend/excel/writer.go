package excel

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bomix-app/backend/logger"
	"bomix-app/backend/types"
)

// ErrInvalidOutputPath 表示無效或無法寫入的匯出路徑錯誤
var ErrInvalidOutputPath = errors.New("invalid export output path")

// Writer defines the interface for Excel export operations
type Writer interface {
	// ExportExcel exports data to Excel files with the specified options
	ExportExcel(options ExportOptions) ([]string, error)
}

// ExportOptions contains options for exporting to Excel
// See product-spec section 6.5.3
type ExportOptions struct {
	Format              types.BOMFormat
	ProjectIDs          []int64
	RevisionIDs         []string       // BOM Revision IDs (format: "Phase-Version")
	ModelCount          int            // Number of models to export
	Description         string         // Description for BigMatrix
	OutputPath          string         // Output file path for single file export
	OutputDir           string         // Output directory for batch export
	ModelCountOverrides map[string]int // Per-revision model count overrides
	PartData            []PartData     // Part data for export
	Revisions           []RevisionData // Revision metadata for export
}

// PartData represents a single part for export
type PartData struct {
	Item           string
	HHPN           string
	Description    string
	Supplier       string
	SupplierPn     string
	Qty            int
	Location       string
	Type           string
	BOMStatus      string
	CCL            string
	Remark         string
	SecondSources  []SecondSourceData
	Selections     map[string]string // Model -> Supplier PN mapping
}

// SecondSourceData represents second source data for export
type SecondSourceData struct {
	HHPN      string
	Supplier  string
	SupplierPn string
	Description string
}

// RevisionData contains BOM revision metadata for export
type RevisionData struct {
	ID              string
	ProjectCode     string
	Description     string
	SchematicVersion string
	PCBVersion      string
	PCAPN           string
	Phase           string
	Version         string
	Date            string
	Mode            string // NPI or MP
	ModelQty        map[string]int // Model name -> quantity
}

// WriterImpl is the main Excel writer implementation
type WriterImpl struct {
	templateManager *TemplateManager
	logger          *logger.Logger
}

// NewWriter creates a new Excel writer
func NewWriter(lg *logger.Logger) (*WriterImpl, error) {
	tm, err := NewTemplateManager()
	if err != nil {
		return nil, err
	}
	return &WriterImpl{
		templateManager: tm,
		logger:          lg,
	}, nil
}

// ExportExcel exports data to Excel files
func (w *WriterImpl) ExportExcel(options ExportOptions) ([]string, error) {
	fmtStr := strings.TrimSpace(string(options.Format))
	if w.logger != nil {
		w.logger.Info(fmt.Sprintf("[Writer] 準備執行 Excel 匯出 (格式: %s)", fmtStr))
		w.logger.Debug(fmt.Sprintf("[Writer] 匯出選項: RevisionsCount=%d, OutputPath=%s", len(options.RevisionIDs), options.OutputPath))
	}

	switch {
	case strings.EqualFold(fmtStr, string(types.FormatBigMatrix)):
		if w.logger != nil {
			w.logger.Info("[Writer] 比對成功 -> 執行 BigMatrix 匯出")
		}
		return w.exportBigMatrix(options)
	case strings.EqualFold(fmtStr, string(types.FormatMatrix)):
		if w.logger != nil {
			w.logger.Info("[Writer] 比對成功 -> 執行 Matrix 匯出")
		}
		return w.exportMatrix(options)
	default:
		if w.logger != nil {
			w.logger.Error(fmt.Sprintf("[Writer] 不支援的匯出格式: %s", options.Format))
		}
		return nil, fmt.Errorf("%w: unsupported format '%s'", ErrInvalidFormat, options.Format)
	}
}

// generateTimestamp returns formatted date string (YYYYMMDD)
func generateTimestamp() string {
	return time.Now().Format("20060102")
}

// generateBigMatrixFileName generates the output filename for BigMatrix
// Format: {Series Name}_BigMatrix_{phase}_{version}_{date}.xlsx
// Phase/Version are omitted if not consistent across all revisions
func generateBigMatrixFileName(seriesName string, revisions []RevisionData, date string) string {
	// Check if all revisions have the same phase and version
	samePhase := len(revisions) > 0
	sameVersion := len(revisions) > 0
	if len(revisions) > 0 {
		firstPhase := revisions[0].Phase
		firstVersion := revisions[0].Version
		for _, r := range revisions {
			if r.Phase != firstPhase {
				samePhase = false
				break
			}
		}
		for _, r := range revisions {
			if r.Version != firstVersion {
				sameVersion = false
				break
			}
		}
	}

	// Build filename
	parts := []string{seriesName, "BigMatrix"}
	if samePhase {
		parts = append(parts, revisions[0].Phase)
	}
	if samePhase && sameVersion {
		parts = append(parts, revisions[0].Version)
	}
	parts = append(parts, date)

	return fmt.Sprintf("%s.xlsx", strings.Join(parts, "_"))
}

// generateMatrixFileName generates the output filename for Matrix
// Format: {project code}_EZBOM_{phase}_{version}_MatrixBOM_{date}.xlsx
func generateMatrixFileName(rev RevisionData, date string) string {
	return fmt.Sprintf("%s_EZBOM_%s_%s_MatrixBOM_%s.xlsx",
		rev.ProjectCode, rev.Phase, rev.Version, date)
}

// resolveOutputPath 解析並確定 Excel 檔案的最終儲存路徑。
// 若傳入的路徑為目錄，或未指定完整的 .xlsx 副檔名，將自動將其與預設檔名組合。
func resolveOutputPath(outputPath, outputDir, defaultFileName string) string {
	target := strings.TrimSpace(outputPath)
	if target == "" {
		target = strings.TrimSpace(outputDir)
	}

	// 若完全未指定目標路徑，則回傳預設檔名
	if target == "" {
		return defaultFileName
	}

	// 檢查 target 是否為現存的目錄
	fi, err := os.Stat(target)
	isDir := err == nil && fi.IsDir()
	
	// 判斷是否帶有 .xlsx 或 .xls 展延名
	hasXlsxExt := strings.HasSuffix(strings.ToLower(target), ".xlsx") || strings.HasSuffix(strings.ToLower(target), ".xls")

	// 如果目標是現存目錄，或是路徑中無檔名副檔名 (例如 D:\Temp\BOMIX)
	if isDir || !hasXlsxExt {
		if isDir || filepath.Ext(target) == "" {
			return filepath.Join(target, defaultFileName)
		}
		// 若有其他副檔名但非 excel 格式，補上 .xlsx
		return target + ".xlsx"
	}

	return target
}

// validateAndPrepareOutputPath 驗證並準備輸出檔案路徑。
// 寫入檔案前確認目標資料夾路徑是否有效、能否成功創建以及是否具備存取權限。
// 若路徑無效或無法存取，回傳 ErrInvalidOutputPath 與詳細錯誤。
func validateAndPrepareOutputPath(outputPath, outputDir, defaultFileName string) (string, error) {
	finalPath := resolveOutputPath(outputPath, outputDir, defaultFileName)
	dir := filepath.Dir(finalPath)

	if dir != "" && dir != "." {
		// 嘗試建立目標目錄
		if err := os.MkdirAll(dir, 0755); err != nil {
			return finalPath, fmt.Errorf("%w: 無效或無法存取的匯出目錄 '%s': %v", ErrInvalidOutputPath, dir, err)
		}

		// 檢查目錄狀態
		fi, err := os.Stat(dir)
		if err != nil || !fi.IsDir() {
			return finalPath, fmt.Errorf("%w: 目錄不存在或無法存取 '%s'", ErrInvalidOutputPath, dir)
		}
	}

	return finalPath, nil
}

