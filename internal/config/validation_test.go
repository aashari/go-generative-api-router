package config

import (
	"testing"

	"github.com/aashari/go-generative-api-router/internal/errors"
	"github.com/stretchr/testify/assert"
)

func TestValidateConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		credentials []Credential
		models      []VendorModel
		expectError bool
		errorType   errors.ErrorType
	}{
		{
			name: "valid_configuration",
			credentials: []Credential{
				{Platform: "openai", Type: "api-key", Value: "sk-test123456789012345"},
			},
			models: []VendorModel{
				{Vendor: "openai", Model: "gpt-4"},
			},
			expectError: false,
		},
		{
			name:        "empty_credentials",
			credentials: []Credential{},
			models: []VendorModel{
				{Vendor: "openai", Model: "gpt-4"},
			},
			expectError: true,
			errorType:   errors.ErrorTypeConfiguration,
		},
		{
			name: "empty_models",
			credentials: []Credential{
				{Platform: "openai", Type: "api-key", Value: "sk-test123456789012345"},
			},
			models:      []VendorModel{},
			expectError: true,
			errorType:   errors.ErrorTypeConfiguration,
		},
		{
			name: "missing_credentials_for_vendor",
			credentials: []Credential{
				{Platform: "openai", Type: "api-key", Value: "sk-test123456789012345"},
			},
			models: []VendorModel{
				{Vendor: "openai", Model: "gpt-4"},
				{Vendor: "gemini", Model: "gemini-pro"},
			},
			expectError: true,
			errorType:   errors.ErrorTypeConfiguration,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfiguration(tt.credentials, tt.models)

			if tt.expectError {
				assert.NotNil(t, err)
				assert.Equal(t, tt.errorType, err.Type)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestValidateCredentials(t *testing.T) {
	tests := []struct {
		name        string
		credentials []Credential
		expectError bool
	}{
		{
			name: "valid_openai_credential",
			credentials: []Credential{
				{Platform: "openai", Type: "api-key", Value: "sk-test123456789012345"},
			},
			expectError: false,
		},
		{
			name: "valid_gemini_credential",
			credentials: []Credential{
				{Platform: "gemini", Type: "api-key", Value: "test123456789"},
			},
			expectError: false,
		},
		{
			name: "invalid_platform",
			credentials: []Credential{
				{Platform: "invalid", Type: "api-key", Value: "test123"},
			},
			expectError: true,
		},
		{
			name: "invalid_type",
			credentials: []Credential{
				{Platform: "openai", Type: "invalid", Value: "sk-test123456789012345"},
			},
			expectError: true,
		},
		{
			name: "empty_value",
			credentials: []Credential{
				{Platform: "openai", Type: "api-key", Value: ""},
			},
			expectError: true,
		},
		{
			name: "invalid_openai_key_format",
			credentials: []Credential{
				{Platform: "openai", Type: "api-key", Value: "invalid-key"},
			},
			expectError: true,
		},
		{
			name: "short_openai_key",
			credentials: []Credential{
				{Platform: "openai", Type: "api-key", Value: "sk-short"},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCredentials(tt.credentials)

			if tt.expectError {
				assert.NotNil(t, err)
				assert.Equal(t, errors.ErrorTypeConfiguration, err.Type)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestValidateVendorModels(t *testing.T) {
	tests := []struct {
		name        string
		models      []VendorModel
		expectError bool
	}{
		{
			name: "valid_models",
			models: []VendorModel{
				{Vendor: "openai", Model: "gpt-4"},
				{Vendor: "gemini", Model: "gemini-pro"},
			},
			expectError: false,
		},
		{
			name:        "empty_models",
			models:      []VendorModel{},
			expectError: true,
		},
		{
			name: "invalid_vendor",
			models: []VendorModel{
				{Vendor: "invalid", Model: "test-model"},
			},
			expectError: true,
		},
		{
			name: "empty_model",
			models: []VendorModel{
				{Vendor: "openai", Model: ""},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVendorModels(tt.models)

			if tt.expectError {
				assert.NotNil(t, err)
				assert.Equal(t, errors.ErrorTypeConfiguration, err.Type)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestValidateAPIKeyFormat(t *testing.T) {
	tests := []struct {
		name        string
		platform    string
		apiKey      string
		expectError bool
	}{
		{"valid_openai_key", "openai", "sk-test123456789012345", false},
		{"invalid_openai_prefix", "openai", "invalid-key", true},
		{"short_openai_key", "openai", "sk-short", true},
		{"valid_gemini_key", "gemini", "test123456789", false},
		{"short_gemini_key", "gemini", "short", true},
		{"valid_anthropic_key", "anthropic", "sk-ant-test123456789", false},
		{"invalid_anthropic_prefix", "anthropic", "sk-test123", true},
		{"unknown_platform", "unknown", "any-key", false}, // Should not error for unknown platforms
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAPIKeyFormat(tt.platform, tt.apiKey)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateBusinessRules(t *testing.T) {
	tests := []struct {
		name        string
		credentials []Credential
		models      []VendorModel
		expectError bool
		description string
	}{
		{
			name: "valid_business_rules",
			credentials: []Credential{
				{Platform: "openai", Type: "api-key", Value: "sk-test123456789012345"},
				{Platform: "gemini", Type: "api-key", Value: "test123456789"},
			},
			models: []VendorModel{
				{Vendor: "openai", Model: "gpt-4"},
				{Vendor: "gemini", Model: "gemini-pro"},
			},
			expectError: false,
			description: "Should pass when all vendors have credentials",
		},
		{
			name: "missing_credentials_for_vendor",
			credentials: []Credential{
				{Platform: "openai", Type: "api-key", Value: "sk-test123456789012345"},
			},
			models: []VendorModel{
				{Vendor: "openai", Model: "gpt-4"},
				{Vendor: "gemini", Model: "gemini-pro"},
			},
			expectError: true,
			description: "Should fail when model vendor has no credentials",
		},
		{
			name: "duplicate_models",
			credentials: []Credential{
				{Platform: "openai", Type: "api-key", Value: "sk-test123456789012345"},
			},
			models: []VendorModel{
				{Vendor: "openai", Model: "gpt-4"},
				{Vendor: "openai", Model: "gpt-4"}, // Duplicate
			},
			expectError: true,
			description: "Should fail when duplicate models exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBusinessRules(tt.credentials, tt.models)

			if tt.expectError {
				assert.NotNil(t, err)
				assert.Equal(t, errors.ErrorTypeConfiguration, err.Type)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
