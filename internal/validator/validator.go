package validator

import (
	"encoding/json"
	"fmt"
)

// ValidateAndModifyRequest validates the request and modifies it with the selected model
// Returns the modified body and the original model value from the request
func ValidateAndModifyRequest(body []byte, model string) ([]byte, string, error) {
	var requestData map[string]interface{}
	if err := json.Unmarshal(body, &requestData); err != nil {
		return nil, "", fmt.Errorf("invalid request format: %v", err)
	}

	// Validate messages exist
	if err := validateMessages(requestData); err != nil {
		return nil, "", err
	}

	// Validate message content format (string or array for vision)
	if err := validateMessageContent(requestData); err != nil {
		return nil, "", err
	}

	// Validate tools if present
	if err := validateTools(requestData); err != nil {
		return nil, "", err
	}

	// Validate tool_choice if present
	if err := validateToolChoice(requestData); err != nil {
		return nil, "", err
	}

	// Validate stream if present
	if err := validateStream(requestData); err != nil {
		return nil, "", err
	}

	// Extract the original model before replacing it
	originalModel, _ := requestData["model"].(string)
	if originalModel == "" {
		originalModel = "any-model" // Default if no model provided
	}

	// Replace the model with our selected one
	requestData["model"] = model

	// Re-encode the modified request
	modifiedBody, err := json.Marshal(requestData)
	if err != nil {
		return nil, "", fmt.Errorf("failed to encode modified request: %v", err)
	}

	return modifiedBody, originalModel, nil
}

// validateMessages checks if the messages field exists
func validateMessages(requestData map[string]interface{}) error {
	if _, ok := requestData["messages"]; !ok {
		return fmt.Errorf("missing 'messages' field in request")
	}
	return nil
}

// validateMessageContent validates the content field in messages
func validateMessageContent(requestData map[string]interface{}) error {
	messages, ok := requestData["messages"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid 'messages' format: must be an array")
	}

	for i, msg := range messages {
		msgMap, ok := msg.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid message at index %d: must be an object", i)
		}

		// Check if content exists
		content, hasContent := msgMap["content"]
		if !hasContent {
			// Content might be optional for some message types (e.g., assistant messages with tool calls)
			continue
		}

		// Content can be either a string or an array (for vision requests)
		switch content := content.(type) {
		case string:
			// Valid string content
			continue
		case []interface{}:
			// Valid array content - validate each part
			if err := validateContentArray(content); err != nil {
				return fmt.Errorf("invalid content array in message %d: %v", i, err)
			}
		default:
			return fmt.Errorf("invalid content type in message %d: must be string or array", i)
		}
	}

	return nil
}

// validateContentArray validates an array of content parts
func validateContentArray(content []interface{}) error {
	if len(content) == 0 {
		return fmt.Errorf("content array cannot be empty")
	}

	for i, part := range content {
		partMap, ok := part.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid content part at index %d: must be an object", i)
		}

		// Validate type field
		typeField, hasType := partMap["type"].(string)
		if !hasType {
			return fmt.Errorf("content part at index %d missing 'type' field", i)
		}

		// Validate based on type
		switch typeField {
		case "text":
			if _, hasText := partMap["text"].(string); !hasText {
				return fmt.Errorf("text content part at index %d missing 'text' field", i)
			}
		case "image_url":
			imageURL, hasImageURL := partMap["image_url"].(map[string]interface{})
			if !hasImageURL {
				return fmt.Errorf("image_url content part at index %d missing 'image_url' field", i)
			}
			if _, hasURL := imageURL["url"].(string); !hasURL {
				return fmt.Errorf("image_url content part at index %d missing 'url' field", i)
			}
		case "file_url":
			fileURL, hasFileURL := partMap["file_url"].(map[string]interface{})
			if !hasFileURL {
				return fmt.Errorf("file_url content part at index %d missing 'file_url' field", i)
			}
			if _, hasURL := fileURL["url"].(string); !hasURL {
				return fmt.Errorf("file_url content part at index %d missing 'url' field", i)
			}
		default:
			return fmt.Errorf("unknown content type '%s' at index %d", typeField, i)
		}
	}

	return nil
}

// validateTools checks if the tools field is properly formatted
func validateTools(requestData map[string]interface{}) error {
	tools, ok := requestData["tools"]
	if !ok {
		// Tools field is optional
		return nil
	}

	toolsArr, ok := tools.([]interface{})
	if !ok {
		return fmt.Errorf("invalid 'tools' format: must be an array")
	}

	for _, tool := range toolsArr {
		toolMap, ok := tool.(map[string]interface{})
		if !ok || toolMap["type"] != "function" || toolMap["function"] == nil {
			return fmt.Errorf("invalid 'tools' format: each tool must have type 'function' and a 'function' object")
		}
	}

	return nil
}

// validateToolChoice checks if the tool_choice field is properly formatted
func validateToolChoice(requestData map[string]interface{}) error {
	toolChoice, ok := requestData["tool_choice"]
	if !ok {
		// Tool choice field is optional
		return nil
	}

	switch v := toolChoice.(type) {
	case string:
		if v != "none" && v != "auto" && v != "required" {
			return fmt.Errorf("invalid 'tool_choice': must be 'none', 'auto', 'required', or a function object")
		}
	case map[string]interface{}:
		if v["type"] != "function" || v["function"] == nil {
			return fmt.Errorf("invalid 'tool_choice': function object must have type 'function' and a 'function' field")
		}
	default:
		return fmt.Errorf("invalid 'tool_choice': must be a string or function object")
	}

	return nil
}

// validateStream ensures the 'stream' field, if present, is boolean
func validateStream(requestData map[string]interface{}) error {
	stream, exists := requestData["stream"]
	if exists {
		if _, ok := stream.(bool); !ok {
			return fmt.Errorf("invalid 'stream' field: must be boolean")
		}
	}
	return nil
}
