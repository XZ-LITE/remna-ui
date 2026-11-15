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

// CheckDNS performs a quick, non-blocking lookup of the domain's A-record.
func CheckDNS(domain string) types.DNSResult {
	clean := NormalizeDomain(domain)
	if clean == "" || clean == "localhost" || clean == "local" || clean == "127.0.0.1" {
		return types.DNSResult{
			Resolved: false,
			IP:       "",
			Message:  "Локальный или пустой адрес",
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	var resolver net.Resolver
	ips, err := resolver.LookupIP(ctx, "ip4", clean)
	if err != nil || len(ips) == 0 {
		return types.DNSResult{
			Resolved: false,
			IP:       "",
			Message:  fmt.Sprintf("Домен %s пока не привязан к IP в DNS. Создайте A-запись у вашего регистратора или в Cloudflare.", clean),
		}
	}

	ipStr := ips[0].String()
	return types.DNSResult{
		Resolved: true,
		IP:       ipStr,
		Message:  fmt.Sprintf("Домен %s успешно указывает на IP: %s", clean, ipStr),
	}
}
