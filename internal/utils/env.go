package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// GetEnvDuration gets a duration from environment variable with a default fallback
func GetEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	return defaultValue
}

// GetEnvString gets a string from environment variable with a default fallback
func GetEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvInt gets an integer from environment variable with a default fallback
func GetEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GetEnvBool gets a boolean from environment variable with a default fallback
func GetEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// GetEnvFloat64 gets a float64 from environment variable with a default fallback
func GetEnvFloat64(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

// GetEnvPort gets a port number from environment variable with validation
func GetEnvPort(key string, defaultValue int) int {
	port := GetEnvInt(key, defaultValue)
	if port < 1 || port > 65535 {
		return defaultValue
	}
	return port
}

// GetEnvTimeout gets a timeout duration with validation
func GetEnvTimeout(key string, defaultValue time.Duration, minValue, maxValue time.Duration) time.Duration {
	duration := GetEnvDuration(key, defaultValue)
	if duration < minValue {
		return minValue
	}
	if duration > maxValue {
		return maxValue
	}
	return duration
}

// IsProduction checks if the application is running in production mode
func IsProduction() bool {
	env := GetEnvString("ENVIRONMENT", "development")
	return env == "production" || env == "prod"
}

// IsDevelopment checks if the application is running in development mode
func IsDevelopment() bool {
	return !IsProduction()
}

// GetLogLevel gets the log level from environment with validation
func GetLogLevel() string {
	level := GetEnvString("LOG_LEVEL", "info")
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}

	if validLevels[level] {
		return level
	}

	if IsProduction() {
		return "info"
	}
	return "debug"
}

// LoadEnvFile loads environment variables from .env file
// This function mimics the behavior of Node.js dotenv package
func LoadEnvFile(envFilePath ...string) error {
	// Default to .env in current directory if no path specified
	var envFile string
	if len(envFilePath) > 0 && envFilePath[0] != "" {
		envFile = envFilePath[0]
	} else {
		envFile = ".env"
	}

	// Check if .env file exists
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		// .env file doesn't exist, which is okay - just continue with system env vars
		return nil
	}

	// Load the .env file
	err := godotenv.Load(envFile)
	if err != nil {
		return fmt.Errorf("error loading %s file: %w", envFile, err)
	}

	return nil
}

// LoadEnvFromMultiplePaths attempts to load .env from multiple possible locations
// This is useful for different deployment scenarios
func LoadEnvFromMultiplePaths() error {
	possiblePaths := []string{
		".env",                                   // Current directory
		"configs/.env",                           // Configs directory
		"../.env",                                // Parent directory
		filepath.Join(os.Getenv("HOME"), ".env"), // Home directory
	}

	for _, path := range possiblePaths {
		if err := LoadEnvFile(path); err != nil {
			continue
		}
		// Successfully loaded from this path
		return nil
	}

	// If we get here, none of the paths worked, but that's okay
	// The application can still run with system environment variables
	return nil
}

// MustLoadEnvFile loads environment variables from .env file and panics on error
// Use this only when .env file is absolutely required
func MustLoadEnvFile(envFilePath ...string) {
	if err := LoadEnvFile(envFilePath...); err != nil {
		panic(fmt.Sprintf("Failed to load environment file: %v", err))
	}
}
