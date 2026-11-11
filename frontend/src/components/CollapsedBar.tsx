import type { Snapshot } from "../../bindings/doter/internal/sysinfo/models";
import { clampPercent, formatPercent } from "../lib/format";

interface CollapsedBarProps {
  snapshot: Snapshot | null;
}

/** The idle pill: CPU and memory at a glance. */
export function CollapsedBar({ snapshot }: CollapsedBarProps) {
  const cpu = snapshot?.cpu.usage ?? 0;
  const memory = snapshot?.memory.usedPercent ?? 0;

  return (
    <div className="collapsed-bar">
      <div className="quick">
        <span className="quick-label">CPU</span>
        <span className="quick-value">{formatPercent(cpu)}</span>
      </div>

      <div className="quick">
        <span className="quick-label">MEM</span>
        <span className="quick-meter" aria-hidden="true">
          <span style={{ width: `${clampPercent(memory)}%` }} />
        </span>
      </div>
    </div>
  );
}