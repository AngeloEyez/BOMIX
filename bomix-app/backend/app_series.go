package backend

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"bomix-app/backend/config"
	"bomix-app/backend/db"
)

// ==================== 系列管理 (Series Management) ====================

// CreateSeries 在指定路徑建立新的 BOMIX 系列資料庫檔案並初始化 Schema
//
// 參數：
//   - path: 系列資料庫目標檔案路徑（如 .bomx）
//   - name: 系列名稱
//   - description: 系列描述
//
// 回傳：
//   - error: 若路徑建立失敗、資料庫開啟失敗或 Schema 遷移失敗則回傳錯誤
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

// OpenSeries 開啟指定路徑的既有系列資料庫檔案
//
// 參數：
//   - path: 系列資料庫檔案路徑
//
// 回傳：
//   - error: 若檔案不存在、開啟失敗或自動遷移失敗則回傳錯誤
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

// CloseSeries 關閉目前已開啟的系列資料庫，並中斷所有執行中的 AI 任務
//
// 回傳：
//   - error: 若關閉資料庫失敗則回傳錯誤
func (a *App) CloseSeries() error {
	a.logger.Debug("正在關閉系列")

	// 若有進行中的 AI 生成與工具調用，一併中斷以防止資料庫關閉時併發存取報錯
	_ = a.AIChatStop()

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

// GetSeriesInfo 取得目前開啟中的系列基本資訊與匯出排序設定
//
// 回傳：
//   - *SeriesInfo: 系列詳細資訊 DTO
//   - error: 若當前無開啟中的系列資料庫則回傳錯誤
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
//   - settings: 專案匯出設定列表（包含 ProjectCode 與 ModelCount）
//
// 回傳：
//   - error: 若未開啟資料庫或 JSON 轉換/寫入失敗則回傳錯誤
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
//
// 回傳：
//   - []*RecentFile: 最近開啟檔案清單
//   - error: 若讀取失敗則回傳錯誤
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

// ==================== 私有輔助函式 (Internal Helpers) ====================

// addToRecentFiles 將指定檔案路徑新增至最近開啟清單前端並持久化至設定檔
//
// 參數：
//   - path: 檔案絕對路徑
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

// getSeriesInfoFromPath 從檔案路徑讀取系列名稱與最後修改時間
// 先檢查檔案是否存在且非目錄，確認存在後才嘗試開啟資料庫，避免 SQLite 自動建立空檔案
//
// 參數：
//   - path: 資料庫檔案路徑
//
// 回傳：
//   - *SeriesInfoWithTime: 系列資訊與修改時間
//   - error: 若路徑為目錄或資料庫讀取失敗則回傳錯誤
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
