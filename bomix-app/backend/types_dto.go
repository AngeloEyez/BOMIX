package backend

// SeriesInfo represents the series information for the frontend
type SeriesInfo struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
}

// SeriesInfoWithTime represents series info with last opened time
type SeriesInfoWithTime struct {
	Name       string `json:"name"`
	LastOpened string `json:"lastOpened"`
}

// RecentFile represents a recently opened file
type RecentFile struct {
	Path       string `json:"path"`
	Name       string `json:"name"`
	LastOpened string `json:"lastOpened"`
}

// Project represents a project for the frontend
type Project struct {
	ID          int64  `json:"id"`
	SeriesID    int64  `json:"seriesId"`
	Code        string `json:"code"`
	Description string `json:"description"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// BomRevision represents a BOM revision for the frontend
type BomRevision struct {
	ID               int64  `json:"id"`
	ProjectID        int64  `json:"projectId"`
	Phase            string `json:"phase"`
	Version          string `json:"version"`
	Description      string `json:"description"`
	SchematicVersion string `json:"schematicVersion"`
	PCBVersion       string `json:"pcbVersion"`
	PCAPN            string `json:"pcaPn"`
	Date             string `json:"date"`
	Mode             string `json:"mode"`
	SourceFile       string `json:"sourceFile"`
	CreatedAt        string `json:"createdAt"`
	UpdatedAt        string `json:"updatedAt"`
}

// Part represents a part for the frontend
type Part struct {
	ID          int64  `json:"id"`
	RevisionID  int64  `json:"revisionId"`
	Type        string `json:"type"`
	Supplier    string `json:"supplier"`
	SupplierPn  string `json:"supplierPn"`
	Description string `json:"description"`
	Location    string `json:"location"`
	Quantity    int    `json:"quantity"`
	Cost        float64 `json:"cost"`
	BOMStatus   string `json:"bomStatus"`
	CCL         string `json:"ccl"`
	Remark      string `json:"remark"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// SecondSource represents a second source for the frontend
type SecondSource struct {
	ID          int64  `json:"id"`
	RevisionID  int64  `json:"revisionId"`
	PartID      int64  `json:"partId"`
	Supplier    string `json:"supplier"`
	SupplierPn  string `json:"supplierPn"`
	Description string `json:"description"`
	Cost        float64 `json:"cost"`
	LeadTime    int    `json:"leadTime"`
	IsActive    bool   `json:"isActive"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// MatrixModel represents a matrix model for the frontend
type MatrixModel struct {
	ID        int64  `json:"id"`
	RevisionID int64  `json:"revisionId"`
	ModelName  string `json:"modelName"`
	Qty       int    `json:"qty"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// MatrixSelection represents a matrix selection for the frontend
type MatrixSelection struct {
	ID                 int64  `json:"id"`
	RevisionID         int64  `json:"revisionId"`
	ModelID            int64  `json:"modelId"`
	PartID             int64  `json:"partId"`
	Group              string `json:"group"`
	Material           string `json:"material"`
	SelectedSupplier   string `json:"selectedSupplier"`
	SelectedSupplierPn string `json:"selectedSupplierPn"`
	IsAutoSelected     bool   `json:"isAutoSelected"`
	CreatedAt          string `json:"createdAt"`
	UpdatedAt          string `json:"updatedAt"`
}

// Task represents a task for the frontend
type Task struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	Status    string  `json:"status"`
	Progress  float64 `json:"progress"`
	Message   string  `json:"message"`
	Error     string  `json:"error,omitempty"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt string  `json:"updatedAt"`
}

// ImportResult represents an import result for the frontend
type ImportResult struct {
	FileName   string `json:"fileName"`
	Format     string `json:"format"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	PartsCount int    `json:"partsCount"`
	Error      string `json:"error,omitempty"`
	TaskID     string `json:"taskID,omitempty"`
}

// ExportOptions represents export options for the frontend
type ExportOptions struct {
	Format              string            `json:"format"`
	ProjectIDs          []int64           `json:"projectIds"`
	RevisionIDs         []int64           `json:"revisionIds"`
	Description         string            `json:"description,omitempty"`
	OutputPath          string            `json:"outputPath,omitempty"`
	OutputDir           string            `json:"outputDir,omitempty"`
	ModelCountOverrides map[string]int    `json:"modelCountOverrides,omitempty"`
}

// LogEntry represents a log entry for the frontend
type LogEntry struct {
	ID        string            `json:"id,omitempty"`
	Level     string            `json:"level"`
	Message   string            `json:"message"`
	Timestamp string            `json:"timestamp"`
	Attrs     map[string]string `json:"attrs,omitempty"`
}

// Settings represents the application settings for the frontend
type Settings struct {
	Theme                   string             `json:"theme"`
	Import                  *ImportSettings    `json:"import"`
	Logger                  *LoggerSettings    `json:"logger"`
	RecentFiles             *RecentFilesSettings `json:"recentFiles"`
	AutoOpenLastFile        bool               `json:"autoOpenLastFile"`
	LastOpenedFile          string             `json:"lastOpenedFile"`
	AutoImportPreviousMatrix bool              `json:"autoImportPreviousMatrix"`
}

// ImportSettings represents import settings for the frontend
type ImportSettings struct {
	ConfirmOverwrite       bool `json:"confirmOverwrite"`
	AutoImportPreviousMatrix bool `json:"autoImportPreviousMatrix"`
}

// LoggerSettings represents logger settings for the frontend
type LoggerSettings struct {
	Level      string `json:"level"`
	MaxEntries int    `json:"maxEntries"`
}

// RecentFilesSettings represents recent files settings for the frontend
type RecentFilesSettings struct {
	MaxRecentFiles int      `json:"maxRecentFiles"`
	RecentFiles    []string `json:"recentFiles"`
}
