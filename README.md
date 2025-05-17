# Generative API Router

[![Go Report Card](https://goreportcard.com/badge/github.com/aashari/generative-api-router)](https://goreportcard.com/report/github.com/aashari/generative-api-router)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/aashari/generative-api-router)](https://github.com/aashari/generative-api-router)

A Go microservice that proxies OpenAI-compatible API calls to multiple LLM vendors (OpenAI, Gemini) using configurable selection strategies. This router simplifies integration with AI services by providing a unified interface while handling the complexity of multi-vendor management.

<!-- 
<div align="center">
  <img src="https://raw.githubusercontent.com/aashari/generative-api-router/main/docs/assets/architecture-diagram.png" alt="Architecture Diagram" width="800">
</div>
-->

## 🌟 Features

- **🔄 Multi-Vendor Support**: Routes requests to OpenAI or Gemini using OpenAI API compatibility
- **🎲 Random Selection**: Automatically distributes requests across configured vendors and models
- **🔎 Vendor Filtering**: Supports explicit vendor selection via `?vendor=` query parameter
- **🔍 Transparent Proxy**: Maintains all original request/response data (except for model selection)
- **⚡ Streaming Support**: Properly handles chunked streaming responses for real-time applications
- **🛠️ Tool Calling**: Supports function calling/tools for AI agents with proper validation
- **📦 Modular Design**: Clean separation of concerns with selector, validator, and client components
- **⚙️ Configuration Driven**: Easily configure available models and credentials via JSON files

## 🚀 Quick Start

### Prerequisites

- Go 1.22 or higher
- API keys for OpenAI and/or Google Gemini

### Installation

1. **Clone the Repository**:
   ```bash
   git clone https://github.com/aashari/generative-api-router.git
   cd generative-api-router
   ```

2. **Configure Credentials**:
   Copy the example file and edit with valid API keys:
   ```bash
   cp credentials.json.example credentials.json
   ```
   
   Then edit `credentials.json` with valid API keys:
   ```json
   [
     {
       "platform": "openai",
       "type": "api-key",
       "value": "sk-your-openai-key"
     },
     {
       "platform": "gemini",
       "type": "api-key",
       "value": "your-gemini-key"
     }
   ]
   ```

3. **Configure Models**:
   Edit `models.json` to define which vendor-model pairs can be selected:
   ```json
   [
     {
       "vendor": "gemini",
       "model": "gemini-1.5-flash"
     },
     {
       "vendor": "openai",
       "model": "gpt-4o"
     }
   ]
   ```

4. **Run Locally**:
   ```bash
   go mod tidy
   go run ./cmd/server
   ```
   
   The service will be available at http://localhost:8082

## 🐳 Docker Deployment

Build and run using Docker Compose:

```bash
docker-compose up --build
```

## 🔌 API Reference

### Health Check

```http
GET /health
```

**Response**: `200 OK` with body `OK` if the service is running properly.

### Models Listing

```http
GET /models
GET /models?vendor=openai
```

**Example Response**:
```json
{
  "object": "list",
  "data": [
    {
      "id": "gpt-4o",
      "object": "model",
      "created": 1715929200,
      "owned_by": "openai"
    }
  ]
}
```

### Chat Completions

```http
POST /chat/completions
POST /chat/completions?vendor=gemini
```

**Basic Example**:
```bash
curl -X POST http://localhost:8082/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "any-model",
    "messages": [{"role": "user", "content": "Hello, how are you?"}]
  }'
```

#### Stream Support

Enable streaming responses by adding `"stream": true` to your request:

```bash
curl -X POST http://localhost:8082/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "any-model",
    "messages": [{"role": "user", "content": "Write a short poem"}],
    "stream": true
  }'
```

#### Tool Calling

Leverage function calling for more advanced use cases:

```bash
curl -X POST http://localhost:8082/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "any-model",
    "messages": [{"role": "user", "content": "What is the weather in Boston?"}],
    "tools": [{
      "type": "function",
      "function": {
        "name": "get_weather",
        "description": "Get weather information for a location",
        "parameters": {
          "type": "object",
          "properties": {
            "location": {"type": "string", "description": "City name"}
          },
          "required": ["location"]
        }
      }
    }],
    "tool_choice": "auto"
  }'
```

## 🏗️ Architecture

The project follows a modular design with clear separation of concerns:

```
generative-api-router/
├── cmd/server/          # Application entry point
├── internal/
│   ├── app/             # Application core and HTTP handlers
│   ├── config/          # Configuration management
│   ├── proxy/           # API client and proxy functionality
│   ├── selector/        # Vendor/model selection strategies
│   └── validator/       # Request validation
└── models.json          # Vendor-model configuration
```

## 🛠️ Development

### Building

```bash
go build -o generative-api-router ./cmd/server
```

### Testing

Test basic functionality:
```bash
# Health check
curl -X GET http://localhost:8082/health

# List models
curl -X GET http://localhost:8082/models

# Basic completion
curl -X POST http://localhost:8082/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model": "any-model", "messages": [{"role": "user", "content": "Hello"}]}'
```

## 🔒 Security Considerations

- The current implementation stores API keys in plain text in `credentials.json`. 
- For production environments, consider using environment variables or a secret management solution.
- The `credentials.json` file is included in `.gitignore` to prevent accidentally committing API keys.
- Consider implementing rate limiting for production deployments.

## 🤝 Contributing

Contributions are welcome! Here's how you can contribute:

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Commit your changes: `git commit -am 'Add new feature'`
4. Push to the branch: `git push origin feature/my-feature`
5. Submit a pull request

## 📜 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- This project was inspired by the need for a unified interface to multiple LLM providers.
- Special thanks to the Go community for the excellent libraries and tools.
