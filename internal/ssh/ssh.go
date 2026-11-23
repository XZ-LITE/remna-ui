package ssh

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"remna-ui/internal/admin"
	"remna-ui/internal/dns"
	"remna-ui/internal/generator"
	"remna-ui/internal/types"
)

func parseSSHAuth(password, keyPath string) []ssh.AuthMethod {
	var authMethods []ssh.AuthMethod

	trimmedKey := strings.Trim(strings.TrimSpace(keyPath), `"'`)
	if trimmedKey != "" {
		var keyBytes []byte
		var err error
		if strings.Contains(trimmedKey, "PRIVATE KEY") {
			keyBytes = []byte(trimmedKey)
		} else {
			keyBytes, err = os.ReadFile(trimmedKey)
		}
		if err == nil && len(keyBytes) > 0 {
			signer, err := ssh.ParsePrivateKey(keyBytes)
			if err == nil {
				authMethods = append(authMethods, ssh.PublicKeys(signer))
			} else if password != "" {
				signer, err = ssh.ParsePrivateKeyWithPassphrase(keyBytes, []byte(password))
				if err == nil {
					authMethods = append(authMethods, ssh.PublicKeys(signer))
				}
			}
		}
	}

	if password != "" {
		authMethods = append(authMethods, ssh.Password(password))
	}

	return authMethods
}

func getSSHConfig(user, password, keyPath string) (*ssh.ClientConfig, error) {
	authMethods := parseSSHAuth(password, keyPath)

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("не указан пароль или SSH-ключ")
	}

	return &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}, nil
}

func TestConnection(host string, port int, user, password, keyPath string) types.SSHTestResult {
	if port == 0 {
		port = 22
	}
	if user == "" {
		user = "root"
	}

	cfg, err := getSSHConfig(user, password, keyPath)
	if err != nil {
		return types.SSHTestResult{
			Success: false,
			Error:   err.Error(),
		}
	}

	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	client, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		errStr := err.Error()
		hint := "Проверьте адрес сервера и реквизиты доступа."
		if strings.Contains(errStr, "unable to authenticate") {
			hint = "Неверный логин или пароль. Либо на сервере выключен вход по паролю (PasswordAuthentication yes в sshd_config). Если для сервера настроен SSH-ключ (.pem / id_rsa / id_ed25519), укажите его в поле 'Приватный SSH-ключ' через 'Выбрать файл...'."
		} else if strings.Contains(errStr, "connection refused") {
			hint = fmt.Sprintf("Порт %d закрыт. Убедитесь, что SSH-сервер запущен на этом порту.", port)
		} else if strings.Contains(errStr, "i/o timeout") {
			hint = fmt.Sprintf("Время ожидания истекло при подключении к %s. Проверьте фаервол или сетевые правила хостинга.", addr)
		}
		return types.SSHTestResult{
			Success: false,
			Error:   fmt.Sprintf("%s\n\nПодсказка: %s", errStr, hint),
		}
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return types.SSHTestResult{
			Success: true,
			OS:      "Linux",
			CPU:     "Неизвестно",
			RAM:     "Неизвестно",
		}
	}
	defer session.Close()

	cmd := `grep -E '^PRETTY_NAME=' /etc/os-release 2>/dev/null | cut -d= -f2 | tr -d '"'; nproc 2>/dev/null; free -h 2>/dev/null | awk '/^Mem:/{print $2}'`
	out, err := session.Output(cmd)
	if err != nil {
		return types.SSHTestResult{
			Success: true,
			OS:      "Linux VPS",
		}
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	osName := "Linux"
	cpu := ""
	ram := ""
	if len(lines) > 0 && lines[0] != "" {
		osName = lines[0]
	}
	if len(lines) > 1 && lines[1] != "" {
		cpu = fmt.Sprintf("%s vCPU", lines[1])
	}
	if len(lines) > 2 && lines[2] != "" {
		ram = fmt.Sprintf("%s RAM", lines[2])
	}

	return types.SSHTestResult{
		Success: true,
		OS:      osName,
		CPU:     cpu,
		RAM:     ram,
	}
}

func buildDecoySiteScript(siteDir, template, cloneURL string) string {
	cleanClone := strings.TrimSpace(cloneURL)
	if cleanClone != "" && !strings.HasPrefix(cleanClone, "http://") && !strings.HasPrefix(cleanClone, "https://") {
		cleanClone = "https://" + cleanClone
	}

	return fmt.Sprintf(`
SITE_DIR="%s"
TEMPLATE="%s"
RAW_CLONE_URL="%s"

mkdir -p "$SITE_DIR"
which curl >/dev/null 2>&1 || (apt-get update -qq && apt-get install -y -qq curl wget unzip >/dev/null 2>&1 || true)
which wget >/dev/null 2>&1 || (apt-get update -qq && apt-get install -y -qq wget curl unzip >/dev/null 2>&1 || true)

UA="Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36"

if [ "$TEMPLATE" = "clone" ] || [ -n "$RAW_CLONE_URL" ]; then
  TARGET="$RAW_CLONE_URL"
  case "$TARGET" in
    http://*|https://*) ;;
    *) TARGET="https://${TARGET}" ;;
  esac

  echo "Клонирование сайта-маскировки с $TARGET в $SITE_DIR..."
  rm -rf "$SITE_DIR"/* 2>/dev/null || true

  # 1. Прямой захват главной страницы через curl с современными заголовками Chrome и авто-распаковкой
  curl -sSL -k \
    -A "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36" \
    -H "Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8" \
    -H "Accept-Language: ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7" \
    -H "Sec-Ch-Ua: \"Google Chrome\";v=\"131\", \"Chromium\";v=\"131\", \"Not_A Brand\";v=\"24\"" \
    -H "Sec-Ch-Ua-Mobile: ?0" \
    -H "Sec-Ch-Ua-Platform: \"Windows\"" \
    -H "Sec-Fetch-Dest: document" \
    -H "Sec-Fetch-Mode: navigate" \
    -H "Sec-Fetch-Site: none" \
    -H "Sec-Fetch-User: ?1" \
    -H "Upgrade-Insecure-Requests: 1" \
    --compressed \
    --connect-timeout 15 \
    --max-time 30 \
    -L "$TARGET" -o "$SITE_DIR/index.html" 2>/dev/null || true

  # 2. Если curl с https не вернул HTML или размер мал, пробуем HTTP
  HTML_SIZE=$(wc -c < "$SITE_DIR/index.html" 2>/dev/null || echo 0)
  if [ ! -s "$SITE_DIR/index.html" ] || [ "$HTML_SIZE" -lt 150 ]; then
    HTTP_TARGET=$(echo "$TARGET" | sed 's|^https://|http://|')
    curl -sSL -k \
      -A "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36" \
      -H "Accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8" \
      -H "Accept-Language: ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7" \
      --compressed \
      --connect-timeout 15 \
      --max-time 30 \
      -L "$HTTP_TARGET" -o "$SITE_DIR/index.html" 2>/dev/null || true
  fi

  # 3. Если всё ещё нет index.html, попытка зеркалирования через wget
  HTML_SIZE=$(wc -c < "$SITE_DIR/index.html" 2>/dev/null || echo 0)
  if [ ! -s "$SITE_DIR/index.html" ] || [ "$HTML_SIZE" -lt 150 ]; then
    wget --page-requisites --convert-links --adjust-extension --no-host-directories \
      --no-check-certificate \
      --directory-prefix="$SITE_DIR" -e robots=off \
      --max-redirect=5 \
      -U "$UA" \
      --header="Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8" \
      --header="Accept-Language: ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7" \
      --connect-timeout=15 --timeout=30 --tries=2 "$TARGET" 2>/dev/null || true

    main_html=$(find "$SITE_DIR" -type f -name "*.html" 2>/dev/null | sort | head -n 1)
    if [ -n "$main_html" ] && [ "$main_html" != "$SITE_DIR/index.html" ]; then
      cp -f "$main_html" "$SITE_DIR/index.html"
    fi
  fi

  # 4. Проверяем валидность полученного HTML и внедряем <base href>
  HTML_SIZE=$(wc -c < "$SITE_DIR/index.html" 2>/dev/null || echo 0)
  if [ -s "$SITE_DIR/index.html" ] && [ "$HTML_SIZE" -ge 100 ]; then
    ORIGIN=$(echo "$TARGET" | grep -o -E 'https?://[^/]+')
    if [ -n "$ORIGIN" ]; then
      if ! grep -qi '<base ' "$SITE_DIR/index.html"; then
        if grep -qi '<head' "$SITE_DIR/index.html"; then
          sed -i -E "s|(<head[^>]*>)|\1<base href=\"${ORIGIN}/\">|i" "$SITE_DIR/index.html" 2>/dev/null || true
        else
          sed -i "1s|^|<base href=\"${ORIGIN}/\">\n|" "$SITE_DIR/index.html" 2>/dev/null || true
        fi
      fi
    fi
    echo "Сайт-маскировка успешно склонирован с $TARGET (размер: ${HTML_SIZE} байт)!"
  else
    echo "Предупреждение: не удалось получить данные с $TARGET. Устанавливается локальный шаблон."
    cat <<'FALLBACK_EOF' > "$SITE_DIR/index.html"
%s
FALLBACK_EOF
  fi

elif [ "$TEMPLATE" != "nothing" ]; then
  echo "Загрузка шаблона сайта-маскировки из каталога eGames..."
  rm -rf /tmp/decoy_tpl && mkdir -p /tmp/decoy_tpl
  DL_OK=0
  for TPL_URL in     "https://ghfast.top/https://github.com/eGamesAPI/simple-web-templates/archive/6b7690b37af85c35117b6da65a36b1b3e0503477.zip"     "https://gh-proxy.com/https://github.com/eGamesAPI/simple-web-templates/archive/6b7690b37af85c35117b6da65a36b1b3e0503477.zip"     "https://github.com/eGamesAPI/simple-web-templates/archive/6b7690b37af85c35117b6da65a36b1b3e0503477.zip"; do
    if curl -fsSL --connect-timeout 10 --max-time 45 -o /tmp/decoy_tpl/tpl.zip "$TPL_URL" 2>/dev/null; then
      if [ -s /tmp/decoy_tpl/tpl.zip ] && [ "$(head -c 2 /tmp/decoy_tpl/tpl.zip)" = "PK" ]; then
        unzip -q -o /tmp/decoy_tpl/tpl.zip -d /tmp/decoy_tpl/out 2>/dev/null || true
        TPL_DIR=$(find /tmp/decoy_tpl/out -mindepth 2 -maxdepth 2 -type d 2>/dev/null | head -n 1)
        if [ -n "$TPL_DIR" ] && [ -d "$TPL_DIR" ]; then
          rm -rf "$SITE_DIR"/* 2>/dev/null || true
          cp -rf "$TPL_DIR"/* "$SITE_DIR/" 2>/dev/null || true
          DL_OK=1
          break
        fi
      fi
    fi
  done
  rm -rf /tmp/decoy_tpl

  if [ "$DL_OK" -eq 0 ] || [ ! -s "$SITE_DIR/index.html" ]; then
    cat <<'FALLBACK_EOF' > "$SITE_DIR/index.html"
%s
FALLBACK_EOF
  fi
fi

if [ "$SITE_DIR" != "/var/www/html" ] && [ -d /var/www/html ]; then
  cp -rf "$SITE_DIR"/* /var/www/html/ 2>/dev/null || true
fi
`, siteDir, template, cleanClone, generator.RenderCamouflageSite(), generator.RenderCamouflageSite())
}

func DeployRemotely(
	ctx context.Context,
	plan types.DeploymentPlan,
	emitLog func(status, msg string),
) (*types.RemoteDeployResult, error) {
	host := plan.SshHost
	port := plan.SshPort
	if port == 0 {
		port = 22
	}
	user := plan.SshUser
	if user == "" {
		user = "root"
	}

	emitLog("info", fmt.Sprintf("Подключение по SSH к %s@%s:%d ...", user, host, port))

	cfg, err := getSSHConfig(user, plan.SshPassword, plan.SshKey)
	if err != nil {
		emitLog("error", err.Error())
		return nil, err
	}

	client, err := ssh.Dial("tcp", net.JoinHostPort(host, fmt.Sprintf("%d", port)), cfg)
	if err != nil {
		emitLog("error", fmt.Sprintf("Ошибка SSH-подключения: %v", err))
		return nil, err
	}
	defer client.Close()

	emitLog("success", "SSH-соединение успешно установлено!")

	runRemoteCmd := func(title, command string) error {
		emitLog("step", title)
		session, err := client.NewSession()
		if err != nil {
			return err
		}
		defer session.Close()

		stdout, _ := session.StdoutPipe()
		stderr, _ := session.StderrPipe()

		if err := session.Start("bash -lc " + quoteCmd(command)); err != nil {
			emitLog("error", fmt.Sprintf("Не удалось запустить команду: %v", err))
			return err
		}

		go streamReader(stdout, func(line string) { emitLog("stdout", line) })
		go streamReader(stderr, func(line string) {
			status := classifyStderrLine(line)
			emitLog(status, line)
		})

		if err := session.Wait(); err != nil {
			emitLog("error", fmt.Sprintf("Команда завершилась с ошибкой: %v", err))
			return err
		}
		return nil
	}

	targetDir := "/opt/remnawave"
	if err := runRemoteCmd("Подготовка каталога", fmt.Sprintf("mkdir -p %s", targetDir)); err != nil {
		return nil, err
	}

	// Automatic Swap setup (vital for 1GB VPS to prevent Out-Of-Memory crashes)
	swapCmd := `if [ $(free -m | awk '/^Swap:/{print $2}') -lt 1024 ]; then ` +
		`fallocate -l 2G /swapfile 2>/dev/null || dd if=/dev/zero of=/swapfile bs=1M count=2048 2>/dev/null; ` +
		`chmod 600 /swapfile; ` +
		`mkswap /swapfile >/dev/null 2>&1 || true; ` +
		`swapon /swapfile >/dev/null 2>&1 || true; ` +
		`grep -q "/swapfile" /etc/fstab || echo "/swapfile none swap sw 0 0" >> /etc/fstab; ` +
		`sysctl vm.swappiness=15 >/dev/null 2>&1 || true; ` +
		`echo "Файл подкачки 2GB успешно подключен!"; ` +
		`else echo "Swap уже настроен: $(free -h | awk '/^Swap:/{print $2}')"; fi`
	_ = runRemoteCmd("Оптимизация памяти (Swap 2GB)", swapCmd)

	if plan.PerformanceOptimization {
		sysctlRamCmd := `cat > /etc/sysctl.d/99-remnawave-ram.conf <<'EOF'
vm.swappiness=10
vm.vfs_cache_pressure=50
vm.dirty_ratio=15
vm.dirty_background_ratio=5
EOF
sysctl --system >/dev/null 2>&1 || sysctl -p /etc/sysctl.d/99-remnawave-ram.conf >/dev/null 2>&1 || true`
		_ = runRemoteCmd("Оптимизация ядра Linux (RAM sysctl)", sysctlRamCmd)
	}

	// Install Docker if missing
	dockerCheck := `if ! command -v docker >/dev/null 2>&1; then ` +
		`echo "Проверка готовности пакетного менеджера dpkg/apt..."; ` +
		`for i in $(seq 1 30); do ` +
		`  if ! fuser /var/lib/dpkg/lock-frontend >/dev/null 2>&1 && ! fuser /var/lib/dpkg/lock >/dev/null 2>&1 && ! fuser /var/lib/apt/lists/lock >/dev/null 2>&1; then ` +
		`    break; ` +
		`  fi; ` +
		`  echo "Ожидание освобождения dpkg/apt (фоновое обновление системы)... $i/30"; ` +
		`  sleep 2; ` +
		`done; ` +
		`systemctl stop unattended-upgrades >/dev/null 2>&1 || true; ` +
		`killall -w -q apt apt-get dpkg unattended-upgr 2>/dev/null || true; ` +
		`dpkg --configure -a >/dev/null 2>&1 || true; ` +
		`for attempt in 1 2 3; do ` +
		`  if command -v docker >/dev/null 2>&1; then break; fi; ` +
		`  echo "Установка Docker (попытка $attempt из 3)..."; ` +
		`  curl -fsSL https://get.docker.com | sh && systemctl enable --now docker || sleep 3; ` +
		`done; ` +
		`command -v docker >/dev/null 2>&1; ` +
		`fi`
	if err := runRemoteCmd("Проверка Docker", dockerCheck); err != nil {
		return nil, err
	}

	// Configure Docker Registry Mirrors & Log Driver
	dockerDaemonCmd := `mkdir -p /etc/docker; ` +
		`if [ ! -f /etc/docker/daemon.json ]; then ` +
		`cat > /etc/docker/daemon.json <<'EOF'
{
  "log-driver": "local",
  "log-opts": {
    "max-size": "50m",
    "max-file": "3"
  },
  "registry-mirrors": [
    "https://mirror.gcr.io",
    "https://dockerhub.timeweb.cloud"
  ]
}
EOF
systemctl restart docker >/dev/null 2>&1 || true
fi`
	_ = runRemoteCmd("Настройка зеркал Docker Hub и ротации логов", dockerDaemonCmd)

	// Enable Google BBR TCP congestion control & network performance sysctl
	bbrCmd := `modprobe tcp_bbr 2>/dev/null || true; ` +
		`echo "tcp_bbr" >> /etc/modules-load.d/bbr.conf 2>/dev/null || true; ` +
		`cat > /etc/sysctl.d/99-remna-bbr.conf <<'EOF'
net.core.default_qdisc = fq
net.ipv4.tcp_congestion_control = bbr
net.ipv4.tcp_fastopen = 3
fs.file-max = 1048576
EOF
sysctl --system >/dev/null 2>&1 || sysctl -p /etc/sysctl.d/99-remna-bbr.conf >/dev/null 2>&1 || true`
	_ = runRemoteCmd("Активация сетевого ускорения (Google BBR + fq)", bbrCmd)

	// Configure UFW if enabled
	if plan.UfwSetup {
		sshPortToAllow := plan.SshPort
		if sshPortToAllow <= 0 {
			sshPortToAllow = 22
		}
		ufwCmd := fmt.Sprintf("ufw allow %d/tcp; ufw allow 80/tcp; ufw allow 443/tcp; ", sshPortToAllow)
		if sshPortToAllow != 22 {
			ufwCmd += "ufw allow 22/tcp; "
		}
		if plan.NodePort > 0 {
			ufwCmd += fmt.Sprintf("ufw allow %d/tcp; ", plan.NodePort)
		}
		ufwCmd += "ufw --force enable || true"
		_ = runRemoteCmd("Настройка UFW фаервола", ufwCmd)
	}

	// Generate and transfer stack files
	emitLog("info", "Формирование и запись конфигураций на сервер...")

	hasPanel := contains(plan.Components, "panel")
	hasNode := contains(plan.Components, "node")
	hasProxy := contains(plan.Components, "proxy")
	hasSub := contains(plan.Components, "subscription")

	// eGames Single Port 443 Stealth Fallback Architecture
	if hasPanel && hasNode && (plan.PortMode == "" || plan.PortMode == "stealth443") {
		return deployStealthAllInOne(ctx, client, plan, targetDir, runRemoteCmd, emitLog)
	}

	if hasPanel {
		var existingDbPass, existingAppSecret, existingMetricsPass, existingWebhookSecret string
		// Try to read existing credentials from /opt/remnawave/panel/.env to avoid resetting passwords on redeploy
		session, err := client.NewSession()
		if err == nil {
			out, err := session.Output(fmt.Sprintf("cat %s/panel/.env 2>/dev/null || true", targetDir))
			session.Close()
			if err == nil && len(out) > 0 {
				lines := strings.Split(string(out), "\n")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "POSTGRES_PASSWORD=") {
						existingDbPass = strings.Trim(strings.TrimPrefix(line, "POSTGRES_PASSWORD="), "\"' \r\n")
					} else if strings.HasPrefix(line, "APP_SECRET=") {
						existingAppSecret = strings.Trim(strings.TrimPrefix(line, "APP_SECRET="), "\"' \r\n")
					} else if strings.HasPrefix(line, "METRICS_PASS=") {
						existingMetricsPass = strings.Trim(strings.TrimPrefix(line, "METRICS_PASS="), "\"' \r\n")
					} else if strings.HasPrefix(line, "WEBHOOK_SECRET_HEADER=") {
						existingWebhookSecret = strings.Trim(strings.TrimPrefix(line, "WEBHOOK_SECRET_HEADER="), "\"' \r\n")
					}
				}
			}
		}

		var env string
		var compose string
		if plan.PerformanceOptimization {
			env = generator.RenderPanelEnvWithCreds(
				plan.PanelDomain,
				plan.SubscriptionDomain,
				hasSub,
				existingAppSecret,
				existingDbPass,
				existingMetricsPass,
				existingWebhookSecret,
			)
			compose = generator.RenderPanelCompose()
		} else {
			env = generator.RenderPanelEnvStandardWithCreds(
				plan.PanelDomain,
				plan.SubscriptionDomain,
				hasSub,
				existingAppSecret,
				existingDbPass,
				existingMetricsPass,
				existingWebhookSecret,
			)
			compose = generator.RenderPanelComposeStandard()
		}
		_ = runRemoteCmd("Запись panel/.env", fmt.Sprintf("mkdir -p %s/panel && cat > %s/panel/.env <<'EOF'\n%s\nEOF", targetDir, targetDir, env))
		_ = runRemoteCmd("Запись panel/docker-compose.yml", fmt.Sprintf("cat > %s/panel/docker-compose.yml <<'EOF'\n%s\nEOF", targetDir, compose))
	}

	if hasSub {
		env := generator.RenderSubscriptionEnv(plan.PanelDomain, "", plan.SubscriptionMode)
		_ = runRemoteCmd("Запись subscription/.env", fmt.Sprintf("mkdir -p %s/subscription && cat > %s/subscription/.env <<'EOF'\n%s\nEOF", targetDir, targetDir, env))
		compose := generator.RenderSubscriptionCompose(plan.SubscriptionMode)
		_ = runRemoteCmd("Запись subscription/docker-compose.yml", fmt.Sprintf("cat > %s/subscription/docker-compose.yml <<'EOF'\n%s\nEOF", targetDir, compose))
	}

	if hasProxy {
		panelDomain := dns.NormalizeDomain(plan.PanelDomain)
		if panelDomain == "" {
			panelDomain = dns.NormalizeDomain(plan.Domain)
		}
		plan.PanelDomain = panelDomain
		bundledSub := hasSub && plan.SubscriptionMode == "bundled"

		if plan.Proxy == "nginx" {
			_ = runRemoteCmd("Подготовка каталога Nginx и SSL", fmt.Sprintf("mkdir -p %s/nginx/site && rm -rf %s/nginx/fullchain.pem %s/nginx/privkey.key 2>/dev/null || true", targetDir, targetDir, targetDir))

			// Determine domains for multi-domain SSL certificate (Panel + Subscription)
			certbotDomains := fmt.Sprintf("-d %s", panelDomain)
			sslSanExt := fmt.Sprintf("DNS:%s", panelDomain)
			if bundledSub && plan.SubscriptionDomain != "" && plan.SubscriptionDomain != panelDomain {
				certbotDomains = fmt.Sprintf("-d %s -d %s", panelDomain, plan.SubscriptionDomain)
				sslSanExt = fmt.Sprintf("DNS:%s,DNS:%s", panelDomain, plan.SubscriptionDomain)
			}

			// Initial valid SSL cert to prevent PR_END_OF_FILE_ERROR
			sslInitCmd := fmt.Sprintf("openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout %s/nginx/privkey.key -out %s/nginx/fullchain.pem -subj '/CN=%s' -addext 'subjectAltName=%s' 2>/dev/null || openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout %s/nginx/privkey.key -out %s/nginx/fullchain.pem -subj '/CN=%s' 2>/dev/null || true", targetDir, targetDir, panelDomain, sslSanExt, targetDir, targetDir, panelDomain)
			_ = runRemoteCmd("Генерация первичного SSL сертификата", sslInitCmd)

			conf := generator.RenderNginxConf(panelDomain, plan.SubscriptionDomain, bundledSub, plan.SecretGate, plan.UseCamouflage)
			_ = runRemoteCmd("Запись nginx/nginx.conf", fmt.Sprintf("cat > %s/nginx/nginx.conf <<'EOF'\n%s\nEOF", targetDir, conf))
			_ = runRemoteCmd("Запись nginx/docker-compose.yml", fmt.Sprintf("cat > %s/nginx/docker-compose.yml <<'EOF'\n%s\nEOF", targetDir, generator.RenderNginxCompose()))
			if plan.UseCamouflage {
				siteDir := fmt.Sprintf("%s/nginx/site", targetDir)
				_ = runRemoteCmd("Подготовка и клонирование сайта маскировки (Nginx)", buildDecoySiteScript(siteDir, plan.DecoyTemplate, plan.DecoyCloneURL))
			}

			if plan.UseCloudflareDns && strings.TrimSpace(plan.CloudflareApiToken) != "" {
				cfToken := strings.TrimSpace(plan.CloudflareApiToken)
				rootDomain := panelDomain
				parts := strings.Split(panelDomain, ".")
				if len(parts) >= 2 {
					rootDomain = strings.Join(parts[len(parts)-2:], ".")
				}
				cfDnsCmd := fmt.Sprintf(`which curl >/dev/null 2>&1 || (apt-get update -qq && apt-get install -y -qq curl >/dev/null 2>&1); `+
					`if [ ! -x ~/.acme.sh/acme.sh ]; then `+
					`curl -fsSL https://get.acme.sh | sh -s email=admin@%s >/dev/null 2>&1 || true; `+
					`fi; `+
					`export CF_Token="%s"; `+
					`~/.acme.sh/acme.sh --upgrade --auto-upgrade >/dev/null 2>&1 || true; `+
					`~/.acme.sh/acme.sh --set-default-ca --server letsencrypt >/dev/null 2>&1 || true; `+
					`~/.acme.sh/acme.sh --issue --dns dns_cf -d %s -d "*.%s" --force >/dev/null 2>&1; `+
					`if [ -f ~/.acme.sh/%s/fullchain.cer ]; then `+
					`~/.acme.sh/acme.sh --install-cert -d %s --key-file %s/nginx/privkey.key --fullchain-file %s/nginx/fullchain.pem >/dev/null 2>&1; `+
					`echo "Сертификат Let's Encrypt Wildcard успешно получен через Cloudflare DNS-01!"; `+
					`else `+
					`echo "Не удалось выпустить сертификат через Cloudflare DNS-01 (проверьте права токена), используется самоподписанный SSL."; fi`,
					rootDomain, cfToken, rootDomain, rootDomain, rootDomain, rootDomain, targetDir, targetDir)
				_ = runRemoteCmd("Получение Wildcard SSL через Cloudflare DNS-01", cfDnsCmd)
			} else {
				// Free port 80 before certbot runs
				_ = runRemoteCmd("Освобождение портов для SSL проверки", "docker stop remnawave-nginx 2>/dev/null || true")

				// Let's Encrypt certificate acquisition via certbot (Docker-first with system fallback)
				certbotCmd := fmt.Sprintf(`if [ -f /etc/letsencrypt/live/%s/fullchain.pem ]; then `+
					`cp -fL /etc/letsencrypt/live/%s/fullchain.pem %s/nginx/fullchain.pem && `+
					`cp -fL /etc/letsencrypt/live/%s/privkey.pem %s/nginx/privkey.key && `+
					`echo "Использован существующий сертификат Let's Encrypt!"; `+
					`else `+
					`docker stop remnawave-nginx 2>/dev/null || true; `+
					`CERT_SUCCESS=0; `+
					`if command -v docker >/dev/null 2>&1; then `+
					`echo "Запрос сертификата Let's Encrypt через официальный Docker Certbot..."; `+
					`docker run --rm -p 80:80 -v /etc/letsencrypt:/etc/letsencrypt -v /var/lib/letsencrypt:/var/lib/letsencrypt certbot/certbot certonly --standalone %s --agree-tos --register-unsafely-without-email --non-interactive 2>&1 && CERT_SUCCESS=1 || true; `+
					`fi; `+
					`if [ "$CERT_SUCCESS" -eq 0 ] && [ ! -f /etc/letsencrypt/live/%s/fullchain.pem ]; then `+
					`echo "Docker Certbot не завершился успехом, попытка через системный пакет..."; `+
					`which certbot >/dev/null 2>&1 || (apt-get update -qq && (apt-get install -y -qq certbot || (apt-get install -y -qq software-properties-common && add-apt-repository -y universe && apt-get update -qq && apt-get install -y -qq certbot) || snap install --classic certbot) >/dev/null 2>&1) || true; `+
					`if command -v certbot >/dev/null 2>&1; then `+
					`certbot certonly --standalone %s --agree-tos --register-unsafely-without-email --non-interactive 2>&1 && CERT_SUCCESS=1 || true; `+
					`fi; `+
					`fi; `+
					`if [ -f /etc/letsencrypt/live/%s/fullchain.pem ]; then `+
					`cp -fL /etc/letsencrypt/live/%s/fullchain.pem %s/nginx/fullchain.pem && `+
					`cp -fL /etc/letsencrypt/live/%s/privkey.pem %s/nginx/privkey.key && `+
					`echo "Сертификат Let's Encrypt успешно получен и применен к Nginx!"; `+
					`else `+
					`echo "Let's Encrypt не смог выпустить сертификат (проверьте A-записи домена в DNS и отключите Cloudflare Proxy/Orange Cloud на момент верификации), используется первичный самоподписанный SSL."; `+
					`fi; fi`,
					panelDomain, panelDomain, targetDir, panelDomain, targetDir, certbotDomains, panelDomain, certbotDomains, panelDomain, panelDomain, targetDir, panelDomain, targetDir)
				_ = runRemoteCmd("Получение доверенного SSL сертификата Let's Encrypt", certbotCmd)
			}
		} else {
			caddyfile := generator.RenderCaddyfile(panelDomain, plan.SubscriptionDomain, bundledSub, plan.SecretGate, plan.UseCamouflage)
			_ = runRemoteCmd("Подготовка каталога Caddy", fmt.Sprintf("mkdir -p %s/caddy/site", targetDir))
			if plan.UseCamouflage {
				siteDir := fmt.Sprintf("%s/caddy/site", targetDir)
				_ = runRemoteCmd("Подготовка и клонирование сайта маскировки (Caddy)", buildDecoySiteScript(siteDir, plan.DecoyTemplate, plan.DecoyCloneURL))
			}
			_ = runRemoteCmd("Запись caddy/Caddyfile", fmt.Sprintf("cat > %s/caddy/Caddyfile <<'EOF'\n%s\nEOF", targetDir, caddyfile))
			_ = runRemoteCmd("Запись caddy/docker-compose.yml", fmt.Sprintf("cat > %s/caddy/docker-compose.yml <<'EOF'\n%s\nEOF", targetDir, generator.RenderCaddyCompose()))
			emitLog("success", "Конфигурация Caddy подготовлена (сертификаты Let's Encrypt будут выпущены автоматически при первом запросе).")
		}
	}

	if hasNode {
		nodePort := plan.NodePort
		if nodePort == 0 {
			nodePort = 2222
		}
		_ = runRemoteCmd("Запись node/docker-compose.yml", fmt.Sprintf("mkdir -p %s/node && cat > %s/node/docker-compose.yml <<'EOF'\n%s\nEOF", targetDir, targetDir, generator.RenderNodeCompose(nodePort, plan.NodeSecret)))
	}

	// Start services
	if hasPanel {
		_ = runRemoteCmd("Остановка предыдущих контейнеров Panel (если были)", fmt.Sprintf("cd %s/panel && docker compose down 2>/dev/null || true", targetDir))

		// Pre-pull images with retry logic against Docker Hub 502/network timeouts
		pullCmd := fmt.Sprintf(`for attempt in 1 2 3; do `+
			`echo "Загрузка Docker образов (попытка $attempt из 3)..."; `+
			`if cd %s/panel && docker compose pull; then `+
			`echo "Все образы успешно загружены!"; break; fi; `+
			`echo "Временный сбой сети Docker Hub. Повтор через 4 секунды..."; `+
			`sleep 4; `+
			`done`, targetDir)
		_ = runRemoteCmd("Загрузка Docker образов (с защитой от сбоев сети)", pullCmd)

		if err := runRemoteCmd("Запуск Panel стека", fmt.Sprintf("cd %s/panel && docker compose up -d", targetDir)); err != nil {
			_ = runRemoteCmd("Диагностика ошибок remnawave-db", fmt.Sprintf("cd %s/panel && docker compose logs --tail=40 remnawave-db || true", targetDir))
			_ = runRemoteCmd("Диагностика ошибок remnawave", fmt.Sprintf("cd %s/panel && docker compose logs --tail=40 remnawave || true", targetDir))
			return nil, err
		}

		// Self-healing database check & password synchronization
		dbSyncScript := fmt.Sprintf(`
echo "Проверка готовности PostgreSQL..."
for i in $(seq 1 30); do
  if docker exec -u postgres remnawave-db pg_isready >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

DB_PASS=$(grep -E '^POSTGRES_PASSWORD=' %s/panel/.env | cut -d= -f2- | tr -d '"\r\n')
DB_PASS_ESC=$(printf '%%s' "$DB_PASS" | sed "s/'/''/g")

# Verify if authentication works with the password from .env
if docker exec remnawave-db psql "postgresql://postgres:${DB_PASS}@127.0.0.1:5432/postgres" -c '\q' >/dev/null 2>&1; then
  echo "Аутентификация в PostgreSQL подтверждена!"
else
  echo "Синхронизация пароля базы данных PostgreSQL с файлом .env..."
  docker exec -u postgres remnawave-db psql -U postgres -d postgres -c "ALTER USER postgres WITH PASSWORD '${DB_PASS_ESC}';" >/dev/null 2>&1 || true
  
  if docker exec remnawave-db psql "postgresql://postgres:${DB_PASS}@127.0.0.1:5432/postgres" -c '\q' >/dev/null 2>&1; then
    echo "Пароль PostgreSQL успешно синхронизирован!"
  else
    echo "Предупреждение: пароль базы данных отличается от .env. Данные томов сохранены."
  fi
fi

# Ensure remnawave container is restarted and migrations apply
echo "Применение миграций Remnawave backend..."
docker compose -f %s/panel/docker-compose.yml restart remnawave >/dev/null 2>&1 || true
sleep 4
docker compose -f %s/panel/docker-compose.yml logs --tail=15 remnawave || true
`, targetDir, targetDir, targetDir)
		_ = runRemoteCmd("Синхронизация и проверка PostgreSQL", dbSyncScript)
	}
	var classicAdminUser, classicAdminPass string
	adminMode := plan.AdminCreationMode
	if adminMode == "" {
		adminMode = "auto"
	}

	if hasPanel {
		if adminMode == "custom" {
			classicAdminUser = strings.TrimSpace(plan.AdminUsername)
			if classicAdminUser == "" {
				classicAdminUser = "admin"
			}
			classicAdminPass = strings.TrimSpace(plan.AdminPassword)
			if len(classicAdminPass) < 24 {
				classicAdminPass = generator.RandomAdminPassword(28)
			}
		} else if adminMode == "auto" {
			classicAdminUser = "admin"
			classicAdminPass = generator.RandomAdminPassword(28)
		} else { // "manual"
			classicAdminUser = ""
			classicAdminPass = ""
		}

		if adminMode != "manual" {
			adminSetupScript := fmt.Sprintf(`
which jq >/dev/null 2>&1 || (apt-get update -qq && apt-get install -y -qq jq >/dev/null 2>&1 || true)
API_BASE="http://127.0.0.1:3000"
TARGET_DIR="%s"
ADMIN_USER="%s"
ADMIN_PASS="%s"

# Check if credentials already exist from a previous install
if [ -f "${TARGET_DIR}/credentials.txt" ]; then
  SAVED_PASS=$(grep '^ADMIN_PASSWORD=' "${TARGET_DIR}/credentials.txt" | cut -d= -f2- | tr -d '\r\n[:space:]')
  if [ -n "$SAVED_PASS" ]; then
    ADMIN_PASS="$SAVED_PASS"
  fi
elif [ -f "${TARGET_DIR}/panel/.env" ]; then
  SAVED_PASS=$(grep '^ADMIN_PASSWORD=' "${TARGET_DIR}/panel/.env" | cut -d= -f2- | tr -d '\r\n[:space:]')
  if [ -n "$SAVED_PASS" ]; then
    ADMIN_PASS="$SAVED_PASS"
  fi
fi

# Multi-fallback JSON parser helper (jq -> python3 -> grep/sed)
extract_token() {
  local json_input="$1"
  local field="$2"
  if command -v jq >/dev/null 2>&1; then
    local res
    res=$(echo "$json_input" | jq -r ".response.${field} // empty" 2>/dev/null || true)
    if [ -n "$res" ] && [ "$res" != "null" ]; then
      echo "$res"
      return 0
    fi
  fi
  if command -v python3 >/dev/null 2>&1; then
    local res
    res=$(python3 -c "import sys, json; data=json.loads(sys.stdin.read()); print(data.get('response', {}).get('${field}') or '')" <<< "$json_input" 2>/dev/null || true)
    if [ -n "$res" ] && [ "$res" != "null" ]; then
      echo "$res"
      return 0
    fi
  fi
  echo "$json_input" | grep -o "\"${field}\":\"[^\"]*\"" | cut -d'"' -f4 || true
}

echo "Ожидание готовности API Remnawave (до 3 минут)..."
API_READY=0
for i in $(seq 1 60); do
  if curl -s -f --connect-timeout 4 --max-time 10 "${API_BASE}/api/auth/status" \
      -H "X-Forwarded-For: 127.0.0.1" \
      -H "X-Forwarded-Proto: https" >/dev/null 2>&1; then
    echo "Remnawave API успешно ответил (попытка $i)!"
    API_READY=1
    break
  fi
  sleep 3
done

IS_CONFIGURED=$(curl -s "${API_BASE}/api/auth/status" \
  -H "X-Forwarded-For: 127.0.0.1" \
  -H "X-Forwarded-Proto: https" 2>/dev/null | grep -oE '"isConfigured":true|"configured":true' || true)

TOKEN=""
if [ -z "$IS_CONFIGURED" ]; then
  echo "Регистрация администратора Remnawave в API..."
  cat > /tmp/rw_reg.json << JSON_REG_EOF
{"username":"${ADMIN_USER}","password":"${ADMIN_PASS}"}
JSON_REG_EOF
  REG_RES=$(curl -s -X POST "${API_BASE}/api/auth/register" \
    -H "Content-Type: application/json" \
    -H "X-Forwarded-For: 127.0.0.1" \
    -H "X-Forwarded-Proto: https" \
    -H "X-Remnawave-Client-Type: browser" \
    --data-binary @/tmp/rw_reg.json)
  rm -f /tmp/rw_reg.json
  TOKEN=$(extract_token "$REG_RES" "accessToken")
fi

if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
  echo "Попытка авторизации администратора Remnawave..."
  cat > /tmp/rw_login.json << JSON_LOGIN_EOF
{"username":"${ADMIN_USER}","password":"${ADMIN_PASS}"}
JSON_LOGIN_EOF
  LOGIN_RES=$(curl -s -X POST "${API_BASE}/api/auth/login" \
    -H "Content-Type: application/json" \
    -H "X-Forwarded-For: 127.0.0.1" \
    -H "X-Forwarded-Proto: https" \
    -H "X-Remnawave-Client-Type: browser" \
    --data-binary @/tmp/rw_login.json)
  rm -f /tmp/rw_login.json
  TOKEN=$(extract_token "$LOGIN_RES" "accessToken")
fi

if echo "$TOKEN" | grep -q '^ey'; then
  echo "Суперадминистратор успешно подтвержден в API Remnawave!"
  mkdir -p "${TARGET_DIR}" "${TARGET_DIR}/panel" 2>/dev/null || true
  echo "$TOKEN" > "${TARGET_DIR}/token" 2>/dev/null || true
  echo "$TOKEN" > "${TARGET_DIR}/panel/token" 2>/dev/null || true
  chmod 600 "${TARGET_DIR}/token" "${TARGET_DIR}/panel/token" 2>/dev/null || true
  echo "ADMIN_USERNAME=${ADMIN_USER}" >> "${TARGET_DIR}/panel/.env" 2>/dev/null || true
  echo "ADMIN_PASSWORD=${ADMIN_PASS}" >> "${TARGET_DIR}/panel/.env" 2>/dev/null || true
  cat > "${TARGET_DIR}/credentials.txt" <<CRED_EOF
ADMIN_USERNAME=${ADMIN_USER}
ADMIN_PASSWORD=${ADMIN_PASS}
CRED_EOF
  chmod 600 "${TARGET_DIR}/credentials.txt" 2>/dev/null || true
else
  echo "ВНИМАНИЕ: Автоматическая регистрация администратора не завершилась через API."
  echo "Завершите регистрацию учетной записи суперадминистратора вручную через веб-интерфейс панели."
  rm -f "${TARGET_DIR}/token" "${TARGET_DIR}/panel/token"
fi
ADMIN_USERNAME=${ADMIN_USER}
ADMIN_PASSWORD=${ADMIN_PASS}
CRED_EOF
chmod 600 "${TARGET_DIR}/credentials.txt" 2>/dev/null || true

if [ -n "$TOKEN" ] && [ "$TOKEN" != "null" ]; then
  MASTER_PAYLOAD='{"name":"remna-ui-manager","expiresInDays":3650,"scopes":["*"]}'
  MASTER_RES=$(curl -s --connect-timeout 10 --max-time 30 -X POST "${API_BASE}/api/tokens" \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json" \
    -H "X-Forwarded-For: 127.0.0.1" \
    -H "X-Forwarded-Proto: https" \
    -H "X-Remnawave-Client-Type: browser" \
    -d "${MASTER_PAYLOAD}")
  MASTER_TOKEN=$(extract_token "$MASTER_RES" "token")
  if [ -n "$MASTER_TOKEN" ] && [ "$MASTER_TOKEN" != "null" ]; then
    echo "$MASTER_TOKEN" > "${TARGET_DIR}/token"
    chmod 600 "${TARGET_DIR}/token" 2>/dev/null || true
  fi

  if [ "%t" = "true" ]; then
    TOKEN_PAYLOAD='{"name":"subscription-page","expiresInDays":3650,"scopes":["subscription-page-configs:list","subscription-page-configs:get","subscriptions:subpage-config","system:metadata","users:by-username"]}'
    TOKEN_RES=$(curl -s --connect-timeout 10 --max-time 30 -X POST "${API_BASE}/api/tokens" \
      -H "Authorization: Bearer ${TOKEN}" \
      -H "Content-Type: application/json" \
      -H "X-Forwarded-For: 127.0.0.1" \
      -H "X-Forwarded-Proto: https" \
      -H "X-Remnawave-Client-Type: browser" \
      -d "${TOKEN_PAYLOAD}")
    SUB_API_TOKEN=$(extract_token "$TOKEN_RES" "token")
    if [ -n "$SUB_API_TOKEN" ] && [ "$SUB_API_TOKEN" != "null" ]; then
      echo "API-токен для страницы подписок успешно сгенерирован!"
      sed -i "s|REMNAWAVE_API_TOKEN=.*|REMNAWAVE_API_TOKEN=${SUB_API_TOKEN}|g" "${TARGET_DIR}/subscription/.env" 2>/dev/null || true
      sed -i "s|REMNAWAVE_API_TOKEN=.*|REMNAWAVE_API_TOKEN=${SUB_API_TOKEN}|g" "${TARGET_DIR}/subscription/docker-compose.yml" 2>/dev/null || true
    else
      echo "Внимание: не удалось выпустить токен подписки (TOKEN_RES: ${TOKEN_RES})"
    fi
  fi
fi
`, targetDir, classicAdminUser, classicAdminPass, hasSub)
			_ = runRemoteCmd("Регистрация администратора и выпуск токенов API", adminSetupScript)
		} else {
			emitLog("info", "Режим регистрации: самостоятельный в веб-интерфейсе. Создайте первого пользователя при первом входе.")
		}
	}

	if hasSub {
		_ = runRemoteCmd("Запуск страницы подписок", fmt.Sprintf("cd %s/subscription && docker compose up -d", targetDir))
	}
	if hasProxy {
		proxyDir := "nginx"
		if plan.Proxy == "caddy" {
			proxyDir = "caddy"
		}
		_ = runRemoteCmd("Остановка предыдущего прокси (если был)", fmt.Sprintf("cd %s/%s && docker compose down 2>/dev/null || true", targetDir, proxyDir))
		if err := runRemoteCmd("Запуск защищённого входа HTTPS", fmt.Sprintf("cd %s/%s && docker compose up -d", targetDir, proxyDir)); err != nil {
			_ = runRemoteCmd("Диагностика прокси", fmt.Sprintf("cd %s/%s && docker compose logs --tail=30 || true", targetDir, proxyDir))
			return nil, err
		}
	}
	if hasNode {
		if strings.TrimSpace(plan.NodeSecret) != "" && plan.NodeSecret != "REPLACE_WITH_SECRET_KEY_FROM_PANEL" {
			nodePort := plan.NodePort
			if nodePort <= 0 {
				nodePort = 2222
			}
			var openPortsCmd string
			if plan.HideNodePort {
				// Port 2222 will be restricted strictly in DOCKER-USER / REMNA_INPUT
				openPortsCmd = "command -v ufw >/dev/null 2>&1 && ufw allow 443/tcp && ufw allow 80/tcp || true; iptables -I INPUT -p tcp --dport 443 -j ACCEPT 2>/dev/null || true; iptables -I INPUT -p tcp --dport 80 -j ACCEPT 2>/dev/null || true"
			} else {
				openPortsCmd = fmt.Sprintf("command -v ufw >/dev/null 2>&1 && ufw allow %d/tcp && ufw allow 443/tcp && ufw allow 80/tcp || true; iptables -I INPUT -p tcp --dport %d -j ACCEPT 2>/dev/null || true; iptables -I INPUT -p tcp --dport 443 -j ACCEPT 2>/dev/null || true; iptables -I INPUT -p tcp --dport 80 -j ACCEPT 2>/dev/null || true", nodePort, nodePort)
			}
			_ = runRemoteCmd(fmt.Sprintf("Открытие клиентских портов (443, 80%s)", func() string {
				if !plan.HideNodePort {
					return fmt.Sprintf(", %d", nodePort)
				}
				return ""
			}()), openPortsCmd)
			_ = runRemoteCmd("Остановка предыдущей ноды (если была)", fmt.Sprintf("cd %s/node && docker compose down 2>/dev/null || true", targetDir))
			_ = runRemoteCmd("Загрузка образов и запуск сервера ноды", fmt.Sprintf("cd %s/node && docker compose pull && docker compose up -d", targetDir))
		} else {
			emitLog("info", "Конфигурация ноды подготовлена (/opt/remnawave/node/docker-compose.yml).")
			emitLog("info", "Для запуска ноды войдите в панель управления, перейдите в Nodes -> Management, скопируйте SECRET_KEY и привяжите ноду.")
		}

		// Apply Anti-Probing Firewall protection if enabled
		if plan.HideNodePort || plan.StealthDrop {
			emitLog("step", "Применение правил защиты от ТСПУ/РКН (DOCKER-USER & Stealth DROP)...")

			// Resolve panel IP if hostname/URL was supplied
			panelIPToAllow := strings.TrimSpace(plan.PanelIP)
			if panelIPToAllow != "" {
				panelHost := panelIPToAllow
				if strings.Contains(panelHost, "://") {
					parts := strings.Split(panelHost, "://")
					if len(parts) > 1 {
						panelHost = parts[1]
					}
				}
				if strings.Contains(panelHost, "/") {
					panelHost = strings.Split(panelHost, "/")[0]
				}
				if strings.Contains(panelHost, ":") {
					panelHost = strings.Split(panelHost, ":")[0]
				}
				panelHost = strings.TrimSpace(panelHost)

				if net.ParseIP(panelHost) == nil {
					// Hostname, resolve to IPv4
					ips, err := net.LookupIP(panelHost)
					if err == nil {
						var v4s []string
						for _, ip := range ips {
							if ip.To4() != nil {
								v4s = append(v4s, ip.String())
							}
						}
						if len(v4s) > 0 {
							panelIPToAllow = strings.Join(v4s, ",")
							emitLog("info", fmt.Sprintf("Хост мастер-панели '%s' успешно разрешен в IP: %s", panelHost, panelIPToAllow))
						}
					} else {
						emitLog("warn", fmt.Sprintf("Не удалось разрешить IP для хоста панели '%s': %v", panelHost, err))
					}
				}
			}

			fwScript := admin.BuildAntiProbingScript(
				panelIPToAllow,
				[]int{80, 443},
				plan.HideNodePort,
				plan.StealthDrop,
				true,
				plan.NodePort,
				plan.SshPort,
			)
			_ = runRemoteCmd("Настройка Anti-Probing файрвола (DOCKER-USER/INPUT)", fwScript)
			emitLog("success", "Защита ноды от сканеров ТСПУ активирована в DOCKER-USER!")
		}

		// Deploy SelfSteal Decoy site on port 8444 if enabled
		if plan.DeployDecoy {
			cleanDecoyDomain := dns.NormalizeDomain(plan.DecoyDomain)
			if cleanDecoyDomain != "" {
				emitLog("step", fmt.Sprintf("Настройка SelfSteal Decoy-сайта для домена %s (Certbot Let's Encrypt + Nginx :8444)...", cleanDecoyDomain))
			} else {
				emitLog("step", "Развёртывание SelfSteal Decoy-сайта на локальном порту 8444...")
			}

			decoyContentScript := buildDecoySiteScript("$DECOY_ROOT", plan.DecoyTemplate, plan.DecoyCloneURL)

			decoyTemplateRaw := `
DOMAIN="%s"
FALLBACK_PORT=8444
DECOY_ROOT="/var/www/decoy"

mkdir -p "$DECOY_ROOT" /etc/nginx/ssl /etc/nginx/sites-available /etc/nginx/sites-enabled

# 1. Check and install dependencies
if ! command -v nginx >/dev/null 2>&1 || ! command -v certbot >/dev/null 2>&1 || ! command -v openssl >/dev/null 2>&1 || ! command -v curl >/dev/null 2>&1; then
  for i in $(seq 1 20); do
    if ! fuser /var/lib/dpkg/lock-frontend >/dev/null 2>&1 && ! fuser /var/lib/dpkg/lock >/dev/null 2>&1; then break; fi
    sleep 2
  done
  apt-get update -y && apt-get install -y nginx certbot openssl curl wget unzip || true
fi

# 2. Setup decoy website content
__DECOY_CONTENT_SCRIPT__

USE_LE=0
if [ -n "$DOMAIN" ] && [[ "$DOMAIN" == *"."* ]] && [[ ! "$DOMAIN" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  CERT_PATH="/etc/letsencrypt/live/$DOMAIN/fullchain.pem"
  KEY_PATH="/etc/letsencrypt/live/$DOMAIN/privkey.pem"

  if [ -f "$CERT_PATH" ] && [ -f "$KEY_PATH" ]; then
    echo "Сертификат Let's Encrypt для $DOMAIN уже существует."
    USE_LE=1
  else
    echo "Запрос валидного SSL-сертификата Let's Encrypt через Certbot..."
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
      echo "SSL-сертификат Let's Encrypt успешно получен!"
      USE_LE=1
    else
      echo "Предупреждение: Certbot не смог получить сертификат для $DOMAIN. Создается временный сертификат OpenSSL."
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

cat << NGINX_CONF > /etc/nginx/sites-available/decoy
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
    listen 127.0.0.1:$FALLBACK_PORT ssl;
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
NGINX_CONF

rm -f /etc/nginx/sites-enabled/default
ln -sf /etc/nginx/sites-available/decoy /etc/nginx/sites-enabled/decoy
nginx -t >/dev/null 2>&1 && (systemctl reload nginx || systemctl restart nginx)
systemctl enable nginx >/dev/null 2>&1 || true

HTTP_CODE=$(curl -k -s -o /dev/null -w "%%{http_code}" https://127.0.0.1:$FALLBACK_PORT || echo "000")
echo "Decoy HTTP Status: $HTTP_CODE"
`
			decoyScript := fmt.Sprintf(decoyTemplateRaw, cleanDecoyDomain, plan.SshHost)
			decoyScript = strings.Replace(decoyScript, "__DECOY_CONTENT_SCRIPT__", decoyContentScript, 1)
			_ = runRemoteCmd("Установка Nginx SelfSteal Decoy (:8444)", decoyScript)
			if cleanDecoyDomain != "" {
				emitLog("success", fmt.Sprintf("SelfSteal Decoy-сайт запущен на 127.0.0.1:8444 для домена %s!", cleanDecoyDomain))
			} else {
				emitLog("success", "SelfSteal Decoy-сайт запущен на 127.0.0.1:8444!")
			}
		}
	}

	emitLog("success", "==================================================")
	emitLog("success", "ВСЕ КОМПОНЕНТЫ УСПЕШНО РАЗВЁРНУТЫ И ЗАПУЩЕНЫ!")
	panelURL := ""
	cleanDomain := dns.NormalizeDomain(plan.PanelDomain)
	if cleanDomain == "" {
		cleanDomain = dns.NormalizeDomain(plan.Domain)
	}
	if hasPanel {
		panelURL = fmt.Sprintf("https://%s", cleanDomain)
		if plan.SecretGate != "" {
			cleanGate := strings.TrimLeft(strings.TrimSpace(plan.SecretGate), "/")
			panelURL = fmt.Sprintf("https://%s/%s", cleanDomain, cleanGate)
		}
		emitLog("success", fmt.Sprintf("Панель управления: %s", panelURL))

		// Check if admin token was minted successfully
		tokenCheckCmd := fmt.Sprintf(`if [ -f "%s/token" ] && grep -q '^ey' "%s/token" 2>/dev/null; then echo "VALID_TOKEN"; elif [ -f "%s/panel/token" ] && grep -q '^ey' "%s/panel/token" 2>/dev/null; then echo "VALID_TOKEN"; else echo "NO_TOKEN"; fi`, targetDir, targetDir, targetDir, targetDir)
		tokenSess, tokenErr := client.NewSession()
		var tokenOut []byte
		if tokenErr == nil {
			tokenOut, _ = tokenSess.Output(tokenCheckCmd)
			tokenSess.Close()
		}
		isTokenValid := strings.Contains(string(tokenOut), "VALID_TOKEN")

		if isTokenValid && adminMode != "manual" {
			emitLog("success", fmt.Sprintf("Логин администратора: %s", classicAdminUser))
			emitLog("success", fmt.Sprintf("Пароль администратора: %s", classicAdminPass))
		} else {
			adminMode = "manual"
			classicAdminUser = ""
			classicAdminPass = ""
			emitLog("info", "Регистрация: самостоятельная. Завершите регистрацию суперадминистратора в веб-интерфейсе панели.")
		}
	}
	subURL := ""
	if hasSub && plan.SubscriptionDomain != "" {
		subURL = fmt.Sprintf("https://%s", dns.NormalizeDomain(plan.SubscriptionDomain))
		emitLog("success", fmt.Sprintf("Страница подписок: %s", subURL))
	}
	emitLog("success", "==================================================")

	return &types.RemoteDeployResult{
		Success:           true,
		AdminCreationMode: adminMode,
		AdminUser:         classicAdminUser,
		AdminPass:         classicAdminPass,
		PanelURL:          panelURL,
		SubURL:            subURL,
	}, nil
}

func streamReader(r io.Reader, callback func(string)) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		text := scanner.Text()
		if strings.TrimSpace(text) != "" {
			callback(text)
		}
	}
}

func quoteCmd(cmd string) string {
	return "'" + strings.ReplaceAll(cmd, "'", "'\\''") + "'"
}

func contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

func classifyStderrLine(line string) string {
	lower := strings.ToLower(line)

	// Harmless progress/status indicators in stderr from Docker, APT, Curl, systemd, etc.
	infoKeywords := []string{
		"pulling fs layer", "downloading", "extracting", "verifying checksum",
		"download complete", "pull complete", "already exists", "waiting",
		"creating", "created", "starting", "started", "network ", "volume ",
		"container ", "get:", "hit:", "ign:", "selecting previously unselected",
		"preparing to unpack", "unpacking", "setting up", "processing triggers",
		"debconf:", "warning:", "notice:", "info:", "progress:",
	}
	for _, kw := range infoKeywords {
		if strings.Contains(lower, kw) {
			return "info"
		}
	}

	// Explicit critical error keywords
	errorKeywords := []string{
		"fatal:", "failed to", "failed:", "failure:", "panic:", "exception:",
		"permission denied", "command not found", "no such file or directory",
		"address already in use", "connection refused", "segmentation fault",
	}
	for _, kw := range errorKeywords {
		if strings.Contains(lower, kw) {
			return "error"
		}
	}

	return "stderr"
}

func deployStealthAllInOne(
	ctx context.Context,
	client *ssh.Client,
	plan types.DeploymentPlan,
	targetDir string,
	runRemoteCmd func(title, command string) error,
	emitLog func(status, msg string),
) (*types.RemoteDeployResult, error) {
	emitLog("step", "Запуск архитектуры единого порта 443 (eGames Stealth Fallback)...")

	panelDomain := dns.NormalizeDomain(plan.PanelDomain)
	if panelDomain == "" {
		panelDomain = dns.NormalizeDomain(plan.Domain)
	}
	subDomain := dns.NormalizeDomain(plan.SubscriptionDomain)
	decoyDomain := strings.TrimSpace(plan.DecoyDomain)
	if decoyDomain == "" {
		decoyDomain = panelDomain
	}

	// 1. Setup Decoy website in /var/www/html
	emitLog("step", "Подготовка Decoy-сайта маскировки (/var/www/html)...")
	decoyScript := buildDecoySiteScript("/var/www/html", plan.DecoyTemplate, plan.DecoyCloneURL)
	_ = runRemoteCmd("Подготовка и клонирование сайта маскировки", decoyScript)
	emitLog("success", "Decoy-сайт маскировки развернут в /var/www/html")

	// 2. SSL Setup
	emitLog("step", "Выпуск и подготовка SSL-сертификатов Let's Encrypt...")
	domainsArg := fmt.Sprintf("-d %s", panelDomain)
	if subDomain != "" && subDomain != panelDomain {
		domainsArg += fmt.Sprintf(" -d %s", subDomain)
	}
	if decoyDomain != "" && decoyDomain != panelDomain && decoyDomain != subDomain {
		domainsArg += fmt.Sprintf(" -d %s", decoyDomain)
	}

	sslPrepCmd := fmt.Sprintf(`
mkdir -p %s/ssl
docker stop remnawave-nginx remnanode 2>/dev/null || true
systemctl stop nginx apache2 caddy 2>/dev/null || true

openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout %s/ssl/privkey.key -out %s/ssl/fullchain.pem -subj '/CN=%s' 2>/dev/null || true

DOMAINS_ARG="%s"
CERT_SUCCESS=0
if command -v docker >/dev/null 2>&1; then
  docker run --rm -p 80:80 -v /etc/letsencrypt:/etc/letsencrypt -v /var/lib/letsencrypt:/var/lib/letsencrypt certbot/certbot certonly --standalone $DOMAINS_ARG --agree-tos --register-unsafely-without-email --non-interactive 2>&1 && CERT_SUCCESS=1 || true
fi
if [ "$CERT_SUCCESS" -eq 0 ] && [ ! -f /etc/letsencrypt/live/%s/fullchain.pem ]; then
  which certbot >/dev/null 2>&1 || (apt-get update -qq && (apt-get install -y -qq certbot || (apt-get install -y -qq software-properties-common && add-apt-repository -y universe && apt-get update -qq && apt-get install -y -qq certbot) || snap install --classic certbot) >/dev/null 2>&1 || true)
  if command -v certbot >/dev/null 2>&1; then
    certbot certonly --standalone $DOMAINS_ARG --agree-tos --register-unsafely-without-email --non-interactive 2>&1 || true
  fi
fi
if [ -f /etc/letsencrypt/live/%s/fullchain.pem ]; then
  cp -fL /etc/letsencrypt/live/%s/fullchain.pem %s/ssl/fullchain.pem
  cp -fL /etc/letsencrypt/live/%s/privkey.pem %s/ssl/privkey.key
  echo "Сертификат Let's Encrypt успешно получен и скопирован!"
fi
`, targetDir, targetDir, targetDir, panelDomain, domainsArg, panelDomain, panelDomain, panelDomain, targetDir, panelDomain, targetDir)
	_ = runRemoteCmd("Настройка SSL сертификатов", sslPrepCmd)

	// 3. Jolymmiles Custom Xray Core (if requested)
	if plan.XrayCoreType == "jolymmiles" {
		emitLog("step", "Загрузка кастомного ядра Xray (Jolymmiles Anti-TSPU с поддержкой MLKEM768)...")
		xrayCoreCmd := fmt.Sprintf(`
mkdir -p %s /tmp/xray_dl
which unzip >/dev/null 2>&1 || (apt-get update -qq && apt-get install -y -qq unzip curl >/dev/null 2>&1 || true)
CORE_OK=0
for PREFIX in "https://ghfast.top/" "https://gh-proxy.com/" "https://ghproxy.net/" ""; do
  URL="${PREFIX}https://github.com/Jolymmiles/Xray-core/releases/download/v26.9.5-0936/Xray-linux-64.zip"
  if curl -fsSL --connect-timeout 8 --max-time 60 -o /tmp/xray_dl/xray.zip "$URL" 2>/dev/null; then
    if [ -s /tmp/xray_dl/xray.zip ] && [ "$(head -c 2 /tmp/xray_dl/xray.zip)" = "PK" ]; then
      unzip -q -o /tmp/xray_dl/xray.zip -d /tmp/xray_dl/out 2>/dev/null || true
      if [ -f /tmp/xray_dl/out/xray ]; then
        mv -f /tmp/xray_dl/out/xray %s/xray-custom
        chmod +x %s/xray-custom
        CORE_OK=1
        break
      fi
    fi
  fi
done
rm -rf /tmp/xray_dl
if [ "$CORE_OK" -eq 1 ]; then
  echo "Кастомное ядро Jolymmiles Xray успешно установлено: $(%s/xray-custom version 2>/dev/null | head -n1)"
else
  echo "Предупреждение: не удалось загрузить кастомное ядро, используется стандартный образ."
fi
`, targetDir, targetDir, targetDir, targetDir)
		_ = runRemoteCmd("Установка Jolymmiles Xray Core", xrayCoreCmd)
	}

	// 4. Credentials and Configuration files
	appSecret := generator.RandomHex(16)
	dbPass := generator.RandomHex(16)
	metricsPass := generator.RandomHex(16)
	adminMode := plan.AdminCreationMode
	if adminMode == "" {
		adminMode = "auto"
	}
	superAdminUser := "admin"
	superAdminPass := generator.RandomAdminPassword(28)
	if adminMode == "custom" {
		superAdminUser = strings.TrimSpace(plan.AdminUsername)
		if superAdminUser == "" {
			superAdminUser = "admin"
		}
		superAdminPass = strings.TrimSpace(plan.AdminPassword)
		if len(superAdminPass) < 24 {
			superAdminPass = generator.RandomAdminPassword(28)
		}
	} else if adminMode == "manual" {
		superAdminUser = ""
		superAdminPass = ""
	}
	tempApiToken := "INIT_AUTOPILOT_TOKEN"
	tempSecretKey := "PUBLIC KEY FROM REMNAWAVE-PANEL"

	// Preserve existing credentials if present
	checkDbCmd := fmt.Sprintf(`cat %s/.env 2>/dev/null || true`, targetDir)
	session, err := client.NewSession()
	if err == nil {
		out, err := session.Output(checkDbCmd)
		session.Close()
		if err == nil && len(out) > 0 {
			lines := strings.Split(string(out), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "POSTGRES_PASSWORD=") {
					dbPass = strings.Trim(strings.TrimPrefix(line, "POSTGRES_PASSWORD="), "\"' \r\n")
				} else if strings.HasPrefix(line, "APP_SECRET=") {
					appSecret = strings.Trim(strings.TrimPrefix(line, "APP_SECRET="), "\"' \r\n")
				} else if strings.HasPrefix(line, "METRICS_PASS=") {
					metricsPass = strings.Trim(strings.TrimPrefix(line, "METRICS_PASS="), "\"' \r\n")
				}
			}
		}
	}

	envContent := generator.RenderPanelEnvWithCreds(
		panelDomain,
		subDomain,
		subDomain != "",
		appSecret,
		dbPass,
		metricsPass,
		"",
	)
	_ = runRemoteCmd("Запись .env", fmt.Sprintf("cat > %s/.env <<'EOF'\n%s\nEOF", targetDir, envContent))

	nginxConf := generator.RenderStealthNginxConf(panelDomain, subDomain, decoyDomain, plan.SecretGate, plan.UseCamouflage)
	_ = runRemoteCmd("Запись nginx.conf (Stealth Fallback)", fmt.Sprintf("cat > %s/nginx.conf <<'EOF'\n%s\nEOF", targetDir, nginxConf))

	composeContent := generator.RenderAllInOneStealthCompose(appSecret, dbPass, metricsPass, tempApiToken, tempSecretKey, plan.XrayCoreType)
	_ = runRemoteCmd("Запись docker-compose.yml", fmt.Sprintf("cat > %s/docker-compose.yml <<'EOF'\n%s\nEOF", targetDir, composeContent))

	// 5. Pull & Up Stack
	emitLog("step", "Загрузка Docker-образов и запуск единого стека...")
	_ = runRemoteCmd("Остановка предыдущих контейнеров", fmt.Sprintf("cd %s && docker compose down 2>/dev/null || true", targetDir))
	_ = runRemoteCmd("Загрузка образов", fmt.Sprintf("cd %s && docker compose pull", targetDir))
	if err := runRemoteCmd("Запуск единого стека Remnawave", fmt.Sprintf("cd %s && docker compose up -d", targetDir)); err != nil {
		return nil, err
	}

	// 6. Zero-Touch Autopilot Execution
	emitLog("step", "Запуск Zero-Touch Autopilot (автоматическая привязка ноды, профиля и подписок)...")
	autopilotScript := fmt.Sprintf(`
set -e
which jq >/dev/null 2>&1 || (apt-get update -qq && apt-get install -y -qq jq >/dev/null 2>&1 || true)

API_BASE="http://127.0.0.1:3000"
ADMIN_MODE="%s"
SUPERADMIN_USER="%s"
SUPERADMIN_PASS="%s"
DECOY_DOMAIN="%s"
TARGET_DIR="%s"
TORRENT_BLOCK="%t"
SPAM_BLOCK="%t"

# Multi-fallback JSON parser helper (jq -> python3 -> grep/sed)
extract_token() {
  local json_input="$1"
  local field="$2"
  if command -v jq >/dev/null 2>&1; then
    local res
    res=$(echo "$json_input" | jq -r ".response.${field} // empty" 2>/dev/null || true)
    if [ -n "$res" ] && [ "$res" != "null" ]; then
      echo "$res"
      return 0
    fi
  fi
  if command -v python3 >/dev/null 2>&1; then
    local res
    res=$(python3 -c "import sys, json; data=json.loads(sys.stdin.read()); print(data.get('response', {}).get('${field}') or '')" <<< "$json_input" 2>/dev/null || true)
    if [ -n "$res" ] && [ "$res" != "null" ]; then
      echo "$res"
      return 0
    fi
  fi
  echo "$json_input" | grep -o "\"${field}\":\"[^\"]*\"" | cut -d'"' -f4 || true
}

# Preserve credentials if already exist
if [ -f "${TARGET_DIR}/credentials.txt" ]; then
  SAVED_PASS=$(grep '^ADMIN_PASSWORD=' "${TARGET_DIR}/credentials.txt" | cut -d= -f2- | tr -d '\r\n[:space:]')
  if [ -n "$SAVED_PASS" ]; then
    SUPERADMIN_PASS="$SAVED_PASS"
  fi
fi

echo "Ожидание готовности API Remnawave (до 3 минут)..."
API_READY=0
for i in $(seq 1 60); do
  if curl -s -f --connect-timeout 4 --max-time 10 "${API_BASE}/api/auth/status" \
      -H "X-Forwarded-For: 127.0.0.1" \
      -H "X-Forwarded-Proto: https" >/dev/null 2>&1; then
    echo "Remnawave API успешно ответил (попытка $i)!"
    API_READY=1
    break
  fi
  sleep 3
done

IS_CONFIGURED=$(curl -s "${API_BASE}/api/auth/status" \
  -H "X-Forwarded-For: 127.0.0.1" \
  -H "X-Forwarded-Proto: https" 2>/dev/null | grep -oE '"isConfigured":true|"configured":true' || true)

TOKEN=""
if [ "${ADMIN_MODE}" = "manual" ]; then
  echo "Режим регистрации: ручной. Завершите регистрацию суперадминистратора через веб-интерфейс панели."
else
  if [ -z "$IS_CONFIGURED" ]; then
    echo "Регистрация администратора Remnawave в API..."
    cat > /tmp/rw_reg.json << JSON_REG_EOF
{"username":"${SUPERADMIN_USER}","password":"${SUPERADMIN_PASS}"}
JSON_REG_EOF
    REG_RES=$(curl -s -X POST "${API_BASE}/api/auth/register" \
      -H "Content-Type: application/json" \
      -H "X-Forwarded-For: 127.0.0.1" \
      -H "X-Forwarded-Proto: https" \
      -H "X-Remnawave-Client-Type: browser" \
      --data-binary @/tmp/rw_reg.json)
    rm -f /tmp/rw_reg.json
    TOKEN=$(extract_token "$REG_RES" "accessToken")
  fi

  if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
    echo "Попытка авторизации администратора Remnawave..."
    cat > /tmp/rw_login.json << JSON_LOGIN_EOF
{"username":"${SUPERADMIN_USER}","password":"${SUPERADMIN_PASS}"}
JSON_LOGIN_EOF
    LOGIN_RES=$(curl -s -X POST "${API_BASE}/api/auth/login" \
      -H "Content-Type: application/json" \
      -H "X-Forwarded-For: 127.0.0.1" \
      -H "X-Forwarded-Proto: https" \
      -H "X-Remnawave-Client-Type: browser" \
      --data-binary @/tmp/rw_login.json)
    rm -f /tmp/rw_login.json
    TOKEN=$(extract_token "$LOGIN_RES" "accessToken")
  fi

  if echo "$TOKEN" | grep -q '^ey'; then
    echo "Суперадминистратор успешно подтвержден в API Remnawave!"
    mkdir -p "${TARGET_DIR}" 2>/dev/null || true
    echo "$TOKEN" > "${TARGET_DIR}/token" 2>/dev/null || true
    chmod 600 "${TARGET_DIR}/token" 2>/dev/null || true
    echo "ADMIN_USERNAME=${SUPERADMIN_USER}" >> "${TARGET_DIR}/.env" 2>/dev/null || true
    echo "ADMIN_PASSWORD=${SUPERADMIN_PASS}" >> "${TARGET_DIR}/.env" 2>/dev/null || true
    cat > "${TARGET_DIR}/credentials.txt" <<CRED_EOF
ADMIN_USERNAME=${SUPERADMIN_USER}
ADMIN_PASSWORD=${SUPERADMIN_PASS}
CRED_EOF
    chmod 600 "${TARGET_DIR}/credentials.txt" 2>/dev/null || true
  else
    echo "ВНИМАНИЕ: Автоматическая регистрация суперадминистратора не завершилась через API."
    echo "Завершите регистрацию аккаунта суперадминистратора вручную через веб-интерфейс панели."
    rm -f "${TARGET_DIR}/token"
  fi
fi

# Mint & save master token for maintenance
if [ -n "$TOKEN" ] && [ "$TOKEN" != "null" ]; then
  MASTER_PAYLOAD='{"name":"remna-ui-manager","expiresInDays":3650,"scopes":["*"]}'
  MASTER_RES=$(curl -s --connect-timeout 10 --max-time 30 -X POST "${API_BASE}/api/tokens" \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json" \
    -H "X-Forwarded-For: 127.0.0.1" \
    -H "X-Forwarded-Proto: https" \
    -H "X-Remnawave-Client-Type: browser" \
    -d "${MASTER_PAYLOAD}")
  MASTER_TOKEN=$(extract_token "$MASTER_RES" "token")
  if [ -n "$MASTER_TOKEN" ] && [ "$MASTER_TOKEN" != "null" ]; then
    echo "$MASTER_TOKEN" > "${TARGET_DIR}/token"
    chmod 600 "${TARGET_DIR}/token" 2>/dev/null || true
  fi
fi

if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
  echo "Не удалось получить токен администратора. Пропускаем автоматическую привязку."
  exit 0
fi

echo "Администратор успешно авторизован!"

PUBKEY=$(curl -s "${API_BASE}/api/keygen" -H "Authorization: Bearer ${TOKEN}" | jq -r '.response.secretKey // .response.pubKey // empty')
if [ -n "$PUBKEY" ] && [ "$PUBKEY" != "null" ]; then
  echo "Публичный ключ ноды успешно получен!"
  sed -i "s|SECRET_KEY=.*|SECRET_KEY="${PUBKEY}"|g" "${TARGET_DIR}/docker-compose.yml"
fi

REALITY_RES=$(curl -s "${API_BASE}/api/system/tools/x25519/generate" -H "Authorization: Bearer ${TOKEN}")
PRIVKEY=$(echo "$REALITY_RES" | jq -r '.response.keypairs[0].privateKey // empty')
if [ -z "$PRIVKEY" ] || [ "$PRIVKEY" = "null" ]; then
  PRIVKEY=$(openssl rand -base64 32)
fi
SHORT_ID=$(openssl rand -hex 8)

DEF_PROFILES=$(curl -s "${API_BASE}/api/config-profiles" -H "Authorization: Bearer ${TOKEN}")
DEF_UUID=$(echo "$DEF_PROFILES" | jq -r '(.response.configProfiles[0].uuid // .response[0].uuid // empty)')
if [ -n "$DEF_UUID" ] && [ "$DEF_UUID" != "null" ]; then
  curl -s -X DELETE "${API_BASE}/api/config-profiles/${DEF_UUID}" -H "Authorization: Bearer ${TOKEN}" >/dev/null 2>&1 || true
fi

RULES='[{"ip":["geoip:private"],"outboundTag":"BLOCK"}'
if [ "$TORRENT_BLOCK" = "true" ]; then
  RULES="${RULES}"',{"protocol":["bittorrent"],"outboundTag":"BLOCK"}'
fi
if [ "$SPAM_BLOCK" = "true" ]; then
  RULES="${RULES}"',{"port":"25,465,587","outboundTag":"BLOCK"}'
fi
RULES="${RULES}"']'

PROFILE_PAYLOAD=$(jq -n   --arg name "Stealth Reality (eGames)"   --arg domain "${DECOY_DOMAIN}"   --arg privkey "${PRIVKEY}"   --arg shortid "${SHORT_ID}"   --argjson rules "${RULES}"   '{
    name: $name,
    config: {
      log: { loglevel: "warning" },
      dns: { queryStrategy: "UseIPv4", servers: [{ address: "https://dns.google/dns-query", skipFallback: false }] },
      inbounds: [{
        tag: "StealthInbound",
        port: 443,
        protocol: "vless",
        settings: { clients: [], decryption: "none" },
        sniffing: { enabled: true, destOverride: ["http", "tls", "quic"] },
        streamSettings: {
          network: "tcp",
          security: "reality",
          realitySettings: {
            show: false,
            xver: 1,
            dest: "/dev/shm/nginx.sock",
            spiderX: "",
            minClientVer: "0.0.0",
            shortIds: [$shortid],
            privateKey: $privkey,
            serverNames: [$domain]
          }
        }
      }],
      outbounds: [{ tag: "DIRECT", protocol: "freedom" }, { tag: "BLOCK", protocol: "blackhole" }],
      routing: { rules: $rules }
    }
  }')

PROF_RES=$(curl -s -X POST "${API_BASE}/api/config-profiles"   -H "Authorization: Bearer ${TOKEN}"   -H "Content-Type: application/json"   -d "${PROFILE_PAYLOAD}")

CONFIG_UUID=$(echo "$PROF_RES" | jq -r '(.response.uuid // .response.configProfile.uuid // empty)')
INBOUND_UUID=$(echo "$PROF_RES" | jq -r '(.response.inbounds[0].uuid // .response.configProfile.inbounds[0].uuid // empty)')

if [ -n "$CONFIG_UUID" ] && [ "$CONFIG_UUID" != "null" ]; then
  echo "Профиль маскировки Stealth Reality успешно создан!"

  NODE_PAYLOAD=$(jq -n     --arg name "Master Node (Stealth 443)"     --arg address "127.0.0.1"     --arg cfgUuid "${CONFIG_UUID}"     --arg inUuid "${INBOUND_UUID}"     '{
      name: $name,
      address: $address,
      port: 2222,
      configProfileUuid: $cfgUuid,
      inboundUuid: $inUuid,
      isTrafficTrackingActive: true
    }')
  curl -s -X POST "${API_BASE}/api/nodes"     -H "Authorization: Bearer ${TOKEN}"     -H "Content-Type: application/json"     -d "${NODE_PAYLOAD}" >/dev/null 2>&1 || true

  HOST_PAYLOAD=$(jq -n     --arg cfgUuid "${CONFIG_UUID}"     --arg inUuid "${INBOUND_UUID}"     --arg addr "${DECOY_DOMAIN}"     '{
      inbound: { configProfileUuid: $cfgUuid, configProfileInboundUuid: $inUuid },
      remark: "Stealth Reality Fallback",
      address: $addr,
      port: 443,
      path: "",
      sni: $addr,
      host: "",
      alpn: null,
      fingerprint: "chrome",
      allowInsecure: false,
      isDisabled: false
    }')
  curl -s -X POST "${API_BASE}/api/hosts"     -H "Authorization: Bearer ${TOKEN}"     -H "Content-Type: application/json"     -d "${HOST_PAYLOAD}" >/dev/null 2>&1 || true

  SQUADS=$(curl -s "${API_BASE}/api/internal-squads" -H "Authorization: Bearer ${TOKEN}")
  SQUAD_UUID=$(echo "$SQUADS" | jq -r '(.response.internalSquads[0].uuid // .response[0].uuid // empty)')
  if [ -n "$SQUAD_UUID" ] && [ "$SQUAD_UUID" != "null" ] && [ -n "$INBOUND_UUID" ]; then
    SQUAD_PAYLOAD=$(jq -n --arg sqUuid "${SQUAD_UUID}" --arg inUuid "${INBOUND_UUID}" '{uuid: $sqUuid, inbounds: [$inUuid]}')
    curl -s -X PATCH "${API_BASE}/api/internal-squads" \
      -H "Authorization: Bearer ${TOKEN}" \
      -H "Content-Type: application/json" \
      -d "${SQUAD_PAYLOAD}" >/dev/null 2>&1 || \
    curl -s -X PATCH "${API_BASE}/api/internal-squads/${SQUAD_UUID}" \
      -H "Authorization: Bearer ${TOKEN}" \
      -H "Content-Type: application/json" \
      -d "{\"inbounds\":[\"${INBOUND_UUID}\"]}" >/dev/null 2>&1 || true
    echo "Нода и сквад успешно сконфигурированы!"
  fi
fi

TOKEN_PAYLOAD=$(jq -n --arg n "subscription-page" '{name:$n, expiresInDays:3650, scopes:["subscription-page-configs:list","subscription-page-configs:get","subscriptions:subpage-config","system:metadata","users:by-username"]}')
TOKEN_RES=$(curl -s --connect-timeout 10 --max-time 30 -X POST "${API_BASE}/api/tokens" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -H "X-Forwarded-For: 127.0.0.1" \
  -H "X-Forwarded-Proto: https" \
  -H "X-Remnawave-Client-Type: browser" \
  -d "${TOKEN_PAYLOAD}")
SUB_API_TOKEN=$(echo "$TOKEN_RES" | jq -r '.response.token // empty')
if [ -n "$SUB_API_TOKEN" ] && [ "$SUB_API_TOKEN" != "null" ]; then
  echo "API-токен для страницы подписок успешно сгенерирован!"
  sed -i "s|REMNAWAVE_API_TOKEN=.*|REMNAWAVE_API_TOKEN=${SUB_API_TOKEN}|g" "${TARGET_DIR}/.env" "${TARGET_DIR}/docker-compose.yml" "${TARGET_DIR}/subscription/.env" 2>/dev/null || true
  docker compose -f "${TARGET_DIR}/docker-compose.yml" restart remnawave-subscription-page 2>/dev/null || true
fi

echo "Перезапуск сервисов с финальными ключами безопасности..."
cd "${TARGET_DIR}" && docker compose down >/dev/null 2>&1 && docker compose up -d >/dev/null 2>&1 || true
echo "Стек успешно запущен в режиме единого порта 443!"
`,
		adminMode,
		superAdminUser,
		superAdminPass,
		decoyDomain,
		targetDir,
		plan.TorrentBlocker,
		plan.BlockSpamPorts,
	)
	_ = runRemoteCmd("Выполнение Autopilot", autopilotScript)

	// 7. Configure UFW port isolation (Stealth Mode)
	sshPort := plan.SshPort
	if sshPort <= 0 {
		sshPort = 22
	}
	ufwCmd := fmt.Sprintf(`
which ufw >/dev/null 2>&1 || (apt-get update -qq && apt-get install -y -qq ufw >/dev/null 2>&1 || true)
if command -v ufw >/dev/null 2>&1; then
  ufw allow 80/tcp >/dev/null 2>&1 || true
  ufw allow 443/tcp >/dev/null 2>&1 || true
  ufw allow %d/tcp >/dev/null 2>&1 || true
  ufw delete allow 2222/tcp 2>/dev/null || true
  ufw allow from 172.30.0.0/16 to any port 2222 proto tcp >/dev/null 2>&1 || true
  ufw --force enable >/dev/null 2>&1 || true
  echo "UFW фаервол настроен: порт 2222 изолирован внутри Docker, порты 80/443/SSH открыты!"
fi
`, sshPort)
	_ = runRemoteCmd("Настройка UFW фаервола и изоляция портов", ufwCmd)

	// 8. Result reporting
	panelURL := fmt.Sprintf("https://%s", panelDomain)
	if plan.SecretGate != "" {
		cleanGate := strings.TrimLeft(strings.TrimSpace(plan.SecretGate), "/")
		panelURL = fmt.Sprintf("https://%s/%s", panelDomain, cleanGate)
	}
	subURL := ""
	if subDomain != "" {
		subURL = fmt.Sprintf("https://%s", subDomain)
	}

	emitLog("success", "==================================================")
	emitLog("success", "ВСЕ КОМПОНЕНТЫ УСПЕШНО РАЗВЁРНУТЫ В РЕЖИМЕ СТЕЛС-443!")
	emitLog("success", fmt.Sprintf("Панель управления: %s", panelURL))
	// Check if admin token was minted successfully
	tokenCheckCmd := fmt.Sprintf(`if [ -f "%s/token" ] && grep -q '^ey' "%s/token" 2>/dev/null; then echo "VALID_TOKEN"; else echo "NO_TOKEN"; fi`, targetDir, targetDir)
	tokenSess, tokenErr := client.NewSession()
	var tokenOut []byte
	if tokenErr == nil {
		tokenOut, _ = tokenSess.Output(tokenCheckCmd)
		tokenSess.Close()
	}
	isTokenValid := strings.Contains(string(tokenOut), "VALID_TOKEN")

	if isTokenValid && adminMode != "manual" {
		emitLog("success", fmt.Sprintf("Логин администратора: %s", superAdminUser))
		emitLog("success", fmt.Sprintf("Пароль администратора: %s", superAdminPass))
	} else {
		adminMode = "manual"
		superAdminUser = ""
		superAdminPass = ""
		emitLog("info", "Регистрация: самостоятельная. Завершите регистрацию суперадминистратора в открывшейся веб-панели.")
	}
	if subURL != "" {
		emitLog("success", fmt.Sprintf("Страница подписок: %s", subURL))
	}
	emitLog("info", "Порт Reality: 443 | Сокет: /dev/shm/nginx.sock | Порт 2222 скрыт от сканеров")
	emitLog("success", "==================================================")

	return &types.RemoteDeployResult{
		Success:           true,
		AdminCreationMode: adminMode,
		AdminUser:         superAdminUser,
		AdminPass:         superAdminPass,
		PanelURL:          panelURL,
		SubURL:            subURL,
	}, nil
}
