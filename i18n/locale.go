package i18n

import (
	"os"
	"strings"

	"golang.org/x/text/language"
)

var SupportedLanguages = []language.Tag{
	language.MustParse("zh-CN"),
	language.English,
	language.MustParse("ja-JP"),
	language.MustParse("ko-KR"),
	language.MustParse("fr-FR"),
}

func DetectSystemLanguage() language.Tag {
	if lang := os.Getenv("LC_ALL"); lang != "" {
		if tag := parseLanguageTag(lang); tag != language.Und {
			return matchLanguage(tag)
		}
	}

	if lang := os.Getenv("LANG"); lang != "" {
		if tag := parseLanguageTag(lang); tag != language.Und {
			return matchLanguage(tag)
		}
	}

	return language.MustParse("zh-CN")
}

func parseLanguageTag(lang string) language.Tag {
	parts := strings.SplitN(lang, ".", 2)
	langPart := parts[0]

	if langPart == "C" {
		return language.English
	}

	langPart = strings.ReplaceAll(langPart, "_", "-")

	tag, err := language.Parse(langPart)
	if err != nil {
		return language.Und
	}

	return tag
}

func matchLanguage(tag language.Tag) language.Tag {
	matcher := language.NewMatcher(SupportedLanguages)
	matched, _, _ := matcher.Match(tag)
	return matched
}
