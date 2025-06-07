package validator

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateAndModifyRequest(t *testing.T) {
	tests := []struct {
		name          string
		body          []byte
		selectedModel string
		wantErr       bool
		expectedOrig  string
		errContains   string
	}{
		{
			name:          "valid basic request",
			body:          []byte(`{"model": "gpt-4", "messages": [{"role": "user", "content": "hello"}]}`),
			selectedModel: "gpt-3.5-turbo",
			wantErr:       false,
			expectedOrig:  "gpt-4",
		},
		{
			name:          "valid request without model",
			body:          []byte(`{"messages": [{"role": "user", "content": "hello"}]}`),
			selectedModel: "gpt-3.5-turbo",
			wantErr:       false,
			expectedOrig:  "any-model",
		},
		{
			name:          "valid request with tools",
			body:          []byte(`{"model": "gpt-4", "messages": [{"role": "user", "content": "hello"}], "tools": [{"type": "function", "function": {"name": "test", "description": "test"}}]}`),
			selectedModel: "gpt-3.5-turbo",
			wantErr:       false,
			expectedOrig:  "gpt-4",
		},
		{
			name:          "valid request with tool_choice string",
			body:          []byte(`{"model": "gpt-4", "messages": [{"role": "user", "content": "hello"}], "tool_choice": "auto"}`),
			selectedModel: "gpt-3.5-turbo",
			wantErr:       false,
			expectedOrig:  "gpt-4",
		},
		{
			name:          "valid request with tool_choice object",
			body:          []byte(`{"model": "gpt-4", "messages": [{"role": "user", "content": "hello"}], "tool_choice": {"type": "function", "function": {"name": "test"}}}`),
			selectedModel: "gpt-3.5-turbo",
			wantErr:       false,
			expectedOrig:  "gpt-4",
		},
		{
			name:          "valid request with stream",
			body:          []byte(`{"model": "gpt-4", "messages": [{"role": "user", "content": "hello"}], "stream": true}`),
			selectedModel: "gpt-3.5-turbo",
			wantErr:       false,
			expectedOrig:  "gpt-4",
		},
		{
			name:        "invalid JSON",
			body:        []byte(`{"model": "gpt-4", "messages": [{"role": "user", "content": "hello"}`),
			wantErr:     true,
			errContains: "invalid request format",
		},
		{
			name:        "missing messages",
			body:        []byte(`{"model": "gpt-4"}`),
			wantErr:     true,
			errContains: "missing 'messages' field",
		},
		{
			name:        "invalid tools format - not array",
			body:        []byte(`{"model": "gpt-4", "messages": [{"role": "user", "content": "hello"}], "tools": "invalid"}`),
			wantErr:     true,
			errContains: "invalid 'tools' format",
		},
		{
			name:        "invalid tools format - wrong type",
			body:        []byte(`{"model": "gpt-4", "messages": [{"role": "user", "content": "hello"}], "tools": [{"type": "invalid"}]}`),
			wantErr:     true,
			errContains: "invalid 'tools' format",
		},
		{
			name:        "invalid tool_choice string",
			body:        []byte(`{"model": "gpt-4", "messages": [{"role": "user", "content": "hello"}], "tool_choice": "invalid"}`),
			wantErr:     true,
			errContains: "invalid 'tool_choice'",
		},
		{
			name:        "invalid tool_choice object",
			body:        []byte(`{"model": "gpt-4", "messages": [{"role": "user", "content": "hello"}], "tool_choice": {"type": "invalid"}}`),
			wantErr:     true,
			errContains: "invalid 'tool_choice'",
		},
		{
			name:        "invalid stream type",
			body:        []byte(`{"model": "gpt-4", "messages": [{"role": "user", "content": "hello"}], "stream": "true"}`),
			wantErr:     true,
			errContains: "invalid 'stream' field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modifiedBody, originalModel, err := ValidateAndModifyRequest(tt.body, tt.selectedModel)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedOrig, originalModel)

			// Verify modified body has the selected model
			var modifiedData map[string]interface{}
			err = json.Unmarshal(modifiedBody, &modifiedData)
			require.NoError(t, err)
			assert.Equal(t, tt.selectedModel, modifiedData["model"])
		})
	}
}

func TestValidateMessages(t *testing.T) {
	tests := []struct {
		name        string
		requestData map[string]interface{}
		wantErr     bool
	}{
		{
			name:        "valid messages",
			requestData: map[string]interface{}{"messages": []interface{}{}},
			wantErr:     false,
		},
		{
			name:        "missing messages",
			requestData: map[string]interface{}{"model": "gpt-4"},
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMessages(tt.requestData)
			if tt.wantErr {
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
		wantErr     bool
	}{
		{
			name:        "no tools field",
			requestData: map[string]interface{}{"model": "gpt-4"},
			wantErr:     false,
		},
		{
			name: "valid tools",
			requestData: map[string]interface{}{
				"tools": []interface{}{
					map[string]interface{}{
						"type": "function",
						"function": map[string]interface{}{
							"name": "test",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid tools - not array",
			requestData: map[string]interface{}{
				"tools": "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid tools - wrong type",
			requestData: map[string]interface{}{
				"tools": []interface{}{
					map[string]interface{}{
						"type": "invalid",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid tools - missing function",
			requestData: map[string]interface{}{
				"tools": []interface{}{
					map[string]interface{}{
						"type": "function",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTools(tt.requestData)
			if tt.wantErr {
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
		wantErr     bool
	}{
		{
			name:        "no tool_choice field",
			requestData: map[string]interface{}{"model": "gpt-4"},
			wantErr:     false,
		},
		{
			name: "valid tool_choice - auto",
			requestData: map[string]interface{}{
				"tool_choice": "auto",
			},
			wantErr: false,
		},
		{
			name: "valid tool_choice - none",
			requestData: map[string]interface{}{
				"tool_choice": "none",
			},
			wantErr: false,
		},
		{
			name: "valid tool_choice - required",
			requestData: map[string]interface{}{
				"tool_choice": "required",
			},
			wantErr: false,
		},
		{
			name: "valid tool_choice - function object",
			requestData: map[string]interface{}{
				"tool_choice": map[string]interface{}{
					"type": "function",
					"function": map[string]interface{}{
						"name": "test",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid tool_choice - invalid string",
			requestData: map[string]interface{}{
				"tool_choice": "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid tool_choice - invalid object",
			requestData: map[string]interface{}{
				"tool_choice": map[string]interface{}{
					"type": "invalid",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid tool_choice - wrong type",
			requestData: map[string]interface{}{
				"tool_choice": 123,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateToolChoice(tt.requestData)
			if tt.wantErr {
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
		wantErr     bool
	}{
		{
			name:        "no stream field",
			requestData: map[string]interface{}{"model": "gpt-4"},
			wantErr:     false,
		},
		{
			name: "valid stream - true",
			requestData: map[string]interface{}{
				"stream": true,
			},
			wantErr: false,
		},
		{
			name: "valid stream - false",
			requestData: map[string]interface{}{
				"stream": false,
			},
			wantErr: false,
		},
		{
			name: "invalid stream - string",
			requestData: map[string]interface{}{
				"stream": "true",
			},
			wantErr: true,
		},
		{
			name: "invalid stream - number",
			requestData: map[string]interface{}{
				"stream": 1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStream(tt.requestData)
			if tt.wantErr {
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
		wantErr     bool
		errContains string
	}{
		{
			name: "valid string content",
			requestData: map[string]interface{}{
				"messages": []interface{}{
					map[string]interface{}{
						"role":    "user",
						"content": "Hello, world!",
					},
				},
			},
			wantErr: false,
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
									"url": "https://example.com/image.jpg",
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid messages format - not array",
			requestData: map[string]interface{}{
				"messages": "not an array",
			},
			wantErr:     true,
			errContains: "must be an array",
		},
		{
			name: "invalid message format - not object",
			requestData: map[string]interface{}{
				"messages": []interface{}{
					"not an object",
				},
			},
			wantErr:     true,
			errContains: "must be an object",
		},
		{
			name: "invalid content type",
			requestData: map[string]interface{}{
				"messages": []interface{}{
					map[string]interface{}{
						"role":    "user",
						"content": 123, // Neither string nor array
					},
				},
			},
			wantErr:     true,
			errContains: "must be string or array",
		},
		{
			name: "empty content array",
			requestData: map[string]interface{}{
				"messages": []interface{}{
					map[string]interface{}{
						"role":    "user",
						"content": []interface{}{},
					},
				},
			},
			wantErr:     true,
			errContains: "content array cannot be empty",
		},
		{
			name: "content part not object",
			requestData: map[string]interface{}{
				"messages": []interface{}{
					map[string]interface{}{
						"role": "user",
						"content": []interface{}{
							"not an object",
						},
					},
				},
			},
			wantErr:     true,
			errContains: "must be an object",
		},
		{
			name: "content part missing type",
			requestData: map[string]interface{}{
				"messages": []interface{}{
					map[string]interface{}{
						"role": "user",
						"content": []interface{}{
							map[string]interface{}{
								"text": "Missing type field",
							},
						},
					},
				},
			},
			wantErr:     true,
			errContains: "missing 'type' field",
		},
		{
			name: "text content missing text field",
			requestData: map[string]interface{}{
				"messages": []interface{}{
					map[string]interface{}{
						"role": "user",
						"content": []interface{}{
							map[string]interface{}{
								"type": "text",
								// Missing text field
							},
						},
					},
				},
			},
			wantErr:     true,
			errContains: "missing 'text' field",
		},
		{
			name: "image_url content missing image_url field",
			requestData: map[string]interface{}{
				"messages": []interface{}{
					map[string]interface{}{
						"role": "user",
						"content": []interface{}{
							map[string]interface{}{
								"type": "image_url",
								// Missing image_url field
							},
						},
					},
				},
			},
			wantErr:     true,
			errContains: "missing 'image_url' field",
		},
		{
			name: "image_url content missing url field",
			requestData: map[string]interface{}{
				"messages": []interface{}{
					map[string]interface{}{
						"role": "user",
						"content": []interface{}{
							map[string]interface{}{
								"type":      "image_url",
								"image_url": map[string]interface{}{
									// Missing url field
								},
							},
						},
					},
				},
			},
			wantErr:     true,
			errContains: "missing 'url' field",
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
							},
						},
					},
				},
			},
			wantErr:     true,
			errContains: "unknown content type",
		},
		{
			name: "mixed valid content types",
			requestData: map[string]interface{}{
				"messages": []interface{}{
					map[string]interface{}{
						"role":    "system",
						"content": "You are helpful",
					},
					map[string]interface{}{
						"role":    "user",
						"content": "Text message",
					},
					map[string]interface{}{
						"role": "user",
						"content": []interface{}{
							map[string]interface{}{
								"type": "text",
								"text": "Describe this:",
							},
							map[string]interface{}{
								"type": "image_url",
								"image_url": map[string]interface{}{
									"url": "data:image/png;base64,iVBORw0KG...",
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "message without content (valid for some cases)",
			requestData: map[string]interface{}{
				"messages": []interface{}{
					map[string]interface{}{
						"role": "assistant",
						// No content field - valid for tool calls
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMessageContent(tt.requestData)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateContentArray(t *testing.T) {
	tests := []struct {
		name        string
		content     []interface{}
		wantErr     bool
		errContains string
	}{
		{
			name: "valid text part",
			content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "Hello",
				},
			},
			wantErr: false,
		},
		{
			name: "valid image_url part",
			content: []interface{}{
				map[string]interface{}{
					"type": "image_url",
					"image_url": map[string]interface{}{
						"url": "https://example.com/image.png",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple valid parts",
			content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "Look at these images:",
				},
				map[string]interface{}{
					"type": "image_url",
					"image_url": map[string]interface{}{
						"url": "https://example.com/image1.png",
					},
				},
				map[string]interface{}{
					"type": "image_url",
					"image_url": map[string]interface{}{
						"url": "data:image/png;base64,iVBORw0KG...",
					},
				},
			},
			wantErr: false,
		},
		{
			name:        "empty array",
			content:     []interface{}{},
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name: "invalid part type",
			content: []interface{}{
				"not an object",
			},
			wantErr:     true,
			errContains: "must be an object",
		},
		{
			name: "missing type field",
			content: []interface{}{
				map[string]interface{}{
					"text": "No type field",
				},
			},
			wantErr:     true,
			errContains: "missing 'type' field",
		},
		{
			name: "invalid type value",
			content: []interface{}{
				map[string]interface{}{
					"type": 123, // Not a string
				},
			},
			wantErr:     true,
			errContains: "missing 'type' field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateContentArray(tt.content)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
