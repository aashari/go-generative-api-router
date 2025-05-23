package vendors

import (
	"context"
	"time"
)

// VendorClient defines the interface for vendor-specific API clients
type VendorClient interface {
	// ChatCompletion sends a non-streaming chat completion request
	ChatCompletion(ctx context.Context, request *ChatRequest) (*ChatResponse, error)
	
	// StreamChatCompletion sends a streaming chat completion request
	StreamChatCompletion(ctx context.Context, request *ChatRequest) (StreamReader, error)
	
	// GetVendorName returns the vendor identifier
	GetVendorName() string
	
	// GetBaseURL returns the base URL for this vendor
	GetBaseURL() string
	
	// ValidateCredential validates the credential for this vendor
	ValidateCredential(ctx context.Context, credential string) error
}

// StreamReader defines the interface for reading streaming responses
type StreamReader interface {
	// Read returns the next chunk from the stream
	Read() ([]byte, error)
	
	// Close closes the stream
	Close() error
	
	// IsEOF returns true if the stream has ended
	IsEOF() bool
}

// ChatRequest represents a standardized chat completion request
type ChatRequest struct {
	Model       string                 `json:"model"`
	Messages    []Message              `json:"messages"`
	Tools       []Tool                 `json:"tools,omitempty"`
	ToolChoice  interface{}            `json:"tool_choice,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
	Temperature *float64               `json:"temperature,omitempty"`
	MaxTokens   *int                   `json:"max_tokens,omitempty"`
	TopP        *float64               `json:"top_p,omitempty"`
	Extra       map[string]interface{} `json:"-"` // For vendor-specific fields
}

// ChatResponse represents a standardized chat completion response
type ChatResponse struct {
	ID                string    `json:"id"`
	Object            string    `json:"object"`
	Created           int64     `json:"created"`
	Model             string    `json:"model"`
	SystemFingerprint string    `json:"system_fingerprint,omitempty"`
	Choices           []Choice  `json:"choices"`
	Usage             *Usage    `json:"usage,omitempty"`
	ServiceTier       string    `json:"service_tier,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role         string      `json:"role"`
	Content      string      `json:"content"`
	ToolCalls    []ToolCall  `json:"tool_calls,omitempty"`
	ToolCallID   string      `json:"tool_call_id,omitempty"`
	Refusal      *string     `json:"refusal,omitempty"`
	Annotations  []interface{} `json:"annotations,omitempty"`
}

// Choice represents a response choice
type Choice struct {
	Index        int      `json:"index"`
	Message      *Message `json:"message,omitempty"`
	Delta        *Message `json:"delta,omitempty"`
	FinishReason *string  `json:"finish_reason,omitempty"`
	Logprobs     interface{} `json:"logprobs,omitempty"`
}

// ToolCall represents a tool call
type ToolCall struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

// Function represents a function call
type Function struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Tool represents a tool definition
type Tool struct {
	Type     string      `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction represents a tool function definition
type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// VendorFactory creates vendor clients
type VendorFactory interface {
	CreateClient(vendorName string, credential string, config VendorConfig) (VendorClient, error)
	GetSupportedVendors() []string
}

// VendorConfig holds vendor-specific configuration
type VendorConfig struct {
	BaseURL     string            `json:"base_url"`
	Timeout     time.Duration     `json:"timeout"`
	MaxRetries  int               `json:"max_retries"`
	ExtraConfig map[string]interface{} `json:"extra_config,omitempty"`
} 