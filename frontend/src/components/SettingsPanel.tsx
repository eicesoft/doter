import { useEffect, useState } from "react";
import type { ReactNode } from "react";

import type { Settings as BackendSettings } from "../../bindings/doter/internal/prefs/models";
import { Check, Version } from "../../bindings/doter/internal/services/updateservice";
import {
  CARD_OPTIONS,
  COLLAPSED_OPTIONS,
  TAB_OPTIONS,
  defaultSettings,
  type CardKey,
  type CollapsedMetric,
  type Settings,
  type TabKey,
} from "../lib/settings";

interface SettingsPanelProps {
  settings: Settings;
  onChange: (next: Settings) => void;
  backendSettings: BackendSettings;
  onSetAlertsEnabled: (enabled: boolean) => void;
  onSetClipboardPaused: (paused: boolean) => void;
  onSetClipboardHotkeyEnabled: (enabled: boolean) => void;
  onSetClipboardHistoryLimit: (limit: number) => void;
  onSetClipboardPreviewBytes: (bytes: number) => void;
  onSetClipboardImageMaxBytes: (bytes: number) => void;
  onSetClipboardAutoPaste: (enabled: boolean) => void;
  onSetClipboardExcludedApps: (apps: string[]) => void;
  onSetAutostart: (enabled: boolean) => void;
  onSetHideOnFullscreen: (enabled: boolean) => void;
  onClose: () => void;
}

type SettingsTab = "general" | "clipboard" | "system";

const TABS: readonly { id: SettingsTab; label: string }[] = [
  { id: "general", label: "界面" },
  { id: "clipboard", label: "剪贴板" },
  { id: "system", label: "系统" },
];

/** The full settings sheet opened from the header gear. It covers the whole
    tab area below the header, so the close button is the way back. */
export function SettingsPanel({
  settings,
  onChange,
  backendSettings,
  onSetAlertsEnabled,
  onSetClipboardPaused,
  onSetClipboardHotkeyEnabled,
  onSetClipboardHistoryLimit,
  onSetClipboardPreviewBytes,
  onSetClipboardImageMaxBytes,
  onSetClipboardAutoPaste,
  onSetClipboardExcludedApps,
  onSetAutostart,
  onSetHideOnFullscreen,
  onClose,
}: SettingsPanelProps) {
  const [version, setVersion] = useState("");
  const [tab, setTab] = useState<SettingsTab>("general");

  const setCard = (key: CardKey, value: boolean) => {
    onChange({ ...settings, cards: { ...settings.cards, [key]: value } });
  };
  const setCollapsed = (index: 0 | 1, metric: CollapsedMetric) => {
    const collapsed: [CollapsedMetric, CollapsedMetric] = [
      settings.collapsed[0],
      settings.collapsed[1],
    ];
    collapsed[index] = metric;
    onChange({ ...settings, collapsed });
  };
  const setTabVisible = (key: TabKey, value: boolean) => {
    if (key === "agents") return;
    onChange({ ...settings, tabs: { ...settings.tabs, [key]: value } });
  };

  useEffect(() => {
    Version()
      .then(setVersion)
      .catch(() => {});
  }, []);

  return (
    <div className="settings-panel" role="dialog" aria-label="设置">
      <div className="settings-tabs" role="tablist" aria-label="设置分类">
        {TABS.map((item) => (
          <button
            key={item.id}
            type="button"
            role="tab"
            aria-selected={tab === item.id}
            className={tab === item.id ? "settings-tab is-active" : "settings-tab"}
            onClick={() => setTab(item.id)}
          >
            {item.label}
          </button>
        ))}
        <button
          type="button"
          className="settings-panel-close"
          onClick={onClose}
          aria-label="关闭设置"
          title="关闭设置"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" aria-hidden="true">
            <path d="M6 6l12 12M18 6 6 18" />
          </svg>
        </button>
      </div>

      <div className="settings-panel-body">
        {tab === "general" && (
          <div className="settings-cols">
            <div className="settings-col">
              <Group title="界面">
                <ToggleRow
                  label="悬停展开"
                  checked={settings.hoverExpand}
                  onChange={(value) => onChange({ ...settings, hoverExpand: value })}
                />
                <ToggleRow
                  label="时钟显示秒"
                  checked={settings.clockSeconds}
                  onChange={(value) => onChange({ ...settings, clockSeconds: value })}
                />
              </Group>

              <Group title="显示指标">
                {CARD_OPTIONS.map((option) => (
                  <ToggleRow
                    key={option.key}
                    label={option.label}
                    accent={option.accent}
                    checked={settings.cards[option.key]}
                    onChange={(value) => setCard(option.key, value)}
                  />
                ))}
              </Group>
            </div>

            <div className="settings-col">
              <Group title="折叠胶囊">
                <SelectRow
                  label="左侧"
                  value={settings.collapsed[0]}
                  onChange={(value) => setCollapsed(0, value)}
                />
                <SelectRow
                  label="右侧"
                  value={settings.collapsed[1]}
                  onChange={(value) => setCollapsed(1, value)}
                />
                <div className="settings-note">
                  两个指标位可重复，默认 CPU 与天气。
                </div>
              </Group>

              <Group title="显示页签">
                {TAB_OPTIONS.map((option) => (
                  <ToggleRow
                    key={option.key}
                    label={option.label}
                    accent={option.accent}
                    checked={settings.tabs[option.key]}
                    locked={option.locked}
                    onChange={(value) => setTabVisible(option.key, value)}
                  />
                ))}
                <div className="settings-note">
                  「Agent 监控」始终显示，不能关闭。
                </div>
              </Group>
            </div>
          </div>
        )}

        {tab === "clipboard" && (
          <div className="settings-cols">
            <div className="settings-col">
              <Group title="记录">
                <ToggleRow
                  label="暂停记录"
                  accent="var(--accent-disk)"
                  checked={backendSettings.clipboardPaused}
                  onChange={onSetClipboardPaused}
                />
                <ToggleRow
                  label="复制后自动粘贴"
                  accent="var(--accent-net)"
                  checked={backendSettings.clipboardAutoPaste}
                  onChange={onSetClipboardAutoPaste}
                />
                <div className="settings-note">
                  自动粘贴会回到打开剪贴板前的应用。macOS 需要辅助功能权限；Linux 需要安装 xdotool（X11）。
                </div>
              </Group>

              <Group title="容量与预览">
                <NumberRow
                  label="历史条数"
                  value={backendSettings.clipboardHistoryLimit}
                  min={10}
                  max={1000}
                  unit="条"
                  onCommit={onSetClipboardHistoryLimit}
                />
                <NumberRow
                  label="文本预览"
                  value={Math.round(backendSettings.clipboardPreviewBytes / 1024)}
                  min={1}
                  max={64}
                  unit="KB"
                  onCommit={(value) => onSetClipboardPreviewBytes(value * 1024)}
                />
                <NumberRow
                  label="图片上限"
                  value={Math.round(backendSettings.clipboardImageMaxBytes / (1 << 20))}
                  min={1}
                  max={64}
                  unit="MB"
                  onCommit={(value) => onSetClipboardImageMaxBytes(value * (1 << 20))}
                />
                <div className="settings-note">
                  文本预览只限制面板里显示的字符数，完整内容仍会无损写回。
                </div>
              </Group>

              <Group title="快捷键">
                <ToggleRow
                  label="剪贴板快捷键"
                  accent="var(--accent-net)"
                  checked={backendSettings.clipboardHotkeyEnabled}
                  onChange={onSetClipboardHotkeyEnabled}
                />
                <div className="settings-note">⌘/Ctrl + Shift + V 打开剪贴板</div>
              </Group>
            </div>

            <div className="settings-col">
              <Group title="排除应用">
                <AppList
                  apps={backendSettings.clipboardExcludedApps ?? []}
                  onChange={onSetClipboardExcludedApps}
                />
                <div className="settings-note">
                  填写 Bundle ID（macOS）或进程名（Windows / Linux），不区分大小写，.exe 可省略。系统标记为隐藏 / 临时的内容仍会被跳过。
                </div>
                <div className="settings-note">
                  富文本（HTML / RTF）三平台均采集，Wayland 取决于合成器；Linux 的来源应用仅在 X11 下可用。
                </div>
              </Group>
            </div>
          </div>
        )}

        {tab === "system" && (
          <div className="settings-cols">
            <div className="settings-col">
              <Group title="提醒">
                <ToggleRow
                  label="桌面告警"
                  accent="var(--accent-warn)"
                  checked={backendSettings.alertsEnabled}
                  onChange={onSetAlertsEnabled}
                />
                <div className="settings-note">
                  通知权限由系统控制；被拒绝时告警只会记录到日志。
                </div>
              </Group>

              <Group title="应用">
                <ToggleRow
                  label="开机自启"
                  accent="var(--accent-cpu)"
                  checked={backendSettings.autostart}
                  onChange={onSetAutostart}
                />
                <ToggleRow
                  label="全屏时自动隐藏"
                  accent="var(--accent-net)"
                  checked={backendSettings.hideOnFullscreen}
                  onChange={onSetHideOnFullscreen}
                />
              </Group>
            </div>
          </div>
        )}
      </div>

      <footer className="settings-panel-foot">
        <span>Doter {version || "—"}</span>
        <div className="settings-panel-foot-actions">
          <button
            type="button"
            className="settings-check"
            onClick={() => void Check().catch(() => {})}
          >
            检查更新
          </button>
          <button
            type="button"
            className="settings-reset"
            onClick={() => onChange(defaultSettings())}
          >
            恢复界面默认
          </button>
        </div>
      </footer>
    </div>
  );
}

function Group({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="settings-group">
      <div className="settings-section">{title}</div>
      {children}
    </section>
  );
}

interface ToggleRowProps {
  label: string;
  checked: boolean;
  accent?: string;
  locked?: boolean;
  onChange: (value: boolean) => void;
}

interface SelectRowProps {
  label: string;
  value: CollapsedMetric;
  onChange: (value: CollapsedMetric) => void;
}

function SelectRow({ label, value, onChange }: SelectRowProps) {
  return (
    <label className="settings-select-row">
      <span>{label}</span>
      <select
        value={value}
        onChange={(event) => onChange(event.target.value as CollapsedMetric)}
      >
        {COLLAPSED_OPTIONS.map((option) => (
          <option key={option.key} value={option.key}>
            {option.label}
          </option>
        ))}
      </select>
    </label>
  );
}

function ToggleRow({ label, checked, accent, locked, onChange }: ToggleRowProps) {
  return (
    <label className={locked ? "settings-row is-locked" : "settings-row"} title={locked ? "此项始终开启" : undefined}>
      <input
        type="checkbox"
        checked={checked}
        disabled={locked}
        onChange={(event) => onChange(event.target.checked)}
      />
      <span className="settings-row-label">
        {accent && <span className="settings-dot" style={{ background: accent }} aria-hidden="true" />}
        {label}
      </span>
      <span className="settings-switch" aria-hidden="true">
        <span className="settings-knob" />
      </span>
    </label>
  );
}

interface NumberRowProps {
  label: string;
  value: number;
  min: number;
  max: number;
  unit: string;
  onCommit: (value: number) => void;
}

/** A number input that only writes back on blur or Enter, so typing does not
    fire a backend update for every digit. */
function NumberRow({ label, value, min, max, unit, onCommit }: NumberRowProps) {
  const [draft, setDraft] = useState(String(value));

  useEffect(() => {
    setDraft(String(value));
  }, [value]);

  const commit = () => {
    const next = Number(draft);
    if (!Number.isFinite(next) || next < min || next > max || next === value) {
      setDraft(String(value));
      return;
    }
    onCommit(next);
  };

  return (
    <label className="settings-number-row">
      <span>{label}</span>
      <span className="settings-number">
        <input
          type="number"
          min={min}
          max={max}
          value={draft}
          onChange={(event) => setDraft(event.target.value)}
          onBlur={commit}
          onKeyDown={(event) => {
            if (event.key === "Enter") event.currentTarget.blur();
          }}
        />
        <em>{unit}</em>
      </span>
    </label>
  );
}

interface AppListProps {
  apps: string[];
  onChange: (apps: string[]) => void;
}

/** A chip editor for the clipboard exclusion list. */
function AppList({ apps, onChange }: AppListProps) {
  const [draft, setDraft] = useState("");

  const add = () => {
    const value = draft.trim();
    if (!value) return;
    if (!apps.some((app) => app.toLowerCase() === value.toLowerCase())) {
      onChange([...apps, value]);
    }
    setDraft("");
  };

  return (
    <div className="settings-apps">
      <div className="settings-app-add">
        <input
          type="text"
          value={draft}
          placeholder="Bundle ID 或进程名"
          onChange={(event) => setDraft(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === "Enter") {
              event.preventDefault();
              add();
            }
          }}
        />
        <button type="button" onClick={add}>
          添加
        </button>
      </div>
      {apps.length === 0 ? (
        <div className="settings-note">没有排除项；隐藏 / 临时标记仍会生效。</div>
      ) : (
        <div className="settings-app-list">
          {apps.map((app) => (
            <span className="settings-app-chip" key={app}>
              <span title={app}>{app}</span>
              <button
                type="button"
                aria-label={`移除 ${app}`}
                onClick={() => onChange(apps.filter((item) => item !== app))}
              >
                ×
              </button>
            </span>
          ))}
        </div>
      )}
    </div>
  );
}
