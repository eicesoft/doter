import type { ReactNode } from "react";

import { clampPercent } from "../lib/format";

interface RingProps {
  value: number;
  size?: number;
  stroke?: number;
  color?: string;
  children?: ReactNode;
  className?: string;
}

/** Circular progress used for CPU and disk utilization. */
export function Ring({
  value,
  size = 76,
  stroke = 6,
  color = "var(--accent-cpu)",
  children,
  className,
}: RingProps) {
  const radius = (size - stroke) / 2;
  const circumference = 2 * Math.PI * radius;
  const offset = circumference * (1 - clampPercent(value) / 100);

  return (
    <div className={`ring ${className ?? ""}`} style={{ width: size, height: size }}>
      <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`} aria-hidden="true">
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          fill="none"
          stroke="rgba(255,255,255,0.07)"
          strokeWidth={stroke}
        />
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          fill="none"
          stroke={color}
          strokeWidth={stroke}
          strokeLinecap="round"
          strokeDasharray={circumference}
          strokeDashoffset={offset}
          transform={`rotate(-90 ${size / 2} ${size / 2})`}
          style={{ transition: "stroke-dashoffset 700ms cubic-bezier(.32,.72,0,1)" }}
        />
      </svg>
      <div className="ring-center">{children}</div>
    </div>
  );
}

interface MeterProps {
  value: number;
  color?: string;
  className?: string;
}

/** Thin horizontal utilization bar. */
export function Meter({ value, color = "var(--accent-mem)", className }: MeterProps) {
  return (
    <div className={`meter ${className ?? ""}`} aria-hidden="true">
      <span
        style={{
          width: `${clampPercent(value)}%`,
          background: color,
          boxShadow: `0 0 10px color-mix(in srgb, ${color} 55%, transparent)`,
        }}
      />
    </div>
  );
}