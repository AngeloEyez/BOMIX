# BOMIX 使用者說明書建置與維護指南

本目錄包含 BOMIX 官方使用者操作手冊的完整原始碼、排版樣式、操作截圖與 PDF 自動建置工具。
（本資料夾位於專案根目錄下 `manual/`，與 `docs/`、`bomix-app/` 平行）

---

## 目錄結構說明

```text
manual/
├── README.md               # 本維護與建置說明指南
├── manual.md               # 說明書 Markdown 原始檔 (主要內容)
├── package.json            # 建置工具相依性定義
├── build.js                # 一鍵建置腳本 (支援 Puppeteer 與 Edge 雙引擎)
├── styles/
│   └── manual-pdf.css      # 現代高端專業排版樣式 (A4 規格、封面、目錄、防截斷)
├── images/                 # 操作畫面截圖存放處
│   ├── 01-welcome-screen.png   # 首頁歡迎畫面
│   ├── 02-import-dialog.png    # 批次匯入對話框
│   ├── 03-workspace-table.png  # 工作區主表格與導覽
│   └── 04-export-bigmatrix.png # 匯出 BigMatrix 設定
└── dist/                   # 建置產出目錄
    ├── BOMIX-使用者手冊.html    # 獨立預覽 HTML (圖片皆已自動 Base64 內嵌)
    └── BOMIX-使用者手冊.pdf     # 出版級高解析 A4 PDF 檔案
```

---

## 快速建置操作

### 方式一：一鍵建置 PDF（推薦，最簡便）

本建置腳本內建「雙引擎設計」：
1. 若有安裝 `puppeteer`，則透過 Puppeteer 背景列印。
2. 若未安裝任何依賴，腳本會自動調用 Windows 11 系統內建的 **Microsoft Edge (Chromium 核心)** 進行無頭輸出，**零依賴即可秒速產出向量 PDF**！

在終端機進入專案目錄執行：
```powershell
node manual/build.js
```
或進入目錄執行：
```powershell
cd manual
node build.js
```

產出成果：
- `manual/dist/BOMIX-使用者手冊.pdf`
- `manual/dist/BOMIX-使用者手冊.html`

### 方式二：僅生成 HTML 網頁預覽
```powershell
node manual/build.js --html-only
```
完成後可直接在瀏覽器中雙擊 `dist/BOMIX-使用者手冊.html` 檢視效果。

---

## 操作畫面截圖更新與替換指引

系統預設內建了 6 張高解析度且具備現代 Fluent 質感的向量示意圖。若您希望更換為本機實際運行的軟體操作畫面截圖，請參考下方清單截取並替換：

| 檔案名稱 | 建議格式 | 對應畫面內容 | 截圖建議情境 / 操作步驟 |
| :--- | :--- | :--- | :--- |
| **`01-welcome-screen.png`** | PNG (視窗比例) | 首頁歡迎畫面 | 啟動軟體後的首頁，顯示「新增系列」、「開啟現有系列」及右側的「最近開啟」紀錄。 |
| **`02-import-dialog.png`** | PNG (裁切彈窗視窗) | 批次匯入對話框 | 點擊 `BOM ▾` -> `Import BOM`，拖入 2~3 個 Excel 檔案後之清單視窗（展示 EBOM 與 Matrix 標籤）。 |
| **`03-workspace-table.png`** | PNG (全視窗) | 工作區主表格 | 左側側邊欄勾選專案版本後，右側 BOM 主表格完整呈現主料、替代料（↳）及視圖切換器之畫面。 |
| **`04-export-bigmatrix.png`** | PNG (主工作區) | 匯出 BigMatrix 設定 | 點擊頂部 `Export`，選擇 BigMatrix 格式，呈現卡片拖曳排序與 Model 數量輸入框之畫面。 |

> **替換說明**：
> 截圖完成後，只需將檔案命名為上述檔名（例如 `01-welcome-screen.png`），放入 `manual/images/` 資料夾，重新執行 `node manual/build.js`，腳本會自動讀取並內嵌新圖片！

---

## 與 GitHub CI / 主程式自動打包整合

若要在每次 GitHub Actions 觸發 Release 或 CI 建置時，**全自動編譯最新版本說明書，並與 `BOMIX.exe` 共同打包至同一 ZIP 壓縮檔**，請參考以下 CI 設定範例：

### GitHub Actions Workflow 整合範例 (`.github/workflows/release.yml`)

```yaml
name: Release BOMIX Application & Manual

on:
  push:
    tags:
      - 'v*'

jobs:
  build-and-release:
    runs-on: windows-latest
    steps:
      - name: Checkout Code
        uses: actions/checkout@v4

      - name: Setup Go Environment
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'

      # 1. 建置前端與 Wails 桌面應用程式
      - name: Install Frontend Dependencies
        working-directory: bomix-app/frontend
        run: npm ci

      - name: Build BOMIX Executable
        run: |
          go install github.com/wailsapp/wails/v3/cmd/wails3@latest
          cd bomix-app
          wails3 build -platform windows/amd64

      # 2. 自動編譯最新版使用說明書 PDF
      - name: Build User Manual PDF
        working-directory: manual
        run: |
          npm ci
          node build.js

      # 3. 將 .exe 與說明書 PDF 共同打包至 Release ZIP
      - name: Package Release ZIP
        shell: pwsh
        run: |
          $distFolder = "release-dist"
          New-Item -ItemType Directory -Path $distFolder -Force
          
          # 複製可執行檔與說明書
          Copy-Item "bomix-app/bin/BOMIX.exe" -Destination "$distFolder/"
          Copy-Item "manual/dist/BOMIX-使用者手冊.pdf" -Destination "$distFolder/"
          
          # 壓縮為單一發布包
          Compress-Archive -Path "$distFolder/*" -DestinationPath "BOMIX-${{ github.ref_name }}-windows-amd64.zip"

      # 4. 發布至 GitHub Releases
      - name: Upload Release Asset
        uses: softprops/action-gh-release@v2
        with:
          files: BOMIX-${{ github.ref_name }}-windows-amd64.zip
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```
