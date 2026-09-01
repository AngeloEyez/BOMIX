package backend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"bomix-app/backend/config"
	"bomix-app/backend/db"
	"bomix-app/backend/excel"
	"bomix-app/backend/logger"
	"bomix-app/backend/task"
	"bomix-app/backend/types"
	"bomix-app/backend/view"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v3/pkg/application"
	"gorm.io/gorm"
)

// App struct for Wails bindings
type App struct {
	app     *application.App
	logger  *logger.Logger
	cfg     *config.Config
	taskMgr *task.Manager
	db      *gorm.DB
	mu      sync.RWMutex
}

// NewApp creates a new App instance
func NewApp(wailsApp *application.App, logger *logger.Logger, cfg *config.Config) *App {
	app := &App{
		app:    wailsApp,
		logger: logger,
		cfg:    cfg,
	}

	// Set logger event callback
	if logger != nil {
		logger.SetEventCallback(app.EmitEvent)
	}

	// Create task manager with app as the event emitter
	app.taskMgr = task.NewManager(logger, app)

	return app
}

// EmitEvent emits an event to the frontend via Wails runtime
func (a *App) EmitEvent(event string, data interface{}) {
	if a.app != nil {
		a.app.Event.Emit(event, data)
	}
}

// GetContext returns the application context
func (a *App) GetContext() context.Context {
	return nil // Context not used in v3
}

// Quit quits the application
func (a *App) Quit() {
	if a.app != nil {
		a.app.Quit()
	}
}

// GetVersion returns the application version
func (a *App) GetVersion() string {
	return "1.0.0"
}

// LogFrontend allows the frontend to send logs to the backend logger
func (a *App) LogFrontend(level string, message string) {
	switch level {
	case "DEBUG":
		a.logger.Debug(message)
	case "WARN":
		a.logger.Warn(message)
	case "ERROR":
		a.logger.Error(message)
	default:
		a.logger.Info(message)
	}
}

// ==================== Series Management ====================

// CreateSeries creates a new series at the specified path
func (a *App) CreateSeries(path, name, description string) error {
	a.logger.Debug(fmt.Sprintf("正在建立新系列: %s", path))

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create the database file
	if err := os.WriteFile(path, []byte{}, 0644); err != nil {
		return fmt.Errorf("failed to create database file: %w", err)
	}

	// Open the database
	database, err := db.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Auto migrate
	a.logger.Debug("建立資料表 (Debug)", "path", path)
	if err := db.AutoMigrate(database); err != nil {
		db.Close(database)
		return fmt.Errorf("failed to auto migrate: %w", err)
	}

	// Create the series
	series, err := db.CreateSeries(database, name, description)
	if err != nil {
		db.Close(database)
		return fmt.Errorf("failed to create series: %w", err)
	}

	// Store the database connection
	a.mu.Lock()
	a.db = database
	a.mu.Unlock()

	// Update config with last opened file
	a.cfg.LastOpenedFile = path
	if err := config.Save(config.GetConfigPath(), a.cfg); err != nil {
		a.logger.Warn(fmt.Sprintf("儲存設定失敗: %v", err))
	}

	// Add to recent files
	a.addToRecentFiles(path)

	a.logger.Info(fmt.Sprintf("成功建立系列: %s (ID: %d)", name, series.ID))
	return nil
}

// OpenSeries opens an existing series at the specified path
func (a *App) OpenSeries(path string) error {
	a.logger.Debug(fmt.Sprintf("準備開啟系列: %s", path))

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("series file not found: %s", path)
	}

	// Close existing database if open
	a.mu.Lock()
	if a.db != nil {
		db.Close(a.db)
		a.db = nil
	}
	a.mu.Unlock()

	// Open the database
	database, err := db.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// 自動遷移 DB schema，確保舊版資料庫自動補齊新欄位 (如 project_export_order)
	if err := db.AutoMigrate(database); err != nil {
		a.logger.Warn(fmt.Sprintf("開啟系列時自動遷移 schema 警告: %v", err))
	}

	// Store the database connection
	a.mu.Lock()
	a.db = database
	a.mu.Unlock()

	// Update config with last opened file
	a.cfg.LastOpenedFile = path
	if err := config.Save(config.GetConfigPath(), a.cfg); err != nil {
		a.logger.Warn(fmt.Sprintf("儲存設定失敗: %v", err))
	}

	// Add to recent files
	a.addToRecentFiles(path)

	a.logger.Info(fmt.Sprintf("成功開啟系列: %s", path))
	return nil
}

// CloseSeries closes the current series
func (a *App) CloseSeries() error {
	a.logger.Debug("正在關閉系列")

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.db != nil {
		if err := db.Close(a.db); err != nil {
			return fmt.Errorf("failed to close database: %w", err)
		}
		a.db = nil
	}

	a.logger.Info("系列已關閉")
	return nil
}

// GetSeriesInfo returns information about the currently open series
func (a *App) GetSeriesInfo() (*SeriesInfo, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.db == nil {
		return nil, fmt.Errorf("no series is currently open")
	}

	// Get series from database
	series, err := db.GetSeriesInfo(a.db)
	if err != nil {
		return nil, fmt.Errorf("failed to get series info: %w", err)
	}

	var projectOrder []string
	projectModelCounts := make(map[string]int)

	if strings.TrimSpace(series.ProjectExportOrder) != "" {
		// 優先解析包含 ProjectCode 與 ModelCount 的設定列表
		var settings []ProjectExportSetting
		if jsonErr := json.Unmarshal([]byte(series.ProjectExportOrder), &settings); jsonErr == nil && len(settings) > 0 {
			projectOrder = make([]string, 0, len(settings))
			for _, st := range settings {
				if st.ProjectCode != "" {
					projectOrder = append(projectOrder, st.ProjectCode)
					if st.ModelCount > 0 {
						projectModelCounts[st.ProjectCode] = st.ModelCount
					}
				}
			}
		} else {
			// 相容舊版僅包含 []string 的 JSON 資料
			_ = json.Unmarshal([]byte(series.ProjectExportOrder), &projectOrder)
		}
	}
	if projectOrder == nil {
		projectOrder = []string{}
	}

	return &SeriesInfo{
		ID:                 series.ID,
		Name:               series.Name,
		Description:        series.Description,
		Path:               a.cfg.LastOpenedFile,
		LastExportPath:     series.LastExportPath,
		ProjectExportOrder: projectOrder,
		ProjectModelCounts: projectModelCounts,
	}, nil
}

// SaveProjectExportOrder 即時儲存 BigMatrix 匯出對話框中的 Project 排序與 Model 數量設定紀錄至 Series 資料表
//
// 參數：
//   - settings：專案匯出設定列表（包含 ProjectCode 與 ModelCount）
//
// 回傳：
//   - error：若未開啟資料庫或 JSON 轉換/寫入失敗則回傳錯誤
func (a *App) SaveProjectExportOrder(settings []ProjectExportSetting) error {
	a.mu.RLock()
	dbConn := a.db
	a.mu.RUnlock()

	if dbConn == nil {
		a.logger.Error("[SaveProjectExportOrder] 失敗: 未開啟 Series 資料庫")
		return fmt.Errorf("no series is currently open")
	}

	orderJSON, err := json.Marshal(settings)
	if err != nil {
		a.logger.Error(fmt.Sprintf("[SaveProjectExportOrder] JSON 序列化 Project 排序與 Model 數量失敗: %v", err))
		return fmt.Errorf("failed to marshal project export settings: %w", err)
	}

	if err := db.UpdateProjectExportOrder(dbConn, string(orderJSON)); err != nil {
		a.logger.Warn(fmt.Sprintf("[SaveProjectExportOrder] 更新 project_export_order 失敗: %v", err))
		return fmt.Errorf("failed to update project export order in db: %w", err)
	}

	a.logger.Info(fmt.Sprintf("[SaveProjectExportOrder] 成功儲存 Project 匯出排序與 Model 數量紀錄: %+v", settings))
	return nil
}

// GetRecentSeries 傳回最近開啟的系列清單
// 1. 若檔案不存在：清理 TOML 設定檔紀錄，不顯示在 UI
// 2. 若檔案存在但無法開啟/讀取（格式不合或損毀）：保留 TOML 紀錄，但回傳標示 IsCorrupted = true 供 UI 顯示
func (a *App) GetRecentSeries() ([]*RecentFile, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	validPaths := make([]string, 0, len(a.cfg.RecentFiles.RecentFiles))
	recentFiles := make([]*RecentFile, 0, len(a.cfg.RecentFiles.RecentFiles))
	removedMissing := false

	for _, path := range a.cfg.RecentFiles.RecentFiles {
		// 1. 先檢查檔案是否存在
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			// 檔案不存在或為目錄：從 TOML 中清理，不保留此筆紀錄
			removedMissing = true
			continue
		}

		// 檔案存在：保留在 validPaths (保留 TOML 紀錄)
		validPaths = append(validPaths, path)

		// 2. 檔案存在，嘗試開啟資料庫讀取系列資訊
		seriesInfo, err := a.getSeriesInfoFromPath(path)
		if err != nil {
			// 檔案存在但無法成功讀取（損毀或格式不合）：保留 TOML 紀錄，標記 IsCorrupted = true
			baseName := filepath.Base(path)
			recentFiles = append(recentFiles, &RecentFile{
				Path:        path,
				Name:        fmt.Sprintf("%s (損毀或無法開啟)", baseName),
				LastOpened:  info.ModTime().Format(time.RFC3339),
				IsCorrupted: true,
			})
			continue
		}

		// 3. 正常讀取成功
		recentFiles = append(recentFiles, &RecentFile{
			Path:        path,
			Name:        seriesInfo.Name,
			LastOpened:  seriesInfo.LastOpened,
			IsCorrupted: false,
		})
	}

	// 若有不存在的檔案被刪除，更新記憶體設定檔並寫回 TOML
	if removedMissing {
		a.cfg.RecentFiles.RecentFiles = validPaths
		_ = config.Save(config.GetConfigPath(), a.cfg)
	}

	return recentFiles, nil
}

// ==================== Project Management ====================

// GetProjects returns all projects in the current series
func (a *App) GetProjects(seriesID int64) ([]*Project, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.db == nil {
		return nil, fmt.Errorf("no series is currently open")
	}

	projects, err := db.GetProjects(a.db, seriesID)
	if err != nil {
		return nil, fmt.Errorf("failed to get projects: %w", err)
	}

	result := make([]*Project, len(projects))
	for i := range projects {
		result[i] = &Project{
			ID:          projects[i].ID,
			SeriesID:    projects[i].SeriesID,
			Code:        projects[i].Code,
			Description: projects[i].Description,
			CreatedAt:   projects[i].CreatedAt.Format(time.RFC3339),
			UpdatedAt:   projects[i].UpdatedAt.Format(time.RFC3339),
		}
	}

	return result, nil
}

// GetProject returns a specific project by ID
func (a *App) GetProject(id int64) (*Project, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.db == nil {
		return nil, fmt.Errorf("no series is currently open")
	}

	project, err := db.GetProject(a.db, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return &Project{
		ID:          project.ID,
		SeriesID:    project.SeriesID,
		Code:        project.Code,
		Description: project.Description,
		CreatedAt:   project.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   project.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// ==================== Revision Management ====================

// GetRevisions returns all revisions for a project
func (a *App) GetRevisions(projectID int64) ([]*BomRevision, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.db == nil {
		return nil, fmt.Errorf("no series is currently open")
	}

	revisions, err := db.GetRevisions(a.db, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get revisions: %w", err)
	}

	result := make([]*BomRevision, len(revisions))
	for i, r := range revisions {
		result[i] = &BomRevision{
			ID:               r.ID,
			ProjectID:        r.ProjectID,
			Phase:            r.Phase,
			Version:          r.Version,
			Description:      r.Description,
			SchematicVersion: r.SchematicVersion,
			PCBVersion:       r.PCBVersion,
			PCAPN:            r.PCAPN,
			Date:             r.Date,
			Mode:             r.Mode,
			SourceFile:       r.SourceFile,
			ModelCount:       len(r.MatrixModels),
			CreatedAt:        r.CreatedAt.Format(time.RFC3339),
			UpdatedAt:        r.UpdatedAt.Format(time.RFC3339),
		}
	}

	return result, nil
}

// GetRevision returns a specific revision by ID
func (a *App) GetRevision(id int64) (*BomRevision, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.db == nil {
		return nil, fmt.Errorf("no series is currently open")
	}

	revision, err := db.GetRevision(a.db, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get revision: %w", err)
	}

	return &BomRevision{
		ID:               revision.ID,
		ProjectID:        revision.ProjectID,
		Phase:            revision.Phase,
		Version:          revision.Version,
		Description:      revision.Description,
		SchematicVersion: revision.SchematicVersion,
		PCBVersion:       revision.PCBVersion,
		PCAPN:            revision.PCAPN,
		Date:             revision.Date,
		Mode:             revision.Mode,
		SourceFile:       revision.SourceFile,
		ModelCount:       len(revision.MatrixModels),
		CreatedAt:        revision.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        revision.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// ==================== BOM View 查詢 ====================

// GetBOMView 查詢 BOM 視圖資料，是 View 系統的 Wails 綁定入口。
//
// View 系統為無狀態設計，前端顯示與後端匯出可同時以不同條件查詢。
//
// 參數：
//   - revisionIDs：要查詢的 BOM Revision ID 列表（1個=單一視圖，多個=整合視圖）
//   - viewType：視圖類型（ALL/SMD/PTH/BOTTOM/NI/PROTO/MP/CCL），空字串預設為 ALL
//
// 回傳：
//   - *view.ViewResult：查詢結果，包含聚合物料群組與 revision 元資料
//   - error：若資料庫連線未開啟或查詢失敗則回傳錯誤
func (a *App) GetBOMView(revisionIDs []int64, viewType string) (*view.ViewResult, error) {
	a.mu.RLock()
	dbConn := a.db
	a.mu.RUnlock()

	if dbConn == nil {
		return nil, fmt.Errorf("no series is currently open")
	}

	query := view.ViewQuery{
		RevisionIDs: revisionIDs,
		ViewType:    viewType,
	}

	svc := view.NewService(dbConn, a.logger)
	result, err := svc.Query(query)
	if err != nil {
		a.logger.Error(fmt.Sprintf("[GetBOMView] 視圖查詢失敗: %v", err))
		return nil, fmt.Errorf("view query failed: %w", err)
	}

	a.logger.Debug(fmt.Sprintf("[GetBOMView] 查詢完成: revisions=%d, parts=%d, viewType=%s",
		len(result.Revisions), len(result.PartGroups), viewType))

	return result, nil
}

// ==================== Import/Export ====================

// isMatrixFile 判斷給定的檔案路徑之檔案名稱是否包含 "matrix"（不區分大小寫）
//
// 參數：
//   - filePath: 檔案完整或相對路徑
//
// 回傳：
//   - 若檔案名稱（不含路徑目錄）包含 "matrix"（忽略大小寫）則回傳 true，否則回傳 false
func isMatrixFile(filePath string) bool {
	fileName := filepath.Base(filePath)
	return strings.Contains(strings.ToLower(fileName), "matrix")
}

// ImportExcel 依兩階段分組匯入 Excel 檔案至資料庫
//
// 執行邏輯：
// 1. 檔案分組：依檔名是否包含 "matrix" (不區分大小寫) 分為非 Matrix 組與 Matrix 組。
// 2. 兩階段執行：
//   - 第一階段：先背景執行非 Matrix 組 (EBOM/一般 BOM) 的匯入任務。
//   - 第二階段：等待第一階段所有群組任務全數結束後，Matrix 組自動接續執行開檔與匯入。
//
// 參數：
//   - filePaths: 欲匯入的 Excel 檔案路徑清單
//
// 回傳：
//   - []*ImportResult: 包含所有提交任務之 Task ID 與狀態資訊
//   - error: 若當前無開啟中的 Series 則回傳錯誤
func (a *App) ImportExcel(filePaths []string) ([]*ImportResult, error) {
	a.logger.Debug(fmt.Sprintf("準備匯入 %d 個 Excel 檔案", len(filePaths)))

	a.mu.RLock()
	dbConn := a.db
	a.mu.RUnlock()

	if dbConn == nil {
		return nil, fmt.Errorf("no series is currently open")
	}

	// 1. 檔案分組：將檔案依檔名是否含 "matrix" 分為兩組
	var nonMatrixPaths []string
	var matrixPaths []string
	for _, fp := range filePaths {
		if isMatrixFile(fp) {
			matrixPaths = append(matrixPaths, fp)
		} else {
			nonMatrixPaths = append(nonMatrixPaths, fp)
		}
	}

	results := make([]*ImportResult, 0, len(filePaths))

	// 第一階段同步機制：追蹤非 Matrix 任務群組之完成狀態
	var nonMatrixWg sync.WaitGroup
	var phase1Done chan struct{}

	if len(nonMatrixPaths) > 0 {
		nonMatrixWg.Add(len(nonMatrixPaths))
		phase1Done = make(chan struct{})

		// 在背景 goroutine 等待第一階段所有任務完成後關閉 channel 通知
		go func() {
			nonMatrixWg.Wait()
			close(phase1Done)
		}()
	}

	// 2. 第一階段：提交非 Matrix 檔案匯入任務
	for _, filePath := range nonMatrixPaths {
		taskID := uuid.New().String()
		taskName := fmt.Sprintf("Import: %s", filepath.Base(filePath))
		fp := filePath

		a.taskMgr.SubmitWithID(
			taskID,
			taskName,
			"Import",
			func(ctx context.Context, progress func(float64, string), taskLogger *logger.Logger) error {
				defer nonMatrixWg.Done()

				taskLogger.Info("開始執行基礎 BOM 匯入作業...", "name", taskName)
				progress(0.2, "正在開啟與辨識 Excel 檔案...")

				// 建立專屬單次開檔與解析的 Reader
				taskExcelReader := excel.NewReader(dbConn, taskLogger)
				taskExcelReader.SetProgressCallback(progress)

				// 執行單次開檔、前置驗證與資料匯入
				importResults, err := taskExcelReader.ImportExcel([]string{fp})
				if err != nil {
					taskLogger.Error("開啟或讀取 Excel 檔案重大失敗", "error", err.Error())
					return err
				}

				if len(importResults) > 0 {
					result := importResults[0]

					// 檢查是否有格式解析錯誤、Unknown 格式，或是非 Matrix 格式但 PartsCount 為 0 筆
					isWarning := len(result.Errors) > 0 || result.Format == types.FormatUnknown || (result.Format != types.FormatMatrix && result.PartsCount == 0)
					if isWarning {
						errMsg := fmt.Sprintf("匯入結果需要確認: 格式=%s, 成功筆數=%d", result.Format, result.PartsCount)
						if len(result.Errors) > 0 {
							errMsg += fmt.Sprintf(", 錯誤=%v", result.Errors)
						}
						taskLogger.Warn(errMsg)
						progress(0.9, errMsg)
						return task.NewWarningError(errors.New(errMsg))
					}

					msg := fmt.Sprintf("成功匯入 %d 筆料件", result.PartsCount)
					progress(0.9, msg)
					taskLogger.Info(msg)
				}

				progress(1.0, "匯入作業完成")
				return nil
			},
		)

		results = append(results, &ImportResult{
			FileName: filepath.Base(fp),
			Status:   "queued",
			Message:  "Task created",
			TaskID:   taskID,
		})
	}

	// 3. 第二階段：提交 Matrix 檔案匯入任務（若有第一階段，等待第一階段結束後再執行）
	for _, filePath := range matrixPaths {
		taskID := uuid.New().String()
		taskName := fmt.Sprintf("Import: %s", filepath.Base(filePath))
		fp := filePath

		a.taskMgr.SubmitWithID(
			taskID,
			taskName,
			"Import",
			func(ctx context.Context, progress func(float64, string), taskLogger *logger.Logger) error {
				// 若有第一階段非 Matrix 任務，先行等待其群組結束
				if phase1Done != nil {
					taskLogger.Info("等待第一階段非 Matrix BOM 檔案匯入完成...", "name", taskName)
					progress(0.05, "等待非 Matrix BOM 檔案匯入完成...")

					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-phase1Done:
						taskLogger.Info("第一階段匯入已完成，開始執行 Matrix BOM 匯入作業...", "name", taskName)
					}
				} else {
					taskLogger.Info("開始執行 Matrix BOM 匯入作業...", "name", taskName)
				}

				progress(0.2, "正在開啟與辨識 Excel 檔案...")

				// 建立專屬單次開檔與解析的 Reader
				taskExcelReader := excel.NewReader(dbConn, taskLogger)
				taskExcelReader.SetProgressCallback(progress)

				// 執行單次開檔、前置驗證與資料匯入
				importResults, err := taskExcelReader.ImportExcel([]string{fp})
				if err != nil {
					taskLogger.Error("開啟或讀取 Excel 檔案重大失敗", "error", err.Error())
					return err
				}

				if len(importResults) > 0 {
					result := importResults[0]

					// 檢查是否有格式解析錯誤、Unknown 格式
					isWarning := len(result.Errors) > 0 || result.Format == types.FormatUnknown
					if isWarning {
						errMsg := fmt.Sprintf("匯入結果需要確認: 格式=%s, 成功筆數=%d", result.Format, result.PartsCount)
						if len(result.Errors) > 0 {
							errMsg += fmt.Sprintf(", 錯誤=%v", result.Errors)
						}
						taskLogger.Warn(errMsg)
						progress(0.9, errMsg)
						return task.NewWarningError(errors.New(errMsg))
					}

					msg := "成功匯入 Matrix BOM 勾選資料與 Model 規格"
					progress(0.9, msg)
					taskLogger.Info(msg)
				}

				progress(1.0, "匯入作業完成")
				return nil
			},
		)

		results = append(results, &ImportResult{
			FileName: filepath.Base(fp),
			Status:   "queued",
			Message:  "Task created",
			TaskID:   taskID,
		})
	}

	return results, nil
}


// CopyMatrixSelections 以異步任務形式，手動將指定 source revision 的 Matrix Model 與 Selection 複製到 target revision。
//
// 此函數為手動版本複製的 Wails 綁定入口，會以 Task 形式提交至背景執行，
// 讓 UI 可透過 Task ID 追蹤執行進度與結果。
//
// 參數：
//   - sourceRevisionID: 來源版本 ID（Matrix 資料的來源）
//   - targetRevisionID: 目標版本 ID（Matrix 資料的目的地）
//
// 回傳：
//   - string: 任務 ID（taskID），可用於前端 Task 追蹤
//   - error: 若資料庫未開啟或 revision 不存在則回傳錯誤
func (a *App) CopyMatrixSelections(sourceRevisionID, targetRevisionID int64) (string, error) {
	a.mu.RLock()
	dbConn := a.db
	a.mu.RUnlock()

	if dbConn == nil {
		return "", fmt.Errorf("no series is currently open")
	}

	// 預先讀取 source 與 target revision 資訊，供 Task Log 使用
	var sourceRev, targetRev db.BomRevision
	if err := dbConn.First(&sourceRev, sourceRevisionID).Error; err != nil {
		return "", fmt.Errorf("找不到來源 Revision ID=%d: %w", sourceRevisionID, err)
	}
	if err := dbConn.First(&targetRev, targetRevisionID).Error; err != nil {
		return "", fmt.Errorf("找不到目標 Revision ID=%d: %w", targetRevisionID, err)
	}

	taskName := fmt.Sprintf("Copy Matrix: %s → %s", sourceRev.Version, targetRev.Version)

	taskID := a.taskMgr.Submit(
		taskName,
		"CopyMatrix",
		func(ctx context.Context, progress func(float64, string), taskLogger *logger.Logger) error {
			progress(0.1, fmt.Sprintf("開始複製 Matrix Selection（%s → %s）", sourceRev.Version, targetRev.Version))
			taskLogger.Info(fmt.Sprintf("[CopyMatrix] 開始執行 | 來源 RevisionID=%d (Version=%s) → 目標 RevisionID=%d (Version=%s)",
				sourceRevisionID, sourceRev.Version, targetRevisionID, targetRev.Version))

			progress(0.3, "正在複製 Matrix Model 與 Selection...")

			// 執行覆蓋式 Matrix 複製
			stats, err := db.ImportMatrixSelections(dbConn, sourceRevisionID, targetRevisionID, taskLogger)
			if err != nil {
				taskLogger.Error(fmt.Sprintf("[CopyMatrix] 複製失敗: %v", err))
				return fmt.Errorf("複製 Matrix Selection 失敗: %w", err)
			}

			// 輸出完整統計結果
			resultMsg := fmt.Sprintf(
				"Matrix 複製完成 | 有效 Model 數=%d, 複製 Selection 數=%d, 忽略主料數=%d, 忽略 2nd 替代料數=%d",
				stats.SourceModelCount, stats.CopiedSelectionsCount,
				stats.IgnoredMainParts, stats.IgnoredSecondParts,
			)
			taskLogger.Info(fmt.Sprintf("[CopyMatrix] %s", resultMsg))
			progress(1.0, resultMsg)

			return nil
		},
	)

	a.logger.Info(fmt.Sprintf("[CopyMatrixSelections] 已提交任務 %s (taskID=%s)", taskName, taskID))
	return taskID, nil
}

// ExportExcel exports data from the database to Excel files
func (a *App) ExportExcel(options *ExportOptions) ([]string, error) {
	formatStr := strings.TrimSpace(options.Format)
	var bomFormat types.BOMFormat
	if strings.EqualFold(formatStr, string(types.FormatBigMatrix)) {
		bomFormat = types.FormatBigMatrix
	} else if strings.EqualFold(formatStr, string(types.FormatMatrix)) {
		bomFormat = types.FormatMatrix
	} else {
		bomFormat = types.BOMFormat(formatStr)
	}

	a.logger.Info(fmt.Sprintf("[ExportExcel] 開始進行 Excel 匯出作業 (Format: %s, 選取 Revisions 數量: %d)", bomFormat, len(options.RevisionIDs)))
	a.logger.Debug(fmt.Sprintf("[ExportExcel] 匯出詳細參數: RevisionIDs=%v, ModelCountOverrides=%+v, OutputDir=%s", options.RevisionIDs, options.ModelCountOverrides, options.OutputDir))

	a.mu.RLock()
	dbConn := a.db
	a.mu.RUnlock()

	if dbConn == nil {
		a.logger.Error("[ExportExcel] 失敗: 未開啟 Series 資料庫")
		return nil, fmt.Errorf("no series is currently open")
	}

	// Update the series LastExportPath
	if options.OutputDir != "" {
		if err := dbConn.Model(&db.Series{}).Where("id = ?", 1).Update("last_export_path", options.OutputDir).Error; err != nil {
			a.logger.Warn(fmt.Sprintf("Failed to update last_export_path: %v", err))
		}
	}

	// 矩陣 (Matrix) 格式：每一個 selected BOM Revision 需各自匯出為獨立的 Matrix 檔案與任務
	if bomFormat == types.FormatMatrix {
		var taskIDs []string
		for _, revIDVal := range options.RevisionIDs {
			revIDStr := fmt.Sprintf("%d", revIDVal)

			// 嘗試讀取 Revision 基本資訊，建立具備專案/Phase/Version識別度的 Task 名稱
			var taskName string
			var revRecord db.BomRevision
			if err := dbConn.First(&revRecord, revIDVal).Error; err == nil {
				var projRecord db.Project
				projCode := "BOMIX"
				if err := dbConn.First(&projRecord, revRecord.ProjectID).Error; err == nil && projRecord.Code != "" {
					projCode = projRecord.Code
				}
				taskName = fmt.Sprintf("Export: Matrix (%s %s-%s)", projCode, revRecord.Phase, revRecord.Version)
			} else {
				taskName = fmt.Sprintf("Export: Matrix (ID: %d)", revIDVal)
			}

			// 多個 Revision 匯出時，若指定了 OutputPath 則取其目錄為 OutputDir
			outDir := options.OutputDir
			outPath := options.OutputPath
			if len(options.RevisionIDs) > 1 && outDir == "" && outPath != "" {
				outDir = filepath.Dir(outPath)
				outPath = ""
			}

			tID := a.taskMgr.Submit(
				taskName,
				"Export",
				func(ctx context.Context, progress func(float64, string), taskLogger *logger.Logger) error {
					progress(0.1, fmt.Sprintf("Preparing export data for Revision %d...", revIDVal))

					// 僅載入此單一 Revision 的 DB View 資料（Matrix 匯出：保留 Type 維度）
					revisions, parts, err := loadExportData(taskLogger, dbConn, []int64{revIDVal}, false)
					if err != nil {
						taskLogger.Warn(fmt.Sprintf("[ExportExcel] 載入 Revision %d 的 View 資料失敗/警告: %v", revIDVal, err))
					} else {
						taskLogger.Info(fmt.Sprintf("[ExportExcel] 成功載入 Revision %d 的 DB 資料: Parts=%d", revIDVal, len(parts)))
					}

					exportOptions := excel.ExportOptions{
						Format:              bomFormat,
						ProjectIDs:          options.ProjectIDs,
						RevisionIDs:         []string{revIDStr},
						Description:         options.Description,
						OutputPath:          outPath,
						OutputDir:           outDir,
						ModelCountOverrides: options.ModelCountOverrides,
						Revisions:           revisions,
						PartData:            parts,
					}

					excelWriter, err := excel.NewWriter(taskLogger)
					if err != nil {
						taskLogger.Error(fmt.Sprintf("[ExportExcel] 建立 Excel Writer 失敗: %v", err))
						return fmt.Errorf("failed to create excel writer: %w", err)
					}

					outputPaths, err := excelWriter.ExportExcel(exportOptions)
					if err != nil {
						if errors.Is(err, excel.ErrInvalidOutputPath) || strings.Contains(err.Error(), "invalid export output path") {
							if taskLogger != nil {
								taskLogger.Warn(fmt.Sprintf("[ExportExcel] 匯出路徑無效或無法寫入: %v", err))
							}
							return task.NewWarningError(fmt.Errorf("無效的匯出路徑: %w", err))
						}
						return fmt.Errorf("failed to export: %w", err)
					}

					if len(outputPaths) > 0 && taskLogger != nil {
						taskLogger.Info(fmt.Sprintf("[ExportExcel] 成功匯出 Matrix 檔案: %s", filepath.Base(outputPaths[0])))
					}

					progress(1.0, fmt.Sprintf("Exported %d files", len(outputPaths)))
					return nil
				},
			)
			taskIDs = append(taskIDs, tID)
		}
		return taskIDs, nil
	}

	// BigMatrix 格式：所有選取的 Revisions 橫向合併於單一 BigMatrix 檔案中
	revisionIDsStr := make([]string, len(options.RevisionIDs))
	for i, id := range options.RevisionIDs {
		revisionIDsStr[i] = fmt.Sprintf("%d", id)
	}

	// Export in a task
	taskID := uuid.New().String()
	taskID = a.taskMgr.Submit(
		fmt.Sprintf("Export: %s", options.Format),
		"Export",
		func(ctx context.Context, progress func(float64, string), taskLogger *logger.Logger) error {
			progress(0.1, "Preparing export data...")

			// Load revisions and part data from DB via View System using taskLogger
			// BigMatrix 格式：需進一步去除 Type 維度，依 (Supplier, SupplierPN) 合併物料
			revisions, parts, err := loadExportData(taskLogger, dbConn, options.RevisionIDs, true)
			if err != nil {
				taskLogger.Warn(fmt.Sprintf("[ExportExcel] 從資料庫載入 View 資料失敗/警告: %v", err))
			} else {
				taskLogger.Info(fmt.Sprintf("[ExportExcel] 成功透過 View 系統載入 DB 資料: Revisions=%d, Parts=%d", len(revisions), len(parts)))
			}

			exportOptions := excel.ExportOptions{
				Format:              bomFormat,
				ProjectIDs:          options.ProjectIDs,
				RevisionIDs:         revisionIDsStr,
				Description:         options.Description,
				OutputPath:          options.OutputPath,
				OutputDir:           options.OutputDir,
				ModelCountOverrides: options.ModelCountOverrides,
				Revisions:           revisions,
				PartData:            parts,
			}

			// Create Excel writer with taskLogger
			excelWriter, err := excel.NewWriter(taskLogger)
			if err != nil {
				taskLogger.Error(fmt.Sprintf("[ExportExcel] 建立 Excel Writer 失敗: %v", err))
				return fmt.Errorf("failed to create excel writer: %w", err)
			}

			// Export to Excel
			outputPaths, err := excelWriter.ExportExcel(exportOptions)
			if err != nil {
				if errors.Is(err, excel.ErrInvalidOutputPath) || strings.Contains(err.Error(), "invalid export output path") {
					if taskLogger != nil {
						taskLogger.Warn(fmt.Sprintf("[ExportExcel] 匯出路徑無效或無法寫入: %v", err))
					}
					return task.NewWarningError(fmt.Errorf("無效的匯出路徑: %w", err))
				}
				return fmt.Errorf("failed to export: %w", err)
			}

			progress(1.0, fmt.Sprintf("Exported %d files", len(outputPaths)))
			return nil
		},
	)

	return []string{taskID}, nil
}

// loadExportData 透過 View 系統從資料庫讀取匯出所需的 Revisions 與 Parts 資料。
//
// 此函數透過 View 系統的 Query() 取得資料，
// 依據 product-spec 8.1.6 規定使用 ViewCCL 視圖條件過濾 (ccl=Y, bom_status=I + P/M)。
//
// 匯出時使用「整合聯集」視圖（多 revision 時取聯集），
// ViewPartGroup 中的 SourceRevisionIDs 會被傳遞至 PartData，
// 供 BigMatrix Writer 判斷哪些儲存格需要填灰色底色。
//
// 參數：
//   - lg：Logger 實例
//   - dbConn：GORM 資料庫連線
//   - revisionIDs：要匯出的 BOM Revision ID 列表
//   - groupByMaterial：是否進一步去除 Type 維度，依 (Supplier, SupplierPN) 合併物料（BigMatrix 匯出為 true，Matrix 匯出為 false）
//
// 回傳：
//   - []excel.RevisionData：revision 元資料列表
//   - []excel.PartData：物料資料列表（包含 SourceRevisionIDs）
//   - error：若查詢失敗則回傳錯誤
func loadExportData(lg *logger.Logger, dbConn *gorm.DB, revisionIDs []int64, groupByMaterial bool) ([]excel.RevisionData, []excel.PartData, error) {
	if len(revisionIDs) == 0 {
		return nil, nil, nil
	}

	query := view.ViewQuery{
		RevisionIDs: revisionIDs,
		ViewType:    view.ViewCCL, // BigMatrix/Matrix 匯出依 product-spec 8.1.6 需使用 CCL 視圖過濾 (ccl=Y, bom_status!=X)
	}

	if lg != nil {
		lg.Info(fmt.Sprintf("[loadExportData] 建立 View 條件: RevisionIDs=%v, ViewType=%s, GroupByMaterial=%v",
			query.RevisionIDs, query.ViewType, groupByMaterial))
	}

	svc := view.NewService(dbConn, lg)
	viewResult, err := svc.Query(query)
	if err != nil {
		return nil, nil, fmt.Errorf("view query failed: %w", err)
	}

	// 若啟用 groupByMaterial (如 BigMatrix 匯出)，進一步去除 Type 維度，依 (Supplier, SupplierPN) 進行二階物料合併
	partGroups := viewResult.PartGroups
	if groupByMaterial {
		partGroups = view.MergePartGroupsByMaterial(viewResult.PartGroups)
		if lg != nil {
			lg.Info(fmt.Sprintf("[loadExportData] 套用二階物料合併 (去除 Type 維度): 原始群組數=%d, 合併後物料數=%d",
				len(viewResult.PartGroups), len(partGroups)))
		}
	}

	// 將 ViewRevision 轉換為 excel.RevisionData
	revDataList := make([]excel.RevisionData, 0, len(viewResult.Revisions))
	for _, vr := range viewResult.Revisions {
		revDataList = append(revDataList, excel.RevisionData{
			ID:               fmt.Sprintf("%d", vr.ID),
			ProjectCode:      vr.ProjectCode,
			Description:      vr.Description,
			SchematicVersion: vr.SchematicVersion,
			PCBVersion:       vr.PCBVersion,
			PCAPN:            vr.PCAPN,
			Phase:            vr.Phase,
			Version:          vr.Version,
			Date:             vr.Date,
			SourceFile:       vr.SourceFile,
			ModelNames:       vr.ModelNames,
			ModelQty:         vr.ModelQty,
			ModelQtyByOrder:  vr.ModelQtyByOrder,
		})
	}

	// 將 ViewPartGroup 轉換為 excel.PartData
	// 物料群組已由 View 系統聚合完畢（locations 已合併、qty 已計算）
	partDataList := make([]excel.PartData, 0, len(partGroups))
	for idx, pg := range partGroups {
		// 整合此群組的 Model 勾選狀態：
		// 1. 單一 Revision 相容: map[modelName]selectedPN 與 map[sortOrder]selectedPN
		// 2. BigMatrix 多 Revision 精確: map[revIDStr]map[sortOrder]selectedPN 與 map[revIDStr]map[modelName]selectedPN
		selections := make(map[string]string)
		selectionsByOrder := make(map[int]string)
		selectionsByRevAndOrder := make(map[string]map[int]string)
		selectionsByRevAndName := make(map[string]map[string]string)
		selectionsByMaterialByOrder := make(map[int]string)
		selectionsByRevAndMaterial := make(map[string]map[int]string)

		for _, sel := range pg.Selections {
			if sel.SelectedPN != "" {
				revIDStr := fmt.Sprintf("%d", sel.RevisionID)
				selectedMat := sel.SelectedMaterial
				if selectedMat == "" {
					if sel.SelectedSupplier != "" {
						selectedMat = fmt.Sprintf("%s|%s", strings.TrimSpace(sel.SelectedSupplier), strings.TrimSpace(sel.SelectedPN))
					} else {
						selectedMat = strings.TrimSpace(sel.SelectedPN)
					}
				}

				// 單一 Revision 相容
				if _, exists := selections[sel.ModelName]; !exists {
					selections[sel.ModelName] = sel.SelectedPN
				}
				if _, exists := selectionsByOrder[sel.SortOrder]; !exists {
					selectionsByOrder[sel.SortOrder] = sel.SelectedPN
				}
				if selectedMat != "" {
					if _, exists := selectionsByMaterialByOrder[sel.SortOrder]; !exists {
						selectionsByMaterialByOrder[sel.SortOrder] = selectedMat
					}
				}

				// 多 Revision 精確映射
				if selectionsByRevAndOrder[revIDStr] == nil {
					selectionsByRevAndOrder[revIDStr] = make(map[int]string)
				}
				selectionsByRevAndOrder[revIDStr][sel.SortOrder] = sel.SelectedPN

				if selectionsByRevAndName[revIDStr] == nil {
					selectionsByRevAndName[revIDStr] = make(map[string]string)
				}
				selectionsByRevAndName[revIDStr][sel.ModelName] = sel.SelectedPN

				if selectedMat != "" {
					if selectionsByRevAndMaterial[revIDStr] == nil {
						selectionsByRevAndMaterial[revIDStr] = make(map[int]string)
					}
					selectionsByRevAndMaterial[revIDStr][sel.SortOrder] = selectedMat
				}
			}
		}

		// 整合 SecondSources
		ssData := make([]excel.SecondSourceData, 0, len(pg.SecondSources))
		for _, ss := range pg.SecondSources {
			ssData = append(ssData, excel.SecondSourceData{
				HHPN:              ss.HHPN,
				Supplier:          ss.Supplier,
				SupplierPn:        ss.SupplierPN,
				Description:       ss.Description,
				Remark:            ss.Remark,
				SourceRevisionIDs: ss.SourceRevisionIDs,
				SelectionsByOrder: ss.SelectionsByOrder,
			})
		}

		partDataList = append(partDataList, excel.PartData{
			Item:                         fmt.Sprintf("%d", idx+1), // 流水號
			HHPN:                         pg.HHPN,
			Description:                  pg.Description,
			Supplier:                     pg.MainSupplier,
			SupplierPn:                   pg.MainSupplierPN,
			Qty:                          pg.Qty,
			Location:                     pg.Locations,
			Type:                         pg.Type,
			BOMStatus:                    pg.BOMStatus,
			CCL:                          pg.CCL,
			Remark:                       pg.Remark,
			SecondSources:                ssData,
			Selections:                   selections,
			SelectionsByOrder:            selectionsByOrder,
			SelectionsByRevAndOrder:      selectionsByRevAndOrder,
			SelectionsByRevAndName:       selectionsByRevAndName,
			SelectionsByMaterialByOrder:  selectionsByMaterialByOrder,
			SelectionsByRevAndMaterial:   selectionsByRevAndMaterial,
			MainSelectionsByOrder:        pg.MainSelectionsByOrder,
			SourceRevisionIDs:            pg.SourceRevisionIDs, // 傳遞來源歸屬，供 BigMatrix 填灰色
		})
	}

	return revDataList, partDataList, nil
}

// ==================== Task Management ====================

// ListTasks returns all tasks
func (a *App) ListTasks() ([]*Task, error) {
	tasks := a.taskMgr.ListTasks()
	result := make([]*Task, len(tasks))

	for i, t := range tasks {
		result[i] = &Task{
			ID:        t.ID,
			Name:      t.Name,
			Type:      t.Type,
			Status:    t.Status,
			Progress:  t.Progress,
			Message:   t.Message,
			Error:     t.Error,
			CreatedAt: t.CreatedAt.Format(time.RFC3339),
			UpdatedAt: t.CreatedAt.Format(time.RFC3339), // Use CreatedAt as UpdatedAt for now
		}
	}

	return result, nil
}

// GetTask returns a specific task by ID
func (a *App) GetTask(id string) (*Task, error) {
	t := a.taskMgr.GetStatus(id)
	if t == nil {
		return nil, fmt.Errorf("task not found: %s", id)
	}

	updatedAt := t.CreatedAt
	if t.CompletedAt != nil {
		updatedAt = *t.CompletedAt
	} else if t.StartedAt != nil {
		updatedAt = *t.StartedAt
	}

	return &Task{
		ID:        t.ID,
		Name:      t.Name,
		Type:      t.Type,
		Status:    t.Status,
		Progress:  t.Progress,
		Message:   t.Message,
		Error:     t.Error,
		CreatedAt: t.CreatedAt.Format(time.RFC3339),
		UpdatedAt: updatedAt.Format(time.RFC3339),
	}, nil
}

// CancelTask cancels a task
func (a *App) CancelTask(id string) error {
	return a.taskMgr.Cancel(id)
}

// ==================== Logs ====================

// GetLogs returns log entries
func (a *App) GetLogs(level string, limit int) ([]*LogEntry, error) {
	entries := a.logger.GetLogs(level, limit)

	result := make([]*LogEntry, len(entries))
	for i, e := range entries {
		result[i] = &LogEntry{
			Level:     e.Level,
			Message:   e.Message,
			Timestamp: e.Timestamp.Format(time.RFC3339),
			Attrs:     e.Attrs,
		}
	}

	return result, nil
}

// ClearLogs clears all log entries
func (a *App) ClearLogs() error {
	a.logger.ClearLogs()
	return nil
}

// ==================== Settings ====================

// GetSettings returns the current settings
func (a *App) GetSettings() (*Settings, error) {
	return &Settings{
		Theme:                    a.cfg.Theme,
		AutoOpenLastFile:         a.cfg.AutoOpenLastFile,
		LastOpenedFile:           a.cfg.LastOpenedFile,
		AutoImportPreviousMatrix: a.cfg.AutoImportPreviousMatrix,
		Import: &ImportSettings{
			ConfirmOverwrite:         a.cfg.Import.ConfirmOverwrite,
			AutoImportPreviousMatrix: a.cfg.Import.AutoImportPreviousMatrix,
		},
		Logger: &LoggerSettings{
			Level:      a.cfg.Logger.Level,
			MaxEntries: a.cfg.Logger.MaxEntries,
		},
		RecentFiles: &RecentFilesSettings{
			MaxRecentFiles: a.cfg.RecentFiles.MaxRecentFiles,
			RecentFiles:    a.cfg.RecentFiles.RecentFiles,
		},
	}, nil
}

// UpdateSettings updates the settings
func (a *App) UpdateSettings(settings *Settings) error {
	if settings == nil {
		return errors.New("settings payload cannot be nil")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Ensure config is initialized
	if a.cfg == nil {
		a.cfg = config.DefaultConfig
	}

	// Update config only with valid non-zero values
	if settings.Theme != "" {
		a.cfg.Theme = settings.Theme
	}
	if settings.Import != nil {
		a.cfg.Import.ConfirmOverwrite = settings.Import.ConfirmOverwrite
		a.cfg.Import.AutoImportPreviousMatrix = settings.Import.AutoImportPreviousMatrix
	}
	if settings.Logger != nil {
		if settings.Logger.Level != "" {
			a.cfg.Logger.Level = settings.Logger.Level
		}
		if settings.Logger.MaxEntries > 0 {
			a.cfg.Logger.MaxEntries = settings.Logger.MaxEntries
		}
	}
	if settings.RecentFiles != nil {
		if settings.RecentFiles.MaxRecentFiles > 0 {
			a.cfg.RecentFiles.MaxRecentFiles = settings.RecentFiles.MaxRecentFiles
		}
		if settings.RecentFiles.RecentFiles != nil {
			a.cfg.RecentFiles.RecentFiles = settings.RecentFiles.RecentFiles
		}
	}
	a.cfg.AutoOpenLastFile = settings.AutoOpenLastFile
	a.cfg.LastOpenedFile = settings.LastOpenedFile
	a.cfg.AutoImportPreviousMatrix = settings.AutoImportPreviousMatrix

	// Save config
	if err := config.Save(config.GetConfigPath(), a.cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	a.logger.Info("設定已更新")
	return nil
}

// ==================== Helpers ====================

// addToRecentFiles adds a file to the recent files list
func (a *App) addToRecentFiles(path string) {
	// Remove if already exists
	index := slices.Index(a.cfg.RecentFiles.RecentFiles, path)
	if index != -1 {
		a.cfg.RecentFiles.RecentFiles = slices.Delete(a.cfg.RecentFiles.RecentFiles, index, index+1)
	}

	// Add to front
	a.cfg.RecentFiles.RecentFiles = append([]string{path}, a.cfg.RecentFiles.RecentFiles...)

	// Limit to max recent files
	maxFiles := a.cfg.RecentFiles.MaxRecentFiles
	if len(a.cfg.RecentFiles.RecentFiles) > maxFiles {
		a.cfg.RecentFiles.RecentFiles = a.cfg.RecentFiles.RecentFiles[:maxFiles]
	}

	// Save config
	_ = config.Save(config.GetConfigPath(), a.cfg)
}

// getSeriesInfoFromPath 從檔案路徑取得系列資訊
// 會先檢查檔案是否存在且非目錄，確認存在後才嘗試開啟資料庫進行讀取，避免 SQLite 自動建立空檔案
func (a *App) getSeriesInfoFromPath(path string) (*SeriesInfoWithTime, error) {
	// 1. 先檢查檔案是否存在且非目錄
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("路徑為目錄而非檔案: %s", path)
	}

	// 2. 檔案存在時，才嘗試開啟資料庫讀取系列資訊
	database, err := db.Open(path)
	if err != nil {
		return nil, err
	}
	defer db.Close(database)

	series, err := db.GetSeriesInfo(database)
	if err != nil {
		return nil, err
	}

	return &SeriesInfoWithTime{
		Name:       series.Name,
		LastOpened: info.ModTime().Format(time.RFC3339),
	}, nil
}
