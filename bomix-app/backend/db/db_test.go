package db

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// setupTestDB creates a temporary SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to initialize test database: %v", err)
	}

	if err := AutoMigrate(database); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	t.Cleanup(func() {
		_ = Close(database)
	})

	return database
}

// TestOpen tests the database initialization
func TestOpen(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_init.db")

	database, err := Open(dbPath)
	assert.NoError(t, err)
	assert.NotNil(t, database)

	// Verify database can be closed
	if err := Close(database); err != nil {
		t.Errorf("failed to close database: %v", err)
	}
}

// TestAutoMigrate tests the AutoMigrate function creates all tables
func TestAutoMigrate(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_migrate.db")

	database, err := Open(dbPath)
	assert.NoError(t, err)
	t.Cleanup(func() {
		_ = Close(database)
	})

	err = AutoMigrate(database)
	assert.NoError(t, err)

	var count int64
	if err := database.Model(&Series{}).Count(&count).Error; err != nil {
		t.Errorf("failed to query Series table: %v", err)
	}
	if err := database.Model(&Project{}).Count(&count).Error; err != nil {
		t.Errorf("failed to query Project table: %v", err)
	}
	if err := database.Model(&BomRevision{}).Count(&count).Error; err != nil {
		t.Errorf("failed to query BomRevision table: %v", err)
	}
	if err := database.Model(&Material{}).Count(&count).Error; err != nil {
		t.Errorf("failed to query Material table: %v", err)
	}
	if err := database.Model(&RevisionComponent{}).Count(&count).Error; err != nil {
		t.Errorf("failed to query RevisionComponent table: %v", err)
	}
	if err := database.Model(&PartLocation{}).Count(&count).Error; err != nil {
		t.Errorf("failed to query PartLocation table: %v", err)
	}
	if err := database.Model(&MatrixModel{}).Count(&count).Error; err != nil {
		t.Errorf("failed to query MatrixModel table: %v", err)
	}
	if err := database.Model(&MatrixSelection{}).Count(&count).Error; err != nil {
		t.Errorf("failed to query MatrixSelection table: %v", err)
	}
}

// TestCreateAndGetSeries tests creating and retrieving a series
func TestCreateAndGetSeries(t *testing.T) {
	database := setupTestDB(t)

	series, err := CreateSeries(database, "Test Series", "Test Description")
	assert.NoError(t, err)
	assert.NotNil(t, series)
	assert.Equal(t, "Test Series", series.Name)
	assert.Equal(t, "Test Description", series.Description)

	retrieved, err := GetSeriesInfo(database)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, series.ID, retrieved.ID)
	assert.Equal(t, "Test Series", retrieved.Name)
}

// TestProjectOperations tests project CRUD operations
func TestProjectOperations(t *testing.T) {
	database := setupTestDB(t)

	series, err := CreateSeries(database, "Test Series", "Test Description")
	assert.NoError(t, err)

	// Create projects
	project1, err := GetOrCreateProject(database, series.ID, "PROJ001", "Description 1")
	assert.NoError(t, err)
	assert.NotNil(t, project1)
	assert.Equal(t, "PROJ001", project1.Code)

	project2, err := GetOrCreateProject(database, series.ID, "PROJ002", "Description 2")
	assert.NoError(t, err)
	assert.NotNil(t, project2)
	assert.Equal(t, "PROJ002", project2.Code)

	// GetOrCreate with existing code should return existing
	project1Again, err := GetOrCreateProject(database, series.ID, "PROJ001", "Different Desc")
	assert.NoError(t, err)
	assert.Equal(t, project1.ID, project1Again.ID)

	// Get all projects
	projects, err := GetProjects(database, series.ID)
	assert.NoError(t, err)
	assert.Len(t, projects, 2)

	// Get project by ID
	retrieved, err := GetProject(database, project1.ID)
	assert.NoError(t, err)
	assert.Equal(t, project1.Code, retrieved.Code)
}

// TestRevisionOperations tests revision CRUD operations
func TestRevisionOperations(t *testing.T) {
	database := setupTestDB(t)

	series, err := CreateSeries(database, "Test Series", "Test Description")
	assert.NoError(t, err)
	project, err := GetOrCreateProject(database, series.ID, "PROJ001", "Test Project")
	assert.NoError(t, err)

	// Create revision
	revision, err := CreateRevision(database, project.ID, "DB", "0.1", "Initial Revision")
	assert.NoError(t, err)
	assert.NotNil(t, revision)
	assert.Equal(t, "DB", revision.Phase)
	assert.Equal(t, "0.1", revision.Version)

	// Get revision by ID
	retrieved, err := GetRevision(database, revision.ID)
	assert.NoError(t, err)
	assert.Equal(t, revision.Phase, retrieved.Phase)

	// Find revision by phase and version
	retrievedByPV, err := FindRevision(database, project.ID, "DB", "0.1")
	assert.NoError(t, err)
	assert.Equal(t, revision.ID, retrievedByPV.ID)

	// Get all revisions for project
	revisions, err := GetRevisions(database, project.ID)
	assert.NoError(t, err)
	assert.Len(t, revisions, 1)

	// Update revision
	retrieved.Description = "Updated Revision Description"
	retrieved.SchematicVersion = "SCH-v1.0"
	retrieved.PCBVersion = "PCB-v1.0"
	err = UpdateRevision(database, retrieved)
	assert.NoError(t, err)

	updated, err := GetRevision(database, revision.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Revision Description", updated.Description)
	assert.Equal(t, "SCH-v1.0", updated.SchematicVersion)
}

// TestMaterialOperations tests Material CRUD and Upsert
func TestMaterialOperations(t *testing.T) {
	database := setupTestDB(t)

	mats := []Material{
		{
			Supplier:    "Samsung",
			SupplierPN:  "CL05B104KO5NNNC",
			HHPN:        "HH-001",
			Description: "CAP 100nF",
			Remark:      "Initial Remark",
		},
		{
			Supplier:    "Murata",
			SupplierPN:  "GRM188R61A106KE15D",
			HHPN:        "HH-002",
			Description: "CAP 10uF",
			Remark:      "",
		},
	}

	// 1. Upsert (Insert)
	ins, upd, err := UpsertMaterials(database, mats, nil)
	assert.NoError(t, err)
	assert.Equal(t, 2, ins)
	assert.Equal(t, 0, upd)

	m1, err := GetMaterialBySupplierPN(database, "Samsung", "CL05B104KO5NNNC")
	assert.NoError(t, err)
	assert.NotNil(t, m1)
	assert.Greater(t, m1.ID, int64(0))

	// 2. Upsert (Update with blank protection & remark clearance)
	updateMats := []Material{
		{
			Supplier:    "Samsung",
			SupplierPN:  "CL05B104KO5NNNC",
			HHPN:        "", // 空白保護：應保留舊值 "HH-001"
			Description: "CAP 100nF Updated",
			Remark:      "", // remark 排除保護：應清空為 ""
		},
	}

	ins2, upd2, err := UpsertMaterials(database, updateMats, nil)
	assert.NoError(t, err)
	assert.Equal(t, 0, ins2)
	assert.Equal(t, 1, upd2)

	retrieved, err := GetMaterial(database, m1.ID)
	assert.NoError(t, err)
	assert.Equal(t, "HH-001", retrieved.HHPN) // 保留
	assert.Equal(t, "CAP 100nF Updated", retrieved.Description)
	assert.Equal(t, "", retrieved.Remark) // 被清空
}

// TestComponentAndLocationOperations tests RevisionComponent and PartLocation operations
func TestComponentAndLocationOperations(t *testing.T) {
	database := setupTestDB(t)

	_, err := CreateSeries(database, "Test Series", "Test Description")
	assert.NoError(t, err)
	project, err := GetOrCreateProject(database, 1, "PROJ001", "Test Project")
	assert.NoError(t, err)
	revision, err := CreateRevision(database, project.ID, "DB", "0.1", "Initial")
	assert.NoError(t, err)

	// 建立物料
	mats := []Material{
		{Supplier: "Samsung", SupplierPN: "CL05B104KO5NNNC"},
		{Supplier: "Murata", SupplierPN: "GRM188R61A106KE15D"},
	}
	_, _, err = UpsertMaterials(database, mats, nil)
	assert.NoError(t, err)

	m1, _ := GetMaterialBySupplierPN(database, "Samsung", "CL05B104KO5NNNC")
	m2, _ := GetMaterialBySupplierPN(database, "Murata", "GRM188R61A106KE15D")

	// 建立主料
	mainComp := RevisionComponent{
		RevisionID: revision.ID,
		MaterialID: m1.ID,
		Role:       "M",
		Item:       "1",
	}
	err = CreateComponentsInBatch(database, []RevisionComponent{mainComp})
	assert.NoError(t, err)

	comps, err := GetComponentsByRevision(database, revision.ID)
	assert.NoError(t, err)
	assert.Len(t, comps, 1)
	mainCompID := comps[0].ID

	// 建立替代料
	secComp := RevisionComponent{
		RevisionID:        revision.ID,
		MaterialID:        m2.ID,
		Role:              "S",
		ParentComponentID: mainCompID,
	}
	err = CreateComponentsInBatch(database, []RevisionComponent{secComp})
	assert.NoError(t, err)

	allComps, err := GetComponentsByRevision(database, revision.ID)
	assert.NoError(t, err)
	assert.Len(t, allComps, 2)

	// 建立 PartLocation
	locs := []PartLocation{
		{ComponentID: mainCompID, Location: "C1", Type: "SMD", BomStatus: "I", CCL: true},
		{ComponentID: mainCompID, Location: "C2", Type: "SMD", BomStatus: "I", CCL: true},
	}
	err = CreatePartLocationsInBatch(database, locs)
	assert.NoError(t, err)

	retrievedLocs, err := GetPartLocationsByComponentIDs(database, []int64{mainCompID})
	assert.NoError(t, err)
	assert.Len(t, retrievedLocs, 2)

	// 取得含 Locations 的 Components
	compsWithLoc, err := GetComponentsByRevisionWithLocations(database, revision.ID)
	assert.NoError(t, err)
	assert.Len(t, compsWithLoc, 2)
}

// TestMatrixModelAndSelectionOperations tests matrix model & selection CRUD
func TestMatrixModelAndSelectionOperations(t *testing.T) {
	database := setupTestDB(t)

	_, err := CreateSeries(database, "Test Series", "Test Description")
	assert.NoError(t, err)
	project, err := GetOrCreateProject(database, 1, "PROJ001", "Test Project")
	assert.NoError(t, err)
	revision, err := CreateRevision(database, project.ID, "DB", "0.1", "Initial")
	assert.NoError(t, err)

	// 建立 MatrixModel
	modelA := &MatrixModel{
		RevisionID: revision.ID,
		SortOrder:  0,
		ModelName:  "ModelA",
		Qty:        1,
	}
	err = CreateMatrixModel(database, modelA)
	assert.NoError(t, err)

	// 建立物料與 Component
	mats := []Material{
		{Supplier: "Samsung", SupplierPN: "CL05B104KO5NNNC"},
	}
	_, _, err = UpsertMaterials(database, mats, nil)
	assert.NoError(t, err)
	m1, _ := GetMaterialBySupplierPN(database, "Samsung", "CL05B104KO5NNNC")

	comp := RevisionComponent{
		RevisionID: revision.ID,
		MaterialID: m1.ID,
		Role:       "M",
	}
	err = CreateComponentsInBatch(database, []RevisionComponent{comp})
	assert.NoError(t, err)
	comps, _ := GetComponentsByRevision(database, revision.ID)
	compID := comps[0].ID

	// 建立 Selection
	selections := []MatrixSelection{
		{
			RevisionID:         revision.ID,
			ModelID:            modelA.ID,
			ComponentID:        compID,
			MainMaterialID:     m1.ID,
			SelectedMaterialID: m1.ID,
			IsAutoSelected:     false,
		},
	}
	err = CreateMatrixSelections(database, selections)
	assert.NoError(t, err)

	retrieved, err := GetMatrixSelections(database, revision.ID, modelA.ID)
	assert.NoError(t, err)
	assert.Len(t, retrieved, 1)
	assert.Equal(t, m1.ID, retrieved[0].SelectedMaterialID)

	// 刪除無效 Selection
	err = DeleteInvalidSelections(database, revision.ID, []int64{compID}, nil)
	assert.NoError(t, err)

	afterDel, err := GetMatrixSelections(database, revision.ID, modelA.ID)
	assert.NoError(t, err)
	assert.Len(t, afterDel, 0)
}

// TestUpsertMatrixSelection 測試 Matrix Selection 的單筆覆蓋更新、互斥與取消選取
func TestUpsertMatrixSelection(t *testing.T) {
	database := setupTestDB(t)

	_, err := CreateSeries(database, "Test Series", "Test Description")
	assert.NoError(t, err)
	project, err := GetOrCreateProject(database, 1, "PROJ001", "Test Project")
	assert.NoError(t, err)
	revision, err := CreateRevision(database, project.ID, "DB", "0.1", "Initial")
	assert.NoError(t, err)

	// 建立物料：主料 m1 與替代料 m2
	mats := []Material{
		{Supplier: "Samsung", SupplierPN: "CL05B104KO5NNNC"},
		{Supplier: "Murata", SupplierPN: "GRM155R71C104KA88D"},
	}
	_, _, err = UpsertMaterials(database, mats, nil)
	assert.NoError(t, err)
	m1, _ := GetMaterialBySupplierPN(database, "Samsung", "CL05B104KO5NNNC")
	m2, _ := GetMaterialBySupplierPN(database, "Murata", "GRM155R71C104KA88D")

	compM := RevisionComponent{
		RevisionID: revision.ID,
		MaterialID: m1.ID,
		Role:       "M",
	}
	err = CreateComponentsInBatch(database, []RevisionComponent{compM})
	assert.NoError(t, err)

	// 1. 測試自動建立預設 Model 並選取主料 (modelID = 0)
	err = UpsertMatrixSelection(database, revision.ID, 0, m1.ID, m1.ID)
	assert.NoError(t, err)

	models, err := GetMatrixModels(database, revision.ID)
	assert.NoError(t, err)
	assert.Len(t, models, 1)
	defaultModelID := models[0].ID

	sels, err := GetMatrixSelections(database, revision.ID, defaultModelID)
	assert.NoError(t, err)
	assert.Len(t, sels, 1)
	assert.Equal(t, m1.ID, sels[0].SelectedMaterialID)

	// 2. 測試改選為替代料 m2 (互斥覆蓋)
	err = UpsertMatrixSelection(database, revision.ID, defaultModelID, m1.ID, m2.ID)
	assert.NoError(t, err)

	sels2, err := GetMatrixSelections(database, revision.ID, defaultModelID)
	assert.NoError(t, err)
	assert.Len(t, sels2, 1)
	assert.Equal(t, m2.ID, sels2[0].SelectedMaterialID)

	// 3. 測試取消選取 (selectedMaterialID = 0)
	err = UpsertMatrixSelection(database, revision.ID, defaultModelID, m1.ID, 0)
	assert.NoError(t, err)

	sels3, err := GetMatrixSelections(database, revision.ID, defaultModelID)
	assert.NoError(t, err)
	assert.Len(t, sels3, 0)
}

// TestSaveRevisionMatrixModels 測試批次更新 Revision MatrixModels (包含更新 Qty、新增 Model 與縮減刪除 Model)
func TestSaveRevisionMatrixModels(t *testing.T) {
	database := setupTestDB(t)

	_, err := CreateSeries(database, "Test Series", "Test Description")
	assert.NoError(t, err)
	project, err := GetOrCreateProject(database, 1, "PROJ001", "Test Project")
	assert.NoError(t, err)
	revision, err := CreateRevision(database, project.ID, "DB", "0.1", "Initial")
	assert.NoError(t, err)

	// 1. 建立初始物料與 Component
	mats := []Material{
		{Supplier: "Samsung", SupplierPN: "CL05B104KO5NNNC"},
	}
	_, _, err = UpsertMaterials(database, mats, nil)
	assert.NoError(t, err)
	m1, _ := GetMaterialBySupplierPN(database, "Samsung", "CL05B104KO5NNNC")

	comp := RevisionComponent{
		RevisionID: revision.ID,
		MaterialID: m1.ID,
		Role:       "M",
	}
	err = CreateComponentsInBatch(database, []RevisionComponent{comp})
	assert.NoError(t, err)

	// 2. 初始批次建立 2 個 Model: A(Qty=5), B(Qty=10)
	initialInputs := []MatrixModelInput{
		{SortOrder: 0, ModelName: "Model A", Qty: 5},
		{SortOrder: 1, ModelName: "Model B", Qty: 10},
	}
	err = SaveRevisionMatrixModels(database, revision.ID, initialInputs)
	assert.NoError(t, err)

	models, err := GetMatrixModels(database, revision.ID)
	assert.NoError(t, err)
	assert.Len(t, models, 2)
	assert.Equal(t, "Model A", models[0].ModelName)
	assert.Equal(t, 5, models[0].Qty)
	assert.Equal(t, "Model B", models[1].ModelName)
	assert.Equal(t, 10, models[1].Qty)
	origModelAID := models[0].ID
	origModelBID := models[1].ID

	// 為 Model A 和 Model B 建立勾選記錄
	err = UpsertMatrixSelection(database, revision.ID, origModelAID, m1.ID, m1.ID)
	assert.NoError(t, err)
	err = UpsertMatrixSelection(database, revision.ID, origModelBID, m1.ID, m1.ID)
	assert.NoError(t, err)

	selA, _ := GetMatrixSelections(database, revision.ID, origModelAID)
	assert.Len(t, selA, 1)
	selB, _ := GetMatrixSelections(database, revision.ID, origModelBID)
	assert.Len(t, selB, 1)

	// 3. 修改 Model A Qty=25，並新增 Model C (SortOrder=2, Qty=30)
	updateInputs := []MatrixModelInput{
		{SortOrder: 0, ModelName: "Model A", Qty: 25},
		{SortOrder: 1, ModelName: "Model B", Qty: 10},
		{SortOrder: 2, ModelName: "Model C", Qty: 30},
	}
	err = SaveRevisionMatrixModels(database, revision.ID, updateInputs)
	assert.NoError(t, err)

	models2, err := GetMatrixModels(database, revision.ID)
	assert.NoError(t, err)
	assert.Len(t, models2, 3)
	assert.Equal(t, origModelAID, models2[0].ID, "Model A 的 ID 應保持不變以維持勾選關聯")
	assert.Equal(t, 25, models2[0].Qty, "Model A 的 Qty 應更新為 25")
	assert.Equal(t, "Model C", models2[2].ModelName)
	assert.Equal(t, 30, models2[2].Qty)

	// 驗證 Model A 的 Selection 依舊完好
	selA2, _ := GetMatrixSelections(database, revision.ID, origModelAID)
	assert.Len(t, selA2, 1, "Model A 的勾選紀錄不應丟失")

	// 4. 縮減/隱藏 Model：Model B 有歷史勾選紀錄，縮減清單時應自動設為隱藏並保留勾選；
	//    Model C 無勾選紀錄且未在清單中，應被清理刪除。
	reduceInputs := []MatrixModelInput{
		{SortOrder: 0, ModelName: "Model A", Qty: 25, IsHidden: false},
	}
	err = SaveRevisionMatrixModels(database, revision.ID, reduceInputs)
	assert.NoError(t, err)

	models3, err := GetMatrixModels(database, revision.ID)
	assert.NoError(t, err)
	assert.Len(t, models3, 2, "有歷史勾選的 Model B 應保留並設為隱藏，無勾選的 Model C 應被移除")
	assert.Equal(t, origModelAID, models3[0].ID)
	assert.False(t, models3[0].IsHidden)
	assert.Equal(t, origModelBID, models3[1].ID)
	assert.True(t, models3[1].IsHidden, "Model B 應被標記為隱藏 (IsHidden=true)")

	// 驗證被隱藏的 Model B 其 Selection 仍完整保留以便日後恢復
	selBPreserved, _ := GetMatrixSelections(database, revision.ID, origModelBID)
	assert.Len(t, selBPreserved, 1, "被隱藏的 Model B 其勾選紀錄應保留於資料庫中")

	// 驗證有效 (Active) Model 統計僅計入未隱藏的 Model A
	activeCounts, err := CountActiveMatrixModelsByRevisions(database, []int64{revision.ID})
	assert.NoError(t, err)
	assert.Equal(t, 1, activeCounts[revision.ID], "有效 Active Model 數應僅為 1 (Model A)")
}

// TestSaveRevisionMatrixModels_SparseModelCAndHideRestore 驗證支援從 Model C (SortOrder=2) 開始、隱藏保留勾選與恢復啟用
func TestSaveRevisionMatrixModels_SparseModelCAndHideRestore(t *testing.T) {
	database := setupTestDB(t)

	_, err := CreateSeries(database, "Series Sparse", "Test")
	assert.NoError(t, err)
	project, err := GetOrCreateProject(database, 1, "PROJ-SPARSE", "Sparse Project")
	assert.NoError(t, err)
	revision, err := CreateRevision(database, project.ID, "SI", "0.1", "Sparse Rev")
	assert.NoError(t, err)

	_, _, err = UpsertMaterials(database, []Material{{Supplier: "YAGEO", SupplierPN: "R10K"}}, nil)
	assert.NoError(t, err)
	m1, _ := GetMaterialBySupplierPN(database, "YAGEO", "R10K")
	assert.NoError(t, CreateComponentsInBatch(database, []RevisionComponent{
		{RevisionID: revision.ID, MaterialID: m1.ID, Role: "M"},
	}))

	// 1. 僅啟用 Model C (SortOrder=2, Qty=15)，Model A 與 B 皆未建立/隱藏
	err = SaveRevisionMatrixModels(database, revision.ID, []MatrixModelInput{
		{SortOrder: 0, ModelName: "Model A", Qty: 0, IsHidden: true},
		{SortOrder: 1, ModelName: "Model B", Qty: 0, IsHidden: true},
		{SortOrder: 2, ModelName: "Model C", Qty: 15, IsHidden: false},
	})
	assert.NoError(t, err)

	models, err := GetMatrixModels(database, revision.ID)
	assert.NoError(t, err)
	assert.Len(t, models, 1, "未建立且隱藏的 Model A/B 不應建立冗餘資料列")
	assert.Equal(t, 2, models[0].SortOrder)
	assert.Equal(t, 15, models[0].Qty)
	assert.False(t, models[0].IsHidden)

	// 2. 在 Model C 勾選物料，再將 Model C 設為隱藏
	assert.NoError(t, UpsertMatrixModelSelection(database, revision.ID, 2, m1.ID, m1.ID))
	err = SaveRevisionMatrixModels(database, revision.ID, []MatrixModelInput{
		{SortOrder: 2, ModelName: "Model C", Qty: 15, IsHidden: true},
	})
	assert.NoError(t, err)

	activeAfterHide, err := CountActiveMatrixModelsByRevisions(database, []int64{revision.ID})
	assert.NoError(t, err)
	assert.Equal(t, 0, activeAfterHide[revision.ID], "全部隱藏後有效 Model 數應為 0")

	selsAfterHide, err := GetMatrixSelections(database, revision.ID, models[0].ID)
	assert.NoError(t, err)
	assert.Len(t, selsAfterHide, 1, "隱藏 Model C 後其歷史勾選紀錄應仍保留")

	// 3. 重新啟用 Model C，確認 Qty 與勾選皆無損恢復
	err = SaveRevisionMatrixModels(database, revision.ID, []MatrixModelInput{
		{SortOrder: 2, ModelName: "Model C", Qty: 15, IsHidden: false},
	})
	assert.NoError(t, err)

	activeAfterRestore, err := CountActiveMatrixModelsByRevisions(database, []int64{revision.ID})
	assert.NoError(t, err)
	assert.Equal(t, 1, activeAfterRestore[revision.ID])
}

// TestImportMatrixSelections_Comprehensive 測試 ImportMatrixSelections 複製邏輯：包含防誤刪、階層替代料驗證與覆蓋複製
func TestImportMatrixSelections_Comprehensive(t *testing.T) {
	database := setupTestDB(t)

	_, err := CreateSeries(database, "Series Matrix Copy", "Test")
	assert.NoError(t, err)
	project, err := GetOrCreateProject(database, 1, "PROJ-COPY", "Copy Project")
	assert.NoError(t, err)

	// 建立來源版本與目標版本
	revSource, err := CreateRevision(database, project.ID, "PV", "0.1", "Source Rev")
	assert.NoError(t, err)
	revTarget, err := CreateRevision(database, project.ID, "PV", "0.2", "Target Rev")
	assert.NoError(t, err)

	// 建立物料
	mats := []Material{
		{Supplier: "VendorA", SupplierPN: "MAIN-101"},
		{Supplier: "VendorB", SupplierPN: "SEC-101"},
		{Supplier: "VendorC", SupplierPN: "OTHER-999"},
	}
	_, _, err = UpsertMaterials(database, mats, nil)
	assert.NoError(t, err)
	matMain, _ := GetMaterialBySupplierPN(database, "VendorA", "MAIN-101")
	matSec, _ := GetMaterialBySupplierPN(database, "VendorB", "SEC-101")
	matOther, _ := GetMaterialBySupplierPN(database, "VendorC", "OTHER-999")

	// 1. 測試：來源無任何 MatrixModel 時，不應覆蓋清空目標版本的既有資料
	targetModelPre := MatrixModel{RevisionID: revTarget.ID, SortOrder: 0, ModelName: "TargetPre", Qty: 1}
	err = database.Create(&targetModelPre).Error
	assert.NoError(t, err)

	statsEmpty, err := ImportMatrixSelections(database, revSource.ID, revTarget.ID, nil)
	assert.NoError(t, err)
	assert.Equal(t, 0, statsEmpty.SourceModelCount)

	// 確認目標版本的既有 Model 依然存在
	var targetModelsAfterEmpty []MatrixModel
	database.Where("revision_id = ?", revTarget.ID).Find(&targetModelsAfterEmpty)
	assert.Len(t, targetModelsAfterEmpty, 1, "來源無 Model 時，目標版本的既有 Model 不應被刪除")

	// 2. 在來源建立 Model 與 Component
	compSourceMain := RevisionComponent{RevisionID: revSource.ID, MaterialID: matMain.ID, Role: "M"}
	err = database.Create(&compSourceMain).Error
	assert.NoError(t, err)
	compSourceSec := RevisionComponent{RevisionID: revSource.ID, MaterialID: matSec.ID, Role: "S", ParentComponentID: compSourceMain.ID}
	err = database.Create(&compSourceSec).Error
	assert.NoError(t, err)

	modelSourceA := MatrixModel{RevisionID: revSource.ID, SortOrder: 0, ModelName: "Model Alpha", Qty: 2}
	modelSourceB := MatrixModel{RevisionID: revSource.ID, SortOrder: 1, ModelName: "Model Beta", Qty: 3}
	err = database.Create(&modelSourceA).Error
	assert.NoError(t, err)
	err = database.Create(&modelSourceB).Error
	assert.NoError(t, err)

	// 來源 Model A 選用主料自身，Model B 選用替代料 matSec
	err = database.Create(&MatrixSelection{
		RevisionID:         revSource.ID,
		ModelID:            modelSourceA.ID,
		ComponentID:        compSourceMain.ID,
		MainMaterialID:     matMain.ID,
		SelectedMaterialID: matMain.ID,
	}).Error
	assert.NoError(t, err)

	err = database.Create(&MatrixSelection{
		RevisionID:         revSource.ID,
		ModelID:            modelSourceB.ID,
		ComponentID:        compSourceMain.ID,
		MainMaterialID:     matMain.ID,
		SelectedMaterialID: matSec.ID,
	}).Error
	assert.NoError(t, err)

	// 3. 在目標版本中建立零件：
	// 目標版本具有 matMain 主料，但故意不將 matSec 作為其替代料；
	// 同時建立第二個主料零件，將 matOther 作為第二個零件的替代料，驗證不同零件的替代料不會被跨零件錯誤採納。
	compTargetMain := RevisionComponent{RevisionID: revTarget.ID, MaterialID: matMain.ID, Role: "M"}
	err = database.Create(&compTargetMain).Error
	assert.NoError(t, err)

	compTargetOtherMain := RevisionComponent{RevisionID: revTarget.ID, MaterialID: 99999, Role: "M"}
	err = database.Create(&compTargetOtherMain).Error
	assert.NoError(t, err)

	compTargetOtherSec := RevisionComponent{RevisionID: revTarget.ID, MaterialID: matOther.ID, Role: "S", ParentComponentID: compTargetOtherMain.ID}
	err = database.Create(&compTargetOtherSec).Error
	assert.NoError(t, err)

	// 執行複製
	stats, err := ImportMatrixSelections(database, revSource.ID, revTarget.ID, nil)
	assert.NoError(t, err)
	assert.Equal(t, 2, stats.SourceModelCount)
	// Model A 成功複製（主料選中），Model B 因目標無該主料直屬替代料 matSec 而被忽略
	assert.Equal(t, 1, stats.CopiedSelectionsCount)
	assert.Equal(t, 1, stats.IgnoredSecondParts)
	assert.Equal(t, 0, stats.IgnoredMainParts)

	// 驗證目標版本的 Model 結構
	targetModels, err := GetMatrixModels(database, revTarget.ID)
	assert.NoError(t, err)
	assert.Len(t, targetModels, 2)
	assert.Equal(t, "Model Alpha", targetModels[0].ModelName)
	assert.Equal(t, "Model Beta", targetModels[1].ModelName)

	// 驗證目標版本的 Selection
	targetSels, err := GetMatrixSelectionsByRevision(database, revTarget.ID)
	assert.NoError(t, err)
	assert.Len(t, targetSels, 1)
	assert.Equal(t, matMain.ID, targetSels[0].SelectedMaterialID)
	assert.True(t, targetSels[0].IsAutoSelected)
	assert.Equal(t, compTargetMain.ID, targetSels[0].ComponentID)
}

// TestGetRevisionMatrixModelSummaries 測試查詢單一 Revision 的 MatrixModel 摘要與勾選筆數統計
func TestGetRevisionMatrixModelSummaries(t *testing.T) {
	database := setupTestDB(t)

	_, err := CreateSeries(database, "Series Model Summary", "Test")
	assert.NoError(t, err)
	project, err := GetOrCreateProject(database, 1, "PROJ-SUM", "Summary Project")
	assert.NoError(t, err)
	revEmpty, err := CreateRevision(database, project.ID, "EVT", "0.1", "Empty Rev")
	assert.NoError(t, err)
	revData, err := CreateRevision(database, project.ID, "EVT", "0.2", "Data Rev")
	assert.NoError(t, err)

	// 1. 參數防呆驗證
	_, err = GetRevisionMatrixModelSummaries(nil, revData.ID)
	assert.Error(t, err)
	_, err = GetRevisionMatrixModelSummaries(database, 0)
	assert.Error(t, err)

	// 2. 空版本回傳空切片
	emptySummaries, err := GetRevisionMatrixModelSummaries(database, revEmpty.ID)
	assert.NoError(t, err)
	assert.Empty(t, emptySummaries)

	// 3. 建立稀疏 SortOrder 與隱藏狀態的 Model
	mats := []Material{
		{Supplier: "VendorA", SupplierPN: "SUM-M1"},
		{Supplier: "VendorB", SupplierPN: "SUM-M2"},
	}
	_, _, err = UpsertMaterials(database, mats, nil)
	assert.NoError(t, err)
	m1, _ := GetMaterialBySupplierPN(database, "VendorA", "SUM-M1")
	m2, _ := GetMaterialBySupplierPN(database, "VendorB", "SUM-M2")

	comp1 := RevisionComponent{RevisionID: revData.ID, MaterialID: m1.ID, Role: "M"}
	comp2 := RevisionComponent{RevisionID: revData.ID, MaterialID: m2.ID, Role: "M"}
	assert.NoError(t, database.Create(&comp1).Error)
	assert.NoError(t, database.Create(&comp2).Error)

	// 特意以倒序建立 SortOrder，驗證查詢結果依 SortOrder ASC 排序與 ModelAlias 正確推算
	modelC := MatrixModel{RevisionID: revData.ID, SortOrder: 2, ModelName: "Custom C", Qty: 10, IsHidden: false}
	modelA := MatrixModel{RevisionID: revData.ID, SortOrder: 0, ModelName: "Custom A", Qty: 25, IsHidden: false}
	modelB := MatrixModel{RevisionID: revData.ID, SortOrder: 1, ModelName: "Hidden B", Qty: 5, IsHidden: true}
	assert.NoError(t, database.Create(&modelC).Error)
	assert.NoError(t, database.Create(&modelA).Error)
	assert.NoError(t, database.Create(&modelB).Error)

	// Model A 有 2 筆有效勾選；Model B (隱藏) 有 1 筆；Model C 有 0 筆有效勾選（含 1 筆 selected_material_id=0 邊界）
	assert.NoError(t, database.Create(&MatrixSelection{
		RevisionID: revData.ID, ModelID: modelA.ID, ComponentID: comp1.ID, MainMaterialID: m1.ID, SelectedMaterialID: m1.ID,
	}).Error)
	assert.NoError(t, database.Create(&MatrixSelection{
		RevisionID: revData.ID, ModelID: modelA.ID, ComponentID: comp2.ID, MainMaterialID: m2.ID, SelectedMaterialID: m2.ID,
	}).Error)
	assert.NoError(t, database.Create(&MatrixSelection{
		RevisionID: revData.ID, ModelID: modelB.ID, ComponentID: comp1.ID, MainMaterialID: m1.ID, SelectedMaterialID: m1.ID,
	}).Error)
	assert.NoError(t, database.Create(&MatrixSelection{
		RevisionID: revData.ID, ModelID: modelC.ID, ComponentID: comp1.ID, MainMaterialID: m1.ID, SelectedMaterialID: 0,
	}).Error)

	summaries, err := GetRevisionMatrixModelSummaries(database, revData.ID)
	assert.NoError(t, err)
	assert.Len(t, summaries, 3)

	assert.Equal(t, modelA.ID, summaries[0].ID)
	assert.Equal(t, 0, summaries[0].SortOrder)
	assert.Equal(t, "A", summaries[0].ModelAlias)
	assert.Equal(t, 25, summaries[0].Qty)
	assert.False(t, summaries[0].IsHidden)
	assert.Equal(t, 2, summaries[0].SelectionCount)

	assert.Equal(t, modelB.ID, summaries[1].ID)
	assert.Equal(t, 1, summaries[1].SortOrder)
	assert.Equal(t, "B", summaries[1].ModelAlias)
	assert.True(t, summaries[1].IsHidden)
	assert.Equal(t, 1, summaries[1].SelectionCount)

	assert.Equal(t, modelC.ID, summaries[2].ID)
	assert.Equal(t, 2, summaries[2].SortOrder)
	assert.Equal(t, "C", summaries[2].ModelAlias)
	assert.False(t, summaries[2].IsHidden)
	assert.Equal(t, 0, summaries[2].SelectionCount)
}

// TestCopyMatrixModelSelections_TableDriven 測試 Model-to-Model Matrix 複製之跨版本、同版本、1對多、連鎖互換與防呆邊界
func TestCopyMatrixModelSelections_TableDriven(t *testing.T) {
	// 輔助結構：每個 subtest 建立獨立的記憶體/暫存 DB 與完整測試夾具
	type copyFixture struct {
		db        *gorm.DB
		revSrc    *BomRevision
		revTgt    *BomRevision
		matM1     *Material
		matM1Sec  *Material
		matM2     *Material
		matOnlySrc *Material
		srcModA   MatrixModel
		srcModB   MatrixModel
		srcModEmpty MatrixModel
		srcModHidden MatrixModel
		tgtModA   MatrixModel
		tgtModB   MatrixModel
		tgtModC   MatrixModel
		tgtModHidden MatrixModel
	}

	setupFixture := func(t *testing.T) copyFixture {
		t.Helper()
		database := setupTestDB(t)
		_, err := CreateSeries(database, "Series M2M", "Test")
		assert.NoError(t, err)
		proj, err := GetOrCreateProject(database, 1, "PROJ-M2M", "M2M")
		assert.NoError(t, err)
		revSrc, err := CreateRevision(database, proj.ID, "PV", "0.1", "Src")
		assert.NoError(t, err)
		revTgt, err := CreateRevision(database, proj.ID, "PV", "0.2", "Tgt")
		assert.NoError(t, err)

		mats := []Material{
			{Supplier: "V", SupplierPN: "M1"},
			{Supplier: "V", SupplierPN: "M1-SEC"},
			{Supplier: "V", SupplierPN: "M2"},
			{Supplier: "V", SupplierPN: "SRC-ONLY"},
			{Supplier: "V", SupplierPN: "M2-SEC-MISSING-IN-TGT"},
		}
		_, _, err = UpsertMaterials(database, mats, nil)
		assert.NoError(t, err)
		matM1, _ := GetMaterialBySupplierPN(database, "V", "M1")
		matM1Sec, _ := GetMaterialBySupplierPN(database, "V", "M1-SEC")
		matM2, _ := GetMaterialBySupplierPN(database, "V", "M2")
		matOnlySrc, _ := GetMaterialBySupplierPN(database, "V", "SRC-ONLY")
		matM2SecOnlySrc, _ := GetMaterialBySupplierPN(database, "V", "M2-SEC-MISSING-IN-TGT")

		// 來源版本 BOM 結構：M1 (+M1-SEC), M2 (+M2-SEC-MISSING-IN-TGT), SRC-ONLY
		srcCompM1 := RevisionComponent{RevisionID: revSrc.ID, MaterialID: matM1.ID, Role: "M"}
		srcCompM2 := RevisionComponent{RevisionID: revSrc.ID, MaterialID: matM2.ID, Role: "M"}
		srcCompOnly := RevisionComponent{RevisionID: revSrc.ID, MaterialID: matOnlySrc.ID, Role: "M"}
		assert.NoError(t, database.Create(&srcCompM1).Error)
		assert.NoError(t, database.Create(&srcCompM2).Error)
		assert.NoError(t, database.Create(&srcCompOnly).Error)
		srcCompM1Sec := RevisionComponent{RevisionID: revSrc.ID, MaterialID: matM1Sec.ID, Role: "S", ParentComponentID: srcCompM1.ID}
		srcCompM2Sec := RevisionComponent{RevisionID: revSrc.ID, MaterialID: matM2SecOnlySrc.ID, Role: "S", ParentComponentID: srcCompM2.ID}
		assert.NoError(t, database.Create(&srcCompM1Sec).Error)
		assert.NoError(t, database.Create(&srcCompM2Sec).Error)

		// 目標版本 BOM 結構：M1 (+M1-SEC), M2 (無替代料)，無 SRC-ONLY
		tgtCompM1 := RevisionComponent{RevisionID: revTgt.ID, MaterialID: matM1.ID, Role: "M"}
		tgtCompM2 := RevisionComponent{RevisionID: revTgt.ID, MaterialID: matM2.ID, Role: "M"}
		assert.NoError(t, database.Create(&tgtCompM1).Error)
		assert.NoError(t, database.Create(&tgtCompM2).Error)
		tgtCompM1Sec := RevisionComponent{RevisionID: revTgt.ID, MaterialID: matM1Sec.ID, Role: "S", ParentComponentID: tgtCompM1.ID}
		assert.NoError(t, database.Create(&tgtCompM1Sec).Error)

		// 建立來源機種
		srcModA := MatrixModel{RevisionID: revSrc.ID, SortOrder: 0, ModelName: "Src A Name", Qty: 100, IsHidden: false}
		srcModB := MatrixModel{RevisionID: revSrc.ID, SortOrder: 1, ModelName: "Src B Name", Qty: 0, IsHidden: false}
		srcModEmpty := MatrixModel{RevisionID: revSrc.ID, SortOrder: 2, ModelName: "Src C Empty", Qty: 50, IsHidden: false}
		srcModHidden := MatrixModel{RevisionID: revSrc.ID, SortOrder: 3, ModelName: "Src D Hidden", Qty: 10, IsHidden: true}
		assert.NoError(t, database.Create(&srcModA).Error)
		assert.NoError(t, database.Create(&srcModB).Error)
		// MatrixModel.Qty 欄位帶有 gorm:"default:1"，建立後顯式更新為 0 以模擬實際 Qty=0 邊界
		assert.NoError(t, database.Model(&srcModB).Updates(map[string]any{"qty": 0}).Error)
		srcModB.Qty = 0
		assert.NoError(t, database.Create(&srcModEmpty).Error)
		assert.NoError(t, database.Create(&srcModHidden).Error)

		// 來源 Model A 勾選：M1->M1-SEC (合法), M2->M2-SEC-MISSING-IN-TGT (目標缺替代料), SRC-ONLY->SRC-ONLY (目標缺主料)
		assert.NoError(t, database.Create(&MatrixSelection{RevisionID: revSrc.ID, ModelID: srcModA.ID, ComponentID: srcCompM1.ID, MainMaterialID: matM1.ID, SelectedMaterialID: matM1Sec.ID}).Error)
		assert.NoError(t, database.Create(&MatrixSelection{RevisionID: revSrc.ID, ModelID: srcModA.ID, ComponentID: srcCompM2.ID, MainMaterialID: matM2.ID, SelectedMaterialID: matM2SecOnlySrc.ID}).Error)
		assert.NoError(t, database.Create(&MatrixSelection{RevisionID: revSrc.ID, ModelID: srcModA.ID, ComponentID: srcCompOnly.ID, MainMaterialID: matOnlySrc.ID, SelectedMaterialID: matOnlySrc.ID}).Error)

		// 來源 Model B 勾選：M1->M1, M2->M2 (皆在目標與來源合法)
		assert.NoError(t, database.Create(&MatrixSelection{RevisionID: revSrc.ID, ModelID: srcModB.ID, ComponentID: srcCompM1.ID, MainMaterialID: matM1.ID, SelectedMaterialID: matM1.ID}).Error)
		assert.NoError(t, database.Create(&MatrixSelection{RevisionID: revSrc.ID, ModelID: srcModB.ID, ComponentID: srcCompM2.ID, MainMaterialID: matM2.ID, SelectedMaterialID: matM2.ID}).Error)

		// 建立目標機種（皆預先有舊勾選）
		tgtModA := MatrixModel{RevisionID: revTgt.ID, SortOrder: 0, ModelName: "Keep Target A Name", Qty: 20, IsHidden: false}
		tgtModB := MatrixModel{RevisionID: revTgt.ID, SortOrder: 1, ModelName: "Keep Target B Name", Qty: 30, IsHidden: false}
		tgtModC := MatrixModel{RevisionID: revTgt.ID, SortOrder: 2, ModelName: "Keep Target C Name", Qty: 40, IsHidden: false}
		tgtModHidden := MatrixModel{RevisionID: revTgt.ID, SortOrder: 3, ModelName: "Hidden Target D", Qty: 5, IsHidden: true}
		assert.NoError(t, database.Create(&tgtModA).Error)
		assert.NoError(t, database.Create(&tgtModB).Error)
		assert.NoError(t, database.Create(&tgtModC).Error)
		assert.NoError(t, database.Create(&tgtModHidden).Error)

		// 目標 A/B/C 預先有 M1->M1, M2->M2 舊勾選
		for _, tm := range []MatrixModel{tgtModA, tgtModB, tgtModC} {
			assert.NoError(t, database.Create(&MatrixSelection{RevisionID: revTgt.ID, ModelID: tm.ID, ComponentID: tgtCompM1.ID, MainMaterialID: matM1.ID, SelectedMaterialID: matM1.ID}).Error)
			assert.NoError(t, database.Create(&MatrixSelection{RevisionID: revTgt.ID, ModelID: tm.ID, ComponentID: tgtCompM2.ID, MainMaterialID: matM2.ID, SelectedMaterialID: matM2.ID}).Error)
		}

		return copyFixture{
			db:           database,
			revSrc:       revSrc,
			revTgt:       revTgt,
			matM1:        matM1,
			matM1Sec:     matM1Sec,
			matM2:        matM2,
			matOnlySrc:   matOnlySrc,
			srcModA:      srcModA,
			srcModB:      srcModB,
			srcModEmpty:  srcModEmpty,
			srcModHidden: srcModHidden,
			tgtModA:      tgtModA,
			tgtModB:      tgtModB,
			tgtModC:      tgtModC,
			tgtModHidden: tgtModHidden,
		}
	}

	t.Run("跨版本1對1複製_CopyQty=false保留目標Qty與ModelName且未指定Model完整保留", func(t *testing.T) {
		f := setupFixture(t)
		stats, err := CopyMatrixModelSelections(f.db, CopyMatrixModelsParams{
			SourceRevisionID: f.revSrc.ID,
			TargetRevisionID: f.revTgt.ID,
			Mappings:         []MatrixModelCopyMapping{{SourceModelID: f.srcModA.ID, TargetModelID: f.tgtModA.ID}},
			CopyQty:          false,
		}, nil)
		assert.NoError(t, err)
		assert.Equal(t, 1, stats.TotalMappings)
		assert.Equal(t, 1, stats.CopiedSelectionsCount)
		assert.Equal(t, 1, stats.IgnoredMainParts)
		assert.Equal(t, 1, stats.IgnoredSecondParts)

		// 驗證目標 Model A 原 Name 與 Qty 皆保留，且僅有 1 筆新 Selection (IsAutoSelected=true)
		tgtA, err := GetMatrixModel(f.db, f.tgtModA.ID)
		assert.NoError(t, err)
		assert.Equal(t, "Keep Target A Name", tgtA.ModelName)
		assert.Equal(t, 20, tgtA.Qty)

		selsA, err := GetMatrixSelections(f.db, f.revTgt.ID, f.tgtModA.ID)
		assert.NoError(t, err)
		assert.Len(t, selsA, 1)
		assert.Equal(t, f.matM1.ID, selsA[0].MainMaterialID)
		assert.Equal(t, f.matM1Sec.ID, selsA[0].SelectedMaterialID)
		assert.True(t, selsA[0].IsAutoSelected)

		// 驗證未指定的目標 Model B 與 C 原有 2 筆勾選完整保留
		selsB, err := GetMatrixSelections(f.db, f.revTgt.ID, f.tgtModB.ID)
		assert.NoError(t, err)
		assert.Len(t, selsB, 2)
		selsC, err := GetMatrixSelections(f.db, f.revTgt.ID, f.tgtModC.ID)
		assert.NoError(t, err)
		assert.Len(t, selsC, 2)
	})

	t.Run("CopyQty=true複製_支援覆寫為Qty=0且絕不更動ModelName", func(t *testing.T) {
		f := setupFixture(t)
		// srcModB 的 Qty 為 0，目標 tgtModA 原 Qty 為 20
		stats, err := CopyMatrixModelSelections(f.db, CopyMatrixModelsParams{
			SourceRevisionID: f.revSrc.ID,
			TargetRevisionID: f.revTgt.ID,
			Mappings:         []MatrixModelCopyMapping{{SourceModelID: f.srcModB.ID, TargetModelID: f.tgtModA.ID}},
			CopyQty:          true,
		}, nil)
		assert.NoError(t, err)
		assert.Equal(t, 2, stats.CopiedSelectionsCount)
		assert.True(t, stats.PairStats[0].QtyCopied)
		assert.Equal(t, 0, stats.PairStats[0].CopiedQty)

		tgtA, err := GetMatrixModel(f.db, f.tgtModA.ID)
		assert.NoError(t, err)
		assert.Equal(t, 0, tgtA.Qty, "來源 Qty=0 時亦應精準覆寫目標 Qty 為 0")
		assert.Equal(t, "Keep Target A Name", tgtA.ModelName, "複製 Qty 不應更動目標 ModelName")
	})

	t.Run("跨版本1對多複製_同時複製到A與C並保留B", func(t *testing.T) {
		f := setupFixture(t)
		stats, err := CopyMatrixModelSelections(f.db, CopyMatrixModelsParams{
			SourceRevisionID: f.revSrc.ID,
			TargetRevisionID: f.revTgt.ID,
			Mappings: []MatrixModelCopyMapping{
				{SourceModelID: f.srcModA.ID, TargetModelID: f.tgtModA.ID},
				{SourceModelID: f.srcModA.ID, TargetModelID: f.tgtModC.ID},
			},
			CopyQty: true,
		}, nil)
		assert.NoError(t, err)
		assert.Equal(t, 2, stats.TotalMappings)
		assert.Equal(t, 2, stats.CopiedSelectionsCount)

		for _, mid := range []int64{f.tgtModA.ID, f.tgtModC.ID} {
			m, err := GetMatrixModel(f.db, mid)
			assert.NoError(t, err)
			assert.Equal(t, 100, m.Qty)
			sels, err := GetMatrixSelections(f.db, f.revTgt.ID, mid)
			assert.NoError(t, err)
			assert.Len(t, sels, 1)
			assert.Equal(t, f.matM1Sec.ID, sels[0].SelectedMaterialID)
		}

		// 未指定的 B 維持原有 Qty=30 與 2 筆勾選
		tgtB, err := GetMatrixModel(f.db, f.tgtModB.ID)
		assert.NoError(t, err)
		assert.Equal(t, 30, tgtB.Qty)
		selsB, err := GetMatrixSelections(f.db, f.revTgt.ID, f.tgtModB.ID)
		assert.NoError(t, err)
		assert.Len(t, selsB, 2)
	})

	t.Run("同版本內單向與連鎖互換複製_快照讀取不受寫入順序影響", func(t *testing.T) {
		f := setupFixture(t)
		// 在 revSrc 內建立 3 個狀態各自不同的 Model：
		// srcModA: Qty=100, M1->M1-SEC (共 3 筆)
		// srcModB: Qty=0,   M1->M1, M2->M2 (共 2 筆)
		// srcModEmpty: Qty=50, 0 筆勾選
		// 同批執行：A→B, B→C, C→A（3 節點輪換），驗證快照語意完全不互蓋！
		stats, err := CopyMatrixModelSelections(f.db, CopyMatrixModelsParams{
			SourceRevisionID: f.revSrc.ID,
			TargetRevisionID: f.revSrc.ID,
			Mappings: []MatrixModelCopyMapping{
				{SourceModelID: f.srcModA.ID, TargetModelID: f.srcModB.ID},
				{SourceModelID: f.srcModB.ID, TargetModelID: f.srcModEmpty.ID},
				{SourceModelID: f.srcModEmpty.ID, TargetModelID: f.srcModA.ID},
			},
			CopyQty: true,
		}, nil)
		assert.NoError(t, err)
		assert.Equal(t, 5, stats.CopiedSelectionsCount) // B 得 A 的 3 筆 + C 得 B 的 2 筆 + A 得 C 的 0 筆

		// A 應變成原本 C 的狀態 (Qty=50, 0 筆勾選)
		modA, _ := GetMatrixModel(f.db, f.srcModA.ID)
		selsA, _ := GetMatrixSelections(f.db, f.revSrc.ID, f.srcModA.ID)
		assert.Equal(t, 50, modA.Qty)
		assert.Empty(t, selsA)

		// B 應變成原本 A 的狀態 (Qty=100, 3 筆勾選)
		modB, _ := GetMatrixModel(f.db, f.srcModB.ID)
		selsB, _ := GetMatrixSelections(f.db, f.revSrc.ID, f.srcModB.ID)
		assert.Equal(t, 100, modB.Qty)
		assert.Len(t, selsB, 3)

		// C 應變成原本 B 的狀態 (Qty=0, 2 筆勾選)
		modC, _ := GetMatrixModel(f.db, f.srcModEmpty.ID)
		selsC, _ := GetMatrixSelections(f.db, f.revSrc.ID, f.srcModEmpty.ID)
		assert.Equal(t, 0, modC.Qty)
		assert.Len(t, selsC, 2)
	})

	t.Run("來源Model為0筆勾選時_會清空目標Model原有勾選", func(t *testing.T) {
		f := setupFixture(t)
		stats, err := CopyMatrixModelSelections(f.db, CopyMatrixModelsParams{
			SourceRevisionID: f.revSrc.ID,
			TargetRevisionID: f.revTgt.ID,
			Mappings:         []MatrixModelCopyMapping{{SourceModelID: f.srcModEmpty.ID, TargetModelID: f.tgtModA.ID}},
			CopyQty:          false,
		}, nil)
		assert.NoError(t, err)
		assert.Equal(t, 0, stats.CopiedSelectionsCount)

		selsA, err := GetMatrixSelections(f.db, f.revTgt.ID, f.tgtModA.ID)
		assert.NoError(t, err)
		assert.Empty(t, selsA, "來源 0 筆勾選應清空目標 Model A 原有勾選")
	})

	t.Run("防呆錯誤案例_驗證錯誤回報且目標資料未被破壞", func(t *testing.T) {
		f := setupFixture(t)
		errorCases := []struct {
			name   string
			params CopyMatrixModelsParams
		}{
			{
				name: "空Mappings",
				params: CopyMatrixModelsParams{
					SourceRevisionID: f.revSrc.ID, TargetRevisionID: f.revTgt.ID, Mappings: nil,
				},
			},
			{
				name: "無效RevisionID",
				params: CopyMatrixModelsParams{
					SourceRevisionID: 0, TargetRevisionID: f.revTgt.ID,
					Mappings: []MatrixModelCopyMapping{{SourceModelID: f.srcModA.ID, TargetModelID: f.tgtModA.ID}},
				},
			},
			{
				name: "無效ModelID",
				params: CopyMatrixModelsParams{
					SourceRevisionID: f.revSrc.ID, TargetRevisionID: f.revTgt.ID,
					Mappings: []MatrixModelCopyMapping{{SourceModelID: 0, TargetModelID: f.tgtModA.ID}},
				},
			},
			{
				name: "重複指定TargetModelID",
				params: CopyMatrixModelsParams{
					SourceRevisionID: f.revSrc.ID, TargetRevisionID: f.revTgt.ID,
					Mappings: []MatrixModelCopyMapping{
						{SourceModelID: f.srcModA.ID, TargetModelID: f.tgtModA.ID},
						{SourceModelID: f.srcModB.ID, TargetModelID: f.tgtModA.ID},
					},
				},
			},
			{
				name: "同版本相同ModelID",
				params: CopyMatrixModelsParams{
					SourceRevisionID: f.revSrc.ID, TargetRevisionID: f.revSrc.ID,
					Mappings: []MatrixModelCopyMapping{{SourceModelID: f.srcModA.ID, TargetModelID: f.srcModA.ID}},
				},
			},
			{
				name: "來源Model不屬於來源Revision",
				params: CopyMatrixModelsParams{
					SourceRevisionID: f.revSrc.ID, TargetRevisionID: f.revTgt.ID,
					Mappings: []MatrixModelCopyMapping{{SourceModelID: f.tgtModA.ID, TargetModelID: f.tgtModA.ID}},
				},
			},
			{
				name: "來源Model為隱藏狀態",
				params: CopyMatrixModelsParams{
					SourceRevisionID: f.revSrc.ID, TargetRevisionID: f.revTgt.ID,
					Mappings: []MatrixModelCopyMapping{{SourceModelID: f.srcModHidden.ID, TargetModelID: f.tgtModA.ID}},
				},
			},
			{
				name: "目標Model為隱藏狀態",
				params: CopyMatrixModelsParams{
					SourceRevisionID: f.revSrc.ID, TargetRevisionID: f.revTgt.ID,
					Mappings: []MatrixModelCopyMapping{{SourceModelID: f.srcModA.ID, TargetModelID: f.tgtModHidden.ID}},
				},
			},
		}

		for _, tc := range errorCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := CopyMatrixModelSelections(f.db, tc.params, nil)
				assert.Error(t, err)
			})
		}

		// 確認所有失敗案例均未破壞目標 Model A 原有的 2 筆勾選與 Qty
		tgtA, _ := GetMatrixModel(f.db, f.tgtModA.ID)
		selsA, _ := GetMatrixSelections(f.db, f.revTgt.ID, f.tgtModA.ID)
		assert.Equal(t, 20, tgtA.Qty)
		assert.Len(t, selsA, 2)
	})
}

