import { useCallback, useEffect, useRef, useState } from "react";

import {
  GetForecast,
  SearchCities,
  SetLocation,
} from "../../bindings/doter/internal/services/weatherservice";
import type { Forecast, Location } from "../../bindings/doter/internal/weather/models";

/** Refetch the forecast while the panel stays open. */
const REFRESH_MS = 10 * 60_000;
/** Wait for the typing to settle before hitting the geocoding API. */
const SEARCH_DEBOUNCE_MS = 300;

/** Everything the weather UI needs: the forecast plus the city search. */
export interface WeatherController {
  forecast: Forecast | null;
  loading: boolean;
  failed: boolean;
  query: string;
  setQuery: (query: string) => void;
  results: Location[];
  choose: (location: Location) => void;
}

/** The weather forecast, refreshed every ten minutes. It lives on the app
    level so the idle pill can show the temperature without the panel mounted. */
export function useWeather(): WeatherController {
  const [forecast, setForecast] = useState<Forecast | null>(null);
  const [loading, setLoading] = useState(true);
  const [failed, setFailed] = useState(false);

  const [query, setQuery] = useState("");
  const [results, setResults] = useState<Location[]>([]);

  const refresh = useCallback(() => {
    setLoading(true);
    GetForecast()
      .then((f) => {
        setForecast(f);
        setFailed(f === null);
      })
      .catch(() => setFailed(true))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    refresh();
    const id = window.setInterval(refresh, REFRESH_MS);
    return () => window.clearInterval(id);
  }, [refresh]);

  const searchTimer = useRef<number | null>(null);
  useEffect(() => {
    if (searchTimer.current !== null) window.clearTimeout(searchTimer.current);
    if (!query.trim()) {
      setResults([]);
      return;
    }
    searchTimer.current = window.setTimeout(() => {
      SearchCities(query.trim())
        .then((list) => setResults(list ?? []))
        .catch(() => setResults([]));
    }, SEARCH_DEBOUNCE_MS);
    return () => {
      if (searchTimer.current !== null) window.clearTimeout(searchTimer.current);
    };
  }, [query]);

  const choose = useCallback((location: Location) => {
    setQuery("");
    setResults([]);
    setLoading(true);
    SetLocation(location)
      .then((f) => {
        setForecast(f);
        setFailed(f === null);
      })
      .catch(() => setFailed(true))
      .finally(() => setLoading(false));
  }, []);

  return { forecast, loading, failed, query, setQuery, results, choose };
}
