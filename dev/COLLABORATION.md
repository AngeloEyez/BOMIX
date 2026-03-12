# BOMIX 協作規範

> 版本：2.0.0 | 最後更新：2026-03-12

## 概述

BOMIX 的開發採用「任務導向、全棧負責」的原則。不論是由哪個 AI Agent 或開發者執行，都必須負責完成從底層邏輯到渲染層 UI 的完整任務需求，並確保系統的一致性與穩定性。

## 開發責任與範圍

開發者必須負責其所開發功能的完整垂直整合：

```
src/
├── main/           ← 核心邏輯、資料庫、服務層、IPC實作
├── preload/        ← API 契約層（由功能開發者同步更新）
└── renderer/       ← UI 元件、頁面、狀態管理、樣式
```

### 開發義務清單

為確保後續開發者（包括其他 Agent）能無縫接手，每項功能開發必須包含：
1.  **完整實作**：包含資料庫 Schema 變更、Service 邏輯、IPC 通道、以及 UI 呈現。
2.  **技術文件化**：同步更新 `dev/` 目錄下的相關文件（如 `ARCHITECTURE.md`, `DATABASE.md` 等）。
3.  **API 註冊**：在 `src/preload/index.js` 暴露 API 並在 `dev/modules/` 撰寫說明。
4.  **自動化測試**：隨附單元測試與（若涉及 UI）整合測試。

| 區域 | 負責事項 |
|------|--------|
| `src/main/database/` | 資料庫 Schema、Repository、連線管理 |
| `src/main/services/` | 業務邏輯層 |
| `src/main/ipc/` | IPC Handler 定義與實作 |
| `src/preload/index.js` | API 橋接層，需確保型別定義與 IPC 通道一致 |
| `src/renderer/` | React 所有 UI 相關程式碼與 Zustand 狀態管理 |
| `dev/` | **強制要求**：同步更新所有受影響的設計與 API 文件 |
| `tests/` | 撰寫對應模組的單元與整合測試 |

## IPC API 契約規範

### 命名規範
```
{模組}:{動作}

範例：
  series:create    — 建立系列
  bom:getMainItems — 取得 BOM 聚合視圖
  excel:import     — 將 Excel 匯入任務加入佇列
```

### API 說明文件 (強制)
所有新增或修改的 API 必須記錄於 `dev/modules/{模組名}.md`：
- API 方法名稱與參數 (含型別)
- 回傳值格式 (JSON 結構)
- 錯誤處理機制
- 呼叫範例

## 工作流程

### 功能開發標準流程
1.  **需求分析**：閱讀 `dev/SPEC.md` 與相關技術文件。
2.  **設計變更**：若涉及架構或資料庫，先更新 `dev/ARCHITECTURE.md` 或 `dev/DATABASE.md`。
3.  **垂直實作**：
    - 實作資料庫與業務邏輯。
    - 建立 IPC 通道並暴露至 Preload。
    - 撰寫/更新 `dev/modules/` API 文件。
    - 實作前端 Store 與 UI。
4.  **驗證**：運行測試並手動確認功能完整性。
5.  **文件收尾**：更新 `dev/lessonlearned.md` 記錄遇到的問題與解決方案。

## 程式碼規範

### 共通規範
- **繁體中文語系**：產生回應、撰寫文件、撰寫程式碼註解必須永遠使用繁體中文。
- **JSDoc 標準**：對於所有的函數 (function) 與類別 (class)，必須加入 JSDoc 說明規格。
- **易讀註解**：程式碼中必須增加人類易讀的註解，解釋複雜邏輯。
- 使用 **ESLint + Prettier** 統一格式。

### 主行程 (Node.js/Electron)
- 使用 ES Module `import`。
- 錯誤統一回傳 `{ success: false, error: '錯誤訊息' }` 格式。

### 渲染層 (React)
- 使用 React 函數元件 + Hooks。
- 狀態管理使用 Zustand。
- 樣式優先使用 SHADCN UI 完成 (注意參考 lessonlearned.md v3/v4的語法差異)。
