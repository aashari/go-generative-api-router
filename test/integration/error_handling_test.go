package integration

import (
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
		request := fixtures.InvalidRequest()

		resp, body, err := ts.MakeRequest("POST", "/v1/chat/completions", request, nil)
		if err != nil {
			t.Fatalf("Invalid request failed: %v", err)
		}

		ts.AssertStatusCode(resp, 400)

		var errorResp helpers.ErrorResponse
		ts.AssertJSONResponse(body, &errorResp)

		if errorResp.Error.Type == "" {
			t.Error("Missing error type")
		}
		if errorResp.Error.Message == "" {
			t.Error("Missing error message")
		}

		t.Logf("Error response: %s - %s", errorResp.Error.Type, errorResp.Error.Message)
	})

	t.Run("invalid_http_method", func(t *testing.T) {
		resp, body, err := ts.MakeRequest("GET", "/v1/chat/completions", nil, nil)
		if err != nil {
			t.Fatalf("Invalid method test failed: %v", err)
		}

		ts.AssertStatusCode(resp, 405)

		var errorResp helpers.ErrorResponse
		ts.AssertJSONResponse(body, &errorResp)

		if errorResp.Error.Type == "" {
			t.Error("Missing error type for invalid method")
		}

		t.Logf("Method not allowed error: %s", errorResp.Error.Message)
	})

	t.Run("invalid_json_payload", func(t *testing.T) {
		// Send invalid JSON
		invalidJSON := `{"model": "gpt-4o", "messages": [`

		resp, body, err := ts.MakeRequest("POST", "/v1/chat/completions", invalidJSON, nil)
		if err != nil {
			t.Fatalf("Invalid JSON test failed: %v", err)
		}

		ts.AssertStatusCode(resp, 400)

		var errorResp helpers.ErrorResponse
		ts.AssertJSONResponse(body, &errorResp)

		if errorResp.Error.Type == "" {
			t.Error("Missing error type for invalid JSON")
		}

		t.Logf("Invalid JSON error: %s", errorResp.Error.Message)
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
			ts.AssertJSONResponse(body, &errorResp)

			t.Logf("Large request rejected (expected): %s", errorResp.Error.Message)
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
			ts.AssertJSONResponse(body, &errorResp)

			t.Logf("Missing content type error: %s", errorResp.Error.Message)
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
		ts.AssertJSONResponse(body, &errorResp)

		if errorResp.Error.Type == "" {
			t.Error("Missing error type for invalid endpoint")
		}

		t.Logf("Invalid endpoint error: %s", errorResp.Error.Message)
	})

	t.Run("empty_request_body", func(t *testing.T) {
		resp, body, err := ts.MakeRequest("POST", "/v1/chat/completions", nil, nil)
		if err != nil {
			t.Fatalf("Empty body test failed: %v", err)
		}

		ts.AssertStatusCode(resp, 400)

		var errorResp helpers.ErrorResponse
		ts.AssertJSONResponse(body, &errorResp)

		if errorResp.Error.Type == "" {
			t.Error("Missing error type for empty body")
		}

		t.Logf("Empty body error: %s", errorResp.Error.Message)
	})

	t.Run("error_response_format", func(t *testing.T) {
		request := fixtures.InvalidRequest()

		resp, body, err := ts.MakeRequest("POST", "/v1/chat/completions", request, nil)
		if err != nil {
			t.Fatalf("Error format test failed: %v", err)
		}

		ts.AssertStatusCode(resp, 400)

		var errorResp helpers.ErrorResponse
		ts.AssertJSONResponse(body, &errorResp)

		// Verify error response structure
		if errorResp.Error.Type == "" {
			t.Error("Missing error.type field")
		}
		if errorResp.Error.Message == "" {
			t.Error("Missing error.message field")
		}

		// Verify content type
		contentType := resp.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type: application/json, got: %s", contentType)
		}

		t.Logf("Error response format verified: %+v", errorResp.Error)
	})
}
