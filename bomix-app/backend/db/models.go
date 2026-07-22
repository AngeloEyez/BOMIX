package db

import (
	"time"

	"gorm.io/gorm"
)

// Series represents a product series
// Table: series
type Series struct {
	ID          int64          `gorm:"primaryKey"`
	Name        string         `gorm:"not null;uniqueIndex:idx_series_name"`
	Description string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
	LastExportPath string
	Projects       []Project      `gorm:"foreignKey:SeriesID;constraint:OnDelete:CASCADE"`
}

// Project represents a project within a series
// Table: projects
type Project struct {
	ID          int64          `gorm:"primaryKey"`
	SeriesID    int64          `gorm:"not null;index:idx_project_series"`
	Code        string         `gorm:"not null;index:idx_project_code"` // Unique within a series
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
	Revisions   []BomRevision  `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`
}

// BomRevision represents a BOM revision
// Table: bom_revisions
type BomRevision struct {
	ID             int64           `gorm:"primaryKey"`
	ProjectID      int64           `gorm:"not null;index:idx_revision_project_phase_version,unique"`
	Phase          string          `gorm:"not null;index:idx_revision_project_phase_version,unique"`
	Version        string          `gorm:"not null;index:idx_revision_project_phase_version,unique"`
	Description    string
	SchematicVersion string
	PCBVersion     string
	PCAPN          string
	Date           string
	Mode           string // NPI or MP
	SourceFile     string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
	Parts          []Part         `gorm:"foreignKey:RevisionID;constraint:OnDelete:CASCADE"`
	SecondSources  []SecondSource `gorm:"foreignKey:RevisionID;constraint:OnDelete:CASCADE"`
	MatrixModels   []MatrixModel  `gorm:"foreignKey:RevisionID;constraint:OnDelete:CASCADE"`
	MatrixSelections []MatrixSelection `gorm:"foreignKey:RevisionID;constraint:OnDelete:CASCADE"`
}

// Part represents a component/part in the BOM
// Table: parts
type Part struct {
	ID          int64           `gorm:"primaryKey"`
	RevisionID  int64           `gorm:"not null;index:idx_part_revision_supplier_pn"`
	Type        string          `gorm:"not null;index:idx_part_revision_type"` // Main, 2nd Source
	Supplier    string          `gorm:"not null;index:idx_part_revision_supplier_pn"`
	SupplierPN  string          `gorm:"not null;index:idx_part_revision_supplier_pn"`
	Description string
	Location    string          // Atomic location (one per row)
	Quantity    int
	Cost        float64
	BOMStatus   string // I, X, P, M
	CCL         string // Y, N
	Remark      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// SecondSource represents a second source for a part
// Table: second_sources
type SecondSource struct {
	ID          int64           `gorm:"primaryKey"`
	RevisionID  int64           `gorm:"not null;index:idx_second_source_revision"`
	PartID      int64           `gorm:"not null;index"`
	Supplier    string          `gorm:"not null"`
	SupplierPN  string          `gorm:"not null"`
	Description string
	Cost        float64
	LeadTime    int
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// MatrixModel represents a matrix model selection
// Table: matrix_models
type MatrixModel struct {
	ID          int64           `gorm:"primaryKey"`
	RevisionID  int64           `gorm:"not null;index:idx_matrix_model_revision_name,unique"`
	ModelName   string          `gorm:"not null;index:idx_matrix_model_revision_name,unique"`
	Qty         int             `gorm:"not null;default:1"` //打件數量
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Selections  []MatrixSelection `gorm:"foreignKey:ModelID;constraint:OnDelete:CASCADE"`
}

// MatrixSelection represents a matrix selection
// Table: matrix_selections
type MatrixSelection struct {
	ID                  int64           `gorm:"primaryKey"`
	RevisionID          int64           `gorm:"not null;index"`
	ModelID             int64           `gorm:"not null;index:idx_matrix_selection_model_group_material,unique"`
	PartID              int64           `gorm:"not null;index"`
	Group               string          `gorm:"not null;index:idx_matrix_selection_model_group_material,unique"` // main_supplier + main_supplier_pn
	Material            string          `gorm:"not null;index:idx_matrix_selection_model_group_material,unique"` // supplier + supplier_pn
	SelectedSupplier    string          // The selected supplier (for reference)
	SelectedSupplierPn  string          // The selected supplier PN (for reference)
	IsAutoSelected      bool            `gorm:"default:false"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
