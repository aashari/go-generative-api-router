package proxy

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aashari/go-generative-api-router/internal/logger"
	"github.com/aashari/go-generative-api-router/internal/selector"
	"github.com/aashari/go-generative-api-router/internal/utils"
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
	// Configure client timeout from environment variable
	// Default to 1200 seconds (20 minutes) to allow for longer AI model responses
	// This prevents 120-second timeouts that can occur with complex requests
	clientTimeout := utils.GetEnvDuration("CLIENT_TIMEOUT", 1200*time.Second)

	httpClient := &http.Client{
		Timeout: clientTimeout,
	}

	logger.Info("API client initialized",
		"client_timeout", clientTimeout,
		"openai_base_url", "https://api.openai.com/v1",
		"gemini_base_url", "https://generativelanguage.googleapis.com/v1beta/openai",
	)

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

	// Log complete vendor request data before sending
	var vendorBodyForLog interface{}
	if err := json.Unmarshal(modifiedBody, &vendorBodyForLog); err != nil {
		vendorBodyForLog = string(modifiedBody)
	}

	logger.LogWithStructure(r.Context(), logger.LevelInfo, "Complete vendor request about to be sent",
		map[string]interface{}{
			"vendor":         selection.Vendor,
			"model":          selection.Model,
			"original_model": originalModel,
			"is_streaming":   isStreaming,
		},
		map[string]interface{}{
			"vendor_method":  req.Method,
			"vendor_url":     req.URL.String(),
			"vendor_headers": map[string][]string(req.Header),
			"vendor_body":    vendorBodyForLog,
			"client_method":  r.Method,
			"client_path":    r.URL.Path,
			"client_headers": map[string][]string(r.Header),
			"remote_addr":    r.RemoteAddr,
		},
		nil, // response
		nil) // error

	// 2. Send request to vendor
	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		logger.LogError(r.Context(), "vendor_communication", err, map[string]any{
			"vendor":          selection.Vendor,
			"url":             req.URL.String(),
			"request_body":    string(modifiedBody),
			"request_headers": map[string][]string(req.Header),
		})
		return fmt.Errorf("failed to send request to vendor: %v", err)
	}
	defer resp.Body.Close()

	// Check for HTTP error status codes and parse vendor errors
	if resp.StatusCode >= 400 {
		// Read response body for error parsing
		errorBody, readErr := c.standardizer.processResponseBody(resp.Body, resp.Header.Get("Content-Encoding"), selection.Vendor)
		if readErr != nil {
			logger.ErrorCtx(r.Context(), "Failed to read error response body",
				"vendor", selection.Vendor,
				"status_code", resp.StatusCode,
				"error", readErr)
			// Create a generic error if we can't read the response
			return ParseVendorError(selection.Vendor, resp.StatusCode, nil)
		}

		// Database logging removed - no longer logging vendor requests

		// Parse the vendor error
		vendorErr := ParseVendorError(selection.Vendor, resp.StatusCode, errorBody)
		if vendorErr != nil {
			logger.LogWithStructure(r.Context(), logger.LevelWarn, "Vendor API error detected",
				map[string]interface{}{
					"vendor":      selection.Vendor,
					"status_code": resp.StatusCode,
					"error_type":  fmt.Sprintf("%T", vendorErr),
					"retriable":   IsRetriableAPIError(vendorErr),
				},
				nil, // request
				map[string]interface{}{
					"response_body":    string(errorBody),
					"response_headers": map[string][]string(resp.Header),
				},
				map[string]interface{}{
					"message": vendorErr.Error(),
					"type":    "vendor_api_error",
				}) // error
			return vendorErr
		}
	}

	// Log complete vendor response headers immediately
	logger.LogWithStructure(r.Context(), logger.LevelInfo, "Complete vendor response headers received",
		map[string]interface{}{
			"vendor":         selection.Vendor,
			"model":          selection.Model,
			"original_model": originalModel,
			"is_streaming":   isStreaming,
		},
		nil, // request
		map[string]interface{}{
			"status_code":    resp.StatusCode,
			"status":         resp.Status,
			"headers":        map[string][]string(resp.Header),
			"content_length": resp.ContentLength,
		},
		nil) // error

	// 3. Handle response based on streaming mode
	if isStreaming {
		// Setup headers for streaming and handle streaming response
		c.setupResponseHeadersWithVendor(w, resp, isStreaming, selection.Vendor)
		return c.handleStreaming(w, r, resp, selection, originalModel, duration, modifiedBody)
	} else {
		// For non-streaming, we need to process the response first to determine compression
		return c.handleNonStreamingWithHeaders(w, r, resp, selection, originalModel, duration, modifiedBody)
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
			// Note: Streaming initiation is logged by the proxy layer with request context
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

// setupResponseHeadersWithVendor sets up response headers with vendor awareness
func (c *APIClient) setupResponseHeadersWithVendor(w http.ResponseWriter, resp *http.Response, isStreaming bool, vendor string) {
	// Set base compliant headers (content-length=0 for streaming to prevent it being set)
	c.standardizer.setCompliantHeaders(w, vendor, 0, false)

	// Log complete header mapping
	logger.LogWithStructure(context.Background(), logger.LevelInfo, "Setting up response headers with complete data",
		map[string]interface{}{
			"vendor":       vendor,
			"is_streaming": isStreaming,
		},
		nil, // request
		map[string]interface{}{
			"vendor_response_headers": map[string][]string(resp.Header),
		},
		nil) // error

	// Override content type for streaming mode
	if isStreaming {
		// Set essential SSE headers - override JSON content type for streaming
		w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		// Remove Content-Length for streaming as it's chunked
		w.Header().Del("Content-Length")
		// Explicitly set Transfer-Encoding to chunked so Go will not add a Content-Length later
		w.Header().Set("Transfer-Encoding", "chunked")
		// Set X-Accel-Buffering to no to prevent nginx from buffering
		w.Header().Set("X-Accel-Buffering", "no")
		// Log complete streaming headers setup
		logger.LogWithStructure(context.Background(), logger.LevelInfo, "Set streaming headers with complete data",
			map[string]interface{}{
				"vendor":                 vendor,
				"final_response_headers": map[string][]string(w.Header()),
				"content_type":           w.Header().Get("Content-Type"),
				"cache_control":          w.Header().Get("Cache-Control"),
				"connection":             w.Header().Get("Connection"),
				"content_length_removed": w.Header().Get("Content-Length") == "",
				"transfer_encoding":      w.Header().Get("Transfer-Encoding"),
				"x_accel_buffering":      w.Header().Get("X-Accel-Buffering"),
			},
			nil, // request
			nil, // response
			nil) // error
	}

	// Write status code after setting all headers
	w.WriteHeader(resp.StatusCode)

	// For streaming, immediately flush headers to ensure chunked transfer encoding
	if isStreaming {
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
	}
}

// handleStreaming processes streaming responses
func (c *APIClient) handleStreaming(w http.ResponseWriter, r *http.Request, resp *http.Response, selection *selector.VendorSelection, originalModel string, duration time.Duration, modifiedBody []byte) error {
	// Log complete streaming request processing start
	logger.LogWithStructure(r.Context(), logger.LevelInfo, "Processing streaming request with complete data",
		map[string]interface{}{
			"vendor":         selection.Vendor,
			"model":          selection.Model,
			"original_model": originalModel,
		},
		map[string]interface{}{
			"method":         r.Method,
			"path":           r.URL.Path,
			"headers":        map[string][]string(r.Header),
			"status_code":    resp.StatusCode,
			"vendor_headers": map[string][]string(resp.Header),
		},
		nil, // response
		nil) // error

	// Generate consistent conversation-level values for streaming responses
	conversationID := ChatCompletionID()
	timestamp := time.Now().Unix()
	systemFingerprint := SystemFingerprint()
	// Log complete streaming values generation
	logger.LogWithStructure(r.Context(), logger.LevelInfo, "Generated streaming values with complete data",
		map[string]interface{}{
			"conversation_id":    conversationID,
			"timestamp":          timestamp,
			"system_fingerprint": systemFingerprint,
			"vendor":             selection.Vendor,
			"model":              selection.Model,
			"original_model":     originalModel,
		},
		nil, // request
		nil, // response
		nil) // error

	// Create stream processor
	streamProcessor := NewStreamProcessor(conversationID, timestamp, systemFingerprint, selection.Vendor, originalModel)

	// Get content encoding for gzip handling
	contentEncoding := resp.Header.Get("Content-Encoding")
	if contentEncoding != "" {
		// Log complete content encoding information
		logger.LogWithStructure(r.Context(), logger.LevelInfo, "Response content encoding with complete data",
			map[string]interface{}{
				"content_encoding": contentEncoding,
				"vendor":           selection.Vendor,
				"is_streaming":     true,
			},
			nil, // request
			map[string]interface{}{
				"complete_response_headers": map[string][]string(resp.Header),
			},
			nil) // error
	}

	// Create the appropriate reader based on content encoding
	var reader *bufio.Reader
	if contentEncoding == "gzip" {
		// Log complete gzip reader setup
		logger.LogWithStructure(r.Context(), logger.LevelInfo, "Setting up gzip reader for streaming with complete data",
			map[string]interface{}{
				"vendor":           selection.Vendor,
				"content_encoding": contentEncoding,
			},
			nil, // request
			map[string]interface{}{
				"complete_response_headers": map[string][]string(resp.Header),
			},
			nil) // error
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			logger.LogError(r.Context(), "streaming_gzip_setup", err, map[string]any{
				"vendor":                    selection.Vendor,
				"content_encoding":          contentEncoding,
				"complete_response_headers": map[string][]string(resp.Header),
			})
			return fmt.Errorf("error creating gzip reader for streaming: %w", err)
		}
		defer gzipReader.Close()
		reader = bufio.NewReader(gzipReader)
	} else {
		reader = bufio.NewReader(resp.Body)
	}

	// Try to get a flusher from the response writer
	flusher, ok := w.(http.Flusher)
	if !ok {
		// Log complete flusher unavailability
		logger.LogWithStructure(r.Context(), logger.LevelWarn, "ResponseWriter does not support flushing with complete data",
			map[string]interface{}{
				"response_writer_type": fmt.Sprintf("%T", w),
				"vendor":               selection.Vendor,
				"streaming":            true,
			},
			nil, // request
			nil, // response
			nil) // error
	}

	return c.processStreamingResponse(w, reader, streamProcessor, flusher)
}

// validateVendorResponse validates JSON responses from vendors
func (s *ResponseStandardizer) validateVendorResponse(body []byte, vendor string) error {
	if len(body) == 0 {
		// Log complete empty response error
		logger.LogError(context.Background(), "response_validation", fmt.Errorf("empty response from vendor"), map[string]any{
			"vendor":        vendor,
			"response_body": body,
			"response_size": len(body),
		})
		return ErrInvalidResponse
	}

	// Quick check if the response is valid JSON
	if !bytes.HasPrefix(bytes.TrimSpace(body), []byte("{")) && !bytes.HasPrefix(bytes.TrimSpace(body), []byte("[")) {
		// Log complete invalid JSON format error
		logger.LogError(context.Background(), "response_validation", fmt.Errorf("invalid JSON format"), map[string]any{
			"vendor":          vendor,
			"response_body":   string(body),
			"response_size":   len(body),
			"response_prefix": string(bytes.TrimSpace(body)[:min(50, len(bytes.TrimSpace(body)))]),
		})
		return ErrInvalidResponse
	}

	// Handle both object and array responses
	var responseData map[string]interface{}
	var err error

	// Check if response starts with array bracket
	if bytes.HasPrefix(bytes.TrimSpace(body), []byte("[")) {
		// Handle array response (common for error responses from some vendors)
		var arrayResponse []interface{}
		if err := json.Unmarshal(body, &arrayResponse); err != nil {
			// Log complete JSON parsing error for array
			logger.LogError(context.Background(), "response_validation", err, map[string]any{
				"vendor":        vendor,
				"response_body": string(body),
				"response_size": len(body),
				"response_type": "array",
			})
			return fmt.Errorf("%w: %v", ErrInvalidResponse, err)
		}

		// Check if array contains error objects
		if len(arrayResponse) > 0 {
			if firstItem, ok := arrayResponse[0].(map[string]interface{}); ok {
				if _, hasError := firstItem["error"]; hasError {
					// This is an error response in array format - treat as valid error response
					logger.LogWithStructure(context.Background(), logger.LevelDebug, "Array error response validation successful",
						map[string]interface{}{
							"vendor":                 vendor,
							"complete_response_data": arrayResponse,
							"response_size":          len(body),
							"response_type":          "array_error",
						},
						nil, // request
						map[string]interface{}{
							"response_body": string(body),
						},
						nil) // error
					return nil
				}
			}
		}

		// Array response but not an error - this is unexpected for OpenAI-compatible APIs
		logger.LogError(context.Background(), "response_validation", fmt.Errorf("unexpected array response format"), map[string]any{
			"vendor":                 vendor,
			"complete_response_data": arrayResponse,
			"response_body":          string(body),
			"response_size":          len(body),
			"response_type":          "unexpected_array",
		})
		return fmt.Errorf("%w: unexpected array response format", ErrInvalidResponse)
	}

	// Handle object response (normal case)
	if err = json.Unmarshal(body, &responseData); err != nil {
		// Log complete JSON parsing error for object
		logger.LogError(context.Background(), "response_validation", err, map[string]any{
			"vendor":        vendor,
			"response_body": string(body),
			"response_size": len(body),
			"response_type": "object",
		})
		return fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}

	// Check if this is an error response first
	if isErrorResponse(responseData) {
		// Log complete successful error response validation
		logger.LogWithStructure(context.Background(), logger.LevelDebug, "Error response validation successful with complete data",
			map[string]interface{}{
				"vendor":                 vendor,
				"complete_response_data": responseData,
				"response_size":          len(body),
				"response_type":          "error",
			},
			nil, // request
			map[string]interface{}{
				"response_body": string(body),
			},
			nil) // error
		return nil
	}

	// Check if this is a response with zero completion tokens
	hasZeroCompletionTokens := false
	if usage, ok := responseData["usage"].(map[string]interface{}); ok {
		if completionTokens, ok := usage["completion_tokens"]; ok {
			if tokens, ok := completionTokens.(float64); ok && tokens == 0 {
				hasZeroCompletionTokens = true
			}
		}
	}

	// Basic validation: check for required fields in non-error responses
	requiredFields := []string{"id", "object"}

	// Only require "choices" field if the response has completion tokens
	// For responses with zero completion tokens, the choices field may be missing or empty
	if !hasZeroCompletionTokens {
		requiredFields = append(requiredFields, "choices")
	}

	for _, field := range requiredFields {
		if _, ok := responseData[field]; !ok {
			// Log complete missing field error
			logger.LogError(context.Background(), "response_validation", fmt.Errorf("missing required field"), map[string]any{
				"missing_field":              field,
				"vendor":                     vendor,
				"complete_response_data":     responseData,
				"response_body":              string(body),
				"required_fields":            requiredFields,
				"has_zero_completion_tokens": hasZeroCompletionTokens,
			})
			return fmt.Errorf("%w: missing required field '%s'", ErrInvalidResponse, field)
		}
	}

	// Additional validation: check if choices is empty when we have non-zero completion tokens
	if !hasZeroCompletionTokens {
		if choices, ok := responseData["choices"].([]interface{}); ok && len(choices) == 0 {
			// Log complete empty choices error
			logger.LogError(context.Background(), "response_validation", fmt.Errorf("empty choices array"), map[string]any{
				"vendor":                     vendor,
				"complete_response_data":     responseData,
				"response_body":              string(body),
				"has_zero_completion_tokens": hasZeroCompletionTokens,
				"choices_length":             0,
			})
			return fmt.Errorf("%w: empty choices array with non-zero completion tokens", ErrInvalidResponse)
		}
	}

	// Log complete successful validation
	logger.LogWithStructure(context.Background(), logger.LevelDebug, "Response validation successful with complete data",
		map[string]interface{}{
			"vendor":                     vendor,
			"complete_response_data":     responseData,
			"response_size":              len(body),
			"validated_fields":           requiredFields,
			"has_zero_completion_tokens": hasZeroCompletionTokens,
		},
		nil, // request
		map[string]interface{}{
			"response_body": string(body),
		},
		nil) // error
	return nil
}

// setCompliantHeaders sets standardized headers for all responses
func (s *ResponseStandardizer) setCompliantHeaders(w http.ResponseWriter, vendor string, contentLength int, isCompressed bool) {
	// Set standard security and cache headers
	for k, v := range s.standardHeaders {
		w.Header().Set(k, v)
	}

	// Set service identification headers
	w.Header().Set("Server", "Generative-API-Router/1.0")
	w.Header().Set("X-Powered-By", "Generative-API-Router")
	w.Header().Set("X-Vendor-Source", vendor)

	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID, X-Response-Time")

	// Set date header
	w.Header().Set("Date", time.Now().UTC().Format(http.TimeFormat))

	// Note: Request ID is already set by the correlation middleware
	// No need to generate a new one here

	// Set content type for JSON responses
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	// Set compression headers if applicable
	if isCompressed {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")
	}

	// Set content length if available
	if contentLength > 0 {
		w.Header().Set("Content-Length", strconv.Itoa(contentLength))
	}

	logger.Debug("Set standardized headers",
		"vendor", vendor,
		"content_length", contentLength,
		"compressed", isCompressed)
}

// processResponseBody handles response body processing
func (s *ResponseStandardizer) processResponseBody(body io.Reader, contentEncoding string, vendor string) ([]byte, error) {
	if contentEncoding == "gzip" {
		logger.Debug("Decompressing gzip response", "vendor", vendor)
		gzipReader, err := gzip.NewReader(body)
		if err != nil {
			logger.Error("Failed to create gzip reader", "vendor", vendor, "error", err)
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzipReader.Close()
		body = gzipReader
	}

	// Read the entire response body
	responseBody, err := io.ReadAll(body)
	if err != nil {
		logger.Error("Failed to read response", "vendor", vendor, "error", err)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	logger.Debug("Processed response body",
		"bytes", len(responseBody),
		"vendor", vendor,
		"gzipped", contentEncoding == "gzip")
	return responseBody, nil
}

// shouldCompress determines if compression should be applied
func (s *ResponseStandardizer) shouldCompress(r *http.Request) bool {
	if !s.enableGzip {
		return false
	}

	// Check Accept-Encoding header
	acceptEncoding := r.Header.Get("Accept-Encoding")
	userAgent := r.Header.Get("User-Agent")

	// Disable compression for known problematic clients
	if strings.Contains(userAgent, "curl/") && !strings.Contains(userAgent, "curl/8") {
		logger.Debug("Disabling compression for older curl client", "user_agent", userAgent)
		return false
	}

	// Disable compression for Postman and Insomnia clients
	if strings.Contains(userAgent, "PostmanRuntime") || strings.Contains(strings.ToLower(userAgent), "insomnia") {
		logger.Debug("Disabling compression for API testing client", "user_agent", userAgent)
		return false
	}

	logger.Debug("Compression check",
		"accept_encoding", acceptEncoding,
		"user_agent", userAgent,
		"will_compress", strings.Contains(acceptEncoding, "gzip"))
	return strings.Contains(acceptEncoding, "gzip")
}

// compressResponseMandatory compresses response data
func (s *ResponseStandardizer) compressResponseMandatory(body []byte) ([]byte, error) {
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)

	_, err := gzipWriter.Write(body)
	if err != nil {
		logger.Error("Gzip compression error", "error", err)
		return body, err
	}

	err = gzipWriter.Close()
	if err != nil {
		logger.Error("Gzip compression close error", "error", err)
		return body, err
	}

	logger.Debug("Compressed response",
		"original_bytes", len(body),
		"compressed_bytes", buf.Len(),
		"reduction_percent", float64(len(body)-buf.Len())*100/float64(len(body)))
	return buf.Bytes(), nil
}

// processStreamingResponse handles streaming SSE responses
func (c *APIClient) processStreamingResponse(w http.ResponseWriter, reader *bufio.Reader, streamProcessor *StreamProcessor, flusher http.Flusher) error {
	for {
		// Read the "data: " line
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			logger.Error("Error reading stream", "error", err)
			return fmt.Errorf("error reading stream: %w", err)
		}

		// Check for [DONE] message
		if strings.Contains(line, "[DONE]") {
			// Forward the [DONE] message
			_, err = w.Write([]byte("data: [DONE]\n\n"))
			if flusher != nil {
				flusher.Flush()
			}
			return err
		}

		// Process the chunk
		processedChunk := streamProcessor.ProcessChunk([]byte(line))
		if processedChunk == nil {
			continue // Skip invalid chunks
		}

		// Log complete streaming chunk data
		logger.LogWithStructure(context.Background(), logger.LevelDebug, "Complete streaming chunk processed",
			map[string]interface{}{
				"vendor":          streamProcessor.Vendor,
				"model":           streamProcessor.OriginalModel,
				"conversation_id": streamProcessor.ConversationID,
			},
			nil, // request
			map[string]interface{}{
				"original_chunk":   string(line),
				"processed_chunk":  string(processedChunk),
				"chunk_size_bytes": len(processedChunk),
			},
			nil) // error

		// Handle SSE line endings (needs \n\n)
		if !bytes.HasSuffix(processedChunk, []byte("\n\n")) {
			if bytes.HasSuffix(processedChunk, []byte("\n")) {
				processedChunk = append(processedChunk, '\n')
			} else {
				processedChunk = append(processedChunk, '\n', '\n')
			}
		}

		// Write the processed chunk
		_, err = w.Write(processedChunk)
		if err != nil {
			return fmt.Errorf("error writing chunk: %w", err)
		}

		// Flush to ensure streaming
		if flusher != nil {
			flusher.Flush()
		}

		// Some SSE implementations have an extra newline after data
		if !strings.HasSuffix(line, "\n\n") {
			_, err := reader.ReadString('\n')
			if err != nil && err != io.EOF {
				logger.Error("Error reading empty line after data", "error", err)
			}
		}
	}
}

// Database logging functionality has been removed

// handleNonStreamingWithHeaders processes non-streaming responses
func (c *APIClient) handleNonStreamingWithHeaders(w http.ResponseWriter, r *http.Request, resp *http.Response, selection *selector.VendorSelection, originalModel string, duration time.Duration, modifiedBody []byte) error {
	logger.InfoCtx(r.Context(), "Processing non-streaming request", "vendor", selection.Vendor)

	// 1. Process response body
	responseBody, err := c.standardizer.processResponseBody(resp.Body, resp.Header.Get("Content-Encoding"), selection.Vendor)
	if err != nil {
		logger.ErrorCtx(r.Context(), "Error processing response body",
			"vendor", selection.Vendor,
			"error", err)
		return err
	}

	// Log complete vendor response body immediately after processing
	var vendorResponseBodyForLog interface{}
	if err := json.Unmarshal(responseBody, &vendorResponseBodyForLog); err != nil {
		vendorResponseBodyForLog = string(responseBody)
	}

	logger.LogWithStructure(r.Context(), logger.LevelInfo, "Complete vendor response body received",
		map[string]interface{}{
			"vendor":         selection.Vendor,
			"model":          selection.Model,
			"original_model": originalModel,
			"is_streaming":   false,
		},
		nil, // request
		map[string]interface{}{
			"status_code":      resp.StatusCode,
			"status":           resp.Status,
			"headers":          map[string][]string(resp.Header),
			"content_length":   resp.ContentLength,
			"body":             vendorResponseBodyForLog,
			"body_size_bytes":  len(responseBody),
			"content_encoding": resp.Header.Get("Content-Encoding"),
		},
		nil) // error

	// 2. Validate response
	if c.standardizer.enableValidation {
		if err := c.standardizer.validateVendorResponse(responseBody, selection.Vendor); err != nil {
			logger.ErrorCtx(r.Context(), "Vendor response validation failed",
				"vendor", selection.Vendor,
				"error", err)

			// Wrap validation errors with vendor information for potential retry
			if errors.Is(err, ErrInvalidResponse) {
				// Check if it's specifically missing 'choices' field
				if strings.Contains(err.Error(), "missing required field 'choices'") {
					return &VendorValidationError{
						Vendor:       selection.Vendor,
						OriginalErr:  err,
						MissingField: "choices",
					}
				}
				// Check if it's empty choices array
				if strings.Contains(err.Error(), "empty choices array") {
					return &VendorValidationError{
						Vendor:       selection.Vendor,
						OriginalErr:  err,
						MissingField: "choices", // Use same field name for both cases
					}
				}
				// Other validation errors
				return &VendorValidationError{
					Vendor:      selection.Vendor,
					OriginalErr: err,
				}
			}
			return err
		}
	}

	// 3. Process response (replace model, format, etc.)
	modifiedResponse, err := ProcessResponse(responseBody, selection.Vendor, resp.Header.Get("Content-Encoding"), originalModel)
	if err != nil {
		logger.ErrorCtx(r.Context(), "Error processing response",
			"vendor", selection.Vendor,
			"error", err)
		return err
	}

	// 4. Determine compression
	shouldCompress := c.standardizer.shouldCompress(r)
	var finalResponse []byte
	var compressErr error

	if shouldCompress {
		finalResponse, compressErr = c.standardizer.compressResponseMandatory(modifiedResponse)
		if compressErr != nil {
			logger.ErrorCtx(r.Context(), "Error compressing response",
				"vendor", selection.Vendor,
				"error", compressErr)
			// Fall back to uncompressed if compression fails
			finalResponse = modifiedResponse
			shouldCompress = false
		} else {
			// Set the Content-Encoding header for compressed responses
			w.Header().Set("Content-Encoding", "gzip")
		}
	} else {
		finalResponse = modifiedResponse
	}

	// 5. Set headers
	c.standardizer.setCompliantHeaders(w, selection.Vendor, len(finalResponse), shouldCompress)

	// 6. Write the response
	_, err = w.Write(finalResponse)
	if err != nil {
		logger.ErrorCtx(r.Context(), "Error writing response",
			"vendor", selection.Vendor,
			"error", err)
		return err
	}

	// Log complete final response sent to client
	var finalResponseForLog interface{}
	if err := json.Unmarshal(finalResponse, &finalResponseForLog); err != nil {
		finalResponseForLog = string(finalResponse)
	}

	logger.LogWithStructure(r.Context(), logger.LevelInfo, "Complete final response sent to client",
		map[string]interface{}{
			"vendor":                 selection.Vendor,
			"model":                  selection.Model,
			"original_model":         originalModel,
			"is_streaming":           false,
			"original_response_size": len(responseBody),
			"modified_response_size": len(modifiedResponse),
			"final_response_size":    len(finalResponse),
			"compression_applied":    shouldCompress,
		},
		nil, // request
		map[string]interface{}{
			"body":             finalResponseForLog,
			"body_size_bytes":  len(finalResponse),
			"headers":          map[string][]string(w.Header()),
			"compressed":       shouldCompress,
			"content_encoding": w.Header().Get("Content-Encoding"),
		},
		nil) // error

	// Database logging removed - no longer logging vendor requests

	return nil
}
