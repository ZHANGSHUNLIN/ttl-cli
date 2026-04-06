package ai

import (
	"fmt"
	"strings"
	"testing"
	"ttl-cli/models"
)

// mockStore 模拟存储
type mockStore struct {
	resources  map[models.ValJsonKey]models.ValJson
	auditStats models.AuditStats
	history    []models.HistoryRecord
	logs       []models.LogRecord
}

func newMockStore() *mockStore {
	return &mockStore{
		resources: make(map[models.ValJsonKey]models.ValJson),
		auditStats: models.AuditStats{
			ByOperation: make(map[string]int),
			ByResource:  make(map[string]int),
		},
	}
}

func (m *mockStore) GetAllResources() (map[models.ValJsonKey]models.ValJson, error) {
	return m.resources, nil
}
func (m *mockStore) SaveResource(key models.ValJsonKey, value models.ValJson) error {
	m.resources[key] = value
	return nil
}
func (m *mockStore) DeleteResource(key models.ValJsonKey) error {
	delete(m.resources, key)
	return nil
}
func (m *mockStore) UpdateResource(key models.ValJsonKey, newValue models.ValJson) error {
	m.resources[key] = newValue
	return nil
}
func (m *mockStore) GetAuditStats() (models.AuditStats, error) {
	return m.auditStats, nil
}
func (m *mockStore) GetAllHistoryRecords() ([]models.HistoryRecord, error) {
	return m.history, nil
}
func (m *mockStore) SaveLogRecord(record models.LogRecord) error {
	m.logs = append(m.logs, record)
	return nil
}
func (m *mockStore) GetLogRecords(startDate, endDate string) ([]models.LogRecord, error) {
	var result []models.LogRecord
	for _, r := range m.logs {
		if r.Date >= startDate && r.Date <= endDate {
			result = append(result, r)
		}
	}
	return result, nil
}
func (m *mockStore) DeleteLogRecord(id int64) error {
	for i, r := range m.logs {
		if r.ID == id {
			m.logs = append(m.logs[:i], m.logs[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("log record not found")
}

// mockReActChat 基于 len(messages) 路由的 mock ChatFunc（适配 ReAct 多轮）
// round1Resp 在 len(messages)==2 时返回（首轮：system + user）
// round2Resp 在 len(messages)==4 时返回（第二轮：+assistant +observation）
// 以此类推
func mockReActChat(responses ...string) ChatFunc {
	return func(messages []Message) (string, error) {
		// len(messages) == 2 → round 1, == 4 → round 2, == 6 → round 3, ...
		round := (len(messages) - 2) / 2 // 0-based
		if round < 0 {
			round = 0
		}
		if round < len(responses) {
			return responses[round], nil
		}
		// 兜底：返回 answer
		return `{"thought":"done","action":"answer","params":{"message":"完成"}}`, nil
	}
}

// --- parseStep 测试 ---

func TestParseStep_ValidJSON(t *testing.T) {
	step, err := parseStep(`{"thought":"查docker","action":"get","params":{"keyword":"docker"}}`)
	if err != nil {
		t.Fatalf("parseStep error: %v", err)
	}
	if step.Action != "get" {
		t.Errorf("action = %s, want get", step.Action)
	}
	if step.Thought != "查docker" {
		t.Errorf("thought = %s, want 查docker", step.Thought)
	}
	if step.Params["keyword"] != "docker" {
		t.Errorf("keyword = %s, want docker", step.Params["keyword"])
	}
}

func TestParseStep_MarkdownWrapped(t *testing.T) {
	step, err := parseStep("```json\n{\"thought\":\"hi\",\"action\":\"answer\",\"params\":{\"message\":\"你好\"}}\n```")
	if err != nil {
		t.Fatalf("parseStep error: %v", err)
	}
	if step.Action != "answer" {
		t.Errorf("action = %s, want answer", step.Action)
	}
}

func TestParseStep_InvalidJSON(t *testing.T) {
	_, err := parseStep("这不是JSON")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseStep_MissingAction(t *testing.T) {
	_, err := parseStep(`{"thought":"hmm","params":{}}`)
	if err == nil {
		t.Fatal("expected error for missing action")
	}
}

func TestParseStep_AnswerAction(t *testing.T) {
	step, err := parseStep(`{"thought":"可以回答了","action":"answer","params":{"message":"你好，我是助手"}}`)
	if err != nil {
		t.Fatalf("parseStep error: %v", err)
	}
	if step.Action != "answer" {
		t.Errorf("action = %s, want answer", step.Action)
	}
	if step.Params["message"] != "你好，我是助手" {
		t.Errorf("message = %s, want greeting", step.Params["message"])
	}
}

// --- 执行操作测试 ---

func TestExecAdd_Success(t *testing.T) {
	store := newMockStore()
	agent := NewAgent(nil, store)

	result, err := agent.execAdd(map[string]string{"key": "test-key", "value": "test-value"})
	if err != nil {
		t.Fatalf("execAdd error: %v", err)
	}
	if !strings.Contains(result, "已保存") {
		t.Errorf("result = %s, should contain 已保存", result)
	}

	k := models.ValJsonKey{Key: "test-key", Type: models.ORIGIN}
	if v, ok := store.resources[k]; !ok || v.Val != "test-value" {
		t.Error("resource not saved correctly")
	}
}

func TestExecAdd_Duplicate(t *testing.T) {
	store := newMockStore()
	k := models.ValJsonKey{Key: "existing", Type: models.ORIGIN}
	store.resources[k] = models.ValJson{Val: "old"}

	agent := NewAgent(nil, store)
	result, err := agent.execAdd(map[string]string{"key": "existing", "value": "new"})
	if err != nil {
		t.Fatalf("execAdd error: %v", err)
	}
	if !strings.Contains(result, "已存在") {
		t.Errorf("result = %s, should contain 已存在", result)
	}
}

func TestExecAdd_MissingParams(t *testing.T) {
	agent := NewAgent(nil, newMockStore())
	result, err := agent.execAdd(map[string]string{"key": "only-key"})
	if err != nil {
		t.Fatalf("execAdd error: %v", err)
	}
	if !strings.Contains(result, "需要提供") {
		t.Errorf("result = %s, should indicate missing params", result)
	}
}

func TestExecGet_All(t *testing.T) {
	store := newMockStore()
	store.resources[models.ValJsonKey{Key: "a", Type: models.ORIGIN}] = models.ValJson{Val: "va", Tag: []string{"tag-a"}}
	store.resources[models.ValJsonKey{Key: "b", Type: models.ORIGIN}] = models.ValJson{Val: "vb", Tag: []string{}}

	agent := NewAgent(nil, store)
	result, err := agent.execGet(map[string]string{})
	if err != nil {
		t.Fatalf("execGet error: %v", err)
	}
	if !strings.Contains(result, "2 条") {
		t.Errorf("result = %s, should show 2 resources", result)
	}
	// 隐私保护：不应包含 val
	if strings.Contains(result, "va") || strings.Contains(result, "vb") {
		t.Errorf("result should NOT contain val, got %s", result)
	}
	// 应包含 tag
	if !strings.Contains(result, "tag-a") {
		t.Errorf("result should contain tag, got %s", result)
	}
}

func TestExecGet_Keyword(t *testing.T) {
	store := newMockStore()
	store.resources[models.ValJsonKey{Key: "docker-link", Type: models.ORIGIN}] = models.ValJson{Val: "https://docker.com", Tag: []string{"运维"}}
	store.resources[models.ValJsonKey{Key: "go-notes", Type: models.ORIGIN}] = models.ValJson{Val: "go tips"}

	agent := NewAgent(nil, store)
	result, err := agent.execGet(map[string]string{"keyword": "docker"})
	if err != nil {
		t.Fatalf("execGet error: %v", err)
	}
	if !strings.Contains(result, "docker-link") {
		t.Errorf("result should contain docker-link, got %s", result)
	}
	if strings.Contains(result, "go-notes") {
		t.Errorf("result should not contain go-notes, got %s", result)
	}
	// 隐私保护：不应包含 val
	if strings.Contains(result, "https://docker.com") {
		t.Errorf("result should NOT contain val (URL), got %s", result)
	}
	// 应包含 tag
	if !strings.Contains(result, "运维") {
		t.Errorf("result should contain tag, got %s", result)
	}
}

func TestExecGet_ByTag(t *testing.T) {
	store := newMockStore()
	store.resources[models.ValJsonKey{Key: "item1", Type: models.ORIGIN}] = models.ValJson{Val: "v1", Tag: []string{"工作"}}
	store.resources[models.ValJsonKey{Key: "item2", Type: models.ORIGIN}] = models.ValJson{Val: "v2", Tag: []string{"生活"}}

	agent := NewAgent(nil, store)
	result, err := agent.execGet(map[string]string{"keyword": "工作"})
	if err != nil {
		t.Fatalf("execGet error: %v", err)
	}
	if !strings.Contains(result, "item1") {
		t.Errorf("should match by tag, got %s", result)
	}
}

func TestExecGet_NoMatch(t *testing.T) {
	store := newMockStore()
	store.resources[models.ValJsonKey{Key: "a", Type: models.ORIGIN}] = models.ValJson{Val: "va"}

	agent := NewAgent(nil, store)
	result, err := agent.execGet(map[string]string{"keyword": "nonexistent"})
	if err != nil {
		t.Fatalf("execGet error: %v", err)
	}
	if !strings.Contains(result, "未找到") {
		t.Errorf("result = %s, should indicate not found", result)
	}
}

func TestExecGet_Empty(t *testing.T) {
	agent := NewAgent(nil, newMockStore())
	result, err := agent.execGet(map[string]string{})
	if err != nil {
		t.Fatalf("execGet error: %v", err)
	}
	if !strings.Contains(result, "没有存储") {
		t.Errorf("result = %s, should indicate empty", result)
	}
}

func TestExecOpen_Success(t *testing.T) {
	store := newMockStore()
	store.resources[models.ValJsonKey{Key: "my-link", Type: models.ORIGIN}] = models.ValJson{Val: "https://example.com", Tag: []string{}}

	opened := ""
	agent := NewAgent(nil, store)
	agent.OpenFn = func(url string) error {
		opened = url
		return nil
	}

	result, err := agent.execOpen(map[string]string{"keyword": "my-link"})
	if err != nil {
		t.Fatalf("execOpen error: %v", err)
	}
	if !strings.Contains(result, "已打开") {
		t.Errorf("result = %s, should contain 已打开", result)
	}
	// OpenFn 应收到正确的 val
	if opened != "https://example.com" {
		t.Errorf("opened = %s, want https://example.com", opened)
	}
	// 隐私保护：Observation 不应包含 val
	if strings.Contains(result, "https://example.com") {
		t.Errorf("result should NOT contain val (URL), got %s", result)
	}
}

func TestExecOpen_NoOpenFn(t *testing.T) {
	store := newMockStore()
	store.resources[models.ValJsonKey{Key: "my-link", Type: models.ORIGIN}] = models.ValJson{Val: "https://example.com", Tag: []string{}}

	agent := NewAgent(nil, store)
	result, err := agent.execOpen(map[string]string{"keyword": "my-link"})
	if err != nil {
		t.Fatalf("execOpen error: %v", err)
	}
	if !strings.Contains(result, "请手动打开") {
		t.Errorf("result = %s, should contain 请手动打开", result)
	}
}

func TestExecOpen_MultipleMatches(t *testing.T) {
	store := newMockStore()
	store.resources[models.ValJsonKey{Key: "link-a", Type: models.ORIGIN}] = models.ValJson{Val: "https://a.com", Tag: []string{"工作"}}
	store.resources[models.ValJsonKey{Key: "link-b", Type: models.ORIGIN}] = models.ValJson{Val: "https://b.com", Tag: []string{"生活"}}

	agent := NewAgent(nil, store)
	result, err := agent.execOpen(map[string]string{"keyword": "link"})
	if err != nil {
		t.Fatalf("execOpen error: %v", err)
	}
	if !strings.Contains(result, "缩小范围") {
		t.Errorf("result = %s, should prompt to narrow down", result)
	}
	// 隐私保护：候选列表不应包含 val
	if strings.Contains(result, "https://a.com") || strings.Contains(result, "https://b.com") {
		t.Errorf("result should NOT contain val (URLs), got %s", result)
	}
}

func TestExecUpdate_Success(t *testing.T) {
	store := newMockStore()
	k := models.ValJsonKey{Key: "mykey", Type: models.ORIGIN}
	store.resources[k] = models.ValJson{Val: "old", Tag: []string{"tag1"}}

	agent := NewAgent(nil, store)
	result, err := agent.execUpdate(map[string]string{"key": "mykey", "value": "new"})
	if err != nil {
		t.Fatalf("execUpdate error: %v", err)
	}
	if !strings.Contains(result, "已更新") {
		t.Errorf("result = %s, should contain 已更新", result)
	}
	if v := store.resources[k]; len(v.Tag) != 1 || v.Tag[0] != "tag1" {
		t.Errorf("tags should be preserved, got %v", v.Tag)
	}
}

func TestExecUpdate_NotFound(t *testing.T) {
	agent := NewAgent(nil, newMockStore())
	result, err := agent.execUpdate(map[string]string{"key": "nokey", "value": "v"})
	if err != nil {
		t.Fatalf("execUpdate error: %v", err)
	}
	if !strings.Contains(result, "未找到") {
		t.Errorf("result = %s, should contain 未找到", result)
	}
}

func TestExecDelete_NeedsConfirm(t *testing.T) {
	store := newMockStore()
	k := models.ValJsonKey{Key: "delme", Type: models.ORIGIN}
	store.resources[k] = models.ValJson{Val: "some value"}

	agent := NewAgent(nil, store)
	result, err := agent.execDelete(map[string]string{"key": "delme", "confirm": "false"})
	if err != nil {
		t.Fatalf("execDelete error: %v", err)
	}
	if !strings.Contains(result, "即将删除") {
		t.Errorf("result = %s, should prompt for confirmation", result)
	}
	// 隐私保护：确认提示不应包含 val
	if strings.Contains(result, "some value") {
		t.Errorf("result should NOT contain val, got %s", result)
	}
	if _, ok := store.resources[k]; !ok {
		t.Error("resource should not be deleted without confirm")
	}
}

func TestExecDelete_Confirmed(t *testing.T) {
	store := newMockStore()
	k := models.ValJsonKey{Key: "delme", Type: models.ORIGIN}
	store.resources[k] = models.ValJson{Val: "some value"}

	agent := NewAgent(nil, store)
	result, err := agent.execDelete(map[string]string{"key": "delme", "confirm": "true"})
	if err != nil {
		t.Fatalf("execDelete error: %v", err)
	}
	if !strings.Contains(result, "已删除") {
		t.Errorf("result = %s, should contain 已删除", result)
	}
	if _, ok := store.resources[k]; ok {
		t.Error("resource should be deleted")
	}
}

func TestExecTag_Success(t *testing.T) {
	store := newMockStore()
	k := models.ValJsonKey{Key: "item", Type: models.ORIGIN}
	store.resources[k] = models.ValJson{Val: "v", Tag: []string{"old"}}

	agent := NewAgent(nil, store)
	result, err := agent.execTag(map[string]string{"key": "item", "tags": "new1, new2"})
	if err != nil {
		t.Fatalf("execTag error: %v", err)
	}
	if !strings.Contains(result, "添加标签") {
		t.Errorf("result = %s, should contain 添加标签", result)
	}
	tags := store.resources[k].Tag
	if len(tags) != 3 {
		t.Errorf("tags = %v, want 3 tags", tags)
	}
}

func TestExecDtag_Success(t *testing.T) {
	store := newMockStore()
	k := models.ValJsonKey{Key: "item", Type: models.ORIGIN}
	store.resources[k] = models.ValJson{Val: "v", Tag: []string{"a", "b", "c"}}

	agent := NewAgent(nil, store)
	result, err := agent.execDtag(map[string]string{"key": "item", "tag": "b"})
	if err != nil {
		t.Fatalf("execDtag error: %v", err)
	}
	if !strings.Contains(result, "删除标签") {
		t.Errorf("result = %s, should contain 删除标签", result)
	}
	tags := store.resources[k].Tag
	if len(tags) != 2 {
		t.Errorf("tags = %v, want 2 tags", tags)
	}
}

func TestExecRename_Success(t *testing.T) {
	store := newMockStore()
	old := models.ValJsonKey{Key: "old-name", Type: models.ORIGIN}
	store.resources[old] = models.ValJson{Val: "v", Tag: []string{"t"}}

	agent := NewAgent(nil, store)
	result, err := agent.execRename(map[string]string{"old_key": "old-name", "new_key": "new-name"})
	if err != nil {
		t.Fatalf("execRename error: %v", err)
	}
	if !strings.Contains(result, "已重命名") {
		t.Errorf("result = %s, should contain 已重命名", result)
	}
	if _, ok := store.resources[old]; ok {
		t.Error("old key should not exist")
	}
	newK := models.ValJsonKey{Key: "new-name", Type: models.ORIGIN}
	if v, ok := store.resources[newK]; !ok || v.Val != "v" {
		t.Error("new key should have old value")
	}
}

func TestExecStats(t *testing.T) {
	store := newMockStore()
	store.auditStats = models.AuditStats{
		TotalOperations: 10,
		ByOperation:     map[string]int{"get": 7, "add": 3},
		ByResource:      map[string]int{"key1": 5, "key2": 5},
	}

	agent := NewAgent(nil, store)
	result, err := agent.execStats()
	if err != nil {
		t.Fatalf("execStats error: %v", err)
	}
	if !strings.Contains(result, "10") {
		t.Errorf("result should show total, got %s", result)
	}
}

func TestExecHistory(t *testing.T) {
	store := newMockStore()
	store.history = []models.HistoryRecord{
		{ResourceKey: "k1", Operation: "get", TimeStr: "2024-01-01 10:00:00", Command: "get"},
		{ResourceKey: "k2", Operation: "add", TimeStr: "2024-01-01 09:00:00", Command: "add"},
	}

	agent := NewAgent(nil, store)
	result, err := agent.execHistory(map[string]string{"limit": "1"})
	if err != nil {
		t.Fatalf("execHistory error: %v", err)
	}
	if !strings.Contains(result, "1 条") {
		t.Errorf("result = %s, should show 1 record", result)
	}
	if strings.Contains(result, "k2") {
		t.Errorf("result should not contain k2 (limit=1), got %s", result)
	}
}

func TestExecHistory_Empty(t *testing.T) {
	agent := NewAgent(nil, newMockStore())
	result, err := agent.execHistory(map[string]string{})
	if err != nil {
		t.Fatalf("execHistory error: %v", err)
	}
	if !strings.Contains(result, "暂无") {
		t.Errorf("result = %s, should indicate empty", result)
	}
}

func TestExecute_UnknownAction(t *testing.T) {
	agent := NewAgent(nil, newMockStore())
	step := &Step{Action: "unknown", Params: map[string]string{}}
	result, err := agent.execute(step)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if !strings.Contains(result, "未知操作") {
		t.Errorf("result = %s, should indicate unknown action", result)
	}
}

// --- ReAct Run 端到端测试 ---

func TestRun_SingleRound_Answer(t *testing.T) {
	// 简单闲聊：第一轮直接 answer
	chatFn := mockReActChat(`{"thought":"用户打招呼","action":"answer","params":{"message":"你好，我是 ttl 助手"}}`)
	agent := NewAgent(chatFn, newMockStore())

	result, err := agent.Run("你好")
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result != "你好，我是 ttl 助手" {
		t.Errorf("result = %s, want greeting", result)
	}
}

func TestRun_SingleRound_Add(t *testing.T) {
	// 直接新增：第一轮 add → Observation → 第二轮 answer
	store := newMockStore()
	chatFn := mockReActChat(
		`{"thought":"用户要添加笔记","action":"add","params":{"key":"my-note","value":"some note"}}`,
		`{"thought":"添加成功了","action":"answer","params":{"message":"已保存：my-note → some note"}}`,
	)
	agent := NewAgent(chatFn, store)

	result, err := agent.Run("帮我记一下 some note")
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if !strings.Contains(result, "已保存") {
		t.Errorf("result = %s, should contain 已保存", result)
	}

	k := models.ValJsonKey{Key: "my-note", Type: models.ORIGIN}
	if _, ok := store.resources[k]; !ok {
		t.Error("resource should be saved")
	}
}

func TestRun_SingleRound_Get(t *testing.T) {
	store := newMockStore()
	store.resources[models.ValJsonKey{Key: "docker-link", Type: models.ORIGIN}] = models.ValJson{Val: "https://docker.com"}

	chatFn := mockReActChat(
		`{"thought":"查docker","action":"get","params":{"keyword":"docker"}}`,
		`{"thought":"找到了","action":"answer","params":{"message":"找到 docker-link → https://docker.com"}}`,
	)
	agent := NewAgent(chatFn, store)

	result, err := agent.Run("查一下docker")
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if !strings.Contains(result, "docker") {
		t.Errorf("result = %s, should contain docker", result)
	}
}

func TestRun_MultiRound_Summary(t *testing.T) {
	// 总结场景：先 get 获取数据 → 再 answer 总结
	store := newMockStore()
	store.resources[models.ValJsonKey{Key: "log1", Type: models.ORIGIN}] = models.ValJson{Val: "日志内容1", Tag: []string{"日志"}}
	store.resources[models.ValJsonKey{Key: "log2", Type: models.ORIGIN}] = models.ValJson{Val: "日志内容2", Tag: []string{"日志"}}

	chatFn := mockReActChat(
		`{"thought":"用户要总结日志，我先获取所有数据","action":"get","params":{"keyword":"日志"}}`,
		`{"thought":"拿到了2条日志，现在总结","action":"answer","params":{"message":"共有2条日志记录：日志内容1和日志内容2"}}`,
	)
	agent := NewAgent(chatFn, store)

	result, err := agent.Run("总结一下我的日志")
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if !strings.Contains(result, "2条日志") {
		t.Errorf("result = %s, should contain summary", result)
	}
}

func TestRun_MultiRound_SearchThenOpen(t *testing.T) {
	// 搜索 → 匹配唯一 → 打开
	store := newMockStore()
	store.resources[models.ValJsonKey{Key: "sugar-board", Type: models.ORIGIN}] = models.ValJson{Val: "https://sugar.example.com"}

	opened := ""
	chatFn := mockReActChat(
		`{"thought":"用户要打开sugar看板","action":"open","params":{"keyword":"sugar"}}`,
		`{"thought":"已打开","action":"answer","params":{"message":"已打开 sugar 看板"}}`,
	)
	agent := NewAgent(chatFn, store)
	agent.OpenFn = func(url string) error {
		opened = url
		return nil
	}

	result, err := agent.Run("打开sugar看板")
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if !strings.Contains(result, "已打开") {
		t.Errorf("result = %s, should contain 已打开", result)
	}
	if opened != "https://sugar.example.com" {
		t.Errorf("opened = %s, want sugar URL", opened)
	}
}

func TestRun_InvalidJSON_Fallback(t *testing.T) {
	// LLM 返回非 JSON → 降级为直接回复
	chatFn := mockReActChat("这不是有效的JSON，我直接回复你")
	agent := NewAgent(chatFn, newMockStore())

	result, err := agent.Run("随便说点什么")
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result != "这不是有效的JSON，我直接回复你" {
		t.Errorf("result = %s, should be raw fallback", result)
	}
}

func TestRun_LLMError(t *testing.T) {
	errFn := func(messages []Message) (string, error) {
		return "", fmt.Errorf("network error")
	}
	agent := NewAgent(errFn, newMockStore())

	_, err := agent.Run("test")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "AI 调用失败") {
		t.Errorf("error = %v, should mention AI call failure", err)
	}
}

func TestRun_MaxSteps_Exceeded(t *testing.T) {
	// 每轮都返回 get，永远不 answer → 超过 MaxSteps
	chatFn := func(messages []Message) (string, error) {
		return `{"thought":"继续查","action":"get","params":{"keyword":"test"}}`, nil
	}

	store := newMockStore()
	store.resources[models.ValJsonKey{Key: "test", Type: models.ORIGIN}] = models.ValJson{Val: "v"}

	agent := NewAgent(chatFn, store)
	agent.MaxSteps = 3

	result, err := agent.Run("无限查")
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	// 应该返回最后一次 Observation（get 的结果）
	if !strings.Contains(result, "test") {
		t.Errorf("result = %s, should contain last observation", result)
	}
}

func TestRun_ExecuteError_AsObservation(t *testing.T) {
	// 执行失败 → 错误作为 Observation 回传 → LLM 可以纠正
	store := newMockStore()

	chatFn := mockReActChat(
		// 第一轮：试图更新不存在的资源
		`{"thought":"更新资源","action":"update","params":{"key":"nonexistent","value":"new"}}`,
		// 第二轮：收到"未找到"后 answer
		`{"thought":"资源不存在","action":"answer","params":{"message":"未找到该资源，请确认名称"}}`,
	)
	agent := NewAgent(chatFn, store)

	result, err := agent.Run("更新 nonexistent")
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if !strings.Contains(result, "未找到") {
		t.Errorf("result = %s, should indicate not found", result)
	}
}

func TestRun_ThreeRounds(t *testing.T) {
	// 三轮：get all → get keyword → answer
	store := newMockStore()
	store.resources[models.ValJsonKey{Key: "work-doc", Type: models.ORIGIN}] = models.ValJson{Val: "工作文档", Tag: []string{"工作"}}
	store.resources[models.ValJsonKey{Key: "life-note", Type: models.ORIGIN}] = models.ValJson{Val: "生活笔记", Tag: []string{"生活"}}

	chatFn := mockReActChat(
		`{"thought":"先看所有数据","action":"get","params":{}}`,
		`{"thought":"有2条，再看工作相关的","action":"get","params":{"keyword":"工作"}}`,
		`{"thought":"分析完成","action":"answer","params":{"message":"工作类资源有1条：work-doc"}}`,
	)
	agent := NewAgent(chatFn, store)

	result, err := agent.Run("哪些是工作相关的")
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if !strings.Contains(result, "work-doc") {
		t.Errorf("result = %s, should contain work-doc", result)
	}
}

// --- Log 操作测试 ---

func TestExecLogWrite_Success(t *testing.T) {
	store := newMockStore()
	agent := NewAgent(nil, store)

	result, err := agent.execLogWrite(map[string]string{"content": "完成用户模块重构", "tags": "项目A, 开发"})
	if err != nil {
		t.Fatalf("execLogWrite error: %v", err)
	}
	if !strings.Contains(result, "日志已记录") {
		t.Errorf("result = %s, should contain 日志已记录", result)
	}
	if !strings.Contains(result, "项目A") {
		t.Errorf("result = %s, should contain tag", result)
	}
	if len(store.logs) != 1 {
		t.Fatalf("logs count = %d, want 1", len(store.logs))
	}
	if store.logs[0].Content != "完成用户模块重构" {
		t.Errorf("log content = %s", store.logs[0].Content)
	}
}

func TestExecLogWrite_MissingContent(t *testing.T) {
	agent := NewAgent(nil, newMockStore())
	result, err := agent.execLogWrite(map[string]string{})
	if err != nil {
		t.Fatalf("execLogWrite error: %v", err)
	}
	if !strings.Contains(result, "需要提供") {
		t.Errorf("result = %s, should indicate missing content", result)
	}
}

func TestExecLogList_Success(t *testing.T) {
	store := newMockStore()
	store.logs = []models.LogRecord{
		{ID: 1, Content: "写代码", Tags: []string{"开发"}, CreatedAt: "2026-04-04 14:30:00", Date: "2026-04-04"},
		{ID: 2, Content: "开会", Tags: []string{"会议"}, CreatedAt: "2026-04-04 10:00:00", Date: "2026-04-04"},
	}

	agent := NewAgent(nil, store)
	result, err := agent.execLogList(map[string]string{"start_date": "2026-04-04", "end_date": "2026-04-04"})
	if err != nil {
		t.Fatalf("execLogList error: %v", err)
	}
	if !strings.Contains(result, "写代码") || !strings.Contains(result, "开会") {
		t.Errorf("result = %s, should contain both logs", result)
	}
}

func TestExecLogList_Empty(t *testing.T) {
	agent := NewAgent(nil, newMockStore())
	result, err := agent.execLogList(map[string]string{"start_date": "2026-04-04", "end_date": "2026-04-04"})
	if err != nil {
		t.Fatalf("execLogList error: %v", err)
	}
	if !strings.Contains(result, "暂无日志") {
		t.Errorf("result = %s, should indicate empty", result)
	}
}

func TestExecLogList_FilterByTag(t *testing.T) {
	store := newMockStore()
	store.logs = []models.LogRecord{
		{ID: 1, Content: "写代码", Tags: []string{"开发"}, CreatedAt: "2026-04-04 14:30:00", Date: "2026-04-04"},
		{ID: 2, Content: "开会", Tags: []string{"会议"}, CreatedAt: "2026-04-04 10:00:00", Date: "2026-04-04"},
	}

	agent := NewAgent(nil, store)
	result, err := agent.execLogList(map[string]string{"start_date": "2026-04-04", "end_date": "2026-04-04", "tag": "开发"})
	if err != nil {
		t.Fatalf("execLogList error: %v", err)
	}
	if !strings.Contains(result, "写代码") {
		t.Errorf("result should contain 写代码, got %s", result)
	}
	if strings.Contains(result, "开会") {
		t.Errorf("result should not contain 开会 (filtered), got %s", result)
	}
}

func TestRun_LogWriteThenSummary(t *testing.T) {
	// 多步：log_write → log_list → answer（周报总结）
	store := newMockStore()
	store.logs = []models.LogRecord{
		{ID: 1, Content: "完成用户模块", Tags: []string{"项目A"}, CreatedAt: "2026-04-01 10:00:00", Date: "2026-04-01"},
		{ID: 2, Content: "修复登录bug", Tags: []string{"项目A"}, CreatedAt: "2026-04-02 14:00:00", Date: "2026-04-02"},
		{ID: 3, Content: "需求评审", Tags: []string{"项目B"}, CreatedAt: "2026-04-03 09:00:00", Date: "2026-04-03"},
	}

	chatFn := mockReActChat(
		`{"thought":"用户要看周报，先获取本周日志","action":"log_list","params":{"range":"week"}}`,
		`{"thought":"拿到3条日志，生成周报","action":"answer","params":{"message":"本周工作总结：\n1. 项目A：完成用户模块开发，修复登录bug\n2. 项目B：参加需求评审"}}`,
	)
	agent := NewAgent(chatFn, store)

	result, err := agent.Run("帮我总结一下这周的周报")
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if !strings.Contains(result, "周") && !strings.Contains(result, "总结") {
		t.Errorf("result = %s, should contain weekly summary", result)
	}
}

// --- truncateObservation 测试 ---

func TestTruncateObservation_Short(t *testing.T) {
	result := truncateObservation("short text", 100)
	if result != "short text" {
		t.Errorf("should not truncate short text, got %s", result)
	}
}

func TestTruncateObservation_Long(t *testing.T) {
	// 构造超长文本
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %d: some content here", i)
	}
	long := strings.Join(lines, "\n")

	result := truncateObservation(long, 200)
	if len(result) > 250 { // allow extra length for truncation notice
		t.Errorf("truncated result too long: %d chars", len(result))
	}
	if !strings.Contains(result, "truncated") {
		t.Errorf("should contain truncation notice, got %s", result)
	}
}

// --- cleanJSONResponse 测试 ---

func TestCleanJSONResponse(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`{"action":"get"}`, `{"action":"get"}`},
		{"```json\n{\"action\":\"get\"}\n```", `{"action":"get"}`},
		{"```\n{\"action\":\"get\"}\n```", `{"action":"get"}`},
		{"  {\"action\":\"get\"}  ", `{"action":"get"}`},
	}

	for _, tt := range tests {
		got := cleanJSONResponse(tt.input)
		if got != tt.want {
			t.Errorf("cleanJSONResponse(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- matchTags 测试 ---

func TestMatchTags(t *testing.T) {
	if !matchTags([]string{"Docker", "运维"}, "docker") {
		t.Error("should match case-insensitive")
	}
	if matchTags([]string{"Docker"}, "python") {
		t.Error("should not match")
	}
	if matchTags([]string{}, "anything") {
		t.Error("empty tags should not match")
	}
}

// --- 数据隐私保护测试 ---

func TestExecGet_All_NoValInObservation(t *testing.T) {
	store := newMockStore()
	store.resources[models.ValJsonKey{Key: "secret-token", Type: models.ORIGIN}] = models.ValJson{
		Val: "sk-12345-very-secret-key",
		Tag: []string{"密钥", "生产"},
	}
	store.resources[models.ValJsonKey{Key: "internal-url", Type: models.ORIGIN}] = models.ValJson{
		Val: "https://internal.example.com/admin",
		Tag: []string{"internal"},
	}

	agent := NewAgent(nil, store)
	result, err := agent.execGet(map[string]string{})
	if err != nil {
		t.Fatalf("execGet error: %v", err)
	}
	// 应包含 key 和 tag
	if !strings.Contains(result, "secret-token") || !strings.Contains(result, "internal-url") {
		t.Errorf("result should contain keys, got %s", result)
	}
	if !strings.Contains(result, "密钥") || !strings.Contains(result, "internal") {
		t.Errorf("result should contain tags, got %s", result)
	}
	// 不应包含敏感 val
	if strings.Contains(result, "sk-12345") || strings.Contains(result, "example.com") {
		t.Errorf("result should NOT contain sensitive val, got %s", result)
	}
}

func TestExecGet_Keyword_NoValInObservation(t *testing.T) {
	store := newMockStore()
	store.resources[models.ValJsonKey{Key: "db-password", Type: models.ORIGIN}] = models.ValJson{
		Val: "root:P@ssw0rd@192.168.1.100:3306",
		Tag: []string{"数据库", "生产"},
	}

	agent := NewAgent(nil, store)
	result, err := agent.execGet(map[string]string{"keyword": "db"})
	if err != nil {
		t.Fatalf("execGet error: %v", err)
	}
	if !strings.Contains(result, "db-password") {
		t.Errorf("result should contain key, got %s", result)
	}
	if !strings.Contains(result, "数据库") {
		t.Errorf("result should contain tag, got %s", result)
	}
	if strings.Contains(result, "P@ssw0rd") || strings.Contains(result, "192.168.1.100") {
		t.Errorf("result should NOT contain sensitive val, got %s", result)
	}
}

func TestExecOpen_SingleMatch_NoValInObservation(t *testing.T) {
	store := newMockStore()
	store.resources[models.ValJsonKey{Key: "admin-panel", Type: models.ORIGIN}] = models.ValJson{
		Val: "https://admin.internal.com/secret-path",
		Tag: []string{"内网"},
	}

	opened := ""
	agent := NewAgent(nil, store)
	agent.OpenFn = func(url string) error {
		opened = url
		return nil
	}

	result, err := agent.execOpen(map[string]string{"keyword": "admin"})
	if err != nil {
		t.Fatalf("execOpen error: %v", err)
	}
	// OpenFn 应收到完整 val（本地执行）
	if opened != "https://admin.internal.com/secret-path" {
		t.Errorf("OpenFn should receive full val, got %s", opened)
	}
	// Observation 不应包含 val
	if strings.Contains(result, "admin.internal.com") || strings.Contains(result, "secret-path") {
		t.Errorf("result should NOT contain val, got %s", result)
	}
	if !strings.Contains(result, "已打开") && !strings.Contains(result, "admin-panel") {
		t.Errorf("result should contain key, got %s", result)
	}
}

func TestExecDelete_NoValInObservation(t *testing.T) {
	store := newMockStore()
	store.resources[models.ValJsonKey{Key: "api-key", Type: models.ORIGIN}] = models.ValJson{
		Val: "Bearer eyJhbGciOiJIUzI1NiJ9.sensitive-jwt-token",
	}

	agent := NewAgent(nil, store)
	result, err := agent.execDelete(map[string]string{"key": "api-key", "confirm": "false"})
	if err != nil {
		t.Fatalf("execDelete error: %v", err)
	}
	if !strings.Contains(result, "即将删除") || !strings.Contains(result, "api-key") {
		t.Errorf("result should contain deletion prompt with key, got %s", result)
	}
	if strings.Contains(result, "Bearer") || strings.Contains(result, "eyJhbGci") {
		t.Errorf("result should NOT contain sensitive val, got %s", result)
	}
}

func TestExecLogList_FullContentInObservation(t *testing.T) {
	store := newMockStore()
	store.logs = []models.LogRecord{
		{ID: 1, Content: "完成API接口开发，包含用户认证模块", Tags: []string{"开发", "后端"}, CreatedAt: "2026-04-04 14:30:00", Date: "2026-04-04"},
	}

	agent := NewAgent(nil, store)
	result, err := agent.execLogList(map[string]string{"start_date": "2026-04-04", "end_date": "2026-04-04"})
	if err != nil {
		t.Fatalf("execLogList error: %v", err)
	}
	// 日志应包含完整 content（不受隐私保护限制）
	if !strings.Contains(result, "完成API接口开发") {
		t.Errorf("result should contain full log content, got %s", result)
	}
	if !strings.Contains(result, "开发") {
		t.Errorf("result should contain log tags, got %s", result)
	}
}
