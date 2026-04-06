package conf

import (
	"fmt"
	"gopkg.in/ini.v1"
	"os"
	"path/filepath"
	"ttl-cli/models"
)

func GetTtlConf() (models.TtlIni, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return models.TtlIni{}, fmt.Errorf("failed to get user directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".ttl")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return models.TtlIni{}, fmt.Errorf("failed to create config directory: %w", err)
	}

	confFilePath := filepath.Join(configDir, "ttl.ini")

	if _, err := os.Stat(confFilePath); os.IsNotExist(err) {
		// 不再预设具体文件名，由 GetDBPath 根据存储类型动态生成
		return createDefaultConfig(confFilePath, "")
	}

	return loadConfFile(confFilePath)
}

func GetTtlConfFromFile(confFile string) (models.TtlIni, error) {
	if _, err := os.Stat(confFile); os.IsNotExist(err) {
		// 创建目录和默认配置
		if err := os.MkdirAll(filepath.Dir(confFile), 0755); err != nil {
			return models.TtlIni{}, fmt.Errorf("failed to create config directory: %w", err)
		}
		// 不再预设具体文件名，由 GetDBPath 根据存储类型动态生成
		return createDefaultConfig(confFile, "")
	}
	return loadConfFile(confFile)
}

func loadConfFile(path string) (models.TtlIni, error) {
	cfg, err := ini.Load(path)
	if err != nil {
		return models.TtlIni{}, fmt.Errorf("failed to load config file: %w", err)
	}
	var ttlIni models.TtlIni
	if err := cfg.Section("").MapTo(&ttlIni); err != nil {
		return models.TtlIni{}, fmt.Errorf("failed to parse config file: %w", err)
	}

	// 尝试从 [storage] section 读取配置（优先级高于根 section）
	if cfg.HasSection("storage") {
		storageSec := cfg.Section("storage")
		if storageType := storageSec.Key("type").String(); storageType != "" {
			ttlIni.StorageType = storageType
		}
		if storagePath := storageSec.Key("path").String(); storagePath != "" {
			ttlIni.DbPath = storagePath
		}
	}

	// 设置默认存储类型为 sqlite
	if ttlIni.StorageType == "" {
		ttlIni.StorageType = "sqlite"
	}

	if err := cfg.Section("ai").MapTo(&ttlIni.AI); err != nil {
		return models.TtlIni{}, fmt.Errorf("failed to parse AI config: %w", err)
	}
	if ttlIni.AI.BaseURL == "" {
		ttlIni.AI.BaseURL = "https://api.openai.com"
	}
	if ttlIni.AI.Model == "" {
		ttlIni.AI.Model = "gpt-4o-mini"
	}
	if ttlIni.AI.Timeout == 0 {
		ttlIni.AI.Timeout = 30
	}

	// 解析 [bbolt] section
	if err := cfg.Section("bbolt").MapTo(&ttlIni.BoltDB); err != nil {
		return models.TtlIni{}, fmt.Errorf("failed to parse bbolt config: %w", err)
	}
	if ttlIni.BoltDB.Timeout == 0 {
		ttlIni.BoltDB.Timeout = 5 // 默认 5 秒超时
	}

	return ttlIni, nil
}

func LoadAIConfig(confFile string) (models.AIConfig, error) {
	var ttlConf models.TtlIni
	var err error
	if confFile != "" {
		ttlConf, err = GetTtlConfFromFile(confFile)
	} else {
		ttlConf, err = GetTtlConf()
	}
	if err != nil {
		return models.AIConfig{}, err
	}
	return ttlConf.AI, nil
}

func SaveAIConfig(confFile string, aiConf models.AIConfig) error {
	path := confFile
	if path == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user directory: %w", err)
		}
		path = filepath.Join(homeDir, ".ttl", "ttl.ini")
	}

	cfg, err := ini.Load(path)
	if err != nil {
		cfg = ini.Empty()
	}

	sec := cfg.Section("ai")
	sec.Key("api_key").SetValue(aiConf.APIKey)
	sec.Key("base_url").SetValue(aiConf.BaseURL)
	sec.Key("model").SetValue(aiConf.Model)
	sec.Key("timeout").SetValue(fmt.Sprintf("%d", aiConf.Timeout))
	// 保存多轮上下文配置
	sec.Key("context_enabled").SetValue(fmt.Sprintf("%v", aiConf.ContextEnabled))
	sec.Key("context_idle_ttl").SetValue(fmt.Sprintf("%d", aiConf.ContextIdleTTL))
	sec.Key("context_max_rounds").SetValue(fmt.Sprintf("%d", aiConf.ContextMaxRounds))
	sec.Key("context_max_tokens").SetValue(fmt.Sprintf("%d", aiConf.ContextMaxTokens))

	return cfg.SaveTo(path)
}

func GetDefaultConfPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user directory: %w", err)
	}
	return filepath.Join(homeDir, ".ttl", "ttl.ini"), nil
}

func createDefaultConfig(configPath, dbPath string) (models.TtlIni, error) {
	cfg := ini.Empty()

	storageSec := cfg.Section("storage")
	storageSec.Key("type").SetValue("sqlite")

	// 只有当明确指定了 dbPath 时才写入配置文件
	// 否则由 GetDBPath() 根据存储类型动态生成路径
	if dbPath != "" {
		storageSec.Key("path").SetValue(dbPath)
		cfg.Section("").Key("db_path").SetValue(dbPath)
	}

	if err := cfg.SaveTo(configPath); err != nil {
		return models.TtlIni{}, fmt.Errorf("failed to save default config: %w", err)
	}

	return models.TtlIni{
		StorageType: "sqlite",
	}, nil
}
