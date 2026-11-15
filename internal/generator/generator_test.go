package generator

import (
	"strings"
	"testing"
)

func TestRenderPanelEnvContainsTrustProxy(t *testing.T) {
	env := RenderPanelEnv("panel.example.com", "sub.example.com", true)
	if !strings.Contains(env, "TRUST_PROXY=true") {
		t.Errorf("Expected TRUST_PROXY=true in panel .env, got: %s", env)
	}
	if !strings.Contains(env, "PANEL_DOMAIN=panel.example.com") {
		t.Errorf("Expected PANEL_DOMAIN=panel.example.com in panel .env")
	}
}

func TestRenderPanelComposeContainsHealthchecks(t *testing.T) {
	compose := RenderPanelCompose()
	if !strings.Contains(compose, "condition: service_healthy") {
		t.Errorf("Expected depends_on with condition: service_healthy in docker-compose.yml")
	}
	if !strings.Contains(compose, "pg_isready") {
		t.Errorf("Expected pg_isready healthcheck for remnawave-db")
	}
	if !strings.Contains(compose, "valkey-cli") {
		t.Errorf("Expected valkey-cli healthcheck for remnawave-redis")
	}
	if !strings.Contains(compose, "max-size: \"20m\"") {
		t.Errorf("Expected log rotation limit in docker-compose.yml")
	}
}

func TestRenderSubscriptionEnvContainsTrustProxy(t *testing.T) {
	env := RenderSubscriptionEnv("panel.example.com", "mytoken", "bundled")
	if !strings.Contains(env, "TRUST_PROXY=true") {
		t.Errorf("Expected TRUST_PROXY=true in subscription .env")
	}
	if !strings.Contains(env, "REMNAWAVE_URL=http://remnawave:3000") {
		t.Errorf("Expected internal bundled URL in subscription .env")
	}
	if !strings.Contains(env, "REMNAWAVE_PANEL_URL=http://remnawave:3000") {
		t.Errorf("Expected REMNAWAVE_PANEL_URL in subscription .env")
	}
}

func TestRenderNodeComposeContainsLogRotation(t *testing.T) {
	node := RenderNodeCompose(2222, "secret123")
	if !strings.Contains(node, "NODE_PORT: \"2222\"") {
		t.Errorf("Expected NODE_PORT 2222 in node compose")
	}
	if !strings.Contains(node, "max-size: \"20m\"") {
		t.Errorf("Expected log rotation in node compose")
	}
}

func TestRenderCaddyfileWithGateAndCamo(t *testing.T) {
	caddy := RenderCaddyfile("panel.example.com", "sub.example.com", true, "secret_gate", true)
	if !strings.Contains(caddy, "panel.example.com") {
		t.Errorf("Expected panel.example.com in Caddyfile")
	}
	if !strings.Contains(caddy, "sub.example.com") {
		t.Errorf("Expected sub.example.com in Caddyfile")
	}
	if !strings.Contains(caddy, "remna_gate=secret_gate") {
		t.Errorf("Expected remna_gate cookie setter in Caddyfile")
	}
	if !strings.Contains(caddy, "file_server") {
		t.Errorf("Expected camouflage file_server in Caddyfile")
	}
}
