package command

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"ttl-cli/db"
	"ttl-cli/i18n"
	"ttl-cli/models"

	"github.com/spf13/cobra"
)

var LogCmd = &cobra.Command{
	Use:   "log [content]",
	Short: i18n.T("command.log.short"),
	Long:  i18n.T("command.log.long"),
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		deleteID, _ := cmd.Flags().GetString("delete")
		listMode, _ := cmd.Flags().GetBool("list")

		if deleteID != "" {
			return runLogDelete(deleteID)
		}
		if listMode {
			return runLogList(cmd)
		}
		if len(args) == 0 {
			return cmd.Help()
		}
		return runLogAdd(cmd, args)
	},
}

func runLogAdd(cmd *cobra.Command, args []string) error {
	content := strings.Join(args, " ")
	tags, _ := cmd.Flags().GetStringArray("tag")

	now := time.Now()
	record := models.LogRecord{
		ID:        now.UnixNano(),
		Content:   content,
		Tags:      tags,
		CreatedAt: now.Format("2006-01-02 15:04:05"),
		Date:      now.Format("2006-01-02"),
	}

	if err := db.SaveLogRecord(record); err != nil {
		return fmt.Errorf(i18n.T("command.log.error_save"), err)
	}

	Printf(i18n.T("command.log.success"), record.CreatedAt)
	return nil
}

func runLogList(cmd *cobra.Command) error {
	fromDate, _ := cmd.Flags().GetString("from")
	toDate, _ := cmd.Flags().GetString("to")
	week, _ := cmd.Flags().GetBool("week")
	month, _ := cmd.Flags().GetBool("month")
	dateFlag, _ := cmd.Flags().GetString("date")
	tagFilter, _ := cmd.Flags().GetStringArray("tag")

	now := time.Now()
	var startDate, endDate string

	switch {
	case fromDate != "" || toDate != "":
		startDate = fromDate
		endDate = toDate
	case week:
		weekday := now.Weekday()
		if weekday == time.Sunday {
			weekday = 7
		}
		monday := now.AddDate(0, 0, -int(weekday-time.Monday))
		startDate = monday.Format("2006-01-02")
		endDate = now.Format("2006-01-02")
	case month:
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
		endDate = now.Format("2006-01-02")
	case dateFlag != "":
		startDate = dateFlag
		endDate = dateFlag
	default:
		today := now.Format("2006-01-02")
		startDate = today
		endDate = today
	}

	records, err := db.GetLogRecords(startDate, endDate)
	if err != nil {
		return fmt.Errorf(i18n.T("command.log.error_query"), err)
	}

	if len(tagFilter) > 0 {
		var filtered []models.LogRecord
		for _, r := range records {
			for _, t := range r.Tags {
				for _, filterTag := range tagFilter {
					if t == filterTag {
						filtered = append(filtered, r)
						break
					}
				}
			}
		}
		records = filtered
	}

	if len(records) == 0 {
		Println(i18n.T("command.log.no_records"))
		return nil
	}

	var currentDate string
	for _, r := range records {
		if r.Date != currentDate {
			if currentDate != "" {
				Println()
			}
			Printf("📅 %s\n", r.Date)
			currentDate = r.Date
		}

		timeStr := r.CreatedAt
		if len(timeStr) > 10 {
			timeStr = timeStr[11:]
		}

		tagStr := ""
		if len(r.Tags) > 0 {
			tagStr = " [" + strings.Join(r.Tags, ", ") + "]"
		}

		Printf("  [%s] %s%s\n", timeStr, r.Content, tagStr)
	}

	return nil
}

func runLogDelete(deleteID string) error {
	id, err := strconv.ParseInt(deleteID, 10, 64)
	if err != nil {
		return fmt.Errorf(i18n.T("command.log.invalid_id"), deleteID)
	}

	if err := db.DeleteLogRecord(id); err != nil {
		return err
	}

	Println(i18n.T("command.log.delete_success"))
	return nil
}

func init() {
	LogCmd.Flags().StringArrayP("tag", "t", nil, i18n.T("command.log.flag_tag"))
	LogCmd.Flags().BoolP("list", "l", false, i18n.T("command.log.flag_list"))
	LogCmd.Flags().StringP("date", "d", "", i18n.T("command.log.flag_date"))
	LogCmd.Flags().BoolP("week", "w", false, i18n.T("command.log.flag_week"))
	LogCmd.Flags().BoolP("month", "m", false, i18n.T("command.log.flag_month"))
	LogCmd.Flags().String("from", "", i18n.T("command.log.flag_from"))
	LogCmd.Flags().String("to", "", i18n.T("command.log.flag_to"))
	LogCmd.Flags().String("delete", "", i18n.T("command.log.flag_delete"))
}
