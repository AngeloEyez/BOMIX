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
