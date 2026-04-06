package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"ttl-cli/models"

	_ "modernc.org/sqlite"
)

type SQLiteStorage struct {
	db          *sql.DB
	dbPath      string
	confFile    string
	journalMode string // WAL | DELETE | TRUNCATE | PERSIST
	cacheSize   int    // 缓存大小
	busyTimeout int    // 繁忙等待毫秒数
	synchronous string // 同步模式
}

func NewSQLiteStorage() *SQLiteStorage {
	return &SQLiteStorage{
		journalMode: "WAL",
		cacheSize:   -64000, // 64MB
		busyTimeout: 5000,
		synchronous: "NORMAL",
	}
}

func (s *SQLiteStorage) SetDBPath(path string) {
	s.dbPath = path
}

func (s *SQLiteStorage) Init() error {
	if s.dbPath == "" {
		dbPath, err := GetDBPath(s.confFile, "sqlite")
		if err != nil {
			return err
		}
		s.dbPath = dbPath
	}

	if err := os.MkdirAll(filepath.Dir(s.dbPath), 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	var err error
	s.db, err = sql.Open("sqlite", s.dbPath+"?_pragma=journal_mode("+s.journalMode+")&_pragma=cache_size("+fmt.Sprintf("%d", s.cacheSize)+")&_pragma=busy_timeout("+fmt.Sprintf("%d", s.busyTimeout)+")&_pragma=synchronous("+s.synchronous+")")
	if err != nil {
		return fmt.Errorf("打开数据库失败: %w", err)
	}

	if err := s.db.Ping(); err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}

	return s.createTables()
}

func (s *SQLiteStorage) createTables() error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS resources (
			key TEXT NOT NULL,
			type TEXT NOT NULL,
			origin_key TEXT,
			value TEXT NOT NULL,
			tags TEXT,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			PRIMARY KEY (key, type)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_resources_created ON resources(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_resources_origin ON resources(type) WHERE type = 'ORIGIN'`,
		`CREATE INDEX IF NOT EXISTS idx_resources_tag ON resources(type) WHERE type = 'TAG'`,
		`CREATE TABLE IF NOT EXISTS audit (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			resource_key TEXT NOT NULL,
			operation TEXT NOT NULL,
			timestamp INTEGER NOT NULL,
			count INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_resource ON audit(resource_key)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit(timestamp)`,
		`CREATE TABLE IF NOT EXISTS history (
			id INTEGER PRIMARY KEY,
			resource_key TEXT,
			operation TEXT NOT NULL,
			timestamp INTEGER NOT NULL,
			time_str TEXT NOT NULL,
			command TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_history_timestamp ON history(timestamp)`,
		`CREATE TABLE IF NOT EXISTS logs (
			id INTEGER PRIMARY KEY,
			content TEXT NOT NULL,
			tags TEXT,
			created_at TEXT NOT NULL,
			date TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_logs_date ON logs(date)`,
		`CREATE TABLE IF NOT EXISTS chats (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			timestamp INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_chats_session_timestamp ON chats(session_id, timestamp)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			session_id TEXT PRIMARY KEY,
			last_active INTEGER NOT NULL
		)`,
	}

	for _, table := range tables {
		if _, err := s.db.Exec(table); err != nil {
			return fmt.Errorf("创建表失败: %w", err)
		}
	}
	return nil
}

func (s *SQLiteStorage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *SQLiteStorage) GetAllResources() (map[models.ValJsonKey]models.ValJson, error) {
	resources := make(map[models.ValJsonKey]models.ValJson)

	rows, err := s.db.Query("SELECT key, type, origin_key, value, tags, created_at, updated_at FROM resources ORDER BY created_at DESC")
	if err != nil {
		return nil, fmt.Errorf("查询资源失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var key, typ, originKey, value, tagsStr string
		var createdAt, updatedAt int64
		if err := rows.Scan(&key, &typ, &originKey, &value, &tagsStr, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("扫描资源行失败: %w", err)
		}

		var tags []string
		if tagsStr != "" {
			if err := json.Unmarshal([]byte(tagsStr), &tags); err != nil {
				return nil, fmt.Errorf("解析标签失败: %w", err)
			}
		}

		keyType := models.ORIGIN
		if typ == "TAG" {
			keyType = models.TAG
		}

		vjk := models.ValJsonKey{
			Key:       key,
			Type:      keyType,
			OriginKey: originKey,
		}

		resources[vjk] = models.ValJson{
			Val:       value,
			Tag:       tags,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		}
	}

	return resources, nil
}

func (s *SQLiteStorage) SaveResource(key models.ValJsonKey, value models.ValJson) error {
	typ := "ORIGIN"
	if key.Type == models.TAG {
		typ = "TAG"
	}

	tagsJSON, err := json.Marshal(value.Tag)
	if err != nil {
		return fmt.Errorf("序列化标签失败: %w", err)
	}

	now := time.Now().Unix()

	// 检查资源是否已存在
	var existingCreatedAt int64
	err = s.db.QueryRow(
		"SELECT created_at FROM resources WHERE key = ? AND type = ?",
		key.Key, typ,
	).Scan(&existingCreatedAt)

	if err == sql.ErrNoRows {
		// 新建资源：created_at = updated_at = now
		_, err = s.db.Exec(
			`INSERT INTO resources (key, type, origin_key, value, tags, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			key.Key, typ, key.OriginKey, value.Val, string(tagsJSON), now, now,
		)
	} else if err != nil {
		return fmt.Errorf("查询资源失败: %w", err)
	} else {
		// 更新资源：保持 created_at 不变，更新 updated_at = now
		_, err = s.db.Exec(
			`UPDATE resources SET origin_key = ?, value = ?, tags = ?, updated_at = ? WHERE key = ? AND type = ?`,
			key.OriginKey, value.Val, string(tagsJSON), now, key.Key, typ,
		)
	}

	if err != nil {
		return fmt.Errorf("保存资源失败: %w", err)
	}

	return nil
}

func (s *SQLiteStorage) DeleteResource(key models.ValJsonKey) error {
	typ := "ORIGIN"
	if key.Type == models.TAG {
		typ = "TAG"
	}

	_, err := s.db.Exec(
		`DELETE FROM resources WHERE key = ? AND type = ?`,
		key.Key, typ,
	)
	if err != nil {
		return fmt.Errorf("删除资源失败: %w", err)
	}

	return nil
}

func (s *SQLiteStorage) UpdateResource(key models.ValJsonKey, newValue models.ValJson) error {
	return s.SaveResource(key, newValue)
}

func (s *SQLiteStorage) SaveAuditRecord(record models.AuditRecord) error {
	_, err := s.db.Exec(
		`INSERT INTO audit (resource_key, operation, timestamp, count) VALUES (?, ?, ?, ?)`,
		record.ResourceKey, record.Operation, record.Timestamp, record.Count,
	)
	if err != nil {
		return fmt.Errorf("保存审计记录失败: %w", err)
	}
	return nil
}

func (s *SQLiteStorage) GetAuditStats() (models.AuditStats, error) {
	stats := models.AuditStats{
		ByOperation: make(map[string]int),
		ByResource:  make(map[string]int),
	}

	rows, err := s.db.Query("SELECT operation, resource_key, count FROM audit")
	if err != nil {
		return stats, fmt.Errorf("查询审计统计失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var operation, resourceKey string
		var count int
		if err := rows.Scan(&operation, &resourceKey, &count); err != nil {
			return stats, fmt.Errorf("扫描审计行失败: %w", err)
		}

		stats.TotalOperations += count
		stats.ByOperation[operation] += count
		stats.ByResource[resourceKey] += count
	}

	return stats, nil
}

func (s *SQLiteStorage) GetAllAuditRecords() ([]models.AuditRecord, error) {
	var records []models.AuditRecord

	rows, err := s.db.Query("SELECT resource_key, operation, timestamp, count FROM audit ORDER BY timestamp DESC")
	if err != nil {
		return nil, fmt.Errorf("查询审计记录失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var record models.AuditRecord
		if err := rows.Scan(&record.ResourceKey, &record.Operation, &record.Timestamp, &record.Count); err != nil {
			return nil, fmt.Errorf("扫描审计行失败: %w", err)
		}
		records = append(records, record)
	}

	return records, nil
}

func (s *SQLiteStorage) DeleteAuditRecords(resourceKey string) error {
	_, err := s.db.Exec(`DELETE FROM audit WHERE resource_key = ?`, resourceKey)
	if err != nil {
		return fmt.Errorf("删除审计记录失败: %w", err)
	}
	return nil
}

func (s *SQLiteStorage) SaveHistoryRecord(record models.HistoryRecord) error {
	_, err := s.db.Exec(
		`INSERT INTO history (id, resource_key, operation, timestamp, time_str, command) VALUES (?, ?, ?, ?, ?, ?)`,
		record.ID, record.ResourceKey, record.Operation, record.Timestamp, record.TimeStr, record.Command,
	)
	if err != nil {
		return fmt.Errorf("保存历史记录失败: %w", err)
	}
	return nil
}

func (s *SQLiteStorage) GetAllHistoryRecords() ([]models.HistoryRecord, error) {
	var records []models.HistoryRecord

	rows, err := s.db.Query(`SELECT id, resource_key, operation, timestamp, time_str, command FROM history ORDER BY timestamp DESC`)
	if err != nil {
		return nil, fmt.Errorf("查询历史记录失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var record models.HistoryRecord
		if err := rows.Scan(&record.ID, &record.ResourceKey, &record.Operation, &record.Timestamp, &record.TimeStr, &record.Command); err != nil {
			return nil, fmt.Errorf("扫描历史行失败: %w", err)
		}
		records = append(records, record)
	}

	return records, nil
}

func (s *SQLiteStorage) GetHistoryRecord(index int, order models.SortOrder) (models.HistoryRecord, error) {
	var orderBy string
	if order == models.Descending {
		orderBy = "timestamp DESC"
	} else {
		orderBy = "timestamp ASC"
	}

	var record models.HistoryRecord
	err := s.db.QueryRow(`
		SELECT id, resource_key, operation, timestamp, time_str, command
		FROM history
		ORDER BY `+orderBy+`
		LIMIT 1 OFFSET ?
	`, index).Scan(&record.ID, &record.ResourceKey, &record.Operation, &record.Timestamp, &record.TimeStr, &record.Command)

	if err != nil {
		if err == sql.ErrNoRows {
			return models.HistoryRecord{}, fmt.Errorf("index %d out of bounds", index)
		}
		return models.HistoryRecord{}, fmt.Errorf("查询历史记录失败: %w", err)
	}

	return record, nil
}

func (s *SQLiteStorage) GetHistoryStats() (models.HistoryStats, error) {
	stats := models.HistoryStats{
		ByOperation: make(map[string]int),
		ByResource:  make(map[string]int),
	}

	records, err := s.GetAllHistoryRecords()
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

func (s *SQLiteStorage) DeleteHistoryRecords(resourceKey string) error {
	_, err := s.db.Exec(`DELETE FROM history WHERE resource_key = ?`, resourceKey)
	if err != nil {
		return fmt.Errorf("删除历史记录失败: %w", err)
	}
	return nil
}

func (s *SQLiteStorage) SaveLogRecord(record models.LogRecord) error {
	tagsJSON, err := json.Marshal(record.Tags)
	if err != nil {
		return fmt.Errorf("序列化标签失败: %w", err)
	}

	_, err = s.db.Exec(
		`INSERT INTO logs (id, content, tags, created_at, date) VALUES (?, ?, ?, ?, ?)`,
		record.ID, record.Content, string(tagsJSON), record.CreatedAt, record.Date,
	)
	if err != nil {
		return fmt.Errorf("保存日志记录失败: %w", err)
	}
	return nil
}

func (s *SQLiteStorage) GetLogRecords(startDate, endDate string) ([]models.LogRecord, error) {
	var records []models.LogRecord

	query := `SELECT id, content, tags, created_at, date FROM logs WHERE 1=1`
	args := []interface{}{}

	if startDate != "" {
		query += ` AND date >= ?`
		args = append(args, startDate)
	}
	if endDate != "" {
		query += ` AND date <= ?`
		args = append(args, endDate)
	}

	query += ` ORDER BY id DESC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询日志记录失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var record models.LogRecord
		var tagsStr string
		if err := rows.Scan(&record.ID, &record.Content, &tagsStr, &record.CreatedAt, &record.Date); err != nil {
			return nil, fmt.Errorf("扫描日志行失败: %w", err)
		}

		if tagsStr != "" {
			if err := json.Unmarshal([]byte(tagsStr), &record.Tags); err != nil {
				return nil, fmt.Errorf("解析标签失败: %w", err)
			}
		}

		records = append(records, record)
	}

	return records, nil
}

func (s *SQLiteStorage) DeleteLogRecord(id int64) error {
	_, err := s.db.Exec(`DELETE FROM logs WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("删除日志记录失败: %w", err)
	}
	return nil
}

// Chat history methods for SQLiteStorage
func (s *SQLiteStorage) SaveChatMessage(sessionID string, message models.ChatMessage) error {
	_, err := s.db.Exec(
		`INSERT INTO chats (session_id, role, content, timestamp) VALUES (?, ?, ?, ?)`,
		sessionID, message.Role, message.Content, message.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("保存聊天消息失败: %w", err)
	}
	return nil
}

func (s *SQLiteStorage) GetChatMessages(sessionID string) ([]models.ChatMessage, error) {
	var messages []models.ChatMessage

	rows, err := s.db.Query(`
		SELECT role, content, timestamp
		FROM chats
		WHERE session_id = ?
		ORDER BY timestamp ASC
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("查询聊天消息失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var msg models.ChatMessage
		if err := rows.Scan(&msg.Role, &msg.Content, &msg.Timestamp); err != nil {
			return nil, fmt.Errorf("扫描聊天消息行失败: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

func (s *SQLiteStorage) ClearChatMessages(sessionID string) error {
	_, err := s.db.Exec(`DELETE FROM chats WHERE session_id = ?`, sessionID)
	if err != nil {
		return fmt.Errorf("清理聊天消息失败: %w", err)
	}
	return nil
}

func (s *SQLiteStorage) GetSessionMeta(sessionID string) (*models.SessionMeta, error) {
	var meta models.SessionMeta

	err := s.db.QueryRow(`
		SELECT session_id, last_active
		FROM sessions
		WHERE session_id = ?
	`, sessionID).Scan(&meta.SessionID, &meta.LastActive)

	if err != nil {
		return nil, fmt.Errorf("查询会话元数据失败: %w", err)
	}

	return &meta, nil
}

func (s *SQLiteStorage) UpdateSessionMeta(sessionID string, lastActive int64) error {
	_, err := s.db.Exec(`
		INSERT INTO sessions (session_id, last_active) VALUES (?, ?)
		ON CONFLICT(session_id) DO UPDATE SET last_active = ?
	`, sessionID, lastActive, lastActive)
	if err != nil {
		return fmt.Errorf("更新会话元数据失败: %w", err)
	}
	return nil
}
