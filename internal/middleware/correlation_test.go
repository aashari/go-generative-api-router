package middleware

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aashari/go-generative-api-router/internal/logger"
)

func TestRequestCorrelationMiddleware_GzippedResponse(t *testing.T) {
	// Initialize logger for testing
	config := logger.Config{
		Level:       logger.LevelDebug,
		Format:      "json",
		Output:      "stdout",
		ServiceName: "test",
		Environment: "test",
	}
	logger.Init(config)

	// Create test JSON response
	testResponse := map[string]interface{}{
		"message": "Hello, World!",
		"status":  "success",
		"data":    []string{"item1", "item2"},
	}
	responseJSON, _ := json.Marshal(testResponse)

	// Compress the JSON response
	var gzipBuffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&gzipBuffer)
	gzipWriter.Write(responseJSON)
	gzipWriter.Close()
	gzippedResponse := gzipBuffer.Bytes()

	// Create a test handler that returns gzipped JSON
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Content-Encoding", "gzip")
		w.WriteHeader(http.StatusOK)
		w.Write(gzippedResponse)
	})

	// Wrap with correlation middleware
	middlewareHandler := RequestCorrelationMiddleware(testHandler)

	// Create test request
	req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"test": "data"}`))
	req.Header.Set("Content-Type", "application/json")

	// Create response recorder
	rr := httptest.NewRecorder()

	// Execute request
	middlewareHandler.ServeHTTP(rr, req)

	// Verify response
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", rr.Code)
	}

	// Verify Content-Encoding header is preserved
	if rr.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf("Expected Content-Encoding: gzip, got %s", rr.Header().Get("Content-Encoding"))
	}

	// Verify response body is gzipped
	responseBody := rr.Body.Bytes()
	if len(responseBody) == 0 {
		t.Error("Expected non-empty response body")
	}

	// Verify we can decompress the response
	gzipReader, err := gzip.NewReader(bytes.NewReader(responseBody))
	if err != nil {
		t.Errorf("Failed to create gzip reader: %v", err)
	}
	defer gzipReader.Close()

	decompressedBytes, err := io.ReadAll(gzipReader)
	if err != nil {
		t.Errorf("Failed to decompress response: %v", err)
	}

	var decompressedResponse map[string]interface{}
	if err := json.Unmarshal(decompressedBytes, &decompressedResponse); err != nil {
		t.Errorf("Failed to parse decompressed JSON: %v", err)
	}

	// Verify the content matches
	if decompressedResponse["message"] != "Hello, World!" {
		t.Errorf("Expected message 'Hello, World!', got %v", decompressedResponse["message"])
	}
}

func TestRequestCorrelationMiddleware_NonGzippedResponse(t *testing.T) {
	// Initialize logger for testing
	config := logger.Config{
		Level:       logger.LevelDebug,
		Format:      "json",
		Output:      "stdout",
		ServiceName: "test",
		Environment: "test",
	}
	logger.Init(config)

	// Create test JSON response
	testResponse := map[string]interface{}{
		"message": "Hello, World!",
		"status":  "success",
	}
	responseJSON, _ := json.Marshal(testResponse)

	// Create a test handler that returns regular JSON
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(responseJSON)
	})

	// Wrap with correlation middleware
	middlewareHandler := RequestCorrelationMiddleware(testHandler)

	// Create test request
	req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"test": "data"}`))
	req.Header.Set("Content-Type", "application/json")

	// Create response recorder
	rr := httptest.NewRecorder()

	// Execute request
	middlewareHandler.ServeHTTP(rr, req)

	// Verify response
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", rr.Code)
	}

	// Verify response body is not gzipped
	responseBody := rr.Body.Bytes()
	var responseData map[string]interface{}
	if err := json.Unmarshal(responseBody, &responseData); err != nil {
		t.Errorf("Failed to parse JSON response: %v", err)
	}

	// Verify the content matches
	if responseData["message"] != "Hello, World!" {
		t.Errorf("Expected message 'Hello, World!', got %v", responseData["message"])
	}
}
