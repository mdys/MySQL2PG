package config

import (
	"testing"
)

func TestStringSet_UnmarshalYAML_List(t *testing.T) {
	// 测试列表形式解析
	var s StringSet
	listData := []byte(`["View1", "VIEW2", "view3"]`)

	// 手动调用 UnmarshalJSON 测试（模拟 Viper 行为）
	err := s.UnmarshalJSON(listData)
	if err != nil {
		t.Fatalf("UnmarshalJSON failed for list: %v", err)
	}

	// 验证集合包含所有元素（转换为小写）
	expected := []string{"view1", "view2", "view3"}
	for _, key := range expected {
		if _, exists := s[key]; !exists {
			t.Errorf("StringSet missing key %q, got: %v", key, s)
		}
	}

	// 验证大小
	if len(s) != 3 {
		t.Errorf("Expected 3 elements, got %d", len(s))
	}
}

func TestStringSet_UnmarshalYAML_Map(t *testing.T) {
	// 测试 map 形式解析
	var s StringSet
	mapData := []byte(`{"View1": {}, "VIEW2": {}, "view3": {}}`)

	err := s.UnmarshalJSON(mapData)
	if err != nil {
		t.Fatalf("UnmarshalJSON failed for map: %v", err)
	}

	// 验证集合包含所有元素（转换为小写）
	expected := []string{"view1", "view2", "view3"}
	for _, key := range expected {
		if _, exists := s[key]; !exists {
			t.Errorf("StringSet missing key %q, got: %v", key, s)
		}
	}

	// 验证大小
	if len(s) != 3 {
		t.Errorf("Expected 3 elements, got %d", len(s))
	}
}

func TestStringSet_CaseInsensitive(t *testing.T) {
	// 测试大小写不敏感
	var s StringSet
	data := []byte(`["MixedCase", "UPPERCASE", "lowercase"]`)

	err := s.UnmarshalJSON(data)
	if err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	// 验证所有键都转换为小写
	expected := []string{"mixedcase", "uppercase", "lowercase"}
	for _, key := range expected {
		if _, exists := s[key]; !exists {
			t.Errorf("StringSet missing lowercase key %q, got: %v", key, s)
		}
	}
}

func TestStringSet_Empty(t *testing.T) {
	// 测试空列表
	var s StringSet
	data := []byte(`[]`)

	err := s.UnmarshalJSON(data)
	if err != nil {
		t.Fatalf("UnmarshalJSON failed for empty list: %v", err)
	}

	if len(s) != 0 {
		t.Errorf("Expected 0 elements, got %d", len(s))
	}
}

func TestStringSet_Duplicates(t *testing.T) {
	// 测试重复元素（应该自动去重）
	var s StringSet
	data := []byte(`["view1", "VIEW1", "View1"]`)

	err := s.UnmarshalJSON(data)
	if err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	// 验证只有 1 个元素（去重）
	if len(s) != 1 {
		t.Errorf("Expected 1 element (deduplicated), got %d", len(s))
	}

	if _, exists := s["view1"]; !exists {
		t.Errorf("StringSet missing key 'view1'")
	}
}
