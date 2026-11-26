import React, { useState, useEffect } from "react";
import { Lock, Unlock, ShieldCheck, KeyRound, AlertCircle, X, Check, RefreshCw } from "lucide-react";
import * as types from "../../../bindings/remna-ui/internal/types/models";
import { PasswordInput } from "../ui/PasswordInput";
import { WinSelect } from "../ui/WinSelect";
import { WinModal } from "../ui/WinModal";
import { WinTooltip } from "../ui/WinTooltip";

interface VaultSettingsModalProps {
  isOpen: boolean;
  vaultStatus: types.VaultStatus | null;
  onClose: () => void;
  onSetMasterPassword: (oldPass: string, newPass: string) => Promise<boolean>;
  onDisableMasterPassword: (currentPass: string) => Promise<boolean>;
  onSetAutoLockMinutes: (mins: number) => Promise<boolean>;
  onToast: (type: "success" | "error" | "info" | "warning", msg: string) => void;
  onRefreshStatus: () => Promise<void>;
}

export const VaultSettingsModal: React.FC<VaultSettingsModalProps> = ({
  isOpen,
  vaultStatus,
  onClose,
  onSetMasterPassword,
  onDisableMasterPassword,
  onSetAutoLockMinutes,
  onToast,
  onRefreshStatus,
}) => {
  const [activeTab, setActiveTab] = useState<"status" | "change" | "disable" | "enable">("status");
  const [currentPassword, setCurrentPassword] = useState<string>("");
  const [newPassword, setNewPassword] = useState<string>("");
  const [confirmPassword, setConfirmPassword] = useState<string>("");
  const [autoLockVal, setAutoLockVal] = useState<number>(vaultStatus?.auto_lock_minutes || 15);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string>("");

  useEffect(() => {
    if (isOpen) {
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
      setError("");
      setAutoLockVal(vaultStatus?.auto_lock_minutes || 15);
      setActiveTab("status");
    }
  }, [isOpen, vaultStatus]);

  const isMasterEnabled = Boolean(vaultStatus?.enabled);

  const handleEnableOrChange = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newPassword || newPassword.length < 8) {
      setError("Мастер-пароль должен содержать не менее 8 символов");
      return;
    }
    if (newPassword !== confirmPassword) {
      setError("Пароли не совпадают");
      return;
    }
    if (isMasterEnabled && !currentPassword) {
      setError("Укажите текущий мастер-пароль");
      return;
    }

    setLoading(true);
    setError("");
    try {
      const ok = await onSetMasterPassword(isMasterEnabled ? currentPassword : "", newPassword);
      if (ok) {
        onToast("success", isMasterEnabled ? "Мастер-пароль успешно изменен!" : "Защита мастер-паролем активирована!");
        await onRefreshStatus();
        onClose();
      } else {
        setError("Не удалось установить мастер-пароль. Проверьте текущий пароль.");
      }
    } catch (err: any) {
      setError(err?.message || "Ошибка при установке мастер-пароля");
    } finally {
      setLoading(false);
    }
  };

  const handleDisable = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!currentPassword) {
      setError("Укажите текущий мастер-пароль для отключения");
      return;
    }

    setLoading(true);
    setError("");
    try {
      const ok = await onDisableMasterPassword(currentPassword);
      if (ok) {
        onToast("info", "Мастер-пароль отключен. Хранилище переведено в режим Windows DPAPI.");
        await onRefreshStatus();
        onClose();
      } else {
        setError("Неверный мастер-пароль.");
      }
    } catch (err: any) {
      setError(err?.message || "Ошибка при отключении мастер-пароля");
    } finally {
      setLoading(false);
    }
  };

  const handleSaveAutoLock = async (val: number) => {
    setAutoLockVal(val);
    try {
      await onSetAutoLockMinutes(val);
      onToast("info", `Интервал автоблокировки обновлен: ${val === 0 ? "Никогда" : val + " мин."}`);
      await onRefreshStatus();
    } catch {
      onToast("error", "Не удалось сохранить интервал автоблокировки");
    }
  };

  return (
    <WinModal
      isOpen={isOpen}
      onClose={onClose}
      maxWidth="520px"
      preventBackdropClose={loading}
      dialogStyle={{
        padding: "24px",
        borderRadius: "12px",
        background: "var(--win-content-bg)",
        boxShadow: "0 20px 40px rgba(0, 0, 0, 0.45)",
        border: "1px solid var(--win-stroke-card)",
      }}
    >
      {({ close }) => (
        <>
        {/* Header */}
        <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: "18px" }}>
          <div style={{ display: "flex", alignItems: "center", gap: "10px" }}>
            <div
              style={{
                width: "36px",
                height: "36px",
                borderRadius: "8px",
                background: "rgba(59, 130, 246, 0.12)",
                color: "var(--win-accent)",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
              }}
            >
              <ShieldCheck size={20} />
            </div>
            <div>
              <div style={{ fontSize: "16px", fontWeight: 600, color: "var(--win-text-primary)" }}>
                Безопасность локальных секретов
              </div>
              <div style={{ fontSize: "12px", color: "var(--win-text-secondary)" }}>
                Шифрование базы серверов, ключей SSH и токенов
              </div>
            </div>
          </div>
          <WinTooltip content="Закрыть" shortcut="Esc">
            <button
              type="button"
              className="win-button"
              style={{ padding: "0 8px", height: "26px" }}
              onClick={() => !loading && close()}
            >
              <X size={14} />
            </button>
          </WinTooltip>
        </div>

        {/* Tab Selector if Master is Enabled */}
        {isMasterEnabled && activeTab === "status" && (
          <div style={{ display: "flex", flexDirection: "column", gap: "16px" }}>
            <div
              style={{
                padding: "14px 16px",
                borderRadius: "8px",
                background: "rgba(16, 185, 129, 0.08)",
                border: "1px solid rgba(16, 185, 129, 0.25)",
                display: "flex",
                alignItems: "flex-start",
                gap: "10px",
              }}
            >
              <Lock size={18} style={{ color: "#10b981", flexShrink: 0, marginTop: "2px" }} />
              <div style={{ fontSize: "12.5px", color: "var(--win-text-secondary)", lineHeight: "1.5" }}>
                <strong style={{ color: "var(--win-text-primary)" }}>Защита активна: AES-256-GCM + Argon2id</strong>
                <div style={{ marginTop: "2px" }}>
                  Все сохраненные пароли VPS, ключи SSH и API-токены зашифрованы вашим мастер-паролем. Доступ к ним невозможен без ввода пароля при запуске приложения.
                </div>
              </div>
            </div>

            {/* Auto-Lock Settings */}
            <div>
              <div style={{ fontSize: "12.5px", fontWeight: 600, color: "var(--win-text-primary)", marginBottom: "6px" }}>
                Автоблокировка при неактивности:
              </div>
              <WinSelect
                style={{ width: "100%" }}
                value={String(autoLockVal)}
                onChange={(val) => handleSaveAutoLock(parseInt(val, 10) || 0)}
                options={[
                  { value: "5", label: "Через 5 минут" },
                  { value: "15", label: "Через 15 минут (Рекомендуется)" },
                  { value: "30", label: "Через 30 минут" },
                  { value: "60", label: "Через 1 час" },
                  { value: "0", label: "Никогда (Только при закрытии программы)" },
                ]}
              />
            </div>

            {/* Actions */}
            <div style={{ display: "flex", gap: "10px", marginTop: "8px" }}>
              <button
                type="button"
                className="win-button"
                style={{ flex: 1, height: "34px", justifyContent: "center" }}
                onClick={() => {
                  setError("");
                  setActiveTab("change");
                }}
              >
                <KeyRound size={14} />
                <span>Сменить пароль</span>
              </button>
              <button
                type="button"
                className="win-button"
                style={{ flex: 1, height: "34px", justifyContent: "center", color: "#ef4444" }}
                onClick={() => {
                  setError("");
                  setActiveTab("disable");
                }}
              >
                <Unlock size={14} />
                <span>Отключить</span>
              </button>
            </div>
          </div>
        )}

        {/* Form: Enable or Change Password */}
        {(activeTab === "enable" || activeTab === "change" || (!isMasterEnabled && activeTab === "status")) && (
          <form onSubmit={handleEnableOrChange} style={{ display: "flex", flexDirection: "column", gap: "12px" }}>
            {!isMasterEnabled && (
              <div
                style={{
                  padding: "12px 14px",
                  borderRadius: "8px",
                  background: "rgba(59, 130, 246, 0.08)",
                  border: "1px solid rgba(59, 130, 246, 0.22)",
                  fontSize: "12px",
                  color: "var(--win-text-secondary)",
                  lineHeight: "1.45",
                }}
              >
                Сейчас используется шифрование <strong>Windows DPAPI</strong>. Установка мастер-пароля переводит базу в режим <strong>Zero-Knowledge</strong>: даже при доступе к вашему компьютеру никто не сможет прочитать пароли без знания мастер-пароля.
              </div>
            )}

            {isMasterEnabled && activeTab === "change" && (
              <div>
                <label style={{ fontSize: "12px", color: "var(--win-text-secondary)", display: "block", marginBottom: "4px" }}>
                  Текущий мастер-пароль:
                </label>
                <PasswordInput
                  value={currentPassword}
                  onChange={(e) => setCurrentPassword(e.target.value)}
                  placeholder="Введите текущий пароль"
                  disabled={loading}
                />
              </div>
            )}

            <div>
              <label style={{ fontSize: "12px", color: "var(--win-text-secondary)", display: "block", marginBottom: "4px" }}>
                {isMasterEnabled ? "Новый мастер-пароль:" : "Задайте мастер-пароль (минимум 8 символов):"}
              </label>
              <PasswordInput
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                placeholder="Надежный мастер-пароль"
                disabled={loading}
              />
            </div>

            <div>
              <label style={{ fontSize: "12px", color: "var(--win-text-secondary)", display: "block", marginBottom: "4px" }}>
                Подтверждение пароля:
              </label>
              <PasswordInput
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                placeholder="Повторите мастер-пароль"
                disabled={loading}
              />
            </div>

            {error && (
              <div style={{ display: "flex", alignItems: "center", gap: "6px", color: "#ef4444", fontSize: "12px" }}>
                <AlertCircle size={14} />
                <span>{error}</span>
              </div>
            )}

            <div style={{ display: "flex", gap: "8px", marginTop: "10px" }}>
              {isMasterEnabled && (
                <button
                  type="button"
                  className="win-button"
                  style={{ flex: 1, height: "34px", justifyContent: "center" }}
                  onClick={() => setActiveTab("status")}
                  disabled={loading}
                >
                  Назад
                </button>
              )}
              <button
                type="submit"
                className="win-button primary"
                style={{ flex: 2, height: "34px", justifyContent: "center" }}
                disabled={loading || !newPassword || !confirmPassword}
              >
                {loading ? (
                  <>
                    <RefreshCw size={14} className="spin" />
                    <span>Перешифрование базы...</span>
                  </>
                ) : (
                  <span>{isMasterEnabled ? "Обновить пароль" : "Активировать защиту"}</span>
                )}
              </button>
            </div>
          </form>
        )}

        {/* Form: Disable Master Password */}
        {activeTab === "disable" && (
          <form onSubmit={handleDisable} style={{ display: "flex", flexDirection: "column", gap: "12px" }}>
            <div
              style={{
                padding: "12px 14px",
                borderRadius: "8px",
                background: "rgba(239, 68, 68, 0.08)",
                border: "1px solid rgba(239, 68, 68, 0.25)",
                fontSize: "12px",
                color: "var(--win-text-secondary)",
                lineHeight: "1.45",
              }}
            >
              Вы собираетесь отключить мастер-пароль. База серверов будет переведена в режим прозрачного шифрования <strong>Windows DPAPI</strong> (без запроса паролей при старте).
            </div>

            <div>
              <label style={{ fontSize: "12px", color: "var(--win-text-secondary)", display: "block", marginBottom: "4px" }}>
                Введите текущий мастер-пароль для подтверждения:
              </label>
              <PasswordInput
                value={currentPassword}
                onChange={(e) => setCurrentPassword(e.target.value)}
                placeholder="Текущий мастер-пароль"
                disabled={loading}
                autoFocus
              />
            </div>

            {error && (
              <div style={{ display: "flex", alignItems: "center", gap: "6px", color: "#ef4444", fontSize: "12px" }}>
                <AlertCircle size={14} />
                <span>{error}</span>
              </div>
            )}

            <div style={{ display: "flex", gap: "8px", marginTop: "10px" }}>
              <button
                type="button"
                className="win-button"
                style={{ flex: 1, height: "34px", justifyContent: "center" }}
                onClick={() => setActiveTab("status")}
                disabled={loading}
              >
                Отмена
              </button>
              <button
                type="submit"
                className="win-button"
                style={{ flex: 2, height: "34px", justifyContent: "center", background: "#ef4444", color: "#fff" }}
                disabled={loading || !currentPassword}
              >
                {loading ? (
                  <>
                    <RefreshCw size={14} className="spin" />
                    <span>Расшифровка в DPAPI...</span>
                  </>
                ) : (
                  <span>Отключить мастер-пароль</span>
                )}
              </button>
            </div>
          </form>
        )}
        </>
      )}
    </WinModal>
  );
};
