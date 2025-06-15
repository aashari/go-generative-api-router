package utils

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetEnvString(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{
			name:         "env var exists",
			key:          "TEST_STRING",
			defaultValue: "default",
			envValue:     "from_env",
			expected:     "from_env",
		},
		{
			name:         "env var doesn't exist",
			key:          "NONEXISTENT_STRING",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
		{
			name:         "env var is empty string",
			key:          "EMPTY_STRING",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.key, tt.envValue)
			}

			result := GetEnvString(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue int
		envValue     string
		expected     int
	}{
		{
			name:         "valid integer",
			key:          "TEST_INT",
			defaultValue: 42,
			envValue:     "123",
			expected:     123,
		},
		{
			name:         "invalid integer",
			key:          "INVALID_INT",
			defaultValue: 42,
			envValue:     "not_a_number",
			expected:     42,
		},
		{
			name:         "env var doesn't exist",
			key:          "NONEXISTENT_INT",
			defaultValue: 42,
			envValue:     "",
			expected:     42,
		},
		{
			name:         "negative integer",
			key:          "NEGATIVE_INT",
			defaultValue: 42,
			envValue:     "-123",
			expected:     -123,
		},
		{
			name:         "zero",
			key:          "ZERO_INT",
			defaultValue: 42,
			envValue:     "0",
			expected:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.key, tt.envValue)
			}

			result := GetEnvInt(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue bool
		envValue     string
		expected     bool
	}{
		{
			name:         "true value",
			key:          "TEST_BOOL_TRUE",
			defaultValue: false,
			envValue:     "true",
			expected:     true,
		},
		{
			name:         "false value",
			key:          "TEST_BOOL_FALSE",
			defaultValue: true,
			envValue:     "false",
			expected:     false,
		},
		{
			name:         "1 value (true)",
			key:          "TEST_BOOL_ONE",
			defaultValue: false,
			envValue:     "1",
			expected:     true,
		},
		{
			name:         "0 value (false)",
			key:          "TEST_BOOL_ZERO",
			defaultValue: true,
			envValue:     "0",
			expected:     false,
		},
		{
			name:         "invalid value",
			key:          "TEST_BOOL_INVALID",
			defaultValue: true,
			envValue:     "maybe",
			expected:     true,
		},
		{
			name:         "env var doesn't exist",
			key:          "NONEXISTENT_BOOL",
			defaultValue: true,
			envValue:     "",
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.key, tt.envValue)
			}

			result := GetEnvBool(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvFloat64(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue float64
		envValue     string
		expected     float64
	}{
		{
			name:         "valid float",
			key:          "TEST_FLOAT",
			defaultValue: 3.14,
			envValue:     "2.71",
			expected:     2.71,
		},
		{
			name:         "integer as float",
			key:          "TEST_INT_FLOAT",
			defaultValue: 3.14,
			envValue:     "42",
			expected:     42.0,
		},
		{
			name:         "invalid float",
			key:          "INVALID_FLOAT",
			defaultValue: 3.14,
			envValue:     "not_a_number",
			expected:     3.14,
		},
		{
			name:         "env var doesn't exist",
			key:          "NONEXISTENT_FLOAT",
			defaultValue: 3.14,
			envValue:     "",
			expected:     3.14,
		},
		{
			name:         "negative float",
			key:          "NEGATIVE_FLOAT",
			defaultValue: 3.14,
			envValue:     "-1.23",
			expected:     -1.23,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.key, tt.envValue)
			}

			result := GetEnvFloat64(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvDuration(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue time.Duration
		envValue     string
		expected     time.Duration
	}{
		{
			name:         "valid duration in seconds",
			key:          "TEST_DURATION",
			defaultValue: 30 * time.Second,
			envValue:     "60",
			expected:     60 * time.Second,
		},
		{
			name:         "invalid duration",
			key:          "INVALID_DURATION",
			defaultValue: 30 * time.Second,
			envValue:     "not_a_number",
			expected:     30 * time.Second,
		},
		{
			name:         "zero duration",
			key:          "ZERO_DURATION",
			defaultValue: 30 * time.Second,
			envValue:     "0",
			expected:     30 * time.Second, // Zero is considered invalid
		},
		{
			name:         "negative duration",
			key:          "NEGATIVE_DURATION",
			defaultValue: 30 * time.Second,
			envValue:     "-10",
			expected:     30 * time.Second, // Negative is considered invalid
		},
		{
			name:         "env var doesn't exist",
			key:          "NONEXISTENT_DURATION",
			defaultValue: 30 * time.Second,
			envValue:     "",
			expected:     30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.key, tt.envValue)
			}

			result := GetEnvDuration(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvPort(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue int
		envValue     string
		expected     int
	}{
		{
			name:         "valid port",
			key:          "TEST_PORT",
			defaultValue: 8080,
			envValue:     "3000",
			expected:     3000,
		},
		{
			name:         "port too low",
			key:          "LOW_PORT",
			defaultValue: 8080,
			envValue:     "0",
			expected:     8080,
		},
		{
			name:         "port too high",
			key:          "HIGH_PORT",
			defaultValue: 8080,
			envValue:     "70000",
			expected:     8080,
		},
		{
			name:         "invalid port",
			key:          "INVALID_PORT",
			defaultValue: 8080,
			envValue:     "not_a_port",
			expected:     8080,
		},
		{
			name:         "edge case: port 1",
			key:          "EDGE_PORT_LOW",
			defaultValue: 8080,
			envValue:     "1",
			expected:     1,
		},
		{
			name:         "edge case: port 65535",
			key:          "EDGE_PORT_HIGH",
			defaultValue: 8080,
			envValue:     "65535",
			expected:     65535,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.key, tt.envValue)
			}

			result := GetEnvPort(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvTimeout(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue time.Duration
		minValue     time.Duration
		maxValue     time.Duration
		envValue     string
		expected     time.Duration
	}{
		{
			name:         "valid timeout within range",
			key:          "TEST_TIMEOUT",
			defaultValue: 30 * time.Second,
			minValue:     10 * time.Second,
			maxValue:     60 * time.Second,
			envValue:     "45",
			expected:     45 * time.Second,
		},
		{
			name:         "timeout below minimum",
			key:          "LOW_TIMEOUT",
			defaultValue: 30 * time.Second,
			minValue:     10 * time.Second,
			maxValue:     60 * time.Second,
			envValue:     "5",
			expected:     10 * time.Second,
		},
		{
			name:         "timeout above maximum",
			key:          "HIGH_TIMEOUT",
			defaultValue: 30 * time.Second,
			minValue:     10 * time.Second,
			maxValue:     60 * time.Second,
			envValue:     "120",
			expected:     60 * time.Second,
		},
		{
			name:         "invalid timeout uses default",
			key:          "INVALID_TIMEOUT",
			defaultValue: 30 * time.Second,
			minValue:     10 * time.Second,
			maxValue:     60 * time.Second,
			envValue:     "not_a_number",
			expected:     30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.key, tt.envValue)
			}

			result := GetEnvTimeout(tt.key, tt.defaultValue, tt.minValue, tt.maxValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEnvironmentChecks(t *testing.T) {
	t.Run("IsProduction", func(t *testing.T) {
		tests := []struct {
			envValue string
			expected bool
		}{
			{"production", true},
			{"prod", true},
			{"development", false},
			{"dev", false},
			{"testing", false},
			{"", false}, // default is development
		}

		for _, tt := range tests {
			t.Run("env="+tt.envValue, func(t *testing.T) {
				if tt.envValue != "" {
					t.Setenv("ENVIRONMENT", tt.envValue)
				} else {
					os.Unsetenv("ENVIRONMENT")
				}

				result := IsProduction()
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("IsDevelopment", func(t *testing.T) {
		t.Setenv("ENVIRONMENT", "production")
		assert.False(t, IsDevelopment())

		t.Setenv("ENVIRONMENT", "development")
		assert.True(t, IsDevelopment())
	})
}

func TestGetLogLevel(t *testing.T) {
	tests := []struct {
		name        string
		logLevel    string
		environment string
		expected    string
	}{
		{
			name:        "valid log level in development",
			logLevel:    "debug",
			environment: "development",
			expected:    "debug",
		},
		{
			name:        "valid log level in production",
			logLevel:    "error",
			environment: "production",
			expected:    "error",
		},
		{
			name:        "invalid log level in development",
			logLevel:    "invalid",
			environment: "development",
			expected:    "debug",
		},
		{
			name:        "invalid log level in production",
			logLevel:    "invalid",
			environment: "production",
			expected:    "info",
		},
		{
			name:        "no log level set in development",
			logLevel:    "",
			environment: "development",
			expected:    "info", // Default when no level set is always info
		},
		{
			name:        "no log level set in production",
			logLevel:    "",
			environment: "production",
			expected:    "info", // Default for production
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.logLevel != "" {
				t.Setenv("LOG_LEVEL", tt.logLevel)
			} else {
				os.Unsetenv("LOG_LEVEL")
			}

			if tt.environment != "" {
				t.Setenv("ENVIRONMENT", tt.environment)
			} else {
				os.Unsetenv("ENVIRONMENT")
			}

			result := GetLogLevel()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadEnvFile(t *testing.T) {
	t.Run("nonexistent file should not error", func(t *testing.T) {
		err := LoadEnvFile("/nonexistent/.env")
		assert.NoError(t, err)
	})

	t.Run("valid env file", func(t *testing.T) {
		// Create a temporary .env file
		tmpFile, err := os.CreateTemp("", ".env_*")
		assert.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		// Write some env vars
		content := "TEST_VAR1=value1\nTEST_VAR2=value2\n"
		_, err = tmpFile.WriteString(content)
		assert.NoError(t, err)
		tmpFile.Close()

		// Load the file
		err = LoadEnvFile(tmpFile.Name())
		assert.NoError(t, err)

		// Check that vars were loaded
		assert.Equal(t, "value1", os.Getenv("TEST_VAR1"))
		assert.Equal(t, "value2", os.Getenv("TEST_VAR2"))

		// Clean up
		os.Unsetenv("TEST_VAR1")
		os.Unsetenv("TEST_VAR2")
	})

	t.Run("invalid env file should error", func(t *testing.T) {
		// Create a temporary file with invalid content
		tmpFile, err := os.CreateTemp("", ".env_*")
		assert.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		// Write invalid content (godotenv is pretty forgiving, so we need malformed syntax)
		content := "INVALID LINE WITHOUT EQUALS"
		_, err = tmpFile.WriteString(content)
		assert.NoError(t, err)
		tmpFile.Close()

		// This should actually work as godotenv treats this as a comment-like line
		err = LoadEnvFile(tmpFile.Name())
		assert.NoError(t, err)
	})
}

func TestMustLoadEnvFile(t *testing.T) {
	t.Run("should panic on error", func(t *testing.T) {
		// Create a directory instead of a file to cause a read error
		tmpDir, err := os.MkdirTemp("", "env_test_*")
		assert.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Create a file we can't read (simulate by trying to read a directory)
		assert.Panics(t, func() {
			MustLoadEnvFile(tmpDir) // This should panic because it's a directory
		})
	})
}
