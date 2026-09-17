package ai

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
)

// 常見危險關鍵字正規表達式（不分大小寫、全詞匹配）
var dangerousKeywordsRegex = regexp.MustCompile(`(?i)\b(INSERT|UPDATE|DELETE|DROP|ALTER|CREATE|ATTACH|DETACH|REPLACE|TRUNCATE|TRANSACTION|COMMIT|ROLLBACK|VACUUM|REINDEX|PRAGMA|EXEC|EXECUTE)\b`)

// SQL 註解清理正規表達式
var (
	singleLineCommentRegex = regexp.MustCompile(`--.*$`)
	multiLineCommentRegex  = regexp.MustCompile(`/\*[\s\S]*?\*/`)
)

// QueryResult 安全查詢回傳結構
type QueryResult struct {
	Columns   []string                 `json:"columns"`
	Rows      []map[string]interface{} `json:"rows"`
	RowCount  int                      `json:"row_count"`
	Truncated bool                     `json:"truncated"`
	Message   string                   `json:"message,omitempty"`
}

// StripSQLComments 移除 SQL 中的單行與多行註解
func StripSQLComments(sql string) string {
	sql = multiLineCommentRegex.ReplaceAllString(sql, " ")
	lines := strings.Split(sql, "\n")
	for i, line := range lines {
		lines[i] = singleLineCommentRegex.ReplaceAllString(line, "")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// ValidateReadOnlySQL 驗證 SQL 語句是否為嚴格唯讀
//
// 驗證機制包含：
// 1. 移除註解後檢查非空
// 2. 僅允許單一 SQL 語句（禁止分號連接多語句執行攻擊）
// 3. 必須以 SELECT 或 WITH 開頭
// 4. 禁止任何寫入、結構變更、交易控制等危險關鍵字
//
// 參數:
//   - sql: 待驗證的 SQL 原始字串
//
// 回傳:
//   - 驗證成功回傳 nil，失敗則回傳具體錯誤原因
func ValidateReadOnlySQL(sql string) error {
	cleaned := StripSQLComments(sql)
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return errors.New("SQL 語句不可為空")
	}

	// 檢查分號：允許結尾有一個分號，但禁止多語句
	semicolons := strings.Split(cleaned, ";")
	nonEmptyCount := 0
	for _, part := range semicolons {
		if strings.TrimSpace(part) != "" {
			nonEmptyCount++
		}
	}
	if nonEmptyCount > 1 {
		return errors.New("僅允許執行單一 SQL 查詢，禁止使用分號串接多條語句")
	}

	// 檢查開頭必須為 SELECT 或 WITH
	upper := strings.ToUpper(cleaned)
	trimmedUpper := strings.TrimSpace(upper)
	if !strings.HasPrefix(trimmedUpper, "SELECT") && !strings.HasPrefix(trimmedUpper, "WITH") {
		return fmt.Errorf("僅允許 SELECT 或 WITH 查詢語句，拒絕執行: %s", strings.Fields(trimmedUpper)[0])
	}

	// 檢查危險關鍵字黑名單
	if match := dangerousKeywordsRegex.FindString(cleaned); match != "" {
		return fmt.Errorf("SQL 語句包含禁止的危險關鍵字: %s", strings.ToUpper(match))
	}

	return nil
}

// ExecuteSafeQuery 在唯讀保護下執行 SQL 並回傳結構化結果
//
// 執行保護措施：
// 1. 透過 ValidateReadOnlySQL 進行語法安全檢查
// 2. 自動補充 LIMIT 100（若未指定或超過上限）
// 3. 設定 10 秒執行逾時避免長查詢卡死
// 4. 動態掃描欄位名稱與資料列轉為 JSON 友善結構
//
// 參數:
//   - db: GORM 資料庫連線實例
//   - sql: 欲執行的唯讀 SQL 語句
//
// 回傳:
//   - *QueryResult: 結構化查詢結果
//   - error: 執行失敗或被安全規則攔截時回傳錯誤
func ExecuteSafeQuery(db *gorm.DB, sql string) (*QueryResult, error) {
	if db == nil {
		return nil, errors.New("資料庫連線尚未建立，請先開啟系列資料庫")
	}

	// 1. 語法安全驗證
	if err := ValidateReadOnlySQL(sql); err != nil {
		return nil, fmt.Errorf("SQL 安全驗證失敗: %w", err)
	}

	// 2. 處理 LIMIT（若無 LIMIT 則補上 LIMIT 100）
	cleaned := strings.TrimRight(strings.TrimSpace(sql), ";")
	upper := strings.ToUpper(cleaned)
	if !strings.Contains(upper, "LIMIT") {
		cleaned = cleaned + " LIMIT 100"
	}

	// 3. 設定執行超時
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 4. 執行查詢
	rows, err := db.WithContext(ctx).Raw(cleaned).Rows()
	if err != nil {
		return nil, fmt.Errorf("SQL 執行失敗: %w", err)
	}
	defer rows.Close()

	// 取得欄位名稱
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("無法取得欄位資訊: %w", err)
	}

	// 掃描資料列（上限 100 筆）
	var resultRows []map[string]interface{}
	maxRows := 100
	truncated := false

	for rows.Next() {
		if len(resultRows) >= maxRows {
			truncated = true
			break
		}

		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("掃描查詢結果失敗: %w", err)
		}

		rowMap := make(map[string]interface{}, len(columns))
		for i, col := range columns {
			val := values[i]
			// 轉換 byte slice 為字串，避免 JSON 序列化成 base64
			if b, ok := val.([]byte); ok {
				rowMap[col] = string(b)
			} else {
				rowMap[col] = val
			}
		}
		resultRows = append(resultRows, rowMap)
	}

	msg := ""
	if truncated {
		msg = "查詢結果已達到 100 筆上限，為確保效能已截斷後續資料。"
	}

	return &QueryResult{
		Columns:   columns,
		Rows:      resultRows,
		RowCount:  len(resultRows),
		Truncated: truncated,
		Message:   msg,
	}, nil
}
