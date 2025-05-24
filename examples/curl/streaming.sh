#!/bin/bash
# Streaming chat completion example

API_URL="http://localhost:8082"

# Basic streaming request
echo "=== Basic Streaming Request ==="
echo "Note: Streaming responses will appear as Server-Sent Events (SSE)"
echo ""

curl -X POST "$API_URL/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "my-streaming-model",
    "messages": [
      {
        "role": "user",
        "content": "Count from 1 to 5 slowly, with a brief explanation for each number."
      }
    ],
    "stream": true,
    "temperature": 0.7
  }'

echo -e "\n\n"

# Streaming with system message
echo "=== Streaming with System Context ==="
curl -X POST "$API_URL/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "story-teller",
    "messages": [
      {
        "role": "system",
        "content": "You are a storyteller. Tell stories in a dramatic, suspenseful way."
      },
      {
        "role": "user",
        "content": "Tell me a very short story about a robot learning to cook."
      }
    ],
    "stream": true,
    "max_tokens": 200
  }'

echo -e "\n\n"

# Note about processing streaming responses
echo "=== Processing Streaming Responses ==="
echo "To process streaming responses programmatically:"
echo "1. Each line starting with 'data: ' contains a JSON chunk"
echo "2. Parse the JSON after removing the 'data: ' prefix"
echo "3. The stream ends with 'data: [DONE]'"
echo ""
echo "Example parsing with curl and jq:"
echo ""

curl -sN -X POST "$API_URL/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "demo-model",
    "messages": [{"role": "user", "content": "Say hello"}],
    "stream": true
  }' | while IFS= read -r line; do
    if [[ $line == data:* ]]; then
        data=${line#data: }
        if [[ $data != "[DONE]" ]]; then
            echo "$data" | jq -r '.choices[0].delta.content // empty' 2>/dev/null | tr -d '\n'
        fi
    fi
done

echo -e "\n" 