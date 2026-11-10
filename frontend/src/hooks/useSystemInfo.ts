import { useEffect, useState } from "react";
import { Events } from "@wailsio/runtime";

import { GetSnapshot, GetStaticInfo } from "../../bindings/doter/internal/services/systemservice";
import type { Snapshot, StaticInfo } from "../../bindings/doter/internal/sysinfo/models";

/** Live system metrics pushed by the Go sampler. */
export function useSystemInfo(): {
  snapshot: Snapshot | null;
  staticInfo: StaticInfo | null;
  live: boolean;
} {
  const [snapshot, setSnapshot] = useState<Snapshot | null>(null);
  const [staticInfo, setStaticInfo] = useState<StaticInfo | null>(null);
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

    GetStaticInfo()
      .then((value) => {
        if (mounted) setStaticInfo(value);
      })
      .catch(() => {});

    const unsubscribe = Events.On("sysinfo:update", (event) => {
      if (!mounted) return;
      setSnapshot(event.data);
      setLive(true);
    });

    return () => {
      mounted = false;
      unsubscribe();
    };
  }, []);

  return { snapshot, staticInfo, live };
}