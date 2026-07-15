# BOMIX 資料儲存方案

> 本文檔記錄 BOMIX 專案的資料庫選擇、資料結構設計與實作建議。

## 📋 目錄

- [技術選型](#技術選型)
- [資料結構設計](#資料結構設計)
- [效能優化建議](#效能優化建議)
- [實作範例](#實作範例)

---

## 技術選型

### 推薦組合：modernc.org/sqlite + GORM

| 項目 | 選擇 | 說明 |
|------|------|------|
| **資料庫** | SQLite | 單一 `.bomix` 檔案，無需伺服器 |
| **SQLite 驅動** | modernc.org/sqlite | 純 Go 實現，無需 CGO |
| **ORM 框架** | GORM | 開發效率優先，支援複雜關聯 |

### 為什麼選擇這個組合？

**modernc.org/sqlite 優勢**
- ✅ **純 Go 實現**：無需 CGO，跨平台編譯極簡
- ✅ **單一指令編譯**：`GOOS=windows go build` 即可生成 Windows exe
- ✅ **體積更小**：exe 約 6-8MB（比 mattn 小 20-30%）
- ✅ **啟動快速**：無 C 初始化開銷
- ✅ **Wails 完美相容**：官方推薦用於桌面應用

**GORM 優勢**
- ✅ **開發效率高**：減少 60% 樣板程式碼
- ✅ **複雜關聯支援**：Preload 自動解決 N+1 問題
- ✅ **型別安全**：編譯期檢查 Struct 欄位
- ✅ **生態成熟**：Go 最流行的 ORM 框架

### 跨平台編譯（無需 CGO）

```bash
# Linux → Windows
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o bomix.exe

# Linux → macOS
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o bomix

# 一鍵編譯所有平台
make all
```

### go.mod 建議版本

```go
require (
    gorm.io/gorm v1.25.9
    gorm.io/driver/sqlite v1.5.4  // 已封裝 modernc
    modernc.org/sqlite v1.38.0
)
```

---

## 資料結構設計

### 核心資料模型

```go
package model

import "time"

// Series 系列（.bomix 檔案）
type Series struct {
    ID          string    `gorm:"primaryKey"`
    Name        string    `gorm:"uniqueIndex"`
    Description string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// Project 專案
type Project struct {
    ID          string    `gorm:"primaryKey"`
    SeriesID    string    `gorm:"index:idx_project_series"`
    Code        string    // TANGLED
    Description string
    CreatedAt   time.Time
}

// BomRevision BOM 版本（Phase + Version）
type BomRevision struct {
    ID            string    `gorm:"primaryKey"`
    ProjectID     string    `gorm:"index:idx_revision_project"`
    PhaseName     string    // DB, EVT, DVT...
    Version       string    // 0.1, 1.0
    SchematicVer  string
    PcbVer        string
    PcaPn         string
    Date          string
    Metadata      string    // JSON 儲存額外屬性
    CreatedAt     time.Time
}

// MatrixModel Matrix 模型定義
type MatrixModel struct {
    ID         string    `gorm:"primaryKey"`
    RevisionID string    `gorm:"index:idx_matrix_revision"`
    Name       string    // A, B, C
    IsDefault  bool
}

// Part 零件（Main + Second Source 統一表）
type Part struct {
    ID           string    `gorm:"primaryKey"`
    RevisionID   string    `gorm:"index:idx_part_revision"`
    Item         int       // Excel 行號
    
    // 基本屬性
    Hhpn         string    `gorm:"index:idx_part_hhpn"`
    Supplier     string
    SupplierPn   string    `gorm:"index:idx_part_supplier"`
    Description  string
    Location     string    // "R1,C2,C3" 逗號分隔（非原子化）
    Type         string    // SMD, PTH, BOTTOM
    BomStatus    string    // I, X, P, M
    Ccl          string    // Y, N
    Remark       string
    
    // 關聯屬性
    IsMainSource bool      `gorm:"default:true"`
    MainKey      string    `gorm:"index:idx_part_mainkey"` // "Supplier|SupplierPn" 組合鍵
    
    // Second Source 關係（使用 JSON 欄位儲存，避免 JOIN 開銷）
    SecondSources string    `gorm:"type:text"` // JSON 陣列
    
    // Matrix 選擇快取（避免每次 JOIN）
    MatrixSelections string   `gorm:"type:text"` // JSON: {"A":"part_id_1", "B":"part_id_2"}
    
    CreatedAt    time.Time
}

// MatrixSelection Matrix 選擇紀錄（完整歷史）
type MatrixSelection struct {
    ID             string    `gorm:"primaryKey"`
    RevisionID     string
    ModelID        string    `gorm:"index:idx_matrix_sel_model"`
    PartGroupID    string    // Main Source 的 ID
    SelectedPartID string
    IsAutoSelected bool
    UpdatedAt      time.Time
    
    Unique: []string{"model_id", "part_group_id"}
}
```

---

## 效能優化建議

### 1. 索引策略

```sql
-- 核心索引
CREATE INDEX idx_parts_revision ON parts(revision_id);
CREATE INDEX idx_parts_hhpn ON parts(hhpn);
CREATE INDEX idx_parts_supplier ON parts(supplier, supplier_pn);
CREATE INDEX idx_parts_mainkey ON parts(main_key);
CREATE INDEX idx_matrix_selection ON matrix_selections(model_id, part_group_id);
CREATE INDEX idx_bom_revision_project ON bom_revisions(project_id, phase_name, version);
```

### 2. 查詢優化

```go
// 使用 preload 避免 N+1 問題
db.Preload("SecondSources").Preload("MatrixSelections").
   Where("revision_id = ?", revisionID).
   Find(&parts)

// 或使用 joins 一次性查詢
db.Table("parts p").
   Joins("LEFT JOIN matrix_selections ms ON ms.part_group_id = p.id").
   Where("p.revision_id = ?", revisionID).
   Select("p.*, ms.model_id, ms.selected_part_id").
   Scan(&result)
```

### 3. 批次操作

```go
// Excel 匯入時使用批次插入
db.CreateInBatches(&parts, 100) // 每 100 筆一筆交易
```

### 4. 快取策略

```go
// 使用 singleflight 避免重複查詢
var dbCache = &sync.Map{}

func GetRevisionParts(revisionID string) ([]Part, error) {
    if cached, ok := dbCache.Load(revisionID); ok {
        return cached.([]Part), nil
    }
    // 查詢資料庫並快取
}
```

---

## 實作範例

### 資料庫初始化

```go
package database

import (
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"
    "path/filepath"
    "os"
)

func NewBomixDB(seriesPath string) (*gorm.DB, error) {
    // 確保目錄存在
    dir := filepath.Dir(seriesPath)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return nil, err
    }

    // 開啟 SQLite（共用快取模式，提升效能）
    db, err := gorm.Open(sqlite.Open(
        seriesPath + "?_busy_timeout=5000&_journal_mode=WAL&_cache_size=-64000",
    ), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Silent),
    })
    if err != nil {
        return nil, err
    }

    // 效能優化設定
    sqlDB, _ := db.DB()
    sqlDB.SetMaxOpenConns(1)  // SQLite 單一寫入
    sqlDB.SetMaxIdleConns(1)

    // 自動遷移
    return db, db.AutoMigrate(
        &model.Series{},
        &model.Project{},
        &model.BomRevision{},
        &model.MatrixModel{},
        &model.Part{},
        &model.MatrixSelection{},
    )
}
```

### 獲取完整 BOM（含 Second Sources + Matrix）

```go
func GetBomRevisionWithDetails(db *gorm.DB, revisionID string) (*model.BomRevision, error) {
    var revision model.BomRevision
    
    // 一次性查詢所有資料（避免 N+1）
    result := db.Preload("MatrixModels").
        Joins("LEFT JOIN parts p ON p.revision_id = ? AND p.is_main_source = true", revisionID).
        Joins("LEFT JOIN matrix_selections ms ON ms.revision_id = ? AND ms.part_group_id = p.id", revisionID).
        Where("revision.id = ?", revisionID).
        First(&revision)
    
    return &revision, result.Error
}
```

### Matrix 選擇（即時更新）

```go
func UpdateMatrixSelection(db *gorm.DB, modelID, partGroupID, selectedPartID string, isAuto bool) error {
    return db.Transaction(func(tx *gorm.DB) error {
        // 使用 INSERT OR REPLACE 確保唯一性
        result := tx.Exec(`
            INSERT OR REPLACE INTO matrix_selections 
            (model_id, part_group_id, selected_part_id, is_auto_selected, updated_at)
            VALUES (?, ?, ?, ?, ?)
        `, modelID, partGroupID, selectedPartID, isAuto, time.Now())
        
        return result.Error
    })
}
```

### 查詢 BOM 完成度（燈號判斷）

```go
func GetMatrixCompletionStatus(db *gorm.DB, revisionID string) (map[string]float32, error) {
    var results []struct {
        ModelID    string
        Total      int
        Completed  int
    }
    
    err := db.Raw(`
        SELECT 
            mm.id as model_id,
            COUNT(DISTINCT p.id) as total,
            COUNT(DISTINCT ms.part_group_id) as completed
        FROM matrix_models mm
        CROSS JOIN parts p 
            ON p.revision_id = ? AND p.is_main_source = true
        LEFT JOIN matrix_selections ms 
            ON ms.model_id = mm.id AND ms.part_group_id = p.id
        WHERE mm.revision_id = ?
        GROUP BY mm.id
    `, revisionID, revisionID).Scan(&results).Error
    
    if err != nil {
        return nil, err
    }
    
    status := make(map[string]float32)
    for _, r := range results {
        if r.Total > 0 {
            status[r.ModelID] = float32(r.Completed) / float32(r.Total)
        }
    }
    return status, nil
}
```

---

## 總結

| 項目 | 推薦方案 |
|------|---------|
| **資料庫** | SQLite（單一 `.bomix` 檔案） |
| **驅動** | modernc.org/sqlite（純 Go，無需 CGO） |
| **ORM** | GORM（開發效率優先） |
| **Location 儲存** | 逗號分隔字串（簡化查詢） |
| **Second Source** | JSON 欄位或主鍵關聯 |
| **Matrix 選擇** | 獨立資料表 + 快取欄位 |
| **索引策略** | `revision_id`, `hhpn`, `main_key` |

這個方案在**開發效率、效能、維護性**之間取得最佳平衡，適合 BOMIX 的桌面應用程式需求。

---

*最後更新：2026-01-15*
