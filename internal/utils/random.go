package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	mathRand "math/rand"
	"time"

	"github.com/google/uuid"
)

// IDGenerator provides centralized ID generation functionality
type IDGenerator struct {
	random *mathRand.Rand
}

// NewIDGenerator creates a new ID generator
func NewIDGenerator() *IDGenerator {
	return &IDGenerator{
		// #nosec G404 - math/rand is used for non-security-critical ID generation
		random: mathRand.New(mathRand.NewSource(time.Now().UnixNano())),
	}
}

// GenerateRequestID generates a unique request ID (16 hex characters)
func (g *IDGenerator) GenerateRequestID() string {
	return g.generateHex(8) // 8 bytes = 16 hex characters
}

// GenerateCorrelationID generates a UUID for correlation tracking
func (g *IDGenerator) GenerateCorrelationID() string {
	return uuid.New().String()
}

// GenerateChatCompletionID generates an OpenAI-compatible chat completion ID
func (g *IDGenerator) GenerateChatCompletionID() string {
	return fmt.Sprintf("chatcmpl-%s", g.generateHex(16)) // 16 bytes = 32 hex characters
}

// GenerateToolCallID generates an OpenAI-compatible tool call ID
func (g *IDGenerator) GenerateToolCallID() string {
	return fmt.Sprintf("call_%s", g.generateHex(12)) // 12 bytes = 24 hex characters
}

// GenerateSystemFingerprint generates a system fingerprint
func (g *IDGenerator) GenerateSystemFingerprint() string {
	return fmt.Sprintf("fp_%s", g.generateHex(6)) // 6 bytes = 12 hex characters
}

// GenerateShortID generates a short ID for internal use
func (g *IDGenerator) GenerateShortID() string {
	return g.generateHex(4) // 4 bytes = 8 hex characters
}

// generateHex generates a random hex string of specified byte length
func (g *IDGenerator) generateHex(byteLength int) string {
	bytes := make([]byte, byteLength)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to math/rand if crypto/rand fails
		for i := range bytes {
			bytes[i] = byte(g.random.Intn(256))
		}
	}
	return hex.EncodeToString(bytes)
}

// generateUUID generates a UUID string
func (g *IDGenerator) generateUUID() string {
	return uuid.New().String()
}

// Global ID generator instance
var globalIDGenerator = NewIDGenerator()

// Convenience functions using the global generator

// GenerateRequestID generates a unique request ID using the global generator
func GenerateRequestID() string {
	return globalIDGenerator.GenerateRequestID()
}

// GenerateCorrelationID generates a correlation ID using the global generator
func GenerateCorrelationID() string {
	return globalIDGenerator.GenerateCorrelationID()
}

// GenerateChatCompletionID generates a chat completion ID using the global generator
func GenerateChatCompletionID() string {
	return globalIDGenerator.GenerateChatCompletionID()
}

// GenerateToolCallID generates a tool call ID using the global generator
func GenerateToolCallID() string {
	return globalIDGenerator.GenerateToolCallID()
}

// GenerateSystemFingerprint generates a system fingerprint using the global generator
func GenerateSystemFingerprint() string {
	return globalIDGenerator.GenerateSystemFingerprint()
}

// GenerateShortID generates a short ID using the global generator
func GenerateShortID() string {
	return globalIDGenerator.GenerateShortID()
}

// GenerateSecureToken generates a cryptographically secure token
func GenerateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secure token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// GenerateSecureInt generates a cryptographically secure integer in range [0, max)
func GenerateSecureInt(max int64) (int64, error) {
	if max <= 0 {
		return 0, fmt.Errorf("max must be positive")
	}

	bigMax := big.NewInt(max)
	n, err := rand.Int(rand.Reader, bigMax)
	if err != nil {
		return 0, fmt.Errorf("failed to generate secure int: %w", err)
	}

	return n.Int64(), nil
}

// GenerateNonce generates a cryptographic nonce
func GenerateNonce(length int) ([]byte, error) {
	nonce := make([]byte, length)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	return nonce, nil
}

// GenerateTimestampID generates a timestamp-based ID for ordering
func GenerateTimestampID() string {
	timestamp := time.Now().UnixNano()
	randomPart := globalIDGenerator.generateHex(4)
	return fmt.Sprintf("%d_%s", timestamp, randomPart)
}
