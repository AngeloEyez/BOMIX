package view

import "strings"

// Filter 執行 ViewPartGroup 的視圖過濾。
//
// Filter 為無狀態結構，可安全地在多個 goroutine 中共用。
// 各視圖的過濾語義與 product-spec 6.4.2 定義一致（依 bom_status 獨立過濾）。
type Filter struct{}

// NewFilter 建立一個新的 Filter 實例。
//
// 回傳：
//   - *Filter：Filter 實例
func NewFilter() *Filter {
	return &Filter{}
}

// Apply 根據 ViewQuery 中的 ViewType 過濾 ViewPartGroup 列表。
//
// 若 ViewType 為空字串，預設行為等同於 "ALL"。
// 物料的上件狀態 bom_status（I, X, P, M）獨立過濾，解除對 BOM Mode (NPI/MP) 的依賴。
//
// 參數：
//   - parts：待過濾的 ViewPartGroup 列表
//   - query：查詢參數（包含 ViewType）
//
// 回傳：
//   - []ViewPartGroup：過濾後的 ViewPartGroup 列表
func (f *Filter) Apply(parts []ViewPartGroup, query ViewQuery) []ViewPartGroup {
	viewType := strings.ToUpper(strings.TrimSpace(query.ViewType))
	if viewType == "" {
		viewType = ViewAll
	}

	switch viewType {
	case ViewAll:
		return f.filterAll(parts)
	case ViewSMD:
		return f.filterByType(parts, "SMD")
	case ViewPTH:
		return f.filterByType(parts, "PTH")
	case ViewBottom:
		return f.filterByType(parts, "BOTTOM")
	case ViewNI:
		return f.filterNI(parts)
	case ViewProto:
		return f.filterByBOMStatus(parts, "P")
	case ViewMP:
		return f.filterByBOMStatus(parts, "M")
	case ViewCCL:
		return f.filterCCL(parts)
	default:
		// 未知視圖類型：回傳全部（不過濾）
		return parts
	}
}

// filterAll 過濾出 ALL 視圖的物料。
//
// 排除 bom_status = X 的所有物料（即保留 I, P, M）。
// See product-spec section 6.4.2
//
// 參數：
//   - parts：待過濾列表
//
// 回傳：
//   - []ViewPartGroup：過濾後的列表
func (f *Filter) filterAll(parts []ViewPartGroup) []ViewPartGroup {
	result := make([]ViewPartGroup, 0, len(parts))
	for _, part := range parts {
		if part.BOMStatus != "X" {
			result = append(result, part)
		}
	}
	return result
}

// filterByType 過濾出特定製程類型（SMD/PTH/BOTTOM）的物料。
//
// 同時排除 bom_status = X。
// See product-spec section 6.4.2
//
// 參數：
//   - parts：待過濾列表
//   - partType：製程類型（SMD、PTH 或 BOTTOM）
//
// 回傳：
//   - []ViewPartGroup：過濾後的列表
func (f *Filter) filterByType(parts []ViewPartGroup, partType string) []ViewPartGroup {
	result := make([]ViewPartGroup, 0)
	for _, part := range parts {
		if strings.EqualFold(part.Type, partType) && part.BOMStatus != "X" {
			result = append(result, part)
		}
	}
	return result
}

// filterNI 過濾出 NI（不上件）視圖的物料。
//
// 僅包含 bom_status = X。
// See product-spec section 6.4.2
//
// 參數：
//   - parts：待過濾列表
//
// 回傳：
//   - []ViewPartGroup：過濾後的列表
func (f *Filter) filterNI(parts []ViewPartGroup) []ViewPartGroup {
	result := make([]ViewPartGroup, 0)
	for _, part := range parts {
		if part.BOMStatus == "X" {
			result = append(result, part)
		}
	}
	return result
}

// filterByBOMStatus 過濾出具有指定 bom_status 的物料。
//
// 用於 PROTO 視圖（status="P"）與 MP 視圖（status="M"）。
// See product-spec section 6.4.2
//
// 參數：
//   - parts：待過濾列表
//   - status：目標 BOM 狀態碼（P 或 M）
//
// 回傳：
//   - []ViewPartGroup：過濾後的列表
func (f *Filter) filterByBOMStatus(parts []ViewPartGroup, status string) []ViewPartGroup {
	result := make([]ViewPartGroup, 0)
	for _, part := range parts {
		if part.BOMStatus == status {
			result = append(result, part)
		}
	}
	return result
}

// filterCCL 過濾出關鍵零件 (CCL = true) 且 bom_status != X 的有效物料。
// See product-spec section 6.4.2 & 8.1.6
func (f *Filter) filterCCL(parts []ViewPartGroup) []ViewPartGroup {
	validParts := f.filterAll(parts)

	result := make([]ViewPartGroup, 0)
	for _, part := range validParts {
		if part.CCL {
			result = append(result, part)
		}
	}
	return result
}
