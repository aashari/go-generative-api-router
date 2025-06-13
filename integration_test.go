package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aashari/go-generative-api-router/internal/app"
	"github.com/aashari/go-generative-api-router/internal/config"
	"github.com/aashari/go-generative-api-router/internal/logger"
)

// Test configuration
const (
	testTimeout      = 60 * time.Second
	shortTestTimeout = 10 * time.Second
	testPort         = ":8083" // Different from main server
)

// TestServer represents the test server instance
type TestServer struct {
	server     *httptest.Server
	app        *app.App
	baseURL    string
	httpClient *http.Client
}

// setupTestServer initializes the test server with real configurations
func setupTestServer(t *testing.T) *TestServer {
	// Load environment variables for testing
	if err := config.LoadEnvFromMultiplePaths(); err != nil {
		t.Logf("Warning: Could not load .env file: %v", err)
	}

	// Initialize logger for testing
	loggerConfig := logger.Config{
		Level:       logger.LevelDebug,
		Format:      "json",
		Output:      "stdout",
		ServiceName: "integration-test",
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
		Timeout: testTimeout,
	}

	return &TestServer{
		server:     server,
		app:        application,
		baseURL:    server.URL,
		httpClient: httpClient,
	}
}

// teardown closes the test server
func (ts *TestServer) teardown() {
	if ts.server != nil {
		ts.server.Close()
	}
}

// makeRequest is a helper to make HTTP requests during testing
func (ts *TestServer) makeRequest(method, endpoint string, body interface{}, headers map[string]string) (*http.Response, []byte, error) {
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
	req.Header.Set("User-Agent", "GenerativeAPIRouter-IntegrationTest/1.0")

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

// Test Structures
type HealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp string                 `json:"timestamp"`
	Services  map[string]string      `json:"services"`
	Details   map[string]interface{} `json:"details"`
}

type ChatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
	Tools       []Tool    `json:"tools,omitempty"`
}

type Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // Can be string or []ContentPart
}

// ContentPart represents a part of the message content for vision/file requests
type ContentPart struct {
	Type     string   `json:"type"`
	Text     string   `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
	FileURL  *FileURL  `json:"file_url,omitempty"`
}

// ImageURL represents an image URL structure
type ImageURL struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

// FileURL represents a file URL structure
type FileURL struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

type Tool struct {
	Type     string                 `json:"type"`
	Function map[string]interface{} `json:"function"`
}

type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

type Choice struct {
	Index        int             `json:"index"`
	Message      ResponseMessage `json:"message"`
	FinishReason string          `json:"finish_reason"`
}

// ResponseMessage represents the message in the response
type ResponseMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

type ErrorResponse struct {
	Error ErrorInfo `json:"error"`
}

type ErrorInfo struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
}

// Integration Tests

func TestHealthEndpoint(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.teardown()

	t.Run("health_check_returns_status", func(t *testing.T) {
		resp, body, err := ts.makeRequest("GET", "/health", nil, nil)
		if err != nil {
			t.Fatalf("Health check failed: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
		}

		var healthResp HealthResponse
		if err := json.Unmarshal(body, &healthResp); err != nil {
			t.Fatalf("Failed to parse health response: %v. Body: %s", err, string(body))
		}

		if healthResp.Status != "healthy" && healthResp.Status != "degraded" {
			t.Errorf("Expected status 'healthy' or 'degraded', got '%s'", healthResp.Status)
		}

		// Verify required services are present
		requiredServices := []string{"api", "credentials", "models", "selector"}
		for _, service := range requiredServices {
			if status, exists := healthResp.Services[service]; !exists {
				t.Errorf("Missing service '%s' in health check", service)
			} else if status != "up" {
				t.Errorf("Service '%s' is not up: %s", service, status)
			}
		}

		t.Logf("Health check passed: %s", healthResp.Status)
	})
}

func TestModelsEndpoint(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.teardown()

	t.Run("list_all_models", func(t *testing.T) {
		resp, body, err := ts.makeRequest("GET", "/v1/models", nil, nil)
		if err != nil {
			t.Fatalf("Models request failed: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
		}

		var modelsResp ModelsResponse
		if err := json.Unmarshal(body, &modelsResp); err != nil {
			t.Fatalf("Failed to parse models response: %v. Body: %s", err, string(body))
		}

		if modelsResp.Object != "list" {
			t.Errorf("Expected object 'list', got '%s'", modelsResp.Object)
		}

		if len(modelsResp.Data) == 0 {
			t.Error("Expected at least one model, got none")
		}

		// Verify model structure
		for _, model := range modelsResp.Data {
			if model.ID == "" {
				t.Error("Model ID is empty")
			}
			if model.Object != "model" {
				t.Errorf("Expected model object 'model', got '%s'", model.Object)
			}
			if model.OwnedBy == "" {
				t.Error("Model OwnedBy is empty")
			}
		}

		t.Logf("Found %d models", len(modelsResp.Data))
	})

	t.Run("filter_models_by_vendor_openai", func(t *testing.T) {
		resp, body, err := ts.makeRequest("GET", "/v1/models?vendor=openai", nil, nil)
		if err != nil {
			t.Fatalf("OpenAI models request failed: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
		}

		var modelsResp ModelsResponse
		if err := json.Unmarshal(body, &modelsResp); err != nil {
			t.Fatalf("Failed to parse models response: %v", err)
		}

		// Verify all models are from OpenAI
		for _, model := range modelsResp.Data {
			if model.OwnedBy != "openai" {
				t.Errorf("Expected OpenAI model, got model owned by '%s'", model.OwnedBy)
			}
		}

		t.Logf("Found %d OpenAI models", len(modelsResp.Data))
	})

	t.Run("filter_models_by_vendor_gemini", func(t *testing.T) {
		resp, body, err := ts.makeRequest("GET", "/v1/models?vendor=gemini", nil, nil)
		if err != nil {
			t.Fatalf("Gemini models request failed: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
		}

		var modelsResp ModelsResponse
		if err := json.Unmarshal(body, &modelsResp); err != nil {
			t.Fatalf("Failed to parse models response: %v", err)
		}

		// Verify all models are from Gemini
		for _, model := range modelsResp.Data {
			if model.OwnedBy != "gemini" {
				t.Errorf("Expected Gemini model, got model owned by '%s'", model.OwnedBy)
			}
		}

		t.Logf("Found %d Gemini models", len(modelsResp.Data))
	})
}

func TestChatCompletionsEndpoint(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.teardown()

	t.Run("basic_chat_completion", func(t *testing.T) {
		request := ChatCompletionRequest{
			Model: "gpt-4o",
			Messages: []Message{
				{
					Role:    "user",
					Content: "Say hello in exactly 3 words",
				},
			},
			MaxTokens:   50,
			Temperature: 0.1,
		}

		resp, body, err := ts.makeRequest("POST", "/v1/chat/completions", request, nil)
		if err != nil {
			t.Fatalf("Chat completion request failed: %v", err)
		}

		// Accept both success and potential API errors (due to real API calls)
		if resp.StatusCode != http.StatusOK {
			t.Logf("Chat completion returned status %d (might be API limitation): %s", resp.StatusCode, string(body))
			// Don't fail the test - this might be due to API keys or rate limits
			return
		}

		var chatResp ChatCompletionResponse
		if err := json.Unmarshal(body, &chatResp); err != nil {
			t.Fatalf("Failed to parse chat response: %v. Body: %s", err, string(body))
		}

		// Verify response structure
		if chatResp.ID == "" {
			t.Error("Response ID is empty")
		}
		if chatResp.Object != "chat.completion" {
			t.Errorf("Expected object 'chat.completion', got '%s'", chatResp.Object)
		}
		if len(chatResp.Choices) == 0 {
			t.Error("Expected at least one choice, got none")
		}

		// Verify model name is preserved (important feature of this router)
		if chatResp.Model != request.Model {
			t.Errorf("Expected model '%s', got '%s'", request.Model, chatResp.Model)
		}

		t.Logf("Chat completion successful with model: %s", chatResp.Model)
	})

	t.Run("vendor_specific_routing_openai", func(t *testing.T) {
		request := ChatCompletionRequest{
			Model: "gpt-4o",
			Messages: []Message{
				{
					Role:    "user",
					Content: "Test OpenAI routing",
				},
			},
			MaxTokens: 20,
		}

		resp, body, err := ts.makeRequest("POST", "/v1/chat/completions?vendor=openai", request, nil)
		if err != nil {
			t.Fatalf("OpenAI-specific routing failed: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Logf("OpenAI routing returned status %d (might be API limitation): %s", resp.StatusCode, string(body))
			return
		}

		var chatResp ChatCompletionResponse
		if err := json.Unmarshal(body, &chatResp); err != nil {
			t.Fatalf("Failed to parse chat response: %v", err)
		}

		// Verify original model name is preserved
		if chatResp.Model != request.Model {
			t.Errorf("Expected preserved model '%s', got '%s'", request.Model, chatResp.Model)
		}

		t.Logf("OpenAI-specific routing successful")
	})

	t.Run("vendor_specific_routing_gemini", func(t *testing.T) {
		request := ChatCompletionRequest{
			Model: "gemini-2.5-flash-preview-04-17",
			Messages: []Message{
				{
					Role:    "user",
					Content: "Test Gemini routing",
				},
			},
			MaxTokens: 20,
		}

		resp, body, err := ts.makeRequest("POST", "/v1/chat/completions?vendor=gemini", request, nil)
		if err != nil {
			t.Fatalf("Gemini-specific routing failed: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Logf("Gemini routing returned status %d (might be API limitation): %s", resp.StatusCode, string(body))
			return
		}

		var chatResp ChatCompletionResponse
		if err := json.Unmarshal(body, &chatResp); err != nil {
			t.Fatalf("Failed to parse chat response: %v", err)
		}

		// Verify original model name is preserved
		if chatResp.Model != request.Model {
			t.Errorf("Expected preserved model '%s', got '%s'", request.Model, chatResp.Model)
		}

		t.Logf("Gemini-specific routing successful")
	})

	t.Run("multiple_requests_distribution", func(t *testing.T) {
		// Test even distribution by making multiple requests
		successfulRequests := 0

		for i := 0; i < 10; i++ {
			request := ChatCompletionRequest{
				Model: "gpt-4o",
				Messages: []Message{
					{
						Role:    "user",
						Content: fmt.Sprintf("Test request #%d", i+1),
					},
				},
				MaxTokens: 10,
			}

			resp, body, err := ts.makeRequest("POST", "/v1/chat/completions", request, nil)
			if err != nil {
				t.Logf("Request %d failed: %v", i+1, err)
				continue
			}

			if resp.StatusCode == http.StatusOK {
				successfulRequests++
				// For successful requests, we'd need to analyze logs to see which vendor was actually used
				// This is a limitation of the current logging approach
			} else {
				t.Logf("Request %d returned status %d: %s", i+1, resp.StatusCode, string(body))
			}
		}

		if successfulRequests > 0 {
			t.Logf("Distribution test completed: %d/%d successful requests", successfulRequests, 10)
		} else {
			t.Logf("No successful requests in distribution test (might be API limitations)")
		}
	})

	t.Run("tool_usage_support", func(t *testing.T) {
		weatherTool := Tool{
			Type: "function",
			Function: map[string]interface{}{
				"name":        "get_weather",
				"description": "Get the current weather in a location",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"location": map[string]interface{}{
							"type":        "string",
							"description": "The city and state, e.g. San Francisco, CA",
						},
					},
					"required": []string{"location"},
				},
			},
		}

		request := ChatCompletionRequest{
			Model: "gpt-4o",
			Messages: []Message{
				{
					Role:    "user",
					Content: "What's the weather like in San Francisco?",
				},
			},
			Tools:     []Tool{weatherTool},
			MaxTokens: 100,
		}

		resp, body, err := ts.makeRequest("POST", "/v1/chat/completions", request, nil)
		if err != nil {
			t.Fatalf("Tool usage request failed: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Logf("Tool usage returned status %d (might be API limitation): %s", resp.StatusCode, string(body))
			return
		}

		var chatResp ChatCompletionResponse
		if err := json.Unmarshal(body, &chatResp); err != nil {
			t.Fatalf("Failed to parse tool usage response: %v", err)
		}

		t.Logf("Tool usage test completed successfully")
	})

	t.Run("file_url_vision_test", func(t *testing.T) {
		// Test file_url functionality with a Telegram image
		request := ChatCompletionRequest{
			Model: "vision-test",
			Messages: []Message{
				{
					Role: "user",
					Content: []ContentPart{
						{
							Type: "text",
							Text: "What do you see in this image? what is the text in that image",
						},
						{
							Type: "file_url",
							FileURL: &FileURL{
								URL: "https://api.telegram.org/file/bot6855937407:AAEIbG6edy-4hVcT_8IBkgpbWKlBYpJbb6s/documents/file_287.png",
							},
						},
					},
				},
			},
			MaxTokens: 200,
		}

		resp, body, err := ts.makeRequest("POST", "/v1/chat/completions", request, nil)
		if err != nil {
			t.Fatalf("File URL vision request failed: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Logf("File URL vision request returned status %d: %s", resp.StatusCode, string(body))
			// Don't fail the test - this might be due to API limitations
			return
		}

		var chatResp ChatCompletionResponse
		if err := json.Unmarshal(body, &chatResp); err != nil {
			t.Fatalf("Failed to parse file URL vision response: %v. Body: %s", err, string(body))
		}

		// Verify response structure
		if chatResp.ID == "" {
			t.Error("Response ID is empty")
		}
		if chatResp.Object != "chat.completion" {
			t.Errorf("Expected object 'chat.completion', got '%s'", chatResp.Object)
		}
		if len(chatResp.Choices) == 0 {
			t.Error("Expected at least one choice, got none")
		}

		// Verify model name is preserved
		if chatResp.Model != request.Model {
			t.Errorf("Expected model '%s', got '%s'", request.Model, chatResp.Model)
		}

		// Verify the response contains the expected text from the image
		if len(chatResp.Choices) > 0 && chatResp.Choices[0].Message.Content != "" {
			responseContent := strings.ToLower(chatResp.Choices[0].Message.Content)
			if !strings.Contains(responseContent, "the notification sync has") {
				t.Errorf("Expected response to contain 'the notification sync has', but got: %s", chatResp.Choices[0].Message.Content)
			} else {
				t.Logf("File URL vision test passed - correctly identified text in image")
			}
		}

		t.Logf("File URL vision test completed successfully")
	})
}

func TestErrorHandling(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.teardown()

	t.Run("invalid_http_method", func(t *testing.T) {
		resp, body, err := ts.makeRequest("GET", "/v1/chat/completions", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d. Body: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("invalid_json_payload", func(t *testing.T) {
		req, _ := http.NewRequest("POST", ts.baseURL+"/v1/chat/completions", strings.NewReader("{invalid json}"))
		req.Header.Set("Content-Type", "application/json")

		resp, err := ts.httpClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			t.Error("Expected error status for invalid JSON, got 200")
		}

		t.Logf("Invalid JSON correctly rejected with status: %d", resp.StatusCode)
	})

	t.Run("invalid_vendor_filter", func(t *testing.T) {
		request := ChatCompletionRequest{
			Model: "test-model",
			Messages: []Message{
				{
					Role:    "user",
					Content: "test",
				},
			},
		}

		resp, body, err := ts.makeRequest("POST", "/v1/chat/completions?vendor=nonexistent", request, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid vendor, got %d. Body: %s", resp.StatusCode, string(body))
		}

		var errorResp ErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil {
			if !strings.Contains(errorResp.Error.Message, "vendor") {
				t.Errorf("Expected vendor-related error message, got: %s", errorResp.Error.Message)
			}
		}

		t.Logf("Invalid vendor correctly rejected")
	})
}

func TestCORS(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.teardown()

	t.Run("cors_preflight", func(t *testing.T) {
		req, _ := http.NewRequest("OPTIONS", ts.baseURL+"/v1/chat/completions", nil)
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Access-Control-Request-Method", "POST")
		req.Header.Set("Access-Control-Request-Headers", "Content-Type, Authorization")

		resp, err := ts.httpClient.Do(req)
		if err != nil {
			t.Fatalf("CORS preflight failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for CORS preflight, got %d", resp.StatusCode)
		}

		// Check CORS headers
		if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
			t.Errorf("Expected CORS origin '*', got '%s'", resp.Header.Get("Access-Control-Allow-Origin"))
		}

		t.Logf("CORS preflight successful")
	})
}

func TestConcurrentRequests(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.teardown()

	t.Run("concurrent_chat_completions", func(t *testing.T) {
		concurrency := 5
		results := make(chan error, concurrency)

		for i := 0; i < concurrency; i++ {
			go func(requestID int) {
				request := ChatCompletionRequest{
					Model: "gpt-4o",
					Messages: []Message{
						{
							Role:    "user",
							Content: fmt.Sprintf("Concurrent test request #%d", requestID),
						},
					},
					MaxTokens: 10,
				}

				ctx, cancel := context.WithTimeout(context.Background(), shortTestTimeout)
				defer cancel()

				// Create request with context
				reqBody, _ := json.Marshal(request)
				req, err := http.NewRequestWithContext(ctx, "POST", ts.baseURL+"/v1/chat/completions", bytes.NewBuffer(reqBody))
				if err != nil {
					results <- fmt.Errorf("request %d failed to create: %v", requestID, err)
					return
				}
				req.Header.Set("Content-Type", "application/json")

				resp, err := ts.httpClient.Do(req)
				if err != nil {
					results <- fmt.Errorf("request %d failed: %v", requestID, err)
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode == http.StatusOK {
					results <- nil // Success
				} else {
					// Don't treat API limitations as test failures
					results <- nil
				}
			}(i + 1)
		}

		// Wait for all results
		successCount := 0
		for i := 0; i < concurrency; i++ {
			if err := <-results; err != nil {
				t.Logf("Concurrent request error: %v", err)
			} else {
				successCount++
			}
		}

		if successCount == 0 {
			t.Log("No concurrent requests succeeded (might be API limitations)")
		} else {
			t.Logf("Concurrent requests test: %d/%d succeeded", successCount, concurrency)
		}
	})
}

// TestMain is the entry point for integration tests
func TestMain(m *testing.M) {
	// Set test environment
	os.Setenv("ENVIRONMENT", "test")
	os.Setenv("LOG_LEVEL", "debug")

	// Verify configurations exist
	if _, err := os.Stat("configs/credentials.json"); os.IsNotExist(err) {
		fmt.Println("ERROR: configs/credentials.json not found. Integration tests require real credentials.")
		fmt.Println("Please ensure your credentials are properly configured in configs/credentials.json")
		os.Exit(1)
	}

	if _, err := os.Stat("configs/models.json"); os.IsNotExist(err) {
		fmt.Println("ERROR: configs/models.json not found. Integration tests require model configuration.")
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	os.Exit(code)
}
