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

	// Process tool calls in the delta if present
	choices, ok := chunkData["choices"].([]interface{})
	if ok && len(choices) > 0 {
		for i, choice := range choices {
			choiceMap, ok := choice.(map[string]interface{})
			if !ok {
				continue
			}

			// First check for delta for streaming responses
			delta, hasDelta := choiceMap["delta"].(map[string]interface{})
			if hasDelta {
				toolCalls, hasToolCalls := delta["tool_calls"].([]interface{})
				if hasToolCalls && len(toolCalls) > 0 {
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
					choiceMap["delta"] = delta
				}
			} else {
				// Check message for non-streaming or first chunk responses
				message, hasMessage := choiceMap["message"].(map[string]interface{})
				if hasMessage {
					toolCalls, hasToolCalls := message["tool_calls"].([]interface{})
					if hasToolCalls && len(toolCalls) > 0 {
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
					}
				}
			}
			choices[i] = choiceMap
		}
		chunkData["choices"] = choices
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
				
				// Extract the JSON part
				jsonData := line[6:]
				var chunkData map[string]interface{}
				
				if err := json.Unmarshal(bytes.TrimSpace(jsonData), &chunkData); err != nil {
					// If not valid JSON, write as-is
					w.Write(line)
					continue
				}
				
				// Add IDs where needed
				
				// Check and generate chat completion ID if missing
				if id, ok := chunkData["id"]; !ok || id == nil || id == "" {
					chunkData["id"] = "chatcmpl-" + generateRandomString(10)
				}
				
				// Process tool calls in choices
				if choices, ok := chunkData["choices"].([]interface{}); ok {
					for i, choice := range choices {
						choiceMap, ok := choice.(map[string]interface{})
						if !ok {
							continue
						}
						
						// Check delta first (used in streaming)
						if delta, ok := choiceMap["delta"].(map[string]interface{}); ok {
							if toolCalls, ok := delta["tool_calls"].([]interface{}); ok {
								for j, toolCall := range toolCalls {
									toolCallMap, ok := toolCall.(map[string]interface{})
									if !ok {
										continue
									}
									
									// Generate ID if missing
									if toolCallID, ok := toolCallMap["id"]; !ok || toolCallID == nil || toolCallID == "" {
										toolCallMap["id"] = "call_" + generateRandomString(16)
										toolCalls[j] = toolCallMap
									}
								}
								delta["tool_calls"] = toolCalls
							}
							choiceMap["delta"] = delta
						}
						
						// Also check message (used in first chunk sometimes)
						if message, ok := choiceMap["message"].(map[string]interface{}); ok {
							if toolCalls, ok := message["tool_calls"].([]interface{}); ok {
								for j, toolCall := range toolCalls {
									toolCallMap, ok := toolCall.(map[string]interface{})
									if !ok {
										continue
									}
									
									// Generate ID if missing
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
				
				// Encode back to JSON
				modifiedJSON, err := json.Marshal(chunkData)
				if err != nil {
					// If error, send original
					w.Write(line)
					continue
				}
				
				// Write modified line
				w.Write([]byte("data: "))
				w.Write(modifiedJSON)
				w.Write([]byte("\n\n"))
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
