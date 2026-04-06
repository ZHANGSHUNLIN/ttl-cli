package integration_test

import (
	"strings"
	"testing"

	"ttl-cli/ai"
	"ttl-cli/db"
	"ttl-cli/models"
)

type dbStoreAdapter struct{}

func (d *dbStoreAdapter) GetAllResources() (map[models.ValJsonKey]models.ValJson, error) {
	return db.GetAllResources()
}
func (d *dbStoreAdapter) SaveResource(key models.ValJsonKey, value models.ValJson) error {
	return db.SaveResource(key, value)
}
func (d *dbStoreAdapter) DeleteResource(key models.ValJsonKey) error {
	return db.DeleteResource(key)
}
func (d *dbStoreAdapter) UpdateResource(key models.ValJsonKey, newValue models.ValJson) error {
	return db.UpdateResource(key, newValue)
}
func (d *dbStoreAdapter) GetAuditStats() (models.AuditStats, error) {
	return db.GetAuditStats()
}
func (d *dbStoreAdapter) GetAllHistoryRecords() ([]models.HistoryRecord, error) {
	return db.GetAllHistoryRecords()
}
func (d *dbStoreAdapter) SaveLogRecord(record models.LogRecord) error {
	return db.SaveLogRecord(record)
}
func (d *dbStoreAdapter) GetLogRecords(startDate, endDate string) ([]models.LogRecord, error) {
	return db.GetLogRecords(startDate, endDate)
}
func (d *dbStoreAdapter) DeleteLogRecord(id int64) error {
	return db.DeleteLogRecord(id)
}

func mockReActChat(responses ...string) ai.ChatFunc {
	return func(messages []ai.Message) (string, error) {
		round := (len(messages) - 2) / 2
		if round < 0 {
			round = 0
		}
		if round < len(responses) {
			return responses[round], nil
		}
		return `{"thought":"done","action":"answer","params":{"message":"完成"}}`, nil
	}
}

func TestAI_AddAndGet(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	store := &dbStoreAdapter{}

	addChat := mockReActChat(
		`{"thought":"用户要记笔记","action":"add","params":{"key":"test-note","value":"这是一条测试笔记"}}`,
		`{"thought":"添加成功","action":"answer","params":{"message":"已保存：test-note"}}`,
	)
	agent := ai.NewAgent(addChat, store)

	result, err := agent.Run("帮我记一下这是一条测试笔记")
	if err != nil {
		t.Fatalf("AI add error: %v", err)
	}
	if !strings.Contains(result, "已保存") {
		t.Errorf("add result = %s, should contain 已保存", result)
	}

	resources, err := db.GetAllResources()
	if err != nil {
		t.Fatalf("GetAllResources error: %v", err)
	}
	k := models.ValJsonKey{Key: "test-note", Type: models.ORIGIN}
	v, ok := resources[k]
	if !ok {
		t.Fatal("resource should exist in db")
	}
	if v.Val != "这是一条测试笔记" {
		t.Errorf("value = %s, want 这是一条测试笔记", v.Val)
	}

	getChat := mockReActChat(
		`{"thought":"查test","action":"get","params":{"keyword":"test"}}`,
		`{"thought":"找到了","action":"answer","params":{"message":"找到 test-note"}}`,
	)
	agent2 := ai.NewAgent(getChat, store)

	result2, err := agent2.Run("查一下test相关的内容")
	if err != nil {
		t.Fatalf("AI get error: %v", err)
	}
	if !strings.Contains(result2, "test-note") {
		t.Errorf("get result = %s, should contain test-note", result2)
	}
}

func TestAI_UpdatePreservesTags(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	store := &dbStoreAdapter{}

	k := models.ValJsonKey{Key: "my-link", Type: models.ORIGIN}
	v := models.ValJson{Val: "https://old.com", Tag: []string{"工作", "收藏"}}
	if err := db.SaveResource(k, v); err != nil {
		t.Fatalf("save error: %v", err)
	}

	updateChat := mockReActChat(
		`{"thought":"更新链接","action":"update","params":{"key":"my-link","value":"https://new.com"}}`,
		`{"thought":"更新成功","action":"answer","params":{"message":"已更新：my-link → https://new.com"}}`,
	)
	agent := ai.NewAgent(updateChat, store)

	result, err := agent.Run("把 my-link 更新成 https://new.com")
	if err != nil {
		t.Fatalf("AI update error: %v", err)
	}
	if !strings.Contains(result, "已更新") {
		t.Errorf("update result = %s, should contain 已更新", result)
	}

	resources, _ := db.GetAllResources()
	updated := resources[k]
	if updated.Val != "https://new.com" {
		t.Errorf("value = %s, want https://new.com", updated.Val)
	}
	if len(updated.Tag) != 2 || updated.Tag[0] != "工作" || updated.Tag[1] != "收藏" {
		t.Errorf("tags = %v, should be preserved", updated.Tag)
	}
}

func TestAI_DeleteConfirmFlow(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	store := &dbStoreAdapter{}

	k := models.ValJsonKey{Key: "temp-item", Type: models.ORIGIN}
	v := models.ValJson{Val: "临时数据", Tag: []string{}}
	if err := db.SaveResource(k, v); err != nil {
		t.Fatalf("save error: %v", err)
	}

	deleteChat := mockReActChat(
		`{"thought":"用户要删除","action":"delete","params":{"key":"temp-item","confirm":"false"}}`,
		`{"thought":"需要确认","action":"answer","params":{"message":"即将删除 temp-item，请确认"}}`,
	)
	agent := ai.NewAgent(deleteChat, store)

	result, err := agent.Run("删掉 temp-item")
	if err != nil {
		t.Fatalf("AI delete error: %v", err)
	}
	if !strings.Contains(result, "即将删除") || !strings.Contains(result, "确认") {
		t.Errorf("delete result = %s, should prompt confirmation", result)
	}

	resources, _ := db.GetAllResources()
	if _, ok := resources[k]; !ok {
		t.Error("resource should still exist (not confirmed)")
	}
}

func TestAI_TagAndDtag(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	store := &dbStoreAdapter{}

	k := models.ValJsonKey{Key: "article", Type: models.ORIGIN}
	v := models.ValJson{Val: "文章内容", Tag: []string{}}
	if err := db.SaveResource(k, v); err != nil {
		t.Fatalf("save error: %v", err)
	}

	tagChat := mockReActChat(
		`{"thought":"打标签","action":"tag","params":{"key":"article","tags":"技术, 收藏"}}`,
		`{"thought":"成功","action":"answer","params":{"message":"已添加标签"}}`,
	)
	agent := ai.NewAgent(tagChat, store)

	result, err := agent.Run("给 article 加上技术和收藏标签")
	if err != nil {
		t.Fatalf("AI tag error: %v", err)
	}
	if !strings.Contains(result, "添加标签") || !strings.Contains(result, "已") {
		t.Errorf("tag result = %s, should contain 添加标签", result)
	}

	resources, _ := db.GetAllResources()
	if len(resources[k].Tag) != 2 {
		t.Errorf("tags = %v, want 2 tags", resources[k].Tag)
	}

	dtagChat := mockReActChat(
		`{"thought":"删标签","action":"dtag","params":{"key":"article","tag":"收藏"}}`,
		`{"thought":"成功","action":"answer","params":{"message":"已删除标签"}}`,
	)
	agent2 := ai.NewAgent(dtagChat, store)

	result2, err := agent2.Run("删掉 article 的收藏标签")
	if err != nil {
		t.Fatalf("AI dtag error: %v", err)
	}
	if !strings.Contains(result2, "删除标签") || !strings.Contains(result2, "已") {
		t.Errorf("dtag result = %s, should contain 删除标签", result2)
	}

	resources, _ = db.GetAllResources()
	if len(resources[k].Tag) != 1 || resources[k].Tag[0] != "技术" {
		t.Errorf("tags after dtag = %v, want [技术]", resources[k].Tag)
	}
}

func TestAI_Rename(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	store := &dbStoreAdapter{}

	k := models.ValJsonKey{Key: "old-name", Type: models.ORIGIN}
	v := models.ValJson{Val: "内容", Tag: []string{"tag1"}}
	if err := db.SaveResource(k, v); err != nil {
		t.Fatalf("save error: %v", err)
	}

	renameChat := mockReActChat(
		`{"thought":"重命名","action":"rename","params":{"old_key":"old-name","new_key":"new-name"}}`,
		`{"thought":"成功","action":"answer","params":{"message":"已重命名：old-name → new-name"}}`,
	)
	agent := ai.NewAgent(renameChat, store)

	result, err := agent.Run("把 old-name 改名为 new-name")
	if err != nil {
		t.Fatalf("AI rename error: %v", err)
	}
	if !strings.Contains(result, "已重命名") {
		t.Errorf("rename result = %s, should contain 已重命名", result)
	}

	resources, _ := db.GetAllResources()
	if _, ok := resources[k]; ok {
		t.Error("old key should not exist")
	}
	newK := models.ValJsonKey{Key: "new-name", Type: models.ORIGIN}
	if r, ok := resources[newK]; !ok || r.Val != "内容" || len(r.Tag) != 1 {
		t.Errorf("new resource = %+v, should have old value and tags", r)
	}
}

func TestAI_Summary_MultiStep(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	store := &dbStoreAdapter{}

	for i := 0; i < 3; i++ {
		k := models.ValJsonKey{Key: strings.ReplaceAll("log-{i}", "{i}", string(rune('1'+i))), Type: models.ORIGIN}
		v := models.ValJson{Val: "日志内容" + string(rune('1'+i)), Tag: []string{"日志"}}
		if err := db.SaveResource(k, v); err != nil {
			t.Fatalf("save error: %v", err)
		}
	}

	summaryChat := mockReActChat(
		`{"thought":"用户要总结日志，先查看数据","action":"get","params":{"keyword":"日志"}}`,
		`{"thought":"拿到日志数据，生成总结","action":"answer","params":{"message":"共有3条日志记录，内容涵盖日志1到日志3。"}}`,
	)

	agent := ai.NewAgent(summaryChat, store)
	result, err := agent.Run("总结一下我的日志")
	if err != nil {
		t.Fatalf("AI summary error: %v", err)
	}
	if !strings.Contains(result, "日志") {
		t.Errorf("summary result = %s, should contain 日志", result)
	}
}

func TestAI_ChatAnswer(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	store := &dbStoreAdapter{}

	chatFn := mockReActChat(
		`{"thought":"用户打招呼","action":"answer","params":{"message":"你好，我是你的数据助手"}}`,
	)
	agent := ai.NewAgent(chatFn, store)

	result, err := agent.Run("你好")
	if err != nil {
		t.Fatalf("AI chat error: %v", err)
	}
	if !strings.Contains(result, "数据助手") {
		t.Errorf("chat result = %s, should contain greeting", result)
	}
}

func TestAI_InvalidJSON_Fallback(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	store := &dbStoreAdapter{}

	chatFn := mockReActChat("这不是有效的JSON，我直接回复你")
	agent := ai.NewAgent(chatFn, store)

	result, err := agent.Run("随便说点什么")
	if err != nil {
		t.Fatalf("AI fallback error: %v", err)
	}
	if result == "" {
		t.Error("should return something even with invalid JSON")
	}
}

func TestAI_StorageIsolation(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	store := &dbStoreAdapter{}

	k := models.ValJsonKey{Key: "direct-add", Type: models.ORIGIN}
	v := models.ValJson{Val: "direct value", Tag: []string{}}
	if err := db.SaveResource(k, v); err != nil {
		t.Fatalf("save error: %v", err)
	}

	getChat := mockReActChat(
		`{"thought":"查direct","action":"get","params":{"keyword":"direct"}}`,
		`{"thought":"找到了","action":"answer","params":{"message":"找到 direct-add"}}`,
	)
	agent := ai.NewAgent(getChat, store)

	result, err := agent.Run("查direct")
	if err != nil {
		t.Fatalf("AI get error: %v", err)
	}
	if !strings.Contains(result, "direct-add") {
		t.Errorf("AI should find resource added via db package, got %s", result)
	}
}

func TestAI_Privacy_GetObservationNoVal(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	store := &dbStoreAdapter{}

	k := models.ValJsonKey{Key: "prod-db-password", Type: models.ORIGIN}
	v := models.ValJson{Val: "root:SecretP@ss@192.168.1.100:3306", Tag: []string{"数据库", "生产"}}
	if err := db.SaveResource(k, v); err != nil {
		t.Fatalf("save error: %v", err)
	}

	var observationContent string
	chatFn := func(messages []ai.Message) (string, error) {
		round := (len(messages) - 2) / 2
		if round == 1 && len(messages) >= 4 {
			observationContent = messages[3].Content
		}
		switch round {
		case 0:
			return `{"thought":"查数据库","action":"get","params":{"keyword":"db"}}`, nil
		default:
			return `{"thought":"找到了","action":"answer","params":{"message":"找到 prod-db-password"}}`, nil
		}
	}

	agent := ai.NewAgent(chatFn, store)
	_, err := agent.Run("查一下数据库相关的")
	if err != nil {
		t.Fatalf("AI get error: %v", err)
	}

	if strings.Contains(observationContent, "SecretP@ss") {
		t.Errorf("Observation should NOT contain password, got: %s", observationContent)
	}
	if strings.Contains(observationContent, "192.168.1.100") {
		t.Errorf("Observation should NOT contain IP, got: %s", observationContent)
	}
	if !strings.Contains(observationContent, "prod-db-password") {
		t.Errorf("Observation should contain key, got: %s", observationContent)
	}
	if !strings.Contains(observationContent, "数据库") {
		t.Errorf("Observation should contain tag, got: %s", observationContent)
	}
}
