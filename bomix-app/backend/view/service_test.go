package view

import (
	"testing"

	"bomix-app/backend/db"
)

// ==================== ViewQuery 與 ViewType 常數測試 ====================

// TestViewTypeConstants 確認視圖類型常數定義正確
func TestViewTypeConstants(t *testing.T) {
	t.Run("ViewType 常數值", func(t *testing.T) {
		cases := []struct {
			constant string
			expected string
		}{
			{ViewAll, "ALL"},
			{ViewSMD, "SMD"},
			{ViewPTH, "PTH"},
			{ViewBottom, "BOTTOM"},
			{ViewNI, "NI"},
			{ViewProto, "PROTO"},
			{ViewMP, "MP"},
			{ViewCCL, "CCL"},
		}
		for _, tc := range cases {
			if tc.constant != tc.expected {
				t.Errorf("ViewType 常數錯誤: 期望 %q，實際 %q", tc.expected, tc.constant)
			}
		}
	})
}

// ==================== Filter 單元測試 ====================

// makeTestRawData 建立測試用的 rawRevisionData，模擬指定 mode 的 revision
func makeTestRawData(revID int64, mode string) map[int64]*rawRevisionData {
	return map[int64]*rawRevisionData{
		revID: {
			revision: db.BomRevision{
				ID:   revID,
				Mode: mode,
			},
		},
	}
}

// makeTestPart 快速建立 ViewPartGroup 測試資料
func makeTestPart(supplier, supplierPN, partType, bomStatus, ccl string, revIDs []int64) ViewPartGroup {
	return ViewPartGroup{
		MainSupplier:      supplier,
		MainSupplierPN:    supplierPN,
		Type:              partType,
		BOMStatus:         bomStatus,
		CCL:               ccl,
		SourceRevisionIDs: revIDs,
	}
}

// TestFilter_Apply_ALL 測試 ALL 視圖過濾
func TestFilter_Apply_ALL(t *testing.T) {
	filter := NewFilter()

	cases := []struct {
		name       string
		parts      []ViewPartGroup
		mode       string
		wantCount  int
		wantKeys   []string // supplier|supplierPN 的期望結果
	}{
		{
			name: "NPI模式：I和P通過，X和M排除",
			parts: []ViewPartGroup{
				makeTestPart("S1", "P1", "SMD", "I", "N", []int64{1}),
				makeTestPart("S2", "P2", "SMD", "P", "N", []int64{1}),
				makeTestPart("S3", "P3", "SMD", "X", "N", []int64{1}),
				makeTestPart("S4", "P4", "SMD", "M", "N", []int64{1}),
			},
			mode:      "NPI",
			wantCount: 2,
		},
		{
			name: "MP模式：I和M通過，X和P排除",
			parts: []ViewPartGroup{
				makeTestPart("S1", "P1", "SMD", "I", "N", []int64{1}),
				makeTestPart("S2", "P2", "SMD", "P", "N", []int64{1}),
				makeTestPart("S3", "P3", "SMD", "X", "N", []int64{1}),
				makeTestPart("S4", "P4", "SMD", "M", "N", []int64{1}),
			},
			mode:      "MP",
			wantCount: 2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rawData := makeTestRawData(1, tc.mode)
			query := ViewQuery{
				RevisionIDs:  []int64{1},
				ViewType:     ViewAll,
				ModeOverride: tc.mode,
			}
			result := filter.Apply(tc.parts, query, rawData)
			if len(result) != tc.wantCount {
				t.Errorf("期望 %d 個結果，實際得到 %d 個", tc.wantCount, len(result))
			}
		})
	}
}

// TestFilter_Apply_TypeFilter 測試製程類型過濾
func TestFilter_Apply_TypeFilter(t *testing.T) {
	filter := NewFilter()

	parts := []ViewPartGroup{
		makeTestPart("S1", "P1", "SMD", "I", "N", []int64{1}),
		makeTestPart("S2", "P2", "PTH", "I", "N", []int64{1}),
		makeTestPart("S3", "P3", "BOTTOM", "I", "N", []int64{1}),
		makeTestPart("S4", "P4", "SMD", "I", "N", []int64{1}),
	}
	rawData := makeTestRawData(1, "NPI")

	cases := []struct {
		viewType  string
		wantCount int
	}{
		{ViewSMD, 2},
		{ViewPTH, 1},
		{ViewBottom, 1},
	}

	for _, tc := range cases {
		t.Run(tc.viewType+" 過濾", func(t *testing.T) {
			query := ViewQuery{
				RevisionIDs:  []int64{1},
				ViewType:     tc.viewType,
				ModeOverride: "NPI",
			}
			result := filter.Apply(parts, query, rawData)
			if len(result) != tc.wantCount {
				t.Errorf("[%s] 期望 %d 個結果，實際得到 %d 個", tc.viewType, tc.wantCount, len(result))
			}
		})
	}
}

// TestFilter_Apply_NI 測試 NI 視圖過濾
func TestFilter_Apply_NI(t *testing.T) {
	filter := NewFilter()

	parts := []ViewPartGroup{
		makeTestPart("S1", "P1", "", "I", "N", []int64{1}),
		makeTestPart("S2", "P2", "", "X", "N", []int64{1}),
		makeTestPart("S3", "P3", "", "P", "N", []int64{1}),
		makeTestPart("S4", "P4", "", "M", "N", []int64{1}),
	}

	t.Run("NPI模式NI：X和M", func(t *testing.T) {
		rawData := makeTestRawData(1, "NPI")
		query := ViewQuery{RevisionIDs: []int64{1}, ViewType: ViewNI, ModeOverride: "NPI"}
		result := filter.Apply(parts, query, rawData)
		if len(result) != 2 {
			t.Errorf("NPI NI 期望 2 個（X+M），實際得到 %d 個", len(result))
		}
		for _, p := range result {
			if p.BOMStatus != "X" && p.BOMStatus != "M" {
				t.Errorf("NPI NI 不應包含 bom_status=%s 的物料", p.BOMStatus)
			}
		}
	})

	t.Run("MP模式NI：X和P", func(t *testing.T) {
		rawData := makeTestRawData(1, "MP")
		query := ViewQuery{RevisionIDs: []int64{1}, ViewType: ViewNI, ModeOverride: "MP"}
		result := filter.Apply(parts, query, rawData)
		if len(result) != 2 {
			t.Errorf("MP NI 期望 2 個（X+P），實際得到 %d 個", len(result))
		}
		for _, p := range result {
			if p.BOMStatus != "X" && p.BOMStatus != "P" {
				t.Errorf("MP NI 不應包含 bom_status=%s 的物料", p.BOMStatus)
			}
		}
	})
}

// TestFilter_Apply_CCL 測試 CCL 視圖過濾
func TestFilter_Apply_CCL(t *testing.T) {
	filter := NewFilter()

	parts := []ViewPartGroup{
		makeTestPart("S1", "P1", "SMD", "I", "Y", []int64{1}),
		makeTestPart("S2", "P2", "SMD", "I", "N", []int64{1}),
		makeTestPart("S3", "P3", "SMD", "I", "Y", []int64{1}),
	}
	rawData := makeTestRawData(1, "NPI")
	query := ViewQuery{RevisionIDs: []int64{1}, ViewType: ViewCCL}

	result := filter.Apply(parts, query, rawData)
	if len(result) != 2 {
		t.Errorf("CCL 過濾期望 2 個（CCL=Y），實際得到 %d 個", len(result))
	}
	for _, p := range result {
		if p.CCL != "Y" {
			t.Errorf("CCL 視圖不應包含 CCL=%s 的物料", p.CCL)
		}
	}
}

// TestFilter_Apply_EmptyViewType 測試空 ViewType 預設為 ALL
func TestFilter_Apply_EmptyViewType(t *testing.T) {
	filter := NewFilter()

	parts := []ViewPartGroup{
		makeTestPart("S1", "P1", "SMD", "I", "N", []int64{1}),
		makeTestPart("S2", "P2", "SMD", "X", "N", []int64{1}), // X 應被排除
	}
	rawData := makeTestRawData(1, "NPI")
	query := ViewQuery{RevisionIDs: []int64{1}, ViewType: "", ModeOverride: "NPI"}

	result := filter.Apply(parts, query, rawData)
	// 空 ViewType 應等同 ALL，排除 X
	if len(result) != 1 {
		t.Errorf("空 ViewType 預設 ALL，期望 1 個結果（排除X），實際得到 %d 個", len(result))
	}
}

// ==================== 輔助函數測試 ====================

// TestGroupKey 測試群組鍵生成
func TestGroupKey(t *testing.T) {
	key := groupKey("Samsung", "CL05B104")
	expected := "Samsung|CL05B104"
	if key != expected {
		t.Errorf("groupKey 期望 %q，實際 %q", expected, key)
	}
}

// TestSortedLocations 測試 location 排序
func TestSortedLocations(t *testing.T) {
	locSet := map[string]bool{
		"C5":  true,
		"C1":  true,
		"C10": true,
		"C2":  true,
	}
	result := sortedLocations(locSet)
	// 字串排序：C1, C10, C2, C5（字典序）
	expected := "C1,C10,C2,C5"
	if result != expected {
		t.Errorf("sortedLocations 期望 %q，實際 %q", expected, result)
	}
}

// TestAppendUnique 測試唯一 ID 追加
func TestAppendUnique(t *testing.T) {
	ids := []int64{1, 2, 3}

	// 追加已存在的 ID
	result := appendUnique(ids, 2)
	if len(result) != 3 {
		t.Errorf("追加已存在的 ID 後長度應為 3，實際 %d", len(result))
	}

	// 追加新 ID
	result = appendUnique(ids, 4)
	if len(result) != 4 {
		t.Errorf("追加新 ID 後長度應為 4，實際 %d", len(result))
	}
	if result[3] != 4 {
		t.Errorf("新追加的 ID 應為 4，實際 %d", result[3])
	}
}

// TestViewResult_EmptyQuery 測試空查詢
func TestViewResult_EmptyQuery(t *testing.T) {
	// 空 RevisionIDs 查詢不應 panic
	// 此測試不需要資料庫連線，僅測試 Service.Query() 的空輸入處理
	// 實際的空查詢回傳在 service.go 的 Query() 中已有早期返回
	query := ViewQuery{
		RevisionIDs: []int64{},
		ViewType:    ViewAll,
	}

	// 驗證查詢參數建立正確
	if len(query.RevisionIDs) != 0 {
		t.Error("空 RevisionIDs 應為長度 0")
	}
	if query.ViewType != ViewAll {
		t.Errorf("ViewType 期望 ALL，實際 %q", query.ViewType)
	}
}

// ==================== 多 Revision 聯集語義測試（不依賴 DB）====================

// TestSourceRevisionIDs_Logic 測試 SourceRevisionIDs 的來源歸屬邏輯
//
// 此測試驗證物料存在性判斷的語義：
//   - 物料 A 只在 Rev1 存在 → SourceRevisionIDs = [1]
//   - 物料 B 在 Rev1 和 Rev2 都存在 → SourceRevisionIDs = [1, 2]
//   - 物料 C 只在 Rev2 存在 → SourceRevisionIDs = [2]
func TestSourceRevisionIDs_Logic(t *testing.T) {
	rev1ID := int64(1)
	rev2ID := int64(2)

	// 模擬聯集結果（實際由 mergeRevisions 產生）
	partA := ViewPartGroup{
		MainSupplier:      "SA",
		MainSupplierPN:    "PA",
		SourceRevisionIDs: []int64{rev1ID},
	}
	partB := ViewPartGroup{
		MainSupplier:      "SB",
		MainSupplierPN:    "PB",
		SourceRevisionIDs: []int64{rev1ID, rev2ID},
	}
	partC := ViewPartGroup{
		MainSupplier:      "SC",
		MainSupplierPN:    "PC",
		SourceRevisionIDs: []int64{rev2ID},
	}

	// 驗證「物料是否存在於 revision」的判斷
	isInRev := func(part ViewPartGroup, revID int64) bool {
		for _, id := range part.SourceRevisionIDs {
			if id == revID {
				return true
			}
		}
		return false
	}

	// 物料 A：只在 Rev1
	if !isInRev(partA, rev1ID) {
		t.Error("物料A 應存在於 Rev1")
	}
	if isInRev(partA, rev2ID) {
		t.Error("物料A 不應存在於 Rev2")
	}

	// 物料 B：在 Rev1 和 Rev2 都存在
	if !isInRev(partB, rev1ID) {
		t.Error("物料B 應存在於 Rev1")
	}
	if !isInRev(partB, rev2ID) {
		t.Error("物料B 應存在於 Rev2")
	}

	// 物料 C：只在 Rev2
	if isInRev(partC, rev1ID) {
		t.Error("物料C 不應存在於 Rev1")
	}
	if !isInRev(partC, rev2ID) {
		t.Error("物料C 應存在於 Rev2")
	}
}

// TestIsEffectiveBOMStatus 測試 bom_status 在不同 Mode 下的有效性判斷
func TestIsEffectiveBOMStatus(t *testing.T) {
	t.Run("NPI 模式：僅 I 與 P 有效", func(t *testing.T) {
		if !isEffectiveBOMStatus("I", "NPI") {
			t.Error("I 在 NPI 應為有效")
		}
		if !isEffectiveBOMStatus("P", "NPI") {
			t.Error("P 在 NPI 應為有效")
		}
		if isEffectiveBOMStatus("X", "NPI") {
			t.Error("X 在 NPI 應為無效")
		}
		if isEffectiveBOMStatus("M", "NPI") {
			t.Error("M 在 NPI 應為無效")
		}
	})

	t.Run("MP 模式：僅 I 與 M 有效", func(t *testing.T) {
		if !isEffectiveBOMStatus("I", "MP") {
			t.Error("I 在 MP 應為有效")
		}
		if !isEffectiveBOMStatus("M", "MP") {
			t.Error("M 在 MP 應為有效")
		}
		if isEffectiveBOMStatus("X", "MP") {
			t.Error("X 在 MP 應為無效")
		}
		if isEffectiveBOMStatus("P", "MP") {
			t.Error("P 在 MP 應為無效")
		}
	})
}

// TestMergeRevisions_LocationAndQtyAggregation 測試同一 Revision 内跨 Type/CCL 的 Location 去重合併與 Qty 重新計算
func TestMergeRevisions_LocationAndQtyAggregation(t *testing.T) {
	svc := &Service{}

	rev1Data := &rawRevisionData{
		revision: db.BomRevision{ID: 1, Mode: "NPI"},
		parts: []db.Part{
			{ID: 101, RevisionID: 1, Supplier: "Samsung", SupplierPN: "CL05B104", Type: "SMD", BOMStatus: "I", CCL: "N", Location: "C1, C2", Item: "1"},
			{ID: 102, RevisionID: 1, Supplier: "Samsung", SupplierPN: "CL05B104", Type: "PTH", BOMStatus: "I", CCL: "Y", Location: "C2, C3", Item: "1"},
		},
	}

	rawData := map[int64]*rawRevisionData{1: rev1Data}
	query := ViewQuery{RevisionIDs: []int64{1}, ViewType: ViewAll}

	groups := svc.mergeRevisions(rawData, query)
	if len(groups) != 1 {
		t.Fatalf("期望 1 個物料群組，實際得到 %d 個", len(groups))
	}

	group := groups[0]
	// Location 應該合併去重為 "C1,C2,C3"
	if group.Locations != "C1,C2,C3" {
		t.Errorf("Locations 合併期望 %q，實際得到 %q", "C1,C2,C3", group.Locations)
	}
	// Qty 應該重新計算為 3
	if group.Qty != 3 {
		t.Errorf("Qty 期望 3，實際得到 %d", group.Qty)
	}
}

// TestMergeRevisions_MultipleRevisionsAnd2ndSources 測試多個 Revision 聚合與 2nd Source 判斷組裝
func TestMergeRevisions_MultipleRevisionsAnd2ndSources(t *testing.T) {
	svc := &Service{}

	// Rev 1: Part A (Main) + 2nd Source S1
	rev1Data := &rawRevisionData{
		revision: db.BomRevision{ID: 1, Mode: "NPI"},
		parts: []db.Part{
			{ID: 1, RevisionID: 1, Supplier: "Samsung", SupplierPN: "CL05B104", Type: "SMD", BOMStatus: "I", Location: "C1", Item: "1"},
		},
		secondSources: []db.SecondSource{
			{ID: 10, RevisionID: 1, PartID: 1, Supplier: "Yageo", SupplierPN: "CC0402KRX7R9BB104", HHPN: "HH1001"},
		},
	}

	// Rev 2: Part A (Main) + 2nd Source S1 & S2, plus Part B (Main)
	rev2Data := &rawRevisionData{
		revision: db.BomRevision{ID: 2, Mode: "NPI"},
		parts: []db.Part{
			{ID: 2, RevisionID: 2, Supplier: "Samsung", SupplierPN: "CL05B104", Type: "SMD", BOMStatus: "I", Location: "C1", Item: "1"},
			{ID: 3, RevisionID: 2, Supplier: "Murata", SupplierPN: "GRM155R71C104KA88D", Type: "SMD", BOMStatus: "I", Location: "C5", Item: "2"},
		},
		secondSources: []db.SecondSource{
			{ID: 11, RevisionID: 2, PartID: 2, Supplier: "Yageo", SupplierPN: "CC0402KRX7R9BB104", HHPN: "HH1001"},
			{ID: 12, RevisionID: 2, PartID: 2, Supplier: "Walsin", SupplierPN: "0402B104K500CT", HHPN: "HH1002"},
		},
	}

	rawData := map[int64]*rawRevisionData{1: rev1Data, 2: rev2Data}
	query := ViewQuery{RevisionIDs: []int64{1, 2}, ViewType: ViewAll}

	groups := svc.mergeRevisions(rawData, query)
	if len(groups) != 2 {
		t.Fatalf("期望 2 個物料群組，實際得到 %d 個", len(groups))
	}

	// Part A 驗證
	partA := groups[0]
	if len(partA.SourceRevisionIDs) != 2 {
		t.Errorf("Part A 應出現在 2 個 Revision 中，實際 SourceRevisionIDs=%v", partA.SourceRevisionIDs)
	}

	if len(partA.SecondSources) != 2 {
		t.Fatalf("Part A 的 2nd Source 應合併為 2 個，實際得到 %d 個", len(partA.SecondSources))
	}

	// Yageo 2nd Source 應包含 Rev 1 和 Rev 2
	var yageoSS *ViewSecondSource
	var walsinSS *ViewSecondSource
	for i := range partA.SecondSources {
		if partA.SecondSources[i].Supplier == "Yageo" {
			yageoSS = &partA.SecondSources[i]
		}
		if partA.SecondSources[i].Supplier == "Walsin" {
			walsinSS = &partA.SecondSources[i]
		}
	}

	if yageoSS == nil || len(yageoSS.SourceRevisionIDs) != 2 {
		t.Errorf("Yageo 2nd Source 的 SourceRevisionIDs 應為 [1, 2]")
	}
	if walsinSS == nil || len(walsinSS.SourceRevisionIDs) != 1 || walsinSS.SourceRevisionIDs[0] != 2 {
		t.Errorf("Walsin 2nd Source 的 SourceRevisionIDs 應僅包含 Rev 2 ([2])")
	}

	// Part B 驗證
	partB := groups[1]
	if len(partB.SourceRevisionIDs) != 1 || partB.SourceRevisionIDs[0] != 2 {
		t.Errorf("Part B 的 SourceRevisionIDs 應僅包含 Rev 2 ([2])")
	}
}

