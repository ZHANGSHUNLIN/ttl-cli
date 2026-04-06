package i18n

import (
	"os"
	"testing"

	"golang.org/x/text/language"
)

func TestDetectSystemLanguage(t *testing.T) {
	// Save original env
	origLC_ALL := os.Getenv("LC_ALL")
	origLANG := os.Getenv("LANG")
	defer func() {
		os.Setenv("LC_ALL", origLC_ALL)
		os.Setenv("LANG", origLANG)
		Reset()
	}()

	tests := []struct {
		name     string
		lcAll    string
		lang     string
		wantBase string // the base language (without region variants)
	}{
		{
			name:     "Chinese simplified",
			lcAll:    "zh_CN.UTF-8",
			wantBase: "zh",
		},
		{
			name:     "Chinese simplified without encoding",
			lcAll:    "zh_CN",
			wantBase: "zh",
		},
		{
			name:     "English US",
			lcAll:    "en_US.UTF-8",
			wantBase: "en",
		},
		{
			name:     "English only",
			lcAll:    "en",
			wantBase: "en",
		},
		{
			name:     "C locale defaults to English",
			lcAll:    "C",
			wantBase: "en",
		},
		{
			name:     "Japanese",
			lcAll:    "ja_JP.UTF-8",
			wantBase: "ja",
		},
		{
			name:     "French",
			lcAll:    "fr_FR.UTF-8",
			wantBase: "fr",
		},
		{
			name:     "LANG falls back when LC_ALL empty",
			lang:     "en_US.UTF-8",
			wantBase: "en",
		},
		{
			name:     "Both empty uses default (Chinese for our users)",
			wantBase: "zh",
		},
		{
			name:     "LC_ALL takes precedence over LANG",
			lcAll:    "zh_CN.UTF-8",
			lang:     "en_US.UTF-8",
			wantBase: "zh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("LC_ALL", tt.lcAll)
			os.Setenv("LANG", tt.lang)

			got := DetectSystemLanguage()

			// Check the base language (first part before any variants)
			gotBase := got.String()
			if len(gotBase) > 2 && gotBase[2] == '-' {
				gotBase = gotBase[:2]
			}

			// For tests without a specific wantBase, just verify it's a supported language
			if tt.wantBase == "" {
				if gotBase != "zh" && gotBase != "en" {
					t.Errorf("DetectSystemLanguage() = %v (base: %v), want zh or en", got.String(), gotBase)
				}
				return
			}

			if gotBase != tt.wantBase {
				t.Errorf("DetectSystemLanguage() = %v (base: %v), want base %v", got.String(), gotBase, tt.wantBase)
			}
		})
	}
}

func TestParseLanguageTag(t *testing.T) {
	tests := []struct {
		name string
		lang string
		want string
	}{
		{"Chinese with encoding", "zh_CN.UTF-8", "zh-CN"},
		{"Chinese without encoding", "zh_CN", "zh-CN"},
		{"English with encoding", "en_US.UTF-8", "en-US"},
		{"English only", "en", "en"},
		{"C locale", "C", "en"},
		{"Japanese with encoding", "ja_JP.UTF-8", "ja-JP"},
		{"Korean with encoding", "ko_KR.UTF-8", "ko-KR"},
		{"Invalid", "invalid", "und"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLanguageTag(tt.lang)
			if got.String() != tt.want {
				t.Errorf("parseLanguageTag(%q) = %v, want %v", tt.lang, got.String(), tt.want)
			}
		})
	}
}

func TestMatchLanguage(t *testing.T) {
	tests := []struct {
		name    string
		tag     string
		wantZh  bool // should match to Chinese
		wantEn  bool // should match to English
		wantAny bool // should match to any supported language
	}{
		{"Exact match zh-CN", "zh-CN", true, false, false},
		{"Exact match en-US", "en-US", false, true, false},
		{"Exact match en", "en", false, true, false},
		{"Chinese simplified variant", "zh-CN", true, false, false},
		{"French", "fr-FR", false, false, false},
		{"Japanese", "ja-JP", false, false, false},
		{"Korean", "ko-KR", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag := language.MustParse(tt.tag)
			got := matchLanguage(tag)

			// Check the base language
			gotBase := got.String()
			if len(gotBase) > 2 && gotBase[2] == '-' {
				gotBase = gotBase[:2]
			}

			if tt.wantAny {
				// Just verify it's a supported language
				if gotBase != "zh" && gotBase != "en" {
					t.Errorf("matchLanguage(%v) = %v (base: %v), want zh or en", tag, got.String(), gotBase)
				}
				return
			}

			if tt.wantZh && gotBase != "zh" {
				t.Errorf("matchLanguage(%v) = %v (base: %v), want Chinese", tag, got.String(), gotBase)
			}
			if tt.wantEn && gotBase != "en" {
				t.Errorf("matchLanguage(%v) = %v (base: %v), want English", tag, got.String(), gotBase)
			}
		})
	}
}

func TestDetectSystemLanguageReal(t *testing.T) {
	// Save original env
	origLC_ALL := os.Getenv("LC_ALL")
	origLANG := os.Getenv("LANG")

	defer func() {
		os.Setenv("LC_ALL", origLC_ALL)
		os.Setenv("LANG", origLANG)
		Reset()
	}()

	os.Setenv("LC_ALL", "zh_CN.UTF-8")
	os.Setenv("LANG", "")

	got := DetectSystemLanguage()
	gotBase := got.String()
	if len(gotBase) > 2 && gotBase[2] == '-' {
		gotBase = gotBase[:2]
	}

	if gotBase != "zh" {
		t.Errorf("DetectSystemLanguage() with LC_ALL=zh_CN.UTF-8 = %v (base: %v), want zh", got.String(), gotBase)
	}
}

func TestParseLanguageTagReal(t *testing.T) {
	tests := []struct {
		name string
		lang string
		want string
	}{
		{"Chinese with encoding", "zh_CN.UTF-8", "zh-CN"},
		{"English with encoding", "en_US.UTF-8", "en-US"},
		{"C locale", "C", "en"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLanguageTag(tt.lang)
			if got.String() != tt.want {
				t.Errorf("parseLanguageTag(%q) = %v, want %v", tt.lang, got.String(), tt.want)
			}
		})
	}
}
