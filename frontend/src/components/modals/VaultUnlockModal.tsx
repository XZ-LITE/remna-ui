import React, { useState, useEffect, useRef } from "react";
import { Lock, ShieldCheck, KeyRound, AlertCircle, RefreshCw } from "lucide-react";
import { PasswordInput } from "../ui/PasswordInput";
import { WinModal } from "../ui/WinModal";

interface VaultUnlockModalProps {
  isOpen: boolean;
  onUnlock: (password: string) => Promise<boolean>;
  onToast: (type: "success" | "error" | "info", msg: string) => void;
}

export const VaultUnlockModal: React.FC<VaultUnlockModalProps> = ({
  isOpen,
  onUnlock,
  onToast,
}) => {
  const [password, setPassword] = useState<string>("");
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string>("");
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (isOpen) {
      setPassword("");
      setError("");
      setTimeout(() => {
        inputRef.current?.focus();
      }, 120);
    }
  }, [isOpen]);

  const handleSubmit = async (e?: React.FormEvent) => {
    if (e) e.preventDefault();
    if (!password) {
      setError("Введите мастер-пароль");
      return;
    }
    setLoading(true);
    setError("");
    try {
      const ok = await onUnlock(password);
      if (ok) {
        onToast("success", "Хранилище успешно разблокировано");
        setPassword("");
      } else {
        setError("Неверный мастер-пароль. Попробуйте снова.");
        inputRef.current?.focus();
      }
    } catch (err: any) {
      setError(err?.message || "Ошибка проверки мастер-пароля");
    } finally {
      setLoading(false);
    }
  };

  return (
    <WinModal
      isOpen={isOpen}
      onClose={() => {}}
      maxWidth="460px"
      zIndex={10000}
      preventBackdropClose={true}
      dialogStyle={{
        padding: "24px",
        textAlign: "center",
        borderRadius: "12px",
        background: "var(--win-modal-bg)",
        boxShadow: "0 20px 40px rgba(0, 0, 0, 0.45)",
        border: "1px solid var(--win-stroke-card)",
      }}
    >
      <div
        style={{
          width: "56px",
          height: "56px",
          borderRadius: "50%",
          background: "rgba(59, 130, 246, 0.12)",
          color: "var(--win-accent)",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          margin: "0 auto 16px",
          border: "1px solid rgba(59, 130, 246, 0.25)",
        }}
      >
        <Lock size={26} />
      </div>

      <h3 style={{ margin: "0 0 6px", fontSize: "18px", fontWeight: 600, color: "var(--win-text-primary)" }}>
        Хранилище заблокировано
      </h3>
      <p style={{ margin: "0 0 20px", fontSize: "12.5px", color: "var(--win-text-secondary)", lineHeight: "1.5" }}>
        База серверов и учетных данных зашифрована. Введите мастер-пароль для разблокировки сессии.
      </p>

      <form onSubmit={handleSubmit}>
        <div style={{ marginBottom: "14px" }}>
          <PasswordInput
            ref={inputRef}
            leftIcon={<KeyRound size={16} />}
            value={password}
            onChange={(e) => {
              setPassword(e.target.value);
              if (error) setError("");
            }}
            placeholder="Мастер-пароль хранилища"
            disabled={loading}
            autoFocus
            style={{ height: "38px", fontSize: "13px" }}
          />
        </div>

        {error && (
          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: "6px",
              color: "#ef4444",
              fontSize: "12px",
              marginBottom: "14px",
              justifyContent: "center",
            }}
          >
            <AlertCircle size={14} />
            <span>{error}</span>
          </div>
        )}

        <div style={{ display: "flex", flexDirection: "column", gap: "10px" }}>
          <button
            type="submit"
            className="win-button primary"
            style={{ width: "100%", height: "36px", fontSize: "13px", fontWeight: 600, justifyContent: "center" }}
            disabled={loading || !password}
          >
            {loading ? (
              <>
                <RefreshCw size={14} className="spin" />
                <span>Проверка Argon2id...</span>
              </>
            ) : (
              <span>Разблокировать хранилище</span>
            )}
          </button>
        </div>
      </form>

      <div
        style={{
          marginTop: "20px",
          paddingTop: "14px",
          borderTop: "1px solid var(--win-stroke-card)",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          gap: "6px",
          fontSize: "11px",
          color: "var(--win-text-tertiary)",
        }}
      >
        <ShieldCheck size={13} style={{ color: "var(--win-accent)" }} />
        <span>Защита AES-256-GCM + Argon2id (Zero-Knowledge)</span>
      </div>
    </WinModal>
  );
};
