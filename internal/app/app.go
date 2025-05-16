package app

import (
	"fmt"
	"log"
	"net/http"

	"github.com/aashari/generative-api-router/internal/config"
	"github.com/aashari/generative-api-router/internal/proxy"
	"github.com/aashari/generative-api-router/internal/selector"
)

// App centralizes the application's dependencies and configuration
type App struct {
	Credentials   []config.Credential
	VendorModels  []config.VendorModel
	APIClient     *proxy.APIClient
	ModelSelector selector.Selector
}

// NewApp creates a new App instance with all dependencies
func NewApp() (*App, error) {
	// Load credentials
	creds, err := config.LoadCredentials("credentials.json")
	if err != nil {
		return nil, err
	}

	if len(creds) == 0 {
		return nil, fmt.Errorf("no credentials found in configuration file")
	}

	// Load vendor-model pairs
	models, err := config.LoadVendorModels("models.json")
	if err != nil {
		return nil, err
	}

	if len(models) == 0 {
		return nil, fmt.Errorf("no vendor-model pairs found in models.json")
	}

	log.Printf("Loaded %d credentials and %d vendor-model pairs", len(creds), len(models))

	// Initialize components
	apiClient := proxy.NewAPIClient()
	modelSelector := selector.NewRandomSelector()

	return &App{
		Credentials:   creds,
		VendorModels:  models,
		APIClient:     apiClient,
		ModelSelector: modelSelector,
	}, nil
}

// HealthHandler handles the health check endpoint
func (a *App) HealthHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Health check endpoint hit")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// ChatCompletionsHandler handles the chat completions endpoint
func (a *App) ChatCompletionsHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request to /chat/completions from %s", r.RemoteAddr)
	proxy.ProxyRequest(w, r, a.Credentials, a.VendorModels, a.APIClient, a.ModelSelector)
} 