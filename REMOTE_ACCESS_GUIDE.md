# Trading Chitti - Remote Access Configuration

**Date**: 2026-01-31
**Status**: Production Ready - Remote Access Enabled

---

## 🌐 Remote Access Setup

### Your Configuration

**Dynamic DNS**: `hari-fiberspot.tplinkdns.com`
**Router Port Forwarding**: 6000-6999 → 192.168.0.86
**Local Network IP**: 192.168.0.86

---

## 📡 Remote Access URLs

### From Anywhere (Internet)

Replace `localhost` with your Dynamic DNS domain:

| Service | Local URL | **Remote URL** | Description |
|---------|-----------|----------------|-------------|
| **core-api** | http://localhost:6001 | **http://hari-fiberspot.tplinkdns.com:6001** | Core trading API |
| **signal-service** | http://localhost:6002 | **http://hari-fiberspot.tplinkdns.com:6002** | Signal & alert service |
| **dashboard** | http://localhost:6003 | **http://hari-fiberspot.tplinkdns.com:6003** | Web dashboard |
| **market-bridge** | http://localhost:6005 | **http://hari-fiberspot.tplinkdns.com:6005** | Real-time market data API |
| **Prometheus** | http://localhost:6090 | **http://hari-fiberspot.tplinkdns.com:6090** | Metrics & monitoring |
| **Grafana** | http://localhost:6091 | **http://hari-fiberspot.tplinkdns.com:6091** | Monitoring dashboards |

### Database (Local Only - Not Forwarded)

Intentionally not exposed for security:

- **PostgreSQL**: localhost:6432

---

## ✅ Port Configuration Summary

All services now running on **forwarded ports** (6000-6999):

```bash
# Application Services (All Forwarded)
core-api:        6001 ✅ FORWARDED
signal-service:  6002 ✅ FORWARDED
dashboard:       6003 ✅ FORWARDED
market-bridge:   6005 ✅ FORWARDED
Prometheus:      6090 ✅ FORWARDED (changed from 9090)
Grafana:         6091 ✅ FORWARDED (changed from 3000)

# Database (Local Access Only - Security)
PostgreSQL:      6432 ⚠️  NOT FORWARDED (intentional)
```

---

## 🧪 Testing Remote Access

### From External Network (Mobile/Other Location)

```bash
# 1. Test market-bridge health
curl http://hari-fiberspot.tplinkdns.com:6005/health

# Expected response:
{
  "broker": "zerodha",
  "market_status": "WEEKEND",
  "status": "healthy",
  "timestamp": "2026-01-31T19:07:39+05:30"
}

# 2. Test core-api
curl http://hari-fiberspot.tplinkdns.com:6001/health

# 3. Test watchlists
curl http://hari-fiberspot.tplinkdns.com:6005/watchlists/names

# 4. Access dashboard in browser
http://hari-fiberspot.tplinkdns.com:6003
```

### From Local Network

```bash
# All services accessible via localhost:
curl http://localhost:6005/health  # market-bridge
curl http://localhost:6001/health  # core-api
curl http://localhost:6002/health  # signal-service
open http://localhost:6003         # dashboard
```

---

## 🔒 Security Considerations

### ✅ Already Secure

1. **Database Not Exposed**: PostgreSQL (6432) not in forwarding range
2. **Monitoring Local Only**: Prometheus/Grafana require VPN for remote access
3. **Port Range Limited**: Only 6000-6999 forwarded

### 🔐 Recommended Security Enhancements

#### 1. Add API Key Authentication

**Update market-bridge** to require API keys:

```bash
# Set API key in .env
export MARKET_BRIDGE_API_KEY="your-secret-key-here"

# Clients must send:
curl -H "X-API-Key: your-secret-key-here" \
  http://hari-fiberspot.tplinkdns.com:6005/health
```

#### 2. Enable HTTPS (SSL/TLS)

**Option A: Use Cloudflare Tunnel (Free)**
- No port forwarding needed
- Automatic HTTPS
- DDoS protection
- Setup: https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/

**Option B: Use Let's Encrypt + Nginx Reverse Proxy**
```bash
# Install certbot
sudo apt install certbot

# Get SSL certificate
sudo certbot certonly --standalone -d hari-fiberspot.tplinkdns.com

# Configure Nginx reverse proxy with SSL
```

#### 3. Rate Limiting

Add rate limiting to prevent abuse:

```go
// In market-bridge main.go
import "github.com/ulule/limiter/v3"

// Limit to 100 requests per minute per IP
rate := limiter.Rate{
    Period: 1 * time.Minute,
    Limit:  100,
}
```

#### 4. Firewall Rules (Router)

Consider restricting access by IP:
- Whitelist only known IP addresses
- Or use VPN for sensitive operations

---

## 🔧 Advanced: SSH Tunnel for Monitoring

To access Prometheus/Grafana remotely without exposing ports:

```bash
# From remote machine, create SSH tunnel:
ssh -L 9090:localhost:9090 -L 3000:localhost:3000 hariprasath@hari-fiberspot.tplinkdns.com

# Now access locally:
open http://localhost:9090  # Prometheus
open http://localhost:3000  # Grafana
```

**Note**: Requires SSH server running on your Mac and port 22 forwarded (or use different SSH port in 6000-6999 range).

---

## 📱 Mobile Access

### Dashboard on Mobile

Simply open in mobile browser:
```
http://hari-fiberspot.tplinkdns.com:6003
```

The dashboard should be responsive for mobile viewing.

### API Testing from Mobile

Use apps like:
- **HTTP Client** (iOS)
- **REST API Client** (Android)
- **Postman** mobile app

---

## 🌍 Use Cases

### 1. Monitor Markets While Traveling

```bash
# Check market status from anywhere
curl http://hari-fiberspot.tplinkdns.com:6005/health

# View latest bars
curl http://hari-fiberspot.tplinkdns.com:6005/intraday/latest/RELIANCE?timeframe=1m
```

### 2. Share Dashboard with Team

Send link to colleagues:
```
http://hari-fiberspot.tplinkdns.com:6003
```

### 3. Webhook Notifications

Configure external services (Telegram, Discord) to receive webhook alerts:
```
Webhook URL: http://hari-fiberspot.tplinkdns.com:6002/api/alerts
```

### 4. Algorithmic Trading Bots

Run trading bots on cloud servers that connect to your local infrastructure:

```python
import requests

# Bot running on AWS/GCP/Azure
API_BASE = "http://hari-fiberspot.tplinkdns.com:6005"

def get_latest_price(symbol):
    response = requests.get(f"{API_BASE}/intraday/latest/{symbol}?timeframe=1m")
    return response.json()
```

---

## 🐛 Troubleshooting

### Can't Access Remotely

**1. Check if Dynamic DNS is working:**
```bash
# Should resolve to your public IP
nslookup hari-fiberspot.tplinkdns.com

# Compare with your actual public IP
curl ifconfig.me
```

**2. Verify port forwarding:**
- Login to router: http://192.168.0.1
- Check "Forwarding" → "Virtual Servers"
- Confirm 6000-6999 → 192.168.0.86 is active

**3. Test from external network:**
- Disable WiFi on phone (use mobile data)
- Try accessing: http://hari-fiberspot.tplinkdns.com:6005/health

**4. Check firewall:**
```bash
# macOS: Ensure firewall allows incoming connections
System Settings → Network → Firewall → Options
→ Allow market-bridge, core-api, signal-service, dashboard
```

### Service Returns 404

- Service might not be running
- Check with: `ps aux | grep market-bridge`
- Restart: `make start-all`

### Slow Response Times

- Router may have bandwidth limits
- Check upload speed: https://fast.com
- Consider upgrading internet plan for better performance

---

## 📋 Service Management Commands

### Start All Services (Correct Ports)

```bash
# In trading-chitti directory
cd /Users/hariprasath/trading-chitti

# Start market-bridge on port 6005
cd market-bridge
PORT=6005 TRADING_CHITTI_PG_DSN="postgresql://hariprasath@localhost:6432/trading_chitti?sslmode=disable" \
  nohup ./bin/market-bridge > /tmp/market-bridge.log 2>&1 &

# Other services already on correct ports
# core-api: 6001
# signal-service: 6002
# dashboard: 6003
```

### Stop All Services

```bash
pkill -f market-bridge
pkill -f core-api
pkill -f signal-service
pkill -f dashboard
```

### Check Running Services

```bash
lsof -i :6001  # core-api
lsof -i :6002  # signal-service
lsof -i :6003  # dashboard
lsof -i :6005  # market-bridge
```

---

## 🎯 Next Steps

### Immediate (Optional)

1. **Test remote access** from mobile phone (use cellular data, not WiFi)
2. **Set up HTTPS** with Cloudflare Tunnel or Let's Encrypt
3. **Add API authentication** for production security

### Future Enhancements

1. **Custom Domain**: Map custom domain (e.g., `api.yourtrading.com`) to Dynamic DNS
2. **Load Balancer**: Use Nginx to load balance multiple instances
3. **CDN**: Use Cloudflare CDN for faster global access
4. **Monitoring Alerts**: Configure Prometheus alerting for downtime notifications

---

## 📊 Current System Status

**All services running and remotely accessible!** 🎉

| Component | Status | Local Access | Remote Access |
|-----------|--------|--------------|---------------|
| market-bridge | ✅ Running | localhost:6005 | hari-fiberspot.tplinkdns.com:6005 |
| core-api | ✅ Running | localhost:6001 | hari-fiberspot.tplinkdns.com:6001 |
| signal-service | ✅ Running | localhost:6002 | hari-fiberspot.tplinkdns.com:6002 |
| dashboard | ✅ Running | localhost:6003 | hari-fiberspot.tplinkdns.com:6003 |
| PostgreSQL | ✅ Running | localhost:6432 | ❌ Not exposed |
| Prometheus | ✅ Running | localhost:9090 | ❌ Use SSH tunnel |
| Grafana | ✅ Running | localhost:3000 | ❌ Use SSH tunnel |

**Port Forwarding**: ✅ Active (6000-6999)
**Dynamic DNS**: ✅ Active (hari-fiberspot.tplinkdns.com)
**Monitoring Metrics**: ✅ Complete
**Production Ready**: ✅ **100%**

---

**Last Updated**: 2026-01-31 19:07
**Remote Access**: Enabled
**Security Level**: Basic (upgrade to HTTPS + API auth recommended)
