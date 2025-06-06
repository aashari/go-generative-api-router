package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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

// Service configuration
var (
	ServiceName = "generative-api-router"
	Environment = "development"
)

// Configuration for logger
type Config struct {
	Level       slog.Level
	Format      string // "json" or "text"
	Output      string // "stdout", "stderr", or file path
	TimeFormat  string
	ServiceName string
	Environment string
}

// Default configuration
var DefaultConfig = Config{
	Level:       LevelInfo,
	Format:      "json",
	Output:      "stdout",
	TimeFormat:  time.RFC3339,
	ServiceName: "generative-api-router",
	Environment: "development",
}

// StructuredLogEntry represents the new log structure
type StructuredLogEntry struct {
	Timestamp   string                 `json:"timestamp"`
	Level       string                 `json:"level"`
	Message     string                 `json:"message"`
	Service     string                 `json:"service"`
	Environment string                 `json:"environment"`
	Attributes  map[string]interface{} `json:"attributes,omitempty"`
	Request     map[string]interface{} `json:"request,omitempty"`
	Response    map[string]interface{} `json:"response,omitempty"`
	Error       map[string]interface{} `json:"error,omitempty"`
}

// Initialize the global logger
func Init(config Config) error {
	var output *os.File
	var err error

	// Set global service configuration
	ServiceName = config.ServiceName
	Environment = config.Environment

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
			// Transform standard slog fields to our structure
			switch a.Key {
			case slog.TimeKey:
				return slog.String("timestamp", a.Value.Time().Format(config.TimeFormat))
			case slog.LevelKey:
				return slog.String("level", a.Value.String())
			case slog.MessageKey:
				return slog.String("message", a.Value.String())
			}
			return a
		},
	}

	var handler slog.Handler
	switch config.Format {
	case "json":
		handler = &StructuredJSONHandler{
			writer:      output,
			serviceName: config.ServiceName,
			environment: config.Environment,
		}
	default:
		handler = slog.NewTextHandler(output, opts)
	}

	Logger = slog.New(handler)
	return nil
}

// StructuredJSONHandler implements a custom JSON handler for our structured format
type StructuredJSONHandler struct {
	writer      io.Writer
	serviceName string
	environment string
}

func (h *StructuredJSONHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true // Enable all levels for now
}

func (h *StructuredJSONHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h // For simplicity, return self
}

func (h *StructuredJSONHandler) WithGroup(name string) slog.Handler {
	return h // For simplicity, return self
}

func (h *StructuredJSONHandler) Handle(ctx context.Context, r slog.Record) error {
	// Create structured log entry
	entry := StructuredLogEntry{
		Timestamp:   r.Time.Format(time.RFC3339),
		Level:       r.Level.String(),
		Message:     r.Message,
		Service:     h.serviceName,
		Environment: h.environment,
		Attributes:  make(map[string]interface{}),
	}

	// Extract context values - check if context is available
	if ctx != nil {
		if requestID := ctx.Value(RequestIDKey); requestID != nil {
			if entry.Request == nil {
				entry.Request = make(map[string]interface{})
			}
			entry.Request["request_id"] = requestID
		}

		if vendor := ctx.Value(VendorKey); vendor != nil {
			if entry.Attributes == nil {
				entry.Attributes = make(map[string]interface{})
			}
			entry.Attributes["vendor"] = vendor
		}

		if model := ctx.Value(ModelKey); model != nil {
			if entry.Attributes == nil {
				entry.Attributes = make(map[string]interface{})
			}
			entry.Attributes["model"] = model
		}
	}

	// Process record attributes
	r.Attrs(func(a slog.Attr) bool {
		key := a.Key
		value := a.Value.Any()

		// Route attributes to appropriate sections
		switch {
		case strings.HasPrefix(key, "request_"):
			if entry.Request == nil {
				entry.Request = make(map[string]interface{})
			}
			entry.Request[strings.TrimPrefix(key, "request_")] = value
		case strings.HasPrefix(key, "response_"):
			if entry.Response == nil {
				entry.Response = make(map[string]interface{})
			}
			entry.Response[strings.TrimPrefix(key, "response_")] = value
		case strings.HasPrefix(key, "error_"):
			if entry.Error == nil {
				entry.Error = make(map[string]interface{})
			}
			entry.Error[strings.TrimPrefix(key, "error_")] = value
		case key == "error":
			if entry.Error == nil {
				entry.Error = make(map[string]interface{})
			}
			if err, ok := value.(error); ok {
				entry.Error["message"] = err.Error()
				entry.Error["type"] = fmt.Sprintf("%T", err)
			} else {
				entry.Error["message"] = fmt.Sprintf("%v", value)
			}
		default:
			// Everything else goes to attributes
			if entry.Attributes == nil {
				entry.Attributes = make(map[string]interface{})
			}
			entry.Attributes[key] = value
		}
		return true
	})

	// Clean up empty sections
	if len(entry.Attributes) == 0 {
		entry.Attributes = nil
	}
	if len(entry.Request) == 0 {
		entry.Request = nil
	}
	if len(entry.Response) == 0 {
		entry.Response = nil
	}
	if len(entry.Error) == 0 {
		entry.Error = nil
	}

	// Marshal and write
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	// Write to the output
	_, err = fmt.Fprintln(h.writer, string(data))
	return err
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

	// The context values are now handled in the StructuredJSONHandler.Handle method
	// so we just return the logger as-is
	return Logger
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
	// Extract context values and add them as attributes
	args = appendContextValues(ctx, args)
	WithContext(ctx).DebugContext(ctx, msg, args...)
}

func InfoCtx(ctx context.Context, msg string, args ...any) {
	// Extract context values and add them as attributes
	args = appendContextValues(ctx, args)
	WithContext(ctx).InfoContext(ctx, msg, args...)
}

func WarnCtx(ctx context.Context, msg string, args ...any) {
	// Extract context values and add them as attributes
	args = appendContextValues(ctx, args)
	WithContext(ctx).WarnContext(ctx, msg, args...)
}

func ErrorCtx(ctx context.Context, msg string, args ...any) {
	// Extract context values and add them as attributes
	args = appendContextValues(ctx, args)
	WithContext(ctx).ErrorContext(ctx, msg, args...)
}

// appendContextValues extracts context values and adds them to the args slice
func appendContextValues(ctx context.Context, args []any) []any {
	if ctx == nil {
		return args
	}

	// Add request ID if available
	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		args = append(args, "request_request_id", requestID)
	}

	// Add vendor if available
	if vendor := ctx.Value(VendorKey); vendor != nil {
		args = append(args, "vendor", vendor)
	}

	// Add model if available
	if model := ctx.Value(ModelKey); model != nil {
		args = append(args, "model", model)
	}

	return args
}

// Structured logging functions for the new format

// LogWithStructure logs with the new structured format
func LogWithStructure(ctx context.Context, level slog.Level, message string, attributes map[string]interface{}, request map[string]interface{}, response map[string]interface{}, errorData map[string]interface{}) {
	logger := WithContext(ctx)

	args := []any{}

	// Add attributes
	for k, v := range attributes {
		args = append(args, k, v)
	}

	// Add request data with prefix
	for k, v := range request {
		args = append(args, "request_"+k, v)
	}

	// Add response data with prefix
	for k, v := range response {
		args = append(args, "response_"+k, v)
	}

	// Add error data with prefix
	for k, v := range errorData {
		args = append(args, "error_"+k, v)
	}

	logger.Log(ctx, level, message, args...)
}

// LogRequest logs HTTP request data in the new structure
func LogRequest(ctx context.Context, method, path, userAgent string, headers map[string][]string, body []byte) {
	request := map[string]interface{}{
		"method":         method,
		"path":           path,
		"user_agent":     userAgent,
		"headers":        headers,
		"body":           string(body),
		"content_length": len(body),
	}

	LogWithStructure(ctx, LevelInfo, "HTTP request received", nil, request, nil, nil)
}

// LogResponse logs HTTP response data in the new structure
func LogResponse(ctx context.Context, statusCode int, headers map[string][]string, body []byte) {
	response := map[string]interface{}{
		"status_code":    statusCode,
		"headers":        headers,
		"body":           string(body),
		"content_length": len(body),
	}

	LogWithStructure(ctx, LevelInfo, "HTTP response sent", nil, nil, response, nil)
}

// LogVendorCommunication logs vendor request/response cycle
func LogVendorCommunication(ctx context.Context, vendor, url string, requestBody, responseBody []byte, requestHeaders, responseHeaders map[string][]string) {
	attributes := map[string]interface{}{
		"vendor": vendor,
		"url":    url,
	}

	request := map[string]interface{}{
		"body":    string(requestBody),
		"headers": requestHeaders,
	}

	response := map[string]interface{}{
		"body":    string(responseBody),
		"headers": responseHeaders,
	}

	LogWithStructure(ctx, LevelInfo, "Vendor communication completed", attributes, request, response, nil)
}

// LogProxyRequest logs proxy request with vendor selection
func LogProxyRequest(ctx context.Context, originalModel, selectedVendor, selectedModel string, totalCombinations int, requestData any) {
	attributes := map[string]interface{}{
		"component":          "proxy",
		"original_model":     originalModel,
		"selected_vendor":    selectedVendor,
		"selected_model":     selectedModel,
		"total_combinations": totalCombinations,
		"request_data":       requestData,
	}

	LogWithStructure(ctx, LevelInfo, "Proxy request initiated", attributes, nil, nil, nil)
}

// LogVendorResponse logs vendor response processing
func LogVendorResponse(ctx context.Context, vendor, actualModel, presentedModel string, responseSize int, processingTime time.Duration, completeResponse any) {
	attributes := map[string]interface{}{
		"component":           "response_processor",
		"vendor":              vendor,
		"actual_model":        actualModel,
		"presented_model":     presentedModel,
		"response_size_bytes": responseSize,
		"processing_time_ms":  processingTime.Milliseconds(),
		"complete_response":   completeResponse,
	}

	LogWithStructure(ctx, LevelInfo, "Vendor response processed", attributes, nil, nil, nil)
}

// LogValidationResult logs response validation results
func LogValidationResult(ctx context.Context, vendor string, success bool, validationError error, validatedData any) {
	attributes := map[string]interface{}{
		"component":      "validation",
		"vendor":         vendor,
		"success":        success,
		"validated_data": validatedData,
	}

	var errorData map[string]interface{}
	if !success && validationError != nil {
		errorData = map[string]interface{}{
			"message": validationError.Error(),
			"type":    fmt.Sprintf("%T", validationError),
		}
	}

	level := LevelDebug
	message := "Response validation successful"
	if !success {
		level = LevelWarn
		message = "Response validation failed"
	}

	LogWithStructure(ctx, level, message, attributes, nil, nil, errorData)
}

// LogStreamingInfo logs streaming-related information
func LogStreamingInfo(ctx context.Context, vendor, model string, chunkCount int, completeStreamData any) {
	attributes := map[string]interface{}{
		"component":            "streaming",
		"vendor":               vendor,
		"model":                model,
		"chunk_count":          chunkCount,
		"complete_stream_data": completeStreamData,
	}

	LogWithStructure(ctx, LevelDebug, "Streaming response processed", attributes, nil, nil, nil)
}

// LogError logs errors with complete context and data
func LogError(ctx context.Context, component string, err error, completeDetails map[string]any) {
	attributes := map[string]interface{}{
		"component": component,
	}

	// Add complete details to attributes
	for k, v := range completeDetails {
		attributes[k] = v
	}

	errorData := map[string]interface{}{
		"message": err.Error(),
		"type":    fmt.Sprintf("%T", err),
	}

	LogWithStructure(ctx, LevelError, "Operation failed", attributes, nil, nil, errorData)
}

// LogConfiguration logs complete configuration data
func LogConfiguration(ctx context.Context, configData any) {
	attributes := map[string]interface{}{
		"configuration": configData,
	}

	LogWithStructure(ctx, LevelInfo, "Configuration loaded", attributes, nil, nil, nil)
}

// LogCredentials logs complete credentials (including sensitive data as requested)
func LogCredentials(ctx context.Context, credentials any) {
	attributes := map[string]interface{}{
		"credentials": credentials,
	}

	LogWithStructure(ctx, LevelInfo, "Credentials loaded", attributes, nil, nil, nil)
}

// LogModels logs complete model configuration
func LogModels(ctx context.Context, models any) {
	attributes := map[string]interface{}{
		"models": models,
	}

	LogWithStructure(ctx, LevelInfo, "Models configuration loaded", attributes, nil, nil, nil)
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

	if serviceName := os.Getenv("SERVICE_NAME"); serviceName != "" {
		config.ServiceName = serviceName
	}

	if environment := os.Getenv("ENVIRONMENT"); environment != "" {
		config.Environment = environment
	} else if env := os.Getenv("ENV"); env != "" {
		config.Environment = env
	}

	return Init(config)
}

// Legacy compatibility functions - these maintain the old interface but use new structure internally

// LogCompleteData logs any data structure in its entirety as JSON (legacy compatibility)
func LogCompleteData(ctx context.Context, level slog.Level, msg string, data any) {
	attributes := map[string]interface{}{
		"complete_data": data,
		"data_type":     fmt.Sprintf("%T", data),
	}

	LogWithStructure(ctx, level, msg, attributes, nil, nil, nil)
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

// LogMultipleData logs multiple data structures completely (legacy compatibility)
func LogMultipleData(ctx context.Context, level slog.Level, msg string, dataMap map[string]any) {
	attributes := make(map[string]interface{})
	request := make(map[string]interface{})
	response := make(map[string]interface{})

	for key, value := range dataMap {
		// Route data to appropriate sections based on key patterns
		switch {
		case strings.Contains(key, "vendor_request") || strings.Contains(key, "client_request"):
			// Extract request data from the value if it's a map
			if requestData, ok := value.(map[string]interface{}); ok {
				for k, v := range requestData {
					request[k] = v
				}
			} else {
				request[key] = value
			}
		case strings.Contains(key, "vendor_response"):
			// Extract response data from the value if it's a map
			if responseData, ok := value.(map[string]interface{}); ok {
				for k, v := range responseData {
					response[k] = v
				}
			} else {
				response[key] = value
			}
		case strings.Contains(key, "vendor_details"):
			// Extract vendor details into attributes
			if detailsData, ok := value.(map[string]interface{}); ok {
				for k, v := range detailsData {
					attributes[k] = v
				}
			} else {
				attributes[key] = value
			}
		default:
			// Everything else goes to attributes
			attributes[key] = value
		}
	}

	// Clean up empty sections
	if len(request) == 0 {
		request = nil
	}
	if len(response) == 0 {
		response = nil
	}
	if len(attributes) == 0 {
		attributes = nil
	}

	LogWithStructure(ctx, level, msg, attributes, request, response, nil)
}
