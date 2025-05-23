package proxy

import (
	"bytes"
	"encoding/json"
	"log"
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

// ProcessChunk processes a single chunk of a streaming response with consistent conversation-level values
func (sp *StreamProcessor) ProcessChunk(chunk []byte) []byte {
	// 1. Validate SSE format
	if !sp.isValidStreamChunk(chunk) {
		return chunk
	}

	// 2. Parse JSON from chunk
	chunkData, err := sp.parseStreamChunk(chunk)
	if err != nil {
		return chunk
	}

	// 3. Apply conversation-level consistency
	sp.applyConversationConsistency(chunkData)

	// 4. Replace model field
	sp.replaceModelField(chunkData)

	// 5. Process choices (delta vs message)
	sp.processStreamChoices(chunkData)

	// 6. Handle usage for first chunk
	if sp.isFirstChunk {
		sp.addUsageForFirstChunk(chunkData)
		sp.isFirstChunk = false
	}

	// 7. Reconstruct SSE format
	return sp.reconstructSSE(chunkData)
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
		log.Printf("Error unmarshaling stream chunk: %v", err)
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

// processStreamChoices processes choices in streaming response (delta vs message)
func (sp *StreamProcessor) processStreamChoices(chunkData map[string]interface{}) {
	choices, ok := chunkData["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		log.Printf("No choices found in stream chunk")
		return
	}

	log.Printf("Processing %d choices in stream chunk", len(choices))
	for i, choice := range choices {
		choiceMap, ok := choice.(map[string]interface{})
		if !ok {
			log.Printf("Stream chunk choice %d is not a map", i)
			continue
		}

		// Add logprobs if missing
		if _, ok := choiceMap["logprobs"]; !ok {
			choiceMap["logprobs"] = nil
		}

		// Check for delta (streaming) or message (first chunk)
		if delta, ok := choiceMap["delta"].(map[string]interface{}); ok {
			sp.processStreamDelta(delta)
			choiceMap["delta"] = delta
		} else if message, ok := choiceMap["message"].(map[string]interface{}); ok {
			sp.processStreamMessage(message)
			choiceMap["message"] = message
		} else {
			log.Printf("No delta or message found in stream chunk choice %d", i)
		}

		choices[i] = choiceMap
	}
	chunkData["choices"] = choices
}

// processStreamDelta processes delta content in streaming response
func (sp *StreamProcessor) processStreamDelta(delta map[string]interface{}) {
	log.Printf("Processing delta in stream chunk")

	// Add annotations array if missing in delta
	if _, ok := delta["annotations"]; !ok {
		delta["annotations"] = []interface{}{}
	}

	// Add refusal if missing in delta
	if _, ok := delta["refusal"]; !ok {
		delta["refusal"] = nil
	}

	// Process tool_calls if present
	if toolCalls, ok := delta["tool_calls"].([]interface{}); ok && len(toolCalls) > 0 {
		log.Printf("Processing %d tool calls in stream chunk delta", len(toolCalls))
		processedToolCalls := ProcessToolCalls(toolCalls, sp.Vendor)
		delta["tool_calls"] = processedToolCalls
	} else {
		log.Printf("No tool calls found in stream chunk delta")
	}
}

// processStreamMessage processes message content in streaming response
func (sp *StreamProcessor) processStreamMessage(message map[string]interface{}) {
	log.Printf("Processing message in stream chunk")

	// Add annotations array if missing
	if _, ok := message["annotations"]; !ok {
		message["annotations"] = []interface{}{}
	}

	// Add refusal if missing
	if _, ok := message["refusal"]; !ok {
		message["refusal"] = nil
	}

	// Process tool_calls if present
	if toolCalls, ok := message["tool_calls"].([]interface{}); ok && len(toolCalls) > 0 {
		log.Printf("Processing %d tool calls in stream chunk message", len(toolCalls))
		processedToolCalls := ProcessToolCalls(toolCalls, sp.Vendor)
		message["tool_calls"] = processedToolCalls
	} else {
		log.Printf("No tool calls found in stream chunk message")
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

// reconstructSSE reconstructs the SSE format from processed data
func (sp *StreamProcessor) reconstructSSE(chunkData map[string]interface{}) []byte {
	// Encode the modified chunk
	modifiedData, err := json.Marshal(chunkData)
	if err != nil {
		log.Printf("Error marshaling modified stream chunk: %v", err)
		return []byte{} // Return empty if marshal fails
	}

	// Reconstruct the chunk with "data: " prefix
	return []byte("data: " + string(modifiedData) + "\n\n")
}
