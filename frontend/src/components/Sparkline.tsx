import { useId } from "react";

interface SparklineProps {
  data: number[];
  /** Upper bound of the scale; the series auto-scales when omitted. */
  max?: number;
  min?: number;
  color?: string;
  height?: number;
  className?: string;
  fill?: boolean;
}

const VIEW_WIDTH = 120;

/** A compact, non-interactive history chart for live metrics. */
export function Sparkline({
  data,
  max,
  min = 0,
  color = "var(--accent-cpu)",
  height = 32,
  className,
  fill = true,
}: SparklineProps) {
  const gradientId = useId().replace(/[:]/g, "");
  const values = data.length > 0 ? data : [0];
  const observedMax = Math.max(...values, 1);
  const ceiling = Math.max(max ?? observedMax, min + 1);
  const range = ceiling - min;
  const stepX = values.length > 1 ? VIEW_WIDTH / (values.length - 1) : 0;

  const points = values.map((value, index) => {
    const x = index * stepX;
    const clamped = Math.min(ceiling, Math.max(min, value));
    const y = height - ((clamped - min) / range) * (height - 2) - 1;
    return `${x.toFixed(2)},${y.toFixed(2)}`;
  });

  const line = `M${points.join(" L")}`;
  const area = `${line} L${VIEW_WIDTH},${height} L0,${height} Z`;

  return (
    <svg
      className={className}
      viewBox={`0 0 ${VIEW_WIDTH} ${height}`}
      preserveAspectRatio="none"
      aria-hidden="true"
    >
      <defs>
        <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor={color} stopOpacity="0.32" />
          <stop offset="100%" stopColor={color} stopOpacity="0" />
        </linearGradient>
      </defs>
      {fill && <path d={area} fill={`url(#${gradientId})`} />}
      <path
        d={line}
        fill="none"
        stroke={color}
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
        vectorEffect="non-scaling-stroke"
      />
    </svg>
  );
}