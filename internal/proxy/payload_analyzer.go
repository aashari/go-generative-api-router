package proxy

import (
	"encoding/json"

	"github.com/aashari/go-generative-api-router/internal/types"
)

// AnalyzePayload extracts routing-relevant information from the request payload
func AnalyzePayload(body []byte) (*types.PayloadContext, error) {
	var requestData map[string]interface{}
	if err := json.Unmarshal(body, &requestData); err != nil {
		return nil, err
	}

	context := &types.PayloadContext{}

	// Extract original model
	if model, ok := requestData["model"].(string); ok {
		context.OriginalModel = model
	} else {
		context.OriginalModel = "any-model"
	}

	// Check for streaming
	if stream, ok := requestData["stream"].(bool); ok {
		context.HasStream = stream
	}

	// Check for tools
	if tools, ok := requestData["tools"].([]interface{}); ok && len(tools) > 0 {
		context.HasTools = true
	}

	// Analyze messages
	if messages, ok := requestData["messages"].([]interface{}); ok {
		context.MessagesCount = len(messages)

		// Check for images and videos in message content
		for _, msg := range messages {
			if msgMap, ok := msg.(map[string]interface{}); ok {
				// Check if content is an array (for multimodal messages)
				if content, ok := msgMap["content"].([]interface{}); ok {
					for _, part := range content {
						if partMap, ok := part.(map[string]interface{}); ok {
							if partType, ok := partMap["type"].(string); ok {
								switch partType {
								case "image_url":
									context.HasImages = true
								case "video_url":
									context.HasVideos = true
								}
							}
						}
					}
				}
			}
		}
	}

	// TODO: Add token estimation logic here if needed

	return context, nil
}

// ShouldExcludeModel determines if a model should be excluded based on payload context
// This will be used when model configuration is extended with capabilities
func ShouldExcludeModel(context *types.PayloadContext, modelConfig map[string]interface{}) bool {
	// Example implementation for future use:
	// if context.HasImages && modelConfig["support_image"] == false {
	//     return true
	// }
	// if context.HasVideos && modelConfig["support_video"] == false {
	//     return true
	// }
	return false
}
