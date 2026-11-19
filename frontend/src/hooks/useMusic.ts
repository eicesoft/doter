import { useCallback, useEffect, useRef, useState } from "react";
import { Events } from "@wailsio/runtime";

import type { NowPlaying } from "../../bindings/doter/internal/music/models";
import {
  CycleRepeat,
  GetArtwork,
  GetNowPlaying,
  Next,
  PlayPause,
  Previous,
  Seek,
  Subscribe,
  ToggleShuffle,
  Unsubscribe,
} from "../../bindings/doter/internal/services/musicservice";

const EVENT_UPDATE = "music:update";
const EVENT_ARTWORK = "music:artwork";

/** How long to wait before calling an empty session "nothing playing". */
const START_GRACE_MS = 700;

interface ArtworkPayload {
  trackId: string;
  key: string;
  data: string;
}

/** Live now-playing state, kept in sync with the backend media adapter. */
export function useMusic() {
  const [now, setNow] = useState<NowPlaying | null>(null);
  const [artwork, setArtwork] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const nowRef = useRef<NowPlaying | null>(null);
  const artTrack = useRef("");
  const artCache = useRef(new Map<string, string>());

  useEffect(() => {
    let mounted = true;

    const apply = (next: NowPlaying) => {
      nowRef.current = next;
      setNow(next);
      if (next.available || next.error) setLoading(false);
    };

    const stopUpdate = Events.On(EVENT_UPDATE, (event) => {
      if (!mounted) return;
      apply(event.data as NowPlaying);
    });
    const stopArtwork = Events.On(EVENT_ARTWORK, (event) => {
      if (!mounted) return;
      const next = event.data as ArtworkPayload;
      if (nowRef.current?.trackId !== next.trackId || !next.data) return;
      artCache.current.set(next.trackId, next.data);
      setArtwork(next.data);
    });

    // The adapter needs a moment to attach; only show the empty state after
    // that grace period so the panel does not flash "nothing playing".
    const grace = window.setTimeout(() => {
      if (mounted) setLoading(false);
    }, START_GRACE_MS);

    Subscribe().catch(() => {});
    GetNowPlaying()
      .then((snapshot) => {
        if (!mounted) return;
        if (snapshot.available || snapshot.error) apply(snapshot);
      })
      .catch(() => {});

    return () => {
      mounted = false;
      window.clearTimeout(grace);
      stopUpdate();
      stopArtwork();
      Unsubscribe().catch(() => {});
    };
  }, []);

  const trackId = now?.trackId ?? "";

  useEffect(() => {
    artTrack.current = trackId;
    if (!trackId) {
      setArtwork(null);
      return;
    }

    const cached = artCache.current.get(trackId);
    if (cached) {
      setArtwork(cached);
      return;
    }

    setArtwork(null);
    GetArtwork(trackId)
      .then((data) => {
        if (artTrack.current !== trackId || !data) return;
        artCache.current.set(trackId, data);
        setArtwork(data);
      })
      .catch(() => {});
  }, [trackId]);

  const playPause = useCallback(() => {
    PlayPause().catch(() => {});
  }, []);
  const next = useCallback(() => {
    Next().catch(() => {});
  }, []);
  const previous = useCallback(() => {
    Previous().catch(() => {});
  }, []);
  const seek = useCallback((seconds: number) => {
    Seek(seconds).catch(() => {});
  }, []);
  const toggleShuffle = useCallback(() => {
    ToggleShuffle().catch(() => {});
  }, []);
  const cycleRepeat = useCallback(() => {
    CycleRepeat().catch(() => {});
  }, []);

  return {
    now,
    artwork,
    loading,
    playPause,
    next,
    previous,
    seek,
    toggleShuffle,
    cycleRepeat,
  };
}
