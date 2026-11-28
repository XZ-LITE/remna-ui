import React, { useState, useRef, useEffect, useLayoutEffect, useCallback } from "react";
import { createPortal } from "react-dom";
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
  const [isRendered, setIsRendered] = useState(false);
  const [isClosing, setIsClosing] = useState(false);
  const [focusedIndex, setFocusedIndex] = useState<number>(-1);
  const [menuCoords, setMenuCoords] = useState<{
    top?: number;
    bottom?: number;
    left: number;
    width: number;
    maxHeight: number;
    placement: "bottom" | "top";
  }>({ left: 0, width: 0, maxHeight: 280, placement: "bottom" });

  const containerRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);
  const closeTimeoutRef = useRef<number | null>(null);

  const selectedOption = options.find((o) => o.value === value);

  const closeMenu = useCallback(() => {
    setIsOpen(false);
    setIsClosing(true);
    if (closeTimeoutRef.current) {
      clearTimeout(closeTimeoutRef.current);
    }
    closeTimeoutRef.current = window.setTimeout(() => {
      setIsRendered(false);
      setIsClosing(false);
    }, 130);
  }, []);

  // Recalculate menu position relative to viewport
  const updatePosition = useCallback(() => {
    if (!triggerRef.current) return;
    const rect = triggerRef.current.getBoundingClientRect();

    // If trigger element is hidden, detached, or scrolled offscreen, close menu
    if (rect.width === 0 && rect.height === 0) {
      closeMenu();
      return;
    }
    if (rect.bottom < 0 || rect.top > window.innerHeight) {
      closeMenu();
      return;
    }

    // Check if nearest modal footer or main bottom-action-bar exists to prevent overlapping
    let bottomBoundary = window.innerHeight;
    const hostModal = triggerRef.current?.closest(".win-modal-dialog, .win-modal-content");
    const bottomBar = hostModal
      ? hostModal.querySelector(".win-modal-footer, .modal-footer")
      : document.querySelector(".bottom-action-bar");
    if (bottomBar) {
      const bRect = bottomBar.getBoundingClientRect();
      if (bRect.width > 0 && bRect.height > 0 && bRect.top > 0 && bRect.top < bottomBoundary) {
        bottomBoundary = bRect.top;
      }
    }

    const spaceBelow = bottomBoundary - rect.bottom;
    const spaceAbove = rect.top;
    const placement = spaceBelow < 200 && spaceAbove > spaceBelow ? "top" : "bottom";

    // Dynamic max-height so menu never spills past the viewport edges or bottom boundary
    const maxHeight = placement === "top"
      ? Math.min(280, Math.max(90, spaceAbove - 16))
      : Math.min(280, Math.max(90, spaceBelow - 12));

    // Menu width logic: minimum is trigger width, or minMenuWidth
    let targetWidth = rect.width;
    if (minMenuWidth) {
      const parsed = parseInt(minMenuWidth, 10);
      if (!isNaN(parsed) && parsed > targetWidth) {
        targetWidth = parsed;
      }
    }

    // Ensure menu does not exceed viewport width
    if (targetWidth > window.innerWidth - 24) {
      targetWidth = window.innerWidth - 24;
    }

    // Keep menu within viewport horizontally
    let left = rect.left;
    if (left + targetWidth > window.innerWidth - 12) {
      left = Math.max(12, window.innerWidth - targetWidth - 12);
    }
    if (left < 12) left = 12;

    if (placement === "top") {
      setMenuCoords({
        bottom: window.innerHeight - rect.top + 4,
        left,
        width: targetWidth,
        maxHeight,
        placement: "top",
      });
    } else {
      setMenuCoords({
        top: rect.bottom + 4,
        left,
        width: targetWidth,
        maxHeight,
        placement: "bottom",
      });
    }
  }, [minMenuWidth, closeMenu]);

  const openMenu = () => {
    if (disabled) return;
    if (closeTimeoutRef.current) {
      clearTimeout(closeTimeoutRef.current);
    }
    updatePosition();
    setIsClosing(false);
    setIsRendered(true);
    setIsOpen(true);
    const idx = options.findIndex((o) => o.value === value);
    setFocusedIndex(idx >= 0 ? idx : 0);
  };

  useLayoutEffect(() => {
    if (isOpen) {
      updatePosition();
    }
  }, [isOpen, updatePosition]);

  // Handle click outside, escape key, and scroll
  useEffect(() => {
    if (!isRendered) return;

    const handleClickOutside = (e: MouseEvent) => {
      const target = e.target as Node;
      if (
        triggerRef.current &&
        !triggerRef.current.contains(target) &&
        menuRef.current &&
        !menuRef.current.contains(target)
      ) {
        closeMenu();
      }
    };

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        e.stopPropagation();
        closeMenu();
        triggerRef.current?.focus();
      }
    };

    const handleScroll = (e: Event) => {
      const target = e.target;
      // If scroll happened inside the dropdown menu itself, let user scroll the options list
      if (menuRef.current && target instanceof Node && menuRef.current.contains(target)) {
        return;
      }
      // Ignore scroll events inside unrelated sibling panels (e.g. log console)
      if (
        target instanceof Node &&
        target !== document &&
        target !== document.documentElement &&
        triggerRef.current &&
        !target.contains(triggerRef.current)
      ) {
        return;
      }
      // Scrolling an ancestor of the trigger closes the dropdown
      closeMenu();
    };

    const handleResize = () => {
      closeMenu();
    };

    document.addEventListener("mousedown", handleClickOutside);
    document.addEventListener("keydown", handleKeyDown);
    window.addEventListener("resize", handleResize);
    window.addEventListener("scroll", handleScroll, true);

    return () => {
      if (closeTimeoutRef.current) {
        clearTimeout(closeTimeoutRef.current);
      }
      document.removeEventListener("mousedown", handleClickOutside);
      document.removeEventListener("keydown", handleKeyDown);
      window.removeEventListener("resize", handleResize);
      window.removeEventListener("scroll", handleScroll, true);
    };
  }, [isRendered, closeMenu]);

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
    closeMenu();
    triggerRef.current?.focus();
  };

  const handleTriggerKeyDown = (e: React.KeyboardEvent) => {
    if (disabled) return;

    if (!isOpen) {
      if (e.key === "Enter" || e.key === " " || e.key === "ArrowDown") {
        e.preventDefault();
        openMenu();
      }
      return;
    }

    if (e.key === "Escape") {
      e.preventDefault();
      closeMenu();
    } else if (e.key === "ArrowDown") {
      e.preventDefault();
      setFocusedIndex((prev) => (prev < options.length - 1 ? prev + 1 : 0));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setFocusedIndex((prev) => (prev > 0 ? prev - 1 : options.length - 1));
    } else if (e.key === "Enter" && focusedIndex >= 0) {
      e.preventDefault();
      if (options[focusedIndex]) {
        handleSelect(options[focusedIndex].value);
      }
    }
  };

  const menuDropdown = isRendered ? (
    <div
      ref={menuRef}
      className={`win-select-menu placement-${menuCoords.placement} ${isClosing ? "exiting" : "entering"}`}
      role="listbox"
      style={{
        position: "fixed",
        top: menuCoords.top !== undefined ? `${menuCoords.top}px` : "auto",
        bottom: menuCoords.bottom !== undefined ? `${menuCoords.bottom}px` : "auto",
        left: `${menuCoords.left}px`,
        width: `${menuCoords.width}px`,
        minWidth: minMenuWidth || `${menuCoords.width}px`,
        maxHeight: `${menuCoords.maxHeight}px`,
        zIndex: 15000,
      }}
      onMouseDown={(e) => e.preventDefault()}
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
  ) : null;

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
        ref={triggerRef}
        type="button"
        disabled={disabled}
        className={`win-select-trigger ${isOpen ? "open" : ""}`}
        onClick={() => {
          if (disabled) return;
          if (!isOpen) {
            openMenu();
          } else {
            closeMenu();
          }
        }}
        onKeyDown={handleTriggerKeyDown}
        onBlur={(e) => {
          const next = e.relatedTarget as Node | null;
          if (isOpen && (!next || (!containerRef.current?.contains(next) && !menuRef.current?.contains(next)))) {
            closeMenu();
          }
        }}
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

      {/* Dropdown Menu Popup in Portal (prevents overflow clipping by parent cards/modals) */}
      {typeof document !== "undefined" && menuDropdown && createPortal(menuDropdown, document.body)}
    </div>
  );
};
