import { useCallback, useEffect, useState } from "react";
import { Events } from "@wailsio/runtime";

import {
  Clear,
  Delete,
  GetHistory,
  Paste,
  Pin,
  WriteBack,
} from "../../bindings/doter/internal/services/clipboardservice";
import type { Entry } from "../../bindings/doter/internal/clipboard/models";

/** The clipboard history, kept in sync with the backend watcher. */
export function useClipboard(): {
  entries: Entry[];
  loading: boolean;
  search: string;
  setSearch: (value: string) => void;
  togglePin: (id: string) => void;
  remove: (id: string) => void;
  clear: () => void;
  copy: (id: string) => Promise<boolean>;
  paste: () => Promise<boolean>;
} {
  const [entries, setEntries] = useState<Entry[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");

  useEffect(() => {
    let mounted = true;

    const load = () => {
      GetHistory()
        .then((list) => {
          if (!mounted) return;
          setEntries(list ?? []);
        })
        .catch(() => {
          // The window can mount before the backend is ready in dev mode.
        })
        .finally(() => {
          if (mounted) setLoading(false);
        });
    };

    load();

    // The backend pushes one entry per pasteboard change.
    const unsubscribe = Events.On("clipboard:update", (event) => {
      if (!mounted) return;
      const entry = event.data as Entry;
      setEntries((current) => {
        const rest = current.filter((e) => e.id !== entry.id);
        const pinned = rest.filter((e) => e.pinned);
        const others = rest.filter((e) => !e.pinned);
        if (entry.pinned) pinned.unshift(entry);
        else others.unshift(entry);
        return [...pinned, ...others];
      });
    });

    // Limits and preview size changed in settings: reload from the backend.
    const unsubscribeConfig = Events.On("clipboard:config", () => {
      if (mounted) load();
    });

    return () => {
      mounted = false;
      unsubscribe();
      unsubscribeConfig();
    };
  }, []);

  const togglePin = useCallback((id: string) => {
    Pin(id)
      .then((ok) => {
        if (!ok) return;
        setEntries((current) =>
          current.map((e) => (e.id === id ? { ...e, pinned: !e.pinned } : e)),
        );
      })
      .catch(() => {});
  }, []);

  const remove = useCallback((id: string) => {
    Delete(id)
      .then((ok) => {
        if (!ok) return;
        setEntries((current) => current.filter((e) => e.id !== id));
      })
      .catch(() => {});
  }, []);

  const clear = useCallback(() => {
    Clear()
      .then(() => setEntries([]))
      .catch(() => {});
  }, []);

  const copy = useCallback((id: string) => WriteBack(id).catch(() => false), []);
  const paste = useCallback(() => Paste().then(() => true).catch(() => false), []);

  return { entries, loading, search, setSearch, togglePin, remove, clear, copy, paste };
}
