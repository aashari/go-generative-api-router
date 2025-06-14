package fixtures

import "github.com/aashari/go-generative-api-router/test/helpers"

// Common test request fixtures

// BasicChatRequest returns a basic chat completion request
func BasicChatRequest() helpers.ChatCompletionRequest {
	return helpers.ChatCompletionRequest{
		Model: "gpt-4o",
		Messages: []helpers.Message{
			{
				Role:    "user",
				Content: "Hello, how are you?",
			},
		},
		MaxTokens:   50,
		Temperature: 0.7,
	}
}

// StreamingChatRequest returns a streaming chat completion request
func StreamingChatRequest() helpers.ChatCompletionRequest {
	return helpers.ChatCompletionRequest{
		Model: "gpt-4o",
		Messages: []helpers.Message{
			{
				Role:    "user",
				Content: "Count from 1 to 5, one number per line",
			},
		},
		MaxTokens:   50,
		Temperature: 0.1,
		Stream:      true,
	}
}

// VisionChatRequest returns a chat request with image content
func VisionChatRequest() helpers.ChatCompletionRequest {
	return helpers.ChatCompletionRequest{
		Model: "gpt-4o",
		Messages: []helpers.Message{
			{
				Role: "user",
				Content: []helpers.ContentPart{
					{
						Type: "text",
						Text: "What do you see in this image?",
					},
					{
						Type: "image_url",
						ImageURL: &helpers.ImageURL{
							URL: "https://upload.wikimedia.org/wikipedia/commons/thumb/3/3a/Cat03.jpg/400px-Cat03.jpg",
						},
					},
				},
			},
		},
		MaxTokens: 100,
	}
}

// VisionWithHeadersRequest returns a vision request with custom headers
func VisionWithHeadersRequest() helpers.ChatCompletionRequest {
	return helpers.ChatCompletionRequest{
		Model: "gpt-4o",
		Messages: []helpers.Message{
			{
				Role: "user",
				Content: []helpers.ContentPart{
					{
						Type: "text",
						Text: "Analyze this protected image",
					},
					{
						Type: "image_url",
						ImageURL: &helpers.ImageURL{
							URL: "https://api.example.com/protected/image.jpg",
							Headers: map[string]string{
								"Authorization": "Bearer test-token",
								"User-Agent":    "CustomBot/1.0",
							},
						},
					},
				},
			},
		},
		MaxTokens: 50,
	}
}

// ToolCallingRequest returns a request with tool calling
func ToolCallingRequest() helpers.ChatCompletionRequest {
	return helpers.ChatCompletionRequest{
		Model: "gpt-4o",
		Messages: []helpers.Message{
			{
				Role:    "user",
				Content: "What is the weather in Boston?",
			},
		},
		Tools: []helpers.Tool{
			{
				Type: "function",
				Function: map[string]interface{}{
					"name":        "get_weather",
					"description": "Get weather information for a location",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"location": map[string]interface{}{
								"type":        "string",
								"description": "City name",
							},
						},
						"required": []string{"location"},
					},
				},
			},
		},
		ToolChoice: "auto",
		MaxTokens:  100,
	}
}

// LargeChatRequest returns a request with very large content
func LargeChatRequest() helpers.ChatCompletionRequest {
	largeContent := ""
	for i := 0; i < 1000; i++ {
		largeContent += "This is a very long message that tests the router's ability to handle large payloads. "
	}

	return helpers.ChatCompletionRequest{
		Model: "gpt-4o",
		Messages: []helpers.Message{
			{
				Role:    "user",
				Content: largeContent,
			},
		},
		MaxTokens: 10,
	}
}

// InvalidRequest returns an invalid request (missing required fields)
func InvalidRequest() map[string]interface{} {
	return map[string]interface{}{
		"model": "gpt-4o",
		// Missing required "messages" field
	}
}

// CustomModelRequest returns a request with a custom model name
func CustomModelRequest(modelName string) helpers.ChatCompletionRequest {
	return helpers.ChatCompletionRequest{
		Model: modelName,
		Messages: []helpers.Message{
			{
				Role:    "user",
				Content: "Test message for custom model",
			},
		},
		MaxTokens: 20,
	}
}

// MinimalRequest returns the most minimal valid request
func MinimalRequest() helpers.ChatCompletionRequest {
	return helpers.ChatCompletionRequest{
		Model: "gpt-4o",
		Messages: []helpers.Message{
			{
				Role:    "user",
				Content: "Hi",
			},
		},
	}
}
