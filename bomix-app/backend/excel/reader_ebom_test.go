package excel

import (
	"testing"

	"github.com/xuri/excelize/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"bomix-app/backend/db"
)

// TestParseHeader tests header parsing from EBOM format
func TestParseHeader(t *testing.T) {
	f := excelize.NewFile()
	wb := &ExcelizeWorkbook{f: f}
	_ = wb
	defer f.Close()

	f.NewSheet("SMD")

	// Set header values according to product-spec 7.1.1
	f.SetCellValue("SMD", "B3", "Product Code: TANGLED")
	f.SetCellValue("SMD", "B4", "Description: MBD,Tangled,Test Board")
	f.SetCellValue("SMD", "D3", "Schematic Version: 1.0")
	f.SetCellValue("SMD", "J3", "Phase: PV")
	f.SetCellValue("SMD", "F3", "PCB Version: 2.1")
	f.SetCellValue("SMD", "F4", "PCA PN: ABC-123")
	f.SetCellValue("SMD", "H3", "BOM Version: 0.3")
	f.SetCellValue("SMD", "H4", "Date: 2026-01-15")

	reader := &EBOMReader{}
	phase, version, description, schematicVersion, pcbVersion, pcaPn, date, projectCode, err := reader.parseHeader(wb, "SMD")

	if err != nil {
		t.Fatalf("parseHeader failed: %v", err)
	}

	if projectCode != "TANGLED" {
		t.Errorf("Expected projectCode 'TANGLED', got '%s'", projectCode)
	}
	if description != "MBD,Tangled,Test Board" {
		t.Errorf("Expected description 'MBD,Tangled,Test Board', got '%s'", description)
	}
	if schematicVersion != "1.0" {
		t.Errorf("Expected schematicVersion '1.0', got '%s'", schematicVersion)
	}
	if phase != "PV" {
		t.Errorf("Expected phase 'PV', got '%s'", phase)
	}
	if pcbVersion != "2.1" {
		t.Errorf("Expected pcbVersion '2.1', got '%s'", pcbVersion)
	}
	if pcaPn != "ABC-123" {
		t.Errorf("Expected pcaPn 'ABC-123', got '%s'", pcaPn)
	}
	if version != "0.3" {
		t.Errorf("Expected version '0.3', got '%s'", version)
	}
	if date != "2026-01-15" {
		t.Errorf("Expected date '2026-01-15', got '%s'", date)
	}
}

// TestParseHeaderVariations tests header parsing with missing colon spaces or direct values
func TestParseHeaderVariations(t *testing.T) {
	tests := []struct {
		name          string
		rawPhase      string
		expectedPhase string
	}{
		{"With Space J3", "Phase: EVT", "EVT"},
		{"Without Space J3", "Phase:DVT", "DVT"},
		{"Multiple Spaces J3", "Phase:   PV", "PV"},
		{"Case Variation J3", "phase: MP", "MP"},
		{"Raw Value Without Prefix J3", "EVT", "EVT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := excelize.NewFile()
			wb := &ExcelizeWorkbook{f: f}
			defer f.Close()

			f.NewSheet("SMD")
			f.SetCellValue("SMD", "J3", tt.rawPhase)

			reader := &EBOMReader{}
			phase, _, _, _, _, _, _, _, err := reader.parseHeader(wb, "SMD")
			if err != nil {
				t.Fatalf("parseHeader failed: %v", err)
			}
			if phase != tt.expectedPhase {
				t.Errorf("Expected phase '%s', got '%s'", tt.expectedPhase, phase)
			}
		})
	}
}

// TestMainVsSecondSource tests Main Source vs 2nd Source determination
func TestMainVsSecondSource(t *testing.T) {
	f := excelize.NewFile()
	wb := &ExcelizeWorkbook{f: f}
	_ = wb
	defer f.Close()

	f.NewSheet("SMD")

	// Set header
	f.SetCellValue("SMD", "H5", "Qty")
	f.SetCellValue("SMD", "J7", "CCL")

	// Row 6: Main Source (has item number)
	f.SetCellValue("SMD", "A6", "1")
	f.SetCellValue("SMD", "B6", "HHPN-001")
	f.SetCellValue("SMD", "E6", "CAPACITOR 10uF")
	f.SetCellValue("SMD", "F6", "Samsung")
	f.SetCellValue("SMD", "G6", "CL10A106MQ8NNNC")
	f.SetCellValue("SMD", "I6", "C1,C2,C3")
	f.SetCellValue("SMD", "J6", "Y")

	// Row 7: 2nd Source (empty item, follows Main Source)
	f.SetCellValue("SMD", "A7", "") // Empty item
	f.SetCellValue("SMD", "B7", "HHPN-001")
	f.SetCellValue("SMD", "E7", "CAPACITOR 10uF Alt")
	f.SetCellValue("SMD", "F7", "Murata")
	f.SetCellValue("SMD", "G7", "GRM188R61A106KE15D")
	f.SetCellValue("SMD", "I7", "")
	f.SetCellValue("SMD", "J7", "N")

	reader := &EBOMReader{}
	parts, secondSources := reader.parseSheet(wb, "SMD", "SMD")

	if len(parts) != 1 {
		t.Errorf("Expected 1 Main Source part, got %d", len(parts))
	}
	if len(secondSources) != 1 {
		t.Errorf("Expected 1 Second Source, got %d", len(secondSources))
	}

	if parts[0].Supplier != "Samsung" {
		t.Errorf("Expected Main Source supplier 'Samsung', got '%s'", parts[0].Supplier)
	}
	if secondSources[0].secondSource.Supplier != "Murata" {
		t.Errorf("Expected Second Source supplier 'Murata', got '%s'", secondSources[0].secondSource.Supplier)
	}
}

// TestDetermineMode_NPI tests NPI mode detection
func TestDetermineMode_NPI(t *testing.T) {
	f := excelize.NewFile()
	wb := &ExcelizeWorkbook{f: f}
	_ = wb
	defer f.Close()

	// Create required sheets
	for _, sheet := range []string{"SMD", "PTH", "BOTTOM", "PROTO"} {
		f.NewSheet(sheet)
	}

	// Set header for EBOM format
	f.SetCellValue("SMD", "H5", "Qty")
	f.SetCellValue("SMD", "J7", "CCL")

	// Add a part in SMD with location C1,C2
	f.SetCellValue("SMD", "A6", "1")
	f.SetCellValue("SMD", "F6", "Samsung")
	f.SetCellValue("SMD", "G6", "CAP-001")
	f.SetCellValue("SMD", "I6", "C1,C2")

	// Add a part in PROTO with overlapping location C1
	f.SetCellValue("PROTO", "A6", "1")
	f.SetCellValue("PROTO", "F6", "Samsung")
	f.SetCellValue("PROTO", "G6", "CAP-001")
	f.SetCellValue("PROTO", "I6", "C1,C3")

	reader := &EBOMReader{}
	mode := reader.determineMode(wb, []string{"SMD", "PTH", "BOTTOM", "PROTO"})

	if mode != "NPI" {
		t.Errorf("Expected mode 'NPI' (due to location overlap), got '%s'", mode)
	}
}

// TestDetermineMode_MP tests MP mode detection
func TestDetermineMode_MP(t *testing.T) {
	f := excelize.NewFile()
	wb := &ExcelizeWorkbook{f: f}
	_ = wb
	defer f.Close()

	// Create required sheets
	for _, sheet := range []string{"SMD", "PTH", "BOTTOM", "PROTO"} {
		f.NewSheet(sheet)
	}

	// Set header for EBOM format
	f.SetCellValue("SMD", "H5", "Qty")
	f.SetCellValue("SMD", "J7", "CCL")

	// Add a part in SMD with location C1,C2
	f.SetCellValue("SMD", "A6", "1")
	f.SetCellValue("SMD", "F6", "Samsung")
	f.SetCellValue("SMD", "G6", "CAP-001")
	f.SetCellValue("SMD", "I6", "C1,C2")

	// Add a part in PROTO with NO overlapping location
	f.SetCellValue("PROTO", "A6", "1")
	f.SetCellValue("PROTO", "F6", "Samsung")
	f.SetCellValue("PROTO", "G6", "CAP-002")
	f.SetCellValue("PROTO", "I6", "C10,C11")

	reader := &EBOMReader{}
	mode := reader.determineMode(wb, []string{"SMD", "PTH", "BOTTOM", "PROTO"})

	if mode != "MP" {
		t.Errorf("Expected mode 'MP' (no location overlap), got '%s'", mode)
	}
}

// TestDetermineMode_NoProto tests MP mode when no PROTO sheet exists
func TestDetermineMode_NoProto(t *testing.T) {
	f := excelize.NewFile()
	wb := &ExcelizeWorkbook{f: f}
	_ = wb
	defer f.Close()

	// Create only SMD sheet
	f.NewSheet("SMD")
	f.SetCellValue("SMD", "H5", "Qty")
	f.SetCellValue("SMD", "J7", "CCL")

	reader := &EBOMReader{}
	mode := reader.determineMode(wb, []string{"SMD"})

	if mode != "MP" {
		t.Errorf("Expected mode 'MP' (no PROTO sheet), got '%s'", mode)
	}
}

// TestAtomizeLocation tests location atomization
func TestAtomizeLocation(t *testing.T) {
	reader := &EBOMReader{}

	tests := []struct {
		input    string
		expected string
	}{
		{"C1,C2,C3", "C1,C2,C3"},
		{"C1, C2, C3", "C1,C2,C3"},
		{"C1  ,  C2  ,  C3", "C1,C2,C3"},
		{"C1", "C1"},
		{"", ""},
		{"C1,C2, C3,C4", "C1,C2,C3,C4"},
	}

	for _, test := range tests {
		result := reader.atomizeLocation(test.input)
		if result != test.expected {
			t.Errorf("atomizeLocation(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

// TestMergeAlgorithm tests the EBOM merge algorithm (product-spec 7.1.7)
func TestMergeAlgorithm(t *testing.T) {
	// Create in-memory database
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect database: %v", err)
	}

	// Auto migrate
	if err := db.AutoMigrate(database); err != nil {
		t.Fatalf("Failed to auto migrate: %v", err)
	}

	// Create test data
	// Setup: Create project and revision
	project := db.Project{
		Code:        "TEST",
		Description: "Test Project",
	}
	if err := database.Create(&project).Error; err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	revision := db.BomRevision{
		ProjectID: project.ID,
		Phase:     "PV",
		Version:   "0.2",
		Mode:      "NPI",
	}
	if err := database.Create(&revision).Error; err != nil {
		t.Fatalf("Failed to create revision: %v", err)
	}

	// Create old parts and second sources (simulating existing database state)
	mainPart := db.Part{
		RevisionID: revision.ID,
		Type:       "Main",
		Supplier:   "SupplierX",
		SupplierPN: "PN-001",
		Description: "Main Part P1",
	}
	if err := database.Create(&mainPart).Error; err != nil {
		t.Fatalf("Failed to create main part: %v", err)
	}

	oldSecondSources := []db.SecondSource{
		{
			RevisionID: revision.ID,
			PartID:     mainPart.ID,
			Supplier:   "SupplierY",
			SupplierPN: "PN-002",
			Description: "Old 2nd Source P2 (will be removed)",
		},
		{
			RevisionID: revision.ID,
			PartID:     mainPart.ID,
			Supplier:   "SupplierZ",
			SupplierPN: "PN-003",
			Description: "Old 2nd Source P3 (will remain)",
		},
	}
	for _, ss := range oldSecondSources {
		if err := database.Create(&ss).Error; err != nil {
			t.Fatalf("Failed to create second source: %v", err)
		}
	}

	// Create MatrixModels
	modelA := db.MatrixModel{
		RevisionID: revision.ID,
		ModelName:  "A",
		Qty:        1,
	}
	if err := database.Create(&modelA).Error; err != nil {
		t.Fatalf("Failed to create model A: %v", err)
	}

	modelB := db.MatrixModel{
		RevisionID: revision.ID,
		ModelName:  "B",
		Qty:        1,
	}
	if err := database.Create(&modelB).Error; err != nil {
		t.Fatalf("Failed to create model B: %v", err)
	}

	modelC := db.MatrixModel{
		RevisionID: revision.ID,
		ModelName:  "C",
		Qty:        1,
	}
	if err := database.Create(&modelC).Error; err != nil {
		t.Fatalf("Failed to create model C: %v", err)
	}

	// Create MatrixSelections (before merge)
	selections := []db.MatrixSelection{
		{
			RevisionID:         revision.ID,
			ModelID:            modelA.ID,
			PartID:             mainPart.ID,
			Group:              "SupplierX|PN-001",
			Material:           "SupplierX|PN-001",
			SelectedSupplier:   "SupplierX",
			SelectedSupplierPn: "PN-001",
		},
		{
			RevisionID:         revision.ID,
			ModelID:            modelB.ID,
			PartID:             mainPart.ID,
			Group:              "SupplierX|PN-001",
			Material:           "SupplierY|PN-002",
			SelectedSupplier:   "SupplierY",
			SelectedSupplierPn: "PN-002",
		},
		{
			RevisionID:         revision.ID,
			ModelID:            modelC.ID,
			PartID:             mainPart.ID,
			Group:              "SupplierX|PN-001",
			Material:           "SupplierZ|PN-003",
			SelectedSupplier:   "SupplierZ",
			SelectedSupplierPn: "PN-003",
		},
	}
	for _, sel := range selections {
		if err := database.Create(&sel).Error; err != nil {
			t.Fatalf("Failed to create selection: %v", err)
		}
	}

	// Now simulate new EBOM data (after merge)
	// New second sources: P3 (kept), P4 (new)
	newSecondSources := []db.SecondSource{
		{
			RevisionID: revision.ID,
			PartID:     mainPart.ID,
			Supplier:   "SupplierZ",
			SupplierPN: "PN-003",
			Description: "Old 2nd Source P3 (updated)",
		},
		{
			RevisionID: revision.ID,
			PartID:     mainPart.ID,
			Supplier:   "SupplierW",
			SupplierPN: "PN-004",
			Description: "New 2nd Source P4",
		},
	}

	// Apply merge algorithm
	reader := &EBOMReader{db: database}
	if err := reader.applyMergeAlgorithm(revision.ID, newSecondSources); err != nil {
		t.Fatalf("applyMergeAlgorithm failed: %v", err)
	}

	// Verify results
	// Check second sources
	var remainingSecondSources []db.SecondSource
	if err := database.Where("revision_id = ?", revision.ID).Find(&remainingSecondSources).Error; err != nil {
		t.Fatalf("Failed to query second sources: %v", err)
	}

	if len(remainingSecondSources) != 2 {
		t.Errorf("Expected 2 second sources after merge, got %d", len(remainingSecondSources))
	}

	// Check that P2 was deleted
	var p2Count int64
	database.Model(&db.SecondSource{}).Where("supplier = ? AND supplier_pn = ?", "SupplierY", "PN-002").Count(&p2Count)
	if p2Count != 0 {
		t.Error("Expected P2 (SupplierY|PN-002) to be deleted")
	}

	// Check that P4 was added
	var p4Count int64
	database.Model(&db.SecondSource{}).Where("supplier = ? AND supplier_pn = ?", "SupplierW", "PN-004").Count(&p4Count)
	if p4Count != 1 {
		t.Error("Expected P4 (SupplierW|PN-004) to be added")
	}

	// Check MatrixSelections
	// Model B selection (pointing to P2 which was deleted) should be removed
	var modelBSelections []db.MatrixSelection
	if err := database.Where("revision_id = ? AND model_id = ?", revision.ID, modelB.ID).Find(&modelBSelections).Error; err != nil {
		t.Fatalf("Failed to query Model B selections: %v", err)
	}

	if len(modelBSelections) != 0 {
		t.Errorf("Expected Model B selection to be removed (P2 deleted), got %d selections", len(modelBSelections))
	}

	// Model A and Model C selections should remain
	var modelASelections []db.MatrixSelection
	if err := database.Where("revision_id = ? AND model_id = ?", revision.ID, modelA.ID).Find(&modelASelections).Error; err != nil {
		t.Fatalf("Failed to query Model A selections: %v", err)
	}

	if len(modelASelections) != 1 {
		t.Errorf("Expected Model A selection to remain, got %d selections", len(modelASelections))
	}

	var modelCSelections []db.MatrixSelection
	if err := database.Where("revision_id = ? AND model_id = ?", revision.ID, modelC.ID).Find(&modelCSelections).Error; err != nil {
		t.Fatalf("Failed to query Model C selections: %v", err)
	}

	if len(modelCSelections) != 1 {
		t.Errorf("Expected Model C selection to remain, got %d selections", len(modelCSelections))
	}
}

// TestImportMatrixSelections tests the Matrix selection import functionality
func TestImportMatrixSelections(t *testing.T) {
	// Create in-memory database
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect database: %v", err)
	}

	// Auto migrate
	if err := db.AutoMigrate(database); err != nil {
		t.Fatalf("Failed to auto migrate: %v", err)
	}

	// Create source revision
	sourceProject := db.Project{
		Code:        "TEST",
		Description: "Test Project",
	}
	if err := database.Create(&sourceProject).Error; err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	sourceRevision := db.BomRevision{
		ProjectID: sourceProject.ID,
		Phase:     "PV",
		Version:   "0.1",
		Mode:      "NPI",
	}
	if err := database.Create(&sourceRevision).Error; err != nil {
		t.Fatalf("Failed to create source revision: %v", err)
	}

	// Create target revision
	targetRevision := db.BomRevision{
		ProjectID: sourceProject.ID,
		Phase:     "PV",
		Version:   "0.2",
		Mode:      "NPI",
	}
	if err := database.Create(&targetRevision).Error; err != nil {
		t.Fatalf("Failed to create target revision: %v", err)
	}

	// Create parts in source revision
	sourcePart := db.Part{
		RevisionID: sourceRevision.ID,
		Type:       "Main",
		Supplier:   "Samsung",
		SupplierPN: "CL10A106MQ8NNNC",
		Description: "CAPACITOR 10uF",
	}
	if err := database.Create(&sourcePart).Error; err != nil {
		t.Fatalf("Failed to create source part: %v", err)
	}

	// Create MatrixModels in source revision
	sourceModelA := db.MatrixModel{
		RevisionID: sourceRevision.ID,
		ModelName:  "A",
		Qty:        1,
	}
	sourceModelB := db.MatrixModel{
		RevisionID: sourceRevision.ID,
		ModelName:  "B",
		Qty:        2,
	}
	if err := database.Create(&sourceModelA).Error; err != nil {
		t.Fatalf("Failed to create source model A: %v", err)
	}
	if err := database.Create(&sourceModelB).Error; err != nil {
		t.Fatalf("Failed to create source model B: %v", err)
	}

	// Create MatrixSelections in source revision
	sourceSelections := []db.MatrixSelection{
		{
			RevisionID:         sourceRevision.ID,
			ModelID:            sourceModelA.ID,
			PartID:             sourcePart.ID,
			Group:              "Samsung|CL10A106MQ8NNNC",
			Material:           "Samsung|CL10A106MQ8NNNC",
			SelectedSupplier:   "Samsung",
			SelectedSupplierPn: "CL10A106MQ8NNNC",
		},
		{
			RevisionID:         sourceRevision.ID,
			ModelID:            sourceModelB.ID,
			PartID:             sourcePart.ID,
			Group:              "Samsung|CL10A106MQ8NNNC",
			Material:           "Samsung|CL10A106MQ8NNNC",
			SelectedSupplier:   "Samsung",
			SelectedSupplierPn: "CL10A106MQ8NNNC",
		},
	}
	for _, sel := range sourceSelections {
		if err := database.Create(&sel).Error; err != nil {
			t.Fatalf("Failed to create source selection: %v", err)
		}
	}

	// Create matching part in target revision
	targetPart := db.Part{
		RevisionID: targetRevision.ID,
		Type:       "Main",
		Supplier:   "Samsung",
		SupplierPN: "CL10A106MQ8NNNC",
		Description: "CAPACITOR 10uF",
	}
	if err := database.Create(&targetPart).Error; err != nil {
		t.Fatalf("Failed to create target part: %v", err)
	}

	// Create MatrixModels in target revision
	targetModelA := db.MatrixModel{
		RevisionID: targetRevision.ID,
		ModelName:  "A",
		Qty:        1,
	}
	targetModelB := db.MatrixModel{
		RevisionID: targetRevision.ID,
		ModelName:  "B",
		Qty:        2,
	}
	targetModelC := db.MatrixModel{
		RevisionID: targetRevision.ID,
		ModelName:  "C",
		Qty:        3,
	}
	if err := database.Create(&targetModelA).Error; err != nil {
		t.Fatalf("Failed to create target model A: %v", err)
	}
	if err := database.Create(&targetModelB).Error; err != nil {
		t.Fatalf("Failed to create target model B: %v", err)
	}
	if err := database.Create(&targetModelC).Error; err != nil {
		t.Fatalf("Failed to create target model C: %v", err)
	}

	// Import Matrix selections from source to target
	if err := db.ImportMatrixSelections(database, sourceRevision.ID, targetRevision.ID); err != nil {
		t.Fatalf("ImportMatrixSelections failed: %v", err)
	}

	// Verify target selections
	var targetSelections []db.MatrixSelection
	if err := database.Where("revision_id = ?", targetRevision.ID).Find(&targetSelections).Error; err != nil {
		t.Fatalf("Failed to query target selections: %v", err)
	}

	// Should have 2 selections (Model A and Model B)
	if len(targetSelections) != 2 {
		t.Errorf("Expected 2 target selections, got %d", len(targetSelections))
	}

	// Check that Model C was not imported (no corresponding source selection)
	var modelCSelections []db.MatrixSelection
	if err := database.Where("revision_id = ? AND model_id = ?", targetRevision.ID, targetModelC.ID).Find(&modelCSelections).Error; err != nil {
		t.Fatalf("Failed to query Model C selections: %v", err)
	}

	if len(modelCSelections) != 0 {
		t.Errorf("Expected no Model C selections, got %d", len(modelCSelections))
	}

	// Check that selections are marked as auto-selected
	for _, sel := range targetSelections {
		if !sel.IsAutoSelected {
			t.Errorf("Expected selection to be marked as auto-selected")
		}
	}
}

// TestParsePartRow tests part row parsing
func TestParsePartRow(t *testing.T) {
	reader := &EBOMReader{}

	row := []string{
		"1",              // A: Item
		"HHPN-001",       // B: HHPN
		"",               // C: (unused)
		"",               // D: (unused)
		"CAPACITOR 10uF", // E: Description
		"Samsung",        // F: Supplier
		"CL10A106MQ8NNNC", // G: Supplier PN
		"",               // H: (unused)
		"C1,C2,C3",      // I: Location
		"Y",              // J: CCL
		"",               // K: (unused)
		"Note here",      // L: Remark
	}

	part := reader.parsePartRow(row, "SMD")

	// Note: Item and Hhpn are not stored in the Part model anymore
	if part.Description != "CAPACITOR 10uF" {
		t.Errorf("Expected Description 'CAPACITOR 10uF', got '%s'", part.Description)
	}
	if part.Supplier != "Samsung" {
		t.Errorf("Expected Supplier 'Samsung', got '%s'", part.Supplier)
	}
	if part.SupplierPN != "CL10A106MQ8NNNC" {
		t.Errorf("Expected SupplierPN 'CL10A106MQ8NNNC', got '%s'", part.SupplierPN)
	}
	if part.Location != "C1,C2,C3" {
		t.Errorf("Expected Location 'C1,C2,C3', got '%s'", part.Location)
	}
	if part.CCL != "Y" {
		t.Errorf("Expected CCL 'Y', got '%s'", part.CCL)
	}
	if part.Type != "SMD" {
		t.Errorf("Expected Type 'SMD', got '%s'", part.Type)
	}
	if part.BOMStatus != "I" {
		t.Errorf("Expected BOMStatus 'I', got '%s'", part.BOMStatus)
	}
}

// TestParseStatusSheet tests parsing of NI/PROTO/MP sheets
func TestParseStatusSheet(t *testing.T) {
	f := excelize.NewFile()
	wb := &ExcelizeWorkbook{f: f}
	_ = wb
	defer f.Close()

	f.NewSheet("NI")

	// Set header
	f.SetCellValue("NI", "H5", "Qty")
	f.SetCellValue("NI", "J7", "CCL")

	// Add some parts in NI sheet starting from row 6
	f.SetCellValue("NI", "A6", "1")
	f.SetCellValue("NI", "B6", "HHPN-001")
	f.SetCellValue("NI", "E6", "CAPACITOR")
	f.SetCellValue("NI", "F6", "Samsung")
	f.SetCellValue("NI", "G6", "CL10A106MQ8NNNC")
	f.SetCellValue("NI", "I6", "C1,C2")
	f.SetCellValue("NI", "J6", "Y")

	f.SetCellValue("NI", "A7", "2")
	f.SetCellValue("NI", "B7", "HHPN-002")
	f.SetCellValue("NI", "E7", "RESISTOR")
	f.SetCellValue("NI", "F7", "Yageo")
	f.SetCellValue("NI", "G7", "RC0402FR-0710KL")
	f.SetCellValue("NI", "I7", "R1,R2")
	f.SetCellValue("NI", "J7", "N")

	reader := &EBOMReader{}
	parts := reader.parseStatusSheet(wb, "NI", "X", "")

	if len(parts) != 2 {
		t.Errorf("Expected 2 parts, got %d", len(parts))
	}

	for _, part := range parts {
		if part.BOMStatus != "X" {
			t.Errorf("Expected BOMStatus 'X', got '%s'", part.BOMStatus)
		}
	}
}
