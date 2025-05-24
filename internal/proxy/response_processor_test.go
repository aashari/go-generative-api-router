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

func TestProcessResponse_EmptyResponse(t *testing.T) {
	result, err := ProcessResponse([]byte{}, "openai", "", "test-model")
	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestProcessResponse_InvalidJSON(t *testing.T) {
	invalidJSON := []byte("not valid json")
	result, err := ProcessResponse(invalidJSON, "openai", "", "test-model")
	assert.NoError(t, err)
	assert.Equal(t, invalidJSON, result) // Returns original on parse error
}

func TestProcessResponse_BasicResponse(t *testing.T) {
	response := map[string]interface{}{
		"model": "gpt-4",
		"choices": []interface{}{
			map[string]interface{}{
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": "Hello",
				},
			},
		},
	}

	responseBytes, _ := json.Marshal(response)
	result, err := ProcessResponse(responseBytes, "openai", "", "my-custom-model")
	assert.NoError(t, err)

	var processedResponse map[string]interface{}
	err = json.Unmarshal(result, &processedResponse)
	require.NoError(t, err)

	// Check model was replaced
	assert.Equal(t, "my-custom-model", processedResponse["model"])

	// Check required fields were added
	assert.NotNil(t, processedResponse["id"])
	assert.Equal(t, "default", processedResponse["service_tier"])
	assert.NotNil(t, processedResponse["system_fingerprint"])
	assert.NotNil(t, processedResponse["usage"])
}

func TestProcessResponse_GzipCompressed(t *testing.T) {
	response := map[string]interface{}{
		"model":   "gpt-4",
		"choices": []interface{}{},
	}

	// Compress the response
	responseBytes, _ := json.Marshal(response)
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	gzipWriter.Write(responseBytes)
	gzipWriter.Close()

	result, err := ProcessResponse(buf.Bytes(), "openai", "gzip", "test-model")
	assert.NoError(t, err)

	var processedResponse map[string]interface{}
	err = json.Unmarshal(result, &processedResponse)
	require.NoError(t, err)

	assert.Equal(t, "test-model", processedResponse["model"])
}

func TestProcessResponse_ArrayResponse(t *testing.T) {
	// Simulate Gemini-style single-element array response
	arrayResponse := []interface{}{
		map[string]interface{}{
			"model":   "gemini-pro",
			"choices": []interface{}{},
		},
	}

	responseBytes, _ := json.Marshal(arrayResponse)
	result, err := ProcessResponse(responseBytes, "gemini", "", "my-model")
	assert.NoError(t, err)

	// Should unwrap the array
	var processedResponse map[string]interface{}
	err = json.Unmarshal(result, &processedResponse)
	require.NoError(t, err)

	assert.Equal(t, "my-model", processedResponse["model"])
}

func TestProcessResponse_ErrorResponse(t *testing.T) {
	errorResponse := map[string]interface{}{
		"error": map[string]interface{}{
			"message": "API key invalid",
			"code":    "invalid_api_key",
		},
	}

	responseBytes, _ := json.Marshal(errorResponse)
	result, err := ProcessResponse(responseBytes, "openai", "", "test-model")
	assert.NoError(t, err)

	var processedResponse map[string]interface{}
	err = json.Unmarshal(result, &processedResponse)
	require.NoError(t, err)

	// Check error was processed
	errorData := processedResponse["error"].(map[string]interface{})
	assert.Equal(t, "invalid_api_key_error", errorData["type"])
	assert.Equal(t, nil, errorData["param"])
}

func TestProcessResponse_MissingID(t *testing.T) {
	response := map[string]interface{}{
		"model":   "gpt-4",
		"choices": []interface{}{},
	}

	responseBytes, _ := json.Marshal(response)
	result, err := ProcessResponse(responseBytes, "openai", "", "test-model")
	assert.NoError(t, err)

	var processedResponse map[string]interface{}
	err = json.Unmarshal(result, &processedResponse)
	require.NoError(t, err)

	// Should generate ID
	id := processedResponse["id"].(string)
	assert.True(t, strings.HasPrefix(id, "chatcmpl-"))
}

func TestProcessResponse_ToolCalls(t *testing.T) {
	response := map[string]interface{}{
		"model": "gpt-4",
		"choices": []interface{}{
			map[string]interface{}{
				"message": map[string]interface{}{
					"role": "assistant",
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

	responseBytes, _ := json.Marshal(response)
	result, err := ProcessResponse(responseBytes, "openai", "", "test-model")
	assert.NoError(t, err)

	var processedResponse map[string]interface{}
	err = json.Unmarshal(result, &processedResponse)
	require.NoError(t, err)

	// Check tool calls were processed
	choices := processedResponse["choices"].([]interface{})
	choice := choices[0].(map[string]interface{})
	message := choice["message"].(map[string]interface{})
	toolCalls := message["tool_calls"].([]interface{})
	toolCall := toolCalls[0].(map[string]interface{})

	assert.NotNil(t, toolCall["id"])
}

func TestProcessResponse_CompleteUsageDetails(t *testing.T) {
	response := map[string]interface{}{
		"model": "gpt-4",
		"usage": map[string]interface{}{
			"prompt_tokens":     10,
			"completion_tokens": 20,
			// total_tokens missing
		},
		"choices": []interface{}{},
	}

	responseBytes, _ := json.Marshal(response)
	result, err := ProcessResponse(responseBytes, "openai", "", "test-model")
	assert.NoError(t, err)

	var processedResponse map[string]interface{}
	err = json.Unmarshal(result, &processedResponse)
	require.NoError(t, err)

	// Check usage was normalized
	usage := processedResponse["usage"].(map[string]interface{})
	assert.Equal(t, float64(10), usage["prompt_tokens"])
	assert.Equal(t, float64(20), usage["completion_tokens"])
	assert.Equal(t, float64(0), usage["total_tokens"]) // Added default

	// Check token details were added
	promptDetails := usage["prompt_tokens_details"].(map[string]interface{})
	assert.Equal(t, float64(0), promptDetails["cached_tokens"])
	assert.Equal(t, float64(0), promptDetails["audio_tokens"])

	completionDetails := usage["completion_tokens_details"].(map[string]interface{})
	assert.Equal(t, float64(0), completionDetails["reasoning_tokens"])
	assert.Equal(t, float64(0), completionDetails["audio_tokens"])
}

func TestProcessResponse_MessageAnnotationsAndRefusal(t *testing.T) {
	response := map[string]interface{}{
		"model": "gpt-4",
		"choices": []interface{}{
			map[string]interface{}{
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": "Hello",
					// annotations and refusal missing
				},
			},
		},
	}

	responseBytes, _ := json.Marshal(response)
	result, err := ProcessResponse(responseBytes, "openai", "", "test-model")
	assert.NoError(t, err)

	var processedResponse map[string]interface{}
	err = json.Unmarshal(result, &processedResponse)
	require.NoError(t, err)

	// Check message fields were added
	choices := processedResponse["choices"].([]interface{})
	choice := choices[0].(map[string]interface{})
	message := choice["message"].(map[string]interface{})

	annotations := message["annotations"].([]interface{})
	assert.Empty(t, annotations)
	assert.Nil(t, message["refusal"])
	assert.Nil(t, choice["logprobs"])
}

func TestDecompressResponse_NotGzip(t *testing.T) {
	data := []byte("plain text data")
	result, err := decompressResponse(data, "")
	assert.NoError(t, err)
	assert.Equal(t, data, result)
}

func TestDecompressResponse_InvalidGzip(t *testing.T) {
	data := []byte("not gzip data")
	result, err := decompressResponse(data, "gzip")
	assert.Error(t, err)
	assert.Equal(t, data, result) // Returns original on error
}

func TestUnwrapArrayResponse_NotArray(t *testing.T) {
	data := []byte(`{"model": "test"}`)
	result := unwrapArrayResponse(data)
	assert.Equal(t, data, result)
}

func TestUnwrapArrayResponse_MultipleElements(t *testing.T) {
	data := []byte(`[{"model": "test1"}, {"model": "test2"}]`)
	result := unwrapArrayResponse(data)
	assert.Equal(t, data, result) // Don't unwrap multi-element arrays
}

func TestReplaceModelField_EmptyOriginalModel(t *testing.T) {
	responseData := map[string]interface{}{
		"model": "gpt-4",
	}

	replaceModelField(responseData, "openai", "")
	assert.Equal(t, "gpt-4", responseData["model"]) // Unchanged
}

func TestNormalizeUsageField_MissingUsage(t *testing.T) {
	responseData := map[string]interface{}{}

	normalizeUsageField(responseData)

	usage := responseData["usage"].(map[string]interface{})
	assert.NotNil(t, usage)
	assert.Equal(t, 0, usage["prompt_tokens"])
	assert.Equal(t, 0, usage["completion_tokens"])
	assert.Equal(t, 0, usage["total_tokens"])
}

func TestProcessErrorResponse_NoCode(t *testing.T) {
	responseData := map[string]interface{}{
		"error": map[string]interface{}{
			"message": "Something went wrong",
		},
	}

	processErrorResponse(responseData)

	errorData := responseData["error"].(map[string]interface{})
	assert.Equal(t, "api_error", errorData["type"])
	assert.Equal(t, nil, errorData["param"])
}
