package middleware

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
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

		// Log request start with complete data
		start := time.Now()
		logger.LogMultipleData(ctx, logger.LevelInfo, "Complete request started", map[string]any{
			"request_details": map[string]any{
				"method":         r.Method,
				"path":           r.URL.Path,
				"query_params":   r.URL.Query(),
				"remote_addr":    r.RemoteAddr,
				"user_agent":     r.Header.Get("User-Agent"),
				"content_length": r.ContentLength,
				"host":           r.Host,
				"request_uri":    r.RequestURI,
			},
			"headers_complete": map[string][]string(r.Header),
			"context_data": map[string]any{
				"request_id": requestID,
				"vendor":     vendor,
			},
		})

		// Create response writer wrapper to capture status and response data
		wrapper := &responseWriterWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			responseData:   &bytes.Buffer{},
		}

		// Process request
		next.ServeHTTP(wrapper, r.WithContext(ctx))

		// Log request completion with complete response data
		duration := time.Since(start)
		logger.LogMultipleData(ctx, logger.LevelInfo, "Complete request completed", map[string]any{
			"response_details": map[string]any{
				"status_code":   wrapper.statusCode,
				"duration_ms":   duration.Milliseconds(),
				"response_size": wrapper.bytesWritten,
				"response_body": wrapper.responseData.String(),
			},
			"response_headers": map[string][]string(wrapper.Header()),
			"timing": map[string]any{
				"start_time": start,
				"end_time":   time.Now(),
				"duration":   duration,
			},
		})
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

// VendorContextMiddleware enriches context with vendor information from request body
func VendorContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This middleware can be enhanced to extract vendor/model from request body
		// For now, it passes through to the next handler
		next.ServeHTTP(w, r)
	})
}
