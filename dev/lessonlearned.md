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

---

### [渲染層/Shadcn] react-resizable-panels v4 的 defaultSize/minSize/maxSize 單位變更 (Breaking Change)
- **問題描述**: 使用 Shadcn `<ResizablePanel>` 元件時，面板寬度無法正常設定（顯示為超小或超大），且 Panel 無法正確依比例分配空間。
- **錯誤原因**: `react-resizable-panels` 在 **v4.x** 有 Breaking Change：`defaultSize`、`minSize`、`maxSize` 傳入數字（如 `defaultSize={25}`）時，**v4 改為解讀成「像素 (px)」**，而非 v3 的「百分比 (%）」。導致 25 表示 25px 而非 25%。
- **解決方案**: 使用字串並明確加上 `%` 後綴：
  ```jsx
  // ❌ v3 語法（v4 錯誤！數字被解讀為 px）
  <ResizablePanel defaultSize={25} minSize={20} maxSize={40} />

  // ✅ v4 正確語法（字串帶 % 表示百分比）
  <ResizablePanel defaultSize="25%" minSize="20%" maxSize="40%" />
  ```
  ```
- **避免方式**: 使用 Resizable 元件時，**一律使用帶 % 的字串**。在閱讀舊範例或 Shadcn 文件時，核對當前安裝的 `react-resizable-panels` 版本 (`npm list react-resizable-panels`)。

---

### [渲染層/Shadcn] react-resizable-panels v4 的持久化儲存 (autoSaveId 棄用與保留雙擊還原)
- **問題描述**: 
  1. 在 `<ResizablePanelGroup>` 加上 `autoSaveId="..."` 會在 Console 中跳出 `React does not recognize the autoSaveId prop on a DOM element` 警告。
  2. 如果自己手動用 `localStorage` 讀出寬度並直接灌給各別 `<ResizablePanel defaultSize={...}>`，會導致使用者雙擊拖曳把手（`ResizableHandle`）時，無法還原回原始設計的比例，而是卡在某個被儲存的奇怪大小。
- **錯誤原因**: `react-resizable-panels` 升級至 v4 後，正式棄用了元件內建的 `autoSaveId` 魔法屬性，改為要求開發者明確管理佈局狀態。此外，直接覆寫各別 `<ResizablePanel>` 的 `defaultSize` 等同於覆寫了「雙擊還原的基準值」。
- **解決方案**: 必須採用「把手動記憶套用在父層」的官方最佳實踐：
  1. 使用 `<ResizablePanelGroup onLayoutChanged={(sizes) => localStorage.setItem('my-layout', JSON.stringify(sizes))} defaultLayout={loadedLayoutArray}>`。
  2. 每一塊 `<ResizablePanel>` **必須明確加上獨立的 `id` 屬性**（如 `id="sidebar"`）。
  3. 各塊 `<ResizablePanel>` 的 `defaultSize` 必須保持為最初設計時的固定百分比（如 `"15%"`），**不可**吃 `localStorage` 的變數。
  這會讓 Group 負責重現上一次拖曳的位置，而 Panel 負責記住「如果使用者雙擊重設，我應該回到多少」。
- **避免方式**: 在實作可記憶板塊寬度的佈局時，絕對不要動態改變 `<ResizablePanel defaultSize>`。統一在外層 `<ResizablePanelGroup>` 用 `defaultLayout` 來達成。


### [渲染層] ScrollArea (shadcn) 內容高度超出容器但不出現卷軸
- **問題描述**: 使用 `<ScrollArea>` 包裹長內容時，內容溢出但垂直捲軸不出現，頁面無法滾動。
- **錯誤原因**: Shadcn 的 `ScrollArea` 元件依賴 Radix UI 實作，需要知道固定的父容器高度才能計算溢出量。若父容器未設定明確高度（例如 `flex-1` 沒有搭配正確的 flex 容器鏈），`ScrollArea` 無法正確判斷內容是否溢出。
- **解決方案**: 改用原生 `div` 搭配 `overflow-y-auto` 取代 `ScrollArea`，並確保父容器具有 `overflow-hidden` 和正確的 `flex flex-col` 鏈：
  ```jsx
  // ❌ ScrollArea 可能不顯示卷軸
  <div className="flex-1">
      <ScrollArea className="h-full">...</ScrollArea>
  </div>

  // ✅ 改用原生滾動，父容器 overflow-hidden 確保子容器尺寸有界
  <div className="flex-1 overflow-hidden flex flex-col">
      <div className="flex-1 overflow-y-auto">...</div>
  </div>
  ```
- **避免方式**: 對於全頁面高度的滾動內容，優先考慮 `overflow-y-auto` 原生滾動。`ScrollArea` 適合用在已知固定高度的容器內（如側欄、模態框內的列表）。

---

### [渲染層/流程] 批次處理與對話框的競態條件 (Race Condition)
- **問題描述**: 匯入 Excel 檔案時，如果檔案名稱包含未定義的 Phase，程式需要暫停並跳出對話框請使用者定義，定義完後要自動接續匯入流程。若未處理好 Promise 或是狀態更新時機，可能導致匯入中斷或對話框閃退。
- **錯誤原因**: 匯入邏輯若放在單一的同步或非同步函數中，一旦遇到需要使用者互動 (UI 中斷) 的步驟，傳統函數無法輕易暫停等待 React State (對話框) 的回呼。
- **解決方案**:
  1. 將「分析與驗證」抽離為獨立的 API (`excel:analyzeFiles`)，在送出匯入排程前先在前端執行。
  2. 若驗證失敗，阻斷原始的匯入流程，將狀態與未完成的路徑 (`pendingPaths`) 存入 Component State，並開啟對話框。
  3. 對話框的 `onSave` 回呼中，處理完定義儲存後，檢查 `pendingPaths`，若有則重新呼叫原先的 `handleImportSubmit`。
- **避免方式**: 當業務邏輯 (如匯入、儲存) 中途需要提示使用者輸入時，應將流程切分為「驗證 -> 阻斷並開啟 UI -> UI 提交後重試/接續執行」的模式，不要試圖用 `while` 或堵斷 UI thread 的方式等待。
