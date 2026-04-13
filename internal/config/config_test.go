package config

import (
	"testing"
)

func TestConvertExclusionLists_ViewList(t *testing.T) {
	// 测试视图排除列表转换
	config := &ConversionConfig{
		Options: OptionsConfig{
			SkipViewList: []string{"View1", "VIEW2", "view3"},
		},
	}

	config.convertExclusionLists()

	// 验证集合已正确转换
	expected := []string{"view1", "view2", "view3"}
	for _, key := range expected {
		if _, exists := config.Options.SkipViewSet[key]; !exists {
			t.Errorf("SkipViewSet missing key %q, got: %v", key, config.Options.SkipViewSet)
		}
	}

	// 验证大小
	if len(config.Options.SkipViewSet) != 3 {
		t.Errorf("Expected 3 elements, got %d", len(config.Options.SkipViewSet))
	}
}

func TestConvertExclusionLists_FunctionList(t *testing.T) {
	// 测试函数排除列表转换
	config := &ConversionConfig{
		Options: OptionsConfig{
			SkipFunctionList: []string{"Func1", "FUNC2", "func3"},
		},
	}

	config.convertExclusionLists()

	// 验证集合已正确转换
	expected := []string{"func1", "func2", "func3"}
	for _, key := range expected {
		if _, exists := config.Options.SkipFunctionSet[key]; !exists {
			t.Errorf("SkipFunctionSet missing key %q, got: %v", key, config.Options.SkipFunctionSet)
		}
	}

	// 验证大小
	if len(config.Options.SkipFunctionSet) != 3 {
		t.Errorf("Expected 3 elements, got %d", len(config.Options.SkipFunctionSet))
	}
}

func TestConvertExclusionLists_Empty(t *testing.T) {
	// 测试空列表转换
	config := &ConversionConfig{
		Options: OptionsConfig{
			SkipViewList:     []string{},
			SkipFunctionList: nil,
		},
	}

	config.convertExclusionLists()

	// 验证集合仍为 nil
	if config.Options.SkipViewSet != nil {
		t.Errorf("Expected nil SkipViewSet, got %v", config.Options.SkipViewSet)
	}
	if config.Options.SkipFunctionSet != nil {
		t.Errorf("Expected nil SkipFunctionSet, got %v", config.Options.SkipFunctionSet)
	}
}

func TestConvertExclusionLists_Duplicates(t *testing.T) {
	// 测试重复元素（应该自动去重）
	config := &ConversionConfig{
		Options: OptionsConfig{
			SkipViewList: []string{"view1", "VIEW1", "View1"},
		},
	}

	config.convertExclusionLists()

	// 验证只有 1 个元素（去重）
	if len(config.Options.SkipViewSet) != 1 {
		t.Errorf("Expected 1 element (deduplicated), got %d", len(config.Options.SkipViewSet))
	}

	if _, exists := config.Options.SkipViewSet["view1"]; !exists {
		t.Errorf("SkipViewSet missing key 'view1'")
	}
}
