# BOM 視圖 (Views) 與 匯出 (Exports) 定義指南

> 版本：1.0.0 | 最後更新：2026-02-15

本文檔說明如何在 BOMIX 中定義 BOM 視圖與 Excel 匯出邏輯。相關程式碼位於 `src/main/services/bom-factory.service.js`。

## 1. 標準視圖 (Standard Views)

View (視圖) 定義了如何從原始 BOM 資料中篩選與聚合零件。

### 1.1 定義位置
所有 View 定義於 `src/main/services/bom-factory.service.js` 的 `VIEWS` 物件中。

### 1.2 View 結構
每個 View 包含 `id` 與 `filter` 設定：

```javascript
{
    id: 'view_id',
    filter: {
        types: ['SMD', 'PTH'], // (選填) 製程類型白名單
        bom_statuses: ['P', 'M'], // (選填) 狀態白名單 (搭配 SPECIFIC 邏輯)
        statusLogic: 'ACTIVE' | 'INACTIVE' | 'SPECIFIC' | 'IGNORE', // 狀態篩選邏輯
        ccl: 'Y' // (選填) 是否只顯示 Critical Parts
    }
}
```

#### statusLogic 說明
- **ACTIVE**: 依據 BOM Mode (NPI/MP) 自動選擇有效的上件狀態。
  - NPI Mode: 顯示 `I` (Install) + `P` (Proto)
  - MP Mode: 顯示 `I` (Install) + `M` (MP Only)
- **INACTIVE**: 依據 BOM Mode (NPI/MP) 自動選擇不上件的狀態。
  - NPI Mode: 顯示 `X` (No Install) + `M` (MP Only - 因 NPI 不上)
  - MP Mode: 顯示 `X` (No Install) + `P` (Proto - 因 MP 不上)
- **SPECIFIC**: 僅顯示 `bom_statuses` 陣列中定義的狀態。
- **IGNORE**: 不篩選狀態（顯示所有）。

### 1.3 如何新增 View
1. 在 `VIEW_IDS` 常數中新增 ID：
   ```javascript
   export const VIEW_IDS = {
       // ...
       NEW_VIEW: 'new_view'
   };
   ```
2. 在 `VIEWS` 物件中新增定義：
   ```javascript
   [VIEW_IDS.NEW_VIEW]: {
       id: VIEW_IDS.NEW_VIEW,
       filter: { types: ['SMD'], statusLogic: 'IGNORE' }
   }
   ```

---

## 2. 匯出定義 (Export Definitions)

Export Definition 定義了 Excel 匯出的結構，包含使用的樣板檔案與 Sheet 的對應關係。

### 2.1 定義位置
所有 Export 定義於 `src/main/services/bom-factory.service.js` 的 `EXPORTS` 物件中。

### 2.2 Export 結構

```javascript
{
    id: 'export_id',
    templateFile: 'template.xlsx', // 位於 resources/templates/ 下的檔名
    sheets: [
        {
            targetSheetName: 'SMD Report', // 輸出 Excel 的 Sheet 名稱
            sourceSheetName: 'SMD',        // 樣板中的 Sheet 名稱
            viewId: 'smd_view'             // 使用的 View ID (資料來源)
        },
        // ...
    ]
}
```

### 2.3 如何新增或修改 Export
1. 若需新增 Export ID，請在 `EXPORT_IDS` 中定義。
2. 若需修改現有的 EBOM 匯出：
   - 修改 `EXPORTS[EXPORT_IDS.EBOM].sheets` 陣列。
   - 可以調整順序、新增 Sheet 或修改 View 對應。
3. **Template 注意事項**：
   - `templateFile` 必須存在於 `resources/templates/` (開發環境) 或打包後的資源目錄中。
   - `sourceSheetName` 必須存在於該樣板 Excel 中。
   - 樣板 Excel 需包含 Tag (如 `{{M_PN}}`, `{{S_PN}}`) 以供 Template Engine 填寫資料。

## 3. 程式呼叫方式

在 Service 層 (如 `export.service.js`) 中：

```javascript
import { getExportDefinition, EXPORT_IDS } from './bom-factory.service.js';

// 取得定義
const exportDef = getExportDefinition(EXPORT_IDS.EBOM);

// 迭代 Sheet 產生報表
for (const sheetDef of exportDef.sheets) {
    // ... 呼叫 bomService.executeView(bomId, getViewDefinition(sheetDef.viewId))
    // ... 呼叫 templateEngine.appendSheetFromTemplate(...)
}
```
