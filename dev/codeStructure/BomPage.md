# BomPage 架構說明

本文件詳細說明 BOMIX 前端主要頁面 `BomPage.jsx` 的架構設計。由於 BOMIX 支援多種不同的視圖（例如標準 BOM 表格、Matrix 模型對比、多專案比較 BigBOM 等），`BomPage` 採用了**策略模式 (Strategy Pattern)** 與 **複合元件 (Compound Components)** 的架構，以確保單一頁面不會因為邏輯過於龐大而難以維護。

## 1. 核心設計概念

- **容器與佈局分離**：`BomPage.jsx` 本身作為「容器元件 (Container)」，只負責處理與全域 Store (`useSeriesStore`, `useBomStore`, `useProjectStore`) 之間的互動，並處理大範圍的佈局（如 `ResizablePanelGroup`、左側 Sidebar 的狀態）。
- **通用工具列 (`BomToolbar.jsx`)**：負責將所有的檢視設定（如搜尋、過濾器、View 切換、匯出等）整合在一起，並將操作透過 Props 往上傳遞給容器，或直接更新 Store。
- **模式專屬視圖 (Views)**：當不同的 `bomMode` 啟用時（例如 `BOM` 或 `MATRIX`），容器會決定載入哪一個 View 元件（如 `StandardBomView.jsx` 或 `MatrixBomView.jsx`）。
- **核心資料表 (`BomTable.jsx`)**：身為底層的純粹 UI 元件，它不應該知道「現在是哪一個業務模式」，只負責接收資料結構 `data`、以及對應的 `columns` 定義，處理表格的渲染、虛擬滾動、行高亮、樹狀結構收合等基礎行為。

---

## 2. 目錄結構

```text
src/renderer/
  ├── pages/
  │   └── BomPage.jsx                # 主要入口：處理全域狀態、Layout 及 View 路由
  ├── components/
  │   ├── layout/
  │   │   └── BomSidebar.jsx         # 左側：專案與版本選取器
  │   ├── toolbar/
  │   │   └── BomToolbar.jsx         # 頂部：共用工具列 (Search, Filters, Export...)
  │   ├── views/
  │   │   ├── StandardBomView.jsx    # 標準模式 View 容器 (負責組裝標準 Columns 傳給 Table)
  │   │   └── MatrixBomView.jsx      # Matrix 模式 View 容器 (負責動態產生專案 Columns 傳給 Table)
  │   └── tables/
  │       └── BomTable.jsx           # 共用核心 DataGrid 元件 (處理 UI 渲染與虛擬滾動)
```

---

## 3. 各種 Table Views 的使用方式

目前 `BomPage` 支援兩種主要的檢視模式 (`bomMode`)，使用者可在側邊欄或特定設定中切換。

### A. StandardBomView (標準 BOM 模式)
- **觸發時機**：當 `bomMode === 'BOM'`，且通常選取了單一或多個 BOM 版本時。
- **職責**：將從 `useBomStore` 取回的 `filteredBom` 傳遞給 `BomTable`，並套用標準的 BOM 欄位（如 Location, Qty, BOM Status, Type, CCL, Remark 等）。
- **邏輯**：內部不需要處理跨專案的複雜表頭，直接展示單一 BOM 結構。

### B. MatrixBomView (矩陣對比模式)
- **觸發時機**：當 `bomMode === 'MATRIX'`，且選擇了一個或多個 BOM 版本時。
- **職責**：處理 Matrix 專屬的資料（從 `useMatrixStore` 取回的 Models 與 Selections）。
- **邏輯**：
  1. 它會將 `useMatrixStore` 中的選取狀態與 `bomView` 資料進行比對。
  2. 動態生成一至多個專案欄位 (Columns)，替換掉標準 BOM 模式下的「Qty, Location...」等欄位。
  3. 將包含 Checkbox 的自訂儲存格元件傳遞給 `BomTable`。

---

## 4. 如何新增一個新的 View (例如 BigBomView)

若未來需要新增一種新的檢視模式（例如跨專案巨量物料分析 `BIGBOM`），請依循以下步驟進行擴充：

### 步驟 1：建立新的 View 容器元件
在 `src/renderer/components/views/` 建立 `BigBomView.jsx`。
此元件需接收 `data` (基礎 BOM 資料)，並決定要渲染哪些特殊的 `columns`。

```javascript
import React from 'react';
import BomTable from '../tables/BomTable';

export default function BigBomView({ data, isLoading, searchTerm, searchFields, viewContextIds }) {
    // 1. (可選) 從對應的 BigBomStore 取得此模式專用的資料
    // const { extraData } = useBigBomStore();

    // 2. 渲染 BomTable，並指定 mode = "BIGBOM"
    return (
        <BomTable
            data={data}
            isLoading={isLoading}
            searchTerm={searchTerm}
            searchFields={searchFields}
            mode="BIGBOM"
            viewContextIds={viewContextIds}
        />
    );
}
```

### 步驟 2：在 BomTable.jsx 擴充欄位定義
開啟 `src/renderer/components/tables/BomTable.jsx`，找到 `columns = useMemo(() => { ... })` 區塊。
根據傳入的 `mode` (`BIGBOM`)，組合出適合此模式的 Columns。

```javascript
// 在 BomTable.jsx 內部
const columns = useMemo(() => {
    // ... baseCols 定義 (HHPN, Desc, Supplier...)

    if (mode === 'MATRIX') {
        return [...baseCols, ...matrixCols];
    } else if (mode === 'BIGBOM') {
        // 返回 BIGBOM 專屬的擴充欄位
        return [...baseCols, ...bigBomCols];
    } else {
        return [...baseCols, ...standardCols];
    }
}, [mode, ...]);
```

### 步驟 3：在 BomPage.jsx 註冊路由
開啟 `src/renderer/pages/BomPage.jsx`，找到 `MainContentView` 的判斷邏輯，加入新模式的切換：

```javascript
// View Routing
let MainContentView = null;
if (selectionCount > 0) {
    if (bomMode === 'MATRIX') {
        MainContentView = <MatrixBomView {...viewProps} />;
    } else if (bomMode === 'BIGBOM') {
        MainContentView = <BigBomView {...viewProps} />;
    } else {
        MainContentView = <StandardBomView {...viewProps} />;
    }
}
```

如此一來，新功能將能獨立運作，且不會影響現有 `StandardBomView` 或 `MatrixBomView` 的邏輯，同時享有 `BomToolbar` 的搜尋與過濾功能以及 `BomTable` 的渲染效能。