package report

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// ParsedReport 从日志解析出的报告数据
type ParsedReport struct {
	LogFile         string
	ErrorFile       string
	MySQLVersion    string
	PGVersion       string
	StageStats      []StageStat
	TotalDuration   float64
	TableDetails    []TableDetail
	Inconsistent    []InconsistentTable
	Warnings        []string
	Errors          []string
	TotalTables     int
	TotalRows       int64
	TotalViews      int
	TotalIndexes    int
	TotalFunctions  int
	TotalUsers      int
	TotalPrivileges int
	// 进度信息（用于标识迁移是否完成）
	ProgressCurrent  int
	ProgressTotal    int
	ProgressComplete bool
}

// StageStat 阶段统计
type StageStat struct {
	Name        string
	ObjectCount int
	Duration    float64
}

// TableDetail 表同步详情
type TableDetail struct {
	Name       string
	RowCount   int64
	Validation string // "数据一致"|"数据不一致"|"跳过验证"|"空表"|"已转换"|"已存在"
	HasError   bool
	ErrorMsg   string
	Warning    string // 警告信息，如"没有主键"
}

// Warning 警告信息
type Warning struct {
	Table   string
	Message string
}

// InconsistentTable 数据不一致表
type InconsistentTable struct {
	Name     string
	MySQLCnt int64
	PGCnt    int64
}

// ParseLog 解析 conversion.log 生成报告数据
func ParseLog(logPath string) (*ParsedReport, error) {
	f, err := os.Open(logPath)
	if err != nil {
		return nil, fmt.Errorf("打开日志文件失败: %w", err)
	}
	defer f.Close()

	r := &ParsedReport{LogFile: logPath}

	// 预编译正则 — 适配实际日志格式
	// [2026-04-07 10:23:55] 转换表 CASE_39_UNDERSCORE 成功
	reTableSuccess := regexp.MustCompile(`\[[\d\-\s:]+\]\s*转换表\s+(\S+)\s+成功`)
	// [2026-04-07 11:27:33] 表 case_41_parent 已存在，跳过创建
	reTableExists := regexp.MustCompile(`\[[\d\-\s:]+\]\s*表\s+(\S+)\s+已存在，跳过创建`)
	// [2026-04-07 11:27:37] 表 case_35_enum_charset 同步完成，0 行数据，数据一致
	reTableSyncDone := regexp.MustCompile(`\[[\d\-\s:]+\]\s*表\s+(\S+)\s+同步完成，(\d+)\s+行数据，(数据一致|数据不一致|跳过验证)`)
	// [2026-04-07 11:27:37] 分页同步表 act_hi_comment 完成，共处理 10 行数据
	rePaginatedSync := regexp.MustCompile(`\[[\d\-\s:]+\]\s*分页同步表\s+(\S+)\s+完成，共处理\s+(\d+)\s+行数据`)
	// [2026-04-07 11:27:37] 分页同步表 act_hi_comment 完成，10 行数据，数据不一致
	rePaginatedSyncWithStatus := regexp.MustCompile(`\[[\d\-\s:]+\]\s*分页同步表\s+(\S+)\s+完成，(\d+)\s+行数据，(数据一致|数据不一致|跳过验证)`)
	// [2026-04-07 10:24:00] 进度: 100.00% (192/192)
	reProgress := regexp.MustCompile(`\[[\d\-\s:]+\]\s*进度:\s*[\d.]+%\s*\((\d+)/(\d+)\)`)
	// [2026-04-07 10:23:53] 表MySQL 的DDL、数据、view、索引、函数、用户和权限的转换到 PostgreSQL ...
	reVersionLine := regexp.MustCompile(`\[[\d\-\s:]+\]\s*(表\w+)\s*的DDL、数据、view、索引、函数、用户和权限的转换到\s*PostgreSQL`)
	// MySQL | 8.0.x  或  PostgreSQL | 15.x
	reMySQLVersion := regexp.MustCompile(`MySQL\s*\|\s*(.+)`)
	rePGVersion := regexp.MustCompile(`PostgreSQL\s*\|\s*(.+)`)
	// 阶段汇总表格: | 表结构 | 192 | 5.2 |
	reStageSummary := regexp.MustCompile(`\|\s*(.+?)\s*\|\s*(\d+)\s*\|\s*([\d.]+)\s*\|`)
	reTotalDuration := regexp.MustCompile(`\|\s*总耗时\s*\|\s*\|\s*([\d.]+)\s*\|`)
	// 数据量校验不一致: | 表名 | 100 | 99 |
	reInconsistent := regexp.MustCompile(`\|\s*(\S+)\s*\|\s*(\d+)\s*\|\s*(\d+)\s*\|`)
	// 空表跳过
	reEmptyTable := regexp.MustCompile(`\[[\d\-\s:]+\]\s*表\s+(\S+)\s+没有数据，跳过同步`)
	// 数据不一致行
	reDataInconsistent := regexp.MustCompile(`\[[\d\-\s:]+\]\s*表\s+(\S+)\s+数据不一致`)
	// [2026-04-07 11:27:37] 警告: 表 case_01_integers 没有主键，将使用传统的OFFSET分页
	reWarning := regexp.MustCompile(`\[[\d\-\s:]+\]\s*警告:\s*表\s+(\S+)\s+(.+)`)
	// [2026-04-07 11:27:37] 表 case_155_rest_dishes 的主键是 dish_id，将使用基于主键的分页
	rePrimaryKey := regexp.MustCompile(`\[[\d\-\s:]+\]\s*表\s+(\S+)\s+的主键是\s+\S+，将使用基于主键的分页`)
	// [2026-04-07 11:27:37] 插入表 case_01_integers 数据失败: ...
	reTableError := regexp.MustCompile(`\[[\d\-\s:]+\]\s*(插入表|查询表|创建表|更新表)\s+(\S+)\s+(?:数据|结构|索引|权限).*?失败[::]\s*(.+)`)
	// 转换完成
	reConversionDone := regexp.MustCompile(`\[[\d\-\s:]+\]\s*转换完成`)

	// 去重集合 — 防止日志重复写入
	seenTables := make(map[string]bool)
	seenInconsistent := make(map[string]bool)
	seenStages := make(map[string]bool)

	scanner := bufio.NewScanner(f)
	// 增大 buffer 以容纳长行
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	inSummaryTable := false
	inInconsistentTable := false

	for scanner.Scan() {
		line := scanner.Text()

		// 提取版本信息（独立行或表格行）
		if m := reMySQLVersion.FindStringSubmatch(line); len(m) > 1 {
			v := strings.TrimSpace(m[1])
			if v != "" && !strings.HasPrefix(v, "|") {
				r.MySQLVersion = v
			}
		}
		if m := rePGVersion.FindStringSubmatch(line); len(m) > 1 {
			v := strings.TrimSpace(m[1])
			if v != "" && !strings.HasPrefix(v, "|") {
				r.PGVersion = v
			}
		}

		// 检测阶段汇总表格开始
		if strings.Contains(line, "各阶段及耗时汇总如下") {
			inSummaryTable = true
			continue
		}

		// 检测不一致表格开始
		if strings.Contains(line, "数据量校验不一致的表统计") {
			inInconsistentTable = true
			continue
		}

		// 解析阶段汇总表格
		if inSummaryTable {
			if m := reTotalDuration.FindStringSubmatch(line); len(m) > 1 {
				r.TotalDuration, _ = strconv.ParseFloat(m[1], 64)
				inSummaryTable = false
				continue
			}
			if m := reStageSummary.FindStringSubmatch(line); len(m) > 1 {
				name := strings.TrimSpace(m[1])
				// 跳过"总耗时"行
				if strings.Contains(name, "总耗时") {
					continue
				}
				// 跳过表头行
				if strings.Contains(name, "阶段") {
					continue
				}
				// 去重：同名阶段只保留第一次
				if seenStages[name] {
					continue
				}
				seenStages[name] = true

				count, _ := strconv.Atoi(strings.TrimSpace(m[2]))
				dur, _ := strconv.ParseFloat(strings.TrimSpace(m[3]), 64)

				stat := StageStat{Name: name, ObjectCount: count, Duration: dur}
				r.StageStats = append(r.StageStats, stat)

				switch {
				case strings.Contains(name, "表结构"):
					r.TotalTables = count
				case strings.Contains(name, "视图"):
					r.TotalViews = count
				case strings.Contains(name, "索引"):
					r.TotalIndexes = count
				case strings.Contains(name, "函数"):
					r.TotalFunctions = count
				case strings.Contains(name, "用户"):
					r.TotalUsers = count
				case strings.Contains(name, "权限"):
					r.TotalPrivileges = count
				}
			}
			continue
		}

		// 解析不一致表格
		if inInconsistentTable {
			// 检测表格结束
			if strings.HasPrefix(line, "+-") && len(r.Inconsistent) > 0 {
				inInconsistentTable = false
				continue
			}
			if m := reInconsistent.FindStringSubmatch(line); len(m) > 1 {
				name := strings.TrimSpace(m[1])
				// 跳过表头行
				if name == "表名" || strings.Contains(name, "表名") || strings.Contains(name, "数据量校验") {
					continue
				}
				// 去重：同表名只保留第一次
				if !seenInconsistent[name] {
					seenInconsistent[name] = true
					mysqlCnt, _ := strconv.ParseInt(strings.TrimSpace(m[2]), 10, 64)
					pgCnt, _ := strconv.ParseInt(strings.TrimSpace(m[3]), 10, 64)
					r.Inconsistent = append(r.Inconsistent, InconsistentTable{
						Name: name, MySQLCnt: mysqlCnt, PGCnt: pgCnt,
					})
				}
			}
			continue
		}

		// 解析转换成功的表（去重）
		if m := reTableSuccess.FindStringSubmatch(line); len(m) > 1 {
			tableName := m[1]
			if !seenTables[tableName] {
				seenTables[tableName] = true
				r.TableDetails = append(r.TableDetails, TableDetail{
					Name: tableName, RowCount: 0, Validation: "已转换",
				})
				r.TotalTables++
			}
		}

		// 解析"表已存在，跳过创建"（去重）
		if m := reTableExists.FindStringSubmatch(line); len(m) > 1 {
			tableName := m[1]
			if !seenTables[tableName] {
				seenTables[tableName] = true
				r.TableDetails = append(r.TableDetails, TableDetail{
					Name: tableName, RowCount: 0, Validation: "已存在",
				})
				r.TotalTables++
			}
		}

		// 解析"表同步完成"（去重）
		if m := reTableSyncDone.FindStringSubmatch(line); len(m) > 1 {
			tableName := m[1]
			rowCount, _ := strconv.ParseInt(m[2], 10, 64)
			validation := m[3]
			if !seenTables[tableName] {
				seenTables[tableName] = true
				r.TableDetails = append(r.TableDetails, TableDetail{
					Name: tableName, RowCount: rowCount, Validation: validation,
				})
				r.TotalTables++
			}
			r.TotalRows += rowCount
		}

		// 解析"分页同步表 完成，共处理 N 行数据"（去重）
		if m := rePaginatedSync.FindStringSubmatch(line); len(m) > 1 {
			tableName := m[1]
			rowCount, _ := strconv.ParseInt(m[2], 10, 64)
			if !seenTables[tableName] {
				seenTables[tableName] = true
				r.TableDetails = append(r.TableDetails, TableDetail{
					Name: tableName, RowCount: rowCount, Validation: "跳过验证",
				})
				r.TotalTables++
			}
			r.TotalRows += rowCount
		}

		// 解析"分页同步表 完成，N 行数据，数据一致/不一致"（去重）
		if m := rePaginatedSyncWithStatus.FindStringSubmatch(line); len(m) > 1 {
			tableName := m[1]
			rowCount, _ := strconv.ParseInt(m[2], 10, 64)
			validation := m[3]
			if !seenTables[tableName] {
				seenTables[tableName] = true
				r.TableDetails = append(r.TableDetails, TableDetail{
					Name: tableName, RowCount: rowCount, Validation: validation,
				})
				r.TotalTables++
			}
			r.TotalRows += rowCount
		}

		// 解析进度行 — 记录当前进度，用于标识迁移是否完成
		if m := reProgress.FindStringSubmatch(line); len(m) > 1 {
			r.ProgressCurrent, _ = strconv.Atoi(m[1])
			r.ProgressTotal, _ = strconv.Atoi(m[2])
			r.ProgressComplete = (r.ProgressCurrent >= r.ProgressTotal)
		}

		// 解析空表（去重）
		if m := reEmptyTable.FindStringSubmatch(line); len(m) > 1 {
			tableName := m[1]
			if !seenTables[tableName] {
				seenTables[tableName] = true
				r.TableDetails = append(r.TableDetails, TableDetail{
					Name: tableName, RowCount: 0, Validation: "空表",
				})
			}
		}

		// 解析数据不一致（去重）
		if m := reDataInconsistent.FindStringSubmatch(line); len(m) > 1 {
			tableName := m[1]
			if !seenTables[tableName] {
				seenTables[tableName] = true
				r.TableDetails = append(r.TableDetails, TableDetail{
					Name: tableName, RowCount: 0, Validation: "数据不一致",
				})
			}
		}

		// 检测版本信息行
		if reVersionLine.MatchString(line) {
			// 这行只记录数据库名，版本需要从其他位置获取
		}

		// 解析警告信息
		if m := reWarning.FindStringSubmatch(line); len(m) > 1 {
			tableName := m[1]
			warnMsg := strings.TrimSpace(m[2])
			r.Warnings = append(r.Warnings, fmt.Sprintf("表 %s: %s", tableName, warnMsg))
			// 同时关联到表详情
			if td, ok := findTableDetail(r.TableDetails, tableName); ok {
				td.Warning = warnMsg
			}
		}

		// 解析主键信息（用于判断分页方式）
		if rePrimaryKey.MatchString(line) {
			// 仅记录，不单独建表详情（表详情已由其他模式创建）
		}

		// 解析表级错误（conversion.log 中的错误，关联到表详情）
		if m := reTableError.FindStringSubmatch(line); len(m) > 1 {
			tableName := m[2]
			errMsg := strings.TrimSpace(m[3])
			if td, ok := findTableDetail(r.TableDetails, tableName); ok {
				td.HasError = true
				td.ErrorMsg = errMsg
			}
			// 同时加入错误列表
			r.Errors = append(r.Errors, fmt.Sprintf("表 %s %s失败: %s", tableName, m[1], errMsg))
		}

		// 检测转换完成
		if reConversionDone.MatchString(line) {
			r.ProgressComplete = true
			r.ProgressCurrent = r.ProgressTotal
		}
	}

	// 汇总总行数（如果 stage stats 里有表数据行数，且前面没有从表详情累加过）
	if r.TotalRows == 0 {
		for _, s := range r.StageStats {
			if strings.Contains(s.Name, "表数据") {
				r.TotalRows = int64(s.ObjectCount)
			}
		}
	}

	// TotalTables 以 TableDetails 实际解析到的数量为准
	r.TotalTables = len(r.TableDetails)

	return r, nil
}

// ParseErrors 解析 errors.log 收集错误信息
func ParseErrors(report *ParsedReport, errorPath string) error {
	f, err := os.Open(errorPath)
	if err != nil {
		// 错误日志不存在不算严重问题
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("打开错误日志失败: %w", err)
	}
	defer f.Close()

	report.ErrorFile = errorPath

	reError := regexp.MustCompile(`\[.*?\]\s*ERROR:\s*(.+)`)
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	// 去重集合 — 相同错误信息只保留一次
	seenErrors := make(map[string]bool)

	for scanner.Scan() {
		line := scanner.Text()
		if m := reError.FindStringSubmatch(line); len(m) > 1 {
			errMsg := strings.TrimSpace(m[1])
			if !seenErrors[errMsg] {
				seenErrors[errMsg] = true
				report.Errors = append(report.Errors, errMsg)
			}
		}
	}

	return nil
}

// findTableDetail 在表详情列表中查找指定表
func findTableDetail(details []TableDetail, name string) (*TableDetail, bool) {
	for i := range details {
		if details[i].Name == name {
			return &details[i], true
		}
	}
	return nil, false
}
