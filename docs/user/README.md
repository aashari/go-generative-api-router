# User Guide

This guide provides comprehensive information for users who want to integrate with and use the Generative API Router service.

> **📖 Quick Start**: For project overview and initial setup, see the [Main README](../../README.md).  
> **🔧 Development**: For contributing and development information, see [Development Documentation](../development/).

## 🚀 Getting Started

### Service Overview
The Generative API Router is a transparent proxy that routes OpenAI-compatible API calls to multiple LLM vendors (OpenAI, Gemini) while maintaining complete API compatibility. The service preserves your original model names in responses while intelligently selecting from available vendor-model combinations.

### Prerequisites
- The service running on your target host (default: `localhost:8082`)
- Valid API keys for at least one supported vendor (OpenAI or Gemini)
- HTTP client capable of making REST API calls

### Basic Usage
Point your OpenAI-compatible client to the router instead of OpenAI directly:

**Local Development**:
```bash
# Instead of: https://api.openai.com/v1/chat/completions
# Use: http://localhost:8082/v1/chat/completions
```

**Production Service** (xyz-aduh-genapi):
```bash
# Production endpoint
# Use: https://genapi.aduh.xyz/v1/chat/completions

# Development endpoint  
# Use: https://dev-genapi.aduh.xyz/v1/chat/completions
```

## 🔌 API Endpoints

### Health Check
```http
GET /health
```

**Purpose**: Verify the service is running and responsive.

**Response**: 
- **200 OK**: Service is healthy
- **Body**: `OK`

**Examples**:
```bash
# Local development
curl http://localhost:8082/health

# Production service (xyz-aduh-genapi)
curl https://genapi.aduh.xyz/health

# Development environment (xyz-aduh-genapi)
curl https://dev-genapi.aduh.xyz/health

# Response: OK
```

### List Available Models
```http
GET /v1/models
GET /v1/models?vendor=openai
```

**Purpose**: Get a list of all available models in OpenAI-compatible format.

**Query Parameters**:
- `vendor` (optional): Filter models by vendor (`openai`, `gemini`)

**Response Format**:
```json
{
  "object": "list",
  "data": [
    {
      "id": "gpt-4o",
      "object": "model",
      "created": 1677610602,
      "owned_by": "openai"
    },
    {
      "id": "gemini-2.0-flash",
      "object": "model", 
      "created": 1677610602,
      "owned_by": "google"
    }
  ]
}
```

**Examples**:
```bash
# List all models
curl http://localhost:8082/v1/models

# List only OpenAI models
curl http://localhost:8082/v1/models?vendor=openai

# List only Gemini models  
curl http://localhost:8082/v1/models?vendor=gemini
```

### Chat Completions
```http
POST /v1/chat/completions
POST /v1/chat/completions?vendor=openai
```

**Purpose**: Generate chat completions using any configured model.

**Key Features**:
- **Transparent Model Names**: Response preserves your requested model name
- **Vendor Selection**: Automatic or explicit vendor selection
- **Streaming Support**: Real-time response streaming
- **Tool Calling**: Function calling capabilities
- **OpenAI Compatibility**: 100% compatible with OpenAI API format

**Request Format** (OpenAI-compatible):
```json
{
  "model": "any-model-name-you-like",
  "messages": [
    {"role": "user", "content": "Hello!"}
  ],
  "stream": false,
  "max_tokens": 100,
  "temperature": 0.7
}
```

**Response Format**:
```json
{
  "id": "chatcmpl-abc123",
  "object": "chat.completion",
  "created": 1677652288,
  "model": "any-model-name-you-like",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Hello! How can I help you today?"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 9,
    "completion_tokens": 12,
    "total_tokens": 21
  }
}
```

## 🎯 Usage Examples

### Basic Chat Completion
```bash
curl -X POST http://localhost:8082/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "my-custom-model-name",
    "messages": [
      {"role": "user", "content": "What is the capital of France?"}
    ]
  }'
```

### Streaming Response
```bash
curl -X POST http://localhost:8082/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "streaming-model",
    "messages": [
      {"role": "user", "content": "Count from 1 to 5"}
    ],
    "stream": true
  }'
```

### Tool Calling
```bash
curl -X POST http://localhost:8082/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "tool-capable-model",
    "messages": [
      {"role": "user", "content": "What is the weather in Boston?"}
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
  }'
```

### Vendor-Specific Requests
```bash
# Force OpenAI vendor
curl -X POST "http://localhost:8082/v1/chat/completions?vendor=openai" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "openai-specific-model",
    "messages": [{"role": "user", "content": "Hello from OpenAI"}]
  }'

# Force Gemini vendor
curl -X POST "http://localhost:8082/v1/chat/completions?vendor=gemini" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-specific-model", 
    "messages": [{"role": "user", "content": "Hello from Gemini"}]
  }'
```

## 🔧 Configuration

### Environment Variables

Configure the service behavior using these environment variables:

| Variable | Description | Values | Default |
|----------|-------------|--------|---------|
| `PORT` | Server port | Any valid port | `8082` |
| `LOG_LEVEL` | Logging detail level | `DEBUG`, `INFO`, `WARN`, `ERROR` | `INFO` |
| `LOG_FORMAT` | Log output format | `json`, `text` | `json` |
| `LOG_OUTPUT` | Log destination | `stdout`, `stderr` | `stdout` |

**Examples**:
```bash
# Development configuration
PORT=8080 LOG_LEVEL=DEBUG LOG_FORMAT=text ./server

# Production configuration
PORT=8082 LOG_LEVEL=INFO LOG_FORMAT=json ./server
```

### Model Configuration

Models are configured in `configs/models.json`. This file defines which vendor-model combinations are available for selection:

```json
[
  {
    "vendor": "openai",
    "model": "gpt-4o"
  },
  {
    "vendor": "openai", 
    "model": "gpt-4o-mini"
  },
  {
    "vendor": "gemini",
    "model": "gemini-2.0-flash"
  },
  {
    "vendor": "gemini",
    "model": "gemini-1.5-pro"
  }
]
```

**Note**: You can request any model name in your API calls. The router will select from configured combinations and return your original model name in the response.

### Credentials Configuration

API keys are stored in `configs/credentials.json` (not included in repository):

```json
[
  {
    "platform": "openai",
    "type": "api-key",
    "value": "sk-your-openai-api-key"
  },
  {
    "platform": "gemini", 
    "type": "api-key",
    "value": "your-gemini-api-key"
  }
]
```

## 🔍 Request Correlation & Debugging

### Request IDs
Every request receives a unique 16-character request ID that:
- Appears in the `X-Request-ID` response header
- Is included in all log entries for the request
- Remains consistent across streaming chunks
- Can be used for debugging and request tracing

**Example**:
```bash
curl -v http://localhost:8082/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model": "test", "messages": [{"role": "user", "content": "Hi"}]}'

# Response headers will include:
# X-Request-ID: 5c75cb5a3f0c3f41
```

### Error Responses
The service returns structured error responses in OpenAI-compatible format:

```json
{
  "error": {
    "type": "invalid_request_error",
    "message": "The 'messages' field is required",
    "code": "missing_required_field"
  }
}
```

**Common Error Types**:
- `invalid_request_error`: Malformed or missing required fields
- `authentication_error`: Invalid or missing API keys
- `rate_limit_error`: Vendor rate limits exceeded
- `server_error`: Internal service errors

## 🌐 Client Integration

### Python Example
```python
import openai

# Configure client to use the router
client = openai.OpenAI(
    base_url="http://localhost:8082/v1",
    api_key="dummy-key"  # Not used by router
)

# Use exactly like OpenAI
response = client.chat.completions.create(
    model="my-custom-model",
    messages=[
        {"role": "user", "content": "Hello!"}
    ]
)

print(response.choices[0].message.content)
```

### Node.js Example
```javascript
import OpenAI from 'openai';

const openai = new OpenAI({
  baseURL: 'http://localhost:8082/v1',
  apiKey: 'dummy-key' // Not used by router
});

const response = await openai.chat.completions.create({
  model: 'my-custom-model',
  messages: [
    { role: 'user', content: 'Hello!' }
  ]
});

console.log(response.choices[0].message.content);
```

### Go Example
```go
package main

import (
    "context"
    "fmt"
    "github.com/sashabaranov/go-openai"
)

func main() {
    config := openai.DefaultConfig("dummy-key")
    config.BaseURL = "http://localhost:8082/v1"
    client := openai.NewClientWithConfig(config)

    resp, err := client.CreateChatCompletion(
        context.Background(),
        openai.ChatCompletionRequest{
            Model: "my-custom-model",
            Messages: []openai.ChatCompletionMessage{
                {
                    Role:    openai.ChatMessageRoleUser,
                    Content: "Hello!",
                },
            },
        },
    )

    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    fmt.Println(resp.Choices[0].Message.Content)
}
```

## 📊 Monitoring & Observability

### Health Monitoring
```bash
# Basic health check
curl http://localhost:8082/health

# Health check with timeout
curl --max-time 5 http://localhost:8082/health
```

### Performance Monitoring
The service includes pprof endpoints for performance analysis:
- `http://localhost:8082/debug/pprof/` - Profile index
- `http://localhost:8082/debug/pprof/heap` - Memory heap profile
- `http://localhost:8082/debug/pprof/goroutine` - Goroutine profile
- `http://localhost:8082/debug/pprof/profile` - CPU profile

### Log Analysis
Logs are structured JSON by default, making them easy to parse and analyze:

```json
{
  "time": "2025-01-27T10:30:45.123Z",
  "level": "info", 
  "msg": "Processing chat completion request",
  "request_id": "5c75cb5a3f0c3f41",
  "vendor": "openai",
  "model": "gpt-4o",
  "original_model": "my-custom-model"
}
```

## 🚨 Troubleshooting

### Common Issues

1. **Connection Refused**
   - **Cause**: Service not running or wrong port
   - **Solution**: Check service status and port configuration

2. **Invalid API Key Errors**
   - **Cause**: Missing or invalid vendor API keys
   - **Solution**: Verify `configs/credentials.json` has valid keys

3. **Model Not Found**
   - **Cause**: Requested model not in `configs/models.json`
   - **Solution**: Add model configuration or use existing model

4. **Streaming Issues**
   - **Cause**: Client not handling Server-Sent Events properly
   - **Solution**: Ensure client supports SSE format

### Debug Mode
Enable debug logging for detailed request/response information:

```bash
LOG_LEVEL=DEBUG ./server
```

### Request Tracing
Use request IDs to trace requests through logs:

```bash
# Find all log entries for a specific request
grep "5c75cb5a3f0c3f41" logs/server.log
```

## 📚 Additional Resources

- **[API Reference](../api/)** - Complete OpenAPI/Swagger documentation
- **[Examples](../../examples/)** - Ready-to-use examples in multiple languages
- **[Development Guide](../development/DEVELOPMENT.md)** - For contributors and developers
- **[Main README](../../README.md)** - Project overview and quick start

---

**Need Help?** Check the [troubleshooting section](../../.cursor/rules/running_and_testing.mdc#troubleshooting) or open an issue on [GitHub](https://github.com/aashari/go-generative-api-router/issues). 