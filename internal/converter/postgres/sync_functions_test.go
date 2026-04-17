package postgres

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/yourusername/mysql2pg/internal/mysql"
)

// splitSQLFunctions 将 SQL 文件内容拆分为独立的函数 DDL 块
func splitSQLFunctions(content string) []string {
	var blocks []string
	for _, block := range strings.Split(content, "DELIMITER //") {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		for _, fb := range strings.Split(block, "END //") {
			fb = strings.TrimSpace(fb)
			if fb == "" {
				continue
			}
			var lines []string
			for _, line := range strings.Split(fb, "\n") {
				upper := strings.ToUpper(strings.TrimSpace(line))
				if strings.HasPrefix(upper, "DROP FUNCTION") ||
					strings.HasPrefix(upper, "DELIMITER") ||
					strings.TrimSpace(line) == "" {
					continue
				}
				lines = append(lines, line)
			}
			ddl := strings.TrimSpace(strings.Join(lines, "\n"))
			if strings.Contains(strings.ToUpper(ddl), "CREATE FUNCTION") {
				blocks = append(blocks, ddl)
			}
		}
	}
	return blocks
}

// TestCreateFunctionSQL_AllFunctions 测试 create_function.sql 中所有函数的转换
func TestCreateFunctionSQL_AllFunctions(t *testing.T) {
	sqlPath := "../../../scripts/mysql/create_function.sql"
	content, err := os.ReadFile(sqlPath)
	if err != nil {
		t.Skipf("无法读取 create_function.sql: %v", err)
	}

	blocks := splitSQLFunctions(string(content))
	if len(blocks) == 0 {
		t.Fatal("未从 create_function.sql 中解析到任何函数")
	}

	t.Logf("共解析到 %d 个函数", len(blocks))

	var successCount, failCount int
	var failedFunctions []string

	for _, ddl := range blocks {
		re := regexp.MustCompile(`(?i)CREATE\s+FUNCTION\s+(\w+)`)
		matches := re.FindStringSubmatch(ddl)
		if matches == nil {
			continue
		}
		funcName := matches[1]

		t.Run(funcName, func(t *testing.T) {
			// 直接调用 ConvertFunctionDDL 入口函数
			result, err := ConvertFunctionDDL(mysql.FunctionInfo{
				Name: funcName,
				DDL:  ddl,
			})
			if err != nil {
				t.Errorf("转换失败: %v", err)
				return
			}

			if !strings.Contains(result, "CREATE OR REPLACE FUNCTION") {
				t.Error("缺少 CREATE OR REPLACE FUNCTION")
			}
			if !strings.Contains(result, "RETURNS") {
				t.Error("缺少 RETURNS")
			}
			if !strings.Contains(result, "LANGUAGE plpgsql") {
				t.Error("缺少 LANGUAGE plpgsql")
			}
		})
	}

	t.Run("__summary__", func(t *testing.T) {
		for _, ddl := range blocks {
			re := regexp.MustCompile(`(?i)CREATE\s+FUNCTION\s+(\w+)`)
			matches := re.FindStringSubmatch(ddl)
			if matches == nil {
				continue
			}
			funcName := matches[1]

			_, err := ConvertFunctionDDL(mysql.FunctionInfo{
				Name: funcName,
				DDL:  ddl,
			})
			if err != nil {
				failCount++
				failedFunctions = append(failedFunctions, funcName+" ("+err.Error()+")")
			} else {
				successCount++
			}
		}

		t.Logf("========== 转换汇总 ==========")
		t.Logf("总数: %d | 成功: %d | 失败: %d", len(blocks), successCount, failCount)
		if failCount > 0 {
			t.Logf("失败列表:")
			for _, f := range failedFunctions {
				t.Logf("  - %s", f)
			}
		}
	})
}

// TestCreateFunctionSQL_CompatibilityFunctions 测试兼容性函数
func TestCreateFunctionSQL_CompatibilityFunctions(t *testing.T) {
	sqlPath := "../../../scripts/mysql/create_function.sql"
	content, err := os.ReadFile(sqlPath)
	if err != nil {
		t.Skipf("无法读取 create_function.sql: %v", err)
	}

	blocks := splitSQLFunctions(string(content))

	for _, ddl := range blocks {
		re := regexp.MustCompile(`(?i)CREATE\s+FUNCTION\s+(fn_case_compat_\w+)`)
		matches := re.FindStringSubmatch(ddl)
		if matches == nil {
			continue
		}
		funcName := matches[1]

		t.Run(funcName, func(t *testing.T) {
			result, err := ConvertFunctionDDL(mysql.FunctionInfo{
				Name: funcName,
				DDL:  ddl,
			})
			if err != nil {
				t.Fatalf("转换失败: %v", err)
			}

			if !strings.Contains(result, "CREATE OR REPLACE FUNCTION") {
				t.Error("缺少 CREATE OR REPLACE FUNCTION")
			}
			t.Logf("转换后的 DDL:\n%s", result)
		})
	}
}

// TestCreateFunctionSQL_ComplexFunctions 测试复杂函数转换
func TestCreateFunctionSQL_ComplexFunctions(t *testing.T) {
	sqlPath := "../../../scripts/mysql/create_function.sql"
	content, err := os.ReadFile(sqlPath)
	if err != nil {
		t.Skipf("无法读取 create_function.sql: %v", err)
	}

	blocks := splitSQLFunctions(string(content))
	complexNames := []string{
		"func_001_complex_analysis",
		"func_008_complex_analysis",
		"func_020_complex_analysis",
		"func_084_complex_analysis",
		"func_092_complex_analysis",
		"func_095_complex_analysis",
	}

	for _, name := range complexNames {
		var ddl string
		for _, block := range blocks {
			if strings.Contains(block, "CREATE FUNCTION "+name) {
				ddl = block
				break
			}
		}
		if ddl == "" {
			t.Skipf("未找到函数 %s", name)
			continue
		}

		t.Run(name, func(t *testing.T) {
			result, err := ConvertFunctionDDL(mysql.FunctionInfo{
				Name: name,
				DDL:  ddl,
			})
			if err != nil {
				t.Fatalf("转换失败: %v", err)
			}

			if !strings.Contains(result, "CREATE OR REPLACE FUNCTION") {
				t.Error("缺少 CREATE OR REPLACE FUNCTION")
			}

			if strings.Contains(strings.ToLower(ddl), "cursor") {
				if !strings.Contains(result, "refcursor") {
					t.Error("游标声明未转换为 refcursor")
				}
				if !strings.Contains(result, "FETCH NEXT FROM") {
					t.Error("FETCH 未转换")
				}
			}

			if strings.Contains(strings.ToLower(ddl), "concat_ws") {
				if strings.Contains(result, "CONCAT_WS(") {
					t.Error("CONCAT_WS 未转换")
				}
			}

			t.Logf("转换成功，DDL 长度: %d 字符", len(result))
		})
	}
}

// TestCreateFunctionSQL_FeaturesCoverage 测试特定功能覆盖
func TestCreateFunctionSQL_FeaturesCoverage(t *testing.T) {
	sqlPath := "../../../scripts/mysql/create_function.sql"
	content, err := os.ReadFile(sqlPath)
	if err != nil {
		t.Skipf("无法读取 create_function.sql: %v", err)
	}

	blocks := splitSQLFunctions(string(content))

	features := map[string]string{
		"JSON_UNQUOTE+JSON_EXTRACT": "func_102_case_157_extract_bizid",
		"DATE_FORMAT":               "func_103_case_158_period_key",
		"LENGTH+BLOB":               "func_104_case_159_attachment_size",
		"DECIMAL 高精度":             "func_105_case_160_numeric_score",
		"AVG 聚合":                   "func_107_case_daily_order_avg_price",
		"STR_TO_DATE":               "func_108_case_daily_payload_event_time",
		"ELSEIF 链":                 "func_110_case_daily_numeric_risk_tag",
		"done=1 兼容":               "fn_case_compat_memberratio",
		"WHILE 循环":                 "fn_case_compat_unspecial",
		"REPLACE 函数":               "fn_case_compat_unspecial",
	}

	for feature, funcName := range features {
		t.Run(feature, func(t *testing.T) {
			var ddl string
			for _, block := range blocks {
				if strings.Contains(block, "CREATE FUNCTION "+funcName) {
					ddl = block
					break
				}
			}
			if ddl == "" {
				t.Skipf("未找到函数 %s", funcName)
			}

			result, err := ConvertFunctionDDL(mysql.FunctionInfo{
				Name: funcName,
				DDL:  ddl,
			})
			if err != nil {
				t.Fatalf("转换失败: %v", err)
			}

			t.Logf("函数 %s 转换成功，DDL:\n%s", funcName, result)
		})
	}
}
