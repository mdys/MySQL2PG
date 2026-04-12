package postgres

import (
	"strings"
	"testing"
)

func TestConvertViewDDL_MapsJSONUnquoteAndExtract(t *testing.T) {
	viewSQL := `SELECT
JSON_EXTRACT(case_08_json.data, '$.name') AS json_name,
JSON_UNQUOTE(JSON_EXTRACT(case_08_json.data, '$.name')) AS json_name_unquoted
FROM case_08_json`

	ddl, err := ConvertViewDDL("v_json_map", viewSQL)
	if err != nil {
		t.Fatalf("ConvertViewDDL 返回错误: %v", err)
	}

	lowerDDL := strings.ToLower(ddl)
	if strings.Contains(lowerDDL, "jsonb_unquote(") {
		t.Fatalf("不应包含不存在的 jsonb_unquote 函数: %s", ddl)
	}
	if !strings.Contains(lowerDDL, "-> 'name'") {
		t.Fatalf("JSON_EXTRACT 未转换为 -> 'name': %s", ddl)
	}
	if !strings.Contains(lowerDDL, "->> 'name'") {
		t.Fatalf("JSON_UNQUOTE(JSON_EXTRACT(...)) 未转换为 ->> 'name': %s", ddl)
	}
}
