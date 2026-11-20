import type { Day } from "../../bindings/doter/internal/weather/models";
import type { WeatherController } from "../hooks/useWeather";
import { WeatherIcon } from "./WeatherIcon";

interface WeatherPanelProps {
  weather: WeatherController;
  onFocusIsland: () => void;
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

/** The weather tab. */
export function WeatherPanel({ weather, onFocusIsland }: WeatherPanelProps) {
  const { forecast, loading, failed, query, setQuery, results, choose } = weather;

  const days = forecast?.days ?? [];
  // A shared axis for the daily range bars, so the bars compare across days.
  const lo = days.length ? Math.min(...days.map((d) => d.min)) : 0;
  const hi = days.length ? Math.max(...days.map((d) => d.max)) : 1;
  const span = Math.max(hi - lo, 1);

  return (
    <div className="weather-panel">
      <div className="clipboard-search weather-search">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round">
          <circle cx="11" cy="11" r="6.5" />
          <path d="M16 16l4.5 4.5" />
        </svg>
        <input
          type="text"
          value={query}
          placeholder={forecast ? `切换城市 · 当前 ${forecast.location.name}` : "搜索城市"}
          onPointerEnter={onFocusIsland}
          onPointerDown={onFocusIsland}
          onChange={(event) => setQuery(event.target.value)}
        />
        {query && (
          <button
            type="button"
            className="clipboard-search-clear"
            onClick={() => setQuery("")}
            aria-label="清除搜索"
          >
            ✕
          </button>
        )}
      </div>

      {query.trim() ? (
        <div className="weather-results">
          {results.length === 0 ? (
            <div className="empty-state">没有找到匹配的城市</div>
          ) : (
            results.map((location) => (
              <button
                key={`${location.name}-${location.lat}-${location.lon}`}
                type="button"
                className="weather-result"
                onClick={() => choose(location)}
              >
                <b>{location.name}</b>
                <span>{[location.region, location.country].filter(Boolean).join(" · ")}</span>
              </button>
            ))
          )}
        </div>
      ) : failed && !forecast ? (
        <div className="empty-state">
          <span className="power-glyph">☁</span>
          <div>
            <b>暂时获取不到天气</b>
            <span>检查网络连接后重试，或在上方搜索城市</span>
          </div>
        </div>
      ) : !forecast ? (
        <div className="empty-state">正在获取天气…</div>
      ) : (
        <div className="wx-body">
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
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                  <path d="M12 21s6.5-6 6.5-10.5a6.5 6.5 0 1 0-13 0C5.5 15 12 21 12 21Z" />
                  <circle cx="12" cy="10.5" r="2.2" />
                </svg>
                {[forecast.location.name, forecast.location.region].filter(Boolean).join(" · ")}
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
        </div>
      )}
    </div>
  );
}
