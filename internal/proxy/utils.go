package proxy

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// generateRandomString generates a random hexadecimal string of specified length
func generateRandomString(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to the current timestamp if random generation fails
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// ChatCompletionID generates a chat completion ID with format: chatcmpl-{10chars}
func ChatCompletionID() string {
	return "chatcmpl-" + generateRandomString(10)
}

// ToolCallID generates a tool call ID with format: call_{16chars}
func ToolCallID() string {
	return "call_" + generateRandomString(16)
}

// SystemFingerprint generates a system fingerprint with format: fp_{9chars}
func SystemFingerprint() string {
	return "fp_" + generateRandomString(9)
}

// RequestID generates a request ID with format: req_{16chars}
func RequestID() string {
	return "req_" + generateRandomString(16)
}
