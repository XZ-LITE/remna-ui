package ssh

import (
	"fmt"
	"net"
	"testing"
)

func TestSSHAudit_QuoteCmd(t *testing.T) {
	testCases := []struct {
		input string
		want  string
	}{
		{"", "''"},
		{"ls", "'ls'"},
		{"echo hello world", "'echo hello world'"},
		{"don't stop", "'don'\\''t stop'"},
		{"\"double quotes\"", "'\"double quotes\"'"},
		{"$VAR and `date` and $(whoami)", "'$VAR and `date` and $(whoami)'"},
		{"line1\nline2", "'line1\nline2'"},
	}

	for _, tc := range testCases {
		got := quoteCmd(tc.input)
		if got != tc.want {
			t.Errorf("quoteCmd(%q) = %q; want %q", tc.input, got, tc.want)
		}
	}
}

func TestSSHAudit_ClassifyStderrLine(t *testing.T) {
	infoLines := []string{
		"Extracting [==================================================>]  10.2MB/10.2MB",
		"Download complete",
		"Get:1 http://archive.ubuntu.com/ubuntu jammy InRelease [270 kB]",
		"Setting up docker-ce (5:24.0.7-1~ubuntu.22.04~jammy) ...",
		"debconf: unable to initialize frontend: Dialog",
		"WARNING: apt-key is deprecated. Manage keyring files in trusted.gpg.d instead.",
	}
	for _, l := range infoLines {
		status := classifyStderrLine(l)
		if status != "info" {
			t.Errorf("classifyStderrLine(%q) = %q; want 'info'", l, status)
		}
	}

	errorLines := []string{
		"fatal: could not read Username for 'https://github.com': No such device or address",
		"failed to solve with frontend dockerfile.v0",
		"Permission denied (publickey).",
		"Error: Address already in use",
		"Command not found: docker-compose",
	}
	for _, l := range errorLines {
		status := classifyStderrLine(l)
		if status != "error" {
			t.Errorf("classifyStderrLine(%q) = %q; want 'error'", l, status)
		}
	}

	regularLines := []string{
		"Some standard diagnostic output from a process",
		"Connecting to remote host...",
	}
	for _, l := range regularLines {
		status := classifyStderrLine(l)
		if status != "stderr" {
			t.Errorf("classifyStderrLine(%q) = %q; want 'stderr'", l, status)
		}
	}
}

func TestSSHAudit_IPv6AddressFormatting(t *testing.T) {
	ipv4 := "192.0.2.1"
	port := 22
	addr4 := net.JoinHostPort(ipv4, fmt.Sprintf("%d", port))
	if addr4 != "192.0.2.1:22" {
		t.Errorf("IPv4 JoinHostPort = %q; want '192.0.2.1:22'", addr4)
	}

	ipv6 := "2001:db8::1"
	addr6 := net.JoinHostPort(ipv6, fmt.Sprintf("%d", port))
	if addr6 != "[2001:db8::1]:22" {
		t.Errorf("IPv6 JoinHostPort = %q; want '[2001:db8::1]:22'", addr6)
	}
}

func TestSSHAudit_ParseSSHAuth(t *testing.T) {
	// Empty password and key -> empty
	auths := parseSSHAuth("", "")
	if len(auths) != 0 {
		t.Errorf("Expected 0 auth methods for empty password and key, got %d", len(auths))
	}

	// Password only -> 1 auth method
	auths = parseSSHAuth("secretPassword", "")
	if len(auths) != 1 {
		t.Errorf("Expected 1 auth method for password only, got %d", len(auths))
	}

	// Non-existent key file + password -> fallback to password (1 auth method)
	auths = parseSSHAuth("secretPassword", "C:/non_existent_key_file_12345.pem")
	if len(auths) != 1 {
		t.Errorf("Expected fallback to 1 password auth method for invalid key, got %d", len(auths))
	}
}
