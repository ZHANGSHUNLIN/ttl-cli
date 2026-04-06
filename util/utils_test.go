package util

import (
	"testing"
)

func TestUnescapeString_Newline(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "单个换行符",
			input:    "a\\nb",
			expected: "a\nb",
		},
		{
			name:     "多个换行符",
			input:    "第一行\\n第二行\\n第三行",
			expected: "第一行\n第二行\n第三行",
		},
		{
			name:     "换行符在开头",
			input:    "\\n开头换行",
			expected: "\n开头换行",
		},
		{
			name:     "换行符在结尾",
			input:    "结尾换行\\n",
			expected: "结尾换行\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UnescapeString(tt.input)
			if result != tt.expected {
				t.Errorf("UnescapeString(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestUnescapeString_Tab(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "单个制表符",
			input:    "a\\tb",
			expected: "a\tb",
		},
		{
			name:     "多个制表符",
			input:    "列1\\t列2\\t列3",
			expected: "列1\t列2\t列3",
		},
		{
			name:     "混合换行和制表符",
			input:    "行1\\n\\t缩进\\n行2",
			expected: "行1\n\t缩进\n行2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UnescapeString(tt.input)
			if result != tt.expected {
				t.Errorf("UnescapeString(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestUnescapeString_SpecialChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "反斜杠",
			input:    "路径\\\\文件",
			expected: "路径\\文件",
		},
		{
			name:     "双引号",
			input:    "他说\\\"你好\\\"",
			expected: "他说\"你好\"",
		},
		{
			name:     "单引号",
			input:    "it\\'s",
			expected: "it's",
		},
		{
			name:     "回车符",
			input:    "a\\rb",
			expected: "a\rb",
		},
		{
			name:     "混合特殊字符",
			input:    "换行\\n制表\\t反斜杠\\\\引号\\\"",
			expected: "换行\n制表\t反斜杠\\引号\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UnescapeString(tt.input)
			if result != tt.expected {
				t.Errorf("UnescapeString(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestUnescapeString_NoEscape(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "普通字符串",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "中文字符串",
			input:    "你好世界",
			expected: "你好世界",
		},
		{
			name:     "空字符串",
			input:    "",
			expected: "",
		},
		{
			name:     "数字字符串",
			input:    "123456",
			expected: "123456",
		},
		{
			name:     "URL",
			input:    "https://example.com/path?query=value",
			expected: "https://example.com/path?query=value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UnescapeString(tt.input)
			if result != tt.expected {
				t.Errorf("UnescapeString(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestUnescapeString_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "单个反斜杠在结尾",
			input:    "结尾\\",
			expected: "结尾\\",
		},
		{
			name:     "连续转义字符",
			input:    "a\\n\\t\\r\\nb",
			expected: "a\n\t\r\nb",
		},
		{
			name:     "多个反斜杠",
			input:    "\\\\\\\\",
			expected: "\\\\",
		},
		{
			name:     "转义字符后没有字符",
			input:    "test\\",
			expected: "test\\",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UnescapeString(tt.input)
			if result != tt.expected {
				t.Errorf("UnescapeString(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestEscapeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "换行符转义",
			input:    "a\nb",
			expected: "a\\nb",
		},
		{
			name:     "制表符转义",
			input:    "a\tb",
			expected: "a\\tb",
		},
		{
			name:     "反斜杠转义",
			input:    "a\\b",
			expected: "a\\\\b",
		},
		{
			name:     "双引号转义",
			input:    "a\"b",
			expected: "a\\\"b",
		},
		{
			name:     "普通字符串不转义",
			input:    "hello",
			expected: "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EscapeString(tt.input)
			if result != tt.expected {
				t.Errorf("EscapeString(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestEscapeUnescapeRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "换行和制表符",
			input: "第一行\n第二行\t第三行",
		},
		{
			name:  "特殊字符",
			input: "路径\\文件\"引号\"",
		},
		{
			name:  "混合内容",
			input: "Hello\nWorld\t测试\\\"完成",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			escaped := EscapeString(tt.input)
			unescaped := UnescapeString(escaped)
			if unescaped != tt.input {
				t.Errorf("Round trip failed: %q -> %q -> %q", tt.input, escaped, unescaped)
			}
		})
	}
}
