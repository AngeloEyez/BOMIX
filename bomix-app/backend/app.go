package backend

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"bomix-app/backend/config"
	"bomix-app/backend/db"
	"bomix-app/backend/excel"
	"bomix-app/backend/logger"
	"bomix-app/backend/task"
	"bomix-app/backend/types"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// App struct for Wails bindings
type App struct {
	app        *application.App
	logger     *logger.Logger
	cfg        *config.Config
	taskMgr    *task.Manager
	db         *gorm.DB
	mu         sync.RWMutex
}

// NewApp creates a new App instance
func NewApp(wailsApp *application.App, logger *logger.Logger, cfg *config.Config) *App {
	app := &App{
		app:     wailsApp,
		logger:  logger,
		cfg:     cfg,
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

// ==================== Series Management ====================

// CreateSeries creates a new series at the specified path
func (a *App) CreateSeries(path, name, description string) error {
	a.logger.Info(fmt.Sprintf("Creating new series: %s", path))

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

	a.logger.Info(fmt.Sprintf("Created series: %s (ID: %d)", name, series.ID))
	return nil
}

// OpenSeries opens an existing series at the specified path
func (a *App) OpenSeries(path string) error {
	a.logger.Info(fmt.Sprintf("Opening series: %s", path))

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

	// Store the database connection
	a.mu.Lock()
	a.db = database
	a.mu.Unlock()

	// Update config with last opened file
	a.cfg.LastOpenedFile = path
	if err := config.Save(config.GetConfigPath(), a.cfg); err != nil {
		a.logger.Warn(fmt.Sprintf("Failed to save config: %v", err))
	}

	// Add to recent files
	a.addToRecentFiles(path)

	a.logger.Info(fmt.Sprintf("Opened series: %s", path))
	return nil
}

// CloseSeries closes the current series
func (a *App) CloseSeries() error {
	a.logger.Info("Closing series")

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.db != nil {
		if err := db.Close(a.db); err != nil {
			return fmt.Errorf("failed to close database: %w", err)
		}
		a.db = nil
	}

	a.logger.Info("Series closed")
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

	return &SeriesInfo{
		ID:          series.ID,
		Name:        series.Name,
		Description: series.Description,
		Path:        a.cfg.LastOpenedFile,
	}, nil
}

// GetRecentSeries returns the list of recently opened series
func (a *App) GetRecentSeries() ([]*RecentFile, error) {
	recentFiles := make([]*RecentFile, 0, len(a.cfg.RecentFiles.RecentFiles))

	for _, path := range a.cfg.RecentFiles.RecentFiles {
		info, err := a.getSeriesInfoFromPath(path)
		if err != nil {
			continue // Skip files that don't exist
		}
		recentFiles = append(recentFiles, &RecentFile{
			Path:       path,
			Name:       info.Name,
			LastOpened: info.LastOpened,
		})
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
		CreatedAt:        revision.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        revision.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// ==================== Import/Export ====================

// ImportExcel imports Excel files into the database
func (a *App) ImportExcel(filePaths []string) ([]*ImportResult, error) {
	a.logger.Info(fmt.Sprintf("Importing %d Excel files", len(filePaths)))

	a.mu.RLock()
	dbConn := a.db
	a.mu.RUnlock()

	if dbConn == nil {
		return nil, fmt.Errorf("no series is currently open")
	}

	// Create Excel reader
	excelReader := excel.NewReader(dbConn)

	results := make([]*ImportResult, 0, len(filePaths))

	for _, filePath := range filePaths {
		taskID := uuid.New().String()

		// Submit import task
		taskID = a.taskMgr.Submit(
			fmt.Sprintf("Import: %s", filepath.Base(filePath)),
			"Import",
			func(ctx context.Context, progress func(float64, string)) error {
				// Update progress
				progress(0.1, "Detecting file format...")

				// Open the file
				f, err := excelize.OpenFile(filePath)
				if err != nil {
					return fmt.Errorf("failed to open file: %w", err)
				}
				defer f.Close()

				progress(0.3, "Importing data...")

				// Import the file
				importResults, err := excelReader.ImportExcel([]string{filePath})
				if err != nil {
					return err
				}

				if len(importResults) > 0 {
					result := importResults[0]
					progress(0.9, fmt.Sprintf("Imported %d parts", result.PartsCount))
				}

				progress(1.0, "Import completed")
				return nil
			},
		)

		results = append(results, &ImportResult{
			FileName:  filepath.Base(filePath),
			Status:    "queued",
			Message:   "Task created",
			TaskID:    taskID,
		})
	}

	return results, nil
}

// ExportExcel exports data from the database to Excel files
func (a *App) ExportExcel(options *ExportOptions) ([]string, error) {
	a.logger.Info("Exporting data to Excel")

	a.mu.RLock()
	dbConn := a.db
	a.mu.RUnlock()

	if dbConn == nil {
		return nil, fmt.Errorf("no series is currently open")
	}

	// Create Excel writer
	excelWriter, err := excel.NewWriter()
	if err != nil {
		return nil, fmt.Errorf("failed to create excel writer: %w", err)
	}

	// Convert options - convert RevisionIDs from []int64 to []string
	revisionIDsStr := make([]string, len(options.RevisionIDs))
	for i, id := range options.RevisionIDs {
		revisionIDsStr[i] = fmt.Sprintf("%d", id)
	}

	exportOptions := excel.ExportOptions{
		Format:              types.BOMFormat(options.Format),
		ProjectIDs:          options.ProjectIDs,
		RevisionIDs:         revisionIDsStr,
		Description:         options.Description,
		OutputPath:          options.OutputPath,
		OutputDir:           options.OutputDir,
		ModelCountOverrides: options.ModelCountOverrides,
	}

	// Export in a task
	taskID := uuid.New().String()
	taskID = a.taskMgr.Submit(
		fmt.Sprintf("Export: %s", options.Format),
		"Export",
		func(ctx context.Context, progress func(float64, string)) error {
			progress(0.1, "Preparing export data...")

			// Export to Excel
			outputPaths, err := excelWriter.ExportExcel(exportOptions)
			if err != nil {
				return fmt.Errorf("failed to export: %w", err)
			}

			progress(1.0, fmt.Sprintf("Exported %d files", len(outputPaths)))
			return nil
		},
	)

	a.logger.Info(fmt.Sprintf("Export task submitted: %s", taskID))
	return []string{taskID}, nil
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
			ConfirmOverwrite:       a.cfg.Import.ConfirmOverwrite,
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
	// Update config
	if settings.Theme != "" {
		a.cfg.Theme = settings.Theme
	}
	if settings.Import != nil {
		a.cfg.Import.ConfirmOverwrite = settings.Import.ConfirmOverwrite
		a.cfg.Import.AutoImportPreviousMatrix = settings.Import.AutoImportPreviousMatrix
	}
	if settings.Logger != nil {
		a.cfg.Logger.Level = settings.Logger.Level
		a.cfg.Logger.MaxEntries = settings.Logger.MaxEntries
	}
	if settings.RecentFiles != nil {
		a.cfg.RecentFiles.MaxRecentFiles = settings.RecentFiles.MaxRecentFiles
		a.cfg.RecentFiles.RecentFiles = settings.RecentFiles.RecentFiles
	}
	a.cfg.AutoOpenLastFile = settings.AutoOpenLastFile
	a.cfg.LastOpenedFile = settings.LastOpenedFile
	a.cfg.AutoImportPreviousMatrix = settings.AutoImportPreviousMatrix

	// Save config
	if err := config.Save(config.GetConfigPath(), a.cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	a.logger.Info("Settings updated")
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

// getSeriesInfoFromPath gets series info from a file path
func (a *App) getSeriesInfoFromPath(path string) (*SeriesInfoWithTime, error) {
	// Open database temporarily
	database, err := db.Open(path)
	if err != nil {
		return nil, err
	}
	defer db.Close(database)

	series, err := db.GetSeriesInfo(database)
	if err != nil {
		return nil, err
	}

	// Get file info for last opened time
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	return &SeriesInfoWithTime{
		Name:       series.Name,
		LastOpened: info.ModTime().Format(time.RFC3339),
	}, nil
}
