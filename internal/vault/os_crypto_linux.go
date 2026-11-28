//go:build linux

package vault

import (
	"context"
	"encoding/hex"
	"os"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
	"remna-ui/internal/crypto"
)

const (
	secretServiceDest      = "org.freedesktop.secrets"
	secretServicePath      = "/org/freedesktop/secrets"
	secretCollectionPath   = "/org/freedesktop/secrets/aliases/default"
	secretServiceInterface = "org.freedesktop.Secret.Service"
	secretCollInterface    = "org.freedesktop.Secret.Collection"
	secretItemLabel        = "Remnawave Manager Vault Key"
)

type dbusSecret struct {
	Session     dbus.ObjectPath
	Parameters  []byte
	Value       []byte
	ContentType string
}

func getLinuxMachineID() string {
	for _, p := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
		if raw, err := os.ReadFile(p); err == nil {
			if s := strings.TrimSpace(string(raw)); s != "" {
				return s
			}
		}
	}
	return "linux-host"
}

func getOrCreateSecretServiceKey() ([]byte, bool) {
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" {
		return nil, false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, err := dbus.ConnectSessionBus(dbus.WithContext(ctx))
	if err != nil {
		return nil, false
	}
	defer conn.Close()

	svc := conn.Object(secretServiceDest, dbus.ObjectPath(secretServicePath))

	// Open a plain D-Bus Secret Service session
	var sessionOut dbus.Variant
	var sessionPath dbus.ObjectPath
	if err := svc.CallWithContext(ctx, secretServiceInterface+".OpenSession", 0, "plain", dbus.MakeVariant("")).Store(&sessionOut, &sessionPath); err != nil {
		return nil, false
	}
	if sessionPath == "" {
		return nil, false
	}

	attrs := map[string]string{
		"application": "remnawave-ui",
		"account":     "vault-os-key",
	}

	// Ensure default collection exists and unlock it if possible
	var unlocked []dbus.ObjectPath
	var prompt dbus.ObjectPath
	_ = svc.CallWithContext(ctx, secretServiceInterface+".Unlock", 0, []dbus.ObjectPath{dbus.ObjectPath(secretCollectionPath)}).Store(&unlocked, &prompt)

	// Search for existing item
	var lockedItems, unlockedItems []dbus.ObjectPath
	if err := svc.CallWithContext(ctx, secretServiceInterface+".SearchItems", 0, attrs).Store(&unlockedItems, &lockedItems); err != nil {
		return nil, false
	}

	if len(unlockedItems) > 0 {
		var secrets map[dbus.ObjectPath]dbusSecret
		if err := svc.CallWithContext(ctx, secretServiceInterface+".GetSecrets", 0, []dbus.ObjectPath{unlockedItems[0]}, sessionPath).Store(&secrets); err == nil {
			if sec, ok := secrets[unlockedItems[0]]; ok {
				if raw, decErr := hex.DecodeString(strings.TrimSpace(string(sec.Value))); decErr == nil && len(raw) == 32 {
					return raw, true
				}
			}
		}
	}

	// Create a new 256-bit key in the default collection
	newKey, err := generateRandomAESKey()
	if err != nil {
		return nil, false
	}
	keyHex := hex.EncodeToString(newKey)

	props := map[string]dbus.Variant{
		"org.freedesktop.Secret.Item.Label":      dbus.MakeVariant(secretItemLabel),
		"org.freedesktop.Secret.Item.Attributes": dbus.MakeVariant(attrs),
	}
	sec := dbusSecret{
		Session:     sessionPath,
		Parameters:  []byte{},
		Value:       []byte(keyHex),
		ContentType: "text/plain",
	}

	coll := conn.Object(secretServiceDest, dbus.ObjectPath(secretCollectionPath))
	var createdItem dbus.ObjectPath
	var createPrompt dbus.ObjectPath
	if err := coll.CallWithContext(ctx, secretCollInterface+".CreateItem", 0, props, sec, true).Store(&createdItem, &createPrompt); err != nil || createdItem == "" {
		crypto.Memclr(newKey)
		return nil, false
	}

	return newKey, true
}

func (m *Manager) resolveLinuxOSKey() ([]byte, bool) {
	m.osKeyMu.Lock()
	defer m.osKeyMu.Unlock()
	if len(m.osKey) == 32 {
		return m.osKey, true
	}
	key, ok := getOrCreateSecretServiceKey()
	if ok {
		m.osKey = key
		return key, true
	}
	return nil, false
}

func (m *Manager) getOSProtectionBackend() string {
	if _, ok := m.resolveLinuxOSKey(); ok {
		return "secret_service"
	}
	return "local_machine"
}

func (m *Manager) encryptOSSecret(secret string) (string, error) {
	if secret == "" || isEncryptedSecret(secret) {
		return secret, nil
	}
	if key, ok := m.resolveLinuxOSKey(); ok {
		if enc, err := crypto.EncryptAESGCM([]byte(secret), key); err == nil {
			return prefixOS + enc, nil
		}
	}
	return encryptLocalMachine(m.storeDir, getLinuxMachineID(), secret), nil
}

func (m *Manager) decryptOSSecret(encrypted string) (string, error) {
	if strings.HasPrefix(encrypted, prefixOS) {
		key, ok := m.resolveLinuxOSKey()
		if !ok {
			return "", fmt.Errorf("системный Secret Service недоступен для расшифровки")
		}
		plain, err := crypto.DecryptAESGCM(strings.TrimPrefix(encrypted, prefixOS), key)
		if err != nil {
			return "", err
		}
		return string(plain), nil
	}
	if strings.HasPrefix(encrypted, prefixLocal) {
		return decryptLocalMachine(m.storeDir, getLinuxMachineID(), encrypted), nil
	}
	return encrypted, nil
}
