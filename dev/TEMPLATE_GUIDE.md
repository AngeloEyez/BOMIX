# Excel 樣板與匯出維護指南

本文件說明 BOMIX 專案中 Excel 匯出模組的運作原理，以及如何進行維護（例如：修改樣式、新增欄位）。

## 1. 架構概述

目前的匯出系統採用 **Hybrid 策略**：
- **Engine**: `exceljs`
- **Pattern**: 樣板驅動 (Template-Driven) + 標籤映射 (Tag Mapping)

### 核心檔案
- **樣板**: `resources/templates/ebom_template.xlsx`
    - 定義了 Excel 的外觀（字體、邊框、顏色、Logo）。
    - 使用 `{{TAG}}` 標籤來佔位。
- **引擎**: `src/main/services/excel-export/template-engine.js`
    - 負責讀取樣板、掃描標籤、填入資料、處理斑馬紋。
- **服務**: `src/main/services/export.service.js`
    - 負責從資料庫撈取 BOM 資料。
    - 將資料轉換為符合標籤 (Tag) 結構的物件。

---

## 2. 樣板標籤 (Tags)

### Meta Tags (單一值)
用於表頭資訊，放置在樣板任意位置。

| Tag | 描述 | 範例 |
| :--- | :--- | :--- |
| `{{PROJECT_CODE}}` | 專案代碼 | `AG-2024` |
| `{{BOM_VERSION}}` | BOM 版本 (原 VERSION) | `V1.0-A` |
| `{{PHASE}}` | 階段 | `EVT` |
| `{{BOM_DATE}}` | 匯出日期 (原 DATE) | `2024-02-14` |
| `{{SCH_VERSION}}` | Schematic 版本 | `V1` |
| `{{PCB_VERSION}}` | PCB 版本 | `V2` |
| `{{PCA_PN}}` | PCA P/N | `PN-12345` |
| `{{DESCRIPTION}}` | 描述 | `Motherboard` |

### Row Tags (列表值)
用於 BOM 列表。系統會尋找包含這些標籤的樣板列 (Template Row)，並根據資料筆數自動複製。

#### 主料 (Main Item) - 需以 `M_` 開頭
| Tag | 描述 |
| :--- | :--- |
| `{{M_ITEM}}` | 項目編號 (1, 2, 3...) |
| `{{M_HHPN}}` | HH 料號 |
| `{{M_DESC}}` | 描述 |
| `{{M_SUPP}}` | 供應商 |
| `{{M_SPN}}` | 供應商料號 |
| `{{M_QTY}}` | 用量 |
| `{{M_LOC}}` | 位置 (Locations) |
| `{{M_CCL}}` | CCL (Critical Component) |
| `{{M_REMARK}}` | 備註 |

#### 替代料 (Second Source) - 需以 `S_` 開頭
| Tag | 描述 |
| :--- | :--- |
| `{{S_HHPN}}` | HH 料號 |
| `{{S_DESC}}` | 描述 |
| `{{S_SUPP}}` | 供應商 |
| `{{S_SPN}}` | 供應商料號 |

---

## 3. 維護操作指引

### 情境 A：修改樣式 (字體、顏色、邊框)
**不需要修改程式碼。**

1.  前往 `resources/templates/`。
2.  使用 Excel 開啟 `ebom_template.xlsx`。
3.  直接修改儲存格的顏色、字體、邊框。
4.  存檔即可。

> **注意**：
> - 請勿刪除包含 `{{TAG}}` 的儲存格，除非您確定要移除該欄位。
> - 系統會自動讀取含有 `{{M_...}}` 的那一列作為「主料樣式」。
> - 系統會自動讀取含有 `{{S_...}}` 的那一列作為「替代料樣式」。

### 情境 B：新增一個欄位 (例如：Lead Time)

此操作涉及 **樣板** 與 **程式碼** 兩邊的修改。

#### Step 1: 修改樣板
1.  開啟 `ebom_template.xlsx`。
2.  在表頭 (Header) 插入一欄，標題寫 "Lead Time"。
3.  在主料樣板列 (通常是隱藏的或在表頭下方)，對應格子填入新標籤 `{{M_LTIME}}`。
4.  存檔。

#### Step 2: 修改程式碼
1.  開啟 `src/main/services/export.service.js`。
2.  找到 `items` 的 map 邏輯。
3.  在 `mainTags` 物件中加入新欄位：

```javascript
const mainTags = {
    // ... 原有欄位
    M_REMARK: item.remark,
    M_LTIME: '', // 新增欄位 (需確認 bomView 是否有此資料)
};
```
> 如果 `bomView` 資料來源 (`bom.service.js` -> `parts.repo.js`) 沒有這個欄位，您可能還需要去 `parts.repo.js` 的 SQL 查詢中加入該欄位。

---

## 4. 斑馬紋邏輯 (Zebra Striping)

斑馬紋的顏色定義在 `src/main/services/excel-export/template-engine.js` 中：

```javascript
const COLOR_WHITE = 'FFFFFFFF';
const COLOR_GRAY = 'FFF2F2F2';
```

若需修改顏色，請直接調整此處變數。
系統是以 **Group (主料 + 其替代料)** 為單位進行變色，確保同一組料的背景色一致。
