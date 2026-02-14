# BOM View 模組化與 Excel 匯出重構任務

## 任務目標
將 BOM 的檢視 (View) 與匯出 (Export) 邏輯模組化，以支援動態篩選與多 Sheet 匯出。
請根據以下規格進行實作。

## 1. 架構變更概述

我們需要兩個核心概念物件：
1.  **BomViewDefinition (視圖定義)**：定義 "資料要如何篩選與聚合"。例如 "SMD View" 只包含 Type=SMD 且 Status=I/P 的零件。
2.  **ExportDefinition (匯出定義)**：定義 "Excel 要如何產生"。例如 "E-BOM Export" 包含 5 個 Sheet，分別對應不同的 View，並使用 `ebom.xlsx` 樣板中的特定 Sheet 作為底版。

## 2. 實作細節

### A. 建立 `src/main/services/bom-factory.service.js`

如果不確定放哪，可以放在 `services` 目錄下。
此檔案負責產生標準的 View 和 Export 定義。

#### 2.1 View Definition 結構
```javascript
{
  id: string,          // e.g., 'smd_view'
  mode: string,        // 'NPI' | 'MP' | 'AUTO' (若為 AUTO 需在執行時解析為實際 Mode)
  filter: {
    types: string[],   // e.g., ['SMD']
    bom_statuses: string[], // e.g., ['I', 'P']
    ccl_only: boolean  // 只顯示 CCL
  }
}
```

#### 2.2 Export Definition 結構
```javascript
{
  id: string,          // e.g., 'ebom'
  templateFile: string,// e.g., 'ebom.xlsx' (放在 resources/templates/)
  sheets: [
    {
      targetSheetName: string, // e.g., 'SMD'
      sourceSheetName: string, // 樣板中的 Sheet 名稱，e.g., 'SMD'
      viewId: string           // 對應的 standard view id, e.g., 'smd_view'
    },
    // ...
  ]
}
```

### B. 重構 `src/main/services/bom.service.js`

目前 `getBomView` 是依靠 Repo 層寫死的 SQL。
請修改為 **In-Memory Aggregation**：

1.  **`getBomData(bomRevisionId)`**:
    - 呼叫 `partsRepo.findByBomRevision` 取得**所有**零件原子資料。
    - 呼叫 `secondSourceRepo.findByBomRevision` 取得**所有**替代料。
    - 回傳原始資料。

2.  **`executeView(bomRevisionId, viewDefinition)`**:
    - 步驟 1: 取得 BOM Revision (確認 Mode)。
    - 步驟 2: 解析 Filter (若 Mode=NPI，則 I/P 為有效狀態; 若 MP，則 I/M 為有效狀態)。
    - 步驟 3: 呼叫 `getBomData`。
    - 步驟 4: **Filter**: 過濾掉不符合 Type, Status, CCL 條件的零件。
    - 步驟 5: **Group**: 將零件依 `supplier + supplier_pn + type` 分組。
        - 計算 `quantity` (count location)。
        - 合併 `locations` (join string)。
        - 決定 Main Item 屬性 (通常取第一筆)。
    - 步驟 6: **Attach Second Sources**: 將替代料掛載到經篩選的 Main Items 上。
    - 回傳最終 View Data。

### C. 重構 `src/main/services/excel-export/template-engine.js`

目前的引擎只支援單一 Sheet 處理。請擴充為支持從樣板 "複製 Sheet" 到新檔案。

1.  **`createWorkbook()`**: 回傳 `new ExcelJS.Workbook()`。
2.  **`loadTemplate(templateName)`**: 讀取樣板檔案並回傳 Workbook 物件 (可做快取)。
3.  **`appendSheetFromTemplate(targetWorkbook, templateWorkbook, sourceSheetName, targetSheetName, data)`**:
    - 從 `templateWorkbook` 找到 `sourceSheetName`。
    - 在 `targetWorkbook` 新增一個 Sheet (複製樣式、欄寬、合併儲存格等)。
        - *注意*: ExcelJS 的 Sheet Copy 可能不完美，若實作困難，可考慮：
        - 方案 B: 讀取 Template Workbook 為基底，然後刪除不需要的 Sheets，或複製並更名現有 Sheets。
        - **建議方案**: 每次 Export 都讀取 Template Workbook 作為起點 (`outputWorkbook`)。
        - 對於每個需要的 Sheet：
            - 如果 Template 中已有同名 Sheet (e.g. 'SMD')，則直接使用並填寫。
            - 如果需要多個相同格式的 Sheet (e.g. 動態產生的 View)，則複製 Template Sheet。
    - 執行 `fillRow` 邏輯填入資料。
4.  **`saveWorkbook(targetWorkbook, outputPath)`**: 存檔。

### D. 重構 `src/main/services/export.service.js`

1.  取得 `ExportDefinition` (預設 EBOM)。
2.  載入 Template Workbook。
3.  針對 `ExportDefinition.sheets` 每一項：
    - 取得 `viewDefinition`。
    - `bomService.executeView` 取得資料。
    - 呼叫 Template Engine 填寫資料到指定 Sheet。
4.  移除 Template 中未被使用的多餘 Sheets (如果有)。
5.  存檔。

## 3. 注意事項

- **模組化**：確保 View 的邏輯封裝在 `bom-factory` 或 `bom.service` 內部，不要散落在 IPC 層。
- **相容性**：暫時保留 `bom.service.js` 中舊的 `getBomView` 介面 (可內部轉呼叫新的邏輯)，以免破壞前端功能，直到前端完成對接。
- **效能**：In-Memory 處理幾千筆資料在 Node.js 中是非常快的，不用擔心效能問題。

## 4. 交付項目
1.  修改 `src/main/services/bom.service.js`
2.  建立 `src/main/services/bom-factory.service.js`
3.  修改 `src/main/services/excel-export/template-engine.js`
4.  修改 `src/main/services/export.service.js`
5.  確保單元測試通過 (需新增與 View 邏輯相關的測試)

### 5. 測試項目

Bom Service Tests (tests/unit/bom.service.test.js):

測試 executeView 搭配不同 Filters (e.g., SMD only, NPI mode)。
驗證聚合邏輯：同料號但不同 bom_status 是否正確分組（或分開）。
驗證 Second Source 是否正確掛載。
Export Service Tests (tests/unit/export.service.test.js):

模擬 BOM 資料，執行 
exportBom
。
檢查輸出的 Excel 是否包含多個 Sheet (ALL, SMD, PTH...)。
驗證特定 Sheet (如 NI) 的資料列是否正確 (應只包含 status=X)。
