# BOMIX 系統架構設計

> 版本：1.2.0 | 最後更新：2026-03-18

## 架構概覽

BOMIX 採用 Electron 三層式架構，主行程與渲染層透過 IPC 通訊：

```
┌─────────────────────────────────────────────┐
│      渲染層 (React + Tailwind CSS)           │  ← 頁面、元件、狀態管理
├─────────────────────────────────────────────┤
│      Preload (contextBridge)                │  ← 安全 API 橋接
├─────────────────────────────────────────────┤
│      主行程 IPC Handlers                     │  ← 接收/回應渲染層請求
├─────────────────────────────────────────────┤
│      Services 層                             │  ← 業務邏輯
├─────────────────────────────────────────────┤
│      Database 層 (better-sqlite3)            │  ← Repository + 連線管理
└─────────────────────────────────────────────┘
```

## 資料流

```
React Component → Zustand Store → window.api.xxx()
→ IPC invoke → Main Process Handler → Service → Repository → SQLite
→ 回傳結果 → IPC reply → Store 更新 → React re-render
```

```
React Component → Zustand Store → window.api.xxx()
→ IPC invoke → Main Process Handler → Service → Repository → SQLite
→ 回傳結果 → IPC reply → Store 更新 → React re-render
```

```
Excel 檔案 → excel.ipc → taskManager.enqueue() → importService (非阻塞) → SQLite
SQLite → bomService → exportService → taskManager.enqueue() → Excel 檔案
SQLite → partsRepo → compareService → 差異報告 → React UI
SQLite → matrixService → Matrix Data (含 Selection) → React UI
```

## UI 架構

### React 元件樹
```
App
├── AppLayout (components/layout/)
│   ├── Header                          ← 頂部標題列（含主題切換）
│   ├── Sidebar                         ← 左側導航列
│   ├── Content Area                    ← 頁面動態切換
│   │   ├── HomePage                    ← 首頁（歡迎畫面）
│   │   ├── ProjectPage                 ← 專案管理
│   │   ├── BomPage                     ← BOM 檢視/編輯/Matrix
│   │   ├── ComparePage                 ← 版本比較
│   │   └── SettingsPage                ← 設定
│   └── StatusBar                       ← 底部狀態列
└── Dialogs (components/dialogs/)
    ├── AboutDialog                     ← 關於對話框
    ├── ChangelogDialog                 ← 更新記錄
    ├── ImportDialog                    ← 匯入對話框
    └── MatrixModelDialog               ← Matrix Model 管理
```

### 導航機制
- 使用 React `useState` 控制頁面切換
- 頁面以條件渲染方式動態顯示（避免重建元件）
- 導航 ID 與頁面元件一對一映射

### 主題系統
- 透過 `<html>` 元素的 `class="dark"` 切換深色模式
- Tailwind CSS 的 `dark:` variant 自動套用深色樣式
- 啟動時偵測系統主題 → 使用者偏好覆蓋
- 設定儲存至 `%APPDATA%/BOMIX/settings.json`

## 主行程架構

### IPC 通訊模式
- 使用 `ipcMain.handle` / `ipcRenderer.invoke` 的 Request-Response 模式
- 所有 API 透過 Preload 的 `contextBridge` 安全暴露
- 渲染層統一透過 `window.api` 存取主行程功能

### IPC 通道命名規範
```
{模組}:{動作}
例：series:create, project:list, bom:getMainItems, matrix:getData
```

### 模組依賴方向
```
IPC Handlers → Services → Repositories → SQLite
                 ↓
              Models（純資料結構）
```

- IPC Handlers 只負責參數解析與回傳，不含業務邏輯
- Services 層負責業務邏輯，可呼叫 Repositories 與使用 Models
- Repositories 層負責 SQL 操作，使用 Repository Pattern
- Models 只定義資料結構，不包含業務邏輯

## 資料庫設計概要

- 7 個資料表：`series_meta`, `projects`, `bom_revisions`, `parts`, `second_sources`, `matrix_models`, `matrix_selections`
- 零件原子化儲存（一個 location = 一行紀錄）
- BOM Main Item 為查詢聚合視圖（`GROUP BY supplier, supplier_pn, type`）
- Matrix Selection 使用 `group_key` 與 BOM Main Item 關聯
- 詳見 [DATABASE.md](DATABASE.md)

## 資料庫連線策略

- 應用程式啟動時不自動連接資料庫
- 使用者透過「開啟系列」功能選擇 `.bomix` 檔案
- 每個系列為獨立資料庫，同時只開啟一個
- 連線管理器（`src/main/database/connection.js`）負責開啟/關閉/切換資料庫

## 設定管理

- 應用程式設定（視窗大小、主題偏好、最近開啟的檔案等）存於使用者目錄
- 使用 JSON 格式儲存
- 設定檔路徑：`%APPDATA%/BOMIX/settings.json`

## 日誌系統

BOMIX 的日誌系統由 `useTaskStore`（Zustand）集中管理，分為兩種來源，統一顯示於底部狀態列與進度對話框。

### 雙來源設計

| 來源 | 寫入方式 | 說明 |
|------|----------|------|
| **Task Log** | 後端透過 IPC `task:update` 推送 | 與後端任務排程一對一綁定 |
| **System Log** | 前端主動呼叫 `addSystemLog()` | 記錄 UI 操作結果（如 Toast 同步來源） |

### 關鍵設計

- `lastGlobalLog`：`useTaskStore` 中的單一欄位，無論哪個來源寫入日誌皆會更新，讓底部狀態列**無需掃描所有 sessions** 即可即時反映最新訊息
- `system-logs` Session：固定 ID 的特殊 Session，儲存所有系統通知，上限為 `MAX_SYSTEM_LOGS = 100` 條（超過自動移除最舊的）
- 底部狀態列過濾 `status === 'SYSTEM'` 的 Session，避免系統通知佔據主任務顯示區

### 相關檔案

| 檔案 | 說明 |
|------|------|
| `src/renderer/stores/useTaskStore.js` | 日誌 Store（`addSystemLog`、`lastGlobalLog`） |
| `src/renderer/components/layout/AppStatusLine.jsx` | 底部狀態列，顯示最新日誌 |
| `src/renderer/components/dialogs/ProgressDialog.jsx` | 進度對話框，顯示完整歷史 |

詳細 API 說明請參閱 [LOG_SYSTEM.md](LOG_SYSTEM.md)
