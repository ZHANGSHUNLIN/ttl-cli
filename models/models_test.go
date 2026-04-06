package models

import (
	"encoding/json"
	"testing"
)

func TestValJsonKey_JsonSerialization(t *testing.T) {
	testCases := []struct {
		name string
		key  ValJsonKey
	}{
		{"原始资源", ValJsonKey{Key: "test", Type: ORIGIN}},
		{"标签资源", ValJsonKey{Key: "tag", Type: TAG, OriginKey: "origin"}},
		{"特殊字符", ValJsonKey{Key: "test@key.com", Type: ORIGIN}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.key)
			if err != nil {
				t.Fatalf("序列化失败: %v", err)
			}

			var decoded ValJsonKey
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("反序列化失败: %v", err)
			}

			if decoded.Key != tc.key.Key {
				t.Errorf("Key 不匹配: 期望 %s, 实际 %s", tc.key.Key, decoded.Key)
			}
			if decoded.Type != tc.key.Type {
				t.Errorf("Type 不匹配: 期望 %d, 实际 %d", tc.key.Type, decoded.Type)
			}
			if decoded.OriginKey != tc.key.OriginKey {
				t.Errorf("OriginKey 不匹配: 期望 %s, 实际 %s", tc.key.OriginKey, decoded.OriginKey)
			}
		})
	}
}

func TestValJson_JsonSerialization(t *testing.T) {
	testCases := []struct {
		name  string
		value ValJson
	}{
		{"简单值", ValJson{Val: "simple", Tag: []string{}}},
		{"带标签", ValJson{Val: "complex", Tag: []string{"tag1", "tag2"}}},
		{"JSON值", ValJson{Val: `{"key": "value"}`, Tag: []string{"json"}}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.value)
			if err != nil {
				t.Fatalf("序列化失败: %v", err)
			}

			var decoded ValJson
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("反序列化失败: %v", err)
			}

			if decoded.Val != tc.value.Val {
				t.Errorf("Val 不匹配: 期望 %s, 实际 %s", tc.value.Val, decoded.Val)
			}

			if len(decoded.Tag) != len(tc.value.Tag) {
				t.Errorf("Tag 长度不匹配: 期望 %d, 实际 %d", len(tc.value.Tag), len(decoded.Tag))
			}

			for i, tag := range decoded.Tag {
				if tag != tc.value.Tag[i] {
					t.Errorf("Tag[%d] 不匹配: 期望 %s, 实际 %s", i, tc.value.Tag[i], tag)
				}
			}
		})
	}
}
