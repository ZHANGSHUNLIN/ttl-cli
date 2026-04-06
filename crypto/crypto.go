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
	NonceSize = 12
	KeySize   = 32
)

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

	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)

	return fmt.Sprintf("%s:%s",
		base64.StdEncoding.EncodeToString(nonce),
		base64.StdEncoding.EncodeToString(ciphertext)), nil
}

func Decrypt(key []byte, encrypted string) (string, error) {
	if len(key) != KeySize {
		return "", fmt.Errorf("invalid key size: expected %d bytes, got %d", KeySize, len(key))
	}

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

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

func IsEncrypted(s string) bool {
	parts := splitN(s, 2, ":")
	if len(parts) != 2 {
		return false
	}

	nonceStr, ciphertextStr := parts[0], parts[1]

	nonce, err := base64.StdEncoding.DecodeString(nonceStr)
	if err != nil {
		return false
	}
	if _, err := base64.StdEncoding.DecodeString(ciphertextStr); err != nil {
		return false
	}

	return len(nonce) == NonceSize
}

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
