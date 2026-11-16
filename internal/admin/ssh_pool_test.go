package admin

import (
	"testing"
	"time"

	"remna-ui/internal/types"
)

func TestSSHPool_KeyGeneration(t *testing.T) {
	s1 := types.SavedServer{
		Host:     "1.2.3.4",
		Port:     22,
		Username: "root",
	}
	key1 := getSSHKey(s1)
	if key1 != "1.2.3.4:22:root" {
		t.Errorf("expected 1.2.3.4:22:root, got %s", key1)
	}

	sDefaultPort := types.SavedServer{
		Host: "example.com",
	}
	key2 := getSSHKey(sDefaultPort)
	if key2 != "example.com:22:root" {
		t.Errorf("expected example.com:22:root, got %s", key2)
	}
}

func TestSSHPool_ConcurrentMapAccess(t *testing.T) {
	// Verify thread-safety of sshPool under concurrent read/writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 50; j++ {
				sshPoolMu.Lock()
				sshPool["test-key"] = &pooledSSHClient{
					client:   nil,
					lastUsed: time.Now(),
				}
				_ = sshPool["test-key"]
				sshPoolMu.Unlock()
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	sshPoolMu.Lock()
	delete(sshPool, "test-key")
	sshPoolMu.Unlock()
}
