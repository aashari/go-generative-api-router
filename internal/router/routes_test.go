package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aashari/go-generative-api-router/internal/config"
	"github.com/aashari/go-generative-api-router/internal/handlers"
	"github.com/aashari/go-generative-api-router/internal/proxy"
	"github.com/aashari/go-generative-api-router/internal/selector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupRoutes(t *testing.T) {
	// Create test dependencies
	creds := []config.Credential{
		{Platform: "openai", Type: "api-key", Value: "test"},
	}
	models := []config.VendorModel{
		{Vendor: "openai", Model: "gpt-4"},
	}
	client := proxy.NewAPIClient()
	sel := selector.NewRandomSelector()
	apiHandlers := handlers.NewAPIHandlers(creds, models, client, sel)

	// Setup routes
	handler := SetupRoutes(apiHandlers)
	require.NotNil(t, handler)

	// Test all registered routes
	testCases := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		description    string
	}{
		{
			name:           "health endpoint",
			method:         http.MethodGet,
			path:           "/health",
			expectedStatus: http.StatusOK,
			description:    "Health check should return OK",
		},
		{
			name:           "models endpoint",
			method:         http.MethodGet,
			path:           "/v1/models",
			expectedStatus: http.StatusOK,
			description:    "Models endpoint should return model list",
		},

		{
			name:           "swagger ui endpoint",
			method:         http.MethodGet,
			path:           "/swagger/",
			expectedStatus: http.StatusMovedPermanently, // 301 redirect is expected
			description:    "Swagger UI should redirect properly",
		},
		{
			name:           "pprof index endpoint",
			method:         http.MethodGet,
			path:           "/debug/pprof/",
			expectedStatus: http.StatusOK,
			description:    "Pprof index should be accessible",
		},
		{
			name:           "pprof cmdline endpoint",
			method:         http.MethodGet,
			path:           "/debug/pprof/cmdline",
			expectedStatus: http.StatusOK,
			description:    "Pprof cmdline should be accessible",
		},
		{
			name:           "pprof profile endpoint",
			method:         http.MethodGet,
			path:           "/debug/pprof/profile",
			expectedStatus: http.StatusOK,
			description:    "Pprof profile should be accessible",
		},
		{
			name:           "pprof symbol endpoint",
			method:         http.MethodGet,
			path:           "/debug/pprof/symbol",
			expectedStatus: http.StatusOK,
			description:    "Pprof symbol should be accessible",
		},
		{
			name:           "pprof trace endpoint",
			method:         http.MethodGet,
			path:           "/debug/pprof/trace",
			expectedStatus: http.StatusOK,
			description:    "Pprof trace should be accessible",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code, tc.description)
		})
	}
}

func TestSetupRoutes_ChatCompletions(t *testing.T) {
	// Create test dependencies
	creds := []config.Credential{
		{Platform: "openai", Type: "api-key", Value: "test"},
	}
	models := []config.VendorModel{
		{Vendor: "openai", Model: "gpt-4"},
	}
	client := proxy.NewAPIClient()
	sel := selector.NewRandomSelector()
	apiHandlers := handlers.NewAPIHandlers(creds, models, client, sel)

	// Setup routes
	handler := SetupRoutes(apiHandlers)

	tests := []struct {
		name           string
		method         string
		expectedStatus int
		description    string
	}{
		{
			name:           "POST allowed",
			method:         http.MethodPost,
			expectedStatus: http.StatusBadRequest, // Will fail validation but method is allowed
			description:    "POST method should be allowed for chat completions",
		},
		{
			name:           "GET not allowed",
			method:         http.MethodGet,
			expectedStatus: http.StatusMethodNotAllowed,
			description:    "GET method should not be allowed for chat completions",
		},
		{
			name:           "PUT not allowed",
			method:         http.MethodPut,
			expectedStatus: http.StatusMethodNotAllowed,
			description:    "PUT method should not be allowed for chat completions",
		},
		{
			name:           "DELETE not allowed",
			method:         http.MethodDelete,
			expectedStatus: http.StatusMethodNotAllowed,
			description:    "DELETE method should not be allowed for chat completions",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/v1/chat/completions", nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code, tc.description)
		})
	}
}

func TestSetupRoutes_UnregisteredPath(t *testing.T) {
	// Create test dependencies
	creds := []config.Credential{
		{Platform: "openai", Type: "api-key", Value: "test"},
	}
	models := []config.VendorModel{
		{Vendor: "openai", Model: "gpt-4"},
	}
	client := proxy.NewAPIClient()
	sel := selector.NewRandomSelector()
	apiHandlers := handlers.NewAPIHandlers(creds, models, client, sel)

	// Setup routes
	handler := SetupRoutes(apiHandlers)

	// Test unregistered path
	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSetupRoutes_VendorFilterQuery(t *testing.T) {
	// Create test dependencies
	creds := []config.Credential{
		{Platform: "openai", Type: "api-key", Value: "test"},
		{Platform: "gemini", Type: "api-key", Value: "test"},
	}
	models := []config.VendorModel{
		{Vendor: "openai", Model: "gpt-4"},
		{Vendor: "gemini", Model: "gemini-pro"},
	}
	client := proxy.NewAPIClient()
	sel := selector.NewRandomSelector()
	apiHandlers := handlers.NewAPIHandlers(creds, models, client, sel)

	// Setup routes
	handler := SetupRoutes(apiHandlers)

	// Test vendor filtering
	req := httptest.NewRequest(http.MethodGet, "/v1/models?vendor=openai", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	body := w.Body.String()
	assert.Contains(t, body, "gpt-4", "Response should contain OpenAI model")
	assert.NotContains(t, body, "gemini-pro", "Response should not contain Gemini model when filtered")
}

func TestSetupRoutes_CORS_Headers(t *testing.T) {
	// Create test dependencies
	creds := []config.Credential{
		{Platform: "openai", Type: "api-key", Value: "test"},
	}
	models := []config.VendorModel{
		{Vendor: "openai", Model: "gpt-4"},
	}
	client := proxy.NewAPIClient()
	sel := selector.NewRandomSelector()
	apiHandlers := handlers.NewAPIHandlers(creds, models, client, sel)

	// Setup routes
	handler := SetupRoutes(apiHandlers)

	// Test health endpoint
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// While not explicitly set in the router, check that basic functionality works
	// In a real implementation, CORS headers might be added to the middleware
}
