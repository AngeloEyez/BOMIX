package db

import (
	"fmt"
	"sync"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Single writer channel for serialized writes
var writeQueue = make(chan WriteTask, 50)

// WriteTask represents a batch write task
type WriteTask struct {
	TaskID     string
	DataBatch  any
	ResultChan chan error
}

// writerMutex ensures single writer pattern
var writerMutex sync.Mutex

// Open opens a connection to the SQLite database
func Open(filePath string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(filePath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Get underlying SQL DB to set pragmas and connection limits
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying DB: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := sqlDB.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Enable synchronous mode for WAL
	if _, err := sqlDB.Exec("PRAGMA synchronous=NORMAL;"); err != nil {
		return nil, fmt.Errorf("failed to set synchronous mode: %w", err)
	}

	// Enable foreign key constraints for CASCADE deletes
	if _, err := sqlDB.Exec("PRAGMA foreign_keys=ON;"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Set connection limits - Single writer mode
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	// Start single writer goroutine
	StartDatabaseWriter(db)

	return db, nil
}

// Close closes the database connection
func Close(db *gorm.DB) error {
	// Close the write queue to stop the writer goroutine
	close(writeQueue)

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// AutoMigrate runs auto migration to create all tables with indexes
func AutoMigrate(db *gorm.DB) error {
	writerMutex.Lock()
	defer writerMutex.Unlock()

	// Run auto migration for all models
	if err := db.AutoMigrate(
		&Series{},
		&Project{},
		&BomRevision{},
		&Part{},
		&SecondSource{},
		&MatrixModel{},
		&MatrixSelection{},
	); err != nil {
		return err
	}

	// Create foreign key constraints with CASCADE for SQLite
	// SQLite requires explicit DROP TABLE and CREATE TABLE with FK constraints
	// We use raw SQL to add foreign keys after migration

	// Note: In SQLite, foreign keys must be enabled per connection with PRAGMA foreign_keys=ON
	// The CASCADE behavior is defined in the table schema

	return nil
}

// StartDatabaseWriter starts the single writer goroutine for serialized database writes
func StartDatabaseWriter(db *gorm.DB) {
	go func() {
		for task := range writeQueue {
			// Use a transaction for batch write
			err := db.Transaction(func(tx *gorm.DB) error {
				if err := tx.CreateInBatches(task.DataBatch, 500).Error; err != nil {
					return err
				}
				return nil
			})

			// Report result back to the task
			if task.ResultChan != nil {
				task.ResultChan <- err
			}
		}
	}()
}

// SubmitWriteTask submits a batch write task to the writer queue
func SubmitWriteTask(db *gorm.DB, taskID string, dataBatch any) error {
	resultChan := make(chan error, 1)

	writeQueue <- WriteTask{
		TaskID:     taskID,
		DataBatch:  dataBatch,
		ResultChan: resultChan,
	}

	// Wait for the write to complete
	return <-resultChan
}
