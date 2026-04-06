package ai

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"ttl-cli/models"
	"ttl-cli/util"
)

const MaxObservationLen = 2000

type Step struct {
	Thought string            `json:"thought"`
	Action  string            `json:"action"`
	Params  map[string]string `json:"params"`
}

type ChatFunc func(messages []Message) (string, error)

type ResourceStore interface {
	GetAllResources() (map[models.ValJsonKey]models.ValJson, error)
	SaveResource(key models.ValJsonKey, value models.ValJson) error
	DeleteResource(key models.ValJsonKey) error
	UpdateResource(key models.ValJsonKey, newValue models.ValJson) error
	GetAuditStats() (models.AuditStats, error)
	GetAllHistoryRecords() ([]models.HistoryRecord, error)
	SaveLogRecord(record models.LogRecord) error
	GetLogRecords(startDate, endDate string) ([]models.LogRecord, error)
	DeleteLogRecord(id int64) error
}

type OpenFunc func(url string) error

type Agent struct {
	ChatFn        ChatFunc
	Store         ResourceStore
	OpenFn        OpenFunc
	MaxSteps      int
	ContextLoader *ContextLoader
}

func NewAgent(chatFn ChatFunc, store ResourceStore) *Agent {
	return &Agent{ChatFn: chatFn, Store: store, MaxSteps: 5}
}

func (a *Agent) WithContext(loader *ContextLoader) *Agent {
	a.ContextLoader = loader
	return a
}

func (a *Agent) Run(input string) (string, error) {
	messages := []Message{
		{Role: "system", Content: ReActSystemPrompt()},
	}

	if a.ContextLoader != nil {
		history, err := a.ContextLoader.LoadMessages()
		if err == nil && len(history) > 0 {
			for _, msg := range history {
				messages = append(messages, Message{
					Role:    msg.Role,
					Content: msg.Content,
				})
			}
		}
		_ = a.ContextLoader.SaveUserMessage(input)
	}

	messages = append(messages, Message{Role: "user", Content: input})

	var lastObservation string
	var finalAnswer string

	for i := 0; i < a.MaxSteps; i++ {
		resp, err := a.ChatFn(messages)
		if err != nil {
			return "", fmt.Errorf("AI 调用失败: %w", err)
		}

		step, err := parseStep(resp)
		if err != nil {
			finalAnswer = resp
			break
		}

		if step.Action == "answer" {
			msg := step.Params["message"]
			if msg == "" {
				msg = step.Thought
			}
			finalAnswer = msg
			break
		}

		messages = append(messages, Message{Role: "assistant", Content: resp})

		observation, execErr := a.execute(step)
		if execErr != nil {
			observation = fmt.Sprintf("执行失败: %s", execErr.Error())
		}

		observation = truncateObservation(observation, MaxObservationLen)
		lastObservation = observation
		messages = append(messages, Message{
			Role:    "user",
			Content: fmt.Sprintf("[Observation] %s", observation),
		})
	}

	if a.ContextLoader != nil && finalAnswer != "" {
		_ = a.ContextLoader.SaveAssistantMessage(finalAnswer)
	}

	if finalAnswer != "" {
		return finalAnswer, nil
	}
	if lastObservation != "" {
		return lastObservation, nil
	}
	return "Operation completed (max steps reached)", nil
}

func parseStep(resp string) (*Step, error) {
	cleaned := cleanJSONResponse(resp)

	var step Step
	if err := json.Unmarshal([]byte(cleaned), &step); err != nil {
		return nil, fmt.Errorf("JSON 解析失败: %w, 原始内容: %q", err, resp)
	}

	if step.Action == "" {
		return nil, fmt.Errorf("missing action field: %q", cleaned)
	}

	return &step, nil
}

func cleanJSONResponse(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	}

	if strings.Contains(s, "\n") {
		lines := strings.Split(s, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "{") && strings.HasSuffix(line, "}") {
				return line
			}
		}
	}

	return s
}

func truncateObservation(s string, max int) string {
	if len(s) <= max {
		return s
	}
	lines := strings.Split(s[:max], "\n")
	if len(lines) > 1 {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n") + "\n...(truncated)"
}

func (a *Agent) execute(step *Step) (string, error) {
	switch step.Action {
	case "add":
		return a.execAdd(step.Params)
	case "get":
		return a.execGet(step.Params)
	case "open":
		return a.execOpen(step.Params)
	case "update":
		return a.execUpdate(step.Params)
	case "delete":
		return a.execDelete(step.Params)
	case "tag":
		return a.execTag(step.Params)
	case "dtag":
		return a.execDtag(step.Params)
	case "rename":
		return a.execRename(step.Params)
	case "stats":
		return a.execStats()
	case "history":
		return a.execHistory(step.Params)
	case "log_write":
		return a.execLogWrite(step.Params)
	case "log_list":
		return a.execLogList(step.Params)
	case "log_delete":
		return a.execLogDelete(step.Params)
	case "export":
		return a.execExport(step.Params)
	default:
		return "未知操作: " + step.Action, nil
	}
}

func (a *Agent) execAdd(params map[string]string) (string, error) {
	key := params["key"]
	value := params["value"]
	if key == "" || value == "" {
		return "添加资源需要提供 key 和 value", nil
	}

	resources, err := a.Store.GetAllResources()
	if err != nil {
		return "", fmt.Errorf("获取资源失败: %w", err)
	}

	k := models.ValJsonKey{Key: key, Type: models.ORIGIN}
	if _, exists := resources[k]; exists {
		return fmt.Sprintf("资源 %s 已存在，如需更新请说「更新 %s」", key, key), nil
	}

	v := models.ValJson{Val: util.UnescapeString(value), Tag: []string{}}
	if err := a.Store.SaveResource(k, v); err != nil {
		return "", fmt.Errorf("保存资源失败: %w", err)
	}

	return fmt.Sprintf("已保存：%s → %s", key, value), nil
}

func (a *Agent) execGet(params map[string]string) (string, error) {
	keyword := params["keyword"]

	resources, err := a.Store.GetAllResources()
	if err != nil {
		return "", fmt.Errorf("获取资源失败: %w", err)
	}

	if keyword == "" {
		type keyTag struct {
			key    string
			tagStr string
		}
		var items []keyTag
		for k, v := range resources {
			if k.Type == models.ORIGIN {
				tagStr := ""
				if len(v.Tag) > 0 {
					tagStr = fmt.Sprintf(" [标签: %s]", strings.Join(v.Tag, ", "))
				}
				items = append(items, keyTag{key: k.Key, tagStr: tagStr})
			}
		}
		sort.Slice(items, func(i, j int) bool { return items[i].key < items[j].key })
		if len(items) == 0 {
			return "当前没有存储任何资源", nil
		}
		var lines []string
		for _, item := range items {
			lines = append(lines, fmt.Sprintf("  %s%s", item.key, item.tagStr))
		}
		return fmt.Sprintf("共 %d 条资源：\n%s", len(items), strings.Join(lines, "\n")), nil
	}

	var matches []string
	for k, v := range resources {
		if k.Type != models.ORIGIN {
			continue
		}
		if util.ContainsIgnoreCase(k.Key, keyword) || matchTags(v.Tag, keyword) {
			tagStr := ""
			if len(v.Tag) > 0 {
				tagStr = fmt.Sprintf(" [标签: %s]", strings.Join(v.Tag, ", "))
			}
			matches = append(matches, fmt.Sprintf("  %s%s", k.Key, tagStr))
		}
	}

	if len(matches) == 0 {
		return fmt.Sprintf("未找到与 \"%s\" 相关的资源", keyword), nil
	}
	sort.Strings(matches)
	return fmt.Sprintf("找到 %d 条匹配：\n%s", len(matches), strings.Join(matches, "\n")), nil
}

func (a *Agent) execOpen(params map[string]string) (string, error) {
	keyword := params["keyword"]
	if keyword == "" {
		return "打开资源需要提供关键词", nil
	}

	resources, err := a.Store.GetAllResources()
	if err != nil {
		return "", fmt.Errorf("获取资源失败: %w", err)
	}

	type match struct {
		key string
		val string
		tag []string
	}
	var matches []match
	for k, v := range resources {
		if k.Type != models.ORIGIN {
			continue
		}
		if util.ContainsIgnoreCase(k.Key, keyword) || matchTags(v.Tag, keyword) {
			matches = append(matches, match{key: k.Key, val: v.Val, tag: v.Tag})
		}
	}

	if len(matches) == 0 {
		return fmt.Sprintf("未找到与 \"%s\" 相关的资源", keyword), nil
	}

	if len(matches) > 1 {
		var lines []string
		for _, m := range matches {
			tagStr := ""
			if len(m.tag) > 0 {
				tagStr = fmt.Sprintf(" [标签: %s]", strings.Join(m.tag, ", "))
			}
			lines = append(lines, fmt.Sprintf("  %s%s", m.key, tagStr))
		}
		return fmt.Sprintf("找到 %d 条匹配，请缩小范围：\n%s", len(matches), strings.Join(lines, "\n")), nil
	}

	target := matches[0]
	if a.OpenFn != nil {
		if err := a.OpenFn(target.val); err != nil {
			return "", fmt.Errorf("打开失败: %w", err)
		}
		return fmt.Sprintf("已打开：%s", target.key), nil
	}

	return fmt.Sprintf("请手动打开：%s", target.val), nil
}

func (a *Agent) execUpdate(params map[string]string) (string, error) {
	key := params["key"]
	value := params["value"]
	if key == "" || value == "" {
		return "更新资源需要提供 key 和 value", nil
	}

	resources, err := a.Store.GetAllResources()
	if err != nil {
		return "", fmt.Errorf("获取资源失败: %w", err)
	}

	k := models.ValJsonKey{Key: key, Type: models.ORIGIN}
	existing, exists := resources[k]
	if !exists {
		return fmt.Sprintf("未找到资源：%s", key), nil
	}

	newVal := models.ValJson{Val: util.UnescapeString(value), Tag: existing.Tag}
	if err := a.Store.UpdateResource(k, newVal); err != nil {
		return "", fmt.Errorf("更新资源失败: %w", err)
	}

	return fmt.Sprintf("已更新：%s → %s", key, value), nil
}

func (a *Agent) execDelete(params map[string]string) (string, error) {
	key := params["key"]
	if key == "" {
		return "删除资源需要提供 key", nil
	}

	confirm := params["confirm"]
	if confirm != "true" {
		resources, err := a.Store.GetAllResources()
		if err != nil {
			return "", fmt.Errorf("获取资源失败: %w", err)
		}
		k := models.ValJsonKey{Key: key, Type: models.ORIGIN}
		_, exists := resources[k]
		if !exists {
			return fmt.Sprintf("未找到资源：%s", key), nil
		}
		return fmt.Sprintf("即将删除：%s\n请使用 ttl del %s 确认删除", key, key), nil
	}

	k := models.ValJsonKey{Key: key, Type: models.ORIGIN}
	if err := a.Store.DeleteResource(k); err != nil {
		return "", fmt.Errorf("删除资源失败: %w", err)
	}

	return fmt.Sprintf("已删除：%s", key), nil
}

func (a *Agent) execTag(params map[string]string) (string, error) {
	key := params["key"]
	tagsStr := params["tags"]
	if key == "" || tagsStr == "" {
		return "打标签需要提供 key 和 tags", nil
	}

	resources, err := a.Store.GetAllResources()
	if err != nil {
		return "", fmt.Errorf("获取资源失败: %w", err)
	}

	k := models.ValJsonKey{Key: key, Type: models.ORIGIN}
	resource, exists := resources[k]
	if !exists {
		return fmt.Sprintf("未找到资源：%s", key), nil
	}

	newTags := strings.Split(tagsStr, ",")
	for i := range newTags {
		newTags[i] = strings.TrimSpace(newTags[i])
	}
	resource.Tag = util.RemoveDuplicates(append(resource.Tag, newTags...))

	if err := a.Store.SaveResource(k, resource); err != nil {
		return "", fmt.Errorf("保存资源失败: %w", err)
	}

	return fmt.Sprintf("已为 %s 添加标签：%s", key, strings.Join(newTags, ", ")), nil
}

func (a *Agent) execDtag(params map[string]string) (string, error) {
	key := params["key"]
	tag := params["tag"]
	if key == "" || tag == "" {
		return "删除标签需要提供 key 和 tag", nil
	}

	resources, err := a.Store.GetAllResources()
	if err != nil {
		return "", fmt.Errorf("获取资源失败: %w", err)
	}

	k := models.ValJsonKey{Key: key, Type: models.ORIGIN}
	resource, exists := resources[k]
	if !exists {
		return fmt.Sprintf("未找到资源：%s", key), nil
	}

	newTags := make([]string, 0, len(resource.Tag))
	for _, t := range resource.Tag {
		if t != tag {
			newTags = append(newTags, t)
		}
	}
	resource.Tag = newTags

	if err := a.Store.SaveResource(k, resource); err != nil {
		return "", fmt.Errorf("保存资源失败: %w", err)
	}

	return fmt.Sprintf("已从 %s 删除标签：%s", key, tag), nil
}

func (a *Agent) execRename(params map[string]string) (string, error) {
	oldKey := params["old_key"]
	newKey := params["new_key"]
	if oldKey == "" || newKey == "" {
		return "重命名需要提供 old_key 和 new_key", nil
	}

	resources, err := a.Store.GetAllResources()
	if err != nil {
		return "", fmt.Errorf("获取资源失败: %w", err)
	}

	ok := models.ValJsonKey{Key: oldKey, Type: models.ORIGIN}
	resource, exists := resources[ok]
	if !exists {
		return fmt.Sprintf("未找到资源：%s", oldKey), nil
	}

	if err := a.Store.DeleteResource(ok); err != nil {
		return "", fmt.Errorf("删除旧资源失败: %w", err)
	}

	nk := models.ValJsonKey{Key: newKey, Type: models.ORIGIN}
	if err := a.Store.SaveResource(nk, resource); err != nil {
		return "", fmt.Errorf("保存新资源失败: %w", err)
	}

	return fmt.Sprintf("已重命名：%s → %s", oldKey, newKey), nil
}

func (a *Agent) execStats() (string, error) {
	stats, err := a.Store.GetAuditStats()
	if err != nil {
		return "", fmt.Errorf("获取审计统计失败: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("总操作次数：%d\n", stats.TotalOperations))

	if len(stats.ByOperation) > 0 {
		sb.WriteString("\n按操作类型：\n")
		for op, count := range stats.ByOperation {
			sb.WriteString(fmt.Sprintf("  %s: %d\n", op, count))
		}
	}

	if len(stats.ByResource) > 0 {
		sb.WriteString("\n按资源（前10）：\n")
		type kv struct {
			k string
			v int
		}
		var sorted []kv
		for k, v := range stats.ByResource {
			sorted = append(sorted, kv{k, v})
		}
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].v > sorted[j].v })
		for i, item := range sorted {
			if i >= 10 {
				break
			}
			sb.WriteString(fmt.Sprintf("  %s: %d\n", item.k, item.v))
		}
	}

	return sb.String(), nil
}

func (a *Agent) execHistory(params map[string]string) (string, error) {
	limit := 20
	if l, ok := params["limit"]; ok && l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	records, err := a.Store.GetAllHistoryRecords()
	if err != nil {
		return "", fmt.Errorf("获取历史记录失败: %w", err)
	}

	if len(records) == 0 {
		return "暂无操作历史", nil
	}

	if len(records) > limit {
		records = records[:limit]
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("最近 %d 条操作历史：\n", len(records)))
	for i, r := range records {
		display := r.ResourceKey
		if display == "" {
			display = r.Command
		}
		sb.WriteString(fmt.Sprintf("  %d. [%s] %s %s\n", i+1, r.TimeStr, r.Operation, display))
	}

	return sb.String(), nil
}

func (a *Agent) execLogWrite(params map[string]string) (string, error) {
	content := params["content"]
	if content == "" {
		return "记录日志需要提供 content（日志正文）", nil
	}

	now := time.Now()
	record := models.LogRecord{
		ID:        now.UnixNano(),
		Content:   content,
		Tags:      []string{},
		CreatedAt: now.Format("2006-01-02 15:04:05"),
		Date:      now.Format("2006-01-02"),
	}

	if tagsStr := params["tags"]; tagsStr != "" {
		tags := strings.Split(tagsStr, ",")
		for i := range tags {
			tags[i] = strings.TrimSpace(tags[i])
		}
		record.Tags = tags
	}

	if err := a.Store.SaveLogRecord(record); err != nil {
		return "", fmt.Errorf("保存日志失败: %w", err)
	}

	tagHint := ""
	if len(record.Tags) > 0 {
		tagHint = fmt.Sprintf(" [%s]", strings.Join(record.Tags, ", "))
	}
	return fmt.Sprintf("日志已记录：%s%s（%s）", content, tagHint, record.CreatedAt), nil
}

func (a *Agent) execLogList(params map[string]string) (string, error) {
	now := time.Now()
	startDate := params["start_date"]
	endDate := params["end_date"]
	filterTag := params["tag"]

	if startDate == "" {
		startDate = now.Format("2006-01-02")
	}
	if endDate == "" {
		endDate = now.Format("2006-01-02")
	}

	if r := params["range"]; r != "" {
		switch r {
		case "week":
			weekday := now.Weekday()
			if weekday == time.Sunday {
				weekday = 7
			}
			monday := now.AddDate(0, 0, -int(weekday-time.Monday))
			startDate = monday.Format("2006-01-02")
			endDate = now.Format("2006-01-02")
		case "month":
			startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
			endDate = now.Format("2006-01-02")
		}
	}

	records, err := a.Store.GetLogRecords(startDate, endDate)
	if err != nil {
		return "", fmt.Errorf("获取日志失败: %w", err)
	}

	if filterTag != "" {
		var filtered []models.LogRecord
		for _, r := range records {
			for _, t := range r.Tags {
				if util.ContainsIgnoreCase(t, filterTag) {
					filtered = append(filtered, r)
					break
				}
			}
		}
		records = filtered
	}

	if len(records) == 0 {
		return fmt.Sprintf("在 %s ~ %s 期间暂无日志记录", startDate, endDate), nil
	}

	var sb strings.Builder
	currentDate := ""
	for _, r := range records {
		if r.Date != currentDate {
			if currentDate != "" {
				sb.WriteString("\n")
			}
			sb.WriteString(fmt.Sprintf("📅 %s\n", r.Date))
			currentDate = r.Date
		}
		timeStr := r.CreatedAt
		if len(timeStr) > 10 {
			timeStr = timeStr[11:]
		}
		tagStr := ""
		if len(r.Tags) > 0 {
			tagStr = fmt.Sprintf(" [%s]", strings.Join(r.Tags, ", "))
		}
		sb.WriteString(fmt.Sprintf("  [%s] %s%s\n", timeStr, r.Content, tagStr))
	}

	return sb.String(), nil
}

func (a *Agent) execLogDelete(params map[string]string) (string, error) {
	idStr := params["id"]
	if idStr == "" {
		return "删除日志需要提供 id（日志 ID）", nil
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return "日志 ID 格式不正确，应为数字", nil
	}

	if err := a.Store.DeleteLogRecord(id); err != nil {
		return "", fmt.Errorf("删除日志失败: %w", err)
	}

	return "日志删除成功", nil
}

func (a *Agent) execExport(params map[string]string) (string, error) {
	exportType := params["type"]
	if exportType == "" {
		exportType = "resources"
	}

	switch exportType {
	case "resources":
		resources, err := a.Store.GetAllResources()
		if err != nil {
			return "", fmt.Errorf("获取资源失败: %w", err)
		}
		count := 0
		for k := range resources {
			if k.Type == models.ORIGIN {
				count++
			}
		}
		return fmt.Sprintf("共有 %d 条资源可导出。请使用 `ttl export --type resources` 导出为 CSV 文件", count), nil
	case "audit":
		stats, err := a.Store.GetAuditStats()
		if err != nil {
			return "", fmt.Errorf("获取审计数据失败: %w", err)
		}
		return fmt.Sprintf("共有 %d 条审计记录可导出。请使用 `ttl export --type audit` 导出为 CSV 文件", stats.TotalOperations), nil
	case "history":
		records, err := a.Store.GetAllHistoryRecords()
		if err != nil {
			return "", fmt.Errorf("获取历史数据失败: %w", err)
		}
		return fmt.Sprintf("共有 %d 条历史记录可导出。请使用 `ttl export --type history` 导出为 CSV 文件", len(records)), nil
	case "log":
		now := time.Now()
		records, err := a.Store.GetLogRecords("1970-01-01", now.Format("2006-01-02"))
		if err != nil {
			return "", fmt.Errorf("获取日志数据失败: %w", err)
		}
		return fmt.Sprintf("共有 %d 条工作日志可导出。请使用 `ttl export --type log` 导出为 CSV 文件", len(records)), nil
	default:
		return fmt.Sprintf("不支持的导出类型：%s。支持的类型：resources, audit, history, log", exportType), nil
	}
}

func matchTags(tags []string, keyword string) bool {
	for _, t := range tags {
		if util.ContainsIgnoreCase(t, keyword) {
			return true
		}
	}
	return false
}
