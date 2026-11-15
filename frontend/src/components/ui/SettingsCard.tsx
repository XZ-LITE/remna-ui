import React, { ReactNode } from "react";

export interface SettingsCardProps {
  icon?: ReactNode;
  iconBg?: string;
  iconColor?: string;
  title: ReactNode;
  description?: ReactNode;
  action?: ReactNode;
  onClick?: () => void;
  className?: string;
  style?: React.CSSProperties;
  children?: ReactNode;
}

export const SettingsCard: React.FC<SettingsCardProps> = ({
  icon,
  iconBg,
  iconColor,
  title,
  description,
  action,
  onClick,
  className = "",
  style,
  children,
}) => {
  return (
    <div
      className={`win-settings-card ${onClick ? "clickable" : ""} ${className}`}
      onClick={onClick}
      style={style}
    >
      <div className="win-settings-card-main">
        <div className="win-settings-card-info">
          {icon && (
            <div
              className="win-settings-card-icon"
              style={{
                backgroundColor: iconBg || "var(--win-icon-bg)",
                borderColor: iconBg ? "transparent" : "var(--win-icon-stroke)",
                color: iconColor || "var(--win-accent)",
              }}
            >
              {icon}
            </div>
          )}
          <div className="win-settings-card-text">
            <div className="win-settings-card-title">{title}</div>
            {description && (
              <div className="win-settings-card-desc">{description}</div>
            )}
          </div>
        </div>

        {action && (
          <div
            className="win-settings-card-action"
            onClick={(e) => onClick && e.stopPropagation()}
          >
            {action}
          </div>
        )}
      </div>

      {children && (
        <div className="win-settings-card-children">
          {children}
        </div>
      )}
    </div>
  );
};
