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

	"github.com/google/uuid"
)

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
func (rl *RequestLogger) LogRequest(
	requestID string,
	r *http.Request,
	statusCode int,
	responseBody string,
	duration time.Duration,
	originalModel string,
	selectedVendor string,
	selectedModel string,
	errorMessage string,
	errorType string,
	isStreaming bool,
	streamChunks int,
	promptTokens int,
	completionTokens int,
	totalTokens int,
) {
	if !rl.enabled || rl.repo == nil {
		return
	}

	// Log asynchronously to avoid blocking the request
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Extract request body if available
		var requestBody string
		if r.Body != nil {
			// Note: In a real implementation, you'd need to capture the body
			// during the request processing, not here (as it's already consumed)
			requestBody = "" // Placeholder - body should be captured earlier
		}

		// Extract headers (excluding sensitive ones)
		headers := make(map[string]string)
		for key, values := range r.Header {
			if !isSensitiveHeader(key) && len(values) > 0 {
				headers[key] = values[0]
			}
		}

		// Extract client IP
		clientIP := r.Header.Get("X-Forwarded-For")
		if clientIP == "" {
			clientIP = r.Header.Get("X-Real-IP")
		}
		if clientIP == "" {
			clientIP = r.RemoteAddr
		}

		// Create request log
		requestLog := &RequestLog{
			RequestID: requestID,
			Timestamp: time.Now(),
			Method:    r.Method,
			Path:      r.URL.Path,
			UserAgent: r.Header.Get("User-Agent"),
			ClientIP:  clientIP,
			Headers:   headers,

			OriginalModel: originalModel,
			RequestBody:   requestBody,

			SelectedVendor:     selectedVendor,
			SelectedModel:      selectedModel,
			SelectedCredential: "", // Could be added if needed

			StatusCode:   statusCode,
			ResponseBody: responseBody,
			DurationMs:   duration.Milliseconds(),

			ErrorMessage: errorMessage,
			ErrorType:    errorType,

			IsStreaming:  isStreaming,
			StreamChunks: streamChunks,

			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      totalTokens,

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

// LogGenerativeVendorRequest logs complete vendor request and response data
// This function stores full request/response payloads without obfuscation or truncation
func LogGenerativeVendorRequest(ctx context.Context, vendorLog GenerativeUsage) {
	// Skip if database logging is disabled
	if !isDBLoggingEnabled() {
		return
	}

	// Use goroutine for non-blocking logging
	go func() {
		// Create a new context with timeout for the database operation
		dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Get repository
		repo, err := NewRepository()
		if err != nil {
			// Log error but don't fail the request
			fmt.Printf("Failed to create repository for vendor logging: %v\n", err)
			return
		}

		// Set environment if not already set
		if vendorLog.Environment == "" {
			vendorLog.Environment = getEnvironment()
		}

		// Create the log entry
		err = repo.CreateGenerativeVendorLog(dbCtx, &vendorLog)
		if err != nil {
			// Log error but don't fail the request
			fmt.Printf("Failed to log generative vendor request: %v\n", err)
			return
		}

		// Optional: Log success for debugging
		if isVerboseLoggingEnabled() {
			fmt.Printf("Successfully logged vendor request: %s to %s model %s\n",
				vendorLog.RequestID, vendorLog.Vendor, vendorLog.Model)
		}
	}()
}

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
