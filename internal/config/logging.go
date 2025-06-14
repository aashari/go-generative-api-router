package config

import (
	"os"
	"strconv"
	"strings"
)

// LoggingConfig holds logging-related configuration
type LoggingConfig struct {
	Level        string `json:"level"`
	Environment  string `json:"environment"`
	ServiceName  string `json:"service_name"`
	Version      string `json:"version"`
	VerboseMode  bool   `json:"verbose_mode"`
	OutputFormat string `json:"output_format"`
	OutputFile   string `json:"output_file,omitempty"`

	// BrainyBuddy API integration settings
	EnableStructuredLogging   bool `json:"enable_structured_logging"`
	EnableRequestResponseLogs bool `json:"enable_request_response_logs"`
	EnableVendorLogging       bool `json:"enable_vendor_logging"`
	EnableHealthCheckLogging  bool `json:"enable_health_check_logging"`

	// Performance settings
	MaxBodyLogSize       int64 `json:"max_body_log_size"`
	MaxAttributesPerLog  int   `json:"max_attributes_per_log"`
	TruncateBase64Values bool  `json:"truncate_base64_values"`

	// Header forwarding settings
	ForwardTrackingHeaders bool     `json:"forward_tracking_headers"`
	CustomHeaders          []string `json:"custom_headers,omitempty"`
}

// LoadLoggingConfig loads logging configuration from environment
func LoadLoggingConfig() *LoggingConfig {
	config := &LoggingConfig{
		Level:        getEnvWithDefault("LOG_LEVEL", "INFO"),
		Environment:  getEnvWithDefault("ENVIRONMENT", "development"),
		ServiceName:  getEnvWithDefault("SERVICE_NAME", "generative-api-router"),
		Version:      getEnvWithDefault("VERSION", "unknown"),
		VerboseMode:  getEnvBool("VERBOSE_LOGGING", false),
		OutputFormat: getEnvWithDefault("LOG_FORMAT", "json"),
		OutputFile:   os.Getenv("LOG_FILE"),

		// BrainyBuddy API integration settings
		EnableStructuredLogging:   getEnvBool("ENABLE_STRUCTURED_LOGGING", true),
		EnableRequestResponseLogs: getEnvBool("ENABLE_REQUEST_RESPONSE_LOGS", true),
		EnableVendorLogging:       getEnvBool("ENABLE_VENDOR_LOGGING", true),
		EnableHealthCheckLogging:  getEnvBool("ENABLE_HEALTH_CHECK_LOGGING", false), // Only errors by default

		// Performance settings
		MaxBodyLogSize:       getEnvInt64("MAX_BODY_LOG_SIZE", 10240), // 10KB default
		MaxAttributesPerLog:  getEnvInt("MAX_ATTRIBUTES_PER_LOG", 50),
		TruncateBase64Values: getEnvBool("TRUNCATE_BASE64_VALUES", true),

		// Header forwarding settings
		ForwardTrackingHeaders: getEnvBool("FORWARD_TRACKING_HEADERS", true),
		CustomHeaders:          getEnvStringArray("CUSTOM_HEADERS", []string{}),
	}

	return config
}

// Validate validates the logging configuration
func (c *LoggingConfig) Validate() error {
	validLevels := map[string]bool{
		"DEBUG": true,
		"INFO":  true,
		"WARN":  true,
		"ERROR": true,
	}

	if !validLevels[strings.ToUpper(c.Level)] {
		c.Level = "INFO" // Default to INFO if invalid
	}

	validFormats := map[string]bool{
		"json": true,
		"text": true,
	}

	if !validFormats[strings.ToLower(c.OutputFormat)] {
		c.OutputFormat = "json" // Default to JSON
	}

	// Ensure reasonable limits
	if c.MaxBodyLogSize < 0 {
		c.MaxBodyLogSize = 0
	}
	if c.MaxBodyLogSize > 1048576 { // 1MB max
		c.MaxBodyLogSize = 1048576
	}

	if c.MaxAttributesPerLog < 0 {
		c.MaxAttributesPerLog = 10
	}
	if c.MaxAttributesPerLog > 200 { // Reasonable upper limit
		c.MaxAttributesPerLog = 200
	}

	return nil
}

// IsVerboseMode returns true if verbose logging is enabled
func (c *LoggingConfig) IsVerboseMode() bool {
	return c.VerboseMode
}

// ShouldLogHealthChecks returns true if health check logging is enabled
func (c *LoggingConfig) ShouldLogHealthChecks() bool {
	return c.EnableHealthCheckLogging
}

// ShouldLogVendorRequests returns true if vendor request logging is enabled
func (c *LoggingConfig) ShouldLogVendorRequests() bool {
	return c.EnableVendorLogging
}

// ShouldLogRequestResponse returns true if request/response logging is enabled
func (c *LoggingConfig) ShouldLogRequestResponse() bool {
	return c.EnableRequestResponseLogs
}

// GetMaxBodyLogSize returns the maximum body size to log
func (c *LoggingConfig) GetMaxBodyLogSize() int64 {
	return c.MaxBodyLogSize
}

// GetCustomHeaders returns the list of custom headers to forward
func (c *LoggingConfig) GetCustomHeaders() []string {
	return c.CustomHeaders
}

// ToMap converts the config to a map for logging purposes
func (c *LoggingConfig) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"level":                        c.Level,
		"environment":                  c.Environment,
		"service_name":                 c.ServiceName,
		"version":                      c.Version,
		"verbose_mode":                 c.VerboseMode,
		"output_format":                c.OutputFormat,
		"enable_structured_logging":    c.EnableStructuredLogging,
		"enable_request_response_logs": c.EnableRequestResponseLogs,
		"enable_vendor_logging":        c.EnableVendorLogging,
		"enable_health_check_logging":  c.EnableHealthCheckLogging,
		"max_body_log_size":            c.MaxBodyLogSize,
		"max_attributes_per_log":       c.MaxAttributesPerLog,
		"truncate_base64_values":       c.TruncateBase64Values,
		"forward_tracking_headers":     c.ForwardTrackingHeaders,
	}
}

// Helper functions

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvStringArray(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}

// GetLoggingConfigForEnvironment returns environment-specific logging configuration
func GetLoggingConfigForEnvironment(env string) *LoggingConfig {
	config := LoadLoggingConfig()
	config.Environment = env

	switch strings.ToLower(env) {
	case "production", "prod":
		// Production: Minimal logging for performance
		config.Level = getEnvWithDefault("LOG_LEVEL", "WARN")
		config.VerboseMode = false
		config.EnableHealthCheckLogging = false
		config.MaxBodyLogSize = 1024 // 1KB for production

	case "staging", "stage":
		// Staging: Balanced logging for debugging
		config.Level = getEnvWithDefault("LOG_LEVEL", "INFO")
		config.VerboseMode = getEnvBool("VERBOSE_LOGGING", false)
		config.EnableHealthCheckLogging = getEnvBool("ENABLE_HEALTH_CHECK_LOGGING", true)
		config.MaxBodyLogSize = 5120 // 5KB for staging

	case "development", "dev", "local":
		// Development: Full logging for debugging
		config.Level = getEnvWithDefault("LOG_LEVEL", "DEBUG")
		config.VerboseMode = getEnvBool("VERBOSE_LOGGING", true)
		config.EnableHealthCheckLogging = true
		config.MaxBodyLogSize = 10240 // 10KB for development

	default:
		// Unknown environment: Use defaults
		config.Validate()
	}

	return config
}
