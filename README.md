# Generative API Router

A Go microservice acting as a universal API gateway that transparently proxies OpenAI-compatible API calls to various AI vendors, including OpenAI and Google's Gemini. This router simplifies integration by providing a unified and consistent API interface, intelligently routing requests based on your configuration.

---

## Features

* **Unified API Endpoint**: Offers an OpenAI-compatible API through endpoints (`/chat/completions` and `/models`).
* **Intelligent Vendor and Model Selection**:

  * Automatically selects credentials and models defined in configuration.
  * Allows explicit vendor selection via query parameter (`?vendor=openai` or `?vendor=gemini`).
* **Streaming Responses**: Seamlessly supports streaming (`"stream": true`) from AI vendors.
* **Transparent Proxying**: Passes headers, request bodies, and responses without alteration (excluding the model selection).
* **Tool Calling Support**: Validates and proxies advanced features like function calling.
* **Flexible Configuration**: Easily manage API keys and models through JSON files (`credentials.json` and `models.json`).
* **Docker-ready**: Containerized deployment made simple.
* **Health Checks**: Includes a straightforward `/health` endpoint.

---

## Setup

### Step 1: Clone the Repository

```bash
git clone https://github.com/aashari/generative-api-router.git
cd generative-api-router
```

### Step 2: Install Dependencies

Ensure you have [Go 1.22+ installed](https://go.dev/doc/install).

### Step 3: Configure Credentials

Create and populate `credentials.json`:

```bash
cp credentials.json.example credentials.json
```

Fill in your API keys:

```json
[
  { "platform": "openai", "type": "api-key", "value": "your-openai-key" },
  { "platform": "gemini", "type": "api-key", "value": "your-gemini-key" }
]
```

### Step 4: Define Models

Update `models.json` with available vendor-model pairs:

```json
[
  {"vendor": "gemini", "model": "gemini-2.5-flash-preview-04-17"},
  {"vendor": "gemini", "model": "gemini-2.5-pro-preview-05-06"},
  {"vendor": "openai", "model": "gpt-4o"}
]
```

### Step 5: Run Locally

**Directly**

```bash
go mod tidy
go run ./cmd/server/main.go
```

The service will run at `http://localhost:8082`.

**Using Docker**

```bash
docker-compose up --build
```

---

## API Endpoints

### Chat Completions

Send requests to `/chat/completions`. The router internally selects the model:

```bash
curl -X POST http://localhost:8082/chat/completions \
-H "Content-Type: application/json" \
-d '{"model": "ignored", "messages": [{"role": "user", "content": "Hello!"}]}'
```

#### Streaming

Enable streaming responses:

```bash
curl -X POST http://localhost:8082/chat/completions \
-H "Content-Type: application/json" \
-d '{"model": "ignored", "messages": [{"role": "user", "content": "Stream test"}], "stream": true}'
```

### Available Models

Retrieve available models from `/models`:

```bash
curl -X GET http://localhost:8082/models
```

Filter by vendor:

```bash
curl -X GET "http://localhost:8082/models?vendor=openai"
```

### Health Check

Check service status:

```bash
curl -X GET http://localhost:8082/health
```

---

## Security Recommendations

* Avoid storing sensitive API keys in plaintext. Use environment variables or dedicated secrets management solutions.
* By default, `credentials.json` is excluded from version control.

---

## Troubleshooting

* **401 Unauthorized**: Verify your API keys.
* **400 Bad Request**: Check your request formatting and parameters.
* **Streaming Issues**: Confirm vendor support and ensure proper headers (`Transfer-Encoding: chunked`) are set.

---

## Future Enhancements

* Additional OpenAI endpoints (`/embeddings`).
* Advanced routing strategies (weighted, latency-based).
* Improved observability and metrics integration.
