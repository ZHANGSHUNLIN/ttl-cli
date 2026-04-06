package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

const (
	// NonceSize 是 GCM nonce 的大小（12 字节是推荐值）
	NonceSize = 12
	// KeySize 是 AES-256 密钥的大小（32 字节）
	KeySize = 32
)

// Encrypt 使用 AES-256-GCM 加密明文数据
// 返回格式：{nonce_base64}:{ciphertext_base64}
func Encrypt(key []byte, plaintext string) (string, error) {
	if len(key) != KeySize {
		return "", fmt.Errorf("invalid key size: expected %d bytes, got %d", KeySize, len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// 生成随机 nonce
	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// 加密数据（GCM 会自动附加认证标签）
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)

	// 格式：nonce:ciphertext，都使用 base64 编码
	return fmt.Sprintf("%s:%s",
		base64.StdEncoding.EncodeToString(nonce),
		base64.StdEncoding.EncodeToString(ciphertext)), nil
}

// Decrypt 解密由 Encrypt 加密的数据
// 输入格式：{nonce_base64}:{ciphertext_base64}
func Decrypt(key []byte, encrypted string) (string, error) {
	if len(key) != KeySize {
		return "", fmt.Errorf("invalid key size: expected %d bytes, got %d", KeySize, len(key))
	}

	// 分离 nonce 和 ciphertext
	parts := splitN(encrypted, 2, ":")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid encrypted format")
	}

	nonce, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return "", fmt.Errorf("failed to decode nonce: %w", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	if len(nonce) != NonceSize {
		return "", fmt.Errorf("invalid nonce size: expected %d bytes, got %d", NonceSize, len(nonce))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// 解密数据
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// IsEncrypted 检查字符串是否是加密格式
func IsEncrypted(s string) bool {
	parts := splitN(s, 2, ":")
	if len(parts) != 2 {
		return false
	}

	nonceStr, ciphertextStr := parts[0], parts[1]

	// 检查 nonce 和 ciphertext 是否都是有效的 base64
	nonce, err := base64.StdEncoding.DecodeString(nonceStr)
	if err != nil {
		return false
	}
	if _, err := base64.StdEncoding.DecodeString(ciphertextStr); err != nil {
		return false
	}

	// nonce 必须是 12 字节（AES-GCM 标准长度）
	return len(nonce) == NonceSize
}

// splitN 分割字符串，最多分成 n 份
func splitN(s string, n int, sep string) []string {
	parts := make([]string, 0, n)
	start := 0
	for i := 0; i < n-1; i++ {
		idx := indexOf(s, sep, start)
		if idx == -1 {
			return append(parts, s[start:])
		}
		parts = append(parts, s[start:idx])
		start = idx + len(sep)
	}
	parts = append(parts, s[start:])
	return parts
}

// indexOf 查找子串的位置
func indexOf(s, substr string, start int) int {
	if start < 0 {
		start = 0
	}
	if start > len(s) {
		return -1
	}
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
