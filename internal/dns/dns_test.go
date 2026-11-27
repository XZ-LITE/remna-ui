package dns

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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

func TestCheckDNS_SystemAndDoHMatch(t *testing.T) {
	res := checkDNSWithDeps(
		"panel.example.com",
		func(ctx context.Context, host string) ([]string, error) {
			return []string{"203.0.113.10"}, nil
		},
		func(ctx context.Context, host string) ([]string, error) {
			return []string{"198.51.100.2", "203.0.113.10"}, nil
		},
		func(ctx context.Context, server, host string) ([]string, error) {
			return nil, errors.New("should not call UDP fallback")
		},
	)

	if !res.Resolved || res.Spoofed || res.IP != "203.0.113.10" || res.DoHIP != "203.0.113.10" {
		t.Fatalf("unexpected DNS match result: %+v", res)
	}
}

func TestCheckDNS_DetectsSpoofing(t *testing.T) {
	res := checkDNSWithDeps(
		"panel.example.com",
		func(ctx context.Context, host string) ([]string, error) {
			return []string{"127.0.0.1"}, nil
		},
		func(ctx context.Context, host string) ([]string, error) {
			return []string{"203.0.113.55"}, nil
		},
		func(ctx context.Context, server, host string) ([]string, error) {
			return nil, errors.New("unused")
		},
	)

	if !res.Resolved || !res.Spoofed || res.IP != "127.0.0.1" || res.DoHIP != "203.0.113.55" {
		t.Fatalf("expected spoof detection, got: %+v", res)
	}
	if !strings.Contains(res.Message, "203.0.113.55") {
		t.Errorf("expected warning message to mention real DoH IP, got: %s", res.Message)
	}
}

func TestCheckDNS_DoHFallbackWhenSystemFails(t *testing.T) {
	res := checkDNSWithDeps(
		"node.example.com",
		func(ctx context.Context, host string) ([]string, error) {
			return nil, errors.New("no such host")
		},
		func(ctx context.Context, host string) ([]string, error) {
			return []string{"198.51.100.44"}, nil
		},
		func(ctx context.Context, server, host string) ([]string, error) {
			return nil, errors.New("unused")
		},
	)

	if !res.Resolved || res.Spoofed || res.IP != "198.51.100.44" || res.DoHIP != "198.51.100.44" {
		t.Fatalf("expected DoH fallback success, got: %+v", res)
	}
}

func TestLookupDoH_HTTPMock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/dns-json" {
			t.Errorf("missing Accept: application/dns-json header")
		}
		w.Header().Set("Content-Type", "application/dns-json")
		_, _ = w.Write([]byte(`{"Status":0,"Answer":[{"name":"test.example.","type":1,"TTL":60,"data":"93.184.216.34"}]}`))
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ips, err := lookupDoH(ctx, srv.Client(), []string{srv.URL}, "test.example")
	if err != nil {
		t.Fatalf("lookupDoH failed: %v", err)
	}
	if len(ips) != 1 || ips[0] != "93.184.216.34" {
		t.Fatalf("unexpected DoH IPs: %v", ips)
	}
}
