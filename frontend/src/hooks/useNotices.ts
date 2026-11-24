import { useEffect, useState } from "react";
import { Events } from "@wailsio/runtime";

import type { Snapshot } from "../../bindings/doter/internal/notice/models";
import { GetSnapshot } from "../../bindings/doter/internal/services/noticeservice";
import { USE_MOCK_NOTICES, mockNoticesSnapshot } from "../mocks/notices";

/** Live system notifications pushed by the Go monitor. */
export function useNotices(): { snapshot: Snapshot | null; live: boolean } {
  const [snapshot, setSnapshot] = useState<Snapshot | null>(null);
  const [live, setLive] = useState(false);

  useEffect(() => {
    // Mock 模式：跳过后端读取，直接注入假数据测试面板界面。
    if (USE_MOCK_NOTICES) {
      setSnapshot(mockNoticesSnapshot());
      setLive(true);
      return;
    }

    let mounted = true;

    GetSnapshot()
      .then((value) => {
        if (mounted) setSnapshot(value);
      })
      .catch(() => {
        // The window can mount before the backend is ready in dev mode.
      });

    const unsubscribe = Events.On("notices:update", (event) => {
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