# Multi-User System Guide

Complete guide for setting up and using Market Bridge's multi-user authentication and multi-account broker management system.

---

## Overview

The multi-user system enables:
- **User Authentication**: JWT-based secure authentication with email/password
- **Multi-Account Support**: Each user can connect multiple broker accounts (Zerodha, Angel One, etc.)
- **Per-User WebSocket Hubs**: Isolated WebSocket connections for each user's broker accounts
- **Session Management**: Refresh tokens, session tracking, and secure logout
- **Audit Logging**: Complete audit trail of all user actions

---

## Setup

### 1. Environment Variables

Add to `.env`:

```bash
# Multi-User Mode
MULTI_USER_MODE=true

# JWT Secret (generate a secure random string)
JWT_SECRET=your-super-secret-jwt-key-min-32-chars

# Database Connection
TRADING_CHITTI_PG_DSN=postgresql://user:password@localhost:5432/trading_chitti

# Optional: Port
PORT=6005
```

### 2. Database Migration

Apply the multi-user schema:

```bash
# Make migration script executable
chmod +x scripts/migrate_multiuser.sh

# Run migration
./scripts/migrate_multiuser.sh
```

This creates:
- `auth.users` - User accounts
- `auth.sessions` - Active sessions with JWT tokens
- `auth.api_keys` - API keys for external integrations
- `auth.audit_log` - Security audit trail
- Updates `brokers.config` with `user_id`, `account_name`, `is_default` fields

### 3. Start Server

```bash
# Start with multi-user mode
export MULTI_USER_MODE=true
export JWT_SECRET=your-super-secret-jwt-key-min-32-chars
go run cmd/server/main.go
```

Or using the Makefile:

```bash
make start
```

---

## API Usage

### Authentication

#### 1. Register a New User

**POST** `/api/auth/register`

```json
{
  "email": "user@example.com",
  "password": "secure-password-123",
  "full_name": "John Doe"
}
```

**Response:**

```json
{
  "user": {
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "full_name": "John Doe"
  },
  "token": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "a1b2c3d4e5f6...",
    "expires_at": "2024-01-30T10:15:00Z",
    "token_type": "Bearer"
  }
}
```

#### 2. Login

**POST** `/api/auth/login`

```json
{
  "email": "user@example.com",
  "password": "secure-password-123"
}
```

**Response:** Same as register

#### 3. Get Current User

**GET** `/api/auth/me`

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response:**

```json
{
  "user": {
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "full_name": "John Doe",
    "created_at": "2024-01-01T00:00:00Z",
    "last_login_at": "2024-01-30T09:00:00Z",
    "email_verified": false
  }
}
```

#### 4. Logout

**POST** `/api/auth/logout`

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response:**

```json
{
  "message": "logged out successfully"
}
```

#### 5. Refresh Token

**POST** `/api/auth/refresh`

```json
{
  "refresh_token": "a1b2c3d4e5f6..."
}
```

**Response:**

```json
{
  "token": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "g7h8i9j0k1l2...",
    "expires_at": "2024-01-30T10:30:00Z",
    "token_type": "Bearer"
  }
}
```

---

### Broker Account Management

All broker routes require authentication (Bearer token).

#### 1. List User's Broker Accounts

**GET** `/api/brokers`

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response:**

```json
{
  "accounts": [
    {
      "config_id": 1,
      "broker_name": "zerodha",
      "account_name": "My Trading Account",
      "is_default": true,
      "is_active": true,
      "created_at": "2024-01-01T00:00:00Z",
      "has_access_token": true
    },
    {
      "config_id": 2,
      "broker_name": "angelone",
      "account_name": "Secondary Account",
      "is_default": false,
      "is_active": true,
      "created_at": "2024-01-15T00:00:00Z",
      "has_access_token": false
    }
  ]
}
```

#### 2. Add Broker Account

**POST** `/api/brokers`

**Headers:**
```
Authorization: Bearer <access_token>
```

**Body:**

```json
{
  "broker_name": "zerodha",
  "api_key": "your_api_key",
  "api_secret": "your_api_secret",
  "account_name": "My Trading Account",
  "is_default": true
}
```

**Response:**

```json
{
  "config_id": 1,
  "broker_name": "zerodha",
  "account_name": "My Trading Account",
  "is_default": true,
  "is_active": true,
  "created_at": "2024-01-30T09:00:00Z"
}
```

#### 3. Get Specific Broker Account

**GET** `/api/brokers/:config_id`

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response:**

```json
{
  "config_id": 1,
  "broker_name": "zerodha",
  "account_name": "My Trading Account",
  "is_default": true,
  "is_active": true,
  "has_access_token": true,
  "token_expires_at": "2024-01-31T00:00:00Z",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-30T09:00:00Z"
}
```

#### 4. Set Default Broker Account

**POST** `/api/brokers/:config_id/set-default`

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response:**

```json
{
  "message": "default broker account updated",
  "config_id": "1"
}
```

#### 5. Delete Broker Account

**DELETE** `/api/brokers/:config_id`

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response:**

```json
{
  "message": "broker account deleted"
}
```

---

## WebSocket Integration

### Per-User WebSocket Hubs

In multi-user mode, each user gets their own isolated WebSocket hub connected to their default broker account.

**WebSocket Connection:**

```javascript
const token = "your_access_token";
const ws = new WebSocket(`ws://localhost:6005/ws/market?token=${token}`);

ws.onopen = () => {
  console.log('Connected to WebSocket');

  // Subscribe to instruments
  ws.send(JSON.stringify({
    type: 'subscribe',
    tokens: [256265, 738561]  // RELIANCE, INFY
  }));
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);

  switch(data.type) {
    case 'tick':
      console.log('Price update:', data);
      break;
    case 'order_update':
      console.log('Order update:', data);
      break;
    case 'status':
      console.log('Status:', data.status);
      break;
  }
};
```

---

## Security Best Practices

### 1. JWT Secret

Generate a strong JWT secret:

```bash
# Generate secure random string (32+ characters)
openssl rand -base64 32
```

Never commit JWT_SECRET to version control!

### 2. Password Requirements

- Minimum 8 characters
- Recommended: Mix of uppercase, lowercase, numbers, symbols

### 3. Token Expiry

- Access Token: 15 minutes (short-lived)
- Refresh Token: 7 days (use to get new access tokens)

### 4. HTTPS in Production

Always use HTTPS in production to protect tokens in transit:

```bash
# Use reverse proxy (Nginx, Caddy) with SSL/TLS
```

### 5. Rate Limiting

TODO: Implement rate limiting for login/register endpoints to prevent brute force attacks.

---

## Database Schema

### Users Table

```sql
CREATE TABLE auth.users (
    user_id UUID PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    full_name TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    last_login_at TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT TRUE,
    email_verified BOOLEAN DEFAULT FALSE
);
```

### Sessions Table

```sql
CREATE TABLE auth.sessions (
    session_id UUID PRIMARY KEY,
    user_id UUID REFERENCES auth.users(user_id),
    token_hash TEXT UNIQUE NOT NULL,
    refresh_token_hash TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    last_used_at TIMESTAMPTZ DEFAULT NOW(),
    ip_address TEXT,
    user_agent TEXT,
    is_revoked BOOLEAN DEFAULT FALSE
);
```

### Broker Configs (Updated)

```sql
ALTER TABLE brokers.config
    ADD COLUMN user_id UUID REFERENCES auth.users(user_id),
    ADD COLUMN account_name TEXT,
    ADD COLUMN is_default BOOLEAN DEFAULT FALSE;
```

---

## Migration from Single-User to Multi-User

### Option 1: Fresh Start

1. Apply multi-user schema
2. Set `MULTI_USER_MODE=true`
3. Register users via API
4. Users add their broker accounts

### Option 2: Migrate Existing Data

1. Create a "system" user:

```sql
INSERT INTO auth.users (email, password_hash, full_name, is_active)
VALUES ('system@localhost', 'n/a', 'System User', TRUE)
RETURNING user_id;
```

2. Update existing broker configs:

```sql
UPDATE brokers.config
SET user_id = '<system_user_id>',
    account_name = 'Legacy Account',
    is_default = TRUE
WHERE user_id IS NULL;
```

3. Enable multi-user mode and create real user accounts

---

## Troubleshooting

### Issue: "JWT_SECRET environment variable must be set"

**Solution:** Set JWT_SECRET in `.env` or environment

```bash
export JWT_SECRET=$(openssl rand -base64 32)
```

### Issue: "no active default broker account found for user"

**Solution:** Add a broker account and set it as default

```bash
curl -X POST http://localhost:6005/api/brokers \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"broker_name": "zerodha", "api_key": "...", "api_secret": "...", "account_name": "My Account", "is_default": true}'
```

### Issue: Token expired

**Solution:** Use refresh token to get new access token

```bash
curl -X POST http://localhost:6005/api/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "your_refresh_token"}'
```

---

## Example: Complete User Workflow

```bash
# 1. Register user
curl -X POST http://localhost:6005/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "trader@example.com", "password": "SecurePass123", "full_name": "Trader Joe"}'

# Save the access_token from response

# 2. Add Zerodha broker account
curl -X POST http://localhost:6005/api/brokers \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"broker_name": "zerodha", "api_key": "abc123", "api_secret": "xyz789", "account_name": "Main Trading", "is_default": true}'

# 3. Get Zerodha login URL (to get access token)
curl http://localhost:6005/login-url

# 4. Complete Zerodha login, get request_token

# 5. Generate session
curl -X POST http://localhost:6005/session \
  -H "Authorization: Bearer <access_token>" \
  -d '{"request_token": "kite_request_token"}'

# 6. Start trading!
curl http://localhost:6005/positions \
  -H "Authorization: Bearer <access_token>"
```

---

## Future Enhancements

### Planned Features

- [ ] Email verification
- [ ] Password reset via email
- [ ] Two-factor authentication (2FA)
- [ ] API key management for programmatic access
- [ ] Role-based access control (admin, trader, viewer)
- [ ] Per-user rate limiting
- [ ] Activity dashboard
- [ ] Multiple sessions management (view all devices)

---

## Support

For issues or questions:
1. Check `PHASE2_MULTI_ACCOUNT.md` for implementation details
2. Review audit logs: `SELECT * FROM auth.audit_log ORDER BY created_at DESC LIMIT 100;`
3. Enable debug logging: `export GIN_MODE=debug`

---

**Version:** 1.0.0
**Last Updated:** 2024-01-30
