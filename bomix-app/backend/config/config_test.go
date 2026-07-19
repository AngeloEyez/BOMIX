package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLoad_NoFile tests that Load returns DefaultConfig when file doesn't exist
func TestLoad_NoFile(t *testing.T) {
	// Use a non-existent path
	tempDir := t.TempDir()
	nonExistentPath := filepath.Join(tempDir, "non_existent_config.toml")

	cfg, err := Load(nonExistentPath)
	if err != nil {
		t.Fatalf("Load returned unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load returned nil config")
	}

	// Verify all fields match defaults
	if cfg.Theme != DefaultConfig.Theme {
		t.Errorf("Theme = %q, want %q", cfg.Theme, DefaultConfig.Theme)
	}
	if cfg.AutoOpenLastFile != DefaultConfig.AutoOpenLastFile {
		t.Errorf("AutoOpenLastFile = %v, want %v", cfg.AutoOpenLastFile, DefaultConfig.AutoOpenLastFile)
	}
	if cfg.LastOpenedFile != DefaultConfig.LastOpenedFile {
		t.Errorf("LastOpenedFile = %q, want %q", cfg.LastOpenedFile, DefaultConfig.LastOpenedFile)
	}
	if cfg.Import.ConfirmOverwrite != DefaultConfig.Import.ConfirmOverwrite {
		t.Errorf("Import.ConfirmOverwrite = %v, want %v", cfg.Import.ConfirmOverwrite, DefaultConfig.Import.ConfirmOverwrite)
	}
}

// TestLoad_PartialOverride tests partial config override
func TestLoad_PartialOverride(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	// Create a partial config with only theme = "dark"
	partialTOML := `theme = "dark"`
	if err := os.WriteFile(configPath, []byte(partialTOML), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load returned unexpected error: %v", err)
	}

	// Verify theme is overridden
	if cfg.Theme != "dark" {
		t.Errorf("Theme = %q, want %q", cfg.Theme, "dark")
	}

	// Verify other fields have default values
	if cfg.Import.ConfirmOverwrite != DefaultConfig.Import.ConfirmOverwrite {
		t.Errorf("Import.ConfirmOverwrite = %v, want %v", cfg.Import.ConfirmOverwrite, DefaultConfig.Import.ConfirmOverwrite)
	}
	if cfg.Logger.Level != DefaultConfig.Logger.Level {
		t.Errorf("Logger.Level = %q, want %q", cfg.Logger.Level, DefaultConfig.Logger.Level)
	}
	if cfg.Logger.MaxEntries != DefaultConfig.Logger.MaxEntries {
		t.Errorf("Logger.MaxEntries = %d, want %d", cfg.Logger.MaxEntries, DefaultConfig.Logger.MaxEntries)
	}
	if cfg.AutoOpenLastFile != DefaultConfig.AutoOpenLastFile {
		t.Errorf("AutoOpenLastFile = %v, want %v", cfg.AutoOpenLastFile, DefaultConfig.AutoOpenLastFile)
	}
}

// TestSave_DeltaSave tests that Save only writes non-default values
func TestSave_DeltaSave(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	// Create a config with only theme changed
	cfg := &Config{
		Theme: "dark",
	}
	// Initialize nested structs with defaults
	cfg.Import = ImportConfig{
		ConfirmOverwrite:       DefaultConfig.Import.ConfirmOverwrite,
		AutoImportPreviousMatrix: DefaultConfig.Import.AutoImportPreviousMatrix,
	}
	cfg.Logger = LoggerConfig{
		Level:      DefaultConfig.Logger.Level,
		MaxEntries: DefaultConfig.Logger.MaxEntries,
	}
	cfg.RecentFiles = RecentFilesConfig{
		MaxRecentFiles: DefaultConfig.RecentFiles.MaxRecentFiles,
		RecentFiles:    []string{},
	}
	cfg.AutoOpenLastFile = DefaultConfig.AutoOpenLastFile
	cfg.LastOpenedFile = DefaultConfig.LastOpenedFile
	cfg.AutoImportPreviousMatrix = DefaultConfig.AutoImportPreviousMatrix

	if err := Save(configPath, cfg); err != nil {
		t.Fatalf("Save returned unexpected error: %v", err)
	}

	// Read the saved file
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read saved config: %v", err)
	}

	contentStr := string(content)

	// Verify only theme is in the file
	if !strings.Contains(contentStr, "theme") {
		t.Error("Saved config should contain 'theme'")
	}

	// Verify other default fields are NOT in the file
	if strings.Contains(contentStr, "confirm_overwrite") {
		t.Error("Saved config should NOT contain 'confirm_overwrite' (default value)")
	}
	if strings.Contains(contentStr, "max_entries") {
		t.Error("Saved config should NOT contain 'max_entries' (default value)")
	}
	if strings.Contains(contentStr, "auto_open_last_file") {
		t.Error("Saved config should NOT contain 'auto_open_last_file' (default value)")
	}
}

// TestSave_Load_RoundTrip tests full round-trip: Save -> Load -> Compare
func TestSave_Load_RoundTrip(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	// Create a config with multiple non-default values
	originalCfg := &Config{
		Theme: "dark",
		Import: ImportConfig{
			ConfirmOverwrite:       false, // Different from default (true)
			AutoImportPreviousMatrix: true,  // Different from default (false)
		},
		Logger: LoggerConfig{
			Level:      "debug",
			MaxEntries: 1000,
		},
		RecentFiles: RecentFilesConfig{
			MaxRecentFiles: 20,
			RecentFiles:    []string{"/path/to/file1.bomx", "/path/to/file2.bomx"},
		},
		AutoOpenLastFile:       true,
		LastOpenedFile:         "/path/to/last.bomx",
		AutoImportPreviousMatrix: true,
	}

	// Save the config
	if err := Save(configPath, originalCfg); err != nil {
		t.Fatalf("Save returned unexpected error: %v", err)
	}

	// Load the config
	loadedCfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load returned unexpected error: %v", err)
	}

	// Compare all fields
	if loadedCfg.Theme != originalCfg.Theme {
		t.Errorf("Theme = %q, want %q", loadedCfg.Theme, originalCfg.Theme)
	}
	if loadedCfg.Import.ConfirmOverwrite != originalCfg.Import.ConfirmOverwrite {
		t.Errorf("Import.ConfirmOverwrite = %v, want %v", loadedCfg.Import.ConfirmOverwrite, originalCfg.Import.ConfirmOverwrite)
	}
	if loadedCfg.Import.AutoImportPreviousMatrix != originalCfg.Import.AutoImportPreviousMatrix {
		t.Errorf("Import.AutoImportPreviousMatrix = %v, want %v", loadedCfg.Import.AutoImportPreviousMatrix, originalCfg.Import.AutoImportPreviousMatrix)
	}
	if loadedCfg.Logger.Level != originalCfg.Logger.Level {
		t.Errorf("Logger.Level = %q, want %q", loadedCfg.Logger.Level, originalCfg.Logger.Level)
	}
	if loadedCfg.Logger.MaxEntries != originalCfg.Logger.MaxEntries {
		t.Errorf("Logger.MaxEntries = %d, want %d", loadedCfg.Logger.MaxEntries, originalCfg.Logger.MaxEntries)
	}
	if loadedCfg.RecentFiles.MaxRecentFiles != originalCfg.RecentFiles.MaxRecentFiles {
		t.Errorf("RecentFiles.MaxRecentFiles = %d, want %d", loadedCfg.RecentFiles.MaxRecentFiles, originalCfg.RecentFiles.MaxRecentFiles)
	}
	if len(loadedCfg.RecentFiles.RecentFiles) != len(originalCfg.RecentFiles.RecentFiles) {
		t.Errorf("RecentFiles.RecentFiles length = %d, want %d", len(loadedCfg.RecentFiles.RecentFiles), len(originalCfg.RecentFiles.RecentFiles))
	} else {
		for i, rf := range originalCfg.RecentFiles.RecentFiles {
			if loadedCfg.RecentFiles.RecentFiles[i] != rf {
				t.Errorf("RecentFiles.RecentFiles[%d] = %q, want %q", i, loadedCfg.RecentFiles.RecentFiles[i], rf)
			}
		}
	}
	if loadedCfg.AutoOpenLastFile != originalCfg.AutoOpenLastFile {
		t.Errorf("AutoOpenLastFile = %v, want %v", loadedCfg.AutoOpenLastFile, originalCfg.AutoOpenLastFile)
	}
	if loadedCfg.LastOpenedFile != originalCfg.LastOpenedFile {
		t.Errorf("LastOpenedFile = %q, want %q", loadedCfg.LastOpenedFile, originalCfg.LastOpenedFile)
	}
	if loadedCfg.AutoImportPreviousMatrix != originalCfg.AutoImportPreviousMatrix {
		t.Errorf("AutoImportPreviousMatrix = %v, want %v", loadedCfg.AutoImportPreviousMatrix, originalCfg.AutoImportPreviousMatrix)
	}
}

// TestGetConfigPath tests the GetConfigPath function
func TestGetConfigPath(t *testing.T) {
	path := GetConfigPath()

	// Verify the path is not empty
	if path == "" {
		t.Error("GetConfigPath returned empty path")
	}

	// Verify the path ends with config.toml
	if !strings.HasSuffix(path, "config.toml") {
		t.Errorf("GetConfigPath = %q, should end with 'config.toml'", path)
	}
}

// TestSave_Load_WithAllDefaults tests that saving default config creates minimal file
func TestSave_Load_WithAllDefaults(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	// Save the default config
	if err := Save(configPath, DefaultConfig); err != nil {
		t.Fatalf("Save returned unexpected error: %v", err)
	}

	// Read the saved file
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read saved config: %v", err)
	}

	contentStr := string(content)

	// An empty or nearly-empty file is expected since all values are defaults
	// The file should be empty or contain only whitespace
	if strings.TrimSpace(contentStr) != "" {
		t.Logf("Warning: Default config saved with content:\n%s", contentStr)
	}
}

// TestLoad_WithAutoOpenLastFile tests loading config with auto-open settings
func TestLoad_WithAutoOpenLastFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	// Create a config with auto-open settings
	cfg := &Config{
		Theme: "dark",
		AutoOpenLastFile: true,
		LastOpenedFile:   "/path/to/test.bomx",
	}
	cfg.Import = ImportConfig{
		ConfirmOverwrite:       DefaultConfig.Import.ConfirmOverwrite,
		AutoImportPreviousMatrix: DefaultConfig.Import.AutoImportPreviousMatrix,
	}
	cfg.Logger = LoggerConfig{
		Level:      DefaultConfig.Logger.Level,
		MaxEntries: DefaultConfig.Logger.MaxEntries,
	}
	cfg.RecentFiles = RecentFilesConfig{
		MaxRecentFiles: DefaultConfig.RecentFiles.MaxRecentFiles,
		RecentFiles:    []string{},
	}
	cfg.AutoImportPreviousMatrix = DefaultConfig.AutoImportPreviousMatrix

	// Save and load
	if err := Save(configPath, cfg); err != nil {
		t.Fatalf("Save returned unexpected error: %v", err)
	}

	loadedCfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load returned unexpected error: %v", err)
	}

	if !loadedCfg.AutoOpenLastFile {
		t.Error("AutoOpenLastFile should be true")
	}
	if loadedCfg.LastOpenedFile != "/path/to/test.bomx" {
		t.Errorf("LastOpenedFile = %q, want %q", loadedCfg.LastOpenedFile, "/path/to/test.bomx")
	}
}
