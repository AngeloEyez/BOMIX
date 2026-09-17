package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"bomix-app/backend/db"
	"bomix-app/backend/logger"
	"bomix-app/backend/view"

	"gorm.io/gorm"
)

// GetBuiltinTools 回傳供 LLM 呼叫的四個雙軌混合工具規格定義
func GetBuiltinTools() []ToolDef {
	return []ToolDef{
		{
			Type: "function",
			Function: FunctionDef{
				Name:        "get_database_schema",
				Description: "Retrieve the complete database schema (table definitions, columns, foreign keys) of the currently opened BOM series SQLite database, along with domain terminology mappings. The LLM must call this tool before generating SQL queries to inspect the schema.",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		},
		{
			Type: "function",
			Function: FunctionDef{
				Name:        "execute_readonly_sql",
				Description: "Execute a read-only SELECT SQL query on the active series SQLite database. Only SELECT and WITH statements are allowed. Any data modification statements are strictly forbidden. Maximum 100 rows returned. Ideal for aggregations, cross-project lookups, and arbitrary filters.",
				Parameters: map[string]interface{}{
					"type": "object",
					"required": []string{"sql"},
					"properties": map[string]interface{}{
						"sql": map[string]interface{}{
							"type":        "string",
							"description": "Read-only SELECT query (supports JOINs, GROUP BY, ORDER BY, subqueries, etc.)",
						},
						"explanation": map[string]interface{}{
							"type":        "string",
							"description": "Brief explanation of what this query aims to find",
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: FunctionDef{
				Name:        "get_series_overview",
				Description: "Get an overview of projects and their BOM revisions (Phase, Version, dates, mode, models count) in the current series. Used to verify project names, check available revisions, and obtain revision IDs for detailed queries.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"project_code": map[string]interface{}{
							"type":        "string",
							"description": "Optional project code to filter by. If omitted, returns all projects in the series.",
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: FunctionDef{
				Name:        "compare_revisions_diff",
				Description: "Compare differences between two BOM revisions (within the same project or across projects). Uses a native Go diff algorithm to categorize added, removed, and modified parts (quantity changes, location changes, 2nd source alterations).",
				Parameters: map[string]interface{}{
					"type": "object",
					"required": []string{"base_revision_id", "target_revision_id"},
					"properties": map[string]interface{}{
						"base_revision_id": map[string]interface{}{
							"type":        "integer",
							"description": "Revision ID of the baseline (older) revision",
						},
						"target_revision_id": map[string]interface{}{
							"type":        "integer",
							"description": "Revision ID of the target (newer) revision",
						},
						"change_type_filter": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"ALL", "ADDED", "REMOVED", "MODIFIED"},
							"description": "Optional filter by change type. Defaults to ALL.",
						},
						"limit": map[string]interface{}{
							"type":        "integer",
							"description": "Optional max number of detail items returned (default 50)",
						},
					},
				},
			},
		},
	}
}

// ToolExecutor 工具執行器，持有當前資料庫連線參照
type ToolExecutor struct {
	db     *gorm.DB
	logger *logger.Logger
}

// NewToolExecutor 建立工具執行器實例
func NewToolExecutor(database *gorm.DB, lg *logger.Logger) *ToolExecutor {
	return &ToolExecutor{
		db:     database,
		logger: lg,
	}
}

// Execute 依據工具名稱分派並執行具體邏輯
//
// 參數:
//   - ctx: 呼叫上下文
//   - name: 工具名稱
//   - argsJSON: LLM 傳入的 JSON 字串參數
//
// 回傳:
//   - string: 工具執行的結構化 JSON 結果字串
//   - error: 執行失敗時回傳錯誤
func (te *ToolExecutor) Execute(ctx context.Context, name string, argsJSON string) (string, error) {
	if te.db == nil {
		return "", errors.New("尚未開啟系列資料庫，無法執行資料查詢工具")
	}

	switch name {
	case "get_database_schema":
		return te.handleGetDatabaseSchema(ctx)
	case "execute_readonly_sql":
		return te.handleExecuteReadonlySQL(ctx, argsJSON)
	case "get_series_overview":
		return te.handleGetSeriesOverview(ctx, argsJSON)
	case "compare_revisions_diff":
		return te.handleCompareRevisionsDiff(ctx, argsJSON)
	default:
		return "", fmt.Errorf("未知的工具名稱: %s", name)
	}
}

// ==================== 工具處理常式 ====================

// handleGetDatabaseSchema 取得資料庫結構與術語對照
func (te *ToolExecutor) handleGetDatabaseSchema(ctx context.Context) (string, error) {
	var tableNames []string
	err := te.db.WithContext(ctx).Raw("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name NOT LIKE 'gorm_%' ORDER BY name;").Scan(&tableNames).Error
	if err != nil {
		return "", fmt.Errorf("查詢資料庫資料表清單失敗: %w", err)
	}

	type ColumnInfo struct {
		CID       int         `gorm:"column:cid" json:"cid"`
		Name      string      `gorm:"column:name" json:"name"`
		Type      string      `gorm:"column:type" json:"type"`
		NotNull   int         `gorm:"column:notnull" json:"notnull"`
		DfltValue interface{} `gorm:"column:dflt_value" json:"dflt_value"`
		PK        int         `gorm:"column:pk" json:"pk"`
	}

	type ForeignKeyInfo struct {
		ID       int    `gorm:"column:id" json:"id"`
		Seq      int    `gorm:"column:seq" json:"seq"`
		Table    string `gorm:"column:table" json:"table"`
		From     string `gorm:"column:from" json:"from"`
		To       string `gorm:"column:to" json:"to"`
		OnUpdate string `gorm:"column:on_update" json:"on_update"`
		OnDelete string `gorm:"column:on_delete" json:"on_delete"`
	}

	type TableSchema struct {
		Columns     []ColumnInfo     `json:"columns"`
		ForeignKeys []ForeignKeyInfo `json:"foreign_keys"`
	}

	schemaMap := make(map[string]TableSchema, len(tableNames))
	for _, tbl := range tableNames {
		var cols []ColumnInfo
		_ = te.db.WithContext(ctx).Raw(fmt.Sprintf("PRAGMA table_info(%s);", tbl)).Scan(&cols).Error

		var fks []ForeignKeyInfo
		_ = te.db.WithContext(ctx).Raw(fmt.Sprintf("PRAGMA foreign_key_list(%s);", tbl)).Scan(&fks).Error

		schemaMap[tbl] = TableSchema{
			Columns:     cols,
			ForeignKeys: fks,
		}
	}

	// 領域術語與核心欄位說明
	glossary := map[string]string{
		"Phase":            "BOM development stage (e.g. DB=Design Build, EVT, DVT, PVT, SI, MP)",
		"Version":          "Sub-version under Phase (e.g. 0.1, 0.2, 1.0)",
		"Location/RefDes":  "Physical circuit designator on PCB (e.g. R1, C5, U3, LR1)",
		"Supplier":         "Component manufacturer / brand name",
		"SupplierPN":       "Manufacturer part number",
		"HHPN":             "Company internal part number",
		"Role":             "M=Main source, S=Second source",
		"BomStatus":        "I=Install, X=Not Install, P=Proto, M=MP",
		"Type":             "SMD (surface mount), PTH (through hole), BOTTOM",
		"CCL":              "Critical Component List flag (boolean)",
		"MatrixModel":      "Board configuration / variant model name",
		"MatrixSelection":  "Selected part (main vs second source) for a specific MatrixModel",
		"Key Relationship": "projects.series_id -> series.id; bom_revisions.project_id -> projects.id; revision_components.revision_id -> bom_revisions.id; part_locations.component_id -> revision_components.id; materials.id shared across revisions",
	}

	res := map[string]interface{}{
		"tables":   schemaMap,
		"glossary": glossary,
	}

	bytes, err := json.Marshal(res)
	if err != nil {
		return "", fmt.Errorf("序列化 Schema 失敗: %w", err)
	}
	return string(bytes), nil
}

// handleExecuteReadonlySQL 執行唯讀 SQL
func (te *ToolExecutor) handleExecuteReadonlySQL(ctx context.Context, argsJSON string) (string, error) {
	var params struct {
		SQL         string `json:"sql"`
		Explanation string `json:"explanation"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &params); err != nil {
		return "", fmt.Errorf("解析 SQL 工具參數失敗: %w", err)
	}

	if strings.TrimSpace(params.SQL) == "" {
		return "", errors.New("未提供欲執行的 SQL 語句")
	}

	res, err := ExecuteSafeQuery(te.db, params.SQL)
	if err != nil {
		return "", err
	}

	bytes, err := json.Marshal(res)
	if err != nil {
		return "", fmt.Errorf("序列化查詢結果失敗: %w", err)
	}
	return string(bytes), nil
}

// handleGetSeriesOverview 取得系列專案與版本總覽
func (te *ToolExecutor) handleGetSeriesOverview(ctx context.Context, argsJSON string) (string, error) {
	var params struct {
		ProjectCode string `json:"project_code"`
	}
	_ = json.Unmarshal([]byte(argsJSON), &params)

	var series db.Series
	err := te.db.WithContext(ctx).Preload("Projects.Revisions.MatrixModels").First(&series).Error
	if err != nil {
		return "", fmt.Errorf("查詢系列資訊失敗: %w", err)
	}

	type RevisionSummary struct {
		ID          int64  `json:"id"`
		Phase       string `json:"phase"`
		Version     string `json:"version"`
		Date        string `json:"date"`
		Mode        string `json:"mode"`
		SourceFile  string `json:"source_file"`
		ModelsCount int    `json:"models_count"`
		ModelNames  []string `json:"model_names"`
	}

	type ProjectSummary struct {
		ID          int64             `json:"id"`
		Code        string            `json:"code"`
		Description string            `json:"description"`
		Revisions   []RevisionSummary `json:"revisions"`
	}

	var projects []ProjectSummary
	for _, p := range series.Projects {
		if params.ProjectCode != "" && !strings.EqualFold(p.Code, params.ProjectCode) {
			continue
		}

		var revSummaries []RevisionSummary
		for _, r := range p.Revisions {
			var modelNames []string
			for _, m := range r.MatrixModels {
				modelNames = append(modelNames, m.ModelName)
			}
			revSummaries = append(revSummaries, RevisionSummary{
				ID:          r.ID,
				Phase:       r.Phase,
				Version:     r.Version,
				Date:        r.Date,
				Mode:        r.Mode,
				SourceFile:  r.SourceFile,
				ModelsCount: len(r.MatrixModels),
				ModelNames:  modelNames,
			})
		}

		projects = append(projects, ProjectSummary{
			ID:          p.ID,
			Code:        p.Code,
			Description: p.Description,
			Revisions:   revSummaries,
		})
	}

	res := map[string]interface{}{
		"series_id":   series.ID,
		"series_name": series.Name,
		"projects":    projects,
	}

	bytes, err := json.Marshal(res)
	if err != nil {
		return "", fmt.Errorf("序列化系列總覽失敗: %w", err)
	}
	return string(bytes), nil
}

// handleCompareRevisionsDiff 比較兩個 BOM 版本的零件差異
func (te *ToolExecutor) handleCompareRevisionsDiff(ctx context.Context, argsJSON string) (string, error) {
	var params struct {
		BaseRevisionID   int64  `json:"base_revision_id"`
		TargetRevisionID int64  `json:"target_revision_id"`
		ChangeTypeFilter string `json:"change_type_filter"`
		Limit            int    `json:"limit"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &params); err != nil {
		return "", fmt.Errorf("解析 Diff 工具參數失敗: %w", err)
	}

	if params.BaseRevisionID <= 0 || params.TargetRevisionID <= 0 {
		return "", errors.New("base_revision_id 與 target_revision_id 均必須為大於 0 的有效 ID")
	}

	if params.Limit <= 0 {
		params.Limit = 50
	}
	filter := strings.ToUpper(strings.TrimSpace(params.ChangeTypeFilter))
	if filter == "" {
		filter = "ALL"
	}

	viewSvc := view.NewService(te.db, te.logger)

	// 查詢基準版本
	baseRes, err := viewSvc.Query(view.ViewQuery{
		RevisionIDs: []int64{params.BaseRevisionID},
		ViewType:    view.ViewAll,
	})
	if err != nil {
		return "", fmt.Errorf("查詢基準版本 (ID: %d) 失敗: %w", params.BaseRevisionID, err)
	}

	// 查詢比對目標版本
	targetRes, err := viewSvc.Query(view.ViewQuery{
		RevisionIDs: []int64{params.TargetRevisionID},
		ViewType:    view.ViewAll,
	})
	if err != nil {
		return "", fmt.Errorf("查詢比對版本 (ID: %d) 失敗: %w", params.TargetRevisionID, err)
	}

	// 建立基準物料 Map (Key: Supplier|SupplierPN)
	baseMap := make(map[string]view.ViewPartGroup, len(baseRes.PartGroups))
	for _, p := range baseRes.PartGroups {
		key := strings.ToLower(p.MainSupplier) + "||" + strings.ToLower(p.MainSupplierPN)
		baseMap[key] = p
	}

	// 建立目標物料 Map
	targetMap := make(map[string]view.ViewPartGroup, len(targetRes.PartGroups))
	for _, p := range targetRes.PartGroups {
		key := strings.ToLower(p.MainSupplier) + "||" + strings.ToLower(p.MainSupplierPN)
		targetMap[key] = p
	}

	type DiffDetail struct {
		ChangeType  string `json:"change_type"` // "ADDED" | "REMOVED" | "MODIFIED"
		Supplier    string `json:"supplier"`
		SupplierPN  string `json:"supplier_pn"`
		HHPN        string `json:"hhpn"`
		Description string `json:"description"`
		BaseQty     int    `json:"base_qty"`
		TargetQty   int    `json:"target_qty"`
		QtyDelta    int    `json:"qty_delta"`
		BaseLocs    string `json:"base_locations,omitempty"`
		TargetLocs  string `json:"target_locations,omitempty"`
		Details     string `json:"details,omitempty"`
	}

	var added []DiffDetail
	var removed []DiffDetail
	var modified []DiffDetail
	unchangedCount := 0

	// 1. 遍歷 targetMap 找出 ADDED 與 MODIFIED
	for key, targetPart := range targetMap {
		basePart, exists := baseMap[key]
		if !exists {
			added = append(added, DiffDetail{
				ChangeType:  "ADDED",
				Supplier:    targetPart.MainSupplier,
				SupplierPN:  targetPart.MainSupplierPN,
				HHPN:        targetPart.HHPN,
				Description: targetPart.Description,
				BaseQty:     0,
				TargetQty:   targetPart.Qty,
				QtyDelta:    targetPart.Qty,
				TargetLocs:  targetPart.Locations,
				Details:     fmt.Sprintf("新版本新增物料，用量: %d", targetPart.Qty),
			})
			continue
		}

		// 檢查是否有變更
		isModified := false
		var modReasons []string

		if basePart.Qty != targetPart.Qty {
			isModified = true
			modReasons = append(modReasons, fmt.Sprintf("用量從 %d 變為 %d (Δ %d)", basePart.Qty, targetPart.Qty, targetPart.Qty-basePart.Qty))
		}

		if strings.TrimSpace(basePart.Locations) != strings.TrimSpace(targetPart.Locations) {
			isModified = true
			modReasons = append(modReasons, "位置分配 (Locations) 有變動")
		}

		if basePart.BOMStatus != targetPart.BOMStatus {
			isModified = true
			modReasons = append(modReasons, fmt.Sprintf("BOM 狀態從 %s 變更為 %s", basePart.BOMStatus, targetPart.BOMStatus))
		}

		if len(basePart.SecondSources) != len(targetPart.SecondSources) {
			isModified = true
			modReasons = append(modReasons, fmt.Sprintf("替代料數量從 %d 變為 %d", len(basePart.SecondSources), len(targetPart.SecondSources)))
		}

		if isModified {
			modified = append(modified, DiffDetail{
				ChangeType:  "MODIFIED",
				Supplier:    targetPart.MainSupplier,
				SupplierPN:  targetPart.MainSupplierPN,
				HHPN:        targetPart.HHPN,
				Description: targetPart.Description,
				BaseQty:     basePart.Qty,
				TargetQty:   targetPart.Qty,
				QtyDelta:    targetPart.Qty - basePart.Qty,
				BaseLocs:    basePart.Locations,
				TargetLocs:  targetPart.Locations,
				Details:     strings.Join(modReasons, "; "),
			})
		} else {
			unchangedCount++
		}
	}

	// 2. 遍歷 baseMap 找出 REMOVED
	for key, basePart := range baseMap {
		if _, exists := targetMap[key]; !exists {
			removed = append(removed, DiffDetail{
				ChangeType:  "REMOVED",
				Supplier:    basePart.MainSupplier,
				SupplierPN:  basePart.MainSupplierPN,
				HHPN:        basePart.HHPN,
				Description: basePart.Description,
				BaseQty:     basePart.Qty,
				TargetQty:   0,
				QtyDelta:    -basePart.Qty,
				BaseLocs:    basePart.Locations,
				Details:     fmt.Sprintf("新版本已移除此物料 (原用量: %d)", basePart.Qty),
			})
		}
	}

	// 排序以保持結果穩定
	sort.Slice(added, func(i, j int) bool { return added[i].SupplierPN < added[j].SupplierPN })
	sort.Slice(removed, func(i, j int) bool { return removed[i].SupplierPN < removed[j].SupplierPN })
	sort.Slice(modified, func(i, j int) bool { return modified[i].SupplierPN < modified[j].SupplierPN })

	// 組合符合篩選條件的明細列表
	var filteredDetails []DiffDetail
	switch filter {
	case "ADDED":
		filteredDetails = added
	case "REMOVED":
		filteredDetails = removed
	case "MODIFIED":
		filteredDetails = modified
	default: // "ALL"
		filteredDetails = append(filteredDetails, added...)
		filteredDetails = append(filteredDetails, removed...)
		filteredDetails = append(filteredDetails, modified...)
	}

	totalFiltered := len(filteredDetails)
	truncated := false
	if len(filteredDetails) > params.Limit {
		filteredDetails = filteredDetails[:params.Limit]
		truncated = true
	}

	var baseInfo, targetInfo string
	if len(baseRes.Revisions) > 0 {
		r := baseRes.Revisions[0]
		baseInfo = fmt.Sprintf("%s %s %s", r.ProjectCode, r.Phase, r.Version)
	}
	if len(targetRes.Revisions) > 0 {
		r := targetRes.Revisions[0]
		targetInfo = fmt.Sprintf("%s %s %s", r.ProjectCode, r.Phase, r.Version)
	}

	diffResult := map[string]interface{}{
		"summary": map[string]interface{}{
			"base_revision":      baseInfo,
			"target_revision":    targetInfo,
			"added_count":        len(added),
			"removed_count":      len(removed),
			"modified_count":     len(modified),
			"unchanged_count":    unchangedCount,
			"total_differences":  len(added) + len(removed) + len(modified),
			"filter_applied":     filter,
			"total_matching":     totalFiltered,
			"returned_count":     len(filteredDetails),
			"truncated":          truncated,
		},
		"details": filteredDetails,
	}

	bytes, err := json.Marshal(diffResult)
	if err != nil {
		return "", fmt.Errorf("序列化 Diff 結果失敗: %w", err)
	}
	return string(bytes), nil
}
