package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

// Logger levels
const (
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
)

// Context keys
type contextKey string

const (
	RequestIDKey contextKey = "request_id"
	VendorKey    contextKey = "vendor"
	ModelKey     contextKey = "model"
)

// Global logger instance
var Logger *slog.Logger

// Configuration for logger
type Config struct {
	Level      slog.Level
	Format     string // "json" or "text"
	Output     string // "stdout", "stderr", or file path
	TimeFormat string
}

// Default configuration
var DefaultConfig = Config{
	Level:      LevelInfo,
	Format:     "json",
	Output:     "stdout",
	TimeFormat: time.RFC3339,
}

// Initialize the global logger
func Init(config Config) error {
	var output *os.File
	var err error

	switch config.Output {
	case "stdout", "":
		output = os.Stdout
	case "stderr":
		output = os.Stderr
	default:
		output, err = os.OpenFile(config.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			return fmt.Errorf("failed to open log file %s: %w", config.Output, err)
		}
	}

	opts := &slog.HandlerOptions{
		Level: config.Level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize time format
			if a.Key == slog.TimeKey {
				return slog.String(slog.TimeKey, a.Value.Time().Format(config.TimeFormat))
			}
			// NO SANITIZATION - log everything as-is
			return a
		},
	}

	var handler slog.Handler
	switch config.Format {
	case "json":
		handler = slog.NewJSONHandler(output, opts)
	default:
		handler = slog.NewTextHandler(output, opts)
	}

	Logger = slog.New(handler)
	return nil
}

// Context-aware logging functions
func WithContext(ctx context.Context) *slog.Logger {
	if Logger == nil {
		// Fallback to default if not initialized
		if err := Init(DefaultConfig); err != nil {
			// If default logger initialization fails, log to stderr and use a temporary stderr logger for this context
			fmt.Fprintf(os.Stderr, "FATAL: Failed to initialize default logger in WithContext: %v\n", err)
			// Return a temporary, minimal logger that writes to stderr
			return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: LevelDebug}))
		}
	}

	logger := Logger

	// Add request ID if available
	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		logger = logger.With("request_id", requestID)
	}

	// Add vendor if available
	if vendor := ctx.Value(VendorKey); vendor != nil {
		logger = logger.With("vendor", vendor)
	}

	// Add model if available
	if model := ctx.Value(ModelKey); model != nil {
		logger = logger.With("model", model)
	}

	return logger
}

// Convenience functions for different log levels
func Debug(msg string, args ...any) {
	if Logger != nil {
		Logger.Debug(msg, args...)
	}
}

func Info(msg string, args ...any) {
	if Logger != nil {
		Logger.Info(msg, args...)
	}
}

func Warn(msg string, args ...any) {
	if Logger != nil {
		Logger.Warn(msg, args...)
	}
}

func Error(msg string, args ...any) {
	if Logger != nil {
		Logger.Error(msg, args...)
	}
}

// Context-aware convenience functions
func DebugCtx(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Debug(msg, args...)
}

func InfoCtx(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Info(msg, args...)
}

func WarnCtx(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Warn(msg, args...)
}

func ErrorCtx(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Error(msg, args...)
}

// Comprehensive data logging functions - log entire data structures

// LogCompleteData logs any data structure in its entirety as JSON
func LogCompleteData(ctx context.Context, level slog.Level, msg string, data any) {
	logger := WithContext(ctx)
	
	// Convert data to JSON for complete logging
	jsonData, err := json.Marshal(data)
	if err != nil {
		logger.Log(ctx, level, msg, "data_marshal_error", err.Error(), "raw_data", fmt.Sprintf("%+v", data))
		return
	}
	
	logger.Log(ctx, level, msg, "complete_data", string(jsonData), "data_type", fmt.Sprintf("%T", data))
}

// LogCompleteDataDebug logs complete data at DEBUG level
func LogCompleteDataDebug(ctx context.Context, msg string, data any) {
	LogCompleteData(ctx, LevelDebug, msg, data)
}

// LogCompleteDataInfo logs complete data at INFO level
func LogCompleteDataInfo(ctx context.Context, msg string, data any) {
	LogCompleteData(ctx, LevelInfo, msg, data)
}

// LogCompleteDataWarn logs complete data at WARN level
func LogCompleteDataWarn(ctx context.Context, msg string, data any) {
	LogCompleteData(ctx, LevelWarn, msg, data)
}

// LogCompleteDataError logs complete data at ERROR level
func LogCompleteDataError(ctx context.Context, msg string, data any) {
	LogCompleteData(ctx, LevelError, msg, data)
}

// LogMultipleData logs multiple data structures completely
func LogMultipleData(ctx context.Context, level slog.Level, msg string, dataMap map[string]any) {
	logger := WithContext(ctx)
	
	args := []any{}
	for key, value := range dataMap {
		// Convert each value to JSON for complete logging
		jsonData, err := json.Marshal(value)
		if err != nil {
			args = append(args, key+"_marshal_error", err.Error())
			args = append(args, key+"_raw", fmt.Sprintf("%+v", value))
		} else {
			args = append(args, key+"_complete", string(jsonData))
		}
		args = append(args, key+"_type", fmt.Sprintf("%T", value))
	}
	
	logger.Log(ctx, level, msg, args...)
}

// LogRequest logs complete HTTP request data
func LogRequest(ctx context.Context, method, path, userAgent string, headers map[string][]string, body []byte) {
	LogMultipleData(ctx, LevelInfo, "Complete HTTP request data", map[string]any{
		"method":     method,
		"path":       path,
		"user_agent": userAgent,
		"headers":    headers,
		"body":       string(body),
		"body_bytes": body,
	})
}

// LogResponse logs complete HTTP response data
func LogResponse(ctx context.Context, statusCode int, headers map[string][]string, body []byte) {
	LogMultipleData(ctx, LevelInfo, "Complete HTTP response data", map[string]any{
		"status_code": statusCode,
		"headers":     headers,
		"body":        string(body),
		"body_bytes":  body,
		"body_size":   len(body),
	})
}

// LogVendorCommunication logs complete vendor request/response cycle
func LogVendorCommunication(ctx context.Context, vendor, url string, requestBody, responseBody []byte, requestHeaders, responseHeaders map[string][]string) {
	LogMultipleData(ctx, LevelInfo, "Complete vendor communication", map[string]any{
		"vendor":           vendor,
		"url":              url,
		"request_body":     string(requestBody),
		"request_body_bytes": requestBody,
		"response_body":    string(responseBody),
		"response_body_bytes": responseBody,
		"request_headers":  requestHeaders,
		"response_headers": responseHeaders,
	})
}

// Specialized logging functions for the proxy service

// LogProxyRequest logs the initial proxy request with complete data
func LogProxyRequest(ctx context.Context, originalModel, selectedVendor, selectedModel string, totalCombinations int, requestData any) {
	LogMultipleData(ctx, LevelInfo, "Proxy request initiated with complete data", map[string]any{
		"component":          "proxy",
		"original_model":     originalModel,
		"selected_vendor":    selectedVendor,
		"selected_model":     selectedModel,
		"total_combinations": totalCombinations,
		"complete_request":   requestData,
	})
}

// LogVendorResponse logs vendor response processing with complete data
func LogVendorResponse(ctx context.Context, vendor, actualModel, presentedModel string, responseSize int, processingTime time.Duration, completeResponse any) {
	LogMultipleData(ctx, LevelInfo, "Vendor response processed with complete data", map[string]any{
		"component":         "response_processor",
		"vendor":            vendor,
		"actual_model":      actualModel,
		"presented_model":   presentedModel,
		"response_size_bytes": responseSize,
		"processing_time_ms": processingTime.Milliseconds(),
		"complete_response": completeResponse,
	})
}

// LogValidationResult logs response validation results with complete data
func LogValidationResult(ctx context.Context, vendor string, success bool, validationError error, validatedData any) {
	if success {
		LogMultipleData(ctx, LevelDebug, "Response validation successful with complete data", map[string]any{
			"component":      "validation",
			"vendor":         vendor,
			"validated_data": validatedData,
		})
	} else {
		LogMultipleData(ctx, LevelWarn, "Response validation failed with complete data", map[string]any{
			"component":      "validation",
			"vendor":         vendor,
			"error":          validationError.Error(),
			"failed_data":    validatedData,
		})
	}
}

// LogStreamingInfo logs streaming-related information with complete data
func LogStreamingInfo(ctx context.Context, vendor, model string, chunkCount int, completeStreamData any) {
	LogMultipleData(ctx, LevelDebug, "Streaming response processed with complete data", map[string]any{
		"component":           "streaming",
		"vendor":              vendor,
		"model":               model,
		"chunk_count":         chunkCount,
		"complete_stream_data": completeStreamData,
	})
}

// LogError logs errors with complete context and data
func LogError(ctx context.Context, component string, err error, completeDetails map[string]any) {
	args := []any{
		"component", component,
		"error", err.Error(),
	}

	// Log all details completely without any filtering
	for k, v := range completeDetails {
		// Convert complex data to JSON for complete logging
		if jsonData, jsonErr := json.Marshal(v); jsonErr == nil {
			args = append(args, k+"_complete", string(jsonData))
		}
		args = append(args, k, v)
		args = append(args, k+"_type", fmt.Sprintf("%T", v))
	}

	WithContext(ctx).Error("Operation failed with complete details", args...)
}

// LogConfiguration logs complete configuration data
func LogConfiguration(ctx context.Context, configData any) {
	LogCompleteDataInfo(ctx, "Complete configuration loaded", configData)
}

// LogCredentials logs complete credentials (including sensitive data as requested)
func LogCredentials(ctx context.Context, credentials any) {
	LogCompleteDataInfo(ctx, "Complete credentials data", credentials)
}

// LogModels logs complete model configuration
func LogModels(ctx context.Context, models any) {
	LogCompleteDataInfo(ctx, "Complete models configuration", models)
}

// Initialize with environment-based configuration
func InitFromEnv() error {
	config := DefaultConfig

	// Override with environment variables
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		switch strings.ToUpper(level) {
		case "DEBUG":
			config.Level = LevelDebug
		case "INFO":
			config.Level = LevelInfo
		case "WARN", "WARNING":
			config.Level = LevelWarn
		case "ERROR":
			config.Level = LevelError
		}
	}

	if format := os.Getenv("LOG_FORMAT"); format != "" {
		config.Format = format
	}

	if output := os.Getenv("LOG_OUTPUT"); output != "" {
		config.Output = output
	}

	return Init(config)
}
