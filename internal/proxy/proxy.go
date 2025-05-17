package proxy

import (
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/aashari/go-generative-api-router/internal/config"
	"github.com/aashari/go-generative-api-router/internal/selector"
	"github.com/aashari/go-generative-api-router/internal/validator"
)

// ProxyRequest handles the incoming request, routes it to the appropriate vendor, and forwards the response
func ProxyRequest(w http.ResponseWriter, r *http.Request, creds []config.Credential, models []config.VendorModel, apiClient *APIClient, modelSelector selector.Selector) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Use the provided selector to determine which vendor and model to use
	selection, err := modelSelector.Select(creds, models)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	r.Body.Close()

	// Validate and modify request
	modifiedBody, err := validator.ValidateAndModifyRequest(body, selection.Model)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Use the provided API client
	err = apiClient.SendRequest(w, r, selection, modifiedBody)
	if err != nil {
		// Check for specific error types
		if errors.Is(err, ErrUnknownVendor) {
			log.Printf("Error: %v, for vendor: %s", err, selection.Vendor)
			http.Error(w, "Internal configuration error: Unknown vendor", http.StatusBadRequest)
			return
		}

		// For other network errors
		http.Error(w, "Failed to communicate with upstream service: "+err.Error(), http.StatusBadGateway)
		return
	}
}
