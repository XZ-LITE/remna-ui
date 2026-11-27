package admin

import (
	"strings"
	"testing"

	"remna-ui/internal/types"
)

func TestBuildWarpInstallScript(t *testing.T) {
	script := buildWarpInstallScript()

	requiredSnippets := []string{
		`WARP_ENDPOINT_V4="162.159.192.1:2408"`,
		`MARK="51888"`,
		`Table = off`,
		`PersistentKeepalive = 25`,
		`AllowedIPs = 0.0.0.0/0`,
		`table inet $TABLE`,
		`meta mark $MARK oifname != "$IFACE" counter drop`,
		`warp-native-killswitch.service`,
		`RESTART_COOLDOWN=120`,
		`/etc/cron.d/warp-native`,
		`warp-cli --accept-tos mode proxy`,
	}

	for _, snip := range requiredSnippets {
		if !strings.Contains(script, snip) {
			t.Errorf("WARP install script missing expected snippet %q", snip)
		}
	}
}

func TestParseSelfStealTarget(t *testing.T) {
	srvWithDomain := types.SavedServer{
		Host:        "203.0.113.10",
		DecoyDomain: "https://mask.example.com/",
	}

	target, err := parseSelfStealTarget(srvWithDomain, "127.0.0.1:8443")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target.ListenHost != "127.0.0.1" || target.ListenPort != 8443 || target.SNIDomain != "mask.example.com" {
		t.Fatalf("unexpected parsed target: %+v", target)
	}

	// Custom port only
	targetPort, err := parseSelfStealTarget(srvWithDomain, "9443")
	if err != nil {
		t.Fatalf("unexpected port parse error: %v", err)
	}
	if targetPort.ListenPort != 9443 || targetPort.ListenHost != "127.0.0.1" {
		t.Fatalf("unexpected port-only target: %+v", targetPort)
	}

	// Plain domain passed via fallbackDomain when server.DecoyDomain is empty
	targetDomain, err := parseSelfStealTarget(types.SavedServer{Host: "203.0.113.10"}, "sub.mask.example.com")
	if err != nil {
		t.Fatalf("unexpected domain fallback error: %v", err)
	}
	if targetDomain.SNIDomain != "sub.mask.example.com" || targetDomain.ListenPort != 8443 {
		t.Fatalf("unexpected fallback domain target: %+v", targetDomain)
	}

	// Invalid shell chars in DecoyDomain must be rejected
	if _, err := parseSelfStealTarget(types.SavedServer{DecoyDomain: "bad.com;id"}, "127.0.0.1:8443"); err == nil {
		t.Errorf("expected error for invalid characters in DecoyDomain")
	}

	// Out-of-range port must be rejected
	if _, err := parseSelfStealTarget(srvWithDomain, "127.0.0.1:70000"); err == nil {
		t.Errorf("expected error for out-of-range port")
	}
}

func TestBuildSelfStealDeployScript(t *testing.T) {
	target := selfStealTarget{
		ListenHost: "127.0.0.1",
		ListenPort: 8443,
		SNIDomain:  "mask.example.com",
	}

	caddyfile := buildSelfStealCaddyfile(target)
	if !strings.Contains(caddyfile, "# managed-by: remna-ui SelfSteal") ||
		!strings.Contains(caddyfile, ":8443") ||
		!strings.Contains(caddyfile, "bind 127.0.0.1") {
		t.Fatalf("unexpected generated Caddyfile:\n%s", caddyfile)
	}

	script := buildSelfStealDeployScript(target, "ZHVtbXk=", "ZHVtbXk=", "ZHVtbXk=")
	expectedPieces := []string{
		"/opt/remnawave/selfsteal",
		".caddyfile.sha256",
		"caddy validate --config /etc/caddy/Caddyfile",
		"rollback_caddy",
		"docker compose up -d --force-recreate",
	}
	for _, piece := range expectedPieces {
		if !strings.Contains(script, piece) {
			t.Errorf("SelfSteal deploy script missing %q", piece)
		}
	}
}
