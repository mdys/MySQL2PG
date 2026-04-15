package mpp

import (
	"regexp"
	"strings"
)

// ParseSearchPath 从 pg_connection_params 中解析 search_path
// 例如: "search_path=myschema connect_timeout=300" → "myschema"
// 支持格式: search_path=myschema 或 search_path=myschema,public 或 search_path=$user,public
// 如果未指定 search_path 或仅包含 $user，返回 "public" 作为默认值
//
// 注意：连接参数使用空格分隔（非 URL 格式），不需要 URL decode 处理。
// 如果未来改为 URL 格式（?search_path=xxx&...），需要添加 url.QueryUnescape。
func ParseSearchPath(pgConnectionParams string) string {
	if pgConnectionParams == "" {
		return "public"
	}

	// 使用正则表达式匹配 search_path=xxx
	// 支持格式: search_path=myschema 或 search_path=myschema,public
	re := regexp.MustCompile(`(?i)search_path=([^&\s]+)`)
	matches := re.FindStringSubmatch(pgConnectionParams)

	if len(matches) > 1 {
		// 解析多个 schema（逗号分隔）
		searchPath := matches[1]
		schemas := strings.Split(searchPath, ",")
		for _, s := range schemas {
			s = strings.TrimSpace(s)
			// 跳过 $user 特殊变量（解析为当前用户名，不是实际 schema）
			if s == "$user" {
				continue
			}
			return s
		}
	}

	return "public"
}
