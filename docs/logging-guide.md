# Logging Guide

This document details the structured logging system used in the Generative API Router service.

## Overview

The service implements an enterprise-grade structured logging system based on Go's `log/slog` package. Key features include:

- **Structured JSON Output**: Machine-parsable logs ideal for aggregation systems
- **Request Correlation**: Unique request IDs for tracking requests across components
- **Context Propagation**: Request context flows through the entire request lifecycle
- **Environment Configuration**: Runtime configurable logging options
- **Complete Data Logging**: Comprehensive logging without redaction or truncation
- **Specialized Log Functions**: Purpose-built logging for proxy operations

## Log Structure

The new structured logging format provides clear separation of concerns:

```json
{
  "timestamp": "2025-06-06T12:22:00.000Z",
  "level": "INFO",
  "message": "Human-readable description of the event",
  "service": "generative-api-router",
  "environment": "production",
  "attributes": {
    "user_id": "123",
    "endpoint": "/api/login",
    "vendor": "openai",
    "model": "gpt-4"
  },
  "request": {
    "request_id": "abc123",
    "method": "POST",
    "path": "/v1/chat/completions",
    "headers": {...},
    "body": "..."
  },
  "response": {
    "status_code": 200,
    "headers": {...},
    "body": "...",
    "content_length": 1024
  },
  "error": {
    "type": "ValidationError",
    "message": "Invalid request format",
    "stacktrace": "..."
  }
}
```

### Field Descriptions

| Field | Description | Required | Example |
|-------|-------------|----------|---------|
| `timestamp` | ISO 8601 format timestamp | ✅ | `2025-06-06T12:22:00.000Z` |
| `level` | Log severity level | ✅ | `INFO`, `ERROR`, `WARN`, `DEBUG` |
| `message` | Human-readable event description | ✅ | `Request processed successfully` |
| `service` | Name of the service | ✅ | `generative-api-router` |
| `environment` | Deployment environment | ✅ | `prod`, `staging`, `development` |
| `attributes` | Additional context data | ❌ | `{"vendor": "openai", "model": "gpt-4"}` |
| `request` | HTTP request details | ❌ | `{"method": "POST", "path": "/v1/chat"}` |
| `response` | HTTP response details | ❌ | `{"status_code": 200, "body": "..."}` |
| `error` | Error information | ❌ | `{"type": "Error", "message": "..."}` |

## Configuration

### Environment Variables

Configure logging behavior with the following environment variables:

| Variable | Description | Values | Default |
|----------|-------------|--------|---------|
| `LOG_LEVEL` | Minimum log level to output | `DEBUG`, `INFO`, `WARN`, `ERROR` | `INFO` |
| `LOG_FORMAT` | Output format | `json`, `text` | `json` |
| `LOG_OUTPUT` | Output destination | `stdout`, `stderr` | `stdout` |
| `SERVICE_NAME` | Service name in logs | Any string | `generative-api-router` |
| `ENVIRONMENT` | Environment name in logs | Any string | `development` |

### Examples

```bash
# Development-friendly configuration
LOG_LEVEL=DEBUG LOG_FORMAT=text LOG_OUTPUT=stdout ./build/server

# Production configuration 
LOG_LEVEL=INFO LOG_FORMAT=json LOG_OUTPUT=stdout SERVICE_NAME=genapi ENVIRONMENT=production ./build/server
```

### Docker Environment Configuration

When using Docker, configure logging in `docker-compose.yml`:

```yaml
environment:
  - LOG_LEVEL=INFO
  - LOG_FORMAT=json
  - LOG_OUTPUT=stdout
  - SERVICE_NAME=generative-api-router
  - ENVIRONMENT=production
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
4. Appears in the `request.request_id` field of all log entries related to the request

Example JSON log with request correlation:

```json
{
  "timestamp": "2025-06-06T10:36:00Z",
  "level": "INFO",
  "message": "Proxy request initiated",
  "service": "generative-api-router",
  "environment": "staging",
  "attributes": {
    "component": "proxy",
    "vendor": "gemini",
    "model": "gemini-2.0-flash-exp",
    "original_model": "final-test-model",
    "selected_vendor": "gemini",
    "selected_model": "gemini-2.0-flash-exp",
    "total_combinations": 228
  },
  "request": {
    "request_id": "5c75cb5a3f0c3f41"
  }
}
```

## Complete Data Logging

The logging system logs complete data structures without any redaction, truncation, or selective filtering. This includes:

- **Complete API Keys**: Full credentials are logged for debugging
- **Complete Request/Response Bodies**: Entire payloads are captured
- **Complete Headers**: All HTTP headers including sensitive ones
- **Complete Error Details**: Full error messages and stack traces

**IMPORTANT**: External logging systems should handle redaction, size management, and sensitive data filtering.

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

### Specialized Logging Functions

The system provides specialized functions for common operations:

#### Request/Response Logging

```go
// Log HTTP request
logger.LogRequest(ctx, "POST", "/v1/chat/completions", "curl/8.0", headers, body)

// Log HTTP response  
logger.LogResponse(ctx, 200, responseHeaders, responseBody)

// Log vendor communication
logger.LogVendorCommunication(ctx, "openai", "https://api.openai.com/v1/chat/completions",
    requestBody, responseBody, requestHeaders, responseHeaders)
```

#### Proxy Operations

```go
// Log proxy request with vendor selection
logger.LogProxyRequest(ctx, originalModel, selectedVendor, selectedModel, totalCombinations, requestData)

// Log vendor response processing
logger.LogVendorResponse(ctx, vendor, actualModel, presentedModel, responseSize, duration, completeResponse)
```

#### Error Logging

```go
// Log errors with complete context
logger.LogError(ctx, "proxy", err, map[string]any{
    "operation": "vendor_request",
    "api_key": "sk-complete-key",
    "request_data": completeRequestData,
})
```

## Middleware Integration

The correlation middleware automatically adds request IDs to the context and response headers:

```go
// In routes.go
r.Use(middleware.RequestCorrelationMiddleware)
```

The middleware logs complete request and response data:

```json
{
  "timestamp": "2025-06-06T07:44:52Z",
  "level": "INFO",
  "message": "Request completed",
  "service": "generative-api-router",
  "environment": "staging",
  "attributes": {
    "component": "middleware",
    "duration_ms": 1537,
    "start_time": "2025-06-06T07:44:52Z",
    "end_time": "2025-06-06T07:44:52Z"
  },
  "request": {
    "request_id": "e5d3efbf66389ab1",
    "method": "GET",
    "path": "/health",
    "headers": {...},
    "content_length": 0
  },
  "response": {
    "request_id": "e5d3efbf66389ab1",
    "status_code": 200,
    "headers": {...},
    "body": "OK",
    "content_length": 2
  }
}
```

## Log Analysis Examples

### Query Request Flows

```bash
# Find all logs for a specific request
grep "request_id.*abc123" logs/server.log | jq .

# Trace vendor selection
grep "Proxy request initiated" logs/server.log | jq '.attributes.selected_vendor'

# Monitor error rates
grep '"level":"ERROR"' logs/server.log | jq '.error.type' | sort | uniq -c
```

### Performance Analysis

```bash
# Find slow requests
grep '"level":"INFO"' logs/server.log | jq 'select(.attributes.duration_ms > 1000)'

# Analyze vendor response times
grep "Vendor response processed" logs/server.log | jq '.attributes.processing_time_ms'
```

### Security Monitoring

```bash
# Monitor authentication failures
grep '"error"' logs/server.log | jq 'select(.error.type | contains("Auth"))'

# Track API key usage
grep '"attributes"' logs/server.log | jq 'select(.attributes.api_key)'
```

## Testing Logs

When writing tests, you can capture and verify logs:

```go
func TestWithLogs(t *testing.T) {
    // Setup test logger with buffer
    var buf bytes.Buffer
    handler := &logger.StructuredJSONHandler{
        Writer:      &buf,
        ServiceName: "test-service",
        Environment: "test",
    }
    
    // Save original and restore after test
    originalLogger := logger.Logger
    defer func() { logger.Logger = originalLogger }()
    logger.Logger = slog.New(handler)
    
    // Run code that produces logs
    // ...
    
    // Verify log output
    output := buf.String()
    var logEntry logger.StructuredLogEntry
    if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
        t.Error("Invalid JSON log output")
    }
    
    if logEntry.Message != "expected message" {
        t.Error("Expected log message not found")
    }
}
```

## 📚 Additional Resources

- **[Development Guide](development-guide.md)** - Complete development setup
- **[Testing Guide](testing-guide.md)** - Testing with logs
- **[API Reference](api-reference.md)** - API documentation

---

**Remember**: Comprehensive logging is crucial for debugging, monitoring, and maintaining production systems. Log everything, let external systems handle filtering. 📝