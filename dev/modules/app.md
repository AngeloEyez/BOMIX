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
