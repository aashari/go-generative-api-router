# Logging System

This document details the structured logging system used in the Generative API Router service.

## Overview

The service implements an enterprise-grade structured logging system based on Go's `log/slog` package. Key features include:

- **Structured JSON Output**: Machine-parsable logs ideal for aggregation systems
- **Request Correlation**: Unique request IDs for tracking requests across components
- **Context Propagation**: Request context flows through the entire request lifecycle
- **Environment Configuration**: Runtime configurable logging options
- **Sensitive Data Protection**: Automatic masking of credentials and API keys
- **Specialized Log Functions**: Purpose-built logging for proxy operations

## Configuration

### Environment Variables

Configure logging behavior with the following environment variables:

| Variable | Description | Values | Default |
|----------|-------------|--------|---------|
| `LOG_LEVEL` | Minimum log level to output | `DEBUG`, `INFO`, `WARN`, `ERROR` | `INFO` |
| `LOG_FORMAT` | Output format | `json`, `text` | `json` |
| `LOG_OUTPUT` | Output destination | `stdout`, `stderr` | `stdout` |

### Examples

```bash
# Development-friendly configuration
LOG_LEVEL=DEBUG LOG_FORMAT=text LOG_OUTPUT=stdout ./build/server

# Production configuration 
LOG_LEVEL=INFO LOG_FORMAT=json LOG_OUTPUT=stdout ./build/server
```

### Docker Environment Configuration

When using Docker, configure logging in `docker-compose.yml`:

```yaml
environment:
  - LOG_LEVEL=INFO
  - LOG_FORMAT=json
  - LOG_OUTPUT=stdout
```

## Log Levels

| Level | Usage |
|-------|-------|
| `DEBUG` | Detailed information for debugging purposes |
| `INFO` | General operational information |
| `WARN` | Warning conditions that don't cause errors |
| `ERROR` | Error conditions that should be addressed |

## Request Correlation

Every request receives a unique 16-character request ID that:

1. Is generated via middleware for each incoming request
2. Is added as an `X-Request-ID` header to responses
3. Is propagated through context to all components
4. Appears in all log entries related to the request

Example JSON log with request correlation:

```json
{
  "time": "2025-05-24T10:36:00Z",
  "level": "INFO",
  "msg": "Proxy request initiated",
  "request_id": "5c75cb5a3f0c3f41",
  "vendor": "gemini",
  "model": "gemini-2.0-flash-exp",
  "component": "proxy",
  "original_model": "final-test-model",
  "selected_vendor": "gemini",
  "selected_model": "gemini-2.0-flash-exp",
  "total_combinations": 228
}
```

## Sensitive Data Protection

The logging system automatically detects and masks sensitive data such as API keys:

```go
// Original data
data := map[string]any{
    "api_key": "sk-1234567890abcdef",
    "model": "gpt-4",
}

// After sanitization
// {
//   "api_key": "sk-1****cdef",
//   "model": "gpt-4"
// }
```

The system detects sensitive keys like:
- `api_key`, `apikey`
- `token`, `secret`
- `password`, `pass`
- `authorization`, `auth`

## Specialized Logging Functions

In addition to standard logging functions, the system provides specialized functions for common operations:

```go
// Log proxy request with vendor selection details
logger.LogProxyRequest(ctx, originalModel, selectedVendor, selectedModel, totalCombinations)

// Log vendor response processing
logger.LogVendorResponse(ctx, vendor, actualModel, presentedModel, responseSize, duration)
```

## Usage in Code

Import the logger package:

```go
import "github.com/aashari/go-generative-api-router/internal/logger"
```

### Basic Logging

```go
// Info level
logger.Info("Operation completed", "item_count", count)

// Debug level
logger.Debug("Processing item", "item_id", id)

// Warning level
logger.Warn("Resource running low", "resource", "memory", "available_mb", 512)

// Error level
logger.Error("Operation failed", "error", err, "operation", "database_query") 
```

### Context-Aware Logging

```go
// Info with context (includes request_id automatically)
logger.InfoCtx(ctx, "Request processed", "status", "success")

// Debug with context
logger.DebugCtx(ctx, "Processing request", "path", "/v1/chat/completions")

// Error with context
if err != nil {
    logger.ErrorCtx(ctx, "Request failed", "error", err)
}
```

### Map Sanitization

```go
// Sanitize a map before logging
sensitiveData := map[string]any{
    "user": "john",
    "api_key": "sk-1234567890abcdef",
    "request": requestBody,
}

// Safe to log
safeData := logger.SanitizeMap(sensitiveData)
logger.Info("Processing request", "data", safeData)
```

## Middleware Integration

The correlation middleware automatically adds request IDs to the context and response headers:

```go
// In routes.go
r.Use(middleware.CorrelationID)
```

## Testing Logs

When writing tests, you can capture and verify logs:

```go
func TestWithLogs(t *testing.T) {
    // Setup test logger with buffer
    var buf bytes.Buffer
    opts := &slog.HandlerOptions{Level: logger.LevelDebug}
    handler := slog.NewJSONHandler(&buf, opts)
    
    // Save original and restore after test
    originalLogger := logger.Logger
    defer func() { logger.Logger = originalLogger }()
    logger.Logger = slog.New(handler)
    
    // Run code that produces logs
    // ...
    
    // Verify log output
    output := buf.String()
    if !strings.Contains(output, "expected message") {
        t.Error("Expected log message not found")
    }
}
``` 