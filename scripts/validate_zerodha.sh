#!/bin/bash

# Zerodha Integration Validation Script
# This script validates the Zerodha Kite Ticker setup

set -e

echo "=================================================="
echo "  Zerodha Integration Validation"
echo "=================================================="
echo ""

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check 1: Environment variables
echo "📋 Checking environment variables..."
if [ -z "$ZERODHA_API_KEY" ]; then
    echo -e "${YELLOW}⚠️  ZERODHA_API_KEY not set${NC}"
    MISSING_VARS=1
else
    echo -e "${GREEN}✅ ZERODHA_API_KEY found${NC}"
fi

if [ -z "$ZERODHA_API_SECRET" ]; then
    echo -e "${YELLOW}⚠️  ZERODHA_API_SECRET not set${NC}"
    MISSING_VARS=1
else
    echo -e "${GREEN}✅ ZERODHA_API_SECRET found${NC}"
fi

if [ -z "$ZERODHA_ACCESS_TOKEN" ]; then
    echo -e "${YELLOW}⚠️  ZERODHA_ACCESS_TOKEN not set${NC}"
    MISSING_VARS=1
else
    echo -e "${GREEN}✅ ZERODHA_ACCESS_TOKEN found${NC}"
fi

echo ""

# Check 2: .env file
echo "📄 Checking .env file..."
if [ -f ".env" ]; then
    echo -e "${GREEN}✅ .env file exists${NC}"

    if grep -q "ZERODHA_API_KEY" .env; then
        echo -e "${GREEN}✅ ZERODHA_API_KEY in .env${NC}"
    else
        echo -e "${YELLOW}⚠️  ZERODHA_API_KEY not in .env${NC}"
    fi

    if grep -q "ZERODHA_ACCESS_TOKEN" .env; then
        echo -e "${GREEN}✅ ZERODHA_ACCESS_TOKEN in .env${NC}"
    else
        echo -e "${YELLOW}⚠️  ZERODHA_ACCESS_TOKEN not in .env${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  .env file not found${NC}"
fi

echo ""

# Check 3: Go dependencies
echo "📦 Checking Go dependencies..."
if go list -m github.com/zerodha/gokiteconnect/v4 &>/dev/null; then
    VERSION=$(go list -m github.com/zerodha/gokiteconnect/v4 | awk '{print $2}')
    echo -e "${GREEN}✅ gokiteconnect v4 installed ($VERSION)${NC}"
else
    echo -e "${RED}❌ gokiteconnect v4 not installed${NC}"
    echo "   Run: go get github.com/zerodha/gokiteconnect/v4"
    exit 1
fi

echo ""

# Check 4: Database connection
echo "🗄️  Checking database connection..."
if [ -z "$TRADING_CHITTI_PG_DSN" ]; then
    echo -e "${YELLOW}⚠️  TRADING_CHITTI_PG_DSN not set${NC}"
    echo "   Using default: postgresql://hariprasath@localhost:6432/trading_chitti"
    export TRADING_CHITTI_PG_DSN="postgresql://hariprasath@localhost:6432/trading_chitti?sslmode=disable"
fi

if psql "$TRADING_CHITTI_PG_DSN" -c "SELECT 1" &>/dev/null; then
    echo -e "${GREEN}✅ Database connection successful${NC}"

    # Check if intraday tables exist
    if psql "$TRADING_CHITTI_PG_DSN" -c "SELECT 1 FROM md.tick_data LIMIT 1" &>/dev/null; then
        echo -e "${GREEN}✅ Intraday tables exist${NC}"
    else
        echo -e "${YELLOW}⚠️  Intraday tables not found${NC}"
        echo "   Run: psql $TRADING_CHITTI_PG_DSN -f internal/database/schema_intraday.sql"
    fi
else
    echo -e "${RED}❌ Database connection failed${NC}"
    exit 1
fi

echo ""

# Check 5: Binary exists
echo "🔨 Checking compiled binary..."
if [ -f "bin/market-bridge" ]; then
    SIZE=$(ls -lh bin/market-bridge | awk '{print $5}')
    echo -e "${GREEN}✅ Binary exists ($SIZE)${NC}"
else
    echo -e "${YELLOW}⚠️  Binary not found${NC}"
    echo "   Building..."
    go build -o bin/market-bridge cmd/server/main.go
    echo -e "${GREEN}✅ Binary built${NC}"
fi

echo ""
echo "=================================================="

# Summary
echo ""
if [ -n "$MISSING_VARS" ]; then
    echo -e "${YELLOW}⚠️  Configuration Incomplete${NC}"
    echo ""
    echo "To complete Zerodha integration:"
    echo ""
    echo "1. Get API credentials from: https://developers.kite.trade/"
    echo ""
    echo "2. Add to .env file:"
    echo "   ZERODHA_API_KEY=your_api_key_here"
    echo "   ZERODHA_API_SECRET=your_api_secret_here"
    echo "   ZERODHA_ACCESS_TOKEN=your_access_token_here"
    echo ""
    echo "3. Or set environment variables:"
    echo "   export ZERODHA_API_KEY=your_api_key_here"
    echo "   export ZERODHA_ACCESS_TOKEN=your_access_token_here"
    echo ""
    echo "For testing without credentials:"
    echo "   Use mock collectors (type: 'mock' instead of 'real')"
    echo ""
else
    echo -e "${GREEN}✅ All checks passed!${NC}"
    echo ""
    echo "Next steps:"
    echo "1. Start service: PORT=8080 ./bin/market-bridge"
    echo "2. Create collector: curl -X POST http://localhost:8080/api/collectors \\"
    echo "     -d '{\"name\":\"zerodha-live\",\"type\":\"real\",\"symbols\":[\"RELIANCE\"],\"auto_start\":true}'"
    echo ""
fi

echo "=================================================="
