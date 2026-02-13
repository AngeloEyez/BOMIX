# BOMIX 目錄結構說明

本文件說明專案各目錄的用途與職責。

---

## 頂層目錄總覽

| 目錄 | 用途 |
|------|------|
| `.agent/` | Antigravity IDE 工作流程設定 |
| `.gemini/` | Google Jules AI Agent 設定 |
| `.github/` | GitHub Actions CI/CD 設定 |
| `.vscode/` | VS Code / Antigravity IDE 編輯器設定與 Tasks |
| `dev/` | 開發者文件（規格書、架構、計畫、資料庫、協作規範） |
| `docs/` | 最終使用者說明文件（操作手冊、FAQ） |
| `references/` | 參考資料（BOM 範本檔案） |
| `resources/` | Electron 打包資源（圖標等） |
| `src/` | 主程式原始碼（Electron 三層架構） |
| `tests/` | 測試腳本與測試資料 |

---

## 詳細說明

### `.agent/workflows/`
Antigravity IDE 的 workflow 定義檔，可透過斜線命令快速執行。

### `.gemini/`
Google Jules AI Agent 的行為規範檔案 `GEMINI.md`，定義程式碼風格、命名規則等開發準則。

### `.github/workflows/`
GitHub Actions 自動化流程定義，用於 CI/CD 持續整合與自動測試。

### `.vscode/`
編輯器設定，包含：
- `settings.json` — ESLint、Prettier、Tailwind 整合
- `launch.json` — Electron 除錯配置
- `tasks.json` — 一鍵建置/測試 Tasks

### `dev/` — 開發者文件
| 文件 | 說明 |
|------|------|
| `SPEC.md` | 軟體規格書（BOM 結構、功能需求、UI 設計） |
| `ARCHITECTURE.md` | 系統架構設計（Electron 三層架構） |
| `DATABASE.md` | 資料庫結構設計（SQLite Schema） |
| `DEVELOPMENT.md` | 開發環境建置指南 |
| `BUILD.md` | 編譯與打包說明 |
| `PLAN.md` | 開發計畫與里程碑 |
| `COLLABORATION.md` | Jules + Antigravity 協作規範 |
| `modules/` | IPC API 模組說明文件 |

### `docs/` — 使用者說明文件
- `GETTING_STARTED.md` — 快速上手指南
- `USER_GUIDE.md` — 完整操作手冊
- `FAQ.md` — 常見問題

### `references/` — 參考資料
- `bom_templates/` — BOM Excel 範本檔案

### `resources/` — Electron 打包資源
- `icon.ico` — 應用程式圖標

### `src/` — 原始碼（Electron 三層架構）

#### `src/main/` — 主行程（Jules 負責）
| 子目錄 | 職責 |
|--------|------|
| `ipc/` | IPC 通道處理器（接收渲染層請求） |
| `services/` | 業務邏輯層 |
| `database/` | SQLite 連線管理、Schema 定義 |
| `database/repositories/` | Repository Pattern 資料存取層 |

#### `src/preload/` — Preload 橋接層
透過 `contextBridge` 安全暴露 API 給渲染層。

#### `src/renderer/` — 渲染層（Antigravity 負責）
| 子目錄 | 職責 |
|--------|------|
| `pages/` | 頁面元件（首頁、專案、BOM、比較、設定） |
| `components/` | 可重用 UI 元件（佈局、表格、對話框） |
| `stores/` | Zustand 狀態管理 |
| `utils/` | 前端工具函數 |

### `tests/` — 測試
- `unit/` — 單元測試（Vitest）
- `e2e/` — E2E 測試
