#!/bin/bash
# Basic chat completion example

API_URL="http://localhost:8082"

# Health check
echo "=== Health Check ==="
curl -X GET "$API_URL/health"
echo -e "\n"

# List available models
echo "=== List Models ==="
curl -X GET "$API_URL/v1/models" | jq
echo -e "\n"

# Basic chat completion
echo "=== Basic Chat Completion ==="
curl -X POST "$API_URL/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "my-custom-model",
    "messages": [
      {
        "role": "user",
        "content": "Hello! Can you explain what you are in one sentence?"
      }
    ],
    "temperature": 0.7,
    "max_tokens": 100
  }' | jq

echo -e "\n"

# Chat with system message
echo "=== Chat with System Message ==="
curl -X POST "$API_URL/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "my-assistant",
    "messages": [
      {
        "role": "system",
        "content": "You are a helpful assistant that speaks like a pirate."
      },
      {
        "role": "user",
        "content": "Tell me about the weather today."
      }
    ]
  }' | jq

echo -e "\n"

# Vendor-specific request (OpenAI)
echo "=== Vendor-Specific Request (OpenAI) ==="
curl -X POST "$API_URL/v1/chat/completions?vendor=openai" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-model",
    "messages": [
      {
        "role": "user",
        "content": "What is 2+2?"
      }
    ]
  }' | jq 