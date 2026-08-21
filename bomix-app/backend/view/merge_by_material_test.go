package view

import (
	"reflect"
	"testing"
)

// TestMergePartGroupsByMaterial_CrossTypeMerge 測試同一物料跨多個 Type (SMD + BOTTOM) 時的合併行為
func TestMergePartGroupsByMaterial_CrossTypeMerge(t *testing.T) {
	input := []ViewPartGroup{
		{
			MainSupplier:      "TI",
			MainSupplierPN:    "INA234AIYBJR",
			Item:              "971",
			HHPN:              "32041TN00-183-G",
			Description:       "Power Monitor IC",
			Type:              "SMD",
			BOMStatus:         "P",
			CCL:               true,
			Qty:               1,
			Locations:         "PSU1",
			SourceRevisionIDs: []int64{1, 2},
			SecondSources: []ViewSecondSource{
				{
					Supplier:          "Analog",
					SupplierPN:        "MAX1234",
					SourceRevisionIDs: []int64{1},
					SelectionsByOrder: map[int]bool{0: true},
				},
			},
			Selections: []ViewModelSelection{
				{RevisionID: 1, SortOrder: 0, ModelName: "Model_A", SelectedPN: "MAX1234"},
			},
			MainSelectionsByOrder: map[int]bool{1: true},
		},
		{
			MainSupplier:      "TI",
			MainSupplierPN:    "INA234AIYBJR",
			Item:              "971",
			HHPN:              "32041TN00-183-G",
			Description:       "Power Monitor IC",
			Type:              "BOTTOM",
			BOMStatus:         "P",
			CCL:               true,
			Qty:               1,
			Locations:         "PSU2",
			SourceRevisionIDs: []int64{1, 2},
			SecondSources: []ViewSecondSource{
				{
					Supplier:          "Analog",
					SupplierPN:        "MAX1234",
					SourceRevisionIDs: []int64{2},
					SelectionsByOrder: map[int]bool{0: true},
				},
				{
					Supplier:          "ADI",
					SupplierPN:        "ADM5678",
					SourceRevisionIDs: []int64{2},
				},
			},
			Selections: []ViewModelSelection{
				{RevisionID: 2, SortOrder: 0, ModelName: "Model_A", SelectedPN: "MAX1234"},
			},
			MainSelectionsByOrder: map[int]bool{0: true},
		},
		// 另一筆不相關的物料
		{
			MainSupplier:      "Samsung",
			MainSupplierPN:    "CL05B104",
			Item:              "101",
			Type:              "SMD",
			BOMStatus:         "I",
			Qty:               2,
			Locations:         "C1,C2",
			SourceRevisionIDs: []int64{1},
		},
	}

	merged := MergePartGroupsByMaterial(input)

	if len(merged) != 2 {
		t.Fatalf("期望合併為 2 筆物料群組，實際得到 %d 筆", len(merged))
	}

	// 驗證 INA234AIYBJR 群組
	var inaGroup *ViewPartGroup
	for i := range merged {
		if merged[i].MainSupplierPN == "INA234AIYBJR" {
			inaGroup = &merged[i]
			break
		}
	}

	if inaGroup == nil {
		t.Fatalf("未找到 INA234AIYBJR 合併後的群組")
	}

	if inaGroup.Locations != "PSU1,PSU2" {
		t.Errorf("Locations 合併期望 'PSU1,PSU2'，實際得到 %q", inaGroup.Locations)
	}
	if inaGroup.Qty != 2 {
		t.Errorf("Qty 期望 2，實際得到 %d", inaGroup.Qty)
	}
	if !inaGroup.CCL {
		t.Errorf("CCL 期望 true")
	}
	if inaGroup.BOMStatus != "P" {
		t.Errorf("BOMStatus 期望 'P'，實際得到 %q", inaGroup.BOMStatus)
	}
	if !reflect.DeepEqual(inaGroup.SourceRevisionIDs, []int64{1, 2}) {
		t.Errorf("SourceRevisionIDs 期望 [1 2]，實際得到 %v", inaGroup.SourceRevisionIDs)
	}

	// 驗證 2nd Sources (MAX1234 與 ADM5678)
	if len(inaGroup.SecondSources) != 2 {
		t.Fatalf("SecondSources 期望 2 個，實際得到 %d 個", len(inaGroup.SecondSources))
	}
	maxSS := inaGroup.SecondSources[1] // 排序後 MAX1234
	if maxSS.SupplierPN != "MAX1234" {
		maxSS = inaGroup.SecondSources[0]
	}
	if len(maxSS.SourceRevisionIDs) != 2 {
		t.Errorf("MAX1234 的 SourceRevisionIDs 期望合併為 2 個 revision，實際得到 %v", maxSS.SourceRevisionIDs)
	}

	// 驗證 Selections 合併 (Rev 1 與 Rev 2)
	if len(inaGroup.Selections) != 2 {
		t.Errorf("Selections 期望合併為 2 筆，實際得到 %d 筆", len(inaGroup.Selections))
	}

	// 驗證 MainSelectionsByOrder 合併 (0: true, 1: true)
	if !inaGroup.MainSelectionsByOrder[0] || !inaGroup.MainSelectionsByOrder[1] {
		t.Errorf("MainSelectionsByOrder 期望包含 0 與 1，實際得到 %v", inaGroup.MainSelectionsByOrder)
	}
}

// TestMergePartGroupsByMaterial_MixedBOMStatus 測試混合 BOMStatus 時的狀態判定
func TestMergePartGroupsByMaterial_MixedBOMStatus(t *testing.T) {
	input := []ViewPartGroup{
		{
			MainSupplier:   "Yageo",
			MainSupplierPN: "RC0402_P",
			BOMStatus:      "P",
			Locations:      "R1",
		},
		{
			MainSupplier:   "Yageo",
			MainSupplierPN: "RC0402_P",
			BOMStatus:      "I",
			Locations:      "R2",
		},
	}

	merged := MergePartGroupsByMaterial(input)
	if len(merged) != 1 {
		t.Fatalf("期望合併為 1 筆，實際得到 %d 筆", len(merged))
	}
	if merged[0].BOMStatus != "I" {
		t.Errorf("BOMStatus 期望降為 'I'，實際得到 %q", merged[0].BOMStatus)
	}
	if merged[0].Locations != "R1,R2" {
		t.Errorf("Locations 期望 'R1,R2'，實際得到 %q", merged[0].Locations)
	}
	if merged[0].Qty != 2 {
		t.Errorf("Qty 期望 2，實際得到 %d", merged[0].Qty)
	}
}
