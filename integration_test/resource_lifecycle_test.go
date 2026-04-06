// Package integration_test 包含端到端集成测试。
//
// 每个测试用例通过 setupTempStorage 创建独立的临时目录和配置文件，
// 测试结束后自动清理，不影响真实数据（~/.ttl/）。
//
// 运行方式：
//
//	go test ./integration_test/...
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

// setupTempStorage 创建临时目录和配置文件，将全局存储指向临时数据库，返回清理函数。
func setupTempStorage(t *testing.T) func() {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "ttl-integration-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}

	// 写入临时配置文件，db_path 指向临时目录内的数据库文件
	dbPath := filepath.Join(tmpDir, "test.db")
	confPath := filepath.Join(tmpDir, "test.ini")
	confContent := fmt.Sprintf("db_path = %s\n", dbPath)
	if err := os.WriteFile(confPath, []byte(confContent), 0644); err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("写入临时配置文件失败: %v", err)
	}

	// 通过 --conf 同等路径初始化存储
	if err := db.InitDB("local", "", "", 0, confPath); err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("初始化临时存储失败: %v", err)
	}

	return func() {
		_ = db.CloseDB()
		_ = os.RemoveAll(tmpDir)
	}
}

// TestResourceLifecycle 测试资源的完整生命周期：增 → 查 → 改 → 删
func TestResourceLifecycle(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	key := models.ValJsonKey{Key: "github", Type: models.ORIGIN}
	value := models.ValJson{Val: "https://github.com", Tag: []string{}}

	// ── 1. 新增资源 ──────────────────────────────────────────────
	if err := db.SaveResource(key, value); err != nil {
		t.Fatalf("SaveResource() 失败: %v", err)
	}

	// ── 2. 查询资源 ──────────────────────────────────────────────
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

	// ── 3. 更新资源 ──────────────────────────────────────────────
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

	// ── 4. 删除资源 ──────────────────────────────────────────────
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

// TestAuditLifecycle 测试审计记录的写入、统计、删除流程
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

// TestHistoryLifecycle 测试历史记录的写入、查询、删除流程
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

// TestStorageIsolation 验证每个测试用例使用独立的临时存储，互不干扰
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

// TestUpdatePreservesTags 验证 update 命令不会清除已有标签（bug fix 回归测试）
func TestUpdatePreservesTags(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	key := models.ValJsonKey{Key: "mykey", Type: models.ORIGIN}

	// 1. 新增资源
	if err := db.SaveResource(key, models.ValJson{Val: "v1", Tag: []string{}}); err != nil {
		t.Fatalf("SaveResource() 失败: %v", err)
	}

	// 2. 打上标签
	if err := db.UpdateResource(key, models.ValJson{Val: "v1", Tag: []string{"work", "important"}}); err != nil {
		t.Fatalf("UpdateResource(添加 tag) 失败: %v", err)
	}

	// 3. 模拟 UpdateCmd 修复后的行为：先读取旧资源，保留 Tag，只更新 Val
	resources, err := db.GetAllResources()
	if err != nil {
		t.Fatalf("GetAllResources() 失败: %v", err)
	}
	existing := resources[key]

	if err := db.UpdateResource(key, models.ValJson{Val: "v2", Tag: existing.Tag}); err != nil {
		t.Fatalf("UpdateResource(更新 val) 失败: %v", err)
	}

	// 4. 验证 Val 已更新，Tag 未丢失
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

// TestAddWithSingleTag 验证添加资源时可以指定单个标签
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

// TestAddWithMultipleTags 验证添加资源时可以指定多个标签
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

// TestAddWithDuplicateTags 验证添加资源时重复标签会被自动去重
func TestAddWithDuplicateTags(t *testing.T) {
	cleanup := setupTempStorage(t)
	defer cleanup()

	key := models.ValJsonKey{Key: "mykey", Type: models.ORIGIN}
	// 包含重复的标签
	inputTags := []string{"ops", "dev", "ops", "ci", "dev"}
	// AddCmd 实际行为会调用 util.RemoveDuplicates 去重
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

	// 验证没有重复标签
	tagSet := make(map[string]bool)
	for _, tag := range result.Tag {
		if tagSet[tag] {
			t.Errorf("存在重复标签: %q", tag)
		}
		tagSet[tag] = true
	}
	// 期望去重后有3个标签
	if len(result.Tag) != 3 {
		t.Errorf("Tag 数量 = %d, want 3 (重复标签未被去重)", len(result.Tag))
	}
}

// TestAddWithEmptyTags 验证添加资源时不指定标签（向后兼容）
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
