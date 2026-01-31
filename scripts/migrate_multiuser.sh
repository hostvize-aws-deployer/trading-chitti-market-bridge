#!/bin/bash

# Multi-User Migration Script
# Applies database schema for authentication and multi-user support

set -e

echo "========================================="
echo " Market Bridge Multi-User Migration"
echo "========================================="
echo ""

# Check for PostgreSQL DSN
if [ -z "$TRADING_CHITTI_PG_DSN" ]; then
    echo "ERROR: TRADING_CHITTI_PG_DSN environment variable is not set"
    echo "Example: export TRADING_CHITTI_PG_DSN='postgresql://user:pass@localhost:5432/dbname'"
    exit 1
fi

# Extract connection parameters from DSN
# Format: postgresql://user:pass@host:port/dbname
echo "Connecting to PostgreSQL..."

# Create auth schema
echo ""
echo "1. Creating auth schema..."
psql "$TRADING_CHITTI_PG_DSN" <<EOF
CREATE SCHEMA IF NOT EXISTS auth;
EOF

# Apply users schema
echo ""
echo "2. Applying user tables schema..."
psql "$TRADING_CHITTI_PG_DSN" -f /Users/hariprasath/trading-chitti/market-bridge/internal/database/schema_users.sql

echo ""
echo "✅ Migration completed successfully!"
echo ""
echo "Next steps:"
echo "1. Start the server: make start"
echo "2. Register a user: POST /auth/register"
echo "3. Login: POST /auth/login"
echo "4. Add broker account: POST /brokers (with auth token)"
echo ""
echo "Documentation: PHASE2_MULTI_ACCOUNT.md"
