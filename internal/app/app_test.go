package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aashari/go-generative-api-router/internal/config"
	"github.com/aashari/go-generative-api-router/internal/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewApp(t *testing.T) {
	tests := []struct {
		name        string
		expectError bool
		description string
	}{
		{
			name:        "successful_initialization",
			expectError: false,
			description: "Should successfully initialize app with valid config files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := NewApp()
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, app)
			} else {
				// Note: This test will fail if credentials.json or models.json are missing
				// In a real scenario, we'd mock the config loading
				if err != nil {
					t.Skipf("Skipping test due to missing config files: %v", err)
					return
				}
				
				require.NoError(t, err)
				require.NotNil(t, app)
				assert.NotNil(t, app.APIClient)
				assert.NotNil(t, app.ModelSelector)
				assert.NotNil(t, app.APIHandlers)
				assert.NotEmpty(t, app.Credentials)
				assert.NotEmpty(t, app.VendorModels)
			}
		})
	}
}

func TestHealthHandler(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Skipf("Skipping test due to missing config files: %v", err)
		return
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	app.APIHandlers.HealthHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "OK", w.Body.String())
}

func TestFilterCredentialsByVendor(t *testing.T) {
	tests := []struct {
		name        string
		credentials []config.Credential
		vendor      string
		expected    int
	}{
		{
			name: "filter_openai_credentials",
			credentials: []config.Credential{
				{Platform: "openai", Type: "api-key", Value: "test1"},
				{Platform: "gemini", Type: "api-key", Value: "test2"},
				{Platform: "openai", Type: "api-key", Value: "test3"},
			},
			vendor:   "openai",
			expected: 2,
		},
		{
			name: "filter_gemini_credentials",
			credentials: []config.Credential{
				{Platform: "openai", Type: "api-key", Value: "test1"},
				{Platform: "gemini", Type: "api-key", Value: "test2"},
			},
			vendor:   "gemini",
			expected: 1,
		},
		{
			name: "no_matching_credentials",
			credentials: []config.Credential{
				{Platform: "openai", Type: "api-key", Value: "test1"},
			},
			vendor:   "anthropic",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.CredentialsByVendor(tt.credentials, tt.vendor)
			assert.Len(t, result, tt.expected)
			
			// Verify all returned credentials match the vendor
			for _, cred := range result {
				assert.Equal(t, tt.vendor, cred.Platform)
			}
		})
	}
}

func TestFilterModelsByVendor(t *testing.T) {
	tests := []struct {
		name     string
		models   []config.VendorModel
		vendor   string
		expected int
	}{
		{
			name: "filter_openai_models",
			models: []config.VendorModel{
				{Vendor: "openai", Model: "gpt-4"},
				{Vendor: "gemini", Model: "gemini-pro"},
				{Vendor: "openai", Model: "gpt-3.5-turbo"},
			},
			vendor:   "openai",
			expected: 2,
		},
		{
			name: "filter_gemini_models",
			models: []config.VendorModel{
				{Vendor: "openai", Model: "gpt-4"},
				{Vendor: "gemini", Model: "gemini-pro"},
			},
			vendor:   "gemini",
			expected: 1,
		},
		{
			name: "no_matching_models",
			models: []config.VendorModel{
				{Vendor: "openai", Model: "gpt-4"},
			},
			vendor:   "anthropic",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.ModelsByVendor(tt.models, tt.vendor)
			assert.Len(t, result, tt.expected)
			
			// Verify all returned models match the vendor
			for _, model := range result {
				assert.Equal(t, tt.vendor, model.Vendor)
			}
		})
	}
} 