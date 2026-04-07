package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
	"ttl-cli/api"
	"ttl-cli/command"
	"ttl-cli/conf"
	"ttl-cli/db"
	"ttl-cli/i18n"
	ttlmcp "ttl-cli/mcp"
	ttlsync "ttl-cli/sync"

	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ttl",
	Short: i18n.T("root.short"),
	Long:  i18n.T("root.long"),
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var debug bool
var storageType string
var cloudAPIURL string
var cloudAPIKey string
var cloudTimeout int
var confFile string

func init() {
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "D", false, i18n.T("root.flag_debug"))
	rootCmd.PersistentFlags().StringVar(&storageType, "storage", "sqlite", i18n.T("root.flag_storage"))
	rootCmd.PersistentFlags().StringVar(&cloudAPIURL, "cloud-url", "", i18n.T("root.flag_cloud_url"))
	rootCmd.PersistentFlags().StringVar(&cloudAPIKey, "cloud-key", "", i18n.T("root.flag_cloud_key"))
	rootCmd.PersistentFlags().IntVar(&cloudTimeout, "cloud-timeout", 30, i18n.T("root.flag_cloud_timeout"))
	rootCmd.PersistentFlags().StringVar(&confFile, "conf", "", i18n.T("root.flag_conf"))

	rootCmd.AddCommand(command.InitCmd)
	rootCmd.AddCommand(command.AddCmd)
	rootCmd.AddCommand(command.GetCmd)
	rootCmd.AddCommand(command.OpenCmd)
	rootCmd.AddCommand(command.UpdateCmd)
	rootCmd.AddCommand(command.DelCmd)
	rootCmd.AddCommand(command.TagCmd)
	rootCmd.AddCommand(command.DtagCmd)
	rootCmd.AddCommand(command.TagsCmd)
	rootCmd.AddCommand(command.RenameCmd)
	rootCmd.AddCommand(command.ConfigCmd)
	rootCmd.AddCommand(command.VersionCmd)
	rootCmd.AddCommand(command.EncryptCmd)
	rootCmd.AddCommand(command.DecryptCmd)
	rootCmd.AddCommand(command.KeyCmd)
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(command.AuditCmd)
	rootCmd.AddCommand(command.HistoryCmd)
	rootCmd.AddCommand(command.ExportCmd)
	rootCmd.AddCommand(command.ImportCmd)
	rootCmd.AddCommand(command.AICmd)
	rootCmd.AddCommand(command.AIContextCmd)
	rootCmd.AddCommand(command.LogCmd)
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(command.WorkspaceCmd)
	rootCmd.AddCommand(command.WsCmd)
}

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: i18n.T("command.mcp.short"),
	Long:  i18n.T("command.mcp.long"),
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := ttlmcp.NewTtlMCPServer()
		defer db.CloseDB()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			cancel()
		}()

		errCh := make(chan error, 1)
		go func() {
			errCh <- mcpserver.ServeStdio(s)
		}()

		select {
		case err := <-errCh:
			return err
		case <-ctx.Done():
			return nil
		}
	},
}

var serverPort int
var serverDataDir string

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: i18n.T("command.server.short"),
	Long:  i18n.T("command.server.long"),
	RunE: func(cmd *cobra.Command, args []string) error {
		return api.StartServer(serverPort, serverDataDir)
	},
}

func init() {
	home, _ := os.UserHomeDir()
	defaultDataDir := filepath.Join(home, ".ttl")

	serverCmd.Flags().IntVar(&serverPort, "port", 8080, i18n.T("command.server.flag_port"))
	serverCmd.Flags().StringVar(&serverDataDir, "data-dir", defaultDataDir, i18n.T("command.server.flag_data_dir"))

	serverCmd.AddCommand(userCmd)
	userCmd.AddCommand(userAddCmd)
	userCmd.AddCommand(userListCmd)
	userCmd.AddCommand(userDisableCmd)
	userCmd.AddCommand(userEnableCmd)
	userCmd.AddCommand(userResetKeyCmd)
	userCmd.AddCommand(userDeleteCmd)

	syncCmd.Flags().StringVar(&syncDirection, "direction", "auto", i18n.T("command.sync.flag_direction"))
	syncCmd.Flags().BoolVar(&syncDryRun, "dry-run", false, i18n.T("command.sync.flag_dry_run"))
}

var userCmd = &cobra.Command{
	Use:   "user",
	Short: i18n.T("command.server.user.short"),
	Long:  i18n.T("command.server.user.long"),
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var userAddID string
var userAddName string

var userAddCmd = &cobra.Command{
	Use:   "add",
	Short: i18n.T("command.server.user_add.short"),
	RunE: func(cmd *cobra.Command, args []string) error {
		store := db.NewUserStore(serverDataDir + "/users.json")
		if err := store.Load(); err != nil {
			return err
		}
		user, err := store.AddUser(userAddID, userAddName)
		if err != nil {
			return err
		}
		fmt.Println(i18n.T("command.server.user_add.success"))
		fmt.Printf("  ID:      %s\n", user.ID)
		fmt.Printf("  Name:    %s\n", user.Name)
		fmt.Printf("  API Key: %s\n", user.APIKey)
		fmt.Println(i18n.T("command.server.user_add.warn_keep_key"))
		return nil
	},
}

func init() {
	userAddCmd.Flags().StringVar(&userAddID, "id", "", i18n.T("command.server.user_add.flag_id"))
	userAddCmd.Flags().StringVar(&userAddName, "name", "", i18n.T("command.server.user_add.flag_name"))
	_ = userAddCmd.MarkFlagRequired("id")
	_ = userAddCmd.MarkFlagRequired("name")
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: i18n.T("command.server.user_list.short"),
	RunE: func(cmd *cobra.Command, args []string) error {
		store := db.NewUserStore(serverDataDir + "/users.json")
		if err := store.Load(); err != nil {
			return err
		}
		users := store.ListUsers()
		if len(users) == 0 {
			fmt.Println(i18n.T("command.server.user_list.no_users"))
			return nil
		}
		fmt.Printf("%-16s %-16s %-8s %-12s %s\n", "ID", "Name", "Active", "API Key", "Created")
		for _, u := range users {
			keyPreview := u.APIKey[:8] + "****"
			created := time.Unix(u.CreatedAt, 0).Format("2006-01-02 15:04:05")
			fmt.Printf("%-16s %-16s %-8v %-12s %s\n", u.ID, u.Name, u.Active, keyPreview, created)
		}
		return nil
	},
}

var userDisableID string

var userDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: i18n.T("command.server.user_disable.short"),
	RunE: func(cmd *cobra.Command, args []string) error {
		store := db.NewUserStore(serverDataDir + "/users.json")
		if err := store.Load(); err != nil {
			return err
		}
		if err := store.SetActive(userDisableID, false); err != nil {
			return err
		}
		fmt.Printf(i18n.T("command.server.user_disable.success"), userDisableID)
		return nil
	},
}

func init() {
	userDisableCmd.Flags().StringVar(&userDisableID, "id", "", i18n.T("command.server.user_disable.flag_id"))
	_ = userDisableCmd.MarkFlagRequired("id")
}

var userEnableID string

var userEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: i18n.T("command.server.user_enable.short"),
	RunE: func(cmd *cobra.Command, args []string) error {
		store := db.NewUserStore(serverDataDir + "/users.json")
		if err := store.Load(); err != nil {
			return err
		}
		if err := store.SetActive(userEnableID, true); err != nil {
			return err
		}
		fmt.Printf(i18n.T("command.server.user_enable.success"), userEnableID)
		return nil
	},
}

func init() {
	userEnableCmd.Flags().StringVar(&userEnableID, "id", "", i18n.T("command.server.user_enable.flag_id"))
	_ = userEnableCmd.MarkFlagRequired("id")
}

var userResetKeyID string

var userResetKeyCmd = &cobra.Command{
	Use:   "reset-key",
	Short: i18n.T("command.server.user_reset_key.short"),
	RunE: func(cmd *cobra.Command, args []string) error {
		store := db.NewUserStore(serverDataDir + "/users.json")
		if err := store.Load(); err != nil {
			return err
		}
		newKey, err := store.ResetKey(userResetKeyID)
		if err != nil {
			return err
		}
		fmt.Printf(i18n.T("command.server.user_reset_key.success"), userResetKeyID)
		fmt.Printf(i18n.T("command.server.user_reset_key.new_key"), newKey)
		return nil
	},
}

func init() {
	userResetKeyCmd.Flags().StringVar(&userResetKeyID, "id", "", i18n.T("command.server.user_reset_key.flag_id"))
	_ = userResetKeyCmd.MarkFlagRequired("id")
}

var userDeleteID string
var userDeleteConfirm bool

var userDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: i18n.T("command.server.user_delete.short"),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !userDeleteConfirm {
			return errors.New(i18n.T("command.server.user_delete.need_confirm"))
		}

		store := db.NewUserStore(serverDataDir + "/users.json")
		if err := store.Load(); err != nil {
			return err
		}
		if err := store.DeleteUser(userDeleteID); err != nil {
			return err
		}

		tenantMgr := db.NewTenantStorageManager(serverDataDir + "/tenants")
		if err := tenantMgr.RemoveStorage(userDeleteID); err != nil {
			fmt.Printf(i18n.T("command.server.user_delete.warn_delete_data"), err)
		}

		fmt.Printf(i18n.T("command.server.user_delete.success"), userDeleteID)
		return nil
	},
}

func init() {
	userDeleteCmd.Flags().StringVar(&userDeleteID, "id", "", i18n.T("command.server.user_delete.flag_id"))
	userDeleteCmd.Flags().BoolVar(&userDeleteConfirm, "confirm", false, i18n.T("command.server.user_delete.flag_confirm"))
	_ = userDeleteCmd.MarkFlagRequired("id")
}

var syncDirection string
var syncDryRun bool

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: i18n.T("command.sync.short"),
	Long:  i18n.T("command.sync.long"),
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if cloudAPIURL == "" {
			return errors.New(i18n.T("command.sync.need_cloud_url"))
		}

		localResources, err := db.GetAllResources()
		if err != nil {
			return fmt.Errorf(i18n.T("command.sync.error_fetch_local"), err)
		}

		remoteStorage := db.NewCloudStorage(cloudAPIURL, cloudAPIKey, cloudTimeout)
		if err := remoteStorage.Init(); err != nil {
			return fmt.Errorf(i18n.T("command.sync.error_connect_remote"), err)
		}
		defer remoteStorage.Close()

		remoteResources, err := remoteStorage.GetAllResources()
		if err != nil {
			return fmt.Errorf(i18n.T("command.sync.error_fetch_remote"), err)
		}

		diff := ttlsync.ComputeDiff(localResources, remoteResources)
		ttlsync.PrintDiff(diff, cloudAPIURL)

		if diff.InSync {
			return nil
		}

		if syncDryRun {
			fmt.Println(i18n.T("command.sync.dry_run_notice"))
			return nil
		}

		switch syncDirection {
		case "pull":
			return ttlsync.ExecutePull(diff, db.Stor, remoteStorage, false)
		case "push":
			return ttlsync.ExecutePush(diff, db.Stor, remoteStorage, false)
		case "auto":
			fmt.Println(i18n.T("command.sync.choose_operation"))
			fmt.Println(i18n.T("command.sync.option_pull"))
			fmt.Println(i18n.T("command.sync.option_push"))
			fmt.Println(i18n.T("command.sync.option_skip"))
			fmt.Print("> ")
			var choice string
			if _, err := fmt.Scan(&choice); err != nil {
				return fmt.Errorf(i18n.T("command.sync.invalid_input"), err)
			}
			switch choice {
			case "pull":
				return ttlsync.ExecutePull(diff, db.Stor, remoteStorage, false)
			case "push":
				return ttlsync.ExecutePush(diff, db.Stor, remoteStorage, false)
			case "skip":
				fmt.Println(i18n.T("command.sync.skipped"))
				return nil
			default:
				return fmt.Errorf(i18n.T("command.sync.invalid_choice"), choice)
			}
		default:
			return fmt.Errorf(i18n.T("command.sync.invalid_direction"), syncDirection)
		}
	},
}

func main() {
	// Initialize i18n
	if err := i18n.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to initialize i18n: %v\n", err)
	}

	// Update command descriptions after i18n initialization
	updateCommandDescriptions(rootCmd)

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == "server" || cmd.Parent() != nil && cmd.Parent().Name() == "user" {
			return nil
		}

		skipDBInit := false
		if cmd.Name() == "workspace" || (cmd.Parent() != nil && cmd.Parent().Name() == "workspace") {
			skipDBInit = true
		}
		if cmd.Name() == "ws" {
			skipDBInit = true
		}

		ctx := context.WithValue(cmd.Context(), "debug", debug)
		ctx = context.WithValue(ctx, "confFile", confFile)

		aiConf, err := conf.LoadAIConfig(confFile)
		if err == nil {
			ctx = context.WithValue(ctx, "ai_config", aiConf)
		}
		if !skipDBInit {
			actualStorageType := storageType
			if storageType == "sqlite" && cmd.Flags().Changed("storage") == false {
				ttlConf, err := conf.GetTtlConfFromFile(confFile)
				if err == nil {
					if ttlConf.Workspace != "" {
						if ws, ok := ttlConf.Workspaces[ttlConf.Workspace]; ok && ws.StorageType != "" {
							actualStorageType = ws.StorageType
						}
					}
					if actualStorageType == "sqlite" && ttlConf.StorageType != "" {
						actualStorageType = ttlConf.StorageType
					}
				}
			}

			if err := db.InitDB(actualStorageType, cloudAPIURL, cloudAPIKey, cloudTimeout, confFile); err != nil {
				return fmt.Errorf(i18n.T("error.init_db"), err)
			}

			replaceSpecialValuesFromHistory(args)

			if !cmd.HasSubCommands() && cmd.Name() != "history" && cmd.Name() != "audit" && cmd.Name() != "export" && cmd.Name() != "mcp" && cmd.Name() != "server" && cmd.Name() != "sync" && cmd.Name() != "log" && cmd.Name() != "tags" {
				resourceKey := ""
				if len(args) > 0 {
					resourceKey = args[0]
				}
				if err := db.RecordCommandHistory(cmd.Name(), resourceKey, debug); err != nil && debug {
					fmt.Printf(i18n.T("error.record_history"), err)
				}
			}
		}
		cmd.SetContext(ctx)
		return nil
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := db.CloseDB(); err != nil && debug {
		fmt.Printf(i18n.T("error.close_db"), err)
	}
}

func replaceSpecialValuesFromHistory(args []string) {
	if len(args) != 1 {
		return
	}
	charsCount := countSpecialChars(args[0])
	if charsCount > 0 {
		record, err := db.GetHistoryRecords(charsCount - 1)
		if err != nil {
			fmt.Printf(i18n.T("error.get_history"), err)
		}
		args[0] = record.ResourceKey
	}
}

func countSpecialChars(input string) int {
	count := 0
	for _, char := range input {
		if char == '~' || char == '^' {
			count++
		} else {
			return 0
		}
	}
	return count
}

// after i18n has been initialized
func updateCommandDescriptions(cmd *cobra.Command) {
	// Update Short and Long if they contain i18n keys
	if cmd.Short != "" {
		cmd.Short = i18n.T(cmd.Short)
	}
	if cmd.Long != "" {
		cmd.Long = i18n.T(cmd.Long)
	}

	// modified after creation. They will display the i18n key name but

	for _, subCmd := range cmd.Commands() {
		updateCommandDescriptions(subCmd)
	}
}
