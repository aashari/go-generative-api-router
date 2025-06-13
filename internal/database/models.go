package database

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PayloadData represents the request/response payload structure
type PayloadData struct {
	Request  interface{} `bson:"request" json:"request"`   // Full request object sent to vendor
	Response interface{} `bson:"response" json:"response"` // Full response object from vendor
}

// GenerativeUsage represents complete vendor request/response logging
// Stores full request and response data without obfuscation or truncation
type GenerativeUsage struct {
	ID primitive.ObjectID `bson:"_id,omitempty" json:"id"`

	// Credential as full object
	Credential interface{} `bson:"credential" json:"credential"` // Full credential object

	// Model as full object
	Model interface{} `bson:"model" json:"model"` // Full model object with config

	// Payload containing request and response
	Payload PayloadData `bson:"payload" json:"payload"`

	// Timestamps
	RequestedAt time.Time `bson:"requested_at" json:"requested_at"` // Start time of the request
	RespondedAt time.Time `bson:"responded_at" json:"responded_at"` // End time when vendor responded
	CreatedAt   time.Time `bson:"created_at" json:"created_at"`     // When this record was created

	// Additional metadata for context (optional, for backward compatibility)
	Vendor      string `bson:"vendor,omitempty" json:"vendor,omitempty"`           // Vendor name (openai, gemini, etc.)
	RequestID   string `bson:"request_id,omitempty" json:"request_id,omitempty"`   // Request ID for correlation
	StatusCode  int    `bson:"status_code,omitempty" json:"status_code,omitempty"` // HTTP status code
	Environment string `bson:"environment,omitempty" json:"environment,omitempty"` // Environment (dev, prod, etc.)
}

// GenerativeVendorLog is deprecated - use GenerativeUsage instead
// Kept for backward compatibility
type GenerativeVendorLog = GenerativeUsage

// RequestLog represents a logged API request for analytics and monitoring
type RequestLog struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	RequestID string             `bson:"request_id" json:"request_id"`
	Timestamp time.Time          `bson:"timestamp" json:"timestamp"`

	// Request details
	Method    string            `bson:"method" json:"method"`
	Path      string            `bson:"path" json:"path"`
	UserAgent string            `bson:"user_agent,omitempty" json:"user_agent,omitempty"`
	ClientIP  string            `bson:"client_ip,omitempty" json:"client_ip,omitempty"`
	Headers   map[string]string `bson:"headers,omitempty" json:"headers,omitempty"`

	// API request details
	OriginalModel string `bson:"original_model,omitempty" json:"original_model,omitempty"`
	RequestBody   string `bson:"request_body,omitempty" json:"request_body,omitempty"` // JSON string

	// Vendor routing details
	SelectedVendor     string `bson:"selected_vendor,omitempty" json:"selected_vendor,omitempty"`
	SelectedModel      string `bson:"selected_model,omitempty" json:"selected_model,omitempty"`
	SelectedCredential string `bson:"selected_credential,omitempty" json:"selected_credential,omitempty"`

	// Response details
	StatusCode   int    `bson:"status_code" json:"status_code"`
	ResponseBody string `bson:"response_body,omitempty" json:"response_body,omitempty"` // JSON string
	DurationMs   int64  `bson:"duration_ms" json:"duration_ms"`

	// Error details (if any)
	ErrorMessage string `bson:"error_message,omitempty" json:"error_message,omitempty"`
	ErrorType    string `bson:"error_type,omitempty" json:"error_type,omitempty"`

	// Streaming details
	IsStreaming  bool `bson:"is_streaming" json:"is_streaming"`
	StreamChunks int  `bson:"stream_chunks,omitempty" json:"stream_chunks,omitempty"`

	// Token usage (if available from vendor response)
	PromptTokens     int `bson:"prompt_tokens,omitempty" json:"prompt_tokens,omitempty"`
	CompletionTokens int `bson:"completion_tokens,omitempty" json:"completion_tokens,omitempty"`
	TotalTokens      int `bson:"total_tokens,omitempty" json:"total_tokens,omitempty"`

	// Metadata
	Environment string                 `bson:"environment" json:"environment"`
	Version     string                 `bson:"version,omitempty" json:"version,omitempty"`
	Metadata    map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`

	// Timestamps
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

// VendorMetrics represents aggregated metrics for vendor performance
type VendorMetrics struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Vendor      string             `bson:"vendor" json:"vendor"`
	Model       string             `bson:"model" json:"model"`
	Environment string             `bson:"environment" json:"environment"`

	// Time period for these metrics
	PeriodStart time.Time `bson:"period_start" json:"period_start"`
	PeriodEnd   time.Time `bson:"period_end" json:"period_end"`
	PeriodType  string    `bson:"period_type" json:"period_type"` // "hour", "day", "week", "month"

	// Request metrics
	TotalRequests  int64 `bson:"total_requests" json:"total_requests"`
	SuccessfulReqs int64 `bson:"successful_requests" json:"successful_requests"`
	FailedRequests int64 `bson:"failed_requests" json:"failed_requests"`

	// Performance metrics
	AvgDurationMs    float64 `bson:"avg_duration_ms" json:"avg_duration_ms"`
	MinDurationMs    int64   `bson:"min_duration_ms" json:"min_duration_ms"`
	MaxDurationMs    int64   `bson:"max_duration_ms" json:"max_duration_ms"`
	MedianDurationMs float64 `bson:"median_duration_ms" json:"median_duration_ms"`

	// Error metrics
	ErrorRate        float64          `bson:"error_rate" json:"error_rate"` // Percentage
	ErrorBreakdown   map[string]int64 `bson:"error_breakdown,omitempty" json:"error_breakdown,omitempty"`
	StatusCodeCounts map[string]int64 `bson:"status_code_counts" json:"status_code_counts"`

	// Token usage metrics
	TotalPromptTokens     int64   `bson:"total_prompt_tokens,omitempty" json:"total_prompt_tokens,omitempty"`
	TotalCompletionTokens int64   `bson:"total_completion_tokens,omitempty" json:"total_completion_tokens,omitempty"`
	TotalTokens           int64   `bson:"total_tokens,omitempty" json:"total_tokens,omitempty"`
	AvgTokensPerRequest   float64 `bson:"avg_tokens_per_request,omitempty" json:"avg_tokens_per_request,omitempty"`

	// Streaming metrics
	StreamingRequests int64   `bson:"streaming_requests" json:"streaming_requests"`
	AvgStreamChunks   float64 `bson:"avg_stream_chunks,omitempty" json:"avg_stream_chunks,omitempty"`

	// Timestamps
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

// SystemHealth represents system health metrics
type SystemHealth struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Timestamp   time.Time          `bson:"timestamp" json:"timestamp"`
	Environment string             `bson:"environment" json:"environment"`

	// Service health
	ServiceStatus  string            `bson:"service_status" json:"service_status"` // "healthy", "degraded", "unhealthy"
	DatabaseStatus string            `bson:"database_status" json:"database_status"`
	VendorStatuses map[string]string `bson:"vendor_statuses" json:"vendor_statuses"`

	// Performance metrics
	RequestsPerMinute float64 `bson:"requests_per_minute" json:"requests_per_minute"`
	AvgResponseTime   float64 `bson:"avg_response_time" json:"avg_response_time"`
	ErrorRate         float64 `bson:"error_rate" json:"error_rate"`

	// Resource usage
	MemoryUsageMB   float64 `bson:"memory_usage_mb,omitempty" json:"memory_usage_mb,omitempty"`
	CPUUsagePercent float64 `bson:"cpu_usage_percent,omitempty" json:"cpu_usage_percent,omitempty"`

	// Circuit breaker status
	CircuitBreakerStates map[string]string `bson:"circuit_breaker_states,omitempty" json:"circuit_breaker_states,omitempty"`

	// Metadata
	Version  string                 `bson:"version,omitempty" json:"version,omitempty"`
	Metadata map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`

	// Timestamps
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}

// UserSession represents user session tracking (if needed for analytics)
type UserSession struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	SessionID   string             `bson:"session_id" json:"session_id"`
	UserAgent   string             `bson:"user_agent" json:"user_agent"`
	ClientIP    string             `bson:"client_ip" json:"client_ip"`
	Environment string             `bson:"environment" json:"environment"`

	// Session details
	FirstSeen    time.Time `bson:"first_seen" json:"first_seen"`
	LastSeen     time.Time `bson:"last_seen" json:"last_seen"`
	RequestCount int64     `bson:"request_count" json:"request_count"`

	// Usage patterns
	ModelsUsed  []string `bson:"models_used" json:"models_used"`
	VendorsUsed []string `bson:"vendors_used" json:"vendors_used"`
	TotalTokens int64    `bson:"total_tokens,omitempty" json:"total_tokens,omitempty"`

	// Metadata
	Metadata map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`

	// Timestamps
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}
