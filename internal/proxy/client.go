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

	"github.com/aashari/go-generative-api-router/internal/config"
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
			utils.HeaderCacheControl:        utils.CacheControlNoStore,
			utils.HeaderXContentTypeOptions: utils.XContentTypeOptionsNoSniff,
			utils.HeaderXFrameOptions:       utils.XFrameOptionsDeny,
			utils.HeaderXXSSProtection:      utils.XXSSProtectionBlock,
			utils.HeaderReferrerPolicy:      utils.ReferrerPolicyStrict,
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
func NewAPIClient(vendors map[string]string) *APIClient {
	// Configure client timeout from environment variable
	// Default to 1200 seconds (20 minutes) to allow for longer AI model responses
	// This prevents 120-second timeouts that can occur with complex requests
	clientTimeout := utils.GetEnvDuration("CLIENT_TIMEOUT", 1200*time.Second)

	httpClient := &http.Client{
		Timeout: clientTimeout,
	}

	logger.Info(context.Background(), "API client initialized",
		"client_timeout", clientTimeout,
		"openai_base_url", vendors["openai"],
		"gemini_base_url", vendors["gemini"],
		"component", "APIClient",
		"stage", "Initialized",
	)

	return &APIClient{
		BaseURLs:     vendors,
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

	// Log complete vendor request data before sending - including full credential and model objects
	var vendorBodyForLog interface{}
	if err := json.Unmarshal(modifiedBody, &vendorBodyForLog); err != nil {
		vendorBodyForLog = string(modifiedBody)
	}

	// Get complete model object from context if available
	var completeModelObject interface{}
	if vendorModels := r.Context().Value("vendor_models"); vendorModels != nil {
		if models, ok := vendorModels.([]config.VendorModel); ok {
			for _, model := range models {
				if model.Vendor == selection.Vendor && model.Model == selection.Model {
					completeModelObject = model
					break
				}
			}
		}
	}

	logger.Info(r.Context(), "Complete vendor request about to be sent",
		"vendor", selection.Vendor,
		"model", selection.Model,
		"original_model", originalModel,
		"is_streaming", isStreaming,
		"complete_credential_object", selection.Credential, // Full credential object as requested
		"complete_model_object", completeModelObject, // Full model object as requested
		"vendor_selection_details", map[string]interface{}{
			"selected_vendor":     selection.Vendor,
			"selected_model":      selection.Model,
			"credential_platform": selection.Credential.Platform,
			"credential_type":     selection.Credential.Type,
			"credential_value":    selection.Credential.Value, // Full API key for debugging
		},
		"vendor_method", req.Method,
		"vendor_url", req.URL.String(),
		"vendor_headers", map[string][]string(req.Header),
		"vendor_body", vendorBodyForLog,
		"client_method", r.Method,
		"client_path", r.URL.Path,
		"client_headers", map[string][]string(r.Header),
		"remote_addr", r.RemoteAddr,
		"component", "APIClient",
		"stage", "SendingRequest",
	)

	// 2. Send request to vendor
	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		logger.Error(r.Context(), "vendor communication failed", err,
			"vendor", selection.Vendor,
			"url", req.URL.String(),
			"request_body", string(modifiedBody),
			"request_headers", map[string][]string(req.Header),
			"complete_credential_object", selection.Credential, // Full credential object in error logs too
			"complete_model_object", completeModelObject, // Full model object in error logs too
			"component", "APIClient",
			"stage", "VendorCommunication",
		)
		return fmt.Errorf("failed to send request to vendor: %v", err)
	}
	defer resp.Body.Close()

	// Check for HTTP error status codes and parse vendor errors
	if resp.StatusCode >= 400 {
		// Read response body for error parsing
		errorBody, readErr := c.standardizer.processResponseBody(resp.Body, resp.Header.Get(utils.HeaderContentEncoding), selection.Vendor)
		if readErr != nil {
			logger.Error(r.Context(), "Failed to read error response body", readErr,
				"vendor", selection.Vendor,
				"status_code", resp.StatusCode,
				"complete_credential_object", selection.Credential, // Full credential object
				"complete_model_object", completeModelObject, // Full model object
				"component", "APIClient",
				"stage", "ErrorResponseRead",
			)
			// Create a generic error if we can't read the response
			return ParseVendorError(selection.Vendor, resp.StatusCode, nil)
		}

		// Database logging removed - no longer logging vendor requests

		// Parse the vendor error
		vendorErr := ParseVendorError(selection.Vendor, resp.StatusCode, errorBody)
		if vendorErr != nil {
			logger.Warn(r.Context(), "Vendor API error detected",
				"vendor", selection.Vendor,
				"status_code", resp.StatusCode,
				"error_type", fmt.Sprintf("%T", vendorErr),
				"retriable", IsRetriableAPIError(vendorErr),
				"complete_credential_object", selection.Credential, // Full credential object in error
				"complete_model_object", completeModelObject, // Full model object in error
				"response_body", string(errorBody),
				"response_headers", map[string][]string(resp.Header),
				"error_message", vendorErr.Error(),
				"error_type_category", "vendor_api_error",
				"component", "APIClient",
				"stage", "VendorAPIError",
			)
			return vendorErr
		}
	}

	// Log complete vendor response headers immediately - including full objects
	logger.Info(r.Context(), "Complete vendor response headers received",
		"vendor", selection.Vendor,
		"model", selection.Model,
		"original_model", originalModel,
		"is_streaming", isStreaming,
		"complete_credential_object", selection.Credential, // Full credential object
		"complete_model_object", completeModelObject, // Full model object
		"status_code", resp.StatusCode,
		"status", resp.Status,
		"headers", map[string][]string(resp.Header),
		"content_length", resp.ContentLength,
		"component", "APIClient",
		"stage", "VendorResponseHeaders",
	)

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

	// Copy request headers (now including compression headers to enable vendor compression)
	for k, vs := range r.Header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}

	// Enable gzip compression for vendor requests to reduce bandwidth and improve performance
	req.Header.Set(utils.HeaderAcceptEncoding, utils.AcceptEncodingGzip)

	// Set authorization header using Bearer token for all vendors
	req.Header.Set(utils.HeaderAuthorization, "Bearer "+selection.Credential.Value)

	return req, isStreaming, nil
}

// setupResponseHeadersWithVendor sets up response headers with vendor awareness
func (c *APIClient) setupResponseHeadersWithVendor(w http.ResponseWriter, resp *http.Response, isStreaming bool, vendor string) {
	// Set base compliant headers (content-length=0 for streaming to prevent it being set)
	c.standardizer.setCompliantHeaders(w, vendor, 0, false)

	// Log complete header mapping
	logger.Info(context.Background(), "Setting up response headers with complete data",
		"vendor", vendor,
		"is_streaming", isStreaming,
		"vendor_response_headers", map[string][]string(resp.Header),
		"component", "APIClient",
		"stage", "ResponseHeaderSetup",
	)

	// Override content type for streaming mode
	if isStreaming {
		// Set essential SSE headers - override JSON content type for streaming
		w.Header().Set(utils.HeaderContentType, utils.ContentTypeEventStreamUTF8)
		w.Header().Set(utils.HeaderCacheControl, utils.CacheControlNoCache)
		w.Header().Set(utils.HeaderConnection, utils.ConnectionKeepAlive)
		// Remove Content-Length for streaming as it's chunked
		w.Header().Del(utils.HeaderContentLength)
		// Explicitly set Transfer-Encoding to chunked so Go will not add a Content-Length later
		w.Header().Set(utils.HeaderTransferEncoding, utils.TransferEncodingChunked)
		// Set X-Accel-Buffering to no to prevent nginx from buffering
		w.Header().Set(utils.HeaderXAccelBuffering, utils.XAccelBufferingNo)
		// Log complete streaming headers setup
		logger.Info(context.Background(), "Set streaming headers with complete data",
			"vendor", vendor,
			"final_response_headers", map[string][]string(w.Header()),
			"content_type", w.Header().Get(utils.HeaderContentType),
			"cache_control", w.Header().Get(utils.HeaderCacheControl),
			"connection", w.Header().Get("Connection"),
			"content_length_removed", w.Header().Get(utils.HeaderContentLength) == "",
			"transfer_encoding", w.Header().Get(utils.HeaderTransferEncoding),
			"x_accel_buffering", w.Header().Get(utils.HeaderXAccelBuffering),
			"component", "APIClient",
			"stage", "StreamingHeadersSetup",
		)
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
	// Get complete model object from context if available
	var completeModelObject interface{}
	if vendorModels := r.Context().Value("vendor_models"); vendorModels != nil {
		if models, ok := vendorModels.([]config.VendorModel); ok {
			for _, model := range models {
				if model.Vendor == selection.Vendor && model.Model == selection.Model {
					completeModelObject = model
					break
				}
			}
		}
	}

	// Log complete streaming request processing start
	logger.Info(r.Context(), "Processing streaming request with complete data",
		"vendor", selection.Vendor,
		"model", selection.Model,
		"original_model", originalModel,
		"complete_credential_object", selection.Credential, // Full credential object
		"complete_model_object", completeModelObject, // Full model object
		"method", r.Method,
		"path", r.URL.Path,
		"headers", map[string][]string(r.Header),
		"status_code", resp.StatusCode,
		"vendor_headers", map[string][]string(resp.Header),
		"component", "APIClient",
		"stage", "StreamingProcessingStart",
	)

	// Generate consistent conversation-level values for streaming responses
	conversationID := utils.GenerateChatCompletionID()
	timestamp := time.Now().Unix()
	systemFingerprint := utils.GenerateSystemFingerprint()
	// Log complete streaming values generation
	logger.Info(r.Context(), "Generated streaming values with complete data",
		"conversation_id", conversationID,
		"timestamp", timestamp,
		"system_fingerprint", systemFingerprint,
		"vendor", selection.Vendor,
		"model", selection.Model,
		"original_model", originalModel,
		"complete_credential_object", selection.Credential, // Full credential object
		"complete_model_object", completeModelObject, // Full model object
		"component", "APIClient",
		"stage", "StreamingValuesGeneration",
	)

	// Create stream processor
	streamProcessor := NewStreamProcessor(conversationID, timestamp, systemFingerprint, selection.Vendor, originalModel)

	// Get content encoding for gzip handling
	contentEncoding := resp.Header.Get(utils.HeaderContentEncoding)
	var reader io.Reader = resp.Body

	// Handle gzip decompression if needed
	if contentEncoding == utils.AcceptEncodingGzip {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			logger.Error(r.Context(), "Failed to create gzip reader for streaming response", err,
				"vendor", selection.Vendor,
				"complete_credential_object", selection.Credential, // Full credential object
				"complete_model_object", completeModelObject, // Full model object
				"component", "APIClient",
				"stage", "StreamingGzipReader",
			)
			return fmt.Errorf("failed to decompress streaming response: %v", err)
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	// Create buffered reader for line-by-line processing
	bufReader := bufio.NewReader(reader)

	// Get flusher for real-time streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
		logger.Error(r.Context(), "ResponseWriter does not support flushing", fmt.Errorf("streaming not supported"),
			"vendor", selection.Vendor,
			"complete_credential_object", selection.Credential, // Full credential object
			"complete_model_object", completeModelObject, // Full model object
			"component", "APIClient",
			"stage", "StreamingFlushCheck",
		)
		return fmt.Errorf("streaming not supported")
	}

	// Process the streaming response
	return c.processStreamingResponse(w, bufReader, streamProcessor, flusher)
}

// validateVendorResponse validates JSON responses from vendors
func (s *ResponseStandardizer) validateVendorResponse(body []byte, vendor string) error {
	if len(body) == 0 {
		// Log complete empty response error
		logger.Error(context.Background(), "empty response from vendor", fmt.Errorf("empty response from vendor"),
			"vendor", vendor,
			"response_body", body,
			"response_size", len(body),
			"component", "ResponseStandardizer",
			"stage", "ValidationEmptyResponse",
		)
		return ErrInvalidResponse
	}

	// Quick check if the response is valid JSON
	if !bytes.HasPrefix(bytes.TrimSpace(body), []byte("{")) && !bytes.HasPrefix(bytes.TrimSpace(body), []byte("[")) {
		// Log complete invalid JSON format error
		logger.Error(context.Background(), "invalid JSON format", fmt.Errorf("invalid JSON format"),
			"vendor", vendor,
			"response_body", string(body),
			"response_size", len(body),
			"response_prefix", string(bytes.TrimSpace(body)[:min(50, len(bytes.TrimSpace(body)))]),
			"component", "ResponseStandardizer",
			"stage", "ValidationInvalidJSON",
		)
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
			logger.Error(context.Background(), "JSON parsing error for array response", err,
				"vendor", vendor,
				"response_body", string(body),
				"response_size", len(body),
				"response_type", "array",
				"component", "ResponseStandardizer",
				"stage", "ValidationArrayParseError",
			)
			return fmt.Errorf("%w: %v", ErrInvalidResponse, err)
		}

		// Check if array contains error objects
		if len(arrayResponse) > 0 {
			if firstItem, ok := arrayResponse[0].(map[string]interface{}); ok {
				if _, hasError := firstItem["error"]; hasError {
					// This is an error response in array format - treat as valid error response
					logger.Debug(context.Background(), "Array error response validation successful",
						"vendor", vendor,
						"complete_response_data", arrayResponse,
						"response_size", len(body),
						"response_type", "array_error",
						"response_body", string(body),
						"component", "ResponseStandardizer",
						"stage", "ValidationArrayError",
					)
					return nil
				}
			}
		}

		// Array response but not an error - this is unexpected for OpenAI-compatible APIs
		logger.Error(context.Background(), "unexpected array response format", fmt.Errorf("unexpected array response format"),
			"vendor", vendor,
			"complete_response_data", arrayResponse,
			"response_body", string(body),
			"response_size", len(body),
			"response_type", "unexpected_array",
			"component", "ResponseStandardizer",
			"stage", "ValidationUnexpectedArray",
		)
		return fmt.Errorf("%w: unexpected array response format", ErrInvalidResponse)
	}

	// Handle object response (normal case)
	if err = json.Unmarshal(body, &responseData); err != nil {
		// Log complete JSON parsing error for object
		logger.Error(context.Background(), "JSON parsing error for object response", err,
			"vendor", vendor,
			"response_body", string(body),
			"response_size", len(body),
			"response_type", "object",
			"component", "ResponseStandardizer",
			"stage", "ValidationObjectParseError",
		)
		return fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}

	// Check if this is an error response first
	if isErrorResponse(responseData) {
		// Log complete successful error response validation
		logger.Debug(context.Background(), "Error response validation successful with complete data",
			"vendor", vendor,
			"complete_response_data", responseData,
			"response_size", len(body),
			"response_type", "error",
			"response_body", string(body),
			"component", "ResponseStandardizer",
			"stage", "ValidationErrorResponse",
		)
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
			logger.Error(context.Background(), "missing required field", fmt.Errorf("missing required field"),
				"missing_field", field,
				"vendor", vendor,
				"complete_response_data", responseData,
				"response_body", string(body),
				"required_fields", requiredFields,
				"has_zero_completion_tokens", hasZeroCompletionTokens,
				"component", "ResponseStandardizer",
				"stage", "ValidationMissingField",
			)
			return fmt.Errorf("%w: missing required field '%s'", ErrInvalidResponse, field)
		}
	}

	// Additional validation: check if choices is empty when we have non-zero completion tokens
	if !hasZeroCompletionTokens {
		if choices, ok := responseData["choices"].([]interface{}); ok && len(choices) == 0 {
			// Log complete empty choices error
			logger.Error(context.Background(), "empty choices array", fmt.Errorf("empty choices array"),
				"vendor", vendor,
				"complete_response_data", responseData,
				"response_body", string(body),
				"has_zero_completion_tokens", hasZeroCompletionTokens,
				"choices_length", 0,
				"component", "ResponseStandardizer",
				"stage", "ValidationEmptyChoices",
			)
			return fmt.Errorf("%w: empty choices array with non-zero completion tokens", ErrInvalidResponse)
		}
	}

	// Log complete successful validation
	logger.Debug(context.Background(), "Response validation successful with complete data",
		"vendor", vendor,
		"complete_response_data", responseData,
		"response_size", len(body),
		"validated_fields", requiredFields,
		"has_zero_completion_tokens", hasZeroCompletionTokens,
		"response_body", string(body),
		"component", "ResponseStandardizer",
		"stage", "ValidationSuccess",
	)
	return nil
}

// setCompliantHeaders sets standardized headers for all responses
func (s *ResponseStandardizer) setCompliantHeaders(w http.ResponseWriter, vendor string, contentLength int, isCompressed bool) {
	// Set standard security and cache headers
	for k, v := range s.standardHeaders {
		w.Header().Set(k, v)
	}

	// Set service identification headers
	w.Header().Set(utils.HeaderServer, utils.ServiceName)
	w.Header().Set(utils.HeaderXPoweredBy, utils.ServicePowered)
	w.Header().Set(utils.HeaderXVendorSource, vendor)

	// Set CORS headers
	w.Header().Set(utils.HeaderAccessControlAllowOrigin, utils.CORSAllowOriginAll)
	w.Header().Set(utils.HeaderAccessControlAllowMethods, utils.CORSAllowMethodsAll)
	w.Header().Set(utils.HeaderAccessControlAllowHeaders, utils.CORSAllowHeadersStd)
	w.Header().Set(utils.HeaderAccessControlExposeHeaders, utils.CORSExposeHeadersStd)

	// Set date header
	w.Header().Set("Date", time.Now().UTC().Format(http.TimeFormat))

	// Note: Request ID is already set by the correlation middleware
	// No need to generate a new one here

	// Set content type for JSON responses
	w.Header().Set(utils.HeaderContentType, utils.ContentTypeJSONUTF8)

	// Set compression headers if applicable
	if isCompressed {
		w.Header().Set(utils.HeaderContentEncoding, utils.AcceptEncodingGzip)
		w.Header().Set(utils.HeaderVary, utils.VaryAcceptEncoding)
	}

	// Set content length if available
	if contentLength > 0 {
		w.Header().Set(utils.HeaderContentLength, strconv.Itoa(contentLength))
	}

	logger.Debug(context.Background(), "Set standardized headers",
		"vendor", vendor,
		"content_length", contentLength,
		"compressed", isCompressed,
		"component", "ResponseStandardizer",
		"stage", "HeadersSet",
	)
}

// processResponseBody handles response body processing
func (s *ResponseStandardizer) processResponseBody(body io.Reader, contentEncoding string, vendor string) ([]byte, error) {
	if contentEncoding == utils.AcceptEncodingGzip {
		logger.Debug(context.Background(), "Decompressing gzip response",
			"vendor", vendor,
			"component", "ResponseStandardizer",
			"stage", "GzipDecompression",
		)
		gzipReader, err := gzip.NewReader(body)
		if err != nil {
			logger.Error(context.Background(), "Failed to create gzip reader", err,
				"vendor", vendor,
				"component", "ResponseStandardizer",
				"stage", "GzipReaderCreation",
			)
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzipReader.Close()
		body = gzipReader
	}

	// Read the entire response body
	responseBody, err := io.ReadAll(body)
	if err != nil {
		logger.Error(context.Background(), "Failed to read response", err,
			"vendor", vendor,
			"component", "ResponseStandardizer",
			"stage", "ResponseReading",
		)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	logger.Debug(context.Background(), "Processed response body",
		"bytes", len(responseBody),
		"vendor", vendor,
		"gzipped", contentEncoding == utils.AcceptEncodingGzip,
		"component", "ResponseStandardizer",
		"stage", "BodyProcessed",
	)
	return responseBody, nil
}

// shouldCompress determines if compression should be applied
func (s *ResponseStandardizer) shouldCompress(r *http.Request) bool {
	if !s.enableGzip {
		return false
	}

	// Check Accept-Encoding header
	acceptEncoding := r.Header.Get(utils.HeaderAcceptEncoding)
	userAgent := r.Header.Get(utils.HeaderUserAgent)

	// Disable compression for known problematic clients
	if strings.Contains(userAgent, "curl/") && !strings.Contains(userAgent, "curl/8") {
		logger.Debug(context.Background(), "Disabling compression for older curl client",
			"user_agent", userAgent,
			"component", "ResponseStandardizer",
			"stage", "CompressionDisabledCurl",
		)
		return false
	}

	// Disable compression for Postman and Insomnia clients
	if strings.Contains(userAgent, "PostmanRuntime") || strings.Contains(strings.ToLower(userAgent), "insomnia") {
		logger.Debug(context.Background(), "Disabling compression for API testing client",
			"user_agent", userAgent,
			"component", "ResponseStandardizer",
			"stage", "CompressionDisabledAPIClient",
		)
		return false
	}

	logger.Debug(context.Background(), "Compression check",
		"accept_encoding", acceptEncoding,
		"user_agent", userAgent,
		"will_compress", strings.Contains(acceptEncoding, utils.AcceptEncodingGzip),
		"component", "ResponseStandardizer",
		"stage", "CompressionCheck",
	)
	return strings.Contains(acceptEncoding, utils.AcceptEncodingGzip)
}

// compressResponseMandatory compresses response data
func (s *ResponseStandardizer) compressResponseMandatory(body []byte) ([]byte, error) {
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)

	_, err := gzipWriter.Write(body)
	if err != nil {
		logger.Error(context.Background(), "Gzip compression error", err,
			"component", "ResponseStandardizer",
			"stage", "GzipCompressionError",
		)
		return body, err
	}

	err = gzipWriter.Close()
	if err != nil {
		logger.Error(context.Background(), "Gzip compression close error", err,
			"component", "ResponseStandardizer",
			"stage", "GzipCompressionCloseError",
		)
		return body, err
	}

	logger.Debug(context.Background(), "Compressed response",
		"original_bytes", len(body),
		"compressed_bytes", buf.Len(),
		"reduction_percent", float64(len(body)-buf.Len())*100/float64(len(body)),
		"component", "ResponseStandardizer",
		"stage", "CompressionComplete",
	)
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
			logger.Error(context.Background(), "Error reading stream", err,
				"component", "APIClient",
				"stage", "StreamReading",
			)
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
		logger.Debug(context.Background(), "Complete streaming chunk processed",
			"vendor", streamProcessor.Vendor,
			"model", streamProcessor.OriginalModel,
			"conversation_id", streamProcessor.ConversationID,
			"original_chunk", string(line),
			"processed_chunk", string(processedChunk),
			"chunk_size_bytes", len(processedChunk),
			"component", "APIClient",
			"stage", "StreamingChunkProcessed",
		)

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
				logger.Error(context.Background(), "Error reading empty line after data", err,
					"component", "APIClient",
					"stage", "StreamEmptyLineReading",
				)
			}
		}
	}
}

// Database logging functionality has been removed

// handleNonStreamingWithHeaders processes non-streaming responses
func (c *APIClient) handleNonStreamingWithHeaders(w http.ResponseWriter, r *http.Request, resp *http.Response, selection *selector.VendorSelection, originalModel string, duration time.Duration, modifiedBody []byte) error {
	logger.Info(r.Context(), "Processing non-streaming request",
		"vendor", selection.Vendor,
		"component", "APIClient",
		"stage", "NonStreamingProcessing",
	)

	// Get complete model object from context if available
	var completeModelObject interface{}
	if vendorModels := r.Context().Value("vendor_models"); vendorModels != nil {
		if models, ok := vendorModels.([]config.VendorModel); ok {
			for _, model := range models {
				if model.Vendor == selection.Vendor && model.Model == selection.Model {
					completeModelObject = model
					break
				}
			}
		}
	}

	// 1. Process response body
	responseBody, err := c.standardizer.processResponseBody(resp.Body, resp.Header.Get(utils.HeaderContentEncoding), selection.Vendor)
	if err != nil {
		logger.Error(r.Context(), "Error processing response body", err,
			"vendor", selection.Vendor,
			"complete_credential_object", selection.Credential, // Full credential object
			"complete_model_object", completeModelObject, // Full model object
			"component", "APIClient",
			"stage", "ResponseBodyProcessing",
		)
		return err
	}

	// Log complete vendor response body immediately after processing
	var vendorResponseBodyForLog interface{}
	if err := json.Unmarshal(responseBody, &vendorResponseBodyForLog); err != nil {
		vendorResponseBodyForLog = string(responseBody)
	}

	logger.Info(r.Context(), "Complete vendor response body received",
		"vendor", selection.Vendor,
		"model", selection.Model,
		"original_model", originalModel,
		"is_streaming", false,
		"complete_credential_object", selection.Credential, // Full credential object
		"complete_model_object", completeModelObject, // Full model object
		"status_code", resp.StatusCode,
		"status", resp.Status,
		"headers", map[string][]string(resp.Header),
		"content_length", resp.ContentLength,
		"body", vendorResponseBodyForLog,
		"body_size_bytes", len(responseBody),
		"content_encoding", resp.Header.Get(utils.HeaderContentEncoding),
		"component", "APIClient",
		"stage", "VendorResponseBodyReceived",
	)

	// 2. Validate response
	if c.standardizer.enableValidation {
		if err := c.standardizer.validateVendorResponse(responseBody, selection.Vendor); err != nil {
			logger.Error(r.Context(), "Vendor response validation failed", err,
				"vendor", selection.Vendor,
				"complete_credential_object", selection.Credential, // Full credential object
				"complete_model_object", completeModelObject, // Full model object
				"component", "APIClient",
				"stage", "ResponseValidation",
			)

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
	modifiedResponse, err := ProcessResponse(responseBody, selection.Vendor, resp.Header.Get(utils.HeaderContentEncoding), originalModel)
	if err != nil {
		logger.Error(r.Context(), "Error processing response", err,
			"vendor", selection.Vendor,
			"complete_credential_object", selection.Credential, // Full credential object
			"complete_model_object", completeModelObject, // Full model object
			"component", "APIClient",
			"stage", "ResponseProcessing",
		)
		return err
	}

	// 4. Determine compression
	shouldCompress := c.standardizer.shouldCompress(r)
	var finalResponse []byte
	var compressErr error

	if shouldCompress {
		finalResponse, compressErr = c.standardizer.compressResponseMandatory(modifiedResponse)
		if compressErr != nil {
			logger.Error(r.Context(), "Error compressing response", compressErr,
				"vendor", selection.Vendor,
				"complete_credential_object", selection.Credential, // Full credential object
				"complete_model_object", completeModelObject, // Full model object
				"component", "APIClient",
				"stage", "ResponseCompression",
			)
			// Fall back to uncompressed if compression fails
			finalResponse = modifiedResponse
			shouldCompress = false
		} else {
			// Set the Content-Encoding header for compressed responses
			w.Header().Set(utils.HeaderContentEncoding, utils.AcceptEncodingGzip)
		}
	} else {
		finalResponse = modifiedResponse
	}

	// 5. Set headers
	c.standardizer.setCompliantHeaders(w, selection.Vendor, len(finalResponse), shouldCompress)

	// 6. Write the response
	_, err = w.Write(finalResponse)
	if err != nil {
		logger.Error(r.Context(), "Error writing response", err,
			"vendor", selection.Vendor,
			"complete_credential_object", selection.Credential, // Full credential object
			"complete_model_object", completeModelObject, // Full model object
			"component", "APIClient",
			"stage", "ResponseWriting",
		)
		return err
	}

	// Log complete final response sent to client
	var finalResponseForLog interface{}
	if err := json.Unmarshal(finalResponse, &finalResponseForLog); err != nil {
		finalResponseForLog = string(finalResponse)
	}

	logger.Info(r.Context(), "Complete final response sent to client",
		"vendor", selection.Vendor,
		"model", selection.Model,
		"original_model", originalModel,
		"is_streaming", false,
		"original_response_size", len(responseBody),
		"modified_response_size", len(modifiedResponse),
		"final_response_size", len(finalResponse),
		"compression_applied", shouldCompress,
		"complete_credential_object", selection.Credential, // Full credential object
		"complete_model_object", completeModelObject, // Full model object
		"body", finalResponseForLog,
		"body_size_bytes", len(finalResponse),
		"headers", map[string][]string(w.Header()),
		"compressed", shouldCompress,
		"content_encoding", w.Header().Get(utils.HeaderContentEncoding),
		"component", "APIClient",
		"stage", "FinalResponseSent",
	)

	// Database logging removed - no longer logging vendor requests

	return nil
}
