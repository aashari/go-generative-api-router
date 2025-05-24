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