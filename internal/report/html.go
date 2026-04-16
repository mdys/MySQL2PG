package report

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// GenerateHTML 生成 HTML 报告文件
func GenerateHTML(r *ParsedReport, outputPath string) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建报告文件失败: %w", err)
	}
	defer f.Close()

	now := time.Now().Format("2006-01-02 15:04:05")

	// 计算各阶段最大耗时用于柱状图比例
	maxDur := 0.0
	for _, s := range r.StageStats {
		if s.Duration > maxDur {
			maxDur = s.Duration
		}
	}

	// 进度信息
	progressStr := ""
	progressPct := 0.0
	if r.ProgressTotal > 0 {
		progressPct = float64(r.ProgressCurrent) / float64(r.ProgressTotal) * 100
		if r.ProgressComplete {
			progressStr = fmt.Sprintf(`%.0f%% (%d/%d) 完成`, progressPct, r.ProgressCurrent, r.ProgressTotal)
		} else {
			progressStr = fmt.Sprintf(`%.0f%% (%d/%d) 进行中`, progressPct, r.ProgressCurrent, r.ProgressTotal)
		}
	}

	// 构建表详情渲染数据
	tableItems := make([]tableRenderItem, 0, len(r.TableDetails))
	var stageRows strings.Builder
	for _, s := range r.StageStats {
		barW := 0.0
		if maxDur > 0 {
			barW = s.Duration / maxDur * 100
		}
		stageRows.WriteString(fmt.Sprintf(`
        <div class="stage-row">
          <span class="stage-name">%s</span>
          <span class="stage-count">%d</span>
          <div class="bar-wrap"><div class="bar" style="width:%.1f%%"></div></div>
          <span class="stage-dur">%.2fs</span>
        </div>`, s.Name, s.ObjectCount, barW, s.Duration))
	}

	for _, td := range r.TableDetails {
		statusClass, statusText, extraInfo := renderTableStatus(td)
		rowCnt := "-"
		if td.RowKnown {
			rowCnt = formatNum(td.RowCount)
		}
		tableItems = append(tableItems, tableRenderItem{
			Name:        td.Name,
			Rows:        rowCnt,
			StatusClass: statusClass,
			StatusText:  statusText,
			ExtraInfo:   extraInfo,
			RowCount:    td.RowCount,
			RowKnown:    td.RowKnown,
		})
	}

	// 仅对有真实行数的表按行数倒序取前 20 张表
	topItems := make([]tableRenderItem, 0, len(tableItems))
	for _, item := range tableItems {
		if item.RowKnown {
			topItems = append(topItems, item)
		}
	}
	sort.SliceStable(topItems, func(i, j int) bool {
		if topItems[i].RowCount == topItems[j].RowCount {
			return topItems[i].Name < topItems[j].Name
		}
		return topItems[i].RowCount > topItems[j].RowCount
	})
	if len(topItems) > 20 {
		topItems = topItems[:20]
	}
	var topTableRows strings.Builder
	for i, item := range topItems {
		topTableRows.WriteString(fmt.Sprintf(`
        <tr>
          <td class="idx">%d</td>
          <td class="tname">%s</td>
          <td class="num trows">%s</td>
          <td><span class="badge %s">%s</span>%s</td>
        </tr>`, i+1, escapeHTML(item.Name), item.Rows, item.StatusClass, item.StatusText, item.ExtraInfo))
	}

	viewItems := buildObjectRenderItems(r.ViewDetails, r.ViewNames)
	functionItems := buildObjectRenderItems(r.FunctionDetails, r.FunctionNames)
	viewCount := len(viewItems)
	functionCount := len(functionItems)

	var inconsistentRows strings.Builder
	for _, inc := range r.Inconsistent {
		diff := inc.MySQLCnt - inc.PGCnt
		if diff < 0 {
			diff = -diff
		}
		inconsistentRows.WriteString(fmt.Sprintf(`
        <tr>
          <td class="tname">%s</td>
          <td class="num">%s</td>
          <td class="num">%s</td>
          <td class="num diff">-%d</td>
        </tr>`, inc.Name, formatNum(inc.MySQLCnt), formatNum(inc.PGCnt), diff))
	}

	var errorRows strings.Builder
	if len(r.Errors) > 0 {
		for i, e := range r.Errors {
			errorRows.WriteString(fmt.Sprintf(`
          <li><span class="error-num">#%d</span> %s</li>`, i+1, escapeHTML(e)))
		}
	} else {
		errorRows.WriteString(`<li class="no-error">无</li>`)
	}

	var warningRows strings.Builder
	if len(r.Warnings) > 0 {
		for i, w := range r.Warnings {
			warningRows.WriteString(fmt.Sprintf(`
          <li><span class="warn-num">#%d</span> %s</li>`, i+1, escapeHTML(w)))
		}
	}

	totalDurStr := formatDuration(r.TotalDuration)
	var avgSpeed string
	if r.TotalRows > 0 && r.TotalDuration > 0 {
		avgSpeed = fmt.Sprintf("%s 行/秒", formatNum(int64(float64(r.TotalRows)/r.TotalDuration)))
	}

	stageRowsStr := stageRows.String()
	topTableRowsStr := topTableRows.String()
	inconsistentRowsStr := inconsistentRows.String()
	warningRowsStr := warningRows.String()
	errorRowsStr := errorRows.String()
	tableItemsJSON := mustJSON(tableItems)
	viewItemsJSON := mustJSON(viewItems)
	functionItemsJSON := mustJSON(functionItems)

	tableHint := "支持搜索和分页，每页 50 条。"
	if len(tableItems) > 1000 {
		tableHint = "当前表数量超过 1000，默认分页展示并支持搜索。"
	}
	viewHint := "支持搜索和分页，每页 50 条。"
	if len(viewItems) > 500 {
		viewHint = "当前视图数量超过 500，默认分页展示并支持搜索。"
	}
	functionHint := "支持搜索和分页，每页 50 条。"
	if len(functionItems) > 500 {
		functionHint = "当前函数数量超过 500，默认分页展示并支持搜索。"
	}

	// 数据库版本行
	dbInfo := ""
	if r.MySQLVersion != "" || r.PGVersion != "" {
		dbInfo = fmt.Sprintf(`
      <div class="db-info">
        <span class="db-tag mysql">MySQL %s</span>
        <span class="arrow">→</span>
        <span class="db-tag pg">PostgreSQL %s</span>
      </div>`, r.MySQLVersion, r.PGVersion)
	}

	// 进度条
	progressBar := ""
	if r.ProgressTotal > 0 {
		barColor := "#10b981"
		if !r.ProgressComplete {
			barColor = "#f59e0b"
		}
		progressBar = fmt.Sprintf(`
      <div class="progress-bar-wrap">
        <div class="progress-bar" style="width:%.1f%%;background:%s"></div>
      </div>
      <div class="progress-label">%s</div>`, progressPct, barColor, progressStr)
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>MySQL2PG 迁移报告</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;700&family=DM+Sans:opsz,wght@9..40,400;9..40,500;9..40,700&display=swap" rel="stylesheet">
<style>
  :root {
    --bg: #0a0e17;
    --bg-elevated: #111827;
    --bg-card: #1a2235;
    --border: #2a3548;
    --text: #e2e8f0;
    --text-dim: #94a3b8;
    --text-muted: #64748b;
    --green: #10b981;
    --green-dim: rgba(16,185,129,.15);
    --red: #ef4444;
    --red-dim: rgba(239,68,68,.15);
    --amber: #f59e0b;
    --amber-dim: rgba(245,158,11,.15);
    --cyan: #06b6d4;
    --cyan-dim: rgba(6,182,212,.15);
    --blue: #3b82f6;
    --blue-dim: rgba(59,130,246,.15);
    --purple: #a855f7;
    --radius: 8px;
  }

  * { margin: 0; padding: 0; box-sizing: border-box; }

  body {
    font-family: 'DM Sans', -apple-system, BlinkMacSystemFont, sans-serif;
    background: var(--bg);
    color: var(--text);
    padding: 0;
    min-height: 100vh;
  }

  /* Scanline texture overlay */
  body::before {
    content: '';
    position: fixed;
    inset: 0;
    background: repeating-linear-gradient(
      0deg,
      transparent,
      transparent 2px,
      rgba(0,0,0,.03) 2px,
      rgba(0,0,0,.03) 4px
    );
    pointer-events: none;
    z-index: 1000;
  }

  .container {
    max-width: 1040px;
    margin: 0 auto;
    padding: 32px 24px;
  }

  /* ===== HEADER ===== */
  .header {
    background: var(--bg-elevated);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 28px 32px;
    margin-bottom: 24px;
    position: relative;
    overflow: hidden;
  }
  .header::before {
    content: '';
    position: absolute;
    top: 0; left: 0; right: 0;
    height: 3px;
    background: linear-gradient(90deg, var(--cyan), var(--blue), var(--purple));
  }
  .header-top {
    display: flex;
    align-items: baseline;
    gap: 12px;
    margin-bottom: 8px;
  }
  .header h1 {
    font-family: 'JetBrains Mono', monospace;
    font-size: 18px;
    font-weight: 700;
    color: var(--text);
    letter-spacing: -.02em;
  }
  .header h1::before {
    content: '>';
    color: var(--cyan);
    margin-right: 8px;
  }
  .header .timestamp {
    font-family: 'JetBrains Mono', monospace;
    font-size: 12px;
    color: var(--text-muted);
  }
  .header .log-source {
    font-family: 'JetBrains Mono', monospace;
    font-size: 12px;
    color: var(--text-muted);
    margin-top: 4px;
  }
  .db-info {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 16px;
  }
  .db-tag {
    font-family: 'JetBrains Mono', monospace;
    font-size: 11px;
    font-weight: 500;
    padding: 3px 10px;
    border-radius: 4px;
  }
  .db-tag.mysql {
    background: var(--amber-dim);
    color: var(--amber);
    border: 1px solid rgba(245,158,11,.25);
  }
  .db-tag.pg {
    background: var(--blue-dim);
    color: var(--blue);
    border: 1px solid rgba(59,130,246,.25);
  }
  .arrow {
    color: var(--text-muted);
    font-size: 14px;
  }

  /* Progress bar */
  .progress-bar-wrap {
    margin-top: 16px;
    height: 6px;
    background: var(--bg);
    border-radius: 3px;
    overflow: hidden;
  }
  .progress-bar {
    height: 100%%;
    border-radius: 3px;
    transition: width .3s ease;
  }
  .progress-label {
    font-family: 'JetBrains Mono', monospace;
    font-size: 11px;
    color: var(--text-muted);
    margin-top: 6px;
    text-align: right;
  }

  /* ===== STAT CARDS ===== */
  .stats {
    display: grid;
    grid-template-columns: repeat(6, 1fr);
    gap: 12px;
    margin-bottom: 24px;
  }
  @media (max-width: 768px) {
    .stats { grid-template-columns: repeat(3, 1fr); }
  }
  @media (max-width: 480px) {
    .stats { grid-template-columns: repeat(2, 1fr); }
  }
  .stat-card {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 16px;
    text-align: center;
    position: relative;
  }
  .stat-card::after {
    content: '';
    position: absolute;
    bottom: 0; left: 16px; right: 16px;
    height: 2px;
    border-radius: 1px;
  }
  .stat-card.green::after { background: var(--green); }
  .stat-card.cyan::after { background: var(--cyan); }
  .stat-card.amber::after { background: var(--amber); }
  .stat-card.red::after { background: var(--red); }
  .stat-card.blue::after { background: var(--blue); }
  .stat-card.purple::after { background: var(--purple); }

  .stat-label {
    font-size: 11px;
    font-weight: 500;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: .06em;
    margin-bottom: 6px;
  }
  .stat-value {
    font-family: 'JetBrains Mono', monospace;
    font-size: 24px;
    font-weight: 700;
  }
  .stat-value.green { color: var(--green); }
  .stat-value.cyan { color: var(--cyan); }
  .stat-value.amber { color: var(--amber); }
  .stat-value.red { color: var(--red); }
  .stat-value.blue { color: var(--blue); }
  .stat-value.purple { color: var(--purple); }

  /* ===== SECTIONS ===== */
  .section {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    margin-bottom: 16px;
    overflow: hidden;
  }
  .section-header {
    padding: 14px 20px;
    border-bottom: 1px solid var(--border);
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .section-header h2 {
    font-family: 'JetBrains Mono', monospace;
    font-size: 13px;
    font-weight: 500;
    color: var(--text);
    letter-spacing: .02em;
  }
  .section-header .tag {
    font-family: 'JetBrains Mono', monospace;
    font-size: 10px;
    padding: 2px 8px;
    border-radius: 3px;
    background: var(--bg);
    color: var(--text-muted);
    border: 1px solid var(--border);
  }
  .section-body {
    padding: 16px 20px;
  }

  /* ===== STAGE BARS ===== */
  .stage-row {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 10px;
    font-size: 13px;
  }
  .stage-row:last-child { margin-bottom: 0; }
  .stage-name {
    font-family: 'JetBrains Mono', monospace;
    font-size: 12px;
    color: var(--text);
    min-width: 110px;
    white-space: nowrap;
  }
  .stage-count {
    font-family: 'JetBrains Mono', monospace;
    font-size: 11px;
    color: var(--text-muted);
    min-width: 36px;
    text-align: right;
  }
  .bar-wrap {
    flex: 1;
    height: 6px;
    background: var(--bg);
    border-radius: 3px;
    overflow: hidden;
  }
  .bar {
    height: 100%%;
    background: linear-gradient(90deg, var(--cyan), var(--blue));
    border-radius: 3px;
    transition: width .3s ease;
  }
  .stage-dur {
    font-family: 'JetBrains Mono', monospace;
    font-size: 12px;
    font-weight: 500;
    color: var(--text);
    min-width: 52px;
    text-align: right;
  }
  .perf-footer {
    margin-top: 16px;
    padding-top: 12px;
    border-top: 1px solid var(--border);
    display: flex;
    justify-content: space-between;
    font-family: 'JetBrains Mono', monospace;
    font-size: 12px;
    color: var(--text-muted);
  }
  .perf-footer strong {
    color: var(--text);
    font-weight: 700;
  }

  /* ===== TABLES ===== */
  table {
    width: 100%%;
    border-collapse: collapse;
  }
  th {
    font-family: 'JetBrains Mono', monospace;
    font-size: 10px;
    font-weight: 500;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: .06em;
    padding: 10px 12px;
    text-align: left;
    border-bottom: 1px solid var(--border);
    background: var(--bg-elevated);
  }
  td {
    padding: 9px 12px;
    font-size: 13px;
    border-bottom: 1px solid var(--border);
    vertical-align: middle;
  }
  tr:last-child td { border-bottom: none; }
  tr:hover td {
    background: rgba(6,182,212,.04);
  }
  .idx {
    font-family: 'JetBrains Mono', monospace;
    font-size: 11px;
    color: var(--text-muted);
    width: 40px;
  }
  .tname {
    font-family: 'JetBrains Mono', monospace;
    font-size: 12px;
    color: var(--text);
  }
  .trows {
    font-family: 'JetBrains Mono', monospace;
    font-size: 12px;
    color: var(--text-dim);
  }
  .num {
    text-align: right;
    font-variant-numeric: tabular-nums;
  }

  /* ===== BADGES ===== */
  .badge {
    font-family: 'JetBrains Mono', monospace;
    font-size: 10px;
    font-weight: 500;
    padding: 2px 8px;
    border-radius: 3px;
    display: inline-block;
    letter-spacing: .02em;
  }
  .badge.ok {
    background: var(--green-dim);
    color: var(--green);
    border: 1px solid rgba(16,185,129,.25);
  }
  .badge.err {
    background: var(--red-dim);
    color: var(--red);
    border: 1px solid rgba(239,68,68,.25);
  }
  .badge.skip {
    background: var(--amber-dim);
    color: var(--amber);
    border: 1px solid rgba(245,158,11,.25);
  }
  .badge.neutral {
    background: rgba(148,163,184,.12);
    color: var(--text-dim);
    border: 1px solid rgba(148,163,184,.25);
  }

  /* Cell indicators */
  .cell-indicator {
    font-family: 'JetBrains Mono', monospace;
    font-size: 9px;
    font-weight: 700;
    padding: 1px 5px;
    border-radius: 2px;
    margin-left: 6px;
    vertical-align: middle;
  }
  .cell-indicator.err {
    background: var(--red-dim);
    color: var(--red);
  }
  .cell-indicator.warn {
    background: var(--amber-dim);
    color: var(--amber);
  }

  /* Diff highlight */
  .diff {
    color: var(--red);
    font-weight: 600;
  }

  /* ===== LISTS ===== */
  .error-list, .warn-list {
    list-style: none;
    padding: 0;
  }
  .error-list li {
    padding: 10px 14px;
    background: var(--red-dim);
    border-left: 3px solid var(--red);
    margin-bottom: 6px;
    font-size: 12px;
    font-family: 'JetBrains Mono', monospace;
    color: var(--text);
    word-break: break-all;
    border-radius: 0 4px 4px 0;
  }
  .error-list li.no-error {
    background: var(--green-dim);
    border-left-color: var(--green);
    color: var(--green);
  }
  .error-num {
    font-weight: 700;
    color: var(--red);
    margin-right: 8px;
  }
  .warn-list li {
    padding: 10px 14px;
    background: var(--amber-dim);
    border-left: 3px solid var(--amber);
    margin-bottom: 6px;
    font-size: 12px;
    font-family: 'JetBrains Mono', monospace;
    color: var(--text);
    word-break: break-all;
    border-radius: 0 4px 4px 0;
  }
  .warn-num {
    font-weight: 700;
    color: var(--amber);
    margin-right: 8px;
  }

  /* ===== LARGE OBJECT LIST ===== */
  .section-note {
    font-size: 12px;
    color: var(--text-muted);
    margin-bottom: 10px;
  }
  .list-toolbar {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 10px;
    margin-bottom: 12px;
  }
  .list-toolbar input {
    flex: 1;
    min-width: 220px;
    background: var(--bg);
    color: var(--text);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 8px 10px;
    font-size: 12px;
    font-family: 'JetBrains Mono', monospace;
  }
  .list-meta {
    font-size: 11px;
    color: var(--text-muted);
    font-family: 'JetBrains Mono', monospace;
  }
  .table-wrap {
    overflow-x: auto;
    border: 1px solid var(--border);
    border-radius: 6px;
  }
  .pager {
    margin-top: 10px;
    display: flex;
    gap: 8px;
    align-items: center;
  }
  .pager button {
    background: var(--bg);
    color: var(--text);
    border: 1px solid var(--border);
    border-radius: 4px;
    padding: 4px 10px;
    font-size: 12px;
    cursor: pointer;
  }
  .pager button[disabled] {
    opacity: .45;
    cursor: not-allowed;
  }
  .pager .pager-info {
    font-size: 11px;
    color: var(--text-muted);
    font-family: 'JetBrains Mono', monospace;
  }

  /* ===== INCONSISTENT TABLE ===== */
  .inconsistent-section .section-header h2 {
    color: var(--red);
  }

  /* ===== FOOTER ===== */
  .footer {
    text-align: center;
    padding: 24px 0 8px;
    font-family: 'JetBrains Mono', monospace;
    font-size: 11px;
    color: var(--text-muted);
    border-top: 1px solid var(--border);
    margin-top: 8px;
  }
  .footer span {
    color: var(--cyan);
  }
</style>
</head>
<body>
<div class="container">

  <!-- HEADER -->
  <div class="header">
    <div class="header-top">
      <h1>MySQL2PG 迁移报告</h1>
      <span class="timestamp">%s</span>
    </div>
    <div class="log-source">Source: %s</div>
    %s%s
  </div>

  <!-- STAT CARDS -->
  <div class="stats">
    <div class="stat-card green">
      <div class="stat-label">Tables</div>
      <div class="stat-value green">%d</div>
    </div>
    <div class="stat-card cyan">
      <div class="stat-label">Rows</div>
      <div class="stat-value cyan">%s</div>
    </div>
    <div class="stat-card blue">
      <div class="stat-label">Views</div>
      <div class="stat-value blue">%d</div>
    </div>
    <div class="stat-card purple">
      <div class="stat-label">Indexes</div>
      <div class="stat-value purple">%d</div>
    </div>
    <div class="stat-card amber">
      <div class="stat-label">Functions</div>
      <div class="stat-value amber">%d</div>
    </div>
    <div class="stat-card red">
      <div class="stat-label">Errors</div>
      <div class="stat-value red">%d</div>
    </div>
  </div>

  <!-- PERFORMANCE -->
  <div class="section">
    <div class="section-header">
      <h2>⚡ Performance</h2>
      <span class="tag">STAGES</span>
    </div>
    <div class="section-body">
      %s
      <div class="perf-footer">
        <span>Total: <strong>%s</strong></span>
        <span>%s</span>
      </div>
    </div>
  </div>

  <!-- TOP TABLES -->
  <div class="section">
    <div class="section-header">
      <h2>🏆 Top 20 Tables By Rows</h2>
      <span class="tag">%d items</span>
    </div>
    <div class="section-body" style="padding:0;overflow-x:auto;">
      <table>
        <thead>
          <tr>
            <th>#</th>
            <th>Table</th>
            <th style="text-align:right">Rows</th>
            <th>Status</th>
          </tr>
        </thead>
        <tbody>%s
        </tbody>
      </table>
    </div>
  </div>

  <!-- TABLE DETAILS -->
  <div class="section">
    <div class="section-header">
      <h2>📋 Tables</h2>
      <span class="tag">%d items</span>
    </div>
    <div class="section-body">
      <div class="section-note">%s</div>
      <div class="list-toolbar">
        <input id="tables-search" placeholder="搜索表名..." />
        <span class="list-meta" id="tables-meta"></span>
      </div>
      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>#</th>
              <th>Table</th>
              <th style="text-align:right">Rows</th>
              <th>Status</th>
            </tr>
          </thead>
          <tbody id="tables-body"></tbody>
        </table>
      </div>
      <div class="pager">
        <button id="tables-prev">Prev</button>
        <button id="tables-next">Next</button>
        <span class="pager-info" id="tables-page"></span>
      </div>
    </div>
  </div>

  <!-- VIEWS -->
  <div class="section">
    <div class="section-header">
      <h2>👁 Views</h2>
      <span class="tag">%d items</span>
    </div>
    <div class="section-body">
      <div class="section-note">%s</div>
      <div class="list-toolbar">
        <input id="views-search" placeholder="搜索视图名..." />
        <span class="list-meta" id="views-meta"></span>
      </div>
      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>#</th>
              <th>View Name</th>
              <th>Status</th>
            </tr>
          </thead>
          <tbody id="views-body"></tbody>
        </table>
      </div>
      <div class="pager">
        <button id="views-prev">Prev</button>
        <button id="views-next">Next</button>
        <span class="pager-info" id="views-page"></span>
      </div>
    </div>
  </div>

  <!-- FUNCTIONS -->
  <div class="section">
    <div class="section-header">
      <h2>🧠 Functions</h2>
      <span class="tag">%d items</span>
    </div>
    <div class="section-body">
      <div class="section-note">%s</div>
      <div class="list-toolbar">
        <input id="functions-search" placeholder="搜索函数名..." />
        <span class="list-meta" id="functions-meta"></span>
      </div>
      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>#</th>
              <th>Function Name</th>
              <th>Status</th>
            </tr>
          </thead>
          <tbody id="functions-body"></tbody>
        </table>
      </div>
      <div class="pager">
        <button id="functions-prev">Prev</button>
        <button id="functions-next">Next</button>
        <span class="pager-info" id="functions-page"></span>
      </div>
    </div>
  </div>`, now, r.LogFile, dbInfo, progressBar, r.TotalTables, formatNum(r.TotalRows), viewCount, r.TotalIndexes, functionCount, len(r.Errors), stageRowsStr, totalDurStr, avgSpeed, len(topItems), topTableRowsStr, len(r.TableDetails), tableHint, viewCount, viewHint, functionCount, functionHint)

	if len(r.Inconsistent) > 0 {
		html += fmt.Sprintf(`
  <!-- INCONSISTENT TABLES -->
  <div class="section inconsistent-section">
    <div class="section-header">
      <h2>⚠ Data Inconsistencies</h2>
      <span class="tag">%d tables</span>
    </div>
    <div class="section-body" style="padding:0;overflow-x:auto;">
      <table>
        <thead>
          <tr>
            <th>Table</th>
            <th style="text-align:right">MySQL</th>
            <th style="text-align:right">PostgreSQL</th>
            <th style="text-align:right">Delta</th>
          </tr>
        </thead>
        <tbody>%s
        </tbody>
      </table>
    </div>
  </div>`, len(r.Inconsistent), inconsistentRowsStr)
	}

	html += fmt.Sprintf(`
  <!-- ERRORS -->
  <div class="section">
    <div class="section-header">
      <h2>❌ Errors</h2>
      <span class="tag">%d</span>
    </div>
    <div class="section-body">
      <ul class="error-list">%s
      </ul>
    </div>
  </div>`, len(r.Errors), errorRowsStr)

	if len(r.Warnings) > 0 {
		html += fmt.Sprintf(`
  <!-- WARNINGS -->
  <div class="section">
    <div class="section-header">
      <h2>⚡ Warnings</h2>
      <span class="tag">%d</span>
    </div>
    <div class="section-body">
      <ul class="warn-list">%s
      </ul>
    </div>
  </div>`, len(r.Warnings), warningRowsStr)
	}

	html += fmt.Sprintf(`
  <script id="data-tables" type="application/json">%s</script>
  <script id="data-views" type="application/json">%s</script>
  <script id="data-functions" type="application/json">%s</script>
  <script>
    (function () {
      const PAGE_SIZE = 50;

      function parseData(id, fallback) {
        const el = document.getElementById(id);
        if (!el || !el.textContent) return fallback;
        try { return JSON.parse(el.textContent); } catch (e) { return fallback; }
      }
      function escapeHtml(v) {
        return String(v)
          .replace(/&/g, "&amp;")
          .replace(/</g, "&lt;")
          .replace(/>/g, "&gt;")
          .replace(/"/g, "&quot;");
      }

      function setupTablePager(config) {
        const input = document.getElementById(config.searchId);
        const body = document.getElementById(config.bodyId);
        const meta = document.getElementById(config.metaId);
        const prev = document.getElementById(config.prevId);
        const next = document.getElementById(config.nextId);
        const page = document.getElementById(config.pageId);
        const source = config.source || [];

        let keyword = "";
        let pageNo = 1;

        function filteredItems() {
          if (!keyword) return source;
          const k = keyword.toLowerCase();
          return source.filter(item => {
            const n = typeof item === "string" ? item : (item.name || "");
            return n.toLowerCase().includes(k);
          });
        }

        function render() {
          const items = filteredItems();
          const total = items.length;
          const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));
          if (pageNo > totalPages) pageNo = totalPages;
          const start = (pageNo - 1) * PAGE_SIZE;
          const list = items.slice(start, start + PAGE_SIZE);

          if (config.type === "table") {
            body.innerHTML = list.map((item, idx) => {
              const rowNo = start + idx + 1;
              const extra = item.extraInfo || "";
              const statusClass = item.statusClass || "neutral";
              const statusText = escapeHtml(item.statusText || "已完成");
              const name = escapeHtml(item.name || "-");
              const rows = escapeHtml(item.rows || "-");
              return '<tr>' +
                '<td class="idx">' + rowNo + '</td>' +
                '<td class="tname">' + name + '</td>' +
                '<td class="num trows">' + rows + '</td>' +
                '<td><span class="badge ' + statusClass + '">' + statusText + '</span>' + extra + '</td>' +
                '</tr>';
            }).join("");
          } else if (config.type === "object") {
            body.innerHTML = list.map((item, idx) => {
              const rowNo = start + idx + 1;
              const name = escapeHtml(item.name || "-");
              const statusClass = item.statusClass || "neutral";
              const statusText = escapeHtml(item.statusText || "未记录");
              return '<tr>' +
                '<td class="idx">' + rowNo + '</td>' +
                '<td class="tname">' + name + '</td>' +
                '<td><span class="badge ' + statusClass + '">' + statusText + '</span></td>' +
                '</tr>';
            }).join("");
          } else {
            body.innerHTML = list.map((name, idx) => {
              const rowNo = start + idx + 1;
              return '<tr>' +
                '<td class="idx">' + rowNo + '</td>' +
                '<td class="tname">' + (name || "-") + '</td>' +
                '</tr>';
            }).join("");
          }

          if (total === 0) {
            const col = config.type === "table" ? 4 : (config.type === "object" ? 3 : 2);
            body.innerHTML = '<tr><td colspan="' + col + '" style="text-align:center;color:#94a3b8;padding:14px;">无匹配项</td></tr>';
          }

          meta.textContent = "当前 " + total + " 条";
          page.textContent = "第 " + pageNo + "/" + totalPages + " 页";
          prev.disabled = pageNo <= 1;
          next.disabled = pageNo >= totalPages;
        }

        input.addEventListener("input", function () {
          keyword = this.value.trim();
          pageNo = 1;
          render();
        });
        prev.addEventListener("click", function () {
          if (pageNo > 1) {
            pageNo--;
            render();
          }
        });
        next.addEventListener("click", function () {
          const totalPages = Math.max(1, Math.ceil(filteredItems().length / PAGE_SIZE));
          if (pageNo < totalPages) {
            pageNo++;
            render();
          }
        });

        render();
      }

      setupTablePager({
        type: "table",
        source: parseData("data-tables", []),
        searchId: "tables-search",
        bodyId: "tables-body",
        metaId: "tables-meta",
        prevId: "tables-prev",
        nextId: "tables-next",
        pageId: "tables-page"
      });
      setupTablePager({
        type: "object",
        source: parseData("data-views", []),
        searchId: "views-search",
        bodyId: "views-body",
        metaId: "views-meta",
        prevId: "views-prev",
        nextId: "views-next",
        pageId: "views-page"
      });
      setupTablePager({
        type: "object",
        source: parseData("data-functions", []),
        searchId: "functions-search",
        bodyId: "functions-body",
        metaId: "functions-meta",
        prevId: "functions-prev",
        nextId: "functions-next",
        pageId: "functions-page"
      });
    })();
  </script>
  <!-- FOOTER -->
  <div class="footer">
    Generated by <span>mysql2pg report</span> · %s
  </div>
</div>
</body>
</html>`, tableItemsJSON, viewItemsJSON, functionItemsJSON, now)

	_, err = f.WriteString(html)
	return err
}

type tableRenderItem struct {
	Name        string `json:"name"`
	Rows        string `json:"rows"`
	StatusClass string `json:"statusClass"`
	StatusText  string `json:"statusText"`
	ExtraInfo   string `json:"extraInfo"`
	RowCount    int64  `json:"-"`
	RowKnown    bool   `json:"-"`
}

type objectRenderItem struct {
	Name        string `json:"name"`
	StatusClass string `json:"statusClass"`
	StatusText  string `json:"statusText"`
}

// renderTableStatus 生成表状态的样式和文本。
func renderTableStatus(td TableDetail) (string, string, string) {
	statusClass := ""
	statusText := ""
	switch td.Validation {
	case "数据一致":
		statusClass = "ok"
		statusText = "一致"
	case "数据不一致":
		statusClass = "err"
		statusText = "不一致"
	case "跳过验证":
		statusClass = "ok"
		statusText = "已完成"
	case "空表":
		statusClass = "neutral"
		statusText = "空表"
	case "已转换":
		statusClass = "ok"
		statusText = "成功"
	case "已存在":
		statusClass = "neutral"
		statusText = "已存在"
	default:
		statusClass = "neutral"
		statusText = "未记录"
	}

	extraInfo := ""
	if td.HasError {
		extraInfo = `<span class="cell-indicator err" title="` + escapeHTML(td.ErrorMsg) + `">ERR</span>`
	} else if td.Warning != "" {
		extraInfo = `<span class="cell-indicator warn" title="` + escapeHTML(td.Warning) + `">WRN</span>`
	}
	return statusClass, statusText, extraInfo
}

// buildObjectRenderItems 构建对象渲染列表并附带同步状态。
func buildObjectRenderItems(details []ObjectDetail, fallbackNames []string) []objectRenderItem {
	items := make([]objectRenderItem, 0, len(details))
	if len(details) == 0 {
		names := dedupSortedNames(fallbackNames)
		for _, n := range names {
			items = append(items, objectRenderItem{
				Name:        n,
				StatusClass: "neutral",
				StatusText:  "未记录",
			})
		}
		return items
	}
	for _, d := range details {
		statusClass := "ok"
		statusText := "已完成"
		if d.Status == "失败" {
			statusClass = "err"
			statusText = "失败"
		}
		items = append(items, objectRenderItem{
			Name:        d.Name,
			StatusClass: statusClass,
			StatusText:  statusText,
		})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].StatusClass == items[j].StatusClass {
			return items[i].Name < items[j].Name
		}
		return objectStatusPriority(items[i].StatusClass) < objectStatusPriority(items[j].StatusClass)
	})
	return items
}

// objectStatusPriority 定义对象状态排序优先级（失败优先显示）。
func objectStatusPriority(statusClass string) int {
	switch statusClass {
	case "err":
		return 0
	case "ok":
		return 1
	default:
		return 2
	}
}

// dedupSortedNames 对对象名去重并按字典序排序。
func dedupSortedNames(names []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(names))
	for _, n := range names {
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		result = append(result, n)
	}
	sort.Strings(result)
	return result
}

// mustJSON 将对象序列化为 JSON 字符串，失败时返回空数组。
func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// formatNum 将数字格式化为千分位字符串。
func formatNum(n int64) string {
	s := fmt.Sprintf("%d", n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

// formatDuration 将秒数格式化为可读时长。
func formatDuration(seconds float64) string {
	if seconds < 60 {
		return fmt.Sprintf("%.1fs", seconds)
	}
	if seconds < 3600 {
		m := int(seconds) / 60
		s := int(seconds) % 60
		return fmt.Sprintf("%dm %ds", m, s)
	}
	h := int(seconds) / 3600
	m := (int(seconds) % 3600) / 60
	s := int(seconds) % 60
	return fmt.Sprintf("%dh %dm %ds", h, m, s)
}

// escapeHTML 对 HTML 特殊字符做转义。
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}
