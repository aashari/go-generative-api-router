package middleware

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
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
		// Get or generate request ID
		requestID := r.Header.Get(RequestIDHeader)
		if requestID == "" {
			requestID = generateRequestID()
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

		// Log request start with new structured format including complete request body
		start := time.Now()
		requestData := map[string]interface{}{
			"request_id":     requestID,
			"method":         r.Method,
			"path":           r.URL.Path,
			"query_params":   r.URL.Query(),
			"remote_addr":    r.RemoteAddr,
			"user_agent":     r.Header.Get("User-Agent"),
			"content_length": r.ContentLength,
			"host":           r.Host,
			"request_uri":    r.RequestURI,
			"headers":        map[string][]string(r.Header),
			"body":           string(requestBody), // Complete request body logging
		}

		attributes := map[string]interface{}{
			"component": "middleware",
		}
		if vendor != "" {
			attributes["vendor"] = vendor
		}

		logger.LogWithStructure(ctx, logger.LevelInfo, "Request started", attributes, requestData, nil, nil)

		// Create response writer wrapper to capture status and response data
		wrapper := &responseWriterWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			responseData:   &bytes.Buffer{},
		}

		// Process request
		next.ServeHTTP(wrapper, r.WithContext(ctx))

		// Log request completion with new structured format
		duration := time.Since(start)
		responseData := map[string]interface{}{
			"request_id":     requestID,
			"status_code":    wrapper.statusCode,
			"content_length": wrapper.bytesWritten,
			"headers":        map[string][]string(wrapper.Header()),
			"body":           wrapper.responseData.String(),
		}

		attributes = map[string]interface{}{
			"component":   "middleware",
			"duration_ms": duration.Milliseconds(),
			"start_time":  start.Format(time.RFC3339),
			"end_time":    time.Now().Format(time.RFC3339),
		}

		logger.LogWithStructure(ctx, logger.LevelInfo, "Request completed", attributes, requestData, responseData, nil)
	})
}

// responseWriterWrapper captures response metadata
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
	responseData *bytes.Buffer
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriterWrapper) Write(data []byte) (int, error) {
	n, err := w.ResponseWriter.Write(data)
	w.bytesWritten += int64(n)

	// Capture response data for complete logging
	if w.responseData != nil {
		w.responseData.Write(data[:n]) // Only write successfully written bytes
	}

	return n, err
}
