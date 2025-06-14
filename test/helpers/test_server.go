package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aashari/go-generative-api-router/internal/app"
	"github.com/aashari/go-generative-api-router/internal/config"
	"github.com/aashari/go-generative-api-router/internal/logger"
)

// ensureProjectRoot changes to the project root directory
func ensureProjectRoot() error {
	// Look for go.mod file to identify project root
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %v", err)
	}

	// Walk up the directory tree looking for go.mod
	for {
		if _, err := os.Stat(filepath.Join(currentDir, "go.mod")); err == nil {
			// Found go.mod, change to this directory
			if err := os.Chdir(currentDir); err != nil {
				return fmt.Errorf("failed to change to project root: %v", err)
			}
			return nil
		}

		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			// Reached filesystem root without finding go.mod
			return fmt.Errorf("could not find project root (go.mod not found)")
		}
		currentDir = parent
	}
}

// TestServer represents a test server instance with common utilities
type TestServer struct {
	server     *httptest.Server
	app        *app.App
	baseURL    string
	httpClient *http.Client
	t          *testing.T
}

// TestConfig holds configuration for test server setup
type TestConfig struct {
	Timeout     time.Duration
	LogLevel    string
	ServiceName string
}

// DefaultTestConfig returns default configuration for tests
func DefaultTestConfig() TestConfig {
	return TestConfig{
		Timeout:     60 * time.Second,
		LogLevel:    "info",
		ServiceName: "test-server",
	}
}

// NewTestServer creates a new test server with the given configuration
func NewTestServer(t *testing.T, testConfig TestConfig) *TestServer {
	// Load environment variables for testing
	if err := config.LoadEnvFromMultiplePaths(); err != nil {
		t.Logf("Warning: Could not load .env file: %v", err)
	}

	// Ensure we're in the project root directory for config files
	if err := ensureProjectRoot(); err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}

	// Initialize logger for testing
	loggerConfig := logger.Config{
		Level:       logger.LevelInfo, // Use constant for now
		Format:      "json",
		Output:      "stdout",
		ServiceName: testConfig.ServiceName,
		Environment: "test",
	}
	if err := logger.Init(loggerConfig); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	// Create the application instance
	application, err := app.NewApp()
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	// Setup routes
	handler := application.SetupRoutes()

	// Create test server
	server := httptest.NewServer(handler)

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: testConfig.Timeout,
	}

	return &TestServer{
		server:     server,
		app:        application,
		baseURL:    server.URL,
		httpClient: httpClient,
		t:          t,
	}
}

// Close shuts down the test server
func (ts *TestServer) Close() {
	if ts.server != nil {
		ts.server.Close()
	}
}

// URL returns the base URL of the test server
func (ts *TestServer) URL() string {
	return ts.baseURL
}

// App returns the application instance
func (ts *TestServer) App() *app.App {
	return ts.app
}

// MakeRequest makes an HTTP request to the test server
func (ts *TestServer) MakeRequest(method, endpoint string, body interface{}, headers map[string]string) (*http.Response, []byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, ts.baseURL+endpoint, reqBody)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set default headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "GenerativeAPIRouter-Test/1.0")

	// Set custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := ts.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, fmt.Errorf("failed to read response body: %v", err)
	}

	return resp, respBody, nil
}

// MakeRequestWithTimeout makes an HTTP request with a custom timeout
func (ts *TestServer) MakeRequestWithTimeout(method, endpoint string, body interface{}, headers map[string]string, timeout time.Duration) (*http.Response, []byte, error) {
	// Create a new client with custom timeout
	client := &http.Client{Timeout: timeout}

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, ts.baseURL+endpoint, reqBody)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set default headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "GenerativeAPIRouter-Test/1.0")

	// Set custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, fmt.Errorf("failed to read response body: %v", err)
	}

	return resp, respBody, nil
}

// AssertStatusCode asserts that the response has the expected status code
func (ts *TestServer) AssertStatusCode(resp *http.Response, expected int) {
	if resp.StatusCode != expected {
		ts.t.Errorf("Expected status code %d, got %d", expected, resp.StatusCode)
	}
}

// AssertJSONResponse asserts that the response body is valid JSON and unmarshals it
func (ts *TestServer) AssertJSONResponse(body []byte, target interface{}) {
	if err := json.Unmarshal(body, target); err != nil {
		ts.t.Fatalf("Failed to parse JSON response: %v\nBody: %s", err, string(body))
	}
}

// LogResponse logs the response for debugging
func (ts *TestServer) LogResponse(resp *http.Response, body []byte) {
	ts.t.Logf("Response Status: %d", resp.StatusCode)
	ts.t.Logf("Response Headers: %v", resp.Header)
	ts.t.Logf("Response Body: %s", string(body))
}
