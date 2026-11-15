package admin

import (
	"strings"
	"testing"
)

func TestBuildAntiProbingScript(t *testing.T) {
	script := BuildAntiProbingScript("1.2.3.4", []int{8443, 2083}, true, true, true, 2222, 22)

	if !strings.Contains(script, "iptables -A REMNA_DOCKER -p tcp --dport 2222 -s 1.2.3.4 -j ACCEPT") {
		t.Errorf("expected script to allow panel IP on port 2222 in DOCKER-USER")
	}
	if !strings.Contains(script, "iptables -A REMNA_DOCKER -p tcp --dport 2222 -j DROP") {
		t.Errorf("expected script to drop foreign traffic to port 2222 in DOCKER-USER")
	}
	if !strings.Contains(script, "iptables -P INPUT DROP") {
		t.Errorf("expected stealth DROP policy on INPUT")
	}
	if !strings.Contains(script, "iptables -A REMNA_INPUT -p tcp --dport 8443 -j ACCEPT") {
		t.Errorf("expected client port 8443 to be open")
	}
	if !strings.Contains(script, "iptables -A REMNA_INPUT -p tcp --dport 2083 -j ACCEPT") {
		t.Errorf("expected client port 2083 to be open")
	}
	if !strings.Contains(script, "iptables -A REMNA_INPUT -p tcp --dport 22 -j ACCEPT") {
		t.Errorf("expected SSH port 22 to be open")
	}
	if !strings.Contains(script, "ip6tables -P INPUT DROP") {
		t.Errorf("expected IPv6 stealth DROP")
	}
}
