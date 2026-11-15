package admin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"remna-ui/internal/types"
)

func TestStorageUnifiedPathAndDPAPI(t *testing.T) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		t.Skip("APPDATA is not set")
	}

	// 1. Verify default store path resolves to remnawave
	expectedProdDir := filepath.Join(appData, "remnawave")
	expectedProdPath := filepath.Join(expectedProdDir, "servers.json")

	actualStorePath := getStorePath()
	if actualStorePath != expectedProdPath {
		t.Fatalf("Expected default store path %s, got %s", expectedProdPath, actualStorePath)
	}

	// 2. Test saving and DPAPI roundtrip
	testID := "audit-test-server-wails3"
	testPassword := "SuperSecretPassword123!"
	testKey := "-----BEGIN OPENSSH PRIVATE KEY-----\ntestkey\n-----END OPENSSH PRIVATE KEY-----"

	server := types.SavedServer{
		ID:       testID,
		Label:    "Wails3 Storage Test",
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

	// 3. Verify file on disk is in remnawave and has DPAPI encryption
	data, err := os.ReadFile(expectedProdPath)
	if err != nil {
		t.Fatalf("Failed to read servers.json: %v", err)
	}

	var rawList []types.SavedServer
	if err := json.Unmarshal(data, &rawList); err != nil {
		t.Fatalf("Failed to parse servers.json: %v", err)
	}

	var foundRaw *types.SavedServer
	for i := range rawList {
		if rawList[i].ID == testID {
			foundRaw = &rawList[i]
			break
		}
	}

	if foundRaw == nil {
		t.Fatalf("Test server was not found in raw servers.json")
	}

	if !strings.HasPrefix(foundRaw.Password, "enc:dpapi:") {
		t.Errorf("Password was not encrypted with DPAPI on disk: %s", foundRaw.Password)
	}
	if !strings.HasPrefix(foundRaw.SSHKey, "enc:dpapi:") {
		t.Errorf("SSH key was not encrypted with DPAPI on disk: %s", foundRaw.SSHKey)
	}

	// 4. Verify GetSavedServers() decrypts properly
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

	// 5. Clean up test server
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

	t.Log("PASSED: Unified storage path and DPAPI encryption verified 100%!")
}
