import React, { useState, useRef, useEffect, useLayoutEffect, useCallback } from "react";
import { createPortal } from "react-dom";

export interface WinActionMenuProps {
  isOpen: boolean;
  anchorRef: React.RefObject<HTMLElement | null>;
  onClose: () => void;
  children: React.ReactNode;
}

export const WinActionMenu: React.FC<WinActionMenuProps> = ({
  isOpen,
  anchorRef,
  onClose,
  children,
}) => {
  const [isRendered, setIsRendered] = useState(isOpen);
  const [isClosing, setIsClosing] = useState(false);
  const [placement, setPlacement] = useState<"bottom" | "top">("bottom");
  const [coords, setCoords] = useState<{ top: number; left: number }>({ top: -9999, left: -9999 });
  const [isPositioned, setIsPositioned] = useState(false);

  const menuRef = useRef<HTMLDivElement | null>(null);
  const closeTimerRef = useRef<number | null>(null);
  const onCloseRef = useRef(onClose);
  onCloseRef.current = onClose;

  const updatePosition = useCallback(() => {
    const anchorEl = anchorRef.current;
    const menuEl = menuRef.current;
    if (!anchorEl || !menuEl) return;

    const rect = anchorEl.getBoundingClientRect();
    const menuWidth = menuEl.offsetWidth || 246;
    const menuHeight = menuEl.offsetHeight || 280;
    const margin = 8;
    const gap = 6;
    const vw = window.innerWidth;
    const vh = window.innerHeight;

    const spaceBelow = vh - rect.bottom - margin;
    const spaceAbove = rect.top - margin;
    const chosen: "bottom" | "top" =
      spaceBelow < menuHeight + gap && spaceAbove > spaceBelow ? "top" : "bottom";

    const rawTop = chosen === "bottom" ? rect.bottom + gap : rect.top - menuHeight - gap;
    const top = Math.max(margin, Math.min(rawTop, vh - menuHeight - margin));
    const rawLeft = rect.right - menuWidth;
    const left = Math.max(margin, Math.min(rawLeft, vw - menuWidth - margin));

    setPlacement(chosen);
    setCoords({ top, left });
    setIsPositioned(true);
  }, [anchorRef]);

  useEffect(() => {
    if (isOpen) {
      if (closeTimerRef.current !== null) {
        window.clearTimeout(closeTimerRef.current);
        closeTimerRef.current = null;
      }
      setIsClosing(false);
      setIsPositioned(false);
      setIsRendered(true);
    } else if (isRendered && !isClosing) {
      setIsClosing(true);
      closeTimerRef.current = window.setTimeout(() => {
        closeTimerRef.current = null;
        setIsRendered(false);
        setIsClosing(false);
        setIsPositioned(false);
      }, 115);
    }
  }, [isOpen, isRendered, isClosing]);

  useLayoutEffect(() => {
    if (isRendered && !isClosing) {
      updatePosition();
    }
  }, [isRendered, isClosing, children, updatePosition]);

  useEffect(() => {
    if (!isRendered || isClosing) return;

    const handleMouseDown = (e: MouseEvent) => {
      const target = e.target as Node;
      if (
        menuRef.current &&
        !menuRef.current.contains(target) &&
        anchorRef.current &&
        !anchorRef.current.contains(target)
      ) {
        onCloseRef.current();
      }
    };

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        e.stopPropagation();
        onCloseRef.current();
      }
    };

    const handleScroll = (e: Event) => {
      const target = e.target;
      if (menuRef.current && target instanceof Node && menuRef.current.contains(target)) {
        return;
      }
      if (
        target instanceof Node &&
        target !== document &&
        target !== document.documentElement &&
        anchorRef.current &&
        !target.contains(anchorRef.current)
      ) {
        return;
      }
      onCloseRef.current();
    };

    const handleResize = () => {
      onCloseRef.current();
    };

    document.addEventListener("mousedown", handleMouseDown);
    document.addEventListener("keydown", handleKeyDown);
    window.addEventListener("scroll", handleScroll, true);
    window.addEventListener("resize", handleResize);
    return () => {
      document.removeEventListener("mousedown", handleMouseDown);
      document.removeEventListener("keydown", handleKeyDown);
      window.removeEventListener("scroll", handleScroll, true);
      window.removeEventListener("resize", handleResize);
    };
  }, [isRendered, isClosing, anchorRef]);

  useEffect(() => {
    return () => {
      if (closeTimerRef.current !== null) {
        window.clearTimeout(closeTimerRef.current);
      }
    };
  }, []);

  if (!isRendered) return null;

  return createPortal(
    <div
      ref={menuRef}
      className={`server-more-menu placement-${placement} ${isClosing ? "exiting" : ""}`}
      style={{
        top: `${coords.top}px`,
        left: `${coords.left}px`,
        visibility: isPositioned ? "visible" : "hidden",
      }}
      onClick={(e) => e.stopPropagation()}
    >
      {children}
    </div>,
    document.body
  );
};
