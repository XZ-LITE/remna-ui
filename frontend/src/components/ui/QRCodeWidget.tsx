import React, { useState, useMemo } from "react";
import { Copy, ExternalLink, Download, QrCode, Globe, Layers, Check } from "lucide-react";
import { generateQRCodeSVGPath } from "./qrGenerator";
import { WinTooltip } from "./WinTooltip";

export interface QRCodeWidgetProps {
  panelUrl: string;
  subscriptionUrl?: string;
  onCopy?: (text: string) => void;
  onOpenUrl?: (url: string) => void;
}

export const QRCodeWidget: React.FC<QRCodeWidgetProps> = ({
  panelUrl,
  subscriptionUrl,
  onCopy,
  onOpenUrl,
}) => {
  const [activeType, setActiveType] = useState<"panel" | "sub">("panel");
  const [copied, setCopied] = useState(false);

  const currentUrl = activeType === "panel" ? panelUrl : (subscriptionUrl || panelUrl);

  const qrData = useMemo(() => {
    if (!currentUrl) return null;
    try {
      return generateQRCodeSVGPath(currentUrl);
    } catch (e) {
      console.error("QR Generation Error:", e);
      return null;
    }
  }, [currentUrl]);

  const handleCopy = () => {
    if (!currentUrl) return;
    navigator.clipboard.writeText(currentUrl);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
    if (onCopy) onCopy(currentUrl);
  };

  const handleDownloadSVG = () => {
    if (!qrData || !currentUrl) return;
    const padding = 3;
    const totalSize = qrData.size + padding * 2;
    const svgContent = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${totalSize} ${totalSize}" shape-rendering="crispEdges">
  <rect width="${totalSize}" height="${totalSize}" fill="#ffffff"/>
  <g transform="translate(${padding}, ${padding})">
    <path d="${qrData.path}" fill="#000000"/>
  </g>
</svg>`;
    const blob = new Blob([svgContent], { type: "image/svg+xml" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `remnawave-qr-${activeType}.svg`;
    a.click();
    URL.revokeObjectURL(url);
  };

  if (!panelUrl && !subscriptionUrl) {
    return (
      <div style={{ padding: "16px", textAlign: "center", color: "var(--win-text-tertiary)", fontSize: "12px" }}>
        URL адреса для генерации QR-кода не заданы.
      </div>
    );
  }

  const padding = 2;
  const totalSize = qrData ? qrData.size + padding * 2 : 25;

  return (
    <div className="win-qr-widget">
      {/* Type Toggle */}
      {subscriptionUrl && (
        <div className="win-qr-toggle-group">
          <button
            type="button"
            className={`win-qr-toggle-btn ${activeType === "panel" ? "active" : ""}`}
            onClick={() => setActiveType("panel")}
          >
            <Globe size={13} />
            <span>Панель (Gate)</span>
          </button>
          <button
            type="button"
            className={`win-qr-toggle-btn ${activeType === "sub" ? "active" : ""}`}
            onClick={() => setActiveType("sub")}
          >
            <Layers size={13} />
            <span>Подписки</span>
          </button>
        </div>
      )}

      {/* QR Code Container */}
      <div className="win-qr-canvas-box">
        {qrData ? (
          <div className="win-qr-svg-wrapper">
            <svg
              viewBox={`0 0 ${totalSize} ${totalSize}`}
              shapeRendering="crispEdges"
              style={{ width: "160px", height: "160px", display: "block" }}
            >
              <rect width={totalSize} height={totalSize} fill="#ffffff" rx={1.5} />
              <g transform={`translate(${padding}, ${padding})`}>
                <path d={qrData.path} fill="#0d1117" />
              </g>
            </svg>
          </div>
        ) : (
          <div style={{ width: "160px", height: "160px", display: "flex", alignItems: "center", justifyContent: "center" }}>
            <QrCode size={40} style={{ color: "var(--win-text-tertiary)" }} />
          </div>
        )}

        <div className="win-qr-meta">
          <div style={{ fontSize: "13px", fontWeight: 600, color: "var(--win-text-primary)", marginBottom: "4px" }}>
            {activeType === "panel" ? "QR-код веб-панели управления" : "QR-код страницы подписок"}
          </div>
          <div style={{ fontSize: "11.5px", color: "var(--win-text-secondary)", marginBottom: "12px", lineHeight: 1.4 }}>
            {activeType === "panel"
              ? "Отсканируйте камерой смартфона для защищенного перехода в панель администратора без ручного ввода ключа."
              : "Отсканируйте в приложении-клиенте (V2rayN, Streisand, Happ) для мгновенного добавления подписки."}
          </div>

          {/* URL Bar */}
          <div style={{ display: "flex", alignItems: "center", gap: "6px", marginBottom: "10px" }}>
            <input
              type="text"
              readOnly
              className="win-textbox"
              style={{ fontSize: "11px", height: "30px", flex: 1 }}
              value={currentUrl}
            />
            <WinTooltip content="Скопировать ссылку">
              <button
                type="button"
                className="win-button"
                style={{ height: "30px", padding: "0 10px" }}
                onClick={handleCopy}
              >
                {copied ? <Check size={13} style={{ color: "#10b981" }} /> : <Copy size={13} />}
                <span>{copied ? "Скопировано" : "Копировать"}</span>
              </button>
            </WinTooltip>
          </div>

          {/* Quick Actions */}
          <div style={{ display: "flex", gap: "6px" }}>
            {onOpenUrl && (
              <button
                type="button"
                className="win-button"
                style={{ height: "28px", fontSize: "11.5px", padding: "0 10px" }}
                onClick={() => onOpenUrl(currentUrl)}
              >
                <ExternalLink size={12} />
                <span>Открыть в браузере</span>
              </button>
            )}
            <WinTooltip content="Скачать векторный QR в формате SVG">
              <button
                type="button"
                className="win-button"
                style={{ height: "28px", fontSize: "11.5px", padding: "0 10px" }}
                onClick={handleDownloadSVG}
              >
                <Download size={12} />
                <span>Скачать SVG</span>
              </button>
            </WinTooltip>
          </div>
        </div>
      </div>
    </div>
  );
};
