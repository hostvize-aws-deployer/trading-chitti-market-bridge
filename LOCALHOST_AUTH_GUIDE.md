# Zerodha Authentication on Localhost

## Quick Setup Guide

### Step 1: Create Kite Connect App

1. Go to [Kite Connect Developer Console](https://developers.kite.trade/apps)
2. Click "Create New App"
3. Fill in details:

```
App Name: Trading Chitti Local
Redirect URL: http://127.0.0.1:8080/auth/callback
Postback URL: (leave empty)
```

4. Save and note down:
   - **API Key**: (e.g., `abc123xyz`)
   - **API Secret**: (e.g., `secret456def`)

### Step 2: Configure Environment

Create or update `market-bridge/.env`:

```env
# Zerodha Credentials
ZERODHA_API_KEY=abc123xyz
ZERODHA_API_SECRET=secret456def

# Database
TRADING_CHITTI_PG_DSN=postgresql://hariprasath@localhost:6432/trading_chitti?sslmode=disable

# Service
PORT=8080
```

### Step 3: Start Service

```bash
cd /Users/hariprasath/trading-chitti/market-bridge
PORT=8080 ./bin/market-bridge
```

### Step 4: Generate Access Token (One-Time)

#### Option A: Browser Flow (Simple)

1. **Get Login URL**:
   ```bash
   curl http://localhost:8080/auth/login-url
   ```

   Response:
   ```json
   {
     "login_url": "https://kite.zerodha.com/connect/login?api_key=abc123xyz&v=3"
   }
   ```

2. **Open in Browser**:
   - Copy the login_url and open in browser
   - Login with your Zerodha credentials
   - Authorize the app

3. **You'll be redirected** to:
   ```
   http://127.0.0.1:8080/auth/callback?request_token=aBcDeF123&action=login&status=success
   ```

4. **Copy the request_token** from the URL bar

5. **Exchange for Access Token**:
   ```bash
   curl -X POST http://localhost:8080/auth/session \
     -H "Content-Type: application/json" \
     -d '{"request_token": "aBcDeF123"}'
   ```

   Response:
   ```json
   {
     "access_token": "your_access_token_here_very_long_string",
     "user_id": "AB1234",
     "refresh_token": "refresh_token_here",
     "expires_in": 86400
   }
   ```

6. **Save Access Token** to `.env`:
   ```env
   ZERODHA_ACCESS_TOKEN=your_access_token_here_very_long_string
   ```

7. **Restart Service** to load the new token:
   ```bash
   # Stop current service (Ctrl+C)
   PORT=8080 ./bin/market-bridge
   ```

#### Option B: Automatic Token Capture (Advanced)

The `/auth/callback` endpoint can automatically save the token:

1. Follow steps 1-3 from Option A
2. After redirect, the service will display the access token
3. Manually add it to `.env` file

### Step 5: Test Real Collector

```bash
# Create Zerodha live collector
curl -X POST 'http://localhost:8080/api/collectors' \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "zerodha-live",
    "type": "real",
    "symbols": ["RELIANCE", "TCS", "INFY"],
    "auto_start": true
  }'

# Check status (should show "running": true during market hours)
curl 'http://localhost:8080/api/collectors/zerodha-live' | jq .

# Wait 1 minute, then check data
sleep 60

# Query latest tick
curl 'http://localhost:8080/intraday/ticks/RELIANCE?limit=1' | jq .

# Should see real market data!
{
  "ticks": [
    {
      "symbol": "RELIANCE",
      "price": 2567.80,
      "quantity": 125,
      "tick_timestamp": "2026-01-31T14:25:43+05:30"
    }
  ]
}
```

---

## Troubleshooting

### Issue: "Redirect URL mismatch"

**Cause**: The redirect URL in your app settings doesn't match the one used in the login flow.

**Solution**:
- Ensure redirect URL in Kite Connect app is exactly: `http://127.0.0.1:8080/auth/callback`
- Use `127.0.0.1` (not `localhost` - they're treated differently)

### Issue: "Access token expired"

**Cause**: Access tokens expire daily.

**Solution**:
- Re-run the OAuth flow to get a new token
- Or implement automatic token refresh (service does this for you)

### Issue: "No ticks received" even though collector is running

**Causes**:
1. Market is closed (9:15 AM - 3:30 PM Mon-Fri only)
2. Invalid access token
3. Symbols not subscribed

**Solution**:
```bash
# Check collector status
curl 'http://localhost:8080/api/collectors/zerodha-live'

# Verify symbols are subscribed
# Should show symbols array with your stocks

# Test during market hours only
```

### Issue: "Invalid API credentials"

**Solution**:
1. Verify API key in `.env` matches Kite Connect app
2. Verify access token is fresh (regenerate if old)
3. Check API key is copied correctly (no extra spaces)

---

## Alternative: Testing Without Credentials

If you don't have Zerodha credentials yet, use **mock collectors**:

```bash
# Create mock collector (no credentials needed)
curl -X POST 'http://localhost:8080/api/collectors' \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "mock-test",
    "type": "mock",
    "symbols": ["RELIANCE", "TCS", "INFY"],
    "auto_start": true
  }'

# Works immediately - generates realistic random data
# Perfect for frontend development and testing
```

---

## Production Deployment

For production (not localhost):

1. **Use a real domain**: `https://yourdomain.com/auth/callback`
2. **Set up HTTPS**: Zerodha requires HTTPS for production
3. **Update redirect URL**: In Kite Connect app settings
4. **Implement token refresh**: Automatic daily token renewal

---

## Security Notes

⚠️ **Important**:

1. **Never commit `.env` to git** - Add to `.gitignore`
2. **Access tokens expire daily** - Implement refresh flow
3. **API secret is sensitive** - Never expose in logs
4. **Use HTTPS in production** - Required by Zerodha

---

## Quick Reference

**Zerodha Kite Connect**:
- Developer Portal: https://developers.kite.trade/
- API Docs: https://kite.trade/docs/connect/v3/
- Go SDK: https://github.com/zerodha/gokiteconnect

**Market Hours** (NSE):
- Monday - Friday
- 9:15 AM - 3:30 PM IST
- No data outside these hours

**Rate Limits**:
- Max 3 requests/second per API key
- Max 3,000 symbols per WebSocket connection

---

Last Updated: 2026-01-31
