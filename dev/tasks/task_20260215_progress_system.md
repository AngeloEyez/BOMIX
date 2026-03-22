# 任務：實作進度回饋系統 (Backend Progress Feedback System)

## 任務目標
在主行程 (Main Process) 中實作一套完整的進度回饋系統，用於追蹤耗時任務（如匯入、匯出）並將狀態、進度與日誌即時傳遞給前端 UI。

此系統需取代舊有的簡單 callback 機制，改用結構化的 `Session` 物件管理。

## 1. 新增服務：ProgressService
**檔案位置**: `src/main/services/progress.service.js`

請實作 `ProgressService` 類別或模組，包含以下功能：

### 資料結構 (Session)
```javascript
{
  id: string,        // UUID (使用 crypto.randomUUID())
  title: string,     // 任務標題，例如 "匯出 Excel"
  taskId: string,    // 識別碼，例如 "export_ebom"
  currentTask: string, // 當前執行動作，例如 "正在寫入工作表: SMD"
  status: 'active' | 'completed' | 'failed',
  progress: number,  // 0-100
  logs: Array<{
      timestamp: number,
      message: string,
      level: 'info' | 'warn' | 'error',
      details?: any
  }>,
  startTime: number,
  endTime: number | null
}
```

### 方法 API
- `createSession(title, taskId)`: 建立新 Session，儲存於記憶體中，並透過 IPC 通知 UI。
- `updateTask(sessionId, currentTask, progress)`: 更新當前動作描述與進度 (0-100)。
- `log(sessionId, message, level = 'info', details = null)`: 新增一筆日誌。
- `complete(sessionId, resultData = null)`: 標記 Session 為 `completed`，設定 `endTime`。
- `fail(sessionId, error)`: 標記 Session 為 `failed`，記錄錯誤日誌。
- `getSession(sessionId)`: 取得 Session 物件。

---

## 2. 新增 IPC 通道
**檔案位置**: `src/main/ipc/progress.ipc.js`

請定義並註冊以下 IPC 通道（Main -> Renderer）：

- `progress:session-start`: 當 `createSession` 被呼叫時發送。
- `progress:session-update`: 當狀態、進度或 currentTask 更新時發送。
- `progress:session-log`: 當有新日誌時發送 (可優化為不送整個 logs 陣列，只送新的一筆)。
- `progress:session-end`: 當 `complete` 或 `fail` 時發送。

**注意**: 請確保這些事件能廣播給所有開啟的視窗 (Renderer)。

---

## 3. 整合 Excel 匯出服務
**檔案位置**: `src/main/services/export.service.js`
**檔案位置**: `src/main/services/excel-export/template-engine.js`

### 3.1 暫存檔寫入 (Safe Write)
- 匯出時先寫入 `<outputFilePath>.bomixtmp`。
- 匯出完成後 (在 `complete` 前)，將檔案重新命名為使用者的目標路徑。
- 若失敗，保留或刪除暫存檔（建議刪除）。

### 3.2 非同步與防阻塞 (Non-blocking)
- 修改 `template-engine.js` 中的迴圈邏輯。
- 使用 `setImmediate` 或 `setTimeout(..., 0)` 讓出 Event Loop，避免 UI 凍結。
- 建議改為 `async` 函式，並在處理每 N 行 (例如 500 行) 資料後 `await new Promise(...)` 使用一下 `setImmediate`。

### 3.3 整合 ProgressService
- 在 `exportBom` 開始時建立 Session。
- 在處理每個 Sheet 時呼叫 `updateTask` (更新進度)。
- 關鍵步驟呼叫 `log`。
- `try-catch` 區塊中處理 `complete` 與 `fail`。

---

## 4. 整合 Excel 匯入服務 (Optional)
**檔案位置**: `src/main/services/import.service.js`

- 比照匯出服務，將 `importBom` 整合 `ProgressService`。
- 在讀取 Sheet、解析資料、寫入資料庫等階段更新進度。

---

## 5. 文件產出
**檔案位置**: `dev/modules/progress-system.md`

請撰寫一份 Markdown 文件，供前端開發者參考：
- 說明 IPC 通道名稱與參數格式。
- 說明 Session 物件結構。
- 範例：前端如何監聽並顯示進度。

## 開發規範提醒
- **語言**: 所有程式碼註解、Commit 訊息、文件, commit, PR, PR 回應  均須使用 **繁體中文**。
- **測試**: 請為 `progress.service.js` 建立單元測試 (`tests/unit/services/progress.service.test.js`)。
