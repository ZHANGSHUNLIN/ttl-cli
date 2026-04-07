package db

import (
	"fmt"
	"time"
	"ttl-cli/conf"
	"ttl-cli/models"
)

var Stor Storage

func InitDB(storageType string, cloudAPIURL string, cloudAPIKey string, cloudTimeout int, confFile string) error {
	if Stor != nil {
		_ = Stor.Close()
	}

	var boltTimeout int
	if confFile == "" {
		defaultConfPath, err := conf.GetDefaultConfPath()
		if err == nil {
			ttlConf, err := conf.GetTtlConfFromFile(defaultConfPath)
			if err == nil {
				boltTimeout = ttlConf.BoltDB.Timeout
			}
		}
	} else {
		ttlConf, err := conf.GetTtlConfFromFile(confFile)
		if err == nil {
			boltTimeout = ttlConf.BoltDB.Timeout
		}
	}

	switch storageType {
	case "sqlite":
		sqliteStorage := NewSQLiteStorage()
		sqliteStorage.confFile = confFile
		Stor = sqliteStorage
	case "local", "bbolt":
		ls := NewLocalStorage()
		ls.confFile = confFile
		if boltTimeout > 0 {
			ls.SetTimeout(boltTimeout)
		}
		Stor = ls
	case "cloud":
		if cloudAPIURL == "" || cloudAPIKey == "" {
			return fmt.Errorf("cloud storage requires API URL and key")
		}
		Stor = NewCloudStorage(cloudAPIURL, cloudAPIKey, cloudTimeout)
	case "sync":
		ls := NewLocalStorage()
		ls.confFile = confFile
		if boltTimeout > 0 {
			ls.SetTimeout(boltTimeout)
		}
		cloud := NewCloudStorage(cloudAPIURL, cloudAPIKey, cloudTimeout)
		Stor = NewSyncStorage(ls, cloud)
	default:
		return fmt.Errorf("unsupported storage type: %s (supported: sqlite, local/bbolt, cloud, sync)", storageType)
	}

	return Stor.Init()
}

func CloseDB() error {
	if Stor != nil {
		return Stor.Close()
	}
	return nil
}

func GetAllResources() (map[models.ValJsonKey]models.ValJson, error) {
	if Stor == nil {
		return nil, fmt.Errorf("storage not initialized")
	}
	return Stor.GetAllResources()
}

func SaveResource(key models.ValJsonKey, value models.ValJson) error {
	if Stor == nil {
		return fmt.Errorf("storage not initialized")
	}
	return Stor.SaveResource(key, value)
}

func DeleteResource(key models.ValJsonKey) error {
	if Stor == nil {
		return fmt.Errorf("storage not initialized")
	}
	return Stor.DeleteResource(key)
}

func UpdateResource(key models.ValJsonKey, newValue models.ValJson) error {
	if Stor == nil {
		return fmt.Errorf("storage not initialized")
	}
	return Stor.UpdateResource(key, newValue)
}

func GetTagStats() ([]models.TagStat, error) {
	if Stor == nil {
		return nil, fmt.Errorf("storage not initialized")
	}
	return Stor.GetTagStats()
}

func MigrateData(sourceType, targetType, sourceAPIURL,
	sourceAPIKey string, sourceTimeout int, cloudAPIURL string,
	cloudAPIKey string, cloudTimeout int, debug bool,
	srcConfFile string, dstConfFile string) error {
	fmt.Printf("Starting data migration: %s -> %s\n", sourceType, targetType)

	var sourceStorage Storage
	switch sourceType {
	case "local":
		ls := NewLocalStorage()
		ls.confFile = srcConfFile
		sourceStorage = ls
	case "cloud":
		if sourceAPIURL == "" || sourceAPIKey == "" {
			return fmt.Errorf("source cloud storage requires API URL and key")
		}
		sourceStorage = NewCloudStorage(sourceAPIURL, sourceAPIKey, sourceTimeout)
	default:
		return fmt.Errorf("unsupported source storage type: %s", sourceType)
	}

	if err := sourceStorage.Init(); err != nil {
		return fmt.Errorf("failed to initialize source storage: %w", err)
	}
	defer func(sourceStorage Storage) {
		err := sourceStorage.Close()
		if err != nil {
		}
	}(sourceStorage)

	var targetStorage Storage
	switch targetType {
	case "local":
		ls := NewLocalStorage()
		ls.confFile = dstConfFile
		targetStorage = ls
	case "cloud":
		if cloudAPIURL == "" || cloudAPIKey == "" {
			return fmt.Errorf("target cloud storage requires API URL and key")
		}
		targetStorage = NewCloudStorage(cloudAPIURL, cloudAPIKey, cloudTimeout)
	default:
		return fmt.Errorf("unsupported target storage type: %s", targetType)
	}

	if err := targetStorage.Init(); err != nil {
		return fmt.Errorf("failed to initialize target storage: %w", err)
	}
	defer func(targetStorage Storage) {
		err := targetStorage.Close()
		if err != nil {
		}
	}(targetStorage)

	fmt.Println("Reading data from source storage...")
	resources, err := sourceStorage.GetAllResources()
	if err != nil {
		return fmt.Errorf("failed to read source data: %w", err)
	}

	fmt.Printf("Found %d resources to migrate\n", len(resources))

	successCount := 0
	failCount := 0

	for key, value := range resources {
		if err := targetStorage.SaveResource(key, value); err != nil {
			fmt.Printf("Failed to migrate resource [%s]: %v\n", key.Key, err)
			failCount++
		} else {
			successCount++
			if debug {
				fmt.Printf("Successfully migrated resource: %s\n", key.Key)
			}
		}
	}

	fmt.Printf("Migration completed! Success: %d, Failed: %d\n", successCount, failCount)

	if failCount > 0 {
		return fmt.Errorf("some resources migration failed")
	}

	return nil
}

func RecordAudit(resourceKey, operation string) error {
	if Stor == nil {
		return fmt.Errorf("storage not initialized")
	}

	record := models.AuditRecord{
		ResourceKey: resourceKey,
		Operation:   operation,
		Timestamp:   time.Now().Unix(),
		Count:       1,
	}

	switch s := Stor.(type) {
	case interface {
		SaveAuditRecord(models.AuditRecord) error
	}:
		return s.SaveAuditRecord(record)
	default:
		return fmt.Errorf("current storage does not support audit")
	}
}

func GetAuditStats() (models.AuditStats, error) {
	if Stor == nil {
		return models.AuditStats{}, fmt.Errorf("storage not initialized")
	}

	switch s := Stor.(type) {
	case interface {
		GetAuditStats() (models.AuditStats, error)
	}:
		return s.GetAuditStats()
	default:
		return models.AuditStats{}, fmt.Errorf("current storage does not support audit")
	}
}

func GetAllAuditRecords() ([]models.AuditRecord, error) {
	if Stor == nil {
		return nil, fmt.Errorf("storage not initialized")
	}
	return Stor.GetAllAuditRecords()
}

func DeleteAuditRecords(resourceKey string) error {
	if Stor == nil {
		return fmt.Errorf("storage not initialized")
	}

	switch s := Stor.(type) {
	case interface {
		DeleteAuditRecords(string) error
	}:
		return s.DeleteAuditRecords(resourceKey)
	default:
		return fmt.Errorf("current storage does not support audit")
	}
}

func GetAllHistoryRecords() ([]models.HistoryRecord, error) {
	if Stor == nil {
		return nil, fmt.Errorf("storage not initialized")
	}

	switch s := Stor.(type) {
	case interface {
		GetAllHistoryRecords() ([]models.HistoryRecord, error)
	}:
		return s.GetAllHistoryRecords()
	default:
		return nil, fmt.Errorf("current storage does not support history")
	}
}

func DeleteHistoryRecords(resourceKey string) error {
	if Stor == nil {
		return fmt.Errorf("storage not initialized")
	}

	switch s := Stor.(type) {
	case interface {
		DeleteHistoryRecords(string) error
	}:
		return s.DeleteHistoryRecords(resourceKey)
	default:
		return fmt.Errorf("current storage does not support history")
	}
}

func GetHistoryRecords(idx int) (models.HistoryRecord, error) {
	if Stor == nil {
		return models.HistoryRecord{}, fmt.Errorf("storage not initialized")
	}
	switch s := Stor.(type) {
	case interface {
		GetHistoryRecord(int, models.SortOrder) (models.HistoryRecord, error)
	}:
		return s.GetHistoryRecord(idx, models.Descending)
	default:
		return models.HistoryRecord{}, fmt.Errorf("current storage does not support history query")
	}
}

func RecordCommandHistory(operation, resourceKey string, debug bool) error {
	if Stor == nil {
		return fmt.Errorf("storage not initialized")
	}

	if operation == "completion" || operation == "help" || operation == "__complete" {
		return nil
	}

	record := models.HistoryRecord{
		ID:          time.Now().UnixNano(),
		ResourceKey: resourceKey,
		Operation:   operation,
		Timestamp:   time.Now().Unix(),
		TimeStr:     time.Now().Format("2006-01-02 15:04:05"),
		Command:     operation,
	}

	switch s := Stor.(type) {
	case interface {
		SaveHistoryRecord(models.HistoryRecord) error
	}:
		return s.SaveHistoryRecord(record)
	default:
		if debug {
			fmt.Printf("current storage does not support history\n")
		}
		return nil
	}
}

func CleanupResourceHistory(resourceKey string, debug bool) {
	if err := DeleteHistoryRecords(resourceKey); err != nil && debug {
		fmt.Printf("Failed to clean resource history: %v\n", err)
	}
	if err := DeleteAuditRecords(resourceKey); err != nil && debug {
		fmt.Printf("Failed to clean resource audit: %v\n", err)
	}
}

func SaveLogRecord(record models.LogRecord) error {
	if Stor == nil {
		return fmt.Errorf("storage not initialized")
	}
	return Stor.SaveLogRecord(record)
}

func GetLogRecords(startDate, endDate string) ([]models.LogRecord, error) {
	if Stor == nil {
		return nil, fmt.Errorf("storage not initialized")
	}
	return Stor.GetLogRecords(startDate, endDate)
}

func DeleteLogRecord(id int64) error {
	if Stor == nil {
		return fmt.Errorf("storage not initialized")
	}
	return Stor.DeleteLogRecord(id)
}

func SaveChatMessage(sessionID string, message models.ChatMessage) error {
	if Stor == nil {
		return fmt.Errorf("storage not initialized")
	}
	return Stor.SaveChatMessage(sessionID, message)
}

func GetChatMessages(sessionID string) ([]models.ChatMessage, error) {
	if Stor == nil {
		return nil, fmt.Errorf("storage not initialized")
	}
	return Stor.GetChatMessages(sessionID)
}

func ClearChatMessages(sessionID string) error {
	if Stor == nil {
		return fmt.Errorf("storage not initialized")
	}
	return Stor.ClearChatMessages(sessionID)
}

func GetSessionMeta(sessionID string) (*models.SessionMeta, error) {
	if Stor == nil {
		return nil, fmt.Errorf("storage not initialized")
	}
	return Stor.GetSessionMeta(sessionID)
}

func UpdateSessionMeta(sessionID string, lastActive int64) error {
	if Stor == nil {
		return fmt.Errorf("storage not initialized")
	}
	return Stor.UpdateSessionMeta(sessionID, lastActive)
}
