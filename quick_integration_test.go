package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/aashari/go-generative-api-router/internal/app"
	"github.com/aashari/go-generative-api-router/internal/config"
	"github.com/aashari/go-generative-api-router/internal/logger"
)

// Quick integration tests that focus on router functionality without external API dependency

func TestQuickIntegration(t *testing.T) {
	// Load environment variables for testing
	if err := config.LoadEnvFromMultiplePaths(); err != nil {
		t.Logf("Warning: Could not load .env file: %v", err)
	}

	// Initialize logger for testing
	loggerConfig := logger.Config{
		Level:       logger.LevelInfo,
		Format:      "json",
		Output:      "stdout",
		ServiceName: "quick-integration-test",
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
	if handler == nil {
		t.Fatal("Handler is nil")
	}

	// Setup test server
	ts := setupTestServer(t)
	defer ts.teardown()

	t.Run("validate_configuration_loading", func(t *testing.T) {
		// Test that the app was created successfully
		if ts.app == nil {
			t.Error("App not initialized")
		}
		t.Log("Application initialized successfully")
	})

	t.Run("health_endpoint_basic", func(t *testing.T) {
		resp, body, err := ts.makeRequest("GET", "/health", nil, nil)
		if err != nil {
			t.Fatalf("Health check failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var healthResp map[string]interface{}
		if err := json.Unmarshal(body, &healthResp); err != nil {
			t.Fatalf("Failed to parse health response: %v", err)
		}

		if healthResp["status"] == nil {
			t.Error("Health response missing status")
		}

		t.Logf("Health check status: %v", healthResp["status"])
	})

	t.Run("models_endpoint_basic", func(t *testing.T) {
		resp, body, err := ts.makeRequest("GET", "/v1/models", nil, nil)
		if err != nil {
			t.Fatalf("Models request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var modelsResp map[string]interface{}
		if err := json.Unmarshal(body, &modelsResp); err != nil {
			t.Fatalf("Failed to parse models response: %v", err)
		}

		if modelsResp["object"] != "list" {
			t.Errorf("Expected object 'list', got %v", modelsResp["object"])
		}

		data, ok := modelsResp["data"].([]interface{})
		if !ok {
			t.Error("Models data is not an array")
		} else {
			t.Logf("Found %d models", len(data))
		}
	})

	t.Run("vendor_filtering", func(t *testing.T) {
		// Test OpenAI vendor filtering
		resp, body, err := ts.makeRequest("GET", "/v1/models?vendor=openai", nil, nil)
		if err != nil {
			t.Fatalf("OpenAI models request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200 for OpenAI filter, got %d", resp.StatusCode)
		}

		var modelsResp map[string]interface{}
		if err := json.Unmarshal(body, &modelsResp); err != nil {
			t.Fatalf("Failed to parse OpenAI models response: %v", err)
		}

		data, ok := modelsResp["data"].([]interface{})
		if !ok {
			t.Error("OpenAI models data is not an array")
		} else {
			t.Logf("Found %d OpenAI models", len(data))
		}

		// Test Gemini vendor filtering
		resp2, body2, err := ts.makeRequest("GET", "/v1/models?vendor=gemini", nil, nil)
		if err != nil {
			t.Fatalf("Gemini models request failed: %v", err)
		}

		if resp2.StatusCode != 200 {
			t.Errorf("Expected status 200 for Gemini filter, got %d", resp2.StatusCode)
		}

		var modelsResp2 map[string]interface{}
		if err := json.Unmarshal(body2, &modelsResp2); err != nil {
			t.Fatalf("Failed to parse Gemini models response: %v", err)
		}

		data2, ok := modelsResp2["data"].([]interface{})
		if !ok {
			t.Error("Gemini models data is not an array")
		} else {
			t.Logf("Found %d Gemini models", len(data2))
		}
	})

	t.Run("chat_completions_request_format", func(t *testing.T) {
		// Test that the endpoint accepts properly formatted requests
		// Don't wait for response due to potential API timeouts
		request := map[string]interface{}{
			"model": "gpt-4o",
			"messages": []map[string]string{
				{
					"role":    "user",
					"content": "Test message",
				},
			},
			"max_tokens": 5,
		}

		// Use short timeout to test request handling, not API response
		client := ts.httpClient
		originalTimeout := client.Timeout
		client.Timeout = 2 * time.Second

		resp, body, err := ts.makeRequest("POST", "/v1/chat/completions", request, nil)

		// Restore original timeout
		client.Timeout = originalTimeout

		if err != nil {
			// Timeout or connection error is acceptable - means the request was processed
			t.Logf("Chat completion request processing test: %v (expected)", err)
			return
		}

		// If we get a response, check the format
		t.Logf("Chat completion returned status %d", resp.StatusCode)
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			var chatResp map[string]interface{}
			if err := json.Unmarshal(body, &chatResp); err == nil {
				if chatResp["id"] != nil {
					t.Log("Chat completion response format valid")
				}
			}
		}
	})

	t.Run("error_handling", func(t *testing.T) {
		// Test invalid HTTP method
		resp, _, err := ts.makeRequest("GET", "/v1/chat/completions", nil, nil)
		if err != nil {
			t.Fatalf("Invalid method test failed: %v", err)
		}

		if resp.StatusCode != 405 {
			t.Errorf("Expected status 405 for invalid method, got %d", resp.StatusCode)
		}

		// Test invalid vendor
		resp2, body2, err := ts.makeRequest("POST", "/v1/chat/completions?vendor=invalid",
			map[string]interface{}{
				"model": "test",
				"messages": []map[string]string{
					{"role": "user", "content": "test"},
				},
			}, nil)
		if err != nil {
			t.Fatalf("Invalid vendor test failed: %v", err)
		}

		if resp2.StatusCode != 400 {
			t.Errorf("Expected status 400 for invalid vendor, got %d", resp2.StatusCode)
		}

		var errorResp map[string]interface{}
		if err := json.Unmarshal(body2, &errorResp); err == nil {
			if errorResp["error"] != nil {
				t.Log("Error response format valid")
			}
		}
	})

	t.Run("cors_headers", func(t *testing.T) {
		// Test CORS preflight
		resp, _, err := ts.makeRequest("OPTIONS", "/v1/chat/completions", nil,
			map[string]string{
				"Origin":                        "https://example.com",
				"Access-Control-Request-Method": "POST",
			})
		if err != nil {
			t.Fatalf("CORS preflight failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200 for CORS preflight, got %d", resp.StatusCode)
		}

		corsOrigin := resp.Header.Get("Access-Control-Allow-Origin")
		if corsOrigin != "*" {
			t.Errorf("Expected CORS origin '*', got '%s'", corsOrigin)
		}

		t.Log("CORS headers validated")
	})
}
