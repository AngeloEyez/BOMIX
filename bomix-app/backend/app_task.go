package backend

import (
	"fmt"
	"time"
)

// ==================== 非同步任務管理 (Async Task Management) ====================

// ListTasks 取得所有背景任務清單及其最新狀態
//
// 回傳：
//   - []*Task: 任務清單 DTO
//   - error: 固定回傳 nil
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

// GetTask 依任務 ID 取得特定背景任務的詳細資訊與進度狀態
//
// 參數：
//   - id: 任務 ID
//
// 回傳：
//   - *Task: 任務詳細資訊 DTO
//   - error: 若任務不存在則回傳錯誤
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

// CancelTask 取消指定 ID 的執行中或等待中任務
//
// 參數：
//   - id: 任務 ID
//
// 回傳：
//   - error: 若取消失敗則回傳錯誤
func (a *App) CancelTask(id string) error {
	return a.taskMgr.Cancel(id)
}
