package proxy

import (
	"errors"
	"fmt"
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

// IsRetriableValidationError checks if the error is retriable (missing choices from Gemini)
func IsRetriableValidationError(err error) bool {
	var vendorErr *VendorValidationError
	if errors.As(err, &vendorErr) {
		// Only retry if it's Gemini with missing 'choices' field
		return vendorErr.Vendor == "gemini" && vendorErr.MissingField == "choices"
	}
	return false
}
