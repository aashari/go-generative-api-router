package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aashari/go-generative-api-router/internal/logger"
	"github.com/aashari/go-generative-api-router/internal/utils"
)

// HTTPLogger provides structured logging for HTTP requests to vendors
type HTTPLogger struct {
	component string
}

// NewHTTPLogger creates a new HTTP logger
func NewHTTPLogger(component string) *HTTPLogger {
	return &HTTPLogger{
		component: component,
	}
}

// LogRequest logs outgoing HTTP request to vendor
func (h *HTTPLogger) LogRequest(ctx context.Context, req *http.Request) {
	ctx = logger.WithComponent(ctx, h.component)
	ctx = logger.WithStage(ctx, logger.LogStages.VendorRequest)

	// Extract request data
	requestData := map[string]interface{}{
		"method":  req.Method,
		"url":     req.URL.String(),
		"headers": extractSafeHeaders(req.Header),
	}

	// Add body if present
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err == nil {
			req.Body = io.NopCloser(bytes.NewBuffer(body))

			var bodyData interface{}
			if json.Unmarshal(body, &bodyData) == nil {
				requestData["body"] = utils.TruncateBase64InData(bodyData)
			} else {
				requestData["body"] = string(body)
			}
		}
	}

	logger.Info(ctx, "Sending vendor request", "request", utils.TruncateStringsInData(requestData))
}

// LogResponse logs HTTP response from vendor
func (h *HTTPLogger) LogResponse(ctx context.Context, resp *http.Response, duration time.Duration) {
	ctx = logger.WithComponent(ctx, h.component)

	stage := logger.LogStages.VendorResponse
	if resp.StatusCode >= 400 {
		stage = logger.LogStages.VendorError
	}
	ctx = logger.WithStage(ctx, stage)

	// Extract response data
	responseData := map[string]interface{}{
		"status_code": resp.StatusCode,
		"duration_ms": duration.Milliseconds(),
		"headers":     extractSafeHeaders(resp.Header),
	}

	// Add response body if available
	if resp.Body != nil {
		body, err := io.ReadAll(resp.Body)
		if err == nil {
			resp.Body = io.NopCloser(bytes.NewBuffer(body))

			var bodyData interface{}
			if json.Unmarshal(body, &bodyData) == nil {
				responseData["body"] = utils.TruncateBase64InData(bodyData)
			} else {
				responseData["body"] = string(body)
			}
		}
	}

	message := "Vendor response received"
	if resp.StatusCode >= 400 {
		message = "Vendor error response received"
	}

	logger.Info(ctx, message, "response", utils.TruncateStringsInData(responseData))
}

// LogError logs HTTP request errors
func (h *HTTPLogger) LogError(ctx context.Context, req *http.Request, err error) {
	ctx = logger.WithComponent(ctx, h.component)
	ctx = logger.WithStage(ctx, logger.LogStages.VendorError)

	requestData := map[string]interface{}{
		"method": req.Method,
		"url":    req.URL.String(),
	}

	logger.Error(ctx, "Vendor request failed", err, "request", utils.TruncateStringsInData(requestData))
}

// LogRequestWithTiming logs request start with timing info
func (h *HTTPLogger) LogRequestWithTiming(ctx context.Context, req *http.Request, vendor, model string) time.Time {
	ctx = logger.WithComponent(ctx, h.component)
	ctx = logger.WithStage(ctx, logger.LogStages.VendorRequest)

	// Extract request data
	requestData := map[string]interface{}{
		"method":  req.Method,
		"url":     req.URL.String(),
		"vendor":  vendor,
		"model":   model,
		"headers": extractSafeHeaders(req.Header),
	}

	// Add tracking headers info
	if requestID := req.Header.Get(utils.HeaderRequestID); requestID != "" {
		requestData["forwarded_request_id"] = requestID
	}
	if correlationID := req.Header.Get(utils.HeaderCorrelationID); correlationID != "" {
		requestData["forwarded_correlation_id"] = correlationID
	}

	// Add body if present
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err == nil {
			req.Body = io.NopCloser(bytes.NewBuffer(body))

			var bodyData interface{}
			if json.Unmarshal(body, &bodyData) == nil {
				requestData["body"] = utils.TruncateBase64InData(bodyData)
			} else {
				requestData["body"] = string(body)
			}
		}
	}

	logger.Info(ctx, "Sending vendor request with tracking", "request", utils.TruncateStringsInData(requestData))
	return time.Now()
}

// LogResponseWithTiming logs response with complete timing info
func (h *HTTPLogger) LogResponseWithTiming(ctx context.Context, resp *http.Response, vendor, model string, start time.Time) {
	duration := time.Since(start)

	ctx = logger.WithComponent(ctx, h.component)
	stage := logger.LogStages.VendorResponse
	if resp.StatusCode >= 400 {
		stage = logger.LogStages.VendorError
	}
	ctx = logger.WithStage(ctx, stage)

	// Extract response data
	responseData := map[string]interface{}{
		"status_code": resp.StatusCode,
		"duration_ms": duration.Milliseconds(),
		"vendor":      vendor,
		"model":       model,
		"headers":     extractSafeHeaders(resp.Header),
	}

	// Add response body if available (limited for performance)
	if resp.Body != nil && resp.ContentLength < 10240 { // Only log small responses
		body, err := io.ReadAll(resp.Body)
		if err == nil {
			resp.Body = io.NopCloser(bytes.NewBuffer(body))

			var bodyData interface{}
			if json.Unmarshal(body, &bodyData) == nil {
				responseData["body"] = utils.TruncateBase64InData(bodyData)
			} else if len(body) < 1024 { // Only log small text responses
				responseData["body"] = string(body)
			} else {
				responseData["body_summary"] = fmt.Sprintf("Large response (%d bytes)", len(body))
			}
		}
	}

	message := "Vendor response received with timing"
	if resp.StatusCode >= 400 {
		message = "Vendor error response received with timing"
	}

	logger.Info(ctx, message, "response", utils.TruncateStringsInData(responseData))
}

// extractSafeHeaders extracts headers while filtering sensitive ones
func extractSafeHeaders(headers http.Header) map[string]string {
	result := make(map[string]string)
	sensitiveHeaders := map[string]bool{
		"authorization": true,
		"x-api-key":     true,
		"api-key":       true,
		"bearer":        true,
		"token":         true,
	}

	for key, values := range headers {
		if len(values) > 0 {
			lowerKey := fmt.Sprintf("%s", key)
			if !sensitiveHeaders[lowerKey] {
				result[key] = values[0]
			} else {
				result[key] = "[REDACTED]"
			}
		}
	}
	return result
}

// ExtractVendorFromContext extracts vendor information from context
func ExtractVendorFromContext(ctx context.Context) (vendor, model string) {
	if v := ctx.Value("vendor"); v != nil {
		vendor = v.(string)
	}
	if m := ctx.Value("model"); m != nil {
		model = m.(string)
	}
	return vendor, model
}
