package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aashari/go-generative-api-router/internal/config"
	"github.com/aashari/go-generative-api-router/internal/proxy"
	"github.com/aashari/go-generative-api-router/internal/selector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	handlers := &APIHandlers{}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handlers.HealthHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "OK", w.Body.String())
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
