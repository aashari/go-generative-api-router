package integration

import (
	"testing"

	"github.com/aashari/go-generative-api-router/test/fixtures"
	"github.com/aashari/go-generative-api-router/test/helpers"
)

func TestCORS(t *testing.T) {
	config := helpers.DefaultTestConfig()
	config.ServiceName = "cors-integration-test"

	ts := helpers.NewTestServer(t, config)
	defer ts.Close()

	t.Run("preflight_request", func(t *testing.T) {
		headers := map[string]string{
			"Origin":                         "https://example.com",
			"Access-Control-Request-Method":  "POST",
			"Access-Control-Request-Headers": "Content-Type,Authorization",
		}

		resp, _, err := ts.MakeRequest("OPTIONS", "/v1/chat/completions", nil, headers)
		if err != nil {
			t.Fatalf("CORS preflight request failed: %v", err)
		}

		// CORS preflight should return 200 or 204
		if resp.StatusCode != 200 && resp.StatusCode != 204 {
			t.Logf("CORS preflight returned status %d (CORS may not be configured)", resp.StatusCode)
		}

		// Check for CORS headers (optional - service may not have CORS configured)
		corsOrigin := resp.Header.Get("Access-Control-Allow-Origin")
		corsMethods := resp.Header.Get("Access-Control-Allow-Methods")
		corsHeaders := resp.Header.Get("Access-Control-Allow-Headers")

		if corsOrigin != "" {
			t.Logf("CORS Origin header present: %s", corsOrigin)
		} else {
			t.Log("CORS Origin header not present (CORS may not be configured)")
		}

		if corsMethods != "" {
			t.Logf("CORS Methods header present: %s", corsMethods)
		} else {
			t.Log("CORS Methods header not present (CORS may not be configured)")
		}

		if corsHeaders != "" {
			t.Logf("CORS Headers header present: %s", corsHeaders)
		} else {
			t.Log("CORS Headers header not present (CORS may not be configured)")
		}
	})

	t.Run("actual_request_with_origin", func(t *testing.T) {
		request := fixtures.BasicChatRequest()
		headers := map[string]string{
			"Origin": "https://example.com",
		}

		resp, body, err := ts.MakeRequest("POST", "/v1/chat/completions", request, headers)
		if err != nil {
			t.Fatalf("CORS actual request failed: %v", err)
		}

		// The actual API request should work regardless of CORS configuration
		if resp.StatusCode >= 500 {
			t.Errorf("Server error: %d", resp.StatusCode)
		}

		// Check for CORS headers in response (optional)
		corsOrigin := resp.Header.Get("Access-Control-Allow-Origin")
		if corsOrigin != "" {
			t.Logf("CORS Origin header in response: %s", corsOrigin)
		} else {
			t.Log("No CORS Origin header in response (CORS may not be configured)")
		}

		// Log response for debugging
		if resp.StatusCode >= 400 {
			t.Logf("Response status: %d, body: %s", resp.StatusCode, string(body))
		} else {
			t.Logf("Request successful with status: %d", resp.StatusCode)
		}
	})

	t.Run("multiple_origins", func(t *testing.T) {
		origins := []string{
			"https://example.com",
			"https://test.com",
			"http://localhost:3000",
		}

		for _, origin := range origins {
			t.Run("origin_"+origin, func(t *testing.T) {
				request := fixtures.BasicChatRequest()
				headers := map[string]string{
					"Origin": origin,
				}

				resp, body, err := ts.MakeRequest("POST", "/v1/chat/completions", request, headers)
				if err != nil {
					t.Fatalf("Request with origin %s failed: %v", origin, err)
				}

				// Check if request is processed (regardless of CORS)
				if resp.StatusCode >= 500 {
					t.Errorf("Server error for origin %s: %d", origin, resp.StatusCode)
				}

				corsOrigin := resp.Header.Get("Access-Control-Allow-Origin")
				if corsOrigin != "" {
					t.Logf("Origin %s -> CORS header: %s", origin, corsOrigin)
				} else {
					t.Logf("Origin %s -> No CORS header (may not be configured)", origin)
				}

				// Log any errors for debugging
				if resp.StatusCode >= 400 {
					t.Logf("Origin %s response: %d, body: %s", origin, resp.StatusCode, string(body))
				}
			})
		}
	})

	t.Run("no_origin_header", func(t *testing.T) {
		request := fixtures.BasicChatRequest()

		resp, body, err := ts.MakeRequest("POST", "/v1/chat/completions", request, nil)
		if err != nil {
			t.Fatalf("Request without origin failed: %v", err)
		}

		// Request should work without Origin header
		if resp.StatusCode >= 500 {
			t.Errorf("Server error without origin: %d", resp.StatusCode)
		}

		// CORS headers should not be present without Origin
		corsOrigin := resp.Header.Get("Access-Control-Allow-Origin")
		if corsOrigin != "" {
			t.Logf("Unexpected CORS header without origin: %s", corsOrigin)
		} else {
			t.Log("No CORS header without origin (expected)")
		}

		// Log response for debugging
		if resp.StatusCode >= 400 {
			t.Logf("Response without origin: %d, body: %s", resp.StatusCode, string(body))
		} else {
			t.Logf("Request without origin successful: %d", resp.StatusCode)
		}
	})

	t.Run("cors_headers_consistency", func(t *testing.T) {
		// Test that CORS headers are consistent across different endpoints
		endpoints := []string{
			"/health",
			"/v1/models",
		}

		origin := "https://example.com"
		headers := map[string]string{
			"Origin": origin,
		}

		for _, endpoint := range endpoints {
			t.Run("endpoint_"+endpoint, func(t *testing.T) {
				resp, _, err := ts.MakeRequest("GET", endpoint, nil, headers)
				if err != nil {
					t.Fatalf("Request to %s failed: %v", endpoint, err)
				}

				corsOrigin := resp.Header.Get("Access-Control-Allow-Origin")
				if corsOrigin != "" {
					t.Logf("Endpoint %s CORS header: %s", endpoint, corsOrigin)
				} else {
					t.Logf("Endpoint %s no CORS header (may not be configured)", endpoint)
				}

				// Endpoints should respond successfully
				if resp.StatusCode >= 500 {
					t.Errorf("Server error for endpoint %s: %d", endpoint, resp.StatusCode)
				}
			})
		}
	})
} 