package integration_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"ttl-cli/db"
	"ttl-cli/models"
	"ttl-cli/util"
)

func setupTempStorage(t *testing.T) func() {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "ttl-integration-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	confPath := filepath.Join(tmpDir, "test.ini")
	confContent := fmt.Sprintf("db_path = %s\n", dbPath)
	if err := os.WriteFile(confPath, []byte(confContent), 0644); err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("写入临时配置文件失败: %v", err)
	}

	if err := db.InitDB("local", "", "", 0, confPath); err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("初始化临时存储失败: %v", err)
	}

	return func() {
		_ = db.CloseDB()
		_ = os.RemoveAll(tmpDir)
	}
}

func TestResourceLifecycle(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	key := models.ValJsonKey{Key: "github", Type: models.ORIGIN}
	value := models.ValJson{Val: "https://github.com", Tag: []string{}}

	if err := db.SaveResource(key, value); err != nil {
		t.Fatalf("SaveResource() 失败: %v", err)
	}

	resources, err := db.GetAllResources()
	if err != nil {
		t.Fatalf("GetAllResources() 失败: %v", err)
	}
	saved, ok := resources[key]
	if !ok {
		t.Fatal("保存后未能查到资源")
	}
	if saved.Val != value.Val {
		t.Errorf("资源值不符: got %q, want %q", saved.Val, value.Val)
	}

	updated := models.ValJson{Val: "https://github.com/new", Tag: []string{"dev"}}
	if err := db.UpdateResource(key, updated); err != nil {
		t.Fatalf("UpdateResource() 失败: %v", err)
	}

	resources, err = db.GetAllResources()
	if err != nil {
		t.Fatalf("更新后 GetAllResources() 失败: %v", err)
	}
	if resources[key].Val != updated.Val {
		t.Errorf("更新后值不符: got %q, want %q", resources[key].Val, updated.Val)
	}

	if err := db.DeleteResource(key); err != nil {
		t.Fatalf("DeleteResource() 失败: %v", err)
	}

	resources, err = db.GetAllResources()
	if err != nil {
		t.Fatalf("删除后 GetAllResources() 失败: %v", err)
	}
	if _, exists := resources[key]; exists {
		t.Error("删除后资源仍然存在")
	}
}

func TestAuditLifecycle(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	if err := db.SaveResource(
		models.ValJsonKey{Key: "my-note", Type: models.ORIGIN},
		models.ValJson{Val: "some content", Tag: []string{}},
	); err != nil {
		t.Fatalf("SaveResource() 失败: %v", err)
	}

	if err := db.RecordAudit("my-note", "add"); err != nil {
		t.Fatalf("RecordAudit(add) 失败: %v", err)
	}
	if err := db.RecordAudit("my-note", "get"); err != nil {
		t.Fatalf("RecordAudit(get) 失败: %v", err)
	}

	stats, err := db.GetAuditStats()
	if err != nil {
		t.Fatalf("GetAuditStats() 失败: %v", err)
	}
	if stats.TotalOperations != 2 {
		t.Errorf("TotalOperations = %d, want 2", stats.TotalOperations)
	}
	if stats.ByResource["my-note"] != 2 {
		t.Errorf("ByResource[my-note] = %d, want 2", stats.ByResource["my-note"])
	}

	if err := db.DeleteAuditRecords("my-note"); err != nil {
		t.Fatalf("DeleteAuditRecords() 失败: %v", err)
	}
	stats, err = db.GetAuditStats()
	if err != nil {
		t.Fatalf("删除后 GetAuditStats() 失败: %v", err)
	}
	if stats.TotalOperations != 0 {
		t.Errorf("删除后 TotalOperations = %d, want 0", stats.TotalOperations)
	}
}

func TestHistoryLifecycle(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	if err := db.RecordCommandHistory("add", "key-a", false); err != nil {
		t.Fatalf("RecordCommandHistory(add) 失败: %v", err)
	}
	if err := db.RecordCommandHistory("get", "key-b", false); err != nil {
		t.Fatalf("RecordCommandHistory(get) 失败: %v", err)
	}

	records, err := db.GetAllHistoryRecords()
	if err != nil {
		t.Fatalf("GetAllHistoryRecords() 失败: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("历史记录数量 = %d, want 2", len(records))
	}
	if records[0].Timestamp < records[1].Timestamp {
		t.Error("历史记录未按时间倒序排列")
	}

	if err := db.DeleteHistoryRecords("key-a"); err != nil {
		t.Fatalf("DeleteHistoryRecords() 失败: %v", err)
	}
	records, err = db.GetAllHistoryRecords()
	if err != nil {
		t.Fatalf("删除后 GetAllHistoryRecords() 失败: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("删除后记录数量 = %d, want 1", len(records))
	}
	if records[0].ResourceKey != "key-b" {
		t.Errorf("剩余记录 ResourceKey = %q, want key-b", records[0].ResourceKey)
	}
}

func TestStorageIsolation(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	key := models.ValJsonKey{Key: "isolation-check", Type: models.ORIGIN}
	_ = db.SaveResource(key, models.ValJson{Val: "value", Tag: []string{}})

	resources, _ := db.GetAllResources()
	if len(resources) != 1 {
		t.Errorf("期望仅有 1 条资源，实际 %d 条（可能受其他测试污染）", len(resources))
	}
}

func TestUpdatePreservesTags(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	key := models.ValJsonKey{Key: "mykey", Type: models.ORIGIN}

	if err := db.SaveResource(key, models.ValJson{Val: "v1", Tag: []string{}}); err != nil {
		t.Fatalf("SaveResource() 失败: %v", err)
	}

	if err := db.UpdateResource(key, models.ValJson{Val: "v1", Tag: []string{"work", "important"}}); err != nil {
		t.Fatalf("UpdateResource(添加 tag) 失败: %v", err)
	}

	resources, err := db.GetAllResources()
	if err != nil {
		t.Fatalf("GetAllResources() 失败: %v", err)
	}
	existing := resources[key]

	if err := db.UpdateResource(key, models.ValJson{Val: "v2", Tag: existing.Tag}); err != nil {
		t.Fatalf("UpdateResource(更新 val) 失败: %v", err)
	}

	resources, err = db.GetAllResources()
	if err != nil {
		t.Fatalf("更新后 GetAllResources() 失败: %v", err)
	}
	result := resources[key]

	if result.Val != "v2" {
		t.Errorf("Val 未更新: got %q, want %q", result.Val, "v2")
	}
	if len(result.Tag) != 2 {
		t.Errorf("Tag 数量 = %d, want 2（tag 被意外清除）", len(result.Tag))
	}
	tagSet := map[string]bool{}
	for _, tag := range result.Tag {
		tagSet[tag] = true
	}
	if !tagSet["work"] || !tagSet["important"] {
		t.Errorf("Tag 内容不符: got %v, want [work important]", result.Tag)
	}
}

func TestAddWithSingleTag(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	key := models.ValJsonKey{Key: "mykey", Type: models.ORIGIN}

	if err := db.SaveResource(key, models.ValJson{Val: "some value", Tag: []string{"dev"}}); err != nil {
		t.Fatalf("SaveResource() 失败: %v", err)
	}

	resources, err := db.GetAllResources()
	if err != nil {
		t.Fatalf("GetAllResources() 失败: %v", err)
	}
	result, ok := resources[key]
	if !ok {
		t.Fatal("保存后未能查到资源")
	}
	if len(result.Tag) != 1 {
		t.Errorf("Tag 数量 = %d, want 1", len(result.Tag))
	}
	if result.Tag[0] != "dev" {
		t.Errorf("Tag[0] = %q, want dev", result.Tag[0])
	}
}

func TestAddWithMultipleTags(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	key := models.ValJsonKey{Key: "mykey", Type: models.ORIGIN}
	tags := []string{"ops", "dev", "ci"}

	if err := db.SaveResource(key, models.ValJson{Val: "some value", Tag: tags}); err != nil {
		t.Fatalf("SaveResource() 失败: %v", err)
	}

	resources, err := db.GetAllResources()
	if err != nil {
		t.Fatalf("GetAllResources() 失败: %v", err)
	}
	result, ok := resources[key]
	if !ok {
		t.Fatal("保存后未能查到资源")
	}
	if len(result.Tag) != len(tags) {
		t.Errorf("Tag 数量 = %d, want %d", len(result.Tag), len(tags))
	}

	tagSet := make(map[string]bool)
	for _, tag := range result.Tag {
		tagSet[tag] = true
	}
	for _, expectedTag := range tags {
		if !tagSet[expectedTag] {
			t.Errorf("缺少标签 %q", expectedTag)
		}
	}
}

func TestAddWithDuplicateTags(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	key := models.ValJsonKey{Key: "mykey", Type: models.ORIGIN}
	inputTags := []string{"ops", "dev", "ops", "ci", "dev"}
	dedupTags := util.RemoveDuplicates(inputTags)

	if err := db.SaveResource(key, models.ValJson{Val: "some value", Tag: dedupTags}); err != nil {
		t.Fatalf("SaveResource() 失败: %v", err)
	}

	resources, err := db.GetAllResources()
	if err != nil {
		t.Fatalf("GetAllResources() 失败: %v", err)
	}
	result, ok := resources[key]
	if !ok {
		t.Fatal("保存后未能查到资源")
	}

	tagSet := make(map[string]bool)
	for _, tag := range result.Tag {
		if tagSet[tag] {
			t.Errorf("存在重复标签: %q", tag)
		}
		tagSet[tag] = true
	}
	if len(result.Tag) != 3 {
		t.Errorf("Tag 数量 = %d, want 3 (重复标签未被去重)", len(result.Tag))
	}
}

func TestAddWithEmptyTags(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	key := models.ValJsonKey{Key: "mykey", Type: models.ORIGIN}

	if err := db.SaveResource(key, models.ValJson{Val: "some value", Tag: []string{}}); err != nil {
		t.Fatalf("SaveResource() 失败: %v", err)
	}

	resources, err := db.GetAllResources()
	if err != nil {
		t.Fatalf("GetAllResources() 失败: %v", err)
	}
	result, ok := resources[key]
	if !ok {
		t.Fatal("保存后未能查到资源")
	}
	if len(result.Tag) != 0 {
		t.Errorf("Tag 数量 = %d, want 0", len(result.Tag))
	}
}
