package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"
	"ttl-cli/db"
	"ttl-cli/i18n"
	"ttl-cli/models"
	"ttl-cli/util"

	"github.com/mark3labs/mcp-go/mcp"
)

func storageFromCtx(ctx context.Context) db.Storage {
	if s := db.GetStorageFromCtx(ctx); s != nil {
		return s
	}
	return db.Stor
}

func handleAdd(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	key, err := request.RequireString("key")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	value, err := request.RequireString("value")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	stor := storageFromCtx(ctx)
	vjk := models.ValJsonKey{Key: key, Type: models.ORIGIN}

	resources, err := stor.GetAllResources()
	if err != nil {
		return mcp.NewToolResultErrorFromErr(i18n.T("mcp.failed_to_get"), err), nil
	}
	if _, exists := resources[vjk]; exists {
		return mcp.NewToolResultError(i18n.T("mcp.key_already_exists") + key), nil
	}

	value = util.UnescapeString(value)
	newResource := models.ValJson{Val: value, Tag: []string{}}

	if err := stor.SaveResource(vjk, newResource); err != nil {
		return mcp.NewToolResultErrorFromErr(i18n.T("mcp.failed_to_save"), err), nil
	}
	_ = stor.SaveAuditRecord(models.AuditRecord{
		ResourceKey: key,
		Operation:   "add",
		Timestamp:   time.Now().Unix(),
		Count:       1,
	})

	return mcp.NewToolResultText(i18n.T("mcp.added_successfully")), nil
}

func handleGet(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	keyword := request.GetString("key", "")

	stor := storageFromCtx(ctx)
	resources, err := stor.GetAllResources()
	if err != nil {
		return mcp.NewToolResultErrorFromErr(i18n.T("mcp.failed_to_get"), err), nil
	}

	if keyword == "" {
		var keys []string
		for k := range resources {
			if k.Type == models.ORIGIN {
				keys = append(keys, k.Key)
			}
		}
		if len(keys) == 0 {
			return mcp.NewToolResultText(i18n.T("mcp.no_resources")), nil
		}
		return mcp.NewToolResultText(i18n.T("mcp.all_resources") + strings.Join(keys, "\n")), nil
	}

	type matchItem struct {
		Key      string
		Value    string
		Tags     []string
		MatchVia string
	}
	var matches []matchItem

	for k, v := range resources {
		if util.ContainsIgnoreCase(k.Key, keyword) {
			displayKey := k.Key
			if k.Type == models.TAG {
				displayKey = k.OriginKey
			}
			matches = append(matches, matchItem{Key: displayKey, Value: v.Val, Tags: v.Tag, MatchVia: "key"})
			continue
		}
		for _, tag := range v.Tag {
			if util.ContainsIgnoreCase(tag, keyword) {
				displayKey := k.Key
				if k.Type == models.TAG {
					displayKey = k.OriginKey
				}
				matches = append(matches, matchItem{Key: displayKey, Value: v.Val, Tags: v.Tag, MatchVia: "tag:" + tag})
				break
			}
		}
	}

	if len(matches) == 0 {
		return mcp.NewToolResultError(i18n.T("mcp.not_found") + keyword), nil
	}

	_ = stor.SaveAuditRecord(models.AuditRecord{
		ResourceKey: matches[0].Key,
		Operation:   "get",
		Timestamp:   time.Now().Unix(),
		Count:       1,
	})

	var sb strings.Builder
	for i, m := range matches {
		if i > 0 {
			sb.WriteString("\n---\n")
		}
		sb.WriteString(i18n.T("mcp.resource_label") + m.Key + "\n")
		sb.WriteString(i18n.T("mcp.tags_label") + strings.Join(m.Tags, ", "))
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func handleUpdate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	key, err := request.RequireString("key")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	value, err := request.RequireString("value")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	stor := storageFromCtx(ctx)
	vjk := models.ValJsonKey{Key: key, Type: models.ORIGIN}

	resources, err := stor.GetAllResources()
	if err != nil {
		return mcp.NewToolResultErrorFromErr(i18n.T("mcp.failed_to_get"), err), nil
	}
	existing, exists := resources[vjk]
	if !exists {
		return mcp.NewToolResultError(i18n.T("mcp.resource_not_found") + key), nil
	}

	value = util.UnescapeString(value)
	if err := stor.UpdateResource(vjk, models.ValJson{Val: value, Tag: existing.Tag}); err != nil {
		return mcp.NewToolResultErrorFromErr(i18n.T("mcp.failed_to_update"), err), nil
	}

	_ = stor.SaveAuditRecord(models.AuditRecord{
		ResourceKey: key,
		Operation:   "update",
		Timestamp:   time.Now().Unix(),
		Count:       1,
	})

	return mcp.NewToolResultText(i18n.T("mcp.added_successfully")), nil
}

func handleDelete(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	key, err := request.RequireString("key")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	stor := storageFromCtx(ctx)
	vjk := models.ValJsonKey{Key: key, Type: models.ORIGIN}

	resources, err := stor.GetAllResources()
	if err != nil {
		return mcp.NewToolResultErrorFromErr(i18n.T("mcp.failed_to_get"), err), nil
	}
	if _, exists := resources[vjk]; !exists {
		return mcp.NewToolResultError(i18n.T("mcp.resource_not_found") + key), nil
	}

	_ = stor.DeleteHistoryRecords(key)
	_ = stor.DeleteAuditRecords(key)

	if err := stor.DeleteResource(vjk); err != nil {
		return mcp.NewToolResultErrorFromErr(i18n.T("mcp.failed_to_save"), err), nil
	}

	return mcp.NewToolResultText(i18n.T("mcp.deleted_successfully")), nil
}

func handleTag(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	key, err := request.RequireString("key")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	tags := request.GetStringSlice("tags", nil)
	if len(tags) == 0 {
		return mcp.NewToolResultError(i18n.T("mcp.at_least_one_tag_required")), nil
	}

	stor := storageFromCtx(ctx)
	vjk := models.ValJsonKey{Key: key, Type: models.ORIGIN}

	resources, err := stor.GetAllResources()
	if err != nil {
		return mcp.NewToolResultErrorFromErr(i18n.T("mcp.failed_to_get"), err), nil
	}
	resource, exists := resources[vjk]
	if !exists {
		return mcp.NewToolResultError(i18n.T("mcp.resource_not_found")), nil
	}

	newTags := append(resource.Tag, tags...)
	resource.Tag = util.RemoveDuplicates(newTags)

	if err := stor.SaveResource(vjk, resource); err != nil {
		return mcp.NewToolResultErrorFromErr(i18n.T("mcp.failed_to_save"), err), nil
	}

	return mcp.NewToolResultText(i18n.T("mcp.tag_added_successfully")), nil
}

func handleDtag(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	key, err := request.RequireString("key")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	tag, err := request.RequireString("tag")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	stor := storageFromCtx(ctx)
	vjk := models.ValJsonKey{Key: key, Type: models.ORIGIN}

	resources, err := stor.GetAllResources()
	if err != nil {
		return mcp.NewToolResultErrorFromErr(i18n.T("mcp.failed_to_get"), err), nil
	}
	resource, exists := resources[vjk]
	if !exists {
		return mcp.NewToolResultError(i18n.T("mcp.resource_not_found")), nil
	}

	newTags := make([]string, 0, len(resource.Tag))
	for _, t := range resource.Tag {
		if t != tag {
			newTags = append(newTags, t)
		}
	}
	resource.Tag = newTags

	if err := stor.SaveResource(vjk, resource); err != nil {
		return mcp.NewToolResultErrorFromErr(i18n.T("mcp.failed_to_save"), err), nil
	}

	return mcp.NewToolResultText(i18n.T("mcp.tag_deleted_successfully")), nil
}

func handleRename(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	oldKey, err := request.RequireString("old_key")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	newKey, err := request.RequireString("new_key")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	stor := storageFromCtx(ctx)
	oldVjk := models.ValJsonKey{Key: oldKey, Type: models.ORIGIN}

	resources, err := stor.GetAllResources()
	if err != nil {
		return mcp.NewToolResultErrorFromErr(i18n.T("mcp.failed_to_get"), err), nil
	}
	resource, exists := resources[oldVjk]
	if !exists {
		return mcp.NewToolResultError(i18n.T("mcp.resource_not_found")), nil
	}

	if err := stor.DeleteResource(oldVjk); err != nil {
		return mcp.NewToolResultErrorFromErr(i18n.T("mcp.failed_to_delete_old"), err), nil
	}

	newVjk := models.ValJsonKey{Key: newKey, Type: models.ORIGIN}
	if err := stor.SaveResource(newVjk, resource); err != nil {
		return mcp.NewToolResultErrorFromErr(i18n.T("mcp.failed_to_save_new"), err), nil
	}

	return mcp.NewToolResultText(i18n.T("mcp.rename_successfully")), nil
}

func handleList(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	stor := storageFromCtx(ctx)
	resources, err := stor.GetAllResources()
	if err != nil {
		return mcp.NewToolResultErrorFromErr(i18n.T("mcp.failed_to_get"), err), nil
	}

	var keys []string
	for k := range resources {
		if k.Type == models.ORIGIN {
			keys = append(keys, k.Key)
		}
	}

	if len(keys) == 0 {
		return mcp.NewToolResultText(i18n.T("mcp.no_resources")), nil
	}

	return mcp.NewToolResultText(strings.Join(keys, "\n")), nil
}

func handleLogAdd(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	content, err := request.RequireString("content")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	tags := request.GetStringSlice("tags", nil)

	stor := storageFromCtx(ctx)
	now := time.Now()
	record := models.LogRecord{
		ID:        now.UnixNano(),
		Content:   content,
		Tags:      util.RemoveDuplicates(tags),
		CreatedAt: now.Format("2006-01-02 15:04:05"),
		Date:      now.Format("2006-01-02"),
	}

	if err := stor.SaveLogRecord(record); err != nil {
		return mcp.NewToolResultErrorFromErr(i18n.T("mcp.failed_to_save_log"), err), nil
	}

	tagHint := ""
	if len(record.Tags) > 0 {
		tagHint = " [" + strings.Join(record.Tags, ", ") + "]"
	}
	return mcp.NewToolResultText(fmt.Sprintf(i18n.T("mcp.log_recorded"), content, tagHint, record.CreatedAt)), nil
}

func handleLogList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	fromDate := request.GetString("from", "")
	toDate := request.GetString("to", "")
	week := request.GetBool("week", false)
	month := request.GetBool("month", false)
	dateFlag := request.GetString("date", "")
	tagFilter := request.GetString("tag", "")

	stor := storageFromCtx(ctx)
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

	records, err := stor.GetLogRecords(startDate, endDate)
	if err != nil {
		return mcp.NewToolResultErrorFromErr(i18n.T("mcp.failed_to_get_log"), err), nil
	}

	if tagFilter != "" {
		var filtered []models.LogRecord
		for _, r := range records {
			for _, t := range r.Tags {
				if t == tagFilter {
					filtered = append(filtered, r)
					break
				}
			}
		}
		records = filtered
	}

	if len(records) == 0 {
		return mcp.NewToolResultText(i18n.T("mcp.no_log_records")), nil
	}

	var sb strings.Builder
	var currentDate string
	for _, r := range records {
		if r.Date != currentDate {
			if currentDate != "" {
				sb.WriteString("\n")
			}
			sb.WriteString("📅 " + r.Date + "\n")
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

		sb.WriteString("  [" + timeStr + "] " + r.Content + tagStr + "\n")
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func handleLogDelete(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := request.GetInt("id", 0)
	if id == 0 {
		return mcp.NewToolResultError(i18n.T("mcp.id_required")), nil
	}

	stor := storageFromCtx(ctx)
	if err := stor.DeleteLogRecord(int64(id)); err != nil {
		return mcp.NewToolResultErrorFromErr(i18n.T("mcp.failed_to_delete_log"), err), nil
	}

	return mcp.NewToolResultText(i18n.T("mcp.log_deleted_successfully")), nil
}

func handleExport(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	exportType := request.GetString("type", "resources")
	withBOM := request.GetBool("bom", false)

	if exportType != "resources" && exportType != "audit" && exportType != "history" && exportType != "log" {
		return mcp.NewToolResultError(i18n.T("mcp.unsupported_export_type") + exportType + i18n.T("mcp.export_unsupported_suffix")), nil
	}

	stor := storageFromCtx(ctx)
	var sb strings.Builder

	if withBOM {
		sb.WriteString("\xEF\xBB\xBF")
	}

	var header, rows []string

	switch exportType {
	case "resources":
		header = []string{"key", "value", "tags"}
		resources, err := stor.GetAllResources()
		if err != nil {
			return mcp.NewToolResultErrorFromErr(i18n.T("mcp.failed_to_get"), err), nil
		}
		for k, v := range resources {
			if k.Type == models.ORIGIN {
				tags := strings.Join(v.Tag, "|")
				rows = append(rows, k.Key+","+v.Val+","+tags)
			}
		}
	case "audit":
		header = []string{"resource_key", "operation", "timestamp", "count"}
		records, err := stor.GetAllAuditRecords()
		if err != nil {
			return mcp.NewToolResultErrorFromErr(i18n.T("mcp.failed_to_get_audit"), err), nil
		}
		for _, r := range records {
			rows = append(rows, r.ResourceKey+","+r.Operation+","+fmt.Sprintf("%d", r.Timestamp)+","+fmt.Sprintf("%d", r.Count))
		}
	case "history":
		header = []string{"id", "resource_key", "operation", "time", "command"}
		records, err := stor.GetAllHistoryRecords()
		if err != nil {
			return mcp.NewToolResultErrorFromErr(i18n.T("mcp.failed_to_get_history"), err), nil
		}
		for _, r := range records {
			rows = append(rows, fmt.Sprintf("%d", r.ID)+","+r.ResourceKey+","+r.Operation+","+r.TimeStr+","+r.Command)
		}
	case "log":
		header = []string{"id", "content", "tags", "created_at", "date"}
		now := time.Now()
		records, err := stor.GetLogRecords("1970-01-01", now.Format("2006-01-02"))
		if err != nil {
			return mcp.NewToolResultErrorFromErr(i18n.T("mcp.failed_to_get_log"), err), nil
		}
		for _, r := range records {
			tags := strings.Join(r.Tags, "|")
			rows = append(rows, fmt.Sprintf("%d", r.ID)+","+r.Content+","+tags+","+r.CreatedAt+","+r.Date)
		}
	}

	sb.WriteString(strings.Join(header, ",") + "\n")
	for _, row := range rows {
		sb.WriteString(row + "\n")
	}

	result := sb.String()
	count := len(rows)
	if count == 0 {
		return mcp.NewToolResultText(result), nil
	}
	return mcp.NewToolResultText(result + "\n" + i18n.T("mcp.total_records") + fmt.Sprintf("%d", count) + " records"), nil
}
