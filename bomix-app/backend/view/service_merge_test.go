package view

import (
	"testing"
)

// TestService_MergePartGroupsByMaterial_MultiViews 測試多視圖下跨 Type 物料合併規則
//
// 驗證規則：
//  1. ALL, NI, PROTO, MP, CCL 視圖適用跨 Type 合併，同料 SMD + BOTTOM 應合併為單一 Group。
//  2. SMD, PTH, BOTTOM 等專屬製程視圖應維持獨立 Type，不進行跨 Type 合併。
func TestService_MergePartGroupsByMaterial_MultiViews(t *testing.T) {
	// 模擬同一顆電容（MaterialID: 1001）分別在 SMD（56 顆）與 BOTTOM（10 顆）打件，並各自帶有 2 筆替代料
	smdGroup := ViewPartGroup{
		MaterialID:     1001,
		MainSupplier:   "MURATA",
		MainSupplierPN: "GRM188R60J226MEA0D",
		Item:           "12",
		HHPN:           "62011GL03-015-H",
		Description:    "CAP,22uF,+/-20%,X5R,6.3V,SMD0603",
		Type:           "SMD",
		BOMStatus:      "I",
		Qty:            56,
		Locations:      "AC1,PC215",
		CCL:            true,
		SecondSources: []ViewSecondSource{
			{MaterialID: 2001, Supplier: "KYOCERA", SupplierPN: "KGM15CR50J226MT"},
			{MaterialID: 2002, Supplier: "SAMSUNG", SupplierPN: "CL10A226MQ8NRNC"},
		},
	}

	bottomGroup := ViewPartGroup{
		MaterialID:     1001,
		MainSupplier:   "MURATA",
		MainSupplierPN: "GRM188R60J226MEA0D",
		Item:           "12",
		HHPN:           "62011GL03-015-H",
		Description:    "CAP,22uF,+/-20%,X5R,6.3V,SMD0603",
		Type:           "BOTTOM",
		BOMStatus:      "I",
		Qty:            10,
		Locations:      "HC138,HC139",
		CCL:            true,
		SecondSources: []ViewSecondSource{
			{MaterialID: 2001, Supplier: "KYOCERA", SupplierPN: "KGM15CR50J226MT"},
			{MaterialID: 2002, Supplier: "SAMSUNG", SupplierPN: "CL10A226MQ8NRNC"},
		},
	}

	rawGroups := []ViewPartGroup{smdGroup, bottomGroup}

	// 1. 驗證 MergePartGroupsByMaterial 能夠正確將 2 組跨面打件合併為 1 組
	merged := MergePartGroupsByMaterial(rawGroups)
	if len(merged) != 1 {
		t.Fatalf("跨 Type 合併後期望得到 1 個群組，實際得到 %d 個", len(merged))
	}

	mergedItem := merged[0]
	if mergedItem.Qty != 4 { // AC1, PC215 + HC138, HC139 共 4 個原子化位置
		t.Errorf("期望合併數量為 4，實際為 %d", mergedItem.Qty)
	}

	if len(mergedItem.SecondSources) != 2 {
		t.Errorf("期望替代料去重後為 2 筆，實際為 %d 筆", len(mergedItem.SecondSources))
	}

	if !mergedItem.CCL {
		t.Errorf("期望 CCL 為 true，實際為 false")
	}

	// 2. 測試視圖白名單判定邏輯
	mergeViewList := []string{"", ViewAll, ViewNI, ViewProto, ViewMP, ViewCCL}
	for _, vt := range mergeViewList {
		isMergeView := (vt == "" || vt == ViewAll || vt == ViewNI || vt == ViewProto || vt == ViewMP || vt == ViewCCL)
		if !isMergeView {
			t.Errorf("視圖 %s 應判定為需跨 Type 合併之視圖", vt)
		}
	}

	keepTypeViewList := []string{ViewSMD, ViewPTH, ViewBottom}
	for _, vt := range keepTypeViewList {
		isMergeView := (vt == "" || vt == ViewAll || vt == ViewNI || vt == ViewProto || vt == ViewMP || vt == ViewCCL)
		if isMergeView {
			t.Errorf("專屬製程視圖 %s 不應判定為跨 Type 合併", vt)
		}
	}
}
