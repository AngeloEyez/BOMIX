package view

// ViewType 視圖類型常數
// See product-spec section 6.4.2
const (
	ViewAll    = "ALL"
	ViewSMD    = "SMD"
	ViewPTH    = "PTH"
	ViewBottom = "BOTTOM"
	ViewNI     = "NI"
	ViewProto  = "PROTO"
	ViewMP     = "MP"
	ViewCCL    = "CCL"
)

// ViewQuery 視圖查詢參數，作為查詢的完整描述。
//
// 設計原則：View 系統為無狀態服務，每次呼叫時傳入完整的 ViewQuery，
// 多個 goroutine 可同時以不同的 ViewQuery 進行查詢而不互相影響。
//
// 欄位說明：
//   - RevisionIDs：查詢目標的 BOM Revision ID 列表。
//     傳入單個 ID → 單一 revision 視圖；
//     傳入多個 ID → 多 revision 整合聯集視圖。
//   - ViewType：視圖過濾類型，空字串預設為 ALL。
type ViewQuery struct {
	RevisionIDs  []int64 // 要查詢的 BOM Revision ID 列表（1個=單一視圖，多個=整合視圖）
	ViewType     string  // 視圖類型：ALL, SMD, PTH, BOTTOM, NI, PROTO, MP, CCL
	ModeOverride string  // 選填：覆寫 BOM 模式 (NPI 或 MP)
}

// ViewSecondSource 替代料（2nd Source）的視圖 DTO。
//
// SourceRevisionIDs 記錄此替代料出現在哪些 BOM Revision 中，
// 用於讓下游（Export/Frontend）判斷「此替代料在特定 revision 中是否存在」。
type ViewSecondSource struct {
	MaterialID        int64           `json:"material_id,omitempty"` // 全域物料 ID
	HHPN              string          `json:"hhpn"`
	Supplier          string          `json:"supplier"`
	SupplierPN        string          `json:"supplier_pn"`
	Description       string          `json:"description"`
	Remark            string          `json:"remark"`
	SourceRevisionIDs []int64         `json:"source_revision_ids"` // 包含此替代料的 Revision ID 列表
	SelectionsByOrder map[int]bool    `json:"selections_by_order"` // Model SortOrder -> 該 2nd Source 是否被勾選
}

// ViewModelSelection 某個 BOM Revision 內、某個 Model 的勾選狀態。
//
// 此 struct 為 ViewPartGroup.Selections 的元素，代表在某個 revision 的
// 某個 Model 中，此物料群組被選中的是哪顆料（主料或替代料）。
type ViewModelSelection struct {
	RevisionID         int64  `json:"revision_id"`
	SortOrder          int    `json:"sort_order"`        // 0-based 排序索引
	ModelName          string `json:"model_name"`
	ModelQty           int    `json:"model_qty"`
	SelectedMaterialID int64  `json:"selected_material_id,omitempty"`
	SelectedPN         string `json:"selected_pn"`       // 被選中的 SupplierPN（空字串=未勾選或尚未設定）
	SelectedSupplier   string `json:"selected_supplier"` // 被選中的 Supplier（廠牌）
	SelectedMaterial   string `json:"selected_material"` // 被選中的完整物料識別鍵 (Supplier|SupplierPN)
}

// ViewPartGroup 聚合後的物料群組，是 View 系統的核心輸出單元。
//
// 物料群組的識別鍵為 (MainSupplier, MainSupplierPN)。
// 當查詢涵蓋多個 BOM Revision 時，同一群組鍵的物料會被合併為一個
// ViewPartGroup，其中的 SourceRevisionIDs 記錄此群組存在於哪些 revision。
//
// 下游消費者（Export/Frontend）利用 SourceRevisionIDs 判斷：
//   if revisionID ∈ part.SourceRevisionIDs → 此物料在該 revision 中存在
//   if revisionID ∉ part.SourceRevisionIDs → 此物料在該 revision 中不存在
//     → BigMatrix 匯出：填灰色底色
//     → Frontend 顯示：加特殊標記
type ViewPartGroup struct {
	// 內部關聯鍵
	MaterialID int64 `json:"material_id,omitempty"` // 主料全域物料 ID（用於聚合與 Late-Binding）

	// 群組識別鍵
	MainSupplier   string `json:"main_supplier"`
	MainSupplierPN string `json:"main_supplier_pn"`

	// 物料基本屬性（取自 SourceRevisionIDs 中第一份包含此物料的 revision）
	Item        string `json:"item"`
	HHPN        string `json:"hhpn"`
	Description string `json:"description"`
	Type        string `json:"type"`       // SMD, PTH, BOTTOM
	BOMStatus   string `json:"bom_status"` // I, X, P, M
	CCL         bool   `json:"ccl"`        // 是否含有 CCL location (bool)
	Remark      string `json:"remark"`

	// 聚合結果（取自第一份有此物料的 revision）
	Qty       int    `json:"qty"`
	Locations string `json:"locations"` // 逗號分隔的位置編號

	// 來源歸屬：此物料群組出現在哪些 BOM Revision 中
	// 這是 View 系統的核心輸出，下游消費者依此判斷物料存在性
	SourceRevisionIDs []int64 `json:"source_revision_ids"`

	// 替代料（所有 revision 的聯集，以 supplier+supplier_pn 去重）
	// 每個 ViewSecondSource 也附帶自己的 SourceRevisionIDs 與 SelectionsByOrder
	SecondSources []ViewSecondSource `json:"second_sources"`

	// Model 勾選狀態（跨 revision × model 的完整矩陣）
	// 包含所有被查詢的 revision 中，此物料群組的所有 model 勾選記錄
	Selections []ViewModelSelection `json:"selections"`

	// 主料在各 Model 中的勾選狀態 (Model SortOrder -> isSelected)
	MainSelectionsByOrder map[int]bool `json:"main_selections_by_order"`
}

// ViewRevision BOM Revision 的元資料摘要，附帶在查詢結果中。
type ViewRevision struct {
	ID               int64          `json:"id"`
	ProjectCode      string         `json:"project_code"`
	Phase            string         `json:"phase"`
	Version          string         `json:"version"`
	Description      string         `json:"description"`
	SchematicVersion string         `json:"schematic_version"`
	PCBVersion       string         `json:"pcb_version"`
	PCAPN            string         `json:"pca_pn"`
	Date             string         `json:"date"`
	SourceFile       string         `json:"source_file"`
	ModelNames       []string       `json:"model_names"`
	ModelQty         map[string]int `json:"model_qty"`          // Model 名稱 -> 打件數量
	ModelQtyByOrder  map[int]int    `json:"model_qty_by_order"` // Model 排序索引 (0,1,2...) -> 打件數量
}

// ViewResult 視圖查詢結果，是 Service.Query() 的回傳值。
type ViewResult struct {
	Query      ViewQuery      `json:"query"`
	PartGroups []ViewPartGroup `json:"part_groups"`
	Revisions  []ViewRevision  `json:"revisions"` // 參與查詢的 revision 元資料（保持查詢順序）
}
