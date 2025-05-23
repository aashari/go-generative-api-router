package transformers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/aashari/go-generative-api-router/internal/vendors"
)

// DefaultToolCallProcessor implements ToolCallProcessor interface
type DefaultToolCallProcessor struct{}

// NewToolCallProcessor creates a new tool call processor
func NewToolCallProcessor() ToolCallProcessor {
	return &DefaultToolCallProcessor{}
}

// ProcessToolCalls processes tool calls in a message, ensuring proper IDs
func (p *DefaultToolCallProcessor) ProcessToolCalls(ctx context.Context, toolCalls []vendors.ToolCall, vendorName string) ([]vendors.ToolCall, error) {
	if len(toolCalls) == 0 {
		return toolCalls, nil
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	log.Printf("Processing %d tool calls for vendor: %s", len(toolCalls), vendorName)
	
	processed := make([]vendors.ToolCall, len(toolCalls))
	for i, toolCall := range toolCalls {
		// Check for context cancellation during processing
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		
		processed[i] = toolCall
		
		// Check if ID needs to be generated or replaced
		needsNewID := false
		
		if toolCall.ID == "" {
			needsNewID = true
			log.Printf("Tool call %d missing ID, will generate new one", i)
		} else if vendorName == "gemini" {
			// Always replace Gemini tool call IDs for consistency
			needsNewID = true
			log.Printf("Tool call %d from Gemini vendor, replacing ID %s", i, toolCall.ID)
		}
		
		if needsNewID {
			newID := p.GenerateToolCallID()
			processed[i].ID = newID
			log.Printf("Generated new tool call ID: %s (was: %s)", newID, toolCall.ID)
		}
		
		// Validate the tool call structure
		if err := p.ValidateToolCall(ctx, processed[i]); err != nil {
			return nil, fmt.Errorf("invalid tool call %d: %w", i, err)
		}
	}
	
	return processed, nil
}

// GenerateToolCallID generates a new tool call ID with the format "call_<random>"
func (p *DefaultToolCallProcessor) GenerateToolCallID() string {
	return "call_" + generateRandomString(16)
}

// ValidateToolCall validates a tool call structure
func (p *DefaultToolCallProcessor) ValidateToolCall(ctx context.Context, toolCall vendors.ToolCall) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	if toolCall.ID == "" {
		return fmt.Errorf("tool call ID is required")
	}
	
	if toolCall.Type == "" {
		return fmt.Errorf("tool call type is required")
	}
	
	if toolCall.Type != "function" {
		return fmt.Errorf("unsupported tool call type: %s", toolCall.Type)
	}
	
	if toolCall.Function.Name == "" {
		return fmt.Errorf("function name is required")
	}
	
	// Arguments can be empty string, but should be valid JSON when present
	// We don't validate JSON here as it might be vendor-specific
	
	return nil
}

// generateRandomString generates a random hexadecimal string of specified length
func generateRandomString(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp if random generation fails
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
} 