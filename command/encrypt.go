package command

import (
	"bufio"
	"fmt"
	"os"
	"ttl-cli/crypto"
	"ttl-cli/db"
	"ttl-cli/i18n"

	"github.com/spf13/cobra"
)

var EncryptCmd = &cobra.Command{
	Use:   "encrypt",
	Short: i18n.T("command.encrypt.short"),
	Long:  i18n.T("command.encrypt.long"),
	RunE: func(cmd *cobra.Command, args []string) error {
		migrate, _ := cmd.Flags().GetBool("migrate")
		force, _ := cmd.Flags().GetBool("force")

		if ls, ok := db.Stor.(*db.LocalStorage); ok && ls.IsEncryptionEnabled() {
			Println(i18n.T("command.encrypt.enabled"))
			return nil
		}

		if !migrate && !force {
			resources, err := db.GetAllResources()
			if err == nil && len(resources) > 0 {
				Println(i18n.T("command.encrypt.migrate_prompt"))
				reader := bufio.NewReader(os.Stdin)
				answer, _ := reader.ReadString('\n')
				if answer != "Y\n" && answer != "y\n" && answer != "Y" && answer != "y" {
					Println(i18n.T("command.encrypt.cancelled"))
					return nil
				}
				migrate = true
			}
		}

		if ls, ok := db.Stor.(*db.LocalStorage); ok {
			if migrate || force {
				if err := ls.EnableEncryption(); err != nil {
					return fmt.Errorf("启用加密失败: %w", err)
				}
				Println(i18n.T("command.encrypt.success"))
				Printf(i18n.T("command.encrypt.backup_prompt")+"\n", crypto.GetKeyFilePath())
			} else {
				if err := crypto.InitEncryption(true); err != nil {
					return fmt.Errorf("初始化加密失败: %w", err)
				}
				Println(i18n.T("command.encrypt.key_generated"))
				Printf(i18n.T("command.encrypt.backup_prompt")+"\n", crypto.GetKeyFilePath())
			}
			return nil
		}

		return fmt.Errorf("加密功能仅支持本地存储模式")
	},
}

var DecryptCmd = &cobra.Command{
	Use:   "decrypt",
	Short: i18n.T("command.decrypt.short"),
	Long:  i18n.T("command.decrypt.long"),
	RunE: func(cmd *cobra.Command, args []string) error {
		keepKey, _ := cmd.Flags().GetBool("keep-key")

		ls, ok := db.Stor.(*db.LocalStorage)
		if !ok {
			return fmt.Errorf("解密功能仅支持本地存储模式")
		}

		if !ls.IsEncryptionEnabled() {
			Println(i18n.T("command.decrypt.not_enabled"))
			return nil
		}

		Println(i18n.T("command.decrypt.warning"))
		Println(i18n.T("command.decrypt.confirm_prompt"))
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		if answer != "y\n" && answer != "Y\n" && answer != "y" && answer != "Y" {
			Println(i18n.T("command.encrypt.cancelled"))
			return nil
		}

		if err := ls.DisableEncryption(); err != nil {
			return fmt.Errorf("禁用加密失败: %w", err)
		}

		if keepKey {
			Println(i18n.T("command.decrypt.data_decrypted") + "，" + i18n.T("command.decrypt.key_kept"))
		} else {
			Println(i18n.T("command.decrypt.success"))
		}

		return nil
	},
}

var KeyCmd = &cobra.Command{
	Use:   "key",
	Short: i18n.T("command.key.short"),
	Long:  i18n.T("command.key.long"),
}

var keyExportCmd = &cobra.Command{
	Use:   "export <path>",
	Short: i18n.T("command.key.export_success"),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := crypto.ExportKey(args[0]); err != nil {
			return fmt.Errorf("导出密钥失败: %w", err)
		}
		Printf(i18n.T("command.key.export_success")+"\n", args[0])
		return nil
	},
}

var keyImportCmd = &cobra.Command{
	Use:   "import <path>",
	Short: i18n.T("command.key.import_success"),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := crypto.ImportKey(args[0]); err != nil {
			return fmt.Errorf("导入密钥失败: %w", err)
		}
		Println(i18n.T("command.key.import_success"))
		return nil
	},
}

var keyVerifyCmd = &cobra.Command{
	Use:   "verify",
	Short: i18n.T("command.key.verify_success"),
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := crypto.VerifyKey(); err != nil {
			return fmt.Errorf("密钥验证失败: %w", err)
		}
		Println(i18n.T("command.key.verify_success"))
		return nil
	},
}

func init() {
	EncryptCmd.Flags().BoolP("migrate", "m", false, i18n.T("command.encrypt.flag_migrate"))
	EncryptCmd.Flags().BoolP("force", "f", false, i18n.T("command.encrypt.flag_force"))

	DecryptCmd.Flags().BoolP("keep-key", "", false, i18n.T("command.decrypt.flag_keep_key"))

	KeyCmd.AddCommand(keyExportCmd)
	KeyCmd.AddCommand(keyImportCmd)
	KeyCmd.AddCommand(keyVerifyCmd)
}
