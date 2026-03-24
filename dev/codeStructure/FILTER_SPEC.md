# BOMIX Filter 規格書

> 版本：1.0.0 | 最後更新：2026-03-10

本文件定義 `queryBomData` API 的 Filter 陣列格式、Operator 規則、statusLogic 特殊處理以及 Options 物件的用法。

---

## 1. 整體呼叫結構

前端透過 `window.api.bom.query(bomIds, filters, options)` 呼叫後端通用查詢 API：

```js
const bomIds  = [1, 2, 3];          // BOM Revision ID 陣列
const filters = [ /* Filter 物件陣列，見下方 */ ];
const options = { /* 選項物件，見下方 */ };

const result = await window.api.bom.query(bomIds, filters, options);
// 回傳 { success: true, data: AggregatedItem[] }
```

### 對應的前端 Store action

```js
// useBomStore.js 中的 fetchData / selectView 會自動組裝 filters
// 一般情況下不需手動呼叫 window.api.bom.query
```

---

## 2. Filter 物件格式

每一個 Filter 條件為一個物件：

```ts
interface Filter {
    field:    string;    // 過濾的欄位名稱（對應 parts 資料表）
    operator: string;    // 過濾邏輯（見支援列表）
    value:    any;       // 過濾值（格式依 operator 而定）
}
```

**filters 為陣列**，多個條件之間為 **AND 關係**：

```js
const filters = [
    { field: 'bom_status', operator: 'statusLogic', value: 'ACTIVE' },
    { field: 'type',       operator: 'in',          value: ['SMD']  },
    { field: 'ccl',        operator: 'eq',           value: 'Y'     }
];
// 語意：ACTIVE 狀態 AND type 在 SMD 內 AND ccl = Y
```

---

## 3. 支援的 Operator

| operator | 說明 | value 型別 | 範例 |
|----------|------|-----------|------|
| `eq` | 等於 | `string` | `{ field: 'ccl', operator: 'eq', value: 'Y' }` |
| `neq` | 不等於 | `string` | `{ field: 'ccl', operator: 'neq', value: 'N' }` |
| `in` | 包含於列表中 | `string[]` | `{ field: 'type', operator: 'in', value: ['SMD', 'PTH'] }` |
| `notIn` | 不在列表中 | `string[]` | `{ field: 'bom_status', operator: 'notIn', value: ['X'] }` |
| `statusLogic` | 特殊：依 BOM mode 決定狀態 | `'ACTIVE' \| 'INACTIVE' \| 'SPECIFIC'` | `{ field: 'bom_status', operator: 'statusLogic', value: 'ACTIVE' }` |

---

## 4. statusLogic 特殊 Operator

`statusLogic` 需要結合 BOM Revision 的 `mode`（NPI 或 MP）來動態決定允許的 `bom_status` 值。

| value | NPI Mode 允許 | MP Mode 允許 |
|-------|-------------|------------|
| `ACTIVE` | `['I', 'P']` | `['I', 'M']` |
| `INACTIVE` | `['X', 'M']` | `['X', 'P']` |
| `SPECIFIC` | 由同陣列中的 `in` filter 指定 | 同左 |

### 範例：ACTIVE filter（前端 View 的常見用法）

```js
// ALL View —— 顯示所有 Active 零件（依 Mode 不同）
const filters = [
    { field: 'bom_status', operator: 'statusLogic', value: 'ACTIVE' }
];
```

### 範例：SPECIFIC filter（指定特定 status）

```js
// 只顯示 bom_status = 'P' 的零件（PROTO View）
const filters = [
    { field: 'bom_status', operator: 'statusLogic', value: 'SPECIFIC' },
    { field: 'bom_status', operator: 'in', value: ['P'] }
];
```

---

## 5. View 與 Filters 的對應關係

View 是一組預設 filters 的快捷方式。`bom-factory.service.js` 的每個 View 定義包含 `filters` 陣列：

| View | filters |
|------|---------|
| ALL | `[{ statusLogic: 'ACTIVE' }]` |
| SMD | `[{ type in SMD }, { statusLogic: 'ACTIVE' }]` |
| PTH | `[{ type in PTH }, { statusLogic: 'ACTIVE' }]` |
| BOTTOM | `[{ type in BOTTOM }, { statusLogic: 'ACTIVE' }]` |
| NI | `[{ statusLogic: 'INACTIVE' }]` |
| PROTO | `[{ statusLogic: 'SPECIFIC' }, { bom_status in P }]` |
| MP | `[{ statusLogic: 'SPECIFIC' }, { bom_status in M }]` |
| CCL | `[{ ccl eq Y }, { statusLogic: 'ACTIVE' }]` |

前端組合 View + 額外條件的流程：

```js
// 1. 取得 View 的預設 filters
const viewFilters = views[currentViewKey].filters; // 從 bom:get-views 取回的定義

// 2. 加入使用者的額外條件（例如 CCL checkbox 勾選）
const extraFilters = cclFilter ? [{ field: 'ccl', operator: 'eq', value: 'Y' }] : [];

// 3. 合併為最終 filters 陣列
const finalFilters = [...viewFilters, ...extraFilters];

// 4. 送入 API
await window.api.bom.query(bomIds, finalFilters, options);
```

---

## 6. Options 物件

選項參數以物件形式傳遞，目前為預留路徑，功能尚未實作：

```js
const options = {
    merged: true,   // 預留：是否合併相同料號（尚未實作）
    // 未來可擴充其他選項
};
```

若無特殊需求，傳入空物件 `{}` 即可。

---

## 7. 回傳格式

與現有 `bom:getView` 完全相同，回傳聚合後的 Main Item 陣列：

```ts
interface AggregatedItem {
    id: number;
    bom_revision_id: number;
    bom_ids: number[];          // 此群組存在的所有 BOM IDs（用於 Union/Matrix 判斷）
    supplier: string;
    supplier_pn: string;
    hhpn: string;
    description: string;
    type: string;
    bom_status: string;
    ccl: string;
    remark: string;
    item: number;
    locations: string;          // 逗號分隔，例如 "C1,C2,C5"
    quantity: number;
    second_sources: SecondSource[];
}
```

---

## 8. 相關 IPC API

| IPC 通道 | 說明 |
|---------|------|
| `bom:query` | 通用查詢 (bomIds, filters, options) → AggregatedItem[] |
| `bom:getView` | **(已棄用)** 舊版查詢，僅保留向下相容 |
| `bom:get-views` | 取得所有 View 定義（含 filters 陣列） |

---

## 9. 後端實作對應

| 元件 | 函式 | 說明 |
|------|------|------|
| `bom.service.js` | `queryBomData(bomIds, filters, options)` | 核心通用查詢函式 |
| `bom.service.js` | `executeView(ids, viewDef)` | @deprecated，包裝 `queryBomData` |
| `bom.ipc.js` | `bom:query` handler | 呼叫 `queryBomData` |
| `bom-factory.service.js` | `getViewFilters(viewId)` | 回傳 View 的 filters 陣列 |
