import React, { useState } from "react";
import { AlertTriangle, X, RefreshCw, RotateCcw, Database, FileText, Layers, ShieldCheck } from "lucide-react";
import * as types from "../../../bindings/remna-ui/internal/types/models";
import { WinModal } from "../ui/WinModal";
import { WinTooltip } from "../ui/WinTooltip";

interface RestoreBackupModalProps {
  server: types.SavedServer;
  fileName: string;
  loading: boolean;
  onClose: () => void;
  onConfirm: (scope: string, createSafetyBackup: boolean) => void;
}

export const RestoreBackupModal: React.FC<RestoreBackupModalProps> = ({
  server,
  fileName,
  loading,
  onClose,
  onConfirm,
}) => {
  const [scope, setScope] = useState<string>("all");
  const [createSafetyBackup, setCreateSafetyBackup] = useState<boolean>(true);

  const handleConfirm = () => {
    onConfirm(scope, createSafetyBackup);
  };

  return (
    <WinModal
      isOpen={true}
      onClose={onClose}
      maxWidth="560px"
      width="95%"
      preventBackdropClose={loading}
    >
      {({ close }) => (
        <>
      <div className="win-modal-header">
          <div style={{ display: "flex", alignItems: "center", gap: "10px" }}>
            <div className="modal-title-icon" style={{ background: "rgba(251,191,36,0.15)", color: "#fbbf24" }}>
              <RotateCcw size={18} />
            </div>
            <div>
              <h3 className="win-modal-title">Безопасное восстановление из архива</h3>
              <p className="win-modal-desc">
                Изолированный накат данных и конфигураций для {server.label || server.host}
              </p>
            </div>
          </div>
          <WinTooltip content="Закрыть" shortcut="Esc">
            <button
              className="win-modal-close"
              disabled={loading}
              onClick={close}
            >
              <X size={15} />
            </button>
          </WinTooltip>
        </div>

        <div className="win-modal-body" style={{ display: "flex", flexDirection: "column", gap: "14px", padding: "18px 22px" }}>
          <div>
            <div style={{ fontSize: "12px", color: "var(--win-text-secondary)", marginBottom: "4px" }}>
              Выбранный архив резервной копии:
            </div>
            <div style={{ background: "var(--win-fill-card-secondary)", padding: "10px 14px", borderRadius: "6px", border: "1px solid var(--win-card-stroke)", fontFamily: "monospace", fontSize: "12px", color: "var(--win-accent)", wordBreak: "break-all" }}>
              {fileName}
            </div>
          </div>

          {/* Scope Selection */}
          <div>
            <div style={{ fontSize: "12px", fontWeight: 600, color: "var(--win-text-primary)", marginBottom: "8px" }}>
              Выберите область восстановления (Scope):
            </div>
            <div style={{ display: "flex", flexDirection: "column", gap: "8px" }}>
              <label
                style={{
                  display: "flex",
                  alignItems: "flex-start",
                  gap: "10px",
                  padding: "10px 12px",
                  borderRadius: "6px",
                  border: scope === "all" ? "1px solid var(--win-accent)" : "1px solid var(--win-card-stroke)",
                  background: scope === "all" ? "rgba(0, 120, 212, 0.08)" : "var(--win-card-bg)",
                  cursor: "pointer",
                }}
              >
                <input
                  type="radio"
                  name="restoreScope"
                  value="all"
                  checked={scope === "all"}
                  onChange={() => setScope("all")}
                  disabled={loading}
                  style={{ marginTop: "3px" }}
                />
                <div style={{ flex: 1 }}>
                  <div style={{ display: "flex", alignItems: "center", gap: "6px", fontSize: "13px", fontWeight: 600, color: "var(--win-text-primary)" }}>
                    <Layers size={14} style={{ color: "var(--win-accent)" }} />
                    <span>Полное восстановление (Рекомендуется)</span>
                  </div>
                  <div style={{ fontSize: "11.5px", color: "var(--win-text-secondary)", marginTop: "2px", lineHeight: "1.4" }}>
                    Восстанавливает базу данных PostgreSQL, все файлы конфигураций (.env, compose, SSL) и медиафайлы uploads/.
                  </div>
                </div>
              </label>

              <label
                style={{
                  display: "flex",
                  alignItems: "flex-start",
                  gap: "10px",
                  padding: "10px 12px",
                  borderRadius: "6px",
                  border: scope === "db_only" ? "1px solid #10b981" : "1px solid var(--win-card-stroke)",
                  background: scope === "db_only" ? "rgba(16, 185, 129, 0.08)" : "var(--win-card-bg)",
                  cursor: "pointer",
                }}
              >
                <input
                  type="radio"
                  name="restoreScope"
                  value="db_only"
                  checked={scope === "db_only"}
                  onChange={() => setScope("db_only")}
                  disabled={loading}
                  style={{ marginTop: "3px" }}
                />
                <div style={{ flex: 1 }}>
                  <div style={{ display: "flex", alignItems: "center", gap: "6px", fontSize: "13px", fontWeight: 600, color: "var(--win-text-primary)" }}>
                    <Database size={14} style={{ color: "#10b981" }} />
                    <span>Только база данных (Database Only)</span>
                  </div>
                  <div style={{ fontSize: "11.5px", color: "var(--win-text-secondary)", marginTop: "2px", lineHeight: "1.4" }}>
                    Восстанавливает только дамп PostgreSQL с предварительной очисткой схемы. Текущие файлы конфигурации не изменяются.
                  </div>
                </div>
              </label>

              <label
                style={{
                  display: "flex",
                  alignItems: "flex-start",
                  gap: "10px",
                  padding: "10px 12px",
                  borderRadius: "6px",
                  border: scope === "configs_only" ? "1px solid #c084fc" : "1px solid var(--win-card-stroke)",
                  background: scope === "configs_only" ? "rgba(192, 132, 252, 0.08)" : "var(--win-card-bg)",
                  cursor: "pointer",
                }}
              >
                <input
                  type="radio"
                  name="restoreScope"
                  value="configs_only"
                  checked={scope === "configs_only"}
                  onChange={() => setScope("configs_only")}
                  disabled={loading}
                  style={{ marginTop: "3px" }}
                />
                <div style={{ flex: 1 }}>
                  <div style={{ display: "flex", alignItems: "center", gap: "6px", fontSize: "13px", fontWeight: 600, color: "var(--win-text-primary)" }}>
                    <FileText size={14} style={{ color: "#c084fc" }} />
                    <span>Только конфигурации и сертификаты</span>
                  </div>
                  <div style={{ fontSize: "11.5px", color: "var(--win-text-secondary)", marginTop: "2px", lineHeight: "1.4" }}>
                    Восстанавливает файлы .env, docker-compose.yml, SSL-сертификаты и uploads/. База данных остается нетронутой.
                  </div>
                </div>
              </label>
            </div>
          </div>

          {/* Safety Backup Checkbox */}
          <label
            style={{
              display: "flex",
              alignItems: "center",
              gap: "8px",
              padding: "10px 12px",
              background: "rgba(34, 197, 94, 0.08)",
              border: "1px solid rgba(34, 197, 94, 0.25)",
              borderRadius: "6px",
              cursor: "pointer",
            }}
          >
            <input
              type="checkbox"
              checked={createSafetyBackup}
              onChange={(e) => setCreateSafetyBackup(e.target.checked)}
              disabled={loading}
            />
            <div style={{ display: "flex", alignItems: "center", gap: "6px", fontSize: "12px", color: "var(--win-text-primary)" }}>
              <ShieldCheck size={14} style={{ color: "#22c55e" }} />
              <span>Создать страховочный снимок текущей системы перед восстановлением</span>
            </div>
          </label>

          {/* Warning */}
          <div style={{ display: "flex", gap: "10px", padding: "10px 12px", background: "var(--win-status-danger-bg)", border: "1px solid rgba(239,68,68,0.28)", borderRadius: "6px", fontSize: "11.5px", color: "var(--win-status-danger)", lineHeight: "1.4" }}>
            <AlertTriangle size={16} style={{ flexShrink: 0, marginTop: "2px" }} />
            <div>
              Во время восстановления контейнеры стека будут временно остановлены для безопасного разрыва активных соединений и предотвращения конфликтов в базе данных.
            </div>
          </div>
        </div>

        <div className="win-modal-footer">
          <button
            className="win-button"
            disabled={loading}
            onClick={close}
          >
            Отмена
          </button>
          <button
            className="win-button"
            style={{ background: "#dc2626", color: "#ffffff", borderColor: "#ef4444" }}
            disabled={loading}
            onClick={handleConfirm}
          >
            {loading ? (
              <RefreshCw size={13} className="animate-spin" />
            ) : (
              <RotateCcw size={13} />
            )}
            <span>{loading ? "Выполняется восстановление..." : "Восстановить"}</span>
          </button>
        </div>
        </>
      )}
    </WinModal>
  );
};
