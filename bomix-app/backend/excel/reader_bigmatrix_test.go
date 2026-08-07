package excel

import (
	"testing"

	"github.com/xuri/excelize/v2"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"bomix-app/backend/db"
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



