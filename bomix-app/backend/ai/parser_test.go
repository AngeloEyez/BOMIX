package ai

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestParseToolCallsFromContent 測試從各種模型文字輸出格式中解析 ToolCall
func TestParseToolCallsFromContent(t *testing.T) {
	// 案例 1：使用者真實遇到的 Qwen 無參數格式
	qwenRaw := `<tool_call>
<function=get_series_overview

</tool_call>`
	tcs1, clean1 := ParseToolCallsFromContent(qwenRaw)
	if len(tcs1) != 1 {
		t.Fatalf("案例 1 預期解析出 1 個 ToolCall，實際為 %d", len(tcs1))
	}
	if tcs1[0].Function.Name != "get_series_overview" {
		t.Errorf("案例 1 函數名稱預期為 get_series_overview，實際為: %s", tcs1[0].Function.Name)
	}
	if tcs1[0].Function.Arguments != "{}" {
		t.Errorf("案例 1 參數預期為 {}，實際為: %s", tcs1[0].Function.Arguments)
	}
	if clean1 != "" {
		t.Errorf("案例 1 清理後文字預期為空，實際為: %q", clean1)
	}

	// 案例 2：Qwen 帶有 parameter 標籤
	qwenParams := `<tool_call>
<function=execute_readonly_sql>
<parameter=sql>
SELECT * FROM parts;
</parameter>
<parameter=explanation>
查詢物料列表
</parameter>
</function>
</tool_call>`
	tcs2, _ := ParseToolCallsFromContent(qwenParams)
	if len(tcs2) != 1 {
		t.Fatalf("案例 2 預期解析出 1 個 ToolCall，實際為 %d", len(tcs2))
	}
	if tcs2[0].Function.Name != "execute_readonly_sql" {
		t.Errorf("案例 2 函數名稱預期為 execute_readonly_sql，實際為: %s", tcs2[0].Function.Name)
	}
	var args2 map[string]interface{}
	if err := json.Unmarshal([]byte(tcs2[0].Function.Arguments), &args2); err != nil {
		t.Fatalf("案例 2 參數 JSON 解析失敗: %v", err)
	}
	if !strings.Contains(args2["sql"].(string), "SELECT * FROM parts") {
		t.Errorf("案例 2 sql 參數不符合預期: %v", args2["sql"])
	}

	// 案例 3：Hermes JSON 格式
	hermesJSON := `<tool_call>
{"name": "compare_revisions_diff", "arguments": {"base_revision_id": 1, "target_revision_id": 2}}
</tool_call>`
	tcs3, _ := ParseToolCallsFromContent(hermesJSON)
	if len(tcs3) != 1 {
		t.Fatalf("案例 3 預期解析出 1 個 ToolCall，實際為 %d", len(tcs3))
	}
	if tcs3[0].Function.Name != "compare_revisions_diff" {
		t.Errorf("案例 3 函數名稱預期為 compare_revisions_diff，實際為: %s", tcs3[0].Function.Name)
	}

	// 案例 4：帶有前導文字與 Qwen 格式
	mixed := `我先為您查詢系列總覽。
<tool_call>
<function=get_series_overview>
</function>
</tool_call>`
	tcs4, clean4 := ParseToolCallsFromContent(mixed)
	if len(tcs4) != 1 {
		t.Fatalf("案例 4 預期解析出 1 個 ToolCall，實際為 %d", len(tcs4))
	}
	if clean4 != "我先為您查詢系列總覽。" {
		t.Errorf("案例 4 預期保留前導文字，實際為: %q", clean4)
	}
}
