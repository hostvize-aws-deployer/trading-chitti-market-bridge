# Intraday API - Quick Start Guide

## Server Started

The Intraday Data API is ready with 9 endpoints for real-time market data queries.

## Quick Test (Copy & Paste)

```bash
# Set base URL
export API_URL="http://localhost:6005"

# 1. Get 1-minute bars for RELIANCE
curl "${API_URL}/intraday/bars/RELIANCE?timeframe=1m&limit=10"

# 2. Get latest bar
curl "${API_URL}/intraday/latest/RELIANCE?timeframe=1m"

# 3. Get today's bars
curl "${API_URL}/intraday/today/RELIANCE?timeframe=5m"

# 4. Get intraday statistics
curl "${API_URL}/intraday/stats/RELIANCE?timeframe=1m"

# 5. Calculate VWAP
curl "${API_URL}/intraday/vwap/RELIANCE?timeframe=1m"

# 6. Get tick data
curl "${API_URL}/intraday/ticks/RELIANCE?limit=10"

# 7. Get order book snapshot
curl "${API_URL}/intraday/orderbook/RELIANCE"

# 8. Check for data gaps
curl "${API_URL}/intraday/gaps/RELIANCE?timeframe=1m&from=2024-01-30T09:15:00Z&to=2024-01-30T15:30:00Z"

# 9. Check data completeness
curl "${API_URL}/intraday/completeness/RELIANCE?timeframe=1m&from=2024-01-30T09:15:00Z&to=2024-01-30T15:30:00Z"
```

## Run All Tests

```bash
cd /Users/hariprasath/trading-chitti/market-bridge
./scripts/test_intraday_api.sh
```

## API Endpoints Summary

| Endpoint | Purpose | Key Parameters |
|----------|---------|----------------|
| `GET /intraday/bars/:symbol` | Historical bars | timeframe, from, to, limit |
| `GET /intraday/latest/:symbol` | Most recent bar | timeframe |
| `GET /intraday/today/:symbol` | Today's bars | timeframe |
| `GET /intraday/stats/:symbol` | Day statistics | timeframe |
| `GET /intraday/vwap/:symbol` | VWAP calculation | timeframe |
| `GET /intraday/ticks/:symbol` | Tick data | from, to, limit |
| `GET /intraday/orderbook/:symbol` | Order book | - |
| `GET /intraday/gaps/:symbol` | Missing data gaps | timeframe, from, to |
| `GET /intraday/completeness/:symbol` | Data quality | timeframe, from, to |

## Supported Timeframes

- `1m` - 1 minute (default)
- `5m` - 5 minutes
- `15m` - 15 minutes
- `1h` - 1 hour
- `day` - Daily

## Example Responses

### Get Bars
```json
{
  "symbol": "RELIANCE",
  "timeframe": "1m",
  "bars_count": 10,
  "bars": [
    {
      "symbol": "RELIANCE",
      "bar_timestamp": "2024-01-30T09:15:00Z",
      "open": 2450.50,
      "high": 2455.00,
      "low": 2448.00,
      "close": 2453.75,
      "volume": 125000,
      "vwap": 2452.10
    }
  ]
}
```

### Get Stats
```json
{
  "symbol": "RELIANCE",
  "stats": {
    "day_low": 2445.00,
    "day_high": 2465.50,
    "day_open": 2450.00,
    "current_price": 2463.25,
    "total_volume": 12500000,
    "bars_count": 375,
    "day_change": 13.25,
    "day_change_pct": 0.54
  }
}
```

### Get VWAP
```json
{
  "symbol": "RELIANCE",
  "timeframe": "1m",
  "date": "2024-01-30",
  "vwap": 2456.85
}
```

### Get Completeness
```json
{
  "symbol": "RELIANCE",
  "timeframe": "1m",
  "completeness": 98.67,
  "quality": "excellent"
}
```

## Documentation

- **Full API Docs**: `docs/INTRADAY_API_IMPLEMENTATION.md`
- **Status Report**: `INTRADAY_API_STATUS.md`
- **Test Script**: `scripts/test_intraday_api.sh`

## Database Connection

- **Host**: localhost:6432
- **Database**: trading_chitti
- **Schema**: md (market data)
- **Tables**: intraday_bars, tick_data, order_book

## Implementation Complete ✅

- 9/9 endpoints implemented
- 15 database functions
- 1,023 lines of code
- Comprehensive error handling
- Full documentation
