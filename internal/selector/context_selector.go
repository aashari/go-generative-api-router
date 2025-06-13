package selector

import (
	"fmt"

	"github.com/aashari/go-generative-api-router/internal/config"
	"github.com/aashari/go-generative-api-router/internal/types"
)

// ContextAwareSelector extends EvenDistributionSelector to filter models based on payload context
type ContextAwareSelector struct {
	*EvenDistributionSelector
}

// NewContextAwareSelector creates a new context-aware selector
func NewContextAwareSelector() *ContextAwareSelector {
	return &ContextAwareSelector{
		EvenDistributionSelector: NewEvenDistributionSelector(),
	}
}

// SelectWithContext selects a model considering the payload context and model capabilities
func (s *ContextAwareSelector) SelectWithContext(creds []config.Credential, models []config.VendorModel, context *types.PayloadContext) (*VendorSelection, error) {
	if len(creds) == 0 {
		return nil, fmt.Errorf("no credentials available")
	}
	if len(models) == 0 {
		return nil, fmt.Errorf("no models available")
	}

	// Filter models based on payload context
	filteredModels := filterModelsByCapabilities(models, context)

	if len(filteredModels) == 0 {
		return nil, fmt.Errorf("no models available that support the required capabilities")
	}

	// Use the parent's Select method with filtered models
	return s.EvenDistributionSelector.Select(creds, filteredModels)
}

// filterModelsByCapabilities filters models based on their capabilities and the payload context
func filterModelsByCapabilities(models []config.VendorModel, context *types.PayloadContext) []config.VendorModel {
	if context == nil {
		// If no context, return all models
		return models
	}

	var filteredModels []config.VendorModel

	for _, model := range models {
		// If model has no config, assume it supports everything
		if model.Config == nil {
			filteredModels = append(filteredModels, model)
			continue
		}

		// Check if model supports required capabilities
		if shouldIncludeModel(model.Config, context) {
			filteredModels = append(filteredModels, model)
		}
	}

	return filteredModels
}

// shouldIncludeModel determines if a model should be included based on its capabilities and the payload context
func shouldIncludeModel(config *config.ModelConfig, context *types.PayloadContext) bool {
	// Check image support
	if context.HasImages && !config.SupportImage {
		return false
	}

	// Check video support
	if context.HasVideos && !config.SupportVideo {
		return false
	}

	// Check tools support
	if context.HasTools && !config.SupportTools {
		return false
	}

	// Check streaming support
	if context.HasStream && !config.SupportStreaming {
		return false
	}

	// Model supports all required capabilities
	return true
}
