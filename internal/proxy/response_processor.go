package proxy

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"

	"github.com/aashari/go-generative-api-router/internal/logger"
)

// ProcessResponse processes the API response, ensuring all required fields are present
func ProcessResponse(responseBody []byte, vendor string, contentEncoding string, originalModel string) ([]byte, error) {
	if len(responseBody) == 0 {
		return responseBody, nil
	}

	// 1. Handle gzip decompression
	decompressed, err := decompressResponse(responseBody, contentEncoding)
	if err != nil {
		return responseBody, err
	}

	// 2. Unwrap array responses (Gemini errors)
	unwrapped := unwrapArrayResponse(decompressed)

	// 3. Parse JSON
	var responseData map[string]interface{}
	if err := json.Unmarshal(unwrapped, &responseData); err != nil {
		logger.Error("Error unmarshaling response", "error", err)
		return unwrapped, nil // Return original response if it's not valid JSON
	}

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
		logger.Error("Error marshaling modified response", "error", err)
		return unwrapped, nil // Return original response if marshal fails
	}

	return modifiedResponseBody, nil
}

// decompressResponse handles gzip content encoding
func decompressResponse(responseBody []byte, contentEncoding string) ([]byte, error) {
	if contentEncoding != "gzip" {
		return responseBody, nil
	}

	logger.Info("Response is gzip encoded, decompressing...")
	gzipReader, err := gzip.NewReader(bytes.NewReader(responseBody))
	if err != nil {
		logger.Error("Error creating gzip reader", "error", err)
		return responseBody, fmt.Errorf("error creating gzip reader: %w", err)
	}
	defer gzipReader.Close()

	decompressedBody, err := io.ReadAll(gzipReader)
	if err != nil {
		logger.Error("Error decompressing gzip response", "error", err)
		return responseBody, fmt.Errorf("error decompressing gzip response: %w", err)
	}

	logger.Info("Successfully decompressed gzip response", 
		"original_size_bytes", len(responseBody), 
		"decompressed_size_bytes", len(decompressedBody))
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
		logger.Info("Generated system_fingerprint", "reason", "missing_or_null", "value", generatedFP)
	} else if _, isString := systemFingerprintValue.(string); !isString {
		// If it exists but is not a string
		generatedFP := SystemFingerprint()
		responseData["system_fingerprint"] = generatedFP
		logger.Info("Replaced non-string system_fingerprint", "value", generatedFP)
	}
}

// replaceModelField replaces the model field with the original requested model
func replaceModelField(responseData map[string]interface{}, vendor string, originalModel string) {
	if model, ok := responseData["model"].(string); ok {
		logger.LogVendorResponse(nil, vendor, model, originalModel, 0, 0)
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
	if choices, ok := responseData["choices"].([]interface{}); ok && len(choices) > 0 {
		processChoices(choices, vendor)
		responseData["choices"] = choices
	}
}

// processChoices processes the choices array in the response
func processChoices(choices []interface{}, vendor string) {
	logger.Info("Processing choices", "count", len(choices))
	for i, choice := range choices {
		choiceMap, ok := choice.(map[string]interface{})
		if !ok {
			logger.Warn("Choice is not a map", "index", i)
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
}

// processMessage processes a message within a choice
func processMessage(message map[string]interface{}, vendor string) {
	logger.Debug("Processing message")

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
		logger.Info("Processing tool calls in message", "count", len(toolCalls))
		processedToolCalls := ProcessToolCalls(toolCalls, vendor)
		message["tool_calls"] = processedToolCalls
	} else {
		logger.Debug("No tool calls found in message")
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
