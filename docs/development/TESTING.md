# Testing Guide

This guide covers testing strategies, best practices, and detailed procedures for the Generative API Router project.

> **📚 Complete Testing Guide**: For comprehensive testing procedures including manual testing, debugging, and troubleshooting, see [Running & Testing Guide](../../.cursor/rules/running_and_testing.mdc).

## 🧪 Testing Overview

The project maintains comprehensive test coverage across all components with multiple testing strategies:

- **Unit Tests**: Test individual functions and methods
- **Integration Tests**: Test component interactions
- **Manual API Tests**: Verify end-to-end functionality
- **Performance Tests**: Ensure acceptable response times
- **Security Tests**: Validate input sanitization and error handling

## 🚀 Running Tests

### Quick Commands
```bash
# Run all tests
make test

# Run with coverage report
make test-coverage

# Run with race detection
make test-race

# Run specific package tests
go test ./internal/handlers

# Run with verbose output
go test -v ./...

# Run tests matching a pattern
go test -run TestProxyHandler ./internal/proxy
```

### Advanced Test Execution
```bash
# Run tests with coverage and generate HTML report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Run tests with memory profiling
go test -memprofile=mem.prof ./internal/proxy

# Run tests with CPU profiling
go test -cpuprofile=cpu.prof ./internal/selector

# Run tests with timeout
go test -timeout 30s ./...

# Run tests in parallel
go test -parallel 4 ./...
```

## 📁 Test Structure

### Directory Organization
```
├── internal/
│   ├── app/
│   │   ├── app.go
│   │   └── app_test.go          # Unit tests for app package
│   ├── proxy/
│   │   ├── proxy.go
│   │   ├── proxy_test.go        # Unit tests for proxy
│   │   └── integration_test.go  # Integration tests
│   └── ...
├── testdata/
│   ├── fixtures/                # Test data files
│   │   ├── requests.json
│   │   └── responses.json
│   └── analysis/                # Test analysis and reports
│       └── COMPARISON_REPORT.md
└── examples/
    └── curl/                    # Manual testing scripts
        ├── basic.sh
        ├── streaming.sh
        └── tools.sh
```

### Test File Naming
- **Unit tests**: `*_test.go` colocated with source files
- **Integration tests**: `integration_test.go` or `*_integration_test.go`
- **Benchmark tests**: `*_bench_test.go`
- **Example tests**: `example_*_test.go`

## ✍️ Writing Tests

### Unit Test Template
```go
package handlers

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestHealthHandler(t *testing.T) {
    // Arrange
    req := httptest.NewRequest("GET", "/health", nil)
    w := httptest.NewRecorder()
    
    // Act
    HealthHandler(w, req)
    
    // Assert
    assert.Equal(t, http.StatusOK, w.Code)
    assert.Equal(t, "OK", w.Body.String())
}
```

### Table-Driven Tests
```go
func TestVendorSelection(t *testing.T) {
    tests := []struct {
        name        string
        credentials []Credential
        models      []Model
        vendor      string
        expected    string
        wantErr     bool
    }{
        {
            name: "successful selection with vendor filter",
            credentials: []Credential{
                {Platform: "openai", Value: "key1"},
            },
            models: []Model{
                {Vendor: "openai", Model: "gpt-4o"},
            },
            vendor:   "openai",
            expected: "gpt-4o",
            wantErr:  false,
        },
        {
            name:     "no credentials available",
            vendor:   "openai",
            wantErr:  true,
        },
        // More test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            selector := NewSelector(tt.credentials, tt.models)
            
            result, err := selector.Select(tt.vendor)
            
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            
            require.NoError(t, err)
            assert.Equal(t, tt.expected, result.Model)
        })
    }
}
```

### Integration Test Example
```go
func TestProxyIntegration(t *testing.T) {
    // Skip if no credentials available
    if !hasValidCredentials() {
        t.Skip("Skipping integration test: no valid credentials")
    }
    
    // Setup test server
    app := setupTestApp(t)
    server := httptest.NewServer(app.Router)
    defer server.Close()
    
    // Test request
    reqBody := `{
        "model": "test-model",
        "messages": [{"role": "user", "content": "Hello"}]
    }`
    
    resp, err := http.Post(
        server.URL+"/v1/chat/completions",
        "application/json",
        strings.NewReader(reqBody),
    )
    
    require.NoError(t, err)
    defer resp.Body.Close()
    
    // Verify response
    assert.Equal(t, http.StatusOK, resp.StatusCode)
    
    var response ChatCompletionResponse
    err = json.NewDecoder(resp.Body).Decode(&response)
    require.NoError(t, err)
    
    // Verify transparent proxy behavior
    assert.Equal(t, "test-model", response.Model)
    assert.NotEmpty(t, response.ID)
    assert.NotEmpty(t, response.Choices)
}
```

### Mock Testing
```go
type MockVendorClient struct {
    mock.Mock
}

func (m *MockVendorClient) SendRequest(ctx context.Context, req Request) (*Response, error) {
    args := m.Called(ctx, req)
    return args.Get(0).(*Response), args.Error(1)
}

func TestProxyWithMock(t *testing.T) {
    // Setup mock
    mockClient := new(MockVendorClient)
    mockClient.On("SendRequest", mock.Anything, mock.Anything).Return(
        &Response{Model: "gpt-4o", Content: "Hello!"},
        nil,
    )
    
    // Test with mock
    proxy := NewProxy(mockClient)
    response, err := proxy.HandleRequest(context.Background(), testRequest)
    
    require.NoError(t, err)
    assert.Equal(t, "test-model", response.Model) // Transparent proxy
    
    // Verify mock was called
    mockClient.AssertExpectations(t)
}
```

## 🎯 Testing Best Practices

### Test Organization
1. **Arrange, Act, Assert**: Structure tests clearly
2. **One assertion per test**: Focus on single behavior
3. **Descriptive names**: Test names should explain what's being tested
4. **Independent tests**: Tests should not depend on each other
5. **Clean up**: Use `defer` for cleanup or `t.Cleanup()`

### Error Testing
```go
func TestValidationErrors(t *testing.T) {
    tests := []struct {
        name        string
        input       Request
        expectedErr string
    }{
        {
            name:        "missing messages",
            input:       Request{Model: "gpt-4o"},
            expectedErr: "messages field is required",
        },
        {
            name:        "empty model",
            input:       Request{Messages: []Message{{Role: "user", Content: "hi"}}},
            expectedErr: "model field is required",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateRequest(tt.input)
            
            require.Error(t, err)
            assert.Contains(t, err.Error(), tt.expectedErr)
        })
    }
}
```

### Context Testing
```go
func TestWithContext(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()
    
    // Test that function respects context cancellation
    err := LongRunningFunction(ctx)
    
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "context deadline exceeded")
}
```

### Concurrent Testing
```go
func TestConcurrentAccess(t *testing.T) {
    selector := NewSelector(credentials, models)
    
    var wg sync.WaitGroup
    errors := make(chan error, 10)
    
    // Run 10 concurrent selections
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            _, err := selector.Select("")
            if err != nil {
                errors <- err
            }
        }()
    }
    
    wg.Wait()
    close(errors)
    
    // Check for race conditions or errors
    for err := range errors {
        t.Errorf("Concurrent access error: %v", err)
    }
}
```

## 📊 Test Coverage

### Coverage Goals
- **Overall coverage**: >80%
- **Critical paths**: >95% (proxy, selector, validator)
- **Error handling**: 100% of error paths tested
- **Public APIs**: 100% of exported functions tested

### Measuring Coverage
```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View coverage by package
go tool cover -func=coverage.out

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html

# Check coverage threshold
go test -cover ./... | grep -E "coverage: [0-9]+\.[0-9]+%" | awk '{if($2 < 80.0) print "Low coverage: " $0}'
```

### Coverage Analysis
```bash
# Find uncovered lines
go tool cover -html=coverage.out

# Coverage by function
go tool cover -func=coverage.out | sort -k3 -nr

# Package-level coverage summary
go test -coverprofile=coverage.out ./... && \
go tool cover -func=coverage.out | grep "total:" | \
awk '{print "Total coverage: " $3}'
```

## 🔧 Test Configuration

### Environment Variables
```bash
# Skip integration tests
export SKIP_INTEGRATION_TESTS=true

# Use test credentials
export TEST_CREDENTIALS_FILE=testdata/fixtures/test_credentials.json

# Enable verbose logging in tests
export LOG_LEVEL=DEBUG

# Set test timeout
export TEST_TIMEOUT=30s
```

### Test Helpers
```go
// testhelpers/setup.go
func SetupTestApp(t *testing.T) *app.App {
    t.Helper()
    
    // Load test configuration
    config := loadTestConfig(t)
    
    // Create test app
    app, err := app.New(config)
    require.NoError(t, err)
    
    return app
}

func LoadTestCredentials(t *testing.T) []Credential {
    t.Helper()
    
    data, err := os.ReadFile("testdata/fixtures/credentials.json")
    require.NoError(t, err)
    
    var creds []Credential
    err = json.Unmarshal(data, &creds)
    require.NoError(t, err)
    
    return creds
}
```

## 🚨 Manual Testing

### API Testing Scripts
The project includes manual testing scripts in `examples/curl/`:

```bash
# Basic functionality
./examples/curl/basic.sh

# Streaming responses
./examples/curl/streaming.sh

# Tool calling
./examples/curl/tools.sh

# Error handling
./examples/curl/errors.sh
```

### Manual Test Checklist
- [ ] Health endpoint responds correctly
- [ ] Models endpoint lists available models
- [ ] Chat completions work with different vendors
- [ ] Streaming responses work correctly
- [ ] Tool calling functions properly
- [ ] Error responses are properly formatted
- [ ] Vendor filtering works via query parameters
- [ ] Request IDs are consistent in streaming
- [ ] Model names are preserved in responses

## 🐛 Debugging Tests

### Common Issues
1. **Flaky tests**: Use `go test -count=100` to identify
2. **Race conditions**: Run with `-race` flag
3. **Memory leaks**: Use `-memprofile` for analysis
4. **Slow tests**: Use `-timeout` and `-cpuprofile`

### Debugging Techniques
```bash
# Run single test with verbose output
go test -v -run TestSpecificFunction ./internal/proxy

# Debug with delve
dlv test ./internal/proxy -- -test.run TestSpecificFunction

# Print test output even on success
go test -v ./... | grep -E "(PASS|FAIL|RUN)"

# Run tests with custom build tags
go test -tags=integration ./...
```

## 📈 Performance Testing

### Benchmark Tests
```go
func BenchmarkVendorSelection(b *testing.B) {
    selector := setupBenchmarkSelector()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := selector.Select("")
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkProxyRequest(b *testing.B) {
    proxy := setupBenchmarkProxy()
    request := createBenchmarkRequest()
    
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            _, err := proxy.HandleRequest(context.Background(), request)
            if err != nil {
                b.Fatal(err)
            }
        }
    })
}
```

### Running Benchmarks
```bash
# Run all benchmarks
go test -bench=. ./...

# Run specific benchmark
go test -bench=BenchmarkVendorSelection ./internal/selector

# Run with memory allocation stats
go test -bench=. -benchmem ./...

# Compare benchmarks
go test -bench=. -count=5 ./... > old.txt
# Make changes
go test -bench=. -count=5 ./... > new.txt
benchcmp old.txt new.txt
```

## 📚 Additional Resources

- **[Development Guide](./DEVELOPMENT.md)** - Complete development setup
- **[Contributing Guidelines](./CONTRIBUTING.md)** - How to contribute
- **[Running & Testing Guide](../../.cursor/rules/running_and_testing.mdc)** - Comprehensive testing procedures
- **[Go Testing Documentation](https://golang.org/pkg/testing/)** - Official Go testing docs
- **[Testify Documentation](https://github.com/stretchr/testify)** - Testing toolkit used in this project

---

**Remember**: Good tests are as important as good code. They serve as documentation, catch regressions, and enable confident refactoring. 🧪 