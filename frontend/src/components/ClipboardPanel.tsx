import { useEffect, useMemo, useState } from "react";

import type { Entry } from "../../bindings/doter/internal/clipboard/models";
import { useClipboard } from "../hooks/useClipboard";
import { formatBytes, formatRelTime } from "../lib/format";

const strokeProps = {
  fill: "none",
  stroke: "currentColor",
  strokeWidth: 1.8,
  strokeLinecap: "round" as const,
  strokeLinejoin: "round" as const,
  "aria-hidden": true,
};

interface ClipboardPanelProps {
  onFocusIsland: () => void;
}

/** The clipboard history tab. */
export function ClipboardPanel({ onFocusIsland }: ClipboardPanelProps) {
  const { entries, loading, search, setSearch, togglePin, remove, clear, copy } =
    useClipboard();
  const [copiedId, setCopiedId] = useState<string | null>(null);

  const filtered = useMemo(() => {
    const query = search.trim().toLowerCase();
    if (!query) return entries;
    // Image entries have no text to match against.
    return entries.filter(
      (entry) => entry.kind !== "image" && entry.content.toLowerCase().includes(query),
    );
  }, [entries, search]);

  // Keep the relative timestamps fresh while the panel is open.
  const [, setTick] = useState(0);
  useEffect(() => {
    const id = window.setInterval(() => setTick((n) => n + 1), 30_000);
    return () => window.clearInterval(id);
  }, []);

  const handleCopy = async (id: string) => {
    const ok = await copy(id);
    if (!ok) return;
    setCopiedId(id);
    window.setTimeout(() => setCopiedId((current) => (current === id ? null : current)), 1_500);
  };

  return (
    <div className="clipboard-panel">
      <div className="clipboard-toolbar">
        <div className="clipboard-search">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round">
            <circle cx="11" cy="11" r="6.5" />
            <path d="M16 16l4.5 4.5" />
          </svg>
          <input
            type="text"
            value={search}
            placeholder="搜索剪贴板"
            onPointerEnter={onFocusIsland}
            onPointerDown={onFocusIsland}
            onChange={(event) => setSearch(event.target.value)}
          />
          {search && (
            <button
              type="button"
              className="clipboard-search-clear"
              onClick={() => setSearch("")}
              aria-label="清除搜索"
            >
              ✕
            </button>
          )}
        </div>
        <button
          type="button"
          className="clipboard-clear"
          onClick={clear}
          disabled={entries.length === 0}
        >
          清空
        </button>
      </div>

      <div className="clipboard-list">
        {loading ? (
          <div className="empty-state">
            <span className="power-glyph">⎘</span>
            <div>
              <b>读取剪贴板…</b>
            </div>
          </div>
        ) : filtered.length === 0 ? (
          <div className="empty-state">
            <span className="power-glyph">⎘</span>
            <div>
              <b>{search ? "没有匹配的内容" : "剪贴板还是空的"}</b>
              <span>
                {search ? "换个关键词试试" : "复制文字或图片后，这里会出现历史记录"}
              </span>
            </div>
          </div>
        ) : (
          filtered.map((entry) => (
            <ClipboardRow
              key={entry.id}
              entry={entry}
              copied={copiedId === entry.id}
              onCopy={() => void handleCopy(entry.id)}
              onPin={() => togglePin(entry.id)}
              onRemove={() => remove(entry.id)}
            />
          ))
        )}
      </div>
    </div>
  );
}

interface ClipboardRowProps {
  entry: Entry;
  copied: boolean;
  onCopy: () => void;
  onPin: () => void;
  onRemove: () => void;
}

function ClipboardRow({ entry, copied, onCopy, onPin, onRemove }: ClipboardRowProps) {
  const [expanded, setExpanded] = useState(false);
  const isImage = entry.kind === "image";
  const long = !isImage && entry.content.length > 120;

  return (
    <div className={entry.pinned ? "clip-row is-pinned" : "clip-row"}>
      <button
        type="button"
        className="clip-main"
        onClick={onCopy}
        title={entry.rich ? "点击复制，保留原始格式" : "点击复制到剪贴板"}
      >
        <span className={entry.kind === "url" ? "clip-kind is-url" : "clip-kind"}>
          {entry.kind === "url" ? (
            <svg viewBox="0 0 24 24" {...strokeProps}>
              <path d="M10.5 13.5a3.5 3.5 0 0 0 5 0l3-3a3.5 3.5 0 0 0-5-5L12 7" />
              <path d="M13.5 10.5a3.5 3.5 0 0 0-5 0l-3 3a3.5 3.5 0 0 0 5 5l1.5-1.5" />
            </svg>
          ) : isImage ? (
            <svg viewBox="0 0 24 24" {...strokeProps}>
              <rect x="3.5" y="4.5" width="17" height="15" rx="2.5" />
              <circle cx="9" cy="10" r="1.6" />
              <path d="M5 17l4.5-4.5 3 3L15 12l5 5" />
            </svg>
          ) : entry.kind === "color" ? (
            <span
              className="clip-swatch"
              style={{ background: entry.content.trim() }}
              aria-hidden="true"
            />
          ) : (
            <svg viewBox="0 0 24 24" {...strokeProps}>
              <path d="M5 7h14M5 12h14M5 17h9" />
            </svg>
          )}
        </span>
        <span className={isImage ? "clip-text is-image" : "clip-text"}>
          {isImage ? (
            <>
              <img className="clip-thumb" src={entry.content} alt="剪贴板图片" />
              <span className="clip-info">
                <span className="clip-info-title">
                  {entry.width ? `${entry.width} × ${entry.height}` : "图片"}
                </span>
                <span className="clip-meta">
                  <span>{formatBytes(entry.contentLen)}</span>
                  <span>PNG</span>
                  <span>{formatRelTime(entry.createdAt)}</span>
                </span>
              </span>
            </>
          ) : (
            <>
              <span className={expanded || !long ? "clip-content" : "clip-content is-clamped"}>
                {entry.content}
              </span>
              <span className="clip-meta">
                <span>{formatRelTime(entry.createdAt)}</span>
                {entry.rich && <span className="clip-rich">富文本</span>}
                {entry.contentLen > entry.content.length && (
                  <span className="clip-truncated">{entry.contentLen} 字节</span>
                )}
              </span>
            </>
          )}
        </span>
        {copied && <span className="clip-copied">已复制</span>}
      </button>

      <div className="clip-actions">
        {long && (
          <button
            type="button"
            className="clip-action"
            onClick={() => setExpanded((v) => !v)}
            title={expanded ? "收起" : "展开"}
          >
            {expanded ? "⌃" : "⌄"}
          </button>
        )}
        <button
          type="button"
          className={entry.pinned ? "clip-action is-pinned" : "clip-action"}
          onClick={onPin}
          title={entry.pinned ? "取消置顶" : "置顶"}
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
            <path d="M9 3h6l-1 6 3.5 3.5h-11L10 9 9 3Z" />
            <path d="M12 12.5V21" />
          </svg>
        </button>
        <button
          type="button"
          className="clip-action"
          onClick={onRemove}
          title="删除"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
            <path d="M5 7h14" />
            <path d="M9 7V5.5h6V7" />
            <path d="M7 7l1 12h8l1-12" />
          </svg>
        </button>
      </div>
    </div>
  );
}
