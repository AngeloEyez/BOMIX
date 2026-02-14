# BOM API

## `window.api.bom.getView(bomRevisionId)`
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
