#!/bin/bash

# Generate secure API key for market-bridge authentication

echo "🔐 Generating Secure API Key for Market-Bridge"
echo "================================================"
echo ""

# Generate random API key (base64 URL-safe, 32 bytes = 43 characters)
API_KEY=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-43)

echo "✅ Generated API Key:"
echo ""
echo "    $API_KEY"
echo ""
echo "================================================"
echo ""
echo "📋 How to Use:"
echo ""
echo "1. Add to .env file:"
echo "   echo 'API_KEY=$API_KEY' >> .env"
echo ""
echo "2. Or export as environment variable:"
echo "   export API_KEY='$API_KEY'"
echo ""
echo "3. Restart market-bridge service:"
echo "   pkill market-bridge"
echo "   PORT=6005 API_KEY='$API_KEY' ./bin/market-bridge"
echo ""
echo "4. Test with API key:"
echo "   curl -H 'X-API-Key: $API_KEY' http://localhost:6005/health"
echo ""
echo "================================================"
echo ""
echo "⚠️  SECURITY:"
echo "- Keep this key secret"
echo "- Don't commit to Git"
echo "- Rotate periodically (every 90 days)"
echo "- Use different keys for dev/staging/prod"
echo ""
