import React, { useState } from "react";
import {
  Activity,
  X,
  RefreshCw,
  CheckCircle2,
  AlertTriangle,
  AlertOctagon,
  Lock,
  Globe,
  Zap,
  Check,
  Copy,
  Layers,
  ArrowRight,
} from "lucide-react";
import * as types from "../../../bindings/remna-ui/internal/types/models";

interface VPNProbeModalProps {
  server: {
    host: string;
    port?: number;
    label?: string;
    id?: string;
    decoy_domain?: string;
    role?: string;
  };
  result: types.VPNProbeResult | null;
  loading: boolean;
  onClose: () => void;
  onRefresh: (customVless?: string) => void;
}

export const VPNProbeModal: React.FC<VPNProbeModalProps> = ({
  server,
  result,
  loading,
  onClose,
  onRefresh,
}) => {
  const [customVlessInput, setCustomVlessInput] = useState("");
  const [copied, setCopied] = useState(false);

  const handleCopyReport = () => {
    if (!result) return;
    const lines = [
      `[Синтетический тест VPN-трафика]`,
      `Хост: ${result.host}:${result.port}`,
      `Decoy Domain (маскировка): ${result.decoy_domain}`,
      `Вердикт: ${result.status.toUpperCase()} (${result.summary})`,
      `L4 TCP Соединение: ${result.tcp_connected ? `Успешно (${result.tcp_latency_ms} ms)` : `Ошибка (${result.tcp_error || "нет ответа"})`}`,
      `L7 TLS / Reality Handshake: ${result.tls_handshake ? `Успешно (${result.tls_latency_ms} ms, ${result.tls_version || ""} ${result.cipher_suite || ""})` : result.is_tspu_dpi_reset ? `БЛОКИРОВКА ТСПУ (TCP RST после ClientHello)` : `Ошибка (${result.tls_error || "сбой"})`}`,
      `L7 End-to-End HTTP 204: ${result.e2e_success ? `Успешно (${result.e2e_latency_ms} ms, HTTP ${result.e2e_http_code})` : `Ошибка (${result.e2e_error || "сбой"})`}`,
      `Целевой эндпоинт: ${result.e2e_target}`,
      `Время проверки: ${result.timestamp}`,
    ].filter(Boolean);

    navigator.clipboard.writeText(lines.join("\n"));
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleTriggerTest = () => {
    onRefresh(customVlessInput.trim());
  };

  const serverTitle = server.label || server.host;
  const currentPort = result?.port || server.port || 443;
  const currentDecoy = result?.decoy_domain || server.decoy_domain || server.host;

  return (
    <div className="win-modal-overlay" onClick={() => !loading && onClose()}>
      <div
        className="win-modal-content"
        style={{ maxWidth: "620px", width: "95%" }}
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="win-modal-header">
          <div style={{ display: "flex", alignItems: "center", gap: "10px" }}>
            <div
              className="modal-title-icon"
              style={{ background: "rgba(168, 85, 247, 0.15)", color: "#c084fc" }}
            >
              <Activity size={18} />
            </div>
            <div>
              <h3 className="win-modal-title">Синтетический тест VPN-трафика</h3>
              <p className="win-modal-desc">
                Проверка прохождения реального клиентского трафика и детекция блокировок ТСПУ для {serverTitle}
              </p>
            </div>
          </div>
          <button
            className="win-modal-close"
            disabled={loading}
            onClick={onClose}
            title="Закрыть"
          >
            <X size={15} />
          </button>
        </div>

        {/* Body */}
        <div
          className="win-modal-body"
          style={{ display: "flex", flexDirection: "column", gap: "14px", padding: "18px 22px" }}
        >
          {/* Target Host Details Chip */}
          <div
            style={{
              display: "flex",
              alignItems: "center",
              justifyContent: "space-between",
              background: "var(--win-card-bg)",
              border: "1px solid var(--win-border-light)",
              borderRadius: "8px",
              padding: "10px 14px",
              fontSize: "12px",
            }}
          >
            <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
              <Globe size={14} style={{ color: "var(--win-accent)" }} />
              <span style={{ fontWeight: 600, color: "var(--win-text-primary)" }}>
                {server.host}:{currentPort}
              </span>
            </div>
            <div style={{ display: "flex", alignItems: "center", gap: "6px", color: "var(--win-text-secondary)" }}>
              <Lock size={12} />
              <span>Маскировка (SNI): <strong style={{ color: "var(--win-text-primary)" }}>{currentDecoy}</strong></span>
            </div>
          </div>

          {/* Optional VLESS URI Input */}
          <div style={{ display: "flex", flexDirection: "column", gap: "6px" }}>
            <label style={{ fontSize: "11px", color: "var(--win-text-secondary)", fontWeight: 500 }}>
              Произвольная VLESS-ссылка для сквозного тестирования (опционально):
            </label>
            <div style={{ display: "flex", gap: "8px" }}>
              <input
                type="text"
                placeholder="vless://uuid@host:443?security=reality&sni=..."
                value={customVlessInput}
                onChange={(e) => setCustomVlessInput(e.target.value)}
                disabled={loading}
                style={{
                  flex: 1,
                  background: "var(--win-input-bg)",
                  border: "1px solid var(--win-border)",
                  borderRadius: "6px",
                  padding: "6px 10px",
                  fontSize: "12px",
                  color: "var(--win-text-primary)",
                  outline: "none",
                }}
              />
              <button
                className="win-btn primary"
                onClick={handleTriggerTest}
                disabled={loading}
                style={{ padding: "6px 12px", fontSize: "12px", whiteSpace: "nowrap" }}
              >
                {loading ? <RefreshCw size={13} className="animate-spin" /> : "Запустить тест"}
              </button>
            </div>
          </div>

          {/* Loading View */}
          {loading ? (
            <div
              style={{
                display: "flex",
                flexDirection: "column",
                alignItems: "center",
                justifyContent: "center",
                padding: "32px 16px",
                gap: "12px",
                textAlign: "center",
              }}
            >
              <RefreshCw size={26} className="animate-spin" style={{ color: "#c084fc" }} />
              <div>
                <div style={{ fontSize: "14px", fontWeight: 600, color: "var(--win-text-primary)" }}>
                  Выполняется синтетический тест соединения...
                </div>
                <div style={{ fontSize: "12px", color: "var(--win-text-secondary)", marginTop: "4px" }}>
                  Проверка L4 TCP сокета, TLS/Reality рукопожатия и сквозного HTTP 204 выхода в интернет
                </div>
              </div>
            </div>
          ) : result ? (
            <>
              {/* Verdict Banner */}
              <div
                style={{
                  display: "flex",
                  alignItems: "flex-start",
                  gap: "12px",
                  padding: "14px 16px",
                  borderRadius: "8px",
                  background:
                    result.status === "clean"
                      ? "rgba(34, 197, 94, 0.12)"
                      : result.status === "dpi_blocked"
                      ? "rgba(239, 68, 68, 0.15)"
                      : result.status === "offline"
                      ? "rgba(100, 116, 139, 0.15)"
                      : "rgba(245, 158, 11, 0.15)",
                  border:
                    result.status === "clean"
                      ? "1px solid rgba(34, 197, 94, 0.35)"
                      : result.status === "dpi_blocked"
                      ? "1px solid rgba(239, 68, 68, 0.4)"
                      : result.status === "offline"
                      ? "1px solid rgba(100, 116, 139, 0.35)"
                      : "1px solid rgba(245, 158, 11, 0.4)",
                }}
              >
                <div style={{ marginTop: "2px" }}>
                  {result.status === "clean" ? (
                    <CheckCircle2 size={20} style={{ color: "#22c55e" }} />
                  ) : result.status === "dpi_blocked" ? (
                    <AlertOctagon size={20} style={{ color: "#ef4444" }} />
                  ) : result.status === "offline" ? (
                    <AlertTriangle size={20} style={{ color: "#94a3b8" }} />
                  ) : (
                    <AlertTriangle size={20} style={{ color: "#f59e0b" }} />
                  )}
                </div>
                <div style={{ flex: 1 }}>
                  <div
                    style={{
                      fontSize: "14px",
                      fontWeight: 600,
                      color:
                        result.status === "clean"
                          ? "#22c55e"
                          : result.status === "dpi_blocked"
                          ? "#ef4444"
                          : result.status === "offline"
                          ? "#cbd5e1"
                          : "#f59e0b",
                    }}
                  >
                    {result.status === "clean"
                      ? "VPN-трафик проходит успешно"
                      : result.status === "dpi_blocked"
                      ? "Обнаружена блокировка ТСПУ (DPI)!"
                      : result.status === "offline"
                      ? "Узел недоступен на сетевом уровне"
                      : "Деградация соединения"}
                  </div>
                  <div
                    style={{
                      fontSize: "12px",
                      color: "var(--win-text-primary)",
                      marginTop: "4px",
                      lineHeight: "1.4",
                    }}
                  >
                    {result.summary}
                  </div>
                </div>
              </div>

              {/* Multi-layer Diagnostic Cards */}
              <div style={{ display: "flex", flexDirection: "column", gap: "10px" }}>
                {/* 1. L4 TCP */}
                <div
                  style={{
                    background: "var(--win-card-bg)",
                    border: "1px solid var(--win-border-light)",
                    borderRadius: "8px",
                    padding: "12px 14px",
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "space-between",
                  }}
                >
                  <div style={{ display: "flex", alignItems: "center", gap: "10px" }}>
                    <Zap
                      size={16}
                      style={{ color: result.tcp_connected ? "#38bdf8" : "#ef4444" }}
                    />
                    <div>
                      <div style={{ fontSize: "13px", fontWeight: 600, color: "var(--win-text-primary)" }}>
                        Уровень L4: TCP Connect
                      </div>
                      <div style={{ fontSize: "11px", color: "var(--win-text-secondary)" }}>
                        {result.tcp_connected ? "Сокет открыт и принимает соединения" : result.tcp_error || "Порт закрыт"}
                      </div>
                    </div>
                  </div>
                  <div
                    style={{
                      fontSize: "12px",
                      fontWeight: 600,
                      color: result.tcp_connected ? "#38bdf8" : "#ef4444",
                    }}
                  >
                    {result.tcp_connected ? `${result.tcp_latency_ms} ms` : "Отказ"}
                  </div>
                </div>

                {/* 2. L7 TLS / Reality Handshake */}
                <div
                  style={{
                    background: "var(--win-card-bg)",
                    border: "1px solid var(--win-border-light)",
                    borderRadius: "8px",
                    padding: "12px 14px",
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "space-between",
                  }}
                >
                  <div style={{ display: "flex", alignItems: "center", gap: "10px" }}>
                    <Lock
                      size={16}
                      style={{
                        color:
                          result.tls_handshake
                            ? "#a855f7"
                            : result.is_tspu_dpi_reset
                            ? "#ef4444"
                            : "#f59e0b",
                      }}
                    />
                    <div>
                      <div style={{ display: "flex", alignItems: "center", gap: "6px" }}>
                        <span style={{ fontSize: "13px", fontWeight: 600, color: "var(--win-text-primary)" }}>
                          Уровень L7: TLS / Reality Handshake
                        </span>
                        {result.is_tspu_dpi_reset && (
                          <span
                            style={{
                              fontSize: "10px",
                              padding: "2px 6px",
                              borderRadius: "4px",
                              background: "rgba(239, 68, 68, 0.2)",
                              color: "#ef4444",
                              fontWeight: 600,
                            }}
                          >
                            ТСПУ TCP RST
                          </span>
                        )}
                      </div>
                      <div style={{ fontSize: "11px", color: "var(--win-text-secondary)", marginTop: "2px" }}>
                        {result.tls_handshake
                          ? `${result.tls_version || "TLS"} | ${result.cipher_suite || "Reality"} | SNI: ${result.decoy_domain}`
                          : result.is_tspu_dpi_reset
                          ? `Сброс сессии сразу после ClientHello для ${result.decoy_domain}`
                          : result.tls_error || "Ошибка рукопожатия"}
                      </div>
                    </div>
                  </div>
                  <div
                    style={{
                      fontSize: "12px",
                      fontWeight: 600,
                      color:
                        result.tls_handshake
                          ? "#a855f7"
                          : result.is_tspu_dpi_reset
                          ? "#ef4444"
                          : "#f59e0b",
                    }}
                  >
                    {result.tls_handshake ? `${result.tls_latency_ms} ms` : result.is_tspu_dpi_reset ? "RST" : "Ошибка"}
                  </div>
                </div>

                {/* 3. L7 End-to-End Egress */}
                <div
                  style={{
                    background: "var(--win-card-bg)",
                    border: "1px solid var(--win-border-light)",
                    borderRadius: "8px",
                    padding: "12px 14px",
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "space-between",
                  }}
                >
                  <div style={{ display: "flex", alignItems: "center", gap: "10px" }}>
                    <Layers
                      size={16}
                      style={{ color: result.e2e_success ? "#22c55e" : "#f59e0b" }}
                    />
                    <div>
                      <div style={{ fontSize: "13px", fontWeight: 600, color: "var(--win-text-primary)" }}>
                        Сквозной выход: HTTP 204 Egress
                      </div>
                      <div style={{ fontSize: "11px", color: "var(--win-text-secondary)", marginTop: "2px" }}>
                        {result.e2e_success
                          ? `Эндпоинт ${result.e2e_target} ответил кодом ${result.e2e_http_code}`
                          : result.e2e_error || "Сбой выхода в глобальную сеть"}
                      </div>
                    </div>
                  </div>
                  <div
                    style={{
                      fontSize: "12px",
                      fontWeight: 600,
                      color: result.e2e_success ? "#22c55e" : "#f59e0b",
                    }}
                  >
                    {result.e2e_success ? `${result.e2e_latency_ms} ms` : "Ошибка"}
                  </div>
                </div>
              </div>

              {/* Timestamp */}
              <div
                style={{
                  fontSize: "11px",
                  color: "var(--win-text-secondary)",
                  display: "flex",
                  justifyContent: "space-between",
                  marginTop: "2px",
                }}
              >
                <span>Проверено: {result.timestamp}</span>
                <span>Цель: Cloudflare generate_204</span>
              </div>
            </>
          ) : null}
        </div>

        {/* Footer */}
        <div
          className="win-modal-footer"
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            padding: "14px 22px",
            borderTop: "1px solid var(--win-border-light)",
          }}
        >
          <div style={{ display: "flex", gap: "8px" }}>
            {result && (
              <button
                className="win-btn secondary"
                onClick={handleCopyReport}
                style={{ display: "flex", alignItems: "center", gap: "6px" }}
              >
                {copied ? <Check size={14} style={{ color: "#22c55e" }} /> : <Copy size={14} />}
                <span>{copied ? "Скопировано" : "Скопировать отчет"}</span>
              </button>
            )}
          </div>
          <div style={{ display: "flex", gap: "8px" }}>
            <button
              className="win-btn secondary"
              onClick={handleTriggerTest}
              disabled={loading}
              style={{ display: "flex", alignItems: "center", gap: "6px" }}
            >
              <RefreshCw size={14} className={loading ? "animate-spin" : ""} />
              <span>Повторить</span>
            </button>
            <button
              className="win-btn primary"
              onClick={onClose}
              disabled={loading}
            >
              Закрыть
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
