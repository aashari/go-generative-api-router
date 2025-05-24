#!/bin/bash
# Tool calling (function calling) example

API_URL="http://localhost:8082"

# Basic tool calling
echo "=== Basic Tool Calling ==="
curl -X POST "$API_URL/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "tool-capable-model",
    "messages": [
      {
        "role": "user",
        "content": "What is the weather like in San Francisco and Tokyo?"
      }
    ],
    "tools": [
      {
        "type": "function",
        "function": {
          "name": "get_current_weather",
          "description": "Get the current weather for a given location",
          "parameters": {
            "type": "object",
            "properties": {
              "location": {
                "type": "string",
                "description": "The city name, e.g., San Francisco"
              },
              "unit": {
                "type": "string",
                "enum": ["celsius", "fahrenheit"],
                "description": "The temperature unit"
              }
            },
            "required": ["location"]
          }
        }
      }
    ],
    "tool_choice": "auto"
  }' | jq

echo -e "\n\n"

# Multiple tools example
echo "=== Multiple Tools Example ==="
curl -X POST "$API_URL/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "assistant-model",
    "messages": [
      {
        "role": "user",
        "content": "Search for Italian restaurants in New York and book a table for 2 at 7 PM tomorrow"
      }
    ],
    "tools": [
      {
        "type": "function",
        "function": {
          "name": "search_restaurants",
          "description": "Search for restaurants in a specific location",
          "parameters": {
            "type": "object",
            "properties": {
              "location": {
                "type": "string",
                "description": "The city or area to search in"
              },
              "cuisine": {
                "type": "string",
                "description": "Type of cuisine"
              },
              "price_range": {
                "type": "string",
                "enum": ["$", "$$", "$$$", "$$$$"],
                "description": "Price range"
              }
            },
            "required": ["location"]
          }
        }
      },
      {
        "type": "function",
        "function": {
          "name": "book_restaurant",
          "description": "Book a table at a restaurant",
          "parameters": {
            "type": "object",
            "properties": {
              "restaurant_name": {
                "type": "string",
                "description": "Name of the restaurant"
              },
              "date": {
                "type": "string",
                "description": "Date for the reservation (YYYY-MM-DD)"
              },
              "time": {
                "type": "string",
                "description": "Time for the reservation (HH:MM)"
              },
              "party_size": {
                "type": "integer",
                "description": "Number of people"
              }
            },
            "required": ["restaurant_name", "date", "time", "party_size"]
          }
        }
      }
    ],
    "tool_choice": "auto"
  }' | jq

echo -e "\n\n"

# Forced tool use
echo "=== Forced Tool Use ==="
curl -X POST "$API_URL/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "calculator-model",
    "messages": [
      {
        "role": "user",
        "content": "Calculate the result"
      }
    ],
    "tools": [
      {
        "type": "function",
        "function": {
          "name": "calculator",
          "description": "Perform mathematical calculations",
          "parameters": {
            "type": "object",
            "properties": {
              "expression": {
                "type": "string",
                "description": "Mathematical expression to evaluate"
              }
            },
            "required": ["expression"]
          }
        }
      }
    ],
    "tool_choice": {
      "type": "function",
      "function": {
        "name": "calculator"
      }
    }
  }' | jq

echo -e "\n\n"

# Tool response handling example
echo "=== Complete Tool Interaction Flow ==="
echo "Step 1: Initial request with tools"
RESPONSE=$(curl -s -X POST "$API_URL/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "weather-assistant",
    "messages": [
      {
        "role": "user",
        "content": "What should I wear in London today?"
      }
    ],
    "tools": [
      {
        "type": "function",
        "function": {
          "name": "get_weather",
          "description": "Get weather information",
          "parameters": {
            "type": "object",
            "properties": {
              "location": {"type": "string"}
            },
            "required": ["location"]
          }
        }
      }
    ]
  }')

echo "$RESPONSE" | jq

# Note: In a real implementation, you would:
# 1. Extract tool_calls from the response
# 2. Execute the actual function
# 3. Send another request with the tool response

echo -e "\n"
echo "Step 2: In practice, you would then:"
echo "- Extract the tool_calls from the response"
echo "- Execute the requested function(s)"
echo "- Send a follow-up request with the tool results" 