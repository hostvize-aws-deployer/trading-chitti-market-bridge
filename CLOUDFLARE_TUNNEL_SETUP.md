# Cloudflare Tunnel Setup for HTTPS

**Free HTTPS without SSL certificates or port forwarding!**

---

## Why Cloudflare Tunnel?

✅ **Free HTTPS** - Automatic SSL/TLS encryption
✅ **No Port Forwarding** - No need to open router ports
✅ **DDoS Protection** - Cloudflare's network shields your home IP
✅ **Custom Domain** - Use your own domain (optional)
✅ **Zero Configuration** - No reverse proxy setup needed

---

## Prerequisites

1. Free Cloudflare account: https://dash.cloudflare.com/sign-up
2. Optional: Custom domain (or use `*.cfargotunnel.com` subdomain)

---

## Installation Steps

### Step 1: Install Cloudflared (Tunnel Client)

```bash
# macOS (using Homebrew)
brew install cloudflare/cloudflare/cloudflared

# Verify installation
cloudflared --version
```

### Step 2: Authenticate with Cloudflare

```bash
cloudflared tunnel login
```

This opens browser - login to Cloudflare and authorize.

### Step 3: Create Tunnel

```bash
# Create tunnel named "trading-chitti"
cloudflared tunnel create trading-chitti

# Note the Tunnel ID (will be shown in output)
# Example: Created tunnel trading-chitti with id: abc123-def456-ghi789
```

### Step 4: Configure Tunnel

Create tunnel configuration file:

```bash
# Create config directory
mkdir -p ~/.cloudflared

# Create config file
cat > ~/.cloudflared/config.yml <<'EOF'
tunnel: trading-chitti
credentials-file: /Users/hariprasath/.cloudflared/abc123-def456-ghi789.json

ingress:
  # Core API
  - hostname: core.trading-chitti.com
    service: http://localhost:6001

  # Signal Service
  - hostname: signals.trading-chitti.com
    service: http://localhost:6002

  # Dashboard
  - hostname: dashboard.trading-chitti.com
    service: http://localhost:6003

  # Market-Bridge API
  - hostname: api.trading-chitti.com
    service: http://localhost:6005

  # Prometheus (Monitoring)
  - hostname: prometheus.trading-chitti.com
    service: http://localhost:6090

  # Grafana (Dashboards)
  - hostname: grafana.trading-chitti.com
    service: http://localhost:6091

  # Catch-all rule (required)
  - service: http_status:404
EOF
```

**Replace:**
- `abc123-def456-ghi789` with YOUR actual Tunnel ID
- `trading-chitti.com` with YOUR domain (or use `*.cfargotunnel.com`)

### Step 5: Configure DNS

**Option A: Using Your Own Domain (Recommended)**

1. Add domain to Cloudflare
2. Point nameservers to Cloudflare
3. Create DNS records:

```bash
# Create CNAME records for each subdomain (all 6000 series ports)
cloudflared tunnel route dns trading-chitti core.trading-chitti.com
cloudflared tunnel route dns trading-chitti signals.trading-chitti.com
cloudflared tunnel route dns trading-chitti dashboard.trading-chitti.com
cloudflared tunnel route dns trading-chitti api.trading-chitti.com
cloudflared tunnel route dns trading-chitti prometheus.trading-chitti.com
cloudflared tunnel route dns trading-chitti grafana.trading-chitti.com
```

**Option B: Using Cloudflare Tunnel Domain (Free)**

Use format: `<tunnel-id>.cfargotunnel.com`

```yaml
# Update config.yml to use tunnel domain:
ingress:
  - hostname: market-bridge-abc123.cfargotunnel.com
    service: http://localhost:6005
  # ... etc
```

### Step 6: Run Tunnel

```bash
# Test run (foreground)
cloudflared tunnel run trading-chitti

# Should see:
# Registered tunnel connection
# Each service now accessible via HTTPS!
```

### Step 7: Run as Background Service (Auto-start)

```bash
# Install as system service
sudo cloudflared service install

# Start service
sudo launchctl start com.cloudflare.cloudflared

# Check status
sudo launchctl list | grep cloudflared
```

---

## Access Your Services

**Before (HTTP only, Dynamic DNS):**
```
http://hari-fiberspot.tplinkdns.com:6005/health
```

**After (HTTPS, Cloudflare Tunnel):**
```
https://core.trading-chitti.com/health      (port 6001)
https://signals.trading-chitti.com/health   (port 6002)
https://dashboard.trading-chitti.com        (port 6003)
https://api.trading-chitti.com/health       (port 6005)
https://prometheus.trading-chitti.com       (port 6090)
https://grafana.trading-chitti.com          (port 6091)
```

No port numbers needed! ✨

---

## Security Benefits

1. **Home IP Hidden** - Cloudflare network shields your real IP
2. **DDoS Protection** - Automatic threat mitigation
3. **SSL/TLS Encryption** - All traffic encrypted
4. **No Open Ports** - Router firewall remains intact
5. **Rate Limiting** - Built-in protection

---

## Monitoring the Tunnel

```bash
# View tunnel info
cloudflared tunnel info trading-chitti

# View logs
tail -f ~/.cloudflared/tunnel.log

# Check connections
cloudflared tunnel list
```

---

## Troubleshooting

### Tunnel Won't Start

```bash
# Check credentials file exists
ls ~/.cloudflared/*.json

# Check config syntax
cloudflared tunnel ingress validate

# Test connectivity
cloudflared tunnel run --loglevel debug trading-chitti
```

### Service Not Accessible

1. Verify service is running locally:
   ```bash
   curl http://localhost:6005/health
   ```

2. Check tunnel status:
   ```bash
   cloudflared tunnel list
   # Should show "HEALTHY"
   ```

3. Verify DNS propagation:
   ```bash
   dig api.trading-chitti.com
   # Should show Cloudflare IPs
   ```

---

## Cost

**100% FREE** for:
- Unlimited bandwidth
- Multiple tunnels
- DDoS protection
- SSL certificates
- Up to 50 concurrent connections

**Paid plans** (optional):
- Cloudflare Access (identity-based authentication): $3/user/month
- Advanced DDoS protection
- Custom firewall rules

---

## Alternative: Quick Setup Without Custom Domain

If you don't have a domain, use this simpler config:

```yaml
tunnel: trading-chitti
credentials-file: /Users/hariprasath/.cloudflared/abc123.json

ingress:
  # Single hostname for all services
  - hostname: trading-abc123.cfargotunnel.com
    service: http://localhost:6005

  - service: http_status:404
```

Access via: `https://trading-abc123.cfargotunnel.com`

---

## Next Steps After Cloudflare Tunnel

1. ✅ Remove port forwarding from router (no longer needed!)
2. ✅ Update API clients to use HTTPS URLs
3. ✅ Add API authentication (see API_AUTH_SETUP.md)
4. ✅ Configure Cloudflare firewall rules (optional)

---

**Setup Time**: ~15 minutes
**Cost**: $0 (free tier)
**Benefit**: Production-grade HTTPS + security
