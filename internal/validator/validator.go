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
