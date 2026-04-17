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
		t.Fatalf("ConvertViewDDL 返回错误：%v", err)
	}

	lowerDDL := strings.ToLower(ddl)
	if strings.Contains(lowerDDL, "jsonb_unquote(") {
		t.Fatalf("不应包含不存在的 jsonb_unquote 函数：%s", ddl)
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
		t.Fatalf("ConvertViewDDL 返回错误：%v", err)
	}

	lowerDDL := strings.ToLower(ddl)
	if strings.Contains(lowerDDL, "year(") || strings.Contains(lowerDDL, "month(") ||
		strings.Contains(lowerDDL, "dayofmonth(") || strings.Contains(lowerDDL, "hour(") ||
		strings.Contains(lowerDDL, "minute(") || strings.Contains(lowerDDL, "second(") {
		t.Fatalf("日期时间提取函数未完整转换：%s", ddl)
	}
	if !strings.Contains(lowerDDL, "extract(year from") ||
		!strings.Contains(lowerDDL, "extract(month from") ||
		!strings.Contains(lowerDDL, "extract(day from") ||
		!strings.Contains(lowerDDL, "extract(hour from") ||
		!strings.Contains(lowerDDL, "extract(minute from") ||
		!strings.Contains(lowerDDL, "extract(second from") {
		t.Fatalf("extract 映射不完整：%s", ddl)
	}
	if !strings.Contains(lowerDDL, "to_char(case_09_datetime.d1, 'yyyy-mm-dd')") {
		t.Fatalf("DATE_FORMAT 日期模板未转换：%s", ddl)
	}
	if !strings.Contains(lowerDDL, "to_char(case_09_datetime.dt1, 'yyyy-mm-dd hh24:mi:ss')") {
		t.Fatalf("DATE_FORMAT 日期时间模板未转换：%s", ddl)
	}
}

// TestConvertViewDDL_RegexpLike 测试 REGEXP_LIKE 函数转换 (MySQL 8.0+)
func TestConvertViewDDL_RegexpLike(t *testing.T) {
	viewSQL := `SELECT
    case_05_charsets.c1,
    case_05_charsets.c2,
    REGEXP_LIKE(case_05_charsets.c1, '^[a-zA-Z]+$') AS is_alpha_c1,
    REGEXP_LIKE(case_05_charsets.c2, '^[0-9]+$') AS is_numeric_c2,
    REGEXP_LIKE(c3, 'test') AS has_test
FROM case_05_charsets`

	ddl, err := ConvertViewDDL("view_case25_mysql8_regexp", viewSQL)
	if err != nil {
		t.Fatalf("ConvertViewDDL 返回错误：%v", err)
	}

	t.Logf("转换结果：%s", ddl)

	// 检查转换结果（SQL 会被转为小写）
	if !strings.Contains(ddl, "~ '^[a-za-z]+$'") {
		t.Errorf("REGEXP_LIKE(c1, '^[a-zA-Z]+$') 未正确转换为 ~ 操作符：%s", ddl)
	}
	if !strings.Contains(ddl, "~ '^[0-9]+$'") {
		t.Errorf("REGEXP_LIKE(c2, '^[0-9]+$') 未正确转换为 ~ 操作符：%s", ddl)
	}
	if !strings.Contains(ddl, "~ 'test'") {
		t.Errorf("REGEXP_LIKE(c3, 'test') 未正确转换为 ~ 操作符：%s", ddl)
	}

	// 检查不再包含 REGEXP_LIKE 函数调用
	lowerDDL := strings.ToLower(ddl)
	if strings.Contains(lowerDDL, "regexp_like(") {
		t.Errorf("转换后仍包含 regexp_like 函数：%s", ddl)
	}
}

// TestConvertViewDDL_RegexpLikeWithQuotes 测试带引号的 REGEXP_LIKE 转换
func TestConvertViewDDL_RegexpLikeWithQuotes(t *testing.T) {
	viewSQL := `SELECT 
    REGEXP_LIKE(name, '^[A-Z][a-z]+') AS valid_name,
    REGEXP_LIKE(email, '^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$') AS valid_email
FROM users`

	ddl, err := ConvertViewDDL("v_users_regexp", viewSQL)
	if err != nil {
		t.Fatalf("ConvertViewDDL 返回错误：%v", err)
	}

	t.Logf("转换结果：%s", ddl)

	// SQL 会被转为小写，检查小写形式
	if !strings.Contains(ddl, "name ~ '^[a-z][a-z]+'") {
		t.Errorf("REGEXP_LIKE(name, ...) 转换失败：%s", ddl)
	}
	if !strings.Contains(ddl, "email ~ '^[a-za-z0-9._%+-]+@[a-za-z0-9.-]+") {
		t.Errorf("REGEXP_LIKE(email, ...) 转换失败：%s", ddl)
	}
}

// TestConvertViewDDL_RegexpLikeWithColumnRef 测试列引用的 REGEXP_LIKE 转换
func TestConvertViewDDL_RegexpLikeWithColumnRef(t *testing.T) {
	viewSQL := `SELECT 
    REGEXP_LIKE(t1.c1, t2.pattern) AS matches
FROM table1 t1, table2 t2`

	ddl, err := ConvertViewDDL("v_cross_regexp", viewSQL)
	if err != nil {
		t.Fatalf("ConvertViewDDL 返回错误：%v", err)
	}

	if !strings.Contains(ddl, "t1.c1 ~ t2.pattern") {
		t.Errorf("REGEXP_LIKE(t1.c1, t2.pattern) 转换失败：%s", ddl)
	}
}

// TestConvertViewDDL_Locate 测试 LOCATE 函数转换
func TestConvertViewDDL_Locate(t *testing.T) {
	viewSQL := `SELECT
    LOCATE('test', case_05_charsets.c4) AS test_pos_c4,
    LOCATE('abc', name) AS pos_name,
    LOCATE(sub, str) AS pos_var
FROM case_05_charsets`

	ddl, err := ConvertViewDDL("view_case25_locate", viewSQL)
	if err != nil {
		t.Fatalf("ConvertViewDDL 返回错误：%v", err)
	}

	t.Logf("转换结果：%s", ddl)

	// 检查转换结果（LOCATE('test', c4) -> strpos(c4, 'test')）
	// SQL 会被转为小写
	if !strings.Contains(ddl, "strpos(case_05_charsets.c4, 'test')") {
		t.Errorf("LOCATE 未正确转换为 strpos：%s", ddl)
	}

	// 检查不再包含 LOCATE 函数调用
	lowerDDL := strings.ToLower(ddl)
	if strings.Contains(lowerDDL, "locate(") {
		t.Errorf("转换后仍包含 locate 函数：%s", ddl)
	}

	// 检查参数顺序正确（substr 和 str 位置交换）
	if !strings.Contains(ddl, "strpos(name, 'abc')") {
		t.Errorf("LOCATE 参数顺序错误，应该是 strpos(str, substr)：%s", ddl)
	}
}
