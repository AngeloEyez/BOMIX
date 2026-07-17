萬事俱備後，在你的 Claude Code 終端機互動介面中，直接輸入 /goal 指令。此時你要給它下達一個具備「分類 honesty（誠實分類）」與「推進條件」的嚴格總體目標：  

```text
/goal 根據 `implementationPlan.md` 的計畫路線與 `claude.md` 的技術規範，啟用 `phase-executor` 子代理進行全自動的階段性開發。

【自動推進合約（Stop Hook Condition）】
1. **階段委派**：針對 `implementationPlan.md` 中的每一個 Phase，你必須呼叫 `phase-executor` 去處理具體的程式碼實作與除錯。
2. **驗證與更新**：只有當該 Phase 定義的所有「驗證指令」（例如 `go test ./...` 或 `pnpm run build`）完全成功呈現 PASS (零錯誤)，且你已完成該 Phase 的 git commit 後，你才可以修改 `implementationPlan.md` 將該項目的 `[ ]` 改為 `[x]`。
3. **無人值守原則**：完成一個 Phase 後，自動滾動進入下一個 Phase。除非遇到以下兩種極端情況，否則【絕對禁止中斷】或詢問我：
   - 產品規格（`docs/product-spec.md`）與技術指南（`claude.md`）出現嚴重矛盾，需要我進行決策。
   - `phase-executor` 重試超過 5 次仍卡在同一個編譯、測試阻礙或環境錯誤。
4. **目標終點**：直到 `implementationPlan.md` 中所有 10 個 Phase 的項目全部變成 `[x]`，且最終執行 `wails3 build` 可成功產出執行檔為止。
```


# claude code
關閉安全審查機制
/config dangerouslyDisableSandbox=true 
