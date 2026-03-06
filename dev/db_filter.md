# BOMIX - BOM 資料撈取與整合邏輯分析

這份文件說明了 BOMIX 系統中，當使用者從 UI 選擇 BOM 後，資料如何經過各層級傳遞、從 SQLite 資料庫撈取、整合（包含過濾與群組化），最終回傳給前端渲染的完整流程。同時包含了單一 BOM 與 Matrix 模式之間的不同邏輯。

---

## 1. 總覽與資料流向 (Data Flow)

**UI 操作** $\rightarrow$ **Zustand Store** $\rightarrow$ **IPC (預載入指令)** $\rightarrow$ **服務層 (Service)** $\rightarrow$ **資料存取層 (Repository)** $\rightarrow$ **SQLite 資料庫**

1. **前端 (Renderer):** 使用者在 UI `BomSidebar` 點擊 BOM 版本、切換 `ALL/SMD/PTH` 視圖、或是輸入搜尋關鍵字。`BomPage.jsx` 會觸發 Store 狀態更新。
2. **Store (Zustand):** `useBomStore` 更新當前選取的 ID 與視圖 (View ID)，並呼叫 IPC 發送請求。
3. **後端 (Main):**
   - **路由:** `bom.ipc.js` (或 `matrix.ipc.js`) 接收請求，轉發給 Service。
   - **服務層:** `bom.service.js` (或 `matrix.service.js`) 根據 View 定義 (例如狀態條件、類型條件)，呼叫 Repository 撈取基礎資料後，在 JavaScript 記憶體中執行過濾、分組、聚合 (Aggregation) 邏輯。
   - **資料庫:** `parts.repo.js` 執行 SQL `SELECT * FROM parts WHERE bom_revision_id IN (...)` 等。
4. **回傳與渲染:** 處理完的資料透過 IPC 回傳至 Store，存入 `bomView` 狀態。最後 `BomPage.jsx` 會根據 `searchTerm` (搜尋列) 做最後一道文字過濾，再傳給 `BomTable.jsx` 渲染。

---

## 2. 核心邏輯：一般 BOM (單選) 與 BigBOM (多選)

當使用者在單一或多個 BOM 版本上進行檢視時，主要透過 `bom.service.js` 的 `executeView` 函數來處理。

### 2.1 觸發階段 (Frontend)
- 在 `useBomStore.toggleRevisionSelection` 中，會將選取的 IDs 存入 `selectedRevisionIds`。如果超過 1 個，`bomMode` 切換至 **BIGBOM**。
- `useBomStore.selectView` 或 `fetchData` 會呼叫 `window.api.bom.getView(idsArray, currentViewId)`。

### 2.2 撈取與過濾階段 (Backend - `bom.service.executeView`)
- **取得定義:** 根據 `viewId` (如 `SMD`, `ALL`)，從 `bom-factory.service` 中取得**過濾定義 (View Definition)**，決定哪些 `type` 或 `bom_status` 予以保留。
- **資料庫查詢:**
  - 呼叫 `partsRepo.findByBomRevisions(ids)` 執行 `SELECT * FROM parts WHERE bom_revision_id IN (...)` 撈取**所有**對應的零件。
  - 呼叫 `secondSourceRepo.findByBomRevision(id)` 撈取替代料 (Second Sources)。
- **JavaScript 過濾 (Filter):**
  - **Type Filter:** 如果視圖限制 `types` (例如 SMD 僅包含 `"SMD"`)，則不在名單內的零件被剔除。
  - **Status Filter:** 根據專案模式 (NPI 或 MP) 與視圖定義過濾 `bom_status`。
    - **ACTIVE:** 保留一般有效料。例如 NPI 模式保留 `['I', 'P']`。
    - **INACTIVE:** 保留刪除料。例如 NPI 模式保留 `['X', 'M']`。
  - **CCL Filter:** 如果設定 `filter.ccl`，則剔除不符的零件。

### 2.3 分組與聚合 (Aggregation)
為了將多個相同料號的打件位置 (Location) 顯示為同一列：
- 使用 `Map` 建立群組，**群組鍵值 (Key)** 為 `supplier|supplier_pn`。
- 相同 `supplier_pn` 的零件：
  - 加總 `quantity` (+1)。
  - 收集所有打件位置 `location` 並合併為陣列。
  - 紀錄該料號存在於哪些 BOM ID (`bom_ids` Set)，支援 Union 檢視。
  - 取最小的 `item` 作為該群組的排序基準。
- **Second Sources 關聯:** 將撈取到的替代料透過 `main_supplier|main_supplier_pn` 鍵值，綁定到對應的 Main Item 群組中 (`item.second_sources`)。
- **回傳結果:** 排序 locations 與 item 後，將結果陣列透過 IPC 回傳至前端。

### 2.4 前端文字搜尋 (Search Term Filter)
資料回到 UI 後，`BomPage.jsx` 會使用 `useMemo`，根據目前的 `searchTerm` 與勾選的 `searchFields` (包含 HHPN, Description, Location 等)，對聚合後的 `bomView` 陣列進行字串比對過濾，產生 `filteredBom` 提供給 `BomTable` 渲染。

---

## 3. 核心邏輯：Matrix 模式 (多 BOM 差異與選擇)

在 Matrix 模式下，雖然使用者選了多個版本，但不僅僅是將資料聚合，還需要產生專案之間的「選擇狀態 (Selections)」。

### 3.1 狀態與資料需求
Matrix 模式需要兩份資料：
1. **基礎 BOM 資料:** 跟一般視圖一樣，`useBomStore` 負責去拉出 `bomView` (通常是 ALL 視圖，過濾 Active，只顯示 `CCL="Y"` 的資料)。
2. **Matrix 模型與選擇:** `useMatrixStore` 負責呼叫 `window.api.matrix.getData(idsArray)`。

### 3.2 Matrix Data 計算 (Backend - `matrix.service.getMatrixData`)
- **取得 Models:** 從資料庫取得這些 BOM 關聯的 Matrix Models (`matrix_models` 表)。
- **取得 explicit selections:** 從資料庫取得所有明確被使用者點擊選擇的紀錄 (`matrix_selections` 表)。
- **取得 Union BOM View:**
  - 呼叫 `bomService.executeView(ids, { filter: { statusLogic: 'ACTIVE', ccl: 'Y' } })` 取得所有版本的 Active CCL 零件。
- **計算隱式選擇 (Implicit Selections):**
  - 不必所有零件都需使用者手動選擇。
  - 系統遍歷上述 Union BOM 的每一個零件群組 (Main Item)。
  - 針對每個 Model 確認：如果該零件**存在於該 Model 的 BOM** (`bom_ids` 包含此 Model 的 BOM ID)，且**該零件沒有 Second Source**，且**資料庫內沒有 explicit selection**，則系統自動產生一筆隱式的 Main Part 選擇 (Implicit Selection)。
- **計算完成度 (Summary):**
  - 分別計算每個 Model 的總零件群組數 (`modelTotalGroups`)。
  - 再根據 (explicit + implicit 選擇數) 判斷該 Model 是否 **Completed** (`selectedCount === modelTotal `)。全系列 Completed 則狀態為 Safe。
- **回傳結果:** 回傳 `{ models, selections, summary }` 到前端的 `useMatrixStore`。

### 3.3 結合渲染 (`BomTable.jsx`)
- 表格列出 `filteredBom` 中每個零件群組 (`Main Item`) 及其替代料。
- 表格右方會動態展開對應 Matrix Models 數量的「選擇欄位」。
- 根據 `useMatrixStore.matrixData` 提供的 `selections`，在對應的欄位上標示綠色勾勾 (Implicit/Explicit Part/Second Source 選擇)。如果沒有選擇，則維持空白或警示。

---

## 4. 總結比較表

| 項目 | 一般 BOM / BigBOM (單/多選) | Matrix 模式 |
| :--- | :--- | :--- |
| **主要處理層** | `bom.service.js` | `bom.service.js` + `matrix.service.js` |
| **BOM撈取範圍**| 取決於視圖 (ALL/SMD/PTH 等條件) | 限定為 Active 狀態且 `CCL="Y"` |
| **回傳結構** | `Array<AggregatedItem>` | 獨立分為 `bomView` (陣列) 與 `matrixData` (包含 Models, Selections, Summary) |
| **聚合分組鍵** | `supplier` + `supplier_pn` | 同左 (藉由呼叫 executeView) |
| **隱式處理** | 無 | 系統自動針對無替代料的零件進行隱式選擇 (Implicit Selection) |
| **前端過濾** | 依據 `searchTerm` 與欄位過濾 | 依據 `searchTerm` 過濾列表，UI 動態渲染選擇勾選狀態 |

這樣的分層架構讓資料存取層保持簡單 (`SELECT *`)，把繁重的 Union、聚合、過濾與業務邏輯集中在 JavaScript 的 Service 處理層，以便更容易適應未來各種複雜的 BOM View 定義與 Matrix 選擇計算需求。
