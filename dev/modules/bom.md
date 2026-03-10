# BOM API

## `window.api.bom.query(bomIds, filters, options)` ⭐ 新 API

通用 BOM 資料查詢。前端透過此方法傳入 BOM IDs、Filter 條件陣列與選項物件，後端執行過濾與聚合後回傳。

Filter 格式詳見 [`dev/FILTER_SPEC.md`](file:///d:/coding/BOMIX/dev/FILTER_SPEC.md)。

- **參數**
  - `bomIds` (number[]) — BOM 版本 ID 陣列（支援多 BOM Union 查詢）
  - `filters` (Object[]) — Filter 條件陣列，每個 Filter 物件格式：
    ```js
    { field: string, operator: string, value: any }
    // field: 資料欄位名稱 (bom_status / type / ccl 等)
    // operator: eq | neq | in | notIn | statusLogic
    // value: 過濾值
    ```
  - `options` (Object) — 預留選項物件（目前傳 `{}` 即可）
- **使用範例**
  ```js
  // ALL View + CCL=Y
  const filters = [
      { field: 'bom_status', operator: 'statusLogic', value: 'ACTIVE' },
      { field: 'ccl', operator: 'eq', value: 'Y' }
  ];
  const result = await window.api.bom.query([1, 2], filters, {});
  ```
- **回傳** — 同 `getView`，見下方格式

---

## ~~`window.api.bom.getView(bomRevisionId, viewId)`~~ （已棄用）

> **@deprecated** 前端請改用 `bom.query`。此方法保留供向下相容，待全面遷移後移除。

取得 BOM 聚合視圖 (Main Items + Second Sources)。

- **參數**
  - `bomRevisionId` (number) — BOM 版本 ID
- **回傳**
  ```javascript
  {
    success: true,
    data: [
      {
        bom_revision_id: 1,
        supplier: "Samsung",
        supplier_pn: "CL05...",
        type: "SMD",
        hhpn: "340...",
        description: "CAP...",
        bom_status: "I",
        locations: "C1,C2",
        quantity: 2,
        second_sources: [
          {
            main_supplier: "Samsung",
            main_supplier_pn: "CL05...",
            supplier: "Yageo",
            supplier_pn: "RC...",
            // ...
          }
        ]
      },
      // ...
    ]
  }
  ```
- **錯誤** `{ success: false, error: '錯誤訊息' }`

## `window.api.bom.updateMainItem(bomRevisionId, originalKey, updates)`
更新 Main Item (同時更新群組內所有零件)。

- **參數**
  - `bomRevisionId` (number)
  - `originalKey` (Object) — `{ supplier, supplier_pn, type }`
  - `updates` (Object) — `{ description: 'New Desc', remark: '...' }`
- **回傳** `{ success: true, data: { success: true } }`

## `window.api.bom.deleteMainItem(bomRevisionId, key)`
刪除 Main Item (同時刪除群組內所有零件與關聯的 Second Sources)。

- **參數**
  - `bomRevisionId` (number)
  - `key` (Object) — `{ supplier, supplier_pn, type }`
- **回傳** `{ success: true, data: { success: true } }`

## `window.api.bom.addSecondSource(data)`
新增 Second Source。

- **參數**
  - `data` (Object)
    - `bom_revision_id` (number)
    - `main_supplier` (string)
    - `main_supplier_pn` (string)
    - `supplier` (string)
    - `supplier_pn` (string)
    - `hhpn` (string, optional)
    - `description` (string, optional)
- **回傳** `{ success: true, data: { id, ... } }`

## `window.api.bom.updateSecondSource(id, data)`
更新 Second Source。

- **參數**
  - `id` (number)
  - `data` (Object) — `{ description: '...', ... }`
- **回傳** `{ success: true, data: { id, ... } }`

## `window.api.bom.deleteSecondSource(id)`
刪除 Second Source。

- **參數**
  - `id` (number)
- **回傳** `{ success: true, data: { success: true } }`

## `window.api.bom.delete(bomRevisionId)`
刪除整個 BOM 版本 (Revision)。

- **參數**
  - `bomRevisionId` (number)
- **回傳** `{ success: true, data: { success: true } }`

## `window.api.bom.updateRevision(id, updates)`
更新 BOM 版本屬性 (Metadata)。

- **參數**
  - `id` (number) — BOM 版本 ID
  - `updates` (Object) — 更新內容
    - `description` (string, optional)
    - `note` (string, optional)
    - `bom_date` (string, optional)
    - `mode` (string, optional) — 'NPI' or 'MP'
    - `suffix` (string, optional)
- **回傳** `{ success: true, data: { id, ...updatedFields } }`
- **錯誤** `{ success: false, error: '錯誤訊息' }`

