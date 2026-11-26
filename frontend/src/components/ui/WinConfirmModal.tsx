import React from "react";
import { AlertTriangle, AlertOctagon, HelpCircle, X, RefreshCw } from "lucide-react";
import { WinModal } from "./WinModal";
import { WinTooltip } from "./WinTooltip";

export interface WinConfirmModalProps {
  isOpen: boolean;
  title: string;
  description: React.ReactNode;
  details?: React.ReactNode;
  confirmLabel?: string;
  cancelLabel?: string;
  variant?: "primary" | "warning" | "danger";
  loading?: boolean;
  onConfirm: () => void | Promise<void>;
  onClose: () => void;
}

export const WinConfirmModal: React.FC<WinConfirmModalProps> = ({
  isOpen,
  title,
  description,
  details,
  confirmLabel = "Подтвердить",
  cancelLabel = "Отмена",
  variant = "warning",
  loading = false,
  onConfirm,
  onClose,
}) => {
  const iconStyle =
    variant === "danger"
      ? { background: "rgba(239, 68, 68, 0.14)", color: "#f87171", border: "1px solid rgba(239, 68, 68, 0.28)" }
      : variant === "warning"
      ? { background: "rgba(245, 158, 11, 0.14)", color: "#fbbf24", border: "1px solid rgba(245, 158, 11, 0.28)" }
      : { background: "rgba(96, 205, 255, 0.14)", color: "var(--win-accent)", border: "1px solid rgba(96, 205, 255, 0.28)" };

  const confirmStyle: React.CSSProperties =
    variant === "danger"
      ? { background: "#dc2626", borderColor: "#ef4444", color: "#ffffff", fontWeight: 600 }
      : variant === "warning"
      ? { background: "#d97706", borderColor: "#f59e0b", color: "#ffffff", fontWeight: 600 }
      : {};

  return (
    <WinModal
      isOpen={isOpen}
      onClose={onClose}
      maxWidth="460px"
      width="94%"
      preventBackdropClose={loading}
    >
      {({ close }) => (
        <>
          <div className="win-modal-header">
            <div style={{ display: "flex", alignItems: "center", gap: "10px" }}>
              <div className="modal-title-icon" style={iconStyle}>
                {variant === "danger" ? (
                  <AlertOctagon size={17} />
                ) : variant === "warning" ? (
                  <AlertTriangle size={17} />
                ) : (
                  <HelpCircle size={17} />
                )}
              </div>
              <div className="win-modal-title">{title}</div>
            </div>
            <WinTooltip content="Закрыть" shortcut="Esc">
              <button
                type="button"
                className="win-modal-close"
                disabled={loading}
                onClick={() => !loading && close()}
              >
                <X size={14} />
              </button>
            </WinTooltip>
          </div>

          <div
            className="win-modal-body"
            style={{ padding: "16px 20px", display: "flex", flexDirection: "column", gap: "12px" }}
          >
            <div style={{ fontSize: "12.5px", color: "var(--win-text-primary)", lineHeight: 1.5 }}>
              {description}
            </div>
            {details && (
              <div
                style={{
                  padding: "10px 12px",
                  borderRadius: "6px",
                  background: "var(--win-fill-card-secondary)",
                  border: "1px solid var(--win-card-stroke)",
                  fontSize: "12px",
                  color: "var(--win-text-secondary)",
                  lineHeight: 1.45,
                }}
              >
                {details}
              </div>
            )}
          </div>

          <div className="win-modal-footer">
            <button
              type="button"
              className="win-button"
              disabled={loading}
              onClick={close}
            >
              {cancelLabel}
            </button>
            <button
              type="button"
              className={`win-button ${variant === "primary" ? "primary" : ""}`}
              style={confirmStyle}
              disabled={loading}
              onClick={onConfirm}
            >
              {loading && <RefreshCw size={13} className="animate-spin" />}
              <span>{confirmLabel}</span>
            </button>
          </div>
        </>
      )}
    </WinModal>
  );
};
