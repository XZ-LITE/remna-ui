import React, { useState } from "react";
import { Radio, Terminal as TerminalIcon, Check, Copy, X, Zap, RefreshCw, ExternalLink } from "lucide-react";
import * as types from "../../../bindings/remna-ui/internal/types/models";
import { WinModal } from "../ui/WinModal";

interface NodeSecretModalProps {
  server: types.SavedServer;
  parentMaster?: types.SavedServer;
  onOpenPanel?: () => void;
  secretKey: string;
  loading: boolean;
  onChangeSecretKey: (val: string) => void;
  onClose: () => void;
  onSubmit: () => void;
  onCopy: (text: string, label: string) => void;
  onToast: (type: "success" | "error" | "info", msg: string) => void;
}

export const NodeSecretModal: React.FC<NodeSecretModalProps> = ({
  server,
  parentMaster,
  onOpenPanel,
  secretKey,
  loading,
  onChangeSecretKey,
  onClose,
  onSubmit,
  onCopy,
  onToast,
}) => {
  const [guideMode, setGuideMode] = useState<"api" | "manual">("api");

  const handlePaste = async () => {
    try {
      const text = await navigator.clipboard.readText();
      if (text && text.trim()) {
        onChangeSecretKey(text.trim());
        onToast("info", "Текст вставлен из буфера обмена!");
      } else {
        onToast("error", "Буфер обмена пуст");
      }
    } catch {
      onToast("error", "Разрешите доступ к буферу обмена");
    }
  };

  return (
    <WinModal
      isOpen={true}
      onClose={onClose}
      maxWidth="620px"
      dialogClassName="node-modal-dialog"
      preventBackdropClose={loading}
    >
      {({ close }) => (
        <>
      {/* Header */}
        <div className="node-modal-header">
          <div className="node-modal-title-wrap">
            <div className="node-modal-icon" style={{ color: "var(--win-node-accent)" }}>
              <Radio size={20} />
            </div>
            <div>
              <div className="node-modal-title">Подключение Xray Reality Ноды</div>
              <div className="node-modal-sub">
                Сервер: <span style={{ color: "var(--win-text-primary)", fontWeight: 600 }}>{server.label || server.host}</span>
                {parentMaster && (
                  <span style={{ marginLeft: "8px", opacity: 0.85 }}>
                    · Панель: <strong>{parentMaster.label || parentMaster.host}</strong>
                  </span>
                )}
              </div>
            </div>
          </div>
          <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
            {onOpenPanel && parentMaster?.web_url && (
              <button
                type="button"
                className="win-button"
                style={{ height: "26px", fontSize: "11.5px", padding: "0 9px", display: "inline-flex", alignItems: "center", gap: "4px" }}
                onClick={onOpenPanel}
                title={`Открыть панель управления ${parentMaster.web_url}`}
              >
                <ExternalLink size={12} />
                <span>Открыть веб-панель</span>
              </button>
            )}
            <button
              className="win-button"
              style={{ padding: "0 8px", height: "26px" }}
              onClick={() => !loading && close()}
              title="Закрыть окно"
            >
              <X size={14} />
            </button>
          </div>
        </div>

        {/* Body */}
        <div style={{ padding: "16px 20px", display: "flex", flexDirection: "column", gap: "14px" }}>
          {/* Segmented Mode Selector (Native Windows 11 Dark Fluent Style) */}
          <div
            style={{
              display: "flex",
              alignItems: "center",
              background: "rgba(0, 0, 0, 0.28)",
              padding: "3px",
              borderRadius: "7px",
              border: "1px solid rgba(255, 255, 255, 0.08)",
              gap: "3px",
            }}
          >
            <button
              type="button"
              style={{
                flex: 1,
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                gap: "6px",
                height: "28px",
                fontSize: "12px",
                fontWeight: guideMode === "api" ? 600 : 400,
                color: guideMode === "api" ? "#ffffff" : "var(--win-text-secondary)",
                background: guideMode === "api" ? "rgba(255, 255, 255, 0.10)" : "transparent",
                borderRadius: "5px",
                border: guideMode === "api" ? "1px solid rgba(255, 255, 255, 0.14)" : "1px solid transparent",
                boxShadow: guideMode === "api" ? "0 1px 4px rgba(0, 0, 0, 0.3)" : "none",
                cursor: "pointer",
                transition: "all 0.15s ease",
              }}
              onClick={() => setGuideMode("api")}
            >
              <Zap size={13} style={{ color: guideMode === "api" ? "var(--win-node-accent)" : "inherit" }} />
              <span>Авто-привязка по API (Рекомендуется)</span>
            </button>
            <button
              type="button"
              style={{
                flex: 1,
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                gap: "6px",
                height: "28px",
                fontSize: "12px",
                fontWeight: guideMode === "manual" ? 600 : 400,
                color: guideMode === "manual" ? "#ffffff" : "var(--win-text-secondary)",
                background: guideMode === "manual" ? "rgba(255, 255, 255, 0.10)" : "transparent",
                borderRadius: "5px",
                border: guideMode === "manual" ? "1px solid rgba(255, 255, 255, 0.14)" : "1px solid transparent",
                boxShadow: guideMode === "manual" ? "0 1px 4px rgba(0, 0, 0, 0.3)" : "none",
                cursor: "pointer",
                transition: "all 0.15s ease",
              }}
              onClick={() => setGuideMode("manual")}
            >
              <span>Ручное создание в панели</span>
            </button>
          </div>

          {/* Mode 1: Auto-API Notice Banner */}
          {guideMode === "api" ? (
            <div
              style={{
                display: "flex",
                alignItems: "flex-start",
                gap: "10px",
                padding: "10px 14px",
                background: "rgba(56, 189, 248, 0.06)",
                border: "1px solid rgba(56, 189, 248, 0.20)",
                borderRadius: "7px",
                fontSize: "12px",
                lineHeight: 1.45,
              }}
            >
              <Check size={16} style={{ color: "var(--win-node-accent)", flexShrink: 0, marginTop: "1px" }} />
              <div>
                <div style={{ color: "var(--win-text-primary)", fontWeight: 600, fontSize: "12.5px" }}>
                  Нода уже автоматически создана в вашей веб-панели!
                </div>
                <div style={{ color: "var(--win-text-secondary)", marginTop: "2px" }}>
                  Вам <strong>не нужно</strong> создавать ноду заново и вписывать IP или порт вручную. Программа уже отправила конфигурацию в панель.
                </div>
              </div>
            </div>
          ) : null}

          {/* 3 Step Visual Progression Cards */}
          <div className="node-guide-steps" style={{ margin: 0 }}>
            {guideMode === "api" ? (
              <>
                {/* Step 1 for API */}
                <div className="node-guide-step">
                  <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
                    <span className="step-badge">1</span>
                    <span className="step-title">Кликните по ноде</span>
                  </div>
                  <div className="step-desc">
                    В панели перейдите в <b>Nodes</b> и нажмите на карточку этой ноды (<b>{server.host}</b>).
                  </div>
                </div>

                {/* Step 2 for API */}
                <div className="node-guide-step">
                  <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
                    <span className="step-badge">2</span>
                    <span className="step-title">Скопируйте ключ</span>
                  </div>
                  <div className="step-desc">
                    В открывшемся окне найдите строку <b>Secret Key (SECRET_KEY)</b> и нажмите иконку копирования.
                  </div>
                </div>

                {/* Step 3 for API */}
                <div className="node-guide-step">
                  <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
                    <span className="step-badge">3</span>
                    <span className="step-title">Вставьте сюда</span>
                  </div>
                  <div className="step-desc">
                    Вставьте ключ в поле ниже и нажмите <b>«Подключить ноду»</b> для запуска Reality.
                  </div>
                </div>
              </>
            ) : (
              <>
                {/* Step 1 for Manual */}
                <div className="node-guide-step">
                  <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
                    <span className="step-badge">1</span>
                    <span className="step-title">Создайте ноду</span>
                  </div>
                  <div className="step-desc">
                    В панели <b>Nodes → Create</b> укажите:
                    <div style={{ display: "flex", gap: "5px", flexWrap: "wrap", marginTop: "4px" }}>
                      <span
                        className="step-click-tag"
                        onClick={() => onCopy(server.host, "Хост")}
                        title="Нажмите, чтобы скопировать хост"
                      >
                        IP: {server.host}
                        <Copy size={9} style={{ opacity: 0.6 }} />
                      </span>
                      <span
                        className="step-click-tag"
                        onClick={() => onCopy("2222", "Порт")}
                        title="Нажмите, чтобы скопировать порт"
                      >
                        Порт: 2222
                        <Copy size={9} style={{ opacity: 0.6 }} />
                      </span>
                    </div>
                  </div>
                </div>

                {/* Step 2 for Manual */}
                <div className="node-guide-step">
                  <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
                    <span className="step-badge">2</span>
                    <span className="step-title">Скопируйте ключ</span>
                  </div>
                  <div className="step-desc">
                    Нажмите «Скопировать docker-compose.yml» либо скопируйте строку <b>SECRET_KEY</b>.
                  </div>
                </div>

                {/* Step 3 for Manual */}
                <div className="node-guide-step">
                  <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
                    <span className="step-badge">3</span>
                    <span className="step-title">Вставьте сюда</span>
                  </div>
                  <div className="step-desc">
                    Вставьте текст ниже — программа настроит ноду, сертификаты и запустит Reality.
                  </div>
                </div>
              </>
            )}
          </div>

          {/* Textarea Input Container */}
          <div>
            <div className="node-input-header">
              <div className="node-input-label">
                <TerminalIcon size={13} style={{ color: "var(--win-accent)" }} />
                <span>Конфигурация docker-compose или SECRET_KEY:</span>
              </div>

              <div style={{ display: "flex", alignItems: "center", gap: "6px" }}>
                {secretKey.includes("services:") || secretKey.includes("remnanode:") ? (
                  <span className="node-format-badge compose">
                    <Check size={11} /> Docker Compose
                  </span>
                ) : secretKey.includes("SECRET_KEY=") || secretKey.trim().startsWith("eyJ") ? (
                  <span className="node-format-badge secret">
                    <Check size={11} /> SECRET_KEY
                  </span>
                ) : secretKey.trim() ? (
                  <span className="node-format-badge secret">
                    Ключ введен
                  </span>
                ) : (
                  <span className="node-format-badge empty">
                    Ожидание данных
                  </span>
                )}

                <button
                  type="button"
                  className="table-action-btn"
                  onClick={handlePaste}
                  title="Вставить из буфера обмена"
                >
                  <Copy size={11} />
                  <span>Вставить</span>
                </button>
              </div>
            </div>

            <textarea
              className="win-textarea node-textarea"
              style={{
                fontFamily: "var(--win-font-mono)",
                fontSize: "12px",
                lineHeight: 1.5,
                height: "110px",
                width: "100%",
                boxSizing: "border-box",
                background: "var(--win-fill-card-secondary)",
                borderColor: secretKey.trim() ? "var(--win-accent)" : "var(--win-stroke-card)",
              }}
              placeholder="Вставьте сюда скопированный SECRET_KEY (начинается с eyJ...) или весь блок docker-compose.yml"
              value={secretKey}
              onChange={(e) => onChangeSecretKey(e.target.value)}
              disabled={loading}
            />
          </div>
        </div>

        {/* Footer */}
        <div
          className="win-modal-footer"
          style={{
            flexShrink: 0,
            padding: "14px 22px",
            background: "rgba(0, 0, 0, 0.22)",
            borderTop: "1px solid var(--win-card-stroke)",
            display: "flex",
            alignItems: "center",
            justifyContent: "flex-end",
            gap: "10px",
          }}
        >
          <button
            type="button"
            className="win-button"
            style={{ height: "32px", fontSize: "12.5px", padding: "0 18px" }}
            onClick={() => !loading && close()}
            disabled={loading}
          >
            Отмена
          </button>

          <button
            type="button"
            className="win-button hero-primary"
            style={{
              height: "32px",
              fontSize: "12.5px",
              padding: "0 20px",
              display: "inline-flex",
              alignItems: "center",
              gap: "7px",
              minWidth: "160px",
              justifyContent: "center",
            }}
            disabled={!secretKey.trim() || loading}
            onClick={onSubmit}
          >
            {loading ? (
              <>
                <RefreshCw size={13} className="animate-spin" />
                <span>Применение ключа...</span>
              </>
            ) : (
              <>
                <Zap size={13} />
                <span>Подключить ноду</span>
              </>
            )}
          </button>
        </div>
        </>
      )}
    </WinModal>
  );
};
