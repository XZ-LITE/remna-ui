import React, { useState, useEffect, useRef, useCallback } from "react";
import { createPortal } from "react-dom";

export interface WinModalProps {
  isOpen: boolean;
  onClose: () => void;
  children: React.ReactNode;
  maxWidth?: string;
  width?: string;
  maxHeight?: string;
  preventBackdropClose?: boolean;
  zIndex?: number;
  className?: string;
  dialogClassName?: string;
  contentStyle?: React.CSSProperties;
  dialogStyle?: React.CSSProperties;
}

export const WinModal: React.FC<WinModalProps> = ({
  isOpen,
  onClose,
  children,
  maxWidth,
  width,
  maxHeight,
  preventBackdropClose = false,
  zIndex = 1000,
  className = "",
  dialogClassName = "",
  contentStyle,
  dialogStyle,
}) => {
  const [isRendered, setIsRendered] = useState(isOpen);
  const [isClosing, setIsClosing] = useState(false);
  const [isWobbling, setIsWobbling] = useState(false);
  const closeTimeoutRef = useRef<number | null>(null);
  const wobbleTimeoutRef = useRef<number | null>(null);

  const triggerWobble = useCallback(() => {
    setIsWobbling(false);
    if (wobbleTimeoutRef.current) {
      clearTimeout(wobbleTimeoutRef.current);
    }
    // Microtask restart so class re-triggers animation
    setTimeout(() => {
      setIsWobbling(true);
    }, 10);
    wobbleTimeoutRef.current = window.setTimeout(() => {
      setIsWobbling(false);
    }, 320);
  }, []);

  const handleClose = useCallback(() => {
    if (isClosing) return;
    setIsClosing(true);
    if (closeTimeoutRef.current) {
      clearTimeout(closeTimeoutRef.current);
    }
    closeTimeoutRef.current = window.setTimeout(() => {
      setIsRendered(false);
      setIsClosing(false);
      onClose();
    }, 140);
  }, [isClosing, onClose]);

  // Handle external isOpen prop transitions
  useEffect(() => {
    if (isOpen) {
      if (closeTimeoutRef.current) {
        clearTimeout(closeTimeoutRef.current);
      }
      setIsClosing(false);
      setIsRendered(true);
    } else if (isRendered && !isClosing) {
      handleClose();
    }
  }, [isOpen, isRendered, isClosing, handleClose]);

  // Escape key handler
  useEffect(() => {
    if (!isRendered) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        if (preventBackdropClose) {
          triggerWobble();
        } else {
          handleClose();
        }
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [isRendered, preventBackdropClose, triggerWobble, handleClose]);

  // Cleanup timers on unmount
  useEffect(() => {
    return () => {
      if (closeTimeoutRef.current) clearTimeout(closeTimeoutRef.current);
      if (wobbleTimeoutRef.current) clearTimeout(wobbleTimeoutRef.current);
    };
  }, []);

  if (!isRendered && typeof document === "undefined") {
    return null;
  }
  if (!isRendered) {
    return null;
  }

  const handleBackdropClick = (e: React.MouseEvent) => {
    if (e.target !== e.currentTarget) return;
    if (preventBackdropClose) {
      triggerWobble();
    } else {
      handleClose();
    }
  };

  const modalNode = (
    <div
      className={`win-modal-backdrop ${isClosing ? "exiting" : "entering"} ${className}`}
      style={{ zIndex, ...contentStyle }}
      onClick={handleBackdropClick}
    >
      <div
        className={`win-modal-dialog ${isClosing ? "exiting" : "entering"} ${isWobbling ? "wobble-attention" : ""} ${dialogClassName}`}
        style={{
          maxWidth: maxWidth || "860px",
          width: width || "100%",
          maxHeight: maxHeight || "85vh",
          ...dialogStyle,
        }}
        onClick={(e) => e.stopPropagation()}
      >
        {children}
      </div>
    </div>
  );

  return createPortal(modalNode, document.body);
};