import { useCallback, useEffect, useState } from "react";

import {
  Get,
  SetAlertsEnabled,
  SetAutostart,
  SetClipboardAutoPaste,
  SetClipboardExcludedApps,
  SetClipboardHistoryLimit,
  SetClipboardHotkeyEnabled,
  SetClipboardImageMaxBytes,
  SetClipboardPaused,
  SetClipboardPreviewBytes,
  SetHideOnFullscreen,
} from "../../bindings/doter/internal/services/settingsservice";
import type { Settings } from "../../bindings/doter/internal/prefs/models";

/** Backend defaults, used until the service answers or when it fails. */
function fallback(): Settings {
  return {
    alertsEnabled: true,
    clipboardPaused: false,
    clipboardHotkeyEnabled: true,
    clipboardHistoryLimit: 100,
    clipboardPreviewBytes: 4096,
    clipboardImageMaxBytes: 16 << 20,
    clipboardAutoPaste: false,
    clipboardExcludedApps: [],
    autostart: false,
    hideOnFullscreen: true,
  };
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
  setClipboardHistoryLimit: (limit: number) => void;
  setClipboardPreviewBytes: (bytes: number) => void;
  setClipboardImageMaxBytes: (bytes: number) => void;
  setClipboardAutoPaste: (enabled: boolean) => void;
  setClipboardExcludedApps: (apps: string[]) => void;
  setAutostart: (enabled: boolean) => void;
  setHideOnFullscreen: (enabled: boolean) => void;
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

  const setClipboardHistoryLimit = useCallback((limit: number) => {
    SetClipboardHistoryLimit(limit)
      .then(setSettings)
      .catch(() => {});
  }, []);

  const setClipboardPreviewBytes = useCallback((bytes: number) => {
    SetClipboardPreviewBytes(bytes)
      .then(setSettings)
      .catch(() => {});
  }, []);

  const setClipboardImageMaxBytes = useCallback((bytes: number) => {
    SetClipboardImageMaxBytes(bytes)
      .then(setSettings)
      .catch(() => {});
  }, []);

  const setClipboardAutoPaste = useCallback((enabled: boolean) => {
    SetClipboardAutoPaste(enabled)
      .then(setSettings)
      .catch(() => {});
  }, []);

  const setClipboardExcludedApps = useCallback((apps: string[]) => {
    SetClipboardExcludedApps(apps)
      .then(setSettings)
      .catch(() => {});
  }, []);

  const setAutostart = useCallback((enabled: boolean) => {
    SetAutostart(enabled)
      .then(setSettings)
      .catch(() => {});
  }, []);

  const setHideOnFullscreen = useCallback((enabled: boolean) => {
    SetHideOnFullscreen(enabled)
      .then(setSettings)
      .catch(() => {});
  }, []);

  return {
    settings,
    setAlertsEnabled,
    setClipboardPaused,
    setClipboardHotkeyEnabled,
    setClipboardHistoryLimit,
    setClipboardPreviewBytes,
    setClipboardImageMaxBytes,
    setClipboardAutoPaste,
    setClipboardExcludedApps,
    setAutostart,
    setHideOnFullscreen,
  };
}
