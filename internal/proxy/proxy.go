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

	// Enrich context with vendor information
	ctx := context.WithValue(r.Context(), logger.VendorKey, selection.Vendor)
	ctx = context.WithValue(ctx, logger.ModelKey, selection.Model)
	r = r.WithContext(ctx)

	logger.DebugCtx(ctx, "Vendor and model selected",
		"selected_vendor", selection.Vendor,
		"selected_model", selection.Model,
	)

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := r.Body.Close(); err != nil {
		logger.WarnCtx(r.Context(), "Failed to close request body", "error", err)
	}

	// Validate and modify request
	modifiedBody, originalModel, err := validator.ValidateAndModifyRequest(body, selection.Model)
	if err != nil {
		logger.ErrorCtx(ctx, "Request validation failed", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Log the transparent proxy behavior
	logger.LogProxyRequest(ctx, originalModel, selection.Vendor, selection.Model, len(creds)*len(models))

	// Use the provided API client
	err = apiClient.SendRequest(w, r, selection, modifiedBody, originalModel)
	if err != nil {
		// Check for specific error types
		if errors.Is(err, ErrUnknownVendor) {
			logger.ErrorCtx(r.Context(), "Unknown vendor configuration error",
				"vendor", selection.Vendor,
				"error", err,
			)
			http.Error(w, "Internal configuration error: Unknown vendor", http.StatusBadRequest)
			return
		}

		// For other network errors
		logger.ErrorCtx(r.Context(), "Failed to communicate with upstream service",
			"vendor", selection.Vendor,
			"error", err,
		)
		http.Error(w, "Failed to communicate with upstream service: "+err.Error(), http.StatusBadGateway)
		return
	}
}
