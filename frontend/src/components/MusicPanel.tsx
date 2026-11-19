import { useEffect, useMemo, useRef, useState } from "react";
import type {
  CSSProperties,
  KeyboardEvent as ReactKeyboardEvent,
  PointerEvent as ReactPointerEvent,
  ReactNode,
} from "react";

import { RepeatMode } from "../../bindings/doter/internal/music/models";
import { useMusic } from "../hooks/useMusic";

const strokeProps = {
  fill: "none",
  stroke: "currentColor",
  strokeWidth: 1.8,
  strokeLinecap: "round" as const,
  strokeLinejoin: "round" as const,
  "aria-hidden": true,
};

function formatTrackTime(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return "0:00";
  const total = Math.floor(value);
  const hours = Math.floor(total / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const seconds = total % 60;
  const ss = String(seconds).padStart(2, "0");
  if (hours > 0) return `${hours}:${String(minutes).padStart(2, "0")}:${ss}`;
  return `${minutes}:${ss}`;
}

function coverFallback(seed: string): string {
  let hue = 0;
  for (let i = 0; i < seed.length; i += 1) {
    hue = (hue * 31 + seed.charCodeAt(i)) % 360;
  }
  return `linear-gradient(145deg, hsl(${hue} 48% 32%), hsl(${(hue + 52) % 360} 40% 12%))`;
}

/** Picks a readable accent colour from the cover's non-grey pixels. */
function useCoverAccent(artwork: string | null): string | null {
  const [accent, setAccent] = useState<string | null>(null);

  useEffect(() => {
    if (!artwork) {
      setAccent(null);
      return;
    }

    let cancelled = false;
    const image = new Image();

    image.onload = () => {
      if (cancelled) return;
      try {
        const canvas = document.createElement("canvas");
        canvas.width = 24;
        canvas.height = 24;
        const ctx = canvas.getContext("2d");
        if (!ctx) return;
        ctx.drawImage(image, 0, 0, 24, 24);
        setAccent(accentFrom(ctx.getImageData(0, 0, 24, 24).data));
      } catch {
        setAccent(null);
      }
    };
    image.onerror = () => {
      if (!cancelled) setAccent(null);
    };
    image.src = artwork;

    return () => {
      cancelled = true;
    };
  }, [artwork]);

  return accent;
}

function accentFrom(pixels: Uint8ClampedArray): string | null {
  let red = 0;
  let green = 0;
  let blue = 0;
  let count = 0;

  for (let i = 0; i < pixels.length; i += 4) {
    if (pixels[i + 3] < 160) continue;
    const max = Math.max(pixels[i], pixels[i + 1], pixels[i + 2]);
    const min = Math.min(pixels[i], pixels[i + 1], pixels[i + 2]);
    // Skip near-grey pixels: their hue is noise, and the panel already has a
    // cyan fallback for covers without a usable colour.
    if (max - min < 14) continue;
    red += pixels[i];
    green += pixels[i + 1];
    blue += pixels[i + 2];
    count += 1;
  }
  if (count === 0) return null;

  const [hue, saturation, lightness] = rgbToHsl(
    red / count,
    green / count,
    blue / count,
  );
  return `hsl(${Math.round(hue)} ${Math.round(Math.min(80, Math.max(35, saturation)))}% ${Math.round(Math.min(72, Math.max(58, lightness)))}%)`;
}

function rgbToHsl(red: number, green: number, blue: number): [number, number, number] {
  const r = red / 255;
  const g = green / 255;
  const b = blue / 255;
  const max = Math.max(r, g, b);
  const min = Math.min(r, g, b);
  const lightness = (max + min) / 2;

  if (max === min) return [0, 0, lightness * 100];

  const delta = max - min;
  const saturation =
    lightness > 0.5 ? delta / (2 - max - min) : delta / (max + min);
  let hue: number;
  switch (max) {
    case r:
      hue = (g - b) / delta + (g < b ? 6 : 0);
      break;
    case g:
      hue = (b - r) / delta + 2;
      break;
    default:
      hue = (r - g) / delta + 4;
  }
  return [hue * 60, saturation * 100, lightness * 100];
}

function Icon({ children }: { children: ReactNode }) {
  return (
    <svg viewBox="0 0 24 24" {...strokeProps}>
      {children}
    </svg>
  );
}

function PlayIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M8.5 5.6 18 12 8.5 18.4Z" fill="currentColor" />
    </svg>
  );
}

function PauseIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <rect x="8" y="6" width="2.8" height="12" rx="1.2" fill="currentColor" />
      <rect x="13.2" y="6" width="2.8" height="12" rx="1.2" fill="currentColor" />
    </svg>
  );
}

function RepeatIcon({ single }: { single: boolean }) {
  return (
    <Icon>
      <path d="M4 9a4 4 0 0 1 4-4h9l2.5 2.5" />
      <path d="M19 5.5 19 9.5 15 9.5" transform="translate(0 -1)" />
      <path d="M20 15a4 4 0 0 1-4 4H7l-2.5-2.5" />
      <path d="M5 18.5 5 14.5 9 14.5" transform="translate(0 1)" />
      {single && (
        <text
          x="12"
          y="14.6"
          textAnchor="middle"
          fontSize="7.5"
          fontWeight="700"
          fill="currentColor"
          stroke="none"
        >
          1
        </text>
      )}
    </Icon>
  );
}

/** The music tab: the system-wide now-playing session and its transport. */
export function MusicPanel() {
  const {
    now,
    artwork,
    loading,
    playPause,
    next,
    previous,
    seek,
    toggleShuffle,
    cycleRepeat,
  } = useMusic();

  const accent = useCoverAccent(artwork);
  const [dragging, setDragging] = useState(false);
  const [dragPosition, setDragPosition] = useState<number | null>(null);
  const barRef = useRef<HTMLDivElement>(null);
  const [tick, setTick] = useState(0);

  // Re-render twice a second so the interpolated position stays live between
  // backend snapshots.
  useEffect(() => {
    if (!now?.isPlaying) return;
    const id = window.setInterval(() => setTick((tick) => tick + 1), 500);
    return () => window.clearInterval(id);
  }, [now?.isPlaying]);

  const duration = now?.duration ?? 0;
  const elapsed = useMemo(() => {
    if (!now) return 0;
    if (dragPosition !== null) return dragPosition;
    const drift = now.isPlaying ? Math.max(0, (Date.now() - now.updatedAt) / 1000) : 0;
    return Math.min(duration > 0 ? duration : Infinity, now.position + drift);
  }, [now, dragPosition, duration, tick]);

  const ratio = duration > 0 ? Math.min(1, Math.max(0, elapsed / duration)) : 0;
  const seed = `${now?.trackId ?? ""}${now?.title ?? ""}`;
  const cover = artwork
    ? `url("${artwork}")`
    : coverFallback(seed);

  if (!now || !now.available) {
    const failed = Boolean(now?.error);
    return (
      <div
        className="music-panel"
        style={{ "--music-accent": "var(--accent-net)" } as CSSProperties}
      >
        <div className="music-empty">
          <span className="music-empty-icon">
            {failed ? (
              <Icon>
                <path d="M12 4 20 19H4Z" />
                <path d="M12 10v4M12 16.8v.2" />
              </Icon>
            ) : (
              <Icon>
                <path d="M9 18V6l10-2v12" />
                <circle cx="6.5" cy="18" r="2.5" />
                <circle cx="16.5" cy="16" r="2.5" />
              </Icon>
            )}
          </span>
          <b>{failed ? "无法读取系统媒体" : loading ? "正在连接媒体…" : "未在播放"}</b>
          <span className="music-empty-text">
            {failed
              ? now?.error
              : "在任意播放器中开始播放后，这里会自动显示当前曲目。"}
          </span>
        </div>
      </div>
    );
  }

  const canSeek = now.canSeek && duration > 0;

  const positionAt = (clientX: number): number => {
    const element = barRef.current;
    if (!element || duration <= 0) return 0;
    const rect = element.getBoundingClientRect();
    const fraction = Math.min(1, Math.max(0, (clientX - rect.left) / rect.width));
    return fraction * duration;
  };

  const handlePointerDown = (event: ReactPointerEvent<HTMLDivElement>) => {
    if (!canSeek) return;
    event.currentTarget.setPointerCapture(event.pointerId);
    setDragging(true);
    setDragPosition(positionAt(event.clientX));
  };

  const handlePointerMove = (event: ReactPointerEvent<HTMLDivElement>) => {
    if (!dragging || !canSeek) return;
    setDragPosition(positionAt(event.clientX));
  };

  const handlePointerUp = (event: ReactPointerEvent<HTMLDivElement>) => {
    if (!dragging || !canSeek) return;
    const target = positionAt(event.clientX);
    setDragging(false);
    setDragPosition(null);
    seek(target);
  };

  const handlePointerCancel = () => {
    setDragging(false);
    setDragPosition(null);
  };

  const handleKeyDown = (event: ReactKeyboardEvent<HTMLDivElement>) => {
    if (!canSeek) return;
    if (event.key !== "ArrowLeft" && event.key !== "ArrowRight") return;
    event.preventDefault();
    const delta = event.key === "ArrowLeft" ? -5 : 5;
    const target = Math.min(duration, Math.max(0, elapsed + delta));
    seek(target);
  };

  return (
    <div
      className="music-panel"
      style={{ "--music-accent": accent ?? "var(--accent-net)" } as CSSProperties}
    >
      <div className="music-hero">
        <div
          className="music-cover"
          key={now.trackId}
          style={{ backgroundImage: cover }}
        >
          {!artwork && (
            <svg viewBox="0 0 24 24" {...strokeProps}>
              <path d="M9 18V6l10-2v12" />
              <circle cx="6.5" cy="18" r="2.5" />
              <circle cx="16.5" cy="16" r="2.5" />
            </svg>
          )}
        </div>

        <div className="music-meta" key={`${now.trackId}-meta`}>
          <div className="music-eyebrow">
            <span className="music-source">{now.source || "媒体播放器"}</span>
            <span className={now.isPlaying ? "music-state" : "music-state is-paused"}>
              {now.isPlaying ? "正在播放" : "已暂停"}
            </span>
          </div>
          <div className="music-title" title={now.title}>
            {now.title || "未知曲目"}
          </div>
          {now.artist && <div className="music-artist">{now.artist}</div>}
          {now.album && now.album !== now.title && (
            <div className="music-album">{now.album}</div>
          )}
        </div>
      </div>

      {duration > 0 && (
        <div className="music-timeline">
          <div
            ref={barRef}
            className={`music-progress${dragging ? " is-dragging" : ""}${canSeek ? "" : " is-static"}`}
            role="slider"
            tabIndex={canSeek ? 0 : -1}
            aria-label="播放进度"
            aria-valuemin={0}
            aria-valuemax={Math.round(duration)}
            aria-valuenow={Math.round(elapsed)}
            aria-valuetext={`${formatTrackTime(elapsed)} / ${formatTrackTime(duration)}`}
            onPointerDown={handlePointerDown}
            onPointerMove={handlePointerMove}
            onPointerUp={handlePointerUp}
            onPointerCancel={handlePointerCancel}
            onKeyDown={handleKeyDown}
          >
            <span className="music-progress-fill" style={{ width: `${ratio * 100}%` }} />
            <span
              className="music-progress-knob"
              style={{ left: `${ratio * 100}%` }}
            />
          </div>
          <div className="music-times">
            <span>{formatTrackTime(elapsed)}</span>
            <span>{formatTrackTime(duration)}</span>
          </div>
        </div>
      )}

      <div className="music-controls">
        {now.canShuffle && (
          <button
            type="button"
            className={now.shuffle ? "music-ctrl is-on" : "music-ctrl"}
            onClick={toggleShuffle}
            title={now.shuffle ? "关闭随机播放" : "随机播放"}
            aria-label={now.shuffle ? "关闭随机播放" : "随机播放"}
            aria-pressed={now.shuffle}
          >
            <Icon>
              <path d="M3 6h4l10 12h4" />
              <path d="M3 18h4l10-12h4" />
              <path d="M17 4l4 2-4 2" transform="translate(-1 0)" />
              <path d="M17 14l4 2-4 2" transform="translate(-1 0)" />
            </Icon>
          </button>
        )}

        <button
          type="button"
          className="music-ctrl"
          onClick={previous}
          disabled={!now.canPrev}
          title="上一首"
          aria-label="上一首"
        >
          <Icon>
            <path d="M6 6v12" />
            <path d="M18 6.5 7.5 12 18 17.5Z" transform="translate(1 0)" />
          </Icon>
        </button>

        <button
          type="button"
          className="music-play"
          onClick={playPause}
          disabled={!now.canPlayPause}
          title={now.isPlaying ? "暂停" : "播放"}
          aria-label={now.isPlaying ? "暂停" : "播放"}
        >
          {now.isPlaying ? <PauseIcon /> : <PlayIcon />}
        </button>

        <button
          type="button"
          className="music-ctrl"
          onClick={next}
          disabled={!now.canNext}
          title="下一首"
          aria-label="下一首"
        >
          <Icon>
            <path d="M18 6v12" />
            <path d="M6 6.5 16.5 12 6 17.5Z" transform="translate(-1 0)" />
          </Icon>
        </button>

        {now.canRepeat && (
          <button
            type="button"
            className={now.repeat === RepeatMode.RepeatOff ? "music-ctrl" : "music-ctrl is-on"}
            onClick={cycleRepeat}
            title={
              now.repeat === RepeatMode.RepeatOff
                ? "循环：关闭"
                : now.repeat === RepeatMode.RepeatTrack
                  ? "循环：单曲"
                  : "循环：列表"
            }
            aria-label={
              now.repeat === RepeatMode.RepeatOff
                ? "循环：关闭"
                : now.repeat === RepeatMode.RepeatTrack
                  ? "循环：单曲"
                  : "循环：列表"
            }
            aria-pressed={now.repeat !== RepeatMode.RepeatOff}
          >
            <RepeatIcon single={now.repeat === RepeatMode.RepeatTrack} />
          </button>
        )}
      </div>
    </div>
  );
}
