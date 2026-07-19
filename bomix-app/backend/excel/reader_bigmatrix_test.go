package excel

import (
	"testing"

	"github.com/xuri/excelize/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"bomix-app/backend/db"
	"bomix-app/backend/types"
)

// TestParseHeader_BigMatrix tests BigMatrix header parsing
func TestParseHeader_BigMatrix(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()

	f.NewSheet("BigMatrix")

	// Set header values according to product-spec 7.2.2
	f.SetCellValue("BigMatrix", "B3", "BOMs: 4")
	f.SetCellValue("BigMatrix", "B4", "Description: MBD,Tangled,Multi-Board")
	f.SetCellValue("BigMatrix", "E4", "Date: 2026-01-15")

	reader := &BigMatrixReader{}
	description, bomCount, date := reader.parseHeader(f, "BigMatrix")

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
	configs, err := reader.parseBOMConfigs(f, "BigMatrix")

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

	// Create project first
	project := db.Project{
		Code:        "TEST_PROJECT",
		Description: "Test Project",
	}
	if err := database.Create(&project).Error; err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Create revision
	revision := db.BomRevision{
		ProjectID: project.ID,
		Phase:     "PV",
		Version:   "0.1",
	}
	if err := database.Create(&revision).Error; err != nil {
		t.Fatalf("Failed to create revision: %v", err)
	}

	// Create test Excel file
	f := excelize.NewFile()
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
		db:     database,
		result: &types.ImportResult{FileName: "test.xlsx"},
	}

	err = reader.Import(f)
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

	// Create project and revision
	project := db.Project{
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
		db:     database,
		result: &types.ImportResult{FileName: "test.xlsx"},
	}

	err = reader.Import(f)
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
