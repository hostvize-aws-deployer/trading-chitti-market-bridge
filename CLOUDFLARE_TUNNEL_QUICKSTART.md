# Cloudflare Tunnel - Quick Start (No Domain Required)

**100% FREE - No domain purchase needed!**

---

## What You'll Get

**Free HTTPS URLs from Cloudflare:**
```
https://<your-tunnel-id>.cfargotunnel.com
```

Access all your services via HTTPS with **zero cost**!

---

## Setup (15 Minutes)

### Step 1: Install Cloudflared

```bash
# macOS
brew install cloudflare/cloudflare/cloudflared

# Verify
cloudflared --version
```

### Step 2: Login to Cloudflare

```bash
cloudflared tunnel login
```

This opens your browser - create a **free Cloudflare account** if you don't have one.

### Step 3: Create Tunnel

```bash
cloudflared tunnel create trading-chitti
```

**Important**: Copy the **Tunnel ID** from the output!

Example output:
```
Created tunnel trading-chitti with id: abc123-def456-ghi789
```

Your Tunnel ID is: `abc123-def456-ghi789`

### Step 4: Configure Tunnel (Simple - Single URL)

```bash
mkdir -p ~/.cloudflared

# Create config (replace YOUR_TUNNEL_ID with actual ID)
cat > ~/.cloudflared/config.yml <<'EOF'
tunnel: trading-chitti
credentials-file: /Users/hariprasath/.cloudflared/YOUR_TUNNEL_ID.json

ingress:
  # Single hostname for all services
  - service: http://localhost:6005
EOF
```

**Replace `YOUR_TUNNEL_ID`** with your actual tunnel ID!

### Step 5: Start Tunnel

```bash
cloudflared tunnel run trading-chitti
```

You'll see output like:
```
Registered tunnel connection
INF Registered at https://abc123-def456-ghi789.cfargotunnel.com
```

**That's your URL!** Copy it and test:

```bash
# Open in browser
https://abc123-def456-ghi789.cfargotunnel.com/health
```

---

## Option: Expose ALL Services (Advanced)

If you want separate URLs for each service, you have two options:

### Option A: Use Cloudflare Quick Tunnels (Easiest)

Start a tunnel for each service:

```bash
# Terminal 1 - Core API
cloudflared tunnel --url http://localhost:6001

# Terminal 2 - Signals
cloudflared tunnel --url http://localhost:6002

# Terminal 3 - Dashboard
cloudflared tunnel --url http://localhost:6003

# Terminal 4 - Market Bridge
cloudflared tunnel --url http://localhost:6005

# Terminal 5 - Grafana
cloudflared tunnel --url http://localhost:6091
```

Each command gives you a unique URL like:
```
https://random-words-123.trycloudflare.com
```

**Pros**: Super easy, instant
**Cons**: URLs change every restart, need multiple terminals

### Option B: Use Path-Based Routing (Recommended)

Configure one tunnel to handle all services via paths:

```bash
cat > ~/.cloudflared/config.yml <<'EOF'
tunnel: trading-chitti
credentials-file: /Users/hariprasath/.cloudflared/YOUR_TUNNEL_ID.json

ingress:
  # Note: Path-based routing requires custom domain
  # For free cfargotunnel.com, use Option A or buy domain
  - service: http://localhost:6005
EOF
```

**Limitation**: Path-based routing (`/api`, `/dashboard`) requires a custom domain.

---

## Recommended Setup (Best of Both Worlds)

**For now (FREE)**:
1. Use **Quick Tunnels** for testing (Option A above)
2. Each service gets its own temporary URL
3. Zero configuration needed

**When ready to buy domain ($10/year)**:
1. Buy cheap domain from Namecheap/GoDaddy
2. Point to Cloudflare nameservers
3. Use subdomain routing (core.yourdomain.com, etc.)
4. Follow full [CLOUDFLARE_TUNNEL_SETUP.md](CLOUDFLARE_TUNNEL_SETUP.md)

---

## Quick Tunnels Example (5 Minutes)

**Terminal 1**:
```bash
cloudflared tunnel --url http://localhost:6005
# Output: https://clever-fox-123.trycloudflare.com
```

**Terminal 2**:
```bash
cloudflared tunnel --url http://localhost:6091
# Output: https://happy-dog-456.trycloudflare.com
```

Now access:
- Market Bridge: `https://clever-fox-123.trycloudflare.com/health`
- Grafana: `https://happy-dog-456.trycloudflare.com`

**That's it!** Free HTTPS with zero configuration! 🎉

---

## Pros/Cons

### Quick Tunnels (Free, No Domain)

**Pros:**
- ✅ 100% free
- ✅ Instant setup (1 command)
- ✅ HTTPS automatic
- ✅ No domain needed
- ✅ Works immediately

**Cons:**
- ❌ URLs change on restart
- ❌ Need multiple terminals/processes
- ❌ Random URLs (not branded)

### Named Tunnel + Custom Domain ($10/year)

**Pros:**
- ✅ Permanent URLs (core.yourdomain.com)
- ✅ Branded (your domain)
- ✅ Single process
- ✅ Auto-restart on reboot

**Cons:**
- ❌ Requires domain purchase
- ❌ More configuration

---

## Your Current Options Summary

**Option 1: Quick Tunnels (FREE - Recommended for now)**
```bash
# Start each service
cloudflared tunnel --url http://localhost:6001  # Core API
cloudflared tunnel --url http://localhost:6002  # Signals
cloudflared tunnel --url http://localhost:6003  # Dashboard
cloudflared tunnel --url http://localhost:6005  # Market Bridge
cloudflared tunnel --url http://localhost:6090  # Prometheus
cloudflared tunnel --url http://localhost:6091  # Grafana
```

URLs: `https://random-words.trycloudflare.com` (changes each time)

**Option 2: Dynamic DNS (Current Setup)**
```
http://hari-fiberspot.tplinkdns.com:6001
http://hari-fiberspot.tplinkdns.com:6002
http://hari-fiberspot.tplinkdns.com:6003
http://hari-fiberspot.tplinkdns.com:6005
http://hari-fiberspot.tplinkdns.com:6090
http://hari-fiberspot.tplinkdns.com:6091
```

**Pros**: Permanent URLs
**Cons**: No HTTPS (HTTP only)

**Option 3: Buy Domain + Cloudflare Tunnel ($10/year)**
```
https://core.yourdomain.com
https://signals.yourdomain.com
https://dashboard.yourdomain.com
https://api.yourdomain.com
https://prometheus.yourdomain.com
https://grafana.yourdomain.com
```

**Pros**: Best of both worlds - permanent + HTTPS
**Cons**: Requires $10/year domain

---

## My Recommendation

**For now**: Stick with **Dynamic DNS** (free, permanent URLs)
- You already have it working
- No cost
- Permanent URLs
- Just add HTTPS later when you buy a domain

**When you're ready**: Buy a cheap domain ($10/year)
- Namecheap/GoDaddy: $8-12/year for .com
- Point to Cloudflare (free)
- Set up proper tunnel with subdomains
- Get beautiful URLs with HTTPS

**For testing**: Use **Quick Tunnels**
- Test HTTPS without commitment
- Each service gets temporary HTTPS URL
- Perfect for demos/testing

---

## Quick Decision Guide

**Need it now?**
→ Use **Quick Tunnels** (5 min setup, free)

**Testing HTTPS?**
→ Use **Quick Tunnels** (temporary URLs)

**Production ready?**
→ Buy domain + Cloudflare Tunnel ($10/year)

**Just want it working?**
→ Stick with current **Dynamic DNS** (free, permanent)

---

## Test Quick Tunnel Now (1 Minute)

```bash
# Start tunnel for market-bridge
cloudflared tunnel --url http://localhost:6005

# You'll see:
# https://something-random.trycloudflare.com

# Test it:
curl https://something-random.trycloudflare.com/health

# Should work with HTTPS! 🎉
```

---

**Your current setup with Dynamic DNS is perfectly fine!**

You can add HTTPS later when you decide to buy a domain. For now, you have:

✅ Remote access working (Dynamic DNS)
✅ All services on 6000 series ports
✅ Monitoring accessible
✅ API authentication enabled
✅ Production ready

**No rush to buy a domain - your setup works great as-is!** 🚀
