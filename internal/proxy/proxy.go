package proxy

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/aashari/go-generative-api-router/internal/config"
	"github.com/aashari/go-generative-api-router/internal/logger"
	"github.com/aashari/go-generative-api-router/internal/reliability"
	"github.com/aashari/go-generative-api-router/internal/selector"
	"github.com/aashari/go-generative-api-router/internal/utils"
	"github.com/aashari/go-generative-api-router/internal/validator"
)

// APIClientInterface defines the interface for API clients
type APIClientInterface interface {
	SendRequest(w http.ResponseWriter, r *http.Request, selection *selector.VendorSelection, modifiedBody []byte, originalModel string) error
}

// ProxyRequest handles the incoming request, routes it to the appropriate vendor, and forwards the response
func ProxyRequest(w http.ResponseWriter, r *http.Request, creds []config.Credential, models []config.VendorModel, apiClient APIClientInterface, modelSelector selector.Selector) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read the request body once and reuse it
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := r.Body.Close(); err != nil {
		ctx := logger.WithComponent(r.Context(), "proxy")
		ctx = logger.WithStage(ctx, "request_handling")
		logger.Warn(ctx, "Failed to close request body", "error", err)
	}

	// Parse payload to extract original model and other context
	payloadContext, err := AnalyzePayload(body)
	var originalModel string

	if err != nil {
		// If parsing fails, set default
		originalModel = "any-model"
		ctx := logger.WithComponent(r.Context(), "proxy")
		ctx = logger.WithStage(ctx, "payload_analysis")
		logger.Warn(ctx, "Failed to parse request payload for routing", "error", err)
	} else {
		originalModel = payloadContext.OriginalModel

		// Log payload context for future routing decisions
		ctx := logger.WithComponent(r.Context(), "proxy")
		ctx = logger.WithStage(ctx, "payload_analysis")
		logger.Debug(ctx, "Payload analyzed for routing",
			"original_model", payloadContext.OriginalModel,
			"has_stream", payloadContext.HasStream,
			"has_tools", payloadContext.HasTools,
			"has_images", payloadContext.HasImages,
			"has_videos", payloadContext.HasVideos,
			"messages_count", payloadContext.MessagesCount)
	}

	// Use context-aware selection if available
	var selection *selector.VendorSelection

	// Check if the selector supports context-aware selection
	if contextSelector, ok := modelSelector.(*selector.ContextAwareSelector); ok && payloadContext != nil {
		// Use context-aware selection
		selection, err = contextSelector.SelectWithContext(creds, models, payloadContext)
		if err != nil {
			ctx := logger.WithComponent(r.Context(), "proxy")
			ctx = logger.WithStage(ctx, "vendor_selection")
			logger.Error(ctx, "Context-aware vendor selection failed", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ctx := logger.WithComponent(r.Context(), "proxy")
		ctx = logger.WithStage(ctx, "vendor_selection")
		logger.Debug(ctx, "Context-aware selection used",
			"selected_vendor", selection.Vendor,
			"selected_model", selection.Model,
			"context_filters", map[string]bool{
				"images": payloadContext.HasImages,
				"videos": payloadContext.HasVideos,
				"tools":  payloadContext.HasTools,
				"stream": payloadContext.HasStream,
			})
	} else {
		// Fall back to regular selection
		selection, err = modelSelector.Select(creds, models)
		if err != nil {
			ctx := logger.WithComponent(r.Context(), "proxy")
			ctx = logger.WithStage(ctx, "vendor_selection")
			logger.Error(ctx, "Vendor selection failed", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Execute the proxy request with retry logic
	// Pass the original model we extracted
	err = executeProxyRequestWithRetry(w, r, selection, body, creds, models, apiClient, modelSelector, originalModel)
	if err != nil {
		// Error already handled in executeProxyRequestWithRetry
		return
	}
}

// executeProxyRequestWithRetry handles the actual proxy request with comprehensive retry logic
func executeProxyRequestWithRetry(w http.ResponseWriter, r *http.Request, selection *selector.VendorSelection, body []byte,
	creds []config.Credential, models []config.VendorModel, apiClient APIClientInterface, modelSelector selector.Selector, originalModel string) error {

	// Enrich context with vendor information and models
	ctx := context.WithValue(r.Context(), "vendor", selection.Vendor)
	ctx = context.WithValue(ctx, "model", selection.Model)
	ctx = context.WithValue(ctx, "vendor_models", models)
	r = r.WithContext(ctx)

	ctx = logger.WithComponent(ctx, "proxy")
	ctx = logger.WithStage(ctx, "execution_setup")
	logger.Debug(ctx, "Vendor and model selected",
		"selected_vendor", selection.Vendor,
		"selected_model", selection.Model)

	// Log complete request data
	logger.Info(ctx, "Processing request",
		"method", r.Method,
		"path", r.URL.Path,
		"user_agent", r.Header.Get(utils.HeaderUserAgent),
		"headers", map[string][]string(r.Header),
		"body_length", len(body),
		"component", "Proxy",
		"stage", "RequestLogging",
	)

	// Process image URLs if present (convert public URLs to base64)
	imageProcessor := NewImageProcessor()
	processedBody, err := imageProcessor.ProcessRequestBody(ctx, body)
	if err != nil {
		ctx = logger.WithStage(ctx, "image_processing")
		logger.Error(ctx, "Image processing failed", err)
		http.Error(w, "Failed to process images: "+err.Error(), http.StatusBadRequest)
		return err
	}

	// Log if images were processed
	if len(processedBody) != len(body) {
		ctx = logger.WithStage(ctx, "image_processing")
		logger.Info(ctx, "Request body modified after image processing",
			"original_size", len(body),
			"processed_size", len(processedBody),
			"size_difference", len(processedBody)-len(body))
	}

	// Validate and modify request
	modifiedBody, _, err := validator.ValidateAndModifyRequest(processedBody, selection.Model)
	if err != nil {
		ctx = logger.WithStage(ctx, "request_validation")
		logger.Error(ctx, "Request validation failed", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	// Use the passed original model (already extracted in ProxyRequest)

	// Log the complete proxy request with all data including full objects
	var completeModelObject interface{}
	for _, model := range models {
		if model.Vendor == selection.Vendor && model.Model == selection.Model {
			completeModelObject = model
			break
		}
	}

	logger.Info(ctx, "Proxy request with complete objects",
		"original_model", originalModel,
		"vendor", selection.Vendor,
		"model", selection.Model,
		"total_combinations", len(creds)*len(models),
		"original_request_body", string(body),
		"processed_request_body", string(processedBody),
		"modified_request_body", string(modifiedBody),
		"request_headers", r.Header,
		"selection_details", map[string]any{
			"vendor":                selection.Vendor,
			"model":                 selection.Model,
			"credentials_available": len(creds),
			"models_available":      len(models),
		},
		"validation_result", map[string]any{
			"original_model": originalModel,
			"target_model":   selection.Model,
		},
		"image_processing", map[string]any{
			"body_modified":  len(processedBody) != len(body),
			"original_size":  len(body),
			"processed_size": len(processedBody),
		},
		"credential", selection.Credential,
		"complete_model_object", completeModelObject,
		"component", "Proxy",
		"stage", "RequestProcessing",
	)

	// Create retry executor with default configuration
	retryExecutor := reliability.NewRetryExecutor(nil) // Uses default config

	// Execute the API request with retry logic
	err = retryExecutor.ExecuteWithRetry(ctx, func() error {
		return apiClient.SendRequest(w, r, selection, modifiedBody, originalModel)
	})

	if err != nil {
		// Check if this is a retriable validation error (vendor fallback)
		if IsRetriableValidationError(err) {
			ctx = logger.WithStage(ctx, "vendor_fallback")
			logger.Warn(ctx, "Vendor validation failed, attempting random fallback",
				"original_vendor", selection.Vendor,
				"original_model", selection.Model,
				"error", err.Error(),
				"fallback_strategy", "random_selection")

			// Check if we have any credentials and models available for fallback
			if len(creds) == 0 || len(models) == 0 {
				logger.Error(ctx, "No credentials or models available for fallback", nil,
					"total_creds", len(creds),
					"total_models", len(models))
				http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
				return err
			}

			// Select a different vendor/model combination for retry
			var fallbackSelection *selector.VendorSelection
			var retryErr error

			// Try context-aware selection for retry if available
			if contextSelector, ok := modelSelector.(*selector.ContextAwareSelector); ok {
				// Re-parse the payload to get context
				payloadContext, _ := AnalyzePayload(body)
				if payloadContext != nil {
					fallbackSelection, retryErr = contextSelector.SelectWithContext(creds, models, payloadContext)
				} else {
					fallbackSelection, retryErr = modelSelector.Select(creds, models)
				}
			} else {
				fallbackSelection, retryErr = modelSelector.Select(creds, models)
			}

			if retryErr != nil {
				logger.Error(ctx, "Failed to select fallback vendor/model", retryErr)
				http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
				return err
			}

			logger.Info(ctx, "Retrying with random fallback selection",
				"fallback_vendor", fallbackSelection.Vendor,
				"fallback_model", fallbackSelection.Model,
				"original_model", originalModel,
				"original_vendor", selection.Vendor)

			// Create a fresh request for the retry (important for proper context)
			retryReq := r.Clone(r.Context())

			// Execute retry with the new selection (no further retries to avoid infinite loops)
			// Note: We don't call executeProxyRequestWithRetry to avoid infinite recursion
			// Instead, we directly call the API client with the new selection
			retryCtx := context.WithValue(retryReq.Context(), "vendor", fallbackSelection.Vendor)
			retryCtx = context.WithValue(retryCtx, "model", fallbackSelection.Model)
			retryReq = retryReq.WithContext(retryCtx)

			// Validate and modify request for the new vendor
			fallbackModifiedBody, _, validationErr := validator.ValidateAndModifyRequest(processedBody, fallbackSelection.Model)
			if validationErr != nil {
				retryCtx = logger.WithStage(retryCtx, "fallback_validation")
				logger.Error(retryCtx, "Fallback request validation failed", validationErr)
				http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
				return validationErr
			}

			// Execute the fallback request directly (no retry to avoid recursion)
			return apiClient.SendRequest(w, retryReq, fallbackSelection, fallbackModifiedBody, originalModel)
		}

		// Check if this is a retriable API error (quota, rate limits, server errors)
		if IsRetriableAPIError(err) {
			isQuotaError := IsQuotaError(err)
			ctx = logger.WithStage(ctx, "api_error_handling")
			logger.Error(ctx, "Retriable API error after all retry attempts", err,
				"vendor", selection.Vendor,
				"error_type", "retriable_api_error_exhausted",
				"is_quota", isQuotaError)

			// For quota or rate limit errors, return 429 status
			if isQuotaError {
				http.Error(w, "API quota or rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
			} else {
				http.Error(w, "Service temporarily unavailable after multiple retries.", http.StatusServiceUnavailable)
			}
			return err
		}

		// Check for specific error types
		if errors.Is(err, ErrUnknownVendor) {
			ctx = logger.WithStage(ctx, "configuration_error")
			logger.Error(ctx, "Unknown vendor configuration error", err,
				"vendor", selection.Vendor)
			http.Error(w, "Internal configuration error: Unknown vendor", http.StatusBadRequest)
			return err
		}

		// For other network errors
		ctx = logger.WithStage(ctx, "communication_error")
		logger.Error(ctx, "Failed to communicate with upstream service", err,
			"vendor", selection.Vendor)
		http.Error(w, "Failed to communicate with upstream service: "+err.Error(), http.StatusBadGateway)
		return err
	}

	return nil
}
