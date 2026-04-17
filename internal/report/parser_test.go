package report

import (
	"os"
	"path/filepath"
	"testing"
)

// TestParseLog_TableConversionSuccess 测试表转换成功日志解析
func TestParseLog_TableConversionSuccess(t *testing.T) {
	logContent := `[2026-04-16 10:00:00] 转换表 users 成功
[2026-04-16 10:00:01] 转换表 orders 成功
[2026-04-16 10:00:02] 表 sessions 已存在，跳过创建
`
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "conversion.log")
	os.WriteFile(logFile, []byte(logContent), 0644)

	report, err := ParseLog(logFile)
	if err != nil {
		t.Fatalf("ParseLog 失败: %v", err)
	}

	if len(report.TableDetails) != 3 {
		t.Errorf("期望 3 个表详情，实际 %d", len(report.TableDetails))
	}

	// 验证表名
	names := []string{}
	for _, td := range report.TableDetails {
		names = append(names, td.Name)
	}
	expectedNames := []string{"users", "orders", "sessions"}
	for i, name := range expectedNames {
		if i >= len(names) || names[i] != name {
			t.Errorf("期望表名 %s，实际 %v", name, names)
			break
		}
	}
}

// TestParseLog_TableSyncWithRowCount 测试表同步带行数
func TestParseLog_TableSyncWithRowCount(t *testing.T) {
	logContent := `[2026-04-16 10:01:00] 分页同步表 users 完成，共处理 100 行数据
[2026-04-16 10:01:01] 表 orders 没有数据，跳过同步
`
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "conversion.log")
	os.WriteFile(logFile, []byte(logContent), 0644)

	report, err := ParseLog(logFile)
	if err != nil {
		t.Fatalf("ParseLog 失败: %v", err)
	}

	// 验证 users 表行数
	for _, td := range report.TableDetails {
		if td.Name == "users" {
			if td.RowCount != 100 {
				t.Errorf("users 表期望 100 行，实际 %d", td.RowCount)
			}
		}
		if td.Name == "orders" {
			if td.Validation != "空表" {
				t.Errorf("orders 表期望验证为 '空表'，实际 %q", td.Validation)
			}
		}
	}
}

// TestParseLog_DataInconsistency 测试数据不一致解析
func TestParseLog_DataInconsistency(t *testing.T) {
	logContent := `[2026-04-16 10:02:00] 表 act_hi_comment 同步完成，10 行数据，数据不一致
`
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "conversion.log")
	os.WriteFile(logFile, []byte(logContent), 0644)

	report, err := ParseLog(logFile)
	if err != nil {
		t.Fatalf("ParseLog 失败: %v", err)
	}

	// 不一致表应该在 TableDetails 中
	found := false
	for _, td := range report.TableDetails {
		if td.Name == "act_hi_comment" && td.Validation == "数据不一致" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("未找到 act_hi_comment 表的数据不一致标记")
	}
}

// TestParseLog_Warnings 测试警告解析
func TestParseLog_Warnings(t *testing.T) {
	logContent := `[2026-04-16 10:03:00] 警告: 表 sessions 没有主键，可能导致同步性能下降
`
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "conversion.log")
	os.WriteFile(logFile, []byte(logContent), 0644)

	report, err := ParseLog(logFile)
	if err != nil {
		t.Fatalf("ParseLog 失败: %v", err)
	}

	if len(report.Warnings) != 1 {
		t.Fatalf("期望 1 个警告，实际 %d", len(report.Warnings))
	}

	if report.Warnings[0] != "表 sessions: 没有主键，可能导致同步性能下降" {
		t.Errorf("警告内容不匹配: %s", report.Warnings[0])
	}
}

// TestParseLog_Errors 测试错误解析
func TestParseLog_Errors(t *testing.T) {
	logContent := `[2026-04-16 10:04:00] 插入表 sessions 数据失败: connection timeout
`
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "conversion.log")
	os.WriteFile(logFile, []byte(logContent), 0644)

	report, err := ParseLog(logFile)
	if err != nil {
		t.Fatalf("ParseLog 失败: %v", err)
	}

	// 表级错误应该在 Errors 列表中
	if len(report.Errors) != 1 {
		t.Fatalf("期望 1 个错误，实际 %d", len(report.Errors))
	}

	if report.Errors[0] != "表 sessions 插入表失败: connection timeout" {
		t.Errorf("错误内容不匹配: %s", report.Errors[0])
	}
}

// TestParseLog_VersionInfo 测试版本信息解析
func TestParseLog_VersionInfo(t *testing.T) {
	logContent := `[2026-04-16 10:00:00] MySQL | 8.0.35
[2026-04-16 10:00:01] PostgreSQL | 16.1
`
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "conversion.log")
	os.WriteFile(logFile, []byte(logContent), 0644)

	report, err := ParseLog(logFile)
	if err != nil {
		t.Fatalf("ParseLog 失败: %v", err)
	}

	if report.MySQLVersion != "8.0.35" {
		t.Errorf("期望 MySQL 版本 8.0.35，实际 %s", report.MySQLVersion)
	}
	if report.PGVersion != "16.1" {
		t.Errorf("期望 PG 版本 16.1，实际 %s", report.PGVersion)
	}
}

// TestParseLog_ProgressSummary 测试进度摘要解析
func TestParseLog_ProgressSummary(t *testing.T) {
	logContent := `[2026-04-16 10:05:00] 进度: 100.00% (192/192)`
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "conversion.log")
	os.WriteFile(logFile, []byte(logContent), 0644)

	report, err := ParseLog(logFile)
	if err != nil {
		t.Fatalf("ParseLog 失败: %v", err)
	}

	if report.ProgressCurrent != 192 {
		t.Errorf("期望当前进度 192，实际 %d", report.ProgressCurrent)
	}
	if report.ProgressTotal != 192 {
		t.Errorf("期望总任务数 192，实际 %d", report.ProgressTotal)
	}
	if !report.ProgressComplete {
		t.Error("期望进度标记为完成")
	}
}

// TestParseLog_Deduplication 测试去重逻辑
func TestParseLog_Deduplication(t *testing.T) {
	// 同一个表出现多次，应该只记录第一次
	logContent := `[2026-04-16 10:00:00] 表 users 已存在，跳过创建
[2026-04-16 10:00:01] 分页同步表 users 完成，共处理 50 行数据
`
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "conversion.log")
	os.WriteFile(logFile, []byte(logContent), 0644)

	report, err := ParseLog(logFile)
	if err != nil {
		t.Fatalf("ParseLog 失败: %v", err)
	}

	// users 表应该只出现一次
	usersCount := 0
	for _, td := range report.TableDetails {
		if td.Name == "users" {
			usersCount++
		}
	}
	if usersCount != 1 {
		t.Errorf("期望 users 表出现 1 次，实际 %d 次", usersCount)
	}
}
