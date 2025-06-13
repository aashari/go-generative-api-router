# Integration Tests for Generative API Router

This document describes the comprehensive integration test suite for the Generative API Router project. These tests perform real API calls to verify end-to-end functionality.

## 🧪 Test Overview

The integration tests are designed to validate the router's functionality with **real API calls** to both OpenAI and Gemini vendors. Unlike unit tests that use mocks, these tests verify that the entire system works correctly in production-like conditions.

### Test Categories

1. **Basic Functionality Tests** (`integration_test.go`)
   - Health endpoint validation
   - Models endpoint with vendor filtering
   - Basic chat completions
   - Vendor-specific routing
   - Request distribution testing
   - Error handling scenarios
   - CORS support
   - Concurrent request handling

2. **Advanced Feature Tests** (`advanced_integration_test.go`)
   - Streaming support
   - Image content detection
   - Vision-capable model routing
   - Large payload handling
   - Vendor-specific parameters
   - Load balancing behavior
   - Response format validation
   - Timeout handling

## 🛠️ Prerequisites

### Required Configuration

1. **Credentials Configuration**: Real API credentials must be configured in `configs/credentials.json`
   ```json
   [
     {
       "platform": "openai",
       "type": "api-key",
       "value": "sk-proj-your-actual-openai-key"
     },
     {
       "platform": "gemini",
       "type": "api-key", 
       "value": "your-actual-gemini-api-key"
     }
   ]
   ```

2. **Models Configuration**: Ensure `configs/models.json` is properly configured
3. **Environment**: Set up your `.env` file based on `.env.example`

### API Quotas and Rate Limits

⚠️ **Important**: These tests make real API calls which:
- Consume your API quotas
- May incur costs
- Are subject to rate limits
- Require active API keys

## 🚀 Running the Tests

### Quick Start

```bash
# Run all integration tests
make test-integration

# Run only unit tests
make test

# Run all tests (unit + integration)
make test-all
```

### Manual Test Execution

```bash
# Run integration tests with verbose output
go test -v -timeout=5m ./integration_test.go

# Run advanced feature tests
go test -v -timeout=5m ./advanced_integration_test.go

# Run specific test function
go test -v -run TestHealthEndpoint ./integration_test.go

# Run tests with additional debugging
go test -v -timeout=10m ./integration_test.go -args -debug
```

### Test Configuration

You can customize test behavior using environment variables:

```bash
# Set test timeout
export TEST_TIMEOUT=300s

# Set log level for tests
export LOG_LEVEL=debug

# Run tests with specific environment
export ENVIRONMENT=integration
```

## 📋 Test Scenarios

### Core API Endpoints

1. **Health Check** (`/health`)
   - Service status validation
   - Component availability checks
   - Uptime and version information

2. **Models List** (`/v1/models`)
   - Complete model listing
   - Vendor-specific filtering (`?vendor=openai`, `?vendor=gemini`)
   - Response format validation

3. **Chat Completions** (`/v1/chat/completions`)
   - Basic completion requests
   - Model name preservation
   - Vendor-specific routing
   - Parameter handling

### Multi-Vendor Testing

- **OpenAI Integration**: Tests with `gpt-4o`, `gpt-4.1`, `o3-mini` models
- **Gemini Integration**: Tests with `gemini-2.5-flash-preview-04-17`, `gemini-2.5-flash-preview-05-20`
- **Even Distribution**: Validates load balancing across vendors
- **Vendor Filtering**: Tests explicit vendor selection

### Advanced Features

1. **Streaming Support**
   - Server-Sent Events (SSE) format
   - Real-time response handling
   - Stream completion detection

2. **Multimodal Content**
   - Image content detection
   - Vision-capable model routing
   - Base64 data URL handling

3. **Tool Support**
   - Function calling capabilities
   - Tool definition validation
   - Tool usage in conversations

4. **Error Handling**
   - Invalid model names
   - Malformed requests
   - Rate limit handling
   - API error propagation

### Load and Performance Testing

- **Concurrent Requests**: Multiple simultaneous API calls
- **Rapid Succession**: Quick sequential requests
- **Large Payloads**: Handling of large message content
- **Timeout Scenarios**: Long-running request handling

## 📊 Expected Results

### Successful Test Runs

When everything works correctly:
```bash
✅ Health check passed: healthy
✅ Found 5 models
✅ Found 3 OpenAI models
✅ Found 2 Gemini models
✅ Chat completion successful with model: gpt-4o
✅ OpenAI-specific routing successful
✅ Gemini-specific routing successful
✅ Distribution test completed: 8/10 successful requests
✅ CORS preflight successful
✅ Concurrent requests test: 4/5 succeeded
```

### Handling API Limitations

The tests are designed to gracefully handle real-world API limitations:

- **Rate Limits**: Tests log but don't fail when rate limited
- **API Errors**: Tests continue when individual API calls fail
- **Timeout Issues**: Tests handle network timeouts appropriately
- **Model Availability**: Tests adapt to temporary model unavailability

## 🔧 Troubleshooting

### Common Issues

1. **Missing Credentials**
   ```
   ERROR: configs/credentials.json not found
   ```
   **Solution**: Create credentials file with real API keys

2. **Invalid API Keys**
   ```
   Chat completion returned status 401 (might be API limitation)
   ```
   **Solution**: Verify your API keys are valid and active

3. **Rate Limiting**
   ```
   Request returned status 429: Rate limit exceeded
   ```
   **Solution**: Wait and retry, or use different API keys

4. **Network Issues**
   ```
   Request failed: context deadline exceeded
   ```
   **Solution**: Check internet connection and increase timeout

### Debug Mode

Enable detailed logging for troubleshooting:

```bash
export LOG_LEVEL=debug
export ENVIRONMENT=test
make test-integration
```

### Test-Specific Debugging

1. **Check server logs**: Tests run a real server instance
2. **Monitor API quotas**: Track usage during test runs
3. **Verify configurations**: Ensure all config files are present
4. **Test individual scenarios**: Run specific test functions

## 📈 Test Metrics

The integration tests provide insights into:

- **API Response Times**: How quickly vendors respond
- **Success Rates**: Percentage of successful API calls
- **Model Distribution**: How evenly requests are distributed
- **Error Patterns**: Common failure modes
- **Feature Coverage**: Which features work across vendors

## 🔒 Security Considerations

- **API Keys**: Never commit real API keys to version control
- **Rate Limits**: Be mindful of API quotas during development
- **Cost Management**: Monitor API usage costs
- **Environment Isolation**: Use test-specific configurations

## 🤝 Contributing to Tests

When adding new integration tests:

1. **Real API Calls**: Use actual vendor APIs, not mocks
2. **Error Tolerance**: Handle API failures gracefully
3. **Resource Cleanup**: Ensure tests clean up after themselves
4. **Documentation**: Update this README with new test scenarios
5. **Cost Awareness**: Be mindful of API costs in test design

## 📚 Related Documentation

- `CLAUDE.md`: Project development guide
- `docs/api-reference.md`: Complete API documentation
- `docs/development-guide.md`: Development workflow
- `docs/contributing-guide.md`: Contribution guidelines

---

**Note**: These integration tests are crucial for validating the router's functionality but should be run judiciously due to real API costs and rate limits. Consider using them primarily for:

- Pre-production validation
- Feature verification
- Regression testing
- Performance benchmarking

For daily development, prefer unit tests and use integration tests for comprehensive validation before releases.