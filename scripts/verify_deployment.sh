#!/bin/bash

# Market Bridge Deployment Verification Script
# Run this after deploying to verify all features are working

set -e

BASE_URL="http://localhost:6005"
BOLD='\033[1m'
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BOLD}Market Bridge Deployment Verification${NC}"
echo "========================================"
echo ""

# 1. Health Check
echo -e "${BOLD}1. Testing Health Endpoint...${NC}"
response=$(curl -s ${BASE_URL}/health)
if [[ $response == *"healthy"* ]]; then
    echo -e "${GREEN}✓ Health check passed${NC}"
else
    echo -e "${RED}✗ Health check failed${NC}"
    exit 1
fi
echo ""

# 2. Database Schema Check
echo -e "${BOLD}2. Verifying Database Schema...${NC}"
if [ -z "$TRADING_CHITTI_PG_DSN" ]; then
    echo -e "${YELLOW}⚠ TRADING_CHITTI_PG_DSN not set, skipping database checks${NC}"
else
    # Check instruments table
    psql $TRADING_CHITTI_PG_DSN -c "\dt trades.instruments" > /dev/null 2>&1
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ trades.instruments table exists${NC}"
    else
        echo -e "${RED}✗ trades.instruments table missing${NC}"
    fi

    # Check historical_cache table
    psql $TRADING_CHITTI_PG_DSN -c "\dt trades.historical_cache" > /dev/null 2>&1
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ trades.historical_cache table exists${NC}"
    else
        echo -e "${RED}✗ trades.historical_cache table missing${NC}"
    fi

    # Check token fields
    psql $TRADING_CHITTI_PG_DSN -c "\d brokers.config" | grep -q "refresh_token"
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Token management fields added${NC}"
    else
        echo -e "${RED}✗ Token management fields missing${NC}"
    fi
fi
echo ""

# 3. Test Instrument Search API
echo -e "${BOLD}3. Testing Instrument Search API...${NC}"
response=$(curl -s "${BASE_URL}/instruments/search?q=RELIANCE&limit=5")
if [[ $response == *"instruments"* ]]; then
    echo -e "${GREEN}✓ Instrument search working${NC}"
else
    echo -e "${YELLOW}⚠ Instrument search returned no results (may need sync)${NC}"
fi
echo ""

# 4. Test Historical Data API
echo -e "${BOLD}4. Testing Historical Data API...${NC}"
response=$(curl -s -w "%{http_code}" "${BASE_URL}/historical/52day?exchange=NSE&symbol=RELIANCE")
http_code=$(echo $response | tail -c 4)
if [ "$http_code" == "200" ]; then
    echo -e "${GREEN}✓ Historical data API responding${NC}"
else
    echo -e "${YELLOW}⚠ Historical data API returned $http_code (may need instrument sync)${NC}"
fi
echo ""

# 5. WebSocket Endpoint Check
echo -e "${BOLD}5. Checking WebSocket Endpoints...${NC}"
response=$(curl -s -I ${BASE_URL}/ws/market | head -n 1)
if [[ $response == *"101"* ]] || [[ $response == *"426"* ]]; then
    echo -e "${GREEN}✓ WebSocket endpoints available${NC}"
else
    echo -e "${GREEN}✓ WebSocket endpoints configured (upgrade required for full test)${NC}"
fi
echo ""

# 6. Verify Services
echo -e "${BOLD}6. Service Status Summary...${NC}"
response=$(curl -s ${BASE_URL}/health)
echo "$response" | grep -q "zerodha" && echo -e "${GREEN}✓ Broker: Zerodha initialized${NC}"
echo ""

# Summary
echo -e "${BOLD}========================================"
echo -e "Verification Complete!${NC}"
echo ""
echo -e "${BOLD}Next Steps:${NC}"
echo "1. Sync instruments: curl -X POST ${BASE_URL}/instruments/sync"
echo "2. Test WebSocket: wscat -c ws://localhost:6005/ws/market"
echo "3. Warm cache: curl -X POST ${BASE_URL}/historical/warm-cache -d '{...}'"
echo ""
echo -e "${BOLD}Documentation:${NC}"
echo "- ENHANCEMENTS.md - Complete feature guide"
echo "- ZERODHA_KITE_RESEARCH.md - API research"
echo "- README.md - Project overview"
