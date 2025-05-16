package proxy

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/aashari/generative-api-router/internal/selector"
)

// APIClient handles communication with vendor APIs
type APIClient struct {
	BaseURLs map[string]string
}

// NewAPIClient creates a new API client with configured base URLs
func NewAPIClient() *APIClient {
	return &APIClient{
		BaseURLs: map[string]string{
			"openai": "https://api.openai.com/v1",
			"gemini": "https://generativelanguage.googleapis.com/v1beta/openai",
		},
	}
}

// SendRequest sends a request to the vendor API and streams the response back
func (c *APIClient) SendRequest(w http.ResponseWriter, r *http.Request, selection *selector.VendorSelection, modifiedBody []byte) error {
	baseURL, ok := c.BaseURLs[selection.Vendor]
	if !ok {
		return fmt.Errorf("unknown vendor: %s", selection.Vendor)
	}

	// All vendors use the same OpenAI-compatible endpoint
	fullURL := baseURL + "/chat/completions"

	// Create the proxied request
	req, err := http.NewRequest(r.Method, fullURL, bytes.NewReader(modifiedBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Copy request headers
	for k, vs := range r.Header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}

	// Set authorization header using Bearer token for all vendors
	req.Header.Set("Authorization", "Bearer "+selection.Credential.Value)

	// Send the request to the vendor
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to vendor: %v", err)
	}
	defer resp.Body.Close()

	// Copy response status code and headers
	w.WriteHeader(resp.StatusCode)
	for k, vs := range resp.Header {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}

	// Stream the response directly to the client
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("Error copying response: %v", err)
	}

	return nil
} 