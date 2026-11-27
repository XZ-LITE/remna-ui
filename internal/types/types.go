package types

type DeploymentPlan struct {
	Components         []string `json:"components"`
	Domain             string   `json:"domain"`
	PanelDomain        string   `json:"panel_domain"`
	SubscriptionDomain string   `json:"subscription_domain"`
	NodePort           int      `json:"node_port"`
	NodeSecret         string   `json:"node_secret"`
	Proxy              string   `json:"proxy"`
	SubscriptionMode   string   `json:"subscription_mode"`
	SecretGate         string   `json:"secret_gate"`
	UseCamouflage      bool     `json:"use_camouflage"`
	UfwSetup           bool     `json:"ufw_setup"`
	SshHost            string   `json:"ssh_host"`
	SshPort            int      `json:"ssh_port"`
	SshUser            string   `json:"ssh_user"`
	SshPassword        string   `json:"ssh_password"`
	SshKey             string   `json:"ssh_key"`
	OutputDir          string   `json:"output_dir"`
	DeploymentMode     string   `json:"deployment_mode"` // "generate" or "remote"
	UseCloudflareDns   bool     `json:"use_cloudflare_dns"`
	CloudflareApiToken string   `json:"cloudflare_api_token"`
	PanelIP            string   `json:"panel_ip"`
	HideNodePort       bool     `json:"hide_node_port"`
	StealthDrop        bool     `json:"stealth_drop"`
	DeployDecoy        bool     `json:"deploy_decoy"`
	DecoyDomain        string   `json:"decoy_domain"`
	PortMode           string   `json:"port_mode"`            // "stealth443" or "classic"
	DecoyTemplate      string   `json:"decoy_template"`       // "random", "simple", "sni", "nothing", "clone"
	DecoyCloneURL      string   `json:"decoy_clone_url"`      // URL to clone if chosen
	TorrentBlocker     bool     `json:"torrent_blocker"`      // Block BitTorrent in Xray
	BlockSpamPorts     bool     `json:"block_spam_ports"`     // Block SMTP 25, 465, 587
	XrayCoreType            string   `json:"xray_core_type"`            // "official" or "jolymmiles"
	EnableTinyAuth          bool     `json:"enable_tinyauth"`           // Optional TinyAuth layer
	PerformanceOptimization bool     `json:"performance_optimization"`  // Valkey Unix Domain Sockets & Postgres 512MB RAM
	AdminCreationMode       string   `json:"admin_creation_mode"`       // "auto", "custom", "manual"
	AdminUsername           string   `json:"admin_username"`            // custom admin username
	AdminPassword           string   `json:"admin_password"`            // custom admin password
}

type DNSResult struct {
	Resolved bool   `json:"resolved"`
	IP       string `json:"ip"`
	DoHIP    string `json:"doh_ip,omitempty"`
	Spoofed  bool   `json:"spoofed,omitempty"`
	Message  string `json:"message"`
}

type SSHTestResult struct {
	Success bool   `json:"success"`
	OS      string `json:"os"`
	CPU     string `json:"cpu"`
	RAM     string `json:"ram"`
	Error   string `json:"error"`
}

type GenerateResult struct {
	Success bool     `json:"success"`
	Files   []string `json:"files"`
	OutDir  string   `json:"out_dir"`
	Error   string   `json:"error"`
}

type RemoteDeployResult struct {
	Success           bool   `json:"success"`
	AdminCreationMode string `json:"admin_creation_mode"`
	AdminUser         string `json:"admin_user"`
	AdminPass         string `json:"admin_pass"`
	PanelURL          string `json:"panel_url"`
	SubURL            string `json:"sub_url"`
	Error             string `json:"error"`
}

type SavedServer struct {
	ID               string `json:"id"`
	Label            string `json:"label"`
	Host             string `json:"host"`
	Port             int    `json:"port"`
	Username         string `json:"username"`
	Password         string `json:"password"`
	SSHKey           string `json:"ssh_key"`
	WebURL           string `json:"web_url"`
	Role             string `json:"role"` // "master" or "node"
	ParentMasterID   string `json:"parent_master_id"` // ID of parent master panel if node
	LocationCode     string `json:"location_code"` // e.g. "NL", "DE", "FI"
	DecoyDomain      string `json:"decoy_domain"` // e.g. "node-nl.domain.com"
	IsExternalMaster bool   `json:"is_external_master"` // true if master installed manually
	LastSeen         string `json:"last_seen"`
	AdminUser        string `json:"admin_user"`
	AdminPassword    string `json:"admin_password"`
	SubURL           string `json:"sub_url"`
	PortMode         string `json:"port_mode"`
	XrayCoreType     string `json:"xray_core_type"`
	ApiToken         string `json:"api_token,omitempty"`
	NodeSecret       string `json:"node_secret,omitempty"`
}

type VaultStatus struct {
	Enabled           bool   `json:"enabled"`
	Unlocked          bool   `json:"unlocked"`
	Mode              string `json:"mode"`                         // "dpapi" or "master_password"
	ProtectionBackend string `json:"protection_backend,omitempty"` // "dpapi", "keychain", "secret_service", "local_machine", "master_password"
	AutoLockMinutes   int    `json:"auto_lock_minutes"`
	HasSavedServers   bool   `json:"has_saved_servers"`
	Message           string `json:"message,omitempty"`
}

type ContainerInfo struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Image   string `json:"image"`
	Running bool   `json:"running"`
	Healthy string `json:"healthy,omitempty"`
}

type AppUpdateInfo struct {
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	HasUpdate      bool   `json:"has_update"`
	ReleaseTitle   string `json:"release_title"`
	ReleaseNotes   string `json:"release_notes"`
	PublishedAt    string `json:"published_at"`
	DownloadURL    string `json:"download_url"`
	ReleaseURL     string `json:"release_url"`
	Error          string `json:"error,omitempty"`
}

type ServerMetrics struct {
	Uptime      string  `json:"uptime"`
	RAMUsedMB   int     `json:"ram_used_mb"`
	RAMTotalMB  int     `json:"ram_total_mb"`
	RAMPercent  int     `json:"ram_percent"`
	DiskUsed    string  `json:"disk_used"`
	DiskTotal   string  `json:"disk_total"`
	DiskPercent int     `json:"disk_percent"`
	CPUSteal    float64 `json:"cpu_steal"`
}

type BackupInfo struct {
	FileName  string `json:"file_name"`
	Size      string `json:"size"`
	CreatedAt string `json:"created_at"`
}

type RestoreBackupResult struct {
	Success          bool   `json:"success"`
	SafetyBackupFile string `json:"safety_backup_file,omitempty"`
	ScopeApplied     string `json:"scope_applied"`
	Output           string `json:"output"`
	Error            string `json:"error,omitempty"`
}

type ServerStatusResult struct {
	Success             bool            `json:"success"`
	HostInfo            string          `json:"host_info"`
	Metrics             ServerMetrics   `json:"metrics"`
	Containers          []ContainerInfo `json:"containers"`
	BBRActive           bool            `json:"bbr_active"`
	DockerMirrorsActive bool            `json:"docker_mirrors_active"`
	AntiAbuseActive     bool            `json:"anti_abuse_active"`
	WarpActive          bool            `json:"warp_active"`
	PerformanceOptimized bool           `json:"performance_optimized"`
	NodePresent         bool            `json:"node_present"`
	LatencyMs           int             `json:"latency_ms"`
	Backups             []BackupInfo    `json:"backups"`
	Error               string          `json:"error"`
}

type CommandResult struct {
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Error   string `json:"error"`
}

type PanelInfoResult struct {
	Success    bool   `json:"success"`
	Version    string `json:"version"`
	NodesCount int    `json:"nodes_count"`
	Error      string `json:"error"`
}

type CreateNodeRequest struct {
	Name              string  `json:"name"`
	Country           string  `json:"country"`
	Address           string  `json:"address"`
	Port              int     `json:"port"`
	TrafficMultiplier float64 `json:"traffic_multiplier"`
	DecoyDomain       string  `json:"decoy_domain"`
	PortMode          string  `json:"port_mode"`
	TorrentBlocker    bool    `json:"torrent_blocker"`
	BlockSpamPorts    bool    `json:"block_spam_ports"`
	IntegrationMode   string  `json:"integration_mode"` // "turnkey" or "custom"
	ConfigProfileUUID string  `json:"config_profile_uuid"`
	InboundUUID       string  `json:"inbound_uuid"`
	InboundPort       int     `json:"inbound_port"`
	SquadUUID         string  `json:"squad_uuid"`
	CreateHost        bool    `json:"create_host"`
	HostAddress       string  `json:"host_address"`
	HostSNI           string  `json:"host_sni"`
}

type CreateNodeResponse struct {
	Success      bool   `json:"success"`
	NodeUUID     string `json:"node_uuid"`
	SecretKey    string `json:"secret_key"`
	HostCreated  bool   `json:"host_created"`
	SquadUpdated bool   `json:"squad_updated"`
	Error        string `json:"error"`
}

type PanelProfileItem struct {
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	InboundUUID string `json:"inbound_uuid"`
	InboundTag  string `json:"inbound_tag"`
	Port        int    `json:"port"`
	Protocol    string `json:"protocol"`
}

type PanelSquadItem struct {
	UUID          string   `json:"uuid"`
	Name          string   `json:"name"`
	InboundsCount int      `json:"inbounds_count"`
	InboundUUIDs  []string `json:"inbound_uuids"`
}

type PanelNodeItem struct {
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	Address     string `json:"address"`
	Port        int    `json:"port"`
	Country     string `json:"country"`
	IsConnected bool   `json:"is_connected"`
}

type NodeSecurityAudit struct {
	NodePortProtected bool     `json:"node_port_protected"`
	PanelWhitelisted  bool     `json:"panel_whitelisted"`
	DockerUserActive  bool     `json:"docker_user_active"`
	StealthDropActive bool     `json:"stealth_drop_active"`
	DecoyActive       bool     `json:"decoy_active"`
	BBRActive         bool     `json:"bbr_active"`
	ContainerRunning  bool     `json:"container_running"`
	Port443Listening  bool     `json:"port_443_listening"`
	XrayCoreType      string   `json:"xray_core_type"`
	OverallScore      int      `json:"overall_score"`
	Summary           string   `json:"summary"`
	Checks            []string `json:"checks"`
}

type ProbeNodeResult struct {
	NodeID      string `json:"node_id"`
	Location    string `json:"location"`
	CountryCode string `json:"country_code"`
	City        string `json:"city"`
	Success     bool   `json:"success"`
	LatencyMs   int    `json:"latency_ms"`
	Error       string `json:"error,omitempty"`
}

type PortProbeSummary struct {
	Port     int               `json:"port"`
	Protocol string            `json:"protocol"` // "SSH (управление)" или "VPN / Reality (трафик)"
	Status   string            `json:"status"`   // "clean" | "blocked" | "partial" | "offline"
	Probes   []ProbeNodeResult `json:"probes"`
}

type IPBlockCheckResult struct {
	Host          string              `json:"host"`
	Port          int                 `json:"port"`
	Status        string              `json:"status"` // "clean" | "blocked" | "partial" | "offline"
	Summary       string              `json:"summary"`
	ISPInfo       string              `json:"isp_info"`
	LocationInfo  string              `json:"location_info"`
	ClientCountry string              `json:"client_country,omitempty"`
	IsVPNDetected bool                `json:"is_vpn_detected"`
	Probes        []ProbeNodeResult   `json:"probes"`
	PortDetails   []PortProbeSummary  `json:"port_details,omitempty"`
	IsRKNBlocked  bool                `json:"is_rkn_blocked"`
	CheckedAt     string              `json:"checked_at"`
}

type CleanDiskResult struct {
	Success     bool   `json:"success"`
	SpaceBefore string `json:"space_before"`
	SpaceAfter  string `json:"space_after"`
	FreedSpace  string `json:"freed_space,omitempty"`
	Output      string `json:"output"`
	Error       string `json:"error,omitempty"`
}

type VPNProbeResult struct {
	Host           string `json:"host"`
	Port           int    `json:"port"`
	DecoyDomain    string `json:"decoy_domain"`
	TCPConnected   bool   `json:"tcp_connected"`
	TCPLatencyMs   int    `json:"tcp_latency_ms"`
	TCPError       string `json:"tcp_error,omitempty"`
	TLSHandshake   bool   `json:"tls_handshake"`
	TLSLatencyMs   int    `json:"tls_latency_ms"`
	TLSVersion     string `json:"tls_version,omitempty"`
	CipherSuite    string `json:"cipher_suite,omitempty"`
	IsTSPUDPIReset bool   `json:"is_tspu_dpi_reset"`
	TLSError       string `json:"tls_error,omitempty"`
	E2ESuccess     bool   `json:"e2e_success"`
	E2ELatencyMs   int    `json:"e2e_latency_ms"`
	E2EHTTPCode    int    `json:"e2e_http_code"`
	E2ETarget      string `json:"e2e_target"`
	E2EError       string `json:"e2e_error,omitempty"`
	Status         string `json:"status"` // "clean" | "dpi_blocked" | "offline" | "degraded"
	Summary        string `json:"summary"`
	Timestamp      string `json:"timestamp"`
}


