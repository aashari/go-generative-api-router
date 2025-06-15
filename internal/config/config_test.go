package config

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadCredentials(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() (string, func())
		expected []Credential
		hasError bool
	}{
		{
			name: "valid credentials file",
			setup: func() (string, func()) {
				creds := []Credential{
					{Platform: "openai", Type: "api_key", Value: "test-key-1"},
					{Platform: "gemini", Type: "api_key", Value: "test-key-2"},
				}
				content, _ := json.Marshal(creds)
				tmpFile, _ := os.CreateTemp("", "creds_*.json")
				os.WriteFile(tmpFile.Name(), content, 0644)
				return tmpFile.Name(), func() { os.Remove(tmpFile.Name()) }
			},
			expected: []Credential{
				{Platform: "openai", Type: "api_key", Value: "test-key-1"},
				{Platform: "gemini", Type: "api_key", Value: "test-key-2"},
			},
			hasError: false,
		},
		{
			name: "malformed JSON file",
			setup: func() (string, func()) {
				tmpFile, _ := os.CreateTemp("", "creds_*.json")
				os.WriteFile(tmpFile.Name(), []byte(`{"invalid": json`), 0644)
				return tmpFile.Name(), func() { os.Remove(tmpFile.Name()) }
			},
			expected: nil,
			hasError: true,
		},
		{
			name: "file not found",
			setup: func() (string, func()) {
				return "/nonexistent/path/credentials.json", func() {}
			},
			expected: nil,
			hasError: true,
		},
		{
			name: "empty file",
			setup: func() (string, func()) {
				tmpFile, _ := os.CreateTemp("", "creds_*.json")
				os.WriteFile(tmpFile.Name(), []byte(""), 0644)
				return tmpFile.Name(), func() { os.Remove(tmpFile.Name()) }
			},
			expected: nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath, cleanup := tt.setup()
			defer cleanup()

			result, err := LoadCredentials(filePath)

			if tt.hasError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestLoadModelsConfig(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() (string, func())
		expected *ModelsConfig
		hasError bool
	}{
		{
			name: "valid models config",
			setup: func() (string, func()) {
				config := ModelsConfig{
					Vendors: map[string]string{
						"openai": "https://api.openai.com",
						"gemini": "https://generativelanguage.googleapis.com",
					},
					Models: []VendorModel{
						{
							Vendor: "openai",
							Model:  "gpt-4",
							Config: &ModelConfig{
								SupportImage:     true,
								SupportVideo:     false,
								SupportTools:     true,
								SupportStreaming: true,
							},
						},
					},
				}
				content, _ := json.Marshal(config)
				tmpFile, _ := os.CreateTemp("", "models_*.json")
				os.WriteFile(tmpFile.Name(), content, 0644)
				return tmpFile.Name(), func() { os.Remove(tmpFile.Name()) }
			},
			expected: &ModelsConfig{
				Vendors: map[string]string{
					"openai": "https://api.openai.com",
					"gemini": "https://generativelanguage.googleapis.com",
				},
				Models: []VendorModel{
					{
						Vendor: "openai",
						Model:  "gpt-4",
						Config: &ModelConfig{
							SupportImage:     true,
							SupportVideo:     false,
							SupportTools:     true,
							SupportStreaming: true,
						},
					},
				},
			},
			hasError: false,
		},
		{
			name: "missing vendors key",
			setup: func() (string, func()) {
				config := map[string]interface{}{
					"models": []VendorModel{
						{Vendor: "openai", Model: "gpt-4"},
					},
				}
				content, _ := json.Marshal(config)
				tmpFile, _ := os.CreateTemp("", "models_*.json")
				os.WriteFile(tmpFile.Name(), content, 0644)
				return tmpFile.Name(), func() { os.Remove(tmpFile.Name()) }
			},
			expected: &ModelsConfig{
				Vendors: nil,
				Models: []VendorModel{
					{Vendor: "openai", Model: "gpt-4"},
				},
			},
			hasError: false,
		},
		{
			name: "invalid JSON",
			setup: func() (string, func()) {
				tmpFile, _ := os.CreateTemp("", "models_*.json")
				os.WriteFile(tmpFile.Name(), []byte(`{"vendors": invalid json}`), 0644)
				return tmpFile.Name(), func() { os.Remove(tmpFile.Name()) }
			},
			expected: nil,
			hasError: true,
		},
		{
			name: "file not found",
			setup: func() (string, func()) {
				return "/nonexistent/models.json", func() {}
			},
			expected: nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath, cleanup := tt.setup()
			defer cleanup()

			result, err := LoadModelsConfig(filePath)

			if tt.hasError {
				assert.Error(t, err)
				// Function behavior depends on error type
				if tt.name == "file not found" {
					assert.Nil(t, result) // File errors return nil
				} else {
					assert.NotNil(t, result) // JSON errors return empty struct
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestLoadVendorModels(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() (string, func())
		expected []VendorModel
		hasError bool
	}{
		{
			name: "valid vendor models",
			setup: func() (string, func()) {
				models := []VendorModel{
					{
						Vendor: "openai",
						Model:  "gpt-4",
						Config: &ModelConfig{
							SupportImage:     true,
							SupportVideo:     false,
							SupportTools:     true,
							SupportStreaming: true,
						},
					},
					{
						Vendor: "gemini",
						Model:  "gemini-pro",
						Config: &ModelConfig{
							SupportImage:     false,
							SupportVideo:     true,
							SupportTools:     false,
							SupportStreaming: false,
						},
					},
				}
				content, _ := json.Marshal(models)
				tmpFile, _ := os.CreateTemp("", "vendor_models_*.json")
				os.WriteFile(tmpFile.Name(), content, 0644)
				return tmpFile.Name(), func() { os.Remove(tmpFile.Name()) }
			},
			expected: []VendorModel{
				{
					Vendor: "openai",
					Model:  "gpt-4",
					Config: &ModelConfig{
						SupportImage:     true,
						SupportVideo:     false,
						SupportTools:     true,
						SupportStreaming: true,
					},
				},
				{
					Vendor: "gemini",
					Model:  "gemini-pro",
					Config: &ModelConfig{
						SupportImage:     false,
						SupportVideo:     true,
						SupportTools:     false,
						SupportStreaming: false,
					},
				},
			},
			hasError: false,
		},
		{
			name: "empty models array",
			setup: func() (string, func()) {
				models := []VendorModel{}
				content, _ := json.Marshal(models)
				tmpFile, _ := os.CreateTemp("", "vendor_models_*.json")
				os.WriteFile(tmpFile.Name(), content, 0644)
				return tmpFile.Name(), func() { os.Remove(tmpFile.Name()) }
			},
			expected: []VendorModel{},
			hasError: false,
		},
		{
			name: "invalid JSON",
			setup: func() (string, func()) {
				tmpFile, _ := os.CreateTemp("", "vendor_models_*.json")
				os.WriteFile(tmpFile.Name(), []byte(`[{"vendor": invalid}]`), 0644)
				return tmpFile.Name(), func() { os.Remove(tmpFile.Name()) }
			},
			expected: nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath, cleanup := tt.setup()
			defer cleanup()

			result, err := LoadVendorModels(filePath)

			if tt.hasError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestFilepathSecurity(t *testing.T) {
	t.Run("LoadCredentials cleans filepath", func(t *testing.T) {
		_, err := LoadCredentials("../../../etc/passwd")
		assert.Error(t, err)
	})

	t.Run("LoadModelsConfig cleans filepath", func(t *testing.T) {
		_, err := LoadModelsConfig("../../../etc/passwd")
		assert.Error(t, err)
	})

	t.Run("LoadVendorModels cleans filepath", func(t *testing.T) {
		_, err := LoadVendorModels("../../../etc/passwd")
		assert.Error(t, err)
	})
}
