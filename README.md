# Generative API Router

[![Go Report Card](https://goreportcard.com/badge/github.com/aashari/go-generative-api-router)](https://goreportcard.com/report/github.com/aashari/go-generative-api-router)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/aashari/go-generative-api-router)](https://github.com/aashari/go-generative-api-router)

A production-ready Go microservice that provides a **unified OpenAI-compatible API** for multiple LLM vendors (OpenAI, Gemini). This transparent proxy router simplifies AI integration by offering a single interface while intelligently distributing requests across multiple vendors and preserving your original model names in responses.

<!-- 
<div align="center">
  <img src="https://raw.githubusercontent.com/aashari/go-generative-api-router/main/docs/assets/architecture-diagram.png" alt="Architecture Diagram" width="800">
</div>
-->

## 🏗️ **Architecture Overview**

### **Multi-Vendor OpenAI-Compatible Router**

This service acts as a **transparent proxy** that provides a unified OpenAI-compatible API interface while routing requests to multiple LLM vendors behind the scenes:

- **OpenAI API Compatibility**: All vendors accessed through OpenAI-compatible endpoints
- **Transparent Model Handling**: Preserves your original model names in responses
- **Multi-Vendor Design**: Currently supports **19 credentials** (18 Gemini + 1 OpenAI) with **4 models**
- **Even Distribution**: Fair selection across **114 vendor-credential-model combinations**

### **Enterprise-Grade Features (2024)**

Recent comprehensive improvements include:

- 🔒 **Security**: AES-GCM encryption for credentials, sensitive data masking in logs
- 🔄 **Reliability**: Exponential backoff retry logic, circuit breaker pattern implementation
- 📊 **Monitoring**: Comprehensive health checks with vendor connectivity monitoring
- ⚡ **Performance**: Production-optimized logging, conditional detail levels
- 🧹 **Code Quality**: DRY principles, centralized utilities, eliminated code duplication

## Features

- **Multi-Vendor Support**: Routes requests to OpenAI or Gemini using OpenAI API compatibility
- **Even Distribution Selection**: Fair distribution across all vendor-credential-model combinations
- **Vendor Filtering**: Supports explicit vendor selection via `?vendor=` query parameter
- **Transparent Proxy**: Maintains all original request/response data (except for model selection)
- **Streaming Support**: Properly handles chunked streaming responses for real-time applications
- **Tool Calling**: Supports function calling/tools for AI agents with proper validation
- **Enterprise Reliability**: Circuit breakers, retry logic, comprehensive health monitoring
- **Security**: Encrypted credential storage, sensitive data masking
- **Modular Design**: Clean separation of concerns with selector, validator, and client components
- **Configuration Driven**: Easily configure available models and credentials via JSON files
- **Health Check**: Built-in health check endpoint with service status monitoring
- **Comprehensive Testing**: Full test coverage with unit tests for all components
- **🌐 Public Image URL Support**: Automatic downloading and conversion of public image URLs to base64

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

3. **Verify Configuration**:
   ```bash
   # Check existing configuration (service likely has working credentials)
   cat configs/credentials.json | jq length && echo "credentials configured"
   cat configs/models.json | jq length && echo "models configured"
   ```

4. **Configure Credentials** (if needed):
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

5. **Configure Models** (if needed):
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

6. **Run the Service**:
   ```bash
   make run
   ```
   
   The service will be available at http://localhost:8082

## Selection Strategy

The router uses an **Even Distribution Selector** that ensures fair distribution across all vendor-credential-model combinations. This approach provides true fairness where each combination has exactly equal probability of being selected.

### How It Works

1. **Combination Generation**: The system creates a flat list of all valid vendor-credential-model combinations
2. **Equal Probability**: Each combination gets exactly `1/N` probability where N = total combinations
3. **Fair Distribution**: Unlike traditional two-stage selection (vendor → model), this ensures no bias toward vendors with fewer models

### Example Distribution

With the current configuration:
- **18 Gemini credentials** × **6 models** = 108 combinations
- **1 OpenAI credential** × **6 models** = 6 combinations
- **Total**: 114 combinations

Each combination has exactly **1/114 = 0.877%** probability:
- **Gemini overall**: 108/114 = 94.7%
- **OpenAI overall**: 6/114 = 5.3%

This reflects the actual resource availability rather than artificial vendor-level balancing.

### Benefits

- ✅ **True Fairness**: Each credential-model combination has exactly equal probability
- ✅ **Resource Proportional**: Distribution reflects actual available resources
- ✅ **Scalable**: Automatically adapts as credentials/models are added/removed
- ✅ **Transparent**: Clear logging shows selection and total combination count
- ✅ **No Bias**: Eliminates bias toward vendors with fewer models per credential

### Monitoring Selection

The service logs each selection decision for transparency:

```
Even distribution selected combination - Vendor: openai, Model: gpt-4o (from 114 total combinations)
```

You can monitor the distribution by checking the server logs to verify fair selection across all combinations.

## Usage

### Basic API Usage

```bash
# Health check
curl http://localhost:8082/health

# List available models
curl http://localhost:8082/v1/models

# Chat completion (any model name)
curl -X POST http://localhost:8082/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "my-preferred-model",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'

# Force specific vendor
curl -X POST "http://localhost:8082/v1/chat/completions?vendor=openai" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "my-model",
    "messages": [{"role": "user", "content": "Hello from OpenAI!"}]
  }'
```

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

## Testing

### Multi-Vendor Testing

**IMPORTANT**: This is a multi-vendor service. Always test both vendors:

```bash
# Test OpenAI vendor
curl -X POST "http://localhost:8082/v1/chat/completions?vendor=openai" \
  -H "Content-Type: application/json" \
  -d '{"model": "test-openai", "messages": [{"role": "user", "content": "Hello"}]}'

# Test Gemini vendor  
curl -X POST "http://localhost:8082/v1/chat/completions?vendor=gemini" \
  -H "Content-Type: application/json" \
  -d '{"model": "test-gemini", "messages": [{"role": "user", "content": "Hello"}]}'

# Monitor vendor distribution
grep "Even distribution selected combination" logs/server.log | tail -5
```

### Development Testing

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Full CI check
make ci-check
```

## Documentation

### 📚 **Complete Documentation**

- **[Documentation Index](docs/README.md)** - Complete documentation roadmap
- **[User Guide](docs/user-guide.md)** - API usage and integration guide
- **[API Reference](docs/api-reference.md)** - Complete API documentation
- **[Development Guide](docs/development-guide.md)** - Setup and development workflow

### 🔧 **Development Guides**

- **[Contributing Guide](docs/contributing-guide.md)** - How to contribute to the project
- **[Testing Guide](docs/testing-guide.md)** - Testing strategies and procedures
- **[Logging Guide](docs/logging-guide.md)** - Comprehensive logging documentation
- **[Deployment Guide](docs/deployment-guide.md)** - AWS infrastructure and deployment

### 📖 **Cursor AI Context**

For Cursor AI development, see the comprehensive guides in `.cursor/rules/`:
- **[Development Guide](.cursor/rules/development_guide.mdc)** - Complete workflow, architecture, Git practices
- **[Running & Testing Guide](.cursor/rules/running_and_testing.mdc)** - Setup, testing, debugging

## Architecture

### Core Components

- **Proxy Handler**: Routes requests to selected vendors, handles streaming/non-streaming responses
- **Vendor Selector**: Implements even distribution selection across vendor-credential-model combinations
- **Request Validator**: Validates OpenAI-compatible requests, preserves original model names
- **Response Processor**: Processes vendor responses while maintaining model name transparency
- **Health Monitor**: Comprehensive health checks with vendor connectivity monitoring
- **Circuit Breaker**: Reliability pattern implementation for vendor communication
- **Retry Logic**: Exponential backoff for failed vendor requests

### Key Principles

1. **Transparent Proxy**: Original model names preserved in responses
2. **Vendor Agnostic**: Unified interface regardless of backend vendor  
3. **Fair Distribution**: Even probability across all vendor-model combinations
4. **OpenAI Compatibility**: 100% compatible with OpenAI API format
5. **Enterprise Reliability**: Circuit breakers, retries, comprehensive monitoring

## Production Deployment

The service is production-ready with:

- **AWS ECS Deployment**: Containerized deployment on AWS
- **Load Balancing**: High availability with load balancer integration
- **Monitoring**: CloudWatch integration and comprehensive logging
- **Security**: Encrypted credentials, sensitive data masking
- **Reliability**: Circuit breakers, retry logic, health monitoring

See [Deployment Guide](docs/deployment-guide.md) for complete deployment instructions.

## Contributing

We welcome contributions! Please see our [Contributing Guide](docs/contributing-guide.md) for details on:

- Development setup and workflow
- Code standards and review process
- Testing requirements
- Pull request guidelines

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- **Documentation**: [Complete documentation](docs/README.md)
- **Issues**: [GitHub Issues](https://github.com/aashari/go-generative-api-router/issues)
- **Discussions**: [GitHub Discussions](https://github.com/aashari/go-generative-api-router/discussions)

---

**Need help?** Check the [documentation](docs/README.md) or open an issue on GitHub.