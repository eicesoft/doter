import type { CSSProperties } from "react";

export type TabId = "overview" | "agents" | "weather" | "music" | "clipboard";

interface TabBarProps {
  active: TabId;
  onSelect: (id: TabId) => void;
  onFocusIsland: () => void;
}

const strokeProps = {
  fill: "none",
  stroke: "currentColor",
  strokeWidth: 1.8,
  strokeLinecap: "round" as const,
  strokeLinejoin: "round" as const,
  "aria-hidden": true,
};

/** Circular glass tab bar at the bottom of the expanded island. */
export function TabBar({ active, onSelect, onFocusIsland }: TabBarProps) {
  const tabs: { id: TabId; label: string; icon: React.ReactNode }[] = [
    {
      id: "overview",
      label: "系统总览",
      icon: (
        <svg viewBox="0 0 24 24" {...strokeProps}>
          <path d="M4.5 18a7.5 7.5 0 1 1 15 0" />
          <path d="M12 14.5 15.5 10" />
          <circle cx="12" cy="15" r="1.4" />
        </svg>
      ),
    },
    {
      id: "agents",
      label: "Agent 监控",
      icon: (
        <svg viewBox="0 0 24 24" {...strokeProps}>
          <rect x="3.5" y="4.5" width="17" height="15" rx="3.5" />
          <path d="M7.5 9 10 11.5 7.5 14M12.5 15h4" />
        </svg>
      ),
    },
    {
      id: "weather",
      label: "天气",
      icon: (
        <svg viewBox="0 0 24 24" {...strokeProps}>
          <circle cx="8.5" cy="8.5" r="2.8" />
          <path d="M8.5 3v1.4M3 8.5h1.4M4.6 4.6 5.6 5.6M12.4 4.6 11.4 5.6" />
          <path d="M9 19.5h8a3.2 3.2 0 0 0 .3-6.4 4.8 4.8 0 0 0-9.1.6A3.4 3.4 0 0 0 9 19.5Z" />
        </svg>
      ),
    },
    {
      id: "music",
      label: "音乐",
      icon: (
        <svg viewBox="0 0 24 24" {...strokeProps}>
          <path d="M9 18V6l10-2v12" />
          <circle cx="6.5" cy="18" r="2.5" />
          <circle cx="16.5" cy="16" r="2.5" />
        </svg>
      ),
    },
    {
      id: "clipboard",
      label: "剪切板",
      icon: (
        <svg viewBox="0 0 24 24" {...strokeProps}>
          <rect x="5" y="5" width="14" height="16" rx="3" />
          <path d="M9.5 5V3.8h5V5" />
          <path d="M9 12h6M9 16h4" />
        </svg>
      ),
    },
  ];

  const activeIndex = tabs.findIndex((tab) => tab.id === active);

  return (
    <nav
      className="tabbar"
      role="tablist"
      style={{ "--tab-index": activeIndex } as CSSProperties}
    >
      <span className="tab-indicator" aria-hidden="true" />
      {tabs.map((tab) => (
        <button
          key={tab.id}
          type="button"
          role="tab"
          aria-selected={active === tab.id}
          className={active === tab.id ? "tab is-active" : "tab"}
          title={tab.label}
          onPointerEnter={onFocusIsland}
          onPointerDown={(event) => {
            // The first click into an inactive window is swallowed by window
            // activation, so switch on pointerdown instead of click.
            event.preventDefault();
            event.stopPropagation();
            onSelect(tab.id);
          }}
        >
          {tab.icon}
        </button>
      ))}
    </nav>
  );
}
