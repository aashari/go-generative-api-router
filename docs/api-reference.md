# API Reference

Complete API documentation for the Generative API Router service.

## Overview

The Generative API Router provides an OpenAI-compatible API that routes requests to multiple LLM vendors while preserving your original model names in responses. It supports all standard OpenAI API features including streaming, tool calling, and function calling.

**Base URL:**
- Production: `https://genapi.example.com`
- Development: `https://dev-genapi.example.com`
- Local: `http://localhost:8082`

**API Version:** 1.0  
**Protocol:** HTTPS/HTTP  
**Content-Type:** `application/json`

## Authentication

All API requests require authentication using an API key in the Authorization header:

```http
Authorization: Bearer YOUR_API_KEY
```

## Common Headers

| Header | Required | Description |
|--------|----------|-------------|
| `Authorization` | Yes | Bearer token for authentication |
| `Content-Type` | Yes | Must be `application/json` for POST requests |
| `User-Agent` | No | Client identification (optional) |
| `X-Request-ID` | No | Custom request ID (auto-generated if not provided) |

## Response Headers

All responses include:

| Header | Description |
|--------|-------------|
| `X-Request-ID` | Unique request identifier for tracking |
| `Content-Type` | Response content type |
| `Content-Length` | Response body size |

## Endpoints

### Health Check

Check if the service is running properly.

#### Request
```http
GET /health
```

#### Response
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "status": "healthy",
  "timestamp": "2025-06-07T05:56:39Z",
  "services": {
    "api": "up",
    "credentials": "up",
    "models": "up",
    "selector": "up"
  },
  "details": {
    "uptime": 196,
    "version": "unknown"
  }
}
```

#### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `status` | string | Overall service status ("healthy" or "unhealthy") |
| `timestamp` | string | ISO 8601 timestamp of the health check |
| `services` | object | Status of individual service components |
| `services.api` | string | API service status ("up" or "down") |
| `services.credentials` | string | Credentials loading status ("up" or "down") |
| `services.models` | string | Models configuration status ("up" or "down") |
| `services.selector` | string | Vendor selector status ("up" or "down") |
| `details` | object | Additional service details |
| `details.uptime` | integer | Service uptime in seconds |
| `details.version` | string | Service version from VERSION environment variable |

### List Models

Retrieve the list of available models.

#### Request
```http
GET /v1/models
```

#### Response
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "object": "list",
  "data": [
    {
      "id": "any-model-name",
      "object": "model",
      "created": 1234567890,
      "owned_by": "generative-api-router"
    }
  ]
}
```

**Note:** The service accepts any model name and routes to available vendors. The actual vendor-model combinations are configured server-side.

### Chat Completions

Create a chat completion response.

#### Request
```http
POST /v1/chat/completions
Content-Type: application/json
Authorization: Bearer YOUR_API_KEY

{
  "model": "your-preferred-model-name",
  "messages": [
    {
      "role": "user",
      "content": "Hello, how are you?"
    }
  ],
  "max_tokens": 150,
  "temperature": 0.7,
  "stream": false
}
```

#### Request Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `model` | string | Yes | - | Any model name (preserved in response) |
| `messages` | array | Yes | - | Array of message objects |
| `max_tokens` | integer | No | - | Maximum tokens to generate |
| `temperature` | float | No | 1.0 | Sampling temperature (0-2) |
| `top_p` | float | No | 1.0 | Nucleus sampling parameter |
| `n` | integer | No | 1 | Number of completions to generate |
| `stream` | boolean | No | false | Whether to stream responses |
| `stop` | string/array | No | null | Stop sequences |
| `presence_penalty` | float | No | 0 | Presence penalty (-2 to 2) |
| `frequency_penalty` | float | No | 0 | Frequency penalty (-2 to 2) |
| `logit_bias` | object | No | null | Token logit biases |
| `user` | string | No | - | End-user identifier |
| `tools` | array | No | - | Available tools for function calling |
| `tool_choice` | string/object | No | "auto" | Tool selection preference |

#### Message Object

```json
{
  "role": "user|assistant|system|tool",
  "content": "Message content",
  "name": "Optional name for user/tool messages",
  "tool_calls": [
    {
      "id": "call_abc123",
      "type": "function",
      "function": {
        "name": "function_name",
        "arguments": "{\"param\": \"value\"}"
      }
    }
  ],
  "tool_call_id": "call_abc123"
}
```

#### Non-Streaming Response
```http
HTTP/1.1 200 OK
Content-Type: application/json
X-Request-ID: abc123def456

{
  "id": "chatcmpl-abc123",
  "object": "chat.completion",
  "created": 1677652288,
  "model": "your-preferred-model-name",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Hello! I'm doing well, thank you for asking. How can I help you today?"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 12,
    "completion_tokens": 20,
    "total_tokens": 32
  }
}
```

#### Streaming Response

When `stream: true` is set, responses are sent as Server-Sent Events:

```http
HTTP/1.1 200 OK
Content-Type: text/event-stream
X-Request-ID: abc123def456

data: {"id":"chatcmpl-abc123","object":"chat.completion.chunk","created":1677652288,"model":"your-preferred-model-name","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}

data: {"id":"chatcmpl-abc123","object":"chat.completion.chunk","created":1677652288,"model":"your-preferred-model-name","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}

data: {"id":"chatcmpl-abc123","object":"chat.completion.chunk","created":1677652288,"model":"your-preferred-model-name","choices":[{"index":0,"delta":{"content":"!"},"finish_reason":null}]}

data: {"id":"chatcmpl-abc123","object":"chat.completion.chunk","created":1677652288,"model":"your-preferred-model-name","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}

data: [DONE]
```

## Advanced Features

### Vendor Selection

Force a specific vendor using query parameters:

```http
POST /v1/chat/completions?vendor=openai
POST /v1/chat/completions?vendor=gemini
```

Available vendors depend on server configuration.

### Tool Calling

The service supports OpenAI-compatible tool calling:

```json
{
  "model": "your-model",
  "messages": [
    {
      "role": "user",
      "content": "What's the weather in Boston?"
    }
  ],
  "tools": [
    {
      "type": "function",
      "function": {
        "name": "get_weather",
        "description": "Get weather information for a location",
        "parameters": {
          "type": "object",
          "properties": {
            "location": {
              "type": "string",
              "description": "City name"
            }
          },
          "required": ["location"]
        }
      }
    }
  ],
  "tool_choice": "auto"
}
```

#### Tool Choice Options

| Value | Description |
|-------|-------------|
| `"auto"` | Let the model decide whether to call tools |
| `"none"` | Never call tools |
| `{"type": "function", "function": {"name": "tool_name"}}` | Force specific tool |

#### Tool Response Format

```json
{
  "id": "chatcmpl-abc123",
  "model": "your-model",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": null,
        "tool_calls": [
          {
            "id": "call_abc123",
            "type": "function",
            "function": {
              "name": "get_weather",
              "arguments": "{\"location\": \"Boston\"}"
            }
          }
        ]
      },
      "finish_reason": "tool_calls"
    }
  ]
}
```

### Error Handling

The service returns structured error responses:

```json
{
  "error": {
    "type": "invalid_request_error",
    "message": "The 'messages' field is required",
    "code": "missing_required_field",
    "param": "messages"
  }
}
```

#### Error Types

| Type | HTTP Status | Description |
|------|-------------|-------------|
| `invalid_request_error` | 400 | Invalid request format or parameters |
| `authentication_error` | 401 | Invalid or missing API key |
| `permission_error` | 403 | Insufficient permissions |
| `not_found_error` | 404 | Resource not found |
| `rate_limit_error` | 429 | Rate limit exceeded |
| `server_error` | 500 | Internal server error |
| `service_unavailable_error` | 503 | Service temporarily unavailable |

#### Common Error Responses

**Missing Messages Field:**
```json
{
  "error": {
    "type": "invalid_request_error",
    "message": "The 'messages' field is required",
    "code": "missing_required_field",
    "param": "messages"
  }
}
```

**Invalid JSON:**
```json
{
  "error": {
    "type": "invalid_request_error",
    "message": "Invalid JSON format in request body",
    "code": "json_parse_error"
  }
}
```

**Authentication Error:**
```json
{
  "error": {
    "type": "authentication_error",
    "message": "Invalid API key provided",
    "code": "invalid_api_key"
  }
}
```

## Rate Limits

Rate limits depend on your API key configuration and vendor limitations. When rate limited, you'll receive:

```http
HTTP/1.1 429 Too Many Requests
X-Request-ID: abc123def456

{
  "error": {
    "type": "rate_limit_error",
    "message": "Rate limit exceeded. Please try again later.",
    "code": "rate_limit_exceeded"
  }
}
```

## Request/Response Examples

### Basic Chat

**Request:**
```bash
curl -X POST https://genapi.example.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "my-custom-model-name",
    "messages": [
      {"role": "user", "content": "Explain quantum computing in simple terms"}
    ],
    "max_tokens": 100
  }'
```

**Response:**
```json
{
  "id": "chatcmpl-abc123",
  "object": "chat.completion",
  "created": 1677652288,
  "model": "my-custom-model-name",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Quantum computing uses quantum mechanics principles to process information differently than classical computers..."
      },
      "finish_reason": "length"
    }
  ],
  "usage": {
    "prompt_tokens": 15,
    "completion_tokens": 100,
    "total_tokens": 115
  }
}
```

### Streaming Chat

**Request:**
```bash
curl -X POST https://genapi.example.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "streaming-model",
    "messages": [
      {"role": "user", "content": "Count from 1 to 5"}
    ],
    "stream": true,
    "max_tokens": 50
  }'
```

**Response:**
```
data: {"id":"chatcmpl-abc123","object":"chat.completion.chunk","created":1677652288,"model":"streaming-model","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}

data: {"id":"chatcmpl-abc123","object":"chat.completion.chunk","created":1677652288,"model":"streaming-model","choices":[{"index":0,"delta":{"content":"1"},"finish_reason":null}]}

data: {"id":"chatcmpl-abc123","object":"chat.completion.chunk","created":1677652288,"model":"streaming-model","choices":[{"index":0,"delta":{"content":", 2"},"finish_reason":null}]}

data: {"id":"chatcmpl-abc123","object":"chat.completion.chunk","created":1677652288,"model":"streaming-model","choices":[{"index":0,"delta":{"content":", 3, 4, 5"},"finish_reason":"stop"}]}

data: [DONE]
```

### Vendor-Specific Request

**Request:**
```bash
curl -X POST "https://genapi.example.com/v1/chat/completions?vendor=openai" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "forced-openai-model",
    "messages": [
      {"role": "user", "content": "Hello from OpenAI!"}
    ]
  }'
```

## Client Libraries

The service is compatible with existing OpenAI client libraries. Simply change the base URL:

### Python (OpenAI SDK)
```python
import openai

client = openai.OpenAI(
    api_key="your-api-key",
    base_url="https://genapi.example.com/v1"
)

response = client.chat.completions.create(
    model="your-preferred-model-name",
    messages=[
        {"role": "user", "content": "Hello!"}
    ]
)
```

### Node.js (OpenAI SDK)
```javascript
import OpenAI from 'openai';

const openai = new OpenAI({
    apiKey: 'your-api-key',
    baseURL: 'https://genapi.example.com/v1',
});

const response = await openai.chat.completions.create({
    model: 'your-preferred-model-name',
    messages: [
        { role: 'user', content: 'Hello!' }
    ],
});
```

### Go
```go
package main

import (
    "context"
    "github.com/sashabaranov/go-openai"
)

func main() {
    config := openai.DefaultConfig("your-api-key")
    config.BaseURL = "https://genapi.example.com/v1"
    client := openai.NewClientWithConfig(config)

    resp, err := client.CreateChatCompletion(
        context.Background(),
        openai.ChatCompletionRequest{
            Model: "your-preferred-model-name",
            Messages: []openai.ChatCompletionMessage{
                {
                    Role:    openai.ChatMessageRoleUser,
                    Content: "Hello!",
                },
            },
        },
    )
}
```

## OpenAPI Specification

The complete OpenAPI/Swagger specification is available at:
- **JSON**: `/swagger.json`
- **YAML**: `/swagger.yaml`
- **Interactive Docs**: `/swagger/` (if enabled)

## Versioning

The API follows semantic versioning. Current version: **1.0**

Version information is included in:
- OpenAPI specification
- User-Agent headers in vendor requests
- Service metadata

## Best Practices

1. **Model Names**: Use descriptive model names that make sense for your application
2. **Error Handling**: Always handle error responses and status codes
3. **Rate Limiting**: Implement exponential backoff for rate limit errors
4. **Streaming**: Use streaming for long responses to improve user experience
5. **Tool Calling**: Validate tool function schemas carefully
6. **Request IDs**: Use the X-Request-ID header for request tracking

## Limits and Quotas

- **Request Size**: Maximum 10MB per request
- **Response Size**: No hard limit (vendor dependent)
- **Rate Limits**: Depend on vendor and API key configuration
- **Concurrent Requests**: No service-level limit (vendor dependent)

---

For more information, see:
- **[User Guide](user-guide.md)** - Complete usage documentation
- **[Examples](../examples/)** - Ready-to-use code examples
- **[Development Guide](development-guide.md)** - Contributing to the service