# BOMIX 協作規範 — Jules × Antigravity

> 版本：1.0.0 | 最後更新：2026-02-13

## 概述

BOMIX 採用兩個 AI Agent 協作開發：
- **Jules**（Google AI）：負責**主行程**（底層邏輯、資料庫、服務層、IPC）
- **Antigravity**（IDE）：負責**渲染層**（React UI、元件、頁面、樣式）

## 分工範圍

```
src/
├── main/           ← 🤖 Jules 負責
│   ├── index.js
│   ├── ipc/        ← Jules 定義 IPC Handler
│   ├── database/   ← Jules 獨佔
│   └── services/   ← Jules 獨佔
├── preload/        ← 🤝 雙方協作
│   └── index.js    ← API 契約層，需同步更新
└── renderer/       ← 🎨 Antigravity 負責
    ├── components/
    ├── pages/
    ├── stores/
    └── utils/
```

| 區域 | 負責人 | 說明 |
|------|--------|------|
| `src/main/database/` | Jules | 資料庫 Schema、Repository、連線管理 |
| `src/main/services/` | Jules | 業務邏輯層 |
| `src/main/ipc/` | Jules | IPC Handler（定義通道名稱與參數） |
| `src/preload/index.js` | 雙方 | API 橋接層（Jules 新增 API 後，通知 Antigravity） |
| `src/renderer/` | Antigravity | React 所有 UI 相關程式碼 |
| `dev/` | 雙方 | 文件同步更新 |
| `tests/unit/` | 對應負責人 | 各自撰寫負責模組的測試 |
| `tests/e2e/` | Antigravity | E2E 測試（需要瀏覽器操作） |

## IPC API 契約

### 命名規範
```
{模組}:{動作}

範例：
  series:create    — 建立系列
  series:open      — 開啟系列
  project:list     — 列出專案
  bom:getMainItems — 取得 BOM 聚合視圖
  excel:import     — 匯入 Excel
```

### 契約定義流程

1. **Jules** 在 `src/main/ipc/` 中實作 IPC Handler
2. **Jules** 在 `src/preload/index.js` 中新增對應的 API
3. **Jules** 建立 `dev/modules/{模組名}.md` 說明文件，記錄：
   - API 方法名稱與參數
   - 回傳值格式
   - 錯誤處理
   - 使用範例
4. **Antigravity** 根據說明文件，在 UI 中呼叫 `window.api.xxx()`

### API 說明文件格式

每個模組的 API 文件放在 `dev/modules/` 目錄：

```markdown
# Series API

## `window.api.series.create(filePath, description)`
建立新的系列資料庫。

- **參數**
  - `filePath` (string) — .bomix 檔案路徑
  - `description` (string) — 系列描述
- **回傳** `{ success: true, data: { id, description, createdAt } }`
- **錯誤** `{ success: false, error: '錯誤訊息' }`
```

## 工作流程

### 新功能開發流程
```
1. Jules：實作 service + repository + IPC handler
2. Jules：更新 preload API + 撰寫 API 文件
3. Jules：撰寫單元測試
4. Antigravity：根據 API 文件，實作 UI 元件 + Zustand Store
5. Antigravity：整合測試
```

### 溝通方式
- **API 變更**：Jules 修改 `preload/index.js` 後，需更新 `dev/modules/` 文件
- **UI 需求**：Antigravity 需要新 API 時，在 `dev/PLAN.md` 中記錄需求
- **共用文件**：`dev/` 下的文件為唯一來源（Single Source of Truth）

## 程式碼規範

### 共通規範
- 所有程式碼加上**繁體中文註解**
- 函數必須有 **JSDoc** 說明
- 使用 **ESLint + Prettier** 統一格式
- Commit 訊息使用繁體中文：`[模組名] 動詞 + 描述`

### 主行程（Jules）
- 使用 ES Module `import`, 不要使用  CommonJS `require`
- 同步 API 優先（better-sqlite3 為同步 API）
- 錯誤統一回傳 `{ success: false, error: '...' }` 格式

### 渲染層（Antigravity）
- 使用 React 函數元件 + Hooks
- 狀態管理使用 Zustand
- 樣式使用 Tailwind CSS
- 元件命名使用 PascalCase
