# Market Bridge - Multi-Broker Trading System

A high-performance Go-based trading system with **WebSocket streaming**, **52-day historical analysis**, and **multi-broker support** (Zerodha, Angel One, Upstox, ICICI Direct, and more).

## 🚀 Features

- **Multi-Broker Architecture**: Pluggable broker system - easily add new brokers
- **Real-Time WebSocket Streaming**: Ultra-low latency market data and order updates
- **52-Day Analysis Engine**: Comprehensive technical analysis with signal generation
- **Production-Ready**: Written in Go for performance and concurrency
- **Dashboard Integration**: WebSocket API for real-time dashboard updates
- **Risk Management**: Automated position sizing, stop-loss, take-profit
- **PostgreSQL Storage**: All trades, analysis, and performance tracked in database

## 📊 Supported Brokers

| Broker | Status | SDK | Features |
|--------|---------|-----|----------|
| **Zerodha** | ✅ Active | gokiteconnect | WebSocket, Full API |
| Angel One | 🔜 Coming Soon | - | - |
| Upstox | 🔜 Coming Soon | - | - |
| ICICI Direct | 🔜 Coming Soon | - | - |

Adding a new broker is simple - just implement the `Broker` interface in `internal/broker/broker.go`.

## 🏗️ Architecture

```
market-bridge/
├── cmd/server/          # Main server entry point
├── internal/
│   ├── broker/          # Pluggable broker interface & implementations
│   │   ├── broker.go    # Broker interface
│   │   ├── zerodha.go   # Zerodha implementation
│   │   └── errors.go    # Common errors
│   ├── analyzer/        # 52-day analysis engine
│   │   └── analyzer52d.go
│   ├── api/             # REST & WebSocket API
│   │   ├── api.go       # REST endpoints
│   │   └── websocket.go # WebSocket handlers
│   └── database/        # PostgreSQL layer
│       └── database.go
├── schema.sql           # Database schema
├── go.mod               # Go dependencies
└── README.md            # This file
```

## 📦 Installation

### Prerequisites

1. **Go 1.21+**
2. **PostgreSQL 14+** (running on port 6432)
3. **Zerodha Kite Connect Account** (or other supported broker)

### Setup

```bash
# Clone repository
cd /Users/hariprasath/trading-chitti/market-bridge

# Install dependencies
go mod download

# Initialize database
export TRADING_CHITTI_PG_DSN="postgresql://user@localhost:6432/trading_chitti"
psql -h localhost -p 6432 -U hariprasath -d trading_chitti -f schema.sql

# Configure environment
cp .env.example .env
# Edit .env with your credentials

# Build
go build -o market-bridge cmd/server/main.go

# Run
./market-bridge
```

## 🔐 Authentication (Zerodha)

Zerodha requires daily authentication:

```bash
# 1. Get login URL
curl http://localhost:6005/auth/login-url

# 2. Visit URL, login, get request_token from redirect
# 3. Generate session
curl -X POST http://localhost:6005/auth/session \
  -H "Content-Type: application/json" \
  -d '{"request_token": "YOUR_REQUEST_TOKEN"}'

# 4. Save access_token to .env
# 5. Restart server
```

## 🌐 WebSocket API (Real-Time)

Market Bridge uses **WebSocket** for real-time streaming - much faster than REST polling!

### Connect to WebSocket

```javascript
const ws = new WebSocket('ws://localhost:6005/ws/market');

ws.onopen = () => {
  console.log('Connected to Market Bridge');
  
  // Subscribe to instruments
  ws.send(JSON.stringify({
    action: 'subscribe',
    tokens: [738561, 2885633]  // RELIANCE, TCS instrument tokens
  }));
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Real-time tick:', data);
  /*
  {
    type: 'tick',
    instrument_token: 738561,
    last_price: 2567.80,
    volume: 1234567,
    ohlc: { open: 2550, high: 2580, low: 2545, close: 2567.80 },
    timestamp: '2026-01-31T15:30:00Z'
  }
  */
};
```

### WebSocket Endpoints

- `ws://localhost:6005/ws/market` - Real-time market data ticks
- `ws://localhost:6005/ws/orders` - Live order updates
- `ws://localhost:6005/ws/positions` - Position changes

## 📡 REST API

### Health & Status

```bash
GET  /              # Service info
GET  /health        # Health check
GET  /market/status # Market open/closed status
```

### Authentication

```bash
GET  /auth/login-url        # Get broker login URL
POST /auth/session          # Generate session from request token
```

### Account

```bash
GET  /account/profile   # User profile
GET  /account/margins   # Available margins
GET  /account/positions # Current positions
GET  /account/holdings  # Long-term holdings
GET  /account/orders    # Orders for the day
```

### Market Data

```bash
POST /market/quote          # Get real-time quotes
POST /market/ltp            # Get last traded prices
GET  /market/instruments/:exchange  # Get all instruments
```

### Trading

```bash
POST /trade/analyze         # Analyze symbols & generate signals
POST /trade/scan            # Scan symbols and execute trades
POST /trade/order           # Place order
PUT  /trade/order/:orderID  # Modify order
DELETE /trade/order/:orderID  # Cancel order
POST /trade/positions/close-all  # Close all positions
```

### Broker Management

```bash
GET  /brokers/            # List all configured brokers
POST /brokers/            # Add new broker
PUT  /brokers/:id         # Update broker config
DELETE /brokers/:id       # Delete broker
POST /brokers/:id/activate  # Activate broker
```

## 📈 52-Day Analysis

The analyzer examines 52 trading days (~2.5 months) and generates:

- **Trend Analysis**: Linear regression, R², classification (STRONG_UPTREND, UPTREND, SIDEWAYS, etc.)
- **Volatility Metrics**: Annualized volatility, ATR, classification
- **Volume Analysis**: Average, recent, trend percentage
- **Support/Resistance**: Top 3 levels each
- **Technical Indicators**: SMA, EMA, RSI, MACD, Bollinger Bands
- **Risk Metrics**: Sharpe ratio, max drawdown, win rate
- **Trading Signals**: Multiple strategies with confidence scores

### Signal Strategies

1. **RSI Oversold/Overbought**: Buy RSI < 30, Sell RSI > 70
2. **SMA Crossover**: Price crosses moving average
3. **Trend + Volume**: Strong trend with increasing volume
4. **Bollinger Band Bounce**: Mean reversion plays

### Example Analysis Request

```bash
curl -X POST http://localhost:6005/trade/analyze \
  -H "Content-Type: application/json" \
  -d '{"symbols": ["RELIANCE", "TCS", "INFY"]}'
```

### Example Response

```json
{
  "analyzed": 3,
  "total_signals": 8,
  "results": [
    {
      "symbol": "RELIANCE",
      "period_days": 52,
      "trend": {
        "direction": "STRONG_UPTREND",
        "slope": 0.75,
        "r_squared": 0.82,
        "strength": 0.91
      },
      "indicators": {
        "rsi": 65.4,
        "macd": 12.5,
        "sma_20": 2550.0,
        "bb_position": "NEUTRAL"
      },
      "signals": [
        {
          "type": "BUY",
          "strategy": "TREND_VOLUME",
          "confidence": 0.85,
          "entry_price": 2567.80,
          "stop_loss": 2490.00,
          "take_profit": 2825.00,
          "reason": "STRONG_UPTREND with increasing volume"
        }
      ]
    }
  ]
}
```

## 🎯 Trading Example

### Place Order

```bash
curl -X POST http://localhost:6005/trade/order \
  -H "Content-Type: application/json" \
  -d '{
    "symbol": "RELIANCE",
    "exchange": "NSE",
    "transaction_type": "BUY",
    "order_type": "MARKET",
    "product": "MIS",
    "quantity": 100
  }'
```

### Automated Trading

```bash
curl -X POST http://localhost:6005/trade/scan \
  -H "Content-Type: application/json" \
  -d '{
    "symbols": ["RELIANCE", "TCS", "INFY", "HDFCBANK"],
    "dry_run": true
  }'
```

## 🗄️ Database Schema

All data is stored in PostgreSQL with two schemas:

- **brokers**: Multi-broker configuration and credentials
- **trades**: Analysis results, signals, executions, performance

View trade history:

```sql
SELECT symbol, action, quantity, entry_price, pnl, confidence
FROM trades.executions
ORDER BY executed_at DESC
LIMIT 20;
```

View generated signals:

```sql
SELECT symbol, signal_type, strategy, confidence, entry_price
FROM trades.signals
WHERE confidence >= 0.75
ORDER BY generated_at DESC
LIMIT 20;
```

## 🔧 Configuration

All settings in `.env`:

```bash
# Database
TRADING_CHITTI_PG_DSN=postgresql://user@localhost:6432/trading_chitti

# Zerodha
ZERODHA_API_KEY=your_api_key
ZERODHA_API_SECRET=your_api_secret
ZERODHA_ACCESS_TOKEN=your_access_token  # Refresh daily

# Server
PORT=6005

# Trading
MAX_POSITIONS=5
MAX_RISK_PER_TRADE=2.0
MIN_CONFIDENCE=0.75
DRY_RUN=true  # Set false for live trading
```

## 🚦 Running in Production

```bash
# Build optimized binary
go build -ldflags="-s -w" -o market-bridge cmd/server/main.go

# Run as systemd service
sudo systemctl start market-bridge
sudo systemctl enable market-bridge

# Check logs
journalctl -u market-bridge -f
```

## 🔌 Dashboard Integration

Connect your dashboard to Market Bridge:

```typescript
// WebSocket connection
const ws = new WebSocket('ws://localhost:6005/ws/market');

// REST API calls
const response = await fetch('http://localhost:6005/account/positions');
const positions = await response.json();
```

## 🧪 Testing

```bash
# Run tests
go test ./...

# Test with dry run
DRY_RUN=true ./market-bridge

# Check WebSocket connection
websocat ws://localhost:6005/ws/market
```

## 📚 Adding a New Broker

1. Create `internal/broker/yourbroker.go`
2. Implement the `Broker` interface
3. Add to factory in `broker.go`:

```go
func NewBroker(config *BrokerConfig) (Broker, error) {
    switch config.BrokerName {
    case "zerodha":
        return NewZerodhaBroker(config)
    case "yourbroker":  // Add here
        return NewYourBroker(config)
    default:
        return nil, ErrBrokerNotSupported
    }
}
```

4. Configure in database:

```sql
INSERT INTO brokers.config (broker_name, display_name, enabled, api_key, api_secret)
VALUES ('yourbroker', 'Your Broker', true, 'key', 'secret');
```

## ⚠️ Disclaimer

**This software is for educational purposes only. Trading involves substantial risk of loss.**

- Always test in DRY RUN mode first
- Start with small capital
- Use proper risk management
- Comply with all applicable laws

## 📄 License

Internal use only - Trading Chitti Project

## 🤝 Support

- GitHub Issues: [trading-chitti/market-bridge](https://github.com/trading-chitti/market-bridge/issues)
- Check logs: `logs/` directory
- Database queries: `trades.*` tables

---

**Built with Go 🚀 | WebSocket for Speed ⚡ | Multi-Broker Architecture 🔌**
