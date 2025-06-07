package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/aashari/go-generative-api-router/internal/logger"
)

// ProcessToolCalls processes a list of tool calls, adding or updating IDs as needed.
// It handles vendor-specific logic for ID generation and validates/splits malformed arguments.
// Returns the processed tool calls array.
func ProcessToolCalls(toolCalls []interface{}, vendor string) []interface{} {
	// Handle nil or empty toolCalls array
	if toolCalls == nil || len(toolCalls) == 0 {
		return toolCalls
	}

	// Log complete tool calls processing with all data
	logger.LogWithStructure(context.Background(), logger.LevelInfo, "Processing tool calls with complete data",
		map[string]interface{}{
			"complete_tool_calls": toolCalls,
			"vendor":              vendor,
			"tool_calls_count":    len(toolCalls),
			"tool_calls_type":     fmt.Sprintf("%T", toolCalls),
		},
		nil, // request
		nil, // response
		nil)

	var processedToolCalls []interface{}

	// Process each tool call
	for j, toolCall := range toolCalls {
		toolCallMap, ok := toolCall.(map[string]interface{})
		if !ok {
			// Log complete data for non-map tool call
			logger.LogWithStructure(context.Background(), logger.LevelInfo, "Tool call not a map with complete data",
				map[string]interface{}{
					"index":              j,
					"complete_tool_call": toolCall,
					"tool_call_type":     fmt.Sprintf("%T", toolCall),
					"vendor":             vendor,
					"all_tool_calls":     toolCalls,
				},
				nil, // request
				nil, // response
				nil)
			processedToolCalls = append(processedToolCalls, toolCall)
			continue
		}

		// Check if "id" field exists and what its value is
		toolCallID, idExists := toolCallMap["id"]
		// Log complete tool call ID information
		logger.LogWithStructure(context.Background(), logger.LevelInfo, "Tool call ID info with complete data",
			map[string]interface{}{
				"index":                  j,
				"id":                     toolCallID,
				"id_exists":              idExists,
				"complete_tool_call_map": toolCallMap,
				"vendor":                 vendor,
				"all_tool_calls":         toolCalls,
			},
			nil, // request
			nil, // response
			nil)

		// Check for malformed arguments and split if needed
		if function, ok := toolCallMap["function"].(map[string]interface{}); ok {
			if arguments, ok := function["arguments"].(string); ok {
				splitToolCalls := validateAndSplitArguments(toolCallMap, arguments, vendor)
				if len(splitToolCalls) > 1 {
					// Log complete split operation data
					logger.LogWithStructure(context.Background(), logger.LevelInfo, "Split malformed tool call with complete data",
						map[string]interface{}{
							"index":                  j,
							"splits_count":           len(splitToolCalls),
							"original_tool_call":     toolCallMap,
							"original_arguments":     arguments,
							"complete_split_results": splitToolCalls,
							"vendor":                 vendor,
							"all_tool_calls":         toolCalls,
						},
						nil, // request
						nil, // response
						nil)
					processedToolCalls = append(processedToolCalls, splitToolCalls...)
					continue
				}
			}
		}

		// Force override for Gemini vendor or if ID is missing/empty
		if vendor == "gemini" {
			// Always generate a new ID for Gemini responses regardless of current value
			newID := ToolCallID()
			// Log complete Gemini ID forcing operation
			logger.LogWithStructure(context.Background(), logger.LevelInfo, "Forcing new tool call ID for Gemini with complete data",
				map[string]interface{}{
					"new_id":                    newID,
					"old_id":                    toolCallID,
					"complete_tool_call_before": toolCallMap,
					"vendor":                    vendor,
					"all_tool_calls":            toolCalls,
					"index":                     j,
				},
				nil, // request
				nil, // response
				nil)
			toolCallMap["id"] = newID
		} else if !idExists || toolCallID == nil || toolCallID == "" {
			// For other vendors, only generate if missing/empty
			newID := ToolCallID()
			// Log complete ID generation operation
			logger.LogWithStructure(context.Background(), logger.LevelInfo, "Generated new tool call ID with complete data",
				map[string]interface{}{
					"vendor":                    vendor,
					"new_id":                    newID,
					"old_id":                    toolCallID,
					"id_existed":                idExists,
					"complete_tool_call_before": toolCallMap,
					"all_tool_calls":            toolCalls,
					"index":                     j,
				},
				nil, // request
				nil, // response
				nil)
			toolCallMap["id"] = newID
		}

		processedToolCalls = append(processedToolCalls, toolCallMap)
	}

	// Log complete processing results
	logger.LogWithStructure(context.Background(), logger.LevelInfo, "Tool calls processing completed with complete data",
		map[string]interface{}{
			"original_tool_calls":  toolCalls,
			"processed_tool_calls": processedToolCalls,
			"vendor":               vendor,
			"original_count":       len(toolCalls),
			"processed_count":      len(processedToolCalls),
		},
		nil, // request
		nil, // response
		nil)

	return processedToolCalls
}

// validateAndSplitArguments validates function call arguments and splits them if they contain multiple JSON objects
func validateAndSplitArguments(originalToolCall map[string]interface{}, arguments string, vendor string) []interface{} {
	// Check for patterns that indicate multiple JSON objects concatenated together
	if !containsMultipleJSONObjects(arguments) {
		// Single valid JSON object, return as-is
		return []interface{}{originalToolCall}
	}

	// Log complete malformed arguments detection
	logger.LogWithStructure(context.Background(), logger.LevelInfo, "Detected malformed arguments with complete data",
		map[string]interface{}{
			"complete_arguments":          arguments,
			"arguments_length":            len(arguments),
			"complete_original_tool_call": originalToolCall,
			"vendor":                      vendor,
		},
		nil, // request
		nil, // response
		nil)

	// Split the arguments into separate JSON objects
	jsonObjects := splitJSONObjects(arguments)
	if len(jsonObjects) <= 1 {
		// Couldn't split properly, return original
		logger.LogWithStructure(context.Background(), logger.LevelInfo, "Failed to split arguments with complete data",
			map[string]interface{}{
				"complete_arguments":          arguments,
				"split_results":               jsonObjects,
				"complete_original_tool_call": originalToolCall,
				"vendor":                      vendor,
			},
			nil, // request
			nil, // response
			nil)
		return []interface{}{originalToolCall}
	}

	// Log complete successful split operation
	logger.LogWithStructure(context.Background(), logger.LevelInfo, "Successfully split arguments with complete data",
		map[string]interface{}{
			"complete_arguments":          arguments,
			"split_count":                 len(jsonObjects),
			"complete_json_objects":       jsonObjects,
			"complete_original_tool_call": originalToolCall,
			"vendor":                      vendor,
		},
		nil, // request
		nil, // response
		nil)

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

		// Log complete split tool call creation
		logger.LogWithStructure(context.Background(), logger.LevelInfo, "Created split tool call with complete data",
			map[string]interface{}{
				"index":                       i + 1,
				"new_id":                      newID,
				"json_object":                 jsonObj,
				"complete_new_tool_call":      newToolCall,
				"complete_original_tool_call": originalToolCall,
				"complete_original_function":  function,
				"vendor":                      vendor,
			},
			nil, // request
			nil, // response
			nil)
		splitToolCalls = append(splitToolCalls, newToolCall)
	}

	// Log complete split operation results
	logger.LogWithStructure(context.Background(), logger.LevelInfo, "Split tool calls operation completed with complete data",
		map[string]interface{}{
			"complete_original_tool_call": originalToolCall,
			"complete_split_tool_calls":   splitToolCalls,
			"original_arguments":          arguments,
			"split_json_objects":          jsonObjects,
			"vendor":                      vendor,
			"split_count":                 len(splitToolCalls),
		},
		nil, // request
		nil, // response
		nil)

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
		// Log complete pattern detection
		logger.LogWithStructure(context.Background(), logger.LevelInfo, "Found multiple JSON objects pattern with complete data",
			map[string]interface{}{
				"pattern":            "}{",
				"complete_arguments": arguments,
				"arguments_length":   len(arguments),
			},
			nil, // request
			nil, // response
			nil)
		return true
	}

	// Pattern 2: ][ indicates two arrays concatenated
	if strings.Contains(arguments, "][") {
		// Log complete pattern detection
		logger.LogWithStructure(context.Background(), logger.LevelInfo, "Found multiple JSON arrays pattern with complete data",
			map[string]interface{}{
				"pattern":            "][",
				"complete_arguments": arguments,
				"arguments_length":   len(arguments),
			},
			nil, // request
			nil, // response
			nil)
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
		// Log complete additional content detection
		logger.LogWithStructure(context.Background(), logger.LevelInfo, "Found additional JSON content after first object with complete data",
			map[string]interface{}{
				"complete_arguments":  arguments,
				"first_parsed_object": firstObj,
				"arguments_length":    len(arguments),
			},
			nil, // request
			nil, // response
			nil)
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
				// Log complete invalid JSON data
				logger.LogWithStructure(context.Background(), logger.LevelInfo, "Invalid JSON after splitting with complete data",
					map[string]interface{}{
						"invalid_part":       part,
						"part_index":         i,
						"complete_parts":     parts,
						"original_arguments": arguments,
						"split_pattern":      "}{",
						"current_results":    results,
					},
					nil, // request
					nil, // response
					nil)
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
				// Log complete invalid JSON array data
				logger.LogWithStructure(context.Background(), logger.LevelInfo, "Invalid JSON array after splitting with complete data",
					map[string]interface{}{
						"invalid_part":       part,
						"part_index":         i,
						"complete_parts":     parts,
						"original_arguments": arguments,
						"split_pattern":      "][",
						"current_results":    results,
					},
					nil, // request
					nil, // response
					nil)
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
				// Log complete JSON parsing error
				logger.LogWithStructure(context.Background(), logger.LevelInfo, "Error parsing JSON object with complete data",
					map[string]interface{}{
						"error":                 err.Error(),
						"complete_arguments":    arguments,
						"parsed_objects_so_far": objects,
						"decoder_position":      "unknown", // decoder doesn't expose position easily,
					},
					nil, // request
					nil, // response
					nil)
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

	// Log complete split operation results
	logger.LogWithStructure(context.Background(), logger.LevelInfo, "Split JSON objects operation completed with complete data",
		map[string]interface{}{
			"original_arguments": arguments,
			"split_results":      results,
			"results_count":      len(results),
			"arguments_length":   len(arguments),
		},
		map[string]interface{}{
			"methods_attempted": []string{"}{_pattern", "][_pattern", "sequential_parsing"},
		},
		nil, // response
		nil)
	return results
}

// isValidJSON checks if a string is valid JSON
func isValidJSON(s string) bool {
	var js interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}
