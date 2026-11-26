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
					inUse:    j % 3,
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

func TestShouldSkipKeepalive(t *testing.T) {
	now := time.Now()
	if shouldSkipKeepalive(time.Time{}, now) {
		t.Errorf("expected zero lastUsed not to skip keepalive")
	}
	if !shouldSkipKeepalive(now.Add(-3*time.Second), now) {
		t.Errorf("expected recent connection (3s) to skip keepalive")
	}
	if shouldSkipKeepalive(now.Add(-12*time.Second), now) {
		t.Errorf("expected older connection (12s) not to skip keepalive")
	}
}

func TestParseMasterCounts(t *testing.T) {
	cases := []struct {
		col8          string
		col9          string
		wantAntiAbuse bool
		wantNode      bool
	}{
		{"0|0", "", false, false},
		{"2|0", "", true, false},
		{"0|3", "", false, true},
		{"1|4\n", "", true, true},
		{"0", "2", false, true},
		{"1", "0", true, false},
	}

	for _, tc := range cases {
		ab, nd := parseMasterCounts(tc.col8, tc.col9)
		if ab != tc.wantAntiAbuse || nd != tc.wantNode {
			t.Errorf("parseMasterCounts(%q, %q) = (%v, %v), want (%v, %v)", tc.col8, tc.col9, ab, nd, tc.wantAntiAbuse, tc.wantNode)
		}
	}
}
