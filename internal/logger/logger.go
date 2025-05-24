package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"regexp"
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
		output, err = os.OpenFile(config.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
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
			// Sanitize sensitive data
			if isSensitiveKey(a.Key) {
				return slog.String(a.Key, sanitizeValue(a.Value.String()))
			}
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

// Sensitive field patterns
var sensitivePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(api[_-]?key|token|secret|password|credential)`),
	regexp.MustCompile(`(?i)(authorization|auth)`),
	regexp.MustCompile(`sk-[a-zA-Z0-9]+`),       // OpenAI API keys
	regexp.MustCompile(`AIza[a-zA-Z0-9_-]{35}`), // Google API keys
	regexp.MustCompile(`[A-Z0-9]{20}`),          // AWS access keys
	regexp.MustCompile(`[a-zA-Z0-9/+=]{40}`),    // AWS secret keys
}

// Check if a key is sensitive
func isSensitiveKey(key string) bool {
	for _, pattern := range sensitivePatterns {
		if pattern.MatchString(key) {
			return true
		}
	}
	return false
}

// Sanitize sensitive values
func sanitizeValue(value string) string {
	for _, pattern := range sensitivePatterns {
		if pattern.MatchString(value) {
			if len(value) > 8 {
				return value[:4] + "****" + value[len(value)-4:]
			}
			return "****"
		}
	}
	return value
}

// Context-aware logging functions
func WithContext(ctx context.Context) *slog.Logger {
	if Logger == nil {
		// Fallback to default if not initialized
		Init(DefaultConfig)
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

// Specialized logging functions for the proxy service

// LogProxyRequest logs the initial proxy request with vendor selection
func LogProxyRequest(ctx context.Context, originalModel, selectedVendor, selectedModel string, totalCombinations int) {
	WithContext(ctx).Info("Proxy request initiated",
		"component", "proxy",
		"original_model", originalModel,
		"selected_vendor", selectedVendor,
		"selected_model", selectedModel,
		"total_combinations", totalCombinations,
	)
}

// LogVendorResponse logs vendor response processing
func LogVendorResponse(ctx context.Context, vendor, actualModel, presentedModel string, responseSize int, processingTime time.Duration) {
	WithContext(ctx).Info("Vendor response processed",
		"component", "response_processor",
		"vendor", vendor,
		"actual_model", actualModel,
		"presented_model", presentedModel,
		"response_size_bytes", responseSize,
		"processing_time_ms", processingTime.Milliseconds(),
	)
}

// LogValidationResult logs response validation results
func LogValidationResult(ctx context.Context, vendor string, success bool, validationError error) {
	logger := WithContext(ctx)
	if success {
		logger.Debug("Response validation successful",
			"component", "validation",
			"vendor", vendor,
		)
	} else {
		logger.Warn("Response validation failed",
			"component", "validation",
			"vendor", vendor,
			"error", validationError.Error(),
		)
	}
}

// LogCompressionInfo logs compression-related information
func LogCompressionInfo(ctx context.Context, vendor string, originalSize, compressedSize int, compressionRatio float64) {
	WithContext(ctx).Debug("Response compression applied",
		"component", "compression",
		"vendor", vendor,
		"original_size_bytes", originalSize,
		"compressed_size_bytes", compressedSize,
		"compression_ratio", compressionRatio,
	)
}

// LogStreamingInfo logs streaming-related information
func LogStreamingInfo(ctx context.Context, vendor, model string, chunkCount int) {
	WithContext(ctx).Debug("Streaming response processed",
		"component", "streaming",
		"vendor", vendor,
		"model", model,
		"chunk_count", chunkCount,
	)
}

// LogError logs errors with appropriate context
func LogError(ctx context.Context, component string, err error, details map[string]any) {
	args := []any{
		"component", component,
		"error", err.Error(),
	}

	for k, v := range details {
		args = append(args, k, v)
	}

	WithContext(ctx).Error("Operation failed", args...)
}

// LogMetrics logs performance metrics
func LogMetrics(ctx context.Context, operation string, duration time.Duration, success bool, details map[string]any) {
	args := []any{
		"component", "metrics",
		"operation", operation,
		"duration_ms", duration.Milliseconds(),
		"success", success,
	}

	for k, v := range details {
		args = append(args, k, v)
	}

	WithContext(ctx).Info("Operation metrics", args...)
}

// Sanitize a map of data for logging
func SanitizeMap(data map[string]any) map[string]any {
	sanitized := make(map[string]any)
	for k, v := range data {
		if isSensitiveKey(k) {
			if str, ok := v.(string); ok {
				sanitized[k] = sanitizeValue(str)
			} else {
				sanitized[k] = "[REDACTED]"
			}
		} else {
			sanitized[k] = v
		}
	}
	return sanitized
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
