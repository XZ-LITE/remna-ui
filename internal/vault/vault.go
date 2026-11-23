package vault

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"remna-ui/internal/crypto"
	"remna-ui/internal/types"
)

var (
	modcrypt32                 = syscall.NewLazyDLL("crypt32.dll")
	procCryptProtectData   = modcrypt32.NewProc("CryptProtectData")
	procCryptUnprotectData = modcrypt32.NewProc("CryptUnprotectData")
)

type dataBlob struct {
	cbData uint32
	pbData *byte
}

type VaultConfigFile struct {
	Enabled         bool   `json:"enabled"`
	SaltHex         string `json:"salt_hex"`
	VerifierHex     string `json:"verifier_hex"`
	AutoLockMinutes int    `json:"auto_lock_minutes"`
}

type Manager struct {
	mu           sync.RWMutex
	storeDir     string
	config       VaultConfigFile
	activeKey    []byte
	isUnlocked   bool
	lastActivity time.Time
}

var (
	instance *Manager
	once     sync.Once
)

func GetManager() *Manager {
	once.Do(func() {
		instance = &Manager{
			storeDir:     resolveStoreDir(),
			lastActivity: time.Now(),
		}
		instance.loadConfig()
	})
	return instance
}

func resolveStoreDir() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		home, _ := os.UserHomeDir()
		appData = filepath.Join(home, ".config")
	}
	subDir := os.Getenv("REMNAWAVE_DATA_DIR")
	if subDir == "" {
		subDir = "remnawave"
	}
	if filepath.IsAbs(subDir) {
		_ = os.MkdirAll(subDir, 0755)
		return subDir
	}
	targetDir := filepath.Join(appData, subDir)
	_ = os.MkdirAll(targetDir, 0755)
	return targetDir
}

func (m *Manager) configPath() string {
	return filepath.Join(m.storeDir, "vault_config.json")
}

func (m *Manager) secretsPath() string {
	return filepath.Join(m.storeDir, "vault_secrets.json")
}

func (m *Manager) serversPath() string {
	return filepath.Join(m.storeDir, "servers.json")
}

func (m *Manager) loadConfig() {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.configPath())
	if err != nil {
		m.config = VaultConfigFile{
			Enabled:         false,
			AutoLockMinutes: 15,
		}
		m.isUnlocked = true
		return
	}

	var cfg VaultConfigFile
	if err := json.Unmarshal(data, &cfg); err == nil && cfg.Enabled {
		if cfg.AutoLockMinutes <= 0 {
			cfg.AutoLockMinutes = 15
		}
		m.config = cfg
		m.isUnlocked = false
	} else {
		m.config = VaultConfigFile{
			Enabled:         false,
			AutoLockMinutes: 15,
		}
		m.isUnlocked = true
	}
}

func (m *Manager) saveConfigLocked() error {
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.configPath(), data, 0600)
}

func (m *Manager) Touch() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastActivity = time.Now()
}

func (m *Manager) CheckAutoLock() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.config.Enabled || !m.isUnlocked || m.config.AutoLockMinutes <= 0 {
		return
	}

	if time.Since(m.lastActivity) > time.Duration(m.config.AutoLockMinutes)*time.Minute {
		if len(m.activeKey) > 0 {
			crypto.Memclr(m.activeKey)
			m.activeKey = nil
		}
		m.isUnlocked = false
	}
}

func (m *Manager) GetStatus() types.VaultStatus {
	m.CheckAutoLock()

	m.mu.RLock()
	defer m.mu.RUnlock()

	mode := "dpapi"
	if m.config.Enabled {
		mode = "master_password"
	}

	hasServers := false
	if fi, err := os.Stat(m.serversPath()); err == nil && fi.Size() > 0 {
		hasServers = true
	}

	return types.VaultStatus{
		Enabled:         m.config.Enabled,
		Unlocked:        m.isUnlocked,
		Mode:            mode,
		AutoLockMinutes: m.config.AutoLockMinutes,
		HasSavedServers: hasServers,
	}
}

func (m *Manager) Unlock(password string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.config.Enabled {
		m.isUnlocked = true
		return true, nil
	}

	salt, err := hex.DecodeString(m.config.SaltHex)
	if err != nil || len(salt) == 0 {
		return false, errors.New("повреждена соль хранилища мастер-пароля")
	}

	storedVerifier, err := hex.DecodeString(m.config.VerifierHex)
	if err != nil || len(storedVerifier) == 0 {
		return false, errors.New("поврежден проверочный хэш мастер-пароля")
	}

	derivedKey := crypto.DeriveKey(password, salt)
	if !crypto.VerifyKey(derivedKey, salt, storedVerifier) {
		crypto.Memclr(derivedKey)
		return false, errors.New("неверный мастер-пароль")
	}

	if len(m.activeKey) > 0 {
		crypto.Memclr(m.activeKey)
	}

	m.activeKey = derivedKey
	m.isUnlocked = true
	m.lastActivity = time.Now()

	return true, nil
}

func (m *Manager) Lock() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.activeKey) > 0 {
		crypto.Memclr(m.activeKey)
		m.activeKey = nil
	}
	if m.config.Enabled {
		m.isUnlocked = false
	}
}

func (m *Manager) SetMasterPassword(oldPass, newPass string, reencryptFn func(decryptOld func(string) (string, error), encryptNew func(string) (string, error)) error) error {
	cleanNew := strings.TrimSpace(newPass)
	if len(cleanNew) < 8 {
		return errors.New("мастер-пароль должен содержать не менее 8 символов")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.config.Enabled {
		salt, _ := hex.DecodeString(m.config.SaltHex)
		storedVerifier, _ := hex.DecodeString(m.config.VerifierHex)
		oldKey := crypto.DeriveKey(oldPass, salt)
		if !crypto.VerifyKey(oldKey, salt, storedVerifier) {
			crypto.Memclr(oldKey)
			return errors.New("текущий мастер-пароль указан неверно")
		}
		crypto.Memclr(oldKey)
	}

	newSalt, err := crypto.GenerateSalt()
	if err != nil {
		return err
	}

	newKey := crypto.DeriveKey(cleanNew, newSalt)
	verifier := crypto.ComputeVerifier(newKey, newSalt)

	oldActiveKey := m.activeKey
	oldConfig := m.config
	wasUnlocked := m.isUnlocked

	decryptOld := func(raw string) (string, error) {
		if strings.HasPrefix(raw, "enc:vault:") {
			if len(oldActiveKey) == 0 {
				return "", errors.New("хранилище заблокировано, расшифровка невозможна")
			}
			data, err := crypto.DecryptAESGCM(strings.TrimPrefix(raw, "enc:vault:"), oldActiveKey)
			if err != nil {
				return "", err
			}
			return string(data), nil
		}
		if strings.HasPrefix(raw, "enc:dpapi:") {
			return decryptDPAPI(raw), nil
		}
		return raw, nil
	}

	encryptNew := func(plain string) (string, error) {
		if plain == "" {
			return "", nil
		}
		enc, err := crypto.EncryptAESGCM([]byte(plain), newKey)
		if err != nil {
			return "", err
		}
		return "enc:vault:" + enc, nil
	}

	if reencryptFn != nil {
		if err := reencryptFn(decryptOld, encryptNew); err != nil {
			crypto.Memclr(newKey)
			return fmt.Errorf("ошибка перешифрования данных: %w", err)
		}
	}

	if err := m.reencryptSecureValuesLocked(decryptOld, encryptNew); err != nil {
		crypto.Memclr(newKey)
		return fmt.Errorf("ошибка перешифрования системных токенов: %w", err)
	}

	if len(oldActiveKey) > 0 {
		crypto.Memclr(oldActiveKey)
	}

	m.config = VaultConfigFile{
		Enabled:         true,
		SaltHex:         hex.EncodeToString(newSalt),
		VerifierHex:     hex.EncodeToString(verifier),
		AutoLockMinutes: oldConfig.AutoLockMinutes,
	}
	if m.config.AutoLockMinutes <= 0 {
		m.config.AutoLockMinutes = 15
	}

	if err := m.saveConfigLocked(); err != nil {
		m.config = oldConfig
		m.activeKey = oldActiveKey
		m.isUnlocked = wasUnlocked
		crypto.Memclr(newKey)
		return fmt.Errorf("ошибка сохранения конфигурации хранилища: %w", err)
	}

	m.activeKey = newKey
	m.isUnlocked = true
	m.lastActivity = time.Now()

	return nil
}

func (m *Manager) DisableMasterPassword(currentPass string, reencryptFn func(decryptOld func(string) (string, error), encryptNew func(string) (string, error)) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.config.Enabled {
		return nil
	}

	salt, _ := hex.DecodeString(m.config.SaltHex)
	storedVerifier, _ := hex.DecodeString(m.config.VerifierHex)
	curKey := crypto.DeriveKey(currentPass, salt)
	if !crypto.VerifyKey(curKey, salt, storedVerifier) {
		crypto.Memclr(curKey)
		return errors.New("неверный мастер-пароль")
	}

	decryptOld := func(raw string) (string, error) {
		if strings.HasPrefix(raw, "enc:vault:") {
			data, err := crypto.DecryptAESGCM(strings.TrimPrefix(raw, "enc:vault:"), curKey)
			if err != nil {
				return "", err
			}
			return string(data), nil
		}
		if strings.HasPrefix(raw, "enc:dpapi:") {
			return decryptDPAPI(raw), nil
		}
		return raw, nil
	}

	encryptNew := func(plain string) (string, error) {
		if plain == "" {
			return "", nil
		}
		return encryptDPAPI(plain), nil
	}

	if reencryptFn != nil {
		if err := reencryptFn(decryptOld, encryptNew); err != nil {
			crypto.Memclr(curKey)
			return fmt.Errorf("ошибка перешифрования серверов в DPAPI: %w", err)
		}
	}

	if err := m.reencryptSecureValuesLocked(decryptOld, encryptNew); err != nil {
		crypto.Memclr(curKey)
		return fmt.Errorf("ошибка перешифрования токенов: %w", err)
	}

	crypto.Memclr(curKey)
	if len(m.activeKey) > 0 {
		crypto.Memclr(m.activeKey)
		m.activeKey = nil
	}

	_ = os.Remove(m.configPath())
	m.config = VaultConfigFile{
		Enabled:         false,
		AutoLockMinutes: 15,
	}
	m.isUnlocked = true

	return nil
}

func (m *Manager) SetAutoLockMinutes(mins int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if mins < 0 {
		mins = 0
	}
	m.config.AutoLockMinutes = mins
	if m.config.Enabled {
		return m.saveConfigLocked()
	}
	return nil
}

func (m *Manager) EncryptSecret(secret string) (string, error) {
	if secret == "" {
		return "", nil
	}

	m.CheckAutoLock()
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.config.Enabled {
		if !m.isUnlocked || len(m.activeKey) == 0 {
			return "", errors.New("хранилище заблокировано, сохранение секретов невозможно")
		}
		enc, err := crypto.EncryptAESGCM([]byte(secret), m.activeKey)
		if err != nil {
			return "", err
		}
		return "enc:vault:" + enc, nil
	}

	return encryptDPAPI(secret), nil
}

func (m *Manager) DecryptSecret(encrypted string) (string, error) {
	if encrypted == "" {
		return "", nil
	}

	if strings.HasPrefix(encrypted, "enc:vault:") {
		m.CheckAutoLock()
		m.mu.RLock()
		defer m.mu.RUnlock()

		if !m.config.Enabled || !m.isUnlocked || len(m.activeKey) == 0 {
			return "", errors.New("хранилище заблокировано мастер-паролем")
		}
		b64Payload := strings.TrimPrefix(encrypted, "enc:vault:")
		data, err := crypto.DecryptAESGCM(b64Payload, m.activeKey)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}

	if strings.HasPrefix(encrypted, "enc:dpapi:") {
		return decryptDPAPI(encrypted), nil
	}

	return encrypted, nil
}

func (m *Manager) SetSecureValue(key, value string) error {
	m.Touch()
	cleanKey := strings.TrimSpace(key)
	if cleanKey == "" {
		return errors.New("ключ не может быть пустым")
	}

	encVal, err := m.EncryptSecret(value)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	secrets := make(map[string]string)
	if data, err := os.ReadFile(m.secretsPath()); err == nil {
		_ = json.Unmarshal(data, &secrets)
	}

	if value == "" {
		delete(secrets, cleanKey)
	} else {
		secrets[cleanKey] = encVal
	}

	out, err := json.MarshalIndent(secrets, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.secretsPath(), out, 0600)
}

func (m *Manager) GetSecureValue(key string) (string, error) {
	m.Touch()
	cleanKey := strings.TrimSpace(key)
	if cleanKey == "" {
		return "", nil
	}

	m.mu.RLock()
	var rawEnc string
	if data, err := os.ReadFile(m.secretsPath()); err == nil {
		var secrets map[string]string
		if err := json.Unmarshal(data, &secrets); err == nil {
			rawEnc = secrets[cleanKey]
		}
	}
	m.mu.RUnlock()

	if rawEnc == "" {
		return "", nil
	}

	return m.DecryptSecret(rawEnc)
}

func (m *Manager) reencryptSecureValuesLocked(decryptOld func(string) (string, error), encryptNew func(string) (string, error)) error {
	data, err := os.ReadFile(m.secretsPath())
	if err != nil {
		return nil
	}

	var secrets map[string]string
	if err := json.Unmarshal(data, &secrets); err != nil {
		return nil
	}

	for k, v := range secrets {
		dec, err := decryptOld(v)
		if err != nil {
			continue
		}
		enc, err := encryptNew(dec)
		if err != nil {
			return err
		}
		secrets[k] = enc
	}

	out, err := json.MarshalIndent(secrets, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.secretsPath(), out, 0600)
}

func encryptDPAPI(secret string) string {
	if secret == "" || strings.HasPrefix(secret, "enc:dpapi:") || strings.HasPrefix(secret, "enc:vault:") {
		return secret
	}
	if runtime.GOOS != "windows" {
		return secret
	}
	bytesData := []byte(secret)
	var inBlob, outBlob dataBlob
	inBlob.cbData = uint32(len(bytesData))
	if len(bytesData) > 0 {
		inBlob.pbData = &bytesData[0]
	}
	r, _, _ := procCryptProtectData.Call(
		uintptr(unsafe.Pointer(&inBlob)),
		0,
		0,
		0,
		0,
		0x01, // CRYPTPROTECT_UI_FORBIDDEN
		uintptr(unsafe.Pointer(&outBlob)),
	)
	if r == 0 || outBlob.pbData == nil {
		return secret
	}
	defer syscall.LocalFree(syscall.Handle(unsafe.Pointer(outBlob.pbData)))
	outBytes := make([]byte, outBlob.cbData)
	for i := uint32(0); i < outBlob.cbData; i++ {
		outBytes[i] = *(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(outBlob.pbData)) + uintptr(i)))
	}
	return "enc:dpapi:" + base64.StdEncoding.EncodeToString(outBytes)
}

func decryptDPAPI(encrypted string) string {
	if !strings.HasPrefix(encrypted, "enc:dpapi:") {
		return encrypted
	}
	if runtime.GOOS != "windows" {
		return encrypted
	}
	rawB64 := strings.TrimPrefix(encrypted, "enc:dpapi:")
	data, err := base64.StdEncoding.DecodeString(rawB64)
	if err != nil || len(data) == 0 {
		return encrypted
	}
	var inBlob, outBlob dataBlob
	inBlob.cbData = uint32(len(data))
	inBlob.pbData = &data[0]
	r, _, _ := procCryptUnprotectData.Call(
		uintptr(unsafe.Pointer(&inBlob)),
		0,
		0,
		0,
		0,
		0x01, // CRYPTPROTECT_UI_FORBIDDEN
		uintptr(unsafe.Pointer(&outBlob)),
	)
	if r == 0 || outBlob.pbData == nil {
		return encrypted
	}
	defer syscall.LocalFree(syscall.Handle(unsafe.Pointer(outBlob.pbData)))
	outBytes := make([]byte, outBlob.cbData)
	for i := uint32(0); i < outBlob.cbData; i++ {
		outBytes[i] = *(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(outBlob.pbData)) + uintptr(i)))
	}
	return string(outBytes)
}
