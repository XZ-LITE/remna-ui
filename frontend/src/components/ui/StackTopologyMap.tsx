import React from "react";
import {
  Smartphone,
  Globe,
  ShieldCheck,
  Server,
  Layers,
  Database,
  Radio,
  Zap,
  Terminal,
  Activity,
  ArrowRight,
  ArrowDown,
} from "lucide-react";
import * as types from "../../../bindings/remna-ui/internal/types/models";

export interface StackTopologyMapProps {
  server: types.SavedServer;
  status?: types.ServerStatusResult;
  onInspectLogs?: (containerName: string) => void;
}

interface TopologyNodeProps {
  icon: React.ReactNode;
  title: string;
  subtitle: string;
  port?: string;
  status: "online" | "warning" | "offline" | "disabled";
  containerName?: string;
  onClick?: () => void;
}

const TopologyNode: React.FC<TopologyNodeProps> = ({
  icon,
  title,
  subtitle,
  port,
  status,
  containerName,
  onClick,
}) => {
  return (
    <div
      className={`win-topology-node ${status} ${containerName && onClick ? "interactive" : ""}`}
      onClick={containerName && onClick ? onClick : undefined}
      title={containerName && onClick ? `Открыть логи контейнера ${containerName}` : undefined}
    >
      <div className="win-topology-node-header">
        <div className="win-topology-node-icon">{icon}</div>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div className="win-topology-node-title">{title}</div>
          <div className="win-topology-node-subtitle">{subtitle}</div>
        </div>
        <span className={`win-status-indicator ${status}`} />
      </div>

      <div className="win-topology-node-footer">
        {port && <span className="win-topology-port-badge">{port}</span>}
        {containerName && onClick && (
          <span className="win-topology-log-hint">
            <Terminal size={10} />
            <span>Логи</span>
          </span>
        )}
      </div>
    </div>
  );
};

export const StackTopologyMap: React.FC<StackTopologyMapProps> = ({
  server,
  status,
  onInspectLogs,
}) => {
  const containers = status?.containers || [];

  const getContainerStatus = (name: string): "online" | "offline" | "disabled" => {
    const found = containers.find((c: types.ContainerInfo) => c.name.toLowerCase().includes(name.toLowerCase()));
    if (!found) return "disabled";
    return found.running ? "online" : "offline";
  };

  const proxyContainer = containers.find(
    (c: types.ContainerInfo) => c.name.includes("nginx") || c.name.includes("caddy")
  );
  const proxyName = proxyContainer ? proxyContainer.name : "remnawave-nginx";
  const proxyStatus = getContainerStatus(proxyName);

  const panelStatus = getContainerStatus("remnawave");
  const subStatus = getContainerStatus("subscription");
  const nodeStatus = getContainerStatus("remnanode");
  const dbStatus = getContainerStatus("remnawave-db");
  const redisStatus = getContainerStatus("remnawave-redis");
  const warpActive = Boolean(status?.warp_active);

  return (
    <div className="win-topology-wrapper">
      <div className="win-topology-header">
        <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
          <Activity size={16} style={{ color: "var(--win-accent)" }} />
          <span style={{ fontWeight: 600, fontSize: "13px", color: "var(--win-text-primary)" }}>
            Интерактивная карта топологии стека
          </span>
        </div>
        <span style={{ fontSize: "11px", color: "var(--win-text-tertiary)" }}>
          Кликните по любому контейнеру для мгновенного просмотра логов
        </span>
      </div>

      {/* Topology Diagram Grid */}
      <div className="win-topology-canvas">
        {/* Level 1: Ingress Clients */}
        <div className="win-topology-level">
          <TopologyNode
            icon={<Smartphone size={16} />}
            title="Клиенты (Apps)"
            subtitle="V2rayN, Sing-Box, Streisand, Happ"
            port="Входящий трафик"
            status="online"
          />
        </div>

        <div className="win-topology-connector-v">
          <ArrowDown size={14} />
        </div>

        {/* Level 2: Edge Reverse Proxy */}
        <div className="win-topology-level">
          <TopologyNode
            icon={<ShieldCheck size={16} />}
            title={proxyName}
            subtitle="HTTPS SSL, Security Gate & Маскировка"
            port=":443, :80"
            status={proxyStatus}
            containerName={proxyName}
            onClick={() => onInspectLogs && onInspectLogs(proxyName)}
          />
        </div>

        <div className="win-topology-connector-v">
          <ArrowDown size={14} />
        </div>

        {/* Level 3: Core Services (Panel, Subscription, Node) */}
        <div className="win-topology-level grid-3">
          {/* Panel */}
          <div className="win-topology-branch">
            <TopologyNode
              icon={<Server size={16} />}
              title="remnawave"
              subtitle="Панель управления (Admin API & UI)"
              port=":3000"
              status={panelStatus}
              containerName="remnawave"
              onClick={() => onInspectLogs && onInspectLogs("remnawave")}
            />
            {/* Sub-branch: Database and Redis */}
            <div className="win-topology-subgroup">
              <div className="win-topology-connector-v sub">
                <ArrowDown size={12} />
              </div>
              <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: "6px" }}>
                <TopologyNode
                  icon={<Database size={13} />}
                  title="remnawave-db"
                  subtitle="PostgreSQL 17"
                  port=":5432"
                  status={dbStatus}
                  containerName="remnawave-db"
                  onClick={() => onInspectLogs && onInspectLogs("remnawave-db")}
                />
                <TopologyNode
                  icon={<Radio size={13} />}
                  title="remnawave-redis"
                  subtitle="Valkey Cache"
                  port="Socket"
                  status={redisStatus}
                  containerName="remnawave-redis"
                  onClick={() => onInspectLogs && onInspectLogs("remnawave-redis")}
                />
              </div>
            </div>
          </div>

          {/* Subscriptions */}
          <div className="win-topology-branch">
            <TopologyNode
              icon={<Layers size={16} />}
              title="remnawave-subscription"
              subtitle="Страница клиентских подписок"
              port=":3010"
              status={subStatus}
              containerName="remnawave-subscription-page"
              onClick={() => onInspectLogs && onInspectLogs("remnawave-subscription-page")}
            />
          </div>

          {/* Node & Outbound */}
          <div className="win-topology-branch">
            <TopologyNode
              icon={<Radio size={16} />}
              title="remnanode"
              subtitle="Xray Reality / VLESS Нода"
              port=":2222 / :443"
              status={nodeStatus}
              containerName="remnanode"
              onClick={() => onInspectLogs && onInspectLogs("remnanode")}
            />
            {/* Outbound WARP node */}
            <div className="win-topology-subgroup">
              <div className="win-topology-connector-v sub">
                <ArrowDown size={12} />
              </div>
              <TopologyNode
                icon={<Zap size={13} />}
                title="Cloudflare WARP"
                subtitle="Чистый исходящий IP (SOCKS5)"
                port=":40000"
                status={warpActive ? "online" : "disabled"}
              />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
