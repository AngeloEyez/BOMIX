# 任務排程管理器 (Task Manager System)

## 概述

任務排程管理器用於追蹤後端長時間執行的任務（如 Excel 匯出、匯入，以及未來可能遇到的非同步任務如資料庫維護、分析），並即時將進度狀態推送到前端 UI。
系統採用 **FIFO (First-In-First-Out) 佇列排程** 與 **Async Task** 模式：API 呼叫後立即回傳 `taskId`，隨後透過 IPC 事件廣播進度更新，並且實作非阻塞的 UI 體驗。

## 核心概念

- **Task ID**: 每個任務的唯一識別碼 (UUID)。
- **Task Status**:
  - `QUEUED`: 任務已建立並加入排程佇列，等待執行
  - `RUNNING`: 任務執行中
  - `COMPLETED`: 任務成功完成
  - `FAILED`: 任務執行失敗
  - `CANCELLED`: 任務被取消（僅 QUEUED 狀態可取消）
- **Progress**: 0 到 100 的整數。
- **Callback 機制**: 任務完成時可透過 `task:completed` 事件觸發前端對應行為（如重新載入資料）。

## API 參考

### 1. `window.api.task`

#### `get(taskId)`
取得指定任務的最新狀態。

- **參數**: `taskId` (string)
- **回傳**: Promise<TaskObject>

#### `cancel(taskId)`
取消指定任務。**注意：僅狀態為 `QUEUED` 的任務可取消。**

- **參數**: `taskId` (string)
- **回傳**: Promise<{ cancelled: true }>

#### `remove(taskId)`
移除已結束任務的紀錄（無法移除 `QUEUED` 或 `RUNNING` 的任務）。

- **參數**: `taskId` (string)
- **回傳**: Promise<{ removed: true }>

#### `getQueueStatus()`
取得當前佇列狀態概覽。

- **回傳**: Promise<{ currentTask: TaskObject|null, queueLength: number, totalTasks: number }>

#### `onUpdate(callback)`
訂閱全域進度更新事件。

- **參數**: `callback` (function(task))
- **回傳**: `unsubscribe` (function) - 呼叫此函數以取消訂閱

#### `onCompleted(callback)`
訂閱全域任務完成事件。用於接收帶有 type 與 result 的事件。

- **參數**: `callback` (function({ id, type, result, metadata }))
- **回傳**: `unsubscribe` (function)

### 2. 相關模組變更

#### Excel 匯出與匯入 (`window.api.excel`)

**行為變更**:
匯出與匯入現在都改走 `window.api.excel` 但背景統一透過 `taskManager.enqueue` 進行排程，回傳 `taskId` 而非等待結果。

- **匯入**: `window.api.excel.import(...)` -> 回傳 `{ success: true, data: { taskId: string } }`
- **匯出**: `window.api.excel.export(...)` -> 回傳 `{ success: true, data: { taskId: string } }`

## 資料結構

### Task Object

```javascript
{
  id: "uuid-string",
  type: "EXPORT_BOM",    // 任務類型 (e.g. EXPORT_BOM, IMPORT_BOM)
  title: "匯出 BOM Excel", // 任務顯示名稱
  status: "RUNNING",     // QUEUED | RUNNING | COMPLETED | FAILED | CANCELLED
  progress: 45,          // 0-100
  message: "Processing sheet: SMD...", // 當前狀態訊息或 info 等級的 log
  logs: [                // 詳細日誌紀錄
      { 
        timestamp: "2023-10-27T10:00:00.000Z", 
        message: "Processing sheet: SMD...", 
        level: "info" // 'info' | 'warn' | 'error'
      }
  ],
  result: { filePath: ... }, // 完成時的結果（視 type 而定），執行中為 null
  error: null,           // 失敗時的錯誤訊息字串
  metadata: { ... },     // 建立任務時的額外來源資訊
  createdAt: "ISO-Date",
  updatedAt: "ISO-Date"
}
```
