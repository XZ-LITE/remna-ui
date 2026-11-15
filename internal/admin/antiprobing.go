package admin

import (
	"fmt"
	"strings"

	"remna-ui/internal/types"
)

// BuildAntiProbingScript creates an idempotent shell script for Anti-Censorship/Anti-Probing
func BuildAntiProbingScript(panelIP string, clientPorts []int, hideNodePort, stealthDrop, enableBBR bool, nodePort, sshPort int) string {
	if nodePort <= 0 {
		nodePort = 2222
	}
	if sshPort <= 0 {
		sshPort = 22
	}
	cleanPanelIP := strings.TrimSpace(panelIP)

	var sb strings.Builder
	sb.WriteString("#!/bin/bash\nset -e\n\n")

	// 1. Prepare Chains
	sb.WriteString("# 1. Clean existing Remna chains\n")
	sb.WriteString("iptables -D INPUT -j REMNA_INPUT 2>/dev/null || true\n")
	sb.WriteString("iptables -F REMNA_INPUT 2>/dev/null || true\n")
	sb.WriteString("iptables -X REMNA_INPUT 2>/dev/null || true\n")
	sb.WriteString("iptables -N REMNA_INPUT\n\n")

	// 2. Allow Loopback and Established
	sb.WriteString("# 2. Allow Established & Loopback\n")
	sb.WriteString("iptables -A REMNA_INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT\n")
	sb.WriteString("iptables -A REMNA_INPUT -i lo -j ACCEPT\n\n")

	// 2.1 Eliminate local service leaks / TCP RST anomalies on sensitive internal ports (DNS 53)
	sb.WriteString("# 2.1 Prevent TCP RST leaks on internal daemon ports\n")
	sb.WriteString("iptables -A REMNA_INPUT ! -i lo -p tcp --dport 53 -j DROP\n")
	sb.WriteString("iptables -A REMNA_INPUT ! -i lo -p udp --dport 53 -j DROP\n\n")

	// 3. Allow SSH (always protect against lockout)
	sb.WriteString(fmt.Sprintf("# 3. Keep SSH Open\niptables -A REMNA_INPUT -p tcp --dport %d -j ACCEPT\n\n", sshPort))

	// 4. Allow Standard Web & Reality ports
	sb.WriteString("# 4. Standard Traffic (HTTP/HTTPS/Reality)\n")
	sb.WriteString("iptables -A REMNA_INPUT -p tcp --dport 80 -j ACCEPT\n")
	sb.WriteString("iptables -A REMNA_INPUT -p tcp --dport 443 -j ACCEPT\n")
	sb.WriteString("iptables -A REMNA_INPUT -p udp --dport 443 -j ACCEPT\n\n")

	// Extra client ports
	for _, p := range clientPorts {
		if p > 0 && p != 80 && p != 443 && p != nodePort && p != sshPort {
			sb.WriteString(fmt.Sprintf("iptables -A REMNA_INPUT -p tcp --dport %d -j ACCEPT\n", p))
			sb.WriteString(fmt.Sprintf("iptables -A REMNA_INPUT -p udp --dport %d -j ACCEPT\n", p))
		}
	}

	// 5. Hide Node Port 2222 in DOCKER-USER (and INPUT)
	if hideNodePort {
		sb.WriteString("\n# 5. Hide Remnanode Port from Public Scanners\n")
		sb.WriteString("iptables -N DOCKER-USER 2>/dev/null || true\n")
		sb.WriteString("iptables -D DOCKER-USER -j REMNA_DOCKER 2>/dev/null || true\n")
		sb.WriteString("iptables -F REMNA_DOCKER 2>/dev/null || true\n")
		sb.WriteString("iptables -X REMNA_DOCKER 2>/dev/null || true\n")
		sb.WriteString("iptables -N REMNA_DOCKER\n\n")

		// Make sure DOCKER-USER preserves established sessions and loopback
		sb.WriteString("iptables -A REMNA_DOCKER -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT\n")
		sb.WriteString(fmt.Sprintf("iptables -A REMNA_DOCKER -s 127.0.0.1 -p tcp --dport %d -j ACCEPT\n", nodePort))

		if cleanPanelIP != "" {
			// cleanPanelIP can be an IPv4 or comma/space separated list of IPs
			ips := strings.FieldsFunc(cleanPanelIP, func(r rune) bool {
				return r == ',' || r == ' ' || r == ';'
			})
			for _, ip := range ips {
				cleanIP := strings.TrimSpace(ip)
				if cleanIP != "" {
					sb.WriteString(fmt.Sprintf("iptables -A REMNA_DOCKER -p tcp --dport %d -s %s -j ACCEPT\n", nodePort, cleanIP))
					sb.WriteString(fmt.Sprintf("iptables -A REMNA_INPUT -p tcp --dport %d -s %s -j ACCEPT\n", nodePort, cleanIP))
				}
			}
			// Only apply DROP when panel IP was explicitly whitelisted to prevent locking out the panel!
			sb.WriteString(fmt.Sprintf("iptables -A REMNA_DOCKER -p tcp --dport %d -j DROP\n", nodePort))
			sb.WriteString(fmt.Sprintf("iptables -A REMNA_INPUT -p tcp --dport %d -j DROP\n", nodePort))
		} else {
			sb.WriteString(fmt.Sprintf("# Note: Panel IP not provided. Port %d left open to prevent panel disconnection.\n", nodePort))
		}
		sb.WriteString("iptables -A REMNA_DOCKER -j RETURN\n")
		sb.WriteString("iptables -I DOCKER-USER 1 -j REMNA_DOCKER\n\n")
	}

	// 6. Connect REMNA_INPUT into INPUT
	sb.WriteString("# 6. Link into INPUT chain\n")
	sb.WriteString("iptables -A INPUT -j REMNA_INPUT\n\n")

	// 7. Stealth DROP (Eliminate TCP RST fingerprint)
	if stealthDrop {
		sb.WriteString("# 7. Stealth DROP Policy (anti-RST fingerprint)\n")
		sb.WriteString("iptables -P INPUT DROP\n")
		sb.WriteString("iptables -P FORWARD ACCEPT\n\n")

		// IPv6 parity
		sb.WriteString("if command -v ip6tables >/dev/null 2>&1; then\n")
		sb.WriteString("  ip6tables -D INPUT -j REMNA_INPUT6 2>/dev/null || true\n")
		sb.WriteString("  ip6tables -F REMNA_INPUT6 2>/dev/null || true\n")
		sb.WriteString("  ip6tables -X REMNA_INPUT6 2>/dev/null || true\n")
		sb.WriteString("  ip6tables -N REMNA_INPUT6 2>/dev/null || true\n")
		sb.WriteString("  ip6tables -A REMNA_INPUT6 -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT 2>/dev/null || true\n")
		sb.WriteString("  ip6tables -A REMNA_INPUT6 -i lo -j ACCEPT 2>/dev/null || true\n")
		sb.WriteString("  ip6tables -A REMNA_INPUT6 ! -i lo -p tcp --dport 53 -j DROP 2>/dev/null || true\n")
		sb.WriteString("  ip6tables -A REMNA_INPUT6 ! -i lo -p udp --dport 53 -j DROP 2>/dev/null || true\n")
		sb.WriteString("  ip6tables -A REMNA_INPUT6 -p icmpv6 -j ACCEPT 2>/dev/null || true\n")
		sb.WriteString(fmt.Sprintf("  ip6tables -A REMNA_INPUT6 -p tcp --dport %d -j ACCEPT 2>/dev/null || true\n", sshPort))
		sb.WriteString("  ip6tables -A REMNA_INPUT6 -p tcp --dport 80 -j ACCEPT 2>/dev/null || true\n")
		sb.WriteString("  ip6tables -A REMNA_INPUT6 -p tcp --dport 443 -j ACCEPT 2>/dev/null || true\n")
		sb.WriteString(fmt.Sprintf("  ip6tables -A REMNA_INPUT6 -p tcp --dport %d -j DROP 2>/dev/null || true\n", nodePort))
		sb.WriteString("  ip6tables -A INPUT -j REMNA_INPUT6 2>/dev/null || true\n")
		sb.WriteString("  ip6tables -P INPUT DROP 2>/dev/null || true\n")
		sb.WriteString("fi\n\n")
	}

	// 8. TCP BBR & FQ Optimization
	if enableBBR {
		sb.WriteString("# 8. Enable TCP BBR and FQ network scheduler\n")
		sb.WriteString("modprobe tcp_bbr 2>/dev/null || true\n")
		sb.WriteString("mkdir -p /etc/sysctl.d\n")
		sb.WriteString("cat << 'BBRCONF' > /etc/sysctl.d/99-remna-bbr.conf\n")
		sb.WriteString("net.core.default_qdisc=fq\n")
		sb.WriteString("net.ipv4.tcp_congestion_control=bbr\n")
		sb.WriteString("BBRCONF\n")
		sb.WriteString("sysctl -p /etc/sysctl.d/99-remna-bbr.conf 2>/dev/null || sysctl -w net.ipv4.tcp_congestion_control=bbr 2>/dev/null || true\n\n")
	}

	// 9. Persist rules
	sb.WriteString("# 9. Save Rules for Reboots\n")
	sb.WriteString("mkdir -p /etc/iptables\n")
	sb.WriteString("iptables-save > /etc/iptables/rules.v4 2>/dev/null || true\n")
	sb.WriteString("command -v ip6tables-save >/dev/null 2>&1 && ip6tables-save > /etc/iptables/rules.v6 2>/dev/null || true\n")
	sb.WriteString("command -v netfilter-persistent >/dev/null 2>&1 && netfilter-persistent save 2>/dev/null || true\n")

	return sb.String()
}

// ApplyAntiProbingFirewall executes the anti-probing firewall on the remote VPS
func ApplyAntiProbingFirewall(server types.SavedServer, panelIP string, clientPorts []int, hideNodePort, stealthDrop, enableBBR bool) types.CommandResult {
	script := BuildAntiProbingScript(panelIP, clientPorts, hideNodePort, stealthDrop, enableBBR, 2222, server.Port)
	client, err := getSSHClient(server)
	if err != nil {
		return types.CommandResult{Success: false, Error: fmt.Sprintf("Ошибка SSH: %v", err)}
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return types.CommandResult{Success: false, Error: fmt.Sprintf("Ошибка сессии: %v", err)}
	}
	defer session.Close()

	remoteCmd := fmt.Sprintf("bash -s << 'EOF'\n%s\nEOF\n", script)
	out, err := session.CombinedOutput(remoteCmd)
	if err != nil {
		return types.CommandResult{
			Success: false,
			Output:  string(out),
			Error:   fmt.Sprintf("Сбой применения правил: %v", err),
		}
	}

	return types.CommandResult{
		Success: true,
		Output:  string(out),
	}
}

// AuditNodeSecurity analyzes the node firewall, services, and probing exposure
func AuditNodeSecurity(server types.SavedServer, panelIP string) types.NodeSecurityAudit {
	client, err := getSSHClient(server)
	if err != nil {
		return types.NodeSecurityAudit{
			OverallScore: 0,
			Summary:      fmt.Sprintf("Не удалось подключиться по SSH: %v", err),
		}
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return types.NodeSecurityAudit{
			OverallScore: 0,
			Summary:      fmt.Sprintf("Не удалось открыть сессию SSH: %v", err),
		}
	}
	defer session.Close()

	auditCmd := `
echo "___SECTION:DOCKER___"
docker ps --filter "name=remnanode" --format "{{.Status}}" 2>/dev/null || echo "NOT_RUNNING"

echo "___SECTION:DOCKER_USER___"
iptables -S DOCKER-USER 2>/dev/null || true
iptables -S REMNA_DOCKER 2>/dev/null || true

echo "___SECTION:INPUT___"
iptables -S INPUT 2>/dev/null || true
iptables -S REMNA_INPUT 2>/dev/null || true

echo "___SECTION:POLICY___"
iptables -L INPUT -n 2>/dev/null | head -n 1 || true

echo "___SECTION:PORTS___"
ss -tulpn 2>/dev/null || netstat -tulpn 2>/dev/null || true

echo "___SECTION:BBR___"
sysctl -n net.ipv4.tcp_congestion_control 2>/dev/null || cat /proc/sys/net/ipv4/tcp_congestion_control 2>/dev/null || echo "unknown"

echo "___SECTION:CORE___"
if [ -f /opt/remnawave/xray-custom ]; then
  echo "CUSTOM_FOUND"
elif docker exec remnanode xray version 2>/dev/null | head -n 1; then
  echo "OFFICIAL_CORE"
else
  echo "STANDARD"
fi

echo "___SECTION:END___"
`
	outBytes, err := session.CombinedOutput(auditCmd)
	if err != nil {
		return types.NodeSecurityAudit{
			OverallScore: 10,
			Summary:      fmt.Sprintf("Ошибка выполнения аудита: %v", err),
		}
	}

	out := string(outBytes)

	// Parse sections safely with delimiters
	sections := make(map[string]string)
	lines := strings.Split(out, "\n")
	currentSection := ""
	var currentBuilder strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "___SECTION:") && strings.HasSuffix(trimmed, "___") {
			if currentSection != "" {
				sections[currentSection] = currentBuilder.String()
			}
			currentSection = strings.TrimSuffix(strings.TrimPrefix(trimmed, "___SECTION:"), "___")
			currentBuilder.Reset()
		} else if currentSection != "" {
			currentBuilder.WriteString(line)
			currentBuilder.WriteString("\n")
		}
	}
	if currentSection != "" {
		sections[currentSection] = currentBuilder.String()
	}

	checks := []string{}
	score := 0

	// Check 1: Remnanode container & Reality port 443
	dockerSection := sections["DOCKER"]
	portsSection := sections["PORTS"]
	containerRunning := strings.Contains(dockerSection, "Up ")
	port443Listening := strings.Contains(portsSection, ":443 ") || strings.Contains(portsSection, ":443\t") || strings.Contains(portsSection, ":::443")

	if containerRunning && port443Listening {
		score += 25
		checks = append(checks, "Служба Xray Reality активна: контейнер remnanode запущен и слушает порт 443.")
	} else if containerRunning {
		score += 15
		checks = append(checks, "Контейнер remnanode запущен, но порт 443 не обнаружен в слушающих сокетах хоста.")
	} else {
		checks = append(checks, "Контейнер remnanode НЕ запущен! Нода не обрабатывает соединения Reality.")
	}

	// Check 2: Port 2222 in DOCKER-USER & Panel Whitelist
	dockerUserRules := sections["DOCKER_USER"]
	cleanPanelIP := strings.TrimSpace(panelIP)
	nodePortProtected := strings.Contains(dockerUserRules, "2222 -j DROP")
	dockerUserActive := strings.Contains(dockerUserRules, "-A DOCKER-USER") || strings.Contains(dockerUserRules, "REMNA_DOCKER")
	panelWhitelisted := false

	if cleanPanelIP != "" {
		panelWhitelisted = strings.Contains(dockerUserRules, cleanPanelIP) && strings.Contains(dockerUserRules, "ACCEPT")
	} else {
		// If panel IP wasn't specified, check if any whitelist rule for port 2222 exists
		panelWhitelisted = strings.Contains(dockerUserRules, "--dport 2222 -j ACCEPT") || strings.Contains(dockerUserRules, "-p tcp -m tcp --dport 2222 -j ACCEPT")
	}

	if nodePortProtected && panelWhitelisted {
		score += 30
		if cleanPanelIP != "" {
			checks = append(checks, fmt.Sprintf("Порт управления (2222) изолирован: доступ разрешен только для мастер-панели (%s), для остальных сканеров активен DROP.", cleanPanelIP))
		} else {
			checks = append(checks, "Порт управления (2222) изолирован в DOCKER-USER с правилом белого списка.")
		}
	} else if nodePortProtected && !panelWhitelisted && cleanPanelIP != "" {
		score += 10
		checks = append(checks, fmt.Sprintf("Внимание: порт 2222 заблокирован в DOCKER-USER, но IP мастер-панели (%s) не найден в белом списке!", cleanPanelIP))
	} else if nodePortProtected {
		score += 20
		checks = append(checks, "Порт 2222 изолирован в DOCKER-USER (молчаливый сброс DROP).")
	} else {
		checks = append(checks, "Уязвимость: порт 2222 выставлен в открытый интернет через Docker NAT. Сканеры ТСПУ легко вычисляют отпечаток ноды.")
	}

	// Check 3: Stealth DROP Policy (anti TCP RST fingerprint)
	policySection := sections["POLICY"]
	inputSection := sections["INPUT"]
	stealthDropActive := strings.Contains(policySection, "DROP") || strings.Contains(inputSection, "-P INPUT DROP")
	if stealthDropActive {
		score += 25
		checks = append(checks, "Скрытный режим портов (Stealth DROP) активен: закрытые порты не отвечают пакетами TCP RST, имитируя выключенный сервер.")
	} else {
		checks = append(checks, "Политика INPUT в режиме ACCEPT: закрытые порты отвечают пакетами TCP RST, выдавая сервер активным сканерам ТСПУ.")
	}

	// Check 4: TCP BBR Congestion Control
	bbrSection := sections["BBR"]
	bbrActive := strings.Contains(strings.ToLower(bbrSection), "bbr")
	if bbrActive {
		score += 20
		checks = append(checks, "Оптимизация TCP BBR + FQ активна: устойчивость к DPI-дропам и поддержание высокой скорости соединения.")
	} else {
		cc := strings.TrimSpace(bbrSection)
		if cc == "" || cc == "unknown" {
			cc = "cubic/reno"
		}
		checks = append(checks, fmt.Sprintf("TCP BBR не включен (текущий: %s). Рекомендуется включить BBR для защиты от троттлинга ТСПУ.", cc))
	}

	// Check 5: Xray Core Variant
	coreSection := sections["CORE"]
	xrayCoreType := "Официальное ядро Xray"
	if strings.Contains(coreSection, "CUSTOM_FOUND") {
		xrayCoreType = "Кастомное ядро (ML-KEM / Anti-Fingerprint)"
		checks = append(checks, "Кастомное ядро Xray активно: усиленная маскировка TLS-отпечатков.")
	} else {
		checks = append(checks, "Используется официальное ядро Xray (стандартный протокол Reality).")
	}

	summary := ""
	if score >= 90 {
		summary = "Максимальный уровень защиты (A+). Нода экранирована от сканеров ТСПУ, порт 2222 защищен, включен TCP BBR."
	} else if score >= 70 {
		summary = "Хороший уровень защиты (A). Основные параметры активны, рекомендуются дополнительные настройки."
	} else if score >= 45 {
		summary = "Частичная защита (B). Порт управления или скрытный режим не настроены."
	} else {
		summary = "Низкий уровень защиты (D). Высокий риск обнаружения активными зондами РКН/ТСПУ."
	}

	return types.NodeSecurityAudit{
		NodePortProtected: nodePortProtected,
		PanelWhitelisted:  panelWhitelisted,
		DockerUserActive:  dockerUserActive,
		StealthDropActive: stealthDropActive,
		DecoyActive:       port443Listening,
		BBRActive:         bbrActive,
		ContainerRunning:  containerRunning,
		Port443Listening:  port443Listening,
		XrayCoreType:      xrayCoreType,
		OverallScore:      score,
		Summary:           summary,
		Checks:            checks,
	}
}

// DeploySelfStealDecoy installs an isolated Nginx decoy site on port 8444
func DeploySelfStealDecoy(server types.SavedServer, domain string) types.CommandResult {
	cleanDomain := strings.TrimSpace(domain)
	if cleanDomain == "" {
		cleanDomain = server.Label
		if cleanDomain == "" {
			cleanDomain = server.Host
		}
	}

	client, err := getSSHClient(server)
	if err != nil {
		return types.CommandResult{Success: false, Error: fmt.Sprintf("SSH ошибка: %v", err)}
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return types.CommandResult{Success: false, Error: fmt.Sprintf("Сессия SSH: %v", err)}
	}
	defer session.Close()

	script := fmt.Sprintf(`
DOMAIN="%s"
FALLBACK_PORT=8444
DECOY_ROOT="/var/www/decoy"

mkdir -p "$DECOY_ROOT" /etc/nginx/ssl /etc/nginx/sites-available /etc/nginx/sites-enabled

# Install Nginx, Certbot and tools if missing
if ! command -v nginx >/dev/null 2>&1 || ! command -v certbot >/dev/null 2>&1 || ! command -v openssl >/dev/null 2>&1; then
  apt-get update -y && apt-get install -y nginx certbot openssl curl || true
fi

# Realistic corporate landing page
cat << 'HTML' > "$DECOY_ROOT/index.html"
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Enterprise Cloud & Infrastructure Services</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #0b0f19; color: #f3f4f6; margin: 0; padding: 0; }
        header { padding: 40px 20px; text-align: center; border-bottom: 1px solid #1f2937; }
        h1 { font-size: 28px; margin: 0 0 10px 0; color: #60cdff; }
        p { color: #9ca3af; font-size: 15px; }
        .grid { display: flex; max-width: 800px; margin: 40px auto; gap: 20px; padding: 0 20px; }
        .card { background: #111827; border: 1px solid #1f2937; border-radius: 8px; padding: 20px; flex: 1; }
        .card h3 { margin-top: 0; color: #e5e7eb; }
        footer { text-align: center; font-size: 12px; color: #6b7280; padding: 30px; }
    </style>
</head>
<body>
    <header>
        <h1>Global Network Solutions</h1>
        <p>Enterprise Edge Relaying, Content Distribution and Cloud Security</p>
    </header>
    <div class="grid">
        <div class="card">
            <h3>Edge Nodes</h3>
            <p>Low-latency Anycast routing nodes providing maximum reliability and bandwidth.</p>
        </div>
        <div class="card">
            <h3>Encrypted Transit</h3>
            <p>State-of-the-art TLS 1.3 transport layer encryption with automated failover.</p>
        </div>
    </div>
    <footer>&copy; 2026 Cloud Infrastructure Systems. All rights reserved.</footer>
</body>
</html>
HTML

USE_LE=0
if [ -n "$DOMAIN" ] && [[ "$DOMAIN" == *"."* ]] && [[ ! "$DOMAIN" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  CERT_PATH="/etc/letsencrypt/live/$DOMAIN/fullchain.pem"
  KEY_PATH="/etc/letsencrypt/live/$DOMAIN/privkey.pem"

  if [ -f "$CERT_PATH" ] && [ -f "$KEY_PATH" ]; then
    USE_LE=1
  else
    cat << 'BOOTCONF' > /etc/nginx/sites-available/decoy_boot
server {
    listen 80;
    server_name _;
    root /var/www/decoy;
    location /.well-known/acme-challenge/ {
        root /var/www/decoy;
    }
    location / {
        return 200 "OK\n";
    }
}
BOOTCONF
    ln -sf /etc/nginx/sites-available/decoy_boot /etc/nginx/sites-enabled/decoy_boot
    nginx -t >/dev/null 2>&1 && (systemctl reload nginx || systemctl restart nginx) || true

    certbot certonly --webroot -w "$DECOY_ROOT" -d "$DOMAIN" --non-interactive --agree-tos --register-unsafely-without-email --keep-until-expiring >/dev/null 2>&1 || true

    if [ ! -f "$CERT_PATH" ]; then
      systemctl stop nginx 2>/dev/null || true
      certbot certonly --standalone -d "$DOMAIN" --non-interactive --agree-tos --register-unsafely-without-email --keep-until-expiring >/dev/null 2>&1 || true
      systemctl start nginx 2>/dev/null || true
    fi

    rm -f /etc/nginx/sites-enabled/decoy_boot /etc/nginx/sites-available/decoy_boot

    if [ -f "$CERT_PATH" ] && [ -f "$KEY_PATH" ]; then
      USE_LE=1
    fi
  fi
fi

if [ "$USE_LE" -eq 1 ]; then
  SSL_CERT="/etc/letsencrypt/live/$DOMAIN/fullchain.pem"
  SSL_KEY="/etc/letsencrypt/live/$DOMAIN/privkey.pem"
  SERVER_NAME="$DOMAIN"
else
  if [ ! -f /etc/nginx/ssl/decoy.crt ]; then
    CN="${DOMAIN:-%s}"
    openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
      -keyout /etc/nginx/ssl/decoy.key \
      -out /etc/nginx/ssl/decoy.crt \
      -subj "/CN=$CN" >/dev/null 2>&1
  fi
  SSL_CERT="/etc/nginx/ssl/decoy.crt"
  SSL_KEY="/etc/nginx/ssl/decoy.key"
  SERVER_NAME="${DOMAIN:-_}"
fi

# Configure Nginx on local port 8444 and port 80 (HTTP redirect + ACME challenge)
cat << CONF > /etc/nginx/sites-available/decoy.conf
server {
    listen 80 default_server;
    listen [::]:80 default_server;
    server_name _;

    location /.well-known/acme-challenge/ {
        root $DECOY_ROOT;
    }

    location / {
        return 301 https://\$host\$request_uri;
    }
}

server {
    listen 127.0.0.1:$FALLBACK_PORT ssl http2;
    server_name $SERVER_NAME;

    ssl_certificate $SSL_CERT;
    ssl_certificate_key $SSL_KEY;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    root $DECOY_ROOT;
    index index.html;

    location / {
        try_files \$uri \$uri/ =404;
    }
}
CONF

rm -f /etc/nginx/sites-enabled/default
ln -sf /etc/nginx/sites-available/decoy.conf /etc/nginx/sites-enabled/decoy.conf
nginx -t && (systemctl reload nginx || systemctl restart nginx)
systemctl enable nginx >/dev/null 2>&1 || true
`, cleanDomain, server.Host)

	out, err := session.CombinedOutput(script)
	if err != nil {
		return types.CommandResult{
			Success: false,
			Output:  string(out),
			Error:   fmt.Sprintf("Ошибка развертывания Decoy сайта: %v", err),
		}
	}

	return types.CommandResult{
		Success: true,
		Output:  string(out),
	}
}
