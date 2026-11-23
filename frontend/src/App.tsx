import React, { useState, useEffect, useRef, useMemo, useCallback } from "react";

import {

  Server,

  Shield,

  Terminal as TerminalIcon,

  CheckCircle2,

  AlertTriangle,
  AlertOctagon,

  Folder,

  Play,

  RotateCcw,

  
  Wand2,

  Layers,

  Globe,

  Lock,
  Unlock,
  KeyRound,

  ChevronRight,
  ChevronDown,
  ChevronUp,
  ChevronsDownUp,
  ChevronsUpDown,

  HardDrive,

  Cpu,

  RefreshCw,

  Trash2,

  ExternalLink,

  Copy,

  Info,

  Check,

  Search,

  Activity,

  Wrench,

  Clock,

  ArrowRight,

  Database,

  Radio,

  X,

  Zap,

  Download,

  Key,

  Archive,

  Gauge,

  ShieldCheck,

  ShieldAlert,

  Edit3,

  Cloud,

  Eye,

  EyeOff,

  Sun,

  Moon,

  Menu,

  LayoutGrid,

  List,

  AlertCircle,

  QrCode,

  Dices,

  Plus,
  MoreVertical,
  MapPin,
  Tag,
  Workflow,
  FileCode,
  Users,
  Share2,
  Network,
  Hash,
  LayoutDashboard,
  Sliders,
  Ghost,
  LayoutTemplate,
  Link2,
  Compass,
  Shuffle,
  Package,
  BookOpen,
  GitBranch,
  User,
} from "lucide-react";



import * as AppBindings from "../bindings/remna-ui/app";
import * as types from "../bindings/remna-ui/internal/types/models";
import { ServerCard } from "./components/servers/ServerCard";
import { Events } from "@wailsio/runtime";

const {
  AttachNodeSecret,
  CheckDNS,
  CheckIPBlock,
  ProbeVPNTraffic,
  CleanServerDisk,
  TestSSH,
  GenerateConfig,
  StartDeployment,
  GetServers,
  SaveServer,
  DeleteServer,
  GetServerStatus,
  GetContainerLogs,
  RestartContainer,
  RestartAllStack,
  FixDatabaseSync,
  OpenBrowserURL,
  SelectDirectory,
  SelectSSHKeyFile,
  OptimizeNetworkBBR,
  ConfigureDockerMirrors,
  CreateBackup,
  DownloadBackup,
  RestoreBackup,
  UpdateStackImages,
  PruneDockerSystem,
  ResetSuperadmin,
  RenewSSL,
  ToggleAntiAbuse,
  ToggleStackPerformance,
  InstallCloudflareWARP,
  ConfigureRealitySelfSteal,
  TestPanelAPI,
  CreateNodeInPanel,
  GetPanelConfigProfiles,
  GetPanelSquads,
  GetPanelNodes,
  ApplyAntiProbingFirewall,
  AuditNodeSecurity,
  DeploySelfStealDecoy,
  AutoFixSubscriptionToken,
  UninstallStack,
  CheckAppUpdate,
  ResetCircuitBreaker,
  GetVaultStatus,
  UnlockVault,
  LockVault,
  SetMasterPassword,
  DisableMasterPassword,
  SetAutoLockMinutes,
  SetSecureValue,
  GetSecureValue,
} = AppBindings;

const EventsOn = (eventName: string, callback: (...args: any[]) => void) => {
  return Events.On(eventName, (ev: any) => {
    callback(ev && ev.data !== undefined ? ev.data : ev);
  });
};
const EventsOff = (eventName: string) => {
  Events.Off(eventName);
};

import { InspectLogsModal } from "./components/modals/InspectLogsModal";
import { NodeSecretModal } from "./components/modals/NodeSecretModal";
import { EditServerModal, EditServerData } from "./components/modals/EditServerModal";
import { RestoreBackupModal } from "./components/modals/RestoreBackupModal";
import { AntiProbingAuditModal } from "./components/modals/AntiProbingAuditModal";
import { IPBlockModal } from "./components/modals/IPBlockModal";
import { VPNProbeModal } from "./components/modals/VPNProbeModal";
import { VaultUnlockModal } from "./components/modals/VaultUnlockModal";
import { VaultSettingsModal } from "./components/modals/VaultSettingsModal";

import { SettingsCard } from "./components/ui/SettingsCard";

import { SettingsExpander } from "./components/ui/SettingsExpander";

import { ShimmerSkeleton, ServerCardSkeleton } from "./components/ui/ShimmerSkeleton";

import { InfoBar } from "./components/ui/InfoBar";

import { CountrySelector } from "./components/ui/CountrySelector";



interface LogEntry {
  status: "info" | "step" | "success" | "error" | "stdout" | "stderr";
  message: string;
  isError?: boolean;
}

interface ToastMessage {
  id: number;
  type: "success" | "error" | "info";
  message: string;
  isExiting?: boolean;
}

// Pre-compiled static regex for maximum throughput in live log streaming
const ANSI_REGEX = /[\u001b\u009b][[()#;?]*(?:[0-9]{1,4}(?:;[0-9]{0,4})*)?[0-9A-ORZcf-nqry=><]/g;

// ANSI Escape sequences and control characters cleaner for Terminal UX
const stripAnsi = (str: string): string => {
  return (str || "")
    .replace(ANSI_REGEX, "")
    .replace(/\r/g, "");
};

// Fast pre-classifier for log lines to eliminate O(N) regex checks during re-renders
const checkIsRealErrorLog = (status: string, message: string): boolean => {
  if (status === "error") return true;
  if (status === "stderr") {
    const m = (message || "").toLowerCase();
    // Harmless progress/status indicators in stderr from apt/docker
    if (
      m.includes("pulling fs layer") ||
      m.includes("downloading") ||
      m.includes("extracting") ||
      m.includes("verifying checksum") ||
      m.includes("download complete") ||
      m.includes("pull complete") ||
      m.includes("already exists") ||
      m.includes("waiting") ||
      m.includes("creating") ||
      m.includes("created") ||
      m.includes("starting") ||
      m.includes("started") ||
      m.includes("network ") ||
      m.includes("volume ") ||
      m.includes("container ") ||
      m.includes("get:") ||
      m.includes("hit:") ||
      m.includes("ign:") ||
      m.includes("selecting previously unselected") ||
      m.includes("preparing to unpack") ||
      m.includes("unpacking") ||
      m.includes("setting up") ||
      m.includes("processing triggers") ||
      m.includes("debconf:") ||
      m.includes("warning:") ||
      m.includes("notice:") ||
      m.includes("info:")
    ) {
      return false;
    }
    return /(?:fatal|failed|failure|panic|exception|permission denied|command not found|address already in use|connection refused)/i.test(message);
  }
  return false;
};

const isIpAddress = (val: string) => /^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$/.test((val || "").trim());

export function App() {
  const [activeTab, setActiveTab] = useState<"wizard" | "servers" | "about">("wizard");
  const [step, setStep] = useState<number>(1);
  const [stepDirection, setStepDirection] = useState<"forward" | "backward">("forward");

  const [theme, setTheme] = useState<"dark" | "light">(() => {

    return (localStorage.getItem("remna_theme") as "dark" | "light") || "dark";

  });



  useEffect(() => {

    document.documentElement.setAttribute("data-theme", theme);

    localStorage.setItem("remna_theme", theme);

  }, [theme]);



  const toggleTheme = () => {

    setTheme((prev) => (prev === "dark" ? "light" : "dark"));

  };



  // Sidebar Collapsed State (WinUI NavigationView compact mode)

  const [sidebarCollapsed, setSidebarCollapsed] = useState<boolean>(() => {

    return localStorage.getItem("remna_sidebar_collapsed") === "true";

  });



  const toggleSidebar = () => {

    setSidebarCollapsed((prev) => {

      const next = !prev;

      localStorage.setItem("remna_sidebar_collapsed", String(next));

      return next;

    });

  };



  useEffect(() => {

    const handleKeyDown = (e: KeyboardEvent) => {

      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "b") {

        e.preventDefault();

        toggleSidebar();

      }

    };

    window.addEventListener("keydown", handleKeyDown);

    return () => window.removeEventListener("keydown", handleKeyDown);

  }, []);



  // Notifications (Toast)

  const [toasts, setToasts] = useState<ToastMessage[]>([]);

  const addToast = (type: "success" | "error" | "info", rawMessage: string) => {
    let message = rawMessage;
    // Map cryptic Go / networking errors to user-friendly Russian messages
    if (message.includes("context deadline exceeded") || message.includes("Client.Timeout exceeded")) {
      message = "Превышено время ожидания ответа сервера (таймаут)";
    } else if (message.includes("connection refused") || message.includes("ECONNREFUSED")) {
      message = "Сервер отклонил подключение (служба не запущена или порт закрыт)";
    } else if (message.includes("no route to host") || message.includes("network is unreachable")) {
      message = "Сервер недоступен в сети (проверьте IP-адрес)";
    } else if (message.includes("A114") || message.includes("already exists in database")) {
      message = "Профиль с таким именем уже существует в базе данных панели";
    }

    // Clean up multiline outputs (e.g. docker pull logs) to keep toast compact and clean
    if (message.includes("\n") || message.length > 140) {
      const lines = message.split(/\r?\n/).map(l => l.trim()).filter(Boolean);
      const summaryLine = lines.find(l => l.includes("успешно") || l.includes("Ошибка") || l.includes("Сбой") || l.startsWith("✓")) || lines[lines.length - 1];
      message = summaryLine && summaryLine.length < 140 ? summaryLine : message.slice(0, 137) + "...";
    }

    const id = Date.now() + Math.random();
    setToasts((prev) => [...prev, { id, type, message, isExiting: false }]);
    setTimeout(() => {
      setToasts((prev) => prev.map((t) => (t.id === id ? { ...t, isExiting: true } : t)));
    }, 3900);
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, 4200);
  };



  // Form State

  const [preset, setPreset] = useState<string>("all-in-one");

  const [components, setComponents] = useState<string[]>(["panel", "node", "proxy", "subscription"]);

  const [panelDomain, setPanelDomain] = useState<string>("");

  const [subDomain, setSubDomain] = useState<string>("");

  const [proxy, setProxy] = useState<string>("nginx");

  const [nodePort, setNodePort] = useState<number>(2222);

  const [nodeSecret, setNodeSecret] = useState<string>("");

  const [nodeLinkMode, setNodeLinkMode] = useState<"auto" | "manual">("auto");

  const [nodePanelUrl, setNodePanelUrl] = useState<string>(() => localStorage.getItem("rw_panel_url") || "");

  const [nodeApiToken, setNodeApiToken] = useState<string>(() => localStorage.getItem("rw_api_token") || "");

  const [showNodeApiToken, setShowNodeApiToken] = useState<boolean>(false);

  const [nodeName, setNodeName] = useState<string>("DE - Frankfurt #1");

  const [nodeCountry, setNodeCountry] = useState<string>("DE");

  const [panelApiTesting, setPanelApiTesting] = useState<boolean>(false);

  const [panelApiStatus, setPanelApiStatus] = useState<types.PanelInfoResult | null>(null);

  // Remnawave Node Integration & Routing State
  const [nodeIntegrationMode, setNodeIntegrationMode] = useState<"turnkey" | "custom">("turnkey");
  const [panelProfiles, setPanelProfiles] = useState<types.PanelProfileItem[]>([]);
  const [panelSquads, setPanelSquads] = useState<types.PanelSquadItem[]>([]);
  const [loadingPanelEntities, setLoadingPanelEntities] = useState<boolean>(false);
  const [selectedProfileInbound, setSelectedProfileInbound] = useState<string>("");
  const [selectedSquadUuid, setSelectedSquadUuid] = useState<string>("default");
  const [createSubscriptionHost, setCreateSubscriptionHost] = useState<boolean>(true);
  const [customHostAddress, setCustomHostAddress] = useState<string>("");

  const [secretGate, setSecretGate] = useState<string>("admin_gate");

  const [useGate, setUseGate] = useState<boolean>(true);

  const [useCamouflage, setUseCamouflage] = useState<boolean>(true);

  const [useUfw, setUseUfw] = useState<boolean>(true);
  const [optimizePerformance, setOptimizePerformance] = useState<boolean>(true);

  const [useCloudflareDns, setUseCloudflareDns] = useState<boolean>(false);

  const [cloudflareToken, setCloudflareToken] = useState<string>("");

  const [showCloudflareToken, setShowCloudflareToken] = useState<boolean>(false);

  const [outDir, setOutDir] = useState<string>("./out");



  // Anti-Probing & Anti-Censorship (remna-mask) State

  const [hideNodePort, setHideNodePort] = useState<boolean>(true);

  const [stealthDrop, setStealthDrop] = useState<boolean>(true);

  const [deployDecoy, setDeployDecoy] = useState<boolean>(true);

  const [portMode, setPortMode] = useState<"stealth443" | "classic">("stealth443");

  const [decoyTemplate, setDecoyTemplate] = useState<string>("random");

  const [decoyCloneUrl, setDecoyCloneUrl] = useState<string>("");

  const [torrentBlocker, setTorrentBlocker] = useState<boolean>(true);

  const [blockSpamPorts, setBlockSpamPorts] = useState<boolean>(true);

  const [xrayCoreType, setXrayCoreType] = useState<"official" | "jolymmiles">("jolymmiles");

  // Admin Account Creation Mode in Wizard Step 2 ("auto" | "custom" | "manual")
  const [adminCreationMode, setAdminCreationMode] = useState<"auto" | "custom" | "manual">("auto");
  const [adminUsername, setAdminUsername] = useState<string>("admin");
  const [adminPassword, setAdminPassword] = useState<string>("");
  const [showAdminPassword, setShowAdminPassword] = useState<boolean>(false);

  const [customPanelIp, setCustomPanelIp] = useState<string>("");

  const [auditModal, setAuditModal] = useState<{

    server: types.SavedServer | null;

    loading: boolean;

    applying: boolean;

    audit: types.NodeSecurityAudit | null;

  }>({ server: null, loading: false, applying: false, audit: null });



  // Remote SSH State

  const [deployMode, setDeployMode] = useState<"generate" | "remote">("remote");

  const [sshHost, setSshHost] = useState<string>("");

  const [sshPort, setSshPort] = useState<number>(22);

  const [sshUser, setSshUser] = useState<string>("root");

  const [sshPass, setSshPass] = useState<string>("");

  const [sshKey, setSshKey] = useState<string>("");

  // Wizard IP Availability Check in Russia
  const [wizardIPChecking, setWizardIPChecking] = useState<boolean>(false);
  const [wizardIPResult, setWizardIPResult] = useState<types.IPBlockCheckResult | null>(null);

  // Standalone Server IP Block Modal
  const [ipCheckModal, setIpCheckModal] = useState<{
    server: types.SavedServer;
    result: types.IPBlockCheckResult | null;
    loading: boolean;
  } | null>(null);

  // Standalone Server VPN Synthetic Probe Modal
  const [vpnProbeModal, setVpnProbeModal] = useState<{
    server: types.SavedServer;
    result: types.VPNProbeResult | null;
    loading: boolean;
  } | null>(null);

  // Clean Disk Loading State per Server
  const [cleaningDiskServerId, setCleaningDiskServerId] = useState<string | null>(null);

  // Local Secrets & Vault Master Password State
  const [vaultStatus, setVaultStatus] = useState<types.VaultStatus | null>(null);
  const [vaultUnlockOpen, setVaultUnlockOpen] = useState<boolean>(false);
  const [vaultSettingsOpen, setVaultSettingsOpen] = useState<boolean>(false);



  // Live DNS Status

  const [dnsStatus, setDnsStatus] = useState<types.DNSResult | null>(null);

  const [dnsLoading, setDnsLoading] = useState<boolean>(false);

  const [subDnsStatus, setSubDnsStatus] = useState<types.DNSResult | null>(null);

  const [subDnsLoading, setSubDnsLoading] = useState<boolean>(false);



  // Decoy Domain State

  const [nodeDecoyDomain, setNodeDecoyDomain] = useState<string>("");

  const [decoyDnsStatus, setDecoyDnsStatus] = useState<types.DNSResult | null>(null);

  const [decoyDnsLoading, setDecoyDnsLoading] = useState<boolean>(false);



  // SSH Test Status

  const [sshStatus, setSshStatus] = useState<types.SSHTestResult | null>(null);

  const [sshLoading, setSshLoading] = useState<boolean>(false);



  // Deployment Logs
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const logBufferRef = useRef<LogEntry[]>([]);
  const logFlushTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const flushLogs = useCallback(() => {
    if (logFlushTimerRef.current) {
      clearTimeout(logFlushTimerRef.current);
      logFlushTimerRef.current = null;
    }
    if (logBufferRef.current.length === 0) return;
    const chunk = logBufferRef.current;
    logBufferRef.current = [];
    setLogs((prev) => {
      const next = [...prev, ...chunk];
      return next.length > 1500 ? next.slice(-1200) : next;
    });
  }, []);

  const [isDeploying, setIsDeploying] = useState<boolean>(false);

  const [deployFinished, setDeployFinished] = useState<boolean>(false);

  const [deploySuccess, setDeploySuccess] = useState<boolean>(false);

  const [deployedWebUrl, setDeployedWebUrl] = useState<string>("");
  const [deployedAdminUser, setDeployedAdminUser] = useState<string>("");
  const [deployedAdminPass, setDeployedAdminPass] = useState<string>("");
  const [deployedSubUrl, setDeployedSubUrl] = useState<string>("");
  const [deployedAdminMode, setDeployedAdminMode] = useState<string>("auto");
  const [showDeployedPass, setShowDeployedPass] = useState<boolean>(false);

  const generateClientSafePassword = useCallback(() => {
    const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-";
    const uppers = "ABCDEFGHIJKLMNOPQRSTUVWXYZ";
    const lowers = "abcdefghijklmnopqrstuvwxyz";
    const digits = "0123456789";
    let res = "";
    res += uppers[Math.floor(Math.random() * uppers.length)];
    res += lowers[Math.floor(Math.random() * lowers.length)];
    res += digits[Math.floor(Math.random() * digits.length)];
    for (let i = 0; i < 25; i++) {
      res += chars[Math.floor(Math.random() * chars.length)];
    }
    const arr = res.split("").sort(() => 0.5 - Math.random());
    setAdminPassword(arr.join(""));
  }, []);

  const validateStep = (currentStep: number): boolean => {
    if (currentStep === 1) {
      if (components.length === 0) {
        addToast("error", "Выберите хотя бы один компонент для развертывания.");
        return false;
      }
      return true;
    }
    if (currentStep === 2) {
      if (preset === "node-only") {
        if (!nodeName.trim()) {
          addToast("error", "Укажите имя ноды.");
          return false;
        }
        if (nodeLinkMode === "auto") {
          if (!nodePanelUrl.trim() || !nodeApiToken.trim()) {
            addToast("error", "Для автоматической привязки ноды укажите URL панели и API токен.");
            return false;
          }
          if (nodeIntegrationMode === "custom" && !selectedProfileInbound) {
            addToast("error", "Выберите профиль конфигурации и входящий порт (Inbound) для кастомной привязки ноды.");
            return false;
          }
        } else {
          if (!nodeSecret.trim()) {
            addToast("error", "Укажите секретный ключ связи ноды с панелью (NODE_SECRET).");
            return false;
          }
        }
        if (deployDecoy && !nodeDecoyDomain.trim()) {
          addToast("error", "Укажите домен маскировочного сайта для SelfSteal Decoy или отключите опцию.");
          return false;
        }
        if (deployDecoy && decoyTemplate === "clone" && !decoyCloneUrl.trim()) {
          addToast("error", "Укажите URL сайта для клонирования маскировочного сайта.");
          return false;
        }
        if (!nodePort || nodePort < 1 || nodePort > 65535) {
          addToast("error", "Порт связи с панелью должен быть в диапазоне от 1 до 65535 (по умолчанию 2222).");
          return false;
        }
        if ([22, 80, 443].includes(nodePort)) {
          addToast("error", `Порт ${nodePort} зарезервирован для системных служб (SSH/HTTP/HTTPS). Выберите другой порт связи.`);
          return false;
        }
        return true;
      } else {
        const cleanDomain = panelDomain.trim().replace(/^https?:\/\//i, "").replace(/\/+$/, "");
        if (!cleanDomain) {
          addToast("error", "Укажите домен панели управления (например: panel.domain.com).");
          return false;
        }
        if (useSubDomain) {
          const cleanSub = subDomain.trim().replace(/^https?:\/\//i, "").replace(/\/+$/, "");
          if (!cleanSub) {
            addToast("error", "Укажите отдельный домен для страницы подписок.");
            return false;
          }
          if (cleanSub.toLowerCase() === cleanDomain.toLowerCase()) {
            addToast("error", "Домен страницы подписок должен отличаться от домена панели управления.");
            return false;
          }
        }
        if (useCloudflareDns && !cloudflareToken.trim()) {
          addToast("error", "Включен выпуск SSL через Cloudflare DNS, но не указан Cloudflare API токен.");
          return false;
        }
        if (useGate) {
          const cleanGate = secretGate.trim().replace(/^\/+/, "");
          if (!cleanGate) {
            addToast("error", "Укажите путь секретного гейта (например: admin_gate).");
            return false;
          }
          if (!/^[a-zA-Z0-9_\-]+$/.test(cleanGate)) {
            addToast("error", "Секретный гейт может содержать только латинские буквы, цифры, дефис и подчеркивание.");
            return false;
          }
        }
        if (components.includes("panel") && adminCreationMode === "custom") {
          if (!adminUsername.trim()) {
            addToast("error", "Укажите логин суперадминистратора.");
            return false;
          }
          if (adminPassword.length < 24 || !/[A-Z]/.test(adminPassword) || !/[a-z]/.test(adminPassword) || !/[0-9]/.test(adminPassword)) {
            addToast("error", "Пароль суперадминистратора должен содержать минимум 24 символа, включая заглавную букву, строчную букву и цифру (политика безопасности Remnawave).");
            return false;
          }
        }
        return true;
      }
    }
    return true;
  };

  const changeStep = (target: number) => {
    if (target > step) {
      for (let s = step; s < target; s++) {
        if (!validateStep(s)) {
          return;
        }
      }
    }
    setStepDirection(target >= step ? "forward" : "backward");
    setStep(target);
  };

  const terminalEndRef = useRef<HTMLDivElement>(null);

  const [copiedLogs, setCopiedLogs] = useState<boolean>(false);

  // Terminal UX: Stepper & Collapsible Drawer (Stage 3)
  const [isTerminalExpanded, setIsTerminalExpanded] = useState<boolean>(false);
  const [logFilter, setLogFilter] = useState<"all" | "error" | "info">("all");
  const [logSearch, setLogSearch] = useState<string>("");

  // Memoized log filtering to prevent laggy re-renders during high-volume log streams
  const terminalErrorCount = useMemo(() => logs.filter((l) => l.isError).length, [logs]);

  const filteredLogs = useMemo(() => {
    const q = logSearch.trim().toLowerCase();
    return logs.filter((log) => {
      if (logFilter === "error" && !log.isError) return false;
      if (logFilter === "info" && log.isError) return false;
      if (q) return log.message.toLowerCase().includes(q);
      return true;
    });
  }, [logs, logFilter, logSearch]);




  // Saved Servers

  const [savedServers, setSavedServers] = useState<types.SavedServer[]>([]);

  const [loadingServersList, setLoadingServersList] = useState<boolean>(true);

  // Active Context Menu for Server Cards (Stage 2: Visual De-cluttering)
  const [activeServerMenu, setActiveServerMenu] = useState<string | null>(null);

  useEffect(() => {
    const handleClickOutside = () => setActiveServerMenu(null);
    if (activeServerMenu) {
      document.addEventListener("click", handleClickOutside);
      return () => document.removeEventListener("click", handleClickOutside);
    }
  }, [activeServerMenu]);

  const [serverStatus, setServerStatus] = useState<{ [id: string]: types.ServerStatusResult }>({});

  const [serverLoading, setServerLoading] = useState<{ [id: string]: boolean }>({});

  const [collapsedServers, setCollapsedServers] = useState<Record<string, boolean>>({});

  const toggleServerCollapse = useCallback((id: string) => {
    setCollapsedServers((prev) => ({
      ...prev,
      [id]: !prev[id],
    }));
  }, []);

  const toggleAllCollapse = useCallback(() => {
    setCollapsedServers((prev) => {
      const allCollapsed = savedServers.length > 0 && savedServers.every((s) => prev[s.id]);
      const next: Record<string, boolean> = {};
      for (const s of savedServers) {
        next[s.id] = !allCollapsed;
      }
      return next;
    });
  }, [savedServers]);

  const areAllCollapsed = useMemo(() => {
    return savedServers.length > 0 && savedServers.every((s) => collapsedServers[s.id]);
  }, [savedServers, collapsedServers]);

  // Memoized server classification and hierarchy tree
  const isMasterServer = useCallback(
    (s: types.SavedServer) => {
      const st = serverStatus[s.id];
      const hasMaster = Boolean(st?.containers?.some((c) => c.name === "remnawave"));
      const hasNode = st?.containers?.some((c) => c.name === "remnanode" && c.running);
      const isNode = Boolean(hasNode || st?.node_present);
      return s.role === "master"
        ? true
        : s.role === "node"
        ? false
        : hasMaster
        ? true
        : isNode
        ? false
        : Boolean(s.web_url && !isIpAddress(s.host));
    },
    [serverStatus]
  );

  const { masterServers, childNodesByMaster, standaloneServers } = useMemo(() => {
    const masters = savedServers.filter((s) => (s as any).is_external_master || isMasterServer(s));
    const masterIds = new Set(masters.map((m) => m.id));
    const children = new Map<string, types.SavedServer[]>();
    const standalone: types.SavedServer[] = [];

    savedServers.forEach((s) => {
      const isM = (s as any).is_external_master || isMasterServer(s);
      if (!isM) {
        const pId = (s as any).parent_master_id;
        if (pId && masterIds.has(pId)) {
          const list = children.get(pId) || [];
          list.push(s);
          children.set(pId, list);
        } else {
          standalone.push(s);
        }
      }
    });

    return { masterServers: masters, childNodesByMaster: children, standaloneServers: standalone };
  }, [savedServers, isMasterServer]);

  const [actionLoading, setActionLoading] = useState<{ [key: string]: boolean }>({});

  // App Update & System Info (About Tab)
  const [checkingUpdate, setCheckingUpdate] = useState<boolean>(false);
  const [updateInfo, setUpdateInfo] = useState<types.AppUpdateInfo | null>(null);

  const handleCheckAppUpdate = async () => {
    setCheckingUpdate(true);
    try {
      const res = await CheckAppUpdate("XZ-LITE/remna-ui");
      setUpdateInfo(res);
      if (res.has_update) {
        addToast("info", `Доступна новая версия: ${res.latest_version}`);
      } else if (res.error) {
        addToast("error", `Не удалось проверить обновления: ${res.error}`);
      } else {
        addToast("success", "У вас установлена последняя версия Remnawave UI!");
      }
    } catch (e: any) {
      addToast("error", `Ошибка проверки: ${e}`);
    } finally {
      setCheckingUpdate(false);
    }
  };

  const handleCopySystemInfo = () => {
    const info = `Remnawave Deployment & Manager Suite
Версия: 2.0.0 (WinUI 3 Fluent Edition)
Стек: Go 1.27 + Wails v3 + React 19 + TypeScript + WebView2
Платформа: Windows 11 / x64
Хранилище: Windows DPAPI (enc:dpapi)`;
    navigator.clipboard.writeText(info).then(() => {
      addToast("info", "Сведения о системе скопированы в буфер обмена!");
    });
  };



  // Modal Logs Inspector
  // Reinstall & Clean VPS Modals
  const [reinstallModal, setReinstallModal] = useState<{ server: types.SavedServer; open: boolean } | null>(null);
  const [uninstallModal, setUninstallModal] = useState<{ server: types.SavedServer; open: boolean; confirmHost: string; loading: boolean } | null>(null);


  const [inspectModal, setInspectModal] = useState<{

    server: types.SavedServer;

    container: string;

    logs: string;

    autoRefresh: boolean;

  } | null>(null);



  // Modal Node Secret Attachment

  const [nodeModal, setNodeModal] = useState<{

    server: types.SavedServer;

    secretKey: string;

    loading: boolean;

  } | null>(null);



  // Modal Server Edit

  const [editServerModal, setEditServerModal] = useState<EditServerData | null>(null);



  // Modal Restore Confirmation

  const [restoreModal, setRestoreModal] = useState<{

    server: types.SavedServer;

    fileName: string;

  } | null>(null);



  // Active sub-tab for each server card

  const [serverSubTab, setServerSubTab] = useState<{

    [id: string]: "overview" | "containers" | "tools" | "backups" | "node" | "topology" | "qr" | "maintenance" | "security";

  }>({});



  // Optimized smooth terminal auto-scroll during deploy without CPU animation lag

  const scrollRafRef = useRef<number | null>(null);

  useEffect(() => {

    if (!terminalEndRef.current) return;

    if (scrollRafRef.current) {

      cancelAnimationFrame(scrollRafRef.current);

    }

    scrollRafRef.current = requestAnimationFrame(() => {

      terminalEndRef.current?.scrollIntoView({ behavior: "auto" });

    });

    return () => {

      if (scrollRafRef.current) cancelAnimationFrame(scrollRafRef.current);

    };

  }, [logs]);



  // Vault Management Handlers
  const checkVaultStatus = useCallback(async () => {
    try {
      const st = await GetVaultStatus();
      setVaultStatus(st);
      if (st.enabled && !st.unlocked) {
        setVaultUnlockOpen(true);
        return false;
      }
      return true;
    } catch (e) {
      console.error("Vault status check error:", e);
      return true;
    }
  }, []);

  const handleUnlockVaultAction = async (password: string): Promise<boolean> => {
    try {
      const res = await UnlockVault(password);
      if (res.success) {
        setVaultUnlockOpen(false);
        const st = await GetVaultStatus();
        setVaultStatus(st);
        await loadServers();

        // Migrate legacy localStorage tokens if present
        const localToken = localStorage.getItem("rw_api_token");
        if (localToken) {
          await SetSecureValue("rw_api_token", localToken);
          localStorage.removeItem("rw_api_token");
        }
        return true;
      }
      return false;
    } catch {
      return false;
    }
  };

  const handleLockVaultAction = async () => {
    try {
      await LockVault();
      const st = await GetVaultStatus();
      setVaultStatus(st);
      setVaultUnlockOpen(true);
      await loadServers();
      addToast("info", "Хранилище заблокировано");
    } catch (err: any) {
      addToast("error", `Ошибка блокировки: ${err?.message || err}`);
    }
  };

  const handleSetMasterPasswordAction = async (oldPass: string, newPass: string): Promise<boolean> => {
    const res = await SetMasterPassword(oldPass, newPass);
    if (res.success) {
      await checkVaultStatus();
      await loadServers();
      return true;
    }
    throw new Error(res.error || "Не удалось установить мастер-пароль");
  };

  const handleDisableMasterPasswordAction = async (currentPass: string): Promise<boolean> => {
    const res = await DisableMasterPassword(currentPass);
    if (res.success) {
      await checkVaultStatus();
      await loadServers();
      return true;
    }
    throw new Error(res.error || "Не удалось отключить мастер-пароль");
  };

  const handleSetAutoLockMinutesAction = async (mins: number): Promise<boolean> => {
    const res = await SetAutoLockMinutes(mins);
    if (res.success) {
      await checkVaultStatus();
      return true;
    }
    return false;
  };

  // Load Saved Servers & Check Vault on Mount
  useEffect(() => {
    const initApp = async () => {
      const isUnlocked = await checkVaultStatus();
      if (isUnlocked) {
        const secToken = await GetSecureValue("rw_api_token");
        if (secToken && !nodeApiToken) {
          setNodeApiToken(secToken);
        }
      }
      loadServers();
    };
    initApp();
  }, []);

  // Global Shortcut Ctrl+L for Quick Vault Lock
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "l") {
        if (vaultStatus?.enabled && vaultStatus?.unlocked) {
          e.preventDefault();
          handleLockVaultAction();
        }
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [vaultStatus]);

  const loadServers = async () => {
    setLoadingServersList(true);
    try {
      const list = await GetServers();
      setSavedServers(list || []);

      // If we have servers, auto-open the servers dashboard and poll status
      if (list && list.length > 0) {
        setActiveTab("servers");
        if (Object.keys(serverStatus).length === 0) {
          handleInspectServer(list[0]);
        }
      }
    } catch (e) {
      console.error(e);
    } finally {
      setLoadingServersList(false);
    }
  };



  // Listen to Wails events for live logs with 60ms batching buffer
  useEffect(() => {
    const logHandler = (data: any) => {
      const cleanMsg = stripAnsi(data.message || "");
      if (!cleanMsg && data.status !== "step") return;

      const status = data.status || "info";
      const isErr = checkIsRealErrorLog(status, cleanMsg);
      logBufferRef.current.push({ status, message: cleanMsg, isError: isErr });

      // Throttle state update to at most once every 60ms
      if (!logFlushTimerRef.current) {
        logFlushTimerRef.current = setTimeout(flushLogs, 60);
      }
    };

    const doneHandler = (data: any) => {
      flushLogs();
      setIsDeploying(false);
      setDeployFinished(true);
      setDeploySuccess(data.success);
      if (data.success) {
        if (data.admin_creation_mode) setDeployedAdminMode(data.admin_creation_mode);
        if (data.admin_user) setDeployedAdminUser(data.admin_user);
        if (data.admin_pass) setDeployedAdminPass(data.admin_pass);
        if (data.sub_url) setDeployedSubUrl(data.sub_url);
        if (data.panel_url) setDeployedWebUrl(data.panel_url);
        addToast("success", "Развертывание стека успешно завершено!");
      } else {
        setIsTerminalExpanded(true);
        addToast("error", "Сбой установки. Проверьте журнал ошибок в терминале.");
      }
    };

    EventsOn("deploy:log", logHandler);
    EventsOn("deploy:done", doneHandler);

    return () => {
      EventsOff("deploy:log");
      EventsOff("deploy:done");
      if (logFlushTimerRef.current) {
        clearTimeout(logFlushTimerRef.current);
        logFlushTimerRef.current = null;
      }
    };
  }, [flushLogs]);



  // Preset Selection Handler
  const handleSelectPreset = (id: string) => {
    setPreset(id);
    if (id === "all-in-one") {
      setComponents(["panel", "node", "proxy", "subscription"]);
      setPortMode("stealth443");
    } else if (id === "panel-only") {
      setComponents(["panel", "proxy", "subscription"]);
    } else if (id === "node-only") {
      setComponents(["node"]);
    }
  };



  const toggleComponent = (comp: string) => {

    setComponents((prev) =>

      prev.includes(comp) ? prev.filter((c) => c !== comp) : [...prev, comp]

    );

  };



  // Debounced Live DNS Check

  useEffect(() => {

    if (!panelDomain || panelDomain.length < 3 || !panelDomain.includes(".")) {

      setDnsStatus(null);

      return;

    }

    const timer = setTimeout(async () => {

      setDnsLoading(true);

      try {

        const res = await CheckDNS(panelDomain);

        setDnsStatus(res);

      } catch (err) {

        console.error("DNS check error", err);

      } finally {

        setDnsLoading(false);

      }

    }, 600);



    return () => clearTimeout(timer);

  }, [panelDomain]);



  // Debounced Live DNS Check for Subscription Subdomain

  useEffect(() => {

    if (!subDomain || subDomain.length < 3 || !subDomain.includes(".")) {

      setSubDnsStatus(null);

      return;

    }

    const timer = setTimeout(async () => {

      setSubDnsLoading(true);

      try {

        const res = await CheckDNS(subDomain);

        setSubDnsStatus(res);

      } catch (err) {

        console.error("Subscription DNS check error", err);

      } finally {

        setSubDnsLoading(false);

      }

    }, 600);



    return () => clearTimeout(timer);

  }, [subDomain]);



  useEffect(() => {

    if (!nodeDecoyDomain || nodeDecoyDomain.length < 3 || !nodeDecoyDomain.includes(".")) {

      setDecoyDnsStatus(null);

      return;

    }

    const timer = setTimeout(async () => {

      setDecoyDnsLoading(true);

      try {

        const res = await CheckDNS(nodeDecoyDomain);

        setDecoyDnsStatus(res);

      } catch (err) {

        console.error("Decoy DNS check error", err);

      } finally {

        setDecoyDnsLoading(false);

      }

    }, 600);



    return () => clearTimeout(timer);

  }, [nodeDecoyDomain]);



  // SSH Hardware & Access Test

  const handleTestSSH = async () => {

    if (!sshHost) return;

    setSshLoading(true);

    setSshStatus(null);

    try {

      const res = await TestSSH(sshHost, sshPort, sshUser, sshPass, sshKey);

      setSshStatus(res);

      if (res.success) {

        addToast("success", "SSH соединение подтверждено!");

      } else {

        addToast("error", `Ошибка SSH: ${res.error}`);

      }

    } catch (err: any) {

      setSshStatus(

        new types.SSHTestResult({

          success: false,

          error: err.toString(),

        })

      );

      addToast("error", `Ошибка подключения: ${err}`);

    } finally {

      setSshLoading(false);

    }

  };

  // Check Wizard IP Availability in Russia
  const handleCheckWizardIP = async () => {
    if (!sshHost) return;
    setWizardIPChecking(true);
    setWizardIPResult(null);
    try {
      const res = await CheckIPBlock(sshHost, sshPort || 22);
      setWizardIPResult(res);
      if (res.status === "clean") {
        addToast("success", "IP доступен из РФ (проверка ТСПУ пройдена)");
      } else if (res.status === "blocked") {
        addToast("error", "Внимание: обнаружена блокировка IP в РФ (ТСПУ / РКН)");
      } else if (res.status === "partial") {
        addToast("info", "Частичная доступность IP в РФ");
      } else {
        addToast("error", "Хост не отвечает на проверочные запросы");
      }
    } catch (err: any) {
      addToast("error", `Ошибка проверки IP: ${err}`);
    } finally {
      setWizardIPChecking(false);
    }
  };

  // Open IP Block Check Modal for Saved Server
  const handleOpenIPCheckModal = async (srv: types.SavedServer) => {
    setIpCheckModal({ server: srv, result: null, loading: true });
    try {
      const res = await CheckIPBlock(srv.host, srv.port || 22);
      setIpCheckModal({ server: srv, result: res, loading: false });
    } catch (err: any) {
      addToast("error", `Ошибка проверки IP сервера ${srv.host}: ${err}`);
      setIpCheckModal((prev) => (prev ? { ...prev, loading: false } : null));
    }
  };

  // Refresh IP Block Check Modal
  const handleRefreshIPCheckModal = async () => {
    if (!ipCheckModal?.server) return;
    const srv = ipCheckModal.server;
    setIpCheckModal((prev) => (prev ? { ...prev, loading: true } : null));
    try {
      const res = await CheckIPBlock(srv.host, srv.port || 22);
      setIpCheckModal({ server: srv, result: res, loading: false });
    } catch (err: any) {
      addToast("error", `Ошибка повторной проверки IP: ${err}`);
      setIpCheckModal((prev) => (prev ? { ...prev, loading: false } : null));
    }
  };

  // Open VPN Synthetic Probe Modal
  const handleOpenVPNProbeModal = async (srv: types.SavedServer, customVless?: string) => {
    setVpnProbeModal({ server: srv, result: null, loading: true });
    try {
      const port = srv.port || 443;
      const res = await ProbeVPNTraffic(srv.host, port, srv.decoy_domain || "", customVless || "", srv, srv.password || "");
      setVpnProbeModal({ server: srv, result: res, loading: false });
    } catch (err: any) {
      addToast("error", `Ошибка проверки VPN-трафика сервера ${srv.host}: ${err}`);
      setVpnProbeModal((prev) => (prev ? { ...prev, loading: false } : null));
    }
  };

  // Safe Smart Clean Disk for Server
  const handleCleanServerDisk = async (srv: types.SavedServer) => {
    if (!window.confirm(`Выполнить безопасную очистку диска на ${srv.label || srv.host}?\n\nБудут удалены неиспользуемые слои Docker (без удаления данных БД и томов), очищены кэши сборщика, усечены системные логи journalctl до 2 дней / 100 МБ и удалены временные файлы.`)) {
      return;
    }
    setCleaningDiskServerId(srv.id);
    addToast("info", `Запущена безопасная очистка диска на ${srv.label || srv.host}...`);
    try {
      const res = await CleanServerDisk(srv, srv.password || "");
      if (res && res.success) {
        addToast("success", `Диск сервера очищен! ${res.freed_space || "Операция завершена успешно."}`);
        handleInspectServer(srv);
      } else {
        addToast("error", `Не удалось очистить диск: ${res?.error || res?.output || "Неизвестная ошибка"}`);
      }
    } catch (err: any) {
      addToast("error", `Ошибка выполнения очистки: ${err}`);
    } finally {
      setCleaningDiskServerId(null);
    }
  };



  // Choose Output Folder

  const handleBrowseFolder = async () => {

    try {

      const dir = await SelectDirectory();

      if (dir) setOutDir(dir);

    } catch (e) {

      console.error(e);

    }

  };



  // Pre-fill credentials from a saved server

  const handlePrefillFromServer = (srv: types.SavedServer) => {

    setSshHost(srv.host);

    setSshPort(srv.port || 22);

    setSshUser(srv.username || "root");

    if ((srv as any).password) setSshPass((srv as any).password);

    if (srv.label && srv.label.includes(".")) {

      setPanelDomain(srv.label);

      setNodeDecoyDomain(srv.label);

    }

    addToast("info", `Загружены реквизиты сервера: ${srv.label || srv.host}`);

  };



  // Load Profiles & Squads for Node Integration
  const loadPanelEntities = async (url: string, token: string) => {
    const cleanUrl = url.trim();
    const cleanToken = token.trim();
    if (!cleanUrl || !cleanToken) {
      addToast("warning", "Сначала укажите URL панели и API токен выше.");
      return;
    }
    setLoadingPanelEntities(true);
    try {
      const [profiles, squads] = await Promise.all([
        GetPanelConfigProfiles(cleanUrl, cleanToken),
        GetPanelSquads(cleanUrl, cleanToken),
      ]);
      if (profiles && Array.isArray(profiles)) {
        setPanelProfiles(profiles);
        if (profiles.length > 0) {
          setSelectedProfileInbound((prev) => prev || `${profiles[0].uuid}:${profiles[0].inbound_uuid}`);
        }
      }
      if (squads && Array.isArray(squads)) {
        setPanelSquads(squads);
        if (squads.length > 0) {
          setSelectedSquadUuid((prev) => {
            if (prev && prev !== "default" && prev !== "none") return prev;
            const def = squads.find(
              (s) =>
                s.name.toLowerCase().includes("default") ||
                s.name.toLowerCase().includes("дефолт")
            );
            return def ? def.uuid : squads[0].uuid;
          });
        }
      }
      if ((profiles && profiles.length > 0) || (squads && squads.length > 0)) {
        addToast("success", `Загружено: профилей (${profiles?.length || 0}), сквадов (${squads?.length || 0})`);
      } else {
        addToast("info", "В панели пока нет сохраненных профилей или сквадов.");
      }
    } catch (err: any) {
      console.warn("Не удалось загрузить профили/сквады панели:", err);
      addToast("error", `Ошибка загрузки данных панели: ${err?.message || err}`);
    } finally {
      setLoadingPanelEntities(false);
    }
  };

  // Test Panel API Connection
  const handleTestPanelAPI = async () => {
    if (!nodePanelUrl.trim() || !nodeApiToken.trim()) {
      addToast("error", "Укажите URL панели и API токен");
      return;
    }

    setPanelApiTesting(true);
    setPanelApiStatus(null);
    try {
      const res = await TestPanelAPI(nodePanelUrl.trim(), nodeApiToken.trim());
      setPanelApiStatus(res);
      if (res.success) {
        addToast("success", `Связь с панелью подтверждена! Найдено нод: ${res.nodes_count}`);
        localStorage.setItem("rw_panel_url", nodePanelUrl.trim());
        localStorage.setItem("rw_api_token", nodeApiToken.trim());
        // Auto-load config profiles and squads for node integration
        loadPanelEntities(nodePanelUrl.trim(), nodeApiToken.trim());
      } else {
        addToast("error", `Ошибка API панели: ${res.error}`);
      }

    } catch (err: any) {

      addToast("error", `Ошибка проверки API: ${err}`);

      setPanelApiStatus(

        new types.PanelInfoResult({

          success: false,

          error: err.toString(),

        })

      );

    } finally {

      setPanelApiTesting(false);

    }

  };



  const extractOriginUrl = (urlStr: string) => {

    if (!urlStr) return "";

    try {

      const trimmed = urlStr.trim();

      const withProto = trimmed.startsWith("http://") || trimmed.startsWith("https://") ? trimmed : `https://${trimmed}`;

      const u = new URL(withProto);

      return `${u.protocol}//${u.host}`;

    } catch {

      return urlStr.replace(/\/gate_[a-zA-Z0-9_-]+.*$/, "").replace(/\/admin_gate.*$/, "").trim();

    }

  };



  // Quick-action: Add Node directly to this Master Panel

  const handleAddNodeToPanel = (srv: types.SavedServer) => {

    setActiveTab("wizard");

    setStep(1);

    handleSelectPreset("node-only");

    setNodeLinkMode("auto");

    const targetUrl = srv.web_url || (srv.label && srv.label.includes(".") ? `https://${srv.label}` : "");

    if (targetUrl) {

      setNodePanelUrl(extractOriginUrl(targetUrl));

    }

    addToast("info", `Мастер настроен для привязки ноды к панели: ${srv.label || srv.host}`);

  };



  // Start Deployment
  const handleStart = async () => {
    if (components.length === 0) {
      addToast("error", "Выберите хотя бы один компонент для развертывания.");
      return;
    }

    if (deployMode === "remote") {
      if (!sshHost.trim()) {
        addToast("error", "Укажите IP-адрес или хост удаленного VPS сервера.");
        return;
      }
      if (!sshPass && !sshKey) {
        addToast("error", "Укажите пароль root или выберите приватный SSH-ключ для подключения к серверу.");
        return;
      }
    }

    let finalSecret = nodeSecret.trim();
    const cleanDecoyDomain = nodeDecoyDomain.trim().replace(/^https?:\/\//i, "").replace(/\/+$/, "");
    const cleanHostAddress = customHostAddress.trim().replace(/^https?:\/\//i, "").replace(/\/+$/, "");

    if (preset === "node-only") {
      if (deployDecoy && !cleanDecoyDomain) {
        addToast("error", "Укажите домен маскировочного сайта для SelfSteal Decoy или отключите опцию для режима Classic Reality.");
        return;
      }
      if (deployDecoy && decoyTemplate === "clone" && !decoyCloneUrl.trim()) {
        addToast("error", "Укажите URL сайта для клонирования маскировочного сайта.");
        return;
      }
      if (!nodePort || nodePort < 1 || nodePort > 65535) {
        addToast("error", "Порт связи с панелью должен быть в диапазоне от 1 до 65535 (по умолчанию 2222).");
        return;
      }
      if ([22, 80, 443].includes(nodePort)) {
        addToast("error", `Порт ${nodePort} зарезервирован для системных служб. Выберите другой порт связи.`);
        return;
      }
    }

    if (preset === "node-only" && nodeLinkMode === "auto") {
      if (!nodePanelUrl.trim() || !nodeApiToken.trim()) {
        addToast("error", "Для авто-привязки ноды укажите URL панели и API токен");
        return;
      }
      setLogs([
        { status: "step", message: "1. Подключение к API мастер-панели Remnawave..." },
        { status: "info", message: `Панель: ${nodePanelUrl.trim()}` },
      ]);
      setIsDeploying(true);
      setDeployFinished(false);

      try {
        let cfgProfUuid = "";
        let inbUuid = "";
        let inbPort = 443;
        if (nodeIntegrationMode === "custom") {
          if (!selectedProfileInbound) {
            addToast("error", "Выберите профиль конфигурации и входящий порт (Inbound) перед продолжением.");
            setIsDeploying(false);
            return;
          }
          const parts = selectedProfileInbound.split(":");
          cfgProfUuid = parts[0] || "";
          inbUuid = parts[1] || "";
          const foundProf = panelProfiles.find((p) => p.uuid === cfgProfUuid && p.inbound_uuid === inbUuid);
          if (foundProf && foundProf.port > 0) {
            inbPort = foundProf.port;
          }
        }

        const createRes = await CreateNodeInPanel(
          nodePanelUrl.trim(),
          nodeApiToken.trim(),
          new types.CreateNodeRequest({
            name: nodeName.trim() || `${nodeCountry} - Node`,
            country: nodeCountry,
            address: sshHost.trim(),
            port: nodePort,
            traffic_multiplier: 1.0,
            decoy_domain: deployDecoy ? cleanDecoyDomain : "",
            port_mode: portMode,
            torrent_blocker: torrentBlocker,
            block_spam_ports: blockSpamPorts,
            integration_mode: nodeIntegrationMode,
            config_profile_uuid: cfgProfUuid,
            inbound_uuid: inbUuid,
            inbound_port: inbPort,
            squad_uuid: selectedSquadUuid || "none",
            create_host: createSubscriptionHost,
            host_address: cleanHostAddress || (deployDecoy && cleanDecoyDomain ? cleanDecoyDomain : sshHost.trim()),
          })
        );
        if (!createRes.success) {
          throw new Error(`Ошибка регистрации в панели: ${createRes.error}`);
        }
        finalSecret = createRes.secret_key;
        setNodeSecret(finalSecret);

        const extraLogs: LogEntry[] = [];
        if (createRes.host_created) {
          extraLogs.push({ status: "success", message: "Создан хост подписки в панели Remnawave!" });
        }
        if (createRes.squad_updated) {
          extraLogs.push({ status: "success", message: "Входящий порт ноды успешно привязан к скваду пользователей!" });
        }

        setLogs((prev) => [
          ...prev,
          { status: "success", message: `Нода успешно зарегистрирована в панели Remnawave! UUID: ${createRes.node_uuid}` },
          ...extraLogs,
          { status: "info", message: `Получен секретный ключ (SECRET_KEY) от панели.` },
          { status: "step", message: "2. Передача конфигурации и запуск контейнеров на VPS по SSH..." },
        ]);

      } catch (err: any) {
        setIsDeploying(false);
        setLogs((prev) => [
          ...prev,
          { status: "error", message: `Сбой регистрации ноды в API: ${err.message || err}` },
        ]);
        addToast("error", `Сбой API панели: ${err.message || err}`);
        return;
      }
    }

    const cleanDomain = panelDomain.trim().replace(/^https?:\/\//i, "").replace(/\/+$/, "");
    const cleanSubDomain = subDomain.trim().replace(/^https?:\/\//i, "").replace(/\/+$/, "");

    if (preset !== "node-only" && !cleanDomain) {
      addToast("error", "Укажите домен панели управления (например: panel.domain.com) для выпуска SSL-сертификата и настройки веб-сервера.");
      return;
    }

    if (preset !== "node-only" && useSubDomain) {
      if (!cleanSubDomain) {
        addToast("error", "Укажите отдельный домен для страницы подписок.");
        return;
      }
      if (cleanSubDomain.toLowerCase() === cleanDomain.toLowerCase()) {
        addToast("error", "Домен страницы подписок должен отличаться от домена панели управления.");
        return;
      }
    }

    if (preset !== "node-only" && components.includes("panel")) {
      if (adminCreationMode === "custom") {
        if (!adminUsername.trim()) {
          addToast("error", "Укажите логин суперадминистратора.");
          return;
        }
        if (adminPassword.length < 24 || !/[A-Z]/.test(adminPassword) || !/[a-z]/.test(adminPassword) || !/[0-9]/.test(adminPassword)) {
          addToast("error", "Пароль суперадминистратора должен содержать минимум 24 символа, включая заглавную букву, строчную букву и цифру (политика безопасности Remnawave).");
          return;
        }
      }
    }

    if (useCloudflareDns && !cloudflareToken.trim()) {
      addToast("error", "Включен выпуск SSL через Cloudflare DNS, но не указан Cloudflare API токен.");
      return;
    }

    if (useGate) {
      const cleanGate = secretGate.trim().replace(/^\/+/, "");
      if (!cleanGate) {
        addToast("error", "Укажите путь секретного гейта (например: admin_gate).");
        return;
      }
      if (!/^[a-zA-Z0-9_\-]+$/.test(cleanGate)) {
        addToast("error", "Секретный гейт может содержать только латинские буквы, цифры, дефис и подчеркивание.");
        return;
      }
    }

    let webUrl = "";
    if (cleanDomain) {
      const gate = useGate && secretGate ? "/" + secretGate.trim().replace(/^\/+/, "") : "";
      webUrl = `https://${cleanDomain}${gate}`;
      setDeployedWebUrl(webUrl);
    }

    let calculatedPanelIp = customPanelIp.trim();
    if (!calculatedPanelIp && nodePanelUrl.trim()) {
      try {
        const u = new URL(nodePanelUrl.trim().startsWith("http") ? nodePanelUrl.trim() : `https://${nodePanelUrl.trim()}`);
        calculatedPanelIp = u.hostname;
      } catch {
        calculatedPanelIp = nodePanelUrl.trim().replace(/^https?:\/\//, "").split("/")[0].split(":")[0];
      }
    }

    const plan = new types.DeploymentPlan({
      components: components,
      domain: cleanDomain,
      panel_domain: cleanDomain,
      subscription_domain: cleanSubDomain,
      node_port: nodePort,
      node_secret: finalSecret,
      proxy: proxy,
      subscription_mode: "bundled",
      secret_gate: useGate ? secretGate.trim().replace(/^\/+/, "") : "",
      use_camouflage: useCamouflage,
      ufw_setup: useUfw,
      ssh_host: sshHost.trim(),
      ssh_port: sshPort,
      ssh_user: sshUser.trim(),
      ssh_password: sshPass,
      ssh_key: sshKey.trim(),
      output_dir: outDir.trim(),
      deployment_mode: deployMode,
      use_cloudflare_dns: useCloudflareDns,
      cloudflare_api_token: cloudflareToken.trim(),
      panel_ip: calculatedPanelIp,
      hide_node_port: hideNodePort,
      stealth_drop: stealthDrop,
      deploy_decoy: deployDecoy,
      decoy_domain: deployDecoy ? cleanDecoyDomain : "",
      port_mode: portMode,
      decoy_template: decoyTemplate,
      decoy_clone_url: decoyCloneUrl.trim(),
      torrent_blocker: torrentBlocker,
      block_spam_ports: blockSpamPorts,
      xray_core_type: xrayCoreType,
      enable_tinyauth: false,
      performance_optimization: optimizePerformance,
      admin_creation_mode: adminCreationMode,
      admin_username: adminCreationMode === "custom" ? adminUsername.trim() : "admin",
      admin_password: adminCreationMode === "custom" ? adminPassword.trim() : "",
    });



    if (deployMode === "generate") {

      try {

        const res = await GenerateConfig(plan);

        if (res.success) {

          setLogs([

            { status: "success", message: "Конфигурационные файлы успешно созданы!" },

            { status: "info", message: `Папка сохранения: ${res.out_dir}` },

            ...res.files.map((f: string) => ({ status: "stdout" as const, message: `  • ${f}` })),

            { status: "step", message: "Что делать дальше:" },

            { status: "info", message: "1. Скопируйте папку на ваш Linux VPS." },

            { status: "info", message: "2. Запустите в терминале: chmod +x deploy.sh && ./deploy.sh" },

          ]);

          setDeployFinished(true);

          setDeploySuccess(true);

          addToast("success", "Файлы конфигурации сгенерированы!");

        } else {

          setLogs([{ status: "error", message: `Ошибка генерации: ${res.error}` }]);

          addToast("error", "Ошибка генерации файлов");

        }

      } catch (err: any) {

        setLogs([{ status: "error", message: `Сбой: ${err}` }]);

      }

    } else {

      if (preset !== "node-only" || nodeLinkMode !== "auto") {
        setLogs([]);
      }

      setIsDeploying(true);

      setDeployFinished(false);

      try {

        await StartDeployment(plan);

        // Save this server to catalog

        await SaveServer(

          new types.SavedServer({
            id: `srv-${Date.now()}`,
            label: cleanDomain || sshHost,
            host: sshHost,
            port: sshPort,
            username: sshUser,
            password: sshPass,
            ssh_key: sshKey,
            web_url: preset === "node-only" ? "" : webUrl,
            role: preset === "node-only" ? "node" : "master",
            last_seen: new Date().toLocaleDateString() + " " + new Date().toLocaleTimeString(),
          })

        );

        loadServers();

      } catch (err: any) {

        setIsDeploying(false);

        setLogs([{ status: "error", message: `Не удалось начать установку: ${err}` }]);

        addToast("error", `Ошибка запуска: ${err}`);

      }

    }

  };



  // Inspect Server Containers & Metrics

  const handleInspectServer = async (srv: types.SavedServer, silentToast: boolean = false): Promise<boolean> => {

    setServerLoading((prev) => ({ ...prev, [srv.id]: true }));

    let isSuccess = false;

    try {
      if (!silentToast) {
        // Manual user trigger explicitly resets Circuit Breaker cooldown
        await ResetCircuitBreaker(srv.id);
      }
      const res = await GetServerStatus(srv, (srv as any).password || sshPass);

      setServerStatus((prev) => ({ ...prev, [srv.id]: res }));

      isSuccess = Boolean(res.success);

      if (!silentToast) {

        if (res.success) {

          addToast("info", `Статус ${srv.label || srv.host} обновлен (Онлайн)`);

        } else {

          addToast("error", `Сервер недоступен: ${res.error}`);

        }

      }

    } catch (e: any) {

      setServerStatus((prev) => ({

        ...prev,

        [srv.id]: new types.ServerStatusResult({

          success: false,

          error: e.toString(),

          containers: [],

        }),

      }));

      if (!silentToast) {

        addToast("error", `Сервер недоступен: ${e}`);

      }

    } finally {

      setServerLoading((prev) => ({ ...prev, [srv.id]: false }));

    }

    return isSuccess;

  };

  // Resilient sliding-window worker pool inspection of all saved servers (limit 4 concurrent SSH streams)
  const handleInspectAllServers = useCallback(async () => {
    if (savedServers.length === 0) return;

    const concurrency = 4;
    let onlineCount = 0;
    let offlineCount = 0;
    let currentIndex = 0;

    const numWorkers = Math.min(concurrency, savedServers.length);
    const workers = Array.from({ length: numWorkers }, async () => {
      while (currentIndex < savedServers.length) {
        const idx = currentIndex++;
        const srv = savedServers[idx];
        try {
          const ok = await handleInspectServer(srv, true);
          if (ok) onlineCount++;
          else offlineCount++;
        } catch {
          offlineCount++;
        }
      }
    });

    await Promise.all(workers);

    if (offlineCount === 0) {
      addToast("success", `Опрос завершен: все серверы (${onlineCount}) в сети`);
    } else if (onlineCount === 0) {
      addToast("error", `Опрос завершен: все серверы (${offlineCount}) недоступны`);
    } else {
      addToast("info", `Опрос завершен: ${onlineCount} в сети, ${offlineCount} недоступно`);
    }
  }, [savedServers, handleInspectServer, addToast]);



  // View Container Logs in Modal

  const handleOpenLogsModal = async (srv: types.SavedServer, container: string) => {

    setInspectModal({ server: srv, container, logs: "Загрузка логов...", autoRefresh: false });

    try {

      const out = await GetContainerLogs(srv, container);

      setInspectModal({

        server: srv,

        container,

        logs: out || "Логи пусты.",

        autoRefresh: false,

      });

    } catch (e: any) {

      setInspectModal({

        server: srv,

        container,

        logs: `Ошибка загрузки логов: ${e}`,

        autoRefresh: false,

      });

    }

  };



  // Auto-refresh logs in modal

  useEffect(() => {

    if (!inspectModal || !inspectModal.autoRefresh) return;

    const interval = setInterval(async () => {

      try {

        const out = await GetContainerLogs(inspectModal.server, inspectModal.container);

        setInspectModal((prev) => (prev ? { ...prev, logs: out || "Логи пусты." } : null));

      } catch (e) {

        console.error(e);

      }

    }, 3000);

    return () => clearInterval(interval);

  }, [inspectModal?.autoRefresh, inspectModal?.container]);
  // 1. Auto-Fix Subscription Token (Contextual Self-Healing)
  const handleAutoFixSub = async (srv: types.SavedServer) => {
    setActionLoading((prev) => ({ ...prev, [`fix-sub-${srv.id}`]: true }));
    addToast("info", "Авто-восстановление токена страницы подписок...");
    try {
      const res = await AutoFixSubscriptionToken(srv);
      if (res.success) {
        addToast("success", "Токен успешно обновлен! Страница подписок перезапущена.");
        await handleInspectServer(srv);
        if (inspectModal && inspectModal.server.id === srv.id) {
          const freshLogs = await GetContainerLogs(srv, "remnawave-subscription-page");
          setInspectModal((prev) => (prev ? { ...prev, logs: freshLogs || "Логи пусты." } : null));
        }
      } else {
        addToast("error", `Сбой восстановления токена: ${res.error}`);
      }
    } catch (e: any) {
      addToast("error", `Ошибка: ${e.message || e}`);
    } finally {
      setActionLoading((prev) => ({ ...prev, [`fix-sub-${srv.id}`]: false }));
    }
  };

  // 2. Reinstall Stack: Soft (Preserve DB) vs Clean (Wipe & Wizard)
  const handleReinstallSoft = async (srv: types.SavedServer) => {
    setReinstallModal(null);
    addToast("info", "Мягкая переустановка: обновление файлов и пересоздание контейнеров...");
    await handleUpdateStack(srv);
  };

  const handleReinstallClean = async (srv: types.SavedServer) => {
    setReinstallModal(null);
    addToast("info", `Чистая переустановка: очистка стека на сервере ${srv.host}...`);
    try {
      const res = await UninstallStack(srv, true);
      if (res.success) {
        addToast("success", "Стек очищен! Заполняем реквизиты в мастере установки...");
        handlePrefillFromServer(srv);
        setStep(1);
        setActiveTab("wizard");
      } else {
        addToast("error", `Ошибка очистки стека: ${res.error}`);
      }
    } catch (e: any) {
      addToast("error", `Ошибка: ${e.message || e}`);
    }
  };

  // 3. Uninstall & Wipe Remnawave from VPS (Clean Server without reinstalling OS)
  const handleUninstallServer = async () => {
    if (!uninstallModal) return;
    const srv = uninstallModal.server;
    setUninstallModal((prev) => (prev ? { ...prev, loading: true } : null));
    addToast("info", `Удаление Remnawave и очистка VPS ${srv.host}...`);
    try {
      const res = await UninstallStack(srv, true);
      if (res.success) {
        addToast("success", "Стек Remnawave успешно удален с сервера. VPS очищен без сброса ОС.");
        await handleDeleteServer(srv.id);
        setUninstallModal(null);
      } else {
        addToast("error", `Сбой удаления стека: ${res.error}`);
        setUninstallModal((prev) => (prev ? { ...prev, loading: false } : null));
      }
    } catch (e: any) {
      addToast("error", `Ошибка: ${e.message || e}`);
      setUninstallModal((prev) => (prev ? { ...prev, loading: false } : null));
    }
  };




  // Restart single container

  const handleRestartContainer = async (srv: types.SavedServer, container: string) => {

    const key = `restart-${srv.id}-${container}`;

    setActionLoading((prev) => ({ ...prev, [key]: true }));

    try {

      const res = await RestartContainer(srv, container);

      if (res.success) {

        addToast("success", `Контейнер ${container} успешно перезапущен!`);

        handleInspectServer(srv);

      } else {

        addToast("error", `Ошибка: ${res.error}`);

      }

    } catch (e: any) {

      addToast("error", `Сбой: ${e}`);

    } finally {

      setActionLoading((prev) => ({ ...prev, [key]: false }));

    }

  };



  // Restart all stack

  const handleRestartAll = async (srv: types.SavedServer) => {

    const key = `restart-all-${srv.id}`;

    setActionLoading((prev) => ({ ...prev, [key]: true }));

    try {

      const res = await RestartAllStack(srv);

      if (res.success) {

        addToast("success", "Все службы Remnawave успешно перезапущены!");

        handleInspectServer(srv);

      } else {

        addToast("error", `Ошибка: ${res.error}`);

      }

    } catch (e: any) {

      addToast("error", `Сбой: ${e}`);

    } finally {

      setActionLoading((prev) => ({ ...prev, [key]: false }));

    }

  };



  // One-click Fix Database Sync

  const handleFixDBSync = async (srv: types.SavedServer) => {

    const key = `fix-db-${srv.id}`;

    setActionLoading((prev) => ({ ...prev, [key]: true }));

    try {

      const res = await FixDatabaseSync(srv);

      if (res.success) {

        addToast("success", "Пароль базы данных PostgreSQL синхронизирован!");

        handleInspectServer(srv);

      } else {

        addToast("error", `Ошибка: ${res.error}`);

      }

    } catch (e: any) {

      addToast("error", `Сбой: ${e}`);

    } finally {

      setActionLoading((prev) => ({ ...prev, [key]: false }));

    }

  };



  // Attach Node Secret Key & Start Node

  const handleAttachNode = async () => {

    if (!nodeModal || !nodeModal.secretKey.trim()) {

      addToast("error", "Введите SECRET_KEY для ноды");

      return;

    }

    setNodeModal((prev) => (prev ? { ...prev, loading: true } : null));

    try {

      const res = await AttachNodeSecret(nodeModal.server, nodeModal.secretKey);

      if (res.success) {

        addToast("success", "Нода успешно настроена и запущена!");

        const currentServer = nodeModal.server;

        setNodeModal(null);

        handleInspectServer(currentServer);

      } else {

        addToast("error", `Ошибка: ${res.error}`);

        setNodeModal((prev) => (prev ? { ...prev, loading: false } : null));

      }

    } catch (e: any) {

      addToast("error", `Сбой: ${e}`);

      setNodeModal((prev) => (prev ? { ...prev, loading: false } : null));

    }

  };



  // Open Anti-Probing Security Audit Modal
  const handleOpenAudit = async (srv: types.SavedServer, customIp?: string) => {
    let panelIp = customIp || "";
    if (!panelIp && srv.parent_master_id) {
      const master = savedServers.find((s) => s.id === srv.parent_master_id);
      if (master && master.host) {
        panelIp = master.host;
      }
    }

    setAuditModal({ server: srv, loading: true, applying: false, audit: null });
    try {
      const res = await AuditNodeSecurity(srv, panelIp);
      setAuditModal({ server: srv, loading: false, applying: false, audit: res });
      addToast("info", `Аудит безопасности ${srv.label || srv.host}: ${res.overall_score}%`);
    } catch (err: any) {
      setAuditModal((prev) => ({
        ...prev,
        loading: false,
        audit: new types.NodeSecurityAudit({
          node_port_protected: false,
          panel_whitelisted: false,
          docker_user_active: false,
          stealth_drop_active: false,
          decoy_active: false,
          bbr_active: false,
          container_running: false,
          port_443_listening: false,
          xray_core_type: "",
          overall_score: 0,
          summary: `Ошибка выполнения проверки: ${err.message || err}`,
          checks: [],
        }),
      }));
      addToast("error", `Ошибка аудита: ${err}`);
    }
  };

  // Apply Anti-Probing Maximum Protection (remna-mask)
  const handleApplyProtection = async (
    panelIp: string,
    hidePort: boolean,
    stealth: boolean,
    enableBBR: boolean
  ) => {
    if (!auditModal.server) return;
    const srv = auditModal.server;
    setAuditModal((prev) => ({ ...prev, applying: true }));

    try {
      if (hidePort || stealth || enableBBR) {
        const fwRes = await ApplyAntiProbingFirewall(
          srv,
          panelIp,
          [80, 443],
          hidePort,
          stealth,
          enableBBR
        );
        if (!fwRes.success) {
          throw new Error(fwRes.error || "Ошибка настройки файрвола");
        }
      }

      addToast("success", "Правила защиты и сетевой экран успешно применены!");
      const auditRes = await AuditNodeSecurity(srv, panelIp);
      setAuditModal({ server: srv, loading: false, applying: false, audit: auditRes });
    } catch (err: any) {
      setAuditModal((prev) => ({ ...prev, applying: false }));
      addToast("error", `Сбой применения защиты: ${err.message || err}`);
    }
  };



  // Optimize Network with Google BBR

  const handleOptimizeBBR = async (srv: types.SavedServer) => {

    const key = `bbr-${srv.id}`;

    setActionLoading((prev) => ({ ...prev, [key]: true }));

    try {

      const res = await OptimizeNetworkBBR(srv);

      if (res.success) {

        addToast("success", "Ускорение сети Google BBR успешно активировано!");

        handleInspectServer(srv);

      } else {

        addToast("error", `Ошибка: ${res.error}`);

      }

    } catch (e: any) {

      addToast("error", `Сбой: ${e}`);

    } finally {

      setActionLoading((prev) => ({ ...prev, [key]: false }));

    }

  };



  // Configure Docker Mirrors

  const handleConfigureMirrors = async (srv: types.SavedServer) => {

    const key = `mirrors-${srv.id}`;

    setActionLoading((prev) => ({ ...prev, [key]: true }));

    try {

      const res = await ConfigureDockerMirrors(srv);

      if (res.success) {

        addToast("success", "Сеть и зеркала Docker успешно настроены!");

        handleInspectServer(srv);

      } else {

        addToast("error", `Ошибка: ${res.error}`);

      }

    } catch (e: any) {

      addToast("error", `Сбой: ${e}`);

    } finally {

      setActionLoading((prev) => ({ ...prev, [key]: false }));

    }

  };



  // Create Backup

  const handleCreateBackup = async (srv: types.SavedServer) => {

    const key = `backup-${srv.id}`;

    setActionLoading((prev) => ({ ...prev, [key]: true }));

    try {

      const res = await CreateBackup(srv);

      if (res.success) {

        addToast("success", "Резервная копия успешно создана!");

        handleInspectServer(srv);

      } else {

        addToast("error", `Ошибка: ${res.error}`);

      }

    } catch (e: any) {

      addToast("error", `Сбой: ${e}`);

    } finally {

      setActionLoading((prev) => ({ ...prev, [key]: false }));

    }

  };



  // Prune Docker System Cache

  const handlePruneDocker = async (srv: types.SavedServer) => {

    const key = `prune-${srv.id}`;

    setActionLoading((prev) => ({ ...prev, [key]: true }));

    try {

      const res = await PruneDockerSystem(srv);

      if (res.success) {

        addToast("success", "Неиспользуемые образы и кэш Docker очищены!");

        handleInspectServer(srv);

      } else {

        addToast("error", `Ошибка: ${res.error}`);

      }

    } catch (e: any) {

      addToast("error", `Сбой: ${e}`);

    } finally {

      setActionLoading((prev) => ({ ...prev, [key]: false }));

    }

  };



  // Reset Superadmin

  const handleResetSuperadmin = async (srv: types.SavedServer) => {

    if (!window.confirm("Вы уверены, что хотите сбросить супер-администратора? После сброса панель предложит пройти первичную регистрацию нового админа.")) {

      return;

    }

    const key = `reset-admin-${srv.id}`;

    setActionLoading((prev) => ({ ...prev, [key]: true }));

    try {

      const res = await ResetSuperadmin(srv);

      if (res.success) {

        addToast("success", "Супер-администратор Remnawave успешно сброшен!");

        handleInspectServer(srv);

      } else {

        addToast("error", `Ошибка: ${res.error}`);

      }

    } catch (e: any) {

      addToast("error", `Сбой: ${e}`);

    } finally {

      setActionLoading((prev) => ({ ...prev, [key]: false }));

    }

  };



  // Renew SSL Certificates

  const handleRenewSSL = async (srv: types.SavedServer) => {

    const key = `ssl-${srv.id}`;

    setActionLoading((prev) => ({ ...prev, [key]: true }));

    try {

      const res = await RenewSSL(srv);

      if (res.success) {

        addToast("success", "SSL-сертификаты Let's Encrypt успешно обновлены!");

        handleInspectServer(srv);

      } else {

        addToast("error", `Ошибка: ${res.error}`);

      }

    } catch (e: any) {

      addToast("error", `Сбой: ${e}`);

    } finally {

      setActionLoading((prev) => ({ ...prev, [key]: false }));

    }

  };



  // Toggle Torrent Blocker & Egress Filter (Anti-Abuse)

  const handleToggleAntiAbuse = async (srv: types.SavedServer, currentActive: boolean) => {

    const key = `anti-abuse-${srv.id}`;

    setActionLoading((prev) => ({ ...prev, [key]: true }));

    try {

      const res = await ToggleAntiAbuse(srv, !currentActive);

      if (res.success) {

        addToast("success", "Статус защиты от торрентов и сканирования обновлен!");

        handleInspectServer(srv);

      } else {

        addToast("error", `Ошибка: ${res.error}`);

      }

    } catch (e: any) {

      addToast("error", `Сбой: ${e}`);

    } finally {
      setActionLoading((prev) => ({ ...prev, [key]: false }));
    }
  };

  // Toggle Performance Optimization (Unix Sockets & 512MB RAM)
  const handleTogglePerformance = async (srv: types.SavedServer, currentOptimized: boolean) => {
    const key = `perf-toggle-${srv.id}`;
    setActionLoading((prev) => ({ ...prev, [key]: true }));
    try {
      const res = await ToggleStackPerformance(srv, !currentOptimized);
      if (res.success) {
        addToast("success", res.output || "Режим производительности успешно изменен!");
        handleInspectServer(srv);
      } else {
        addToast("error", `Ошибка: ${res.error}`);
      }
    } catch (e: any) {
      addToast("error", `Сбой: ${e}`);
    } finally {
      setActionLoading((prev) => ({ ...prev, [key]: false }));
    }
  };

  // Cloudflare WARP Outbound Setup

  const handleInstallWARP = async (srv: types.SavedServer) => {

    const key = `warp-${srv.id}`;

    setActionLoading((prev) => ({ ...prev, [key]: true }));

    try {

      const res = await InstallCloudflareWARP(srv);

      if (res.success) {

        addToast("success", "Cloudflare WARP успешно настроен на сервере!");

        handleInspectServer(srv);

      } else {

        addToast("error", `Ошибка WARP: ${res.error}`);

      }

    } catch (e: any) {

      addToast("error", `Сбой: ${e}`);

    } finally {

      setActionLoading((prev) => ({ ...prev, [key]: false }));

    }

  };



  // Reality SelfSteal on port 443 with Fallback 8443

  const handleConfigureSelfSteal = async (srv: types.SavedServer) => {

    const key = `selfsteal-${srv.id}`;

    setActionLoading((prev) => ({ ...prev, [key]: true }));

    try {

      const res = await ConfigureRealitySelfSteal(srv, "127.0.0.1:8443");

      if (res.success) {

        addToast("success", "SelfSteal Decoy-сайт успешно развёрнут!");

        handleInspectServer(srv);

      } else {

        addToast("error", `Ошибка SelfSteal: ${res.error}`);

      }

    } catch (e: any) {

      addToast("error", `Сбой: ${e}`);

    } finally {

      setActionLoading((prev) => ({ ...prev, [key]: false }));

    }

  };



  // Download Backup to Local PC

  const handleDownloadBackup = async (srv: types.SavedServer, fileName: string) => {

    const key = `dl-${srv.id}-${fileName}`;

    setActionLoading((prev) => ({ ...prev, [key]: true }));

    try {

      const res = await DownloadBackup(srv, fileName);

      if (res.success) {

        addToast("success", "Резервная копия успешно сохранена на компьютер!");

      } else {

        if (!res.error?.includes("отменено")) {

          addToast("error", `Ошибка загрузки: ${res.error}`);

        }

      }

    } catch (e: any) {

      addToast("error", `Сбой загрузки: ${e}`);

    } finally {

      setActionLoading((prev) => ({ ...prev, [key]: false }));

    }

  };



  // Restore Backup
  const handleRestoreBackup = async (
    srv: types.SavedServer,
    fileName: string,
    scope: string = "all",
    createSafetyBackup: boolean = true
  ) => {
    const key = `restore-${srv.id}`;
    setActionLoading((prev) => ({ ...prev, [key]: true }));
    try {
      const res = await RestoreBackup(srv, fileName, scope, createSafetyBackup);
      if (res && res.success) {
        if (res.safety_backup_file) {
          addToast("success", `Сервер успешно восстановлен! Создан страховочный снимок: ${res.safety_backup_file}`);
        } else {
          addToast("success", "Сервер успешно восстановлен из резервной копии!");
        }
        setRestoreModal(null);
        handleInspectServer(srv);
      } else {
        addToast("error", `Ошибка восстановления: ${res?.error || "Неизвестная ошибка"}`);
      }
    } catch (e: any) {
      addToast("error", `Сбой восстановления: ${e}`);
    } finally {
      setActionLoading((prev) => ({ ...prev, [key]: false }));
    }
  };



  // Update Stack Images

  const handleUpdateStack = async (srv: types.SavedServer) => {

    const key = `update-stack-${srv.id}`;

    setActionLoading((prev) => ({ ...prev, [key]: true }));

    try {

      const res = await UpdateStackImages(srv);

      if (res.success) {

        addToast("success", "Службы Remnawave успешно обновлены до новейших версий!");

        handleInspectServer(srv);

      } else {

        addToast("error", `Ошибка обновления: ${res.error}`);

      }

    } catch (e: any) {

      addToast("error", `Сбой обновления: ${e}`);

    } finally {

      setActionLoading((prev) => ({ ...prev, [key]: false }));

    }

  };



  // Start Edit Server

  const handleStartEditServer = (srv: types.SavedServer) => {
    const currentRole = (srv.role === "node" || srv.role === "master")
      ? (srv.role as "master" | "node")
      : (srv.web_url || (srv.label && !isIpAddress(srv.label) && srv.label.toLowerCase().includes("panel")))
      ? "master"
      : "node";

    setEditServerModal({
      server: srv,
      name: srv.label || "",
      host: srv.host || "",
      port: srv.port || 22,
      user: srv.username || "root",
      authMethod: srv.ssh_key ? "key" : "password",
      password: srv.password || "",
      privateKey: srv.ssh_key || "",
      role: currentRole,
      parentMasterId: (srv as any).parent_master_id || "",
      locationCode: (srv as any).location_code || "",
      decoyDomain: (srv as any).decoy_domain || "",
      isExternalMaster: Boolean((srv as any).is_external_master),
      webUrl: srv.web_url || "",
    });
  };



  // Save Edited Server

  const handleSaveEditedServer = async () => {

    if (!editServerModal) return;

    const { server, name, host, port, user, authMethod, password, privateKey } = editServerModal;

    if (!host || !host.trim()) {

      addToast("error", "Укажите IP-адрес или хост сервера");

      return;

    }

    if (authMethod === "password" && !password?.trim()) {

      addToast("error", "Укажите пароль root для подключения по SSH");

      return;

    }

    if (authMethod === "key" && !privateKey?.trim()) {

      addToast("error", "Укажите путь к приватному SSH-ключу");

      return;

    }

    let calculatedWebUrl = "";
    if (editServerModal.role === "master") {
      if (editServerModal.webUrl?.trim()) {
        calculatedWebUrl = editServerModal.webUrl.trim();
        if (!calculatedWebUrl.startsWith("http://") && !calculatedWebUrl.startsWith("https://")) {
          calculatedWebUrl = `https://${calculatedWebUrl}`;
        }
      } else if (server.web_url) {
        calculatedWebUrl = server.web_url;
      } else if (name && name.includes(".")) {
        calculatedWebUrl = name.startsWith("http") ? name : `https://${name}`;
      } else if (host && !host.includes(":")) {
        calculatedWebUrl = `https://${host}`;
      }
    }

    const updated = new types.SavedServer({
      id: server.id,
      label: name || host.trim(),
      host: host.trim(),
      port: Number(port) || 22,
      username: user || "root",
      password: authMethod === "password" ? password : "",
      ssh_key: authMethod === "key" ? privateKey : "",
      web_url: calculatedWebUrl,
      role: editServerModal.role || (calculatedWebUrl ? "master" : "node"),
      parent_master_id: editServerModal.role === "node" ? (editServerModal.parentMasterId || "") : "",
      location_code: editServerModal.role === "node" ? (editServerModal.locationCode || "") : "",
      decoy_domain: editServerModal.role === "node" ? (editServerModal.decoyDomain || "") : "",
      is_external_master: editServerModal.role === "master" ? Boolean(editServerModal.isExternalMaster) : false,
      last_seen: server.last_seen || "Только что добавлен",
      admin_password: server.admin_password || "",
      sub_url: server.sub_url || "",
      port_mode: server.port_mode || "",
      xray_core_type: server.xray_core_type || "",
      api_token: server.api_token || "",
      node_secret: server.node_secret || "",
    });



    try {

      const ok = await SaveServer(updated);

      if (ok) {

        addToast("success", "Параметры сервера успешно сохранены!");

        setEditServerModal(null);

        const list = await GetServers();

        setSavedServers(list || []);

        handleInspectServer(updated);

      } else {

        addToast("error", "Не удалось сохранить сервер");

      }

    } catch (e: any) {

      addToast("error", `Ошибка: ${e}`);

    }

  };



  // Open web panel in browser

  const handleOpenBrowser = (url: string) => {

    if (!url) return;

    OpenBrowserURL(url);

    addToast("info", "Открываем панель управления в браузере...");

  };



  // Delete Server

  const handleDeleteServer = async (id: string) => {

    await DeleteServer(id);

    loadServers();

    addToast("info", "Сервер удален из каталога");

  };



  // Copy text to Clipboard (Stage 5: Micro-Interactions without toast spam)
  const [copiedKey, setCopiedKey] = useState<string | null>(null);

  const handleCopy = (text: string, label: string, key?: string) => {
    navigator.clipboard.writeText(text);
    if (key) {
      setCopiedKey(key);
      setTimeout(() => setCopiedKey((curr) => (curr === key ? null : curr)), 1500);
    } else {
      addToast("success", `${label} скопировано в буфер!`);
    }
  };



  // Copy Deploy Logs

  const handleCopyLogs = () => {

    const text = logs.map((l) => l.message).join("\n");

    handleCopy(text, "Журнал установки");

    setCopiedLogs(true);

    setTimeout(() => setCopiedLogs(false), 2000);

  };



  return (

    <div className="app-container">

      {/* Toast Notification Container */}

      <div className="win-toast-container">

        {toasts.map((t) => (

          <div key={t.id} className={`win-toast ${t.type} ${t.isExiting ? "win-toast-exit" : ""}`}>

            {t.type === "success" && <CheckCircle2 size={16} color="var(--win-status-success)" />}

            {t.type === "error" && <AlertTriangle size={16} color="var(--win-status-danger)" />}

            {t.type === "info" && <Info size={16} color="var(--win-accent)" />}

            <span className="win-toast-message">{t.message}</span>

          </div>

        ))}

      </div>



      {/* ====================================================================

          WinUI 3 Navigation View (Sidebar)

          ==================================================================== */}

      <aside className={`win-sidebar ${sidebarCollapsed ? "collapsed" : ""}`}>

        <div className="brand-header">

          <button

            type="button"

            className="sidebar-toggle-btn"

            title={sidebarCollapsed ? "Развернуть панель (Ctrl+B)" : "Свернуть панель (Ctrl+B)"}

            onClick={toggleSidebar}

          >

            <Menu size={16} />

          </button>

          {!sidebarCollapsed && (

            <>

              <div className="brand-icon">

                <Shield size={16} />

              </div>

              <div className="brand-title-wrap">

                <div className="brand-title">Remnawave</div>

                <div className="brand-subtitle">Manager v3.0 Fluent</div>

              </div>

            </>

          )}

        </div>



        <div className="nav-section-title">Разделы</div>

        <nav className="nav-list">

          <div

            className={`nav-item ${activeTab === "wizard" ? "active" : ""}`}

            title="Мастер установки"

            onClick={() => setActiveTab("wizard")}

          >

            <span className="nav-item-icon"><Layers size={16} /></span>

            <span>Мастер установки</span>

          </div>



          <div

            className={`nav-item ${activeTab === "servers" ? "active" : ""}`}

            title={`Мои серверы ${savedServers.length > 0 ? `(${savedServers.length})` : ""}`}

            onClick={() => setActiveTab("servers")}

          >

            <span className="nav-item-icon"><Server size={16} /></span>

            <span>Мои серверы</span>

            {savedServers.length > 0 && <span className="nav-badge">{savedServers.length}</span>}

          </div>



          <div

            className={`nav-item ${activeTab === "about" ? "active" : ""}`}

            title="О программе"

            onClick={() => setActiveTab("about")}

          >

            <span className="nav-item-icon"><Info size={16} /></span>

            <span>О программе</span>

          </div>

        </nav>



        <div className="sidebar-footer">

          <div
            className="nav-item"
            title="Безопасность локальных секретов"
            onClick={() => setVaultSettingsOpen(true)}
          >
            <span className="nav-item-icon">
              {vaultStatus?.enabled ? (
                vaultStatus.unlocked ? (
                  <KeyRound size={15} style={{ color: "var(--win-accent-primary)" }} />
                ) : (
                  <Lock size={15} style={{ color: "#ef4444" }} />
                )
              ) : (
                <ShieldCheck size={15} style={{ color: "#10b981" }} />
              )}
            </span>
            <span>Безопасность</span>
          </div>

          <div
            className="nav-item"
            title="Официальная документация Remnawave"
            onClick={() => OpenBrowserURL("https://docs.rw/overview/quick-start/")}
          >
            <span className="nav-item-icon"><ExternalLink size={15} /></span>
            <span>Документация Remna</span>
          </div>

        </div>

      </aside>



      {/* ====================================================================

          Main Content Canvas

          ==================================================================== */}

      <main className="win-main-content">

        {/* WinUI 3 CommandBar / Top Bar */}

        <header className="win-commandbar">

          <div className="win-commandbar-left">

            <span className="win-breadcrumb-root">Remnawave</span>

            <ChevronRight size={13} className="win-breadcrumb-sep" />

            <span className="win-breadcrumb-current">

              {activeTab === "wizard" && "Мастер развертывания"}

              {activeTab === "servers" && "Мои серверы"}

              {activeTab === "about" && "О программе"}

            </span>

          </div>



          <div className="win-commandbar-right">

            {/* Vault Lock / Security Status */}
            {vaultStatus?.enabled ? (
              vaultStatus?.unlocked ? (
                <>
                  <button
                    type="button"
                    className="win-commandbar-btn"
                    title="Заблокировать хранилище паролей (Ctrl+L)"
                    onClick={handleLockVaultAction}
                  >
                    <Lock size={14} style={{ color: "var(--win-accent-primary)" }} />
                    <span>Заблокировать</span>
                  </button>
                  <button
                    type="button"
                    className="win-commandbar-btn"
                    title="Настройки мастер-пароля и безопасности"
                    onClick={() => setVaultSettingsOpen(true)}
                  >
                    <KeyRound size={14} />
                  </button>
                </>
              ) : (
                <button
                  type="button"
                  className="win-commandbar-btn"
                  title="Хранилище заблокировано. Нажмите для ввода мастер-пароля"
                  style={{ background: "rgba(239, 68, 68, 0.12)", border: "1px solid rgba(239, 68, 68, 0.25)" }}
                  onClick={() => setVaultUnlockOpen(true)}
                >
                  <Lock size={14} style={{ color: "#ef4444" }} />
                  <span style={{ color: "#ef4444", fontWeight: 600 }}>Заблокировано</span>
                </button>
              )
            ) : (
              <button
                type="button"
                className="win-commandbar-btn"
                title="Безопасность: данные защищены Windows DPAPI. Нажмите для настройки мастер-пароля"
                onClick={() => setVaultSettingsOpen(true)}
              >
                <ShieldCheck size={14} style={{ color: "#10b981" }} />
                <span>Безопасность</span>
              </button>
            )}

            <div className="win-commandbar-divider" />

            <button

              type="button"

              className="win-commandbar-btn"

              title={theme === "dark" ? "Переключить на светлую тему" : "Переключить на тёмную тему"}

              onClick={toggleTheme}

            >

              {theme === "dark" ? <Sun size={15} style={{ color: "#fbbf24" }} /> : <Moon size={15} style={{ color: "#818cf8" }} />}

              <span>{theme === "dark" ? "Светлая тема" : "Тёмная тема"}</span>

            </button>

            <div className="win-commandbar-divider" />

            <button
              type="button"
              className="win-commandbar-btn"
              title="Официальная документация Remnawave"
              onClick={() => OpenBrowserURL("https://docs.rw/overview/quick-start/")}
            >
              <ExternalLink size={13} />
              <span>Документация</span>
            </button>

          </div>

        </header>



        <div className="win-content-scroll">

          <div className={`content-container ${activeTab === "servers" ? "servers-wide" : ""}`}>

            {/* ================================================================

                TAB: WIZARD

                ================================================================ */}

            {activeTab === "wizard" && (

              <div key="tab-wizard" className="tab-enter">

                {/* Page Title Header */}

                <div className="page-header">

                  <h1 className="page-title">Мастер развертывания</h1>

                  <p className="page-desc">

                    Пошаговая настройка стека Remnawave с автоматической проверкой доменов и развертыванием на Linux VPS.

                  </p>

                </div>

                                {/* Windows 11 Stepper Bar */}

                <div className="stepper-bar">
                  <div
                    className={`step-tab ${step === 1 ? "active" : ""} ${step > 1 ? "completed" : ""}`}
                    onClick={() => changeStep(1)}
                  >
                    <span className="step-number">{step > 1 ? <Check size={12} /> : "1"}</span>
                    <span>Роль сервера</span>
                  </div>

                  <div
                    className={`step-tab ${step === 2 ? "active" : ""} ${step > 2 ? "completed" : ""}`}
                    onClick={() => changeStep(2)}
                  >
                    <span className="step-number">{step > 2 ? <Check size={12} /> : "2"}</span>
                    <span>{preset === "node-only" ? "Параметры ноды" : "Домен и защита"}</span>
                  </div>

                  <div
                    className={`step-tab ${step === 3 ? "active" : ""}`}
                    onClick={() => changeStep(3)}
                  >
                    <span className="step-number">3</span>
                    <span>Развертывание</span>
                  </div>
                </div>

                {/* STEP 1: COMPONENT SELECTION & PRESETS */}
                {step === 1 && (
                  <div key="step-1" className={`step-enter ${stepDirection === "forward" ? "step-forward" : "step-backward"}`}>

                    {/* Presets Grid */}

                    {/* Server Role Selection */}

                    <div className="settings-group">

                      <div className="settings-group-title">Роль устанавливаемого сервера</div>

                      <div className="presets-grid">

                        {/* Preset 1: All in one */}

                        <div

                          className={`preset-tile ${preset === "all-in-one" ? "selected" : ""}`}

                          onClick={() => handleSelectPreset("all-in-one")}

                        >

                          <div className="preset-tile-header">
                            <div className="preset-tile-icon">
                              <Layers size={18} />
                            </div>

                            <div className="preset-radio-indicator">

                              {preset === "all-in-one" && <div className="preset-radio-dot" />}

                            </div>

                          </div>

                          <div className="preset-tile-title">Все в одном (Панель + Нода)</div>

                          <div className="preset-tile-desc">

                            Полная инфраструктура на одном VPS: веб-панель администратора, страница подписок, база данных, HTTPS-прокси и Xray Reality нода.

                          </div>

                          <div className="preset-badges">

                            <span className="preset-badge">Panel :3000</span>

                            <span className="preset-badge">Sub :3010</span>

                            <span className="preset-badge">Postgres 17</span>

                            <span className="preset-badge">Valkey</span>

                            <span className="preset-badge">HTTPS Прокси</span>

                            <span className="preset-badge">Xray Нода</span>

                          </div>

                        </div>



                        {/* Preset 2: Panel only */}

                        <div

                          className={`preset-tile ${preset === "panel-only" ? "selected" : ""}`}

                          onClick={() => handleSelectPreset("panel-only")}

                        >

                          <div className="preset-tile-header">
                            <div className="preset-tile-icon">
                              <LayoutDashboard size={18} />
                            </div>

                            <div className="preset-radio-indicator">

                              {preset === "panel-only" && <div className="preset-radio-dot" />}

                            </div>

                          </div>

                          <div className="preset-tile-title">Панель управления (Master)</div>

                          <div className="preset-tile-desc">

                            Центральный сервер управления пользователями, тарифами и подписками. Выделенные VPN-ноды в разных странах подключаются удалённо по API.

                          </div>

                          <div className="preset-badges">

                            <span className="preset-badge">Panel :3000</span>

                            <span className="preset-badge">Sub :3010</span>

                            <span className="preset-badge">Postgres 17</span>

                            <span className="preset-badge">Valkey</span>

                            <span className="preset-badge">HTTPS Прокси</span>

                          </div>

                        </div>



                        {/* Preset 3: Node only */}

                        <div

                          className={`preset-tile ${preset === "node-only" ? "selected" : ""}`}

                          onClick={() => handleSelectPreset("node-only")}

                        >

                          <div className="preset-tile-header">

                            <div className="preset-tile-icon">

                              <Server size={18} />

                            </div>

                            <div className="preset-radio-indicator">

                              {preset === "node-only" && <div className="preset-radio-dot" />}

                            </div>

                          </div>

                          <div className="preset-tile-title">Xray Reality Нода (Edge)</div>

                          <div className="preset-tile-desc">

                            Выделенный сервер для туннелирования трафика (Xray-core, Reality/VLESS). Управляется из вашей существующей панели Remnawave.

                          </div>

                          <div className="preset-badges">

                            <span className="preset-badge">Xray Core</span>

                            <span className="preset-badge">VLESS Reality</span>

                            <span className="preset-badge">Связь по API</span>

                          </div>

                        </div>

                      </div>

                    </div>



                    {/* Fine-tuning Settings Cards */}

                    <div className="settings-group">

                      <div className="settings-group-title">Параметры выбранной роли</div>

                      <div className="settings-card-group">

                        {components.includes("panel") && (
                          <SettingsCard
                            icon={<Shuffle size={18} />}
                            title="Обратный прокси-сервер"
                            description="Обеспечивает SSL-шифрование, защиту Security Gate и маскировку веб-трафика панели"
                            action={
                              <select
                                className="win-select"
                                value={proxy}
                                onChange={(e) => setProxy(e.target.value)}
                              >
                                <option value="nginx">Nginx (Рекомендуется — eGames Gate + Let's Encrypt)</option>
                                <option value="caddy">Caddy (Авто-выпуск SSL)</option>
                              </select>
                            }
                          />
                        )}

                        {components.includes("node") && (
                          <SettingsCard
                            icon={<Hash size={18} />}
                            title="Порт взаимодействия Ноды"

                            description="Внутренний порт связи панели с нодой (по умолчанию 2222)"

                            action={

                              <input

                                type="number"

                                className="win-textbox"

                                style={{ width: "100px", minWidth: "100px" }}

                                value={nodePort}

                                onChange={(e) => setNodePort(parseInt(e.target.value) || 2222)}

                              />

                            }

                          />

                        )}

                      </div>

                    </div>

                  </div>

                )}



                {/* STEP 2: DOMAIN & SECURITY (OR NODE SETTINGS) */}
                {step === 2 && (
                  <div key="step-2" className={`step-enter ${stepDirection === "forward" ? "step-forward" : "step-backward"}`}>

                    {preset === "node-only" ? (

                      <div className="settings-group">

                        <div className="settings-group-title">Параметры ноды (Xray Reality Edge)</div>



                        {/* Architecture Hero Card */}

                        <div

                          style={{

                            display: "flex",

                            gap: "14px",

                            alignItems: "flex-start",

                            padding: "16px 18px",

                            borderRadius: "8px",

                            background: "linear-gradient(135deg, rgba(96, 205, 255, 0.08) 0%, rgba(255, 255, 255, 0.02) 100%)",

                            border: "1px solid rgba(96, 205, 255, 0.25)",

                            borderLeft: "4px solid var(--win-accent)",

                            marginBottom: "20px",

                          }}

                        >

                          <div

                            style={{

                              width: "36px",

                              height: "36px",

                              borderRadius: "8px",

                              background: "rgba(96, 205, 255, 0.15)",

                              display: "flex",

                              alignItems: "center",

                              justifyContent: "center",

                              color: "var(--win-accent)",

                              flexShrink: 0,

                            }}

                          >

                            <Radio size={18} />

                          </div>

                          <div style={{ flex: 1 }}>

                            <div style={{ display: "flex", alignItems: "center", gap: "8px", flexWrap: "wrap", marginBottom: "4px" }}>

                              <span style={{ fontWeight: 600, fontSize: "14px", color: "var(--win-text-primary)" }}>

                                Архитектура Xray Reality Ноды

                              </span>

                              <span

                                style={{

                                  fontSize: "11px",

                                  fontWeight: 500,

                                  padding: "2px 8px",

                                  borderRadius: "12px",

                                  background: "rgba(96, 205, 255, 0.18)",

                                  color: "var(--win-accent)",

                                }}

                              >

                                Домен не требуется

                              </span>

                            </div>

                            <div style={{ fontSize: "12.5px", color: "var(--win-text-secondary)", lineHeight: "1.5" }}>

                              Нода туннелирует зашифрованный трафик VLESS Reality. Веб-интерфейс и SSL-сертификаты серверу ноды не нужны — мастер-панель Remnawave связывается с ней напрямую по IP-адресу и защищенному порту API (2222).

                            </div>

                          </div>

                        </div>



                        {/* Mode Selection Tiles */}

                        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: "12px", marginBottom: "20px" }}>

                          <div

                            className={`preset-tile ${nodeLinkMode === "auto" ? "selected" : ""}`}

                            onClick={() => setNodeLinkMode("auto")}

                            style={{ padding: "14px 16px" }}

                          >

                            <div className="preset-tile-header">

                              <div className="preset-tile-icon">

                                <Zap size={16} />

                              </div>

                              <div className="preset-radio-indicator">

                                {nodeLinkMode === "auto" && <div className="preset-radio-dot" />}

                              </div>

                            </div>

                            <div className="preset-tile-title" style={{ fontSize: "13.5px" }}>Авто-привязка по API (Zero-Touch)</div>

                            <div className="preset-tile-desc" style={{ marginBottom: "0" }}>

                              Программа сама обратится к вашей мастер-панели Remnawave, зарегистрирует ноду и прошьёт SECRET_KEY при деплое.

                            </div>

                          </div>



                          <div

                            className={`preset-tile ${nodeLinkMode === "manual" ? "selected" : ""}`}

                            onClick={() => setNodeLinkMode("manual")}

                            style={{ padding: "14px 16px" }}

                          >

                            <div className="preset-tile-header">

                              <div className="preset-tile-icon">

                                <Key size={16} />

                              </div>

                              <div className="preset-radio-indicator">

                                {nodeLinkMode === "manual" && <div className="preset-radio-dot" />}

                              </div>

                            </div>

                            <div className="preset-tile-title" style={{ fontSize: "13.5px" }}>Ручной ввод ключа (SECRET_KEY)</div>

                            <div className="preset-tile-desc" style={{ marginBottom: "0" }}>

                              Создайте ноду в веб-интерфейсе панели самостоятельно и вставьте сгенерированный секретный ключ сюда.

                            </div>

                          </div>

                        </div>



                        <div className="settings-card-group">

                          {nodeLinkMode === "auto" ? (

                            <>

                              <SettingsCard

                                icon={<Globe size={18} />}

                                title="Мастер-панель Remnawave"

                                description="Адрес вашей работающей панели управления (например, https://panel.mydomain.com)"

                                action={

                                  <div style={{ display: "flex", gap: "8px", alignItems: "center" }}>

                                    {savedServers.filter((s) => s.web_url || (s.label && s.label.includes("."))).length > 0 && (

                                      <select

                                        className="win-select"

                                        style={{ maxWidth: "160px", fontSize: "11px" }}

                                        onChange={(e) => {

                                          if (e.target.value) {

                                            setNodePanelUrl(e.target.value);

                                          }

                                        }}

                                        defaultValue=""

                                      >

                                        <option value="" disabled>

                                          Из сохранённых...

                                        </option>

                                        {savedServers

                                          .filter((s) => s.web_url || (s.label && s.label.includes(".")))

                                          .map((s) => (

                                            <option key={s.id} value={extractOriginUrl(s.web_url || `https://${s.label}`)}>

                                              {s.label || s.host}

                                            </option>

                                          ))}

                                      </select>

                                    )}

                                    <input

                                      type="text"

                                      className="win-textbox"

                                      placeholder="https://panel.example.com"

                                      style={{ width: "260px" }}

                                      value={nodePanelUrl}

                                      onChange={(e) => setNodePanelUrl(e.target.value.trim())}

                                      onBlur={(e) => setNodePanelUrl(extractOriginUrl(e.target.value))}

                                    />

                                  </div>

                                }

                              />



                              <SettingsCard

                                icon={<Key size={18} />}

                                title="API Токен панели"

                                description="Токен для создания ноды (создаётся в панели Remnawave: Settings → API Tokens)"

                                action={

                                  <div style={{ display: "flex", gap: "8px", alignItems: "center" }}>

                                    <div style={{ position: "relative" }}>

                                      <input

                                        type={showNodeApiToken ? "text" : "password"}

                                        className="win-textbox"

                                        placeholder="rw_live_..."

                                        style={{ width: "220px", paddingRight: "32px" }}

                                        value={nodeApiToken}

                                        onChange={(e) => setNodeApiToken(e.target.value.trim())}

                                      />

                                      <button

                                        type="button"

                                        style={{

                                          position: "absolute",

                                          right: "8px",

                                          top: "50%",

                                          transform: "translateY(-50%)",

                                          background: "none",

                                          border: "none",

                                          cursor: "pointer",

                                          color: "var(--win-text-secondary)",

                                        }}

                                        onClick={() => setShowNodeApiToken(!showNodeApiToken)}

                                        title={showNodeApiToken ? "Скрыть токен" : "Показать токен"}

                                      >

                                        {showNodeApiToken ? <EyeOff size={13} /> : <Eye size={13} />}

                                      </button>

                                    </div>

                                    <button

                                      type="button"

                                      className="win-button"

                                      disabled={panelApiTesting || !nodePanelUrl || !nodeApiToken}

                                      onClick={handleTestPanelAPI}

                                      style={{ height: "32px", fontSize: "12px", padding: "0 10px" }}

                                    >

                                      {panelApiTesting ? <RefreshCw size={13} className="spin" /> : <Zap size={13} />}

                                      <span>Проверить API</span>

                                    </button>

                                  </div>

                                }

                              >

                                {panelApiStatus && (

                                  <InfoBar

                                    severity={panelApiStatus.success ? "success" : "error"}

                                    title={panelApiStatus.success ? "Связь с панелью подтверждена" : "Ошибка связи с API"}

                                    message={

                                      panelApiStatus.success ? (

                                        <>

                                          Токен валиден. Найдено активных нод в панели: <strong>{panelApiStatus.nodes_count}</strong>.

                                        </>

                                      ) : (

                                        panelApiStatus.error

                                      )

                                    }

                                    style={{ marginTop: "12px", marginBottom: "0" }}

                                  />

                                )}

                              </SettingsCard>



                              <SettingsCard
                                icon={<MapPin size={18} />}
                                title="Локация сервера (Страна)"
                                description="Географическое расположение сервера (флаг и страна отображаются в приложении клиентов)"
                                action={
                                  <CountrySelector
                                    value={nodeCountry}
                                    onChange={(code) => {
                                      setNodeCountry(code);
                                      setNodeName(`${code} - Node #${Math.floor(Math.random() * 90 + 10)}`);
                                    }}
                                  />
                                }
                              />

                              <SettingsCard
                                icon={<Tag size={18} />}
                                title="Название ноды в панели"
                                description="Отображаемое имя ноды в админке Remnawave и у клиентов"
                                action={
                                  <input
                                    type="text"
                                    className="win-textbox"
                                    style={{ width: "220px" }}
                                    value={nodeName}
                                    onChange={(e) => setNodeName(e.target.value)}
                                  />
                                }
                              />

                              <SettingsCard
                                icon={<Workflow size={18} />}
                                title="Режим интеграции в панели"
                                description="Способ привязки ноды к профилям конфигурации, сквадам пользователей и ссылкам подписки"
                                action={
                                  <div
                                    style={{
                                      display: "flex",
                                      alignItems: "center",
                                      background: "rgba(0, 0, 0, 0.28)",
                                      padding: "3px",
                                      borderRadius: "7px",
                                      border: "1px solid var(--win-stroke-card)",
                                      gap: "3px",
                                    }}
                                  >
                                    <button
                                      type="button"
                                      style={{
                                        display: "flex",
                                        alignItems: "center",
                                        gap: "5px",
                                        height: "28px",
                                        padding: "0 10px",
                                        fontSize: "12px",
                                        fontWeight: nodeIntegrationMode === "turnkey" ? 600 : 400,
                                        color: nodeIntegrationMode === "turnkey" ? "#ffffff" : "var(--win-text-secondary)",
                                        background: nodeIntegrationMode === "turnkey" ? "rgba(255, 255, 255, 0.10)" : "transparent",
                                        borderRadius: "5px",
                                        border: nodeIntegrationMode === "turnkey" ? "1px solid rgba(255, 255, 255, 0.14)" : "1px solid transparent",
                                        cursor: "pointer",
                                        transition: "all 0.15s ease",
                                      }}
                                      onClick={() => setNodeIntegrationMode("turnkey")}
                                    >
                                      <Zap size={12} style={{ color: nodeIntegrationMode === "turnkey" ? "var(--win-node-accent)" : "inherit" }} />
                                      <span>Под ключ (Все включено)</span>
                                    </button>
                                    <button
                                      type="button"
                                      style={{
                                        display: "flex",
                                        alignItems: "center",
                                        gap: "5px",
                                        height: "28px",
                                        padding: "0 10px",
                                        fontSize: "12px",
                                        fontWeight: nodeIntegrationMode === "custom" ? 600 : 400,
                                        color: nodeIntegrationMode === "custom" ? "#ffffff" : "var(--win-text-secondary)",
                                        background: nodeIntegrationMode === "custom" ? "rgba(255, 255, 255, 0.10)" : "transparent",
                                        borderRadius: "5px",
                                        border: nodeIntegrationMode === "custom" ? "1px solid rgba(255, 255, 255, 0.14)" : "1px solid transparent",
                                        cursor: "pointer",
                                        transition: "all 0.15s ease",
                                      }}
                                      onClick={() => {
                                        setNodeIntegrationMode("custom");
                                        if (!nodePanelUrl.trim() || !nodeApiToken.trim()) {
                                          addToast("warning", "Укажите URL панели и API токен выше, чтобы загрузить профили и сквады.");
                                        } else if (panelProfiles.length === 0) {
                                          loadPanelEntities(nodePanelUrl.trim(), nodeApiToken.trim());
                                        }
                                      }}
                                    >
                                      <span>Существующий профиль / хост</span>
                                    </button>
                                  </div>
                                }
                              >
                                {nodeIntegrationMode === "turnkey" ? (
                                  <InfoBar
                                    severity="info"
                                    title="Автоматическая интеграция под ключ"
                                    message="Программа зарегистрирует ноду, использует или создаст Reality профиль, автоматически создаст хост в разделе «Подписка» → «Хосты» (для генерации клиентских ссылок) и привяжет инбаунд к дефолтному скваду. Нода будет сразу готова к выдаче клиентам."
                                    style={{ marginTop: "10px", marginBottom: "0" }}
                                  />
                                ) : (
                                  <InfoBar
                                    severity="info"
                                    message="Вы можете привязать новую ноду к существующему профилю Xray и скваду пользователей вашей панели, не создавая дублирующих настроек."
                                    style={{ marginTop: "10px", marginBottom: "0" }}
                                  />
                                )}
                              </SettingsCard>

                              {nodeIntegrationMode === "custom" && (
                                <>
                                  <SettingsCard
                                    icon={<FileCode size={18} />}
                                    title="Профиль конфигурации и входящий порт (Inbound)"
                                    description="Выберите существующий профиль Xray (протокол и порт Reality / VLESS) для привязки ноды"
                                    action={
                                      loadingPanelEntities ? (
                                        <div style={{ display: "flex", alignItems: "center", gap: "6px", fontSize: "12px", color: "var(--win-text-secondary)" }}>
                                          <RefreshCw size={13} className="spin" />
                                          <span>Загрузка профилей...</span>
                                        </div>
                                      ) : panelProfiles.length > 0 ? (
                                        <select
                                          className="win-select"
                                          style={{ minWidth: "260px" }}
                                          value={selectedProfileInbound}
                                          onChange={(e) => setSelectedProfileInbound(e.target.value)}
                                        >
                                          {panelProfiles.map((p) => (
                                            <option key={`${p.uuid}:${p.inbound_uuid}`} value={`${p.uuid}:${p.inbound_uuid}`}>
                                              {p.name} ({p.protocol.toUpperCase()} :{p.port})
                                            </option>
                                          ))}
                                        </select>
                                      ) : (
                                        <div style={{ display: "flex", gap: "6px", alignItems: "center" }}>
                                          <span style={{ fontSize: "12px", color: "var(--win-text-tertiary)" }}>Не загружены</span>
                                          <button
                                            type="button"
                                            className="win-button"
                                            style={{ height: "28px", fontSize: "11.5px", padding: "0 10px" }}
                                            onClick={() => loadPanelEntities(nodePanelUrl.trim(), nodeApiToken.trim())}
                                          >
                                            Загрузить
                                          </button>
                                        </div>
                                      )
                                    }
                                  />

                                  <SettingsCard
                                    icon={<Users size={18} />}
                                    title="Внутренний сквад пользователей (Internal Squad)"
                                    description="Сквад доступа в панели Remnawave (Пользователи → Внутренние сквады)"
                                    action={
                                      loadingPanelEntities ? (
                                        <div style={{ display: "flex", alignItems: "center", gap: "6px", fontSize: "12px", color: "var(--win-text-secondary)" }}>
                                          <RefreshCw size={13} className="spin" />
                                          <span>Загрузка сквадов...</span>
                                        </div>
                                      ) : panelSquads.length > 0 ? (
                                        <select
                                          className="win-select"
                                          style={{ minWidth: "260px" }}
                                          value={selectedSquadUuid}
                                          onChange={(e) => setSelectedSquadUuid(e.target.value)}
                                        >
                                          <option value="none">Не добавлять в сквад (настрою вручную)</option>
                                          {panelSquads.map((sq) => (
                                            <option key={sq.uuid} value={sq.uuid}>
                                              {sq.name} ({sq.inbounds_count} инбаундов)
                                            </option>
                                          ))}
                                        </select>
                                      ) : (
                                        <div style={{ display: "flex", gap: "6px", alignItems: "center" }}>
                                          <span style={{ fontSize: "12px", color: "var(--win-text-tertiary)" }}>Не загружены</span>
                                          <select
                                            className="win-select"
                                            style={{ minWidth: "220px" }}
                                            value={selectedSquadUuid}
                                            onChange={(e) => setSelectedSquadUuid(e.target.value)}
                                          >
                                            <option value="default">Авто-привязка к Default-Squad</option>
                                            <option value="none">Не добавлять в сквад</option>
                                          </select>
                                        </div>
                                      )
                                    }
                                  />

                                  <SettingsCard
                                    icon={<Share2 size={18} />}
                                    title="Создать хост в подписке (Subscription Host)"
                                    description="Автоматически добавить хост в раздел «Подписка» → «Хосты» для формирования клиентских ссылок"
                                    action={
                                      <div
                                        className={`win-switch ${createSubscriptionHost ? "active" : ""}`}
                                        onClick={() => setCreateSubscriptionHost(!createSubscriptionHost)}
                                      >
                                        <div className="win-switch-track">
                                          <div className="win-switch-thumb" />
                                        </div>
                                        <span className="win-switch-label">{createSubscriptionHost ? "Вкл" : "Откл"}</span>
                                      </div>
                                    }
                                  />

                                  {createSubscriptionHost && (
                                    <SettingsCard
                                      icon={<Compass size={18} />}
                                      title="Адрес хоста в подписке (Address / Domain)"
                                      description="IP-адрес или домен ноды, к которому подключаются клиенты (по умолчанию: IP VPS)"
                                      action={
                                        <input
                                          type="text"
                                          className="win-textbox"
                                          style={{ width: "240px" }}
                                          placeholder={deployDecoy && nodeDecoyDomain ? nodeDecoyDomain : (sshHost || "IP ноды")}
                                          value={customHostAddress}
                                          onChange={(e) => setCustomHostAddress(e.target.value.trim())}
                                        />
                                      }
                                    />
                                  )}
                                </>
                              )}

                            </>

                          ) : (

                            <SettingsCard

                              icon={<Key size={18} />}

                              title="Секретный ключ ноды (Node Secret)"

                              description="Секретный ключ (SECRET_KEY) из вашей панели Remnawave (раздел «Ноды» → «Добавить ноду»). Можно указать сейчас или привязать позже в 1 клик."

                              action={

                                <input

                                  type="text"

                                  className="win-textbox"

                                  placeholder="Ключ из панели (опционально)"

                                  style={{ width: "280px" }}

                                  value={nodeSecret}

                                  onChange={(e) => setNodeSecret(e.target.value.trim())}

                                />

                              }

                            />

                          )}



                          <SettingsCard
                            icon={<Hash size={18} />}
                            title="Порт связи с панелью (API Port)"
                            description="Внутренний порт связи мастер-панели с этой нодой (по умолчанию 2222)"
                            action={
                              <input
                                type="number"
                                className="win-textbox"
                                style={{ width: "100px", minWidth: "100px" }}
                                value={nodePort}
                                onChange={(e) => setNodePort(parseInt(e.target.value) || 2222)}
                              />
                            }
                          />

                          <SettingsCard
                            icon={<Shield size={18} />}
                            title="Системный файрвол (UFW)"
                            description="Автоматически настроить правила безопасности: открыть SSH (22), порт связи (2222) и VPN-порты (443, 80 TCP/UDP)"
                            action={
                              <div
                                className={`win-switch ${useUfw ? "active" : ""}`}
                                onClick={() => setUseUfw(!useUfw)}
                              >
                                <div className="win-switch-track">
                                  <div className="win-switch-thumb" />
                                </div>
                                <span className="win-switch-label">{useUfw ? "Вкл" : "Откл"}</span>
                              </div>
                            }
                          />

                        </div>

                        {/* Anti-Probing & TSPU Protection Section */}
                        <div style={{ marginTop: "24px" }}>
                          <div className="settings-group-title">
                            Защита от цензуры и ТСПУ (Anti-Probing & remna-mask)
                          </div>

                          <div
                            style={{
                              display: "flex",
                              gap: "12px",
                              alignItems: "flex-start",
                              padding: "14px 16px",
                              borderRadius: "8px",
                              background: "rgba(16, 185, 129, 0.08)",
                              border: "1px solid rgba(16, 185, 129, 0.25)",
                              borderLeft: "4px solid #10b981",
                              marginBottom: "16px",
                            }}
                          >
                            <ShieldCheck size={18} style={{ color: "#10b981", flexShrink: 0, marginTop: "2px" }} />
                            <div style={{ fontSize: "12.5px", color: "var(--win-text-secondary)", lineHeight: "1.5" }}>
                              <strong style={{ color: "var(--win-text-primary)" }}>Активная маскировка узла (eGames / remna-mask):</strong> Docker публикует порт 2222 напрямую в сеть в обход UFW. Встроенная защита изолирует порт в <code style={{ color: "var(--win-accent)" }}>DOCKER-USER</code> только для IP вашей мастер-панели, устраняет палевные TCP RST пакеты и разворачивает Nginx Decoy-сайт, делая ноду невидимой для сканеров цензуры.
                            </div>
                          </div>

                          <div className="settings-card-group">
                            <SettingsCard
                              icon={<EyeOff size={18} />}
                              title="Скрыть порт ноды (2222) от сканеров ТСПУ"
                              description="Изоляция порта 2222 в цепочке DOCKER-USER: доступ разрешается исключительно IP-адресу вашей мастер-панели. Для внешних сканеров пакеты молча отбрасываются (DROP)."
                              action={
                                <div
                                  className={`win-switch ${hideNodePort ? "active" : ""}`}
                                  onClick={() => setHideNodePort(!hideNodePort)}
                                >
                                  <div className="win-switch-track">
                                    <div className="win-switch-thumb" />
                                  </div>
                                  <span className="win-switch-label">{hideNodePort ? "Вкл" : "Откл"}</span>
                                </div>
                              }
                            />

                            {hideNodePort && nodeLinkMode === "manual" && (
                              <SettingsCard
                                icon={<Network size={18} />}
                                title="IP-адрес мастер-панели (для белого списка)"
                                description="Публичный IP-адрес вашей мастер-панели Remnawave, с которого разрешено подключаться к порту 2222"
                                action={
                                  <input
                                    type="text"
                                    className="win-textbox"
                                    placeholder="1.2.3.4"
                                    style={{ width: "200px" }}
                                    value={customPanelIp}
                                    onChange={(e) => setCustomPanelIp(e.target.value.trim())}
                                  />
                                }
                              />
                            )}

                            <SettingsCard
                              icon={<Ghost size={18} />}
                              title="Скрытный режим портов (Stealth DROP)"
                              description="Устранение деанонимизирующих аномалий TCP RST. Неоткрытые порты не отвечают сигналом сброса соединения — сервер выглядит полностью выключенным."
                              action={
                                <div
                                  className={`win-switch ${stealthDrop ? "active" : ""}`}
                                  onClick={() => setStealthDrop(!stealthDrop)}
                                >
                                  <div className="win-switch-track">
                                    <div className="win-switch-thumb" />
                                  </div>
                                  <span className="win-switch-label">{stealthDrop ? "Вкл" : "Откл"}</span>
                                </div>
                              }
                            />

                            <SettingsCard
                              icon={<LayoutTemplate size={18} />}
                              title="Развернуть SelfSteal Decoy-сайт (:8444)"

                              description="Локальный сайт Nginx с SSL-сертификатом для Reality Fallback. Любая HTTPS-проба без Reality-ключа видит легальный корпоративный сайт."

                              action={

                                <div

                                  className={`win-switch ${deployDecoy ? "active" : ""}`}

                                  onClick={() => setDeployDecoy(!deployDecoy)}

                                >

                                  <div className="win-switch-track">

                                    <div className="win-switch-thumb" />

                                  </div>

                                  <span className="win-switch-label">{deployDecoy ? "Вкл" : "Откл"}</span>

                                </div>

                              }

                            >

                              {deployDecoy ? (
                                <div className="decoy-subpanel">
                                  <div className="decoy-subpanel-header">
                                    <div className="decoy-subpanel-title">Домен маскировочного сайта ноды (FQDN)</div>
                                    <div className="decoy-subpanel-desc">
                                      Домен или поддомен с A-записью на IP этой ноды. Remnawave UI автоматически выпустит валидный SSL-сертификат Let's Encrypt через Certbot.
                                    </div>
                                  </div>

                                  <div className="decoy-input-wrapper">
                                    <div className="decoy-input-icon">
                                      <Globe size={16} />
                                    </div>
                                    <input
                                      type="text"
                                      className="win-textbox decoy-input"
                                      placeholder="например, self.yourdomain.com"
                                      value={nodeDecoyDomain}
                                      onChange={(e) => setNodeDecoyDomain(e.target.value.trim())}
                                    />
                                  </div>

                                  <div style={{ display: "flex", flexDirection: "column", gap: "6px", marginTop: "4px" }}>
                                    <div style={{ fontSize: "12px", fontWeight: 600, color: "var(--win-text-primary)" }}>
                                      Шаблон маскировочного сайта (Decoy):
                                    </div>
                                    <select
                                      className="win-select"
                                      value={decoyTemplate}
                                      onChange={(e) => setDecoyTemplate(e.target.value)}
                                      style={{ width: "100%" }}
                                    >
                                      <option value="random">Случайный шаблон из каталога (Рекомендуется)</option>
                                      <option value="it_corp">Корпоративный IT-сайт (ИТ-компания, SaaS)</option>
                                      <option value="ecommerce">Интернет-магазин & Сервисы (E-Commerce)</option>
                                      <option value="portfolio">Персональное портфолио / Блог</option>
                                      <option value="clone">Клонировать существующий сайт (ввод URL)</option>
                                    </select>
                                    {decoyTemplate === "clone" && (
                                      <input
                                        type="text"
                                        className="win-textbox"
                                        placeholder="https://example.com"
                                        value={decoyCloneUrl}
                                        onChange={(e) => setDecoyCloneUrl(e.target.value.trim())}
                                        style={{ width: "100%", marginTop: "4px" }}
                                      />
                                    )}
                                  </div>

                                  {decoyDnsLoading && (
                                    <InfoBar
                                      severity="info"
                                      message="Проверка A-записи домена в глобальном DNS..."
                                      style={{ margin: 0 }}
                                    />
                                  )}
                                  {!decoyDnsLoading && decoyDnsStatus && (
                                    <InfoBar
                                      severity={decoyDnsStatus.resolved ? "success" : "warning"}
                                      title={decoyDnsStatus.resolved ? "A-запись подтверждена" : "Внимание по DNS"}
                                      message={
                                        decoyDnsStatus.resolved ? (
                                          <>
                                            A-запись найдена: <strong>{decoyDnsStatus.ip}</strong>{" "}
                                            {sshHost && decoyDnsStatus.ip === sshHost ? "(Совпадает с VPS ноды)" : ""}
                                          </>
                                        ) : (
                                          decoyDnsStatus.ip
                                            ? `Домен указывает на ${decoyDnsStatus.ip}.`
                                            : decoyDnsStatus.message || "A-запись не найдена в DNS. Добавьте A-запись у вашего DNS-провайдера."
                                        )
                                      }
                                      style={{ margin: 0 }}
                                    />
                                  )}
                                </div>
                              ) : (

                                <div style={{ marginTop: "8px", fontSize: "12px", color: "var(--win-text-secondary)" }}>

                                  Режим Classic Reality: собственный домен ноды не требуется. Xray Reality маскируется под внешний трафик (www.microsoft.com).

                                </div>

                              )}

                            </SettingsCard>

                            <SettingsCard
                              icon={<Radio size={18} />}
                              title="Блокировать BitTorrent трафик (Torrent Blocker)"
                              description="Защищает VPS от жалоб правообладателей (DMCA) и бана хостингом. Блокирует BitTorrent и DHT на уровне ядра Xray."
                              action={
                                <div
                                  className={`win-switch ${torrentBlocker ? "active" : ""}`}
                                  onClick={() => setTorrentBlocker(!torrentBlocker)}
                                >
                                  <div className="win-switch-track">
                                    <div className="win-switch-thumb" />
                                  </div>
                                  <span className="win-switch-label">{torrentBlocker ? "Вкл" : "Откл"}</span>
                                </div>
                              }
                            />

                            <SettingsCard
                              icon={<ShieldAlert size={18} />}
                              title="Блокировать спам-порты (SMTP 25/465/587)"
                              description="Предотвращает рассылку почтового спама и защищает IP сервера от попадания в спам-базы Spamhaus/CBL."
                              action={
                                <div
                                  className={`win-switch ${blockSpamPorts ? "active" : ""}`}
                                  onClick={() => setBlockSpamPorts(!blockSpamPorts)}
                                >
                                  <div className="win-switch-track">
                                    <div className="win-switch-thumb" />
                                  </div>
                                  <span className="win-switch-label">{blockSpamPorts ? "Вкл" : "Откл"}</span>
                                </div>
                              }
                            />

                            <SettingsCard
                              icon={<Zap size={18} />}
                              title="Ядро Xray-ноды (Xray Core Engine)"
                              description="Анти-ТСПУ форк Jolymmiles: поддержка X25519MLKEM768 и маскировка TLS фингерпринтов."
                              action={
                                <select
                                  className="win-select"
                                  value={xrayCoreType}
                                  onChange={(e) => setXrayCoreType(e.target.value as "official" | "jolymmiles")}
                                  style={{ minWidth: "190px" }}
                                >
                                  <option value="jolymmiles">Jolymmiles Anti-TSPU</option>
                                  <option value="official">XTLS / Xray-core</option>
                                </select>
                              }
                            />

                          </div>

                        </div>

                      </div>

                    ) : (

                      <>

                        {/* Network Architecture & Port Mode (Single Port 443 Stealth vs Classic) */}
                        {(preset === "all-in-one" || components.includes("node")) && (
                          <div className="settings-group">
                            <div className="settings-group-title">Сетевая архитектура и порты (eGames Gate)</div>
                            <div className="presets-grid" style={{ gridTemplateColumns: "1fr 1fr", marginBottom: "16px" }}>
                              <div
                                className={`preset-tile ${portMode === "stealth443" ? "selected" : ""}`}
                                onClick={() => setPortMode("stealth443")}
                                style={{ cursor: "pointer", padding: "16px" }}
                              >
                                <div className="preset-tile-header">
                                  <div className="preset-tile-icon" style={{ background: "rgba(16, 185, 129, 0.15)", color: "#10b981" }}>
                                    <ShieldCheck size={18} />
                                  </div>
                                  <div className="preset-radio-indicator">
                                    {portMode === "stealth443" && <div className="preset-radio-dot" />}
                                  </div>
                                </div>
                                <div style={{ display: "flex", alignItems: "center", gap: "6px", marginBottom: "4px" }}>
                                  <div className="preset-tile-title" style={{ margin: 0 }}>Режим единого порта 443</div>
                                  <span className="badge badge-success" style={{ fontSize: "10.5px", padding: "1px 6px" }}>Stealth Fallback</span>
                                </div>
                                <div className="preset-tile-desc">
                                  Весь входящий трафик (VPN + Панель + Подписки + Decoy) идет через порт 443. Xray и Nginx общаются через сокет в RAM (/dev/shm). Полная маскировка от активного зондирования ТСПУ.
                                </div>
                              </div>

                              <div
                                className={`preset-tile ${portMode === "classic" ? "selected" : ""}`}
                                onClick={() => {
                                  if (preset === "all-in-one") {
                                    addToast("info", "Для установки «Все в одном» на одном VPS используется режим единого порта 443 (Stealth Fallback), исключающий конфликт портов Nginx и Xray.");
                                    return;
                                  }
                                  setPortMode("classic");
                                }}
                                style={{
                                  cursor: preset === "all-in-one" ? "not-allowed" : "pointer",
                                  padding: "16px",
                                  opacity: preset === "all-in-one" ? 0.65 : 1,
                                }}
                              >
                                <div className="preset-tile-header">
                                  <div className="preset-tile-icon">
                                    <Radio size={18} />
                                  </div>
                                  <div className="preset-radio-indicator">
                                    {portMode === "classic" && <div className="preset-radio-dot" />}
                                  </div>
                                </div>
                                <div style={{ display: "flex", alignItems: "center", gap: "6px", marginBottom: "4px" }}>
                                  <div className="preset-tile-title" style={{ margin: 0 }}>Классический режим</div>
                                  {preset === "all-in-one" && (
                                    <span className="badge badge-warning" style={{ fontSize: "10.5px", padding: "1px 6px" }}>Только для раздельных VPS</span>
                                  )}
                                </div>
                                <div className="preset-tile-desc">
                                  Раздельные порты: веб-панель и подписки работают на порту 443, а узел Xray Reality принимает подключения на отдельном порту (2222 / 8444).
                                </div>
                              </div>
                            </div>
                          </div>
                        )}

                        {/* Domain Configuration */}

                        <div className="settings-group">

                          <div className="settings-group-title">Сетевые адреса</div>

                          <div className="settings-card-group">

                            <SettingsCard

                              icon={<Globe size={18} />}

                              title="Домен панели управления"

                              description="FQDN адрес для веб-интерфейса администратора (например, panel.myserver.com)"

                              action={

                                <input

                                  type="text"

                                  className="win-textbox"

                                  placeholder="panel.example.com"

                                  style={{ width: "260px" }}

                                  value={panelDomain}

                                  onChange={(e) => setPanelDomain(e.target.value.trim())}

                                />

                              }

                            >

                              {/* Live DNS Status Badge via InfoBar */}

                              {dnsLoading && (

                                <InfoBar

                                  severity="info"

                                  message="Проверка A-записи домена в глобальном DNS..."

                                  style={{ marginTop: "12px", marginBottom: "0" }}

                                />

                              )}

                              {!dnsLoading && dnsStatus && (

                                <InfoBar

                                  severity={dnsStatus.resolved ? "success" : "warning"}

                                  title={dnsStatus.resolved ? "A-запись подтверждена" : "Внимание по DNS"}

                                  message={

                                    dnsStatus.resolved ? (

                                      <>

                                        A-запись найдена: <strong>{dnsStatus.ip}</strong>{" "}

                                        {sshHost && dnsStatus.ip === sshHost ? "(Совпадает с VPS)" : ""}

                                      </>

                                    ) : (

                                      dnsStatus.ip

                                        ? `Домен указывает на ${dnsStatus.ip}.`

                                        : dnsStatus.message || "A-запись не найдена в DNS. Добавьте A-запись у вашего регистратора."

                                    )

                                  }

                                  style={{ marginTop: "12px", marginBottom: "0" }}

                                />

                              )}

                            </SettingsCard>



                            <SettingsCard
                              icon={<Link2 size={18} />}
                              title="Отдельный поддомен для подписок"

                              description="Рекомендуется Remnawave: изоляция страницы подписок (порт 3010) от админ-панели."

                              action={

                                <div style={{ display: "flex", gap: "8px", alignItems: "center" }}>

                                  {panelDomain && panelDomain.includes(".") && !subDomain && (

                                    <button

                                      type="button"

                                      className="win-button"

                                      style={{ height: "28px", fontSize: "11px", padding: "0 8px" }}

                                      onClick={() => {

                                        const parts = panelDomain.split(".");

                                        const base = parts.length > 2 ? parts.slice(1).join(".") : panelDomain;

                                        setSubDomain(`sub.${base}`);

                                        addToast("info", `Задан поддомен: sub.${base}`);

                                      }}

                                      title="Автоматически сформировать поддомен sub.<домен>"

                                    >

                                      <Wand2 size={11} style={{ color: "var(--win-accent)" }} />

                                      <span>Авто: sub.*</span>

                                    </button>

                                  )}

                                  <input

                                    type="text"

                                    className="win-textbox"

                                    placeholder="sub.example.com (опционально)"

                                    style={{ width: "240px" }}

                                    value={subDomain}

                                    onChange={(e) => setSubDomain(e.target.value.trim())}

                                  />

                                  {subDomain && (

                                    <button

                                      type="button"

                                      className="table-action-btn"

                                      onClick={() => setSubDomain("")}

                                      title="Очистить отдельный поддомен"

                                    >

                                      <X size={12} />

                                    </button>

                                  )}

                                </div>

                              }

                            >

                              {/* Live Subdomain DNS Status Badge via InfoBar */}

                              {subDnsLoading && (

                                <InfoBar

                                  severity="info"

                                  message="Проверка A-записи поддомена подписок в DNS..."

                                  style={{ marginTop: "12px", marginBottom: "0" }}

                                />

                              )}

                              {!subDnsLoading && subDnsStatus && (

                                <InfoBar

                                  severity={subDnsStatus.resolved ? "success" : "warning"}

                                  title={subDnsStatus.resolved ? "A-запись поддомена подтверждена" : "Внимание по DNS"}

                                  message={

                                    subDnsStatus.resolved ? (

                                      <>

                                        A-запись поддомена найдена: <strong>{subDnsStatus.ip}</strong>{" "}

                                        {sshHost && subDnsStatus.ip === sshHost ? "(Совпадает с VPS)" : ""}

                                      </>

                                    ) : (

                                      subDnsStatus.ip

                                        ? `Поддомен указывает на ${subDnsStatus.ip}.`

                                        : subDnsStatus.message || "A-запись поддомена не найдена в DNS. Добавьте запись sub у регистратора."

                                    )

                                  }

                                  style={{ marginTop: "12px", marginBottom: "0" }}

                                />

                              )}

                            </SettingsCard>

                          </div>

                        </div>

                        {preset !== "node-only" && components.includes("panel") && (
                          <div className="settings-group">
                            <div className="settings-group-title">Учетная запись суперадминистратора</div>
                            <div className="settings-card-group">
                              <SettingsCard
                                icon={<User size={18} />}
                                title="Способ создания аккаунта в Remnawave"
                                description="Выберите, как будет инициализирован доступ к веб-панели управления."
                                action={
                                  <div style={{ display: "flex", gap: "6px" }}>
                                    <button
                                      type="button"
                                      className={`win-button ${adminCreationMode === "auto" ? "win-button-accent" : ""}`}
                                      style={{ height: "30px", fontSize: "12px", padding: "0 12px" }}
                                      onClick={() => setAdminCreationMode("auto")}
                                    >
                                      Автоматически
                                    </button>
                                    <button
                                      type="button"
                                      className={`win-button ${adminCreationMode === "custom" ? "win-button-accent" : ""}`}
                                      style={{ height: "30px", fontSize: "12px", padding: "0 12px" }}
                                      onClick={() => {
                                        setAdminCreationMode("custom");
                                        if (!adminPassword) generateClientSafePassword();
                                      }}
                                    >
                                      Свои логин и пароль
                                    </button>
                                    <button
                                      type="button"
                                      className={`win-button ${adminCreationMode === "manual" ? "win-button-accent" : ""}`}
                                      style={{ height: "30px", fontSize: "12px", padding: "0 12px" }}
                                      onClick={() => setAdminCreationMode("manual")}
                                    >
                                      В браузере вручную
                                    </button>
                                  </div>
                                }
                              >
                                {adminCreationMode === "auto" && (
                                  <div style={{ marginTop: "12px", fontSize: "12px", color: "var(--win-text-secondary)" }}>
                                    Программа автоматически сгенерирует надежный пароль (28 символов) и создаст учетную запись суперадминистратора через API при первом запуске. Готовые реквизиты для входа будут показаны на финальном экране.
                                  </div>
                                )}

                                {adminCreationMode === "custom" && (
                                  <div style={{ marginTop: "14px", display: "flex", flexDirection: "column", gap: "10px" }}>
                                    <div style={{ display: "flex", gap: "12px", alignItems: "center" }}>
                                      <div style={{ width: "160px", fontSize: "12px", color: "var(--win-text-secondary)" }}>
                                        Имя пользователя:
                                      </div>
                                      <input
                                        type="text"
                                        className="win-textbox"
                                        style={{ width: "220px" }}
                                        value={adminUsername}
                                        onChange={(e) => setAdminUsername(e.target.value)}
                                        placeholder="admin"
                                      />
                                    </div>
                                    <div style={{ display: "flex", gap: "12px", alignItems: "center" }}>
                                      <div style={{ width: "160px", fontSize: "12px", color: "var(--win-text-secondary)" }}>
                                        Пароль администратора:
                                      </div>
                                      <div style={{ position: "relative", width: "320px", display: "flex", alignItems: "center" }}>
                                        <input
                                          type={showAdminPassword ? "text" : "password"}
                                          className="win-textbox"
                                          style={{ width: "100%", paddingRight: "70px", fontFamily: showAdminPassword ? "monospace" : "inherit" }}
                                          value={adminPassword}
                                          onChange={(e) => setAdminPassword(e.target.value)}
                                          placeholder="Минимум 24 символа (Aa1...)"
                                        />
                                        <div style={{ position: "absolute", right: "6px", display: "flex", gap: "4px" }}>
                                          <button
                                            type="button"
                                            className="table-action-btn"
                                            onClick={() => setShowAdminPassword(!showAdminPassword)}
                                            title={showAdminPassword ? "Скрыть пароль" : "Показать пароль"}
                                          >
                                            {showAdminPassword ? <EyeOff size={14} /> : <Eye size={14} />}
                                          </button>
                                          <button
                                            type="button"
                                            className="table-action-btn"
                                            onClick={generateClientSafePassword}
                                            title="Сгенерировать надежный пароль"
                                          >
                                            <Dices size={14} />
                                          </button>
                                        </div>
                                      </div>
                                    </div>
                                    <div style={{ fontSize: "11px", color: "var(--win-text-secondary)", marginTop: "2px" }}>
                                      Требование безопасности Remnawave: длина не менее 24 символов, обязательное наличие заглавных букв, строчных букв и цифр.
                                    </div>
                                  </div>
                                )}

                                {adminCreationMode === "manual" && (
                                  <InfoBar
                                    severity="info"
                                    title="Регистрация через веб-интерфейс"
                                    message="Автоматическое создание пользователя будет отключено. После завершения развертывания вы сможете перейти по ссылке в браузер и самостоятельно пройти первичную регистрацию супер-админа."
                                    style={{ marginTop: "12px", marginBottom: "0" }}
                                  />
                                )}
                              </SettingsCard>
                            </div>
                          </div>
                        )}

                        {/* Security & Gate via SettingsExpander */}

                        <div className="settings-group">

                          <div className="settings-group-title">Безопасность и маскировка</div>

                          <div className="settings-card-group">

                            <SettingsExpander

                              icon={<Shield size={18} />}

                              title="Безопасность периметра и защита от цензуры"

                              description="Настройки защитного шлюза, маскировки, системного файрвола и SSL сертификатов"

                              defaultExpanded={true}

                              items={[

                                {

                                  icon: <Lock size={16} />,

                                  title: "Security Gate (Защита от сканеров)",

                                  description: `Скрывает панель за секретным URL. При обращении к корню без ключа панель недоступна (/${secretGate || "..."})`,

                                  action: (

                                    <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>

                                      {useGate && (

                                        <div style={{ display: "flex", alignItems: "center", gap: "6px" }}>

                                          <input

                                            type="text"

                                            className="win-textbox"

                                            style={{ width: "150px" }}

                                            value={secretGate}

                                            placeholder="gate_token"

                                            onChange={(e) => setSecretGate(e.target.value)}

                                          />

                                          <button

                                            type="button"

                                            className="win-button"

                                            title="Сгенерировать случайный токен гейта"

                                            onClick={() => {

                                              const rnd = "gate_" + Math.random().toString(36).substring(2, 10);

                                              setSecretGate(rnd);

                                              addToast("info", `Сгенерирован защитный путь: /${rnd}`);

                                            }}

                                            style={{ height: "30px", padding: "0 8px" }}

                                          >

                                            <Dices size={14} />

                                            <span>Сгенерировать</span>

                                          </button>

                                        </div>

                                      )}

                                      <div

                                        className={`win-switch ${useGate ? "active" : ""}`}

                                        onClick={() => setUseGate(!useGate)}

                                      >

                                        <div className="win-switch-track">

                                          <div className="win-switch-thumb" />

                                        </div>

                                        <span className="win-switch-label">{useGate ? "Вкл" : "Откл"}</span>

                                      </div>

                                    </div>

                                  ),

                                },

                                {

                                  icon: <LayoutTemplate size={16} />,

                                  title: (preset === "all-in-one" || components.includes("node"))
                                    ? "Маскировочный веб-сайт (Decoy Fallback)"
                                    : "Сайт-заглушка для панели (Gate Camouflage)",

                                  description: (preset === "all-in-one" || components.includes("node"))
                                    ? "Посетители и сканеры без Reality-ключа видят легитимный сайт-визитку вместо админки и Reality-ноды (Fallback на 443)"
                                    : "Сайт-заглушка для Security Gate: скрывает админ-панель за нейтральной страницей для посетителей без секретного пути (Gate)",

                                  action: (

                                    <div

                                      className={`win-switch ${useCamouflage ? "active" : ""}`}

                                      onClick={() => setUseCamouflage(!useCamouflage)}

                                    >

                                      <div className="win-switch-track">

                                        <div className="win-switch-thumb" />

                                      </div>

                                      <span className="win-switch-label">{useCamouflage ? "Вкл" : "Откл"}</span>

                                    </div>

                                  ),

                                },

                                {

                                  icon: <Cloud size={16} />,

                                  title: "Cloudflare DNS-01 API (Wildcard SSL)",

                                  description: `Выпуск Wildcard (*.${panelDomain && panelDomain.includes(".") ? panelDomain.split(".").slice(-2).join(".") : "domain.com"}) без открытого порта 80. Работает даже за Cloudflare Proxy.`,

                                  action: (

                                    <div

                                      className={`win-switch ${useCloudflareDns ? "active" : ""}`}

                                      onClick={() => setUseCloudflareDns(!useCloudflareDns)}

                                    >

                                      <div className="win-switch-track">

                                        <div className="win-switch-thumb" />

                                      </div>

                                      <span className="win-switch-label">{useCloudflareDns ? "Вкл" : "Откл"}</span>

                                    </div>

                                  ),

                                },

                                {

                                  icon: <ShieldCheck size={16} />,

                                  title: "Системный файрвол (UFW)",

                                  description: "Автоматически открывает порты 80, 443 и SSH, блокируя нежелательный доступ",

                                  action: (

                                    <div

                                      className={`win-switch ${useUfw ? "active" : ""}`}

                                      onClick={() => setUseUfw(!useUfw)}

                                    >

                                      <div className="win-switch-track">

                                        <div className="win-switch-thumb" />

                                      </div>

                                      <span className="win-switch-label">{useUfw ? "Вкл" : "Откл"}</span>

                                    </div>

                                  ),

                                },

                                ...((preset === "all-in-one" || components.includes("panel") || preset === "panel-only") ? [
                                  {
                                    icon: <Cpu size={16} />,
                                    title: "Оптимизация производительности (RAM Boost & Sockets)",
                                    description: "Valkey через Unix Domain Socket (без TCP loopback оверхеда), 512MB shared memory для Postgres, ulimits 1M дескрипторов и тюнинг swappiness ядра (Рекомендуется).",
                                    action: (
                                      <div
                                        className={`win-switch ${optimizePerformance ? "active" : ""}`}
                                        onClick={() => setOptimizePerformance(!optimizePerformance)}
                                      >
                                        <div className="win-switch-track">
                                          <div className="win-switch-thumb" />
                                        </div>
                                        <span className="win-switch-label">{optimizePerformance ? "Вкл" : "Откл"}</span>
                                      </div>
                                    ),
                                  },
                                ] : []),

                                ...((preset === "all-in-one" || components.includes("node")) ? [
                                  {
                                    icon: <Radio size={16} />,
                                    title: "Блокировать BitTorrent трафик (Torrent Blocker)",
                                    description: "Защищает VPS от жалоб правообладателей (DMCA) и бана хостингом. Блокирует BitTorrent и DHT на уровне Xray.",
                                    action: (
                                      <div
                                        className={`win-switch ${torrentBlocker ? "active" : ""}`}
                                        onClick={() => setTorrentBlocker(!torrentBlocker)}
                                      >
                                        <div className="win-switch-track">
                                          <div className="win-switch-thumb" />
                                        </div>
                                        <span className="win-switch-label">{torrentBlocker ? "Вкл" : "Откл"}</span>
                                      </div>
                                    ),
                                  },
                                  {
                                    icon: <ShieldAlert size={16} />,
                                    title: "Блокировать спам-порты (SMTP 25/465/587)",
                                    description: "Предотвращает рассылку почтового спама и защищает IP сервера от попадания в спам-базы Spamhaus/CBL.",
                                    action: (
                                      <div
                                        className={`win-switch ${blockSpamPorts ? "active" : ""}`}
                                        onClick={() => setBlockSpamPorts(!blockSpamPorts)}
                                      >
                                        <div className="win-switch-track">
                                          <div className="win-switch-thumb" />
                                        </div>
                                        <span className="win-switch-label">{blockSpamPorts ? "Вкл" : "Откл"}</span>
                                      </div>
                                    ),
                                  },
                                  {
                                    icon: <Zap size={16} />,
                                    title: "Ядро Xray-ноды (Xray Core Engine)",
                                    description: "Анти-ТСПУ форк Jolymmiles: поддержка X25519MLKEM768 и маскировка TLS фингерпринтов.",
                                    action: (
                                      <select
                                        className="win-select"
                                        value={xrayCoreType}
                                        onChange={(e) => setXrayCoreType(e.target.value as "official" | "jolymmiles")}
                                        style={{ minWidth: "190px" }}
                                      >
                                        <option value="jolymmiles">Jolymmiles Anti-TSPU</option>
                                        <option value="official">XTLS / Xray-core</option>
                                      </select>
                                    ),
                                  },
                                ] : []),

                              ]}

                            >

                              {useCamouflage && (preset === "all-in-one" || components.includes("node")) && (
                                <div style={{ padding: "14px 16px", background: "var(--win-fill-card-secondary)", borderTop: "1px solid var(--win-stroke-card)", display: "flex", flexDirection: "column", gap: "10px" }}>
                                  <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", flexWrap: "wrap", gap: "10px" }}>
                                    <div>
                                      <div style={{ fontSize: "13px", fontWeight: 600, color: "var(--win-text-primary)" }}>
                                        Шаблон маскировочного сайта (Decoy / SelfSteal)
                                      </div>
                                      <div style={{ fontSize: "12px", color: "var(--win-text-secondary)" }}>
                                        Шаблон сайта из каталога eGames или автоматическое клонирование внешнего ресурса
                                      </div>
                                    </div>
                                    <select
                                      className="win-select"
                                      value={decoyTemplate}
                                      onChange={(e) => setDecoyTemplate(e.target.value)}
                                      style={{ minWidth: "260px" }}
                                    >
                                      <option value="random">Случайный шаблон из каталога (Рекомендуется)</option>
                                      <option value="it_corp">Корпоративный IT-сайт (ИТ-компания, SaaS)</option>
                                      <option value="ecommerce">Интернет-магазин & Сервисы (E-Commerce)</option>
                                      <option value="portfolio">Персональное портфолио / Блог</option>
                                      <option value="clone">Клонировать существующий сайт (ввод URL)</option>
                                    </select>
                                  </div>
                                  {decoyTemplate === "clone" && (
                                    <div style={{ marginTop: "2px" }}>
                                      <input
                                        type="text"
                                        className="win-textbox"
                                        placeholder="https://example.com (введите URL сайта для клонирования)"
                                        style={{ width: "100%" }}
                                        value={decoyCloneUrl}
                                        onChange={(e) => setDecoyCloneUrl(e.target.value.trim())}
                                      />
                                      <div style={{ fontSize: "11px", color: "var(--win-text-tertiary)", marginTop: "4px" }}>
                                        Укажите полный адрес легитимного веб-сайта. Скрипт создаст точную локальную копию (HTML, CSS, изображения).
                                      </div>
                                    </div>
                                  )}
                                </div>
                              )}

                              {useCloudflareDns && (

                                <div style={{ padding: "12px 16px", background: "var(--win-fill-card-secondary)", borderTop: "1px solid var(--win-stroke-card)", display: "flex", flexDirection: "column", gap: "6px" }}>

                                  <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>

                                    <div style={{ position: "relative", flex: 1 }}>

                                      <input

                                        type={showCloudflareToken ? "text" : "password"}

                                        className="win-textbox"

                                        placeholder="Cloudflare API Token (Zone:DNS:Edit)"

                                        style={{ width: "100%", paddingRight: "36px" }}

                                        value={cloudflareToken}

                                        onChange={(e) => setCloudflareToken(e.target.value.trim())}

                                      />

                                      <button

                                        type="button"

                                        style={{

                                          position: "absolute",

                                          right: "8px",

                                          top: "50%",

                                          transform: "translateY(-50%)",

                                          background: "none",

                                          border: "none",

                                          cursor: "pointer",

                                          color: "var(--win-text-secondary)",

                                        }}

                                        onClick={() => setShowCloudflareToken(!showCloudflareToken)}

                                        title={showCloudflareToken ? "Скрыть токен" : "Показать токен"}

                                      >

                                        {showCloudflareToken ? <EyeOff size={14} /> : <Eye size={14} />}

                                      </button>

                                    </div>

                                  </div>

                                  <span style={{ fontSize: "11px", color: "var(--win-text-tertiary)" }}>

                                    Токен создается в панели Cloudflare: My Profile → API Tokens → Create Token (шаблон: Edit zone DNS).

                                  </span>

                                </div>

                              )}

                            </SettingsExpander>

                          </div>

                        </div>

                      </>

                    )}

                  </div>

                )}



                {/* STEP 3: DEPLOYMENT & TERMINAL */}
                {step === 3 && (
                  <div key="step-3" className={`step-enter ${stepDirection === "forward" ? "step-forward" : "step-backward"}`}>

                    {/* Method Choice */}

                    <div className="settings-group">

                      <div className="settings-group-title">Способ развертывания</div>

                      <div className="settings-card-group">


                        <div className="settings-card-row">

                          <div className="settings-card-info">

                            <div className="settings-card-icon">

                              <Server size={18} />

                            </div>

                            <div className="settings-card-text">

                              <div className="settings-card-title">Автоматическая установка на удаленный Linux VPS</div>

                              <div className="settings-card-desc">

                                Подключение по SSH, создание Swap файла, настройка Docker и запуск Remnawave в 1 клик

                              </div>

                            </div>

                          </div>

                          <div className="settings-card-action">

                            <div

                              className={`win-switch ${deployMode === "remote" ? "active" : ""}`}

                              onClick={() => setDeployMode(deployMode === "remote" ? "generate" : "remote")}

                            >

                              <div className="win-switch-track">

                                <div className="win-switch-thumb" />

                              </div>

                              <span className="win-switch-label">

                                {deployMode === "remote" ? "SSH" : "Файлы"}

                              </span>

                            </div>

                          </div>

                        </div>

                      </div>

                    </div>



                    {/* Quick Prefill from Saved Servers */}

                    {deployMode === "remote" && savedServers.length > 0 && (

                      <div className="settings-group">

                        <div className="settings-group-title">Использовать сохранённый сервер</div>

                        <div style={{ display: "flex", gap: "8px", flexWrap: "wrap" }}>

                          {savedServers.map((srv) => (

                            <button

                              key={srv.id}

                              className="win-button"

                              onClick={() => handlePrefillFromServer(srv)}

                            >

                              <Server size={13} />

                              <span>{srv.label || srv.host}</span>

                              <span style={{ fontSize: "11px", color: "var(--win-text-tertiary)" }}>

                                ({srv.host})

                              </span>

                            </button>

                          ))}

                        </div>

                      </div>

                    )}



                    {/* SSH Credentials if Remote */}

                    {deployMode === "remote" ? (

                      <div className="settings-group">

                        <div className="settings-group-title">Параметры подключения SSH</div>

                        <div className="settings-card-group">

                          <div className="settings-card-row" style={{ flexWrap: "wrap", gap: "12px" }}>

                            <div style={{ display: "flex", gap: "12px", flex: 1, minWidth: "300px" }}>

                              <div style={{ flex: 2 }}>

                                <div className="hw-spec-label">IP адрес или хост</div>

                                <input

                                  type="text"

                                  className="win-textbox"

                                  style={{ width: "100%" }}

                                  placeholder="194.87.100.25"

                                  value={sshHost}

                                  onChange={(e) => {
                                    setSshHost(e.target.value.trim());
                                    setSshStatus(null);
                                    setWizardIPResult(null);
                                  }}

                                />

                              </div>

                              <div style={{ width: "80px" }}>

                                <div className="hw-spec-label">Порт</div>

                                <input

                                  type="number"

                                  className="win-textbox"

                                  style={{ width: "100%", minWidth: "auto" }}

                                  value={sshPort}

                                  onChange={(e) => setSshPort(parseInt(e.target.value) || 22)}

                                />

                              </div>

                            </div>



                            <div style={{ display: "flex", gap: "12px", flex: 1, minWidth: "300px" }}>

                              <div style={{ width: "120px" }}>

                                <div className="hw-spec-label">Пользователь</div>

                                <input

                                  type="text"

                                  className="win-textbox"

                                  style={{ width: "100%", minWidth: "auto" }}

                                  value={sshUser}

                                  onChange={(e) => setSshUser(e.target.value)}

                                />

                              </div>

                              <div style={{ flex: 1 }}>

                                <div className="hw-spec-label">Пароль root</div>

                                <input

                                  type="password"

                                  className="win-textbox"

                                  style={{ width: "100%" }}

                                  placeholder="Пароль root"

                                  value={sshPass}

                                  onChange={(e) => setSshPass(e.target.value)}

                                />

                              </div>

                            </div>

                          </div>



                          {/* Row: SSH Key (Optional) */}

                          <div className="settings-card-row" style={{ alignItems: "center" }}>

                            <div style={{ flex: 1 }}>

                              <div className="hw-spec-label">Приватный SSH-ключ (опционально, если вход по ключу)</div>

                              <div style={{ display: "flex", gap: "8px", alignItems: "center", marginTop: "4px" }}>

                                <input

                                  type="text"

                                  className="win-textbox"

                                  style={{ flex: 1 }}

                                  placeholder="Путь к id_rsa / id_ed25519 (например: C:\Users\user\.ssh\id_ed25519)"

                                  value={sshKey}

                                  onChange={(e) => setSshKey(e.target.value)}

                                />

                                <button

                                  type="button"

                                  className="win-button"

                                  onClick={async () => {

                                    try {

                                      const keyPath = await SelectSSHKeyFile();

                                      if (keyPath) {

                                        setSshKey(keyPath);

                                        addToast("info", "Выбран SSH-ключ: " + keyPath);

                                      }

                                    } catch (e) {

                                      console.error(e);

                                    }

                                  }}

                                  style={{ height: "30px", whiteSpace: "nowrap" }}

                                >

                                  <Key size={13} />

                                  <span>Выбрать файл...</span>

                                </button>

                                {sshKey && (

                                  <button

                                    type="button"

                                    className="win-button"

                                    onClick={() => setSshKey("")}

                                    title="Очистить путь к ключу"

                                    style={{ height: "30px", padding: "0 8px" }}

                                  >

                                    <X size={13} />

                                  </button>

                                )}

                              </div>

                            </div>

                          </div>



                          {/* SSH Test & IP Block Check Action Row */}

                          <div className="settings-card-row" style={{ flexWrap: "wrap", gap: "10px" }}>

                            <div className="settings-card-info" style={{ flex: "0 0 auto", display: "flex", gap: "8px", flexWrap: "wrap" }}>

                              <button

                                className="win-button"

                                disabled={!sshHost || (!sshPass && !sshKey) || sshLoading}

                                onClick={handleTestSSH}

                              >

                                {sshLoading ? (

                                  <>

                                    <RefreshCw size={14} className="animate-spin" />

                                    <span>Опрос сервера...</span>

                                  </>

                                ) : (

                                  <>

                                    <Server size={14} />

                                    <span>Проверить соединение и железо</span>

                                  </>

                                )}

                              </button>

                              <button

                                type="button"

                                className="win-button"

                                disabled={!sshHost || wizardIPChecking}

                                onClick={handleCheckWizardIP}

                                title="Проверить доступность IP-адреса из РФ (проверка блокировки ТСПУ / РКН)"

                                style={{ display: "inline-flex", alignItems: "center", gap: "6px" }}

                              >

                                {wizardIPChecking ? (

                                  <>

                                    <RefreshCw size={14} className="animate-spin" />

                                    <span>Диагностика ТСПУ...</span>

                                  </>

                                ) : (

                                  <>

                                    <ShieldAlert size={14} style={{ color: "#38bdf8" }} />

                                    <span>Проверить доступность из РФ</span>

                                  </>

                                )}

                              </button>

                            </div>

                            <div className="settings-card-action">

                              {sshStatus && (

                                <span

                                  style={{

                                    fontSize: "12px",

                                    padding: "4px 10px",

                                    borderRadius: "12px",

                                    fontWeight: 600,

                                    background: sshStatus.success ? "rgba(16, 185, 129, 0.15)" : "rgba(239, 68, 68, 0.15)",

                                    color: sshStatus.success ? "#34d399" : "#f87171",

                                    border: `1px solid ${sshStatus.success ? "rgba(16, 185, 129, 0.3)" : "rgba(239, 68, 68, 0.3)"}`,

                                    display: "inline-flex",

                                    alignItems: "center",

                                    gap: "6px",

                                  }}

                                >

                                  {sshStatus.success ? <Check size={12} /> : <AlertTriangle size={12} />}

                                  <span>{sshStatus.success ? "Соединение установлено" : "Ошибка подключения"}</span>

                                </span>

                              )}

                            </div>

                          </div>

                          {/* Wizard IP Check Result Card */}
                          {wizardIPResult && (
                            <div style={{ padding: "0 16px 12px 16px" }}>
                              <div className={`ip-check-card ${wizardIPResult.status}`}>
                                <div className="ip-check-header">
                                  <div className="ip-check-verdict-group">
                                    <span className={`ip-check-badge ${wizardIPResult.status}`}>
                                      {wizardIPResult.status === "clean" ? (
                                        <>
                                          <CheckCircle2 size={12} />
                                          <span>Чистый IP · Доступен в РФ</span>
                                        </>
                                      ) : wizardIPResult.status === "blocked" ? (
                                        <>
                                          <AlertTriangle size={12} />
                                          <span>Блокировка ТСПУ / РКН</span>
                                        </>
                                      ) : wizardIPResult.status === "partial" ? (
                                        <>
                                          <AlertCircle size={12} />
                                          <span>Частичная доступность</span>
                                        </>
                                      ) : (
                                        <>
                                          <AlertOctagon size={12} />
                                          <span>Хост недоступен</span>
                                        </>
                                      )}
                                    </span>
                                    <span style={{ fontSize: "12px", fontWeight: 600, color: "var(--win-text-primary)" }}>
                                      {wizardIPResult.host}:{wizardIPResult.port}
                                    </span>
                                  </div>

                                  <div style={{ display: "flex", alignItems: "center", gap: "6px" }}>
                                    {wizardIPResult.isp_info && (
                                      <span className="server-chip" style={{ fontSize: "11px" }}>
                                        <Server size={10} />
                                        <span>{wizardIPResult.isp_info}</span>
                                      </span>
                                    )}
                                    {wizardIPResult.location_info && (
                                      <span className="server-chip" style={{ fontSize: "11px" }}>
                                        <MapPin size={10} />
                                        <span>{wizardIPResult.location_info}</span>
                                      </span>
                                    )}
                                    {wizardIPResult.is_vpn_detected && wizardIPResult.client_country && (
                                      <span
                                        className="server-chip"
                                        style={{
                                          fontSize: "11px",
                                          color: "#38bdf8",
                                          borderColor: "rgba(56, 189, 248, 0.35)",
                                          background: "rgba(56, 189, 248, 0.08)",
                                        }}
                                        title={`Ваш ПК подключен через VPN (${wizardIPResult.client_country})`}
                                      >
                                        <Globe size={10} />
                                        <span>VPN: {wizardIPResult.client_country}</span>
                                      </span>
                                    )}
                                    <button
                                      type="button"
                                      className="win-button"
                                      style={{ height: "22px", padding: "0 6px", fontSize: "11px" }}
                                      onClick={() => setWizardIPResult(null)}
                                      title="Закрыть отчет"
                                    >
                                      <X size={11} />
                                    </button>
                                  </div>
                                </div>

                                <p className="ip-check-summary">{wizardIPResult.summary}</p>

                                {/* Multi-Port Breakdown or Legacy Probes Row */}
                                {wizardIPResult.port_details && wizardIPResult.port_details.length > 0 ? (
                                  <div style={{ display: "flex", flexDirection: "column", gap: "8px", marginTop: "8px" }}>
                                    {wizardIPResult.port_details.map((pd: any, pdIdx: number) => (
                                      <div key={pdIdx} className="ip-port-detail-box">
                                        <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: "8px" }}>
                                          <span style={{ fontSize: "11.5px", fontWeight: 600, color: "var(--win-text-primary)" }}>
                                            {pd.protocol}
                                          </span>
                                          <span className={`ip-check-badge ${pd.status}`} style={{ fontSize: "10.5px", padding: "1px 6px" }}>
                                            {pd.status === "clean" ? "Доступен" : pd.status === "blocked" ? "Блокировка ТСПУ" : pd.status === "partial" ? "Частично" : "Офлайн"}
                                          </span>
                                        </div>
                                        <div className="ip-probe-pills-row">
                                          {pd.probes.map((pr: any, pIdx: number) => {
                                            const isRef = pr.country_code === "de";
                                            const pillClass = isRef ? "ref" : pr.success ? "ok" : "fail";
                                            return (
                                              <span key={pIdx} className={`ip-probe-pill ${pillClass}`} title={pr.error || `Задержка: ${pr.latency_ms} ms`}>
                                                {pr.success ? <Check size={10} /> : <X size={10} />}
                                                <span>{pr.location}: {pr.success ? `${pr.latency_ms} ms` : (pr.error || "Сбой")}</span>
                                              </span>
                                            );
                                          })}
                                        </div>
                                      </div>
                                    ))}
                                  </div>
                                ) : wizardIPResult.probes && wizardIPResult.probes.length > 0 ? (
                                  <div className="ip-probe-pills-row">
                                    {wizardIPResult.probes.map((pr: any, pIdx: number) => {
                                      const isRef = pr.country_code === "de";
                                      const pillClass = isRef ? "ref" : pr.success ? "ok" : "fail";
                                      return (
                                        <span key={pIdx} className={`ip-probe-pill ${pillClass}`} title={pr.error || `Задержка: ${pr.latency_ms} ms`}>
                                          {pr.success ? <Check size={10} /> : <X size={10} />}
                                          <span>{pr.location}: {pr.success ? `${pr.latency_ms} ms` : (pr.error || "Сбой")}</span>
                                        </span>
                                      );
                                    })}
                                  </div>
                                ) : null}
                              </div>
                            </div>
                          )}

                          {/* Dedicated Error Callout if SSH connection failed */}

                          {sshStatus && !sshStatus.success && (

                            <div

                              style={{

                                padding: "12px 18px",

                                background: "rgba(239, 68, 68, 0.08)",

                                borderBottom: "1px solid rgba(239, 68, 68, 0.2)",

                                borderLeft: "3.5px solid #ef4444",

                                display: "flex",

                                alignItems: "flex-start",

                                gap: "10px",

                                fontSize: "12px",

                                color: "var(--win-text-primary)",

                                lineHeight: 1.45,

                              }}

                            >

                              <AlertTriangle size={16} style={{ color: "#ef4444", flexShrink: 0, marginTop: "2px" }} />

                              <div style={{ wordBreak: "break-word" }}>

                                <strong style={{ color: "#f87171" }}>Не удалось подключиться по SSH:</strong>{" "}

                                <span>{sshStatus.error}</span>

                              </div>

                            </div>

                          )}



                          {/* Hardware Specs Display */}

                          {sshStatus && sshStatus.success && (

                            <div className="hw-specs-grid">

                              <div className="hw-spec-item">

                                <span className="hw-spec-label">Операционная система</span>

                                <span className="hw-spec-val">{sshStatus.os || "Linux"}</span>

                              </div>

                              <div className="hw-spec-item">

                                <span className="hw-spec-label">Процессор (vCPU)</span>

                                <span className="hw-spec-val">{sshStatus.cpu || "—"}</span>

                              </div>

                              <div className="hw-spec-item">

                                <span className="hw-spec-label">Оперативная память</span>

                                <span className="hw-spec-val">{sshStatus.ram || "—"}</span>

                              </div>

                              <div className="hw-spec-item">

                                <span className="hw-spec-label">Статус проверки</span>

                                <span className="hw-spec-val">Доступ подтвержден</span>

                              </div>

                            </div>

                          )}

                        </div>

                      </div>

                    ) : (

                      /* Local Generation Folder */

                      <div className="settings-group">

                        <div className="settings-group-title">Папка для сохранения конфигурации</div>

                        <div className="settings-card-group">

                          <div className="settings-card-row">

                            <div className="settings-card-info">

                              <div className="settings-card-icon">

                                <Folder size={18} />

                              </div>

                              <div className="settings-card-text">

                                <div className="settings-card-title">Директория экспорта</div>

                                <div className="settings-card-desc">{outDir}</div>

                              </div>

                            </div>

                            <div className="settings-card-action">

                              <button className="win-button" onClick={handleBrowseFolder}>

                                Выбрать папку...

                              </button>

                            </div>

                          </div>

                        </div>

                      </div>

                    )}



                    {/* Deployment Parameters Summary Card */}
                    <div className="settings-group">
                      <div className="settings-group-title">Сводка параметров перед запуском</div>
                      <div className="settings-card-group" style={{ padding: "14px 16px", background: "var(--win-fill-card-secondary)", border: "1px solid var(--win-stroke-card)", borderRadius: "8px" }}>
                        <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(200px, 1fr))", gap: "12px", fontSize: "12.5px" }}>
                          <div>
                            <span style={{ color: "var(--win-text-secondary)", display: "block", marginBottom: "2px" }}>Режим топологии:</span>
                            <strong style={{ color: "var(--win-text-primary)" }}>
                              {portMode === "stealth443" ? "eGames Stealth 443 (Single Port)" : "Классический режим портов"}
                            </strong>
                          </div>
                          {preset !== "node-only" && components.includes("panel") && (
                            <div>
                              <span style={{ color: "var(--win-text-secondary)", display: "block", marginBottom: "2px" }}>Суперадминистратор панели:</span>
                              <strong style={{ color: "var(--win-accent)" }}>
                                {adminCreationMode === "auto"
                                  ? "Автоматически (генерация пароля 28 симв.)"
                                  : adminCreationMode === "custom"
                                  ? `Свой аккаунт (логин: ${adminUsername.trim() || "admin"})`
                                  : "В браузере вручную при первом входе"}
                              </strong>
                            </div>
                          )}
                          {panelDomain && (
                            <div>
                              <span style={{ color: "var(--win-text-secondary)", display: "block", marginBottom: "2px" }}>Домен панели:</span>
                              <strong style={{ color: "var(--win-text-primary)" }}>{panelDomain}</strong>
                            </div>
                          )}
                          {components.includes("subscription") && subDomain && (
                            <div>
                              <span style={{ color: "var(--win-text-secondary)", display: "block", marginBottom: "2px" }}>Домен подписок:</span>
                              <strong style={{ color: "var(--win-text-primary)" }}>{subDomain}</strong>
                            </div>
                          )}
                        </div>
                      </div>
                    </div>

                    {/* Launch Button */}
                    <div style={{ marginTop: "16px", marginBottom: "8px", display: "flex", justifyContent: "flex-end" }}>

                      <button

                        className="win-button primary"

                        style={{ height: "36px", padding: "0 22px", fontSize: "14px" }}

                        disabled={isDeploying || (deployMode === "remote" && (!sshHost || (!sshPass && !sshKey)))}

                        onClick={handleStart}

                      >

                        {isDeploying ? (

                          <>

                            <RefreshCw size={15} className="animate-spin" />

                            <span>Выполняется установка...</span>

                          </>

                        ) : deployMode === "remote" ? (

                          <>

                            <Play size={15} />

                            <span>Запустить установку на сервере</span>

                          </>

                        ) : (

                          <>

                            <Folder size={15} />

                            <span>Сгенерировать файлы конфигурации</span>

                          </>

                        )}

                      </button>

                    </div>



                    {/* Success Hero Card after deploy */}

                    {deployFinished && deploySuccess && (

                      <div className="hero-success-card">

                        <div className="hero-success-header">

                          <div className="hero-success-icon">

                            <CheckCircle2 size={24} />

                          </div>

                          <div>
                            <div className="hero-success-title">
                              {preset === "node-only"
                                ? "Xray Reality Нода успешно развернута!"
                                : preset === "all-in-one"
                                ? "Стек Remnawave (Панель + Нода) успешно развернут!"
                                : "Панель Remnawave успешно развернута!"}
                            </div>
                            <div className="hero-success-desc">
                              {preset === "node-only"
                                ? "Контейнер remnanode запущен, защита от ТСПУ активна, нода подключена к панели."
                                : preset === "all-in-one"
                                ? "Весь стек развернут, база данных настроена, Reality нода привязана к панели."
                                : "Все контейнеры запущены, база данных инициализирована, SSL сертификат получен."}
                            </div>
                          </div>

                        </div>



                        {deployedWebUrl && preset !== "node-only" && (

                          <div className="hero-link-box">

                            <span>{deployedWebUrl}</span>

                            <button

                              className="win-button"

                              style={{ height: "26px", padding: "0 8px", fontSize: "11px" }}

                              onClick={() => handleCopy(deployedWebUrl, "Адрес панели", "deployed-web-url")}

                            >

                              {copiedKey === "deployed-web-url" ? (
                                <>
                                  <Check size={12} style={{ color: "#34d399" }} />
                                  <span style={{ color: "#34d399", fontWeight: 600 }}>Скопировано!</span>
                                </>
                              ) : (
                                <>
                                  <Copy size={12} />
                                  <span>Копировать</span>
                                </>
                              )}

                            </button>

                          </div>

                        )}

                        {(deployedAdminUser || deployedAdminPass || deployedSubUrl) && preset !== "node-only" && deployedAdminMode !== "manual" && (
                          <div
                            style={{
                              margin: "12px 0",
                              padding: "14px 16px",
                              background: "var(--win-fill-card-secondary)",
                              border: "1px solid var(--win-stroke-card)",
                              borderRadius: "8px",
                              display: "flex",
                              flexDirection: "column",
                              gap: "10px",
                            }}
                          >
                            <div style={{ fontSize: "12.5px", fontWeight: 600, color: "var(--win-accent)", display: "flex", alignItems: "center", gap: "6px" }}>
                              <Shield size={14} />
                              <span>Учетные данные доступа к системе</span>
                              {portMode === "stealth443" && (
                                <span
                                  style={{
                                    marginLeft: "auto",
                                    fontSize: "11px",
                                    padding: "2px 8px",
                                    borderRadius: "10px",
                                    background: "rgba(96, 205, 255, 0.12)",
                                    color: "var(--win-accent)",
                                    border: "1px solid rgba(96, 205, 255, 0.25)",
                                  }}
                                >
                                  eGames Stealth 443
                                </span>
                              )}
                            </div>

                            {deployedAdminUser && (
                              <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: "10px", fontSize: "12px" }}>
                                <span style={{ color: "var(--win-text-secondary)" }}>Логин суперадминистратора:</span>
                                <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
                                  <code style={{ background: "var(--win-chip-bg)", padding: "2px 8px", borderRadius: "4px", fontWeight: 600 }}>
                                    {deployedAdminUser}
                                  </code>
                                  <button
                                    className="win-button"
                                    style={{ height: "24px", padding: "0 8px", fontSize: "11px" }}
                                    onClick={() => handleCopy(deployedAdminUser, "Логин", "cred-user")}
                                  >
                                    {copiedKey === "cred-user" ? (
                                      <span style={{ color: "#34d399", fontWeight: 600 }}>Скопировано!</span>
                                    ) : (
                                      <>
                                        <Copy size={11} />
                                        <span>Копировать</span>
                                      </>
                                    )}
                                  </button>
                                </div>
                              </div>
                            )}

                            {deployedAdminPass && (
                              <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: "10px", fontSize: "12px" }}>
                                <span style={{ color: "var(--win-text-secondary)" }}>Пароль администратора:</span>
                                <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
                                  <code style={{ background: "var(--win-chip-bg)", padding: "2px 8px", borderRadius: "4px", fontWeight: 600 }}>
                                    {showDeployedPass ? deployedAdminPass : "••••••••••••"}
                                  </code>
                                  <button
                                    className="win-button"
                                    style={{ height: "24px", padding: "0 8px", fontSize: "11px" }}
                                    onClick={() => setShowDeployedPass(!showDeployedPass)}
                                    title={showDeployedPass ? "Скрыть пароль" : "Показать пароль"}
                                  >
                                    {showDeployedPass ? <EyeOff size={11} /> : <Eye size={11} />}
                                  </button>
                                  <button
                                    className="win-button"
                                    style={{ height: "24px", padding: "0 8px", fontSize: "11px" }}
                                    onClick={() => handleCopy(deployedAdminPass, "Пароль", "cred-pass")}
                                  >
                                    {copiedKey === "cred-pass" ? (
                                      <span style={{ color: "#34d399", fontWeight: 600 }}>Скопировано!</span>
                                    ) : (
                                      <>
                                        <Copy size={11} />
                                        <span>Копировать</span>
                                      </>
                                    )}
                                  </button>
                                </div>
                              </div>
                            )}

                            {deployedSubUrl && (
                              <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: "10px", fontSize: "12px", borderTop: "1px solid var(--win-card-row-border)", paddingTop: "8px" }}>
                                <span style={{ color: "var(--win-text-secondary)" }}>Страница клиентских подписок:</span>
                                <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
                                  <code style={{ background: "var(--win-chip-bg)", padding: "2px 8px", borderRadius: "4px", maxWidth: "260px", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                                    {deployedSubUrl}
                                  </code>
                                  <button
                                    className="win-button"
                                    style={{ height: "24px", padding: "0 8px", fontSize: "11px" }}
                                    onClick={() => handleCopy(deployedSubUrl, "Ссылка на подписку", "cred-sub")}
                                  >
                                    {copiedKey === "cred-sub" ? (
                                      <span style={{ color: "#34d399", fontWeight: 600 }}>Скопировано!</span>
                                    ) : (
                                      <>
                                        <Copy size={11} />
                                        <span>Копировать</span>
                                      </>
                                    )}
                                  </button>
                                </div>
                              </div>
                            )}
                          </div>
                        )}

                        {deployedAdminMode === "manual" && preset !== "node-only" && (
                          <div
                            style={{
                              margin: "12px 0",
                              padding: "14px 16px",
                              background: "rgba(96, 205, 255, 0.08)",
                              border: "1px solid rgba(96, 205, 255, 0.25)",
                              borderRadius: "8px",
                              display: "flex",
                              flexDirection: "column",
                              gap: "10px",
                            }}
                          >
                            <div style={{ fontSize: "12.5px", fontWeight: 600, color: "var(--win-accent)", display: "flex", alignItems: "center", gap: "6px" }}>
                              <User size={14} />
                              <span>Регистрация суперадминистратора в веб-панели</span>
                            </div>
                            <div style={{ fontSize: "12px", color: "var(--win-text-secondary)", lineHeight: "1.5" }}>
                              Требуется завершить регистрацию: нажмите кнопку «Открыть панель управления в браузере» ниже и укажите логин и надежный пароль в открывшейся форме первоначальной настройки Remnawave.
                            </div>
                            {deployedSubUrl && (
                              <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: "10px", fontSize: "12px", borderTop: "1px solid var(--win-card-row-border)", paddingTop: "8px" }}>
                                <span style={{ color: "var(--win-text-secondary)" }}>Страница клиентских подписок:</span>
                                <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
                                  <code style={{ background: "var(--win-chip-bg)", padding: "2px 8px", borderRadius: "4px", maxWidth: "260px", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                                    {deployedSubUrl}
                                  </code>
                                  <button
                                    className="win-button"
                                    style={{ height: "24px", padding: "0 8px", fontSize: "11px" }}
                                    onClick={() => handleCopy(deployedSubUrl, "Ссылка на подписку", "cred-sub")}
                                  >
                                    {copiedKey === "cred-sub" ? (
                                      <span style={{ color: "#34d399", fontWeight: 600 }}>Скопировано!</span>
                                    ) : (
                                      <>
                                        <Copy size={11} />
                                        <span>Копировать</span>
                                      </>
                                    )}
                                  </button>
                                </div>
                              </div>
                            )}
                          </div>
                        )}



                        {preset === "node-only" && (
                          <div
                            style={{
                              margin: "12px 0",
                              padding: "14px 16px",
                              background: "var(--win-fill-card-secondary)",
                              border: "1px solid var(--win-stroke-card)",
                              borderRadius: "8px",
                              display: "flex",
                              flexDirection: "column",
                              gap: "10px",
                            }}
                          >
                            <div style={{ fontSize: "12.5px", fontWeight: 600, color: "var(--win-accent)", display: "flex", alignItems: "center", gap: "6px" }}>
                              <Shield size={14} />
                              <span>Параметры развернутой Reality Ноды</span>
                              <span
                                style={{
                                  marginLeft: "auto",
                                  fontSize: "11px",
                                  padding: "2px 8px",
                                  borderRadius: "10px",
                                  background: "rgba(16, 185, 129, 0.12)",
                                  color: "#34d399",
                                  border: "1px solid rgba(16, 185, 129, 0.25)",
                                }}
                              >
                                Активна и защищена
                              </span>
                            </div>
                            {nodeDecoyDomain && (
                              <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: "10px", fontSize: "12px" }}>
                                <span style={{ color: "var(--win-text-secondary)" }}>Decoy-домен маскировки (SNI):</span>
                                <code style={{ background: "var(--win-chip-bg)", padding: "2px 8px", borderRadius: "4px", fontWeight: 600 }}>
                                  {nodeDecoyDomain}
                                </code>
                              </div>
                            )}
                            <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: "10px", fontSize: "12px" }}>
                              <span style={{ color: "var(--win-text-secondary)" }}>Сервер ноды (IP / Порт):</span>
                              <code style={{ background: "var(--win-chip-bg)", padding: "2px 8px", borderRadius: "4px", fontWeight: 600 }}>
                                {sshHost || "127.0.0.1"}:{nodePort || 2222}
                              </code>
                            </div>
                            {xrayCoreType && (
                              <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: "10px", fontSize: "12px" }}>
                                <span style={{ color: "var(--win-text-secondary)" }}>Ядро Xray:</span>
                                <span style={{ fontSize: "12px", color: "var(--win-text-primary)", fontWeight: 500 }}>
                                  {xrayCoreType === "jolymmiles" ? "Jolymmiles Anti-TSPU" : "XTLS / Xray-core"}
                                </span>
                              </div>
                            )}
                          </div>
                        )}

                        <div className="hero-actions-row">

                          {deployedWebUrl && preset !== "node-only" && (

                            <button

                              className="win-button primary"

                              onClick={() => handleOpenBrowser(deployedWebUrl)}

                            >

                              <ExternalLink size={14} />

                              <span>Открыть панель управления в браузере</span>

                            </button>

                          )}

                          <button

                            className="win-button"

                            onClick={() => {

                              setActiveTab("servers");

                              loadServers();

                            }}

                          >

                            <Server size={14} />

                            <span>Перейти в «Мои серверы»</span>

                          </button>

                        </div>

                      </div>

                    )}



                    {/* Direct Windows Terminal (Unified Fluent Log Viewer) */}

                    {(logs.length > 0 || isDeploying) && (
                      <div className="win-terminal-window">
                        <div className="win-terminal-titlebar">
                          <div className="terminal-titlebar-host">
                            <TerminalIcon size={13} style={{ color: "var(--win-accent)", flexShrink: 0 }} />
                            <span className="terminal-host-text">
                              {deployMode === "remote" ? `${sshUser}@${sshHost || "vps"}` : "local-generator"}
                            </span>
                            
                          </div>

                          <div className="terminal-segmented-control">
                            <button
                              type="button"
                              className={`terminal-segment-btn ${logFilter === "all" ? "active" : ""}`}
                              onClick={() => setLogFilter("all")}
                            >
                              Все ({logs.length})
                            </button>
                            <button
                              type="button"
                              className={`terminal-segment-btn ${logFilter === "error" ? "active danger" : ""}`}
                              onClick={() => setLogFilter("error")}
                            >
                              Ошибки {terminalErrorCount > 0 && <span className="terminal-segment-badge">{terminalErrorCount}</span>}
                            </button>
                            <button
                              type="button"
                              className={`terminal-segment-btn ${logFilter === "info" ? "active" : ""}`}
                              onClick={() => setLogFilter("info")}
                            >
                              Инфо ({logs.length - terminalErrorCount})
                            </button>
                          </div>

                            <div className="terminal-search-box">
                              <Search size={12} className="terminal-search-icon" />
                              <input
                                type="text"
                                className="terminal-search-input"
                                placeholder="Поиск..."
                                value={logSearch}
                                onChange={(e) => setLogSearch(e.target.value)}
                              />
                              {logSearch && (
                                <button
                                  type="button"
                                  className="terminal-search-clear"
                                  onClick={() => setLogSearch("")}
                                  title="Очистить поиск"
                                >
                                  <X size={11} />
                                </button>
                              )}
                            </div>

                            <div className="terminal-actions">
                              <button
                                type="button"
                                className="terminal-mini-btn"
                                onClick={handleCopyLogs}
                                title="Скопировать логи"
                              >
                                {copiedLogs ? <Check size={12} style={{ color: "#34d399" }} /> : <Copy size={12} />}
                                <span>{copiedLogs ? "Скопировано" : "Копировать"}</span>
                              </button>

                              <button
                                type="button"
                                className="terminal-mini-btn"
                                onClick={() => setLogs([])}
                                title="Очистить терминал"
                              >
                                <RotateCcw size={12} />
                                <span>Очистить</span>
                              </button>
                            </div>
                          </div>

                          <div className="win-terminal-body">
                            {filteredLogs.length === 0 ? (
                              <div style={{ color: "var(--win-text-secondary)", fontSize: "12px", padding: "12px", textAlign: "center" }}>
                                {logSearch ? "Строк по вашему запросу не найдено" : "Нет записей в выбранном фильтре"}
                              </div>
                            ) : (
                              filteredLogs.map((log, idx) => {
                                const lineStatus = log.isError ? "error" : log.status === "stderr" ? "info" : log.status;
                                return (
                                  <div key={idx} className={`terminal-line ${lineStatus}`}>
                                    <span>{log.message}</span>
                                  </div>
                                );
                              })
                            )}

                            {isDeploying && (
                              <div className="terminal-line step">
                                <span>Выполняется команда...</span>
                              </div>
                            )}
                            <div ref={terminalEndRef} />
                          </div>

                          <div className="win-terminal-statusbar">
                            <span>UTF-8</span>
                            <span>
                              {deployFinished
                                ? deploySuccess
                                  ? "Готово (Успешно)"
                                  : "Завершено с ошибками"
                                : isDeploying
                                ? "Выполняется..."
                                : "Ожидание"}
                            </span>
                            <span>Показано: {filteredLogs.length} из {logs.length}</span>
                          </div>
                        </div>
                    )}

                  </div>

                )}

              </div>

            )}



            {/* ================================================================

                TAB: SAVED SERVERS

                ================================================================ */}

            {activeTab === "servers" && (

              <div key="tab-servers" className="tab-enter">

                <div className="catalog-page-header">
                  <div className="catalog-header-text">
                    <h1 className="page-title">Мои серверы</h1>
                    <p className="page-desc">
                      Управление стеком Remnawave, мониторинг системных ресурсов и логи контейнеров в реальном времени.
                    </p>
                  </div>

                  <div className="catalog-toolbar-group">
                    <button
                      className="win-button"
                      disabled={Object.values(serverLoading).some(Boolean)}
                      onClick={handleInspectAllServers}
                      title="Опросить состояние всех сохраненных серверов (параллельно)"
                    >
                      <RefreshCw size={13} className={Object.values(serverLoading).some(Boolean) ? "animate-spin" : ""} />
                      <span>Обновить все</span>
                    </button>

                    <button
                      className="win-button"
                      disabled={savedServers.length === 0}
                      onClick={toggleAllCollapse}
                      title={areAllCollapsed ? "Развернуть все серверные карточки" : "Свернуть все серверные карточки в компактный вид (Accordion)"}
                    >
                      {areAllCollapsed ? <ChevronsUpDown size={13} /> : <ChevronsDownUp size={13} />}
                      <span>{areAllCollapsed ? "Развернуть все" : "Свернуть все"}</span>
                    </button>

                    <div className="catalog-toolbar-divider" />

                    <button
                      className="win-button"
                      onClick={() => {
                        setEditServerModal({
                          server: new types.SavedServer({
                            id: "srv_" + Date.now().toString(36),
                            label: "",
                            host: "",
                            port: 22,
                            username: "root",
                            web_url: "",
                            role: "master",
                            last_seen: "Только что добавлен",
                          }),
                          name: "",
                          host: "",
                          port: 22,
                          user: "root",
                          authMethod: "password",
                          password: "",
                          role: "master",
                          isExternalMaster: true,
                          webUrl: "",
                          locationCode: "",
                          decoyDomain: "",
                        });
                      }}
                      title="Добавить уже работающий сервер (Master или Node) вручную в каталог"
                    >
                      <Plus size={13} />
                      <span>Добавить вручную</span>
                    </button>

                    <button className="win-button primary" onClick={() => setActiveTab("wizard")}>
                      <Plus size={14} />
                      <span>Развернуть новый сервер</span>
                    </button>
                  </div>
                </div>



                {loadingServersList ? (

                  <div className="servers-container">

                    <ServerCardSkeleton />

                    <ServerCardSkeleton />

                  </div>

                ) : savedServers.length === 0 ? (

                  <div

                    style={{

                      padding: "48px 24px",

                      display: "flex",

                      flexDirection: "column",

                      alignItems: "center",

                      justifyContent: "center",

                      textAlign: "center",

                      background: "var(--win-card-bg)",

                      border: "1px solid var(--win-card-stroke)",

                      borderRadius: "8px",

                    }}

                  >

                    <Server size={36} style={{ color: "var(--win-text-tertiary)", marginBottom: "12px" }} />

                    <div style={{ fontSize: "15px", fontWeight: 500, marginBottom: "4px" }}>

                      Список серверов пуст

                    </div>

                    <div style={{ fontSize: "13px", color: "var(--win-text-tertiary)", marginBottom: "16px" }}>

                      После первого успешного развертывания через мастер сервер автоматически появится здесь.

                    </div>

                    <div style={{ display: "flex", gap: "10px", marginTop: "8px", flexWrap: "wrap", justifyContent: "center" }}>
                      <button className="win-button primary" onClick={() => setActiveTab("wizard")}>
                        Перейти в Мастер развертывания
                      </button>
                      <button
                        className="win-button"
                        onClick={() => {
                          setEditServerModal({
                            server: new types.SavedServer({
                              id: "srv_" + Date.now().toString(36),
                              label: "",
                              host: "",
                              port: 22,
                              username: "root",
                              web_url: "",
                              role: "master",
                              last_seen: "Только что добавлен",
                            }),
                            name: "",
                            host: "",
                            port: 22,
                            user: "root",
                            authMethod: "password",
                            password: "",
                            role: "master",
                            isExternalMaster: true,
                            webUrl: "",
                            locationCode: "",
                            decoyDomain: "",
                          });
                        }}
                      >
                        <Server size={13} />
                        <span>Добавить существующий сервер</span>
                      </button>
                    </div>

                  </div>

                ) : (

                  <div className="servers-container">

                    {(() => {
                      const renderServerCard = (srv: types.SavedServer, isChild: boolean = false, childCount?: number, cardIndex: number = 0) => (
                        <ServerCard
                          key={srv.id}
                          srv={srv}
                          isChild={isChild}
                          childCount={childCount}
                          cardIndex={cardIndex}
                          status={serverStatus[srv.id]}
                          isLoading={serverLoading[srv.id]}
                          isCollapsed={Boolean(collapsedServers[srv.id])}
                          activeServerMenu={activeServerMenu}
                          subTab={serverSubTab[srv.id] || "overview"}
                          actionLoading={actionLoading}
                          cleaningDiskServerId={cleaningDiskServerId}
                          savedServers={savedServers}
                          copiedKey={copiedKey}
                          toggleServerCollapse={toggleServerCollapse}
                          setActiveServerMenu={setActiveServerMenu}
                          setServerSubTab={setServerSubTab}
                          handleCopy={handleCopy}
                          handleOpenBrowser={handleOpenBrowser}
                          handleOpenAudit={handleOpenAudit}
                          handleInspectServer={handleInspectServer}
                          handleAddNodeToPanel={handleAddNodeToPanel}
                          handleRestartAll={handleRestartAll}
                          handleUpdateStack={handleUpdateStack}
                          handleOptimizeBBR={handleOptimizeBBR}
                          handleTogglePerformance={handleTogglePerformance}
                          setNodeModal={setNodeModal}
                          handleOpenLogsModal={handleOpenLogsModal}
                          handleRestartContainer={handleRestartContainer}
                          handleOpenIPCheckModal={handleOpenIPCheckModal}
                          handleOpenVPNProbeModal={handleOpenVPNProbeModal}
                          handleCleanServerDisk={handleCleanServerDisk}
                          handleStartEditServer={handleStartEditServer}
                          setReinstallModal={setReinstallModal}
                          setUninstallModal={setUninstallModal}
                          handleDeleteServer={handleDeleteServer}
                          handleCreateBackup={handleCreateBackup}
                          handleDownloadBackup={handleDownloadBackup}
                          setRestoreModal={setRestoreModal}
                          handleFixDBSync={handleFixDBSync}
                          handleInstallWARP={handleInstallWARP}
                          handleResetSuperadmin={handleResetSuperadmin}
                          handleToggleAntiAbuse={handleToggleAntiAbuse}
                        />
                      );

                    let cardCounter = 0;
                    return (
                      <>
                        {masterServers.map((master) => {
                          const children = childNodesByMaster.get(master.id) || [];
                          const masterIdx = cardCounter++;
                          return (
                            <div key={master.id} className="server-tree-group">
                              {renderServerCard(master, false, children.length, masterIdx)}
                              {children.length > 0 && (
                                <div className="server-tree-children">
                                  {children.map((child) => renderServerCard(child, true, undefined, cardCounter++))}
                                </div>
                              )}
                            </div>
                          );
                        })}
                        {standaloneServers.map((srv) => renderServerCard(srv, false, undefined, cardCounter++))}
                      </>
                    );
                  })()}

                  </div>

                )}

              </div>

            )}



            {/* ================================================================

                TAB: ABOUT

                ================================================================ */}

            {activeTab === "about" && (
              <div key="tab-about" className="tab-enter">
                <div className="page-header">
                  <h1 className="page-title">О программе</h1>
                  <p className="page-desc">
                    Нативное десктопное приложение для управления инфраструктурой Remnawave на Windows.
                  </p>
                </div>

                {/* Hero Header Card */}
                <div
                  style={{
                    background: "var(--win-card-bg)",
                    border: "1px solid var(--win-card-stroke)",
                    borderRadius: "10px",
                    padding: "20px 24px",
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "space-between",
                    flexWrap: "wrap",
                    gap: "16px",
                    marginBottom: "20px",
                    boxShadow: "var(--win-card-shadow)",
                  }}
                >
                  <div style={{ display: "flex", alignItems: "center", gap: "16px" }}>
                    <div
                      style={{
                        width: "48px",
                        height: "48px",
                        borderRadius: "10px",
                        background: "linear-gradient(135deg, var(--win-accent-primary), #0284c7)",
                        display: "flex",
                        alignItems: "center",
                        justifyContent: "center",
                        color: "#ffffff",
                        boxShadow: "0 4px 14px rgba(0, 103, 192, 0.4)",
                        flexShrink: 0,
                      }}
                    >
                      <Zap size={24} />
                    </div>
                    <div>
                      <div style={{ display: "flex", alignItems: "center", gap: "10px" }}>
                        <span style={{ fontSize: "17px", fontWeight: 600, color: "var(--win-text-primary)" }}>
                          Remnawave Deployment & Manager Suite
                        </span>
                        <span className="preset-badge" style={{ background: "rgba(16, 185, 129, 0.15)", color: "#10b981", border: "1px solid rgba(16, 185, 129, 0.3)" }}>
                          v2.0.0 Stable
                        </span>
                      </div>
                      <div style={{ fontSize: "12.5px", color: "var(--win-text-secondary)", marginTop: "3px" }}>
                        WinUI 3 Fluent Design Edition · Windows x64
                      </div>
                    </div>
                  </div>

                  <div style={{ display: "flex", gap: "10px", alignItems: "center" }}>
                    <button
                      type="button"
                      className="win-button"
                      disabled={checkingUpdate}
                      onClick={handleCheckAppUpdate}
                      style={{ height: "32px", fontSize: "12.5px" }}
                    >
                      <RefreshCw size={13} className={checkingUpdate ? "animate-spin" : ""} />
                      <span>{checkingUpdate ? "Проверка..." : "Проверить обновления"}</span>
                    </button>
                  </div>
                </div>

                {updateInfo && (
                  <div style={{ marginBottom: "20px" }}>
                    <InfoBar
                      severity={updateInfo.has_update ? "info" : updateInfo.error ? "error" : "success"}
                      title={
                        updateInfo.has_update
                          ? `Доступна новая версия: ${updateInfo.latest_version}`
                          : updateInfo.error
                          ? "Ошибка проверки обновлений"
                          : "У вас установлена последняя версия"
                      }
                      message={
                        updateInfo.has_update ? (
                          <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginTop: "4px" }}>
                            <span>{updateInfo.release_title || "Вышло свежее обновление на GitHub."}</span>
                            {updateInfo.release_url && (
                              <button
                                type="button"
                                className="win-button"
                                style={{ height: "26px", fontSize: "11.5px" }}
                                onClick={() => OpenBrowserURL(updateInfo.release_url)}
                              >
                                <ExternalLink size={12} />
                                <span>Открыть релиз</span>
                              </button>
                            )}
                          </div>
                        ) : updateInfo.error ? (
                          updateInfo.error
                        ) : (
                          "Все компоненты системы и драйверы актуальны."
                        )
                      }
                    />
                  </div>
                )}

                {/* Specification Group */}
                <div className="settings-group-title">Архитектура и безопасность ядра</div>
                <div className="settings-card-group" style={{ marginBottom: "20px" }}>
                  <div className="settings-card-row">
                    <div className="settings-card-info">
                      <div className="settings-card-icon">
                        <Cpu size={18} />
                      </div>
                      <div className="settings-card-text">
                        <div className="settings-card-title">Архитектура ядра</div>
                        <div className="settings-card-desc">
                          Go 1.27 + Wails v2 + React 19 + TypeScript + WebView2 Mica Composition
                        </div>
                      </div>
                    </div>
                    <div className="settings-card-action">
                      <button
                        type="button"
                        className="win-button"
                        style={{ height: "28px", fontSize: "11.5px" }}
                        onClick={handleCopySystemInfo}
                        title="Скопировать сведения о системе"
                      >
                        <Copy size={12} />
                        <span>Копировать инфо</span>
                      </button>
                    </div>
                  </div>

                  <div className="settings-card-row">
                    <div className="settings-card-info">
                      <div className="settings-card-icon">
                        <Package size={18} />
                      </div>
                      <div className="settings-card-text">
                        <div className="settings-card-title">Полная автономность</div>
                        <div className="settings-card-desc">
                          Автономный компилируемый бинарник. Все скрипты, шаблоны Nginx/Xray и SSH-драйверы встроены в EXE файл.
                        </div>
                      </div>
                    </div>
                  </div>

                  <div className="settings-card-row">
                    <div className="settings-card-info">
                      <div className="settings-card-icon">
                        {vaultStatus?.enabled ? (
                          <Lock size={18} style={{ color: "var(--win-accent-primary)" }} />
                        ) : (
                          <ShieldCheck size={18} style={{ color: "#10b981" }} />
                        )}
                      </div>
                      <div className="settings-card-text">
                        <div className="settings-card-title">
                          Шифрование локальных секретов: {vaultStatus?.enabled ? "Мастер-пароль (Argon2id + AES-256-GCM)" : "Windows DPAPI (CryptProtectData)"}
                        </div>
                        <div className="settings-card-desc">
                          {vaultStatus?.enabled
                            ? `База серверов и API-токены зашифрованы мастер-паролем. Автоблокировка: ${vaultStatus.auto_lock_minutes > 0 ? `${vaultStatus.auto_lock_minutes} мин.` : "отключена"}.`
                            : "Пароли серверов, ключи и токены защищены системным аппаратным шифрованием Windows DPAPI."}
                        </div>
                      </div>
                    </div>
                    <div className="settings-card-action">
                      <button
                        type="button"
                        className="win-button"
                        style={{ height: "28px", fontSize: "11.5px" }}
                        onClick={() => setVaultSettingsOpen(true)}
                      >
                        <KeyRound size={12} />
                        <span>Настройки защиты</span>
                      </button>
                    </div>
                  </div>
                </div>

                {/* Links & Community Group */}
                <div className="settings-group-title">Ресурсы и документация</div>
                <div className="settings-card-group">
                  <div className="settings-card-row">
                    <div className="settings-card-info">
                      <div className="settings-card-icon">
                        <GitBranch size={18} />
                      </div>
                      <div className="settings-card-text">
                        <div className="settings-card-title">Репозиторий GitHub</div>
                        <div className="settings-card-desc">
                          Исходный код проекта, список изменений, сообщение об ошибках
                        </div>
                      </div>
                    </div>
                    <div className="settings-card-action">
                      <button
                        type="button"
                        className="win-button"
                        style={{ height: "28px", fontSize: "11.5px" }}
                        onClick={() => OpenBrowserURL("https://github.com/XZ-LITE/remna-ui")}
                      >
                        <ExternalLink size={12} />
                        <span>Открыть GitHub</span>
                      </button>
                    </div>
                  </div>

                  <div className="settings-card-row">
                    <div className="settings-card-info">
                      <div className="settings-card-icon">
                        <BookOpen size={18} />
                      </div>
                      <div className="settings-card-text">
                        <div className="settings-card-title">Официальная документация Remnawave</div>
                        <div className="settings-card-desc">
                          Руководство пользователя, протоколы Xray, API и клиентские приложения
                        </div>
                      </div>
                    </div>
                    <div className="settings-card-action">
                      <button
                        type="button"
                        className="win-button"
                        style={{ height: "28px", fontSize: "11.5px" }}
                        onClick={() => OpenBrowserURL("https://docs.rw/overview/quick-start/")}
                      >
                        <ExternalLink size={12} />
                        <span>Документация</span>
                      </button>
                    </div>
                  </div>

                  <div className="settings-card-row">
                    <div className="settings-card-info">
                      <div className="settings-card-icon">
                        <ShieldCheck size={18} />
                      </div>
                      <div className="settings-card-text">
                        <div className="settings-card-title">Лицензия проекта</div>
                        <div className="settings-card-desc">
                          Свободное программное обеспечение с открытым исходным кодом (MIT License)
                        </div>
                      </div>
                    </div>
                    <div className="settings-card-action">
                      <span className="preset-badge">MIT</span>
                    </div>
                  </div>
                </div>
              </div>
            )}

          </div>

        </div>



        {/* ====================================================================

            Bottom Action Bar (Navigation Controls)

            ==================================================================== */}

        {activeTab === "wizard" && (

          <footer className="bottom-action-bar">

            <div className="bottom-action-left">

              {step > 1 && (
                <button className="win-button" onClick={() => changeStep(step - 1)}>
                  Назад
                </button>
              )}
            </div>

            <div className="bottom-action-right">
              {step < 3 ? (
                <button
                  className="win-button primary"
                  onClick={() => {
                    changeStep(step + 1);
                  }}
                >
                  <span>Далее: {step === 1 ? (preset === "node-only" ? "Параметры ноды" : "Домен и защита") : "Развертывание"}</span>
                  <ChevronRight size={15} />
                </button>

              ) : (

                <button

                  className="win-button primary"

                  disabled={isDeploying || (deployMode === "remote" && (!sshHost || (!sshPass && !sshKey)))}

                  onClick={handleStart}

                >

                  <Play size={14} />

                  <span>{deployMode === "remote" ? "Установить на VPS" : "Сгенерировать"}</span>

                </button>

              )}

            </div>

          </footer>

        )}

      </main>



      {/* Fluent Acrylic Logs Modal Window */}

      {inspectModal && (

        <InspectLogsModal

          server={inspectModal.server}

          container={inspectModal.container}

          logs={inspectModal.logs}

          autoRefresh={inspectModal.autoRefresh}

          onClose={() => setInspectModal(null)}

          onRefresh={async () => {

            try {

              const out = await GetContainerLogs(inspectModal.server, inspectModal.container);

              setInspectModal((prev) => (prev ? { ...prev, logs: out || "Логи пусты." } : null));

              addToast("info", "Логи обновлены");

            } catch (e) {

              console.error(e);

            }

          }}

          onToggleAutoRefresh={() =>

            setInspectModal((prev) =>

              prev ? { ...prev, autoRefresh: !prev.autoRefresh } : null

            )

          }
          onCopy={handleCopy}
        />

      )}
      {/* Modal: Reinstall Stack */}
      {reinstallModal && (
        <div className="win-modal-backdrop" onClick={() => setReinstallModal(null)}>
          <div className="win-modal-dialog" style={{ maxWidth: "520px" }} onClick={(e) => e.stopPropagation()}>
            <div className="win-modal-header">
              <div className="win-modal-title">
                <RotateCcw size={16} color="var(--win-accent)" />
                <span>Переустановка стека Remnawave</span>
                <span style={{ fontSize: "11px", color: "var(--win-text-tertiary)" }}>
                  ({reinstallModal.server.host})
                </span>
              </div>
              <button className="win-button" style={{ padding: "0 8px", height: "26px" }} onClick={() => setReinstallModal(null)}>
                <X size={14} />
              </button>
            </div>
            <div className="win-modal-body" style={{ display: "flex", flexDirection: "column", gap: "12px", padding: "16px 20px" }}>
              <p style={{ margin: 0, fontSize: "12.5px", color: "var(--win-text-secondary)" }}>
                Выберите подходящий вариант переустановки для сервера <strong>{reinstallModal.server.host}</strong>:
              </p>

              {/* Option 1: Soft */}
              <div
                style={{
                  padding: "14px",
                  borderRadius: "8px",
                  background: "var(--win-fill-card-secondary)",
                  border: "1px solid var(--win-stroke-card)",
                  cursor: "pointer",
                  transition: "all 0.15s ease",
                }}
                onClick={() => handleReinstallSoft(reinstallModal.server)}
              >
                <div style={{ display: "flex", alignItems: "center", gap: "8px", fontWeight: 600, color: "var(--win-text-primary)", fontSize: "13px" }}>
                  <Shield size={16} style={{ color: "#34d399" }} />
                  <span>Мягкая переустановка (С сохранением базы данных)</span>
                </div>
                <div style={{ marginTop: "6px", fontSize: "11.5px", color: "var(--win-text-secondary)", lineHeight: 1.4 }}>
                  Автоматически синхронизирует и перевыпускает токены служб (страницы подписок), обновляет конфигурации, загружает свежие образы и перезапускает контейнеры. База данных, клиенты и ключи будут полностью сохранены.
                </div>
              </div>

              {/* Option 2: Clean */}
              <div
                style={{
                  padding: "14px",
                  borderRadius: "8px",
                  background: "rgba(56, 189, 248, 0.06)",
                  border: "1px solid rgba(56, 189, 248, 0.22)",
                  cursor: "pointer",
                  transition: "all 0.15s ease",
                }}
                onClick={() => handleReinstallClean(reinstallModal.server)}
              >
                <div style={{ display: "flex", alignItems: "center", gap: "8px", fontWeight: 600, color: "#60cdff", fontSize: "13px" }}>
                  <Wand2 size={16} style={{ color: "#60cdff" }} />
                  <span>Чистая переустановка с нуля (Мастер деплоя)</span>
                </div>
                <div style={{ marginTop: "6px", fontSize: "11.5px", color: "var(--win-text-secondary)", lineHeight: 1.4 }}>
                  Удалит старый стек с сервера и откроет Мастер установки с уже заполненными SSH-реквизитами для установки на чистый VPS.
                </div>
              </div>
            </div>
            <div className="win-modal-footer">
              <button className="win-button" style={{ height: "28px" }} onClick={() => setReinstallModal(null)}>
                Отмена
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Modal: Uninstall & Wipe Stack from VPS */}
      {uninstallModal && (
        <div className="win-modal-backdrop" onClick={() => !uninstallModal.loading && setUninstallModal(null)}>
          <div className="win-modal-dialog" style={{ maxWidth: "500px" }} onClick={(e) => e.stopPropagation()}>
            <div className="win-modal-header">
              <div className="win-modal-title" style={{ color: "#fb7185" }}>
                <AlertOctagon size={16} color="#fb7185" />
                <span>Очистка VPS и удаление Remnawave</span>
              </div>
              {!uninstallModal.loading && (
                <button className="win-button" style={{ padding: "0 8px", height: "26px" }} onClick={() => setUninstallModal(null)}>
                  <X size={14} />
                </button>
              )}
            </div>
            <div className="win-modal-body" style={{ display: "flex", flexDirection: "column", gap: "12px", padding: "16px 20px" }}>
              <div style={{ background: "rgba(244, 63, 94, 0.1)", border: "1px solid rgba(244, 63, 94, 0.25)", borderRadius: "6px", padding: "10px 12px", fontSize: "12px", color: "#fb7185", lineHeight: 1.45 }}>
                Внимание! С сервера <strong>{uninstallModal.server.host}</strong> будут принудительно остановлены и удалены все Docker-контейнеры, каталоги /opt/remnawave и правила фаервола.
              </div>
              <p style={{ margin: 0, fontSize: "12px", color: "var(--win-text-secondary)", lineHeight: 1.45 }}>
                Операционная система Linux (Ubuntu/Debian) останется абсолютно чистой. Вам не придется переустанавливать ОС у хостинга.
              </p>
              <div style={{ display: "flex", flexDirection: "column", gap: "6px" }}>
                <label style={{ fontSize: "12px", color: "var(--win-text-primary)", fontWeight: 500 }}>
                  Для подтверждения введите IP/хост сервера: <code>{uninstallModal.server.host}</code>
                </label>
                <input
                  type="text"
                  className="win-input"
                  style={{ height: "30px", fontSize: "12px" }}
                  placeholder={uninstallModal.server.host}
                  value={uninstallModal.confirmHost}
                  onChange={(e) => setUninstallModal({ ...uninstallModal, confirmHost: e.target.value })}
                  disabled={uninstallModal.loading}
                />
              </div>
            </div>
            <div className="win-modal-footer">
              <button className="win-button" style={{ height: "28px" }} disabled={uninstallModal.loading} onClick={() => setUninstallModal(null)}>
                Отмена
              </button>
              <button
                className="win-button"
                style={{ height: "28px", background: "#ef4444", color: "#fff", borderColor: "#dc2626", fontWeight: 600 }}
                disabled={uninstallModal.confirmHost !== uninstallModal.server.host || uninstallModal.loading}
                onClick={handleUninstallServer}
              >
                {uninstallModal.loading ? (
                  <>
                    <RefreshCw size={12} className="animate-spin" />
                    <span>Очистка VPS...</span>
                  </>
                ) : (
                  <>
                    <Trash2 size={12} />
                    <span>Очистить сервер и удалить</span>
                  </>
                )}
              </button>
            </div>
          </div>
        </div>
      )}




      {/* Modal: Attach Node Secret Key */}

      {nodeModal && (

        <NodeSecretModal

          server={nodeModal.server}

          secretKey={nodeModal.secretKey}

          loading={nodeModal.loading}

          onChangeSecretKey={(val) => setNodeModal((prev) => (prev ? { ...prev, secretKey: val } : null))}

          onClose={() => setNodeModal(null)}

          onSubmit={handleAttachNode}

          onCopy={handleCopy}

          onToast={addToast}

        />

      )}



      {/* Modal: Edit Server Settings */}

      {editServerModal && (

        <EditServerModal

          data={editServerModal}

          availableMasters={savedServers.filter((s) => s.role === "master" || Boolean(s.web_url))}

          onChange={(updated) => setEditServerModal(updated)}

          onClose={() => setEditServerModal(null)}

          onSave={handleSaveEditedServer}

          onSelectKeyFile={SelectSSHKeyFile}

          onTestSSH={(h, p, u, pass, key) => TestSSH(h, p, u, pass, key)}

          onCheckIPBlock={(h, p) => CheckIPBlock(h, p)}

        />

      )}



      {/* Modal: Restore Backup Confirmation */}

      {restoreModal && (

        <RestoreBackupModal
          server={restoreModal.server}
          fileName={restoreModal.fileName}
          loading={Boolean(actionLoading[`restore-${restoreModal.server.id}`])}
          onClose={() => setRestoreModal(null)}
          onConfirm={(scope, createSafetyBackup) =>
            handleRestoreBackup(restoreModal.server, restoreModal.fileName, scope, createSafetyBackup)
          }
        />

      )}



      {/* Modal: Anti-Probing Security Audit */}
      {auditModal.server && (
        <AntiProbingAuditModal
          server={auditModal.server}
          parentMasterIp={
            savedServers.find((s) => s.id === auditModal.server?.parent_master_id)?.host ||
            (auditModal.server?.role === "node" ? savedServers.find((s) => s.role === "master" || (s as any).is_external_master || Boolean(s.web_url))?.host : "") ||
            ""
          }
          masterServers={savedServers.filter((s) => s.id !== auditModal.server?.id && (s.role === "master" || (s as any).is_external_master || Boolean(s.web_url)))}
          audit={auditModal.audit}
          loading={auditModal.loading}
          applying={auditModal.applying}
          onClose={() => setAuditModal({ server: null, loading: false, applying: false, audit: null })}
          onRefresh={(panelIp) => handleOpenAudit(auditModal.server!, panelIp)}
          onApplyProtection={handleApplyProtection}
          onToast={addToast}
        />
      )}

      {/* Modal: IP Availability & TSPU Check */}
      {ipCheckModal && (
        <IPBlockModal
          server={ipCheckModal.server}
          result={ipCheckModal.result}
          loading={ipCheckModal.loading}
          onClose={() => setIpCheckModal(null)}
          onRefresh={handleRefreshIPCheckModal}
        />
      )}

      {/* Modal: Synthetic VPN Traffic & TSPU Probe */}
      {vpnProbeModal && (
        <VPNProbeModal
          server={vpnProbeModal.server}
          result={vpnProbeModal.result}
          loading={vpnProbeModal.loading}
          onClose={() => setVpnProbeModal(null)}
          onRefresh={(customVless) => handleOpenVPNProbeModal(vpnProbeModal.server, customVless)}
        />
      )}

      {/* Modal: Master Password Vault Unlock */}
      <VaultUnlockModal
        isOpen={vaultUnlockOpen}
        onUnlock={handleUnlockVaultAction}
        onToast={addToast}
      />

      {/* Modal: Master Password & Security Settings */}
      <VaultSettingsModal
        isOpen={vaultSettingsOpen}
        vaultStatus={vaultStatus}
        onClose={() => setVaultSettingsOpen(false)}
        onSetMasterPassword={handleSetMasterPasswordAction}
        onDisableMasterPassword={handleDisableMasterPasswordAction}
        onSetAutoLockMinutes={handleSetAutoLockMinutesAction}
        onToast={addToast}
        onRefreshStatus={checkVaultStatus}
      />

    </div>

  );

}



export default App;

