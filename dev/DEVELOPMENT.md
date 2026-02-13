# BOMIX 開發環境建置指南

## 系統需求

- **作業系統**：Windows 10/11
- **Node.js**：20.x 或以上版本
- **npm**：10.x 或以上版本
- **IDE**：VS Code / Antigravity（推薦）

## 常見問題故障排除

### 1. `npm run build:win` 失敗：Cannot create symbolic link

**原因**：Electron-builder 在打包過程中需要建立符號連結（Symbolic Links），但 Windows 預設安全性原則限制普通使用者建立符號連結。

**解決方案（推薦）：啟用 Windows 開發者模式**
1. 開啟 Windows **設定** (Settings)
2. 進入 **系統** > **開發人員專用** (System > For developers)
3. 開啟 **開發人員模式** (Developer Mode)
   - 這允許非管理員帳戶建立符號連結，且無需重新啟動 IDE。

**解決方案（替代）：以管理員身分執行**
1. 關閉 VS Code
2. 右鍵點選 VS Code 圖標，選擇「以系統管理員身分執行」
3. 再次執行 `npm run build:win`

## 快速建置

### 1. 複製專案

```powershell
git clone <repository-url>
cd BOMIX
```

### 2. 安裝依賴

```powershell
npm install
```

### 3. 啟動開發模式

```powershell
npm run dev
```

Electron 視窗會自動開啟，渲染層支援 HMR（Hot Module Replacement），修改 React 元件會即時更新。

## 可用指令

| 指令 | 說明 |
|------|------|
| `npm run dev` | 啟動開發模式（Electron + Vite HMR） |
| `npm run build` | 編譯主行程、Preload、渲染層 |
| `npm run build:win` | 編譯 + 打包為 Windows 執行檔 |
| `npm run build:unpack` | 編譯 + 打包為未壓縮目錄（測試用） |
| `npm run lint` | 執行 ESLint 檢查 |
| `npm run lint:fix` | 自動修復 ESLint 問題 |
| `npm run format` | 使用 Prettier 格式化程式碼 |
| `npm run test` | 執行 Vitest 單元測試 |
| `npm run test:watch` | 監控模式執行測試 |
| `npm run test:coverage` | 執行測試並產生覆蓋率報告 |

## IDE 設定

專案已包含 `.vscode/` 設定，開啟專案後 IDE 會自動偵測：
- Electron 除錯配置
- ESLint 整合
- VS Code Tasks（一鍵建置/測試）

### VS Code Tasks

專案提供預設的 VS Code Tasks，可透過 `Ctrl+Shift+B` 或 `Ctrl+Shift+P` → `Tasks: Run Task` 執行：
- **BOMIX: Dev** — 啟動開發模式
- **BOMIX: Build** — 編譯專案
- **BOMIX: Build Win** — 打包為 Windows 執行檔
- **BOMIX: Test** — 執行測試
- **BOMIX: Lint** — 程式碼檢查
