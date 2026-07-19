package task

// Event constants for task-related Wails events
// See product-spec section 5.1.4
const (
	EventTaskProgress  = "task:progress"   // Progress update
	EventTaskComplete  = "task:complete"   // Task completed
	EventTaskFailed    = "task:failed"     // Task failed
	EventTaskCancelled = "task:cancelled"  // Task cancelled
	EventTaskRunning   = "task:running"    // Task started running
)

// TaskProgressEvent is emitted when a task's progress updates
type TaskProgressEvent struct {
	TaskID   string  `json:"taskID"`
	Progress float64 `json:"progress"`
	Message  string  `json:"message"`
}

// TaskCompleteEvent is emitted when a task completes successfully
type TaskCompleteEvent struct {
	TaskID string `json:"taskID"`
}

// TaskFailedEvent is emitted when a task fails
type TaskFailedEvent struct {
	TaskID string `json:"taskID"`
	Error  string `json:"error"`
}

// TaskCancelledEvent is emitted when a task is cancelled
type TaskCancelledEvent struct {
	TaskID string `json:"taskID"`
}

// TaskRunningEvent is emitted when a task starts running
type TaskRunningEvent struct {
	TaskID string `json:"taskID"`
	Name   string `json:"name"`
}
