package ai

import (
	"time"
	"ttl-cli/models"
)

const defaultSessionID = "default"

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

func NewContextLoader(enabled bool, idleTTL, maxRounds, maxTokens int) *ContextLoader {
	return &ContextLoader{
		Enabled:   enabled,
		IdleTTL:   idleTTL,
		MaxRounds: maxRounds,
		MaxTokens: maxTokens,
		SessionID: defaultSessionID,
	}
}

func (c *ContextLoader) LoadMessages() ([]models.ChatMessage, error) {
	if !c.Enabled {
		return nil, nil
	}

	meta, err := c.GetMeta(c.SessionID)
	if err == nil {
		idleMinutes := int(time.Since(time.Unix(meta.LastActive, 0)).Minutes())
		if idleMinutes > c.IdleTTL {
			_ = c.ClearMessages(c.SessionID)
			return nil, nil
		}
	}

	messages, err := c.GetMessages(c.SessionID)
	if err != nil {
		return nil, err
	}

	if c.MaxRounds > 0 {
		messages = c.limitByRounds(messages)
	}

	if c.MaxTokens > 0 {
		messages = c.limitByTokens(messages)
	}

	return messages, nil
}

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

	return c.UpdateMeta(c.SessionID, message.Timestamp)
}

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

func (c *ContextLoader) limitByRounds(messages []models.ChatMessage) []models.ChatMessage {
	if c.MaxRounds <= 0 || len(messages) == 0 {
		return messages
	}

	var userIndices []int
	for i, msg := range messages {
		if msg.Role == "user" {
			userIndices = append(userIndices, i)
		}
	}

	if len(userIndices) <= c.MaxRounds {
		return messages
	}

	startIdx := userIndices[len(userIndices)-c.MaxRounds]

	return messages[startIdx:]
}

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

func estimateTokens(text string) int {
	chineseChars := 0
	for _, r := range text {
		if r >= 0x4e00 && r <= 0x9fff {
			chineseChars++
		}
	}
	englishChars := len(text) - chineseChars
	return chineseChars/2 + englishChars/4
}

func CountChineseChars(s string) int {
	count := 0
	for _, r := range s {
		if r >= 0x4e00 && r <= 0x9fff {
			count++
		}
	}
	return count
}
