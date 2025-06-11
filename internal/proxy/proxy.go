package proxy

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/aashari/go-generative-api-router/internal/config"
	"github.com/aashari/go-generative-api-router/internal/logger"
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

	// Use the provided selector to determine which vendor and model to use
	selection, err := modelSelector.Select(creds, models)
	if err != nil {
		logger.ErrorCtx(r.Context(), "Vendor selection failed", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

	// Execute the proxy request with retry logic
	err = executeProxyRequestWithRetry(w, r, selection, body, creds, models, apiClient, modelSelector, "")
	if err != nil {
		// Error already handled in executeProxyRequestWithRetry
		return
	}
}

// executeProxyRequestWithRetry handles the actual proxy request with optional retry for Gemini validation errors
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

	// Use the detected original model if not already set (for retry scenarios)
	if originalModel == "" {
		originalModel = detectedOriginalModel
	}

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

	// Use the provided API client
	err = apiClient.SendRequest(w, r, selection, modifiedBody, originalModel)
	if err != nil {
		// Check if this is a retriable validation error
		if IsRetriableValidationError(err) {
			logger.LogWithStructure(ctx, logger.LevelWarn, "Gemini validation failed, attempting OpenAI fallback",
				map[string]interface{}{
					"original_vendor": selection.Vendor,
					"original_model":  selection.Model,
					"error":           err.Error(),
					"fallback_vendor": "openai",
				},
				nil, // request
				nil, // response
				map[string]interface{}{
					"message": err.Error(),
					"type":    "vendor_validation_error",
				}) // error

			// Filter credentials and models for OpenAI only
			openaiCreds := filterCredentialsByVendor(creds, "openai")
			openaiModels := filterModelsByVendor(models, "openai")

			if len(openaiCreds) == 0 || len(openaiModels) == 0 {
				logger.ErrorCtx(ctx, "No OpenAI credentials or models available for fallback",
					"openai_creds", len(openaiCreds),
					"openai_models", len(openaiModels),
				)
				http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
				return err
			}

			// Select an OpenAI model for retry
			openaiSelection, retryErr := modelSelector.Select(openaiCreds, openaiModels)
			if retryErr != nil {
				logger.ErrorCtx(ctx, "Failed to select OpenAI model for fallback", "error", retryErr)
				http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
				return err
			}

			logger.LogWithStructure(ctx, logger.LevelInfo, "Retrying with OpenAI fallback",
				map[string]interface{}{
					"fallback_vendor": openaiSelection.Vendor,
					"fallback_model":  openaiSelection.Model,
					"original_model":  originalModel,
				},
				nil, // request
				nil, // response
				nil) // error

			// Create a fresh request for the retry (important for proper context)
			retryReq := r.Clone(r.Context())

			// Execute retry with OpenAI (no further retries)
			return executeProxyRequestWithRetry(w, retryReq, openaiSelection, body, creds, models, apiClient, modelSelector, originalModel)
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

// filterCredentialsByVendor filters credentials by vendor platform
func filterCredentialsByVendor(creds []config.Credential, vendor string) []config.Credential {
	var result []config.Credential
	for _, c := range creds {
		if c.Platform == vendor {
			result = append(result, c)
		}
	}
	return result
}

// filterModelsByVendor filters models by vendor
func filterModelsByVendor(models []config.VendorModel, vendor string) []config.VendorModel {
	var result []config.VendorModel
	for _, m := range models {
		if m.Vendor == vendor {
			result = append(result, m)
		}
	}
	return result
}
