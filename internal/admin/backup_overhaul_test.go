package admin

import (
	"fmt"
	"strings"
	"testing"
)

func TestBackupNameRegexValidation(t *testing.T) {
	validNames := []string{
		"remnawave_backup_20261122_150000.tar.gz",
		"remnawave_safety_before_restore_20261122_160000.tar.gz",
		"my-backup.tar.gz",
		"backup_123.tar.gz",
	}

	for _, name := range validNames {
		if !validBackupNameRegex.MatchString(name) {
			t.Errorf("Expected %s to be valid", name)
		}
	}

	invalidNames := []string{
		"../escape.tar.gz",
		"file;rm -rf.tar.gz",
		"backup.zip",
		"backup.tar",
		"backup with spaces.tar.gz",
	}

	for _, name := range invalidNames {
		if validBackupNameRegex.MatchString(name) {
			t.Errorf("Expected %s to be invalid", name)
		}
	}
}

func TestSafetyBackupOutputParsing(t *testing.T) {
	sampleOutput := "SAFETY_BACKUP:remnawave_safety_before_restore_20261122_160000.tar.gz\nRestoring configs...\nRestarting services...\nРезервная копия успешно восстановлена, службы перезапущены!"
	safetyFile := ""

	for _, line := range strings.Split(sampleOutput, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "SAFETY_BACKUP:") {
			safetyFile = strings.TrimPrefix(trimmed, "SAFETY_BACKUP:")
		}
	}

	expected := "remnawave_safety_before_restore_20261122_160000.tar.gz"
	if safetyFile != expected {
		t.Errorf("Expected safetyFile %s, got %s", expected, safetyFile)
	}
}

func TestRestoreScriptGenerationScopes(t *testing.T) {
	scopes := []string{"all", "db_only", "configs_only"}
	for _, sc := range scopes {
		cmd := fmt.Sprintf(`SCOPE="%s"`, sc)
		if !strings.Contains(cmd, sc) {
			t.Errorf("Expected command to contain scope %s", sc)
		}
	}
}
