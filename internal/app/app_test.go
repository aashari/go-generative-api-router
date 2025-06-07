package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/aashari/go-generative-api-router/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewApp_Success(t *testing.T) {
	// Initialize logger for tests
	if err := logger.Init(logger.DefaultConfig); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	// Create configs directory
	os.MkdirAll("configs", 0755)

	// Create temporary test files
	credsContent := `[
		{"platform": "openai", "type": "api-key", "value": "test-key"},
		{"platform": "gemini", "type": "api-key", "value": "test-key-2"}
	]`
	modelsContent := `[
		{"vendor": "openai", "model": "gpt-4"},
		{"vendor": "gemini", "model": "gemini-pro"}
	]`

	// Create temp credentials file
	credsFile, err := os.CreateTemp("", "credentials*.json")
	require.NoError(t, err)
	defer os.Remove(credsFile.Name())
	_, err = credsFile.WriteString(credsContent)
	require.NoError(t, err)
	credsFile.Close()

	// Create temp models file
	modelsFile, err := os.CreateTemp("", "models*.json")
	require.NoError(t, err)
	defer os.Remove(modelsFile.Name())
	_, err = modelsFile.WriteString(modelsContent)
	require.NoError(t, err)
	modelsFile.Close()

	// Temporarily rename the files to match expected names
	originalCredsPath := "configs/credentials.json"
	originalModelsPath := "configs/models.json"

	// Backup original files if they exist
	credsBackup := false
	if _, err := os.Stat(originalCredsPath); err == nil {
		os.Rename(originalCredsPath, originalCredsPath+".bak")
		credsBackup = true
	}
	modelsBackup := false
	if _, err := os.Stat(originalModelsPath); err == nil {
		os.Rename(originalModelsPath, originalModelsPath+".bak")
		modelsBackup = true
	}

	// Copy temp files to expected locations
	os.Link(credsFile.Name(), originalCredsPath)
	os.Link(modelsFile.Name(), originalModelsPath)
	defer func() {
		os.Remove(originalCredsPath)
		os.Remove(originalModelsPath)
		if credsBackup {
			os.Rename(originalCredsPath+".bak", originalCredsPath)
		}
		if modelsBackup {
			os.Rename(originalModelsPath+".bak", originalModelsPath)
		}
	}()

	// Test NewApp
	app, err := NewApp()
	require.NoError(t, err)
	require.NotNil(t, app)

	assert.NotNil(t, app.APIClient)
	assert.NotNil(t, app.ModelSelector)
	assert.NotNil(t, app.APIHandlers)
	assert.Len(t, app.Credentials, 2)
	assert.Len(t, app.VendorModels, 2)
}

func TestNewApp_MissingCredentialsFile(t *testing.T) {
	// Initialize logger for tests
	if err := logger.Init(logger.DefaultConfig); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	// Create configs directory
	os.MkdirAll("configs", 0755)

	// Backup original files if they exist
	credsPath := "configs/credentials.json"
	modelsPath := "configs/models.json"
	credsBackup := false
	modelsBackup := false

	if _, err := os.Stat(credsPath); err == nil {
		os.Rename(credsPath, credsPath+".bak")
		credsBackup = true
	}
	if _, err := os.Stat(modelsPath); err == nil {
		os.Rename(modelsPath, modelsPath+".bak")
		modelsBackup = true
	}

	defer func() {
		if credsBackup {
			os.Rename(credsPath+".bak", credsPath)
		}
		if modelsBackup {
			os.Rename(modelsPath+".bak", modelsPath)
		}
	}()

	// Ensure both files don't exist
	os.Remove(credsPath)
	os.Remove(modelsPath)

	app, err := NewApp()
	assert.Error(t, err)
	assert.Nil(t, app)
	// With our new credential loading priority, it may hit models error first
	assert.True(t,
		strings.Contains(err.Error(), "failed to load credentials") ||
			strings.Contains(err.Error(), "failed to load vendor models"),
		"Expected error about credentials or models, got: %s", err.Error())
}

func TestNewApp_InvalidCredentialsJSON(t *testing.T) {
	// Initialize logger for tests
	if err := logger.Init(logger.DefaultConfig); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	// Create configs directory
	os.MkdirAll("configs", 0755)

	// Create invalid JSON file
	invalidContent := `{invalid json`

	// Backup original files if they exist
	credsPath := "configs/credentials.json"
	modelsPath := "configs/models.json"
	credsBackup := false
	modelsBackup := false

	if _, err := os.Stat(credsPath); err == nil {
		os.Rename(credsPath, credsPath+".bak")
		credsBackup = true
	}
	if _, err := os.Stat(modelsPath); err == nil {
		os.Rename(modelsPath, modelsPath+".bak")
		modelsBackup = true
	}

	defer func() {
		os.Remove(credsPath)
		if credsBackup {
			os.Rename(credsPath+".bak", credsPath)
		}
		if modelsBackup {
			os.Rename(modelsPath+".bak", modelsPath)
		}
	}()

	err := os.WriteFile(credsPath, []byte(invalidContent), 0644)
	require.NoError(t, err)

	app, err := NewApp()
	assert.Error(t, err)
	assert.Nil(t, app)
	// With our new credential loading priority, it may hit models error first
	assert.True(t,
		strings.Contains(err.Error(), "failed to load credentials") ||
			strings.Contains(err.Error(), "failed to load vendor models"),
		"Expected error about credentials or models, got: %s", err.Error())
}

func TestNewApp_MissingModelsFile(t *testing.T) {
	// Initialize logger for tests
	if err := logger.Init(logger.DefaultConfig); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	// Create configs directory
	os.MkdirAll("configs", 0755)

	// Create valid credentials file
	credsContent := `[{"platform": "openai", "type": "api-key", "value": "test-key"}]`

	// Backup original files
	credsPath := "configs/credentials.json"
	modelsPath := "configs/models.json"
	credsBackup := false
	modelsBackup := false

	if _, err := os.Stat(credsPath); err == nil {
		os.Rename(credsPath, credsPath+".bak")
		credsBackup = true
	}
	if _, err := os.Stat(modelsPath); err == nil {
		os.Rename(modelsPath, modelsPath+".bak")
		modelsBackup = true
	}

	defer func() {
		os.Remove(credsPath)
		if credsBackup {
			os.Rename(credsPath+".bak", credsPath)
		}
		if modelsBackup {
			os.Rename(modelsPath+".bak", modelsPath)
		}
	}()

	// Create credentials file
	err := os.WriteFile(credsPath, []byte(credsContent), 0644)
	require.NoError(t, err)

	// Ensure models.json doesn't exist
	os.Remove(modelsPath)

	app, err := NewApp()
	assert.Error(t, err)
	assert.Nil(t, app)
	assert.Contains(t, err.Error(), "failed to load vendor models")
}

func TestNewApp_ValidationError(t *testing.T) {
	// Initialize logger for tests
	if err := logger.Init(logger.DefaultConfig); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	// Create configs directory
	os.MkdirAll("configs", 0755)

	// Create credentials and models that will fail validation
	// (models reference vendors without credentials)
	credsContent := `[{"platform": "openai", "type": "api-key", "value": "test-key"}]`
	modelsContent := `[
		{"vendor": "openai", "model": "gpt-4"},
		{"vendor": "anthropic", "model": "claude-3"}
	]`

	// Backup and create test files
	credsPath := "configs/credentials.json"
	modelsPath := "configs/models.json"
	credsBackup := false
	modelsBackup := false

	if _, err := os.Stat(credsPath); err == nil {
		os.Rename(credsPath, credsPath+".bak")
		credsBackup = true
	}
	if _, err := os.Stat(modelsPath); err == nil {
		os.Rename(modelsPath, modelsPath+".bak")
		modelsBackup = true
	}

	defer func() {
		os.Remove(credsPath)
		os.Remove(modelsPath)
		if credsBackup {
			os.Rename(credsPath+".bak", credsPath)
		}
		if modelsBackup {
			os.Rename(modelsPath+".bak", modelsPath)
		}
	}()

	err := os.WriteFile(credsPath, []byte(credsContent), 0644)
	require.NoError(t, err)
	err = os.WriteFile(modelsPath, []byte(modelsContent), 0644)
	require.NoError(t, err)

	app, err := NewApp()
	assert.Error(t, err)
	assert.Nil(t, app)
	assert.Contains(t, err.Error(), "configuration validation failed")
}

func TestApp_SetupRoutes(t *testing.T) {
	// Initialize logger for tests
	if err := logger.Init(logger.DefaultConfig); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	// Create configs directory
	os.MkdirAll("configs", 0755)

	// Create minimal valid config files
	credsContent := `[{"platform": "openai", "type": "api-key", "value": "test"}]`
	modelsContent := `[{"vendor": "openai", "model": "gpt-4"}]`

	// Setup test files
	credsPath := "configs/credentials.json"
	modelsPath := "configs/models.json"
	credsBackup := false
	modelsBackup := false

	if _, err := os.Stat(credsPath); err == nil {
		os.Rename(credsPath, credsPath+".bak")
		credsBackup = true
	}
	if _, err := os.Stat(modelsPath); err == nil {
		os.Rename(modelsPath, modelsPath+".bak")
		modelsBackup = true
	}

	defer func() {
		os.Remove(credsPath)
		os.Remove(modelsPath)
		if credsBackup {
			os.Rename(credsPath+".bak", credsPath)
		}
		if modelsBackup {
			os.Rename(modelsPath+".bak", modelsPath)
		}
	}()

	err := os.WriteFile(credsPath, []byte(credsContent), 0644)
	require.NoError(t, err)
	err = os.WriteFile(modelsPath, []byte(modelsContent), 0644)
	require.NoError(t, err)

	// Create app and test SetupRoutes
	app, err := NewApp()
	require.NoError(t, err)

	handler := app.SetupRoutes()
	assert.NotNil(t, handler)

	// Test that routes are properly configured by hitting health endpoint
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	// Verify it's valid JSON with expected structure
	body := w.Body.String()
	assert.Contains(t, body, "\"status\":")
	assert.Contains(t, body, "\"timestamp\":")
	assert.Contains(t, body, "\"services\":")
	assert.Contains(t, body, "\"details\":")
}

func TestNewApp_EmptyCredentials(t *testing.T) {
	// Initialize logger for tests
	if err := logger.Init(logger.DefaultConfig); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	// Create configs directory
	os.MkdirAll("configs", 0755)

	// Create empty credentials and models files
	credsContent := `[]`
	modelsContent := `[]`

	// Setup test files
	credsPath := "configs/credentials.json"
	modelsPath := "configs/models.json"
	credsBackup := false
	modelsBackup := false

	if _, err := os.Stat(credsPath); err == nil {
		os.Rename(credsPath, credsPath+".bak")
		credsBackup = true
	}
	if _, err := os.Stat(modelsPath); err == nil {
		os.Rename(modelsPath, modelsPath+".bak")
		modelsBackup = true
	}

	defer func() {
		os.Remove(credsPath)
		os.Remove(modelsPath)
		if credsBackup {
			os.Rename(credsPath+".bak", credsPath)
		}
		if modelsBackup {
			os.Rename(modelsPath+".bak", modelsPath)
		}
	}()

	err := os.WriteFile(credsPath, []byte(credsContent), 0644)
	require.NoError(t, err)
	err = os.WriteFile(modelsPath, []byte(modelsContent), 0644)
	require.NoError(t, err)

	app, err := NewApp()
	assert.Error(t, err)
	assert.Nil(t, app)
	assert.Contains(t, err.Error(), "configuration validation failed")
}
