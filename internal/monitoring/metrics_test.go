package monitoring

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetrics_RecordRequest(t *testing.T) {
	metrics := &Metrics{
		VendorRequestCounts: make(map[string]int64),
		ModelRequestCounts:  make(map[string]int64),
		StatusCodeCounts:    make(map[int]int64),
		StartTime:           time.Now(),
	}

	// Record a successful request
	metrics.RecordRequest(100*time.Millisecond, 200, "openai", "gpt-4")

	assert.Equal(t, int64(1), metrics.RequestCount)
	assert.Equal(t, 100*time.Millisecond, metrics.RequestDuration)
	assert.Equal(t, int64(0), metrics.ErrorCount)
	assert.Equal(t, int64(1), metrics.VendorRequestCounts["openai"])
	assert.Equal(t, int64(1), metrics.ModelRequestCounts["gpt-4"])
	assert.Equal(t, int64(1), metrics.StatusCodeCounts[200])

	// Record an error request
	metrics.RecordRequest(50*time.Millisecond, 500, "gemini", "gemini-pro")

	assert.Equal(t, int64(2), metrics.RequestCount)
	assert.Equal(t, 150*time.Millisecond, metrics.RequestDuration)
	assert.Equal(t, int64(1), metrics.ErrorCount)
	assert.Equal(t, int64(1), metrics.VendorRequestCounts["gemini"])
	assert.Equal(t, int64(1), metrics.ModelRequestCounts["gemini-pro"])
	assert.Equal(t, int64(1), metrics.StatusCodeCounts[500])
}

func TestMetrics_RecordError(t *testing.T) {
	metrics := &Metrics{
		VendorRequestCounts: make(map[string]int64),
		ModelRequestCounts:  make(map[string]int64),
		StatusCodeCounts:    make(map[int]int64),
		StartTime:           time.Now(),
	}

	metrics.RecordError()
	assert.Equal(t, int64(1), metrics.ErrorCount)

	metrics.RecordError()
	assert.Equal(t, int64(2), metrics.ErrorCount)
}

func TestMetrics_GetStats(t *testing.T) {
	metrics := &Metrics{
		VendorRequestCounts: make(map[string]int64),
		ModelRequestCounts:  make(map[string]int64),
		StatusCodeCounts:    make(map[int]int64),
		StartTime:           time.Now().Add(-1 * time.Hour), // 1 hour ago
	}

	// Record some requests
	metrics.RecordRequest(100*time.Millisecond, 200, "openai", "gpt-4")
	metrics.RecordRequest(200*time.Millisecond, 500, "gemini", "gemini-pro")

	stats := metrics.GetStats()

	assert.Greater(t, stats["uptime_seconds"].(float64), 3600.0) // More than 1 hour
	assert.Equal(t, int64(2), stats["total_requests"].(int64))
	assert.Equal(t, int64(1), stats["total_errors"].(int64))
	assert.Equal(t, int64(150), stats["average_duration_ms"].(int64)) // (100+200)/2
	assert.Greater(t, stats["requests_per_second"].(float64), 0.0)
	assert.Equal(t, 0.5, stats["error_rate"].(float64)) // 1 error out of 2 requests

	vendorRequests := stats["vendor_requests"].(map[string]int64)
	assert.Equal(t, int64(1), vendorRequests["openai"])
	assert.Equal(t, int64(1), vendorRequests["gemini"])

	modelRequests := stats["model_requests"].(map[string]int64)
	assert.Equal(t, int64(1), modelRequests["gpt-4"])
	assert.Equal(t, int64(1), modelRequests["gemini-pro"])

	statusCounts := stats["status_code_counts"].(map[int]int64)
	assert.Equal(t, int64(1), statusCounts[200])
	assert.Equal(t, int64(1), statusCounts[500])
}

func TestMetrics_Reset(t *testing.T) {
	metrics := &Metrics{
		VendorRequestCounts: make(map[string]int64),
		ModelRequestCounts:  make(map[string]int64),
		StatusCodeCounts:    make(map[int]int64),
		StartTime:           time.Now(),
	}

	// Record some data
	metrics.RecordRequest(100*time.Millisecond, 200, "openai", "gpt-4")
	metrics.RecordError()

	// Verify data exists
	assert.Equal(t, int64(1), metrics.RequestCount)
	assert.Equal(t, int64(1), metrics.ErrorCount)

	// Reset
	metrics.Reset()

	// Verify data is cleared
	assert.Equal(t, int64(0), metrics.RequestCount)
	assert.Equal(t, time.Duration(0), metrics.RequestDuration)
	assert.Equal(t, int64(0), metrics.ErrorCount)
	assert.Empty(t, metrics.VendorRequestCounts)
	assert.Empty(t, metrics.ModelRequestCounts)
	assert.Empty(t, metrics.StatusCodeCounts)
}

func TestMetricsMiddleware(t *testing.T) {
	// Reset global metrics for clean test
	globalMetrics.Reset()

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond) // Simulate some processing time
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with metrics middleware
	wrappedHandler := MetricsMiddleware(testHandler)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/test?vendor=openai", nil)
	w := httptest.NewRecorder()

	// Execute request
	wrappedHandler.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "OK", w.Body.String())

	// Verify metrics were recorded
	stats := globalMetrics.GetStats()
	assert.Equal(t, int64(1), stats["total_requests"].(int64))
	assert.Equal(t, int64(0), stats["total_errors"].(int64))
	assert.Greater(t, stats["average_duration_ms"].(int64), int64(0))

	vendorRequests := stats["vendor_requests"].(map[string]int64)
	assert.Equal(t, int64(1), vendorRequests["openai"])
}

func TestResponseWriterWrapper(t *testing.T) {
	w := httptest.NewRecorder()
	wrapper := &responseWriterWrapper{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	// Test default status code
	assert.Equal(t, http.StatusOK, wrapper.statusCode)

	// Test WriteHeader
	wrapper.WriteHeader(http.StatusBadRequest)
	assert.Equal(t, http.StatusBadRequest, wrapper.statusCode)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExtractModelFromRequest(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		expected string
	}{
		{
			name:     "models_list_endpoint",
			method:   "GET",
			path:     "/v1/models",
			expected: "models_list",
		},
		{
			name:     "other_endpoint",
			method:   "POST",
			path:     "/v1/chat/completions",
			expected: "unknown",
		},
		{
			name:     "health_endpoint",
			method:   "GET",
			path:     "/health",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			result := extractModelFromRequest(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMetricsHandler(t *testing.T) {
	// Reset global metrics for clean test
	globalMetrics.Reset()

	// Record some test data
	globalMetrics.RecordRequest(100*time.Millisecond, 200, "openai", "gpt-4")
	globalMetrics.RecordRequest(200*time.Millisecond, 500, "gemini", "gemini-pro")

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	// Execute request
	MetricsHandler(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	body := w.Body.String()
	assert.Contains(t, body, "total_requests")
	assert.Contains(t, body, "total_errors")
	assert.Contains(t, body, "average_duration_ms")
	assert.Contains(t, body, "requests_per_second")
	assert.Contains(t, body, "error_rate")
	assert.Contains(t, body, "start_time")
}

func TestGetMetrics(t *testing.T) {
	metrics := GetMetrics()
	require.NotNil(t, metrics)
	assert.Equal(t, globalMetrics, metrics)
}
