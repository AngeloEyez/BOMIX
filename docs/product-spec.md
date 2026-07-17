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
    Description         string         // 使用者輸入的描述文字（選填，僅適用於 BigMatrix）
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
EBOM 匯入時，在database建立或覆蓋相對應的BOM revision資料以及零件資訊

#### 7.1.1 表頭解析（bom_revisions 資料）

bom revision基本資料由 "SMD" sheet 的表頭取得。

| 儲存格 | 欄位 | 解析規則 | 範例 |
|--------|------|----------|------|
| B3 | project_code | 取 `"Product Code: "` 後方文字 | `Product Code: TANGLED` → `TANGLED` |
| B4 | description | 取 `"Description: "` 後方文字 | `Description: MBD,Tangled,...` |
| D3 | schematic_version | 取 `"Schematic Version: "` 後方文字 | `Schematic Version: 1.0` → `1.0` |
| D4 | phase_name | 取 `"Phase: "` 後方文字 | `Phase: DB` → `DB` |
| F3 | pcb_version | 取 `"PCB Version: "` 後方文字 | `PCB Version: 2.1` → `2.1` |
| F4 | pca_pn | 取 `"PCA PN: "` 後方文字 | `PCA PN: ABC-123` → `ABC-123` |
| H3 | version | 取 `"BOM Version: "` 後方文字 | `BOM Version: 0.1` → `0.1` |
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

所有匯出功能皆遵循以下三大設計原則：

1. **範本驅動 (Template-Driven)**：所有匯出格式（包含 BigMatrix 與 Matrix）皆須使用事先設計好的 Excel 範本 (`.xlsx`) 作為基底。程式碼內部**不得硬編碼（Hardcode）任何樣式資訊**（例如：字型大小、框線粗細、背景顏色等），所有外觀設計皆由範本決定。

2. **標籤置換法 (Tag Replacement)**：範本中的靜態儲存格若需填入動態資料（如描述、日期、BOM 數量等），應在範本中預先寫入**佔位標籤**（例如 `{{.Description}}`）。程式在匯出時遍歷所有儲存格，找到標籤字串後以真實資料取代。

3. **樣式參考原型 (Style Prototypes)**：針對需要動態生成的欄或列，程式應從範本中預先設計好的「原型欄 / 原型列」**複製（Clone）儲存格樣式**到新產生的目標儲存格，以確保外觀一致性。

- **物料群組底色交替**：兩種交替底色（底色 1 / 底色 2）在範本的 Row 6 和 Row 7 先定義好，程式交替複製該列樣式即可。

### 8.1 BigMatrix 匯出規則

#### 8.1.1 Template 檔案

- 路徑：`template/bigmatrix.xlsx`
- 嵌入方式：`//go:embed`

#### 8.1.2 輸出 Sheet 結構

- 固定輸出至名為 **`BigMatrix`** 的 Sheet。

#### 8.1.3 表頭寫入（標籤置換法）

範本表頭區域的儲存格預先填入以下佔位標籤，程式在匯出時掃描並逐一取代：

| 範本佔位標籤 | 取代內容說明 | 取代後範例 |
|-------------|----------|----------|
| `{{.BOMCount}}` | 匯出的 BOM 數量 | `4` |
| `{{.Description}}` | 使用者輸入的描述文字 | `MBD,Tangled,...` |
| `{{.Date}}` | 匯出當日日期 | `2026-01-15` |

> **範本設計範例**：B3 儲存格的文字可設計為 `BOMs: {{.BOMCount}}`，匯出後自動變為 `BOMs: 4`。

#### 8.1.4 橫向多 BOM 與動態 Model 欄位寫入（三欄樣式繼承規則）

由於每份 BOM 的 Model 數量不固定，H 欄以後的 Model 欄位必須在匯出時動態生成。為了讓每份 BOM 之間有粗框線視覺分隔，範本中以 **H、I、J 三欄**作為樣式參考原型：

| 範本原型欄 | 角色 | 邊框規格 |
|-----------|------|----------|
| **H 欄**（Start Column） | 每份 BOM 的最左欄（第 1 個 Model） | 左邊界為**粗框線**，右邊界為細框線 |
| **I 欄**（Inner Column） | 每份 BOM 的中間欄（第 2 至 N-1 個 Model） | 左、右邊界均為細框線 |
| **J 欄**（End Column） | 每份 BOM 的最右欄（第 N 個 Model） | 左邊界為細框線，右邊界為**粗框線** |

**程式寫入邏輯：**

1. 計算第一份 BOM 的 Model 數量 N，從 H 欄開始依序生成 N 個欄位：
   - 第 **1** 個 Model 欄：複製 **H 欄**樣式；Row 4 寫入 Model 名稱（`A`），Row 5 寫入 `qty`。
   - 第 **2 至 N-1** 個 Model 欄：皆複製 **I 欄**樣式；Row 4、Row 5 寫入對應名稱與數量。
   - 第 **N** 個 Model 欄（最後一欄）：複製 **J 欄**樣式；寫入對應名稱與數量。

2. **Row 2 / Row 3 動態合併儲存格（Merge Cells）**：
   - `project_code` 與 `revision_id` 屬於整份 BOM 群組共用的標題，應視覺上橫跨該份 BOM 的所有 N 個欄位。
   - 程式應在生成 N 個欄位後，對 Row 2 與 Row 3 各自執行跨欄合併：合併範圍為 `[起始欄, Row 2]` 至 `[起始欄 + N - 1, Row 2]`，Row 3 同理。
   - 合併後將 `project_code` 寫入合併後第一格（起始欄 Row 2），`revision_id` 寫入合併後第一格（起始欄 Row 3）。
   - 合併後的儲存格樣式複製自範本 **H2**（Row 2）與 **H3**（Row 3）。
   - **範本設計建議**：在範本中可預先將 H2:J2 及 H3:J3 設為合併狀態（代表預設 3 欄的樣板），程式若遇到 N ≠ 3 的情況，應先**解除原有合併、再重新建立正確範圍的合併**。

3. 下一份 BOM 從上一份結束欄的右側開始，重複上述步驟，確保每份 BOM 兩側皆有粗框線。
4. Row 6 以下的資料列，依物料群組交替複製 Row 6（底色 1）或 Row 7（底色 2）的樣式。

> **邊界效果說明**：H 欄的粗左框線 = 前一份 BOM 的 J 欄粗右框線，兩條線緊鄰即形成視覺上的粗分隔線，無需額外設計。

#### 8.1.5 零件資料寫入

從 **Row 6** 開始往下逐行寫入零件資料，按物料群組交替複製範本 Row 6 / Row 7 的列樣式。

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

因為物料列表是多份 BOM 聚合的結果，`qty` 與 `location` 以第一份 BOM 的資料為主；若第一份 BOM 無此物料，則往後尋找下一份 BOM，直到找到對應物料的資料為止。

##### 8.1.5.2 Main Source / 2nd Source 排列

- **Main Source** 行：`item` 與 `location` 欄位皆填入值。
- **2nd Source** 行：`item` 與 `location` 欄位留空，緊接在其所屬的 Main Source 行之後。

##### 8.1.5.3 Model 選擇狀態寫入（H 欄往右）

依據 `8.1.4` 動態生成的欄位對應關係，定位至各 BOM × Model 的儲存格後：

- 若該 Model 對該物料存在 `MatrixSelection` 記錄（即被勾選），寫入 **"V"**。
- 若未被勾選，儲存格留空。
- 若該物料在對應的 BOM 中完全不存在，則將該儲存格背景填滿灰色。
- 若該物料群組只有 Main Source、無 2nd Source，則相對應的 Model 位置自動寫入 **"V"**。

#### 8.1.6 零件篩選條件

匯出至 BigMatrix 時，僅包含符合以下條件的零件：
- `ccl = Y`
- `bom_status = I` 或 `bom_status = P`

#### 8.1.7 零件排序規則

按照資料庫原始順序（維持匯入時的順序）。

#### 8.1.8 匯出參數

```go
// BigMatrixExportParams BigMatrix 匯出參數
type BigMatrixExportParams struct {
    RevisionIDs []string // 欲匯出的 BOM Revision ID 列表（需透過 RevisionID 關聯查詢所屬的 project_code）
    Description string   // 使用者在介面上輸入的選填 Description
}
```

#### 8.1.9 效能設計建議

**In-Memory 模式下的批次寫入優化策略**

為避免逐格呼叫 `SetCellValue()` 造成不必要的開銷，實作時應遵循以下原則：

1. **Phase 1 — 表頭處理（In-Memory，隨機存取）**
   - 執行 Tag 置換（掃描並取代 `{{.xxx}}` 佔位標籤）
   - 動態生成 Model 欄位（複製 H / I / J 樣式）
   - 執行 Row 2 / Row 3 的跨欄合併（`MergeCell`）
   - 完成後**預先取得 Row 6 / Row 7 的 Style ID**，供後續資料行使用。

2. **Phase 2 — 資料行批次寫入（`SetSheetRow`）**
   - 從 Row 6 開始，每次以 `SetSheetRow(sheet, "A6", &row)` 寫入整行資料，**一次呼叫寫入該行所有欄位**，避免逐格呼叫。
   - 套用樣式時直接帶入 Phase 1 預先取得的 Style ID，不在每行重新查找。
   - 物料群組底色交替：奇數群組使用 Row 6 的 Style ID，偶數群組使用 Row 7 的 Style ID。

#### 8.1.10 匯出檔名自動命名規則

BigMatrix 匯出時，預設的 Excel 檔名需遵循以下自動命名規則：

**基礎格式**：`{Series Name}_BigMatrix_{phase}_{Version}_{date}.xlsx`

**動態省略規則**：
因 BigMatrix 會同時匯出多份 BOM Revision，需根據選擇的 BOM 屬性動態調整檔名：
1. **Phase 區段**：若所選的 **所有 BOM Revision** 其 `phase` 皆相同，則檔名保留 `{phase}` 字段；否則檔名將跳過此字段。
2. **Version 區段**：若所選的 **所有 BOM Revision** 其 `phase` 皆相同，**且** `version` 也都相同，則檔名保留 `{Version}` 字段；否則檔名將跳過此字段。

> **範例說明**：
> - 所有 BOM 皆為 `PV` 且為 `0.3` 版：`FY27_BigMatrix_PV_0.3_20260717.xlsx`
> - 所有 BOM 皆為 `PV` 但包含不同版本：`FY27_BigMatrix_PV_20260717.xlsx`
> - 包含 `PV` 與 `EV` 等不同 phase：`FY27_BigMatrix_20260717.xlsx`



### 8.2 Matrix 匯出規則

Matrix 匯出為**單一 BOM** 的 Model 選擇表，與 BigMatrix（橫向多 BOM 聚合）不同。表頭資訊主要來源與 EBOM 相同（參見 `7.1.1`），並以標籤置換法寫入。Model 欄位從 K 欄開始，最少產生 **6 個** Model 欄位。

#### 8.2.1 Template 檔案

- 路徑：`template/matrix.xlsx`
- 嵌入方式：`//go:embed`

#### 8.2.2 輸出 Sheet 結構

根據零件基本屬性 `type` 分別產生三個 sheet：
- **`SMD`**：篩選 `type = SMD` 的零件。
- **`PTH`**：篩選 `type = PTH` 的零件。
- **`BOTTOM`**：篩選 `type = BOTTOM` 的零件。

#### 8.2.3 表頭寫入（標籤置換法）

範本表頭區域的儲存格預先填入以下佔位標籤，程式在匯出時掃描並逐一取代。表頭主要資訊與 EBOM（`7.1.1`）相同：

| 範本佔位標籤 | 取代內容說明 | 對應 EBOM 欄位 | 取代後範例 |
|-------------|----------|--------------|----------|
| `{{.ProjectCode}}` | 專案代碼 | `project_code` | `TANGLED` |
| `{{.Description}}` | BOM 描述 | `description` | `MBD,Tangled,...` |
| `{{.SchematicVersion}}` | 線路圖版本 | `schematic_version` | `1.0` |
| `{{.PCBVersion}}` | PCB 版本 | `pcb_version` | `2.1` |
| `{{.PCAPN}}` | PCA 料號 | `pca_pn` | `ABC-123` |
| `{{.Date}}` | 匯出當日日期 | `date` | `2026-01-15` |

> **範本設計範例**：B3 儲存格的文字可設計為 `Product Code: {{.ProjectCode}}`，匯出後自動變為 `Product Code: TANGLED`。

#### 8.2.4 動態 Model 欄位寫入

Model 欄位從 **K 欄** 開始向右生成，最少產生 **6 個** Model 欄位。若專案實際 Model 數量不足 6 個，仍須產生 6 個欄位。

範本中以 **K 欄** 作為樣式參考原型。每個 Model 欄位僅需複製 K 欄格式即可，不需要區分三種格式。

**程式寫入邏輯：**

1. 計算實際 Model 數量 N（N = max(專案 Model 數, 6)），從 K 欄開始依序生成 N 個欄位：
   - 所有生成的 Model 欄位皆複製 **K 欄**樣式；Row 4 寫入 Model 名稱，Row 5 寫入 `qty`。

2. **Row 4（Model Name）寫入規則**：
   - K4 固定寫入 `Model A`，L4 固定寫入 `Model B`，M4 固定寫入 `Model C`……依序以 `Model` + 空格 + 英文字母命名。
   - **不論該 Model 是否實際存在，只要產生了該欄位，就必須寫入 Model 名稱，不可留空。**
   - 若專案實際 Model 數少於 6 個，超出部分（如無對應資料的 Model）的 Row 5 (`qty`) 留空。

3. **Row 5（Model Qty）寫入規則**：
   - 寫入該 Model 對應的打件數量數字（取自資料庫中 `MatrixModel.qty`）。

4. **Remark 欄位定位**：
   - Remark 欄位固定位於所有 Model 欄位結束後的下一欄。例如：6 個 Model 欄位佔據 K～P 欄，Remark 欄位即為 **Q 欄**。
   - 程式應在匯出時根據實際生成的 Model 欄位數量，動態計算 Remark 欄位的位置。

5. Row 6 以下的資料列，依物料群組交替複製 Row 6（底色 1）或 Row 7（底色 2）的樣式。

#### 8.2.5 零件資料寫入

從 **Row 6** 開始往下逐行寫入零件資料，按物料群組交替複製範本 Row 6 / Row 7 的列樣式。

##### 8.2.5.1 零件基本欄位

| Excel 欄 | 資料庫欄位 | 說明 |
|----------|-----------|------|
| A | `item` | 項目編號 |
| B | `hhpn` | 公司內部料號 |
| C | 無 | 空白 |
| D | `description` | 零件描述 |
| E | `supplier` | 供應商名稱 |
| F | `supplier_pn` | 供應商料號 |
| G | `qty` | 打件數量 |
| H | `location` | 零件位置編號（逗號分隔） |
| I | 無 | 填入公式（見 `8.2.5.3`） |
| J | 無 | 填入公式（見 `8.2.5.4`） |

##### 8.2.5.2 Main Source / 2nd Source 排列

- **Main Source** 行：`item` 與 `location` 欄位皆填入值。
- **2nd Source** 行：`item` 與 `location` 欄位留空，緊接在其所屬的 Main Source 行之後。

##### 8.2.5.3 I 欄公式（總數量計算）

I 欄為 `qty × J 欄` 的乘積公式，用於計算該料件的合計用量。程式應根據**當前資料行號**動態生成公式：

```
=G{row}*J{row}
```

> **範例**：Row 6 的 I 欄寫入 `=G6*J6`，Row 7 寫入 `=G7*J7`，以此類推。

##### 8.2.5.4 J 欄公式（Model 選擇數量加總）

J 欄透過公式檢查各 Model 欄位是否被勾選（`"V"`），若被勾選則加上該 Model 的 Row 5 數量。公式應根據**實際 Model 欄位數量**動態生成，最少涵蓋 6 個 Model 欄位。

通用公式模板（以 Row `{row}` 為例，Model 欄位從 `{col1}` 到 `{colN}`）：

```
=IF(EXACT({col1}{row},"V"),{col1}$5,0)+IF(EXACT({col2}{row},"V"),{col2}$5,0)+...+IF(EXACT({colN}{row},"V"),{colN}$5,0)
```

> **範例**（Row 6，6 個 Model 欄位 K～P）：
> ```
> =IF(EXACT(K6,"V"),K$5,0)+IF(EXACT(L6,"V"),L$5,0)+IF(EXACT(M6,"V"),M$5,0)+IF(EXACT(N6,"V"),N$5,0)+IF(EXACT(O6,"V"),O$5,0)+IF(EXACT(P6,"V"),P$5,0)
> ```

> **注意**：若專案有 8 個 Model（K～R），則公式應延伸至 R 欄，涵蓋所有 Model 欄位。

##### 8.2.5.5 Remark 欄位

Remark 欄位位於所有 Model 欄位結束後的下一欄（參見 `8.2.4` 第 4 點），寫入資料庫的 `remark` 欄位值。


#### 8.2.6 零件篩選條件

匯出至 Matrix 時，除了依據 `type` 分別產生三個 Sheet 外，還需滿足以下過濾條件：
- `ccl = Y`
- `bom_status = I` 或 `bom_status = P`

#### 8.2.7 零件排序規則

按照資料庫原始順序（維持匯入時的順序）。

#### 8.2.8 匯出參數

```go
// MatrixExportParams Matrix 匯出參數
type MatrixExportParams struct {
    RevisionID string // 欲匯出的 BOM Revision ID
}
```

#### 8.2.9 效能設計建議

**In-Memory 模式下的批次寫入優化策略**

為避免逐格呼叫 `SetCellValue()` 造成不必要的開銷，實作時應遵循以下原則：

1. **Phase 1 — 表頭處理（In-Memory，隨機存取）**
   - 執行 Tag 置換（掃描並取代 `{{.xxx}}` 佔位標籤）
   - 動態生成 Model 欄位（複製 K 欄樣式，最少 6 欄），並在此階段同步完成 Model Name (Row 4) 與 Model Qty (Row 5) 的寫入。
   - 完成後**預先取得 Row 6 / Row 7 的 Style ID**，供後續資料行使用。

2. **Phase 2 — 資料行批次寫入（`SetSheetRow`）**
   - 從 Row 6 開始，每次以 `SetSheetRow(sheet, "A6", &row)` 寫入整行資料（A～H 欄），**一次呼叫寫入該行所有基本欄位**，避免逐格呼叫。
   - I 欄與 J 欄透過 `SetCellFormula()` 寫入公式。
   - K 欄以後的 Model 選擇狀態與 Remark 欄位，以 `SetCellValue()` 逐格寫入。
   - 套用樣式時直接帶入 Phase 1 預先取得的 Style ID，不在每行重新查找。
   - 物料群組底色交替：奇數群組使用 Row 6 的 Style ID，偶數群組使用 Row 7 的 Style ID。

#### 8.2.10 匯出檔名自動命名規則

Matrix 匯出時，預設的 Excel 檔名需遵循以下自動命名規則：

**基礎格式**：`{project code}_EZBOM_{phase}_{version}_MatrixBOM_{date}.xlsx`

> **範例說明**：
> 假設專案代碼為 `TANGLED`，階段為 `PV`，版本為 `0.3`，匯出日期為 `20260717`：
> `TANGLED_EZBOM_PV_0.3_MatrixBOM_20260717.xlsx`

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

主視窗採用「上、中、下」三區塊佈局。標題列位於上方，側邊欄與主內容區位於中段，任務狀態與日誌面板橫跨整個視窗底部。

```
┌──────────────────────────────────────────────────────┐
│  [Logo] BOMIX - [Series Name]                  [⚙️]  │  ← 標題列（上區）
├──────────┬───────────────────────────────────────────┤
│          │                                           │
│  側邊欄   │              主內容區域                    │  ← 中區
│          │                                           │
│  ┌─────┐ │  ┌──────────────────────────────────────┐ │
│  │ 專案 │ │  │                                      │ │
│  │ 清單 │ │  │     BOM 表格 / 匯入匯出操作區          │ │
│  │      │ │  │                                      │ │
│  └─────┘ │  └──────────────────────────────────────┘ │
│          │                                           │
├──────────┴───────────────────────────────────────────┤
│                 任務狀態 / 日誌面板                    │  ← 下區
└──────────────────────────────────────────────────────┘
```

**邊界調整與預設值規則：**

1. **側邊欄 (Sidebar)**：
   - 與主內容區之間的邊界可拖曳調整寬度。
   - **最小寬度**：0（側邊欄完全收合，僅保留拖曳把手可見）。
   - **預設寬度**：視窗寬度的 20%。
   - **回復預設**：雙擊拖曳把手可直接回復到 20% 預設寬度。

2. **任務狀態 / 日誌面板 (Bottom Panel)**：
   - 與上方中區塊的邊界可拖曳調整高度。
   - **最小高度**：單行高度（縮至極小即可）。
   - **預設高度**：單行高度。
   - **回復預設**：雙擊拖曳把手可直接回復到單行高度。

> Note: 可以利用 PrimeVue 最強大的 Pass-Through (PT) 特性，直接對內部的拖曳把手（gutter）綁定原生的雙擊事件（onDblclick），並搭配 Vue 3 的 ref 來動態重置尺寸。   

### 9.3 Wave 1 頁面清單

| 頁面 | 功能 |
|------|------|
| **首頁 / 歡迎** | 建立/開啟 `.bomx`、最近開啟列表 |
| **操作區** | 顯示匯入bom的input box，也能接受拖曳檔案匯入。顯示功能操作按鈕，以及一些必要顯示的資訊 |
| **日誌/任務面板** | 底部面板，顯示日誌/任務訊息 |
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
// ImportResult 單一檔案匯入結果
type ImportResult struct {
    FilePath string   // 檔案路徑
    TaskID   string   // 成功建立的 taskID（若 Skipped 為 true 則為空）
    Error    string   // 錯誤訊息（如格式無法辨識）
    Skipped  bool     // 是否因格式不明而跳過
}

ImportExcel(filePaths []string) ([]ImportResult, error)  // 批次匯入多檔，自動偵測格式
ExportExcel(options ExportOptions) ([]string, error)     // 回傳 taskID 列表（BigMatrix 為單一元素，Matrix 多選時為多個）
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

#### 10.1.9 Matrix Model 查詢

```go
GetMatrixModels(revisionID string) ([]MatrixModel, error)
GetMatrixSelections(revisionID string, modelID string) ([]MatrixSelection, error)
```

#### 10.1.10 檔案對話框

> 使用 Wails v3 內建的 Dialog API（如 `OpenFileDialog`、`SaveFileDialog`、`OpenDirectoryDialog` 等）處理檔案選擇與儲存對話框，無需額外定義 Go 綁定方法。

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
| **Reader (BigMatrix)** | 表頭解析（bom_count, description, date）、橫向多 BOM 動態欄解析、Model 數量計算、零件 Main/2nd 判斷、Model 選擇狀態（V/v）解析、僅更新 Matrix 狀態不新增物料 |
| **Reader (Matrix)** | Placeholder 驗證（確認回傳「不支援」錯誤訊息） |
| **Aggregator** | 群組合併、Qty 計算、Location 合併 |
| **Filter** | 各視圖的過濾正確性 |
| **Writer (BigMatrix)** | Template 填入、樣式正確性 |
| **Writer (Matrix)** | Template 填入、樣式正確性 |
| **TaskManager** | 生命週期、取消、回呼 |
| **DB (Series/Project)** | 建立、開啟、關閉、最近開啟列表、專案自動建立 |
| **DB (Revision/Part)** | 建立、查詢、覆蓋匯入、唯一約束、原子化儲存 |
| **DB (Matrix)** | Model 增減同步、Selection 互斥性驗證 |

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

### 12.0 環境前置條件

| 工具 | 版本 | 安裝方式 |
|------|------|----------|
| **Go** | 1.21+ | https://go.dev/dl/ |
| **Node.js** | 18+ | https://nodejs.org/ |
| **Wails CLI** | v3 | `go install github.com/wailsapp/wails/v3/cmd/wails3@latest` |

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
| **BigMatrix** | 多專案聚合的 Matrix BOM 格式，在單一 Sheet 中橫向排列多份 BOM 的 Model 選擇狀態 |
| **Matrix** | 單一專案的 Matrix BOM 格式，包含 SMD/PTH/BOTTOM 三個 Sheet，每個 Sheet 含該專案所有 Model 的選擇狀態 |
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
    Item            string    // Excel 項目編號（字串型別，支援空值與非數字格式）
    Hhpn            string    `gorm:"index:idx_part_hhpn"`
    Supplier        string
    SupplierPn      string    `gorm:"index:idx_part_supplier_pn"`
    Description     string
    Location        string    // 單一位置（原子化）
    Type            string    // SMD, PTH, BOTTOM
    BomStatus       string    `gorm:"default:I"` // I, X, P, M
    Ccl             string    `gorm:"default:N"`
    Remark          string
    SortOrder       int       // 匯入時的排序順序，用於維持原始排列
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
    SortOrder       int       // 匯入時的排序順序，用於維持原始排列
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
    CreatedAt       time.Time
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
db.Exec("CREATE INDEX IF NOT EXISTS idx_part_sort ON parts(revision_id, sort_order)")
db.Exec("CREATE INDEX IF NOT EXISTS idx_2nd_sort ON second_sources(revision_id, main_supplier, main_supplier_pn, sort_order)")
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


---

*本文檔由 BOMIX 開發團隊維護*  
*Wave 1 規格書 — 最後更新：2026-07-16*
