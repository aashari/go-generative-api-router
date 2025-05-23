package vendors

import (
	"fmt"

	"github.com/aashari/go-generative-api-router/internal/config"
	"github.com/aashari/go-generative-api-router/internal/httpclient"
	// TODO: Uncomment these when implementing Phase 2
	// "github.com/aashari/go-generative-api-router/internal/vendors/gemini"
	// "github.com/aashari/go-generative-api-router/internal/vendors/openai"
)

// Factory implements VendorFactory interface
type Factory struct {
	httpClientFactory *httpclient.Factory
}

// NewFactory creates a new vendor factory
func NewFactory(httpClientFactory *httpclient.Factory) *Factory {
	return &Factory{
		httpClientFactory: httpClientFactory,
	}
}

// CreateClient creates a vendor client based on the vendor name and configuration
func (f *Factory) CreateClient(vendorName string, credential string, config VendorConfig) (VendorClient, error) {
	// Validate HTTP client configuration (we'll use the httpClient in Phase 2)
	_, err := f.httpClientFactory.CreateClient(httpclient.Options{
		Timeout:    config.Timeout,
		MaxRetries: config.MaxRetries,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client for vendor %s: %w", vendorName, err)
	}

	// Create vendor-specific client
	switch vendorName {
	case "openai":
		// TODO: Implement in Phase 2
		// httpClient, _ := f.httpClientFactory.CreateClient(httpclient.Options{...})
		// return openai.NewClient(httpClient, config.BaseURL, credential), nil
		return nil, fmt.Errorf("OpenAI client not yet implemented - Phase 2")
	case "gemini":
		// TODO: Implement in Phase 2
		// httpClient, _ := f.httpClientFactory.CreateClient(httpclient.Options{...})
		// return gemini.NewClient(httpClient, config.BaseURL, credential), nil
		return nil, fmt.Errorf("Gemini client not yet implemented - Phase 2")
	default:
		return nil, fmt.Errorf("unsupported vendor: %s", vendorName)
	}
}

// GetSupportedVendors returns a list of supported vendor names
func (f *Factory) GetSupportedVendors() []string {
	return []string{"openai", "gemini"}
}

// CreateClientFromConfig creates a vendor client from a config.VendorConfig
func (f *Factory) CreateClientFromConfig(vendorConfig config.VendorConfig, credential string) (VendorClient, error) {
	config := VendorConfig{
		BaseURL:     vendorConfig.BaseURL,
		Timeout:     vendorConfig.Timeout,
		MaxRetries:  vendorConfig.MaxRetries,
		ExtraConfig: vendorConfig.ExtraConfig,
	}
	
	return f.CreateClient(vendorConfig.Name, credential, config)
}

// ValidateVendorConfig validates a vendor configuration
func (f *Factory) ValidateVendorConfig(vendorName string, config VendorConfig) error {
	// Check if vendor is supported
	supported := false
	for _, v := range f.GetSupportedVendors() {
		if v == vendorName {
			supported = true
			break
		}
	}
	
	if !supported {
		return fmt.Errorf("unsupported vendor: %s", vendorName)
	}
	
	// Validate configuration
	if config.BaseURL == "" {
		return fmt.Errorf("base URL is required for vendor %s", vendorName)
	}
	
	if config.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive for vendor %s", vendorName)
	}
	
	if config.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative for vendor %s", vendorName)
	}
	
	return nil
} 