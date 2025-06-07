package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/aashari/go-generative-api-router/internal/utils"
)

// ConditionalLogger provides environment-aware logging with performance optimization
type ConditionalLogger struct {
	production     bool
	logLevel       slog.Level
	sensitiveData  bool
	masker         *utils.SensitiveDataMasker
	baseLogger     *slog.Logger
}

// LoggingConfig defines configuration for conditional logging
type LoggingConfig struct {
	Production    bool
	LogLevel      string
	SensitiveData bool
	MaskData      bool
}

// NewConditionalLogger creates a new conditional logger
func NewConditionalLogger(config LoggingConfig) *ConditionalLogger {
	// Determine if we're in production
	production := config.Production
	if !production {
		// Check environment variables
		env := strings.ToLower(os.Getenv("ENVIRONMENT"))
		production = env == "production" || env == "prod"
	}

	// Parse log level
	var logLevel slog.Level
	switch strings.ToLower(config.LogLevel) {
	case "debug":
		logLevel = LevelDebug
	case "info":
		logLevel = LevelInfo
	case "warn", "warning":
		logLevel = LevelWarn
	case "error":
		logLevel = LevelError
	default:
		if production {
			logLevel = LevelInfo
		} else {
			logLevel = LevelDebug
		}
	}

	// Initialize masker if needed
	var masker *utils.SensitiveDataMasker
	if config.MaskData || production {
		masker = utils.NewSensitiveDataMasker()
	}

	return &ConditionalLogger{
		production:    production,
		logLevel:      logLevel,
		sensitiveData: config.SensitiveData && !production,
		masker:        masker,
		baseLogger:    Logger, // Use the global logger
	}
}

// LogRequestOptimal logs requests with production-optimized approach
func (l *ConditionalLogger) LogRequestOptimal(ctx context.Context, method, path, userAgent string, headers map[string][]string, body []byte) {
	if l.production {
		// Production: Log minimal, essential information only
		l.baseLogger.InfoContext(ctx, "Request received",
			"method", method,
			"path", path,
			"size", len(body),
			"user_agent", l.maskUserAgent(userAgent))
	} else {
		// Development: Log detailed information
		maskedHeaders := headers
		maskedBody := string(body)
		
		if l.masker != nil {
			maskedHeaders = l.masker.MaskHeaders(headers)
			maskedBody = l.masker.MaskJSON(string(body))
		}
		
		l.baseLogger.DebugContext(ctx, "Request details",
			"method", method,
			"path", path,
			"headers", maskedHeaders,
			"body", maskedBody,
			"user_agent", userAgent)
	}
}

// LogVendorCommunicationOptimal logs vendor communication with conditional detail
func (l *ConditionalLogger) LogVendorCommunicationOptimal(ctx context.Context, vendor, url string, statusCode int, duration int64, body []byte) {
	if l.production {
		// Production: Log essential metrics only
		l.baseLogger.InfoContext(ctx, "Vendor request completed",
			"vendor", vendor,
			"status_code", statusCode,
			"duration_ms", duration,
			"response_size", len(body))
	} else {
		// Development: Log detailed information
		maskedBody := string(body)
		if l.masker != nil && len(body) > 0 {
			maskedBody = l.masker.MaskJSON(string(body))
		}
		
		l.baseLogger.DebugContext(ctx, "Vendor communication details",
			"vendor", vendor,
			"url", url,
			"status_code", statusCode,
			"duration_ms", duration,
			"response_body", maskedBody)
	}
}

// LogResponseOptimal logs responses with conditional detail
func (l *ConditionalLogger) LogResponseOptimal(ctx context.Context, statusCode int, responseSize int, processingTime int64) {
	if l.production {
		// Production: Log essential metrics only
		l.baseLogger.InfoContext(ctx, "Response sent",
			"status_code", statusCode,
			"size", responseSize,
			"processing_ms", processingTime)
	} else {
		// Development: Log more detailed information
		l.baseLogger.DebugContext(ctx, "Response processing completed",
			"status_code", statusCode,
			"response_size", responseSize,
			"processing_time_ms", processingTime)
	}
}

// LogErrorOptimal logs errors with appropriate detail level
func (l *ConditionalLogger) LogErrorOptimal(ctx context.Context, component string, err error, metadata map[string]any) {
	// Always log errors, but mask sensitive data in production
	maskedMetadata := metadata
	if l.masker != nil && l.production {
		maskedMetadata = l.masker.MaskSensitiveData(metadata).(map[string]any)
	}
	
	l.baseLogger.ErrorContext(ctx, "Error occurred",
		"component", component,
		"error", err.Error(),
		"metadata", maskedMetadata)
}

// LogConfigurationOptimal logs configuration loading with security considerations
func (l *ConditionalLogger) LogConfigurationOptimal(ctx context.Context, credCount, modelCount int, vendors []string) {
	if l.production {
		// Production: Log counts and vendors only, no sensitive details
		l.baseLogger.InfoContext(ctx, "Configuration loaded",
			"credentials_count", credCount,
			"models_count", modelCount,
			"vendors", vendors)
	} else {
		// Development: Log more details but still mask sensitive data
		l.baseLogger.DebugContext(ctx, "Configuration loaded with details",
			"credentials_count", credCount,
			"models_count", modelCount,
			"available_vendors", vendors)
	}
}

// LogSelectionOptimal logs vendor/model selection with conditional detail
func (l *ConditionalLogger) LogSelectionOptimal(ctx context.Context, vendor, model, originalModel string, totalCombinations int) {
	if l.production {
		// Production: Log essential selection info only
		l.baseLogger.InfoContext(ctx, "Vendor selected",
			"vendor", vendor,
			"model", model)
	} else {
		// Development: Log detailed selection information
		l.baseLogger.DebugContext(ctx, "Selection details",
			"vendor", vendor,
			"selected_model", model,
			"original_model", originalModel,
			"total_combinations", totalCombinations)
	}
}

// LogStreamingOptimal logs streaming operations with conditional detail
func (l *ConditionalLogger) LogStreamingOptimal(ctx context.Context, vendor string, chunkCount int, totalBytes int) {
	if l.production {
		// Production: Log streaming metrics only
		l.baseLogger.InfoContext(ctx, "Streaming completed",
			"vendor", vendor,
			"chunks", chunkCount,
			"total_bytes", totalBytes)
	} else {
		// Development: Log detailed streaming information
		l.baseLogger.DebugContext(ctx, "Streaming processing details",
			"vendor", vendor,
			"chunk_count", chunkCount,
			"total_bytes", totalBytes)
	}
}

// maskUserAgent masks potentially sensitive user agent information
func (l *ConditionalLogger) maskUserAgent(userAgent string) string {
	if l.masker != nil {
		return l.masker.GetMaskedString(userAgent)
	}
	return userAgent
}

// IsProductionMode returns whether the logger is in production mode
func (l *ConditionalLogger) IsProductionMode() bool {
	return l.production
}

// GetLogLevel returns the current log level
func (l *ConditionalLogger) GetLogLevel() slog.Level {
	return l.logLevel
}

// ShouldLogSensitiveData returns whether sensitive data should be logged
func (l *ConditionalLogger) ShouldLogSensitiveData() bool {
	return l.sensitiveData
}

// LogHealthCheck logs health check results with appropriate detail
func (l *ConditionalLogger) LogHealthCheck(ctx context.Context, status string, details map[string]interface{}) {
	if l.production {
		// Production: Log status only
		l.baseLogger.InfoContext(ctx, "Health check",
			"status", status)
	} else {
		// Development: Log detailed health information
		maskedDetails := details
		if l.masker != nil {
			maskedDetails = l.masker.MaskSensitiveData(details).(map[string]interface{})
		}
		
		l.baseLogger.DebugContext(ctx, "Health check details",
			"status", status,
			"details", maskedDetails)
	}
}

// LogPerformanceMetrics logs performance metrics
func (l *ConditionalLogger) LogPerformanceMetrics(ctx context.Context, operation string, duration int64, memoryUsage int64) {
	if l.production {
		// Production: Log essential performance metrics
		l.baseLogger.InfoContext(ctx, "Performance metric",
			"operation", operation,
			"duration_ms", duration)
	} else {
		// Development: Log detailed performance information
		l.baseLogger.DebugContext(ctx, "Performance details",
			"operation", operation,
			"duration_ms", duration,
			"memory_bytes", memoryUsage)
	}
} 