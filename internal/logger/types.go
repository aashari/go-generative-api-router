package logger

import "time"

// LogLevel represents standardized log levels
type LogLevel string

const (
	LevelDEBUG LogLevel = "DEBUG"
	LevelINFO  LogLevel = "INFO"
	LevelWARN  LogLevel = "WARN"
	LevelERROR LogLevel = "ERROR"
)

// LogEntry represents the standardized log structure matching BrainyBuddy API
type LogEntry struct {
	Environment string                 `json:"environment"`
	Level       LogLevel               `json:"level"`
	Message     string                 `json:"message"`
	Timestamp   string                 `json:"timestamp"`
	Version     string                 `json:"version"`
	Service     string                 `json:"service"`
	Component   string                 `json:"component,omitempty"`
	Stage       string                 `json:"stage,omitempty"`
	Request     *RequestContext        `json:"request,omitempty"`
	Response    *ResponseContext       `json:"response,omitempty"`
	Attributes  map[string]interface{} `json:"attributes,omitempty"`
	Error       *ErrorContext          `json:"error,omitempty"`
}

// RequestContext contains all request-related information
type RequestContext struct {
	RequestID     string            `json:"request_id,omitempty"`
	CorrelationID string            `json:"correlation_id,omitempty"`
	Method        string            `json:"method,omitempty"`
	Endpoint      string            `json:"endpoint,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	Body          interface{}       `json:"body,omitempty"`
	ClientIP      string            `json:"client_ip,omitempty"`
	UserAgent     string            `json:"user_agent,omitempty"`
}

// ResponseContext contains all response-related information
type ResponseContext struct {
	RequestID     string            `json:"request_id,omitempty"`
	CorrelationID string            `json:"correlation_id,omitempty"`
	StatusCode    int               `json:"status_code,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	Body          interface{}       `json:"body,omitempty"`
	DurationMs    int64             `json:"duration_ms,omitempty"`
}

// ErrorContext contains standardized error information
type ErrorContext struct {
	Message    string                 `json:"message"`
	Type       string                 `json:"type"`
	Stacktrace string                 `json:"stacktrace,omitempty"`
	Code       string                 `json:"code,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
}

// SerializableValue handles special Go types that don't exist in TypeScript/Node.js
func SerializeValue(val interface{}) interface{} {
	switch v := val.(type) {
	case time.Time:
		return v.Format(time.RFC3339Nano)
	case time.Duration:
		return v.Milliseconds()
	default:
		return val
	}
}
