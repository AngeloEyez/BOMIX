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
}

// DefaultConfig returns the default configuration with all preset values
var DefaultConfig = &Config{
	Theme: "light",
	Import: ImportConfig{
		ConfirmOverwrite:       true,
		AutoImportPreviousMatrix: false,
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
	AutoImportPreviousMatrix: false,
}
