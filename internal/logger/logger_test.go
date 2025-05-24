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

func TestSensitiveDataSanitization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "OpenAI API Key",
			input:    "sk-1234567890abcdef",
			expected: "sk-1****cdef",
		},
		{
			name:     "Google API Key",
			input:    "AIzaSyExample_Fake_Key_For_Testing_Only",
			expected: "AIza****Only",
		},
		{
			name:     "Short sensitive value",
			input:    "secret",
			expected: "****",
		},
		{
			name:     "Non-sensitive value",
			input:    "normal-value",
			expected: "normal-value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeValue(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeValue(%s) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSensitiveKeyDetection(t *testing.T) {
	tests := []struct {
		key         string
		isSensitive bool
	}{
		{"api_key", true},
		{"api-key", true},
		{"apikey", true},
		{"token", true},
		{"secret", true},
		{"password", true},
		{"authorization", true},
		{"auth", true},
		{"normal_field", false},
		{"username", false},
		{"model", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := isSensitiveKey(tt.key)
			if result != tt.isSensitive {
				t.Errorf("isSensitiveKey(%s) = %v, expected %v", tt.key, result, tt.isSensitive)
			}
		})
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

func TestSpecializedLoggingFunctions(t *testing.T) {
	var buf bytes.Buffer

	// Setup logger with buffer
	originalLogger := Logger
	defer func() { Logger = originalLogger }()

	opts := &slog.HandlerOptions{Level: LevelDebug}
	handler := slog.NewJSONHandler(&buf, opts)
	Logger = slog.New(handler)

	ctx := context.WithValue(context.Background(), RequestIDKey, "test-123")

	// Test LogProxyRequest
	LogProxyRequest(ctx, "my-model", "openai", "gpt-4", 10)

	output := buf.String()
	if !strings.Contains(output, "Proxy request initiated") {
		t.Errorf("Expected proxy request log message: %s", output)
	}
	if !strings.Contains(output, "my-model") {
		t.Errorf("Expected original model in log: %s", output)
	}
	if !strings.Contains(output, "openai") {
		t.Errorf("Expected vendor in log: %s", output)
	}

	// Reset buffer for next test
	buf.Reset()

	// Test LogVendorResponse
	LogVendorResponse(ctx, "openai", "gpt-4-actual", "my-model", 1024, 500*time.Millisecond)

	output = buf.String()
	if !strings.Contains(output, "Vendor response processed") {
		t.Errorf("Expected vendor response log message: %s", output)
	}
	if !strings.Contains(output, "1024") {
		t.Errorf("Expected response size in log: %s", output)
	}
}

func TestSanitizeMap(t *testing.T) {
	input := map[string]any{
		"api_key":     "sk-1234567890abcdef",
		"model":       "gpt-4",
		"secret_key":  "very-secret-value",
		"normal_data": "public-info",
	}

	result := SanitizeMap(input)

	// Check that sensitive fields are sanitized
	if result["api_key"] == input["api_key"] {
		t.Error("API key should be sanitized")
	}
	if result["secret_key"] == input["secret_key"] {
		t.Error("Secret key should be sanitized")
	}

	// Check that non-sensitive fields are preserved
	if result["model"] != input["model"] {
		t.Error("Model field should be preserved")
	}
	if result["normal_data"] != input["normal_data"] {
		t.Error("Normal data should be preserved")
	}
}

func TestLogLevels(t *testing.T) {
	var buf bytes.Buffer

	// Setup logger with INFO level
	originalLogger := Logger
	defer func() { Logger = originalLogger }()

	opts := &slog.HandlerOptions{Level: LevelInfo}
	handler := slog.NewJSONHandler(&buf, opts)
	Logger = slog.New(handler)

	// Debug should not appear
	Debug("Debug message")
	if strings.Contains(buf.String(), "Debug message") {
		t.Error("Debug message should not appear with INFO level")
	}

	// Info should appear
	Info("Info message")
	if !strings.Contains(buf.String(), "Info message") {
		t.Error("Info message should appear with INFO level")
	}

	// Error should appear
	Error("Error message")
	if !strings.Contains(buf.String(), "Error message") {
		t.Error("Error message should appear with INFO level")
	}
}
