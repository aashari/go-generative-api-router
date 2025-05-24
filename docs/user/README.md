# User Guide

## Quick Start
1. Ensure the service is running on port 8082
2. Configure your API client to point to `http://localhost:8082`
3. Use OpenAI-compatible endpoints

## Available Endpoints
- `GET /health` - Health check
- `GET /v1/models` - List available models
- `POST /v1/chat/completions` - Chat completions (streaming supported)

## Examples
See the `examples/` directory for usage examples in various languages.

## Configuration
- Models are configured in `configs/models.json`
- API keys are stored in `configs/credentials.json`

## Logging and Monitoring

### Request Correlation
Each request is assigned a unique ID that appears in:
- Response headers as `X-Request-ID`
- All log entries related to the request
- Error responses when debugging is enabled

You can use this ID for troubleshooting and tracking requests through the system.

### Environment Variables
Configure the service behavior with these environment variables:

| Variable | Description | Values | Default |
|----------|-------------|--------|---------|
| `PORT` | Server port | Any valid port | `8082` |
| `LOG_LEVEL` | Logging detail level | `DEBUG`, `INFO`, `WARN`, `ERROR` | `INFO` |
| `LOG_FORMAT` | Log output format | `json`, `text` | `json` |
| `LOG_OUTPUT` | Log destination | `stdout`, `stderr` | `stdout` |