/** Dashboard preferences, persisted locally in the webview. */

export type CardKey =
  | "cpu"
  | "memory"
  | "storage"
  | "network"
  | "power"
  | "system"
  | "processes";

export interface Settings {
  /** Expand the island when the pointer enters it. */
  hoverExpand: boolean;
  /** Show seconds in the header clock. */
  clockSeconds: boolean;
  /** Metric cards shown on the overview tab. */
  cards: Record<CardKey, boolean>;
}

export const CARD_KEYS: readonly CardKey[] = [
  "cpu",
  "memory",
  "storage",
  "network",
  "power",
  "system",
  "processes",
];

/** Settings rows in display order, with the accent the card uses. */
export const CARD_OPTIONS: readonly { key: CardKey; label: string; accent: string }[] = [
  { key: "cpu", label: "处理器", accent: "var(--accent-cpu)" },
  { key: "memory", label: "内存", accent: "var(--accent-mem)" },
  { key: "storage", label: "存储", accent: "var(--accent-disk)" },
  { key: "network", label: "网络", accent: "var(--accent-net)" },
  { key: "power", label: "电源", accent: "var(--accent-battery)" },
  { key: "system", label: "系统", accent: "var(--accent-sys)" },
  { key: "processes", label: "进程榜", accent: "var(--accent-cpu)" },
];

export const DEFAULT_SETTINGS: Settings = {
  hoverExpand: true,
  clockSeconds: true,
  cards: {
    cpu: true,
    memory: true,
    storage: true,
    network: true,
    power: true,
    system: true,
    processes: true,
  },
};

const STORAGE_KEY = "doter:settings:v1";

/** defaultSettings returns a copy, so callers never mutate the constant. */
export function defaultSettings(): Settings {
  return { ...DEFAULT_SETTINGS, cards: { ...DEFAULT_SETTINGS.cards } };
}

/** loadSettings reads the saved preferences, falling back field by field. */
export function loadSettings(): Settings {
  const fallback = defaultSettings();
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return fallback;
    const saved = JSON.parse(raw) as Partial<Settings> & {
      cards?: Partial<Record<CardKey, boolean>>;
    };
    return {
      hoverExpand:
        typeof saved.hoverExpand === "boolean" ? saved.hoverExpand : fallback.hoverExpand,
      clockSeconds:
        typeof saved.clockSeconds === "boolean" ? saved.clockSeconds : fallback.clockSeconds,
      cards: CARD_KEYS.reduce(
        (cards, key) => {
          const value = saved.cards?.[key];
          cards[key] = typeof value === "boolean" ? value : fallback.cards[key];
          return cards;
        },
        { ...fallback.cards },
      ),
    };
  } catch {
    return fallback;
  }
}

/** saveSettings persists preferences; a blocked localStorage is not fatal. */
export function saveSettings(settings: Settings): void {
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(settings));
  } catch {
    // Private browsing or a locked-down webview: keep the in-memory values.
  }
}
