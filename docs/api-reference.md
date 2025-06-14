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

Authentication is optional for this service. If you need authentication, provide an API key in the Authorization header:

```http
Authorization: Bearer YOUR_API_KEY
```

**Note:** The service will function without authentication for development and testing purposes.

## Common Headers

| Header | Required | Description |
|--------|----------|-------------|
| `Authorization` | No | API key for authentication (format: `Bearer YOUR_API_KEY`) |
| `Content-Type` | Yes | Must be `application/json` |
| `X-Request-ID` | No | Custom request ID (auto-generated if not provided) |
| `Accept-Encoding` | No | Supports `gzip` compression |
| `User-Agent` | No | Client identification (optional) |

## Response Headers

All responses include:

| Header | Description |
|--------|-------------|
| `X-Request-ID` | Unique request identifier for tracking |
| `X-Vendor-Source` | Indicates which vendor handled the request (e.g., "gemini", "openai") |
| `Content-Type` | Always `application/json; charset=utf-8` for successful responses |
| `Content-Encoding` | `gzip` if compression is used |
| `Content-Length` | Response body size |
| `Server` | Always `Generative-API-Router/1.0` |
| `X-Powered-By` | Always `Generative-API-Router` |

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
  "timestamp": "2025-06-14T08:57:03Z",
  "services": {
    "api": "up",
    "credentials": "up",
    "database": "up",
    "models": "up",
    "selector": "up"
  },
  "details": {
    "uptime": 254,
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
| `services.database` | string | Database connectivity status ("up" or "down") |
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

Messages can contain either simple text content or multi-part content with images and files:

**Simple Text Message:**
```json
{
  "role": "user|assistant|system|tool",
  "content": "Simple text message",
  "name": "Optional name for user/tool messages",
  "tool_calls": [...],
  "tool_call_id": "call_abc123"
}
```

**Multi-Part Content (Vision and File Processing):**
```json
{
  "role": "user",
  "content": [
    {
      "type": "text",
      "text": "Please analyze this image and document:"
    },
    {
      "type": "image_url",
      "image_url": {
        "url": "https://example.com/image.jpg",
        "headers": {
          "Authorization": "Bearer token"
        }
      }
    },
    {
      "type": "file_url",
      "file_url": {
        "url": "https://example.com/document.pdf",
        "headers": {
          "Authorization": "Bearer token",
          "User-Agent": "CustomBot/1.0"
        }
      }
    }
  ]
}
```

**Content Part Types:**

| Type | Description | Required Fields | Optional Fields |
|------|-------------|-----------------|-----------------|
| `text` | Plain text content | `text` | - |
| `image_url` | Image from URL (auto-converted to base64) | `image_url.url` | `image_url.headers` |
| `file_url` | Document from URL (auto-converted to text) | `file_url.url` | `file_url.headers` |

#### Non-Streaming Response
```http
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
X-Request-ID: 7da546cae5df562d
X-Vendor-Source: gemini

{
  "id": "5zhNaIiVG5PQz7IPy_v90A8",
  "object": "chat.completion",
  "created": 1749891303,
  "model": "test-model",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Hello! Received your test message.\n\nHow can I assist you today?",
        "annotations": [],
        "refusal": null
      },
      "finish_reason": "stop",
      "logprobs": null
    }
  ],
  "usage": {
    "prompt_tokens": 8,
    "completion_tokens": 15,
    "completion_tokens_details": {
      "accepted_prediction_tokens": 0,
      "audio_tokens": 0,
      "reasoning_tokens": 0,
      "rejected_prediction_tokens": 0
    },
    "prompt_tokens_details": {
      "audio_tokens": 0,
      "cached_tokens": 0
    },
    "total_tokens": 520
  },
  "service_tier": "default",
  "system_fingerprint": "fp_8ccf39dc93814d5f89"
}
```

#### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique completion identifier |
| `object` | string | Always "chat.completion" |
| `created` | integer | Unix timestamp of completion creation |
| `model` | string | The model name you provided in the request (preserved) |
| `choices` | array | Array of completion choices |
| `choices[].index` | integer | Choice index (usually 0) |
| `choices[].message` | object | The assistant's message |
| `choices[].message.role` | string | Always "assistant" |
| `choices[].message.content` | string | The generated response content |
| `choices[].message.annotations` | array | Message annotations (usually empty) |
| `choices[].message.refusal` | string\|null | Refusal message if any (usually null) |
| `choices[].finish_reason` | string | Reason completion finished ("stop", "length", "tool_calls") |
| `choices[].logprobs` | object\|null | Log probabilities if requested (usually null) |
| `usage` | object | Token usage statistics |
| `usage.prompt_tokens` | integer | Number of tokens in the prompt |
| `usage.completion_tokens` | integer | Number of tokens in the completion |
| `usage.total_tokens` | integer | Total tokens used |
| `usage.completion_tokens_details` | object | Detailed completion token breakdown |
| `usage.prompt_tokens_details` | object | Detailed prompt token breakdown |
| `service_tier` | string | Service tier used (usually "default") |
| `system_fingerprint` | string | System fingerprint for consistency |

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

### File Processing

The service supports automatic processing of documents and images from URLs. Files are downloaded and converted to text or base64 format automatically.

#### Supported File Types

**Documents (via markitdown):**
- PDF files (.pdf)
- Microsoft Word (.docx, .doc)
- Microsoft Excel (.xlsx, .xls)
- Microsoft PowerPoint (.pptx, .ppt)
- Plain text files (.txt, .md)
- ZIP archives (extracts and processes contents)
- CSV files (.csv)
- JSON files (.json)
- XML files (.xml)
- HTML files (.html, .htm)

**Images (auto-converted to base64):**
- PNG (.png)
- JPEG (.jpg, .jpeg)
- GIF (.gif)
- WebP (.webp)
- BMP (.bmp)
- TIFF (.tiff, .tif)
- SVG (.svg)

#### File Processing Request

```json
{
  "model": "your-model",
  "messages": [
    {
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "Please analyze this research paper:"
        },
        {
          "type": "file_url",
          "file_url": {
            "url": "https://example.com/research-paper.pdf",
            "headers": {
              "Authorization": "Bearer token",
              "User-Agent": "Custom-Agent"
            }
          }
        }
      ]
    }
  ]
}
```

#### File Processing Features

- **Automatic Format Detection**: Files are processed based on content and URL
- **Custom Headers**: Support for authentication and custom headers
- **Graceful Error Handling**: Failed downloads result in user-friendly error messages
- **Size Limits**: 20MB maximum file size per file
- **Concurrent Processing**: Multiple files processed simultaneously
- **No Pre-validation**: Files are processed without URL validation
- **Vendor Compatibility**: Error messages appear as regular user content

#### File Processing Error Handling

When file processing fails, the system generates user-friendly error messages instead of technical errors:

```json
{
  "role": "user",
  "content": [
    {
      "type": "text",
      "text": "Please analyze this file:"
    },
    {
      "type": "text",
      "text": "I couldn't access the file at https://example.com/broken.pdf due to network connectivity issues. The file server appears to be unreachable or the domain doesn't exist. Please verify the URL or provide an alternative file."
    }
  ]
}
```

#### Common Error Scenarios

| Error Type | Generated Message |
|------------|-------------------|
| Network connectivity | "I couldn't access the file due to network connectivity issues..." |
| Authentication required | "The file requires authentication or access permissions that weren't provided..." |
| File not found (404) | "The file URL appears to be broken or the file has been moved/deleted..." |
| File too large | "The file is too large to process (exceeds 20MB limit)..." |
| Timeout | "The file took too long to download due to slow response from the file server..." |
| Unsupported format | "The file couldn't be converted to text. The file format may not be supported..." |
| Empty URL | "Error: No file URL provided. Please provide a valid file URL to process." |

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

The service returns error responses as plain text for most validation errors:

**Missing Required Field (400):**
```http
HTTP/1.1 400 Bad Request
Content-Type: text/plain; charset=utf-8
X-Request-ID: 82d7462ee81699d4

missing 'messages' field in request
```

**Invalid JSON Format (400):**
```http
HTTP/1.1 400 Bad Request
Content-Type: text/plain; charset=utf-8
X-Request-ID: 126e4a6bb0e281fa

Failed to process images: invalid request format: invalid character 'i' looking for beginning of value
```

**Structured Error Response (for some errors):**
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
```http
HTTP/1.1 400 Bad Request
Content-Type: text/plain; charset=utf-8

missing 'messages' field in request
```

**Invalid JSON:**
```http
HTTP/1.1 400 Bad Request
Content-Type: text/plain; charset=utf-8

Failed to process images: invalid request format: invalid character 'i' looking for beginning of value
```

**Authentication Error (if authentication is enabled):**
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
curl -X POST http://localhost:8082/v1/chat/completions \
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
  "id": "5zhNaIiVG5PQz7IPy_v90A8",
  "object": "chat.completion",
  "created": 1749891303,
  "model": "my-custom-model-name",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Quantum computing uses quantum mechanics principles to process information differently than classical computers...",
        "annotations": [],
        "refusal": null
      },
      "finish_reason": "length",
      "logprobs": null
    }
  ],
  "usage": {
    "prompt_tokens": 15,
    "completion_tokens": 100,
    "completion_tokens_details": {
      "accepted_prediction_tokens": 0,
      "audio_tokens": 0,
      "reasoning_tokens": 0,
      "rejected_prediction_tokens": 0
    },
    "prompt_tokens_details": {
      "audio_tokens": 0,
      "cached_tokens": 0
    },
    "total_tokens": 115
  },
  "service_tier": "default",
  "system_fingerprint": "fp_8ccf39dc93814d5f89"
}
```

### Streaming Chat

**Request:**
```bash
curl -X POST http://localhost:8082/v1/chat/completions \
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

### File Processing Request

**PDF Document Processing:**
```bash
curl -X POST http://localhost:8082/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "document-analyzer",
    "messages": [
      {
        "role": "user",
        "content": [
          {
            "type": "text",
            "text": "Please summarize this research paper:"
          },
          {
            "type": "file_url",
            "file_url": {
              "url": "https://ml-site.cdn-apple.com/papers/the-illusion-of-thinking.pdf"
            }
          }
        ]
      }
    ]
  }'
```

**ZIP Archive Processing:**
```bash
curl -X POST http://localhost:8082/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "file-analyzer",
    "messages": [
      {
        "role": "user",
        "content": [
          {
            "type": "text",
            "text": "What documents are in this ZIP file?"
          },
          {
            "type": "file_url",
            "file_url": {
              "url": "https://example.com/documents.zip",
              "headers": {
                "Authorization": "Bearer file-server-token"
              }
            }
          }
        ]
      }
    ]
  }'
```

**Multiple Files Processing:**
```bash
curl -X POST http://localhost:8082/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "multi-file-analyzer",
    "messages": [
      {
        "role": "user",
        "content": [
          {
            "type": "text",
            "text": "Compare these documents:"
          },
          {
            "type": "file_url",
            "file_url": {
              "url": "https://example.com/report1.pdf"
            }
          },
          {
            "type": "file_url",
            "file_url": {
              "url": "https://example.com/report2.docx"
            }
          }
        ]
      }
    ]
  }'
```

### Vendor-Specific Request

**Request:**
```bash
curl -X POST "http://localhost:8082/v1/chat/completions?vendor=openai" \
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
    api_key="not-required",  # API key not required for local development
    base_url="http://localhost:8082/v1"
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
    apiKey: 'not-required',  // API key not required for local development
    baseURL: 'http://localhost:8082/v1',
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
    config := openai.DefaultConfig("not-required")  // API key not required for local development
    config.BaseURL = "http://localhost:8082/v1"
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
6. **Request IDs**: 
   - The service prioritizes CF-Ray headers (from Cloudflare) for request tracking
   - Falls back to X-Request-ID header if CF-Ray is not present
   - Auto-generates an ID if neither header is provided
   - Use the returned X-Request-ID header for tracking and debugging

## Limits and Quotas

- **Request Size**: Maximum 10MB per request (JSON payload)
- **Response Size**: No hard limit (vendor dependent)
- **File Processing**: 
  - Maximum 20MB per file
  - 30-second download timeout per file
  - Concurrent processing of multiple files
  - No limit on number of files per request
- **Rate Limits**: Depend on vendor and API key configuration
- **Concurrent Requests**: No service-level limit (vendor dependent)

---

For more information, see:
- **[User Guide](user-guide.md)** - Complete usage documentation
- **[Examples](../examples/)** - Ready-to-use code examples
- **[Development Guide](development-guide.md)** - Contributing to the service