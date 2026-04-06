package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"ttl-cli/crypto"
	"ttl-cli/db"
	"ttl-cli/models"
)

func TestEncryptionLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	oldKeyPath := crypto.GetKeyFilePath()
	crypto.SetKeyFilePath(filepath.Join(tmpDir, ".key"))
	defer func() { crypto.SetKeyFilePath(oldKeyPath) }()

	confPath := filepath.Join(tmpDir, "ttl.ini")
	confContent := `db_path = ` + filepath.Join(tmpDir, "data.db") + `
`

	if err := os.WriteFile(confPath, []byte(confContent), 0600); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	if err := db.InitDB("local", "", "", 0, confPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.CloseDB()

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

	allResources, err := db.GetAllResources()
	if err != nil {
		t.Fatalf("Failed to get resources: %v", err)
	}

	if len(allResources) != len(resources) {
		t.Errorf("Expected %d resources, got %d", len(resources), len(allResources))
	}

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

	ls, ok := db.Stor.(*db.LocalStorage)
	if !ok {
		t.Fatal("Expected LocalStorage")
	}

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

	db.CloseDB()
	if err := db.InitDB("local", "", "", 0, confPath); err != nil {
		t.Fatalf("Failed to re-init DB: %v", err)
	}
	defer db.CloseDB()

	encryptedResources, err := db.GetAllResources()
	if err != nil {
		t.Fatalf("Failed to get encrypted resources: %v", err)
	}

	for _, r := range resources {
		key := models.ValJsonKey{Key: r.key, Type: models.ORIGIN}
		val, exists := encryptedResources[key]
		if !exists {
			t.Errorf("Encrypted resource %s not found", r.key)
			continue
		}
		if val.Val != r.value {
			t.Errorf("After decrypt, expected value %q, got %q", r.value, val.Val)
		}
	}

	rawDB, err := db.GetDBPath(confPath, "local")
	if err != nil {
		t.Fatalf("Failed to get DB path: %v", err)
	}
	_ = rawDB

	ls2, ok := db.Stor.(*db.LocalStorage)
	if !ok {
		t.Fatal("Expected LocalStorage after re-init")
	}

	if err := ls2.DisableEncryption(); err != nil {
		t.Fatalf("Failed to disable encryption: %v", err)
	}

	db.CloseDB()
	if err := db.InitDB("local", "", "", 0, confPath); err != nil {
		t.Fatalf("Failed to re-init DB: %v", err)
	}
	defer db.CloseDB()

	plainResources, err := db.GetAllResources()
	if err != nil {
		t.Fatalf("Failed to get plain resources: %v", err)
	}

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

func TestKeyManagement(t *testing.T) {
	tmpDir := t.TempDir()
	oldKeyPath := crypto.GetKeyFilePath()
	crypto.SetKeyFilePath(filepath.Join(tmpDir, ".key"))
	defer func() { crypto.SetKeyFilePath(oldKeyPath) }()

	if crypto.KeyExists() {
		t.Error("Key should not exist initially")
	}

	if err := crypto.VerifyKey(); err == nil {
		t.Error("VerifyKey should fail when key doesn't exist")
	}

	if err := crypto.InitEncryption(true); err != nil {
		t.Fatalf("InitEncryption failed: %v", err)
	}

	if !crypto.KeyExists() {
		t.Error("Key should exist after InitEncryption")
	}

	if err := crypto.VerifyKey(); err != nil {
		t.Errorf("VerifyKey failed: %v", err)
	}

	exportPath := filepath.Join(tmpDir, "exported.key")
	if err := crypto.ExportKey(exportPath); err != nil {
		t.Fatalf("ExportKey failed: %v", err)
	}

	if err := crypto.DeleteKey(); err != nil {
		t.Fatalf("DeleteKey failed: %v", err)
	}

	if crypto.KeyExists() {
		t.Error("Key should not exist after deletion")
	}

	if err := crypto.ImportKey(exportPath); err != nil {
		t.Fatalf("ImportKey failed: %v", err)
	}

	if !crypto.KeyExists() {
		t.Error("Key should exist after import")
	}

	if err := crypto.VerifyKey(); err != nil {
		t.Errorf("VerifyKey failed after import: %v", err)
	}
}

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
		{"encrypted", "YWJjZGVmZ2hpams=:YWJjZGVmZ2hpams", false},
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

	decrypted, err := crypto.Decrypt(key, encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted != plain {
		t.Errorf("Expected %q, got %q", plain, decrypted)
	}
}

func TestEncryptionCommands(t *testing.T) {
	tmpDir := t.TempDir()
	oldKeyPath := crypto.GetKeyFilePath()
	crypto.SetKeyFilePath(filepath.Join(tmpDir, ".key"))
	defer func() { crypto.SetKeyFilePath(oldKeyPath) }()

	confPath := filepath.Join(tmpDir, "ttl.ini")
	dbPath := filepath.Join(tmpDir, "data.db")
	confContent := "db_path = " + dbPath + "\n"

	if err := os.WriteFile(confPath, []byte(confContent), 0600); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	t.Run("encrypt command", func(t *testing.T) {
		if err := db.InitDB("local", "", "", 0, confPath); err != nil {
			t.Fatalf("Failed to init DB: %v", err)
		}
		defer db.CloseDB()

		key := models.ValJsonKey{Key: "test", Type: models.ORIGIN}
		value := models.ValJson{Val: "sensitive data", Tag: []string{}}
		if err := db.SaveResource(key, value); err != nil {
			t.Fatalf("Failed to save resource: %v", err)
		}

		ls, ok := db.Stor.(*db.LocalStorage)
		if !ok {
			t.Fatal("Expected LocalStorage")
		}

		if err := ls.EnableEncryption(); err != nil {
			t.Fatalf("EnableEncryption failed: %v", err)
		}

		all, err := db.GetAllResources()
		if err != nil {
			t.Fatalf("GetAllResources failed: %v", err)
		}

		val := all[key]
		if val.Val != "sensitive data" {
			t.Errorf("Expected 'sensitive data', got %q", val.Val)
		}
	})

	t.Run("decrypt command", func(t *testing.T) {
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

		if err := ls.DisableEncryption(); err != nil {
			t.Fatalf("DisableEncryption failed: %v", err)
		}

		if ls.IsEncryptionEnabled() {
			t.Error("Encryption should be disabled")
		}
	})

	t.Run("key commands", func(t *testing.T) {
		exportPath := filepath.Join(tmpDir, "backup.key")

		if err := crypto.InitEncryption(true); err != nil {
			t.Fatalf("InitEncryption failed: %v", err)
		}

		if err := crypto.ExportKey(exportPath); err != nil {
			t.Fatalf("ExportKey failed: %v", err)
		}

		if _, err := os.Stat(exportPath); os.IsNotExist(err) {
			t.Error("Exported key file should exist")
		}

		if err := crypto.DeleteKey(); err != nil {
			t.Fatalf("DeleteKey failed: %v", err)
		}

		if err := crypto.ImportKey(exportPath); err != nil {
			t.Fatalf("ImportKey failed: %v", err)
		}

		if err := crypto.VerifyKey(); err != nil {
			t.Errorf("VerifyKey failed: %v", err)
		}
	})
}
