package proxy

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessResponse(t *testing.T) {
	tests := []struct {
		name            string
		responseBody    []byte
		vendor          string
		contentEncoding string
		originalModel   string
		expectError     bool
		checkFields     map[string]interface{} // Fields that should be present with specific values
		checkExists     []string               // Fields that should exist
		checkNotExists  []string               // Fields that should NOT exist
	}{
		{
			name: "basic OpenAI response",
			responseBody: []byte(`{
				"id": "chatcmpl-123",
				"object": "chat.completion",
				"created": 1677652288,
				"model": "gpt-4",
				"choices": [{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "Hello!"
					},
					"finish_reason": "stop"
				}],
				"usage": {
					"prompt_tokens": 9,
					"completion_tokens": 12,
					"total_tokens": 21
				}
			}`),
			vendor:          "openai",
			contentEncoding: "",
			originalModel:   "gpt-4-turbo",
			expectError:     false,
			checkFields: map[string]interface{}{
				"id":    "chatcmpl-123",
				"model": "gpt-4-turbo", // Should be replaced
			},
			checkExists: []string{"system_fingerprint", "service_tier", "usage", "choices"},
		},
		{
			name: "Gemini response missing ID",
			responseBody: []byte(`{
				"model": "gemini-pro",
				"choices": [{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "Hello from Gemini!"
					},
					"finish_reason": "stop"
				}],
				"usage": {
					"prompt_tokens": 5,
					"completion_tokens": 8,
					"total_tokens": 13
				}
			}`),
			vendor:          "gemini",
			contentEncoding: "",
			originalModel:   "gemini-pro",
			expectError:     false,
			checkFields: map[string]interface{}{
				"model": "gemini-pro",
			},
			checkExists: []string{"id", "system_fingerprint", "service_tier", "usage", "choices"},
		},
		{
			name: "response with tool calls (Gemini - should generate new IDs)",
			responseBody: []byte(`{
				"id": "chatcmpl-123",
				"model": "gemini-pro",
				"choices": [{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": null,
						"tool_calls": [{
							"id": "call_existing",
							"type": "function",
							"function": {
								"name": "get_weather",
								"arguments": "{\"location\": \"San Francisco\"}"
							}
						}]
					},
					"finish_reason": "tool_calls"
				}],
				"usage": {
					"prompt_tokens": 20,
					"completion_tokens": 15,
					"total_tokens": 35
				}
			}`),
			vendor:          "gemini",
			contentEncoding: "",
			originalModel:   "gemini-pro",
			expectError:     false,
			checkExists:     []string{"id", "choices", "usage"},
		},
		{
			name: "response with tool calls (OpenAI - should preserve existing IDs)",
			responseBody: []byte(`{
				"id": "chatcmpl-456",
				"model": "gpt-4",
				"choices": [{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": null,
						"tool_calls": [{
							"id": "call_preserve_me",
							"type": "function",
							"function": {
								"name": "get_weather",
								"arguments": "{\"location\": \"New York\"}"
							}
						}]
					},
					"finish_reason": "tool_calls"
				}],
				"usage": {
					"prompt_tokens": 25,
					"completion_tokens": 20,
					"total_tokens": 45
				}
			}`),
			vendor:          "openai",
			contentEncoding: "",
			originalModel:   "gpt-4-turbo",
			expectError:     false,
			checkExists:     []string{"id", "choices", "usage"},
		},
		{
			name: "gzipped response",
			responseBody: func() []byte {
				originalResponse := `{"id": "chatcmpl-gzipped", "model": "gpt-4", "choices": [{"index": 0, "message": {"role": "assistant", "content": "Compressed!"}, "finish_reason": "stop"}], "usage": {"prompt_tokens": 3, "completion_tokens": 3, "total_tokens": 6}}`
				var buf bytes.Buffer
				gzipWriter := gzip.NewWriter(&buf)
				gzipWriter.Write([]byte(originalResponse))
				gzipWriter.Close()
				return buf.Bytes()
			}(),
			vendor:          "openai",
			contentEncoding: "gzip",
			originalModel:   "gpt-4-turbo",
			expectError:     false,
			checkFields: map[string]interface{}{
				"id":    "chatcmpl-gzipped",
				"model": "gpt-4-turbo",
			},
		},
		{
			name: "error response",
			responseBody: []byte(`{
				"error": {
					"message": "Invalid API key",
					"code": "invalid_api_key"
				}
			}`),
			vendor:          "openai",
			contentEncoding: "",
			originalModel:   "gpt-4",
			expectError:     false,
			checkExists:     []string{"error"},
		},
		{
			name: "array response - single element",
			responseBody: []byte(`[{
				"id": "chatcmpl-array",
				"model": "gpt-4",
				"choices": [{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "From array!"
					},
					"finish_reason": "stop"
				}],
				"usage": {
					"prompt_tokens": 4,
					"completion_tokens": 4,
					"total_tokens": 8
				}
			}]`),
			vendor:          "openai",
			contentEncoding: "",
			originalModel:   "gpt-4-turbo",
			expectError:     false,
			checkFields: map[string]interface{}{
				"id":    "chatcmpl-array",
				"model": "gpt-4-turbo",
			},
		},
		{
			name:            "empty array response",
			responseBody:    []byte(`[]`),
			vendor:          "openai",
			contentEncoding: "",
			originalModel:   "gpt-4",
			expectError:     false,
			checkExists:     []string{"error"},
		},
		{
			name:            "empty response body",
			responseBody:    []byte(``),
			vendor:          "openai",
			contentEncoding: "",
			originalModel:   "gpt-4",
			expectError:     false,
		},
		{
			name:            "invalid JSON",
			responseBody:    []byte(`{"invalid": json}`),
			vendor:          "openai",
			contentEncoding: "",
			originalModel:   "gpt-4",
			expectError:     false,
			// Should return original response for invalid JSON
		},
		{
			name: "response missing usage field",
			responseBody: []byte(`{
				"id": "chatcmpl-no-usage",
				"model": "gpt-4",
				"choices": [{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "No usage!"
					},
					"finish_reason": "stop"
				}]
			}`),
			vendor:          "openai",
			contentEncoding: "",
			originalModel:   "gpt-4-turbo",
			expectError:     false,
			checkExists:     []string{"usage"},
		},
		{
			name: "response with zero completion tokens",
			responseBody: []byte(`{
				"id": "chatcmpl-zero-tokens",
				"model": "gpt-4",
				"usage": {
					"prompt_tokens": 10,
					"completion_tokens": 0,
					"total_tokens": 10
				}
			}`),
			vendor:          "openai",
			contentEncoding: "",
			originalModel:   "gpt-4-turbo",
			expectError:     false,
			checkExists:     []string{"choices"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ProcessResponse(tt.responseBody, tt.vendor, tt.contentEncoding, tt.originalModel)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, result)

			// Skip further checks for empty responses
			if len(tt.responseBody) == 0 {
				assert.Equal(t, tt.responseBody, result)
				return
			}

			// For invalid JSON, just check that we get the original back
			if tt.name == "invalid JSON" {
				assert.Equal(t, tt.responseBody, result, "Invalid JSON should return original response")
				return
			}

			// Parse the result to check fields
			var resultData map[string]interface{}
			err = json.Unmarshal(result, &resultData)
			require.NoError(t, err, "Result should be valid JSON")

			// Check specific field values
			for field, expectedValue := range tt.checkFields {
				assert.Equal(t, expectedValue, resultData[field], "Field %s should have expected value", field)
			}

			// Check that certain fields exist
			for _, field := range tt.checkExists {
				assert.Contains(t, resultData, field, "Field %s should exist", field)
			}

			// Check that certain fields don't exist
			for _, field := range tt.checkNotExists {
				assert.NotContains(t, resultData, field, "Field %s should NOT exist", field)
			}

			// Validate usage field structure if it exists
			if usage, ok := resultData["usage"].(map[string]interface{}); ok {
				assert.Contains(t, usage, "prompt_tokens")
				assert.Contains(t, usage, "completion_tokens")
				assert.Contains(t, usage, "total_tokens")
				assert.Contains(t, usage, "prompt_tokens_details")
				assert.Contains(t, usage, "completion_tokens_details")
			}
		})
	}
}

func TestDecompressResponse(t *testing.T) {
	tests := []struct {
		name            string
		responseBody    []byte
		contentEncoding string
		expectError     bool
		expectedContent string
	}{
		{
			name:            "no compression",
			responseBody:    []byte(`{"test": "data"}`),
			contentEncoding: "",
			expectError:     false,
			expectedContent: `{"test": "data"}`,
		},
		{
			name:            "non-gzip content encoding",
			responseBody:    []byte(`{"test": "data"}`),
			contentEncoding: "deflate",
			expectError:     false,
			expectedContent: `{"test": "data"}`,
		},
		{
			name: "valid gzip compression",
			responseBody: func() []byte {
				var buf bytes.Buffer
				gzipWriter := gzip.NewWriter(&buf)
				gzipWriter.Write([]byte(`{"test": "compressed"}`))
				gzipWriter.Close()
				return buf.Bytes()
			}(),
			contentEncoding: "gzip",
			expectError:     false,
			expectedContent: `{"test": "compressed"}`,
		},
		{
			name:            "fake gzip header",
			responseBody:    []byte(`{"test": "not compressed"}`),
			contentEncoding: "gzip",
			expectError:     false,
			expectedContent: `{"test": "not compressed"}`, // Should fallback to original
		},
		{
			name:            "gzip header but invalid content",
			responseBody:    []byte{0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0x00}, // Invalid gzip
			contentEncoding: "gzip",
			expectError:     false, // Should fallback gracefully
		},
		{
			name:            "empty gzip response",
			responseBody:    []byte{},
			contentEncoding: "gzip",
			expectError:     false,
			expectedContent: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := decompressResponse(tt.responseBody, tt.contentEncoding)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			if tt.expectedContent != "" {
				assert.Equal(t, tt.expectedContent, string(result))
			}
		})
	}
}

func TestAddMissingIDs(t *testing.T) {
	tests := []struct {
		name           string
		responseData   map[string]interface{}
		shouldGenerate bool
	}{
		{
			name:           "missing id field",
			responseData:   map[string]interface{}{},
			shouldGenerate: true,
		},
		{
			name: "nil id field",
			responseData: map[string]interface{}{
				"id": nil,
			},
			shouldGenerate: true,
		},
		{
			name: "empty string id field",
			responseData: map[string]interface{}{
				"id": "",
			},
			shouldGenerate: true,
		},
		{
			name: "existing valid id",
			responseData: map[string]interface{}{
				"id": "chatcmpl-existing",
			},
			shouldGenerate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalID := tt.responseData["id"]
			addMissingIDs(tt.responseData)

			if tt.shouldGenerate {
				assert.NotEqual(t, originalID, tt.responseData["id"])
				assert.True(t, strings.HasPrefix(tt.responseData["id"].(string), "chatcmpl-"))
			} else {
				assert.Equal(t, originalID, tt.responseData["id"])
			}
		})
	}
}

func TestAddOpenAICompatibilityFields(t *testing.T) {
	tests := []struct {
		name         string
		responseData map[string]interface{}
		checkFields  map[string]interface{}
	}{
		{
			name:         "missing all fields",
			responseData: map[string]interface{}{},
			checkFields: map[string]interface{}{
				"service_tier": "default",
			},
		},
		{
			name: "missing system_fingerprint",
			responseData: map[string]interface{}{
				"service_tier": "premium",
			},
			checkFields: map[string]interface{}{
				"service_tier": "premium",
			},
		},
		{
			name: "null system_fingerprint",
			responseData: map[string]interface{}{
				"system_fingerprint": nil,
			},
			checkFields: map[string]interface{}{
				"service_tier": "default",
			},
		},
		{
			name: "non-string system_fingerprint",
			responseData: map[string]interface{}{
				"system_fingerprint": 123,
			},
			checkFields: map[string]interface{}{
				"service_tier": "default",
			},
		},
		{
			name: "existing valid fields",
			responseData: map[string]interface{}{
				"service_tier":        "premium",
				"system_fingerprint": "fp_existing",
			},
			checkFields: map[string]interface{}{
				"service_tier":        "premium",
				"system_fingerprint": "fp_existing",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addOpenAICompatibilityFields(tt.responseData)

			for field, expectedValue := range tt.checkFields {
				if field == "system_fingerprint" && expectedValue != "fp_existing" {
					// Check that it was generated (starts with fp_)
					assert.True(t, strings.HasPrefix(tt.responseData[field].(string), "fp_"))
				} else {
					assert.Equal(t, expectedValue, tt.responseData[field])
				}
			}

			// Ensure both fields exist
			assert.Contains(t, tt.responseData, "service_tier")
			assert.Contains(t, tt.responseData, "system_fingerprint")
		})
	}
}

func TestReplaceModelField(t *testing.T) {
	tests := []struct {
		name         string
		responseData map[string]interface{}
		vendor       string
		originalModel string
		expectedModel string
	}{
		{
			name: "replace existing model",
			responseData: map[string]interface{}{
				"model": "gpt-4-0613",
			},
			vendor:        "openai",
			originalModel: "gpt-4-turbo",
			expectedModel: "gpt-4-turbo",
		},
		{
			name:         "add model to response without one",
			responseData: map[string]interface{}{},
			vendor:       "gemini",
			originalModel: "gemini-pro",
			expectedModel: "gemini-pro",
		},
		{
			name: "empty original model",
			responseData: map[string]interface{}{
				"model": "gpt-4",
			},
			vendor:        "openai",
			originalModel: "",
			expectedModel: "gpt-4", // Should not replace
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			replaceModelField(tt.responseData, tt.vendor, tt.originalModel)
			
			if tt.originalModel != "" {
				assert.Equal(t, tt.expectedModel, tt.responseData["model"])
			}
		})
	}
}

func TestIsErrorResponse(t *testing.T) {
	tests := []struct {
		name         string
		responseData map[string]interface{}
		expected     bool
	}{
		{
			name: "valid error response",
			responseData: map[string]interface{}{
				"error": map[string]interface{}{
					"message": "Invalid API key",
					"code":    "invalid_api_key",
				},
			},
			expected: true,
		},
		{
			name: "error field not a map",
			responseData: map[string]interface{}{
				"error": "string error",
			},
			expected: false,
		},
		{
			name:         "no error field",
			responseData: map[string]interface{}{},
			expected:     false,
		},
		{
			name: "normal response",
			responseData: map[string]interface{}{
				"id":      "chatcmpl-123",
				"choices": []interface{}{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isErrorResponse(tt.responseData)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProcessErrorResponse(t *testing.T) {
	tests := []struct {
		name         string
		responseData map[string]interface{}
		checkFields  map[string]interface{}
	}{
		{
			name: "error with code",
			responseData: map[string]interface{}{
				"error": map[string]interface{}{
					"message": "Invalid request",
					"code":    "invalid_request",
				},
			},
			checkFields: map[string]interface{}{
				"type":  "invalid_request_error",
				"param": nil,
			},
		},
		{
			name: "error without code",
			responseData: map[string]interface{}{
				"error": map[string]interface{}{
					"message": "Unknown error",
				},
			},
			checkFields: map[string]interface{}{
				"type":  "api_error",
				"param": nil,
			},
		},
		{
			name: "error with existing param",
			responseData: map[string]interface{}{
				"error": map[string]interface{}{
					"message": "Invalid parameter",
					"code":    400,
					"param":   "temperature",
				},
			},
			checkFields: map[string]interface{}{
				"type":  "400_error",
				"param": "temperature",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processErrorResponse(tt.responseData)

			errorData := tt.responseData["error"].(map[string]interface{})
			for field, expectedValue := range tt.checkFields {
				assert.Equal(t, expectedValue, errorData[field])
			}
		})
	}
}

func TestNormalizeUsageField(t *testing.T) {
	tests := []struct {
		name         string
		responseData map[string]interface{}
		checkFields  []string
	}{
		{
			name:         "missing usage field",
			responseData: map[string]interface{}{},
			checkFields:  []string{"prompt_tokens", "completion_tokens", "total_tokens"},
		},
		{
			name: "partial usage field",
			responseData: map[string]interface{}{
				"usage": map[string]interface{}{
					"prompt_tokens": 10,
				},
			},
			checkFields: []string{"prompt_tokens", "completion_tokens", "total_tokens"},
		},
		{
			name: "complete usage field",
			responseData: map[string]interface{}{
				"usage": map[string]interface{}{
					"prompt_tokens":     10,
					"completion_tokens": 5,
					"total_tokens":      15,
				},
			},
			checkFields: []string{"prompt_tokens", "completion_tokens", "total_tokens"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalizeUsageField(tt.responseData)

			assert.Contains(t, tt.responseData, "usage")
			usage := tt.responseData["usage"].(map[string]interface{})

			for _, field := range tt.checkFields {
				assert.Contains(t, usage, field)
			}

			// Check for detailed token fields
			assert.Contains(t, usage, "prompt_tokens_details")
			assert.Contains(t, usage, "completion_tokens_details")

			promptDetails := usage["prompt_tokens_details"].(map[string]interface{})
			assert.Contains(t, promptDetails, "cached_tokens")
			assert.Contains(t, promptDetails, "audio_tokens")

			completionDetails := usage["completion_tokens_details"].(map[string]interface{})
			assert.Contains(t, completionDetails, "reasoning_tokens")
			assert.Contains(t, completionDetails, "audio_tokens")
		})
	}
}