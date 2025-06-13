package database

import (
	"os"
	"testing"
)

func TestGetDatabaseConfig(t *testing.T) {
	// Test default configuration
	config := GetDatabaseConfig()

	if config.Environment != "development" {
		t.Errorf("Expected default environment to be 'development', got '%s'", config.Environment)
	}

	if config.DatabaseName == "" {
		t.Error("Expected database name to be generated")
	}

	if config.URI == "" {
		t.Error("Expected URI to have a default value")
	}

	// Test with environment variables
	os.Setenv("ENVIRONMENT", "production")
	os.Setenv("SERVICE_NAME", "test-service")
	os.Setenv("MONGODB_URI", "mongodb://test:27017")

	config = GetDatabaseConfig()

	if config.Environment != "production" {
		t.Errorf("Expected environment to be 'production', got '%s'", config.Environment)
	}

	if config.DatabaseName != "prod-test-service" {
		t.Errorf("Expected database name to be 'prod-test-service', got '%s'", config.DatabaseName)
	}

	if config.URI != "mongodb://test:27017" {
		t.Errorf("Expected URI to be 'mongodb://test:27017', got '%s'", config.URI)
	}

	// Clean up
	os.Unsetenv("ENVIRONMENT")
	os.Unsetenv("SERVICE_NAME")
	os.Unsetenv("MONGODB_URI")
}

func TestDatabaseNameGeneration(t *testing.T) {
	testCases := []struct {
		environment string
		serviceName string
		expectedDB  string
	}{
		{"development", "go-generative-api-router", "dev-generative-api-router"},
		{"production", "my-service", "prod-my-service"},
		{"local", "test_service", "loc-test-service"},
		{"test", "go-test-app", "test-test-app"},
		{"staging", "app", "dev-app"}, // staging defaults to dev
	}

	for _, tc := range testCases {
		os.Setenv("ENVIRONMENT", tc.environment)
		os.Setenv("SERVICE_NAME", tc.serviceName)

		config := GetDatabaseConfig()

		if config.DatabaseName != tc.expectedDB {
			t.Errorf("For env=%s, service=%s: expected DB name '%s', got '%s'",
				tc.environment, tc.serviceName, tc.expectedDB, config.DatabaseName)
		}

		os.Unsetenv("ENVIRONMENT")
		os.Unsetenv("SERVICE_NAME")
	}
}

func TestMaskSensitiveData(t *testing.T) {
	config := &DatabaseConfig{
		URI: "mongodb://user:password@localhost:27017/db",
	}

	masked := config.MaskSensitiveData()

	if !contains(masked.URI, "***:***@") {
		t.Errorf("Expected URI credentials to be masked, got '%s'", masked.URI)
	}

	// Test URI without credentials (should remain unchanged)
	configNoAuth := &DatabaseConfig{
		URI: "mongodb://localhost:27017/db",
	}

	maskedNoAuth := configNoAuth.MaskSensitiveData()
	if maskedNoAuth.URI != "mongodb://localhost:27017/db" {
		t.Errorf("Expected URI without credentials to remain unchanged, got '%s'", maskedNoAuth.URI)
	}
}

func TestGetConnectionString(t *testing.T) {
	config := &DatabaseConfig{
		URI:          "mongodb://localhost:27017",
		DatabaseName: "test-db",
	}

	connStr := config.GetConnectionString()
	expected := "mongodb://localhost:27017/test-db"

	if connStr != expected {
		t.Errorf("Expected connection string '%s', got '%s'", expected, connStr)
	}

	// Test with URI that already has database
	config.URI = "mongodb://localhost:27017/existing-db"
	connStr = config.GetConnectionString()

	if connStr != config.URI {
		t.Errorf("Expected connection string to remain unchanged when DB already in URI")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		len(s) > len(substr) && s[len(s)-len(substr):] == substr ||
		len(s) > len(substr) && s[len(s)-len(substr):] != substr && s[:len(substr)] != substr &&
			len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
