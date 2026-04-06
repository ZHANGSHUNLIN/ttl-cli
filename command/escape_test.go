package command

import (
	"testing"
	"ttl-cli/util"
)

// TestUnescapeString 测试转义字符功能（add/update 使用）
func TestUnescapeString(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"换行", "a\nb\nc", "a\nb\nc"},
		{"制表", "a\tb", "a\tb"},
		{"回车", "a\rb", "a\rb"},
		{"反斜杠", "a\\b", "a\b"},
		{"双引号", "a\"b", "a\"b"},
		{"单引号", "a'b", "a'b"},
		{"十六进制", "a\x41b", "aAb"},
		{"Unicode", "\u4E2D", "中"},
		{"混合", "a\n\t\\b", "a\n\t\b"},
		{"多行文本", "line1\nline2\nline3", "line1\nline2\nline3"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := util.UnescapeString(tc.input)
			if result != tc.expected {
				t.Errorf("UnescapeString(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

// TestEscapeString 测试转义字符功能
func TestEscapeString(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"换行", "a\nb\nc", "a\\nb\\nc"},
		{"制表", "a\tb", "a\\tb"},
		{"回车", "a\rb", "a\\rb"},
		{"反斜杠", "a\\b", "a\\\\b"},
		{"双引号", "a\"b", "a\\\"b"},
		{"单引号", "a'b", "a\\'b"},
		{"混合", "a\n\t\\b", "a\\n\\t\\\\b"},
		{"多行文本", "line1\nline2\nline3", "line1\\nline2\\nline3"},
		{"空字符", "a\x00b", "a\\0b"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := util.EscapeString(tc.input)
			if result != tc.expected {
				t.Errorf("EscapeString(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

// TestEscapeUnescapeRoundtrip 测试转义与还原的一致性
func TestEscapeUnescapeRoundtrip(t *testing.T) {
	testValues := []string{
		"simple text",
		"a\nb\nc",
		"a\tb\tc",
		"a\\b\\c",
		"a\"b\"c",
		"a'b'c",
		"mixed: \n\t\\\"",
		"line1\nline2\nline3\nline4",
	}

	for _, value := range testValues {
		t.Run(value, func(t *testing.T) {
			escaped := util.EscapeString(value)
			unescaped := util.UnescapeString(escaped)
			if unescaped != value {
				t.Errorf("往返不一致: %q -> %q -> %q", value, escaped, unescaped)
			}
		})
	}
}
