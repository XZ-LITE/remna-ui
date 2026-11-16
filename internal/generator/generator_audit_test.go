package generator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"remna-ui/internal/types"
)

func TestGeneratorAudit_FullStackDeploymentPlan(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "remna_gen_audit_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	plan := types.DeploymentPlan{
		Components:         []string{"panel", "node", "proxy", "subscription"},
		Domain:             "https://example.com/ ",
		PanelDomain:        "  PANEL.EXAMPLE.COM  ",
		SubscriptionDomain: "https://sub.example.com:443/",
		Proxy:              "caddy",
		SubscriptionMode:   "bundled",
		NodePort:                2222,
		PerformanceOptimization: true,
		OutputDir:               tempDir,
	}

	res := WriteOutputFiles(plan, tempDir)
	if !res.Success {
		t.Fatalf("WriteOutputFiles failed: %s", res.Error)
	}

	// 1. Verify manifest.json
	manifestPath := filepath.Join(tempDir, "manifest.json")
	mData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("Failed to read manifest.json: %v", err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(mData, &manifest); err != nil {
		t.Fatalf("Failed to parse manifest.json: %v", err)
	}
	if manifest["panel_domain"] != "panel.example.com" {
		t.Errorf("manifest panel_domain not normalized: got %v", manifest["panel_domain"])
	}
	if manifest["subscription_domain"] != "sub.example.com" {
		t.Errorf("manifest subscription_domain not normalized: got %v", manifest["subscription_domain"])
	}

	// 2. Verify panel .env and compose
	panelEnvPath := filepath.Join(tempDir, "panel", ".env")
	envData, err := os.ReadFile(panelEnvPath)
	if err != nil {
		t.Fatalf("Failed to read panel/.env: %v", err)
	}
	envStr := string(envData)
	if !strings.Contains(envStr, "PANEL_DOMAIN=panel.example.com") {
		t.Errorf("panel/.env missing normalized domain: %s", envStr)
	}
	if !strings.Contains(envStr, "TRUST_PROXY=true") {
		t.Errorf("panel/.env missing TRUST_PROXY=true")
	}

	panelComposePath := filepath.Join(tempDir, "panel", "docker-compose.yml")
	composeData, err := os.ReadFile(panelComposePath)
	if err != nil {
		t.Fatalf("Failed to read panel/docker-compose.yml: %v", err)
	}
	compStr := string(composeData)
	if !strings.Contains(compStr, "remnawave:") || !strings.Contains(compStr, "remnawave-db:") {
		t.Errorf("panel compose missing core services: %s", compStr)
	}

	// 3. Verify node docker-compose
	nodeComposePath := filepath.Join(tempDir, "node", "docker-compose.yml")
	nodeData, err := os.ReadFile(nodeComposePath)
	if err != nil {
		t.Fatalf("Failed to read node/docker-compose.yml: %v", err)
	}
	nodeStr := string(nodeData)
	if !strings.Contains(nodeStr, `NODE_PORT: "2222"`) {
		t.Errorf("node compose missing node port 2222: %s", nodeStr)
	}

	// 4. Verify Caddyfile
	caddyPath := filepath.Join(tempDir, "caddy", "Caddyfile")
	caddyData, err := os.ReadFile(caddyPath)
	if err != nil {
		t.Fatalf("Failed to read caddy/Caddyfile: %v", err)
	}
	caddyStr := string(caddyData)
	if !strings.Contains(caddyStr, "panel.example.com") || !strings.Contains(caddyStr, "sub.example.com") {
		t.Errorf("Caddyfile missing normalized domains: %s", caddyStr)
	}
}

func TestGeneratorAudit_NodeOnlyPlan(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "remna_gen_node_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	plan := types.DeploymentPlan{
		Components: []string{"node"},
		NodePort:   0, // Should default to 2222
		OutputDir:  tempDir,
	}

	res := WriteOutputFiles(plan, tempDir)
	if !res.Success {
		t.Fatalf("WriteOutputFiles failed: %s", res.Error)
	}

	// Panel should NOT exist
	panelEnvPath := filepath.Join(tempDir, "panel", ".env")
	if _, err := os.Stat(panelEnvPath); !os.IsNotExist(err) {
		t.Errorf("Expected panel/.env to NOT exist for node-only plan")
	}

	// Node should exist with default port 2222
	nodeData, err := os.ReadFile(filepath.Join(tempDir, "node", "docker-compose.yml"))
	if err != nil {
		t.Fatalf("Failed to read node compose: %v", err)
	}
	if !strings.Contains(string(nodeData), `NODE_PORT: "2222"`) {
		t.Errorf("Expected node compose to default to port 2222, got: %s", string(nodeData))
	}
}
