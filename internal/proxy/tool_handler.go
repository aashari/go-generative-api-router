package proxy

import (
	"log"
)

// ProcessToolCalls processes a list of tool calls, adding or updating IDs as needed.
// It handles vendor-specific logic for ID generation.
// Returns the processed tool calls array.
func ProcessToolCalls(toolCalls []interface{}, vendor string) []interface{} {
	// Handle nil or empty toolCalls array
	if toolCalls == nil || len(toolCalls) == 0 {
		return toolCalls
	}

	log.Printf("Processing %d tool calls for vendor: %s", len(toolCalls), vendor)

	// Process each tool call
	for j, toolCall := range toolCalls {
		toolCallMap, ok := toolCall.(map[string]interface{})
		if !ok {
			log.Printf("Tool call %d is not a map, skipping processing", j)
			continue
		}

		// Check if "id" field exists and what its value is
		toolCallID, idExists := toolCallMap["id"]
		log.Printf("Tool call %d has ID: %v (exists: %v)", j, toolCallID, idExists)

		// Force override for Gemini vendor or if ID is missing/empty
		if vendor == "gemini" {
			// Always generate a new ID for Gemini responses regardless of current value
			newID := ToolCallID()
			log.Printf("FORCING new tool call ID for Gemini vendor: %s (was: %v)", newID, toolCallID)
			toolCallMap["id"] = newID
		} else if !idExists || toolCallID == nil || toolCallID == "" {
			// For other vendors, only generate if missing/empty
			newID := ToolCallID()
			log.Printf("Generated new tool call ID for %s: %s", vendor, newID)
			toolCallMap["id"] = newID
		}

		toolCalls[j] = toolCallMap
	}

	return toolCalls
} 