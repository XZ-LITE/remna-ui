package vault

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"

	"remna-ui/internal/crypto"
)

const (
	prefixOS    = "enc:os:"
	prefixLocal = "enc:local:"
	prefixDPAPI = "enc:dpapi:"
	prefixVault = "enc:vault:"
)

func isEncryptedSecret(s string) bool {
	return strings.HasPrefix(s, prefixDPAPI) ||
		strings.HasPrefix(s, prefixVault) ||
		strings.HasPrefix(s, prefixOS) ||
		strings.HasPrefix(s, prefixLocal)
}

func generateRandomAESKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

func loadOrCreateLocalSalt(storeDir string) ([]byte, error) {
	if storeDir == "" {
		return nil, os.ErrInvalid
	}
	_ = os.MkdirAll(storeDir, 0755)
	saltPath := filepath.Join(storeDir, "vault_local.salt")
	if raw, err := os.ReadFile(saltPath); err == nil {
		if salt, decErr := hex.DecodeString(strings.TrimSpace(string(raw))); decErr == nil && len(salt) >= 16 {
			return salt, nil
		}
	}
	salt := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	if err := os.WriteFile(saltPath, []byte(hex.EncodeToString(salt)), 0600); err != nil {
		return nil, err
	}
	return salt, nil
}

func deriveLocalMachineKey(storeDir, machineID string) ([]byte, error) {
	salt, err := loadOrCreateLocalSalt(storeDir)
	if err != nil {
		return nil, err
	}
	h := sha256.New()
	h.Write([]byte("remnawave-local-v1|"))
	h.Write([]byte(strings.TrimSpace(machineID)))
	h.Write([]byte("|"))
	h.Write(salt)
	return h.Sum(nil), nil
}

func encryptLocalMachine(storeDir, machineID, secret string) string {
	if secret == "" || isEncryptedSecret(secret) {
		return secret
	}
	key, err := deriveLocalMachineKey(storeDir, machineID)
	if err != nil {
		return secret
	}
	defer crypto.Memclr(key)
	enc, err := crypto.EncryptAESGCM([]byte(secret), key)
	if err != nil {
		return secret
	}
	return prefixLocal + enc
}

func decryptLocalMachine(storeDir, machineID, encrypted string) string {
	if !strings.HasPrefix(encrypted, prefixLocal) {
		return encrypted
	}
	key, err := deriveLocalMachineKey(storeDir, machineID)
	if err != nil {
		return encrypted
	}
	defer crypto.Memclr(key)
	plain, err := crypto.DecryptAESGCM(strings.TrimPrefix(encrypted, prefixLocal), key)
	if err != nil {
		return encrypted
	}
	return string(plain)
}
