# BOMIX - 高效能 Excel BOM 處理與矩陣分析工具

BOMIX 是一款專為 Windows 環境設計的高效能 Excel BOM (Bill of Materials) 處理與數據矩陣計算工具。結合 Go 語言的強大併發與記憶體串流能力，以及 Vue 3 / PrimeVue 所提供的現代化 GUI 介面，能流暢處理巨量 BOM 資料的解析、比對、聚合與導出。

---

## 🛠 技術棧 (Technology Stack)

- **核心語言 (Backend):** Go (Golang 1.21+)
- **桌面應用框架 (GUI Framework):** [Wails v3](https://v3.wails.io/) (Go + Webview)
- **前端 UI (Frontend):** Vue 3 (Composition API) + TypeScript + Vite
- **UI 元件與樣式 (Styling & Components):** PrimeVue v4 + Tailwind CSS v4
- **Excel 處理引擎:** [Excelize v2](https://github.com/xuri/excelize)

---

## 📁 專案目錄結構 (Project Structure)

專案採用 Nested Mono-repo 結構，主應用程式位於 `bomix-app` 目錄中：

```text
BOMIX/                         # 工作區根目錄
├── README.md                  # 專案說明文件 (本檔案)
├── AGENTS.md                  # 專案開發規範與架構指南
├── implementationPlan.md      # 開發計畫與功能清單
├── docs/                      # 產品需求規格 (PRD) 與 API 文件
├── scripts/                   # 全域自動化腳本
├── Dockerfile                 # 開發環境容器設定
├── start-dev.sh               # Docker 開發環境啟動腳本
└── bomix-app/                 # 主應用程式碼目錄 (Wails v3 核心)
    ├── main.go                # Wails 應用程式進入點
    ├── wails.json             # Wails 專案設定檔
    ├── Taskfile.yml           # 建置與自動化任務設定
    ├── go.mod                 # Go 模組設定檔
    ├── template/              # Excel 導出樣板 (透過 go:embed 打包)
    ├── backend/               # Go 後端核心邏輯
    │   ├── app.go             # Wails 綁定與生命週期管理
    │   ├── config/            # 系統設定
    │   ├── db/                # SQLite 資料庫儲存層 (GORM)
    │   ├── excel/             # Excel 讀寫、解析與流式導出
    │   ├── processor/         # BOM 精確聚合與過濾演算法
    │   ├── task/              # 異步任務管理與進度追蹤
    │   └── logger/            # 結構化日誌 (slog)
    └── frontend/              # Vue 3 前端專案
        ├── package.json       # 前端套件設定
        ├── vite.config.ts     # Vite 建置設定
        └── src/               # 前端源碼 (Pages, Components, Pinia Stores, Services)
```

---

## 🚀 快速開始與開發操作 (Development Guide)

### 1. 環境需求 (Prerequisites)

在開始開發前，請確保開發環境已安裝以下工具：

- **Go**: `1.21` 或以上版本
- **Node.js**: `v18` 或以上版本（建議使用 LTS 版本）與 `npm` 或 `pnpm`
- **Wails v3 CLI**:
  ```bash
  go install github.com/wailsapp/wails/v3/cmd/wails3@latest
  ```

---

### 2. 啟動開發模式 (Run Development Mode)

開發模式支援前端熱重載 (Hot-Reload) 與後端 Go 程式碼自動重新編譯。

#### 方法一：使用 Wails3 CLI 啟動 (推薦)

```bash
# 1. 切換至應用程式目錄
cd bomix-app

# 2. (首次執行) 安裝前端依賴套件
cd frontend && npm install && cd ..

# 3. 啟動開發伺服器
wails3 dev
```

#### 方法二：使用 Task 啟動

若環境中有安裝 `task` (TaskfileRunner)：

```bash
cd bomix-app
task dev
```

#### 方法三：前端獨立開發 UI 預覽

若僅需要調整 UI 介面樣式或元件，可直接啟動 Vite 開發伺服器：

```bash
cd bomix-app/frontend
npm install
npm run dev
```
> *註：獨立啟動前端時，涉及 Go 原生 API 的呼叫將需要 Mock 資料支援。*

---

### 3. 生產環境打包與版本發布 (Production Build & Release)

BOMIX 生產環境編譯最終會產生單一可執行的獨立 `.exe` 檔案（所有前端靜態資源與 Excel 樣板皆以 `//go:embed` 打包進執行檔）。專案採用 **Git Tag 作為版本號單一真實來源 (SSOT)**，並以 **Taskfile** 作為統一建置核心。

#### 本機建置 (使用 Taskfile)：

```bash
# 進入應用程式目錄
cd bomix-app

# 1. 預設建置 (標記為 dev 版本，自動推導 Git Commit 與時間)
task build

# 2. 模擬特定版本建置 (例如 1.2.0，自動同步至 Windows 屬性與 UI)
task build VERSION=1.2.0
```
*編譯完成的可執行檔將輸出至 `bomix-app/bin/BOMIX.exe`。*

#### 自動化版本發布 (Release SOP)：

專案配置了 GitHub Actions CI 自動化發布流水線（以 Taskfile 為核心）：
1. 建立 Git Tag：`git tag -a v1.0.0 -m "Release v1.0.0"`
2. 推送至遠端：`git push origin v1.0.0`
3. CI 將自動觸發、解析版本號、執行 `task build VERSION=1.0.0` 並發布 Release。

> 📖 更詳盡之版本架構與 SOP 請參閱 [建置與發布手冊 (docs/build-and-release.md)](./docs/build-and-release.md)。

---

### 4. 執行測試 (Running Tests)

專案嚴格區分後端與前端的測試邏輯：

#### 執行 Go 後端單元測試：

```bash
cd bomix-app
go test -v ./backend/...
```

#### 執行 Vue 前端單元測試：

```bash
cd bomix-app/frontend
npm run test:unit
```

---

### 5. 使用 Docker 進行隔離開發 (Optional Docker Environment)

專案根目錄提供了 Dockerfile 與輔助腳本 `start-dev.sh`，可輕鬆建立統一的開發環境：

```bash
# 啟動並進入 Docker 開發容器
./start-dev.sh
```

---

## 💡 開發注意事項與規範 (Guidelines)

1. **記憶體與效能最佳化：**
   - 處理大檔案 Excel 讀寫時，必須使用 `excelize.NewStreamWriter` 進行流式處置，避免全量載入記憶體。
   - 精確數值計算需使用 `github.com/shopspring/decimal`，避免浮點數 IEEE 754 精度誤差。
2. **註解與文件語言：**
   - 所有程式碼註解、JSDoc / GoDoc 說明、commit 訊息與說明文件，除專有名詞外**一律使用繁體中文**。
3. **架構原則：**
   - 遵循 SOLID 原則，實作與介面解耦（`backend/excel/` 介面導向設計）。

---

## 📄 授權條款 (License)

內部專案，版權所有 (c) BOMIX Team.
