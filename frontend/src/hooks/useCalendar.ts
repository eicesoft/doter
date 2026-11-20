import { useCallback, useEffect, useState } from "react";

import {
  createEvent,
  loadEvents,
  saveEvents,
  type CalendarEvent,
  type EventDraft,
} from "../lib/calendar";

/** Calendar events, persisted locally in the webview. */
export function useCalendar(): {
  events: CalendarEvent[];
  addEvent: (draft: EventDraft) => CalendarEvent;
  updateEvent: (id: string, draft: EventDraft) => void;
  removeEvent: (id: string) => void;
} {
  const [events, setEvents] = useState<CalendarEvent[]>(loadEvents);

  useEffect(() => {
    saveEvents(events);
  }, [events]);

  const addEvent = useCallback((draft: EventDraft) => {
    const event = createEvent(draft);
    setEvents((previous) => [...previous, event]);
    return event;
  }, []);

  const updateEvent = useCallback((id: string, draft: EventDraft) => {
    setEvents((previous) =>
      previous.map((event) =>
        event.id === id
          ? {
              ...event,
              title: draft.title,
              date: draft.date,
              start: draft.start,
              end: draft.end,
            }
          : event,
      ),
    );
  }, []);

  const removeEvent = useCallback((id: string) => {
    setEvents((previous) => previous.filter((event) => event.id !== id));
  }, []);

  return { events, addEvent, updateEvent, removeEvent };
}
