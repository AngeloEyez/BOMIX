# Settings 模組 API 文件

## 概述

設定系統負責管理應用程式的使用者偏好（主題、色彩等），並提供持久化儲存功能。
所有設定的定義請參考 [`dev/SETTINGS_GUIDE.md`](../SETTINGS_GUIDE.md)。

## 涉及檔案

| 檔案 | 職責 |
|---|---|
| `src/renderer/config/settingsConfig.js` | 設定宣告（單一真實來源）|
| `src/renderer/stores/useSettingsStore.js` | Zustand Store，管理狀態與 actions |
| `src/renderer/pages/SettingsPage.jsx` | UI 頁面，從 settingsConfig 動態渲染 |

---

## settingsConfig.js API

### `SETTINGS_CONFIG`
```js
// 完整的設定項目定義陣列，供 Store 和 UI 使用
import { SETTINGS_CONFIG } from '../config/settingsConfig'
```

### `getDefaults()`
```js
import { getDefaults } from '../config/settingsConfig'

const defaults = getDefaults()
// 回傳: { theme: 'light', activeThemeId: 'default', startupPage: 'dashboard', ... }
```

### `getPersistKeys()`
```js
import { getPersistKeys } from '../config/settingsConfig'

const keys = getPersistKeys()
// 回傳: ['theme', 'activeThemeId', 'startupPage', ...]（persist: true 的 key 陣列）
```

### `buildSettingsTree()`
```js
import { buildSettingsTree } from '../config/settingsConfig'

const tree = buildSettingsTree()
// 回傳: [{ id, title, icon, items: [{ id, title, settings: [...] }] }]
```

---

## useSettingsStore State

```js
import useSettingsStore from '../stores/useSettingsStore'
```

| 屬性 | 型別 | 說明 |
|---|---|---|
| `theme` | `'light' \| 'dark'` | 目前的明暗主題 |
| `activeThemeId` | `string` | 目前套用的色彩主題 ID |
| `availableThemes` | `Array<{id, name}>` | 從後端取得的可用主題列表 |
| `bomSidebarWidth` | `number` | BOM 側邊欄寬度（px）|
| `isBomSidebarCollapsed` | `boolean` | BOM 側邊欄是否收合 |
| `isLoading` | `boolean` | 設定是否正在初始化 |

> 其他由 `settingsConfig.js` 定義的設定 key（如 `startupPage`）也可直接從 Store 讀取。

---

## useSettingsStore Actions

### `loadSettings()` / `initSettings()`
```js
// 從後端載入設定，通常在 AppLayout 初始化時呼叫
const { loadSettings } = useSettingsStore()
await loadSettings()
```

### `toggleTheme()`
```js
// 切換 Light/Dark 模式並儲存
const { toggleTheme } = useSettingsStore()
await toggleTheme()
```

### `setThemeId(themeId)`
```js
// 切換配色主題並儲存
const { setThemeId } = useSettingsStore()
await setThemeId('ocean-blue')
```

### `updateSettings(newSettings)`
```js
// 更新任意設定並儲存（通用）
const { updateSettings } = useSettingsStore()
await updateSettings({ bomSidebarWidth: 280 })
```

### `resetToDefaults()`
```js
// 還原所有設定至 settingsConfig.js 定義的預設值，並重新套用主題效果
const { resetToDefaults } = useSettingsStore()
await resetToDefaults()
```

### `saveSettings()`
```js
// 手動觸發持久化（通常由其他 action 自動呼叫，不需要手動呼叫）
const { saveSettings } = useSettingsStore()
await saveSettings()
```

---

## 在元件中讀取設定

```js
// 方式 1：訂閱單一欄位（效能最佳）
function MyComponent() {
    const theme = useSettingsStore(state => state.theme)
    const activeThemeId = useSettingsStore(state => state.activeThemeId)
}

// 方式 2：取出多個欄位
function MyComponent() {
    const { theme, activeThemeId, setThemeId } = useSettingsStore()
}
```

---

## 後端 IPC API

設定由 `window.api.settings` 提供：

| 方法 | 說明 |
|---|---|
| `window.api.settings.get()` | 讀取儲存的使用者設定 |
| `window.api.settings.save(payload)` | 儲存設定 |

`save` 的 payload 格式（`themeId` 為 `activeThemeId` 的別名，用於後端相容）：
```js
{
    theme: 'dark',
    themeId: 'default',         // 對應 Store 的 activeThemeId
    startupPage: 'dashboard',
    bomSidebarWidth: 250,
    isBomSidebarCollapsed: false,
}
```
