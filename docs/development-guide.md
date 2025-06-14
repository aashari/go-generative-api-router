# Development Guide

This guide provides essential information for developers working on the Generative API Router project.

> **📚 Complete Guides**: For comprehensive development workflows, testing procedures, and deployment instructions, see the detailed guides in [`.cursor/rules/`](../.cursor/rules/):
> - **[Development Guide](../.cursor/rules/development_guide.mdc)** - Complete workflow, architecture, Git practices
> - **[Running & Testing Guide](../.cursor/rules/running_and_testing.mdc)** - Setup, testing, debugging
>
> **⚠️ Note**: This document provides a quick overview. For detailed development procedures, always refer to the comprehensive guides above.

## 🚀 Quick Start

### Prerequisites
- **Go 1.21+** - [Download](https://golang.org/dl/)
- **Docker & Docker Compose** (optional) - [Install](https://docs.docker.com/get-docker/)
- **golangci-lint** - [Install](https://golangci-lint.run/usage/install/)
- **Make** - For build automation

### Initial Setup
```bash
# 1. Clone and navigate
git clone https://github.com/aashari/go-generative-api-router.git
cd go-generative-api-router

# 2. Run setup (installs dependencies, creates config files)
make setup

# 3. Configure API keys
cp configs/credentials.json.example configs/credentials.json
# Edit configs/credentials.json with your API keys

# 4. Build and run
make build
make run
```

## 🔄 Development Workflow

### Daily Development Commands
```bash
# Development with auto-reload
make run-dev

# Run tests frequently
make test

# Format code before committing
make format

# Lint before committing
make lint

# Full CI check (format, lint, test, build)
make ci-check
```

### Project Structure
```
├── cmd/server/          # Application entry point
├── internal/            # Private application code
│   ├── app/            # Application configuration
│   ├── proxy/          # Core proxy logic
│   ├── selector/       # Vendor selection strategies
│   ├── validator/      # Request validation
│   ├── handlers/       # HTTP handlers
│   └── ...
├── configs/            # Configuration files
├── examples/           # Usage examples
├── docs/              # Documentation
└── deployments/       # Docker and deployment files
```

## 🧪 Testing

### Running Tests
```bash
# All tests
make test

# With coverage
make test-coverage

# Specific package
go test ./internal/handlers

# With race detection
go test -race ./...

# Verbose output
go test -v ./...
```

### Test Structure
- **Unit tests**: Colocated with source files (`*_test.go`)
- **Test fixtures**: `testdata/fixtures/`
- **Test utilities**: `testdata/analysis/`
- **Coverage target**: >80%

### Writing Tests
- Use table-driven tests for multiple scenarios
- Mock external dependencies (API calls, etc.)
- Test both success and error cases
- Include edge cases and boundary conditions

## 🏗️ Architecture Overview

### Core Components

1. **Proxy Handler** (`internal/proxy/`)
   - Routes requests to selected vendors
   - Handles streaming and non-streaming responses
   - Maintains transparent proxy behavior

2. **Vendor Selector** (`internal/selector/`)
   - Implements even distribution selection strategy
   - Manages vendor-credential-model combinations
   - Supports vendor filtering via query parameters

3. **Request Validator** (`internal/validator/`)
   - Validates incoming OpenAI-compatible requests
   - Extracts and preserves original model names
   - Ensures request structure compliance

4. **Response Processor** (`internal/proxy/`)
   - Processes vendor responses
   - Maintains model name transparency
   - Handles both streaming and non-streaming formats

### Key Principles

- **Transparent Proxy**: Original model names preserved in responses
- **Vendor Agnostic**: Unified interface regardless of backend vendor
- **Fair Distribution**: Even probability across all vendor-model combinations
- **OpenAI Compatibility**: 100% compatible with OpenAI API format

## 🔧 Configuration

### Credentials (`configs/credentials.json`)
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

### Models (`configs/models.json`)
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

## 📝 Structured Logging

The service uses a structured logging system based on Go's `log/slog` package:

```go
import "github.com/aashari/go-generative-api-router/internal/logger"

// Basic logging
logger.Info("Operation completed", "key", value)

// Context-aware logging (includes request_id)
logger.InfoCtx(ctx, "Request processed", "status", "success")

// Error logging
if err != nil {
    logger.Error("Operation failed", "error", err)
}
```

**Complete documentation**: [Logging Guide](logging-guide.md)

## 🐳 Docker Development

### Local Development
```bash
# Build and run with Docker Compose
make docker-build
make docker-run

# Stop services
make docker-stop

# View logs
docker-compose -f deployments/docker/docker-compose.yml logs -f
```

### Production Deployment
```bash
# Build for production
docker build -f deployments/docker/Dockerfile -t genapi-router .

# Run with environment variables
docker run -p 8082:8082 \
  -e LOG_LEVEL=INFO \
  -e LOG_FORMAT=json \
  genapi-router
```

## 🔍 Debugging

### Common Issues
1. **Port conflicts**: Check if port 8082 is in use (`lsof -i :8082`)
2. **API key errors**: Verify credentials in `configs/credentials.json`
3. **Build failures**: Run `make clean` then `make build`
4. **Test failures**: Check for race conditions with `go test -race`

### Debugging Tools
- **Health check**: `curl http://localhost:8082/health`
- **Logs**: Check `logs/server.log` or console output
- **Profiling**: Available at `/debug/pprof/` endpoints
- **Request tracing**: Each request gets a unique ID in logs and headers

## 📋 Code Quality

### Standards
- **Go formatting**: Use `gofmt` (automated via `make format`)
- **Linting**: Pass `golangci-lint` checks (run via `make lint`)
- **Testing**: Maintain >80% test coverage
- **Documentation**: Document public APIs and complex logic
- **Error handling**: Use structured error types from `internal/errors`

### Pre-commit Checklist
- [ ] `make format` - Code formatted
- [ ] `make lint` - No linting errors
- [ ] `make test` - All tests pass
- [ ] `make build` - Builds successfully
- [ ] Manual testing of changed functionality

## 🚀 Deployment

### Local Testing
```bash
# Start service
make run

# Test endpoints
curl http://localhost:8082/health
curl http://localhost:8082/v1/models
```

### Production Deployment
The project is deployed on AWS as the `go-generative-api-router` service. See the [Deployment Guide](deployment-guide.md) for comprehensive AWS infrastructure documentation and deployment procedures.

**Quick Deployment Status Check**:
```bash
# Check production service health
curl https://genapi.example.com/health

# Check AWS deployment status
aws --profile ${AWS_ACCOUNT_ID} --region ap-southeast-3 ecs describe-services \
  --cluster prod-${SERVICE_NAME} --services prod-${SERVICE_NAME}
```

## 📚 Additional Resources

- **[Contributing Guide](contributing-guide.md)** - How to contribute
- **[Testing Guide](testing-guide.md)** - Detailed testing information  
- **[Logging Guide](logging-guide.md)** - Complete logging system docs
- **[Deployment Guide](deployment-guide.md)** - AWS infrastructure and deployment procedures
- **[API Reference](api-reference.md)** - Complete API documentation
- **[User Guide](user-guide.md)** - Service usage documentation
- **[Examples](../examples/)** - Usage examples in multiple languages

---

**Need Help?** 
- Check the [complete development guide](../.cursor/rules/development_guide.mdc)
- Review [troubleshooting section](../.cursor/rules/running_and_testing.mdc#troubleshooting)
- Open an issue on [GitHub](https://github.com/aashari/go-generative-api-router/issues)