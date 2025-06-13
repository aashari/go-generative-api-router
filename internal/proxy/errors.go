package proxy

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// VendorValidationError wraps validation errors with vendor information
type VendorValidationError struct {
	Vendor       string
	OriginalErr  error
	MissingField string
}

// Error implements the error interface
func (e *VendorValidationError) Error() string {
	if e.MissingField != "" {
		return fmt.Sprintf("vendor %s validation failed: missing required field '%s'", e.Vendor, e.MissingField)
	}
	return fmt.Sprintf("vendor %s validation failed: %v", e.Vendor, e.OriginalErr)
}

// Unwrap allows errors.Is/As to work with wrapped errors
func (e *VendorValidationError) Unwrap() error {
	return e.OriginalErr
}

// VendorAPIError wraps API errors from vendors (quota, rate limits, etc.)
type VendorAPIError struct {
	Vendor     string
	StatusCode int
	ErrorType  string
	Message    string
	Retriable  bool
}

// Error implements the error interface
func (e *VendorAPIError) Error() string {
	return fmt.Sprintf("vendor %s API error [%d]: %s - %s", e.Vendor, e.StatusCode, e.ErrorType, e.Message)
}

// IsRetriable implements the RetryableError interface
func (e *VendorAPIError) IsRetriable() bool {
	return e.Retriable
}

// IsRetriableValidationError checks if the error is retriable (missing choices from Gemini)
func IsRetriableValidationError(err error) bool {
	var vendorErr *VendorValidationError
	if errors.As(err, &vendorErr) {
		// Only retry if it's Gemini with missing 'choices' field
		return vendorErr.Vendor == "gemini" && vendorErr.MissingField == "choices"
	}
	return false
}

// IsRetriableAPIError checks if the API error is retriable with backoff
func IsRetriableAPIError(err error) bool {
	var apiErr *VendorAPIError
	if errors.As(err, &apiErr) {
		return apiErr.Retriable
	}
	return false
}

// IsQuotaError checks if the error is specifically a quota/rate limit error
func IsQuotaError(err error) bool {
	var apiErr *VendorAPIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorType == "insufficient_quota" ||
			apiErr.ErrorType == "rate_limit_exceeded" ||
			apiErr.StatusCode == http.StatusTooManyRequests
	}
	return false
}

// ParseVendorError analyzes vendor response and creates appropriate error types
func ParseVendorError(vendor string, statusCode int, responseBody []byte) error {
	// For successful responses, no error
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	// Try to parse JSON error response
	if len(responseBody) > 0 {
		// Simple JSON parsing without importing json package
		bodyStr := string(responseBody)

		// Check for common error patterns
		if strings.Contains(bodyStr, "insufficient_quota") {
			return &VendorAPIError{
				Vendor:     vendor,
				StatusCode: statusCode,
				ErrorType:  "insufficient_quota",
				Message:    "API quota exceeded",
				Retriable:  true, // Quota errors should be retried with backoff
			}
		}

		if strings.Contains(bodyStr, "rate_limit") || statusCode == http.StatusTooManyRequests {
			return &VendorAPIError{
				Vendor:     vendor,
				StatusCode: statusCode,
				ErrorType:  "rate_limit_exceeded",
				Message:    "Rate limit exceeded",
				Retriable:  true, // Rate limits should be retried with backoff
			}
		}
	}

	// Handle HTTP status codes
	switch statusCode {
	case http.StatusTooManyRequests: // 429
		return &VendorAPIError{
			Vendor:     vendor,
			StatusCode: statusCode,
			ErrorType:  "rate_limit_exceeded",
			Message:    "Too many requests",
			Retriable:  true,
		}
	case http.StatusInternalServerError, // 500
		http.StatusBadGateway,         // 502
		http.StatusServiceUnavailable, // 503
		http.StatusGatewayTimeout:     // 504
		return &VendorAPIError{
			Vendor:     vendor,
			StatusCode: statusCode,
			ErrorType:  "server_error",
			Message:    fmt.Sprintf("Server error: %d", statusCode),
			Retriable:  true, // Server errors should be retried
		}
	case http.StatusUnauthorized: // 401
		return &VendorAPIError{
			Vendor:     vendor,
			StatusCode: statusCode,
			ErrorType:  "authentication_error",
			Message:    "Invalid API key or authentication failed",
			Retriable:  false, // Auth errors should not be retried
		}
	case http.StatusForbidden: // 403
		return &VendorAPIError{
			Vendor:     vendor,
			StatusCode: statusCode,
			ErrorType:  "permission_error",
			Message:    "Access forbidden",
			Retriable:  false, // Permission errors should not be retried
		}
	case http.StatusBadRequest: // 400
		return &VendorAPIError{
			Vendor:     vendor,
			StatusCode: statusCode,
			ErrorType:  "invalid_request",
			Message:    "Bad request",
			Retriable:  false, // Bad requests should not be retried
		}
	default:
		return &VendorAPIError{
			Vendor:     vendor,
			StatusCode: statusCode,
			ErrorType:  "unknown_error",
			Message:    fmt.Sprintf("Unknown error: %d", statusCode),
			Retriable:  statusCode >= 500, // Only retry server errors
		}
	}
}
