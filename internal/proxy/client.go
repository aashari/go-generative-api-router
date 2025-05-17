package proxy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/aashari/generative-api-router/internal/selector"
)

// Error types for common API client errors
var (
	ErrUnknownVendor = errors.New("unknown vendor")
)

// APIClient handles communication with vendor APIs
type APIClient struct {
	BaseURLs map[string]string
	httpClient *http.Client
}

// NewAPIClient creates a new API client with configured base URLs
func NewAPIClient() *APIClient {
	httpClient := &http.Client{
		Timeout: 60 * time.Second,
	}

	return &APIClient{
		BaseURLs: map[string]string{
			"openai": "https://api.openai.com/v1",
			"gemini": "https://generativelanguage.googleapis.com/v1beta/openai",
		},
		httpClient: httpClient,
	}
}

// SendRequest sends a request to the vendor API and streams the response back
func (c *APIClient) SendRequest(w http.ResponseWriter, r *http.Request, selection *selector.VendorSelection, modifiedBody []byte) error {
	baseURL, ok := c.BaseURLs[selection.Vendor]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownVendor, selection.Vendor)
	}

	// Log the vendor and model being used
	isStreaming := false
	var requestData map[string]interface{}
	if err := json.Unmarshal(modifiedBody, &requestData); err == nil {
		if stream, ok := requestData["stream"].(bool); ok && stream {
			isStreaming = true
			log.Printf("Initiating streaming from vendor %s, model %s", selection.Vendor, selection.Model)
		}
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
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to vendor: %v", err)
	}
	defer resp.Body.Close()

	// Copy response headers before setting status code
	for k, vs := range resp.Header {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}

	// Set Transfer-Encoding explicitly if streaming
	if isStreaming || resp.Header.Get("Transfer-Encoding") == "chunked" {
		w.Header().Set("Transfer-Encoding", "chunked")
	}

	// Now write status code after headers
	w.WriteHeader(resp.StatusCode)

	// Stream the response directly to the client
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("Streaming error: %v", err)
	}

	return nil
} 