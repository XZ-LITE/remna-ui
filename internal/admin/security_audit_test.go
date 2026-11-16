package admin

import (
	"strings"
	"testing"

	"remna-ui/internal/types"
)

func TestSecurity_ContainerNameValidation(t *testing.T) {
	validNames := []string{
		"remnanode",
		"remna-caddy",
		"remna_db",
		"caddy-1.2.3",
		"web_service_01",
	}
	for _, name := range validNames {
		if !isValidContainerName(name) {
			t.Errorf("isValidContainerName(%q) = false; want true", name)
		}
	}

	invalidNames := []string{
		"",
		"   ",
		"remnanode; rm -rf /",
		"remnanode && echo pwned",
		"remnanode|cat /etc/shadow",
		"remnanode`whoami`",
		"remnanode$(id)",
		"../remnanode",
		"/remnanode",
		"-invalidStart",
		"remna node",
		"remna\nnode",
		"remna\x00node",
	}
	for _, name := range invalidNames {
		if isValidContainerName(name) {
			t.Errorf("isValidContainerName(%q) = true; want false (CRITICAL security bypass)", name)
		}
	}
}

func TestSecurity_BackupNameValidation(t *testing.T) {
	validNames := []string{
		"remna_backup_2026-11-15.tar.gz",
		"backup-1.tar.gz",
		"full_backup.tar.gz",
	}
	for _, name := range validNames {
		if !validBackupNameRegex.MatchString(name) {
			t.Errorf("validBackupNameRegex(%q) = false; want true", name)
		}
	}

	invalidNames := []string{
		"",
		"../../etc/shadow",
		"../backup.tar.gz",
		"backup.tar.gz; rm -rf /",
		"backup.tar.gz && echo 1",
		"backup.tar.gz|sh",
		"backup`id`.tar.gz",
		"backup$(whoami).tar.gz",
		"backup.zip",
		"backup.tar",
		"backup.tar.gz.exe",
		"/opt/remnawave/backups/backup.tar.gz",
	}
	for _, name := range invalidNames {
		if validBackupNameRegex.MatchString(name) {
			t.Errorf("validBackupNameRegex(%q) = true; want false (CRITICAL security bypass)", name)
		}
	}
}

func TestSecurity_ConfigureSubscriptionTokenValidation(t *testing.T) {
	dummyServer := types.SavedServer{
		Host:     "127.0.0.1",
		Port:     22,
		Username: "root",
	}

	// 1. Empty token
	res := ConfigureSubscriptionToken(dummyServer, "")
	if res.Success || !strings.Contains(res.Error, "пустым") {
		t.Errorf("Expected rejection of empty token, got: %v", res)
	}

	// 2. Token with newlines (heredoc escape attempt)
	res = ConfigureSubscriptionToken(dummyServer, "mytoken\nEOF\nrm -rf /")
	if res.Success || !strings.Contains(res.Error, "недопустимые") {
		t.Errorf("Expected rejection of multiline token, got: %v", res)
	}

	// 3. Token with bash command injection chars
	res = ConfigureSubscriptionToken(dummyServer, "token;whoami")
	if res.Success || !strings.Contains(res.Error, "недопустимые") {
		t.Errorf("Expected rejection of token with semicolon, got: %v", res)
	}
}

func TestSecurity_ConfigureRealitySelfStealValidation(t *testing.T) {
	dummyServer := types.SavedServer{
		Host:     "127.0.0.1",
		Port:     22,
		Username: "root",
	}

	// Injected fallbackDomain
	res := ConfigureRealitySelfSteal(dummyServer, `test"; rm -rf /; echo "`)
	if res.Success || !strings.Contains(res.Error, "недопустимые") {
		t.Errorf("Expected rejection of shell injection in fallbackDomain, got: %v", res)
	}
}

func TestSecurity_DeploySelfStealDecoyValidation(t *testing.T) {
	dummyServer := types.SavedServer{
		Host:     "127.0.0.1",
		Port:     22,
		Username: "root",
	}

	// Injected domain
	res := DeploySelfStealDecoy(dummyServer, `evil.com"; rm -rf /; #`)
	if res.Success || !strings.Contains(res.Error, "недопустимые") {
		t.Errorf("Expected rejection of shell injection in decoy domain, got: %v", res)
	}
}

func TestSecurity_DPAPIExtremeValues(t *testing.T) {
	testCases := []string{
		"SimplePassword123!",
		"ПарольНаРусскомЯзыке_12345",
		"!@#$%^&*()_+-=[]{}|;':\",./<>?`~",
		"Password With Spaces And Tabs \t",
		strings.Repeat("LongSecretKey_ABC1234567890", 200), // ~5KB string
	}

	for _, tc := range testCases {
		encrypted := encryptSecret(tc)
		if !strings.HasPrefix(encrypted, "enc:dpapi:") {
			t.Errorf("Encrypted string missing enc:dpapi: prefix: %s", encrypted[:20])
		}

		decrypted := decryptSecret(encrypted)
		if decrypted != tc {
			t.Errorf("Decrypted mismatch. Expected len=%d, got len=%d", len(tc), len(decrypted))
		}
	}

	// Empty string case
	encEmpty := encryptSecret("")
	if encEmpty != "" {
		t.Errorf("encryptSecret(\"\") should return \"\", got %q", encEmpty)
	}
	decEmpty := decryptSecret("")
	if decEmpty != "" {
		t.Errorf("decryptSecret(\"\") should return \"\", got %q", decEmpty)
	}

	// Plain text pass-through (if not encrypted with DPAPI, e.g. legacy plain text)
	plain := "legacy-plain-password"
	decPlain := decryptSecret(plain)
	if decPlain != plain {
		t.Errorf("decryptSecret legacy plain mismatch: got %q, want %q", decPlain, plain)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
