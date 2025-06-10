package middleware

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aashari/go-generative-api-router/internal/logger"
)

// RequestIDHeader is the header name for request ID
const RequestIDHeader = "X-Request-ID"

// generateRequestID creates a unique request ID
func generateRequestID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if random fails
		return hex.EncodeToString([]byte(time.Now().Format("20060102150405.000")))
	}
	return hex.EncodeToString(bytes)
}

// RequestCorrelationMiddleware adds request correlation ID and enriches context
func RequestCorrelationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Priority order for request ID:
		// 1. CF-Ray header (from Cloudflare)
		// 2. X-Request-ID header (custom header)
		// 3. Generate new ID
		var requestID string
		var requestIDSource string

		// Check CF-Ray first (Cloudflare's ray ID)
		if cfRay := r.Header.Get("CF-Ray"); cfRay != "" {
			requestID = cfRay
			requestIDSource = "cf-ray"
		} else if xRequestID := r.Header.Get(RequestIDHeader); xRequestID != "" {
			// Fall back to X-Request-ID
			requestID = xRequestID
			requestIDSource = "x-request-id"
		} else {
			// Generate new ID if neither header is present
			requestID = generateRequestID()
			requestIDSource = "generated"
		}

		// Set response header
		w.Header().Set(RequestIDHeader, requestID)

		// Create enriched context
		ctx := context.WithValue(r.Context(), logger.RequestIDKey, requestID)

		// Extract vendor from query parameter if present
		vendor := r.URL.Query().Get("vendor")
		if vendor != "" {
			ctx = context.WithValue(ctx, logger.VendorKey, vendor)
		}

		// Check if this is a health check request
		isHealthCheck := r.URL.Path == "/health"

		// Read and capture request body for complete logging
		var requestBody []byte
		var err error
		if r.Body != nil {
			requestBody, err = io.ReadAll(r.Body)
			if err != nil {
				logger.LogError(ctx, "middleware", err, map[string]any{
					"operation": "read_request_body",
					"method":    r.Method,
					"path":      r.URL.Path,
					"headers":   map[string][]string(r.Header),
				})
				http.Error(w, "Failed to read request body", http.StatusBadRequest)
				return
			}
			// Close the original body
			r.Body.Close()

			// Create a new ReadCloser with the captured body for downstream handlers
			r.Body = io.NopCloser(bytes.NewReader(requestBody))
		}

		// Only log request start for non-health check requests or when in debug mode
		start := time.Now()
		if !isHealthCheck {
			// Log request start with new structured format including complete request body
			requestData := map[string]interface{}{
				"request_id":        requestID,
				"request_id_source": requestIDSource,
				"method":            r.Method,
				"path":              r.URL.Path,
				"query_params":      r.URL.Query(),
				"remote_addr":       r.RemoteAddr,
				"user_agent":        r.Header.Get("User-Agent"),
				"content_length":    r.ContentLength,
				"host":              r.Host,
				"request_uri":       r.RequestURI,
				"headers":           map[string][]string(r.Header),
			}

			// Parse body as JSON if it's JSON content type
			// This allows the logger's base64 truncation to work on structured data
			if strings.Contains(r.Header.Get("Content-Type"), "application/json") && len(requestBody) > 0 {
				var bodyData interface{}
				if err := json.Unmarshal(requestBody, &bodyData); err == nil {
					// Successfully parsed as JSON - log the structured data
					// The logger will automatically truncate base64 values
					requestData["body"] = bodyData
				} else {
					// Failed to parse - log as string
					requestData["body"] = string(requestBody)
					requestData["body_parse_error"] = err.Error()
				}
			} else {
				// Non-JSON body - log as string
				requestData["body"] = string(requestBody)
			}

			attributes := map[string]interface{}{
				"component": "middleware",
			}
			if vendor != "" {
				attributes["vendor"] = vendor
			}

			logger.LogWithStructure(ctx, logger.LevelInfo, "Request started", attributes, requestData, nil, nil)
		}

		// Create response writer wrapper to capture status and response data
		wrapper := &responseWriterWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			responseData:   &bytes.Buffer{},
		}

		// Process request
		next.ServeHTTP(wrapper, r.WithContext(ctx))

		// Handle logging based on request type and response status
		duration := time.Since(start)
		
		// For health checks, only log if there's an error or at debug level
		if isHealthCheck {
			if wrapper.statusCode >= 400 {
				// Log health check errors at WARN level
				attributes := map[string]interface{}{
					"component":   "middleware",
					"duration_ms": duration.Milliseconds(),
					"path":        r.URL.Path,
					"status_code": wrapper.statusCode,
				}
				logger.LogWithStructure(ctx, logger.LevelWarn, "Health check failed", attributes, nil, nil, nil)
			} else {
				// Log successful health checks at DEBUG level with minimal information
				logger.LogWithStructure(ctx, logger.LevelDebug, "Health check", map[string]interface{}{
					"duration_ms": duration.Milliseconds(),
					"status":      wrapper.statusCode,
				}, nil, nil, nil)
			}
		} else {
			// Log request completion with new structured format for non-health check requests
			responseData := map[string]interface{}{
				"request_id":     requestID,
				"status_code":    wrapper.statusCode,
				"content_length": wrapper.bytesWritten,
				"headers":        map[string][]string(wrapper.Header()),
			}

			// Only include response body for non-streaming responses
			if wrapper.isStreaming {
				responseData["body"] = "[STREAMING_RESPONSE]"
				responseData["streaming"] = true
			} else if wrapper.responseData != nil && wrapper.responseData.Len() > 0 {
				// Parse response body as JSON if it's JSON content type
				// This allows the logger's base64 truncation to work on structured data
				contentType := wrapper.Header().Get("Content-Type")
				if strings.Contains(contentType, "application/json") {
					var bodyData interface{}
					if err := json.Unmarshal(wrapper.responseData.Bytes(), &bodyData); err == nil {
						// Successfully parsed as JSON - log the structured data
						// The logger will automatically truncate base64 values
						responseData["body"] = bodyData
					} else {
						// Failed to parse - log as string
						responseData["body"] = wrapper.responseData.String()
						responseData["body_parse_error"] = err.Error()
					}
				} else {
					// Non-JSON response - log as string
					responseData["body"] = wrapper.responseData.String()
				}
				responseData["streaming"] = false
			} else {
				responseData["body"] = ""
				responseData["streaming"] = false
			}

			// Include request data for context
			requestData := map[string]interface{}{
				"request_id":        requestID,
				"request_id_source": requestIDSource,
				"method":            r.Method,
				"path":              r.URL.Path,
				"query_params":      r.URL.Query(),
				"remote_addr":       r.RemoteAddr,
				"user_agent":        r.Header.Get("User-Agent"),
				"content_length":    r.ContentLength,
				"host":              r.Host,
				"request_uri":       r.RequestURI,
				"headers":           map[string][]string(r.Header),
			}

			// Parse body as JSON if it's JSON content type
			if strings.Contains(r.Header.Get("Content-Type"), "application/json") && len(requestBody) > 0 {
				var bodyData interface{}
				if err := json.Unmarshal(requestBody, &bodyData); err == nil {
					requestData["body"] = bodyData
				} else {
					requestData["body"] = string(requestBody)
					requestData["body_parse_error"] = err.Error()
				}
			} else {
				requestData["body"] = string(requestBody)
			}

			attributes := map[string]interface{}{
				"component":   "middleware",
				"duration_ms": duration.Milliseconds(),
				"start_time":  start.Format(time.RFC3339),
				"end_time":    time.Now().Format(time.RFC3339),
			}

			logger.LogWithStructure(ctx, logger.LevelInfo, "Request completed", attributes, requestData, responseData, nil)
		}
	})
}

// responseWriterWrapper captures response metadata
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
	responseData *bytes.Buffer
	isStreaming  bool
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode

	// Check if this is a streaming response
	contentType := w.Header().Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") {
		w.isStreaming = true
		// Don't buffer streaming responses
		w.responseData = nil
	}

	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriterWrapper) Write(data []byte) (int, error) {
	n, err := w.ResponseWriter.Write(data)
	w.bytesWritten += int64(n)

	// Only capture response data for non-streaming responses
	if w.responseData != nil && !w.isStreaming {
		w.responseData.Write(data[:n]) // Only write successfully written bytes
	}

	return n, err
}
