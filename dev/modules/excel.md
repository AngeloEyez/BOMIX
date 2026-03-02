# Excel API

## `window.api.excel.import(filePath, projectId, phaseName, version)`
從 Excel 檔案匯入 BOM。

- **參數**
  - `filePath` (string) — Excel 檔案完整路徑
  - `projectId` (number) — 專案 ID
  - `phaseName` (string) — Phase 名稱 (如 "EVT")
  - `version` (string) — 版本號 (如 "0.1")
- **回傳**
  ```javascript
  {
    success: true,
    data: {
      taskId: 'uuid'
    }
  }
  ```
  *(註：此 API 為非同步排程，需透過 `window.api.task.onUpdate` 監聽進度，完成後會拋出 `task:completed` 事件。原有的同步回傳 `bomRevisionId` 行為已變更。)*

- **邏輯**
  - 解析 Excel 標頭 (Product Code, Description, Date 等)
  - 根據 SMD, PTH, BOTTOM 等 Sheet 讀取零件
  - 根據 PROTO, MP Sheet 判斷 NPI/MP Mode
  - 自動處理 Location 原子化與 Main/2nd Source 判斷
  - 以上步驟以 TaskManager 背景佇列執行，透過 `setImmediate` 防止阻塞 UI。

## `window.api.excel.export(bomRevisionId, outputFilePath)`
將 BOM 匯出為 Excel 檔案。

- **參數**
  - `bomRevisionId` (number) — BOM 版本 ID
  - `outputFilePath` (string) — 輸出檔案完整路徑
- **回傳** `{ success: true, data: { taskId: 'uuid' } }`
  *(註：此 API 將任務加入佇列，透過 `window.api.task` API 追蹤進度與完成狀態。)*

- **邏輯**
  - 根據 NPI/MP Mode 自動過濾零件 (如 NPI 不顯示 MP only parts)
  - 產生 ALL, SMD, PTH, BOTTOM, NI, PROTO, MP, CCL 等多個 Sheet
  - 包含完整表頭與基本格式
  - 以 TaskManager 背景執行，避免長時間匯出阻塞 UI
