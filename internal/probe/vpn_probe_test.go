package probe

import (
	"errors"
	"testing"
)

func TestIsRSTError(t *testing.T) {
	cases := []struct {
		err      error
		expected bool
	}{
		{nil, false},
		{errors.New("connection refused"), false},
		{errors.New("read: connection reset by peer"), true},
		{errors.New("wsarecv: An existing connection was forcibly closed by the remote host."), true},
		{errors.New("syscall: WSAECONNRESET (10054)"), true},
		{errors.New("write: broken pipe"), true},
		{errors.New("i/o timeout"), false},
	}

	for _, c := range cases {
		actual := isRSTError(c.err)
		if actual != c.expected {
			t.Errorf("isRSTError(%v) = %v; expected %v", c.err, actual, c.expected)
		}
	}
}

func TestFormatTLSVersion(t *testing.T) {
	if formatTLSVersion(0x0304) != "TLS 1.3" {
		t.Errorf("Expected TLS 1.3, got %s", formatTLSVersion(0x0304))
	}
	if formatTLSVersion(0x0303) != "TLS 1.2" {
		t.Errorf("Expected TLS 1.2, got %s", formatTLSVersion(0x0303))
	}
}

func TestProbeVPNNodeTraffic_EmptyHost(t *testing.T) {
	res := ProbeVPNNodeTraffic("", 443, "", "", nil)
	if res.Status != "offline" {
		t.Errorf("Expected offline status for empty host, got %s", res.Status)
	}
}

func TestProbeVPNNodeTraffic_MockEgress(t *testing.T) {
	mockRunner := func() (bool, int, int, string) {
		return true, 204, 85, ""
	}

	// 127.0.0.1:0 is guaranteed to fail TCP dial fast
	res := ProbeVPNNodeTraffic("127.0.0.1:0", 0, "example.com", "", mockRunner)
	if res.Status != "offline" {
		t.Errorf("Expected offline status for closed port, got %s", res.Status)
	}
	if res.TCPConnected {
		t.Errorf("Expected TCPConnected to be false for port 0")
	}
}
