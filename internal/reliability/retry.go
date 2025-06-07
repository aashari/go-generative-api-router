package reliability

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/aashari/go-generative-api-router/internal/logger"
)

// RetryConfig defines configuration for retry behavior
type RetryConfig struct {
	MaxAttempts    int
	InitialDelay   time.Duration
	MaxDelay       time.Duration
	BackoffFactor  float64
	RetryableErrors []error
}

// DefaultRetryConfig returns a sensible default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      5 * time.Second,
		BackoffFactor: 2.0,
		RetryableErrors: []error{
			// Add common retryable errors here
		},
	}
}

// RetryExecutor handles retry logic with exponential backoff
type RetryExecutor struct {
	config RetryConfig
}

// NewRetryExecutor creates a new retry executor with the given configuration
func NewRetryExecutor(config RetryConfig) *RetryExecutor {
	return &RetryExecutor{
		config: config,
	}
}

// ExecuteWithRetry executes an operation with retry logic
func (r *RetryExecutor) ExecuteWithRetry(ctx context.Context, operation func() error) error {
	var lastErr error

	for attempt := 1; attempt <= r.config.MaxAttempts; attempt++ {
		// Execute the operation
		if err := operation(); err != nil {
			lastErr = err

			// Check if error is retryable
			if !r.isRetryableError(err) {
				logger.ErrorCtx(ctx, "Non-retryable error encountered",
					"attempt", attempt,
					"error", err)
				return err
			}

			// If this is the last attempt, don't wait
			if attempt >= r.config.MaxAttempts {
				logger.ErrorCtx(ctx, "Max retry attempts reached",
					"max_attempts", r.config.MaxAttempts,
					"final_error", err)
				break
			}

			// Calculate backoff delay
			delay := r.calculateBackoff(attempt)
			
			logger.WarnCtx(ctx, "Operation failed, retrying",
				"attempt", attempt,
				"max_attempts", r.config.MaxAttempts,
				"delay_ms", delay.Milliseconds(),
				"error", err)

			// Wait for backoff delay or context cancellation
			select {
			case <-time.After(delay):
				continue
			case <-ctx.Done():
				logger.ErrorCtx(ctx, "Retry cancelled due to context cancellation",
					"attempt", attempt,
					"context_error", ctx.Err())
				return ctx.Err()
			}
		} else {
			// Operation succeeded
			if attempt > 1 {
				logger.InfoCtx(ctx, "Operation succeeded after retry",
					"successful_attempt", attempt,
					"total_attempts", attempt)
			}
			return nil
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", r.config.MaxAttempts, lastErr)
}

// calculateBackoff calculates the backoff delay for a given attempt
func (r *RetryExecutor) calculateBackoff(attempt int) time.Duration {
	// Exponential backoff: delay = initialDelay * (backoffFactor ^ (attempt - 1))
	delay := float64(r.config.InitialDelay) * math.Pow(r.config.BackoffFactor, float64(attempt-1))
	
	// Cap at maximum delay
	if time.Duration(delay) > r.config.MaxDelay {
		delay = float64(r.config.MaxDelay)
	}
	
	return time.Duration(delay)
}

// isRetryableError checks if an error should trigger a retry
func (r *RetryExecutor) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check against configured retryable errors
	for _, retryableErr := range r.config.RetryableErrors {
		if err == retryableErr {
			return true
		}
	}

	// Check for common retryable error patterns
	errStr := err.Error()
	
	// Network-related errors that are typically retryable
	retryablePatterns := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"temporary failure",
		"network is unreachable",
		"no such host",
		"context deadline exceeded",
		"i/o timeout",
		"connection timed out",
		"broken pipe",
		"connection aborted",
	}

	for _, pattern := range retryablePatterns {
		if contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    (len(s) > len(substr) && 
		     (s[:len(substr)] == substr || 
		      s[len(s)-len(substr):] == substr || 
		      containsSubstring(s, substr))))
}

// containsSubstring performs a simple substring search
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// RetryableOperation defines a function that can be retried
type RetryableOperation func() error

// RetryWithConfig executes an operation with retry using the provided config
func RetryWithConfig(ctx context.Context, config RetryConfig, operation RetryableOperation) error {
	executor := NewRetryExecutor(config)
	return executor.ExecuteWithRetry(ctx, operation)
}

// Retry executes an operation with default retry configuration
func Retry(ctx context.Context, operation RetryableOperation) error {
	return RetryWithConfig(ctx, DefaultRetryConfig(), operation)
}

// RetryHTTP executes an HTTP-related operation with HTTP-specific retry configuration
func RetryHTTP(ctx context.Context, operation RetryableOperation) error {
	config := RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  200 * time.Millisecond,
		MaxDelay:      2 * time.Second,
		BackoffFactor: 1.5,
	}
	return RetryWithConfig(ctx, config, operation)
}

// RetryVendorAPI executes a vendor API operation with vendor-specific retry configuration
func RetryVendorAPI(ctx context.Context, operation RetryableOperation) error {
	config := RetryConfig{
		MaxAttempts:   2, // Conservative for external APIs
		InitialDelay:  500 * time.Millisecond,
		MaxDelay:      3 * time.Second,
		BackoffFactor: 2.0,
	}
	return RetryWithConfig(ctx, config, operation)
} 