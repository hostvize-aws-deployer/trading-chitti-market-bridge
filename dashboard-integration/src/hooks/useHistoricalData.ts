import { useState, useCallback } from 'react';
import { useMarketBridge } from './useMarketBridge';

export interface Candle {
  candle_timestamp: string;
  open: number;
  high: number;
  low: number;
  close: number;
  volume: number;
  oi?: number;
}

export interface HistoricalDataResponse {
  exchange: string;
  symbol: string;
  interval: string;
  count: number;
  candles: Candle[];
}

export interface HistoricalDataRequest {
  exchange: string;
  symbol: string;
  interval: '1m' | '5m' | '15m' | '1h' | 'day';
  from_date: string; // YYYY-MM-DD
  to_date: string; // YYYY-MM-DD
}

export function useHistoricalData() {
  const { baseUrl } = useMarketBridge();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchHistoricalData = useCallback(
    async (request: HistoricalDataRequest): Promise<Candle[]> => {
      setLoading(true);
      setError(null);

      try {
        const response = await fetch(`${baseUrl}/historical/`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify(request),
        });

        if (!response.ok) {
          throw new Error(`Fetch failed: ${response.statusText}`);
        }

        const data: HistoricalDataResponse = await response.json();
        return data.candles;
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Fetch failed';
        setError(errorMessage);
        return [];
      } finally {
        setLoading(false);
      }
    },
    [baseUrl]
  );

  const fetch52DayData = useCallback(
    async (exchange: string, symbol: string): Promise<Candle[]> => {
      setLoading(true);
      setError(null);

      try {
        const response = await fetch(
          `${baseUrl}/historical/52day?exchange=${exchange}&symbol=${symbol}`
        );

        if (!response.ok) {
          throw new Error(`Fetch failed: ${response.statusText}`);
        }

        const data: HistoricalDataResponse = await response.json();
        return data.candles;
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Fetch failed';
        setError(errorMessage);
        return [];
      } finally {
        setLoading(false);
      }
    },
    [baseUrl]
  );

  const warmCache = useCallback(
    async (
      exchange: string,
      symbols: string[],
      interval: string,
      days: number
    ): Promise<boolean> => {
      setLoading(true);
      setError(null);

      try {
        const response = await fetch(`${baseUrl}/historical/warm-cache`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            exchange,
            symbols,
            interval,
            days,
          }),
        });

        if (!response.ok) {
          throw new Error(`Cache warming failed: ${response.statusText}`);
        }

        return true;
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Cache warming failed';
        setError(errorMessage);
        return false;
      } finally {
        setLoading(false);
      }
    },
    [baseUrl]
  );

  return {
    fetchHistoricalData,
    fetch52DayData,
    warmCache,
    loading,
    error,
  };
}
