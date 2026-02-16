# BOMIX 軟體規格書

> 版本：1.0.0 | 最後更新：2026-02-13

## 1. 系統概述

BOMIX 是一個桌面應用程式，用於管理與追蹤電子 BOM（Bill of Materials）的變化。

### 1.1 核心目標
- 管理多專案、多 Phase、多版本的 BOM 資料
- 追蹤 BOM 版本間的變化
- 支援 Excel 匯入/匯出
- 提供圖形化操作介面

---

## 2. BOM 資料結構

### 2.1 零件基本屬性

| 欄位 | 資料庫欄位名 | 說明 | 範例 |
|------|-------------|------|------|
| Item | `item` | 項目編號，從 Excel 導入，僅作紀錄 | 1, 2, 3 |
| HHPN | `hhpn` | 公司內部料號 | 34065Y600-GRT-H |
| Supplier | `supplier` | 供應商名稱 | Samsung, Murata |
| Supplier PN | `supplier_pn` | 供應商料號 | CL05B104KO5NNNC |
| Description | `description` | 零件描述 | CAP,22uF,+/-20%,X5R,6.3V,SMD0603,ROHS,HF |
| Location | `location` | 零件位置編號（原子化，每行一個） | R1, XU1, C3 |
| Type | `type` | 製程類型，可為空 | SMD, PTH, BOTTOM |
| BOM | `bom_status` | 上件狀態 | I, X, P, M |
| CCL | `ccl` | 是否為 Critical Part | Y, N |
| Remark | `remark` | 註記資料 | — |

#### BOM 上件狀態說明

| 狀態碼 | 說明 |
|--------|------|
| **I** | 上件（Install） |
| **X** | 不上件 |
| **P** | Proto Part，僅原型階段上件 |
| **M** | MP Only，僅量產上件，驗證階段不上件 |

### 2.2 BOM Main Item（檢視用）
- 相同 **Supplier + Supplier PN** 的零件合併為一個 BOM Main Item
- **Type** 不再作為合併條件（即相同供應商與料號的零件，即使製程不同，也視為同一 Main Item）
- 位置編號以逗號分隔合併,中間無空格（例：`C1,C2,C3,C5`）
- 數量由位置編號的數量自動計算
- **此為 UI 呈現的聚合視圖**，資料庫中以原子化儲存（一個 location = 一筆紀錄）

### 2.3 Second Source（替代料）

每個 BOM Main Item 可附帶 **0 到多個** Second Source，屬性如下：

| 欄位 | 資料庫欄位名 | 說明 | 範例 |
|------|-------------|------|------|
| HHPN | `hhpn` | 公司內部料號 | 34065Y600-GRT-H |
| Supplier | `supplier` | 供應商名稱 | Yageo |
| Supplier PN | `supplier_pn` | 供應商料號 | RC0402FR-0710KL |
| Description | `description` | 零件描述 | CAP,22uF,+/-20%,X5R,6.3V,SMD0603,ROHS,HF |

- Second Source **不包含** Location、Type、BOM Status 等欄位（已由 Main Source 定義）
- 透過 `(bom_revision_id, main_supplier, main_supplier_pn)` 邏輯鍵關聯至主料群組

---

## 3. 專案層次結構

```
系列（Series）              ← 一個獨立的 .bomix 資料庫檔案
├── series_meta             ← 系列設定與資訊
├── 專案 A（Project）
│   ├── DB 0.1              ← bom_revision（phase_name + version）
│   ├── DB 0.2
│   ├── SI 0.1
│   ├── PV 0.1
│   └── MVB 0.1
└── 專案 B（Project）
    ├── EVT 0.1             ← Phase 名稱可自訂
    ├── DVT 0.1
    └── PVT 0.1
```

### 3.1 Phase 類型
- Phase 名稱**可自訂**，每個專案可依需求命名
- 常見範例：DB, SI, PV, MVB, EVT, DVT, PVT 等
- 不限定固定清單，使用者可輸入任意名稱

### 3.2 版本編號
- 每個 Phase 可以有多個 BOM 版本
- 版本號格式：`0.1`, `0.2`, `1.0`, `1.1` ...
- Phase + Version 的組合在同一專案中唯一

---

## 4. 功能需求

### 4.1 系列與專案管理
- 建立/開啟/關閉系列資料庫（`.bomix` 檔案）
- **編輯系列資訊**：修改 description（可編輯）、顯示 create date / modify date（唯讀）
- 新增/編輯/刪除專案
- 管理 BOM Revision（Phase + Version）

### 4.2 BOM 管理
- 以**表格**方式呈現 BOM（使用 TanStack Table）
- 檢視選定的 BOM，以**聚合視圖**方式根據篩選規則顯示：
  - 支援**多種 BOM 視圖 (View)** 切換，邏輯由後端統一管理：
    - **ALL**：顯示所有 Active 零件 (排除不需上件項目 `X`)。
    - **SMD**：僅顯示製程為 `SMD` 的 Active 零件。
    - **PTH**：僅顯示製程為 `PTH` 的 Active 零件。
    - **BOTTOM**：僅顯示製程為 `BOTTOM` 的 Active 零件。
  - 以 Main Source 為主體，顯示 HHPN、Description、Supplier、Supplier PN、Qty、Location、CCL、Remark 等欄位。
  - Main Source 顯示「**Main**」標記，其下方帶入 2nd Source 列表 (無標記)。
  - UI 設計須讓人**容易區分** Main Source 和 2nd Source（例如：縮排、顏色差異、圖標、Main 粗體）。
- **表格排序功能**：
  - 支援點擊表頭對 **Main Item** 進行排序 (如 HHPN、Description 等)。
  - **分組保持**：排序僅影響 Main Item 的順序，其附屬的 2nd Source 會**永遠緊隨**其 Main Item，不受排序影響。
- 能夠在表格上**直接編輯**零件資訊（包含 Main Source 和 2nd Source）。
- 能夠**新增或刪除** 2nd Source。

### 4.3 Excel 匯入/匯出

#### 匯入
- 支援讀取 `.xls` 與 `.xlsx` 格式
- 支援**拖曳開啟**檔案，同時支援點擊按鈕選擇檔案
- 匯入邏輯詳見 [4.3.1 Excel BOM 匯入規則](#431-excel-bom-匯入規則)

#### 匯出
- 僅支援寫入 `.xlsx` 格式
- 匯出邏輯詳見 [4.3.2 Excel BOM 匯出規則](#432-excel-bom-匯出規則)

### 4.4 版本比較
- 比較同一專案不同版本的 BOM
- 標示新增、刪除、修改的項目

---

### 4.3.1 Excel BOM 匯入規則

#### 表頭解析（bom_revisions 資料）

從 Excel 固定儲存格讀取，並解析冒號後的文字內容：

| 儲存格 | 欄位 | 解析規則 | 範例 |
|--------|------|----------|------|
| B3 | project_code | 取 `"Product Code: "` 後方文字 | `Product Code: TANGLED` → `TANGLED` |
| B4 | description | 取 `"Description: "` 後方文字 | `Description: MBD,Tangled,...` |
| D3 | schematic_version | 取 `"Schematic Version: "` 後方文字 | `Schematic Version: 1.0` → `1.0` |
| F3 | pcb_version | 取 `"PCB Version: "` 後方文字 | `PCB Version: 2.1` → `2.1` |
| F4 | pca_pn | 取 `"PCA PN: "` 後方文字 | `PCA PN: ABC-123` → `ABC-123` |
| H4 | date | 取 `"Date: "` 後方文字 | `Date: 2026-01-15` → `2026-01-15` |

#### 零件資料解析（parts 資料）

從 **Row 6** 開始往下逐行讀取，欄位對應：

| Excel 欄 | 資料庫欄位 | 說明 |
|----------|-----------|------|
| A | `item` | 項目編號 |
| B | `hhpn` | 公司內部料號 |
| E | `description` | 零件描述 |
| F | `supplier` | 供應商名稱 |
| G | `supplier_pn` | 供應商料號 |
| I | `location` | 零件位置編號（逗號分隔） |
| J | `ccl` | 是否為 Critical Part |
| L | `remark` | 註記 |

#### Main Source vs 2nd Source 判斷邏輯

- 若 `item` 和 `location` **都有值** → 此行為 **Main Source**
- 否則 → 此行為 **2nd Source**，歸屬於上一個讀到的 Main Source 群組

#### Location 原子化處理

讀取到 Main Source 時，將 `location` 欄位以 `","` 分隔拆開為單一位置編號，逐一存入 `parts` 表。

#### 工作表（Sheet）讀取順序與規則

**階段一：讀取製程頁面（Type 頁面）**

| 工作表名稱 | `type` 值 | `bom_status` 預設值 |
|-----------|----------|-------------------|
| SMD | SMD | I |
| PTH | PTH | I |
| BOTTOM | BOTTOM | I |

- 工作表名稱即為該頁面所有零件的 `type`
- 這三個頁面的零件 `bom_status` 一律先設定為 `I`（上件）

**階段二：讀取狀態頁面（BOM Status 頁面）**

| 工作表名稱 | `bom_status` 值 | `type` 值 |
|-----------|----------------|----------|
| NI | X（不上件） | 留空 |
| PROTO | P（Proto Part） | 留空 |
| MP | M（MP Only） | 留空 |

- 工作表名稱對應 `bom_status`（NI→X, PROTO→P, MP→M）
- `type` 留空

**覆蓋與新增規則：**

系統需先依據零件分佈判斷 Mode (NPI / MP)，再依據下列規則處理狀態頁面的零件：

1.  **NI 頁面**：
    -   若零件已存在於 Phase 1 (SMD/PTH/BOTTOM)，**新增**一筆 `bom_status=X` 的紀錄（不覆蓋原紀錄）。
    -   若零件不存在，則新增一筆 `bom_status=X` 的紀錄。

2.  **PROTO 頁面**：
    -   若 Mode = **NPI**：
        -   若零件已存在於 Phase 1，**覆蓋**原紀錄的 `bom_status` 為 `P`。
        -   若零件不存在，則新增一筆 `bom_status=P` 的紀錄。
    -   若 Mode = **MP**：
        -   若零件已存在於 Phase 1，**新增**一筆 `bom_status=P` 的紀錄（保留原紀錄，例如 `I`）。
        -   若零件不存在，則新增一筆 `bom_status=P` 的紀錄。

3.  **MP 頁面**：
    -   若 Mode = **MP**：
        -   若零件已存在於 Phase 1，**覆蓋**原紀錄的 `bom_status` 為 `M`。
        -   若零件不存在，則新增一筆 `bom_status=M` 的紀錄。
    -   若 Mode = **NPI**：
        -   若零件已存在於 Phase 1，**新增**一筆 `bom_status=M` 的紀錄（保留原紀錄，例如 `I`）。
        -   若零件不存在，則新增一筆 `bom_status=M` 的紀錄。

#### NPI / MP 模式判斷邏輯 (Mode Determination)

在匯入過程中，需根據零件分佈自動判斷 `bom_revisions.mode`：

1.  **NPI 模式**：若 `PROTO` 頁面中有任何零件 **同時出現在** `SMD`、`PTH` 或 `BOTTOM` 任一頁面中。
    - 意即：原型階段零件被包含在主製程中。
2.  **MP 模式**：若 `MP` 頁面中有任何零件 **同時出現在** `SMD`、`PTH` 或 `BOTTOM` 任一頁面中。
    - 意即：量產專用零件被包含在主製程中。
3.  **預設 (NPI)**：若上述兩者皆不成立（例如 PROTO 與 MP 頁面皆空，或皆無交集）。

---

### 4.3.2 Excel BOM 匯出規則

#### 支援格式
- 匯出格式為 `.xlsx`。
- 格式參考：`references/bom_templates/TANGLED_EZBOM_SI_0.3_BOM_20240627_0900WithProtoPart(compared).xls`

#### 輸出 Sheet 清單
1.  **Changelist** (佔位，待 Phase 7 實作)
2.  **Changelist by Sheet** (佔位，待 Phase 7 實作)
3.  **ALL**: 完整 BOM (依 Mode 過濾)
4.  **SMD**: 僅 SMD 製程 (依 Mode 過濾)
5.  **PTH**: 僅 PTH 製程 (依 Mode 過濾)
6.  **BOTTOM**: 僅 BOTTOM 製程 (依 Mode 過濾)
7.  **NI**: 不上件清單 (依 Mode 過濾)
8.  **PROTO**: 原型階段上件清單
9.  **MP**: 量產階段上件清單
10. **CCL**: 關鍵零件清單 (Critical Component List)

#### 頁面內容過濾邏輯

| Sheet | 顯示條件 (Filter Logic) |
|-------|-------------------------|
| **ALL** | 若 Mode=NPI，顯示 `bom_status` 為 `I` 或 `P` 的零件<br>若 Mode=MP，顯示 `bom_status` 為 `I` 或 `M` 的零件 |
| **SMD** | Type=`SMD` **且** (若 Mode=NPI 顯示 `I`+`P`; 若 Mode=MP 顯示 `I`+`M`) |
| **PTH** | Type=`PTH` **且** (若 Mode=NPI 顯示 `I`+`P`; 若 Mode=MP 顯示 `I`+`M`) |
| **BOTTOM** | Type=`BOTTOM` **且** (若 Mode=NPI 顯示 `I`+`P`; 若 Mode=MP 顯示 `I`+`M`) |
| **NI** | 若 Mode=NPI，顯示 `bom_status` 為 `X` 或 `M` 的零件<br>若 Mode=MP，顯示 `bom_status` 為 `X` 或 `P` 的零件 |
| **PROTO** | 顯示 `bom_status` 為 `P` 的零件 |
| **MP** | 顯示 `bom_status` 為 `M` 的零件 |
| **CCL** | 顯示 `ccl` 為 `Y` 的零件 |

#### 表頭格式 (Header)

所有 Sheet (除了 Changelist 類) 皆採用統一表頭格式：

- **A1:M2**: 跨欄置中，垂直置中
    - 第一行: "FUJIN PRECISION INDUSTRY(SHENZHEN) CO.,LTD"
    - 第二行: "BILL OF MATERIAL"
- **B3**: "Product Code: " + `project_code`
- **B4**: "Description: " + `description`
- **D3**: "Schematic Version: " + `schematic_version`
- **F3**: "PCB Version: " + `pcb_version`
- **F4**: "PCA PN: " + `pca_pn`
- **H3**: "BOM Version: " + `bom_version` (如 0.3)
- **H4**: "Date: " + `bom_date`
- **J3**: "Phase: " + `bom_phase`

#### 資料表格 (Data Table)

- **Row 5**: 欄位名稱
    - Item, HH PN, STD PN, GRP PN, Description, Supplier, Supplier PN, Qty, Location, CCL, Lead Time, Remark, Comp Approval
- **Row 6+**: 零件資料 (Main Source + 2nd Sources)

#### 樣式要求
- **BOM Group**: 不同 Group (Main Items) 之間使用**條紋底色**交替切換 (Zebra Striping by Group)，以區分不同零件群組。
- **邊框**: BOM 資料區域需有細框線 (Thin Border)。
- **格線**: 其餘無內容區域隱藏 Excel 格線。

---

## 5. 資料庫設計

### 5.1 儲存策略
- 每個**系列**存成一個獨立的 **`.bomix`** 檔案（實際為 SQLite 格式）
- 使用者可以手動備份、搬移、分享資料庫檔案
- 零件採用**原子化儲存**：每一行紀錄一個 location 的原始資訊

### 5.2 主要資料表

| 資料表 | 說明 |
|--------|------|
| `series_meta` | 系列設定與資訊（description、建立/修改時間、未來擴充設定） |
| `projects` | 專案資訊 |
| `bom_revisions` | BOM 版本（合併 Phase + Version，含 schematic/PCB 版本等） |
| `parts` | 原子化零件紀錄（一個 location 一行） |
| `second_sources` | 替代料（透過邏輯鍵關聯零件群組） |

> 詳細 Schema 定義請參考 [DATABASE.md](DATABASE.md)

---

## 6. 技術規格

| 項目 | 規格 |
|------|------|
| 執行環境 | Electron |
| UI 框架 | React + Tailwind CSS |
| 表格元件 | TanStack Table |
| 建置工具 | Vite（electron-vite） |
| 狀態管理 | Zustand |
| 資料庫 | SQLite（better-sqlite3） |
| Excel 處理 | xlsx（SheetJS） |
| 打包工具 | electron-builder |
| 目標平台 | Windows（安裝版 + 便攜版） |
| 資料庫副檔名 | .bomix |

---

## 7. UI 設計規格

### 7.1 設計風格

採用 **Windows 11 Fluent Design** 現代化風格：

| 設計元素 | 規格 |
|---------|------|
| 圓角 | 所有卡片、按鈕、對話框使用 `rounded-xl`（12px） |
| 陰影 | 卡片與浮動元素使用 `shadow-sm` / `shadow-md` |
| 間距 | 統一使用 4px 網格系統（Tailwind spacing scale） |
| 字型 | Segoe UI / Microsoft JhengHei / system-ui |
| 圖標 | 使用 emoji 或可替換為 Lucide React icons |
| 動畫 | 使用 CSS transition（`transition-all duration-200`） |
| 模糊背景 | 標題列使用 `backdrop-blur-sm` 達成透明感 |

### 7.2 主題系統

- **系統主題偵測**：啟動時讀取系統主題設定（Dark/Light）
- **手動切換**：提供 Dark/Light 模式切換按鈕（位於標題列右側）
- **即時套用**：切換後透過 CSS class 即時更新所有 UI 元件外觀
- **記憶設定**：儲存使用者偏好至 `%APPDATA%/BOMIX/settings.json`
- **色彩方案**：透過 Tailwind CSS `dark:` variant 實現

| 模式 | 主色調 | 背景 | 文字 |
|------|--------|------|------|
| Light | `primary-600` | 白/淺灰 | 深灰/黑 |
| Dark | `primary-400` | 深灰/黑 | 白/淺灰 |

### 7.3 應用程式佈局

```
┌──────────────────────────────────────────────────┐
│  標題列：BOMIX - [系列名稱]  [🏠/📊/🔄/⚙️] [🌙/☀️] │
├──────────────────────────────────────────────────┤
│                                                  │
│            主內容區域                            │
│                                                  │
│  ┌─────────────────────────────────┐             │
│  │                                 │             │
│  │    儀表板 / BOM / 比較 / 設定    │             │
│  │                                 │             │
│  └─────────────────────────────────┘             │
│                                                  │
└──────────────────────────────────────────────────┘
```

#### 標題列與導航
- **頂部導航**：導航圖標整合於視窗標題列左側或中間。
- **頁面項目**：儀表板 (Dashboard)、BOM 檢視、版本比較、設定。
- **視窗標題**：動態顯示「BOMIX - [系列名稱]」。

#### 主內容區域
- 根據導航選擇動態切換頁面元件。
- 支援頁面間的淡入淡出動畫。

### 7.4 儀表板 (Dashboard)

- **未開啟系列時**：
  - 顯示歡迎畫面與快速操作（建立/開啟系列）。
  -顯示最近開啟的系列列表。
- **開啟系列後**：
  - **系列資訊**：顯示名稱、描述（可編輯）。
  - **樹狀視圖**：階層式顯示 `系列 -> 專案 -> BOM`。
  - **編輯功能**：
    - 專案：編輯代碼與描述。
    - BOM：編輯屬性 (Mode, Date, Suffix, Desc)。
  - **導航**：點擊 BOM 版本直接跳轉至 BOM 功能頁面。

### 7.5 About 對話框

透過設定頁面觸發，以 Modal 呈現：

| 項目 | 內容 |
|------|------|
| 應用程式名稱 | BOMIX |
| 版本號 | 讀取 `package.json` version |
| 描述 | BOM 變化管理與追蹤工具 |
| 授權 | MIT License |
| GitHub 連結 | 可點擊開啟瀏覽器 |
| 技術資訊 | Electron 版本、Node.js 版本 |

### 7.6 Change Log 顯示

- 讀取專案的 `CHANGELOG.md` 檔案（打包時置於 `extraResources`）
- 以 Markdown 格式渲染於 Modal 中
- 若檔案不存在，顯示「暫無更新記錄」
- 可從 About 對話框中開啟，或從設定頁面存取
