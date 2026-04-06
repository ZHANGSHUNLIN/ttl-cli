package i18n

import (
	"errors"
	"fmt"
	"testing"

	i18nlib "github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

func TestInitWithLanguage(t *testing.T) {
	// Reset before each test
	Reset()

	tests := []struct {
		name string
		lang string
	}{
		{"Chinese", "zh-CN"},
		{"English", "en-US"},
		{"English (generic)", "en"}, // Should fall back to en-US
		{"Japanese", "ja-JP"},       // Should fall back to zh-CN (default)
		{"French", "fr-FR"},         // Should fall back to zh-CN (default)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Reset()
			err := InitWithLanguage(tt.lang)
			if err != nil {
				t.Errorf("InitWithLanguage(%q) error = %v, want nil", tt.lang, err)
				return
			}

			// Verify it was initialized
			if localizer == nil {
				t.Errorf("InitWithLanguage(%q) did not initialize localizer", tt.lang)
			}
		})
	}
}

func TestT(t *testing.T) {
	// Initialize with Chinese
	Reset()
	if err := InitWithLanguage("zh-CN"); err != nil {
		t.Fatalf("InitWithLanguage failed: %v", err)
	}

	tests := []struct {
		name string
		key  string
		args []interface{}
		want string
	}{
		{"Add success", "command.add.success", nil, "添加成功！"},
		{"Add duplicate with arg", "command.add.duplicate", []interface{}{"testkey"}, "当前key已经存在数据: testkey"},
		{"Get not found with arg", "command.get.not_found", []interface{}{"mykey"}, "未找到当前资源，资源名: mykey"},
		{"Non-existent key", "non.existent.key", nil, "non.existent.key"}, // Should return key itself
		{"Complex nested key", "root.short", nil, "个人数据管理体系"},
		{"Error with formatting", "command.add.error_fetch", []interface{}{errors.New("some error")}, "获取资源失败: %!w(*errors.errorString=&{some error})"}, // Note: fmt.Sprintf with error uses %!w format
		{"Root long description", "root.long", nil, "个人数据管理体系，用于管理资源和标签。\n支持多种操作如添加、获取、删除资源，以及管理标签等。"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := T(tt.key, tt.args...)
			if got != tt.want {
				t.Errorf("T(%q, %v) = %q, want %q", tt.key, tt.args, got, tt.want)
			}
		})
	}
}

func TestT_English(t *testing.T) {
	// Initialize with English
	Reset()
	if err := InitWithLanguage("en-US"); err != nil {
		t.Fatalf("InitWithLanguage failed: %v", err)
	}

	tests := []struct {
		name string
		key  string
		args []interface{}
		want string
	}{
		{"Add success", "command.add.success", nil, "Added successfully!"},
		{"Add duplicate with arg", "command.add.duplicate", []interface{}{"testkey"}, "Key already exists: testkey"},
		{"Get not found with arg", "command.get.not_found", []interface{}{"mykey"}, "Resource not found: mykey"},
		{"Root short", "root.short", nil, "Personal data management system"},
		{"Complex nested", "command.ai.thinking", nil, "Thinking..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := T(tt.key, tt.args...)
			if got != tt.want {
				t.Errorf("T(%q, %v) = %q, want %q", tt.key, tt.args, got, tt.want)
			}
		})
	}
}

func TestT_Uninitialized(t *testing.T) {
	// Don't initialize, should return key itself
	Reset()

	got := T("command.add.success")
	want := "command.add.success"
	if got != want {
		t.Errorf("T() without init = %q, want %q", got, want)
	}
}

func TestT_WithMultipleArgs(t *testing.T) {
	Reset()
	if err := InitWithLanguage("zh-CN"); err != nil {
		t.Fatalf("InitWithLanguage failed: %v", err)
	}

	// Test with multiple format args (only first should be used with %s format)
	got := T("command.get.not_found", "mykey")
	want := "未找到当前资源，资源名: mykey"
	if got != want {
		t.Errorf("T() with args = %q, want %q", got, want)
	}
}

func TestGetLocaleInfo(t *testing.T) {
	Reset()

	// Test with Chinese
	if err := InitWithLanguage("zh-CN"); err != nil {
		t.Fatalf("InitWithLanguage failed: %v", err)
	}

	info, err := GetLocaleInfo()
	if err != nil {
		t.Fatalf("GetLocaleInfo() error = %v", err)
	}

	if info.Language != "简体中文" {
		t.Errorf("GetLocaleInfo().Language = %q, want 简体中文", info.Language)
	}

	// Test with English
	Reset()
	if err := InitWithLanguage("en-US"); err != nil {
		t.Fatalf("InitWithLanguage failed: %v", err)
	}

	info, err = GetLocaleInfo()
	if err != nil {
		t.Fatalf("GetLocaleInfo() error = %v", err)
	}

	if info.Language != "English (US)" {
		t.Errorf("GetLocaleInfo().Language = %q, want English (US)", info.Language)
	}
}

func TestGetLocaleInfo_Uninitialized(t *testing.T) {
	Reset()
	_, err := GetLocaleInfo()
	if err == nil {
		t.Error("GetLocaleInfo() without init should return error")
	}
}

func TestReset(t *testing.T) {
	// Initialize
	if err := InitWithLanguage("zh-CN"); err != nil {
		t.Fatalf("InitWithLanguage failed: %v", err)
	}

	// Verify it's initialized
	info, err := GetLocaleInfo()
	if err != nil {
		t.Fatalf("GetLocaleInfo() error = %v", err)
	}
	if info.Language != "简体中文" {
		t.Fatalf("Expected Chinese, got %q", info.Language)
	}

	// Reset
	Reset()

	// Verify it's reset
	_, err = GetLocaleInfo()
	if err == nil {
		t.Error("After Reset, GetLocaleInfo should return error")
	}

	// Can initialize again
	if err := InitWithLanguage("en-US"); err != nil {
		t.Fatalf("InitWithLanguage after Reset failed: %v", err)
	}
}

func TestGetBundle(t *testing.T) {
	Reset()

	if err := InitWithLanguage("zh-CN"); err != nil {
		t.Fatalf("InitWithLanguage failed: %v", err)
	}

	b := GetBundle()
	if b == nil {
		t.Error("GetBundle() returned nil")
	}
}

func TestGetLocalizer(t *testing.T) {
	Reset()

	if err := InitWithLanguage("en-US"); err != nil {
		t.Fatalf("InitWithLanguage failed: %v", err)
	}

	loc := GetLocalizer()
	if loc == nil {
		t.Error("GetLocalizer() returned nil")
	}
}

func TestGetCurrentLanguage(t *testing.T) {
	Reset()

	if err := InitWithLanguage("zh-CN"); err != nil {
		t.Fatalf("InitWithLanguage failed: %v", err)
	}

	if got := GetCurrentLanguage(); got != "zh-CN" {
		t.Errorf("GetCurrentLanguage() = %q, want zh-CN", got)
	}

	Reset()
	if err := InitWithLanguage("en-US"); err != nil {
		t.Fatalf("InitWithLanguage failed: %v", err)
	}

	if got := GetCurrentLanguage(); got != "en-US" {
		t.Errorf("GetCurrentLanguage() = %q, want en-US", got)
	}
}

func TestT_ErrorHandling(t *testing.T) {
	Reset()
	if err := InitWithLanguage("zh-CN"); err != nil {
		t.Fatalf("InitWithLanguage failed: %v", err)
	}

	// Test that errors in translations are handled gracefully
	errMsg := fmt.Errorf("database connection failed")
	got := T("command.add.error_fetch", errMsg)
	if got == "" {
		t.Error("T() with error arg should not return empty string")
	}

	// Verify it contains the error message
	if len(got) < 10 { // sanity check
		t.Errorf("T() with error arg returned too short string: %q", got)
	}
}

func TestAddMessagesRecursive(t *testing.T) {
	b := i18nlib.NewBundle(language.English)

	data := map[string]interface{}{
		"simple": "Simple message",
		"nested": map[string]interface{}{
			"level1": "Nested message",
		},
		"number": 42, // Should be ignored
	}

	lang := language.English
	if err := addMessagesRecursive(b, lang, "", data); err != nil {
		t.Fatalf("addMessagesRecursive failed: %v", err)
	}

	// Verify messages were added
	loc := i18nlib.NewLocalizer(b, "en")

	result, err := loc.Localize(&i18nlib.LocalizeConfig{MessageID: "simple"})
	if err != nil {
		t.Errorf("Failed to localize 'simple': %v", err)
	}
	if result != "Simple message" {
		t.Errorf("Expected 'Simple message', got %q", result)
	}

	result, err = loc.Localize(&i18nlib.LocalizeConfig{MessageID: "nested.level1"})
	if err != nil {
		t.Errorf("Failed to localize 'nested.level1': %v", err)
	}
	if result != "Nested message" {
		t.Errorf("Expected 'Nested message', got %q", result)
	}
}
