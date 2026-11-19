import { useCallback, useEffect, useState } from "react";

import {
  Get,
  SetAlertsEnabled,
  SetClipboardHotkeyEnabled,
  SetClipboardPaused,
} from "../../bindings/doter/internal/services/settingsservice";
import type { Settings } from "../../bindings/doter/internal/prefs/models";

/** Backend defaults, used until the service answers or when it fails. */
function fallback(): Settings {
  return { alertsEnabled: true, clipboardPaused: false, clipboardHotkeyEnabled: true };
}

/**
 * Preferences that the Go side needs before the webview exists: desktop
 * alerts and clipboard capture. UI-only preferences stay in localStorage.
 */
export function useBackendSettings(): {
  settings: Settings;
  setAlertsEnabled: (enabled: boolean) => void;
  setClipboardPaused: (paused: boolean) => void;
  setClipboardHotkeyEnabled: (enabled: boolean) => void;
} {
  const [settings, setSettings] = useState<Settings>(fallback);

  useEffect(() => {
    let mounted = true;
    Get()
      .then((value) => {
        if (mounted) setSettings(value);
      })
      .catch(() => {
        // The window can mount before the backend is ready in dev mode.
      });
    return () => {
      mounted = false;
    };
  }, []);

  const setAlertsEnabled = useCallback((enabled: boolean) => {
    SetAlertsEnabled(enabled)
      .then(setSettings)
      .catch(() => {});
  }, []);

  const setClipboardPaused = useCallback((paused: boolean) => {
    SetClipboardPaused(paused)
      .then(setSettings)
      .catch(() => {});
  }, []);

  const setClipboardHotkeyEnabled = useCallback((enabled: boolean) => {
    SetClipboardHotkeyEnabled(enabled)
      .then(setSettings)
      .catch(() => {});
  }, []);

  return { settings, setAlertsEnabled, setClipboardPaused, setClipboardHotkeyEnabled };
}
