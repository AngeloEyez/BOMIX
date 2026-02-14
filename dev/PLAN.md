# BOMIX 開發計畫

> 版本：1.1.0 | 最後更新：2026-02-13

## 開發策略

採用 **前後端分離、循序開發** 策略：
- **Jules (Backend)** 先行開發 API 與資料邏輯
- **Antigravity (Frontend)** 接續整合 UI
- 透過明確的 Phase 切分，讓兩位 Agent 的職責更清晰

---

## 里程碑規劃

### Phase 1：專案骨架與 UI 框架 ⭐ 已完成
建立 Electron + React + Vite 專案骨架，包含完整導航與主題切換。

- [x] Electron + React + Vite 專案初始化
- [x] Tailwind CSS v4 配置
- [x] 專案目錄結構建立
- [x] ESLint + Prettier 設定
- [x] `.gitignore` 建立
- [x] GitHub CI/CD 更新
- [x] VS Code Tasks 配置
- [x] 所有開發文件更新
- [x] **主行程基礎**
  - [x] `src/main/index.js` — 主視窗建立
  - [x] `src/main/ipc/index.js` — IPC handler 註冊框架
- [x] **Preload**
  - [x] `src/preload/index.js` — contextBridge API
- [x] **UI 主框架**
  - [x] `src/renderer/App.jsx` — 主元件與頁面切換
  - [x] `components/layout/AppLayout.jsx` — 主佈局
  - [x] `components/layout/Sidebar.jsx` — 導航列
  - [x] `components/layout/StatusBar.jsx` — 狀態列
- [x] **基礎頁面（佔位）**
  - [x] `pages/HomePage.jsx` — 首頁
  - [x] `pages/ProjectPage.jsx` — 專案管理（佔位）
  - [x] `pages/BomPage.jsx` — BOM 檢視（佔位）
  - [x] `pages/ComparePage.jsx` — 版本比較（佔位）
  - [x] `pages/SettingsPage.jsx` — 設定
- [x] **UI 收尾**
  - [x] Dark/Light 模式切換邏輯
  - [x] `components/dialogs/AboutDialog.jsx`
  - [x] `components/dialogs/ChangelogDialog.jsx`

### Phase 2：主行程資料層 (Jules)
**負責人：Jules**
建立資料庫架構與 Repository Pattern，不含 IPC 與 UI 整合。

- [x] `src/main/database/connection.js` — SQLite 連線管理
- [x] `src/main/database/schema.js` — 建表 SQL (Schema 定義)
- [x] `src/main/database/repositories/series.repo.js`
- [x] `src/main/database/repositories/project.repo.js`
- [x] `src/main/database/repositories/bom-revision.repo.js`
- [x] `src/main/database/repositories/parts.repo.js`
- [x] `src/main/database/repositories/second-source.repo.js`
- [x] 單元測試 (Repositories)

### Phase 3：系列與專案管理 - 後端 (Jules)
**負責人：Jules**
實作核心業務邏輯並開放 API。

- [x] `src/main/services/series.service.js`
- [x] `src/main/services/project.service.js`
- [x] `src/main/ipc/series.ipc.js`
- [x] `src/main/ipc/project.ipc.js`
- [x] 更新 `src/preload/index.js`
- [x] 撰寫 API 文件 (`dev/modules/`)
- [x] 單元測試 (Services)

### Phase 4：系列與專案管理 - 前端 (Antigravity)
**負責人：Antigravity**
整合 Phase 3 開放的 API，完成 UI 功能。

- [x] `stores/useSeriesStore.js`
- [x] `stores/useProjectStore.js`
- [x] 更新 `pages/HomePage.jsx` (建立/開啟系列)
- [x] 更新 `pages/ProjectPage.jsx` (專案 CRUD、版本列表)
- [ ] 整合測試

### Phase 5：BOM 管理與 Excel 整合 - 後端 (Jules)
**負責人：Jules**
BOM 核心邏輯、聚合視圖計算、Excel 解析與匯出。

- [x] `src/main/services/bom.service.js` (含 Mode 判斷邏輯)
- [x] `src/main/services/import.service.js`
- [x] `src/main/services/export.service.js` (實作完整 Excel 匯出規格)
- [x] `src/main/ipc/bom.ipc.js`
- [x] `src/main/ipc/excel.ipc.js`
- [x] 更新 `src/preload/index.js`
- [x] 撰寫 API 文件 (`dev/modules/`)
- [x] 單元測試 (含 Excel 範本與匯出格式驗證)

### Phase 6：BOM 管理與 Excel 整合 - 前端 (Antigravity)
**負責人：Antigravity**
整合 BOM 表格與 Excel 匯入匯出功能。

- [ ] `stores/useBomStore.js`
- [ ] `components/tables/BomTable.jsx` (TanStack Table 實作)
- [ ] 更新 `pages/BomPage.jsx`
- [ ] `components/dialogs/ImportDialog.jsx`
- [ ] 拖曳匯入功能實作
- [ ] 整合測試

### Phase 7：版本比較 - 後端 (Jules)
**負責人：Jules**
實作 BOM 版本差異比對演算法。

- [ ] `src/main/services/compare.service.js`
- [ ] `src/main/ipc/compare.ipc.js`
- [ ] 更新 `src/preload/index.js`
- [ ] 撰寫 API 文件 (`dev/modules/`)
- [ ] 單元測試 (演算法驗證)

### Phase 8：版本比較 - 前端 (Antigravity)
**負責人：Antigravity**
呈現比對結果。

- [ ] `stores/useCompareStore.js`
- [ ] 更新 `pages/ComparePage.jsx` (差異視覺化呈現)
- [ ] 整合測試

### Phase 9：打包與發佈
最終打包、E2E 測試與文件完善。

- [ ] electron-builder 完整打包測試
- [ ] 使用者操作手冊完善
- [ ] Bug Bash 與修復
- [ ] CI/CD 流程驗證
