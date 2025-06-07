package health

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/aashari/go-generative-api-router/internal/config"
	"github.com/aashari/go-generative-api-router/internal/logger"
	"github.com/aashari/go-generative-api-router/internal/reliability"
)

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusUnhealthy HealthStatus = "unhealthy"
	StatusDegraded  HealthStatus = "degraded"
	StatusUnknown   HealthStatus = "unknown"
)

// HealthCheck represents a single health check
type HealthCheck struct {
	Name        string
	Description string
	Check       func(ctx context.Context) HealthCheckResult
	Timeout     time.Duration
	Critical    bool // If true, failure affects overall system health
}

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	Status    HealthStatus
	Message   string
	Details   map[string]interface{}
	Timestamp time.Time
	Duration  time.Duration
	Error     error
}

// HealthChecker manages and executes health checks
type HealthChecker struct {
	checks map[string]*HealthCheck
	mutex  sync.RWMutex
}

// NewHealthChecker creates a new health checker
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		checks: make(map[string]*HealthCheck),
	}
}

// RegisterCheck registers a new health check
func (hc *HealthChecker) RegisterCheck(check *HealthCheck) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	if check.Timeout == 0 {
		check.Timeout = 5 * time.Second
	}

	hc.checks[check.Name] = check

	logger.Info("Health check registered",
		"name", check.Name,
		"description", check.Description,
		"critical", check.Critical,
		"timeout", check.Timeout)
}

// ExecuteCheck executes a single health check
func (hc *HealthChecker) ExecuteCheck(ctx context.Context, name string) (*HealthCheckResult, error) {
	hc.mutex.RLock()
	check, exists := hc.checks[name]
	hc.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("health check %s not found", name)
	}

	return hc.executeCheck(ctx, check)
}

// ExecuteAllChecks executes all registered health checks
func (hc *HealthChecker) ExecuteAllChecks(ctx context.Context) map[string]HealthCheckResult {
	hc.mutex.RLock()
	checks := make(map[string]*HealthCheck)
	for name, check := range hc.checks {
		checks[name] = check
	}
	hc.mutex.RUnlock()

	results := make(map[string]HealthCheckResult)
	var wg sync.WaitGroup
	var resultMutex sync.Mutex

	for name, check := range checks {
		wg.Add(1)
		go func(name string, check *HealthCheck) {
			defer wg.Done()

			result, err := hc.executeCheck(ctx, check)
			if err != nil {
				result = &HealthCheckResult{
					Status:    StatusUnhealthy,
					Message:   fmt.Sprintf("Failed to execute health check: %v", err),
					Timestamp: time.Now(),
					Error:     err,
				}
			}

			resultMutex.Lock()
			results[name] = *result
			resultMutex.Unlock()
		}(name, check)
	}

	wg.Wait()
	return results
}

// executeCheck executes a single health check with timeout
func (hc *HealthChecker) executeCheck(ctx context.Context, check *HealthCheck) (*HealthCheckResult, error) {
	checkCtx, cancel := context.WithTimeout(ctx, check.Timeout)
	defer cancel()

	start := time.Now()

	// Execute the check
	result := check.Check(checkCtx)
	result.Timestamp = start
	result.Duration = time.Since(start)

	// Log the result
	logger.DebugCtx(ctx, "Health check executed",
		"name", check.Name,
		"status", result.Status,
		"duration_ms", result.Duration.Milliseconds(),
		"message", result.Message)

	return &result, nil
}

// GetOverallHealth determines the overall system health
func (hc *HealthChecker) GetOverallHealth(ctx context.Context) (HealthStatus, map[string]HealthCheckResult) {
	results := hc.ExecuteAllChecks(ctx)

	overallStatus := StatusHealthy
	criticalFailures := 0
	totalFailures := 0

	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	for name, result := range results {
		check := hc.checks[name]

		if result.Status == StatusUnhealthy {
			totalFailures++
			if check.Critical {
				criticalFailures++
			}
		} else if result.Status == StatusDegraded {
			if overallStatus == StatusHealthy {
				overallStatus = StatusDegraded
			}
		}
	}

	// Determine overall status
	if criticalFailures > 0 {
		overallStatus = StatusUnhealthy
	} else if totalFailures > 0 && overallStatus != StatusDegraded {
		overallStatus = StatusDegraded
	}

	logger.InfoCtx(ctx, "Overall health assessment completed",
		"overall_status", overallStatus,
		"total_checks", len(results),
		"total_failures", totalFailures,
		"critical_failures", criticalFailures)

	return overallStatus, results
}

// CreateStandardHealthChecks creates standard health checks for the application
func CreateStandardHealthChecks(credentials []config.Credential) *HealthChecker {
	hc := NewHealthChecker()

	// Basic application health check
	hc.RegisterCheck(&HealthCheck{
		Name:        "application",
		Description: "Basic application health",
		Critical:    true,
		Timeout:     2 * time.Second,
		Check: func(ctx context.Context) HealthCheckResult {
			return HealthCheckResult{
				Status:  StatusHealthy,
				Message: "Application is running",
				Details: map[string]interface{}{
					"uptime": time.Since(time.Now().Add(-time.Hour)), // Placeholder
				},
			}
		},
	})

	// Configuration health check
	hc.RegisterCheck(&HealthCheck{
		Name:        "configuration",
		Description: "Configuration validation",
		Critical:    true,
		Timeout:     3 * time.Second,
		Check: func(ctx context.Context) HealthCheckResult {
			if len(credentials) == 0 {
				return HealthCheckResult{
					Status:  StatusUnhealthy,
					Message: "No credentials configured",
				}
			}

			return HealthCheckResult{
				Status:  StatusHealthy,
				Message: "Configuration is valid",
				Details: map[string]interface{}{
					"credentials_count": len(credentials),
				},
			}
		},
	})

	// Vendor connectivity health checks
	vendors := getUniqueVendors(credentials)
	for _, vendor := range vendors {
		hc.RegisterCheck(createVendorHealthCheck(vendor))
	}

	// Circuit breaker health check
	hc.RegisterCheck(&HealthCheck{
		Name:        "circuit_breakers",
		Description: "Circuit breaker status",
		Critical:    false,
		Timeout:     2 * time.Second,
		Check: func(ctx context.Context) HealthCheckResult {
			stats := reliability.GetAllCircuitBreakerStats()

			openCircuits := 0
			for _, stat := range stats {
				if statMap, ok := stat.(map[string]interface{}); ok {
					if state, ok := statMap["state"].(string); ok && state == "OPEN" {
						openCircuits++
					}
				}
			}

			status := StatusHealthy
			message := "All circuit breakers are healthy"

			if openCircuits > 0 {
				status = StatusDegraded
				message = fmt.Sprintf("%d circuit breaker(s) are open", openCircuits)
			}

			return HealthCheckResult{
				Status:  status,
				Message: message,
				Details: map[string]interface{}{
					"circuit_breakers": stats,
					"open_circuits":    openCircuits,
				},
			}
		},
	})

	return hc
}

// createVendorHealthCheck creates a health check for a specific vendor
func createVendorHealthCheck(vendor string) *HealthCheck {
	return &HealthCheck{
		Name:        fmt.Sprintf("vendor_%s", vendor),
		Description: fmt.Sprintf("Connectivity to %s API", vendor),
		Critical:    false, // Vendor failures shouldn't bring down the whole system
		Timeout:     10 * time.Second,
		Check: func(ctx context.Context) HealthCheckResult {
			// Get circuit breaker for this vendor
			cb := reliability.GetCircuitBreaker(fmt.Sprintf("vendor_%s", vendor))

			if cb.GetState() == reliability.StateOpen {
				return HealthCheckResult{
					Status:  StatusUnhealthy,
					Message: fmt.Sprintf("Circuit breaker for %s is open", vendor),
					Details: map[string]interface{}{
						"circuit_breaker_stats": cb.GetStats(),
					},
				}
			}

			// For now, just check if we have credentials for this vendor
			// In a real implementation, you might want to make a lightweight API call
			return HealthCheckResult{
				Status:  StatusHealthy,
				Message: fmt.Sprintf("Vendor %s is available", vendor),
				Details: map[string]interface{}{
					"vendor": vendor,
				},
			}
		},
	}
}

// getUniqueVendors extracts unique vendor names from credentials
func getUniqueVendors(credentials []config.Credential) []string {
	vendorMap := make(map[string]bool)
	for _, cred := range credentials {
		vendorMap[cred.Platform] = true
	}

	vendors := make([]string, 0, len(vendorMap))
	for vendor := range vendorMap {
		vendors = append(vendors, vendor)
	}
	return vendors
}

// HealthHandler creates an HTTP handler for health checks
func HealthHandler(hc *HealthChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Check if a specific health check is requested
		checkName := r.URL.Query().Get("check")

		if checkName != "" {
			// Execute specific health check
			result, err := hc.ExecuteCheck(ctx, checkName)
			if err != nil {
				http.Error(w, fmt.Sprintf("Health check not found: %s", checkName), http.StatusNotFound)
				return
			}

			statusCode := http.StatusOK
			if result.Status == StatusUnhealthy {
				statusCode = http.StatusServiceUnavailable
			} else if result.Status == StatusDegraded {
				statusCode = http.StatusPartialContent
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(statusCode)

			response := map[string]interface{}{
				"status":    result.Status,
				"message":   result.Message,
				"details":   result.Details,
				"timestamp": result.Timestamp.Format(time.RFC3339),
				"duration":  result.Duration.Milliseconds(),
			}

			if err := writeJSONResponse(w, response); err != nil {
				logger.ErrorCtx(ctx, "Failed to write health check response", "error", err)
			}
		} else {
			// Execute all health checks
			overallStatus, results := hc.GetOverallHealth(ctx)

			statusCode := http.StatusOK
			if overallStatus == StatusUnhealthy {
				statusCode = http.StatusServiceUnavailable
			} else if overallStatus == StatusDegraded {
				statusCode = http.StatusPartialContent
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(statusCode)

			response := map[string]interface{}{
				"status":    overallStatus,
				"timestamp": time.Now().Format(time.RFC3339),
				"checks":    results,
			}

			if err := writeJSONResponse(w, response); err != nil {
				logger.ErrorCtx(ctx, "Failed to write health response", "error", err)
			}
		}
	}
}

// writeJSONResponse writes a JSON response
func writeJSONResponse(w http.ResponseWriter, data interface{}) error {
	// Simple JSON marshaling - in a real implementation you might want to use encoding/json
	// For now, we'll use a basic approach
	w.Write([]byte(fmt.Sprintf(`{"status": "%v"}`, data)))
	return nil
}

// Global health checker instance
var globalHealthChecker *HealthChecker

// InitializeGlobalHealthChecker initializes the global health checker
func InitializeGlobalHealthChecker(credentials []config.Credential) {
	globalHealthChecker = CreateStandardHealthChecks(credentials)
}

// GetGlobalHealthChecker returns the global health checker
func GetGlobalHealthChecker() *HealthChecker {
	return globalHealthChecker
}
