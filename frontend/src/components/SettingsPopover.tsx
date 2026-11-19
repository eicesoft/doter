import type { Settings as BackendSettings } from "../../bindings/doter/internal/prefs/models";
import {
  CARD_OPTIONS,
  defaultSettings,
  type CardKey,
  type Settings,
} from "../lib/settings";

interface SettingsPopoverProps {
  settings: Settings;
  onChange: (next: Settings) => void;
  backendSettings: BackendSettings;
  onSetAlertsEnabled: (enabled: boolean) => void;
  onSetClipboardPaused: (paused: boolean) => void;
  onSetClipboardHotkeyEnabled: (enabled: boolean) => void;
  onClose: () => void;
}

/** The settings popover opened from the header gear. */
export function SettingsPopover({
  settings,
  onChange,
  backendSettings,
  onSetAlertsEnabled,
  onSetClipboardPaused,
  onSetClipboardHotkeyEnabled,
  onClose,
}: SettingsPopoverProps) {
  const setCard = (key: CardKey, value: boolean) => {
    onChange({ ...settings, cards: { ...settings.cards, [key]: value } });
  };

  return (
    <div className="settings-popover" role="dialog" aria-label="设置">
      <div className="settings-title">
        <span>设置</span>
        <button type="button" className="settings-close" onClick={onClose} aria-label="关闭设置">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" aria-hidden="true">
            <path d="M6 6l12 12M18 6 6 18" />
          </svg>
        </button>
      </div>

      <div className="settings-section">界面</div>
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

      <div className="settings-section">显示指标</div>
      {CARD_OPTIONS.map((option) => (
        <ToggleRow
          key={option.key}
          label={option.label}
          accent={option.accent}
          checked={settings.cards[option.key]}
          onChange={(value) => setCard(option.key, value)}
        />
      ))}

      <div className="settings-section">提醒与隐私</div>
      <ToggleRow
        label="桌面告警"
        accent="var(--accent-warn)"
        checked={backendSettings.alertsEnabled}
        onChange={onSetAlertsEnabled}
      />
      <ToggleRow
        label="暂停剪贴板记录"
        accent="var(--accent-disk)"
        checked={backendSettings.clipboardPaused}
        onChange={onSetClipboardPaused}
      />
      <ToggleRow
        label="剪贴板快捷键"
        accent="var(--accent-net)"
        checked={backendSettings.clipboardHotkeyEnabled}
        onChange={onSetClipboardHotkeyEnabled}
      />
      <div className="settings-note">⌘/Ctrl + Shift + V 打开剪贴板</div>

      <div className="settings-foot">
        <span>更改即时生效</span>
        <button
          type="button"
          className="settings-reset"
          onClick={() => onChange(defaultSettings())}
        >
          恢复默认
        </button>
      </div>
    </div>
  );
}

interface ToggleRowProps {
  label: string;
  checked: boolean;
  accent?: string;
  onChange: (value: boolean) => void;
}

function ToggleRow({ label, checked, accent, onChange }: ToggleRowProps) {
  return (
    <label className="settings-row">
      <input
        type="checkbox"
        checked={checked}
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
