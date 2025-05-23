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