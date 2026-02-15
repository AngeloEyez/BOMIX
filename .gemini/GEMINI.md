# BOMIX 專案 - Google Jules AI Agent 規範

Agent 必須使用繁體中文回覆, 包含程式碼註解, Commit 訊息, 任務規劃, 任務執行報告, 提出PR說明, PR回應, 與使用者對話等所有產出物。

## 專案概述
BOMIX 是一個 BOM（Bill of Materials）變化管理與追蹤工具，用於管理主機板電子 BOM 的版本變化。

## 技術堆疊
- **執行環境**：Electron
- **UI 框架**：React + Tailwind CSS v4
- **表格元件**：TanStack Table
- **建置工具**：Vite（electron-vite）
- **狀態管理**：Zustand
- **資料庫**：SQLite（better-sqlite3，主行程中同步存取）
- **Excel 處理**：xlsx（SheetJS），支援 `.xls` 與 `.xlsx`
- **打包工具**：electron-builder（Windows 安裝版 + 便攜版）
- **測試框架**：Vitest
- **程式碼風格**：ESLint + Prettier

## 專案結構
- Electron 三層架構：`src/main/`（主行程）→ `src/preload/`（橋接）→ `src/renderer/`（渲染層）
- 主行程模組化：`ipc/` → `services/` → `database/repositories/`
- 渲染層：`pages/` + `components/` + `stores/`
- 測試位於 `tests/`（unit / e2e）

## ⚠️ Jules 負責範圍

**Jules 專注於主行程開發**，包含：
- `src/main/` 下的所有程式碼
- `src/preload/index.js` 的 API 定義
- `tests/unit/` 的主行程相關測試

**不要觸碰** `src/renderer/` 下的 React UI 程式碼（由 Antigravity 負責）。

## ⚠️ 開發任務執行流程

### 每次任務開始前，必須按以下順序閱讀文件：
1. **`dev/PLAN.md`** — 確認當前開發階段與待辦事項
2. **`dev/SPEC.md`** — 了解功能需求與資料結構定義
3. **`dev/ARCHITECTURE.md`** — 了解系統架構
4. **`.gemini/GEMINI.md`** — 了解開發規範（即本文件）
5. **`dev/DATABASE.md`** — 了解資料庫 Schema
6. **`dev/COLLABORATION.md`** — 了解 Jules 與 Antigravity 的協作規範

### Agent Workspace
當 Agent 需要撰寫臨時工具程式或測試腳本時，請將檔案放置於 `agent-workspace/` 目錄中。
- 禁止將臨時檔案直接建立在專案根目錄。

### 標準工作流程：
1. **閱讀** — 讀取上述文件，確認任務範圍
2. **規劃** — 列出要建立/修改的檔案清單
3. **實作** — 按照本文件規範撰寫程式碼
4. **測試** — 為新功能撰寫對應的測試
5. **文件** — 更新 `dev/modules/` 下的 API 說明文件
6. **提交** — 使用繁體中文 commit 訊息

### 重要規則：
- **先做當前 Phase**：只處理 PLAN.md 中標記為「目前」的 Phase
- **不要跳過**：按照 PLAN.md 的檔案清單逐一完成
- **漸進式**：每次 commit 應是一個可運作的狀態
- **API 文件同步**：每次新增/修改 IPC API，必須同步更新 `dev/modules/` 說明文件

---

## 開發規範

### 1. 程式碼風格

#### 1.1 中文註解要求
- **所有程式碼必須加上繁體中文註解**
- 註解應簡明扼要，說明「為什麼」而非僅描述「做什麼」
- 區塊級別的功能說明使用 `// ========` 分隔

#### 1.2 函數 JSDoc 規範
每個函數都必須包含標準化的 JSDoc：

```javascript
/**
 * 計算兩個 BOM 版本之間的差異。
 *
 * 比較舊版與新版 BOM 的零件清單，找出新增、刪除、修改的項目。
 *
 * @param {Array} oldBom - 舊版 BOM 零件清單
 * @param {Array} newBom - 新版 BOM 零件清單
 * @returns {{ added: Array, removed: Array, modified: Array }} 差異物件
 */
function calculateBomDiff(oldBom, newBom) { ... }
```

#### 1.3 命名規範
- **變數與函數**：使用 camelCase (例: `bomItemCount`)
- **React 元件**：使用 PascalCase (例: `ProjectPage`)
- **常數**：使用 UPPER_SNAKE_CASE (例: `MAX_RETRY_COUNT`)
- **檔案命名**：
  - 元件檔案：PascalCase (例: `BomTable.jsx`)
  - 其他檔案：kebab-case (例: `bom-revision.repo.js`)
- **IPC 通道**：`{模組}:{動作}` (例: `series:create`)

#### 1.4 其他規範
- 每行最多 100 字元
- 使用 JSDoc 型別標註
- import 排序：Node.js 內建 → Electron → 第三方套件 → 本地模組

#### 1.5 模組 / API 說明文件
- 每個 IPC 模組必須有說明文件，目的是讓 Antigravity Agent 快速理解如何呼叫 API
- 說明文件位置：`dev/modules/`，檔名與模組同名
- 每次調整 IPC API，需同時將說明文件更新到最新狀態
- 說明文件應包含：
    - API 方法名稱
    - 參數說明
    - 回傳值格式
    - 錯誤處理
    - 使用範例

---

### 2. 架構與模組規範

#### 2.1 模組依賴方向
```
IPC Handlers → Services → Repositories → SQLite
                 ↓
              Models（純資料結構）
```

- IPC Handler 只負責參數解析與回傳，不含業務邏輯
- Services 層負責業務邏輯
- Repositories 層負責 SQL 操作，使用 **Repository Pattern**
- 所有 API 回傳統一格式：`{ success: true, data: ... }` 或 `{ success: false, error: '...' }`

#### 2.2 錯誤處理
- IPC Handler 中統一 try-catch，回傳標準錯誤格式
- Service 層可拋出自訂錯誤
- 禁止讓未捕捉的錯誤傳到渲染層

#### 2.3 資料庫操作
- 所有 SQL 操作封裝在 repositories 中
- 使用參數化查詢，禁止字串拼接 SQL
- better-sqlite3 為同步 API，可直接回傳結果

---

---

### 3. 文件與提交規範

#### 3.0 程式碼修改規範 (Agent 專用)
- **禁止使用縮寫取代**：在使用 `replace_file_content` 修改程式碼時，**絕對禁止**使用 `// ... unchanged` 或 `// ... (rest of code)` 等註解來代表未修改的程式碼。這會導致原始程式碼被註解覆蓋而遺失。
- **完整保留**：必須完整保留該區塊的所有原始程式碼，或是使用 `multi_replace_file_content` 針對特定行數進行精準修改。

#### 3.1 說明文件
- 除了專業術語外，盡量使用**繁體中文**
- 文件格式統一使用 Markdown

#### 3.2 Commit 訊息
- 使用繁體中文撰寫
- 格式：`[模組名] 動詞 + 描述`
- 範例：`[database] 新增 BOM 項目資料存取層`

---

### 4. 測試要求
- 每個 service 函數都應有對應的單元測試
- 資料庫操作需有整合測試
- 測試使用 Vitest 框架
- 測試檔名對應原始檔名：`bom.service.js` → `bom.service.test.js`
- 匯入 Excel BOM 的範例檔案：`references/bom_templates/TANGLED_EZBOM_SI_0.3_BOM_20240627_0900WithProtoPart(compared).xls`

---

### 文件參考
- 軟體規格書：`dev/SPEC.md`
- 開發計畫：`dev/PLAN.md`
- 架構設計：`dev/ARCHITECTURE.md`
- 資料庫設計：`dev/DATABASE.md`
- 協作規範：`dev/COLLABORATION.md`
