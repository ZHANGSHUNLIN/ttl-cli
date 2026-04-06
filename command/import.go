package command

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"ttl-cli/db"
	"ttl-cli/i18n"
	"ttl-cli/models"

	"github.com/spf13/cobra"
)

var ImportCmd = &cobra.Command{
	Use:   "import [file]",
	Short: "Import data from CSV or JSON file",
	Long:  "Import data from CSV or JSON file into the database. Supports resources, audit, history, and log data.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		importType, _ := cmd.Flags().GetString("type")
		importFormat, _ := cmd.Flags().GetString("format")
		mergeMode, _ := cmd.Flags().GetBool("merge")

		var input *os.File
		var err error

		if len(args) == 0 || args[0] == "-" {
			input = os.Stdin
		} else {
			input, err = os.Open(args[0])
			if err != nil {
				return fmt.Errorf("无法打开文件 %s: %w", args[0], err)
			}
			defer input.Close()
		}

		if importFormat == "auto" && len(args) == 1 {
			if strings.HasSuffix(args[0], ".json") {
				importFormat = "json"
			} else if strings.HasSuffix(args[0], ".csv") {
				importFormat = "csv"
			}
		}
		if importFormat == "auto" {
			importFormat = "csv"
		}

		var added, skipped, failed int
		switch importFormat {
		case "csv":
			added, skipped, failed, err = importCSV(input, importType, mergeMode)
		case "json":
			added, skipped, failed, err = importJSON(input, importType, mergeMode)
		default:
			return fmt.Errorf("不支持的格式: %s (支持: csv, json)", importFormat)
		}

		if err != nil {
			return err
		}

		Printf(i18n.T("command.import.success"), added, skipped, failed)
		if failed > 0 {
			return errors.New(i18n.T("command.import.partial_failed"))
		}
		return nil
	},
}

func importCSV(input *os.File, importType string, mergeMode bool) (added, skipped, failed int, err error) {
	r := csv.NewReader(input)
	records, err := r.ReadAll()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("读取 CSV 失败: %w", err)
	}

	if len(records) < 2 {
		return 0, 0, 0, fmt.Errorf("CSV 文件为空或只有表头")
	}

	headers := records[0]

	switch importType {
	case "resources":
		return importResourcesFromCSV(headers, records[1:], mergeMode)
	case "audit":
		return importAuditFromCSV(headers, records[1:], mergeMode)
	case "history":
		return importHistoryFromCSV(headers, records[1:], mergeMode)
	case "log":
		return importLogFromCSV(headers, records[1:], mergeMode)
	default:
		return 0, 0, 0, fmt.Errorf("不支持的导入类型: %s", importType)
	}
}

func importResourcesFromCSV(headers []string, records [][]string, mergeMode bool) (added, skipped, failed int, err error) {
	keyIdx := -1
	valueIdx := -1
	tagsIdx := -1
	for i, h := range headers {
		h = strings.ToLower(strings.TrimSpace(h))
		switch h {
		case "key":
			keyIdx = i
		case "value":
			valueIdx = i
		case "tags":
			tagsIdx = i
		}
	}

	if keyIdx == -1 || valueIdx == -1 {
		return 0, 0, 0, fmt.Errorf("CSV 缺少必要的列 (key, value)")
	}

	existingResources := make(map[string]bool)
	if mergeMode {
		allResources, e := db.GetAllResources()
		if e != nil {
			return 0, 0, 0, fmt.Errorf("获取现有资源失败: %w", e)
		}
		for key := range allResources {
			if key.Type == models.ORIGIN {
				existingResources[key.Key] = true
			}
		}
	}

	for _, row := range records {
		if len(row) < maxInt(keyIdx, valueIdx)+1 {
			failed++
			continue
		}

		key := strings.TrimSpace(row[keyIdx])
		value := strings.TrimSpace(row[valueIdx])
		if key == "" {
			failed++
			continue
		}

		if mergeMode && existingResources[key] {
			skipped++
			continue
		}

		var tags []string
		if tagsIdx >= 0 && tagsIdx < len(row) {
			tagsStr := strings.TrimSpace(row[tagsIdx])
			if tagsStr != "" {
				tags = strings.Split(tagsStr, "|")
			}
		}

		vjk := models.ValJsonKey{Key: key, Type: models.ORIGIN}
		if err := db.SaveResource(vjk, models.ValJson{Val: value, Tag: tags}); err != nil {
			failed++
			continue
		}

		added++
	}

	return added, skipped, failed, nil
}

func importAuditFromCSV(headers []string, records [][]string, mergeMode bool) (added, skipped, failed int, err error) {
	for _, row := range records {
		if len(row) < 4 {
			failed++
			continue
		}

		timestamp, e := strconv.ParseInt(strings.TrimSpace(row[2]), 10, 64)
		if e != nil {
			failed++
			continue
		}

		count, e := strconv.Atoi(strings.TrimSpace(row[3]))
		if e != nil {
			count = 1
		}

		record := models.AuditRecord{
			ResourceKey: strings.TrimSpace(row[0]),
			Operation:   strings.TrimSpace(row[1]),
			Timestamp:   timestamp,
			Count:       count,
		}

		if err := db.Stor.SaveAuditRecord(record); err != nil {
			failed++
			continue
		}

		added++
	}
	return added, skipped, failed, nil
}

func importHistoryFromCSV(headers []string, records [][]string, mergeMode bool) (added, skipped, failed int, err error) {
	for _, row := range records {
		if len(row) < 5 {
			failed++
			continue
		}

		id, e := strconv.ParseInt(strings.TrimSpace(row[0]), 10, 64)
		if e != nil {
			failed++
			continue
		}

		record := models.HistoryRecord{
			ID:          id,
			ResourceKey: strings.TrimSpace(row[1]),
			Operation:   strings.TrimSpace(row[2]),
			Timestamp:   0,
			TimeStr:     strings.TrimSpace(row[3]),
			Command:     strings.TrimSpace(row[4]),
		}

		if err := db.Stor.SaveHistoryRecord(record); err != nil {
			failed++
			continue
		}

		added++
	}
	return added, skipped, failed, nil
}

func importLogFromCSV(headers []string, records [][]string, mergeMode bool) (added, skipped, failed int, err error) {
	for _, row := range records {
		if len(row) < 5 {
			failed++
			continue
		}

		id, e := strconv.ParseInt(strings.TrimSpace(row[0]), 10, 64)
		if e != nil {
			failed++
			continue
		}

		var tags []string
		if len(row) > 2 {
			tagsStr := strings.TrimSpace(row[2])
			if tagsStr != "" {
				tags = strings.Split(tagsStr, "|")
			}
		}

		record := models.LogRecord{
			ID:        id,
			Content:   strings.TrimSpace(row[1]),
			Tags:      tags,
			CreatedAt: strings.TrimSpace(row[3]),
			Date:      strings.TrimSpace(row[4]),
		}

		if err := db.Stor.SaveLogRecord(record); err != nil {
			failed++
			continue
		}

		added++
	}
	return added, skipped, failed, nil
}

func importJSON(input *os.File, importType string, mergeMode bool) (added, skipped, failed int, err error) {
	var data struct {
		Type       string `json:"type"`
		ExportedAt string `json:"exported_at"`
		Items      []interface{}
	}

	decoder := json.NewDecoder(input)
	if err := decoder.Decode(&data); err != nil {
		return 0, 0, 0, fmt.Errorf("解析 JSON 失败: %w", err)
	}

	existingResources := make(map[string]bool)
	if mergeMode && importType == "resources" {
		allResources, e := db.GetAllResources()
		if e != nil {
			return 0, 0, 0, fmt.Errorf("获取现有资源失败: %w", e)
		}
		for key := range allResources {
			if key.Type == models.ORIGIN {
				existingResources[key.Key] = true
			}
		}
	}

	for _, item := range data.Items {
		switch importType {
		case "resources":
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				failed++
				continue
			}

			key, _ := itemMap["key"].(string)
			value, _ := itemMap["value"].(string)
			tagsArray, _ := itemMap["tags"].([]interface{})

			if key == "" {
				failed++
				continue
			}

			if mergeMode && existingResources[key] {
				skipped++
				continue
			}

			var tags []string
			for _, t := range tagsArray {
				if tagStr, ok := t.(string); ok {
					tags = append(tags, tagStr)
				}
			}

			vjk := models.ValJsonKey{Key: key, Type: models.ORIGIN}
			if err := db.SaveResource(vjk, models.ValJson{Val: value, Tag: tags}); err != nil {
				failed++
				continue
			}

			added++

		case "audit":
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				failed++
				continue
			}

			timestamp := int64(itemMap["timestamp"].(float64))
			count := 1
			if c, ok := itemMap["count"].(float64); ok {
				count = int(c)
			}

			record := models.AuditRecord{
				ResourceKey: itemMap["resource_key"].(string),
				Operation:   itemMap["operation"].(string),
				Timestamp:   timestamp,
				Count:       count,
			}

			if err := db.Stor.SaveAuditRecord(record); err != nil {
				failed++
				continue
			}
			added++

		case "history":
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				failed++
				continue
			}

			id := int64(itemMap["id"].(float64))
			record := models.HistoryRecord{
				ID:          id,
				ResourceKey: itemMap["resource_key"].(string),
				Operation:   itemMap["operation"].(string),
				Timestamp:   0,
				TimeStr:     itemMap["time"].(string),
				Command:     itemMap["command"].(string),
			}

			if err := db.Stor.SaveHistoryRecord(record); err != nil {
				failed++
				continue
			}
			added++

		case "log":
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				failed++
				continue
			}

			id := int64(itemMap["id"].(float64))
			tagsArray, _ := itemMap["tags"].([]interface{})
			var tags []string
			for _, t := range tagsArray {
				if tagStr, ok := t.(string); ok {
					tags = append(tags, tagStr)
				}
			}

			record := models.LogRecord{
				ID:        id,
				Content:   itemMap["content"].(string),
				Tags:      tags,
				CreatedAt: itemMap["created_at"].(string),
				Date:      itemMap["date"].(string),
			}

			if err := db.Stor.SaveLogRecord(record); err != nil {
				failed++
				continue
			}
			added++
		}
	}

	return added, skipped, failed, nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func init() {
	ImportCmd.Flags().StringP("type", "t", "resources", "Import type: resources, audit, history, or log")
	ImportCmd.Flags().StringP("format", "f", "auto", "Import format: csv, json, or auto")
	ImportCmd.Flags().Bool("merge", false, "Merge mode: skip existing records instead of overwriting")
}
