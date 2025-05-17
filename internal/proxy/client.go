package proxy

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/aashari/go-generative-api-router/internal/selector"
)

// Error types for common API client errors
var (
	ErrUnknownVendor = errors.New("unknown vendor")
)

// APIClient handles communication with vendor APIs
type APIClient struct {
	BaseURLs   map[string]string
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

// generateRandomString generates a random hexadecimal string of specified length
func generateRandomString(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to the current timestamp if random generation fails
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// processResponse processes the API response, ensuring all required fields are present
func processResponse(responseBody []byte) ([]byte, error) {
	if len(responseBody) == 0 {
		return responseBody, nil
	}

	var responseData map[string]interface{}
	if err := json.Unmarshal(responseBody, &responseData); err != nil {
		return responseBody, nil // Return original response if it's not valid JSON
	}

	// Check and generate chat completion ID if missing
	if id, ok := responseData["id"]; !ok || id == nil || id == "" {
		responseData["id"] = "chatcmpl-" + generateRandomString(10)
	}

	// Process tool calls if present
	choices, ok := responseData["choices"].([]interface{})
	if ok && len(choices) > 0 {
		for i, choice := range choices {
			choiceMap, ok := choice.(map[string]interface{})
			if !ok {
				continue
			}

			message, ok := choiceMap["message"].(map[string]interface{})
			if !ok {
				continue
			}

			toolCalls, ok := message["tool_calls"].([]interface{})
			if !ok {
				continue
			}

			for j, toolCall := range toolCalls {
				toolCallMap, ok := toolCall.(map[string]interface{})
				if !ok {
					continue
				}

				// Check and generate tool call ID if missing
				if toolCallID, ok := toolCallMap["id"]; !ok || toolCallID == nil || toolCallID == "" {
					toolCallMap["id"] = "call_" + generateRandomString(16)
					toolCalls[j] = toolCallMap
				}
			}
			message["tool_calls"] = toolCalls
			choiceMap["message"] = message
			choices[i] = choiceMap
		}
		responseData["choices"] = choices
	}

	// Encode the modified response
	modifiedResponseBody, err := json.Marshal(responseData)
	if err != nil {
		return responseBody, nil // Return original response if marshal fails
	}

	return modifiedResponseBody, nil
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

	// Handle the response based on whether it's streaming or not
	if isStreaming {
		// For streaming responses, directly copy the response
		_, err = io.Copy(w, resp.Body)
		if err != nil {
			log.Printf("Streaming error: %v", err)
		}
	} else {
		// For non-streaming responses, read, process, and then write
		responseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading response: %v", err)
			return err
		}

		// Process the response to ensure IDs are present
		modifiedResponse, err := processResponse(responseBody)
		if err != nil {
			log.Printf("Error processing response: %v", err)
			w.Write(responseBody) // Write original response if processing fails
			return nil
		}

		// Write the modified response
		_, err = w.Write(modifiedResponse)
		if err != nil {
			log.Printf("Error writing response: %v", err)
		}
	}

	return nil
}
