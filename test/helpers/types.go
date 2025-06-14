package helpers

// Common test request/response types used across all tests

// HealthResponse represents the health endpoint response
type HealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp string                 `json:"timestamp"`
	Services  map[string]string      `json:"services"`
	Details   map[string]interface{} `json:"details"`
}

// ChatCompletionRequest represents a chat completion request
type ChatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
	Tools       []Tool    `json:"tools,omitempty"`
	ToolChoice  string    `json:"tool_choice,omitempty"`
}

// Message represents a message in the conversation
type Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // Can be string or []ContentPart
}

// ContentPart represents a part of the message content for vision/file requests
type ContentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
	FileURL  *FileURL  `json:"file_url,omitempty"`
}

// ImageURL represents an image URL structure
type ImageURL struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

// FileURL represents a file URL structure
type FileURL struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

// Tool represents a function tool
type Tool struct {
	Type     string                 `json:"type"`
	Function map[string]interface{} `json:"function"`
}

// ChatCompletionResponse represents a chat completion response
type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a choice in the response
type Choice struct {
	Index        int             `json:"index"`
	Message      ResponseMessage `json:"message"`
	FinishReason string          `json:"finish_reason"`
}

// ResponseMessage represents the message in the response
type ResponseMessage struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall represents a tool call in the response
type ToolCall struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Function map[string]interface{} `json:"function"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ModelsResponse represents the models endpoint response
type ModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

// Model represents a model in the models list
type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error ErrorInfo `json:"error"`
}

// ErrorInfo represents error information
type ErrorInfo struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
}

// StreamEvent represents a streaming event
type StreamEvent struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []StreamChoice `json:"choices"`
}

// StreamChoice represents a choice in a streaming response
type StreamChoice struct {
	Index        int                   `json:"index"`
	Delta        StreamResponseMessage `json:"delta"`
	FinishReason *string               `json:"finish_reason"`
}

// StreamResponseMessage represents a delta message in streaming
type StreamResponseMessage struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}
