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

func TestConvertViewDDL_MapsDatetimeExtractFunctions(t *testing.T) {
	viewSQL := `SELECT
YEAR(case_09_datetime.d1) AS year_only,
MONTH(case_09_datetime.d1) AS month_only,
DAYOFMONTH(case_09_datetime.d1) AS day_only,
HOUR(case_09_datetime.t1) AS hour_only,
MINUTE(case_09_datetime.t1) AS minute_only,
SECOND(case_09_datetime.t1) AS second_only,
DATE_FORMAT(case_09_datetime.d1, '%Y-%m-%d') AS fmt_date,
DATE_FORMAT(case_09_datetime.dt1, '%Y-%m-%d %H:%i:%s') AS fmt_datetime
FROM case_09_datetime`

	ddl, err := ConvertViewDDL("v_datetime_map", viewSQL)
	if err != nil {
		t.Fatalf("ConvertViewDDL 返回错误: %v", err)
	}

	lowerDDL := strings.ToLower(ddl)
	if strings.Contains(lowerDDL, "year(") || strings.Contains(lowerDDL, "month(") ||
		strings.Contains(lowerDDL, "dayofmonth(") || strings.Contains(lowerDDL, "hour(") ||
		strings.Contains(lowerDDL, "minute(") || strings.Contains(lowerDDL, "second(") {
		t.Fatalf("日期时间提取函数未完整转换: %s", ddl)
	}
	if !strings.Contains(lowerDDL, "extract(year from") ||
		!strings.Contains(lowerDDL, "extract(month from") ||
		!strings.Contains(lowerDDL, "extract(day from") ||
		!strings.Contains(lowerDDL, "extract(hour from") ||
		!strings.Contains(lowerDDL, "extract(minute from") ||
		!strings.Contains(lowerDDL, "extract(second from") {
		t.Fatalf("extract 映射不完整: %s", ddl)
	}
	if !strings.Contains(lowerDDL, "to_char(case_09_datetime.d1, 'yyyy-mm-dd')") {
		t.Fatalf("DATE_FORMAT 日期模板未转换: %s", ddl)
	}
	if !strings.Contains(lowerDDL, "to_char(case_09_datetime.dt1, 'yyyy-mm-dd hh24:mi:ss')") {
		t.Fatalf("DATE_FORMAT 日期时间模板未转换: %s", ddl)
	}
}
