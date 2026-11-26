import React, { useState } from "react";
import {
  Edit3,
  X,
  Folder,
  Check,
  Server,
  Radio,
  Key,
  FileKey,
  Eye,
  EyeOff,
  Globe,
  Lock,
  Plus,
  Activity,
  RefreshCw,
  CheckCircle2,
  AlertCircle,
  AlertTriangle,
  AlertOctagon,
  ShieldAlert,
  MapPin,
} from "lucide-react";
import * as types from "../../../bindings/remna-ui/internal/types/models";
import { PasswordInput } from "../ui/PasswordInput";
import { WinSelect } from "../ui/WinSelect";
import { WinModal } from "../ui/WinModal";
import { WinTooltip } from "../ui/WinTooltip";

export interface EditServerData {
  server: types.SavedServer;
  name: string;
  host: string;
  port: number;
  user: string;
  authMethod: "password" | "key";
  password?: string;
  privateKey?: string;
  role?: "master" | "node";
  parentMasterId?: string;
  locationCode?: string;
  decoyDomain?: string;
  isExternalMaster?: boolean;
  webUrl?: string;
}

interface EditServerModalProps {
  data: EditServerData;
  availableMasters: types.SavedServer[];
  onChange: (updated: EditServerData) => void;
  onClose: () => void;
  onSave: () => void;
  onSelectKeyFile: () => Promise<string>;
  onTestSSH?: (host: string, port: number, user: string, pass: string, key: string) => Promise<any>;
  onCheckIPBlock?: (host: string, port: number) => Promise<any>;
}

export const EditServerModal: React.FC<EditServerModalProps> = ({
  data,
  availableMasters,
  onChange,
  onClose,
  onSave,
  onSelectKeyFile,
  onTestSSH,
  onCheckIPBlock,
}) => {
  const [testingSSH, setTestingSSH] = useState<boolean>(false);
  const [sshTestResult, setSshTestResult] = useState<types.SSHTestResult | null>(null);
  const [checkingIP, setCheckingIP] = useState<boolean>(false);
  const [ipCheckResult, setIpCheckResult] = useState<types.IPBlockCheckResult | null>(null);

  const isNew = !data.server.host || data.server.last_seen === "Только что добавлен";

  const handleCheckIPBlock = async () => {
    if (!onCheckIPBlock || !data.host) return;
    setCheckingIP(true);
    setIpCheckResult(null);
    try {
      const res = await onCheckIPBlock(data.host, data.port || 22);
      setIpCheckResult(res);
    } catch (err: any) {
      console.error(err);
    } finally {
      setCheckingIP(false);
    }
  };

  const handleTestSSH = async () => {
    if (!onTestSSH || !data.host) return;
    setTestingSSH(true);
    setSshTestResult(null);
    try {
      const pass = data.authMethod === "password" ? (data.password || "") : "";
      const key = data.authMethod === "key" ? (data.privateKey || "") : "";
      const res = await onTestSSH(data.host, data.port || 22, data.user || "root", pass, key);
      setSshTestResult(res);
    } catch (err: any) {
      setSshTestResult(
        new types.SSHTestResult({
          success: false,
          error: err?.message || String(err),
        })
      );
    } finally {
      setTestingSSH(false);
    }
  };

  return (
    <WinModal
      isOpen={true}
      onClose={onClose}
      maxWidth="600px"
      maxHeight="88vh"
      dialogStyle={{
        display: "flex",
        flexDirection: "column",
      }}
    >
      {({ close }) => (
        <>
      {/* Header */}
        <div className="win-modal-header" style={{ flexShrink: 0 }}>
          <div style={{ display: "flex", alignItems: "center", gap: "10px" }}>
            <div
              className="modal-title-icon"
              style={{
                width: "34px",
                height: "34px",
                borderRadius: "6px",
                background: "rgba(96, 205, 255, 0.12)",
                border: "1px solid rgba(96, 205, 255, 0.25)",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                color: "var(--win-accent)",
              }}
            >
              {isNew ? <Plus size={17} /> : <Edit3 size={17} />}
            </div>
            <div>
              <h3 className="win-modal-title" style={{ fontSize: "14px", fontWeight: 600 }}>
                {isNew ? "Добавление сервера в каталог" : "Редактирование сервера"}
              </h3>
              <p className="win-modal-desc" style={{ fontSize: "12px", color: "var(--win-text-tertiary)" }}>
                {isNew
                  ? "Подключение уже работающего сервера (Master-панель или Edge-нода)"
                  : "Обновление учетных данных и параметров подключения SSH"}
              </p>
            </div>
          </div>
          <WinTooltip content="Закрыть" shortcut="Esc">
            <button className="win-modal-close" onClick={close}>
              <X size={14} />
            </button>
          </WinTooltip>
        </div>

        {/* Scrollable Body */}
        <div
          className="win-modal-body"
          style={{
            flex: 1,
            overflowY: "auto",
            padding: "20px 24px",
            display: "flex",
            flexDirection: "column",
            gap: "16px",
          }}
        >
          {/* Section 1: Server Identity & Role */}
          <div className="modal-section-card">
            <div className="modal-section-title">
              <Server size={13} style={{ color: "var(--win-accent)" }} />
              <span>Конфигурация сервера</span>
            </div>

            <div className="form-group" style={{ margin: 0 }}>
              <label className="form-label">Понятное имя / Домен</label>
              <input
                type="text"
                className="win-textbox"
                value={data.name}
                onChange={(e) => onChange({ ...data, name: e.target.value })}
                placeholder="panel.example.com или Production VPS"
              />
            </div>

            <div className="form-group" style={{ margin: 0 }}>
              <label className="form-label">Роль сервера в каталоге</label>
              <div className="win-segmented-group">
                <button
                  type="button"
                  className={`win-segmented-btn ${data.role === "master" ? "active master" : ""}`}
                  onClick={() => onChange({ ...data, role: "master" })}
                >
                  <Server size={14} />
                  <span>Master Panel</span>
                </button>
                <button
                  type="button"
                  className={`win-segmented-btn ${data.role === "node" ? "active node" : ""}`}
                  onClick={() => onChange({ ...data, role: "node" })}
                >
                  <Radio size={14} />
                  <span>Edge Node</span>
                </button>
              </div>
            </div>

            {data.role === "master" && (
              <div style={{ display: "flex", flexDirection: "column", gap: "10px", paddingTop: "2px" }}>
                <div className="form-group" style={{ margin: 0 }}>
                  <label className="form-label" style={{ display: "flex", alignItems: "center", gap: "6px" }}>
                    <Globe size={12} style={{ color: "var(--win-accent)" }} />
                    <span>URL веб-панели (необязательно)</span>
                  </label>
                  <input
                    type="text"
                    className="win-textbox"
                    value={data.webUrl || ""}
                    onChange={(e) => onChange({ ...data, webUrl: e.target.value.trim() })}
                    placeholder="https://panel.example.com"
                  />
                  <span style={{ fontSize: "11px", color: "var(--win-text-tertiary)", marginTop: "2px", display: "block" }}>
                    Если не указано, URL будет сформирован автоматически на основе домена или IP.
                  </span>
                </div>

                <div style={{ paddingTop: "2px" }}>
                  <label
                    style={{
                      display: "flex",
                      alignItems: "flex-start",
                      gap: "9px",
                      cursor: "pointer",
                      fontSize: "12.5px",
                      color: "var(--win-text-primary)",
                    }}
                  >
                    <input
                      type="checkbox"
                      checked={Boolean(data.isExternalMaster)}
                      onChange={(e) => onChange({ ...data, isExternalMaster: e.target.checked })}
                      style={{ marginTop: "3px" }}
                    />
                    <div>
                      <span style={{ fontWeight: 500 }}>Внешняя мастер-панель (установлена вручную)</span>
                      <span
                        style={{
                          display: "block",
                          fontSize: "11px",
                          color: "var(--win-text-tertiary)",
                          marginTop: "2px",
                        }}
                      >
                        Отметьте, если развернули Remnawave самостоятельно без мастера установки.
                      </span>
                    </div>
                  </label>
                </div>
              </div>
            )}

            {data.role === "node" && (
              <div style={{ display: "flex", flexDirection: "column", gap: "10px", paddingTop: "2px" }}>
                <div className="form-group" style={{ margin: 0 }}>
                  <label className="form-label">Привязана к Мастер-панели</label>
                  <WinSelect
                    value={data.parentMasterId || ""}
                    onChange={(val) => onChange({ ...data, parentMasterId: val })}
                    options={[
                      { value: "", label: "Не привязана (Автономная нода)" },
                      ...availableMasters.map((m) => ({
                        value: m.id,
                        label: `${m.label || m.host} (${m.host})`,
                      })),
                    ]}
                    style={{ width: "100%" }}
                  />
                </div>

                <div style={{ display: "grid", gridTemplateColumns: "110px minmax(0, 1fr)", gap: "10px" }}>
                  <div className="form-group" style={{ margin: 0 }}>
                    <label className="form-label" style={{ display: "flex", alignItems: "center", gap: "5px" }}>
                      <MapPin size={12} style={{ color: "var(--win-accent)" }} />
                      <span>Локация</span>
                    </label>
                    <input
                      type="text"
                      className="win-textbox"
                      maxLength={5}
                      value={data.locationCode || ""}
                      onChange={(e) => onChange({ ...data, locationCode: e.target.value.toUpperCase().trim() })}
                      placeholder="NL, DE, US"
                    />
                  </div>

                  <div className="form-group" style={{ margin: 0 }}>
                    <label className="form-label">Decoy SNI домен (Маскировка Reality)</label>
                    <input
                      type="text"
                      className="win-textbox"
                      value={data.decoyDomain || ""}
                      onChange={(e) => onChange({ ...data, decoyDomain: e.target.value.trim() })}
                      placeholder="decoy.domain.com"
                    />
                  </div>
                </div>
              </div>
            )}
          </div>

          {/* Section 2: SSH Connection Settings */}
          <div className="modal-section-card">
            <div className="modal-section-title">
              <Lock size={13} style={{ color: "var(--win-accent)" }} />
              <span>Параметры подключения SSH</span>
            </div>

            <div style={{ display: "grid", gridTemplateColumns: "minmax(0, 1fr) 90px", gap: "12px", width: "100%", boxSizing: "border-box" }}>
              <div className="form-group" style={{ margin: 0, minWidth: 0 }}>
                <label className="form-label">IP-адрес или Hostname *</label>
                <input
                  type="text"
                  className="win-textbox"
                  style={{ width: "100%", minWidth: 0, boxSizing: "border-box" }}
                  value={data.host}
                  onChange={(e) => {
                    onChange({ ...data, host: e.target.value.trim() });
                    setSshTestResult(null);
                  }}
                  placeholder="81.90.25.120"
                />
              </div>

              <div className="form-group" style={{ margin: 0, minWidth: 0, maxWidth: "90px" }}>
                <label className="form-label">SSH Порт</label>
                <input
                  type="number"
                  className="win-textbox"
                  style={{ width: "100%", maxWidth: "90px", minWidth: 0, boxSizing: "border-box" }}
                  value={data.port}
                  onChange={(e) => {
                    onChange({ ...data, port: parseInt(e.target.value) || 22 });
                    setSshTestResult(null);
                  }}
                  placeholder="22"
                />
              </div>
            </div>

            <div className="form-group" style={{ margin: 0 }}>
              <label className="form-label">Имя пользователя SSH</label>
              <input
                type="text"
                className="win-textbox"
                value={data.user}
                onChange={(e) => {
                  onChange({ ...data, user: e.target.value.trim() });
                  setSshTestResult(null);
                }}
                placeholder="root"
              />
            </div>

            <div className="form-group" style={{ margin: 0 }}>
              <label className="form-label">Метод авторизации</label>
              <div className="win-segmented-group">
                <button
                  type="button"
                  className={`win-segmented-btn ${data.authMethod === "password" ? "active" : ""}`}
                  onClick={() => {
                    onChange({ ...data, authMethod: "password" });
                    setSshTestResult(null);
                  }}
                >
                  <Key size={13} />
                  <span>Пароль root</span>
                </button>
                <button
                  type="button"
                  className={`win-segmented-btn ${data.authMethod === "key" ? "active" : ""}`}
                  onClick={() => {
                    onChange({ ...data, authMethod: "key" });
                    setSshTestResult(null);
                  }}
                >
                  <FileKey size={13} />
                  <span>SSH-ключ</span>
                </button>
              </div>
            </div>

            {data.authMethod === "password" ? (
              <div className="form-group" style={{ margin: 0 }}>
                <label className="form-label">Пароль root / sudo</label>
                <PasswordInput
                  value={data.password || ""}
                  onChange={(e) => {
                    onChange({ ...data, password: e.target.value });
                    setSshTestResult(null);
                  }}
                  placeholder="Введите пароль (будет сохранен в защищенном виде)"
                />
              </div>
            ) : (
              <div className="form-group" style={{ margin: 0 }}>
                <label className="form-label">Приватный SSH-ключ</label>
                <div style={{ display: "flex", gap: "8px", alignItems: "center" }}>
                  <input
                    type="text"
                    className="win-textbox"
                    style={{ flex: 1 }}
                    value={data.privateKey || ""}
                    onChange={(e) => {
                      onChange({ ...data, privateKey: e.target.value });
                      setSshTestResult(null);
                    }}
                    placeholder="Путь к id_rsa / id_ed25519"
                  />
                  <button
                    type="button"
                    className="win-button"
                    style={{ height: "32px", whiteSpace: "nowrap" }}
                    onClick={async () => {
                      const keyPath = await onSelectKeyFile();
                      if (keyPath) {
                        onChange({ ...data, privateKey: keyPath });
                        setSshTestResult(null);
                      }
                    }}
                  >
                    <Folder size={13} />
                    <span>Обзор...</span>
                  </button>
                </div>
              </div>
            )}

            {/* Live SSH & IP Block Test Action & Result */}
            {(onTestSSH || onCheckIPBlock) && (
              <div
                style={{
                  marginTop: "4px",
                  paddingTop: "12px",
                  borderTop: "1px dashed var(--win-card-stroke)",
                  display: "flex",
                  flexDirection: "column",
                  gap: "10px",
                }}
              >
                <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: "10px", flexWrap: "wrap" }}>
                  <div style={{ display: "flex", alignItems: "center", gap: "8px", flexWrap: "wrap" }}>
                    {onTestSSH && (
                      <button
                        type="button"
                        className="win-button"
                        disabled={testingSSH || !data.host}
                        onClick={handleTestSSH}
                        style={{ display: "flex", alignItems: "center", gap: "6px" }}
                      >
                        {testingSSH ? (
                          <RefreshCw size={13} className="animate-spin" />
                        ) : (
                          <Activity size={13} style={{ color: "var(--win-accent)" }} />
                        )}
                        <span>{testingSSH ? "Проверка SSH..." : "Тест SSH"}</span>
                      </button>
                    )}

                    {onCheckIPBlock && (
                      <button
                        type="button"
                        className="win-button"
                        disabled={checkingIP || !data.host}
                        onClick={handleCheckIPBlock}
                        style={{ display: "flex", alignItems: "center", gap: "6px" }}
                        title="Проверить доступность IP-адреса из РФ (проверка блокировки ТСПУ / РКН)"
                      >
                        {checkingIP ? (
                          <RefreshCw size={13} className="animate-spin" />
                        ) : (
                          <ShieldAlert size={13} style={{ color: "#38bdf8" }} />
                        )}
                        <span>{checkingIP ? "Диагностика ТСПУ..." : "Проверить IP в РФ"}</span>
                      </button>
                    )}
                  </div>

                  {sshTestResult && (
                    <div
                      style={{
                        display: "inline-flex",
                        alignItems: "center",
                        gap: "6px",
                        fontSize: "12px",
                        padding: "4px 10px",
                        borderRadius: "4px",
                        background: sshTestResult.success ? "rgba(50, 205, 50, 0.12)" : "rgba(255, 69, 58, 0.12)",
                        border: `1px solid ${sshTestResult.success ? "rgba(50, 205, 50, 0.3)" : "rgba(255, 69, 58, 0.3)"}`,
                        color: sshTestResult.success ? "var(--win-success, #4ade80)" : "var(--win-danger, #f87171)",
                      }}
                    >
                      {sshTestResult.success ? (
                        <>
                          <CheckCircle2 size={13} />
                          <span>Доступно: {sshTestResult.os || "Linux"} ({sshTestResult.cpu || "CPU"} / {sshTestResult.ram || "RAM"})</span>
                        </>
                      ) : (
                        <>
                          <AlertCircle size={13} />
                          <span>Ошибка SSH: {sshTestResult.error || "Не удалось подключиться"}</span>
                        </>
                      )}
                    </div>
                  )}
                </div>

                {/* Render IP Check Result Card if available */}
                {ipCheckResult && (
                  <div className={`ip-check-card ${ipCheckResult.status}`}>
                    <div className="ip-check-header">
                      <div className="ip-check-verdict-group">
                        <span className={`ip-check-badge ${ipCheckResult.status}`}>
                          {ipCheckResult.status === "clean" ? (
                            <>
                              <CheckCircle2 size={12} />
                              <span>Чистый IP · Доступен в РФ</span>
                            </>
                          ) : ipCheckResult.status === "blocked" ? (
                            <>
                              <AlertTriangle size={12} />
                              <span>Блокировка ТСПУ / РКН</span>
                            </>
                          ) : ipCheckResult.status === "partial" ? (
                            <>
                              <AlertCircle size={12} />
                              <span>Частичная доступность</span>
                            </>
                          ) : (
                            <>
                              <AlertOctagon size={12} />
                              <span>Сервер недоступен</span>
                            </>
                          )}
                        </span>
                        <span style={{ fontSize: "12px", fontWeight: 600, color: "var(--win-text-primary)" }}>
                          {ipCheckResult.host}:{ipCheckResult.port}
                        </span>
                      </div>

                      <div style={{ display: "flex", alignItems: "center", gap: "6px" }}>
                        {ipCheckResult.isp_info && (
                          <span className="server-chip" style={{ fontSize: "11px" }}>
                            <Server size={10} />
                            <span>{ipCheckResult.isp_info}</span>
                          </span>
                        )}
                        {ipCheckResult.is_vpn_detected && ipCheckResult.client_country && (
                          <span
                            className="server-chip"
                            style={{
                              fontSize: "11px",
                              color: "#38bdf8",
                              borderColor: "rgba(56, 189, 248, 0.35)",
                              background: "rgba(56, 189, 248, 0.08)",
                            }}
                            title={`Ваш ПК подключен через VPN (${ipCheckResult.client_country})`}
                          >
                            <Globe size={10} />
                            <span>VPN: {ipCheckResult.client_country}</span>
                          </span>
                        )}
                        <button
                          type="button"
                          className="win-button"
                          style={{ height: "22px", padding: "0 6px", fontSize: "11px" }}
                          onClick={() => setIpCheckResult(null)}
                          title="Закрыть отчет"
                        >
                          <X size={11} />
                        </button>
                      </div>
                    </div>

                    <p className="ip-check-summary">
                      {ipCheckResult.summary}
                    </p>

                    {ipCheckResult.port_details && ipCheckResult.port_details.length > 0 ? (
                      <div style={{ display: "flex", flexDirection: "column", gap: "8px", marginTop: "8px" }}>
                        {ipCheckResult.port_details.map((pd: any, pdIdx: number) => (
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
                    ) : ipCheckResult.probes && ipCheckResult.probes.length > 0 ? (
                      <div className="ip-probe-pills-row">
                        {ipCheckResult.probes.map((pr: any, pIdx: number) => {
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
                )}
              </div>
            )}
          </div>
        </div>

        {/* Footer */}
        <div
          className="win-modal-footer"
          style={{
            flexShrink: 0,
            padding: "12px 24px",
            background: "var(--win-card-row-hover)",
            borderTop: "1px solid var(--win-card-stroke)",
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
          }}
        >
          <button className="win-button" onClick={close}>
            Отмена
          </button>
          <button className="win-button primary" onClick={onSave}>
            {isNew ? <Plus size={14} /> : <Check size={14} />}
            <span>{isNew ? "Добавить в каталог" : "Сохранить изменения"}</span>
          </button>
        </div>
        </>
      )}
    </WinModal>
  );
};
