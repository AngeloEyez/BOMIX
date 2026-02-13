# BOMIX 開發計畫

> 版本：1.0.0 | 最後更新：2026-02-13

## 開發策略

採用 **UI 優先、漸進式開發** 策略：
- 先建立可運作的基礎 UI 框架，讓使用者能看到完整的應用程式外觀
- 每個 Phase 同步開發 UI + 後端功能，完成後即可進行使用者測試
- 主行程（Jules）與渲染層（Antigravity）可依協作規範平行開發

---

## 里程碑規劃

### Phase 1：專案骨架與 UI 框架 ⭐ 目前
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
- [ ] **主題切換**
  - [ ] Dark/Light 模式切換邏輯
  - [ ] 系統主題偵測
- [ ] **對話框**
  - [ ] `components/dialogs/AboutDialog.jsx`
  - [ ] `components/dialogs/ChangelogDialog.jsx`

### Phase 2：主行程資料層
建立資料庫架構與 Repository Pattern，不含 UI 整合。

- [ ] `src/main/database/connection.js` — SQLite 連線管理
- [ ] `src/main/database/schema.js` — 建表 SQL
- [ ] `src/main/database/repositories/series.repo.js`
- [ ] `src/main/database/repositories/project.repo.js`
- [ ] `src/main/database/repositories/bom-revision.repo.js`
- [ ] `src/main/database/repositories/parts.repo.js`
- [ ] `src/main/database/repositories/second-source.repo.js`
- [ ] 單元測試（Vitest）

### Phase 3：系列與專案管理
IPC + Service + UI 同步開發。

- [ ] `src/main/services/series.service.js`
- [ ] `src/main/services/project.service.js`
- [ ] `src/main/ipc/series.ipc.js`
- [ ] `src/main/ipc/project.ipc.js`
- [ ] 更新 `src/preload/index.js`
- [ ] 更新 `pages/HomePage.jsx` — 整合系列開啟功能
- [ ] 更新 `pages/ProjectPage.jsx` — 整合專案 CRUD
- [ ] `stores/useSeriesStore.js` — Zustand 狀態
- [ ] `stores/useProjectStore.js`
- [ ] 整合測試

### Phase 4：BOM 管理
核心功能：BOM 表格檢視、編輯、Second Source 管理。

- [ ] `src/main/services/bom.service.js`
- [ ] `src/main/ipc/bom.ipc.js`
- [ ] 更新 `pages/BomPage.jsx` — TanStack Table 表格
- [ ] `components/tables/BomTable.jsx`
- [ ] `stores/useBomStore.js`
- [ ] 整合測試

### Phase 5：Excel 匯入/匯出
Excel 解析與產生功能。

- [ ] `src/main/services/import.service.js`
- [ ] `src/main/services/export.service.js`
- [ ] `src/main/ipc/excel.ipc.js`
- [ ] `components/dialogs/ImportDialog.jsx`
- [ ] 拖曳開啟 .xls/.xlsx 支援
- [ ] 整合測試

### Phase 6：版本比較
BOM 版本差異比對功能。

- [ ] `src/main/services/compare.service.js`
- [ ] 更新 `pages/ComparePage.jsx`
- [ ] 整合測試

### Phase 7：打包發佈
最終打包與文件完善。

- [ ] electron-builder 打包測試
- [ ] 使用者操作手冊完善
- [ ] 全面測試與 Bug 修復
- [ ] CI/CD 流程驗證
