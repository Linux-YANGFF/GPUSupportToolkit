#!/bin/bash
# QA Test: Verify arbitrary file read is prevented
# Start gst-server first: go run ./cmd/gst-server -port=8080 -browser=false &
# Then run this script.

set -e

BASE="http://localhost:8080"

echo "=== Test 1: Reject /etc/passwd ==="
RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE/api/log/parse" \
  -H "Content-Type: application/json" \
  -d '{"path":"/etc/passwd"}')
HTTP_CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | sed '$d')
echo "HTTP status: $HTTP_CODE"
echo "Body: $BODY"
if [ "$HTTP_CODE" != "400" ]; then
  echo "FAIL: Expected 400, got $HTTP_CODE"
  exit 1
fi
echo "PASS"

echo ""
echo "=== Test 2: Reject ../ traversal ==="
RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE/api/log/parse" \
  -H "Content-Type: application/json" \
  -d '{"path":"../../../etc/passwd"}')
HTTP_CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | sed '$d')
echo "HTTP status: $HTTP_CODE"
echo "Body: $BODY"
if [ "$HTTP_CODE" != "400" ]; then
  echo "FAIL: Expected 400, got $HTTP_CODE"
  exit 1
fi
echo "PASS"

echo ""
echo "=== Test 3: Accept valid file within allowed dir ==="
RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE/api/log/parse" \
  -H "Content-Type: application/json" \
  -d '{"path":"../exmple_log/1frame_demo_api.txt"}')
HTTP_CODE=$(echo "$RESP" | tail -1)
echo "HTTP status: $HTTP_CODE"
if [ "$HTTP_CODE" = "200" ]; then
  echo "PASS (file parsed successfully)"
elif [ "$HTTP_CODE" = "400" ]; then
  echo "NOTE: Path rejected by whitelist (GST_LOG_DIR not set)"
  echo "  Set GST_LOG_DIR to allow directory access:"
  echo "  GST_LOG_DIR=/root/code/GPUSupportToolkit/GPUSupportToolkit go run ./cmd/gst-server"
fi

echo ""
echo "All QA tests completed."
