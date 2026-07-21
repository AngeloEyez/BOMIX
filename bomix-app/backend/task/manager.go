package task

import (
	"sync"
	"time"

	"bomix-app/backend/logger"
	"bomix-app/backend/types"
)

// Manager extends TaskManager with additional management features
type Manager struct {
	*TaskManager
	workers int
	mu      sync.RWMutex
}

// NewManager creates a new task manager with worker pool
func NewManager(logger *logger.Logger, emitter EventEmitter) *Manager {
	return &Manager{
		TaskManager: NewTaskManager(logger, emitter),
		workers:     4, // Default to 4 workers
	}
}

// SubmitWithPriority submits a task with priority
// For now, just uses the base Submit method
// Priority handling can be added later with a queue system
func (m *Manager) SubmitWithPriority(name, taskType string, priority int, fn TaskFunc) string {
	_ = priority // Priority not yet implemented
	return m.TaskManager.Submit(name, taskType, fn)
}

// SubmitWithIDAndPriority submits a task with a specific ID and priority
func (m *Manager) SubmitWithIDAndPriority(taskID, name, taskType string, priority int, fn TaskFunc) string {
	_ = priority
	return m.TaskManager.SubmitWithID(taskID, name, taskType, fn)
}

// GetActiveTasks returns all active (running or queued) tasks
func (m *Manager) GetActiveTasks() []*Task {
	tasks := m.TaskManager.ListTasks()
	var active []*Task

	for _, task := range tasks {
		if task.Status == string(types.TaskQueued) || task.Status == string(types.TaskRunning) {
			active = append(active, task)
		}
	}
	return active
}

// CleanupCompleted removes completed/failed tasks older than duration
func (m *Manager) CleanupCompleted(duration time.Duration) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-duration)
	count := 0

	// Note: This accesses internal state, which may need refactoring
	// For now, this is a placeholder for future implementation
	_ = cutoff
	_ = count

	return 0
}

// GetQueueSize returns the number of queued tasks
func (m *Manager) GetQueueSize() int {
	tasks := m.TaskManager.ListTasks()
	count := 0

	for _, task := range tasks {
		if task.Status == string(types.TaskQueued) || task.Status == string(types.TaskRunning) {
			count++
		}
	}
	return count
}
