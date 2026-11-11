import { useCallback, useEffect, useRef, useState } from "react";
import { Events } from "@wailsio/runtime";

import * as IslandService from "../bindings/doter/internal/services/islandservice";
import { CollapsedBar } from "./components/CollapsedBar";
import { ExpandedPanel } from "./components/ExpandedPanel";
import { useClock } from "./hooks/useClock";
import { useSeries } from "./hooks/useSeries";
import { useSystemInfo } from "./hooks/useSystemInfo";

/** Wait for the collapse transition before shrinking the native window. */
const COLLAPSE_DELAY_MS = 460;

function App() {
  const { snapshot, staticInfo } = useSystemInfo();
  const clock = useClock();

  const cpuSeries = useSeries(snapshot?.cpu.usage ?? 0, snapshot?.timestamp ?? 0, 48);
  const netSeries = useSeries(snapshot?.network.downSpeed ?? 0, snapshot?.timestamp ?? 0, 48);

  const [open, setOpen] = useState(false);
  const [pinned, setPinned] = useState(false);

  const openRef = useRef(false);
  const pinnedRef = useRef(false);
  const collapseTimer = useRef<number | null>(null);

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
  }, []);

  const collapse = useCallback((force = false) => {
    if (!openRef.current) return;
    if (pinnedRef.current && !force) return;

    if (force) {
      pinnedRef.current = false;
      setPinned(false);
    }

    openRef.current = false;
    setOpen(false);

    if (collapseTimer.current !== null) window.clearTimeout(collapseTimer.current);
    collapseTimer.current = window.setTimeout(() => {
      collapseTimer.current = null;
      if (!openRef.current) {
        IslandService.SetExpanded(false).catch(() => {});
      }
    }, COLLAPSE_DELAY_MS);
  }, []);

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
    root.style.setProperty("--island-idle-width", `${window.innerWidth - 12}px`);
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
        expand();
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
  }, [expand, collapse]);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        collapse(true);
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [collapse]);

  return (
    <div
      className="island"
      data-open={open}
      onMouseEnter={() => void expand()}
      onMouseLeave={() => collapse()}
      onClick={() => {
        if (!openRef.current) void expand();
      }}
    >
      <CollapsedBar snapshot={snapshot} />

      <ExpandedPanel
        snapshot={snapshot}
        staticInfo={staticInfo}
        cpuSeries={cpuSeries}
        netSeries={netSeries}
        clock={clock}
        pinned={pinned}
        onTogglePin={togglePin}
        onFocusIsland={focusIsland}
      />
    </div>
  );
}

export default App;