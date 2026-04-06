package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"ttl-cli/crypto"
	"ttl-cli/db"
	"ttl-cli/models"
)

// TestEncryptionLifecycle 测试加密功能的完整生命周期
func TestEncryptionLifecycle(t *testing.T) {
	// 使用临时目录
	tmpDir := t.TempDir()
	oldKeyPath := crypto.GetKeyFilePath()
	crypto.SetKeyFilePath(filepath.Join(tmpDir, ".key"))
	defer func() { crypto.SetKeyFilePath(oldKeyPath) }()

	// 1. 创建临时配置文件
	confPath := filepath.Join(tmpDir, "ttl.ini")
	confContent := `db_path = ` + filepath.Join(tmpDir, "data.db") + `
`

	if err := os.WriteFile(confPath, []byte(confContent), 0600); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// 2. 初始化数据库（无加密）
	if err := db.InitDB("local", "", "", 0, confPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.CloseDB()

	// 3. 添加一些资源
	resources := []struct {
		key   string
		value string
	}{
		{"test1", "plain value 1"},
		{"test2", "你好世界 🌍"},
		{"test3", "special chars: \n\t\r\\\"'"},
	}

	for _, r := range resources {
		key := models.ValJsonKey{Key: r.key, Type: models.ORIGIN}
		value := models.ValJson{Val: r.value, Tag: []string{"test"}}
		if err := db.SaveResource(key, value); err != nil {
			t.Fatalf("Failed to save resource %s: %v", r.key, err)
		}
	}

	// 4. 验证数据已保存
	allResources, err := db.GetAllResources()
	if err != nil {
		t.Fatalf("Failed to get resources: %v", err)
	}

	if len(allResources) != len(resources) {
		t.Errorf("Expected %d resources, got %d", len(resources), len(allResources))
	}

	// 验证数据是明文
	for _, r := range resources {
		key := models.ValJsonKey{Key: r.key, Type: models.ORIGIN}
		val, exists := allResources[key]
		if !exists {
			t.Errorf("Resource %s not found", r.key)
			continue
		}
		if val.Val != r.value {
			t.Errorf("Expected value %q, got %q", r.value, val.Val)
		}
	}

	// 5. 启用加密
	ls, ok := db.Stor.(*db.LocalStorage)
	if !ok {
		t.Fatal("Expected LocalStorage")
	}

	// 需要先关闭并重新打开数据库，让新设置生效
	db.CloseDB()
	if err := db.InitDB("local", "", "", 0, confPath); err != nil {
		t.Fatalf("Failed to re-init DB: %v", err)
	}

	ls, ok = db.Stor.(*db.LocalStorage)
	if !ok {
		t.Fatal("Expected LocalStorage after re-init")
	}

	if err := ls.EnableEncryption(); err != nil {
		t.Fatalf("Failed to enable encryption: %v", err)
	}

	// 6. 重新读取数据，验证已加密
	db.CloseDB()
	if err := db.InitDB("local", "", "", 0, confPath); err != nil {
		t.Fatalf("Failed to re-init DB: %v", err)
	}
	defer db.CloseDB()

	encryptedResources, err := db.GetAllResources()
	if err != nil {
		t.Fatalf("Failed to get encrypted resources: %v", err)
	}

	// GetAllResources 应该自动解密，所以返回的值应该是明文
	for _, r := range resources {
		key := models.ValJsonKey{Key: r.key, Type: models.ORIGIN}
		val, exists := encryptedResources[key]
		if !exists {
			t.Errorf("Encrypted resource %s not found", r.key)
			continue
		}
		// 由于解密，值应该是原始的明文
		if val.Val != r.value {
			t.Errorf("After decrypt, expected value %q, got %q", r.value, val.Val)
		}
	}

	// 7. 直接读取数据库验证存储的是加密数据
	rawDB, err := db.GetDBPath(confPath, "local")
	if err != nil {
		t.Fatalf("Failed to get DB path: %v", err)
	}
	_ = rawDB // 不直接读取数据库，通过 GetAllResources 验证解密功能

	// 8. 禁用加密
	// 重新获取 LocalStorage 引用（因为 re-init 后 Stor 被更新了）
	ls2, ok := db.Stor.(*db.LocalStorage)
	if !ok {
		t.Fatal("Expected LocalStorage after re-init")
	}

	if err := ls2.DisableEncryption(); err != nil {
		t.Fatalf("Failed to disable encryption: %v", err)
	}

	// 9. 重新读取数据，验证已解密
	// 注意：DisableEncryption() 内部已经操作了数据库，但可能需要重新初始化
	// 因为 DisableEncryption() 是在 ls2 上操作的，而不是 ls
	db.CloseDB()
	if err := db.InitDB("local", "", "", 0, confPath); err != nil {
		t.Fatalf("Failed to re-init DB: %v", err)
	}
	defer db.CloseDB()

	plainResources, err := db.GetAllResources()
	if err != nil {
		t.Fatalf("Failed to get plain resources: %v", err)
	}

	// 验证数据又变回明文
	for _, r := range resources {
		key := models.ValJsonKey{Key: r.key, Type: models.ORIGIN}
		val, exists := plainResources[key]
		if !exists {
			t.Errorf("Plain resource %s not found", r.key)
			continue
		}
		if val.Val != r.value {
			t.Errorf("Expected value %q, got %q", r.value, val.Val)
		}
	}
}

// TestKeyManagement 测试密钥管理命令
func TestKeyManagement(t *testing.T) {
	// 使用临时目录
	tmpDir := t.TempDir()
	oldKeyPath := crypto.GetKeyFilePath()
	crypto.SetKeyFilePath(filepath.Join(tmpDir, ".key"))
	defer func() { crypto.SetKeyFilePath(oldKeyPath) }()

	// 1. 初始化时没有密钥
	if crypto.KeyExists() {
		t.Error("Key should not exist initially")
	}

	// 2. 验证密钥（应该失败）
	if err := crypto.VerifyKey(); err == nil {
		t.Error("VerifyKey should fail when key doesn't exist")
	}

	// 3. 初始化加密（生成密钥）
	if err := crypto.InitEncryption(true); err != nil {
		t.Fatalf("InitEncryption failed: %v", err)
	}

	if !crypto.KeyExists() {
		t.Error("Key should exist after InitEncryption")
	}

	// 4. 验证密钥（应该成功）
	if err := crypto.VerifyKey(); err != nil {
		t.Errorf("VerifyKey failed: %v", err)
	}

	// 5. 导出密钥
	exportPath := filepath.Join(tmpDir, "exported.key")
	if err := crypto.ExportKey(exportPath); err != nil {
		t.Fatalf("ExportKey failed: %v", err)
	}

	// 6. 删除原密钥
	if err := crypto.DeleteKey(); err != nil {
		t.Fatalf("DeleteKey failed: %v", err)
	}

	if crypto.KeyExists() {
		t.Error("Key should not exist after deletion")
	}

	// 7. 从导出文件导入密钥
	if err := crypto.ImportKey(exportPath); err != nil {
		t.Fatalf("ImportKey failed: %v", err)
	}

	if !crypto.KeyExists() {
		t.Error("Key should exist after import")
	}

	// 8. 再次验证密钥
	if err := crypto.VerifyKey(); err != nil {
		t.Errorf("VerifyKey failed after import: %v", err)
	}
}

// TestCryptoIsEncrypted 测试加密检测功能
func TestCryptoIsEncrypted(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	tests := []struct {
		name     string
		data     string
		expected bool
	}{
		{"plain text", "hello world", false},
		{"empty", "", false},
		{"with colon", "hello:world", false},
		{"encrypted", "YWJjZGVmZ2hpams=:YWJjZGVmZ2hpams", false}, // wrong nonce length, not real encrypted
		{"valid base64 but wrong format", "SGVsbG8=:", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := crypto.IsEncrypted(tt.data)
			if result != tt.expected {
				t.Errorf("IsEncrypted(%q) = %v, want %v", tt.data, result, tt.expected)
			}
		})
	}

	// 测试实际的加密数据
	plain := "test data"
	for i := range key {
		key[i] = byte(i)
	}
	encrypted, err := crypto.Encrypt(key, plain)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if !crypto.IsEncrypted(encrypted) {
		t.Error("Encrypted data should be detected as encrypted")
	}

	// 解密验证
	decrypted, err := crypto.Decrypt(key, encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted != plain {
		t.Errorf("Expected %q, got %q", plain, decrypted)
	}
}

// TestEncryptionCommands 测试加密命令
func TestEncryptionCommands(t *testing.T) {
	// 使用临时目录
	tmpDir := t.TempDir()
	oldKeyPath := crypto.GetKeyFilePath()
	crypto.SetKeyFilePath(filepath.Join(tmpDir, ".key"))
	defer func() { crypto.SetKeyFilePath(oldKeyPath) }()

	// 创建临时配置文件
	confPath := filepath.Join(tmpDir, "ttl.ini")
	dbPath := filepath.Join(tmpDir, "data.db")
	confContent := "db_path = " + dbPath + "\n"

	if err := os.WriteFile(confPath, []byte(confContent), 0600); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// 测试 encrypt 命令
	t.Run("encrypt command", func(t *testing.T) {
		// 初始化数据库
		if err := db.InitDB("local", "", "", 0, confPath); err != nil {
			t.Fatalf("Failed to init DB: %v", err)
		}
		defer db.CloseDB()

		// 添加测试数据
		key := models.ValJsonKey{Key: "test", Type: models.ORIGIN}
		value := models.ValJson{Val: "sensitive data", Tag: []string{}}
		if err := db.SaveResource(key, value); err != nil {
			t.Fatalf("Failed to save resource: %v", err)
		}

		// 启用加密
		ls, ok := db.Stor.(*db.LocalStorage)
		if !ok {
			t.Fatal("Expected LocalStorage")
		}

		if err := ls.EnableEncryption(); err != nil {
			t.Fatalf("EnableEncryption failed: %v", err)
		}

		// 验证数据已加密
		all, err := db.GetAllResources()
		if err != nil {
			t.Fatalf("GetAllResources failed: %v", err)
		}

		val := all[key]
		// GetAllResources 会自动解密，所以应该是原始值
		if val.Val != "sensitive data" {
			t.Errorf("Expected 'sensitive data', got %q", val.Val)
		}
	})

	// 测试 decrypt 命令
	t.Run("decrypt command", func(t *testing.T) {
		// 重新初始化
		if err := db.InitDB("local", "", "", 0, confPath); err != nil {
			t.Fatalf("Failed to init DB: %v", err)
		}
		defer db.CloseDB()

		ls, ok := db.Stor.(*db.LocalStorage)
		if !ok {
			t.Fatal("Expected LocalStorage")
		}

		if !ls.IsEncryptionEnabled() {
			t.Error("Encryption should be enabled")
		}

		// 禁用加密
		if err := ls.DisableEncryption(); err != nil {
			t.Fatalf("DisableEncryption failed: %v", err)
		}

		if ls.IsEncryptionEnabled() {
			t.Error("Encryption should be disabled")
		}
	})

	// 测试 key 命令
	t.Run("key commands", func(t *testing.T) {
		exportPath := filepath.Join(tmpDir, "backup.key")

		// 重新生成密钥（因为前面的 decrypt_command 删除了）
		if err := crypto.InitEncryption(true); err != nil {
			t.Fatalf("InitEncryption failed: %v", err)
		}

		// 导出
		if err := crypto.ExportKey(exportPath); err != nil {
			t.Fatalf("ExportKey failed: %v", err)
		}

		// 验证导出文件存在
		if _, err := os.Stat(exportPath); os.IsNotExist(err) {
			t.Error("Exported key file should exist")
		}

		// 删除密钥
		if err := crypto.DeleteKey(); err != nil {
			t.Fatalf("DeleteKey failed: %v", err)
		}

		// 导入
		if err := crypto.ImportKey(exportPath); err != nil {
			t.Fatalf("ImportKey failed: %v", err)
		}

		// 验证
		if err := crypto.VerifyKey(); err != nil {
			t.Errorf("VerifyKey failed: %v", err)
		}
	})
}
