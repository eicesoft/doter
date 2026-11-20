/** Local calendar events and the date helpers the calendar tab needs. */

export type CalendarView = "month" | "week" | "day";

export interface CalendarEvent {
  id: string;
  title: string;
  /** Local date key, YYYY-MM-DD. */
  date: string;
  /** HH:MM; both empty means an all-day event. */
  start: string;
  end: string;
}

export interface EventDraft {
  title: string;
  date: string;
  start: string;
  end: string;
}

const STORAGE_KEY = "doter:calendar:v1";

export const WEEKDAY_LABELS = ["周一", "周二", "周三", "周四", "周五", "周六", "周日"] as const;

const pad = (value: number) => String(value).padStart(2, "0");

/** Local date key, immune to UTC offsets. */
export function dateKey(date: Date): string {
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
}

/** Noon avoids DST shifts moving a date across midnight. */
export function dateFromKey(key: string): Date {
  const [year, month, day] = key.split("-").map(Number);
  return new Date(year, (month || 1) - 1, day || 1, 12);
}

export function dayAt(date: Date): Date {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate(), 12);
}

export function addDays(date: Date, days: number): Date {
  const next = dayAt(date);
  next.setDate(next.getDate() + days);
  return next;
}

/** Anchors to the first of the month so short months never overflow. */
export function addMonths(date: Date, months: number): Date {
  return new Date(date.getFullYear(), date.getMonth() + months, 1, 12);
}

/** Monday-first week containing the date. */
export function startOfWeek(date: Date): Date {
  const day = dayAt(date);
  const offset = (day.getDay() + 6) % 7;
  return addDays(day, -offset);
}

export function weekDays(date: Date): Date[] {
  const start = startOfWeek(date);
  return Array.from({ length: 7 }, (_, index) => addDays(start, index));
}

/** Same day-of-month in the anchor's month, clamped to that month's length. */
export function sameDayInMonth(date: Date, anchor: Date): Date {
  const month = anchor.getMonth();
  const lastDay = new Date(anchor.getFullYear(), month + 1, 0, 12).getDate();
  return new Date(anchor.getFullYear(), month, Math.min(date.getDate(), lastDay), 12);
}

/** A fixed six-week month grid, Monday-first. */
export function monthDays(date: Date): Date[] {
  const start = startOfWeek(new Date(date.getFullYear(), date.getMonth(), 1, 12));
  return Array.from({ length: 42 }, (_, index) => addDays(start, index));
}

export function sameDay(a: Date, b: Date): boolean {
  return dateKey(a) === dateKey(b);
}

export function formatMonthTitle(date: Date): string {
  return `${date.getFullYear()}年${date.getMonth() + 1}月`;
}

export function formatWeekTitle(date: Date): string {
  const days = weekDays(date);
  const first = days[0];
  const last = days[6];
  if (first.getMonth() === last.getMonth()) {
    return `${first.getMonth() + 1}月${first.getDate()}日 – ${last.getDate()}日`;
  }
  return `${first.getMonth() + 1}月${first.getDate()}日 – ${last.getMonth() + 1}月${last.getDate()}日`;
}

export function formatDayTitle(date: Date): string {
  const weekday = WEEKDAY_LABELS[(date.getDay() + 6) % 7];
  return `${date.getFullYear()}年${date.getMonth() + 1}月${date.getDate()}日 · ${weekday}`;
}

export function formatDayShort(date: Date): string {
  return `${date.getMonth() + 1}月${date.getDate()}日`;
}

/** Minutes since midnight, or -1 when the time is empty / malformed. */
export function minutesOf(time: string): number {
  const match = /^(\d{1,2}):(\d{2})$/.exec(time);
  if (!match) return -1;
  const hours = Number(match[1]);
  const minutes = Number(match[2]);
  if (hours > 23 || minutes > 59) return -1;
  return hours * 60 + minutes;
}

/**
 * Normalizes "9:30", "09:30" and "0930" to "09:30".
 * Returns "" for empty or malformed input; native time inputs are avoided
 * because WebKit's 12-hour picker rejects plain 24-hour typing.
 */
export function normalizeTime(value: string): string {
  const trimmed = value.trim();
  if (!trimmed) return "";
  const withColon = /^(\d{1,2}):(\d{2})$/.exec(trimmed);
  const compact = /^(\d{1,2})(\d{2})$/.exec(trimmed);
  if (!withColon && !compact) return "";
  const hours = Number((withColon ?? compact)![1]);
  const minutes = Number((withColon ?? compact)![2]);
  if (hours > 23 || minutes > 59) return "";
  return `${pad(hours)}:${pad(minutes)}`;
}

export function eventTimeLabel(event: CalendarEvent): string {
  if (!event.start) return "全天";
  if (event.end) return `${event.start}–${event.end}`;
  return event.start;
}

/** All-day first, then by start time; untimed events tie-break by title. */
export function eventOrder(a: CalendarEvent, b: CalendarEvent): number {
  const aStart = a.start ? minutesOf(a.start) : -1;
  const bStart = b.start ? minutesOf(b.start) : -1;
  if (aStart !== bStart) return aStart - bStart;
  return a.title.localeCompare(b.title);
}

function makeId(): string {
  const cryptoApi = globalThis.crypto;
  if (cryptoApi && "randomUUID" in cryptoApi) {
    return cryptoApi.randomUUID();
  }
  return `event-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

function isEvent(value: unknown): value is CalendarEvent {
  if (!value || typeof value !== "object") return false;
  const event = value as Partial<CalendarEvent>;
  return (
    typeof event.id === "string" &&
    typeof event.title === "string" &&
    typeof event.date === "string" &&
    typeof event.start === "string" &&
    typeof event.end === "string"
  );
}

/** Calendar events read from local storage; malformed entries are dropped. */
export function loadEvents(): CalendarEvent[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter(isEvent).sort(eventOrder);
  } catch {
    return [];
  }
}

/** Best effort: a blocked localStorage must not break the dashboard. */
export function saveEvents(events: CalendarEvent[]): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(events));
  } catch {
    // Presentation-only data: the panel still works for this session.
  }
}

export function createEvent(draft: EventDraft): CalendarEvent {
  return {
    id: makeId(),
    title: draft.title,
    date: draft.date,
    start: draft.start,
    end: draft.end,
  };
}
