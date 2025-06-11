package proxy

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/aashari/go-generative-api-router/internal/logger"
)

// ProcessResponse processes the API response, ensuring all required fields are present
func ProcessResponse(responseBody []byte, vendor string, contentEncoding string, originalModel string) ([]byte, error) {
	// Log complete response processing start
	logger.LogWithStructure(context.Background(), logger.LevelInfo, "Processing response with complete data",
		map[string]interface{}{
			"vendor":           vendor,
			"content_encoding": contentEncoding,
			"original_model":   originalModel,
			"response_size":    len(responseBody),
		},
		nil, // request
		map[string]interface{}{
			"response_body":       string(responseBody),
			"response_body_bytes": responseBody,
		},
		nil) // error

	if len(responseBody) == 0 {
		return responseBody, nil
	}

	// 1. Handle gzip decompression
	decompressed, err := decompressResponse(responseBody, contentEncoding)
	if err != nil {
		return nil, err
	}

	// 2. Unwrap array responses (Gemini errors)
	unwrapped := unwrapArrayResponse(decompressed)

	// 3. Parse JSON
	var responseData map[string]interface{}
	if err := json.Unmarshal(unwrapped, &responseData); err != nil {
		// Log complete unmarshaling error
		logger.LogError(context.Background(), "response_processor", err, map[string]any{
			"response_body_bytes": unwrapped,
			"response_size":       len(unwrapped),
			"vendor":              vendor,
			"content_encoding":    contentEncoding,
			"original_model":      originalModel,
			"response_body":       string(unwrapped),
		})
		return unwrapped, nil // Return original response on parse error
	}

	// Log complete parsed response data
	logger.LogWithStructure(context.Background(), logger.LevelDebug, "Response parsed successfully with complete data",
		map[string]interface{}{
			"vendor":                   vendor,
			"original_model":           originalModel,
			"complete_parsed_response": responseData,
		},
		nil, // request
		map[string]interface{}{
			"response_body": string(unwrapped),
		},
		nil) // error

	// 4. Generate missing IDs and add compatibility fields
	addMissingIDs(responseData)
	addOpenAICompatibilityFields(responseData)

	// 5. Replace model field with original model
	replaceModelField(responseData, vendor, originalModel)

	// 6. Process error responses or normal responses
	if isErrorResponse(responseData) {
		processErrorResponse(responseData)
	} else {
		processNormalResponse(responseData, vendor)
	}

	// 7. Normalize usage field
	normalizeUsageField(responseData)

	// 8. Marshal back to JSON
	modifiedResponseBody, err := json.Marshal(responseData)
	if err != nil {
		// Log complete marshaling error
		logger.LogError(context.Background(), "response_processor", err, map[string]any{
			"vendor":                 vendor,
			"original_model":         originalModel,
			"complete_response_data": responseData,
			"original_response_body": string(unwrapped),
		})
		return unwrapped, fmt.Errorf("error marshaling modified response: %w", err)
	}

	// Log complete processing completion
	logger.LogWithStructure(context.Background(), logger.LevelInfo, "Response processing completed with complete data",
		map[string]interface{}{
			"vendor":                 vendor,
			"original_model":         originalModel,
			"original_size":          len(unwrapped),
			"modified_size":          len(modifiedResponseBody),
			"complete_response_data": responseData,
		},
		nil, // request
		map[string]interface{}{
			"original_response": string(unwrapped),
			"modified_response": string(modifiedResponseBody),
		},
		nil) // error

	return modifiedResponseBody, nil
}

// decompressResponse handles gzip content encoding
func decompressResponse(responseBody []byte, contentEncoding string) ([]byte, error) {
	if contentEncoding != "gzip" {
		return responseBody, nil
	}

	// Log complete decompression start
	logger.LogWithStructure(context.Background(), logger.LevelInfo, "Response is gzip encoded, decompressing with complete data",
		map[string]interface{}{
			"content_encoding": contentEncoding,
			"compressed_size":  len(responseBody),
		},
		nil, // request
		map[string]interface{}{
			"compressed_body": responseBody,
		},
		nil) // error

	gzipReader, err := gzip.NewReader(bytes.NewReader(responseBody))
	if err != nil {
		// Log complete gzip reader error
		logger.LogError(context.Background(), "response_processor", err, map[string]any{
			"content_encoding": contentEncoding,
			"compressed_body":  responseBody,
			"compressed_size":  len(responseBody),
		})
		return responseBody, fmt.Errorf("error creating gzip reader: %w", err)
	}
	defer gzipReader.Close()

	decompressedBody, err := io.ReadAll(gzipReader)
	if err != nil {
		// Log complete decompression error
		logger.LogError(context.Background(), "response_processor", err, map[string]any{
			"content_encoding": contentEncoding,
			"compressed_body":  responseBody,
			"compressed_size":  len(responseBody),
		})
		return responseBody, fmt.Errorf("error decompressing gzip response: %w", err)
	}

	// Log complete decompression success
	logger.LogWithStructure(context.Background(), logger.LevelInfo, "Successfully decompressed gzip response with complete data",
		map[string]interface{}{
			"content_encoding":  contentEncoding,
			"compressed_size":   len(responseBody),
			"decompressed_size": len(decompressedBody),
		},
		nil, // request
		map[string]interface{}{
			"compressed_body":   responseBody,
			"decompressed_body": decompressedBody,
		},
		nil) // error
	return decompressedBody, nil
}

// unwrapArrayResponse handles single-element arrays (Gemini errors)
func unwrapArrayResponse(responseBody []byte) []byte {
	if !bytes.HasPrefix(bytes.TrimSpace(responseBody), []byte("[")) {
		return responseBody
	}

	var arrayData []interface{}
	if err := json.Unmarshal(responseBody, &arrayData); err == nil {
		// If it's a single element array, unwrap it to be consistent with OpenAI
		if len(arrayData) == 1 {
			// Convert the single element back to JSON
			unwrappedData, err := json.Marshal(arrayData[0])
			if err == nil {
				return unwrappedData
			}
		}
	}
	return responseBody
}

// addMissingIDs generates missing chat completion IDs
func addMissingIDs(responseData map[string]interface{}) {
	if id, ok := responseData["id"]; !ok || id == nil || id == "" {
		responseData["id"] = ChatCompletionID()
	}
}

// addOpenAICompatibilityFields adds required OpenAI compatibility fields
func addOpenAICompatibilityFields(responseData map[string]interface{}) {
	// Add service_tier if missing
	if _, ok := responseData["service_tier"]; !ok {
		responseData["service_tier"] = "default"
	}

	// Add system_fingerprint if missing or invalid
	systemFingerprintValue, systemFingerprintExists := responseData["system_fingerprint"]
	if !systemFingerprintExists || systemFingerprintValue == nil {
		generatedFP := SystemFingerprint()
		responseData["system_fingerprint"] = generatedFP
		// Log complete system fingerprint generation
		logger.LogWithStructure(context.Background(), logger.LevelInfo, "Generated system_fingerprint with complete data",
			map[string]interface{}{
				"reason":                 "missing_or_null",
				"generated_value":        generatedFP,
				"complete_response_data": responseData,
				"original_value":         systemFingerprintValue,
				"value_existed":          systemFingerprintExists,
			},
			nil, // request
			nil, // response
			nil) // error
	} else if _, isString := systemFingerprintValue.(string); !isString {
		// If it exists but is not a string
		generatedFP := SystemFingerprint()
		responseData["system_fingerprint"] = generatedFP
		// Log complete system fingerprint replacement
		logger.LogWithStructure(context.Background(), logger.LevelInfo, "Replaced non-string system_fingerprint with complete data",
			map[string]interface{}{
				"generated_value":        generatedFP,
				"original_value":         systemFingerprintValue,
				"original_type":          fmt.Sprintf("%T", systemFingerprintValue),
				"complete_response_data": responseData,
			},
			nil, // request
			nil, // response
			nil) // error
	}
}

// replaceModelField replaces the model field with the original requested model
func replaceModelField(responseData map[string]interface{}, vendor string, originalModel string) {
	if model, ok := responseData["model"].(string); ok {
		// Log complete model field processing
		logger.LogWithStructure(context.Background(), logger.LevelInfo, "Processing response from actual model with complete data",
			map[string]interface{}{
				"actual_model":           model,
				"vendor":                 vendor,
				"presented_as":           originalModel,
				"complete_response_data": responseData,
			},
			nil, // request
			nil, // response
			nil) // error
	}

	if originalModel != "" {
		responseData["model"] = originalModel
	}
}

// isErrorResponse checks if the response is an error response
func isErrorResponse(responseData map[string]interface{}) bool {
	_, ok := responseData["error"].(map[string]interface{})
	return ok
}

// processErrorResponse handles error response processing
func processErrorResponse(responseData map[string]interface{}) {
	errorData := responseData["error"].(map[string]interface{})

	if code, ok := errorData["code"]; ok {
		// Convert the code to a string type if needed
		errorType := fmt.Sprintf("%v", code)
		errorData["type"] = errorType + "_error"
	} else {
		errorData["type"] = "api_error"
	}

	if _, ok := errorData["param"]; !ok {
		errorData["param"] = nil
	}

	responseData["error"] = errorData
}

// processNormalResponse handles normal (non-error) response processing
func processNormalResponse(responseData map[string]interface{}, vendor string) {
	// Check if choices field exists
	if choices, ok := responseData["choices"].([]interface{}); ok && len(choices) > 0 {
		processChoices(choices, vendor)
		responseData["choices"] = choices
	} else {
		// Check if this is a response with zero completion tokens
		hasZeroCompletionTokens := false
		if usage, ok := responseData["usage"].(map[string]interface{}); ok {
			if completionTokens, ok := usage["completion_tokens"]; ok {
				if tokens, ok := completionTokens.(float64); ok && tokens == 0 {
					hasZeroCompletionTokens = true
				}
			}
		}

		// If choices field is missing and we have zero completion tokens, add an empty choices array
		if hasZeroCompletionTokens && !ok {
			// Log complete empty choices array addition
			logger.LogWithStructure(context.Background(), logger.LevelInfo, "Adding empty choices array for zero completion tokens response",
				map[string]interface{}{
					"vendor":                     vendor,
					"has_zero_completion_tokens": hasZeroCompletionTokens,
					"complete_response_data":     responseData,
					"reason":                     "missing_choices_with_zero_tokens",
				},
				nil, // request
				nil, // response
				nil) // error

			// Add empty choices array with a single choice indicating no content was generated
			responseData["choices"] = []interface{}{
				map[string]interface{}{
					"index": 0,
					"message": map[string]interface{}{
						"role":        "assistant",
						"content":     "",
						"annotations": []interface{}{},
						"refusal":     nil,
					},
					"logprobs":      nil,
					"finish_reason": "stop",
				},
			}
		} else if !ok {
			// Log complete missing choices warning for non-zero token responses
			logger.LogWithStructure(context.Background(), logger.LevelWarn, "Missing choices field in non-zero completion tokens response",
				map[string]interface{}{
					"vendor":                     vendor,
					"has_zero_completion_tokens": hasZeroCompletionTokens,
					"complete_response_data":     responseData,
					"reason":                     "missing_choices_with_tokens",
				},
				nil, // request
				nil, // response
				nil) // error
		}
	}
}

// processChoices processes the choices array in the response
func processChoices(choices []interface{}, vendor string) {
	// Log complete choices processing start
	logger.LogWithStructure(context.Background(), logger.LevelInfo, "Processing choices with complete data",
		map[string]interface{}{
			"choices_count":    len(choices),
			"complete_choices": choices,
			"vendor":           vendor,
		},
		nil, // request
		nil, // response
		nil) // error

	for i, choice := range choices {
		choiceMap, ok := choice.(map[string]interface{})
		if !ok {
			// Log complete non-map choice data
			logger.LogWithStructure(context.Background(), logger.LevelWarn, "Choice is not a map with complete data",
				map[string]interface{}{
					"choice_index":    i,
					"complete_choice": choice,
					"choice_type":     fmt.Sprintf("%T", choice),
					"all_choices":     choices,
					"vendor":          vendor,
				},
				nil, // request
				nil, // response
				nil) // error
			continue
		}

		// Add logprobs if missing
		if _, ok := choiceMap["logprobs"]; !ok {
			choiceMap["logprobs"] = nil
		}

		// Process message if present
		if message, ok := choiceMap["message"].(map[string]interface{}); ok {
			processMessage(message, vendor)
			choiceMap["message"] = message
		}

		choices[i] = choiceMap
	}

	// Log complete choices processing completion
	logger.LogWithStructure(context.Background(), logger.LevelDebug, "Choices processing completed with complete data",
		map[string]interface{}{
			"processed_choices": choices,
			"choices_count":     len(choices),
			"vendor":            vendor,
		},
		nil, // request
		nil, // response
		nil) // error
}

// processMessage processes a message within a choice
func processMessage(message map[string]interface{}, vendor string) {
	// Log complete message processing start
	logger.LogWithStructure(context.Background(), logger.LevelDebug, "Processing message with complete data",
		map[string]interface{}{
			"complete_message": message,
			"vendor":           vendor,
		},
		nil, // request
		nil, // response
		nil) // error

	// Add annotations array if missing
	if _, ok := message["annotations"]; !ok {
		message["annotations"] = []interface{}{}
	}

	// Add refusal if missing
	if _, ok := message["refusal"]; !ok {
		message["refusal"] = nil
	}

	// Handle tool_calls if present
	if toolCalls, ok := message["tool_calls"].([]interface{}); ok && len(toolCalls) > 0 {
		// Log complete tool calls processing in message
		logger.LogWithStructure(context.Background(), logger.LevelInfo, "Processing tool calls in message with complete data",
			map[string]interface{}{
				"tool_calls_count":    len(toolCalls),
				"complete_tool_calls": toolCalls,
				"complete_message":    message,
				"vendor":              vendor,
			},
			nil, // request
			nil, // response
			nil) // error
		processedToolCalls := ProcessToolCalls(toolCalls, vendor)
		message["tool_calls"] = processedToolCalls
	} else {
		// Log complete no tool calls data
		logger.LogWithStructure(context.Background(), logger.LevelDebug, "No tool calls found in message with complete data",
			map[string]interface{}{
				"complete_message": message,
				"vendor":           vendor,
			},
			nil, // request
			nil, // response
			nil) // error
	}
}

// normalizeUsageField ensures usage field is present with all required subfields
func normalizeUsageField(responseData map[string]interface{}) {
	if usage, ok := responseData["usage"].(map[string]interface{}); ok {
		// Make sure all required usage fields are present
		if _, ok := usage["prompt_tokens"]; !ok {
			usage["prompt_tokens"] = 0
		}
		if _, ok := usage["completion_tokens"]; !ok {
			usage["completion_tokens"] = 0
		}
		if _, ok := usage["total_tokens"]; !ok {
			usage["total_tokens"] = 0
		}

		// Add token details subfields if not present (OpenAI compatibility)
		if _, ok := usage["prompt_tokens_details"]; !ok {
			usage["prompt_tokens_details"] = map[string]interface{}{
				"cached_tokens": 0,
				"audio_tokens":  0,
			}
		}

		if _, ok := usage["completion_tokens_details"]; !ok {
			usage["completion_tokens_details"] = map[string]interface{}{
				"reasoning_tokens":           0,
				"audio_tokens":               0,
				"accepted_prediction_tokens": 0,
				"rejected_prediction_tokens": 0,
			}
		}

		responseData["usage"] = usage
	} else {
		// If usage is completely missing, add a placeholder with default values
		responseData["usage"] = map[string]interface{}{
			"prompt_tokens":     0,
			"completion_tokens": 0,
			"total_tokens":      0,
			"prompt_tokens_details": map[string]interface{}{
				"cached_tokens": 0,
				"audio_tokens":  0,
			},
			"completion_tokens_details": map[string]interface{}{
				"reasoning_tokens":           0,
				"audio_tokens":               0,
				"accepted_prediction_tokens": 0,
				"rejected_prediction_tokens": 0,
			},
		}
	}
}
