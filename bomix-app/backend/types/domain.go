package types

// BOMFormat represents the different BOM import/export formats.
// See product-spec section 6.3.1
type BOMFormat string

const (
	FormatEBOM      BOMFormat = "EBOM"
	FormatBigMatrix BOMFormat = "BigMatrix"
	FormatMatrix    BOMFormat = "Matrix"
	FormatUnknown   BOMFormat = "Unknown"
)

// TaskType represents the type of background task.
// See product-spec section 5.1
type TaskType string

const (
	TaskImport   TaskType = "Import"
	TaskExport   TaskType = "Export"
	TaskAnalysis TaskType = "Analysis"
)

// TaskStatus represents the status of a background task.
// See product-spec section 5.1.1
type TaskStatus string

const (
	TaskCreated   TaskStatus = "Created"
	TaskQueued    TaskStatus = "Queued"
	TaskRunning   TaskStatus = "Running"
	TaskCompleted TaskStatus = "Completed"
	TaskFailed    TaskStatus = "Failed"
	TaskCancelled TaskStatus = "Cancelled"
)

// ExportOptions contains options for exporting to Excel.
// See product-spec section 6.5.3
type ExportOptions struct {
	Format              BOMFormat      // BigMatrix / Matrix
	RevisionIDs         []string       // 需匯出的 BOM Revision ID 列表（支援多選）
	Description         string         // 使用者輸入的描述文字（選填，僅適用於 BigMatrix）
	OutputPath          string         // 單一檔案的輸出路徑（單檔匯出時使用，如 BigMatrix）
	OutputDir           string         // 輸出的目標目錄（批次匯出多檔案時使用，如多選 Matrix）
	ModelCountOverrides map[string]int // 各 Revision ID 對應的 Model 匯出數量（僅適用於 BigMatrix）
}

// ImportResult contains the result of an import operation.
// See product-spec section 10.1.5
type ImportResult struct {
	FileName      string
	Format        BOMFormat
	PartsCount    int
	SecondSources int
	SkippedRows   int
	Errors        []string
}

// SecondSourceDTO represents a second source for a part (DTO for frontend).
// See product-spec section 4.7
type SecondSourceDTO struct {
	Hhpn        string `json:"hhpn"`
	Supplier    string `json:"supplier"`
	SupplierPn  string `json:"supplier_pn"`
	Description string `json:"description"`
}

// AggregatedPart represents a merged part from multiple locations (DTO for frontend).
// See product-spec section 4.7
type AggregatedPart struct {
	Item           string            `json:"item"`
	MainSupplier   string            `json:"main_supplier"`
	MainSupplierPn string            `json:"main_supplier_pn"`
	Hhpn           string            `json:"hhpn"`
	Description    string            `json:"description"`
	Type           string            `json:"type"`
	Qty            int               `json:"qty"`
	Locations      string            `json:"locations"` // 逗號分隔
	BOMStatus      string            `json:"bom_status"`
	CCL            string            `json:"ccl"`
	Remark         string            `json:"remark"`
	SecondSources  []SecondSourceDTO `json:"second_sources"`
}

// BOMSummary represents a summary of a BOM revision (DTO for frontend).
// See product-spec section 4.7
type BOMSummary struct {
	TotalParts    int `json:"total_parts"`    // 總零件數
	MainSources   int `json:"main_sources"`   // 主料數量
	SecondSources int `json:"second_sources"` // 替代料數量
	SMDParts      int `json:"smd_parts"`      // SMD 製程零件數
	PTHParts      int `json:"pth_parts"`      // PTH 製程零件數
	BottomParts   int `json:"bottom_parts"`   // BOTTOM 製程零件數
	CriticalParts int `json:"critical_parts"` // 重要零件數
}
