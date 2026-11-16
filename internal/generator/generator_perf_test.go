package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"remna-ui/internal/types"
)

func TestGenerator_PerformanceTemplates(t *testing.T) {
	// 1. Verify Optimized Compose
	optCompose := RenderPanelCompose()
	if !strings.Contains(optCompose, "shm_size: 512mb") {
		t.Errorf("RenderPanelCompose missing 'shm_size: 512mb'")
	}
	if !strings.Contains(optCompose, "--unixsocket") || !strings.Contains(optCompose, "/var/run/valkey/valkey.sock") {
		t.Errorf("RenderPanelCompose missing Valkey unix socket configuration")
	}
	if !strings.Contains(optCompose, "--port") || !strings.Contains(optCompose, `"0"`) {
		t.Errorf("RenderPanelCompose missing --port 0 for Valkey")
	}
	if !strings.Contains(optCompose, "--maxmemory") || !strings.Contains(optCompose, "256mb") {
		t.Errorf("RenderPanelCompose missing --maxmemory 256mb for Valkey")
	}
	if !strings.Contains(optCompose, "ulimits:") || !strings.Contains(optCompose, "1048576") {
		t.Errorf("RenderPanelCompose missing 1M ulimits nofile")
	}
	if !strings.Contains(optCompose, "REDIS_SOCKET=/var/run/valkey/valkey.sock") {
		t.Errorf("RenderPanelCompose missing REDIS_SOCKET environment variable")
	}

	// 2. Verify Standard Compose (Rollback)
	stdCompose := RenderPanelComposeStandard()
	if strings.Contains(stdCompose, "shm_size: 512mb") {
		t.Errorf("RenderPanelComposeStandard must not contain shm_size: 512mb")
	}
	if strings.Contains(stdCompose, "/var/run/valkey/valkey.sock") {
		t.Errorf("RenderPanelComposeStandard must not contain unix socket path")
	}
	if !strings.Contains(stdCompose, "REDIS_HOST=remnawave-redis") || !strings.Contains(stdCompose, "REDIS_PORT=6379") {
		t.Errorf("RenderPanelComposeStandard missing standard REDIS_HOST/PORT fallback")
	}
	if !strings.Contains(stdCompose, "valkey-server") {
		t.Errorf("RenderPanelComposeStandard missing valkey-server command")
	}

	// 3. Verify Caddy compression
	caddy := RenderCaddyfile("panel.example.com", "sub.example.com", true, "admin@example.com", false)
	if !strings.Contains(caddy, "encode zstd gzip") {
		t.Errorf("RenderCaddyfile missing 'encode zstd gzip'")
	}

	// 4. Verify Nginx compression
	nginx := RenderNginxConf("panel.example.com", "sub.example.com", true, "admin@example.com", false)
	if !strings.Contains(nginx, "gzip on;") {
		t.Errorf("RenderNginxConf missing 'gzip on;'")
	}
}

func TestGenerator_DeploymentPlanPerformanceSwitch(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "remna_plan_perf_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Scenario A: PerformanceOptimization = true
	dirOpt := filepath.Join(tempDir, "opt")
	planOpt := types.DeploymentPlan{
		Components:              []string{"panel"},
		Domain:                  "example.com",
		PanelDomain:             "panel.example.com",
		PerformanceOptimization: true,
	}
	resOpt := WriteOutputFiles(planOpt, dirOpt)
	if !resOpt.Success {
		t.Fatalf("WriteOutputFiles (opt) failed: %v", resOpt.Error)
	}

	envOpt, _ := os.ReadFile(filepath.Join(dirOpt, "panel", ".env"))
	if !strings.Contains(string(envOpt), "REDIS_SOCKET=/var/run/valkey/valkey.sock") {
		t.Errorf("Optimized panel/.env missing REDIS_SOCKET")
	}
	compOpt, _ := os.ReadFile(filepath.Join(dirOpt, "panel", "docker-compose.yml"))
	if !strings.Contains(string(compOpt), "shm_size: 512mb") {
		t.Errorf("Optimized docker-compose.yml missing shm_size: 512mb")
	}

	// Scenario B: PerformanceOptimization = false
	dirStd := filepath.Join(tempDir, "std")
	planStd := types.DeploymentPlan{
		Components:              []string{"panel"},
		Domain:                  "example.com",
		PanelDomain:             "panel.example.com",
		PerformanceOptimization: false,
	}
	resStd := WriteOutputFiles(planStd, dirStd)
	if !resStd.Success {
		t.Fatalf("WriteOutputFiles (std) failed: %v", resStd.Error)
	}

	envStd, _ := os.ReadFile(filepath.Join(dirStd, "panel", ".env"))
	if !strings.Contains(string(envStd), "#REDIS_SOCKET=") || !strings.Contains(string(envStd), "REDIS_HOST=remnawave-redis") {
		t.Errorf("Standard panel/.env should have REDIS_HOST and commented REDIS_SOCKET: %s", string(envStd))
	}
	compStd, _ := os.ReadFile(filepath.Join(dirStd, "panel", "docker-compose.yml"))
	if strings.Contains(string(compStd), "shm_size: 512mb") {
		t.Errorf("Standard docker-compose.yml should not have shm_size: 512mb")
	}
}
