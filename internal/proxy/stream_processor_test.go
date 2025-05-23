package proxy

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStreamProcessor(t *testing.T) {
	sp := NewStreamProcessor("test-id", 123456789, "fp_test", "openai", "original-model")
	
	assert.Equal(t, "test-id", sp.ConversationID)
	assert.Equal(t, int64(123456789), sp.Timestamp)
	assert.Equal(t, "fp_test", sp.SystemFingerprint)
	assert.Equal(t, "openai", sp.Vendor)
	assert.Equal(t, "original-model", sp.OriginalModel)
	assert.True(t, sp.isFirstChunk)
}

func TestProcessChunk_InvalidSSE(t *testing.T) {
	sp := NewStreamProcessor("test-id", 123456789, "fp_test", "openai", "original-model")
	
	tests := []struct {
		name  string
		chunk []byte
	}{
		{
			name:  "empty chunk",
			chunk: []byte{},
		},
		{
			name:  "no data prefix",
			chunk: []byte("invalid chunk"),
		},
		{
			name:  "done marker",
			chunk: []byte("data: [DONE]"),
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sp.ProcessChunk(tt.chunk)
			assert.Equal(t, tt.chunk, result) // Returns original
		})
	}
}

func TestProcessChunk_InvalidJSON(t *testing.T) {
	sp := NewStreamProcessor("test-id", 123456789, "fp_test", "openai", "original-model")
	
	chunk := []byte("data: invalid json")
	result := sp.ProcessChunk(chunk)
	assert.Equal(t, chunk, result) // Returns original on JSON error
}

func TestProcessChunk_BasicStreaming(t *testing.T) {
	sp := NewStreamProcessor("test-id", 123456789, "fp_test", "openai", "my-custom-model")
	
	chunkData := map[string]interface{}{
		"id":     "chatcmpl-xxx",
		"model":  "gpt-4",
		"created": 999999999,
		"choices": []interface{}{
			map[string]interface{}{
				"delta": map[string]interface{}{
					"content": "Hello",
				},
			},
		},
	}
	
	jsonData, _ := json.Marshal(chunkData)
	chunk := []byte("data: " + string(jsonData))
	
	result := sp.ProcessChunk(chunk)
	
	// Parse result
	resultStr := string(result)
	assert.True(t, strings.HasPrefix(resultStr, "data: "))
	assert.True(t, strings.HasSuffix(resultStr, "\n\n"))
	
	// Extract JSON
	jsonStr := strings.TrimPrefix(resultStr, "data: ")
	jsonStr = strings.TrimSuffix(jsonStr, "\n\n")
	
	var processedData map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &processedData)
	require.NoError(t, err)
	
	// Verify conversation consistency
	assert.Equal(t, "test-id", processedData["id"])
	assert.Equal(t, float64(123456789), processedData["created"])
	assert.Equal(t, "fp_test", processedData["system_fingerprint"])
	assert.Equal(t, "my-custom-model", processedData["model"])
	assert.Equal(t, "default", processedData["service_tier"])
}

func TestProcessChunk_FirstChunkWithRole(t *testing.T) {
	sp := NewStreamProcessor("test-id", 123456789, "fp_test", "openai", "original-model")
	
	chunkData := map[string]interface{}{
		"choices": []interface{}{
			map[string]interface{}{
				"delta": map[string]interface{}{
					"role": "assistant", // First chunk indicator
				},
			},
		},
	}
	
	jsonData, _ := json.Marshal(chunkData)
	chunk := []byte("data: " + string(jsonData))
	
	result := sp.ProcessChunk(chunk)
	
	// Parse result
	resultStr := strings.TrimPrefix(string(result), "data: ")
	resultStr = strings.TrimSuffix(resultStr, "\n\n")
	
	var processedData map[string]interface{}
	err := json.Unmarshal([]byte(resultStr), &processedData)
	require.NoError(t, err)
	
	// Should add usage for first chunk
	usage, exists := processedData["usage"].(map[string]interface{})
	assert.True(t, exists)
	assert.NotNil(t, usage)
	assert.Equal(t, float64(0), usage["prompt_tokens"])
}

func TestProcessChunk_ToolCallsInDelta(t *testing.T) {
	sp := NewStreamProcessor("test-id", 123456789, "fp_test", "openai", "original-model")
	
	chunkData := map[string]interface{}{
		"choices": []interface{}{
			map[string]interface{}{
				"delta": map[string]interface{}{
					"tool_calls": []interface{}{
						map[string]interface{}{
							"type": "function",
							"function": map[string]interface{}{
								"name": "get_weather",
							},
						},
					},
				},
			},
		},
	}
	
	jsonData, _ := json.Marshal(chunkData)
	chunk := []byte("data: " + string(jsonData))
	
	result := sp.ProcessChunk(chunk)
	
	// Parse result
	resultStr := strings.TrimPrefix(string(result), "data: ")
	resultStr = strings.TrimSuffix(resultStr, "\n\n")
	
	var processedData map[string]interface{}
	err := json.Unmarshal([]byte(resultStr), &processedData)
	require.NoError(t, err)
	
	// Check tool calls were processed
	choices := processedData["choices"].([]interface{})
	choice := choices[0].(map[string]interface{})
	delta := choice["delta"].(map[string]interface{})
	toolCalls := delta["tool_calls"].([]interface{})
	toolCall := toolCalls[0].(map[string]interface{})
	
	assert.NotNil(t, toolCall["id"])
	
	// Check delta has required fields
	assert.NotNil(t, delta["annotations"])
	assert.Nil(t, delta["refusal"])
}

func TestProcessChunk_MessageInsteadOfDelta(t *testing.T) {
	sp := NewStreamProcessor("test-id", 123456789, "fp_test", "openai", "original-model")
	
	chunkData := map[string]interface{}{
		"choices": []interface{}{
			map[string]interface{}{
				"message": map[string]interface{}{ // Some vendors might send message
					"role":    "assistant",
					"content": "Hello",
				},
			},
		},
	}
	
	jsonData, _ := json.Marshal(chunkData)
	chunk := []byte("data: " + string(jsonData))
	
	result := sp.ProcessChunk(chunk)
	
	// Parse result
	resultStr := strings.TrimPrefix(string(result), "data: ")
	resultStr = strings.TrimSuffix(resultStr, "\n\n")
	
	var processedData map[string]interface{}
	err := json.Unmarshal([]byte(resultStr), &processedData)
	require.NoError(t, err)
	
	// Check message was processed
	choices := processedData["choices"].([]interface{})
	choice := choices[0].(map[string]interface{})
	message := choice["message"].(map[string]interface{})
	
	assert.NotNil(t, message["annotations"])
	assert.Nil(t, message["refusal"])
}

func TestProcessChunk_NoChoices(t *testing.T) {
	sp := NewStreamProcessor("test-id", 123456789, "fp_test", "openai", "original-model")
	
	chunkData := map[string]interface{}{
		"model": "gpt-4",
		// No choices field
	}
	
	jsonData, _ := json.Marshal(chunkData)
	chunk := []byte("data: " + string(jsonData))
	
	result := sp.ProcessChunk(chunk)
	
	// Should still process and add consistency fields
	resultStr := strings.TrimPrefix(string(result), "data: ")
	resultStr = strings.TrimSuffix(resultStr, "\n\n")
	
	var processedData map[string]interface{}
	err := json.Unmarshal([]byte(resultStr), &processedData)
	require.NoError(t, err)
	
	assert.Equal(t, "test-id", processedData["id"])
	assert.Equal(t, "original-model", processedData["model"])
}

func TestProcessChunk_LogprobsAdded(t *testing.T) {
	sp := NewStreamProcessor("test-id", 123456789, "fp_test", "openai", "original-model")
	
	chunkData := map[string]interface{}{
		"choices": []interface{}{
			map[string]interface{}{
				"delta": map[string]interface{}{
					"content": "Hello",
				},
				// logprobs missing
			},
		},
	}
	
	jsonData, _ := json.Marshal(chunkData)
	chunk := []byte("data: " + string(jsonData))
	
	result := sp.ProcessChunk(chunk)
	
	// Parse result
	resultStr := strings.TrimPrefix(string(result), "data: ")
	resultStr = strings.TrimSuffix(resultStr, "\n\n")
	
	var processedData map[string]interface{}
	err := json.Unmarshal([]byte(resultStr), &processedData)
	require.NoError(t, err)
	
	choices := processedData["choices"].([]interface{})
	choice := choices[0].(map[string]interface{})
	assert.Nil(t, choice["logprobs"])
}

func TestProcessChunk_GeminiVendor(t *testing.T) {
	sp := NewStreamProcessor("test-id", 123456789, "fp_test", "gemini", "original-model")
	
	chunkData := map[string]interface{}{
		"choices": []interface{}{
			map[string]interface{}{
				"delta": map[string]interface{}{
					"tool_calls": []interface{}{
						map[string]interface{}{
							"id":   "existing_id",
							"type": "function",
							"function": map[string]interface{}{
								"name": "test",
							},
						},
					},
				},
			},
		},
	}
	
	jsonData, _ := json.Marshal(chunkData)
	chunk := []byte("data: " + string(jsonData))
	
	result := sp.ProcessChunk(chunk)
	
	// Parse result
	resultStr := strings.TrimPrefix(string(result), "data: ")
	resultStr = strings.TrimSuffix(resultStr, "\n\n")
	
	var processedData map[string]interface{}
	err := json.Unmarshal([]byte(resultStr), &processedData)
	require.NoError(t, err)
	
	// Check Gemini tool call ID override
	choices := processedData["choices"].([]interface{})
	choice := choices[0].(map[string]interface{})
	delta := choice["delta"].(map[string]interface{})
	toolCalls := delta["tool_calls"].([]interface{})
	toolCall := toolCalls[0].(map[string]interface{})
	
	assert.NotEqual(t, "existing_id", toolCall["id"])
	assert.True(t, strings.HasPrefix(toolCall["id"].(string), "call_"))
}

func TestStreamProcessor_StateTracking(t *testing.T) {
	sp := NewStreamProcessor("test-id", 123456789, "fp_test", "openai", "original-model")
	
	// First chunk
	assert.True(t, sp.isFirstChunk)
	
	firstChunk := map[string]interface{}{
		"choices": []interface{}{
			map[string]interface{}{
				"delta": map[string]interface{}{
					"role": "assistant",
				},
			},
		},
	}
	
	jsonData, _ := json.Marshal(firstChunk)
	chunk := []byte("data: " + string(jsonData))
	sp.ProcessChunk(chunk)
	
	// After first chunk
	assert.False(t, sp.isFirstChunk)
	
	// Process another chunk
	secondChunk := map[string]interface{}{
		"choices": []interface{}{
			map[string]interface{}{
				"delta": map[string]interface{}{
					"content": "Hello",
				},
			},
		},
	}
	
	jsonData2, _ := json.Marshal(secondChunk)
	chunk2 := []byte("data: " + string(jsonData2))
	result := sp.ProcessChunk(chunk2)
	
	// Second chunk should not have usage
	resultStr := strings.TrimPrefix(string(result), "data: ")
	resultStr = strings.TrimSuffix(resultStr, "\n\n")
	
	var processedData map[string]interface{}
	err := json.Unmarshal([]byte(resultStr), &processedData)
	require.NoError(t, err)
	
	_, hasUsage := processedData["usage"]
	assert.False(t, hasUsage)
} 