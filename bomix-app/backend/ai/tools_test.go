package ai

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"bomix-app/backend/db"
	"bomix-app/backend/logger"
)

// TestHandleGetDatabaseSchema_NoDfltValueParseError 驗證 PRAGMA table_info 查詢能正確解析包含預設值的欄位，不會產生 DfltValue unsupported data type 錯誤
func TestHandleGetDatabaseSchema_NoDfltValueParseError(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_schema.bomx")

	testDB, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("開啟資料庫失敗: %v", err)
	}
	defer db.Close(testDB)

	if err := db.AutoMigrate(testDB); err != nil {
		t.Fatalf("AutoMigrate 失敗: %v", err)
	}

	log := logger.NewLogger(100)
	executor := NewToolExecutor(testDB, log)

	ctx := context.Background()
	schemaJSON, err := executor.Execute(ctx, "get_database_schema", "{}")
	if err != nil {
		t.Fatalf("Execute get_database_schema 失敗: %v", err)
	}

	var res struct {
		Tables   map[string]interface{} `json:"tables"`
		Glossary map[string]string      `json:"glossary"`
	}

	if err := json.Unmarshal([]byte(schemaJSON), &res); err != nil {
		t.Fatalf("解析 schemaJSON 失敗: %v", err)
	}

	if len(res.Tables) == 0 {
		t.Fatal("預期回傳資料表 Schema，但 tables 為空")
	}

	// 驗證 projects 資料表是否有 columns
	projTable, ok := res.Tables["projects"].(map[string]interface{})
	if !ok {
		t.Fatal("Schema 中未找到 projects 資料表")
	}

	cols, ok := projTable["columns"].([]interface{})
	if !ok || len(cols) == 0 {
		t.Fatal("projects 資料表的 columns 不應為空")
	}
}
