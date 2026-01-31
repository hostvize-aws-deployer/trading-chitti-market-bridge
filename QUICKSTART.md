# Market Bridge - Quick Start Guide

**Get started with market-bridge in 5 minutes!**

---

## Prerequisites

- ✅ Go 1.21+ installed
- ✅ PostgreSQL 14+ running on port 6432
- ✅ Zerodha Kite API credentials (or other supported broker)

---

## Step 1: Environment Setup

```bash
cd /Users/hariprasath/trading-chitti/market-bridge

# Copy environment template
cp .env.example .env

# Edit .env with your credentials
nano .env
```

**Required variables**:
```bash
TRADING_CHITTI_PG_DSN=postgresql://hariprasath@localhost:6432/trading_chitti
ZERODHA_API_KEY=your_api_key_here
ZERODHA_API_SECRET=your_api_secret_here
ZERODHA_ACCESS_TOKEN=your_access_token_here
PORT=6005
```

---

## Step 2: Database Initialization

```bash
export TRADING_CHITTI_PG_DSN="postgresql://hariprasath@localhost:6432/trading_chitti"

# Create database schema
psql -h localhost -p 6432 -U hariprasath -d trading_chitti -f schema.sql
```

**Expected output**:
```
✅ Market Bridge schema created successfully
   - Created schemas: brokers, trades
   - Instrument token mapping: trades.instruments
   - Historical data caching: trades.historical_cache
   - Token auto-refresh: brokers.config
```

---

## Step 3: Build & Run

```bash
# Build
go build -o market-bridge cmd/server/main.go

# Run
./market-bridge
```

**Expected logs**:
```
✅ Zerodha broker initialized
✅ WebSocket hub initialized and started
✅ Token refresh service started
🚀 Market Bridge API starting on port 6005
📊 Active Broker: zerodha
📈 Market Status: open
🔌 WebSocket: ws://localhost:6005/ws/market
```

---

## Step 4: Sync Instruments (One-Time Setup)

In a new terminal:

```bash
# Sync all NSE instruments (~10,000+)
curl -X POST http://localhost:6005/instruments/sync?exchange=NSE

# Or sync all exchanges (takes ~2 minutes)
curl -X POST http://localhost:6005/instruments/sync
```

**Progress logs** in server terminal:
```
🔄 Starting instrument sync...
📥 Fetched 50,234 instruments from broker
📊 Synced 10000/50234 instruments
✅ Instrument sync completed: 50,234 instruments synced
```

---

## Step 5: Verify Deployment

```bash
# Run verification script
./scripts/verify_deployment.sh
```

**Expected output**:
```
✓ Health check passed
✓ trades.instruments table exists
✓ Instrument search working
✓ Historical data API responding
✓ WebSocket endpoints available
```

---

## 🎯 Quick Feature Tests

### Test 1: Search Instruments

```bash
curl "http://localhost:6005/instruments/search?q=RELIANCE&limit=5"
```

**Response**:
```json
{
  "query": "RELIANCE",
  "count": 5,
  "instruments": [
    {
      "instrument_token": 738561,
      "tradingsymbol": "RELIANCE",
      "name": "Reliance Industries Ltd",
      "exchange": "NSE"
    }
  ]
}
```

### Test 2: Fetch Historical Data (with Caching)

```bash
# First call: Fetches from broker API
time curl "http://localhost:6005/historical/52day?exchange=NSE&symbol=RELIANCE"
# Output: ~2 seconds

# Second call: Returns from cache
time curl "http://localhost:6005/historical/52day?exchange=NSE&symbol=RELIANCE"
# Output: ~50ms (40x faster!)
```

### Test 3: WebSocket Real-Time Data

**Using wscat** (install: `npm install -g wscat`):
```bash
wscat -c ws://localhost:6005/ws/market
```

**Subscribe to instruments**:
```json
{"action": "subscribe", "tokens": [738561, 2885633]}
```

**Receive ticks**:
```json
{
  "type": "tick",
  "instrument_token": 738561,
  "last_price": 2567.80,
  "volume": 1234567,
  "ohlc": {"open": 2550, "high": 2580, "low": 2545, "close": 2567.80}
}
```

### Test 4: Analyze with 52-Day Data

```bash
curl -X POST http://localhost:6005/trade/analyze \
  -H "Content-Type: application/json" \
  -d '{"symbols": ["RELIANCE", "TCS", "INFY"]}'
```

**Response includes**:
- Trend analysis (STRONG_UPTREND, UPTREND, etc.)
- Technical indicators (RSI, MACD, SMA, EMA)
- Trading signals with confidence scores
- Entry/exit prices, stop-loss, take-profit

---

## 🔧 Optional Optimizations

### Cache Warming (Recommended)

Pre-fetch historical data for your watchlist:

```bash
curl -X POST http://localhost:6005/historical/warm-cache \
  -H "Content-Type: application/json" \
  -d '{
    "exchange": "NSE",
    "symbols": ["RELIANCE", "TCS", "INFY", "HDFCBANK", "ICICIBANK"],
    "interval": "day",
    "days": 365
  }'
```

Runs in background. Check server logs for progress.

### Auto-Sync Instruments on Startup

Add to `.env`:
```bash
SYNC_INSTRUMENTS_ON_START=true
```

Restart server - instruments sync automatically.

---

## 📊 API Endpoints Reference

### Health & Info
- `GET /` - Service info
- `GET /health` - Health check

### Authentication (Zerodha)
- `GET /auth/login-url` - Get login URL
- `POST /auth/session` - Generate session

### Instruments
- `GET /instruments/search?q=<query>` - Search instruments
- `GET /instruments/:token` - Get instrument by token
- `POST /instruments/sync` - Sync from broker

### Historical Data
- `POST /historical/` - Fetch with caching
- `GET /historical/52day?exchange=NSE&symbol=RELIANCE` - 52 trading days
- `POST /historical/warm-cache` - Pre-fetch bulk data

### Trading
- `POST /trade/analyze` - Analyze symbols (52-day analysis)
- `POST /trade/scan` - Scan and execute trades
- `POST /trade/order` - Place order
- `PUT /trade/order/:orderID` - Modify order
- `DELETE /trade/order/:orderID` - Cancel order

### WebSocket
- `ws://localhost:6005/ws/market` - Market data stream
- `ws://localhost:6005/ws/orders` - Order updates
- `ws://localhost:6005/ws/positions` - Position updates

---

## 🐛 Troubleshooting

### Issue: "No active broker configured"

**Solution**: Update `.env` with Zerodha credentials or create broker in database:

```sql
INSERT INTO brokers.config (broker_name, display_name, enabled, api_key, api_secret, access_token)
VALUES ('zerodha', 'Zerodha', true, 'your_key', 'your_secret', 'your_token');
```

### Issue: "Instrument token not found"

**Solution**: Sync instruments first:
```bash
curl -X POST http://localhost:6005/instruments/sync
```

### Issue: WebSocket not connecting

**Solution**: Check if API credentials are set:
```bash
echo $ZERODHA_API_KEY
echo $ZERODHA_ACCESS_TOKEN
```

### Issue: Historical data returns empty

**Solution**:
1. Ensure instruments are synced
2. Check if market is open (for real-time data)
3. Verify date range is valid

---

## 📚 Next Steps

1. **Read Full Documentation**:
   - [ENHANCEMENTS.md](ENHANCEMENTS.md) - Complete feature guide
   - [ZERODHA_KITE_RESEARCH.md](ZERODHA_KITE_RESEARCH.md) - API research
   - [README.md](README.md) - Project overview

2. **Integrate with Dashboard**:
   - Connect frontend to WebSocket endpoints
   - Use instrument search for autocomplete
   - Display 52-day analysis results

3. **Production Deployment**:
   - Set up systemd service
   - Configure reverse proxy (nginx)
   - Enable SSL/TLS
   - Set up monitoring (Prometheus/Grafana)

4. **Advanced Features** (Phase 2):
   - Enhanced indicators (ATR, VWAP, SuperTrend)
   - Pattern recognition
   - Backtesting engine
   - ML-based predictions

---

## 🆘 Support

- **Issues**: Check server logs in terminal
- **Database**: `psql $TRADING_CHITTI_PG_DSN`
- **API Test**: Use Postman or curl
- **WebSocket Test**: Use wscat or browser console

---

**Happy Trading! 🚀📈**

*Built with Go, PostgreSQL, and Zerodha Kite Connect v4*
