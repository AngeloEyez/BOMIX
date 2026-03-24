# App & Settings API

## `window.api.getVersion()`
取得應用程式版本號。

- **參數**: 無
- **回傳**: (string) 版本號 (例如 "0.2.0")

---

## `window.api.app.getChangelog()`
讀取 `CHANGELOG.md` 內容。

- **參數**: 無
- **回傳**: `{ success: true, data: string }`
- **錯誤**: `{ success: false, error: string }`

---

## `window.api.settings.get()`
取得使用者設定。

- **參數**: 無
- **回傳**: `{ success: true, data: Object }` (例如 `{ theme: 'dark' }`)
- **錯誤**: `{ success: false, error: string }`

---

## `window.api.settings.save(settings)`
儲存使用者設定。

- **參數**: 
    - `settings` (Object) — 需要合併或儲存的設定對象
- **回傳**: `{ success: true }`
- **錯誤**: `{ success: false, error: string }`

---

## `window.api.dialog.showOpen(options)`
顯示檔案開啟對話框。

- **參數**: 
    - `options` (Object, 選填)
        - `title` (string) — 對話框標題
        - `filters` (Array) — 檔案篩選器，預設 `[{ name: 'BOMIX 系列檔', extensions: ['bomix'] }]`
- **回傳**: `{ success: true, data: string }` (選取的檔案路徑)
- **取消**: `{ success: true, canceled: true }`
- **錯誤**: `{ success: false, error: string }`

---

## `window.api.dialog.showSave(options)`
顯示檔案儲存對話框。

- **參數**: 
    - `options` (Object, 選填)
        - `title` (string) — 對話框標題
        - `defaultPath` (string) — 預設檔案名稱
        - `filters` (Array) — 檔案篩選器，預設 `[{ name: 'BOMIX 系列檔', extensions: ['bomix'] }]`
- **回傳**: `{ success: true, data: string }` (儲存的檔案路徑)
- **取消**: `{ success: true, canceled: true }`
- **錯誤**: `{ success: false, error: string }`
