import React, {
  createContext,
  useContext,
  useEffect,
  useState,
  useRef,
  useCallback,
} from "react";
import { createPortal } from "react-dom";

interface WinModalContextValue {
  close: () => void;
}

const WinModalContext = createContext<WinModalContextValue | null>(null);

/**
 * Returns the nearest WinModal's animated close function.
 * Calling `close()` plays the exit animation before invoking `onClose`.
 */
export function useWinModal(): WinModalContextValue {
  const ctx = useContext(WinModalContext);
  return ctx ?? { close: () => {} };
}

export interface WinModalProps {
  isOpen: boolean;
  onClose: () => void;
  children: React.ReactNode | ((helpers: { close: () => void }) => React.ReactNode);
  maxWidth?: string;
  width?: string;
  maxHeight?: string;
  zIndex?: number;
  dialogClassName?: string;
  dialogStyle?: React.CSSProperties;
  preventBackdropClose?: boolean;
  closeOnEscape?: boolean;
}

const EXIT_DURATION_MS = 140;
const ENTER_DURATION_MS = 250;

export const WinModal: React.FC<WinModalProps> = ({
  isOpen,
  onClose,
  children,
  maxWidth,
  width,
  maxHeight,
  zIndex = 1000,
  dialogClassName = "",
  dialogStyle,
  preventBackdropClose = false,
  closeOnEscape = true,
}) => {
  const [isRendered, setIsRendered] = useState(isOpen);
  const [isEntering, setIsEntering] = useState(isOpen);
  const [isClosing, setIsClosing] = useState(false);
  const [isWobbling, setIsWobbling] = useState(false);

  const closeTimerRef = useRef<number | null>(null);
  const enterTimerRef = useRef<number | null>(null);
  const wobbleTimerRef = useRef<number | null>(null);
  const onCloseRef = useRef(onClose);
  onCloseRef.current = onClose;

  const clearTimers = useCallback(() => {
    if (closeTimerRef.current !== null) {
      window.clearTimeout(closeTimerRef.current);
      closeTimerRef.current = null;
    }
    if (enterTimerRef.current !== null) {
      window.clearTimeout(enterTimerRef.current);
      enterTimerRef.current = null;
    }
    if (wobbleTimerRef.current !== null) {
      window.clearTimeout(wobbleTimerRef.current);
      wobbleTimerRef.current = null;
    }
  }, []);

  const triggerWobble = useCallback(() => {
    setIsEntering(false);
    setIsWobbling(false);
    if (wobbleTimerRef.current !== null) {
      window.clearTimeout(wobbleTimerRef.current);
    }
    requestAnimationFrame(() => {
      setIsWobbling(true);
      wobbleTimerRef.current = window.setTimeout(() => {
        setIsWobbling(false);
        wobbleTimerRef.current = null;
      }, 300);
    });
  }, []);

  const handleClose = useCallback(() => {
    if (isClosing) return;
    if (enterTimerRef.current !== null) {
      window.clearTimeout(enterTimerRef.current);
      enterTimerRef.current = null;
    }
    setIsEntering(false);
    setIsWobbling(false);
    setIsClosing(true);
    closeTimerRef.current = window.setTimeout(() => {
      closeTimerRef.current = null;
      setIsRendered(false);
      setIsClosing(false);
      onCloseRef.current();
    }, EXIT_DURATION_MS);
  }, [isClosing]);

  useEffect(() => {
    if (isOpen) {
      clearTimers();
      setIsRendered(true);
      setIsClosing(false);
      setIsWobbling(false);
      setIsEntering(true);
      enterTimerRef.current = window.setTimeout(() => {
        setIsEntering(false);
        enterTimerRef.current = null;
      }, ENTER_DURATION_MS);
    } else if (isRendered && !isClosing) {
      handleClose();
    }
  }, [isOpen, isRendered, isClosing, handleClose, clearTimers]);

  useEffect(() => {
    if (!isRendered || isClosing) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        if (preventBackdropClose || !closeOnEscape) {
          e.preventDefault();
          triggerWobble();
        } else {
          e.preventDefault();
          handleClose();
        }
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isRendered, isClosing, preventBackdropClose, closeOnEscape, handleClose, triggerWobble]);

  useEffect(() => {
    return () => {
      clearTimers();
    };
  }, [clearTimers]);

  if (!isRendered) return null;

  const handleBackdropClick = (e: React.MouseEvent<HTMLDivElement>) => {
    if (e.target !== e.currentTarget) return;
    if (preventBackdropClose) {
      triggerWobble();
    } else {
      handleClose();
    }
  };

  const mergedDialogStyle: React.CSSProperties = {
    ...(maxWidth ? { maxWidth } : {}),
    ...(width ? { width } : {}),
    ...(maxHeight ? { maxHeight } : {}),
    ...dialogStyle,
  };

  const dialogStateClasses = [
    "win-modal-dialog",
    dialogClassName,
    isEntering && !isClosing && !isWobbling ? "entering" : "",
    isClosing ? "exiting" : "",
    isWobbling ? "wobble-attention" : "",
  ]
    .filter(Boolean)
    .join(" ");

  const content =
    typeof children === "function" ? children({ close: handleClose }) : children;

  const modalNode = (
    <WinModalContext.Provider value={{ close: handleClose }}>
      <div
        className={`win-modal-backdrop ${isEntering && !isClosing ? "entering" : ""} ${isClosing ? "exiting" : ""}`}
        style={{ zIndex }}
        onMouseDown={(e) => {
          if (e.target === e.currentTarget) {
            (e.currentTarget as any)._mouseDownOnBackdrop = true;
          } else {
            (e.currentTarget as any)._mouseDownOnBackdrop = false;
          }
        }}
        onClick={(e) => {
          if ((e.currentTarget as any)._mouseDownOnBackdrop) {
            handleBackdropClick(e);
          }
          (e.currentTarget as any)._mouseDownOnBackdrop = false;
        }}
      >
        <div
          className={dialogStateClasses}
          style={mergedDialogStyle}
          onClick={(e) => e.stopPropagation()}
        >
          {content}
        </div>
      </div>
    </WinModalContext.Provider>
  );

  return createPortal(modalNode, document.body);
};
