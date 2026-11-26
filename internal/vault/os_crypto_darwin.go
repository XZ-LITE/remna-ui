//go:build darwin

package vault

import (
	"encoding/hex"
	"os/exec"
	"regexp"
	"strings"

	"remna-ui/internal/crypto"
)

const (
	keychainService = "org.remnawave.ui"
	keychainAccount = "vault-os-key"
)

var platformUUIDRegex = regexp.MustCompile(`"IOPlatformUUID"\s*=\s*"([^"]+)"`)

func getDarwinPlatformUUID() string {
	out, err := exec.Command("/usr/sbin/ioreg", "-rd1", "-c", "IOPlatformExpertDevice").Output()
	if err == nil {
		if m := platformUUIDRegex.FindSubmatch(out); len(m) == 2 {
			return strings.TrimSpace(string(m[1]))
		}
	}
	return "darwin-host"
}

func getOrCreateKeychainKey() ([]byte, bool) {
	// Try reading existing 32-byte hex key from macOS Keychain
	if out, err := exec.Command("/usr/bin/security", "find-generic-password", "-s", keychainService, "-a", keychainAccount, "-w").Output(); err == nil {
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
	if err := exec.Command("/usr/bin/security", "add-generic-password", "-U", "-s", keychainService, "-a", keychainAccount, "-w", keyHex).Run(); err != nil {
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

func (m *Manager) encryptOSSecret(secret string) string {
	if secret == "" || isEncryptedSecret(secret) {
		return secret
	}
	if key, ok := m.resolveDarwinOSKey(); ok {
		if enc, err := crypto.EncryptAESGCM([]byte(secret), key); err == nil {
			return prefixOS + enc
		}
	}
	return encryptLocalMachine(m.storeDir, getDarwinPlatformUUID(), secret)
}

func (m *Manager) decryptOSSecret(encrypted string) string {
	if strings.HasPrefix(encrypted, prefixOS) {
		if key, ok := m.resolveDarwinOSKey(); ok {
			if plain, err := crypto.DecryptAESGCM(strings.TrimPrefix(encrypted, prefixOS), key); err == nil {
				return string(plain)
			}
		}
		return encrypted
	}
	if strings.HasPrefix(encrypted, prefixLocal) {
		return decryptLocalMachine(m.storeDir, getDarwinPlatformUUID(), encrypted)
	}
	return encrypted
}
