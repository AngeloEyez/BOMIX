# BOMIX Agent 開發指南

本文件定義 AI Agent 的開發規範、工作流程與引導 Prompt，確保開發者能高效且完整地完成開發任務。

## 協作核心原則

1.  **文件驅動開發**：在開始任何代碼編寫前，必須閱讀 `dev/` 下的規格與架構文件。
2.  **全棧垂直整合**：不論任務目標為何，開發者必須負責從資料庫、IPC、Service 到 UI 的完整實作。
3.  **強制技術記錄**：完成功能後，必須更新 `dev/modules/` 下的 API 文件，並確保架構圖與數據結構同步。
4.  **語言規範**：回應、註解、文件必須永遠使用**繁體中文**。

---

## 標準開發階段 (Phases)

開發者應遵循以下階段執行任務：

### 第一階段：需求與技術分析
**目標**：確保對現有架構與新需求有完整理解。
- 閱讀 `dev/PLAN.md` 確認當前進度。
- 閱讀 `dev/SPEC.md` 了解功能與 UI 規格。
- 閱讀 `dev/ARCHITECTURE.md` 了解系統結構。
- 閱讀 `dev/COLLABORATION.md` 確保遵守協作規範。

### 第二階段：垂直功能實作
**目標**：完成功能的端到端開發。
1.  **資料層**：更新 `src/main/database/` 中的 Schema 與 Repository。
2.  **業務層**：實作 `src/main/services/` 邏輯。
3.  **通訊層**：
    - 在 `src/main/ipc/` 實作 Handler。
    - 在 `src/preload/index.js` 暴露 API。
4.  **UI 層**：實作 React 元件、Zustand Store 與 Tailwind 樣式。

### 第三階段：文件化與驗證
**目標**：留下清晰的開發紀錄並驗證結果。
1.  **API 文件**：在 `dev/modules/` 建立或更新對應的 Markdown 文件。
2.  **架構更新**：若有新的全局邏輯，更新 `dev/ARCHITECTURE.md`。
3.  **測試**：撰寫並執行 Vitest 單元測試與整合測試。
4.  **進度回報**：更新 `dev/PLAN.md` 中的項目狀態。

---

## 任務引導 Prompt 範本 (供 User 使用)

當需要 Agent 執行特定 Phase 時，可參考以下範本：

```markdown
@Agent
請按順序閱讀以下文件以確保理解現狀：
- dev/PLAN.md
- dev/COLLABORATION.md
- dev/AGENT_GUIDE.md

任務目標：實作 [功能名稱] 的垂直整合。

執行步驟：
1. 分析受影響的資料庫表結構並更新 `dev/DATABASE.md`。
2. 實作後端 Repository 與 Service 層，並隨附測試。
3. 更新 Preload API 並在 `dev/modules/` 記錄新 API 規格。
4. 完成前端 UI 整合與狀態管理。
5. 更新 `dev/PLAN.md` 並記錄學習總結。

注意事項：
- 嚴格遵守繁體中文與 JSDoc 規範。
- 確保 UI 符合 `SPEC.md` 的視覺設計。
```

## 常見問題處理流程

1.  **環境問題**：參考 `dev/DEVELOPMENT.md` 的故障排除章節。
2.  **逻辑衝突**：若發現需求與現有架構衝突，應先在回應中提出討論，不要直接破壞基礎設施。
3.  **API 變更**：若修改了既有 API，必須搜索整個專案（PowerShell: `Select-String`）確保所有調用端同步更新。
