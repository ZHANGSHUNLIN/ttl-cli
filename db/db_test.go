package db

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ttl-cli/models"
)

// TestGetDBPath 测试获取数据库路径
func TestGetDBPath(t *testing.T) {
	// 测试 sqlite 存储类型
	path, err := GetDBPath("", "sqlite")
	if err != nil {
		t.Fatalf("GetDBPath() error = %v", err)
	}

	if path == "" {
		t.Error("GetDBPath() returned empty path")
	}

	// 验证路径是否包含必要的目录结构
	if !filepath.IsAbs(path) {
		t.Errorf("GetDBPath() returned relative path: %s", path)
	}

	// 验证 sqlite 路径以 .db 结尾
	if filepath.Ext(path) != ".db" {
		t.Errorf("GetDBPath() for sqlite should end with .db, got: %s", path)
	}

	// 测试 local/bbolt 存储类型
	bboltPath, err := GetDBPath("", "local")
	if err != nil {
		t.Fatalf("GetDBPath() for local error = %v", err)
	}

	if filepath.Ext(bboltPath) != ".bbolt" {
		t.Errorf("GetDBPath() for local should end with .bbolt, got: %s", bboltPath)
	}
}

// TestLocalStorage_Init 测试本地存储初始化
func TestLocalStorage_Init(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "ttl-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	storage := NewLocalStorage()
	storage.dbPath = dbPath

	// 测试初始化
	err = storage.Init()
	if err != nil {
		t.Errorf("Init() error = %v", err)
	}

	// 验证数据库文件是否创建
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("Database file not created at %s", dbPath)
	}

	// 测试关闭
	err = storage.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

// TestLocalStorage_SaveGetResource 测试保存和获取资源
func TestLocalStorage_SaveGetResource(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "ttl-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	storage := NewLocalStorage()
	storage.dbPath = dbPath

	err = storage.Init()
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer storage.Close()

	// 测试数据
	key := models.ValJsonKey{
		Key:  "test-key",
		Type: models.ORIGIN,
	}

	value := models.ValJson{
		Val: "test value",
		Tag: []string{"tag1", "tag2"},
	}

	// 保存资源
	err = storage.SaveResource(key, value)
	if err != nil {
		t.Errorf("SaveResource() error = %v", err)
	}

	// 获取所有资源
	resources, err := storage.GetAllResources()
	if err != nil {
		t.Errorf("GetAllResources() error = %v", err)
	}

	// 验证资源
	savedValue, exists := resources[key]
	if !exists {
		t.Errorf("Resource not found after save")
	}

	if savedValue.Val != value.Val {
		t.Errorf("Saved value mismatch: got %q, want %q", savedValue.Val, value.Val)
	}

	if len(savedValue.Tag) != len(value.Tag) {
		t.Errorf("Saved tag count mismatch: got %d, want %d", len(savedValue.Tag), len(value.Tag))
	}
}

// TestLocalStorage_UpdateResource 测试更新资源
func TestLocalStorage_UpdateResource(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "ttl-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	storage := NewLocalStorage()
	storage.dbPath = dbPath

	err = storage.Init()
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer storage.Close()

	// 先保存资源
	key := models.ValJsonKey{
		Key:  "test-key",
		Type: models.ORIGIN,
	}

	initialValue := models.ValJson{
		Val: "initial value",
		Tag: []string{"tag1"},
	}

	err = storage.SaveResource(key, initialValue)
	if err != nil {
		t.Errorf("SaveResource() error = %v", err)
	}

	// 更新资源
	updatedValue := models.ValJson{
		Val: "updated value",
		Tag: []string{"tag2", "tag3"},
	}

	err = storage.UpdateResource(key, updatedValue)
	if err != nil {
		t.Errorf("UpdateResource() error = %v", err)
	}

	// 验证更新
	resources, err := storage.GetAllResources()
	if err != nil {
		t.Errorf("GetAllResources() error = %v", err)
	}

	savedValue, exists := resources[key]
	if !exists {
		t.Errorf("Resource not found after update")
	}

	if savedValue.Val != updatedValue.Val {
		t.Errorf("Updated value mismatch: got %q, want %q", savedValue.Val, updatedValue.Val)
	}
}

// TestLocalStorage_DeleteResource 测试删除资源
func TestLocalStorage_DeleteResource(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "ttl-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	storage := NewLocalStorage()
	storage.dbPath = dbPath

	err = storage.Init()
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer storage.Close()

	// 先保存资源
	key := models.ValJsonKey{
		Key:  "test-key",
		Type: models.ORIGIN,
	}

	value := models.ValJson{
		Val: "test value",
		Tag: []string{"tag1"},
	}

	err = storage.SaveResource(key, value)
	if err != nil {
		t.Errorf("SaveResource() error = %v", err)
	}

	// 验证资源存在
	resources, err := storage.GetAllResources()
	if err != nil {
		t.Errorf("GetAllResources() error = %v", err)
	}

	if _, exists := resources[key]; !exists {
		t.Errorf("Resource should exist before deletion")
	}

	// 删除资源
	err = storage.DeleteResource(key)
	if err != nil {
		t.Errorf("DeleteResource() error = %v", err)
	}

	// 验证资源已删除
	resources, err = storage.GetAllResources()
	if err != nil {
		t.Errorf("GetAllResources() error = %v", err)
	}

	if _, exists := resources[key]; exists {
		t.Errorf("Resource should not exist after deletion")
	}
}

// TestLocalStorage_AuditFunctions 测试审计功能
func TestLocalStorage_AuditFunctions(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "ttl-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	storage := NewLocalStorage()
	storage.dbPath = dbPath

	err = storage.Init()
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer storage.Close()

	// 创建审计记录
	record := models.AuditRecord{
		ResourceKey: "test-resource",
		Operation:   "get",
		Timestamp:   time.Now().Unix(),
		Count:       1,
	}

	// 保存审计记录
	err = storage.SaveAuditRecord(record)
	if err != nil {
		t.Errorf("SaveAuditRecord() error = %v", err)
	}

	// 获取审计统计
	stats, err := storage.GetAuditStats()
	if err != nil {
		t.Errorf("GetAuditStats() error = %v", err)
	}

	// 验证统计
	if stats.TotalOperations != 1 {
		t.Errorf("TotalOperations = %d, want 1", stats.TotalOperations)
	}

	if stats.ByOperation["get"] != 1 {
		t.Errorf("ByOperation['get'] = %d, want 1", stats.ByOperation["get"])
	}

	if stats.ByResource["test-resource"] != 1 {
		t.Errorf("ByResource['test-resource'] = %d, want 1", stats.ByResource["test-resource"])
	}

	// 测试删除审计记录
	err = storage.DeleteAuditRecords("test-resource")
	if err != nil {
		t.Errorf("DeleteAuditRecords() error = %v", err)
	}

	// 验证审计记录已删除
	stats, err = storage.GetAuditStats()
	if err != nil {
		t.Errorf("GetAuditStats() after deletion error = %v", err)
	}

	if stats.TotalOperations != 0 {
		t.Errorf("TotalOperations after deletion = %d, want 0", stats.TotalOperations)
	}
}

// TestLocalStorage_HistoryFunctions 测试历史记录功能
func TestLocalStorage_HistoryFunctions(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "ttl-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	storage := NewLocalStorage()
	storage.dbPath = dbPath

	err = storage.Init()
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer storage.Close()

	// 创建历史记录
	record1 := models.HistoryRecord{
		ID:          1,
		ResourceKey: "resource1",
		Operation:   "add",
		Timestamp:   time.Now().Unix() - 10,
		TimeStr:     time.Now().Format("2006-01-02 15:04:05"),
		Command:     "ttl add resource1 value1",
	}

	record2 := models.HistoryRecord{
		ID:          2,
		ResourceKey: "resource2",
		Operation:   "get",
		Timestamp:   time.Now().Unix(),
		TimeStr:     time.Now().Format("2006-01-02 15:04:05"),
		Command:     "ttl get resource2",
	}

	// 保存历史记录
	err = storage.SaveHistoryRecord(record1)
	if err != nil {
		t.Errorf("SaveHistoryRecord(record1) error = %v", err)
	}

	err = storage.SaveHistoryRecord(record2)
	if err != nil {
		t.Errorf("SaveHistoryRecord(record2) error = %v", err)
	}

	// 获取所有历史记录
	records, err := storage.GetAllHistoryRecords()
	if err != nil {
		t.Errorf("GetAllHistoryRecords() error = %v", err)
	}

	// 验证记录数量
	if len(records) != 2 {
		t.Errorf("Got %d history records, want 2", len(records))
	}

	// 验证排序（应该按时间倒序）
	if records[0].Timestamp < records[1].Timestamp {
		t.Errorf("Records not sorted in descending order")
	}

	// 获取历史统计
	stats, err := storage.GetHistoryStats()
	if err != nil {
		t.Errorf("GetHistoryStats() error = %v", err)
	}

	if stats.TotalRecords != 2 {
		t.Errorf("TotalRecords = %d, want 2", stats.TotalRecords)
	}

	if stats.ByOperation["add"] != 1 {
		t.Errorf("ByOperation['add'] = %d, want 1", stats.ByOperation["add"])
	}

	// 测试删除历史记录
	err = storage.DeleteHistoryRecords("resource1")
	if err != nil {
		t.Errorf("DeleteHistoryRecords() error = %v", err)
	}

	// 验证历史记录已删除
	records, err = storage.GetAllHistoryRecords()
	if err != nil {
		t.Errorf("GetAllHistoryRecords() after deletion error = %v", err)
	}

	if len(records) != 1 {
		t.Errorf("Got %d history records after deletion, want 1", len(records))
	}

	if records[0].ResourceKey == "resource1" {
		t.Errorf("Resource1 should have been deleted")
	}
}

// TestJsonSerialization 测试JSON序列化
func TestJsonSerialization(t *testing.T) {
	// 测试 ValJsonKey 序列化
	key := models.ValJsonKey{
		Key:       "test-key",
		Type:      models.ORIGIN,
		OriginKey: "test-origin",
	}

	keyJson, err := json.Marshal(key)
	if err != nil {
		t.Errorf("Failed to marshal ValJsonKey: %v", err)
	}

	var keyDecoded models.ValJsonKey
	err = json.Unmarshal(keyJson, &keyDecoded)
	if err != nil {
		t.Errorf("Failed to unmarshal ValJsonKey: %v", err)
	}

	if keyDecoded.Key != key.Key {
		t.Errorf("ValJsonKey.Key mismatch: got %q, want %q", keyDecoded.Key, key.Key)
	}

	// 测试 ValJson 序列化
	value := models.ValJson{
		Val: "test value",
		Tag: []string{"tag1", "tag2"},
	}

	valueJson, err := json.Marshal(value)
	if err != nil {
		t.Errorf("Failed to marshal ValJson: %v", err)
	}

	var valueDecoded models.ValJson
	err = json.Unmarshal(valueJson, &valueDecoded)
	if err != nil {
		t.Errorf("Failed to unmarshal ValJson: %v", err)
	}

	if valueDecoded.Val != value.Val {
		t.Errorf("ValJson.Val mismatch: got %q, want %q", valueDecoded.Val, value.Val)
	}
}

// TestCloudStorage 测试云端存储（使用 mock server）
func TestCloudStorage(t *testing.T) {
	srv := mockAPIServer(t)
	defer srv.Close()

	storage := NewCloudStorage(srv.URL, "test-key", 30)

	// 测试初始化
	err := storage.Init()
	if err != nil {
		t.Errorf("CloudStorage.Init() error = %v", err)
	}

	// 测试获取资源（应该为空）
	resources, err := storage.GetAllResources()
	if err != nil {
		t.Errorf("CloudStorage.GetAllResources() error = %v", err)
	}

	if len(resources) != 0 {
		t.Errorf("Expected empty resources from cloud storage")
	}

	// 测试审计函数（空操作，不应报错）
	record := models.AuditRecord{
		ResourceKey: "test",
		Operation:   "get",
		Timestamp:   time.Now().Unix(),
		Count:       1,
	}

	err = storage.SaveAuditRecord(record)
	if err != nil {
		t.Errorf("CloudStorage.SaveAuditRecord() error = %v", err)
	}

	// 测试关闭
	err = storage.Close()
	if err != nil {
		t.Errorf("CloudStorage.Close() error = %v", err)
	}
}
