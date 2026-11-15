import React, { useState, ReactNode } from "react";
import { ChevronDown } from "lucide-react";

export interface SettingsExpanderItem {
  icon?: ReactNode;
  title: ReactNode;
  description?: ReactNode;
  action?: ReactNode;
}

export interface SettingsExpanderProps {
  icon?: ReactNode;
  iconBg?: string;
  iconColor?: string;
  title: ReactNode;
  description?: ReactNode;
  action?: ReactNode;
  defaultExpanded?: boolean;
  expanded?: boolean;
  onExpandedChange?: (expanded: boolean) => void;
  items?: SettingsExpanderItem[];
  children?: ReactNode;
  className?: string;
  style?: React.CSSProperties;
}

export const SettingsExpander: React.FC<SettingsExpanderProps> = ({
  icon,
  iconBg,
  iconColor,
  title,
  description,
  action,
  defaultExpanded = false,
  expanded: controlledExpanded,
  onExpandedChange,
  items,
  children,
  className = "",
  style,
}) => {
  const [internalExpanded, setInternalExpanded] = useState(defaultExpanded);
  const isExpanded = controlledExpanded !== undefined ? controlledExpanded : internalExpanded;

  const handleToggle = () => {
    const next = !isExpanded;
    if (controlledExpanded === undefined) {
      setInternalExpanded(next);
    }
    if (onExpandedChange) {
      onExpandedChange(next);
    }
  };

  return (
    <div
      className={`win-settings-expander ${isExpanded ? "expanded" : ""} ${className}`}
      style={style}
    >
      <div className="win-settings-expander-header" onClick={handleToggle}>
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

        <div className="win-settings-expander-actions" onClick={(e) => e.stopPropagation()}>
          {action && <div className="win-settings-card-action">{action}</div>}
          <button
            type="button"
            className="win-settings-expander-chevron-btn"
            title={isExpanded ? "Свернуть" : "Развернуть"}
            onClick={handleToggle}
          >
            <ChevronDown
              size={16}
              className={`win-settings-chevron ${isExpanded ? "rotate-180" : ""}`}
            />
          </button>
        </div>
      </div>

      <div className="win-settings-expander-body">
        <div className="win-settings-expander-tray">
          {items && items.map((item, idx) => (
            <div key={idx} className="win-settings-expander-item">
              <div className="win-settings-card-info">
                {item.icon && (
                  <div className="win-settings-card-icon" style={{ width: "28px", height: "28px", minWidth: "28px" }}>
                    {item.icon}
                  </div>
                )}
                <div className="win-settings-card-text">
                  <div className="win-settings-card-title" style={{ fontSize: "12.5px" }}>{item.title}</div>
                  {item.description && (
                    <div className="win-settings-card-desc" style={{ fontSize: "11px" }}>{item.description}</div>
                  )}
                </div>
              </div>
              {item.action && <div className="win-settings-card-action">{item.action}</div>}
            </div>
          ))}
          {children}
        </div>
      </div>
    </div>
  );
};
