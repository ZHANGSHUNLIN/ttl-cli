package command

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"ttl-cli/db"
	"ttl-cli/i18n"
	"ttl-cli/models"

	"github.com/spf13/cobra"
)

var ExportCmd = &cobra.Command{
	Use:   "export",
	Short: i18n.T("command.export.short"),
	Long:  i18n.T("command.export.long"),
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		outputPath, _ := cmd.Flags().GetString("output")
		exportType, _ := cmd.Flags().GetString("type")
		exportFormat, _ := cmd.Flags().GetString("format")
		withBOM, _ := cmd.Flags().GetBool("bom")

		toFile := outputPath != ""
		var out io.Writer
		if toFile {
			f, err := os.Create(outputPath)
			if err != nil {
				return fmt.Errorf(i18n.T("command.export.error_create_file"), err)
			}
			defer f.Close()
			out = f
		} else {
			out = os.Stdout
		}

		if withBOM && exportFormat == "csv" {
			if _, err := out.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
				return fmt.Errorf(i18n.T("command.export.error_write_bom"), err)
			}
		}

		var count int
		var err error

		switch exportFormat {
		case "json":
			count, err = exportJSON(out, exportType, toFile)
		case "csv":
			count, err = exportCSV(out, exportType)
		default:
			return fmt.Errorf(i18n.T("command.export.unsupported_format"), exportFormat)
		}

		if err != nil {
			return err
		}

		if toFile {
			Printf(i18n.T("command.export.success"), outputPath, count)
		}
		return nil
	},
}

func exportCSV(out io.Writer, exportType string) (int, error) {
	w := csv.NewWriter(out)
	var count int
	var err error

	switch exportType {
	case "resources":
		resources, e := db.GetAllResources()
		if e != nil {
			return 0, fmt.Errorf(i18n.T("command.export.error_fetch_resources"), e)
		}
		count, err = WriteResourcesCSV(w, resources)
	case "audit":
		records, e := db.GetAllAuditRecords()
		if e != nil {
			return 0, fmt.Errorf(i18n.T("command.export.error_fetch_audit"), e)
		}
		count, err = WriteAuditCSV(w, records)
	case "history":
		records, e := db.GetAllHistoryRecords()
		if e != nil {
			return 0, fmt.Errorf(i18n.T("command.export.error_fetch_history"), e)
		}
		count, err = WriteHistoryCSV(w, records)
	case "log":
		records, e := db.GetLogRecords("", "")
		if e != nil {
			return 0, fmt.Errorf(i18n.T("command.export.error_fetch_log"), e)
		}
		count, err = WriteLogCSV(w, records)
	default:
		return 0, fmt.Errorf(i18n.T("command.export.unsupported_type"), exportType)
	}

	if err != nil {
		return 0, err
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return 0, fmt.Errorf(i18n.T("command.export.error_write_csv"), err)
	}

	return count, nil
}

func exportJSON(out io.Writer, exportType string, toFile bool) (int, error) {
	var data interface{}
	var items []interface{}

	switch exportType {
	case "resources":
		resources, err := db.GetAllResources()
		if err != nil {
			return 0, fmt.Errorf(i18n.T("command.export.error_fetch_resources"), err)
		}
		for key, val := range resources {
			if key.Type == models.ORIGIN {
				items = append(items, map[string]interface{}{
					"key":   key.Key,
					"value": val.Val,
					"tags":  val.Tag,
				})
			}
		}
		data = map[string]interface{}{
			"type":        "resources",
			"exported_at": time.Now().Format(time.RFC3339),
			"items":       items,
		}
	case "audit":
		records, err := db.GetAllAuditRecords()
		if err != nil {
			return 0, fmt.Errorf(i18n.T("command.export.error_fetch_audit"), err)
		}
		for _, r := range records {
			items = append(items, r)
		}
		data = map[string]interface{}{
			"type":        "audit",
			"exported_at": time.Now().Format(time.RFC3339),
			"items":       items,
		}
	case "history":
		records, err := db.GetAllHistoryRecords()
		if err != nil {
			return 0, fmt.Errorf(i18n.T("command.export.error_fetch_history"), err)
		}
		for _, r := range records {
			items = append(items, r)
		}
		data = map[string]interface{}{
			"type":        "history",
			"exported_at": time.Now().Format(time.RFC3339),
			"items":       items,
		}
	case "log":
		records, err := db.GetLogRecords("", "")
		if err != nil {
			return 0, fmt.Errorf(i18n.T("command.export.error_fetch_log"), err)
		}
		for _, r := range records {
			items = append(items, r)
		}
		data = map[string]interface{}{
			"type":        "log",
			"exported_at": time.Now().Format(time.RFC3339),
			"items":       items,
		}
	default:
		return 0, fmt.Errorf(i18n.T("command.export.unsupported_type"), exportType)
	}

	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return 0, fmt.Errorf("JSON 编码失败: %w", err)
	}

	return len(items), nil
}

func WriteResourcesCSV(w *csv.Writer, resources map[models.ValJsonKey]models.ValJson) (int, error) {
	if err := w.Write([]string{"key", "value", "tags"}); err != nil {
		return 0, err
	}
	count := 0
	for key, val := range resources {
		if key.Type != models.ORIGIN {
			continue
		}
		tags := strings.Join(val.Tag, "|")
		if err := w.Write([]string{key.Key, val.Val, tags}); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func WriteAuditCSV(w *csv.Writer, records []models.AuditRecord) (int, error) {
	if err := w.Write([]string{"resource_key", "operation", "timestamp", "count"}); err != nil {
		return 0, err
	}
	for _, r := range records {
		if err := w.Write([]string{
			r.ResourceKey,
			r.Operation,
			fmt.Sprintf("%d", r.Timestamp),
			fmt.Sprintf("%d", r.Count),
		}); err != nil {
			return 0, err
		}
	}
	return len(records), nil
}

func WriteHistoryCSV(w *csv.Writer, records []models.HistoryRecord) (int, error) {
	if err := w.Write([]string{"id", "resource_key", "operation", "time", "command"}); err != nil {
		return 0, err
	}
	for _, r := range records {
		if err := w.Write([]string{
			fmt.Sprintf("%d", r.ID),
			r.ResourceKey,
			r.Operation,
			r.TimeStr,
			r.Command,
		}); err != nil {
			return 0, err
		}
	}
	return len(records), nil
}

func WriteLogCSV(w *csv.Writer, records []models.LogRecord) (int, error) {
	if err := w.Write([]string{"id", "content", "tags", "created_at", "date"}); err != nil {
		return 0, err
	}
	for _, r := range records {
		if err := w.Write([]string{
			fmt.Sprintf("%d", r.ID),
			r.Content,
			strings.Join(r.Tags, "|"),
			r.CreatedAt,
			r.Date,
		}); err != nil {
			return 0, err
		}
	}
	return len(records), nil
}

func init() {
	ExportCmd.Flags().StringP("output", "o", "", i18n.T("command.export.flag_output"))
	ExportCmd.Flags().StringP("type", "t", "resources", i18n.T("command.export.flag_type"))
	ExportCmd.Flags().StringP("format", "f", "csv", i18n.T("command.export.flag_format"))
	ExportCmd.Flags().Bool("bom", false, i18n.T("command.export.flag_bom"))
}
