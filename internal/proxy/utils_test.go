package proxy

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateRandomString(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{
			name:   "zero length",
			length: 0,
		},
		{
			name:   "small length",
			length: 5,
		},
		{
			name:   "standard length",
			length: 10,
		},
		{
			name:   "large length",
			length: 32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateRandomString(tt.length)

			// Check that result has expected length (hex encoding doubles the bytes)
			assert.Len(t, result, tt.length*2)

			// Check that result is valid hex
			for _, char := range result {
				assert.True(t, (char >= '0' && char <= '9') || (char >= 'a' && char <= 'f'),
					"Character %c is not valid hex", char)
			}

			// Test randomness - generate multiple and ensure they're different
			if tt.length > 0 {
				results := make(map[string]bool)
				for i := 0; i < 10; i++ {
					results[generateRandomString(tt.length)] = true
				}
				assert.Greater(t, len(results), 1, "Expected different random strings")
			}
		})
	}
}

func TestChatCompletionID(t *testing.T) {
	// Test format
	id := ChatCompletionID()
	assert.True(t, strings.HasPrefix(id, "chatcmpl-"), "ID should start with 'chatcmpl-'")

	// Test length (prefix + 20 hex chars)
	assert.Len(t, id, 29) // "chatcmpl-" (9) + 20 hex chars

	// Test uniqueness
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		ids[ChatCompletionID()] = true
	}
	assert.Equal(t, 100, len(ids), "All IDs should be unique")
}

func TestToolCallID(t *testing.T) {
	// Test format
	id := ToolCallID()
	assert.True(t, strings.HasPrefix(id, "call_"), "ID should start with 'call_'")

	// Test length (prefix + 32 hex chars)
	assert.Len(t, id, 37) // "call_" (5) + 32 hex chars

	// Test uniqueness
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		ids[ToolCallID()] = true
	}
	assert.Equal(t, 100, len(ids), "All IDs should be unique")
}

func TestSystemFingerprint(t *testing.T) {
	// Test format
	fp := SystemFingerprint()
	assert.True(t, strings.HasPrefix(fp, "fp_"), "Fingerprint should start with 'fp_'")

	// Test length (prefix + 18 hex chars)
	assert.Len(t, fp, 21) // "fp_" (3) + 18 hex chars

	// Test uniqueness
	fps := make(map[string]bool)
	for i := 0; i < 100; i++ {
		fps[SystemFingerprint()] = true
	}
	assert.Equal(t, 100, len(fps), "All fingerprints should be unique")
}

func TestRequestID(t *testing.T) {
	// Test format
	id := RequestID()
	assert.True(t, strings.HasPrefix(id, "req_"), "ID should start with 'req_'")

	// Test length (prefix + 32 hex chars)
	assert.Len(t, id, 36) // "req_" (4) + 32 hex chars

	// Test uniqueness
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		ids[RequestID()] = true
	}
	assert.Equal(t, 100, len(ids), "All IDs should be unique")
}

func TestAllIDsAreDifferent(t *testing.T) {
	// Ensure different ID types don't overlap
	allIDs := make(map[string]string)

	for i := 0; i < 10; i++ {
		chatID := ChatCompletionID()
		toolID := ToolCallID()
		fpID := SystemFingerprint()
		reqID := RequestID()

		require.NotContains(t, allIDs, chatID, "Chat ID collision")
		require.NotContains(t, allIDs, toolID, "Tool ID collision")
		require.NotContains(t, allIDs, fpID, "Fingerprint collision")
		require.NotContains(t, allIDs, reqID, "Request ID collision")

		allIDs[chatID] = "chat"
		allIDs[toolID] = "tool"
		allIDs[fpID] = "fp"
		allIDs[reqID] = "req"
	}
}
