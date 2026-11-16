package admin

import (
	"strings"
	"testing"
)

func TestAntiProbingAudit_MultiIPWhitelist(t *testing.T) {
	panelIPs := "1.2.3.4, 5.6.7.8, 9.10.11.12"
	script := BuildAntiProbingScript(panelIPs, []int{8443}, true, true, true, 2222, 22022)

	// Verify all 3 IPs are whitelisted
	for _, ip := range []string{"1.2.3.4", "5.6.7.8", "9.10.11.12"} {
		expectedAccept := "iptables -A REMNA_DOCKER -p tcp --dport 2222 -s " + ip + " -j ACCEPT"
		if !strings.Contains(script, expectedAccept) {
			t.Errorf("Missing expected accept rule for panel IP %s in DOCKER-USER", ip)
		}
	}

	// Verify foreign DROP rule exists AFTER whitelist
	if !strings.Contains(script, "iptables -A REMNA_DOCKER -p tcp --dport 2222 -j DROP") {
		t.Errorf("Missing DROP rule for foreign scanners on port 2222")
	}

	// Verify custom SSH port 22022 is kept open
	if !strings.Contains(script, "iptables -A REMNA_INPUT -p tcp --dport 22022 -j ACCEPT") {
		t.Errorf("Custom SSH port 22022 was not preserved")
	}

	// Verify port 53 RST leak prevention
	if !strings.Contains(script, "iptables -A REMNA_INPUT ! -i lo -p tcp --dport 53 -j DROP") {
		t.Errorf("Missing port 53 TCP RST leak prevention")
	}
}

func TestAntiProbingAudit_NoPanelIP_NoLockout(t *testing.T) {
	// If panel IP is not specified, port 2222 must NOT be dropped to prevent locking out the master!
	script := BuildAntiProbingScript("", []int{443}, true, false, false, 2222, 22)

	if strings.Contains(script, "iptables -A REMNA_DOCKER -p tcp --dport 2222 -j DROP") {
		t.Errorf("CRITICAL: When panel IP is empty, port 2222 must NOT be dropped to avoid master panel lockout")
	}
}

func TestAntiProbingAudit_StealthDropPolicy(t *testing.T) {
	// With stealthDrop=true
	scriptDrop := BuildAntiProbingScript("1.2.3.4", nil, false, true, false, 2222, 22)
	if !strings.Contains(scriptDrop, "iptables -P INPUT DROP") {
		t.Errorf("Expected iptables -P INPUT DROP when stealthDrop=true")
	}

	// With stealthDrop=false
	scriptAccept := BuildAntiProbingScript("1.2.3.4", nil, false, false, false, 2222, 22)
	if strings.Contains(scriptAccept, "iptables -P INPUT DROP") {
		t.Errorf("Expected iptables -P INPUT to NOT be DROP when stealthDrop=false")
	}
}
