package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

const (
	ArgonTime    = 3
	ArgonMemory  = 64 * 1024 // 64 MB
	ArgonThreads = 4
	ArgonKeyLen  = 32         // 256 bits
	SaltLen      = 16         // 128 bits
	NonceLen     = 12         // Standard GCM 96 bits
)

// GenerateSalt creates cryptographically secure random salt bytes
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, SaltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("ошибка генерации соли: %w", err)
	}
	return salt, nil
}

// DeriveKey derives a 256-bit symmetric key from a master password using Argon2id
func DeriveKey(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, ArgonTime, ArgonMemory, ArgonThreads, ArgonKeyLen)
}

// ComputeVerifier computes a verification hash from derived key to check password validity without decrypting all data
func ComputeVerifier(key, salt []byte) []byte {
	return argon2.IDKey(key, salt, 1, 16*1024, 2, 32)
}

// VerifyKey checks whether derived key matches stored verifier hash in constant time
func VerifyKey(key, salt, storedVerifier []byte) bool {
	computed := ComputeVerifier(key, salt)
	return subtle.ConstantTimeCompare(computed, storedVerifier) == 1
}

// EncryptAESGCM encrypts plaintext with AES-256-GCM. Returns base64 encoded [nonce + ciphertext + tag]
func EncryptAESGCM(plaintext []byte, key []byte) (string, error) {
	if len(key) != 32 {
		return "", errors.New("длина ключа AES-256 должна составлять 32 байта")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("ошибка создания шифра AES: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("ошибка создания GCM режима: %w", err)
	}

	nonce := make([]byte, NonceLen)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("ошибка генерации nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	payload := append(nonce, ciphertext...)

	return base64.StdEncoding.EncodeToString(payload), nil
}

// DecryptAESGCM decrypts base64 encoded [nonce + ciphertext + tag] with AES-256-GCM
func DecryptAESGCM(encodedCiphertext string, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("длина ключа AES-256 должна составлять 32 байта")
	}

	raw, err := base64.StdEncoding.DecodeString(encodedCiphertext)
	if err != nil {
		return nil, fmt.Errorf("ошибка декодирования base64: %w", err)
	}

	if len(raw) < NonceLen {
		return nil, errors.New("длина зашифрованных данных меньше размера nonce")
	}

	nonce := raw[:NonceLen]
	ciphertext := raw[NonceLen:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("ошибка инициализации AES: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("ошибка инициализации GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("сбой аутентификации или неверный ключ расшифровки")
	}

	return plaintext, nil
}

// Memclr securely overwrites sensitive byte slice with zeros
func Memclr(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
