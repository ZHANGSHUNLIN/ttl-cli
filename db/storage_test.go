package db

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"ttl-cli/models"
)

func newTempInitDB(t *testing.T) func() {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "stor-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	dbPath := filepath.Join(tmpDir, "test.db")
	confPath := filepath.Join(tmpDir, "test.ini")
	if err := os.WriteFile(confPath, []byte(fmt.Sprintf("db_path = %s\n", dbPath)), 0644); err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("写入临时配置失败: %v", err)
	}
	if err := InitDB("local", "", "", 0, confPath); err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("InitDB 失败: %v", err)
	}
	return func() {
		_ = CloseDB()
		_ = os.RemoveAll(tmpDir)
	}
}

func TestInitDB(t *testing.T) {
	t.Run("初始化本地存储", func(t *testing.T) {
		cleanup := newTempInitDB(t)
		defer cleanup()
		if Stor == nil {
			t.Error("Global storage not initialized")
		}
	})

	t.Run("初始化云存储失败-缺少参数", func(t *testing.T) {
		err := InitDB("cloud", "", "", 0, "")
		if err == nil {
			t.Error("Expected error but got none")
		}
	})

	t.Run("不支持的存储类型", func(t *testing.T) {
		err := InitDB("invalid", "", "", 0, "")
		if err == nil {
			t.Error("Expected error but got none")
		}
	})
}

func TestCloseDB(t *testing.T) {
	t.Run("关闭本地存储", func(t *testing.T) {
		cleanup := newTempInitDB(t)
		defer func() {
			_ = os.RemoveAll("")
		}()
		_ = cleanup
		cleanup()
	})

	t.Run("关闭空存储", func(t *testing.T) {
		old := Stor
		Stor = nil
		defer func() { Stor = old }()
		if err := CloseDB(); err != nil {
			t.Errorf("CloseDB() on nil error = %v", err)
		}
	})
}

func TestGetAllResources(t *testing.T) {
	t.Run("获取本地存储资源", func(t *testing.T) {
		cleanup := newTempInitDB(t)
		defer cleanup()
		resources, err := GetAllResources()
		if err != nil {
			t.Errorf("GetAllResources() error = %v", err)
		}
		if resources == nil {
			t.Error("GetAllResources() returned nil map")
		}
	})

	t.Run("获取未初始化存储资源", func(t *testing.T) {
		old := Stor
		Stor = nil
		defer func() { Stor = old }()
		_, err := GetAllResources()
		if err == nil {
			t.Error("Expected error but got none")
		}
	})
}

func TestRecordAudit(t *testing.T) {
	t.Run("本地存储记录审计", func(t *testing.T) {
		cleanup := newTempInitDB(t)
		defer cleanup()
		if err := RecordAudit("test-resource", "get"); err != nil {
			t.Errorf("RecordAudit() error = %v", err)
		}
	})

	t.Run("未初始化存储记录审计", func(t *testing.T) {
		old := Stor
		Stor = nil
		defer func() { Stor = old }()
		if err := RecordAudit("test-resource", "get"); err == nil {
			t.Error("Expected error but got none")
		}
	})
}

func TestGetAuditStats(t *testing.T) {
	cleanup := newTempInitDB(t)
	defer cleanup()
	stats, err := GetAuditStats()
	if err != nil {
		t.Errorf("GetAuditStats() error = %v", err)
	}
	if stats.ByOperation == nil {
		t.Error("GetAuditStats() returned nil ByOperation map")
	}
	if stats.ByResource == nil {
		t.Error("GetAuditStats() returned nil ByResource map")
	}
}

func TestMigrateData(t *testing.T) {
	srcDir, err := os.MkdirTemp("", "migrate-src-*")
	if err != nil {
		t.Fatalf("创建 src 临时目录失败: %v", err)
	}
	defer os.RemoveAll(srcDir)

	dstDir, err := os.MkdirTemp("", "migrate-dst-*")
	if err != nil {
		t.Fatalf("创建 dst 临时目录失败: %v", err)
	}
	defer os.RemoveAll(dstDir)

	writeConf := func(dir string) string {
		p := filepath.Join(dir, "test.ini")
		content := fmt.Sprintf("db_path = %s\n", filepath.Join(dir, "test.db"))
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			t.Fatalf("写入配置失败: %v", err)
		}
		return p
	}

	err = MigrateData("local", "local", "", "", 30, "", "", 30, false,
		writeConf(srcDir), writeConf(dstDir))
	if err != nil {
		t.Errorf("MigrateData() error = %v", err)
	}
}

func TestGetHistoryRecords(t *testing.T) {
	cleanup := newTempInitDB(t)
	defer cleanup()
	_, err := GetHistoryRecords(0)
	if err == nil {
		t.Log("索引0不存在时返回 nil error（空库），可接受")
	}
}

func TestCleanupResourceHistory(t *testing.T) {
	cleanup := newTempInitDB(t)
	defer cleanup()
	CleanupResourceHistory("test-resource", false)
}

func TestInterfaceMethods(t *testing.T) {
	localStorage := NewLocalStorage()
	cloudStorage := NewCloudStorage("https://api.example.com", "test-key", 30)

	var storages []Storage
	storages = append(storages, localStorage)
	storages = append(storages, cloudStorage)

	for i, storage := range storages {
		if storage == nil {
			t.Errorf("Storage %d is nil", i)
		}
	}
}

func TestValJsonKeyComparison(t *testing.T) {
	key1 := models.ValJsonKey{Key: "test", Type: models.ORIGIN}
	key2 := models.ValJsonKey{Key: "test", Type: models.ORIGIN}
	key3 := models.ValJsonKey{Key: "test", Type: models.TAG}

	if key1.Key != key2.Key || key1.Type != key2.Type {
		t.Errorf("Key comparison failed: %+v != %+v", key1, key2)
	}
	if key1.Key == key3.Key && key1.Type == key3.Type {
		t.Errorf("Key comparison should show difference: %+v vs %+v", key1, key3)
	}
}
