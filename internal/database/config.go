package database

import (
	"fmt"
	"os"
	"strings"
)

// DatabaseConfig holds MongoDB connection configuration
type DatabaseConfig struct {
	// MongoDB connection URI (includes all connection details including auth)
	URI string
	// The current environment (local, development, production, or test)
	Environment string
	// Database name based on environment and service name
	DatabaseName string
	// Application name for MongoDB connection
	AppName string
}

// GetDatabaseConfig retrieves the MongoDB database configuration based on environment variables
// Following the BrainyBuddy pattern: auto-generates database name based on service name and environment
func GetDatabaseConfig() *DatabaseConfig {
	// Get environment, default to development
	environment := strings.ToLower(os.Getenv("ENVIRONMENT"))
	if environment == "" {
		environment = "development"
	}

	// Get service name from environment
	serviceName := os.Getenv("SERVICE_NAME")
	if serviceName == "" {
		serviceName = "go-generative-api-router"
	}

	// Determine environment prefix based on environment
	var envPrefix string
	switch environment {
	case "production", "prod":
		envPrefix = "prod"
		environment = "production"
	case "local":
		envPrefix = "loc"
	case "test":
		envPrefix = "test"
	default:
		// Default to development for any other value (development, staging, etc.)
		envPrefix = "dev"
		environment = "development"
	}

	// Auto-generate database name: {env-prefix}-{service-name}
	// Convert service name to be database-friendly (replace underscores/hyphens)
	dbServiceName := strings.ReplaceAll(serviceName, "_", "-")
	dbServiceName = strings.ReplaceAll(dbServiceName, "go-", "") // Remove go- prefix if present
	databaseName := fmt.Sprintf("%s-%s", envPrefix, dbServiceName)

	// Get MongoDB connection URI (includes all connection details including auth)
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}

	appName := serviceName

	return &DatabaseConfig{
		URI:          uri,
		Environment:  environment,
		DatabaseName: databaseName,
		AppName:      appName,
	}
}

// GetConnectionString returns the full MongoDB connection string with database name
func (c *DatabaseConfig) GetConnectionString() string {
	// If URI already contains database name (has path after port), return as is
	if strings.Contains(c.URI, "/") && !strings.HasSuffix(c.URI, "/") {
		// Check if there's a path after the port (e.g., mongodb://host:port/dbname)
		parts := strings.Split(c.URI, "/")
		if len(parts) > 3 && parts[3] != "" {
			return c.URI // Already has database name
		}
	}

	// Append database name to URI
	if strings.HasSuffix(c.URI, "/") {
		return c.URI + c.DatabaseName
	}
	return c.URI + "/" + c.DatabaseName
}

// MaskSensitiveData returns a copy of the config with sensitive data masked for logging
func (c *DatabaseConfig) MaskSensitiveData() *DatabaseConfig {
	masked := *c
	// Mask credentials in URI if present
	if strings.Contains(masked.URI, "@") {
		parts := strings.Split(masked.URI, "@")
		if len(parts) >= 2 {
			// Replace credentials part with ***:***
			credsPart := strings.Split(parts[0], "//")
			if len(credsPart) >= 2 {
				masked.URI = credsPart[0] + "//***:***@" + strings.Join(parts[1:], "@")
			}
		}
	}
	return &masked
} 