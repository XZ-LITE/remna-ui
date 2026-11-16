package dns

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"remna-ui/internal/types"
)

// NormalizeDomain strips protocol schemes, ports, and trailing paths.
func NormalizeDomain(raw string) string {
	d := strings.TrimSpace(raw)
	d = strings.TrimPrefix(d, "https://")
	d = strings.TrimPrefix(d, "http://")
	if idx := strings.Index(d, "/"); idx != -1 {
		d = d[:idx]
	}
	if idx := strings.Index(d, ":"); idx != -1 {
		d = d[:idx]
	}
	d = strings.TrimPrefix(d, "www.")
	return strings.ToLower(strings.TrimSpace(d))
}

// CheckDNS performs a resilient two-tier lookup: first via system resolver, then fallback to public DNS.
func CheckDNS(domain string) types.DNSResult {
	clean := NormalizeDomain(domain)
	if clean == "" || clean == "localhost" || clean == "local" || clean == "127.0.0.1" {
		return types.DNSResult{
			Resolved: false,
			IP:       "",
			Message:  "Локальный или пустой адрес",
		}
	}

	// 1. First attempt: Quick check using the local system resolver (1200ms)
	ctxSys, cancelSys := context.WithTimeout(context.Background(), 1200*time.Millisecond)
	ips, err := net.DefaultResolver.LookupIP(ctxSys, "ip", clean)
	cancelSys()

	if err == nil && len(ips) > 0 {
		ipStr := ips[0].String()
		return types.DNSResult{
			Resolved: true,
			IP:       ipStr,
			Message:  fmt.Sprintf("Домен %s успешно указывает на IP: %s", clean, ipStr),
		}
	}

	// 2. Fallback attempt: Direct lookup via public DNS (Cloudflare 1.1.1.1, Google 8.8.8.8)
	// Bypasses poisoned, delayed, or stale local ISP/router DNS caches.
	fallbackServers := []string{"1.1.1.1:53", "8.8.8.8:53"}
	for _, dnsServer := range fallbackServers {
		ctxFb, cancelFb := context.WithTimeout(context.Background(), 2000*time.Millisecond)
		customResolver := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{Timeout: 1500 * time.Millisecond}
				return d.DialContext(ctx, network, dnsServer)
			},
		}
		fbIPs, fbErr := customResolver.LookupIP(ctxFb, "ip", clean)
		cancelFb()

		if fbErr == nil && len(fbIPs) > 0 {
			ipStr := fbIPs[0].String()
			return types.DNSResult{
				Resolved: true,
				IP:       ipStr,
				Message:  fmt.Sprintf("Домен %s успешно указывает на IP: %s (через публичный DNS)", clean, ipStr),
			}
		}
	}

	return types.DNSResult{
		Resolved: false,
		IP:       "",
		Message:  fmt.Sprintf("Домен %s пока не привязан к IP в DNS. Создайте A-запись у вашего регистратора или в Cloudflare.", clean),
	}
}
