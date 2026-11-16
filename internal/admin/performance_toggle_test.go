package admin

import (
	"encoding/base64"
	"strings"
	"testing"

	"remna-ui/internal/generator"
	"remna-ui/internal/types"
)

func TestPerformance_ToggleScriptsAndPayload(t *testing.T) {
	// 1. Verify Optimized Payload Base64
	optCompose := generator.RenderPanelCompose()
	b64Opt := base64.StdEncoding.EncodeToString([]byte(optCompose))
	decodedOpt, err := base64.StdEncoding.DecodeString(b64Opt)
	if err != nil {
		t.Fatalf("Failed to decode base64 optimized compose: %v", err)
	}
	if !strings.Contains(string(decodedOpt), "shm_size: 512mb") {
		t.Errorf("Decoded optimized compose missing shm_size: 512mb")
	}

	// 2. Verify Standard Payload Base64
	stdCompose := generator.RenderPanelComposeStandard()
	b64Std := base64.StdEncoding.EncodeToString([]byte(stdCompose))
	decodedStd, err := base64.StdEncoding.DecodeString(b64Std)
	if err != nil {
		t.Fatalf("Failed to decode base64 standard compose: %v", err)
	}
	if strings.Contains(string(decodedStd), "shm_size: 512mb") {
		t.Errorf("Decoded standard compose should not contain shm_size: 512mb")
	}

	// 3. Verify Error on unreachable server
	srv := types.SavedServer{
		ID:       "test-unreachable",
		Label:    "Test Server",
		Host:     "127.0.0.1",
		Port:     65534,
		Username: "root",
		Password: "wrongpassword",
	}

	resEnable := ToggleStackPerformance(srv, true)
	if resEnable.Success {
		t.Errorf("Expected failure on unreachable server, got success")
	}
	if !strings.Contains(resEnable.Error, "Ошибка подключения SSH") {
		t.Errorf("Expected SSH connection error, got: %s", resEnable.Error)
	}

	resDisable := ToggleStackPerformance(srv, false)
	if resDisable.Success {
		t.Errorf("Expected failure on unreachable server, got success")
	}
	if !strings.Contains(resDisable.Error, "Ошибка подключения SSH") {
		t.Errorf("Expected SSH connection error, got: %s", resDisable.Error)
	}
}

func TestPerformance_ServerStatusResultField(t *testing.T) {
	status := types.ServerStatusResult{
		Success:              true,
		HostInfo:             "Ubuntu 24.04 LTS",
		BBRActive:            true,
		PerformanceOptimized: true,
	}

	if !status.PerformanceOptimized {
		t.Errorf("Expected PerformanceOptimized to be true")
	}
}
