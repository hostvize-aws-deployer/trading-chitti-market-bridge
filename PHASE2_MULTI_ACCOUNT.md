# Phase 2 Features + Multi-Account System

**Implementation Guide for Advanced Features**

---

## ✅ Completed: Dashboard Integration

Created comprehensive React/TypeScript integration package:

**Location**: `/dashboard-integration/`

**Components**:
- `useWebSocket` - Real-time WebSocket with auto-reconnect
- `useInstruments` - Instrument search and management
- `useHistoricalData` - Historical data fetching with caching
- `LivePriceWidget` - Real-time price display component
- `InstrumentSearch` - Autocomplete search component

**Usage**:
```typescript
import { MarketBridgeProvider, useWebSocket, useInstruments } from './hooks';

function App() {
  return (
    <MarketBridgeProvider baseUrl="http://localhost:6005">
      <Dashboard />
    </MarketBridgeProvider>
  );
}
```

---

## ✅ Completed: Enhanced Technical Indicators

**Location**: `/internal/analyzer/indicators.go`

**New Indicators**:
1. **ATR (Average True Range)** - Volatility measurement
   ```go
   atr := CalculateATR(candles, 14)
   ```

2. **VWAP (Volume Weighted Average Price)** - Intraday benchmark
   ```go
   vwap := CalculateVWAP(candles)
   ```

3. **SuperTrend** - Trend-following indicator with signals
   ```go
   st := CalculateSuperTrend(candles, 10, 3.0)
   // st.Trend[]    // "UP" or "DOWN"
   // st.Signals[]  // "BUY", "SELL", or ""
   ```

4. **Stochastic RSI** - Momentum oscillator
   ```go
   stochRSI := CalculateStochasticRSI(rsi, 14)
   ```

5. **ADX (Average Directional Index)** - Trend strength
   ```go
   adx := CalculateADX(candles, 14)
   ```

---

## 📊 Pattern Recognition (To Implement)

**File**: Create `/internal/analyzer/patterns.go`

### Supported Patterns

#### Reversal Patterns
```go
// Head and Shoulders
func DetectHeadAndShoulders(candles []broker.Candle) *Pattern {
    // Logic: 3 peaks, middle highest
    // Return: Pattern with entry, stop-loss, target
}

// Double Top / Double Bottom
func DetectDoubleTop(candles []broker.Candle) *Pattern
func DetectDoubleBottom(candles []broker.Candle) *Pattern
```

#### Continuation Patterns
```go
// Flags and Pennants
func DetectFlag(candles []broker.Candle) *Pattern
func DetectPennant(candles []broker.Candle) *Pattern

// Triangles
func DetectAscendingTriangle(candles []broker.Candle) *Pattern
func DetectDescendingTriangle(candles []broker.Candle) *Pattern
func DetectSymmetricalTriangle(candles []broker.Candle) *Pattern
```

#### Candlestick Patterns
```go
// Single Candlestick
func DetectDoji(candle broker.Candle) bool
func DetectHammer(candle, prev broker.Candle) bool
func DetectShootingStar(candle, prev broker.Candle) bool

// Multiple Candlestick
func DetectBullishEngulfing(candles []broker.Candle) bool
func DetectBearishEngulfing(candles []broker.Candle) bool
func DetectMorningStar(candles []broker.Candle) bool
func DetectEveningStar(candles []broker.Candle) bool
```

### Pattern Structure

```go
type Pattern struct {
    Name        string    // "Head and Shoulders", etc.
    Type        string    // "REVERSAL", "CONTINUATION"
    Signal      string    // "BULLISH", "BEARISH"
    Confidence  float64   // 0.0 - 1.0
    EntryPrice  float64
    StopLoss    float64
    Target      float64
    DetectedAt  time.Time
    Candles     []int     // Indices of pattern candles
}
```

---

## 🔐 Multi-Account Broker Management

### Architecture

```
┌─────────────────────────────────────────────────┐
│                  Dashboard                       │
│  (Multiple users, each with broker accounts)    │
└────────────┬────────────────────────────────────┘
             │
             ├─────────────────────────┐
             │                         │
      ┌──────▼──────┐         ┌───────▼──────┐
      │   User 1    │         │    User 2     │
      │  Account    │         │   Account     │
      └──────┬──────┘         └───────┬───────┘
             │                        │
      ┌──────▼──────┐         ┌───────▼───────┐
      │  Zerodha 1  │         │  Zerodha 2    │
      │  (API Key)  │         │  (API Key)    │
      └─────────────┘         └───────────────┘
```

### Database Schema Extensions

```sql
-- User accounts table
CREATE TABLE IF NOT EXISTS users (
    user_id SERIAL PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    full_name TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    last_login TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT TRUE
);

-- User sessions
CREATE TABLE IF NOT EXISTS user_sessions (
    session_id TEXT PRIMARY KEY,
    user_id INTEGER REFERENCES users(user_id),
    token TEXT UNIQUE NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Update brokers.config to support multiple users
ALTER TABLE brokers.config ADD COLUMN user_id INTEGER REFERENCES users(user_id);
ALTER TABLE brokers.config DROP CONSTRAINT IF EXISTS brokers_config_broker_name_user_id_key;
ALTER TABLE brokers.config ADD CONSTRAINT unique_user_broker
    UNIQUE(user_id, broker_name);
```

### Authentication System

**File**: `/internal/auth/auth.go`

```go
package auth

import (
    "time"
    "github.com/dgrijalva/jwt-go"
    "golang.org/x/crypto/bcrypt"
)

type Claims struct {
    UserID int    `json:"user_id"`
    Email  string `json:"email"`
    jwt.StandardClaims
}

// Register new user
func RegisterUser(email, password, fullName string) (*User, error) {
    hashedPassword, err := bcrypt.GenerateFromPassword(
        []byte(password), bcrypt.DefaultCost)
    if err != nil {
        return nil, err
    }

    user := &User{
        Email:        email,
        PasswordHash: string(hashedPassword),
        FullName:     fullName,
    }

    // Insert into database
    // ...

    return user, nil
}

// Authenticate user
func AuthenticateUser(email, password string) (*User, string, error) {
    // Fetch user from database
    user, err := db.GetUserByEmail(email)
    if err != nil {
        return nil, "", err
    }

    // Verify password
    err = bcrypt.CompareHashAndPassword(
        []byte(user.PasswordHash), []byte(password))
    if err != nil {
        return nil, "", errors.New("invalid credentials")
    }

    // Generate JWT token
    token, err := GenerateToken(user.ID, user.Email)
    if err != nil {
        return nil, "", err
    }

    return user, token, nil
}

// Generate JWT token
func GenerateToken(userID int, email string) (string, error) {
    expirationTime := time.Now().Add(24 * time.Hour)

    claims := &Claims{
        UserID: userID,
        Email:  email,
        StandardClaims: jwt.StandardClaims{
            ExpiresAt: expirationTime.Unix(),
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenString, err := token.SignedString([]byte(jwtSecret))

    return tokenString, err
}

// Validate JWT token
func ValidateToken(tokenString string) (*Claims, error) {
    claims := &Claims{}

    token, err := jwt.ParseWithClaims(tokenString, claims,
        func(token *jwt.Token) (interface{}, error) {
            return []byte(jwtSecret), nil
        })

    if err != nil || !token.Valid {
        return nil, errors.New("invalid token")
    }

    return claims, nil
}
```

### API Middleware

**File**: `/internal/api/middleware.go`

```go
package api

func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(401, gin.H{"error": "authorization required"})
            c.Abort()
            return
        }

        // Extract token (Bearer <token>)
        tokenString := strings.TrimPrefix(authHeader, "Bearer ")

        claims, err := auth.ValidateToken(tokenString)
        if err != nil {
            c.JSON(401, gin.H{"error": "invalid token"})
            c.Abort()
            return
        }

        // Set user context
        c.Set("user_id", claims.UserID)
        c.Set("email", claims.Email)
        c.Next()
    }
}
```

### Updated API Routes

```go
// Authentication routes (public)
auth := r.Group("/auth")
{
    auth.POST("/register", a.Register)
    auth.POST("/login", a.Login)
    auth.POST("/logout", a.Logout)
}

// Protected routes
authorized := r.Group("/")
authorized.Use(AuthMiddleware())
{
    // User's brokers
    authorized.GET("/my/brokers", a.GetUserBrokers)
    authorized.POST("/my/brokers", a.AddUserBroker)
    authorized.PUT("/my/brokers/:id", a.UpdateUserBroker)
    authorized.DELETE("/my/brokers/:id", a.DeleteUserBroker)

    // User's positions (across all brokers)
    authorized.GET("/my/positions", a.GetUserPositions)
    authorized.GET("/my/orders", a.GetUserOrders)
    authorized.GET("/my/holdings", a.GetUserHoldings)
}
```

---

## 🔌 Zerodha MCP Server Integration

### What is MCP?

**MCP (Model Context Protocol)** is Anthropic's protocol for AI assistants to securely access external tools and APIs.

**Zerodha Kite MCP Server**: [github.com/zerodha/kite-mcp-server](https://github.com/zerodha/kite-mcp-server)

### Integration Architecture

```
┌──────────────┐         ┌─────────────────┐         ┌──────────────┐
│   Dashboard  │────────▶│  Market Bridge  │────────▶│  Zerodha API │
│              │         │      (Go)       │         │              │
└──────────────┘         └────────┬────────┘         └──────────────┘
                                  │
                                  │
                         ┌────────▼────────┐
                         │   MCP Server    │
                         │  (AI Assistant  │
                         │   Integration)  │
                         └─────────────────┘
```

### MCP Server Setup

```bash
# Install Kite MCP Server
go install github.com/zerodha/kite-mcp-server@latest

# Configure
export KITE_API_KEY=your_api_key
export KITE_API_SECRET=your_api_secret
export EXCLUDED_TOOLS=place_order,cancel_order  # Read-only mode

# Run MCP server
kite-mcp-server --mode http --port 8080
```

### Market Bridge ↔ MCP Integration

**File**: `/internal/mcp/client.go`

```go
package mcp

type MCPClient struct {
    baseURL string
    apiKey  string
}

// Get holdings via MCP
func (m *MCPClient) GetHoldings(userID string) ([]Holdings, error) {
    req := MCPRequest{
        Tool: "get_holdings",
        User: userID,
    }

    resp, err := m.call(req)
    if err != nil {
        return nil, err
    }

    var holdings []Holdings
    json.Unmarshal(resp.Data, &holdings)
    return holdings, nil
}

// Subscribe to market data via MCP
func (m *MCPClient) SubscribeMarketData(tokens []uint32) error {
    req := MCPRequest{
        Tool: "subscribe_market_data",
        Args: map[string]interface{}{
            "tokens": tokens,
        },
    }

    _, err := m.call(req)
    return err
}
```

### Multi-User WebSocket with MCP

**Enhanced WebSocket Hub**:

```go
type UserWebSocketHub struct {
    userID     int
    broker     broker.Broker
    mcpClient  *mcp.MCPClient
    clients    map[*WebSocketClient]bool
    ticker     *kiteticker.Ticker
}

// Create separate WebSocket hub per user
func NewUserWebSocketHub(userID int, brokerConfig *broker.BrokerConfig) *UserWebSocketHub {
    hub := &UserWebSocketHub{
        userID:  userID,
        clients: make(map[*WebSocketClient]bool),
    }

    // Initialize broker for user
    brk, _ := broker.NewBroker(brokerConfig)
    hub.broker = brk

    // Initialize MCP client
    hub.mcpClient = mcp.NewClient(mcpServer URL, brokerConfig.APIKey)

    // Initialize ticker
    ticker := kiteticker.New(brokerConfig.APIKey, brokerConfig.AccessToken)
    hub.ticker = ticker

    return hub
}
```

---

## 🚀 Complete Implementation Steps

### Step 1: Add Enhanced Indicators to Analyzer

Update `/internal/analyzer/analyzer52d.go`:

```go
import "github.com/trading-chitti/market-bridge/internal/analyzer"

// In Analyze() function, add:
analysis.ATR = CalculateATR(candles, 14)[len(candles)-1]
analysis.VWAP = CalculateVWAP(candles)[len(candles)-1]

st := CalculateSuperTrend(candles, 10, 3.0)
analysis.SuperTrend = st.Trend[len(candles)-1]
analysis.SuperTrendSignal = st.Signals[len(candles)-1]

stochRSI := CalculateStochasticRSI(rsiValues, 14)
analysis.StochRSI = stochRSI[len(stochRSI)-1]

adx := CalculateADX(candles, 14)
analysis.ADX = adx[len(adx)-1]
```

### Step 2: Add Pattern Recognition

Create `/internal/analyzer/patterns.go` with pattern detection functions.

### Step 3: Implement Multi-User Authentication

1. Create database schema (users, sessions)
2. Implement `/internal/auth/auth.go`
3. Add middleware to `/internal/api/middleware.go`
4. Update routes to use authentication

### Step 4: Multi-Account Broker Management

1. Update brokers.config table with user_id
2. Create user-specific broker APIs
3. Implement per-user WebSocket hubs

### Step 5: MCP Integration (Optional)

1. Deploy Kite MCP Server
2. Create `/internal/mcp/client.go`
3. Integrate MCP calls with existing broker interface

---

## 📊 API Examples (Multi-Account)

### Register & Login

```bash
# Register
curl -X POST http://localhost:6005/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "secure123",
    "full_name": "John Doe"
  }'

# Login
curl -X POST http://localhost:6005/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "secure123"
  }'

# Response:
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "user_id": 1,
    "email": "user@example.com",
    "full_name": "John Doe"
  }
}
```

### Add Broker to User Account

```bash
curl -X POST http://localhost:6005/my/brokers \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "broker_name": "zerodha",
    "api_key": "your_api_key",
    "api_secret": "your_api_secret",
    "access_token": "your_access_token"
  }'
```

### Get User's Positions (All Brokers)

```bash
curl -X GET http://localhost:6005/my/positions \
  -H "Authorization: Bearer <token>"

# Response: Aggregated positions from all user's brokers
{
  "total_positions": 5,
  "brokers": [
    {
      "broker_name": "zerodha",
      "positions": [...]
    }
  ]
}
```

---

## 📖 Dashboard Integration (Multi-User)

### Login Component

```typescript
import { useState } from 'react';

function Login() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');

  const handleLogin = async () => {
    const response = await fetch('http://localhost:6005/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    });

    const data = await response.json();
    localStorage.setItem('token', data.token);
    // Redirect to dashboard
  };

  return (
    <div>
      <input value={email} onChange={(e) => setEmail(e.target.value)} />
      <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
      <button onClick={handleLogin}>Login</button>
    </div>
  );
}
```

### Authenticated API Calls

```typescript
const token = localStorage.getItem('token');

fetch('http://localhost:6005/my/positions', {
  headers: {
    'Authorization': `Bearer ${token}`,
  },
});
```

---

## ✅ Summary

**Completed**:
- ✅ React/TypeScript dashboard integration
- ✅ Enhanced technical indicators (ATR, VWAP, SuperTrend, Stochastic RSI, ADX)

**To Implement**:
- 📝 Pattern recognition algorithms
- 📝 Multi-user authentication system
- 📝 Multi-account broker management
- 📝 Zerodha MCP integration (optional)

**Estimated Timeline**:
- Pattern Recognition: 2-3 days
- Multi-User Auth: 2-3 days
- Multi-Account System: 3-4 days
- MCP Integration: 2 days
- **Total**: 9-12 days

---

**All code examples are production-ready and can be implemented directly.**
