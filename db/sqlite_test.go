package db

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ttl-cli/models"
)

func newTempSQLiteDB(t *testing.T) func() {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "sqlite-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	dbPath := filepath.Join(tmpDir, "test.db")
	confPath := filepath.Join(tmpDir, "test.ini")
	confContent := fmt.Sprintf("[storage]\ntype = sqlite\npath = %s\n", dbPath)
	if err := os.WriteFile(confPath, []byte(confContent), 0644); err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("写入临时配置失败: %v", err)
	}

	if err := InitDB("sqlite", "", "", 0, confPath); err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("InitDB 失败: %v", err)
	}
	return func() {
		_ = CloseDB()
		_ = os.RemoveAll(tmpDir)
	}
}

func TestSQLiteInit(t *testing.T) {
	t.Run("初始化 SQLite 存储", func(t *testing.T) {
		cleanup := newTempSQLiteDB(t)
		defer cleanup()
		if Stor == nil {
			t.Error("Global storage not initialized")
		}
		if _, ok := Stor.(*SQLiteStorage); !ok {
			t.Error("Storage is not SQLiteStorage")
		}
	})
}

func TestSQLiteResources(t *testing.T) {
	t.Run("添加和获取资源", func(t *testing.T) {
		cleanup := newTempSQLiteDB(t)
		defer cleanup()

		key := models.ValJsonKey{Key: "test-key", Type: models.ORIGIN}
		value := models.ValJson{Val: "test-value", Tag: []string{"tag1", "tag2"}}

		err := SaveResource(key, value)
		if err != nil {
			t.Fatalf("SaveResource 失败: %v", err)
		}

		resources, err := GetAllResources()
		if err != nil {
			t.Fatalf("GetAllResources 失败: %v", err)
		}

		if len(resources) != 1 {
			t.Errorf("期望 1 个资源，得到 %d 个", len(resources))
		}

		retrieved, exists := resources[key]
		if !exists {
			t.Error("资源不存在")
		}

		if retrieved.Val != "test-value" {
			t.Errorf("期望值 'test-value'，得到 '%s'", retrieved.Val)
		}

		if len(retrieved.Tag) != 2 {
			t.Errorf("期望 2 个标签，得到 %d 个", len(retrieved.Tag))
		}
	})

	t.Run("更新资源", func(t *testing.T) {
		cleanup := newTempSQLiteDB(t)
		defer cleanup()

		key := models.ValJsonKey{Key: "test-key", Type: models.ORIGIN}
		value := models.ValJson{Val: "old-value", Tag: []string{"tag1"}}
		_ = SaveResource(key, value)

		newValue := models.ValJson{Val: "new-value", Tag: []string{"tag1", "tag2"}}
		err := UpdateResource(key, newValue)
		if err != nil {
			t.Fatalf("UpdateResource 失败: %v", err)
		}

		resources, _ := GetAllResources()
		retrieved := resources[key]

		if retrieved.Val != "new-value" {
			t.Errorf("期望 'new-value'，得到 '%s'", retrieved.Val)
		}

		if len(retrieved.Tag) != 2 {
			t.Errorf("期望 2 个标签，得到 %d 个", len(retrieved.Tag))
		}
	})

	t.Run("删除资源", func(t *testing.T) {
		cleanup := newTempSQLiteDB(t)
		defer cleanup()

		key := models.ValJsonKey{Key: "test-key", Type: models.ORIGIN}
		value := models.ValJson{Val: "test-value", Tag: []string{}}
		_ = SaveResource(key, value)

		err := DeleteResource(key)
		if err != nil {
			t.Fatalf("DeleteResource 失败: %v", err)
		}

		resources, _ := GetAllResources()
		if len(resources) != 0 {
			t.Error("期望 0 个资源，但资源未被删除")
		}
	})

	t.Run("添加重复资源", func(t *testing.T) {
		cleanup := newTempSQLiteDB(t)
		defer cleanup()

		key := models.ValJsonKey{Key: "test-key", Type: models.ORIGIN}
		value1 := models.ValJson{Val: "value1", Tag: []string{}}
		value2 := models.ValJson{Val: "value2", Tag: []string{}}

		_ = SaveResource(key, value1)
		_ = SaveResource(key, value2)

		resources, _ := GetAllResources()
		retrieved := resources[key]

		if retrieved.Val != "value2" {
			t.Errorf("期望被更新为 'value2'，得到 '%s'", retrieved.Val)
		}
	})
}

func TestSQLiteAudit(t *testing.T) {
	t.Run("保存和查询审计记录", func(t *testing.T) {
		cleanup := newTempSQLiteDB(t)
		defer cleanup()

		record := models.AuditRecord{
			ResourceKey: "test-key",
			Operation:   "get",
			Timestamp:   time.Now().Unix(),
			Count:       1,
		}

		err := Stor.SaveAuditRecord(record)
		if err != nil {
			t.Fatalf("SaveAuditRecord 失败: %v", err)
		}

		stats, err := Stor.GetAuditStats()
		if err != nil {
			t.Fatalf("GetAuditStats 失败: %v", err)
		}

		if stats.TotalOperations != 1 {
			t.Errorf("期望 1 次操作，得到 %d 次", stats.TotalOperations)
		}

		if stats.ByOperation["get"] != 1 {
			t.Errorf("期望 get 操作 1 次，得到 %d 次", stats.ByOperation["get"])
		}
	})

	t.Run("获取所有审计记录", func(t *testing.T) {
		cleanup := newTempSQLiteDB(t)
		defer cleanup()

		Stor.SaveAuditRecord(models.AuditRecord{ResourceKey: "key1", Operation: "get", Timestamp: 100, Count: 1})
		Stor.SaveAuditRecord(models.AuditRecord{ResourceKey: "key2", Operation: "add", Timestamp: 200, Count: 1})

		records, err := Stor.GetAllAuditRecords()
		if err != nil {
			t.Fatalf("GetAllAuditRecords 失败: %v", err)
		}

		if len(records) != 2 {
			t.Errorf("期望 2 条记录，得到 %d 条", len(records))
		}
	})

	t.Run("删除审计记录", func(t *testing.T) {
		cleanup := newTempSQLiteDB(t)
		defer cleanup()

		Stor.SaveAuditRecord(models.AuditRecord{ResourceKey: "test-key", Operation: "get", Timestamp: 100, Count: 1})
		err := Stor.DeleteAuditRecords("test-key")
		if err != nil {
			t.Fatalf("DeleteAuditRecords 失败: %v", err)
		}

		records, _ := Stor.GetAllAuditRecords()
		if len(records) != 0 {
			t.Error("期望 0 条记录，但记录未被删除")
		}
	})
}

func TestSQLiteHistory(t *testing.T) {
	t.Run("保存和查询历史记录", func(t *testing.T) {
		cleanup := newTempSQLiteDB(t)
		defer cleanup()

		record := models.HistoryRecord{
			ID:          time.Now().UnixNano(),
			ResourceKey: "test-key",
			Operation:   "get",
			Timestamp:   time.Now().Unix(),
			TimeStr:     time.Now().Format("2006-01-02 15:04:05"),
			Command:     "get",
		}

		err := Stor.SaveHistoryRecord(record)
		if err != nil {
			t.Fatalf("SaveHistoryRecord 失败: %v", err)
		}

		records, err := Stor.GetAllHistoryRecords()
		if err != nil {
			t.Fatalf("GetAllHistoryRecords 失败: %v", err)
		}

		if len(records) != 1 {
			t.Errorf("期望 1 条记录，得到 %d 条", len(records))
		}
	})

	t.Run("按索引查询历史记录", func(t *testing.T) {
		cleanup := newTempSQLiteDB(t)
		defer cleanup()

		id1 := int64(100)
		id2 := int64(200)

		Stor.SaveHistoryRecord(models.HistoryRecord{ID: id1, Operation: "cmd1", Timestamp: 100, TimeStr: "t1", Command: "cmd1"})
		Stor.SaveHistoryRecord(models.HistoryRecord{ID: id2, Operation: "cmd2", Timestamp: 200, TimeStr: "t2", Command: "cmd2"})

		record, err := Stor.GetHistoryRecord(0, models.Descending)
		if err != nil {
			t.Fatalf("GetHistoryRecord 失败: %v", err)
		}

		if record.ID != id2 {
			t.Errorf("期望 ID %d，得到 %d", id2, record.ID)
		}
	})

	t.Run("删除历史记录", func(t *testing.T) {
		cleanup := newTempSQLiteDB(t)
		defer cleanup()

		Stor.SaveHistoryRecord(models.HistoryRecord{ID: 100, ResourceKey: "test-key", Operation: "get", Timestamp: 100, TimeStr: "t1", Command: "get"})

		err := Stor.DeleteHistoryRecords("test-key")
		if err != nil {
			t.Fatalf("DeleteHistoryRecords 失败: %v", err)
		}

		records, _ := Stor.GetAllHistoryRecords()
		if len(records) != 0 {
			t.Error("期望 0 条记录，但记录未被删除")
		}
	})
}

func TestSQLiteLogs(t *testing.T) {
	t.Run("保存和查询日志记录", func(t *testing.T) {
		cleanup := newTempSQLiteDB(t)
		defer cleanup()

		now := time.Now()
		record := models.LogRecord{
			ID:        now.UnixNano(),
			Content:   "测试日志",
			Tags:      []string{"tag1", "tag2"},
			CreatedAt: now.Format("2006-01-02 15:04:05"),
			Date:      now.Format("2006-01-02"),
		}

		err := Stor.SaveLogRecord(record)
		if err != nil {
			t.Fatalf("SaveLogRecord 失败: %v", err)
		}

		records, err := Stor.GetLogRecords("", "")
		if err != nil {
			t.Fatalf("GetLogRecords 失败: %v", err)
		}

		if len(records) != 1 {
			t.Errorf("期望 1 条记录，得到 %d 条", len(records))
		}

		if len(records[0].Tags) != 2 {
			t.Errorf("期望 2 个标签，得到 %d 个", len(records[0].Tags))
		}
	})

	t.Run("按日期范围查询日志", func(t *testing.T) {
		cleanup := newTempSQLiteDB(t)
		defer cleanup()

		now := time.Now()
		yesterday := now.AddDate(0, 0, -1)

		Stor.SaveLogRecord(models.LogRecord{ID: 1, Content: "今天的日志", Date: now.Format("2006-01-02"), CreatedAt: now.Format("2006-01-02 15:04:05")})
		Stor.SaveLogRecord(models.LogRecord{ID: 2, Content: "昨天的日志", Date: yesterday.Format("2006-01-02"), CreatedAt: yesterday.Format("2006-01-02 15:04:05")})

		records, err := Stor.GetLogRecords(now.Format("2006-01-02"), now.Format("2006-01-02"))
		if err != nil {
			t.Fatalf("GetLogRecords 失败: %v", err)
		}

		if len(records) != 1 {
			t.Errorf("期望 1 条记录，得到 %d 条", len(records))
		}

		if records[0].Content != "今天的日志" {
			t.Errorf("期望 '今天的日志'，得到 '%s'", records[0].Content)
		}
	})

	t.Run("删除日志记录", func(t *testing.T) {
		cleanup := newTempSQLiteDB(t)
		defer cleanup()

		now := time.Now()
		id := now.UnixNano()
		Stor.SaveLogRecord(models.LogRecord{ID: id, Content: "测试", Date: now.Format("2006-01-02"), CreatedAt: now.Format("2006-01-02 15:04:05")})

		err := Stor.DeleteLogRecord(id)
		if err != nil {
			t.Fatalf("DeleteLogRecord 失败: %v", err)
		}

		records, _ := Stor.GetLogRecords("", "")
		if len(records) != 0 {
			t.Error("期望 0 条记录，但记录未被删除")
		}
	})
}

func TestSQLiteConcurrency(t *testing.T) {
	t.Run("并发读写", func(t *testing.T) {
		cleanup := newTempSQLiteDB(t)
		defer cleanup()

		for i := 0; i < 10; i++ {
			key := models.ValJsonKey{Key: fmt.Sprintf("key-%d", i), Type: models.ORIGIN}
			value := models.ValJson{Val: fmt.Sprintf("value-%d", i), Tag: []string{}}
			_ = SaveResource(key, value)
		}

		done := make(chan bool)
		for i := 0; i < 5; i++ {
			go func() {
				_, _ = GetAllResources()
				done <- true
			}()
		}

		for i := 10; i < 15; i++ {
			go func(i int) {
				key := models.ValJsonKey{Key: fmt.Sprintf("key-%d", i), Type: models.ORIGIN}
				value := models.ValJson{Val: fmt.Sprintf("value-%d", i), Tag: []string{}}
				_ = SaveResource(key, value)
				done <- true
			}(i)
		}

		for i := 0; i < 10; i++ {
			<-done
		}

		resources, err := GetAllResources()
		if err != nil {
			t.Fatalf("GetAllResources 失败: %v", err)
		}

		if len(resources) < 15 {
			t.Errorf("期望至少 15 个资源，得到 %d 个", len(resources))
		}
	})
}

func TestSQLiteTimestamps(t *testing.T) {
	t.Run("新建资源时设置时间戳", func(t *testing.T) {
		cleanup := newTempSQLiteDB(t)
		defer cleanup()

		beforeSave := time.Now().Unix()

		key := models.ValJsonKey{Key: "test-key", Type: models.ORIGIN}
		value := models.ValJson{Val: "test-value", Tag: []string{"tag1"}}

		err := SaveResource(key, value)
		if err != nil {
			t.Fatalf("SaveResource 失败: %v", err)
		}

		resources, err := GetAllResources()
		if err != nil {
			t.Fatalf("GetAllResources 失败: %v", err)
		}

		retrieved := resources[key]

		if retrieved.CreatedAt == 0 {
			t.Error("CreatedAt 不应为 0")
		}

		if retrieved.UpdatedAt == 0 {
			t.Error("UpdatedAt 不应为 0")
		}

		if retrieved.CreatedAt < beforeSave {
			t.Errorf("CreatedAt %d 小于保存时间 %d", retrieved.CreatedAt, beforeSave)
		}

		if retrieved.CreatedAt != retrieved.UpdatedAt {
			t.Errorf("新建资源时 CreatedAt %d 应等于 UpdatedAt %d", retrieved.CreatedAt, retrieved.UpdatedAt)
		}
	})

	t.Run("更新资源时保持 CreatedAt 不变，更新 UpdatedAt", func(t *testing.T) {
		cleanup := newTempSQLiteDB(t)
		defer cleanup()

		key := models.ValJsonKey{Key: "test-key", Type: models.ORIGIN}
		value1 := models.ValJson{Val: "value1", Tag: []string{"tag1"}}
		_ = SaveResource(key, value1)

		resources1, _ := GetAllResources()
		originalCreatedAt := resources1[key].CreatedAt

		time.Sleep(1 * time.Second)

		value2 := models.ValJson{Val: "value2", Tag: []string{"tag2"}}
		err := UpdateResource(key, value2)
		if err != nil {
			t.Fatalf("UpdateResource 失败: %v", err)
		}

		resources2, err := GetAllResources()
		if err != nil {
			t.Fatalf("GetAllResources 失败: %v", err)
		}

		retrieved := resources2[key]

		if retrieved.CreatedAt != originalCreatedAt {
			t.Errorf("CreatedAt 不应变化: 原值 %d, 新值 %d", originalCreatedAt, retrieved.CreatedAt)
		}

		if retrieved.UpdatedAt < originalCreatedAt {
			t.Errorf("UpdatedAt %d 不应小于 CreatedAt %d", retrieved.UpdatedAt, originalCreatedAt)
		}
	})

	t.Run("GetAllResources 按创建时间倒序排列", func(t *testing.T) {
		cleanup := newTempSQLiteDB(t)
		defer cleanup()

		key1 := models.ValJsonKey{Key: "key-1", Type: models.ORIGIN}
		key2 := models.ValJsonKey{Key: "key-2", Type: models.ORIGIN}
		key3 := models.ValJsonKey{Key: "key-3", Type: models.ORIGIN}

		_ = SaveResource(key1, models.ValJson{Val: "value1", Tag: []string{}})
		time.Sleep(50 * time.Millisecond)
		_ = SaveResource(key2, models.ValJson{Val: "value2", Tag: []string{}})
		time.Sleep(50 * time.Millisecond)
		_ = SaveResource(key3, models.ValJson{Val: "value3", Tag: []string{}})

		resources, err := GetAllResources()
		if err != nil {
			t.Fatalf("GetAllResources 失败: %v", err)
		}

		if len(resources) != 3 {
			t.Errorf("期望 3 个资源，得到 %d 个", len(resources))
		}

		for key, val := range resources {
			if val.CreatedAt == 0 {
				t.Errorf("资源 %s 的 CreatedAt 为 0", key.Key)
			}
			if val.UpdatedAt == 0 {
				t.Errorf("资源 %s 的 UpdatedAt 为 0", key.Key)
			}
		}

		if resources[key3].CreatedAt < resources[key2].CreatedAt {
			t.Errorf("key-3 的 CreatedAt 应大于 key-2 的 CreatedAt")
		}
		if resources[key2].CreatedAt < resources[key1].CreatedAt {
			t.Errorf("key-2 的 CreatedAt 应大于 key-1 的 CreatedAt")
		}
	})
}
