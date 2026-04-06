package command

import (
	"testing"
)

// TestContainsHelper 测试辅助函数
func TestContainsHelper(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "包含子串",
			s:        "Hello World",
			substr:   "World",
			expected: true,
		},
		{
			name:     "不包含子串",
			s:        "Hello World",
			substr:   "Test",
			expected: false,
		},
		{
			name:     "不区分大小写包含",
			s:        "Hello World",
			substr:   "WORLD",
			expected: false, // 注意：contains 是区分大小写的
		},
		{
			name:     "空字符串",
			s:        "",
			substr:   "Test",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, result, tt.expected)
			}
		})
	}
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
