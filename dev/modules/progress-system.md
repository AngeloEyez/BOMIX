# 進度回饋系統 (Progress Feedback System)

## 概述

進度回饋系統用於追蹤後端長時間執行的任務（如 Excel 匯出、匯入），並即時將進度狀態推送到前端 UI。
系統採用 **Async Task** 模式：API 呼叫後立即回傳 `taskId`，隨後透過 IPC 事件廣播進度更新。

## 核心概念

- **Task ID**: 每個任務的唯一識別碼 (UUID)。
- **Task Status**:
  - `PENDING`: 任務已建立但尚未開始
  - `RUNNING`: 任務執行中
  - `COMPLETED`: 任務成功完成
  - `FAILED`: 任務執行失敗
  - `CANCELLED`: 任務被取消
- **Progress**: 0 到 100 的整數。

## API 參考

### 1. `window.api.progress`

#### `get(taskId)`
取得指定任務的最新狀態。

- **參數**: `taskId` (string)
- **回傳**: Promise<TaskObject>
- **範例**:
  ```javascript
  const task = await window.api.progress.get('some-task-id');
  console.log(task.status, task.progress);
  ```

#### `cancel(taskId)`
取消指定任務。

- **參數**: `taskId` (string)
- **回傳**: Promise<{ cancelled: true }>

#### `onUpdate(callback)`
訂閱全域進度更新事件。

- **參數**: `callback` (function(task))
- **回傳**: `unsubscribe` (function) - 呼叫此函數以取消訂閱
- **範例**:
  ```javascript
  useEffect(() => {
    const unsubscribe = window.api.progress.onUpdate((task) => {
      console.log('Task Update:', task.id, task.progress, task.message);
      if (task.id === myCurrentTaskId) {
        setProgress(task.progress);
      }
    });
    return unsubscribe;
  }, []);
  ```

### 2. 相關模組變更

#### Excel 匯出 (`window.api.excel.export`)

**行為變更**:
此 API 不再等待匯出完成，而是立即回傳 `taskId`。

- **參數**:
  - `bomRevisionId` (number)
  - `outputFilePath` (string, Optional) - 若提供，完成後會自動移動檔案至此路徑；若不提供，檔案保留在 Temp 目錄。
- **回傳**: `{ success: true, data: { taskId: string } }`
- **流程**:
  1. 呼叫 `excel.export(...)` 取得 `taskId`。
  2. 監聽 `progress.onUpdate` 更新 UI 進度條。
  3. 當 `task.status === 'COMPLETED'` 時，`task.result` 包含 `{ filePath: string }`，即為最終檔案路徑。
  4. 當 `task.status === 'FAILED'` 時，`task.error` 包含錯誤訊息。

## 資料結構

### Task Object

```javascript
{
  id: "uuid-string",
  type: "EXPORT_BOM",    // 任務類型
  status: "RUNNING",     // PENDING | RUNNING | COMPLETED | FAILED | CANCELLED
  progress: 45,          // 0-100
  message: "Processing sheet: SMD...",
  result: null,          // 完成時的結果 (例如 { filePath: ... })
  error: null,           // 失敗時的錯誤訊息
  metadata: { ... },     // 建立任務時的額外資訊
  createdAt: "ISO-Date",
  updatedAt: "ISO-Date"
}
```
