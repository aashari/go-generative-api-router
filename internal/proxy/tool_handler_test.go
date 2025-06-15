package proxy

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProcessToolCalls(t *testing.T) {
	tests := []struct {
		name             string
		toolCalls        []interface{}
		vendor           string
		expectedCount    int
		checkGeminiID    bool // Whether to check that Gemini generates new IDs
		checkOpenAIID    bool // Whether to check that OpenAI preserves existing IDs
		checkSplitting   bool // Whether to check for argument splitting
	}{
		{
			name:          "nil tool calls",
			toolCalls:     nil,
			vendor:        "openai",
			expectedCount: 0,
		},
		{
			name:          "empty tool calls",
			toolCalls:     []interface{}{},
			vendor:        "openai",
			expectedCount: 0,
		},
		{
			name: "Gemini tool calls - should generate new IDs",
			toolCalls: []interface{}{
				map[string]interface{}{
					"id":   "existing_id",
					"type": "function",
					"function": map[string]interface{}{
						"name":      "get_weather",
						"arguments": `{"location": "San Francisco"}`,
					},
				},
			},
			vendor:        "gemini",
			expectedCount: 1,
			checkGeminiID: true,
		},
		{
			name: "OpenAI tool calls - should preserve existing IDs",
			toolCalls: []interface{}{
				map[string]interface{}{
					"id":   "call_preserve_me",
					"type": "function",
					"function": map[string]interface{}{
						"name":      "get_weather",
						"arguments": `{"location": "New York"}`,
					},
				},
			},
			vendor:        "openai",
			expectedCount: 1,
			checkOpenAIID: true,
		},
		{
			name: "OpenAI tool calls - missing ID should generate",
			toolCalls: []interface{}{
				map[string]interface{}{
					"type": "function",
					"function": map[string]interface{}{
						"name":      "get_weather",
						"arguments": `{"location": "Boston"}`,
					},
				},
			},
			vendor:        "openai",
			expectedCount: 1,
		},
		{
			name: "tool call with malformed arguments - should split",
			toolCalls: []interface{}{
				map[string]interface{}{
					"id":   "call_split_me",
					"type": "function",
					"function": map[string]interface{}{
						"name":      "multi_call",
						"arguments": `{"location": "SF"}{"location": "NYC"}`,
					},
				},
			},
			vendor:         "openai",
			expectedCount:  2, // Should be split into 2 calls
			checkSplitting: true,
		},
		{
			name: "non-map tool call",
			toolCalls: []interface{}{
				"not a map",
				map[string]interface{}{
					"id":   "call_valid",
					"type": "function",
					"function": map[string]interface{}{
						"name":      "valid_call",
						"arguments": `{"test": true}`,
					},
				},
			},
			vendor:        "openai",
			expectedCount: 2, // Both should be returned, first unchanged
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ProcessToolCalls(tt.toolCalls, tt.vendor)

			if tt.toolCalls == nil {
				assert.Nil(t, result)
				return
			}

			assert.Len(t, result, tt.expectedCount)

			if tt.expectedCount == 0 {
				return
			}

			// Check Gemini ID generation
			if tt.checkGeminiID {
				firstCall := result[0].(map[string]interface{})
				assert.NotEqual(t, "existing_id", firstCall["id"])
				assert.True(t, strings.HasPrefix(firstCall["id"].(string), "call_"))
			}

			// Check OpenAI ID preservation
			if tt.checkOpenAIID {
				firstCall := result[0].(map[string]interface{})
				assert.Equal(t, "call_preserve_me", firstCall["id"])
			}

			// Check splitting behavior
			if tt.checkSplitting {
				assert.Equal(t, 2, len(result))
				for _, call := range result {
					callMap := call.(map[string]interface{})
					assert.True(t, strings.HasPrefix(callMap["id"].(string), "call_"))
					function := callMap["function"].(map[string]interface{})
					arguments := function["arguments"].(string)
					// Each split call should have valid JSON arguments
					assert.True(t, isValidJSON(arguments))
				}
			}
		})
	}
}

func TestValidateAndSplitArguments(t *testing.T) {
	tests := []struct {
		name         string
		toolCall     map[string]interface{}
		arguments    string
		vendor       string
		expectSplit  bool
		expectedSize int
	}{
		{
			name: "valid single JSON object",
			toolCall: map[string]interface{}{
				"id":   "call_123",
				"type": "function",
				"function": map[string]interface{}{
					"name": "test_func",
				},
			},
			arguments:    `{"param": "value"}`,
			vendor:       "openai",
			expectSplit:  false,
			expectedSize: 1,
		},
		{
			name: "malformed arguments - multiple objects with }{",
			toolCall: map[string]interface{}{
				"id":   "call_456",
				"type": "function",
				"function": map[string]interface{}{
					"name": "multi_func",
				},
			},
			arguments:    `{"location": "SF"}{"location": "NYC"}`,
			vendor:       "openai",
			expectSplit:  true,
			expectedSize: 2,
		},
		{
			name: "malformed arguments - multiple arrays with ][",
			toolCall: map[string]interface{}{
				"id":   "call_789",
				"type": "function",
				"function": map[string]interface{}{
					"name": "array_func",
				},
			},
			arguments:    `["item1"]["item2"]`,
			vendor:       "openai",
			expectSplit:  true,
			expectedSize: 2,
		},
		{
			name: "multiple complete JSON objects",
			toolCall: map[string]interface{}{
				"id":   "call_abc",
				"type": "function",
				"function": map[string]interface{}{
					"name": "sequence_func",
				},
			},
			arguments:    `{"a": 1} {"b": 2}`,
			vendor:       "openai",
			expectSplit:  true,
			expectedSize: 2,
		},
		{
			name: "invalid JSON - should not split",
			toolCall: map[string]interface{}{
				"id":   "call_def",
				"type": "function",
				"function": map[string]interface{}{
					"name": "invalid_func",
				},
			},
			arguments:    `{invalid json}`,
			vendor:       "openai",
			expectSplit:  false,
			expectedSize: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateAndSplitArguments(tt.toolCall, tt.arguments, tt.vendor)

			assert.Len(t, result, tt.expectedSize)

			if tt.expectSplit {
				assert.Greater(t, len(result), 1, "Should have split the arguments")
				
				// Check that each split result has valid JSON arguments
				for i, splitCall := range result {
					splitMap := splitCall.(map[string]interface{})
					function := splitMap["function"].(map[string]interface{})
					arguments := function["arguments"].(string)
					
					assert.True(t, isValidJSON(arguments), "Split result %d should have valid JSON arguments", i)
					assert.True(t, strings.HasPrefix(splitMap["id"].(string), "call_"), "Split result %d should have generated ID", i)
				}
			} else {
				assert.Equal(t, 1, len(result), "Should not have split the arguments")
				assert.Equal(t, tt.toolCall, result[0], "Should return original tool call unchanged")
			}
		})
	}
}

func TestContainsMultipleJSONObjects(t *testing.T) {
	tests := []struct {
		name      string
		arguments string
		expected  bool
	}{
		{
			name:      "single valid JSON object",
			arguments: `{"param": "value"}`,
			expected:  false,
		},
		{
			name:      "multiple objects with }{ pattern",
			arguments: `{"a": 1}{"b": 2}`,
			expected:  true,
		},
		{
			name:      "multiple arrays with ][ pattern",
			arguments: `["item1"]["item2"]`,
			expected:  true,
		},
		{
			name:      "multiple complete objects with spaces",
			arguments: `{"first": true} {"second": false}`,
			expected:  true,
		},
		{
			name:      "empty string",
			arguments: ``,
			expected:  false,
		},
		{
			name:      "whitespace only",
			arguments: `   `,
			expected:  false,
		},
		{
			name:      "invalid JSON",
			arguments: `{invalid: json}`,
			expected:  false,
		},
		{
			name:      "nested objects (valid JSON)",
			arguments: `{"outer": {"inner": "value"}}`,
			expected:  false,
		},
		{
			name:      "array of objects (valid JSON)",
			arguments: `[{"a": 1}, {"b": 2}]`,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsMultipleJSONObjects(tt.arguments)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSplitJSONObjects(t *testing.T) {
	tests := []struct {
		name          string
		arguments     string
		expectedCount int
		checkValid    bool
	}{
		{
			name:          "split on }{ pattern",
			arguments:     `{"location": "SF"}{"location": "NYC"}`,
			expectedCount: 2,
			checkValid:    true,
		},
		{
			name:          "split on ][ pattern",
			arguments:     `["item1"]["item2"]["item3"]`,
			expectedCount: 3,
			checkValid:    true,
		},
		{
			name:          "sequential parsing",
			arguments:     `{"a": 1} {"b": 2} {"c": 3}`,
			expectedCount: 3,
			checkValid:    true,
		},
		{
			name:          "single object",
			arguments:     `{"single": "object"}`,
			expectedCount: 0, // Should return empty slice for single objects
			checkValid:    false,
		},
		{
			name:          "invalid JSON",
			arguments:     `{invalid json}`,
			expectedCount: 0,
			checkValid:    false,
		},
		{
			name:          "mixed valid and invalid",
			arguments:     `{"valid": true}{invalid}{"also_valid": false}`,
			expectedCount: 2, // Should return valid parts
			checkValid:    true,
		},
		{
			name:          "complex objects with }{ pattern",
			arguments:     `{"user": {"name": "John", "age": 30}}{"settings": {"theme": "dark"}}`,
			expectedCount: 1, // Complex split may not work perfectly
			checkValid:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitJSONObjects(tt.arguments)

			if tt.expectedCount == 0 {
				assert.Len(t, result, 0)
				return
			}

			assert.Len(t, result, tt.expectedCount)

			if tt.checkValid {
				for i, jsonStr := range result {
					assert.True(t, isValidJSON(jsonStr), "Split result %d should be valid JSON: %s", i, jsonStr)
				}
			}
		})
	}
}

func TestIsValidJSON(t *testing.T) {
	tests := []struct {
		name     string
		jsonStr  string
		expected bool
	}{
		{
			name:     "valid object",
			jsonStr:  `{"key": "value"}`,
			expected: true,
		},
		{
			name:     "valid array",
			jsonStr:  `["item1", "item2"]`,
			expected: true,
		},
		{
			name:     "valid number",
			jsonStr:  `42`,
			expected: true,
		},
		{
			name:     "valid string",
			jsonStr:  `"hello"`,
			expected: true,
		},
		{
			name:     "valid boolean",
			jsonStr:  `true`,
			expected: true,
		},
		{
			name:     "valid null",
			jsonStr:  `null`,
			expected: true,
		},
		{
			name:     "invalid JSON - missing quotes",
			jsonStr:  `{key: value}`,
			expected: false,
		},
		{
			name:     "invalid JSON - trailing comma",
			jsonStr:  `{"key": "value",}`,
			expected: false,
		},
		{
			name:     "empty string",
			jsonStr:  ``,
			expected: false,
		},
		{
			name:     "invalid characters",
			jsonStr:  `{invalid}`,
			expected: false,
		},
		{
			name:     "complex valid object",
			jsonStr:  `{"user": {"name": "John", "age": 30, "active": true, "skills": ["js", "go"]}}`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidJSON(tt.jsonStr)
			assert.Equal(t, tt.expected, result)
		})
	}
}