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

	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"time"

	"unsafe"



	"golang.org/x/crypto/ssh"

	"remna-ui/internal/generator"
	"remna-ui/internal/types"

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

	if secret == "" || strings.HasPrefix(secret, "enc:dpapi:") {

		return secret

	}

	if runtime.GOOS != "windows" {

		return secret

	}

	bytes := []byte(secret)

	var inBlob, outBlob dataBlob

	inBlob.cbData = uint32(len(bytes))

	if len(bytes) > 0 {

		inBlob.pbData = &bytes[0]

	}

	r, _, _ := procCryptProtectData.Call(

		uintptr(unsafe.Pointer(&inBlob)),

		0,

		0,

		0,

		0,

		0x01, // CRYPTPROTECT_UI_FORBIDDEN

		uintptr(unsafe.Pointer(&outBlob)),

	)

	if r == 0 || outBlob.pbData == nil {

		return secret

	}

	defer syscall.LocalFree(syscall.Handle(unsafe.Pointer(outBlob.pbData)))

	outBytes := make([]byte, outBlob.cbData)

	for i := uint32(0); i < outBlob.cbData; i++ {

		outBytes[i] = *(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(outBlob.pbData)) + uintptr(i)))

	}

	return "enc:dpapi:" + base64.StdEncoding.EncodeToString(outBytes)

}



func decryptSecret(encrypted string) string {

	if !strings.HasPrefix(encrypted, "enc:dpapi:") {

		return encrypted

	}

	if runtime.GOOS != "windows" {

		return encrypted

	}

	rawB64 := strings.TrimPrefix(encrypted, "enc:dpapi:")

	data, err := base64.StdEncoding.DecodeString(rawB64)

	if err != nil || len(data) == 0 {

		return encrypted

	}

	var inBlob, outBlob dataBlob

	inBlob.cbData = uint32(len(data))

	inBlob.pbData = &data[0]

	r, _, _ := procCryptUnprotectData.Call(

		uintptr(unsafe.Pointer(&inBlob)),

		0,

		0,

		0,

		0,

		0x01, // CRYPTPROTECT_UI_FORBIDDEN

		uintptr(unsafe.Pointer(&outBlob)),

	)

	if r == 0 || outBlob.pbData == nil {

		return encrypted

	}

	defer syscall.LocalFree(syscall.Handle(unsafe.Pointer(outBlob.pbData)))

	outBytes := make([]byte, outBlob.cbData)

	for i := uint32(0); i < outBlob.cbData; i++ {

		outBytes[i] = *(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(outBlob.pbData)) + uintptr(i)))

	}

	return string(outBytes)

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
		if fi, err := os.Stat(target); os.IsNotExist(err) || fi.Size() == 0 {
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
	for i := range servers {
		if strings.HasPrefix(servers[i].Password, "enc:dpapi:") {
			servers[i].Password = decryptSecret(servers[i].Password)
		}
		if strings.HasPrefix(servers[i].AdminPassword, "enc:dpapi:") {
			servers[i].AdminPassword = decryptSecret(servers[i].AdminPassword)
		}
		if strings.HasPrefix(servers[i].SSHKey, "enc:dpapi:") {
			servers[i].SSHKey = decryptSecret(servers[i].SSHKey)
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
		if toSave[i].Password != "" && !strings.HasPrefix(toSave[i].Password, "enc:dpapi:") {
			toSave[i].Password = encryptSecret(toSave[i].Password)
		}
		if toSave[i].AdminPassword != "" && !strings.HasPrefix(toSave[i].AdminPassword, "enc:dpapi:") {
			toSave[i].AdminPassword = encryptSecret(toSave[i].AdminPassword)
		}
		if toSave[i].SSHKey != "" && !strings.HasPrefix(toSave[i].SSHKey, "enc:dpapi:") {
			toSave[i].SSHKey = encryptSecret(toSave[i].SSHKey)
		}
	}

	data, err := json.MarshalIndent(toSave, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(getStorePath(), data, 0600)
}

func DeleteServer(id string) error {
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
		if toSave[i].Password != "" && !strings.HasPrefix(toSave[i].Password, "enc:dpapi:") {
			toSave[i].Password = encryptSecret(toSave[i].Password)
		}
		if toSave[i].AdminPassword != "" && !strings.HasPrefix(toSave[i].AdminPassword, "enc:dpapi:") {
			toSave[i].AdminPassword = encryptSecret(toSave[i].AdminPassword)
		}
		if toSave[i].SSHKey != "" && !strings.HasPrefix(toSave[i].SSHKey, "enc:dpapi:") {
			toSave[i].SSHKey = encryptSecret(toSave[i].SSHKey)
		}
	}



	data, err := json.MarshalIndent(toSave, "", "  ")

	if err != nil {

		return err

	}

	return os.WriteFile(getStorePath(), data, 0600)

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

	if strings.HasPrefix(effectivePass, "enc:dpapi:") {

		effectivePass = decryptSecret(effectivePass)

	}



	trimmedKey := strings.Trim(strings.TrimSpace(server.SSHKey), `"'`)

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

	sshPoolMu.Lock()
	if entry, exists := sshPool[key]; exists && entry.client != nil {
		// Verify connection health with a lightweight keepalive request
		_, _, err := entry.client.SendRequest("keepalive@openssh.com", true, nil)
		if err == nil {
			session, sessErr := entry.client.NewSession()
			if sessErr == nil {
				entry.inUse++
				entry.lastUsed = time.Now()
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
		// Connection failed or session cannot be opened; clean up stale client if not currently in use
		if entry.inUse == 0 {
			entry.client.Close()
			delete(sshPool, key)
		}
	}
	sshPoolMu.Unlock()

	// Dial fresh client outside of global lock to prevent blocking concurrent dials to different hosts
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
	if old, exists := sshPool[key]; exists && old.client != nil && old.inUse == 0 {
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



	var cmd string
	if server.Role == "node" {
		// Lightweight interrogation optimized for Edge Nodes: avoids heavy & non-existent Postgres/Panel checks
		cmd = `grep -E '^PRETTY_NAME=' /etc/os-release 2>/dev/null | cut -d= -f2 | tr -d '"'; echo "---"; free -m | awk '/^Mem:/{print $3 "|" $2}'; echo "---"; df -h / | awk 'NR==2{print $3 "|" $2 "|" $5}'; echo "---"; uptime -p 2>/dev/null || uptime; echo "---"; docker ps -a --format '{{.Names}}\t{{.Status}}\t{{.Image}}' 2>/dev/null || true; echo "---"; sysctl -n net.ipv4.tcp_congestion_control 2>/dev/null || echo "unknown"; echo "---"; [ -f /etc/docker/daemon.json ] && grep -q "registry-mirrors" /etc/docker/daemon.json && echo "yes" || echo "no"; echo "---"; echo ""; echo "---"; echo "0"; echo "---"; echo "0"; echo "---"; (command -v warp-cli >/dev/null 2>&1 && warp-cli --accept-tos status 2>/dev/null | grep -qi "connected" && echo "yes") || echo "no"; echo "---"; echo "no"`
	} else {
		// Comprehensive interrogation for Master Panel VPS (includes Postgres DB, Backups, and Valkey Sockets)
		cmd = `grep -E '^PRETTY_NAME=' /etc/os-release 2>/dev/null | cut -d= -f2 | tr -d '"'; echo "---"; free -m | awk '/^Mem:/{print $3 "|" $2}'; echo "---"; df -h / | awk 'NR==2{print $3 "|" $2 "|" $5}'; echo "---"; uptime -p 2>/dev/null || uptime; echo "---"; docker ps -a --format '{{.Names}}\t{{.Status}}\t{{.Image}}' 2>/dev/null || true; echo "---"; sysctl -n net.ipv4.tcp_congestion_control 2>/dev/null || echo "unknown"; echo "---"; [ -f /etc/docker/daemon.json ] && grep -q "registry-mirrors" /etc/docker/daemon.json && echo "yes" || echo "no"; echo "---"; ls -lh /opt/remnawave/backups/*.tar.gz 2>/dev/null | awk '{print $9 "|" $5 "|" $6 " " $7 " " $8}' || true; echo "---"; docker exec -u postgres remnawave-db psql -U postgres -d postgres -t -A -c "SELECT count(*) FROM node_plugin np JOIN nodes n ON n.active_plugin_uuid = np.uuid WHERE (np.plugin_config->'torrentBlocker'->>'enabled')::boolean = true;" 2>/dev/null || echo "0"; echo "---"; docker exec -u postgres remnawave-db psql -U postgres -d postgres -t -A -c "SELECT count(*) FROM nodes;" 2>/dev/null || echo "0"; echo "---"; (command -v warp-cli >/dev/null 2>&1 && warp-cli --accept-tos status 2>/dev/null | grep -qi "connected" && echo "yes") || echo "no"; echo "---"; ([ -f /opt/remnawave/panel/docker-compose.yml ] && grep -q "shm_size: 512mb" /opt/remnawave/panel/docker-compose.yml && grep -q "valkey.sock" /opt/remnawave/panel/docker-compose.yml && echo "yes") || ([ -f /opt/remnawave/docker-compose.yml ] && grep -q "shm_size: 512mb" /opt/remnawave/docker-compose.yml && grep -q "valkey.sock" /opt/remnawave/docker-compose.yml && echo "yes") || echo "no"`
	}

	out, err := session.Output(cmd)

	latency := int(time.Since(start).Milliseconds())

	if err != nil {
		RecordServerInspectionResult(server.ID, false)
		return types.ServerStatusResult{Success: false, Error: err.Error()}
	}



	parts := strings.Split(string(out), "---")

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

DB_PASS=$(grep -E '^POSTGRES_PASSWORD=' /opt/remnawave/panel/.env 2>/dev/null | cut -d= -f2- | tr -d '"\r\n')

if [ -z "$DB_PASS" ]; then

  echo "Файл /opt/remnawave/panel/.env не найден"

  exit 1

fi

if [ -d /opt/remnawave/subscription ] && [ -f /opt/remnawave/subscription/docker-compose.yml ]; then

  echo ">>> Обновление образов страницы подписки..."

  if [ -f /opt/remnawave/subscription/.env ]; then
    grep -q 'REMNAWAVE_PANEL_URL' /opt/remnawave/subscription/.env || echo 'REMNAWAVE_PANEL_URL=http://remnawave:3000' >> /opt/remnawave/subscription/.env
  fi

  cd /opt/remnawave/subscription && docker compose pull && docker compose up -d --remove-orphans

  UPDATED_COUNT=$((UPDATED_COUNT + 1))

fi

DB_PASS_ESC=$(printf '%s' "$DB_PASS" | sed "s/'/''/g")

docker exec remnawave-db psql -U postgres -d postgres -c "ALTER USER postgres WITH PASSWORD '$DB_PASS_ESC';"

docker compose -f /opt/remnawave/panel/docker-compose.yml restart remnawave

for i in $(seq 1 25); do

  if docker logs remnawave --tail 15 2>&1 | grep -q 'Nest application successfully started'; then

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



	cmd := fmt.Sprintf(`mkdir -p /opt/remnawave/node && echo '%s' | base64 -d > /opt/remnawave/node/docker-compose.yml

command -v ufw >/dev/null 2>&1 && ufw allow 2222/tcp >/dev/null 2>&1 || true

cd /opt/remnawave/node && docker compose down 2>/dev/null || true

cd /opt/remnawave/node && docker compose up -d

sleep 2

docker compose ps

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

mkdir -p /opt/remnawave/backups

STAMP=$(date +%Y%m%d_%H%M%S)

BACKUP_NAME="remnawave_backup_${STAMP}.tar.gz"

TEMP_DIR=$(mktemp -d)



# Dump PostgreSQL if container is running

if docker ps --format '{{.Names}}' | grep -q 'remnawave-db'; then

  docker exec -u postgres remnawave-db pg_dump -U postgres postgres > "$TEMP_DIR/database.sql" 2>/dev/null || true

fi



# Copy all .env and docker-compose configs

mkdir -p "$TEMP_DIR/configs"

cp -rf /opt/remnawave/panel/.env "$TEMP_DIR/configs/panel.env" 2>/dev/null || true

cp -rf /opt/remnawave/panel/docker-compose.yml "$TEMP_DIR/configs/panel-compose.yml" 2>/dev/null || true

cp -rf /opt/remnawave/node/docker-compose.yml "$TEMP_DIR/configs/node-compose.yml" 2>/dev/null || true

cp -rf /opt/remnawave/nginx/nginx.conf "$TEMP_DIR/configs/nginx.conf" 2>/dev/null || true

cp -rf /opt/remnawave/nginx/fullchain.pem "$TEMP_DIR/configs/" 2>/dev/null || true

cp -rf /opt/remnawave/nginx/privkey.key "$TEMP_DIR/configs/" 2>/dev/null || true

cp -rf /opt/remnawave/caddy/Caddyfile "$TEMP_DIR/configs/Caddyfile" 2>/dev/null || true

cp -rf /opt/remnawave/caddy/docker-compose.yml "$TEMP_DIR/configs/caddy-compose.yml" 2>/dev/null || true



tar -czf "/opt/remnawave/backups/$BACKUP_NAME" -C "$TEMP_DIR" .

rm -rf "$TEMP_DIR"

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

docker exec -u postgres remnawave-db psql -U postgres -d postgres -c "TRUNCATE TABLE admin CASCADE;" 2>/dev/null || true

docker compose -f /opt/remnawave/panel/docker-compose.yml restart remnawave-redis remnawave >/dev/null 2>&1 || true

for i in $(seq 1 25); do

  if docker logs remnawave --tail 15 2>&1 | grep -q 'Nest application successfully started'; then

    break

  fi

  sleep 1

done

echo "Супер-администратор сброшен. Панель готова к созданию нового администратора!"

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

if [ -x ~/.acme.sh/acme.sh ]; then

  echo ">>> Обновление SSL сертификатов через acme.sh..."

  ~/.acme.sh/acme.sh --renew-all --force 2>&1 || true

  ACME_DOMAIN=$(ls -1 ~/.acme.sh/ 2>/dev/null | grep -v 'ca\|acme.sh\|deploy' | head -n 1)

  if [ -n "$ACME_DOMAIN" ] && [ -f ~/.acme.sh/$ACME_DOMAIN/fullchain.cer ]; then

    cp -fL ~/.acme.sh/$ACME_DOMAIN/fullchain.cer /opt/remnawave/nginx/fullchain.pem 2>/dev/null || true

    cp -fL ~/.acme.sh/$ACME_DOMAIN/$ACME_DOMAIN.key /opt/remnawave/nginx/privkey.key 2>/dev/null || true

  fi

  docker restart remnawave-nginx 2>/dev/null || true

  echo "SSL сертификаты Let's Encrypt (acme.sh) успешно обновлены!"

elif command -v docker >/dev/null 2>&1 && [ -d /etc/letsencrypt/live ]; then

  docker stop remnawave-nginx 2>/dev/null || true

  echo ">>> Обновление SSL сертификатов через Docker Certbot..."

  docker run --rm -p 80:80 \
    -v /etc/letsencrypt:/etc/letsencrypt \
    -v /var/lib/letsencrypt:/var/lib/letsencrypt \
    certbot/certbot renew --non-interactive 2>&1 || true

  DOMAIN=$(ls -1 /etc/letsencrypt/live/ 2>/dev/null | grep -v 'README' | head -n 1)

  if [ -n "$DOMAIN" ] && [ -f "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" ]; then

    cp -fL "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" /opt/remnawave/nginx/fullchain.pem 2>/dev/null || true

    cp -fL "/etc/letsencrypt/live/$DOMAIN/privkey.pem" /opt/remnawave/nginx/privkey.key 2>/dev/null || true

  fi

  docker start remnawave-nginx 2>/dev/null || true

  echo "SSL сертификаты Let's Encrypt (Docker Certbot) успешно обновлены!"

elif command -v certbot >/dev/null 2>&1; then

  docker stop remnawave-nginx 2>/dev/null || true

  certbot renew --non-interactive 2>&1

  DOMAIN=$(ls -1 /etc/letsencrypt/live/ 2>/dev/null | grep -v 'README' | head -n 1)

  if [ -n "$DOMAIN" ] && [ -f "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" ]; then

    cp -fL "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" /opt/remnawave/nginx/fullchain.pem 2>/dev/null || true

    cp -fL "/etc/letsencrypt/live/$DOMAIN/privkey.pem" /opt/remnawave/nginx/privkey.key 2>/dev/null || true

  fi

  docker start remnawave-nginx 2>/dev/null || true

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



func RestoreBackup(server types.SavedServer, fileName string) types.CommandResult {

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



	cmd := fmt.Sprintf(`

RESTORE_FILE="/opt/remnawave/backups/%s"

if [ ! -f "$RESTORE_FILE" ]; then

  echo "Файл бэкапа не найден: $RESTORE_FILE"

  exit 1

fi



TEMP_DIR=$(mktemp -d)

tar -xzf "$RESTORE_FILE" -C "$TEMP_DIR"



# Восстанавливаем конфигурационные файлы

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



# Если есть дамп базы данных, восстанавливаем его в PostgreSQL

if [ -f "$TEMP_DIR/database.sql" ]; then

  if [ -f /opt/remnawave/panel/docker-compose.yml ]; then

    cd /opt/remnawave/panel && docker compose up -d remnawave-db >/dev/null 2>&1 || true

    sleep 3

    docker exec -i remnawave-db psql -U postgres postgres < "$TEMP_DIR/database.sql" >/dev/null 2>&1 || true

  fi

fi



rm -rf "$TEMP_DIR"



# Перезапускаем контейнеры

if [ -f /opt/remnawave/panel/docker-compose.yml ]; then

  cd /opt/remnawave/panel && docker compose restart >/dev/null 2>&1 || true

fi

if [ -f /opt/remnawave/node/docker-compose.yml ]; then

  cd /opt/remnawave/node && docker compose restart >/dev/null 2>&1 || true

fi

if [ -f /opt/remnawave/caddy/docker-compose.yml ]; then

  cd /opt/remnawave/caddy && docker compose restart >/dev/null 2>&1 || true

fi

docker restart remna-nginx remnawave-nginx remnawave-caddy >/dev/null 2>&1 || true



echo "Резервная копия успешно восстановлена, службы перезапущены!"

`, fileName)



	out, err := session.CombinedOutput(cmd)

	if err != nil {

		return types.CommandResult{Success: false, Error: string(out)}

	}

	return types.CommandResult{Success: true, Output: strings.TrimSpace(string(out))}

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

    # 2. Если мастер-токен отсутствует, считываем пароль администратора
    if [ -z "$ACCESS_TOKEN" ]; then
      ADMIN_USER="admin"
      ADMIN_PASS=""
      for cred_file in /opt/remnawave/credentials.txt /opt/remnawave/.env /opt/remnawave/panel/.env; do
        if [ -f "$cred_file" ]; then
          p=$(grep -E '^ADMIN_PASSWORD=|^INIT_ADMIN_PASSWORD=' "$cred_file" 2>/dev/null | cut -d= -f2- | tr -d '"\r\n' | head -n1)
          if [ -n "$p" ]; then ADMIN_PASS="$p"; break; fi
        fi
      done

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

NODE_DIR="/opt/remnawave/node"

if [ ! -d "$NODE_DIR" ]; then

  echo "Каталог $NODE_DIR не обнаружен. Установите компонент Remnanode на этом сервере."

  exit 1

fi



if docker ps --format '{{.Names}}' | grep -q "remnanode"; then

  echo "Нода remnanode активна."

  echo "Архитектура SelfSteal готова: используйте dest/fallback = %s в настройках ноды в панели Remnawave."

  echo "При подозрительном трафике или активном зондировании Xray отдает локальный маскировочный сайт."

else

  echo "Контейнер remnanode не запущен."

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
  ADMIN_USER="admin"
  ADMIN_PASS=""
  for cred_file in /opt/remnawave/credentials.txt /opt/remnawave/.env /opt/remnawave/panel/.env; do
    if [ -f "$cred_file" ]; then
      p=$(grep -E '^ADMIN_PASSWORD=|^INIT_ADMIN_PASSWORD=' "$cred_file" 2>/dev/null | cut -d= -f2- | tr -d '"\r\n' | head -n1)
      if [ -n "$p" ]; then ADMIN_PASS="$p"; break; fi
    fi
  done

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
for c in $(docker ps -a --format '{{.Names}}' 2>/dev/null | grep -E 'remna|caddy|nginx'); do
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

echo "==> Сброс правил фаервола UFW..."
if command -v ufw >/dev/null 2>&1; then
  ufw delete allow 8444/tcp 2>/dev/null || true
  ufw delete allow 2222/tcp 2>/dev/null || true
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
