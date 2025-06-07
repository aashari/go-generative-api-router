package utils

import (
	"os"
	"strconv"
	"time"
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
