package main

import (
	"errors"
	"ttl-cli/db"
	"ttl-cli/i18n"

	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate [source] [target]",
	Short: i18n.T("command.migrate.short"),
	Long:  i18n.T("command.migrate.long"),
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		sourceType := args[0]
		targetType := args[1]

		if sourceType != "local" && sourceType != "cloud" {
			return errors.New(i18n.T("command.migrate.invalid_source"))
		}
		if targetType != "local" && targetType != "cloud" {
			return errors.New(i18n.T("command.migrate.invalid_target"))
		}
		if sourceType == targetType {
			return errors.New(i18n.T("command.migrate.same_type"))
		}

		var sourceAPIURL, sourceAPIKey string
		var sourceTimeout int

		if sourceType == "cloud" {
			sourceAPIURL, _ = cmd.Flags().GetString("source-url")
			sourceAPIKey, _ = cmd.Flags().GetString("source-key")
			sourceTimeout, _ = cmd.Flags().GetInt("source-timeout")

			if sourceAPIURL == "" || sourceAPIKey == "" {
				return errors.New(i18n.T("command.migrate.need_source_config"))
			}
		}

		return db.MigrateData(
			sourceType, targetType,
			sourceAPIURL, sourceAPIKey, sourceTimeout,
			cloudAPIURL, cloudAPIKey, cloudTimeout,
			debug, confFile, confFile)
	},
}

func init() {
	migrateCmd.Flags().String("source-url", "", i18n.T("command.migrate.flag_source_url"))
	migrateCmd.Flags().String("source-key", "", i18n.T("command.migrate.flag_source_key"))
	migrateCmd.Flags().Int("source-timeout", 30, i18n.T("command.migrate.flag_source_timeout"))
}
