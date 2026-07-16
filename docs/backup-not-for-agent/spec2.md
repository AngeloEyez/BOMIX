# BOMIX 產品規格書 v2.0

> 基於 Wails v3 架構的完整產品規格定義  
> 最後更新：2026-07-15

---

## 目錄

1. [執行摘要](#1-執行摘要)
2. [產品概述](#2-產品概述)
3. [技術架構](#3-技術架構)
4. [功能需求](#4-功能需求)
5. [資料庫設計](#5-資料庫設計)
6. [API 介面規範](#6-api-介面規範)
7. UI 設計規格
8. 資料匯入/匯出規範
9. 測試策略
10. 部署與建置
11. 開發時程規劃

---

## 1. 執行摘要

### 1.1 產品定位

BOMIX 是一款專業級的桌面應用程式，專為電子製造業設計，用於管理與追蹤 BOM（Bill of Materials）的完整生命週期變化。

### 1.2 核心價值

| 價值點 | 說明 |
|--------|------|
| **版本追蹤** | 完整記錄專案各階段的 BOM 演變 |
| **Matrix 管理** | 支援多 Model 的物料選擇與驗證 |
| **精確聚合** | 自動處理 Main Source 與 Second Source 的關聯 |
| **Excel 原生** | 无缝的 Excel 匯入/匯出體驗 |

### 1.3 目標用戶

- 電子製造業的 BOM 管理人員
- 硬體工程師與採購人員
- 專案管理與供應鏈團隊

---

## 2. 產品概述

### 2.1 核心功能清單

| 模組 | 功能 | 優先級 | 狀態 |
|------|------|--------|------|
| 系列管理 | 建立/開啟/關閉 .bomix 資料庫 | P0 | ✅ 已定義 |
| 專案管理 | 新增/編輯/刪除專案 | P0 | ✅ 已定義 |
| BOM 版本 | 管理 Phase + Version 組合 | P0 | ✅ 已定義 |
| BOM 檢視 | 表格化呈現聚合視圖 | P0 | 🔄 開發中 |
| BOM 編輯 | 表格內直接編輯零件 | P0 | 🔄 開發中 |
| Excel 匯入 | 支援 .xls/.xlsx 拖曳匯入 | P0 | 🔄 開發中 |
| Excel 匯出 | 生成格式化的 .xlsx 檔案 | P0 | 🔄 開發中 |
| Matrix Model | 多 Model 物料選擇管理 | P1 | ⏳ 待開發 |
| 版本比較 | 比較不同 BOM 版本的差異 | P1 | ⏳ 待開發 |
| 儀表板 | 專案概覽與 Matrix 狀態 | P1 | ⏳ 待開發 |

### 2.2 資料層次結構

```
Series (.bomix 資料庫檔案)
├── series_meta              # 系列元資料
├── Project A
│   ├── BOM Revision 0.1     # Phase + Version
│   │   ├── Parts            # 原子化零件紀錄
│   │   ├── Second Sources   # 替代料
│   │   └── Matrix Selections# Model 勾選紀錄 (可選)
│   ├── BOM Revision 0.2
│   └── ...
└── Project B
    └── ...
```

---

## 3. 技術架構

### 3.1 技術棧總覽

| 層級 | 技術 | 版本 | 備註 |
|------|------|------|------|
| **Runtime** | Wails | v3 | Go + Webview 桌面框架 |
| **Backend** | Go | 1.21+ | 核心業務邏輯 |
| **Frontend** | Vue 3 | 3.4+ | Composition API |
| **UI 元件** | PrimeVue | 4.0+ | 完整元件庫 |
| **樣式** | Tailwind CSS | 4.0+ |  utility-first |
| **資料庫** | SQLite | 3 | better-sqlite3 |
| **Excel** | excelize/v2 | 2.8+ | xlsx 讀寫 |
| **建構工具** | Vite | 5.0+ | 前端開發伺服器 |

### 3.2 專案結構

```
BOMIX/
├── bomix-app/                 # Wails 應用程式根目錄
│   ├── main.go                # 程式入口與 Wails 初始化
│   ├── wails.json             # Wails 配置
│   ├── go.mod                 # Go 模組定義
│   ├── backend/               # 後端核心邏輯
│   │   ├── app.go             # Wails 綁定與生命週期
│   │   ├── db/                # 資料庫層
│   │   │   ├── connection.go  # SQLite 連接管理
│   │   │   ├── series.go      # Series 資料存取
│   │   │   ├── project.go     # Project 資料存取
│   │   │   ├── bom_revision.go# BOM Revision 資料存取
│   │   │   └── part.go        # Part 資料存取
│   │   ├── excel/             # Excel 處理模組
│   │   │   ├── reader.go      # Excel 匯入邏輯
│   │   │   ├── writer.go      # Excel 匯出邏輯
│   │   │   └── template.go    # 內嵌模板管理
│   │   ├── processor/         # 資料處理模組
│   │   │   ├── logic.go       # 業務邏輯處理
│   │   │   └── aggregator.go  # 資料聚合邏輯
│   │   └── matrix/            # Matrix BOM 模組
│   │       ├── model.go       # Matrix Model 管理
│   │       └── selection.go   # 勾選紀錄管理
│   └── frontend/              # Vue 3 前端應用程式
│       ├── index.html
│       ├── package.json
│       ├── vite.config.ts
│       ├── tailwind.config.js
│       └── src/
│           ├── main.ts        # Vue 應用入口
│           ├── App.vue        # 根元件
│           ├── router/        # Vue Router 配置
│           ├── stores/        # Pinia 狀態管理
│           ├── components/    # 通用元件
│           ├── views/         # 頁面元件
│           │   ├── Dashboard.vue
│           │   ├── BOMView.vue
│           │   ├── CompareView.vue
│           │   └── Settings.vue
│           └── services/      # Wails 綁定呼叫
│               └── api.ts
├── docs/                      # 開發文件
├── scripts/                   # 自動化腳本
└── CLAUDE.md                  # 開發規範
```

### 3.3 核心架構原則

#### 3.3.1 SOLID 設計

| 原則 | 應用 |
|------|------|
| **S**ingle Responsibility | 每個模組僅負責單一職責（如 excel/reader 僅處理匯入） |
| **O**pen/Closed | 透過介面擴充，不修改既有程式碼 |
| **L**iskov Substitution | 使用介面替換具體實作（如 ExcelReader 介面） |
| **I**nterface Segregation | 定義細粒度介面（ExcelReader, ExcelWriter 分開） |
| **D**ependency Injection | 透過建構函式注入依賴 |

#### 3.3.2 資料流程

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│  Excel File │ ──> │  ExcelReader │ ──> │   Parts DB  │
└─────────────┘     └──────────────┘     └─────────────┘
                           │
                           v
                    ┌──────────────┐
                    │  Aggregator  │
                    └──────────────┘
                           │
                           v
                    ┌──────────────┐
                    │   UI Table   │
                    └──────────────┘
```

---

## 4. 功能需求

### 4.1 系列管理 (Series Management)

#### 4.1.1 功能描述

管理 `.bomix` 資料庫檔案的完整生命週期，包含建立、開啟、關閉操作。

#### 4.1.2 使用者操作

| 操作 | 觸發方式 | 預期行為 |
|------|----------|----------|
| 建立新系列 | 點擊「建立新系列」按鈕 | 開啟檔案選擇對話框，選擇儲存位置 |
| 開啟既有系列 | 點擊「開啟系列」按鈕 | 開啟檔案選擇對話框，讀取 .bomix 檔案 |
| 關閉系列 | 點擊「關閉系列」按鈕 | 保存未儲存變更，釋放資料庫資源 |
| 最近開啟 | 儀表板顯示 | 列出最近開啟的 10 個系列路徑 |

#### 4.1.3 系列資訊欄位

| 欄位 | 類型 | 可編輯 | 說明 |
|------|------|--------|------|
| name | string | ✅ | 系列名稱 |
| description | text | ✅ | 系列描述 |
| create_date | datetime | ❌ | 建立時間（唯讀） |
| modify_date | datetime | ❌ | 修改時間（唯讀） |

### 4.2 專案管理 (Project Management)

#### 4.2.1 功能描述

在系列內管理多個專案，每個專案包含多個 BOM 版本。

#### 4.2.2 專案屬性

| 欄位 | 類型 | 必填 | 說明 |
|------|------|------|------|
| code | string | ✅ | 專案代碼（如 TANGLED） |
| description | text | ✅ | 專案描述 |
| create_date | datetime | ❌ | 建立時間 |

### 4.3 BOM 版本管理 (BOM Revision)

#### 4.3.1 Phase 定義

Phase 名稱由使用者自訂，常見範例：

| Phase 名稱 | 說明 |
|-----------|------|
| DB | Design Basis / 設計基準階段 |
| EVT | Engineering Validation Test |
| DVT | Design Validation Test |
| PVT | Production Validation Test |
| SI | System Integration |
| MVB | Main Verification Board |
| MP | Mass Production |

#### 4.3.2 版本編號規則

- 格式：`{major}.{minor}`（例：0.1, 0.2, 1.0, 1.1）
- 同一專案內，Phase + Version 組合必須唯一
- 版本號應遞增，但允許跳號

### 4.4 BOM 檢視與編輯

#### 4.4.1 表格視圖

| 特性 | 規格 |
|------|------|
| 元件 | PrimeVue DataTable / VirtualScroller |
| 排序 | 點擊表頭排序，Main Item 排序時 2nd Source 跟隨 |
| 編輯 | 直接在儲存格編輯，即時寫入資料庫 |
| 分頁 | 可選，預設顯示全部資料 |
| 過濾 | 依製程類型（SMD/PTH/BOTTOM）過濾 |

#### 4.4.2 BOM 視圖切換

| 視圖 | 過濾條件 |
|------|----------|
| **ALL** | 排除 `bom_status = X` 的所有零件 |
| **SMD** | 僅顯示 `type = SMD` 的零件 |
| **PTH** | 僅顯示 `type = PTH` 的零件 |
| **BOTTOM** | 僅顯示 `type = BOTTOM` 的零件 |

#### 4.4.3 Main Source vs Second Source 顯示規則

```
┌─────────────────────────────────────────────────┐
│ [Main] HHPN: 34065Y600-GRT-H                    │ ← 粗體 + "Main" 標記
│    Supplier: Samsung                            │
│    Supplier PN: CL05B104KO5NNNC                 │
│    Location: C1,C2,C3                           │
│    ─────────────────────────────────────────    │
│    [2nd] Yageo - RC0402FR-0710KL               │ ← 縮排 + 不同樣式
│    [2nd] Murata - GRM155B11H104ZA01            │
└─────────────────────────────────────────────────┘
```

### 4.5 Matrix BOM 功能

#### 4.5.1 核心概念

針對物料群組（Main + 2nd sources），在不同 Model 中選擇不同的物料組合。

#### 4.5.2 Model 管理

| 操作 | 說明 |
|------|------|
| 新增 Model | 輸入 Model 名稱（預設 A, B, C） |
| 編輯 Model | 修改 Model 名稱 |
| 刪除 Model | 若有勾選資料需二次確認 |

#### 4.5.3 選擇規則

| 規則 | 說明 |
|------|------|
| 單選限制 | 每個群組在每個 Model 中只能選擇一個物料 |
| Implicit Selection | 只有 Main Source 的群組視為已自動選擇 |
| 即時儲存 | 勾選時立即寫入資料庫 |

#### 4.5.4 狀態燈號

| 燈號 | 條件 |
|------|------|
| 🟢 綠色 | 所有 Model 的所有群組皆已完成選擇 |
| 🟡 黃色 | 任一 Model 尚有未完成選擇的群組 |

### 4.6 Excel 匯入/匯出

#### 4.6.1 匯入規範

- 支援格式：`.xls`, `.xlsx`
- 操作方式：拖曳檔案或點擊按鈕選擇
- 解析邏輯：詳見 [資料匯入/匯出規範](#8-資料匯入匯出規範)

#### 4.6.2 匯出規範

- 支援格式：`.xlsx`
- 輸出內容：依 Mode 過濾的多個 Sheet
- 樣式：包含表頭、邊框、條紋底色

---

## 5. 資料庫設計

### 5.1 資料表 ER 圖

```
┌──────────────┐       ┌──────────────┐       ┌──────────────┐
│  series_meta │ 1───n │  projects    │ 1───n │ bom_revisions│
└──────────────┘       └──────────────┘       └──────────────┘
                                                       │
                    ┌──────────────┐                  │ n
                    │ matrix_models│                  │
                    └──────────────┘                  │
                          │ 1                         │
                          │                           │
                          │ n                         │
                    ┌──────────────┐                  │
                    │matrix_select.│──────────────────┘
                    └──────────────┘
                               │
                               │ n
                    ┌──────────────┐       ┌──────────────┐
                    │    parts     │       │second_sources│
                    └──────────────┘       └──────────────┘
```

### 5.2 資料表 Schema 定義

```sql
-- ============================================
-- Series Meta Table
-- ============================================
CREATE TABLE series_meta (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT NOT NULL,
    description     TEXT,
    file_path       TEXT NOT NULL UNIQUE,
    create_date     TEXT NOT NULL DEFAULT (datetime('now')),
    modify_date     TEXT NOT NULL DEFAULT (datetime('now')),
    settings        TEXT DEFAULT '{}'  -- JSON 儲存額外設定
);

-- ============================================
-- Projects Table
-- ============================================
CREATE TABLE projects (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    series_id       INTEGER NOT NULL,
    code            TEXT NOT NULL,
    description     TEXT,
    create_date     TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(series_id, code),
    FOREIGN KEY (series_id) REFERENCES series_meta(id) ON DELETE CASCADE
);

-- ============================================
-- BOM Revisions Table
-- ============================================
CREATE TABLE bom_revisions (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id      INTEGER NOT NULL,
    phase_name      TEXT NOT NULL,
    version         TEXT NOT NULL,
    mode            TEXT NOT NULL DEFAULT 'NPI',  -- NPI | MP
    schematic_version TEXT,
    pcb_version     TEXT,
    pca_pn          TEXT,
    bom_date        TEXT,
    create_date     TEXT NOT NULL DEFAULT (datetime('now')),
    modify_date     TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(project_id, phase_name, version),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

-- ============================================
-- Parts Table (原子化儲存)
-- ============================================
CREATE TABLE parts (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    bom_revision_id INTEGER NOT NULL,
    item            INTEGER,
    hhpn            TEXT NOT NULL,
    supplier        TEXT,
    supplier_pn     TEXT,
    description     TEXT,
    location        TEXT NOT NULL,
    type            TEXT,  -- SMD, PTH, BOTTOM
    bom_status      TEXT NOT NULL DEFAULT 'I',  -- I, X, P, M
    ccl             TEXT DEFAULT 'N',
    remark          TEXT,
    FOREIGN KEY (bom_revision_id) REFERENCES bom_revisions(id) ON DELETE CASCADE
);

-- 建立索引以提升查詢效能
CREATE INDEX idx_parts_bom_revision ON parts(bom_revision_id);
CREATE INDEX idx_parts_hhpn ON parts(hhpn);
CREATE INDEX idx_parts_location ON parts(location);

-- ============================================
-- Second Sources Table
-- ============================================
CREATE TABLE second_sources (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    bom_revision_id INTEGER NOT NULL,
    hhpn            TEXT NOT NULL,
    supplier        TEXT NOT NULL,
    supplier_pn     TEXT NOT NULL,
    description     TEXT,
    FOREIGN KEY (bom_revision_id) REFERENCES bom_revisions(id) ON DELETE CASCADE
);

-- 建立唯一索引避免重複的 Second Source
CREATE UNIQUE INDEX idx_second_sources_unique 
    ON second_sources(bom_revision_id, hhpn, supplier, supplier_pn);

-- ============================================
-- Matrix Models Table
-- ============================================
CREATE TABLE matrix_models (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    bom_revision_id INTEGER NOT NULL,
    name            TEXT NOT NULL,  -- 例：A, B, C
    create_date     TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(bom_revision_id, name),
    FOREIGN KEY (bom_revision_id) REFERENCES bom_revisions(id) ON DELETE CASCADE
);

-- ============================================
-- Matrix Selections Table
-- ============================================
CREATE TABLE matrix_selections (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    matrix_model_id INTEGER NOT NULL,
    hhpn            TEXT NOT NULL,
    supplier        TEXT NOT NULL,
    supplier_pn     TEXT NOT NULL,
    create_date     TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(matrix_model_id, hhpn),
    FOREIGN KEY (matrix_model_id) REFERENCES matrix_models(id) ON DELETE CASCADE
);

-- ============================================
-- Views for Aggregated Data
-- ============================================

-- Main Item 聚合視圖
CREATE VIEW v_main_items AS
SELECT 
    bom_revision_id,
    hhpn,
    supplier,
    supplier_pn,
    description,
    GROUP_CONCAT(DISTINCT location) as locations,
    COUNT(DISTINCT location) as qty,
    type,
    bom_status,
    MAX(ccl) as ccl,
    GROUP_CONCAT(DISTINCT remark) as remarks
FROM parts
WHERE bom_status != 'X'
GROUP BY bom_revision_id, hhpn, supplier, supplier_pn;

-- CCL 關鍵零件視圖
CREATE VIEW v_ccl_parts AS
SELECT * FROM parts WHERE ccl = 'Y';
```

### 5.3 資料庫操作規範

| 操作類型 | 實作方式 | 備註 |
|----------|----------|------|
| 連接管理 | 單一 connection pool | 使用 `sql.Open` 並設定最大連接數 |
| 事務處理 | 使用 `db.BeginTx()` | 批量操作需包裹在事務中 |
| 錯誤處理 | 自訂 error types | 定義 `ErrNotFound`, `ErrDuplicate` 等 |
| 查詢建構 | 使用 `sqlx` 或原生 `sql` | 避免 ORM，保持精確控制 |

---

## 6. API 介面規範

### 6.1 Wails 綁定方法

所有後端方法需透過 Wails 綁定，供前端 JavaScript 呼叫。

```go
// backend/app.go 綁定結構
type App struct {
    // 依賴注入
    db *sql.DB
}

// 綁定方法清單
type Bind struct {
    // Series 管理
    CreateSeries(name, description, filePath string) (*Series, error)
    OpenSeries(filePath string) (*Series, error)
    CloseSeries() error
    GetSeriesInfo() (*SeriesMeta, error)
    GetRecentSeries() ([]SeriesMeta, error)

    // Project 管理
    CreateProject(projectCode, description string) (*Project, error)
    UpdateProject(id int, code, description string) error
    DeleteProject(id int) error
    GetProjects(seriesID int) ([]Project, error)

    // BOM Revision 管理
    CreateBOMRevision(projectID int, phase, version string) (*BOMRevision, error)
    UpdateBOMRevision(id int, data map[string]interface{}) error
    DeleteBOMRevision(id int) error
    GetBOMRevisions(projectID int) ([]BOMRevision, error)
    GetCurrentBOMRevision() (*BOMRevision, error)

    // BOM 管理
    GetBOMParts(revisionID int, view string) ([]Part, error)
    UpdatePart(id int, data map[string]interface{}) error
    CreatePart(revisionID int, part Part) (*Part, error)
    DeletePart(id int) error
    ImportExcel(fileData []byte) error
    ExportExcel(revisionID int, format string) ([]byte, error)

    // Matrix 管理
    CreateMatrixModel(revisionID int, name string) (*MatrixModel, error)
    UpdateMatrixModel(id int, name string) error
    DeleteMatrixModel(id int) error
    GetMatrixModels(revisionID int) ([]MatrixModel, error)
    SelectMatrixItem(modelID int, hhpn, supplier, supplierPN string) error
    GetMatrixSelections(modelID int) ([]MatrixSelection, error)
    GetMatrixStatus(revisionID int) (*MatrixStatus, error)

    // 系統方法
    GetVersion() string
    GetSettings() (*Settings, error)
    UpdateSettings(settings map[string]interface{}) error
}
```

### 6.2 前端 API 服務

```typescript
// services/api.ts
export interface BOMIXAPI {
  // Series
  createSeries(name: string, description: string, filePath: string): Promise<Series>;
  openSeries(filePath: string): Promise<Series>;
  closeSeries(): Promise<void>;
  getSeriesInfo(): Promise<SeriesMeta>;
  getRecentSeries(): Promise<SeriesMeta[]>;

  // Project
  createProject(code: string, description: string): Promise<Project>;
  updateProject(id: number, code: string, description: string): Promise<void>;
  deleteProject(id: number): Promise<void>;
  getProjects(seriesId: number): Promise<Project[]>;

  // BOM Revision
  createBOMRevision(projectId: number, phase: string, version: string): Promise<BOMRevision>;
  updateBOMRevision(id: number, data: Partial<BOMRevision>): Promise<void>;
  deleteBOMRevision(id: number): Promise<void>;
  getBOMRevisions(projectId: number): Promise<BOMRevision[]>;

  // BOM Parts
  getBOMParts(revisionId: number, view: BOMView): Promise<Part[]>;
  updatePart(id: number, data: Partial<Part>): Promise<void>;
  createPart(revisionId: number, part: Omit<Part, 'id'>): Promise<Part>;
  deletePart(id: number): Promise<void>;
  
  // Excel
  importExcel(file: File): Promise<void>;
  exportExcel(revisionId: number, format: string): Promise<Blob>;

  // Matrix
  createMatrixModel(revisionId: number, name: string): Promise<MatrixModel>;
  updateMatrixModel(id: number, name: string): Promise<void>;
  deleteMatrixModel(id: number): Promise<void>;
  getMatrixModels(revisionId: number): Promise<MatrixModel[]>;
  selectMatrixItem(modelId: number, hhpn: string, supplier: string, supplierPn: string): Promise<void>;
  getMatrixSelections(modelId: number): Promise<MatrixSelection[]>;
  getMatrixStatus(revisionId: number): Promise<MatrixStatus>;

  // System
  getVersion(): Promise<string>;
  getSettings(): Promise<Settings>;
  updateSettings(settings: Partial<Settings>): Promise<void>;
}
```

### 6.3 事件機制

Wails 事件用於異步任務進度回報：

```go
// 事件定義
const (
    EventProgress    = "progress"
    EventComplete    = "complete"
    EventError       = "error"
    EventBOMUpdated  = "bom-updated"
)

// 使用範例
func (a *App) ProcessBOM(revisionID int) error {
    // 發送進度事件
    events.Emit(EventProgress, map[string]interface{}{
        "progress": 50,
        "message":  "Processing parts...",
    })
    
    // 處理完成後發送完成事件
    events.Emit(EventComplete, map[string]interface{}{
        "message": "BOM processed successfully",
    })
    
    return nil
}
```

### 6.4 錯誤碼定義

```go
// 錯誤類型定義
var (
    ErrNotFound         = errors.New("resource not found")
    ErrInvalidFormat    = errors.New("invalid file format")
    ErrDuplicate        = errors.New("duplicate entry")
    ErrInvalidMode      = errors.New("invalid mode: must be NPI or MP")
    ErrMatrixIncomplete = errors.New("matrix selection incomplete")
)

// 錯誤回應結構
type ErrorResponse struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
}
```

---

## 7. UI 設計規格

### 7.1 設計系統

| 元素 | 規格 |
|------|------|
| **色彩** | Tailwind 預設色彩系統，主色調 `blue-600` (light) / `blue-400` (dark) |
| **圓角** | `rounded-xl` (12px) 用於卡片、按鈕、對話框 |
| **陰影** | `shadow-sm` 用於卡片，`shadow-lg` 用於懸浮元素 |
| **間距** | 統一使用 4px 網格系統（Tailwind spacing scale） |
| **字型** | system-ui (Segoe UI / Microsoft JhengHei / Inter) |
| **圖標** | PrimeVue Icons / Lucide React |

### 7.2 主題系統

| 模式 | 背景色 | 文字色 | 主色調 |
|------|--------|--------|--------|
| **Light** | `bg-white` | `text-gray-900` | `blue-600` |
| **Dark** | `bg-gray-900` | `text-gray-100` | `blue-400` |

### 7.3 主要頁面佈局

#### 7.3.1 儀表板 (Dashboard)

```
┌───────────────────────────────────────────────────────┐
│  [Logo] BOMIX                    [🌙] [設定] [關於]   │
├───────────────────────────────────────────────────────┤
│                                                       │
│  ┌─────────────────────────────────────────────────┐ │
│  │  歡迎使用 BOMIX                                 │ │
│  │                                                 │ │
│  │  [建立新系列]    [開啟既有系列]                  │ │
│  │                                                 │ │
│  │  ─────────────────────────────────────────────  │ │
│  │                                                 │ │
│  │  最近開啟的系列：                               │ │
│  │  • TANGLEN Series    (2026-07-10)    [開啟]    │ │
│  │  • GALAXY Series     (2026-07-08)    [開啟]    │ │
│  │  • ORION Series      (2026-07-05)    [開啟]    │ │
│  │                                                 │ │
│  └─────────────────────────────────────────────────┘ │
└───────────────────────────────────────────────────────┘
```

#### 7.3.2 BOM 檢視頁面

```
┌───────────────────────────────────────────────────────┐
│  [Logo] BOMIX - TANGLEN Series     [🏠] [BOM] [⚙️]   │
├───────────────────────────────────────────────────────┤
│  專案：TANGLEN  │  Phase: DB  │  Version: 0.3        │
├───────────────────────────────────────────────────────┤
│  [📊 ALL] [SMD] [PTH] [BOTTOM]        [匯入] [匯出]  │
├───────────────────────────────────────────────────────┤
│  HHPN      │ Description  │ Supplier │ Supplier PN   │
│  ──────────┼──────────────┼──────────┼────────────── │
│  [Main]    │ CAP,22uF...  │ Samsung  │ CL05B104...   │ ← 粗體
│    34065...│              │          │               │
│    └─[2nd] │ CAP,22uF...  │ Yageo    │ RC0402FR...   │ ← 縮排
│    └─[2nd] │ CAP,22uF...  │ Murata   │ GRM155B1...   │
│  ──────────┼──────────────┼──────────┼────────────── │
│  [Main]    │ RES,10K...   │ VISHAY   │ CRCW0402...   │
│    ...     │              │          │               │
└───────────────────────────────────────────────────────┘
```

### 7.4 元件清單

| 元件名稱 | 用途 | 備註 |
|----------|------|------|
| `BOMTable` | BOM 資料表格 | 使用 PrimeVue DataTable |
| `MatrixSelector` | Matrix 選擇器 | Checkbox 多選互斥 |
| `RevisionTree` | BOM 版本樹狀圖 | 階層式導覽 |
| `ImportDialog` | 匯入對話框 | 拖曳檔案支援 |
| `ExportDialog` | 匯出對話框 | 格式選擇 |
| `SettingsPanel` | 設定面板 | 主題切換等 |

---

## 8. 資料匯入/匯出規範

### 8.1 Excel 匯入規則

#### 8.1.1 表頭解析

| 儲存格 | 欄位 | 解析規則 |
|--------|------|----------|
| B3 | project_code | 取 `"Product Code: "` 後方文字 |
| B4 | description | 取 `"Description: "` 後方文字 |
| D3 | schematic_version | 取 `"Schematic Version: "` 後方文字 |
| F3 | pcb_version | 取 `"PCB Version: "` 後方文字 |
| F4 | pca_pn | 取 `"PCA PN: "` 後方文字 |
| H4 | date | 取 `"Date: "` 後方文字 |

#### 8.1.2 零件資料解析

| Excel 欄 | 資料庫欄位 | 說明 |
|----------|-----------|------|
| A | item | 項目編號 |
| B | hhpn | 公司內部料號 |
| E | description | 零件描述 |
| F | supplier | 供應商名稱 |
| G | supplier_pn | 供應商料號 |
| I | location | 零件位置編號 |
| J | ccl | 是否為 Critical Part |
| L | remark | 註記 |

#### 8.1.3 Main vs Second Source 判斷

- **Main Source**: `item` 和 `location` 都有值
- **Second Source**: 缺少任一欄位，歸屬於上一個 Main Source

#### 8.1.4 Sheet 讀取順序

| Sheet 名稱 | type | bom_status |
|-----------|------|------------|
| SMD | SMD | I |
| PTH | PTH | I |
| BOTTOM | BOTTOM | I |
| NI | (空) | X |
| PROTO | (空) | P |
| MP | (空) | M |

### 8.2 Excel 匯出規則

#### 8.2.1 輸出 Sheet 清單

| Sheet | 內容 |
|-------|------|
| Changelist | 變更清單（待實作） |
| Changelist by Sheet | 按製程分類的變更清單（待實作） |
| ALL | 完整 BOM（依 Mode 過濾） |
| SMD | 僅 SMD 製程 |
| PTH | 僅 PTH 製程 |
| BOTTOM | 僅 BOTTOM 製程 |
| NI | 不上件清單 |
| PROTO | 原型階段上件清單 |
| MP | 量產階段上件清單 |
| CCL | 關鍵零件清單 |

#### 8.2.2 表頭格式

```
┌──────────────────────────────────────────────────────┐
│  FUJIN PRECISION INDUSTRY(SHENZHEN) CO.,LTD         │
│  BILL OF MATERIAL                                    │
├──────────────────────────────────────────────────────┤
│  B3: Product Code: TANGLEN                          │
│  B4: Description: MBD, Tangled...                   │
│  D3: Schematic Version: 1.0                         │
│  F3: PCB Version: 2.1                               │
│  F4: PCA PN: ABC-123                                │
│  H3: BOM Version: 0.3                               │
│  H4: Date: 2026-07-15                               │
│  J3: Phase: DB                                      │
└──────────────────────────────────────────────────────┘
```

---

## 9. 測試策略

### 9.1 測試層級

| 層級 | 工具 | 範圍 |
|------|------|------|
| **單元測試** | Go testing, Vitest | 個別函式/元件 |
| **整合測試** | Testcontainers | 資料庫操作 |
| **E2E 測試** | Playwright | 使用者流程 |

### 9.2 Go 測試規範

- 使用 Table-Driven Tests
- 測試檔案命名：`{filename}_test.go`
- 覆蓋率目標：> 80%

### 9.3 前端測試規範

- 使用 Vitest + Vue Test Utils
- 測試檔案命名：`*.spec.ts`
- Mock Wails 綁定

---

## 10. 部署與建置

### 10.1 建置命令

```bash
# 開發模式
wails3 dev

# 生產建置（Windows）
wails3 build -platform windows/amd64

# 生產建置（macOS）
wails3 build -platform darwin/arm64

# 生產建置（Linux）
wails3 build -platform linux/amd64
```

### 10.2 輸出檔案

| 平台 | 輸出路徑 | 檔案名稱 |
|------|----------|----------|
| Windows | `build/bin/` | `BOMIX.exe` |
| macOS | `build/bin/` | `BOMIX.app` |
| Linux | `build/bin/` | `BOMIX` |

### 10.3 版本管理

- 遵循 Semantic Versioning
- 版本號定義在 `package.json` 和 `wails.json`
- 使用 Git Tags 追蹤版本

---

## 11. 開發時程規劃

### Phase 1: 基礎架構 (已完成)
- [x] 專案初始化
- [x] 資料庫 Schema 設計
- [x] 基礎項目結構建立

### Phase 2: 核心功能 (開發中)
- [ ] Excel 匯入邏輯
- [ ] BOM 表格檢視
- [ ] BOM 編輯功能

### Phase 3: 進階功能
- [ ] Excel 匯出
- [ ] 版本比較
- [ ] Matrix BOM 支援

### Phase 4: 優化與測試
- [ ] 性能優化
- [ ] 單元測試覆蓋
- [ ] E2E 測試

---

## 附錄

### A. 術語表

| 術語 | 定義 |
|------|------|
| BOM | Bill of Materials，物料清單 |
| Main Source | 主要供應商物料 |
| Second Source | 替代料 |
| Matrix BOM | 多 Model 的物料選擇配置 |
| Phase | 開發階段（如 DB, DVT, PVT） |

### B. 參考文件

- [Wails v3 官方文件](https://wails.io/)
- [Vue 3 官方文件](https://vuejs.org/)
- [PrimeVue 元件庫](https://primevue.org/)
- [excelize Go 套件](https://pkg.go.dev/github.com/xuri/excelize/v2)

---

*本文檔由 BOMIX 開發團隊維護，最後更新：2026-07-15*
