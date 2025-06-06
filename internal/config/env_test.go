package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvFile(t *testing.T) {
	// Create a temporary .env file for testing
	tempDir := t.TempDir()
	envFile := filepath.Join(tempDir, ".env")

	envContent := `TEST_VAR=test_value
TEST_NUMBER=42
TEST_BOOL=true
`

	err := os.WriteFile(envFile, []byte(envContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test .env file: %v", err)
	}

	// Load the .env file
	err = LoadEnvFile(envFile)
	if err != nil {
		t.Fatalf("Failed to load .env file: %v", err)
	}

	// Verify environment variables were loaded
	if os.Getenv("TEST_VAR") != "test_value" {
		t.Errorf("Expected TEST_VAR to be 'test_value', got '%s'", os.Getenv("TEST_VAR"))
	}

	if os.Getenv("TEST_NUMBER") != "42" {
		t.Errorf("Expected TEST_NUMBER to be '42', got '%s'", os.Getenv("TEST_NUMBER"))
	}

	if os.Getenv("TEST_BOOL") != "true" {
		t.Errorf("Expected TEST_BOOL to be 'true', got '%s'", os.Getenv("TEST_BOOL"))
	}

	// Clean up
	os.Unsetenv("TEST_VAR")
	os.Unsetenv("TEST_NUMBER")
	os.Unsetenv("TEST_BOOL")
}

func TestLoadEnvFileNotExists(t *testing.T) {
	// Try to load a non-existent .env file
	err := LoadEnvFile("non_existent.env")
	if err != nil {
		t.Errorf("Expected no error when .env file doesn't exist, got: %v", err)
	}
}

func TestGetEnvWithDefault(t *testing.T) {
	// Test with existing environment variable
	os.Setenv("TEST_EXISTING", "existing_value")
	defer os.Unsetenv("TEST_EXISTING")

	result := GetEnvWithDefault("TEST_EXISTING", "default_value")
	if result != "existing_value" {
		t.Errorf("Expected 'existing_value', got '%s'", result)
	}

	// Test with non-existing environment variable
	result = GetEnvWithDefault("TEST_NON_EXISTING", "default_value")
	if result != "default_value" {
		t.Errorf("Expected 'default_value', got '%s'", result)
	}
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		value    string
		expected bool
	}{
		{"true", true},
		{"TRUE", true},
		{"1", true},
		{"yes", true},
		{"YES", true},
		{"on", true},
		{"ON", true},
		{"false", false},
		{"FALSE", false},
		{"0", false},
		{"no", false},
		{"NO", false},
		{"off", false},
		{"OFF", false},
		{"invalid", true}, // Should return default
	}

	for _, test := range tests {
		os.Setenv("TEST_BOOL_VAR", test.value)
		result := GetEnvBool("TEST_BOOL_VAR", true)
		if result != test.expected {
			t.Errorf("For value '%s', expected %v, got %v", test.value, test.expected, result)
		}
		os.Unsetenv("TEST_BOOL_VAR")
	}

	// Test with non-existing variable
	result := GetEnvBool("TEST_NON_EXISTING_BOOL", false)
	if result != false {
		t.Errorf("Expected false for non-existing variable, got %v", result)
	}
}

func TestGetEnvInt(t *testing.T) {
	// Test with valid integer
	os.Setenv("TEST_INT", "123")
	defer os.Unsetenv("TEST_INT")

	result := GetEnvInt("TEST_INT", 456)
	if result != 123 {
		t.Errorf("Expected 123, got %d", result)
	}

	// Test with invalid integer
	os.Setenv("TEST_INVALID_INT", "not_a_number")
	defer os.Unsetenv("TEST_INVALID_INT")

	result = GetEnvInt("TEST_INVALID_INT", 456)
	if result != 456 {
		t.Errorf("Expected default value 456, got %d", result)
	}

	// Test with non-existing variable
	result = GetEnvInt("TEST_NON_EXISTING_INT", 789)
	if result != 789 {
		t.Errorf("Expected default value 789, got %d", result)
	}
}
