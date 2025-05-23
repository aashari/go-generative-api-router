package monitoring

import (
	"log"
	"net/http"
	"net/http/pprof"
	"strconv"
	"sync"
	"time"
)

// Metrics holds application metrics
type Metrics struct {
	mu                    sync.RWMutex
	RequestCount          int64
	RequestDuration       time.Duration
	ErrorCount            int64
	VendorRequestCounts   map[string]int64
	ModelRequestCounts    map[string]int64
	StatusCodeCounts      map[int]int64
	StartTime             time.Time
}

// Global metrics instance
var globalMetrics = &Metrics{
	VendorRequestCounts: make(map[string]int64),
	ModelRequestCounts:  make(map[string]int64),
	StatusCodeCounts:    make(map[int]int64),
	StartTime:           time.Now(),
}

// GetMetrics returns the global metrics instance
func GetMetrics() *Metrics {
	return globalMetrics
}

// RecordRequest records a request with its duration and status
func (m *Metrics) RecordRequest(duration time.Duration, statusCode int, vendor, model string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RequestCount++
	m.RequestDuration += duration
	m.StatusCodeCounts[statusCode]++

	if vendor != "" {
		m.VendorRequestCounts[vendor]++
	}

	if model != "" {
		m.ModelRequestCounts[model]++
	}

	if statusCode >= 400 {
		m.ErrorCount++
	}
}

// RecordError records an error
func (m *Metrics) RecordError() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ErrorCount++
}

// GetStats returns current statistics
func (m *Metrics) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	uptime := time.Since(m.StartTime)
	avgDuration := time.Duration(0)
	if m.RequestCount > 0 {
		avgDuration = m.RequestDuration / time.Duration(m.RequestCount)
	}

	// Copy maps to avoid race conditions
	vendorCounts := make(map[string]int64)
	for k, v := range m.VendorRequestCounts {
		vendorCounts[k] = v
	}

	modelCounts := make(map[string]int64)
	for k, v := range m.ModelRequestCounts {
		modelCounts[k] = v
	}

	statusCounts := make(map[int]int64)
	for k, v := range m.StatusCodeCounts {
		statusCounts[k] = v
	}

	return map[string]interface{}{
		"uptime_seconds":        uptime.Seconds(),
		"total_requests":        m.RequestCount,
		"total_errors":          m.ErrorCount,
		"average_duration_ms":   avgDuration.Milliseconds(),
		"requests_per_second":   float64(m.RequestCount) / uptime.Seconds(),
		"error_rate":           float64(m.ErrorCount) / float64(m.RequestCount),
		"vendor_requests":      vendorCounts,
		"model_requests":       modelCounts,
		"status_code_counts":   statusCounts,
		"start_time":           m.StartTime.Format(time.RFC3339),
	}
}

// Reset resets all metrics (useful for testing)
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RequestCount = 0
	m.RequestDuration = 0
	m.ErrorCount = 0
	m.VendorRequestCounts = make(map[string]int64)
	m.ModelRequestCounts = make(map[string]int64)
	m.StatusCodeCounts = make(map[int]int64)
	m.StartTime = time.Now()
}

// MetricsMiddleware wraps HTTP handlers to collect metrics
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Create a response writer wrapper to capture status code
		wrapper := &responseWriterWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // Default to 200
		}

		// Call the next handler
		next.ServeHTTP(wrapper, r)

		// Record metrics
		duration := time.Since(start)
		vendor := r.URL.Query().Get("vendor")
		
		// Try to extract model from request (this is a simplified approach)
		model := extractModelFromRequest(r)

		globalMetrics.RecordRequest(duration, wrapper.statusCode, vendor, model)

		// Log request for debugging
		log.Printf("Request: %s %s - Status: %d - Duration: %v - Vendor: %s - Model: %s",
			r.Method, r.URL.Path, wrapper.statusCode, duration, vendor, model)
	})
}

// responseWriterWrapper wraps http.ResponseWriter to capture status code
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// extractModelFromRequest attempts to extract model from request
func extractModelFromRequest(r *http.Request) string {
	// This is a simplified approach - in a real implementation,
	// you might want to parse the request body for POST requests
	if r.Method == "GET" && r.URL.Path == "/v1/models" {
		return "models_list"
	}
	return "unknown"
}

// SetupPprofRoutes adds pprof endpoints to the router
func SetupPprofRoutes(mux *http.ServeMux) {
	// Add pprof endpoints for performance profiling
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	mux.Handle("/debug/pprof/block", pprof.Handler("block"))
}

// MetricsHandler returns current metrics as JSON
func MetricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	stats := globalMetrics.GetStats()
	
	// Simple JSON encoding (you could use json.Marshal for more complex cases)
	response := "{\n"
	response += "  \"uptime_seconds\": " + strconv.FormatFloat(stats["uptime_seconds"].(float64), 'f', 2, 64) + ",\n"
	response += "  \"total_requests\": " + strconv.FormatInt(stats["total_requests"].(int64), 10) + ",\n"
	response += "  \"total_errors\": " + strconv.FormatInt(stats["total_errors"].(int64), 10) + ",\n"
	response += "  \"average_duration_ms\": " + strconv.FormatInt(stats["average_duration_ms"].(int64), 10) + ",\n"
	response += "  \"requests_per_second\": " + strconv.FormatFloat(stats["requests_per_second"].(float64), 'f', 2, 64) + ",\n"
	response += "  \"error_rate\": " + strconv.FormatFloat(stats["error_rate"].(float64), 'f', 4, 64) + ",\n"
	response += "  \"start_time\": \"" + stats["start_time"].(string) + "\"\n"
	response += "}"
	
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(response))
} 