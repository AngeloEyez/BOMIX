# BOMIX - BOM 變化管理與追蹤工具

> 電子 BOM（Bill of Materials）版本管理與變化追蹤系統

## 功能特色

- 📋 **BOM 管理**：管理零件的廠商、料號、描述、位置、製程等資訊
- 🔄 **版本追蹤**：支援多專案、多 Phase（DB/SI/PV/MVB）、多版本 BOM 管理
- 📊 **版本比較**：比較不同版本 BOM 的差異，追蹤變化歷程
- 📥 **Excel 匯入/匯出**：從 Excel 解析 BOM 資料，也可匯出為 Excel 格式
- 🗄️ **本地資料庫**：每個系列存成獨立 SQLite 檔案，方便手動備份
- 🔗 **Second Source 管理**：每個 BOM 主項目可附帶多個替代料來源

## 技術架構

| 項目 | 技術 |
|------|------|
| 執行環境 | Electron |
| UI 框架 | React + Tailwind CSS v4 |
| 表格元件 | TanStack Table |
| 建置工具 | Vite (electron-vite) |
| 狀態管理 | Zustand |
| 資料庫 | SQLite (better-sqlite3) |
| Excel 處理 | xlsx (SheetJS) |
| 打包工具 | electron-builder |

## 快速開始

### 環境需求

- Node.js 20+
- npm 10+

### 安裝與開發

```powershell
# 安裝依賴
npm install

# 啟動開發模式
npm run dev

# 打包為 Windows 執行檔
npm run build:win

# 執行測試
npm run test
```

### VS Code Tasks

使用 `Ctrl+Shift+B` 或 `Ctrl+Shift+P` → `Tasks: Run Task` 快速執行建置任務。

## 目錄結構

詳見 [DIRECTORY.md](DIRECTORY.md)

## 授權

MIT License
