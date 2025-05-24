package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/aashari/go-generative-api-router/internal/config"
	"github.com/aashari/go-generative-api-router/internal/errors"
	"github.com/aashari/go-generative-api-router/internal/filter"
	"github.com/aashari/go-generative-api-router/internal/logger"
	"github.com/aashari/go-generative-api-router/internal/proxy"
	"github.com/aashari/go-generative-api-router/internal/selector"
)

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
// @Description  Returns "OK" if the service is running properly
// @Tags         health
// @Accept       json
// @Produce      plain
// @Success      200  {string}  string  "OK"
// @Router       /health [get]
func (h *APIHandlers) HealthHandler(w http.ResponseWriter, r *http.Request) {
	logger.DebugCtx(r.Context(), "Health check endpoint accessed")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("OK")); err != nil {
		logger.WarnCtx(r.Context(), "Failed to write health check response", "error", err)
	}
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
func (h *APIHandlers) ChatCompletionsHandler(w http.ResponseWriter, r *http.Request) {
	logger.InfoCtx(r.Context(), "Chat completions request received", "remote_addr", r.RemoteAddr)

	// Optional vendor filter via query parameter
	vendorFilter := r.URL.Query().Get("vendor")

	// Filter credentials and models if vendor is specified
	creds := h.Credentials
	models := h.VendorModels
	if vendorFilter != "" {
		logger.DebugCtx(r.Context(), "Filtering by vendor", "vendor", vendorFilter)
		creds = filter.CredentialsByVendor(creds, vendorFilter)
		models = filter.ModelsByVendor(models, vendorFilter)

		// Check if we have credentials and models for this vendor
		if len(creds) == 0 {
			logger.WarnCtx(r.Context(), "No credentials available for vendor", "vendor", vendorFilter)
			err := errors.NewValidationError(fmt.Sprintf("No credentials available for vendor: %s", vendorFilter))
			errors.HandleError(w, err, http.StatusBadRequest)
			return
		}
		if len(models) == 0 {
			logger.WarnCtx(r.Context(), "No models available for vendor", "vendor", vendorFilter)
			err := errors.NewValidationError(fmt.Sprintf("No models available for vendor: %s", vendorFilter))
			errors.HandleError(w, err, http.StatusBadRequest)
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
// @Success      200     {object}  ModelsResponse "List of available models"
// @Router       /v1/models [get]
func (h *APIHandlers) ModelsHandler(w http.ResponseWriter, r *http.Request) {
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
	models := h.VendorModels
	if vendorFilter != "" {
		logger.DebugCtx(r.Context(), "Filtering models by vendor", "vendor", vendorFilter)
		models = filter.ModelsByVendor(models, vendorFilter)
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

	logger.DebugCtx(r.Context(), "Models list generated", "count", len(response.Data), "vendor_filter", vendorFilter)

	jsonResp, err := json.Marshal(response)
	if err != nil {
		logger.ErrorCtx(r.Context(), "Failed to marshal models response", "error", err)
		apiErr := errors.NewInternalError("Failed to generate model list")
		errors.HandleError(w, apiErr, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(jsonResp); err != nil {
		logger.WarnCtx(r.Context(), "Failed to write models response", "error", err)
	}
}
