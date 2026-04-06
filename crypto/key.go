package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const (
	KeyFileName = ".key"
	KeyFileMode = 0600
)

var keyFilePath string

func GetKeyFilePath() string {
	if keyFilePath != "" {
		return keyFilePath
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "." // fallback
	}

	configDir := filepath.Join(homeDir, ".ttl")
	keyFilePath = filepath.Join(configDir, KeyFileName)
	return keyFilePath
}

func SetKeyFilePath(path string) {
	keyFilePath = path
}

func LoadKey() ([]byte, error) {
	path := GetKeyFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	key, err := hex.DecodeString(string(data))
	if err != nil {
		return nil, fmt.Errorf("invalid key file format: %w", err)
	}

	if len(key) != KeySize {
		return nil, fmt.Errorf("invalid key size in file: expected %d bytes, got %d", KeySize, len(key))
	}

	return key, nil
}

func GenerateKey() ([]byte, error) {
	key := make([]byte, KeySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	return key, nil
}

func SaveKey(key []byte) error {
	if len(key) != KeySize {
		return fmt.Errorf("invalid key size: expected %d bytes, got %d", KeySize, len(key))
	}

	path := GetKeyFilePath()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create key directory: %w", err)
	}

	keyHex := hex.EncodeToString(key)
	if err := os.WriteFile(path, []byte(keyHex), KeyFileMode); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}

	if err := setKeyFilePermissions(path); err != nil {
		fmt.Printf("Warning: failed to set extra permissions on key file: %v\n", err)
	}

	return nil
}

func setKeyFilePermissions(path string) error {
	switch runtime.GOOS {
	case "windows":
		return setWindowsFilePermissions(path)
	default:
		return nil
	}
}

func setWindowsFilePermissions(path string) error {
	return nil
}

func DeleteKey() error {
	path := GetKeyFilePath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete key file: %w", err)
	}
	return nil
}

func KeyExists() bool {
	path := GetKeyFilePath()
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func ExportKey(destPath string) error {
	key, err := LoadKey()
	if err != nil {
		return fmt.Errorf("failed to load key: %w", err)
	}
	if key == nil {
		return fmt.Errorf("key file does not exist")
	}

	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create export directory: %w", err)
	}

	if err := os.WriteFile(destPath, []byte(hex.EncodeToString(key)), KeyFileMode); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}

	return nil
}

func ImportKey(srcPath string) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	key, err := hex.DecodeString(string(data))
	if err != nil {
		return fmt.Errorf("invalid key file format: %w", err)
	}

	if len(key) != KeySize {
		return fmt.Errorf("invalid key size: expected %d bytes, got %d", KeySize, len(key))
	}

	return SaveKey(key)
}

func VerifyKey() error {
	key, err := LoadKey()
	if err != nil {
		return fmt.Errorf("failed to load key: %w", err)
	}
	if key == nil {
		return fmt.Errorf("key file does not exist")
	}

	testData := "encryption-test"
	encrypted, err := Encrypt(key, testData)
	if err != nil {
		return fmt.Errorf("key encryption test failed: %w", err)
	}
	_, err = Decrypt(key, encrypted)
	if err != nil {
		return fmt.Errorf("key decryption test failed: %w", err)
	}

	return nil
}

func InitEncryption(createIfMissing bool) error {
	key, err := LoadKey()
	if err != nil {
		return err
	}

	if key == nil && createIfMissing {
		key, err = GenerateKey()
		if err != nil {
			return fmt.Errorf("failed to generate key: %w", err)
		}
		if err := SaveKey(key); err != nil {
			return fmt.Errorf("failed to save key: %w", err)
		}
	}

	return nil
}

func IsEncryptionEnabled() bool {
	return KeyExists()
}
