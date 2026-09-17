package ai

import (
	"testing"
)

func TestValidateReadOnlySQL(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		wantErr bool
	}{
		{
			name:    "合法的簡單 SELECT",
			sql:     "SELECT * FROM series;",
			wantErr: false,
		},
		{
			name:    "合法的多行 SELECT 帶 JOIN 與 GROUP BY",
			sql:     "SELECT p.code, count(*) FROM projects p JOIN bom_revisions r ON r.project_id = p.id GROUP BY p.code;",
			wantErr: false,
		},
		{
			name:    "合法的 WITH 遞迴或 CTE 語句",
			sql:     "WITH latest_rev AS (SELECT id, project_id FROM bom_revisions ORDER BY id DESC) SELECT * FROM latest_rev;",
			wantErr: false,
		},
		{
			name:    "合法的包含註解的查詢",
			sql:     "-- 這是測試註解\nSELECT /* 內嵌註解 */ id, name FROM series WHERE 1=1;",
			wantErr: false,
		},
		{
			name:    "拒絕空字串",
			sql:     "   ",
			wantErr: true,
		},
		{
			name:    "拒絕 INSERT 寫入語句",
			sql:     "INSERT INTO series (name) VALUES ('Hacked');",
			wantErr: true,
		},
		{
			name:    "拒絕 UPDATE 修改語句",
			sql:     "UPDATE series SET name = 'Foo';",
			wantErr: true,
		},
		{
			name:    "拒絕 DELETE 刪除語句",
			sql:     "DELETE FROM bom_revisions WHERE id = 1;",
			wantErr: true,
		},
		{
			name:    "拒絕 DROP 刪除表語句",
			sql:     "DROP TABLE projects;",
			wantErr: true,
		},
		{
			name:    "拒絕 ALTER 結構修改語句",
			sql:     "ALTER TABLE series ADD COLUMN secret TEXT;",
			wantErr: true,
		},
		{
			name:    "拒絕分號串接雙重語句攻擊",
			sql:     "SELECT * FROM series; DROP TABLE projects;",
			wantErr: true,
		},
		{
			name:    "拒絕 PRAGMA 寫入或查詢",
			sql:     "PRAGMA writable_schema = 1;",
			wantErr: true,
		},
		{
			name:    "拒絕 ATTACH 資料庫",
			sql:     "ATTACH DATABASE 'evil.db' AS evil;",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateReadOnlySQL(tc.sql)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateReadOnlySQL(%q) error = %v, wantErr %v", tc.sql, err, tc.wantErr)
			}
		})
	}
}

func TestStripSQLComments(t *testing.T) {
	input := "-- Header comment\nSELECT * FROM test; /* block comment */"
	cleaned := StripSQLComments(input)
	if cleaned != "SELECT * FROM test;" {
		t.Errorf("StripSQLComments() = %q, want %q", cleaned, "SELECT * FROM test;")
	}
}
