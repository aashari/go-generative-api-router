package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aashari/go-generative-api-router/internal/logger"
	"github.com/aashari/go-generative-api-router/internal/utils"
)

// Request and Correlation ID tracking with BrainyBuddy priority cascade

// TrackingIDSources contains information about where tracking IDs came from
type TrackingIDSources struct {
	RequestIDSource     string `json:"request_id_source"`
	CorrelationIDSource string `json:"correlation_id_source"`
}

// RequestCorrelationMiddleware implements BrainyBuddy-style tracking with priority cascade
func RequestCorrelationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Tracking ID extraction with priority cascade
		requestID, correlationID, sources := extractTrackingIDsWithPriority(r)

		// Set response headers
		w.Header().Set(RequestIDHeader, requestID)
		w.Header().Set(CorrelationIDHeader, correlationID)

		// Create enriched context
		ctx := context.WithValue(r.Context(), logger.RequestIDKey, requestID)
		ctx = context.WithValue(ctx, logger.CorrelationIDKey, correlationID)
		ctx = logger.WithComponent(ctx, logger.ComponentNames.Middleware)

		// Log tracking ID sources for debugging
		logger.Debug(logger.WithStage(ctx, logger.LogStages.TrackingSetup),
			"Generated tracking IDs",
			"request_id_source", sources.RequestIDSource,
			"correlation_id_source", sources.CorrelationIDSource,
		)

		// Health check handling with conditional logging
		if r.URL.Path == "/health" {
			handleHealthCheck(ctx, w, r, next)
			return
		}

		// General request handling with structured logging
		handleGeneralRequest(ctx, w, r, next)
	})
}

// extractTrackingIDsWithPriority implements BrainyBuddy-style priority cascade
func extractTrackingIDsWithPriority(r *http.Request) (requestID, correlationID string, sources TrackingIDSources) {
	// Priority cascade for Request ID (matching BrainyBuddy logic)

	// 1. Client-provided X-Request-ID (highest priority)
	if clientRequestID := r.Header.Get(utils.HeaderRequestID); clientRequestID != "" {
		requestID = clientRequestID
		sources.RequestIDSource = "client-x-request-id"
	} else if cfRay := r.Header.Get(utils.HeaderCloudFlareRay); cfRay != "" {
		// 2. CloudFlare Ray ID
		requestID = cfRay
		sources.RequestIDSource = "cloudflare-ray"
	} else if xForwardedFor := r.Header.Get(utils.HeaderXForwardedFor); xForwardedFor != "" {
		// 3. X-Forwarded-For based ID (for load balancer scenarios)
		requestID = generateHashFromIP(xForwardedFor)
		sources.RequestIDSource = "x-forwarded-for-hash"
	} else {
		// 4. Generated fallback
		requestID = utils.GenerateRequestID()
		sources.RequestIDSource = "generated-uuid"
	}

	// Priority cascade for Correlation ID

	// 1. Client-provided X-Correlation-ID (highest priority)
	if clientCorrelationID := r.Header.Get(utils.HeaderCorrelationID); clientCorrelationID != "" {
		correlationID = clientCorrelationID
		sources.CorrelationIDSource = "client-x-correlation-id"
	} else if cfRay := r.Header.Get(utils.HeaderCloudFlareRay); cfRay != "" {
		// 2. CloudFlare Ray ID for correlation
		correlationID = cfRay
		sources.CorrelationIDSource = "cloudflare-ray"
	} else {
		// 3. Use request ID as fallback
		correlationID = requestID
		sources.CorrelationIDSource = "request-id-fallback"
	}

	return requestID, correlationID, sources
}

// generateHashFromIP creates a deterministic hash from IP for tracking
func generateHashFromIP(ipHeader string) string {
	ip := strings.Split(ipHeader, ",")[0] // Get first IP from X-Forwarded-For
	ip = strings.TrimSpace(ip)
	hash := fmt.Sprintf("%x", ip)
	if len(hash) > 16 {
		hash = hash[:16]
	}
	return hash + "-" + fmt.Sprintf("%d", time.Now().Unix()%10000)
}

// handleHealthCheck processes health checks with conditional logging
func handleHealthCheck(ctx context.Context, w http.ResponseWriter, r *http.Request, next http.Handler) {
	start := time.Now()

	// Create response wrapper
	wrapper := &responseWriterWrapper{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		body:           &bytes.Buffer{},
	}

	// Process health check
	next.ServeHTTP(wrapper, r.WithContext(ctx))

	duration := time.Since(start)

	// Only log health checks when there are errors or warnings
	if wrapper.statusCode >= 400 {
		// Log error health check
		logger.Error(logger.WithStage(ctx, logger.LogStages.HealthCheckFailed),
			"Health check failed", fmt.Errorf("status code: %d", wrapper.statusCode),
			"response", map[string]interface{}{
				"status_code": wrapper.statusCode,
				"duration_ms": duration.Milliseconds(),
				"body":        wrapper.body.String(),
			},
		)
	} else if wrapper.statusCode == http.StatusOK {
		// Check response body for warnings
		if wrapper.body.Len() > 0 {
			var healthData map[string]interface{}
			if err := json.Unmarshal(wrapper.body.Bytes(), &healthData); err == nil {
				if status, ok := healthData["status"].(string); ok && status != "healthy" {
					// Log degraded health
					logger.Warn(logger.WithStage(ctx, logger.LogStages.HealthCheckWarning),
						"Health check shows degraded status",
						"response", map[string]interface{}{
							"status_code": wrapper.statusCode,
							"duration_ms": duration.Milliseconds(),
							"health_data": healthData,
						},
					)
				}
			}
		}
	}

	// Copy response to original writer
	for key, values := range wrapper.Header() {
		w.Header().Del(key) // Clear any existing values for this header
		for _, value := range values {
			w.Header().Add(key, value) // Add all values for this header
		}
	}
	w.WriteHeader(wrapper.statusCode)
	w.Write(wrapper.body.Bytes())
}

// handleGeneralRequest processes general requests with structured logging
func handleGeneralRequest(ctx context.Context, w http.ResponseWriter, r *http.Request, next http.Handler) {
	start := time.Now()

	// Read and preserve request body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error(logger.WithStage(ctx, logger.LogStages.Error),
			"Failed to read request body", err)
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	r.Body.Close()
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// Log structured request
	logStructuredRequest(ctx, r, bodyBytes)

	// Process request
	wrapper := &responseWriterWrapper{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		body:           &bytes.Buffer{},
	}

	next.ServeHTTP(wrapper, r.WithContext(ctx))

	// Log structured response
	duration := time.Since(start)
	logStructuredResponse(ctx, wrapper, duration)

	// Copy response to original writer
	for key, values := range wrapper.Header() {
		w.Header().Del(key) // Clear any existing values for this header
		for _, value := range values {
			w.Header().Add(key, value) // Add all values for this header
		}
	}
	w.WriteHeader(wrapper.statusCode)
	w.Write(wrapper.body.Bytes())
}

// logStructuredRequest logs incoming request with nested structure
func logStructuredRequest(ctx context.Context, r *http.Request, body []byte) {
	// Create structured request data
	requestData := map[string]interface{}{
		"method":     r.Method,
		"endpoint":   r.URL.Path,
		"user_agent": r.Header.Get(utils.HeaderUserAgent),
		"client_ip":  getClientIP(r),
		"headers":    utils.SanitizeHeaders(r.Header),
	}

	// Add body if present
	if len(body) > 0 {
		var bodyData interface{}
		if err := json.Unmarshal(body, &bodyData); err == nil {
			requestData["body"] = utils.TruncateBase64InData(bodyData)
		} else {
			requestData["body"] = "Non-JSON body omitted"
		}
	}

	logger.Info(logger.WithStage(ctx, logger.LogStages.RequestReceived),
		"Incoming request",
		"request", requestData,
	)
}

// logStructuredResponse logs outgoing response with nested structure
func logStructuredResponse(ctx context.Context, w *responseWriterWrapper, duration time.Duration) {
	responseData := map[string]interface{}{
		"status_code":    w.statusCode,
		"duration_ms":    duration.Milliseconds(),
		"content_length": w.body.Len(),
		"headers":        utils.SanitizeHeaders(w.Header()),
	}

	// Add response body if available
	if w.body.Len() > 0 && !w.isStreaming {
		var bodyData interface{}
		if err := json.Unmarshal(w.body.Bytes(), &bodyData); err == nil {
			responseData["body"] = utils.TruncateBase64InData(bodyData)
		}
	} else if w.isStreaming {
		responseData["body"] = "[streaming response]"
	}

	stage := logger.LogStages.RequestCompleted
	if w.statusCode >= 400 {
		stage = logger.LogStages.RequestFailed
	}

	logger.Info(logger.WithStage(ctx, stage),
		"Request completed",
		"response", responseData,
	)
}

// getClientIP extracts client IP with priority cascade
func getClientIP(r *http.Request) string {
	// Priority: X-Forwarded-For > X-Real-IP > CF-Connecting-IP > RemoteAddr
	if forwardedFor := r.Header.Get(utils.HeaderXForwardedFor); forwardedFor != "" {
		return strings.Split(forwardedFor, ",")[0]
	}
	if realIP := r.Header.Get(utils.HeaderXRealIP); realIP != "" {
		return realIP
	}
	if cfIP := r.Header.Get(utils.HeaderCFConnectingIP); cfIP != "" {
		return cfIP
	}
	return r.RemoteAddr
}

// responseWriterWrapper captures response data for logging
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode    int
	body          *bytes.Buffer
	isStreaming   bool
	headerWritten bool
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	// Don't write header to original writer yet - let middleware handle it
	w.headerWritten = true
}

func (w *responseWriterWrapper) Write(data []byte) (int, error) {
	// Check if it's a streaming response
	if strings.Contains(w.Header().Get(utils.HeaderContentType), utils.ContentTypeEventStream) {
		w.isStreaming = true
		// For streaming, write directly to original writer
		if !w.headerWritten {
			w.WriteHeader(http.StatusOK)
		}
		w.ResponseWriter.WriteHeader(w.statusCode)
		return w.ResponseWriter.Write(data)
	}

	// For non-streaming responses, only capture data for logging
	// Don't write to original writer yet - let middleware handle it
	if w.body.Len() < 10240 { // Limit to 10KB
		w.body.Write(data)
	}

	return len(data), nil // Return success without actually writing yet
}

// Flush implements http.Flusher interface for streaming support
func (w *responseWriterWrapper) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Header constants
const (
	RequestIDHeader     = utils.HeaderRequestID
	CorrelationIDHeader = utils.HeaderCorrelationID
)

