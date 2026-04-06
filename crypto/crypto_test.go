package crypto

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	key := make([]byte, KeySize)
	for i := range key {
		key[i] = byte(i)
	}

	tests := []struct {
		name string
		data string
	}{
		{"simple", "hello world"},
		{"special chars", "test\n\t\r\\\"'"},
		{"unicode", "你好世界 🌍"},
		{"empty", ""},
		{"long", string(make([]byte, 1000))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := Encrypt(key, tt.data)
			if err != nil {
				t.Fatalf("Encrypt failed: %v", err)
			}

			decrypted, err := Decrypt(key, encrypted)
			if err != nil {
				t.Fatalf("Decrypt failed: %v", err)
			}

			if decrypted != tt.data {
				t.Errorf("Decrypted data mismatch: got %q, want %q", decrypted, tt.data)
			}
		})
	}
}

func TestEncryptInvalidKey(t *testing.T) {
	invalidKeys := [][]byte{
		nil,
		{},
		make([]byte, 16),
		make([]byte, 64),
	}

	for _, key := range invalidKeys {
		t.Run("", func(t *testing.T) {
			_, err := Encrypt(key, "test")
			if err == nil {
				t.Error("Expected error with invalid key size")
			}
		})
	}
}

func TestDecryptInvalidFormat(t *testing.T) {
	key := make([]byte, KeySize)

	invalidData := []string{
		"",
		"invalid",
		"invalid:format",
		"a:b:c",
	}

	for _, data := range invalidData {
		t.Run(data, func(t *testing.T) {
			_, err := Decrypt(key, data)
			if err == nil {
				t.Error("Expected error with invalid format")
			}
		})
	}
}

func TestIsEncrypted(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		expected bool
	}{
		{"plain", "hello world", false},
		{"simple encrypted", "MTIzNDU2Nzg5MDEy:WAAAAAAAAAAAAAAAAAAAAAA=", true},
		{"invalid format", "not:valid:here", false},
		{"with special chars", "test\\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsEncrypted(tt.data)
			if result != tt.expected {
				t.Errorf("IsEncrypted(%q) = %v, want %v", tt.data, result, tt.expected)
			}
		})
	}
}

func TestKeyManagement(t *testing.T) {
	tmpDir := t.TempDir()
	oldKeyPath := keyFilePath
	keyFilePath = filepath.Join(tmpDir, KeyFileName)
	defer func() { keyFilePath = oldKeyPath }()

	exists := KeyExists()
	if exists {
		t.Error("Expected key file to not exist")
	}

	_, err := LoadKey()
	if err != nil {
		t.Errorf("LoadKey on non-existent file should return nil, got error: %v", err)
	}

	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	if len(key) != KeySize {
		t.Errorf("Generated key has wrong size: got %d, want %d", len(key), KeySize)
	}

	err = SaveKey(key)
	if err != nil {
		t.Fatalf("SaveKey failed: %v", err)
	}

	exists = KeyExists()
	if !exists {
		t.Error("Expected key file to exist")
	}

	loadedKey, err := LoadKey()
	if err != nil {
		t.Fatalf("LoadKey failed: %v", err)
	}

	if len(loadedKey) != KeySize {
		t.Errorf("Loaded key has wrong size: got %d, want %d", len(loadedKey), KeySize)
	}

	testData := "test-data-123"
	encrypted, err := Encrypt(loadedKey, testData)
	if err != nil {
		t.Fatalf("Encrypt with loaded key failed: %v", err)
	}

	decrypted, err := Decrypt(loadedKey, encrypted)
	if err != nil {
		t.Fatalf("Decrypt with loaded key failed: %v", err)
	}

	if decrypted != testData {
		t.Errorf("Decrypted data mismatch: got %q, want %q", decrypted, testData)
	}

	err = VerifyKey()
	if err != nil {
		t.Errorf("VerifyKey failed: %v", err)
	}

	err = DeleteKey()
	if err != nil {
		t.Fatalf("DeleteKey failed: %v", err)
	}

	exists = KeyExists()
	if exists {
		t.Error("Expected key file to be deleted")
	}
}

func TestExportImportKey(t *testing.T) {
	tmpDir := t.TempDir()
	oldKeyPath := keyFilePath
	keyFilePath = filepath.Join(tmpDir, KeyFileName)
	defer func() { keyFilePath = oldKeyPath }()

	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	err = SaveKey(key)
	if err != nil {
		t.Fatalf("SaveKey failed: %v", err)
	}

	exportPath := filepath.Join(tmpDir, "exported.key")
	err = ExportKey(exportPath)
	if err != nil {
		t.Fatalf("ExportKey failed: %v", err)
	}

	err = DeleteKey()
	if err != nil {
		t.Fatalf("DeleteKey failed: %v", err)
	}

	err = ImportKey(exportPath)
	if err != nil {
		t.Fatalf("ImportKey failed: %v", err)
	}

	loadedKey, err := LoadKey()
	if err != nil {
		t.Fatalf("LoadKey failed: %v", err)
	}

	testData := "export-import-test"
	encrypted, err := Encrypt(loadedKey, testData)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := Decrypt(loadedKey, encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted != testData {
		t.Errorf("Decrypted data mismatch after import: got %q, want %q", decrypted, testData)
	}
}

func TestInitEncryption(t *testing.T) {
	tmpDir := t.TempDir()
	oldKeyPath := keyFilePath
	keyFilePath = filepath.Join(tmpDir, KeyFileName)
	defer func() { keyFilePath = oldKeyPath }()

	err := InitEncryption(false)
	if err != nil {
		t.Fatalf("InitEncryption failed: %v", err)
	}

	if KeyExists() {
		t.Error("Key should not be created when createIfMissing is false")
	}

	err = InitEncryption(true)
	if err != nil {
		t.Fatalf("InitEncryption failed: %v", err)
	}

	if !KeyExists() {
		t.Error("Key should be created when createIfMissing is true")
	}

	err = InitEncryption(true)
	if err != nil {
		t.Fatalf("InitEncryption failed with existing key: %v", err)
	}
}

func TestInvalidKeyFile(t *testing.T) {
	tmpDir := t.TempDir()
	oldKeyPath := keyFilePath
	keyFilePath = filepath.Join(tmpDir, KeyFileName)
	defer func() { keyFilePath = oldKeyPath }()

	path := GetKeyFilePath()
	os.WriteFile(path, []byte("invalid-hex"), 0600)

	_, err := LoadKey()
	if err == nil {
		t.Error("Expected error with invalid key file content")
	}

	os.WriteFile(path, []byte("ab12"), 0600)

	_, err = LoadKey()
	if err == nil {
		t.Error("Expected error with wrong key size")
	}
}
