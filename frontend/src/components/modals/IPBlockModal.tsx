import React, { useState } from "react";
import {
  ShieldAlert,
  X,
  RefreshCw,
  CheckCircle2,
  AlertTriangle,
  AlertCircle,
  AlertOctagon,
  Server,
  MapPin,
  Check,
  Copy,
  Info,
  Globe,
} from "lucide-react";
import * as types from "../../../bindings/remna-ui/internal/types/models";
import { WinModal } from "../ui/WinModal";

interface IPProbeItem {
  node_id: string;
  location: string;
  country_code: string;
  city: string;
  success: boolean;
  latency_ms: number;
  error?: string;
}

interface PortDetailItem {
  port: number;
  protocol: string;
  status: string;
  probes: IPProbeItem[];
}

interface IPBlockData {
  host: string;
  port: number;
  status: string;
  summary: string;
  isp_info?: string;
  location_info?: string;
  client_country?: string;
  is_vpn_detected?: boolean;
  probes?: IPProbeItem[];
  port_details?: PortDetailItem[];
  is_rkn_blocked?: boolean;
  checked_at?: string;
}

interface IPBlockModalProps {
  server: { host: string; port?: number; label?: string; id?: string };
  result: IPBlockData | null;
  loading: boolean;
  onClose: () => void;
  onRefresh: () => void;
}

export const IPBlockModal: React.FC<IPBlockModalProps> = ({
  server,
  result,
  loading,
  onClose,
  onRefresh,
}) => {
  const [copied, setCopied] = useState(false);

  const handleCopyReport = () => {
    if (!result) return;
    const lines = [
      `[Диагностика доступности IP в РФ]`,
      `Хост: ${result.host}:${result.port}`,
      `Вердикт: ${result.status.toUpperCase()} (${result.summary})`,
      result.isp_info ? `Провайдер/ASN: ${result.isp_info}` : "",
      result.location_info ? `Локация: ${result.location_info}` : "",
      result.is_vpn_detected && result.client_country ? `ПК шлюз: VPN (${result.client_country})` : "",
      `Время проверки: ${result.checked_at}`,
      ...(result.port_details && result.port_details.length > 0
        ? result.port_details.flatMap((pd) => [
            `\n[${pd.protocol} - Статус: ${pd.status.toUpperCase()}]:`,
            ...pd.probes.map(
              (p: IPProbeItem) => `  - ${p.location} (${p.city || p.country_code}): ${p.success ? `OK (${p.latency_ms} ms)` : `СБОЙ (${p.error || "нет ответа"})`}`
            ),
          ])
        : [
            `\nУзлы мониторинга:`,
            ...(result.probes || []).map(
              (p: IPProbeItem) => `  - ${p.location} (${p.city || p.country_code}): ${p.success ? `OK (${p.latency_ms} ms)` : `СБОЙ (${p.error || "нет ответа"})`}`
            ),
          ]),
    ].filter(Boolean);

    navigator.clipboard.writeText(lines.join("\n"));
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const serverTitle = server.label || server.host;
  const currentPort = result?.port || server.port || 22;

  return (
    <WinModal
      isOpen={true}
      onClose={onClose}
      maxWidth="600px"
      width="95%"
      preventBackdropClose={loading}
    >
      {/* Header */}
        <div className="win-modal-header">
          <div style={{ display: "flex", alignItems: "center", gap: "10px" }}>
            <div
              className="modal-title-icon"
              style={{ background: "rgba(56, 189, 248, 0.15)", color: "#38bdf8" }}
            >
              <ShieldAlert size={18} />
            </div>
            <div>
              <h3 className="win-modal-title">Доступность IP в России</h3>
              <p className="win-modal-desc">
                Проверка блокировок ТСПУ и РКН для {serverTitle}
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
          {loading ? (
            <div
              style={{
                display: "flex",
                flexDirection: "column",
                alignItems: "center",
                justifyContent: "center",
                padding: "36px 16px",
                gap: "14px",
                textAlign: "center",
              }}
            >
              <RefreshCw size={28} className="animate-spin" style={{ color: "var(--win-accent)" }} />
              <div>
                <div style={{ fontSize: "14px", fontWeight: 600, color: "var(--win-text-primary)" }}>
                  Диагностика контрольных узлов в РФ и ЕС...
                </div>
                <div style={{ fontSize: "12px", color: "var(--win-text-secondary)", marginTop: "4px" }}>
                  Опрос тестовых точек: Москва, Санкт-Петербург и европейский эталон
                </div>
              </div>
            </div>
          ) : result ? (
            <>
              {/* Verdict Card */}
              <div className={`ip-check-card ${result.status}`}>
                <div className="ip-check-header">
                  <div className="ip-check-verdict-group">
                    <span className={`ip-check-badge ${result.status}`}>
                      {result.status === "clean" ? (
                        <>
                          <CheckCircle2 size={13} />
                          <span>Чистый IP · Доступен в РФ</span>
                        </>
                      ) : result.status === "blocked" ? (
                        <>
                          <AlertTriangle size={13} />
                          <span>Блокировка ТСПУ / РКН</span>
                        </>
                      ) : result.status === "partial" ? (
                        <>
                          <AlertCircle size={13} />
                          <span>Частичная доступность</span>
                        </>
                      ) : (
                        <>
                          <AlertOctagon size={13} />
                          <span>Хост недоступен</span>
                        </>
                      )}
                    </span>
                    <span style={{ fontSize: "12.5px", fontWeight: 600, color: "var(--win-text-primary)" }}>
                      {result.host}:{result.port}
                    </span>
                  </div>

                  <div style={{ display: "flex", alignItems: "center", gap: "6px", flexWrap: "wrap" }}>
                    {result.isp_info && (
                      <span className="server-chip" style={{ fontSize: "11px" }}>
                        <Server size={10} />
                        <span>{result.isp_info}</span>
                      </span>
                    )}
                    {result.location_info && (
                      <span className="server-chip" style={{ fontSize: "11px" }}>
                        <MapPin size={10} />
                        <span>{result.location_info}</span>
                      </span>
                    )}
                    {result.is_vpn_detected && result.client_country && (
                      <span
                        className="server-chip"
                        style={{
                          fontSize: "11px",
                          color: "#38bdf8",
                          borderColor: "rgba(56, 189, 248, 0.35)",
                          background: "rgba(56, 189, 248, 0.08)",
                        }}
                        title={`Ваш ПК подключен через VPN или прокси-шлюз в ${result.client_country}`}
                      >
                        <Globe size={10} />
                        <span>ПК через VPN: {result.client_country}</span>
                      </span>
                    )}
                  </div>
                </div>

                <p className="ip-check-summary">{result.summary}</p>

                {/* Multi-Port Probe Breakdown or Single Probes Fallback */}
                {result.port_details && result.port_details.length > 0 ? (
                  <div style={{ display: "flex", flexDirection: "column", gap: "10px", marginTop: "10px" }}>
                    {result.port_details.map((pd, pdIdx) => (
                      <div key={pdIdx} className="ip-port-detail-box">
                        <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: "8px" }}>
                          <span style={{ fontSize: "12px", fontWeight: 600, color: "var(--win-text-primary)" }}>
                            {pd.protocol}
                          </span>
                          <span className={`ip-check-badge ${pd.status}`} style={{ fontSize: "11px", padding: "2px 7px" }}>
                            {pd.status === "clean" ? "Доступен" : pd.status === "blocked" ? "Блокировка ТСПУ" : pd.status === "partial" ? "Частично" : "Офлайн"}
                          </span>
                        </div>
                        <div className="ip-probe-pills-row">
                          {pd.probes.map((pr: IPProbeItem, pIdx: number) => {
                            const isRef = pr.country_code === "de";
                            const pillClass = isRef ? "ref" : pr.success ? "ok" : "fail";
                            return (
                              <span
                                key={pIdx}
                                className={`ip-probe-pill ${pillClass}`}
                                title={pr.error || `Задержка: ${pr.latency_ms} ms (${pr.location})`}
                              >
                                {pr.success ? <Check size={10} /> : <X size={10} />}
                                <span>
                                  {pr.location}: {pr.success ? `${pr.latency_ms} ms` : (pr.error || "Сбой")}
                                </span>
                              </span>
                            );
                          })}
                        </div>
                      </div>
                    ))}
                  </div>
                ) : result.probes && result.probes.length > 0 ? (
                  <div style={{ marginTop: "10px" }}>
                    <div style={{ fontSize: "11px", color: "var(--win-text-secondary)", marginBottom: "6px", fontWeight: 500 }}>
                      Результаты распределенных пробников:
                    </div>
                    <div className="ip-probe-pills-row">
                      {result.probes.map((pr: IPProbeItem, pIdx: number) => {
                        const isRef = pr.country_code === "de";
                        const pillClass = isRef ? "ref" : pr.success ? "ok" : "fail";
                        return (
                          <span
                            key={pIdx}
                            className={`ip-probe-pill ${pillClass}`}
                            title={pr.error || `Задержка: ${pr.latency_ms} ms (${pr.location})`}
                          >
                            {pr.success ? <Check size={10} /> : <X size={10} />}
                            <span>
                              {pr.location}: {pr.success ? `${pr.latency_ms} ms` : (pr.error || "Сбой")}
                            </span>
                          </span>
                        );
                      })}
                    </div>
                  </div>
                ) : null}
              </div>

              {/* Explanatory Recommendations */}
              {result.status === "blocked" && (
                <div
                  style={{
                    padding: "12px 14px",
                    background: "rgba(239, 68, 68, 0.08)",
                    border: "1px solid rgba(239, 68, 68, 0.25)",
                    borderLeft: "3.5px solid #ef4444",
                    borderRadius: "6px",
                    display: "flex",
                    gap: "10px",
                    fontSize: "12px",
                    color: "var(--win-text-primary)",
                    lineHeight: "1.45",
                  }}
                >
                  <AlertTriangle size={16} style={{ color: "#ef4444", flexShrink: 0, marginTop: "2px" }} />
                  <div>
                    <strong style={{ color: "#f87171" }}>Рекомендация:</strong> Этот IP-адрес или подсеть заблокированы на ТСПУ магистральных операторов РФ.
                    Клиенты из России не смогут подключаться к этой ноде напрямую. Рекомендуется запросить замену IP-адреса у хостера или настроить Multi-hop / Reverse-proxy транзит через незаблокированный сервер.
                  </div>
                </div>
              )}

              {result.status === "clean" && (
                <div
                  style={{
                    padding: "12px 14px",
                    background: "rgba(16, 185, 129, 0.08)",
                    border: "1px solid rgba(16, 185, 129, 0.25)",
                    borderLeft: "3.5px solid #10b981",
                    borderRadius: "6px",
                    display: "flex",
                    gap: "10px",
                    fontSize: "12px",
                    color: "var(--win-text-primary)",
                    lineHeight: "1.45",
                  }}
                >
                  <CheckCircle2 size={16} style={{ color: "#10b981", flexShrink: 0, marginTop: "2px" }} />
                  <div>
                    <strong style={{ color: "#34d399" }}>IP чист:</strong> Успешный отклик со всех контрольных точек в РФ.
                    Прямой доступ к панели и нодам Remnawave из России свободен и не подвержен блокировкам ТСПУ.
                  </div>
                </div>
              )}

              {result.status === "partial" && (
                <div
                  style={{
                    padding: "12px 14px",
                    background: "rgba(245, 158, 11, 0.08)",
                    border: "1px solid rgba(245, 158, 11, 0.25)",
                    borderLeft: "3.5px solid #f59e0b",
                    borderRadius: "6px",
                    display: "flex",
                    gap: "10px",
                    fontSize: "12px",
                    color: "var(--win-text-primary)",
                    lineHeight: "1.45",
                  }}
                >
                  <AlertCircle size={16} style={{ color: "#f59e0b", flexShrink: 0, marginTop: "2px" }} />
                  <div>
                    <strong style={{ color: "#fbbf24" }}>Внимание:</strong> Частичная доступность. IP доступен не через всех провайдеров в РФ (возможны проблемы у мобильных операторов или в отдельных регионах).
                  </div>
                </div>
              )}

              {result.status === "offline" && (
                <div
                  style={{
                    padding: "12px 14px",
                    background: "rgba(100, 116, 139, 0.1)",
                    border: "1px solid rgba(100, 116, 139, 0.25)",
                    borderLeft: "3.5px solid #94a3b8",
                    borderRadius: "6px",
                    display: "flex",
                    gap: "10px",
                    fontSize: "12px",
                    color: "var(--win-text-primary)",
                    lineHeight: "1.45",
                  }}
                >
                  <AlertOctagon size={16} style={{ color: "#94a3b8", flexShrink: 0, marginTop: "2px" }} />
                  <div>
                    <strong style={{ color: "#cbd5e1" }}>Хост не отвечает:</strong> Порт {currentPort} закрыт или сервер выключен.
                    Проверьте запущен ли SSH/сервис на VPS и не блокирует ли файрвол (ufw / security group) входящие подключения.
                  </div>
                </div>
              )}
            </>
          ) : (
            <div style={{ textAlign: "center", padding: "24px 0", color: "var(--win-text-secondary)" }}>
              Нажмите «Запустить проверку» для анализа доступности IP-адреса.
            </div>
          )}
        </div>

        {/* Footer */}
        <div
          className="win-modal-footer"
          style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}
        >
          <div>
            {result && (
              <button
                type="button"
                className="win-button"
                onClick={handleCopyReport}
                title="Скопировать текстовый отчет в буфер обмена"
                style={{ display: "inline-flex", alignItems: "center", gap: "6px" }}
              >
                {copied ? <Check size={13} style={{ color: "var(--win-success)" }} /> : <Copy size={13} />}
                <span>{copied ? "Скопировано" : "Скопировать отчет"}</span>
              </button>
            )}
          </div>

          <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
            <button
              type="button"
              className="win-button"
              disabled={loading}
              onClick={onRefresh}
              style={{ display: "inline-flex", alignItems: "center", gap: "6px" }}
            >
              <RefreshCw size={13} className={loading ? "animate-spin" : ""} />
              <span>{loading ? "Диагностика..." : "Повторить"}</span>
            </button>
            <button
              type="button"
              className="win-button"
              style={{ background: "var(--win-accent)", color: "#ffffff", borderColor: "var(--win-accent)" }}
              onClick={onClose}
            >
              Закрыть
            </button>
          </div>
        </div>
    </WinModal>
  );
};
