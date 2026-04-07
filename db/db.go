package db

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"ttl-cli/conf"
	"ttl-cli/crypto"
	"ttl-cli/models"

	"go.etcd.io/bbolt"
)

type Storage interface {
	Init() error
	Close() error
	GetAllResources() (map[models.ValJsonKey]models.ValJson, error)
	SaveResource(key models.ValJsonKey, value models.ValJson) error
	DeleteResource(key models.ValJsonKey) error
	UpdateResource(key models.ValJsonKey, newValue models.ValJson) error
	GetTagStats() ([]models.TagStat, error)
	SaveAuditRecord(record models.AuditRecord) error
	GetAuditStats() (models.AuditStats, error)
	GetAllAuditRecords() ([]models.AuditRecord, error)
	DeleteAuditRecords(resourceKey string) error
	SaveHistoryRecord(record models.HistoryRecord) error
	GetAllHistoryRecords() ([]models.HistoryRecord, error)
	GetHistoryRecord(index int, order models.SortOrder) (models.HistoryRecord, error)
	GetHistoryStats() (models.HistoryStats, error)
	DeleteHistoryRecords(resourceKey string) error
	SaveLogRecord(record models.LogRecord) error
	GetLogRecords(startDate, endDate string) ([]models.LogRecord, error)
	DeleteLogRecord(id int64) error
	SaveChatMessage(sessionID string, message models.ChatMessage) error
	GetChatMessages(sessionID string) ([]models.ChatMessage, error)
	ClearChatMessages(sessionID string) error
	GetSessionMeta(sessionID string) (*models.SessionMeta, error)
	UpdateSessionMeta(sessionID string, lastActive int64) error
}

type LocalStorage struct {
	db            *bbolt.DB
	dbPath        string
	confFile      string
	encryptionKey []byte
	encrypted     bool
	timeout       int
}

func NewLocalStorage() *LocalStorage {
	return &LocalStorage{}
}

func (ls *LocalStorage) SetDBPath(path string) {
	ls.dbPath = path
}

func (ls *LocalStorage) SetTimeout(timeout int) {
	ls.timeout = timeout
}

func (ls *LocalStorage) Init() error {
	if ls.dbPath == "" {
		dbPath, err := GetDBPath(ls.confFile, "local")
		if err != nil {
			return err
		}
		ls.dbPath = dbPath
	}

	if err := os.MkdirAll(filepath.Dir(ls.dbPath), 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	timeoutSec := ls.timeout
	if timeoutSec <= 0 {
		timeoutSec = 5
	}
	db, err := bbolt.Open(ls.dbPath, 0600, &bbolt.Options{Timeout: time.Duration(timeoutSec) * time.Second})
	if err != nil {
		if err == bbolt.ErrTimeout {
			return fmt.Errorf("数据库文件被锁定，请检查是否有其他 ttl 进程正在运行")
		}
		return fmt.Errorf("打开数据库失败: %w", err)
	}
	ls.db = db

	key, err := crypto.LoadKey()
	if err != nil {
		return fmt.Errorf("加载密钥失败: %w", err)
	}
	if key != nil {
		ls.encryptionKey = key
		ls.encrypted = true
	}

	return ls.db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("resources"))
		return err
	})
}

func (ls *LocalStorage) Close() error {
	if ls.db != nil {
		return ls.db.Close()
	}
	return nil
}

func (ls *LocalStorage) GetAllResources() (map[models.ValJsonKey]models.ValJson, error) {
	resources := make(map[models.ValJsonKey]models.ValJson)
	var resourceList []struct {
		key models.ValJsonKey
		val models.ValJson
	}

	err := ls.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("resources"))
		if bucket == nil {
			return errors.New("资源桶不存在")
		}

		return bucket.ForEach(func(k, v []byte) error {
			var key models.ValJsonKey
			if err := json.Unmarshal(k, &key); err != nil {
				return fmt.Errorf("解析key失败: %w", err)
			}

			var val models.ValJson
			if err := json.Unmarshal(v, &val); err != nil {
				return fmt.Errorf("解析value失败: %w", err)
			}

			if ls.encrypted && crypto.IsEncrypted(val.Val) {
				decrypted, err := crypto.Decrypt(ls.encryptionKey, val.Val)
				if err != nil {
					return fmt.Errorf("解密val失败: %w", err)
				}
				val.Val = decrypted
			}

			resourceList = append(resourceList, struct {
				key models.ValJsonKey
				val models.ValJson
			}{key, val})
			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	sort.Slice(resourceList, func(i, j int) bool {
		return resourceList[i].val.CreatedAt > resourceList[j].val.CreatedAt
	})

	for _, item := range resourceList {
		resources[item.key] = item.val
	}

	return resources, nil
}

func (ls *LocalStorage) SaveResource(key models.ValJsonKey, value models.ValJson) error {
	return ls.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("resources"))
		if bucket == nil {
			return errors.New("资源桶不存在")
		}

		keyBytes, err := json.Marshal(key)
		if err != nil {
			return fmt.Errorf("序列化key失败: %w", err)
		}

		existingVal := bucket.Get(keyBytes)
		now := time.Now().Unix()

		var saveValue models.ValJson
		if existingVal != nil {
			var existing models.ValJson
			if err := json.Unmarshal(existingVal, &existing); err == nil {
				saveValue = value
				saveValue.CreatedAt = existing.CreatedAt
				saveValue.UpdatedAt = now
			} else {
				saveValue = value
				saveValue.CreatedAt = now
				saveValue.UpdatedAt = now
			}
		} else {
			saveValue = value
			saveValue.CreatedAt = now
			saveValue.UpdatedAt = now
		}

		if ls.encrypted {
			encryptedVal, err := crypto.Encrypt(ls.encryptionKey, saveValue.Val)
			if err != nil {
				return fmt.Errorf("加密val失败: %w", err)
			}
			saveValue = models.ValJson{Val: encryptedVal, Tag: saveValue.Tag, CreatedAt: saveValue.CreatedAt, UpdatedAt: saveValue.UpdatedAt}
		}

		valBytes, err := json.Marshal(saveValue)
		if err != nil {
			return fmt.Errorf("序列化value失败: %w", err)
		}

		return bucket.Put(keyBytes, valBytes)
	})
}

func (ls *LocalStorage) DeleteResource(key models.ValJsonKey) error {
	return ls.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("resources"))
		if bucket == nil {
			return errors.New("资源桶不存在")
		}

		keyBytes, err := json.Marshal(key)
		if err != nil {
			return fmt.Errorf("序列化key失败: %w", err)
		}

		return bucket.Delete(keyBytes)
	})
}

func (ls *LocalStorage) UpdateResource(key models.ValJsonKey, newValue models.ValJson) error {
	return ls.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("resources"))
		if bucket == nil {
			return errors.New("资源桶不存在")
		}

		keyBytes, err := json.Marshal(key)
		if err != nil {
			return fmt.Errorf("序列化key失败: %w", err)
		}

		saveValue := newValue
		if ls.encrypted {
			encryptedVal, err := crypto.Encrypt(ls.encryptionKey, newValue.Val)
			if err != nil {
				return fmt.Errorf("加密val失败: %w", err)
			}
			saveValue = models.ValJson{Val: encryptedVal, Tag: newValue.Tag}
		}

		valBytes, err := json.Marshal(saveValue)
		if err != nil {
			return fmt.Errorf("序列化value失败: %w", err)
		}

		return bucket.Put(keyBytes, valBytes)
	})
}

func (ls *LocalStorage) GetTagStats() ([]models.TagStat, error) {
	resources, err := ls.GetAllResources()
	if err != nil {
		return nil, err
	}

	tagMap := make(map[string]models.TagStat)

	for key, val := range resources {
		if key.Type != models.ORIGIN {
			continue
		}

		for _, tag := range val.Tag {
			if stat, exists := tagMap[tag]; exists {
				stat.Count++
				stat.ResourceKeys = append(stat.ResourceKeys, key.Key)
				tagMap[tag] = stat
			} else {
				tagMap[tag] = models.TagStat{
					Tag:          tag,
					Count:        1,
					ResourceKeys: []string{key.Key},
				}
			}
		}
	}

	stats := make([]models.TagStat, 0, len(tagMap))
	for _, stat := range tagMap {
		stats = append(stats, stat)
	}

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Tag < stats[j].Tag
	})

	return stats, nil
}

func (ls *LocalStorage) IsEncryptionEnabled() bool {
	return ls.encrypted
}

func (ls *LocalStorage) EnableEncryption() error {
	if ls.encrypted {
		return errors.New("加密已启用")
	}

	key, err := crypto.LoadKey()
	if err != nil {
		return fmt.Errorf("加载密钥失败: %w", err)
	}
	if key == nil {
		key, err = crypto.GenerateKey()
		if err != nil {
			return fmt.Errorf("生成密钥失败: %w", err)
		}
		if err := crypto.SaveKey(key); err != nil {
			return fmt.Errorf("保存密钥失败: %w", err)
		}
	}

	ls.encryptionKey = key
	ls.encrypted = true

	return ls.migrateToEncrypted()
}

func (ls *LocalStorage) DisableEncryption() error {
	if !ls.encrypted {
		return errors.New("加密未启用")
	}

	err := ls.migrateToPlain()
	if err != nil {
		return fmt.Errorf("解密数据失败: %w", err)
	}

	if err := crypto.DeleteKey(); err != nil {
		return fmt.Errorf("删除密钥失败: %w", err)
	}

	ls.encryptionKey = nil
	ls.encrypted = false
	return nil
}

func (ls *LocalStorage) migrateToEncrypted() error {
	resources := make(map[models.ValJsonKey]models.ValJson)

	err := ls.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("resources"))
		if bucket == nil {
			return nil
		}

		return bucket.ForEach(func(k, v []byte) error {
			var key models.ValJsonKey
			if err := json.Unmarshal(k, &key); err != nil {
				return err
			}

			var val models.ValJson
			if err := json.Unmarshal(v, &val); err != nil {
				return err
			}

			if !crypto.IsEncrypted(val.Val) {
				resources[key] = val
			}
			return nil
		})
	})

	if err != nil {
		return err
	}

	if len(resources) == 0 {
		return nil
	}

	for key, value := range resources {
		if err := ls.SaveResource(key, value); err != nil {
			return fmt.Errorf("加密资源 [%s] 失败: %w", key.Key, err)
		}
	}

	return nil
}

func (ls *LocalStorage) migrateToPlain() error {
	resources := make(map[models.ValJsonKey]models.ValJson)

	err := ls.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("resources"))
		if bucket == nil {
			return nil
		}

		return bucket.ForEach(func(k, v []byte) error {
			var key models.ValJsonKey
			if err := json.Unmarshal(k, &key); err != nil {
				return err
			}

			var val models.ValJson
			if err := json.Unmarshal(v, &val); err != nil {
				return err
			}

			if crypto.IsEncrypted(val.Val) {
				resources[key] = val
			}
			return nil
		})
	})

	if err != nil {
		return err
	}

	if len(resources) == 0 {
		return nil
	}

	oldKey := ls.encryptionKey
	ls.encryptionKey = nil
	ls.encrypted = false

	for key, value := range resources {
		decryptedVal, err := crypto.Decrypt(oldKey, value.Val)
		if err != nil {
			return fmt.Errorf("解密资源 [%s] 失败: %w", key.Key, err)
		}
		value.Val = decryptedVal
		if err := ls.SaveResource(key, value); err != nil {
			return fmt.Errorf("保存资源 [%s] 失败: %w", key.Key, err)
		}
	}

	return nil
}

type CloudStorage struct {
	apiURL     string
	apiKey     string
	timeout    int
	httpClient *http.Client
}

type cloudAPIResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type cloudResourceDTO struct {
	Key       string   `json:"key"`
	Value     string   `json:"value"`
	Tags      []string `json:"tags"`
	CreatedAt int64    `json:"createdAt"`
	UpdatedAt int64    `json:"updatedAt"`
}

func NewCloudStorage(apiURL, apiKey string, timeout int) *CloudStorage {
	return &CloudStorage{
		apiURL:  strings.TrimRight(apiURL, "/"),
		apiKey:  apiKey,
		timeout: timeout,
	}
}

func (cs *CloudStorage) Init() error {
	cs.httpClient = &http.Client{
		Timeout: time.Duration(cs.timeout) * time.Second,
	}
	return nil
}

func (cs *CloudStorage) Close() error {
	if cs.httpClient != nil {
		cs.httpClient.CloseIdleConnections()
	}
	return nil
}

func (cs *CloudStorage) doRequest(method, path string, body any) (*cloudAPIResponse, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("序列化请求体失败: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, cs.apiURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if cs.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+cs.apiKey)
	}

	resp, err := cs.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var apiResp cloudAPIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, body: %s", err, string(respBody))
	}

	if apiResp.Code != 0 {
		return &apiResp, fmt.Errorf("API 错误: %s", apiResp.Message)
	}

	return &apiResp, nil
}

func (cs *CloudStorage) GetAllResources() (map[models.ValJsonKey]models.ValJson, error) {
	apiResp, err := cs.doRequest(http.MethodGet, "/api/v1/resources", nil)
	if err != nil {
		return nil, err
	}

	var dtos []cloudResourceDTO
	if err := json.Unmarshal(apiResp.Data, &dtos); err != nil {
		return nil, fmt.Errorf("解析资源列表失败: %w", err)
	}

	sort.Slice(dtos, func(i, j int) bool {
		return dtos[i].CreatedAt > dtos[j].CreatedAt
	})

	resources := make(map[models.ValJsonKey]models.ValJson, len(dtos))
	for _, dto := range dtos {
		key := models.ValJsonKey{Key: dto.Key, Type: models.ORIGIN}
		tags := dto.Tags
		if tags == nil {
			tags = []string{}
		}
		resources[key] = models.ValJson{
			Val:       dto.Value,
			Tag:       tags,
			CreatedAt: dto.CreatedAt,
			UpdatedAt: dto.UpdatedAt,
		}
	}
	return resources, nil
}

func (cs *CloudStorage) SaveResource(key models.ValJsonKey, value models.ValJson) error {
	now := time.Now().Unix()
	body := map[string]any{
		"key":       key.Key,
		"value":     value.Val,
		"tags":      value.Tag,
		"createdAt": value.CreatedAt,
		"updatedAt": now,
	}
	_, err := cs.doRequest(http.MethodPost, "/api/v1/resources", body)
	return err
}

func (cs *CloudStorage) DeleteResource(key models.ValJsonKey) error {
	_, err := cs.doRequest(http.MethodDelete, "/api/v1/resources/"+key.Key, nil)
	return err
}

func (cs *CloudStorage) UpdateResource(key models.ValJsonKey, newValue models.ValJson) error {
	body := map[string]string{"value": newValue.Val}
	_, err := cs.doRequest(http.MethodPut, "/api/v1/resources/"+key.Key, body)
	return err
}

func (cs *CloudStorage) GetTagStats() ([]models.TagStat, error) {
	resources, err := cs.GetAllResources()
	if err != nil {
		return nil, err
	}

	tagMap := make(map[string]models.TagStat)

	for key, val := range resources {
		if key.Type != models.ORIGIN {
			continue
		}

		for _, tag := range val.Tag {
			if stat, exists := tagMap[tag]; exists {
				stat.Count++
				stat.ResourceKeys = append(stat.ResourceKeys, key.Key)
				tagMap[tag] = stat
			} else {
				tagMap[tag] = models.TagStat{
					Tag:          tag,
					Count:        1,
					ResourceKeys: []string{key.Key},
				}
			}
		}
	}

	stats := make([]models.TagStat, 0, len(tagMap))
	for _, stat := range tagMap {
		stats = append(stats, stat)
	}

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Tag < stats[j].Tag
	})

	return stats, nil
}

type SyncStorage struct {
	localStorage Storage
	cloudStorage Storage
}

func NewSyncStorage(local Storage, cloud Storage) *SyncStorage {
	return &SyncStorage{
		localStorage: local,
		cloudStorage: cloud,
	}
}

func (ss *SyncStorage) Init() error {
	if err := ss.localStorage.Init(); err != nil {
		return err
	}
	return ss.cloudStorage.Init()
}

func (ss *SyncStorage) Close() error {
	if err := ss.localStorage.Close(); err != nil {
		return err
	}
	return ss.cloudStorage.Close()
}

func (ss *SyncStorage) GetAllResources() (map[models.ValJsonKey]models.ValJson, error) {
	return ss.localStorage.GetAllResources()
}

func (ss *SyncStorage) SaveResource(key models.ValJsonKey, value models.ValJson) error {
	if err := ss.localStorage.SaveResource(key, value); err != nil {
		return err
	}
	return ss.cloudStorage.SaveResource(key, value)
}

func (ss *SyncStorage) DeleteResource(key models.ValJsonKey) error {
	if err := ss.localStorage.DeleteResource(key); err != nil {
		return err
	}
	return ss.cloudStorage.DeleteResource(key)
}

func (ss *SyncStorage) UpdateResource(key models.ValJsonKey, newValue models.ValJson) error {
	if err := ss.localStorage.UpdateResource(key, newValue); err != nil {
		return err
	}
	return ss.cloudStorage.UpdateResource(key, newValue)
}

func (ss *SyncStorage) GetTagStats() ([]models.TagStat, error) {
	return ss.localStorage.GetTagStats()
}

func GetDBPath(confFile string, storageType string) (string, error) {
	var (
		ttlConf models.TtlIni
		err     error
	)
	if confFile != "" {
		ttlConf, err = conf.GetTtlConfFromFile(confFile)
	} else {
		ttlConf, err = conf.GetTtlConf()
	}
	if err != nil {
		return "", err
	}

	workspaceName := ttlConf.Workspace
	if workspaceName == "" {
		workspaceName = "default"
	}

	var baseDir string
	var dbPath string

	if ws, ok := ttlConf.Workspaces[workspaceName]; ok && ws.DbPath != "" {
		return ws.DbPath, nil
	} else if ttlConf.DbPath != "" {
		dbPath = ttlConf.DbPath
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("获取用户目录失败: %w", err)
		}
		baseDir = filepath.Join(homeDir, ".ttl")

		switch storageType {
		case "sqlite":
			return filepath.Join(baseDir, "data.db"), nil
		case "local", "bbolt":
			return filepath.Join(baseDir, "data.bbolt"), nil
		default:
			return filepath.Join(baseDir, "data.db"), nil
		}
	}

	if dbPath != "" {
		ext := filepath.Ext(dbPath)
		basePath := dbPath[:len(dbPath)-len(ext)]

		switch storageType {
		case "sqlite":
			return basePath + ".db", nil
		case "local", "bbolt":
			return basePath + ".bbolt", nil
		default:
			return basePath + ".db", nil
		}
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户目录失败: %w", err)
	}
	baseDir = filepath.Join(homeDir, ".ttl")

	switch storageType {
	case "sqlite":
		return filepath.Join(baseDir, "data.db"), nil
	case "local", "bbolt":
		return filepath.Join(baseDir, "data.bbolt"), nil
	default:
		return filepath.Join(baseDir, "data.db"), nil
	}
}

func (ls *LocalStorage) SaveAuditRecord(record models.AuditRecord) error {
	return ls.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("audit"))
		if bucket == nil {
			bucket, _ = tx.CreateBucket([]byte("audit"))
		}

		key := fmt.Sprintf("%s_%s_%d", record.ResourceKey, record.Operation, record.Timestamp)
		valBytes, err := json.Marshal(record)
		if err != nil {
			return fmt.Errorf("序列化审计记录失败: %w", err)
		}

		return bucket.Put([]byte(key), valBytes)
	})
}

func (ls *LocalStorage) GetAuditStats() (models.AuditStats, error) {
	stats := models.AuditStats{
		ByOperation: make(map[string]int),
		ByResource:  make(map[string]int),
	}

	err := ls.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("audit"))
		if bucket == nil {
			return nil
		}

		return bucket.ForEach(func(k, v []byte) error {
			var record models.AuditRecord
			if err := json.Unmarshal(v, &record); err != nil {
				return fmt.Errorf("解析审计记录失败: %w", err)
			}

			stats.TotalOperations += record.Count
			stats.ByOperation[record.Operation] += record.Count
			stats.ByResource[record.ResourceKey] += record.Count

			return nil
		})
	})

	return stats, err
}

func (ls *LocalStorage) GetAllAuditRecords() ([]models.AuditRecord, error) {
	var records []models.AuditRecord

	err := ls.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("audit"))
		if bucket == nil {
			return nil
		}
		return bucket.ForEach(func(_, v []byte) error {
			var record models.AuditRecord
			if err := json.Unmarshal(v, &record); err != nil {
				return fmt.Errorf("解析审计记录失败: %w", err)
			}
			records = append(records, record)
			return nil
		})
	})
	return records, err
}

func (ls *LocalStorage) DeleteAuditRecords(resourceKey string) error {
	return ls.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("audit"))
		if bucket == nil {
			return nil
		}

		var toDelete [][]byte
		err := bucket.ForEach(func(k, v []byte) error {
			var record models.AuditRecord
			if err := json.Unmarshal(v, &record); err != nil {
				return err
			}

			if record.ResourceKey == resourceKey {
				toDelete = append(toDelete, k)
			}
			return nil
		})

		if err != nil {
			return err
		}

		for _, key := range toDelete {
			if err := bucket.Delete(key); err != nil {
				return err
			}
		}

		return nil
	})
}

func (cs *CloudStorage) SaveAuditRecord(_ models.AuditRecord) error {
	return nil
}

func (cs *CloudStorage) GetAuditStats() (models.AuditStats, error) {
	apiResp, err := cs.doRequest(http.MethodGet, "/api/v1/audit/stats", nil)
	if err != nil {
		return models.AuditStats{ByOperation: make(map[string]int), ByResource: make(map[string]int)}, err
	}
	var stats models.AuditStats
	if err := json.Unmarshal(apiResp.Data, &stats); err != nil {
		return models.AuditStats{ByOperation: make(map[string]int), ByResource: make(map[string]int)}, err
	}
	return stats, nil
}

func (cs *CloudStorage) DeleteAuditRecords(_ string) error {
	return nil
}

func (cs *CloudStorage) GetAllAuditRecords() ([]models.AuditRecord, error) {
	return []models.AuditRecord{}, nil
}

func (ss *SyncStorage) SaveAuditRecord(record models.AuditRecord) error {
	if err := ss.localStorage.SaveAuditRecord(record); err != nil {
		return err
	}
	return ss.cloudStorage.SaveAuditRecord(record)
}

func (ss *SyncStorage) GetAuditStats() (models.AuditStats, error) {
	return ss.localStorage.GetAuditStats()
}

func (ss *SyncStorage) GetAllAuditRecords() ([]models.AuditRecord, error) {
	return ss.localStorage.GetAllAuditRecords()
}

func (ss *SyncStorage) DeleteAuditRecords(resourceKey string) error {
	if err := ss.localStorage.DeleteAuditRecords(resourceKey); err != nil {
		return err
	}
	return ss.cloudStorage.DeleteAuditRecords(resourceKey)
}

func (ls *LocalStorage) SaveHistoryRecord(record models.HistoryRecord) error {
	return ls.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("history"))
		if bucket == nil {
			bucket, _ = tx.CreateBucket([]byte("history"))
		}

		key := fmt.Sprintf("%d_%d", record.Timestamp, record.ID)
		valBytes, err := json.Marshal(record)
		if err != nil {
			return fmt.Errorf("序列化历史记录失败: %w", err)
		}

		return bucket.Put([]byte(key), valBytes)
	})
}

func (ls *LocalStorage) GetAllHistoryRecords() ([]models.HistoryRecord, error) {
	var records []models.HistoryRecord

	err := ls.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("history"))
		if bucket == nil {
			return nil
		}

		return bucket.ForEach(func(k, v []byte) error {
			var record models.HistoryRecord
			if err := json.Unmarshal(v, &record); err != nil {
				return fmt.Errorf("解析历史记录失败: %w", err)
			}
			records = append(records, record)
			return nil
		})
	})

	sort.Slice(records, func(i, j int) bool {
		return records[i].Timestamp > records[j].Timestamp
	})

	return records, err
}

func (ls *LocalStorage) GetHistoryRecord(index int, order models.SortOrder) (models.HistoryRecord, error) {
	var record models.HistoryRecord
	var found bool

	err := ls.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("history"))
		if bucket := tx.Bucket([]byte("history")); bucket == nil {
			return fmt.Errorf("history bucket does not exist")
		}

		cursor := bucket.Cursor()

		total := 0
		for k, _ := cursor.First(); k != nil; k, _ = cursor.Next() {
			total++
		}

		if index < 0 || index >= total {
			return nil
		}

		var targetIndex int
		if order == models.Descending {
			targetIndex = total - 1 - index
		} else if order == models.Ascending {
			targetIndex = index
		}

		idx := 0
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			if idx == targetIndex {
				if err := json.Unmarshal(v, &record); err != nil {
					return fmt.Errorf("failed to unmarshal record: %w", err)
				}
				found = true
				return nil
			}
			idx++
		}

		return nil
	})

	if err != nil {
		return models.HistoryRecord{}, err
	}

	if !found {
		return models.HistoryRecord{}, fmt.Errorf("index %d out of bounds", index)
	}

	return record, nil
}

func (ls *LocalStorage) GetHistoryStats() (models.HistoryStats, error) {
	stats := models.HistoryStats{
		ByOperation: make(map[string]int),
		ByResource:  make(map[string]int),
	}

	records, err := ls.GetAllHistoryRecords()
	if err != nil {
		return stats, err
	}

	stats.TotalRecords = len(records)
	stats.Records = records

	for _, record := range records {
		stats.ByOperation[record.Operation]++
		stats.ByResource[record.ResourceKey]++
	}

	return stats, nil
}

func (ls *LocalStorage) DeleteHistoryRecords(resourceKey string) error {
	return ls.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("history"))
		if bucket == nil {
			return nil
		}

		var toDelete [][]byte
		err := bucket.ForEach(func(k, v []byte) error {
			var record models.HistoryRecord
			if err := json.Unmarshal(v, &record); err != nil {
				return err
			}

			if record.ResourceKey == resourceKey {
				toDelete = append(toDelete, k)
			}
			return nil
		})

		if err != nil {
			return err
		}

		for _, key := range toDelete {
			if err := bucket.Delete(key); err != nil {
				return err
			}
		}

		return nil
	})
}

func (cs *CloudStorage) SaveHistoryRecord(_ models.HistoryRecord) error {
	return nil
}

func (cs *CloudStorage) GetAllHistoryRecords() ([]models.HistoryRecord, error) {
	apiResp, err := cs.doRequest(http.MethodGet, "/api/v1/history", nil)
	if err != nil {
		return []models.HistoryRecord{}, err
	}
	var records []models.HistoryRecord
	if err := json.Unmarshal(apiResp.Data, &records); err != nil {
		return []models.HistoryRecord{}, err
	}
	return records, nil
}

func (cs *CloudStorage) GetHistoryRecord(_ int, _ models.SortOrder) (models.HistoryRecord, error) {
	return models.HistoryRecord{}, fmt.Errorf("云端存储不支持按索引获取历史记录")
}

func (cs *CloudStorage) GetHistoryStats() (models.HistoryStats, error) {
	return models.HistoryStats{
		ByOperation: make(map[string]int),
		ByResource:  make(map[string]int),
	}, nil
}

func (cs *CloudStorage) DeleteHistoryRecords(_ string) error {
	return nil
}

func (ss *SyncStorage) SaveHistoryRecord(record models.HistoryRecord) error {
	if err := ss.localStorage.SaveHistoryRecord(record); err != nil {
		return err
	}
	return ss.cloudStorage.SaveHistoryRecord(record)
}

func (ss *SyncStorage) GetAllHistoryRecords() ([]models.HistoryRecord, error) {
	return ss.localStorage.GetAllHistoryRecords()
}

func (ss *SyncStorage) GetHistoryRecord(index int, order models.SortOrder) (models.HistoryRecord, error) {
	return ss.localStorage.GetHistoryRecord(index, order)
}

func (ss *SyncStorage) GetHistoryStats() (models.HistoryStats, error) {
	return ss.localStorage.GetHistoryStats()
}

func (ss *SyncStorage) DeleteHistoryRecords(resourceKey string) error {
	if err := ss.localStorage.DeleteHistoryRecords(resourceKey); err != nil {
		return err
	}
	return ss.cloudStorage.DeleteHistoryRecords(resourceKey)
}

func (ls *LocalStorage) SaveLogRecord(record models.LogRecord) error {
	return ls.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("logs"))
		if bucket == nil {
			bucket, _ = tx.CreateBucket([]byte("logs"))
		}

		key := fmt.Sprintf("%s_%d", record.Date, record.ID)
		valBytes, err := json.Marshal(record)
		if err != nil {
			return fmt.Errorf("序列化日志记录失败: %w", err)
		}

		return bucket.Put([]byte(key), valBytes)
	})
}

func (ls *LocalStorage) GetLogRecords(startDate, endDate string) ([]models.LogRecord, error) {
	var records []models.LogRecord

	err := ls.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("logs"))
		if bucket == nil {
			return nil
		}

		return bucket.ForEach(func(k, v []byte) error {
			var record models.LogRecord
			if err := json.Unmarshal(v, &record); err != nil {
				return fmt.Errorf("解析日志记录失败: %w", err)
			}

			if startDate != "" && record.Date < startDate {
				return nil
			}
			if endDate != "" && record.Date > endDate {
				return nil
			}

			records = append(records, record)
			return nil
		})
	})

	sort.Slice(records, func(i, j int) bool {
		return records[i].ID > records[j].ID
	})

	return records, err
}

func (ls *LocalStorage) DeleteLogRecord(id int64) error {
	return ls.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("logs"))
		if bucket == nil {
			return fmt.Errorf("未找到该日志记录")
		}

		var targetKey []byte
		err := bucket.ForEach(func(k, v []byte) error {
			var record models.LogRecord
			if err := json.Unmarshal(v, &record); err != nil {
				return err
			}
			if record.ID == id {
				targetKey = make([]byte, len(k))
				copy(targetKey, k)
			}
			return nil
		})
		if err != nil {
			return err
		}

		if targetKey == nil {
			return fmt.Errorf("未找到该日志记录")
		}
		return bucket.Delete(targetKey)
	})
}

func (cs *CloudStorage) SaveLogRecord(_ models.LogRecord) error {
	return nil
}

func (cs *CloudStorage) GetLogRecords(_, _ string) ([]models.LogRecord, error) {
	return []models.LogRecord{}, nil
}

func (cs *CloudStorage) DeleteLogRecord(_ int64) error {
	return nil
}

func (ss *SyncStorage) SaveLogRecord(record models.LogRecord) error {
	if err := ss.localStorage.SaveLogRecord(record); err != nil {
		return err
	}
	return ss.cloudStorage.SaveLogRecord(record)
}

func (ss *SyncStorage) GetLogRecords(startDate, endDate string) ([]models.LogRecord, error) {
	return ss.localStorage.GetLogRecords(startDate, endDate)
}

func (ss *SyncStorage) DeleteLogRecord(id int64) error {
	if err := ss.localStorage.DeleteLogRecord(id); err != nil {
		return err
	}
	return ss.cloudStorage.DeleteLogRecord(id)
}

func (ls *LocalStorage) SaveChatMessage(sessionID string, message models.ChatMessage) error {
	return ls.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("chats"))
		if bucket == nil {
			bucket, _ = tx.CreateBucket([]byte("chats"))
		}

		key := fmt.Sprintf("%s_%d", sessionID, message.Timestamp)
		valBytes, err := json.Marshal(message)
		if err != nil {
			return fmt.Errorf("序列化聊天消息失败: %w", err)
		}

		return bucket.Put([]byte(key), valBytes)
	})
}

func (ls *LocalStorage) GetChatMessages(sessionID string) ([]models.ChatMessage, error) {
	var messages []models.ChatMessage

	err := ls.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("chats"))
		if bucket == nil {
			return nil
		}

		prefix := []byte(sessionID + "_")
		cursor := bucket.Cursor()
		for k, v := cursor.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = cursor.Next() {
			var msg models.ChatMessage
			if err := json.Unmarshal(v, &msg); err != nil {
				continue
			}
			messages = append(messages, msg)
		}
		return nil
	})

	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Timestamp < messages[j].Timestamp
	})

	return messages, err
}

func (ls *LocalStorage) ClearChatMessages(sessionID string) error {
	return ls.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("chats"))
		if bucket == nil {
			return nil
		}

		var toDelete [][]byte
		prefix := []byte(sessionID + "_")
		err := bucket.ForEach(func(k, v []byte) error {
			if bytes.HasPrefix(k, prefix) {
				toDelete = append(toDelete, k)
			}
			return nil
		})

		if err != nil {
			return err
		}

		for _, key := range toDelete {
			if err := bucket.Delete(key); err != nil {
				return err
			}
		}

		return nil
	})
}

func (ls *LocalStorage) GetSessionMeta(sessionID string) (*models.SessionMeta, error) {
	var meta models.SessionMeta

	err := ls.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("sessions"))
		if bucket == nil {
			return fmt.Errorf("会话不存在")
		}

		val := bucket.Get([]byte(sessionID))
		if val == nil {
			return fmt.Errorf("会话不存在")
		}

		return json.Unmarshal(val, &meta)
	})

	if err != nil {
		return nil, err
	}

	return &meta, nil
}

func (ls *LocalStorage) UpdateSessionMeta(sessionID string, lastActive int64) error {
	return ls.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("sessions"))
		if bucket == nil {
			bucket, _ = tx.CreateBucket([]byte("sessions"))
		}

		meta := models.SessionMeta{
			SessionID:  sessionID,
			LastActive: lastActive,
		}

		valBytes, err := json.Marshal(meta)
		if err != nil {
			return fmt.Errorf("序列化会话元数据失败: %w", err)
		}

		return bucket.Put([]byte(sessionID), valBytes)
	})
}

func (cs *CloudStorage) SaveChatMessage(_ string, _ models.ChatMessage) error {
	return nil
}

func (cs *CloudStorage) GetChatMessages(_ string) ([]models.ChatMessage, error) {
	return []models.ChatMessage{}, nil
}

func (cs *CloudStorage) ClearChatMessages(_ string) error {
	return nil
}

func (cs *CloudStorage) GetSessionMeta(_ string) (*models.SessionMeta, error) {
	return nil, fmt.Errorf("云端存储不支持会话元数据")
}

func (cs *CloudStorage) UpdateSessionMeta(_ string, _ int64) error {
	return nil
}

func (ss *SyncStorage) SaveChatMessage(sessionID string, message models.ChatMessage) error {
	if err := ss.localStorage.SaveChatMessage(sessionID, message); err != nil {
		return err
	}
	return ss.cloudStorage.SaveChatMessage(sessionID, message)
}

func (ss *SyncStorage) GetChatMessages(sessionID string) ([]models.ChatMessage, error) {
	return ss.localStorage.GetChatMessages(sessionID)
}

func (ss *SyncStorage) ClearChatMessages(sessionID string) error {
	if err := ss.localStorage.ClearChatMessages(sessionID); err != nil {
		return err
	}
	return ss.cloudStorage.ClearChatMessages(sessionID)
}

func (ss *SyncStorage) GetSessionMeta(sessionID string) (*models.SessionMeta, error) {
	return ss.localStorage.GetSessionMeta(sessionID)
}

func (ss *SyncStorage) UpdateSessionMeta(sessionID string, lastActive int64) error {
	if err := ss.localStorage.UpdateSessionMeta(sessionID, lastActive); err != nil {
		return err
	}
	return ss.cloudStorage.UpdateSessionMeta(sessionID, lastActive)
}
