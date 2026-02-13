# Jules 開發計畫與 Prompts

本文件定義 Jules (Backend Agent) 的工作階段、協作機制以及對應的 Prompts。

本計畫已與 `dev/PLAN.md` 同步，Jules 負責其中的 **Phase 2, 3, 5, 7**。

## 協作機制

### 溝通核心
1.  **文件驅動**：Jules 必須嚴格遵守 `dev/COLLABORATION.md`。
2.  **API 契約**：Jules 完成每個功能後，必須：
    *   在 `src/main/ipc/` 實作 Handler。
    *   在 `src/preload/index.js` 暴露 API。
    *   更新 `dev/modules/{模組名}.md` 說明文件。
3.  **進度回報**：Jules 完成任務後，應更新 `dev/PLAN.md` 中對應的項目狀態。

---

## Jules 開發階段 (Phases)

請依序執行以下階段。每個階段完成後，請通知使用戶切換至 Antigravity Agent 進行對應的前端開發。

### Phase 2: 主行程資料層 (Database Foundation)
**對應 `PLAN.md`: Phase 2**
**目標**：建立 SQLite 連線、Schema 定義與 Repository 層。不涉及 IPC 與 Service。

#### Prompt
```markdown
@Jules
請按順序閱讀以下文件，以確保對專案目標與規範有完整的理解： 
 
dev/PLAN.md — 確認開發計畫
dev/SPEC.md — 了解功能需求與 UI 設計規格 
dev/ARCHITECTURE.md — 了解系統架構與頁面佈局設計 
dev/COLLABORATION.md — AI Agent合作注意事項
.gemini/GEMINI.md — 理解開發任務流程與程式碼規範（單一規範來源） 
閱讀完成後請開始執行 **Phase 2: 主行程資料層**。

**任務目標**：
建立 BOMIX 的資料庫底層架構，確保資料能正確寫入與讀取 SQLite。

**執行步驟**：
1. 閱讀 `dev/DATABASE.md` 熟悉 Schema 設計。
2. 實作 `src/main/database/connection.js` (better-sqlite3 連線)。
3. 實作 `src/main/database/schema.js` (定義 CREATE TABLE SQL)。
4. 實作所有 Repositories (`src/main/database/repositories/*.js`)，包含基本的 CRUD 方法。
5. 使用 `npm run build` 來確認可以正確編譯。
6. 為每個 Repository 撰寫單元測試 (`tests/unit/repositories/*.test.js`) 並確保通過。
7. 更新 `dev/PLAN.md` 的 Phase 2 完成狀態。

**注意事項**：
- 使用 Repository Pattern，不要在 Service 層寫 SQL。
- 測試資料庫請使用 `:memory:` 或暫存檔案。
- 嚴格遵守 `dev/COLLABORATION.md` 的程式碼規範（繁體中文註解、JSDoc）。
```

---

### Phase 3: 系列與專案管理 - 後端 (Series & Project Backend)
**對應 `PLAN.md`: Phase 3**
**目標**：實作系列與專案的業務邏輯，並開放 IPC API 供前端呼叫。

#### Prompt
```markdown
@Jules
請按順序閱讀以下文件，以確保對專案目標與規範有完整的理解： 
 
dev/PLAN.md — 確認開發計畫
dev/SPEC.md — 了解功能需求與 UI 設計規格 
dev/ARCHITECTURE.md — 了解系統架構與頁面佈局設計 
dev/COLLABORATION.md — AI Agent合作注意事項
.gemini/GEMINI.md — 理解開發任務流程與程式碼規範（單一規範來源） 
閱讀完成後請開始執行 **Phase 3: 系列與專案管理 - 後端**。

**任務目標**：
實作「系列 (Series)」與「專案 (Project)」的核心業務邏輯與 API。

**執行步驟**：
1. 閱讀 `dev/SPEC.md` (4.1 節) 與 `dev/COLLABORATION.md`。
2. 實作 Service 層：
   - `src/main/services/series.service.js` (建立/開啟資料庫, 更新 meta)
   - `src/main/services/project.service.js` (CRUD)
3. 實作 IPC Handler：
   - `src/main/ipc/series.ipc.js`
   - `src/main/ipc/project.ipc.js`
4. 更新 `src/preload/index.js` 暴露 API (`window.api.series.*`, `window.api.project.*`)。
5. **關鍵**：建立/更新 API 文件 `dev/modules/series.md` 與 `dev/modules/project.md`。
6. 使用 `npm run build` 來確認可以正確編譯。
7. 撰寫單元測試。
8. 更新 `dev/PLAN.md` 的 Phase 3 完成狀態。

**API 規範**：
- IPC 通道命名：`series:create`, `series:open`, `project:create` 等。
- 確保所有 Error 都有被捕捉並回傳標準錯誤格式。
```

---

### Phase 5: BOM 管理與 Excel 整合 - 後端 (BOM & Excel Backend)
**對應 `PLAN.md`: Phase 5**
**目標**：核心 BOM 編輯功能與 Excel 匯入/匯出。

#### Prompt
```markdown
@Jules
請按順序閱讀以下文件，以確保對專案目標與規範有完整的理解： 
 
dev/PLAN.md — 確認開發計畫
dev/SPEC.md — 了解功能需求與 UI 設計規格 
dev/ARCHITECTURE.md — 了解系統架構與頁面佈局設計 
dev/COLLABORATION.md — AI Agent合作注意事項
.gemini/GEMINI.md — 理解開發任務流程與程式碼規範（單一規範來源） 
閱讀完成後請開始執行 **Phase 5: BOM 管理與 Excel 整合 - 後端**。

**任務目標**：
實作 BOM 的讀取、編輯、Second Source 管理，NPI/MP Mode 判斷，以及符合規格的 Excel 匯入匯出功能。

**執行步驟**：
1. **BOM 核心**：
   - 實作 `src/main/services/bom.service.js` (BOM 聚合視圖邏輯、CRUD)。
   - **新增**：實作 NPI/MP Mode 自動判斷邏輯 (參考 `SPEC.md` 4.3.1)。
   - 實作 `src/main/ipc/bom.ipc.js`。
   - 更新 Preload 與 `dev/modules/bom.md`。
2. **Excel 整合**：
   - 閱讀 `dev/SPEC.md` (4.3 節) 關於 Excel 解析與匯出規則。
   - 實作 `src/main/services/import.service.js` (解析 .xls/.xlsx)。
   - 實作 `src/main/services/export.service.js` (產生 .xlsx，需符合 4.3.2 完整格式要求)。
   - 實作 `src/main/ipc/excel.ipc.js`。
   - 更新 Preload 與 `dev/modules/excel.md`。
3. 使用 `npm run build` 來確認可以正確編譯。
4. **測試**：
   - 使用 `references/bom_templates/` 下的範本檔進行匯入測試，驗證 Mode 判斷正確性。
   - 驗證匯出的 Excel 格式是否符合規格 (Sheet、Header、Style)。
5. 更新 `dev/PLAN.md` 的 Phase 5 完成狀態。

**注意事項**：
- 解析 Excel 時需特別注意 `Main Source` 與 `2nd Source` 的判斷邏輯。
- BOM 聚合視圖 (Main Item) 是動態計算的，資料庫儲存的是原子化 (Location-based) 資料。
```

---

### Phase 7: 版本比較 - 後端 (Comparison Backend)
**對應 `PLAN.md`: Phase 7**
**目標**：比對兩個 BOM 版本差異。

#### Prompt
```markdown
@Jules
請按順序閱讀以下文件，以確保對專案目標與規範有完整的理解： 
 
dev/PLAN.md — 確認開發計畫
dev/SPEC.md — 了解功能需求與 UI 設計規格 
dev/ARCHITECTURE.md — 了解系統架構與頁面佈局設計 
dev/COLLABORATION.md — AI Agent合作注意事項
.gemini/GEMINI.md — 理解開發任務流程與程式碼規範（單一規範來源） 
閱讀完成後請開始執行 **Phase 7: 版本比較 - 後端**。

**任務目標**：
實作 BOM 版本差異比對演算法。

**執行步驟**：
1. 閱讀 `dev/SPEC.md` (4.4 節)。
2. 實作 `src/main/services/compare.service.js`。
   - 邏輯：比對兩個 BOM Revision 的零件清單。
   - 輸出：Added, Removed, Modified, Unchanged 列表。
3. 實作 `src/main/ipc/compare.ipc.js`。
4. 更新 Preload 與 `dev/modules/compare.md`。
5. 使用 `npm run build` 來確認可以正確編譯。
6. 撰寫單元測試，驗證比對邏輯準確性。
7. 更新 `dev/PLAN.md` 的 Phase 7 完成狀態。

**注意事項**：
- 比對鍵值 (Key) 通常結合 `HHPN` + `Location`。
- 需考慮 2nd Source 的變動。
```
