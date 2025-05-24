package errors

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aashari/go-generative-api-router/internal/logger"
)

// ErrorType represents different types of errors
type ErrorType string

const (
	ErrorTypeValidation     ErrorType = "validation_error"
	ErrorTypeAuthentication ErrorType = "authentication_error"
	ErrorTypeAuthorization  ErrorType = "authorization_error"
	ErrorTypeNotFound       ErrorType = "not_found_error"
	ErrorTypeInternal       ErrorType = "internal_error"
	ErrorTypeExternal       ErrorType = "external_error"
	ErrorTypeConfiguration  ErrorType = "configuration_error"
)

// APIError represents a structured API error
type APIError struct {
	Type    ErrorType `json:"type"`
	Message string    `json:"message"`
	Code    string    `json:"code,omitempty"`
	Details string    `json:"details,omitempty"`
}

// Error implements the error interface
func (e *APIError) Error() string {
	return e.Message
}

// ErrorResponse represents the JSON error response format
type ErrorResponse struct {
	Error APIError `json:"error"`
}

// NewAPIError creates a new APIError
func NewAPIError(errorType ErrorType, message string) *APIError {
	return &APIError{
		Type:    errorType,
		Message: message,
	}
}

// NewAPIErrorWithCode creates a new APIError with a code
func NewAPIErrorWithCode(errorType ErrorType, message, code string) *APIError {
	return &APIError{
		Type:    errorType,
		Message: message,
		Code:    code,
	}
}

// NewAPIErrorWithDetails creates a new APIError with details
func NewAPIErrorWithDetails(errorType ErrorType, message, details string) *APIError {
	return &APIError{
		Type:    errorType,
		Message: message,
		Details: details,
	}
}

// HandleError writes a standardized error response to the HTTP response writer
func HandleError(w http.ResponseWriter, err error, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	var apiError *APIError

	// Check if it's already an APIError
	if ae, ok := err.(*APIError); ok {
		apiError = ae
	} else {
		// Convert regular error to APIError
		apiError = inferErrorType(err, statusCode)
	}

	response := ErrorResponse{Error: *apiError}

	if jsonBytes, jsonErr := json.Marshal(response); jsonErr == nil {
		w.Write(jsonBytes)
	} else {
		// Fallback if JSON marshaling fails
		logger.Error("Error marshaling error response", "error", jsonErr)
		w.Write([]byte(`{"error":{"type":"internal_error","message":"Internal server error"}}`))
	}

	// Log the error for debugging
	logger.Error("API Error",
		"status_code", statusCode,
		"error_type", string(apiError.Type),
		"message", apiError.Message,
	)
}

// inferErrorType attempts to infer the error type based on the error message and status code
func inferErrorType(err error, statusCode int) *APIError {
	message := err.Error()

	switch statusCode {
	case http.StatusBadRequest:
		return NewAPIError(ErrorTypeValidation, message)
	case http.StatusUnauthorized:
		return NewAPIError(ErrorTypeAuthentication, message)
	case http.StatusForbidden:
		return NewAPIError(ErrorTypeAuthorization, message)
	case http.StatusNotFound:
		return NewAPIError(ErrorTypeNotFound, message)
	case http.StatusInternalServerError:
		return NewAPIError(ErrorTypeInternal, message)
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return NewAPIError(ErrorTypeExternal, message)
	default:
		return NewAPIError(ErrorTypeInternal, message)
	}
}

// Common error constructors for convenience

// NewValidationError creates a validation error
func NewValidationError(message string) *APIError {
	return NewAPIError(ErrorTypeValidation, message)
}

// NewAuthenticationError creates an authentication error
func NewAuthenticationError(message string) *APIError {
	return NewAPIError(ErrorTypeAuthentication, message)
}

// NewAuthorizationError creates an authorization error
func NewAuthorizationError(message string) *APIError {
	return NewAPIError(ErrorTypeAuthorization, message)
}

// NewNotFoundError creates a not found error
func NewNotFoundError(message string) *APIError {
	return NewAPIError(ErrorTypeNotFound, message)
}

// NewInternalError creates an internal error
func NewInternalError(message string) *APIError {
	return NewAPIError(ErrorTypeInternal, message)
}

// NewExternalError creates an external service error
func NewExternalError(message string) *APIError {
	return NewAPIError(ErrorTypeExternal, message)
}

// NewConfigurationError creates a configuration error
func NewConfigurationError(message string) *APIError {
	return NewAPIError(ErrorTypeConfiguration, message)
}

// Validation helpers

// ValidateRequired checks if a required field is present
func ValidateRequired(value, fieldName string) *APIError {
	if value == "" {
		return NewValidationError(fmt.Sprintf("Field '%s' is required", fieldName))
	}
	return nil
}

// ValidateNonEmpty checks if a slice is non-empty
func ValidateNonEmpty(slice interface{}, fieldName string) *APIError {
	switch s := slice.(type) {
	case []string:
		if len(s) == 0 {
			return NewValidationError(fmt.Sprintf("Field '%s' cannot be empty", fieldName))
		}
	case []interface{}:
		if len(s) == 0 {
			return NewValidationError(fmt.Sprintf("Field '%s' cannot be empty", fieldName))
		}
	default:
		return NewValidationError(fmt.Sprintf("Field '%s' has invalid type", fieldName))
	}
	return nil
}
