package app

import (
	"fmt"
	"net/http"

	_ "github.com/aashari/go-generative-api-router/docs/api" // This is necessary for Swagger documentation
	"github.com/aashari/go-generative-api-router/internal/config"
	"github.com/aashari/go-generative-api-router/internal/handlers"
	"github.com/aashari/go-generative-api-router/internal/logger"
	"github.com/aashari/go-generative-api-router/internal/proxy"
	"github.com/aashari/go-generative-api-router/internal/router"
	"github.com/aashari/go-generative-api-router/internal/selector"
)

// App centralizes the application's dependencies and configuration
type App struct {
	Credentials   []config.Credential
	VendorModels  []config.VendorModel
	APIClient     *proxy.APIClient
	ModelSelector selector.Selector
	APIHandlers   *handlers.APIHandlers
}

// NewApp creates a new App instance with all dependencies
func NewApp() (*App, error) {
	// Load credentials
	creds, err := config.LoadCredentials("configs/credentials.json")
	if err != nil {
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}

	// Load vendor-model pairs
	models, err := config.LoadVendorModels("configs/models.json")
	if err != nil {
		return nil, fmt.Errorf("failed to load vendor models: %w", err)
	}

	// Validate configuration
	if validationErr := config.ValidateConfiguration(creds, models); validationErr != nil {
		return nil, fmt.Errorf("configuration validation failed: %s", validationErr.Error())
	}

	logger.Info("Configuration loaded and validated",
		"credentials_count", len(creds),
		"vendor_model_pairs", len(models),
	)

	// Initialize components
	apiClient := proxy.NewAPIClient()
	modelSelector := selector.NewEvenDistributionSelector()
	apiHandlers := handlers.NewAPIHandlers(creds, models, apiClient, modelSelector)

	return &App{
		Credentials:   creds,
		VendorModels:  models,
		APIClient:     apiClient,
		ModelSelector: modelSelector,
		APIHandlers:   apiHandlers,
	}, nil
}

// SetupRoutes configures all routes for the application
func (a *App) SetupRoutes() http.Handler {
	return router.SetupRoutes(a.APIHandlers)
}
