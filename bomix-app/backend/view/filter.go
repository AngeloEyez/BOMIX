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
// 支援傳入可選的 BOM Mode（NPI 或 MP），用於精確控制 ALL 視圖下包含的狀態（I+P 或 I+M）。
//
// 參數：
//   - parts：待過濾的 ViewPartGroup 列表
//   - query：查詢參數（包含 ViewType 與可選 ModeOverride）
//   - modes：可選的 BOM Mode（若 query.ModeOverride 為空，則取第一個 mode）
//
// 回傳：
//   - []ViewPartGroup：過濾後的 ViewPartGroup 列表
func (f *Filter) Apply(parts []ViewPartGroup, query ViewQuery, modes ...string) []ViewPartGroup {
	viewType := strings.ToUpper(strings.TrimSpace(query.ViewType))
	if viewType == "" {
		viewType = ViewAll
	}

	mode := "NPI"
	if query.ModeOverride != "" {
		mode = strings.ToUpper(strings.TrimSpace(query.ModeOverride))
	} else if len(modes) > 0 && modes[0] != "" {
		mode = strings.ToUpper(strings.TrimSpace(modes[0]))
	}

	switch viewType {
	case ViewAll:
		return f.filterAll(parts, mode)
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
		return f.filterCCL(parts, mode)
	default:
		// 未知視圖類型：回傳全部（不過濾）
		return parts
	}
}

// filterAll 過濾出 ALL 視圖的物料。
//
// 根據 BOM Mode (NPI 或 MP) 決定有效的 bom_status 組合：
//   - Mode = "MP"：包含 bom_status 為 'I' 和 'M' 的物料（排除 'X' 和 'P'）
//   - Mode = "NPI"（預設）：包含 bom_status 為 'I' 和 'P' 的物料（排除 'X' 和 'M'）
//
// 參數：
//   - parts：待過濾列表
//   - mode：當前 BOM 模式（NPI 或 MP）
//
// 回傳：
//   - []ViewPartGroup：過濾後的列表
func (f *Filter) filterAll(parts []ViewPartGroup, mode string) []ViewPartGroup {
	result := make([]ViewPartGroup, 0, len(parts))
	mode = strings.ToUpper(strings.TrimSpace(mode))

	for _, part := range parts {
		if mode == "MP" {
			if part.BOMStatus == "I" || part.BOMStatus == "M" {
				result = append(result, part)
			}
		} else {
			if part.BOMStatus == "I" || part.BOMStatus == "P" {
				result = append(result, part)
			}
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

// filterCCL 過濾出關鍵零件 (CCL = true) 且 bom_status 符合當前 BOM 模式 (NPI: I+P, MP: I+M) 的有效物料。
// See product-spec section 6.4.2 & 8.1.6
//
// 參數：
//   - parts：待過濾列表
//   - mode：當前 BOM 模式（NPI 或 MP）
//
// 回傳：
//   - []ViewPartGroup：過濾後的列表
func (f *Filter) filterCCL(parts []ViewPartGroup, mode string) []ViewPartGroup {
	result := make([]ViewPartGroup, 0)
	mode = strings.ToUpper(strings.TrimSpace(mode))

	for _, part := range parts {
		if !part.CCL {
			continue
		}
		if mode == "MP" {
			if part.BOMStatus == "I" || part.BOMStatus == "M" {
				result = append(result, part)
			}
		} else {
			if part.BOMStatus == "I" || part.BOMStatus == "P" {
				result = append(result, part)
			}
		}
	}
	return result
}

// DescribeCondition 回傳指定視圖類型與 BOM 模式的過濾條件人類易讀說明。
//
// 此函式用於日誌記錄、除錯輸出與 UI 提示，條件規則依據 product-spec 6.4.2。
//
// 參數：
//   - viewType：視圖類型（ALL, SMD, PTH, BOTTOM, NI, PROTO, MP, CCL）
//   - mode：可選的 BOM 模式（NPI 或 MP）
//
// 回傳：
//   - string：過濾條件文字描述
func DescribeCondition(viewType string, mode string) string {
	vType := strings.ToUpper(strings.TrimSpace(viewType))
	if vType == "" {
		vType = ViewAll
	}
	m := "NPI"
	if strings.ToUpper(strings.TrimSpace(mode)) == "MP" {
		m = "MP"
	}

	switch vType {
	case ViewAll:
		if m == "MP" {
			return "bom_status in ('I', 'M') (排除 'X' 不上件與 'P' 試產)"
		}
		return "bom_status in ('I', 'P') (排除 'X' 不上件與 'M' 量產)"
	case ViewSMD:
		return "type = 'SMD' AND bom_status != 'X'"
	case ViewPTH:
		return "type = 'PTH' AND bom_status != 'X'"
	case ViewBottom:
		return "type = 'BOTTOM' AND bom_status != 'X'"
	case ViewNI:
		return "bom_status = 'X' (不上件)"
	case ViewProto:
		return "bom_status = 'P' (試產專用)"
	case ViewMP:
		return "bom_status = 'M' (量產專用)"
	case ViewCCL:
		if m == "MP" {
			return "ccl = true AND bom_status in ('I', 'M')"
		}
		return "ccl = true AND bom_status in ('I', 'P')"
	default:
		return "不過濾 (全部顯示)"
	}
}
