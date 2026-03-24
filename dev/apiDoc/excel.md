# Excel API

## `window.api.excel.import(filePaths)`
從多個 Excel 檔案批次匯入 BOM。

- **參數**
  - `filePaths` (Array<string> | string) — Excel 檔案完整路徑陣列或單一字串。
- **回傳**
  ```javascript
  {
    success: true,
    data: {
      taskId: 'uuid' // 此為 Batch Task ID
    }
  }
  ```
  *(註：此 API 為非同步排程，需透過 `window.api.task.onUpdate` 監聽進度，完成後會拋出 `task:completed` 事件，事件結果帶有所有的解析與匯入結果。)*

- **邏輯**
  - 自動透過檔名提取 Project Name, Phase, Version 以及 BOM Type (BOM 或是 MatrixBOM)。
  - 解析檔名檢查專案與主 BOM 存在與否的相依關係。
  - 根據關聯性依序將合法檔案推入獨立的 `IMPORT_BOM` 子 Task 中。
  - 解析 Excel 標頭 (Project Code, Date 等) 並與檔名做交叉比對，若專案不存在會自動建立專案。
  - **零件處理細節**：
    - 優先從 `SMD`, `PTH`, `BOTTOM` 工作表讀取零件。
    - 根據 `PROTO`, `MP` 工作表是否存在特定 Location 零件自動判定 NPI/MP Mode。
    - 自動執行 Location 原子化（拆分 `C1-C5` 為獨立位號）並處理 Main/2nd Source 判斷。
  - 這些所有步驟都會以 TaskManager 背景佇列執行，透過 `setImmediate` 防止阻塞 UI。

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
