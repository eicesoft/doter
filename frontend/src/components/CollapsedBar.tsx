import type { Snapshot } from "../../bindings/doter/internal/sysinfo/models";
import type { CSSProperties } from "react";
import type { Forecast } from "../../bindings/doter/internal/weather/models";
import type { CollapsedMetric } from "../lib/settings";
import { formatClock, formatPercent, formatRateCompact } from "../lib/format";
import { WeatherIcon, weatherAccent } from "./WeatherIcon";

interface CollapsedBarProps {
  snapshot: Snapshot | null;
  clock: Date;
  agents: number;
  forecast: Forecast | null;
  metrics: readonly CollapsedMetric[];
}

/** Dot colour shared by the collapsed stats: calm below 60%, warning above. */
function loadColour(percent: number) {
  if (percent >= 80) return "var(--accent-danger)";
  if (percent >= 60) return "var(--accent-warn)";
  return "var(--accent-cpu)";
}

function batteryColour(percent: number) {
  if (percent <= 20) return "var(--accent-danger)";
  if (percent <= 40) return "var(--accent-warn)";
  return "var(--accent-battery)";
}

interface MetricProps {
  metric: CollapsedMetric;
  snapshot: Snapshot | null;
  clock: Date;
  agents: number;
  forecast: Forecast | null;
}

function Metric({ metric, snapshot, clock, agents, forecast }: MetricProps) {
  const cpu = snapshot?.cpu.usage ?? 0;
  const memory = snapshot?.memory.usedPercent ?? 0;
  const battery = snapshot?.battery;

  switch (metric) {
    case "cpu":
      return <Stat colour={loadColour(cpu)} value={formatPercent(cpu)} title="CPU 使用率" />;
    case "memory":
      return <Stat colour={loadColour(memory)} value={formatPercent(memory)} title="内存占用" />;
    case "network":
      return (
        <Stat
          colour="var(--accent-net)"
          value={`↓${formatRateCompact(snapshot?.network.downSpeed ?? 0)}`}
          title="下行速率"
        />
      );
    case "battery":
      return (
        <Stat
          colour={battery ? batteryColour(battery.percent) : "var(--dim)"}
          value={battery ? formatPercent(battery.percent) : "—"}
          title={battery ? `电池 ${battery.state}` : "无电池"}
        />
      );
    case "agents":
      return (
        <Stat
          colour={agents > 0 ? "var(--accent-sys)" : "var(--dim)"}
          value={agents > 0 ? String(agents) : "0"}
          title="运行中的 Agent"
        />
      );
    case "clock":
      return (
        <div className="quick quick-stacked quick-clock" title="时间">
          <span className="stat-dot is-hidden" aria-hidden="true" />
          <span className="quick-value">{formatClock(clock, false)}</span>
        </div>
      );
    case "weather": {
      const current = forecast?.current;
      // The glyph sits in the dot slot so the pill keeps one row height; it
      // is deliberately tiny, and the temperature carries the reading. The
      // glyph alone takes the sky's colour, as it would in the expanded tab.
      const accent = current ? weatherAccent(current.code, !current.isDay) : undefined;
      return (
        <div
          className="quick quick-stacked quick-weather"
          title={weatherTitle(forecast)}
          style={accent ? ({ "--wx-accent": accent } as CSSProperties) : undefined}
        >
          {current ? (
            <WeatherIcon code={current.code} size={9} night={!current.isDay} />
          ) : (
            <span className="quick-weather-void" aria-hidden="true" />
          )}
          <span className="quick-value">{current ? `${Math.round(current.temperature)}°` : "—"}</span>
        </div>
      );
    }
  }
}

function weatherTitle(forecast: Forecast | null): string {
  const current = forecast?.current;
  if (!current) return "天气数据暂不可用";
  const place = forecast?.location.name;
  return [place, current.description, `体感 ${Math.round(current.apparent)}°`]
    .filter(Boolean)
    .join(" · ");
}

function Stat({ colour, value, title }: { colour: string; value: string; title: string }) {
  return (
    <div className="quick quick-stacked" title={title}>
      <span
        className="stat-dot"
        style={{ background: colour, boxShadow: `0 0 6px ${colour}` }}
        aria-hidden="true"
      />
      <span className="quick-value">{value}</span>
    </div>
  );
}

/** The idle pill: two user-selected metrics at a glance. */
export function CollapsedBar({ snapshot, clock, agents, forecast, metrics }: CollapsedBarProps) {
  return (
    <div className={metrics.length === 1 ? "collapsed-bar is-single" : "collapsed-bar"}>
      {metrics.map((metric, index) => (
        <Metric
          key={`${metric}-${index}`}
          metric={metric}
          snapshot={snapshot}
          clock={clock}
          agents={agents}
          forecast={forecast}
        />
      ))}
    </div>
  );
}
