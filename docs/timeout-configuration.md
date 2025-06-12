# Timeout Configuration Guide

This document explains the timeout configuration for the Generative API Router to ensure clients don't experience premature timeouts, especially the common 120-second timeout issue.

## Overview

The service has been configured with generous timeout values to accommodate:
- Long AI model response times
- Large file processing (images, audio, documents)
- Complex streaming responses
- Network latency variations

## Timeout Hierarchy

### 1. Server-Level Timeouts (HTTP Server)

**Purpose**: Controls how long the HTTP server waits for client requests and responses.

```go
// Default values (can be overridden via environment variables)
READ_TIMEOUT=1500   // 25 minutes - time to read request
WRITE_TIMEOUT=1500  // 25 minutes - time to write response  
IDLE_TIMEOUT=1800   // 30 minutes - keep-alive timeout
```

**Environment Variables**:
- `READ_TIMEOUT` - Maximum time to read the entire request (including body)
- `WRITE_TIMEOUT` - Maximum time to write the response
- `IDLE_TIMEOUT` - Maximum time to wait for the next request when keep-alives are enabled

### 2. Client-Level Timeouts (Vendor API Calls)

**Purpose**: Controls how long to wait for responses from vendor APIs (OpenAI, Gemini).

```go
// Default value (can be overridden via environment variable)
CLIENT_TIMEOUT=1200  // 20 minutes
```

**Environment Variable**:
- `CLIENT_TIMEOUT` - Maximum time to wait for vendor API responses

### 3. Component-Level Timeouts

**Purpose**: Specific timeouts for different processing components.

#### Image Processing
```go
Timeout: 120 * time.Second  // 2 minutes for image downloads
```

#### Audio Processing  
```go
Timeout: 180 * time.Second  // 3 minutes for audio downloads
```

#### File Processing
```go
Timeout: 120 * time.Second  // 2 minutes for file downloads
```

## Configuration Methods

### 1. Environment Variables (Recommended)

Set these in your `.env` file or environment:

```bash
# Client timeout for vendor API calls (20 minutes)
CLIENT_TIMEOUT=1200

# Server timeouts (25 minutes each)
READ_TIMEOUT=1500
WRITE_TIMEOUT=1500

# Idle timeout (30 minutes)
IDLE_TIMEOUT=1800
```

### 2. Docker Configuration

The Docker image includes these defaults:

```dockerfile
ENV READ_TIMEOUT=1500
ENV WRITE_TIMEOUT=1500
ENV IDLE_TIMEOUT=1800
ENV CLIENT_TIMEOUT=1200
```

### 3. Code Defaults

If no environment variables are set, the code uses these defaults:
- Client timeout: 20 minutes (1200 seconds)
- Server read/write timeout: 25 minutes (1500 seconds)
- Server idle timeout: 30 minutes (1800 seconds)

## Common Timeout Scenarios

### 1. Long AI Responses

**Problem**: Complex prompts can take 5-15 minutes to process.
**Solution**: CLIENT_TIMEOUT=1200 (20 minutes) provides adequate buffer.

### 2. Large File Processing

**Problem**: Processing large images/audio files can take several minutes.
**Solution**: Component-specific timeouts (2-3 minutes) handle file processing.

### 3. Streaming Responses

**Problem**: Streaming can take a long time for lengthy responses.
**Solution**: WRITE_TIMEOUT=1500 (25 minutes) accommodates long streams.

### 4. Network Latency

**Problem**: Slow networks can cause premature timeouts.
**Solution**: Generous timeouts account for network variations.

## Troubleshooting Timeouts

### Client Receives 120-Second Timeout

**Likely Causes**:
1. Load balancer timeout (most common)
2. Proxy server timeout
3. Client-side timeout setting

**Solutions**:
1. Configure load balancer timeout > 1200 seconds
2. Configure proxy timeout > 1200 seconds  
3. Increase client timeout in application code

### Server Logs Show Timeout Errors

**Check These Settings**:
1. Verify environment variables are loaded
2. Check CLIENT_TIMEOUT for vendor API calls
3. Review component-specific timeouts

### Vendor API Timeouts

**Symptoms**: "context deadline exceeded" errors
**Solution**: Increase CLIENT_TIMEOUT environment variable

## Best Practices

### 1. Timeout Hierarchy
- Server timeouts should be longer than client timeouts
- Component timeouts should be shorter than client timeouts
- Always provide buffer time for network latency

### 2. Environment-Specific Configuration
```bash
# Development (shorter for faster feedback)
CLIENT_TIMEOUT=600    # 10 minutes
READ_TIMEOUT=900      # 15 minutes
WRITE_TIMEOUT=900     # 15 minutes

# Production (longer for reliability)
CLIENT_TIMEOUT=1200   # 20 minutes
READ_TIMEOUT=1500     # 25 minutes
WRITE_TIMEOUT=1500    # 25 minutes
```

### 3. Monitoring
- Monitor timeout-related errors in logs
- Track response time percentiles
- Alert on timeout spikes

## Infrastructure Considerations

### Load Balancers
Ensure your load balancer timeout is configured appropriately:
- AWS ALB: Set target group timeout > 1200 seconds
- NGINX: Set `proxy_read_timeout` > 1200 seconds
- CloudFlare: Configure timeout settings

### Container Orchestration
- ECS: Ensure health check timeout < service timeout
- Kubernetes: Configure appropriate `timeoutSeconds`

## Verification

### Test Long Requests
```bash
# Test with a request that should take > 2 minutes
curl -X POST http://localhost:8082/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "Write a detailed 2000-word essay about quantum computing"}],
    "max_tokens": 3000
  }'
```

### Monitor Timeout Logs
```bash
# Check for timeout-related errors
grep -i "timeout\|deadline" logs/server.log

# Monitor response times
grep "Request completed" logs/server.log | jq '.attributes.duration_ms'
```

## Summary

The service is now configured with generous timeouts to prevent the 120-second timeout issue:

- **Client API calls**: 20 minutes (1200s)
- **Server operations**: 25 minutes (1500s) 
- **File processing**: 2-3 minutes (120-180s)

These settings should accommodate even the longest AI responses and file processing operations while maintaining reasonable resource usage. 