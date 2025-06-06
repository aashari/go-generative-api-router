package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/aashari/go-generative-api-router/internal/config"
	"github.com/aashari/go-generative-api-router/internal/errors"
	"github.com/aashari/go-generative-api-router/internal/filter"
	"github.com/aashari/go-generative-api-router/internal/logger"
	"github.com/aashari/go-generative-api-router/internal/proxy"
	"github.com/aashari/go-generative-api-router/internal/selector"
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
	w.Header().Set("Content-Type", "application/json")

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
		logger.LogError(r.Context(), "health_handler", err, map[string]any{
			"complete_request": map[string]any{
				"method":      r.Method,
				"path":        r.URL.Path,
				"headers":     map[string][]string(r.Header),
				"remote_addr": r.RemoteAddr,
				"user_agent":  r.Header.Get("User-Agent"),
			},
			"health_response": healthResponse,
			"marshal_error":   err.Error(),
		})
		// Fallback to simple error response
		http.Error(w, `{"status":"unhealthy","error":"failed to generate health response"}`, http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(jsonResponse); err != nil {
		logger.LogError(r.Context(), "health_handler", err, map[string]any{
			"complete_request": map[string]any{
				"method":      r.Method,
				"path":        r.URL.Path,
				"headers":     map[string][]string(r.Header),
				"remote_addr": r.RemoteAddr,
				"user_agent":  r.Header.Get("User-Agent"),
			},
			"health_response": healthResponse,
			"json_response":   string(jsonResponse),
			"write_error":     err.Error(),
		})
	}

	// Only log health checks when there are issues (not healthy status)
	// This reduces log noise from frequent health check monitoring
	if overallStatus != "healthy" {
		logger.LogWithStructure(r.Context(), logger.LevelError, "Health check failed",
			map[string]interface{}{
				"overall_status":    overallStatus,
				"services_status":   services,
				"version":           version,
				"uptime_seconds":    uptime,
				"credentials_count": len(h.Credentials),
				"models_count":      len(h.VendorModels),
				"api_client_status": h.APIClient != nil,
				"selector_status":   h.ModelSelector != nil,
			},
			nil, // request
			map[string]interface{}{
				"status_code":   statusCode,
				"response_body": string(jsonResponse),
				"content_type":  "application/json",
				"response_size": len(jsonResponse),
			},
			nil) // error
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
	// Log complete chat completions request data
	logger.LogWithStructure(r.Context(), logger.LevelInfo, "Chat completions request received with complete data",
		map[string]interface{}{
			"credentials_available": len(h.Credentials),
			"models_available":      len(h.VendorModels),
			"complete_credentials":  h.Credentials,
			"complete_models":       h.VendorModels,
		},
		map[string]interface{}{
			"method":         r.Method,
			"path":           r.URL.Path,
			"headers":        map[string][]string(r.Header),
			"query_params":   r.URL.Query(),
			"remote_addr":    r.RemoteAddr,
			"user_agent":     r.Header.Get("User-Agent"),
			"content_length": r.ContentLength,
			"host":           r.Host,
			"request_uri":    r.RequestURI,
		},
		nil, // response
		nil) // error

	// Optional vendor filter via query parameter
	vendorFilter := r.URL.Query().Get("vendor")

	// Filter credentials and models if vendor is specified
	creds := h.Credentials
	models := h.VendorModels
	if vendorFilter != "" {
		// Log complete filtering operation
		logger.LogWithStructure(r.Context(), logger.LevelDebug, "Filtering by vendor with complete data",
			map[string]interface{}{
				"vendor_filter":         vendorFilter,
				"original_credentials":  creds,
				"original_models":       models,
				"complete_query_params": r.URL.Query(),
			},
			nil, // request
			nil, // response
			nil) // error

		creds = filter.CredentialsByVendor(creds, vendorFilter)
		models = filter.ModelsByVendor(models, vendorFilter)

		// Log complete filtering results
		logger.LogWithStructure(r.Context(), logger.LevelDebug, "Vendor filtering results with complete data",
			map[string]interface{}{
				"vendor_filter":              vendorFilter,
				"filtered_credentials":       creds,
				"filtered_models":            models,
				"original_credentials_count": len(h.Credentials),
				"original_models_count":      len(h.VendorModels),
				"filtered_credentials_count": len(creds),
				"filtered_models_count":      len(models),
			},
			nil, // request
			nil, // response
			nil) // error

		// Check if we have credentials and models for this vendor
		if len(creds) == 0 {
			logger.LogError(r.Context(), "vendor_filtering",
				fmt.Errorf("no credentials available for vendor: %s", vendorFilter), map[string]any{
					"vendor_filter":                 vendorFilter,
					"complete_original_credentials": h.Credentials,
					"complete_filtered_credentials": creds,
					"complete_request_data": map[string]any{
						"method":       r.Method,
						"path":         r.URL.Path,
						"headers":      map[string][]string(r.Header),
						"query_params": r.URL.Query(),
					},
				})
			err := errors.NewValidationError(fmt.Sprintf("No credentials available for vendor: %s", vendorFilter))
			errors.HandleError(w, err, http.StatusBadRequest)
			return
		}
		if len(models) == 0 {
			logger.LogError(r.Context(), "vendor_filtering",
				fmt.Errorf("no models available for vendor: %s", vendorFilter), map[string]any{
					"vendor_filter":            vendorFilter,
					"complete_original_models": h.VendorModels,
					"complete_filtered_models": models,
					"complete_request_data": map[string]any{
						"method":       r.Method,
						"path":         r.URL.Path,
						"headers":      map[string][]string(r.Header),
						"query_params": r.URL.Query(),
					},
				})
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
		// Log complete models filtering operation
		logger.LogWithStructure(r.Context(), logger.LevelDebug, "Filtering models by vendor with complete data",
			map[string]interface{}{
				"vendor_filter":   vendorFilter,
				"original_models": models,
			},
			map[string]interface{}{
				"method":       r.Method,
				"path":         r.URL.Path,
				"headers":      map[string][]string(r.Header),
				"query_params": r.URL.Query(),
			},
			nil, // response
			nil) // error
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

	// Log complete models response generation
	logger.LogWithStructure(r.Context(), logger.LevelDebug, "Models list generated with complete data",
		map[string]interface{}{
			"complete_response": response,
			"vendor_filter":     vendorFilter,
			"original_models":   h.VendorModels,
			"filtered_models":   models,
			"response_count":    len(response.Data),
			"timestamp_used":    timestamp,
		},
		map[string]interface{}{
			"method":       r.Method,
			"path":         r.URL.Path,
			"headers":      map[string][]string(r.Header),
			"query_params": r.URL.Query(),
		},
		nil, // response
		nil) // error

	jsonResp, err := json.Marshal(response)
	if err != nil {
		logger.LogError(r.Context(), "models_handler", err, map[string]any{
			"complete_response_data": response,
			"vendor_filter":          vendorFilter,
			"complete_models":        models,
			"complete_request": map[string]any{
				"method":  r.Method,
				"path":    r.URL.Path,
				"headers": map[string][]string(r.Header),
			},
		})
		apiErr := errors.NewInternalError("Failed to generate model list")
		errors.HandleError(w, apiErr, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(jsonResp); err != nil {
		logger.LogError(r.Context(), "models_handler", err, map[string]any{
			"complete_response_data": response,
			"json_response":          string(jsonResp),
			"response_size":          len(jsonResp),
			"complete_request": map[string]any{
				"method":  r.Method,
				"path":    r.URL.Path,
				"headers": map[string][]string(r.Header),
			},
		})
	}
}

// Swagger type definitions (aliases to app package types for swagger generation)

// ChatCompletionRequest represents a request to the chat completions API
type ChatCompletionRequest struct {
	Messages []Message `json:"messages" example:"[]"`
	Model    string    `json:"model" example:"gpt-4o"`
	Stream   bool      `json:"stream,omitempty" example:"false"`
	// Added OpenAI-compatible fields
	MaxTokens        int                  `json:"max_tokens,omitempty" example:"100"`
	Temperature      float64              `json:"temperature,omitempty" example:"0.7"`
	TopP             float64              `json:"top_p,omitempty" example:"1"`
	N                int                  `json:"n,omitempty" example:"1"`
	Stop             []string             `json:"stop,omitempty"`
	PresencePenalty  float64              `json:"presence_penalty,omitempty" example:"0"`
	FrequencyPenalty float64              `json:"frequency_penalty,omitempty" example:"0"`
	LogitBias        map[string]float64   `json:"logit_bias,omitempty"`
	User             string               `json:"user,omitempty" example:"user-123"`
	Functions        []FunctionDefinition `json:"functions,omitempty"`
	FunctionCall     string               `json:"function_call,omitempty" example:"auto"`
	Tools            []Tool               `json:"tools,omitempty"`
	ToolChoice       string               `json:"tool_choice,omitempty" example:"auto"`
	ResponseFormat   map[string]string    `json:"response_format,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role       string     `json:"role" example:"user"`
	Content    string     `json:"content" example:"Hello, how are you?"`
	Name       string     `json:"name,omitempty" example:"John"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// FunctionDefinition represents an available function definition
type FunctionDefinition struct {
	Name        string                 `json:"name" example:"get_weather"`
	Description string                 `json:"description,omitempty" example:"Get the current weather in a location"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// Tool represents a tool available to the model
type Tool struct {
	Type     string                 `json:"type" example:"function"`
	Function map[string]interface{} `json:"function,omitempty"`
}

// ToolCall represents a call to a tool
type ToolCall struct {
	ID       string                 `json:"id" example:"call_8qty38"`
	Type     string                 `json:"type" example:"function"`
	Function map[string]interface{} `json:"function"`
}

// ChatCompletionResponse represents a response from the chat completions API
type ChatCompletionResponse struct {
	ID                string   `json:"id" example:"chatcmpl-abc123"`
	Object            string   `json:"object" example:"chat.completion"`
	Created           int64    `json:"created" example:"1677652288"`
	Model             string   `json:"model" example:"gpt-4o"`
	SystemFingerprint string   `json:"system_fingerprint,omitempty" example:"fp_abc123"`
	Choices           []Choice `json:"choices"`
	Usage             Usage    `json:"usage"`
	ServiceTier       string   `json:"service_tier,omitempty" example:"default"`
}

// Choice represents a completion choice
type Choice struct {
	Index        int     `json:"index" example:"0"`
	Message      Message `json:"message"`
	LogProbs     string  `json:"logprobs" example:"null"`
	FinishReason string  `json:"finish_reason" example:"stop"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens" example:"10"`
	CompletionTokens int `json:"completion_tokens" example:"20"`
	TotalTokens      int `json:"total_tokens" example:"30"`
	// Added additional usage details for OpenAI compatibility
	PromptTokensDetails     TokenDetails `json:"prompt_tokens_details"`
	CompletionTokensDetails TokenDetails `json:"completion_tokens_details"`
}

// TokenDetails represents detailed token usage information
type TokenDetails struct {
	CachedTokens             int `json:"cached_tokens" example:"0"`
	AudioTokens              int `json:"audio_tokens" example:"0"`
	ReasoningTokens          int `json:"reasoning_tokens,omitempty" example:"0"`
	AcceptedPredictionTokens int `json:"accepted_prediction_tokens,omitempty" example:"0"`
	RejectedPredictionTokens int `json:"rejected_prediction_tokens,omitempty" example:"0"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error ErrorInfo `json:"error"`
}

// ErrorInfo contains details about an error
type ErrorInfo struct {
	Message string `json:"message" example:"Invalid model specified"`
	Type    string `json:"type" example:"invalid_request_error"`
	Param   string `json:"param" example:"model"`
	Code    string `json:"code,omitempty" example:"invalid_model"`
}

// ModelsResponse represents the response from the models endpoint
type ModelsResponse struct {
	Object string  `json:"object" example:"list"`
	Data   []Model `json:"data"`
}

// Model represents a language model
type Model struct {
	ID      string `json:"id" example:"gpt-4o"`
	Object  string `json:"object" example:"model"`
	Created int64  `json:"created" example:"1677610602"`
	OwnedBy string `json:"owned_by" example:"openai"`
}
