package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed locales/*.json
var localeFS embed.FS

// ReadEmbeddedJSON reads a locale file from embedded filesystem
func ReadEmbeddedJSON(name string) ([]byte, error) {
	return localeFS.ReadFile("locales/" + name)
}

var (
	localizer   *i18n.Localizer
	bundle      *i18n.Bundle
	once        sync.Once
	initError   error
	currentLang string
)

func Init() error {
	once.Do(func() {
		bundle, initError = createBundle()
		if initError != nil {
			return
		}

		detectedLang := DetectSystemLanguage()
		currentLang = normalizeLanguageTag(detectedLang)
		localizer = i18n.NewLocalizer(bundle, currentLang)
	})
	return initError
}

func normalizeLanguageTag(tag language.Tag) string {
	base, _ := tag.Base()
	region, _ := tag.Region()

	switch base.String() {
	case "zh":
		return "zh-CN"
	case "en":
		if region.String() != "ZZ" && region.String() != "" {
			return "en-US"
		}
		return "en-US"
	case "ja":
		return "ja-JP"
	case "ko":
		return "ko-KR"
	case "fr":
		return "fr-FR"
	default:
		return "en-US"
	}
}

func InitWithLanguage(lang string) error {
	once.Do(func() {
		bundle, initError = createBundle()
		if initError != nil {
			return
		}
		currentLang = lang
		localizer = i18n.NewLocalizer(bundle, lang)
	})
	return initError
}

func createBundle() (*i18n.Bundle, error) {
	b := i18n.NewBundle(language.English)

	// 加载中文翻译 - 从嵌入文件系统读取
	data, err := localeFS.ReadFile("locales/zh-CN.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read zh-CN.json: %w", err)
	}

	var translations map[string]interface{}
	if err := json.Unmarshal(data, &translations); err != nil {
		return nil, fmt.Errorf("failed to parse zh-CN.json: %w", err)
	}

	if err := addMessagesRecursive(b, language.MustParse("zh-CN"), "", translations); err != nil {
		return nil, fmt.Errorf("failed to add messages for zh-CN: %w", err)
	}

	// 加载英文翻译
	data, err = localeFS.ReadFile("locales/en-US.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read en-US.json: %w", err)
	}

	if err := json.Unmarshal(data, &translations); err != nil {
		return nil, fmt.Errorf("failed to parse en-US.json: %w", err)
	}

	if err := addMessagesRecursive(b, language.MustParse("en-US"), "", translations); err != nil {
		return nil, fmt.Errorf("failed to add messages for en-US: %w", err)
	}

	// 加载日文翻译
	data, err = localeFS.ReadFile("locales/ja-JP.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read ja-JP.json: %w", err)
	}

	if err := json.Unmarshal(data, &translations); err != nil {
		return nil, fmt.Errorf("failed to parse ja-JP.json: %w", err)
	}

	if err := addMessagesRecursive(b, language.MustParse("ja-JP"), "", translations); err != nil {
		return nil, fmt.Errorf("failed to add messages for ja-JP: %w", err)
	}

	// 加载韩文翻译
	data, err = localeFS.ReadFile("locales/ko-KR.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read ko-KR.json: %w", err)
	}

	if err := json.Unmarshal(data, &translations); err != nil {
		return nil, fmt.Errorf("failed to parse ko-KR.json: %w", err)
	}

	if err := addMessagesRecursive(b, language.MustParse("ko-KR"), "", translations); err != nil {
		return nil, fmt.Errorf("failed to add messages for ko-KR: %w", err)
	}

	// 加载法文翻译
	data, err = localeFS.ReadFile("locales/fr-FR.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read fr-FR.json: %w", err)
	}

	if err := json.Unmarshal(data, &translations); err != nil {
		return nil, fmt.Errorf("failed to parse fr-FR.json: %w", err)
	}

	if err := addMessagesRecursive(b, language.MustParse("fr-FR"), "", translations); err != nil {
		return nil, fmt.Errorf("failed to add messages for fr-FR: %w", err)
	}

	return b, nil
}

func addMessagesRecursive(b *i18n.Bundle, tag language.Tag, prefix string, data map[string]interface{}) error {
	for key, value := range data {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		switch v := value.(type) {
		case string:
			b.AddMessages(tag, &i18n.Message{
				ID:    fullKey,
				Other: v,
			})
		case map[string]interface{}:
			if err := addMessagesRecursive(b, tag, fullKey, v); err != nil {
				return err
			}
		}
	}
	return nil
}

func T(key string, args ...interface{}) string {
	if localizer == nil {
		return key
	}

	cfg := &i18n.LocalizeConfig{
		MessageID: key,
	}

	result, err := localizer.Localize(cfg)
	if err != nil {
		result = key
	}

	if len(args) > 0 {
		return fmt.Sprintf(result, args...)
	}

	return result
}

func TPlural(key string, count int, args ...interface{}) string {
	if localizer == nil {
		return key
	}

	cfg := &i18n.LocalizeConfig{
		MessageID:   key,
		PluralCount: count,
		TemplateData: map[string]interface{}{
			"Count": count,
		},
	}

	if len(args) > 0 {
		cfg.TemplateData = args[0]
	}

	return localizer.MustLocalize(cfg)
}

func GetBundle() *i18n.Bundle {
	return bundle
}

func GetLocalizer() *i18n.Localizer {
	return localizer
}

func Reset() {
	once = sync.Once{}
	localizer = nil
	bundle = nil
	initError = nil
	currentLang = ""
}

type LocaleInfo struct {
	Language string `json:"language"`
	Version  string `json:"version"`
}

func GetLocaleInfo() (LocaleInfo, error) {
	if localizer == nil {
		return LocaleInfo{}, fmt.Errorf("i18n not initialized")
	}

	filename := currentLang + ".json"
	data, err := localeFS.ReadFile("locales/" + filename)
	if err != nil {
		return LocaleInfo{}, fmt.Errorf("failed to read locale file: %w", err)
	}

	var info LocaleInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return LocaleInfo{}, fmt.Errorf("failed to parse locale info: %w", err)
	}

	return info, nil
}

func GetCurrentLanguage() string {
	return currentLang
}
