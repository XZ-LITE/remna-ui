import React, { useState, useMemo } from "react";
import { Terminal as TerminalIcon, Search, RotateCcw, RefreshCw, Copy, X, AlertTriangle, Wrench } from "lucide-react";
import * as types from "../../../bindings/remna-ui/internal/types/models";
import { WinModal } from "../ui/WinModal";

interface InspectLogsModalProps {
  server: types.SavedServer;
  container: string;
  logs: string;
  autoRefresh: boolean;
  onClose: () => void;
  onRefresh: () => Promise<void>;
  onToggleAutoRefresh: () => void;
  onCopy: (text: string, label: string) => void;
}

export const InspectLogsModal: React.FC<InspectLogsModalProps> = ({
  server,
  container,
  logs,
  autoRefresh,
  onClose,
  onRefresh,
  onToggleAutoRefresh,
  onCopy,
}) => {
  const [searchQuery, setSearchQuery] = useState("");

  const filteredLogs = useMemo(() => {
    if (!searchQuery.trim()) return logs;
    const q = searchQuery.toLowerCase();
    return logs
      .split("\n")
      .filter((line) => line.toLowerCase().includes(q))
      .join("\n");
  }, [logs, searchQuery]);

  const totalLines = useMemo(() => logs.split("\n").length, [logs]);

  return (
    <WinModal isOpen={true} onClose={onClose} dialogClassName="inspect-logs-dialog">
      <div className="win-modal-header">
          <div className="win-modal-title">
            <TerminalIcon size={16} color="var(--win-accent)" />
            <span>Журнал контейнера: {container}</span>
            <span style={{ fontSize: "11px", color: "var(--win-text-tertiary)", fontWeight: 400 }}>
              ({server.label || server.host})
            </span>
          </div>
          <button
            className="win-button"
            style={{ padding: "0 8px", height: "26px" }}
            onClick={onClose}
          >
            <X size={14} />
          </button>
        </div>

        {/* Modal Toolbar: Search & Auto-refresh */}
        <div className="win-modal-toolbar">
          <div className="win-modal-search">
            <Search size={13} className="win-modal-search-icon" />
            <input
              type="text"
              placeholder="Поиск по логам (например: error, ready, starting)..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
            {searchQuery && (
              <button
                className="win-modal-search-clear"
                onClick={() => setSearchQuery("")}
                title="Очистить фильтр"
              >
                <X size={14} />
              </button>
            )}
          </div>

          <button
            className="win-button"
            style={{ height: "28px", fontSize: "11.5px" }}
            onClick={onRefresh}
            title="Обновить журнал прямо сейчас"
          >
            <RotateCcw size={12} />
            <span>Обновить</span>
          </button>

          <button
            className={`win-button ${autoRefresh ? "primary" : ""}`}
            style={{ height: "28px", fontSize: "11.5px" }}
            onClick={onToggleAutoRefresh}
          >
            <RefreshCw
              size={12}
              className={autoRefresh ? "animate-spin" : ""}
            />
            <span>Авто-обновление (3с)</span>
          </button>

          <button
            className="win-button"
            style={{ height: "28px", fontSize: "11.5px" }}
            onClick={() => onCopy(logs, "Логи контейнера")}
          >
            <Copy size={12} />
            <span>Копировать</span>
          </button>
        </div>

        {/* Modal Body */}
        <div className="win-modal-body terminal-view">
          {filteredLogs || "Нет строк, соответствующих фильтру."}
        </div>

        {/* Modal Footer */}
        <div className="win-modal-footer">
          <span>Всего строк: {totalLines}</span>
          <button
            className="win-button primary"
            style={{ height: "26px", fontSize: "11.5px" }}
            onClick={onClose}
          >
            Закрыть
          </button>
        </div>
    </WinModal>
  );
};
