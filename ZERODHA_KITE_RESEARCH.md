# Zerodha Kite Connect API Research

**Date**: 2026-01-31
**Purpose**: Comprehensive analysis of Zerodha Kite Connect API, WebSocket streaming, and integration patterns for market-bridge

---

## Table of Contents

1. [Official Zerodha Repositories](#official-zerodha-repositories)
2. [Kite Connect API Overview](#kite-connect-api-overview)
3. [WebSocket Streaming Deep Dive](#websocket-streaming-deep-dive)
4. [GoKiteConnect v4 SDK](#gokiteconnect-v4-sdk)
5. [Historical Data API](#historical-data-api)
6. [Rate Limits & Best Practices](#rate-limits--best-practices)
7. [Community Tools & Resources](#community-tools--resources)
8. [Implementation Recommendations](#implementation-recommendations)

---

## Official Zerodha Repositories

Zerodha maintains **50+ public repositories** with multiple SDKs and tools:

### Trading SDKs (Multi-Language Support)

| SDK | Language | Stars | Repository |
|-----|----------|-------|------------|
| **gokiteconnect** | Go | 203 | [github.com/zerodha/gokiteconnect](https://github.com/zerodha/gokiteconnect) |
| **pykiteconnect** | Python | 1.2k | [github.com/zerodha/pykiteconnect](https://github.com/zerodha/pykiteconnect) |
| **kiteconnectjs** | TypeScript | 379 | [github.com/zerodha/kiteconnectjs](https://github.com/zerodha/kiteconnectjs) |
| **javakiteconnect** | Java | 222 | [github.com/zerodha/javakiteconnect](https://github.com/zerodha/javakiteconnect) |
| **dotnetkiteconnect** | C# | 92 | [github.com/zerodha/dotnetkiteconnect](https://github.com/zerodha/dotnetkiteconnect) |
| **cppkiteconnect** | C++ | 59 | [github.com/zerodha/cppkiteconnect](https://github.com/zerodha/cppkiteconnect) |
| **phpkiteconnect** | PHP | 46 | [github.com/zerodha/phpkiteconnect](https://github.com/zerodha/phpkiteconnect) |

**Recommendation**: ✅ **Use gokiteconnect v4** for our Go-based market-bridge (already integrated)

### Useful Infrastructure Tools

- **kite-mcp-server** (209⭐): Model Context Protocol integration for AI assistants - provides tools for portfolio management, trading operations, and market intelligence
- **dungbeetle** (1.1k⭐): Distributed job server for SQL queries (supports MySQL, Postgres, ClickHouse)
- **kaf-relay** (86⭐): Kafka topic replication tool
- **nomad-cluster-setup** (163⭐): Terraform modules for AWS Nomad deployment

---

## Kite Connect API Overview

**Official Documentation**: [kite.trade/docs/connect/v3/](https://kite.trade/docs/connect/v3/)

### Core Capabilities

- **Real-time Order Execution**: Equities, commodities, F&O, mutual funds
- **Portfolio Management**: Holdings, positions, margins
- **Live Market Data**: WebSocket streaming (tick-level data)
- **Historical Data**: OHLCV candles with multiple intervals
- **GTT Orders**: Good-Till-Triggered conditional orders
- **Webhooks**: Postbacks for order updates

### Authentication Flow

```
1. Get Login URL → kc.GetLoginURL()
2. User logs in via browser → Redirect with request_token
3. Generate Session → kc.GenerateSession(requestToken, apiSecret)
4. Set Access Token → kc.SetAccessToken(session.AccessToken)
5. Make Authenticated Calls
```

**Security Requirements**:
- API Key + API Secret credential pair
- Access token expires daily (must regenerate)
- Refresh token for seamless re-authentication
- TOTP 2FA required on Zerodha account

**Critical**: Never embed API secret in client-side apps - always use backend server

---

## WebSocket Streaming Deep Dive

**Endpoint**: `wss://ws.kite.trade`
**Auth**: Pass `api_key` and `access_token` as query parameters

### Connection Limits

- **Max 3000 instruments** per WebSocket connection
- **Max 3 WebSocket connections** per API key
- Heartbeat messages (1 byte) maintain inactive connections

### Streaming Modes

| Mode | Description | Data Size | Use Case |
|------|-------------|-----------|----------|
| **ltp** | Last Traded Price only | 8 bytes | Lightweight price monitoring |
| **quote** | OHLC + Volume (no depth) | 44 bytes | Standard quotes |
| **full** | Complete data + market depth | 184 bytes | Advanced order flow analysis |

### Subscription Format (JSON)

```json
{
  "a": "subscribe",
  "v": [408065, 884737, 738561]
}
```

**Actions**: `subscribe`, `unsubscribe`, `mode`

### Binary Message Structure

Messages are **binary-encoded** for efficiency:

```
Bytes 0-2:   Number of packets (SHORT)
Bytes 2-4:   First packet length (SHORT)
Bytes 4+:    Quote packet data
... (repeats for additional packets)
```

**Quote Packet Fields** (all prices in paise):
- Instrument token
- Last traded price, quantity, average traded price
- Volume traded
- Buy/Sell quantities
- OHLC (Open, High, Low, Close)
- Timestamp, Open Interest
- Exchange timestamp
- **Market Depth** (full mode only): 5 bid levels + 5 ask levels (120 bytes)

**Market Depth Entry**: 12 bytes each
- Quantity (int32)
- Price (int32)
- Order count (int16)
- 2-byte padding

### Text-Based Postbacks

Order updates, errors, and broker messages delivered as JSON:

```json
{
  "type": "order",
  "data": {
    "order_id": "...",
    "status": "COMPLETE",
    ...
  }
}
```

**Recommendation**: Use official client libraries (gokiteconnect) to handle binary parsing complexity

**Source**: [WebSocket Streaming Documentation](https://kite.trade/docs/connect/v3/websocket/)

---

## GoKiteConnect v4 SDK

**Latest Version**: v4.3.5 (Released: January 17, 2025)
**Package**: `github.com/zerodha/gokiteconnect/v4`
**License**: MIT

### Installation

```bash
go get github.com/zerodha/gokiteconnect/v4
```

### Core Client API

#### Order Management

```go
// Place order
order, err := kc.PlaceOrder(kiteconnect.VarietyRegular,
    kiteconnect.OrderParams{
        Exchange:        kiteconnect.ExchangeNSE,
        Tradingsymbol:   "RELIANCE",
        TransactionType: kiteconnect.TransactionTypeBuy,
        OrderType:       kiteconnect.OrderTypeMarket,
        Product:         kiteconnect.ProductMIS,
        Quantity:        100,
    })

// Modify order
kc.ModifyOrder(variety, orderID, orderParams)

// Cancel order
kc.CancelOrder(variety, orderID, nil)

// Get orders
orders, err := kc.GetOrders()
orderHistory, err := kc.GetOrderHistory(orderID)
```

#### Market Data

```go
// Get full quote (OHLC + depth)
quote, err := kc.GetQuote("NSE:RELIANCE", "NSE:TCS")

// Get LTP only (lightweight)
ltp, err := kc.GetLTP("NSE:RELIANCE")

// Get OHLC
ohlc, err := kc.GetOHLC("NSE:RELIANCE")

// Historical data
candles, err := kc.GetHistoricalData(
    instrumentToken,
    "5minute",           // interval
    fromDate, toDate,
    false,               // continuous
    false)               // OI
```

#### Portfolio & Account

```go
// User profile
profile, err := kc.GetUserProfile()

// Margins
margins, err := kc.GetUserMargins()
segmentMargins, err := kc.GetUserSegmentMargins("equity")

// Holdings & Positions
holdings, err := kc.GetHoldings()
positions, err := kc.GetPositions()

// Convert position (MIS → CNC)
kc.ConvertPosition(positionParams)
```

#### GTT Orders (Good-Till-Triggered)

```go
// Place GTT
gtt, err := kc.PlaceGTT(GTTParams{
    TriggerType: kiteconnect.GTTTypeSingle,
    Tradingsymbol: "RELIANCE",
    Exchange: kiteconnect.ExchangeNSE,
    TriggerValues: []float64{2500.0},
    LastPrice: 2450.0,
    Orders: []GTTOrder{{
        TransactionType: kiteconnect.TransactionTypeBuy,
        Quantity: 100,
        OrderType: kiteconnect.OrderTypeLimit,
        Price: 2500.0,
        Product: kiteconnect.ProductCNC,
    }},
})

// Modify/Delete GTT
kc.ModifyGTT(triggerID, gttParams)
kc.DeleteGTT(triggerID)

// Get GTTs
gtts, err := kc.GetGTTs()
```

#### Mutual Funds

```go
// Place MF order
mfOrder, err := kc.PlaceMFOrder(MFOrderParams{
    Tradingsymbol: "INF123456789",
    TransactionType: kiteconnect.TransactionTypeBuy,
    Amount: 10000.0,
})

// SIP management
kc.PlaceMFSIP(sipParams)
kc.ModifyMFSIP(sipID, modifyParams)
kc.CancelMFSIP(sipID)

// MF holdings
holdings, err := kc.GetMFHoldings()
```

**Source**: [GoKiteConnect v4 API Documentation](https://pkg.go.dev/github.com/zerodha/gokiteconnect/v4)

---

## GoKiteConnect Ticker (WebSocket)

**Package**: `github.com/zerodha/gokiteconnect/v4/ticker`

### Complete Implementation Pattern

```go
package main

import (
    "log"
    kiteconnect "github.com/zerodha/gokiteconnect/v4"
    kiteticker "github.com/zerodha/gokiteconnect/v4/ticker"
)

func main() {
    apiKey := "your_api_key"
    accessToken := "your_access_token"

    // Create ticker instance
    ticker := kiteticker.New(apiKey, accessToken)

    // Set auto-reconnect
    ticker.SetAutoReconnect(true)
    ticker.SetReconnectMaxRetries(10)
    ticker.SetReconnectMaxDelay(60 * time.Second)

    // OnConnect: Subscribe to instruments
    ticker.OnConnect(func() {
        log.Println("✅ WebSocket Connected")

        // Subscribe to instrument tokens
        tokens := []uint32{738561, 2885633}  // RELIANCE, TCS
        err := ticker.Subscribe(tokens)
        if err != nil {
            log.Printf("Subscribe error: %v", err)
        }

        // Set mode to FULL for complete data
        ticker.SetMode(kiteticker.ModeFull, tokens)
    })

    // OnTick: Process tick data
    ticker.OnTick(func(tick kiteconnect.Tick) {
        log.Printf("Tick: %s | LTP: %.2f | Volume: %d | OHLC: O=%.2f H=%.2f L=%.2f C=%.2f",
            tick.InstrumentToken,
            tick.LastPrice,
            tick.VolumeTraded,
            tick.OHLC.Open,
            tick.OHLC.High,
            tick.OHLC.Low,
            tick.OHLC.Close)

        // Access market depth (full mode only)
        if len(tick.Depth.Buy) > 0 {
            log.Printf("Best Bid: %.2f x %d",
                tick.Depth.Buy[0].Price,
                tick.Depth.Buy[0].Quantity)
        }
    })

    // OnError: Handle errors
    ticker.OnError(func(err error) {
        log.Printf("❌ Ticker error: %v", err)
    })

    // OnClose: Handle disconnection
    ticker.OnClose(func(code int, reason string) {
        log.Printf("⚠️  Connection closed: %d - %s", code, reason)
    })

    // OnReconnect: Track reconnection attempts
    ticker.OnReconnect(func(attempt int, delay time.Duration) {
        log.Printf("🔄 Reconnecting... attempt %d, delay %v", attempt, delay)
    })

    // OnOrderUpdate: Process order postbacks
    ticker.OnOrderUpdate(func(order kiteconnect.Order) {
        log.Printf("📋 Order Update: %s | Status: %s",
            order.OrderID,
            order.Status)
    })

    // Start serving (blocking)
    go ticker.Serve()

    // Keep main goroutine alive
    select {}
}
```

### Ticker API Reference

#### Modes

```go
const (
    ModeLTP   Mode = "ltp"    // Last traded price (8 bytes)
    ModeFull  Mode = "full"   // All fields + depth (184 bytes)
    ModeQuote Mode = "quote"  // Quote without depth (44 bytes)
)
```

#### Configuration Methods

```go
ticker.SetAccessToken(accessToken)
ticker.SetAutoReconnect(true)
ticker.SetReconnectMaxRetries(10)
ticker.SetReconnectMaxDelay(60 * time.Second)
ticker.SetConnectTimeout(30 * time.Second)
```

#### Subscription Management

```go
// Subscribe to instruments
ticker.Subscribe([]uint32{738561, 2885633})

// Unsubscribe
ticker.Unsubscribe([]uint32{738561})

// Set mode for specific instruments
ticker.SetMode(kiteticker.ModeFull, []uint32{738561})

// Resubscribe (after reconnection)
ticker.Resubscribe()
```

#### Connection Management

```go
// Start connection (blocking)
ticker.Serve()

// With context support
ctx := context.Background()
ticker.ServeWithContext(ctx)

// Graceful shutdown
ticker.Close()

// Stop all goroutines
ticker.Stop()
```

#### Callback Types

```go
OnConnect(f func())                              // Connection established
OnTick(f func(tick models.Tick))                 // Tick received
OnOrderUpdate(f func(order kiteconnect.Order))   // Order update
OnError(f func(err error))                       // Error occurred
OnClose(f func(code int, reason string))         // Connection closed
OnReconnect(f func(attempt int, delay time.Duration))  // Reconnecting
OnNoReconnect(f func(attempt int))               // Max retries exceeded
OnMessage(f func(messageType int, message []byte))  // Raw message
```

**Source**: [KiteTicker Package Documentation](https://pkg.go.dev/github.com/zerodha/gokiteconnect/v4/ticker)

---

## Historical Data API

**Endpoint**: `/instruments/historical/{instrument_token}/{interval}`

### Available Intervals

| Interval | Max Date Range | Use Case |
|----------|----------------|----------|
| `minute` | 60 days | Intraday scalping |
| `2minute` | 60 days | Short-term patterns |
| `3minute` | 100 days | Medium-term intraday |
| `5minute` | 100 days | Standard intraday |
| `10minute` | 100 days | Swing intraday |
| `15minute` | 200 days | Positional intraday |
| `30minute` | 200 days | EOD analysis |
| `60minute` / `hour` | 400 days | Daily charts |
| `2hour`, `3hour`, `4hour` | 400 days | Multi-day patterns |
| `day` | 2000 days (~5.5 years) | Long-term analysis |
| `week` | No limit | Weekly charts |

### API Example

```go
fromDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
toDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

candles, err := kc.GetHistoricalData(
    738561,        // RELIANCE instrument token
    "5minute",     // interval
    fromDate,
    toDate,
    false,         // continuous (for futures)
    false)         // OI (open interest)

// Response: []HistoricalData
type HistoricalData struct {
    Date   time.Time
    Open   float64
    High   float64
    Low    float64
    Close  float64
    Volume int
    OI     int  // For F&O only
}
```

### Data Fetch Strategy for 52-Day Analysis

To fetch 52 trading days (~2.5 months):

```go
// Fetch daily candles for 52 days
endDate := time.Now()
startDate := endDate.AddDate(0, 0, -75)  // Buffer for weekends/holidays

candles, err := kc.GetHistoricalData(
    instrumentToken,
    "day",
    startDate,
    endDate,
    false,
    false)

// Filter to exactly 52 trading days
if len(candles) > 52 {
    candles = candles[len(candles)-52:]
}
```

**Source**: [Historical Data Documentation](https://medium.com/@ganeshnagarvani/collecting-data-through-kites-historical-api-for-algorithmic-trading-9bf8ce425f45)

---

## Rate Limits & Best Practices

### API Rate Limits

| API Category | Rate Limit |
|--------------|------------|
| **Historical Data** | 3 requests/second |
| **General APIs** | 10 requests/second (per API key) |
| **Order Placement** | Special limits (not publicly disclosed) |
| **WebSocket Connections** | Max 3 per API key |
| **Instruments per WebSocket** | Max 3000 |

**No per-minute or per-day limits** except for order placement.

### Best Practices

#### 1. Authentication

- Store access token securely (database/environment variable)
- Implement automatic token refresh using refresh token
- Never log or expose API secret
- Handle token expiration gracefully

```go
// Check if token expired
_, err := kc.GetUserProfile()
if err != nil && strings.Contains(err.Error(), "TokenException") {
    // Regenerate session
    newSession, _ := kc.GenerateSession(requestToken, apiSecret)
    kc.SetAccessToken(newSession.AccessToken)
}
```

#### 2. WebSocket Management

- Use auto-reconnect feature
- Implement exponential backoff for retries
- Subscribe to instruments in batches (avoid hitting 3000 limit)
- Use appropriate mode (LTP for price monitoring, Full for analysis)

```go
ticker.SetAutoReconnect(true)
ticker.SetReconnectMaxRetries(10)
ticker.SetReconnectMaxDelay(60 * time.Second)
```

#### 3. Historical Data

- Respect 3 req/sec limit for historical data
- Implement rate limiting with token bucket algorithm
- Cache historical data locally to reduce API calls
- Batch requests for multiple symbols

```go
import "time"

rateLimiter := time.NewTicker(350 * time.Millisecond)  // ~3 req/sec
defer rateLimiter.Stop()

for _, symbol := range symbols {
    <-rateLimiter.C  // Wait for rate limiter
    candles, err := kc.GetHistoricalData(...)
    // Process candles
}
```

#### 4. Error Handling

```go
type KiteError struct {
    Code      int
    ErrorType string
    Message   string
}

// Handle specific errors
switch err.ErrorType {
case "TokenException":
    // Refresh token
case "NetworkException":
    // Retry with backoff
case "DataException":
    // Instrument not found
case "OrderException":
    // Invalid order parameters
}
```

#### 5. Order Management

- Always validate order parameters before placement
- Use `dry_run` mode for testing
- Implement order confirmation flow
- Track order status using WebSocket postbacks
- Store order history in database

**Sources**:
- [Rate Limits Discussion](https://kite.trade/forum/discussion/13397/rate-limits)
- [API Rate Limits Forum](https://kite.trade/forum/discussion/8577/api-rate-limits)

---

## Community Tools & Resources

### Open Source Trading Platforms

#### 1. **hjAlgos** - AI-Based Algorithmic Trading

**Repository**: [github.com/hemangjoshi37a/hjAlgos](https://github.com/hemangjoshi37a/hjAlgos)

**Features**:
- Transformer-based neural network for stock price predictions
- Real-time trade execution
- Backtesting capabilities
- Zerodha Kite integration

#### 2. **anandaanv/zerodha-algo-trading**

**Repository**: [github.com/anandaanv/zerodha-algo-trading](https://github.com/anandaanv/zerodha-algo-trading)

**Features**:
- Market data sync from Kite Connect
- Custom screeners in Kotlin
- OpenAI vision for chart analysis
- Algo trading library integration

#### 3. **AdityaPawade/Zerodha_Live_Automate_Trading**

**Repository**: [github.com/AdityaPawade/Zerodha_Live_Automate_Trading-_using_AI_ML_on_Indian_stock_market](https://github.com/AdityaPawade/Zerodha_Live_Automate_Trading-_using_AI_ML_on_Indian_stock_market)

**Features**:
- AI/ML-based trading
- Live bots with indicators
- Screeners and backtesters
- REST API + WebSocket integration

#### 4. **Technical Indicators Library**

**Repository**: [github.com/debanshur/algotrading](https://github.com/debanshur/algotrading)

**Includes**: ATR, EMA, MACD, RSI, SuperTrend, VWAP

### Trading Strategy Ideas

From community repositories:

- **Momentum Trading**: RSI + MACD crossover
- **Swing Trading**: SuperTrend + ATR-based stop-loss
- **Intraday**: VWAP + volume analysis
- **Breakout**: Price + volume confirmation
- **Mean Reversion**: Bollinger Bands bounce

**Source**: [GitHub Zerodha Topics](https://github.com/topics/zerodha)

---

## Implementation Recommendations for Market-Bridge

### 1. WebSocket Architecture (Already Implemented ✅)

Our current `internal/api/websocket.go` correctly uses:

```go
ticker := kiteticker.New(apiKey, accessToken)
ticker.OnConnect(onTickerConnect)
ticker.OnTick(onTick)
ticker.OnError(onTickerError)
ticker.OnClose(onTickerClose)
go ticker.Serve()
```

**Enhancements to Consider**:

- ✅ Add `SetAutoReconnect(true)` for resilience
- ✅ Implement `OnReconnect` callback for monitoring
- ✅ Add `OnOrderUpdate` for real-time order tracking
- ✅ Store subscriptions in database for persistence

### 2. Historical Data Caching

**Recommendation**: Cache historical data in PostgreSQL to reduce API calls

```sql
-- Add to schema.sql
CREATE TABLE IF NOT EXISTS trades.historical_cache (
    instrument_token BIGINT NOT NULL,
    interval TEXT NOT NULL,
    candle_date TIMESTAMPTZ NOT NULL,
    open NUMERIC(12,2),
    high NUMERIC(12,2),
    low NUMERIC(12,2),
    close NUMERIC(12,2),
    volume BIGINT,
    oi BIGINT,
    PRIMARY KEY (instrument_token, interval, candle_date)
);

CREATE INDEX idx_historical_cache_token_date
ON trades.historical_cache(instrument_token, candle_date DESC);
```

**Fetch Strategy**:

```go
// 1. Check cache first
cachedCandles := db.GetHistoricalFromCache(instrumentToken, interval, fromDate, toDate)

// 2. Identify gaps
missingRanges := findMissingRanges(cachedCandles, fromDate, toDate)

// 3. Fetch missing data from API (with rate limiting)
for _, gap := range missingRanges {
    newCandles, err := kc.GetHistoricalData(instrumentToken, interval, gap.Start, gap.End, false, false)
    db.StoreHistoricalCandles(newCandles)
}

// 4. Merge and return
allCandles := mergeCachedAndNew(cachedCandles, newCandles)
```

### 3. Instrument Token Management

**Problem**: Our code uses `symbol` (e.g., "RELIANCE"), but WebSocket requires `instrument_token` (uint32)

**Solution**: Maintain instrument mapping table

```sql
CREATE TABLE IF NOT EXISTS trades.instruments (
    instrument_token BIGINT PRIMARY KEY,
    exchange TEXT NOT NULL,
    tradingsymbol TEXT NOT NULL,
    name TEXT,
    segment TEXT,
    expiry DATE,
    strike NUMERIC(12,2),
    lot_size INTEGER,
    instrument_type TEXT,
    last_updated TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(exchange, tradingsymbol)
);

CREATE INDEX idx_instruments_symbol ON trades.instruments(tradingsymbol);
```

**Sync Script** (run daily):

```go
func SyncInstruments(kc *kiteconnect.Client, db *Database) error {
    instruments, err := kc.GetInstruments()
    if err != nil {
        return err
    }

    for _, inst := range instruments {
        db.UpsertInstrument(inst)
    }

    return nil
}
```

**Usage**:

```go
// Convert symbol to token
func (db *Database) GetInstrumentToken(exchange, symbol string) (uint32, error) {
    query := `SELECT instrument_token FROM trades.instruments
              WHERE exchange = $1 AND tradingsymbol = $2`
    var token uint32
    err := db.QueryRow(query, exchange, symbol).Scan(&token)
    return token, err
}
```

### 4. Multi-Broker Token Management

For daily token refresh automation:

```sql
-- Add to brokers.config table
ALTER TABLE brokers.config ADD COLUMN refresh_token TEXT;
ALTER TABLE brokers.config ADD COLUMN token_expires_at TIMESTAMPTZ;
```

**Auto-Refresh Logic**:

```go
func AutoRefreshTokens(db *Database) {
    // Run every hour
    ticker := time.NewTicker(1 * time.Hour)

    for range ticker.C {
        configs, _ := db.GetExpiringSoonBrokerConfigs()  // Expires in <6 hours

        for _, config := range configs {
            if config.RefreshToken != "" {
                newTokens, err := RefreshBrokerToken(config)
                if err == nil {
                    db.UpdateBrokerTokens(config.ID, newTokens)
                }
            }
        }
    }
}
```

### 5. Enhanced 52-Day Analyzer

**Current Implementation**: ✅ Already in `internal/analyzer/analyzer52d.go`

**Enhancements**:

1. **Add More Indicators**:
   - ATR (Average True Range) - for volatility-based stops
   - VWAP (Volume Weighted Average Price) - for intraday
   - SuperTrend - for trend identification
   - Fibonacci retracement levels

2. **Pattern Recognition**:
   - Head & Shoulders
   - Double Top/Bottom
   - Triangles, Flags, Pennants

3. **Volume Analysis**:
   - Unusual volume spikes
   - Volume profile
   - On-Balance Volume (OBV)

4. **Sentiment Integration**:
   - News sentiment from `news.articles`
   - Entity linking to symbols

### 6. Risk Management Enhancements

**Add Position Sizing**:

```go
type PositionSizer struct {
    AccountCapital float64
    MaxRiskPerTrade float64  // 2%
    MaxPositions int         // 5
}

func (ps *PositionSizer) CalculateQuantity(
    entryPrice float64,
    stopLoss float64,
) int {
    riskAmount := ps.AccountCapital * ps.MaxRiskPerTrade
    riskPerShare := entryPrice - stopLoss
    quantity := int(riskAmount / riskPerShare)

    // Adjust for lot size (if F&O)
    // quantity = (quantity / lotSize) * lotSize

    return quantity
}
```

### 7. Backtesting Integration

Leverage historical data for strategy validation:

```go
type Backtest struct {
    Strategy Strategy
    HistoricalData []Candle
    InitialCapital float64
}

func (bt *Backtest) Run() *BacktestResults {
    // Simulate strategy on historical data
    // Track P&L, drawdown, Sharpe ratio
}
```

### 8. WebSocket Dashboard Integration

**Frontend Connection** (React/TypeScript):

```typescript
const ws = new WebSocket('ws://localhost:6005/ws/market');

ws.onopen = () => {
  console.log('Connected to market-bridge');

  // Subscribe to instruments
  ws.send(JSON.stringify({
    action: 'subscribe',
    tokens: [738561, 2885633]  // RELIANCE, TCS
  }));
};

ws.onmessage = (event) => {
  const tick = JSON.parse(event.data);

  if (tick.type === 'tick') {
    updatePriceWidget(tick.instrument_token, tick.last_price);
    updateChart(tick);
  }
};
```

---

## Summary: What We Learned

### ✅ WebSocket Implementation is Correct

Our current implementation in `market-bridge/internal/api/websocket.go` follows best practices:

1. Uses official `gokiteconnect/v4/ticker` package
2. Implements callback-based event handling
3. Broadcasts ticks to all connected clients via hub
4. Supports subscription management

### 🔧 Recommended Improvements

| Priority | Enhancement | Impact |
|----------|-------------|--------|
| **HIGH** | Instrument token mapping table | Required for symbol→token conversion |
| **HIGH** | Historical data caching | Reduce API calls, faster analysis |
| **HIGH** | Auto-reconnect + retry logic | Production resilience |
| **MEDIUM** | Token auto-refresh mechanism | Reduce manual intervention |
| **MEDIUM** | Enhanced indicators (ATR, VWAP) | Better signal quality |
| **LOW** | Pattern recognition | Advanced analysis |

### 📚 Key Takeaways

1. **gokiteconnect v4** is production-ready and actively maintained (latest: Jan 2025)
2. **WebSocket binary format** is handled automatically by SDK - no manual parsing needed
3. **Rate limits** are generous (10 req/sec general, 3 req/sec historical)
4. **Historical data** supports all major intervals (minute to weekly)
5. **Community tools** provide great inspiration for strategies and patterns
6. **Multi-broker architecture** in market-bridge positions us well for future expansion

---

## Next Steps

### Phase 1: Essential Fixes (This Week)

1. Create instruments table and sync script
2. Add auto-reconnect to WebSocket ticker
3. Implement historical data caching

### Phase 2: Enhancements (Next Week)

1. Token auto-refresh mechanism
2. Enhanced indicators (ATR, VWAP, SuperTrend)
3. Pattern recognition basics

### Phase 3: Advanced Features (Next Month)

1. Backtesting engine integration
2. ML-based signal scoring
3. Multi-timeframe analysis
4. Real-time portfolio tracking

---

## Sources

- [Zerodha GitHub Organization](https://github.com/zerodha)
- [GoKiteConnect Repository](https://github.com/zerodha/gokiteconnect)
- [Kite Connect API Documentation](https://kite.trade/docs/connect/v3/)
- [WebSocket Streaming Docs](https://kite.trade/docs/connect/v3/websocket/)
- [GoKiteConnect v4 Package Docs](https://pkg.go.dev/github.com/zerodha/gokiteconnect/v4)
- [KiteTicker Package Docs](https://pkg.go.dev/github.com/zerodha/gokiteconnect/v4/ticker)
- [Historical Data Guide](https://medium.com/@ganeshnagarvani/collecting-data-through-kites-historical-api-for-algorithmic-trading-9bf8ce425f45)
- [Rate Limits Forum Discussion](https://kite.trade/forum/discussion/13397/rate-limits)
- [GitHub Zerodha Topics](https://github.com/topics/zerodha)
- [Community Trading Tools](https://github.com/hemangjoshi37a/hjAlgos)

---

**Document Version**: 1.0
**Last Updated**: 2026-01-31
**Maintainer**: Trading Chitti Team
