package db

import (
	"time"

	"gorm.io/gorm"
)

// Series represents a product series
// Table: series
type Series struct {
	ID             int64          `gorm:"primaryKey"`
	Name           string         `gorm:"not null;uniqueIndex:idx_series_name"`
	Description    string
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
	Code        string         `gorm:"not null;index:idx_project_code"` // 系列內唯一
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
	Revisions   []BomRevision  `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`
}

// BomRevision represents a BOM revision
// Table: bom_revisions
type BomRevision struct {
	ID               int64             `gorm:"primaryKey"`
	ProjectID        int64             `gorm:"not null;index:idx_revision_project_phase_version,unique"`
	Phase            string            `gorm:"not null;index:idx_revision_project_phase_version,unique"`
	Version          string            `gorm:"not null;index:idx_revision_project_phase_version,unique"`
	Description      string
	SchematicVersion string
	PCBVersion       string
	PCAPN            string
	Date             string
	// Mode 代表此 BOM 的量產模式。
	// 於 EBOM 匯入 Phase 2 中自動判斷：
	//   - 若 PROTO 頁面任一 location 在 Phase 1 中已建立 → NPI
	//   - 若 MP 頁面任一 location 在 Phase 1 中已建立    → MP
	//   - 預設為 NPI
	Mode             string            `gorm:"default:NPI"`
	SourceFile       string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        gorm.DeletedAt    `gorm:"index"`
	Parts            []Part            `gorm:"foreignKey:RevisionID;constraint:OnDelete:CASCADE"`
	SecondSources    []SecondSource    `gorm:"foreignKey:RevisionID;constraint:OnDelete:CASCADE"`
	MatrixModels     []MatrixModel     `gorm:"foreignKey:RevisionID;constraint:OnDelete:CASCADE"`
	MatrixSelections []MatrixSelection `gorm:"foreignKey:RevisionID;constraint:OnDelete:CASCADE"`
}

// Part represents a component/part in the BOM（主料表，純化物料資訊）
//
// Location 原子化後，location / quantity / bom_status / ccl 均移至 PartLocation 表。
// 相同 Revision 中相同 (supplier, supplier_pn) 的物料只存一筆。
// Table: parts
type Part struct {
	ID         int64          `gorm:"primaryKey"`
	RevisionID int64          `gorm:"not null;index:idx_part_revision_supplier_pn"`
	// Type 代表製程類型：SMD / PTH / BOTTOM。
	// NI（不上件）不是 type，而是以 PartLocation.BomStatus = 'X' 表示。
	// 若物料僅出現在 NI sheet 且未在任何製程 sheet，則 Type 為空字串。
	Type        string         `gorm:"index:idx_part_revision_type"`
	Item        string         // 料號流水編號
	HHPN        string         // HH 內部料號
	Supplier    string         `gorm:"not null;index:idx_part_revision_supplier_pn"`
	SupplierPN  string         `gorm:"not null;index:idx_part_revision_supplier_pn"`
	Description string
	Cost        float64
	Remark      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
	// Locations 為關聯的 PartLocation 清單（一對多）。
	// 每個 PartLocation 代表一個原子化 location，並附帶獨立的 BomStatus 與 CCL 屬性。
	Locations []PartLocation `gorm:"foreignKey:PartID;constraint:OnDelete:CASCADE"`
}

// PartLocation 為 Location 原子化獨立表，每個 location 位置編號一筆紀錄。
// bom_status 與 ccl 屬性跟著 location 走，不再存放於 Part 表。
// Table: part_locations
type PartLocation struct {
	ID     int64  `gorm:"primaryKey"`
	PartID int64  `gorm:"not null;index:idx_part_location_part"`
	// Location 為單一原子化位置編號，例如 "C1"、"R5"。
	Location string `gorm:"not null;index:idx_part_location_name"`
	// BomStatus 代表此 location 的 BOM 狀態：
	//   I = Install（上件，預設）
	//   X = Not Install（不上件，NI sheet）
	//   P = Proto（PROTO sheet 覆寫）
	//   M = Mass Production（MP sheet 覆寫）
	BomStatus string `gorm:"default:I;index:idx_part_location_status"`
	// CCL 標記此 location 是否為 Critical Component（關鍵零件）。
	// 由 CCL sheet 或 Excel 中 CCL 欄位覆寫設定為 true。
	CCL bool `gorm:"default:false;index:idx_part_location_ccl"`
}

// SecondSource represents a second source for a part（替代料）
// Table: second_sources
type SecondSource struct {
	ID          int64     `gorm:"primaryKey"`
	RevisionID  int64     `gorm:"not null;index:idx_second_source_revision"`
	PartID      int64     `gorm:"not null;index"` // 關聯主料 Part
	HHPN        string
	Supplier    string    `gorm:"not null"`
	SupplierPN  string    `gorm:"not null"`
	Description string
	Cost        float64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// MatrixModel represents a matrix model selection
// Table: matrix_models
type MatrixModel struct {
	ID         int64             `gorm:"primaryKey"`
	RevisionID int64             `gorm:"not null;index:idx_matrix_model_revision_name,unique"`
	ModelName  string            `gorm:"not null;index:idx_matrix_model_revision_name,unique"`
	Qty        int               `gorm:"not null;default:1"` // 打件數量
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Selections []MatrixSelection `gorm:"foreignKey:ModelID;constraint:OnDelete:CASCADE"`
}

// MatrixSelection represents a matrix selection
// Table: matrix_selections
type MatrixSelection struct {
	ID                 int64     `gorm:"primaryKey"`
	RevisionID         int64     `gorm:"not null;index"`
	ModelID            int64     `gorm:"not null;index:idx_matrix_selection_model_group_material,unique"`
	PartID             int64     `gorm:"not null;index"`
	Group              string    `gorm:"not null;index:idx_matrix_selection_model_group_material,unique"` // main_supplier|main_supplier_pn
	Material           string    `gorm:"not null;index:idx_matrix_selection_model_group_material,unique"` // supplier|supplier_pn
	SelectedSupplier   string
	SelectedSupplierPn string
	IsAutoSelected     bool      `gorm:"default:false"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
