# BOM 視圖 (Views) 與 匯出 (Exports) 定義指南

> 版本：1.2.0 | 最後更新：2026-02-15

本文檔說明 BOMIX 專案中 Excel 匯出模組的運作原理，以及如何定義 BOM 視圖與 Excel 匯出邏輯。相關程式碼位於 `src/main/services/bom-factory.service.js` 與 `src/main/services/excel-export/template-engine.js`。

## 1. 系統架構概述

目前的匯出系統採用 **Hybrid 策略**：
- **Engine**: `exceljs`
- **Pattern**: 樣板驅動 (Template-Driven) + 標籤映射 (Tag Mapping)
- **Configuration**: 視圖 (View) 與 匯出 (Export) 定義工廠 (`bom-factory.service.js`)

### 核心檔案
- **樣板**: `resources/templates/*.xlsx` (如 `ebom.xlsx`)
    - 定義了 Excel 的外觀（字體、邊框、顏色、Logo）。
    - 使用 `{{TAG}}` 標籤來佔位。
- **定義工廠**: `src/main/services/bom-factory.service.js`
    - 定義標準 View (篩選邏輯) 與 Export (匯出結構)。
- **引擎**: `src/main/services/excel-export/template-engine.js`
    - 負責讀取樣板、掃描標籤、填入資料、處理斑馬紋。
- **服務**: `src/main/services/export.service.js`
    - 負責 orchestrate 整個流程：取得 Export 定義 -> 執行 View -> 呼叫引擎產生 Excel。

---

## 2. 標準視圖 (Standard Views)

View (視圖) 定義了如何從原始 BOM 資料中篩選與聚合零件。所有 View 定義於 `src/main/services/bom-factory.service.js` 的 `VIEWS` 物件中。

### 2.1 View 結構
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

---

## 3. 匯出定義 (Export Definitions)

Export Definition 定義了 Excel 匯出的結構，包含使用的樣板檔案與 Sheet 的對應關係。所有 Export 定義於 `src/main/services/bom-factory.service.js` 的 `EXPORTS` 物件中。

### 3.1 Export 結構

每個 Sheet 可以獨立定義使用的 Template File。

```javascript
{
    id: 'export_id',
    sheets: [
        {
            targetSheetName: 'SMD Report',  // 輸出 Excel 的 Sheet 名稱
            templateFile: 'ebom.xlsx',      // 樣板檔名 (resources/templates/ 下)
            sourceSheetName: 'SMD',         // 樣板中的 Sheet 名稱
            viewId: VIEW_IDS.SMD            // 使用的 View ID (資料來源)
        },
        // ...
    ]
}
```

### 3.2 樣板 Sheet 來源的回退機制 (Fallback Logic)
當 `export.service.js` 嘗試從 `templateFile` 讀取 `sourceSheetName` 指定的 Sheet 時，若找不到該名稱的 Sheet，系統會執行以下回退邏輯：

1.  **優先嘗試**：尋找名稱完全相符的 Sheet。
2.  **回退 (Fallback)**：若找不到指定名稱，且該樣板 Workbook 中有至少一個 Sheet，則**預設使用第一個 Sheet**。
    - 此機制適用於許多單 Sheet 的樣板檔案，無論樣板內的 Sheet 名稱是 "Sheet1" 或其他名稱，只要指定該檔案，系統就能正確讀取。
3.  **失敗**：若樣板檔案為空或讀取失敗，則跳過該 Sheet 的匯出並記錄警告。

---

## 4. 樣板標籤 (Template Tags)

系統使用 `{{TAG}}` 標籤在 Excel 樣板中佔位。

### 4.1 Meta Tags (單一值)
用於表頭資訊，放置在樣板任意位置。

| Tag | 描述 | 範例 |
| :--- | :--- | :--- |
| `{{PROJECT_CODE}}` | 專案代碼 | `AG-2024` |
| `{{BOM_VERSION}}` | BOM 版本 | `V1.0-A` |
| `{{PHASE}}` | 階段 | `EVT` |
| `{{BOM_DATE}}` | 匯出日期 | `2024-02-14` |

### 4.2 Row Tags (列表值)
用於 BOM 列表。系統會尋找包含這些標籤的樣板列 (Template Row)，並根據資料筆數自動複製。

- **主料 (Main Item)**: 需以 `M_` 開頭，例如 `{{M_HHPN}}`, `{{M_DESC}}`, `{{M_QTY}}`。
- **替代料 (Second Source)**: 需以 `S_` 開頭，例如 `{{S_HHPN}}`, `{{S_DESC}}`。

### 4.3 Footer Tags (表尾統計)
可放置在資料列表下方的任意列。程式會自動將這些列推到資料的最下方。
- `{{TOTAL_QTY}}`: 自動產生 `SUM` 公式計算用量總和。

---

## 5. 維護操作指引

### 如何新增 View
1. 在 `src/main/services/bom-factory.service.js` 的 `VIEW_IDS` 常數中新增 ID。
2. 在 `VIEWS` 物件中新增定義。

### 如何新增或修改 Export Sheet
1. 修改 `src/main/services/bom-factory.service.js` 中的 `EXPORTS` 定義。
2. 在 `sheets` 陣列中新增物件，指定 `targetSheetName`, `templateFile`, `sourceSheetName`, 與 `viewId`。
3. 確保 `resources/templates/` 下存在對應的 `templateFile`。

### 如何修改 Excel 樣式
直接使用 Excel 編輯 `resources/templates/` 下的 `.xlsx` 檔案即可。
- 請勿刪除包含 `{{TAG}}` 的儲存格。
- 系統會自動識別含有 `{{M_...}}` 的列為主料樣式，含有 `{{S_...}}` 的列為替代料樣式。

### 斑馬紋邏輯 (Zebra Striping)
斑馬紋顏色定義在 `src/main/services/excel-export/template-engine.js` (`COLOR_WHITE`, `COLOR_GRAY`)。系統以 Group (主料 + 替代料) 為單位變色。
