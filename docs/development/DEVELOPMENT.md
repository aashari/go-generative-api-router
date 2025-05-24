# Development Guide

## Prerequisites
- Go 1.21 or higher
- Docker and Docker Compose (optional)
- golangci-lint (for linting)

## Quick Start
1. Clone the repository
2. Run setup: `make setup`
3. Configure API keys in `configs/credentials.json`
4. Build: `make build`
5. Run: `make run`

## Development Workflow
- Use `make run-dev` for development (auto-recompiles)
- Run tests frequently: `make test`
- Format code: `make format`
- Lint before committing: `make lint`

## Structured Logging

The service uses a structured logging system based on Go's `log/slog` package. Basic usage:

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

For complete documentation on the logging system, see [LOGGING.md](./LOGGING.md).

See [CONTRIBUTING.md](./CONTRIBUTING.md) for contribution guidelines. 