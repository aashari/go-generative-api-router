package proxy

import (
	"net/http"

	"github.com/aashari/go-generative-api-router/internal/config"
	"github.com/aashari/go-generative-api-router/internal/selector"
)

// ProxyHandler encapsulates the proxy functionality for easier testing
type ProxyHandler struct {
	credentials   []config.Credential
	models        []config.VendorModel
	apiClient     APIClientInterface
	modelSelector selector.Selector
}

// NewProxyHandler creates a new proxy handler with all dependencies
func NewProxyHandler(credentials []config.Credential, models []config.VendorModel, apiClient APIClientInterface, modelSelector selector.Selector) *ProxyHandler {
	return &ProxyHandler{
		credentials:   credentials,
		models:        models,
		apiClient:     apiClient,
		modelSelector: modelSelector,
	}
}

// HandleChatCompletions handles chat completions requests using the proxy pipeline
func (ph *ProxyHandler) HandleChatCompletions(w http.ResponseWriter, r *http.Request) {
	ProxyRequest(w, r, ph.credentials, ph.models, ph.apiClient, ph.modelSelector)
}