# BOMIX 設定系統維護指南

## 架構總覽

設定系統採用**單一真實來源（Single Source of Truth）**設計，所有設定的定義集中於一個宣告檔：

```
src/renderer/config/settingsConfig.js   ← 唯一維護點
        │
        ├── useSettingsStore.js          ← 自動建立初始值、持久化邏輯
        └── SettingsPage.jsx             ← 自動渲染 UI、目錄結構
```

**你只需修改 `settingsConfig.js`**，Store 和 SettingsPage 會自動反映變更。

---

## 目錄與子目錄

目錄結構**集中**定義在 `settingsConfig.js`，分兩層：

| 常數 | 說明 |
|---|---|
| `SETTINGS_CATEGORIES` | 第一層大分類（左側目錄主節點）|
| `SETTINGS_SUB_CATEGORIES` | 第二層子分類（左側目錄子節點）|

設定項目只引用 `subCategoryId`（字串），由 `buildSettingsTree()` 自動組合。

---

## 新增一個設定項目

### Step 1：確認分類是否存在

打開 `src/renderer/config/settingsConfig.js`，確認 `SETTINGS_CATEGORIES` 和 `SETTINGS_SUB_CATEGORIES` 中是否已有適合的分類。

若需要新增分類：
```js
// 新增大分類
export const SETTINGS_CATEGORIES = [
    // ...既有分類...
    { id: 'advanced', title: '進階', icon: Sliders },
]

// 新增子分類
export const SETTINGS_SUB_CATEGORIES = [
    // ...既有子分類...
    { id: 'advanced.export', categoryId: 'advanced', title: '匯出設定' },
]
```

### Step 2：在 `SETTINGS_CONFIG` 中加入設定項目

```js
export const SETTINGS_CONFIG = [
    // ...既有設定...
    {
        key: 'exportFormat',           // Store 中的狀態鍵名（唯一）
        default: 'xlsx',              // 預設值
        title: '匯出格式',             // UI 標題
        description: '預設的 BOM 匯出檔案格式。',  // UI 說明
        subCategoryId: 'advanced.export',         // 歸屬的子分類 ID
        type: 'select',               // UI 類型
        options: [
            { value: 'xlsx', label: 'Excel (.xlsx)' },
            { value: 'csv',  label: 'CSV (.csv)' },
        ],
        persist: true,                // 是否持久化至後端
    },
]
```

### Step 3：完成

SettingsPage 會自動出現新的設定項目，Store 的初始值也自動更新。**不需要修改其他任何檔案。**

---

## 設定的 type 類型

| type | UI 元件 | value 格式 | options |
|---|---|---|---|
| `toggle` | Switch | `boolean` 或 `'dark'/'light'`（`theme` 特殊） | 不需要 |
| `select` | Select Dropdown | `string` | 必要（靜態：直接給陣列；動態：設為 `null`，由 Store 提供）|
| `number` | Input | `number` | 不需要 |
| `readonly` | 顯示用文字 | 任何 | 不需要 |

---

## 動態選項（options: null）

當選項需要從後端或 Store 動態取得，設定 `options: null`，並在 `SettingsPage.jsx` 的 `getDynamicOptions()` 函數中加入對映：

```js
const getDynamicOptions = (key) => {
    if (key === 'activeThemeId') {
        return availableThemes.map(t => ({ value: t.id, label: t.name }))
    }
    // 新增你的動態選項提供邏輯
    return []
}
```

---

## 在元件中讀取設定值

```js
import useSettingsStore from '../stores/useSettingsStore'

function MyComponent() {
    const theme = useSettingsStore(state => state.theme)
    const activeThemeId = useSettingsStore(state => state.activeThemeId)
    // ...
}
```

詳細 Store API 請參考 [`dev/modules/settings.md`](./modules/settings.md)。

---

## 非 UI 設定（不在 settingsConfig 中管理）

`bomSidebarWidth` 和 `isBomSidebarCollapsed` 是純 UI 狀態，不透過 settingsConfig 管理（因為沒有對應的 UI 設定項目），直接在 Store 中維護。

---

## 後端儲存格式

後端 API (`window.api.settings.save`) 接收的格式與 Store 的 key 基本一致，但 `activeThemeId` 在儲存時會被轉換為 `themeId`（歷史相容性）。
