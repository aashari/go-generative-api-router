package logger

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestLoggerInitialization(t *testing.T) {
	// Test JSON format initialization
	var buf bytes.Buffer
	config := Config{
		Level:      LevelDebug,
		Format:     "json",
		Output:     "stdout",
		TimeFormat: time.RFC3339,
	}

	// Temporarily redirect output to buffer for testing
	originalLogger := Logger
	defer func() { Logger = originalLogger }()

	// Create a logger that writes to our buffer
	opts := &slog.HandlerOptions{Level: config.Level}
	handler := slog.NewJSONHandler(&buf, opts)
	Logger = slog.New(handler)

	// Test logging
	Info("Test message", "key", "value")

	// Verify JSON output
	output := buf.String()
	if !strings.Contains(output, "Test message") {
		t.Errorf("Expected log message not found in output: %s", output)
	}

	// Verify it's valid JSON
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Errorf("Log output is not valid JSON: %v", err)
	}

	// Verify structured fields
	if logEntry["msg"] != "Test message" {
		t.Errorf("Expected msg field to be 'Test message', got: %v", logEntry["msg"])
	}
	if logEntry["key"] != "value" {
		t.Errorf("Expected key field to be 'value', got: %v", logEntry["key"])
	}
}

func TestContextAwareLogging(t *testing.T) {
	var buf bytes.Buffer

	// Setup logger with buffer
	originalLogger := Logger
	defer func() { Logger = originalLogger }()

	opts := &slog.HandlerOptions{Level: LevelDebug}
	handler := slog.NewJSONHandler(&buf, opts)
	Logger = slog.New(handler)

	// Create context with request ID
	ctx := context.WithValue(context.Background(), RequestIDKey, "test-request-123")
	ctx = context.WithValue(ctx, VendorKey, "openai")
	ctx = context.WithValue(ctx, ModelKey, "gpt-4")

	// Log with context
	InfoCtx(ctx, "Test context logging", "operation", "test")

	// Verify output contains context values
	output := buf.String()
	if !strings.Contains(output, "test-request-123") {
		t.Errorf("Expected request ID in log output: %s", output)
	}
	if !strings.Contains(output, "openai") {
		t.Errorf("Expected vendor in log output: %s", output)
	}
	if !strings.Contains(output, "gpt-4") {
		t.Errorf("Expected model in log output: %s", output)
	}
}

func TestCompleteDataLogging(t *testing.T) {
	var buf bytes.Buffer

	// Setup logger with buffer
	originalLogger := Logger
	defer func() { Logger = originalLogger }()

	opts := &slog.HandlerOptions{Level: LevelDebug}
	handler := slog.NewJSONHandler(&buf, opts)
	Logger = slog.New(handler)

	ctx := context.WithValue(context.Background(), RequestIDKey, "test-123")

	// Test complex data structure
	testData := map[string]interface{}{
		"api_key":     "sk-1234567890abcdef", // Should NOT be sanitized
		"model":       "gpt-4",
		"secret_data": "very-secret-value",
		"nested": map[string]interface{}{
			"credentials": "sensitive-info",
			"tokens":      []string{"token1", "token2"},
		},
	}

	// Test LogCompleteDataInfo
	LogCompleteDataInfo(ctx, "Test complete data logging", testData)

	output := buf.String()

	// Verify complete data is logged as JSON
	if !strings.Contains(output, "sk-1234567890abcdef") {
		t.Errorf("Expected complete API key in log output (no sanitization): %s", output)
	}
	if !strings.Contains(output, "very-secret-value") {
		t.Errorf("Expected complete secret data in log output: %s", output)
	}
	if !strings.Contains(output, "sensitive-info") {
		t.Errorf("Expected complete nested sensitive data in log output: %s", output)
	}
	if !strings.Contains(output, "complete_data") {
		t.Errorf("Expected complete_data field in log output: %s", output)
	}
}

func TestLogMultipleData(t *testing.T) {
	var buf bytes.Buffer

	// Setup logger with buffer
	originalLogger := Logger
	defer func() { Logger = originalLogger }()

	opts := &slog.HandlerOptions{Level: LevelDebug}
	handler := slog.NewJSONHandler(&buf, opts)
	Logger = slog.New(handler)

	ctx := context.WithValue(context.Background(), RequestIDKey, "test-123")

	// Test multiple data logging with complete data structures
	requestData := map[string]string{
		"method": "POST",
		"path":   "/v1/chat/completions",
	}

	credentialsData := map[string]string{
		"api_key": "sk-secret-key",
		"token":   "bearer-token",
	}

	responseData := []byte(`{"id":"chatcmpl-123","object":"chat.completion"}`)

	LogMultipleData(ctx, LevelInfo, "Test multiple data logging", map[string]any{
		"request":     requestData,
		"credentials": credentialsData,
		"response":    responseData,
	})

	output := buf.String()
	t.Logf("Expected complete response data in log: %s", output)

	// Verify all data is logged completely
	if !strings.Contains(output, "sk-secret-key") {
		t.Errorf("Expected complete API key in multiple data log: %s", output)
	}
	if !strings.Contains(output, "bearer-token") {
		t.Errorf("Expected complete token in multiple data log: %s", output)
	}
	// The response data is base64 encoded, so check for the base64 representation
	expectedBase64 := base64.StdEncoding.EncodeToString(responseData)
	if !strings.Contains(output, expectedBase64) {
		t.Errorf("Expected complete response data (base64) in log: %s", output)
	}
}

func TestLogRequest(t *testing.T) {
	var buf bytes.Buffer

	// Setup logger with buffer
	originalLogger := Logger
	defer func() { Logger = originalLogger }()

	opts := &slog.HandlerOptions{Level: LevelDebug}
	handler := slog.NewJSONHandler(&buf, opts)
	Logger = slog.New(handler)

	ctx := context.WithValue(context.Background(), RequestIDKey, "test-123")

	headers := map[string][]string{
		"Authorization": {"Bearer sk-secret-token"},
		"Content-Type":  {"application/json"},
	}
	body := []byte(`{"model":"gpt-4","messages":[{"role":"user","content":"Hello"}]}`)

	LogRequest(ctx, "POST", "/v1/chat/completions", "curl/8.0", headers, body)

	output := buf.String()

	// Verify complete request data is logged
	if !strings.Contains(output, "sk-secret-token") {
		t.Errorf("Expected complete authorization header in request log: %s", output)
	}
	if !strings.Contains(output, "Hello") {
		t.Errorf("Expected complete request body in log: %s", output)
	}
	if !strings.Contains(output, "Complete HTTP request data") {
		t.Errorf("Expected request log message: %s", output)
	}
}

func TestLogVendorCommunication(t *testing.T) {
	var buf bytes.Buffer

	// Setup logger with buffer
	originalLogger := Logger
	defer func() { Logger = originalLogger }()

	opts := &slog.HandlerOptions{Level: LevelDebug}
	handler := slog.NewJSONHandler(&buf, opts)
	Logger = slog.New(handler)

	ctx := context.WithValue(context.Background(), RequestIDKey, "test-123")

	requestBody := []byte(`{"model":"gpt-4","messages":[{"role":"user","content":"Hello"}],"api_key":"sk-secret"}`)
	responseBody := []byte(`{"id":"chatcmpl-123","choices":[{"message":{"content":"Hi there!"}}]}`)

	requestHeaders := map[string][]string{
		"Authorization": {"Bearer sk-secret-key"},
	}
	responseHeaders := map[string][]string{
		"Content-Type": {"application/json"},
	}

	LogVendorCommunication(ctx, "openai", "https://api.openai.com/v1/chat/completions",
		requestBody, responseBody, requestHeaders, responseHeaders)

	output := buf.String()

	// Verify complete vendor communication is logged
	if !strings.Contains(output, "sk-secret") {
		t.Errorf("Expected complete API key in vendor communication log: %s", output)
	}
	if !strings.Contains(output, "sk-secret-key") {
		t.Errorf("Expected complete authorization header in vendor communication log: %s", output)
	}
	if !strings.Contains(output, "Hi there!") {
		t.Errorf("Expected complete response content in vendor communication log: %s", output)
	}
}

func TestSpecializedLoggingFunctions(t *testing.T) {
	var buf bytes.Buffer

	// Setup logger with buffer
	originalLogger := Logger
	defer func() { Logger = originalLogger }()

	opts := &slog.HandlerOptions{Level: LevelDebug}
	handler := slog.NewJSONHandler(&buf, opts)
	Logger = slog.New(handler)

	ctx := context.WithValue(context.Background(), RequestIDKey, "test-123")

	// Test LogProxyRequest with complete data
	requestData := map[string]interface{}{
		"model":   "my-model",
		"api_key": "sk-secret-key",
		"messages": []map[string]string{
			{"role": "user", "content": "Hello"},
		},
	}

	LogProxyRequest(ctx, "my-model", "openai", "gpt-4", 10, requestData)

	output := buf.String()
	if !strings.Contains(output, "Proxy request initiated with complete data") {
		t.Errorf("Expected proxy request log message: %s", output)
	}
	if !strings.Contains(output, "sk-secret-key") {
		t.Errorf("Expected complete API key in proxy request log: %s", output)
	}
	if !strings.Contains(output, "my-model") {
		t.Errorf("Expected original model in log: %s", output)
	}
	if !strings.Contains(output, "openai") {
		t.Errorf("Expected vendor in log: %s", output)
	}

	// Reset buffer for next test
	buf.Reset()

	// Test LogVendorResponse with complete data
	responseData := map[string]interface{}{
		"id": "chatcmpl-123",
		"choices": []map[string]interface{}{
			{"message": map[string]string{"content": "Hello there!"}},
		},
		"usage": map[string]int{
			"prompt_tokens":     10,
			"completion_tokens": 5,
		},
	}

	LogVendorResponse(ctx, "openai", "gpt-4-actual", "my-model", 1024, 500*time.Millisecond, responseData)

	output = buf.String()
	if !strings.Contains(output, "Vendor response processed with complete data") {
		t.Errorf("Expected vendor response log message: %s", output)
	}
	if !strings.Contains(output, "Hello there!") {
		t.Errorf("Expected complete response content in log: %s", output)
	}
	if !strings.Contains(output, "1024") {
		t.Errorf("Expected response size in log: %s", output)
	}
}

func TestLogCredentials(t *testing.T) {
	var buf bytes.Buffer

	// Setup logger with buffer
	originalLogger := Logger
	defer func() { Logger = originalLogger }()

	opts := &slog.HandlerOptions{Level: LevelDebug}
	handler := slog.NewJSONHandler(&buf, opts)
	Logger = slog.New(handler)

	ctx := context.Background()

	// Test logging complete credentials (including sensitive data)
	credentials := []map[string]string{
		{
			"platform": "openai",
			"type":     "api-key",
			"value":    "sk-1234567890abcdef",
		},
		{
			"platform": "gemini",
			"type":     "api-key",
			"value":    "AIzaSyExample_Secret_Key",
		},
	}

	LogCredentials(ctx, credentials)

	output := buf.String()

	// Verify complete credentials are logged (no sanitization)
	if !strings.Contains(output, "sk-1234567890abcdef") {
		t.Errorf("Expected complete OpenAI API key in credentials log: %s", output)
	}
	if !strings.Contains(output, "AIzaSyExample_Secret_Key") {
		t.Errorf("Expected complete Gemini API key in credentials log: %s", output)
	}
	if !strings.Contains(output, "Complete credentials data") {
		t.Errorf("Expected credentials log message: %s", output)
	}
}

func TestLogLevels(t *testing.T) {
	var buf bytes.Buffer

	// Setup logger with buffer
	originalLogger := Logger
	defer func() { Logger = originalLogger }()

	opts := &slog.HandlerOptions{Level: LevelDebug}
	handler := slog.NewJSONHandler(&buf, opts)
	Logger = slog.New(handler)

	ctx := context.Background()
	testData := map[string]string{"key": "value"}

	// Test all log levels with complete data
	LogCompleteDataDebug(ctx, "Debug message", testData)
	LogCompleteDataInfo(ctx, "Info message", testData)
	LogCompleteDataWarn(ctx, "Warn message", testData)
	LogCompleteDataError(ctx, "Error message", testData)

	output := buf.String()

	// Verify all levels are logged
	if !strings.Contains(output, "Debug message") {
		t.Errorf("Expected debug message in log output: %s", output)
	}
	if !strings.Contains(output, "Info message") {
		t.Errorf("Expected info message in log output: %s", output)
	}
	if !strings.Contains(output, "Warn message") {
		t.Errorf("Expected warn message in log output: %s", output)
	}
	if !strings.Contains(output, "Error message") {
		t.Errorf("Expected error message in log output: %s", output)
	}
}
