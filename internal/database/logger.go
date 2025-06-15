package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/aashari/go-generative-api-router/internal/utils"
)

// RequestLogParams contains all parameters for logging a request
type RequestLogParams struct {
	RequestID        string
	Request          *http.Request
	StatusCode       int
	ResponseBody     string
	Duration         time.Duration
	OriginalModel    string
	SelectedVendor   string
	SelectedModel    string
	ErrorMessage     string
	ErrorType        string
	IsStreaming      bool
	StreamChunks     int
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// RequestLogger handles logging requests to MongoDB
type RequestLogger struct {
	repo        *Repository
	environment string
	version     string
	enabled     bool
}

// NewRequestLogger creates a new request logger
func NewRequestLogger() *RequestLogger {
	environment := os.Getenv("ENVIRONMENT")
	if environment == "" {
		environment = "development"
	}

	version := os.Getenv("VERSION")
	if version == "" {
		version = "unknown"
	}

	// Check if MongoDB URI is provided
	mongoURI := os.Getenv("MONGODB_URI")
	enabled := false
	var repo *Repository

	if mongoURI != "" {
		// Try to initialize repository and connect to MongoDB
		var err error
		repo, err = NewRepository()
		if err != nil {
			log.Printf("Warning: MongoDB URI provided but failed to initialize database repository: %v", err)
			enabled = false
		} else {
			// Test the connection
			if repo.conn != nil && repo.conn.IsConnected() {
				log.Printf("Database logging enabled: Successfully connected to MongoDB")
				enabled = true
			} else {
				log.Printf("Warning: MongoDB URI provided but connection failed")
				enabled = false
			}
		}
	} else {
		log.Printf("Database logging disabled: No MongoDB URI provided or using default local URI")
	}

	return &RequestLogger{
		repo:        repo,
		environment: environment,
		version:     version,
		enabled:     enabled,
	}
}

// LogRequest logs a request to MongoDB asynchronously
func (rl *RequestLogger) LogRequest(params *RequestLogParams) {
	if !rl.enabled || rl.repo == nil {
		return
	}

	// Log asynchronously to avoid blocking the request
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Extract request body if available
		var requestBody string
		if params.Request.Body != nil {
			// Note: In a real implementation, you'd need to capture the body
			// during the request processing, not here (as it's already consumed)
			requestBody = "" // Placeholder - body should be captured earlier
		}

		// Extract headers (excluding sensitive ones)
		headers := make(map[string]string)
		for key, values := range params.Request.Header {
			if !isSensitiveHeader(key) && len(values) > 0 {
				headers[key] = values[0]
			}
		}

		// Extract client IP
		clientIP := params.Request.Header.Get(utils.HeaderXForwardedFor)
		if clientIP == "" {
			clientIP = params.Request.Header.Get(utils.HeaderXRealIP)
		}
		if clientIP == "" {
			clientIP = params.Request.RemoteAddr
		}

		// Create request log
		requestLog := &RequestLog{
			RequestID: params.RequestID,
			Timestamp: time.Now(),
			Method:    params.Request.Method,
			Path:      params.Request.URL.Path,
			UserAgent: params.Request.Header.Get(utils.HeaderUserAgent),
			ClientIP:  clientIP,
			Headers:   headers,

			OriginalModel: params.OriginalModel,
			RequestBody:   requestBody,

			SelectedVendor:     params.SelectedVendor,
			SelectedModel:      params.SelectedModel,
			SelectedCredential: "", // Could be added if needed

			StatusCode:   params.StatusCode,
			ResponseBody: params.ResponseBody,
			DurationMs:   params.Duration.Milliseconds(),

			ErrorMessage: params.ErrorMessage,
			ErrorType:    params.ErrorType,

			IsStreaming:  params.IsStreaming,
			StreamChunks: params.StreamChunks,

			PromptTokens:     params.PromptTokens,
			CompletionTokens: params.CompletionTokens,
			TotalTokens:      params.TotalTokens,

			Environment: rl.environment,
			Version:     rl.version,
			Metadata:    make(map[string]interface{}),
		}

		// Insert the log
		if err := rl.repo.GetRequestLogRepository().InsertRequestLog(ctx, requestLog); err != nil {
			log.Printf("Warning: Failed to log request to database: %v", err)
		}
	}()
}

// LogSystemHealth logs system health metrics
func (rl *RequestLogger) LogSystemHealth(
	serviceStatus string,
	databaseStatus string,
	vendorStatuses map[string]string,
	requestsPerMinute float64,
	avgResponseTime float64,
	errorRate float64,
	circuitBreakerStates map[string]string,
) {
	if !rl.enabled || rl.repo == nil {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		health := &SystemHealth{
			Timestamp:   time.Now(),
			Environment: rl.environment,

			ServiceStatus:  serviceStatus,
			DatabaseStatus: databaseStatus,
			VendorStatuses: vendorStatuses,

			RequestsPerMinute: requestsPerMinute,
			AvgResponseTime:   avgResponseTime,
			ErrorRate:         errorRate,

			CircuitBreakerStates: circuitBreakerStates,

			Version:  rl.version,
			Metadata: make(map[string]interface{}),
		}

		if err := rl.repo.GetSystemHealthRepository().InsertSystemHealth(ctx, health); err != nil {
			log.Printf("Warning: Failed to log system health to database: %v", err)
		}
	}()
}

// LogUserSession logs or updates user session information
func (rl *RequestLogger) LogUserSession(
	sessionID string,
	userAgent string,
	clientIP string,
	modelsUsed []string,
	vendorsUsed []string,
) {
	if !rl.enabled || rl.repo == nil {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		session := &UserSession{
			SessionID:   sessionID,
			UserAgent:   userAgent,
			ClientIP:    clientIP,
			Environment: rl.environment,
			ModelsUsed:  modelsUsed,
			VendorsUsed: vendorsUsed,
			Metadata:    make(map[string]interface{}),
		}

		if err := rl.repo.GetUserSessionRepository().UpsertUserSession(ctx, session); err != nil {
			log.Printf("Warning: Failed to log user session to database: %v", err)
		}
	}()
}

// GenerateSessionID generates a session ID from user agent and IP
func (rl *RequestLogger) GenerateSessionID(userAgent, clientIP string) string {
	// Create a deterministic session ID based on user agent and IP
	// This allows tracking sessions without cookies
	sessionData := userAgent + "|" + clientIP
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(sessionData)).String()
}

// isSensitiveHeader checks if a header contains sensitive information
func isSensitiveHeader(header string) bool {
	sensitiveHeaders := []string{
		"authorization",
		"cookie",
		"x-api-key",
		"x-auth-token",
		"bearer",
	}

	headerLower := strings.ToLower(header)
	for _, sensitive := range sensitiveHeaders {
		if strings.Contains(headerLower, sensitive) {
			return true
		}
	}
	return false
}

// MaskSensitiveData masks sensitive information in request/response data
func (rl *RequestLogger) MaskSensitiveData(data string) string {
	if data == "" {
		return data
	}

	// Try to parse as JSON and mask sensitive fields
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(data), &jsonData); err != nil {
		// If not JSON, return as is (or implement other masking logic)
		return data
	}

	// Mask common sensitive fields
	sensitiveFields := []string{"api_key", "token", "password", "secret", "authorization"}
	for _, field := range sensitiveFields {
		if _, exists := jsonData[field]; exists {
			jsonData[field] = "***"
		}
	}

	// Convert back to JSON
	maskedData, err := json.Marshal(jsonData)
	if err != nil {
		return data // Return original if masking fails
	}

	return string(maskedData)
}

// LogGenerativeVendorRequest - REMOVED: Database logging functionality has been removed

// truncateMessageContent - REMOVED: No longer needed after removing database logging

// isVerboseLoggingEnabled checks if verbose logging is enabled
func isVerboseLoggingEnabled() bool {
	return os.Getenv("VERBOSE_LOGGING") == "true" || os.Getenv("LOG_LEVEL") == "debug"
}

// isDBLoggingEnabled checks if database logging is enabled
func isDBLoggingEnabled() bool {
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" || mongoURI == "mongodb://localhost:27017" {
		return false
	}

	// Try to get a connection to verify it's working
	conn, err := GetConnection()
	if err != nil {
		return false
	}

	return conn.IsConnected()
}

// getEnvironment returns the current environment
func getEnvironment() string {
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "development"
	}
	return env
}

// sanitizeUTF8Content - REMOVED: No longer needed after removing database logging

// sanitizeUTF8Interface recursively sanitizes UTF-8 content in interface{} values
func sanitizeUTF8Interface(data interface{}) interface{} {
	switch v := data.(type) {
	case string:
		return sanitizeUTF8String(v)
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			result[sanitizeUTF8String(key)] = sanitizeUTF8Interface(value)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, value := range v {
			result[i] = sanitizeUTF8Interface(value)
		}
		return result
	case map[interface{}]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			keyStr := fmt.Sprintf("%v", key)
			result[sanitizeUTF8String(keyStr)] = sanitizeUTF8Interface(value)
		}
		return result
	default:
		// For other types (int, float, bool, etc.), return as-is
		return v
	}
}

// sanitizeUTF8String removes or replaces invalid UTF-8 sequences in a string
func sanitizeUTF8String(s string) string {
	if utf8.ValidString(s) {
		return s
	}

	// If string contains invalid UTF-8, clean it
	var result strings.Builder
	result.Grow(len(s))

	for _, r := range s {
		if r == utf8.RuneError {
			// Replace invalid UTF-8 sequences with a replacement character
			result.WriteRune('\uFFFD')
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}
