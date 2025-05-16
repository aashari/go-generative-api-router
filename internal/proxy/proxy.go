package proxy

import (
	"io"
	"net/http"

	"github.com/aashari/generative-api-router/internal/config"
	"github.com/aashari/generative-api-router/internal/selector"
	"github.com/aashari/generative-api-router/internal/validator"
)

// ProxyRequest handles the incoming request, routes it to the appropriate vendor, and forwards the response
func ProxyRequest(w http.ResponseWriter, r *http.Request, creds []config.Credential, models []config.VendorModel) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Create a selector that determines which vendor and model to use
	sel := selector.NewRandomSelector()
	selection, err := sel.Select(creds, models)
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

	// Create and use API client
	client := NewAPIClient()
	err = client.SendRequest(w, r, selection, modifiedBody)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error()[:7] == "unknown" {
			statusCode = http.StatusBadRequest
		}
		http.Error(w, err.Error(), statusCode)
		return
	}
} 