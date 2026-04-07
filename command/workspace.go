package command

import (
	"errors"
	"fmt"
	"sort"
	"ttl-cli/conf"
	"ttl-cli/db"
	"ttl-cli/i18n"

	"github.com/spf13/cobra"
)

var WorkspaceCmd = &cobra.Command{
	Use:   "workspace [subcommand]",
	Short: i18n.T("command.workspace.short"),
	Long:  i18n.T("command.workspace.long"),
}

var WorkspaceListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   i18n.T("command.workspace.list.short"),
	Long:    i18n.T("command.workspace.list.long"),
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		confFile := cmd.Context().Value("confFile").(string)
		names, current, err := conf.ListWorkspaces(confFile)
		if err != nil {
			return fmt.Errorf("failed to list workspaces: %w", err)
		}

		sort.Strings(names)

		for _, name := range names {
			if name == current {
				Printf("* %s", name)
			} else {
				Printf("  %s", name)
			}

			dbPath, storageType, count, err := conf.GetWorkspaceInfo(confFile, name)
			if err == nil && dbPath != "" {
				if count > 0 {
					Printf("    (db: %s, type: %s)", dbPath, storageType)
				} else {
					Printf("    (db: %s)", dbPath)
				}
			}
			Println()
		}

		if len(names) == 0 {
			Println(i18n.T("command.workspace.list.empty"))
		}

		return nil
	},
}

var WorkspaceSwitchCmd = &cobra.Command{
	Use:   "switch <name>",
	Short: i18n.T("command.workspace.switch.short"),
	Long:  i18n.T("command.workspace.switch.long"),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		confFile := cmd.Context().Value("confFile").(string)

		current, err := conf.GetCurrentWorkspace(confFile)
		if err == nil && current == name {
			Println(i18n.T("command.workspace.switch.already_current", name))
			return nil
		}

		if err := conf.SwitchWorkspace(confFile, name); err != nil {
			return err
		}

		if err := db.CloseDB(); err != nil {
			return fmt.Errorf("failed to close current database: %w", err)
		}

		Println(i18n.T("command.workspace.switch.success", name))
		return nil
	},
}

var WorkspaceCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: i18n.T("command.workspace.create.short"),
	Long:  i18n.T("command.workspace.create.long"),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		confFile := cmd.Context().Value("confFile").(string)

		dbPath, err := conf.CreateWorkspace(confFile, name)
		if err != nil {
			return err
		}

		Println(i18n.T("command.workspace.create.success", name, dbPath))
		return nil
	},
}

var WorkspaceDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: i18n.T("command.workspace.delete.short"),
	Long:  i18n.T("command.workspace.delete.long"),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		confFile := cmd.Context().Value("confFile").(string)

		current, err := conf.GetCurrentWorkspace(confFile)
		if err == nil && current == name {
			return errors.New(i18n.T("command.workspace.delete.cannot_delete_current"))
		}

		if err := conf.DeleteWorkspace(confFile, name); err != nil {
			return err
		}

		Println(i18n.T("command.workspace.delete.success", name))
		return nil
	},
}

var WorkspaceCurrentCmd = &cobra.Command{
	Use:     "current",
	Aliases: []string{"get"},
	Short:   i18n.T("command.workspace.current.short"),
	Long:    i18n.T("command.workspace.current.long"),
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		confFile := cmd.Context().Value("confFile").(string)
		workspaceName, err := conf.GetCurrentWorkspace(confFile)
		if err != nil {
			return err
		}

		dbPath, storageType, count, err := conf.GetWorkspaceInfo(confFile, workspaceName)
		if err != nil {
			return err
		}

		Println(i18n.T("command.workspace.current.label"), workspaceName)
		Printf("  Database:     %s\n", dbPath)
		Printf("  Storage Type: %s\n", storageType)
		if count > 0 {
			Println("  Status:       active (database exists)")
		} else {
			Println("  Status:       inactive (database not created)")
		}

		return nil
	},
}

var WorkspaceShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: i18n.T("command.workspace.show.short"),
	Long:  i18n.T("command.workspace.show.long"),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		confFile := cmd.Context().Value("confFile").(string)

		current, err := conf.GetCurrentWorkspace(confFile)
		if err != nil {
			current = ""
		}

		dbPath, storageType, count, err := conf.GetWorkspaceInfo(confFile, name)
		if err != nil {
			return err
		}

		Println("Workspace:", name)
		Printf("  Database:     %s\n", dbPath)
		Printf("  Storage Type: %s\n", storageType)
		if count > 0 {
			Println("  Status:       active")

			storageType := storageType
			if storageType == "" {
				storageType = "sqlite"
			}
			tempStorage := db.NewSQLiteStorage()
			tempStorage.SetDBPath(dbPath)
			if err := tempStorage.Init(); err == nil {
				resources, _ := tempStorage.GetAllResources()
				Printf("  Resources:    %d\n", len(resources))
				tempStorage.Close()
			} else {
				Println("  Resources:    0")
			}
		} else {
			Println("  Status:       inactive")
			Println("  Resources:    0")
		}

		if name == current {
			Println("  Status:       *active*")
		}

		return nil
	},
}

var WsCmd = &cobra.Command{
	Use:     "ws <name>",
	Short:   "Switch workspace (alias for 'workspace switch')",
	Long:    "Quickly switch to a workspace. Alias for 'ttl workspace switch <name>'",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		confFile := cmd.Context().Value("confFile").(string)

		current, err := conf.GetCurrentWorkspace(confFile)
		if err == nil && current == name {
			Println(i18n.T("command.workspace.switch.already_current", name))
			return nil
		}

		if err := conf.SwitchWorkspace(confFile, name); err != nil {
			return err
		}

		if err := db.CloseDB(); err != nil {
			return fmt.Errorf("failed to close current database: %w", err)
		}

		Println(i18n.T("command.workspace.switch.success", name))
		return nil
	},
}

func init() {
	WorkspaceCmd.AddCommand(WorkspaceListCmd)
	WorkspaceCmd.AddCommand(WorkspaceSwitchCmd)
	WorkspaceCmd.AddCommand(WorkspaceCreateCmd)
	WorkspaceCmd.AddCommand(WorkspaceDeleteCmd)
	WorkspaceCmd.AddCommand(WorkspaceCurrentCmd)
	WorkspaceCmd.AddCommand(WorkspaceShowCmd)
}
