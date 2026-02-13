# BOMIX 編譯與打包說明

## 概述

BOMIX 使用 `electron-vite` 作為建置工具，`electron-builder` 作為打包工具。
所有操作均可透過 npm scripts 或 VS Code Tasks 執行。

## 前置需求

- Windows 10/11
- Node.js 20+
- 已執行 `npm install`

## 操作方式

### 方式一：VS Code Tasks（推薦）

使用 `Ctrl+Shift+P` → `Tasks: Run Task`，選擇對應的任務：

| Task 名稱 | 說明 |
|-----------|------|
| **BOMIX: Dev** | 啟動開發模式（Electron + Vite HMR） |
| **BOMIX: Build** | 編譯主行程、Preload、渲染層 |
| **BOMIX: Build Win** | 編譯 + 打包為 Windows 執行檔 |
| **BOMIX: Test** | 執行 Vitest 單元測試 |
| **BOMIX: Lint** | 執行 ESLint 檢查 |

也可使用 `Ctrl+Shift+B` 快速執行預設的 Build Task。

### 方式二：npm scripts

```powershell
# 開發模式
npm run dev

# 編譯
npm run build

# 打包為 Windows 執行檔（安裝版 + 便攜版）
npm run build:win

# 測試
npm run test

# Lint 檢查
npm run lint
```

## 打包產出

`npm run build:win` 完成後，產物位於 `dist/` 目錄：

| 檔案 | 說明 |
|------|------|
| `BOMIX Setup x.x.x.exe` | NSIS 安裝程式 |
| `BOMIX x.x.x.exe` | 便攜版（Portable），免安裝直接執行 |

## 注意事項

1. **better-sqlite3 原生模組**：首次安裝時需要編譯原生模組，確保系統已有 C/C++ 建置工具（`npm install --global windows-build-tools` 或 Visual Studio Build Tools）
2. **靜態資源**：`CHANGELOG.md` 會作為 `extraResources` 包含在打包產物中
3. **圖標**：應用程式圖標放置在 `resources/icon.ico`
4. **編譯產物**：`out/`（編譯中間產物）和 `dist/`（最終打包）已加入 `.gitignore`
