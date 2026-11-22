package generator

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"remna-ui/internal/dns"
)

func randomHex(bytes int) string {
	b := make([]byte, bytes)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func RandomHex(bytes int) string {
	return randomHex(bytes)
}

func randInt(max int) int {
	if max <= 0 {
		return 0
	}
	b := make([]byte, 1)
	_, _ = rand.Read(b)
	return int(b[0]) % max
}

// RandomAdminPassword generates a secure password meeting Remnawave backend requirements:
// minimum 24 chars, contains uppercase, lowercase, and digits: /^(?=.*?[A-Z])(?=.*?[a-z])(?=.*?[0-9]).{24,}$/
func RandomAdminPassword(length int) string {
	if length < 28 {
		length = 28
	}
	const (
		upper   = "ABCDEFGHJKLMNPQRSTUVWXYZ"
		lower   = "abcdefghijkmnopqrstuvwxyz"
		digits  = "23456789"
		symbols = "_-"
	)
	all := upper + lower + digits + symbols

	b := make([]byte, length)
	b[0] = upper[randInt(len(upper))]
	b[1] = upper[randInt(len(upper))]
	b[2] = lower[randInt(len(lower))]
	b[3] = lower[randInt(len(lower))]
	b[4] = digits[randInt(len(digits))]
	b[5] = digits[randInt(len(digits))]
	b[6] = symbols[randInt(len(symbols))]
	for i := 7; i < length; i++ {
		b[i] = all[randInt(len(all))]
	}

	// Fisher-Yates shuffle
	for i := len(b) - 1; i > 0; i-- {
		j := randInt(i + 1)
		b[i], b[j] = b[j], b[i]
	}
	return string(b)
}

func RenderPanelEnvWithCreds(panelDomain, subDomain string, subEnabled bool, appSecret, dbPass, metricsPass, webhookSecret string) string {
	domain := dns.NormalizeDomain(panelDomain)
	sub := dns.NormalizeDomain(subDomain)

	if appSecret == "" {
		appSecret = randomHex(32)
	}
	if dbPass == "" {
		dbPass = randomHex(16)
	}
	if metricsPass == "" {
		metricsPass = randomHex(16)
	}
	if webhookSecret == "" {
		webhookSecret = randomHex(16)
	}

	subPublicDomain := ""
	if subEnabled && sub != "" && sub != domain {
		subPublicDomain = sub
	}

	return fmt.Sprintf(`APP_PORT=3000
METRICS_PORT=3001
API_INSTANCES=1
TRUST_PROXY=true
APP_SECRET=%s
METRICS_USER=admin
METRICS_PASS=%s
WEBHOOK_ENABLED=false
WEBHOOK_SECRET_HEADER=%s
POSTGRES_USER=postgres
POSTGRES_PASSWORD=%s
POSTGRES_DB=postgres
DATABASE_URL=postgresql://postgres:%s@remnawave-db:5432/postgres
REDIS_SOCKET=/var/run/valkey/valkey.sock
PANEL_DOMAIN=%s
FRONT_END_DOMAIN=%s
SUB_PUBLIC_DOMAIN=%s
JWT_AUTH_LIFETIME=168
`, appSecret, metricsPass, webhookSecret, dbPass, dbPass, domain, domain, subPublicDomain)
}

func RenderPanelEnv(panelDomain, subDomain string, subEnabled bool) string {
	return RenderPanelEnvWithCreds(panelDomain, subDomain, subEnabled, "", "", "", "")
}

func RenderPanelEnvStandardWithCreds(panelDomain, subDomain string, subEnabled bool, appSecret, dbPass, metricsPass, webhookSecret string) string {
	domain := dns.NormalizeDomain(panelDomain)
	sub := dns.NormalizeDomain(subDomain)

	if appSecret == "" {
		appSecret = randomHex(32)
	}
	if dbPass == "" {
		dbPass = randomHex(16)
	}
	if metricsPass == "" {
		metricsPass = randomHex(16)
	}
	if webhookSecret == "" {
		webhookSecret = randomHex(16)
	}

	subPublicDomain := ""
	if subEnabled && sub != "" && sub != domain {
		subPublicDomain = sub
	}

	return fmt.Sprintf(`APP_PORT=3000
METRICS_PORT=3001
API_INSTANCES=1
TRUST_PROXY=true
APP_SECRET=%s
METRICS_USER=admin
METRICS_PASS=%s
WEBHOOK_ENABLED=false
WEBHOOK_SECRET_HEADER=%s
POSTGRES_USER=postgres
POSTGRES_PASSWORD=%s
POSTGRES_DB=postgres
DATABASE_URL=postgresql://postgres:%s@remnawave-db:5432/postgres
#REDIS_SOCKET=
REDIS_HOST=remnawave-redis
REDIS_PORT=6379
PANEL_DOMAIN=%s
FRONT_END_DOMAIN=%s
SUB_PUBLIC_DOMAIN=%s
JWT_AUTH_LIFETIME=168
`, appSecret, metricsPass, webhookSecret, dbPass, dbPass, domain, domain, subPublicDomain)
}

func RenderPanelEnvStandard(panelDomain, subDomain string, subEnabled bool) string {
	return RenderPanelEnvStandardWithCreds(panelDomain, subDomain, subEnabled, "", "", "", "")
}

func RenderPanelCompose() string {
	return `services:
  remnawave:
    image: remnawave/backend:3
    container_name: remnawave
    hostname: remnawave
    restart: always
    ulimits:
      nofile:
        soft: 1048576
        hard: 1048576
    environment:
      - NODE_OPTIONS=--max-old-space-size=512
      - REDIS_SOCKET=/var/run/valkey/valkey.sock
    ports:
      - "127.0.0.1:3000:3000"
      - "127.0.0.1:3001:3001"
    env_file:
      - .env
    networks:
      - remnawave-network
    volumes:
      - valkey-socket:/var/run/valkey
    depends_on:
      remnawave-db:
        condition: service_healthy
      remnawave-redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD-SHELL", "curl -f http://localhost:3001/health || exit 1"]
      interval: 15s
      timeout: 5s
      retries: 3
      start_period: 25s
    logging:
      driver: "json-file"
      options:
        max-size: "20m"
        max-file: "3"

  remnawave-db:
    image: postgres:18.4
    container_name: remnawave-db
    hostname: remnawave-db
    restart: always
    shm_size: 512mb
    ulimits:
      nofile:
        soft: 1048576
        hard: 1048576
    env_file:
      - .env
    environment:
      - POSTGRES_USER=${POSTGRES_USER}
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
      - POSTGRES_DB=${POSTGRES_DB}
      - TZ=UTC
    networks:
      - remnawave-network
    ports:
      - 127.0.0.1:6767:5432
    volumes:
      - remnawave-db-data:/var/lib/postgresql
    healthcheck:
      test: ['CMD-SHELL', 'pg_isready -U $${POSTGRES_USER} -d $${POSTGRES_DB} || pg_isready -h localhost || pg_isready']
      interval: 3s
      timeout: 10s
      retries: 10
      start_period: 15s
    logging:
      driver: "json-file"
      options:
        max-size: "20m"
        max-file: "3"

  remnawave-redis:
    image: valkey/valkey:9-alpine
    container_name: remnawave-redis
    hostname: remnawave-redis
    restart: always
    ulimits:
      nofile:
        soft: 1048576
        hard: 1048576
    command:
      - valkey-server
      - --save
      - ""
      - --appendonly
      - "no"
      - --maxmemory
      - "256mb"
      - --maxmemory-policy
      - "allkeys-lru"
      - --loglevel
      - warning
      - --unixsocket
      - /var/run/valkey/valkey.sock
      - --unixsocketperm
      - "777"
      - --port
      - "0"
    networks:
      - remnawave-network
    volumes:
      - valkey-socket:/var/run/valkey
    healthcheck:
      test: ["CMD", "valkey-cli", "-s", "/var/run/valkey/valkey.sock", "ping"]
      interval: 3s
      timeout: 5s
      retries: 5
    logging:
      driver: "json-file"
      options:
        max-size: "20m"
        max-file: "3"

volumes:
  remnawave-db-data:
    name: remnawave-db-data
    driver: local
    external: false
  valkey-socket:
    name: valkey-socket
    driver: local
    external: false

networks:
  remnawave-network:
    name: remnawave-network
    driver: bridge
    external: false
`
}

func RenderPanelComposeStandard() string {
	return `services:
  remnawave:
    image: remnawave/backend:3
    container_name: remnawave
    hostname: remnawave
    restart: always
    environment:
      - REDIS_HOST=remnawave-redis
      - REDIS_PORT=6379
    ports:
      - "127.0.0.1:3000:3000"
      - "127.0.0.1:3001:3001"
    env_file:
      - .env
    networks:
      - remnawave-network
    depends_on:
      remnawave-db:
        condition: service_healthy
      remnawave-redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD-SHELL", "curl -f http://localhost:3001/health || exit 1"]
      interval: 15s
      timeout: 5s
      retries: 3
      start_period: 25s
    logging:
      driver: "json-file"
      options:
        max-size: "20m"
        max-file: "3"

  remnawave-db:
    image: postgres:18.4
    container_name: remnawave-db
    hostname: remnawave-db
    restart: always
    env_file:
      - .env
    environment:
      - POSTGRES_USER=${POSTGRES_USER}
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
      - POSTGRES_DB=${POSTGRES_DB}
      - TZ=UTC
    networks:
      - remnawave-network
    ports:
      - 127.0.0.1:6767:5432
    volumes:
      - remnawave-db-data:/var/lib/postgresql
    healthcheck:
      test: ['CMD-SHELL', 'pg_isready -U $${POSTGRES_USER} -d $${POSTGRES_DB} || pg_isready -h localhost || pg_isready']
      interval: 3s
      timeout: 10s
      retries: 10
      start_period: 15s
    logging:
      driver: "json-file"
      options:
        max-size: "20m"
        max-file: "3"

  remnawave-redis:
    image: valkey/valkey:9-alpine
    container_name: remnawave-redis
    hostname: remnawave-redis
    restart: always
    command:
      - valkey-server
      - --save
      - ""
      - --appendonly
      - "no"
      - --loglevel
      - warning
    networks:
      - remnawave-network
    healthcheck:
      test: ["CMD", "valkey-cli", "ping"]
      interval: 3s
      timeout: 5s
      retries: 5
    logging:
      driver: "json-file"
      options:
        max-size: "20m"
        max-file: "3"

volumes:
  remnawave-db-data:
    name: remnawave-db-data
    driver: local
    external: false

networks:
  remnawave-network:
    name: remnawave-network
    driver: bridge
    external: false
`
}

func RenderNodeCompose(nodePort int, secretKey string) string {
	if secretKey == "" {
		secretKey = "REPLACE_WITH_SECRET_KEY_FROM_PANEL"
	}
	return fmt.Sprintf(`services:
  remnanode:
    image: remnawave/node:latest
    container_name: remnanode
    hostname: remnanode
    restart: always
    network_mode: host
    cap_add:
      - NET_ADMIN
    environment:
      NODE_PORT: "%d"
      SECRET_KEY: "%s"
    logging:
      driver: "json-file"
      options:
        max-size: "20m"
        max-file: "3"
`, nodePort, secretKey)
}

func RenderNginxConf(domain, subDomain string, subEnabled bool, secretGate string, useCamouflage bool) string {
	cleanDomain := dns.NormalizeDomain(domain)
	cleanSub := dns.NormalizeDomain(subDomain)

	redirectDomains := cleanDomain
	if subEnabled && cleanSub != "" && cleanSub != cleanDomain {
		redirectDomains = fmt.Sprintf("%s %s", cleanDomain, cleanSub)
	}

	subServer := ""
	if subEnabled && cleanSub != "" && cleanSub != cleanDomain {
		subServer = fmt.Sprintf(`
upstream remnawave-subscription-page {
    server remnawave-subscription-page:3010;
}

server {
    server_name %s;
    listen 443 ssl;
    listen [::]:443 ssl;
    http2 on;

    ssl_certificate /etc/nginx/ssl/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/privkey.key;
    ssl_protocols TLSv1.2 TLSv1.3;

    gzip on;
    gzip_vary on;
    gzip_proxied any;
    gzip_comp_level 6;
    gzip_types text/plain text/css text/xml application/json application/javascript application/rss+xml application/atom+xml image/svg+xml application/wasm;

    client_max_body_size 50m;

    location / {
        proxy_http_version 1.1;
        proxy_pass http://remnawave-subscription-page;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
    }
}
`, cleanSub)
	}

	gateBlock := ""
	gateCheck := ""
	camoLocation := ""
	if secretGate != "" {
		cleanGate := strings.TrimLeft(strings.TrimSpace(secretGate), "/")
		gateBlock = fmt.Sprintf(`
    # eGames Security Gate
    location = /%s {
        add_header Set-Cookie "remna_gate=%s; Path=/; HttpOnly; SameSite=Lax; Max-Age=86400; Secure";
        return 302 /;
    }

    location ^~ /api/ {
        proxy_http_version 1.1;
        proxy_pass http://remnawave;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-Host $host;
        proxy_set_header X-Forwarded-Port $server_port;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
    }

    # Public client subscription links bypass security gate
    location ^~ /sub/ {
        proxy_http_version 1.1;
        proxy_pass http://remnawave;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-Host $host;
        proxy_set_header X-Forwarded-Port $server_port;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
    }
`, cleanGate, cleanGate)

		if useCamouflage {
			gateCheck = fmt.Sprintf(`
        error_page 418 = @camo;
        if ($cookie_remna_gate != "%s") {
            return 418;
        }`, cleanGate)
			camoLocation = `
    location @camo {
        root /var/www/html;
        try_files $uri $uri/ /index.html =404;
    }
`
		} else {
			gateCheck = fmt.Sprintf(`
        if ($cookie_remna_gate != "%s") {
            return 404;
        }`, cleanGate)
		}
	}

	return fmt.Sprintf(`map $http_upgrade $connection_upgrade {
    default upgrade;
    "" close;
}

upstream remnawave {
    server remnawave:3000;
}

server {
    listen 80;
    listen [::]:80;
    server_name %s;
    return 301 https://$host$request_uri;
}

server {
    server_name %s;
    listen 443 ssl;
    listen [::]:443 ssl;
    http2 on;

    ssl_certificate /etc/nginx/ssl/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/privkey.key;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_prefer_server_ciphers off;

    gzip on;
    gzip_vary on;
    gzip_proxied any;
    gzip_comp_level 6;
    gzip_types text/plain text/css text/xml application/json application/javascript application/rss+xml application/atom+xml image/svg+xml application/wasm;

    client_max_body_size 50m;
%s
    # SSE Stream for real-time users list updates
    location = /api/users/stream {
        proxy_http_version 1.1;
        proxy_pass http://remnawave;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 86400s;
        proxy_send_timeout 86400s;
        chunked_transfer_encoding off;
    }

    location / {%s
        proxy_http_version 1.1;
        proxy_pass http://remnawave;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-Host $host;
        proxy_set_header X-Forwarded-Port $server_port;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_buffering off;
    }
%s}
%s`, redirectDomains, cleanDomain, gateBlock, gateCheck, camoLocation, subServer)
}

func RenderNginxCompose() string {
	return `services:
  remnawave-nginx:
    image: nginx:alpine
    container_name: remnawave-nginx
    hostname: remnawave-nginx
    restart: always
    ports:
      - "0.0.0.0:80:80"
      - "0.0.0.0:443:443"
    networks:
      - remnawave-network
    volumes:
      - ./nginx.conf:/etc/nginx/conf.d/default.conf:ro
      - ./fullchain.pem:/etc/nginx/ssl/fullchain.pem:ro
      - ./privkey.key:/etc/nginx/ssl/privkey.key:ro
      - ./site:/var/www/html:ro

networks:
  remnawave-network:
    name: remnawave-network
    driver: bridge
    external: true
`
}

func RenderCaddyfile(domain, subDomain string, subEnabled bool, secretGate string, useCamouflage bool) string {
	cleanDomain := dns.NormalizeDomain(domain)
	cleanSub := dns.NormalizeDomain(subDomain)

	subBlock := ""
	if subEnabled && cleanSub != "" && cleanSub != cleanDomain {
		subBlock = fmt.Sprintf(`
%s {
    encode zstd gzip
    reverse_proxy remnawave-subscription-page:3010
}
`, cleanSub)
	}

	if secretGate == "" {
		return fmt.Sprintf(`%s {
    encode zstd gzip
    reverse_proxy remnawave:3000
}
%s`, cleanDomain, subBlock)
	}

	cleanGate := strings.TrimLeft(strings.TrimSpace(secretGate), "/")
	fallbackAction := `respond "Not Found" 404`
	if useCamouflage {
		fallbackAction = `root * /var/www/html
        file_server`
	}

	return fmt.Sprintf(`%s {
    encode zstd gzip

    @gate_entry path /%s
    handle @gate_entry {
        header Set-Cookie "remna_gate=%s; Path=/; HttpOnly; SameSite=Lax; Max-Age=86400; Secure"
        redir / 302
    }

    @public_calls path /api/* /sub/*
    handle @public_calls {
        reverse_proxy remnawave:3000
    }

    @has_gate header_regexp Cookie remna_gate=[a-zA-Z0-9_.-]+
    handle @has_gate {
        reverse_proxy remnawave:3000
    }

    handle {
        %s
    }
}
%s`, cleanDomain, cleanGate, cleanGate, fallbackAction, subBlock)
}

func RenderCaddyCompose() string {
	return `services:
  caddy:
    image: caddy:2-alpine
    container_name: remnawave-caddy
    hostname: remnawave-caddy
    restart: always
    ports:
      - "80:80"
      - "443:443"
    networks:
      - remnawave-network
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - ./site:/var/www/html:ro
      - caddy-data:/data
      - caddy-config:/config
    logging:
      driver: "json-file"
      options:
        max-size: "20m"
        max-file: "3"

volumes:
  caddy-data:
    name: remnawave-caddy-data
    driver: local
  caddy-config:
    name: remnawave-caddy-config
    driver: local

networks:
  remnawave-network:
    name: remnawave-network
    driver: bridge
    external: true
`
}

func RenderSubscriptionEnv(panelDomain, apiToken, subMode string) string {
	cleanPanel := dns.NormalizeDomain(panelDomain)
	if apiToken == "" {
		apiToken = "REPLACE_WITH_API_TOKEN"
	}
	url := fmt.Sprintf("https://%s", cleanPanel)
	if subMode == "bundled" {
		url = "http://remnawave:3000"
	}

	return fmt.Sprintf(`PORT=3010
APP_PORT=3010
HOST=0.0.0.0
TRUST_PROXY=true
REMNAWAVE_PANEL_URL=%s
REMNAWAVE_URL=%s
REMNAWAVE_API_TOKEN=%s
`, url, url, apiToken)
}

func RenderSubscriptionCompose(subMode string) string {
	external := "true"
	if subMode == "separate" {
		external = "false"
	}
	return fmt.Sprintf(`services:
  remnawave-subscription-page:
    image: remnawave/subscription-page:latest
    container_name: remnawave-subscription-page
    hostname: remnawave-subscription-page
    restart: always
    env_file:
      - .env
    networks:
      - remnawave-network
    logging:
      driver: "json-file"
      options:
        max-size: "20m"
        max-file: "3"

networks:
  remnawave-network:
    name: remnawave-network
    driver: bridge
    external: %s
`, external)
}

func RenderCamouflageSite() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Enterprise Cloud Infrastructure & Network Solutions</title>
    <style>
        :root {
            --bg: #0d1117;
            --surface: #161b22;
            --border: #30363d;
            --text: #c9d1d9;
            --text-strong: #f0f6fc;
            --accent: #58a6ff;
        }
        body {
            margin: 0;
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
            background: var(--bg);
            color: var(--text);
            line-height: 1.6;
        }
        .header {
            border-bottom: 1px solid var(--border);
            padding: 1.25rem 2rem;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .logo {
            font-size: 1.2rem;
            font-weight: 700;
            color: var(--text-strong);
            letter-spacing: -0.5px;
        }
        .nav-links a {
            color: var(--text);
            text-decoration: none;
            margin-left: 1.5rem;
            font-size: 0.9rem;
            transition: color 0.2s;
        }
        .nav-links a:hover {
            color: var(--accent);
        }
        .container {
            max-width: 960px;
            margin: 3.5rem auto;
            padding: 0 1.5rem;
        }
        .hero {
            text-align: center;
            margin-bottom: 3.5rem;
        }
        .hero h1 {
            font-size: 2.5rem;
            color: var(--text-strong);
            margin-bottom: 1rem;
            letter-spacing: -0.5px;
        }
        .hero p {
            font-size: 1.15rem;
            color: #8b949e;
            max-width: 620px;
            margin: 0 auto;
        }
        .grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
            gap: 1.5rem;
            margin-top: 2rem;
        }
        .card {
            background: var(--surface);
            border: 1px solid var(--border);
            border-radius: 8px;
            padding: 1.5rem;
        }
        .card h3 {
            color: var(--text-strong);
            margin-top: 0;
            font-size: 1.15rem;
        }
        .footer {
            border-top: 1px solid var(--border);
            text-align: center;
            padding: 2rem;
            margin-top: 4rem;
            color: #8b949e;
            font-size: 0.85rem;
        }
    </style>
</head>
<body>
    <header class="header">
        <div class="logo">Apex Networks</div>
        <nav class="nav-links">
            <a href="#services">Services</a>
            <a href="#architecture">Architecture</a>
            <a href="#status">Network Status</a>
        </nav>
    </header>
    <main class="container">
        <section class="hero">
            <h1>Global Low-Latency Edge Delivery</h1>
            <p>High-reliability distributed edge routing, TLS 1.3 cryptographic acceleration, and automated failover architecture.</p>
        </section>
        <section id="services" class="grid">
            <div class="card">
                <h3>Anycast Edge Transit</h3>
                <p>Distributed point-of-presence network topology optimizing packet traversal across global fiber backbones.</p>
            </div>
            <div class="card">
                <h3>Hardware Cryptography</h3>
                <p>TLS 1.3 zero-RTT session resumption and cryptographic termination for modern web applications.</p>
            </div>
            <div class="card">
                <h3>99.99&#37; Availability</h3>
                <p>Multi-homed BGP peering agreements and continuous real-time health-checking telemetry.</p>
            </div>
        </section>
    </main>
    <footer class="footer">
        <p>&copy; 2026 Apex Networks International Inc. All rights reserved.</p>
    </footer>
</body>
</html>
`
}

func RenderXraySample() string {
	return `{
  "log": {
    "loglevel": "warning"
  },
  "inbounds": [
    {
      "port": 443,
      "protocol": "vless",
      "settings": {
        "clients": [],
        "decryption": "none",
        "fallbacks": [
          {
            "dest": 8080,
            "xver": 1
          }
        ]
      },
      "streamSettings": {
        "network": "tcp",
        "security": "reality",
        "realitySettings": {
          "show": false,
          "dest": "www.microsoft.com:443",
          "xver": 0,
          "serverNames": [
            "www.microsoft.com"
          ],
          "privateKey": "GENERATE_VIA_XRAY_X25519",
          "shortIds": [
            ""
          ]
        }
      }
    }
  ],
  "outbounds": [
    {
      "protocol": "freedom"
    }
  ]
}
`
}

func RenderDeploySh() string {
	return `#!/usr/bin/env bash
# Auto-generated deployment script for Remnawave stack
set -euo pipefail

echo "========================================="
echo "   Remnawave Stack Automatic Deployer    "
echo "========================================="

if ! command -v docker >/dev/null 2>&1; then
    echo "==> Docker not found. Installing official Docker..."
    curl -fsSL https://get.docker.com | sh
    systemctl enable --now docker
fi

if ! docker compose version >/dev/null 2>&1; then
    echo "==> Installing Docker Compose plugin..."
    apt-get update && apt-get install -y docker-compose-plugin
fi

cd "$(dirname "$0")"

if [ -d "./panel" ]; then
    echo "==> Starting Panel stack..."
    (cd panel && docker compose up -d)
fi

if [ -d "./subscription" ]; then
    echo "==> Starting Subscription page..."
    (cd subscription && docker compose up -d)
fi

if [ -d "./nginx" ]; then
    echo "==> Starting Nginx reverse proxy..."
    (cd nginx && docker compose up -d)
elif [ -d "./caddy" ]; then
    echo "==> Starting Caddy reverse proxy..."
    (cd caddy && docker compose up -d)
fi

if [ -d "./node" ]; then
    echo "==> Starting Node..."
    (cd node && docker compose up -d)
fi

echo ""
echo "==> All configured components have been deployed successfully!"
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
`
}

func RenderDeployPs1() string {
	return `# PowerShell helper to transfer files and deploy to a Linux VPS
param(
    [Parameter(Mandatory=$true, Position=0)]
    [string]$Server
)

$targetDir = "/opt/remnawave"
Write-Host "==> Copying setup files to $Server:$targetDir ..." -ForegroundColor Cyan
ssh $Server "mkdir -p $targetDir"
scp -r ./* "${Server}:${targetDir}/"

Write-Host "==> Running deploy.sh on remote server ..." -ForegroundColor Cyan
ssh $Server "cd $targetDir && chmod +x deploy.sh && ./deploy.sh"
`
}

func RenderStealthNginxConf(panelDomain, subDomain, decoyDomain, secretGate string, useCamouflage bool) string {
	cleanPanel := dns.NormalizeDomain(panelDomain)
	cleanSub := dns.NormalizeDomain(subDomain)
	cleanDecoy := dns.NormalizeDomain(decoyDomain)
	if cleanDecoy == "" {
		cleanDecoy = cleanPanel
	}

	gateBlock := ""
	gateCheck := ""
	camoLocation := ""
	if secretGate != "" {
		cleanGate := strings.TrimLeft(strings.TrimSpace(secretGate), "/")
		gateBlock = fmt.Sprintf(`
    # eGames Security Gate
    location = /%s {
        add_header Set-Cookie "remna_gate=%s; Path=/; HttpOnly; SameSite=Lax; Max-Age=31536000; Secure";
        return 302 /;
    }

    location ^~ /api/ {
        proxy_http_version 1.1;
        proxy_pass http://remnawave;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $proxy_protocol_addr;
        proxy_set_header X-Forwarded-For $proxy_protocol_addr;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-Host $host;
        proxy_set_header X-Forwarded-Port $server_port;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
    }

    # Public client subscription links bypass security gate
    location ^~ /sub/ {
        proxy_http_version 1.1;
        proxy_pass http://remnawave;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $proxy_protocol_addr;
        proxy_set_header X-Forwarded-For $proxy_protocol_addr;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-Host $host;
        proxy_set_header X-Forwarded-Port $server_port;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
    }
`, cleanGate, cleanGate)

		if useCamouflage {
			gateCheck = fmt.Sprintf(`
        error_page 418 = @camo;
        if ($cookie_remna_gate != "%s") {
            return 418;
        }`, cleanGate)
			camoLocation = `
    location @camo {
        root /var/www/html;
        try_files $uri $uri/ /index.html =404;
    }
`
		} else {
			gateCheck = fmt.Sprintf(`
        if ($cookie_remna_gate != "%s") {
            return 404;
        }`, cleanGate)
		}
	}

	subBlock := ""
	if cleanSub != "" && cleanSub != cleanPanel {
		subBlock = fmt.Sprintf(`
server {
    server_name %s;
    listen unix:/dev/shm/nginx.sock ssl proxy_protocol;
    http2 on;
    gzip on;

    ssl_certificate /etc/nginx/ssl/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/privkey.key;
    ssl_protocols TLSv1.2 TLSv1.3;

    location / {
        proxy_http_version 1.1;
        proxy_pass http://json;
        proxy_set_header Host $host;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_set_header X-Real-IP $proxy_protocol_addr;
        proxy_set_header X-Forwarded-For $proxy_protocol_addr;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-Host $host;
        proxy_set_header X-Forwarded-Port $server_port;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
        proxy_intercept_errors on;
        error_page 400 404 500 502 =444 @drop;
    }

    location @drop {
        return 444;
    }
}
`, cleanSub)
	}

	decoyBlock := ""
	if cleanDecoy != "" && cleanDecoy != cleanPanel && cleanDecoy != cleanSub {
		decoyBlock = fmt.Sprintf(`
server {
    server_name %s;
    listen unix:/dev/shm/nginx.sock ssl proxy_protocol;
    http2 on;

    ssl_certificate /etc/nginx/ssl/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/privkey.key;
    ssl_protocols TLSv1.2 TLSv1.3;

    root /var/www/html;
    index index.html;
    add_header X-Robots-Tag "noindex, nofollow, noarchive, nosnippet, noimageindex" always;
}
`, cleanDecoy)
	}

	return fmt.Sprintf(`server_names_hash_bucket_size 64;

gzip on;
gzip_vary on;
gzip_proxied any;
gzip_comp_level 6;
gzip_min_length 1024;
gzip_types
    application/javascript
    application/json
    application/manifest+json
    application/xml
    application/wasm
    font/opentype
    font/eot
    font/otf
    font/ttf
    image/svg+xml
    text/css
    text/javascript
    text/plain
    text/xml;

upstream remnawave {
    server 127.0.0.1:3000;
}

upstream json {
    server 127.0.0.1:3010;
}

map $http_upgrade $connection_upgrade {
    default upgrade;
    ""      close;
}

# HTTP 80 for Let's Encrypt Certbot challenge and redirect
server {
    listen 80;
    listen [::]:80;
    server_name _;
    location /.well-known/acme-challenge/ {
        root /var/www/html;
    }
    location / {
        return 301 https://$host$request_uri;
    }
}

# Master Admin Panel
server {
    server_name %s;
    listen unix:/dev/shm/nginx.sock ssl proxy_protocol;
    http2 on;

    ssl_certificate /etc/nginx/ssl/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/privkey.key;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_prefer_server_ciphers off;

    client_max_body_size 50m;
%s
    # SSE Stream for real-time users list updates
    location = /api/users/stream {
        proxy_http_version 1.1;
        proxy_pass http://remnawave;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $proxy_protocol_addr;
        proxy_set_header X-Forwarded-For $proxy_protocol_addr;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 86400s;
        proxy_send_timeout 86400s;
        chunked_transfer_encoding off;
    }

    location / {%s
        proxy_http_version 1.1;
        proxy_pass http://remnawave;
        proxy_set_header Host $host;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_set_header X-Real-IP $proxy_protocol_addr;
        proxy_set_header X-Forwarded-For $proxy_protocol_addr;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-Host $host;
        proxy_set_header X-Forwarded-Port $server_port;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }
%s}
%s%s
# Default Server - Reject all unrecognized SNI and raw IP scanners
server {
    listen unix:/dev/shm/nginx.sock ssl proxy_protocol default_server;
    server_name _;
    add_header X-Robots-Tag "noindex, nofollow, noarchive, nosnippet, noimageindex" always;
    ssl_reject_handshake on;
    return 444;
}
`, cleanPanel, gateBlock, gateCheck, camoLocation, subBlock, decoyBlock)
}

func RenderAllInOneStealthCompose(appSecret, dbPass, metricsPass, apiToken, secretKey, xrayCoreType string) string {
	if secretKey == "" {
		secretKey = "PUBLIC KEY FROM REMNAWAVE-PANEL"
	}
	customXrayMount := ""
	if xrayCoreType == "jolymmiles" {
		customXrayMount = "      - ./xray-custom:/usr/local/bin/xray\n"
	}

	return fmt.Sprintf(`services:
  remnawave-db:
    image: postgres:18.4
    container_name: remnawave-db
    hostname: remnawave-db
    restart: always
    shm_size: 512mb
    environment:
      - POSTGRES_USER=postgres
      - POSTGRES_PASSWORD=%s
      - POSTGRES_DB=postgres
      - TZ=UTC
    ports:
      - '127.0.0.1:6767:5432'
    volumes:
      - remnawave-db-data:/var/lib/postgresql
    healthcheck:
      test: ['CMD-SHELL', 'pg_isready -U postgres -d postgres || pg_isready -h localhost || pg_isready']
      interval: 3s
      timeout: 10s
      retries: 5
    networks:
      - remnawave-network

  remnawave-redis:
    image: valkey/valkey:9-alpine
    container_name: remnawave-redis
    hostname: remnawave-redis
    restart: always
    command: >
      valkey-server
      --save ""
      --appendonly no
      --maxmemory-policy noeviction
      --loglevel warning
      --unixsocket /var/run/valkey/valkey.sock
      --unixsocketperm 777
      --port 0
    volumes:
      - valkey-socket:/var/run/valkey
    healthcheck:
      test: ['CMD', 'valkey-cli', '-s', '/var/run/valkey/valkey.sock', 'ping']
      interval: 3s
      timeout: 5s
      retries: 3
    networks:
      - remnawave-network

  remnawave:
    image: remnawave/backend:3
    container_name: remnawave
    hostname: remnawave
    restart: always
    environment:
      - NODE_OPTIONS=--max-old-space-size=512
      - REDIS_SOCKET=/var/run/valkey/valkey.sock
    ports:
      - '127.0.0.1:3000:3000'
      - '127.0.0.1:3001:3001'
    env_file:
      - .env
    volumes:
      - valkey-socket:/var/run/valkey
    depends_on:
      remnawave-db:
        condition: service_healthy
      remnawave-redis:
        condition: service_healthy
    healthcheck:
      test: ['CMD-SHELL', 'curl -f http://localhost:3001/health || exit 1']
      interval: 15s
      timeout: 5s
      retries: 3
      start_period: 25s
    networks:
      - remnawave-network

  remnawave-nginx:
    image: nginx:alpine
    container_name: remnawave-nginx
    hostname: remnawave-nginx
    restart: always
    network_mode: host
    volumes:
      - ./nginx.conf:/etc/nginx/conf.d/default.conf:ro
      - ./ssl:/etc/nginx/ssl:ro
      - /dev/shm:/dev/shm:rw
      - /var/www/html:/var/www/html:ro
    command: sh -c 'rm -f /dev/shm/nginx.sock && exec nginx -g "daemon off;"'
    depends_on:
      - remnawave

  remnawave-subscription-page:
    image: remnawave/subscription-page:latest
    container_name: remnawave-subscription-page
    hostname: remnawave-subscription-page
    restart: always
    depends_on:
      remnawave:
        condition: service_healthy
    environment:
      - REMNAWAVE_PANEL_URL=http://127.0.0.1:3000
      - APP_PORT=3010
      - REMNAWAVE_API_TOKEN=%s
    network_mode: host

  remnanode:
    image: remnawave/node:latest
    container_name: remnanode
    hostname: remnanode
    restart: always
    network_mode: host
    cap_add:
      - NET_ADMIN
    environment:
      - NODE_PORT=2222
      - SECRET_KEY=%s
    volumes:
      - /dev/shm:/dev/shm:rw
%s
volumes:
  remnawave-db-data:
    name: remnawave-db-data
  valkey-socket:
    name: valkey-socket

networks:
  remnawave-network:
    name: remnawave-network
    driver: bridge
`, dbPass, apiToken, secretKey, customXrayMount)
}

