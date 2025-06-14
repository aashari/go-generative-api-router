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
	ctx := context.Background()
	ctx = logger.WithComponent(ctx, "response_processor")
	ctx = logger.WithStage(ctx, "response_processing")
	logger.Info(ctx, "Processing response with complete data",
		"vendor", vendor,
		"content_encoding", contentEncoding,
		"original_model", originalModel,
		"response_size", len(responseBody),
		"response_body", string(responseBody),
		"response_body_bytes", responseBody)

	if len(responseBody) == 0 {
		return responseBody, nil
	}

	// 1. Handle gzip decompression
	decompressed, err := decompressResponse(responseBody, contentEncoding)
	if err != nil {
		return nil, err
	}

	// 2. Check if response is an array
	trimmed := bytes.TrimSpace(decompressed)
	if bytes.HasPrefix(trimmed, []byte("[")) {
		// Handle array response
		var arrayResponse []interface{}
		if err := json.Unmarshal(decompressed, &arrayResponse); err != nil {
			ctx = logger.WithComponent(ctx, "response_processor")
			ctx = logger.WithStage(ctx, "array_parsing")
			logger.Error(ctx, "Array response parsing failed", err,
				"response_body_bytes", decompressed,
				"response_size", len(decompressed),
				"vendor", vendor,
				"content_encoding", contentEncoding,
				"original_model", originalModel,
				"response_body", string(decompressed),
				"response_type", "array_parse_error")
			return decompressed, nil // Return original response on parse error
		}

		// Log array response details
		ctx = logger.WithComponent(ctx, "response_processor")
		ctx = logger.WithStage(ctx, "array_handling")
		logger.Info(ctx, "Received array response from vendor",
			"vendor", vendor,
			"array_length", len(arrayResponse),
			"original_model", originalModel,
			"array_response", arrayResponse)

		// Handle different array response scenarios
		if len(arrayResponse) == 0 {
			// Empty array - create error response
			errorResponse := map[string]interface{}{
				"error": map[string]interface{}{
					"message": "Empty response array from vendor",
					"type":    "invalid_response_error",
					"param":   nil,
					"code":    "empty_array",
				},
			}
			modifiedResponseBody, _ := json.Marshal(errorResponse)
			return modifiedResponseBody, nil
		} else if len(arrayResponse) == 1 {
			// Single element array - unwrap it
			if firstElem, ok := arrayResponse[0].(map[string]interface{}); ok {
				decompressed, _ = json.Marshal(firstElem)
			} else {
				// First element is not an object - create error response
				errorResponse := map[string]interface{}{
					"error": map[string]interface{}{
						"message": fmt.Sprintf("Invalid array element type: %T", arrayResponse[0]),
						"type":    "invalid_response_error",
						"param":   nil,
						"code":    "invalid_array_element",
					},
				}
				modifiedResponseBody, _ := json.Marshal(errorResponse)
				return modifiedResponseBody, nil
			}
		} else {
			// Multi-element array - check if it's a batch response or error
			// For now, we'll take the first valid object response
			var validResponse map[string]interface{}
			for _, elem := range arrayResponse {
				if obj, ok := elem.(map[string]interface{}); ok {
					// Check if it's an error object
					if _, hasError := obj["error"]; hasError {
						// Use the first error
						validResponse = obj
						break
					}
					// Check if it's a valid completion response
					if _, hasID := obj["id"]; hasID {
						validResponse = obj
						break
					}
				}
			}

			if validResponse != nil {
				decompressed, _ = json.Marshal(validResponse)
			} else {
				// No valid response found - create error
				errorResponse := map[string]interface{}{
					"error": map[string]interface{}{
						"message": fmt.Sprintf("No valid response found in array of %d elements", len(arrayResponse)),
						"type":    "invalid_response_error",
						"param":   nil,
						"code":    "no_valid_response",
					},
				}
				modifiedResponseBody, _ := json.Marshal(errorResponse)
				return modifiedResponseBody, nil
			}
		}
	}

	// 3. Parse JSON (now handles both objects and processed arrays)
	var responseData map[string]interface{}
	if err := json.Unmarshal(decompressed, &responseData); err != nil {
		// Log complete unmarshaling error
		ctx = logger.WithComponent(ctx, "response_processor")
		ctx = logger.WithStage(ctx, "json_parsing")
		logger.Error(ctx, "Response JSON parsing failed", err,
			"response_body_bytes", decompressed,
			"response_size", len(decompressed),
			"vendor", vendor,
			"content_encoding", contentEncoding,
			"original_model", originalModel,
			"response_body", string(decompressed))
		return decompressed, nil // Return original response on parse error
	}

	// Log complete parsed response data
	ctx = logger.WithComponent(ctx, "response_processor")
	ctx = logger.WithStage(ctx, "parsing_success")
	logger.Debug(ctx, "Response parsed successfully with complete data",
		"vendor", vendor,
		"original_model", originalModel,
		"complete_parsed_response", responseData,
		"response_body", string(decompressed))

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
		ctx = logger.WithComponent(ctx, "response_processor")
		ctx = logger.WithStage(ctx, "marshaling")
		logger.Error(ctx, "Response marshaling failed", err,
			"vendor", vendor,
			"original_model", originalModel,
			"complete_response_data", responseData,
			"original_response_body", string(decompressed))
		return decompressed, fmt.Errorf("error marshaling modified response: %w", err)
	}

	// Log complete processing completion
	ctx = logger.WithComponent(ctx, "response_processor")
	ctx = logger.WithStage(ctx, "completion")
	logger.Info(ctx, "Response processing completed with complete data",
		"vendor", vendor,
		"original_model", originalModel,
		"original_size", len(decompressed),
		"modified_size", len(modifiedResponseBody),
		"complete_response_data", responseData,
		"original_response", string(decompressed),
		"modified_response", string(modifiedResponseBody))

	return modifiedResponseBody, nil
}

// decompressResponse handles gzip content encoding
func decompressResponse(responseBody []byte, contentEncoding string) ([]byte, error) {
	if contentEncoding != "gzip" {
		return responseBody, nil
	}

	// Log complete decompression start
	ctx := context.Background()
	ctx = logger.WithComponent(ctx, "response_processor")
	ctx = logger.WithStage(ctx, "decompression")
	logger.Info(ctx, "Response is gzip encoded, decompressing with complete data",
		"content_encoding", contentEncoding,
		"compressed_size", len(responseBody),
		"compressed_body", responseBody)

	// Check if the response is actually gzip compressed by looking at the magic bytes
	// Gzip files start with bytes 0x1f 0x8b
	if len(responseBody) < 2 || responseBody[0] != 0x1f || responseBody[1] != 0x8b {
		// The response claims to be gzip but isn't actually compressed
		// This can happen with some vendors that set the header incorrectly
		logger.Warn(ctx, "Response claims gzip encoding but is not actually compressed",
			"content_encoding", contentEncoding,
			"first_bytes", responseBody[:min(10, len(responseBody))],
			"response_size", len(responseBody))
		return responseBody, nil
	}

	gzipReader, err := gzip.NewReader(bytes.NewReader(responseBody))
	if err != nil {
		// Log complete gzip reader error
		logger.Error(ctx, "Gzip reader creation failed", err,
			"content_encoding", contentEncoding,
			"compressed_body", responseBody,
			"compressed_size", len(responseBody))
		// Fall back to returning the original response body
		// Some vendors might incorrectly set Content-Encoding header
		logger.Warn(ctx, "Falling back to uncompressed response due to gzip reader error",
			"content_encoding", contentEncoding,
			"error", err.Error())
		return responseBody, nil
	}
	defer gzipReader.Close()

	decompressedBody, err := io.ReadAll(gzipReader)
	if err != nil {
		// Log complete decompression error
		logger.Error(ctx, "Gzip decompression failed", err,
			"content_encoding", contentEncoding,
			"compressed_body", responseBody,
			"compressed_size", len(responseBody))
		// Fall back to returning the original response body
		logger.Warn(ctx, "Falling back to uncompressed response due to decompression error",
			"content_encoding", contentEncoding,
			"error", err.Error())
		return responseBody, nil
	}

	// Log complete decompression success
	logger.Info(ctx, "Successfully decompressed gzip response with complete data",
		"content_encoding", contentEncoding,
		"compressed_size", len(responseBody),
		"decompressed_size", len(decompressedBody),
		"compressed_body", responseBody,
		"decompressed_body", decompressedBody)
	return decompressedBody, nil
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
		ctx := context.Background()
		ctx = logger.WithComponent(ctx, "response_processor")
		ctx = logger.WithStage(ctx, "fingerprint_generation")
		logger.Info(ctx, "Generated system_fingerprint with complete data",
			"reason", "missing_or_null",
			"generated_value", generatedFP,
			"complete_response_data", responseData,
			"original_value", systemFingerprintValue,
			"value_existed", systemFingerprintExists)
	} else if _, isString := systemFingerprintValue.(string); !isString {
		// If it exists but is not a string
		generatedFP := SystemFingerprint()
		responseData["system_fingerprint"] = generatedFP
		// Log complete system fingerprint replacement
		ctx := context.Background()
		ctx = logger.WithComponent(ctx, "response_processor")
		ctx = logger.WithStage(ctx, "fingerprint_replacement")
		logger.Info(ctx, "Replaced non-string system_fingerprint with complete data",
			"generated_value", generatedFP,
			"original_value", systemFingerprintValue,
			"original_type", fmt.Sprintf("%T", systemFingerprintValue),
			"complete_response_data", responseData)
	}
}

// replaceModelField replaces the model field with the original requested model
func replaceModelField(responseData map[string]interface{}, vendor string, originalModel string) {
	if model, ok := responseData["model"].(string); ok {
		// Log complete model field processing
		ctx := context.Background()
		ctx = logger.WithComponent(ctx, "response_processor")
		ctx = logger.WithStage(ctx, "model_replacement")
		logger.Info(ctx, "Processing response from actual model with complete data",
			"actual_model", model,
			"vendor", vendor,
			"presented_as", originalModel,
			"complete_response_data", responseData)
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
			ctx := context.Background()
			ctx = logger.WithComponent(ctx, "response_processor")
			ctx = logger.WithStage(ctx, "choices_normalization")
			logger.Info(ctx, "Adding empty choices array for zero completion tokens response",
				"vendor", vendor,
				"has_zero_completion_tokens", hasZeroCompletionTokens,
				"complete_response_data", responseData,
				"reason", "missing_choices_with_zero_tokens")

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
			ctx := context.Background()
			ctx = logger.WithComponent(ctx, "response_processor")
			ctx = logger.WithStage(ctx, "choices_validation")
			logger.Warn(ctx, "Missing choices field in non-zero completion tokens response",
				"vendor", vendor,
				"has_zero_completion_tokens", hasZeroCompletionTokens,
				"complete_response_data", responseData,
				"reason", "missing_choices_with_tokens")
		}
	}
}

// processChoices processes the choices array in the response
func processChoices(choices []interface{}, vendor string) {
	// Log complete choices processing start
	ctx := context.Background()
	ctx = logger.WithComponent(ctx, "response_processor")
	ctx = logger.WithStage(ctx, "choices_processing")
	logger.Info(ctx, "Processing choices with complete data",
		"choices_count", len(choices),
		"complete_choices", choices,
		"vendor", vendor)

	for i, choice := range choices {
		choiceMap, ok := choice.(map[string]interface{})
		if !ok {
			// Log complete non-map choice data
			logger.Warn(ctx, "Choice is not a map with complete data",
				"choice_index", i,
				"complete_choice", choice,
				"choice_type", fmt.Sprintf("%T", choice),
				"all_choices", choices,
				"vendor", vendor)
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
	logger.Debug(ctx, "Choices processing completed with complete data",
		"processed_choices", choices,
		"choices_count", len(choices),
		"vendor", vendor)
}

// processMessage processes a message within a choice
func processMessage(message map[string]interface{}, vendor string) {
	// Log complete message processing start
	ctx := context.Background()
	ctx = logger.WithComponent(ctx, "response_processor")
	ctx = logger.WithStage(ctx, "message_processing")
	logger.Debug(ctx, "Processing message with complete data",
		"complete_message", message,
		"vendor", vendor)

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
		ctx = logger.WithStage(ctx, "tool_calls_processing")
		logger.Info(ctx, "Processing tool calls in message with complete data",
			"tool_calls_count", len(toolCalls),
			"complete_tool_calls", toolCalls,
			"complete_message", message,
			"vendor", vendor)
		processedToolCalls := ProcessToolCalls(toolCalls, vendor)
		message["tool_calls"] = processedToolCalls
	} else {
		// Log complete no tool calls data
		logger.Debug(ctx, "No tool calls found in message with complete data",
			"complete_message", message,
			"vendor", vendor)
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
