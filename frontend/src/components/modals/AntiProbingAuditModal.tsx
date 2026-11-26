import React, { useState, useEffect } from "react";
import {
  ShieldCheck,
  ShieldAlert,
  Shield,
  CheckCircle2,
  AlertTriangle,
  RefreshCw,
  Lock,
  Globe,
  Radio,
  X,
  Zap,
  Cpu,
  Server,
  Activity,
  Network,
  Sliders,
  Sparkles,
  Info,
} from "lucide-react";
import * as types from "../../../bindings/remna-ui/internal/types/models";
import { WinModal } from "../ui/WinModal";
import { WinTooltip } from "../ui/WinTooltip";

interface AntiProbingAuditModalProps {
  server: types.SavedServer;
  parentMasterIp?: string;
  masterServers?: types.SavedServer[];
  audit: types.NodeSecurityAudit | null;
  loading: boolean;
  applying: boolean;
  onClose: () => void;
  onRefresh: (panelIp?: string) => void;
  onApplyProtection: (panelIp: string, hideNodePort: boolean, stealthDrop: boolean, enableBBR: boolean) => void;
  onToast: (type: "success" | "error" | "info", msg: string) => void;
}

export const AntiProbingAuditModal: React.FC<AntiProbingAuditModalProps> = ({
  server,
  parentMasterIp,
  masterServers = [],
  audit,
  loading,
  applying,
  onClose,
  onRefresh,
  onApplyProtection,
  onToast,
}) => {
  const [panelIpInput, setPanelIpInput] = useState<string>(parentMasterIp || "");
  const [enableHidePort, setEnableHidePort] = useState<boolean>(true);
  const [enableStealthDrop, setEnableStealthDrop] = useState<boolean>(true);
  const [enableBBR, setEnableBBR] = useState<boolean>(true);

  // If parentMasterIp updates and input was empty, fill it
  useEffect(() => {
    if (parentMasterIp && !panelIpInput) {
      setPanelIpInput(parentMasterIp);
    }
  }, [parentMasterIp]);

  const score = audit?.overall_score ?? 0;
  const scoreColor =
    score >= 90 ? "#10b981" : score >= 70 ? "#38bdf8" : score >= 45 ? "#f59e0b" : "#ef4444";
  const scoreBg =
    score >= 90
      ? "rgba(16, 185, 129, 0.12)"
      : score >= 70
      ? "rgba(56, 189, 248, 0.12)"
      : score >= 45
      ? "rgba(245, 158, 11, 0.12)"
      : "rgba(239, 68, 68, 0.12)";
  const scoreBadgeText =
    score >= 90
      ? "Максимальная защита (A+)"
      : score >= 70
      ? "Хорошая защита (A)"
      : score >= 45
      ? "Частичная защита (B)"
      : "Уязвимо для сканеров (D)";

  const isFullyProtected = score >= 90;

  return (
    <WinModal
      isOpen={true}
      onClose={onClose}
      maxWidth="640px"
      width="95%"
      maxHeight="88vh"
      preventBackdropClose={loading || applying}
      dialogStyle={{
        display: "flex",
        flexDirection: "column",
        overflow: "hidden",
      }}
    >
      {({ close }) => (
        <>
      {/* Header - Fixed at Top */}
        <div className="node-modal-header" style={{ flexShrink: 0, padding: "14px 18px" }}>
          <div className="node-modal-title-wrap">
            <div
              className="node-modal-icon"
              style={{
                background: scoreBg,
                color: scoreColor,
                width: "32px",
                height: "32px",
                borderRadius: "6px",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                flexShrink: 0,
              }}
            >
              <ShieldCheck size={18} />
            </div>
            <div>
              <div className="node-modal-title" style={{ fontSize: "14px", fontWeight: 600 }}>
                Сетевой экран и защита ноды от сканеров
              </div>
              <div className="node-modal-sub" style={{ fontSize: "11.5px" }}>
                Сервер:{" "}
                <span style={{ color: "var(--win-text-primary)", fontWeight: 600 }}>
                  {server.label || server.host}
                </span>{" "}
                ({server.host})
              </div>
            </div>
          </div>
          <WinTooltip content="Закрыть" shortcut="Esc">
            <button
              className="win-button"
              style={{ padding: "0 8px", height: "28px" }}
              onClick={() => !loading && !applying && close()}
              disabled={loading || applying}
            >
              <X size={14} />
            </button>
          </WinTooltip>
        </div>

        {/* Scrollable Body - Adapts cleanly to any window height */}
        <div
          className="win-modal-body"
          style={{
            flex: 1,
            minHeight: 0,
            overflowY: "auto",
            padding: "14px 18px",
            display: "flex",
            flexDirection: "column",
            gap: "12px",
          }}
        >
          {/* Score Hero Banner */}
          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: "14px",
              padding: "12px 16px",
              borderRadius: "8px",
              background: scoreBg,
              border: `1px solid ${scoreColor}40`,
              flexShrink: 0,
            }}
          >
            <div
              style={{
                width: "52px",
                height: "52px",
                borderRadius: "50%",
                background: "rgba(0, 0, 0, 0.4)",
                border: `2.5px solid ${scoreColor}`,
                display: "flex",
                flexDirection: "column",
                alignItems: "center",
                justifyContent: "center",
                flexShrink: 0,
              }}
            >
              <span style={{ fontSize: "15px", fontWeight: 700, color: scoreColor, lineHeight: 1 }}>
                {loading ? "..." : `${score}%`}
              </span>
              <span style={{ fontSize: "8.5px", color: "var(--win-text-tertiary)", marginTop: "2px" }}>
                Score
              </span>
            </div>

            <div style={{ flex: 1, minWidth: 0 }}>
              <div style={{ display: "flex", alignItems: "center", gap: "8px", marginBottom: "3px" }}>
                <span
                  style={{
                    fontSize: "11px",
                    fontWeight: 600,
                    padding: "1px 7px",
                    borderRadius: "8px",
                    background: scoreColor,
                    color: "#000000",
                  }}
                >
                  {scoreBadgeText}
                </span>
              </div>
              <div style={{ fontSize: "12px", color: "var(--win-text-primary)", lineHeight: 1.35 }}>
                {loading
                  ? "Выполняется диагностика файрвола, служб Docker и параметров ядра..."
                  : audit?.summary || "Нажмите «Повторить проверку» для запуска аудита."}
              </div>
            </div>
          </div>

          {/* Checklist Results */}
          <div
            style={{
              background: "var(--win-card-bg)",
              border: "1px solid var(--win-card-stroke)",
              borderRadius: "8px",
              padding: "12px 14px",
              flexShrink: 0,
            }}
          >
            <div
              style={{
                fontSize: "11px",
                fontWeight: 600,
                color: "var(--win-text-secondary)",
                textTransform: "uppercase",
                letterSpacing: "0.5px",
                marginBottom: "10px",
              }}
            >
              Результаты проверки защитных механизмов:
            </div>

            <div style={{ display: "flex", flexDirection: "column", gap: "9px" }}>
              {/* Check 1: Remnanode service & port 443 */}
              <div className="audit-check-row" style={{ display: "flex", alignItems: "flex-start", gap: "9px", animationDelay: "50ms" }}>
                {audit?.container_running && audit?.port_443_listening ? (
                  <CheckCircle2 size={15} className="audit-check-icon" style={{ color: "#10b981", flexShrink: 0, marginTop: "2px", animationDelay: "100ms" }} />
                ) : audit?.container_running ? (
                  <AlertTriangle size={15} className="audit-check-icon" style={{ color: "#f59e0b", flexShrink: 0, marginTop: "2px", animationDelay: "100ms" }} />
                ) : (
                  <AlertTriangle size={15} className="audit-check-icon" style={{ color: "#ef4444", flexShrink: 0, marginTop: "2px", animationDelay: "100ms" }} />
                )}
                <div>
                  <div style={{ fontSize: "12.5px", fontWeight: 500, color: "var(--win-text-primary)" }}>
                    Служба Xray Reality (порт 443)
                  </div>
                  <div style={{ fontSize: "11.5px", color: "var(--win-text-secondary)", lineHeight: 1.35 }}>
                    {audit?.container_running && audit?.port_443_listening
                      ? "Контейнер remnanode запущен в Docker и слушает порт 443 (Reality)."
                      : audit?.container_running
                      ? "Контейнер remnanode запущен, но порт 443 не обнаружен среди слушающих сокетов."
                      : "Контейнер remnanode не запущен. Нода не принимает входящий трафик."}
                  </div>
                </div>
              </div>

              {/* Check 2: Port 2222 Isolation & Whitelist */}
              <div className="audit-check-row" style={{ display: "flex", alignItems: "flex-start", gap: "9px", animationDelay: "110ms" }}>
                {audit?.node_port_protected && audit?.panel_whitelisted ? (
                  <CheckCircle2 size={15} className="audit-check-icon" style={{ color: "#10b981", flexShrink: 0, marginTop: "2px", animationDelay: "160ms" }} />
                ) : audit?.node_port_protected ? (
                  <AlertTriangle size={15} className="audit-check-icon" style={{ color: "#f59e0b", flexShrink: 0, marginTop: "2px", animationDelay: "160ms" }} />
                ) : (
                  <AlertTriangle size={15} className="audit-check-icon" style={{ color: "#ef4444", flexShrink: 0, marginTop: "2px", animationDelay: "160ms" }} />
                )}
                <div>
                  <div style={{ fontSize: "12.5px", fontWeight: 500, color: "var(--win-text-primary)" }}>
                    Изоляция порта управления (2222) в DOCKER-USER
                  </div>
                  <div style={{ fontSize: "11.5px", color: "var(--win-text-secondary)", lineHeight: 1.35 }}>
                    {audit?.node_port_protected && audit?.panel_whitelisted
                      ? `Порт 2222 отфильтрован в DOCKER-USER: открыт только для мастер-панели (${panelIpInput || "белый список"}), для остальных сканеров активен DROP.`
                      : audit?.node_port_protected
                      ? "Порт 2222 изолирован в DOCKER-USER, но белый список IP панели не подтвержден. Проверьте поле IP панели ниже."
                      : "Уязвимость: порт 2222 выставлен в публичный интернет через Docker NAT. Сканеры ТСПУ легко вычисляют отпечаток ноды."}
                  </div>
                </div>
              </div>

              {/* Check 3: Stealth DROP */}
              <div className="audit-check-row" style={{ display: "flex", alignItems: "flex-start", gap: "9px", animationDelay: "170ms" }}>
                {audit?.stealth_drop_active ? (
                  <CheckCircle2 size={15} className="audit-check-icon" style={{ color: "#10b981", flexShrink: 0, marginTop: "2px", animationDelay: "220ms" }} />
                ) : (
                  <AlertTriangle size={15} className="audit-check-icon" style={{ color: "#f59e0b", flexShrink: 0, marginTop: "2px", animationDelay: "220ms" }} />
                )}
                <div>
                  <div style={{ fontSize: "12.5px", fontWeight: 500, color: "var(--win-text-primary)" }}>
                    Скрытный режим портов (Stealth DROP)
                  </div>
                  <div style={{ fontSize: "11.5px", color: "var(--win-text-secondary)", lineHeight: 1.35 }}>
                    {audit?.stealth_drop_active
                      ? "Политика INPUT DROP активна. Ответные пакеты TCP RST подавлены, сервер имитирует выключенный хост."
                      : "Политика INPUT в режиме ACCEPT. Закрытые порты отвечают пакетами TCP RST, выдавая сервер сетевым зондам."}
                  </div>
                </div>
              </div>

              {/* Check 4: TCP BBR */}
              <div className="audit-check-row" style={{ display: "flex", alignItems: "flex-start", gap: "9px", animationDelay: "230ms" }}>
                {audit?.bbr_active ? (
                  <CheckCircle2 size={15} className="audit-check-icon" style={{ color: "#10b981", flexShrink: 0, marginTop: "2px", animationDelay: "280ms" }} />
                ) : (
                  <AlertTriangle size={15} className="audit-check-icon" style={{ color: "#f59e0b", flexShrink: 0, marginTop: "2px", animationDelay: "280ms" }} />
                )}
                <div>
                  <div style={{ fontSize: "12.5px", fontWeight: 500, color: "var(--win-text-primary)" }}>
                    Оптимизация TCP BBR + FQ (защита от DPI)
                  </div>
                  <div style={{ fontSize: "11.5px", color: "var(--win-text-secondary)", lineHeight: 1.35 }}>
                    {audit?.bbr_active
                      ? "Алгоритм контроля перегрузки Google BBR активен. Высокая скорость при потерях пакетов от ТСПУ."
                      : "TCP BBR не включен (стандартный cubic). При искусственных дропах пакетов скорость может проседать."}
                  </div>
                </div>
              </div>

              {/* Check 5: Xray Core Variant */}
              <div className="audit-check-row" style={{ display: "flex", alignItems: "flex-start", gap: "9px", animationDelay: "290ms" }}>
                <CheckCircle2 size={15} className="audit-check-icon" style={{ color: "var(--win-node-accent)", flexShrink: 0, marginTop: "2px", animationDelay: "340ms" }} />
                <div>
                  <div style={{ fontSize: "12.5px", fontWeight: 500, color: "var(--win-text-primary)" }}>
                    Ядро Xray Core
                  </div>
                  <div style={{ fontSize: "11.5px", color: "var(--win-text-secondary)", lineHeight: 1.35 }}>
                    {audit?.xray_core_type || "Официальное ядро Xray"}. Стандартные протоколы Reality функционируют в штатном режиме.
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* Configuration to Apply - Fluent WinUI 3 Settings Group */}
          <div
            className="settings-card-group"
            style={{
              background: "var(--win-card-bg)",
              border: "1px solid var(--win-card-stroke)",
              borderRadius: "8px",
              overflow: "hidden",
              boxShadow: "var(--win-card-shadow)",
              flexShrink: 0,
            }}
          >
            {/* Group Header */}
            <div
              style={{
                display: "flex",
                alignItems: "center",
                justifyContent: "space-between",
                padding: "10px 16px",
                background: "var(--win-card-row-hover)",
                borderBottom: "1px solid var(--win-card-row-border)",
              }}
            >
              <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
                <Sliders size={14} style={{ color: "var(--win-node-accent)" }} />
                <span style={{ fontSize: "12.5px", fontWeight: 600, color: "var(--win-text-primary)" }}>
                  Параметры сетевого экрана
                </span>
              </div>
              {isFullyProtected ? (
                <span
                  style={{
                    fontSize: "11px",
                    color: "#10b981",
                    background: "rgba(16, 185, 129, 0.12)",
                    border: "1px solid rgba(16, 185, 129, 0.25)",
                    padding: "2px 8px",
                    borderRadius: "10px",
                    display: "inline-flex",
                    alignItems: "center",
                    gap: "4px",
                    fontWeight: 500,
                  }}
                >
                  <CheckCircle2 size={11} /> Все правила активны
                </span>
              ) : (
                <span style={{ fontSize: "11px", color: "var(--win-text-tertiary)" }}>
                  Конфигурация защиты
                </span>
              )}
            </div>

            {/* Row 1: Panel IP */}
            <div className="settings-card-row" style={{ padding: "12px 16px" }}>
              <div className="settings-card-info" style={{ gap: "12px" }}>
                <div
                  className="settings-card-icon"
                  style={{
                    width: "32px",
                    height: "32px",
                    borderRadius: "6px",
                    background: "rgba(56, 189, 248, 0.1)",
                    borderColor: "rgba(56, 189, 248, 0.25)",
                    color: "var(--win-node-accent)",
                  }}
                >
                  <Network size={16} />
                </div>
                <div className="settings-card-text">
                  <div style={{ display: "flex", alignItems: "center", gap: "6px" }}>
                    <span className="settings-card-title" style={{ fontSize: "12.5px", fontWeight: 500, margin: 0 }}>
                      IP мастер-панели (белый список)
                    </span>
                    {parentMasterIp && panelIpInput === parentMasterIp && (
                      <span
                        style={{
                          fontSize: "10px",
                          color: "var(--win-node-accent)",
                          background: "rgba(56, 189, 248, 0.12)",
                          padding: "1px 6px",
                          borderRadius: "4px",
                          fontWeight: 500,
                        }}
                      >
                        Определен из панели
                      </span>
                    )}
                  </div>
                  <span className="settings-card-desc" style={{ fontSize: "11px", marginTop: "2px" }}>
                    {panelIpInput
                      ? `Порт 2222 будет открыт исключительно для IP ${panelIpInput}`
                      : "Укажите IP панели, чтобы разрешить ей связь с нодой по порту 2222"}
                  </span>
                  {!panelIpInput && masterServers.length > 0 && (
                    <div style={{ display: "flex", alignItems: "center", gap: "6px", marginTop: "5px" }}>
                      <span style={{ fontSize: "10.5px", color: "var(--win-text-tertiary)" }}>
                        Подставить IP:
                      </span>
                      {masterServers.map((m) => (
                        <button
                          key={m.id}
                          type="button"
                          className="win-button"
                          style={{
                            padding: "2px 8px",
                            height: "22px",
                            fontSize: "11px",
                            borderRadius: "4px",
                            background: "rgba(56, 189, 248, 0.08)",
                            borderColor: "rgba(56, 189, 248, 0.2)",
                            color: "var(--win-node-accent)",
                          }}
                          onClick={() => setPanelIpInput(m.host)}
                          title={`Использовать адрес ${m.host}`}
                        >
                          <Sparkles size={11} style={{ marginRight: "4px" }} />
                          {m.label || m.host} ({m.host})
                        </button>
                      ))}
                    </div>
                  )}
                </div>
              </div>
              <div className="settings-card-action">
                <input
                  type="text"
                  className="win-textbox"
                  style={{ width: "170px", height: "28px", fontSize: "12px" }}
                  placeholder="например: 81.90.25.120"
                  value={panelIpInput}
                  onChange={(e) => setPanelIpInput(e.target.value.trim())}
                />
              </div>
            </div>

            {/* Row 2: Port 2222 Isolation */}
            <div className="settings-card-row" style={{ padding: "12px 16px" }}>
              <div className="settings-card-info" style={{ gap: "12px" }}>
                <div
                  className="settings-card-icon"
                  style={{
                    width: "32px",
                    height: "32px",
                    borderRadius: "6px",
                    background: "rgba(56, 189, 248, 0.1)",
                    borderColor: "rgba(56, 189, 248, 0.25)",
                    color: "var(--win-node-accent)",
                  }}
                >
                  <Lock size={16} />
                </div>
                <div className="settings-card-text">
                  <span className="settings-card-title" style={{ fontSize: "12.5px", fontWeight: 500, margin: 0 }}>
                    Изоляция порта ноды (2222) в DOCKER-USER
                  </span>
                  <span className="settings-card-desc" style={{ fontSize: "11px", marginTop: "2px" }}>
                    Молчаливый сброс DROP для всех публичных сканеров интернета
                  </span>
                </div>
              </div>
              <div className="settings-card-action">
                <div
                  className={`win-switch ${enableHidePort ? "active" : ""}`}
                  onClick={() => setEnableHidePort(!enableHidePort)}
                >
                  <div className="win-switch-track">
                    <div className="win-switch-thumb" />
                  </div>
                  <span className="win-switch-label">{enableHidePort ? "Вкл" : "Откл"}</span>
                </div>
              </div>
            </div>

            {/* Row 3: Stealth DROP */}
            <div className="settings-card-row" style={{ padding: "12px 16px" }}>
              <div className="settings-card-info" style={{ gap: "12px" }}>
                <div
                  className="settings-card-icon"
                  style={{
                    width: "32px",
                    height: "32px",
                    borderRadius: "6px",
                    background: "rgba(96, 205, 255, 0.1)",
                    borderColor: "rgba(96, 205, 255, 0.25)",
                    color: "#60cdff",
                  }}
                >
                  <ShieldAlert size={16} />
                </div>
                <div className="settings-card-text">
                  <span className="settings-card-title" style={{ fontSize: "12.5px", fontWeight: 500, margin: 0 }}>
                    Скрытный режим портов (Stealth DROP)
                  </span>
                  <span className="settings-card-desc" style={{ fontSize: "11px", marginTop: "2px" }}>
                    Подавление ответных пакетов TCP RST (маскировка под выключенный сервер)
                  </span>
                </div>
              </div>
              <div className="settings-card-action">
                <div
                  className={`win-switch ${enableStealthDrop ? "active" : ""}`}
                  onClick={() => setEnableStealthDrop(!enableStealthDrop)}
                >
                  <div className="win-switch-track">
                    <div className="win-switch-thumb" />
                  </div>
                  <span className="win-switch-label">{enableStealthDrop ? "Вкл" : "Откл"}</span>
                </div>
              </div>
            </div>

            {/* Row 4: TCP BBR */}
            <div className="settings-card-row" style={{ padding: "12px 16px" }}>
              <div className="settings-card-info" style={{ gap: "12px" }}>
                <div
                  className="settings-card-icon"
                  style={{
                    width: "32px",
                    height: "32px",
                    borderRadius: "6px",
                    background: "rgba(16, 185, 129, 0.1)",
                    borderColor: "rgba(16, 185, 129, 0.25)",
                    color: "#10b981",
                  }}
                >
                  <Zap size={16} />
                </div>
                <div className="settings-card-text">
                  <span className="settings-card-title" style={{ fontSize: "12.5px", fontWeight: 500, margin: 0 }}>
                    Включить ускорение TCP BBR + FQ
                  </span>
                  <span className="settings-card-desc" style={{ fontSize: "11px", marginTop: "2px" }}>
                    Оптимизация сетевого стека ядра Linux для защиты от DPI-дропов скорости
                  </span>
                </div>
              </div>
              <div className="settings-card-action">
                <div
                  className={`win-switch ${enableBBR ? "active" : ""}`}
                  onClick={() => setEnableBBR(!enableBBR)}
                >
                  <div className="win-switch-track">
                    <div className="win-switch-thumb" />
                  </div>
                  <span className="win-switch-label">{enableBBR ? "Вкл" : "Откл"}</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Pinned Action Footer - ALWAYS VISIBLE regardless of screen height */}
        <div
          className="win-modal-footer"
          style={{
            flexShrink: 0,
            padding: "12px 18px",
            background: "var(--win-card-row-hover)",
            borderTop: "1px solid var(--win-card-stroke)",
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
          }}
        >
          <button
            className="win-button"
            disabled={loading || applying}
            onClick={() => onRefresh(panelIpInput)}
            title="Повторно опросить параметры фаервола на сервере"
          >
            {loading ? <RefreshCw size={13} className="spin" /> : <RefreshCw size={13} />}
            <span>Повторить проверку</span>
          </button>

          <div style={{ display: "flex", gap: "8px" }}>
            <button
              className="win-button"
              disabled={loading || applying}
              onClick={close}
            >
              Закрыть
            </button>

            <button
              className="win-button hero-primary"
              disabled={loading || applying}
              onClick={() =>
                onApplyProtection(
                  panelIpInput,
                  enableHidePort,
                  enableStealthDrop,
                  enableBBR
                )
              }
            >
              {applying ? <RefreshCw size={13} className="spin" /> : <ShieldCheck size={13} />}
              <span>
                {applying
                  ? "Применение правил..."
                  : isFullyProtected
                  ? "Обновить правила защиты"
                  : "Применить правила защиты"}
              </span>
            </button>
          </div>
        </div>
        </>
      )}
    </WinModal>
  );
};
