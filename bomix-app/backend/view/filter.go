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
		return f.filterCCL(parts, query, mode)
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

// hasAnyMatrixSelection 檢查該物料群組在指定 revisionIDs 列表中是否有任何 Matrix Selection 勾選。
//
// 判定規則：
//  1. 優先檢查 part.Selections：若存在任一項其 RevisionID 屬於目標 revisionIDs（若未指定 revisionIDs 則視為全部有效），
//     且該項已被選中（SelectedMaterialID != 0 或 SelectedPN != ""），則判定為有勾選。
//  2. 容錯備援：若 part.Selections 為空（例如手動 Mock 測試資料），且查詢未限定多 revision，
//     則檢查 part.MainSelectionsByOrder 或其 SecondSources[].SelectionsByOrder 是否有任一項為 true。
//
// 參數：
//   - part: 物料群組
//   - revisionIDs: 當前查詢或匯出的 BOM Revision ID 列表
//
// 回傳：
//   - bool: 是否存在任何有效的 Matrix Selection 勾選
func hasAnyMatrixSelection(part ViewPartGroup, revisionIDs []int64) bool {
	targetRevMap := make(map[int64]bool, len(revisionIDs))
	for _, id := range revisionIDs {
		targetRevMap[id] = true
	}

	for _, sel := range part.Selections {
		if len(targetRevMap) > 0 && !targetRevMap[sel.RevisionID] {
			continue
		}
		if sel.SelectedMaterialID != 0 || sel.SelectedPN != "" {
			return true
		}
	}

	// 容錯檢查：若 Selections 列表為空且未指定 revisionIDs 或僅指定單一 revision
	if len(part.Selections) == 0 && len(targetRevMap) <= 1 {
		for _, isSel := range part.MainSelectionsByOrder {
			if isSel {
				return true
			}
		}
		for _, ss := range part.SecondSources {
			for _, isSel := range ss.SelectionsByOrder {
				if isSel {
					return true
				}
			}
		}
	}

	return false
}

// filterCCL 過濾出關鍵零件 (CCL = true 且符合當前 BOM 模式) 或在查詢/匯出 revisions 中有任何 matrix selection 勾選的有效物料。
// See product-spec section 6.4.2 & 8.1.6
//
// 過濾規則 (滿足任一條件即保留，但嚴格排除模式互斥物料)：
//  1. 模式互斥排除：
//     - 若 mode 為 "MP"，排除 bom_status 為 'P' (試產專用) 的物料。
//     - 若 mode 為 "NPI" (預設)，排除 bom_status 為 'M' (量產專用) 的物料。
//  2. 原有條件：part.CCL 為 true，且 bom_status 符合當前 BOM 模式 (NPI: I+P, MP: I+M)。
//  3. 矩陣勾選條件：在 query.RevisionIDs 範圍內，該物料群組存在任一 Model 的 Matrix Selection 勾選。
//
// 參數：
//   - parts：待過濾列表
//   - query：視圖查詢參數（包含目標 RevisionIDs）
//   - mode：當前 BOM 模式（NPI 或 MP）
//
// 回傳：
//   - []ViewPartGroup：過濾後的列表
func (f *Filter) filterCCL(parts []ViewPartGroup, query ViewQuery, mode string) []ViewPartGroup {
	result := make([]ViewPartGroup, 0)
	mode = strings.ToUpper(strings.TrimSpace(mode))

	for _, part := range parts {
		// 1. 模式互斥檢查：
		// 在 NPI 模式下，量產專用物料 ('M') 絕不屬於試產階段，必須排除
		// 在 MP 模式下，試產專用物料 ('P') 絕不屬於量產階段，必須排除
		if mode == "MP" {
			if part.BOMStatus == "P" {
				continue
			}
		} else {
			if part.BOMStatus == "M" {
				continue
			}
		}

		// 2. 檢查原有 CCL 條件
		isCCLMatch := false
		if part.CCL {
			if mode == "MP" {
				if part.BOMStatus == "I" || part.BOMStatus == "M" {
					isCCLMatch = true
				}
			} else {
				if part.BOMStatus == "I" || part.BOMStatus == "P" {
					isCCLMatch = true
				}
			}
		}

		// 3. 檢查在匯出/查詢的 revisions 中是否有任何 matrix selection 勾選
		hasSelection := hasAnyMatrixSelection(part, query.RevisionIDs)

		// 符合任一條件即保留
		if isCCLMatch || hasSelection {
			result = append(result, part)
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
			return "(ccl = true AND bom_status in ('I', 'M')) OR (選取 revision 中有 matrix selection 勾選且 bom_status != 'P')"
		}
		return "(ccl = true AND bom_status in ('I', 'P')) OR (選取 revision 中有 matrix selection 勾選且 bom_status != 'M')"
	default:
		return "不過濾 (全部顯示)"
	}
}
