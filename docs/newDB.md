# BOMIX 資料庫架構優化方案

> 本文檔描述 Location 原子化儲存的最佳化架構設計，以及從當前 v1 分支的遷移方案。

---

## 目錄

1. [資料結構設計](#1-資料結構設計)
2. [匯入方式](#2-匯入方式)
3. [儲存方式](#3-儲存方式)
4. [JOIN 與記憶體聚合](#4-join-與記憶體聚合)
5. [查詢方式](#5-查詢方式)
6. [v1 專案架構分析](#6-v1-專案架構分析)
7. [修改方案計畫](#7-修改方案計畫)

---

## 1. 資料結構設計

### 1.1 資料表結構

#### Series (系列元資料表)
| 欄位 | 型別 | 約束 | 說明 |
|------|------|------|------|
| id | INTEGER | PRIMARY KEY | 自增主鍵 |
| name | TEXT | NOT NULL, UNIQUE | 系列名稱 |
| description | TEXT | | 描述 |
| last_export_path | TEXT | | 上次匯出路徑 |
| created_at | DATETIME | | 建立時間 |
| updated_at | DATETIME | | 更新時間 |
| deleted_at | DATETIME | | 軟刪除時間 |

#### Project (專案表)
| 欄位 | 型別 | 約束 | 說明 |
|------|------|------|------|
| id | INTEGER | PRIMARY KEY | 自增主鍵 |
| series_id | INTEGER | NOT NULL, INDEX | 關聯系列 |
| code | TEXT | NOT NULL, UNIQUE | 專案代碼 (系列內唯一) |
| description | TEXT | | 描述 |
| created_at | DATETIME | | 建立時間 |
| updated_at | DATETIME | | 更新時間 |
| deleted_at | DATETIME | | 軟刪除時間 |

#### BomRevision (BOM 版本表)
| 欄位 | 型別 | 約束 | 說明 |
|------|------|------|------|
| id | INTEGER | PRIMARY KEY | 自增主鍵 |
| project_id | INTEGER | NOT NULL, INDEX | 關聯專案 |
| phase | TEXT | NOT NULL | Phase 名稱 (DB, SI, PV 等) |
| version | TEXT | NOT NULL | 版本號 (0.1, 0.2 等) |
| description | TEXT | | BOM 描述 |
| schematic_version | TEXT | | 電路圖版本 |
| pcb_version | TEXT | | PCB 版本 |
| pca_pn | TEXT | | PCA 料號 |
| date | TEXT | | BOM 日期 |
| mode | TEXT | DEFAULT 'NPI' | NPI 或 MP 模式 |
| source_file | TEXT | | 來源 Excel 檔名 |
| created_at | DATETIME | | 建立時間 |
| updated_at | DATETIME | | 更新時間 |
| deleted_at | DATETIME | | 軟刪除時間 |

**Unique Constraint**: `(project_id, phase, version)` 組合唯一

#### Part (主料表) - 物料基本資訊表
| 欄位 | 型別 | 約束 | 說明 |
|------|------|------|------|
| id | INTEGER | PRIMARY KEY | 自增主鍵 |
| revision_id | INTEGER | NOT NULL, INDEX | 關聯 BOM 版本 |
| type | TEXT | NOT NULL, INDEX | 類型 (Main, 2nd Source) |
| supplier | TEXT | NOT NULL, INDEX | 供應商名稱 |
| supplier_pn | TEXT | NOT NULL, INDEX | 供應商料號 |
| description | TEXT | | 零件描述 |
| cost | REAL | | 成本 |
| remark | TEXT | | 註記 |
| created_at | DATETIME | | 建立時間 |
| updated_at | DATETIME | | 更新時間 |
| deleted_at | DATETIME | | 軟刪除時間 |

**索引**:
- `idx_part_revision_supplier_pn`: `(revision_id, supplier, supplier_pn)` - 用於物料群組查詢
- `idx_part_revision_type`: `(revision_id, type)` - 用於類型過濾

#### PartLocation (Location 獨立表)
| 欄位 | 型別 | 約束 | 說明 |
|------|------|------|------|
| id | INTEGER | PRIMARY KEY | 自增主鍵 |
| part_id | INTEGER | NOT NULL, INDEX | 關聯主料 |
| location | TEXT | NOT NULL, INDEX | 零件位置編號 |
| bom_status | TEXT | DEFAULT 'I', INDEX | BOM 狀態 (跟著 Location) |
| ccl | BOOLEAN | DEFAULT false, INDEX | 是否為 CCL 物料 (true: 是, false: 否) |

**索引**:
- `idx_part_location_part`: `(part_id)` - 用於 JOIN 查詢
- `idx_part_location_name`: `(location)` - 用於精確查詢
- `idx_part_location_status`: `(bom_status)` - 用於狀態過濾
- `idx_part_location_ccl`: `(ccl)` - 用於 CCL 過濾
- **複合索引**: `idx_part_loc_status_ccl`: `(bom_status, ccl)` - 用於視圖過濾

#### SecondSource (替代料表)
| 欄位 | 型別 | 約束 | 說明 |
|------|------|------|------|
| id | INTEGER | PRIMARY KEY | 自增主鍵 |
| revision_id | INTEGER | NOT NULL, INDEX | 關聯 BOM 版本 |
| part_id | INTEGER | NOT NULL, INDEX | 關聯主料 |
| supplier | TEXT | NOT NULL | 供應商名稱 |
| supplier_pn | TEXT | NOT NULL | 供應商料號 |
| description | TEXT | | 零件描述 |
| cost | REAL | | 成本 |
| lead_time | INTEGER | | 交貨期 |
| created_at | DATETIME | | 建立時間 |
| updated_at | DATETIME | | 更新時間 |

#### MatrixModel (Matrix 模型表)
| 欄位 | 型別 | 約束 | 說明 |
|------|------|------|------|
| id | INTEGER | PRIMARY KEY | 自增主鍵 |
| revision_id | INTEGER | NOT NULL, INDEX | 關聯 BOM 版本 |
| model_name | TEXT | NOT NULL, UNIQUE | Model 名稱 (A, B, C...) |
| qty | INTEGER | DEFAULT 1 | 打件數量 |
| created_at | DATETIME | | 建立時間 |
| updated_at | DATETIME | | 更新時間 |

#### MatrixSelection (Matrix 勾選表)
| 欄位 | 型別 | 約束 | 說明 |
|------|------|------|------|
| id | INTEGER | PRIMARY KEY | 自增主鍵 |
| revision_id | INTEGER | NOT NULL, INDEX | 關聯 BOM 版本 |
| model_id | INTEGER | NOT NULL, INDEX | 關聯 Matrix 模型 |
| part_id | INTEGER | NOT NULL, INDEX | 關聯被選中的物料 |
| group | TEXT | NOT NULL, INDEX | 群組鍵 (main_supplier + main_supplier_pn) |
| material | TEXT | NOT NULL, INDEX | 物料鍵 (supplier + supplier_pn) |
| selected_supplier | TEXT | | 選中的供應商 |
| selected_supplier_pn | TEXT | | 選中的供應商料號 |
| is_auto_selected | BOOLEAN | DEFAULT false | 是否自動選中 |
| created_at | DATETIME | | 建立時間 |
| updated_at | DATETIME | | 更新時間 |

---

### 1.2 資料關係圖

```
Series (1) ──< Project (1) ──< BomRevision (1) ──< Part (N)
                                                         │
                                                         │
                                            ┌────────────┴────────────┐
                                            │                         │
                                          (1)                       (N)
                                            │                         │
                                  PartLocation (N)       SecondSource (N)
                                            │
                                    bom_status, ccl
                                    (跟著 Location 走)

BomRevision (1) ──< MatrixModel (N) ──< MatrixSelection (N)
                                              │
                                              │
                                         part_id (關聯 Part)
```

---

### 1.3 關鍵設計原則

| 原則 | 說明 |
|------|------|
| **Location 原子化** | 每個 location 為獨立紀錄，存在 `part_locations` 表 |
| **CCL/Status 跟 Location** | `bom_status` 和 `ccl` (bool) 欄位移至 `part_locations` 表，因為它們是跟著 location 走 |
| **Part 純化物料資訊** | `parts` 表僅存放獨立物料屬性，不再存放 location, quantity, bom_status, ccl 等資訊 |
| **Type 跟著 Part** | `type` 欄位 (SMD/PTH/BOTTOM) 存在 `parts` 表，因為它是物料屬性 |
| **物料去重** | 相同 `(supplier, supplier_pn)` 的物料只存一筆在 `parts` 表 |
| **Location 聚合** | 查詢時使用 `GROUP_CONCAT` 或記憶體聚合將 location 合併為逗號分隔字串 |

---

## 2. 匯入方式

### 2.1 EBOM 匯入流程

```
Excel 檔案 → 格式偵測 → 表頭解析 → 零件解析 → Mode 判斷 → 資料庫寫入
```

#### 步驟 1: 格式偵測
- 檢查 Excel 工作表結構
- EBOM: 包含 SMD/PTH/BOTTOM/NI/PROTO/MP sheets
- BigMatrix: 包含 BigMatrix sheet
- Matrix: 包含 SMD sheet (暫不支援)

#### 步驟 2: 表頭解析 (從 SMD Sheet)
| 儲存格 | 欄位 | 解析規則 |
|--------|------|----------|
| B3 | project_code | "Product Code: {value}" |
| B4 | description | "Description: {value}" |
| D3 | schematic_version | "Schematic Version: {value}" |
| J3 | phase | "Phase: {value}" |
| F3 | pcb_version | "PCB Version: {value}" |
| F4 | pca_pn | "PCA PN: {value}" |
| H3 | version | "BOM Version: {value}" |
| H4 | date | "Date: {value}" |

#### 步驟 3: 零件解析與兩階段匯入流程

```
[Phase 1] 製程頁面 (SMD/PTH/BOTTOM) & NI 頁面
  ├─ 建立 Parts (主料去重 deduplication)
  └─ 建立 PartLocations (bom_status = 'I' 或 'X', ccl = false)
                      ↓
[Phase 2] 狀態與屬性頁面 (PROTO & CCL 頁面)
  ├─ PROTO 頁面：僅更新既有 Location 的 bom_status = 'P'
  └─ CCL 頁面：僅更新既有 Location 的 ccl = true
```

##### 階段一 (Phase 1)：物料去重與 Location 建置

| Sheet 來源 | 處理規則 | Part (主料) 處理 | PartLocation 處理 |
|-----------|---------|-----------------|------------------|
| **SMD / PTH / BOTTOM** | 主製程頁面 | 依 `(supplier, supplier_pn)` 去重，若已存在則重用既有 Part ID；若無則建立新 Part (`type` = Sheet Name) | Location 原子化，寫入 `part_locations` (`bom_status` = `'I'`, `ccl` = `false`) |
| **NI** | 不上件頁面 | 依 `(supplier, supplier_pn)` 比對，若已存在則重用既有 Part ID；若無則建立新 Part (`type` = `'NI'`) | Location 原子化，寫入 `part_locations` (`bom_status` = `'X'`, `ccl` = `false`) |

##### 階段二 (Phase 2)：Location 屬性與狀態覆寫

| Sheet 來源 | 處理規則 | 動作說明 |
|-----------|---------|---------|
| **PROTO** | Proto 狀態頁面 | 不建立新物料，僅尋找 Phase 1 已建立的匹配 Location，將其 `bom_status` 欄位更新為 `'P'` |
| **CCL** | Critical Part 頁面 | 不建立新物料，僅尋找 Phase 1 已建立的匹配 Location，將其 `ccl` 欄位更新為 `true` |

##### 解析範例程式碼 (Go)

```go
// 兩階段 EBOM 解析與資料建置
type ParsedData struct {
    Parts         []*Part
    PartLocations []*PartLocation
    SecondSources []*SecondSource
}

func parseEBOM(sheets map[string][][]string, revisionID int64) (*ParsedData, error) {
    partMap := make(map[string]*Part)            // Key: "supplier|supplier_pn" (用於主料去重)
    partList := make([]*Part, 0)
    locationList := make([]*PartLocation, 0)
    secondSourceList := make([]*SecondSource, 0)
    
    // ================= Phase 1: 建立 Part 與 PartLocation =================
    
    // Step 1.1: 處理主製程頁面 (SMD, PTH, BOTTOM)
    mainSheets := []string{"SMD", "PTH", "BOTTOM"}
    for _, sheetName := range mainSheets {
        rows, ok := sheets[sheetName]
        if !ok {
            continue
        }
        
        var currentMainPart *Part
        for i := 5; i < len(rows); i++ {
            row := rows[i]
            item := strings.TrimSpace(row[0])
            
            if item != "" { // Main Source
                supplier := strings.TrimSpace(row[1])
                supplierPN := strings.TrimSpace(row[2])
                locStr := strings.TrimSpace(row[3])
                
                if supplier == "" || supplierPN == "" {
                    continue
                }
                
                key := fmt.Sprintf("%s|%s", supplier, supplierPN)
                part, exists := partMap[key]
                if !exists {
                    // 主料不存在：建立新主料
                    part = &Part{
                        RevisionID:  revisionID,
                        Type:        sheetName, // SMD / PTH / BOTTOM
                        Supplier:    supplier,
                        SupplierPN:  supplierPN,
                        Description: strings.TrimSpace(row[4]),
                        Cost:        parseCost(row[5]),
                        Remark:      strings.TrimSpace(row[6]),
                    }
                    partMap[key] = part
                    partList = append(partList, part)
                }
                currentMainPart = part
                
                // 原子化 Location 並建立 PartLocation (bom_status = 'I')
                for _, loc := range atomizeLocation(locStr) {
                    locationList = append(locationList, &PartLocation{
                        Part:      part,
                        Location:  loc,
                        BomStatus: "I",
                        CCL:       false,
                    })
                }
            } else if currentMainPart != nil { // Second Source
                secondSource := parseSecondSource(row, revisionID, currentMainPart)
                if secondSource != nil {
                    secondSourceList = append(secondSourceList, secondSource)
                }
            }
        }
    }
    
    // Step 1.2: 處理 NI 頁面 (不上件, bom_status = 'X')
    if niRows, ok := sheets["NI"]; ok {
        for i := 5; i < len(niRows); i++ {
            row := niRows[i]
            supplier := strings.TrimSpace(row[1])
            supplierPN := strings.TrimSpace(row[2])
            locStr := strings.TrimSpace(row[3])
            
            if supplier == "" || supplierPN == "" {
                continue
            }
            
            key := fmt.Sprintf("%s|%s", supplier, supplierPN)
            part, exists := partMap[key]
            if !exists {
                // 主料不存在：建立新主料
                part = &Part{
                    RevisionID:  revisionID,
                    Type:        "NI",
                    Supplier:    supplier,
                    SupplierPN:  supplierPN,
                    Description: strings.TrimSpace(row[4]),
                }
                partMap[key] = part
                partList = append(partList, part)
            }
            
            for _, loc := range atomizeLocation(locStr) {
                locationList = append(locationList, &PartLocation{
                    Part:      part,
                    Location:  loc,
                    BomStatus: "X", // 不上件
                    CCL:       false,
                })
            }
        }
    }
    
    // ================= Phase 2: Location 狀態與屬性覆寫更新 =================
    
    // 建立 Location 快速查詢 Mapping (Location 名稱 -> *PartLocation 紀錄)
    locIndexMap := make(map[string]*PartLocation)
    for _, locObj := range locationList {
        locIndexMap[locObj.Location] = locObj
    }
    
    // Step 2.1: 處理 PROTO 頁面 (僅將對應 Location 的 bom_status 更新為 'P')
    if protoRows, ok := sheets["PROTO"]; ok {
        for i := 5; i < len(protoRows); i++ {
            locStr := strings.TrimSpace(protoRows[i][3])
            for _, loc := range atomizeLocation(locStr) {
                if targetLoc, exists := locIndexMap[loc]; exists {
                    targetLoc.BomStatus = "P"
                }
            }
        }
    }
    
    // Step 2.2: 處理 CCL 頁面 (僅將對應 Location 的 ccl 更新為 true)
    if cclRows, ok := sheets["CCL"]; ok {
        for i := 5; i < len(cclRows); i++ {
            locStr := strings.TrimSpace(cclRows[i][3])
            for _, loc := range atomizeLocation(locStr) {
                if targetLoc, exists := locIndexMap[loc]; exists {
                    targetLoc.CCL = true
                }
            }
        }
    }
    
    return &ParsedData{
        Parts:         partList,
        PartLocations: locationList,
        SecondSources: secondSourceList,
    }, nil
}

func atomizeLocation(locationStr string) []string {
    parts := strings.Split(locationStr, ",")
    var atomized []string
    for _, p := range parts {
        trimmed := strings.TrimSpace(p)
        if trimmed != "" {
            atomized = append(atomized, trimmed)
        }
    }
    return atomized
}
```

#### 步驟 4: Mode 判斷

```go
func determineMode(mainLocations, protoLocations, mpLocations map[string]bool) string {
    // NPI Mode: PROTO 中有零件出現在主製程中
    for loc := range protoLocations {
        if mainLocations[loc] {
            return "NPI"
        }
    }
    
    // MP Mode: MP 中有零件出現在主製程中
    for loc := range mpLocations {
        if mainLocations[loc] {
            return "MP"
        }
    }
    
    return "NPI" // 預設
}
```

#### 步驟 5: 資料庫寫入

```go
func saveEBOM(db *gorm.DB, revisionID int64, data *ParsedData) error {
    return db.Transaction(func(tx *gorm.DB) error {
        // 1. 刪除該 Revision 舊資料
        if err := tx.Where("revision_id = ?", revisionID).Delete(&Part{}).Error; err != nil {
            return err
        }
        if err := tx.Where("revision_id = ?", revisionID).Delete(&PartLocation{}).Error; err != nil {
            return err
        }
        if err := tx.Where("revision_id = ?", revisionID).Delete(&SecondSource{}).Error; err != nil {
            return err
        }

        // 2. 批次插入 Parts (寫入後取得 GORM 生成的 ID)
        if len(data.Parts) > 0 {
            if err := tx.CreateInBatches(data.Parts, 500).Error; err != nil {
                return err
            }
        }

        // 3. 更新 PartLocations 的 PartID 並批次插入
        for _, locObj := range data.PartLocations {
            if locObj.Part != nil {
                locObj.PartID = locObj.Part.ID
            }
        }
        if len(data.PartLocations) > 0 {
            if err := tx.CreateInBatches(data.PartLocations, 500).Error; err != nil {
                return err
            }
        }

        // 4. 更新 SecondSources 的 PartID 與 RevisionID 並批次插入
        for _, sec := range data.SecondSources {
            if sec.Part != nil {
                sec.PartID = sec.Part.ID
            }
        }
        if len(data.SecondSources) > 0 {
            if err := tx.CreateInBatches(data.SecondSources, 500).Error; err != nil {
                return err
            }
        }

        return nil
    })
}
```

---

## 3. 儲存方式

### 3.1 資料儲存範例

**Excel 原始資料**:
```
┌─────┬─────────┬──────────────┬─────────────────────────────┬─────────┬────────┐
│ item│ supplier│ supplier_pn  │ location                    │ type    │ ccl    │
├─────┼─────────┼──────────────┼─────────────────────────────┼─────────┼────────┤
│  1  │ Samsung │ CL05B104KO5  │ C1,C2,C3,C4,C5              │ SMD     │ Y      │
└─────┴─────────┴──────────────┴─────────────────────────────┴─────────┴────────┘
```

**parts 表儲存**:
```
┌────┬───────────────┬─────────┬──────────────┬──────────────┐
│ id │ revision_id   │ type    │ supplier     │ supplier_pn  │
├────┼───────────────┼─────────┼──────────────┼──────────────┤
│ 101│ 5             │ Main    │ Samsung      │ CL05B104     │
└────┴───────────────┴─────────┴──────────────┴──────────────┘
```

**part_locations 表儲存**:
```
┌────┬──────────────┬─────────┬──────────┬────────┐
│ id │ part_id      │ location│ bom_status│ ccl   │
├────┼──────────────┼─────────┼──────────┼────────┤
│ 1  │ 101          │ C1      │ I        │ true   │
├────┼──────────────┼─────────┼──────────┼────────┤
│ 2  │ 101          │ C2      │ I        │ true   │
├────┼──────────────┼─────────┼──────────┼────────┤
│ 3  │ 101          │ C3      │ I        │ true   │
├────┼──────────────┼─────────┼──────────┼────────┤
│ 4  │ 101          │ C4      │ I        │ true   │
├────┼──────────────┼─────────┼──────────┼────────┤
│ 5  │ 101          │ C5      │ I        │ true   │
└────┴──────────────┴─────────┴──────────┴────────┘
```

### 3.2 儲存效率分析

| 設計 | 100 個 location 的儲存 | 備註 |
|------|---------------------|------|
| **舊設計 (單表)** | 100 筆完整紀錄 | 每筆含所有欄位，冗餘度高 |
| **新設計 (正規化)** | 1 筆 (parts) + 100 筆 (part_locations) | parts 僅存必要欄位，location 表僅存 location + status + ccl |
| **節省** | ~60-70% 冗餘減少 | |

---

## 4. JOIN 與記憶體聚合

### 4.1 JOIN 查詢

**單一 BOM 完整資料查詢**:

```sql
SELECT 
    p.id,
    p.revision_id,
    p.type,
    p.supplier,
    p.supplier_pn,
    p.description,
    p.cost,
    p.remark,
    GROUP_CONCAT(pl.location, ',') AS locations,
    COUNT(pl.id) AS quantity,
    -- 注意：bom_status 可能多個值，需根據視圖過濾
    MIN(pl.bom_status) AS bom_status,
    MAX(pl.ccl) AS ccl
FROM parts p
LEFT JOIN part_locations pl ON p.id = pl.part_id
WHERE p.revision_id = ?
GROUP BY p.id
ORDER BY p.id;
```

**視圖過濾查詢 (SMD)**:

```sql
SELECT 
    p.*,
    GROUP_CONCAT(pl.location, ',') AS locations,
    COUNT(pl.id) AS quantity
FROM parts p
JOIN part_locations pl ON p.id = pl.part_id
WHERE p.revision_id = ?
  AND pl.bom_status IN ('I', 'P')  -- NPI Mode
GROUP BY p.id;
```

**CCL 視圖查詢**:

```sql
SELECT 
    p.*,
    GROUP_CONCAT(pl.location, ',') AS locations,
    COUNT(pl.id) AS quantity
FROM parts p
JOIN part_locations pl ON p.id = pl.part_id
WHERE p.revision_id = ?
  AND pl.ccl = true
GROUP BY p.id;
```

### 4.2 記憶體聚合演算法

```go
func executeView(db *gorm.DB, revisionIDs []int64, viewType string, mode string) ([]AggregatedPart, error) {
    // 1. 一次性 JOIN 查詢所有原始資料
    var rawResults []struct {
        PartID       int64
        RevisionID   int64
        Supplier     string
        SupplierPN   string
        Type         string
        Location     string
        BomStatus    string
        CCL          bool
    }
    
    query := `
        SELECT 
            p.id as part_id, p.revision_id, p.supplier, p.supplier_pn,
            p.type, pl.location, pl.bom_status, pl.ccl
        FROM parts p
        JOIN part_locations pl ON p.id = pl.part_id
        WHERE p.revision_id IN ?
    `
    
    // 根據視圖類型添加過濾
    if viewType == "SMD" || viewType == "PTH" || viewType == "BOTTOM" {
        query += ` AND p.type = ? AND pl.bom_status IN ?`
        // 執行時追加參數
    } else if viewType == "CCL" {
        query += ` AND pl.ccl = true`
    } else if viewType == "NI" {
        query += ` AND pl.bom_status = 'X'`
    } else {
        // ALL 視圖：根據 Mode 過濾
        statuses := []string{"I", "P"}
        if mode == "MP" {
            statuses = []string{"I", "M"}
        }
        query += ` AND pl.bom_status IN ?`
        // 執行時追加參數
    }
    
    err := db.Raw(query, append([]interface{}{revisionIDs}, filterParams...)...).Scan(&rawResults).Error
    if err != nil {
        return nil, err
    }
    
    // 2. 記憶體聚合 (O(n) 複雜度)
    groupedMap := make(map[string]*AggregatedPart)
    
    for _, row := range rawResults {
        key := fmt.Sprintf("%s|%s", row.Supplier, row.SupplierPN)
        
        if _, exists := groupedMap[key]; !exists {
            groupedMap[key] = &AggregatedPart{
                PartID:       row.PartID,
                RevisionID:   row.RevisionID,
                Supplier:     row.Supplier,
                SupplierPN:   row.SupplierPN,
                Type:         row.Type,
                Locations:    []string{},
                CCL:          row.CCL,
            }
        }
        
        groupedMap[key].Locations = append(groupedMap[key].Locations, row.Location)
    }
    
    // 3. 轉換為最終結果
    results := make([]AggregatedPart, 0, len(groupedMap))
    for _, item := range groupedMap {
        sort.Strings(item.Locations)
        item.LocationStr = strings.Join(item.Locations, ",")
        item.Quantity = len(item.Locations)
        results = append(results, *item)
    }
    
    return results, nil
}
```

### 4.3 效能比較

| 操作 | 舊設計 (單表) | 新設計 (JOIN + 記憶體) |
|------|-------------|---------------------|
| **單 BOM 查詢** | ~5ms | ~8ms (JOIN 一次) |
| **8 BOM 聯集** | ~40ms | ~15ms (JOIN + 記憶體聚合) |
| **視圖切換** | ~2ms | ~3ms |
| **儲存空間** | ~24MB (120,000 × 200B) | ~8MB (去重後) |

---

## 5. 查詢方式

### 5.1 常見查詢模式

#### 查詢單一 BOM 的所有物料

```go
func GetBomView(db *gorm.DB, revisionID int64) ([]AggregatedPart, error) {
    var results []AggregatedPart
    
    query := `
        SELECT 
            p.id, p.revision_id, p.supplier, p.supplier_pn, p.type,
            p.description, p.cost, p.remark,
            GROUP_CONCAT(pl.location, ',') AS locations,
            COUNT(pl.id) AS quantity,
            MAX(pl.ccl) AS ccl
        FROM parts p
        LEFT JOIN part_locations pl ON p.id = pl.part_id
        WHERE p.revision_id = ?
        GROUP BY p.id
        ORDER BY p.id
    `
    
    err := db.Raw(query, revisionID).Scan(&results).Error
    return results, err
}
```

#### 依視圖過濾查詢

```go
func GetBomViewByFilter(db *gorm.DB, revisionID int64, viewType string, mode string) ([]AggregatedPart, error) {
    var results []AggregatedPart
    
    // 根據視圖類型建構查詢
    var query string
    
    switch viewType {
    case "SMD", "PTH", "BOTTOM":
        query = `
            SELECT p.*, GROUP_CONCAT(pl.location, ',') AS locations, COUNT(pl.id) AS quantity
            FROM parts p
            JOIN part_locations pl ON p.id = pl.part_id
            WHERE p.revision_id = ? AND p.type = ? AND pl.bom_status IN ?
            GROUP BY p.id
        `
        statuses := []string{"I", "P"}
        if mode == "MP" {
            statuses = []string{"I", "M"}
        }
        err := db.Raw(query, revisionID, viewType, statuses).Scan(&results).Error
        return results, err
        
    case "NI":
        query = `
            SELECT p.*, GROUP_CONCAT(pl.location, ',') AS locations, COUNT(pl.id) AS quantity
            FROM parts p
            JOIN part_locations pl ON p.id = pl.part_id
            WHERE p.revision_id = ? AND pl.bom_status = 'X'
            GROUP BY p.id
        `
        
    case "PROTO":
        query = `
            SELECT p.*, GROUP_CONCAT(pl.location, ',') AS locations, COUNT(pl.id) AS quantity
            FROM parts p
            JOIN part_locations pl ON p.id = pl.part_id
            WHERE p.revision_id = ? AND pl.bom_status = 'P'
            GROUP BY p.id
        `
        
    case "MP":
        query = `
            SELECT p.*, GROUP_CONCAT(pl.location, ',') AS locations, COUNT(pl.id) AS quantity
            FROM parts p
            JOIN part_locations pl ON p.id = pl.part_id
            WHERE p.revision_id = ? AND pl.bom_status = 'M'
            GROUP BY p.id
        `
        
    case "CCL":
        query = `
            SELECT p.*, GROUP_CONCAT(pl.location, ',') AS locations, COUNT(pl.id) AS quantity
            FROM parts p
            JOIN part_locations pl ON p.id = pl.part_id
            WHERE p.revision_id = ? AND pl.ccl = true
            GROUP BY p.id
        `
        
    default: // ALL
        statuses := []string{"I", "P"}
        if mode == "MP" {
            statuses = []string{"I", "M"}
        }
        query = `
            SELECT p.*, GROUP_CONCAT(pl.location, ',') AS locations, COUNT(pl.id) AS quantity
            FROM parts p
            JOIN part_locations pl ON p.id = pl.part_id
            WHERE p.revision_id = ? AND pl.bom_status IN ?
            GROUP BY p.id
        `
        err := db.Raw(query, revisionID, statuses).Scan(&results).Error
        return results, err
    }
    
    err := db.Raw(query, revisionID).Scan(&results).Error
    return results, err
}
```

#### 多 BOM 聯集查詢

```go
func GetMultiBomView(db *gorm.DB, revisionIDs []int64, viewType string, mode string) ([]AggregatedPart, error) {
    var results []AggregatedPart
    
    placeholders := make([]string, len(revisionIDs))
    for i, id := range revisionIDs {
        placeholders[i] = fmt.Sprintf("%d", id)
    }
    inClause := strings.Join(placeholders, ",")
    
    query := fmt.Sprintf(`
        SELECT 
            p.id, p.revision_id, p.supplier, p.supplier_pn, p.type,
            p.description, p.cost, p.remark,
            GROUP_CONCAT(pl.location, ',') AS locations,
            COUNT(pl.id) AS quantity,
            MAX(pl.ccl) AS ccl
        FROM parts p
        JOIN part_locations pl ON p.id = pl.part_id
        WHERE p.revision_id IN (%s)
        GROUP BY p.id
        ORDER BY p.id
    `, inClause)
    
    err := db.Raw(query).Scan(&results).Error
    return results, err
}
```

#### 查詢特定 location 的詳細資訊

```go
func GetPartByLocation(db *gorm.DB, revisionID int64, location string) (*Part, error) {
    var part Part
    
    err := db.Table("parts").
        Joins("JOIN part_locations pl ON parts.id = pl.part_id").
        Where("pl.location = ? AND parts.revision_id = ?", location, revisionID).
        First(&part).Error
    
    if err != nil {
        return nil, err
    }
    
    return &part, nil
}
```

---

## 6. v1 專案架構分析

### 6.1 專案結構

```
bomix-app/
├── main.go                      # 應用程式入口
├── greetservice.go              # Greet 服務 (測試用)
├── bindings/
│   └── bindings.go              # Wails 綁定
├── backend/
│   ├── types/
│   │   ├── domain.go            # 領域型別定義
│   │   ├── errors.go            # 錯誤定義
│   │   └── interfaces.go        # 介面定義
│   ├── config/
│   │   ├── config.go            # 設定載入與儲存
│   │   └── defaults.go          # 預設值定義
│   ├── logger/
│   │   ├── logger.go            # 日誌服務
│   │   └── buffer.go            # 環形緩衝區
│   ├── task/
│   │   ├── task.go              # 任務定義
│   │   ├── manager.go           # 任務管理器
│   │   └── callback.go          # 回呼定義
│   ├── db/
│   │   ├── models.go            # GORM Model 定義
│   │   ├── connection.go        # 資料庫連線管理
│   │   ├── series.go            # Series CRUD
│   │   ├── project.go           # Project CRUD
│   │   ├── revision.go          # BomRevision CRUD
│   │   ├── part.go              # Part CRUD
│   │   └── matrix.go            # Matrix CRUD
│   ├── excel/
│   │   ├── reader.go            # Excel 讀取總入口
│   │   ├── detector.go          # 格式偵測
│   │   ├── reader_ebom.go       # EBOM 讀取
│   │   ├── reader_bigmatrix.go  # BigMatrix 讀取
│   │   ├── reader_matrix.go     # Matrix 讀取 (Placeholder)
│   │   ├── writer.go            # Excel 寫入總入口
│   │   ├── writer_bigmatrix.go  # BigMatrix 寫入
│   │   ├── writer_matrix.go     # Matrix 寫入
│   │   ├── template.go          # Excel 範本載入
│   │   └── workbook*.go         # Excel 抽象層
│   ├── processor/
│   │   ├── aggregator.go        # 資料聚合
│   │   └── filter.go            # 視圖過濾
│   └── view/
│       ├── service.go           # View 服務
│       ├── types.go             # View 型別定義
│       └── filter.go            # View 過濾邏輯
└── frontend/                    # Vue 3 前端
```

### 6.2 核心模組分析

#### 資料庫層 (backend/db/)

| 檔案 | 功能 | 備註 |
|------|------|------|
| `models.go` | GORM Model 定義 | 包含 Series, Project, BomRevision, Part, SecondSource, MatrixModel, MatrixSelection |
| `connection.go` | 資料庫連線管理 | 使用 `github.com/glebarez/sqlite`，支援 WAL 模式、Single Writer |
| `series.go` | Series CRUD | 取得系列元資料 |
| `project.go` | Project CRUD | 取得或建立專案 |
| `revision.go` | BomRevision CRUD | 包含版本搜尋、上一版查詢 |
| `part.go` | Part CRUD | 包含批次寫入、依類型查詢 |
| `matrix.go` | Matrix CRUD | 包含 MatrixSelection 清理、匯入功能 |

#### Excel 匯入層 (backend/excel/)

| 檔案 | 功能 | 備註 |
|------|------|------|
| `reader.go` | 匯入總入口 | 調用對應格式的 Reader |
| `detector.go` | 格式偵測 | 根據 Sheet 結構判斷格式 |
| `reader_ebom.go` | EBOM 讀取 | 完整實作表頭解析、零件解析、Mode 判斷、Merge 演算法 |
| `reader_bigmatrix.go` | BigMatrix 讀取 | 解析橫向多 BOM 與 Model |
| `reader_matrix.go` | Matrix 讀取 | Placeholder，回傳錯誤 |

#### 資料處理層 (backend/processor/)

| 檔案 | 功能 | 備註 |
|------|------|------|
| `aggregator.go` | 資料聚合 | 依 (supplier, supplier_pn) 群組合併 |
| `filter.go` | 視圖過濾 | 實作 8 種視圖過濾規則 |

#### View 層 (backend/view/)

| 檔案 | 功能 | 備註 |
|------|------|------|
| `service.go` | View 服務 | 查詢 BOM 視圖資料 |
| `types.go` | View 型別定義 | ViewQuery, ViewPartGroup, ViewResult |
| `filter.go` | View 過濾邏輯 | 依視圖類型過濾 |

---

## 7. 修改方案計畫

### 7.1 實作階段規劃 (不考慮遷移)

#### 階段一：資料庫 Schema 更新

1. **更新 `parts` 表結構**
   - 僅保留獨立物料基本欄位 (`supplier`, `supplier_pn`, `description`, `cost`, `remark` 等)
   - 移除 `location`, `quantity`, `bom_status`, `ccl` 欄位與索引

2. **新增/更新 `part_locations` 表結構**
   - 包含 `part_id`, `location`, `bom_status`, `ccl` (bool)
   - 建立 `idx_part_location_part`, `idx_part_location_ccl` 及複合索引 `idx_part_loc_status_ccl`

3. **更新 `second_sources` 表結構**
   - 移除 `is_active` 欄位

#### 階段二：匯入邏輯修改

1. **修改 `reader_ebom.go`**
   - 將 Location 原子化為 `[]string`
   - 在儲存時同時為每個 location 建立 `PartLocation` 紀錄 (`ccl` 儲存為 `bool`)

2. **修改 `saveParts` 函數**
   - 批次插入純化後的 `Part` 紀錄
   - 批次插入 `PartLocation` 紀錄 (連結對應的 `part_id`)
   - 批次插入 `SecondSource` 紀錄

#### 階段三：查詢邏輯修改

1. **修改 `part.go`**
   - 新增 `PartLocation` CRUD 函數
   - 修改 `GetPartsByRevision` 等查詢以 JOIN `part_locations`

2. **修改 `aggregator.go`**
   - 從 `part_locations` 聚合 location 字串與計算 quantity
   - 依 `part_locations.bom_status` 與 `part_locations.ccl` 進行狀態判斷

3. **修改 `filter.go` 與 `view/service.go`**
   - 修改過濾邏輯以使用 `part_locations.bom_status` 和 `part_locations.ccl` (bool 條件)

#### 階段四：測試與驗證

1. **單元測試**
   - 測試 Location 原子化
   - 測試視圖過濾 (包含 CCL bool 視圖過濾)
   - 測試聚合邏輯

2. **整合測試**
   - 測試完整匯入流程
   - 測試匯出流程
   - 測試多 BOM 聯集

3. **效能測試**
   - 比較舊設計與新設計的效能
   - 驗證索引效果

### 7.2 檔案修改清單

| 檔案 | 修改類型 | 說明 |
|------|---------|------|
| `backend/db/models.go` | 修改 | 重構 `Part`, `SecondSource` 並新增 `PartLocation` Model |
| `backend/db/part.go` | 修改 | 新增 `PartLocation` CRUD，調整 Part 寫入/查詢 |
| `backend/db/matrix.go` | 修改 | 修改關聯查詢 |
| `backend/excel/reader_ebom.go` | 修改 | 修改 Location 解析與儲存邏輯 |
| `backend/processor/aggregator.go` | 修改 | 修改聚合邏輯 (改用 part_locations) |
| `backend/processor/filter.go` | 修改 | 修改過濾邏輯 (ccl bool 條件) |
| `backend/view/service.go` | 修改 | 修改 View 查詢 (JOIN part_locations) |
| `backend/view/filter.go` | 修改 | 修改 View 過濾 |

### 7.3 時間估計

| 階段 | 預估時間 | 備註 |
|------|---------|------|
| 資料結構更新 | 1-2 小時 | Model 定義與索引建立 |
| 匯入邏輯修改 | 4-6 小時 | 包含測試 |
| 查詢邏輯修改 | 6-8 小時 | 包含所有視圖 |
| 測試與驗證 | 4-6 小時 | 完整測試 |
| **總計** | **15-22 小時** | |

### 7.4 風險評估

| 風險 | 等級 | 緩解措施 |
|------|------|----------|
| 查詢效能下降 | 中 | 建立適當索引，進行效能測試 |
| 現有功能破壞 | 高 | 完整的回歸測試與 View 測試 |
| JOIN 效能問題 | 中 | 使用記憶體聚合優化 |

---

## 結語

本架構設計在 **儲存效率** 和 **查詢效能** 之間取得了最佳平衡：

1. **Location 原子化**：每個 location 獨立儲存，支援精確查詢與過濾
2. **CCL/Status 跟 Location**：確保資料準確性，每個 location 可獨立設定狀態與 CCL (bool)
3. **Part 表純化**：主料表僅保留物料本身屬性，使架構更符合資料庫正規化原則
4. **JOIN + 記憶體聚合**：一次性查詢所有資料，在記憶體中進行聚合，避免 N+1 問題
5. **索引優化**：針對常見查詢模式建立複合索引

此設計可支援：
- 單 BOM 載入 < 10ms
- 8 BOM 聯集 < 20ms
- 視圖切換 < 5ms
- 儲存空間節省 60-70%
