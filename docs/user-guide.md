# User Guide

This guide provides comprehensive information for users who want to integrate with and use the Generative API Router service.

> **📖 Quick Start**: For project overview and initial setup, see the [Main README](../README.md).  
> **🔧 Development**: For contributing and development information, see [Contributing Guide](contributing-guide.md).  
> **📋 API Documentation**: For complete API specifications, see [API Reference](api-reference.md).

## 🚀 Getting Started

### Service Overview
The Generative API Router is a transparent proxy that routes OpenAI-compatible API calls to multiple LLM vendors (OpenAI, Gemini) while maintaining complete API compatibility. The service preserves your original model names in responses while intelligently selecting from available vendor-model combinations.

**Key Benefits:**
- **Vendor Agnostic**: Use any model name - the service handles vendor selection
- **Transparent**: Your original model names are preserved in responses
- **Multi-Vendor**: Automatic load distribution across multiple vendors
- **OpenAI Compatible**: Works with existing OpenAI SDKs and tools

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

## 🔌 Integration Patterns

### Service Endpoints
The service provides OpenAI-compatible endpoints:

- **Base URL**: `http://localhost:8082` (local) or your deployed service URL
- **Health Check**: `GET /health` - Check service status
- **List Models**: `GET /v1/models` - List available models (accepts any model name)
- **Chat Completions**: `POST /v1/chat/completions` - Main AI interaction endpoint

> **📋 Complete API Documentation**: See [API Reference](api-reference.md) for detailed endpoint specifications, request/response formats, and examples.

### Authentication
Authentication is optional for development. For production, include your API key:

```bash
Authorization: Bearer YOUR_API_KEY
```

## 🔧 Core Features

### Multi-Vendor Intelligence
The service automatically distributes requests across multiple vendors:

- **Automatic Selection**: Service chooses optimal vendor-model combinations
- **Forced Vendor Selection**: Use `?vendor=openai` or `?vendor=gemini` query parameters
- **Load Distribution**: Even distribution across all configured vendor credentials
- **Model Name Preservation**: Your requested model name is always returned in responses

### Advanced Capabilities

**Streaming Support**: Enable real-time responses with `"stream": true`

**File Processing**: Automatic document and image processing from URLs
- Supports PDF, Word, Excel, PowerPoint, images, and more
- Custom headers for protected files (authentication, user-agent, etc.)
- Multiple files in a single request
- Automatic format detection and conversion

**Tool Calling**: Full OpenAI-compatible function calling support

**Multi-Modal**: Text, images, and documents in the same conversation

> **📋 Detailed Examples**: See [API Reference](api-reference.md) for complete request/response examples and specifications for all features.

## 📚 Client Integration

### Ready-to-Use Examples
See the [examples/](../examples/) directory for complete working examples:

- **cURL**: `basic.sh`, `streaming.sh`, `tools.sh` for command-line testing
- **Python**: OpenAI SDK integration with file processing examples
- **Node.js**: JavaScript/TypeScript integration patterns
- **Go**: Golang client implementation

### OpenAI SDK Compatibility

The service is fully compatible with existing OpenAI SDKs. Simply change the base URL:

**Python:**
```python
import openai

client = openai.OpenAI(
    api_key="not-required",  # Optional for development
    base_url="http://localhost:8082/v1"
)
```

**Node.js:**
```javascript
const openai = new OpenAI({
    apiKey: 'not-required',  // Optional for development
    baseURL: 'http://localhost:8082/v1',
});
```

**Go:**
```go
config := openai.DefaultConfig("not-required")  // Optional for development
config.BaseURL = "http://localhost:8082/v1"
client := openai.NewClientWithConfig(config)
```

> **💡 Pro Tip**: Any existing OpenAI code will work immediately - just change the base URL!

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