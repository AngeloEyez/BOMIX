# Series API

## `window.api.series.create(filePath, description)`
建立新的系列資料庫檔案 (.bomix)。

- **參數**
  - `filePath` (string) — .bomix 檔案完整路徑 (需包含檔名與副檔名)
  - `description` (string) — 系列描述 (可選)
- **回傳** `{ success: true, data: { id, description, created_at, updated_at } }`
- **錯誤** `{ success: false, error: '錯誤訊息' }`
- **範例**
  ```javascript
  const result = await window.api.series.create('C:/BOMIX/MySeries.bomix', '我的主機板系列');
  if (result.success) {
    console.log('系列建立成功', result.data);
  }
  ```

## `window.api.series.open(filePath)`
開啟現有的系列資料庫檔案。

- **參數**
  - `filePath` (string) — .bomix 檔案完整路徑
- **回傳** `{ success: true, data: { id, description, created_at, updated_at } }`
- **錯誤** `{ success: false, error: '錯誤訊息' }`

## `window.api.series.getMeta()`
取得目前開啟系列的元資料。

- **參數** 無
- **回傳** `{ success: true, data: { id, description, created_at, updated_at } }`
- **錯誤** `{ success: false, error: '尚未開啟系列' }`

## `window.api.series.updateMeta(description)`
更新目前系列的描述。

- **參數**
  - `description` (string) — 新的描述
- **回傳** `{ success: true, data: { id, description, created_at, updated_at } }`
- **錯誤** `{ success: false, error: '錯誤訊息' }`
