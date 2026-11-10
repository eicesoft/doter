import { useEffect, useState } from "react";

/**
 * Keeps a bounded history for sparklines. `tick` is the sample identity, so
 * identical metric values still produce a new point on every reading.
 */
export function useSeries(value: number, tick: number, length = 40): number[] {
  const [series, setSeries] = useState<number[]>([]);

  useEffect(() => {
    if (!tick) return;
    setSeries((previous) => {
      const next =
        previous.length >= length
          ? previous.slice(previous.length - length + 1)
          : previous.slice();
      next.push(Number.isFinite(value) ? value : 0);
      return next;
    });
    // Only a new sample should append; value is read from the closed-over render.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tick, length]);

  return series;
}