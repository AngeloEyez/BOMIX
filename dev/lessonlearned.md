# 避坑指南與經驗總結 (Lessons Learned)

本文件用於紀錄開發過程中遇到的系統性錯誤、踩過的坑，以及對應的解決方案。
AI Agent 在啟動任務時必須優先閱讀此文件，並在解決新的系統級問題後主動更新此文件，以避免未來重複嘗試錯誤的解法，減少 Token 浪費。

## 紀錄規則

當遇到非單一事件、未來可能重複發生的問題時，請依以下格式記錄：
### [問題簡述或相關模組]
- **問題描述**: 遇到什麼錯誤或非預期行為。
- **錯誤原因**: 為什麼會發生此問題。
- **解決方案**: 最終有效的正確解法。
- **避免方式**: 日後開發時應如何避開此問題。

---

## 已知問題紀錄

### [主行程] F12 開發者工具無法顯示 (僅顯示終端機訊息)
- **問題描述**: 按下 F12 時，終端機顯示 "Open dev tool..." 但畫面上沒有出現 DevTools 視窗。
- **錯誤原因**: `@electron-toolkit/utils` 的 `optimizer.watchWindowShortcuts` 將 DevTools 模式硬編碼為 `undocked`。在 Windows 下，這常導致視窗位置紀錄錯誤而彈出在螢幕可視範圍外（Off-screen）。另外，Electron 40 有已知 Bug 導致 DevTools bundled 頁面在 BrowserWindow 內無法正常載入（見下條）。
- **解決方案**: 在 `src/main/index.js` 中手動攔截 `before-input-event` 的 F12 事件，並使用 `mainWindow.webContents.openDevTools({ mode: 'detach' })`。`detach` 模式會開啟獨立的新視窗且通常會出現在螢幕正中央。
- **避免方式**: 不要完全依賴底層工具套件的預設快速鍵行為，對於 UI/視窗相關的關鍵操作應保留自定義彈性。

---

### [主行程] GPU Process error_code=18 導致應用程式無法啟動
- **問題描述**: Electron 應用程式在某些 Windows 環境下啟動即崩潰，錯誤訊息為「GPU process isn't usable. Goodbye.」或 `error_code=18`。
- **錯誤原因**: 在從網路磁碟執行、或特定 Windows 安全設定下，Chromium 的 GPU Process Sandbox 機制會被系統阻擋。
- **解決方案**: 在 `app.whenReady()` 之前加上 `app.commandLine.appendSwitch('disable-gpu-sandbox')`，僅停用 GPU 沙箱而保留 GPU 功能。
- **避免方式**: ⚠️ **絕對禁止**同時加入 `--disable-gpu` 或 `--disable-software-rasterizer`。這兩個旗標會完全停用 Chromium 的渲染管線，導致包含 DevTools 在內的所有彈出視窗都因為「沒有渲染器可用」而無法繪製畫面（只有 log 輸出，沒有視窗顯示）。

---

### [主行程] Electron 40 DevTools 在 BrowserWindow 內白屏 / 無法開啟
- **問題描述**: Electron 40.x 版本中，使用 `openDevTools()` 或 `openDevTools({ mode: 'bottom' })` 時，DevTools 視窗出現白屏，或完全無法在視窗內嵌入。
- **錯誤原因**: Electron 40 有一個已知 Bug，`devtools://devtools/bundled/devtools_app.html` 在 BrowserWindow 內的嵌入模式（bottom/right/left）下無法正常載入。
- **解決方案**: 必須使用 `openDevTools({ mode: 'detach' })` 強制以獨立視窗開啟 DevTools，才能繞過此 Bug。
- **避免方式**: 在所有呼叫 `openDevTools()` 的地方（包含自動開啟與 F12 快速鍵觸發）都必須傳入 `{ mode: 'detach' }`，直到 Electron 發布修補版本為止。

---

### [資料庫/Repository] 呼叫未定義或未匯出的 Repository 方法
- **問題描述**: 執行匯入任務時，主行程報錯 `TypeError: projectRepo.findByCode is not a function`。
- **錯誤原因**: `import.service.js` 呼叫了 `projectRepo.findByCode()`，但 `project.repo.js` 雖然實作了方法卻忘記將其加入 `export default` 物件中。
- **解決方案**: 補齊 `project.repo.js` 的 `findByCode` 方法並確保匯出。
- **避免方式**: AI Agent 在修改 Repository 後，必須逐一檢查 `export default` 是否包含所有對外公開的方法。

---

### [渲染層] 匯入 Excel 後 UI 未自動更新
- **問題描述**: 匯入成功後 Dashboard 樹狀列表沒反應，需手動重整。
- **錯誤原因**: UI 僅監聽 `BATCH_IMPORT` (外殼任務)，未監聽真正執行匯入的 `IMPORT_BOM` (子任務)；且 `BomSidebar` 使用本地快取導致無法感知 `useProjectStore` 的變化。
- **解決方案**: 
  1. 在 `Dashboard` 與 `BomPage` 增加 `IMPORT_BOM` 任務完成監聽。
  2. 將 BOM 快取從 `BomSidebar` 本地 state 移至 `useProjectStore` 的 `allBoms`，實現單一事實來源 (Single Source of Truth)。
- **避免方式**: 避免在複雜元件內部維護重複的業務資料快取，應盡可能共用 Store。
