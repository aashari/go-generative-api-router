# Test Structure Documentation

This directory contains the improved, organized test structure for the Generative API Router project.

## 📁 Directory Structure

```
test/
├── README.md                    # This documentation
├── helpers/                     # Shared test utilities
│   ├── test_server.go          # Test server setup and utilities
│   └── types.go                # Common test types and structures
├── fixtures/                    # Test data and request fixtures
│   └── requests.go             # Pre-built request fixtures
├── integration/                 # Integration tests (organized by feature)
│   ├── health_test.go          # Health endpoint tests
│   ├── models_test.go          # Models endpoint tests
│   ├── chat_basic_test.go      # Basic chat completion tests
│   └── error_handling_test.go  # Error handling tests
└── unit/                       # Unit tests (future expansion)
```

## 🧪 Test Categories

### **Unit Tests**
- **Location**: `test/unit/` (future expansion)
- **Purpose**: Test individual components in isolation
- **Current**: Unit tests remain in `internal/*/` directories

### **Integration Tests**
- **Location**: `test/integration/`
- **Purpose**: Test complete API workflows with real server
- **Features**: Organized by functionality, shared utilities

### **Legacy Tests**
- **Location**: Root directory (`integration_test.go`, etc.)
- **Status**: Maintained for compatibility, will be migrated

## 🛠️ Test Utilities

### **TestServer Helper** (`helpers/test_server.go`)

Provides a reusable test server with common utilities:

```go
// Create test server
config := helpers.DefaultTestConfig()
config.ServiceName = "my-test"
ts := helpers.NewTestServer(t, config)
defer ts.Close()

// Make requests
resp, body, err := ts.MakeRequest("GET", "/health", nil, nil)

// Assertions
ts.AssertStatusCode(resp, 200)
ts.AssertJSONResponse(body, &response)
```

### **Test Fixtures** (`fixtures/requests.go`)

Pre-built request patterns for common scenarios:

```go
// Basic requests
request := fixtures.BasicChatRequest()
request := fixtures.StreamingChatRequest()
request := fixtures.VisionChatRequest()

// Error scenarios
request := fixtures.InvalidRequest()
request := fixtures.LargeChatRequest()
```

### **Common Types** (`helpers/types.go`)

Shared type definitions for all tests:

```go
var healthResp helpers.HealthResponse
var chatResp helpers.ChatCompletionResponse
var errorResp helpers.ErrorResponse
```

## 🚀 Running Tests

### **New Structured Tests**
```bash
# Run all new integration tests
make test-integration

# Run specific test files
go test -v ./test/integration/health_test.go
go test -v ./test/integration/models_test.go
go test -v ./test/integration/chat_basic_test.go
```

### **Legacy Tests** (for compatibility)
```bash
# Run legacy integration tests
make test-integration-legacy

# Run specific legacy tests
go test -v ./integration_test.go
go test -v ./quick_integration_test.go
```

### **All Tests**
```bash
# Run unit + new integration tests
make test-all

# Run with coverage
make test-coverage
```

## ✅ Benefits of New Structure

### **1. Better Organization**
- ✅ Tests grouped by functionality
- ✅ Clear separation of concerns
- ✅ Easy to find and maintain

### **2. Reduced Code Duplication**
- ✅ Shared test server setup
- ✅ Common request fixtures
- ✅ Reusable assertion helpers

### **3. Improved Maintainability**
- ✅ Smaller, focused test files
- ✅ Consistent testing patterns
- ✅ Easy to add new tests

### **4. Enhanced Readability**
- ✅ Clear test names and structure
- ✅ Comprehensive documentation
- ✅ Logical test grouping

## 📋 Test Writing Guidelines

### **1. Use Shared Utilities**
```go
// ✅ Good: Use shared test server
ts := helpers.NewTestServer(t, helpers.DefaultTestConfig())
defer ts.Close()

// ❌ Avoid: Duplicating server setup
```

### **2. Use Fixtures for Common Requests**
```go
// ✅ Good: Use pre-built fixtures
request := fixtures.BasicChatRequest()

// ❌ Avoid: Building requests manually every time
```

### **3. Follow Naming Conventions**
```go
// ✅ Good: Descriptive test names
func TestChatCompletionsBasic(t *testing.T) {
    t.Run("model_name_preservation", func(t *testing.T) {
        // Test model name preservation
    })
}
```

### **4. Handle API Limitations Gracefully**
```go
// ✅ Good: Handle both success and API limitations
if resp.StatusCode == 200 {
    // Verify successful response
} else {
    t.Logf("Request returned status %d (might be API limitation)", resp.StatusCode)
}
```

## 🔄 Migration Plan

### **Phase 1: New Structure** ✅
- [x] Create organized test directories
- [x] Build shared utilities and fixtures
- [x] Implement core integration tests
- [x] Update Makefile with new targets

### **Phase 2: Gradual Migration** (Future)
- [ ] Migrate remaining legacy tests
- [ ] Add unit tests to `test/unit/`
- [ ] Expand test coverage
- [ ] Remove legacy test files

### **Phase 3: Advanced Features** (Future)
- [ ] Add performance tests
- [ ] Add load testing utilities
- [ ] Add test data generators
- [ ] Add mock server utilities

## 🎯 Test Coverage Goals

### **Current Coverage**
- ✅ Health endpoint
- ✅ Models endpoint  
- ✅ Basic chat completions
- ✅ Error handling
- ✅ Multi-vendor routing

### **Future Expansion**
- [ ] Streaming responses
- [ ] Vision API
- [ ] Tool calling
- [ ] Advanced error scenarios
- [ ] Performance testing
- [ ] Database integration tests

## 📚 Related Documentation

- **[Development Guide](../.cursor/rules/development_guide.mdc)** - Complete development workflow
- **[Running Guide](../.cursor/rules/running_and_testing.mdc)** - Setup and testing procedures
- **[API Reference](../docs/api-reference.md)** - Complete API documentation
- **[Examples](../examples/)** - Usage examples for different languages 