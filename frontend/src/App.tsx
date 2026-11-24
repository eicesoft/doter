import { useCallback, useEffect, useRef, useState } from "react";
import { Events } from "@wailsio/runtime";

import type { Alert } from "../bindings/doter/internal/alerts/models";
import type { Snapshot as NoticeSnapshot } from "../bindings/doter/internal/notice/models";
import * as IslandService from "../bindings/doter/internal/services/islandservice";
import { CollapsedBar } from "./components/CollapsedBar";
import { ExpandedPanel } from "./components/ExpandedPanel";
import { useAgentSessions } from "./hooks/useAgentSessions";
import { useBackendSettings } from "./hooks/useBackendSettings";
import { useClock } from "./hooks/useClock";
import { useSeries } from "./hooks/useSeries";
import { useSystemInfo } from "./hooks/useSystemInfo";
import { useWeather } from "./hooks/useWeather";
import { TAB_OPTIONS, loadSettings, saveSettings, type Settings, type TabKey } from "./lib/settings";
import { appLabel } from "./lib/format";

/** Wait for the collapse transition before shrinking the native window. */
const COLLAPSE_DELAY_MS = 460;

/** Let the island grow most of the way before the native frost fades in. */
const GLASS_DELAY_MS = 200;

/** How long an agent toast hangs below the pill. */
const TOAST_MS = 6000;

/** Agent alerts that get an in-app bubble; other alerts stay native-only. */
const TOAST_LABELS: Record<string, string> = {
  "agent-idle": "等待输入",
  "agent-done": "已完成",
};

/** One bubble shown under the idle pill. */
interface Toast {
  kind: string;
  title: string;
  detail: string;
}

function App() {
  const { snapshot, staticInfo } = useSystemInfo();
  const agents = useAgentSessions();
  const weather = useWeather();
  const clock = useClock();

  const cpuSeries = useSeries(snapshot?.cpu.usage ?? 0, snapshot?.timestamp ?? 0, 48);
  const netSeries = useSeries(snapshot?.network.downSpeed ?? 0, snapshot?.timestamp ?? 0, 48);

  const [open, setOpen] = useState(false);
  // Drives the island's opaque backdrop: it only turns transparent once the
  // native frost is on screen, so the panel never looks half-finished.
  const [glassOn, setGlassOn] = useState(false);
  const [pinned, setPinned] = useState(false);
  const [activeTab, setActiveTab] = useState<TabKey>("overview");
  const [settings, setSettings] = useState<Settings>(loadSettings);
  const backend = useBackendSettings();
  const [toast, setToast] = useState<Toast | null>(null);

  // Agent alerts already carry the completion rules (run length, cooldown), so
  // the bubble mirrors the native notification instead of re-deriving state.
  useEffect(() => {
    return Events.On("alerts:update", (event) => {
      const alert = event.data as Alert;
      if (!alert || !TOAST_LABELS[alert.kind]) return;
      setToast({ kind: alert.kind, title: TOAST_LABELS[alert.kind], detail: alert.body });
    });
  }, []);

  // Other apps' notifications arrive from the platform store. The panel keeps
  // the full history; the pill only shows the newest arrival.
  useEffect(() => {
    return Events.On("notices:update", (event) => {
      const snapshot = event.data as NoticeSnapshot;
      const entry = snapshot?.new?.[0];
      if (!entry) return;
      setToast({
        kind: "notice",
        title: appLabel(entry.app),
        detail: entry.title || entry.body,
      });
    });
  }, []);

  // The bubble hangs below the pill; the native window has to reserve that
  // room first and release it once the bubble is gone.
  useEffect(() => {
    IslandService.SetToastVisible(toast !== null).catch(() => {});
  }, [toast]);

  useEffect(() => {
    if (!toast) return;
    const timer = window.setTimeout(() => setToast(null), TOAST_MS);
    return () => window.clearTimeout(timer);
  }, [toast]);

  // The dashboard covers the toast area; drop it instead of squeezing it.
  useEffect(() => {
    if (open) setToast(null);
  }, [open]);

  // Preferences live in the webview: they only affect presentation, and the
  // dashboard must still come up if storage is unavailable.
  useEffect(() => {
    saveSettings(settings);
  }, [settings]);

  // Keep the dashboard on a page that is still enabled; disabling the current
  // tab lands on the first visible one (system overview is always first and
  // cannot be hidden).
  const fallbackTab: TabKey =
    TAB_OPTIONS.find((option) => settings.tabs[option.key])?.key ?? "overview";
  useEffect(() => {
    if (!settings.tabs[activeTab]) setActiveTab(fallbackTab);
  }, [activeTab, fallbackTab, settings.tabs]);

  const openRef = useRef(false);
  const pinnedRef = useRef(false);
  const collapseTimer = useRef<number | null>(null);
  const glassTimer = useRef<number | null>(null);

  const clearGlassTimer = useCallback(() => {
    if (glassTimer.current !== null) {
      window.clearTimeout(glassTimer.current);
      glassTimer.current = null;
    }
  }, []);

  const expand = useCallback(() => {
    if (collapseTimer.current !== null) {
      window.clearTimeout(collapseTimer.current);
      collapseTimer.current = null;
    }
    if (openRef.current) return;

    // Update the UI immediately; the native window grows in parallel. Resizing
    // the webview from inside an awaited binding call can stall the response,
    // so this call is intentionally fire-and-forget.
    openRef.current = true;
    setOpen(true);
    IslandService.SetExpanded(true).catch(() => {});

    // Reveal the native frost only once the island has nearly finished growing:
    // the native layer covers the whole expanded island, so showing it up front
    // would paint over space the island has not reached yet.
    clearGlassTimer();
    glassTimer.current = window.setTimeout(() => {
      glassTimer.current = null;
      if (openRef.current) {
        // The native layer is up: drop the opaque backdrop so the blur shows.
        setGlassOn(true);
        IslandService.SetGlassVisible(true).catch(() => {});
      }
    }, GLASS_DELAY_MS);
  }, [clearGlassTimer]);

  const collapse = useCallback((force = false) => {
    if (!openRef.current) return;
    if (pinnedRef.current && !force) return;

    if (force) {
      pinnedRef.current = false;
      setPinned(false);
    }

    openRef.current = false;
    setOpen(false);
    setGlassOn(false);
    clearGlassTimer();
    // Hide the native glass the moment the island starts shrinking; the
    // window itself only resizes once the animation has finished.
    IslandService.SetGlassVisible(false).catch(() => {});

    if (collapseTimer.current !== null) window.clearTimeout(collapseTimer.current);
    collapseTimer.current = window.setTimeout(() => {
      collapseTimer.current = null;
      if (!openRef.current) {
        IslandService.SetExpanded(false).catch(() => {});
      }
    }, COLLAPSE_DELAY_MS);
  }, [clearGlassTimer]);

  const togglePin = useCallback(() => {
    const next = !pinnedRef.current;
    pinnedRef.current = next;
    setPinned(next);
    if (next) expand();
  }, [expand]);

  const focusIsland = useCallback(() => {
    // Activating on pin hover means the following click is delivered to the
    // webview instead of being consumed by window activation.
    IslandService.Focus().catch(() => {});
  }, []);

  useEffect(() => {
    // The native window snaps to the expanded size before the CSS transition
    // runs. Percentages would jump with it, so lock the idle size in pixels
    // while the window still shows the collapsed frame: the island then grows
    // smoothly from the pill instead of appearing at full size.
    const root = document.documentElement;
    // Leave one top-shoulder width on each side (--island-shoulder in
    // styles.css), so the inverted corners have room inside the transparent window.
    root.style.setProperty("--island-idle-width", `${window.innerWidth - 24}px`);
    root.style.setProperty("--island-idle-height", `${window.innerHeight - 8}px`);
  }, []);

  useEffect(() => {
    // Native hover events fire even while the app is inactive, where the DOM
    // enter/leave handlers below stay silent.
    let leaveTimer: number | null = null;
    const unsubscribe = Events.On("island:hover", (event) => {
      if (leaveTimer !== null) {
        window.clearTimeout(leaveTimer);
        leaveTimer = null;
      }
      if (event.data === true) {
        if (settings.hoverExpand) expand();
        return;
      }
      // Resizing the window on expand can bounce the tracking area; this delay
      // keeps that from collapsing the island right after it opened.
      leaveTimer = window.setTimeout(() => {
        leaveTimer = null;
        collapse();
      }, 120);
    });

    return () => {
      if (leaveTimer !== null) window.clearTimeout(leaveTimer);
      unsubscribe();
    };
  }, [expand, collapse, settings.hoverExpand]);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        collapse(true);
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [collapse]);

  useEffect(() => {
    // A global shortcut or a notification click asks the native side to
    // reveal the island; this is the half that pins it and switches tabs.
    const unsubscribe = Events.On("island:open-tab", (event) => {
      const tab = event.data as TabKey;
      if (!tab) return;
      setActiveTab(settings.tabs[tab] ? tab : fallbackTab);
      pinnedRef.current = true;
      setPinned(true);
      expand();
    });
    return () => unsubscribe();
  }, [expand, fallbackTab, settings.tabs]);

  return (
    <div
      className="island"
      data-open={open}
      data-glass={glassOn}
      onMouseEnter={() => {
        if (settings.hoverExpand) void expand();
      }}
      onMouseLeave={() => collapse()}
      onClick={() => {
        if (!openRef.current) void expand();
      }}
    >
      <CollapsedBar
        snapshot={snapshot}
        clock={clock}
        agents={agents.snapshot?.running ?? 0}
        forecast={weather.forecast}
        metrics={settings.collapsed}
      />

      <ExpandedPanel
        snapshot={snapshot}
        agentSnapshot={agents.snapshot}
        staticInfo={staticInfo}
        cpuSeries={cpuSeries}
        netSeries={netSeries}
        clock={clock}
        weather={weather}
        pinned={pinned}
        onTogglePin={togglePin}
        onFocusIsland={focusIsland}
        onDismiss={() => collapse(true)}
        activeTab={activeTab}
        onSelectTab={setActiveTab}
        settings={settings}
        onChangeSettings={setSettings}
        backendSettings={backend.settings}
        onSetAlertsEnabled={backend.setAlertsEnabled}
        onSetClipboardPaused={backend.setClipboardPaused}
        onSetClipboardHotkeyEnabled={backend.setClipboardHotkeyEnabled}
        onSetClipboardHistoryLimit={backend.setClipboardHistoryLimit}
        onSetClipboardPreviewBytes={backend.setClipboardPreviewBytes}
        onSetClipboardImageMaxBytes={backend.setClipboardImageMaxBytes}
        onSetClipboardAutoPaste={backend.setClipboardAutoPaste}
        onSetClipboardExcludedApps={backend.setClipboardExcludedApps}
        onSetAutostart={backend.setAutostart}
        onSetHideOnFullscreen={backend.setHideOnFullscreen}
      />

      {toast && !open && (
        <div className="agent-toast" data-kind={toast.kind} role="status">
          <span className="agent-toast-title">{toast.title}</span>
          <span className="agent-toast-detail">{toast.detail}</span>
        </div>
      )}
    </div>
  );
}

export default App;
