package proxy

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
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

// processStreamChunk processes a single chunk of a streaming response with consistent conversation-level values
func processStreamChunk(chunk []byte, vendor string, originalModel string, conversationID string, timestamp int64, systemFingerprint string) []byte {
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
		log.Printf("Error unmarshaling stream chunk: %v", err)
		return chunk // Return original chunk if it's not valid JSON
	}

	// Always use the consistent conversation ID (override vendor-provided ID for transparency)
	chunkData["id"] = conversationID

	// Always use the consistent timestamp (override vendor-provided timestamp for consistency)
	chunkData["created"] = timestamp

	// Add service_tier if missing (OpenAI compatibility)
	if _, ok := chunkData["service_tier"]; !ok {
		chunkData["service_tier"] = "default"
	}

	// Always use the consistent system fingerprint (override vendor-provided fingerprint for consistency)
	chunkData["system_fingerprint"] = systemFingerprint

	// Override the model field with the original model requested by the client
	if originalModel != "" {
		chunkData["model"] = originalModel
	}

	// Process choices if present
	if choices, ok := chunkData["choices"].([]interface{}); ok && len(choices) > 0 {
		log.Printf("Processing %d choices in stream chunk", len(choices))
		for i, choice := range choices {
			choiceMap, ok := choice.(map[string]interface{})
			if !ok {
				log.Printf("Stream chunk choice %d is not a map", i)
				continue
			}

			// Add logprobs if missing
			if _, ok := choiceMap["logprobs"]; !ok {
				choiceMap["logprobs"] = nil
			}

			// Check for delta (streaming) or message (first chunk)
			if delta, ok := choiceMap["delta"].(map[string]interface{}); ok {
				log.Printf("Processing delta in stream chunk choice %d", i)
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
					log.Printf("Processing %d tool calls in stream chunk delta", len(toolCalls))
					for j, toolCall := range toolCalls {
						toolCallMap, ok := toolCall.(map[string]interface{})
						if !ok {
							log.Printf("Stream chunk tool call %d is not a map", j)
							continue
						}

						// Check if "id" field exists and what its value is
						toolCallID, idExists := toolCallMap["id"]
						log.Printf("Tool call %d has ID: %v (exists: %v)", j, toolCallID, idExists)

						// Force override for Gemini vendor or if ID is missing/empty
						if vendor == "gemini" {
							// Always generate a new ID for Gemini responses regardless of current value
							newID := "call_" + generateRandomString(16)
							log.Printf("FORCING new tool call ID for Gemini vendor: %s (was: %v)", newID, toolCallID)
							toolCallMap["id"] = newID
							toolCalls[j] = toolCallMap
						} else if !idExists || toolCallID == nil || toolCallID == "" {
							// For other vendors, only generate if missing/empty
							newID := "call_" + generateRandomString(16)
							log.Printf("Generated new tool call ID for %s: %s", vendor, newID)
							toolCallMap["id"] = newID
							toolCalls[j] = toolCallMap
						}
					}
					delta["tool_calls"] = toolCalls
				} else {
					log.Printf("No tool calls found in stream chunk delta")
				}
				choiceMap["delta"] = delta
			} else if message, ok := choiceMap["message"].(map[string]interface{}); ok {
				log.Printf("Processing message in stream chunk choice %d", i)
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
					log.Printf("Processing %d tool calls in stream chunk message", len(toolCalls))
					for j, toolCall := range toolCalls {
						toolCallMap, ok := toolCall.(map[string]interface{})
						if !ok {
							log.Printf("Stream chunk message tool call %d is not a map", j)
							continue
						}

						// Check if "id" field exists and what its value is
						toolCallID, idExists := toolCallMap["id"]
						log.Printf("Tool call %d has ID: %v (exists: %v)", j, toolCallID, idExists)

						// Force override for Gemini vendor or if ID is missing/empty
						if vendor == "gemini" {
							// Always generate a new ID for Gemini responses regardless of current value
							newID := "call_" + generateRandomString(16)
							log.Printf("FORCING new tool call ID for Gemini vendor: %s (was: %v)", newID, toolCallID)
							toolCallMap["id"] = newID
							toolCalls[j] = toolCallMap
						} else if !idExists || toolCallID == nil || toolCallID == "" {
							// For other vendors, only generate if missing/empty
							newID := "call_" + generateRandomString(16)
							log.Printf("Generated new tool call ID for %s: %s", vendor, newID)
							toolCallMap["id"] = newID
							toolCalls[j] = toolCallMap
						}
					}
					message["tool_calls"] = toolCalls
				} else {
					log.Printf("No tool calls found in stream chunk message")
				}
				choiceMap["message"] = message
			} else {
				log.Printf("No delta or message found in stream chunk choice %d", i)
			}

			choices[i] = choiceMap
		}
		chunkData["choices"] = choices
	} else {
		log.Printf("No choices found in stream chunk")
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
				"prompt_tokens":     0,
				"completion_tokens": 0,
				"total_tokens":      0,
				"prompt_tokens_details": map[string]interface{}{
					"cached_tokens": 0,
					"audio_tokens":  0,
				},
				"completion_tokens_details": map[string]interface{}{
					"reasoning_tokens":           0,
					"audio_tokens":               0,
					"accepted_prediction_tokens": 0,
					"rejected_prediction_tokens": 0,
				},
			}
		}
	}

	// Encode the modified chunk
	modifiedData, err := json.Marshal(chunkData)
	if err != nil {
		log.Printf("Error marshaling modified stream chunk: %v", err)
		return chunk // Return original chunk if marshal fails
	}

	// Reconstruct the chunk with "data: " prefix
	return []byte("data: " + string(modifiedData) + "\n\n")
}

// processResponse processes the API response, ensuring all required fields are present
func processResponse(responseBody []byte, vendor string, contentEncoding string, originalModel string) ([]byte, error) {
	if len(responseBody) == 0 {
		return responseBody, nil
	}

	// Handle gzip content encoding
	if contentEncoding == "gzip" {
		log.Printf("Response is gzip encoded, decompressing...")
		gzipReader, err := gzip.NewReader(bytes.NewReader(responseBody))
		if err != nil {
			log.Printf("Error creating gzip reader: %v", err)
			return responseBody, fmt.Errorf("error creating gzip reader: %w", err) // Return error
		}
		defer gzipReader.Close()

		decompressedBody, err := io.ReadAll(gzipReader)
		if err != nil {
			log.Printf("Error decompressing gzip response: %v", err)
			return responseBody, fmt.Errorf("error decompressing gzip response: %w", err) // Return error
		}
		log.Printf("Successfully decompressed gzip response. Original size: %d, Decompressed size: %d", len(responseBody), len(decompressedBody))
		responseBody = decompressedBody // Use decompressed body for further processing
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
		log.Printf("Error unmarshaling response: %v", err)
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
	systemFingerprintValue, systemFingerprintExists := responseData["system_fingerprint"]
	if !systemFingerprintExists || systemFingerprintValue == nil {
		generatedFP := "fp_" + generateRandomString(9)
		responseData["system_fingerprint"] = generatedFP
		log.Printf("Generated system_fingerprint because it was missing or null. New value: %s", generatedFP)
	} else if _, isString := systemFingerprintValue.(string); !isString {
		// If it exists but is not a string (e.g. some other non-null, non-string type from a vendor)
		generatedFP := "fp_" + generateRandomString(9)
		responseData["system_fingerprint"] = generatedFP
		log.Printf("Replaced non-string system_fingerprint with generated one. New value: %s", generatedFP)
	}

	// Log the actual model used and replace it with the original model
	if model, ok := responseData["model"].(string); ok {
		log.Printf("Processing response from actual model: %s (vendor: %s), will be presented as: %s",
			model, vendor, originalModel)
	}

	// Override the model field with the original model requested by the client
	if originalModel != "" {
		responseData["model"] = originalModel
	}

	// Check if this is an error response
	if errorData, ok := responseData["error"].(map[string]interface{}); ok {
		// Process error fields only if this is an error response
		if code, ok := errorData["code"]; ok {
			// Convert the code to a string type if needed
			errorType := fmt.Sprintf("%v", code)
			errorData["type"] = errorType + "_error"
		} else {
			errorData["type"] = "api_error"
		}

		if _, ok := errorData["param"]; !ok {
			errorData["param"] = nil
		}

		responseData["error"] = errorData

		// Process choices and other fields only if this is not an error response
	} else if choices, ok := responseData["choices"].([]interface{}); ok && len(choices) > 0 {
		log.Printf("Processing %d choices", len(choices))
		for i, choice := range choices {
			choiceMap, ok := choice.(map[string]interface{})
			if !ok {
				log.Printf("Choice %d is not a map", i)
				continue
			}

			// Add logprobs if missing
			if _, ok := choiceMap["logprobs"]; !ok {
				choiceMap["logprobs"] = nil
			}

			// Process message if present
			if message, ok := choiceMap["message"].(map[string]interface{}); ok {
				log.Printf("Processing message in choice %d", i)
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
					log.Printf("Processing %d tool calls in choice %d", len(toolCalls), i)
					for j, toolCall := range toolCalls {
						toolCallMap, ok := toolCall.(map[string]interface{})
						if !ok {
							log.Printf("Tool call %d is not a map", j)
							continue
						}

						// Check if "id" field exists and what its value is
						toolCallID, idExists := toolCallMap["id"]
						log.Printf("Tool call %d has ID: %v (exists: %v)", j, toolCallID, idExists)

						// Force override for Gemini vendor or if ID is missing/empty
						if vendor == "gemini" {
							// Always generate a new ID for Gemini responses regardless of current value
							newID := "call_" + generateRandomString(16)
							log.Printf("FORCING new tool call ID for Gemini vendor: %s (was: %v)", newID, toolCallID)
							toolCallMap["id"] = newID
							toolCalls[j] = toolCallMap
						} else if !idExists || toolCallID == nil || toolCallID == "" {
							// For other vendors, only generate if missing/empty
							newID := "call_" + generateRandomString(16)
							log.Printf("Generated new tool call ID for %s: %s", vendor, newID)
							toolCallMap["id"] = newID
							toolCalls[j] = toolCallMap
						}
					}
					message["tool_calls"] = toolCalls
				} else {
					log.Printf("No tool calls found in message for choice %d", i)
				}

				choiceMap["message"] = message
			} else {
				log.Printf("No message found in choice %d", i)
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
				"audio_tokens":  0,
			}
		}

		if _, ok := usage["completion_tokens_details"]; !ok {
			usage["completion_tokens_details"] = map[string]interface{}{
				"reasoning_tokens":           0,
				"audio_tokens":               0,
				"accepted_prediction_tokens": 0,
				"rejected_prediction_tokens": 0,
			}
		}

		responseData["usage"] = usage
	} else {
		// If usage is completely missing, add a placeholder with default values
		responseData["usage"] = map[string]interface{}{
			"prompt_tokens":     0,
			"completion_tokens": 0,
			"total_tokens":      0,
			"prompt_tokens_details": map[string]interface{}{
				"cached_tokens": 0,
				"audio_tokens":  0,
			},
			"completion_tokens_details": map[string]interface{}{
				"reasoning_tokens":           0,
				"audio_tokens":               0,
				"accepted_prediction_tokens": 0,
				"rejected_prediction_tokens": 0,
			},
		}
	}

	// Encode the modified response
	modifiedResponseBody, err := json.Marshal(responseData)
	if err != nil {
		log.Printf("Error marshaling modified response: %v", err)
		return responseBody, nil // Return original response if marshal fails
	}

	return modifiedResponseBody, nil
}

// SendRequest sends a request to the vendor API and streams the response back
func (c *APIClient) SendRequest(w http.ResponseWriter, r *http.Request, selection *selector.VendorSelection, modifiedBody []byte, originalModel string) error {
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
			log.Printf("Initiating streaming from vendor %s, model %s, will be presented as %s",
				selection.Vendor, selection.Model, originalModel)
		}
	}

	// Generate consistent conversation-level values for streaming responses
	var conversationID string
	var timestamp int64
	var systemFingerprint string
	
	if isStreaming {
		conversationID = "chatcmpl-" + generateRandomString(10)
		timestamp = time.Now().Unix()
		systemFingerprint = "fp_" + generateRandomString(9)
		log.Printf("Generated consistent streaming values: ID=%s, timestamp=%d, fingerprint=%s", 
			conversationID, timestamp, systemFingerprint)
	}

	// All vendors use the same OpenAI-compatible endpoint
	// Do not change this
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

	// Store content encoding for later use in processResponse or stream handling
	contentEncoding := resp.Header.Get("Content-Encoding")

	// Set headers for streaming BEFORE copying vendor headers
	if isStreaming {
		// Set essential SSE headers first
		w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		// Don't explicitly set Transfer-Encoding, let Go handle it
	}

	// Whitelist approach: Only copy specific headers we want to pass through
	allowedHeaders := map[string]bool{
		"date":                      true,
		"server":                    true,
		"vary":                       true,
		"x-request-id":              true,
		"x-content-type-options":    true,
		"x-frame-options":           true,
		"x-xss-protection":          true,
		"access-control-allow-origin": true,
		"access-control-allow-methods": true,
		"access-control-allow-headers": true,
		"access-control-expose-headers": true,
		"server-timing":             true,
		"connection":                true,  // Added for keep-alive support
	}

	// Copy only whitelisted response headers
	for k, vs := range resp.Header {
		lowerK := strings.ToLower(k)
		
		// Skip these headers for streaming responses
		if isStreaming && (lowerK == "content-type" || lowerK == "content-length" || lowerK == "connection") {
			continue
		}
		
		// Skip Content-Length since we're modifying the body
		if lowerK == "content-length" {
			continue
		}
		
		// Skip Content-Encoding if we decompressed
		if contentEncoding == "gzip" && lowerK == "content-encoding" {
			continue
		}
		
		// Only copy whitelisted headers
		if allowedHeaders[lowerK] {
			for _, v := range vs {
				w.Header().Add(k, v)
			}
		}
	}

	// Ensure X-Request-ID header exists (generate if missing)
	if w.Header().Get("X-Request-ID") == "" {
		requestID := "req_" + generateRandomString(16)
		w.Header().Set("X-Request-ID", requestID)
		log.Printf("Generated X-Request-ID: %s", requestID)
	}

	// Ensure Access-Control-Expose-Headers includes X-Request-ID
	exposeHeaders := w.Header().Get("Access-Control-Expose-Headers")
	if exposeHeaders == "" {
		w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")
	} else if !strings.Contains(strings.ToLower(exposeHeaders), "x-request-id") {
		w.Header().Set("Access-Control-Expose-Headers", exposeHeaders + ", X-Request-ID")
	}

	// Now write status code after headers
	w.WriteHeader(resp.StatusCode)

	// Get flusher for streaming responses
	var flusher http.Flusher
	if isStreaming {
		if f, ok := w.(http.Flusher); ok {
			flusher = f
		} else {
			log.Printf("Warning: ResponseWriter does not support flushing")
		}
	}

	// Handle the response based on whether it's streaming or not
	if isStreaming {
		log.Printf("VERBOSE_DEBUG: SendRequest - Streaming - Vendor passed for processing: '%s'", selection.Vendor)
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

			// Process data lines
			if bytes.HasPrefix(line, []byte("data: ")) {
				// Check if it's the [DONE] marker
				if bytes.Contains(line, []byte("[DONE]")) {
					// Write the [DONE] marker with proper SSE format
					w.Write([]byte("data: [DONE]\n\n"))
					if flusher != nil {
						flusher.Flush()
					}
					// Exit the loop after [DONE] to properly close the connection
					break
				}

				// Use our processStreamChunk function to handle all the modifications
				// This ensures consistency with our standalone function
				modifiedLine := processStreamChunk(line, selection.Vendor, originalModel, conversationID, timestamp, systemFingerprint)
				
				// Write the modified line which already includes proper SSE formatting
				w.Write(modifiedLine)
				
				// CRITICAL: Flush after each chunk
				if flusher != nil {
					flusher.Flush()
				}
				
				// Skip the empty line that follows a data line in SSE format
				// since processStreamChunk already adds the required newlines
				nextLine, err := reader.ReadBytes('\n')
				if err != nil && err != io.EOF {
					log.Printf("Error reading empty line after data: %v", err)
				}
				// If it's not an empty line, we need to process it
				if len(bytes.TrimSpace(nextLine)) > 0 {
					// Put it back by creating a new reader with the line prepended
					remaining, _ := io.ReadAll(reader)
					reader = bufio.NewReader(io.MultiReader(bytes.NewReader(nextLine), bytes.NewReader(remaining)))
				}
			} else if len(bytes.TrimSpace(line)) == 0 {
				// This is an empty line not following a data line, pass it through
				w.Write(line)
				if flusher != nil {
					flusher.Flush()
				}
			} else {
				// For non-data, non-empty lines, pass through unchanged
				w.Write(line)
				if flusher != nil {
					flusher.Flush()
				}
			}
		}
	} else {
		log.Printf("VERBOSE_DEBUG: SendRequest - Non-Streaming - Vendor passed for processing: '%s'", selection.Vendor)
		// For non-streaming responses, read, process, and then write
		responseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading response: %v", err)
			return err
		}

		// Process the response to ensure IDs are present
		modifiedResponse, err := processResponse(responseBody, selection.Vendor, contentEncoding, originalModel)
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
