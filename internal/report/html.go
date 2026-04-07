package report

import (
	"fmt"
	"os"
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
	consistentCount := len(r.TableDetails) - len(r.Inconsistent)
	for _, td := range r.TableDetails {
		if td.Validation == "空表" {
			consistentCount--
		}
	}

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

	var tableRows strings.Builder
	for i, td := range r.TableDetails {
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
			statusClass = "skip"
			statusText = "跳过"
		case "空表":
			statusClass = "skip"
			statusText = "空表"
		case "已转换":
			statusClass = "ok"
			statusText = "成功"
		case "已存在":
			statusClass = "skip"
			statusText = "已存在"
		default:
			statusClass = "skip"
			statusText = "完成"
		}

		extraInfo := ""
		if td.HasError {
			extraInfo = `<span class="cell-indicator err" title="` + escapeHTML(td.ErrorMsg) + `">ERR</span>`
		} else if td.Warning != "" {
			extraInfo = `<span class="cell-indicator warn" title="` + escapeHTML(td.Warning) + `">WRN</span>`
		}

		rowCnt := "-"
		if td.RowCount > 0 {
			rowCnt = formatNum(td.RowCount)
		}
		tableRows.WriteString(fmt.Sprintf(`
        <tr>
          <td class="idx">%d</td>
          <td class="tname">%s</td>
          <td class="num trows">%s</td>
          <td><span class="badge %s">%s</span>%s</td>
        </tr>`, i+1, td.Name, rowCnt, statusClass, statusText, extraInfo))
	}

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
	tableRowsStr := tableRows.String()
	inconsistentRowsStr := inconsistentRows.String()
	warningRowsStr := warningRows.String()
	errorRowsStr := errorRows.String()

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

  <!-- TABLE DETAILS -->
  <div class="section">
    <div class="section-header">
      <h2>📋 Tables</h2>
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
  </div>`, now, r.LogFile, dbInfo, progressBar, r.TotalTables, formatNum(r.TotalRows), r.TotalViews, r.TotalIndexes, r.TotalFunctions, len(r.Errors), stageRowsStr, totalDurStr, avgSpeed, len(r.TableDetails), tableRowsStr)

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
  <!-- FOOTER -->
  <div class="footer">
    Generated by <span>mysql2pg report</span> · %s
  </div>
</div>
</body>
</html>`, now)

	_, err = f.WriteString(html)
	return err
}

func formatNum(n int64) string {
	s := fmt.Sprintf("%d", n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

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

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}
