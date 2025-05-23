package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Loader handles loading configuration from multiple sources
type Loader struct {
	v *viper.Viper
}

// NewLoader creates a new configuration loader
func NewLoader() *Loader {
	v := viper.New()
	
	// Set default configuration
	setDefaults(v)
	
	// Configure environment variable handling
	v.SetEnvPrefix("GENAPI")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	
	return &Loader{v: v}
}

// LoadConfig loads configuration from multiple sources in priority order:
// 1. Environment variables (highest priority)
// 2. Configuration files
// 3. Default values (lowest priority)
func (l *Loader) LoadConfig(configPaths ...string) (*Config, error) {
	// Set configuration file search paths
	if len(configPaths) == 0 {
		configPaths = []string{".", "./config", "/etc/genapi"}
	}
	
	for _, path := range configPaths {
		l.v.AddConfigPath(path)
	}
	
	// Set configuration file names to search for
	l.v.SetConfigName("config")
	l.v.SetConfigType("yaml") // Primary format
	
	// Try to read configuration file (optional)
	if err := l.v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found is OK, we'll use defaults and env vars
	}
	
	// Load legacy JSON files if they exist
	if err := l.loadLegacyFiles(); err != nil {
		return nil, fmt.Errorf("error loading legacy files: %w", err)
	}
	
	// Unmarshal into config struct
	var config Config
	if err := l.v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}
	
	// Validate configuration
	if err := l.validateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}
	
	return &config, nil
}

// LoadCredentials loads credentials from file or environment
func (l *Loader) LoadCredentials(filePath string) ([]CredentialConfig, error) {
	// Try environment variables first
	if creds := l.loadCredentialsFromEnv(); len(creds) > 0 {
		return creds, nil
	}
	
	// Fall back to file
	if filePath == "" {
		filePath = l.v.GetString("credentials_file")
		if filePath == "" {
			filePath = "credentials.json"
		}
	}
	
	return l.loadCredentialsFromFile(filePath)
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8082)
	v.SetDefault("server.read_timeout", "30s")
	v.SetDefault("server.write_timeout", "30s")
	v.SetDefault("server.idle_timeout", "60s")
	
	// Vendor defaults
	v.SetDefault("vendors.openai.name", "openai")
	v.SetDefault("vendors.openai.base_url", "https://api.openai.com/v1")
	v.SetDefault("vendors.openai.timeout", "60s")
	v.SetDefault("vendors.openai.max_retries", 3)
	v.SetDefault("vendors.openai.enabled", true)
	
	v.SetDefault("vendors.gemini.name", "gemini")
	v.SetDefault("vendors.gemini.base_url", "https://generativelanguage.googleapis.com/v1beta/openai")
	v.SetDefault("vendors.gemini.timeout", "60s")
	v.SetDefault("vendors.gemini.max_retries", 3)
	v.SetDefault("vendors.gemini.enabled", true)
	
	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
	v.SetDefault("logging.output", "stdout")
	v.SetDefault("logging.structured", true)
	
	// Security defaults
	v.SetDefault("security.cors.allowed_origins", []string{"*"})
	v.SetDefault("security.cors.allowed_methods", []string{"GET", "POST", "OPTIONS", "PUT", "DELETE"})
	v.SetDefault("security.cors.allowed_headers", []string{"Accept", "Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization"})
	
	// File paths
	v.SetDefault("credentials_file", "credentials.json")
	v.SetDefault("models_file", "models.json")
}

// loadLegacyFiles loads the existing JSON configuration files
func (l *Loader) loadLegacyFiles() error {
	// Load models from models.json
	modelsFile := l.v.GetString("models_file")
	if _, err := os.Stat(modelsFile); err == nil {
		models, err := l.loadModelsFromFile(modelsFile)
		if err != nil {
			return fmt.Errorf("error loading models file %s: %w", modelsFile, err)
		}
		l.v.Set("models", models)
	}
	
	return nil
}

// loadModelsFromFile loads models from a JSON file
func (l *Loader) loadModelsFromFile(filePath string) ([]ModelConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	
	// Support both old and new format
	var oldModels []struct {
		Vendor string `json:"vendor"`
		Model  string `json:"model"`
	}
	
	if err := json.Unmarshal(data, &oldModels); err != nil {
		return nil, err
	}
	
	models := make([]ModelConfig, len(oldModels))
	for i, om := range oldModels {
		models[i] = ModelConfig{
			Vendor: om.Vendor,
			Model:  om.Model,
			Weight: 1, // Default weight
		}
	}
	
	return models, nil
}

// loadCredentialsFromFile loads credentials from a JSON file
func (l *Loader) loadCredentialsFromFile(filePath string) ([]CredentialConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	
	var creds []CredentialConfig
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}
	
	return creds, nil
}

// loadCredentialsFromEnv loads credentials from environment variables
func (l *Loader) loadCredentialsFromEnv() []CredentialConfig {
	var creds []CredentialConfig
	
	// Check for OpenAI API key
	if openaiKey := os.Getenv("GENAPI_OPENAI_API_KEY"); openaiKey != "" {
		creds = append(creds, CredentialConfig{
			Platform: "openai",
			Type:     "api-key",
			Value:    openaiKey,
		})
	}
	
	// Check for Gemini API key
	if geminiKey := os.Getenv("GENAPI_GEMINI_API_KEY"); geminiKey != "" {
		creds = append(creds, CredentialConfig{
			Platform: "gemini",
			Type:     "api-key",
			Value:    geminiKey,
		})
	}
	
	return creds
}

// validateConfig validates the loaded configuration
func (l *Loader) validateConfig(config *Config) error {
	// Validate server configuration
	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}
	
	// Validate vendor configurations
	for name, vendor := range config.Vendors {
		if vendor.BaseURL == "" {
			return fmt.Errorf("vendor %s missing base_url", name)
		}
		if vendor.Timeout <= 0 {
			return fmt.Errorf("vendor %s has invalid timeout: %v", name, vendor.Timeout)
		}
	}
	
	// Validate models
	for i, model := range config.Models {
		if model.Vendor == "" {
			return fmt.Errorf("model %d missing vendor", i)
		}
		if model.Model == "" {
			return fmt.Errorf("model %d missing model name", i)
		}
		if _, exists := config.Vendors[model.Vendor]; !exists {
			return fmt.Errorf("model %d references unknown vendor: %s", i, model.Vendor)
		}
	}
	
	return nil
}

// GetConfigExample returns an example configuration file content
func GetConfigExample() string {
	return `# Generative API Router Configuration
server:
  host: "0.0.0.0"
  port: 8082
  read_timeout: "30s"
  write_timeout: "30s"
  idle_timeout: "60s"

vendors:
  openai:
    name: "openai"
    base_url: "https://api.openai.com/v1"
    timeout: "60s"
    max_retries: 3
    enabled: true
  gemini:
    name: "gemini"
    base_url: "https://generativelanguage.googleapis.com/v1beta/openai"
    timeout: "60s"
    max_retries: 3
    enabled: true

models:
  - vendor: "openai"
    model: "gpt-4o"
    weight: 1
  - vendor: "gemini"
    model: "gemini-2.0-flash"
    weight: 1

logging:
  level: "info"
  format: "json"
  output: "stdout"
  structured: true

security:
  cors:
    allowed_origins: ["*"]
    allowed_methods: ["GET", "POST", "OPTIONS", "PUT", "DELETE"]
    allowed_headers: ["Accept", "Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization"]
`
} 