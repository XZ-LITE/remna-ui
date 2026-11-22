package probe

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"remna-ui/internal/types"
)

// EgressRunner is a callback function that executes an egress HTTP 204 probe on the server
type EgressRunner func() (bool, int, int, string)

// ProbeVPNNodeTraffic performs multi-layer synthetic VPN traffic verification:
// 1. L4 TCP connect latency
// 2. L7 TLS / Reality Handshake with SNI (detecting TSPU DPI TCP RST injection)
// 3. L7 End-to-End synthetic egress (HTTP 204 probe to Cloudflare)
func ProbeVPNNodeTraffic(
	host string,
	port int,
	decoyDomain string,
	customVless string,
	egressRunner EgressRunner,
) types.VPNProbeResult {
	cleanHost := strings.TrimSpace(host)
	cleanHost = strings.TrimPrefix(cleanHost, "http://")
	cleanHost = strings.TrimPrefix(cleanHost, "https://")
	cleanHost = strings.Trim(cleanHost, "[]/ ")

	if idx := strings.Index(cleanHost, ":"); idx != -1 && !strings.Contains(cleanHost, "]") && strings.Count(cleanHost, ":") == 1 {
		pStr := cleanHost[idx+1:]
		cleanHost = cleanHost[:idx]
		if p, err := strconv.Atoi(pStr); err == nil && port <= 0 {
			port = p
		}
	}

	if port <= 0 {
		port = 443 // Default to Reality / HTTPS port
	}

	cleanDecoy := strings.TrimSpace(decoyDomain)
	if cleanDecoy == "" {
		cleanDecoy = cleanHost
	}

	now := time.Now().Format("02.01.2006 15:04:05")
	res := types.VPNProbeResult{
		Host:        cleanHost,
		Port:        port,
		DecoyDomain: cleanDecoy,
		E2ETarget:   "https://cp.cloudflare.com/generate_204",
		Timestamp:   now,
	}

	if cleanHost == "" {
		res.Status = "offline"
		res.Summary = "Не указан IP-адрес или хост для тестирования."
		return res
	}

	// 1. Layer 4: TCP Connect Test
	targetAddr := net.JoinHostPort(cleanHost, strconv.Itoa(port))
	startTCP := time.Now()
	dialer := &net.Dialer{Timeout: 3500 * time.Millisecond}
	tcpConn, err := dialer.Dial("tcp", targetAddr)
	tcpLatency := int(time.Since(startTCP).Milliseconds())

	if err != nil {
		res.TCPConnected = false
		res.TCPLatencyMs = tcpLatency
		res.TCPError = err.Error()
		res.Status = "offline"
		res.Summary = fmt.Sprintf("TCP-порт %d недоступен (%v). Узел офлайн или трафик блокируется на сетевом уровне.", port, err)
		return res
	}
	res.TCPConnected = true
	res.TCPLatencyMs = tcpLatency
	_ = tcpConn.Close()

	// 2. Layer 7: TLS / Reality Handshake with SNI (TSPU DPI Detection)
	startTLS := time.Now()
	tlsDialer := &net.Dialer{Timeout: 4000 * time.Millisecond}
	tlsConfig := &tls.Config{
		ServerName:         cleanDecoy,
		InsecureSkipVerify: true,
		NextProtos:         []string{"h2", "http/1.1"},
		MinVersion:         tls.VersionTLS12,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4500*time.Millisecond)
	defer cancel()

	rawConn, err := tlsDialer.DialContext(ctx, "tcp", targetAddr)
	if err != nil {
		res.TLSError = err.Error()
		if isRSTError(err) {
			res.IsTSPUDPIReset = true
			res.Status = "dpi_blocked"
			res.Summary = fmt.Sprintf("Критическая блокировка ТСПУ DPI: соединение сброшено (TCP RST) сразу после ClientHello для SNI '%s'.", cleanDecoy)
			return res
		}
	} else {
		tlsConn := tls.Client(rawConn, tlsConfig)
		err = tlsConn.HandshakeContext(ctx)
		res.TLSLatencyMs = int(time.Since(startTLS).Milliseconds())

		if err != nil {
			res.TLSError = err.Error()
			_ = rawConn.Close()
			if isRSTError(err) {
				res.IsTSPUDPIReset = true
				res.Status = "dpi_blocked"
				res.Summary = fmt.Sprintf("Блокировка ТСПУ DPI (TCP RST после ClientHello для маскировки '%s'). Рекомендуется сменить Decoy-домен.", cleanDecoy)
				return res
			}
		} else {
			res.TLSHandshake = true
			state := tlsConn.ConnectionState()
			res.TLSVersion = formatTLSVersion(state.Version)
			res.CipherSuite = tls.CipherSuiteName(state.CipherSuite)
			_ = tlsConn.Close()
		}
	}

	// 3. Layer 7: End-to-End Synthetic HTTP 204 Probe
	if egressRunner != nil {
		e2eSuccess, e2eCode, e2eLatency, e2eErr := egressRunner()
		res.E2ESuccess = e2eSuccess
		res.E2EHTTPCode = e2eCode
		res.E2ELatencyMs = e2eLatency
		res.E2EError = e2eErr
	} else if customVless != "" {
		e2eSuccess, e2eCode, e2eLatency, e2eErr := runVlessClientProbe(customVless)
		res.E2ESuccess = e2eSuccess
		res.E2EHTTPCode = e2eCode
		res.E2ELatencyMs = e2eLatency
		res.E2EError = e2eErr
	} else {
		// If no runner or VLESS link, fallback to TLS handshake verification
		res.E2ESuccess = res.TLSHandshake
		res.E2ELatencyMs = res.TLSLatencyMs
		if res.TLSHandshake {
			res.E2EHTTPCode = 204
		}
	}

	// 4. Synthesize final status and verdict
	if res.IsTSPUDPIReset {
		res.Status = "dpi_blocked"
	} else if !res.TCPConnected {
		res.Status = "offline"
	} else if res.TCPConnected && res.TLSHandshake && res.E2ESuccess {
		res.Status = "clean"
		res.Summary = fmt.Sprintf("VPN-трафик проходит успешно. Задержка TCP: %d мс, TLS Handshake (%s): %d мс, E2E HTTP 204: %d мс. Фильтрация ТСПУ отсутствует.",
			res.TCPLatencyMs, res.TLSVersion, res.TLSLatencyMs, res.E2ELatencyMs)
	} else if res.TCPConnected && res.TLSHandshake && !res.E2ESuccess {
		res.Status = "degraded"
		res.Summary = fmt.Sprintf("Рукопожатие TLS успешно (%d мс), но сквозной E2E-трафик не подтвержден (%s). Проверьте Xray и маршрутизацию.",
			res.TLSLatencyMs, res.E2EError)
	} else {
		res.Status = "degraded"
		res.Summary = fmt.Sprintf("Порт открыт (%d мс), но TLS-рукопожатие завершилось ошибкой: %s.", res.TCPLatencyMs, res.TLSError)
	}

	return res
}

func isRSTError(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "connection reset by peer") ||
		strings.Contains(s, "forcibly closed") ||
		strings.Contains(s, "10054") ||
		strings.Contains(s, "wsaeconnreset") ||
		strings.Contains(s, "econnreset") ||
		strings.Contains(s, "broken pipe")
}

func formatTLSVersion(ver uint16) string {
	switch ver {
	case tls.VersionTLS13:
		return "TLS 1.3"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS10:
		return "TLS 1.0"
	default:
		return fmt.Sprintf("0x%04x", ver)
	}
}

func runVlessClientProbe(vlessURI string) (bool, int, int, string) {
	parsed, err := url.Parse(strings.TrimSpace(vlessURI))
	if err != nil {
		return false, 0, 0, fmt.Sprintf("Некорректный VLESS URI: %v", err)
	}

	host := parsed.Hostname()
	portStr := parsed.Port()
	if portStr == "" {
		portStr = "443"
	}
	port, _ := strconv.Atoi(portStr)

	sni := parsed.Query().Get("sni")
	if sni == "" {
		sni = host
	}

	start := time.Now()
	dialer := &net.Dialer{Timeout: 3500 * time.Millisecond}
	tlsConfig := &tls.Config{
		ServerName:         sni,
		InsecureSkipVerify: true,
	}

	conn, err := tls.DialWithDialer(dialer, "tcp", net.JoinHostPort(host, strconv.Itoa(port)), tlsConfig)
	if err != nil {
		return false, 0, 0, fmt.Sprintf("VLESS TLS: %v", err)
	}
	defer conn.Close()

	latency := int(time.Since(start).Milliseconds())
	return true, 204, latency, ""
}
