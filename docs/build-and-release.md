# BOMIX 建置、版本號管理與發布手冊 (Build & Release Guide)

本手冊規範 BOMIX 專案的**版本號單一來源 (Single Source of Truth, SSOT)** 架構、本地建置操作指令，以及透過 Git Tag 觸發 GitHub Actions 自動化打包發布的標準作業流程 (SOP)。

---

## 1. 版本號單一來源 (SSOT) 架構設計

為了避免人工在多處（如 `package.json`、`config.yml`、`info.json`、Go 原始碼）重複維護版本號導致不同步，BOMIX 確立以 **Git Tag** 作為正式發布時的唯一真實源頭：

```text
               【Git Tag (例: v1.2.0)】
                          │
            ┌─────────────┴─────────────┐
            ▼                           ▼
  [本地建置 / 測試]               [GitHub Actions CI]
  task build VERSION=1.2.0      .github/workflows/wails-release.yml
            │                           │
            └─────────────┬─────────────┘
                          ▼
            【Taskfile (統一建置核心)】
                          │
         ┌────────────────┴────────────────┐
         ▼                                 ▼
   [info.json 同步]                [Go -ldflags 編譯注入]
   - scripts/sync-version.ps1      -X Version=1.2.0
   - 產生 Windows .syso 檔案       -X GitCommit=7a1b2c3
   - 寫入 .exe 檔案右鍵屬性         -X BuildTime=2026-09-21T...
                                           │
                                           ▼
                                 【後端 version 套件】
                                           │
                          ┌────────────────┴────────────────┐
                          ▼                                 ▼
                 [Wails 原生綁定 API]             [前端 UI 呈現]
                 - App.GetVersion()               - Settings -> About
                 - App.GetAppInfo()               - 版本徽章、Commit、建置時間
```

### 核心原則：
1. **日常開發零負擔**：日常開發中無需改動任何版本號檔案，預設自動標記為 `dev`，Commit 與建置時間自動推導。
2. **建置核心單一化**：本地開發者與 GitHub Actions CI **一律統一透過 Taskfile (`task build`)** 執行，杜絕維護多份編譯腳本的風險。
3. **編譯期動態注入**：版本號由編譯器 `-ldflags` 注入 Go 後端，並藉由 `scripts/sync-version.ps1` 自動同步 Windows 執行檔右鍵屬性 (`info.json`)，無任何硬編碼。

---

## 2. 本地建置操作指南 (Local Build)

### 前置需求
請確保本機環境已具備以下工具：
- **Go**: 1.21+ (建議 1.24 或 1.25)
- **Node.js**: v22+ (LTS)
- **Task**: [TaskfileRunner](https://taskfile.dev/)
- **Wails v3 CLI**:
  ```powershell
  go install github.com/wailsapp/wails/v3/cmd/wails3@latest
  ```

### 常用建置指令

所有建置任務皆在 `bomix-app/` 目錄下執行：

#### 1. 預設本機建置 (標記為 dev 版本)
```powershell
cd bomix-app
task build
```
- 自動執行前端編譯 (`npm run build`)
- 將執行檔屬性標記為 `0.0.0 (dev)`
- 注入版本號 `dev`、當前 Git Commit 短碼與時間
- 輸出檔案：`bomix-app/bin/BOMIX.exe`

#### 2. 模擬特定版本建置 (例如 1.2.0)
若本機需要測試特定正式版本或確認安裝檔打包：
```powershell
cd bomix-app
task build VERSION=1.2.0
```
- 自動將 Windows 執行檔屬性同步為 `1.2.0`
- 透過 `-ldflags` 注入 `Version=1.2.0`
- 進入程式 Settings 頁面即可看到 `v1.2.0` 徽章

#### 3. 執行開發模式 (熱重載)
```powershell
cd bomix-app
task dev
# 或
wails3 dev
```

---

## 3. 版本發布操作流程 (Release SOP)

BOMIX 採用 Git Tag 驅動自動化發布。當新版本功能開發、測試完畢並合併至相應分支後，只需建立符合語意化版本（Semantic Versioning）的 Git Tag 並推送至遠端，CI/CD 將自動接手打包與發布。

### 步驟說明

#### 步驟 1：確保工作區乾淨且所有測試通過
在發布前，請在本機確認所有後端測試與前端編譯皆正常：
```powershell
# 1. 執行後端單元測試
cd bomix-app
go test ./backend/...

# 2. 執行前端型別檢查與編譯
cd frontend
npm run build
cd ..
```

#### 步驟 2：建立 Git Tag
版本 Tag 格式必須遵循 `v<Major>.<Minor>.<Patch>`（例如 `v1.0.0`、`v1.2.1`）：
```powershell
# 建立帶有註解的 annotated tag
git tag -a v1.0.0 -m "Release v1.0.0: 正式釋出 BOMIX 初版"
```

#### 步驟 3：推送 Tag 至 GitHub
```powershell
git push origin v1.0.0
```

#### 步驟 4：自動化 CI 流水線處理
推送 Tag 後，GitHub Actions 會自動觸發 `.github/workflows/wails-release.yml`：
1. **環境檢出**：完整取得 Repository 與 Git 歷史紀錄。
2. **版本號解析**：自動將 `v1.0.0` 轉換為應用程式版本號 `1.0.0`。
3. **正式/測試版判定**：
   - 若該 Tag 建立於 `main` 分支的提交歷史中，自動判定為**正式釋出 (Release)**。
   - 若該 Tag 建立於開發或功能分支，自動判定為**預發布/測試版 (Pre-release / Beta)**。
4. **Taskfile 核心建置**：調用 `task build VERSION=1.0.0` 生成標準 Windows 執行檔。
5. **打包產物**：將二進位檔封裝為帶有版本資訊的壓縮包 `BOMIX_<版本號>.zip`（例如 `BOMIX_v1.0.0.zip` 或 `BOMIX_v1.11.zip`）。
6. **建立 GitHub Release**：自動建立對應 Release 頁面並上傳 ZIP 檔案供使用者下載。

---

## 4. UI 介面版本號檢視

啟動應用程式後，可透過介面檢視當前版本與建置環境：
1. 點擊頂部或側邊導航前往 **Settings (系統設定)** 頁面。
2. 在左側導航樹最下方點選 **About** 節點（或手動滾動右側頁面到底端）。
3. 頁面將呈現「About」卡片：
   - **應用程式名稱與版本徽章** (例如 `v1.2.0` 或 `dev`)
   - **版本號 (Version)**
   - **Git Commit 雜湊碼**
   - **建置時間 (Build Time)**
   - **核心技術架構說明**
