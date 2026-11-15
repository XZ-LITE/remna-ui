import React from "react";
import { AlertTriangle, X, RefreshCw, RotateCcw } from "lucide-react";
import { types } from "../../../wailsjs/go/models";

interface RestoreBackupModalProps {
  server: types.SavedServer;
  fileName: string;
  loading: boolean;
  onClose: () => void;
  onConfirm: () => void;
}

export const RestoreBackupModal: React.FC<RestoreBackupModalProps> = ({
  server,
  fileName,
  loading,
  onClose,
  onConfirm,
}) => {
  return (
    <div className="win-modal-overlay" onClick={() => !loading && onClose()}>
      <div
        className="win-modal-content"
        style={{ maxWidth: "520px" }}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="win-modal-header">
          <div style={{ display: "flex", alignItems: "center", gap: "10px" }}>
            <div className="modal-title-icon" style={{ background: "rgba(251,191,36,0.15)", color: "#fbbf24" }}>
              <AlertTriangle size={18} />
            </div>
            <div>
              <h3 className="win-modal-title">Подтверждение восстановления</h3>
              <p className="win-modal-desc">
                Откат конфигураций и базы данных из резервной копии
              </p>
            </div>
          </div>
          <button
            className="win-modal-close"
            disabled={loading}
            onClick={onClose}
          >
            <X size={15} />
          </button>
        </div>

        <div className="win-modal-body" style={{ display: "flex", flexDirection: "column", gap: "12px" }}>
          <div style={{ fontSize: "13px", lineHeight: "1.5", color: "var(--win-text-primary)" }}>
            Вы собираетесь восстановить сервер <strong>{server.label || server.host}</strong> из архива:
          </div>
          <div style={{ background: "rgba(255,255,255,0.04)", padding: "10px 14px", borderRadius: "6px", border: "1px solid var(--win-card-stroke)", fontFamily: "monospace", fontSize: "12px", color: "#60cdff" }}>
            {fileName}
          </div>
          <div style={{ display: "flex", gap: "10px", padding: "10px 12px", background: "rgba(239,68,68,0.1)", border: "1px solid rgba(239,68,68,0.25)", borderRadius: "6px", fontSize: "11.5px", color: "#fca5a5", lineHeight: "1.4" }}>
            <AlertTriangle size={16} style={{ flexShrink: 0, marginTop: "2px" }} />
            <div>
              Восстановление перезапишет конфигурации <code>.env</code>, <code>docker-compose.yml</code>, SSL-сертификаты и базу данных PostgreSQL (если архив содержит дамп), после чего перезапустит контейнеры Remnawave.
            </div>
          </div>
        </div>

        <div className="win-modal-footer">
          <button
            className="win-button"
            disabled={loading}
            onClick={onClose}
          >
            Отмена
          </button>
          <button
            className="win-button"
            style={{ background: "#dc2626", color: "#ffffff", borderColor: "#ef4444" }}
            disabled={loading}
            onClick={onConfirm}
          >
            {loading ? (
              <RefreshCw size={13} className="animate-spin" />
            ) : (
              <RotateCcw size={13} />
            )}
            <span>Да, восстановить из архива</span>
          </button>
        </div>
      </div>
    </div>
  );
};
