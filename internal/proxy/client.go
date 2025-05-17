package proxy

import (
	"bufio"
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

// processStreamChunk processes a single chunk of a streaming response
func processStreamChunk(chunk []byte) []byte {
	// Handle empty chunks or non-data chunks
	if len(chunk) == 0 || !bytes.HasPrefix(chunk, []byte("data: ")) {
		return chunk
	}

	// Skip "[DONE]" message
	if bytes.Contains(chunk, []byte("[DONE]")) {
		return chunk
	}

	// Extract the JSON portion from the chunk
	jsonData := chunk[6:] // Skip "data: " prefix
	
	var chunkData map[string]interface{}
	if err := json.Unmarshal(jsonData, &chunkData); err != nil {
		return chunk // Return original chunk if it's not valid JSON
	}

	// Add chat completion ID if missing
	if id, ok := chunkData["id"]; !ok || id == nil || id == "" {
		chunkData["id"] = "chatcmpl-" + generateRandomString(10)
	}
	
	// Add service_tier if missing (OpenAI compatibility)
	if _, ok := chunkData["service_tier"]; !ok {
		chunkData["service_tier"] = "default"
	}

	// Add system_fingerprint if missing (OpenAI compatibility)
	if _, ok := chunkData["system_fingerprint"]; !ok {
		chunkData["system_fingerprint"] = "fp_" + generateRandomString(9)
	}

	// Process choices if present
	if choices, ok := chunkData["choices"].([]interface{}); ok && len(choices) > 0 {
		for i, choice := range choices {
			choiceMap, ok := choice.(map[string]interface{})
			if !ok {
				continue
			}

			// Add logprobs if missing
			if _, ok := choiceMap["logprobs"]; !ok {
				choiceMap["logprobs"] = nil
			}

			// Check for delta (streaming) or message (first chunk)
			if delta, ok := choiceMap["delta"].(map[string]interface{}); ok {
				// Add annotations array if missing in delta
				if _, ok := delta["annotations"]; !ok {
					delta["annotations"] = []interface{}{}
				}
				
				// Add refusal if missing in delta
				if _, ok := delta["refusal"]; !ok {
					delta["refusal"] = nil
				}
			
				// Process tool_calls if present
				if toolCalls, ok := delta["tool_calls"].([]interface{}); ok && len(toolCalls) > 0 {
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
					delta["tool_calls"] = toolCalls
				}
				choiceMap["delta"] = delta
			} else if message, ok := choiceMap["message"].(map[string]interface{}); ok {
				// Add annotations array if missing
				if _, ok := message["annotations"]; !ok {
					message["annotations"] = []interface{}{}
				}
				
				// Add refusal if missing
				if _, ok := message["refusal"]; !ok {
					message["refusal"] = nil
				}
			
				// Process tool_calls if present
				if toolCalls, ok := message["tool_calls"].([]interface{}); ok && len(toolCalls) > 0 {
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
				}
				choiceMap["message"] = message
			}
			
			choices[i] = choiceMap
		}
		chunkData["choices"] = choices
	}
	
	// Add usage information for the first chunk if needed
	// First chunk is usually identified by delta containing role field
	isFirstChunk := false
	if choices, ok := chunkData["choices"].([]interface{}); ok && len(choices) > 0 {
		if choiceMap, ok := choices[0].(map[string]interface{}); ok {
			if delta, ok := choiceMap["delta"].(map[string]interface{}); ok {
				if _, ok := delta["role"]; ok {
					isFirstChunk = true
				}
			}
		}
	}
	
	if isFirstChunk {
		// Add usage if missing
		_, hasUsage := chunkData["usage"]
		if !hasUsage {
			chunkData["usage"] = map[string]interface{}{
				"prompt_tokens": 0,
				"completion_tokens": 0,
				"total_tokens": 0,
				"prompt_tokens_details": map[string]interface{}{
					"cached_tokens": 0,
					"audio_tokens": 0,
				},
				"completion_tokens_details": map[string]interface{}{
					"reasoning_tokens": 0,
					"audio_tokens": 0,
					"accepted_prediction_tokens": 0,
					"rejected_prediction_tokens": 0,
				},
			}
		}
	}

	// Encode the modified chunk
	modifiedData, err := json.Marshal(chunkData)
	if err != nil {
		return chunk // Return original chunk if marshal fails
	}

	// Reconstruct the chunk with "data: " prefix
	return []byte("data: " + string(modifiedData) + "\n\n")
}

// processResponse processes the API response, ensuring all required fields are present
func processResponse(responseBody []byte) ([]byte, error) {
	if len(responseBody) == 0 {
		return responseBody, nil
	}

	// Check if response is a single-element array (which happens with Gemini errors)
	if bytes.HasPrefix(bytes.TrimSpace(responseBody), []byte("[")) {
		var arrayData []interface{}
		if err := json.Unmarshal(responseBody, &arrayData); err == nil {
			// If it's a single element array, unwrap it to be consistent with OpenAI
			if len(arrayData) == 1 {
				// Convert the single element back to JSON
				unwrappedData, err := json.Marshal(arrayData[0])
				if err == nil {
					responseBody = unwrappedData
				}
			}
		}
	}

	var responseData map[string]interface{}
	if err := json.Unmarshal(responseBody, &responseData); err != nil {
		return responseBody, nil // Return original response if it's not valid JSON
	}

	// Check and generate chat completion ID if missing
	if id, ok := responseData["id"]; !ok || id == nil || id == "" {
		responseData["id"] = "chatcmpl-" + generateRandomString(10)
	}

	// Add service_tier if missing (OpenAI compatibility)
	if _, ok := responseData["service_tier"]; !ok {
		responseData["service_tier"] = "default"
	}

	// Add system_fingerprint if missing (OpenAI compatibility)
	if _, ok := responseData["system_fingerprint"]; !ok {
		responseData["system_fingerprint"] = "fp_" + generateRandomString(9)
	}

	// Check if this is an error response and ensure it has the OpenAI-compatible structure
	if errorData, hasError := responseData["error"].(map[string]interface{}); hasError {
		// Add usage field to error responses if missing
		if _, hasUsage := responseData["usage"]; !hasUsage {
			responseData["usage"] = map[string]interface{}{
				"prompt_tokens": 0,
				"completion_tokens": 0,
				"total_tokens": 0,
				"prompt_tokens_details": map[string]interface{}{
					"cached_tokens": 0,
					"audio_tokens": 0,
				},
				"completion_tokens_details": map[string]interface{}{
					"reasoning_tokens": 0,
					"audio_tokens": 0,
					"accepted_prediction_tokens": 0,
					"rejected_prediction_tokens": 0,
				},
			}
		}

		// Ensure OpenAI-compatible error fields
		if _, ok := errorData["type"]; !ok {
			if code, hasCode := errorData["code"]; hasCode {
				// Convert the code to a string type if needed
				errorType := fmt.Sprintf("%v", code)
				errorData["type"] = errorType + "_error"
			} else {
				errorData["type"] = "api_error"
			}
		}

		if _, ok := errorData["param"]; !ok {
			errorData["param"] = nil
		}

		responseData["error"] = errorData
		
		// Process choices and other fields only if this is not an error response
	} else if choices, ok := responseData["choices"].([]interface{}); ok && len(choices) > 0 {
		for i, choice := range choices {
			choiceMap, ok := choice.(map[string]interface{})
			if !ok {
				continue
			}
			
			// Add logprobs if missing
			if _, ok := choiceMap["logprobs"]; !ok {
				choiceMap["logprobs"] = nil
			}

			// Process message if present
			if message, ok := choiceMap["message"].(map[string]interface{}); ok {
				// Add annotations array if missing
				if _, ok := message["annotations"]; !ok {
					message["annotations"] = []interface{}{}
				}
				
				// Add refusal if missing
				if _, ok := message["refusal"]; !ok {
					message["refusal"] = nil
				}
				
				// Handle tool_calls if present
				if toolCalls, ok := message["tool_calls"].([]interface{}); ok && len(toolCalls) > 0 {
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
				}
				
				choiceMap["message"] = message
			}
			
			choices[i] = choiceMap
		}
		responseData["choices"] = choices
	}

	// Ensure usage field is present with all required subfields
	if usage, ok := responseData["usage"].(map[string]interface{}); ok {
		// Make sure all required usage fields are present
		if _, ok := usage["prompt_tokens"]; !ok {
			usage["prompt_tokens"] = 0
		}
		if _, ok := usage["completion_tokens"]; !ok {
			usage["completion_tokens"] = 0
		}
		if _, ok := usage["total_tokens"]; !ok {
			usage["total_tokens"] = 0
		}
		
		// Add token details subfields if not present (OpenAI compatibility)
		if _, ok := usage["prompt_tokens_details"]; !ok {
			usage["prompt_tokens_details"] = map[string]interface{}{
				"cached_tokens": 0,
				"audio_tokens": 0,
			}
		}
		
		if _, ok := usage["completion_tokens_details"]; !ok {
			usage["completion_tokens_details"] = map[string]interface{}{
				"reasoning_tokens": 0,
				"audio_tokens": 0,
				"accepted_prediction_tokens": 0,
				"rejected_prediction_tokens": 0,
			}
		}
		
		responseData["usage"] = usage
	} else {
		// If usage is completely missing, add a placeholder with default values
		responseData["usage"] = map[string]interface{}{
			"prompt_tokens": 0,
			"completion_tokens": 0,
			"total_tokens": 0,
			"prompt_tokens_details": map[string]interface{}{
				"cached_tokens": 0,
				"audio_tokens": 0,
			},
			"completion_tokens_details": map[string]interface{}{
				"reasoning_tokens": 0,
				"audio_tokens": 0,
				"accepted_prediction_tokens": 0,
				"rejected_prediction_tokens": 0,
			},
		}
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
		// Skip Content-Length header since we're modifying the body
		if k == "Content-Length" {
			continue
		}
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
		// For streaming responses, we need a special handling
		reader := bufio.NewReader(resp.Body)
		
		for {
			// Read a line up to \n
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err != io.EOF {
					log.Printf("Error reading stream: %v", err)
				}
				break
			}
			
			// Skip empty lines
			if len(bytes.TrimSpace(line)) == 0 {
				w.Write(line)
				continue
			}
			
			// Process data lines
			if bytes.HasPrefix(line, []byte("data: ")) {
				// Check if it's the [DONE] marker
				if bytes.Contains(line, []byte("[DONE]")) {
					w.Write(line)
					continue
				}
				
				// Use our processStreamChunk function to handle all the modifications
				// This ensures consistency with our standalone function
				modifiedLine := processStreamChunk(line)
				w.Write(modifiedLine)
			} else {
				// For non-data lines, pass through unchanged
				w.Write(line)
			}
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
