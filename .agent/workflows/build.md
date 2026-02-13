---
description: 建置說明 dev/prod/test 環境切換說明
---

# BOMIX 專案建置與打包說明

// turbo-all

> **建議使用 VS Code Tasks**（`Ctrl+Shift+B` 或 `Ctrl+Shift+P` → Tasks）執行以下操作，
> 這些 Tasks 已在 `.vscode/tasks.json` 中定義，不需要額外消耗 token。

## 開發環境建置說明

1. 啟動開發模式（Electron + Vite HMR）
```powershell
npm run dev
```

## 正式環境打包說明（使用 electron-builder 打包）

1. 打包正式版執行檔
```powershell
npm run build:win
```

## 測試環境建置

1. 執行測試
```powershell
npm run test
```

## 初次環境設定與必要套件安裝

1. 安裝所有依賴套件
```powershell
npm install
```