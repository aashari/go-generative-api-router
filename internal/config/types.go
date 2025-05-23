package config

import "time"

// Config represents the complete application configuration
type Config struct {
	Server    ServerConfig             `json:"server" yaml:"server" mapstructure:"server"`
	Vendors   map[string]VendorConfig  `json:"vendors" yaml:"vendors" mapstructure:"vendors"`
	Models    []ModelConfig            `json:"models" yaml:"models" mapstructure:"models"`
	Logging   LoggingConfig            `json:"logging" yaml:"logging" mapstructure:"logging"`
	Security  SecurityConfig           `json:"security" yaml:"security" mapstructure:"security"`
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Host         string        `json:"host" yaml:"host" mapstructure:"host" default:"0.0.0.0"`
	Port         int           `json:"port" yaml:"port" mapstructure:"port" default:"8082"`
	ReadTimeout  time.Duration `json:"read_timeout" yaml:"read_timeout" mapstructure:"read_timeout" default:"30s"`
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout" mapstructure:"write_timeout" default:"30s"`
	IdleTimeout  time.Duration `json:"idle_timeout" yaml:"idle_timeout" mapstructure:"idle_timeout" default:"60s"`
}

// VendorConfig holds vendor-specific configuration
type VendorConfig struct {
	Name        string            `json:"name" yaml:"name" mapstructure:"name"`
	BaseURL     string            `json:"base_url" yaml:"base_url" mapstructure:"base_url"`
	Timeout     time.Duration     `json:"timeout" yaml:"timeout" mapstructure:"timeout" default:"60s"`
	MaxRetries  int               `json:"max_retries" yaml:"max_retries" mapstructure:"max_retries" default:"3"`
	Enabled     bool              `json:"enabled" yaml:"enabled" mapstructure:"enabled" default:"true"`
	ExtraConfig map[string]interface{} `json:"extra_config,omitempty" yaml:"extra_config,omitempty" mapstructure:"extra_config"`
}

// ModelConfig represents a model configuration
type ModelConfig struct {
	Vendor string `json:"vendor" yaml:"vendor" mapstructure:"vendor"`
	Model  string `json:"model" yaml:"model" mapstructure:"model"`
	Weight int    `json:"weight" yaml:"weight" mapstructure:"weight" default:"1"` // For weighted selection
}

// CredentialConfig represents credential configuration
type CredentialConfig struct {
	Platform string `json:"platform" yaml:"platform" mapstructure:"platform"`
	Type     string `json:"type" yaml:"type" mapstructure:"type"`
	Value    string `json:"value" yaml:"value" mapstructure:"value"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level      string `json:"level" yaml:"level" mapstructure:"level" default:"info"`
	Format     string `json:"format" yaml:"format" mapstructure:"format" default:"json"`
	Output     string `json:"output" yaml:"output" mapstructure:"output" default:"stdout"`
	Structured bool   `json:"structured" yaml:"structured" mapstructure:"structured" default:"true"`
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	CORS CORSConfig `json:"cors" yaml:"cors" mapstructure:"cors"`
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins []string `json:"allowed_origins" yaml:"allowed_origins" mapstructure:"allowed_origins" default:"*"`
	AllowedMethods []string `json:"allowed_methods" yaml:"allowed_methods" mapstructure:"allowed_methods" default:"GET,POST,OPTIONS,PUT,DELETE"`
	AllowedHeaders []string `json:"allowed_headers" yaml:"allowed_headers" mapstructure:"allowed_headers" default:"Accept,Content-Type,Content-Length,Accept-Encoding,X-CSRF-Token,Authorization"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         "0.0.0.0",
			Port:         8082,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		Vendors: map[string]VendorConfig{
			"openai": {
				Name:       "openai",
				BaseURL:    "https://api.openai.com/v1",
				Timeout:    60 * time.Second,
				MaxRetries: 3,
				Enabled:    true,
			},
			"gemini": {
				Name:       "gemini",
				BaseURL:    "https://generativelanguage.googleapis.com/v1beta/openai",
				Timeout:    60 * time.Second,
				MaxRetries: 3,
				Enabled:    true,
			},
		},
		Models: []ModelConfig{},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			Structured: true,
		},
		Security: SecurityConfig{
			CORS: CORSConfig{
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET", "POST", "OPTIONS", "PUT", "DELETE"},
				AllowedHeaders: []string{"Accept", "Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization"},
			},
		},
	}
} 