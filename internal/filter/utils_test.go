package filter

import (
	"testing"

	"github.com/aashari/go-generative-api-router/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestCredentialsByVendor(t *testing.T) {
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
		{
			name:        "empty_credentials",
			credentials: []config.Credential{},
			vendor:      "openai",
			expected:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CredentialsByVendor(tt.credentials, tt.vendor)
			assert.Len(t, result, tt.expected)
			
			// Verify all returned credentials match the vendor
			for _, cred := range result {
				assert.Equal(t, tt.vendor, cred.Platform)
			}
		})
	}
}

func TestModelsByVendor(t *testing.T) {
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
		{
			name:     "empty_models",
			models:   []config.VendorModel{},
			vendor:   "openai",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ModelsByVendor(tt.models, tt.vendor)
			assert.Len(t, result, tt.expected)
			
			// Verify all returned models match the vendor
			for _, model := range result {
				assert.Equal(t, tt.vendor, model.Vendor)
			}
		})
	}
} 