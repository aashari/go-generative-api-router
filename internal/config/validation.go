package config

import (
	"fmt"
	"strings"

	"github.com/aashari/go-generative-api-router/internal/errors"
	"github.com/go-playground/validator/v10"
)

// AppConfig represents the complete application configuration
type AppConfig struct {
	Credentials  []Credential  `validate:"required,min=1,dive"`
	VendorModels []VendorModel `validate:"required,min=1,dive"`
}

// Credential validation tags
type ValidatedCredential struct {
	Platform string `validate:"required,oneof=openai gemini anthropic"`
	Type     string `validate:"required,oneof=api-key oauth"`
	Value    string `validate:"required,min=1"`
}

// VendorModel validation tags
type ValidatedVendorModel struct {
	Vendor string `validate:"required,oneof=openai gemini anthropic"`
	Model  string `validate:"required,min=1"`
}

var validate *validator.Validate

func init() {
	validate = validator.New()
}

// ValidateConfiguration validates the complete application configuration
func ValidateConfiguration(creds []Credential, models []VendorModel) *errors.APIError {
	config := AppConfig{
		Credentials:  creds,
		VendorModels: models,
	}

	if err := validate.Struct(config); err != nil {
		return formatValidationError(err)
	}

	// Additional business logic validation
	if err := validateBusinessRules(creds, models); err != nil {
		return err
	}

	return nil
}

// ValidateCredentials validates credential configuration
func ValidateCredentials(creds []Credential) *errors.APIError {
	if len(creds) == 0 {
		return errors.NewConfigurationError("No credentials provided")
	}

	for i, cred := range creds {
		if err := validateCredential(cred, i); err != nil {
			return err
		}
	}

	return nil
}

// ValidateVendorModels validates vendor model configuration
func ValidateVendorModels(models []VendorModel) *errors.APIError {
	if len(models) == 0 {
		return errors.NewConfigurationError("No vendor models provided")
	}

	for i, model := range models {
		if err := validateVendorModel(model, i); err != nil {
			return err
		}
	}

	return nil
}

// validateCredential validates a single credential
func validateCredential(cred Credential, index int) *errors.APIError {
	validatedCred := ValidatedCredential{
		Platform: cred.Platform,
		Type:     cred.Type,
		Value:    cred.Value,
	}

	if err := validate.Struct(validatedCred); err != nil {
		return formatCredentialValidationError(err, index)
	}

	// Additional validation for API key format
	if cred.Type == "api-key" {
		if err := validateAPIKeyFormat(cred.Platform, cred.Value); err != nil {
			return errors.NewConfigurationError(fmt.Sprintf("Credential %d: %s", index, err.Error()))
		}
	}

	return nil
}

// validateVendorModel validates a single vendor model
func validateVendorModel(model VendorModel, index int) *errors.APIError {
	validatedModel := ValidatedVendorModel{
		Vendor: model.Vendor,
		Model:  model.Model,
	}

	if err := validate.Struct(validatedModel); err != nil {
		return formatVendorModelValidationError(err, index)
	}

	return nil
}

// validateAPIKeyFormat validates API key format for different vendors
func validateAPIKeyFormat(platform, apiKey string) error {
	switch platform {
	case "openai":
		if !strings.HasPrefix(apiKey, "sk-") {
			return fmt.Errorf("OpenAI API key must start with 'sk-'")
		}
		if len(apiKey) < 20 {
			return fmt.Errorf("OpenAI API key appears to be too short")
		}
	case "gemini":
		if len(apiKey) < 10 {
			return fmt.Errorf("Gemini API key appears to be too short")
		}
	case "anthropic":
		if !strings.HasPrefix(apiKey, "sk-ant-") {
			return fmt.Errorf("Anthropic API key must start with 'sk-ant-'")
		}
	}
	return nil
}

// validateBusinessRules validates business logic rules
func validateBusinessRules(creds []Credential, models []VendorModel) *errors.APIError {
	// Check that we have credentials for all vendors in models
	vendorCreds := make(map[string]bool)
	for _, cred := range creds {
		vendorCreds[cred.Platform] = true
	}

	var missingCreds []string
	for _, model := range models {
		if !vendorCreds[model.Vendor] {
			missingCreds = append(missingCreds, model.Vendor)
		}
	}

	if len(missingCreds) > 0 {
		return errors.NewConfigurationError(fmt.Sprintf("Missing credentials for vendors: %s", strings.Join(missingCreds, ", ")))
	}

	// Check for duplicate models
	modelKeys := make(map[string]bool)
	for _, model := range models {
		key := fmt.Sprintf("%s:%s", model.Vendor, model.Model)
		if modelKeys[key] {
			return errors.NewConfigurationError(fmt.Sprintf("Duplicate model configuration: %s", key))
		}
		modelKeys[key] = true
	}

	return nil
}

// formatValidationError formats validator errors into APIError
func formatValidationError(err error) *errors.APIError {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		var messages []string
		for _, e := range validationErrors {
			messages = append(messages, formatFieldError(e))
		}
		return errors.NewConfigurationError(fmt.Sprintf("Configuration validation failed: %s", strings.Join(messages, "; ")))
	}
	return errors.NewConfigurationError(fmt.Sprintf("Configuration validation failed: %s", err.Error()))
}

// formatCredentialValidationError formats credential validation errors
func formatCredentialValidationError(err error, index int) *errors.APIError {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		var messages []string
		for _, e := range validationErrors {
			messages = append(messages, formatFieldError(e))
		}
		return errors.NewConfigurationError(fmt.Sprintf("Credential %d validation failed: %s", index, strings.Join(messages, "; ")))
	}
	return errors.NewConfigurationError(fmt.Sprintf("Credential %d validation failed: %s", index, err.Error()))
}

// formatVendorModelValidationError formats vendor model validation errors
func formatVendorModelValidationError(err error, index int) *errors.APIError {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		var messages []string
		for _, e := range validationErrors {
			messages = append(messages, formatFieldError(e))
		}
		return errors.NewConfigurationError(fmt.Sprintf("Vendor model %d validation failed: %s", index, strings.Join(messages, "; ")))
	}
	return errors.NewConfigurationError(fmt.Sprintf("Vendor model %d validation failed: %s", index, err.Error()))
}

// formatFieldError formats a single field validation error
func formatFieldError(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return fmt.Sprintf("field '%s' is required", e.Field())
	case "min":
		return fmt.Sprintf("field '%s' must have at least %s items", e.Field(), e.Param())
	case "oneof":
		return fmt.Sprintf("field '%s' must be one of: %s", e.Field(), e.Param())
	default:
		return fmt.Sprintf("field '%s' failed validation: %s", e.Field(), e.Tag())
	}
}
