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
	LastExportPath     string
	ProjectExportOrder string `gorm:"column:project_export_order"`
	Projects           []Project      `gorm:"foreignKey:SeriesID;constraint:OnDelete:CASCADE"`
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
	Components       []RevisionComponent `gorm:"foreignKey:RevisionID;constraint:OnDelete:CASCADE"`
	MatrixModels     []MatrixModel       `gorm:"foreignKey:RevisionID;constraint:OnDelete:CASCADE"`
	MatrixSelections []MatrixSelection   `gorm:"foreignKey:RevisionID;constraint:OnDelete:CASCADE"`
}

// Material 全域物料表 — 跨專案、跨版本共享物料主檔
// 以 (supplier, supplier_pn) 作為全域唯一鍵
// 僅在 EBOM 匯入時依「非空值覆寫」與「remark 允許清空」規則更新
// Table: materials
type Material struct {
	ID          int64          `gorm:"primaryKey"`
	Supplier    string         `gorm:"not null;uniqueIndex:idx_material_supplier_pn"`
	SupplierPN  string         `gorm:"not null;uniqueIndex:idx_material_supplier_pn"`
	HHPN        string         // HH 內部料號（依附於 supplier + supplier_pn）
	Description string         // 規格描述
	Remark      string         // 備註（若匯入為空則清空）
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// RevisionComponent 版本零件關聯表
// 記錄特定 Revision 包含哪些物料，以及主料與替代料的群組階層
// Table: revision_components
type RevisionComponent struct {
	ID                int64          `gorm:"primaryKey"`
	RevisionID        int64          `gorm:"not null;index:idx_rc_revision_material,unique"`
	MaterialID        int64          `gorm:"not null;index:idx_rc_revision_material,unique;index:idx_rc_material"`
	Role              string         `gorm:"size:1;not null;index:idx_rc_role"` // "M"=主料, "S"=替代料
	ParentComponentID int64          `gorm:"not null;default:0;index:idx_rc_revision_material,unique;index:idx_rc_parent"` // 若為替代料("S")，指向主料的 RevisionComponent.ID；主料則為 0
	Item              string         // Excel 原件項次流水號
	CreatedAt         time.Time
	UpdatedAt         time.Time
	Locations         []PartLocation `gorm:"foreignKey:ComponentID;constraint:OnDelete:CASCADE"`
}

// PartLocation 為 Location 原子化獨立表，每個 location 位置編號一筆紀錄。
// 製程類型 Type、bom_status 與 ccl 屬性跟著 location 走。
// Table: part_locations
type PartLocation struct {
	ID          int64  `gorm:"primaryKey"`
	ComponentID int64  `gorm:"not null;index:idx_part_location_component"` // 關聯 RevisionComponent.ID
	Location    string `gorm:"not null;index:idx_part_location_name"`
	// Type 代表該 location 所在之製程面別/型態：SMD / PTH / BOTTOM。
	Type string `gorm:"index:idx_part_location_type"`
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

// MatrixModel represents a matrix model selection
// Table: matrix_models
type MatrixModel struct {
	ID         int64             `gorm:"primaryKey"`
	RevisionID int64             `gorm:"not null;index:idx_matrix_model_revision_order,unique"`
	SortOrder  int               `gorm:"not null;index:idx_matrix_model_revision_order,unique"` // 0-based 排序索引 (0, 1, 2...)
	ModelName  string            // 顯示名稱 (選填)
	Qty        int               `gorm:"not null;default:1"` // 打件數量
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Selections []MatrixSelection `gorm:"foreignKey:ModelID;constraint:OnDelete:CASCADE"`
}

// MatrixSelection 代表矩陣勾選狀態
// Table: matrix_selections
type MatrixSelection struct {
	ID                 int64     `gorm:"primaryKey"`
	RevisionID         int64     `gorm:"not null;index"`
	ModelID            int64     `gorm:"not null;index:idx_matrix_sel_unique,unique"`
	ComponentID        int64     `gorm:"not null;index"`                              // 主料 RevisionComponent.ID
	MainMaterialID     int64     `gorm:"not null;index:idx_matrix_sel_unique,unique"` // 主料 MaterialID
	SelectedMaterialID int64     `gorm:"not null;index:idx_matrix_sel_unique,unique"` // 選中物料 MaterialID
	IsAutoSelected     bool      `gorm:"default:false"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
