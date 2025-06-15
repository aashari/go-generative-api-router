package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIDGenerator(t *testing.T) {
	generator := NewIDGenerator()

	t.Run("GenerateRequestID", func(t *testing.T) {
		id1 := generator.GenerateRequestID()
		id2 := generator.GenerateRequestID()

		// Check format (16 hex characters)
		assert.Len(t, id1, 16)
		assert.Len(t, id2, 16)
		assert.Regexp(t, "^[0-9a-f]{16}$", id1)
		assert.Regexp(t, "^[0-9a-f]{16}$", id2)

		// IDs should be different
		assert.NotEqual(t, id1, id2)
	})

	t.Run("GenerateCorrelationID", func(t *testing.T) {
		id1 := generator.GenerateCorrelationID()
		id2 := generator.GenerateCorrelationID()

		// Check UUID format
		assert.Regexp(t, "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$", id1)
		assert.Regexp(t, "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$", id2)

		// IDs should be different
		assert.NotEqual(t, id1, id2)
	})

	t.Run("GenerateChatCompletionID", func(t *testing.T) {
		id1 := generator.GenerateChatCompletionID()
		id2 := generator.GenerateChatCompletionID()

		// Check format (chatcmpl- + 32 hex characters)
		assert.True(t, strings.HasPrefix(id1, "chatcmpl-"))
		assert.True(t, strings.HasPrefix(id2, "chatcmpl-"))
		assert.Len(t, id1, 41) // "chatcmpl-" (9 chars) + 32 hex chars
		assert.Len(t, id2, 41)

		// Extract hex part and validate
		hexPart1 := strings.TrimPrefix(id1, "chatcmpl-")
		hexPart2 := strings.TrimPrefix(id2, "chatcmpl-")
		assert.Regexp(t, "^[0-9a-f]{32}$", hexPart1)
		assert.Regexp(t, "^[0-9a-f]{32}$", hexPart2)

		// IDs should be different
		assert.NotEqual(t, id1, id2)
	})

	t.Run("GenerateToolCallID", func(t *testing.T) {
		id1 := generator.GenerateToolCallID()
		id2 := generator.GenerateToolCallID()

		// Check format (call_ + 24 hex characters)
		assert.True(t, strings.HasPrefix(id1, "call_"))
		assert.True(t, strings.HasPrefix(id2, "call_"))
		assert.Len(t, id1, 29) // "call_" (5 chars) + 24 hex chars
		assert.Len(t, id2, 29)

		// Extract hex part and validate
		hexPart1 := strings.TrimPrefix(id1, "call_")
		hexPart2 := strings.TrimPrefix(id2, "call_")
		assert.Regexp(t, "^[0-9a-f]{24}$", hexPart1)
		assert.Regexp(t, "^[0-9a-f]{24}$", hexPart2)

		// IDs should be different
		assert.NotEqual(t, id1, id2)
	})

	t.Run("GenerateSystemFingerprint", func(t *testing.T) {
		id1 := generator.GenerateSystemFingerprint()
		id2 := generator.GenerateSystemFingerprint()

		// Check format (fp_ + 12 hex characters)
		assert.True(t, strings.HasPrefix(id1, "fp_"))
		assert.True(t, strings.HasPrefix(id2, "fp_"))
		assert.Len(t, id1, 15) // "fp_" (3 chars) + 12 hex chars
		assert.Len(t, id2, 15)

		// Extract hex part and validate
		hexPart1 := strings.TrimPrefix(id1, "fp_")
		hexPart2 := strings.TrimPrefix(id2, "fp_")
		assert.Regexp(t, "^[0-9a-f]{12}$", hexPart1)
		assert.Regexp(t, "^[0-9a-f]{12}$", hexPart2)

		// IDs should be different
		assert.NotEqual(t, id1, id2)
	})

	t.Run("GenerateShortID", func(t *testing.T) {
		id1 := generator.GenerateShortID()
		id2 := generator.GenerateShortID()

		// Check format (8 hex characters)
		assert.Len(t, id1, 8)
		assert.Len(t, id2, 8)
		assert.Regexp(t, "^[0-9a-f]{8}$", id1)
		assert.Regexp(t, "^[0-9a-f]{8}$", id2)

		// IDs should be different
		assert.NotEqual(t, id1, id2)
	})
}

func TestGlobalFunctions(t *testing.T) {
	t.Run("GenerateRequestID", func(t *testing.T) {
		id := GenerateRequestID()
		assert.Len(t, id, 16)
		assert.Regexp(t, "^[0-9a-f]{16}$", id)
	})

	t.Run("GenerateCorrelationID", func(t *testing.T) {
		id := GenerateCorrelationID()
		assert.Regexp(t, "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$", id)
	})

	t.Run("GenerateChatCompletionID", func(t *testing.T) {
		id := GenerateChatCompletionID()
		assert.True(t, strings.HasPrefix(id, "chatcmpl-"))
		assert.Len(t, id, 41)
	})

	t.Run("GenerateToolCallID", func(t *testing.T) {
		id := GenerateToolCallID()
		assert.True(t, strings.HasPrefix(id, "call_"))
		assert.Len(t, id, 29)
	})

	t.Run("GenerateSystemFingerprint", func(t *testing.T) {
		id := GenerateSystemFingerprint()
		assert.True(t, strings.HasPrefix(id, "fp_"))
		assert.Len(t, id, 15)
	})

	t.Run("GenerateShortID", func(t *testing.T) {
		id := GenerateShortID()
		assert.Len(t, id, 8)
		assert.Regexp(t, "^[0-9a-f]{8}$", id)
	})

	t.Run("GenerateTimestampID", func(t *testing.T) {
		id1 := GenerateTimestampID()
		id2 := GenerateTimestampID()

		// Check format: timestamp_hexstring
		parts1 := strings.Split(id1, "_")
		parts2 := strings.Split(id2, "_")
		assert.Len(t, parts1, 2)
		assert.Len(t, parts2, 2)

		// Check timestamp part is numeric
		assert.Regexp(t, "^[0-9]+$", parts1[0])
		assert.Regexp(t, "^[0-9]+$", parts2[0])

		// Check random part is 8 hex chars
		assert.Len(t, parts1[1], 8)
		assert.Len(t, parts2[1], 8)
		assert.Regexp(t, "^[0-9a-f]{8}$", parts1[1])
		assert.Regexp(t, "^[0-9a-f]{8}$", parts2[1])

		// IDs should be different
		assert.NotEqual(t, id1, id2)
	})
}

func TestSecureFunctions(t *testing.T) {
	t.Run("GenerateSecureToken", func(t *testing.T) {
		token, err := GenerateSecureToken(16)
		require.NoError(t, err)

		// Should be 32 hex characters (16 bytes * 2)
		assert.Len(t, token, 32)
		assert.Regexp(t, "^[0-9a-f]{32}$", token)

		// Generate another and ensure they're different
		token2, err := GenerateSecureToken(16)
		require.NoError(t, err)
		assert.NotEqual(t, token, token2)
	})

	t.Run("GenerateSecureInt", func(t *testing.T) {
		// Test valid range
		n, err := GenerateSecureInt(100)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, n, int64(0))
		assert.Less(t, n, int64(100))

		// Test edge case: max = 1
		n, err = GenerateSecureInt(1)
		require.NoError(t, err)
		assert.Equal(t, int64(0), n)

		// Test invalid max
		_, err = GenerateSecureInt(0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max must be positive")

		_, err = GenerateSecureInt(-1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max must be positive")
	})

	t.Run("GenerateNonce", func(t *testing.T) {
		nonce, err := GenerateNonce(12)
		require.NoError(t, err)
		assert.Len(t, nonce, 12)

		// Generate another and ensure they're different
		nonce2, err := GenerateNonce(12)
		require.NoError(t, err)
		assert.NotEqual(t, nonce, nonce2)
	})
}

func TestIDUniqueness(t *testing.T) {
	// Test that multiple calls produce unique IDs
	t.Run("bulk uniqueness test", func(t *testing.T) {
		generator := NewIDGenerator()
		iterations := 1000

		// Test request IDs
		requestIDs := make(map[string]bool)
		for i := 0; i < iterations; i++ {
			id := generator.GenerateRequestID()
			assert.False(t, requestIDs[id], "duplicate request ID: %s", id)
			requestIDs[id] = true
		}

		// Test chat completion IDs
		chatIDs := make(map[string]bool)
		for i := 0; i < iterations; i++ {
			id := generator.GenerateChatCompletionID()
			assert.False(t, chatIDs[id], "duplicate chat completion ID: %s", id)
			chatIDs[id] = true
		}

		// Test tool call IDs
		toolIDs := make(map[string]bool)
		for i := 0; i < iterations; i++ {
			id := generator.GenerateToolCallID()
			assert.False(t, toolIDs[id], "duplicate tool call ID: %s", id)
			toolIDs[id] = true
		}
	})
}
