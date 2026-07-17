# BOMIX Wave 1 — Implementation Plan

> **用途**：供 Claude Code Agent 按 Phase 順序自動推進開發  
> **規格來源**：`docs/product-spec.md`（產品規格書）  
> **架構來源**：`claude.md`（專案指引與目錄結構）  
> **推進規則**：每個 Phase 的所有 `[ ]` 項目與驗證都通過後，才可進入下一個 Phase

---

## Phase 0 — 專案初始化

> 建立 Wails v3 專案骨架與前端依賴，確保開發環境可以成功啟動。

### 任務

- [x] 初始化 Wails v3 專案（使用 `wails3 init` 並確認選擇 vue+vite），注意確保目錄結構與 claude.md 中描述相同（僅有單層 bomix-app 目錄）
- [x] 確認 `go.mod` 存在且模組名稱合理（如 `bomix-app`）
- [x] 確認 `frontend/` 內的 Vue 3 專案結構正確（`package.json`、`vite.config.ts`、`src/App.vue`）
- [x] 安裝前端核心依賴：`pnpm add primevue @primevue/themes primeicons vue-router pinia`
- [x] 安裝前端開發依賴：`pnpm add -D tailwindcss @tailwindcss/vite`（Tailwind CSS v4）
- [x] 設定 `vite.config.ts` 中的 Tailwind CSS v4 plugin
- [x] 在 `src/main.ts` 中初始化 PrimeVue（使用 Aura 主題）、Router、Pinia
- [x] 在 Go 後端安裝核心依賴：`go get gorm.io/gorm modernc.org/sqlite gorm.io/driver/sqlite github.com/xuri/excelize/v2 github.com/BurntSushi/toml github.com/google/uuid`
- [x] 建立 `claude.md` 中定義的完整目錄結構骨架（所有空 `.go` / `.ts` / `.vue` 檔案，內含 `package` 宣告或最小合法語法）
- [x] 建立空的 Excel 範本檔案佔位：`bomix-app/template/bigmatrix.xlsx`、`bomix-app/template/matrix.xlsx`

### 驗證

```bash
cd bomix-app && go build ./...                  # Go 編譯通過
cd bomix-app/frontend && pnpm run build         # 前端建置成功
cd bomix-app && wails3 dev                      # 開發伺服器啟動，視窗顯示（手動確認後 Ctrl+C）
```

### 完成條件

- `go build ./...` 零錯誤
- `pnpm run build` 零錯誤
- 所有 `claude.md` Section 2 定義的目錄與檔案皆已建立

---

## Phase 1 — 日誌系統 (Logger)

> 建立基於 `slog` 的結構化日誌系統，含環形緩衝區與 Wails 事件推送。後續所有模組都將依賴此模組進行除錯輸出。

### 任務

- [x] 實作 `backend/logger/logger.go`
  - 定義 `Logger` struct，封裝 `slog.Logger`
  - 支援 `Debug` / `Info` / `Warn` / `Error` 四個等級方法
  - 每次 log 呼叫同時寫入環形緩衝區並透過 Wails 事件 `log:new` 推送至前端
  - 提供 `GetLogs(level string, limit int) []LogEntry` 方法供前端拉取歷史日誌
  - 提供 `ClearLogs()` 方法
- [x] 實作 `backend/logger/buffer.go`
  - 環形緩衝區（Ring Buffer），預設容量 500 條
  - `LogEntry` 結構：`{ID, Level, Message, Timestamp, Attrs map[string]string}`
  - 支援依等級過濾
  - 線程安全（`sync.RWMutex`）
- [x] 撰寫單元測試 `backend/logger/logger_test.go`
  - 測試寫入多條不同等級的日誌後，`GetLogs` 回傳正確
  - 測試環形緩衝區溢出行為（寫入超過 500 條後，最舊的被覆蓋）
  - 測試等級過濾

### 驗證

```bash
cd bomix-app && go test ./backend/logger/... -v -count=1
```

### 完成條件

- 所有測試 PASS
- `go vet ./backend/logger/...` 無警告

---

## Phase 2 — 設定系統 (Config)

> 建立 TOML 格式的設定檔讀寫系統，支援差異寫入與預設值 fallback。

### 任務

- `[ ]` 實作 `backend/config/defaults.go`
  - 定義所有設定項目的 struct（`Config`、`ImportConfig`、`LoggerConfig`、`RecentFilesConfig`）
  - 定義 `DefaultConfig` 變數（所有預設值，見 product-spec 5.3.3）
  - 包含新增的設定項：`AutoOpenLastFile`（預設 false）、`LastOpenedFile`（預設空）、`AutoImportPreviousMatrix`（預設 false）
- `[ ]` 實作 `backend/config/config.go`
  - `Load(path string) (*Config, error)`：讀取 TOML 檔案並合併至預設值
  - `Save(path string, cfg *Config) error`：僅寫入與 `DefaultConfig` 不同的項目（差異寫入）
  - `GetConfigPath() string`：回傳 `%APPDATA%/BOMIX/config.toml`
  - 若檔案不存在，不建立檔案，直接回傳 `DefaultConfig`
- `[ ]` 撰寫單元測試 `backend/config/config_test.go`
  - 測試無檔案時 Load 回傳 DefaultConfig
  - 測試部分覆蓋（僅寫入 `theme = "dark"` 的 TOML，Load 後 theme 為 dark，其餘為預設值）
  - 測試 Save 的差異寫入（修改一個欄位後 Save，確認 TOML 僅包含該欄位）
  - 測試完整 round-trip（Save → Load → 比較）

### 驗證

```bash
cd bomix-app && go test ./backend/config/... -v -count=1
```

### 完成條件

- 所有測試 PASS
- `go vet ./backend/config/...` 無警告

---

## Phase 3 — 領域型別與錯誤定義 (Types)

> 建立共用的領域型別、介面、錯誤定義。

### 任務

- `[ ]` 實作 `backend/types/domain.go`
  - 定義 `BOMFormat`（EBOM / BigMatrix / Matrix / Unknown）
  - 定義 `TaskType`（Import / Export / Analysis）
  - 定義 `TaskStatus`（Created / Queued / Running / Completed / Failed / Cancelled）
  - 定義 `ExportOptions` struct（見 product-spec 6.5.3）
  - 定義 `ImportResult` struct（見 product-spec 10.1.5）
  - 定義 DTO：`AggregatedPart`、`SecondSourceDTO`、`BOMSummary`（見 product-spec 4.7）
- `[ ]` 實作 `backend/types/errors.go`
  - 定義所有 Sentinel Error（見 product-spec 10.3）
- `[ ]` 實作 `backend/types/interfaces.go`
  - 定義 `ExcelReader` 介面
  - 定義 `ExcelWriter` 介面

### 驗證

```bash
cd bomix-app && go build ./backend/types/...
cd bomix-app && go vet ./backend/types/...
```

### 完成條件

- 編譯通過、`go vet` 無警告
- 所有 product-spec 中定義的型別、DTO、錯誤皆已覆蓋

---

## Phase 4 — 非同步任務系統 (Task Manager)

> 建立任務管理器，支援提交、取消、進度追蹤與 Wails 事件回呼。

### 任務

- `[ ]` 實作 `backend/task/task.go`
  - 定義 `Task` struct（見 product-spec 5.1.2）
  - 定義 `TaskFunc` 類型（任務的可執行函數簽名，接收進度回呼）
- `[ ]` 實作 `backend/task/manager.go`
  - `NewTaskManager(logger, app)` 建構函數
  - `Submit(name, taskType, fn TaskFunc) string`：建立任務、啟動 goroutine 執行、回傳 TaskID
  - `Cancel(taskID) error`：透過 context 取消任務
  - `GetStatus(taskID) *Task`
  - `ListTasks() []Task`
  - 任務狀態變化時透過 Wails 事件推送：`task:progress`、`task:complete`、`task:failed`、`task:cancelled`
  - 所有狀態變更與日誌記錄到 Logger
- `[ ]` 實作 `backend/task/callback.go`
  - 定義事件常數（見 product-spec 5.1.4）
  - 定義事件 payload struct
- `[ ]` 撰寫單元測試 `backend/task/manager_test.go`
  - 測試提交任務 → 狀態從 Created 經 Running 到 Completed
  - 測試任務失敗 → 狀態變為 Failed，Error 欄位有值
  - 測試取消任務 → 狀態變為 Cancelled
  - 測試 ListTasks 回傳所有任務
  - 測試進度更新（模擬 progress 0.0→0.5→1.0）

### 驗證

```bash
cd bomix-app && go test ./backend/task/... -v -count=1
```

### 完成條件

- 所有測試 PASS
- `go vet ./backend/task/...` 無警告

---

## Phase 5 — 資料庫層 (DB)

> 建立 GORM Model、連線管理、所有 CRUD 操作、Single Writer 模式。

### 任務

- `[ ]` 實作 `backend/db/models.go`
  - 定義所有 GORM Model（見 product-spec 附錄 B）：`Series`、`Project`、`BomRevision`、`Part`、`SecondSource`、`MatrixModel`、`MatrixSelection`
  - 包含所有 CASCADE 約束與子集合欄位
- `[ ]` 實作 `backend/db/connection.go`
  - `Open(filePath string) (*gorm.DB, error)`：開啟 .bomx 檔案，設定 WAL 模式、限制單一連線
  - `Close(db *gorm.DB) error`：正確關閉連線
  - `AutoMigrate(db *gorm.DB) error`：執行 GORM AutoMigrate + 建立所有複合索引（見 product-spec 附錄 B 資料庫索引）
  - 啟動 Single Writer goroutine（見 product-spec 3.3.3）
- `[ ]` 實作 `backend/db/series.go`
  - `CreateSeries(db, name, description) (*Series, error)`
  - `GetSeriesInfo(db) (*Series, error)`
- `[ ]` 實作 `backend/db/project.go`
  - `GetOrCreateProject(db, code, description) (*Project, error)`
  - `GetProjects(db, seriesID) ([]Project, error)`
  - `GetProject(db, id) (*Project, error)`
- `[ ]` 實作 `backend/db/revision.go`
  - `CreateRevision(db, revision) (*BomRevision, error)`
  - `GetRevision(db, id) (*BomRevision, error)`
  - `GetRevisions(db, projectID) ([]BomRevision, error)`
  - `FindRevision(db, projectID, phase, version) (*BomRevision, error)`
  - `UpdateRevision(db, revision) error`
  - `FindPreviousRevision(db, projectID, phase, currentVersion) (*BomRevision, error)`：搜尋同 project + phase 下版本號最接近且小於 currentVersion 的 Revision（見 product-spec 7.1.8）
- `[ ]` 實作 `backend/db/part.go`
  - `CreatePartsInBatch(db, parts []Part) error`：批次寫入
  - `DeletePartsByRevision(db, revisionID) error`
  - `GetPartsByRevision(db, revisionID) ([]Part, error)`
  - `GetPartsByRevisionAndType(db, revisionID, type) ([]Part, error)`
- `[ ]` 實作 `backend/db/matrix.go`
  - `CreateMatrixModel(db, model) error`
  - `GetMatrixModels(db, revisionID) ([]MatrixModel, error)`
  - `CreateMatrixSelections(db, selections []MatrixSelection) error`
  - `DeleteMatrixSelectionsByRevision(db, revisionID) error`
  - `GetMatrixSelections(db, revisionID, modelID) ([]MatrixSelection, error)`
  - SecondSource 相關：`GetSecondSourcesByRevision`、`CreateSecondSourcesInBatch`、`DeleteSecondSourcesByRevision`、`DeleteSecondSource`、`UpdateSecondSource`
  - MatrixSelection 清理：`DeleteInvalidSelections(db, revisionID, removedGroups, removedMaterials)`（見 product-spec 7.1.7 步驟 4）
- `[ ]` 撰寫整合測試 `backend/db/db_test.go`（使用 SQLite in-memory）
  - 測試 Open → AutoMigrate → 建立 Series → 建立 Project → 建立 Revision → 新增 Parts → 查詢
  - 測試唯一索引約束（重複 project code 應失敗）
  - 測試 CASCADE 刪除（刪除 Revision 後，Parts / SecondSources / MatrixModels / MatrixSelections 應一併消失）
  - 測試 FindPreviousRevision（建立 0.1, 0.2, 0.4 後搜尋 0.5 → 應回傳 0.4）

### 驗證

```bash
cd bomix-app && go test ./backend/db/... -v -count=1
```

### 完成條件

- 所有測試 PASS
- `go vet ./backend/db/...` 無警告
- CASCADE 刪除測試通過

---

## Phase 6 — UI 腳手架與日誌面板

> 建立前端完整佈局骨架、路由、狀態管理、日誌/任務面板。後續 Phase 的功能模組在此骨架上擴充。

### 任務

- `[ ]` 實作 Vue Router（`frontend/src/router/`）
  - `/` → 歡迎頁（建立/開啟 .bomx）
  - `/workspace` → 工作區主頁（側邊欄 + 主內容 + 底部面板）
  - `/settings` → 設定頁
- `[ ]` 實作 Pinia Store（`frontend/src/stores/`）
  - `useAppStore`：管理 series 連線狀態（isOpen、seriesInfo）
  - `useTaskStore`：管理任務列表、監聽 `task:*` 事件
  - `useLogStore`：管理日誌列表、監聽 `log:new` 事件
  - `useProjectStore`：管理專案列表與當前選中的專案/版本
- `[ ]` 實作主佈局 `App.vue`
  - 採用 PrimeVue `Splitter` 實作三區塊佈局（見 product-spec 9.2）
  - 側邊欄可拖曳調整寬度（最小 0、預設 20%、雙擊回復）
  - 底部面板可拖曳調整高度（最小單行、預設單行、雙擊回復）
- `[ ]` 實作歡迎頁 `views/WelcomePage.vue`
  - 建立新 Series 按鈕（呼叫 Wails SaveFileDialog → CreateSeries）
  - 開啟 Series 按鈕（呼叫 Wails OpenFileDialog → OpenSeries）
  - 最近開啟列表（呼叫 GetRecentSeries）
- `[ ]` 實作工作區頁 `views/WorkspacePage.vue`
  - 側邊欄：顯示專案列表（從 `useProjectStore` 讀取）
  - 主內容區：placeholder 文字（後續 Phase 填入 BOM 表格）
  - 頂部工具列：匯入按鈕、匯出按鈕（目前為 disabled placeholder）
- `[ ]` 實作日誌/任務面板 `components/LogPanel.vue`
  - 從 `useLogStore` 讀取日誌列表渲染
  - 支援依等級過濾（Tabs：全部 / INFO / WARN / ERROR）
  - 任務日誌顯示：任務名稱、狀態 badge、進度條、最新訊息（即時更新）
  - 清除日誌按鈕
- `[ ]` 實作設定頁 `views/SettingsPage.vue`
  - 主題切換（Light / Dark / System）
  - 匯入設定：`confirmOverwrite`、`autoImportPreviousMatrix`
  - 自動開啟：`autoOpenLastFile`
  - 儲存按鈕 → 呼叫 `UpdateSettings`
- `[ ]` 實作 Wails 綁定呼叫層 `frontend/src/services/api.ts`
  - 封裝所有後端 API 呼叫方法（CreateSeries、OpenSeries、GetLogs 等）
  - 統一錯誤處理
- `[ ]` 實作 `backend/app.go`（Wails 主綁定）
  - 初始化 Logger、Config、TaskManager
  - 綁定所有 API 方法（product-spec 10.1 的所有方法簽名）
  - 啟動時若 `autoOpenLastFile` 為 true，自動開啟上次的 .bomx
  - 開啟 .bomx 時記錄路徑到 config `lastOpenedFile`
  - 管理最近開啟清單（`RecentFiles`）

### 驗證

```bash
cd bomix-app && go build ./...                   # Go 編譯通過
cd bomix-app/frontend && pnpm run build          # 前端建置成功
cd bomix-app && wails3 dev                       # 手動驗證：
#   1. 歡迎頁顯示正確
#   2. 建立 .bomx 後跳轉至工作區
#   3. 底部日誌面板可見且可拖曳
#   4. 設定頁可切換主題
```

### 完成條件

- Go 與前端皆可編譯
- 歡迎頁 → 工作區 → 設定頁 三頁面路由正常
- 日誌面板可顯示日誌條目
- .bomx 建立/開啟/關閉流程完整

---

## Phase 7 — Excel 匯入模組

> 實作 Detector、EBOM Reader、BigMatrix Reader、Merge 演算法、MatrixSelection 匯入模組。

### 任務

- `[ ]` 實作 `backend/excel/detector.go`
  - `Detect(file *excelize.File) BOMFormat`（見 product-spec 6.3.1 偵測規則對照表）
- `[ ]` 實作 `backend/excel/reader.go`
  - 定義 `Reader` 介面
  - `ImportExcel(filePaths []string) ([]ImportResult, error)` 總入口邏輯（見 product-spec 6.3.2 流程）
- `[ ]` 實作 `backend/excel/reader_ebom.go`
  - 表頭解析（見 product-spec 7.1.1）
  - 零件解析（見 product-spec 7.1.2）
  - Main Source / 2nd Source 判斷（見 product-spec 7.1.3）
  - Location 原子化（見 product-spec 7.1.4）
  - Sheet 讀取順序與覆蓋規則（見 product-spec 7.1.5）
  - NPI / MP 模式判斷（見 product-spec 7.1.6）
  - Merge 演算法（見 product-spec 7.1.7 步驟 1~4）
  - 自動帶入上一版 Matrix 勾選（見 product-spec 7.1.8）
- `[ ]` 實作 `backend/excel/reader_bigmatrix.go`
  - 表頭解析（見 product-spec 7.2.2）
  - 橫向多 BOM + Model 動態列解析（見 product-spec 7.2.2.1）
  - 零件與 Model 資料解析（見 product-spec 7.2.3）
  - BigMatrix 匯入規則：僅更新 Matrix 勾選，清除舊的 MatrixSelections 後重新插入（見 product-spec 7.0.2 + 7.2.4）
- `[ ]` 實作 `backend/excel/reader_matrix.go`
  - Placeholder：回傳 `ErrInvalidFormat`（暫不支援）
- `[ ]` 實作 MatrixSelection 匯入功能模組（可放在 `backend/db/matrix.go` 或獨立檔案）
  - `ImportMatrixSelections(db, sourceRevisionID, targetRevisionID) error`（見 product-spec 7.4）
- `[ ]` 撰寫單元測試 `backend/excel/detector_test.go`
  - 建立三種格式的最小 mock xlsx 檔案並測試正確辨識
- `[ ]` 撰寫單元測試 `backend/excel/reader_ebom_test.go`
  - 測試表頭解析
  - 測試 Main/2nd 判斷
  - 測試 NPI/MP 模式判斷
  - 測試 Merge 演算法（使用 product-spec 7.1.7 的 P1~P4 範例）
- `[ ]` 撰寫單元測試 `backend/excel/reader_bigmatrix_test.go`
  - 測試橫向多 BOM 解析
  - 測試 Model 勾選狀態解析

### 驗證

```bash
cd bomix-app && go test ./backend/excel/... -v -count=1
```

### 完成條件

- 所有測試 PASS
- Detector 正確辨識三種格式
- EBOM Merge 演算法通過 P1~P4 範例驗證
- BigMatrix 匯入正確解析多 BOM 橫向結構

---

## Phase 8 — 資料聚合與視圖過濾 (Processor)

> 實作 BOM 資料聚合（Main Item 合併）與視圖過濾邏輯。

### 任務

- `[ ]` 實作 `backend/processor/aggregator.go`
  - `Aggregate(parts []Part, secondSources []SecondSource) []AggregatedPart`
  - 以 `(supplier, supplier_pn)` 為群組鍵合併
  - 合併 Location 為逗號分隔字串
  - Qty = Location 數量
  - 附加 SecondSources
- `[ ]` 實作 `backend/processor/filter.go`
  - `FilterByView(parts []AggregatedPart, view string, mode string) []AggregatedPart`
  - 實作所有視圖過濾規則（見 product-spec 6.4.2）：ALL / SMD / PTH / BOTTOM / NI / PROTO / MP / CCL
- `[ ]` 撰寫單元測試 `backend/processor/aggregator_test.go`
  - 測試同群組的 Location 合併與 Qty 計算
  - 測試 SecondSource 正確附加
- `[ ]` 撰寫單元測試 `backend/processor/filter_test.go`
  - 依 product-spec 6.4.2 的 8 種視圖各寫一個測試 case

### 驗證

```bash
cd bomix-app && go test ./backend/processor/... -v -count=1
```

### 完成條件

- 所有測試 PASS
- 8 種視圖過濾邏輯全部正確

---

## Phase 9 — Excel 匯出模組

> 實作 BigMatrix 與 Matrix 兩種格式的 Template-Driven 匯出。

### 任務

- `[ ]` 實作 `backend/excel/template.go`
  - 使用 `//go:embed` 嵌入 `template/bigmatrix.xlsx` 與 `template/matrix.xlsx`
  - 提供 `LoadTemplate(format BOMFormat) (*excelize.File, error)` 方法
- `[ ]` 實作 `backend/excel/writer.go`
  - 定義 `Writer` 介面
  - `ExportExcel(options ExportOptions) ([]string, error)` 總入口邏輯（見 product-spec 6.5.2）
- `[ ]` 實作 `backend/excel/writer_bigmatrix.go`
  - Tag 置換法填入表頭（見 product-spec 8.1.2）
  - 動態欄生成（橫向多 BOM + Model，見 product-spec 8.1.3）
  - 零件資料填入（見 product-spec 8.1.4）
  - 物料群組底色交替、公式寫入（見 product-spec 8.1.5）
  - 檔名自動命名（見 product-spec 8.1.7）
- `[ ]` 實作 `backend/excel/writer_matrix.go`
  - Tag 置換法填入表頭（見 product-spec 8.2.2）
  - 動態 Model 欄生成（見 product-spec 8.2.3）
  - 零件資料填入（見 product-spec 8.2.4）
  - SMD / PTH / BOTTOM 三 Sheet 分別寫入（見 product-spec 8.2.5）
  - 公式寫入（見 product-spec 8.2.5.3 / 8.2.5.4）
  - 零件篩選（見 product-spec 8.2.6）
  - 檔名自動命名（見 product-spec 8.2.10）
- `[ ]` 建立實際的 Excel 範本檔案
  - `template/bigmatrix.xlsx`：含 Tag 佔位標籤、原型列樣式
  - `template/matrix.xlsx`：含 Tag 佔位標籤、三個 Sheet（SMD / PTH / BOTTOM）、原型列樣式
- `[ ]` 撰寫單元測試 `backend/excel/writer_bigmatrix_test.go`
  - 測試 Tag 置換
  - 測試匯出後 xlsx 可被 excelize 正確讀取
- `[ ]` 撰寫單元測試 `backend/excel/writer_matrix_test.go`
  - 測試 Tag 置換
  - 測試三 Sheet 均有資料

### 驗證

```bash
cd bomix-app && go test ./backend/excel/... -v -count=1
```

### 完成條件

- 所有測試 PASS
- 匯出的 xlsx 檔案可被 Excel 正確開啟（可透過 excelize 讀取回驗）

---

## Phase 10 — 端到端整合與 UI 串接

> 串接前後端，完成匯入/匯出 UI 流程，執行端到端整合測試。

### 任務

- `[ ]` 前端串接匯入流程
  - 工作區的匯入按鈕 → 開啟檔案對話框 → 呼叫 `ImportExcel` → 顯示任務進度 → 完成後刷新資料
  - 支援拖曳檔案到匯入區域
  - 匯入結果彈窗（被跳過的檔案）
- `[ ]` 前端串接匯出流程
  - 工作區的匯出按鈕 → 匯出對話框（選擇格式、勾選 Revisions、設定 Model 數量）→ 呼叫 `ExportExcel`
  - BigMatrix：顯示 description 輸入欄位、Model 數量調整
  - Matrix：多選 Revision 時選擇輸出目錄
- `[ ]` 前端串接 BOM 表格
  - 工作區主內容顯示 `BOMTable` 元件（PrimeVue DataTable）
  - 切換 View（ALL / SMD / PTH 等）
  - 展開列顯示 Second Source
- `[ ]` 前端串接側邊欄
  - 專案列表 → 展開顯示 Revision 列表 → 點選載入 BOM 表格
- `[ ]` 重複匯入確認彈窗
  - 若 `confirmOverwrite` 為 true，匯入重複 BOM 時顯示確認對話框
- `[ ]` 撰寫端到端整合測試（Go 層級）
  - 建立 in-memory DB → 匯入測試 EBOM xlsx → 查詢 Parts → 匯出 BigMatrix → 讀取匯出檔案驗證
  - 測試 EBOM 重新匯入 Merge 流程（含 MatrixSelection 保留驗證）
- `[ ]` 最終驗證
  - `go build ./...` 零錯誤
  - `go test ./...` 全部 PASS
  - `pnpm run build` 零錯誤
  - `wails3 build -platform windows/amd64` 產出 BOMIX.exe

### 驗證

```bash
cd bomix-app && go test ./... -v -count=1        # 所有 Go 測試
cd bomix-app/frontend && pnpm run build           # 前端建置
cd bomix-app && wails3 build -platform windows/amd64  # 生產建置
```

### 完成條件

- `go test ./...` 全部 PASS，零失敗
- `wails3 build` 成功產出 `build/bin/BOMIX.exe`
- 手動驗證完整流程：建立 .bomx → 匯入 EBOM → 檢視 BOM 表格 → 匯出 BigMatrix → 開啟匯出檔案確認

---

## 自動推進規則

1. **每個 Phase 完成後**：執行該 Phase 的驗證指令，全部通過後將 `[ ]` 改為 `[x]`，commit 變更，然後自動進入下一個 Phase。
2. **除非遇到以下情況，否則禁止中斷**：
   - 產品規格或邏輯出現嚴重衝突，需要使用者決策。
   - 重試超過 5 次仍卡在同一個編譯/測試錯誤。
3. **目標終點**：所有 Phase 的所有 `[ ]` 全部變成 `[x]`，且 `wails3 build` 成功。
