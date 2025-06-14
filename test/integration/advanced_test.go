package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aashari/go-generative-api-router/test/helpers"
)

func TestImageSupportDetection(t *testing.T) {
	ts := helpers.SetupTestServer(t)
	defer ts.Teardown()

	t.Run("image_in_message_content", func(t *testing.T) {
		// Test with image content (base64 data URL simulation)
		imageMessage := helpers.Message{
			Role:    "user",
			Content: "What do you see in this image? data:image/jpeg;base64,/9j/4AAQSkZJRgABAQAAAQABAAD/2wBDAAYEBQYFBAYGBQYHBwYIChAKCgkJChQODwwQFxQYGBcUFhYaHSUfGhsjHBYWICwgIyYnKSopGR8tMC0oMCUoKSj/2wBDAQcHBwoIChMKChMoGhYaKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCj/wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAv/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFQEBAQAAAAAAAAAAAAAAAAAAAAX/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIRAxEAPwCdABmX/9k=",
		}

		request := helpers.ChatCompletionRequest{
			Model:     "gpt-4o",
			Messages:  []helpers.Message{imageMessage},
			MaxTokens: 50,
		}

		resp, body, err := ts.MakeRequest("POST", "/v1/chat/completions", request, nil)
		if err != nil {
			t.Fatalf("Image content request failed: %v", err)
		}

		// The router should handle image content detection
		if resp.StatusCode != http.StatusOK {
			t.Logf("Image content returned status %d (might be API limitation): %s", resp.StatusCode, string(body))
			return
		}

		t.Log("Image content detection test completed")
	})

	t.Run("vision_capable_model_routing", func(t *testing.T) {
		// Test that vision-capable models are properly routed
		request := helpers.ChatCompletionRequest{
			Model: "gpt-4o", // Should support images according to config
			Messages: []helpers.Message{
				{
					Role:    "user",
					Content: "Describe an image with a cat",
				},
			},
			MaxTokens: 30,
		}

		resp, body, err := ts.MakeRequest("POST", "/v1/chat/completions", request, nil)
		if err != nil {
			t.Fatalf("Vision model request failed: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Logf("Vision model returned status %d (might be API limitation): %s", resp.StatusCode, string(body))
			return
		}

		var chatResp helpers.ChatCompletionResponse
		if err := json.Unmarshal(body, &chatResp); err != nil {
			t.Fatalf("Failed to parse vision response: %v", err)
		}

		// Verify model name preservation
		if chatResp.Model != request.Model {
			t.Errorf("Expected model '%s', got '%s'", request.Model, chatResp.Model)
		}

		t.Log("Vision-capable model routing test completed")
	})
}

func TestVendorSpecificFeatures(t *testing.T) {
	ts := helpers.SetupTestServer(t)
	defer ts.Teardown()

	t.Run("openai_specific_parameters", func(t *testing.T) {
		request := map[string]interface{}{
			"model": "gpt-4o",
			"messages": []map[string]string{
				{
					"role":    "user",
					"content": "Test OpenAI-specific features",
				},
			},
			"max_tokens":        20,
			"temperature":       0.7,
			"top_p":             1.0,
			"frequency_penalty": 0.0,
			"presence_penalty":  0.0,
			"logit_bias":        map[string]float64{},
		}

		resp, body, err := ts.MakeRequest("POST", "/v1/chat/completions?vendor=openai", request, nil)
		if err != nil {
			t.Fatalf("OpenAI-specific request failed: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Logf("OpenAI-specific returned status %d (might be API limitation): %s", resp.StatusCode, string(body))
			return
		}

		t.Log("OpenAI-specific parameters test completed")
	})

	t.Run("gemini_specific_routing", func(t *testing.T) {
		request := helpers.ChatCompletionRequest{
			Model: "gemini-2.5-flash-preview-04-17",
			Messages: []helpers.Message{
				{
					Role:    "user",
					Content: "Test Gemini-specific routing",
				},
			},
			MaxTokens: 20,
		}

		resp, body, err := ts.MakeRequest("POST", "/v1/chat/completions?vendor=gemini", request, nil)
		if err != nil {
			t.Fatalf("Gemini-specific request failed: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Logf("Gemini-specific returned status %d (might be API limitation): %s", resp.StatusCode, string(body))
			return
		}

		var chatResp helpers.ChatCompletionResponse
		if err := json.Unmarshal(body, &chatResp); err != nil {
			t.Fatalf("Failed to parse Gemini response: %v", err)
		}

		// Verify original model name is preserved
		if chatResp.Model != request.Model {
			t.Errorf("Expected preserved model '%s', got '%s'", request.Model, chatResp.Model)
		}

		t.Log("Gemini-specific routing test completed")
	})
}

func TestTimeoutHandling(t *testing.T) {
	ts := helpers.SetupTestServer(t)
	defer ts.Teardown()

	t.Run("request_timeout_behavior", func(t *testing.T) {
		// Create a request that might take a while
		request := helpers.ChatCompletionRequest{
			Model: "gpt-4o",
			Messages: []helpers.Message{
				{
					Role:    "user",
					Content: "Write a detailed explanation of quantum physics in exactly 1000 words",
				},
			},
			MaxTokens: 1500, // Large token count
		}

		// Set a shorter timeout for this specific test
		client := &http.Client{
			Timeout: 10 * time.Second,
		}

		reqBody, _ := json.Marshal(request)
		req, err := http.NewRequest("POST", ts.BaseURL()+"/v1/chat/completions", bytes.NewBuffer(reqBody))
		if err != nil {
			t.Fatalf("Failed to create timeout test request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			// Timeout is expected behavior
			t.Logf("Request timed out as expected: %v", err)
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		t.Logf("Timeout test completed with status %d", resp.StatusCode)
		if resp.StatusCode != http.StatusOK {
			t.Logf("Response: %s", string(body))
		}
	})
}

func TestAdvancedErrorScenarios(t *testing.T) {
	ts := helpers.SetupTestServer(t)
	defer ts.Teardown()

	t.Run("extremely_large_request", func(t *testing.T) {
		// Test with very large content
		largeContent := strings.Repeat("This is a very long message that tests the router's ability to handle large payloads. ", 1000)

		request := helpers.ChatCompletionRequest{
			Model: "gpt-4o",
			Messages: []helpers.Message{
				{
					Role:    "user",
					Content: largeContent,
				},
			},
			MaxTokens: 10,
		}

		resp, body, err := ts.MakeRequest("POST", "/v1/chat/completions", request, nil)
		if err != nil {
			t.Fatalf("Large request failed: %v", err)
		}

		// Large requests might be rejected by the APIs, which is acceptable
		if resp.StatusCode != http.StatusOK {
			t.Logf("Large request returned status %d (acceptable): %s", resp.StatusCode, string(body))
			return
		}

		t.Log("Large request handling test completed")
	})

	t.Run("malformed_model_name", func(t *testing.T) {
		request := helpers.ChatCompletionRequest{
			Model: "non-existent-model-12345",
			Messages: []helpers.Message{
				{
					Role:    "user",
					Content: "Test with invalid model",
				},
			},
		}

		resp, body, err := ts.MakeRequest("POST", "/v1/chat/completions", request, nil)
		if err != nil {
			t.Fatalf("Invalid model request failed: %v", err)
		}

		// The router should still try to route this, vendor APIs will reject
		// This tests the router's fallback behavior
		t.Logf("Invalid model request returned status %d: %s", resp.StatusCode, string(body))
	})

	t.Run("mixed_content_types", func(t *testing.T) {
		// Test with different content types mixed together
		messages := []helpers.Message{
			{
				Role:    "user",
				Content: "First message",
			},
			{
				Role:    "assistant",
				Content: "Response message",
			},
			{
				Role:    "user",
				Content: "Follow-up question",
			},
		}

		request := helpers.ChatCompletionRequest{
			Model:     "gpt-4o",
			Messages:  messages,
			MaxTokens: 50,
		}

		resp, body, err := ts.MakeRequest("POST", "/v1/chat/completions", request, nil)
		if err != nil {
			t.Fatalf("Mixed content request failed: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Logf("Mixed content returned status %d (might be API limitation): %s", resp.StatusCode, string(body))
			return
		}

		t.Log("Mixed content types test completed")
	})
}

func TestResponseValidation(t *testing.T) {
	ts := helpers.SetupTestServer(t)
	defer ts.Teardown()

	t.Run("response_format_consistency", func(t *testing.T) {
		request := helpers.ChatCompletionRequest{
			Model: "gpt-4o",
			Messages: []helpers.Message{
				{
					Role:    "user",
					Content: "Reply with a single word",
				},
			},
			MaxTokens: 10,
		}

		resp, body, err := ts.MakeRequest("POST", "/v1/chat/completions", request, nil)
		if err != nil {
			t.Fatalf("Response format test failed: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Logf("Response format test returned status %d: %s", resp.StatusCode, string(body))
			return
		}

		var chatResp helpers.ChatCompletionResponse
		if err := json.Unmarshal(body, &chatResp); err != nil {
			t.Fatalf("Failed to parse response format: %v", err)
		}

		// Validate OpenAI-compatible format
		if chatResp.Object != "chat.completion" {
			t.Errorf("Expected object 'chat.completion', got '%s'", chatResp.Object)
		}

		if len(chatResp.Choices) == 0 {
			t.Error("Expected at least one choice")
		}

		for i, choice := range chatResp.Choices {
			if choice.Index != i {
				t.Errorf("Expected choice index %d, got %d", i, choice.Index)
			}
			if choice.Message.Role != "assistant" {
				t.Errorf("Expected assistant role, got '%s'", choice.Message.Role)
			}
		}

		// Validate usage information
		if chatResp.Usage.TotalTokens <= 0 {
			t.Error("Expected positive total tokens")
		}

		t.Log("Response format consistency test completed")
	})
} 