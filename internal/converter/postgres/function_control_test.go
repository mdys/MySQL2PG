package postgres

import (
	"regexp"
	"strings"
	"testing"

	"github.com/yourusername/mysql2pg/internal/mysql"
)

// TestConvertFunction_LEAVE_ITERATE 测试 LEAVE 和 ITERATE 转换
func TestConvertFunction_LEAVE_ITERATE(t *testing.T) {
	mysqlDDL := `CREATE FUNCTION test_leave_iterate(param INT) RETURNS TEXT
READS SQL DATA
BEGIN
    DECLARE done INT DEFAULT FALSE;
    DECLARE v_result TEXT DEFAULT '';
    DECLARE cur_test CURSOR FOR SELECT id FROM test_table;
    DECLARE CONTINUE HANDLER FOR NOT FOUND SET done = TRUE;

    OPEN cur_test;
    read_loop: LOOP
        FETCH cur_test INTO v_result;
        IF done THEN
            LEAVE read_loop;
        END IF;
        IF LENGTH(v_result) > 10 THEN
            ITERATE read_loop;
        END IF;
    END LOOP;
    CLOSE cur_test;

    RETURN v_result;
END`

	result, err := ConvertFunctionDDL(mysql.FunctionInfo{
		Name: "test_leave_iterate",
		DDL:  mysqlDDL,
	})

	if err != nil {
		t.Fatalf("转换失败：%v", err)
	}

	t.Logf("转换结果：%s", result)

	// 检查 LEAVE 转换为 EXIT（不检查标签，因为标签可能被保留）
	if !strings.Contains(result, "EXIT") {
		t.Error("LEAVE 未转换为 EXIT")
	}
	// 检查是否有 LEAVE 关键字（不区分大小写，但不匹配标签名）
	leaveRegex := regexp.MustCompile(`(?i)\bLEAVE\s+\w+`)
	if leaveRegex.MatchString(result) {
		t.Error("结果中仍包含 LEAVE 关键字")
	}

	// 检查 ITERATE 转换为 CONTINUE
	if !strings.Contains(result, "CONTINUE") {
		t.Error("ITERATE 未转换为 CONTINUE")
	}
	// 检查是否有 ITERATE 关键字（不区分大小写，但不匹配标签名）
	iterateRegex := regexp.MustCompile(`(?i)\bITERATE\s+\w+`)
	if iterateRegex.MatchString(result) {
		t.Error("结果中仍包含 ITERATE 关键字")
	}
}

// TestConvertFunction_WHILE_DO 测试 WHILE ... DO 转换
func TestConvertFunction_WHILE_DO(t *testing.T) {
	mysqlDDL := `CREATE FUNCTION test_while_do(_limit INT) RETURNS INT
READS SQL DATA
BEGIN
    DECLARE v_counter INT DEFAULT 0;
    DECLARE v_sum INT DEFAULT 0;

    WHILE v_counter < _limit DO
        SET v_counter = v_counter + 1;
        SET v_sum = v_sum + v_counter;
    END WHILE;

    RETURN v_sum;
END`

	result, err := ConvertFunctionDDL(mysql.FunctionInfo{
		Name: "test_while_do",
		DDL:  mysqlDDL,
	})

	if err != nil {
		t.Fatalf("转换失败：%v", err)
	}

	t.Logf("转换结果：%s", result)

	// 检查 WHILE ... DO 转换为 WHILE ... LOOP
	if !strings.Contains(result, "WHILE") || !strings.Contains(result, "LOOP") {
		t.Error("WHILE ... DO 未转换为 WHILE ... LOOP")
	}
	if strings.Contains(strings.ToUpper(result), "END WHILE") {
		t.Error("结果中仍包含 END WHILE")
	}
}

// TestConvertFunction_ReturnTypes 测试返回类型转换
func TestConvertFunction_ReturnTypes(t *testing.T) {
	testCases := []struct {
		name         string
		mysqlType    string
		expectedType string
	}{
		{"DOUBLE", "DOUBLE", "DOUBLE PRECISION"},
		{"DOUBLE(10,2)", "DOUBLE(10,2)", "DOUBLE PRECISION"},
		{"INT(11)", "INT(11)", "INTEGER"},
		{"INT UNSIGNED", "INT UNSIGNED", "INTEGER"},
		{"DECIMAL(65,30)", "DECIMAL(65,30)", "DECIMAL(65,30)"},
		{"VARCHAR(255)", "VARCHAR(255)", "VARCHAR(255)"},
		{"DATETIME(6)", "DATETIME(6)", "TIMESTAMP(6)"},
		{"TINYINT(1)", "TINYINT(1)", "INTEGER"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mysqlDDL := "CREATE FUNCTION test_type() RETURNS " + tc.mysqlType + " READS SQL DATA BEGIN RETURN 1; END"

			result, err := ConvertFunctionDDL(mysql.FunctionInfo{
				Name: "test_type",
				DDL:  mysqlDDL,
			})

			if err != nil {
				t.Fatalf("转换失败：%v", err)
			}

			t.Logf("转换结果：%s", result)

			if !strings.Contains(result, "RETURNS "+tc.expectedType) {
				t.Errorf("期望返回类型包含 %s，但得到：%s", tc.expectedType, result)
			}
		})
	}
}

// TestConvertFunction_READS_SQL_DATA 测试 READS SQL DATA 移除
func TestConvertFunction_READS_SQL_DATA(t *testing.T) {
	mysqlDDL := `CREATE FUNCTION test_reads_sql_data(id INT) RETURNS TEXT
READS SQL DATA
BEGIN
    DECLARE v_result TEXT;
    SELECT name INTO v_result FROM test_table WHERE id = id;
    RETURN v_result;
END`

	result, err := ConvertFunctionDDL(mysql.FunctionInfo{
		Name: "test_reads_sql_data",
		DDL:  mysqlDDL,
	})

	if err != nil {
		t.Fatalf("转换失败：%v", err)
	}

	t.Logf("转换结果：%s", result)

	// 检查 READS SQL DATA 已被移除
	if strings.Contains(strings.ToUpper(result), "READS SQL DATA") {
		t.Error("READS SQL DATA 未被移除")
	}
}

// TestConvertFunction_DETERMINISTIC 测试 DETERMINISTIC 转换
func TestConvertFunction_DETERMINISTIC(t *testing.T) {
	mysqlDDL := `CREATE FUNCTION test_deterministic(x INT) RETURNS INT
DETERMINISTIC
BEGIN
    RETURN x * 2;
END`

	result, err := ConvertFunctionDDL(mysql.FunctionInfo{
		Name: "test_deterministic",
		DDL:  mysqlDDL,
	})

	if err != nil {
		t.Fatalf("转换失败：%v", err)
	}

	t.Logf("转换结果：%s", result)

	// 检查 DETERMINISTIC 转换为 IMMUTABLE 或 STABLE
	resultUpper := strings.ToUpper(result)
	if !strings.Contains(resultUpper, "IMMUTABLE") && !strings.Contains(resultUpper, "STABLE") {
		t.Error("DETERMINISTIC 未转换为 IMMUTABLE 或 STABLE")
	}
}
