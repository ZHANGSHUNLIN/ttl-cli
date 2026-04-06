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
	// KeyFileName 密钥文件名
	KeyFileName = ".key"
	// KeyFileMode 密钥文件权限（仅所有者可读写）
	KeyFileMode = 0600
)

var keyFilePath string

// GetKeyFilePath 获取密钥文件的完整路径
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

// SetKeyFilePath 设置自定义密钥文件路径（用于测试）
func SetKeyFilePath(path string) {
	keyFilePath = path
}

// LoadKey 从文件加载密钥
// 如果密钥文件不存在，返回 nil, nil（表示未启用加密）
func LoadKey() ([]byte, error) {
	path := GetKeyFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // 密钥文件不存在
		}
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	// 密钥以 hex 格式存储
	key, err := hex.DecodeString(string(data))
	if err != nil {
		return nil, fmt.Errorf("invalid key file format: %w", err)
	}

	if len(key) != KeySize {
		return nil, fmt.Errorf("invalid key size in file: expected %d bytes, got %d", KeySize, len(key))
	}

	return key, nil
}

// GenerateKey 生成新的加密密钥
func GenerateKey() ([]byte, error) {
	key := make([]byte, KeySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	return key, nil
}

// SaveKey 保存密钥到文件
func SaveKey(key []byte) error {
	if len(key) != KeySize {
		return fmt.Errorf("invalid key size: expected %d bytes, got %d", KeySize, len(key))
	}

	path := GetKeyFilePath()

	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create key directory: %w", err)
	}

	// 以 hex 格式存储密钥
	keyHex := hex.EncodeToString(key)
	if err := os.WriteFile(path, []byte(keyHex), KeyFileMode); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}

	// 设置额外的权限保护（平台特定）
	if err := setKeyFilePermissions(path); err != nil {
		// 不阻止保存，但记录警告
		fmt.Printf("Warning: failed to set extra permissions on key file: %v\n", err)
	}

	return nil
}

// setKeyFilePermissions 设置密钥文件的平台特定权限
func setKeyFilePermissions(path string) error {
	switch runtime.GOOS {
	case "windows":
		return setWindowsFilePermissions(path)
	default:
		// macOS 和 Linux 已经通过 os.WriteFile 的 mode 设置了 0600
		return nil
	}
}

// setWindowsFilePermissions 设置 Windows 文件权限
func setWindowsFilePermissions(path string) error {
	// Windows 使用 ACL，这里简化处理
	// 实际项目中可以使用 syscall 或第三方库设置更精细的 ACL
	return nil
}

// DeleteKey 删除密钥文件
func DeleteKey() error {
	path := GetKeyFilePath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete key file: %w", err)
	}
	return nil
}

// KeyExists 检查密钥文件是否存在
func KeyExists() bool {
	path := GetKeyFilePath()
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// ExportKey 导出密钥到指定路径
func ExportKey(destPath string) error {
	key, err := LoadKey()
	if err != nil {
		return fmt.Errorf("failed to load key: %w", err)
	}
	if key == nil {
		return fmt.Errorf("key file does not exist")
	}

	// 确保目录存在
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create export directory: %w", err)
	}

	if err := os.WriteFile(destPath, []byte(hex.EncodeToString(key)), KeyFileMode); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}

	return nil
}

// ImportKey 从指定路径导入密钥
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

// VerifyKey 验证密钥文件的有效性
func VerifyKey() error {
	key, err := LoadKey()
	if err != nil {
		return fmt.Errorf("failed to load key: %w", err)
	}
	if key == nil {
		return fmt.Errorf("key file does not exist")
	}

	// 尝试用密钥加密解密一个测试字符串来验证
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

// InitEncryption 初始化加密功能
// 如果密钥文件不存在且 createIfMissing 为 true，则生成新密钥
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

// IsEncryptionEnabled 检查是否启用了加密
func IsEncryptionEnabled() bool {
	return KeyExists()
}
