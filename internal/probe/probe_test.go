package probe

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"remna-ui/internal/types"
)

func TestClassifyResults_CleanIP(t *testing.T) {
	local := types.ProbeNodeResult{
		NodeID:      "local_pc",
		Success:     true,
		LatencyMs:   25,
		CountryCode: "local",
	}
	remote := []types.ProbeNodeResult{
		{NodeID: "ru1.node.check-host.net", Success: true, LatencyMs: 28, CountryCode: "ru"},
		{NodeID: "ru2.node.check-host.net", Success: true, LatencyMs: 32, CountryCode: "ru"},
		{NodeID: "ru3.node.check-host.net", Success: true, LatencyMs: 35, CountryCode: "ru"},
		{NodeID: "de1.node.check-host.net", Success: true, LatencyMs: 18, CountryCode: "de"},
	}

	res := classifyResults("1.2.3.4", 443, local, remote, "Hetzner (AS24940)", "Frankfurt, Germany", "Russia", false, append([]types.ProbeNodeResult{local}, remote...), "17.11.2026")

	if res.Status != "clean" {
		t.Errorf("Expected status clean, got %s", res.Status)
	}
	if res.IsRKNBlocked {
		t.Errorf("Expected IsRKNBlocked false, got true")
	}
}

func TestClassifyResults_TSPUBlocked(t *testing.T) {
	local := types.ProbeNodeResult{
		NodeID:      "local_pc",
		Success:     false,
		Error:       "Таймаут соединения",
		CountryCode: "local",
	}
	remote := []types.ProbeNodeResult{
		{NodeID: "ru1.node.check-host.net", Success: false, Error: "Таймаут (ТСПУ / Блокировка)", CountryCode: "ru"},
		{NodeID: "ru2.node.check-host.net", Success: false, Error: "Таймаут (ТСПУ / Блокировка)", CountryCode: "ru"},
		{NodeID: "ru3.node.check-host.net", Success: false, Error: "Таймаут (ТСПУ / Блокировка)", CountryCode: "ru"},
		{NodeID: "de1.node.check-host.net", Success: true, LatencyMs: 19, CountryCode: "de"},
	}

	res := classifyResults("1.2.3.4", 443, local, remote, "OVH SAS (AS16276)", "Gravelines, France", "Russia", false, append([]types.ProbeNodeResult{local}, remote...), "17.11.2026")

	if res.Status != "blocked" {
		t.Errorf("Expected status blocked, got %s", res.Status)
	}
	if !res.IsRKNBlocked {
		t.Errorf("Expected IsRKNBlocked true, got false")
	}
}

func TestClassifyResults_TSPUBlocked_WithUserVPN(t *testing.T) {
	// CRITICAL USER SCENARIO: User on PC has active VPN tunnel (e.g. exiting through Latvia/Germany).
	// Local TCP check succeeds through the VPN tunnel, BUT independent Russian probes in Moscow and SPb
	// all fail (timeout/refused), while German reference probe succeeds.
	local := types.ProbeNodeResult{
		NodeID:      "local_pc",
		Success:     true,
		LatencyMs:   38,
		CountryCode: "local",
		Location:    "Локальный ПК (VPN: Latvia)",
	}
	remote := []types.ProbeNodeResult{
		{NodeID: "ru1.node.check-host.net", Success: false, Error: "Таймаут (ТСПУ / Блокировка)", CountryCode: "ru"},
		{NodeID: "ru2.node.check-host.net", Success: false, Error: "Таймаут (ТСПУ / Блокировка)", CountryCode: "ru"},
		{NodeID: "ru3.node.check-host.net", Success: false, Error: "Таймаут (ТСПУ / Блокировка)", CountryCode: "ru"},
		{NodeID: "de1.node.check-host.net", Success: true, LatencyMs: 15, CountryCode: "de"},
	}

	res := classifyResults("1.2.3.4", 443, local, remote, "OVH SAS (AS16276)", "Gravelines, France", "Latvia", true, append([]types.ProbeNodeResult{local}, remote...), "17.11.2026")

	if res.Status != "blocked" {
		t.Fatalf("Expected status blocked even when user PC has VPN, got %s", res.Status)
	}
	if !res.IsRKNBlocked {
		t.Errorf("Expected IsRKNBlocked true, got false")
	}
	if !res.IsVPNDetected {
		t.Errorf("Expected IsVPNDetected true, got false")
	}
	if !strings.Contains(res.Summary, "VPN") {
		t.Errorf("Expected summary to explain that local success is due to active VPN, got: %s", res.Summary)
	}
}

func TestClassifyResults_PartialBlock(t *testing.T) {
	local := types.ProbeNodeResult{
		NodeID:      "local_pc",
		Success:     true,
		LatencyMs:   35,
		CountryCode: "local",
	}
	remote := []types.ProbeNodeResult{
		{NodeID: "ru1.node.check-host.net", Success: true, LatencyMs: 30, CountryCode: "ru"},
		{NodeID: "ru2.node.check-host.net", Success: false, Error: "Таймаут", CountryCode: "ru"},
		{NodeID: "ru3.node.check-host.net", Success: false, Error: "Таймаут", CountryCode: "ru"},
		{NodeID: "de1.node.check-host.net", Success: true, LatencyMs: 20, CountryCode: "de"},
	}

	res := classifyResults("1.2.3.4", 443, local, remote, "DigitalOcean (AS14061)", "Amsterdam, Netherlands", "Russia", false, append([]types.ProbeNodeResult{local}, remote...), "17.11.2026")

	if res.Status != "partial" {
		t.Errorf("Expected status partial, got %s", res.Status)
	}
	if !res.IsRKNBlocked {
		t.Errorf("Expected IsRKNBlocked true, got false")
	}
}

func TestClassifyResults_OfflineHost(t *testing.T) {
	local := types.ProbeNodeResult{
		NodeID:      "local_pc",
		Success:     false,
		Error:       "Таймаут соединения",
		CountryCode: "local",
	}
	remote := []types.ProbeNodeResult{
		{NodeID: "ru1.node.check-host.net", Success: false, Error: "Таймаут", CountryCode: "ru"},
		{NodeID: "ru2.node.check-host.net", Success: false, Error: "Таймаут", CountryCode: "ru"},
		{NodeID: "ru3.node.check-host.net", Success: false, Error: "Таймаут", CountryCode: "ru"},
		{NodeID: "de1.node.check-host.net", Success: false, Error: "Таймаут", CountryCode: "de"},
	}

	res := classifyResults("1.2.3.4", 443, local, remote, "Unknown", "Unknown", "Russia", false, append([]types.ProbeNodeResult{local}, remote...), "17.11.2026")

	if res.Status != "offline" {
		t.Errorf("Expected status offline, got %s", res.Status)
	}
	if res.IsRKNBlocked {
		t.Errorf("Expected IsRKNBlocked false, got true")
	}
}

func TestClassifyResults_FallbackLocalOnly(t *testing.T) {
	local := types.ProbeNodeResult{
		NodeID:      "local_pc",
		Success:     true,
		LatencyMs:   42,
		CountryCode: "local",
	}
	remote := []types.ProbeNodeResult{} // e.g. Rate limit on Check-Host

	res := classifyResults("1.2.3.4", 22, local, remote, "Hostkey (AS57043)", "Meppel, Netherlands", "Russia", false, []types.ProbeNodeResult{local}, "17.11.2026")

	if res.Status != "clean" {
		t.Errorf("Expected status clean from local probe fallback, got %s", res.Status)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestQueryGeoIP_MockRoundTripper(t *testing.T) {
	oldClient := httpClient
	defer func() { httpClient = oldClient }()

	httpClient = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			resp := ipAPIResponse{
				Status:      "success",
				Country:     "Finland",
				CountryCode: "FI",
				City:        "Helsinki",
				ISP:         "Hetzner Online GmbH",
				AS:          "AS24940 Hetzner Online GmbH",
			}
			data, _ := json.Marshal(resp)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(data)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	isp, loc := queryGeoIP("95.217.1.1")
	if isp != "Hetzner Online GmbH (AS24940)" {
		t.Errorf("Unexpected isp: %s", isp)
	}
	if loc != "Helsinki, Finland" {
		t.Errorf("Unexpected location: %s", loc)
	}
}
