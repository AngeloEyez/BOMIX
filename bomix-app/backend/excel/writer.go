package excel

import (
	"fmt"
	"strings"
	"time"

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
}

// NewWriter creates a new Excel writer
func NewWriter() (*WriterImpl, error) {
	tm, err := NewTemplateManager()
	if err != nil {
		return nil, err
	}
	return &WriterImpl{
		templateManager: tm,
	}, nil
}

// ExportExcel exports data to Excel files
func (w *WriterImpl) ExportExcel(options ExportOptions) ([]string, error) {
	switch options.Format {
	case types.FormatBigMatrix:
		return w.exportBigMatrix(options)
	case types.FormatMatrix:
		return w.exportMatrix(options)
	default:
		return nil, ErrInvalidFormat
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
