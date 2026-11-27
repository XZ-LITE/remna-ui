package dns

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"remna-ui/internal/types"
)

var defaultDoHEndpoints = []string{
	"https://cloudflare-dns.com/dns-query",
	"https://dns.google/resolve",
}

type dohJSONResponse struct {
	Status int `json:"Status"`
	Answer []struct {
		Type int    `json:"type"`
		Data string `json:"data"`
	} `json:"Answer"`
}

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

// CheckDNS performs a multi-tier DNS lookup: system resolver + encrypted DoH verification
// to detect ISP DNS spoofing/stub blockpages, with public DNS fallback.
func CheckDNS(domain string) types.DNSResult {
	httpClient := &http.Client{Timeout: 2500 * time.Millisecond}
	sysLookup := func(ctx context.Context, host string) ([]string, error) {
		ips, err := net.DefaultResolver.LookupIP(ctx, "ip4", host)
		if err != nil {
			ips, err = net.DefaultResolver.LookupIP(ctx, "ip", host)
		}
		if err != nil {
			return nil, err
		}
		return ipsToStrings(ips), nil
	}
	udpLookup := func(ctx context.Context, dnsServer, host string) ([]string, error) {
		customResolver := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{Timeout: 1500 * time.Millisecond}
				return d.DialContext(ctx, network, dnsServer)
			},
		}
		ips, err := customResolver.LookupIP(ctx, "ip", host)
		if err != nil {
			return nil, err
		}
		return ipsToStrings(ips), nil
	}
	return checkDNSWithDeps(domain, sysLookup, func(ctx context.Context, host string) ([]string, error) {
		return lookupDoH(ctx, httpClient, defaultDoHEndpoints, host)
	}, udpLookup)
}

type ipLookupFunc func(ctx context.Context, host string) ([]string, error)
type udpLookupFunc func(ctx context.Context, server, host string) ([]string, error)

func checkDNSWithDeps(domain string, sysLookup, dohLookup ipLookupFunc, udpLookup udpLookupFunc) types.DNSResult {
	clean := NormalizeDomain(domain)
	if clean == "" || clean == "localhost" || clean == "local" || clean == "127.0.0.1" {
		return types.DNSResult{
			Resolved: false,
			IP:       "",
			Message:  "Локальный или пустой адрес",
		}
	}

	// Run system DNS and encrypted DoH verification in parallel so verification adds minimal latency.
	type lookupRes struct {
		ips []string
		err error
	}
	sysCh := make(chan lookupRes, 1)
	dohCh := make(chan lookupRes, 1)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
		defer cancel()
		ips, err := sysLookup(ctx, clean)
		sysCh <- lookupRes{ips: ips, err: err}
	}()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
		defer cancel()
		ips, err := dohLookup(ctx, clean)
		dohCh <- lookupRes{ips: ips, err: err}
	}()

	sysOut := <-sysCh
	dohOut := <-dohCh

	hasSys := sysOut.err == nil && len(sysOut.ips) > 0
	hasDoH := dohOut.err == nil && len(dohOut.ips) > 0

	if hasSys && hasDoH {
		if matched, ok := firstIntersection(sysOut.ips, dohOut.ips); ok {
			return types.DNSResult{
				Resolved: true,
				IP:       matched,
				DoHIP:    matched,
				Spoofed:  false,
				Message:  fmt.Sprintf("Домен %s указывает на IP %s (подтверждено DoH)", clean, matched),
			}
		}

		sysIP := sysOut.ips[0]
		dohIP := dohOut.ips[0]
		if isLikelySpoofed(sysIP, dohIP) {
			return types.DNSResult{
				Resolved: true,
				IP:       sysIP,
				DoHIP:    dohIP,
				Spoofed:  true,
				Message:  fmt.Sprintf("Внимание: системный DNS вернул %s, но шифрованный DoH указывает на %s (возможна подмена DNS или заглушка провайдера)", sysIP, dohIP),
			}
		}

		// CDN / anycast rotation in the same /16 or multiple public pools: warn mildly or report both.
		return types.DNSResult{
			Resolved: true,
			IP:       sysIP,
			DoHIP:    dohIP,
			Spoofed:  true,
			Message:  fmt.Sprintf("Расхождение DNS: системный резолвер %s, DoH %s (проверьте кэш или фильтрацию DNS)", sysIP, dohIP),
		}
	}

	if hasSys {
		ipStr := sysOut.ips[0]
		return types.DNSResult{
			Resolved: true,
			IP:       ipStr,
			Message:  fmt.Sprintf("Домен %s успешно указывает на IP: %s", clean, ipStr),
		}
	}

	if hasDoH {
		dohIP := dohOut.ips[0]
		return types.DNSResult{
			Resolved: true,
			IP:       dohIP,
			DoHIP:    dohIP,
			Message:  fmt.Sprintf("Домен %s указывает на IP: %s (через шифрованный DoH; системный DNS не ответил)", clean, dohIP),
		}
	}

	// Fallback attempt: Direct lookup via public DNS (Cloudflare 1.1.1.1, Google 8.8.8.8)
	// Bypasses broken local ISP/router DNS resolvers when HTTPS DoH is blocked or unavailable.
	fallbackServers := []string{"1.1.1.1:53", "8.8.8.8:53"}
	for _, dnsServer := range fallbackServers {
		ctxFb, cancelFb := context.WithTimeout(context.Background(), 2000*time.Millisecond)
		fbIPs, fbErr := udpLookup(ctxFb, dnsServer, clean)
		cancelFb()

		if fbErr == nil && len(fbIPs) > 0 {
			ipStr := fbIPs[0]
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

func lookupDoH(ctx context.Context, client *http.Client, endpoints []string, domain string) ([]string, error) {
	var lastErr error
	escaped := url.QueryEscape(domain)

	for _, base := range endpoints {
		reqURL := fmt.Sprintf("%s?name=%s&type=A", strings.TrimRight(base, "?"), escaped)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Accept", "application/dns-json")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("DoH %s status %d", base, resp.StatusCode)
			continue
		}

		var parsed dohJSONResponse
		if err := json.Unmarshal(body, &parsed); err != nil {
			lastErr = err
			continue
		}
		if parsed.Status != 0 {
			lastErr = fmt.Errorf("DoH RCODE %d", parsed.Status)
			continue
		}

		var out []string
		for _, ans := range parsed.Answer {
			if ans.Type == 1 || ans.Type == 28 {
				if ip := net.ParseIP(strings.TrimSpace(ans.Data)); ip != nil {
					out = append(out, ip.String())
				}
			}
		}
		if len(out) > 0 {
			return out, nil
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no DNS A records returned by DoH")
	}
	return nil, lastErr
}

func ipsToStrings(ips []net.IP) []string {
	out := make([]string, 0, len(ips))
	for _, ip := range ips {
		if ip == nil {
			continue
		}
		if v4 := ip.To4(); v4 != nil {
			out = append(out, v4.String())
		} else {
			out = append(out, ip.String())
		}
	}
	return out
}

func firstIntersection(a, b []string) (string, bool) {
	set := make(map[string]struct{}, len(b))
	for _, item := range b {
		set[item] = struct{}{}
	}
	for _, item := range a {
		if _, ok := set[item]; ok {
			return item, true
		}
	}
	return "", false
}

func isLikelySpoofed(sysIPStr, dohIPStr string) bool {
	sysIP := net.ParseIP(sysIPStr)
	dohIP := net.ParseIP(dohIPStr)
	if sysIP == nil || dohIP == nil {
		return sysIPStr != dohIPStr
	}
	if (sysIP.IsLoopback() || sysIP.IsPrivate() || sysIP.IsUnspecified() || sysIP.IsLinkLocalUnicast()) &&
		!dohIP.IsLoopback() && !dohIP.IsPrivate() {
		return true
	}
	return sysIPStr != dohIPStr
}
