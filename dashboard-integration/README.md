# Market Bridge Dashboard Integration

React/TypeScript components and hooks for integrating with Market Bridge API.

## Installation

```bash
npm install
# or
yarn install
```

## Quick Start

```typescript
import { MarketBridgeProvider, useWebSocket, useInstruments } from './hooks';

function App() {
  return (
    <MarketBridgeProvider baseUrl="http://localhost:6005">
      <Dashboard />
    </MarketBridgeProvider>
  );
}

function Dashboard() {
  const { connected, subscribe, ticks } = useWebSocket();
  const { searchInstruments } = useInstruments();

  useEffect(() => {
    if (connected) {
      subscribe([738561]); // RELIANCE
    }
  }, [connected]);

  return (
    <div>
      <h1>Market Dashboard</h1>
      {ticks.map(tick => (
        <div key={tick.instrument_token}>
          {tick.instrument_token}: ₹{tick.last_price}
        </div>
      ))}
    </div>
  );
}
```

## Features

- ✅ WebSocket real-time data streaming
- ✅ Auto-reconnect with status tracking
- ✅ Instrument search with autocomplete
- ✅ Historical data fetching with caching
- ✅ Order management
- ✅ 52-day analysis integration
- ✅ TypeScript support
- ✅ React hooks

## API Reference

See individual component documentation in `/src` folder.
