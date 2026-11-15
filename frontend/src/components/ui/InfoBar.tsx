import React, { ReactNode, CSSProperties } from "react";
import { Info, CheckCircle2, AlertTriangle, AlertCircle, X } from "lucide-react";

export type InfoBarSeverity = "info" | "success" | "warning" | "error";

export interface InfoBarProps {
  severity?: InfoBarSeverity;
  title?: string;
  message: ReactNode;
  action?: ReactNode;
  closable?: boolean;
  onClose?: () => void;
  className?: string;
  style?: CSSProperties;
}

export const InfoBar: React.FC<InfoBarProps> = ({
  severity = "info",
  title,
  message,
  action,
  closable = false,
  onClose,
  className = "",
  style = {},
}) => {
  const getIcon = () => {
    switch (severity) {
      case "success":
        return <CheckCircle2 size={16} className="win-infobar-icon success" />;
      case "warning":
        return <AlertTriangle size={16} className="win-infobar-icon warning" />;
      case "error":
        return <AlertCircle size={16} className="win-infobar-icon error" />;
      case "info":
      default:
        return <Info size={16} className="win-infobar-icon info" />;
    }
  };

  return (
    <div className={`win-infobar win-infobar-${severity} ${className}`} style={style} role="status">
      <div className="win-infobar-leading">
        {getIcon()}
        <div className="win-infobar-content">
          {title && <span className="win-infobar-title">{title}</span>}
          <div className="win-infobar-message">{message}</div>
        </div>
      </div>
      <div className="win-infobar-trailing">
        {action && <div className="win-infobar-action">{action}</div>}
        {closable && (
          <button
            type="button"
            className="win-infobar-close"
            onClick={onClose}
            aria-label="Закрыть уведомление"
          >
            <X size={14} />
          </button>
        )}
      </div>
    </div>
  );
};
