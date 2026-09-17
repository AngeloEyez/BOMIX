package config

// ImportConfig holds configuration for import operations
type ImportConfig struct {
	ConfirmOverwrite       bool `toml:"confirm_overwrite"`
	AutoImportPreviousMatrix bool `toml:"auto_import_previous_matrix"`
}

// LoggerConfig holds configuration for the logger
type LoggerConfig struct {
	Level      string `toml:"level"`
	MaxEntries int    `toml:"max_entries"`
}

// RecentFilesConfig holds configuration for recently opened files
type RecentFilesConfig struct {
	MaxRecentFiles int      `toml:"max_recent_files"`
	RecentFiles    []string `toml:"recent_files"`
}

// Config represents the full application configuration
type Config struct {
	// General settings
	Theme string `toml:"theme"`

	// Import settings
	Import ImportConfig `toml:"import"`

	// Logger settings
	Logger LoggerConfig `toml:"logger"`

	// Recent files settings
	RecentFiles RecentFilesConfig `toml:"recent_files"`

	// Auto-open last file setting
	AutoOpenLastFile bool `toml:"auto_open_last_file"`

	// Last opened file path
	LastOpenedFile string `toml:"last_opened_file"`

	// Auto import previous matrix setting
	AutoImportPreviousMatrix bool `toml:"auto_import_previous_matrix"`

	// AI assistant settings
	AI AIConfig `toml:"ai"`
}

// AIConfig 保存 AI 助手相關設定
type AIConfig struct {
	Enabled     bool    `toml:"enabled"`     // 是否啟用 AI 功能
	BaseURL     string  `toml:"base_url"`    // OpenAI 相容端點 (預設 https://api.openai.com/v1)
	APIKey      string  `toml:"api_key"`     // API 金鑰
	Model       string  `toml:"model"`       // 模型名稱 (預設 gpt-4o-mini)
	Temperature float64 `toml:"temperature"` // 溫度 (預設 0.1)
	MaxTokens   int     `toml:"max_tokens"`  // 單次回應 Token 上限 (預設 4096)
	Timeout     int     `toml:"timeout"`     // HTTP 請求超時秒數 (預設 60)
	Language    string  `toml:"language"`    // 回應語言偏好 (預設 "zh-TW")
}

// DefaultConfig returns the default configuration with all preset values
var DefaultConfig = &Config{
	Theme: "light",
	Import: ImportConfig{
		ConfirmOverwrite:       true,
		AutoImportPreviousMatrix: true,
	},
	Logger: LoggerConfig{
		Level:      "info",
		MaxEntries: 500,
	},
	RecentFiles: RecentFilesConfig{
		MaxRecentFiles: 10,
		RecentFiles:    []string{},
	},
	AutoOpenLastFile:       false,
	LastOpenedFile:         "",
	AutoImportPreviousMatrix: true,
	AI: AIConfig{
		Enabled:     false,
		BaseURL:     "https://api.openai.com/v1",
		APIKey:      "",
		Model:       "gpt-4o-mini",
		Temperature: 0.1,
		MaxTokens:   4096,
		Timeout:     60,
		Language:    "zh-TW",
	},
}

