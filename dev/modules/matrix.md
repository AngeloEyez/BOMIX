# Matrix BOM API

> 對應 Phase 7 開發項目

## `window.api.matrix.createModels(bomRevisionId, models)`
建立 Matrix Models (批次)。若未提供 `models`，預設建立 A, B, C 三個 Model。

- **參數**
  - `bomRevisionId` (number) - BOM 版本 ID
  - `models` (Array<{name, description}>, optional) - Model 定義列表
- **回傳** `{ success: true, data: Array<MatrixModel> }`
- **錯誤** `{ success: false, error: '...' }`

## `window.api.matrix.listModels(bomRevisionId)`
列出指定 BOM 的所有 Matrix Models。

- **參數**
  - `bomRevisionId` (number)
- **回傳** `{ success: true, data: Array<MatrixModel> }`

## `window.api.matrix.updateModel(id, updates)`
更新指定 Matrix Model。

- **參數**
  - `id` (number) - Model ID
  - `updates` (Object) - `{ name?: string, description?: string }`
- **回傳** `{ success: true, data: MatrixModel }`

## `window.api.matrix.deleteModel(id)`
刪除指定 Matrix Model。若該 Model 已有選擇紀錄，將拋出錯誤。

- **參數**
  - `id` (number)
- **回傳** `{ success: true, data: { success: true } }`

## `window.api.matrix.saveSelection(selectionData)`
儲存或更新使用者的選擇 (Upsert)。

- **參數**
  - `selectionData` (Object)
    - `matrix_model_id` (number)
    - `group_key` (string) - `supplier|supplier_pn`
    - `selected_type` (string) - `'part'` or `'second_source'`
    - `selected_id` (number) - 對應 parts.id 或 second_sources.id
- **回傳** `{ success: true, data: MatrixSelection }`

## `window.api.matrix.getData(bomRevisionId)`
取得 Matrix 完整資料，包含 Models、Selections (含隱式選擇) 與狀態摘要。

- **參數**
  - `bomRevisionId` (number)
- **回傳** `{ success: true, data: { models, selections, summary } }`
  - `models`: `Array<MatrixModel>`
  - `selections`: `Array<MatrixSelection>` (含 `is_implicit: true` 標記)
  - `summary`: `{ totalGroups, modelStatus: { [modelId]: { selectedCount, isComplete } }, isSafe }`

## `window.api.matrix.getSummary(bomRevisionId)`
僅取得 Matrix 狀態摘要 (供 Dashboard 顯示)。

- **參數**
  - `bomRevisionId` (number)
- **回傳** `{ success: true, data: summary }`
