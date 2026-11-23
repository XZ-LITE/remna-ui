import React, { useState, useRef, useEffect, useCallback, cloneElement } from "react";
import { createPortal } from "react-dom";

export interface WinTooltipProps {
  content: React.ReactNode;
  shortcut?: string;
  placement?: "top" | "bottom" | "left" | "right";
  delay?: number;
  disabled?: boolean;
  className?: string;
  children: React.ReactElement;
}

// Module-level warm tooltip tracker for instant transitions between neighboring controls
let lastTooltipDismissTime = 0;

export const WinTooltip: React.FC<WinTooltipProps> = ({
  content,
  shortcut,
  placement = "top",
  delay = 250,
  disabled = false,
  className = "",
  children,
}) => {
  const [isVisible, setIsVisible] = useState(false);
  const [coords, setCoords] = useState<{ top: number; left: number; actualPlacement: string }>({
    top: 0,
    left: 0,
    actualPlacement: placement,
  });

  const triggerRef = useRef<HTMLElement | null>(null);
  const tooltipRef = useRef<HTMLDivElement | null>(null);
  const timerRef = useRef<number | null>(null);

  const clearTimer = () => {
    if (timerRef.current) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
  };

  const calculatePosition = useCallback(() => {
    if (!triggerRef.current) return;
    const rect = triggerRef.current.getBoundingClientRect();
    const gap = 6;

    // Approximate size for first layout calculation
    const tipWidth = tooltipRef.current?.offsetWidth || 120;
    const tipHeight = tooltipRef.current?.offsetHeight || 28;

    let targetTop = 0;
    let targetLeft = 0;
    let currentPlacement = placement;

    // Auto-flip vertical placement if close to window edge
    if (placement === "top" && rect.top - tipHeight - gap < 4) {
      currentPlacement = "bottom";
    } else if (placement === "bottom" && rect.bottom + tipHeight + gap > window.innerHeight - 4) {
      currentPlacement = "top";
    }

    switch (currentPlacement) {
      case "bottom":
        targetTop = rect.bottom + gap;
        targetLeft = rect.left + rect.width / 2 - tipWidth / 2;
        break;
      case "left":
        targetTop = rect.top + rect.height / 2 - tipHeight / 2;
        targetLeft = rect.left - tipWidth - gap;
        break;
      case "right":
        targetTop = rect.top + rect.height / 2 - tipHeight / 2;
        targetLeft = rect.right + gap;
        break;
      case "top":
      default:
        targetTop = rect.top - tipHeight - gap;
        targetLeft = rect.left + rect.width / 2 - tipWidth / 2;
        break;
    }

    // Viewport bounds clamping
    targetLeft = Math.max(8, Math.min(window.innerWidth - tipWidth - 8, targetLeft));
    targetTop = Math.max(8, Math.min(window.innerHeight - tipHeight - 8, targetTop));

    setCoords({
      top: targetTop,
      left: targetLeft,
      actualPlacement: currentPlacement,
    });
  }, [placement]);

  const showTooltip = () => {
    if (disabled || !content) return;
    clearTimer();

    // Check warm tooltip cache (within 350ms of last dismissal)
    const isWarm = Date.now() - lastTooltipDismissTime < 350;
    const effectiveDelay = isWarm ? 0 : delay;

    if (effectiveDelay === 0) {
      calculatePosition();
      setIsVisible(true);
    } else {
      timerRef.current = window.setTimeout(() => {
        calculatePosition();
        setIsVisible(true);
      }, effectiveDelay);
    }
  };

  const hideTooltip = () => {
    clearTimer();
    if (isVisible) {
      lastTooltipDismissTime = Date.now();
      setIsVisible(false);
    }
  };

  // Recalculate accurately when tooltip element mounts
  useEffect(() => {
    if (isVisible && tooltipRef.current) {
      calculatePosition();
    }
  }, [isVisible, calculatePosition]);

  // Dismiss on any scroll or window resize
  useEffect(() => {
    if (!isVisible) return;

    const handleDismiss = () => {
      hideTooltip();
    };

    window.addEventListener("scroll", handleDismiss, true);
    window.addEventListener("resize", handleDismiss);

    return () => {
      window.removeEventListener("scroll", handleDismiss, true);
      window.removeEventListener("resize", handleDismiss);
    };
  }, [isVisible]);

  // Clean timer on unmount
  useEffect(() => {
    return () => clearTimer();
  }, []);

  const childProps = (children as any).props || {};

  const clonedChild = cloneElement(children as React.ReactElement<any>, {
    ref: (node: HTMLElement | null) => {
      triggerRef.current = node;
      // Preserve original ref if child has one
      const originalRef = (children as any).ref;
      if (typeof originalRef === "function") {
        originalRef(node);
      } else if (originalRef && typeof originalRef === "object") {
        originalRef.current = node;
      }
    },
    onMouseEnter: (e: React.MouseEvent) => {
      childProps.onMouseEnter?.(e);
      showTooltip();
    },
    onMouseLeave: (e: React.MouseEvent) => {
      childProps.onMouseLeave?.(e);
      hideTooltip();
    },
    onClick: (e: React.MouseEvent) => {
      childProps.onClick?.(e);
      hideTooltip();
    },
    onFocus: (e: React.FocusEvent) => {
      childProps.onFocus?.(e);
      showTooltip();
    },
    onBlur: (e: React.FocusEvent) => {
      childProps.onBlur?.(e);
      hideTooltip();
    },
  } as any);

  return (
    <>
      {clonedChild}
      {isVisible &&
        typeof document !== "undefined" &&
        createPortal(
          <div
            ref={tooltipRef}
            className={`win-tooltip-popup placement-${coords.actualPlacement} ${className}`}
            role="tooltip"
            style={{
              position: "fixed",
              top: `${coords.top}px`,
              left: `${coords.left}px`,
              zIndex: 16000,
            }}
          >
            <span className="win-tooltip-text">{content}</span>
            {shortcut && <kbd className="win-kbd">{shortcut}</kbd>}
          </div>,
          document.body
        )}
    </>
  );
};