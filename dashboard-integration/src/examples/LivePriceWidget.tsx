import React, { useEffect, useState } from 'react';
import { useWebSocket, useInstruments } from '../hooks';

export interface LivePriceWidgetProps {
  symbol: string;
  exchange: string;
}

export function LivePriceWidget({ symbol, exchange }: LivePriceWidgetProps) {
  const { connected, subscribe, unsubscribe, getTickByToken } = useWebSocket();
  const { getInstrumentByToken } = useInstruments();
  const [instrumentToken, setInstrumentToken] = useState<number | null>(null);
  const [price, setPrice] = useState<number | null>(null);
  const [change, setChange] = useState<number>(0);

  // Fetch instrument token on mount
  useEffect(() => {
    async function fetchInstrument() {
      const instruments = await fetch(
        `http://localhost:6005/instruments/search?q=${symbol}&limit=1`
      ).then((r) => r.json());

      if (instruments.instruments.length > 0) {
        const inst = instruments.instruments[0];
        setInstrumentToken(inst.instrument_token);
      }
    }

    fetchInstrument();
  }, [symbol]);

  // Subscribe to instrument when connected
  useEffect(() => {
    if (connected && instrumentToken) {
      subscribe([instrumentToken]);

      return () => {
        unsubscribe([instrumentToken]);
      };
    }
  }, [connected, instrumentToken, subscribe, unsubscribe]);

  // Update price from ticks
  useEffect(() => {
    if (instrumentToken) {
      const tick = getTickByToken(instrumentToken);
      if (tick) {
        const prevPrice = price || tick.ohlc.open;
        setPrice(tick.last_price);
        setChange(((tick.last_price - prevPrice) / prevPrice) * 100);
      }
    }
  }, [instrumentToken, getTickByToken]);

  const changeColor = change > 0 ? 'green' : change < 0 ? 'red' : 'gray';

  return (
    <div style={{ padding: '16px', border: '1px solid #ddd', borderRadius: '8px' }}>
      <div style={{ fontSize: '14px', color: '#666' }}>{symbol}</div>
      <div style={{ fontSize: '24px', fontWeight: 'bold', marginTop: '8px' }}>
        ₹{price?.toFixed(2) || '--'}
      </div>
      <div style={{ fontSize: '12px', color: changeColor, marginTop: '4px' }}>
        {change > 0 ? '+' : ''}
        {change.toFixed(2)}%
      </div>
      <div style={{ fontSize: '10px', color: '#999', marginTop: '4px' }}>
        {connected ? '🟢 Live' : '🔴 Disconnected'}
      </div>
    </div>
  );
}
