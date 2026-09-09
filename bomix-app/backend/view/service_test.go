package view

import (
	"testing"

	"bomix-app/backend/db"
	"gorm.io/gorm"
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

// makeTestRawData 建立測試用的 rawRevisionData
func makeTestRawData(revID int64) map[int64]*rawRevisionData {
	return map[int64]*rawRevisionData{
		revID: {
			revision: db.BomRevision{
				ID: revID,
			},
		},
	}
}

// makeTestPart 快速建立 ViewPartGroup 測試資料
func makeTestPart(supplier, supplierPN, partType, bomStatus string, ccl bool, revIDs []int64) ViewPartGroup {
	return ViewPartGroup{
		MainSupplier:      supplier,
		MainSupplierPN:    supplierPN,
		Type:              partType,
		BOMStatus:         bomStatus,
		CCL:               ccl,
		SourceRevisionIDs: revIDs,
	}
}

// TestFilter_Apply_ALL 測試 ALL 視圖過濾（依據 Mode 包含 I+P 或 I+M）
func TestFilter_Apply_ALL(t *testing.T) {
	filter := NewFilter()

	parts := []ViewPartGroup{
		makeTestPart("S1", "P1", "SMD", "I", false, []int64{1}),
		makeTestPart("S2", "P2", "SMD", "P", false, []int64{1}),
		makeTestPart("S3", "P3", "SMD", "X", false, []int64{1}),
		makeTestPart("S4", "P4", "SMD", "M", false, []int64{1}),
	}

	t.Run("NPI 模式 (預設)", func(t *testing.T) {
		query := ViewQuery{
			RevisionIDs: []int64{1},
			ViewType:    ViewAll,
		}
		result := filter.Apply(parts, query, "NPI")
		// NPI 模式下 ALL 應包含 I, P，排除 X, M (共 2 個)
		if len(result) != 2 {
			t.Errorf("ALL 視圖 (NPI) 期望 2 個結果（包含 I, P），實際得到 %d 個", len(result))
		}
		for _, p := range result {
			if p.BOMStatus != "I" && p.BOMStatus != "P" {
				t.Errorf("ALL 視圖 (NPI) 不應包含 bom_status=%s 的物料", p.BOMStatus)
			}
		}
	})

	t.Run("MP 模式", func(t *testing.T) {
		query := ViewQuery{
			RevisionIDs:  []int64{1},
			ViewType:     ViewAll,
			ModeOverride: "MP",
		}
		result := filter.Apply(parts, query)
		// MP 模式下 ALL 應包含 I, M，排除 X, P (共 2 個)
		if len(result) != 2 {
			t.Errorf("ALL 視圖 (MP) 期望 2 個結果（包含 I, M），實際得到 %d 個", len(result))
		}
		for _, p := range result {
			if p.BOMStatus != "I" && p.BOMStatus != "M" {
				t.Errorf("ALL 視圖 (MP) 不應包含 bom_status=%s 的物料", p.BOMStatus)
			}
		}
	})
}

// TestFilter_Apply_TypeFilter 測試製程類型過濾
func TestFilter_Apply_TypeFilter(t *testing.T) {
	filter := NewFilter()

	parts := []ViewPartGroup{
		makeTestPart("S1", "P1", "SMD", "I", false, []int64{1}),
		makeTestPart("S2", "P2", "PTH", "I", false, []int64{1}),
		makeTestPart("S3", "P3", "BOTTOM", "I", false, []int64{1}),
		makeTestPart("S4", "P4", "SMD", "I", false, []int64{1}),
	}

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
				RevisionIDs: []int64{1},
				ViewType:    tc.viewType,
			}
			result := filter.Apply(parts, query)
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
		makeTestPart("S1", "P1", "", "I", false, []int64{1}),
		makeTestPart("S2", "P2", "", "X", false, []int64{1}),
		makeTestPart("S3", "P3", "", "P", false, []int64{1}),
		makeTestPart("S4", "P4", "", "M", false, []int64{1}),
	}

	query := ViewQuery{RevisionIDs: []int64{1}, ViewType: ViewNI}
	result := filter.Apply(parts, query)
	// NI 應僅包含 bom_status = X
	if len(result) != 1 {
		t.Errorf("NI 視圖期望 1 個結果（僅X），實際得到 %d 個", len(result))
	}
	if result[0].BOMStatus != "X" {
		t.Errorf("NI 視圖期望 bom_status=X，實際=%s", result[0].BOMStatus)
	}
}

// TestFilter_Apply_CCL 測試 CCL 視圖過濾 (包含 BOM Mode NPI 與 MP 判定)
func TestFilter_Apply_CCL(t *testing.T) {
	filter := NewFilter()

	parts := []ViewPartGroup{
		makeTestPart("S1", "P1", "SMD", "I", true, []int64{1}),
		makeTestPart("S2", "P2", "SMD", "I", false, []int64{1}),
		makeTestPart("S3", "P3", "SMD", "P", true, []int64{1}),
		makeTestPart("S4", "P4", "SMD", "M", true, []int64{1}),
		makeTestPart("S5", "P5", "SMD", "X", true, []int64{1}),
	}

	t.Run("NPI 模式", func(t *testing.T) {
		query := ViewQuery{RevisionIDs: []int64{1}, ViewType: ViewCCL, ModeOverride: "NPI"}
		result := filter.Apply(parts, query)
		// 期望包含 I(S1) 與 P(S3)，排除 ccl=false(S2)、M(S4)與 X(S5)
		if len(result) != 2 {
			t.Errorf("NPI 模式 CCL 過濾期望 2 個（S1, S3），實際得到 %d 個", len(result))
		}
		for _, p := range result {
			if !p.CCL || (p.BOMStatus != "I" && p.BOMStatus != "P") {
				t.Errorf("NPI 模式 CCL 視圖不應包含 Status=%s, CCL=%v 的物料", p.BOMStatus, p.CCL)
			}
		}
	})

	t.Run("MP 模式", func(t *testing.T) {
		query := ViewQuery{RevisionIDs: []int64{1}, ViewType: ViewCCL, ModeOverride: "MP"}
		result := filter.Apply(parts, query)
		// 期望包含 I(S1) 與 M(S4)，排除 ccl=false(S2)、P(S3)與 X(S5)
		if len(result) != 2 {
			t.Errorf("MP 模式 CCL 過濾期望 2 個（S1, S4），實際得到 %d 個", len(result))
		}
		for _, p := range result {
			if !p.CCL || (p.BOMStatus != "I" && p.BOMStatus != "M") {
				t.Errorf("MP 模式 CCL 視圖不應包含 Status=%s, CCL=%v 的物料", p.BOMStatus, p.CCL)
			}
		}
	})
}

// TestFilter_Apply_EmptyViewType 測試空 ViewType 預設為 ALL
func TestFilter_Apply_EmptyViewType(t *testing.T) {
	filter := NewFilter()

	parts := []ViewPartGroup{
		makeTestPart("S1", "P1", "SMD", "I", false, []int64{1}),
		makeTestPart("S2", "P2", "SMD", "X", false, []int64{1}), // X 應被排除
	}
	query := ViewQuery{RevisionIDs: []int64{1}, ViewType: ""}

	result := filter.Apply(parts, query)
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

// TestIsEffectiveBOMStatus 測試 bom_status 有效性判斷
func TestIsEffectiveBOMStatus(t *testing.T) {
	t.Run("有效與無效狀態測試", func(t *testing.T) {
		if !isEffectiveBOMStatus("I") {
			t.Error("I 應為有效狀態")
		}
		if !isEffectiveBOMStatus("P") {
			t.Error("P 應為有效狀態")
		}
		if !isEffectiveBOMStatus("M") {
			t.Error("M 應為有效狀態")
		}
		if isEffectiveBOMStatus("X") {
			t.Error("X 應為無效狀態")
		}
	})
}

// setupTestDBForView 建立測試用資料庫
func setupTestDBForView(t *testing.T) *gorm.DB {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("建立記憶體資料庫失敗: %v", err)
	}
	if err := db.AutoMigrate(database); err != nil {
		t.Fatalf("AutoMigrate 失敗: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close(database)
	})
	return database
}

// TestMergeRevisions_LocationAndQtyAggregation 測試同一 Revision 内同 Type/CCL 的 Location 去重合併與 Qty 重新計算
func TestMergeRevisions_LocationAndQtyAggregation(t *testing.T) {
	database := setupTestDBForView(t)
	svc := NewService(database)

	// 建立物料
	mats := []db.Material{
		{ID: 1, Supplier: "Samsung", SupplierPN: "CL05B104"},
	}
	_ = database.Create(&mats).Error

	rev1Data := &rawRevisionData{
		revision: db.BomRevision{ID: 1},
		components: []db.RevisionComponent{
			{ID: 101, RevisionID: 1, MaterialID: 1, Role: "M", Item: "1"},
			{ID: 102, RevisionID: 1, MaterialID: 1, Role: "M", Item: "1"},
			{ID: 103, RevisionID: 1, MaterialID: 1, Role: "M", Item: "2"},
		},
		partLocations: []db.PartLocation{
			{ID: 1, ComponentID: 101, Location: "C1", Type: "SMD", BomStatus: "I", CCL: false},
			{ID: 2, ComponentID: 101, Location: "C2", Type: "SMD", BomStatus: "I", CCL: false},
			{ID: 3, ComponentID: 102, Location: "C2", Type: "SMD", BomStatus: "I", CCL: true},
			{ID: 4, ComponentID: 102, Location: "C3", Type: "SMD", BomStatus: "I", CCL: true},
			{ID: 5, ComponentID: 103, Location: "C4", Type: "PTH", BomStatus: "I", CCL: false},
		},
	}

	rawData := map[int64]*rawRevisionData{1: rev1Data}
	query := ViewQuery{RevisionIDs: []int64{1}, ViewType: ViewAll}

	groups := svc.mergeRevisions(rawData, query)
	_ = svc.hydratePartGroups(groups)

	if len(groups) != 2 {
		t.Fatalf("期望 2 個物料群組（1個SMD，1個PTH），實際得到 %d 個", len(groups))
	}

	smdGroup := groups[0]
	if smdGroup.Type != "SMD" {
		t.Errorf("第一個群組 Type 期望 SMD，實際得到 %s", smdGroup.Type)
	}
	// Location 應該合併去重為 "C1,C2,C3"
	if smdGroup.Locations != "C1,C2,C3" {
		t.Errorf("Locations 合併期望 %q，實際得到 %q", "C1,C2,C3", smdGroup.Locations)
	}
	// Qty 應該重新計算為 3
	if smdGroup.Qty != 3 {
		t.Errorf("Qty 期望 3，實際得到 %d", smdGroup.Qty)
	}

	pthGroup := groups[1]
	if pthGroup.Type != "PTH" {
		t.Errorf("第二個群組 Type 期望 PTH，實際得到 %s", pthGroup.Type)
	}
	if pthGroup.Locations != "C4" {
		t.Errorf("PTH Locations 期望 C4，實際得到 %s", pthGroup.Locations)
	}
}

// TestMergeRevisions_MultipleRevisionsAnd2ndSources 測試多個 Revision 聚合與 2nd Source 判斷組裝
func TestMergeRevisions_MultipleRevisionsAnd2ndSources(t *testing.T) {
	database := setupTestDBForView(t)
	svc := NewService(database)

	mats := []db.Material{
		{ID: 1, Supplier: "Samsung", SupplierPN: "CL05B104"},
		{ID: 2, Supplier: "Murata", SupplierPN: "GRM155R71C104KA88D"},
		{ID: 10, Supplier: "Yageo", SupplierPN: "CC0402KRX7R9BB104", HHPN: "HH1001"},
		{ID: 11, Supplier: "Walsin", SupplierPN: "0402B104K500CT", HHPN: "HH1002"},
	}
	_ = database.Create(&mats).Error

	// Rev 1: Part A (Main) + 2nd Source S1
	rev1Data := &rawRevisionData{
		revision: db.BomRevision{ID: 1},
		components: []db.RevisionComponent{
			{ID: 1, RevisionID: 1, MaterialID: 1, Role: "M", Item: "1"},
			{ID: 10, RevisionID: 1, MaterialID: 10, Role: "S", ParentComponentID: 1},
		},
		partLocations: []db.PartLocation{
			{ID: 1, ComponentID: 1, Location: "C1", Type: "SMD", BomStatus: "I", CCL: false},
		},
	}

	// Rev 2: Part A (Main) + 2nd Source S1 & S2, plus Part B (Main)
	rev2Data := &rawRevisionData{
		revision: db.BomRevision{ID: 2},
		components: []db.RevisionComponent{
			{ID: 2, RevisionID: 2, MaterialID: 1, Role: "M", Item: "1"},
			{ID: 3, RevisionID: 2, MaterialID: 2, Role: "M", Item: "2"},
			{ID: 11, RevisionID: 2, MaterialID: 10, Role: "S", ParentComponentID: 2},
			{ID: 12, RevisionID: 2, MaterialID: 11, Role: "S", ParentComponentID: 2},
		},
		partLocations: []db.PartLocation{
			{ID: 2, ComponentID: 2, Location: "C1", Type: "SMD", BomStatus: "I", CCL: false},
			{ID: 3, ComponentID: 3, Location: "C5", Type: "SMD", BomStatus: "I", CCL: false},
		},
	}

	rawData := map[int64]*rawRevisionData{1: rev1Data, 2: rev2Data}
	query := ViewQuery{RevisionIDs: []int64{1, 2}, ViewType: ViewAll}

	groups := svc.mergeRevisions(rawData, query)
	_ = svc.hydratePartGroups(groups)

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

// TestMergeRevisions_SequentialModels 驗證當多個 Model 名稱完全相同（如全為 "" 或 "Model"）時，
// mergeRevisions 仍可依據 SortOrder 順序性正確聚合每個 Model 的 Selection，不會導致舊 Model 被覆蓋。
func TestMergeRevisions_SequentialModels(t *testing.T) {
	database := setupTestDBForView(t)
	svc := NewService(database)

	mats := []db.Material{
		{ID: 1, Supplier: "Yageo", SupplierPN: "R100K"},
	}
	_ = database.Create(&mats).Error

	rev1Data := &rawRevisionData{
		revision: db.BomRevision{ID: 1},
		project:  db.Project{Code: "PROJ1"},
		models: []db.MatrixModel{
			{ID: 10, RevisionID: 1, SortOrder: 0, ModelName: "Model", Qty: 2},
			{ID: 11, RevisionID: 1, SortOrder: 1, ModelName: "Model", Qty: 5},
			{ID: 12, RevisionID: 1, SortOrder: 2, ModelName: "Model", Qty: 10},
		},
		components: []db.RevisionComponent{
			{ID: 100, RevisionID: 1, MaterialID: 1, Role: "M", Item: "1"},
		},
		partLocations: []db.PartLocation{
			{ID: 1000, ComponentID: 100, Location: "R1", Type: "SMD", BomStatus: "I", CCL: true},
		},
		selections: []db.MatrixSelection{
			{ID: 1, RevisionID: 1, ModelID: 10, ComponentID: 100, MainMaterialID: 1, SelectedMaterialID: 1},
			{ID: 2, RevisionID: 1, ModelID: 11, ComponentID: 100, MainMaterialID: 1, SelectedMaterialID: 1},
			{ID: 3, RevisionID: 1, ModelID: 12, ComponentID: 100, MainMaterialID: 1, SelectedMaterialID: 1},
		},
	}

	rawData := map[int64]*rawRevisionData{1: rev1Data}
	query := ViewQuery{RevisionIDs: []int64{1}, ViewType: ViewAll}

	groups := svc.mergeRevisions(rawData, query)
	_ = svc.hydratePartGroups(groups)

	if len(groups) != 1 {
		t.Fatalf("期望 1 個物料群組，實際得到 %d 個", len(groups))
	}

	pg := groups[0]
	if len(pg.Selections) != 3 {
		t.Fatalf("期望 3 個 Model Selection (依據 SortOrder 0, 1, 2)，實際得到 %d 個", len(pg.Selections))
	}

	for idx, sel := range pg.Selections {
		if sel.SortOrder != idx {
			t.Errorf("Selection[%d] 的 SortOrder 應為 %d，實際為 %d", idx, idx, sel.SortOrder)
		}
		if sel.SelectedPN != "R100K" {
			t.Errorf("Selection[%d] (SortOrder %d) 的 SelectedPN 應為 R100K，實際為 %q", idx, idx, sel.SelectedPN)
		}
	}
}

// TestBuildViewRevisions_Order 驗證 buildViewRevisions 會嚴格依照 requestedIDs 指定的順序回傳 ViewRevision 列表
func TestBuildViewRevisions_Order(t *testing.T) {
	rawData := map[int64]*rawRevisionData{
		1: {
			revision: db.BomRevision{ID: 1, Phase: "PV", Version: "0.1"},
			project:  db.Project{Code: "PROJ1"},
		},
		2: {
			revision: db.BomRevision{ID: 2, Phase: "DV", Version: "0.2"},
			project:  db.Project{Code: "PROJ2"},
		},
		3: {
			revision: db.BomRevision{ID: 3, Phase: "MP", Version: "1.0"},
			project:  db.Project{Code: "PROJ3"},
		},
	}

	t.Run("自訂反轉順序 [3, 1, 2]", func(t *testing.T) {
		reqIDs := []int64{3, 1, 2}
		revs := buildViewRevisions(reqIDs, rawData)

		if len(revs) != 3 {
			t.Fatalf("期望 3 個 ViewRevision，實際 %d 個", len(revs))
		}
		expectedIDs := []int64{3, 1, 2}
		for i, rev := range revs {
			if rev.ID != expectedIDs[i] {
				t.Errorf("Index %d 期望 Revision ID=%d，實際 got ID=%d", i, expectedIDs[i], rev.ID)
			}
		}
	})

	t.Run("自訂非連續順序 [2, 3]", func(t *testing.T) {
		reqIDs := []int64{2, 3}
		revs := buildViewRevisions(reqIDs, rawData)

		if len(revs) != 3 {
			t.Fatalf("期望 3 個 ViewRevision（2個由 requestedIDs 指定，1個補齊），實際 %d 個", len(revs))
		}
		expectedIDs := []int64{2, 3, 1}
		for i, rev := range revs {
			if rev.ID != expectedIDs[i] {
				t.Errorf("Index %d 期望 Revision ID=%d，實際 got ID=%d", i, expectedIDs[i], rev.ID)
			}
		}
	})
}

// TestMergeRevisions_AllProtoRequiredForPStatus 驗證物料群組必須全部 location 為 P 時 BOMStatus 才為 P
func TestMergeRevisions_AllProtoRequiredForPStatus(t *testing.T) {
	database := setupTestDBForView(t)
	svc := NewService(database)

	mats := []db.Material{
		{ID: 1, Supplier: "Samsung", SupplierPN: "MIXED_PN"},
		{ID: 2, Supplier: "Samsung", SupplierPN: "ALL_P_PN"},
	}
	_ = database.Create(&mats).Error

	revData := &rawRevisionData{
		revision: db.BomRevision{ID: 1},
		components: []db.RevisionComponent{
			{ID: 1, RevisionID: 1, MaterialID: 1, Role: "M", Item: "1"},
			{ID: 2, RevisionID: 1, MaterialID: 2, Role: "M", Item: "2"},
		},
		partLocations: []db.PartLocation{
			// Part 1 (MIXED_PN): 有 2 個 location，一個是 P，一個是 I
			{ID: 1, ComponentID: 1, Location: "C1", Type: "SMD", BomStatus: "P", CCL: false},
			{ID: 2, ComponentID: 1, Location: "C2", Type: "SMD", BomStatus: "I", CCL: false},

			// Part 2 (ALL_P_PN): 有 2 個 location，全都是 P
			{ID: 3, ComponentID: 2, Location: "C3", Type: "SMD", BomStatus: "P", CCL: false},
			{ID: 4, ComponentID: 2, Location: "C4", Type: "SMD", BomStatus: "P", CCL: false},
		},
	}

	rawData := map[int64]*rawRevisionData{1: revData}
	query := ViewQuery{RevisionIDs: []int64{1}, ViewType: ViewAll}

	groups := svc.mergeRevisions(rawData, query)
	_ = svc.hydratePartGroups(groups)

	if len(groups) != 2 {
		t.Fatalf("期望 2 個物料群組，實際得到 %d 個", len(groups))
	}

	var mixedGroup, allPGroup *ViewPartGroup
	for i := range groups {
		if groups[i].MainSupplierPN == "MIXED_PN" {
			mixedGroup = &groups[i]
		} else if groups[i].MainSupplierPN == "ALL_P_PN" {
			allPGroup = &groups[i]
		}
	}

	if mixedGroup == nil || allPGroup == nil {
		t.Fatalf("無法找到對應測試群組")
	}

	// 混合狀態 (P + I) 的群組，不應判定為 P（應退回 I）
	if mixedGroup.BOMStatus != "I" {
		t.Errorf("混合狀態群組 (P+I) BOMStatus 期望為 'I'，實際得到 %q", mixedGroup.BOMStatus)
	}

	// 全為 P 狀態的群組，應判定為 P
	if allPGroup.BOMStatus != "P" {
		t.Errorf("全 P 狀態群組 BOMStatus 期望為 'P'，實際得到 %q", allPGroup.BOMStatus)
	}
}

// TestMergeRevisions_NIViewSupport 驗證不上件 (BomStatus=X) 物料能正確歸納為 X 並在 NI 視圖中被查詢出
func TestMergeRevisions_NIViewSupport(t *testing.T) {
	database := setupTestDBForView(t)
	svc := NewService(database)

	mats := []db.Material{
		{ID: 10, Supplier: "Foxconn", SupplierPN: "PHCX01G11012"},
		{ID: 11, Supplier: "Naxin", SupplierPN: "101068500"},
	}
	_ = database.Create(&mats).Error

	revData := &rawRevisionData{
		revision: db.BomRevision{ID: 1},
		components: []db.RevisionComponent{
			{ID: 101, RevisionID: 1, MaterialID: 10, Role: "M", Item: "5"},
			{ID: 102, RevisionID: 1, MaterialID: 11, Role: "M", Item: "6"},
		},
		partLocations: []db.PartLocation{
			// Part 10 (Foxconn): 只有一個 location @U4，狀態為 X
			{ID: 1, ComponentID: 101, Location: "@U4", Type: "", BomStatus: "X", CCL: true},
			// Part 11 (Naxin): location 為 @U4，狀態為 I (PTH)
			{ID: 2, ComponentID: 102, Location: "@U4", Type: "PTH", BomStatus: "I", CCL: true},
		},
	}

	rawData := map[int64]*rawRevisionData{1: revData}

	// 1. 測試 mergeRevisions 聚合結果
	groups := svc.mergeRevisions(rawData, ViewQuery{RevisionIDs: []int64{1}, ViewType: ViewAll})
	_ = svc.hydratePartGroups(groups)

	if len(groups) != 2 {
		t.Fatalf("mergeRevisions 期望產出 2 個物料群組（Foxconn NI 與 Naxin PTH），實際得到 %d 個", len(groups))
	}

	filter := NewFilter()

	// 2. 測試 NI 視圖過濾
	niResult := filter.Apply(groups, ViewQuery{RevisionIDs: []int64{1}, ViewType: ViewNI})
	if len(niResult) != 1 {
		t.Fatalf("ViewNI 視圖期望過濾出 1 個群組，實際得到 %d 個", len(niResult))
	}
	if niResult[0].MainSupplierPN != "PHCX01G11012" || niResult[0].BOMStatus != "X" || niResult[0].Locations != "@U4" {
		t.Errorf("ViewNI 結果不符預期: %+v", niResult[0])
	}

	// 3. 測試 ALL 視圖過濾（應排除 X，僅保留 Naxin）
	allResult := filter.Apply(groups, ViewQuery{RevisionIDs: []int64{1}, ViewType: ViewAll})
	if len(allResult) != 1 {
		t.Fatalf("ViewAll 視圖期望過濾出 1 個群組，實際得到 %d 個", len(allResult))
	}
	if allResult[0].MainSupplierPN != "101068500" || allResult[0].BOMStatus != "I" {
		t.Errorf("ViewAll 結果不符預期: %+v", allResult[0])
	}
}



