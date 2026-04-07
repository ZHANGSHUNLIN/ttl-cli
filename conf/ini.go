package conf

import (
	"fmt"
	"gopkg.in/ini.v1"
	"ttl-cli/models"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
		return createDefaultConfig(confFilePath, "")
	}

	return loadConfFile(confFilePath)
}

func GetTtlConfFromFile(confFile string) (models.TtlIni, error) {
	if _, err := os.Stat(confFile); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(confFile), 0755); err != nil {
			return models.TtlIni{}, fmt.Errorf("failed to create config directory: %w", err)
		}
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

	if cfg.HasSection("storage") {
		storageSec := cfg.Section("storage")
		if storageType := storageSec.Key("type").String(); storageType != "" {
			ttlIni.StorageType = storageType
		}
		if storagePath := storageSec.Key("path").String(); storagePath != "" {
			ttlIni.DbPath = storagePath
		}
	}

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

	if err := cfg.Section("bbolt").MapTo(&ttlIni.BoltDB); err != nil {
		return models.TtlIni{}, fmt.Errorf("failed to parse bbolt config: %w", err)
	}
	if ttlIni.BoltDB.Timeout == 0 {
		ttlIni.BoltDB.Timeout = 5
	}

	ttlIni.Workspaces = make(map[string]models.WorkspaceConfig)
	for _, section := range cfg.Sections() {
		name := section.Name()
		if strings.HasPrefix(name, "workspaces.") {
			wsName := strings.TrimPrefix(name, "workspaces.")
			if wsName != "" {
				wsConfig := models.WorkspaceConfig{
					DbPath:      section.Key("db_path").String(),
					StorageType: section.Key("storage_type").String(),
				}
				if wsConfig.StorageType == "" {
					wsConfig.StorageType = ttlIni.StorageType
				}
				ttlIni.Workspaces[wsName] = wsConfig
			}
		}
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

	if dbPath != "" {
		storageSec.Key("path").SetValue(dbPath)
		cfg.Section("").Key("db_path").SetValue(dbPath)
	}

	if err := cfg.SaveTo(configPath); err != nil {
		return models.TtlIni{}, fmt.Errorf("failed to save default config: %w", err)
	}

	return models.TtlIni{}, nil
}

func GetWorkspaceDBPath(confFile string) (string, string, error) {
	var ttlConf models.TtlIni
	var err error
	if confFile != "" {
		ttlConf, err = GetTtlConfFromFile(confFile)
	} else {
		ttlConf, err = GetTtlConf()
	}
	if err != nil {
		return "", "", err
	}

	workspaceName := ttlConf.Workspace
	if workspaceName == "" {
		workspaceName = "default"
	}

	if ws, ok := ttlConf.Workspaces[workspaceName]; ok && ws.DbPath != "" {
		return workspaceName, ws.DbPath, nil
	}

	if ttlConf.DbPath != "" {
		return workspaceName, ttlConf.DbPath, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("failed to get user directory: %w", err)
	}
	return workspaceName, filepath.Join(homeDir, ".ttl", "data.db"), nil
}

func ValidateWorkspaceName(name string) bool {
	if name == "" {
		return false
	}
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, name)
	return matched
}

func CreateWorkspace(confFile, name string) (string, error) {
	if !ValidateWorkspaceName(name) {
		return "", fmt.Errorf("工作空间名称只能包含字母、数字、下划线、连字符")
	}

	path := confFile
	if path == "" {
		var err error
		path, err = GetDefaultConfPath()
		if err != nil {
			return "", err
		}
	}

	cfg, err := ini.Load(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg = ini.Empty()
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return "", fmt.Errorf("failed to create config directory: %w", err)
			}
		} else {
			return "", fmt.Errorf("failed to load config file: %w", err)
		}
	}

	sectionName := "workspaces." + name
	if cfg.HasSection(sectionName) {
		return "", fmt.Errorf("工作空间已存在: %s", name)
	}

	// 确定工作空间目录：基于配置文件所在目录
	var wsDir string
	confDir := filepath.Dir(path)
	// 如果配置文件在 ~/.ttl/ 目录下，则使用 ~/.ttl/workspaces/
	homeDir, err := os.UserHomeDir()
	if err == nil {
		defaultConfDir := filepath.Join(homeDir, ".ttl")
		if confDir == defaultConfDir {
			wsDir = filepath.Join(homeDir, ".ttl", "workspaces")
		}
	}
	// 否则在配置文件同目录下创建 workspaces 子目录
	if wsDir == "" {
		wsDir = filepath.Join(confDir, "workspaces")
	}
	if err := os.MkdirAll(wsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create workspaces directory: %w", err)
	}

	storageType := ""
	if cfg.HasSection("storage") {
		storageType = cfg.Section("storage").Key("type").String()
		if storageType == "" {
			storageType = cfg.Section("storage").Key("storage_type").String()
		}
	}
	if storageType == "" {
		storageType = cfg.Section("").Key("storage_type").String()
	}
	if storageType == "" {
		storageType = "sqlite"
	}

	// 根据存储类型决定数据库文件后缀
	dbExt := ".db"
	if storageType == "local" || storageType == "bbolt" {
		dbExt = ".bbolt"
	}

	dbPath := filepath.Join(wsDir, name+dbExt)

	sec := cfg.Section(sectionName)
	sec.Key("db_path").SetValue(dbPath)
	sec.Key("storage_type").SetValue(storageType)

	if err := cfg.SaveTo(path); err != nil {
		return "", fmt.Errorf("failed to save config: %w", err)
	}

	return dbPath, nil
}

func DeleteWorkspace(confFile, name string) error {
	path := confFile
	if path == "" {
		var err error
		path, err = GetDefaultConfPath()
		if err != nil {
			return err
		}
	}

	cfg, err := ini.Load(path)
	if err != nil {
		return fmt.Errorf("failed to load config file: %w", err)
	}

	sectionName := "workspaces." + name
	if !cfg.HasSection(sectionName) {
		return fmt.Errorf("工作空间不存在: %s", name)
	}

	sec := cfg.Section(sectionName)
	dbPath := sec.Key("db_path").String()

	cfg.DeleteSection(sectionName)

	if err := cfg.SaveTo(path); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	if dbPath != "" {
		if _, err := os.Stat(dbPath); err == nil {
			if err := os.Remove(dbPath); err != nil {
				return fmt.Errorf("failed to delete database file: %w", err)
			}
		}
	}

	keyFile := dbPath[:len(dbPath)-len(filepath.Ext(dbPath))] + ".key"
	if _, err := os.Stat(keyFile); err == nil {
		os.Remove(keyFile)
	}

	return nil
}

func SwitchWorkspace(confFile, name string) error {
	path := confFile
	if path == "" {
		var err error
		path, err = GetDefaultConfPath()
		if err != nil {
			return err
		}
	}

	cfg, err := ini.Load(path)
	if err != nil {
		return fmt.Errorf("failed to load config file: %w", err)
	}

	sectionName := "workspaces." + name
	if !cfg.HasSection(sectionName) && name != "default" {
		return fmt.Errorf("工作空间不存在: %s", name)
	}

	workspaceSec := cfg.Section("")
	workspaceSec.Key("workspace").SetValue(name)

	if err := cfg.SaveTo(path); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

func GetCurrentWorkspace(confFile string) (string, error) {
	var ttlConf models.TtlIni
	var err error
	if confFile != "" {
		ttlConf, err = GetTtlConfFromFile(confFile)
	} else {
		ttlConf, err = GetTtlConf()
	}
	if err != nil {
		return "", err
	}

	if ttlConf.Workspace == "" {
		return "default", nil
	}
	return ttlConf.Workspace, nil
}

func ListWorkspaces(confFile string) ([]string, string, error) {
	var ttlConf models.TtlIni
	var err error
	if confFile != "" {
		ttlConf, err = GetTtlConfFromFile(confFile)
	} else {
		ttlConf, err = GetTtlConf()
	}
	if err != nil {
		return nil, "", err
	}

	current := ttlConf.Workspace
	if current == "" {
		current = "default"
	}

	var names []string
	for name := range ttlConf.Workspaces {
		names = append(names, name)
	}

	return names, current, nil
}

func GetWorkspaceInfo(confFile, name string) (string, string, int, error) {
	var ttlConf models.TtlIni
	var err error
	if confFile != "" {
		ttlConf, err = GetTtlConfFromFile(confFile)
	} else {
		ttlConf, err = GetTtlConf()
	}
	if err != nil {
		return "", "", 0, err
	}

	var dbPath, storageType string

	if name == "default" {
		dbPath = ttlConf.DbPath
		storageType = ttlConf.StorageType
		if storageType == "" {
			storageType = "sqlite"
		}
	} else {
		ws, ok := ttlConf.Workspaces[name]
		if !ok {
			return "", "", 0, fmt.Errorf("工作空间不存在: %s", name)
		}
		dbPath = ws.DbPath
		storageType = ws.StorageType
		if storageType == "" {
			storageType = "sqlite"
		}
	}

	var count int
	if dbPath != "" {
		if _, err := os.Stat(dbPath); err == nil {
			count = 1
		}
	}

	return dbPath, storageType, count, nil
}

func MigrateToWorkspaces(confFile string) error {
	path := confFile
	if path == "" {
		var err error
		path, err = GetDefaultConfPath()
		if err != nil {
			return err
		}
	}

	cfg, err := ini.Load(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg = ini.Empty()
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return fmt.Errorf("failed to create config directory: %w", err)
			}
		} else {
			return fmt.Errorf("failed to load config file: %w", err)
		}
	}

	storageSec := cfg.Section("storage")
	oldDbPath := cfg.Section("").Key("db_path").String()
	storageType := storageSec.Key("type").String()
	if storageType == "" {
		storageType = cfg.Section("").Key("storage_type").String()
	}
	if storageType == "" {
		storageType = "sqlite"
	}

	if oldDbPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user directory: %w", err)
		}
		oldDbPath = filepath.Join(homeDir, ".ttl", "data.db")
	}

	if cfg.HasSection("workspaces.default") {
		return nil
	}

	defaultSec := cfg.Section("workspaces.default")
	defaultSec.Key("db_path").SetValue(oldDbPath)
	defaultSec.Key("storage_type").SetValue(storageType)

	workspaceSec := cfg.Section("")
	if !workspaceSec.HasKey("workspace") {
		workspaceSec.Key("workspace").SetValue("default")
	}

	return cfg.SaveTo(path)
}
