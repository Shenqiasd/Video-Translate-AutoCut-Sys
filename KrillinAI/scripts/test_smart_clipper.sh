#!/bin/bash
# Test Script for Smart Clipper API

BASE_URL="http://localhost:8888"

echo "=== Smart Clipper Verification ==="
echo "Target: $BASE_URL"

# 1. Check if endpoint exists (Method Check)
echo -n "1. Checking API Endpoint Availability... "
HTTP_CODE=$(curl -o /dev/null -s -w "%{http_code}\n" -X POST "$BASE_URL/api/smart_clipper/analyze" -H "Content-Type: application/json" -d '{}')

if [ "$HTTP_CODE" == "404" ]; then
    echo "FAILED (404 Not Found)"
    echo "❌ The Smart Clipper API is not reachable. It seems the new version hasn't been deployed yet."
    echo "👉 Please run 'make deploy' first."
    exit 1
elif [ "$HTTP_CODE" == "200" ] || [ "$HTTP_CODE" == "400" ] || [ "$HTTP_CODE" == "500" ]; then
    echo "SUCCESS (Endpoint Exists)"
else
    echo "WARNING (Unexpected Status: $HTTP_CODE)"
fi

# 2. Test Analysis with Invalid URL (Should return specific error, not crash)
echo -n "2. Testing Input Validation... "
RESPONSE=$(curl -s -X POST "$BASE_URL/api/smart_clipper/analyze" \
    -H "Content-Type: application/json" \
    -d '{"url": "invalid_url"}')

# Check if response contains "params error" or "failed"
if echo "$RESPONSE" | grep -q "failed" || echo "$RESPONSE" | grep -q "error"; then
    echo "PASS (Correctly handled invalid input)"
else
    echo "FAIL (Unexpected response: $RESPONSE)"
fi

echo ""
echo "=== Frontend Asset Check ==="
echo -n "3. Checking smart_clipper.js... "
JS_CODE=$(curl -o /dev/null -s -w "%{http_code}\n" "$BASE_URL/static/js/smart_clipper.js")
if [ "$JS_CODE" == "200" ]; then
    echo "PASS (File found)"
else
    echo "FAIL (File not found, Status: $JS_CODE)"
fi

echo ""
echo "✅ Verification Script Completed."
