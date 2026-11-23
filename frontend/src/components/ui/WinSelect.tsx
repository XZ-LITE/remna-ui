import React, { useState, useRef, useEffect, useCallback } from "react";
import { ChevronDown, Check } from "lucide-react";

export interface WinSelectOption {
  value: string;
  label: string;
  icon?: React.ReactNode;
  badge?: string;
  description?: string;
}

export interface WinSelectProps {
  value: string;
  options: WinSelectOption[];
  onChange: (value: string) => void;
  placeholder?: string;
  disabled?: boolean;
  style?: React.CSSProperties;
  className?: string;
  minMenuWidth?: string;
}

export const WinSelect: React.FC<WinSelectProps> = ({
  value,
  options,
  onChange,
  placeholder = "Выберите значение...",
  disabled = false,
  style,
  className = "",
  minMenuWidth,
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const [focusedIndex, setFocusedIndex] = useState<number>(-1);
  const containerRef = useRef<HTMLDivElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);

  const selectedOption = options.find((o) => o.value === value);

  // Close when clicking outside
  useEffect(() => {
    if (!isOpen) return;

    const handleClickOutside = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setIsOpen(false);
      }
    };

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        setIsOpen(false);
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [isOpen]);

  // Keep focused item visible on keyboard nav
  useEffect(() => {
    if (isOpen && focusedIndex >= 0 && menuRef.current) {
      const items = menuRef.current.querySelectorAll(".win-select-option");
      if (items[focusedIndex]) {
        (items[focusedIndex] as HTMLElement).scrollIntoView({ block: "nearest" });
      }
    }
  }, [focusedIndex, isOpen]);

  const handleSelect = (val: string) => {
    onChange(val);
    setIsOpen(false);
  };

  const handleTriggerKeyDown = (e: React.KeyboardEvent) => {
    if (disabled) return;

    if (e.key === "Enter" || e.key === " " || e.key === "ArrowDown") {
      e.preventDefault();
      if (!isOpen) {
        setIsOpen(true);
        const idx = options.findIndex((o) => o.value === value);
        setFocusedIndex(idx >= 0 ? idx : 0);
      } else if (e.key === "ArrowDown") {
        setFocusedIndex((prev) => (prev < options.length - 1 ? prev + 1 : 0));
      }
    } else if (e.key === "ArrowUp" && isOpen) {
      e.preventDefault();
      setFocusedIndex((prev) => (prev > 0 ? prev - 1 : options.length - 1));
    } else if (e.key === "Enter" && isOpen && focusedIndex >= 0) {
      e.preventDefault();
      if (options[focusedIndex]) {
        handleSelect(options[focusedIndex].value);
      }
    }
  };

  return (
    <div
      ref={containerRef}
      className={`win-select-container ${disabled ? "disabled" : ""} ${className}`}
      style={{
        position: "relative",
        display: "inline-block",
        width: style?.width || "auto",
        minWidth: style?.minWidth || "auto",
        ...style,
      }}
    >
      {/* Trigger Button */}
      <button
        type="button"
        disabled={disabled}
        className={`win-select-trigger ${isOpen ? "open" : ""}`}
        onClick={() => !disabled && setIsOpen(!isOpen)}
        onKeyDown={handleTriggerKeyDown}
        aria-haspopup="listbox"
        aria-expanded={isOpen}
      >
        <span className="win-select-label" title={selectedOption?.label || placeholder}>
          {selectedOption ? (
            <span style={{ display: "inline-flex", alignItems: "center", gap: "6px" }}>
              {selectedOption.icon}
              <span>{selectedOption.label}</span>
            </span>
          ) : (
            <span style={{ color: "var(--win-text-disabled)" }}>{placeholder}</span>
          )}
        </span>

        <ChevronDown
          size={14}
          className={`win-select-chevron ${isOpen ? "open" : ""}`}
        />
      </button>

      {/* Dropdown Menu Popup */}
      {isOpen && (
        <div
          ref={menuRef}
          className="win-select-menu"
          role="listbox"
          style={{ minWidth: minMenuWidth || "100%" }}
        >
          {options.map((opt, idx) => {
            const isSelected = opt.value === value;
            const isFocused = idx === focusedIndex;

            return (
              <div
                key={opt.value}
                role="option"
                aria-selected={isSelected}
                className={`win-select-option ${isSelected ? "selected" : ""} ${isFocused ? "focused" : ""}`}
                onClick={() => handleSelect(opt.value)}
                onMouseEnter={() => setFocusedIndex(idx)}
              >
                <div style={{ display: "flex", alignItems: "center", gap: "8px", flex: 1, minWidth: 0 }}>
                  {opt.icon && <span style={{ flexShrink: 0 }}>{opt.icon}</span>}
                  <div style={{ display: "flex", flexDirection: "column", minWidth: 0 }}>
                    <span className="win-select-option-text">{opt.label}</span>
                    {opt.description && (
                      <span className="win-select-option-desc">{opt.description}</span>
                    )}
                  </div>
                  {opt.badge && (
                    <span className="win-select-option-badge">{opt.badge}</span>
                  )}
                </div>

                {isSelected && (
                  <Check size={13} className="win-select-check" />
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
};
