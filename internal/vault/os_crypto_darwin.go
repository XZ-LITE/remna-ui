//go:build darwin

package vault

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"remna-ui/internal/crypto"
)

const (
	keychainService = "org.remnawave.ui"
	keychainAccount = "vault-os-key"
)

var platformUUIDRegex = regexp.MustCompile(`"IOPlatformUUID"\s*=\s*"([^"]+)"`)

func getDarwinPlatformUUID() string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "/usr/sbin/ioreg", "-rd1", "-c", "IOPlatformExpertDevice").Output()
	if err == nil {
		if m := platformUUIDRegex.FindSubmatch(out); len(m) == 2 {
			return strings.TrimSpace(string(m[1]))
		}
	}
	return "darwin-host"
}

func getOrCreateKeychainKey() ([]byte, bool) {
	// Skip interactive Keychain prompts in headless CI test runners
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
		return nil, false
	}

	ctxFind, cancelFind := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelFind()
	if out, err := exec.CommandContext(ctxFind, "/usr/bin/security", "find-generic-password", "-s", keychainService, "-a", keychainAccount, "-w").Output(); err == nil {
		keyHex := strings.TrimSpace(string(out))
		if raw, decErr := hex.DecodeString(keyHex); decErr == nil && len(raw) == 32 {
			return raw, true
		}
	}

	// Generate fresh 256-bit key and store in macOS Keychain
	newKey, err := generateRandomAESKey()
	if err != nil {
		return nil, false
	}
	keyHex := hex.EncodeToString(newKey)
	ctxAdd, cancelAdd := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelAdd()
	if err := exec.CommandContext(ctxAdd, "/usr/bin/security", "add-generic-password", "-U", "-s", keychainService, "-a", keychainAccount, "-w", keyHex).Run(); err != nil {
		crypto.Memclr(newKey)
		return nil, false
	}
	return newKey, true
}

func (m *Manager) resolveDarwinOSKey() ([]byte, bool) {
	m.osKeyMu.Lock()
	defer m.osKeyMu.Unlock()
	if len(m.osKey) == 32 {
		return m.osKey, true
	}
	key, ok := getOrCreateKeychainKey()
	if ok {
		m.osKey = key
		return key, true
	}
	return nil, false
}

func (m *Manager) getOSProtectionBackend() string {
	if _, ok := m.resolveDarwinOSKey(); ok {
		return "keychain"
	}
	return "local_machine"
}

func (m *Manager) encryptOSSecret(secret string) (string, error) {
	if secret == "" || isEncryptedSecret(secret) {
		return secret, nil
	}
	if key, ok := m.resolveDarwinOSKey(); ok {
		if enc, err := crypto.EncryptAESGCM([]byte(secret), key); err == nil {
			return prefixOS + enc, nil
		}
	}
	return encryptLocalMachine(m.storeDir, getDarwinPlatformUUID(), secret), nil
}

func (m *Manager) decryptOSSecret(encrypted string) (string, error) {
	if strings.HasPrefix(encrypted, prefixOS) {
		key, ok := m.resolveDarwinOSKey()
		if !ok {
			return "", fmt.Errorf("macOS Keychain недоступен для расшифровки")
		}
		plain, err := crypto.DecryptAESGCM(strings.TrimPrefix(encrypted, prefixOS), key)
		if err != nil {
			return "", err
		}
		return string(plain), nil
	}
	if strings.HasPrefix(encrypted, prefixLocal) {
		return decryptLocalMachine(m.storeDir, getDarwinPlatformUUID(), encrypted), nil
	}
	return encrypted, nil
}
