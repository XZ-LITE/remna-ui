import React, { useState, forwardRef } from "react";
import { Eye, EyeOff } from "lucide-react";
import { WinTooltip } from "./WinTooltip";

export interface PasswordInputProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, "type"> {
  leftIcon?: React.ReactNode;
  extraAction?: React.ReactNode;
}

export const PasswordInput = forwardRef<HTMLInputElement, PasswordInputProps>(({
  className = "",
  style,
  leftIcon,
  extraAction,
  disabled,
  onKeyDown,
  onKeyUp,
  onBlur,
  ...props
}, ref) => {
  const [visible, setVisible] = useState(false);
  const [capsLock, setCapsLock] = useState(false);

  const rightPadding = extraAction
    ? capsLock && !disabled ? "108px" : "68px"
    : capsLock && !disabled ? "76px" : "36px";
  const leftPadding = leftIcon ? "36px" : undefined;

  const updateCapsLock = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (typeof e.getModifierState === "function") {
      setCapsLock(e.getModifierState("CapsLock"));
    }
  };

  return (
    <div
      className="win-password-wrap"
      style={{
        position: "relative",
        display: "flex",
        alignItems: "center",
        width: style?.width || "100%",
      }}
    >
      {leftIcon && (
        <div
          style={{
            position: "absolute",
            left: "10px",
            color: "var(--win-text-tertiary)",
            display: "flex",
            alignItems: "center",
            pointerEvents: "none",
            zIndex: 1,
          }}
        >
          {leftIcon}
        </div>
      )}

      <input
        ref={ref}
        type={visible ? "text" : "password"}
        disabled={disabled}
        className={`win-textbox ${className}`}
        style={{
          width: "100%",
          paddingRight: rightPadding,
          paddingLeft: leftPadding,
          ...style,
        }}
        onKeyDown={(e) => {
          updateCapsLock(e);
          onKeyDown?.(e);
        }}
        onKeyUp={(e) => {
          updateCapsLock(e);
          onKeyUp?.(e);
        }}
        onBlur={(e) => {
          setCapsLock(false);
          onBlur?.(e);
        }}
        {...props}
      />

      <div
        style={{
          position: "absolute",
          right: "6px",
          display: "flex",
          alignItems: "center",
          gap: "4px",
          zIndex: 2,
        }}
      >
        {capsLock && !disabled && (
          <span
            style={{
              fontSize: "9.5px",
              fontWeight: 700,
              letterSpacing: "0.04em",
              padding: "1px 5px",
              borderRadius: "4px",
              background: "rgba(245, 158, 11, 0.16)",
              border: "1px solid rgba(245, 158, 11, 0.35)",
              color: "#f59e0b",
              userSelect: "none",
              pointerEvents: "none",
            }}
          >
            CAPS
          </span>
        )}

        <WinTooltip content={visible ? "Скрыть пароль" : "Показать пароль"}>
          <button
            type="button"
            tabIndex={-1}
            disabled={disabled}
            className="win-password-toggle"
            onMouseDown={(e) => e.preventDefault()}
            onClick={() => setVisible(!visible)}
          >
            {visible ? <EyeOff size={14} /> : <Eye size={14} />}
          </button>
        </WinTooltip>

        {extraAction}
      </div>
    </div>
  );
});

PasswordInput.displayName = "PasswordInput";
