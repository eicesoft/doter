/** Dashboard preferences, persisted locally in the webview. */

export type CardKey =
  | "cpu"
  | "memory"
  | "storage"
  | "network"
  | "power"
  | "system"
  | "processes";

/** Pages of the expanded dashboard; matches the bottom tab bar order. */
export type TabKey = "overview" | "calendar" | "clipboard" | "weather" | "music" | "agents";

/** The two metrics shown in the idle pill. */
export type CollapsedMetric =
  | "cpu"
  | "memory"
  | "network"
  | "battery"
  | "agents"
  | "clock"
  | "weather";

export interface Settings {
  /** Expand the island when the pointer enters it. */
  hoverExpand: boolean;
  /** Show seconds in the header clock. */
  clockSeconds: boolean;
  /** Metric cards shown on the overview tab. */
  cards: Record<CardKey, boolean>;
  /** Left and right pill metrics. */
  collapsed: [CollapsedMetric, CollapsedMetric];
  /** Pages shown in the bottom tab bar. Agents is always on. */
  tabs: Record<TabKey, boolean>;
}

/** Tab rows in tab-bar order; `locked` tabs cannot be hidden. */
export const TAB_OPTIONS: readonly {
  key: TabKey;
  label: string;
  accent: string;
  locked?: boolean;
}[] = [
  { key: "overview", label: "系统总览", accent: "var(--accent-cpu)", locked: true },
  { key: "calendar", label: "日历", accent: "var(--accent-sys)" },
  { key: "clipboard", label: "剪切板", accent: "var(--accent-disk)" },
  { key: "weather", label: "天气", accent: "var(--accent-net)" },
  { key: "music", label: "音乐", accent: "var(--accent-net)" },
  { key: "agents", label: "Agent 监控", accent: "var(--accent-cpu)" },
];

const TAB_KEYS: readonly TabKey[] = TAB_OPTIONS.map((option) => option.key);

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

export const COLLAPSED_OPTIONS: readonly { key: CollapsedMetric; label: string }[] = [
  { key: "cpu", label: "处理器" },
  { key: "memory", label: "内存" },
  { key: "network", label: "网络" },
  { key: "battery", label: "电池" },
  { key: "agents", label: "Agent" },
  { key: "clock", label: "时钟" },
  { key: "weather", label: "天气" },
];

const COLLAPSED_KEYS = new Set<CollapsedMetric>(COLLAPSED_OPTIONS.map((option) => option.key));

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
  collapsed: ["cpu", "weather"],
  tabs: {
    overview: true,
    calendar: true,
    clipboard: true,
    weather: true,
    music: true,
    agents: true,
  },
};

const STORAGE_KEY = "doter:settings:v1";

/** defaultSettings returns a copy, so callers never mutate the constant. */
export function defaultSettings(): Settings {
  return {
    ...DEFAULT_SETTINGS,
    cards: { ...DEFAULT_SETTINGS.cards },
    collapsed: [...DEFAULT_SETTINGS.collapsed],
    tabs: { ...DEFAULT_SETTINGS.tabs },
  };
}

function loadCollapsed(value: unknown, fallback: [CollapsedMetric, CollapsedMetric]): [CollapsedMetric, CollapsedMetric] {
  if (!Array.isArray(value) || value.length !== 2) return [...fallback];
  const [left, right] = value;
  if (!COLLAPSED_KEYS.has(left) || !COLLAPSED_KEYS.has(right)) return [...fallback];
  return [left, right];
}

/** loadSettings reads the saved preferences, falling back field by field. */
export function loadSettings(): Settings {
  const fallback = defaultSettings();
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return fallback;
    const saved = JSON.parse(raw) as Partial<Settings> & {
      cards?: Partial<Record<CardKey, boolean>>;
      tabs?: Partial<Record<TabKey, boolean>>;
    };
    const collapsed = "collapsed" in saved ? saved.collapsed : undefined;
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
      collapsed: loadCollapsed(collapsed, fallback.collapsed),
      tabs: TAB_KEYS.reduce(
        (tabs, key) => {
          const value = saved.tabs?.[key];
          // The overview tab is the dashboard's home and stays available even
          // if an old or hand-edited file turned it off.
          tabs[key] = key === "overview" ? true : typeof value === "boolean" ? value : fallback.tabs[key];
          return tabs;
        },
        { ...fallback.tabs },
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
