package admin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"remna-ui/internal/types"
)

func TestLiveServerStatus(t *testing.T) {
	appData := os.Getenv("APPDATA")
	cfgPath := filepath.Join(appData, "remnawave", "servers.json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Skipf("No servers.json found: %v", err)
	}

	var servers []types.SavedServer
	if err := json.Unmarshal(data, &servers); err != nil || len(servers) == 0 {
		t.Skipf("No servers configured")
	}

	server := servers[0]
	res := GetServerStatus(server, "")
	if !res.Success {
		t.Skipf("Live server unreachable: %s", res.Error)
	}

	t.Logf("Host info: %s", res.HostInfo)
	t.Logf("Containers found: %d", len(res.Containers))
	for _, c := range res.Containers {
		t.Logf("  Container: %s (%s, running=%v)", c.Name, c.Status, c.Running)
	}
	t.Logf("Metrics: RAM=%d/%dMB (%d%%), Disk=%s/%s (%d%%), Uptime=%s",
		res.Metrics.RAMUsedMB, res.Metrics.RAMTotalMB, res.Metrics.RAMPercent,
		res.Metrics.DiskUsed, res.Metrics.DiskTotal, res.Metrics.DiskPercent,
		res.Metrics.Uptime)

	if len(res.Containers) == 0 {
		t.Errorf("Expected containers to be running")
	}
}

func TestCheckNodeLogs(t *testing.T) {
	appData := os.Getenv("APPDATA")
	cfgPath := filepath.Join(appData, "remnawave", "servers.json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Skipf("No servers.json found: %v", err)
	}
	var servers []types.SavedServer
	if err := json.Unmarshal(data, &servers); err != nil || len(servers) == 0 {
		t.Skipf("No servers configured")
	}
	server := servers[0]

	logs := GetContainerLogs(server, "remnanode")
	t.Logf("REMNANODE LOGS:\n%s", logs)

	client, err := getSSHClient(server)
	if err != nil {
		t.Skipf("Live server unreachable: %v", err)
	}
	defer client.Close()
	sess, _ := client.NewSession()
	defer sess.Close()
	composeContent, _ := sess.CombinedOutput("cat /opt/remnawave/node/docker-compose.yml 2>/dev/null || cat /opt/remnanode/docker-compose.yml 2>/dev/null")
	t.Logf("COMPOSE CONTENT:\n%s", string(composeContent))
}

func TestParseNodeComposeInput(t *testing.T) {
	// Case 1: Full docker-compose
	fullCompose := `services:
  remnanode:
    image: remnawave/node:latest
    environment:
      - SECRET_KEY="ey..."`
	res := parseNodeComposeInput(fullCompose)
	if res != fullCompose {
		t.Errorf("Expected full compose to be returned as-is")
	}

	// Case 2: SECRET_KEY="..."
	envLine := `SECRET_KEY="eyJtest123"`
	res = parseNodeComposeInput(envLine)
	if !strings.Contains(res, `SECRET_KEY="eyJtest123"`) {
		t.Errorf("Expected extracted SECRET_KEY in compose: %s", res)
	}

	// Case 3: Raw base64 string
	rawKey := `eyJsomeBase64Key`
	res = parseNodeComposeInput(rawKey)
	if !strings.Contains(res, `SECRET_KEY="eyJsomeBase64Key"`) {
		t.Errorf("Expected raw key to be wrapped in SECRET_KEY: %s", res)
	}
}

func TestCheckServerFeatures(t *testing.T) {
	appData := os.Getenv("APPDATA")
	cfgPath := filepath.Join(appData, "remnawave", "servers.json")
	data, _ := os.ReadFile(cfgPath)
	var servers []types.SavedServer
	json.Unmarshal(data, &servers)
	if len(servers) == 0 {
		t.Skip("No servers")
	}
	server := servers[0]

	client, err := getSSHClient(server)
	if err != nil {
		t.Fatalf("SSH fail: %v", err)
	}
	defer client.Close()

	sess, _ := client.NewSession()
	defer sess.Close()
	cmd := `
docker restart remnanode >/dev/null 2>&1
sleep 4
echo "=== REMNANODE LOGS AFTER RESTART ==="
docker logs remnanode --tail 35 2>&1
`
	out, _ := sess.CombinedOutput(cmd)
	t.Logf("REMNANODE STARTUP LOGS:\n%s", string(out))
}

func TestNewFeaturesEgamesParity(t *testing.T) {
	appData := os.Getenv("APPDATA")
	cfgPath := filepath.Join(appData, "remnawave", "servers.json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Skipf("No servers.json: %v", err)
	}
	var servers []types.SavedServer
	json.Unmarshal(data, &servers)
	if len(servers) == 0 {
		t.Skip("No servers")
	}
	server := servers[0]

	// 1. Check initial status
	t.Log("Testing GetServerStatus...")
	status := GetServerStatus(server, "")
	if !status.Success {
		t.Fatalf("GetServerStatus failed: %s", status.Error)
	}
	t.Logf("Latency: %d ms, BBR Active: %v, Mirrors Active: %v, Backups: %d",
		status.LatencyMs, status.BBRActive, status.DockerMirrorsActive, len(status.Backups))

	// 2. Test BBR Optimization
	t.Log("Testing OptimizeNetworkBBR...")
	bbrRes := OptimizeNetworkBBR(server)
	if !bbrRes.Success {
		t.Errorf("OptimizeNetworkBBR failed: %s", bbrRes.Error)
	} else {
		t.Logf("BBR Result: %s", bbrRes.Output)
	}

	// 3. Test Docker Mirrors Configuration
	t.Log("Testing ConfigureDockerMirrors...")
	mirrorsRes := ConfigureDockerMirrors(server)
	if !mirrorsRes.Success {
		t.Errorf("ConfigureDockerMirrors failed: %s", mirrorsRes.Error)
	} else {
		t.Logf("Mirrors Result: %s", mirrorsRes.Output)
	}

	// 4. Test Backup Creation
	t.Log("Testing CreateBackup...")
	backupRes := CreateBackup(server)
	if !backupRes.Success {
		t.Errorf("CreateBackup failed: %s", backupRes.Error)
	} else {
		t.Logf("Backup Result: %s", backupRes.Output)
	}

	// 5. Verify updated status
	t.Log("Verifying updated status...")
	statusAfter := GetServerStatus(server, "")
	t.Logf("AFTER: Latency: %d ms, BBR Active: %v, Mirrors Active: %v, Backups count: %d",
		statusAfter.LatencyMs, statusAfter.BBRActive, statusAfter.DockerMirrorsActive, len(statusAfter.Backups))
	for _, b := range statusAfter.Backups {
		t.Logf("  Backup archive: %s (%s, %s)", b.FileName, b.Size, b.CreatedAt)
	}

	if !statusAfter.BBRActive {
		t.Errorf("Expected BBR to be active after optimization")
	}
	if !statusAfter.DockerMirrorsActive {
		t.Errorf("Expected Docker mirrors to be active after configuration")
	}
	if len(statusAfter.Backups) == 0 {
		t.Errorf("Expected at least 1 backup archive to be listed")
	}
}

func TestAntiAbuseToggle(t *testing.T) {
	appData := os.Getenv("APPDATA")
	cfgPath := filepath.Join(appData, "remnawave", "servers.json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Skipf("No servers.json: %v", err)
	}
	var servers []types.SavedServer
	json.Unmarshal(data, &servers)
	if len(servers) == 0 {
		t.Skip("No servers")
	}
	server := servers[0]

	status := GetServerStatus(server, "")
	if !status.Success {
		t.Fatalf("GetServerStatus failed: %s", status.Error)
	}
	t.Logf("Initial AntiAbuseActive: %v, NodePresent: %v", status.AntiAbuseActive, status.NodePresent)
	if !status.NodePresent {
		t.Skip("No node present on server, skipping anti-abuse test")
	}

	t.Log("Testing ToggleAntiAbuse (enable=true)...")
	res := ToggleAntiAbuse(server, true)
	if !res.Success {
		t.Fatalf("ToggleAntiAbuse(enable=true) failed: %s", res.Error)
	}
	t.Logf("ToggleAntiAbuse output: %s", res.Output)

	statusAfter := GetServerStatus(server, "")
	if !statusAfter.AntiAbuseActive {
		t.Errorf("Expected AntiAbuseActive to be true after enabling")
	}
	t.Logf("Verified AntiAbuseActive after enable: %v", statusAfter.AntiAbuseActive)
}

func TestVersionComparison(t *testing.T) {
	tests := []struct {
		current string
		latest  string
		want    bool
	}{
		{"3.0.0", "3.0.1", true},
		{"3.0.0", "3.1.0", true},
		{"3.0.0", "4.0.0", true},
		{"3.0.0", "3.0.0", false},
		{"3.0.0", "2.9.9", false},
		{"v3.0.0", "v3.0.5", true},
		{"v3.1.2", "v3.1.1", false},
	}

	for _, tt := range tests {
		got := isVersionNewer(tt.latest, tt.current)
		if got != tt.want {
			t.Errorf("isVersionNewer(latest=%q, current=%q) = %v; want %v", tt.latest, tt.current, got, tt.want)
		}
	}
}



