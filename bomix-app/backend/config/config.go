package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Load reads the configuration from a TOML file and merges with defaults
// If the file doesn't exist, returns DefaultConfig without creating the file
func Load(path string) (*Config, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return DefaultConfig, nil
	}

	// Read the TOML file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Decode into Config struct with metadata to track which fields were set
	var cfg Config
	md, err := toml.Decode(string(data), &cfg)
	if err != nil {
		return nil, err
	}

	// Merge with defaults for any fields that were not set
	mergeWithDefaults(&cfg, md)

	return &cfg, nil
}

// Save writes the configuration to a TOML file
// Only writes fields that differ from DefaultConfig (delta save)
func Save(path string, cfg *Config) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Create the file
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Encode only non-default values using a map
	deltaMap := createDeltaMap(cfg)

	encoder := toml.NewEncoder(f)
	return encoder.Encode(deltaMap)
}

// GetConfigPath returns the default config file path
// Returns %APPDATA%/BOMIX/config.toml on Windows
func GetConfigPath() string {
	// Use xdg for cross-platform config path
	// On Windows: %APPDATA%\BOMIX\config.toml
	// On Linux: ~/.config/BOMIX/config.toml
	// On macOS: ~/Library/Application Support/BOMIX/config.toml
	configDir := filepath.Join(os.Getenv("APPDATA"), "BOMIX")
	if os.Getenv("APPDATA") == "" {
		// Fallback for non-Windows systems
		homeDir, _ := os.UserHomeDir()
		configDir = filepath.Join(homeDir, ".config", "BOMIX")
	}
	return filepath.Join(configDir, "config.toml")
}

// mergeWithDefaults fills in missing fields with default values
// Uses metadata to determine which fields were actually set in the config file
func mergeWithDefaults(cfg *Config, md toml.MetaData) {
	// Theme - only set if not in metadata
	if !md.IsDefined("theme") {
		cfg.Theme = DefaultConfig.Theme
	}

	// Import settings
	if !md.IsDefined("import", "confirm_overwrite") {
		cfg.Import.ConfirmOverwrite = DefaultConfig.Import.ConfirmOverwrite
	}
	if !md.IsDefined("import", "auto_import_previous_matrix") {
		cfg.Import.AutoImportPreviousMatrix = DefaultConfig.Import.AutoImportPreviousMatrix
	}

	// Logger settings
	if !md.IsDefined("logger", "level") {
		cfg.Logger.Level = DefaultConfig.Logger.Level
	}
	if !md.IsDefined("logger", "max_entries") {
		cfg.Logger.MaxEntries = DefaultConfig.Logger.MaxEntries
	}

	// Recent files settings
	if !md.IsDefined("recent_files", "max_recent_files") {
		cfg.RecentFiles.MaxRecentFiles = DefaultConfig.RecentFiles.MaxRecentFiles
	}

	// Top-level settings
	if !md.IsDefined("auto_open_last_file") {
		cfg.AutoOpenLastFile = DefaultConfig.AutoOpenLastFile
	}
	if !md.IsDefined("last_opened_file") {
		cfg.LastOpenedFile = DefaultConfig.LastOpenedFile
	}
	if !md.IsDefined("auto_import_previous_matrix") {
		cfg.AutoImportPreviousMatrix = DefaultConfig.AutoImportPreviousMatrix
	}
}

// createDeltaMap creates a map with only non-default values for TOML encoding
func createDeltaMap(cfg *Config) map[string]interface{} {
	delta := make(map[string]interface{})

	// Only include fields that differ from defaults
	if cfg.Theme != DefaultConfig.Theme {
		delta["theme"] = cfg.Theme
	}

	// Import settings - only add if there are non-default values
	importChanged := false
	importMap := make(map[string]interface{})
	if cfg.Import.ConfirmOverwrite != DefaultConfig.Import.ConfirmOverwrite {
		importMap["confirm_overwrite"] = cfg.Import.ConfirmOverwrite
		importChanged = true
	}
	if cfg.Import.AutoImportPreviousMatrix != DefaultConfig.Import.AutoImportPreviousMatrix {
		importMap["auto_import_previous_matrix"] = cfg.Import.AutoImportPreviousMatrix
		importChanged = true
	}
	if importChanged {
		delta["import"] = importMap
	}

	// Logger settings - only add if there are non-default values
	loggerChanged := false
	loggerMap := make(map[string]interface{})
	if cfg.Logger.Level != DefaultConfig.Logger.Level {
		loggerMap["level"] = cfg.Logger.Level
		loggerChanged = true
	}
	if cfg.Logger.MaxEntries != DefaultConfig.Logger.MaxEntries {
		loggerMap["max_entries"] = cfg.Logger.MaxEntries
		loggerChanged = true
	}
	if loggerChanged {
		delta["logger"] = loggerMap
	}

	// Recent files settings - only add if there are non-default values
	recentFilesChanged := false
	recentFilesMap := make(map[string]interface{})
	if cfg.RecentFiles.MaxRecentFiles != DefaultConfig.RecentFiles.MaxRecentFiles {
		recentFilesMap["max_recent_files"] = cfg.RecentFiles.MaxRecentFiles
		recentFilesChanged = true
	}
	if len(cfg.RecentFiles.RecentFiles) > 0 {
		recentFilesMap["recent_files"] = cfg.RecentFiles.RecentFiles
		recentFilesChanged = true
	}
	if recentFilesChanged {
		delta["recent_files"] = recentFilesMap
	}

	// Top-level settings
	if cfg.AutoOpenLastFile != DefaultConfig.AutoOpenLastFile {
		delta["auto_open_last_file"] = cfg.AutoOpenLastFile
	}
	if cfg.LastOpenedFile != DefaultConfig.LastOpenedFile {
		delta["last_opened_file"] = cfg.LastOpenedFile
	}
	if cfg.AutoImportPreviousMatrix != DefaultConfig.AutoImportPreviousMatrix {
		delta["auto_import_previous_matrix"] = cfg.AutoImportPreviousMatrix
	}

	return delta
}
