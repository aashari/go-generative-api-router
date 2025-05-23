package transformers

import (
	"context"

	"github.com/aashari/go-generative-api-router/internal/vendors"
)

// ResponseTransformer handles transformation of vendor responses to standardized format
type ResponseTransformer interface {
	// TransformResponse transforms a vendor response to the standard format
	TransformResponse(ctx context.Context, response *vendors.ChatResponse, originalModel string) (*vendors.ChatResponse, error)
	
	// TransformStreamChunk transforms a streaming response chunk
	TransformStreamChunk(ctx context.Context, chunk []byte, originalModel string, metadata StreamMetadata) ([]byte, error)
	
	// GetVendorName returns the vendor this transformer handles
	GetVendorName() string
}

// StreamMetadata holds metadata for streaming transformations
type StreamMetadata struct {
	ConversationID    string
	Timestamp         int64
	SystemFingerprint string
	VendorName        string
}

// ToolCallProcessor handles tool call processing and ID generation
type ToolCallProcessor interface {
	// ProcessToolCalls processes tool calls in a message
	ProcessToolCalls(ctx context.Context, toolCalls []vendors.ToolCall, vendorName string) ([]vendors.ToolCall, error)
	
	// GenerateToolCallID generates a new tool call ID
	GenerateToolCallID() string
	
	// ValidateToolCall validates a tool call structure
	ValidateToolCall(ctx context.Context, toolCall vendors.ToolCall) error
}

// TransformerFactory creates response transformers
type TransformerFactory interface {
	// CreateTransformer creates a transformer for the specified vendor
	CreateTransformer(ctx context.Context, vendorName string) (ResponseTransformer, error)
	
	// GetSupportedVendors returns supported vendor names
	GetSupportedVendors() []string
} 