# Data Backfill Utility

This utility fills historical intraday data gaps in the database by fetching data from the broker API.

## Usage

### Basic Usage

```bash
# Backfill specific symbols
./backfill -symbols RELIANCE,TCS,INFY -from 2024-01-01 -to 2024-12-31

# Backfill entire watchlist
./backfill -watchlist NIFTY50 -from 2024-01-01 -to 2024-12-31

# Backfill with specific timeframe
./backfill -symbols RELIANCE -from 2024-01-01 -timeframe day
```

### Flags

| Flag | Description | Default | Required |
|------|-------------|---------|----------|
| `-symbols` | Comma-separated list of symbols | - | Yes (or `-watchlist`) |
| `-watchlist` | Watchlist name (NIFTY50, BANKNIFTY, etc.) | - | Yes (or `-symbols`) |
| `-from` | Start date (YYYY-MM-DD) | - | Yes |
| `-to` | End date (YYYY-MM-DD) | Today | No |
| `-timeframe` | Data timeframe | `day` | No |
| `-concurrent` | Number of concurrent requests | `5` | No |
| `-dry-run` | Test run without inserting data | `false` | No |

### Timeframe Options

- `minute` - 1-minute candles
- `5minute` - 5-minute candles
- `15minute` - 15-minute candles
- `day` - Daily candles

### Environment Variables

```bash
export ZERODHA_API_KEY="your_api_key"
export ZERODHA_ACCESS_TOKEN="your_access_token"
export DB_PATH="./market_bridge.db"  # Optional
```

### Examples

#### 1. Backfill NIFTY50 for Last Year

```bash
./backfill \
  -watchlist NIFTY50 \
  -from 2023-01-01 \
  -to 2023-12-31 \
  -timeframe day \
  -concurrent 10
```

#### 2. Backfill Specific Stocks (Dry Run)

```bash
./backfill \
  -symbols RELIANCE,TCS,INFY,HDFCBANK \
  -from 2024-01-01 \
  -dry-run
```

#### 3. Backfill Intraday Data

```bash
./backfill \
  -watchlist BANKNIFTY \
  -from 2024-01-30 \
  -to 2024-01-30 \
  -timeframe 5minute \
  -concurrent 3
```

#### 4. Fill Gaps for All Sectors

```bash
# IT sector
./backfill -watchlist IT -from 2024-01-01

# Pharma sector
./backfill -watchlist PHARMA -from 2024-01-01

# Auto sector
./backfill -watchlist AUTO -from 2024-01-01
```

### Building

```bash
cd cmd/backfill
go build -o backfill main.go
```

### Output

The utility provides real-time progress updates:

```
📊 Backfill Configuration
   Symbols: 50
   From: 2024-01-01
   To: 2024-12-31
   Timeframe: day
   Concurrent: 5
   Dry Run: false

🔄 Fetching data for RELIANCE (token: 738561)...
✅ RELIANCE: 245 bars inserted
🔄 Fetching data for TCS (token: 2953217)...
✅ TCS: 245 bars inserted
...

📈 Backfill Summary
   Total Symbols: 50
   Successful: 48
   Failed: 2
   Total Bars: 12,250
   Duration: 2m15s
```

### Best Practices

1. **Test with Dry Run**: Always test with `-dry-run` first
2. **Limit Concurrency**: Don't overwhelm the API (max 10 concurrent)
3. **Backfill in Chunks**: For large date ranges, backfill in monthly chunks
4. **Check API Limits**: Be aware of broker API rate limits
5. **Verify Data**: Check data completeness after backfill

### Troubleshooting

**Error: "instrument token not found"**
- Solution: Sync instruments first using `/instruments/sync` API

**Error: "rate limit exceeded"**
- Solution: Reduce `-concurrent` value or add delays

**Error: "session expired"**
- Solution: Refresh `ZERODHA_ACCESS_TOKEN` environment variable

### Integration with Cron

Schedule daily backfills:

```bash
# Add to crontab
0 18 * * 1-5 cd /path/to/market-bridge && ./backfill -watchlist NIFTY50 -from $(date -d "1 day ago" +%Y-%m-%d)
```

### Future Enhancements

- [ ] Automatic gap detection and filling
- [ ] Progress bar for large backfills
- [ ] Resume from last checkpoint
- [ ] Email/webhook notifications on completion
- [ ] Parallel exchange support (NSE + BSE)
- [ ] Data validation and deduplication
