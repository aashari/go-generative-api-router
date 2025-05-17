# Generative API Router

A Go microservice that proxies OpenAI-compatible API calls to multiple LLM vendors (OpenAI, Gemini) using configurable selection strategies. Supports vendor filtering, streaming responses, and tool calling while maintaining transparent request/response handling.

## Features

- **Multi-Vendor Support**: Routes requests to OpenAI or Gemini using OpenAI API compatibility
- **Random Selection**: Automatically selects from configured vendors and models
- **Vendor Filtering**: Supports explicit vendor selection via `?vendor=` query parameter
- **Transparent Proxy**: Maintains all original request/response data (except the intentional model override)
- **Streaming Support**: Properly handles streamed responses from both vendors
- **Tool Calling**: Supports function calling for AI tool use with proper validation
- **Modular Design**: Clean separation of concerns with selector, validator, and client components
- **Configuration Driven**: Easily configure available models and credentials via JSON files

## Setup

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
       "value": "your-openai-key"
     },
     {
       "platform": "gemini",
       "type": "api-key",
       "value": "your-gemini-key"
     }
   ]
   ```

3. **Configure Models**:
   Edit `models.json` to define which vendor-model pairs can be randomly selected:
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

5. **Docker Support**:
   ```bash
   docker-compose up --build
   ```

## API Endpoints

### Health Check
```
GET /health
```

### Models Listing
```
GET /models
GET /models?vendor=openai
```

Returns a list of available models in OpenAI-compatible format.

### Chat Completions
```
POST /chat/completions
POST /chat/completions?vendor=gemini
```

Send OpenAI-compatible requests to generate completions from either vendor.

#### Stream Support
```json
{
  "model": "any-model",
  "messages": [{"role": "user", "content": "Hello"}],
  "stream": true
}
```

#### Tool Calling
```json
{
  "model": "any-model",
  "messages": [{"role": "user", "content": "What's the weather?"}],
  "tools": [{
    "type": "function",
    "function": {
      "name": "get_weather",
      "description": "Get weather information",
      "parameters": {
        "type": "object",
        "properties": {
          "location": {"type": "string"}
        },
        "required": ["location"]
      }
    }
  }],
  "tool_choice": "auto"
}
```

## Architecture

The project follows a modular design with clear separation of concerns:

- **App**: Central configuration and HTTP handlers
- **Selector**: Vendor and model selection strategies
- **Validator**: Request validation and modification
- **Proxy Client**: Communication with LLM vendor APIs
- **Config**: Configuration management

## Development

### Building
```bash
go build -o generative-api-router ./cmd/server
```

### Testing
Test basic functionality:
```bash
curl -X GET http://localhost:8082/health
curl -X GET http://localhost:8082/models
curl -X POST http://localhost:8082/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model": "any-model", "messages": [{"role": "user", "content": "Hello"}]}'
```

## Security Notes

- The current implementation stores API keys in plain text in `credentials.json`. 
- For production environments, consider using environment variables or a secret management solution.
- The `credentials.json` file is included in `.gitignore` to prevent accidentally committing API keys.

## License

MIT

## Acknowledgments

This project was inspired by the need for a unified interface to multiple LLM providers.
