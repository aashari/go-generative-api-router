# User Guide

This guide provides comprehensive information for users who want to integrate with and use the Generative API Router service.

> **📖 Quick Start**: For project overview and initial setup, see the [Main README](../README.md).  
> **🔧 Development**: For contributing and development information, see [Contributing Guide](contributing-guide.md).

## 🚀 Getting Started

### Service Overview
The Generative API Router is a transparent proxy that routes OpenAI-compatible API calls to multiple LLM vendors (OpenAI, Gemini) while maintaining complete API compatibility. The service preserves your original model names in responses while intelligently selecting from available vendor-model combinations.

### Prerequisites
- The service running on your target host (default: `localhost:8082`)
- Valid API keys for at least one supported vendor (OpenAI or Gemini)
- Any HTTP client or OpenAI-compatible SDK

## 📊 Service Configuration

### Environment Setup
The service uses environment variables for configuration. Create a `.env` file in the project root:

```bash
# Server Configuration
PORT=8082
LOG_LEVEL=info

# API Keys (configure in configs/credentials.json instead)
GENAPI_API_KEY=your-api-key-here
```

### API Keys Configuration
Configure your vendor API keys in `configs/credentials.json`:

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

### Models Configuration
Configure available models in `configs/models.json`:

```json
[
  {
    "vendor": "openai",
    "model": "gpt-4o"
  },
  {
    "vendor": "gemini", 
    "model": "gemini-2.0-flash"
  }
]
```

## 🔌 API Usage

### Base URL
- **Local Development**: `http://localhost:8082`
- **Production**: Your deployed service URL

### Authentication
The service uses API key authentication. Include your API key in the Authorization header:

```bash
Authorization: Bearer YOUR_API_KEY
```

### Endpoints

#### Health Check
```bash
GET /health
```

**Response:**
```json
{
  "status": "OK",
  "timestamp": "2024-01-27T10:30:45.123Z"
}
```

#### List Models
```bash
GET /v1/models
```

**Response:**
```json
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

#### Chat Completions
```bash
POST /v1/chat/completions
```

**Request:**
```json
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

**Response:**
```json
{
  "id": "chatcmpl-abc123",
  "object": "chat.completion",
  "created": 1234567890,
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

## 🔧 Advanced Features

### Vendor Selection
Force a specific vendor by adding a query parameter:

```bash
POST /v1/chat/completions?vendor=openai
POST /v1/chat/completions?vendor=gemini
```

### Streaming Responses
Enable streaming by setting `"stream": true` in your request:

```json
{
  "model": "your-model",
  "messages": [...],
  "stream": true
}
```

Streaming responses follow the Server-Sent Events format with `data:` prefixed JSON chunks.

### Tool Calling
The service supports OpenAI-compatible tool calling:

```json
{
  "model": "your-model",
  "messages": [...],
  "tools": [
    {
      "type": "function",
      "function": {
        "name": "get_weather",
        "description": "Get weather information",
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

## 📚 Client Integration

### cURL Examples
See the [examples/curl/](../examples/curl/) directory for ready-to-use cURL scripts:

- `basic.sh` - Basic chat completion
- `streaming.sh` - Streaming responses
- `tools.sh` - Tool calling examples

### SDK Integration

#### Python (OpenAI SDK)
```python
import openai

client = openai.OpenAI(
    api_key="your-api-key",
    base_url="http://localhost:8082/v1"
)

response = client.chat.completions.create(
    model="your-preferred-model-name",
    messages=[
        {"role": "user", "content": "Hello!"}
    ]
)

print(response.choices[0].message.content)
```

#### Node.js (OpenAI SDK)
```javascript
import OpenAI from 'openai';

const openai = new OpenAI({
    apiKey: 'your-api-key',
    baseURL: 'http://localhost:8082/v1',
});

const response = await openai.chat.completions.create({
    model: 'your-preferred-model-name',
    messages: [
        { role: 'user', content: 'Hello!' }
    ],
});

console.log(response.choices[0].message.content);
```

#### Go
```go
package main

import (
    "context"
    "fmt"
    "github.com/sashabaranov/go-openai"
)

func main() {
    config := openai.DefaultConfig("your-api-key")
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

    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    fmt.Println(resp.Choices[0].Message.Content)
}
```

## 🔍 Troubleshooting

### Common Issues

#### Connection Refused
- Verify the service is running: `curl http://localhost:8082/health`
- Check the correct port is being used
- Ensure no firewall blocking the connection

#### Authentication Errors
- Verify your API key is correctly set in the Authorization header
- Check that the API key is valid and has appropriate permissions
- Ensure the service has valid vendor API keys configured

#### Model Not Found
- Any model name is accepted - the service routes to available vendors
- Check that at least one vendor is configured with valid credentials
- Verify `configs/models.json` contains valid model configurations

#### Rate Limiting
- The service respects vendor rate limits
- Implement appropriate retry logic with exponential backoff
- Consider distributing load across multiple vendor accounts

### Getting Help

1. Check the [Development Guide](development-guide.md) for setup issues
2. Review the [API Reference](api-reference.md) for detailed API documentation
3. Examine service logs for error details
4. Consult the [examples/](../examples/) directory for working code samples

## 📈 Best Practices

### Performance Optimization
- Use appropriate `max_tokens` values to avoid unnecessary costs
- Implement client-side caching for repeated similar requests
- Use streaming for long responses to improve perceived performance

### Error Handling
- Always implement proper error handling for API calls
- Check response status codes and error messages
- Implement retry logic with exponential backoff for transient failures

### Security
- Never expose API keys in client-side code
- Use environment variables or secure secret management
- Implement rate limiting on your client side if needed
- Validate and sanitize user inputs before sending to the API

### Cost Management
- Monitor usage across different vendors
- Use appropriate temperature and max_tokens settings
- Consider implementing usage quotas for your users
- Log and analyze API usage patterns

---

For more detailed information, see:
- [API Reference](api-reference.md) - Complete API specification
- [Development Guide](development-guide.md) - Contributing and development setup
- [Examples](../examples/) - Ready-to-use code examples