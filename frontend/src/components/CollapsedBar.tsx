import type { Snapshot } from "../../bindings/doter/internal/sysinfo/models";
import { clampPercent, formatPercent } from "../lib/format";

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

/** A memory stick glyph standing in for the "MEM" label. */
function MemoryIcon() {
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
      aria-label="Memory"
    >
      <rect x="2.5" y="7" width="19" height="9" rx="1.5" />
      <path d="M6 16v3M11 16v3M16 16v3" />
    </svg>
  );
}

/** The idle pill: CPU and memory at a glance. */
export function CollapsedBar({ snapshot }: CollapsedBarProps) {
  const cpu = snapshot?.cpu.usage ?? 0;
  const memory = snapshot?.memory.usedPercent ?? 0;

  return (
    <div className="collapsed-bar">
      <div className="quick">
        <CpuIcon />
        <span className="quick-value">{formatPercent(cpu)}</span>
      </div>

      <div className="quick">
        <MemoryIcon />
        <span className="quick-meter" aria-hidden="true">
          <span style={{ width: `${clampPercent(memory)}%` }} />
        </span>
      </div>
    </div>
  );
}