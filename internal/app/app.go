package app

import (
	"context"
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
	// Load credentials using secure method (environment variables preferred)
	creds, err := config.LoadCredentialsSecurely()
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
	modelSelector := selector.NewContextAwareSelector()
	apiHandlers := handlers.NewAPIHandlers(creds, models, apiClient, modelSelector)

	// Log configuration loaded with complete data
	logger.LogConfiguration(context.Background(), map[string]any{
		"credentials": creds,
		"models":      models,
		"config_summary": map[string]any{
			"credentials_count":   len(creds),
			"vendor_model_pairs":  len(models),
			"available_vendors":   getUniqueVendors(models),
			"available_platforms": getUniquePlatforms(creds),
		},
	})

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

// Helper functions for comprehensive logging
func getUniqueVendors(models []config.VendorModel) []string {
	vendorMap := make(map[string]bool)
	for _, model := range models {
		vendorMap[model.Vendor] = true
	}

	vendors := make([]string, 0, len(vendorMap))
	for vendor := range vendorMap {
		vendors = append(vendors, vendor)
	}
	return vendors
}

func getUniquePlatforms(credentials []config.Credential) []string {
	platformMap := make(map[string]bool)
	for _, cred := range credentials {
		platformMap[cred.Platform] = true
	}

	platforms := make([]string, 0, len(platformMap))
	for platform := range platformMap {
		platforms = append(platforms, platform)
	}
	return platforms
}
