package task

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"bomix-app/backend/logger"
	"bomix-app/backend/types"
)

// EventEmitter is an interface for emitting events
type EventEmitter interface {
	EmitEvent(event string, data interface{})
}

// Task represents a background task
// See product-spec section 5.1.2
type Task struct {
	ID          string     // UUID
	Name        string     // Human-readable name
	Type        string     // Import / Export / Analysis
	Status      string     // Created / Queued / Running / Completed / Failed / Cancelled
	Progress    float64    // 0.0 ~ 1.0
	Message     string     // Current step description
	Error       string     // Error message if failed
	CreatedAt   time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
	ctx         context.Context // Context for cancellation
	cancel      context.CancelFunc
	mu          sync.RWMutex // Mutex for thread-safe access
}

// GetStatus returns the task status (thread-safe)
func (t *Task) GetStatus() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Status
}

// GetProgress returns the task progress (thread-safe)
func (t *Task) GetProgress() float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Progress
}

// TaskFunc is the signature for task functions
// It receives a context for cancellation and a progress callback
type TaskFunc func(ctx context.Context, progressCb func(progress float64, message string)) error

// TaskManager manages background tasks
// See product-spec section 5.1.3
type TaskManager struct {
	tasks   map[string]*Task
	logger  *logger.Logger
	emitter EventEmitter
	mu      sync.RWMutex
}

// NewTaskManager creates a new task manager
func NewTaskManager(logger *logger.Logger, emitter EventEmitter) *TaskManager {
	return &TaskManager{
		tasks:   make(map[string]*Task),
		logger:  logger,
		emitter: emitter,
	}
}

// Submit submits a new task for execution
// Returns the task ID
func (tm *TaskManager) Submit(name, taskType string, fn TaskFunc) string {
	taskID := uuid.New().String()
	return tm.SubmitWithID(taskID, name, taskType, fn)
}

// SubmitWithID submits a new task with a specific ID
func (tm *TaskManager) SubmitWithID(taskID, name, taskType string, fn TaskFunc) string {
	task := &Task{
		ID:        taskID,
		Name:      name,
		Type:      taskType,
		Status:    string(types.TaskCreated),
		Progress:  0.0,
		CreatedAt: time.Now(),
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	task.ctx = ctx
	task.cancel = cancel

	tm.mu.Lock()
	tm.tasks[taskID] = task
	tm.mu.Unlock()

	// Log task creation
	if tm.logger != nil {
		tm.logger.Info("任務已建立",
			"taskID", taskID,
			"name", name,
			"type", taskType,
			"taskStatus", "queued",
		)
	}

	// Start task in a goroutine
	go func() {
		defer cancel()

		// Update status to running
		task.mu.Lock()
		task.Status = string(types.TaskRunning)
		now := time.Now()
		task.StartedAt = &now
		task.mu.Unlock()

		// Emit running event
		tm.emitEvent(EventTaskRunning, map[string]interface{}{
			"taskID": taskID,
			"name":   name,
		})

		if tm.logger != nil {
			tm.logger.Info("任務已開始",
				"taskID", taskID,
				"name", name,
				"taskStatus", "running",
			)
		}

		// Create progress callback
		progressCb := func(progress float64, message string) {
			task.mu.Lock()
			task.Progress = progress
			task.Message = message
			task.mu.Unlock()

			// Emit progress event
			tm.emitEvent(EventTaskProgress, map[string]interface{}{
				"taskID":   taskID,
				"progress": progress,
				"message":  message,
			})

			if tm.logger != nil {
				tm.logger.Debug("任務進度",
					"taskID", taskID,
					"progress", progress,
					"message", message,
					"taskStatus", "running",
				)
			}
		}

		// Execute the task
		err := fn(ctx, progressCb)

		task.mu.Lock()
		if err != nil {
			if ctx.Err() == context.Canceled {
				// Task was cancelled
				task.Status = string(types.TaskCancelled)
				task.Message = "Task was cancelled"
				task.mu.Unlock()
				tm.emitEvent(EventTaskCancelled, map[string]interface{}{
					"taskID": taskID,
				})
				if tm.logger != nil {
					tm.logger.Info("任務已取消",
						"taskID", taskID,
						"name", name,
						"taskStatus", "cancelled",
					)
				}
				return
			}

			// Task failed with error
			task.Status = string(types.TaskFailed)
			task.Error = err.Error()
			task.Message = "Task failed"
			task.mu.Unlock()

			tm.emitEvent(EventTaskFailed, map[string]interface{}{
				"taskID": taskID,
				"error":  err.Error(),
			})

			if tm.logger != nil {
				tm.logger.Error("任務失敗",
					"taskID", taskID,
					"name", name,
					"error", err.Error(),
					"taskStatus", "error",
				)
			}
		} else {
			// Task completed successfully
			task.Status = string(types.TaskCompleted)
			task.Progress = 1.0
			task.Message = "Task completed successfully"
			task.mu.Unlock()

			tm.emitEvent(EventTaskComplete, map[string]interface{}{
				"taskID": taskID,
			})

			if tm.logger != nil {
				tm.logger.Info("任務已完成",
					"taskID", taskID,
					"name", name,
					"taskStatus", "done",
				)
			}
		}

		completedAt := time.Now()
		task.CompletedAt = &completedAt
	}()

	return taskID
}

// Cancel cancels a task
func (tm *TaskManager) Cancel(taskID string) error {
	tm.mu.RLock()
	task, exists := tm.tasks[taskID]
	tm.mu.RUnlock()

	if !exists {
		return errors.New("task not found")
	}

	task.mu.Lock()
	// Only cancel if the task is still running
	if task.Status == string(types.TaskRunning) || task.Status == string(types.TaskCreated) || task.Status == string(types.TaskQueued) {
		if task.cancel != nil {
			task.cancel()
		}
		task.Status = string(types.TaskCancelled)
	}
	task.mu.Unlock()

	tm.emitEvent(EventTaskCancelled, map[string]interface{}{
		"taskID": taskID,
	})

	if tm.logger != nil {
		tm.logger.Info("使用者取消了任務",
			"taskID", taskID,
			"name", task.Name,
			"taskStatus", "cancelled",
		)
	}

	return nil
}

// GetStatus returns the status of a task
// Returns a new Task with the same data but without the internal mutex/context
func (tm *TaskManager) GetStatus(taskID string) *Task {
	tm.mu.RLock()
	task, exists := tm.tasks[taskID]
	tm.mu.RUnlock()

	if !exists {
		return nil
	}

	task.mu.RLock()
	defer task.mu.RUnlock()

	// Return a new Task with copied data (excluding ctx and cancel which are internal)
	return &Task{
		ID:          task.ID,
		Name:        task.Name,
		Type:        task.Type,
		Status:      task.Status,
		Progress:    task.Progress,
		Message:     task.Message,
		Error:       task.Error,
		CreatedAt:   task.CreatedAt,
		StartedAt:   task.StartedAt,
		CompletedAt: task.CompletedAt,
	}
}

// ListTasks returns all tasks
// Returns a slice of new Tasks with copied data (excluding internal fields)
func (tm *TaskManager) ListTasks() []*Task {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tasks := make([]*Task, 0, len(tm.tasks))
	for _, task := range tm.tasks {
		task.mu.RLock()
		taskCopy := &Task{
			ID:          task.ID,
			Name:        task.Name,
			Type:        task.Type,
			Status:      task.Status,
			Progress:    task.Progress,
			Message:     task.Message,
			Error:       task.Error,
			CreatedAt:   task.CreatedAt,
			StartedAt:   task.StartedAt,
			CompletedAt: task.CompletedAt,
		}
		task.mu.RUnlock()
		tasks = append(tasks, taskCopy)
	}
	return tasks
}

// emitEvent emits a task event to the frontend
func (tm *TaskManager) emitEvent(event string, data interface{}) {
	if tm.emitter != nil {
		tm.emitter.EmitEvent(event, data)
	}
}
