package command

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"
	"ttl-cli/ai"
	"ttl-cli/db"
	"ttl-cli/i18n"
	"ttl-cli/models"

	"github.com/spf13/cobra"
)

type storeAdapter struct{}

func (s *storeAdapter) GetAllResources() (map[models.ValJsonKey]models.ValJson, error) {
	return db.GetAllResources()
}

func (s *storeAdapter) SaveResource(key models.ValJsonKey, value models.ValJson) error {
	return db.SaveResource(key, value)
}

func (s *storeAdapter) DeleteResource(key models.ValJsonKey) error {
	return db.DeleteResource(key)
}

func (s *storeAdapter) UpdateResource(key models.ValJsonKey, newValue models.ValJson) error {
	return db.UpdateResource(key, newValue)
}

func (s *storeAdapter) GetAuditStats() (models.AuditStats, error) {
	return db.GetAuditStats()
}

func (s *storeAdapter) GetAllHistoryRecords() ([]models.HistoryRecord, error) {
	return db.GetAllHistoryRecords()
}

func (s *storeAdapter) SaveLogRecord(record models.LogRecord) error {
	return db.SaveLogRecord(record)
}

func (s *storeAdapter) GetLogRecords(startDate, endDate string) ([]models.LogRecord, error) {
	return db.GetLogRecords(startDate, endDate)
}

func (s *storeAdapter) DeleteLogRecord(id int64) error {
	return db.DeleteLogRecord(id)
}

type spinner struct {
	mu      sync.Mutex
	stopped bool
	done    chan struct{}
}

func newSpinner(msg string) *spinner {
	s := &spinner{done: make(chan struct{})}
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	go func() {
		i := 0
		for {
			select {
			case <-s.done:
				return
			default:
				s.mu.Lock()
				if s.stopped {
					s.mu.Unlock()
					return
				}
				s.mu.Unlock()
				fmt.Fprintf(os.Stderr, "\r%s %s", frames[i%len(frames)], msg)
				i++
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
	return s
}

func (s *spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.stopped {
		s.stopped = true
		close(s.done)
		fmt.Fprintf(os.Stderr, "\r\033[K")
	}
}

var AICmd = &cobra.Command{
	Use:   "ai <input>",
	Short: i18n.T("command.ai.short"),
	Long:  i18n.T("command.ai.long"),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		aiConf, ok := cmd.Context().Value("ai_config").(models.AIConfig)
		if !ok || aiConf.APIKey == "" {
			return errors.New(i18n.T("command.ai.not_configured"))
		}

		client := ai.NewClient(aiConf.BaseURL, aiConf.APIKey, aiConf.Model, aiConf.Timeout)

		chatWithSpinner := func(messages []ai.Message) (string, error) {
			sp := newSpinner(i18n.T("command.ai.thinking"))
			defer sp.Stop()
			return client.Chat(messages)
		}

		agent := ai.NewAgent(chatWithSpinner, &storeAdapter{})

		if aiConf.ContextEnabled {
			contextLoader := ai.NewContextLoader(
				aiConf.ContextEnabled,
				aiConf.ContextIdleTTL,
				aiConf.ContextMaxRounds,
				aiConf.ContextMaxTokens,
			)
			contextLoader.SaveMessage = db.SaveChatMessage
			contextLoader.GetMessages = db.GetChatMessages
			contextLoader.ClearMessages = db.ClearChatMessages
			contextLoader.GetMeta = db.GetSessionMeta
			contextLoader.UpdateMeta = db.UpdateSessionMeta

			agent = agent.WithContext(contextLoader)
		}

		agent.OpenFn = func(url string) error {
			switch runtime.GOOS {
			case "darwin":
				return exec.Command("open", url).Run()
			case "windows":
				return exec.Command("explorer", url).Run()
			default:
				return exec.Command("xdg-open", url).Run()
			}
		}

		result, err := agent.Run(args[0])
		if err != nil {
			return err
		}

		Println(result)
		return nil
	},
}
