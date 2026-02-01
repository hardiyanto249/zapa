#!/bin/bash
# Load env vars
if [ -f .env ]; then
  export $(grep -v '^#' .env | xargs)
else
  echo ".env file not found"
  exit 1
fi

if [ -z "$GEMINI_API_KEY" ]; then
  echo "GEMINI_API_KEY is not set in .env"
  exit 1
fi

echo "Testing Gemini API Key..."
echo "Key: ${GEMINI_API_KEY:0:5}..."

# Test 5: Generate Content (gemini-2.0-flash)
echo "--- Test 5: Generate Content (gemini-2.0-flash) ---"
response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" -H 'Content-Type: application/json' \
  -d '{"contents":[{"parts":[{"text":"Hello"}]}]}' \
  "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=${GEMINI_API_KEY}")

http_status=$(echo "$response" | grep "HTTP_STATUS" | cut -d: -f2)
body=$(echo "$response" | grep -v "HTTP_STATUS")

if [ "$http_status" == "200" ]; then
  echo "✅ Generate Content (gemini-2.0-flash) SUCCESS."
  echo "Response snippet: ${body:0:100}..."
else
  echo "❌ Generate Content (gemini-2.0-flash) FAILED. HTTP Status: $http_status"
  echo "Response body: $body"
fi
