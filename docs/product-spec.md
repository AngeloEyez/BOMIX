# BOMIX 產品規格書 — Wave 1

> **版本**: 1.0-draft  
> **最後更新**: 2026-07-16  
> **範圍**: Wave 1（導入 / 儲存 / 分析 / 導出）  
> **技術棧**: Go + Wails v3 + Vue 3 + modernc.org/sqlite + GORM

---

## 目錄

1. [執行摘要](#1-執行摘要)
2. [Wave 規劃](#2-wave-規劃)
3. [技術架構](#3-技術架構)
4. [資料模型](#4-資料模型)
5. [系統核心模組](#5-系統核心模組)
6. [功能需求 — Wave 1](#6-功能需求--wave-1)
7. [Excel 匯入規範](#7-excel-匯入規範)
8. [Excel 匯出規範](#8-excel-匯出規範)
9. [UI 設計規格 — Wave 1](#9-ui-設計規格--wave-1)
10. [API 介面規範](#10-api-介面規範)
11. [測試策略](#11-測試策略)
12. [部署與建置](#12-部署與建置)
13. [附錄](#附錄)

---

## 1. 執行摘要

### 1.1 產品定位

BOMIX 是一款專業級 Windows 桌面應用程式，專為電子製造業設計，用於：

- 讀入不同格式的 Excel BOM 檔案（EBOM / BigMatrix / Matrix）
- 將 BOM 資料統一儲存至 SQLite 單一資料檔（`.bomx`）
- 彈性分析 BOM 資料，產生各種視圖
- 根據 Excel Template 匯出 BigMatrix / Matrix 格式的 BOM 表

### 1.2 核心價值

| 價值點 | 說明 |
|--------|------|
| **統一儲存** | 所有 BOM 資料集中於單一 `.bomx` 檔案，便於備份與搬移 |
| **多格式支援** | 支援 EBOM / BigMatrix / Matrix 三種 Excel 格式匯入 |
| **彈性分析** | 多種視圖切換，動態聚合 Main/2nd Source 資料 |
| **Matrix 管理** | 支援多 Model 的物料選擇與驗證狀態追蹤 |
| **Template 輸出** | 根據預定義 Excel Template 精準輸出格式化報表 |

### 1.3 目標用戶

- 電子製造業的 BOM 管理人員
- 硬體工程師與採購人員
- 專案管理與供應鏈團隊

---

## 2. Wave 規劃

### 2.1 Wave 1 — 核心功能（本規格書範圍）

| 模組 | 功能 | 優先級 |
|------|------|--------|
| 資料庫管理 | 建立/開啟/關閉 `.bomx` 資料庫 | P0 |
| Excel 匯入 | 支援 EBOM / BigMatrix / Matrix 三種格式自動辨識與導入 | P0 |
| 資料分析 | BOM 聚合視圖、多種過濾條件 | P0 |
| Excel 匯出 | BigMatrix / Matrix 兩種格式（根據 Template 輸出） | P0 |
| 簡易 UI | 基礎操作介面（匯入/匯出/視圖切換/任務狀態） | P0 |
| 非同步任務 | Excel 讀寫等大型操作的非同步執行與狀態管理 | P0 |
| 日誌系統 | 支援 info/warning/error 分級，任務狀態追蹤 | P0 |
| 設定系統 | 設定檔（可選）+ 程式內預設值 | P1 |

### 2.2 Wave 2 — 未來 Roadmap（不在本規格書範圍）

- 完整的 UI BOM 表管理介面（編輯/新增/刪除零件）
- 版本比較（Diff 功能）
- 拖曳匯入
- 儀表板（Dashboard）
- Matrix 狀態燈號與健康度
- 多專案/系列管理
- EBOM 匯出

---

## 3. 技術架構

### 3.1 技術棧

| 層級 | 技術 | 版本 | 備註 |
|------|------|------|------|
| **Runtime** | Wails | v3 | Go + Webview 桌面框架 |
| **Backend** | Go | 1.21+ | 核心業務邏輯 |
| **Frontend** | Vue 3 | 3.4+ | Composition API + `<script setup>` |
| **UI 元件** | PrimeVue | v4 | 完整元件庫 |
| **樣式** | Tailwind CSS | v4 | utility-first CSS |
| **資料庫驅動** | modernc.org/sqlite | 1.38+ | 純 Go 實現，無需 CGO |
| **ORM** | GORM | 1.25+ | 開發效率優先 |
| **Excel 引擎** | excelize/v2 | 2.8+ | xlsx 讀寫 |
| **建構工具** | Vite | 5.0+ | 前端開發伺服器 |
| **目標平台** | Windows | — | 單一 `.exe` 可執行檔 |
| **資料庫副檔名** | `.bomx` | — | SQLite 格式 |

### 3.2 專案結構

請參考工作區根目錄下的 [claude.md](../claude.md) 文件（Section 2），其中定義了最完整且具權威性的目錄結構。開發代理程式（如 Claude Code）將嚴格遵守該檔案中定義的結構來建立與放置檔案。

### 3.3 核心架構原則

#### 3.3.1 SOLID 設計

請嚴格遵循 SOLID 原則開發，確保 Reader 與 Writer 介面具備良好的擴充性。

#### 3.3.2 資料流程

```
┌─────────────┐     ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  Excel File │ ──> │   Detector   │ ──> │    Reader    │ ──> │   Database   │
│ (.xls/.xlsx)│     │ (格式辨識)    │     │ (格式專用)    │     │   (.bomx)    │
└─────────────┘     └──────────────┘     └──────────────┘     └──────────────┘
                                                                      │
                                                                      v
┌─────────────┐     ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│ Excel Output│ <── │    Writer    │ <── │  Aggregator  │ <── │   Query DB   │
│   (.xlsx)   │     │ (Template)   │     │  (聚合/過濾)  │     │              │
└─────────────┘     └──────────────┘     └──────────────┘     └──────────────┘
```

---

## 4. 資料模型

### 4.1 BOM 零件基本屬性

| 欄位 | 資料庫欄位名 | 說明 | 範例 |
|------|-------------|------|------|
| Item | `item` | 項目編號，從 Excel 導入，僅作紀錄 | 1, 2, 3 |
| HHPN | `hhpn` | 公司內部料號 | 34065Y600-GRT-H |
| Supplier | `supplier` | 供應商名稱 | Samsung, Murata |
| Supplier PN | `supplier_pn` | 供應商料號 | CL05B104KO5NNNC |
| Description | `description` | 零件描述 | CAP,22uF,+/-20%,X5R,6.3V,SMD0603 |
| Location | `location` | 零件位置編號（原子化，每行一個） | R1, XU1, C3 |
| Type | `type` | 製程類型，可為空 | SMD, PTH, BOTTOM |
| BOM Status | `bom_status` | 上件狀態 | I, X, P, M |
| CCL | `ccl` | 是否為 Critical Part | Y, N |
| Remark | `remark` | 註記資料 | — |

#### BOM 上件狀態碼

| 狀態碼 | 說明 |
|--------|------|
| **I** | 上件（Install） |
| **X** | 不上件 |
| **P** | Proto Part，僅原型階段上件 |
| **M** | MP Only，僅量產上件 |

### 4.2 BOM Main Item（聚合視圖）

- 相同 **Supplier + Supplier PN** 的零件合併為一個 BOM Main Item
- 可再根據 main source 的 **Type** 、 **BOM Status** 、 **CCL** 拆分
- 位置編號以逗號分隔合併（`C1,C2,C3,C5`）
- 數量由位置編號的數量自動計算
- 此為聚合視圖，資料庫中仍以原子化儲存

### 4.3 Second Source（替代料）

每個 BOM Main Item 可附帶 **0 到多個** Second Source：

| 欄位 | 資料庫欄位名 | 說明 | 範例 |
|------|-------------|------|------|
| HHPN | `hhpn` | 公司內部料號 | 34065Y600-GRT-H |
| Supplier | `supplier` | 供應商名稱 | Yageo |
| Supplier PN | `supplier_pn` | 供應商料號 | RC0402FR-0710KL |
| Description | `description` | 零件描述 | CAP,22uF,... |

- Second Source **不包含** Location、Type、BOM Status 等欄位
- 透過 `(bom_revision_id, main_supplier, main_supplier_pn)` 邏輯鍵關聯至主料群組

### 4.4 資料層次結構

```
Series (.bomx 資料庫檔案)
├── series_meta              # 系列元資料
├── Project A
│   ├── BOM Revision 0.1     # Phase + Version
│   │   ├── Parts            # 原子化零件紀錄
│   │   ├── Second Sources   # 替代料
│   │   └── Matrix Models    # Model 定義 + 選擇紀錄
│   ├── BOM Revision 0.2
│   └── ...
└── Project B
    └── ...
```

### 4.5 Phase 與版本

- Phase 名稱 **可自訂**（常見：DB, SI, PV, MVB, EVT, DVT, PVT 等）
- 版本格式：`{major}.{minor}`（例：0.1, 0.2, 1.0）
- 同一專案內 Phase + Version 組合唯一 (bom_revision_id)

### 4.6 Matrix BOM 資料模型

#### 4.6.1 核心屬性與規則

Matrix BOM 在資料模型中是圍繞以下核心規則設計：

1. **附屬關係**：Matrix 資訊必須依附於特定專案的特定 BOM 版本（BOM Revision）下。
2. **Model 組成**：
   - 每個 BOM 版本可以包含多個 Model。
   - Model 數量預設為 **3 個**，最少為 **2 個**，可視需求動態擴充。
   - Model 的命名固定為大寫英文字母，依序為 `A`, `B`, `C`... 等。
3. **Model 屬性 (Qty)**：
   - 每個 Model 帶有一個屬性 `qty`（打件數量）。
   - `qty` 必須為**正整數**（代表該 Model 於生產/打件時的實際需求數量）。
4. **選擇限制與互斥性 (Selections)**：
   - 針對每一個 BOM Main Item 零件群組（包含其 Main Source 和所有 Second Sources），使用者在同一個 Model 中，**必須且只能勾選其中的一個物料**（可以選擇勾選 Main Source 或是其中一個替代料 2nd Source）。
   - 該選擇關係在同一個 Model、同一個零件群組中具有**單選互斥性**。

---

## 5. 系統核心模組

### 5.1 非同步任務系統

所有 Excel 讀寫或大型操作均透過非同步任務系統執行，避免阻塞 UI。

#### 5.1.1 任務生命週期

```
Created → Queued → Running → Completed / Failed / Cancelled
```

#### 5.1.2 任務結構

```go
// Task 代表一個非同步任務
type Task struct {
    ID          string       // UUID
    Name        string       // 人類可讀名稱（如 "匯入 EBOM: TANGLED_DB_0.1.xlsx"）
    Type        TaskType     // Import / Export / Analysis
    Status      TaskStatus   // Created / Queued / Running / Completed / Failed / Cancelled
    Progress    float64      // 0.0 ~ 1.0
    Message     string       // 當前步驟描述
    Error       string       // 錯誤訊息（若失敗）
    CreatedAt   time.Time
    StartedAt   time.Time
    CompletedAt time.Time
    Result      interface{}  // 任務結果
}
```

#### 5.1.3 任務管理器

| 功能 | 說明 |
|------|------|
| Submit(task) | 提交任務至佇列 |
| Cancel(taskID) | 取消執行中的任務 |
| GetStatus(taskID) | 取得任務狀態 |
| ListTasks() | 列出所有任務 |
| OnProgress(callback) | 註冊進度回呼 |
| OnComplete(callback) | 註冊完成回呼 |
| OnError(callback) | 註冊錯誤回呼 |

#### 5.1.4 Callback 機制

- 使用 Wails v3 Event System 將任務狀態即時推送至前端
- 前端透過 `window.runtime.EventsOn` 監聽事件

```go
// 事件定義
const (
    EventTaskProgress  = "task:progress"   // 進度更新
    EventTaskComplete  = "task:complete"   // 任務完成
    EventTaskFailed    = "task:failed"     // 任務失敗
    EventTaskCancelled = "task:cancelled"  // 任務取消
)
```

### 5.2 日誌系統

基於 Go `slog` 的結構化日誌系統，支援分級與 UI 顯示。

#### 5.2.1 日誌等級

| 等級 | 用途 | UI 顯示 |
|------|------|---------|
| `DEBUG` | 開發除錯資訊 | 僅在開發模式顯示 |
| `INFO` | 一般操作記錄 | ℹ️ 藍色 |
| `WARN` | 警告訊息 | ⚠️ 黃色 |
| `ERROR` | 錯誤訊息 | ❌ 紅色 |

#### 5.2.2 日誌格式

```go
// 結構化日誌範例
slog.Info("Excel 匯入完成",
    slog.String("file", "TANGLED_DB_0.1.xlsx"),
    slog.String("format", "EBOM"),
    slog.Int("parts_count", 156),
    slog.Duration("elapsed", elapsed),
)
```

#### 5.2.3 日誌緩衝

- 維護一個環形緩衝區（預設 500 條），供 UI 即時顯示
- 前端透過 Wails 綁定方法讀取日誌列表
- 支援依等級過濾

#### 5.2.4 任務日誌與狀態追蹤

- **即時任務日誌**：日誌系統必須支援顯示非同步任務的狀態。
- **動態更新**：當一個任務被創建時，日誌列表中會出現一條對應的任務日誌紀錄，並包含以下資訊：
  - **任務名稱**
  - **任務當下資訊**（會隨著任務進度即時動態更新）
  - **任務狀態**（例如：創建、執行中、完成、失敗等）
- **整合顯示**：任務的生命週期變化直接反映在日誌清單中，讓使用者能清楚追蹤每個大型操作的進度與最終結果。

### 5.3 設定系統

#### 5.3.1 設計原則

1. **自動建立**：第一次啟動時，若設定檔不存在，系統會自動建立設定檔，並寫入預設值。
2. **預設值定義**：程式內建完整預設值，統一定義在 `config/defaults.go` 中。
3. **合併策略**：若設定檔存在，則讀取設定檔的內容。設定檔中有覆蓋到的項目採用設定檔的值，沒有設定到的項目則自動退回使用程式內的預設值。
4. **設定檔格式**：TOML

#### 5.3.2 設定檔路徑

```
%APPDATA%/BOMIX/config.toml    （使用者全域設定）
```

#### 5.3.3 設定項目

```go
// defaults.go — 所有預設值集中管理
var DefaultConfig = Config{
    // 一般設定
    Language:        "zh-TW",
    Theme:           "system",     // system / light / dark

    // 匯入設定
    Import: ImportConfig{
        ConfirmOverwrite: false,   // 匯入重複 BOM 時是否彈窗確認
    },

    // 日誌設定
    Logger: LoggerConfig{
        Level:          "INFO",
        BufferSize:     500,        // 環形緩衝區大小
        FileLogging:    false,      // 是否寫入檔案
    },

    // 最近開啟
    RecentFiles: RecentFilesConfig{
        MaxCount:       10,
    },
}
```

---

## 6. 功能需求 — Wave 1

### 6.1 資料庫管理

#### 6.1.1 基本操作

| 操作 | 說明 |
|------|------|
| 建立新 Series | 選擇儲存路徑，建立新 `.bomx` 檔案並初始化 Schema |
| 開啟 Series | 選擇既有 `.bomx` 檔案開啟 |
| 關閉 Series | 釋放資料庫連線 |
| 最近開啟 | 記錄並顯示最近開啟的 `.bomx` 檔案列表 |

#### 6.1.2 自動初始化

開啟或建立 `.bomx` 檔案時，自動執行 GORM AutoMigrate 建立/更新資料表結構。

### 6.2 專案管理（Wave 1 簡化版）

Wave 1 中的專案管理為簡化版本：

| 操作 | 說明 |
|------|------|
| 自動建立 | 匯入 Excel 時根據表頭 project_code 自動建立專案 |
| 列表顯示 | 在側邊欄或下拉選單中顯示專案列表 |
| 選擇切換 | 切換目前操作的專案 |

> **完整的 CRUD 介面延至 Wave 2 實作。**

### 6.3 Excel 匯入

#### 6.3.1 支援格式與偵測規則 (Detector)

系統支援三種 BOM 格式的匯入。在讀取 Excel 檔案時，會由 Detector 自動依據 Excel 工作表（Sheet）的結構或特定儲存格內容來辨識格式。

```go
// BOMFormat 列舉
type BOMFormat string

const (
    FormatEBOM      BOMFormat = "EBOM"
    FormatBigMatrix BOMFormat = "BigMatrix"
    FormatMatrix    BOMFormat = "Matrix"
    FormatUnknown   BOMFormat = "Unknown"
)

// Detect 根據 Excel 檔案結構判斷 BOM 格式
func Detect(file *excelize.File) BOMFormat {
    // 判斷邏輯：
    // 1. 檢查 Sheet 名稱清單是否符合特定格式的特徵
    // 2. 檢查特定儲存格內容
    // 3. 回傳判斷結果
}
```

##### 格式辨識與偵測規則對照表

| 格式 | 偵測條件 / 識別方式 | 說明 |
|------|--------------------|------|
| **EBOM** | 同時包含 `SMD`、`PTH`、`BOTTOM`、`NI`、`PROTO`、`MP` Sheet；且 SMD/PTH/BOTTOM 表頭 H5=`Qty`、J7=`CCL` | 標準 BOM 格式 |
| **BigMatrix** | 存在 sheet `BigMatrix` | 多專案整合的 Matrix BOM |
| **Matrix** | 存在 sheet `SMD` 且 SMD 表頭 H5=`Location`、J7=`Total Set` | 單一專案 Matrix BOM |

#### 6.3.2 匯入流程 (支援多檔案批次匯入)

當使用者選擇或拖曳單個或多個 Excel 檔案時，系統會依次執行以下流程：

1. **辨識格式**：對每個選擇的檔案進行格式辨識。若格式判定為 `Unknown`，則將該檔案名稱記錄於排除清單中並直接 **Bypass**（跳過）。
2. **啟動非同步任務**：針對所有成功識別格式的檔案，系統會分別為其建立非同步任務，並啟動對應格式的 Reader。
3. **錯誤提示彈窗**：當所有有效檔案都已順利創建任務後，若排除清單中存在被 Bypass 的 `Unknown` 格式檔案，系統會以**彈窗（Popup）通知使用者**哪些檔案因格式無法辨識而未被匯入。
4. **解析與儲存**：各 Reader 任務在後端平行/序列解析資料，並寫入 SQLite 資料庫中。
5. **匯入完成與更新**：當個別檔案任務完成後，透過非同步回呼機制通知前端 UI 即時重新整理與載入資料。

### 6.4 BOM 資料分析

#### 6.4.1 聚合邏輯

將原子化的零件資料聚合為 Main Item 視圖：

1. 以 `(supplier, supplier_pn)` 為群組鍵
2. 合併同群組的 Location 為逗號分隔字串
3. Qty 自動計算（Location 數量）
4. 附加該群組的所有 Second Source

#### 6.4.2 視圖過濾

| 視圖 | 過濾條件 |
|------|----------|
| **ALL** | 排除 `bom_status = X`，依 Mode 過濾（NPI 顯示 I+P，MP 顯示 I+M） |
| **SMD** | `type = SMD` 且符合 Mode 過濾 |
| **PTH** | `type = PTH` 且符合 Mode 過濾 |
| **BOTTOM** | `type = BOTTOM` 且符合 Mode 過濾 |
| **NI** | NPI: `bom_status = X 或 M`；MP: `bom_status = X 或 P` |
| **PROTO** | `bom_status = P` |
| **MP** | `bom_status = M` |
| **CCL** | `ccl = Y` |

### 6.5 Excel 匯出

#### 6.5.1 支援格式

| 格式 | Template | 說明 |
|------|----------|------|
| **BigMatrix** | `template/bigmatrix.xlsx` | 多專案整合的 Matrix BOM |
| **Matrix** | `template/matrix.xlsx` | 單一專案 Matrix BOM |

#### 6.5.2 匯出流程

當使用者執行匯出作業時，系統會依以下流程處理：

1. **選擇格式與來源**：使用者選擇匯出格式（BigMatrix 或 Matrix），並勾選一個或多個要匯出的 BOM Revisions。
   - **Model 匯出原則**：不論是 BigMatrix 還是 Matrix，匯出時均會取該 BOM 表中**最大的 Model 數量全部匯出**，而非只匯出單一 Model。
   - **BigMatrix Model 數量調整**：
     1. 若選擇 BigMatrix 匯出，介面需提供選項供使用者調整（增減）每個 BOM 表所要匯出的 Model 數量。預設值為該 BOM 已存在的最長 Model 數量。若使用者將數量調大（例如：原本只有 3 個 Model，選擇匯出 4 個），則第 4 個 Model 會按照範本格式繪製，但其內容格留空。
     2. 介面提供一個可輸入description的欄位, 選填。匯出的BigMatrix的 description欄位會填入使用者輸入的description。
2. **啟動非同步任務與處理策略**：
   - **BigMatrix 格式**：系統僅會啟動**一個**非同步任務。該任務會自資料庫讀取所有被勾選的 BOM Revisions 資料，並將它們**整合合併**匯出至單一 BigMatrix Excel 報表中。
   - **Matrix 格式**：系統會為每個勾選 the BOM Revision **分別啟動獨立的非同步任務**，各別讀取資料並依 Template 輸出為多個單一專案的 Matrix Excel 檔案。
3. **設定儲存位置**：
   - **單一檔案匯出**（如 BigMatrix 或僅選單個 Matrix）：使用者透過檔案儲存對話框選擇具體的儲存檔案路徑。
   - **多檔案批次匯出**（如多選 Matrix）：使用者選擇一個輸出目錄（OutputDir），系統自動依據 BOM 屬性命名並輸出檔案至該目錄。
4. **解析與填寫**：對應格式的 Writer 自資料庫讀取 BOM 資料（與對應的 Matrix Model 勾選狀態），並將數據寫入內嵌的 Excel 範本。
5. **匯出完成與通知**：各個任務執行完畢後，透過 Wails 事件發送完成回呼以通知 UI 更新進度或提示成功。

#### 6.5.3 匯出參數

```go
// ExportOptions 匯出選項
type ExportOptions struct {
    Format              BOMFormat      // BigMatrix / Matrix
    RevisionIDs         []string       // 需匯出的 BOM Revision ID 列表（支援多選）
    OutputPath          string         // 單一檔案的輸出路徑（單檔匯出時使用，如 BigMatrix）
    OutputDir           string         // 輸出的目標目錄（批次匯出多檔案時使用，如多選 Matrix）
    ModelCountOverrides map[string]int // 各 Revision ID 對應的 Model 匯出數量（僅適用於 BigMatrix，鍵為 RevisionID，值為自訂 Model 數量；未指定則預設使用資料庫中最大數量）
}
```

---

## 7. Excel 匯入規範

### 7.0 重複 BOM 匯入處理規則

當匯入的 BOM（以 `project`, `phase`, `version` 作為唯一識別）已經存在於資料庫中時，系統應依據以下規則執行：

1. **覆蓋策略**：以新匯入的 Excel 資料完全覆蓋（Overwrite）資料庫中原有的 BOM 資料（包括原子化零件 `parts`、`second_sources` 及相關的 `matrix_selections` 等）。
2. **彈窗確認控制**：
   - 系統是否彈窗提示使用者確認覆蓋，將由應用程式設定檔（`config.toml`）中的 `[import]` 區塊設定決定。
   - **`confirmOverwrite = false`（預設值）**：自動以新資料覆蓋，不彈窗打擾使用者。
   - **`confirmOverwrite = true`**：在執行覆蓋前，系統應彈出確認對話框（Confirm Dialog），待使用者點擊確認後才執行覆蓋；若使用者取消，則 Bypass 該檔案的匯入。

### 7.1 EBOM 匯入規則

#### 7.1.1 表頭解析（bom_revisions 資料）

| 儲存格 | 欄位 | 解析規則 | 範例 |
|--------|------|----------|------|
| B3 | project_code | 取 `"Product Code: "` 後方文字 | `Product Code: TANGLED` → `TANGLED` |
| B4 | description | 取 `"Description: "` 後方文字 | `Description: MBD,Tangled,...` |
| D3 | schematic_version | 取 `"Schematic Version: "` 後方文字 | `Schematic Version: 1.0` → `1.0` |
| F3 | pcb_version | 取 `"PCB Version: "` 後方文字 | `PCB Version: 2.1` → `2.1` |
| F4 | pca_pn | 取 `"PCA PN: "` 後方文字 | `PCA PN: ABC-123` → `ABC-123` |
| H4 | date | 取 `"Date: "` 後方文字 | `Date: 2026-01-15` → `2026-01-15` |

#### 7.1.2 零件資料解析

從 **Row 6** 開始往下逐行讀取：

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

#### 7.1.3 Main Source vs 2nd Source 判斷

- 若 `item` 和 `location` **都有值** → **Main Source**
- 否則 → **2nd Source**，歸屬於上一個 Main Source 群組

#### 7.1.4 Location 原子化處理

Main Source 的 `location` 欄位以 `","` 分隔，逐一存入 `parts` 表。

#### 7.1.5 Sheet 讀取順序與規則

**階段一：讀取製程頁面**

| Sheet 名稱 | `type` 值 | `bom_status` 預設值 |
|-----------|----------|-------------------|
| SMD | SMD | I |
| PTH | PTH | I |
| BOTTOM | BOTTOM | I |

**階段二：讀取狀態頁面**

| Sheet 名稱 | `bom_status` 值 | `type` 值 |
|-----------|----------------|----------|
| NI | X（不上件） | 留空 |
| PROTO | P（Proto Part） | 留空 |
| MP | M（MP Only） | 留空 |

**階段二覆蓋與新增規則：**

1. **NI 頁面**：
   - 若零件已存在，**新增**一筆 `bom_status=X` 的紀錄 （不覆蓋原紀錄）
   - 若零件不存在，新增一筆 `bom_status=X` 的紀錄

2. **PROTO 頁面**：
   - Mode=NPI：若零件已存在則**覆蓋** `bom_status=P`；不存在則新增
   - Mode=MP：若零件已存在則**新增**一筆 `bom_status=P`；不存在則新增

3. **MP 頁面**：
   - Mode=MP：若零件已存在則**覆蓋** `bom_status=M`；不存在則新增
   - Mode=NPI：若零件已存在則**新增**一筆 `bom_status=M`；不存在則新增

#### 7.1.6 NPI / MP 模式判斷

在匯入過程中，系統會依據以下條件自動判斷該 BOM 版本的模式（`bom_revisions.mode`）：

- **NPI**：若 `PROTO` 頁面中有任何零件**同時出現在** `SMD`、`PTH` 或 `BOTTOM` 任一頁面中。
- **MP**：不滿足上述 NPI 條件的所有其他情況（即 `PROTO` 頁面無零件與主製程重疊，則判定為量產模式 `MP`）。

### 7.2 BigMatrix 匯入規則

#### 7.2.1 Sheet 結構
BigMatrix BOM 的 sheet, 固定讀取 `BigMatrix` sheet.

#### 7.2.2 表頭解析

| 儲存格 | 欄位 | 解析規則 | 範例 |
|--------|------|----------|------|
| B3 | bom_count | 取 `"BOMs: "` 後方數字 | `BOMs: 4` → 4 |
| B4 | description | 取 `"Description: "` 後方文字 | `Description: MBD,Tangled,...` |
| E4 | date | 取 `"Date: "` 後方文字 | `Date: 2026-01-15` → `2026-01-15` |

##### 7.2.2.1 橫向多 BOM 與 Model 動態列解析演算法

BigMatrix 格式的工作表中，自 **H 欄** 開始向右排列了多份 BOM 表的 Model 選擇欄位。每份 BOM 的寬度（欄數）由其 Model 數量決定。系統應採用以下演算法遍歷並解析：

1. **第一份 BOM 表資訊**：
   - 起始欄定位在 **H 欄**。
   - `H2` 儲存格：第一份 BOM 的 `project_code` (專案代碼)。
   - `H3` 儲存格：第一份 BOM 的 `revision_id` (格式為 `Phase-Version`，例如 `PV-0.3`。需拆分解析出 `PhaseName = PV`，`Version = 0.3`)。
   - `H4` 儲存格：第一份 BOM 的第一個 Model 名稱（預設通常為 `A`，右側依序為 `B`、`C`...）。
   - `H5` 儲存格：第一份 BOM 的第一個 Model 的打件數量 (qty)。

2. **Model 數量與跨度計算**：
   - 自起始欄的第 4 列（Row 4，如 `H4`）開始向右遍歷，依序讀取 Model 名稱（`A`, `B`, `C`...）。
   - 當**再次遇到名稱為 `A` 的儲存格**時，即代表上一份 BOM 表的 Model 遍歷已結束，且此 `A` 所在的欄位即為**下一份 BOM 表的起始欄**。
   - **範例**：
     - 若第一份 BOM 有 3 個 Model：
       - `H4` = `A`, `I4` = `B`, `J4` = `C`
       - `K4` = `A` (此為下一份 BOM 的起點)
     - 此時可得知：
       - 第一份 BOM 的 Model 跨度為 H 至 J 欄，總共有 3 個 Model (A, B, C)。
       - **第二份 BOM** 的起始欄為 **K 欄**。
       - 第二份 BOM 的 `project_code` 儲存於 `K2`，`revision_id` 儲存於 `K3`。

3. **遞迴解析**：
   - 重複上述步驟向右遍歷，直到解析出的 BOM 數量達到 `B3` 中 `project_count` 的設定值。


#### 7.2.3 零件與 Model 資料解析

從 **Row 6** 開始往下逐行讀取，解析零件基本欄位與各 BOM 的 Model 選擇狀態：

##### 7.2.3.1 零件基本欄位對應（A 至 G 欄）

| Excel 欄 | 資料庫欄位 | 說明 |
|----------|-----------|------|
| A | `item` | 項目編號 |
| B | `hhpn` | 公司內部料號 |
| C | `description` | 零件描述 |
| D | `supplier` | 供應商名稱 |
| E | `supplier_pn` | 供應商料號 |
| F | `qty` | 打件數量 |
| G | `location` | 零件位置編號（逗號分隔） |

##### 7.2.3.2 Main Source vs 2nd Source 判斷

- 若 `item` 和 `location` **都有值** → **Main Source**
- 否則 → **2nd Source**，歸屬於上一個讀取到的 Main Source 群組。

##### 7.2.3.3 Model 選擇狀態解析（H 欄往右）

自 **H 欄** 開始向右為動態的 Model 選擇欄位。系統應依據 `7.2.2.1` 計算出的每份 BOM 列跨度與 Model 數量，依序向右檢查：

1. **欄位定位**：
   - 遍歷當前 Row 的每一欄，依據欄位索引對應至其所屬的 BOM Revision 與其對應的 Model（如 A, B, C...）。
2. **勾選識別 ("V" / "v")**：
   - 檢查儲存格內容是否為 **`"V"`** 或 **`"v"`**（不區分大小寫）。
   - 若為 `"V"` 或 `"v"`，代表此 Model 選擇了當前這列的物料（不論是 Main Source 還是 2nd Source）。
3. **儲存至資料庫**：
   - 若被勾選，系統需建立 `MatrixSelection` 記錄：
     - `ModelID`：對應當前 BOM 版本的 `MatrixModel`。
     - `MainSupplier` / `MainSupplierPn`：當前物料群組中 **Main Source** 的 Supplier 與 Supplier PN（作為 Group Key 關聯）。
     - `SelectedSupplier` / `SelectedSupplierPn`：當前被選中這列的物料之 Supplier 與 Supplier PN。
     - `IsAutoSelected`：設定為 `false`。

#### 7.2.4 BigMatrix 匯入規則:

BigMatrix 匯入時不建立新的物料, 僅更新資料庫中每份BOM的matrix 勾選狀態以及 每個model的qty, 若有增減model數量, 則同步於database中調整。

### 7.3 Matrix 匯入規則

place holder, 暫時不支援 Matrix BOM 匯入, 但在程式碼中保留 placeholder 以便後續擴充

---

## 8. Excel 匯出規範

### 通用匯出格式定義

- **物料群組底色交替**：為容易區分不同物料群組，物料群組之間的儲存格底色採用兩種顏色交互出現。
- **底色定義**：底色 1 和底色 2 定義於設定檔中，預設底色 1 為白色，底色 2 為淡米黃。

### 8.1 BigMatrix 匯出規則

#### 8.1.1 Template 檔案

- 路徑：`template/bigmatrix.xlsx`
- 嵌入方式：`//go:embed`

#### 8.1.2 輸出 Sheet 結構

- 固定輸出至名為 **`BigMatrix`** 的 Sheet。

#### 8.1.3 表頭寫入

| 儲存格 | 寫入內容 | 範例 |
|--------|----------|------|
| B3 | `"BOMs: "` + 匯出的 BOM 數量 | `BOMs: 4` |
| B4 | `"Description: "` + 匯出的專案描述文字 | `Description: MBD,Tangled,...` |
| E4 | `"Date: "` + 匯出日期 | `Date: 2026-01-15` |


#### 8.1.4 橫向多 BOM 與 Model 欄位寫入

自 **H 欄** 開始，依序向右寫入每份 BOM 的 Model 欄位：

1. **每份 BOM 的表頭資訊**（以起始欄 `X` 表示）：
   - `X2`：寫入 `project_code`（專案代碼）
   - `X3`：寫入 `revision_id`（格式為 `Phase-Version`，例如 `PV-0.3`）
   - `X4` 起向右：依序寫入各 Model 名稱（`A`, `B`, `C`...）
   - `X5` 起向右：依序寫入各 Model 的打件數量 (qty)

2. **欄位跨度計算**：
   - 第一份 BOM 起始於 H 欄，佔用的欄數 = 該 BOM 的 Model 數量。
   - 下一份 BOM 的起始欄 = 上一份 BOM 起始欄 + 該 BOM 的 Model 數量。
   - 依序排列直到所有 BOM 寫入完成。

#### 8.1.5 零件資料寫入

從 **Row 6** 開始往下逐行寫入零件資料。

##### 8.1.5.1 零件基本欄位

| Excel 欄 | 資料庫欄位 | 說明 |
|----------|-----------|------|
| A | `item` | 項目編號 |
| B | `hhpn` | 公司內部料號 |
| C | `description` | 零件描述 |
| D | `supplier` | 供應商名稱 |
| E | `supplier_pn` | 供應商料號 |
| F | `qty` | 打件數量 |
| G | `location` | 零件位置編號（逗號分隔） |

因為物料列表是多份BOM聚合的結果, qty與location以第一份BOM的資料為主,若第一份BOM無此物料, 往後尋找下一個BOM直到找到對應物料的資料。

##### 8.1.5.2 Main Source / 2nd Source 排列

- **Main Source** 行：`item` 與 `location` 欄位皆填入值。
- **2nd Source** 行：`item` 與 `location` 欄位留空，緊接在其所屬的 Main Source 行之後。

##### 8.1.5.3 Model 選擇狀態寫入（H 欄往右，對應 7.2.3.3）

- 依據 `8.1.4` 計算出的每份 BOM 欄位範圍與 Model 對應關係，定位至正確的儲存格。
- 若該 Model 對該物料存在 `MatrixSelection` 記錄（即被勾選），寫入 **`"V"`**。
- 若未被勾選，儲存格留空。
- 若該物料在對應的BOM中完全不存在, 則將該儲存格背景填滿灰色
- 若該物料群組只有main source, 無2nd source, 則相對應的model 位置自動寫入 "V"

#### 8.1.6 零件排序規則

按照資料庫原始順序（維持匯入時的順序）。

#### 8.1.7 匯出參數

```go
// BigMatrixExportParams BigMatrix 匯出參數
type BigMatrixExportParams struct {
    RevisionIDs []string // 欲匯出的 BOM Revision ID 列表（需透過 RevisionID 關聯查詢所屬的 project_code）
    Description string   // 使用者在介面上輸入的選填 Description
}
```

#### 8.1.8 樣式規範

- **物料群組樣式**：物料群組按照「通用匯出格式定義」的樣式規範處理（底色交替）。
- **BOM 表分隔線**：在執行 `8.1.4` 橫向多 BOM 與 Model 欄位寫入時，每一個 BOM 表之間的垂直框線需加粗，以區分不同的 BOM 版本。

### 8.2 Matrix 匯出規則

> [!IMPORTANT]
> 此區塊待使用者提供 Matrix Template 與格式定義後補充。

#### 8.2.1 Template 檔案

- 路徑：`template/matrix.xlsx`
- 嵌入方式：`//go:embed`

#### 8.2.2 輸出 Sheet 清單

<!-- TODO: 待補充 -->

#### 8.2.3 欄位對應

<!-- TODO: 待補充 -->

#### 8.2.4 匯出參數

```go
// MatrixExportParams Matrix 匯出參數
type MatrixExportParams struct {
    RevisionID  string   // BOM Revision ID
    ModelIDs    []string // 匯出的 Model 列表
    // TODO: 待補充其他必要參數
}
```

#### 8.2.5 樣式規範

<!-- TODO: 待補充 -->

---

## 9. UI 設計規格 — Wave 1

### 9.1 設計原則

Wave 1 的 UI 為**簡潔實用**導向，提供核心操作所需的最小完整介面。

| 設計元素 | 規格 |
|---------|------|
| **風格** | Windows 11 Fluent Design |
| **主題** | 支援 Light / Dark 切換（PrimeVue 內建主題） |
| **字型** | Segoe UI / Microsoft JhengHei / system-ui |
| **元件庫** | PrimeVue v4 |
| **樣式** | Tailwind CSS v4 |
| **響應式** | 支援視窗縮放自適應 |

### 9.2 頁面佈局

```
┌──────────────────────────────────────────────────────┐
│  [Logo] BOMIX - [Series Name]          [🌙/☀️] [⚙️]  │  ← 標題列
├──────────┬───────────────────────────────────────────┤
│          │                                           │
│  側邊欄   │              主內容區域                    │
│          │                                           │
│  ┌─────┐ │  ┌──────────────────────────────────────┐ │
│  │ 專案 │ │  │                                      │ │
│  │ 清單 │ │  │     BOM 表格 / 匯入匯出操作區          │ │
│  │      │ │  │                                      │ │
│  │      │ │  │                                      │ │
│  │      │ │  └──────────────────────────────────────┘ │
│  │      │ │                                           │
│  │      │ │  ┌──────────────────────────────────────┐ │
│  └─────┘ │  │     任務狀態 / 日誌面板                 │ │
│          │  └──────────────────────────────────────┘ │
└──────────┴───────────────────────────────────────────┘
```

### 9.3 Wave 1 頁面清單

| 頁面 | 功能 |
|------|------|
| **首頁 / 歡迎** | 建立/開啟 `.bomx`、最近開啟列表 |
| **BOM 檢視** | 表格顯示聚合視圖、視圖切換（ALL/SMD/PTH/...） |
| **匯入操作** | 選擇檔案、格式確認、匯入進度 |
| **匯出操作** | 選擇格式/Revision/Model、匯出進度 |
| **任務面板** | 底部面板，顯示任務列表與狀態 |
| **日誌面板** | 底部面板（與任務面板共用或分頁），顯示日誌訊息 |
| **設定** | 主題切換、匯入/匯出預設設定 |

### 9.4 主要元件

| 元件 | 用途 | PrimeVue 元件 |
|------|------|--------------|
| `BOMTable` | BOM 聚合表格 | DataTable / VirtualScroller |
| `ImportDialog` | 匯入操作對話框 | Dialog + FileUpload |
| `ExportDialog` | 匯出操作對話框 | Dialog + Dropdown |
| `TaskPanel` | 任務狀態面板 | Panel + ProgressBar |
| `LogPanel` | 日誌訊息面板 | Panel + 自訂列表 |
| `SeriesSelector` | 系列選擇器 | Dropdown / Listbox |
| `ViewSwitcher` | 視圖切換器 | TabMenu / SelectButton |

---

## 10. API 介面規範

### 10.1 Wails 綁定方法

```go
// App Wails 綁定主結構
type App struct {
    db     *gorm.DB
    config *Config
    tasks  *TaskManager
    logger *Logger
}
```

#### 10.1.1 Series 管理

```go
CreateSeries(name, description, filePath string) (*Series, error)
OpenSeries(filePath string) (*Series, error)
CloseSeries() error
GetSeriesInfo() (*SeriesMeta, error)
GetRecentSeries() ([]RecentFile, error)
```

#### 10.1.2 Project 管理

```go
GetProjects() ([]Project, error)
GetProject(id string) (*Project, error)
```

#### 10.1.3 BOM Revision 管理

```go
GetBOMRevisions(projectID string) ([]BOMRevision, error)
GetBOMRevision(id string) (*BOMRevision, error)
```

#### 10.1.4 BOM 資料查詢

```go
GetBOMParts(revisionID string, view string) ([]AggregatedPart, error)
GetBOMSummary(revisionID string) (*BOMSummary, error)
```

#### 10.1.5 Excel 匯入/匯出

```go
ImportExcel(filePath string) (string, error)           // 回傳 taskID
ImportExcelWithFormat(filePath string, format string) (string, error)
ExportExcel(options ExportOptions) (string, error)     // 回傳 taskID
```

#### 10.1.6 任務管理

```go
GetTaskStatus(taskID string) (*Task, error)
GetAllTasks() ([]Task, error)
CancelTask(taskID string) error
```

#### 10.1.7 日誌查詢

```go
GetLogs(level string, limit int) ([]LogEntry, error)
ClearLogs() error
```

#### 10.1.8 設定管理

```go
GetSettings() (*Config, error)
UpdateSettings(settings map[string]interface{}) error
```

### 10.2 事件定義

| 事件名稱 | 方向 | 承載資料 | 說明 |
|----------|------|----------|------|
| `task:progress` | Backend → Frontend | `{taskID, progress, message}` | 任務進度更新 |
| `task:complete` | Backend → Frontend | `{taskID, result}` | 任務完成 |
| `task:failed` | Backend → Frontend | `{taskID, error}` | 任務失敗 |
| `task:cancelled` | Backend → Frontend | `{taskID}` | 任務已取消 |
| `log:new` | Backend → Frontend | `{level, message, timestamp}` | 新日誌訊息 |
| `bom:updated` | Backend → Frontend | `{revisionID}` | BOM 資料已更新 |

### 10.3 錯誤型別

```go
var (
    ErrNotFound         = errors.New("resource not found")
    ErrInvalidFormat    = errors.New("invalid file format")
    ErrDuplicate        = errors.New("duplicate entry")
    ErrInvalidMode      = errors.New("invalid mode: must be NPI or MP")
    ErrFormatDetectFail = errors.New("unable to detect BOM format")
    ErrTaskCancelled    = errors.New("task was cancelled")
    ErrDatabaseClosed   = errors.New("database is not open")
)
```

---

## 11. 測試策略

### 11.1 測試層級

| 層級 | 工具 | 範圍 |
|------|------|------|
| **單元測試** | Go `testing` + Table-Driven | 業務邏輯、聚合、過濾 |
| **整合測試** | Go `testing` + SQLite in-memory | 資料庫操作 |
| **前端測試** | Vitest + Vue Test Utils | Vue 元件 |

### 11.2 測試重點（Wave 1）

| 模組 | 測試項目 |
|------|----------|
| **Detector** | 三種格式的正確辨識 |
| **Reader (EBOM)** | 表頭解析、零件解析、Main/2nd 判斷、Mode 判斷 |
| **Reader (BigMatrix)** | <!-- TODO --> |
| **Reader (Matrix)** | <!-- TODO --> |
| **Aggregator** | 群組合併、Qty 計算、Location 合併 |
| **Filter** | 各視圖的過濾正確性 |
| **Writer (BigMatrix)** | Template 填入、樣式正確性 |
| **Writer (Matrix)** | Template 填入、樣式正確性 |
| **TaskManager** | 生命週期、取消、回呼 |

### 11.3 測試命名規範

```bash
# Go 測試
go test ./backend/excel/...
go test ./backend/processor/...
go test ./backend/task/...

# 前端測試
npm run test
```

---

## 12. 部署與建置

### 12.1 開發環境

```bash
# 安裝依賴
cd bomix-app
go mod tidy
cd frontend && npm install && cd ..

# 開發模式（Hot Reload）
wails3 dev
```

### 12.2 生產建置

```bash
# Windows 建置（單一 exe）
wails3 build -platform windows/amd64
```

### 12.3 輸出產物

| 項目 | 說明 |
|------|------|
| **輸出路徑** | `build/bin/BOMIX.exe` |
| **嵌入資源** | 前端靜態資源 + Excel Template（`//go:embed`） |
| **外部依賴** | 無（單一執行檔） |

### 12.4 版本管理

- 遵循 Semantic Versioning
- 版本號定義在 `wails.json`

---

## 附錄

### A. 術語表

| 術語 | 定義 |
|------|------|
| **BOM** | Bill of Materials，物料清單 |
| **EBOM** | Engineering BOM，工程物料清單（標準多 Sheet 格式） |
| **Main Source** | 主要供應商物料 |
| **Second Source / 2nd Source** | 替代料 |
| **BigMatrix** | 大型 Matrix BOM 格式（<!-- TODO: 待補充定義 -->） |
| **Matrix** | Matrix BOM 格式（<!-- TODO: 待補充定義 -->） |
| **Phase** | 開發階段（如 DB, DVT, PVT） |
| **Series** | 系列，一個 `.bomx` 資料庫檔案代表一個系列 |
| **NPI** | New Product Introduction，新產品導入 |
| **MP** | Mass Production，量產 |
| **CCL** | Critical Component List，關鍵零件清單 |
| **HHPN** | 公司內部料號 |

### B. 資料庫 Schema（GORM Model）

```go
package model

import "time"

// Series 系列元資料（.bomx 檔案）
type Series struct {
    ID          string    `gorm:"primaryKey"`
    Name        string    `gorm:"uniqueIndex"`
    Description string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// Project 專案
type Project struct {
    ID          string       `gorm:"primaryKey"`
    SeriesID    string       `gorm:"index:idx_project_series"`
    Code        string       // 專案代碼（如 TANGLED）
    Description string
    CreatedAt   time.Time
    Series      Series       `gorm:"foreignKey:SeriesID"`
    Revisions   []BomRevision `gorm:"foreignKey:ProjectID"`
}

// BomRevision BOM 版本（Phase + Version）
type BomRevision struct {
    ID              string    `gorm:"primaryKey"`
    ProjectID       string    `gorm:"index:idx_revision_project"`
    PhaseName       string    // DB, EVT, DVT, PVT...
    Version         string    // 0.1, 0.2, 1.0
    Mode            string    `gorm:"default:NPI"` // NPI | MP
    ProjectCode     string    // 快取，來自 Excel 表頭
    Description     string    // 快取，來自 Excel 表頭
    SchematicVer    string
    PcbVer          string
    PcaPn           string
    BomDate         string
    SourceFormat    string    // 匯入時的來源格式（EBOM / BigMatrix / Matrix）
    SourceFile      string    // 匯入時的來源檔名
    Metadata        string    `gorm:"type:text"` // JSON 儲存額外屬性
    CreatedAt       time.Time
    UpdatedAt       time.Time
    Project         Project   `gorm:"foreignKey:ProjectID"`
}

// Part 原子化零件紀錄
type Part struct {
    ID              string    `gorm:"primaryKey"`
    RevisionID      string    `gorm:"index:idx_part_revision"`
    Item            int       // Excel 項目編號
    Hhpn            string    `gorm:"index:idx_part_hhpn"`
    Supplier        string
    SupplierPn      string    `gorm:"index:idx_part_supplier_pn"`
    Description     string
    Location        string    // 單一位置（原子化）
    Type            string    // SMD, PTH, BOTTOM
    BomStatus       string    `gorm:"default:I"` // I, X, P, M
    Ccl             string    `gorm:"default:N"`
    Remark          string
    CreatedAt       time.Time
    Revision        BomRevision `gorm:"foreignKey:RevisionID"`
}

// SecondSource 替代料
type SecondSource struct {
    ID              string    `gorm:"primaryKey"`
    RevisionID      string    `gorm:"index:idx_2nd_revision"`
    MainSupplier    string    // 關聯主料的 Supplier
    MainSupplierPn  string    // 關聯主料的 Supplier PN
    Hhpn            string
    Supplier        string
    SupplierPn      string
    Description     string
    CreatedAt       time.Time
    Revision        BomRevision `gorm:"foreignKey:RevisionID"`
}

// MatrixModel Matrix 模型定義
type MatrixModel struct {
    ID              string    `gorm:"primaryKey"`
    RevisionID      string    `gorm:"index:idx_matrix_revision"`
    Name            string    // A, B, C...
    Qty             int       `gorm:"default:1"` // 打件數量（正整數）
    SortOrder       int       // 排序順序
    CreatedAt       time.Time
    Revision        BomRevision `gorm:"foreignKey:RevisionID"`
}

// MatrixSelection Matrix 勾選紀錄
type MatrixSelection struct {
    ID              string    `gorm:"primaryKey"`
    ModelID         string    `gorm:"index:idx_sel_model"`
    MainSupplier    string    // 群組的 Main Supplier
    MainSupplierPn  string    // 群組的 Main Supplier PN
    SelectedSupplier   string // 被選中物料的 Supplier
    SelectedSupplierPn string // 被選中物料的 Supplier PN
    IsAutoSelected  bool      // 是否為自動選擇（只有 Main 無 2nd）
    UpdatedAt       time.Time
    Model           MatrixModel `gorm:"foreignKey:ModelID"`
}
```

#### 資料庫索引

```go
// 額外複合索引（於 AutoMigrate 後建立）
db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_project_series_code ON projects(series_id, code)")
db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_revision_phase_ver ON bom_revisions(project_id, phase_name, version)")
db.Exec("CREATE INDEX IF NOT EXISTS idx_part_group ON parts(revision_id, supplier, supplier_pn)")
db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_2nd_unique ON second_sources(revision_id, main_supplier, main_supplier_pn, supplier, supplier_pn)")
db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_matrix_model_name ON matrix_models(revision_id, name)")
db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_sel_unique ON matrix_selections(model_id, main_supplier, main_supplier_pn)")
```

### C. 設定檔範例（config.toml）

```toml
# 一般設定
language = "zh-TW"
theme = "dark"

[import]
confirmOverwrite = false


[logger]
level = "INFO"
bufferSize = 500
fileLogging = false

[recentFiles]
maxCount = 10
```

### D. 待補充項目清單

> [!WARNING]
> 以下項目需要使用者提供資料或決策後才能補充完整。

| 項目 | 狀態 | 需要的資料 |
|------|------|-----------|
| BigMatrix 匯入規則（§7.2） | ⏳ 待補充 | BigMatrix 範例 Excel 檔案 + 格式說明 |
| Matrix 匯入規則（§7.3） | ⏳ 待補充 | Matrix 範例 Excel 檔案 + 格式說明 |
| BigMatrix 偵測條件（§6.3.3） | ⏳ 待補充 | 偵測特徵（Sheet 結構/特定儲存格） |
| Matrix 偵測條件（§6.3.3） | ⏳ 待補充 | 偵測特徵 |
| BigMatrix 匯出規則（§8.1） | ⏳ 待補充 | BigMatrix Template + 欄位定義 |
| Matrix 匯出規則（§8.2） | ⏳ 待補充 | Matrix Template + 欄位定義 |
| BigMatrix vs Matrix 差異定義 | ⏳ 待補充 | 兩種格式的區別說明 |

---

*本文檔由 BOMIX 開發團隊維護*  
*Wave 1 規格書 — 最後更新：2026-07-16*
