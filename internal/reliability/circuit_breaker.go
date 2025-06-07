package reliability

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aashari/go-generative-api-router/internal/logger"
)

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	// StateClosed - circuit is closed, requests are allowed
	StateClosed CircuitState = iota
	// StateOpen - circuit is open, requests are rejected
	StateOpen
	// StateHalfOpen - circuit is testing if service has recovered
	StateHalfOpen
)

// String returns the string representation of the circuit state
func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreakerConfig defines configuration for circuit breaker behavior
type CircuitBreakerConfig struct {
	Name                string
	MaxFailures         int
	ResetTimeout        time.Duration
	FailureThreshold    float64
	MinRequestThreshold int
	MaxConcurrentCalls  int
}

// DefaultCircuitBreakerConfig returns a sensible default configuration
func DefaultCircuitBreakerConfig(name string) CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Name:                name,
		MaxFailures:         5,
		ResetTimeout:        30 * time.Second,
		FailureThreshold:    0.6, // 60% failure rate
		MinRequestThreshold: 3,   // Minimum requests before calculating failure rate
		MaxConcurrentCalls:  100,
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	config           CircuitBreakerConfig
	state            CircuitState
	failureCount     int
	successCount     int
	requestCount     int
	lastFailureTime  time.Time
	nextRetryTime    time.Time
	concurrentCalls  int
	mutex            sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		config: config,
		state:  StateClosed,
	}
}

// Execute executes an operation through the circuit breaker
func (cb *CircuitBreaker) Execute(ctx context.Context, operation func() error) error {
	// Check if we can execute the operation
	if err := cb.beforeCall(ctx); err != nil {
		return err
	}

	// Track concurrent calls
	cb.incrementConcurrentCalls()
	defer cb.decrementConcurrentCalls()

	// Execute the operation
	err := operation()

	// Record the result
	cb.afterCall(ctx, err)

	return err
}

// beforeCall checks if the operation can be executed
func (cb *CircuitBreaker) beforeCall(ctx context.Context) error {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	// Check concurrent call limit
	if cb.concurrentCalls >= cb.config.MaxConcurrentCalls {
		logger.WarnCtx(ctx, "Circuit breaker rejecting call due to concurrent limit",
			"circuit_name", cb.config.Name,
			"concurrent_calls", cb.concurrentCalls,
			"max_concurrent", cb.config.MaxConcurrentCalls)
		return fmt.Errorf("circuit breaker %s: too many concurrent calls (%d/%d)",
			cb.config.Name, cb.concurrentCalls, cb.config.MaxConcurrentCalls)
	}

	switch cb.state {
	case StateClosed:
		// Allow the call
		return nil

	case StateOpen:
		// Check if we should transition to half-open
		if time.Now().After(cb.nextRetryTime) {
			cb.state = StateHalfOpen
			logger.InfoCtx(ctx, "Circuit breaker transitioning to half-open",
				"circuit_name", cb.config.Name,
				"failure_count", cb.failureCount)
			return nil
		}
		
		logger.WarnCtx(ctx, "Circuit breaker rejecting call - circuit is open",
			"circuit_name", cb.config.Name,
			"failure_count", cb.failureCount,
			"next_retry", cb.nextRetryTime.Format(time.RFC3339))
		return fmt.Errorf("circuit breaker %s is open", cb.config.Name)

	case StateHalfOpen:
		// Allow limited calls to test if service has recovered
		return nil

	default:
		return fmt.Errorf("circuit breaker %s in unknown state", cb.config.Name)
	}
}

// afterCall records the result of the operation
func (cb *CircuitBreaker) afterCall(ctx context.Context, err error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.requestCount++

	if err != nil {
		cb.onFailure(ctx, err)
	} else {
		cb.onSuccess(ctx)
	}
}

// onFailure handles a failed operation
func (cb *CircuitBreaker) onFailure(ctx context.Context, err error) {
	cb.failureCount++
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateClosed:
		// Check if we should open the circuit
		if cb.shouldOpen() {
			cb.openCircuit(ctx)
		}

	case StateHalfOpen:
		// Failure in half-open state means service hasn't recovered
		cb.openCircuit(ctx)
		logger.WarnCtx(ctx, "Circuit breaker reopening due to failure in half-open state",
			"circuit_name", cb.config.Name,
			"error", err)
	}
}

// onSuccess handles a successful operation
func (cb *CircuitBreaker) onSuccess(ctx context.Context) {
	cb.successCount++

	switch cb.state {
	case StateHalfOpen:
		// Success in half-open state means service has recovered
		cb.closeCircuit(ctx)

	case StateClosed:
		// Reset failure count on success
		if cb.failureCount > 0 {
			logger.DebugCtx(ctx, "Circuit breaker resetting failure count after success",
				"circuit_name", cb.config.Name,
				"previous_failures", cb.failureCount)
			cb.failureCount = 0
		}
	}
}

// shouldOpen determines if the circuit should be opened
func (cb *CircuitBreaker) shouldOpen() bool {
	// Need minimum number of requests before considering opening
	if cb.requestCount < cb.config.MinRequestThreshold {
		return false
	}

	// Check failure threshold
	failureRate := float64(cb.failureCount) / float64(cb.requestCount)
	return failureRate >= cb.config.FailureThreshold || cb.failureCount >= cb.config.MaxFailures
}

// openCircuit transitions the circuit to open state
func (cb *CircuitBreaker) openCircuit(ctx context.Context) {
	cb.state = StateOpen
	cb.nextRetryTime = time.Now().Add(cb.config.ResetTimeout)
	
	logger.ErrorCtx(ctx, "Circuit breaker opened",
		"circuit_name", cb.config.Name,
		"failure_count", cb.failureCount,
		"request_count", cb.requestCount,
		"failure_rate", float64(cb.failureCount)/float64(cb.requestCount),
		"next_retry", cb.nextRetryTime.Format(time.RFC3339))
}

// closeCircuit transitions the circuit to closed state
func (cb *CircuitBreaker) closeCircuit(ctx context.Context) {
	previousState := cb.state
	cb.state = StateClosed
	cb.failureCount = 0
	cb.requestCount = 0
	cb.successCount = 0
	
	logger.InfoCtx(ctx, "Circuit breaker closed",
		"circuit_name", cb.config.Name,
		"previous_state", previousState.String())
}

// incrementConcurrentCalls safely increments the concurrent call counter
func (cb *CircuitBreaker) incrementConcurrentCalls() {
	cb.mutex.Lock()
	cb.concurrentCalls++
	cb.mutex.Unlock()
}

// decrementConcurrentCalls safely decrements the concurrent call counter
func (cb *CircuitBreaker) decrementConcurrentCalls() {
	cb.mutex.Lock()
	cb.concurrentCalls--
	cb.mutex.Unlock()
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// GetStats returns statistics about the circuit breaker
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	failureRate := 0.0
	if cb.requestCount > 0 {
		failureRate = float64(cb.failureCount) / float64(cb.requestCount)
	}

	return map[string]interface{}{
		"name":             cb.config.Name,
		"state":            cb.state.String(),
		"failure_count":    cb.failureCount,
		"success_count":    cb.successCount,
		"request_count":    cb.requestCount,
		"failure_rate":     failureRate,
		"concurrent_calls": cb.concurrentCalls,
		"last_failure":     cb.lastFailureTime.Format(time.RFC3339),
		"next_retry":       cb.nextRetryTime.Format(time.RFC3339),
	}
}

// Reset resets the circuit breaker to its initial state
func (cb *CircuitBreaker) Reset(ctx context.Context) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	previousState := cb.state
	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.requestCount = 0
	cb.concurrentCalls = 0
	cb.lastFailureTime = time.Time{}
	cb.nextRetryTime = time.Time{}

	logger.InfoCtx(ctx, "Circuit breaker reset",
		"circuit_name", cb.config.Name,
		"previous_state", previousState.String())
}

// CircuitBreakerManager manages multiple circuit breakers
type CircuitBreakerManager struct {
	breakers map[string]*CircuitBreaker
	mutex    sync.RWMutex
}

// NewCircuitBreakerManager creates a new circuit breaker manager
func NewCircuitBreakerManager() *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers: make(map[string]*CircuitBreaker),
	}
}

// GetOrCreate gets an existing circuit breaker or creates a new one
func (cbm *CircuitBreakerManager) GetOrCreate(name string, config CircuitBreakerConfig) *CircuitBreaker {
	cbm.mutex.Lock()
	defer cbm.mutex.Unlock()

	if cb, exists := cbm.breakers[name]; exists {
		return cb
	}

	config.Name = name
	cb := NewCircuitBreaker(config)
	cbm.breakers[name] = cb
	return cb
}

// GetStats returns statistics for all circuit breakers
func (cbm *CircuitBreakerManager) GetStats() map[string]interface{} {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()

	stats := make(map[string]interface{})
	for name, cb := range cbm.breakers {
		stats[name] = cb.GetStats()
	}
	return stats
}

// Global circuit breaker manager
var globalCBManager = NewCircuitBreakerManager()

// GetCircuitBreaker gets or creates a circuit breaker with default configuration
func GetCircuitBreaker(name string) *CircuitBreaker {
	return globalCBManager.GetOrCreate(name, DefaultCircuitBreakerConfig(name))
}

// GetCircuitBreakerWithConfig gets or creates a circuit breaker with custom configuration
func GetCircuitBreakerWithConfig(name string, config CircuitBreakerConfig) *CircuitBreaker {
	return globalCBManager.GetOrCreate(name, config)
}

// GetAllCircuitBreakerStats returns statistics for all circuit breakers
func GetAllCircuitBreakerStats() map[string]interface{} {
	return globalCBManager.GetStats()
} 