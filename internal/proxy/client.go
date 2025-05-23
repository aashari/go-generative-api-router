package proxy

import (
	"bufio"
	"bytes"
	"compress/gzip"
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

// SendRequest sends a request to the vendor API and streams the response back
func (c *APIClient) SendRequest(w http.ResponseWriter, r *http.Request, selection *selector.VendorSelection, modifiedBody []byte, originalModel string) error {
	// 1. Setup request
	req, isStreaming, err := c.setupRequest(r, selection, modifiedBody, originalModel)
	if err != nil {
		return err
	}

	// 2. Send request to vendor
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to vendor: %v", err)
	}
	defer resp.Body.Close()

	// 3. Setup response headers
	c.setupResponseHeaders(w, resp, isStreaming)

	// 4. Handle response based on streaming mode
	if isStreaming {
		return c.handleStreaming(w, resp, selection, originalModel)
	} else {
		return c.handleNonStreaming(w, resp, selection, originalModel)
	}
}

// setupRequest prepares the HTTP request for the vendor API
func (c *APIClient) setupRequest(r *http.Request, selection *selector.VendorSelection, modifiedBody []byte, originalModel string) (*http.Request, bool, error) {
	baseURL, ok := c.BaseURLs[selection.Vendor]
	if !ok {
		return nil, false, fmt.Errorf("%w: %s", ErrUnknownVendor, selection.Vendor)
	}

	// Check if this is a streaming request
	isStreaming := false
	var requestData map[string]interface{}
	if err := json.Unmarshal(modifiedBody, &requestData); err == nil {
		if stream, ok := requestData["stream"].(bool); ok && stream {
			isStreaming = true
			log.Printf("Initiating streaming from vendor %s, model %s, will be presented as %s",
				selection.Vendor, selection.Model, originalModel)
		}
	}

	// All vendors use the same OpenAI-compatible endpoint
	fullURL := baseURL + "/chat/completions"

	// Create the proxied request
	req, err := http.NewRequest(r.Method, fullURL, bytes.NewReader(modifiedBody))
	if err != nil {
		return nil, false, fmt.Errorf("failed to create request: %v", err)
	}

	// Copy request headers
	for k, vs := range r.Header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}

	// Set authorization header using Bearer token for all vendors
	req.Header.Set("Authorization", "Bearer "+selection.Credential.Value)

	return req, isStreaming, nil
}

// setupResponseHeaders configures response headers
func (c *APIClient) setupResponseHeaders(w http.ResponseWriter, resp *http.Response, isStreaming bool) {
	// Set headers for streaming BEFORE writing status code
	if isStreaming {
		// Set essential SSE headers
		w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
	}

	// Set X-Request-ID header
	requestID := RequestID()
	w.Header().Set("X-Request-ID", requestID)
	log.Printf("Generated X-Request-ID: %s", requestID)

	// Write status code after headers
	w.WriteHeader(resp.StatusCode)
}

// handleStreaming processes streaming responses
func (c *APIClient) handleStreaming(w http.ResponseWriter, resp *http.Response, selection *selector.VendorSelection, originalModel string) error {
	log.Printf("VERBOSE_DEBUG: SendRequest - Streaming - Vendor passed for processing: '%s'", selection.Vendor)

	// Generate consistent conversation-level values for streaming responses
	conversationID := ChatCompletionID()
	timestamp := time.Now().Unix()
	systemFingerprint := SystemFingerprint()
	log.Printf("Generated consistent streaming values: ID=%s, timestamp=%d, fingerprint=%s",
		conversationID, timestamp, systemFingerprint)

	// Create stream processor
	streamProcessor := NewStreamProcessor(conversationID, timestamp, systemFingerprint, selection.Vendor, originalModel)

	// Get content encoding for gzip handling
	contentEncoding := resp.Header.Get("Content-Encoding")
	if contentEncoding != "" {
		log.Printf("Response has Content-Encoding: %s for vendor: %s, streaming: true", contentEncoding, selection.Vendor)
	}

	// Create the appropriate reader based on content encoding
	var reader *bufio.Reader
	if contentEncoding == "gzip" {
		log.Printf("Streaming response is gzip encoded, creating gzip reader")
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			log.Printf("Error creating gzip reader for streaming: %v", err)
			return fmt.Errorf("error creating gzip reader for streaming: %w", err)
		}
		defer gzipReader.Close()
		reader = bufio.NewReader(gzipReader)
	} else {
		reader = bufio.NewReader(resp.Body)
	}

	// Get flusher for streaming responses
	var flusher http.Flusher
	if f, ok := w.(http.Flusher); ok {
		flusher = f
	} else {
		log.Printf("Warning: ResponseWriter does not support flushing")
	}

	// Process streaming response
	return c.processStreamingResponse(w, reader, streamProcessor, flusher)
}

// processStreamingResponse handles the actual streaming response processing
func (c *APIClient) processStreamingResponse(w http.ResponseWriter, reader *bufio.Reader, streamProcessor *StreamProcessor, flusher http.Flusher) error {
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

			// Use stream processor to handle all the modifications
			modifiedLine := streamProcessor.ProcessChunk(line)

			// Write the modified line which already includes proper SSE formatting
			w.Write(modifiedLine)

			// CRITICAL: Flush after each chunk
			if flusher != nil {
				flusher.Flush()
			}

			// Skip the empty line that follows a data line in SSE format
			// since ProcessChunk already adds the required newlines
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
	return nil
}

// handleNonStreaming processes non-streaming responses
func (c *APIClient) handleNonStreaming(w http.ResponseWriter, resp *http.Response, selection *selector.VendorSelection, originalModel string) error {
	log.Printf("VERBOSE_DEBUG: SendRequest - Non-Streaming - Vendor passed for processing: '%s'", selection.Vendor)

	// Read response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response: %v", err)
		return err
	}

	// Get content encoding for processing
	contentEncoding := resp.Header.Get("Content-Encoding")
	if contentEncoding != "" {
		log.Printf("Response has Content-Encoding: %s for vendor: %s, streaming: false", contentEncoding, selection.Vendor)
	}

	// Process the response using the response processor
	modifiedResponse, err := ProcessResponse(responseBody, selection.Vendor, contentEncoding, originalModel)
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

	return nil
} 