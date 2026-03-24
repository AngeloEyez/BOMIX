# BOMIX 日誌系統說明

> 版本：1.0.0 | 最後更新：2026-03-18

## 概述

BOMIX 的日誌系統由 `useTaskStore`（Zustand）集中管理，分為兩種來源：

| 來源 | 說明 |
|------|------|
| **Task Log**（任務日誌） | 由後端透過 IPC `task:update` 推送，與任務排程一對一綁定 |
| **System Log**（系統通知） | 由前端元件主動呼叫 `addSystemLog()` 寫入，記錄 UI 操作結果 |

兩種來源都會更新 `lastGlobalLog`，使底部狀態列即時反映最新一條訊息。

---

## Store 狀態欄位

位置：`src/renderer/stores/useTaskStore.js`

| 欄位 | 型別 | 說明 |
|------|------|------|
| `sessions` | `Map<string, Session>` | 所有任務（含歷史）。固定 ID `system-logs` 的 Session 用來儲存系統通知 |
| `lastGlobalLog` | `{ message, level, timestamp } \| null` | 最新一條日誌，任務日誌與系統通知皆會更新此欄位 |
| `queueLength` | `number` | 後端排隊等待的任務數 |

### 常數

| 常數 | 預設值 | 說明 |
|------|--------|------|
| `MAX_TASKS` | `30` | sessions Map 最多保留的任務數 |
| `SYSTEM_LOG_SESSION_ID` | `'system-logs'` | 系統通知 Session 的固定 ID |
| `MAX_SYSTEM_LOGS` | `100` | 系統通知 Session 最多保留的日誌條數，超過時自動移除最舊的 |

---

## API

### `addSystemLog(message, level?)`

新增一條系統通知日誌。

```js
// 在 React 元件或 Zustand action 中使用：
const addSystemLog = useTaskStore(state => state.addSystemLog)

addSystemLog('資料庫開啟成功', 'info')
addSystemLog('Phase 格式不符，已略過該檔案', 'warn')
addSystemLog('儲存失敗：連線逾時', 'error')
addSystemLog('匯入完成', 'success')
```

**參數：**

| 參數 | 型別 | 必填 | 說明 |
|------|------|------|------|
| `message` | `string` | ✅ | 日誌訊息內容 |
| `level` | `'info' \| 'warn' \| 'error' \| 'success'` | ❌（預設 `'info'`） | 嚴重程度 |

**效果：**
1. 將訊息寫入 `sessions.get('system-logs').logs`
2. 若日誌數超過 `MAX_SYSTEM_LOGS`（100 條），自動刪除最舊的一筆
3. 更新 `lastGlobalLog`，底部狀態列立即反映最新訊息

---

## 顯示流程

```
addSystemLog(msg, level)
        │
        ▼
useTaskStore.lastGlobalLog  ─────────────────── AppStatusLine（狀態列即時顯示）
        │
        ▼
sessions.get('system-logs').logs                 ProgressDialog（點開後查看歷史）
```

### `AppStatusLine.jsx` 的顯示邏輯

1. 優先顯示 `lastGlobalLog.message`（最新任何來源的日誌）
2. 若無全域日誌，退而顯示活躍任務的最後一條 log
3. 若無任何日誌，顯示 idle

### `ProgressDialog.jsx` 的顯示邏輯

- 「系統通知」Session（ID = `system-logs`）會出現在左側清單，`StatusBadge` 顯示為紫色 **SYSTEM** 標籤
- 右側詳細日誌面板與一般任務相同，支援時間戳記、等級顏色

---

## 日誌等級顏色對照

| `level` | Badge 顏色（ProgressDialog） |
|---------|------------------------------|
| `info` | 藍色 |
| `warn` | 黃色 |
| `error` | 紅色 |
| `success` | 以 `info` 顏色顯示（藍色）|

---

## 使用範例

### 在元件 handler 中記錄操作結果

```jsx
import useTaskStore from '@/stores/useTaskStore'

function MyComponent() {
    const addSystemLog = useTaskStore(state => state.addSystemLog)

    const handleSave = async () => {
        const result = await window.api.data.save(payload)
        if (result.success) {
            addSystemLog('資料儲存成功', 'success')
        } else {
            addSystemLog(`資料儲存失敗：${result.error}`, 'error')
        }
    }
}
```

### 不需要搭配 Toast 使用

`addSystemLog` 與 Toast 是完全獨立的系統。視情況：
- **僅需日誌記錄**（不需要彈出通知）→ 只呼叫 `addSystemLog`
- **同時需要彈出通知**（如錯誤提示需要使用者立即注意）→ 同時呼叫 `addSystemLog` + `addToast`

---

## 注意事項

- `system-logs` Session 不會被 `clearCompletedSessions()` 清除（其 `status` 為 `'SYSTEM'`，不在清除條件內）
- `AppStatusLine` 的 `activeSession` 計算時會過濾掉 `status === 'SYSTEM'` 的 Session，避免「系統通知」佔據主任務顯示區
- `lastGlobalLog` 在應用程式重整後會重置為 `null`（Store 為 memory-only）
