# Generative API Router

A Go microservice that proxies OpenAI-compatible API calls to multiple vendors (OpenAI, Gemini) using randomly selected vendor-model pairs.

## Features

- Exposes an OpenAI-compatible `/chat/completions` endpoint
- Randomly selects a vendor and model from a predefined list
- Proxies requests to the selected vendor's API
- Supports both OpenAI and Google's Gemini API (using OpenAI compatibility mode)
- Supports tool calling (function calling) for integrating with external APIs

## Setup

1. **Clone the Repository**:
   ```bash
   git clone https://github.com/aashari/generative-api-router.git
   cd generative-api-router
   ```

2. **Install Go**:
   Ensure Go 1.22+ is installed (https://go.dev/doc/install).

3. **Configure Credentials**:
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

4. **Configure Vendor-Model Pairs**:
   Edit `models.json` to define which vendor-model pairs can be randomly selected:
   ```json
   [
     {
       "vendor": "gemini",
       "model": "gemini-1.5-flash"
     },
     {
       "vendor": "gemini",
       "model": "gemini-1.5-pro"
     },
     {
       "vendor": "openai",
       "model": "gpt-4o"
     },
     {
       "vendor": "openai",
       "model": "gpt-4o-mini"
     }
   ]
   ```

5. **Run Locally**:
   ```bash
   go mod tidy
   go run ./cmd/server
   ```

6. **Run with Docker**:
   ```bash
   docker-compose up --build
   ```

## Testing the Service

### Basic Chat Completion

Send a request to the `/chat/completions` endpoint:
```bash
curl -X POST http://localhost:8082/chat/completions \
     -H "Content-Type: application/json" \
     -d '{"model": "any-model", "messages": [{"role": "user", "content": "Hello"}]}'
```

### Tool Calling

Test tool calling with a function definition:
```bash
curl -X POST http://localhost:8082/chat/completions \
     -H "Content-Type: application/json" \
     -d '{
       "model": "any-model",
       "messages": [{"role": "user", "content": "What is the weather in Boston?"}],
       "tools": [{
         "type": "function",
         "function": {
           "name": "get_current_weather",
           "description": "Get the current weather in a given location",
           "parameters": {
             "type": "object",
             "properties": {
               "location": {
                 "type": "string",
                 "description": "The city, e.g., Boston"
               }
             },
             "required": ["location"]
           }
         }
       }],
       "tool_choice": "auto"
     }'
```

**Note**: The `model` field in your request will be ignored and replaced with a randomly selected model from `models.json`. You can provide any value for this field, but it will not affect the processing.

## How It Works

1. When a request arrives at the `/chat/completions` endpoint, the service:
   - Randomly selects a vendor-model pair from `models.json`
   - Finds a matching credential for that vendor in `credentials.json`
   - Replaces the `model` field in the request with the selected model
   - Forwards the request to the appropriate vendor API
   - Streams the response back to the client

2. For tool calling:
   - The service validates and forwards `tools` and `tool_choice` parameters
   - The selected model may respond with `tool_calls` for function execution
   - The service streams these responses back to the client for processing
   - The client is responsible for executing the functions and continuing the conversation

3. This approach ensures:
   - Randomized distribution across different AI vendors and models
   - Clients don't need to be aware of vendor-specific model names
   - OpenAI API compatibility is maintained for client applications
   - Tool calling is supported across vendors that implement it

## Security Notes

- The current implementation stores API keys in plain text in `credentials.json`. 
- For production environments, consider using environment variables or a secret management solution.
- The `credentials.json` file is included in `.gitignore` to prevent accidentally committing API keys.

## Known Limitations

- The client's requested model is completely ignored and replaced with a randomly selected one
- Only the `/chat/completions` endpoint is supported
- Each vendor in `models.json` must have at least one matching credential in `credentials.json`
- Tool execution is the client's responsibility; the router only forwards `tool_calls`

## Troubleshooting

- If you encounter 401 errors, check that your API keys are valid.
- For 400 errors, check the request format and ensure all required fields are present.
- Response format differences between vendors may occur, especially for error cases.
- If tool calls aren't working, ensure the model supports tool calling and the prompt is clear enough to trigger a function call.
- For streaming issues with tool calls, check that your client can handle streamed `tool_calls` in the response.

## Future Enhancements

- Add support for more endpoints (e.g., `/embeddings`)
- Implement robust error handling and retry mechanisms
- Add observability with metrics and tracing
- Develop CI/CD pipelines for automated testing and deployment 