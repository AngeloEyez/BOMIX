package excel

import (
	"fmt"
	"strings"
	"time"

	"bomix-app/backend/logger"
	"bomix-app/backend/types"
)

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
func NewWriter(loggers ...*logger.Logger) (*WriterImpl, error) {
	var lg *logger.Logger
	if len(loggers) > 0 {
		lg = loggers[0]
	}
	tm, err := NewTemplateManager()
	if err != nil {
		return nil, err
	}
	return &WriterImpl{
		templateManager: tm,
		logger:          lg,
	}, nil
}

func (w *WriterImpl) logInfo(msg string, attrs ...any) {
	if w.logger != nil {
		w.logger.Info(msg, attrs...)
	}
}

func (w *WriterImpl) logDebug(msg string, attrs ...any) {
	if w.logger != nil {
		w.logger.Debug(msg, attrs...)
	}
}

func (w *WriterImpl) logError(msg string, attrs ...any) {
	if w.logger != nil {
		w.logger.Error(msg, attrs...)
	}
}

// ExportExcel exports data to Excel files
func (w *WriterImpl) ExportExcel(options ExportOptions) ([]string, error) {
	fmtStr := strings.TrimSpace(string(options.Format))
	w.logInfo(fmt.Sprintf("[Writer] 準備執行 Excel 匯出 (格式: %s)", fmtStr))
	w.logDebug(fmt.Sprintf("[Writer] 匯出選項: RevisionsCount=%d, OutputPath=%s", len(options.RevisionIDs), options.OutputPath))

	switch {
	case strings.EqualFold(fmtStr, string(types.FormatBigMatrix)):
		w.logInfo("[Writer] 比對成功 -> 執行 BigMatrix 匯出")
		return w.exportBigMatrix(options)
	case strings.EqualFold(fmtStr, string(types.FormatMatrix)):
		w.logInfo("[Writer] 比對成功 -> 執行 Matrix 匯出")
		return w.exportMatrix(options)
	default:
		w.logError(fmt.Sprintf("[Writer] 不支援的匯出格式: %s", options.Format))
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
	samePhase := true
	sameVersion := true
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
