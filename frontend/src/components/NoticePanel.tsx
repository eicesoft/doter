import { useEffect, useState } from "react";

import type { Entry } from "../../bindings/doter/internal/notice/models";
import { OpenApp, GetAppIcon, Clear, Remove } from "../../bindings/doter/internal/services/noticeservice";
import { useNotices } from "../hooks/useNotices";
import { appLabel, formatRelTime } from "../lib/format";
import { Card } from "./Card";

const strokeProps = {
  fill: "none",
  stroke: "currentColor",
  strokeWidth: 1.8,
  strokeLinecap: "round" as const,
  strokeLinejoin: "round" as const,
  "aria-hidden": true,
};

/** Notifications posted by other apps, read from the system store. */
export function NoticePanel() {
  const { snapshot } = useNotices();
  const entries = snapshot?.entries ?? [];
  const groups = groupByApp(entries);
  const [expanded, setExpanded] = useState<Set<string>>(() => new Set());

  const toggleExpand = (app: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(app)) next.delete(app);
      else next.add(app);
      return next;
    });
  };

  const remove = (ids: string[]) => {
    Remove(ids).catch(() => {});
  };

  // Relative timestamps stay fresh while the panel is visible.
  const [, setTick] = useState(0);
  useEffect(() => {
    const timer = window.setInterval(() => setTick((value) => value + 1), 10_000);
    return () => window.clearInterval(timer);
  }, []);

  return (
    <Card
      label="Notifications"
      inline={<span className="notice-count">{entries.length}</span>}
      actions={
        <button
          type="button"
          className="notice-clear"
          title="清空通知"
          aria-label="清空通知"
          disabled={entries.length === 0}
          onClick={() => {
            Clear().catch(() => {});
          }}
        >
          <svg viewBox="0 0 24 24" {...strokeProps}>
            <path d="M4.5 7h15" />
            <path d="M9 7V5.2A1.2 1.2 0 0 1 10.2 4h3.6A1.2 1.2 0 0 1 15 5.2V7" />
            <path d="M6.5 7l.8 12.1A1.9 1.9 0 0 0 9.2 21h5.6a1.9 1.9 0 0 0 1.9-1.9L17.5 7" />
            <path d="M10 11v6M14 11v6" />
          </svg>
        </button>
      }
      accent="var(--accent-net)"
      wide
    >
      <div className="agents-panel">
        <div className="agents-list">
          {snapshot?.error ? (
            <div className="agents-empty">
              <svg viewBox="0 0 24 24" {...strokeProps}>
                <path d="M12 8v5M12 16.5v.2" />
                <circle cx="12" cy="12" r="8.6" />
              </svg>
              <b>无法读取通知</b>
              <span>{snapshot.error}</span>
            </div>
          ) : entries.length === 0 ? (
            <div className="agents-empty">
              <svg viewBox="0 0 24 24" {...strokeProps}>
                <path d="M6.2 16.5V11a5.8 5.8 0 0 1 11.6 0v5.5l1.7 2H4.5l1.7-2Z" />
                <path d="M10 20.2a2.2 2.2 0 0 0 4 0" />
              </svg>
              <b>没有收到通知</b>
              <span>其他应用的系统通知会显示在这里</span>
            </div>
          ) : (
            groups.map(({ app, items }) => {
              if (items.length === 1) {
                return (
                  <NoticeRow
                    key={items[0].id}
                    entry={items[0]}
                    onRemove={() => remove([items[0].id])}
                  />
                );
              }
              // Four layers at most: the front row plus three edges peeking
              // out below it.
              const depth = Math.min(items.length, 4);
              const open = expanded.has(app);
              return (
                <div className="notice-group" key={app}>
                  <div
                    className="notice-stack"
                    style={open ? undefined : { marginBottom: (depth - 1) * 4 }}
                  >
                    {!open &&
                      Array.from({ length: depth - 1 }, (_, index) => {
                        const layer = depth - 1 - index;
                        return (
                          <span
                            key={layer}
                            className="notice-stack-layer"
                            style={{
                              left: layer * 8,
                              right: layer * 8,
                              top: layer * 4,
                              bottom: layer * -4,
                              // The newest card stays solid; each card behind
                              // it gets a little more transparent.
                              opacity: 1 - layer * 0.05,
                            }}
                          />
                        );
                      })}
                    <NoticeRow
                      entry={items[0]}
                      count={items.length}
                      expanded={open}
                      onClick={() => toggleExpand(app)}
                      onRemove={() => remove(items.map((entry) => entry.id))}
                    />
                  </div>
                  {open &&
                    items.slice(1).map((entry) => (
                      <NoticeRow
                        key={entry.id}
                        entry={entry}
                        nested
                        onRemove={() => remove([entry.id])}
                      />
                    ))}
                </div>
              );
            })
          )}
        </div>
      </div>
    </Card>
  );
}

/** Stacks notifications by app, newest stack first. Entries arrive sorted. */
function groupByApp(entries: Entry[]): { app: string; items: Entry[] }[] {
  const byApp = new Map<string, Entry[]>();
  for (const entry of entries) {
    const list = byApp.get(entry.app);
    if (list) list.push(entry);
    else byApp.set(entry.app, [entry]);
  }
  return Array.from(byApp, ([app, items]) => ({ app, items }));
}

interface NoticeRowProps {
  entry: Entry;
  /** Stack size; more than one renders a badge and toggles instead of opening. */
  count?: number;
  expanded?: boolean;
  /** Indented below its stack head. */
  nested?: boolean;
  onClick?: () => void;
  onRemove?: () => void;
}

function NoticeRow({ entry, count = 0, expanded, nested, onClick, onRemove }: NoticeRowProps) {
  const title = [entry.title, entry.body].filter(Boolean).join(" · ");
  const removeLabel = count > 1 ? `移除该应用的 ${count} 条通知` : "移除通知";

  return (
    <div className="notice-row" data-nested={nested || undefined}>
      <button
        type="button"
        className="notice-main"
        aria-expanded={count > 1 ? expanded : undefined}
        title={title || entry.app}
        onClick={() => {
          if (onClick) onClick();
          else OpenApp(entry.app).catch(() => {});
        }}
      >
        <AppIcon app={entry.app} />
        <span className="notice-text">
          <b>{entry.title || entry.body}</b>
          {entry.title && entry.body ? ` · ${entry.body}` : ""}
        </span>
        {count > 1 && <span className="notice-badge">{count}</span>}
        <span className="notice-time">{formatRelTime(entry.date)}</span>
      </button>
      {onRemove && (
        <button type="button" className="notice-remove" title={removeLabel} aria-label={removeLabel} onClick={onRemove}>
          <svg viewBox="0 0 24 24" {...strokeProps}>
            <path d="M5 7h14" />
            <path d="M9 7V5.5h6V7" />
            <path d="M7 7l1 12h8l1-12" />
          </svg>
        </button>
      )}
    </div>
  );
}

// One icon request per app, shared by every row that app posted.
const iconRequests = new Map<string, Promise<string>>();

function useAppIcon(app: string): string {
  const [icon, setIcon] = useState("");

  useEffect(() => {
    let active = true;
    let request = iconRequests.get(app);
    if (!request) {
      request = GetAppIcon(app).catch(() => "");
      iconRequests.set(app, request);
    }
    request.then((value) => {
      if (active) setIcon(value);
    });
    return () => {
      active = false;
    };
  }, [app]);

  return icon;
}

function AppIcon({ app }: { app: string }) {
  const icon = useAppIcon(app);
  if (icon) {
    return <img className="notice-icon" src={icon} alt="" />;
  }
  const label = appLabel(app);
  return (
    <span className="notice-icon is-fallback" aria-hidden="true">
      {label.slice(0, 1).toUpperCase()}
    </span>
  );
}
