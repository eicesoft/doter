import type { Day } from "../../bindings/doter/internal/weather/models";
import { useWeather } from "../hooks/useWeather";

interface WeatherPanelProps {
  onFocusIsland: () => void;
}

/** WMO weather code groups the icon set is drawn from. */
type Kind = "clear" | "partly" | "cloudy" | "fog" | "rain" | "snow" | "storm";

function kindOf(code: number): Kind {
  switch (code) {
    case 0:
      return "clear";
    case 1:
    case 2:
      return "partly";
    case 3:
      return "cloudy";
    case 45:
    case 48:
      return "fog";
    case 51:
    case 53:
    case 55:
    case 56:
    case 57:
    case 61:
    case 63:
    case 65:
    case 66:
    case 67:
    case 80:
    case 81:
    case 82:
      return "rain";
    case 71:
    case 73:
    case 75:
    case 77:
    case 85:
    case 86:
      return "snow";
    default:
      return code >= 95 ? "storm" : "cloudy";
  }
}

/** A minimal weather glyph, drawn in the panel's stroke style. */
function WeatherIcon({ code, size = 20 }: { code: number; size?: number }) {
  const kind = kindOf(code);
  const props = {
    width: size,
    height: size,
    viewBox: "0 0 24 24",
    fill: "none",
    stroke: "currentColor",
    strokeWidth: 1.7,
    strokeLinecap: "round" as const,
    strokeLinejoin: "round" as const,
    "aria-hidden": true,
  };

  switch (kind) {
    case "clear":
      return (
        <svg {...props} className="wx-icon wx-sun">
          <circle cx="12" cy="12" r="4" />
          <path d="M12 3v2M12 19v2M3 12h2M19 12h2M6 6l1.4 1.4M16.6 16.6 18 18M6 18l1.4-1.4M16.6 7.4 18 6" />
        </svg>
      );
    case "partly":
      return (
        <svg {...props} className="wx-icon wx-sun">
          <circle cx="8.5" cy="8.5" r="3" />
          <path d="M8.5 2.5v1.6M2.5 8.5h1.6M4.3 4.3l1.1 1.1M12.7 4.3l-1.1 1.1" />
          <path d="M9 20h8.5a3.5 3.5 0 0 0 .3-7 5 5 0 0 0-9.6 1.4A2.8 2.8 0 0 0 9 20Z" />
        </svg>
      );
    case "cloudy":
      return (
        <svg {...props} className="wx-icon wx-cloud">
          <path d="M7 18h10a3.6 3.6 0 0 0 .4-7.2 5.4 5.4 0 0 0-10.3-.6A3.9 3.9 0 0 0 7 18Z" />
        </svg>
      );
    case "fog":
      return (
        <svg {...props} className="wx-icon wx-fog">
          <path d="M7 13h10a3.6 3.6 0 0 0 .4-7.2 5.4 5.4 0 0 0-10.3-.6A3.9 3.9 0 0 0 7 13Z" />
          <path d="M5 17h14M8 20h9" />
        </svg>
      );
    case "snow":
      return (
        <svg {...props} className="wx-icon wx-snow">
          <path d="M7 14h10a3.6 3.6 0 0 0 .4-7.2 5.4 5.4 0 0 0-10.3-.6A3.9 3.9 0 0 0 7 14Z" />
          <path d="M8 18h.01M12 20h.01M16 18h.01" />
        </svg>
      );
    case "storm":
      return (
        <svg {...props} className="wx-icon wx-storm">
          <path d="M7 14h10a3.6 3.6 0 0 0 .4-7.2 5.4 5.4 0 0 0-10.3-.6A3.9 3.9 0 0 0 7 14Z" />
          <path d="M12.5 16 10 20h3l-1 3" />
        </svg>
      );
    default:
      return (
        <svg {...props} className="wx-icon wx-rain">
          <path d="M7 13h10a3.6 3.6 0 0 0 .4-7.2 5.4 5.4 0 0 0-10.3-.6A3.9 3.9 0 0 0 7 13Z" />
          <path d="M8.5 16.5 7.5 19M12 16.5 11 19M15.5 16.5 14.5 19" />
        </svg>
      );
  }
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

/** The weather tab. */
export function WeatherPanel({ onFocusIsland }: WeatherPanelProps) {
  const { forecast, loading, failed, query, setQuery, results, choose } = useWeather();

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
        <>
          <div className="weather-now">
            <WeatherIcon code={forecast.current.code} size={44} />
            <div className="weather-temperature">
              {degree(forecast.current.temperature)}
              <span>{forecast.current.description}</span>
            </div>
            <div className="weather-facts">
              <span>体感 {degree(forecast.current.apparent)}</span>
              <span>湿度 {forecast.current.humidity}%</span>
              <span>风速 {Math.round(forecast.current.windSpeed)} km/h</span>
            </div>
            <div className="weather-place">
              {[forecast.location.name, forecast.location.region]
                .filter(Boolean)
                .join(" · ")}
              {loading && <span className="weather-refreshing">更新中…</span>}
            </div>
          </div>

          <div className="weather-days">
            {(forecast.days ?? []).map((day: Day, index) => (
              <div className="weather-day" key={day.date}>
                <span className="weather-day-name">{weekday(day.date, index)}</span>
                <WeatherIcon code={day.code} size={18} />
                <span className="weather-day-desc">{day.description}</span>
                <span className="weather-day-temp">
                  {degree(day.max)} <em>{degree(day.min)}</em>
                </span>
                {day.precipProb > 0 && (
                  <span className="weather-day-rain">☂ {day.precipProb}%</span>
                )}
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  );
}
