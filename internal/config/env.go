package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
)

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

// GetEnvWithDefault gets an environment variable with a default fallback
// This is a convenience function similar to the ones in main.go
func GetEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvBool gets a boolean environment variable with a default fallback
func GetEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	switch value {
	case "true", "TRUE", "1", "yes", "YES", "on", "ON":
		return true
	case "false", "FALSE", "0", "no", "NO", "off", "OFF":
		return false
	default:
		return defaultValue
	}
}

// GetEnvInt gets an integer environment variable with a default fallback
func GetEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	if intValue, err := strconv.Atoi(value); err == nil {
		return intValue
	}

	return defaultValue
}
