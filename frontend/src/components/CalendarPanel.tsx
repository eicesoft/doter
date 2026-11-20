import { useEffect, useMemo, useRef, useState } from "react";
import type { CSSProperties, FormEvent } from "react";

import { useCalendar } from "../hooks/useCalendar";
import {
  WEEKDAY_LABELS,
  addDays,
  addMonths,
  dateFromKey,
  dateKey,
  dayAt,
  eventOrder,
  eventTimeLabel,
  formatDayShort,
  formatDayTitle,
  formatMonthTitle,
  formatWeekTitle,
  minutesOf,
  monthDays,
  normalizeTime,
  sameDay,
  sameDayInMonth,
  weekDays,
  type CalendarEvent,
  type CalendarView,
  type EventDraft,
} from "../lib/calendar";
import { Card } from "./Card";

const HOUR_HEIGHT = 26;

const VIEW_OPTIONS: { id: CalendarView; label: string }[] = [
  { id: "month", label: "月" },
  { id: "week", label: "周" },
  { id: "day", label: "日" },
];

const strokeProps = {
  fill: "none",
  stroke: "currentColor",
  strokeWidth: 1.6,
  strokeLinecap: "round" as const,
  strokeLinejoin: "round" as const,
  "aria-hidden": true,
};

/** The local calendar tab: month / week / day views with event authoring. */
export function CalendarPanel() {
  const { events, addEvent, updateEvent, removeEvent } = useCalendar();
  const today = useMemo(() => dayAt(new Date()), []);

  const [view, setView] = useState<CalendarView>("month");
  const [cursor, setCursor] = useState(today);
  const [selected, setSelected] = useState(today);
  const [formOpen, setFormOpen] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [error, setError] = useState("");
  const [draft, setDraft] = useState<EventDraft>({
    title: "",
    date: dateKey(today),
    start: "",
    end: "",
  });

  const byDay = useMemo(() => {
    const map = new Map<string, CalendarEvent[]>();
    for (const event of events) {
      const list = map.get(event.date) ?? [];
      list.push(event);
      map.set(event.date, list);
    }
    for (const list of map.values()) list.sort(eventOrder);
    return map;
  }, [events]);

  const selectedEvents = byDay.get(dateKey(selected)) ?? [];
  const selectedWeekday = WEEKDAY_LABELS[(selected.getDay() + 6) % 7];

  const selectedDay = (day: Date) => {
    setSelected(day);
    setCursor(day);
  };

  const shift = (direction: number) => {
    if (view === "month") {
      const next = addMonths(cursor, direction);
      setCursor(next);
      setSelected(sameDayInMonth(selected, next));
      return;
    }
    if (view === "week") {
      const next = addDays(cursor, direction * 7);
      setCursor(next);
      setSelected(addDays(selected, direction * 7));
      return;
    }
    const next = addDays(cursor, direction);
    setCursor(next);
    setSelected(next);
  };

  const goToday = () => {
    setCursor(today);
    setSelected(today);
  };

  const openForm = () => {
    setDraft({ title: "", date: dateKey(selected), start: "", end: "" });
    setEditingId(null);
    setError("");
    setFormOpen(true);
  };

  const openEdit = (event: CalendarEvent) => {
    setDraft({
      title: event.title,
      date: event.date,
      start: event.start,
      end: event.end,
    });
    setEditingId(event.id);
    setError("");
    setFormOpen(true);
    const day = dateFromKey(event.date);
    setSelected(day);
    setCursor(day);
  };

  const closeForm = () => {
    setFormOpen(false);
    setEditingId(null);
    setError("");
  };

  const deleteEvent = (id: string) => {
    removeEvent(id);
    if (editingId === id) closeForm();
  };

  const submit = (formEvent: FormEvent<HTMLFormElement>) => {
    formEvent.preventDefault();
    const title = draft.title.trim();
    if (!title) {
      setError("请输入日程标题");
      return;
    }
    const start = normalizeTime(draft.start);
    const end = normalizeTime(draft.end);
    if (draft.start.trim() && !start) {
      setError("开始时间格式不正确，请填写 09:30");
      return;
    }
    if (draft.end.trim() && !end) {
      setError("结束时间格式不正确，请填写 10:30");
      return;
    }
    if (end && !start) {
      setError("填写结束时间前请先填写开始时间");
      return;
    }
    if (start && end && minutesOf(end) <= minutesOf(start)) {
      setError("结束时间需晚于开始时间");
      return;
    }
    if (!/^\d{4}-\d{2}-\d{2}$/.test(draft.date)) {
      setError("请选择日期");
      return;
    }

    if (editingId) {
      updateEvent(editingId, { title, date: draft.date, start, end });
    } else {
      addEvent({ title, date: draft.date, start, end });
    }
    const day = dateFromKey(draft.date);
    setSelected(day);
    setCursor(day);
    closeForm();
  };

  const title =
    view === "month"
      ? formatMonthTitle(cursor)
      : view === "week"
        ? formatWeekTitle(cursor)
        : formatDayTitle(cursor);

  return (
    <Card label="Calendar" hint={`${events.length} 个日程 · 本地保存`} accent="var(--accent-mem)" wide>
      <div className="calendar-panel">
        <div className="cal-toolbar">
          <div className="cal-nav">
            <button
              type="button"
              className="cal-nav-btn"
              aria-label="上一页"
              onClick={() => shift(-1)}
            >
              <svg viewBox="0 0 24 24" {...strokeProps}>
                <path d="m14.5 6-6 6 6 6" />
              </svg>
            </button>
            <button type="button" className="cal-today" onClick={goToday}>
              今天
            </button>
            <button
              type="button"
              className="cal-nav-btn"
              aria-label="下一页"
              onClick={() => shift(1)}
            >
              <svg viewBox="0 0 24 24" {...strokeProps}>
                <path d="m9.5 6 6 6-6 6" />
              </svg>
            </button>
            <span className="cal-title">{title}</span>
          </div>
          <div className="cal-views" role="group" aria-label="日历视图">
            {VIEW_OPTIONS.map((option) => (
              <button
                key={option.id}
                type="button"
                className={view === option.id ? "cal-view is-active" : "cal-view"}
                aria-pressed={view === option.id}
                onClick={() => {
                  setView(option.id);
                  setCursor(selected);
                }}
              >
                {option.label}
              </button>
            ))}
          </div>
        </div>

        <div className="cal-content">
          {view === "month" ? (
            <MonthView
              cursor={cursor}
              selected={selected}
              today={today}
              byDay={byDay}
              onSelectDay={selectedDay}
            />
          ) : (
            <TimeGrid
              key={view}
              days={view === "week" ? weekDays(cursor) : [cursor]}
              byDay={byDay}
              selected={selected}
              today={today}
              onSelectDay={selectedDay}
              onEditEvent={openEdit}
            />
          )}

          <aside className="cal-rail">
            <div className="cal-rail-head">
              <b>{formatDayShort(selected)}</b>
              <span>{selectedWeekday}</span>
              {selectedEvents.length > 0 && <em>{selectedEvents.length} 项</em>}
            </div>

            <div className="cal-rail-list">
              {selectedEvents.length === 0 ? (
                <div className="cal-rail-empty">当天没有日程</div>
              ) : (
                selectedEvents.map((event) => (
                  <div
                    className="cal-event"
                    key={event.id}
                    data-editing={editingId === event.id}
                  >
                    <button
                      type="button"
                      className="cal-event-main"
                      aria-label={`编辑 ${event.title}`}
                      onClick={() => openEdit(event)}
                    >
                      <span className="cal-event-time">{eventTimeLabel(event)}</span>
                      <span className="cal-event-title" title={event.title}>
                        {event.title}
                      </span>
                    </button>
                    <button
                      type="button"
                      className="cal-event-del"
                      aria-label={`删除 ${event.title}`}
                      onClick={() => deleteEvent(event.id)}
                    >
                      <svg viewBox="0 0 24 24" {...strokeProps}>
                        <path d="m7 7 10 10M17 7 7 17" />
                      </svg>
                    </button>
                  </div>
                ))
              )}
            </div>

            {formOpen ? (
              <form className="cal-form" onSubmit={submit} noValidate>
                <input
                  className="cal-input"
                  type="text"
                  placeholder="日程标题"
                  aria-label="日程标题"
                  maxLength={80}
                  autoFocus
                  value={draft.title}
                  onChange={(inputEvent) => {
                    setError("");
                    setDraft((previous) => ({ ...previous, title: inputEvent.target.value }));
                  }}
                />
                <input
                  className="cal-input"
                  type="date"
                  aria-label="日期"
                  required
                  value={draft.date}
                  onChange={(inputEvent) =>
                    setDraft((previous) => ({ ...previous, date: inputEvent.target.value }))
                  }
                />
                <div className="cal-form-times">
                  <input
                    className="cal-input"
                    type="text"
                    inputMode="numeric"
                    placeholder="09:30"
                    aria-label="开始时间"
                    maxLength={5}
                    spellCheck={false}
                    value={draft.start}
                    onChange={(inputEvent) => {
                      setError("");
                      setDraft((previous) => ({ ...previous, start: inputEvent.target.value }));
                    }}
                  />
                  <span>–</span>
                  <input
                    className="cal-input"
                    type="text"
                    inputMode="numeric"
                    placeholder="10:30"
                    aria-label="结束时间"
                    maxLength={5}
                    spellCheck={false}
                    value={draft.end}
                    onChange={(inputEvent) => {
                      setError("");
                      setDraft((previous) => ({ ...previous, end: inputEvent.target.value }));
                    }}
                  />
                </div>
                {error && <span className="cal-form-error">{error}</span>}
                <div className="cal-form-actions">
                  <button type="submit" className="cal-form-submit">
                    {editingId ? "保存" : "添加"}
                  </button>
                  <button type="button" className="cal-form-cancel" onClick={closeForm}>
                    取消
                  </button>
                </div>
              </form>
            ) : (
              <button type="button" className="cal-add" onClick={openForm}>
                <svg viewBox="0 0 24 24" {...strokeProps}>
                  <path d="M12 5v14M5 12h14" />
                </svg>
                新建日程
              </button>
            )}
          </aside>
        </div>
      </div>
    </Card>
  );
}

interface MonthViewProps {
  cursor: Date;
  selected: Date;
  today: Date;
  byDay: Map<string, CalendarEvent[]>;
  onSelectDay: (day: Date) => void;
}

function MonthView({ cursor, selected, today, byDay, onSelectDay }: MonthViewProps) {
  const days = useMemo(() => monthDays(cursor), [cursor]);

  return (
    <div className="cal-month">
      <div className="cal-weekdays">
        {WEEKDAY_LABELS.map((label) => (
          <span key={label}>{label}</span>
        ))}
      </div>
      <div className="cal-month-grid">
        {days.map((day) => {
          const key = dateKey(day);
          const items = byDay.get(key) ?? [];
          const outside = day.getMonth() !== cursor.getMonth();
          const isToday = sameDay(day, today);
          const isSelected = sameDay(day, selected);

          return (
            <button
              key={key}
              type="button"
              className="cal-day"
              data-outside={outside}
              data-today={isToday}
              data-selected={isSelected}
              aria-label={`${formatDayShort(day)}，${items.length} 个日程`}
              aria-pressed={isSelected}
              onClick={() => onSelectDay(day)}
            >
              <span className="cal-daynum">{day.getDate()}</span>
              <span className="cal-day-chips">
                {items.slice(0, 2).map((event) => (
                  <span
                    className="cal-chip"
                    key={event.id}
                    title={`${event.title} · ${eventTimeLabel(event)}`}
                  >
                    {event.title}
                  </span>
                ))}
                {items.length > 2 && <span className="cal-chip cal-chip-more">+{items.length - 2}</span>}
              </span>
            </button>
          );
        })}
      </div>
    </div>
  );
}

interface TimeGridProps {
  days: Date[];
  byDay: Map<string, CalendarEvent[]>;
  selected: Date;
  today: Date;
  onSelectDay: (day: Date) => void;
  onEditEvent: (event: CalendarEvent) => void;
}

const HOURS = Array.from({ length: 24 }, (_, index) => index);

/** Shared time-grid for the week and day views. */
function TimeGrid({ days, byDay, selected, today, onSelectDay, onEditEvent }: TimeGridProps) {
  const scrollRef = useRef<HTMLDivElement>(null);

  // Week / day views open around the working day instead of midnight.
  useEffect(() => {
    if (scrollRef.current) scrollRef.current.scrollTop = 7 * HOUR_HEIGHT;
  }, []);

  return (
    <div className="cal-time" style={{ "--cal-cols": days.length } as CSSProperties}>
      <div className="cal-time-head">
        <span className="cal-time-corner" />
        {days.map((day) => (
          <button
            key={dateKey(day)}
            type="button"
            className="cal-day-head"
            data-selected={sameDay(day, selected)}
            data-today={sameDay(day, today)}
            aria-pressed={sameDay(day, selected)}
            onClick={() => onSelectDay(day)}
          >
            <span>{WEEKDAY_LABELS[(day.getDay() + 6) % 7]}</span>
            <b>{day.getDate()}</b>
          </button>
        ))}
      </div>

      <div className="cal-time-allday">
        <span className="cal-time-allday-label">全天</span>
        {days.map((day) => {
          const allDay = (byDay.get(dateKey(day)) ?? []).filter((event) => !event.start);
          return (
            <div
              className="cal-time-allday-cell"
              key={dateKey(day)}
              onClick={() => onSelectDay(day)}
            >
              {allDay.slice(0, 2).map((event) => (
                <button
                  type="button"
                  className="cal-allday-chip"
                  key={event.id}
                  title={event.title}
                  aria-label={`编辑 ${event.title}`}
                  onClick={(clickEvent) => {
                    clickEvent.stopPropagation();
                    onEditEvent(event);
                  }}
                >
                  {event.title}
                </button>
              ))}
              {allDay.length > 2 && (
                <span className="cal-allday-chip cal-allday-more">+{allDay.length - 2}</span>
              )}
            </div>
          );
        })}
      </div>

      <div className="cal-time-scroll" ref={scrollRef}>
        <div className="cal-time-grid">
          <div className="cal-hour-gutter" style={{ height: 24 * HOUR_HEIGHT }}>
            {HOURS.map((hour) => (
              <span
                className="cal-hour-label"
                key={hour}
                style={{ top: hour * HOUR_HEIGHT + 2 }}
              >
                {`${String(hour).padStart(2, "0")}:00`}
              </span>
            ))}
          </div>
          {days.map((day) => {
            const timed = (byDay.get(dateKey(day)) ?? []).filter((event) => event.start);
            return (
              <div
                className="cal-time-col"
                key={dateKey(day)}
                style={{ height: 24 * HOUR_HEIGHT }}
                onClick={() => onSelectDay(day)}
              >
                {timed.map((event) => {
                  const start = minutesOf(event.start);
                  if (start < 0) return null;
                  const end = event.end ? minutesOf(event.end) : -1;
                  const duration = end > start ? end - start : 45;
                  return (
                    <button
                      type="button"
                      className="cal-block"
                      key={event.id}
                      style={{
                        top: (start / 60) * HOUR_HEIGHT,
                        height: Math.max(20, (duration / 60) * HOUR_HEIGHT - 2),
                      }}
                      title={`${event.title} · ${eventTimeLabel(event)}`}
                      aria-label={`编辑 ${event.title}`}
                      onClick={(clickEvent) => {
                        clickEvent.stopPropagation();
                        onEditEvent(event);
                      }}
                    >
                      <b>{event.title}</b>
                      <span>{eventTimeLabel(event)}</span>
                    </button>
                  );
                })}
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
