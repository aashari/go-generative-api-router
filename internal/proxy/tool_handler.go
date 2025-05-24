package proxy

import (
	"encoding/json"
	"log"
	"regexp"
	"strings"
)

// ProcessToolCalls processes a list of tool calls, adding or updating IDs as needed.
// It handles vendor-specific logic for ID generation and validates/splits malformed arguments.
// Returns the processed tool calls array.
func ProcessToolCalls(toolCalls []interface{}, vendor string) []interface{} {
	// Handle nil or empty toolCalls array
	if toolCalls == nil || len(toolCalls) == 0 {
		return toolCalls
	}

	log.Printf("Processing %d tool calls for vendor: %s", len(toolCalls), vendor)

	var processedToolCalls []interface{}

	// Process each tool call
	for j, toolCall := range toolCalls {
		toolCallMap, ok := toolCall.(map[string]interface{})
		if !ok {
			log.Printf("Tool call %d is not a map, skipping processing", j)
			processedToolCalls = append(processedToolCalls, toolCall)
			continue
		}

		// Check if "id" field exists and what its value is
		toolCallID, idExists := toolCallMap["id"]
		log.Printf("Tool call %d has ID: %v (exists: %v)", j, toolCallID, idExists)

		// Check for malformed arguments and split if needed
		if function, ok := toolCallMap["function"].(map[string]interface{}); ok {
			if arguments, ok := function["arguments"].(string); ok {
				splitToolCalls := validateAndSplitArguments(toolCallMap, arguments, vendor)
				if len(splitToolCalls) > 1 {
					log.Printf("Split malformed tool call %d into %d separate calls", j, len(splitToolCalls))
					processedToolCalls = append(processedToolCalls, splitToolCalls...)
					continue
				}
			}
		}

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

		processedToolCalls = append(processedToolCalls, toolCallMap)
	}

	return processedToolCalls
}

// validateAndSplitArguments validates function call arguments and splits them if they contain multiple JSON objects
func validateAndSplitArguments(originalToolCall map[string]interface{}, arguments string, vendor string) []interface{} {
	// Check for patterns that indicate multiple JSON objects concatenated together
	if !containsMultipleJSONObjects(arguments) {
		// Single valid JSON object, return as-is
		return []interface{}{originalToolCall}
	}

	log.Printf("Detected malformed arguments with multiple JSON objects: %s", arguments)

	// Split the arguments into separate JSON objects
	jsonObjects := splitJSONObjects(arguments)
	if len(jsonObjects) <= 1 {
		// Couldn't split properly, return original
		log.Printf("Failed to split arguments, returning original tool call")
		return []interface{}{originalToolCall}
	}

	log.Printf("Successfully split arguments into %d JSON objects", len(jsonObjects))

	var splitToolCalls []interface{}
	function := originalToolCall["function"].(map[string]interface{})

	// Create separate tool calls for each JSON object
	for i, jsonObj := range jsonObjects {
		// Create a copy of the original tool call
		newToolCall := make(map[string]interface{})
		for k, v := range originalToolCall {
			newToolCall[k] = v
		}

		// Create a copy of the function object
		newFunction := make(map[string]interface{})
		for k, v := range function {
			newFunction[k] = v
		}

		// Update the arguments with the split JSON object
		newFunction["arguments"] = jsonObj
		newToolCall["function"] = newFunction

		// Generate new ID for each split tool call
		newID := ToolCallID()
		newToolCall["id"] = newID

		log.Printf("Created split tool call %d with ID %s and arguments: %s", i+1, newID, jsonObj)
		splitToolCalls = append(splitToolCalls, newToolCall)
	}

	return splitToolCalls
}

// containsMultipleJSONObjects checks if the arguments string contains multiple JSON objects
func containsMultipleJSONObjects(arguments string) bool {
	// Look for patterns that indicate multiple JSON objects:
	// 1. }{  - closing brace followed by opening brace
	// 2. "][" - closing bracket followed by opening bracket
	// 3. Multiple complete JSON objects

	// Pattern 1: }{ indicates two objects concatenated
	if strings.Contains(arguments, "}{") {
		log.Printf("Found }{ pattern indicating multiple JSON objects")
		return true
	}

	// Pattern 2: ][ indicates two arrays concatenated
	if strings.Contains(arguments, "][") {
		log.Printf("Found ][ pattern indicating multiple JSON arrays")
		return true
	}

	// Pattern 3: Try to parse as JSON and see if there's trailing content
	arguments = strings.TrimSpace(arguments)
	if len(arguments) == 0 {
		return false
	}

	var firstObj interface{}
	decoder := json.NewDecoder(strings.NewReader(arguments))
	if err := decoder.Decode(&firstObj); err != nil {
		// Not valid JSON at all
		return false
	}

	// Check if there's more content after the first valid JSON object
	if decoder.More() {
		log.Printf("Found additional JSON content after first object")
		return true
	}

	return false
}

// splitJSONObjects splits a string containing multiple JSON objects into separate valid JSON strings
func splitJSONObjects(arguments string) []string {
	var results []string

	// Method 1: Split on }{ pattern
	if strings.Contains(arguments, "}{") {
		parts := regexp.MustCompile(`\}\s*\{`).Split(arguments, -1)
		for i, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			// Add missing braces
			if i == 0 {
				// First part - add closing brace
				if !strings.HasSuffix(part, "}") {
					part += "}"
				}
			} else if i == len(parts)-1 {
				// Last part - add opening brace
				if !strings.HasPrefix(part, "{") {
					part = "{" + part
				}
			} else {
				// Middle parts - add both braces
				if !strings.HasPrefix(part, "{") {
					part = "{" + part
				}
				if !strings.HasSuffix(part, "}") {
					part += "}"
				}
			}

			// Validate the JSON
			if isValidJSON(part) {
				results = append(results, part)
			} else {
				log.Printf("Invalid JSON after splitting: %s", part)
			}
		}
	}

	// Method 2: Split on ][ pattern for arrays
	if len(results) == 0 && strings.Contains(arguments, "][") {
		parts := regexp.MustCompile(`\]\s*\[`).Split(arguments, -1)
		for i, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			// Add missing brackets
			if i == 0 {
				if !strings.HasSuffix(part, "]") {
					part += "]"
				}
			} else if i == len(parts)-1 {
				if !strings.HasPrefix(part, "[") {
					part = "[" + part
				}
			} else {
				if !strings.HasPrefix(part, "[") {
					part = "[" + part
				}
				if !strings.HasSuffix(part, "]") {
					part += "]"
				}
			}

			if isValidJSON(part) {
				results = append(results, part)
			} else {
				log.Printf("Invalid JSON array after splitting: %s", part)
			}
		}
	}

	// Method 3: Try to parse multiple complete JSON objects sequentially
	// Only use this if we haven't found results from pattern matching
	if len(results) == 0 {
		decoder := json.NewDecoder(strings.NewReader(arguments))
		var objects []string

		for decoder.More() {
			var obj interface{}
			if err := decoder.Decode(&obj); err != nil {
				log.Printf("Error parsing JSON object: %v", err)
				break
			}

			// Convert back to JSON string
			if jsonBytes, err := json.Marshal(obj); err == nil {
				objects = append(objects, string(jsonBytes))
			}
		}

		// Only return results if we found multiple objects
		if len(objects) > 1 {
			results = objects
		}
	}

	log.Printf("Split into %d JSON objects: %v", len(results), results)
	return results
}

// isValidJSON checks if a string is valid JSON
func isValidJSON(s string) bool {
	var js interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}
