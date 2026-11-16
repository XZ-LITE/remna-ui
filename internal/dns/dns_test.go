package dns

import (
	"testing"
)

func TestNormalizeDomain(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://panel.example.com", "panel.example.com"},
		{"http://panel.example.com:8443/api", "panel.example.com"},
		{"www.google.com", "google.com"},
		{"  Sub.Domain.COM  ", "sub.domain.com"},
		{"localhost", "localhost"},
	}

	for _, tt := range tests {
		got := NormalizeDomain(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeDomain(%q) = %q; want %q", tt.input, got, tt.want)
		}
	}
}

func TestCheckDNS_Local(t *testing.T) {
	res := CheckDNS("localhost")
	if res.Resolved {
		t.Errorf("expected localhost to be unresolved, got: %+v", res)
	}

	resEmpty := CheckDNS("")
	if resEmpty.Resolved {
		t.Errorf("expected empty string to be unresolved")
	}
}
