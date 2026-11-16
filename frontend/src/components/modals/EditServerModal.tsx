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
  MapPin,
} from "lucide-react";
import { types } from "../../../wailsjs/go/models";

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
  onTestSSH?: (host: string, port: number, user: string, pass: string, key: string) => Promise<types.SSHTestResult>;
}

export const EditServerModal: React.FC<EditServerModalProps> = ({
  data,
  availableMasters,
  onChange,
  onClose,
  onSave,
  onSelectKeyFile,
  onTestSSH,
}) => {
  const [showPassword, setShowPassword] = useState<boolean>(false);
  const [testingSSH, setTestingSSH] = useState<boolean>(false);
  const [sshTestResult, setSshTestResult] = useState<types.SSHTestResult | null>(null);

  const isNew = !data.server.host || data.server.last_seen === "Только что добавлен";

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
    <div className="win-modal-overlay" onClick={onClose}>
      <div
        className="win-modal-content"
        style={{
          maxWidth: "600px",
          width: "100%",
          maxHeight: "88vh",
          display: "flex",
          flexDirection: "column",
        }}
        onClick={(e) => e.stopPropagation()}
      >
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
          <button className="win-modal-close" onClick={onClose} title="Закрыть">
            <X size={14} />
          </button>
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
                  <select
                    className="win-select"
                    value={data.parentMasterId || ""}
                    onChange={(e) => onChange({ ...data, parentMasterId: e.target.value })}
                  >
                    <option value="">Не привязана (Автономная нода)</option>
                    {availableMasters.map((m) => (
                      <option key={m.id} value={m.id}>
                        {m.label || m.host} ({m.host})
                      </option>
                    ))}
                  </select>
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
                <div style={{ position: "relative", display: "flex", alignItems: "center" }}>
                  <input
                    type={showPassword ? "text" : "password"}
                    className="win-textbox"
                    style={{ width: "100%", paddingRight: "36px" }}
                    value={data.password || ""}
                    onChange={(e) => {
                      onChange({ ...data, password: e.target.value });
                      setSshTestResult(null);
                    }}
                    placeholder="Введите пароль (будет сохранен в защищенном виде)"
                  />
                  <button
                    type="button"
                    className="win-button"
                    style={{
                      position: "absolute",
                      right: "4px",
                      height: "24px",
                      width: "28px",
                      padding: 0,
                    }}
                    onClick={() => setShowPassword(!showPassword)}
                    title={showPassword ? "Скрыть пароль" : "Показать пароль"}
                  >
                    {showPassword ? <EyeOff size={13} /> : <Eye size={13} />}
                  </button>
                </div>
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

            {/* Live SSH Test Action & Result */}
            {onTestSSH && (
              <div
                style={{
                  marginTop: "4px",
                  paddingTop: "12px",
                  borderTop: "1px dashed var(--win-card-stroke)",
                  display: "flex",
                  flexDirection: "column",
                  gap: "8px",
                }}
              >
                <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: "10px", flexWrap: "wrap" }}>
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
                    <span>{testingSSH ? "Проверка SSH..." : "Проверить SSH соединение"}</span>
                  </button>

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
                          <span>Ошибка: {sshTestResult.error || "Не удалось подключиться"}</span>
                        </>
                      )}
                    </div>
                  )}
                </div>
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
          <button className="win-button" onClick={onClose}>
            Отмена
          </button>
          <button className="win-button primary" onClick={onSave}>
            {isNew ? <Plus size={14} /> : <Check size={14} />}
            <span>{isNew ? "Добавить в каталог" : "Сохранить изменения"}</span>
          </button>
        </div>
      </div>
    </div>
  );
};
