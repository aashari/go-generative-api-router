package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/aashari/go-generative-api-router/internal/config"
	"github.com/aashari/go-generative-api-router/internal/database"
	"github.com/aashari/go-generative-api-router/internal/errors"
	"github.com/aashari/go-generative-api-router/internal/filter"
	"github.com/aashari/go-generative-api-router/internal/logger"
	"github.com/aashari/go-generative-api-router/internal/proxy"
	"github.com/aashari/go-generative-api-router/internal/selector"
	"github.com/aashari/go-generative-api-router/internal/types"
	"github.com/aashari/go-generative-api-router/internal/utils"
)

// startTime tracks when the application started
var startTime = time.Now()

// HealthResponse represents the structured health check response
type HealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp string                 `json:"timestamp"`
	Services  map[string]string      `json:"services"`
	Details   map[string]interface{} `json:"details"`
}

// APIHandlers contains the dependencies needed for API handlers
type APIHandlers struct {
	Credentials   []config.Credential
	VendorModels  []config.VendorModel
	APIClient     *proxy.APIClient
	ModelSelector selector.Selector
}

// NewAPIHandlers creates a new APIHandlers instance
func NewAPIHandlers(creds []config.Credential, models []config.VendorModel, client *proxy.APIClient, selector selector.Selector) *APIHandlers {
	return &APIHandlers{
		Credentials:   creds,
		VendorModels:  models,
		APIClient:     client,
		ModelSelector: selector,
	}
}

// HealthHandler handles the health check endpoint
// @Summary      Health check endpoint
// @Description  Returns structured health information including status, services, and version details
// @Tags         health
// @Accept       json
// @Produce      json
// @Success      200  {object}  handlers.HealthResponse  "Structured health response"
// @Router       /health [get]
func (h *APIHandlers) HealthHandler(w http.ResponseWriter, r *http.Request) {
	// Skip logging for health checks to reduce log noise
	// Only errors will be logged if health check fails

	// Calculate uptime in seconds
	uptime := int64(time.Since(startTime).Seconds())

	// Get version from environment variable (CI/CD pipeline sets this to git commit hash)
	version := os.Getenv("VERSION")
	if version == "" {
		version = "unknown"
	}

	// Check service components status
	services := make(map[string]string)
	overallStatus := "healthy"

	// Check API client status
	if h.APIClient != nil {
		services["api"] = "up"
	} else {
		services["api"] = "down"
		overallStatus = "unhealthy"
	}

	// Check credentials availability
	if len(h.Credentials) > 0 {
		services["credentials"] = "up"
	} else {
		services["credentials"] = "down"
		overallStatus = "degraded" // Service can run but with limited functionality
	}

	// Check models availability
	if len(h.VendorModels) > 0 {
		services["models"] = "up"
	} else {
		services["models"] = "down"
		overallStatus = "degraded" // Service can run but with limited functionality
	}

	// Check model selector
	if h.ModelSelector != nil {
		services["selector"] = "up"
	} else {
		services["selector"] = "down"
		overallStatus = "unhealthy"
	}

	// Check database connectivity (MongoDB)
	dbConn, err := database.GetConnection()
	if err != nil {
		services["database"] = "down"
		if overallStatus == "healthy" {
			overallStatus = "degraded" // Database is optional for basic functionality
		}
	} else if err := dbConn.HealthCheck(); err != nil {
		services["database"] = "unhealthy"
		if overallStatus == "healthy" {
			overallStatus = "degraded"
		}
	} else {
		services["database"] = "up"
	}

	// Create structured health response
	healthResponse := HealthResponse{
		Status:    overallStatus,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Services:  services,
		Details: map[string]interface{}{
			"version": version,
			"uptime":  uptime,
		},
	}

	// Set content type to JSON
	w.Header().Set(utils.HeaderContentType, utils.ContentTypeJSON)

	// Determine HTTP status code based on overall health
	statusCode := http.StatusOK
	if overallStatus == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	} else if overallStatus == "degraded" {
		statusCode = http.StatusOK // Still return 200 for degraded but functional service
	}

	w.WriteHeader(statusCode)

	// Marshal and send JSON response
	jsonResponse, err := json.Marshal(healthResponse)
	if err != nil {
		ctx := logger.WithComponent(r.Context(), "HealthHandler")
		ctx = logger.WithStage(ctx, "ResponseMarshal")
		logger.Error(ctx, "Failed to marshal health response", err,
			"health_response", healthResponse,
			"status", overallStatus,
		)
		// Fallback to simple error response
		http.Error(w, `{"status":"unhealthy","error":"failed to generate health response"}`, http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(jsonResponse); err != nil {
		ctx := logger.WithComponent(r.Context(), "HealthHandler")
		ctx = logger.WithStage(ctx, "ResponseWrite")
		logger.Error(ctx, "Failed to write health response", err,
			"health_response", healthResponse,
			"response_size", len(jsonResponse),
		)
	}

	// Only log health checks when there are issues (not healthy status)
	// This reduces log noise from frequent health check monitoring
	if overallStatus != "healthy" {
		ctx := logger.WithComponent(r.Context(), "HealthHandler")
		ctx = logger.WithStage(ctx, "HealthCheck")
		logger.Warn(ctx, "Health check degraded or unhealthy",
			"overall_status", overallStatus,
			"services_status", services,
			"version", version,
			"uptime_seconds", uptime,
			"credentials_count", len(h.Credentials),
			"models_count", len(h.VendorModels),
		)
	}
}

// ChatCompletionsHandler handles the chat completions endpoint
// @Summary      Chat completions API
// @Description  Routes chat completion requests to different language model providers
// @Tags         chat
// @Accept       json
// @Produce      json
// @Param        vendor  query     string                 false  "Optional vendor to target (e.g., 'openai', 'gemini')"
// @Param        request body      types.ChatCompletionRequest  true   "Chat completion request in OpenAI-compatible format"
// @Security     BearerAuth
// @Success      200     {object}  types.ChatCompletionResponse "OpenAI-compatible chat completion response"
// @Failure      400     {object}  types.ErrorResponse          "Bad request error"
// @Failure      401     {object}  types.ErrorResponse          "Unauthorized error"
// @Failure      500     {object}  types.ErrorResponse          "Internal server error"
// @Router       /v1/chat/completions [post]
func (h *APIHandlers) ChatCompletionsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := logger.WithComponent(r.Context(), "ChatCompletionsHandler")
	ctx = logger.WithStage(ctx, "Request")

	// Log complete chat completions request data
	logger.Info(ctx, "Chat completions request received",
		"credentials_available", len(h.Credentials),
		"models_available", len(h.VendorModels),
		"method", r.Method,
		"path", r.URL.Path,
		"query_params", r.URL.Query(),
	)

	// Optional vendor filter via query parameter
	vendorFilter := r.URL.Query().Get("vendor")

	// Filter credentials and models if vendor is specified
	creds := h.Credentials
	models := h.VendorModels
	if vendorFilter != "" {
		// Log complete filtering operation
		logger.Debug(ctx, "Filtering by vendor",
			"vendor_filter", vendorFilter,
			"original_credentials_count", len(creds),
			"original_models_count", len(models),
		)

		creds = filter.CredentialsByVendor(creds, vendorFilter)
		models = filter.ModelsByVendor(models, vendorFilter)

		// Log complete filtering results
		logger.Debug(ctx, "Vendor filtering completed",
			"vendor_filter", vendorFilter,
			"filtered_credentials_count", len(creds),
			"filtered_models_count", len(models),
		)

		// Check if we have credentials and models for this vendor
		if len(creds) == 0 {
			err := fmt.Errorf("no credentials available for vendor: %s", vendorFilter)
			logger.Error(ctx, "Vendor filtering failed", err,
				"vendor_filter", vendorFilter,
			)
			validationErr := errors.NewValidationError(err.Error())
			errors.HandleError(w, validationErr, http.StatusBadRequest)
			return
		}
		if len(models) == 0 {
			err := fmt.Errorf("no models available for vendor: %s", vendorFilter)
			logger.Error(ctx, "Vendor filtering failed", err,
				"vendor_filter", vendorFilter,
			)
			validationErr := errors.NewValidationError(err.Error())
			errors.HandleError(w, validationErr, http.StatusBadRequest)
			return
		}
	}

	proxy.ProxyRequest(w, r, creds, models, h.APIClient, h.ModelSelector)
}

// ModelsHandler handles the models endpoint
// @Summary      List available models
// @Description  Returns a list of available language models in OpenAI-compatible format
// @Tags         models
// @Accept       json
// @Produce      json
// @Param        vendor  query     string         false  "Optional vendor to filter models (e.g., 'openai', 'gemini')"
// @Success      200     {object}  types.ModelsResponse "List of available models"
// @Router       /v1/models [get]
func (h *APIHandlers) ModelsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := logger.WithComponent(r.Context(), "ModelsHandler")
	ctx = logger.WithStage(ctx, "Request")

	w.Header().Set(utils.HeaderContentType, utils.ContentTypeJSON)

	var response types.ModelsResponse

	// Optional vendor filter via query parameter
	vendorFilter := r.URL.Query().Get("vendor")
	models := h.VendorModels
	if vendorFilter != "" {
		// Log complete models filtering operation
		logger.Debug(ctx, "Filtering models by vendor",
			"vendor_filter", vendorFilter,
			"original_models_count", len(models),
		)
		models = filter.ModelsByVendor(models, vendorFilter)
	}

	response.Object = "list"
	timestamp := time.Now().Unix() // or a fixed timestamp if preferred

	for _, vm := range models {
		model := types.Model{
			ID:      vm.Model,
			Object:  "model",
			Created: timestamp,
			OwnedBy: vm.Vendor, // either "openai" or "gemini"
		}
		response.Data = append(response.Data, model)
	}

	// Log complete models response generation
	logger.Debug(ctx, "Models list generated",
		"vendor_filter", vendorFilter,
		"response_count", len(response.Data),
		"timestamp_used", timestamp,
	)

	jsonResp, err := json.Marshal(response)
	if err != nil {
		logger.Error(ctx, "Failed to marshal models response", err,
			"vendor_filter", vendorFilter,
			"models_count", len(models),
		)
		apiErr := errors.NewInternalError("Failed to generate model list")
		errors.HandleError(w, apiErr, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(jsonResp); err != nil {
		logger.Error(ctx, "Failed to write models response", err,
			"response_size", len(jsonResp),
		)
	}
}

// ImageToTextHandler handles the image description endpoint
// @Summary      Describe image
// @Description  Generates a detailed text description of a single image
// @Tags         images
// @Accept       json
// @Produce      json
// @Param        vendor  query     string                     false  "Optional vendor to target (e.g., 'openai', 'gemini')"
// @Param        request body      types.ImageToTextRequest   true   "Image description request"
// @Security     BearerAuth
// @Success      200  {object}  types.ChatCompletionResponse "OpenAI-compatible chat completion response"
// @Failure      400  {object}  types.ErrorResponse          "Bad request error"
// @Failure      500  {object}  types.ErrorResponse          "Internal server error"
// @Router       /v1/images/text [post]
func (h *APIHandlers) ImageToTextHandler(w http.ResponseWriter, r *http.Request) {
	ctx := logger.WithComponent(r.Context(), "ImageToTextHandler")
	ctx = logger.WithStage(ctx, "Request")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var imgReq types.ImageToTextRequest
	if err := json.NewDecoder(r.Body).Decode(&imgReq); err != nil {
		logger.Error(ctx, "Failed to decode request", err)
		validationErr := errors.NewValidationError("invalid request format")
		errors.HandleError(w, validationErr, http.StatusBadRequest)
		return
	}

	if imgReq.Type != "image_url" || imgReq.ImageURL.URL == "" {
		validationErr := errors.NewValidationError("invalid image_url object")
		errors.HandleError(w, validationErr, http.StatusBadRequest)
		return
	}

	// Build chat completion payload with system instruction
	userContent := map[string]interface{}{
		"type":      "image_url",
		"image_url": map[string]interface{}{"url": imgReq.ImageURL.URL},
	}
	if len(imgReq.ImageURL.Headers) > 0 {
		userContent["image_url"].(map[string]interface{})["headers"] = imgReq.ImageURL.Headers
	}

	payload := map[string]interface{}{
		"model": utils.DefaultImageModel,
		"messages": []interface{}{
			map[string]interface{}{
				"role":    "system",
				"content": utils.ImageDescriptionPrompt,
			},
			map[string]interface{}{
				"role":    "user",
				"content": []interface{}{userContent},
			},
		},
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		logger.Error(ctx, "Failed to marshal payload", err)
		apiErr := errors.NewInternalError("failed to build request")
		errors.HandleError(w, apiErr, http.StatusInternalServerError)
		return
	}

	newReq := r.Clone(r.Context())
	newReq.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	newReq.ContentLength = int64(len(bodyBytes))

	vendorFilter := r.URL.Query().Get("vendor")
	creds := h.Credentials
	models := h.VendorModels
	if vendorFilter != "" {
		creds = filter.CredentialsByVendor(creds, vendorFilter)
		models = filter.ModelsByVendor(models, vendorFilter)

		if len(creds) == 0 || len(models) == 0 {
			validationErr := errors.NewValidationError("no credentials or models for vendor")
			errors.HandleError(w, validationErr, http.StatusBadRequest)
			return
		}
	}

	proxy.ProxyRequest(w, newReq, creds, models, h.APIClient, h.ModelSelector)
}
