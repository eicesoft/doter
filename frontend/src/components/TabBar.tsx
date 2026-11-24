import { useState } from "react";
import type { CSSProperties } from "react";

import type { TabKey } from "../lib/settings";

interface TabBarProps {
  active: TabKey;
  onSelect: (id: TabKey) => void;
  onFocusIsland: () => void;
  /** Pages the user kept enabled in settings. */
  enabled: Record<TabKey, boolean>;
  /** The weather tab takes the current sky's colour while it is active. */
  weatherAccent?: string;
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
export function TabBar({ active, onSelect, onFocusIsland, enabled, weatherAccent }: TabBarProps) {
  const allTabs: { id: TabKey; label: string; icon: React.ReactNode }[] = [
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
      id: "calendar",
      label: "日历",
      icon: (
        <svg viewBox="0 0 24 24" {...strokeProps}>
          <rect x="4" y="5.6" width="16" height="14.4" rx="3.2" />
          <path d="M4 10h16M8.4 3v4M15.6 3v4" />
          <path d="M8 14h2.2M13.8 14H16M8 17.2h2.2" />
        </svg>
      ),
    },
    {
      id: "notifications",
      label: "通知",
      icon: (
        <svg viewBox="0 0 24 24" {...strokeProps}>
          <path d="M6.2 16.5V11a5.8 5.8 0 0 1 11.6 0v5.5l1.7 2H4.5l1.7-2Z" />
          <path d="M10 20.2a2.2 2.2 0 0 0 4 0" />
        </svg>
      ),
    },
    {
      id: "clipboard",
      label: "剪切板",
      icon: (
        <svg viewBox="0 0 24 24" {...strokeProps}>
          <path d="M15.8 4.2h2a2.6 2.6 0 0 1 2.6 2.6v12.6a2.6 2.6 0 0 1-2.6 2.6H6.2a2.6 2.6 0 0 1-2.6-2.6V6.8a2.6 2.6 0 0 1 2.6-2.6h2" />
          <rect x="8.4" y="1.6" width="7.2" height="4.8" rx="1.8" />
          <path d="M8.6 11h6.8M8.6 15h4.4" />
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
      id: "translate",
      label: "翻译",
      icon: (
        <svg viewBox="0 0 24 24" {...strokeProps}>
          <rect x="3.4" y="4.2" width="10.6" height="9.4" rx="2.6" />
          <path d="M6.4 11.2 8.6 6.6l2.2 4.6M7.2 9.6h2.8" />
          <path d="M16.8 8.6h4M19 6.4l2.2 2.2L19 10.8" />
          <path d="M20.6 15.8h-4M18.4 13.6l-2.2 2.2 2.2 2.2" />
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
      id: "agents",
      label: "Agent 监控",
      icon: (
        <svg viewBox="0 0 24 24" {...strokeProps}>
          <circle cx="12" cy="2.6" r="1" />
          <path d="M12 3.6v2.6" />
          <rect x="4.4" y="6.2" width="15.2" height="12.6" rx="3.4" />
          <circle cx="9.1" cy="11.6" r="1.1" />
          <circle cx="14.9" cy="11.6" r="1.1" />
          <path d="M9.8 15.6h4.4" />
          <path d="M2.5 10.6v3.4M21.5 10.6v3.4" />
        </svg>
      ),
    },
  ];

  const tabs = allTabs.filter((tab) => enabled[tab.id]);
  const activeIndex = Math.max(
    tabs.findIndex((tab) => tab.id === active),
    0,
  );

  // One random glass recipe per app run: the rim angle, the sheen clusters and
  // the turbulence seeds all shift, so the highlights never repeat exactly.
  const [glass] = useState(() => ({
    seed: Math.floor(Math.random() * 900) + 1,
    rimAngle: Math.round(96 + Math.random() * 68),
    conicAngle: Math.round(168 + Math.random() * 64),
    x1: Math.round(8 + Math.random() * 18),
    x2: Math.round(36 + Math.random() * 22),
    x3: Math.round(60 + Math.random() * 20),
    x4: Math.round(84 + Math.random() * 12),
  }));

  return (
    <nav
      className="tabbar"
      role="tablist"
      style={
        {
          "--tab-index": activeIndex,
          "--glass-rim-angle": `${glass.rimAngle}deg`,
          "--glass-conic-angle": `${glass.conicAngle}deg`,
          "--glass-x1": `${glass.x1}%`,
          "--glass-x2": `${glass.x2}%`,
          "--glass-x3": `${glass.x3}%`,
          "--glass-x4": `${glass.x4}%`,
        } as CSSProperties
      }
    >
      {/* Organic refraction: turbulence displaces the rim and the specular
          sheen, so neither reads as a perfect geometric gradient. */}
      <svg className="tabbar-defs" aria-hidden="true" focusable="false">
        <filter id="tabbar-liquid-rim" x="-15%" y="-25%" width="130%" height="150%" colorInterpolationFilters="sRGB">
          <feTurbulence
            type="fractalNoise"
            baseFrequency="0.011 0.027"
            numOctaves={2}
            seed={glass.seed}
            result="rimNoise"
          />
          <feDisplacementMap
            in="SourceGraphic"
            in2="rimNoise"
            scale={3.2}
            xChannelSelector="R"
            yChannelSelector="G"
          />
        </filter>
        <filter id="tabbar-liquid-sheen" x="-20%" y="-30%" width="140%" height="160%" colorInterpolationFilters="sRGB">
          <feTurbulence
            type="fractalNoise"
            baseFrequency="0.016 0.038"
            numOctaves={2}
            seed={(glass.seed * 7) % 997}
            result="sheenNoise"
          />
          <feDisplacementMap
            in="SourceGraphic"
            in2="sheenNoise"
            scale={7}
            xChannelSelector="R"
            yChannelSelector="G"
          />
        </filter>
      </svg>
      <span className="tab-indicator" aria-hidden="true" />
      {tabs.map((tab) => (
        <button
          key={tab.id}
          type="button"
          role="tab"
          aria-selected={active === tab.id}
          className={active === tab.id ? "tab is-active" : "tab"}
          title={tab.label}
          style={
            tab.id === "weather" && active === tab.id && weatherAccent
              ? { color: weatherAccent }
              : undefined
          }
          onPointerEnter={onFocusIsland}
          onPointerDown={(event) => {
            // The first click into an inactive window is swallowed by window
            // activation, so switch on pointerdown instead of click.
            event.preventDefault();
            event.stopPropagation();
            onSelect(tab.id);
          }}
          onClick={(event) => {
            // Fallback for the activation edge: if the pointerdown was
            // consumed but the click still arrives, switch here too.
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
