import { useEffect, useLayoutEffect, useRef, useState, type PointerEvent as ReactPointerEvent } from "react";

import type { Day, Forecast, Location } from "../../bindings/doter/internal/weather/models";
import type { WeatherController } from "../hooks/useWeather";
import { WeatherIcon } from "./WeatherIcon";

interface WeatherPanelProps {
  weather: WeatherController;
  onFocusIsland: () => void;
}

/** Pixels the pointer must travel before a drag counts as a city swipe. */
const SWIPE_TRIGGER = 56;
/** How long the strip takes to reach its page after release. */
const GLIDE_MS = 420;

function wrapIndex(index: number, count: number): number {
  return ((index % count) + count) % count;
}

function weekday(date: string, index: number): string {
  if (index === 0) return "今天";
  if (index === 1) return "明天";
  const d = new Date(`${date}T12:00:00`);
  return ["周日", "周一", "周二", "周三", "周四", "周五", "周六"][d.getDay()] ?? date;
}

function degree(value: number): string {
  return `${Math.round(value)}°`;
}

/** Compass bearing to a Chinese wind name. */
function windDir(deg: number): string {
  const names = ["北", "东北", "东", "东南", "南", "西南", "西", "西北"];
  return names[Math.round(((deg % 360) + 360) % 360 / 45) % 8];
}

function uvLabel(index: number): string {
  if (index < 0.5) return "无";
  if (index < 3) return "弱";
  if (index < 6) return "中等";
  if (index < 8) return "强";
  if (index < 11) return "很强";
  return "极强";
}

function hhmm(stamp: number): string {
  const d = new Date(stamp);
  return `${d.getHours().toString().padStart(2, "0")}:${d.getMinutes().toString().padStart(2, "0")}`;
}

/** One metric tile in the facts grid. `ratio` draws a thin progress track. */
function Metric({
  label,
  value,
  sub,
  ratio,
}: {
  label: string;
  value: string;
  sub?: string;
  ratio?: number;
}) {
  return (
    <div className="wx-metric">
      <span className="wx-metric-label">{label}</span>
      <span className="wx-metric-value">{value}</span>
      {ratio !== undefined ? (
        <span className="wx-metric-track">
          <i style={{ width: `${Math.max(0, Math.min(1, ratio)) * 100}%` }} />
        </span>
      ) : (
        <span className="wx-metric-sub">{sub}</span>
      )}
    </div>
  );
}

interface WeatherContentProps {
  forecast: Forecast;
  cities: Location[];
  active: number;
  loading: boolean;
  onFocusIsland: () => void;
  onStep: (delta: number) => void;
  onJump: (index: number) => void;
}

/** One city's readings. The track renders up to three of these, side by side. */
function WeatherContent({
  forecast,
  cities,
  active,
  loading,
  onFocusIsland,
  onStep,
  onJump,
}: WeatherContentProps) {
  const days = forecast.days ?? [];
  // A shared axis for the daily range bars, so the bars compare across days.
  const lo = days.length ? Math.min(...days.map((d) => d.min)) : 0;
  const hi = days.length ? Math.max(...days.map((d) => d.max)) : 1;
  const span = Math.max(hi - lo, 1);

  return (
    <>
      <section className="wx-hero">
        <div className="wx-medallion">
          <WeatherIcon code={forecast.current.code} size={58} night={!forecast.current.isDay} />
        </div>

        <div className="wx-hero-main">
          <div className="wx-hero-temp">
            {degree(forecast.current.temperature)}
            <span>{forecast.current.description}</span>
          </div>
          <div className="wx-hero-sub">
            {cities.length > 1 && (
              <button
                type="button"
                className="wx-city-step"
                onPointerEnter={onFocusIsland}
                onPointerDown={onFocusIsland}
                onClick={() => onStep(-1)}
                title="上一个城市"
                aria-label="上一个城市"
              >
                ‹
              </button>
            )}
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
              <path d="M12 21s6.5-6 6.5-10.5a6.5 6.5 0 1 0-13 0C5.5 15 12 21 12 21Z" />
              <circle cx="12" cy="10.5" r="2.2" />
            </svg>
            {[forecast.location.name, forecast.location.region].filter(Boolean).join(" · ")}
            {cities.length > 1 && (
              <>
                <button
                  type="button"
                  className="wx-city-step"
                  onPointerEnter={onFocusIsland}
                  onPointerDown={onFocusIsland}
                  onClick={() => onStep(1)}
                  title="下一个城市"
                  aria-label="下一个城市"
                >
                  ›
                </button>
                <div className="wx-city-dots">
                  {cities.map((city, index) => (
                    <button
                      key={`${city.lat},${city.lon}`}
                      type="button"
                      className={index === active ? "wx-city-dot is-active" : "wx-city-dot"}
                      onPointerEnter={onFocusIsland}
                      onPointerDown={onFocusIsland}
                      onClick={() => onJump(index)}
                      title={city.name}
                      aria-label={`切换到 ${city.name}`}
                      aria-current={index === active ? "true" : undefined}
                    />
                  ))}
                </div>
              </>
            )}
            <span className="wx-dot" />
            {forecast.current.isDay ? "白天" : "夜间"}
          </div>
        </div>

        <div className="wx-hero-side">
          <div className="wx-sun">
            <span className="wx-sun-up">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                <path d="M3 18h18" />
                <path d="M7.5 18a4.5 4.5 0 0 1 9 0" />
                <path d="M12 3.5v3M6.6 6.1l1.5 1.5M17.4 6.1l-1.5 1.5" />
              </svg>
              日出 <b>{days[0]?.sunrise || "—"}</b>
            </span>
            <span className="wx-sun-down">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                <path d="M3 18h18" />
                <path d="M7.5 18a4.5 4.5 0 0 1 9 0" />
                <path d="M12 3.5v3.5M10.2 5.5 12 7.3l1.8-1.8" />
              </svg>
              日落 <b>{days[0]?.sunset || "—"}</b>
            </span>
          </div>
        </div>
      </section>

      <section className="wx-metrics">
        <Metric
          label="体感"
          value={degree(forecast.current.apparent)}
          sub={`实际 ${degree(forecast.current.temperature)}`}
        />
        <Metric
          label="湿度"
          value={`${forecast.current.humidity}%`}
          ratio={forecast.current.humidity / 100}
        />
        <Metric
          label="风力"
          value={`${windDir(forecast.current.windDir)}风`}
          sub={`${Math.round(forecast.current.windSpeed)} km/h · 阵风 ${Math.round(forecast.current.windGusts)}`}
        />
        <Metric
          label="降水"
          value={`${days[0]?.precipProb ?? 0}%`}
          sub={`今日 ${(days[0]?.precipSum ?? 0).toFixed(1)} mm`}
        />
        <Metric
          label="云量"
          value={`${forecast.current.cloudCover}%`}
          ratio={forecast.current.cloudCover / 100}
        />
        <Metric
          label="气压"
          value={`${Math.round(forecast.current.pressure)}`}
          sub="hPa"
        />
        <Metric
          label="能见度"
          value={`${(forecast.current.visibility / 1000).toFixed(1)}`}
          sub="km"
        />
        <Metric
          label="紫外线"
          value={`${forecast.current.uvIndex.toFixed(1)}`}
          sub={uvLabel(forecast.current.uvIndex)}
        />
      </section>

      <section className="wx-days">
        {days.map((day: Day, index) => {
          const left = ((day.min - lo) / span) * 100;
          const width = Math.max(((day.max - day.min) / span) * 100, 6);
          return (
            <div className="wx-day" key={day.date}>
              <span className="wx-day-name">{weekday(day.date, index)}</span>
              <WeatherIcon code={day.code} size={22} />
              <span className="wx-day-desc">{day.description}</span>
              <span className={day.precipProb > 0 ? "wx-day-rain is-wet" : "wx-day-rain"}>
                {day.precipProb > 0 ? `${day.precipProb}%` : ""}
              </span>
              <span className="wx-day-bar">
                <i style={{ left: `${left}%`, width: `${width}%` }} />
              </span>
              <span className="wx-day-temp">
                {degree(day.max)}
                <em>{degree(day.min)}</em>
              </span>
            </div>
          );
        })}
      </section>

      <footer className="wx-foot">
        <span className={loading ? "wx-updated is-live" : "wx-updated"}>
          {loading ? "更新中…" : `更新于 ${hhmm(forecast.updatedAt)}`}
        </span>
        <span>Open-Meteo</span>
      </footer>
    </>
  );
}

interface WeatherPaneProps {
  forecast: Forecast | null;
  city: Location | null;
  cities: Location[];
  active: number;
  loading: boolean;
  side?: boolean;
  onFocusIsland: () => void;
  onStep: (delta: number) => void;
  onJump: (index: number) => void;
}

/** One page of the strip: readings, or a waiting card until prefetch lands. */
function WeatherPane({ forecast, city, cities, active, loading, side, onFocusIsland, onStep, onJump }: WeatherPaneProps) {
  return (
    <div className={side ? "wx-pane is-side" : "wx-pane"}>
      {forecast ? (
        <WeatherContent
          forecast={forecast}
          cities={cities}
          active={active}
          loading={loading}
          onFocusIsland={onFocusIsland}
          onStep={onStep}
          onJump={onJump}
        />
      ) : (
        <div className="wx-pending">
          <b>{city ? city.name : "—"}</b>
          <span>{city?.region || city?.country || ""}</span>
          <em>正在获取天气…</em>
        </div>
      )}
    </div>
  );
}

/** The weather tab: a three-page strip (previous / current / next) that the
    pointer drags directly. Neighbouring cities are prefetched, so releasing
    simply finishes the slide — no bounce back and no wait. */
export function WeatherPanel({ weather, onFocusIsland }: WeatherPanelProps) {
  const { cities, active, loading, failed, step, forecastAt, warm } = weather;

  const panelRef = useRef<HTMLDivElement>(null);
  const bodyRef = useRef<HTMLDivElement>(null);
  const trackRef = useRef<HTMLDivElement>(null);
  const drag = useRef<{ id: number; x: number; y: number; dx: number; width: number; locked: boolean } | null>(null);
  const busy = useRef(false);
  const flight = useRef<{ to: number } | null>(null);
  const timers = useRef<number[]>([]);
  const shineId = useRef(0);
  // Drag frames are coalesced to one style write per display refresh, and the
  // last painted offset is what the release glide animates from.
  const frame = useRef(0);
  const queued = useRef<number | null>(null);
  const applied = useRef(0);
  const glideRef = useRef<Animation | null>(null);

  // The page on screen. It is pinned while the strip travels and cleared once
  // the hook's active index has caught up with it.
  const [center, setCenter] = useState<number | null>(null);
  const [recenter, setRecenter] = useState(0);
  const [shine, setShine] = useState<{ dir: number; key: number } | null>(null);

  const count = cities.length;
  const shown = center !== null && center < count ? center : active;
  const wrap = (index: number) => wrapIndex(index, count);

  const clearTimers = () => {
    timers.current.forEach((id) => window.clearTimeout(id));
    timers.current = [];
  };
  const later = (fn: () => void, ms: number) => {
    timers.current.push(window.setTimeout(fn, ms));
  };

  useEffect(() => {
    warm();
  }, [warm]);

  // The local page hand-off is complete once the hook agrees on the index.
  useEffect(() => {
    if (center !== null && center === active && !busy.current) setCenter(null);
  }, [center, active]);

  // A city was added or removed; drop any pinned page so the indices stay valid.
  useEffect(() => {
    setCenter(null);
  }, [count]);

  useEffect(() => {
    return () => {
      clearTimers();
      if (frame.current) cancelAnimationFrame(frame.current);
      glideRef.current?.cancel();
    };
  }, []);

  // No live gesture means the CSS base transform is authoritative: a third of
  // the strip for several cities, none for a single one.
  useLayoutEffect(() => {
    applied.current = 0;
    if (trackRef.current) trackRef.current.style.transform = "";
  }, [count]);

  // Re-lay the idle stack at the CSS base if the panel is resized between
  // gestures (the percentage base follows the new width by itself).
  useEffect(() => {
    const body = bodyRef.current;
    const track = trackRef.current;
    if (!body || !track) return;
    const observer = new ResizeObserver(() => {
      if (busy.current || drag.current) return;
      applied.current = 0;
      track.style.transform = "";
    });
    observer.observe(body);
    return () => observer.disconnect();
  }, [count]);

  // After the strip parks on the target page, rebase it to the middle slot in
  // the same paint the content swaps in: cancel the glide, drop the pixel
  // offset, and show the new centre — nothing visibly moves.
  useLayoutEffect(() => {
    if (!recenter) return;
    const body = bodyRef.current;
    const track = trackRef.current;
    if (!body || !track) return;
    glideRef.current?.cancel();
    glideRef.current = null;
    applied.current = 0;
    track.style.transform = `translate3d(${count > 1 ? -(body.clientWidth || 0) : 0}px, 0, 0)`;
  }, [recenter, count]);

  const finish = () => {
    const done = flight.current;
    flight.current = null;
    busy.current = false;
    if (!done) return;
    setCenter(done.to);
    setRecenter((n) => n + 1);
  };

  /** Glide from the last painted drag offset to `shift` (px, relative to the
      centred page) with one compositor animation. */
  const glide = (shift: number) => {
    const body = bodyRef.current;
    const track = trackRef.current;
    if (!body || !track) {
      finish();
      return;
    }
    if (frame.current) {
      cancelAnimationFrame(frame.current);
      frame.current = 0;
    }
    queued.current = null;
    const width = body.clientWidth || 260;
    const base = count > 1 ? -width : 0;
    glideRef.current?.cancel();
    if (typeof track.animate !== "function") {
      track.style.transform = `translate3d(${base + shift}px, 0, 0)`;
      finish();
      return;
    }
    const anim = track.animate(
      [
        { transform: `translate3d(${base + applied.current}px, 0, 0)` },
        { transform: `translate3d(${base + shift}px, 0, 0)` },
      ],
      { duration: GLIDE_MS, easing: "cubic-bezier(0.22, 0.9, 0.24, 1)", fill: "forwards" },
    );
    glideRef.current = anim;
    const settle = !flight.current;
    anim.onfinish = () => {
      if (glideRef.current !== anim) return;
      applied.current = shift;
      if (settle) {
        // Spring-back: nothing re-renders, so hand the final offset back to a
        // plain inline transform and drop the finished animation.
        glideRef.current = null;
        anim.cancel();
        track.style.transform = `translate3d(${base + shift}px, 0, 0)`;
      }
      finish();
    };
  };

  /** Slide one page towards `delta` and commit it. */
  const slideTo = (delta: number) => {
    if (!delta || busy.current || count < 2) return;
    const from = shown;
    const to = wrap(from + delta);
    const dir = Math.sign(delta);
    busy.current = true;
    flight.current = { to };
    setCenter(from);
    step(delta);
    shineId.current += 1;
    setShine({ dir, key: shineId.current });
    later(() => setShine(null), 800);
    glide(-dir * (bodyRef.current?.clientWidth || 260));
  };

  const jump = (index: number) => {
    if (count < 2 || index === shown) return;
    let delta = index - shown;
    if (delta > count / 2) delta -= count;
    if (delta < -count / 2) delta += count;
    if (!delta) return;
    if (Math.abs(delta) === 1) {
      slideTo(delta);
      return;
    }
    // A far dot: the strip only carries adjacent pages, so commit directly.
    busy.current = true;
    setCenter(index);
    step(delta);
    later(() => {
      busy.current = false;
    }, GLIDE_MS);
  };

  const begin = (event: ReactPointerEvent<HTMLDivElement>) => {
    if (drag.current || busy.current || count < 2 || !forecastAt(shown)) return;
    if ((event.target as HTMLElement).closest("button")) return;
    drag.current = { id: event.pointerId, x: event.clientX, y: event.clientY, dx: 0, width: 0, locked: false };
    onFocusIsland();
  };

  /** One style write per display refresh, using the width captured at lock. */
  const paint = () => {
    frame.current = 0;
    const move = drag.current;
    const strip = trackRef.current;
    const shift = queued.current;
    if (!move || !move.locked || !strip || shift === null) return;
    applied.current = shift;
    strip.style.transform = `translate3d(${-move.width + shift}px, 0, 0)`;
  };

  const track = (event: ReactPointerEvent<HTMLDivElement>) => {
    const move = drag.current;
    if (!move || move.id !== event.pointerId) return;
    const dx = event.clientX - move.x;
    const dy = event.clientY - move.y;
    if (!move.locked) {
      // A 4px deadzone keeps taps intact without a visible jump when the
      // horizontal intent wins; everything after that is 1:1 with the finger.
      if (Math.abs(dx) < 4) return;
      if (Math.abs(dx) <= Math.abs(dy) + 6) {
        drag.current = null;
        return;
      }
      move.locked = true;
      move.width = bodyRef.current?.clientWidth || 260;
      event.currentTarget.setPointerCapture(event.pointerId);
      panelRef.current?.setAttribute("data-swiping", "true");
    }
    move.dx = dx;
    const magnitude = Math.abs(dx);
    // Page for page; past one page the strip gets heavier.
    const shift = magnitude <= move.width ? dx : Math.sign(dx) * (move.width + (magnitude - move.width) * 0.2);
    queued.current = shift;
    if (!frame.current) frame.current = requestAnimationFrame(paint);
  };

  const release = (event: ReactPointerEvent<HTMLDivElement>) => {
    const move = drag.current;
    drag.current = null;
    if (!move) return;
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId);
    }
    panelRef.current?.removeAttribute("data-swiping");
    if (!move.locked) return;
    if (Math.abs(move.dx) >= SWIPE_TRIGGER) {
      slideTo(move.dx < 0 ? 1 : -1);
      return;
    }
    // Below the trigger: glide the strip back to the middle page.
    busy.current = true;
    glide(0);
  };

  const centerForecast = forecastAt(shown);
  const prevForecast = count > 1 ? forecastAt(wrap(shown - 1)) : null;
  const nextForecast = count > 1 ? forecastAt(wrap(shown + 1)) : null;
  const solo = count < 2;

  return (
    <div
      ref={panelRef}
      className="weather-panel"
      onPointerDown={begin}
      onPointerMove={track}
      onPointerUp={release}
      onPointerCancel={release}
    >
      {count === 0 ? (
        <div className="empty-state">
          {failed ? (
            <>
              <span className="power-glyph">☁</span>
              <div>
                <b>暂时获取不到天气</b>
                <span>检查网络连接，或在设置里添加城市</span>
              </div>
            </>
          ) : (
            <span>正在获取天气…</span>
          )}
        </div>
      ) : failed && !centerForecast ? (
        <div className="empty-state">
          <span className="power-glyph">☁</span>
          <div>
            <b>暂时获取不到天气</b>
            <span>检查网络连接，或在设置里添加城市</span>
          </div>
        </div>
      ) : (
        <div
          className="wx-body"
          ref={bodyRef}
          data-solo={solo ? "true" : undefined}
        >
          <div className="wx-track" ref={trackRef}>
            {!solo && (
              <WeatherPane
                side
                forecast={prevForecast}
                city={cities[wrap(shown - 1)]}
                cities={cities}
                active={wrap(shown - 1)}
                loading={false}
                onFocusIsland={onFocusIsland}
                onStep={() => undefined}
                onJump={() => undefined}
              />
            )}
            <WeatherPane
              forecast={centerForecast}
              city={cities[shown] ?? null}
              cities={cities}
              active={shown}
              loading={loading}
              onFocusIsland={onFocusIsland}
              onStep={(delta) => slideTo(delta)}
              onJump={jump}
            />
            {!solo && (
              <WeatherPane
                side
                forecast={nextForecast}
                city={cities[wrap(shown + 1)]}
                cities={cities}
                active={wrap(shown + 1)}
                loading={false}
                onFocusIsland={onFocusIsland}
                onStep={() => undefined}
                onJump={() => undefined}
              />
            )}
          </div>
          {shine && (
            <span
              className={shine.dir > 0 ? "wx-sheen" : "wx-sheen is-prev"}
              key={`sheen-${shine.key}`}
              aria-hidden="true"
            />
          )}
        </div>
      )}
    </div>
  );
}
