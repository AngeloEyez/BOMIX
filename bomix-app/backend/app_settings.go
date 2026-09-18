package backend

import (
	"errors"
	"fmt"
	"strings"

	"bomix-app/backend/config"
)

// ==================== 設定管理 (Settings & Config SSOT) ====================

// configToSettings 將後端 Config 結構映射轉換為前端 Settings DTO
//
// 參數：
//   - cfg: 後端設定物件
//   - maskKey: 是否對敏感 API 金鑰進行末 4 碼遮罩
//
// 回傳：
//   - *Settings: 前端設定 DTO
func configToSettings(cfg *config.Config, maskKey bool) *Settings {
	if cfg == nil {
		cfg = config.DefaultConfig
	}
	apiKey := cfg.AI.APIKey
	if maskKey && apiKey != "" {
		apiKey = maskAPIKey(apiKey)
	}

	timeout := cfg.AI.Timeout
	if timeout <= 0 {
		timeout = config.DefaultConfig.AI.Timeout
	}
	maxTokens := cfg.AI.MaxTokens
	if maxTokens <= 0 {
		maxTokens = config.DefaultConfig.AI.MaxTokens
	}
	maxIterations := cfg.AI.MaxIterations
	if maxIterations <= 0 {
		maxIterations = config.DefaultConfig.AI.MaxIterations
	}
	temperature := cfg.AI.Temperature
	if temperature < 0 || temperature > 2.0 {
		temperature = config.DefaultConfig.AI.Temperature
	}

	return &Settings{
		Theme:                    cfg.Theme,
		AutoOpenLastFile:         cfg.AutoOpenLastFile,
		LastOpenedFile:           cfg.LastOpenedFile,
		AutoImportPreviousMatrix: cfg.AutoImportPreviousMatrix,
		Import: &ImportSettings{
			ConfirmOverwrite:         cfg.Import.ConfirmOverwrite,
			AutoImportPreviousMatrix: cfg.Import.AutoImportPreviousMatrix,
		},
		Logger: &LoggerSettings{
			Level:      cfg.Logger.Level,
			MaxEntries: cfg.Logger.MaxEntries,
		},
		RecentFiles: &RecentFilesSettings{
			MaxRecentFiles: cfg.RecentFiles.MaxRecentFiles,
			RecentFiles:    cfg.RecentFiles.RecentFiles,
		},
		AI: &AISettings{
			Enabled:       cfg.AI.Enabled,
			BaseURL:       cfg.AI.BaseURL,
			APIKey:        apiKey,
			Model:         cfg.AI.Model,
			Temperature:   temperature,
			MaxTokens:     maxTokens,
			MaxIterations: maxIterations,
			Timeout:       timeout,
			Language:      cfg.AI.Language,
		},
	}
}

// GetSettings 取得當前應用程式設定（API 金鑰將會被遮罩以確保安全）
//
// 回傳：
//   - *Settings: 當前設定 DTO
//   - error: 固定回傳 nil
func (a *App) GetSettings() (*Settings, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return configToSettings(a.cfg, true), nil
}

// GetDefaultSettings 取得應用程式預設配置作為 Settings DTO（單一真理來源 Single Source of Truth）
//
// 回傳：
//   - *Settings: 系統預設配置 DTO
//   - error: 固定回傳 nil
func (a *App) GetDefaultSettings() (*Settings, error) {
	return configToSettings(config.DefaultConfig, false), nil
}

// UpdateSettings 接收前端傳入之 Settings DTO，校驗有效值後更新並持久化至設定檔
//
// 參數：
//   - settings: 欲更新的設定物件
//
// 回傳：
//   - error: 若設定為空或持久化存檔失敗則回傳錯誤
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
	if settings.AI != nil {
		a.cfg.AI.Enabled = settings.AI.Enabled
		if settings.AI.BaseURL != "" {
			a.cfg.AI.BaseURL = settings.AI.BaseURL
		}
		if settings.AI.APIKey != "" && !strings.HasPrefix(settings.AI.APIKey, "****") {
			a.cfg.AI.APIKey = settings.AI.APIKey
		}
		if settings.AI.Model != "" {
			a.cfg.AI.Model = settings.AI.Model
		}
		if settings.AI.Temperature >= 0 && settings.AI.Temperature <= 2.0 {
			a.cfg.AI.Temperature = settings.AI.Temperature
		} else {
			a.cfg.AI.Temperature = config.DefaultConfig.AI.Temperature
		}
		if settings.AI.MaxTokens > 0 {
			a.cfg.AI.MaxTokens = settings.AI.MaxTokens
		} else {
			a.cfg.AI.MaxTokens = config.DefaultConfig.AI.MaxTokens
		}
		if settings.AI.MaxIterations > 0 {
			a.cfg.AI.MaxIterations = settings.AI.MaxIterations
		} else {
			a.cfg.AI.MaxIterations = config.DefaultConfig.AI.MaxIterations
		}
		if settings.AI.Timeout > 0 {
			a.cfg.AI.Timeout = settings.AI.Timeout
		} else {
			a.cfg.AI.Timeout = config.DefaultConfig.AI.Timeout
		}
		if settings.AI.Language != "" {
			a.cfg.AI.Language = settings.AI.Language
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

// ==================== 私有輔助函式 (Internal Helpers) ====================

// maskAPIKey 遮罩 API 金鑰以保護隱私，只顯示末 4 碼
//
// 參數：
//   - key: 原始 API 金鑰字串
//
// 回傳：
//   - 遮罩後的金鑰字串（如 "****1234"）
func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 4 {
		return "****"
	}
	return "****" + key[len(key)-4:]
}
