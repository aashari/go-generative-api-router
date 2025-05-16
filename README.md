# Generative API Router

A Go microservice that proxies OpenAI-compatible API calls to multiple vendors (e.g., OpenAI, Gemini). It intelligently selects a vendor (based on a randomly chosen credential) and an appropriate model for that vendor from your configuration, all while maintaining a consistent OpenAI API interface for the client.

## Features

- Exposes an OpenAI-compatible `/chat/completions` endpoint.
- **Selection Logic:**
  1.  Randomly selects a _credential_ from `credentials.json`. The `platform` field in the credential determines the vendor.
  2.  Randomly selects a _model_ from `models.json` that matches the chosen vendor.
- Proxies requests transparently to the selected vendor's API.
- Supports OpenAI and Google's Gemini API (using its OpenAI compatibility mode) out-of-the-box.
- Validates and forwards tool calling (function calling) parameters.
- Configurable via `credentials.json` (for API keys and vendor platforms) and `models.json` (for vendor-specific model lists).
- Easy to deploy with Docker.
- Includes a simple `/health` check endpoint.

## Setup

1.  **Clone the Repository**:

    ```bash
    git clone https://github.com/aashari/generative-api-router.git
    cd generative-api-router
    ```

2.  **Install Go**:
    Ensure Go 1.22+ is installed (as per `go.mod` and `Dockerfile`). (https://go.dev/doc/install).

3.  **Configure Credentials**:
    Copy the example file and edit with your valid API keys:

    ```bash
    cp credentials.json.example credentials.json
    ```

    Then edit `credentials.json`. The `platform` field is crucial as it links the credential to a vendor specified in `models.json`.

    ```json
    [
      {
        "platform": "openai", // This vendor name must match a vendor in models.json
        "type": "api-key",
        "value": "your-openai-key"
      },
      {
        "platform": "gemini", // This vendor name must match a vendor in models.json
        "type": "api-key",
        "value": "your-gemini-key"
      }
    ]
    ```

4.  **Configure Vendor-Model Pairs**:
    Edit `models.json` to define which models are available for each vendor. The `vendor` field here must correspond to a `platform` in `credentials.json`.

    ```json
    [
      {
        "vendor": "gemini",  // Matches a "platform" in credentials.json
        "model": "gemini-1.5-flash"
      },
      {
        "vendor": "gemini",
        "model": "gemini-1.5-pro"
      },
      {
        "vendor": "openai", // Matches a "platform" in credentials.json
        "model": "gpt-4o"
      },
      {
        "vendor": "openai",
        "model": "gpt-4o-mini"
      }
    ]
    ```

5.  **Build and Run Locally**:

    ```bash
    go mod tidy
    go run ./cmd/server/main.go
    ```

    The server will start on `http://localhost:8082`.

6.  **Build and Run with Docker**:
    ```bash
    docker-compose up --build
    ```

## Testing the Service

The router listens on `http://localhost:8082`.

### Basic Chat Completion

Send a request to the `/chat/completions` endpoint. The `model` field in your request will be **ignored and replaced** by the router.

```bash
curl -X POST http://localhost:8082/chat/completions \
     -H "Content-Type: application/json" \
     -d '{"model": "any-model-will-be-ignored", "messages": [{"role": "user", "content": "Hello, router!"}]}'
```

### Tool Calling (Function Calling)

Test tool calling with a function definition:

```bash
curl -X POST http://localhost:8082/chat/completions \
     -H "Content-Type: application/json" \
     -d '{
       "model": "ignored-model",
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

**Note**: The `model` field in your request payload is completely ignored. The router determines the model based on its random selection logic.

## How It Works

1.  When a request arrives at the `/chat/completions` endpoint, the service:

    - **Selects a Credential:** Randomly picks an entry from the `credentials.json` array. The `platform` field of this chosen credential determines the target **vendor**.
    - **Selects a Model:** Filters the `models.json` array to get all models associated with the selected **vendor**. From this filtered list, it randomly picks one model.
    - **Modifies Request:** Replaces the `model` field in the incoming client request with the randomly selected model name.
    - **Validates (Optional Fields):** Checks for the presence and basic structure of `tools` and `tool_choice` if they exist in the request.
    - **Forwards Request:** Proxies the (modified) request, including all original headers (plus the vendor-specific `Authorization` header) and the full body, to the appropriate vendor's OpenAI-compatible API endpoint (e.g., `https://api.openai.com/v1/chat/completions` or `https://generativelanguage.googleapis.com/v1beta/openai/chat/completions`).
    - **Streams Response:** Streams the vendor's entire response (status code, headers, and body) directly back to the original client.

2.  This approach ensures:
    - **Simplified Client Integration:** Clients target a single router endpoint and don't need to manage multiple vendor SDKs, API keys, or specific model names.
    - **Randomized Distribution:** Requests are randomly distributed across configured vendors and their models, useful for basic load balancing or A/B testing.
    - **OpenAI API Compatibility:** The router maintains compatibility with the OpenAI API spec for client applications.
    - **Tool Calling Support:** `tools` and `tool_choice` parameters are handled correctly for vendors supporting this feature via the OpenAI-compatible interface.

## Security Notes

- The current implementation stores API keys in plain text in `credentials.json`.
- **For production environments, strongly consider using environment variables or a dedicated secret management solution (e.g., HashiCorp Vault, AWS Secrets Manager, GCP Secret Manager) instead of a file for API keys.**
- The `credentials.json` file is included in `.gitignore` to help prevent accidentally committing API keys. Ensure it remains out of version control if you use it.

## Known Limitations

- The client's requested `model` in the payload is completely ignored and replaced by the router's selection.
- Only the `/chat/completions` and `/health` endpoints are currently supported.
- Each `platform` (vendor) referenced in `credentials.json` that you intend to use must have corresponding `model` entries in `models.json` with a matching `vendor` name. Conversely, every `vendor` in `models.json` must have at least one credential defined in `credentials.json`.
- The client is responsible for executing any functions if `tool_calls` are returned by the model; the router only proxies the `tool_calls` information.

## Troubleshooting

- **401 Unauthorized Errors:** Double-check that your API keys in `credentials.json` are correct, valid, and have the necessary permissions for the selected models.
- **400 Bad Request Errors (from router):**
  - May indicate an issue with the router's internal selection (e.g., no models found for a selected vendor based on your `models.json` and `credentials.json` setup). Check server logs.
  - Ensure your request body is valid JSON and contains the required `messages` field.
  - If using `tools` or `tool_choice`, ensure their structure is correct as per OpenAI specs.
- **400 Bad Request Errors (from vendor, proxied):** The vendor API rejected your request. The error message in the response body should provide details. This could be due to an invalid parameter (other than `model`), malformed messages, or exceeding token limits for the _actually selected_ model.
- **5xx Server Errors (from router or proxied):**
  - If from the router, check its logs for issues like failing to connect to the vendor or internal processing errors.
  - If proxied from the vendor, it indicates an issue on the vendor's side.
- **Tool calls not working:** Ensure the prompt is clear and the model selected by the router actually supports tool calling and is capable of understanding the request to use a tool.

## Future Enhancements

- Add support for more OpenAI API endpoints (e.g., `/embeddings`).
- Implement more sophisticated model/vendor selection strategies (e.g., weighted, least-latency, based on request parameters).
- Introduce robust error handling with standardized error responses from the router itself.
- Add observability features: metrics (e.g., Prometheus), logging (structured), and tracing (e.g., OpenTelemetry).
- Support for request retries with backoff strategies.
- More flexible configuration for `http.Client` settings (timeouts, transport).
