import type { Snapshot } from "../../bindings/doter/internal/sysinfo/models";
import { formatPercent } from "../lib/format";

interface CollapsedBarProps {
  snapshot: Snapshot | null;
}

/** A chip glyph standing in for the "CPU" label. */
function CpuIcon() {
  return (
    <svg
      className="quick-icon"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      role="img"
      aria-label="CPU"
    >
      <rect x="5.5" y="5.5" width="13" height="13" rx="3" />
      <rect x="10" y="10" width="4" height="4" rx="1" />
    </svg>
  );
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
        <CpuIcon />
        <span className="quick-value">{formatPercent(cpu)}</span>
      </div>

      <div className="quick">
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