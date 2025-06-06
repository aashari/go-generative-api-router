package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aashari/go-generative-api-router/internal/logger"
)

// StreamProcessor handles stateful processing of streaming responses
type StreamProcessor struct {
	ConversationID    string
	Timestamp         int64
	SystemFingerprint string
	Vendor            string
	OriginalModel     string
	isFirstChunk      bool
}

// NewStreamProcessor creates a new stream processor with conversation-level values
func NewStreamProcessor(conversationID string, timestamp int64, systemFingerprint string, vendor string, originalModel string) *StreamProcessor {
	return &StreamProcessor{
		ConversationID:    conversationID,
		Timestamp:         timestamp,
		SystemFingerprint: systemFingerprint,
		Vendor:            vendor,
		OriginalModel:     originalModel,
		isFirstChunk:      true,
	}
}

// ProcessChunk processes a single streaming chunk
func (sp *StreamProcessor) ProcessChunk(chunk []byte) []byte {
	// Skip empty chunks
	if len(chunk) == 0 {
		return chunk
	}

	// Log complete chunk processing start
	logger.LogMultipleData(context.Background(), logger.LevelDebug, "Processing streaming chunk with complete data", map[string]any{
		"chunk":              string(chunk),
		"chunk_bytes":        chunk,
		"chunk_size":         len(chunk),
		"vendor":             sp.Vendor,
		"conversation_id":    sp.ConversationID,
		"timestamp":          sp.Timestamp,
		"system_fingerprint": sp.SystemFingerprint,
		"original_model":     sp.OriginalModel,
	})

	// Handle SSE format - look for "data: " prefix
	chunkStr := string(chunk)
	if !strings.HasPrefix(chunkStr, "data: ") {
		return chunk // Return as-is if not SSE format
	}

	// Extract JSON data after "data: "
	jsonData := strings.TrimPrefix(chunkStr, "data: ")
	jsonData = strings.TrimSpace(jsonData)

	// Skip [DONE] messages
	if jsonData == "[DONE]" {
		return chunk
	}

	// Parse the JSON chunk
	var chunkData map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &chunkData); err != nil {
		// Log complete unmarshaling error
		logger.LogError(context.Background(), "stream_processor", err, map[string]any{
			"vendor":          sp.Vendor,
			"chunk":           string(chunk),
			"json_data":       jsonData,
			"conversation_id": sp.ConversationID,
			"original_model":  sp.OriginalModel,
		})
		return chunk // Return original chunk if parsing fails
	}

	// Log complete parsed chunk data
	logger.LogMultipleData(context.Background(), logger.LevelDebug, "Stream chunk parsed successfully with complete data", map[string]any{
		"vendor":              sp.Vendor,
		"complete_chunk_data": chunkData,
		"original_chunk":      string(chunk),
		"json_data":           jsonData,
		"conversation_id":     sp.ConversationID,
		"original_model":      sp.OriginalModel,
	})

	// Process the chunk data
	sp.processChunkData(chunkData)

	// Convert back to JSON
	modifiedJSON, err := json.Marshal(chunkData)
	if err != nil {
		// Log complete marshaling error
		logger.LogError(context.Background(), "stream_processor", err, map[string]any{
			"vendor":              sp.Vendor,
			"complete_chunk_data": chunkData,
			"original_chunk":      string(chunk),
			"conversation_id":     sp.ConversationID,
			"original_model":      sp.OriginalModel,
		})
		return chunk // Return original chunk if marshaling fails
	}

	// Log complete chunk processing completion
	logger.LogMultipleData(context.Background(), logger.LevelDebug, "Stream chunk processing completed with complete data", map[string]any{
		"vendor":              sp.Vendor,
		"original_chunk":      string(chunk),
		"modified_chunk":      string(modifiedJSON),
		"complete_chunk_data": chunkData,
		"conversation_id":     sp.ConversationID,
		"original_model":      sp.OriginalModel,
	})

	// Return the modified chunk in SSE format
	result := []byte("data: " + string(modifiedJSON))
	// Add double newline for SSE format
	result = append(result, '\n', '\n')
	return result
}

// processChunkData processes the parsed chunk data
func (sp *StreamProcessor) processChunkData(chunkData map[string]interface{}) {
	// Set consistent values
	chunkData["id"] = sp.ConversationID
	chunkData["created"] = sp.Timestamp
	chunkData["system_fingerprint"] = sp.SystemFingerprint
	chunkData["model"] = sp.OriginalModel

	// Add service_tier if missing (OpenAI compatibility)
	if _, ok := chunkData["service_tier"]; !ok {
		chunkData["service_tier"] = "default"
	}

	// Process choices if present
	if choices, ok := chunkData["choices"].([]interface{}); ok && len(choices) > 0 {
		// Log complete choices processing in stream chunk
		logger.LogMultipleData(context.Background(), logger.LevelDebug, "Processing choices in stream chunk with complete data", map[string]any{
			"choices_count":       len(choices),
			"complete_choices":    choices,
			"vendor":              sp.Vendor,
			"complete_chunk_data": chunkData,
			"conversation_id":     sp.ConversationID,
			"original_model":      sp.OriginalModel,
		})
		sp.processStreamChoices(choices)

		// Check if this is the first chunk and add usage if needed
		sp.addUsageForFirstChunk(chunkData)

		// Mark that we've processed the first chunk
		if sp.isFirstChunk {
			sp.isFirstChunk = false
		}
	} else {
		// Log complete no choices data
		logger.LogMultipleData(context.Background(), logger.LevelDebug, "No choices found in stream chunk with complete data", map[string]any{
			"vendor":              sp.Vendor,
			"complete_chunk_data": chunkData,
			"conversation_id":     sp.ConversationID,
			"original_model":      sp.OriginalModel,
		})
	}
}

// processStreamChoices processes choices in streaming chunks
func (sp *StreamProcessor) processStreamChoices(choices []interface{}) {
	for i, choice := range choices {
		choiceMap, ok := choice.(map[string]interface{})
		if !ok {
			// Log complete non-map choice data in stream
			logger.LogMultipleData(context.Background(), logger.LevelWarn, "Stream chunk choice is not a map with complete data", map[string]any{
				"choice_index":    i,
				"complete_choice": choice,
				"choice_type":     fmt.Sprintf("%T", choice),
				"vendor":          sp.Vendor,
				"all_choices":     choices,
				"conversation_id": sp.ConversationID,
				"original_model":  sp.OriginalModel,
			})
			continue
		}

		// Add logprobs if missing
		if _, ok := choiceMap["logprobs"]; !ok {
			choiceMap["logprobs"] = nil
		}

		// Process delta or message
		if delta, ok := choiceMap["delta"].(map[string]interface{}); ok {
			sp.processStreamDelta(delta, i)
		} else if message, ok := choiceMap["message"].(map[string]interface{}); ok {
			sp.processStreamMessage(message, i)
		} else {
			// Log complete no delta or message data
			logger.LogMultipleData(context.Background(), logger.LevelWarn, "No delta or message found in stream chunk choice with complete data", map[string]any{
				"choice_index":        i,
				"complete_choice_map": choiceMap,
				"vendor":              sp.Vendor,
				"conversation_id":     sp.ConversationID,
				"original_model":      sp.OriginalModel,
			})
		}

		choices[i] = choiceMap
	}
}

// processStreamDelta processes delta in streaming chunks
func (sp *StreamProcessor) processStreamDelta(delta map[string]interface{}, choiceIndex int) {
	// Log complete delta processing start
	logger.LogMultipleData(context.Background(), logger.LevelDebug, "Processing delta in stream chunk with complete data", map[string]any{
		"vendor":          sp.Vendor,
		"complete_delta":  delta,
		"choice_index":    choiceIndex,
		"conversation_id": sp.ConversationID,
		"original_model":  sp.OriginalModel,
	})

	// Add annotations if missing
	if _, ok := delta["annotations"]; !ok {
		delta["annotations"] = []interface{}{}
	}

	// Add refusal if missing
	if _, ok := delta["refusal"]; !ok {
		delta["refusal"] = nil
	}

	// Handle tool_calls if present
	if toolCalls, ok := delta["tool_calls"].([]interface{}); ok && len(toolCalls) > 0 {
		// Log complete tool calls processing in stream chunk delta
		logger.LogMultipleData(context.Background(), logger.LevelDebug, "Processing tool calls in stream chunk delta with complete data", map[string]any{
			"tool_calls_count":    len(toolCalls),
			"complete_tool_calls": toolCalls,
			"vendor":              sp.Vendor,
			"complete_delta":      delta,
			"choice_index":        choiceIndex,
			"conversation_id":     sp.ConversationID,
			"original_model":      sp.OriginalModel,
		})
		processedToolCalls := ProcessToolCalls(toolCalls, sp.Vendor)
		delta["tool_calls"] = processedToolCalls
	} else {
		// Log complete no tool calls data in delta
		logger.LogMultipleData(context.Background(), logger.LevelDebug, "No tool calls found in stream chunk delta with complete data", map[string]any{
			"vendor":          sp.Vendor,
			"complete_delta":  delta,
			"choice_index":    choiceIndex,
			"conversation_id": sp.ConversationID,
			"original_model":  sp.OriginalModel,
		})
	}
}

// processStreamMessage processes message in streaming chunks
func (sp *StreamProcessor) processStreamMessage(message map[string]interface{}, choiceIndex int) {
	// Log complete message processing start in stream
	logger.LogMultipleData(context.Background(), logger.LevelDebug, "Processing message in stream chunk with complete data", map[string]any{
		"vendor":           sp.Vendor,
		"complete_message": message,
		"choice_index":     choiceIndex,
		"conversation_id":  sp.ConversationID,
		"original_model":   sp.OriginalModel,
	})

	// Add annotations if missing
	if _, ok := message["annotations"]; !ok {
		message["annotations"] = []interface{}{}
	}

	// Add refusal if missing
	if _, ok := message["refusal"]; !ok {
		message["refusal"] = nil
	}

	// Handle tool_calls if present
	if toolCalls, ok := message["tool_calls"].([]interface{}); ok && len(toolCalls) > 0 {
		// Log complete tool calls processing in stream chunk message
		logger.LogMultipleData(context.Background(), logger.LevelDebug, "Processing tool calls in stream chunk message with complete data", map[string]any{
			"tool_calls_count":    len(toolCalls),
			"complete_tool_calls": toolCalls,
			"vendor":              sp.Vendor,
			"complete_message":    message,
			"choice_index":        choiceIndex,
			"conversation_id":     sp.ConversationID,
			"original_model":      sp.OriginalModel,
		})
		processedToolCalls := ProcessToolCalls(toolCalls, sp.Vendor)
		message["tool_calls"] = processedToolCalls
	} else {
		// Log complete no tool calls data in message
		logger.LogMultipleData(context.Background(), logger.LevelDebug, "No tool calls found in stream chunk message with complete data", map[string]any{
			"vendor":           sp.Vendor,
			"complete_message": message,
			"choice_index":     choiceIndex,
			"conversation_id":  sp.ConversationID,
			"original_model":   sp.OriginalModel,
		})
	}
}

// isValidStreamChunk validates the SSE format
func (sp *StreamProcessor) isValidStreamChunk(chunk []byte) bool {
	// Handle empty chunks or non-data chunks
	if len(chunk) == 0 || !bytes.HasPrefix(chunk, []byte("data: ")) {
		return false
	}

	// Skip "[DONE]" message
	if bytes.Contains(chunk, []byte("[DONE]")) {
		return false
	}

	return true
}

// parseStreamChunk extracts and parses JSON from SSE chunk
func (sp *StreamProcessor) parseStreamChunk(chunk []byte) (map[string]interface{}, error) {
	// Extract the JSON portion from the chunk
	jsonData := chunk[6:] // Skip "data: " prefix

	var chunkData map[string]interface{}
	if err := json.Unmarshal(jsonData, &chunkData); err != nil {
		logger.Error("Error unmarshaling stream chunk", "error", err, "vendor", sp.Vendor)
		return nil, err
	}

	return chunkData, nil
}

// applyConversationConsistency applies consistent conversation-level values
func (sp *StreamProcessor) applyConversationConsistency(chunkData map[string]interface{}) {
	// Always use the consistent conversation ID (override vendor-provided ID for transparency)
	chunkData["id"] = sp.ConversationID

	// Always use the consistent timestamp (override vendor-provided timestamp for consistency)
	chunkData["created"] = sp.Timestamp

	// Add service_tier if missing (OpenAI compatibility)
	if _, ok := chunkData["service_tier"]; !ok {
		chunkData["service_tier"] = "default"
	}

	// Always use the consistent system fingerprint (override vendor-provided fingerprint for consistency)
	chunkData["system_fingerprint"] = sp.SystemFingerprint
}

// replaceModelField replaces the model field with the original requested model
func (sp *StreamProcessor) replaceModelField(chunkData map[string]interface{}) {
	if sp.OriginalModel != "" {
		chunkData["model"] = sp.OriginalModel
	}
}

// addUsageForFirstChunk adds usage information for the first chunk if needed
func (sp *StreamProcessor) addUsageForFirstChunk(chunkData map[string]interface{}) {
	// First chunk is usually identified by delta containing role field
	isFirstChunk := false
	if choices, ok := chunkData["choices"].([]interface{}); ok && len(choices) > 0 {
		if choiceMap, ok := choices[0].(map[string]interface{}); ok {
			if delta, ok := choiceMap["delta"].(map[string]interface{}); ok {
				if _, ok := delta["role"]; ok {
					isFirstChunk = true
				}
			}
		}
	}

	if isFirstChunk {
		// Add usage if missing
		_, hasUsage := chunkData["usage"]
		if !hasUsage {
			chunkData["usage"] = map[string]interface{}{
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
}

// reconstructSSE reconstructs SSE format from processed data
func (sp *StreamProcessor) reconstructSSE(chunkData map[string]interface{}) []byte {
	// Marshal the processed data back to JSON
	modifiedJSON, err := json.Marshal(chunkData)
	if err != nil {
		logger.Error("Error marshaling modified stream chunk", "error", err, "vendor", sp.Vendor)
		return nil
	}

	// Return the SSE formatted chunk with proper line endings
	result := append([]byte("data: "), modifiedJSON...)
	result = append(result, '\n', '\n')
	return result
}
