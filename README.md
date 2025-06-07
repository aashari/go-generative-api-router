# Generative API Router

[![Go Report Card](https://goreportcard.com/badge/github.com/aashari/go-generative-api-router)](https://goreportcard.com/report/github.com/aashari/go-generative-api-router)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/aashari/go-generative-api-router)](https://github.com/aashari/go-generative-api-router)

A Go microservice that proxies OpenAI-compatible API calls to multiple LLM vendors (OpenAI, Gemini) using configurable selection strategies. This router simplifies integration with AI services by providing a unified interface while handling the complexity of multi-vendor management.

<!-- 
<div align="center">
  <img src="https://raw.githubusercontent.com/aashari/go-generative-api-router/main/docs/assets/architecture-diagram.png" alt="Architecture Diagram" width="800">
</div>
-->

## Features

- **Multi-Vendor Support**: Routes requests to OpenAI or Gemini using OpenAI API compatibility
- **Even Distribution Selection**: Fair distribution across all vendor-credential-model combinations
- **Vendor Filtering**: Supports explicit vendor selection via `?vendor=` query parameter
- **Transparent Proxy**: Maintains all original request/response data (except for model selection)
- **Streaming Support**: Properly handles chunked streaming responses for real-time applications
- **Tool Calling**: Supports function calling/tools for AI agents with proper validation
- **Modular Design**: Clean separation of concerns with selector, validator, and client components
- **Configuration Driven**: Easily configure available models and credentials via JSON files
- **Health Check**: Built-in health check endpoint
- **Comprehensive Testing**: Full test coverage with unit tests for all components
- **🌐 Public Image URL Support**: Automatic downloading and conversion of public image URLs to base64

## Quick Start

### Prerequisites

- Go 1.21 or higher
- API keys for OpenAI and/or Google Gemini
- Make (for build automation)

### Installation

1. **Clone the Repository**:
   ```bash
   git clone https://github.com/aashari/go-generative-api-router.git
   cd go-generative-api-router
   ```

2. **Setup Environment**:
   ```bash
   make setup
   ```
   This will:
   - Download Go dependencies
   - Install development tools
   - Create `configs/credentials.json` from the example template

3. **Configure Credentials**:
   Edit `configs/credentials.json` with your API keys:
   ```json
   [
     {
       "platform": "openai",
       "type": "api-key",
       "value": "sk-your-openai-key"
     },
     {
       "platform": "gemini",
       "type": "api-key",
       "value": "your-gemini-key"
     }
   ]
   ```

4. **Configure Models**:
   Edit `configs/models.json` to define which vendor-model pairs can be selected:
   ```json
   [
     {
       "vendor": "gemini",
       "model": "gemini-2.0-flash"
     },
     {
       "vendor": "openai",
       "model": "gpt-4o"
     }
   ]
   ```

5. **Run the Service**:
   ```bash
   make run
   ```
   
   The service will be available at http://localhost:8082

## Selection Strategy

The router uses an **Even Distribution Selector** that ensures fair distribution across all vendor-credential-model combinations. This approach provides true fairness where each combination has exactly equal probability of being selected.

### How It Works

1. **Combination Generation**: The system creates a flat list of all valid vendor-credential-model combinations
2. **Equal Probability**: Each combination gets exactly `1/N` probability where N = total combinations
3. **Fair Distribution**: Unlike traditional two-stage selection (vendor → model), this ensures no bias toward vendors with fewer models

### Example Distribution

With the following configuration:
- **18 Gemini credentials** × **6 models** = 108 combinations
- **1 OpenAI credential** × **6 models** = 6 combinations
- **Total**: 114 combinations

Each combination has exactly **1/114 = 0.877%** probability:
- **Gemini overall**: 108/114 = 94.7%
- **OpenAI overall**: 6/114 = 5.3%

This reflects the actual resource availability rather than artificial vendor-level balancing.

### Benefits

- ✅ **True Fairness**: Each credential-model combination has exactly equal probability
- ✅ **Resource Proportional**: Distribution reflects actual available resources
- ✅ **Scalable**: Automatically adapts as credentials/models are added/removed
- ✅ **Transparent**: Clear logging shows selection and total combination count
- ✅ **No Bias**: Eliminates bias toward vendors with fewer models per credential

### Monitoring Selection

The service logs each selection decision for transparency:

```
Even distribution selected combination - Vendor: openai, Model: gpt-4o (from 114 total combinations)
```

You can monitor the distribution by checking the server logs to verify fair selection across all combinations.

## Usage

### Using Example Scripts

Example scripts are provided for common use cases:

```bash
# Basic usage examples
./examples/curl/basic.sh

# Streaming examples
./examples/curl/streaming.sh

# Tool calling examples
./examples/curl/tools.sh
```

### Client Libraries

Example implementations are available for multiple languages:

- **Python**: `examples/clients/python/client.py`
- **Node.js**: `examples/clients/nodejs/client.js`
- **Go**: `examples/clients/go/client.go`

### Docker Deployment

Build and run using Docker:

```bash
# Build and run with Docker Compose
make docker-build
make docker-run

# Stop the service
make docker-stop
```