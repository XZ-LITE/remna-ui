import React, { useState, useRef, useEffect, useLayoutEffect, useCallback } from "react";
import { createPortal } from "react-dom";

export interface WinTooltipProps {
  content: React.ReactNode;
  shortcut?: string;
  placement?: "top" | "bottom" | "left" | "right";
  delay?: number;
  disabled?: boolean;
  children: React.ReactElement;
}

// Global timestamp to enable warm-up chaining between adjacent tooltips
let lastTooltipCloseTime = 0;

export const WinTooltip: React.FC<WinTooltipProps> = ({
  content,
  shortcut,
  placement = "top",
  delay = 280,
  disabled = false,
  children,
}) => {
  const [isVisible, setIsVisible] = useState(false);
  const [isPositioned, setIsPositioned] = useState(false);
  const [coords, setCoords] = useState<{ top: number; left: number }>({ top: -9999, left: -9999 });
  const [actualPlacement, setActualPlacement] = useState<"top" | "bottom" | "left" | "right">(placement);

  const triggerRef = useRef<HTMLElement | null>(null);
  const tooltipRef = useRef<HTMLDivElement | null>(null);
  const timerRef = useRef<number | null>(null);
  const isVisibleRef = useRef(false);
  isVisibleRef.current = isVisible;

  const calculatePosition = useCallback(() => {
    const triggerEl = triggerRef.current;
    const tipEl = tooltipRef.current;
    if (!triggerEl || !tipEl) return;

    const rect = triggerEl.getBoundingClientRect();
    const tipWidth = tipEl.offsetWidth;
    const tipHeight = tipEl.offsetHeight;
    if (tipWidth === 0 || tipHeight === 0) return;

    const gap = 8;
    const margin = 8;
    const vw = window.innerWidth;
    const vh = window.innerHeight;

    let chosen = placement;

    if (chosen === "top" && rect.top - tipHeight - gap < margin) {
      chosen = "bottom";
    } else if (chosen === "bottom" && rect.bottom + tipHeight + gap > vh - margin) {
      chosen = "top";
    } else if (chosen === "left" && rect.left - tipWidth - gap < margin) {
      chosen = "right";
    } else if (chosen === "right" && rect.right + tipWidth + gap > vw - margin) {
      chosen = "left";
    }

    let top = 0;
    let left = 0;

    switch (chosen) {
      case "top":
        top = rect.top - tipHeight - gap;
        left = rect.left + (rect.width - tipWidth) / 2;
        break;
      case "bottom":
        top = rect.bottom + gap;
        left = rect.left + (rect.width - tipWidth) / 2;
        break;
      case "left":
        top = rect.top + (rect.height - tipHeight) / 2;
        left = rect.left - tipWidth - gap;
        break;
      case "right":
        top = rect.top + (rect.height - tipHeight) / 2;
        left = rect.right + gap;
        break;
    }

    left = Math.max(margin, Math.min(left, vw - tipWidth - margin));
    top = Math.max(margin, Math.min(top, vh - tipHeight - margin));

    setCoords({ top, left });
    setActualPlacement(chosen);
    setIsPositioned(true);
  }, [placement]);

  const hideTooltip = useCallback(() => {
    if (timerRef.current !== null) {
      window.clearTimeout(timerRef.current);
      timerRef.current = null;
    }
    if (isVisibleRef.current) {
      lastTooltipCloseTime = Date.now();
    }
    setIsVisible(false);
    setIsPositioned(false);
  }, []);

  const scheduleShow = useCallback(() => {
    if (disabled || !content) return;
    if (timerRef.current !== null) {
      window.clearTimeout(timerRef.current);
    }
    const isWarm = Date.now() - lastTooltipCloseTime < 450;
    const waitMs = isWarm ? 60 : delay;

    timerRef.current = window.setTimeout(() => {
      timerRef.current = null;
      setIsPositioned(false);
      setIsVisible(true);
    }, waitMs);
  }, [disabled, content, delay]);

  // Measure and position synchronously before browser paint
  useLayoutEffect(() => {
    if (isVisible) {
      calculatePosition();
    }
  }, [isVisible, content, shortcut, calculatePosition]);

  // Dismiss on scroll, resize, window blur, or Escape while visible
  useEffect(() => {
    if (!isVisible) return;

    const handleDismiss = () => hideTooltip();
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") hideTooltip();
    };

    window.addEventListener("scroll", handleDismiss, true);
    window.addEventListener("resize", handleDismiss);
    window.addEventListener("blur", handleDismiss);
    window.addEventListener("keydown", handleKeyDown);
    return () => {
      window.removeEventListener("scroll", handleDismiss, true);
      window.removeEventListener("resize", handleDismiss);
      window.removeEventListener("blur", handleDismiss);
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [isVisible, hideTooltip]);

  // Hide if disabled becomes true while visible (WebView2 skips mouseleave on disabled elements)
  const childDisabled = Boolean((children.props as any)?.disabled);
  useEffect(() => {
    if ((disabled || childDisabled) && isVisible) {
      hideTooltip();
    }
  }, [disabled, childDisabled, isVisible, hideTooltip]);

  useEffect(() => {
    return () => {
      if (timerRef.current !== null) {
        window.clearTimeout(timerRef.current);
      }
    };
  }, []);

  const childProps = children.props as any;
  const existingRef = childProps?.ref;

  const mergedRef = useCallback(
    (node: HTMLElement | null) => {
      triggerRef.current = node;
      if (typeof existingRef === "function") {
        existingRef(node);
      } else if (existingRef && typeof existingRef === "object") {
        existingRef.current = node;
      }
    },
    [existingRef]
  );

  const clonedChild = React.cloneElement(children as React.ReactElement<any>, {
    ref: mergedRef,
    onMouseEnter: (e: React.MouseEvent) => {
      scheduleShow();
      if (typeof childProps?.onMouseEnter === "function") childProps.onMouseEnter(e);
    },
    onMouseLeave: (e: React.MouseEvent) => {
      hideTooltip();
      if (typeof childProps?.onMouseLeave === "function") childProps.onMouseLeave(e);
    },
    onMouseDown: (e: React.MouseEvent) => {
      hideTooltip();
      if (typeof childProps?.onMouseDown === "function") childProps.onMouseDown(e);
    },
    onFocus: (e: React.FocusEvent) => {
      scheduleShow();
      if (typeof childProps?.onFocus === "function") childProps.onFocus(e);
    },
    onBlur: (e: React.FocusEvent) => {
      hideTooltip();
      if (typeof childProps?.onBlur === "function") childProps.onBlur(e);
    },
  });

  const tooltipNode =
    isVisible && content
      ? createPortal(
          <div
            ref={tooltipRef}
            className={`win-tooltip-popup placement-${actualPlacement} ${isPositioned ? "positioned" : ""}`}
            style={{
              top: `${coords.top}px`,
              left: `${coords.left}px`,
              visibility: isPositioned ? "visible" : "hidden",
            }}
            role="tooltip"
          >
            <span className="win-tooltip-text">{content}</span>
            {shortcut && <kbd className="win-kbd">{shortcut}</kbd>}
          </div>,
          document.body
        )
      : null;

  return (
    <>
      {clonedChild}
      {tooltipNode}
    </>
  );
};
