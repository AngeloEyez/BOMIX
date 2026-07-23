package view

import "strings"

// Filter 執行 ViewPartGroup 的視圖過濾。
//
// Filter 為無狀態結構，可安全地在多個 goroutine 中共用。
// 各視圖的過濾語義與 product-spec 6.4.2 定義一致。
type Filter struct{}

// NewFilter 建立一個新的 Filter 實例。
//
// 回傳：
//   - *Filter：Filter 實例
func NewFilter() *Filter {
	return &Filter{}
}

// Apply 根據 ViewQuery 中的 ViewType 與 Mode，過濾 ViewPartGroup 列表。
//
// 若 ViewType 為空字串，預設行為等同於 "ALL"。
// 過濾時，每個物料的 Mode（NPI/MP）優先使用 query.ModeOverride，
// 若 ModeOverride 為空則使用該物料所屬 revision 的 Mode。
//
// 由於單一 ViewPartGroup 可能來自多個 revision（SourceRevisionIDs），
// 此時 Mode 取 SourceRevisionIDs 中第一個 revision 的 Mode。
//
// 參數：
//   - parts：待過濾的 ViewPartGroup 列表
//   - query：查詢參數（包含 ViewType 與 ModeOverride）
//   - rawData：各 revision 的原始資料（用於查找 revision 的 Mode）
//
// 回傳：
//   - []ViewPartGroup：過濾後的 ViewPartGroup 列表
func (f *Filter) Apply(parts []ViewPartGroup, query ViewQuery, rawData map[int64]*rawRevisionData) []ViewPartGroup {
	viewType := strings.ToUpper(strings.TrimSpace(query.ViewType))
	if viewType == "" {
		viewType = ViewAll
	}

	switch viewType {
	case ViewAll:
		return f.filterAll(parts, query, rawData)
	case ViewSMD:
		return f.filterByType(parts, "SMD", query, rawData)
	case ViewPTH:
		return f.filterByType(parts, "PTH", query, rawData)
	case ViewBottom:
		return f.filterByType(parts, "BOTTOM", query, rawData)
	case ViewNI:
		return f.filterNI(parts, query, rawData)
	case ViewProto:
		return f.filterByBOMStatus(parts, "P")
	case ViewMP:
		return f.filterByBOMStatus(parts, "M")
	case ViewCCL:
		return f.filterCCL(parts, query, rawData)
	default:
		// 未知視圖類型：回傳全部（不過濾）
		return parts
	}
}

// resolveMode 決定物料的有效 Mode。
//
// 優先使用 query.ModeOverride；若未設定則從 rawData 中查找
// SourceRevisionIDs 的第一個 revision 的 Mode。
//
// 參數：
//   - part：物料群組
//   - query：查詢參數
//   - rawData：各 revision 的原始資料
//
// 回傳：
//   - string：有效的 Mode（"NPI" 或 "MP"）
func (f *Filter) resolveMode(part ViewPartGroup, query ViewQuery, rawData map[int64]*rawRevisionData) string {
	if query.ModeOverride != "" {
		return strings.ToUpper(query.ModeOverride)
	}
	// 取第一個 SourceRevisionID 對應的 revision Mode
	if len(part.SourceRevisionIDs) > 0 {
		if data, ok := rawData[part.SourceRevisionIDs[0]]; ok {
			return strings.ToUpper(data.revision.Mode)
		}
	}
	// fallback：NPI
	return "NPI"
}

// matchesMode 判斷物料的 BOMStatus 是否符合指定 Mode 的顯示條件。
//
// NPI 模式：顯示 I 與 P（Install + Proto）
// MP  模式：顯示 I 與 M（Install + MP Only）
//
// 參數：
//   - bomStatus：物料的上件狀態
//   - mode：NPI 或 MP
//
// 回傳：
//   - bool：是否符合顯示條件
func (f *Filter) matchesMode(bomStatus, mode string) bool {
	switch strings.ToUpper(mode) {
	case "NPI":
		return bomStatus == "I" || bomStatus == "P"
	case "MP":
		return bomStatus == "I" || bomStatus == "M"
	default:
		return bomStatus != "X"
	}
}

// filterAll 過濾出 ALL 視圖的物料。
//
// 排除 bom_status = X，依 Mode 決定顯示 I+P (NPI) 或 I+M (MP)。
// See product-spec section 6.4.2
//
// 參數：
//   - parts：待過濾列表
//   - query：查詢參數
//   - rawData：revision 原始資料（用於 Mode 查找）
//
// 回傳：
//   - []ViewPartGroup：過濾後的列表
func (f *Filter) filterAll(parts []ViewPartGroup, query ViewQuery, rawData map[int64]*rawRevisionData) []ViewPartGroup {
	result := make([]ViewPartGroup, 0, len(parts))
	for _, part := range parts {
		if part.BOMStatus == "X" {
			continue
		}
		mode := f.resolveMode(part, query, rawData)
		if f.matchesMode(part.BOMStatus, mode) {
			result = append(result, part)
		}
	}
	return result
}

// filterByType 過濾出特定製程類型（SMD/PTH/BOTTOM）的物料。
//
// 同時排除 bom_status = X，並依 Mode 過濾。
// See product-spec section 6.4.2
//
// 參數：
//   - parts：待過濾列表
//   - partType：製程類型（SMD、PTH 或 BOTTOM）
//   - query：查詢參數
//   - rawData：revision 原始資料
//
// 回傳：
//   - []ViewPartGroup：過濾後的列表
func (f *Filter) filterByType(parts []ViewPartGroup, partType string, query ViewQuery, rawData map[int64]*rawRevisionData) []ViewPartGroup {
	result := make([]ViewPartGroup, 0)
	for _, part := range parts {
		if !strings.EqualFold(part.Type, partType) {
			continue
		}
		if part.BOMStatus == "X" {
			continue
		}
		mode := f.resolveMode(part, query, rawData)
		if f.matchesMode(part.BOMStatus, mode) {
			result = append(result, part)
		}
	}
	return result
}

// filterNI 過濾出 NI（不上件）視圖的物料。
//
// NPI 模式：bom_status = X 或 M
// MP  模式：bom_status = X 或 P
// See product-spec section 6.4.2
//
// 參數：
//   - parts：待過濾列表
//   - query：查詢參數
//   - rawData：revision 原始資料
//
// 回傳：
//   - []ViewPartGroup：過濾後的列表
func (f *Filter) filterNI(parts []ViewPartGroup, query ViewQuery, rawData map[int64]*rawRevisionData) []ViewPartGroup {
	result := make([]ViewPartGroup, 0)
	for _, part := range parts {
		mode := f.resolveMode(part, query, rawData)
		switch strings.ToUpper(mode) {
		case "NPI":
			if part.BOMStatus == "X" || part.BOMStatus == "M" {
				result = append(result, part)
			}
		default: // MP
			if part.BOMStatus == "X" || part.BOMStatus == "P" {
				result = append(result, part)
			}
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

// filterCCL 過濾出關鍵零件 (CCL = Y) 且符合 Mode 的有效 BOM 狀態 (I + P/M)。
// See product-spec section 6.4.2 & 8.1.6
func (f *Filter) filterCCL(parts []ViewPartGroup, query ViewQuery, rawData map[int64]*rawRevisionData) []ViewPartGroup {
	// 先經過 filterAll 進行 bom_status 與 Mode 的有效性過濾 (NPI: I+P, MP: I+M, 排除 X)
	validParts := f.filterAll(parts, query, rawData)

	result := make([]ViewPartGroup, 0)
	for _, part := range validParts {
		if strings.EqualFold(part.CCL, "Y") {
			result = append(result, part)
		}
	}
	return result
}
