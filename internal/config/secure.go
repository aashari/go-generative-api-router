package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aashari/go-generative-api-router/internal/logger"
)

// SecureConfigManager handles encrypted credential management
type SecureConfigManager struct {
	encryptionKey []byte
	gcm           cipher.AEAD
}

// VaultInterface defines the interface for external secret management
type VaultInterface interface {
	GetSecret(path string) (string, error)
	SetSecret(path, value string) error
}

// NewSecureConfigManager creates a new secure configuration manager
func NewSecureConfigManager() (*SecureConfigManager, error) {
	// Get encryption key from environment or generate one
	keyStr := os.Getenv("CONFIG_ENCRYPTION_KEY")
	if keyStr == "" {
		logger.Warn("No CONFIG_ENCRYPTION_KEY found, using default key (not recommended for production)")
		keyStr = "default-key-change-in-production-32b" // 32 bytes
	}

	key := []byte(keyStr)
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be exactly 32 bytes, got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	return &SecureConfigManager{
		encryptionKey: key,
		gcm:           gcm,
	}, nil
}

// LoadCredentialsFromEnv loads credentials from environment variables
func LoadCredentialsFromEnv() ([]Credential, error) {
	var credentials []Credential

	// Check for OpenAI credentials
	if openaiKey := os.Getenv("OPENAI_API_KEY"); openaiKey != "" {
		credentials = append(credentials, Credential{
			Platform: "openai",
			Type:     "api-key",
			Value:    openaiKey,
		})
	}

	// Check for Gemini credentials
	if geminiKey := os.Getenv("GEMINI_API_KEY"); geminiKey != "" {
		credentials = append(credentials, Credential{
			Platform: "gemini",
			Type:     "api-key",
			Value:    geminiKey,
		})
	}

	// Check for Anthropic credentials
	if anthropicKey := os.Getenv("ANTHROPIC_API_KEY"); anthropicKey != "" {
		credentials = append(credentials, Credential{
			Platform: "anthropic",
			Type:     "api-key",
			Value:    anthropicKey,
		})
	}

	// Check for multiple credentials with numbered suffixes
	for i := 1; i <= 20; i++ {
		if openaiKey := os.Getenv(fmt.Sprintf("OPENAI_API_KEY_%d", i)); openaiKey != "" {
			credentials = append(credentials, Credential{
				Platform: "openai",
				Type:     "api-key",
				Value:    openaiKey,
			})
		}
		if geminiKey := os.Getenv(fmt.Sprintf("GEMINI_API_KEY_%d", i)); geminiKey != "" {
			credentials = append(credentials, Credential{
				Platform: "gemini",
				Type:     "api-key",
				Value:    geminiKey,
			})
		}
	}

	if len(credentials) == 0 {
		return nil, fmt.Errorf("no credentials found in environment variables")
	}

	logger.Info("Loaded credentials from environment variables",
		"count", len(credentials),
		"platforms", getUniquePlatforms(credentials))

	return credentials, nil
}

// LoadEncryptedCredentials loads and decrypts credentials from file
func (s *SecureConfigManager) LoadEncryptedCredentials(path string) ([]Credential, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read encrypted credentials file: %w", err)
	}

	// Check if file is already encrypted (base64 encoded)
	if isBase64Encoded(string(data)) {
		// Decrypt the data
		decryptedData, err := s.decrypt(data)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt credentials: %w", err)
		}
		data = decryptedData
	}

	var credentials []Credential
	if err := json.Unmarshal(data, &credentials); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credentials: %w", err)
	}

	return credentials, nil
}

// SaveEncryptedCredentials encrypts and saves credentials to file
func (s *SecureConfigManager) SaveEncryptedCredentials(path string, credentials []Credential) error {
	data, err := json.Marshal(credentials)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	encryptedData, err := s.encrypt(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt credentials: %w", err)
	}

	if err := os.WriteFile(path, encryptedData, 0600); err != nil {
		return fmt.Errorf("failed to write encrypted credentials: %w", err)
	}

	return nil
}

// encrypt encrypts data using AES-GCM
func (s *SecureConfigManager) encrypt(data []byte) ([]byte, error) {
	nonce := make([]byte, s.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := s.gcm.Seal(nonce, nonce, data, nil)
	
	// Encode to base64 for safe storage
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(ciphertext)))
	base64.StdEncoding.Encode(encoded, ciphertext)
	
	return encoded, nil
}

// decrypt decrypts base64 encoded data using AES-GCM
func (s *SecureConfigManager) decrypt(encodedData []byte) ([]byte, error) {
	// Decode from base64
	ciphertext := make([]byte, base64.StdEncoding.DecodedLen(len(encodedData)))
	n, err := base64.StdEncoding.Decode(ciphertext, encodedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}
	ciphertext = ciphertext[:n]

	nonceSize := s.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := s.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// isBase64Encoded checks if a string is base64 encoded
func isBase64Encoded(s string) bool {
	s = strings.TrimSpace(s)
	if len(s)%4 != 0 {
		return false
	}
	_, err := base64.StdEncoding.DecodeString(s)
	return err == nil
}

// getUniquePlatforms extracts unique platforms from credentials
func getUniquePlatforms(credentials []Credential) []string {
	platformMap := make(map[string]bool)
	for _, cred := range credentials {
		platformMap[cred.Platform] = true
	}

	platforms := make([]string, 0, len(platformMap))
	for platform := range platformMap {
		platforms = append(platforms, platform)
	}
	return platforms
}

// LoadCredentialsSecurely attempts to load credentials using the most secure method available
func LoadCredentialsSecurely() ([]Credential, error) {
	// Priority 1: Existing configuration file (current working method)
	if creds, err := LoadCredentials("configs/credentials.json"); err == nil {
		logger.Info("Loaded credentials from configuration file")
		return creds, nil
	}

	// Priority 2: Environment variables (only if file loading fails)
	if creds, err := LoadCredentialsFromEnv(); err == nil && len(creds) > 0 {
		logger.Info("Loaded credentials from environment variables (secure)")
		return creds, nil
	}

	// Priority 3: Encrypted file (future enhancement)
	secureManager, err := NewSecureConfigManager()
	if err != nil {
		logger.Warn("Failed to initialize secure config manager", "error", err)
	} else {
		if creds, err := secureManager.LoadEncryptedCredentials("configs/credentials.json"); err == nil {
			logger.Info("Loaded credentials from encrypted file")
			return creds, nil
		}
	}

	return nil, fmt.Errorf("no credentials could be loaded from any source")
} 