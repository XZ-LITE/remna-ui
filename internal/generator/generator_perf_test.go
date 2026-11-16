package generator

import (
	"strings"
	"testing"
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

	// 2. Verify Standard Compose (Rollback)
	stdCompose := RenderPanelComposeStandard()
	if strings.Contains(stdCompose, "shm_size: 512mb") {
		t.Errorf("RenderPanelComposeStandard must not contain shm_size: 512mb")
	}
	if strings.Contains(stdCompose, "/var/run/valkey/valkey.sock") {
		t.Errorf("RenderPanelComposeStandard must not contain unix socket path")
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
