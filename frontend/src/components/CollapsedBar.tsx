import type { Snapshot } from "../../bindings/doter/internal/sysinfo/models";
import { formatPercent } from "../lib/format";

interface CollapsedBarProps {
  snapshot: Snapshot | null;
}

/** The idle pill: CPU and memory at a glance. */
export function CollapsedBar({ snapshot }: CollapsedBarProps) {
  const cpu = snapshot?.cpu.usage ?? 0;
  const memory = snapshot?.memory.usedPercent ?? 0;

  const memoryColour =
    memory >= 80
      ? "var(--accent-danger)"
      : memory >= 60
        ? "var(--accent-warn)"
        : "var(--accent-cpu)";

  return (
    <div className="collapsed-bar">
      <div className="quick">
        <span className="quick-value">{formatPercent(cpu)}</span>
      </div>

      <div className="quick quick-stacked">
        <span
          className="memory-dot"
          style={{ background: memoryColour, boxShadow: `0 0 6px ${memoryColour}` }}
          aria-hidden="true"
        />
        <span className="quick-value">{formatPercent(memory)}</span>
      </div>
    </div>
  );
}