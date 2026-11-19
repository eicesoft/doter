import { useCallback, useEffect, useMemo, useRef, useState } from "react";

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
  paused: boolean;
  onTogglePause: () => void;
  onDismiss: () => void;
}

/** The clipboard history tab. */
export function ClipboardPanel({
  onFocusIsland,
  paused,
  onTogglePause,
  onDismiss,
}: ClipboardPanelProps) {
  const { entries, loading, search, setSearch, togglePin, remove, clear, copy } =
    useClipboard();
  const [copiedId, setCopiedId] = useState<string | null>(null);
  const [cursor, setCursor] = useState(0);
  const listRef = useRef<HTMLDivElement>(null);
  const [showScrollTop, setShowScrollTop] = useState(false);

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

  const handleCopy = useCallback(
    async (id: string) => {
      const ok = await copy(id);
      if (!ok) return false;
      setCopiedId(id);
      window.setTimeout(() => setCopiedId((current) => (current === id ? null : current)), 1_500);
      return true;
    },
    [copy],
  );

  // A new copy or a different filter starts at the top of the list.
  useEffect(() => {
    setCursor(0);
  }, [entries, search]);

  useEffect(() => {
    setCursor((current) => Math.min(current, Math.max(filtered.length - 1, 0)));
  }, [filtered.length]);

  // Keep the keyboard selection visible.
  useEffect(() => {
    const row = listRef.current?.querySelector<HTMLElement>(`[data-clip-index="${cursor}"]`);
    row?.scrollIntoView({ block: "nearest" });
  }, [cursor, filtered.length]);

  // Keyboard-first use is the point of the global shortcut: arrows move,
  // Enter pastes and collapses, and Cmd/Ctrl+1…9 picks a row directly.
  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement | null;
      if (
        target &&
        (target.tagName === "INPUT" ||
          target.tagName === "TEXTAREA" ||
          target.tagName === "BUTTON" ||
          target.isContentEditable)
      ) {
        return;
      }
      if (filtered.length === 0) return;

      if (event.key === "ArrowDown" || event.key === "ArrowUp") {
        event.preventDefault();
        setCursor((current) => {
          const next = event.key === "ArrowDown" ? current + 1 : current - 1;
          return Math.max(0, Math.min(next, filtered.length - 1));
        });
        return;
      }

      const pick = (index: number) => {
        const entry = filtered[index];
        if (!entry) return;
        event.preventDefault();
        void handleCopy(entry.id).then((ok) => {
          if (ok) onDismiss();
        });
      };

      if (event.key === "Enter") {
        pick(cursor);
        return;
      }
      if ((event.metaKey || event.ctrlKey) && /^[1-9]$/.test(event.key)) {
        pick(Number(event.key) - 1);
      }
    };

    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [filtered, cursor, handleCopy, onDismiss]);

  const handleListScroll = () => {
    const el = listRef.current;
    if (!el) return;
    setShowScrollTop(el.scrollTop > 200);
  };

  const scrollToTop = () => {
    listRef.current?.scrollTo({ top: 0, behavior: "smooth" });
  };

  const handleSearchChange = (value: string) => {
    setSearch(value);
    // A filtered list starts at its own top, not wherever the old one sat.
    listRef.current?.scrollTo({ top: 0 });
    setShowScrollTop(false);
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
            onChange={(event) => handleSearchChange(event.target.value)}
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
          className={paused ? "clipboard-clear is-active" : "clipboard-clear"}
          onClick={onTogglePause}
          aria-pressed={paused}
          title={paused ? "继续记录剪贴板" : "暂停记录剪贴板"}
        >
          {paused ? "继续" : "暂停"}
        </button>
        <button
          type="button"
          className="clipboard-clear"
          onClick={clear}
          disabled={entries.length === 0}
        >
          清空
        </button>
      </div>

      {paused && (
        <div className="clipboard-paused" role="status">
          <span>剪贴板记录已暂停，复制的新内容不会保存。</span>
          <button type="button" onClick={onTogglePause}>
            继续记录
          </button>
        </div>
      )}

      <div className="clipboard-list" ref={listRef} onScroll={handleListScroll}>
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
          filtered.map((entry, index) => (
            <ClipboardRow
              key={entry.id}
              entry={entry}
              index={index}
              active={index === cursor}
              copied={copiedId === entry.id}
              onCopy={() => void handleCopy(entry.id)}
              onPin={() => togglePin(entry.id)}
              onRemove={() => remove(entry.id)}
            />
          ))
        )}
      </div>

      <button
        type="button"
        className="clip-scroll-top"
        data-visible={showScrollTop ? "true" : "false"}
        aria-label="回到顶部"
        aria-hidden={!showScrollTop}
        tabIndex={showScrollTop ? 0 : -1}
        onClick={scrollToTop}
      >
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
          <path d="M12 18.5V6" />
          <path d="M6.4 11.6 12 6l5.6 5.6" />
        </svg>
      </button>
    </div>
  );
}

interface ClipboardRowProps {
  entry: Entry;
  index: number;
  active: boolean;
  copied: boolean;
  onCopy: () => void;
  onPin: () => void;
  onRemove: () => void;
}

function ClipboardRow({
  entry,
  index,
  active,
  copied,
  onCopy,
  onPin,
  onRemove,
}: ClipboardRowProps) {
  const [expanded, setExpanded] = useState(false);
  const isImage = entry.kind === "image";
  const long = !isImage && entry.content.length > 120;
  const className = ["clip-row", entry.pinned ? "is-pinned" : "", active ? "is-active" : ""]
    .filter(Boolean)
    .join(" ");

  return (
    <div className={className} data-clip-index={index}>
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
                  {entry.source && (
                    <span className="clip-source" title={entry.source}>
                      {entry.source}
                    </span>
                  )}
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
                {entry.source && (
                  <span className="clip-source" title={entry.source}>
                    {entry.source}
                  </span>
                )}
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
