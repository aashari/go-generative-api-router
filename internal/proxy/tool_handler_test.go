package proxy

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProcessToolCalls_NilInput(t *testing.T) {
	result := ProcessToolCalls(nil, "openai")
	assert.Nil(t, result)
}

func TestProcessToolCalls_EmptyArray(t *testing.T) {
	result := ProcessToolCalls([]interface{}{}, "openai")
	assert.Empty(t, result)
}

func TestProcessToolCalls_InvalidToolCallType(t *testing.T) {
	toolCalls := []interface{}{
		"not a map", // Invalid type
		123,         // Another invalid type
	}

	result := ProcessToolCalls(toolCalls, "openai")

	// Should return the same array unchanged
	assert.Equal(t, toolCalls, result)
}

func TestProcessToolCalls_MissingID(t *testing.T) {
	toolCalls := []interface{}{
		map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name": "test_function",
			},
		},
	}

	result := ProcessToolCalls(toolCalls, "openai")

	// Should add an ID
	toolCall := result[0].(map[string]interface{})
	id, exists := toolCall["id"]
	assert.True(t, exists, "ID should be added")
	assert.NotNil(t, id)
	assert.True(t, strings.HasPrefix(id.(string), "call_"), "ID should have correct prefix")
}

func TestProcessToolCalls_EmptyID(t *testing.T) {
	toolCalls := []interface{}{
		map[string]interface{}{
			"id":   "",
			"type": "function",
			"function": map[string]interface{}{
				"name": "test_function",
			},
		},
	}

	result := ProcessToolCalls(toolCalls, "openai")

	// Should replace empty ID
	toolCall := result[0].(map[string]interface{})
	id := toolCall["id"].(string)
	assert.NotEmpty(t, id)
	assert.True(t, strings.HasPrefix(id, "call_"), "ID should have correct prefix")
}

func TestProcessToolCalls_NilID(t *testing.T) {
	toolCalls := []interface{}{
		map[string]interface{}{
			"id":   nil,
			"type": "function",
			"function": map[string]interface{}{
				"name": "test_function",
			},
		},
	}

	result := ProcessToolCalls(toolCalls, "openai")

	// Should replace nil ID
	toolCall := result[0].(map[string]interface{})
	id := toolCall["id"].(string)
	assert.NotNil(t, id)
	assert.True(t, strings.HasPrefix(id, "call_"), "ID should have correct prefix")
}

func TestProcessToolCalls_ExistingID_NonGemini(t *testing.T) {
	existingID := "existing_id_123"
	toolCalls := []interface{}{
		map[string]interface{}{
			"id":   existingID,
			"type": "function",
			"function": map[string]interface{}{
				"name": "test_function",
			},
		},
	}

	result := ProcessToolCalls(toolCalls, "openai")

	// Should keep existing ID for non-Gemini vendors
	toolCall := result[0].(map[string]interface{})
	assert.Equal(t, existingID, toolCall["id"])
}

func TestProcessToolCalls_GeminiAlwaysOverrides(t *testing.T) {
	existingID := "existing_gemini_id"
	toolCalls := []interface{}{
		map[string]interface{}{
			"id":   existingID,
			"type": "function",
			"function": map[string]interface{}{
				"name": "test_function",
			},
		},
	}

	result := ProcessToolCalls(toolCalls, "gemini")

	// Should override existing ID for Gemini
	toolCall := result[0].(map[string]interface{})
	newID := toolCall["id"].(string)
	assert.NotEqual(t, existingID, newID)
	assert.True(t, strings.HasPrefix(newID, "call_"), "ID should have correct prefix")
}

func TestProcessToolCalls_MultipleToolCalls(t *testing.T) {
	toolCalls := []interface{}{
		map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name": "function1",
			},
		},
		map[string]interface{}{
			"id":   "existing_id",
			"type": "function",
			"function": map[string]interface{}{
				"name": "function2",
			},
		},
		map[string]interface{}{
			"id":   "",
			"type": "function",
			"function": map[string]interface{}{
				"name": "function3",
			},
		},
	}

	result := ProcessToolCalls(toolCalls, "openai")

	// First tool call should get new ID
	toolCall1 := result[0].(map[string]interface{})
	id1 := toolCall1["id"].(string)
	assert.NotEmpty(t, id1)
	assert.True(t, strings.HasPrefix(id1, "call_"))

	// Second tool call should keep existing ID
	toolCall2 := result[1].(map[string]interface{})
	assert.Equal(t, "existing_id", toolCall2["id"])

	// Third tool call should get new ID (empty string)
	toolCall3 := result[2].(map[string]interface{})
	id3 := toolCall3["id"].(string)
	assert.NotEmpty(t, id3)
	assert.True(t, strings.HasPrefix(id3, "call_"))

	// All generated IDs should be unique
	assert.NotEqual(t, id1, id3)
}

func TestProcessToolCalls_MixedValidInvalid(t *testing.T) {
	toolCalls := []interface{}{
		map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name": "valid_function",
			},
		},
		"invalid tool call", // Should be skipped
		map[string]interface{}{
			"id":   "existing_id",
			"type": "function",
			"function": map[string]interface{}{
				"name": "another_valid",
			},
		},
	}

	result := ProcessToolCalls(toolCalls, "openai")

	// Should process valid tool calls and skip invalid ones
	assert.Len(t, result, 3) // Original length preserved

	// First valid tool call should get ID
	toolCall1 := result[0].(map[string]interface{})
	assert.NotNil(t, toolCall1["id"])

	// Invalid one stays unchanged
	assert.Equal(t, "invalid tool call", result[1])

	// Last valid tool call keeps its ID
	toolCall3 := result[2].(map[string]interface{})
	assert.Equal(t, "existing_id", toolCall3["id"])
}

func TestProcessToolCalls_PreservesOtherFields(t *testing.T) {
	toolCalls := []interface{}{
		map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":      "test_function",
				"arguments": "{\"param\": \"value\"}",
			},
			"extra_field": "should_be_preserved",
		},
	}

	result := ProcessToolCalls(toolCalls, "openai")

	// Should add ID but preserve all other fields
	toolCall := result[0].(map[string]interface{})
	assert.NotNil(t, toolCall["id"])
	assert.Equal(t, "function", toolCall["type"])
	assert.Equal(t, "should_be_preserved", toolCall["extra_field"])

	function := toolCall["function"].(map[string]interface{})
	assert.Equal(t, "test_function", function["name"])
	assert.Equal(t, "{\"param\": \"value\"}", function["arguments"])
}

// New tests for malformed argument validation and splitting

func TestProcessToolCalls_SplitMalformedArguments_BracePattern(t *testing.T) {
	// Test the }{ pattern that indicates multiple JSON objects
	toolCalls := []interface{}{
		map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":      "shell",
				"arguments": "{\"command\":[\"grep\",\"-r\",\"-E\",\"import \\\"log\\\"\",\"--include\",\"*.go\",\".\"]}{\"command\":[\"grep\",\"-r\",\"-E\",\"log.Printf|log.Println|log.Fatal\",\"--include\",\"*.go\",\".\"]}",
			},
		},
	}

	result := ProcessToolCalls(toolCalls, "openai")

	// Should split into 2 separate tool calls
	assert.Len(t, result, 2, "Should split malformed arguments into 2 tool calls")

	// Check first tool call
	toolCall1 := result[0].(map[string]interface{})
	assert.Equal(t, "function", toolCall1["type"])
	assert.NotNil(t, toolCall1["id"])
	assert.True(t, strings.HasPrefix(toolCall1["id"].(string), "call_"))

	function1 := toolCall1["function"].(map[string]interface{})
	assert.Equal(t, "shell", function1["name"])
	assert.Equal(t, "{\"command\":[\"grep\",\"-r\",\"-E\",\"import \\\"log\\\"\",\"--include\",\"*.go\",\".\"]}", function1["arguments"])

	// Check second tool call
	toolCall2 := result[1].(map[string]interface{})
	assert.Equal(t, "function", toolCall2["type"])
	assert.NotNil(t, toolCall2["id"])
	assert.True(t, strings.HasPrefix(toolCall2["id"].(string), "call_"))

	function2 := toolCall2["function"].(map[string]interface{})
	assert.Equal(t, "shell", function2["name"])
	assert.Equal(t, "{\"command\":[\"grep\",\"-r\",\"-E\",\"log.Printf|log.Println|log.Fatal\",\"--include\",\"*.go\",\".\"]}", function2["arguments"])

	// IDs should be different
	assert.NotEqual(t, toolCall1["id"], toolCall2["id"])
}

func TestProcessToolCalls_SplitMalformedArguments_BracketPattern(t *testing.T) {
	// Test the ][ pattern for arrays
	toolCalls := []interface{}{
		map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":      "process_arrays",
				"arguments": "[\"item1\",\"item2\"][\"item3\",\"item4\"]",
			},
		},
	}

	result := ProcessToolCalls(toolCalls, "openai")

	// Should split into 2 separate tool calls
	assert.Len(t, result, 2, "Should split malformed array arguments into 2 tool calls")

	// Check first tool call
	toolCall1 := result[0].(map[string]interface{})
	function1 := toolCall1["function"].(map[string]interface{})
	assert.Equal(t, "[\"item1\",\"item2\"]", function1["arguments"])

	// Check second tool call
	toolCall2 := result[1].(map[string]interface{})
	function2 := toolCall2["function"].(map[string]interface{})
	assert.Equal(t, "[\"item3\",\"item4\"]", function2["arguments"])
}

func TestProcessToolCalls_ValidArguments_NoSplit(t *testing.T) {
	// Test that valid arguments are not split
	toolCalls := []interface{}{
		map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":      "valid_function",
				"arguments": "{\"param1\": \"value1\", \"param2\": [\"item1\", \"item2\"]}",
			},
		},
	}

	result := ProcessToolCalls(toolCalls, "openai")

	// Should remain as single tool call
	assert.Len(t, result, 1, "Valid arguments should not be split")

	toolCall := result[0].(map[string]interface{})
	function := toolCall["function"].(map[string]interface{})
	assert.Equal(t, "{\"param1\": \"value1\", \"param2\": [\"item1\", \"item2\"]}", function["arguments"])
}

func TestProcessToolCalls_EmptyArguments_NoSplit(t *testing.T) {
	// Test that empty arguments are not split
	toolCalls := []interface{}{
		map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":      "empty_function",
				"arguments": "",
			},
		},
	}

	result := ProcessToolCalls(toolCalls, "openai")

	// Should remain as single tool call
	assert.Len(t, result, 1, "Empty arguments should not be split")
}

func TestProcessToolCalls_InvalidJSON_NoSplit(t *testing.T) {
	// Test that invalid JSON that can't be split properly is left as-is
	toolCalls := []interface{}{
		map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":      "invalid_function",
				"arguments": "{invalid json}{also invalid}",
			},
		},
	}

	result := ProcessToolCalls(toolCalls, "openai")

	// Should remain as single tool call since splitting failed
	assert.Len(t, result, 1, "Invalid JSON should not be split")

	toolCall := result[0].(map[string]interface{})
	function := toolCall["function"].(map[string]interface{})
	assert.Equal(t, "{invalid json}{also invalid}", function["arguments"])
}

func TestProcessToolCalls_MixedValidAndMalformed(t *testing.T) {
	// Test processing multiple tool calls where some are malformed and some are valid
	toolCalls := []interface{}{
		map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":      "valid_function",
				"arguments": "{\"param\": \"value\"}",
			},
		},
		map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":      "malformed_function",
				"arguments": "{\"cmd1\": \"value1\"}{\"cmd2\": \"value2\"}",
			},
		},
		map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":      "another_valid",
				"arguments": "{\"other\": \"param\"}",
			},
		},
	}

	result := ProcessToolCalls(toolCalls, "openai")

	// Should have 4 tool calls total (1 + 2 split + 1)
	assert.Len(t, result, 4, "Should have 4 tool calls after splitting malformed one")

	// Check first (valid)
	toolCall1 := result[0].(map[string]interface{})
	function1 := toolCall1["function"].(map[string]interface{})
	assert.Equal(t, "valid_function", function1["name"])
	assert.Equal(t, "{\"param\": \"value\"}", function1["arguments"])

	// Check second (first split from malformed)
	toolCall2 := result[1].(map[string]interface{})
	function2 := toolCall2["function"].(map[string]interface{})
	assert.Equal(t, "malformed_function", function2["name"])
	assert.Equal(t, "{\"cmd1\": \"value1\"}", function2["arguments"])

	// Check third (second split from malformed)
	toolCall3 := result[2].(map[string]interface{})
	function3 := toolCall3["function"].(map[string]interface{})
	assert.Equal(t, "malformed_function", function3["name"])
	assert.Equal(t, "{\"cmd2\": \"value2\"}", function3["arguments"])

	// Check fourth (valid)
	toolCall4 := result[3].(map[string]interface{})
	function4 := toolCall4["function"].(map[string]interface{})
	assert.Equal(t, "another_valid", function4["name"])
	assert.Equal(t, "{\"other\": \"param\"}", function4["arguments"])

	// All should have unique IDs
	ids := []string{
		toolCall1["id"].(string),
		toolCall2["id"].(string),
		toolCall3["id"].(string),
		toolCall4["id"].(string),
	}
	for i := 0; i < len(ids); i++ {
		for j := i + 1; j < len(ids); j++ {
			assert.NotEqual(t, ids[i], ids[j], "All tool call IDs should be unique")
		}
	}
}

func TestProcessToolCalls_GeminiWithMalformedArguments(t *testing.T) {
	// Test that Gemini vendor still gets new IDs even with splitting
	toolCalls := []interface{}{
		map[string]interface{}{
			"id":   "existing_gemini_id",
			"type": "function",
			"function": map[string]interface{}{
				"name":      "gemini_function",
				"arguments": "{\"param1\": \"value1\"}{\"param2\": \"value2\"}",
			},
		},
	}

	result := ProcessToolCalls(toolCalls, "gemini")

	// Should split into 2 tool calls
	assert.Len(t, result, 2, "Should split malformed Gemini arguments")

	// Both should have new IDs (not the existing one)
	toolCall1 := result[0].(map[string]interface{})
	toolCall2 := result[1].(map[string]interface{})

	assert.NotEqual(t, "existing_gemini_id", toolCall1["id"])
	assert.NotEqual(t, "existing_gemini_id", toolCall2["id"])
	assert.True(t, strings.HasPrefix(toolCall1["id"].(string), "call_"))
	assert.True(t, strings.HasPrefix(toolCall2["id"].(string), "call_"))
	assert.NotEqual(t, toolCall1["id"], toolCall2["id"])
}

// Test helper functions

func TestContainsMultipleJSONObjects(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid single object",
			input:    "{\"param\": \"value\"}",
			expected: false,
		},
		{
			name:     "brace pattern",
			input:    "{\"param1\": \"value1\"}{\"param2\": \"value2\"}",
			expected: true,
		},
		{
			name:     "bracket pattern",
			input:    "[\"item1\"][\"item2\"]",
			expected: true,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "invalid json",
			input:    "not json at all",
			expected: false,
		},
		{
			name:     "nested objects",
			input:    "{\"nested\": {\"inner\": \"value\"}}",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsMultipleJSONObjects(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSplitJSONObjects(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "brace pattern",
			input:    "{\"cmd1\": \"value1\"}{\"cmd2\": \"value2\"}",
			expected: []string{"{\"cmd1\": \"value1\"}", "{\"cmd2\": \"value2\"}"},
		},
		{
			name:     "bracket pattern",
			input:    "[\"item1\"][\"item2\"]",
			expected: []string{"[\"item1\"]", "[\"item2\"]"},
		},
		{
			name:     "three objects",
			input:    "{\"a\": 1}{\"b\": 2}{\"c\": 3}",
			expected: []string{"{\"a\": 1}", "{\"b\": 2}", "{\"c\": 3}"},
		},
		{
			name:     "single object",
			input:    "{\"single\": \"object\"}",
			expected: nil, // No splitting needed, returns nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitJSONObjects(tt.input)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestIsValidJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid object",
			input:    "{\"key\": \"value\"}",
			expected: true,
		},
		{
			name:     "valid array",
			input:    "[\"item1\", \"item2\"]",
			expected: true,
		},
		{
			name:     "invalid json",
			input:    "{invalid}",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "null",
			input:    "null",
			expected: true,
		},
		{
			name:     "number",
			input:    "123",
			expected: true,
		},
		{
			name:     "string",
			input:    "\"hello\"",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidJSON(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
