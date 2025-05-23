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
	"strconv"
	"strings"
	"time"

	"github.com/aashari/go-generative-api-router/internal/selector"
)

// Error types for common API client errors
var (
	ErrUnknownVendor   = errors.New("unknown vendor")
	ErrInvalidResponse = errors.New("invalid vendor response")
)

// ResponseStandardizer handles vendor response standardization
type ResponseStandardizer struct {
	enableGzip       bool
	enableValidation bool
	standardHeaders  map[string]string
}

// NewResponseStandardizer creates a new response standardizer
func NewResponseStandardizer() *ResponseStandardizer {
	return &ResponseStandardizer{
		enableGzip:       true,
		enableValidation: true,
		standardHeaders: map[string]string{
			"Cache-Control":          "no-cache, no-store, must-revalidate",
			"X-Content-Type-Options": "nosniff",
			"X-Frame-Options":        "DENY",
			"X-XSS-Protection":       "1; mode=block",
			"Referrer-Policy":        "strict-origin-when-cross-origin",
		},
	}
}

// APIClient handles communication with vendor APIs
type APIClient struct {
	BaseURLs     map[string]string
	httpClient   *http.Client
	standardizer *ResponseStandardizer
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
		httpClient:   httpClient,
		standardizer: NewResponseStandardizer(),
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

	// 3. Handle response based on streaming mode
	if isStreaming {
		// Setup headers for streaming and handle streaming response
		c.setupResponseHeadersWithVendor(w, resp, isStreaming, selection.Vendor)
		return c.handleStreaming(w, r, resp, selection, originalModel)
	} else {
		// For non-streaming, we need to process the response first to determine compression
		return c.handleNonStreamingWithHeaders(w, r, resp, selection, originalModel)
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

	// Copy request headers (excluding compression headers to avoid vendor compression)
	for k, vs := range r.Header {
		// Skip compression-related headers - we handle compression at our service level
		if strings.ToLower(k) == "accept-encoding" {
			continue
		}
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}

	// Set authorization header using Bearer token for all vendors
	req.Header.Set("Authorization", "Bearer "+selection.Credential.Value)

	return req, isStreaming, nil
}

// setupResponseHeadersWithVendor configures response headers with vendor info
func (c *APIClient) setupResponseHeadersWithVendor(w http.ResponseWriter, resp *http.Response, isStreaming bool, vendor string) {
	// Set base compliant headers (content-length will be set per chunk for streaming)
	c.standardizer.setCompliantHeaders(w, vendor, 0, false)

	// Override content type for streaming mode
	if isStreaming {
		// Set essential SSE headers - override JSON content type for streaming
		w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		// Remove Content-Length for streaming as it's chunked
		w.Header().Del("Content-Length")
		log.Printf("STREAMING HEADERS: Set SSE headers for vendor %s", vendor)
	}

	// Write status code after setting all headers
	w.WriteHeader(resp.StatusCode)
}

// AddCustomServiceHeader adds a custom header specific to the proxy service.
// This function can be used to add service identification or custom metadata headers.
func AddCustomServiceHeader(w http.ResponseWriter, key, value string) {
	w.Header().Set(key, value)
}

// handleStreaming processes streaming responses
func (c *APIClient) handleStreaming(w http.ResponseWriter, r *http.Request, resp *http.Response, selection *selector.VendorSelection, originalModel string) error {
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

// validateVendorResponse validates that the vendor response meets our standards
func (s *ResponseStandardizer) validateVendorResponse(body []byte, vendor string) error {
	if !s.enableValidation {
		return nil
	}

	// Basic JSON validation
	if !json.Valid(body) {
		log.Printf("VALIDATION ERROR: Invalid JSON from vendor %s", vendor)
		return fmt.Errorf("%w: invalid JSON from vendor %s", ErrInvalidResponse, vendor)
	}

	// Parse and validate structure
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		log.Printf("VALIDATION ERROR: Failed to parse JSON from vendor %s: %v", vendor, err)
		return fmt.Errorf("%w: failed to parse response from vendor %s", ErrInvalidResponse, vendor)
	}

	// Validate required fields based on OpenAI standard
	requiredFields := []string{"choices", "created", "id", "model", "object"}
	for _, field := range requiredFields {
		if _, exists := response[field]; !exists {
			log.Printf("VALIDATION ERROR: Missing required field '%s' from vendor %s", field, vendor)
			return fmt.Errorf("%w: missing required field '%s' from vendor %s", ErrInvalidResponse, field, vendor)
		}
	}

	log.Printf("VALIDATION SUCCESS: Response from vendor %s passed validation", vendor)
	return nil
}

// setCompliantHeaders sets all standard headers with complete HTTP compliance
func (s *ResponseStandardizer) setCompliantHeaders(w http.ResponseWriter, vendor string, contentLength int, isCompressed bool) {
	// SECURITY HEADERS - Always applied for security compliance
	for key, value := range s.standardHeaders {
		w.Header().Set(key, value)
	}

	// CORS HEADERS - Complete CORS compliance
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID, X-Response-Time")

	// CONTENT HEADERS - HTTP compliance
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(contentLength))

	// COMPRESSION HEADERS - Mandatory for all responses
	if isCompressed {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")
	}

	// SERVICE IDENTIFICATION
	w.Header().Set("X-Powered-By", "Generative-API-Router")
	w.Header().Set("X-Vendor-Source", vendor)
	w.Header().Set("X-Request-ID", RequestID())

	// STANDARD HTTP HEADERS - Generated by our service (NO vendor pass-through)
	w.Header().Set("Date", time.Now().UTC().Format(http.TimeFormat))
	w.Header().Set("Server", "Generative-API-Router/1.0")

	log.Printf("COMPLIANT HEADERS: Set standardized headers for vendor %s (content-length: %d, compressed: %t)",
		vendor, contentLength, isCompressed)
}

// processResponseBody handles gzip decompression and response processing
func (s *ResponseStandardizer) processResponseBody(body io.Reader, contentEncoding string, vendor string) ([]byte, error) {
	var reader io.Reader = body

	// Handle gzip decompression
	if strings.Contains(strings.ToLower(contentEncoding), "gzip") {
		log.Printf("GZIP PROCESSING: Decompressing gzip response from vendor %s", vendor)
		gzipReader, err := gzip.NewReader(body)
		if err != nil {
			log.Printf("GZIP ERROR: Failed to create gzip reader for vendor %s: %v", vendor, err)
			return nil, fmt.Errorf("gzip decompression error for vendor %s: %w", vendor, err)
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	// Read the response body
	responseBody, err := io.ReadAll(reader)
	if err != nil {
		log.Printf("BODY READ ERROR: Failed to read response from vendor %s: %v", vendor, err)
		return nil, fmt.Errorf("failed to read response body from vendor %s: %w", vendor, err)
	}

	log.Printf("BODY PROCESSING: Successfully processed %d bytes from vendor %s (gzip: %t)",
		len(responseBody), vendor, strings.Contains(strings.ToLower(contentEncoding), "gzip"))

	return responseBody, nil
}

// shouldCompress determines if we should compress based on client's Accept-Encoding header (HTTP compliance)
func (s *ResponseStandardizer) shouldCompress(r *http.Request) bool {
	if !s.enableGzip {
		return false
	}

	// Check User-Agent for clients that have compression display issues
	userAgent := strings.ToLower(r.Header.Get("User-Agent"))
	if strings.Contains(userAgent, "postman") || strings.Contains(userAgent, "insomnia") || strings.Contains(userAgent, "paw") {
		log.Printf("COMPRESSION CHECK: Detected client with display issues (%s), disabling compression", userAgent)
		return false
	}

	// Only compress if client explicitly requests it via Accept-Encoding header
	acceptEncoding := r.Header.Get("Accept-Encoding")
	clientWantsGzip := strings.Contains(strings.ToLower(acceptEncoding), "gzip")

	log.Printf("COMPRESSION CHECK: Client Accept-Encoding='%s', User-Agent='%s', will compress=%t",
		acceptEncoding, userAgent, clientWantsGzip)

	return clientWantsGzip
}

// compressResponseMandatory compresses the response when client requests it
func (s *ResponseStandardizer) compressResponseMandatory(body []byte) ([]byte, error) {
	// Compress the response (called only when client requests compression)
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)

	if _, err := gzipWriter.Write(body); err != nil {
		log.Printf("GZIP COMPRESSION ERROR: %v", err)
		return nil, fmt.Errorf("gzip compression error: %w", err)
	}

	if err := gzipWriter.Close(); err != nil {
		log.Printf("GZIP COMPRESSION CLOSE ERROR: %v", err)
		return nil, fmt.Errorf("gzip compression close error: %w", err)
	}

	compressionRatio := float64(len(body)-buf.Len()) / float64(len(body)) * 100
	log.Printf("GZIP COMPRESSION: Compressed %d bytes to %d bytes (%.1f%% reduction)",
		len(body), buf.Len(), compressionRatio)

	return buf.Bytes(), nil
}

// compressStreamingChunk compresses individual streaming chunks (mandatory compression)
func (s *ResponseStandardizer) compressStreamingChunk(body []byte) ([]byte, error) {
	// Always compress streaming chunks for consistency
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)

	if _, err := gzipWriter.Write(body); err != nil {
		log.Printf("STREAMING GZIP COMPRESSION ERROR: %v", err)
		return nil, fmt.Errorf("streaming gzip compression error: %w", err)
	}

	if err := gzipWriter.Close(); err != nil {
		log.Printf("STREAMING GZIP COMPRESSION CLOSE ERROR: %v", err)
		return nil, fmt.Errorf("streaming gzip compression close error: %w", err)
	}

	log.Printf("STREAMING GZIP: Compressed chunk from %d to %d bytes", len(body), buf.Len())

	return buf.Bytes(), nil
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

// handleNonStreamingWithHeaders processes non-streaming responses and sets headers correctly
func (c *APIClient) handleNonStreamingWithHeaders(w http.ResponseWriter, r *http.Request, resp *http.Response, selection *selector.VendorSelection, originalModel string) error {
	log.Printf("VERBOSE_DEBUG: SendRequest - Non-Streaming - Vendor passed for processing: '%s'", selection.Vendor)

	// Process response body with gzip support and validation
	contentEncoding := resp.Header.Get("Content-Encoding")
	processedBody, err := c.standardizer.processResponseBody(resp.Body, contentEncoding, selection.Vendor)
	if err != nil {
		log.Printf("Error processing response body from vendor %s: %v", selection.Vendor, err)
		return err
	}

	// Validate vendor response
	if err := c.standardizer.validateVendorResponse(processedBody, selection.Vendor); err != nil {
		log.Printf("Vendor response validation failed for %s: %v", selection.Vendor, err)
		// Continue processing even if validation fails, but log the issue
	}

	// Process the response using the existing response processor to handle model name substitution
	// Note: processedBody is already decompressed, so we pass empty contentEncoding
	modifiedResponse, err := ProcessResponse(processedBody, selection.Vendor, "", originalModel)
	if err != nil {
		log.Printf("Error processing response from vendor %s: %v", selection.Vendor, err)
		modifiedResponse = processedBody // Use original response if processing fails
	}

	// Determine compression based on client's Accept-Encoding header (HTTP compliance)
	shouldCompress := c.standardizer.shouldCompress(r)
	var finalResponse []byte

	if shouldCompress {
		compressedResponse, compressErr := c.standardizer.compressResponseMandatory(modifiedResponse)
		if compressErr != nil {
			log.Printf("Error compressing response for vendor %s: %v", selection.Vendor, compressErr)
			// Fall back to uncompressed on compression error
			finalResponse = modifiedResponse
			shouldCompress = false
		} else {
			finalResponse = compressedResponse
		}
	} else {
		finalResponse = modifiedResponse
	}

	// Set all standard headers with complete compliance
	c.standardizer.setCompliantHeaders(w, selection.Vendor, len(finalResponse), shouldCompress)

	// Write status and headers
	w.WriteHeader(resp.StatusCode)

	// Write the final response
	_, err = w.Write(finalResponse)
	if err != nil {
		log.Printf("Error writing response for vendor %s: %v", selection.Vendor, err)
	}

	return nil
}
