import { useEffect, useState } from "react";
import { Events } from "@wailsio/runtime";

import { GetSnapshot } from "../../bindings/doter/internal/services/agentsservice";
import type { Snapshot } from "../../bindings/doter/internal/agents/models";

/** Live Claude/Codex sessions pushed by the Go monitor. */
export function useAgentSessions(): { snapshot: Snapshot | null; live: boolean } {
  const [snapshot, setSnapshot] = useState<Snapshot | null>(null);
  const [live, setLive] = useState(false);

  useEffect(() => {
    let mounted = true;

    GetSnapshot()
      .then((value) => {
        if (mounted) setSnapshot(value);
      })
      .catch(() => {
        // The window can mount before the backend is ready in dev mode.
      });

    const unsubscribe = Events.On("agents:update", (event) => {
      if (!mounted) return;
      setSnapshot(event.data as Snapshot);
      setLive(true);
    });

    return () => {
      mounted = false;
      unsubscribe();
    };
  }, []);

  return { snapshot, live };
}
