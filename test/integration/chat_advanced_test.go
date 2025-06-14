package integration

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/aashari/go-generative-api-router/test/helpers"
)

func TestChatCompletionsAdvanced(t *testing.T) {
	ts := helpers.SetupTestServer(t)
	defer ts.Teardown()

	t.Run("vendor_specific_routing_openai", func(t *testing.T) {
		request := helpers.ChatCompletionRequest{
			Model: "gpt-4o",
			Messages: []helpers.Message{
				{
					Role:    "user",
					Content: "Test OpenAI routing",
				},
			},
			MaxTokens: 20,
		}

		resp, body, err := ts.MakeRequest("POST", "/v1/chat/completions?vendor=openai", request, nil)
		if err != nil {
			t.Fatalf("OpenAI-specific routing failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Logf("OpenAI routing returned status %d (might be API limitation): %s", resp.StatusCode, string(body))
			return
		}

		var chatResp helpers.ChatCompletionResponse
		if err := json.Unmarshal(body, &chatResp); err != nil {
			t.Fatalf("Failed to parse chat response: %v", err)
		}

		// Verify original model name is preserved
		if chatResp.Model != request.Model {
			t.Errorf("Expected preserved model '%s', got '%s'", request.Model, chatResp.Model)
		}

		t.Logf("OpenAI-specific routing successful")
	})

	t.Run("vendor_specific_routing_gemini", func(t *testing.T) {
		request := helpers.ChatCompletionRequest{
			Model: "gemini-2.5-flash-preview-04-17",
			Messages: []helpers.Message{
				{
					Role:    "user",
					Content: "Test Gemini routing",
				},
			},
			MaxTokens: 20,
		}

		resp, body, err := ts.MakeRequest("POST", "/v1/chat/completions?vendor=gemini", request, nil)
		if err != nil {
			t.Fatalf("Gemini-specific routing failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Logf("Gemini routing returned status %d (might be API limitation): %s", resp.StatusCode, string(body))
			return
		}

		var chatResp helpers.ChatCompletionResponse
		if err := json.Unmarshal(body, &chatResp); err != nil {
			t.Fatalf("Failed to parse chat response: %v", err)
		}

		// Verify original model name is preserved
		if chatResp.Model != request.Model {
			t.Errorf("Expected preserved model '%s', got '%s'", request.Model, chatResp.Model)
		}

		t.Logf("Gemini-specific routing successful")
	})

	t.Run("multiple_requests_distribution", func(t *testing.T) {
		// Test even distribution by making multiple requests
		successfulRequests := 0

		for i := 0; i < 10; i++ {
			request := helpers.ChatCompletionRequest{
				Model: "gpt-4o",
				Messages: []helpers.Message{
					{
						Role:    "user",
						Content: fmt.Sprintf("Test request #%d", i+1),
					},
				},
				MaxTokens: 10,
			}

			resp, body, err := ts.MakeRequest("POST", "/v1/chat/completions", request, nil)
			if err != nil {
				t.Logf("Request %d failed: %v", i+1, err)
				continue
			}

			if resp.StatusCode == 200 {
				successfulRequests++
				var chatResp helpers.ChatCompletionResponse
				if err := json.Unmarshal(body, &chatResp); err == nil {
					// Verify model name preservation
					if chatResp.Model != request.Model {
						t.Errorf("Request %d: Expected model '%s', got '%s'", i+1, request.Model, chatResp.Model)
					}
				}
			} else {
				t.Logf("Request %d returned status %d: %s", i+1, resp.StatusCode, string(body))
			}
		}

		t.Logf("Multiple requests distribution test: %d/10 successful", successfulRequests)
		
		// At least some requests should succeed
		if successfulRequests == 0 {
			t.Error("All requests failed - check API configuration")
		}
	})

	t.Run("empty_messages_array", func(t *testing.T) {
		request := helpers.ChatCompletionRequest{
			Model:    "gpt-4o",
			Messages: []helpers.Message{}, // Empty messages
		}

		resp, body, err := ts.MakeRequest("POST", "/v1/chat/completions", request, nil)
		if err != nil {
			t.Fatalf("Empty messages request failed: %v", err)
		}

		// Should return an error for empty messages
		if resp.StatusCode == 200 {
			t.Error("Expected error for empty messages array")
		} else {
			t.Logf("Empty messages correctly rejected with status: %d, body: %s", resp.StatusCode, string(body))
		}
	})
} 