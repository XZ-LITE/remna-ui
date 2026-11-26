import React, { memo, useRef } from "react";
import {
  Server,
  Radio,
  Globe,
  Database,
  Activity,
  Layers,
  ChevronDown,
  ExternalLink,
  ShieldCheck,
  RefreshCw,
  MoreVertical,
  Plus,
  RotateCcw,
  Download,
  Zap,
  Cpu,
  Terminal as TerminalIcon,
  ShieldAlert,
  Trash2,
  Edit3,
  Wrench,
  AlertOctagon,
  Copy,
  Check,
  CheckCircle2,
  AlertTriangle,
  AlertCircle,
  HardDrive,
  Key,
  Shield,
  Clock,
  Gauge,
  Cloud,
  Archive,
  Lock,
} from "lucide-react";
import * as types from "../../../bindings/remna-ui/internal/types/models";
import { WinTooltip } from "../ui/WinTooltip";
import { WinActionMenu } from "../ui/WinActionMenu";

const isIpAddress = (val: string) => /^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$/.test((val || "").trim());

export interface ServerCardProps {
  srv: types.SavedServer;
  isChild?: boolean;
  childCount?: number;
  cardIndex?: number;
  status?: types.ServerStatusResult;
  isLoading?: boolean;
  isCollapsed?: boolean;
  activeServerMenu?: string | null;
  subTab?: string;
  actionLoading?: Record<string, boolean>;
  cleaningDiskServerId?: string | null;
  savedServers?: types.SavedServer[];
  copiedKey?: string | null;

  toggleServerCollapse: (id: string) => void;
  setActiveServerMenu: (id: string | null) => void;
  setServerSubTab: React.Dispatch<React.SetStateAction<any>>;
  handleCopy: (text: string, label: string, key?: string) => void;
  handleOpenBrowser: (url: string) => void;
  handleOpenAudit: (srv: types.SavedServer) => void;
  handleInspectServer: (srv: types.SavedServer) => void;
  handleAddNodeToPanel: (srv: types.SavedServer) => void;
  handleRestartAll: (srv: types.SavedServer) => void;
  handleUpdateStack: (srv: types.SavedServer) => void;
  handleOptimizeBBR: (srv: types.SavedServer) => void;
  handleTogglePerformance: (srv: types.SavedServer, currentOpt: boolean) => void;
  setNodeModal: (modal: any) => void;
  handleOpenLogsModal: (srv: types.SavedServer, container: string) => void;
  handleRestartContainer: (srv: types.SavedServer, container: string) => void;
  handleOpenIPCheckModal: (srv: types.SavedServer) => void;
  handleOpenVPNProbeModal: (srv: types.SavedServer) => void;
  handleCleanServerDisk: (srv: types.SavedServer) => void;
  handleStartEditServer: (srv: types.SavedServer) => void;
  setReinstallModal: (modal: any) => void;
  setUninstallModal: (modal: any) => void;
  handleDeleteServer: (id: string) => void;
  handleCreateBackup: (srv: types.SavedServer) => void;
  handleDownloadBackup: (srv: types.SavedServer, fileName: string) => void;
  setRestoreModal: (modal: any) => void;
  handleFixDBSync: (srv: types.SavedServer) => void;
  handleInstallWARP: (srv: types.SavedServer) => void;
  handleResetSuperadmin: (srv: types.SavedServer) => void;
  handleToggleAntiAbuse?: (srv: types.SavedServer, current: boolean) => void;
}

export const ServerCard: React.FC<ServerCardProps> = memo(({
  srv,
  isChild = false,
  childCount,
  cardIndex = 0,
  status,
  isLoading = false,
  isCollapsed = false,
  activeServerMenu,
  subTab = "overview",
  actionLoading = {},
  cleaningDiskServerId,
  savedServers = [],
  copiedKey,
  toggleServerCollapse,
  setActiveServerMenu,
  setServerSubTab,
  handleCopy,
  handleOpenBrowser,
  handleOpenAudit,
  handleInspectServer,
  handleAddNodeToPanel,
  handleRestartAll,
  handleUpdateStack,
  handleOptimizeBBR,
  handleTogglePerformance,
  setNodeModal,
  handleOpenLogsModal,
  handleRestartContainer,
  handleOpenIPCheckModal,
  handleOpenVPNProbeModal,
  handleCleanServerDisk,
  handleStartEditServer,
  setReinstallModal,
  setUninstallModal,
  handleDeleteServer,
  handleCreateBackup,
  handleDownloadBackup,
  setRestoreModal,
  handleFixDBSync,
  handleInstallWARP,
  handleResetSuperadmin,
  handleToggleAntiAbuse,
}) => {
                      const moreMenuAnchorRef = useRef<HTMLButtonElement | null>(null);
                      const webUrl = srv.role === "node" ? "" : (srv.web_url || "");

                      const runningCount = status?.containers?.filter((c) => c.running).length || 0;
                      const totalCount = status?.containers?.length || 0;
                      const hasRunningNode = status?.containers?.some((c) => c.name === "remnanode" && c.running);
                      const isNodeAvailable = Boolean(hasRunningNode || status?.node_present);
                      const hasMasterContainer = Boolean(status?.containers?.some((c) => c.name === "remnawave"));
                      // isCollapsed passed via prop

                      // Reliable role detection:
                      // 1. Explicit role if set in SavedServer ("node" vs "master")
                      // 2. Active containers detected on VPS (remnawave = panel, remnanode = node)
                      // 3. Valid web_url (only master panels have web_url)
                      // 4. Default: raw IP or server with no web_url is an Edge Node
                      const isMaster = srv.role === "master"
                        ? true
                        : srv.role === "node"
                        ? false
                        : hasMasterContainer
                        ? true
                        : isNodeAvailable
                        ? false
                        : Boolean(srv.web_url && !isIpAddress(srv.host));

                      const getServiceMeta = (name: string) => {
                        switch (name) {
                          case "remnawave":
                            return { role: "Панель & API Core", icon: <Server size={14} style={{ color: "var(--win-accent)" }} /> };
                          case "remnawave-db":
                            return { role: "PostgreSQL База данных", icon: <Database size={14} style={{ color: "#818cf8" }} /> };
                          case "remnawave-redis":
                            return { role: "Valkey Сессии & Кэш", icon: <Activity size={14} style={{ color: "#fbbf24" }} /> };
                          case "remnawave-nginx":
                            return { role: "Nginx SSL & Прокси", icon: <Globe size={14} style={{ color: "#34d399" }} /> };
                          case "remnawave-caddy":
                          case "caddy":
                            return { role: "Caddy Авто-SSL & Прокси", icon: <Globe size={14} style={{ color: "#38bdf8" }} /> };
                          case "remnanode":
                            return { role: "Xray Reality Нода", icon: <Radio size={14} style={{ color: "var(--win-node-accent)" }} /> };
                          default:
                            return { role: "Docker Служба", icon: <Layers size={14} style={{ color: "#9ca3af" }} /> };
                        }
                      };

                      const rawSubTab = subTab || "overview";
                      const currentSubTab = rawSubTab === "containers" ? "overview" : rawSubTab;

                      return (
                        <div
                          key={srv.id}
                          className={`server-item-card ${isMaster ? "master-card" : "node-card"} ${isChild ? "child-node-card" : ""} ${activeServerMenu === srv.id ? "menu-open" : ""} ${isCollapsed ? "collapsed" : ""}`}
                          style={{ "--server-index": cardIndex } as React.CSSProperties}
                        >
                          {/* Server Card Header (Modern 2-Tier Responsive Layout) */}
                          <div
                            className="server-item-header"
                            onClick={() => {
                              if (isCollapsed) {
                                toggleServerCollapse(srv.id);
                              }
                            }}
                          >
                            {/* Row 1: Identity & Primary Hero Action */}
                            <div className="server-header-top">
                              <div className="server-identity-group">
                                <button
                                  className="server-accordion-btn"
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    toggleServerCollapse(srv.id);
                                  }}
                                  title={isCollapsed ? "Развернуть карточку сервера" : "Свернуть карточку сервера"}
                                >
                                  <ChevronDown
                                    size={14}
                                    className={`server-accordion-chevron ${isCollapsed ? "collapsed" : "expanded"}`}
                                  />
                                </button>
                                <div className="server-item-icon" style={{ color: isMaster ? "var(--win-master-accent)" : "var(--win-node-accent)" }}>
                                  {isMaster ? <Server size={22} /> : <Radio size={22} />}
                                </div>
                                <div className="server-identity-text">
                                  <span className="server-name" title={srv.label || srv.host}>
                                    {(srv as any).location_code && `${(srv as any).location_code} · `}
                                    {srv.label || srv.host}
                                  </span>
                                  {isMaster ? (
                                    <>
                                      <span className="server-chip master-chip">
                                        <Server size={10} />
                                        <span>Master Panel</span>
                                      </span>
                                      {(srv as any).is_external_master && (
                                        <span className="server-chip external-chip" title="Панель развернута пользователем вручную">
                                          <span>Ручная / Внешняя</span>
                                        </span>
                                      )}
                                      {childCount !== undefined && childCount > 0 && (
                                        <span className="server-chip parent-link-chip" title="Привязанные ноды в инфраструктуре">
                                          <Radio size={10} />
                                          <span>{childCount} {childCount === 1 ? "нода" : childCount < 5 ? "ноды" : "нод"}</span>
                                        </span>
                                      )}
                                    </>
                                  ) : (
                                    <>
                                      <span className="server-chip node-chip">
                                        <Radio size={10} />
                                        <span>Edge Node</span>
                                      </span>
                                      {(srv as any).parent_master_id && (
                                        <span className="server-chip parent-link-chip" title="Привязана к мастер-панели">
                                          <Server size={10} />
                                          <span>{savedServers.find((p) => p.id === (srv as any).parent_master_id)?.label || "Master"}</span>
                                        </span>
                                      )}
                                      {(srv as any).decoy_domain && (
                                        <span
                                          className={`server-chip decoy-chip ${copiedKey === `decoy-${srv.id}` ? "copied" : ""}`}
                                          title="Decoy SNI маскировка. Нажмите для копирования"
                                          onClick={(e) => {
                                            e.stopPropagation();
                                            handleCopy((srv as any).decoy_domain, "Decoy Домен", `decoy-${srv.id}`);
                                          }}
                                        >
                                          {copiedKey === `decoy-${srv.id}` ? (
                                            <>
                                              <Check size={10} style={{ color: "#34d399" }} />
                                              <span style={{ color: "#34d399", fontWeight: 600 }}>Скопировано!</span>
                                            </>
                                          ) : (
                                            <>
                                              <Globe size={10} />
                                              <span>{(srv as any).decoy_domain}</span>
                                            </>
                                          )}
                                        </span>
                                      )}
                                    </>
                                  )}
                                  {status && (() => {
                                    const isOnline = Boolean(status.success);
                                    const healthTooltip = isOnline
                                      ? `Сервер в сети\n• SSH задержка: ${status.latency_ms || 0} ms\n• Службы: ${runningCount}/${totalCount} активны\n• BBR: ${status.bbr_active ? "активно" : "выключено"}${status.host_info ? `\n• ОС: ${status.host_info.split("(")[0].trim()}` : ""}`
                                      : `Сервер недоступен\n• Ошибка: ${status.error || "Нет ответа по SSH"}\n• Проверьте порт ${srv.port} и сетевой экран`;

                                    return (
                                      <span
                                        className={`server-status-pill ${isOnline ? "online" : "offline"}`}
                                        title={healthTooltip}
                                      >
                                        <span
                                          className={`status-dot-pulse ${isOnline ? "running" : "stopped"}`}
                                        />
                                        <span>{isOnline ? "В сети" : "Недоступен"}</span>
                                      </span>
                                    );
                                  })()}

                                  {isCollapsed && status?.success && (
                                    <div className="server-compact-metrics">
                                      {status.metrics?.ram_percent !== undefined && status.metrics.ram_percent > 0 && (
                                        <span className="server-compact-pill ram" title={`Использование памяти: ${status.metrics.ram_used_mb || 0} / ${status.metrics.ram_total_mb || 0} MB`}>
                                          <Cpu size={10} />
                                          <span>RAM {status.metrics.ram_percent}%</span>
                                        </span>
                                      )}
                                      {totalCount > 0 && (
                                        <span
                                          className={`server-compact-pill ${runningCount === totalCount ? "services-ok" : "services-warn"}`}
                                          title={`Контейнеры: ${runningCount} из ${totalCount} активны`}
                                        >
                                          <Layers size={10} />
                                          <span>{runningCount}/{totalCount} служб</span>
                                        </span>
                                      )}
                                      {status.latency_ms ? (
                                        <span className="server-compact-pill" title={`SSH задержка: ${status.latency_ms} ms`}>
                                          <Zap size={10} style={{ color: "#fbbf24" }} />
                                          <span>{status.latency_ms} ms</span>
                                        </span>
                                      ) : null}
                                      {status.metrics?.cpu_steal !== undefined && status.metrics.cpu_steal >= 5 && (
                                        <span
                                          className={`server-compact-pill ${status.metrics.cpu_steal >= 15 ? "services-warn" : ""}`}
                                          title={`CPU Steal: ${status.metrics.cpu_steal.toFixed(1)}% (потеря тактов из-за оверселлинга хостера)`}
                                          style={status.metrics.cpu_steal >= 15 ? { color: "#ef4444", borderColor: "#ef4444" } : { color: "#f59e0b" }}
                                        >
                                          <Activity size={10} />
                                          <span>Steal {status.metrics.cpu_steal.toFixed(1)}%</span>
                                        </span>
                                      )}
                                    </div>
                                  )}
                                </div>
                              </div>

                              <div style={{ display: "flex", alignItems: "center", gap: "8px", flexShrink: 0 }}>
                                {isMaster ? (
                                  webUrl && (
                                    <WinTooltip content={`Открыть веб-панель: ${webUrl}`}>
                                      <button
                                        className="win-button hero-primary"
                                        onClick={() => handleOpenBrowser(webUrl)}
                                      >
                                        <ExternalLink size={14} />
                                        <span>Открыть панель</span>
                                      </button>
                                    </WinTooltip>
                                  )
                                ) : (
                                  <WinTooltip content="Запустить диагностику и аудит защиты ноды">
                                    <button
                                      className="win-button hero-primary"
                                      style={{
                                        background: "rgba(56, 189, 248, 0.10)",
                                        color: "var(--win-node-accent)",
                                        borderColor: "var(--win-node-border)",
                                      }}
                                      onClick={() => handleOpenAudit(srv)}
                                    >
                                      <ShieldCheck size={14} />
                                      <span>Диагностика ноды</span>
                                    </button>
                                  </WinTooltip>
                                )}

                                <WinTooltip content="Опросить состояние сервера" shortcut="F5">
                                  <button
                                    className="win-button toolbar-action-btn icon-only"
                                    disabled={isLoading}
                                    onClick={() => handleInspectServer(srv)}
                                  >
                                    <RefreshCw size={13} className={isLoading ? "animate-spin" : ""} />
                                  </button>
                                </WinTooltip>

                                <div className="server-actions-wrapper">
                                  <WinTooltip content="Дополнительные действия...">
                                    <button
                                      ref={moreMenuAnchorRef}
                                      className={`win-button toolbar-action-btn icon-only ${activeServerMenu === srv.id ? "active" : ""}`}
                                      onClick={(e) => {
                                        e.stopPropagation();
                                        setActiveServerMenu(activeServerMenu === srv.id ? null : srv.id);
                                      }}
                                    >
                                      <MoreVertical size={14} />
                                    </button>
                                  </WinTooltip>

                                  <WinActionMenu
                                    isOpen={activeServerMenu === srv.id}
                                    anchorRef={moreMenuAnchorRef}
                                    onClose={() => setActiveServerMenu(null)}
                                  >
                                      {isMaster ? (
                                        <>
                                          <button
                                            className="server-menu-item"
                                            onClick={() => {
                                              setActiveServerMenu(null);
                                              handleAddNodeToPanel(srv);
                                            }}
                                          >
                                            <Plus size={14} style={{ color: "var(--win-accent)" }} />
                                            <span>Привязать новую ноду</span>
                                          </button>
                                          <button
                                            className="server-menu-item"
                                            onClick={() => {
                                              setActiveServerMenu(null);
                                              handleRestartAll(srv);
                                            }}
                                            disabled={actionLoading[`restart-${srv.id}`]}
                                          >
                                            <RotateCcw size={14} />
                                            <span>Перезапустить стек</span>
                                          </button>
                                          <button
                                            className="server-menu-item"
                                            onClick={() => {
                                              setActiveServerMenu(null);
                                              handleUpdateStack(srv);
                                            }}
                                            disabled={actionLoading[`pull-${srv.id}`]}
                                          >
                                            <Download size={14} />
                                            <span>Обновить образы (Pull & Up)</span>
                                          </button>
                                          <button
                                            className="server-menu-item"
                                            onClick={() => {
                                              setActiveServerMenu(null);
                                              handleOptimizeBBR(srv);
                                            }}
                                            disabled={actionLoading[`bbr-${srv.id}`]}
                                          >
                                            <Zap size={14} />
                                            <span>{status?.bbr_active ? "BBR уже настроен" : "Включить BBR"}</span>
                                          </button>
                                          <button
                                            className="server-menu-item"
                                            onClick={() => {
                                              setActiveServerMenu(null);
                                              handleTogglePerformance(srv, !!status?.performance_optimized);
                                            }}
                                            disabled={actionLoading[`perf-toggle-${srv.id}`]}
                                          >
                                            <Cpu size={14} style={{ color: "#a855f7" }} />
                                            <span>{status?.performance_optimized ? "Отключить RAM Boost" : "RAM Boost & Sockets"}</span>
                                          </button>
                                        </>
                                      ) : (
                                        <>
                                          <button
                                            className="server-menu-item"
                                            onClick={() => {
                                              setActiveServerMenu(null);
                                              handleOpenAudit(srv);
                                            }}
                                          >
                                            <ShieldCheck size={14} style={{ color: "#10b981" }} />
                                            <span>Аудит защиты от ТСПУ</span>
                                          </button>
                                          <button
                                            className="server-menu-item"
                                            onClick={() => {
                                              setActiveServerMenu(null);
                                              setNodeModal({ server: srv, secretKey: "", loading: false });
                                            }}
                                          >
                                            <Radio size={14} style={{ color: "var(--win-node-accent)" }} />
                                            <span>Сменить SECRET_KEY ноды</span>
                                          </button>
                                          <button
                                            className="server-menu-item"
                                            onClick={() => {
                                              setActiveServerMenu(null);
                                              handleOpenLogsModal(srv, "remnanode");
                                            }}
                                          >
                                            <TerminalIcon size={14} />
                                            <span>Журнал логов ноды</span>
                                          </button>
                                          <button
                                            className="server-menu-item"
                                            onClick={() => {
                                              setActiveServerMenu(null);
                                              handleRestartContainer(srv, "remnanode");
                                            }}
                                            disabled={actionLoading[`restart-${srv.id}-remnanode`]}
                                          >
                                            <RotateCcw size={14} />
                                            <span>Перезапустить ноду</span>
                                          </button>
                                          <button
                                            className="server-menu-item"
                                            onClick={() => {
                                              setActiveServerMenu(null);
                                              handleOptimizeBBR(srv);
                                            }}
                                            disabled={actionLoading[`bbr-${srv.id}`]}
                                          >
                                            <Zap size={14} />
                                            <span>{status?.bbr_active ? "BBR уже настроен" : "Включить BBR"}</span>
                                          </button>
                                        </>
                                      )}

                                      <div className="server-menu-divider" />

                                      <button
                                        className="server-menu-item"
                                        onClick={() => {
                                          setActiveServerMenu(null);
                                          handleOpenIPCheckModal(srv);
                                        }}
                                      >
                                        <ShieldAlert size={14} style={{ color: "#38bdf8" }} />
                                        <span>Проверить доступность из РФ</span>
                                      </button>

                                      <button
                                        className="server-menu-item"
                                        onClick={() => {
                                          setActiveServerMenu(null);
                                          handleOpenVPNProbeModal(srv);
                                        }}
                                      >
                                        <Activity size={14} style={{ color: "#c084fc" }} />
                                        <span>Тест VPN-трафика (E2E)</span>
                                      </button>

                                      <button
                                        className="server-menu-item"
                                        onClick={() => {
                                          setActiveServerMenu(null);
                                          handleCleanServerDisk(srv);
                                        }}
                                        disabled={cleaningDiskServerId === srv.id}
                                      >
                                        <Trash2 size={14} style={{ color: "#f87171" }} />
                                        <span>{cleaningDiskServerId === srv.id ? "Очистка диска..." : "Очистить диск (Smart Clean)"}</span>
                                      </button>

                                      <button
                                        className="server-menu-item"
                                        onClick={() => {
                                          setActiveServerMenu(null);
                                          handleStartEditServer(srv);
                                        }}
                                      >
                                        <Edit3 size={14} />
                                        <span>Редактировать параметры</span>
                                      </button>
                                      <button
                                        className="server-menu-item"
                                        onClick={() => {
                                          setActiveServerMenu(null);
                                          setReinstallModal({ server: srv, open: true });
                                        }}
                                      >
                                        <Wrench size={14} style={{ color: "var(--win-accent)" }} />
                                        <span>Переустановить стек...</span>
                                      </button>

                                      <button
                                        className="server-menu-item danger"
                                        onClick={() => {
                                          setActiveServerMenu(null);
                                          setUninstallModal({ server: srv, open: true, confirmHost: "", loading: false });
                                        }}
                                      >
                                        <AlertOctagon size={14} />
                                        <span>Очистить VPS (Удалить Remnawave)</span>
                                      </button>

                                      <div className="server-menu-divider" />

                                      <button
                                        className="server-menu-item"
                                        style={{ color: "var(--win-text-secondary)" }}
                                        onClick={() => {
                                          setActiveServerMenu(null);
                                          handleDeleteServer(srv.id);
                                        }}
                                      >
                                        <Trash2 size={14} />
                                        <span>Удалить только из каталога</span>
                                      </button>
                                  </WinActionMenu>
                                </div>
                              </div>
                            </div>

                            {/* Row 2: Metadata Chips & Secondary Action Toolbar */}
                            {!isCollapsed && (
                              <div className="server-header-bottom">
                                <div className="server-meta-chips">
                                  <span
                                    className={`server-chip mono ${copiedKey === `ssh-${srv.id}` ? "copied" : ""}`}
                                    onClick={() => handleCopy(`${srv.username}@${srv.host}:${srv.port}`, "SSH", `ssh-${srv.id}`)}
                                    title="Нажмите, чтобы скопировать SSH"
                                  >
                                    {copiedKey === `ssh-${srv.id}` ? (
                                      <>
                                        <Check size={11} style={{ color: "#34d399" }} />
                                        <span style={{ color: "#34d399", fontWeight: 600 }}>Скопировано!</span>
                                      </>
                                    ) : (
                                      <>
                                        <TerminalIcon size={11} />
                                        <span>{srv.username}@{srv.host}:{srv.port}</span>
                                        <Copy size={10} style={{ opacity: 0.6 }} />
                                      </>
                                    )}
                                  </span>

                                  {status?.latency_ms ? (
                                    <span
                                      className={`server-chip latency ${
                                        status.latency_ms < 60
                                          ? "fast"
                                          : status.latency_ms < 150
                                          ? "medium"
                                          : "slow"
                                      }`}
                                      title="Задержка SSH соединения"
                                    >
                                      <Zap size={11} />
                                      <span>{status.latency_ms} ms</span>
                                    </span>
                                  ) : null}

                                  {status?.host_info ? (
                                    <span className="server-chip os" title={status.host_info}>
                                      <Layers size={11} />
                                      <span>{status.host_info.split("(")[0].replace("GNU/Linux", "").trim()}</span>
                                    </span>
                                  ) : null}

                                  {status?.bbr_active ? (
                                    <span className="server-chip bbr" title="Google BBR ускорение сети включено">
                                      <Zap size={11} />
                                      <span>BBR</span>
                                    </span>
                                  ) : null}

                                  {status?.performance_optimized ? (
                                    <span className="server-chip bbr" style={{ background: "rgba(168, 85, 247, 0.15)", color: "#c084fc", borderColor: "rgba(168, 85, 247, 0.3)" }} title="Оптимизация RAM (512MB) и Unix сокетов Valkey активна">
                                      <Cpu size={11} />
                                      <span>RAM 512M</span>
                                    </span>
                                  ) : null}
                                </div>

                                <div className="server-toolbar-actions">
                                  {isMaster && (
                                    <WinTooltip content="Развернуть новую ноду и связать её с этой мастер-панелью">
                                      <button
                                        className="win-button toolbar-action-btn"
                                        onClick={() => handleAddNodeToPanel(srv)}
                                      >
                                        <Plus size={13} style={{ color: "var(--win-accent)" }} />
                                        <span>Нода</span>
                                      </button>
                                    </WinTooltip>
                                  )}
                                </div>
                              </div>
                            )}
                          </div>

                          {!isCollapsed && (
                            <>
                              {/* Offline Alert Banner (Fast Retry & Delete) */}
                          {status && !status.success && (
                            <div className="server-offline-banner">
                              <AlertTriangle size={15} style={{ color: "#f87171", flexShrink: 0 }} />
                              <div style={{ flex: 1, minWidth: 0, fontSize: "12px", color: "var(--win-text-secondary)" }}>
                                <strong style={{ color: "#f87171" }}>Сервер недоступен:</strong>{" "}
                                <span>{status.error || "Нет ответа по SSH. Проверьте сеть, порт или обновите статус."}</span>
                              </div>
                              <div style={{ display: "flex", gap: "6px", flexShrink: 0 }}>
                                <WinTooltip content="Повторить опрос состояния">
                                  <button
                                    className="win-button"
                                    style={{ height: "26px", fontSize: "11.5px" }}
                                    disabled={isLoading}
                                    onClick={() => handleInspectServer(srv)}
                                  >
                                    <RefreshCw size={11} className={isLoading ? "animate-spin" : ""} />
                                    <span>Повторить</span>
                                  </button>
                                </WinTooltip>
                                <WinTooltip content="Удалить недоступный сервер из каталога">
                                  <button
                                    className="win-button danger-subtle"
                                    style={{ height: "26px", fontSize: "11.5px", color: "#f87171" }}
                                    onClick={() => handleDeleteServer(srv.id)}
                                  >
                                    <Trash2 size={11} />
                                    <span>Удалить</span>
                                  </button>
                                </WinTooltip>
                              </div>
                            </div>
                          )}

                          {/* Server Sub-tabs Navigation Bar (Contextual by Role) */}
                          <div className="server-subtabs-bar">
                            {isMaster ? (
                              <>
                                <button
                                  className={`server-subtab ${currentSubTab === "overview" ? "active" : ""}`}
                                  onClick={() => setServerSubTab((prev: any) => ({ ...prev, [srv.id]: "overview" }))}
                                >
                                  <Gauge size={13} />
                                  <span>Обзор & Службы</span>
                                  <span className="server-subtab-badge">
                                    {runningCount}/{totalCount}
                                  </span>
                                </button>

                                <button
                                  className={`server-subtab ${currentSubTab === "maintenance" ? "active" : ""}`}
                                  onClick={() => setServerSubTab((prev: any) => ({ ...prev, [srv.id]: "maintenance" }))}
                                >
                                  <Wrench size={13} />
                                  <span>Обслуживание & Бэкапы</span>
                                  {status?.backups && status.backups.length > 0 && (
                                    <span className="server-subtab-badge">
                                      {status.backups.length}
                                    </span>
                                  )}
                                </button>
                              </>
                            ) : (
                              <>
                                <button
                                  className={`server-subtab ${currentSubTab === "overview" ? "active" : ""}`}
                                  onClick={() => setServerSubTab((prev: any) => ({ ...prev, [srv.id]: "overview" }))}
                                >
                                  <Radio size={13} />
                                  <span>Статус & Служба</span>
                                  <span
                                    className="server-subtab-badge"
                                    style={{
                                      background: hasRunningNode ? "rgba(16,185,129,0.2)" : "rgba(251,191,36,0.2)",
                                      color: hasRunningNode ? "#34d399" : "#fbbf24",
                                    }}
                                  >
                                    {hasRunningNode ? "В сети" : "Ожидает"}
                                  </span>
                                </button>

                                <button
                                  className={`server-subtab ${currentSubTab === "security" ? "active" : ""}`}
                                  onClick={() => setServerSubTab((prev: any) => ({ ...prev, [srv.id]: "security" }))}
                                >
                                  <Shield size={13} />
                                  <span>Защита от ТСПУ & Сеть</span>
                                </button>
                              </>
                            )}
                          </div>

                          {/* =======================================================
                              MASTER PANEL: TAB 1 (OVERVIEW & SERVICES)
                              ======================================================= */}
                          {isMaster && currentSubTab === "overview" && (
                            <div key={`subtab-${srv.id}-master-overview`} className="subtab-anim-enter">
                              {/* Resource Metrics KPI Grid */}
                              {status && status.success && status.metrics && (
                                <div className="server-metrics-grid">
                                  {/* RAM */}
                                  <div className="metric-card">
                                    <div className="metric-header">
                                      <div style={{ display: "flex", alignItems: "center", gap: "5px" }}>
                                        <Cpu size={12} style={{ color: "var(--win-accent)" }} />
                                        <span>Память (RAM)</span>
                                      </div>
                                    </div>
                                    <div className="metric-value">
                                      {status.metrics.ram_used_mb} / {status.metrics.ram_total_mb} MB
                                    </div>
                                    <div className="metric-bar-track">
                                      <div
                                        className={`metric-bar-fill ${
                                          status.metrics.ram_percent > 85
                                            ? "danger"
                                            : status.metrics.ram_percent > 70
                                            ? "warning"
                                            : "normal"
                                        }`}
                                        style={{ width: `${Math.min(status.metrics.ram_percent, 100)}%` }}
                                      />
                                    </div>
                                    <div className="metric-sub">{status.metrics.ram_percent}% использовано</div>
                                  </div>

                                  {/* Disk */}
                                  <div className="metric-card">
                                    <div className="metric-header" style={{ justifyContent: "space-between" }}>
                                      <div style={{ display: "flex", alignItems: "center", gap: "5px" }}>
                                        <HardDrive size={12} style={{ color: "#38bdf8" }} />
                                        <span>Диск (/)</span>
                                      </div>
                                      <WinTooltip content="Безопасная очистка Docker-слоев и логов journalctl">
                                        <button
                                          style={{
                                            padding: "1px 6px",
                                            fontSize: "10px",
                                            display: "flex",
                                            alignItems: "center",
                                            gap: "3px",
                                            border: "1px solid var(--win-card-stroke)",
                                            borderRadius: "4px",
                                            background: "transparent",
                                            color: "var(--win-text-secondary)",
                                            cursor: "pointer",
                                          }}
                                          disabled={cleaningDiskServerId === srv.id}
                                          onClick={() => handleCleanServerDisk(srv)}
                                        >
                                          <Trash2 size={9} className={cleaningDiskServerId === srv.id ? "animate-spin" : ""} />
                                          <span>{cleaningDiskServerId === srv.id ? "Очистка..." : "Очистить"}</span>
                                        </button>
                                      </WinTooltip>
                                    </div>
                                    <div className="metric-value">
                                      {status.metrics.disk_used} / {status.metrics.disk_total}
                                    </div>
                                    <div className="metric-bar-track">
                                      <div
                                        className={`metric-bar-fill ${
                                          status.metrics.disk_percent > 85
                                            ? "danger"
                                            : status.metrics.disk_percent > 70
                                            ? "warning"
                                            : "normal"
                                        }`}
                                        style={{ width: `${Math.min(status.metrics.disk_percent, 100)}%` }}
                                      />
                                    </div>
                                    <div className="metric-sub">{status.metrics.disk_percent}% заполнено</div>
                                  </div>

                                  {/* Uptime */}
                                  <div className="metric-card">
                                    <div className="metric-header">
                                      <div style={{ display: "flex", alignItems: "center", gap: "5px" }}>
                                        <Clock size={12} style={{ color: "#34d399" }} />
                                        <span>Время работы</span>
                                      </div>
                                    </div>
                                    <div className="metric-value" style={{ fontSize: "13px" }}>
                                      {status.metrics.uptime || "В сети"}
                                    </div>
                                    <div style={{ height: "5px" }} />
                                    <div className="metric-sub">Непрерывная работа ОС</div>
                                  </div>

                                  {/* Containers Count */}
                                  <div className="metric-card">
                                    <div className="metric-header">
                                      <div style={{ display: "flex", alignItems: "center", gap: "5px" }}>
                                        <Layers size={12} style={{ color: "#c084fc" }} />
                                        <span>Службы стека</span>
                                      </div>
                                    </div>
                                    <div className="metric-value">
                                      {runningCount} / {totalCount} онлайн
                                    </div>
                                    <div style={{ height: "5px" }} />
                                    <div className="metric-sub">
                                      {runningCount === totalCount ? "Все службы активны" : "Есть остановленные службы"}
                                    </div>
                                  </div>

                                  {/* CPU Steal (VPS Overselling) */}
                                  <div className="metric-card">
                                    <div className="metric-header">
                                      <div style={{ display: "flex", alignItems: "center", gap: "5px" }}>
                                        <Activity
                                          size={12}
                                          style={{
                                            color:
                                              (status.metrics.cpu_steal || 0) >= 15
                                                ? "#ef4444"
                                                : (status.metrics.cpu_steal || 0) >= 5
                                                ? "#f59e0b"
                                                : "#22c55e",
                                          }}
                                        />
                                        <span>CPU Steal (VPS)</span>
                                      </div>
                                    </div>
                                    <div className="metric-value">
                                      {status.metrics.cpu_steal !== undefined ? `${status.metrics.cpu_steal.toFixed(1)}%` : "0.0%"}
                                    </div>
                                    <div className="metric-bar-track">
                                      <div
                                        className={`metric-bar-fill ${
                                          (status.metrics.cpu_steal || 0) >= 15
                                            ? "danger"
                                            : (status.metrics.cpu_steal || 0) >= 5
                                            ? "warning"
                                            : "normal"
                                        }`}
                                        style={{ width: `${Math.min((status.metrics.cpu_steal || 0) * 4, 100)}%` }}
                                      />
                                    </div>
                                    <div className="metric-sub">
                                      {(status.metrics.cpu_steal || 0) >= 15
                                        ? "Критическая деградация VPS (хостер)"
                                        : (status.metrics.cpu_steal || 0) >= 5
                                        ? "Умеренная нагрузка от соседей"
                                        : "Норма (шумные соседи отсутствуют)"}
                                    </div>
                                  </div>
                                </div>
                              )}

                              {/* Services Table */}
                              <div className="overview-stack-section">
                                <div className="overview-stack-header">
                                  <div className="overview-stack-title">
                                    <Layers size={14} style={{ color: "var(--win-accent)" }} />
                                    <span>Службы стека Remnawave</span>
                                    <span className="badge">{runningCount} из {totalCount} в сети</span>
                                  </div>
                                  <div style={{ display: "flex", gap: "8px" }}>
                                    <WinTooltip content="Загрузить свежие Docker-образы Remnawave и перезапустить контейнеры">
                                      <button
                                        className="win-button"
                                        style={{ height: "26px", fontSize: "11.5px" }}
                                        disabled={actionLoading[`update-stack-${srv.id}`]}
                                        onClick={() => handleUpdateStack(srv)}
                                      >
                                        {actionLoading[`update-stack-${srv.id}`] ? (
                                          <RefreshCw size={12} className="animate-spin" />
                                        ) : (
                                          <Download size={12} style={{ color: "var(--win-accent)" }} />
                                        )}
                                        <span>Обновить образы</span>
                                      </button>
                                    </WinTooltip>

                                    <WinTooltip content="Перезапустить все контейнеры стека">
                                      <button
                                        className="win-button"
                                        style={{ height: "26px", fontSize: "11.5px" }}
                                        disabled={actionLoading[`restart-all-${srv.id}`]}
                                        onClick={() => handleRestartAll(srv)}
                                      >
                                        {actionLoading[`restart-all-${srv.id}`] ? (
                                          <RefreshCw size={12} className="animate-spin" />
                                        ) : (
                                          <RotateCcw size={12} />
                                        )}
                                        <span>Перезапустить стек</span>
                                      </button>
                                    </WinTooltip>
                                  </div>
                                </div>

                                {status && status.success && status.containers && (
                                  <div className="server-containers-table-wrap">
                                    <table className="server-containers-table">
                                      <thead>
                                        <tr>
                                          <th className="col-service">Служба</th>
                                          <th className="col-image">Docker Образ</th>
                                          <th className="col-status">Состояние</th>
                                          <th className="col-actions">Управление</th>
                                        </tr>
                                      </thead>
                                      <tbody>
                                        {status.containers.map((c, cIdx) => {
                                          const meta = getServiceMeta(c.name);
                                          const statusStr = (c.status || "").toLowerCase();
                                          const healthyField = ((c as any).healthy || "").toLowerCase();
                                          const isRestarting = statusStr.startsWith("restarting") || healthyField === "restarting";
                                          const isUnhealthy = !isRestarting && (statusStr.includes("unhealthy") || healthyField === "unhealthy");
                                          const isStarting = !isRestarting && !isUnhealthy && (statusStr.includes("starting") || healthyField === "starting");
                                          const isExplicitHealthy = !isRestarting && !isUnhealthy && !isStarting && (statusStr.includes("(healthy)") || healthyField === "healthy");
                                          const isRunningStandard = c.running && !isRestarting && !isUnhealthy && !isStarting;

                                          const dotClass = isUnhealthy || isRestarting
                                            ? "unhealthy"
                                            : isStarting
                                            ? "starting"
                                            : isRunningStandard || isExplicitHealthy
                                            ? "running"
                                            : "stopped";

                                          return (
                                            <tr key={cIdx}>
                                              <td className="col-service" title={c.name}>
                                                <div style={{ display: "flex", alignItems: "center", gap: "9px" }}>
                                                  <span className={`status-dot-pulse ${dotClass}`} />
                                                  <div style={{ display: "flex", alignItems: "center", justifyContent: "center", width: "24px", height: "24px", flexShrink: 0 }}>
                                                    {meta.icon}
                                                  </div>
                                                  <div style={{ display: "flex", flexDirection: "column", minWidth: 0 }}>
                                                    <span style={{ fontWeight: 600, color: "var(--win-text-primary)", whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis" }}>
                                                      {c.name}
                                                    </span>
                                                    <span style={{ fontSize: "11px", color: "var(--win-text-secondary)", opacity: 0.85, whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis" }}>
                                                      {meta.role}
                                                    </span>
                                                  </div>
                                                </div>
                                              </td>
                                              <td className="col-image" title={c.image}>
                                                <span style={{ background: "var(--win-chip-bg)", border: "1px solid var(--win-chip-border)", padding: "2px 6px", borderRadius: "4px" }}>
                                                  {c.image}
                                                </span>
                                              </td>
                                              <td className="col-status" title={c.status}>
                                                {isExplicitHealthy ? (
                                                  <span className="container-health-badge healthy" title="Контейнер активен и успешно проходит встроенную проверку здоровья (Docker Healthcheck)">
                                                    <CheckCircle2 size={12} />
                                                    <span>{c.status.replace(/\s*\(healthy\)/i, "")} · Здоров</span>
                                                  </span>
                                                ) : isStarting ? (
                                                  <span className="container-health-badge starting" title="Служба запускается и прогревается (Docker Healthcheck Starting)">
                                                    <Clock size={12} />
                                                    <span>{c.status.replace(/\s*\(health:\s*starting\)/i, "").replace(/\s*starting/i, "")} · Запуск...</span>
                                                  </span>
                                                ) : isUnhealthy ? (
                                                  <span className="container-health-badge unhealthy" title="Внутренний сбой службы или отказ healthcheck-пробы">
                                                    <AlertTriangle size={12} />
                                                    <span>{c.status.replace(/\s*\(unhealthy\)/i, "")} · Сбой</span>
                                                  </span>
                                                ) : isRestarting ? (
                                                  <span className="container-health-badge restarting" title="Контейнер в циклическом перезапуске (CrashLoopBackOff)">
                                                    <RotateCcw size={12} className="animate-spin" />
                                                    <span>Перезапуск...</span>
                                                  </span>
                                                ) : isRunningStandard ? (
                                                  <span className="container-health-badge running" title="Контейнер активен и функционирует">
                                                    <Check size={12} />
                                                    <span>{c.status}</span>
                                                  </span>
                                                ) : (
                                                  <span className="container-health-badge stopped" title="Контейнер остановлен">
                                                    <AlertCircle size={12} />
                                                    <span>{c.status || "Остановлен"}</span>
                                                  </span>
                                                )}
                                              </td>
                                              <td className="col-actions">
                                                <div style={{ display: "inline-flex", gap: "6px", justifyContent: "flex-end", flexWrap: "nowrap" }}>
                                                  <WinTooltip content="Перезапустить контейнер">
                                                    <button
                                                      className="table-action-btn"
                                                      disabled={actionLoading[`restart-${srv.id}-${c.name}`]}
                                                      onClick={() => handleRestartContainer(srv, c.name)}
                                                    >
                                                      {actionLoading[`restart-${srv.id}-${c.name}`] ? (
                                                        <RefreshCw size={12} className="animate-spin" />
                                                      ) : (
                                                        <RotateCcw size={12} />
                                                      )}
                                                      <span>Рестарт</span>
                                                    </button>
                                                  </WinTooltip>

                                                  <WinTooltip content="Просмотр логов контейнера">
                                                    <button
                                                      className="table-action-btn"
                                                      onClick={() => handleOpenLogsModal(srv, c.name)}
                                                    >
                                                      <TerminalIcon size={12} />
                                                      <span>Логи</span>
                                                    </button>
                                                  </WinTooltip>
                                                </div>
                                              </td>
                                            </tr>
                                          );
                                        })}
                                      </tbody>
                                    </table>
                                  </div>
                                )}

                                {status && !status.success && (
                                  <div
                                    style={{
                                      display: "flex",
                                      alignItems: "center",
                                      justifyContent: "space-between",
                                      padding: "16px 18px",
                                      background: "rgba(239, 68, 68, 0.08)",
                                      border: "1px solid rgba(239, 68, 68, 0.25)",
                                      borderLeft: "4px solid #ef4444",
                                      borderRadius: "6px",
                                      marginTop: "12px",
                                      gap: "12px",
                                      flexWrap: "wrap",
                                    }}
                                  >
                                    <div style={{ display: "flex", alignItems: "center", gap: "12px", minWidth: 0, flex: 1 }}>
                                      <AlertTriangle size={20} style={{ color: "#ef4444", flexShrink: 0 }} />
                                      <div>
                                        <div style={{ fontSize: "13px", fontWeight: 600, color: "var(--win-text-primary)" }}>
                                          Сервер не отвечает по SSH ({srv.host}:{srv.port || 22})
                                        </div>
                                        <div style={{ fontSize: "11.5px", color: "var(--win-text-secondary)", marginTop: "2px", lineHeight: 1.4 }}>
                                          {status.error || "Соединение отклонено или превышен таймаут ответа. Проверьте сеть и статус VPS."}
                                        </div>
                                      </div>
                                    </div>
                                    <div style={{ display: "flex", gap: "8px", flexShrink: 0 }}>
                                      <WinTooltip content="Повторить опрос состояния">
                                        <button
                                          className="win-button"
                                          style={{ height: "28px", fontSize: "11.5px" }}
                                          disabled={isLoading}
                                          onClick={() => handleInspectServer(srv)}
                                        >
                                          <RefreshCw size={12} className={isLoading ? "animate-spin" : ""} />
                                          <span>Повторить опрос</span>
                                        </button>
                                      </WinTooltip>
                                      <WinTooltip content="Удалить сервер из каталога">
                                        <button
                                          className="win-button danger-subtle"
                                          style={{ height: "28px", fontSize: "11.5px" }}
                                          onClick={() => handleDeleteServer(srv.id)}
                                        >
                                          <Trash2 size={12} />
                                          <span>Удалить сервер</span>
                                        </button>
                                      </WinTooltip>
                                    </div>
                                  </div>
                                )}
                              </div>
                            </div>
                          )}

                          {/* =======================================================
                              MASTER PANEL: TAB 2 (MAINTENANCE & BACKUPS)
                              ======================================================= */}
                          {isMaster && currentSubTab === "maintenance" && (
                            <div key={`subtab-${srv.id}-master-maintenance`} className="subtab-anim-enter" style={{ padding: "16px", display: "flex", flexDirection: "column", gap: "20px" }}>
                              {/* Section 1: Backups */}
                              <div
                                style={{
                                  background: "var(--win-card-bg)",
                                  border: "1px solid var(--win-card-stroke)",
                                  borderRadius: "8px",
                                  padding: "16px 18px",
                                }}
                              >
                                <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "14px", flexWrap: "wrap", gap: "10px" }}>
                                  <div>
                                    <div style={{ display: "flex", alignItems: "center", gap: "8px", fontSize: "14px", fontWeight: 600 }}>
                                      <Archive size={16} style={{ color: "var(--win-accent)" }} />
                                      <span>Резервные копии базы данных (PostgreSQL)</span>
                                    </div>
                                    <div style={{ fontSize: "12px", color: "var(--win-text-secondary)", marginTop: "2px" }}>
                                      Хранятся локально на сервере в каталоге /opt/remnawave/backups
                                    </div>
                                  </div>
                                  <button
                                    className="win-button hero-primary"
                                    style={{ height: "30px", fontSize: "12px" }}
                                    disabled={actionLoading[`backup-${srv.id}`]}
                                    onClick={() => handleCreateBackup(srv)}
                                  >
                                    {actionLoading[`backup-${srv.id}`] ? (
                                      <RefreshCw size={12} className="animate-spin" />
                                    ) : (
                                      <Archive size={12} />
                                    )}
                                    <span>Создать бэкап сейчас</span>
                                  </button>
                                </div>

                                {status?.backups && status.backups.length > 0 ? (
                                  <div className="backup-list-container">
                                    {status.backups.map((b, bIdx) => (
                                      <div key={bIdx} className="backup-item-card">
                                        <div className="backup-item-left">
                                          <div className="backup-item-icon">
                                            <Archive size={16} />
                                          </div>
                                          <div className="backup-item-details">
                                            <div className="backup-item-name">{b.file_name}</div>
                                            <div className="backup-item-meta">
                                              <span>{b.size}</span>
                                              <span className="backup-item-dot">•</span>
                                              <span>{b.created_at}</span>
                                            </div>
                                          </div>
                                        </div>
                                        <div className="backup-item-actions">
                                          <WinTooltip content="Скачать бэкап на компьютер">
                                            <button
                                              className="table-action-btn"
                                              onClick={() => handleDownloadBackup(srv, b.file_name)}
                                            >
                                              <Download size={12} />
                                              <span>Скачать</span>
                                            </button>
                                          </WinTooltip>
                                          <WinTooltip content="Восстановить БД из этой копии">
                                            <button
                                              className="table-action-btn"
                                              onClick={() => setRestoreModal({ server: srv, fileName: b.file_name })}
                                            >
                                              <RotateCcw size={12} />
                                              <span>Восстановить</span>
                                            </button>
                                          </WinTooltip>
                                        </div>
                                      </div>
                                    ))}
                                  </div>
                                ) : (
                                  <div style={{ padding: "16px", textAlign: "center", color: "var(--win-text-tertiary)", fontSize: "12.5px" }}>
                                    Резервных копий пока нет. Нажмите «Создать бэкап сейчас» для создания первого снимка БД.
                                  </div>
                                )}
                              </div>

                              <div
                                style={{
                                  background: "var(--win-card-bg)",
                                  border: "1px solid var(--win-card-stroke)",
                                  borderRadius: "8px",
                                  padding: "16px 18px",
                                }}
                              >
                                <div style={{ display: "flex", alignItems: "center", gap: "8px", fontSize: "14px", fontWeight: 600, marginBottom: "14px" }}>
                                  <Wrench size={16} style={{ color: "#c084fc" }} />
                                  <span>Сервисные операции и восстановление</span>
                                </div>

                                <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(260px, 1fr))", gap: "12px" }}>
                                  {/* Sync DB */}
                                  <div className="tool-card" style={{ padding: "12px 14px" }}>
                                    <div style={{ display: "flex", alignItems: "center", gap: "8px", marginBottom: "6px" }}>
                                      <Database size={15} style={{ color: "#818cf8" }} />
                                      <span style={{ fontWeight: 600, fontSize: "13px" }}>Синхронизация пароля БД</span>
                                    </div>
                                    <p style={{ fontSize: "12px", color: "var(--win-text-secondary)", margin: "0 0 10px 0", lineHeight: 1.4 }}>
                                      Сверяет пароль PostgreSQL с файлом .env и восстанавливает соединение при ошибках авторизации.
                                    </p>
                                    <button
                                      className="win-button"
                                      style={{ width: "100%", justifyContent: "center", height: "28px", fontSize: "12px" }}
                                      disabled={actionLoading[`fix-db-${srv.id}`]}
                                      onClick={() => handleFixDBSync(srv)}
                                    >
                                      {actionLoading[`fix-db-${srv.id}`] ? <RefreshCw size={12} className="animate-spin" /> : <Wrench size={12} />}
                                      <span>Синхронизировать БД</span>
                                    </button>
                                  </div>

                                  {/* Reset Superadmin */}
                                  <div className="tool-card" style={{ padding: "12px 14px" }}>
                                    <div style={{ display: "flex", alignItems: "center", gap: "8px", marginBottom: "6px" }}>
                                      <Key size={15} style={{ color: "var(--win-accent)" }} />
                                      <span style={{ fontWeight: 600, fontSize: "13px" }}>Сброс супер-админа</span>
                                    </div>
                                    <p style={{ fontSize: "12px", color: "var(--win-text-secondary)", margin: "0 0 10px 0", lineHeight: 1.4 }}>
                                      Экстренный сброс: открывает мастер первичного создания администратора при входе в панель.
                                    </p>
                                    <button
                                      className="win-button"
                                      style={{ width: "100%", justifyContent: "center", height: "28px", fontSize: "12px" }}
                                      disabled={actionLoading[`reset-admin-${srv.id}`]}
                                      onClick={() => handleResetSuperadmin(srv)}
                                    >
                                      {actionLoading[`reset-admin-${srv.id}`] ? <RefreshCw size={12} className="animate-spin" /> : <Key size={12} />}
                                      <span>Сбросить администратора</span>
                                    </button>
                                  </div>

                                  {/* Prune Docker */}
                                  <div className="tool-card" style={{ padding: "12px 14px" }}>
                                    <div style={{ display: "flex", alignItems: "center", gap: "8px", marginBottom: "6px" }}>
                                      <Trash2 size={15} style={{ color: "#f87171" }} />
                                      <span style={{ fontWeight: 600, fontSize: "13px" }}>Безопасная очистка диска</span>
                                    </div>
                                    <p style={{ fontSize: "12px", color: "var(--win-text-secondary)", margin: "0 0 10px 0", lineHeight: 1.4 }}>
                                      Удаляет неиспользуемые слои Docker, кэши buildx и сжимает системные журналы journalctl без удаления данных БД.
                                    </p>
                                    <button
                                      className="win-button"
                                      style={{ width: "100%", justifyContent: "center", height: "28px", fontSize: "12px" }}
                                      disabled={cleaningDiskServerId === srv.id}
                                      onClick={() => handleCleanServerDisk(srv)}
                                    >
                                      {cleaningDiskServerId === srv.id ? <RefreshCw size={12} className="animate-spin" /> : <Trash2 size={12} />}
                                      <span>Очистить диск сервера</span>
                                    </button>
                                  </div>

                                  {/* BBR */}
                                  <div className="tool-card" style={{ padding: "12px 14px" }}>
                                    <div style={{ display: "flex", alignItems: "center", gap: "8px", marginBottom: "6px" }}>
                                      <Zap size={15} style={{ color: "#fbbf24" }} />
                                      <span style={{ fontWeight: 600, fontSize: "13px" }}>Ускорение сети (BBR)</span>
                                    </div>
                                    <p style={{ fontSize: "12px", color: "var(--win-text-secondary)", margin: "0 0 10px 0", lineHeight: 1.4 }}>
                                      Включает сетевой алгоритм Google BBR + FQ qdisc в ядре Linux для снижения сетевых задержек.
                                    </p>
                                    <button
                                      className="win-button"
                                      style={{ width: "100%", justifyContent: "center", height: "28px", fontSize: "12px" }}
                                      disabled={actionLoading[`bbr-${srv.id}`]}
                                      onClick={() => handleOptimizeBBR(srv)}
                                    >
                                      {actionLoading[`bbr-${srv.id}`] ? <RefreshCw size={12} className="animate-spin" /> : <Zap size={12} />}
                                      <span>{status?.bbr_active ? "BBR активен" : "Включить BBR"}</span>
                                    </button>
                                  </div>

                                  {/* RAM & Unix Sockets Performance Optimization */}
                                  <div className="tool-card" style={{ padding: "12px 14px" }}>
                                    <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: "6px" }}>
                                      <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
                                        <Cpu size={15} style={{ color: "#c084fc" }} />
                                        <span style={{ fontWeight: 600, fontSize: "13px" }}>RAM Boost & Sockets</span>
                                      </div>
                                      <span className={`tool-badge-status ${status?.performance_optimized ? "active" : "inactive"}`}>
                                        {status?.performance_optimized ? "512MB + Sockets" : "Стандартный"}
                                      </span>
                                    </div>
                                    <p style={{ fontSize: "12px", color: "var(--win-text-secondary)", margin: "0 0 10px 0", lineHeight: 1.4 }}>
                                      Valkey через Unix Domain Socket, 512MB shared memory для Postgres, ulimits 1M и тюнинг swappiness ядра Linux.
                                    </p>
                                    <button
                                      className={`win-button ${status?.performance_optimized ? "" : "hero-primary"}`}
                                      style={{ width: "100%", justifyContent: "center", height: "28px", fontSize: "12px" }}
                                      disabled={actionLoading[`perf-toggle-${srv.id}`]}
                                      onClick={() => handleTogglePerformance(srv, !!status?.performance_optimized)}
                                    >
                                      {actionLoading[`perf-toggle-${srv.id}`] ? (
                                        <RefreshCw size={12} className="animate-spin" />
                                      ) : status?.performance_optimized ? (
                                        <RotateCcw size={12} />
                                      ) : (
                                        <Zap size={12} />
                                      )}
                                      <span>{status?.performance_optimized ? "Вернуть стандартный режим" : "Включить RAM Boost"}</span>
                                    </button>
                                  </div>
                                </div>
                              </div>
                            </div>
                          )}

                          {/* =======================================================
                              EDGE NODE: TAB 1 (STATUS & SERVICE)
                              ======================================================= */}
                          {!isMaster && currentSubTab === "overview" && (
                            <div key={`subtab-${srv.id}-node-overview`} className="subtab-anim-enter" style={{ padding: "16px", display: "flex", flexDirection: "column", gap: "16px" }}>
                              {hasRunningNode ? (
                                <div
                                  style={{
                                    display: "flex",
                                    justifyContent: "space-between",
                                    alignItems: "center",
                                    padding: "16px 18px",
                                    background: "rgba(16, 185, 129, 0.08)",
                                    border: "1px solid rgba(16, 185, 129, 0.25)",
                                    borderLeft: "4px solid #10b981",
                                    borderRadius: "8px",
                                    flexWrap: "wrap",
                                    gap: "12px",
                                  }}
                                >
                                  <div style={{ display: "flex", alignItems: "center", gap: "12px" }}>
                                    <CheckCircle2 size={24} style={{ color: "#10b981", flexShrink: 0 }} />
                                    <div>
                                      <div style={{ fontSize: "14px", fontWeight: 600, color: "var(--win-text-primary)" }}>
                                        Xray Reality Нода активна и принимает клиентский трафик
                                      </div>
                                      <div style={{ fontSize: "12px", color: "var(--win-text-secondary)", marginTop: "2px" }}>
                                        Контейнер <code>remnanode</code> запущен. Внутренний порт связи с панелью: <strong>2222</strong> (изолирован в DOCKER-USER).
                                      </div>
                                    </div>
                                  </div>
                                  <div style={{ display: "flex", gap: "8px" }}>
                                    <button
                                      className="win-button"
                                      style={{ height: "30px", fontSize: "12px" }}
                                      onClick={() => handleOpenLogsModal(srv, "remnanode")}
                                    >
                                      <TerminalIcon size={12} />
                                      <span>Логи ноды</span>
                                    </button>
                                    <button
                                      className="win-button"
                                      style={{ height: "30px", fontSize: "12px" }}
                                      disabled={actionLoading[`restart-${srv.id}-remnanode`]}
                                      onClick={() => handleRestartContainer(srv, "remnanode")}
                                    >
                                      <RotateCcw size={12} />
                                      <span>Перезапустить</span>
                                    </button>
                                    <button
                                      className="win-button"
                                      style={{ height: "30px", fontSize: "12px" }}
                                      onClick={() => setNodeModal({ server: srv, secretKey: "", loading: false })}
                                    >
                                      <Radio size={12} />
                                      <span>Сменить ключ</span>
                                    </button>
                                  </div>
                                </div>
                              ) : (
                                <div
                                  style={{
                                    display: "flex",
                                    justifyContent: "space-between",
                                    alignItems: "center",
                                    padding: "16px 18px",
                                    background: "rgba(251, 191, 36, 0.08)",
                                    border: "1px solid rgba(251, 191, 36, 0.25)",
                                    borderLeft: "4px solid #fbbf24",
                                    borderRadius: "8px",
                                    flexWrap: "wrap",
                                    gap: "12px",
                                  }}
                                >
                                  <div style={{ display: "flex", alignItems: "flex-start", gap: "12px" }}>
                                    <AlertTriangle size={24} style={{ color: "#fbbf24", flexShrink: 0, marginTop: "2px" }} />
                                    <div>
                                      <div style={{ fontSize: "14px", fontWeight: 600, color: "var(--win-text-primary)" }}>
                                        Нода ожидает привязки SECRET_KEY от мастер-панели
                                      </div>
                                      <div style={{ fontSize: "12px", color: "var(--win-text-secondary)", marginTop: "4px", lineHeight: 1.5 }}>
                                        {(srv as any).parent_master_id ? (
                                          <>
                                            Нода уже зарегистрирована в мастер-панели через API.<br />
                                            Откройте панель в браузере, кликните по ноде в списке <strong>Nodes</strong>, скопируйте поле <strong>Secret Key (SECRET_KEY)</strong> и привяжите здесь.
                                          </>
                                        ) : (
                                          <>
                                            1. Откройте панель Remnawave → <strong>Nodes → Create Node</strong>.<br />
                                            2. Укажите адрес <code>{srv.host}</code> и порт <code>2222</code>.<br />
                                            3. Скопируйте <code>SECRET_KEY</code> и привяжите его к ноде.
                                          </>
                                        )}
                                      </div>
                                    </div>
                                  </div>
                                  <button
                                    className="win-button hero-primary"
                                    style={{ height: "34px", fontSize: "12.5px" }}
                                    onClick={() => setNodeModal({ server: srv, secretKey: "", loading: false })}
                                  >
                                    <Lock size={13} />
                                    <span>Привязать SECRET_KEY</span>
                                  </button>
                                </div>
                              )}

                              {/* Node System Metrics */}
                              {status && status.success && status.metrics && (
                                <div className="server-metrics-grid">
                                  {/* RAM */}
                                  <div className="metric-card">
                                    <div className="metric-header">
                                      <div style={{ display: "flex", alignItems: "center", gap: "5px" }}>
                                        <Cpu size={12} style={{ color: "var(--win-accent)" }} />
                                        <span>Память (RAM)</span>
                                      </div>
                                    </div>
                                    <div className="metric-value">
                                      {status.metrics.ram_used_mb} / {status.metrics.ram_total_mb} MB
                                    </div>
                                    <div className="metric-bar-track">
                                      <div
                                        className={`metric-bar-fill ${status.metrics.ram_percent > 85 ? "danger" : "normal"}`}
                                        style={{ width: `${Math.min(status.metrics.ram_percent, 100)}%` }}
                                      />
                                    </div>
                                    <div className="metric-sub">{status.metrics.ram_percent}% использовано</div>
                                  </div>

                                  {/* Disk */}
                                  <div className="metric-card">
                                    <div className="metric-header" style={{ justifyContent: "space-between" }}>
                                      <div style={{ display: "flex", alignItems: "center", gap: "5px" }}>
                                        <HardDrive size={12} style={{ color: "#38bdf8" }} />
                                        <span>Диск (/)</span>
                                      </div>
                                      <WinTooltip content="Безопасная очистка Docker-слоев и логов journalctl">
                                        <button
                                          style={{
                                            padding: "1px 6px",
                                            fontSize: "10px",
                                            display: "flex",
                                            alignItems: "center",
                                            gap: "3px",
                                            border: "1px solid var(--win-card-stroke)",
                                            borderRadius: "4px",
                                            background: "transparent",
                                            color: "var(--win-text-secondary)",
                                            cursor: "pointer",
                                          }}
                                          disabled={cleaningDiskServerId === srv.id}
                                          onClick={() => handleCleanServerDisk(srv)}
                                        >
                                          <Trash2 size={9} className={cleaningDiskServerId === srv.id ? "animate-spin" : ""} />
                                          <span>{cleaningDiskServerId === srv.id ? "Очистка..." : "Очистить"}</span>
                                        </button>
                                      </WinTooltip>
                                    </div>
                                    <div className="metric-value">
                                      {status.metrics.disk_used} / {status.metrics.disk_total}
                                    </div>
                                    <div className="metric-bar-track">
                                      <div
                                        className="metric-bar-fill normal"
                                        style={{ width: `${Math.min(status.metrics.disk_percent, 100)}%` }}
                                      />
                                    </div>
                                    <div className="metric-sub">{status.metrics.disk_percent}% заполнено</div>
                                  </div>

                                  {/* Ping */}
                                  <div className="metric-card">
                                    <div className="metric-header">
                                      <div style={{ display: "flex", alignItems: "center", gap: "5px" }}>
                                        <Zap size={12} style={{ color: "#34d399" }} />
                                        <span>Задержка (Ping)</span>
                                      </div>
                                    </div>
                                    <div className="metric-value" style={{ fontSize: "14px" }}>
                                      {status.latency_ms ? `${status.latency_ms} ms` : "—"}
                                    </div>
                                    <div style={{ height: "5px" }} />
                                    <div className="metric-sub">Отклик SSH соединения</div>
                                  </div>

                                  {/* Uptime */}
                                  <div className="metric-card">
                                    <div className="metric-header">
                                      <div style={{ display: "flex", alignItems: "center", gap: "5px" }}>
                                        <Clock size={12} style={{ color: "#c084fc" }} />
                                        <span>Аптайм</span>
                                      </div>
                                    </div>
                                    <div className="metric-value" style={{ fontSize: "13px" }}>
                                      {status.metrics.uptime || "В сети"}
                                    </div>
                                    <div style={{ height: "5px" }} />
                                    <div className="metric-sub">Время непрерывной работы</div>
                                  </div>

                                  {/* CPU Steal (VPS Overselling) */}
                                  <div className="metric-card">
                                    <div className="metric-header">
                                      <div style={{ display: "flex", alignItems: "center", gap: "5px" }}>
                                        <Activity
                                          size={12}
                                          style={{
                                            color:
                                              (status.metrics.cpu_steal || 0) >= 15
                                                ? "#ef4444"
                                                : (status.metrics.cpu_steal || 0) >= 5
                                                ? "#f59e0b"
                                                : "#22c55e",
                                          }}
                                        />
                                        <span>CPU Steal (VPS)</span>
                                      </div>
                                    </div>
                                    <div className="metric-value">
                                      {status.metrics.cpu_steal !== undefined ? `${status.metrics.cpu_steal.toFixed(1)}%` : "0.0%"}
                                    </div>
                                    <div className="metric-bar-track">
                                      <div
                                        className={`metric-bar-fill ${
                                          (status.metrics.cpu_steal || 0) >= 15
                                            ? "danger"
                                            : (status.metrics.cpu_steal || 0) >= 5
                                            ? "warning"
                                            : "normal"
                                        }`}
                                        style={{ width: `${Math.min((status.metrics.cpu_steal || 0) * 4, 100)}%` }}
                                      />
                                    </div>
                                    <div className="metric-sub">
                                      {(status.metrics.cpu_steal || 0) >= 15
                                        ? "Критическая деградация VPS (хостер)"
                                        : (status.metrics.cpu_steal || 0) >= 5
                                        ? "Умеренная нагрузка от соседей"
                                        : "Норма (шумные соседи отсутствуют)"}
                                    </div>
                                  </div>
                                </div>
                              )}
                            </div>
                          )}

                          {/* =======================================================
                              EDGE NODE: TAB 2 (SECURITY & TSPU)
                              ======================================================= */}
                          {!isMaster && currentSubTab === "security" && (
                            <div key={`subtab-${srv.id}-node-security`} className="subtab-anim-enter" style={{ padding: "16px", display: "flex", flexDirection: "column", gap: "16px" }}>
                              {/* Hero: Anti-Probing TSPU Protection */}
                              <div
                                style={{
                                  display: "flex",
                                  justifyContent: "space-between",
                                  alignItems: "center",
                                  padding: "16px 18px",
                                  background: "linear-gradient(135deg, rgba(16, 185, 129, 0.12) 0%, rgba(16, 185, 129, 0.02) 100%)",
                                  border: "1px solid rgba(16, 185, 129, 0.3)",
                                  borderLeft: "4px solid #10b981",
                                  borderRadius: "8px",
                                  flexWrap: "wrap",
                                  gap: "14px",
                                }}
                              >
                                <div style={{ display: "flex", alignItems: "flex-start", gap: "12px", maxWidth: "560px" }}>
                                  <ShieldCheck size={22} style={{ color: "#10b981", flexShrink: 0, marginTop: "2px" }} />
                                  <div>
                                    <div style={{ fontSize: "14px", fontWeight: 600, color: "var(--win-text-primary)" }}>
                                      Защита ноды от ТСПУ и активных сканеров (remna-mask)
                                    </div>
                                    <div style={{ fontSize: "12px", color: "var(--win-text-secondary)", marginTop: "4px", lineHeight: 1.5 }}>
                                      Изолирует порт 2222 в DOCKER-USER (доступен только панели), устраняет деанонимизирующие аномалии TCP RST (Stealth DROP) и обеспечивает SelfSteal Decoy на порту 8444.
                                    </div>
                                  </div>
                                </div>
                                <div style={{ display: "flex", gap: "8px", flexWrap: "wrap" }}>
                                  <WinTooltip content="Синтетическая проверка реального VPN-трафика и детекция блокировок ТСПУ">
                                    <button
                                      className="win-button"
                                      style={{ height: "34px", fontSize: "12.5px" }}
                                      onClick={() => handleOpenVPNProbeModal(srv)}
                                    >
                                      <Activity size={14} style={{ color: "#c084fc" }} />
                                      <span>Тест VPN-трафика</span>
                                    </button>
                                  </WinTooltip>
                                  <button
                                    className="win-button hero-primary"
                                    style={{ height: "34px", fontSize: "12.5px" }}
                                    onClick={() => handleOpenAudit(srv)}
                                  >
                                    <ShieldCheck size={14} />
                                    <span>Запустить аудит ТСПУ</span>
                                  </button>
                                </div>
                              </div>

                              <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(280px, 1fr))", gap: "14px" }}>
                                {/* Torrent & Spam Filter */}
                                <div className="tool-card" style={{ padding: "14px 16px" }}>
                                  <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", marginBottom: "8px" }}>
                                    <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
                                      <ShieldAlert size={16} style={{ color: "#34d399" }} />
                                      <span style={{ fontWeight: 600, fontSize: "13.5px" }}>Защита от торрентов & спама</span>
                                    </div>
                                    <span className={`tool-badge-status ${status?.anti_abuse_active ? "active" : "inactive"}`}>
                                      {status?.anti_abuse_active ? "Активна" : "Отключена"}
                                    </span>
                                  </div>
                                  <p style={{ fontSize: "12px", color: "var(--win-text-secondary)", margin: "0 0 12px 0", lineHeight: 1.4 }}>
                                    Блокирует BitTorrent трафик на уровне nftables и закрывает исходящие порты почтового спама (25, 465, 587) для защиты от абуз хостера.
                                  </p>
                                  <button
                                    className={`win-button ${status?.anti_abuse_active ? "" : "hero-primary"}`}
                                    style={{ width: "100%", justifyContent: "center", height: "30px", fontSize: "12px" }}
                                    disabled={actionLoading[`anti-abuse-${srv.id}`]}
                                    onClick={() => handleToggleAntiAbuse?.(srv, !!status?.anti_abuse_active)}
                                  >
                                    {actionLoading[`anti-abuse-${srv.id}`] ? (
                                      <RefreshCw size={12} className="animate-spin" />
                                    ) : (
                                      <ShieldAlert size={12} />
                                    )}
                                    <span>{status?.anti_abuse_active ? "Отключить фильтрацию" : "Включить фильтрацию"}</span>
                                  </button>
                                </div>

                                {/* Cloudflare WARP */}
                                <div className="tool-card" style={{ padding: "14px 16px" }}>
                                  <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", marginBottom: "8px" }}>
                                    <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
                                      <Cloud size={16} style={{ color: "#38bdf8" }} />
                                      <span style={{ fontWeight: 600, fontSize: "13.5px" }}>Cloudflare WARP Outbound</span>
                                    </div>
                                    <span className={`tool-badge-status ${status?.warp_active ? "active" : "inactive"}`}>
                                      {status?.warp_active ? "Активен" : "Не установлен"}
                                    </span>
                                  </div>
                                  <p style={{ fontSize: "12px", color: "var(--win-text-secondary)", margin: "0 0 12px 0", lineHeight: 1.4 }}>
                                    Маршрутизирует исходящий трафик ноды через сеть Cloudflare для обхода блокировок сервисов (OpenAI ChatGPT, Netflix).
                                  </p>
                                  <button
                                    className="win-button"
                                    style={{ width: "100%", justifyContent: "center", height: "30px", fontSize: "12px" }}
                                    disabled={status?.warp_active || actionLoading[`warp-${srv.id}`]}
                                    onClick={() => handleInstallWARP(srv)}
                                  >
                                    {actionLoading[`warp-${srv.id}`] ? (
                                      <RefreshCw size={12} className="animate-spin" />
                                    ) : (
                                      <Cloud size={12} />
                                    )}
                                    <span>{status?.warp_active ? "WARP подключен" : "Установить WARP"}</span>
                                  </button>
                                </div>

                                {/* BBR Optimization */}
                                <div className="tool-card" style={{ padding: "14px 16px" }}>
                                  <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", marginBottom: "8px" }}>
                                    <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
                                      <Zap size={16} style={{ color: "#fbbf24" }} />
                                      <span style={{ fontWeight: 600, fontSize: "13.5px" }}>Сетевое ускорение (BBR)</span>
                                    </div>
                                    <span className={`tool-badge-status ${status?.bbr_active ? "active" : "inactive"}`}>
                                      {status?.bbr_active ? "BBR активен" : "Cubic"}
                                    </span>
                                  </div>
                                  <p style={{ fontSize: "12px", color: "var(--win-text-secondary)", margin: "0 0 12px 0", lineHeight: 1.4 }}>
                                    Активирует алгоритм Google BBR + FQ qdisc в ядре Linux, снижая буферизацию и сетевые задержки для клиентов VPN.
                                  </p>
                                  <button
                                    className="win-button"
                                    style={{ width: "100%", justifyContent: "center", height: "30px", fontSize: "12px" }}
                                    disabled={actionLoading[`bbr-${srv.id}`]}
                                    onClick={() => handleOptimizeBBR(srv)}
                                  >
                                    {actionLoading[`bbr-${srv.id}`] ? (
                                      <RefreshCw size={12} className="animate-spin" />
                                    ) : (
                                      <Zap size={12} />
                                    )}
                                    <span>{status?.bbr_active ? "BBR настроен" : "Включить BBR"}</span>
                                  </button>
                                </div>
                              </div>
                            </div>
                          )}
                            </>
                          )}
                        </div>
                      );
});

ServerCard.displayName = "ServerCard";
