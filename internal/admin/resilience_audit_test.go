package admin

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"remna-ui/internal/types"
)

func TestResilience_ConcurrentStorageAccess(t *testing.T) {
	// Isolate to temp test directory
	tempDir, err := os.MkdirTemp("", "remna_store_concurrency_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	origEnv := os.Getenv("REMNAWAVE_DATA_DIR")
	defer os.Setenv("REMNAWAVE_DATA_DIR", origEnv)

	// Set temporary subdir inside APPDATA
	testSubDir := filepath.Base(tempDir)
	os.Setenv("REMNAWAVE_DATA_DIR", testSubDir)

	var wg sync.WaitGroup
	workers := 15
	opsPerWorker := 10

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < opsPerWorker; i++ {
				srv := types.SavedServer{
					ID:       fmt.Sprintf("srv-%d-%d", workerID, i),
					Label:    fmt.Sprintf("Worker Server %d-%d", workerID, i),
					Host:     fmt.Sprintf("10.0.%d.%d", workerID, i),
					Port:     22,
					Username: "root",
					Password: fmt.Sprintf("Pass_%d_%d", workerID, i),
				}
				if err := SaveServer(srv); err != nil {
					t.Errorf("Worker %d failed to save server: %v", workerID, err)
				}
				servers := GetSavedServers()
				if len(servers) == 0 {
					t.Errorf("Worker %d got 0 servers after saving", workerID)
				}
				time.Sleep(2 * time.Millisecond)
			}
		}(w)
	}

	wg.Wait()

	finalList := GetSavedServers()
	expectedCount := workers * opsPerWorker
	if len(finalList) != expectedCount {
		t.Errorf("Expected %d servers in storage, got %d", expectedCount, len(finalList))
	}

	// Clean up by deleting half
	for i := 0; i < expectedCount/2; i++ {
		if err := DeleteServer(finalList[i].ID); err != nil {
			t.Errorf("Failed to delete server %s: %v", finalList[i].ID, err)
		}
	}

	remaining := GetSavedServers()
	if len(remaining) != expectedCount-(expectedCount/2) {
		t.Errorf("Expected %d remaining servers, got %d", expectedCount-(expectedCount/2), len(remaining))
	}
}

func TestResilience_OfflineServerHandling(t *testing.T) {
	offlineServer := types.SavedServer{
		ID:       "offline-test",
		Host:     "192.0.2.1", // Non-routable test network
		Port:     2222,
		Username: "root",
		Password: "testpassword",
	}

	// Verify GetServerStatus handles timeout gracefully without crashing
	res := GetServerStatus(offlineServer, "")
	if res.Success {
		t.Errorf("Expected offline server to return Success: false")
	}
	if res.Error == "" {
		t.Errorf("Expected non-empty error message for offline server")
	}
}

func TestResilience_InvalidInputFailsFast(t *testing.T) {
	dummyServer := types.SavedServer{
		Host:     "127.0.0.1",
		Port:     22,
		Username: "root",
	}

	// 1. RestartContainer with invalid name
	resRestart := RestartContainer(dummyServer, "invalid; name")
	if resRestart.Success || resRestart.Error == "" {
		t.Errorf("RestartContainer should fail fast on invalid container name")
	}

	// 2. RestoreBackup with invalid name
	resRestore := RestoreBackup(dummyServer, "../../etc/shadow")
	if resRestore.Success || resRestore.Error == "" {
		t.Errorf("RestoreBackup should fail fast on path traversal")
	}

	// 3. DownloadBackupStream with invalid name
	resDownload := DownloadBackupStream(dummyServer, "bad_backup.zip", "local.zip")
	if resDownload.Success || resDownload.Error == "" {
		t.Errorf("DownloadBackupStream should fail fast on non .tar.gz name")
	}
}
