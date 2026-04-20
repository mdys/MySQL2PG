package postgres

import (
	"strings"
	"testing"

	"github.com/yourusername/mysql2pg/internal/mysql"
)

// TestFunctionVariableDeclarationOrder 测试变量声明顺序问题
// 复现错误：func_110_case_daily_numeric_risk_tag 的 DECLARE 语句被错误处理
func TestFunctionVariableDeclarationOrder(t *testing.T) {
	mysqlDDL := `CREATE FUNCTION func_110_case_daily_numeric_risk_tag(_id BIGINT UNSIGNED)
RETURNS VARCHAR(16)
READS SQL DATA
BEGIN
    DECLARE v_dec_high DECIMAL(65,30) DEFAULT 0;
    DECLARE v_ratio NUMERIC(20,10) DEFAULT 0;
    SELECT IFNULL(dec_high, 0), IFNULL(ratio, 0)
      INTO v_dec_high, v_ratio
    FROM case_160_numeric_boundary
    WHERE id = _id
    LIMIT 1;
    IF v_dec_high >= 1000000000000 OR v_ratio >= 100000 THEN
        RETURN 'HIGH';
    ELSEIF v_dec_high >= 1000000 OR v_ratio >= 1000 THEN
        RETURN 'MEDIUM';
    END IF;
    RETURN 'LOW';
END`

	result, err := ConvertFunctionDDL(mysql.FunctionInfo{
		Name: "func_110_case_daily_numeric_risk_tag",
		DDL:  mysqlDDL,
	})

	if err != nil {
		t.Fatalf("转换失败：%v", err)
	}

	t.Logf("转换结果:\n%s", result)

	// 检查 DECLARE 块是否正确
	if !strings.Contains(result, "DECLARE") {
		t.Error("缺少 DECLARE 块")
	}

	// 检查变量声明是否在 DECLARE 块中（而不是在 BEGIN 之后）
	if strings.Contains(result, "BEGIN\nv_ratio") || strings.Contains(result, "BEGIN\nv_dec_high") {
		t.Error("变量声明错误地出现在 BEGIN 之后，应该在 DECLARE 块中")
	}

	// 检查是否包含正确的变量声明
	if !strings.Contains(result, "v_dec_high DECIMAL") && !strings.Contains(result, "v_dec_high NUMERIC") {
		t.Error("缺少 v_dec_high 变量声明")
	}

	if !strings.Contains(result, "v_ratio NUMERIC") && !strings.Contains(result, "v_ratio DECIMAL") {
		t.Error("缺少 v_ratio 变量声明")
	}
}

// TestFunctionCursorClose 测试游标 CLOSE 语句处理
// 复现错误：func_001_complex_analysis 的 CLOSE cur_complex 被错误处理为 cur_complex;
func TestFunctionCursorClose(t *testing.T) {
	mysqlDDL := `CREATE FUNCTION func_001_complex_analysis(param_limit INT) RETURNS TEXT
READS SQL DATA
BEGIN
    DECLARE done INT DEFAULT FALSE;
    DECLARE v_result TEXT DEFAULT '';
    DECLARE v_counter INT DEFAULT 0;
    DECLARE cur_complex CURSOR FOR
        SELECT id FROM case_01_integers LIMIT param_limit;
    DECLARE CONTINUE HANDLER FOR NOT FOUND SET done = TRUE;

    OPEN cur_complex;
    read_loop: LOOP
        FETCH cur_complex INTO v_result;
        IF done THEN
            LEAVE read_loop;
        END IF;
        SET v_counter = v_counter + 1;
    END LOOP;
    CLOSE cur_complex;

    RETURN v_result;
END`

	result, err := ConvertFunctionDDL(mysql.FunctionInfo{
		Name: "func_001_complex_analysis",
		DDL:  mysqlDDL,
	})

	if err != nil {
		t.Fatalf("转换失败：%v", err)
	}

	t.Logf("转换结果:\n%s", result)

	// 检查是否有正确的 CLOSE 语句
	if !strings.Contains(result, "CLOSE cur_complex") {
		t.Error("缺少 CLOSE cur_complex 语句")
	}

	// 检查是否有错误的孤立游标名称
	if strings.Contains(result, "END LOOP;\ncur_complex") || strings.Contains(result, "END LOOP; cur_complex") {
		t.Error("游标名称错误地出现在 END LOOP 之后，应该是 CLOSE 语句")
	}
}

// TestFunctionNumericTypeInDeclare 测试 NUMERIC 类型在 DECLARE 中的处理
func TestFunctionNumericTypeInDeclare(t *testing.T) {
	mysqlDDL := `CREATE FUNCTION test_numeric_declare(_id BIGINT)
RETURNS VARCHAR(16)
READS SQL DATA
BEGIN
    DECLARE v_ratio NUMERIC(20,10) DEFAULT 0;
    SELECT ratio INTO v_ratio FROM test_table WHERE id = _id;
    RETURN 'LOW';
END`

	result, err := ConvertFunctionDDL(mysql.FunctionInfo{
		Name: "test_numeric_declare",
		DDL:  mysqlDDL,
	})

	if err != nil {
		t.Fatalf("转换失败：%v", err)
	}

	t.Logf("转换结果:\n%s", result)

	// 检查 NUMERIC 类型是否被正确保留
	if !strings.Contains(result, "NUMERIC(20,10)") && !strings.Contains(result, "DECIMAL(20,10)") {
		t.Error("NUMERIC(20,10) 类型没有被正确转换")
	}
}

// TestFunctionConcatWsWithCommaSeparator 测试 CONCAT_WS 在分隔符为逗号时的转换
func TestFunctionConcatWsWithCommaSeparator(t *testing.T) {
	mysqlDDL := `CREATE FUNCTION test_concat_ws_cursor(param_limit INT) RETURNS TEXT
READS SQL DATA
BEGIN
    DECLARE done INT DEFAULT FALSE;
    DECLARE v_result TEXT DEFAULT '';
    DECLARE cur_complex CURSOR FOR
        SELECT CONCAT_WS(',', t1.id, t1.event_date, t2.id) FROM case_97_partition_range_columns t1
        LEFT JOIN case_88_year_conversion t2 ON t1.id = t2.id LIMIT param_limit;
    DECLARE CONTINUE HANDLER FOR NOT FOUND SET done = TRUE;

    OPEN cur_complex;
    read_loop: LOOP
        FETCH cur_complex INTO v_result;
        IF done THEN
            LEAVE read_loop;
        END IF;
    END LOOP;
    CLOSE cur_complex;
    RETURN v_result;
END`

	result, err := ConvertFunctionDDL(mysql.FunctionInfo{
		Name: "test_concat_ws_cursor",
		DDL:  mysqlDDL,
	})
	if err != nil {
		t.Fatalf("转换失败：%v", err)
	}

	lowerResult := strings.ToLower(result)
	if strings.Contains(lowerResult, "concat_ws(") {
		t.Fatalf("CONCAT_WS 未被转换：%s", result)
	}
	if strings.Contains(lowerResult, "array_to_string(array[',") {
		t.Fatalf("CONCAT_WS 发生参数错位：%s", result)
	}
	if !strings.Contains(lowerResult, "array_to_string(array[") {
		t.Fatalf("缺少 ARRAY_TO_STRING 转换结果：%s", result)
	}
}

// 辅助函数：打印转换结果的调试信息
func debugConversionResult(t *testing.T, name, ddl string) {
	result, err := ConvertFunctionDDL(mysql.FunctionInfo{
		Name: name,
		DDL:  ddl,
	})
	if err != nil {
		t.Logf("转换 %s 失败：%v", name, err)
		return
	}
	t.Logf("=== %s 转换结果 ===", name)
	t.Logf("%s", result)
	t.Logf("=== END ===")
}

func TestDebugFunctionConversion(t *testing.T) {
	// 从 create_function.sql 中读取实际函数进行测试
	t.Run("func_110_case_daily_numeric_risk_tag", func(t *testing.T) {
		mysqlDDL := `CREATE FUNCTION func_110_case_daily_numeric_risk_tag(_id BIGINT UNSIGNED)
RETURNS VARCHAR(16)
READS SQL DATA
BEGIN
    DECLARE v_dec_high DECIMAL(65,30) DEFAULT 0;
    DECLARE v_ratio NUMERIC(20,10) DEFAULT 0;
    SELECT IFNULL(dec_high, 0), IFNULL(ratio, 0)
      INTO v_dec_high, v_ratio
    FROM case_160_numeric_boundary
    WHERE id = _id
    LIMIT 1;
    IF v_dec_high >= 1000000000000 OR v_ratio >= 100000 THEN
        RETURN 'HIGH';
    ELSEIF v_dec_high >= 1000000 OR v_ratio >= 1000 THEN
        RETURN 'MEDIUM';
    END IF;
    RETURN 'LOW';
END`
		debugConversionResult(t, "func_110_case_daily_numeric_risk_tag", mysqlDDL)
	})

	t.Run("func_001_complex_analysis", func(t *testing.T) {
		mysqlDDL := `CREATE FUNCTION func_001_complex_analysis(param_limit INT) RETURNS TEXT
READS SQL DATA
BEGIN
    DECLARE done INT DEFAULT FALSE;
    DECLARE v_result TEXT DEFAULT '';
    DECLARE v_counter INT DEFAULT 0;
    DECLARE cur_complex CURSOR FOR
        SELECT id FROM case_01_integers LIMIT param_limit;
    DECLARE CONTINUE HANDLER FOR NOT FOUND SET done = TRUE;

    OPEN cur_complex;
    read_loop: LOOP
        FETCH cur_complex INTO v_result;
        IF done THEN
            LEAVE read_loop;
        END IF;
        SET v_counter = v_counter + 1;
    END LOOP;
    CLOSE cur_complex;

    RETURN v_result;
END`
		debugConversionResult(t, "func_001_complex_analysis", mysqlDDL)
	})
}

// TestFunctionConcatWsWithCommaSeparator 测试 CONCAT_WS 在分隔符为逗号时的转换正确性。
func TestFunctionConcatWsWithCommaSeparator(t *testing.T) {
	mysqlDDL := `CREATE FUNCTION func_concat_ws_case()
RETURNS TEXT
READS SQL DATA
BEGIN
    DECLARE v_result TEXT DEFAULT '';
    SELECT CONCAT_WS(',', t1.id, t1.col_int, t1.col_bigint)
      INTO v_result
    FROM case_01_integers t1
    LIMIT 1;
    RETURN v_result;
END`

	result, err := ConvertFunctionDDL(mysql.FunctionInfo{
		Name: "func_concat_ws_case",
		DDL:  mysqlDDL,
	})
	if err != nil {
		t.Fatalf("转换失败：%v", err)
	}

	t.Logf("转换结果:\n%s", result)
	if !strings.Contains(result, "ARRAY_TO_STRING(ARRAY[t1.id, t1.col_int, t1.col_bigint], ',')") {
		t.Fatalf("CONCAT_WS 转换结果不符合预期：%s", result)
	}
}
