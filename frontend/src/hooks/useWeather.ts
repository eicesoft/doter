import { useCallback, useEffect, useRef, useState } from "react";

import {
  AddCity,
  GetCityForecast,
  GetState,
  RemoveCity,
  StepCity,
} from "../../bindings/doter/internal/services/weatherservice";
import type { Forecast, Location, State } from "../../bindings/doter/internal/weather/models";

/** Refetch the forecast while the panel stays open. */
const REFRESH_MS = 10 * 60_000;

/** Everything the weather UI needs: the active forecast and the city list. */
export interface WeatherController {
  forecast: Forecast | null;
  cities: Location[];
  active: number;
  loading: boolean;
  failed: boolean;
  /** Move the active city left (-1) or right (+1), wrapping around. */
  step: (delta: number) => void;
  /** Add a city and make it active. */
  addCity: (location: Location) => void;
  /** Drop a city by its index in the list. */
  removeCity: (index: number) => void;
  /** A city's forecast by index, including prefetched neighbours. */
  forecastAt: (index: number) => Forecast | null;
  /** Start loading the neighbouring cities' pages for the swipe strip. */
  warm: () => void;
}

function cityKey(loc: Location): string {
  return `${loc.name}@${loc.lat},${loc.lon}`;
}

/** The weather forecast, refreshed every ten minutes. It lives on the app
    level so the idle pill can show the temperature without the panel mounted. */
export function useWeather(): WeatherController {
  const [forecast, setForecast] = useState<Forecast | null>(null);
  const [cities, setCities] = useState<Location[]>([]);
  const [active, setActive] = useState(0);
  const [loading, setLoading] = useState(true);
  const [failed, setFailed] = useState(false);
  // Neighbour forecasts, so the swipe strip can show the adjacent pages before
  // the user lets go. Keyed by city, filled by Peek and by normal state loads.
  const [known, setKnown] = useState<Record<string, Forecast>>({});
  const warmed = useRef(false);
  const [warmTick, setWarmTick] = useState(0);

  // Only the newest request may write state; a city step and the timed
  // refresh can otherwise land out of order.
  const seq = useRef(0);

  const remember = useCallback((f: Forecast | null | undefined) => {
    if (!f) return;
    const key = cityKey(f.location);
    setKnown((prev) => {
      if (prev[key]?.updatedAt === f.updatedAt) return prev;
      return { ...prev, [key]: f };
    });
  }, []);

  const applyState = useCallback((state: State | null) => {
    if (!state) {
      setFailed(true);
      return;
    }
    setCities(state.cities ?? []);
    setActive(state.active ?? 0);
    setForecast(state.forecast ?? null);
    setFailed(!state.forecast);
    remember(state.forecast);
  }, [remember]);

  const applyCities = useCallback((state: State | null) => {
    if (!state) return;
    setCities(state.cities ?? []);
    setActive(state.active ?? 0);
  }, []);

  const refresh = useCallback(() => {
    const id = ++seq.current;
    setLoading(true);
    GetState()
      .then((state) => {
        if (id === seq.current) applyState(state);
      })
      .catch(() => {
        if (id === seq.current) setFailed(true);
      })
      .finally(() => {
        if (id === seq.current) setLoading(false);
      });
  }, [applyState]);

  const mutate = useCallback(
    (operation: () => Promise<State | null>) => {
      const id = ++seq.current;
      setLoading(true);
      operation()
        .then((state) => {
          if (id !== seq.current) return undefined;
          applyCities(state);
          return GetState();
        })
        .then((state) => {
          if (id === seq.current && state !== undefined) applyState(state);
        })
        .catch(() => {
          if (id === seq.current) setFailed(true);
        })
        .finally(() => {
          if (id === seq.current) setLoading(false);
        });
    },
    [applyCities, applyState],
  );

  useEffect(() => {
    refresh();
    const id = window.setInterval(refresh, REFRESH_MS);
    return () => window.clearInterval(id);
  }, [refresh]);

  const warm = useCallback(() => {
    if (warmed.current) return;
    warmed.current = true;
    setWarmTick((n) => n + 1);
  }, []);

  // Keep both adjacent pages ready. With two cities both sides are the same
  // city, so one request covers it.
  useEffect(() => {
    if (!warmed.current || cities.length < 2) return;
    let alive = true;
    const offsets = cities.length === 2 ? [1] : [-1, 1];
    offsets.forEach((offset) => {
      GetCityForecast(offset)
        .then((f) => {
          if (alive) remember(f);
        })
        .catch(() => undefined);
    });
    return () => {
      alive = false;
    };
  }, [active, cities.length, warmTick, remember]);

  const forecastAt = useCallback(
    (index: number) => {
      const city = cities[index];
      if (!city) return null;
      const key = cityKey(city);
      if (forecast && cityKey(forecast.location) === key) return forecast;
      return known[key] ?? null;
    },
    [cities, forecast, known],
  );

  const step = useCallback((delta: number) => mutate(() => StepCity(delta)), [mutate]);
  const addCity = useCallback((location: Location) => mutate(() => AddCity(location)), [mutate]);
  const removeCity = useCallback((index: number) => mutate(() => RemoveCity(index)), [mutate]);

  return {
    forecast,
    cities,
    active,
    loading,
    failed,
    step,
    addCity,
    removeCity,
    forecastAt,
    warm,
  };
}
