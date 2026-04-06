package command

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"ttl-cli/conf"
	"ttl-cli/db"
	"ttl-cli/i18n"
	"ttl-cli/models"
	"ttl-cli/util"

	"github.com/spf13/cobra"
)

var OutputWriter io.Writer = os.Stdout

func SetOutputWriter(w io.Writer) {
	OutputWriter = w
}

func ResetOutputWriter() {
	OutputWriter = os.Stdout
}

func Print(a ...any) (n int, err error) {
	return fmt.Fprint(OutputWriter, a...)
}

func Printf(format string, a ...any) (n int, err error) {
	return fmt.Fprintf(OutputWriter, format, a...)
}

func Println(a ...any) (n int, err error) {
	return fmt.Fprintln(OutputWriter, a...)
}

var AddCmd = &cobra.Command{
	Use:   "add [key] [value]",
	Short: i18n.T("command.add.short"),
	Long:  i18n.T("command.add.long"),
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := models.ValJsonKey{
			Key:  args[0],
			Type: models.ORIGIN,
		}

		resources, err := db.GetAllResources()
		if err != nil {
			return fmt.Errorf(i18n.T("command.add.error_fetch"), err)
		}

		if _, exists := resources[key]; exists {
			return errors.New(i18n.T("command.add.duplicate", args[0]))
		}

		value := util.UnescapeString(args[1])

		newResource := models.ValJson{
			Val: value,
			Tag: util.RemoveDuplicates(addTags),
		}

		if err := db.SaveResource(key, newResource); err != nil {
			return fmt.Errorf(i18n.T("command.add.error_save"), err)
		}
		debug := cmd.Context().Value("debug").(bool)

		if err := db.RecordAudit(args[0], "add"); err != nil && debug {
			Printf(i18n.T("command.add.audit_error"), err)
		}

		Println(i18n.T("command.add.success"))
		return nil
	},
}

var addTags []string

func init() {
	AddCmd.Flags().StringSliceVarP(&addTags, "tag", "t", nil, i18n.T("command.add.flag_tag"))
}

var GetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: i18n.T("command.get.short"),
	Long:  i18n.T("command.get.long"),
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		resources, err := db.GetAllResources()
		if err != nil {
			return fmt.Errorf(i18n.T("command.get.error_fetch"), err)
		}

		if len(args) == 0 {
			Println(i18n.T("command.get.no_filter_notice"))
			Println()

			type resourceWithKey struct {
				key models.ValJsonKey
				val models.ValJson
			}
			sortedResources := make([]resourceWithKey, 0, len(resources))
			for k, v := range resources {
				if k.Type != models.TAG {
					sortedResources = append(sortedResources, resourceWithKey{k, v})
				}
			}
			sort.Slice(sortedResources, func(i, j int) bool {
				return sortedResources[i].val.CreatedAt > sortedResources[j].val.CreatedAt
			})

			for _, item := range sortedResources {
				Println("  ", item.key.Key)
			}
			Println()
			return nil
		}
		debug := cmd.Context().Value("debug").(bool)

		return PossiblyRun(debug, "get", resources, args[0], func(key models.ValJsonKey, json models.ValJson) error {
			Println()
			if key.Type == models.ORIGIN {
				Println(i18n.T("command.get.resource_label"), key.Key)
			} else {
				Println(i18n.T("command.get.resource_label"), key.OriginKey)
			}
			Println()
			Println(json.Val)
			Println()
			Println(i18n.T("command.get.tags_label"), json.Tag)
			resourceKey := key.Key
			if key.Type == models.TAG {
				resourceKey = key.OriginKey
			}
			if err := db.RecordAudit(resourceKey, "get"); err != nil && debug {
				Printf(i18n.T("command.get.audit_error"), err)
			}
			if key.Type == models.TAG {
				resourceKey = key.OriginKey
			}
			return nil
		})
	},
}

var OpenCmd = &cobra.Command{
	Use:   "open [key]",
	Short: i18n.T("command.open.short"),
	Long:  i18n.T("command.open.long"),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		resources, err := db.GetAllResources()
		if err != nil {
			return fmt.Errorf(i18n.T("command.open.error_fetch"), err)
		}
		debug := cmd.Context().Value("debug").(bool)

		return PossiblyRun(debug, "open", resources, args[0], func(key models.ValJsonKey, json models.ValJson) error {
			switch os := runtime.GOOS; os {
			case "darwin":
				if debug {
					Println(i18n.T("command.open.macos_detected"))
				}
				cmd := exec.Command("open", json.Val)
				return cmd.Run()
			case "linux":
				return errors.New(i18n.T("command.open.linux_not_supported"))
			case "windows":
				if debug {
					Println(i18n.T("command.open.windows_detected"), json.Val)
				}
				cmd := exec.Command("explorer", json.Val)
				return cmd.Run()
			default:
				return fmt.Errorf(i18n.T("command.open.system_not_supported"), os)
			}
		})
	},
}

var DelCmd = &cobra.Command{
	Use:     "del [key]",
	Aliases: []string{"rm"},
	Short:   i18n.T("command.delete.short"),
	Long:    i18n.T("command.delete.long"),
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := models.ValJsonKey{
			Key:  args[0],
			Type: models.ORIGIN,
		}

		resources, err := db.GetAllResources()
		if err != nil {
			return fmt.Errorf(i18n.T("command.delete.error_fetch"), err)
		}

		_, exists := resources[key]
		if !exists {
			Printf(i18n.T("command.delete.not_found"), args[0])
			return nil
		}
		debug := cmd.Context().Value("debug").(bool)

		db.CleanupResourceHistory(args[0], debug)

		if err := db.DeleteResource(key); err != nil {
			return fmt.Errorf(i18n.T("command.delete.error_delete"), err)
		}

		Println(i18n.T("command.delete.success"))
		return nil
	},
}

var TagCmd = &cobra.Command{
	Use:   "tag [key] [tags...]",
	Short: i18n.T("command.tag.short"),
	Long:  i18n.T("command.tag.long"),
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := models.ValJsonKey{
			Key:  args[0],
			Type: models.ORIGIN,
		}

		resources, err := db.GetAllResources()
		if err != nil {
			return fmt.Errorf(i18n.T("command.tag.error_fetch"), err)
		}

		resource, exists := resources[key]
		if !exists {
			return errors.New(i18n.T("command.tag.not_found"))
		}

		newTags := append(resource.Tag, args[1:]...)
		resource.Tag = util.RemoveDuplicates(newTags)

		if err := db.SaveResource(key, resource); err != nil {
			return fmt.Errorf(i18n.T("command.tag.error_save"), err)
		}

		Println(i18n.T("command.tag.success"))
		return nil
	},
}

var DtagCmd = &cobra.Command{
	Use:   "dtag [key] [tag]",
	Short: i18n.T("command.dtag.short"),
	Long:  i18n.T("command.dtag.long"),
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := models.ValJsonKey{
			Key:  args[0],
			Type: models.ORIGIN,
		}

		resources, err := db.GetAllResources()
		if err != nil {
			return fmt.Errorf(i18n.T("command.dtag.error_fetch"), err)
		}

		resource, exists := resources[key]
		if !exists {
			return errors.New(i18n.T("command.dtag.not_found"))
		}

		newTags := make([]string, 0, len(resource.Tag))
		for _, tag := range resource.Tag {
			if tag != args[1] {
				newTags = append(newTags, tag)
			}
		}

		resource.Tag = newTags
		if err := db.SaveResource(key, resource); err != nil {
			return fmt.Errorf(i18n.T("command.dtag.error_save"), err)
		}

		Println(i18n.T("command.dtag.success"))
		return nil
	},
}

var RenameCmd = &cobra.Command{
	Use:     "rename [oldKey] [newKey]",
	Aliases: []string{"mv"},
	Short:   i18n.T("command.rename.short"),
	Long:    i18n.T("command.rename.long"),
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		oldKey := models.ValJsonKey{
			Key:  args[0],
			Type: models.ORIGIN,
		}

		resources, err := db.GetAllResources()
		if err != nil {
			return fmt.Errorf(i18n.T("command.rename.error_fetch"), err)
		}

		resource, exists := resources[oldKey]
		if !exists {
			return errors.New(i18n.T("command.rename.not_found"))
		}

		if err := db.DeleteResource(oldKey); err != nil {
			return fmt.Errorf(i18n.T("command.rename.error_delete_old"), err)
		}

		newKey := models.ValJsonKey{
			Key:  args[1],
			Type: models.ORIGIN,
		}

		if err := db.SaveResource(newKey, resource); err != nil {
			return fmt.Errorf(i18n.T("command.rename.error_save_new"), err)
		}

		Println(i18n.T("command.rename.success"))
		return nil
	},
}

var ConfigCmd = &cobra.Command{
	Use:   "config",
	Short: i18n.T("command.config.short"),
	Long:  i18n.T("command.config.long"),
	RunE: func(cmd *cobra.Command, args []string) error {
		ttlConf, err := conf.GetTtlConf()
		if err != nil {
			return fmt.Errorf(i18n.T("command.config.error_get_path"), err)
		}

		dbPath, err := db.GetDBPath("", ttlConf.StorageType)
		if err != nil {
			return fmt.Errorf(i18n.T("command.config.error_get_path"), err)
		}
		Println(i18n.T("command.config.data_path_label"), dbPath)
		Println(i18n.T("command.config.storage_type_label"), ttlConf.StorageType)

		confPath, err := conf.GetDefaultConfPath()
		if err == nil {
			Println(i18n.T("command.config.config_path_label"), confPath)
		}

		aiConf, err := conf.LoadAIConfig("")
		if err == nil && aiConf.APIKey != "" {
			masked := aiConf.APIKey[:4] + "****"
			Println()
			Println(i18n.T("command.config.ai_config_label"))
			Println("  api_key  =", masked)
			Println("  base_url =", aiConf.BaseURL)
			Println("  model    =", aiConf.Model)
			Println("  timeout  =", aiConf.Timeout)
		}
		return nil
	},
}

var configAICmd = &cobra.Command{
	Use:   "ai",
	Short: i18n.T("command.config_ai.short"),
	Long:  i18n.T("command.config_ai.long"),
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)

		existing, _ := conf.LoadAIConfig("")

		Println(i18n.T("command.config_ai.title"))
		Println()

		if existing.APIKey != "" {
			masked := existing.APIKey[:4] + "****"
			Printf(i18n.T("command.config_ai.api_key_current"), masked)
		} else {
			Print(i18n.T("command.config_ai.api_key_prompt"))
		}
		apiKey := readLine(reader)
		if apiKey == "" {
			apiKey = existing.APIKey
		}
		if apiKey == "" {
			return errors.New(i18n.T("command.config_ai.api_key_empty"))
		}

		defaultURL := existing.BaseURL
		if defaultURL == "" {
			defaultURL = "https://api.openai.com"
		}
		Printf(i18n.T("command.config_ai.api_url_prompt"), defaultURL)
		baseURL := readLine(reader)
		if baseURL == "" {
			baseURL = defaultURL
		}

		defaultModel := existing.Model
		if defaultModel == "" {
			defaultModel = "gpt-4o-mini"
		}
		Printf(i18n.T("command.config_ai.model_prompt"), defaultModel)
		model := readLine(reader)
		if model == "" {
			model = defaultModel
		}

		defaultTimeout := existing.Timeout
		if defaultTimeout == 0 {
			defaultTimeout = 30
		}
		Printf(i18n.T("command.config_ai.timeout_prompt"), defaultTimeout)
		timeoutStr := readLine(reader)
		timeout := defaultTimeout
		if timeoutStr != "" {
			if n, err := strconv.Atoi(timeoutStr); err == nil && n > 0 {
				timeout = n
			}
		}

		aiConf := models.AIConfig{
			APIKey:  apiKey,
			BaseURL: baseURL,
			Model:   model,
			Timeout: timeout,
		}

		if err := conf.SaveAIConfig("", aiConf); err != nil {
			return fmt.Errorf(i18n.T("command.config_ai.error_save"), err)
		}

		masked := apiKey[:4] + "****"
		Println()
		Println(i18n.T("command.config_ai.success"))
		Println("  api_key  =", masked)
		Println("  base_url =", baseURL)
		Println("  model    =", model)
		Println("  timeout  =", timeout)
		return nil
	},
}

var configAIContextCmd = &cobra.Command{
	Use:   "ai-context",
	Short: "Configure AI multi-turn chat context settings",
	Long:  "Configure AI multi-turn chat context settings (idle timeout, max rounds, max tokens)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)

		existing, _ := conf.LoadAIConfig("")

		Println("配置多轮对话上下文")
		Println()

		defaultEnabled := "n"
		if existing.ContextEnabled {
			defaultEnabled = "y"
		}
		Printf("启用多轮上下文？ (y/n, 默认: %s): ", defaultEnabled)
		enabledStr := readLine(reader)
		if enabledStr == "" {
			enabledStr = defaultEnabled
		}
		enabled := enabledStr == "y" || enabledStr == "Y"

		var idleTTL, maxRounds, maxTokens int

		if enabled {
			defaultIdleTTL := existing.ContextIdleTTL
			if defaultIdleTTL == 0 {
				defaultIdleTTL = 30
			}
			Printf("会话空闲超时(分钟) (默认: %d): ", defaultIdleTTL)
			idleTTLStr := readLine(reader)
			if idleTTLStr == "" {
				idleTTL = defaultIdleTTL
			} else if n, err := strconv.Atoi(idleTTLStr); err == nil && n > 0 {
				idleTTL = n
			} else {
				idleTTL = defaultIdleTTL
			}

			defaultMaxRounds := existing.ContextMaxRounds
			if defaultMaxRounds == 0 {
				defaultMaxRounds = 10
			}
			Printf("最大对话轮次 (默认: %d, 0=不限制): ", defaultMaxRounds)
			maxRoundsStr := readLine(reader)
			if maxRoundsStr == "" {
				maxRounds = defaultMaxRounds
			} else if n, err := strconv.Atoi(maxRoundsStr); err == nil && n >= 0 {
				maxRounds = n
			} else {
				maxRounds = defaultMaxRounds
			}

			defaultMaxTokens := existing.ContextMaxTokens
			if defaultMaxTokens == 0 {
				defaultMaxTokens = 4000
			}
			Printf("最大 Token 数 (默认: %d, 0=不限制): ", defaultMaxTokens)
			maxTokensStr := readLine(reader)
			if maxTokensStr == "" {
				maxTokens = defaultMaxTokens
			} else if n, err := strconv.Atoi(maxTokensStr); err == nil && n >= 0 {
				maxTokens = n
			} else {
				maxTokens = defaultMaxTokens
			}
		}

		aiConf := existing
		aiConf.ContextEnabled = enabled
		aiConf.ContextIdleTTL = idleTTL
		aiConf.ContextMaxRounds = maxRounds
		aiConf.ContextMaxTokens = maxTokens

		if err := conf.SaveAIConfig("", aiConf); err != nil {
			return fmt.Errorf("保存配置失败: %w", err)
		}

		Println()
		Println("配置已保存：")
		if enabled {
			Printf("  多轮上下文: 已启用 (空闲超时=%d分钟, 最大%d轮, 最大%d tokens)\n", idleTTL, maxRounds, maxTokens)
		} else {
			Println("  多轮上下文: 已禁用")
		}
		return nil
	},
}

func init() {
	ConfigCmd.AddCommand(configAICmd)
	ConfigCmd.AddCommand(configAIContextCmd)
}

func readLine(reader *bufio.Reader) string {
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: i18n.T("command.version.short"),
	Long:  i18n.T("command.version.long"),
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		Println(models.Version)
		return nil
	},
}

var UpdateCmd = &cobra.Command{
	Use:     "update <key> <new_value>",
	Aliases: []string{"put", "mod"},
	Short:   i18n.T("command.update.short"),
	Long:    i18n.T("command.update.long"),
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := models.ValJsonKey{Key: args[0], Type: models.ORIGIN}
		debug := cmd.Context().Value("debug").(bool)

		resources, err := db.GetAllResources()
		if err != nil {
			return fmt.Errorf(i18n.T("command.update.error_fetch"), err)
		}
		existing, exists := resources[key]
		if !exists {
			return fmt.Errorf(i18n.T("command.update.not_found"), args[0])
		}

		if err := db.RecordAudit(args[0], "update"); err != nil && debug {
			Printf(i18n.T("command.update.audit_error"), err)
		}
		value := util.UnescapeString(args[1])
		return db.UpdateResource(key, models.ValJson{Val: value, Tag: existing.Tag})
	},
}

func PossiblyRun(debug bool, opt string, resources map[models.ValJsonKey]models.ValJson, searchKey string,
	action func(models.ValJsonKey, models.ValJson) error) error {

	if debug {
		Println("当前是", opt, "操作")
	}

	matches := make(map[models.ValJsonKey]models.ValJson)
	for key, val := range resources {
		if util.ContainsIgnoreCase(key.Key, searchKey) {
			matches[key] = val
		}

		for _, s := range val.Tag {
			if util.ContainsIgnoreCase(s, searchKey) {
				matches[key] = val
			}
		}
	}

	if len(matches) == 0 {
		return errors.New(i18n.T("command.get.not_found", searchKey))
	}

	if len(matches) == 1 {
		for key := range matches {
			return action(key, matches[key])
		}
	}

	Println(i18n.T("command.get.multiple_matches"))

	keys := make([]models.ValJsonKey, 0, len(matches))
	for key := range matches {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Key < keys[j].Key
	})

	order := make([]models.ValJsonKey, 0, len(matches))
	idx := 1
	for _, key := range keys {
		Printf("%d. ", idx)
		if key.Type == models.ORIGIN {
			Println(key.Key)
		} else {
			Printf("%s "+i18n.T("command.get.tag_hint")+"\n", key.OriginKey, key.Key)
		}
		order = append(order, key)
		idx++
	}

	var choice int
	if _, err := fmt.Scan(&choice); err != nil {
		return fmt.Errorf(i18n.T("command.get.invalid_input"), err)
	}

	if choice < 1 || choice > len(order) {
		return errors.New(i18n.T("command.get.invalid_choice"))
	}

	selectedKey := order[choice-1]
	return action(selectedKey, matches[selectedKey])
}

var AuditCmd = &cobra.Command{
	Use:   "audit",
	Short: i18n.T("command.audit.short"),
	Long:  i18n.T("command.audit.long"),
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		stats, err := db.GetAuditStats()
		if err != nil {
			return fmt.Errorf(i18n.T("command.audit.error_fetch"), err)
		}

		Println(i18n.T("command.audit.title"))
		Printf(i18n.T("command.audit.total_ops"), stats.TotalOperations)
		Println()

		Println(i18n.T("command.audit.by_operation"))
		for op, count := range stats.ByOperation {
			Printf("  %s: %d\n", op, count)
		}
		Println()

		Println(i18n.T("command.audit.by_resource"))
		type resourceStat struct {
			key   string
			count int
		}
		var sortedStats []resourceStat
		for key, count := range stats.ByResource {
			sortedStats = append(sortedStats, resourceStat{key, count})
		}

		for i := 0; i < len(sortedStats)-1; i++ {
			for j := i + 1; j < len(sortedStats); j++ {
				if sortedStats[i].count < sortedStats[j].count {
					sortedStats[i], sortedStats[j] = sortedStats[j], sortedStats[i]
				}
			}
		}

		for i, stat := range sortedStats {
			if i >= 10 {
				break
			}
			Printf("  %s: %d\n", stat.key, stat.count)
		}

		return nil
	},
}

var HistoryCmd = &cobra.Command{
	Use:   "history [limit]",
	Short: i18n.T("command.history.short"),
	Long:  i18n.T("command.history.long"),
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		limit := 20
		if len(args) > 0 {
			if _, err := fmt.Sscanf(args[0], "%d", &limit); err != nil {
				return fmt.Errorf(i18n.T("command.history.invalid_limit"), args[0])
			}
		}

		records, err := db.GetAllHistoryRecords()
		if err != nil {
			return fmt.Errorf(i18n.T("command.history.error_fetch"), err)
		}

		if len(records) == 0 {
			Println(i18n.T("command.history.no_history"))
			return nil
		}

		displayRecords := records
		if limit > 0 && len(records) > limit {
			displayRecords = records[:limit]
		}

		Println(i18n.T("command.history.title"))
		Printf(i18n.T("command.history.showing"), len(displayRecords), len(records))
		Println()

		for i, record := range displayRecords {
			displayText := record.ResourceKey
			if displayText == "" {
				displayText = record.Command
			}

			Printf("%4d. [%s] %-8s %s\n",
				i+1,
				record.TimeStr,
				record.Operation,
				displayText)
		}

		return nil
	},
}
