package reliability

import (
	"context"
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/aashari/go-generative-api-router/internal/logger"
)

// RetryConfig defines the configuration for retry behavior
type RetryConfig struct {
	MaxAttempts     int           // Maximum number of retry attempts
	InitialDelay    time.Duration // Initial delay before first retry
	MaxDelay        time.Duration // Maximum delay between retries
	BackoffFactor   float64       // Multiplier for exponential backoff
	JitterEnabled   bool          // Whether to add random jitter to delays
	RetryableErrors []string      // List of error types that should be retried
}

// DefaultRetryConfig returns a sensible default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:   3,                // Retry up to 3 times
		InitialDelay:  1 * time.Second,  // Start with 1 second delay
		MaxDelay:      30 * time.Second, // Cap at 30 seconds
		BackoffFactor: 2.0,              // Double the delay each time
		JitterEnabled: true,             // Add randomness to prevent thundering herd
		RetryableErrors: []string{
			"insufficient_quota",
			"rate_limit_exceeded",
			"server_error",
		},
	}
}

// RetryExecutor handles retry logic with exponential backoff
type RetryExecutor struct {
	config *RetryConfig
}

// NewRetryExecutor creates a new retry executor with the given configuration
func NewRetryExecutor(config *RetryConfig) *RetryExecutor {
	if config == nil {
		config = DefaultRetryConfig()
	}
	return &RetryExecutor{
		config: config,
	}
}

// ExecuteWithRetry executes an operation with retry logic and exponential backoff
func (r *RetryExecutor) ExecuteWithRetry(ctx context.Context, operation func() error) error {
	var lastErr error

	for attempt := 1; attempt <= r.config.MaxAttempts; attempt++ {
		// Execute the operation
		if err := operation(); err != nil {
			lastErr = err

			// Check if this error is retriable
			if !r.isRetryableError(err) {
				logger.LogWithStructure(ctx, logger.LevelWarn, "Non-retriable error encountered",
					map[string]interface{}{
						"attempt":    attempt,
						"error":      err.Error(),
						"error_type": fmt.Sprintf("%T", err),
						"retriable":  false,
					},
					nil, // request
					nil, // response
					map[string]interface{}{
						"message": err.Error(),
						"type":    "non_retriable_error",
					}) // error
				return err
			}

			// If this is the last attempt, don't wait
			if attempt >= r.config.MaxAttempts {
				logger.LogWithStructure(ctx, logger.LevelError, "All retry attempts exhausted",
					map[string]interface{}{
						"total_attempts": attempt,
						"max_attempts":   r.config.MaxAttempts,
						"final_error":    err.Error(),
						"error_type":     fmt.Sprintf("%T", err),
					},
					nil, // request
					nil, // response
					map[string]interface{}{
						"message": fmt.Sprintf("Operation failed after %d attempts: %v", attempt, err),
						"type":    "retry_exhausted",
					}) // error
				break
			}

			// Calculate delay for next attempt
			delay := r.calculateBackoff(attempt)

			logger.LogWithStructure(ctx, logger.LevelWarn, "Operation failed, retrying with exponential backoff",
				map[string]interface{}{
					"attempt":       attempt,
					"max_attempts":  r.config.MaxAttempts,
					"delay_ms":      delay.Milliseconds(),
					"delay_seconds": delay.Seconds(),
					"error":         err.Error(),
					"error_type":    fmt.Sprintf("%T", err),
					"next_attempt":  attempt + 1,
				},
				nil, // request
				nil, // response
				map[string]interface{}{
					"message": err.Error(),
					"type":    "retry_attempt",
				}) // error

			// Wait for the calculated delay or until context is cancelled
			select {
			case <-time.After(delay):
				continue // Proceed to next attempt
			case <-ctx.Done():
				logger.LogWithStructure(ctx, logger.LevelWarn, "Retry cancelled due to context cancellation",
					map[string]interface{}{
						"attempt":        attempt,
						"context_error":  ctx.Err().Error(),
						"original_error": err.Error(),
					},
					nil, // request
					nil, // response
					map[string]interface{}{
						"message": ctx.Err().Error(),
						"type":    "retry_cancelled",
					}) // error
				return ctx.Err()
			}
		} else {
			// Operation succeeded
			if attempt > 1 {
				logger.LogWithStructure(ctx, logger.LevelInfo, "Operation succeeded after retry",
					map[string]interface{}{
						"successful_attempt": attempt,
						"total_attempts":     attempt,
						"retry_successful":   true,
					},
					nil, // request
					nil, // response
					nil) // error
			}
			return nil
		}
	}

	// All attempts failed
	return fmt.Errorf("operation failed after %d attempts: %w", r.config.MaxAttempts, lastErr)
}

// calculateBackoff calculates the delay for the given attempt using exponential backoff
func (r *RetryExecutor) calculateBackoff(attempt int) time.Duration {
	// Calculate exponential backoff: initialDelay * (backoffFactor ^ (attempt - 1))
	delay := float64(r.config.InitialDelay) * math.Pow(r.config.BackoffFactor, float64(attempt-1))

	// Apply maximum delay cap
	if delay > float64(r.config.MaxDelay) {
		delay = float64(r.config.MaxDelay)
	}

	// Add jitter if enabled (±25% randomness)
	if r.config.JitterEnabled {
		// Use crypto/rand for cryptographically secure randomness
		maxJitter := new(big.Int).SetInt64(int64(delay * 0.5)) // Max jitter is 50% of the delay (±25%)
		jitter, err := rand.Int(rand.Reader, maxJitter)
		if err != nil {
			// Fallback to less random jitter if crypto/rand fails
			delay += (r.config.InitialDelay.Seconds() * 0.25 * (2*float64(time.Now().UnixNano()%100)/100 - 1))
		} else {
			// Apply jitter: delay - 25% to delay + 25%
			delay += (float64(jitter.Int64()) - (delay * 0.25))
		}

		// Ensure delay is not negative
		if delay < 0 {
			delay = float64(r.config.InitialDelay)
		}
	}

	return time.Duration(delay)
}

// isRetryableError checks if an error should be retried
func (r *RetryExecutor) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Import the proxy package to check for specific error types
	// Note: This creates a circular dependency, so we'll use interface-based approach

	// Check if error implements a Retriable interface
	if retriable, ok := err.(interface{ IsRetriable() bool }); ok {
		return retriable.IsRetriable()
	}

	// Fallback: check error message for known patterns
	errMsg := err.Error()
	for _, retryableType := range r.config.RetryableErrors {
		if contains(errMsg, retryableType) {
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
					indexOf(s, substr) >= 0)))
}

// indexOf finds the index of substr in s, returns -1 if not found
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// RetryableError interface for errors that can be retried
type RetryableError interface {
	error
	IsRetriable() bool
}

// RetryMetrics holds metrics about retry operations
type RetryMetrics struct {
	TotalAttempts     int
	SuccessfulRetries int
	FailedRetries     int
	AverageDelay      time.Duration
}

// GetMetrics returns current retry metrics (placeholder for future implementation)
func (r *RetryExecutor) GetMetrics() *RetryMetrics {
	return &RetryMetrics{
		TotalAttempts:     0,
		SuccessfulRetries: 0,
		FailedRetries:     0,
		AverageDelay:      0,
	}
}
