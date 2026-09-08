package backend

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"bomix-app/backend/config"
	"bomix-app/backend/db"
	"bomix-app/backend/logger"
	"bomix-app/backend/types"
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

	// 2. 儲存 Project 匯出排序與 Model 數量紀錄
	expectedSettings := []ProjectExportSetting{
		{ProjectCode: "PROJ_B", ModelCount: 4},
		{ProjectCode: "PROJ_A", ModelCount: 2},
		{ProjectCode: "PROJ_C", ModelCount: 5},
	}
	if err := app.SaveProjectExportOrder(expectedSettings); err != nil {
		t.Fatalf("SaveProjectExportOrder 失敗: %v", err)
	}

	// 3. 再次讀取並驗證 Project 順序與 ModelCounts
	info2, err := app.GetSeriesInfo()
	if err != nil {
		t.Fatalf("GetSeriesInfo 失敗: %v", err)
	}
	if len(info2.ProjectExportOrder) != 3 {
		t.Fatalf("期望 ProjectExportOrder 元素數為 3，實際 got %d", len(info2.ProjectExportOrder))
	}
	expectedOrder := []string{"PROJ_B", "PROJ_A", "PROJ_C"}
	for i, code := range expectedOrder {
		if info2.ProjectExportOrder[i] != code {
			t.Errorf("Index %d 期望 %s，實際 got %s", i, code, info2.ProjectExportOrder[i])
		}
	}
	if info2.ProjectModelCounts["PROJ_B"] != 4 {
		t.Errorf("期望 PROJ_B ModelCount=4，實際 got %d", info2.ProjectModelCounts["PROJ_B"])
	}
	if info2.ProjectModelCounts["PROJ_A"] != 2 {
		t.Errorf("期望 PROJ_A ModelCount=2，實際 got %d", info2.ProjectModelCounts["PROJ_A"])
	}
}

// TestIsMatrixFile 測試檔案名稱之 Matrix 判定邏輯（不區分大小寫，且僅以檔名為主）
func TestIsMatrixFile(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{"純小寫 matrix", "test_matrix.xlsx", true},
		{"純大寫 MATRIX", "PROJ_MATRIX_BOM.XLSX", true},
		{"大小寫混雜 Matrix", "Project_Matrix_V1.xlsx", true},
		{"BigMatrix", "system_bigmatrix_2026.xlsx", true},
		{"一般 EBOM 檔名", "PROJ_M_EZBOM_EVT_0.1.xlsx", false},
		{"一般純英文檔名", "sample_parts_list.xls", false},
		{"路徑目錄含 matrix 但檔名不含", "C:\\matrix_dir\\normal_ebom.xlsx", false},
		{"Unix 風格路徑包含 Matrix 檔名", "/home/user/docs/EVT_Matrix.xlsx", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := isMatrixFile(tc.filePath)
			if actual != tc.expected {
				t.Errorf("isMatrixFile(%q) = %v, want %v", tc.filePath, actual, tc.expected)
			}
		})
	}
}

// TestImportExcel_TwoPhaseGrouping 測試 ImportExcel 之分組與分階段背景任務機制
func TestImportExcel_TwoPhaseGrouping(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_import_grouping.bomx")

	testDB, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("無法建立測試資料庫: %v", err)
	}
	defer db.Close(testDB)

	if err := db.AutoMigrate(testDB); err != nil {
		t.Fatalf("無法初始化資料庫結構: %v", err)
	}
	if _, err := db.CreateSeries(testDB, "Import Grouping Test Series", "Desc"); err != nil {
		t.Fatalf("無法建立 Series: %v", err)
	}

	log := logger.NewLogger(100)
	cfg := &config.Config{}
	app := NewApp(nil, log, cfg)
	app.db = testDB

	// 建立臨時假檔案 (一個非 matrix，一個 matrix)
	fileEBOM := filepath.Join(tempDir, "PROJ_EZBOM_EVT.xlsx")
	fileMatrix := filepath.Join(tempDir, "PROJ_Matrix_EVT.xlsx")
	_ = os.WriteFile(fileEBOM, []byte("fake ebom content"), 0644)
	_ = os.WriteFile(fileMatrix, []byte("fake matrix content"), 0644)

	inputFiles := []string{fileMatrix, fileEBOM}

	// 呼叫 ImportExcel
	results, err := app.ImportExcel(inputFiles, false)
	if err != nil {
		t.Fatalf("ImportExcel 執行失敗: %v", err)
	}

	// 驗證回傳結果筆數
	if len(results) != 2 {
		t.Fatalf("預期回傳 2 筆結果，實際得到 %d 筆", len(results))
	}

	// 驗證 Task 是否皆已成功註冊至 TaskManager
	for _, res := range results {
		if res.TaskID == "" {
			t.Errorf("檔案 %s 的 TaskID 不應為空", res.FileName)
		}
		status := app.taskMgr.GetStatus(res.TaskID)
		if status == nil {
			t.Errorf("Task %s 未在 TaskManager 中找到", res.TaskID)
		}
	}

	// 等待背景任務執行結束
	for i := 0; i < 50; i++ {
		allDone := true
		for _, res := range results {
			st := app.taskMgr.GetStatus(res.TaskID)
			if st != nil && (st.Status == "Running" || st.Status == "Queued" || st.Status == "Created") {
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

// TestAppImportExcel_TarisPCM_RealFiles 測試使用真實 TARIS-PCM EBOM (.xls) 與 Matrix (.xlsx) 檔案進行兩階段批次匯入
func TestAppImportExcel_TarisPCM_RealFiles(t *testing.T) {
	ebomPath := filepath.Join("excel", "testdata", "TARIS-PCM_EZBOM_SI1_0.3_BOM_20260817_1000.WP(compared).xls")
	matrixPath := filepath.Join("excel", "testdata", "TARIS-PCM_EZBOM_SI1_0.3_MatrixBOM_20260817_1000.WP.xlsx")

	absEBOM, err := filepath.Abs(ebomPath)
	if err != nil {
		t.Fatalf("路徑錯誤: %v", err)
	}
	absMatrix, err := filepath.Abs(matrixPath)
	if err != nil {
		t.Fatalf("路徑錯誤: %v", err)
	}

	l := logger.NewLogger(100)
	tempDBPath := filepath.Join(t.TempDir(), "test_app_taris_real.bomx")
	testDB, err := db.Open(tempDBPath)
	if err != nil {
		t.Fatalf("建立測試 DB 失敗: %v", err)
	}
	defer db.Close(testDB)
	if err := db.AutoMigrate(testDB); err != nil {
		t.Fatalf("AutoMigrate 失敗: %v", err)
	}
	_, err = db.CreateSeries(testDB, "abc", "Desc")
	if err != nil {
		t.Fatalf("CreateSeries 失敗: %v", err)
	}

	app := NewApp(nil, l, &config.Config{})
	app.db = testDB

	// 同時傳入兩個檔案進行兩階段排程匯入
	results, err := app.ImportExcel([]string{absEBOM, absMatrix}, false)
	if err != nil {
		t.Fatalf("App.ImportExcel 失敗: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("預期提交 2 個任務，實際為 %d", len(results))
	}

	// 等待所有任務完成
	for i := 0; i < 200; i++ {
		allDone := true
		for _, r := range results {
			st := app.taskMgr.GetStatus(r.TaskID)
			if st != nil && (st.Status == "Running" || st.Status == "Queued" || st.Status == "Created") {
				allDone = false
				break
			}
		}
		if allDone {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// 驗證兩個任務皆順利完成（EBOM 任務因 MP sheet 含有 Phase 1 未建立的位號 U19/U68 應為 Warning，Matrix 應為 Completed）
	for _, r := range results {
		st := app.taskMgr.GetStatus(r.TaskID)
		if st == nil {
			t.Fatalf("未找到任務 %s", r.TaskID)
		}
		if strings.Contains(st.Name, "Matrix") {
			if st.Status != string(types.TaskCompleted) {
				t.Errorf("任務 %s (%s) 狀態應為 Completed，實際為 %s，錯誤: %s", st.ID, st.Name, st.Status, st.Error)
			}
		} else {
			if st.Status != string(types.TaskWarning) {
				t.Errorf("EBOM 任務 %s (%s) 狀態應為 Warning，實際為 %s，錯誤: %s", st.ID, st.Name, st.Status, st.Error)
			}
		}
	}
}

// TestAppImportExcel_ConfirmOverwrite 測試當開啟 confirmOverwrite 且發現既有版本時的暫停與確認流程
func TestAppImportExcel_ConfirmOverwrite(t *testing.T) {
	ebomPath := filepath.Join("excel", "testdata", "TARIS-PCM_EZBOM_SI1_0.3_BOM_20260817_1000.WP(compared).xls")
	absEBOM, err := filepath.Abs(ebomPath)
	if err != nil {
		t.Fatalf("路徑錯誤: %v", err)
	}

	l := logger.NewLogger(100)
	tempDBPath := filepath.Join(t.TempDir(), "test_app_confirm_overwrite.bomx")
	testDB, err := db.Open(tempDBPath)
	if err != nil {
		t.Fatalf("建立測試 DB 失敗: %v", err)
	}
	defer db.Close(testDB)
	if err := db.AutoMigrate(testDB); err != nil {
		t.Fatalf("AutoMigrate 失敗: %v", err)
	}
	_, err = db.CreateSeries(testDB, "abc", "Desc")
	if err != nil {
		t.Fatalf("CreateSeries 失敗: %v", err)
	}

	app := NewApp(nil, l, &config.Config{})
	app.db = testDB

	// 第一次匯入：建立版本
	results1, err := app.ImportExcel([]string{absEBOM}, false)
	if err != nil {
		t.Fatalf("第一次匯入失敗: %v", err)
	}
	for i := 0; i < 100; i++ {
		st := app.taskMgr.GetStatus(results1[0].TaskID)
		if st != nil && (st.Status == "Completed" || st.Status == "Warning") {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// 第二次匯入：開啟 confirmOverwrite = true，預期任務會暫停進入 WaitingConfirm 狀態
	results2, err := app.ImportExcel([]string{absEBOM}, true)
	if err != nil {
		t.Fatalf("第二次匯入失敗: %v", err)
	}
	taskID := results2[0].TaskID

	// 等待任務進入 WaitingConfirm 狀態
	isWaiting := false
	for i := 0; i < 100; i++ {
		st := app.taskMgr.GetStatus(taskID)
		if st != nil && st.Status == string(types.TaskWaitingConfirm) {
			isWaiting = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if !isWaiting {
		t.Fatalf("任務未如預期進入 WaitingConfirm 狀態")
	}

	// 呼叫 ConfirmTaskOverwrite 解除阻塞確認覆蓋
	if err := app.ConfirmTaskOverwrite(taskID, true); err != nil {
		t.Fatalf("ConfirmTaskOverwrite 失敗: %v", err)
	}

	// 等待任務順利完成
	isCompleted := false
	for i := 0; i < 100; i++ {
		st := app.taskMgr.GetStatus(taskID)
		if st != nil && (st.Status == "Completed" || st.Status == "Warning") {
			isCompleted = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if !isCompleted {
		t.Fatalf("確認覆蓋後任務未順利完成")
	}
}

// TestAppImportExcel_PartialWaitingConfirm_OthersContinue 測試當 Phase 1 多個檔案中僅有部分需要確認時，不需要確認的任務能獨立執行完成，不會被設定為 WaitingConfirm
func TestAppImportExcel_PartialWaitingConfirm_OthersContinue(t *testing.T) {
	ebomPath := filepath.Join("excel", "testdata", "TARIS-PCM_EZBOM_SI1_0.3_BOM_20260817_1000.WP(compared).xls")
	absEBOM, err := filepath.Abs(ebomPath)
	if err != nil {
		t.Fatalf("路徑錯誤: %v", err)
	}

	l := logger.NewLogger(100)
	tempDBPath := filepath.Join(t.TempDir(), "test_app_partial_confirm.bomx")
	testDB, err := db.Open(tempDBPath)
	if err != nil {
		t.Fatalf("建立測試 DB 失敗: %v", err)
	}
	defer db.Close(testDB)
	if err := db.AutoMigrate(testDB); err != nil {
		t.Fatalf("AutoMigrate 失敗: %v", err)
	}
	series, err := db.CreateSeries(testDB, "abc", "Desc")
	if err != nil {
		t.Fatalf("CreateSeries 失敗: %v", err)
	}

	// 預先在資料庫建立 TARIS-PCM SI1 0.3 版本，模擬該版本已存在
	project, err := db.GetOrCreateProject(testDB, series.ID, "TARIS-PCM", "Description")
	if err != nil {
		t.Fatalf("建立 Project 失敗: %v", err)
	}
	rev := db.BomRevision{
		ProjectID: project.ID,
		Phase:     "SI1",
		Version:   "0.3",
	}
	if err := testDB.Create(&rev).Error; err != nil {
		t.Fatalf("建立 Revision 失敗: %v", err)
	}

	app := NewApp(nil, l, &config.Config{})
	app.db = testDB

	// 同時提交兩個檔案：
	// 檔案 1: TARIS-PCM SI1 0.3（已存在，預期進入 WaitingConfirm）
	// 檔案 2: 虛構或不同檔案（假設也是非 Matrix 檔案，但若格式錯誤或不同，預期不會進入 WaitingConfirm，而是繼續執行至 Completed 或 Warning）
	tempDir := t.TempDir()
	fileNonExisting := filepath.Join(tempDir, "NEW_PROJ_EZBOM_EVT.xlsx")
	_ = os.WriteFile(fileNonExisting, []byte("fake content"), 0644)

	results, err := app.ImportExcel([]string{absEBOM, fileNonExisting}, true)
	if err != nil {
		t.Fatalf("ImportExcel 失敗: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("預期提交 2 個任務，實際為 %d", len(results))
	}

	taskEBOM_ID := results[0].TaskID
	taskOther_ID := results[1].TaskID

	// 等待一段時間，驗證：
	// 1. taskEBOM_ID 應進入 WaitingConfirm 狀態
	// 2. taskOther_ID 絕不可被設為 WaitingConfirm，而應正常執行結束（Warning 或 Failed，因為 fake content）
	for i := 0; i < 50; i++ {
		stEBOM := app.taskMgr.GetStatus(taskEBOM_ID)
		stOther := app.taskMgr.GetStatus(taskOther_ID)

		if stOther != nil && stOther.Status == string(types.TaskWaitingConfirm) {
			t.Fatalf("錯誤：不需要確認的任務 %s 被錯誤設定為 WaitingConfirm", taskOther_ID)
		}

		if stEBOM != nil && stEBOM.Status == string(types.TaskWaitingConfirm) {
			// EBOM 順利進入 WaitingConfirm
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// 再次確認 stOther 狀態絕不是 WaitingConfirm
	stOther := app.taskMgr.GetStatus(taskOther_ID)
	if stOther != nil && stOther.Status == string(types.TaskWaitingConfirm) {
		t.Errorf("taskOther_ID 狀態不應為 WaitingConfirm")
	}

	// 確認 EBOM 任務
	if err := app.ConfirmTaskOverwrite(taskEBOM_ID, true); err != nil {
		t.Fatalf("ConfirmTaskOverwrite 失敗: %v", err)
	}

	// 等待 EBOM 任務完成
	for i := 0; i < 100; i++ {
		stEBOM := app.taskMgr.GetStatus(taskEBOM_ID)
		if stEBOM != nil && (stEBOM.Status == "Completed" || stEBOM.Status == "Warning") {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	finalEBOM := app.taskMgr.GetStatus(taskEBOM_ID)
	if finalEBOM.Status != "Completed" && finalEBOM.Status != "Warning" {
		t.Errorf("EBOM 任務預期 Completed 或 Warning，實際為 %s", finalEBOM.Status)
	}
}



