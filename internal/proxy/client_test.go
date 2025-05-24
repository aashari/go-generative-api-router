package proxy

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponseStandardizer_SetCompliantHeaders(t *testing.T) {
	tests := []struct {
		name            string
		vendor          string
		contentLength   int
		isCompressed    bool
		expectedHeaders map[string]string
	}{
		{
			name:          "OpenAI vendor - standard headers",
			vendor:        "openai",
			contentLength: 1234,
			isCompressed:  false,
			expectedHeaders: map[string]string{
				"Content-Type":                  "application/json; charset=utf-8",
				"Content-Length":                "1234",
				"Server":                        "Generative-API-Router/1.0",
				"X-Powered-By":                  "Generative-API-Router",
				"X-Vendor-Source":               "openai",
				"Cache-Control":                 "no-cache, no-store, must-revalidate",
				"X-Content-Type-Options":        "nosniff",
				"X-Frame-Options":               "DENY",
				"X-XSS-Protection":              "1; mode=block",
				"Referrer-Policy":               "strict-origin-when-cross-origin",
				"Access-Control-Allow-Origin":   "*",
				"Access-Control-Allow-Methods":  "POST, GET, OPTIONS, PUT, DELETE",
				"Access-Control-Allow-Headers":  "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization",
				"Access-Control-Expose-Headers": "X-Request-ID, X-Response-Time",
			},
		},
		{
			name:          "Gemini vendor - with compression",
			vendor:        "gemini",
			contentLength: 5678,
			isCompressed:  true,
			expectedHeaders: map[string]string{
				"Content-Type":     "application/json; charset=utf-8",
				"Content-Length":   "5678",
				"Content-Encoding": "gzip",
				"Vary":             "Accept-Encoding",
				"Server":           "Generative-API-Router/1.0",
				"X-Powered-By":     "Generative-API-Router",
				"X-Vendor-Source":  "gemini",
			},
		},
		{
			name:          "Unknown vendor",
			vendor:        "anthropic",
			contentLength: 100,
			isCompressed:  false,
			expectedHeaders: map[string]string{
				"X-Vendor-Source": "anthropic",
				"Server":          "Generative-API-Router/1.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test response writer
			w := httptest.NewRecorder()

			// Create standardizer
			standardizer := NewResponseStandardizer()

			// Call setCompliantHeaders
			standardizer.setCompliantHeaders(w, tt.vendor, tt.contentLength, tt.isCompressed)

			// Check expected headers
			for key, expectedValue := range tt.expectedHeaders {
				actualValue := w.Header().Get(key)
				assert.Equal(t, expectedValue, actualValue, "Header %s mismatch", key)
			}

			// Check Date header is present and valid
			dateHeader := w.Header().Get("Date")
			assert.NotEmpty(t, dateHeader)
			_, err := time.Parse(http.TimeFormat, dateHeader)
			assert.NoError(t, err, "Date header should be in valid HTTP format")

			// Check X-Request-ID is present and has correct format
			requestID := w.Header().Get("X-Request-ID")
			assert.NotEmpty(t, requestID, "X-Request-ID header should not be empty")
			assert.True(t, strings.HasPrefix(requestID, "req_"), "X-Request-ID '%s' should start with 'req_' (not 'req-')", requestID)
		})
	}
}

func TestResponseStandardizer_ProcessResponseBody(t *testing.T) {
	tests := []struct {
		name            string
		vendor          string
		responseBody    string
		isGzipped       bool
		contentEncoding string
		expectError     bool
	}{
		{
			name:            "Plain text response",
			vendor:          "openai",
			responseBody:    `{"id":"chatcmpl-123","choices":[{"message":{"content":"Hello"}}]}`,
			isGzipped:       false,
			contentEncoding: "",
			expectError:     false,
		},
		{
			name:            "Gzipped response",
			vendor:          "gemini",
			responseBody:    `{"candidates":[{"content":{"parts":[{"text":"Hello"}]}}]}`,
			isGzipped:       true,
			contentEncoding: "gzip",
			expectError:     false,
		},
		{
			name:            "Gzip header but plain body",
			vendor:          "openai",
			responseBody:    `{"error": "test"}`,
			isGzipped:       false,
			contentEncoding: "gzip",
			expectError:     true, // Should fail to decompress
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			standardizer := NewResponseStandardizer()

			// Prepare response body
			var body io.Reader
			if tt.isGzipped {
				var buf bytes.Buffer
				gz := gzip.NewWriter(&buf)
				_, err := gz.Write([]byte(tt.responseBody))
				require.NoError(t, err)
				err = gz.Close()
				require.NoError(t, err)
				body = &buf
			} else {
				body = strings.NewReader(tt.responseBody)
			}

			// Process response body
			result, err := standardizer.processResponseBody(body, tt.contentEncoding, tt.vendor)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.responseBody, string(result))
			}
		})
	}
}

func TestResponseStandardizer_ShouldCompress(t *testing.T) {
	tests := []struct {
		name           string
		acceptEncoding string
		userAgent      string
		expectCompress bool
	}{
		{
			name:           "Client accepts gzip",
			acceptEncoding: "gzip, deflate, br",
			userAgent:      "Mozilla/5.0",
			expectCompress: true,
		},
		{
			name:           "Client does not accept gzip",
			acceptEncoding: "deflate, br",
			userAgent:      "Mozilla/5.0",
			expectCompress: false,
		},
		{
			name:           "Postman client - compression disabled",
			acceptEncoding: "gzip, deflate",
			userAgent:      "PostmanRuntime/7.29.0",
			expectCompress: false,
		},
		{
			name:           "Insomnia client - compression disabled",
			acceptEncoding: "gzip",
			userAgent:      "insomnia/2021.4.1",
			expectCompress: false,
		},
		{
			name:           "Empty Accept-Encoding",
			acceptEncoding: "",
			userAgent:      "CustomClient/1.0",
			expectCompress: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			standardizer := NewResponseStandardizer()

			// Create test request
			req := httptest.NewRequest("POST", "/v1/chat/completions", nil)
			req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			req.Header.Set("User-Agent", tt.userAgent)

			// Test shouldCompress
			result := standardizer.shouldCompress(req)
			assert.Equal(t, tt.expectCompress, result)
		})
	}
}

func TestResponseStandardizer_CompressResponseMandatory(t *testing.T) {
	standardizer := NewResponseStandardizer()

	testData := []byte(`{"test": "data", "long_field": "` + strings.Repeat("a", 1000) + `"}`)

	compressed, err := standardizer.compressResponseMandatory(testData)
	require.NoError(t, err)

	// Verify compression actually happened
	assert.Less(t, len(compressed), len(testData), "Compressed data should be smaller")

	// Verify we can decompress it
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	require.NoError(t, err)
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	require.NoError(t, err)

	assert.Equal(t, testData, decompressed, "Decompressed data should match original")
}

func TestResponseStandardizer_ValidateVendorResponse(t *testing.T) {
	tests := []struct {
		name        string
		vendor      string
		response    string
		expectError bool
	}{
		{
			name:   "Valid OpenAI response",
			vendor: "openai",
			response: `{
				"id": "chatcmpl-123",
				"object": "chat.completion",
				"created": 1234567890,
				"model": "gpt-4",
				"choices": [{"index": 0, "message": {"content": "Hello"}}]
			}`,
			expectError: false,
		},
		{
			name:        "Invalid JSON",
			vendor:      "openai",
			response:    `{invalid json`,
			expectError: true,
		},
		{
			name:   "Missing required field - id",
			vendor: "gemini",
			response: `{
				"object": "chat.completion",
				"created": 1234567890,
				"model": "gemini-pro",
				"choices": []
			}`,
			expectError: true,
		},
		{
			name:   "Missing required field - choices",
			vendor: "openai",
			response: `{
				"id": "chatcmpl-123",
				"object": "chat.completion",
				"created": 1234567890,
				"model": "gpt-4"
			}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			standardizer := NewResponseStandardizer()

			err := standardizer.validateVendorResponse([]byte(tt.response), tt.vendor)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAPIClient_SetupResponseHeadersWithVendor(t *testing.T) {
	tests := []struct {
		name         string
		vendor       string
		isStreaming  bool
		checkHeaders map[string]string
	}{
		{
			name:        "Non-streaming response",
			vendor:      "openai",
			isStreaming: false,
			checkHeaders: map[string]string{
				"X-Vendor-Source": "openai",
				"Server":          "Generative-API-Router/1.0",
			},
		},
		{
			name:        "Streaming response",
			vendor:      "gemini",
			isStreaming: true,
			checkHeaders: map[string]string{
				"Content-Type":    "text/event-stream; charset=utf-8",
				"Cache-Control":   "no-cache",
				"Connection":      "keep-alive",
				"X-Vendor-Source": "gemini",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewAPIClient()
			w := httptest.NewRecorder()

			// Create a mock vendor response
			vendorResp := &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{},
			}

			// Call setupResponseHeadersWithVendor
			client.setupResponseHeadersWithVendor(w, vendorResp, tt.isStreaming, tt.vendor)

			// Check headers
			for key, expectedValue := range tt.checkHeaders {
				actualValue := w.Header().Get(key)
				assert.Equal(t, expectedValue, actualValue, "Header %s mismatch", key)
			}

			// For streaming, ensure Content-Length is not set
			if tt.isStreaming {
				assert.Empty(t, w.Header().Get("Content-Length"))
			}

			// Check status code was written
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestAddCustomServiceHeader(t *testing.T) {
	w := httptest.NewRecorder()

	// Test adding custom headers
	AddCustomServiceHeader(w, "X-Custom-Header", "custom-value")
	AddCustomServiceHeader(w, "X-Another-Header", "another-value")

	assert.Equal(t, "custom-value", w.Header().Get("X-Custom-Header"))
	assert.Equal(t, "another-value", w.Header().Get("X-Another-Header"))
}

func TestHeaderStandardization_VendorHeadersCompletelyDiscarded(t *testing.T) {
	tests := []struct {
		name            string
		vendor          string
		vendorHeaders   http.Header
		expectedAbsent  []string
		expectedPresent []string
	}{
		{
			name:   "OpenAI vendor headers completely discarded",
			vendor: "openai",
			vendorHeaders: http.Header{
				// OpenAI specific headers
				"X-Ratelimit-Limit-Tokens":       []string{"60000"},
				"X-Ratelimit-Remaining-Tokens":   []string{"59900"},
				"X-Ratelimit-Reset-Tokens":       []string{"100ms"},
				"X-Ratelimit-Limit-Requests":     []string{"10000"},
				"X-Ratelimit-Remaining-Requests": []string{"9999"},
				"X-Ratelimit-Reset-Requests":     []string{"6s"},
				"Cf-Ray":                         []string{"8e7d8f9g0h1i2j3k-LAX"},
				"Cf-Cache-Status":                []string{"DYNAMIC"},
				"Alt-Svc":                        []string{"h3=\":443\"; ma=86400"},
				// Standard HTTP headers from vendor
				"Server":         []string{"cloudflare"},
				"Date":           []string{"Mon, 23 May 2025 21:00:00 GMT"},
				"Vary":           []string{"Origin"},
				"Content-Type":   []string{"application/json"},
				"Content-Length": []string{"1234"},
				"X-Request-Id":   []string{"req_openai_123456"},
			},
			expectedAbsent: []string{
				"X-Ratelimit-Limit-Tokens",
				"X-Ratelimit-Remaining-Tokens",
				"X-Ratelimit-Reset-Tokens",
				"X-Ratelimit-Limit-Requests",
				"X-Ratelimit-Remaining-Requests",
				"X-Ratelimit-Reset-Requests",
				"Cf-Ray",
				"Cf-Cache-Status",
				"Alt-Svc",
				"Vary", // Vendor's Vary header should be replaced
			},
			expectedPresent: []string{
				"Server", // Our server header
				"X-Vendor-Source",
				"X-Powered-By",
				"Content-Type", // Standard headers we set
				"Date",         // Our date
				"X-Request-ID", // Our request ID
			},
		},
		{
			name:   "Gemini vendor headers completely discarded",
			vendor: "gemini",
			vendorHeaders: http.Header{
				// Gemini/Google specific headers
				"X-Goog-Api-Version":     []string{"v1beta"},
				"X-Goog-Safety-Encoding": []string{"base64"},
				"X-Goog-Safety-Schema":   []string{"1.0.0"},
				"Grpc-Server-Stats-Bin":  []string{"AAABBBCCCDDDeeefff"},
				"X-Goog-Trace-Id":        []string{"abc123def456ghi789"},
				// Standard HTTP headers from vendor
				"Server":        []string{"scaffolding on HTTPServer2"},
				"Vary":          []string{"Origin, X-Origin, Referer"},
				"Cache-Control": []string{"private"},
				"Content-Type":  []string{"application/json; charset=UTF-8"},
			},
			expectedAbsent: []string{
				"X-Goog-Api-Version",
				"X-Goog-Safety-Encoding",
				"X-Goog-Safety-Schema",
				"Grpc-Server-Stats-Bin",
				"X-Goog-Trace-Id",
				// Note: Cache-Control is not here because we set our own
			},
			expectedPresent: []string{
				"Server",          // Our server header, not vendor's
				"X-Vendor-Source", // Should be "gemini"
				"Cache-Control",   // Our cache control settings
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test response writer
			w := httptest.NewRecorder()

			// Create API client and call setupResponseHeadersWithVendor
			client := NewAPIClient()
			vendorResp := &http.Response{
				StatusCode: http.StatusOK,
				Header:     tt.vendorHeaders,
			}

			// This simulates what happens in the actual flow
			client.setupResponseHeadersWithVendor(w, vendorResp, false, tt.vendor)

			// Verify all vendor-specific headers are absent
			for _, header := range tt.expectedAbsent {
				value := w.Header().Get(header)
				assert.Empty(t, value, "Vendor header '%s' should be completely discarded, but found: %s", header, value)
			}

			// Verify our standard headers are present
			for _, header := range tt.expectedPresent {
				value := w.Header().Get(header)
				assert.NotEmpty(t, value, "Standard header '%s' should be present", header)
			}

			// Verify specific header values
			assert.Equal(t, "Generative-API-Router/1.0", w.Header().Get("Server"), "Server header should be our service, not vendor's")
			assert.Equal(t, tt.vendor, w.Header().Get("X-Vendor-Source"), "X-Vendor-Source should match vendor")
			assert.Equal(t, "Generative-API-Router", w.Header().Get("X-Powered-By"))
			assert.Equal(t, "no-cache, no-store, must-revalidate", w.Header().Get("Cache-Control"), "Cache-Control should be our standard value")

			// Verify our Date header is set (not vendor's)
			ourDate := w.Header().Get("Date")
			assert.NotEmpty(t, ourDate)
			assert.NotEqual(t, tt.vendorHeaders.Get("Date"), ourDate, "Date header should be freshly generated, not vendor's")

			// Verify our X-Request-ID format
			ourRequestID := w.Header().Get("X-Request-ID")
			assert.True(t, strings.HasPrefix(ourRequestID, "req_"), "X-Request-ID should use our format")
			assert.NotEqual(t, tt.vendorHeaders.Get("X-Request-Id"), ourRequestID, "X-Request-ID should be ours, not vendor's")
		})
	}
}

func TestHeaderStandardization_ConsistencyAcrossVendors(t *testing.T) {
	vendors := []string{"openai", "gemini", "anthropic"}
	headerSets := make(map[string]http.Header)

	// Collect headers from each vendor
	for _, vendor := range vendors {
		w := httptest.NewRecorder()
		client := NewAPIClient()

		vendorResp := &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				// Simulate different vendor headers
				"X-Vendor-Specific-" + vendor: []string{"should-not-appear"},
				"Server":                      []string{vendor + "-server"},
				"Vary":                        []string{vendor + "-vary"},
			},
		}

		client.setupResponseHeadersWithVendor(w, vendorResp, false, vendor)
		headerSets[vendor] = w.Header()
	}

	// Verify consistency across vendors
	baseVendor := vendors[0]
	baseHeaders := headerSets[baseVendor]

	// These headers should be identical across all vendors
	consistentHeaders := []string{
		"Server",
		"X-Powered-By",
		"Cache-Control",
		"X-Content-Type-Options",
		"X-Frame-Options",
		"Access-Control-Allow-Origin",
		"Access-Control-Allow-Methods",
		"Access-Control-Allow-Headers",
		"Access-Control-Expose-Headers",
		"Content-Type",
	}

	for _, vendor := range vendors[1:] {
		vendorHeaders := headerSets[vendor]

		for _, header := range consistentHeaders {
			baseValue := baseHeaders.Get(header)
			vendorValue := vendorHeaders.Get(header)
			assert.Equal(t, baseValue, vendorValue,
				"Header '%s' should be consistent across vendors. %s: '%s', %s: '%s'",
				header, baseVendor, baseValue, vendor, vendorValue)
		}

		// Verify X-Vendor-Source is different but present
		assert.NotEqual(t, baseHeaders.Get("X-Vendor-Source"), vendorHeaders.Get("X-Vendor-Source"),
			"X-Vendor-Source should be different for each vendor")
		assert.Equal(t, vendor, vendorHeaders.Get("X-Vendor-Source"),
			"X-Vendor-Source should match the vendor name")

		// Verify no vendor-specific headers leaked through
		for key := range vendorHeaders {
			assert.False(t, strings.Contains(key, "X-Vendor-Specific-"),
				"Vendor-specific header '%s' should not be present", key)
		}
	}
}
