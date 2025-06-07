package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aashari/go-generative-api-router/internal/config"
	"github.com/aashari/go-generative-api-router/internal/logger"
	"github.com/aashari/go-generative-api-router/internal/proxy"
	"github.com/aashari/go-generative-api-router/internal/selector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain runs before all tests in this package
func TestMain(m *testing.M) {
	// Initialize logger for all tests
	if err := logger.Init(logger.DefaultConfig); err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	m.Run()
}

func TestNewAPIHandlers(t *testing.T) {
	creds := []config.Credential{
		{Platform: "openai", Type: "api-key", Value: "test"},
	}
	models := []config.VendorModel{
		{Vendor: "openai", Model: "gpt-4"},
	}
	client := proxy.NewAPIClient()
	selector := selector.NewRandomSelector()

	handlers := NewAPIHandlers(creds, models, client, selector)

	require.NotNil(t, handlers)
	assert.Equal(t, creds, handlers.Credentials)
	assert.Equal(t, models, handlers.VendorModels)
	assert.Equal(t, client, handlers.APIClient)
	assert.Equal(t, selector, handlers.ModelSelector)
}

func TestHealthHandler(t *testing.T) {
	// Create handlers with proper dependencies
	creds := []config.Credential{
		{Platform: "openai", Type: "api-key", Value: "test"},
	}
	models := []config.VendorModel{
		{Vendor: "openai", Model: "gpt-4"},
	}
	client := proxy.NewAPIClient()
	sel := selector.NewRandomSelector()

	handlers := NewAPIHandlers(creds, models, client, sel)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handlers.HealthHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	// Parse the JSON response
	var healthResponse HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &healthResponse)
	require.NoError(t, err)

	// Verify the structure
	assert.Equal(t, "healthy", healthResponse.Status)
	assert.NotEmpty(t, healthResponse.Timestamp)
	assert.NotNil(t, healthResponse.Services)
	assert.NotNil(t, healthResponse.Details)

	// Verify services are up
	assert.Equal(t, "up", healthResponse.Services["api"])
	assert.Equal(t, "up", healthResponse.Services["credentials"])
	assert.Equal(t, "up", healthResponse.Services["models"])
	assert.Equal(t, "up", healthResponse.Services["selector"])

	// Verify details
	assert.Contains(t, healthResponse.Details, "version")
	assert.Contains(t, healthResponse.Details, "uptime")

	// Uptime should be a number (int64)
	uptime, ok := healthResponse.Details["uptime"].(float64) // JSON unmarshals numbers as float64
	assert.True(t, ok)
	assert.GreaterOrEqual(t, uptime, float64(0))
}

func TestHealthHandler_Degraded(t *testing.T) {
	// Create handlers with missing credentials (degraded state)
	var creds []config.Credential // Empty credentials
	models := []config.VendorModel{
		{Vendor: "openai", Model: "gpt-4"},
	}
	client := proxy.NewAPIClient()
	sel := selector.NewRandomSelector()

	handlers := NewAPIHandlers(creds, models, client, sel)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handlers.HealthHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code) // Still 200 for degraded
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	// Parse the JSON response
	var healthResponse HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &healthResponse)
	require.NoError(t, err)

	// Verify degraded status
	assert.Equal(t, "degraded", healthResponse.Status)
	assert.Equal(t, "down", healthResponse.Services["credentials"])
	assert.Equal(t, "up", healthResponse.Services["models"])
}

func TestHealthHandler_Unhealthy(t *testing.T) {
	// Create handlers with nil API client (unhealthy state)
	creds := []config.Credential{
		{Platform: "openai", Type: "api-key", Value: "test"},
	}
	models := []config.VendorModel{
		{Vendor: "openai", Model: "gpt-4"},
	}

	handlers := &APIHandlers{
		Credentials:   creds,
		VendorModels:  models,
		APIClient:     nil, // This will cause unhealthy status
		ModelSelector: selector.NewRandomSelector(),
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handlers.HealthHandler(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code) // 503 for unhealthy
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	// Parse the JSON response
	var healthResponse HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &healthResponse)
	require.NoError(t, err)

	// Verify unhealthy status
	assert.Equal(t, "unhealthy", healthResponse.Status)
	assert.Equal(t, "down", healthResponse.Services["api"])
}

func TestModelsHandler(t *testing.T) {
	models := []config.VendorModel{
		{Vendor: "openai", Model: "gpt-4"},
		{Vendor: "gemini", Model: "gemini-pro"},
	}

	handlers := &APIHandlers{
		VendorModels: models,
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	w := httptest.NewRecorder()

	handlers.ModelsHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	// Check that response contains JSON
	body := w.Body.String()
	assert.Contains(t, body, "gpt-4")
	assert.Contains(t, body, "gemini-pro")
	assert.Contains(t, body, "\"object\":\"list\"")
}

func TestModelsHandlerWithVendorFilter(t *testing.T) {
	models := []config.VendorModel{
		{Vendor: "openai", Model: "gpt-4"},
		{Vendor: "gemini", Model: "gemini-pro"},
	}

	handlers := &APIHandlers{
		VendorModels: models,
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/models?vendor=openai", nil)
	w := httptest.NewRecorder()

	handlers.ModelsHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Check that response contains only OpenAI model
	body := w.Body.String()
	assert.Contains(t, body, "gpt-4")
	assert.NotContains(t, body, "gemini-pro")
}

func TestChatCompletionsHandler_Success(t *testing.T) {
	creds := []config.Credential{
		{Platform: "openai", Type: "api-key", Value: "test"},
	}
	models := []config.VendorModel{
		{Vendor: "openai", Model: "gpt-4"},
	}

	// Create a mock API client that doesn't make real requests
	client := proxy.NewAPIClient()
	selector := selector.NewRandomSelector()

	handlers := &APIHandlers{
		Credentials:   creds,
		VendorModels:  models,
		APIClient:     client,
		ModelSelector: selector,
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	w := httptest.NewRecorder()

	// This will fail at the proxy level due to no request body, but it tests the handler path
	handlers.ChatCompletionsHandler(w, req)

	// Should get a bad request error from the proxy layer
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestChatCompletionsHandler_VendorFilter(t *testing.T) {
	creds := []config.Credential{
		{Platform: "openai", Type: "api-key", Value: "test-openai"},
		{Platform: "gemini", Type: "api-key", Value: "test-gemini"},
	}
	models := []config.VendorModel{
		{Vendor: "openai", Model: "gpt-4"},
		{Vendor: "gemini", Model: "gemini-pro"},
	}

	client := proxy.NewAPIClient()
	selector := selector.NewRandomSelector()

	handlers := &APIHandlers{
		Credentials:   creds,
		VendorModels:  models,
		APIClient:     client,
		ModelSelector: selector,
	}

	// Test with valid vendor filter
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions?vendor=openai", nil)
	w := httptest.NewRecorder()

	handlers.ChatCompletionsHandler(w, req)

	// Should still get bad request from proxy layer (no body), but vendor filtering works
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestChatCompletionsHandler_InvalidVendorNoCredentials(t *testing.T) {
	creds := []config.Credential{
		{Platform: "openai", Type: "api-key", Value: "test"},
	}
	models := []config.VendorModel{
		{Vendor: "openai", Model: "gpt-4"},
	}

	client := proxy.NewAPIClient()
	selector := selector.NewRandomSelector()

	handlers := &APIHandlers{
		Credentials:   creds,
		VendorModels:  models,
		APIClient:     client,
		ModelSelector: selector,
	}

	// Request non-existent vendor
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions?vendor=anthropic", nil)
	w := httptest.NewRecorder()

	handlers.ChatCompletionsHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "No credentials available for vendor: anthropic")
}

func TestChatCompletionsHandler_InvalidVendorNoModels(t *testing.T) {
	creds := []config.Credential{
		{Platform: "openai", Type: "api-key", Value: "test"},
		{Platform: "gemini", Type: "api-key", Value: "test"},
	}
	models := []config.VendorModel{
		{Vendor: "openai", Model: "gpt-4"},
		// No models for gemini
	}

	client := proxy.NewAPIClient()
	selector := selector.NewRandomSelector()

	handlers := &APIHandlers{
		Credentials:   creds,
		VendorModels:  models,
		APIClient:     client,
		ModelSelector: selector,
	}

	// Request vendor with credentials but no models
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions?vendor=gemini", nil)
	w := httptest.NewRecorder()

	handlers.ChatCompletionsHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "No models available for vendor: gemini")
}

func TestChatCompletionsHandler_EmptyVendorFilter(t *testing.T) {
	creds := []config.Credential{
		{Platform: "openai", Type: "api-key", Value: "test"},
	}
	models := []config.VendorModel{
		{Vendor: "openai", Model: "gpt-4"},
	}

	client := proxy.NewAPIClient()
	selector := selector.NewRandomSelector()

	handlers := &APIHandlers{
		Credentials:   creds,
		VendorModels:  models,
		APIClient:     client,
		ModelSelector: selector,
	}

	// Empty vendor parameter should be ignored
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions?vendor=", nil)
	w := httptest.NewRecorder()

	handlers.ChatCompletionsHandler(w, req)

	// Should proceed to proxy (and fail with bad request due to no body)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestModelsHandler_EmptyModels(t *testing.T) {
	handlers := &APIHandlers{
		VendorModels: []config.VendorModel{},
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	w := httptest.NewRecorder()

	handlers.ModelsHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	// Check that response contains empty data array
	body := w.Body.String()
	assert.Contains(t, body, "\"object\":\"list\"")
	// When no models exist, Go encodes nil slice as null
	assert.Contains(t, body, "\"data\":null")
}

func TestModelsHandler_AllFields(t *testing.T) {
	models := []config.VendorModel{
		{Vendor: "openai", Model: "gpt-4"},
	}

	handlers := &APIHandlers{
		VendorModels: models,
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	w := httptest.NewRecorder()

	handlers.ModelsHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Check all required fields in response
	body := w.Body.String()
	assert.Contains(t, body, "\"id\":\"gpt-4\"")
	assert.Contains(t, body, "\"object\":\"model\"")
	assert.Contains(t, body, "\"created\":")
	assert.Contains(t, body, "\"owned_by\":\"openai\"")
}
