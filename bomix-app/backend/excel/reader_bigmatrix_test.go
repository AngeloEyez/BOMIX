package excel

import (
	"fmt"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"bomix-app/backend/db"
	"bomix-app/backend/view"
)

// TestParseHeader_BigMatrix tests BigMatrix header parsing
func TestParseHeader_BigMatrix(t *testing.T) {
	f := excelize.NewFile()
	wb := &ExcelizeWorkbook{f: f}
	_ = wb
	defer f.Close()

	f.NewSheet("BigMatrix")

	// Set header values according to product-spec 7.2.2
	f.SetCellValue("BigMatrix", "B3", "BOMs: 4")
	f.SetCellValue("BigMatrix", "B4", "Description: MBD,Tangled,Multi-Board")
	f.SetCellValue("BigMatrix", "E4", "Date: 2026-01-15")

	reader := &BigMatrixReader{}
	description, bomCount, date := reader.parseHeader(wb, "BigMatrix")

	if bomCount != 4 {
		t.Errorf("Expected BOM count 4, got %d", bomCount)
	}
	if description != "MBD,Tangled,Multi-Board" {
		t.Errorf("Expected description 'MBD,Tangled,Multi-Board', got '%s'", description)
	}
	if date != "2026-01-15" {
		t.Errorf("Expected date '2026-01-15', got '%s'", date)
	}
}

// TestParseBOMConfigs tests horizontal multi-BOM configuration parsing
// See product-spec section 7.2.2.1
func TestParseBOMConfigs(t *testing.T) {
	f := excelize.NewFile()
	wb := &ExcelizeWorkbook{f: f}
	_ = wb
	defer f.Close()

	f.NewSheet("BigMatrix")

	// Create 2 BOMs with different Model counts
	// BOM 1: 3 Models (A, B, C) - columns H, I, J
	// BOM 2: 2 Models (A, B) - columns K, L

	// BOM 1 header
	f.SetCellValue("BigMatrix", "H2", "PROJECT_A")
	f.SetCellValue("BigMatrix", "H3", "PV-0.1")
	f.SetCellValue("BigMatrix", "H4", "A")
	f.SetCellValue("BigMatrix", "H5", "1")
	f.SetCellValue("BigMatrix", "I4", "B")
	f.SetCellValue("BigMatrix", "I5", "2")
	f.SetCellValue("BigMatrix", "J4", "C")
	f.SetCellValue("BigMatrix", "J5", "3")

	// BOM 2 header (starts at column K)
	f.SetCellValue("BigMatrix", "K2", "PROJECT_B")
	f.SetCellValue("BigMatrix", "K3", "PV-0.2")
	f.SetCellValue("BigMatrix", "K4", "A")
	f.SetCellValue("BigMatrix", "K5", "2")
	f.SetCellValue("BigMatrix", "L4", "B")
	f.SetCellValue("BigMatrix", "L5", "1")

	reader := &BigMatrixReader{}
	configs, err := reader.parseBOMConfigs(wb, "BigMatrix")

	if err != nil {
		t.Fatalf("parseBOMConfigs failed: %v", err)
	}

	if len(configs) != 2 {
		t.Fatalf("Expected 2 BOM configs, got %d", len(configs))
	}

	// Check BOM 1
	if configs[0].ProjectCode != "PROJECT_A" {
		t.Errorf("Expected BOM 1 project 'PROJECT_A', got '%s'", configs[0].ProjectCode)
	}
	if configs[0].Phase != "PV" {
		t.Errorf("Expected BOM 1 phase 'PV', got '%s'", configs[0].Phase)
	}
	if configs[0].Version != "0.1" {
		t.Errorf("Expected BOM 1 version '0.1', got '%s'", configs[0].Version)
	}
	if configs[0].ModelCount != 3 {
		t.Errorf("Expected BOM 1 model count 3, got %d", configs[0].ModelCount)
	}
	if configs[0].Models[0].ModelName != "A" || configs[0].Models[0].Qty != 1 {
		t.Error("Expected BOM 1 Model A with qty 1")
	}
	if configs[0].Models[1].ModelName != "B" || configs[0].Models[1].Qty != 2 {
		t.Error("Expected BOM 1 Model B with qty 2")
	}
	if configs[0].Models[2].ModelName != "C" || configs[0].Models[2].Qty != 3 {
		t.Error("Expected BOM 1 Model C with qty 3")
	}

	// Check BOM 2
	if configs[1].ProjectCode != "PROJECT_B" {
		t.Errorf("Expected BOM 2 project 'PROJECT_B', got '%s'", configs[1].ProjectCode)
	}
	if configs[1].Phase != "PV" {
		t.Errorf("Expected BOM 2 phase 'PV', got '%s'", configs[1].Phase)
	}
	if configs[1].Version != "0.2" {
		t.Errorf("Expected BOM 2 version '0.2', got '%s'", configs[1].Version)
	}
	if configs[1].ModelCount != 2 {
		t.Errorf("Expected BOM 2 model count 2, got %d", configs[1].ModelCount)
	}
}

// TestParsePartDataRow tests part data row parsing
func TestParsePartDataRow(t *testing.T) {
	reader := &BigMatrixReader{}

	row := []string{
		"1",              // A: Item
		"HHPN-001",       // B: HHPN
		"CAPACITOR 10uF", // C: Description
		"Samsung",        // D: Supplier
		"CL10A106MQ8NNNC", // E: Supplier PN
		"3",              // F: Qty
		"C1,C2,C3",      // G: Location
	}

	data := reader.parsePartDataRow(row)

	if data == nil {
		t.Fatal("Expected non-nil part data")
	}

	if data.item != "1" {
		t.Errorf("Expected item '1', got '%s'", data.item)
	}
	if data.hhpn != "HHPN-001" {
		t.Errorf("Expected hhpn 'HHPN-001', got '%s'", data.hhpn)
	}
	if data.description != "CAPACITOR 10uF" {
		t.Errorf("Expected description 'CAPACITOR 10uF', got '%s'", data.description)
	}
	if data.supplier != "Samsung" {
		t.Errorf("Expected supplier 'Samsung', got '%s'", data.supplier)
	}
	if data.supplierPN != "CL10A106MQ8NNNC" {
		t.Errorf("Expected supplierPN 'CL10A106MQ8NNNC', got '%s'", data.supplierPN)
	}
	if data.qty != 3 {
		t.Errorf("Expected qty 3, got %d", data.qty)
	}
	if data.location != "C1,C2,C3" {
		t.Errorf("Expected location 'C1,C2,C3', got '%s'", data.location)
	}
}

// TestParsePartDataRow_SecondSource tests 2nd Source row parsing
func TestParsePartDataRow_SecondSource(t *testing.T) {
	reader := &BigMatrixReader{}

	// 2nd Source row (empty item)
	row := []string{
		"",               // A: Item (empty for 2nd source)
		"HHPN-001",       // B: HHPN
		"CAPACITOR Alt",  // C: Description
		"Murata",         // D: Supplier
		"GRM188R61A106KE15D", // E: Supplier PN
		"",              // F: Qty
		"",              // G: Location
	}

	data := reader.parsePartDataRow(row)

	if data == nil {
		t.Fatal("Expected non-nil part data")
	}

	if data.item != "" {
		t.Errorf("Expected empty item for 2nd source, got '%s'", data.item)
	}
	if data.supplier != "Murata" {
		t.Errorf("Expected supplier 'Murata', got '%s'", data.supplier)
	}
}

// TestColToCell tests column index to Excel cell conversion
func TestColToCell(t *testing.T) {
	tests := []struct {
		col    int
		row    int
		expect string
	}{
		{7, 2, "H2"},
		{8, 4, "I4"},
		{9, 5, "J5"},
		{10, 6, "K6"},
		{25, 1, "Z1"},
		{26, 1, "AA1"},
	}

	for _, test := range tests {
		result := colToCell(test.col, test.row)
		if result != test.expect {
			t.Errorf("colToCell(%d, %d) = %s, expected %s", test.col, test.row, result, test.expect)
		}
	}
}

// TestImport_BigMatrix_Integration tests full BigMatrix import flow
func TestImport_BigMatrix_Integration(t *testing.T) {
	// Create in-memory database
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect database: %v", err)
	}

	// Auto migrate
	if err := db.AutoMigrate(database); err != nil {
		t.Fatalf("Failed to auto migrate: %v", err)
	}

	// Create series, project, and revision
	series := db.Series{Name: "Test Series"}
	if err := database.Create(&series).Error; err != nil {
		t.Fatalf("Failed to create series: %v", err)
	}

	project := db.Project{
		SeriesID:    series.ID,
		Code:        "TEST_PROJECT",
		Description: "Test Project",
	}
	if err := database.Create(&project).Error; err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	revision := db.BomRevision{
		ProjectID: project.ID,
		Phase:     "PV",
		Version:   "0.1",
	}
	if err := database.Create(&revision).Error; err != nil {
		t.Fatalf("Failed to create revision: %v", err)
	}

	// 預先建立 Part（因為 BigMatrix 匯入不建立/更新 Part）
	part := db.Part{
		RevisionID: revision.ID,
		Supplier:   "Samsung",
		SupplierPN: "CL10A106MQ8NNNC",
		Type:       "SMD",
	}
	if err := database.Create(&part).Error; err != nil {
		t.Fatalf("Failed to create test part: %v", err)
	}

	// Create test Excel file
	f := excelize.NewFile()
	wb := &ExcelizeWorkbook{f: f}
	_ = wb
	defer f.Close()

	f.NewSheet("BigMatrix")

	// Set header
	f.SetCellValue("BigMatrix", "B3", "BOMs: 1")
	f.SetCellValue("BigMatrix", "B4", "Description: Test BigMatrix")
	f.SetCellValue("BigMatrix", "E4", "Date: 2026-01-15")

	// Create BOM config with 2 Models
	f.SetCellValue("BigMatrix", "H2", "TEST_PROJECT")
	f.SetCellValue("BigMatrix", "H3", "PV-0.1")
	f.SetCellValue("BigMatrix", "H4", "A")
	f.SetCellValue("BigMatrix", "H5", "1")
	f.SetCellValue("BigMatrix", "I4", "B")
	f.SetCellValue("BigMatrix", "I5", "2")

	// Add a part with Matrix selection
	f.SetCellValue("BigMatrix", "A6", "1")
	f.SetCellValue("BigMatrix", "B6", "HHPN-001")
	f.SetCellValue("BigMatrix", "C6", "CAPACITOR 10uF")
	f.SetCellValue("BigMatrix", "D6", "Samsung")
	f.SetCellValue("BigMatrix", "E6", "CL10A106MQ8NNNC")
	f.SetCellValue("BigMatrix", "F6", "3")
	f.SetCellValue("BigMatrix", "G6", "C1,C2,C3")
	f.SetCellValue("BigMatrix", "H6", "V") // Model A selected
	f.SetCellValue("BigMatrix", "I6", "")  // Model B not selected

	// Import
	reader := &BigMatrixReader{
		db: database,
	}

	err = reader.Import(wb)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Verify project was created
	if err := database.Where("code = ?", "TEST_PROJECT").First(&project).Error; err != nil {
		t.Fatalf("Project not found: %v", err)
	}

	// Verify revision was created
	if err := database.Where("project_id = ? AND phase = ? AND version = ?",
		project.ID, "PV", "0.1").First(&revision).Error; err != nil {
		t.Fatalf("Revision not found: %v", err)
	}

	// Verify MatrixModel was created
	var models []db.MatrixModel
	if err := database.Where("revision_id = ?", revision.ID).Find(&models).Error; err != nil {
		t.Fatalf("Failed to query models: %v", err)
	}

	if len(models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(models))
	}

	// Verify MatrixSelection was created
	var selections []db.MatrixSelection
	if err := database.Where("revision_id = ?", revision.ID).Find(&selections).Error; err != nil {
		t.Fatalf("Failed to query selections: %v", err)
	}

	// Should have 1 selection (Model A selected)
	if len(selections) != 1 {
		t.Errorf("Expected 1 selection, got %d", len(selections))
	}
}

// TestImport_BigMatrix_ClearsOldSelections tests that BigMatrix import clears old selections
// See product-spec section 7.0.2
func TestImport_BigMatrix_ClearsOldSelections(t *testing.T) {
	// Create in-memory database
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect database: %v", err)
	}

	// Auto migrate
	if err := db.AutoMigrate(database); err != nil {
		t.Fatalf("Failed to auto migrate: %v", err)
	}

	series := db.Series{Name: "Test Series"}
	_ = database.Create(&series).Error

	// Create project and revision
	project := db.Project{
		SeriesID:    series.ID,
		Code:        "TEST_PROJECT",
		Description: "Test Project",
	}
	if err := database.Create(&project).Error; err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	revision := db.BomRevision{
		ProjectID: project.ID,
		Phase:     "PV",
		Version:   "0.1",
	}
	if err := database.Create(&revision).Error; err != nil {
		t.Fatalf("Failed to create revision: %v", err)
	}

	// 預先建立物料
	newPart := db.Part{
		RevisionID: revision.ID,
		Supplier:   "NewSupplier",
		SupplierPN: "NEW-PN",
		Type:       "SMD",
	}
	_ = database.Create(&newPart).Error

	// Create existing MatrixModel and Selections
	modelA := db.MatrixModel{
		RevisionID: revision.ID,
		ModelName:  "A",
		Qty:        1,
	}
	if err := database.Create(&modelA).Error; err != nil {
		t.Fatalf("Failed to create model A: %v", err)
	}

	// Create an old selection that should be cleared
	oldSelection := db.MatrixSelection{
		RevisionID:         revision.ID,
		ModelID:            modelA.ID,
		PartID:             1,
		Group:              "Old|Group",
		Material:           "Old|Material",
		SelectedSupplier:   "Old",
		SelectedSupplierPn: "PN",
	}
	if err := database.Create(&oldSelection).Error; err != nil {
		t.Fatalf("Failed to create old selection: %v", err)
	}

	// Create test Excel file with new selections
	f := excelize.NewFile()
	wb := &ExcelizeWorkbook{f: f}
	_ = wb
	defer f.Close()
	f.NewSheet("BigMatrix")

	f.SetCellValue("BigMatrix", "B3", "BOMs: 1")
	f.SetCellValue("BigMatrix", "H2", "TEST_PROJECT")
	f.SetCellValue("BigMatrix", "H3", "PV-0.1")
	f.SetCellValue("BigMatrix", "H4", "A")
	f.SetCellValue("BigMatrix", "H5", "1")

	// Add a new part with selection
	f.SetCellValue("BigMatrix", "A6", "1")
	f.SetCellValue("BigMatrix", "B6", "HHPN-NEW")
	f.SetCellValue("BigMatrix", "D6", "NewSupplier")
	f.SetCellValue("BigMatrix", "E6", "NEW-PN")
	f.SetCellValue("BigMatrix", "H6", "V")

	// Import
	reader := &BigMatrixReader{
		db: database,
	}

	err = reader.Import(wb)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Verify old selection was deleted
	var remainingSelections []db.MatrixSelection
	if err := database.Where("revision_id = ?", revision.ID).Find(&remainingSelections).Error; err != nil {
		t.Fatalf("Failed to query selections: %v", err)
	}

	// The old selection should be deleted, only new one should exist
	// (Note: This depends on the actual import implementation)
	for _, sel := range remainingSelections {
		if sel.Group == "Old|Group" {
			t.Error("Expected old selection to be deleted")
		}
	}
}

// TestImport_BigMatrix_MultipleModelsQtyAndSelections tests that multiple models in a revision
// maintain distinct Qtys (A:10, B:20, C:30) and their respective selections are properly populated.
func TestImport_BigMatrix_MultipleModelsQtyAndSelections(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect database: %v", err)
	}
	if err := db.AutoMigrate(database); err != nil {
		t.Fatalf("Failed to auto migrate: %v", err)
	}

	series := db.Series{Name: "Test Series"}
	_ = database.Create(&series).Error

	project := db.Project{
		SeriesID: series.ID,
		Code:     "MULTI_MODEL_PROJ",
	}
	if err := database.Create(&project).Error; err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	revision := db.BomRevision{
		ProjectID: project.ID,
		Phase:     "PV",
		Version:   "0.1",
	}
	if err := database.Create(&revision).Error; err != nil {
		t.Fatalf("Failed to create revision: %v", err)
	}

	// 預先建立 Parts
	part1 := db.Part{RevisionID: revision.ID, Supplier: "SupA", SupplierPN: "PN1", Type: "SMD"}
	part2 := db.Part{RevisionID: revision.ID, Supplier: "SupB", SupplierPN: "PN2", Type: "SMD"}
	_ = database.Create(&part1).Error
	_ = database.Create(&part2).Error

	// 建立 Excel 檔，其中 Row 2 (H2, I2, J2) 橫向皆被填寫為專案代碼 (模擬併欄/填滿狀況)
	f := excelize.NewFile()
	wb := &ExcelizeWorkbook{f: f}
	defer f.Close()

	f.NewSheet("BigMatrix")
	f.SetCellValue("BigMatrix", "B3", "BOMs: 1")
	f.SetCellValue("BigMatrix", "H2", "MULTI_MODEL_PROJ")
	f.SetCellValue("BigMatrix", "I2", "MULTI_MODEL_PROJ")
	f.SetCellValue("BigMatrix", "J2", "MULTI_MODEL_PROJ")

	f.SetCellValue("BigMatrix", "H3", "PV-0.1")
	f.SetCellValue("BigMatrix", "I3", "PV-0.1")
	f.SetCellValue("BigMatrix", "J3", "PV-0.1")

	// 3 個 Model (A:10, B:20, C:30)
	f.SetCellValue("BigMatrix", "H4", "A")
	f.SetCellValue("BigMatrix", "H5", "10")
	f.SetCellValue("BigMatrix", "I4", "B")
	f.SetCellValue("BigMatrix", "I5", "20")
	f.SetCellValue("BigMatrix", "J4", "C")
	f.SetCellValue("BigMatrix", "J5", "30")

	// Part 1: Model A & Model C 勾選 V
	f.SetCellValue("BigMatrix", "A6", "1")
	f.SetCellValue("BigMatrix", "D6", "SupA")
	f.SetCellValue("BigMatrix", "E6", "PN1")
	f.SetCellValue("BigMatrix", "H6", "V") // Model A
	f.SetCellValue("BigMatrix", "I6", "")  // Model B
	f.SetCellValue("BigMatrix", "J6", "V") // Model C

	// Part 2: Model B 勾選 V
	f.SetCellValue("BigMatrix", "A7", "2")
	f.SetCellValue("BigMatrix", "D7", "SupB")
	f.SetCellValue("BigMatrix", "E7", "PN2")
	f.SetCellValue("BigMatrix", "H7", "")  // Model A
	f.SetCellValue("BigMatrix", "I7", "V") // Model B
	f.SetCellValue("BigMatrix", "J7", "")  // Model C

	reader := &BigMatrixReader{db: database}
	if err := reader.Import(wb); err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// 1. 驗證 MatrixModel
	var models []db.MatrixModel
	if err := database.Where("revision_id = ?", revision.ID).Order("sort_order ASC").Find(&models).Error; err != nil {
		t.Fatalf("Failed to query models: %v", err)
	}
	if len(models) != 3 {
		t.Fatalf("Expected 3 models, got %d", len(models))
	}

	if models[0].ModelName != "A" || models[0].Qty != 10 {
		t.Errorf("Model A mismatch: name=%s, qty=%d (expected A, 10)", models[0].ModelName, models[0].Qty)
	}
	if models[1].ModelName != "B" || models[1].Qty != 20 {
		t.Errorf("Model B mismatch: name=%s, qty=%d (expected B, 20)", models[1].ModelName, models[1].Qty)
	}
	if models[2].ModelName != "C" || models[2].Qty != 30 {
		t.Errorf("Model C mismatch: name=%s, qty=%d (expected C, 30)", models[2].ModelName, models[2].Qty)
	}

	// 2. 驗證 MatrixSelections 數量 (應有 3 筆: Part1在Model A與C, Part2在Model B)
	var selections []db.MatrixSelection
	if err := database.Where("revision_id = ?", revision.ID).Find(&selections).Error; err != nil {
		t.Fatalf("Failed to query selections: %v", err)
	}
	if len(selections) != 3 {
		t.Errorf("Expected 3 selections, got %d", len(selections))
	}
}

// TestDDDBomx_ImportExportCycle tests the exact roundtrip scenario with testdata/ddd.bomx
func TestDDDBomx_ImportExportCycle(t *testing.T) {
	// 檢查 testdata/ddd.bomx 是否存在
	dbPath := "testdata/ddd.bomx"
	database, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Skipf("Skipping test: testdata/ddd.bomx not accessible: %v", err)
	}

	// 統計匯入前原始 DB 各 Revision 的 MatrixSelection 筆數
	var revisions []db.BomRevision
	database.Find(&revisions)
	for _, rev := range revisions {
		var cnt int64
		database.Model(&db.MatrixSelection{}).Where("revision_id = ?", rev.ID).Count(&cnt)
		t.Logf("Original DB Revision ID=%d (Phase=%s, Version=%s) Selections: %d", rev.ID, rev.Phase, rev.Version, cnt)
	}

	// 讀取既有 BigMatrix xlsx (若有)
	f, err := excelize.OpenFile("testdata/ddd_BigMatrix_SI1_0.1_20260807.xlsx")
	if err != nil {
		t.Skipf("Skipping import test: testdata/ddd_BigMatrix_SI1_0.1_20260807.xlsx not found: %v", err)
	}
	defer f.Close()

	wb := &ExcelizeWorkbook{f: f}
	reader := &BigMatrixReader{db: database}

	// 執行匯入
	if err := reader.Import(wb); err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// 統計匯入後 DB 各 Revision 的 MatrixSelection 筆數
	for _, rev := range revisions {
		var cnt int64
		database.Model(&db.MatrixSelection{}).Where("revision_id = ?", rev.ID).Count(&cnt)
		t.Logf("After Import DB Revision ID=%d (Phase=%s, Version=%s) Selections: %d", rev.ID, rev.Phase, rev.Version, cnt)
	}
}

// TestInspect_AZ5125 inspects AMAZING_AZ5125-01H.R7G in testdata/ddd.bomx
func TestInspect_AZ5125(t *testing.T) {
	dbPath := "testdata/ddd.bomx"
	database, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Skipf("Skipping test: testdata/ddd.bomx not accessible: %v", err)
	}

	// 1. 查詢 Parts 表中包含 AZ5125 的記錄
	var parts []db.Part
	database.Where("supplier_pn LIKE ?", "%AZ5125%").Find(&parts)
	t.Logf("Parts count matching AZ5125: %d", len(parts))
	partIDs := make([]int64, 0, len(parts))
	for _, p := range parts {
		t.Logf("Part ID=%d, RevID=%d, Supplier=%s, SupplierPN=%s, HHPN=%s", p.ID, p.RevisionID, p.Supplier, p.SupplierPN, p.HHPN)
		partIDs = append(partIDs, p.ID)
	}

	// 2. 查詢 SecondSources 表中關聯此 PartID 的記錄
	var ssources []db.SecondSource
	if len(partIDs) > 0 {
		database.Where("part_id IN ?", partIDs).Find(&ssources)
	}
	t.Logf("SecondSources count for AZ5125 parts: %d", len(ssources))
	for _, ss := range ssources {
		t.Logf("SecondSource ID=%d, PartID=%d, RevID=%d, SS_Supplier=%s, SS_SupplierPN=%s", ss.ID, ss.PartID, ss.RevisionID, ss.Supplier, ss.SupplierPN)
	}

	// 3. 查詢 PartLocations 表中關聯此 PartID 的記錄
	var locs []db.PartLocation
	if len(partIDs) > 0 {
		database.Where("part_id IN ?", partIDs).Find(&locs)
	}
	t.Logf("PartLocations count for AZ5125 parts: %d", len(locs))
	for _, l := range locs {
		t.Logf("PartLocation ID=%d, PartID=%d, Loc=%s, BomStatus=%s, CCL=%v", l.ID, l.PartID, l.Location, l.BomStatus, l.CCL)
	}

	// 4. 呼叫 View Service
	viewSvc := view.NewService(database)
	res, err := viewSvc.Query(view.ViewQuery{
		RevisionIDs: []int64{1, 2, 3},
		ViewType:    "ALL",
	})
	if err != nil {
		t.Fatalf("View query failed: %v", err)
	}

	for _, pg := range res.PartGroups {
		if strings.Contains(pg.MainSupplierPN, "AZ5125") {
			t.Logf("ViewPartGroup: Supplier=%s, MainSupplierPN=%s, SS Count=%d", pg.MainSupplier, pg.MainSupplierPN, len(pg.SecondSources))
			for _, ss := range pg.SecondSources {
				t.Logf("   -> SS Supplier=%s, SS SupplierPN=%s", ss.Supplier, ss.SupplierPN)
			}
		}
	}
}

// TestInspect_AZ5125_ExportedMatrix inspects how AZ5125 is converted for export in app.go logic
func TestInspect_AZ5125_ExportedMatrix(t *testing.T) {
	dbPath := "testdata/ddd.bomx"
	database, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Skipf("Skipping test: testdata/ddd.bomx not accessible: %v", err)
	}

	viewSvc := view.NewService(database)
	res, err := viewSvc.Query(view.ViewQuery{
		RevisionIDs: []int64{1},
		ViewType:    "ALL",
	})
	if err != nil {
		t.Fatalf("View query failed: %v", err)
	}

	// 模擬 app.go 轉換 PartData 過程
	partDataList := make([]PartData, 0, len(res.PartGroups))
	for idx, pg := range res.PartGroups {
		ssData := make([]SecondSourceData, 0, len(pg.SecondSources))
		for _, ss := range pg.SecondSources {
			ssData = append(ssData, SecondSourceData{
				HHPN:              ss.HHPN,
				Supplier:          ss.Supplier,
				SupplierPn:        ss.SupplierPN,
				Description:       ss.Description,
				Remark:            ss.Remark,
				SourceRevisionIDs: ss.SourceRevisionIDs,
			})
		}
		partDataList = append(partDataList, PartData{
			Item:          fmt.Sprintf("%d", idx+1),
			HHPN:          pg.HHPN,
			Description:   pg.Description,
			Supplier:      pg.MainSupplier,
			SupplierPn:    pg.MainSupplierPN,
			Qty:           pg.Qty,
			Location:      pg.Locations,
			Type:          pg.Type,
			BOMStatus:     pg.BOMStatus,
			CCL:           pg.CCL,
			Remark:        pg.Remark,
			SecondSources: ssData,
		})
	}

	// 印出 AZ5125 的 PartData
	for _, pd := range partDataList {
		if strings.Contains(pd.SupplierPn, "AZ5125") {
			t.Logf("PartData: Supplier=%s, SupplierPN=%s, Type=%s, SS Count=%d", pd.Supplier, pd.SupplierPn, pd.Type, len(pd.SecondSources))
			for _, ss := range pd.SecondSources {
				t.Logf("   PartData SS: Supplier=%s, SupplierPN=%s", ss.Supplier, ss.SupplierPn)
			}
		}
	}

	// 呼叫 exportBigMatrixDetailed
	writer, _ := NewWriter(nil)
	revDataList := []RevisionData{
		{
			ID:          "1",
			ProjectCode: "TEST_PROJ",
			Phase:       "SI1",
			Version:     "0.1",
		},
	}
	outPaths, err := writer.exportBigMatrixDetailed(ExportOptions{
		OutputPath: t.TempDir() + "/test_bm.xlsx",
	}, revDataList, partDataList)
	if err != nil {
		t.Fatalf("exportBigMatrixDetailed failed: %v", err)
	}

	fBM, err := excelize.OpenFile(outPaths[0])
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	defer fBM.Close()

	bmRows, _ := fBM.GetRows("BigMatrix")
	t.Logf("BigMatrix Sheet total rows: %d", len(bmRows))
	for rIdx, r := range bmRows {
		if len(r) > 4 {
			supplierPN := r[4]
			if strings.Contains(supplierPN, "AZ5125") || strings.Contains(supplierPN, "SYT21M05ANO") || strings.Contains(supplierPN, "AU0521D5-F") {
				t.Logf("BigMatrix Row %d: Item='%s', Supplier='%s', SupplierPN='%s'", rIdx+1, safeGetCol(r, 0), safeGetCol(r, 3), safeGetCol(r, 4))
			}
		}
	}

	// 呼叫 exportMatrixDetailed
	matrixPaths, err := writer.exportMatrixDetailed(ExportOptions{
		OutputDir: t.TempDir(),
	}, revDataList[0], partDataList)
	if err != nil {
		t.Fatalf("exportMatrixDetailed failed: %v", err)
	}

	fMat, err := excelize.OpenFile(matrixPaths[0])
	if err != nil {
		t.Fatalf("OpenFile Matrix failed: %v", err)
	}
	defer fMat.Close()

	smdRows, _ := fMat.GetRows("SMD")
	t.Logf("SMD Sheet total rows: %d", len(smdRows))
	for rIdx, r := range smdRows {
		if len(r) > 5 {
			supplierPN := r[5]
			if strings.Contains(supplierPN, "AZ5125") || strings.Contains(supplierPN, "SYT21M05ANO") || strings.Contains(supplierPN, "AU0521D5-F") {
				t.Logf("Matrix SMD Row %d: Item='%s', Supplier='%s', SupplierPN='%s'", rIdx+1, safeGetCol(r, 0), safeGetCol(r, 4), safeGetCol(r, 5))
			}
		}
	}
}

// TestAZ5125_FullRoundtripVerify tests export -> import -> export roundtrip for AZ5125 second sources
func TestAZ5125_FullRoundtripVerify(t *testing.T) {
	dbPath := "testdata/ddd.bomx"
	database, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Skipf("Skipping test: testdata/ddd.bomx not accessible: %v", err)
	}

	// 1. 第一次匯出 BigMatrix
	viewSvc := view.NewService(database)
	res1, err := viewSvc.Query(view.ViewQuery{RevisionIDs: []int64{1}, ViewType: "ALL"})
	if err != nil {
		t.Fatalf("View query 1 failed: %v", err)
	}

	partDataList1 := convertToPartDataList(res1.PartGroups)
	writer, _ := NewWriter(nil)
	revData1 := RevisionData{ID: "1", ProjectCode: "TEST", Phase: "SI1", Version: "0.1"}

	tmpDir := t.TempDir()
	bmPath1, err := writer.exportBigMatrixDetailed(ExportOptions{OutputPath: tmpDir + "/BM1.xlsx"}, []RevisionData{revData1}, partDataList1)
	if err != nil {
		t.Fatalf("Export BM1 failed: %v", err)
	}

	// 2. 執行匯入 BM1 到 DB
	fBM1, _ := excelize.OpenFile(bmPath1[0])
	wb1 := &ExcelizeWorkbook{f: fBM1}
	reader := &BigMatrixReader{db: database}
	if err := reader.Import(wb1); err != nil {
		t.Fatalf("Import BM1 failed: %v", err)
	}
	fBM1.Close()

	// 3. 第二次從 DB 匯出 BigMatrix 與 Matrix
	res2, err := viewSvc.Query(view.ViewQuery{RevisionIDs: []int64{1}, ViewType: "ALL"})
	if err != nil {
		t.Fatalf("View query 2 failed: %v", err)
	}

	partDataList2 := convertToPartDataList(res2.PartGroups)
	bmPath2, err := writer.exportBigMatrixDetailed(ExportOptions{OutputPath: tmpDir + "/BM2.xlsx"}, []RevisionData{revData1}, partDataList2)
	if err != nil {
		t.Fatalf("Export BM2 failed: %v", err)
	}

	matPath2, err := writer.exportMatrixDetailed(ExportOptions{OutputDir: tmpDir}, revData1, partDataList2)
	if err != nil {
		t.Fatalf("Export Matrix2 failed: %v", err)
	}

	// 4. 檢驗 BM2 中 AZ5125 的 2nd sources 數量
	fBM2, _ := excelize.OpenFile(bmPath2[0])
	defer fBM2.Close()
	bm2Rows, _ := fBM2.GetRows("BigMatrix")

	az2ndSourceCountBM2 := 0
	for _, r := range bm2Rows {
		if len(r) > 4 {
			supplierPN := r[4]
			item := safeGetCol(r, 0)
			if item == "" && (strings.Contains(supplierPN, "SYT21M05ANO") || strings.Contains(supplierPN, "AU0521D5-F") || strings.Contains(supplierPN, "LESD5Z5") || strings.Contains(supplierPN, "WE1119K95") || strings.Contains(supplierPN, "ESD5471S")) {
				az2ndSourceCountBM2++
			}
		}
	}
	t.Logf("BM2 AZ5125 2nd source rows count: %d", az2ndSourceCountBM2)

	// 5. 檢驗 Matrix2 中 AZ5125 的 2nd sources 數量
	fMat2, _ := excelize.OpenFile(matPath2[0])
	defer fMat2.Close()
	mat2Rows, _ := fMat2.GetRows("SMD")

	az2ndSourceCountMat2 := 0
	for _, r := range mat2Rows {
		if len(r) > 5 {
			supplierPN := r[5]
			item := safeGetCol(r, 0)
			if item == "" && (strings.Contains(supplierPN, "SYT21M05ANO") || strings.Contains(supplierPN, "AU0521D5-F") || strings.Contains(supplierPN, "LESD5Z5") || strings.Contains(supplierPN, "WE1119K95") || strings.Contains(supplierPN, "ESD5471S")) {
				az2ndSourceCountMat2++
			}
		}
	}
	t.Logf("Matrix2 AZ5125 2nd source rows count: %d", az2ndSourceCountMat2)

	if az2ndSourceCountBM2 == 0 || az2ndSourceCountMat2 == 0 {
		t.Errorf("AZ5125 2nd sources lost after roundtrip! BM2 count=%d, Matrix2 count=%d", az2ndSourceCountBM2, az2ndSourceCountMat2)
	}
}

// convertToPartDataList 輔助函數
func convertToPartDataList(groups []view.ViewPartGroup) []PartData {
	list := make([]PartData, 0, len(groups))
	for idx, pg := range groups {
		ssData := make([]SecondSourceData, 0, len(pg.SecondSources))
		for _, ss := range pg.SecondSources {
			ssData = append(ssData, SecondSourceData{
				HHPN:              ss.HHPN,
				Supplier:          ss.Supplier,
				SupplierPn:        ss.SupplierPN,
				Description:       ss.Description,
				Remark:            ss.Remark,
				SourceRevisionIDs: ss.SourceRevisionIDs,
			})
		}
		list = append(list, PartData{
			Item:          fmt.Sprintf("%d", idx+1),
			HHPN:          pg.HHPN,
			Description:   pg.Description,
			Supplier:      pg.MainSupplier,
			SupplierPn:    pg.MainSupplierPN,
			Qty:           pg.Qty,
			Location:      pg.Locations,
			Type:          pg.Type,
			BOMStatus:     pg.BOMStatus,
			CCL:           pg.CCL,
			Remark:        pg.Remark,
			SecondSources: ssData,
		})
	}
	return list
}











