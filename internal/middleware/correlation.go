package middleware

import (
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
		if vendor := r.URL.Query().Get("vendor"); vendor != "" {
			ctx = context.WithValue(ctx, logger.VendorKey, vendor)
		}

		// Log request start
		start := time.Now()
		logger.InfoCtx(ctx, "Request started",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.Header.Get("User-Agent"),
			"content_length", r.ContentLength,
		)

		// Create response writer wrapper to capture status
		wrapper := &responseWriterWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Process request
		next.ServeHTTP(wrapper, r.WithContext(ctx))

		// Log request completion
		duration := time.Since(start)
		logger.InfoCtx(ctx, "Request completed",
			"status_code", wrapper.statusCode,
			"duration_ms", duration.Milliseconds(),
			"response_size", wrapper.bytesWritten,
		)
	})
}

// responseWriterWrapper captures response metadata
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriterWrapper) Write(data []byte) (int, error) {
	n, err := w.ResponseWriter.Write(data)
	w.bytesWritten += int64(n)
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
