# Project API

## `window.api.project.create(projectCode, description)`
在目前的系列中建立新專案。

- **參數**
  - `projectCode` (string) — 專案代碼 (必須唯一)
  - `description` (string) — 專案描述 (可選)
- **回傳** `{ success: true, data: { id, project_code, description, created_at, updated_at } }`
- **錯誤** `{ success: false, error: '專案代碼已存在' }`
- **範例**
  ```javascript
  const result = await window.api.project.create('TANGLED', 'Tangled Project Phase 1');
  ```

## `window.api.project.getAll()`
取得目前系列中的所有專案列表。

- **參數** 無
- **回傳** `{ success: true, data: [ { id, project_code, description, ... }, ... ] }`
- **錯誤** `{ success: false, error: '錯誤訊息' }`

## `window.api.project.getById(id)`
根據 ID 取得特定專案資訊。

- **參數**
  - `id` (number) — 專案 ID
- **回傳** `{ success: true, data: { id, project_code, description, ... } }`
- **錯誤** `{ success: false, error: '專案不存在' }`

## `window.api.project.update(id, data)`
更新專案資訊 (代碼或描述)。

- **參數**
  - `id` (number) — 專案 ID
  - `data` (Object) — 更新內容
    - `project_code` (string, optional) — 新的專案代碼 (需唯一)
    - `description` (string, optional) — 新的專案描述
- **回傳** `{ success: true, data: { id, project_code, description, ... } }`
- **錯誤** `{ success: false, error: '專案代碼已存在' }`

## `window.api.project.delete(id)`
刪除專案 (包含其下所有 BOM 版本)。

- **參數**
  - `id` (number) — 專案 ID
- **回傳** `{ success: true, data: { success: true } }`
- **錯誤** `{ success: false, error: '錯誤訊息' }`
