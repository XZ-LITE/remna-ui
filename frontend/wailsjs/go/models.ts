export namespace types {
	
	export class AppUpdateInfo {
	    current_version: string;
	    latest_version: string;
	    has_update: boolean;
	    release_title: string;
	    release_notes: string;
	    published_at: string;
	    download_url: string;
	    release_url: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new AppUpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.current_version = source["current_version"];
	        this.latest_version = source["latest_version"];
	        this.has_update = source["has_update"];
	        this.release_title = source["release_title"];
	        this.release_notes = source["release_notes"];
	        this.published_at = source["published_at"];
	        this.download_url = source["download_url"];
	        this.release_url = source["release_url"];
	        this.error = source["error"];
	    }
	}
	export class BackupInfo {
	    file_name: string;
	    size: string;
	    created_at: string;
	
	    static createFrom(source: any = {}) {
	        return new BackupInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file_name = source["file_name"];
	        this.size = source["size"];
	        this.created_at = source["created_at"];
	    }
	}
	export class CommandResult {
	    success: boolean;
	    output: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new CommandResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.output = source["output"];
	        this.error = source["error"];
	    }
	}
	export class ContainerInfo {
	    name: string;
	    status: string;
	    image: string;
	    running: boolean;
	    healthy?: string;
	
	    static createFrom(source: any = {}) {
	        return new ContainerInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.status = source["status"];
	        this.image = source["image"];
	        this.running = source["running"];
	        this.healthy = source["healthy"];
	    }
	}
	export class CreateNodeRequest {
	    name: string;
	    country: string;
	    address: string;
	    port: number;
	    traffic_multiplier: number;
	    decoy_domain: string;
	    port_mode: string;
	    torrent_blocker: boolean;
	    block_spam_ports: boolean;
	    integration_mode: string;
	    config_profile_uuid: string;
	    inbound_uuid: string;
	    squad_uuid: string;
	    create_host: boolean;
	    host_address: string;
	    host_sni: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateNodeRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.country = source["country"];
	        this.address = source["address"];
	        this.port = source["port"];
	        this.traffic_multiplier = source["traffic_multiplier"];
	        this.decoy_domain = source["decoy_domain"];
	        this.port_mode = source["port_mode"];
	        this.torrent_blocker = source["torrent_blocker"];
	        this.block_spam_ports = source["block_spam_ports"];
	        this.integration_mode = source["integration_mode"];
	        this.config_profile_uuid = source["config_profile_uuid"];
	        this.inbound_uuid = source["inbound_uuid"];
	        this.squad_uuid = source["squad_uuid"];
	        this.create_host = source["create_host"];
	        this.host_address = source["host_address"];
	        this.host_sni = source["host_sni"];
	    }
	}
	export class CreateNodeResponse {
	    success: boolean;
	    node_uuid: string;
	    secret_key: string;
	    host_created: boolean;
	    squad_updated: boolean;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateNodeResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.node_uuid = source["node_uuid"];
	        this.secret_key = source["secret_key"];
	        this.host_created = source["host_created"];
	        this.squad_updated = source["squad_updated"];
	        this.error = source["error"];
	    }
	}
	export class PanelProfileItem {
	    uuid: string;
	    name: string;
	    inbound_uuid: string;
	    inbound_tag: string;
	    port: number;
	    protocol: string;
	
	    static createFrom(source: any = {}) {
	        return new PanelProfileItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uuid = source["uuid"];
	        this.name = source["name"];
	        this.inbound_uuid = source["inbound_uuid"];
	        this.inbound_tag = source["inbound_tag"];
	        this.port = source["port"];
	        this.protocol = source["protocol"];
	    }
	}
	export class PanelSquadItem {
	    uuid: string;
	    name: string;
	    inbounds_count: number;
	    inbound_uuids: string[];
	
	    static createFrom(source: any = {}) {
	        return new PanelSquadItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uuid = source["uuid"];
	        this.name = source["name"];
	        this.inbounds_count = source["inbounds_count"];
	        this.inbound_uuids = source["inbound_uuids"];
	    }
	}
	export class DNSResult {
	    resolved: boolean;
	    ip: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new DNSResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.resolved = source["resolved"];
	        this.ip = source["ip"];
	        this.message = source["message"];
	    }
	}
	export class DeploymentPlan {
	    components: string[];
	    domain: string;
	    panel_domain: string;
	    subscription_domain: string;
	    node_port: number;
	    node_secret: string;
	    proxy: string;
	    subscription_mode: string;
	    secret_gate: string;
	    use_camouflage: boolean;
	    ufw_setup: boolean;
	    ssh_host: string;
	    ssh_port: number;
	    ssh_user: string;
	    ssh_password: string;
	    ssh_key: string;
	    output_dir: string;
	    deployment_mode: string;
	    use_cloudflare_dns: boolean;
	    cloudflare_api_token: string;
	    panel_ip: string;
	    hide_node_port: boolean;
	    stealth_drop: boolean;
	    deploy_decoy: boolean;
	    decoy_domain: string;
	    port_mode: string;
	    decoy_template: string;
	    decoy_clone_url: string;
	    torrent_blocker: boolean;
	    block_spam_ports: boolean;
	    xray_core_type: string;
	    enable_tinyauth: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DeploymentPlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.components = source["components"];
	        this.domain = source["domain"];
	        this.panel_domain = source["panel_domain"];
	        this.subscription_domain = source["subscription_domain"];
	        this.node_port = source["node_port"];
	        this.node_secret = source["node_secret"];
	        this.proxy = source["proxy"];
	        this.subscription_mode = source["subscription_mode"];
	        this.secret_gate = source["secret_gate"];
	        this.use_camouflage = source["use_camouflage"];
	        this.ufw_setup = source["ufw_setup"];
	        this.ssh_host = source["ssh_host"];
	        this.ssh_port = source["ssh_port"];
	        this.ssh_user = source["ssh_user"];
	        this.ssh_password = source["ssh_password"];
	        this.ssh_key = source["ssh_key"];
	        this.output_dir = source["output_dir"];
	        this.deployment_mode = source["deployment_mode"];
	        this.use_cloudflare_dns = source["use_cloudflare_dns"];
	        this.cloudflare_api_token = source["cloudflare_api_token"];
	        this.panel_ip = source["panel_ip"];
	        this.hide_node_port = source["hide_node_port"];
	        this.stealth_drop = source["stealth_drop"];
	        this.deploy_decoy = source["deploy_decoy"];
	        this.decoy_domain = source["decoy_domain"];
	        this.port_mode = source["port_mode"];
	        this.decoy_template = source["decoy_template"];
	        this.decoy_clone_url = source["decoy_clone_url"];
	        this.torrent_blocker = source["torrent_blocker"];
	        this.block_spam_ports = source["block_spam_ports"];
	        this.xray_core_type = source["xray_core_type"];
	        this.enable_tinyauth = source["enable_tinyauth"];
	    }
	}
	export class GenerateResult {
	    success: boolean;
	    files: string[];
	    out_dir: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new GenerateResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.files = source["files"];
	        this.out_dir = source["out_dir"];
	        this.error = source["error"];
	    }
	}
	export class NodeSecurityAudit {
	    node_port_protected: boolean;
	    panel_whitelisted: boolean;
	    docker_user_active: boolean;
	    stealth_drop_active: boolean;
	    decoy_active: boolean;
	    bbr_active: boolean;
	    container_running: boolean;
	    port_443_listening: boolean;
	    xray_core_type: string;
	    overall_score: number;
	    summary: string;
	    checks: string[];
	
	    static createFrom(source: any = {}) {
	        return new NodeSecurityAudit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.node_port_protected = source["node_port_protected"];
	        this.panel_whitelisted = source["panel_whitelisted"];
	        this.docker_user_active = source["docker_user_active"];
	        this.stealth_drop_active = source["stealth_drop_active"];
	        this.decoy_active = source["decoy_active"];
	        this.bbr_active = source["bbr_active"];
	        this.container_running = source["container_running"];
	        this.port_443_listening = source["port_443_listening"];
	        this.xray_core_type = source["xray_core_type"];
	        this.overall_score = source["overall_score"];
	        this.summary = source["summary"];
	        this.checks = source["checks"];
	    }
	}
	export class PanelInfoResult {
	    success: boolean;
	    version: string;
	    nodes_count: number;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new PanelInfoResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.version = source["version"];
	        this.nodes_count = source["nodes_count"];
	        this.error = source["error"];
	    }
	}
	export class PanelNodeItem {
	    uuid: string;
	    name: string;
	    address: string;
	    port: number;
	    country: string;
	    is_connected: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PanelNodeItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uuid = source["uuid"];
	        this.name = source["name"];
	        this.address = source["address"];
	        this.port = source["port"];
	        this.country = source["country"];
	        this.is_connected = source["is_connected"];
	    }
	}
	export class SSHTestResult {
	    success: boolean;
	    os: string;
	    cpu: string;
	    ram: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new SSHTestResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.os = source["os"];
	        this.cpu = source["cpu"];
	        this.ram = source["ram"];
	        this.error = source["error"];
	    }
	}
	export class SavedServer {
	    id: string;
	    label: string;
	    host: string;
	    port: number;
	    username: string;
	    password: string;
	    ssh_key: string;
	    web_url: string;
	    role: string;
	    parent_master_id: string;
	    location_code: string;
	    decoy_domain: string;
	    is_external_master: boolean;
	    last_seen: string;
	    admin_user: string;
	    admin_password: string;
	    sub_url: string;
	    port_mode: string;
	    xray_core_type: string;
	
	    static createFrom(source: any = {}) {
	        return new SavedServer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.ssh_key = source["ssh_key"];
	        this.web_url = source["web_url"];
	        this.role = source["role"];
	        this.parent_master_id = source["parent_master_id"];
	        this.location_code = source["location_code"];
	        this.decoy_domain = source["decoy_domain"];
	        this.is_external_master = source["is_external_master"];
	        this.last_seen = source["last_seen"];
	        this.admin_user = source["admin_user"];
	        this.admin_password = source["admin_password"];
	        this.sub_url = source["sub_url"];
	        this.port_mode = source["port_mode"];
	        this.xray_core_type = source["xray_core_type"];
	    }
	}
	export class ServerMetrics {
	    uptime: string;
	    ram_used_mb: number;
	    ram_total_mb: number;
	    ram_percent: number;
	    disk_used: string;
	    disk_total: string;
	    disk_percent: number;
	
	    static createFrom(source: any = {}) {
	        return new ServerMetrics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uptime = source["uptime"];
	        this.ram_used_mb = source["ram_used_mb"];
	        this.ram_total_mb = source["ram_total_mb"];
	        this.ram_percent = source["ram_percent"];
	        this.disk_used = source["disk_used"];
	        this.disk_total = source["disk_total"];
	        this.disk_percent = source["disk_percent"];
	    }
	}
	export class ServerStatusResult {
	    success: boolean;
	    host_info: string;
	    metrics: ServerMetrics;
	    containers: ContainerInfo[];
	    bbr_active: boolean;
	    docker_mirrors_active: boolean;
	    anti_abuse_active: boolean;
	    warp_active: boolean;
	    node_present: boolean;
	    latency_ms: number;
	    backups: BackupInfo[];
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new ServerStatusResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.host_info = source["host_info"];
	        this.metrics = this.convertValues(source["metrics"], ServerMetrics);
	        this.containers = this.convertValues(source["containers"], ContainerInfo);
	        this.bbr_active = source["bbr_active"];
	        this.docker_mirrors_active = source["docker_mirrors_active"];
	        this.anti_abuse_active = source["anti_abuse_active"];
	        this.warp_active = source["warp_active"];
	        this.node_present = source["node_present"];
	        this.latency_ms = source["latency_ms"];
	        this.backups = this.convertValues(source["backups"], BackupInfo);
	        this.error = source["error"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

