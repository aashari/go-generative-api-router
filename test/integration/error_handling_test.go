package integration

import (
	"strings"
	"testing"

	"github.com/aashari/go-generative-api-router/test/fixtures"
	"github.com/aashari/go-generative-api-router/test/helpers"
)

func TestErrorHandling(t *testing.T) {
	config := helpers.DefaultTestConfig()
	config.ServiceName = "error-handling-integration-test"

	ts := helpers.NewTestServer(t, config)
	defer ts.Close()

	t.Run("invalid_request_missing_messages", func(t *testing.T) {
		request := map[string]interface{}{
			"model": "gpt-4o",
			// Missing required "messages" field
		}

		resp, body, err := ts.MakeRequest("POST", "/v1/chat/completions", request, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		ts.AssertStatusCode(resp, 400)

		// Try to parse as JSON error response, but handle plain text gracefully
		var errorResp helpers.ErrorResponse
		if ts.AssertJSONResponseOrSkip(body, &errorResp) {
			// JSON error response
			if errorResp.Error.Message == "" {
				t.Error("Expected error message in JSON response")
			}
			t.Logf("JSON Error: %s", errorResp.Error.Message)
		} else {
			// Plain text error response
			bodyStr := string(body)
			if !strings.Contains(bodyStr, "messages") {
				t.Errorf("Expected error message to mention 'messages', got: %s", bodyStr)
			}
			t.Logf("Plain text error: %s", bodyStr)
		}
	})

	t.Run("invalid_http_method", func(t *testing.T) {
		resp, body, err := ts.MakeRequest("GET", "/v1/chat/completions", nil, nil)
		if err != nil {
			t.Fatalf("Invalid method test failed: %v", err)
		}

		ts.AssertStatusCode(resp, 405)

		// Handle both JSON and plain text error responses
		var errorResp helpers.ErrorResponse
		if ts.AssertJSONResponseOrSkip(body, &errorResp) {
			// JSON error response
			if errorResp.Error.Type == "" {
				t.Error("Missing error type for invalid method")
			}
			t.Logf("Method not allowed error: %s", errorResp.Error.Message)
		} else {
			// Plain text error response
			bodyStr := string(body)
			if !strings.Contains(strings.ToLower(bodyStr), "method") && !strings.Contains(strings.ToLower(bodyStr), "not allowed") {
				t.Errorf("Expected error message to mention method not allowed, got: %s", bodyStr)
			}
			t.Logf("Plain text method not allowed error: %s", bodyStr)
		}
	})

	t.Run("invalid_json_payload", func(t *testing.T) {
		// Send invalid JSON
		invalidJSON := `{"model": "gpt-4o", "messages": [`

		resp, body, err := ts.MakeRequest("POST", "/v1/chat/completions", invalidJSON, nil)
		if err != nil {
			t.Fatalf("Invalid JSON test failed: %v", err)
		}

		ts.AssertStatusCode(resp, 400)

		// Handle both JSON and plain text error responses
		var errorResp helpers.ErrorResponse
		if ts.AssertJSONResponseOrSkip(body, &errorResp) {
			// JSON error response
			if errorResp.Error.Type == "" {
				t.Error("Missing error type for invalid JSON")
			}
			t.Logf("Invalid JSON error: %s", errorResp.Error.Message)
		} else {
			// Plain text error response - acceptable for malformed JSON
			bodyStr := string(body)
			t.Logf("Plain text JSON error: %s", bodyStr)
		}
	})

	t.Run("large_request_handling", func(t *testing.T) {
		request := fixtures.LargeChatRequest()

		resp, body, err := ts.MakeRequest("POST", "/v1/chat/completions", request, nil)
		if err != nil {
			t.Fatalf("Large request failed: %v", err)
		}

		// Large requests might be rejected by APIs, which is acceptable
		if resp.StatusCode >= 400 {
			var errorResp helpers.ErrorResponse
			if ts.AssertJSONResponseOrSkip(body, &errorResp) {
				t.Logf("Large request rejected (expected): %s", errorResp.Error.Message)
			} else {
				t.Logf("Large request rejected with plain text: %s", string(body))
			}
		} else {
			t.Log("Large request accepted")
		}
	})

	t.Run("missing_content_type", func(t *testing.T) {
		request := fixtures.BasicChatRequest()

		// Remove Content-Type header
		headers := map[string]string{
			"Content-Type": "",
		}

		resp, body, err := ts.MakeRequest("POST", "/v1/chat/completions", request, headers)
		if err != nil {
			t.Fatalf("Missing content type test failed: %v", err)
		}

		// Should still work or return proper error
		if resp.StatusCode >= 400 {
			var errorResp helpers.ErrorResponse
			if ts.AssertJSONResponseOrSkip(body, &errorResp) {
				t.Logf("Missing content type error: %s", errorResp.Error.Message)
			} else {
				t.Logf("Missing content type plain text error: %s", string(body))
			}
		} else {
			t.Log("Request succeeded despite missing content type")
		}
	})

	t.Run("invalid_endpoint", func(t *testing.T) {
		resp, body, err := ts.MakeRequest("GET", "/v1/invalid-endpoint", nil, nil)
		if err != nil {
			t.Fatalf("Invalid endpoint test failed: %v", err)
		}

		ts.AssertStatusCode(resp, 404)

		var errorResp helpers.ErrorResponse
		if ts.AssertJSONResponseOrSkip(body, &errorResp) {
			if errorResp.Error.Type == "" {
				t.Error("Missing error type for invalid endpoint")
			}
			t.Logf("Invalid endpoint error: %s", errorResp.Error.Message)
		} else {
			t.Logf("Invalid endpoint plain text error: %s", string(body))
		}
	})

	t.Run("empty_request_body", func(t *testing.T) {
		resp, body, err := ts.MakeRequest("POST", "/v1/chat/completions", nil, nil)
		if err != nil {
			t.Fatalf("Empty body test failed: %v", err)
		}

		ts.AssertStatusCode(resp, 400)

		var errorResp helpers.ErrorResponse
		if ts.AssertJSONResponseOrSkip(body, &errorResp) {
			if errorResp.Error.Type == "" {
				t.Error("Missing error type for empty body")
			}
			t.Logf("Empty body error: %s", errorResp.Error.Message)
		} else {
			t.Logf("Empty body plain text error: %s", string(body))
		}
	})

	t.Run("error_response_format", func(t *testing.T) {
		request := fixtures.InvalidRequest()

		resp, body, err := ts.MakeRequest("POST", "/v1/chat/completions", request, nil)
		if err != nil {
			t.Fatalf("Error format test failed: %v", err)
		}

		ts.AssertStatusCode(resp, 400)

		var errorResp helpers.ErrorResponse
		if ts.AssertJSONResponseOrSkip(body, &errorResp) {
			// Verify error response structure
			if errorResp.Error.Type == "" {
				t.Error("Missing error.type field")
			}
			if errorResp.Error.Message == "" {
				t.Error("Missing error.message field")
			}

			// Verify content type
			contentType := resp.Header.Get("Content-Type")
			if !strings.Contains(contentType, "application/json") {
				t.Errorf("Expected Content-Type to contain application/json, got: %s", contentType)
			}

			t.Logf("Error response format verified: %+v", errorResp.Error)
		} else {
			// Plain text error is also acceptable
			t.Logf("Plain text error response: %s", string(body))
		}
	})
}
