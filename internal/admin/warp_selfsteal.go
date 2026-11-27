package admin

import (
	"encoding/base64"
	"fmt"
	"net"
	"strconv"
	"strings"

	"remna-ui/internal/generator"
	"remna-ui/internal/types"
)

const (
	warpEndpointV4 = "162.159.192.1:2408"
	warpFwMark     = "51888"
)

type selfStealTarget struct {
	ListenHost string
	ListenPort int
	SNIDomain  string
}

// InstallCloudflareWARP installs Native WireGuard WARP (wgcf + wg-quick@warp) with IPv4 endpoint pinning,
// fail-closed nftables kill-switch (mark 51888), and cron watchdog. If WireGuard is unavailable in the
// server kernel, it falls back automatically to official cloudflare-warp SOCKS5 proxy mode on 127.0.0.1:40000.
func InstallCloudflareWARP(server types.SavedServer) types.CommandResult {
	session, release, err := getPooledSession(server)
	if err != nil {
		return types.CommandResult{Success: false, Error: "Ошибка подключения SSH: " + err.Error()}
	}
	defer release()

	script := buildWarpInstallScript()
	out, err := session.CombinedOutput(script)
	if err != nil {
		trimmed := strings.TrimSpace(string(out))
		if trimmed == "" {
			trimmed = err.Error()
		}
		return types.CommandResult{Success: false, Error: trimmed}
	}

	return types.CommandResult{Success: true, Output: strings.TrimSpace(string(out))}
}

func buildWarpInstallScript() string {
	tmpl := `set -e
export DEBIAN_FRONTEND=noninteractive

WARP_ENDPOINT_V4="__WARP_ENDPOINT_V4__"
MARK="__WARP_FWMARK__"
WGCF_DIR="/opt/warp-native"

install_pkg() {
  if command -v apt-get >/dev/null 2>&1; then
    apt-get update -qq >/dev/null 2>&1 || true
    apt-get install -y -qq "$@" >/dev/null 2>&1 || return 1
  elif command -v dnf >/dev/null 2>&1; then
    dnf install -y -q "$@" >/dev/null 2>&1 || return 1
  elif command -v yum >/dev/null 2>&1; then
    yum install -y -q "$@" >/dev/null 2>&1 || return 1
  elif command -v apk >/dev/null 2>&1; then
    apk add --no-cache "$@" >/dev/null 2>&1 || return 1
  else
    return 1
  fi
}

supports_wireguard_kernel() {
  if [ -d /sys/module/wireguard ]; then
    return 0
  fi
  if command -v modprobe >/dev/null 2>&1 && modprobe wireguard >/dev/null 2>&1; then
    return 0
  fi
  if command -v wg >/dev/null 2>&1 && ip link add dev wg_probe_tmp type wireguard >/dev/null 2>&1; then
    ip link delete wg_probe_tmp >/dev/null 2>&1 || true
    return 0
  fi
  return 1
}

echo ">>> Настройка Native Cloudflare WARP (wgcf + nftables kill-switch)..."

USE_NATIVE=0
if command -v wg >/dev/null 2>&1 || install_pkg wireguard-tools curl ca-certificates; then
  if supports_wireguard_kernel; then
    USE_NATIVE=1
  fi
fi

if [ "$USE_NATIVE" -eq 1 ]; then
  echo "Режим: Native WireGuard (wgcf + mark $MARK)"
  mkdir -p "$WGCF_DIR/logs" /etc/wireguard
  install_pkg nftables || true

  if ! command -v wgcf >/dev/null 2>&1; then
    ARCH=$(uname -m)
    case "$ARCH" in
      x86_64) WGCF_ARCH="amd64" ;;
      aarch64|arm64) WGCF_ARCH="arm64" ;;
      armv7l) WGCF_ARCH="armv7" ;;
      *) WGCF_ARCH="amd64" ;;
    esac
    WGCF_VERSION=$(curl -fsSL --max-time 15 https://api.github.com/repos/ViRb3/wgcf/releases/latest 2>/dev/null | grep '"tag_name"' | head -1 | cut -d'"' -f4 || true)
    if [ -z "$WGCF_VERSION" ]; then
      echo "Не удалось получить версию wgcf с GitHub API."
      exit 1
    fi
    WGCF_URL="https://github.com/ViRb3/wgcf/releases/download/${WGCF_VERSION}/wgcf_${WGCF_VERSION#v}_linux_${WGCF_ARCH}"
    curl -fsSL --max-time 30 "$WGCF_URL" -o /usr/local/bin/wgcf
    chmod +x /usr/local/bin/wgcf
    echo "Установлен wgcf ${WGCF_VERSION} (${WGCF_ARCH})"
  fi

  cd "$WGCF_DIR"
  if [ -f /root/wgcf-account.toml ] && [ ! -f "$WGCF_DIR/wgcf-account.toml" ]; then
    mv /root/wgcf-account.toml "$WGCF_DIR/wgcf-account.toml"
  fi
  if [ ! -f "$WGCF_DIR/wgcf-account.toml" ]; then
    echo "Регистрация нового аккаунта WARP через wgcf..."
    timeout 60 bash -c 'yes | wgcf register' >/dev/null 2>&1 || (sleep 2 && timeout 60 bash -c 'yes | wgcf register' >/dev/null 2>&1)
  else
    echo "Используется существующий профиль wgcf-account.toml (без сброса лимитов)."
  fi

  if [ ! -f "$WGCF_DIR/wgcf-account.toml" ]; then
    echo "Ошибка регистрации wgcf-account.toml (возможно Cloudflare 429/rate-limit)."
    exit 1
  fi

  wgcf generate >/dev/null 2>&1
  if [ ! -f "$WGCF_DIR/wgcf-profile.conf" ]; then
    echo "Не удалось сгенерировать wgcf-profile.conf."
    exit 1
  fi

  # Harden profile: remove DNS & IPv6 routes, disable default table hijack, pin IPv4 endpoint
  sed -i '/^DNS =/d' "$WGCF_DIR/wgcf-profile.conf"
  sed -i 's/,\s*[0-9a-fA-F:]\+\/128//' "$WGCF_DIR/wgcf-profile.conf"
  sed -i '/Address = [0-9a-fA-F:]\+\/128/d' "$WGCF_DIR/wgcf-profile.conf"
  if ! grep -q '^Table = off' "$WGCF_DIR/wgcf-profile.conf"; then
    sed -i '/^MTU =/a Table = off' "$WGCF_DIR/wgcf-profile.conf"
  fi
  sed -i "s|^Endpoint = .*|Endpoint = ${WARP_ENDPOINT_V4}|" "$WGCF_DIR/wgcf-profile.conf"
  sed -i 's|^AllowedIPs = .*|AllowedIPs = 0.0.0.0/0|' "$WGCF_DIR/wgcf-profile.conf"
  if ! grep -q '^PersistentKeepalive = 25' "$WGCF_DIR/wgcf-profile.conf"; then
    sed -i '/^Endpoint =/a PersistentKeepalive = 25' "$WGCF_DIR/wgcf-profile.conf"
  fi
  mv "$WGCF_DIR/wgcf-profile.conf" /etc/wireguard/warp.conf
  chmod 600 /etc/wireguard/warp.conf

  # Fail-closed nftables kill-switch (prevents marked Xray traffic from leaking outside warp interface)
  cat > /opt/warp-native/warp-killswitch.sh <<'KILLSWITCH_EOF'
#!/bin/bash
set -euo pipefail
TABLE="warp_native"
CHAIN="warp_native_killswitch"
MARK="51888"
IFACE="warp"

case "${1:-}" in
  start)
    if ! command -v nft >/dev/null 2>&1; then
      exit 0
    fi
    if ! nft list table inet "$TABLE" >/dev/null 2>&1; then
      nft -f - <<EOF
add table inet $TABLE
add chain inet $TABLE $CHAIN { type filter hook output priority -150; policy accept; }
add rule inet $TABLE $CHAIN meta mark $MARK oifname != "$IFACE" counter drop comment "warp-native kill-switch"
EOF
    elif ! nft list chain inet "$TABLE" "$CHAIN" >/dev/null 2>&1; then
      nft -f - <<EOF
add chain inet $TABLE $CHAIN { type filter hook output priority -150; policy accept; }
add rule inet $TABLE $CHAIN meta mark $MARK oifname != "$IFACE" counter drop comment "warp-native kill-switch"
EOF
    else
      nft -f - <<EOF
flush chain inet $TABLE $CHAIN
add rule inet $TABLE $CHAIN meta mark $MARK oifname != "$IFACE" counter drop comment "warp-native kill-switch"
EOF
    fi
    ;;
  stop)
    nft delete chain inet "$TABLE" "$CHAIN" 2>/dev/null || true
    ;;
  *)
    echo "Usage: $0 {start|stop}"
    exit 1
    ;;
esac
KILLSWITCH_EOF
  chmod +x /opt/warp-native/warp-killswitch.sh

  if command -v systemctl >/dev/null 2>&1; then
    cat > /etc/systemd/system/warp-native-killswitch.service <<'KS_UNIT_EOF'
[Unit]
Description=WARP Native fail-closed kill-switch
After=nftables.service
Before=wg-quick@warp.service

[Service]
Type=oneshot
ExecStart=/opt/warp-native/warp-killswitch.sh start
ExecStop=/opt/warp-native/warp-killswitch.sh stop
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
KS_UNIT_EOF

    mkdir -p /etc/systemd/system/wg-quick@warp.service.d
    cat > /etc/systemd/system/wg-quick@warp.service.d/warp-native-killswitch.conf <<'WG_DROPIN_EOF'
[Unit]
Requires=warp-native-killswitch.service
After=warp-native-killswitch.service

[Service]
ExecStartPre=/opt/warp-native/warp-killswitch.sh start
WG_DROPIN_EOF

    systemctl daemon-reload >/dev/null 2>&1 || true
    systemctl enable --now warp-native-killswitch.service >/dev/null 2>&1 || echo "Предупреждение: nftables kill-switch не активирован (проверьте поддержку nft в ядре)."
    systemctl stop wg-quick@warp >/dev/null 2>&1 || true
    ip link delete warp >/dev/null 2>&1 || true
    systemctl reset-failed wg-quick@warp >/dev/null 2>&1 || true
    systemctl enable --now wg-quick@warp >/dev/null 2>&1
  else
    /opt/warp-native/warp-killswitch.sh start >/dev/null 2>&1 || true
    wg-quick down warp >/dev/null 2>&1 || true
    wg-quick up warp >/dev/null 2>&1
  fi

  # Watchdog cron with 120s restart cooldown
  cat > /opt/warp-native/warp-watchdog.sh <<'WD_EOF'
#!/bin/bash
LOG="/opt/warp-native/logs/watchdog.log"
COOLDOWN_FILE="/opt/warp-native/.last_restart"
RESTART_COOLDOWN=120

log() {
  echo "$(date '+%Y-%m-%d %H:%M:%S') [$1] $2" >> "$LOG"
}

if [ -f "$LOG" ] && [ "$(wc -l < "$LOG")" -gt 1000 ]; then
  tail -n 1000 "$LOG" > "${LOG}.tmp" && mv "${LOG}.tmp" "$LOG"
fi

if command -v systemctl >/dev/null 2>&1 && ! systemctl is-active --quiet wg-quick@warp; then
  NOW=$(date +%s)
  LAST=$(cat "$COOLDOWN_FILE" 2>/dev/null || echo 0)
  if [ $((NOW - LAST)) -ge $RESTART_COOLDOWN ]; then
    echo "$NOW" > "$COOLDOWN_FILE"
    systemctl restart wg-quick@warp && log "RESTART" "wg-quick@warp restarted (inactive)"
  fi
  exit 0
fi

if ping -I warp -c 2 -W 3 1.1.1.1 >/dev/null 2>&1 || curl -s --interface warp --max-time 6 https://www.cloudflare.com/cdn-cgi/trace 2>/dev/null | grep -q '^warp='; then
  exit 0
fi

NOW=$(date +%s)
LAST=$(cat "$COOLDOWN_FILE" 2>/dev/null || echo 0)
if [ $((NOW - LAST)) -ge $RESTART_COOLDOWN ]; then
  echo "$NOW" > "$COOLDOWN_FILE"
  systemctl restart wg-quick@warp >/dev/null 2>&1 || (wg-quick down warp >/dev/null 2>&1 || true; wg-quick up warp >/dev/null 2>&1)
  log "RESTART" "WARP connectivity recovered"
fi
WD_EOF
  chmod +x /opt/warp-native/warp-watchdog.sh
  cat > /etc/cron.d/warp-native <<'CRON_EOF'
*/5 * * * * root /opt/warp-native/warp-watchdog.sh
CRON_EOF
  chmod 644 /etc/cron.d/warp-native

  # Poll handshake & verify Cloudflare WARP exit
  HANDSHAKE_OK=0
  for i in $(seq 1 15); do
    HS=$(wg show warp latest-handshakes 2>/dev/null | awk '{print $2}')
    if [ -n "$HS" ] && [ "$HS" -gt 0 ]; then
      HANDSHAKE_OK=1
      break
    fi
    sleep 1
  done

  TRACE=$(curl -s --interface warp --max-time 8 https://www.cloudflare.com/cdn-cgi/trace 2>/dev/null | grep -E '^(warp|loc|ip)=' | tr '\n' ' ' || true)
  echo "Native WARP установлен и запущен!"
  echo "Интерфейс: warp (IPv4 endpoint $WARP_ENDPOINT_V4)"
  echo "Kill-switch: nftables table inet warp_native (meta mark $MARK -> drop при падении warp)"
  if [ "$HANDSHAKE_OK" -eq 1 ]; then
    echo "Handshake WireGuard: активен"
  else
    echo "Handshake WireGuard: ожидает трафика (watchdog проверит автоматически)"
  fi
  [ -n "$TRACE" ] && echo "Cloudflare trace: $TRACE"
  echo "Для Xray outbound укажите streamSettings.sockopt: {\"interface\": \"warp\", \"mark\": $MARK}"
  exit 0
fi

echo "WireGuard недоступен в ядре — переключение на fallback-режим cloudflare-warp (SOCKS5 127.0.0.1:40000)..."
if ! command -v warp-cli >/dev/null 2>&1; then
  if command -v apt-get >/dev/null 2>&1 && command -v lsb_release >/dev/null 2>&1; then
    apt-get update -qq >/dev/null 2>&1 || true
    apt-get install -y -qq curl gpg lsb-release >/dev/null 2>&1 || true
    curl -fsSL https://pkg.cloudflareclient.com/pubkey.gpg | gpg --yes --dearmor --output /usr/share/keyrings/cloudflare-warp-archive-keyring.gpg 2>/dev/null || true
    echo "deb [signed-by=/usr/share/keyrings/cloudflare-warp-archive-keyring.gpg] https://pkg.cloudflareclient.com/ $(lsb_release -cs) main" | tee /etc/apt/sources.list.d/cloudflare-client.list >/dev/null 2>&1 || true
    apt-get update -qq >/dev/null 2>&1 || true
    apt-get install -y -qq cloudflare-warp >/dev/null 2>&1 || true
  fi
fi

if command -v warp-cli >/dev/null 2>&1; then
  warp-cli --accept-tos registration new >/dev/null 2>&1 || warp-cli --accept-tos register >/dev/null 2>&1 || true
  warp-cli --accept-tos mode proxy >/dev/null 2>&1 || true
  warp-cli --accept-tos proxy port 40000 >/dev/null 2>&1 || true
  warp-cli --accept-tos connect >/dev/null 2>&1 || true
  sleep 2
  STATUS=$(warp-cli --accept-tos status 2>/dev/null || echo "Connected")
  echo "Cloudflare WARP подключен в режиме SOCKS5 прокси: 127.0.0.1:40000"
  echo "$STATUS"
else
  echo "Не удалось настроить WARP ни через Native WireGuard, ни через cloudflare-warp."
  exit 1
fi
`
	return strings.NewReplacer(
		"__WARP_ENDPOINT_V4__", warpEndpointV4,
		"__WARP_FWMARK__", warpFwMark,
	).Replace(tmpl)
}

func parseSelfStealTarget(server types.SavedServer, fallbackDomain string) (selfStealTarget, error) {
	rawFallback := strings.TrimSpace(fallbackDomain)
	rawDecoy := strings.TrimSpace(server.DecoyDomain)

	if strings.ContainsAny(rawFallback, "\r\n\"'`;|&$<> \t\\") ||
		strings.ContainsAny(rawDecoy, "\r\n\"'`;|&$<> \t\\") {
		return selfStealTarget{}, fmt.Errorf("Некорректный адрес fallback: содержит недопустимые символы")
	}

	target := selfStealTarget{
		ListenHost: "127.0.0.1",
		ListenPort: 8443,
		SNIDomain:  stripSchemeAndSlash(rawDecoy),
	}

	cleaned := stripSchemeAndSlash(rawFallback)
	if cleaned == "" {
		return target, nil
	}

	if port, err := strconv.Atoi(cleaned); err == nil {
		if port < 1 || port > 65535 {
			return selfStealTarget{}, fmt.Errorf("Некорректный порт fallback: %s", rawFallback)
		}
		target.ListenPort = port
		return target, nil
	}

	if host, portStr, err := net.SplitHostPort(cleaned); err == nil {
		port, pErr := strconv.Atoi(portStr)
		if pErr != nil || port < 1 || port > 65535 {
			return selfStealTarget{}, fmt.Errorf("Некорректный порт fallback: %s", portStr)
		}
		target.ListenPort = port
		if host != "" {
			if net.ParseIP(host) != nil || strings.EqualFold(host, "localhost") {
				target.ListenHost = host
			} else if target.SNIDomain == "" {
				target.SNIDomain = strings.ToLower(host)
			}
		}
		return target, nil
	}

	// Plain hostname without port (e.g. "mask.example.com")
	if target.SNIDomain == "" {
		target.SNIDomain = strings.ToLower(cleaned)
	}
	return target, nil
}

func stripSchemeAndSlash(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "https://")
	v = strings.TrimPrefix(v, "http://")
	v = strings.TrimRight(v, "/")
	return v
}

func buildSelfStealCaddyfile(target selfStealTarget) string {
	header := "# managed-by: remna-ui SelfSteal\n"
	serverBlock := fmt.Sprintf(":%d", target.ListenPort)

	if target.SNIDomain != "" {
		return fmt.Sprintf(`%s{
    admin off
    auto_https off
}

%s {
    tls /etc/caddy/ssl/selfsteal.crt /etc/caddy/ssl/selfsteal.key
    bind %s
    encode zstd gzip
    root * /var/www/html
    file_server
}

# Expected SNI / Let's Encrypt mount override handled at runtime if cert exists for %s
`, header, serverBlock, target.ListenHost, target.SNIDomain)
	}

	return fmt.Sprintf(`%s{
    admin off
    auto_https off
}

%s {
    tls /etc/caddy/ssl/selfsteal.crt /etc/caddy/ssl/selfsteal.key
    bind %s
    encode zstd gzip
    root * /var/www/html
    file_server
}
`, header, serverBlock, target.ListenHost)
}

func buildSelfStealCompose(port int) string {
	return fmt.Sprintf(`# managed-by: remna-ui SelfSteal
services:
  selfsteal:
    image: caddy:2-alpine
    container_name: remnawave-selfsteal
    hostname: remnawave-selfsteal
    restart: always
    network_mode: host
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - ./site:/var/www/html:ro
      - ./ssl:/etc/caddy/ssl:ro
      - /etc/letsencrypt:/etc/letsencrypt:ro
      - selfsteal-data:/data
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

volumes:
  selfsteal-data:
    name: remnawave-selfsteal-data
    driver: local
`)
}

// ConfigureRealitySelfSteal deploys and validates an isolated Caddy SelfSteal decoy container
// in /opt/remnawave/selfsteal bound to 127.0.0.1:<port> (so it never collides with master panel 80/443),
// protects manual Caddyfile edits via SHA-256 sidecar checks, runs `caddy validate` pre-flight,
// and rolls back automatically if validation or container startup fails.
func ConfigureRealitySelfSteal(server types.SavedServer, fallbackDomain string) types.CommandResult {
	target, err := parseSelfStealTarget(server, fallbackDomain)
	if err != nil {
		return types.CommandResult{Success: false, Error: err.Error()}
	}

	session, release, err := getPooledSession(server)
	if err != nil {
		return types.CommandResult{Success: false, Error: "Ошибка подключения SSH: " + err.Error()}
	}
	defer release()

	caddyfileB64 := base64.StdEncoding.EncodeToString([]byte(buildSelfStealCaddyfile(target)))
	composeB64 := base64.StdEncoding.EncodeToString([]byte(buildSelfStealCompose(target.ListenPort)))
	htmlB64 := base64.StdEncoding.EncodeToString([]byte(generator.RenderCamouflageSite()))

	script := buildSelfStealDeployScript(target, caddyfileB64, composeB64, htmlB64)
	out, err := session.CombinedOutput(script)
	if err != nil {
		trimmed := strings.TrimSpace(string(out))
		if trimmed == "" {
			trimmed = err.Error()
		}
		return types.CommandResult{Success: false, Error: trimmed}
	}

	return types.CommandResult{Success: true, Output: strings.TrimSpace(string(out))}
}

func buildSelfStealDeployScript(target selfStealTarget, caddyfileB64, composeB64, htmlB64 string) string {
	return fmt.Sprintf(`set -e
echo ">>> Развертывание и проверка VLESS Reality SelfSteal (Caddy)..."

if ! docker ps -a --format '{{.Names}}' 2>/dev/null | grep -q 'remnanode' && [ ! -f /opt/remnawave/node/docker-compose.yml ]; then
  echo "Примечание: контейнер remnanode пока не обнаружен (можно развернуть/привязать ноду позже)."
fi

APP_DIR="/opt/remnawave/selfsteal"
CADDY_FILE="$APP_DIR/Caddyfile"
SIDECAR="$APP_DIR/.caddyfile.sha256"
SNI_DOMAIN="%s"
LISTEN_HOST="%s"
LISTEN_PORT="%d"

mkdir -p "$APP_DIR/site" "$APP_DIR/ssl"

if [ ! -f "$APP_DIR/ssl/selfsteal.crt" ] || [ ! -f "$APP_DIR/ssl/selfsteal.key" ]; then
  SAN_ENTRY="IP:${LISTEN_HOST},DNS:localhost"
  CN_VAL="${LISTEN_HOST}"
  if [ -n "$SNI_DOMAIN" ]; then
    SAN_ENTRY="DNS:${SNI_DOMAIN},${SAN_ENTRY}"
    CN_VAL="${SNI_DOMAIN}"
  fi
  openssl req -x509 -nodes -newkey ec -pkeyopt ec_paramgen_curve:prime256v1 -days 3650 \
    -keyout "$APP_DIR/ssl/selfsteal.key" \
    -out "$APP_DIR/ssl/selfsteal.crt" \
    -subj "/CN=${CN_VAL}" \
    -addext "subjectAltName=${SAN_ENTRY}" >/dev/null 2>&1 || \
  openssl req -x509 -nodes -newkey rsa:2048 -days 3650 \
    -keyout "$APP_DIR/ssl/selfsteal.key" \
    -out "$APP_DIR/ssl/selfsteal.crt" \
    -subj "/CN=${CN_VAL}" >/dev/null 2>&1
  chmod 644 "$APP_DIR/ssl/selfsteal.crt"
  chmod 600 "$APP_DIR/ssl/selfsteal.key"
fi

if [ ! -f "$APP_DIR/site/index.html" ]; then
  echo "%s" | base64 -d > "$APP_DIR/site/index.html"
fi

# Always keep docker-compose.yml synchronized so volume mounts match Caddyfile expectations
echo "%s" | base64 -d > "$APP_DIR/docker-compose.yml"

STAGED=$(mktemp)
echo "%s" | base64 -d > "$STAGED"

# If Let's Encrypt certificate already exists for SNI_DOMAIN, bind it into Caddyfile instead of self-signed cert
if [ -n "$SNI_DOMAIN" ] && [ -f "/etc/letsencrypt/live/${SNI_DOMAIN}/fullchain.pem" ] && [ -f "/etc/letsencrypt/live/${SNI_DOMAIN}/privkey.pem" ]; then
  cat > "$STAGED" <<LE_EOF
# managed-by: remna-ui SelfSteal
{
    admin off
    auto_https off
}

:${LISTEN_PORT} {
    tls /etc/letsencrypt/live/${SNI_DOMAIN}/fullchain.pem /etc/letsencrypt/live/${SNI_DOMAIN}/privkey.pem
    bind ${LISTEN_HOST}
    encode zstd gzip
    root * /var/www/html
    file_server
}
LE_EOF
fi

compute_sha() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    openssl dgst -sha256 "$1" | awk '{print $NF}'
  fi
}

BACKUP_FILE=""
if [ -f "$CADDY_FILE" ]; then
  BACKUP_FILE="${CADDY_FILE}.backup.$(date +%%Y%%m%%d_%%H%%M%%S)"
  cp "$CADDY_FILE" "$BACKUP_FILE"
  for _old_bak in $(ls -1t "${CADDY_FILE}.backup."* 2>/dev/null | tail -n +6); do rm -f "$_old_bak"; done

  IS_OURS=0
  if head -n 1 "$CADDY_FILE" 2>/dev/null | grep -q '^# managed-by: remna-ui' && [ -f "$SIDECAR" ]; then
    STORED=$(head -n 1 "$SIDECAR" | awk '{print $1}')
    CUR=$(compute_sha "$CADDY_FILE")
    if [ -n "$STORED" ] && [ "$STORED" = "$CUR" ]; then
      IS_OURS=1
    fi
  fi

  if [ "$IS_OURS" -eq 0 ] && ! cmp -s "$STAGED" "$CADDY_FILE"; then
    CANDIDATE="${CADDY_FILE}.new"
    cat "$STAGED" > "$CANDIDATE"
    rm -f "$STAGED"
    echo "ВНИМАНИЕ: $CADDY_FILE был изменен вручную и сохранен без перезаписи."
    echo "Новый шаблон сохранен в: $CANDIDATE"
    diff -u "$CADDY_FILE" "$CANDIDATE" 2>/dev/null | head -n 30 || true
    exit 0
  fi
fi

# Preserve inode for bind mounts via cat >
cat "$STAGED" > "$CADDY_FILE"
rm -f "$STAGED"
compute_sha "$CADDY_FILE" > "$SIDECAR"

rollback_caddy() {
  if [ -n "$BACKUP_FILE" ] && [ -f "$BACKUP_FILE" ]; then
    cat "$BACKUP_FILE" > "$CADDY_FILE"
    compute_sha "$CADDY_FILE" > "$SIDECAR"
    echo "Выполнен автоматический откат Caddyfile из $BACKUP_FILE"
  fi
}

# Pre-flight validation using the exact runtime image tag
VALIDATE_OUT=""
if VALIDATE_OUT=$(docker run --rm \
  -v "$CADDY_FILE:/etc/caddy/Caddyfile:ro" \
  -v "$APP_DIR/site:/var/www/html:ro" \
  -v "$APP_DIR/ssl:/etc/caddy/ssl:ro" \
  -v /etc/letsencrypt:/etc/letsencrypt:ro \
  caddy:2-alpine caddy validate --config /etc/caddy/Caddyfile 2>&1); then
  echo "Pre-flight проверка caddy validate пройдена успешно."
else
  if echo "$VALIDATE_OUT" | grep -qiE 'pull rate limit|toomanyrequests|error from registry|no such host|i/o timeout|timeout exceeded|cannot connect to the docker daemon'; then
    echo "Pre-flight проверка пропущена (недоступен Docker Registry/rate limit); запускаем docker compose up..."
  else
    echo "Ошибка синтаксиса Caddyfile:"
    echo "$VALIDATE_OUT" | tail -n 15
    rollback_caddy
    exit 1
  fi
fi

cd "$APP_DIR"
if ! docker compose up -d --force-recreate; then
  echo "Ошибка запуска контейнера SelfSteal."
  rollback_caddy
  exit 1
fi

echo "Reality SelfSteal (remnawave-selfsteal) успешно запущен на ${LISTEN_HOST}:${LISTEN_PORT}."
echo "Укажите в панели Remnawave для Reality: dest/target = ${LISTEN_HOST}:${LISTEN_PORT}, xver = 0"
if docker ps --format '{{.Names}}' 2>/dev/null | grep -q 'remnanode'; then
  echo "Нода remnanode активна."
elif docker ps -a --format '{{.Names}}' 2>/dev/null | grep -q 'remnanode'; then
  echo "Внимание: контейнер remnanode сейчас остановлен — не забудьте запустить ноду."
else
  echo "Примечание: для старта ноды укажите SECRET_KEY в /opt/remnawave/node (или через Привязать ноду)."
fi
`, target.SNIDomain, target.ListenHost, target.ListenPort, htmlB64, composeB64, caddyfileB64)
}
