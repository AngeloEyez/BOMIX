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
      bomRevisionId: 1
    }
  }
  ```
- **邏輯**
  - 解析 Excel 標頭 (Product Code, Description, Date 等)
  - 根據 SMD, PTH, BOTTOM 等 Sheet 讀取零件
  - 根據 PROTO, MP Sheet 判斷 NPI/MP Mode
  - 自動處理 Location 原子化與 Main/2nd Source 判斷

## `window.api.excel.export(bomRevisionId, outputFilePath)`
將 BOM 匯出為 Excel 檔案。

- **參數**
  - `bomRevisionId` (number) — BOM 版本 ID
  - `outputFilePath` (string) — 輸出檔案完整路徑
- **回傳** `{ success: true, data: { success: true } }`
- **邏輯**
  - 根據 NPI/MP Mode 自動過濾零件 (如 NPI 不顯示 MP only parts)
  - 產生 ALL, SMD, PTH, BOTTOM, NI, PROTO, MP, CCL 等多個 Sheet
  - 包含完整表頭與基本格式
