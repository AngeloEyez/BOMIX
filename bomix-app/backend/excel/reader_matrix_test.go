package excel

import (
	"fmt"
	"testing"
	"time"

	"bomix-app/backend/db"
	"bomix-app/backend/types"

	"github.com/glebarez/sqlite"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// setupMatrixTestDB 建立記憶體 SQLite 供 Matrix 測試使用
func setupMatrixTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dbName := fmt.Sprintf("file:mem_%d_%p?mode=memory&cache=shared", time.Now().UnixNano(), t)
	gdb, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open memory db: %v", err)
	}

	err = db.AutoMigrate(gdb)
	if err != nil {
		t.Fatalf("failed to migrate DB: %v", err)
	}

	// 模擬系統預設建置單一 Series
	var count int64
	gdb.Model(&db.Series{}).Count(&count)
	if count == 0 {
		gdb.Create(&db.Series{Name: "Default Series"})
	}

	return gdb
}

// TestParseHeader_Matrix 測試 Matrix 表頭解析
func TestParseHeader_Matrix(t *testing.T) {
	f := excelize.NewFile()
	wb := &ExcelizeWorkbook{f: f}
	defer f.Close()

	smdSheet := "SMD"
	f.NewSheet(smdSheet)

	f.SetCellValue(smdSheet, "B3", "Product Code: PROJ_TEST")
	f.SetCellValue(smdSheet, "B4", "Description: Test Matrix Description")
	f.SetCellValue(smdSheet, "D3", "Schematic Version: S1.0")
	f.SetCellValue(smdSheet, "F3", "PCB Version: P1.0")
	f.SetCellValue(smdSheet, "F4", "PCA PN: PCA12345")
	f.SetCellValue(smdSheet, "H3", "BOM Version: 0.1")
	f.SetCellValue(smdSheet, "H4", "Date: 2026-08-06")
	f.SetCellValue(smdSheet, "J3", "Phase: EVT")

	reader := &MatrixReader{}
	phase, version, description, schematicVersion, pcbVersion, pcaPn, date, projectCode, err := reader.parseHeader(wb, smdSheet)
	if err != nil {
		t.Fatalf("parseHeader error: %v", err)
	}

	if phase != "EVT" || version != "0.1" || projectCode != "PROJ_TEST" {
		t.Errorf("parseHeader unexpected: phase=%s, version=%s, proj=%s", phase, version, projectCode)
	}
	if description != "Test Matrix Description" || schematicVersion != "S1.0" || pcbVersion != "P1.0" || pcaPn != "PCA12345" || date != "2026-08-06" {
		t.Errorf("parseHeader details unexpected: desc=%s, sch=%s, pcb=%s, pca=%s, date=%s", description, schematicVersion, pcbVersion, pcaPn, date)
	}
}

// TestParseValidModels_Matrix 測試 Model 欄位與數量解析
func TestParseValidModels_Matrix(t *testing.T) {
	f := excelize.NewFile()
	wb := &ExcelizeWorkbook{f: f}
	defer f.Close()

	smdSheet := "SMD"
	f.NewSheet(smdSheet)

	// SMD 工作表: Model A (qty 10), Model B (qty 20)
	f.SetCellValue(smdSheet, "K4", "Model_A")
	f.SetCellValue(smdSheet, "K5", "10")
	f.SetCellValue(smdSheet, "L4", "Model_B")
	f.SetCellValue(smdSheet, "L5", "20")

	reader := &MatrixReader{}
	models, err := reader.parseValidModels(wb, smdSheet)
	if err != nil {
		t.Fatalf("parseValidModels error: %v", err)
	}

	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}

	modelMap := make(map[string]validModelInfo)
	for _, m := range models {
		modelMap[m.ModelName] = m
	}

	if mA, ok := modelMap["Model_A"]; !ok || mA.Qty != 10 {
		t.Errorf("Model_A mismatch: %+v", mA)
	}
	if mB, ok := modelMap["Model_B"]; !ok || mB.Qty != 20 {
		t.Errorf("Model_B mismatch: %+v", mB)
	}
}

// TestParseValidModels_Matrix_Filters 測試 Matrix Model 數量與 Qty 篩選
func TestParseValidModels_Matrix_Filters(t *testing.T) {
	f := excelize.NewFile()
	wb := &ExcelizeWorkbook{f: f}
	defer f.Close()

	smdSheet := "SMD"
	f.NewSheet(smdSheet)

	// K4: Model A, K5: 1 (Valid)
	f.SetCellValue(smdSheet, "K4", "Model A")
	f.SetCellValue(smdSheet, "K5", "1")

	// L4: Model B, L5: 0 (Invalid - qty <= 0)
	f.SetCellValue(smdSheet, "L4", "Model B")
	f.SetCellValue(smdSheet, "L5", "0")

	// M4: Model C, M5: "" (Invalid - empty)
	f.SetCellValue(smdSheet, "M4", "Model C")
	f.SetCellValue(smdSheet, "M5", "")

	// N4: Model D, N5: 3 (Valid)
	f.SetCellValue(smdSheet, "N4", "Model D")
	f.SetCellValue(smdSheet, "N5", "3")

	reader := &MatrixReader{}
	validModels, err := reader.parseValidModels(wb, smdSheet)
	if err != nil {
		t.Fatalf("parseValidModels error: %v", err)
	}

	if len(validModels) != 2 {
		t.Fatalf("expected 2 valid models, got %d", len(validModels))
	}

	if validModels[0].ModelName != "Model A" || validModels[0].Qty != 1 || validModels[0].ColIndex != 10 {
		t.Errorf("model 0 mismatch: %+v", validModels[0])
	}
	if validModels[1].ModelName != "Model D" || validModels[1].Qty != 3 || validModels[1].ColIndex != 13 {
		t.Errorf("model 1 mismatch: %+v", validModels[1])
	}
}

// TestMatrixReader_RevisionNotFound 測試當 BOM Revision 不存在時，記錄 Warning 且不變更 DB
func TestMatrixReader_RevisionNotFound(t *testing.T) {
	gdb := setupMatrixTestDB(t)

	// 建置預設 Series，但故意不建置對應的 Project / Revision
	series, err := db.GetSeriesInfo(gdb)
	if err != nil {
		t.Fatalf("failed to get series info: %v", err)
	}
	_ = series

	f := excelize.NewFile()
	wb := &ExcelizeWorkbook{f: f}
	defer f.Close()

	smdSheet := "SMD"
	f.NewSheet(smdSheet)
	f.SetCellValue(smdSheet, "B3", "PROJ_NONEXIST")
	f.SetCellValue(smdSheet, "H3", "0.1")
	f.SetCellValue(smdSheet, "J3", "EVT")
	f.SetCellValue(smdSheet, "K4", "Model A")
	f.SetCellValue(smdSheet, "K5", "1")

	importRes := types.ImportResult{FileName: "test.xlsx", Format: types.FormatMatrix}
	reader := NewMatrixReader(gdb, &importRes, nil)

	err = reader.Import(wb)
	if err != nil {
		t.Fatalf("expected nil error on revision not found warning, got: %v", err)
	}

	if len(importRes.Errors) != 1 {
		t.Fatalf("expected 1 warning error message, got: %d", len(importRes.Errors))
	}

	expectedWarnPrefix := "warning: BOM revision for project 'PROJ_NONEXIST'"
	if len(importRes.Errors[0]) < len(expectedWarnPrefix) || importRes.Errors[0][:len(expectedWarnPrefix)] != expectedWarnPrefix {
		t.Errorf("expected warning message starting with '%s', got '%s'", expectedWarnPrefix, importRes.Errors[0])
	}
}

// TestImport_Matrix_CleanAndOverwriteOldData 測試匯入 Matrix 時能清理原本舊的 Model 與 Selection
func TestImport_Matrix_CleanAndOverwriteOldData(t *testing.T) {
	gdb := setupMatrixTestDB(t)

	// 1. 建立測試基礎資料庫環境 (Series -> Project -> BomRevision -> Material -> Component)
	series, err := db.GetSeriesInfo(gdb)
	if err != nil {
		t.Fatalf("failed to get series info: %v", err)
	}

	project := db.Project{SeriesID: series.ID, Code: "PROJ_TEST", Description: "Test Project"}
	gdb.Create(&project)

	revision := db.BomRevision{
		ProjectID: project.ID,
		Phase:     "EVT",
		Version:   "0.1",
		Mode:      "NPI",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	gdb.Create(&revision)

	// 物料 1 (主料)
	mat1 := db.Material{Supplier: "SUPP_A", SupplierPN: "PN_A", HHPN: "HH001"}
	gdb.Create(&mat1)
	comp1 := db.RevisionComponent{RevisionID: revision.ID, MaterialID: mat1.ID, Role: "M", Item: "1"}
	gdb.Create(&comp1)

	// 物料 2 (替代料)
	mat2 := db.Material{Supplier: "SUPP_B", SupplierPN: "PN_B", HHPN: "HH002"}
	gdb.Create(&mat2)
	comp2 := db.RevisionComponent{RevisionID: revision.ID, MaterialID: mat2.ID, Role: "S", ParentComponentID: comp1.ID}
	gdb.Create(&comp2)

	// 2. 建立原本舊的 MatrixModel 與 MatrixSelection (測試是否會被清空覆蓋)
	oldModel := db.MatrixModel{RevisionID: revision.ID, ModelName: "OldModel", Qty: 99}
	gdb.Create(&oldModel)

	oldSel := db.MatrixSelection{
		RevisionID:         revision.ID,
		ModelID:            oldModel.ID,
		ComponentID:        comp1.ID,
		MainMaterialID:     mat1.ID,
		SelectedMaterialID: mat1.ID,
	}
	gdb.Create(&oldSel)

	// 3. 準備測試用 Excel 檔案
	f := excelize.NewFile()
	wb := &ExcelizeWorkbook{f: f}
	defer f.Close()

	smdSheet := "SMD"
	f.NewSheet(smdSheet)

	// 表頭
	f.SetCellValue(smdSheet, "B3", "Product Code: PROJ_TEST")
	f.SetCellValue(smdSheet, "B4", "Description: Matrix Import Test")
	f.SetCellValue(smdSheet, "H3", "BOM Version: 0.1")
	f.SetCellValue(smdSheet, "J3", "Phase: EVT")

	// Model 欄位 (K, L)
	f.SetCellValue(smdSheet, "K4", "Model_A")
	f.SetCellValue(smdSheet, "K5", "5")
	f.SetCellValue(smdSheet, "L4", "Model_B")
	f.SetCellValue(smdSheet, "L5", "15")

	// Row 6 (Index 5) 物料列表
	// A6=1, E6=SUPP_A, F6=PN_A, K6="V" (Model A 選擇主料), L6=""
	f.SetCellValue(smdSheet, "A6", "1")
	f.SetCellValue(smdSheet, "E6", "SUPP_A")
	f.SetCellValue(smdSheet, "F6", "PN_A")
	f.SetCellValue(smdSheet, "K6", "V")

	// Row 7 (Index 6) 替代料 (A7="", E7=SUPP_B, F7=PN_B)
	// L7="v" (Model B 選擇替代料)
	f.SetCellValue(smdSheet, "A7", "")
	f.SetCellValue(smdSheet, "E7", "SUPP_B")
	f.SetCellValue(smdSheet, "F7", "PN_B")
	f.SetCellValue(smdSheet, "L7", "v")

	importRes := types.ImportResult{FileName: "test.xlsx", Format: types.FormatMatrix}
	reader := NewMatrixReader(gdb, &importRes, nil)

	err = reader.Import(wb)
	if err != nil {
		t.Fatalf("Matrix Import failed: %v", err)
	}

	// 4. 驗證資料庫狀態
	var models []db.MatrixModel
	gdb.Where("revision_id = ?", revision.ID).Find(&models)
	if len(models) != 2 {
		t.Fatalf("expected 2 new models, got %d", len(models))
	}

	var selections []db.MatrixSelection
	gdb.Where("revision_id = ?", revision.ID).Find(&selections)
	if len(selections) != 2 {
		t.Fatalf("expected 2 new selections, got %d", len(selections))
	}

	// 驗證原本舊的 Model 是否已被清理
	for _, m := range models {
		if m.ModelName == "OldModel" {
			t.Errorf("old model was not deleted!")
		}
	}

	// 驗證新的 selection 內容
	selMap := make(map[int64]db.MatrixSelection)
	for _, s := range selections {
		selMap[s.SelectedMaterialID] = s
	}

	if sA, ok := selMap[mat1.ID]; !ok || sA.MainMaterialID != mat1.ID {
		t.Errorf("mat1 selection mismatch: %+v", sA)
	}
	if sB, ok := selMap[mat2.ID]; !ok || sB.MainMaterialID != mat1.ID {
		t.Errorf("mat2 selection mismatch: %+v", sB)
	}
}
