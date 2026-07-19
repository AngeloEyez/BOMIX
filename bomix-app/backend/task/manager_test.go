package task

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"bomix-app/backend/types"
)

// mockEmitter is a mock implementation of EventEmitter for testing
type mockEmitter struct {
	events chan string
}

func (m *mockEmitter) EmitEvent(event string, data interface{}) {
	if m.events != nil {
		m.events <- event
	}
}

func TestTaskManager_Submit_Completed(t *testing.T) {
	emitter := &mockEmitter{}

	tm := NewTaskManager(nil, emitter)

	// Submit a task that completes successfully
	taskID := tm.Submit("Test Task", "Import", func(ctx context.Context, progressCb func(float64, string)) error {
		// Simulate progress updates
		progressCb(0.0, "Starting...")
		time.Sleep(10 * time.Millisecond)
		progressCb(0.5, "Halfway...")
		time.Sleep(10 * time.Millisecond)
		progressCb(1.0, "Done!")
		return nil
	})

	// Wait for task to complete
	time.Sleep(100 * time.Millisecond)

	// Check task status
	task := tm.GetStatus(taskID)
	if task == nil {
		t.Fatal("Task not found")
	}

	if task.Status != string(types.TaskCompleted) {
		t.Errorf("Expected status %s, got %s", types.TaskCompleted, task.Status)
	}

	if task.Progress != 1.0 {
		t.Errorf("Expected progress 1.0, got %f", task.Progress)
	}

	if task.Error != "" {
		t.Errorf("Expected no error, got %s", task.Error)
	}
}

func TestTaskManager_Submit_Failed(t *testing.T) {
	emitter := &mockEmitter{}

	tm := NewTaskManager(nil, emitter)

	// Submit a task that fails
	testErr := errors.New("test error")
	taskID := tm.Submit("Failing Task", "Import", func(ctx context.Context, progressCb func(float64, string)) error {
		progressCb(0.3, "About to fail...")
		return testErr
	})

	// Wait for task to fail
	time.Sleep(100 * time.Millisecond)

	// Check task status
	task := tm.GetStatus(taskID)
	if task == nil {
		t.Fatal("Task not found")
	}

	if task.Status != string(types.TaskFailed) {
		t.Errorf("Expected status %s, got %s", types.TaskFailed, task.Status)
	}

	if task.Error != testErr.Error() {
		t.Errorf("Expected error %q, got %q", testErr.Error(), task.Error)
	}
}

func TestTaskManager_Cancel(t *testing.T) {
	emitter := &mockEmitter{}

	tm := NewTaskManager(nil, emitter)

	// Submit a long-running task
	taskID := tm.Submit("Long Task", "Import", func(ctx context.Context, progressCb func(float64, string)) error {
		for i := 0; i < 100; i++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				progressCb(float64(i)/100.0, "Working...")
				time.Sleep(10 * time.Millisecond)
			}
		}
		return nil
	})

	// Wait a bit for task to start
	time.Sleep(50 * time.Millisecond)

	// Cancel the task
	err := tm.Cancel(taskID)
	if err != nil {
		t.Fatalf("Failed to cancel task: %v", err)
	}

	// Wait for cancellation to propagate
	time.Sleep(50 * time.Millisecond)

	// Check task status
	task := tm.GetStatus(taskID)
	if task == nil {
		t.Fatal("Task not found")
	}

	if task.Status != string(types.TaskCancelled) {
		t.Errorf("Expected status %s, got %s", types.TaskCancelled, task.Status)
	}
}

func TestTaskManager_ListTasks(t *testing.T) {
	emitter := &mockEmitter{}

	tm := NewTaskManager(nil, emitter)

	// Submit multiple tasks
	taskID1 := tm.Submit("Task 1", "Import", func(ctx context.Context, progressCb func(float64, string)) error {
		return nil
	})

	taskID2 := tm.Submit("Task 2", "Export", func(ctx context.Context, progressCb func(float64, string)) error {
		return nil
	})

	taskID3 := tm.Submit("Task 3", "Analysis", func(ctx context.Context, progressCb func(float64, string)) error {
		return nil
	})

	// Wait for tasks to complete
	time.Sleep(100 * time.Millisecond)

	// List all tasks
	tasks := tm.ListTasks()

	if len(tasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(tasks))
	}

	// Verify all tasks are present
	taskMap := make(map[string]bool)
	for _, task := range tasks {
		taskMap[task.ID] = true
	}

	if !taskMap[taskID1] {
		t.Error("Task 1 not found in list")
	}
	if !taskMap[taskID2] {
		t.Error("Task 2 not found in list")
	}
	if !taskMap[taskID3] {
		t.Error("Task 3 not found in list")
	}
}

func TestTaskManager_ProgressUpdates(t *testing.T) {
	emitter := &mockEmitter{}

	tm := NewTaskManager(nil, emitter)

	// Submit a task with multiple progress updates
	taskID := tm.Submit("Progress Task", "Import", func(ctx context.Context, progressCb func(float64, string)) error {
		// Simulate progress: 0.0 -> 0.25 -> 0.5 -> 0.75 -> 1.0
		progressCb(0.0, "Starting...")
		time.Sleep(10 * time.Millisecond)
		progressCb(0.25, "First quarter...")
		time.Sleep(10 * time.Millisecond)
		progressCb(0.5, "Halfway...")
		time.Sleep(10 * time.Millisecond)
		progressCb(0.75, "Three quarters...")
		time.Sleep(10 * time.Millisecond)
		progressCb(1.0, "Complete!")
		return nil
	})

	// Wait for task to complete
	time.Sleep(200 * time.Millisecond)

	// Check final task status
	task := tm.GetStatus(taskID)
	if task == nil {
		t.Fatal("Task not found")
	}

	if task.Status != string(types.TaskCompleted) {
		t.Errorf("Expected status %s, got %s", types.TaskCompleted, task.Status)
	}

	if task.Progress != 1.0 {
		t.Errorf("Expected progress 1.0, got %f", task.Progress)
	}
}

func TestTaskManager_GetStatus_NonExistent(t *testing.T) {
	emitter := &mockEmitter{}

	tm := NewTaskManager(nil, emitter)

	// Try to get status of a non-existent task
	task := tm.GetStatus("non-existent-id")
	if task != nil {
		t.Errorf("Expected nil for non-existent task, got %+v", task)
	}
}

func TestTaskManager_Cancel_NonExistent(t *testing.T) {
	emitter := &mockEmitter{}

	tm := NewTaskManager(nil, emitter)

	// Try to cancel a non-existent task
	err := tm.Cancel("non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent task, got nil")
	}
}

func TestTaskStatusTransitions(t *testing.T) {
	emitter := &mockEmitter{}

	tm := NewTaskManager(nil, emitter)

	// Submit a task that goes through all states
	taskID := tm.Submit("State Transition Task", "Import", func(ctx context.Context, progressCb func(float64, string)) error {
		time.Sleep(10 * time.Millisecond)
		progressCb(0.5, "In progress...")
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	// Wait for task to complete
	time.Sleep(100 * time.Millisecond)

	// Verify the task completed successfully
	task := tm.GetStatus(taskID)
	if task == nil {
		t.Fatal("Task not found")
	}

	if task.Status != string(types.TaskCompleted) {
		t.Errorf("Expected final status %s, got %s", types.TaskCompleted, task.Status)
	}

	// Verify timestamps are set correctly
	if task.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if task.StartedAt == nil || task.StartedAt.IsZero() {
		t.Error("StartedAt should be set")
	}
	if task.CompletedAt == nil || task.CompletedAt.IsZero() {
		t.Error("CompletedAt should be set")
	}

	// Verify StartedAt is after CreatedAt
	if task.StartedAt.Before(task.CreatedAt) {
		t.Error("StartedAt should be after CreatedAt")
	}

	// Verify CompletedAt is after StartedAt
	if task.CompletedAt.Before(*task.StartedAt) {
		t.Error("CompletedAt should be after StartedAt")
	}
}

func TestTaskManager_ConcurrentAccess(t *testing.T) {
	emitter := &mockEmitter{}

	tm := NewTaskManager(nil, emitter)

	// Submit multiple tasks concurrently
	var wg sync.WaitGroup
	numTasks := 10

	for i := 0; i < numTasks; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			taskID := tm.Submit("Concurrent Task", "Import", func(ctx context.Context, progressCb func(float64, string)) error {
				time.Sleep(10 * time.Millisecond)
				return nil
			})

			// Wait a bit and check status
			time.Sleep(50 * time.Millisecond)
			task := tm.GetStatus(taskID)
			if task == nil {
				t.Errorf("Task %d not found", id)
			}
		}(i)
	}

	wg.Wait()

	// All tasks should be listed
	tasks := tm.ListTasks()
	if len(tasks) != numTasks {
		t.Errorf("Expected %d tasks, got %d", numTasks, len(tasks))
	}
}
