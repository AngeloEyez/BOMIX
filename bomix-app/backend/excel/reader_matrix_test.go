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

	err = gdb.AutoMigrate(
		&db.Series{},
		&db.Project{},
		&db.BomRevision{},
		&db.Part{},
		&db.PartLocation{},
		&db.SecondSource{},
		&db.MatrixModel{},
		&db.MatrixSelection{},
	)
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

	if projectCode != "PROJ_TEST" {
		t.Errorf("expected projectCode 'PROJ_TEST', got '%s'", projectCode)
	}
	if phase != "EVT" {
		t.Errorf("expected phase 'EVT', got '%s'", phase)
	}
	if version != "0.1" {
		t.Errorf("expected version '0.1', got '%s'", version)
	}
	if description != "Test Matrix Description" {
		t.Errorf("expected description 'Test Matrix Description', got '%s'", description)
	}
	if schematicVersion != "S1.0" {
		t.Errorf("expected schematicVersion 'S1.0', got '%s'", schematicVersion)
	}
	if pcbVersion != "P1.0" {
		t.Errorf("expected pcbVersion 'P1.0', got '%s'", pcbVersion)
	}
	if pcaPn != "PCA12345" {
		t.Errorf("expected pcaPn 'PCA12345', got '%s'", pcaPn)
	}
	if date != "2026-08-06" {
		t.Errorf("expected date '2026-08-06', got '%s'", date)
	}
}

// TestParseValidModels_Matrix 測試 Matrix Model 數量與 Qty 篩選
func TestParseValidModels_Matrix(t *testing.T) {
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

// TestMatrixReader_ImportSuccess 測試完整匯入成功流程 (包含清空舊資料、寫入 Model Qty 與 Selection)
func TestMatrixReader_ImportSuccess(t *testing.T) {
	gdb := setupMatrixTestDB(t)

	// 1. 建立測試基礎資料庫環境 (Series -> Project -> BomRevision -> Part)
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

	// 主料 1
	part1 := db.Part{
		RevisionID: revision.ID,
		Item:       "1",
		HHPN:       "HH001",
		Supplier:   "SUPP_A",
		SupplierPN: "PN_A",
	}
	gdb.Create(&part1)

	// 2. 建立原本舊的 MatrixModel 與 MatrixSelection (測試是否會被清空覆蓋)
	oldModel := db.MatrixModel{RevisionID: revision.ID, ModelName: "OldModel", Qty: 99}
	gdb.Create(&oldModel)

	oldSel := db.MatrixSelection{
		RevisionID:         revision.ID,
		ModelID:            oldModel.ID,
		PartID:             part1.ID,
		Group:              "SUPP_A|PN_A",
		Material:           "SUPP_A|PN_A",
		SelectedSupplier:   "SUPP_A",
		SelectedSupplierPn: "PN_A",
	}
	gdb.Create(&oldSel)

	// 3. 準備測試用 Excel 檔案
	f := excelize.NewFile()
	wb := &ExcelizeWorkbook{f: f}
	defer f.Close()

	smdSheet := "SMD"
	f.NewSheet(smdSheet)

	// 表頭
	f.SetCellValue(smdSheet, "B3", "PROJ_TEST")
	f.SetCellValue(smdSheet, "H3", "0.1")
	f.SetCellValue(smdSheet, "J3", "EVT")

	// Models: K5=1 (Model A), L5=2 (Model B)
	f.SetCellValue(smdSheet, "K4", "Model A")
	f.SetCellValue(smdSheet, "K5", "1")
	f.SetCellValue(smdSheet, "L4", "Model B")
	f.SetCellValue(smdSheet, "L5", "2")

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
	selMap := make(map[string]db.MatrixSelection)
	for _, s := range selections {
		selMap[s.SelectedSupplierPn] = s
	}

	if sA, ok := selMap["PN_A"]; !ok || sA.SelectedSupplier != "SUPP_A" {
		t.Errorf("PN_A selection mismatch: %+v", sA)
	}
	if sB, ok := selMap["PN_B"]; !ok || sB.SelectedSupplier != "SUPP_B" {
		t.Errorf("PN_B selection mismatch: %+v", sB)
	}
}
