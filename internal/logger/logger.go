package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aashari/go-generative-api-router/internal/utils"
)

// ContextKey defines a type for context keys to avoid collisions.
type ContextKey string

// Context keys used throughout the application for structured logging.
const (
	RequestIDKey     ContextKey = "request_id"
	CorrelationIDKey ContextKey = "correlation_id"
	ComponentKey     ContextKey = "component"
	StageKey         ContextKey = "stage"
)

var (
	globalLogger *slog.Logger
	once         sync.Once
	version      = "unknown"
	serviceName  = "generative-api-router"
	environment  = "development"
)

// Init initializes the global logger with the specified configuration.
// It is safe to call Init multiple times.
func Init(writer io.Writer, level slog.Level, appVersion, appServiceName, appEnvironment string) {
	once.Do(func() {
		if appVersion != "" {
			version = appVersion
		}
		if appServiceName != "" {
			serviceName = appServiceName
		}
		if appEnvironment != "" {
			environment = appEnvironment
		}

		handler := NewStructuredJSONHandler(writer, &slog.HandlerOptions{
			Level: level,
		})
		globalLogger = slog.New(handler)
	})
}

// Log performs logging with a structured format.
func Log(ctx context.Context, level slog.Level, msg string, attrs ...any) {
	if globalLogger == nil {
		Init(os.Stdout, slog.LevelInfo, os.Getenv("VERSION"), os.Getenv("SERVICE_NAME"), os.Getenv("ENVIRONMENT"))
	}
	globalLogger.Log(ctx, level, msg, attrs...)
}

// StructuredJSONHandler is a custom slog.Handler that writes logs in a specific JSON format.
type StructuredJSONHandler struct {
	handler slog.Handler
	writer  io.Writer
}

// NewStructuredJSONHandler creates a new handler.
func NewStructuredJSONHandler(w io.Writer, opts *slog.HandlerOptions) *StructuredJSONHandler {
	return &StructuredJSONHandler{
		handler: slog.NewJSONHandler(w, opts),
		writer:  w,
	}
}

// Enabled reports whether the handler handles records at the given level.
func (h *StructuredJSONHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

// WithAttrs returns a new handler with the given attributes.
func (h *StructuredJSONHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &StructuredJSONHandler{handler: h.handler.WithAttrs(attrs), writer: h.writer}
}

// WithGroup returns a new handler with the given group name.
func (h *StructuredJSONHandler) WithGroup(name string) slog.Handler {
	return &StructuredJSONHandler{handler: h.handler.WithGroup(name), writer: h.writer}
}

// Handle processes the given log record and writes it to the output.
func (h *StructuredJSONHandler) Handle(ctx context.Context, r slog.Record) error {
	// Create standardized log entry
	logEntry := &LogEntry{
		Timestamp:   r.Time.UTC().Format(time.RFC3339Nano),
		Level:       LogLevel(strings.ToUpper(r.Level.String())),
		Message:     r.Message,
		Service:     serviceName,
		Environment: environment,
		Version:     version,
	}

	// Extract context values
	if component := ctx.Value(ComponentKey); component != nil {
		if comp, ok := component.(string); ok {
			logEntry.Component = comp
		}
	}

	if stage := ctx.Value(StageKey); stage != nil {
		if stg, ok := stage.(string); ok {
			logEntry.Stage = stg
		}
	}

	// Initialize request context from tracking IDs
	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		if logEntry.Request == nil {
			logEntry.Request = &RequestContext{}
		}
		logEntry.Request.RequestID = requestID.(string)
	}

	if correlationID := ctx.Value(CorrelationIDKey); correlationID != nil {
		if logEntry.Request == nil {
			logEntry.Request = &RequestContext{}
		}
		logEntry.Request.CorrelationID = correlationID.(string)
	}

	// Process attributes and categorize them
	attributes := make(map[string]interface{})
	var requestData, responseData map[string]interface{}
	var errorData error

	r.Attrs(func(a slog.Attr) bool {
		val := a.Value.Any()

		switch a.Key {
		case "error":
			if err, ok := val.(error); ok {
				errorData = err
			}
		case "request":
			if req, ok := val.(map[string]interface{}); ok {
				requestData = req
			}
		case "response":
			if resp, ok := val.(map[string]interface{}); ok {
				responseData = resp
			}
		default:
			attributes[a.Key] = SerializeValue(val)
		}
		return true
	})

	// Handle error serialization
	if errorData != nil {
		logEntry.Error = &ErrorContext{
			Message: errorData.Error(),
			Type:    fmt.Sprintf("%T", errorData),
		}

		// Extract stack trace if available
		if stackTracer, ok := errorData.(interface{ StackTrace() string }); ok {
			logEntry.Error.Stacktrace = stackTracer.StackTrace()
		}
	}

	// Merge request data
	if requestData != nil {
		if logEntry.Request == nil {
			logEntry.Request = &RequestContext{}
		}
		mergeRequestData(logEntry.Request, requestData)
	}

	// Handle response data
	if responseData != nil {
		logEntry.Response = &ResponseContext{}
		mergeResponseData(logEntry.Response, responseData)

		// Copy tracking IDs to response
		if logEntry.Request != nil {
			logEntry.Response.RequestID = logEntry.Request.RequestID
			logEntry.Response.CorrelationID = logEntry.Request.CorrelationID
		}
	}

	// Set remaining attributes
	if len(attributes) > 0 {
		logEntry.Attributes = utils.TruncateBase64InData(attributes).(map[string]interface{})
	}

	// Marshal and write
	b, err := json.Marshal(logEntry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	_, err = h.writer.Write(append(b, '\n'))
	return err
}

// Convenience functions
func Info(ctx context.Context, msg string, attrs ...any) {
	Log(ctx, slog.LevelInfo, msg, attrs...)
}

func Debug(ctx context.Context, msg string, attrs ...any) {
	Log(ctx, slog.LevelDebug, msg, attrs...)
}

func Warn(ctx context.Context, msg string, attrs ...any) {
	Log(ctx, slog.LevelWarn, msg, attrs...)
}

func Error(ctx context.Context, msg string, err error, attrs ...any) {
	args := append(attrs, "error", err)
	Log(ctx, slog.LevelError, msg, args...)
}

// WithComponent returns a new context with the component key set.
func WithComponent(ctx context.Context, component string) context.Context {
	return context.WithValue(ctx, ComponentKey, component)
}

// WithStage returns a new context with the stage key set.
func WithStage(ctx context.Context, stage string) context.Context {
	return context.WithValue(ctx, StageKey, stage)
}

// InitFromEnv initializes the logger from environment variables.
func InitFromEnv() {
	logLevel := slog.LevelInfo
	if levelStr := os.Getenv("LOG_LEVEL"); levelStr != "" {
		switch strings.ToUpper(levelStr) {
		case "DEBUG":
			logLevel = slog.LevelDebug
		case "WARN":
			logLevel = slog.LevelWarn
		case "ERROR":
			logLevel = slog.LevelError
		}
	}

	output := os.Stdout
	if logFile := os.Getenv("LOG_OUTPUT"); logFile != "" && logFile != "stdout" {
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			output = f
		}
	}

	Init(
		output,
		logLevel,
		os.Getenv("VERSION"),
		os.Getenv("SERVICE_NAME"),
		os.Getenv("ENVIRONMENT"),
	)
}

// Helper functions for data serialization and merging
func mergeRequestData(target *RequestContext, source map[string]interface{}) {
	if method, ok := source["method"].(string); ok {
		target.Method = method
	}
	if endpoint, ok := source["endpoint"].(string); ok {
		target.Endpoint = endpoint
	}
	if path, ok := source["path"].(string); ok && target.Endpoint == "" {
		target.Endpoint = path
	}
	if headers, ok := source["headers"].(map[string]interface{}); ok {
		target.Headers = make(map[string]string)
		for k, v := range headers {
			target.Headers[k] = fmt.Sprintf("%v", v)
		}
	}
	if body, ok := source["body"]; ok {
		target.Body = body
	}
	if clientIP, ok := source["client_ip"].(string); ok {
		target.ClientIP = clientIP
	}
	if userAgent, ok := source["user_agent"].(string); ok {
		target.UserAgent = userAgent
	}
}

func mergeResponseData(target *ResponseContext, source map[string]interface{}) {
	if statusCode, ok := source["status_code"].(int); ok {
		target.StatusCode = statusCode
	}
	if status, ok := source["status"].(int); ok && target.StatusCode == 0 {
		target.StatusCode = status
	}
	if headers, ok := source["headers"].(map[string]interface{}); ok {
		target.Headers = make(map[string]string)
		for k, v := range headers {
			target.Headers[k] = fmt.Sprintf("%v", v)
		}
	}
	if body, ok := source["body"]; ok {
		target.Body = body
	}
	if duration, ok := source["duration_ms"].(int64); ok {
		target.DurationMs = duration
	}
}
