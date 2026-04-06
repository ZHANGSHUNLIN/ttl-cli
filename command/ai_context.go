package command

import (
	"fmt"
	"time"
	"ttl-cli/conf"
	"ttl-cli/db"

	"github.com/spf13/cobra"
)

const defaultSessionID = "default"

var AIContextCmd = &cobra.Command{
	Use:   "ai-context",
	Short: "Manage AI chat context sessions",
	Long:  "Manage AI chat context sessions (status, clear, new)",
}

var aiContextStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current AI context session status",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		aiConf, err := conf.LoadAIConfig("")
		if err != nil {
			return fmt.Errorf("加载配置失败: %w", err)
		}

		Println("当前会话:", defaultSessionID)
		if aiConf.ContextEnabled {
			Println("上下文状态: 启用")
		} else {
			Println("上下文状态: 禁用")
		}
		Println("配置: 空闲超时=", aiConf.ContextIdleTTL, "分钟, 最大轮次=", aiConf.ContextMaxRounds, ", 最大token=", aiConf.ContextMaxTokens)

		messages, err := db.GetChatMessages(defaultSessionID)
		if err != nil {
			Println("消息统计: 获取失败")
			return nil
		}

		if len(messages) == 0 {
			Println("消息统计: 无消息")
			return nil
		}

		userCount := 0
		assistantCount := 0
		for _, msg := range messages {
			if msg.Role == "user" {
				userCount++
			} else if msg.Role == "assistant" {
				assistantCount++
			}
		}

		Println("消息统计: 共", len(messages), "条(user:", userCount, ", assistant:", assistantCount, ")")

		// 获取会话元数据
		meta, err := db.GetSessionMeta(defaultSessionID)
		if err == nil {
			Println("最后活跃:", time.Unix(meta.LastActive, 0).Format("2006-01-02 15:04:05"))

			idleMinutes := int(time.Since(time.Unix(meta.LastActive, 0)).Minutes())
			if idleMinutes > aiConf.ContextIdleTTL {
				Println("会话状态: 已过期（距离上次对话", idleMinutes, "分钟）")
			} else {
				Println("会话状态: 活跃（距离上次对话", idleMinutes, "分钟）")
			}
		}

		return nil
	},
}

var aiContextClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear current AI context session",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		messages, err := db.GetChatMessages(defaultSessionID)
		if err != nil {
			return fmt.Errorf("获取消息失败: %w", err)
		}

		if err := db.ClearChatMessages(defaultSessionID); err != nil {
			return fmt.Errorf("清理上下文失败: %w", err)
		}

		Println("已清理当前会话的上下文（共", len(messages), "条消息）")
		return nil
	},
}

var aiContextNewCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a new AI context session",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 生成随机 session ID
		sessionID := fmt.Sprintf("sess_%x", time.Now().UnixNano())

		Println("已创建新会话:", sessionID)
		return nil
	},
}

func init() {
	AIContextCmd.AddCommand(aiContextStatusCmd)
	AIContextCmd.AddCommand(aiContextClearCmd)
	AIContextCmd.AddCommand(aiContextNewCmd)
}
