package app

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/aashari/go-generative-api-router/docs" // This is necessary for Swagger documentation
	"github.com/aashari/go-generative-api-router/internal/config"
	"github.com/aashari/go-generative-api-router/internal/proxy"
	"github.com/aashari/go-generative-api-router/internal/selector"
	httpSwagger "github.com/swaggo/http-swagger" // http-swagger middleware
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

// Helper function to filter credentials by vendor
func filterCredentialsByVendor(creds []config.Credential, vendor string) []config.Credential {
	var result []config.Credential
	for _, c := range creds {
		if c.Platform == vendor {
			result = append(result, c)
		}
	}
	return result
}

// Helper function to filter models by vendor
func filterModelsByVendor(models []config.VendorModel, vendor string) []config.VendorModel {
	var result []config.VendorModel
	for _, m := range models {
		if m.Vendor == vendor {
			result = append(result, m)
		}
	}
	return result
}

// HealthHandler handles the health check endpoint
// @Summary      Health check endpoint
// @Description  Returns "OK" if the service is running properly
// @Tags         health
// @Accept       json
// @Produce      plain
// @Success      200  {string}  string  "OK"
// @Router       /health [get]
func (a *App) HealthHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Health check endpoint hit")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// ChatCompletionsHandler handles the chat completions endpoint
// @Summary      Chat completions API
// @Description  Routes chat completion requests to different language model providers
// @Tags         chat
// @Accept       json
// @Produce      json
// @Param        vendor  query     string                 false  "Optional vendor to target (e.g., 'openai', 'gemini')"
// @Param        request body      ChatCompletionRequest  true   "Chat completion request in OpenAI-compatible format"
// @Security     BearerAuth
// @Success      200     {object}  ChatCompletionResponse "OpenAI-compatible chat completion response"
// @Failure      400     {object}  ErrorResponse          "Bad request error"
// @Failure      401     {object}  ErrorResponse          "Unauthorized error"
// @Failure      500     {object}  ErrorResponse          "Internal server error"
// @Router       /v1/chat/completions [post]
func (a *App) ChatCompletionsHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request to /v1/chat/completions from %s", r.RemoteAddr)

	// Optional vendor filter via query parameter
	vendorFilter := r.URL.Query().Get("vendor")

	// Filter credentials and models if vendor is specified
	creds := a.Credentials
	models := a.VendorModels
	if vendorFilter != "" {
		log.Printf("Filtering by vendor: %s", vendorFilter)
		creds = filterCredentialsByVendor(creds, vendorFilter)
		models = filterModelsByVendor(models, vendorFilter)

		// Check if we have credentials and models for this vendor
		if len(creds) == 0 {
			http.Error(w, fmt.Sprintf("No credentials available for vendor: %s", vendorFilter), http.StatusBadRequest)
			return
		}
		if len(models) == 0 {
			http.Error(w, fmt.Sprintf("No models available for vendor: %s", vendorFilter), http.StatusBadRequest)
			return
		}
	}

	proxy.ProxyRequest(w, r, creds, models, a.APIClient, a.ModelSelector)
}

// ModelsHandler handles the models endpoint
// @Summary      List available models
// @Description  Returns a list of available language models in OpenAI-compatible format
// @Tags         models
// @Accept       json
// @Produce      json
// @Param        vendor  query     string         false  "Optional vendor to filter models (e.g., 'openai', 'gemini')"
// @Success      200     {object}  ModelsResponse "List of available models"
// @Router       /v1/models [get]
func (a *App) ModelsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	type Model struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	}

	var response struct {
		Object string  `json:"object"`
		Data   []Model `json:"data"`
	}

	// Optional vendor filter via query parameter
	vendorFilter := r.URL.Query().Get("vendor")
	models := a.VendorModels
	if vendorFilter != "" {
		log.Printf("Filtering models by vendor: %s", vendorFilter)
		models = filterModelsByVendor(models, vendorFilter)
	}

	response.Object = "list"
	timestamp := time.Now().Unix() // or a fixed timestamp if preferred

	for _, vm := range models {
		model := Model{
			ID:      vm.Model,
			Object:  "model",
			Created: timestamp,
			OwnedBy: vm.Vendor, // either "openai" or "gemini"
		}
		response.Data = append(response.Data, model)
	}

	jsonResp, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Failed to generate model list", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonResp)
}

// SetupRoutes configures all routes for the application
func (a *App) SetupRoutes() http.Handler {
	mux := http.NewServeMux()

	// Register API handlers
	mux.HandleFunc("/health", a.HealthHandler)
	mux.HandleFunc("/v1/chat/completions", a.ChatCompletionsHandler)
	mux.HandleFunc("/v1/models", a.ModelsHandler)

	// Serve Swagger UI with proper configuration
	mux.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"), // The URL pointing to API definition
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("swagger-ui"),
	))

	return mux
}
