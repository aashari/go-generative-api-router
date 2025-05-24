# Generative API Router

[![Go Report Card](https://goreportcard.com/badge/github.com/aashari/go-generative-api-router)](https://goreportcard.com/report/github.com/aashari/go-generative-api-router)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/aashari/go-generative-api-router)](https://github.com/aashari/go-generative-api-router)

A Go microservice that proxies OpenAI-compatible API calls to multiple LLM vendors (OpenAI, Gemini) using configurable selection strategies. This router simplifies integration with AI services by providing a unified interface while handling the complexity of multi-vendor management.

<!-- 
<div align="center">
  <img src="https://raw.githubusercontent.com/aashari/go-generative-api-router/main/docs/assets/architecture-diagram.png" alt="Architecture Diagram" width="800">
</div>
-->

## Features

- **Multi-Vendor Support**: Routes requests to OpenAI or Gemini using OpenAI API compatibility
- **Random Selection**: Automatically distributes requests across configured vendors and models
- **Vendor Filtering**: Supports explicit vendor selection via `?vendor=` query parameter
- **Transparent Proxy**: Maintains all original request/response data (except for model selection)
- **Streaming Support**: Properly handles chunked streaming responses for real-time applications
- **Tool Calling**: Supports function calling/tools for AI agents with proper validation
- **Modular Design**: Clean separation of concerns with selector, validator, and client components
- **Configuration Driven**: Easily configure available models and credentials via JSON files
- **Metrics & Monitoring**: Built-in Prometheus metrics and health check endpoints
- **Comprehensive Testing**: Full test coverage with unit tests for all components

## Quick Start

### Prerequisites

- Go 1.21 or higher
- API keys for OpenAI and/or Google Gemini
- Make (for build automation)

### Installation

1. **Clone the Repository**:
   ```bash
   git clone https://github.com/aashari/go-generative-api-router.git
   cd go-generative-api-router
   ```

2. **Setup Environment**:
   ```bash
   make setup
   ```
   This will:
   - Download Go dependencies
   - Install development tools
   - Create `configs/credentials.json` from the example template

3. **Configure Credentials**:
   Edit `configs/credentials.json` with your API keys:
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

4. **Configure Models**:
   Edit `configs/models.json` to define which vendor-model pairs can be selected:
   ```json
   [
     {
       "vendor": "gemini",
       "model": "gemini-2.0-flash"
     },
     {
       "vendor": "openai",
       "model": "gpt-4o"
     }
   ]
   ```

5. **Run the Service**:
   ```bash
   make run
   ```
   
   The service will be available at http://localhost:8082

## Usage

### Using Example Scripts

Example scripts are provided for common use cases:

```bash
# Basic usage examples
./examples/curl/basic.sh

# Streaming examples
./examples/curl/streaming.sh

# Tool calling examples
./examples/curl/tools.sh
```

### Client Libraries

Example implementations are available for multiple languages:

- **Python**: `examples/clients/python/client.py`
- **Node.js**: `examples/clients/nodejs/client.js`
- **Go**: `examples/clients/go/client.go`

### Docker Deployment

Build and run using Docker:

```bash
# Build and run with Docker Compose
make docker-build
make docker-run

# Stop the service
make docker-stop
```

Or manually:

```bash
docker-compose -f deployments/docker/docker-compose.yml up --build
```

## API Reference

### Health Check

```http
GET /health
```

**Response**: `200 OK` with body `OK` if the service is running properly.

### Metrics

```http
GET /metrics
```

**Response**: Prometheus-formatted metrics including request counts, durations, and error rates.

### Models Listing

```http
GET /v1/models
GET /v1/models?vendor=openai
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
POST /v1/chat/completions
POST /v1/chat/completions?vendor=gemini
```

**Basic Example**:
```bash
curl -X POST http://localhost:8082/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "any-model",
    "messages": [{"role": "user", "content": "Hello, how are you?"}]
  }'
```

#### Stream Support

Enable streaming responses by adding `"stream": true` to your request:

```bash
curl -X POST http://localhost:8082/v1/chat/completions \
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
curl -X POST http://localhost:8082/v1/chat/completions \
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

## Architecture

The project follows a modular design with clear separation of concerns:

```
generative-api-router/
├── cmd/server/          # Application entry point
├── configs/             # Configuration files
│   ├── credentials.json # API keys (gitignored)
│   └── models.json      # Vendor-model mappings
├── deployments/         # Deployment configurations
│   └── docker/          # Docker files
├── docs/                # Documentation
│   ├── api/             # API documentation
│   ├── development/     # Development guides
│   └── user/            # User guides
├── examples/            # Usage examples
│   ├── curl/            # cURL examples
│   └── clients/         # Client library examples
├── internal/            # Core application code
│   ├── app/             # Application initialization
│   ├── config/          # Configuration management
│   ├── errors/          # Error handling
│   ├── filter/          # Filtering utilities
│   ├── handlers/        # HTTP handlers
│   ├── monitoring/      # Metrics collection
│   ├── proxy/           # Proxy functionality
│   ├── router/          # Route definitions
│   ├── selector/        # Vendor/model selection
│   └── validator/       # Request validation
├── scripts/             # Operational scripts
├── testdata/            # Test fixtures and analysis
└── Makefile             # Build automation
```

## Development

### Building

```bash
# Build the application
make build

# Build Docker image
make docker-build
```

### Testing

Run the comprehensive test suite:

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run tests with race detection
go test -race ./...
```

### Code Quality

```bash
# Format code
make format

# Run linter
make lint

# Clean build artifacts
make clean
```

### Development Mode

```bash
# Run without building (using go run)
make run-dev

# Run with logging to file
make run-with-logs
```

## Configuration

### Environment Variables

The service supports the following environment variables:

- `PORT`: Server port (default: 8082)
- `LOG_LEVEL`: Logging level (default: info)

### Configuration Files

- `configs/credentials.json`: API keys for vendors
- `configs/models.json`: Available models and their vendors

## Security Considerations

- API keys are stored in `configs/credentials.json` which is gitignored
- For production environments, consider using:
  - Environment variables for sensitive data
  - Secret management solutions (AWS Secrets Manager, HashiCorp Vault)
  - Encrypted configuration files
- Consider implementing rate limiting for production deployments
- Use HTTPS in production environments

## Contributing

Contributions are welcome! Please see our [Contributing Guide](docs/development/CONTRIBUTING.md) for details.

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Commit your changes: `git commit -am 'Add new feature'`
4. Push to the branch: `git push origin feature/my-feature`
5. Submit a pull request

## Documentation

- [Development Guide](docs/development/DEVELOPMENT.md)
- [Testing Guide](docs/development/TESTING.md)
- [API Documentation](docs/api/)
- [User Guide](docs/user/README.md)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- This project was inspired by the need for a unified interface to multiple LLM providers
- Special thanks to the Go community for the excellent libraries and tools
- Built with modern Go best practices and clean architecture principles
