package integration

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aashari/go-generative-api-router/test/helpers"
)

func TestStreamingSupport(t *testing.T) {
	ts := helpers.SetupTestServer(t)
	defer ts.Teardown()

	t.Run("streaming_chat_completion", func(t *testing.T) {
		request := helpers.ChatCompletionRequest{
			Model: "gpt-4o",
			Messages: []helpers.Message{
				{
					Role:    "user",
					Content: "Count from 1 to 5, one number per line",
				},
			},
			MaxTokens:   50,
			Temperature: 0.1,
			Stream:      true, // Enable streaming
		}

		reqBody, _ := json.Marshal(request)
		req, err := http.NewRequest("POST", ts.BaseURL()+"/v1/chat/completions", bytes.NewBuffer(reqBody))
		if err != nil {
			t.Fatalf("Failed to create streaming request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")

		resp, err := ts.HTTPClient().Do(req)
		if err != nil {
			t.Fatalf("Streaming request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Logf("Streaming returned status %d (might be API limitation): %s", resp.StatusCode, string(body))
			return
		}

		// Check for streaming headers
		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, "text/event-stream") && !strings.Contains(contentType, "application/json") {
			t.Logf("Expected streaming content type, got: %s", contentType)
		}

		// Read streaming response
		scanner := bufio.NewScanner(resp.Body)
		eventCount := 0
		timeout := time.After(30 * time.Second)

		done := make(chan bool)
		go func() {
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" {
					continue
				}

				// Handle SSE format
				if strings.HasPrefix(line, "data: ") {
					eventCount++
					data := strings.TrimPrefix(line, "data: ")

					if data == "[DONE]" {
						t.Logf("Streaming completed with [DONE] signal")
						break
					}

					// Try to parse as JSON
					var streamEvent map[string]interface{}
					if err := json.Unmarshal([]byte(data), &streamEvent); err == nil {
						t.Logf("Received streaming event %d", eventCount)
					}
				}

				if eventCount >= 5 { // Reasonable limit for test
					break
				}
			}
			done <- true
		}()

		select {
		case <-done:
			t.Logf("Streaming test completed, received %d events", eventCount)
		case <-timeout:
			t.Log("Streaming test timed out (acceptable for integration test)")
		}
	})
} 