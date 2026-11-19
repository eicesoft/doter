import { useEffect, useRef, useState } from "react";
import type { CSSProperties, ReactNode } from "react";

import type { Settings as BackendSettings } from "../../bindings/doter/internal/prefs/models";
import type { Snapshot, StaticInfo } from "../../bindings/doter/internal/sysinfo/models";
import { CARD_KEYS, type Settings } from "../lib/settings";
import {
  formatBytes,
  formatClock,
  formatDate,
  formatDuration,
  formatPercent,
  formatSpeed,
  formatUptime,
  platformLabel,
  shortCPUModel,
} from "../lib/format";
import { Meter, Ring } from "./Ring";
import { Sparkline } from "./Sparkline";
import { TabBar, type TabId } from "./TabBar";
import { ClipboardPanel } from "./ClipboardPanel";
import { AgentsPanel } from "./AgentsPanel";
import { MusicPanel } from "./MusicPanel";
import { WeatherPanel } from "./WeatherPanel";
import { SettingsPopover } from "./SettingsPopover";

interface ExpandedPanelProps {
  snapshot: Snapshot | null;
  staticInfo: StaticInfo | null;
  cpuSeries: number[];
  netSeries: number[];
  clock: Date;
  pinned: boolean;
  onTogglePin: () => void;
  onFocusIsland: () => void;
  onDismiss: () => void;
  activeTab: TabId;
  onSelectTab: (id: TabId) => void;
  settings: Settings;
  onChangeSettings: (next: Settings) => void;
  backendSettings: BackendSettings;
  onSetAlertsEnabled: (enabled: boolean) => void;
  onSetClipboardPaused: (paused: boolean) => void;
  onSetClipboardHotkeyEnabled: (enabled: boolean) => void;
  onSetAutostart: (enabled: boolean) => void;
  onSetHideOnFullscreen: (enabled: boolean) => void;
}

interface CardProps {
  label: string;
  hint?: string;
  accent: string;
  wide?: boolean;
  /** Settings can switch a card off without unmounting its siblings. */
  hidden?: boolean;
  children: ReactNode;
}

function Card({ label, hint, accent, wide, hidden, children }: CardProps) {
  if (hidden) return null;
  return (
    <section
      className={wide ? "card card-wide" : "card"}
      style={{ "--accent": accent } as CSSProperties}
    >
      <div className="card-head">
        <span className="card-label">{label}</span>
        {hint && <span className="card-hint">{hint}</span>}
      </div>
      <div className="card-body">{children}</div>
    </section>
  );
}

function KV({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="kv">
      <span>{label}</span>
      <b>{value}</b>
    </div>
  );
}

/** The hover dashboard. */
export function ExpandedPanel({
  snapshot,
  staticInfo,
  cpuSeries,
  netSeries,
  clock,
  pinned,
  onTogglePin,
  onFocusIsland,
  onDismiss,
  activeTab,
  onSelectTab,
  settings,
  onChangeSettings,
  backendSettings,
  onSetAlertsEnabled,
  onSetClipboardPaused,
  onSetClipboardHotkeyEnabled,
  onSetAutostart,
  onSetHideOnFullscreen,
}: ExpandedPanelProps) {
  const [settingsOpen, setSettingsOpen] = useState(false);
  const settingsWrapRef = useRef<HTMLDivElement>(null);

  const cpuUsage = snapshot?.cpu.usage ?? 0;
  const perCore = snapshot?.cpu.perCore ?? [];
  const memory = snapshot?.memory;
  const swap = snapshot?.swap;
  const disks = snapshot?.disks ?? [];
  const network = snapshot?.network;
  const battery = snapshot?.battery;
  const load = snapshot?.load;
  const processes = snapshot?.processes ?? [];

  const hostname = staticInfo?.hostname || "localhost";
  const platform = staticInfo
    ? platformLabel(staticInfo.os, staticInfo.platform, staticInfo.platformVersion)
    : "System";
  const threads = staticInfo?.cpuThreads || perCore.length || 0;
  const cores = staticInfo?.cpuCores || 0;
  const primaryDisk = disks[0];
  const secondaryDisk = disks[1];
  const netInterface = network?.interfaces?.[0]?.name;

  // Dismiss the popover on an outside press or Esc. Esc is handled in the
  // capture phase so it closes the popover instead of collapsing the island.
  useEffect(() => {
    if (!settingsOpen) return;
    const onPointerDown = (event: PointerEvent) => {
      if (!settingsWrapRef.current?.contains(event.target as Node)) setSettingsOpen(false);
    };
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "Escape") return;
      event.stopPropagation();
      setSettingsOpen(false);
    };
    document.addEventListener("pointerdown", onPointerDown);
    document.addEventListener("keydown", onKeyDown, true);
    return () => {
      document.removeEventListener("pointerdown", onPointerDown);
      document.removeEventListener("keydown", onKeyDown, true);
    };
  }, [settingsOpen]);

  return (
    <div className="panel">
      <header className="panel-header">
        <div className="panel-identity">
          <span className="brand-dot is-live" />
          <div>
            <div className="panel-brand">DOTER</div>
            <div className="panel-host">
              {hostname} <span className="dot-sep">·</span> {platform}
            </div>
          </div>
        </div>

        <div className="panel-header-right">
          <div className="panel-clock-block">
            <span className="panel-clock">{formatClock(clock, settings.clockSeconds)}</span>
            <span className="panel-date">{formatDate(clock)}</span>
          </div>
          <div className="panel-actions" ref={settingsWrapRef}>
            <button
              type="button"
              className={settingsOpen ? "settings-button is-open" : "settings-button"}
              onPointerEnter={onFocusIsland}
              onPointerDown={onFocusIsland}
              onClick={() => setSettingsOpen((value) => !value)}
              title="设置"
              aria-label="设置"
              aria-haspopup="dialog"
              aria-expanded={settingsOpen}
            >
              <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.8"
                strokeLinecap="round"
                strokeLinejoin="round"
                aria-hidden="true"
              >
                <path d="M4 8h7M17 8h3M4 16h3M13 16h7" />
                <circle cx="14" cy="8" r="2.2" />
                <circle cx="10" cy="16" r="2.2" />
              </svg>
            </button>
            <button
              type="button"
              className={pinned ? "pin-button is-on" : "pin-button"}
              onPointerEnter={onFocusIsland}
              onPointerDown={(event) => {
                // Pin on pointerdown: the first click into an inactive window is
                // consumed by Windows activation, so waiting for `click` loses it.
                event.preventDefault();
                event.stopPropagation();
                onTogglePin();
              }}
              title={pinned ? "Unpin dashboard" : "Keep dashboard open"}
              aria-pressed={pinned}
            >
              <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.8"
                strokeLinecap="round"
                strokeLinejoin="round"
                aria-hidden="true"
              >
                <path d="M9 3h6l-1 6 3.5 3.5h-11L10 9 9 3Z" />
                <path d="M12 12.5V21" />
              </svg>
            </button>

            {settingsOpen && (
              <SettingsPopover
                settings={settings}
                onChange={onChangeSettings}
                backendSettings={backendSettings}
                onSetAlertsEnabled={onSetAlertsEnabled}
                onSetClipboardPaused={onSetClipboardPaused}
                onSetClipboardHotkeyEnabled={onSetClipboardHotkeyEnabled}
                onSetAutostart={onSetAutostart}
                onSetHideOnFullscreen={onSetHideOnFullscreen}
                onClose={() => setSettingsOpen(false)}
              />
            )}
          </div>
        </div>
      </header>

      <div
        className={settings.cards.processes ? "panel-body has-wide" : "panel-body"}
        data-tab={activeTab}
        key={activeTab}
      >
        {activeTab === "overview" && (
          <>
            <Card
              label="Processor"
              hint={shortCPUModel(staticInfo?.cpuModel ?? "")}
              accent="var(--accent-cpu)"
              hidden={!settings.cards.cpu}
            >
              <div className="cpu-split">
                <Ring value={cpuUsage} size={62} stroke={5} color="var(--accent-cpu)">
                  <span className="ring-value">
                    {Math.round(cpuUsage)}
                    <small>%</small>
                  </span>
                </Ring>
                <div className="cpu-facts">
                  <KV label="Threads" value={threads || "—"} />
                  <KV label="Cores" value={cores || "—"} />
                </div>
              </div>
              <Sparkline
                data={cpuSeries}
                max={100}
                height={20}
                color="var(--accent-cpu)"
                className="card-spark"
              />
            </Card>

            <Card
              label="Memory"
              hint={formatPercent(memory?.usedPercent ?? 0)}
              accent="var(--accent-mem)"
              hidden={!settings.cards.memory}
            >
              <div className="metric-big">
                {formatPercent(memory?.usedPercent ?? 0)}
                <span className="metric-sub">
                  {formatBytes(memory?.used ?? 0)} / {formatBytes(memory?.total ?? 0)}
                </span>
              </div>
              <Meter value={memory?.usedPercent ?? 0} color="var(--accent-mem)" />
              <div className="card-foot">
                <span>Swap</span>
                <span>
                  {swap && swap.total > 0
                    ? `${formatBytes(swap.used)} / ${formatBytes(swap.total)}`
                    : "None"}
                </span>
              </div>
            </Card>

            <Card
              label="Storage"
              hint={disks.length > 1 ? `${disks.length} volumes` : primaryDisk?.fstype}
              accent="var(--accent-disk)"
              hidden={!settings.cards.storage}
            >
              {primaryDisk ? (
                <div className="disk-list">
                  {[primaryDisk, secondaryDisk].filter(Boolean).map((disk) => (
                    <div className="disk-row" key={disk!.mountpoint}>
                      <div className="disk-meta">
                        <span className="disk-name" title={disk!.mountpoint}>
                          {disk!.mountpoint}
                        </span>
                        <span className="disk-pct">{formatPercent(disk!.usedPercent)}</span>
                      </div>
                      <Meter value={disk!.usedPercent} color="var(--accent-disk)" />
                      <div className="disk-sub">
                        {formatBytes(disk!.used)} of {formatBytes(disk!.total)}
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="empty-state">Reading volumes…</div>
              )}
            </Card>

            <Card
              label="Network"
              hint={netInterface ?? "offline"}
              accent="var(--accent-net)"
              hidden={!settings.cards.network}
            >
              <div className="net-speeds">
                <div className="net-speed">
                  <span className="net-arrow down">↓</span>
                  <b>{formatSpeed(network?.downSpeed ?? 0)}</b>
                </div>
                <div className="net-speed">
                  <span className="net-arrow up">↑</span>
                  <b>{formatSpeed(network?.upSpeed ?? 0)}</b>
                </div>
              </div>
              <Sparkline
                data={netSeries}
                height={24}
                color="var(--accent-net)"
                className="card-spark"
              />
            </Card>

            <Card
              label="Power"
              hint={battery ? battery.state : "AC"}
              accent="var(--accent-battery)"
              hidden={!settings.cards.power}
            >
              {battery ? (
                <>
                  <div className="metric-big">
                    {formatPercent(battery.percent)}
                    <span className="metric-sub">
                      {battery.secondsLeft >= 0
                        ? `${formatDuration(battery.secondsLeft)} left`
                        : battery.pluggedIn
                          ? "on power"
                          : "estimating…"}
                    </span>
                  </div>
                  <Meter value={battery.percent} color="var(--accent-battery)" />
                  <div className="card-foot">
                    <span>{battery.pluggedIn ? "Charger connected" : "On battery"}</span>
                    <span>{battery.state}</span>
                  </div>
                </>
              ) : (
                <div className="empty-state">
                  <span className="power-glyph">⚡</span>
                  <div>
                    <b>AC power</b>
                    <span>No battery detected</span>
                  </div>
                </div>
              )}
            </Card>

            <Card
              label="System"
              accent="var(--accent-sys)"
              hidden={!settings.cards.system}
            >
              <div className="facts">
                <KV label="Uptime" value={formatUptime(snapshot?.uptime ?? 0)} />
                <KV label="Load 1m" value={load ? load.load1.toFixed(2) : "—"} />
                <KV label="Arch" value={staticInfo?.kernelArch || "—"} />
              </div>
            </Card>

            <Card
              label="Top processes"
              hint="CPU · MEM"
              accent="var(--accent-cpu)"
              wide
              hidden={!settings.cards.processes}
            >
              <div className="proc-table">
                {processes.length === 0 ? (
                  <div className="empty-state">Collecting process data…</div>
                ) : (
                  processes.map((process) => (
                    <div className="proc-row" key={process.pid}>
                      <span className="proc-name" title={process.name}>
                        {process.name}
                      </span>
                      <span className="proc-pid">{process.pid}</span>
                      <span className="proc-cpu">{formatPercent(process.cpuPercent, 1)}</span>
                      <span className="proc-mem">{formatPercent(process.memoryPercent, 1)}</span>
                    </div>
                  ))
                )}
              </div>
            </Card>

            {!CARD_KEYS.some((key) => settings.cards[key]) && (
              <div className="overview-empty">
                <svg
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="1.8"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  aria-hidden="true"
                >
                  <path d="M4 8h7M17 8h3M4 16h3M13 16h7" />
                  <circle cx="14" cy="8" r="2.2" />
                  <circle cx="10" cy="16" r="2.2" />
                </svg>
                <div>
                  <b>没有启用的指标</b>
                  <span>在右上角的设置里打开需要的卡片</span>
                </div>
              </div>
            )}
          </>
        )}

        {activeTab === "agents" && (
          <Card label="Agents" hint="Claude · Codex" accent="var(--accent-cpu)" wide>
            <AgentsPanel />
          </Card>
        )}

        {activeTab === "weather" && (
          <Card label="Weather" accent="var(--accent-net)" wide>
            <WeatherPanel onFocusIsland={onFocusIsland} />
          </Card>
        )}

        {activeTab === "music" && (
          <Card label="Music" accent="var(--accent-net)" wide>
            <MusicPanel />
          </Card>
        )}

        {activeTab === "clipboard" && (
          <Card label="Clipboard" accent="var(--accent-disk)" wide>
            <ClipboardPanel
              onFocusIsland={onFocusIsland}
              paused={backendSettings.clipboardPaused}
              onTogglePause={() => onSetClipboardPaused(!backendSettings.clipboardPaused)}
              onDismiss={onDismiss}
            />
          </Card>
        )}
      </div>

      <TabBar active={activeTab} onSelect={onSelectTab} onFocusIsland={onFocusIsland} />
    </div>
  );
}