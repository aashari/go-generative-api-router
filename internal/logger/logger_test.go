package logger

import (
	"bytes"
	"context"
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
		Level:       LevelDebug,
		Format:      "json",
		Output:      "stdout",
		TimeFormat:  time.RFC3339,
		ServiceName: "test-service",
		Environment: "test",
	}

	// Temporarily redirect output to buffer for testing
	originalLogger := Logger
	defer func() { Logger = originalLogger }()

	// Create a logger that writes to our buffer
	handler := &StructuredJSONHandler{
		writer:      &buf,
		serviceName: config.ServiceName,
		environment: config.Environment,
	}
	Logger = slog.New(handler)

	// Test logging
	Info("Test message", "key", "value")

	// Verify JSON output
	output := buf.String()
	if !strings.Contains(output, "Test message") {
		t.Errorf("Expected log message not found in output: %s", output)
	}

	// Verify it's valid JSON
	var logEntry StructuredLogEntry
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Errorf("Log output is not valid JSON: %v", err)
	}

	// Verify structured fields
	if logEntry.Message != "Test message" {
		t.Errorf("Expected message field to be 'Test message', got: %v", logEntry.Message)
	}
	if logEntry.Service != "test-service" {
		t.Errorf("Expected service field to be 'test-service', got: %v", logEntry.Service)
	}
	if logEntry.Environment != "test" {
		t.Errorf("Expected environment field to be 'test', got: %v", logEntry.Environment)
	}
	if logEntry.Attributes["key"] != "value" {
		t.Errorf("Expected attributes.key field to be 'value', got: %v", logEntry.Attributes["key"])
	}
}

func TestContextAwareLogging(t *testing.T) {
	var buf bytes.Buffer

	// Setup logger with buffer
	originalLogger := Logger
	defer func() { Logger = originalLogger }()

	handler := &StructuredJSONHandler{
		writer:      &buf,
		serviceName: "test-service",
		environment: "test",
	}
	Logger = slog.New(handler)

	// Create context with request ID
	ctx := context.WithValue(context.Background(), RequestIDKey, "test-request-123")
	ctx = context.WithValue(ctx, VendorKey, "openai")
	ctx = context.WithValue(ctx, ModelKey, "gpt-4")

	// Log with context
	InfoCtx(ctx, "Test context logging", "operation", "test")

	// Verify output contains context values
	output := buf.String()

	var logEntry StructuredLogEntry
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Errorf("Log output is not valid JSON: %v", err)
	}

	if logEntry.Request == nil || logEntry.Request["request_id"] != "test-request-123" {
		t.Errorf("Expected request.request_id in log output: %v", logEntry.Request)
	}
	if logEntry.Attributes == nil || logEntry.Attributes["vendor"] != "openai" {
		t.Errorf("Expected attributes.vendor in log output: %v", logEntry.Attributes)
	}
	if logEntry.Attributes == nil || logEntry.Attributes["model"] != "gpt-4" {
		t.Errorf("Expected attributes.model in log output: %v", logEntry.Attributes)
	}
}

func TestStructuredLogging(t *testing.T) {
	var buf bytes.Buffer

	// Setup logger with buffer
	originalLogger := Logger
	originalServiceName := ServiceName
	originalEnvironment := Environment
	defer func() {
		Logger = originalLogger
		ServiceName = originalServiceName
		Environment = originalEnvironment
	}()

	// Set the global service name and environment
	ServiceName = "test-service"
	Environment = "test"

	handler := &StructuredJSONHandler{
		writer:      &buf,
		serviceName: "test-service",
		environment: "test",
	}
	Logger = slog.New(handler)

	ctx := context.WithValue(context.Background(), RequestIDKey, "test-123")

	// Test structured logging with all sections
	attributes := map[string]interface{}{
		"api_key":     "sk-1234567890abcdef", // Should NOT be sanitized
		"model":       "gpt-4",
		"secret_data": "very-secret-value",
	}

	request := map[string]interface{}{
		"method": "POST",
		"path":   "/v1/chat/completions",
	}

	response := map[string]interface{}{
		"status_code": 200,
		"body":        "response content",
	}

	errorData := map[string]interface{}{
		"type":    "ValidationError",
		"message": "Invalid request",
	}

	// Test LogWithStructure
	LogWithStructure(ctx, LevelInfo, "Test structured logging", attributes, request, response, errorData)

	output := buf.String()
	var logEntry StructuredLogEntry
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Errorf("Log output is not valid JSON: %v", err)
		return
	}

	// Verify all sections are present
	if logEntry.Attributes["api_key"] != "sk-1234567890abcdef" {
		t.Errorf("Expected complete API key in attributes (no sanitization): %v", logEntry.Attributes)
	}
	if logEntry.Request["method"] != "POST" {
		t.Errorf("Expected request method in request section: %v", logEntry.Request)
	}
	if logEntry.Response["status_code"] != float64(200) { // JSON unmarshals numbers as float64
		t.Errorf("Expected response status_code in response section: %v", logEntry.Response)
	}
	if logEntry.Error["type"] != "ValidationError" {
		t.Errorf("Expected error type in error section: %v", logEntry.Error)
	}
}

func TestLogRequest(t *testing.T) {
	var buf bytes.Buffer

	// Setup logger with buffer
	originalLogger := Logger
	originalServiceName := ServiceName
	originalEnvironment := Environment
	defer func() {
		Logger = originalLogger
		ServiceName = originalServiceName
		Environment = originalEnvironment
	}()

	// Set the global service name and environment
	ServiceName = "test-service"
	Environment = "test"

	handler := &StructuredJSONHandler{
		writer:      &buf,
		serviceName: "test-service",
		environment: "test",
	}
	Logger = slog.New(handler)

	ctx := context.WithValue(context.Background(), RequestIDKey, "test-123")

	headers := map[string][]string{
		"Authorization": {"Bearer sk-secret-token"},
		"Content-Type":  {"application/json"},
	}
	body := []byte(`{"model":"gpt-4","messages":[{"role":"user","content":"Hello"}]}`)

	LogRequest(ctx, "POST", "/v1/chat/completions", "curl/8.0", headers, body)

	output := buf.String()
	var logEntry StructuredLogEntry
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Errorf("Log output is not valid JSON: %v\nActual output: %s", err, output)
		return
	}

	// Verify request data is in request section
	if logEntry.Request["method"] != "POST" {
		t.Errorf("Expected method in request section: %v", logEntry.Request)
	}
	if logEntry.Request["path"] != "/v1/chat/completions" {
		t.Errorf("Expected path in request section: %v", logEntry.Request)
	}
	if !strings.Contains(logEntry.Request["body"].(string), "Hello") {
		t.Errorf("Expected request body content: %v", logEntry.Request["body"])
	}

	// Verify complete headers are logged (including sensitive data)
	requestHeaders := logEntry.Request["headers"].(map[string]interface{})
	authHeaders := requestHeaders["Authorization"].([]interface{})
	if authHeaders[0] != "Bearer sk-secret-token" {
		t.Errorf("Expected complete authorization header: %v", authHeaders)
	}
}

func TestLogVendorCommunication(t *testing.T) {
	var buf bytes.Buffer

	// Setup logger with buffer
	originalLogger := Logger
	defer func() { Logger = originalLogger }()

	handler := &StructuredJSONHandler{
		writer:      &buf,
		serviceName: "test-service",
		environment: "test",
	}
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
	var logEntry StructuredLogEntry
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Errorf("Log output is not valid JSON: %v", err)
	}

	// Verify vendor in attributes
	if logEntry.Attributes["vendor"] != "openai" {
		t.Errorf("Expected vendor in attributes: %v", logEntry.Attributes)
	}

	// Verify request and response sections
	if !strings.Contains(logEntry.Request["body"].(string), "sk-secret") {
		t.Errorf("Expected complete API key in request body: %v", logEntry.Request["body"])
	}
	if !strings.Contains(logEntry.Response["body"].(string), "Hi there!") {
		t.Errorf("Expected response content: %v", logEntry.Response["body"])
	}
}

func TestLogError(t *testing.T) {
	var buf bytes.Buffer

	// Setup logger with buffer
	originalLogger := Logger
	defer func() { Logger = originalLogger }()

	handler := &StructuredJSONHandler{
		writer:      &buf,
		serviceName: "test-service",
		environment: "test",
	}
	Logger = slog.New(handler)

	ctx := context.Background()

	// Test error logging
	testError := &TestError{message: "test error"}
	details := map[string]any{
		"operation": "test_operation",
		"api_key":   "sk-secret-key",
	}

	LogError(ctx, "test_component", testError, details)

	output := buf.String()
	var logEntry StructuredLogEntry
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Errorf("Log output is not valid JSON: %v", err)
	}

	// Verify error section
	if logEntry.Error["message"] != "test error" {
		t.Errorf("Expected error message in error section: %v", logEntry.Error)
	}
	if !strings.Contains(logEntry.Error["type"].(string), "TestError") {
		t.Errorf("Expected error type in error section: %v", logEntry.Error)
	}

	// Verify attributes section contains details
	if logEntry.Attributes["component"] != "test_component" {
		t.Errorf("Expected component in attributes: %v", logEntry.Attributes)
	}
	if logEntry.Attributes["api_key"] != "sk-secret-key" {
		t.Errorf("Expected complete API key in attributes: %v", logEntry.Attributes)
	}
}

func TestLogLevels(t *testing.T) {
	var buf bytes.Buffer

	// Setup logger with buffer
	originalLogger := Logger
	defer func() { Logger = originalLogger }()

	handler := &StructuredJSONHandler{
		writer:      &buf,
		serviceName: "test-service",
		environment: "test",
	}
	Logger = slog.New(handler)

	// Test all log levels
	Debug("Debug message", "key", "debug_value")
	Info("Info message", "key", "info_value")
	Warn("Warn message", "key", "warn_value")
	Error("Error message", "key", "error_value")

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Verify all levels are logged
	levels := []string{"DEBUG", "INFO", "WARN", "ERROR"}
	messages := []string{"Debug message", "Info message", "Warn message", "Error message"}

	for i, line := range lines {
		var logEntry StructuredLogEntry
		if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
			t.Errorf("Log line %d is not valid JSON: %v", i, err)
			continue
		}

		if logEntry.Level != levels[i] {
			t.Errorf("Expected level %s, got %s", levels[i], logEntry.Level)
		}
		if logEntry.Message != messages[i] {
			t.Errorf("Expected message %s, got %s", messages[i], logEntry.Message)
		}
	}
}

// TestError is a simple error type for testing
type TestError struct {
	message string
}

func (e *TestError) Error() string {
	return e.message
}

// TestLegacyCompatibility removed - legacy functions have been removed
