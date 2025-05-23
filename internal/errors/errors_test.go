package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAPIError(t *testing.T) {
	err := NewAPIError(ErrorTypeValidation, "test message")

	assert.Equal(t, ErrorTypeValidation, err.Type)
	assert.Equal(t, "test message", err.Message)
	assert.Empty(t, err.Code)
	assert.Empty(t, err.Details)
}

func TestNewAPIErrorWithCode(t *testing.T) {
	err := NewAPIErrorWithCode(ErrorTypeValidation, "test message", "TEST_CODE")

	assert.Equal(t, ErrorTypeValidation, err.Type)
	assert.Equal(t, "test message", err.Message)
	assert.Equal(t, "TEST_CODE", err.Code)
	assert.Empty(t, err.Details)
}

func TestNewAPIErrorWithDetails(t *testing.T) {
	err := NewAPIErrorWithDetails(ErrorTypeValidation, "test message", "detailed info")

	assert.Equal(t, ErrorTypeValidation, err.Type)
	assert.Equal(t, "test message", err.Message)
	assert.Empty(t, err.Code)
	assert.Equal(t, "detailed info", err.Details)
}

func TestAPIErrorImplementsError(t *testing.T) {
	err := NewAPIError(ErrorTypeValidation, "test message")

	// Should implement error interface
	var _ error = err
	assert.Equal(t, "test message", err.Error())
}

func TestHandleError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		statusCode     int
		expectedType   ErrorType
		expectedStatus int
	}{
		{
			name:           "api_error",
			err:            NewValidationError("validation failed"),
			statusCode:     http.StatusBadRequest,
			expectedType:   ErrorTypeValidation,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "regular_error_400",
			err:            fmt.Errorf("bad request"),
			statusCode:     http.StatusBadRequest,
			expectedType:   ErrorTypeValidation,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "regular_error_401",
			err:            fmt.Errorf("unauthorized"),
			statusCode:     http.StatusUnauthorized,
			expectedType:   ErrorTypeAuthentication,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "regular_error_500",
			err:            fmt.Errorf("internal error"),
			statusCode:     http.StatusInternalServerError,
			expectedType:   ErrorTypeInternal,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			HandleError(w, tt.err, tt.statusCode)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			var response ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedType, response.Error.Type)
			assert.NotEmpty(t, response.Error.Message)
		})
	}
}

func TestInferErrorType(t *testing.T) {
	tests := []struct {
		statusCode   int
		expectedType ErrorType
	}{
		{http.StatusBadRequest, ErrorTypeValidation},
		{http.StatusUnauthorized, ErrorTypeAuthentication},
		{http.StatusForbidden, ErrorTypeAuthorization},
		{http.StatusNotFound, ErrorTypeNotFound},
		{http.StatusInternalServerError, ErrorTypeInternal},
		{http.StatusBadGateway, ErrorTypeExternal},
		{http.StatusServiceUnavailable, ErrorTypeExternal},
		{http.StatusGatewayTimeout, ErrorTypeExternal},
		{http.StatusTeapot, ErrorTypeInternal}, // Unknown status code
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%d", tt.statusCode), func(t *testing.T) {
			err := fmt.Errorf("test error")
			apiErr := inferErrorType(err, tt.statusCode)

			assert.Equal(t, tt.expectedType, apiErr.Type)
			assert.Equal(t, "test error", apiErr.Message)
		})
	}
}

func TestConvenienceConstructors(t *testing.T) {
	tests := []struct {
		name         string
		constructor  func(string) *APIError
		expectedType ErrorType
	}{
		{"validation", NewValidationError, ErrorTypeValidation},
		{"authentication", NewAuthenticationError, ErrorTypeAuthentication},
		{"authorization", NewAuthorizationError, ErrorTypeAuthorization},
		{"not_found", NewNotFoundError, ErrorTypeNotFound},
		{"internal", NewInternalError, ErrorTypeInternal},
		{"external", NewExternalError, ErrorTypeExternal},
		{"configuration", NewConfigurationError, ErrorTypeConfiguration},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.constructor("test message")

			assert.Equal(t, tt.expectedType, err.Type)
			assert.Equal(t, "test message", err.Message)
		})
	}
}

func TestValidateRequired(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		fieldName string
		expectErr bool
	}{
		{"valid_value", "test", "field", false},
		{"empty_value", "", "field", true},
		{"whitespace_only", "   ", "field", false}, // Note: this doesn't trim whitespace
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRequired(tt.value, tt.fieldName)

			if tt.expectErr {
				assert.NotNil(t, err)
				assert.Equal(t, ErrorTypeValidation, err.Type)
				assert.Contains(t, err.Message, tt.fieldName)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestValidateNonEmpty(t *testing.T) {
	tests := []struct {
		name      string
		slice     interface{}
		fieldName string
		expectErr bool
	}{
		{"valid_string_slice", []string{"item"}, "field", false},
		{"empty_string_slice", []string{}, "field", true},
		{"valid_interface_slice", []interface{}{"item"}, "field", false},
		{"empty_interface_slice", []interface{}{}, "field", true},
		{"invalid_type", "not a slice", "field", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNonEmpty(tt.slice, tt.fieldName)

			if tt.expectErr {
				assert.NotNil(t, err)
				assert.Equal(t, ErrorTypeValidation, err.Type)
				assert.Contains(t, err.Message, tt.fieldName)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
