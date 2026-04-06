package ai

import (
	"time"
	"ttl-cli/models"
)

const defaultSessionID = "default"

// ContextLoader 负责加载和保存聊天上下文
type ContextLoader struct {
	Enabled       bool
	IdleTTL       int
	MaxRounds     int
	MaxTokens     int
	SessionID     string
	SaveMessage   func(sessionID string, message models.ChatMessage) error
	GetMessages   func(sessionID string) ([]models.ChatMessage, error)
	ClearMessages func(sessionID string) error
	GetMeta       func(sessionID string) (*models.SessionMeta, error)
	UpdateMeta    func(sessionID string, lastActive int64) error
}

// NewContextLoader 创建上下文加载器
func NewContextLoader(enabled bool, idleTTL, maxRounds, maxTokens int) *ContextLoader {
	return &ContextLoader{
		Enabled:   enabled,
		IdleTTL:   idleTTL,
		MaxRounds: maxRounds,
		MaxTokens: maxTokens,
		SessionID: defaultSessionID,
	}
}

// LoadMessages 加载历史消息
func (c *ContextLoader) LoadMessages() ([]models.ChatMessage, error) {
	if !c.Enabled {
		return nil, nil
	}

	// 检查会话是否过期
	meta, err := c.GetMeta(c.SessionID)
	if err == nil {
		idleMinutes := int(time.Since(time.Unix(meta.LastActive, 0)).Minutes())
		if idleMinutes > c.IdleTTL {
			// 会话过期，清理并返回空
			_ = c.ClearMessages(c.SessionID)
			return nil, nil
		}
	}

	// 加载所有消息
	messages, err := c.GetMessages(c.SessionID)
	if err != nil {
		return nil, err
	}

	// 应用轮次限制
	if c.MaxRounds > 0 {
		messages = c.limitByRounds(messages)
	}

	// 应用 token 限制
	if c.MaxTokens > 0 {
		messages = c.limitByTokens(messages)
	}

	return messages, nil
}

// SaveUserMessage 保存用户消息
func (c *ContextLoader) SaveUserMessage(content string) error {
	if !c.Enabled {
		return nil
	}

	message := models.ChatMessage{
		Role:      "user",
		Content:   content,
		Timestamp: time.Now().Unix(),
	}

	if err := c.SaveMessage(c.SessionID, message); err != nil {
		return err
	}

	// 更新会话最后活跃时间
	return c.UpdateMeta(c.SessionID, message.Timestamp)
}

// SaveAssistantMessage 保存助手回复
func (c *ContextLoader) SaveAssistantMessage(content string) error {
	if !c.Enabled {
		return nil
	}

	message := models.ChatMessage{
		Role:      "assistant",
		Content:   content,
		Timestamp: time.Now().Unix(),
	}

	return c.SaveMessage(c.SessionID, message)
}

// limitByRounds 按轮次限制消息
func (c *ContextLoader) limitByRounds(messages []models.ChatMessage) []models.ChatMessage {
	if c.MaxRounds <= 0 || len(messages) == 0 {
		return messages
	}

	// 统计 user 消息数量（assistant 消息不单独计数，与 user 配对）
	var userIndices []int
	for i, msg := range messages {
		if msg.Role == "user" {
			userIndices = append(userIndices, i)
		}
	}

	// 如果 user 消息数量不超过限制，直接返回
	if len(userIndices) <= c.MaxRounds {
		return messages
	}

	// 只保留最后 MaxRounds 个 user 消息及其后的消息
	startIdx := userIndices[len(userIndices)-c.MaxRounds]

	return messages[startIdx:]
}

// limitByTokens 按 token 数限制消息
func (c *ContextLoader) limitByTokens(messages []models.ChatMessage) []models.ChatMessage {
	if c.MaxTokens <= 0 {
		return messages
	}

	totalTokens := 0
	startIdx := 0
	for i := len(messages) - 1; i >= 0; i-- {
		msgTokens := estimateTokens(messages[i].Content)
		if totalTokens+msgTokens > c.MaxTokens {
			startIdx = i + 1
			break
		}
		totalTokens += msgTokens
	}

	if startIdx >= len(messages) {
		return []models.ChatMessage{}
	}
	return messages[startIdx:]
}

// estimateTokens 估算 token 数
func estimateTokens(text string) int {
	chineseChars := 0
	for _, r := range text {
		if r >= 0x4e00 && r <= 0x9fff {
			chineseChars++
		}
	}
	englishChars := len(text) - chineseChars
	// 中文约 0.5 字符/token，英文约 0.25 字符/token
	return chineseChars/2 + englishChars/4
}

// CountChineseChars 统计中文字符数
func CountChineseChars(s string) int {
	count := 0
	for _, r := range s {
		if r >= 0x4e00 && r <= 0x9fff {
			count++
		}
	}
	return count
}
