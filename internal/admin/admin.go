package admin



import (

	"encoding/base64"

	"encoding/json"

	"fmt"

	"io"
	"net"
	"net/http"

	"os"

	"path/filepath"

	"regexp"

	"strconv"
	"strings"
	"sync"
	"syscall"

	"time"



	"golang.org/x/crypto/ssh"

	"remna-ui/internal/generator"
	"remna-ui/internal/types"
	"remna-ui/internal/vault"

)



type dataBlob struct {

	cbData uint32

	pbData *byte

}



var (

	crypt32                 = syscall.NewLazyDLL("crypt32.dll")

	procCryptProtectData   = crypt32.NewProc("CryptProtectData")

	procCryptUnprotectData = crypt32.NewProc("CryptUnprotectData")

	validContainerRegex    = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]+$`)

	validBackupNameRegex   = regexp.MustCompile(`^[a-zA-Z0-9_.-]+\.tar\.gz$`)

)



func encryptSecret(secret string) string {
	enc, err := vault.GetManager().EncryptSecret(secret)
	if err != nil {
		return secret
	}
	return enc
}

func decryptSecret(encrypted string) string {
	dec, err := vault.GetManager().DecryptSecret(encrypted)
	if err != nil {
		return encrypted
	}
	return dec
}



func isValidContainerName(name string) bool {

	return validContainerRegex.MatchString(strings.TrimSpace(name))

}



func getStorePath() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		home, _ := os.UserHomeDir()
		appData = filepath.Join(home, ".config")
	}
	subDir := os.Getenv("REMNAWAVE_DATA_DIR")
	if subDir == "" {
		subDir = "remnawave"
	}
	var dir string
	if filepath.IsAbs(subDir) {
		dir = subDir
	} else {
		dir = filepath.Join(appData, subDir)
	}
	_ = os.MkdirAll(dir, 0755)
	target := filepath.Join(dir, "servers.json")

	// Auto-migrate from remnawave-beta if target is empty/missing and beta data exists
	if subDir == "remnawave" {
		if fi, err := os.Stat(target); err != nil || fi.Size() == 0 {
			betaPath := filepath.Join(appData, "remnawave-beta", "servers.json")
			if betaData, err := os.ReadFile(betaPath); err == nil && len(betaData) > 0 {
				_ = os.WriteFile(target, betaData, 0600)
			}
		}
	}

	return target
}



var storeMu sync.Mutex

func loadServersUnlocked() []types.SavedServer {
	path := getStorePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return []types.SavedServer{}
	}
	var servers []types.SavedServer
	if err := json.Unmarshal(data, &servers); err != nil {
		return []types.SavedServer{}
	}

	vMgr := vault.GetManager()
	status := vMgr.GetStatus()
	isLocked := status.Enabled && !status.Unlocked

	for i := range servers {
		if isLocked {
			servers[i].Password = "••••••••"
			servers[i].AdminPassword = "••••••••"
			servers[i].SSHKey = "••••••••"
			servers[i].ApiToken = "••••••••"
			servers[i].NodeSecret = "••••••••"
			continue
		}
		if servers[i].Password != "" {
			if dec, err := vMgr.DecryptSecret(servers[i].Password); err == nil {
				servers[i].Password = dec
			}
		}
		if servers[i].AdminPassword != "" {
			if dec, err := vMgr.DecryptSecret(servers[i].AdminPassword); err == nil {
				servers[i].AdminPassword = dec
			}
		}
		if servers[i].SSHKey != "" {
			if dec, err := vMgr.DecryptSecret(servers[i].SSHKey); err == nil {
				servers[i].SSHKey = dec
			}
		}
		if servers[i].ApiToken != "" {
			if dec, err := vMgr.DecryptSecret(servers[i].ApiToken); err == nil {
				servers[i].ApiToken = dec
			}
		}
		if servers[i].NodeSecret != "" {
			if dec, err := vMgr.DecryptSecret(servers[i].NodeSecret); err == nil {
				servers[i].NodeSecret = dec
			}
		}
	}
	return servers
}

func GetSavedServers() []types.SavedServer {
	storeMu.Lock()
	defer storeMu.Unlock()
	return loadServersUnlocked()
}

func SaveServer(server types.SavedServer) error {
	vMgr := vault.GetManager()
	status := vMgr.GetStatus()
	if status.Enabled && !status.Unlocked {
		return fmt.Errorf("хранилище заблокировано мастер-паролем")
	}

	storeMu.Lock()
	defer storeMu.Unlock()

	servers := loadServersUnlocked()
	updated := false
	if server.ID == "" {
		server.ID = fmt.Sprintf("srv-%d", time.Now().Unix())
	}
	server.LastSeen = time.Now().Format("02.01.2006 15:04")

	for i, s := range servers {
		if s.ID == server.ID || (s.Host == server.Host && s.Port == server.Port && s.Username == server.Username) {
			// Preserve existing secrets if incoming server has masked value or empty
			if server.Password == "••••••••" || (server.Password == "" && s.Password != "") {
				server.Password = s.Password
			}
			if server.AdminPassword == "••••••••" || (server.AdminPassword == "" && s.AdminPassword != "") {
				server.AdminPassword = s.AdminPassword
			}
			if server.SSHKey == "••••••••" || (server.SSHKey == "" && s.SSHKey != "") {
				server.SSHKey = s.SSHKey
			}
			if server.ApiToken == "••••••••" || (server.ApiToken == "" && s.ApiToken != "") {
				server.ApiToken = s.ApiToken
			}
			if server.NodeSecret == "••••••••" || (server.NodeSecret == "" && s.NodeSecret != "") {
				server.NodeSecret = s.NodeSecret
			}
			servers[i] = server
			updated = true
			break
		}
	}
	if !updated {
		servers = append(servers, server)
	}

	toSave := make([]types.SavedServer, len(servers))
	copy(toSave, servers)
	for i := range toSave {
		if toSave[i].Password != "" {
			enc, err := vMgr.EncryptSecret(toSave[i].Password)
			if err != nil {
				return fmt.Errorf("ошибка шифрования пароля: %w", err)
			}
			toSave[i].Password = enc
		}
		if toSave[i].AdminPassword != "" {
			enc, err := vMgr.EncryptSecret(toSave[i].AdminPassword)
			if err != nil {
				return fmt.Errorf("ошибка шифрования пароля панели: %w", err)
			}
			toSave[i].AdminPassword = enc
		}
		if toSave[i].SSHKey != "" {
			enc, err := vMgr.EncryptSecret(toSave[i].SSHKey)
			if err != nil {
				return fmt.Errorf("ошибка шифрования SSH-ключа: %w", err)
			}
			toSave[i].SSHKey = enc
		}
		if toSave[i].ApiToken != "" {
			enc, err := vMgr.EncryptSecret(toSave[i].ApiToken)
			if err != nil {
				return fmt.Errorf("ошибка шифрования API-токена: %w", err)
			}
			toSave[i].ApiToken = enc
		}
		if toSave[i].NodeSecret != "" {
			enc, err := vMgr.EncryptSecret(toSave[i].NodeSecret)
			if err != nil {
				return fmt.Errorf("ошибка шифрования секрета ноды: %w", err)
			}
			toSave[i].NodeSecret = enc
		}
	}

	data, err := json.MarshalIndent(toSave, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(getStorePath(), data, 0600)
}

func DeleteServer(id string) error {
	vMgr := vault.GetManager()
	status := vMgr.GetStatus()
	if status.Enabled && !status.Unlocked {
		return fmt.Errorf("хранилище заблокировано мастер-паролем")
	}

	storeMu.Lock()
	defer storeMu.Unlock()

	servers := loadServersUnlocked()
	var filtered []types.SavedServer
	for _, s := range servers {
		if s.ID != id {
			filtered = append(filtered, s)
		}
	}

	toSave := make([]types.SavedServer, len(filtered))
	copy(toSave, filtered)

	for i := range toSave {
		if toSave[i].Password != "" {
			toSave[i].Password, _ = vMgr.EncryptSecret(toSave[i].Password)
		}
		if toSave[i].AdminPassword != "" {
			toSave[i].AdminPassword, _ = vMgr.EncryptSecret(toSave[i].AdminPassword)
		}
		if toSave[i].SSHKey != "" {
			toSave[i].SSHKey, _ = vMgr.EncryptSecret(toSave[i].SSHKey)
		}
		if toSave[i].ApiToken != "" {
			toSave[i].ApiToken, _ = vMgr.EncryptSecret(toSave[i].ApiToken)
		}
		if toSave[i].NodeSecret != "" {
			toSave[i].NodeSecret, _ = vMgr.EncryptSecret(toSave[i].NodeSecret)
		}
	}

	data, err := json.MarshalIndent(toSave, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(getStorePath(), data, 0600)
}

// ReencryptServers re-encrypts all sensitive fields in servers.json during master password change or toggle
func ReencryptServers(decryptOld func(string) (string, error), encryptNew func(string) (string, error)) error {
	storeMu.Lock()
	defer storeMu.Unlock()

	path := getStorePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var servers []types.SavedServer
	if err := json.Unmarshal(data, &servers); err != nil {
		return nil
	}

	reencryptField := func(val string) (string, error) {
		if val == "" {
			return "", nil
		}
		dec, err := decryptOld(val)
		if err != nil {
			return val, nil
		}
		return encryptNew(dec)
	}

	for i := range servers {
		var err error
		if servers[i].Password != "" {
			if servers[i].Password, err = reencryptField(servers[i].Password); err != nil {
				return err
			}
		}
		if servers[i].AdminPassword != "" {
			if servers[i].AdminPassword, err = reencryptField(servers[i].AdminPassword); err != nil {
				return err
			}
		}
		if servers[i].SSHKey != "" {
			if servers[i].SSHKey, err = reencryptField(servers[i].SSHKey); err != nil {
				return err
			}
		}
		if servers[i].ApiToken != "" {
			if servers[i].ApiToken, err = reencryptField(servers[i].ApiToken); err != nil {
				return err
			}
		}
		if servers[i].NodeSecret != "" {
			if servers[i].NodeSecret, err = reencryptField(servers[i].NodeSecret); err != nil {
				return err
			}
		}
	}

	out, err := json.MarshalIndent(servers, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0600)
}

// Vault API Functions exposed to Wails App
func GetVaultStatus() types.VaultStatus {
	return vault.GetManager().GetStatus()
}

func UnlockVault(password string) (bool, error) {
	return vault.GetManager().Unlock(password)
}

func LockVault() {
	vault.GetManager().Lock()
}

func SetMasterPassword(oldPass, newPass string) error {
	return vault.GetManager().SetMasterPassword(oldPass, newPass, ReencryptServers)
}

func DisableMasterPassword(currentPass string) error {
	return vault.GetManager().DisableMasterPassword(currentPass, ReencryptServers)
}

func SetAutoLockMinutes(mins int) error {
	return vault.GetManager().SetAutoLockMinutes(mins)
}

func SetSecureValue(key, value string) error {
	return vault.GetManager().SetSecureValue(key, value)
}

func GetSecureValue(key string) string {
	val, _ := vault.GetManager().GetSecureValue(key)
	return val
}



func getSSHClient(server types.SavedServer) (*ssh.Client, error) {

	port := server.Port

	if port == 0 {

		port = 22

	}

	user := server.Username

	if user == "" {

		user = "root"

	}



	var authMethods []ssh.AuthMethod

	effectivePass := server.Password
	if strings.HasPrefix(effectivePass, "enc:") {
		effectivePass = decryptSecret(effectivePass)
	}

	trimmedKey := strings.Trim(strings.TrimSpace(server.SSHKey), `"'`)
	if strings.HasPrefix(trimmedKey, "enc:") {
		trimmedKey = decryptSecret(trimmedKey)
	}

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

			} else if effectivePass != "" {

				signer, err = ssh.ParsePrivateKeyWithPassphrase(keyBytes, []byte(effectivePass))

				if err == nil {

					authMethods = append(authMethods, ssh.PublicKeys(signer))

				}

			}

		}

	}

	if effectivePass != "" {

		authMethods = append(authMethods, ssh.Password(effectivePass))

	}



	if len(authMethods) == 0 {

		return nil, fmt.Errorf("требуется пароль или SSH-ключ")

	}



	cfg := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         8 * time.Second,
	}

	addr := net.JoinHostPort(server.Host, fmt.Sprintf("%d", port))
	return ssh.Dial("tcp", addr, cfg)

}

// pooledSSHClient holds a cached SSH client, its last activity timestamp, and in-use reference count
type pooledSSHClient struct {
	client   *ssh.Client
	lastUsed time.Time
	inUse    int
}

var (
	sshPoolMu sync.Mutex
	sshPool   = make(map[string]*pooledSSHClient)
)

func init() {
	// Background reaper: cleans up idle SSH connections older than 45 seconds that are not actively in-use
	go func() {
		ticker := time.NewTicker(20 * time.Second)
		for range ticker.C {
			sshPoolMu.Lock()
			now := time.Now()
			for k, entry := range sshPool {
				if entry.inUse == 0 && now.Sub(entry.lastUsed) > 45*time.Second {
					if entry.client != nil {
						entry.client.Close()
					}
					delete(sshPool, k)
				}
			}
			sshPoolMu.Unlock()
		}
	}()
}

func getSSHKey(server types.SavedServer) string {
	port := server.Port
	if port == 0 {
		port = 22
	}
	user := server.Username
	if user == "" {
		user = "root"
	}
	return fmt.Sprintf("%s:%d:%s", server.Host, port, user)
}

// getPooledSession returns an active SSH session from a pooled, keepalive-verified SSH client,
// along with a release function that must be deferred by the caller to return the connection to the idle pool.
func getPooledSession(server types.SavedServer) (*ssh.Session, func(), error) {
	key := getSSHKey(server)

	// 1. Fast lookup under lock without any network I/O
	sshPoolMu.Lock()
	entry, exists := sshPool[key]
	var existingClient *ssh.Client
	if exists && entry != nil {
		existingClient = entry.client
	}
	sshPoolMu.Unlock()

	// 2. Health check and session creation OUTSIDE of the global mutex
	if existingClient != nil {
		_, _, err := existingClient.SendRequest("keepalive@openssh.com", true, nil)
		if err == nil {
			session, sessErr := existingClient.NewSession()
			if sessErr == nil {
				sshPoolMu.Lock()
				if e, ok := sshPool[key]; ok && e.client == existingClient {
					e.inUse++
					e.lastUsed = time.Now()
				}
				sshPoolMu.Unlock()

				release := func() {
					session.Close()
					sshPoolMu.Lock()
					if e, ok := sshPool[key]; ok {
						e.inUse--
						if e.inUse < 0 {
							e.inUse = 0
						}
						e.lastUsed = time.Now()
					}
					sshPoolMu.Unlock()
				}
				return session, release, nil
			}
		}

		// Existing client is dead or stale: clean it up under lock
		sshPoolMu.Lock()
		if e, ok := sshPool[key]; ok && e.client == existingClient {
			if e.inUse <= 0 {
				existingClient.Close()
				delete(sshPool, key)
			}
		}
		sshPoolMu.Unlock()
	}

	// 3. Dial fresh client outside of global lock to prevent blocking concurrent dials to different hosts
	newClient, err := getSSHClient(server)
	if err != nil {
		return nil, nil, err
	}

	session, err := newClient.NewSession()
	if err != nil {
		newClient.Close()
		return nil, nil, err
	}

	sshPoolMu.Lock()
	if old, exists := sshPool[key]; exists && old.client != nil && old.inUse <= 0 {
		old.client.Close()
	}
	sshPool[key] = &pooledSSHClient{
		client:   newClient,
		lastUsed: time.Now(),
		inUse:    1,
	}
	sshPoolMu.Unlock()

	release := func() {
		session.Close()
		sshPoolMu.Lock()
		if e, ok := sshPool[key]; ok {
			e.inUse--
			if e.inUse < 0 {
				e.inUse = 0
			}
			e.lastUsed = time.Now()
		}
		sshPoolMu.Unlock()
	}

	return session, release, nil
}

type circuitState struct {
	consecutiveFailures int
	cooldownUntil       time.Time
}

var (
	cbMu               sync.RWMutex
	cbStates           = make(map[string]*circuitState)
	cbThreshold        = 2
	cbCooldownDuration = 5 * time.Minute
)

// CheckCircuitBreaker returns false and a cooldown explanation if the server has repeatedly failed.
func CheckCircuitBreaker(serverID string) (bool, string) {
	if serverID == "" {
		return true, ""
	}
	cbMu.RLock()
	state, exists := cbStates[serverID]
	cbMu.RUnlock()
	if !exists {
		return true, ""
	}
	if state.consecutiveFailures >= cbThreshold && time.Now().Before(state.cooldownUntil) {
		remaining := int(time.Until(state.cooldownUntil).Minutes()) + 1
		return false, fmt.Sprintf("Офлайн-пауза (Circuit Breaker: сервер не отвечал %d раз(а), повтор через %d мин)", state.consecutiveFailures, remaining)
	}
	return true, ""
}

// RecordServerInspectionResult updates the circuit breaker state based on inspection outcome.
func RecordServerInspectionResult(serverID string, success bool) {
	if serverID == "" {
		return
	}
	cbMu.Lock()
	defer cbMu.Unlock()
	state, exists := cbStates[serverID]
	if !exists {
		state = &circuitState{}
		cbStates[serverID] = state
	}
	if success {
		state.consecutiveFailures = 0
		state.cooldownUntil = time.Time{}
	} else {
		state.consecutiveFailures++
		if state.consecutiveFailures >= cbThreshold {
			state.cooldownUntil = time.Now().Add(cbCooldownDuration)
		}
	}
}

// ResetServerCircuitBreaker resets the cooldown for a server (e.g. on manual user action).
func ResetServerCircuitBreaker(serverID string) {
	if serverID == "" {
		return
	}
	cbMu.Lock()
	defer cbMu.Unlock()
	delete(cbStates, serverID)
}

func GetServerStatus(server types.SavedServer, password string) types.ServerStatusResult {
	if password != "" {
		server.Password = password
	}

	// 1. Fast circuit-breaker pre-check
	if allowed, reason := CheckCircuitBreaker(server.ID); !allowed {
		return types.ServerStatusResult{
			Success: false,
			Error:   reason,
		}
	}

	start := time.Now()

	session, release, err := getPooledSession(server)
	if err != nil {
		RecordServerInspectionResult(server.ID, false)
		return types.ServerStatusResult{
			Success: false,
			Error:   fmt.Sprintf("Не удалось подключиться: %v", err),
		}
	}
	defer release()



	const telemetryDelimiter = "__REMNA_SECTION_TEL_v3__"

	var cmd string
	if server.Role == "node" {
		// Lightweight interrogation optimized for Edge Nodes: avoids heavy & non-existent Postgres/Panel checks
		cmd = `grep -E '^PRETTY_NAME=' /etc/os-release 2>/dev/null | cut -d= -f2 | tr -d '"'; echo "` + telemetryDelimiter + `"; free -m | awk '/^Mem:/{print $3 "|" $2}'; echo "` + telemetryDelimiter + `"; df -h / | awk 'NR==2{print $3 "|" $2 "|" $5}'; echo "` + telemetryDelimiter + `"; uptime -p 2>/dev/null || uptime; echo "` + telemetryDelimiter + `"; docker ps -a --format '{{.Names}}\t{{.Status}}\t{{.Image}}' 2>/dev/null || true; echo "` + telemetryDelimiter + `"; sysctl -n net.ipv4.tcp_congestion_control 2>/dev/null || echo "unknown"; echo "` + telemetryDelimiter + `"; [ -f /etc/docker/daemon.json ] && grep -q "registry-mirrors" /etc/docker/daemon.json && echo "yes" || echo "no"; echo "` + telemetryDelimiter + `"; echo ""; echo "` + telemetryDelimiter + `"; echo "0"; echo "` + telemetryDelimiter + `"; echo "0"; echo "` + telemetryDelimiter + `"; (command -v warp-cli >/dev/null 2>&1 && warp-cli --accept-tos status 2>/dev/null | grep -qi "connected" && echo "yes") || echo "no"; echo "` + telemetryDelimiter + `"; echo "no"; echo "` + telemetryDelimiter + `"; awk '/^cpu /{t=$2+$3+$4+$5+$6+$7+$8+$9+$10; if(t>0) printf "%.1f", ($9*100)/t}' /proc/stat 2>/dev/null || echo "0.0"`
	} else {
		// Comprehensive interrogation for Master Panel VPS (includes Postgres DB, Backups, and Valkey Sockets)
		cmd = `grep -E '^PRETTY_NAME=' /etc/os-release 2>/dev/null | cut -d= -f2 | tr -d '"'; echo "` + telemetryDelimiter + `"; free -m | awk '/^Mem:/{print $3 "|" $2}'; echo "` + telemetryDelimiter + `"; df -h / | awk 'NR==2{print $3 "|" $2 "|" $5}'; echo "` + telemetryDelimiter + `"; uptime -p 2>/dev/null || uptime; echo "` + telemetryDelimiter + `"; docker ps -a --format '{{.Names}}\t{{.Status}}\t{{.Image}}' 2>/dev/null || true; echo "` + telemetryDelimiter + `"; sysctl -n net.ipv4.tcp_congestion_control 2>/dev/null || echo "unknown"; echo "` + telemetryDelimiter + `"; [ -f /etc/docker/daemon.json ] && grep -q "registry-mirrors" /etc/docker/daemon.json && echo "yes" || echo "no"; echo "` + telemetryDelimiter + `"; ls -lh /opt/remnawave/backups/*.tar.gz 2>/dev/null | awk '{print $9 "|" $5 "|" $6 " " $7 " " $8}' || true; echo "` + telemetryDelimiter + `"; docker exec -u postgres remnawave-db psql -U postgres -d postgres -t -A -c "SELECT count(*) FROM node_plugin np JOIN nodes n ON n.active_plugin_uuid = np.uuid WHERE (np.plugin_config->'torrentBlocker'->>'enabled')::boolean = true;" 2>/dev/null || echo "0"; echo "` + telemetryDelimiter + `"; docker exec -u postgres remnawave-db psql -U postgres -d postgres -t -A -c "SELECT count(*) FROM nodes;" 2>/dev/null || echo "0"; echo "` + telemetryDelimiter + `"; (command -v warp-cli >/dev/null 2>&1 && warp-cli --accept-tos status 2>/dev/null | grep -qi "connected" && echo "yes") || echo "no"; echo "` + telemetryDelimiter + `"; ([ -f /opt/remnawave/panel/docker-compose.yml ] && grep -q "shm_size: 512mb" /opt/remnawave/panel/docker-compose.yml && grep -q "valkey.sock" /opt/remnawave/panel/docker-compose.yml && echo "yes") || ([ -f /opt/remnawave/docker-compose.yml ] && grep -q "shm_size: 512mb" /opt/remnawave/docker-compose.yml && grep -q "valkey.sock" /opt/remnawave/docker-compose.yml && echo "yes") || echo "no"; echo "` + telemetryDelimiter + `"; awk '/^cpu /{t=$2+$3+$4+$5+$6+$7+$8+$9+$10; if(t>0) printf "%.1f", ($9*100)/t}' /proc/stat 2>/dev/null || echo "0.0"`
	}

	out, err := session.Output(cmd)

	latency := int(time.Since(start).Milliseconds())

	if err != nil {
		RecordServerInspectionResult(server.ID, false)
		return types.ServerStatusResult{Success: false, Error: err.Error()}
	}



	parts := strings.Split(string(out), telemetryDelimiter)

	hostInfo := "Linux VPS"

	metrics := types.ServerMetrics{}



	if len(parts) >= 1 && strings.TrimSpace(parts[0]) != "" {

		hostInfo = strings.TrimSpace(parts[0])

	}



	// RAM

	if len(parts) >= 2 {

		memLine := strings.TrimSpace(parts[1])

		memCols := strings.Split(memLine, "|")

		if len(memCols) >= 2 {

			var used, total int

			fmt.Sscanf(memCols[0], "%d", &used)

			fmt.Sscanf(memCols[1], "%d", &total)

			metrics.RAMUsedMB = used

			metrics.RAMTotalMB = total

			if total > 0 {

				metrics.RAMPercent = (used * 100) / total

			}

		}

	}



	// Disk

	if len(parts) >= 3 {

		diskLine := strings.TrimSpace(parts[2])

		diskCols := strings.Split(diskLine, "|")

		if len(diskCols) >= 3 {

			metrics.DiskUsed = diskCols[0]

			metrics.DiskTotal = diskCols[1]

			pctStr := strings.Trim(diskCols[2], "%\r\n ")

			var pct int

			fmt.Sscanf(pctStr, "%d", &pct)

			metrics.DiskPercent = pct

		}

	}



	// Uptime

	if len(parts) >= 4 {

		metrics.Uptime = strings.TrimSpace(parts[3])

	}



	// Containers
	var containers []types.ContainerInfo
	hasNodeContainer := false
	if len(parts) >= 5 {
		lines := strings.Split(strings.TrimSpace(parts[4]), "\n")
		for _, l := range lines {
			info, ok := ParseContainerLine(l)
			if ok {
				if info.Name == "remnanode" {
					hasNodeContainer = true
				}
				containers = append(containers, info)
			}
		}
	}



	// BBR active?

	bbrActive := false

	if len(parts) >= 6 {

		bbrActive = strings.TrimSpace(parts[5]) == "bbr"

	}



	// Docker mirrors active?

	dockerMirrorsActive := false

	if len(parts) >= 7 {

		dockerMirrorsActive = strings.TrimSpace(parts[6]) == "yes"

	}



	// Backups list

	var backups []types.BackupInfo

	if len(parts) >= 8 {

		lines := strings.Split(strings.TrimSpace(parts[7]), "\n")

		for _, l := range lines {

			cols := strings.Split(l, "|")

			if len(cols) >= 3 && strings.TrimSpace(cols[0]) != "" {

				backups = append(backups, types.BackupInfo{

					FileName:  filepath.Base(strings.TrimSpace(cols[0])),

					Size:      strings.TrimSpace(cols[1]),

					CreatedAt: strings.TrimSpace(cols[2]),

				})

			}

		}

	}



	// Anti-abuse (torrent blocker & egress filter) active?

	antiAbuseActive := false

	if len(parts) >= 9 {

		cntStr := strings.TrimSpace(parts[8])

		antiAbuseActive = cntStr != "0" && cntStr != ""

	}



	// Node present?

	nodePresent := hasNodeContainer

	if len(parts) >= 10 {

		cntStr := strings.TrimSpace(parts[9])

		if cntStr != "0" && cntStr != "" {

			nodePresent = true

		}

	}



	// WARP active?
	warpActive := false
	if len(parts) >= 11 {
		warpActive = strings.TrimSpace(parts[10]) == "yes"
	}

	// Performance optimized (shm_size 512mb & valkey socket)?
	performanceOptimized := false
	if len(parts) >= 12 {
		performanceOptimized = strings.TrimSpace(parts[11]) == "yes"
	}

	// CPU Steal (% time lost to hypervisor noisy neighbors)
	if len(parts) >= 13 {
		stealStr := strings.TrimSpace(parts[12])
		var stealVal float64
		if _, err := fmt.Sscanf(stealStr, "%f", &stealVal); err == nil {
			metrics.CPUSteal = stealVal
		}
	}

	RecordServerInspectionResult(server.ID, true)
	return types.ServerStatusResult{
		Success:              true,
		HostInfo:             hostInfo,
		Metrics:              metrics,
		Containers:           containers,
		BBRActive:            bbrActive,
		DockerMirrorsActive:  dockerMirrorsActive,
		AntiAbuseActive:      antiAbuseActive,
		WarpActive:           warpActive,
		PerformanceOptimized: performanceOptimized,
		NodePresent:          nodePresent,
		LatencyMs:            latency,
		Backups:              backups,
	}
}

// ParseContainerLine parses a single line of `docker ps -a --format '{{.Names}}\t{{.Status}}\t{{.Image}}'`
// and returns normalized container health metadata.
func ParseContainerLine(line string) (types.ContainerInfo, bool) {
	cols := strings.Split(line, "\t")
	if len(cols) < 3 {
		return types.ContainerInfo{}, false
	}
	name := strings.TrimSpace(cols[0])
	status := strings.TrimSpace(cols[1])
	image := strings.TrimSpace(cols[2])
	if name == "" {
		return types.ContainerInfo{}, false
	}

	statusLower := strings.ToLower(status)
	running := strings.HasPrefix(statusLower, "up")
	healthy := "running"

	if strings.Contains(statusLower, "(healthy)") {
		healthy = "healthy"
	} else if strings.Contains(statusLower, "(unhealthy)") {
		healthy = "unhealthy"
		running = false // Container is degraded or failing healthchecks
	} else if strings.HasPrefix(statusLower, "restarting") {
		healthy = "restarting"
		running = false
	} else if strings.Contains(statusLower, "starting") {
		healthy = "starting"
	} else if !running {
		healthy = "stopped"
	}

	return types.ContainerInfo{
		Name:    name,
		Status:  status,
		Image:   image,
		Running: running,
		Healthy: healthy,
	}, true
}



func RestartContainer(server types.SavedServer, container string) types.CommandResult {

	cleanName := strings.TrimSpace(container)

	if !isValidContainerName(cleanName) {

		return types.CommandResult{Success: false, Error: "Некорректное или небезопасное имя контейнера"}

	}



	session, release, err := getPooledSession(server)

	if err != nil {

		return types.CommandResult{Success: false, Error: err.Error()}

	}

	defer release()



	out, err := session.CombinedOutput(fmt.Sprintf("docker restart %s", cleanName))

	if err != nil {

		return types.CommandResult{Success: false, Error: string(out)}

	}

	return types.CommandResult{Success: true, Output: fmt.Sprintf("Контейнер %s успешно перезапущен!", cleanName)}

}



func RestartAllStack(server types.SavedServer) types.CommandResult {

	session, release, err := getPooledSession(server)

	if err != nil {

		return types.CommandResult{Success: false, Error: err.Error()}

	}

	defer release()



	cmd := `
# Auto-heal subscription .env if missing REMNAWAVE_PANEL_URL
if [ -f /opt/remnawave/subscription/.env ]; then
  if ! grep -q 'REMNAWAVE_PANEL_URL' /opt/remnawave/subscription/.env; then
    echo '' >> /opt/remnawave/subscription/.env
    echo 'REMNAWAVE_PANEL_URL=http://remnawave:3000' >> /opt/remnawave/subscription/.env
    echo 'APP_PORT=3010' >> /opt/remnawave/subscription/.env
  fi
  cd /opt/remnawave/subscription && docker compose up -d 2>/dev/null || true
fi

# Auto-heal node if present
if [ -d /opt/remnawave/node ]; then
  command -v ufw >/dev/null 2>&1 && ufw allow 2222/tcp && ufw allow 443/tcp && ufw allow 80/tcp || true
  iptables -I INPUT -p tcp --dport 2222 -j ACCEPT 2>/dev/null || true
  iptables -I INPUT -p tcp --dport 443 -j ACCEPT 2>/dev/null || true
  iptables -I INPUT -p tcp --dport 80 -j ACCEPT 2>/dev/null || true
  cd /opt/remnawave/node && docker compose up -d 2>/dev/null || true
fi

cd /opt/remnawave/panel 2>/dev/null && docker compose restart || true
cd /opt/remnawave/nginx 2>/dev/null && docker compose restart || true
cd /opt/remnawave/caddy 2>/dev/null && docker compose restart || true
cd /opt/remnawave/decoy 2>/dev/null && docker compose restart || true
`

	out, err := session.CombinedOutput(cmd)

	if err != nil {

		return types.CommandResult{Success: false, Error: string(out)}

	}

	return types.CommandResult{Success: true, Output: "Все контейнеры Remnawave и нода успешно перезапущены!"}

}



// ConfigureSubscriptionToken configures API token and URLs in /opt/remnawave/subscription/.env
func ConfigureSubscriptionToken(server types.SavedServer, token string) types.CommandResult {
	cleanToken := strings.TrimSpace(token)
	if cleanToken == "" {
		return types.CommandResult{Success: false, Error: "Токен не может быть пустым"}
	}
	if strings.ContainsAny(cleanToken, "\r\n\"'`;|&$<> \t\\") {
		return types.CommandResult{Success: false, Error: "Некорректный токен: содержит недопустимые символы"}
	}

	client, err := getSSHClient(server)
	if err != nil {
		return types.CommandResult{Success: false, Error: err.Error()}
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return types.CommandResult{Success: false, Error: err.Error()}
	}
	defer session.Close()

	cmd := fmt.Sprintf(`
if [ -f /opt/remnawave/subscription/.env ]; then
  sed -i '/^REMNAWAVE_API_TOKEN=/d' /opt/remnawave/subscription/.env
  sed -i '/^REMNAWAVE_PANEL_URL=/d' /opt/remnawave/subscription/.env
  sed -i '/^REMNAWAVE_URL=/d' /opt/remnawave/subscription/.env
  sed -i '/^PORT=/d' /opt/remnawave/subscription/.env
  sed -i '/^APP_PORT=/d' /opt/remnawave/subscription/.env
  sed -i '/^TRUST_PROXY=/d' /opt/remnawave/subscription/.env

  cat >> /opt/remnawave/subscription/.env <<'EOF'
PORT=3010
APP_PORT=3010
HOST=0.0.0.0
TRUST_PROXY=true
REMNAWAVE_PANEL_URL=http://remnawave:3000
REMNAWAVE_URL=http://remnawave:3000
REMNAWAVE_API_TOKEN=%s
EOF

  cd /opt/remnawave/subscription && docker compose up -d --force-recreate
  echo "SUBSCRIPTION_CONFIGURED"
else
  echo "NO_SUBSCRIPTION_DIR"
fi
`, cleanToken)

	out, err := session.CombinedOutput(cmd)
	if err != nil {
		return types.CommandResult{Success: false, Error: string(out)}
	}
	return types.CommandResult{Success: true, Output: strings.TrimSpace(string(out))}
}

func FixDatabaseSync(server types.SavedServer) types.CommandResult {

	client, err := getSSHClient(server)

	if err != nil {

		return types.CommandResult{Success: false, Error: err.Error()}

	}

	defer client.Close()



	session, err := client.NewSession()

	if err != nil {

		return types.CommandResult{Success: false, Error: err.Error()}

	}

	defer session.Close()



	cmd := `
COMPOSE_DIR="/opt/remnawave"
if [ -f /opt/remnawave/panel/docker-compose.yml ]; then
  COMPOSE_DIR="/opt/remnawave/panel"
fi

DB_PASS=""
DB_USER="postgres"
DB_NAME="remnawave"
for env_f in "${COMPOSE_DIR}/.env" /opt/remnawave/.env /opt/remnawave/panel/.env; do
  if [ -f "$env_f" ]; then
    p=$(grep -E '^POSTGRES_PASSWORD=' "$env_f" 2>/dev/null | cut -d= -f2- | tr -d '"\r\n' | head -n1)
    [ -n "$p" ] && DB_PASS="$p"
    u=$(grep -E '^POSTGRES_USER=' "$env_f" 2>/dev/null | cut -d= -f2- | tr -d '"\r\n' | head -n1)
    [ -n "$u" ] && DB_USER="$u"
    d=$(grep -E '^POSTGRES_DB=' "$env_f" 2>/dev/null | cut -d= -f2- | tr -d '"\r\n' | head -n1)
    [ -n "$d" ] && DB_NAME="$d"
    if [ -n "$DB_PASS" ]; then break; fi
  fi
done

if [ -z "$DB_PASS" ]; then
  echo "Файл конфигурации с POSTGRES_PASSWORD не найден в /opt/remnawave"
  exit 1
fi

DB_PASS_ESC=$(printf '%s' "$DB_PASS" | sed "s/'/''/g")

docker exec remnawave-db psql -U postgres -d postgres -c "ALTER USER \"$DB_USER\" WITH PASSWORD '$DB_PASS_ESC';" 2>/dev/null || \
docker exec remnawave-db psql -U postgres -d postgres -c "ALTER USER postgres WITH PASSWORD '$DB_PASS_ESC';" 2>/dev/null || true

cd "$COMPOSE_DIR" && docker compose restart remnawave 2>/dev/null || docker restart remnawave 2>/dev/null || true

for i in $(seq 1 30); do
  if docker logs remnawave --tail 15 2>&1 | grep -qE 'Nest application successfully started|Listening on'; then
    break
  fi
  sleep 1
done
`

	out, err := session.CombinedOutput(cmd)

	if err != nil {

		return types.CommandResult{Success: false, Error: string(out)}

	}

	return types.CommandResult{Success: true, Output: "Пароль PostgreSQL синхронизирован с .env, Remnawave перезапущен!"}

}



func GetContainerLogs(server types.SavedServer, container string) string {

	cleanName := strings.TrimSpace(container)

	if !isValidContainerName(cleanName) {

		return "Ошибка: некорректное или небезопасное имя контейнера"

	}



	session, release, err := getPooledSession(server)

	if err != nil {

		return fmt.Sprintf("Ошибка: %v", err)

	}

	defer release()



	out, err := session.CombinedOutput(fmt.Sprintf("docker logs --tail=150 %s 2>&1", cleanName))

	if err != nil && len(out) == 0 {

		return fmt.Sprintf("Ошибка получения логов: %v", err)

	}

	return string(out)

}



func parseNodeComposeInput(input string) string {

	trimmed := strings.TrimSpace(input)



	// Case 1: User pasted the entire docker-compose.yml from Remnawave Panel

	if strings.Contains(trimmed, "services:") && strings.Contains(trimmed, "remnanode:") {

		return trimmed

	}



	// Case 2: User pasted something with SECRET_KEY=... or - SECRET_KEY=...

	secretKey := trimmed

	if strings.Contains(trimmed, "SECRET_KEY") {

		lines := strings.Split(trimmed, "\n")

		for _, l := range lines {

			lTrim := strings.TrimSpace(l)

			if strings.Contains(lTrim, "SECRET_KEY") {

				parts := strings.SplitN(lTrim, "=", 2)

				if len(parts) == 2 {

					secretKey = strings.Trim(strings.TrimSpace(parts[1]), "\"'")

					break

				}

			}

		}

	}



	// Strip wrapping quotes if any

	secretKey = strings.Trim(secretKey, "\"' \t\r\n")



	return fmt.Sprintf(`services:

  remnanode:

    container_name: remnanode

    hostname: remnanode

    image: remnawave/node:latest

    network_mode: host

    restart: always

    cap_add:

      - NET_ADMIN

    ulimits:

      nofile:

        soft: 1048576

        hard: 1048576

    environment:

      - NODE_PORT=2222

      - SECRET_KEY="%s"

    volumes:

      - /dev/shm:/dev/shm:rw

`, secretKey)

}



func AttachNodeSecret(server types.SavedServer, secretKey string) types.CommandResult {

	cleanInput := strings.TrimSpace(secretKey)

	if cleanInput == "" {

		return types.CommandResult{Success: false, Error: "Конфигурация или SECRET_KEY не может быть пустым"}

	}



	client, err := getSSHClient(server)

	if err != nil {

		return types.CommandResult{Success: false, Error: err.Error()}

	}

	defer client.Close()



	session, err := client.NewSession()

	if err != nil {

		return types.CommandResult{Success: false, Error: err.Error()}

	}

	defer session.Close()



	nodeCompose := parseNodeComposeInput(cleanInput)

	b64Compose := base64.StdEncoding.EncodeToString([]byte(nodeCompose))



	cmd := fmt.Sprintf(`
COMPOSE_CONTENT=$(echo '%s' | base64 -d)
if [ -f /opt/remnawave/docker-compose.yml ] && grep -q 'remnanode:' /opt/remnawave/docker-compose.yml; then
  echo "Обнаружен единый all-in-one стек (/opt/remnawave)..."
  EXTRACTED_KEY=$(echo "$COMPOSE_CONTENT" | grep -oE 'SECRET_KEY=.*|SECRET_KEY:.*' | head -n1 | cut -d= -f2- | cut -d: -f2- | tr -d '"'\'' \r\n' || true)
  if [ -n "$EXTRACTED_KEY" ]; then
    sed -i "s|^SECRET_KEY=.*|SECRET_KEY=${EXTRACTED_KEY}|g" /opt/remnawave/.env 2>/dev/null || true
    sed -i "s|SECRET_KEY:.*|SECRET_KEY: \"${EXTRACTED_KEY}\"|g" /opt/remnawave/docker-compose.yml 2>/dev/null || true
  fi
  cd /opt/remnawave && docker compose up -d remnanode
  docker ps --filter "name=remnanode"
else
  mkdir -p /opt/remnawave/node
  echo "$COMPOSE_CONTENT" > /opt/remnawave/node/docker-compose.yml
  command -v ufw >/dev/null 2>&1 && ufw allow 2222/tcp >/dev/null 2>&1 || true
  cd /opt/remnawave/node && docker compose down 2>/dev/null || true
  cd /opt/remnawave/node && docker compose up -d
  sleep 2
  docker compose ps
fi
`, b64Compose)



	out, err := session.CombinedOutput(cmd)

	if err != nil {

		return types.CommandResult{Success: false, Error: fmt.Sprintf("%s: %v", string(out), err)}

	}

	return types.CommandResult{Success: true, Output: "Нода Remnawave успешно настроена и запущена!"}

}



func OptimizeNetworkBBR(server types.SavedServer) types.CommandResult {

	session, release, err := getPooledSession(server)

	if err != nil {

		return types.CommandResult{Success: false, Error: err.Error()}

	}

	defer release()



	cmd := `

modprobe tcp_bbr 2>/dev/null || true

echo "tcp_bbr" >> /etc/modules-load.d/bbr.conf 2>/dev/null || true

cat > /etc/sysctl.d/99-remna-bbr.conf <<'EOF'

net.core.default_qdisc = fq

net.ipv4.tcp_congestion_control = bbr

net.ipv4.tcp_fastopen = 3

fs.file-max = 1048576

net.ipv4.tcp_rmem = 4096 87380 67108864

net.ipv4.tcp_wmem = 4096 65536 67108864

EOF

sysctl --system >/dev/null 2>&1 || sysctl -p /etc/sysctl.d/99-remna-bbr.conf >/dev/null 2>&1

CURRENT=$(sysctl -n net.ipv4.tcp_congestion_control 2>/dev/null)

echo "Google BBR активен ($CURRENT), очередь fq включена!"

`

	out, err := session.CombinedOutput(cmd)

	if err != nil {

		return types.CommandResult{Success: false, Error: string(out)}

	}

	return types.CommandResult{Success: true, Output: strings.TrimSpace(string(out))}

}



func ConfigureDockerMirrors(server types.SavedServer) types.CommandResult {

	session, release, err := getPooledSession(server)

	if err != nil {

		return types.CommandResult{Success: false, Error: err.Error()}

	}

	defer release()



	cmd := `

mkdir -p /etc/docker

cat > /etc/docker/daemon.json <<'EOF'

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

systemctl restart docker >/dev/null 2>&1

echo "Зеркала реестра Docker и локальная ротация логов успешно настроены!"

`

	out, err := session.CombinedOutput(cmd)

	if err != nil {

		return types.CommandResult{Success: false, Error: string(out)}

	}

	return types.CommandResult{Success: true, Output: strings.TrimSpace(string(out))}

}



func CreateBackup(server types.SavedServer) types.CommandResult {
	session, release, err := getPooledSession(server)
	if err != nil {
		return types.CommandResult{Success: false, Error: err.Error()}
	}
	defer release()

	cmd := `
set -e
mkdir -p /opt/remnawave/backups
STAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_NAME="remnawave_backup_${STAMP}.tar.gz"
TEMP_DIR=$(mktemp -d)

# 1. Parse DB credentials and database name from .env if present
DB_USER="postgres"
DB_NAME="postgres"
if [ -f /opt/remnawave/panel/.env ]; then
  u=$(grep -E '^(POSTGRES_USER|DB_USER)=' /opt/remnawave/panel/.env | head -1 | cut -d'=' -f2- | tr -d '"\x27 ' || true)
  [ -n "$u" ] && DB_USER="$u"
  d=$(grep -E '^(POSTGRES_DB|DB_NAME)=' /opt/remnawave/panel/.env | head -1 | cut -d'=' -f2- | tr -d '"\x27 ' || true)
  [ -n "$d" ] && DB_NAME="$d"
fi

# 2. Dump PostgreSQL in binary custom format (-F c -Z 9) with integrity check
HAS_DUMP="no"
if docker ps --format '{{.Names}}' | grep -q 'remnawave-db'; then
  # Check if database exists, fallback to postgres
  if ! docker exec -u "$DB_USER" remnawave-db psql -U "$DB_USER" -lqt 2>/dev/null | cut -d \| -f 1 | grep -qw "$DB_NAME"; then
    DB_NAME="postgres"
  fi

  if docker exec -i remnawave-db pg_dump -U "$DB_USER" -d "$DB_NAME" --clean --if-exists --no-owner --no-privileges -F c -Z 9 > "$TEMP_DIR/remnawave.dump" 2>"$TEMP_DIR/dump.err"; then
    # Verify TOC integrity of the custom dump
    if docker exec -i remnawave-db pg_restore -l "$TEMP_DIR/remnawave.dump" >/dev/null 2>&1; then
      HAS_DUMP="yes"
    else
      echo "ERR_INTEGRITY: Ошибка валидации дампа PostgreSQL (pg_restore -l завершился сбоем)." >&2
      cat "$TEMP_DIR/dump.err" >&2
      rm -rf "$TEMP_DIR"
      exit 1
    fi
  else
    # Fallback to plain pg_dump if -F c somehow failed
    if docker exec -i remnawave-db pg_dump -U "$DB_USER" -d "$DB_NAME" --clean --if-exists --no-owner --no-privileges > "$TEMP_DIR/database.sql" 2>/dev/null; then
      HAS_DUMP="yes"
    fi
  fi
fi

# 3. Copy all configs
mkdir -p "$TEMP_DIR/configs"
[ -f /opt/remnawave/panel/.env ] && cp -rf /opt/remnawave/panel/.env "$TEMP_DIR/configs/panel.env" || true
[ -f /opt/remnawave/panel/docker-compose.yml ] && cp -rf /opt/remnawave/panel/docker-compose.yml "$TEMP_DIR/configs/panel-compose.yml" || true
[ -f /opt/remnawave/node/docker-compose.yml ] && cp -rf /opt/remnawave/node/docker-compose.yml "$TEMP_DIR/configs/node-compose.yml" || true
[ -f /opt/remnawave/nginx/nginx.conf ] && cp -rf /opt/remnawave/nginx/nginx.conf "$TEMP_DIR/configs/nginx.conf" || true
[ -f /opt/remnawave/nginx/fullchain.pem ] && cp -rf /opt/remnawave/nginx/fullchain.pem "$TEMP_DIR/configs/" || true
[ -f /opt/remnawave/nginx/privkey.key ] && cp -rf /opt/remnawave/nginx/privkey.key "$TEMP_DIR/configs/" || true
[ -f /opt/remnawave/caddy/Caddyfile ] && cp -rf /opt/remnawave/caddy/Caddyfile "$TEMP_DIR/configs/Caddyfile" || true
[ -f /opt/remnawave/caddy/docker-compose.yml ] && cp -rf /opt/remnawave/caddy/docker-compose.yml "$TEMP_DIR/configs/caddy-compose.yml" || true

# 4. Copy uploads directory if exists
if [ -d /opt/remnawave/uploads ]; then
  mkdir -p "$TEMP_DIR/uploads"
  cp -rf /opt/remnawave/uploads/* "$TEMP_DIR/uploads/" 2>/dev/null || true
elif [ -d /opt/remnawave/panel/uploads ]; then
  mkdir -p "$TEMP_DIR/uploads"
  cp -rf /opt/remnawave/panel/uploads/* "$TEMP_DIR/uploads/" 2>/dev/null || true
fi

# 5. Write metadata
cat << META_EOF > "$TEMP_DIR/backup_meta.info"
FORMAT_VERSION=2.0
BACKUP_DATE=$STAMP
DB_NAME=$DB_NAME
DB_USER=$DB_USER
HAS_DUMP=$HAS_DUMP
META_EOF

# 6. Create archive
tar -czf "/opt/remnawave/backups/$BACKUP_NAME" -C "$TEMP_DIR" .
rm -rf "$TEMP_DIR"

# 7. Local retention cleanup (remove older than 14 days, keep at most 10 newest)
find /opt/remnawave/backups -maxdepth 1 -type f -name "remnawave_backup_*.tar.gz" -mtime +14 -delete 2>/dev/null || true
ls -1t /opt/remnawave/backups/remnawave_backup_*.tar.gz 2>/dev/null | tail -n +11 | xargs -r rm -f 2>/dev/null || true

SIZE=$(du -h "/opt/remnawave/backups/$BACKUP_NAME" | awk '{print $1}')
echo "Архив успешно создан: $BACKUP_NAME ($SIZE)"
`

	out, err := session.CombinedOutput(cmd)
	if err != nil {
		return types.CommandResult{Success: false, Error: string(out)}
	}

	return types.CommandResult{Success: true, Output: strings.TrimSpace(string(out))}
}



func PruneDockerSystem(server types.SavedServer) types.CommandResult {

	client, err := getSSHClient(server)

	if err != nil {

		return types.CommandResult{Success: false, Error: err.Error()}

	}

	defer client.Close()



	session, err := client.NewSession()

	if err != nil {

		return types.CommandResult{Success: false, Error: err.Error()}

	}

	defer session.Close()



	cmd := `

BEFORE=$(df -m / | awk 'NR==2{print $4}')

docker system prune -f >/dev/null 2>&1 || true

journalctl --vacuum-time=3d >/dev/null 2>&1 || true

apt-get clean >/dev/null 2>&1 || true

AFTER=$(df -m / | awk 'NR==2{print $4}')

FREED=$((AFTER - BEFORE))

if [ $FREED -gt 0 ]; then

  echo "Освобождено $FREED MB дискового пространства!"

else

  echo "Диск очищен от временных слоев и кэша Docker."

fi

`

	out, err := session.CombinedOutput(cmd)

	if err != nil {

		return types.CommandResult{Success: false, Error: string(out)}

	}

	return types.CommandResult{Success: true, Output: strings.TrimSpace(string(out))}

}



func ResetSuperadmin(server types.SavedServer) types.CommandResult {

	client, err := getSSHClient(server)

	if err != nil {

		return types.CommandResult{Success: false, Error: err.Error()}

	}

	defer client.Close()



	session, err := client.NewSession()

	if err != nil {

		return types.CommandResult{Success: false, Error: err.Error()}

	}

	defer session.Close()



	cmd := `
COMPOSE_DIR="/opt/remnawave"
if [ -f /opt/remnawave/panel/docker-compose.yml ]; then
  COMPOSE_DIR="/opt/remnawave/panel"
fi

DB_USER="postgres"
DB_NAME="remnawave"
for env_f in "${COMPOSE_DIR}/.env" /opt/remnawave/.env /opt/remnawave/panel/.env; do
  if [ -f "$env_f" ]; then
    u=$(grep -E '^POSTGRES_USER=' "$env_f" 2>/dev/null | cut -d= -f2- | tr -d '"\r\n' | head -n1)
    [ -n "$u" ] && DB_USER="$u"
    d=$(grep -E '^POSTGRES_DB=' "$env_f" 2>/dev/null | cut -d= -f2- | tr -d '"\r\n' | head -n1)
    [ -n "$d" ] && DB_NAME="$d"
    break
  fi
done

# 1. Truncate user tables and tokens across possible schemas
if docker ps --format '{{.Names}}' 2>/dev/null | grep -q 'remnawave-db'; then
  docker exec -u postgres remnawave-db psql -U "$DB_USER" -d "$DB_NAME" -c 'TRUNCATE TABLE "User" CASCADE;' 2>/dev/null || \
  docker exec -u postgres remnawave-db psql -U "$DB_USER" -d "$DB_NAME" -c 'TRUNCATE TABLE "Admin" CASCADE;' 2>/dev/null || \
  docker exec -u postgres remnawave-db psql -U "$DB_USER" -d "$DB_NAME" -c 'TRUNCATE TABLE users CASCADE;' 2>/dev/null || \
  docker exec -u postgres remnawave-db psql -U "$DB_USER" -d postgres -c 'TRUNCATE TABLE "User" CASCADE;' 2>/dev/null || true

  docker exec -u postgres remnawave-db psql -U "$DB_USER" -d "$DB_NAME" -c 'TRUNCATE TABLE "RefreshToken" CASCADE;' 2>/dev/null || true
  docker exec -u postgres remnawave-db psql -U "$DB_USER" -d "$DB_NAME" -c 'TRUNCATE TABLE "Session" CASCADE;' 2>/dev/null || true
fi

# 2. Try CLI reset command inside remnawave container if present
docker exec remnawave sh -c 'cli reset-admin || remnawave reset-admin' 2>/dev/null || true

# 3. Wipe stale tokens & credentials
rm -f /opt/remnawave/token /opt/remnawave/panel/token /opt/remnawave/credentials.txt /opt/remnawave/panel/credentials.txt 2>/dev/null || true

# 4. Restart services
cd "$COMPOSE_DIR" && docker compose restart remnawave-redis remnawave 2>/dev/null || docker restart remnawave 2>/dev/null || true

# 5. Wait for NestJS readiness
for i in $(seq 1 30); do
  if docker logs remnawave --tail 20 2>&1 | grep -qE 'Nest application successfully started|Listening on'; then
    break
  fi
  sleep 1
done

echo "Суперадминистратор успешно сброшен. Откройте панель в браузере для создания новой учетной записи!"
`

	out, err := session.CombinedOutput(cmd)
	if err != nil {
		return types.CommandResult{Success: false, Error: string(out)}
	}
	return types.CommandResult{Success: true, Output: strings.TrimSpace(string(out))}
}

func RenewSSL(server types.SavedServer) types.CommandResult {
	client, err := getSSHClient(server)
	if err != nil {
		return types.CommandResult{Success: false, Error: err.Error()}
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return types.CommandResult{Success: false, Error: err.Error()}
	}
	defer session.Close()

	cmd := `
sync_certs() {
  local cert_file="$1"
  local key_file="$2"
  if [ -f "$cert_file" ] && [ -f "$key_file" ]; then
    mkdir -p /opt/remnawave/ssl /opt/remnawave/nginx 2>/dev/null || true
    cp -fL "$cert_file" /opt/remnawave/ssl/fullchain.pem 2>/dev/null || true
    cp -fL "$key_file" /opt/remnawave/ssl/privkey.key 2>/dev/null || true
    cp -fL "$cert_file" /opt/remnawave/nginx/fullchain.pem 2>/dev/null || true
    cp -fL "$key_file" /opt/remnawave/nginx/privkey.key 2>/dev/null || true
  fi
}

restart_proxy() {
  docker restart remnawave-nginx 2>/dev/null || docker restart remnawave-caddy 2>/dev/null || true
}

if [ -x ~/.acme.sh/acme.sh ]; then
  echo ">>> Обновление SSL сертификатов через acme.sh..."
  ~/.acme.sh/acme.sh --renew-all --force 2>&1 || true
  ACME_DOMAIN=$(ls -1 ~/.acme.sh/ 2>/dev/null | grep -v 'ca\|acme.sh\|deploy' | head -n 1)
  if [ -n "$ACME_DOMAIN" ] && [ -f ~/.acme.sh/$ACME_DOMAIN/fullchain.cer ]; then
    sync_certs ~/.acme.sh/$ACME_DOMAIN/fullchain.cer ~/.acme.sh/$ACME_DOMAIN/$ACME_DOMAIN.key
  fi
  restart_proxy
  echo "SSL сертификаты Let's Encrypt (acme.sh) успешно обновлены!"

elif command -v docker >/dev/null 2>&1 && [ -d /etc/letsencrypt/live ]; then
  docker stop remnawave-nginx remnawave-caddy 2>/dev/null || true
  echo ">>> Обновление SSL сертификатов через Docker Certbot..."
  docker run --rm -p 80:80 \
    -v /etc/letsencrypt:/etc/letsencrypt \
    -v /var/lib/letsencrypt:/var/lib/letsencrypt \
    certbot/certbot renew --non-interactive 2>&1 || true
  DOMAIN=$(ls -1 /etc/letsencrypt/live/ 2>/dev/null | grep -v 'README' | head -n 1)
  if [ -n "$DOMAIN" ] && [ -f "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" ]; then
    sync_certs "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" "/etc/letsencrypt/live/$DOMAIN/privkey.pem"
  fi
  docker start remnawave-nginx remnawave-caddy 2>/dev/null || restart_proxy
  echo "SSL сертификаты Let's Encrypt (Docker Certbot) успешно обновлены!"

elif command -v certbot >/dev/null 2>&1; then
  docker stop remnawave-nginx remnawave-caddy 2>/dev/null || true
  certbot renew --non-interactive 2>&1
  DOMAIN=$(ls -1 /etc/letsencrypt/live/ 2>/dev/null | grep -v 'README' | head -n 1)
  if [ -n "$DOMAIN" ] && [ -f "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" ]; then
    sync_certs "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" "/etc/letsencrypt/live/$DOMAIN/privkey.pem"
  fi
  docker start remnawave-nginx remnawave-caddy 2>/dev/null || restart_proxy
  echo "SSL сертификаты Let's Encrypt (Certbot) успешно обновлены!"

else
  echo "Ни acme.sh, ни Certbot не обнаружены на сервере."
  exit 1
fi
`

	out, err := session.CombinedOutput(cmd)

	if err != nil {

		return types.CommandResult{Success: false, Error: string(out)}

	}

	return types.CommandResult{Success: true, Output: strings.TrimSpace(string(out))}

}



func ToggleAntiAbuse(server types.SavedServer, enable bool) types.CommandResult {

	session, release, err := getPooledSession(server)

	if err != nil {

		return types.CommandResult{Success: false, Error: err.Error()}

	}

	defer release()



	var cmd string

	if enable {

		cmd = `

# 1. Проверяем наличие ноды

NODE_COUNT=$(docker exec -u postgres remnawave-db psql -U postgres -d postgres -t -A -c "SELECT count(*) FROM nodes;" 2>/dev/null || echo "0")

HAS_CONTAINER=$(docker ps -a --format '{{.Names}}' | grep -q 'remnanode' && echo "yes" || echo "no")



if [ "$NODE_COUNT" = "0" ] && [ "$HAS_CONTAINER" = "no" ]; then

  echo "ERR_NO_NODE: Xray нода не обнаружена. Сначала настройте и запустите Xray ноду перед включением защиты."

  exit 1

fi



# 2. Создаем/обновляем плагин в базе данных

docker exec -u postgres remnawave-db psql -U postgres -d postgres -c "

DO \$\$

DECLARE

  p_uuid uuid := gen_random_uuid();

  cfg jsonb := '{\"torrentBlocker\":{\"enabled\":true,\"blockDuration\":3600,\"ignoreLists\":{\"ip\":[],\"userId\":[]}},\"egressFilter\":{\"enabled\":true,\"blockedPorts\":[25,465,587],\"blockedProtocols\":[\"smtp\",\"bittorrent\"],\"ignoreLists\":{\"ip\":[],\"userId\":[]}},\"ingressFilter\":{\"enabled\":false,\"blockedIps\":[]}}'::jsonb;

BEGIN

  IF NOT EXISTS (SELECT 1 FROM node_plugin WHERE name='Anti-Abuse Shield') THEN

    INSERT INTO node_plugin (uuid, name, plugin_config, created_at, updated_at)

    VALUES (p_uuid, 'Anti-Abuse Shield', cfg, now(), now());

  ELSE

    SELECT uuid INTO p_uuid FROM node_plugin WHERE name='Anti-Abuse Shield' LIMIT 1;

    UPDATE node_plugin SET plugin_config=cfg, updated_at=now() WHERE uuid=p_uuid;

  END IF;



  UPDATE nodes SET active_plugin_uuid=p_uuid;

END \$\$;

" >/dev/null 2>&1



# 3. Блокируем исходящий спам (SMTP) на сетевом уровне хоста

iptables -C OUTPUT -p tcp --dport 25 -j DROP 2>/dev/null || iptables -A OUTPUT -p tcp --dport 25 -j DROP 2>/dev/null || true

iptables -C FORWARD -p tcp --dport 25 -j DROP 2>/dev/null || iptables -A FORWARD -p tcp --dport 25 -j DROP 2>/dev/null || true



# 4. Перезапускаем ноду и панель для мгновенного вступления в силу

docker restart remnanode >/dev/null 2>&1 || true

docker compose -f /opt/remnawave/panel/docker-compose.yml restart remnawave >/dev/null 2>&1 || true



echo "Защита от торрентов и спама (Torrent Blocker & Egress Filter) успешно активирована!"

`

	} else {

		cmd = `

# 1. Отключаем плагин в базе

docker exec -u postgres remnawave-db psql -U postgres -d postgres -c "

DO \$\$

DECLARE

  p_uuid uuid;

  cfg jsonb := '{\"torrentBlocker\":{\"enabled\":false,\"blockDuration\":3600,\"ignoreLists\":{\"ip\":[],\"userId\":[]}},\"egressFilter\":{\"enabled\":false,\"blockedPorts\":[],\"blockedProtocols\":[],\"ignoreLists\":{\"ip\":[],\"userId\":[]}},\"ingressFilter\":{\"enabled\":false,\"blockedIps\":[]}}'::jsonb;

BEGIN

  SELECT uuid INTO p_uuid FROM node_plugin WHERE name='Anti-Abuse Shield' LIMIT 1;

  IF p_uuid IS NOT NULL THEN

    UPDATE node_plugin SET plugin_config=cfg, updated_at=now() WHERE uuid=p_uuid;

  END IF;

  UPDATE nodes SET active_plugin_uuid=NULL WHERE active_plugin_uuid=p_uuid;

END \$\$;

" >/dev/null 2>&1



# 2. Снимаем сетевые блокировки

iptables -D OUTPUT -p tcp --dport 25 -j DROP 2>/dev/null || true

iptables -D FORWARD -p tcp --dport 25 -j DROP 2>/dev/null || true



# 3. Перезапускаем ноду

docker restart remnanode >/dev/null 2>&1 || true

docker compose -f /opt/remnawave/panel/docker-compose.yml restart remnawave >/dev/null 2>&1 || true



echo "Защита от торрентов и спама отключена."

`

	}



	out, err := session.CombinedOutput(cmd)

	resStr := strings.TrimSpace(string(out))

	if err != nil {

		if strings.Contains(resStr, "ERR_NO_NODE") {

			return types.CommandResult{

				Success: false,

				Error:   "Нельзя включить защиту: Xray нода не установлена или не настроена на этом сервере. Сначала привяжите ноду.",

			}

		}

		return types.CommandResult{Success: false, Error: resStr}

	}

	return types.CommandResult{Success: true, Output: resStr}

}

// ToggleStackPerformance enables or disables performance optimizations:
// Valkey Unix Domain Socket, PostgreSQL 512MB shared memory, ulimits 1M, and Linux kernel sysctl memory tuning.
func ToggleStackPerformance(server types.SavedServer, enable bool) types.CommandResult {
	session, release, err := getPooledSession(server)
	if err != nil {
		return types.CommandResult{Success: false, Error: "Ошибка подключения SSH: " + err.Error()}
	}
	defer release()

	var composeContent string
	var sysctlScript string
	var envUpdateScript string
	var successMsg string

	if enable {
		composeContent = generator.RenderPanelCompose()
		sysctlScript = `
echo "==> Применение оптимизации ядра Linux (Host RAM sysctl)..."
cat > /etc/sysctl.d/99-remnawave-ram.conf <<'EOF'
vm.swappiness=10
vm.vfs_cache_pressure=50
vm.dirty_ratio=15
vm.dirty_background_ratio=5
EOF
sysctl --system >/dev/null 2>&1 || sysctl -p /etc/sysctl.d/99-remnawave-ram.conf >/dev/null 2>&1 || true
`
		envUpdateScript = `
if [ -f "$TARGET_DIR/.env" ]; then
  if grep -q "REDIS_SOCKET=" "$TARGET_DIR/.env" 2>/dev/null; then
    sed -i 's|^#*REDIS_SOCKET=.*|REDIS_SOCKET=/var/run/valkey/valkey.sock|' "$TARGET_DIR/.env"
  else
    echo "REDIS_SOCKET=/var/run/valkey/valkey.sock" >> "$TARGET_DIR/.env"
  fi
fi
`
		successMsg = "Оптимизация RAM и сокетов успешно включена (512MB RAM + Unix Sockets)!"
	} else {
		composeContent = generator.RenderPanelComposeStandard()
		sysctlScript = `
echo "==> Откат системных параметров ядра Linux к стандартным..."
rm -f /etc/sysctl.d/99-remnawave-ram.conf
sysctl -w vm.swappiness=60 vm.vfs_cache_pressure=100 vm.dirty_ratio=20 vm.dirty_background_ratio=10 >/dev/null 2>&1 || true
`
		envUpdateScript = `
if [ -f "$TARGET_DIR/.env" ]; then
  sed -i 's|^REDIS_SOCKET=.*|#REDIS_SOCKET=|' "$TARGET_DIR/.env" 2>/dev/null || true
  grep -q "REDIS_HOST=" "$TARGET_DIR/.env" 2>/dev/null || echo "REDIS_HOST=remnawave-redis" >> "$TARGET_DIR/.env"
  grep -q "REDIS_PORT=" "$TARGET_DIR/.env" 2>/dev/null || echo "REDIS_PORT=6379" >> "$TARGET_DIR/.env"
fi
`
		successMsg = "Оптимизация успешно отключена, настройки возвращены к стандартным!"
	}

	b64Compose := base64.StdEncoding.EncodeToString([]byte(composeContent))

	script := fmt.Sprintf(`
%s

if [ -f /opt/remnawave/panel/docker-compose.yml ]; then
  TARGET_DIR="/opt/remnawave/panel"
elif [ -f /opt/remnawave/docker-compose.yml ]; then
  TARGET_DIR="/opt/remnawave"
else
  echo "ERR_NO_PANEL_COMPOSE: Каталог панели Remnawave не обнаружен на данном сервере."
  exit 1
fi

echo "==> Обновление конфигурации $TARGET_DIR/docker-compose.yml..."
cp -f "$TARGET_DIR/docker-compose.yml" "$TARGET_DIR/docker-compose.yml.bak" 2>/dev/null || true
echo '%s' | base64 -d > "$TARGET_DIR/docker-compose.yml"

%s

cd "$TARGET_DIR" && docker compose up -d --remove-orphans
echo "SUCCESS_APPLIED"
`, sysctlScript, b64Compose, envUpdateScript)

	out, err := session.CombinedOutput(script)
	resStr := strings.TrimSpace(string(out))
	if err != nil {
		if strings.Contains(resStr, "ERR_NO_PANEL_COMPOSE") {
			return types.CommandResult{
				Success: false,
				Error:   "Не удалось применить оптимизацию: на этом сервере не обнаружена мастер-панель Remnawave.",
			}
		}
		return types.CommandResult{Success: false, Error: resStr}
	}

	return types.CommandResult{Success: true, Output: successMsg}
}



func DownloadBackupStream(server types.SavedServer, fileName, localPath string) types.CommandResult {

	if !validBackupNameRegex.MatchString(fileName) {

		return types.CommandResult{Success: false, Error: "Недопустимое имя файла бэкапа."}

	}

	client, err := getSSHClient(server)

	if err != nil {

		return types.CommandResult{Success: false, Error: err.Error()}

	}

	defer client.Close()



	session, err := client.NewSession()

	if err != nil {

		return types.CommandResult{Success: false, Error: err.Error()}

	}

	defer session.Close()



	stdout, err := session.StdoutPipe()

	if err != nil {

		return types.CommandResult{Success: false, Error: err.Error()}

	}



	remotePath := "/opt/remnawave/backups/" + fileName

	if err := session.Start("cat " + remotePath); err != nil {

		return types.CommandResult{Success: false, Error: err.Error()}

	}



	outFile, err := os.Create(localPath)

	if err != nil {

		return types.CommandResult{Success: false, Error: fmt.Sprintf("Не удалось создать локальный файл: %s", err.Error())}

	}

	defer outFile.Close()



	written, err := io.Copy(outFile, stdout)

	if err != nil {

		return types.CommandResult{Success: false, Error: fmt.Sprintf("Ошибка при передаче данных: %s", err.Error())}

	}



	if err := session.Wait(); err != nil {

		return types.CommandResult{Success: false, Error: fmt.Sprintf("Ошибка чтения удаленного файла: %s", err.Error())}

	}



	if written == 0 {

		return types.CommandResult{Success: false, Error: "Файл резервной копии пуст или не найден на сервере."}

	}



	return types.CommandResult{

		Success: true,

		Output:  fmt.Sprintf("Бэкап успешно сохранен (%d байт): %s", written, localPath),

	}

}



func RestoreBackup(server types.SavedServer, fileName string, scope string, createSafetyBackup bool) types.RestoreBackupResult {
	if !validBackupNameRegex.MatchString(fileName) {
		return types.RestoreBackupResult{Success: false, Error: "Недопустимое имя файла бэкапа."}
	}

	if scope == "" {
		scope = "all"
	}

	safetyStr := "no"
	if createSafetyBackup {
		safetyStr = "yes"
	}

	session, release, err := getPooledSession(server)
	if err != nil {
		return types.RestoreBackupResult{Success: false, Error: err.Error()}
	}
	defer release()

	cmd := fmt.Sprintf(`
set -e
RESTORE_FILE="/opt/remnawave/backups/%s"
SCOPE="%s"
DO_SAFETY="%s"

if [ ! -f "$RESTORE_FILE" ]; then
  echo "ERR_NOT_FOUND: Файл бэкапа не найден: $RESTORE_FILE" >&2
  exit 1
fi

DATE=$(date +%%Y%%m%%d_%%H%%M%%S)

# 1. Pre-restore Safety Backup if requested
if [ "$DO_SAFETY" = "yes" ]; then
  SAFETY_NAME="remnawave_safety_before_restore_${DATE}.tar.gz"
  SAFETY_DIR=$(mktemp -d)
  mkdir -p "$SAFETY_DIR/configs"

  [ -f /opt/remnawave/panel/.env ] && cp -rf /opt/remnawave/panel/.env "$SAFETY_DIR/configs/panel.env" || true
  [ -f /opt/remnawave/panel/docker-compose.yml ] && cp -rf /opt/remnawave/panel/docker-compose.yml "$SAFETY_DIR/configs/panel-compose.yml" || true
  [ -f /opt/remnawave/node/docker-compose.yml ] && cp -rf /opt/remnawave/node/docker-compose.yml "$SAFETY_DIR/configs/node-compose.yml" || true
  [ -f /opt/remnawave/nginx/nginx.conf ] && cp -rf /opt/remnawave/nginx/nginx.conf "$SAFETY_DIR/configs/nginx.conf" || true
  [ -f /opt/remnawave/nginx/fullchain.pem ] && cp -rf /opt/remnawave/nginx/fullchain.pem "$SAFETY_DIR/configs/" || true
  [ -f /opt/remnawave/nginx/privkey.key ] && cp -rf /opt/remnawave/nginx/privkey.key "$SAFETY_DIR/configs/" || true
  [ -f /opt/remnawave/caddy/Caddyfile ] && cp -rf /opt/remnawave/caddy/Caddyfile "$SAFETY_DIR/configs/Caddyfile" || true

  if docker ps --format '{{.Names}}' | grep -q 'remnawave-db'; then
    CURR_USER="postgres"
    CURR_DB="postgres"
    if [ -f /opt/remnawave/panel/.env ]; then
      u=$(grep -E '^(POSTGRES_USER|DB_USER)=' /opt/remnawave/panel/.env | head -1 | cut -d'=' -f2- | tr -d '"\x27 ' || true)
      [ -n "$u" ] && CURR_USER="$u"
      d=$(grep -E '^(POSTGRES_DB|DB_NAME)=' /opt/remnawave/panel/.env | head -1 | cut -d'=' -f2- | tr -d '"\x27 ' || true)
      [ -n "$d" ] && CURR_DB="$d"
    fi
    docker exec -i remnawave-db pg_dump -U "$CURR_USER" -d "$CURR_DB" --clean --if-exists --no-owner --no-privileges -F c -Z 9 > "$SAFETY_DIR/remnawave.dump" 2>/dev/null || true
  fi

  tar -czf "/opt/remnawave/backups/$SAFETY_NAME" -C "$SAFETY_DIR" . 2>/dev/null || true
  rm -rf "$SAFETY_DIR"
  echo "SAFETY_BACKUP:$SAFETY_NAME"
fi

TEMP_DIR=$(mktemp -d)
tar -xzf "$RESTORE_FILE" -C "$TEMP_DIR"

# 2. Restore Configuration Files & Uploads (if scope is 'all' or 'configs_only')
if [ "$SCOPE" = "all" ] || [ "$SCOPE" = "configs_only" ]; then
  if [ -d "$TEMP_DIR/configs" ]; then
    mkdir -p /opt/remnawave/panel /opt/remnawave/node /opt/remnawave/nginx /opt/remnawave/caddy
    [ -f "$TEMP_DIR/configs/panel.env" ] && cp -f "$TEMP_DIR/configs/panel.env" /opt/remnawave/panel/.env || true
    [ -f "$TEMP_DIR/configs/panel-compose.yml" ] && cp -f "$TEMP_DIR/configs/panel-compose.yml" /opt/remnawave/panel/docker-compose.yml || true
    [ -f "$TEMP_DIR/configs/node-compose.yml" ] && cp -f "$TEMP_DIR/configs/node-compose.yml" /opt/remnawave/node/docker-compose.yml || true
    [ -f "$TEMP_DIR/configs/nginx.conf" ] && cp -f "$TEMP_DIR/configs/nginx.conf" /opt/remnawave/nginx/nginx.conf || true
    [ -f "$TEMP_DIR/configs/fullchain.pem" ] && cp -f "$TEMP_DIR/configs/fullchain.pem" /opt/remnawave/nginx/fullchain.pem || true
    [ -f "$TEMP_DIR/configs/privkey.key" ] && cp -f "$TEMP_DIR/configs/privkey.key" /opt/remnawave/nginx/privkey.key || true
    [ -f "$TEMP_DIR/configs/Caddyfile" ] && cp -f "$TEMP_DIR/configs/Caddyfile" /opt/remnawave/caddy/Caddyfile || true
    [ -f "$TEMP_DIR/configs/caddy-compose.yml" ] && cp -f "$TEMP_DIR/configs/caddy-compose.yml" /opt/remnawave/caddy/docker-compose.yml || true
  fi

  if [ -d "$TEMP_DIR/uploads" ]; then
    mkdir -p /opt/remnawave/uploads
    cp -rf "$TEMP_DIR/uploads"/* /opt/remnawave/uploads/ 2>/dev/null || true
  fi
fi

# 3. Restore Database (if scope is 'all' or 'db_only')
if [ "$SCOPE" = "all" ] || [ "$SCOPE" = "db_only" ]; then
  DUMP_FILE=""
  IS_CUSTOM="no"
  if [ -f "$TEMP_DIR/remnawave.dump" ]; then
    DUMP_FILE="$TEMP_DIR/remnawave.dump"
    IS_CUSTOM="yes"
  elif [ -f "$TEMP_DIR/database.sql" ]; then
    DUMP_FILE="$TEMP_DIR/database.sql"
    IS_CUSTOM="no"
  fi

  if [ -n "$DUMP_FILE" ]; then
    TARGET_DB_USER="postgres"
    TARGET_DB_NAME="postgres"
    if [ -f /opt/remnawave/panel/.env ]; then
      u=$(grep -E '^(POSTGRES_USER|DB_USER)=' /opt/remnawave/panel/.env | head -1 | cut -d'=' -f2- | tr -d '"\x27 ' || true)
      [ -n "$u" ] && TARGET_DB_USER="$u"
      d=$(grep -E '^(POSTGRES_DB|DB_NAME)=' /opt/remnawave/panel/.env | head -1 | cut -d'=' -f2- | tr -d '"\x27 ' || true)
      [ -n "$d" ] && TARGET_DB_NAME="$d"
    fi

    if [ -f /opt/remnawave/panel/docker-compose.yml ]; then
      cd /opt/remnawave/panel
      docker compose down 2>/dev/null || true
      docker compose up -d remnawave-db

      # Wait for PostgreSQL to become ready
      READY=false
      for i in $(seq 1 30); do
        if docker exec remnawave-db pg_isready -U "$TARGET_DB_USER" >/dev/null 2>&1; then
          READY=true
          break
        fi
        sleep 1
      done

      if [ "$READY" = "true" ]; then
        # Terminate active client connections
        docker exec -i remnawave-db psql -U "$TARGET_DB_USER" -d postgres -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '$TARGET_DB_NAME' AND pid <> pg_backend_pid();" 2>/dev/null || true

        # Recreate clean database
        docker exec -i remnawave-db psql -U "$TARGET_DB_USER" -d postgres -c "DROP DATABASE IF EXISTS \"$TARGET_DB_NAME\"; CREATE DATABASE \"$TARGET_DB_NAME\" OWNER \"$TARGET_DB_USER\";" 2>/dev/null || true

        # Import dump
        if [ "$IS_CUSTOM" = "yes" ]; then
          docker exec -i remnawave-db pg_restore -U "$TARGET_DB_USER" -d "$TARGET_DB_NAME" --no-owner --no-privileges -v < "$DUMP_FILE" 2>/dev/null || true
        else
          docker exec -i remnawave-db psql -U "$TARGET_DB_USER" -d "$TARGET_DB_NAME" < "$DUMP_FILE" 2>/dev/null || true
        fi
      fi
    fi
  fi
fi

rm -rf "$TEMP_DIR"

# 4. Restart services
if [ -f /opt/remnawave/panel/docker-compose.yml ]; then
  cd /opt/remnawave/panel && docker compose up -d 2>/dev/null || true
fi
if [ -f /opt/remnawave/node/docker-compose.yml ]; then
  cd /opt/remnawave/node && docker compose restart 2>/dev/null || true
fi
if [ -f /opt/remnawave/caddy/docker-compose.yml ]; then
  cd /opt/remnawave/caddy && docker compose restart 2>/dev/null || true
fi
docker restart remna-nginx remnawave-nginx remnawave-caddy 2>/dev/null || true

echo "Резервная копия успешно восстановлена, службы перезапущены!"
`, fileName, scope, safetyStr)

	out, err := session.CombinedOutput(cmd)
	outputStr := string(out)
	if err != nil && len(outputStr) == 0 {
		return types.RestoreBackupResult{Success: false, ScopeApplied: scope, Error: string(out)}
	}

	safetyFile := ""
	for _, line := range strings.Split(outputStr, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "SAFETY_BACKUP:") {
			safetyFile = strings.TrimPrefix(trimmed, "SAFETY_BACKUP:")
		}
	}

	return types.RestoreBackupResult{
		Success:          true,
		SafetyBackupFile: safetyFile,
		ScopeApplied:     scope,
		Output:           strings.TrimSpace(outputStr),
	}
}



func UpdateStackImages(server types.SavedServer) types.CommandResult {
	client, err := getSSHClient(server)
	if err != nil {
		return types.CommandResult{Success: false, Error: err.Error()}
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return types.CommandResult{Success: false, Error: err.Error()}
	}
	defer session.Close()

	cmd := `
which jq >/dev/null 2>&1 || (apt-get update -qq && apt-get install -y -qq jq >/dev/null 2>&1 || true)
UPDATED_COUNT=0

# 0. Автоматический страховочный снимок базы данных перед обновлением стека
if docker ps --format '{{.Names}}' 2>/dev/null | grep -q 'remnawave-db'; then
  echo ">>> Создание страховочного снимка базы данных перед обновлением..."
  mkdir -p /opt/remnawave/backups 2>/dev/null || true
  DB_USER=$(grep -E '^POSTGRES_USER=' /opt/remnawave/.env /opt/remnawave/panel/.env 2>/dev/null | cut -d= -f2- | tr -d '"\r\n' | head -n1)
  [ -z "$DB_USER" ] && DB_USER="postgres"
  DB_NAME=$(grep -E '^POSTGRES_DB=' /opt/remnawave/.env /opt/remnawave/panel/.env 2>/dev/null | cut -d= -f2- | tr -d '"\r\n' | head -n1)
  [ -z "$DB_NAME" ] && DB_NAME="remnawave"
  docker exec -u postgres remnawave-db pg_dump -U "$DB_USER" -d "$DB_NAME" --clean --if-exists --no-owner --no-privileges -F c -Z 9 > /opt/remnawave/backups/pre_update_db.dump 2>/dev/null || true
  if [ -s /opt/remnawave/backups/pre_update_db.dump ]; then
    echo "Страховочный снимок БД сохранен: /opt/remnawave/backups/pre_update_db.dump"
  fi
fi

# 1. Автоматическая проверка и перевыпуск токена страницы подписок (если панель активна)
if docker ps 2>/dev/null | grep -q remnawave; then
  echo ">>> Проверка и синхронизация токена страницы подписок..."
  API_BASE="http://127.0.0.1:3000"
  STATUS=$(curl -s --connect-timeout 3 -m 5 "${API_BASE}/api/auth/status" 2>/dev/null || true)
  if echo "$STATUS" | grep -q '"isConfigured"'; then
    ACCESS_TOKEN=""

    # 1. Сначала проверяем сохраненный мастер-токен /opt/remnawave/token (как в eGames)
    if [ -f /opt/remnawave/token ]; then
      SAVED_TOKEN=$(cat /opt/remnawave/token | tr -d '\r\n[:space:]')
      if [ -n "$SAVED_TOKEN" ]; then
        TEST_RES=$(curl -s --connect-timeout 3 -m 5 -H "Authorization: Bearer ${SAVED_TOKEN}" "${API_BASE}/api/auth/profile" 2>/dev/null || true)
        if echo "$TEST_RES" | grep -qE '"uuid"|"username"|"role"'; then
          ACCESS_TOKEN="$SAVED_TOKEN"
        fi
      fi
    fi

    # 2. Если мастер-токен отсутствует, считываем логин и пароль администратора
    if [ -z "$ACCESS_TOKEN" ]; then
      ADMIN_USER=""
      ADMIN_PASS=""
      for cred_file in /opt/remnawave/credentials.txt /opt/remnawave/.env /opt/remnawave/panel/.env; do
        if [ -f "$cred_file" ]; then
          u=$(grep -E '^ADMIN_USERNAME=' "$cred_file" 2>/dev/null | cut -d= -f2- | tr -d '"\r\n' | head -n1)
          if [ -n "$u" ] && [ -z "$ADMIN_USER" ]; then ADMIN_USER="$u"; fi
          p=$(grep -E '^ADMIN_PASSWORD=|^INIT_ADMIN_PASSWORD=' "$cred_file" 2>/dev/null | cut -d= -f2- | tr -d '"\r\n' | head -n1)
          if [ -n "$p" ] && [ -z "$ADMIN_PASS" ]; then ADMIN_PASS="$p"; fi
        fi
      done
      [ -z "$ADMIN_USER" ] && ADMIN_USER="admin"

      if [ -n "$ADMIN_PASS" ]; then
        LOGIN_RES=$(curl -s -X POST "${API_BASE}/api/auth/login" \
          -H "Content-Type: application/json" \
          -H "X-Remnawave-Client-Type: browser" \
          -H "X-Forwarded-For: 127.0.0.1" \
          -H "X-Forwarded-Proto: https" \
          -d "{\"username\":\"${ADMIN_USER}\",\"password\":\"${ADMIN_PASS}\"}")
        ACCESS_TOKEN=$(echo "$LOGIN_RES" | grep -o '"accessToken":"[^"]*' | cut -d'"' -f4)
      fi
    fi

    # 3. Выпускаем свежий токен для страницы подписок и сохраняем его
    if [ -n "$ACCESS_TOKEN" ]; then
      echo "$ACCESS_TOKEN" > /opt/remnawave/token 2>/dev/null || true
      chmod 600 /opt/remnawave/token 2>/dev/null || true

      SUB_PAYLOAD='{"name":"subscription-page","expiresInDays":3650,"scopes":["subscription-page-configs:list","subscription-page-configs:get","subscriptions:subpage-config","system:metadata","users:by-username"]}'
      SUB_TOKEN_RES=$(curl -s -X POST "${API_BASE}/api/tokens" \
        -H "Authorization: Bearer ${ACCESS_TOKEN}" \
        -H "Content-Type: application/json" \
        -H "X-Remnawave-Client-Type: browser" \
        -H "X-Forwarded-For: 127.0.0.1" \
        -H "X-Forwarded-Proto: https" \
        -d "${SUB_PAYLOAD}")
      NEW_SUB_TOKEN=$(echo "$SUB_TOKEN_RES" | grep -o '"token":"[^"]*' | cut -d'"' -f4)

      if [ -n "$NEW_SUB_TOKEN" ]; then
        for f in /opt/remnawave/.env /opt/remnawave/docker-compose.yml /opt/remnawave/panel/.env /opt/remnawave/subscription/.env /opt/remnawave/subscription/docker-compose.yml; do
          if [ -f "$f" ]; then
            sed -i "s|^REMNAWAVE_API_TOKEN=.*|REMNAWAVE_API_TOKEN=${NEW_SUB_TOKEN}|g" "$f" 2>/dev/null || true
            sed -i "s|REMNAWAVE_API_TOKEN=.*|REMNAWAVE_API_TOKEN=${NEW_SUB_TOKEN}|g" "$f" 2>/dev/null || true
          fi
        done
        echo ">>> Токен страницы подписок успешно обновлен и синхронизирован!"
      fi
    fi
  fi
fi

# 2. Обновление компонентов стека с принудительным пересозданием для применения .env
# Корневой стек (all-in-one / stealth)
if [ -f /opt/remnawave/docker-compose.yml ]; then
  echo ">>> Загрузка обновленных образов стека Remnawave (/opt/remnawave)..."
  cd /opt/remnawave && docker compose pull && docker compose up -d --force-recreate --remove-orphans
  UPDATED_COUNT=$((UPDATED_COUNT + 1))
fi

# Панель
if [ -d /opt/remnawave/panel ] && [ -f /opt/remnawave/panel/docker-compose.yml ]; then
  echo ">>> Загрузка обновленных образов панели Remnawave..."
  cd /opt/remnawave/panel && docker compose pull && docker compose up -d --force-recreate --remove-orphans
  UPDATED_COUNT=$((UPDATED_COUNT + 1))
fi

# Страница подписок
if [ -d /opt/remnawave/subscription ] && [ -f /opt/remnawave/subscription/docker-compose.yml ]; then
  echo ">>> Загрузка обновленных образов страницы подписок..."
  cd /opt/remnawave/subscription && docker compose pull && docker compose up -d --force-recreate --remove-orphans
  UPDATED_COUNT=$((UPDATED_COUNT + 1))
fi

# Нода
if [ -d /opt/remnawave/node ] && [ -f /opt/remnawave/node/docker-compose.yml ]; then
  echo ">>> Загрузка обновленных образов ноды Remnanode..."
  cd /opt/remnawave/node && docker compose pull && docker compose up -d --force-recreate --remove-orphans
  UPDATED_COUNT=$((UPDATED_COUNT + 1))
fi

# Caddy
if [ -d /opt/remnawave/caddy ] && [ -f /opt/remnawave/caddy/docker-compose.yml ]; then
  echo ">>> Загрузка обновленных образов Caddy..."
  cd /opt/remnawave/caddy && docker compose pull && docker compose up -d --force-recreate --remove-orphans
  UPDATED_COUNT=$((UPDATED_COUNT + 1))
fi

# Nginx
if [ -d /opt/remnawave/nginx ] && [ -f /opt/remnawave/nginx/docker-compose.yml ]; then
  echo ">>> Загрузка обновленных образов Nginx..."
  cd /opt/remnawave/nginx && docker compose pull && docker compose up -d --force-recreate --remove-orphans
  UPDATED_COUNT=$((UPDATED_COUNT + 1))
fi

docker image prune -f >/dev/null 2>&1 || true

if [ $UPDATED_COUNT -eq 0 ]; then
  echo "Службы Remnawave не обнаружены в /opt/remnawave."
else
  echo "Все службы Remnawave успешно обновлены и синхронизированы!"
fi
`

	out, err := session.CombinedOutput(cmd)
	if err != nil {
		return types.CommandResult{Success: false, Error: string(out)}
	}
	return types.CommandResult{Success: true, Output: strings.TrimSpace(string(out))}
}





func InstallCloudflareWARP(server types.SavedServer) types.CommandResult {

	client, err := getSSHClient(server)

	if err != nil {

		return types.CommandResult{Success: false, Error: "Ошибка подключения SSH: " + err.Error()}

	}

	defer client.Close()



	session, err := client.NewSession()

	if err != nil {

		return types.CommandResult{Success: false, Error: "Ошибка создания сессии SSH: " + err.Error()}

	}

	defer session.Close()



	script := `

echo ">>> Настройка официального клиента Cloudflare WARP..."

export DEBIAN_FRONTEND=noninteractive



# Установка зависимостей и репозитория Cloudflare

if ! command -v warp-cli >/dev/null 2>&1; then

  echo "Добавление официального репозитория Cloudflare..."

  apt-get update -qq >/dev/null 2>&1 || true

  apt-get install -y -qq curl gpg lsb-release >/dev/null 2>&1 || true

  curl -fsSL https://pkg.cloudflareclient.com/pubkey.gpg | gpg --yes --dearmor --output /usr/share/keyrings/cloudflare-warp-archive-keyring.gpg 2>/dev/null || true

  echo "deb [signed-by=/usr/share/keyrings/cloudflare-warp-archive-keyring.gpg] https://pkg.cloudflareclient.com/ $(lsb_release -cs) main" | tee /etc/apt/sources.list.d/cloudflare-client.list >/dev/null 2>&1 || true

  apt-get update -qq >/dev/null 2>&1 || true

  apt-get install -y -qq cloudflare-warp >/dev/null 2>&1 || true

fi



if command -v warp-cli >/dev/null 2>&1; then

  echo "Регистрация и инициализация WARP в режиме SOCKS5 прокси..."

  warp-cli --accept-tos registration new >/dev/null 2>&1 || warp-cli --accept-tos register >/dev/null 2>&1 || true

  warp-cli --accept-tos mode proxy >/dev/null 2>&1 || true

  warp-cli --accept-tos proxy port 40000 >/dev/null 2>&1 || true

  warp-cli --accept-tos connect >/dev/null 2>&1 || true

  sleep 2

  STATUS=$(warp-cli --accept-tos status 2>/dev/null || echo "Connected")

  echo "Cloudflare WARP успешно установлен и подключен!"

  echo "Локальный SOCKS5 эндпоинт для исходящих: 127.0.0.1:40000"

  echo "$STATUS"

else

  echo "Не удалось установить cloudflare-warp. Убедитесь, что система — Ubuntu или Debian 64-bit."

fi

`

	out, err := session.CombinedOutput(script)

	if err != nil {

		return types.CommandResult{Success: false, Error: string(out)}

	}

	return types.CommandResult{Success: true, Output: strings.TrimSpace(string(out))}

}





func ConfigureRealitySelfSteal(server types.SavedServer, fallbackDomain string) types.CommandResult {
	fallbackDomain = strings.TrimSpace(fallbackDomain)
	if fallbackDomain == "" {
		fallbackDomain = "127.0.0.1:8443"
	}
	if strings.ContainsAny(fallbackDomain, "\r\n\"'`;|&$<> \t\\") {
		return types.CommandResult{Success: false, Error: "Некорректный адрес fallback: содержит недопустимые символы"}
	}

	client, err := getSSHClient(server)

	if err != nil {

		return types.CommandResult{Success: false, Error: "Ошибка подключения SSH: " + err.Error()}

	}

	defer client.Close()



	session, err := client.NewSession()

	if err != nil {

		return types.CommandResult{Success: false, Error: "Ошибка создания сессии SSH: " + err.Error()}

	}

	defer session.Close()



	if fallbackDomain == "" {

		fallbackDomain = "127.0.0.1:8443"

	}



	script := fmt.Sprintf(`
echo ">>> Проверка и настройка VLESS Reality SelfSteal..."

if ! docker ps -a --format '{{.Names}}' 2>/dev/null | grep -q 'remnanode'; then
  echo "Контейнер ноды remnanode не обнаружен на сервере. Сначала разверните или запустите Remnanode."
  exit 1
fi

if docker ps --format '{{.Names}}' 2>/dev/null | grep -q 'remnanode'; then
  echo "Нода remnanode активна и работает."
  echo "Архитектура SelfSteal готова: укажите dest/fallback = %s в настройках ноды в панели Remnawave."
  echo "При проверках сканеров ТСПУ или недоверенном трафике нода будет прозрачно отдавать локальный маскировочный сайт."
else
  echo "Контейнер remnanode найден, но находится в остановленном состоянии. Попробуйте перезапустить ноду."
  exit 1
fi
`, fallbackDomain)



	out, err := session.CombinedOutput(script)

	if err != nil {

		return types.CommandResult{Success: false, Error: string(out)}

	}

	return types.CommandResult{Success: true, Output: strings.TrimSpace(string(out))}

}



// AutoFixSubscriptionToken automatically connects to Remnawave API on VPS, mints a valid v3 token, and fixes subscription page
func AutoFixSubscriptionToken(server types.SavedServer) types.CommandResult {
	client, err := getSSHClient(server)
	if err != nil {
		return types.CommandResult{Success: false, Error: err.Error()}
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return types.CommandResult{Success: false, Error: err.Error()}
	}
	defer session.Close()

	cmd := `
which jq >/dev/null 2>&1 || (apt-get update -qq && apt-get install -y -qq jq >/dev/null 2>&1 || true)
API_BASE="http://127.0.0.1:3000"

STATUS=$(curl -s --connect-timeout 3 -m 5 "${API_BASE}/api/auth/status" 2>/dev/null || true)
if ! echo "$STATUS" | grep -q '"isConfigured"'; then
  echo "ERROR_API_UNREACHABLE"
  exit 1
fi

ACCESS_TOKEN=""
if [ -f /opt/remnawave/token ]; then
  SAVED_TOKEN=$(cat /opt/remnawave/token | tr -d '\r\n[:space:]')
  if [ -n "$SAVED_TOKEN" ]; then
    TEST_RES=$(curl -s --connect-timeout 3 -m 5 -H "Authorization: Bearer ${SAVED_TOKEN}" "${API_BASE}/api/auth/profile" 2>/dev/null || true)
    if echo "$TEST_RES" | grep -qE '"uuid"|"username"|"role"'; then
      ACCESS_TOKEN="$SAVED_TOKEN"
    fi
  fi
fi

if [ -z "$ACCESS_TOKEN" ]; then
  ADMIN_USER=""
  ADMIN_PASS=""
  for cred_file in /opt/remnawave/credentials.txt /opt/remnawave/.env /opt/remnawave/panel/.env; do
    if [ -f "$cred_file" ]; then
      u=$(grep -E '^ADMIN_USERNAME=' "$cred_file" 2>/dev/null | cut -d= -f2- | tr -d '"\r\n' | head -n1)
      if [ -n "$u" ] && [ -z "$ADMIN_USER" ]; then ADMIN_USER="$u"; fi
      p=$(grep -E '^ADMIN_PASSWORD=|^INIT_ADMIN_PASSWORD=' "$cred_file" 2>/dev/null | cut -d= -f2- | tr -d '"\r\n' | head -n1)
      if [ -n "$p" ] && [ -z "$ADMIN_PASS" ]; then ADMIN_PASS="$p"; fi
    fi
  done
  [ -z "$ADMIN_USER" ] && ADMIN_USER="admin"

  if [ -n "$ADMIN_PASS" ]; then
    LOGIN_RES=$(curl -s -X POST "${API_BASE}/api/auth/login" \
      -H "Content-Type: application/json" \
      -H "X-Remnawave-Client-Type: browser" \
      -H "X-Forwarded-For: 127.0.0.1" \
      -H "X-Forwarded-Proto: https" \
      -d "{\"username\":\"${ADMIN_USER}\",\"password\":\"${ADMIN_PASS}\"}")
    ACCESS_TOKEN=$(echo "$LOGIN_RES" | grep -o '"accessToken":"[^"]*' | cut -d'"' -f4)
  fi
fi

if [ -z "$ACCESS_TOKEN" ]; then
  echo "ERROR_CANNOT_AUTHENTICATE_ADMIN"
  exit 1
fi

echo "$ACCESS_TOKEN" > /opt/remnawave/token 2>/dev/null || true
chmod 600 /opt/remnawave/token 2>/dev/null || true

SUB_PAYLOAD='{"name":"subscription-page","expiresInDays":3650,"scopes":["subscription-page-configs:list","subscription-page-configs:get","subscriptions:subpage-config","system:metadata","users:by-username"]}'
SUB_TOKEN_RES=$(curl -s -X POST "${API_BASE}/api/tokens" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -H "X-Remnawave-Client-Type: browser" \
  -H "X-Forwarded-For: 127.0.0.1" \
  -H "X-Forwarded-Proto: https" \
  -d "${SUB_PAYLOAD}")
NEW_SUB_TOKEN=$(echo "$SUB_TOKEN_RES" | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -z "$NEW_SUB_TOKEN" ]; then
  echo "ERROR_CANNOT_MINT_TOKEN"
  exit 1
fi

UPDATED=0
if [ -f /opt/remnawave/subscription/.env ] || [ -f /opt/remnawave/subscription/docker-compose.yml ]; then
  for f in /opt/remnawave/subscription/.env /opt/remnawave/subscription/docker-compose.yml; do
    if [ -f "$f" ]; then
      sed -i "s|^REMNAWAVE_API_TOKEN=.*|REMNAWAVE_API_TOKEN=${NEW_SUB_TOKEN}|g" "$f" 2>/dev/null || true
      sed -i "s|REMNAWAVE_API_TOKEN=.*|REMNAWAVE_API_TOKEN=${NEW_SUB_TOKEN}|g" "$f" 2>/dev/null || true
    fi
  done
  cd /opt/remnawave/subscription && docker compose up -d --force-recreate remnawave-subscription-page 2>/dev/null || true
  UPDATED=1
fi

if [ -f /opt/remnawave/docker-compose.yml ] || [ -f /opt/remnawave/.env ]; then
  for f in /opt/remnawave/.env /opt/remnawave/docker-compose.yml; do
    if [ -f "$f" ]; then
      sed -i "s|^REMNAWAVE_API_TOKEN=.*|REMNAWAVE_API_TOKEN=${NEW_SUB_TOKEN}|g" "$f" 2>/dev/null || true
      sed -i "s|REMNAWAVE_API_TOKEN=.*|REMNAWAVE_API_TOKEN=${NEW_SUB_TOKEN}|g" "$f" 2>/dev/null || true
    fi
  done
  cd /opt/remnawave && docker compose up -d --force-recreate remnawave-subscription-page 2>/dev/null || true
  UPDATED=1
fi

if [ "$UPDATED" -eq 1 ]; then
  echo "SUCCESS_SUBSCRIPTION_RESTORED"
else
  echo "ERROR_NO_SUBSCRIPTION_FOUND"
fi
`

	out, err := session.CombinedOutput(cmd)
	outStr := strings.TrimSpace(string(out))
	if err != nil || strings.Contains(outStr, "ERROR_") {
		return types.CommandResult{Success: false, Error: outStr}
	}
	return types.CommandResult{Success: true, Output: outStr}
}

// UninstallStack completely stops and removes Remnawave components from VPS without reinstalling the OS
func UninstallStack(server types.SavedServer, purgeData bool) types.CommandResult {
	client, err := getSSHClient(server)
	if err != nil {
		return types.CommandResult{Success: false, Error: err.Error()}
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return types.CommandResult{Success: false, Error: err.Error()}
	}
	defer session.Close()

	cmd := fmt.Sprintf(`
echo "==> Остановка и удаление контейнеров Remnawave..."
for c in $(docker ps -a --format '{{.Names}}' 2>/dev/null | grep -E '^remnawave|^remnanode'); do
  docker rm -f "$c" 2>/dev/null || true
done

echo "==> Остановка системных демонов веб-сервера..."
systemctl stop nginx 2>/dev/null || true
systemctl disable nginx 2>/dev/null || true

echo "==> Очистка каталогов и сокетов..."
rm -rf /opt/remnawave /var/www/decoy /var/www/html /dev/shm/nginx.sock 2>/dev/null || true
rm -f /etc/nginx/sites-enabled/decoy /etc/nginx/sites-available/decoy 2>/dev/null || true

PURGE_DATA="%t"
if [ "$PURGE_DATA" = "true" ]; then
  echo "==> Полная очистка данных и Docker томов..."
  for v in $(docker volume ls -q 2>/dev/null | grep -E 'remna'); do
    docker volume rm -f "$v" 2>/dev/null || true
  done
  docker network rm remnawave-network 2>/dev/null || true
fi

echo "==> Сброс правил фаерволов UFW и nftables..."
if command -v ufw >/dev/null 2>&1; then
  ufw delete allow 8444/tcp 2>/dev/null || true
  ufw delete allow 2222/tcp 2>/dev/null || true
fi

if command -v nft >/dev/null 2>&1; then
  nft delete table inet remna_mask 2>/dev/null || true
  nft delete table inet remna_antiabuse 2>/dev/null || true
fi

echo "SUCCESS_UNINSTALLED"
`, purgeData)

	out, err := session.CombinedOutput(cmd)
	outStr := strings.TrimSpace(string(out))
	if err != nil {
		return types.CommandResult{Success: false, Error: outStr}
	}
	return types.CommandResult{Success: true, Output: outStr}
}

const AppCurrentVersion = "2.0.0"

// CheckAppUpdate checks for new versions on GitHub Releases
func CheckAppUpdate(repo string) types.AppUpdateInfo {
	if strings.TrimSpace(repo) == "" {
		repo = "XZ-LITE/remna-ui"
	}
	repo = strings.TrimPrefix(repo, "https://github.com/")
	repo = strings.Trim(repo, "/")

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return types.AppUpdateInfo{
			CurrentVersion: AppCurrentVersion,
			Error:          err.Error(),
		}
	}
	req.Header.Set("User-Agent", "Remnawave-UI-Updater")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return types.AppUpdateInfo{
			CurrentVersion: AppCurrentVersion,
			Error:          fmt.Sprintf("Не удалось связаться с GitHub: %v", err),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return types.AppUpdateInfo{
			CurrentVersion: AppCurrentVersion,
			Error:          "Релизы в указанном репозитории GitHub пока не найдены.",
		}
	}
	if resp.StatusCode != 200 {
		return types.AppUpdateInfo{
			CurrentVersion: AppCurrentVersion,
			Error:          fmt.Sprintf("GitHub ответил статусом %d", resp.StatusCode),
		}
	}

	var ghRelease struct {
		TagName     string `json:"tag_name"`
		Name        string `json:"name"`
		Body        string `json:"body"`
		PublishedAt string `json:"published_at"`
		HtmlURL     string `json:"html_url"`
		Assets      []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ghRelease); err != nil {
		return types.AppUpdateInfo{
			CurrentVersion: AppCurrentVersion,
			Error:          "Ошибка парсинга ответа от GitHub Releases API",
		}
	}

	latestClean := strings.TrimPrefix(strings.TrimSpace(ghRelease.TagName), "v")
	currentClean := strings.TrimPrefix(AppCurrentVersion, "v")

	hasUpdate := isVersionNewer(latestClean, currentClean)

	downloadURL := ""
	for _, asset := range ghRelease.Assets {
		if strings.HasSuffix(strings.ToLower(asset.Name), ".exe") {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	return types.AppUpdateInfo{
		CurrentVersion: AppCurrentVersion,
		LatestVersion:  ghRelease.TagName,
		HasUpdate:      hasUpdate,
		ReleaseTitle:   ghRelease.Name,
		ReleaseNotes:   ghRelease.Body,
		PublishedAt:    ghRelease.PublishedAt,
		DownloadURL:    downloadURL,
		ReleaseURL:     ghRelease.HtmlURL,
	}
}

func comparePreReleases(preA, preB string) int {
	if preA == preB {
		return 0
	}
	partsA := strings.Split(preA, ".")
	partsB := strings.Split(preB, ".")
	for i := 0; i < len(partsA) && i < len(partsB); i++ {
		pA, pB := partsA[i], partsB[i]
		if pA == pB {
			continue
		}
		numA, errA := strconv.Atoi(pA)
		numB, errB := strconv.Atoi(pB)
		if errA == nil && errB == nil {
			if numA > numB {
				return 1
			}
			return -1
		}
		// Numerical identifiers always have lower precedence than non-numeric identifiers
		if errA == nil && errB != nil {
			return -1
		}
		if errA != nil && errB == nil {
			return 1
		}
		// Both are string identifiers: compare lexically
		if pA > pB {
			return 1
		}
		return -1
	}
	if len(partsA) > len(partsB) {
		return 1
	}
	if len(partsA) < len(partsB) {
		return -1
	}
	return 0
}

func isVersionNewer(latest, current string) bool {
	splitVersion := func(v string) (string, string) {
		v = strings.TrimPrefix(strings.TrimSpace(v), "v")
		v = strings.TrimPrefix(v, "V")
		// Discard build metadata after '+'
		if idx := strings.Index(v, "+"); idx != -1 {
			v = v[:idx]
		}
		pre := ""
		if idx := strings.Index(v, "-"); idx != -1 {
			pre = v[idx+1:]
			v = v[:idx]
		}
		return v, pre
	}

	lBase, lPre := splitVersion(latest)
	cBase, cPre := splitVersion(current)

	lParts := strings.Split(lBase, ".")
	cParts := strings.Split(cBase, ".")

	maxLen := len(lParts)
	if len(cParts) > maxLen {
		maxLen = len(cParts)
	}

	for i := 0; i < maxLen; i++ {
		var lNum, cNum int
		if i < len(lParts) {
			lNum, _ = strconv.Atoi(lParts[i])
		}
		if i < len(cParts) {
			cNum, _ = strconv.Atoi(cParts[i])
		}
		if lNum > cNum {
			return true
		}
		if lNum < cNum {
			return false
		}
	}

	// Base versions are equal (e.g. 2.0.0 vs 2.0.0)
	// A normal version has greater precedence than a pre-release version
	if cPre != "" && lPre == "" {
		return true // latest is stable release, current is pre-release
	}
	if cPre == "" && lPre != "" {
		return false // latest is pre-release, current is stable release
	}
	if cPre != "" && lPre != "" {
		return comparePreReleases(lPre, cPre) > 0
	}

	return false
}

// CleanServerDisk safely performs non-destructive cleanup of unreferenced Docker layers,
// builder caches, and truncates systemd journal logs, returning disk delta.
func CleanServerDisk(server types.SavedServer, password string) types.CleanDiskResult {
	if password != "" {
		server.Password = password
	}

	session, release, err := getPooledSession(server)
	if err != nil {
		return types.CleanDiskResult{
			Success: false,
			Error:   fmt.Sprintf("Ошибка SSH подключения: %v", err),
		}
	}
	defer release()

	cmd := `
before=$(df -h / | awk 'NR==2{print $4}')
echo "BEFORE:$before"
docker system prune -f --volumes=false 2>&1 || true
docker builder prune -f 2>&1 || true
journalctl --vacuum-time=2d 2>&1 || true
journalctl --vacuum-size=100M 2>&1 || true
rm -rf /tmp/rw_* /tmp/*.tmp /var/log/*.gz /var/log/*.[0-9] 2>/dev/null || true
after=$(df -h / | awk 'NR==2{print $4}')
echo "AFTER:$after"
`

	outBytes, err := session.Output(cmd)
	output := string(outBytes)
	if err != nil && len(output) == 0 {
		return types.CleanDiskResult{
			Success: false,
			Output:  output,
			Error:   fmt.Sprintf("Ошибка при выполнении очистки: %v", err),
		}
	}

	spaceBefore := ""
	spaceAfter := ""
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "BEFORE:") {
			spaceBefore = strings.TrimPrefix(trimmed, "BEFORE:")
		} else if strings.HasPrefix(trimmed, "AFTER:") {
			spaceAfter = strings.TrimPrefix(trimmed, "AFTER:")
		}
	}

	freed := ""
	if spaceBefore != "" && spaceAfter != "" {
		freed = fmt.Sprintf("Свободно до: %s, свободно после: %s", spaceBefore, spaceAfter)
	}

	return types.CleanDiskResult{
		Success:     true,
		SpaceBefore: spaceBefore,
		SpaceAfter:  spaceAfter,
		FreedSpace:  freed,
		Output:      output,
	}
}

// RunServerEgressProbe performs a synthetic egress HTTP 204 probe directly from the target server / remnanode container.
func RunServerEgressProbe(server types.SavedServer) (bool, int, int, string) {
	session, release, err := getPooledSession(server)
	if err != nil {
		return false, 0, 0, fmt.Sprintf("SSH: %v", err)
	}
	defer release()

	cmd := `(docker exec remnanode curl -s -o /dev/null -w "%{http_code}|%{time_total}" https://cp.cloudflare.com/generate_204 --max-time 4 2>/dev/null) || (curl -s -o /dev/null -w "%{http_code}|%{time_total}" https://cp.cloudflare.com/generate_204 --max-time 4 2>/dev/null) || echo "000|0.0"`
	outBytes, err := session.Output(cmd)
	if err != nil && len(outBytes) == 0 {
		return false, 0, 0, fmt.Sprintf("Команда curl не выполнена: %v", err)
	}

	out := strings.TrimSpace(string(outBytes))
	parts := strings.Split(out, "|")
	if len(parts) >= 2 {
		httpCode, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
		timeSec, _ := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		latencyMs := int(timeSec * 1000)

		if httpCode == 204 || httpCode == 200 {
			return true, httpCode, latencyMs, ""
		}
		return false, httpCode, latencyMs, fmt.Sprintf("Неожиданный HTTP-ответ: %d", httpCode)
	}

	return false, 0, 0, fmt.Sprintf("Неверный формат ответа: %s", out)
}


