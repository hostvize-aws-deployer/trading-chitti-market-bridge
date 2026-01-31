import React, { createContext, useContext, ReactNode } from 'react';

export interface MarketBridgeContextValue {
  baseUrl: string;
  wsUrl: string;
}

const MarketBridgeContext = createContext<MarketBridgeContextValue | null>(null);

export interface MarketBridgeProviderProps {
  baseUrl?: string;
  wsUrl?: string;
  children: ReactNode;
}

export function MarketBridgeProvider({
  baseUrl = 'http://localhost:6005',
  wsUrl,
  children,
}: MarketBridgeProviderProps) {
  // Auto-generate WebSocket URL from HTTP URL
  const defaultWsUrl = baseUrl.replace(/^http/, 'ws') + '/ws/market';

  const value: MarketBridgeContextValue = {
    baseUrl,
    wsUrl: wsUrl || defaultWsUrl,
  };

  return (
    <MarketBridgeContext.Provider value={value}>
      {children}
    </MarketBridgeContext.Provider>
  );
}

export function useMarketBridge() {
  const context = useContext(MarketBridgeContext);

  if (!context) {
    throw new Error('useMarketBridge must be used within MarketBridgeProvider');
  }

  return context;
}
