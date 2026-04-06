package models

const (
	ORIGIN = iota // Original resource
	TAG           // Tag resource
)

const Version = "1.0.0"

type ValJsonKey struct {
	Key       string `json:"key"`
	Type      int    `json:"type"`
	OriginKey string `json:"originKey"`
}

type ValJson struct {
	Val       string   `json:"val"`
	Tag       []string `json:"tag"`
	CreatedAt int64    `json:"createdAt"`
	UpdatedAt int64    `json:"updatedAt"`
}

type AuditRecord struct {
	ResourceKey string `json:"resourceKey"`
	Operation   string `json:"operation"`
	Timestamp   int64  `json:"timestamp"`
	Count       int    `json:"count"`
}

type AuditStats struct {
	TotalOperations int            `json:"totalOperations"`
	ByOperation     map[string]int `json:"byOperation"`
	ByResource      map[string]int `json:"byResource"`
}

type HistoryRecord struct {
	ID          int64  `json:"id"`
	ResourceKey string `json:"resourceKey"`
	Operation   string `json:"operation"`
	Timestamp   int64  `json:"timestamp"`
	TimeStr     string `json:"timeStr"`
	Command     string `json:"command"`
	Args        string `json:"args"`
}

type HistoryStats struct {
	TotalRecords int             `json:"totalRecords"`
	Records      []HistoryRecord `json:"records"`
	ByOperation  map[string]int  `json:"byOperation"`
	ByResource   map[string]int  `json:"byResource"`
}

type LogRecord struct {
	ID        int64    `json:"id"`
	Content   string   `json:"content"`
	Tags      []string `json:"tags"`
	CreatedAt string   `json:"createdAt"`
	Date      string   `json:"date"`
}

// ChatMessage 表示单条聊天消息
type ChatMessage struct {
	Role      string `json:"role"`      // "system" | "user" | "assistant"
	Content   string `json:"content"`   // 消息内容
	Timestamp int64  `json:"timestamp"` // Unix 时间戳
}

// SessionMeta 表示会话元数据
type SessionMeta struct {
	SessionID  string `json:"session_id"`  // 会话 ID
	LastActive int64  `json:"last_active"` // 最后活跃时间（Unix 时间戳）
}

// SortOrder 定义排序方向类型
type SortOrder string

const (
	Ascending  SortOrder = "asc"
	Descending SortOrder = "desc"
)

type TtlIni struct {
	StorageType string       `ini:"storage_type"`
	DbPath      string       `ini:"db_path"`
	AI          AIConfig     `ini:"ai"`
	BoltDB      BoltDBConfig `ini:"bbolt"`
}

type AIConfig struct {
	APIKey  string `ini:"api_key"`
	BaseURL string `ini:"base_url"`
	Model   string `ini:"model"`
	Timeout int    `ini:"timeout"`

	// 多轮对话上下文配置
	ContextEnabled   bool `ini:"context_enabled"`
	ContextIdleTTL   int  `ini:"context_idle_ttl"`
	ContextMaxRounds int  `ini:"context_max_rounds"`
	ContextMaxTokens int  `ini:"context_max_tokens"`
}

type BoltDBConfig struct {
	Timeout int `ini:"timeout"` // 超时时间（秒），默认 5 秒
}
