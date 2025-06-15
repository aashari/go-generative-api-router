package validator

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateAndModifyRequest(t *testing.T) {
	tests := []struct {
		name              string
		input             interface{}
		selectedModel     string
		expectError       bool
		expectedModel     string
		expectedFields    []string // Fields that should be present in output
		notExpectedFields []string // Fields that should NOT be present in output
	}{
		{
			name: "valid basic request",
			input: map[string]interface{}{
				"model":       "gpt-4",
				"messages":    []interface{}{map[string]interface{}{"role": "user", "content": "Hello"}},
				"temperature": 0.7,
				"max_tokens":  100,
			},
			selectedModel:     "gemini-pro",
			expectError:       false,
			expectedModel:     "gpt-4",
			expectedFields:    []string{"model", "messages"},
			notExpectedFields: []string{"temperature", "max_tokens"},
		},
		{
			name: "request with tools",
			input: map[string]interface{}{
				"model":   "gpt-4",
				"messages": []interface{}{map[string]interface{}{"role": "user", "content": "Hello"}},
				"tools": []interface{}{
					map[string]interface{}{
						"type": "function",
						"function": map[string]interface{}{
							"name": "get_weather",
						},
					},
				},
				"tool_choice": "auto",
			},
			selectedModel:  "gpt-4-tools",
			expectError:    false,
			expectedModel:  "gpt-4",
			expectedFields: []string{"model", "messages", "tools", "tool_choice"},
		},
		{
			name: "request with streaming",
			input: map[string]interface{}{
				"model":    "gpt-4",
				"messages": []interface{}{map[string]interface{}{"role": "user", "content": "Hello"}},
				"stream":   true,
			},
			selectedModel:  "gpt-4-turbo",
			expectError:    false,
			expectedModel:  "gpt-4",
			expectedFields: []string{"model", "messages", "stream"},
		},
		{
			name: "vision request with image_url",
			input: map[string]interface{}{
				"model": "gpt-4-vision",
				"messages": []interface{}{
					map[string]interface{}{
						"role": "user",
						"content": []interface{}{
							map[string]interface{}{
								"type": "text",
								"text": "What's in this image?",
							},
							map[string]interface{}{
								"type": "image_url",
								"image_url": map[string]interface{}{
									"url": "data:image/jpeg;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg==",
								},
							},
						},
					},
				},
			},
			selectedModel:  "gemini-pro-vision",
			expectError:    false,
			expectedModel:  "gpt-4-vision",
			expectedFields: []string{"model", "messages"},
		},
		{
			name: "request missing messages",
			input: map[string]interface{}{
				"model": "gpt-4",
			},
			selectedModel: "gpt-4",
			expectError:   true,
		},
		{
			name: "invalid JSON",
			input:         "invalid json string",
			selectedModel: "gpt-4",
			expectError:   true,
		},
		{
			name: "messages not array",
			input: map[string]interface{}{
				"model":    "gpt-4",
				"messages": "should be array",
			},
			selectedModel: "gpt-4",
			expectError:   true,
		},
		{
			name: "invalid tool format",
			input: map[string]interface{}{
				"model":    "gpt-4",
				"messages": []interface{}{map[string]interface{}{"role": "user", "content": "Hello"}},
				"tools": []interface{}{
					map[string]interface{}{
						"type": "invalid",
					},
				},
			},
			selectedModel: "gpt-4",
			expectError:   true,
		},
		{
			name: "invalid tool_choice",
			input: map[string]interface{}{
				"model":       "gpt-4",
				"messages":    []interface{}{map[string]interface{}{"role": "user", "content": "Hello"}},
				"tool_choice": "invalid_choice",
			},
			selectedModel: "gpt-4",
			expectError:   true,
		},
		{
			name: "invalid stream type",
			input: map[string]interface{}{
				"model":    "gpt-4",
				"messages": []interface{}{map[string]interface{}{"role": "user", "content": "Hello"}},
				"stream":   "should_be_boolean",
			},
			selectedModel: "gpt-4",
			expectError:   true,
		},
		{
			name: "no model provided - defaults to any-model",
			input: map[string]interface{}{
				"messages": []interface{}{map[string]interface{}{"role": "user", "content": "Hello"}},
			},
			selectedModel:  "gemini-pro",
			expectError:    false,
			expectedModel:  "any-model",
			expectedFields: []string{"model", "messages"},
		},
		{
			name: "assistant message with tool calls (no content)",
			input: map[string]interface{}{
				"model": "gpt-4",
				"messages": []interface{}{
					map[string]interface{}{
						"role": "assistant",
						"tool_calls": []interface{}{
							map[string]interface{}{
								"id":   "call_123",
								"type": "function",
								"function": map[string]interface{}{
									"name":      "get_weather",
									"arguments": "{}",
								},
							},
						},
					},
				},
			},
			selectedModel:  "gpt-4",
			expectError:    false,
			expectedModel:  "gpt-4",
			expectedFields: []string{"model", "messages"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert input to JSON bytes
			var inputBytes []byte
			var err error

			switch v := tt.input.(type) {
			case string:
				inputBytes = []byte(v)
			default:
				inputBytes, err = json.Marshal(v)
				require.NoError(t, err)
			}

			// Test the function
			modifiedBody, originalModel, err := ValidateAndModifyRequest(inputBytes, tt.selectedModel)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, modifiedBody)
				assert.Empty(t, originalModel)
				return
			}

			// Should not error
			require.NoError(t, err)
			assert.NotNil(t, modifiedBody)
			assert.Equal(t, tt.expectedModel, originalModel)

			// Parse the modified body to check structure
			var result map[string]interface{}
			err = json.Unmarshal(modifiedBody, &result)
			require.NoError(t, err)

			// Check that the model was replaced
			assert.Equal(t, tt.selectedModel, result["model"])

			// Check expected fields are present
			for _, field := range tt.expectedFields {
				assert.Contains(t, result, field, "Expected field %s to be present", field)
			}

			// Check unexpected fields are not present
			for _, field := range tt.notExpectedFields {
				assert.NotContains(t, result, field, "Expected field %s to NOT be present", field)
			}
		})
	}
}

func TestValidateMessages(t *testing.T) {
	tests := []struct {
		name        string
		requestData map[string]interface{}
		expectError bool
	}{
		{
			name: "valid messages",
			requestData: map[string]interface{}{
				"messages": []interface{}{
					map[string]interface{}{"role": "user", "content": "Hello"},
				},
			},
			expectError: false,
		},
		{
			name:        "missing messages",
			requestData: map[string]interface{}{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMessages(tt.requestData)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateMessageContent(t *testing.T) {
	tests := []struct {
		name        string
		requestData map[string]interface{}
		expectError bool
	}{
		{
			name: "valid string content",
			requestData: map[string]interface{}{
				"messages": []interface{}{
					map[string]interface{}{
						"role":    "user",
						"content": "Hello world",
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid array content with text and image",
			requestData: map[string]interface{}{
				"messages": []interface{}{
					map[string]interface{}{
						"role": "user",
						"content": []interface{}{
							map[string]interface{}{
								"type": "text",
								"text": "What's in this image?",
							},
							map[string]interface{}{
								"type": "image_url",
								"image_url": map[string]interface{}{
									"url": "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQAAAQABAAD/2w==",
								},
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid file_url content",
			requestData: map[string]interface{}{
				"messages": []interface{}{
					map[string]interface{}{
						"role": "user",
						"content": []interface{}{
							map[string]interface{}{
								"type": "file_url",
								"file_url": map[string]interface{}{
									"url": "https://example.com/document.pdf",
								},
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid audio_url content",
			requestData: map[string]interface{}{
				"messages": []interface{}{
					map[string]interface{}{
						"role": "user",
						"content": []interface{}{
							map[string]interface{}{
								"type": "audio_url",
								"audio_url": map[string]interface{}{
									"url": "https://example.com/audio.mp3",
								},
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid input_audio content",
			requestData: map[string]interface{}{
				"messages": []interface{}{
					map[string]interface{}{
						"role": "user",
						"content": []interface{}{
							map[string]interface{}{
								"type": "input_audio",
								"input_audio": map[string]interface{}{
									"data":   "base64encodedaudio",
									"format": "wav",
								},
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "message without content (valid for assistant with tool calls)",
			requestData: map[string]interface{}{
				"messages": []interface{}{
					map[string]interface{}{
						"role": "assistant",
						"tool_calls": []interface{}{
							map[string]interface{}{
								"id":   "call_123",
								"type": "function",
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "messages not array",
			requestData: map[string]interface{}{
				"messages": "not an array",
			},
			expectError: true,
		},
		{
			name: "message not object",
			requestData: map[string]interface{}{
				"messages": []interface{}{"not an object"},
			},
			expectError: true,
		},
		{
			name: "invalid content type",
			requestData: map[string]interface{}{
				"messages": []interface{}{
					map[string]interface{}{
						"role":    "user",
						"content": 123, // Should be string or array
					},
				},
			},
			expectError: true,
		},
		{
			name: "empty content array",
			requestData: map[string]interface{}{
				"messages": []interface{}{
					map[string]interface{}{
						"role":    "user",
						"content": []interface{}{}, // Empty array
					},
				},
			},
			expectError: true,
		},
		{
			name: "content part not object",
			requestData: map[string]interface{}{
				"messages": []interface{}{
					map[string]interface{}{
						"role": "user",
						"content": []interface{}{
							"not an object", // Should be object
						},
					},
				},
			},
			expectError: true,
		},
		{
			name: "content part missing type",
			requestData: map[string]interface{}{
				"messages": []interface{}{
					map[string]interface{}{
						"role": "user",
						"content": []interface{}{
							map[string]interface{}{
								"text": "Hello", // Missing type field
							},
						},
					},
				},
			},
			expectError: true,
		},
		{
			name: "text content part missing text field",
			requestData: map[string]interface{}{
				"messages": []interface{}{
					map[string]interface{}{
						"role": "user",
						"content": []interface{}{
							map[string]interface{}{
								"type": "text", // Missing text field
							},
						},
					},
				},
			},
			expectError: true,
		},
		{
			name: "image_url content part missing image_url field",
			requestData: map[string]interface{}{
				"messages": []interface{}{
					map[string]interface{}{
						"role": "user",
						"content": []interface{}{
							map[string]interface{}{
								"type": "image_url", // Missing image_url field
							},
						},
					},
				},
			},
			expectError: true,
		},
		{
			name: "image_url content part missing url in image_url",
			requestData: map[string]interface{}{
				"messages": []interface{}{
					map[string]interface{}{
						"role": "user",
						"content": []interface{}{
							map[string]interface{}{
								"type": "image_url",
								"image_url": map[string]interface{}{
									"detail": "high", // Missing url field
								},
							},
						},
					},
				},
			},
			expectError: true,
		},
		{
			name: "unknown content type",
			requestData: map[string]interface{}{
				"messages": []interface{}{
					map[string]interface{}{
						"role": "user",
						"content": []interface{}{
							map[string]interface{}{
								"type": "unknown_type",
								"text": "Hello",
							},
						},
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMessageContent(tt.requestData)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateTools(t *testing.T) {
	tests := []struct {
		name        string
		requestData map[string]interface{}
		expectError bool
	}{
		{
			name: "valid tools",
			requestData: map[string]interface{}{
				"tools": []interface{}{
					map[string]interface{}{
						"type": "function",
						"function": map[string]interface{}{
							"name":        "get_weather",
							"description": "Get current weather",
						},
					},
				},
			},
			expectError: false,
		},
		{
			name:        "no tools field (optional)",
			requestData: map[string]interface{}{},
			expectError: false,
		},
		{
			name: "tools not array",
			requestData: map[string]interface{}{
				"tools": "not an array",
			},
			expectError: true,
		},
		{
			name: "tool missing type",
			requestData: map[string]interface{}{
				"tools": []interface{}{
					map[string]interface{}{
						"function": map[string]interface{}{
							"name": "get_weather",
						},
					},
				},
			},
			expectError: true,
		},
		{
			name: "tool wrong type",
			requestData: map[string]interface{}{
				"tools": []interface{}{
					map[string]interface{}{
						"type": "invalid",
						"function": map[string]interface{}{
							"name": "get_weather",
						},
					},
				},
			},
			expectError: true,
		},
		{
			name: "tool missing function",
			requestData: map[string]interface{}{
				"tools": []interface{}{
					map[string]interface{}{
						"type": "function",
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTools(tt.requestData)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateToolChoice(t *testing.T) {
	tests := []struct {
		name        string
		requestData map[string]interface{}
		expectError bool
	}{
		{
			name: "valid tool_choice: auto",
			requestData: map[string]interface{}{
				"tool_choice": "auto",
			},
			expectError: false,
		},
		{
			name: "valid tool_choice: none",
			requestData: map[string]interface{}{
				"tool_choice": "none",
			},
			expectError: false,
		},
		{
			name: "valid tool_choice: required",
			requestData: map[string]interface{}{
				"tool_choice": "required",
			},
			expectError: false,
		},
		{
			name: "valid tool_choice function object",
			requestData: map[string]interface{}{
				"tool_choice": map[string]interface{}{
					"type": "function",
					"function": map[string]interface{}{
						"name": "get_weather",
					},
				},
			},
			expectError: false,
		},
		{
			name:        "no tool_choice field (optional)",
			requestData: map[string]interface{}{},
			expectError: false,
		},
		{
			name: "invalid tool_choice string",
			requestData: map[string]interface{}{
				"tool_choice": "invalid",
			},
			expectError: true,
		},
		{
			name: "invalid tool_choice object missing type",
			requestData: map[string]interface{}{
				"tool_choice": map[string]interface{}{
					"function": map[string]interface{}{
						"name": "get_weather",
					},
				},
			},
			expectError: true,
		},
		{
			name: "invalid tool_choice object wrong type",
			requestData: map[string]interface{}{
				"tool_choice": map[string]interface{}{
					"type": "invalid",
					"function": map[string]interface{}{
						"name": "get_weather",
					},
				},
			},
			expectError: true,
		},
		{
			name: "invalid tool_choice object missing function",
			requestData: map[string]interface{}{
				"tool_choice": map[string]interface{}{
					"type": "function",
				},
			},
			expectError: true,
		},
		{
			name: "invalid tool_choice type",
			requestData: map[string]interface{}{
				"tool_choice": 123,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateToolChoice(tt.requestData)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateStream(t *testing.T) {
	tests := []struct {
		name        string
		requestData map[string]interface{}
		expectError bool
	}{
		{
			name: "valid stream: true",
			requestData: map[string]interface{}{
				"stream": true,
			},
			expectError: false,
		},
		{
			name: "valid stream: false",
			requestData: map[string]interface{}{
				"stream": false,
			},
			expectError: false,
		},
		{
			name:        "no stream field (optional)",
			requestData: map[string]interface{}{},
			expectError: false,
		},
		{
			name: "invalid stream type",
			requestData: map[string]interface{}{
				"stream": "true", // Should be boolean
			},
			expectError: true,
		},
		{
			name: "invalid stream number",
			requestData: map[string]interface{}{
				"stream": 1, // Should be boolean
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStream(tt.requestData)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}