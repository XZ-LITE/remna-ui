package admin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"remna-ui/internal/types"
)

func TestStorageBetaIsolationAndDPAPI(t *testing.T) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		t.Skip("APPDATA is not set")
	}

	expectedBetaDir := filepath.Join(appData, "remnawave-beta")
	expectedBetaPath := filepath.Join(expectedBetaDir, "servers.json")
	prodPath := filepath.Join(appData, "remnawave", "servers.json")

	// 1. Verify getStorePath() resolves to remnawave-beta
	actualStorePath := getStorePath()
	if actualStorePath != expectedBetaPath {
		t.Fatalf("Expected store path %s, got %s", expectedBetaPath, actualStorePath)
	}

	// Record prod file modification time or existence
	var prodExisted bool
	var prodModTime int64
	if stat, err := os.Stat(prodPath); err == nil {
		prodExisted = true
		prodModTime = stat.ModTime().UnixNano()
	}

	// 2. Test saving a test server
	testID := "audit-test-server-wails3"
	testPassword := "SuperSecretPassword123!"
	testKey := "-----BEGIN OPENSSH PRIVATE KEY-----\ntestkey\n-----END OPENSSH PRIVATE KEY-----"

	server := types.SavedServer{
		ID:       testID,
		Label:    "Wails3 Audit Test",
		Host:     "192.0.2.1",
		Port:     22,
		Username: "root",
		Password: testPassword,
		SSHKey:   testKey,
		Role:     "master",
	}

	err := SaveServer(server)
	if err != nil {
		t.Fatalf("Failed to save server: %v", err)
	}

	// 3. Verify file on disk is in remnawave-beta
	data, err := os.ReadFile(expectedBetaPath)
	if err != nil {
		t.Fatalf("Failed to read beta servers.json: %v", err)
	}

	var rawList []types.SavedServer
	if err := json.Unmarshal(data, &rawList); err != nil {
		t.Fatalf("Failed to parse beta servers.json: %v", err)
	}

	var foundRaw *types.SavedServer
	for i := range rawList {
		if rawList[i].ID == testID {
			foundRaw = &rawList[i]
			break
		}
	}

	if foundRaw == nil {
		t.Fatalf("Test server was not found in raw beta servers.json")
	}

	// 4. Verify DPAPI encryption on disk
	if !strings.HasPrefix(foundRaw.Password, "enc:dpapi:") {
		t.Errorf("Password was not encrypted with DPAPI on disk: %s", foundRaw.Password)
	}
	if !strings.HasPrefix(foundRaw.SSHKey, "enc:dpapi:") {
		t.Errorf("SSH key was not encrypted with DPAPI on disk: %s", foundRaw.SSHKey)
	}

	// 5. Verify GetSavedServers() decrypts properly
	loadedServers := GetSavedServers()
	var foundDecrypted *types.SavedServer
	for i := range loadedServers {
		if loadedServers[i].ID == testID {
			foundDecrypted = &loadedServers[i]
			break
		}
	}

	if foundDecrypted == nil {
		t.Fatalf("Test server was not returned by GetSavedServers")
	}
	if foundDecrypted.Password != testPassword {
		t.Errorf("Password mismatch after decryption. Expected %s, got %s", testPassword, foundDecrypted.Password)
	}
	if foundDecrypted.SSHKey != testKey {
		t.Errorf("SSH key mismatch after decryption. Expected %s, got %s", testKey, foundDecrypted.SSHKey)
	}

	// 6. Verify prod file was NOT touched
	if prodExisted {
		if stat, err := os.Stat(prodPath); err != nil || stat.ModTime().UnixNano() != prodModTime {
			t.Errorf("CRITICAL: Production servers.json in remnawave was modified during beta test!")
		}
	}

	// 7. Clean up test server
	if err := DeleteServer(testID); err != nil {
		t.Errorf("Failed to delete test server: %v", err)
	}

	// Verify deletion
	loadedAfterDelete := GetSavedServers()
	for i := range loadedAfterDelete {
		if loadedAfterDelete[i].ID == testID {
			t.Errorf("Test server still present after DeleteServer")
		}
	}

	t.Log("PASSED: Storage isolation and DPAPI encryption verified 100%!")
}
