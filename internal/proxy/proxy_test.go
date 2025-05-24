package proxy

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aashari/go-generative-api-router/internal/config"
	"github.com/aashari/go-generative-api-router/internal/selector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockSelector implements selector.Selector for testing
type MockSelector struct {
	selection *selector.VendorSelection
	err       error
}

func (m *MockSelector) Select(creds []config.Credential, models []config.VendorModel) (*selector.VendorSelection, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.selection, nil
}

// Type alias for cleaner code
type VendorSelection = selector.VendorSelection

// MockAPIClient for testing
type MockAPIClient struct {
	sendRequestFunc func(w http.ResponseWriter, r *http.Request, selection *VendorSelection, modifiedBody []byte, originalModel string) error
}

func (m *MockAPIClient) SendRequest(w http.ResponseWriter, r *http.Request, selection *selector.VendorSelection, modifiedBody []byte, originalModel string) error {
	if m.sendRequestFunc != nil {
		return m.sendRequestFunc(w, r, selection, modifiedBody, originalModel)
	}
	return nil
}

func TestProxyRequest_MethodValidation(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "POST allowed",
			method:         http.MethodPost,
			expectedStatus: http.StatusBadRequest, // Will fail validation but method is allowed
		},
		{
			name:           "GET not allowed",
			method:         http.MethodGet,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed",
		},
		{
			name:           "PUT not allowed",
			method:         http.MethodPut,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed",
		},
		{
			name:           "DELETE not allowed",
			method:         http.MethodDelete,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creds := []config.Credential{{Platform: "openai", Type: "api-key", Value: "test"}}
			models := []config.VendorModel{{Vendor: "openai", Model: "gpt-4"}}
			selector := &MockSelector{
				selection: &selector.VendorSelection{
					Vendor:     "openai",
					Model:      "gpt-4",
					Credential: creds[0],
				},
			}
			client := NewAPIClient()

			w := httptest.NewRecorder()
			r := httptest.NewRequest(tt.method, "/v1/chat/completions", nil)

			ProxyRequest(w, r, creds, models, client, selector)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}
		})
	}
}

func TestProxyRequest_SelectorError(t *testing.T) {
	creds := []config.Credential{{Platform: "openai", Type: "api-key", Value: "test"}}
	models := []config.VendorModel{{Vendor: "openai", Model: "gpt-4"}}
	selector := &MockSelector{
		err: assert.AnError,
	}
	client := NewAPIClient()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/chat/completions",
		strings.NewReader(`{"messages": [{"role": "user", "content": "test"}]}`))

	ProxyRequest(w, r, creds, models, client, selector)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "assert.AnError")
}

func TestProxyRequest_InvalidRequestBody(t *testing.T) {
	creds := []config.Credential{{Platform: "openai", Type: "api-key", Value: "test"}}
	models := []config.VendorModel{{Vendor: "openai", Model: "gpt-4"}}
	selector := &MockSelector{
		selection: &selector.VendorSelection{
			Vendor:     "openai",
			Model:      "gpt-4",
			Credential: creds[0],
		},
	}
	client := NewAPIClient()

	w := httptest.NewRecorder()
	// Invalid JSON body
	r := httptest.NewRequest(http.MethodPost, "/v1/chat/completions",
		strings.NewReader(`{"invalid": json`))

	ProxyRequest(w, r, creds, models, client, selector)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid request format")
}

func TestProxyRequest_MissingMessages(t *testing.T) {
	creds := []config.Credential{{Platform: "openai", Type: "api-key", Value: "test"}}
	models := []config.VendorModel{{Vendor: "openai", Model: "gpt-4"}}
	selector := &MockSelector{
		selection: &selector.VendorSelection{
			Vendor:     "openai",
			Model:      "gpt-4",
			Credential: creds[0],
		},
	}
	client := NewAPIClient()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/chat/completions",
		strings.NewReader(`{"model": "test-model"}`))

	ProxyRequest(w, r, creds, models, client, selector)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "missing 'messages' field")
}

func TestProxyRequest_UnknownVendorError(t *testing.T) {
	creds := []config.Credential{{Platform: "openai", Type: "api-key", Value: "test"}}
	models := []config.VendorModel{{Vendor: "openai", Model: "gpt-4"}}
	selector := &MockSelector{
		selection: &selector.VendorSelection{
			Vendor:     "unknown-vendor",
			Model:      "gpt-4",
			Credential: creds[0],
		},
	}
	client := NewAPIClient()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/chat/completions",
		strings.NewReader(`{"model": "test-model", "messages": [{"role": "user", "content": "test"}]}`))

	ProxyRequest(w, r, creds, models, client, selector)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Internal configuration error: Unknown vendor")
}

func TestProxyRequest_NetworkError(t *testing.T) {
	creds := []config.Credential{{Platform: "openai", Type: "api-key", Value: "test"}}
	models := []config.VendorModel{{Vendor: "openai", Model: "gpt-4"}}
	selector := &MockSelector{
		selection: &selector.VendorSelection{
			Vendor:     "openai",
			Model:      "gpt-4",
			Credential: creds[0],
		},
	}

	// Mock client that returns network error
	mockClient := &MockAPIClient{
		sendRequestFunc: func(w http.ResponseWriter, r *http.Request, selection *VendorSelection, modifiedBody []byte, originalModel string) error {
			return assert.AnError
		},
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/chat/completions",
		strings.NewReader(`{"model": "test-model", "messages": [{"role": "user", "content": "test"}]}`))

	ProxyRequest(w, r, creds, models, mockClient, selector)

	assert.Equal(t, http.StatusBadGateway, w.Code)
	assert.Contains(t, w.Body.String(), "Failed to communicate with upstream service")
}

func TestProxyRequest_SuccessfulModelTransformation(t *testing.T) {
	creds := []config.Credential{{Platform: "openai", Type: "api-key", Value: "test"}}
	models := []config.VendorModel{{Vendor: "openai", Model: "gpt-4"}}
	selector := &MockSelector{
		selection: &selector.VendorSelection{
			Vendor:     "openai",
			Model:      "gpt-4",
			Credential: creds[0],
		},
	}

	// Track the parameters passed to SendRequest
	var capturedOriginalModel string
	var capturedModifiedBody []byte
	var capturedSelection *VendorSelection

	mockClient := &MockAPIClient{
		sendRequestFunc: func(w http.ResponseWriter, r *http.Request, selection *VendorSelection, modifiedBody []byte, originalModel string) error {
			capturedOriginalModel = originalModel
			capturedModifiedBody = modifiedBody
			capturedSelection = selection

			// Verify the modified body has the actual model
			var requestData map[string]interface{}
			err := json.Unmarshal(modifiedBody, &requestData)
			require.NoError(t, err)
			assert.Equal(t, "gpt-4", requestData["model"])

			w.WriteHeader(http.StatusOK)
			return nil
		},
	}

	w := httptest.NewRecorder()
	originalModelName := "my-custom-model"
	requestBody := map[string]interface{}{
		"model":    originalModelName,
		"messages": []map[string]string{{"role": "user", "content": "test"}},
	}
	bodyBytes, _ := json.Marshal(requestBody)
	r := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(bodyBytes))

	ProxyRequest(w, r, creds, models, mockClient, selector)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, originalModelName, capturedOriginalModel)
	assert.Equal(t, "openai", capturedSelection.Vendor)
	assert.Equal(t, "gpt-4", capturedSelection.Model)

	// Verify the modified body has the selected model
	var modifiedData map[string]interface{}
	err := json.Unmarshal(capturedModifiedBody, &modifiedData)
	require.NoError(t, err)
	assert.Equal(t, "gpt-4", modifiedData["model"])
}

func TestProxyRequest_StreamingRequest(t *testing.T) {
	creds := []config.Credential{{Platform: "openai", Type: "api-key", Value: "test"}}
	models := []config.VendorModel{{Vendor: "openai", Model: "gpt-4"}}
	selector := &MockSelector{
		selection: &selector.VendorSelection{
			Vendor:     "openai",
			Model:      "gpt-4",
			Credential: creds[0],
		},
	}

	var capturedOriginalModel string
	var capturedIsStreaming bool

	mockClient := &MockAPIClient{
		sendRequestFunc: func(w http.ResponseWriter, r *http.Request, selection *VendorSelection, modifiedBody []byte, originalModel string) error {
			capturedOriginalModel = originalModel

			// Check if request body has stream: true
			var requestData map[string]interface{}
			err := json.Unmarshal(modifiedBody, &requestData)
			require.NoError(t, err)

			if stream, ok := requestData["stream"].(bool); ok && stream {
				capturedIsStreaming = true
			}

			w.WriteHeader(http.StatusOK)
			return nil
		},
	}

	w := httptest.NewRecorder()
	originalModelName := "my-streaming-model"
	requestBody := map[string]interface{}{
		"model":    originalModelName,
		"messages": []map[string]string{{"role": "user", "content": "test"}},
		"stream":   true,
	}
	bodyBytes, _ := json.Marshal(requestBody)
	r := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(bodyBytes))

	ProxyRequest(w, r, creds, models, mockClient, selector)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, originalModelName, capturedOriginalModel)
	assert.True(t, capturedIsStreaming)
}

func TestProxyRequest_DefaultModelName(t *testing.T) {
	creds := []config.Credential{{Platform: "openai", Type: "api-key", Value: "test"}}
	models := []config.VendorModel{{Vendor: "openai", Model: "gpt-4"}}
	selector := &MockSelector{
		selection: &selector.VendorSelection{
			Vendor:     "openai",
			Model:      "gpt-4",
			Credential: creds[0],
		},
	}

	var capturedOriginalModel string

	mockClient := &MockAPIClient{
		sendRequestFunc: func(w http.ResponseWriter, r *http.Request, selection *VendorSelection, modifiedBody []byte, originalModel string) error {
			capturedOriginalModel = originalModel
			w.WriteHeader(http.StatusOK)
			return nil
		},
	}

	w := httptest.NewRecorder()
	// Request without model field
	requestBody := map[string]interface{}{
		"messages": []map[string]string{{"role": "user", "content": "test"}},
	}
	bodyBytes, _ := json.Marshal(requestBody)
	r := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(bodyBytes))

	ProxyRequest(w, r, creds, models, mockClient, selector)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "any-model", capturedOriginalModel) // Default value when no model provided
}

func TestProxyRequest_MultipleVendors(t *testing.T) {
	creds := []config.Credential{
		{Platform: "openai", Type: "api-key", Value: "openai-test"},
		{Platform: "gemini", Type: "api-key", Value: "gemini-test"},
	}
	models := []config.VendorModel{
		{Vendor: "openai", Model: "gpt-4"},
		{Vendor: "gemini", Model: "gemini-pro"},
	}

	testCases := []struct {
		name           string
		selectedVendor string
		selectedModel  string
	}{
		{
			name:           "OpenAI selection",
			selectedVendor: "openai",
			selectedModel:  "gpt-4",
		},
		{
			name:           "Gemini selection",
			selectedVendor: "gemini",
			selectedModel:  "gemini-pro",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var selectedCred config.Credential
			for _, cred := range creds {
				if cred.Platform == tc.selectedVendor {
					selectedCred = cred
					break
				}
			}

			selector := &MockSelector{
				selection: &selector.VendorSelection{
					Vendor:     tc.selectedVendor,
					Model:      tc.selectedModel,
					Credential: selectedCred,
				},
			}

			var capturedSelection *VendorSelection

			mockClient := &MockAPIClient{
				sendRequestFunc: func(w http.ResponseWriter, r *http.Request, selection *VendorSelection, modifiedBody []byte, originalModel string) error {
					capturedSelection = selection
					w.WriteHeader(http.StatusOK)
					return nil
				},
			}

			w := httptest.NewRecorder()
			requestBody := map[string]interface{}{
				"model":    "user-requested-model",
				"messages": []map[string]string{{"role": "user", "content": "test"}},
			}
			bodyBytes, _ := json.Marshal(requestBody)
			r := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(bodyBytes))

			ProxyRequest(w, r, creds, models, mockClient, selector)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, tc.selectedVendor, capturedSelection.Vendor)
			assert.Equal(t, tc.selectedModel, capturedSelection.Model)
			assert.Equal(t, selectedCred.Value, capturedSelection.Credential.Value)
		})
	}
}

func TestProxyRequest_ComplexToolsValidation(t *testing.T) {
	creds := []config.Credential{{Platform: "openai", Type: "api-key", Value: "test"}}
	models := []config.VendorModel{{Vendor: "openai", Model: "gpt-4"}}
	selector := &MockSelector{
		selection: &selector.VendorSelection{
			Vendor:     "openai",
			Model:      "gpt-4",
			Credential: creds[0],
		},
	}

	mockClient := &MockAPIClient{
		sendRequestFunc: func(w http.ResponseWriter, r *http.Request, selection *VendorSelection, modifiedBody []byte, originalModel string) error {
			w.WriteHeader(http.StatusOK)
			return nil
		},
	}

	w := httptest.NewRecorder()
	requestBody := map[string]interface{}{
		"model":    "test-model",
		"messages": []map[string]string{{"role": "user", "content": "What's the weather?"}},
		"tools": []map[string]interface{}{
			{
				"type": "function",
				"function": map[string]interface{}{
					"name":        "get_weather",
					"description": "Get weather information",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"location": map[string]interface{}{
								"type":        "string",
								"description": "City name",
							},
						},
						"required": []string{"location"},
					},
				},
			},
		},
		"tool_choice": "auto",
	}
	bodyBytes, _ := json.Marshal(requestBody)
	r := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(bodyBytes))

	ProxyRequest(w, r, creds, models, mockClient, selector)

	assert.Equal(t, http.StatusOK, w.Code)
}
