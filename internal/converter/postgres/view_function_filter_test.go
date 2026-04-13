package postgres

import (
	"testing"

	"github.com/yourusername/mysql2pg/internal/config"
)

func TestShouldSkipView_WithSet(t *testing.T) {
	// 使用集合形式
	excludeSet := config.StringSet{
		"view1":          {},
		"complex_report": {},
		"temp_stats":     {},
	}

	tests := []struct {
		name     string
		viewName string
		set      config.StringSet
		want     bool
	}{
		{"精确匹配", "view1", excludeSet, true},
		{"大小写不敏感", "VIEW1", excludeSet, true},
		{"混合大小写", "Complex_Report", excludeSet, true},
		{"不匹配", "view2", excludeSet, false},
		{"nil 集合", "view1", nil, false},
		{"空集合", "view1", config.StringSet{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldSkipView(tt.viewName, tt.set); got != tt.want {
				t.Errorf("shouldSkipView(%q) = %v, want %v", tt.viewName, got, tt.want)
			}
		})
	}
}

func TestShouldSkipFunction_WithSet(t *testing.T) {
	excludeSet := config.StringSet{
		"calc_total":    {},
		"get_user_info": {},
	}

	tests := []struct {
		name     string
		funcName string
		set      config.StringSet
		want     bool
	}{
		{"精确匹配", "calc_total", excludeSet, true},
		{"大小写不敏感", "CALC_TOTAL", excludeSet, true},
		{"混合大小写", "Get_User_Info", excludeSet, true},
		{"不匹配", "other_func", excludeSet, false},
		{"nil 集合", "calc_total", nil, false},
		{"空集合", "calc_total", config.StringSet{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldSkipFunction(tt.funcName, tt.set); got != tt.want {
				t.Errorf("shouldSkipFunction(%q) = %v, want %v", tt.funcName, got, tt.want)
			}
		})
	}
}

func TestShouldSkipView_Performance(t *testing.T) {
	// 测试大规模集合的查找性能
	largeSet := make(config.StringSet, 1000)
	for i := 0; i < 1000; i++ {
		largeSet[string(rune(i))] = struct{}{}
	}

	// 执行 10000 次查找
	for i := 0; i < 10000; i++ {
		_ = shouldSkipView(string(rune(i%1000)), largeSet)
	}

	// 应该能快速完成（O(1) 查找）
	t.Log("Large set lookup test completed successfully")
}

func TestShouldSkipFunction_Performance(t *testing.T) {
	// 测试大规模集合的查找性能
	largeSet := make(config.StringSet, 1000)
	for i := 0; i < 1000; i++ {
		largeSet[string(rune(i))] = struct{}{}
	}

	// 执行 10000 次查找
	for i := 0; i < 10000; i++ {
		_ = shouldSkipFunction(string(rune(i%1000)), largeSet)
	}

	// 应该能快速完成（O(1) 查找）
	t.Log("Large set lookup test completed successfully")
}
