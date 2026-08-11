package backend

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"bomix-app/backend/config"
	"bomix-app/backend/db"
	"bomix-app/backend/logger"
)

// TestGetRecentSeries_MissingAndCorruptedFiles 測試最近開啟檔案的過濾與損毀標記邏輯
func TestGetRecentSeries_MissingAndCorruptedFiles(t *testing.T) {
	tempDir := t.TempDir()

	// 1. 建立一個正常的 BOMIX 資料庫檔案
	validPath := filepath.Join(tempDir, "valid.bomx")
	validDB, err := db.Open(validPath)
	if err != nil {
		t.Fatalf("無法建立測試資料庫: %v", err)
	}
	if err := db.AutoMigrate(validDB); err != nil {
		db.Close(validDB)
		t.Fatalf("無法初始化資料庫結構: %v", err)
	}
	_, err = db.CreateSeries(validDB, "測試系列 A", "描述 A")
	if err != nil {
		db.Close(validDB)
		t.Fatalf("無法建立系列資訊: %v", err)
	}
	db.Close(validDB)

	// 2. 建立一個損毀/無效的檔案 (純文字檔，非 SQLite 資料庫)
	corruptedPath := filepath.Join(tempDir, "corrupted.bomx")
	if err := os.WriteFile(corruptedPath, []byte("this is not a sqlite db"), 0644); err != nil {
		t.Fatalf("無法建立損毀測試檔: %v", err)
	}

	// 3. 設定一個不存在的檔案路徑
	missingPath := filepath.Join(tempDir, "non_existent_file.bomx")

	// 初始化測試 App 與 Config
	log := logger.NewLogger(100)
	cfg := &config.Config{
		RecentFiles: config.RecentFilesConfig{
			MaxRecentFiles: 10,
			RecentFiles:    []string{missingPath, corruptedPath, validPath},
		},
	}
	app := NewApp(nil, log, cfg)

	// 執行 GetRecentSeries
	recentFiles, err := app.GetRecentSeries()
	if err != nil {
		t.Fatalf("GetRecentSeries 傳回非預期錯誤: %v", err)
	}

	// 檢查 1：不存在的檔案路徑是否已被排除在 TOML/Config 的 RecentFiles 清單中
	if len(cfg.RecentFiles.RecentFiles) != 2 {
		t.Errorf("記憶體設定中的 RecentFiles 長度 = %d，預期為 2 (應該只保留存在的兩個檔案)", len(cfg.RecentFiles.RecentFiles))
	}

	// 檢查 2：絕對不能在 missingPath 上自動建立空檔案！
	if _, err := os.Stat(missingPath); !os.IsNotExist(err) {
		t.Errorf("GetRecentSeries() 建立了空檔案: %s，預期該檔案不應被建立", missingPath)
	}

	// 檢查 3：回傳給 UI 的 RecentFile 列表應為 2 筆
	if len(recentFiles) != 2 {
		t.Fatalf("GetRecentSeries() 回傳筆數 = %d，預期為 2", len(recentFiles))
	}

	// 驗證損毀檔案項目
	corruptedItem := recentFiles[0]
	if corruptedItem.Path != corruptedPath {
		t.Errorf("第 1 筆檔案路徑 = %q, 預期 %q", corruptedItem.Path, corruptedPath)
	}
	if !corruptedItem.IsCorrupted {
		t.Errorf("損毀檔案的 IsCorrupted 應為 true，實際為 false")
	}

	// 驗證正常檔案項目
	validItem := recentFiles[1]
	if validItem.Path != validPath {
		t.Errorf("第 2 筆檔案路徑 = %q, 預期 %q", validItem.Path, validPath)
	}
	if validItem.IsCorrupted {
		t.Errorf("正常檔案的 IsCorrupted 應為 false，實際為 true")
	}
	if validItem.Name != "測試系列 A" {
		t.Errorf("正常檔案的 Name = %q, 預期 %q", validItem.Name, "測試系列 A")
	}
}

// TestExportExcel_MultipleMatrixRevisionsTasks 測試當選取多個 BOM Revision 匯出 Matrix 時，
// App.ExportExcel 應會為每一個 selected Revision 依序建立獨立的 Task 任務與檔案
func TestExportExcel_MultipleMatrixRevisionsTasks(t *testing.T) {
	tempDir := t.TempDir()

	validPath := filepath.Join(tempDir, "test_matrix_multi.bomx")
	validDB, err := db.Open(validPath)
	if err != nil {
		t.Fatalf("無法建立測試資料庫: %v", err)
	}
	if err := db.AutoMigrate(validDB); err != nil {
		db.Close(validDB)
		t.Fatalf("無法初始化資料庫結構: %v", err)
	}

	series, err := db.CreateSeries(validDB, "Matrix Multi Test Series", "Series Desc")
	if err != nil {
		db.Close(validDB)
		t.Fatalf("無法建立 Series: %v", err)
	}
	proj := &db.Project{SeriesID: series.ID, Code: "PROJ_MULT", Description: "Proj Desc"}
	if err := validDB.Create(proj).Error; err != nil {
		db.Close(validDB)
		t.Fatalf("無法建立 Project: %v", err)
	}
	rev1, err := db.CreateRevision(validDB, proj.ID, "EVT", "0.1", "Rev 1 Desc")
	if err != nil {
		db.Close(validDB)
		t.Fatalf("無法建立 Rev 1: %v", err)
	}
	rev2, err := db.CreateRevision(validDB, proj.ID, "DVT", "0.2", "Rev 2 Desc")
	if err != nil {
		db.Close(validDB)
		t.Fatalf("無法建立 Rev 2: %v", err)
	}

	log := logger.NewLogger(100)
	cfg := &config.Config{}
	app := NewApp(nil, log, cfg)
	app.db = validDB
	defer db.Close(validDB)

	exportOpts := &ExportOptions{
		Format:      "Matrix",
		RevisionIDs: []int64{rev1.ID, rev2.ID},
		OutputDir:   tempDir,
	}

	taskIDs, err := app.ExportExcel(exportOpts)
	if err != nil {
		t.Fatalf("ExportExcel 失敗: %v", err)
	}

	if len(taskIDs) != 2 {
		t.Fatalf("多 Revision 匯出 Matrix 預期回傳 2 個 Task ID，實際得到 %d 個: %v", len(taskIDs), taskIDs)
	}

	tasks, err := app.ListTasks()
	if err != nil {
		t.Fatalf("ListTasks 失敗: %v", err)
	}

	if len(tasks) < 2 {
		t.Fatalf("Task Manager 中預期至少有 2 個任務，實際有 %d 個", len(tasks))
	}

	foundTask1 := false
	foundTask2 := false
	for _, tk := range tasks {
		if tk.ID == taskIDs[0] {
			foundTask1 = true
		}
		if tk.ID == taskIDs[1] {
			foundTask2 = true
		}
	}

	if !foundTask1 || !foundTask2 {
		t.Errorf("Task Manager 中未正確註冊多個 Matrix 匯出 Task ID: %v vs %v", taskIDs, tasks)
	}

	// 等待非同步任務執行完成
	for i := 0; i < 50; i++ {
		allDone := true
		for _, tk := range app.taskMgr.ListTasks() {
			if tk.Status == "Running" || tk.Status == "Queued" {
				allDone = false
				break
			}
		}
		if allDone {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// TestSaveProjectExportOrder 驗證 Project 匯出排序紀錄之即時儲存與 GetSeriesInfo 讀取
func TestSaveProjectExportOrder(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_order.bomx")

	testDB, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("無法建立測試資料庫: %v", err)
	}
	defer db.Close(testDB)

	if err := db.AutoMigrate(testDB); err != nil {
		t.Fatalf("無法初始化資料庫結構: %v", err)
	}
	if _, err := db.CreateSeries(testDB, "Test Series", "Desc"); err != nil {
		t.Fatalf("無法建立 Series: %v", err)
	}

	app := &App{
		db:     testDB,
		logger: logger.NewLogger(100),
		cfg:    &config.Config{},
	}

	// 1. 初始狀態 ProjectExportOrder 應為空陣列
	info, err := app.GetSeriesInfo()
	if err != nil {
		t.Fatalf("GetSeriesInfo 失敗: %v", err)
	}
	if len(info.ProjectExportOrder) != 0 {
		t.Errorf("初始 ProjectExportOrder 應為空，實際 got %v", info.ProjectExportOrder)
	}

	// 2. 儲存 Project 匯出排序紀錄
	expectedOrder := []string{"PROJ_B", "PROJ_A", "PROJ_C"}
	if err := app.SaveProjectExportOrder(expectedOrder); err != nil {
		t.Fatalf("SaveProjectExportOrder 失敗: %v", err)
	}

	// 3. 再次讀取並驗證
	info2, err := app.GetSeriesInfo()
	if err != nil {
		t.Fatalf("GetSeriesInfo 失敗: %v", err)
	}
	if len(info2.ProjectExportOrder) != 3 {
		t.Fatalf("期望 ProjectExportOrder 元素數為 3，實際 got %d", len(info2.ProjectExportOrder))
	}
	for i, code := range expectedOrder {
		if info2.ProjectExportOrder[i] != code {
			t.Errorf("Index %d 期望 %s，實際 got %s", i, code, info2.ProjectExportOrder[i])
		}
	}
}

