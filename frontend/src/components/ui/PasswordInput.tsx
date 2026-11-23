import React, { useState, forwardRef } from "react";
import { Eye, EyeOff } from "lucide-react";

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
  ...props
}, ref) => {
  const [visible, setVisible] = useState(false);

  const rightPadding = extraAction ? "68px" : "36px";
  const leftPadding = leftIcon ? "36px" : undefined;

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
        {...props}
      />

      <div
        style={{
          position: "absolute",
          right: "6px",
          display: "flex",
          alignItems: "center",
          gap: "3px",
          zIndex: 2,
        }}
      >
        <button
          type="button"
          tabIndex={-1}
          disabled={disabled}
          className="win-password-toggle"
          onMouseDown={(e) => e.preventDefault()}
          onClick={() => setVisible(!visible)}
          title={visible ? "Скрыть пароль" : "Показать пароль"}
        >
          {visible ? <EyeOff size={14} /> : <Eye size={14} />}
        </button>

        {extraAction}
      </div>
    </div>
  );
});

PasswordInput.displayName = "PasswordInput";
