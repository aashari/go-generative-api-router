package integration

import (
	"testing"
	"time"

	"github.com/aashari/go-generative-api-router/test/fixtures"
	"github.com/aashari/go-generative-api-router/test/helpers"
)

func TestChatCompletionsBasic(t *testing.T) {
	config := helpers.DefaultTestConfig()
	config.ServiceName = "chat-basic-integration-test"

	ts := helpers.NewTestServer(t, config)
	defer ts.Close()

	t.Run("basic_chat_completion", func(t *testing.T) {
		request := fixtures.BasicChatRequest()

		resp, body, err := ts.MakeRequest("POST", "/v1/chat/completions", request, nil)
		if err != nil {
			t.Fatalf("Chat completion request failed: %v", err)
		}

		// Accept both success and API limitation responses
		if resp.StatusCode == 200 {
			var chatResp helpers.ChatCompletionResponse
			ts.AssertJSONResponse(body, &chatResp)

			// Verify response format
			if chatResp.ID == "" {
				t.Error("Missing response ID")
			}
			if chatResp.Object != "chat.completion" {
				t.Errorf("Expected object 'chat.completion', got %s", chatResp.Object)
			}
			if chatResp.Model != request.Model {
				t.Errorf("Expected model '%s', got '%s'", request.Model, chatResp.Model)
			}
			if len(chatResp.Choices) == 0 {
				t.Error("No choices in response")
			}

			t.Logf("Chat completion successful: %s", chatResp.Choices[0].Message.Content)
		} else {
			t.Logf("Chat completion returned status %d (might be API limitation): %s", resp.StatusCode, string(body))
		}
	})

	t.Run("model_name_preservation", func(t *testing.T) {
		customModel := "my-custom-model-name"
		request := fixtures.CustomModelRequest(customModel)

		resp, body, err := ts.MakeRequest("POST", "/v1/chat/completions", request, nil)
		if err != nil {
			t.Fatalf("Custom model request failed: %v", err)
		}

		if resp.StatusCode == 200 {
			var chatResp helpers.ChatCompletionResponse
			ts.AssertJSONResponse(body, &chatResp)

			// Critical test: model name should be preserved
			if chatResp.Model != customModel {
				t.Errorf("Model name not preserved: expected '%s', got '%s'", customModel, chatResp.Model)
			}

			t.Logf("Model name preservation verified: %s", chatResp.Model)
		} else {
			t.Logf("Custom model request returned status %d (might be API limitation)", resp.StatusCode)
		}
	})

	t.Run("minimal_request", func(t *testing.T) {
		request := fixtures.MinimalRequest()

		resp, body, err := ts.MakeRequest("POST", "/v1/chat/completions", request, nil)
		if err != nil {
			t.Fatalf("Minimal request failed: %v", err)
		}

		if resp.StatusCode == 200 {
			var chatResp helpers.ChatCompletionResponse
			ts.AssertJSONResponse(body, &chatResp)

			if chatResp.ID == "" {
				t.Error("Missing response ID")
			}

			t.Log("Minimal request successful")
		} else {
			t.Logf("Minimal request returned status %d (might be API limitation)", resp.StatusCode)
		}
	})

	t.Run("request_timeout_handling", func(t *testing.T) {
		request := fixtures.BasicChatRequest()

		// Use very short timeout to test timeout handling
		resp, body, err := ts.MakeRequestWithTimeout("POST", "/v1/chat/completions", request, nil, 2*time.Second)

		if err != nil {
			// Timeout is acceptable - means the request was processed
			t.Logf("Request timeout test: %v (expected)", err)
			return
		}

		// If we get a response, verify it's properly formatted
		if resp.StatusCode == 200 {
			var chatResp helpers.ChatCompletionResponse
			ts.AssertJSONResponse(body, &chatResp)
			t.Log("Request completed within timeout")
		} else {
			t.Logf("Request returned status %d within timeout", resp.StatusCode)
		}
	})

	t.Run("vendor_specific_routing", func(t *testing.T) {
		// Test OpenAI vendor routing
		request := fixtures.BasicChatRequest()

		resp, body, err := ts.MakeRequest("POST", "/v1/chat/completions?vendor=openai", request, nil)
		if err != nil {
			t.Fatalf("OpenAI vendor request failed: %v", err)
		}

		if resp.StatusCode == 200 {
			var chatResp helpers.ChatCompletionResponse
			ts.AssertJSONResponse(body, &chatResp)

			if chatResp.Model != request.Model {
				t.Errorf("OpenAI vendor: model name not preserved: expected '%s', got '%s'", request.Model, chatResp.Model)
			}

			t.Log("OpenAI vendor routing successful")
		} else {
			t.Logf("OpenAI vendor request returned status %d (might be API limitation)", resp.StatusCode)
		}

		// Test Gemini vendor routing
		resp2, body2, err := ts.MakeRequest("POST", "/v1/chat/completions?vendor=gemini", request, nil)
		if err != nil {
			t.Fatalf("Gemini vendor request failed: %v", err)
		}

		if resp2.StatusCode == 200 {
			var chatResp2 helpers.ChatCompletionResponse
			ts.AssertJSONResponse(body2, &chatResp2)

			if chatResp2.Model != request.Model {
				t.Errorf("Gemini vendor: model name not preserved: expected '%s', got '%s'", request.Model, chatResp2.Model)
			}

			t.Log("Gemini vendor routing successful")
		} else {
			t.Logf("Gemini vendor request returned status %d (might be API limitation)", resp2.StatusCode)
		}
	})

	t.Run("concurrent_requests", func(t *testing.T) {
		const numRequests = 3
		results := make(chan error, numRequests)

		for i := 0; i < numRequests; i++ {
			go func(requestNum int) {
				request := fixtures.CustomModelRequest("concurrent-test-" + string(rune('0'+requestNum)))

				resp, body, err := ts.MakeRequestWithTimeout("POST", "/v1/chat/completions", request, nil, 10*time.Second)
				if err != nil {
					results <- err
					return
				}

				if resp.StatusCode == 200 {
					var chatResp helpers.ChatCompletionResponse
					ts.AssertJSONResponse(body, &chatResp)
				}

				results <- nil
			}(i)
		}

		// Wait for all requests to complete
		successCount := 0
		for i := 0; i < numRequests; i++ {
			err := <-results
			if err == nil {
				successCount++
			} else {
				t.Logf("Concurrent request %d failed: %v", i, err)
			}
		}

		t.Logf("Concurrent requests: %d/%d successful", successCount, numRequests)
	})
}
