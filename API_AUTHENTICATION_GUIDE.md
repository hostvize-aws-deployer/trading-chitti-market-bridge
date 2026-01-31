# API Authentication Setup Guide

**Secure your trading platform APIs with API key authentication**

---

## Overview

API key authentication protects your trading platform from unauthorized access while remaining simple to implement and use.

**What's Protected:**
- ✅ market-bridge API (all endpoints except /health and /metrics)
- ✅ core-api (coming soon)
- ✅ signal-service (coming soon)

**Not Protected (Intentional):**
- `/health` - Health check endpoint
- `/metrics` - Prometheus metrics (protected by network isolation)

---

## Step 1: Generate API Key

```bash
cd /Users/hariprasath/trading-chitti/market-bridge

# Generate secure random API key
./scripts/generate_api_key.sh
```

**Output Example:**
```
🔐 Generating Secure API Key for Market-Bridge

✅ Generated API Key:
    kJ9mP2xN4vB7qR5wL8tY3hG6fD1cS9zA

Use this key to authenticate API requests.
```

**Security Notes:**
- Key is 43 characters (base64 URL-safe)
- Cryptographically secure random generation
- No special characters (easy to copy/paste)

---

## Step 2: Configure API Key

### Option A: Environment Variable (Recommended for Development)

```bash
export API_KEY="kJ9mP2xN4vB7qR5wL8tY3hG6fD1cS9zA"

# Start service
PORT=6005 ./bin/market-bridge
```

### Option B: .env File (Recommended for Production)

```bash
# Add to market-bridge/.env
echo 'API_KEY=kJ9mP2xN4vB7qR5wL8tY3hG6fD1cS9zA' >> .env

# Start service (reads .env automatically)
PORT=6005 ./bin/market-bridge
```

### Option C: System Service (Production Deployment)

```bash
# Add to systemd service file
Environment="API_KEY=kJ9mP2xN4vB7qR5wL8tY3hG6fD1cS9zA"
```

---

## Step 3: Enable Authentication in Code

The API key middleware is already implemented! Just need to add it to main.go:

```go
// In cmd/server/main.go, add after MetricsMiddleware:

// Add API key authentication (production)
if os.Getenv("API_KEY") != "" {
    router.Use(api.APIKeyMiddleware())
    log.Println("🔐 API key authentication enabled")
}
```

This enables authentication **only when API_KEY is set**, allowing:
- ✅ Development without keys (local testing)
- ✅ Production with keys (security)

---

## Step 4: Using the API with Authentication

### cURL Examples

```bash
# Method 1: X-API-Key header (recommended)
curl -H "X-API-Key: kJ9mP2xN4vB7qR5wL8tY3hG6fD1cS9zA" \
  http://localhost:6005/watchlists/names

# Method 2: Authorization Bearer token
curl -H "Authorization: Bearer kJ9mP2xN4vB7qR5wL8tY3hG6fD1cS9zA" \
  http://localhost:6005/watchlists/names

# Without API key (will fail if API_KEY is set)
curl http://localhost:6005/watchlists/names
# Response: {"error":"missing API key","message":"provide X-API-Key header"}
```

### JavaScript/Frontend

```javascript
// Fetch API
const API_KEY = "kJ9mP2xN4vB7qR5wL8tY3hG6fD1cS9zA";

fetch("http://localhost:6005/watchlists/names", {
  headers: {
    "X-API-Key": API_KEY
  }
})
.then(res => res.json())
.then(data => console.log(data));

// Axios
import axios from 'axios';

const api = axios.create({
  baseURL: "http://localhost:6005",
  headers: {
    "X-API-Key": API_KEY
  }
});

api.get("/watchlists/names")
  .then(response => console.log(response.data));
```

### Python

```python
import requests

API_KEY = "kJ9mP2xN4vB7qR5wL8tY3hG6fD1cS9zA"

headers = {"X-API-Key": API_KEY}

response = requests.get(
    "http://localhost:6005/watchlists/names",
    headers=headers
)

print(response.json())
```

### Go

```go
package main

import (
    "net/http"
    "io"
)

func main() {
    apiKey := "kJ9mP2xN4vB7qR5wL8tY3hG6fD1cS9zA"

    req, _ := http.NewRequest("GET", "http://localhost:6005/watchlists/names", nil)
    req.Header.Set("X-API-Key", apiKey)

    client := &http.Client{}
    resp, _ := client.Do(req)
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)
    println(string(body))
}
```

---

## Security Best Practices

### ✅ DO

1. **Use HTTPS in production** (Cloudflare Tunnel provides this)
2. **Rotate keys periodically** (every 90 days)
3. **Use different keys for different environments:**
   - Development: `DEV_KEY_xxx`
   - Staging: `STAGING_KEY_xxx`
   - Production: `PROD_KEY_xxx`
4. **Store keys in environment variables**, never in code
5. **Add keys to .gitignore** if using .env files
6. **Log authentication failures** for security monitoring

### ❌ DON'T

1. **Don't commit API keys to Git**
2. **Don't share production keys via email/Slack**
3. **Don't hardcode keys in frontend code** (use environment variables)
4. **Don't reuse the same key across multiple services**
5. **Don't log API keys** in application logs

---

## Advanced: Multiple API Keys

For different clients/users, extend the middleware:

```go
// In api_auth.go
var validKeys = map[string]string{
    "client1": "key1_xxxxxx",
    "client2": "key2_xxxxxx",
    "admin":   "admin_key_xxx",
}

func APIKeyMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        providedKey := c.GetHeader("X-API-Key")

        // Check if key exists and is valid
        for client, validKey := range validKeys {
            if compareKeys(validKey, providedKey) {
                c.Set("client_id", client)
                c.Next()
                return
            }
        }

        c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid API key"})
        c.Abort()
    }
}
```

---

## Advanced: Rate Limiting by API Key

Implement rate limiting per client:

```go
import "github.com/ulule/limiter/v3"
import "github.com/ulule/limiter/v3/drivers/store/memory"

var limiters = make(map[string]*limiter.Limiter)

func RateLimitMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        clientID, _ := c.Get("client_id")

        // Create limiter for this client if doesn't exist
        if _, exists := limiters[clientID.(string)]; !exists {
            rate := limiter.Rate{
                Period: 1 * time.Minute,
                Limit:  100, // 100 requests per minute
            }
            store := memory.NewStore()
            limiters[clientID.(string)] = limiter.New(store, rate)
        }

        // Check rate limit
        context := limiter.Context{
            Limit: limiters[clientID.(string)],
        }

        if _, err := context.Get(c, clientID.(string)); err != nil {
            c.JSON(http.StatusTooManyRequests, gin.H{
                "error": "rate limit exceeded"
            })
            c.Abort()
            return
        }

        c.Next()
    }
}
```

---

## Monitoring & Logging

### Track Authentication Events

```go
// In api_auth.go, add logging:

import "log"

func APIKeyMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // ... validation logic ...

        if !compareKeys(expectedKey, providedKey) {
            log.Printf("⚠️  Failed authentication from %s for %s",
                c.ClientIP(), c.Request.URL.Path)

            c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid API key"})
            c.Abort()
            return
        }

        log.Printf("✅ Authenticated request from %s for %s",
            c.ClientIP(), c.Request.URL.Path)

        c.Next()
    }
}
```

### Prometheus Metrics

Add authentication metrics:

```go
// In internal/metrics/metrics.go
var (
    AuthenticationAttempts = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "marketbridge_auth_attempts_total",
            Help: "Total authentication attempts",
        },
        []string{"status"}, // "success" or "failure"
    )
)

// In api_auth.go
if !compareKeys(expectedKey, providedKey) {
    metrics.AuthenticationAttempts.WithLabelValues("failure").Inc()
    // ...
} else {
    metrics.AuthenticationAttempts.WithLabelValues("success").Inc()
}
```

---

## Troubleshooting

### "Missing API key" error

**Cause**: API key not provided in request
**Solution**: Add `X-API-Key` header to all requests

```bash
curl -H "X-API-Key: YOUR_KEY" http://localhost:6005/health
```

### "Invalid API key" error

**Cause**: Wrong API key provided
**Solution**: Verify API key matches `API_KEY` environment variable

```bash
# Check configured key
echo $API_KEY

# Test with correct key
curl -H "X-API-Key: $API_KEY" http://localhost:6005/health
```

### Authentication works locally but not remotely

**Cause**: API key may contain special characters causing issues
**Solution**: Regenerate key using the provided script (no special chars)

### Want to disable authentication temporarily

```bash
# Unset API_KEY environment variable
unset API_KEY

# Restart service (authentication disabled)
PORT=6005 ./bin/market-bridge
```

---

## Migration Guide

### From No Authentication → With Authentication

**1. Current State (No auth):**
```bash
curl http://localhost:6005/health  # Works
```

**2. Add API key:**
```bash
./scripts/generate_api_key.sh
export API_KEY="generated_key_here"
```

**3. Restart service:**
```bash
pkill market-bridge
PORT=6005 ./bin/market-bridge
```

**4. Update clients:**
```bash
# Old (will fail):
curl http://localhost:6005/watchlists/names

# New (works):
curl -H "X-API-Key: generated_key_here" \
  http://localhost:6005/watchlists/names
```

**5. Update frontend:**
```javascript
// Add to all API calls
headers: {
  "X-API-Key": process.env.REACT_APP_API_KEY
}
```

---

## Production Deployment Checklist

- [ ] Generate strong API key (`./scripts/generate_api_key.sh`)
- [ ] Add to .env file (don't commit!)
- [ ] Add .env to .gitignore
- [ ] Update main.go to enable middleware
- [ ] Rebuild service
- [ ] Test authentication works
- [ ] Update frontend to include API key
- [ ] Enable HTTPS (Cloudflare Tunnel)
- [ ] Document key location securely
- [ ] Set calendar reminder to rotate key (90 days)
- [ ] Configure monitoring/alerts for auth failures

---

**Setup Time**: ~5 minutes
**Security Level**: Medium (upgrade to JWT/OAuth for multi-user)
**Recommended For**: Single-tenant deployments, API integrations
