package vault

import (
	"os"
	"strings"
	"sync"
	"testing"
)

func TestVaultLifecycle(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "remna_vault_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr := &Manager{
		storeDir: tempDir,
	}
	mgr.loadConfig()

	status := mgr.GetStatus()
	if status.Enabled {
		t.Fatalf("expected vault to be disabled by default")
	}
	if !status.Unlocked {
		t.Fatalf("expected vault to be unlocked in DPAPI mode")
	}

	testPlainSecret := "my-super-secret-ssh-key-password"

	// 1. Encrypt in default mode (DPAPI or plaintext on non-windows)
	enc1, err := mgr.EncryptSecret(testPlainSecret)
	if err != nil {
		t.Fatalf("EncryptSecret in default mode failed: %v", err)
	}
	dec1, err := mgr.DecryptSecret(enc1)
	if err != nil {
		t.Fatalf("DecryptSecret failed: %v", err)
	}
	if dec1 != testPlainSecret {
		t.Fatalf("decrypted mismatch: got %s, want %s", dec1, testPlainSecret)
	}

	// 2. Enable Master Password
	masterPass := "CorrectVaultPass2026!"
	err = mgr.SetMasterPassword("", masterPass, nil)
	if err != nil {
		t.Fatalf("SetMasterPassword failed: %v", err)
	}

	status = mgr.GetStatus()
	if !status.Enabled {
		t.Fatalf("expected vault to be enabled")
	}
	if !status.Unlocked {
		t.Fatalf("expected vault to remain unlocked right after setting password")
	}

	// 3. Encrypt in Master Password mode
	encVault, err := mgr.EncryptSecret(testPlainSecret)
	if err != nil {
		t.Fatalf("EncryptSecret in vault mode failed: %v", err)
	}
	if !strings.HasPrefix(encVault, "enc:vault:") {
		t.Fatalf("expected enc:vault: prefix, got: %s", encVault)
	}

	decVault, err := mgr.DecryptSecret(encVault)
	if err != nil {
		t.Fatalf("DecryptSecret in vault mode failed: %v", err)
	}
	if decVault != testPlainSecret {
		t.Fatalf("decrypted mismatch in vault: got %s, want %s", decVault, testPlainSecret)
	}

	// 4. Lock vault
	mgr.Lock()
	status = mgr.GetStatus()
	if status.Unlocked {
		t.Fatalf("expected vault to be locked")
	}

	// Decrypting while locked should fail
	_, errLocked := mgr.DecryptSecret(encVault)
	if errLocked == nil {
		t.Fatalf("DecryptSecret while locked should fail")
	}

	// 5. Unlock with incorrect password
	unlocked, errWrong := mgr.Unlock("WrongPassword")
	if unlocked || errWrong == nil {
		t.Fatalf("Unlock with wrong password should fail")
	}

	// 6. Unlock with correct password
	unlocked, errRight := mgr.Unlock(masterPass)
	if !unlocked || errRight != nil {
		t.Fatalf("Unlock with right password failed: %v", errRight)
	}

	decAfterUnlock, err := mgr.DecryptSecret(encVault)
	if err != nil || decAfterUnlock != testPlainSecret {
		t.Fatalf("DecryptSecret after unlock failed: %v", err)
	}

	// 7. Secure values (key-value storage for API tokens)
	err = mgr.SetSecureValue("rw_api_token", "test-token-12345")
	if err != nil {
		t.Fatalf("SetSecureValue failed: %v", err)
	}
	tok, err := mgr.GetSecureValue("rw_api_token")
	if err != nil || tok != "test-token-12345" {
		t.Fatalf("GetSecureValue mismatch: got %s, want test-token-12345", tok)
	}

	// 8. Disable Master Password
	err = mgr.DisableMasterPassword(masterPass, nil)
	if err != nil {
		t.Fatalf("DisableMasterPassword failed: %v", err)
	}
	status = mgr.GetStatus()
	if status.Enabled {
		t.Fatalf("expected vault to be disabled after DisableMasterPassword")
	}
}

func TestLocalMachineEncryption(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "remna_local_crypto_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	secret := "ssh-password-example-123"
	enc := encryptLocalMachine(tempDir, "machine-abc-001", secret)
	if !strings.HasPrefix(enc, prefixLocal) {
		t.Fatalf("expected %s prefix, got %q", prefixLocal, enc)
	}

	dec := decryptLocalMachine(tempDir, "machine-abc-001", enc)
	if dec != secret {
		t.Fatalf("decryptLocalMachine got %q, want %q", dec, secret)
	}

	// Wrong machine ID must NOT decrypt successfully
	wrong := decryptLocalMachine(tempDir, "machine-other-999", enc)
	if wrong == secret {
		t.Fatalf("expected wrong machine ID not to decrypt secret")
	}
}

func TestVault_CorruptedConfigStaysLocked(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "remna_vault_corrupt_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr := &Manager{storeDir: tempDir}
	if err := os.WriteFile(mgr.configPath(), []byte("{not-json"), 0600); err != nil {
		t.Fatalf("write corrupt config: %v", err)
	}
	mgr.loadConfig()
	st := mgr.GetStatus()
	if !st.Enabled || st.Unlocked || st.Message == "" {
		t.Fatalf("expected corrupted config to stay locked with message, got: %+v", st)
	}
}

func TestVault_NoDeadlockWithConcurrentReencrypt(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "remna_vault_race_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr := &Manager{storeDir: tempDir}
	mgr.loadConfig()

	var externalMu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			externalMu.Lock()
			_, _ = mgr.EncryptSecret("secret-value")
			_ = mgr.GetStatus()
			externalMu.Unlock()
		}
	}()

	go func() {
		defer wg.Done()
		_ = mgr.SetMasterPassword("", "StrongPass123!", func(dec func(string) (string, error), enc func(string) (string, error)) error {
			externalMu.Lock()
			defer externalMu.Unlock()
			_, _ = enc("secret-value")
			return nil
		})
	}()

	wg.Wait()
}
