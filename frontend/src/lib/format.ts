const BYTE_UNITS = ["B", "KB", "MB", "GB", "TB", "PB"];

/** Human readable byte count using binary units. */
export function formatBytes(bytes: number, digits = 1): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return "0 B";
  const exponent = Math.min(
    Math.floor(Math.log(bytes) / Math.log(1024)),
    BYTE_UNITS.length - 1,
  );
  const value = bytes / 1024 ** exponent;
  const places = exponent === 0 ? 0 : value >= 100 ? 0 : digits;
  return `${value.toFixed(places)} ${BYTE_UNITS[exponent]}`;
}

export function formatSpeed(bytesPerSecond: number): string {
  return `${formatBytes(bytesPerSecond)}/s`;
}

/** Ultra-compact rate used inside the collapsed pill: 1.2M, 240K. */
export function formatRateCompact(bytesPerSecond: number): string {
  if (!Number.isFinite(bytesPerSecond) || bytesPerSecond <= 0) return "0";
  const kb = 1024;
  const mb = kb * 1024;
  const gb = mb * 1024;
  if (bytesPerSecond < kb) return `${Math.round(bytesPerSecond)}B`;
  if (bytesPerSecond < mb) {
    const value = bytesPerSecond / kb;
    return `${value >= 100 ? value.toFixed(0) : value.toFixed(1)}K`;
  }
  if (bytesPerSecond < gb) {
    const value = bytesPerSecond / mb;
    return `${value >= 100 ? value.toFixed(0) : value.toFixed(1)}M`;
  }
  return `${(bytesPerSecond / gb).toFixed(1)}G`;
}

export function formatPercent(value: number, digits = 0): string {
  if (!Number.isFinite(value)) return "0%";
  return `${value.toFixed(digits)}%`;
}

export function clampPercent(value: number): number {
  if (!Number.isFinite(value)) return 0;
  return Math.min(100, Math.max(0, value));
}

/** Uptime style: 3d 4h, 4h 12m, 12m. */
export function formatUptime(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds <= 0) return "—";
  const days = Math.floor(seconds / 86_400);
  const hours = Math.floor((seconds % 86_400) / 3_600);
  const minutes = Math.floor((seconds % 3_600) / 60);
  if (days > 0) return `${days}d ${hours}h`;
  if (hours > 0) return `${hours}h ${minutes}m`;
  return `${minutes}m`;
}

/** Battery time remaining style: 2h 14m. */
export function formatDuration(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) return "—";
  const hours = Math.floor(seconds / 3_600);
  const minutes = Math.floor((seconds % 3_600) / 60);
  if (hours > 0) return `${hours}h ${minutes}m`;
  return `${minutes}m`;
}

export function formatClock(date: Date, withSeconds = false): string {
  return date.toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
    ...(withSeconds ? { second: "2-digit" } : {}),
    hour12: false,
  });
}

export function formatDate(date: Date): string {
  return date.toLocaleDateString([], {
    weekday: "short",
    month: "short",
    day: "numeric",
  });
}

/** Short, friendly platform label for the panel header. */
export function platformLabel(
  os: string,
  platform: string,
  version: string,
): string {
  const name = platform || os || "System";
  const pretty =
    name === "darwin" ? "macOS" : name === "windows" ? "Windows" : name === "linux" ? "Linux" : name;
  return version ? `${pretty} ${version}` : pretty;
}

/** Trim the marketing noise out of long CPU model strings. */
export function shortCPUModel(model: string): string {
  if (!model) return "CPU";
  return model
    .replace(/\s*\(R\)|\s*\(TM\)|\s*CPU\s*@.*$/gi, "")
    .replace(/\s+/g, " ")
    .trim();
}