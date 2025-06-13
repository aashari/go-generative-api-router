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
		logger.WarnCtx(r.Context(), "Failed to close request body", "error", err)
	}

	// Parse payload to extract original model and other context
	payloadContext, err := AnalyzePayload(body)
	var originalModel string

	if err != nil {
		// If parsing fails, set default
		originalModel = "any-model"
		logger.WarnCtx(r.Context(), "Failed to parse request payload for routing", "error", err)
	} else {
		originalModel = payloadContext.OriginalModel

		// Log payload context for future routing decisions
		logger.DebugCtx(r.Context(), "Payload analyzed for routing",
			"original_model", payloadContext.OriginalModel,
			"has_stream", payloadContext.HasStream,
			"has_tools", payloadContext.HasTools,
			"has_images", payloadContext.HasImages,
			"has_videos", payloadContext.HasVideos,
			"messages_count", payloadContext.MessagesCount,
		)
	}

	// Use context-aware selection if available
	var selection *selector.VendorSelection

	// Check if the selector supports context-aware selection
	if contextSelector, ok := modelSelector.(*selector.ContextAwareSelector); ok && payloadContext != nil {
		// Use context-aware selection
		selection, err = contextSelector.SelectWithContext(creds, models, payloadContext)
		if err != nil {
			logger.ErrorCtx(r.Context(), "Context-aware vendor selection failed", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		logger.DebugCtx(r.Context(), "Context-aware selection used",
			"selected_vendor", selection.Vendor,
			"selected_model", selection.Model,
			"context_filters", map[string]bool{
				"images": payloadContext.HasImages,
				"videos": payloadContext.HasVideos,
				"tools":  payloadContext.HasTools,
				"stream": payloadContext.HasStream,
			},
		)
	} else {
		// Fall back to regular selection
		selection, err = modelSelector.Select(creds, models)
		if err != nil {
			logger.ErrorCtx(r.Context(), "Vendor selection failed", "error", err)
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

	// Enrich context with vendor information
	ctx := context.WithValue(r.Context(), logger.VendorKey, selection.Vendor)
	ctx = context.WithValue(ctx, logger.ModelKey, selection.Model)
	r = r.WithContext(ctx)

	logger.DebugCtx(ctx, "Vendor and model selected",
		"selected_vendor", selection.Vendor,
		"selected_model", selection.Model,
	)

	// Log complete request data
	logger.LogRequest(ctx, r.Method, r.URL.Path, r.Header.Get("User-Agent"),
		map[string][]string(r.Header), body)

	// Process image URLs if present (convert public URLs to base64)
	imageProcessor := NewImageProcessor()
	processedBody, err := imageProcessor.ProcessRequestBody(ctx, body)
	if err != nil {
		logger.ErrorCtx(ctx, "Image processing failed", "error", err)
		http.Error(w, "Failed to process images: "+err.Error(), http.StatusBadRequest)
		return err
	}

	// Log if images were processed
	if len(processedBody) != len(body) {
		logger.LogWithStructure(ctx, logger.LevelInfo, "Request body modified after image processing",
			map[string]interface{}{
				"original_size":   len(body),
				"processed_size":  len(processedBody),
				"size_difference": len(processedBody) - len(body),
			},
			nil, // request
			nil, // response
			nil) // error
	}

	// Validate and modify request
	modifiedBody, detectedOriginalModel, err := validator.ValidateAndModifyRequest(processedBody, selection.Model)
	if err != nil {
		logger.ErrorCtx(ctx, "Request validation failed", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	// Use the passed original model (already extracted in ProxyRequest)
	// The detectedOriginalModel is kept for backward compatibility but not used
	_ = detectedOriginalModel

	// Log the complete proxy request with all data
	logger.LogProxyRequest(ctx, originalModel, selection.Vendor, selection.Model, len(creds)*len(models), map[string]any{
		"original_request_body":  string(body),
		"processed_request_body": string(processedBody),
		"modified_request_body":  string(modifiedBody),
		"request_headers":        r.Header,
		"selection_details": map[string]any{
			"vendor":                selection.Vendor,
			"model":                 selection.Model,
			"credentials_available": len(creds),
			"models_available":      len(models),
		},
		"validation_result": map[string]any{
			"original_model": originalModel,
			"target_model":   selection.Model,
		},
		"image_processing": map[string]any{
			"body_modified":  len(processedBody) != len(body),
			"original_size":  len(body),
			"processed_size": len(processedBody),
		},
	})

	// Create retry executor with default configuration
	retryExecutor := reliability.NewRetryExecutor(nil) // Uses default config

	// Execute the API request with retry logic
	err = retryExecutor.ExecuteWithRetry(ctx, func() error {
		return apiClient.SendRequest(w, r, selection, modifiedBody, originalModel)
	})

	if err != nil {
		// Check if this is a retriable validation error (vendor fallback)
		if IsRetriableValidationError(err) {
			logger.LogWithStructure(ctx, logger.LevelWarn, "Vendor validation failed, attempting random fallback",
				map[string]interface{}{
					"original_vendor":   selection.Vendor,
					"original_model":    selection.Model,
					"error":             err.Error(),
					"fallback_strategy": "random_selection",
				},
				nil, // request
				nil, // response
				map[string]interface{}{
					"message": err.Error(),
					"type":    "vendor_validation_error",
				}) // error

			// Check if we have any credentials and models available for fallback
			if len(creds) == 0 || len(models) == 0 {
				logger.ErrorCtx(ctx, "No credentials or models available for fallback",
					"total_creds", len(creds),
					"total_models", len(models),
				)
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
				logger.ErrorCtx(ctx, "Failed to select fallback vendor/model", "error", retryErr)
				http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
				return err
			}

			logger.LogWithStructure(ctx, logger.LevelInfo, "Retrying with random fallback selection",
				map[string]interface{}{
					"fallback_vendor": fallbackSelection.Vendor,
					"fallback_model":  fallbackSelection.Model,
					"original_model":  originalModel,
					"original_vendor": selection.Vendor,
				},
				nil, // request
				nil, // response
				nil) // error

			// Create a fresh request for the retry (important for proper context)
			retryReq := r.Clone(r.Context())

			// Execute retry with the new selection (no further retries to avoid infinite loops)
			// Note: We don't call executeProxyRequestWithRetry to avoid infinite recursion
			// Instead, we directly call the API client with the new selection
			retryCtx := context.WithValue(retryReq.Context(), logger.VendorKey, fallbackSelection.Vendor)
			retryCtx = context.WithValue(retryCtx, logger.ModelKey, fallbackSelection.Model)
			retryReq = retryReq.WithContext(retryCtx)

			// Validate and modify request for the new vendor
			fallbackModifiedBody, _, validationErr := validator.ValidateAndModifyRequest(processedBody, fallbackSelection.Model)
			if validationErr != nil {
				logger.ErrorCtx(retryCtx, "Fallback request validation failed", "error", validationErr)
				http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
				return validationErr
			}

			// Execute the fallback request directly (no retry to avoid recursion)
			return apiClient.SendRequest(w, retryReq, fallbackSelection, fallbackModifiedBody, originalModel)
		}

		// Check if this is a retriable API error (quota, rate limits, server errors)
		if IsRetriableAPIError(err) {
			isQuotaError := IsQuotaError(err)
			logger.LogWithStructure(ctx, logger.LevelError, "Retriable API error after all retry attempts",
				map[string]interface{}{
					"vendor":     selection.Vendor,
					"error":      err.Error(),
					"error_type": "retriable_api_error_exhausted",
					"is_quota":   isQuotaError,
				},
				nil, // request
				nil, // response
				map[string]interface{}{
					"message": err.Error(),
					"type":    "api_error_retry_exhausted",
				}) // error

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
			logger.ErrorCtx(r.Context(), "Unknown vendor configuration error",
				"vendor", selection.Vendor,
				"error", err,
			)
			http.Error(w, "Internal configuration error: Unknown vendor", http.StatusBadRequest)
			return err
		}

		// For other network errors
		logger.ErrorCtx(r.Context(), "Failed to communicate with upstream service",
			"vendor", selection.Vendor,
			"error", err,
		)
		http.Error(w, "Failed to communicate with upstream service: "+err.Error(), http.StatusBadGateway)
		return err
	}

	return nil
}
