import { useState, useCallback } from 'react';
import { useMarketBridge } from './useMarketBridge';

export interface Instrument {
  instrument_token: number;
  exchange_token: number;
  tradingsymbol: string;
  name: string;
  exchange: string;
  segment: string;
  instrument_type: string;
  isin?: string;
  expiry?: string;
  strike?: number;
  tick_size?: number;
  lot_size?: number;
  last_price?: number;
  last_updated: string;
}

export interface InstrumentSearchResponse {
  query: string;
  count: number;
  instruments: Instrument[];
}

export function useInstruments() {
  const { baseUrl } = useMarketBridge();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const searchInstruments = useCallback(
    async (query: string, limit = 20): Promise<Instrument[]> => {
      setLoading(true);
      setError(null);

      try {
        const response = await fetch(
          `${baseUrl}/instruments/search?q=${encodeURIComponent(query)}&limit=${limit}`
        );

        if (!response.ok) {
          throw new Error(`Search failed: ${response.statusText}`);
        }

        const data: InstrumentSearchResponse = await response.json();
        return data.instruments;
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Search failed';
        setError(errorMessage);
        return [];
      } finally {
        setLoading(false);
      }
    },
    [baseUrl]
  );

  const getInstrumentByToken = useCallback(
    async (token: number): Promise<Instrument | null> => {
      setLoading(true);
      setError(null);

      try {
        const response = await fetch(`${baseUrl}/instruments/${token}`);

        if (!response.ok) {
          if (response.status === 404) {
            return null;
          }
          throw new Error(`Fetch failed: ${response.statusText}`);
        }

        const data: Instrument = await response.json();
        return data;
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Fetch failed';
        setError(errorMessage);
        return null;
      } finally {
        setLoading(false);
      }
    },
    [baseUrl]
  );

  const syncInstruments = useCallback(
    async (exchange?: string): Promise<boolean> => {
      setLoading(true);
      setError(null);

      try {
        const url = exchange
          ? `${baseUrl}/instruments/sync?exchange=${exchange}`
          : `${baseUrl}/instruments/sync`;

        const response = await fetch(url, { method: 'POST' });

        if (!response.ok) {
          throw new Error(`Sync failed: ${response.statusText}`);
        }

        return true;
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Sync failed';
        setError(errorMessage);
        return false;
      } finally {
        setLoading(false);
      }
    },
    [baseUrl]
  );

  return {
    searchInstruments,
    getInstrumentByToken,
    syncInstruments,
    loading,
    error,
  };
}
